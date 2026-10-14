"""Ordered verification stages; stops at the first failure.

Order: parse, format, build, vet, test, probe, docs, manifests.
Why one pipeline: the same function judges first attempts, repairs and eval runs, so
numbers are comparable. Deliberately does not: continue past a failure (later stages
would only produce noise), or retry flaky stages.
"""
from dataclasses import dataclass, field
from pathlib import Path

from specgen.verify import go_checks
from specgen.verify.common import DEFAULT_TIMEOUTS, StageResult, passed, failed

ALL_STAGES = ["parse", "format", "build", "vet", "test", "probe", "docs", "manifests"]
CATEGORIES = ["parse_error", "format", "vet", "compile_error", "test_failure", "no_tests", "probe_health",
              "probe_metrics", "probe_endpoint", "probe_behavior", "docs_mismatch", "timeout", "llm_error", "manifest_error"]


@dataclass
class VerifyResult:
    stages: list = field(default_factory=list)

    @property
    def ok(self):
        return all(s.status != "failed" for s in self.stages)

    @property
    def first_failure(self):
        return next((s for s in self.stages if s.status == "failed"), None)

    @property
    def skipped(self):
        out = [s.stage for s in self.stages if s.status == "skipped"]
        for s in self.stages:
            out += [f"{s.stage}:{t}" for t in s.detail.get("skipped_tools", [])]
        return out

    def to_dict(self):
        return {"ok": self.ok, "stages": [s.to_dict() for s in self.stages]}


def parse_failure(message: str) -> VerifyResult:
    return VerifyResult([failed("parse", 0, "parse_error", message)])


def verify(project, spec, timeouts=None, on_stage=None) -> VerifyResult:
    """Run all stages on `project`. `on_stage(StageResult)` lets the CLI print live progress."""
    from specgen.verify import docs_check, manifests, probes
    timeouts = {**DEFAULT_TIMEOUTS, **(timeouts or {})}
    project = Path(project).resolve()
    result = VerifyResult([passed("parse", 0)])
    if on_stage:
        on_stage(result.stages[0])
    steps = [
        ("format", lambda: go_checks.check_format(project, timeouts)),
        ("build", lambda: go_checks.check_build(project, timeouts)),
        ("vet", lambda: go_checks.check_vet(project, timeouts)),
        ("test", lambda: go_checks.check_test(project, timeouts, spec.min_tests)),
    ]
    service = probes.Service(project, timeouts["probe"])
    steps += [
        ("probe", lambda: probes.check_probes(service, spec)),
        ("docs", lambda: docs_check.check_docs(project, spec, service)),
        ("manifests", lambda: manifests.check_manifests(project)),
    ]
    try:
        for name, fn in steps:
            r = fn()
            result.stages.append(r)
            if on_stage:
                on_stage(r)
            if r.status == "failed":
                break
    finally:
        service.stop()
    return result
