"""Runtime probes: start the built binary and talk to it over HTTP.

Why: code that compiles and passes its own (LLM-written) tests can still serve the wrong
thing. Probes check what the spec promised: /health 200, every declared metric on
/metrics, every endpoint answering, and the spec's `probes` (status expectations).
Deliberately does not: judge response bodies. 404 is tolerated on endpoints with path
params (an unknown id legitimately 404s), so those are only checked for 405/5xx.
"""
import socket
import time
import urllib.error
import urllib.request
from pathlib import Path

from specgen.verify.common import excerpt, failed, kill_group, passed

import subprocess
import os


class Service:
    """Lazily started service process, shared by the probe and docs stages."""

    def __init__(self, project, budget_s):
        self.project, self.budget, self.proc = Path(project), budget_s, None
        self.port, self.deadline = None, None
        self.started_ok, self.metrics_text, self.start_error = None, "", ""

    def log(self):
        f = self.project / "svc.log"
        return f.read_text() if f.exists() else ""

    def start(self):
        if self.started_ok is not None:
            return self.started_ok
        self.deadline = time.monotonic() + self.budget
        with socket.socket() as s:
            s.bind(("127.0.0.1", 0))
            self.port = s.getsockname()[1]
        binary = self.project / "svc.bin"
        if not binary.exists():
            self.started_ok, self.start_error = False, "binary svc.bin not built"
            return False
        logf = open(self.project / "svc.log", "w")
        self.proc = subprocess.Popen([str(binary)], cwd=self.project, stdout=logf, stderr=subprocess.STDOUT,
                                     env={**os.environ, "PORT": str(self.port)}, start_new_session=True)
        while time.monotonic() < self.deadline:
            if self.proc.poll() is not None:
                break
            try:
                if self.request("GET", "/health", timeout=1)[0] == 200:
                    self.started_ok = True
                    return True
            except OSError:
                time.sleep(0.1)
        self.started_ok = False
        self.start_error = f"service did not answer /health in time (exit={self.proc.poll()})\n" + excerpt(self.log())
        return False

    def request(self, method, path, body=None, timeout=5):
        data = body.encode() if isinstance(body, str) else None
        req = urllib.request.Request(f"http://127.0.0.1:{self.port}{path}", data=data, method=method)
        if data is not None:
            req.add_header("Content-Type", "application/json")
        try:
            with urllib.request.urlopen(req, timeout=timeout) as r:
                return r.status, r.read().decode(errors="replace")
        except urllib.error.HTTPError as e:
            return e.code, e.read().decode(errors="replace")

    def stop(self):
        if self.proc:
            kill_group(self.proc)
            try:
                self.proc.wait(timeout=3)
            except subprocess.TimeoutExpired:
                pass


def endpoint_problem(status, has_param):
    """Why a status counts as 'endpoint not really there', or None if fine."""
    if status >= 500 or status == 405 or (status == 404 and not has_param):
        return f"status {status}"
    return None


def body_for(method):
    return "{}" if method in ("POST", "PUT", "PATCH") else None


def check_probes(service: Service, spec):
    t0 = time.monotonic()
    if not service.start():
        return failed("probe", time.monotonic() - t0, "probe_health", service.start_error)
    problems = []  # (category, message)

    def req(method, path, body=None):
        left = service.deadline - time.monotonic()
        if left <= 0:
            raise TimeoutError
        return service.request(method, path, body, timeout=min(5, left))

    try:
        st, _ = req("GET", "/health")
        if st != 200:
            problems.append(("probe_health", f"GET /health -> {st}"))
        st, text = req("GET", "/metrics")
        service.metrics_text = text
        missing = [m.name for m in spec.metrics if m.name not in text]
        if st != 200:
            problems.append(("probe_metrics", f"GET /metrics -> {st}"))
        elif missing:
            problems.append(("probe_metrics", f"metrics missing from /metrics: {missing}"))
        for p in spec.probes:
            st, text = req(p.method, p.path, p.body)
            if st not in p.expect_status:
                problems.append(("probe_behavior", f"{p.method} {p.path} body={p.body} -> {st}, expected {p.expect_status}: {text[:120]}"))
        for e in spec.endpoints:
            if e.builtin:
                continue
            st, _ = req(e.method, e.concrete_path(), body_for(e.method))
            why = endpoint_problem(st, e.has_param)
            if why:
                problems.append(("probe_endpoint", f"{e.method} {e.path} -> {why}"))
    except (TimeoutError, OSError) as e:  # hung handler or connection dropped mid-probe
        cat = "timeout" if isinstance(e, TimeoutError) or "timed out" in str(e) else "probe_endpoint"
        problems.append((cat, f"request failed: {e}"))
    dur = time.monotonic() - t0
    if problems:
        return failed("probe", dur, problems[0][0], "\n".join(m for _, m in problems), problems=problems)
    return passed("probe", dur)
