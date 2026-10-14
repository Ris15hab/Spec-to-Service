import sys
from pathlib import Path

import pytest

sys.path.insert(0, str(Path(__file__).resolve().parent.parent))

from specgen.generate import write_project
from specgen.llm.mock import good_files
from specgen.spec import load_spec
from specgen.verify.pipeline import verify

ROOT = Path(__file__).resolve().parent.parent
TRIVIAL_TESTS = """package main

import "testing"

func TestA(t *testing.T) {}
func TestB(t *testing.T) {}
func TestC(t *testing.T) {}
"""


@pytest.fixture(scope="session")
def spec():
    return load_spec(ROOT / "specs/dev/gpu-job-watcher.yaml")


@pytest.fixture
def run(tmp_path, spec):
    """run(mutate) -> VerifyResult for the mock's good project after `mutate(files)`.
    mutate may also return {'metrics_drop': name} style overrides via post(project)."""
    def _run(mutate=None, post=None, timeouts=None):
        files = good_files(spec)
        if mutate:
            mutate(files)
        proj = tmp_path / "p"
        write_project(proj, spec, files)
        if post:
            post(proj)
        return verify(proj, spec, timeouts)
    return _run
