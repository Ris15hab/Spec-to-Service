import pytest

from conftest import ROOT
from specgen.llm.base import LLMError
from specgen.llm.mock import MockClient
from specgen.parse import ParseError, check_required, parse_files
from specgen.repair import run_spec


def test_parse_tolerates_fences_prose_and_missing_end():
    text = "Sure!\n=== FILE: handlers.go ===\n```go\npackage main\n```\n=== END FILE ===\nthanks\n=== FILE: README.md ===\n# hi"
    files = parse_files(text)
    assert files["handlers.go"] == "package main\n"
    assert files["README.md"] == "# hi\n"


@pytest.mark.parametrize("text", ["no blocks here", "=== FILE: main.go ===\npackage main\n=== END FILE ===",
                                  "=== FILE: ../x.go ===\na\n=== END FILE ===", "=== FILE: a.txt ===\na\n=== END FILE ==="])
def test_parse_errors(text):
    with pytest.raises(ParseError):
        parse_files(text)


def test_required_files():
    with pytest.raises(ParseError):
        check_required({"handlers.go": "x"})


def test_garbage_then_good_is_repaired_as_parse_error(tmp_path, spec):
    res = run_spec(spec, MockClient("m", ["garbage", "good"]), tmp_path)
    assert res["first_failure_category"] == "parse_error" and res["final_pass"] and res["repairs_used"] == 1


def test_weakened_tests_are_reverted_but_other_fixes_apply(tmp_path, spec):
    res = run_spec(spec, MockClient("m", ["compile_error", "weaken_tests"]), tmp_path)
    assert res["final_pass"] and res["attempts"][1]["note"] == "weakened_tests_reverted"
    from specgen.generate import count_tests
    from specgen.parse import parse_files
    assert count_tests({n: (tmp_path / "project" / n).read_text() for n in ["handlers_test.go"]}) == 3


def test_persistent_failure_stops_after_three_repairs(tmp_path, spec):
    res = run_spec(spec, MockClient("m", ["compile_error"]), tmp_path)
    assert not res["final_pass"] and res["repairs_used"] == 3


def test_llm_error_category(tmp_path, spec):
    class Boom(MockClient):
        def complete(self, *a, **k):
            raise LLMError("down")
    res = run_spec(spec, Boom("m"), tmp_path)
    assert res["first_failure_category"] == "llm_error" and not res["final_pass"]


def test_autoformat_fixes_spacing_but_leaves_broken_code():
    from specgen.generate import autoformat
    ok, changed = autoformat({"a.go": 'package main\n\nvar x = map[string]int{\n"a":   1,\n"bbb":  2,\n}\n', "README.md": "x"})
    assert changed == ["a.go"] and '\t"a":' in ok["a.go"] and ok["README.md"] == "x"
    bad, changed = autoformat({"b.go": "package main\nfunc ("})
    assert changed == [] and bad["b.go"] == "package main\nfunc ("


def test_count_tests_counts_distinct_names():
    from specgen.generate import count_tests
    dup = {"a_test.go": "func TestA(t *testing.T) {}\nfunc TestA(t *testing.T) {}\nfunc TestB(t *testing.T) {}\n"}
    assert count_tests(dup) == 2
