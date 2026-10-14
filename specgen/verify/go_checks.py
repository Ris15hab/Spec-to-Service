"""Go toolchain stages: format, build, vet, test.

Why: cheapest, most objective signals first. Build runs before vet so type errors are
categorized `compile_error` (go vet would report them as vet failures).
Deliberately does not: run linters beyond gofmt/vet, or modify the code under test.
"""
import re
from pathlib import Path

from specgen.verify.common import failed, passed, run_cmd

STDLIB_OK_PREFIX = "github.com/prometheus/client_golang"
_IMPORT_BLOCK = re.compile(r'import\s*\((.*?)\)|import\s+(?:\w+\s+)?"([^"]+)"', re.S)


def _third_party_imports(project: Path):
    bad = set()
    for f in project.glob("*.go"):
        for block, single in _IMPORT_BLOCK.findall(f.read_text()):
            for path in re.findall(r'"([^"]+)"', block) if block else [single]:
                if "." in path.split("/")[0] and not path.startswith(STDLIB_OK_PREFIX):
                    bad.add(path)
    return sorted(bad)


def check_format(project, timeouts):
    rc, out, to, dur = run_cmd(["gofmt", "-l", "."], project, 30)
    if rc != 0 or out.strip():
        return failed("format", dur, "format", "files need gofmt:\n" + out)
    return passed("format", dur)


def check_build(project, timeouts):
    bad = _third_party_imports(project)
    if bad:
        return failed("build", 0, "compile_error", f"disallowed third-party imports: {bad} (stdlib + prometheus only)")
    t = timeouts["build"]
    rc, out, to, dur = run_cmd(["go", "mod", "tidy"], project, t)
    if to:
        return failed("build", dur, "timeout", "go mod tidy timed out")
    if rc != 0:
        return failed("build", dur, "compile_error", "go mod tidy failed:\n" + out)
    rc, out2, to, dur2 = run_cmd(["go", "build", "-o", "svc.bin", "."], project, t)
    if to:
        return failed("build", dur + dur2, "timeout", out2)
    if rc != 0:
        return failed("build", dur + dur2, "compile_error", out2)
    return passed("build", dur + dur2)


def check_vet(project, timeouts):
    rc, out, to, dur = run_cmd(["go", "vet", "./..."], project, timeouts["build"])
    if to:
        return failed("vet", dur, "timeout", out)
    return failed("vet", dur, "vet", out) if rc != 0 else passed("vet", dur)


def count_tests(project: Path) -> int:
    return sum(len(re.findall(r"^func Test\w*\(", f.read_text(), re.M)) for f in project.glob("*_test.go"))


def check_test(project, timeouts, min_tests=3):
    n = count_tests(Path(project))
    if n < min_tests:
        return failed("test", 0, "no_tests", f"found {n} Test functions, need at least {min_tests}")
    limit = timeouts["test"]
    go_limit = f"{max(1, int(limit * 0.75))}s"
    rc, out, to, dur = run_cmd(["go", "test", "./...", "-count=1", f"-timeout={go_limit}"], project, limit + 10)
    if to or "test timed out" in out:
        return failed("test", dur, "timeout", out, tests=n)
    if rc != 0:
        return failed("test", dur, "test_failure", out, tests=n)
    return passed("test", dur, out, tests=n)
