"""Docs-vs-reality check on the generated README.md.

Rules: every spec endpoint and metric must be documented; every documented endpoint
must answer on the running service; every documented metric must be on /metrics.
Why: generated docs that lie are worse than none. Mismatches are reported by name.
Deliberately does not: judge prose quality. Endpoints are found by `METHOD /path`
anywhere in the README; metrics are the backticked names inside a heading containing
"metric" (until the next heading).
"""
import re
import time

from specgen.verify.common import failed, passed
from specgen.verify.probes import body_for, endpoint_problem

_EP = re.compile(r"\b(GET|POST|PUT|PATCH|DELETE)\b[\s|`*]+(/[^\s|`*)\],;]*)")
_NAME = re.compile(r"^[a-zA-Z_:][a-zA-Z0-9_:]*$")


def norm_path(p):
    p = p.rstrip(".,;:")
    p = re.sub(r"\{[^}]*\}", "{}", p)
    p = re.sub(r"/:[A-Za-z_]+", "/{}", p)
    return p.rstrip("/") or "/"


def documented_endpoints(text):
    return {(m, norm_path(p)) for m, p in _EP.findall(text)}


def documented_metrics(text):
    names, level, inside = set(), 0, False
    for line in text.split("\n"):
        h = re.match(r"^(#+)\s+(.*)", line)
        if h:
            if inside and len(h.group(1)) <= level:
                inside = False
            if "metric" in h.group(2).lower():
                inside, level = True, len(h.group(1))
            continue
        if inside:
            names |= {t for t in re.findall(r"`([^`]+)`", line) if _NAME.match(t)}
    return names


def check_docs(project, spec, service):
    t0 = time.monotonic()
    readme = project / "README.md"
    if not readme.exists():
        return failed("docs", 0, "docs_mismatch", "README.md missing")
    text = readme.read_text()
    doc_eps, doc_metrics = documented_endpoints(text), documented_metrics(text)
    problems = []
    for e in spec.endpoints:
        if (e.method, norm_path(e.path)) not in doc_eps:
            problems.append(f"endpoint {e.method} {e.path} is in the spec but not documented")
    for m in spec.metrics:
        if m.name not in doc_metrics:
            problems.append(f"metric {m.name} is in the spec but not documented")
    if not service.start():
        problems.append("service could not start to verify documented endpoints")
    else:
        spec_eps = {(e.method, norm_path(e.path)) for e in spec.endpoints}
        for method, path in sorted(doc_eps - spec_eps):
            st, _ = service.request(method, re.sub(r"\{\}", "x", path), body_for(method))
            why = endpoint_problem(st, "{" in path)
            if why:
                problems.append(f"documented endpoint {method} {path} does not exist ({why})")
        for name in sorted(doc_metrics):
            if name not in service.metrics_text and name not in {m.name for m in spec.metrics}:
                problems.append(f"documented metric {name} is not on /metrics")
    dur = time.monotonic() - t0
    return failed("docs", dur, "docs_mismatch", "\n".join(problems)) if problems else passed("docs", dur)
