"""Shared verifier types and the one place that runs subprocesses safely.

Why: everything the verifier executes is LLM-written code, so every command gets a
timeout and runs in its own process group that is killed afterwards.
Deliberately does not: sandbox. This is NOT a hardened sandbox (no network/filesystem
isolation); it only bounds time and cleans up stray processes. Run on a dev machine.
"""
import os
import signal
import subprocess
import time
from dataclasses import dataclass, field

DEFAULT_TIMEOUTS = {"build": 120, "test": 60, "probe": 20}


@dataclass
class StageResult:
    stage: str
    status: str  # passed | failed | skipped
    duration: float = 0.0
    output_excerpt: str = ""
    failure_category: str | None = None
    detail: dict = field(default_factory=dict)

    def to_dict(self):
        return dict(self.__dict__)


def excerpt(text: str, lines: int = 60, chars: int = 6000) -> str:
    tail = "\n".join(text.strip().split("\n")[-lines:])
    return tail[-chars:]


def kill_group(proc):
    try:
        os.killpg(proc.pid, signal.SIGKILL)
    except (ProcessLookupError, PermissionError):
        pass


def run_cmd(cmd, cwd, timeout, env=None):
    """Returns (returncode, combined_output, timed_out, seconds)."""
    t0 = time.monotonic()
    proc = subprocess.Popen(cmd, cwd=cwd, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True,
                            start_new_session=True, env={**os.environ, **(env or {})})
    try:
        out, _ = proc.communicate(timeout=timeout)
        timed_out = False
    except subprocess.TimeoutExpired:
        kill_group(proc)
        out, _ = proc.communicate()
        timed_out = True
    kill_group(proc)  # stray children that outlived the parent
    return proc.returncode, out or "", timed_out, time.monotonic() - t0


def passed(stage, dur, out="", **detail):
    return StageResult(stage, "passed", dur, excerpt(out), None, detail)


def failed(stage, dur, category, out, **detail):
    return StageResult(stage, "failed", dur, excerpt(out), category, detail)
