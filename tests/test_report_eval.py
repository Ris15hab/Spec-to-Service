import json
import shutil

import pytest

from conftest import ROOT
from specgen import evalrun, report


def fake(model, spec, split, trial, p1, final, cat=None, mock=False):
    return {"spec": spec, "split": split, "model": model, "mock": mock, "trial": trial, "pass_at_1": p1,
            "final_pass": final, "repairs_used": 0 if p1 else 1, "first_failure_stage": None if p1 else "build",
            "first_failure_category": None if p1 else cat, "final_failure_category": None if final else cat,
            "tokens_in": 100, "tokens_out": 50, "wall_time_s": 2.0, "skipped": ["manifests:hadolint"],
            "attempts": [{"n": 0, "ok": p1, "failed_stage": None if p1 else "build", "category": None if p1 else cat, "note": None},
                         {"n": 1, "ok": final, "failed_stage": None, "category": None, "note": None}]}


def test_report_aggregates_and_is_deterministic(tmp_path, monkeypatch):
    monkeypatch.chdir(tmp_path)
    d = tmp_path / "runs" / "r1"
    rows = [fake("m", "a", "dev", 1, True, True), fake("m", "b", "dev", 1, False, True, "compile_error"),
            fake("m", "c", "heldout", 1, False, False, "probe_behavior"), fake("mock", "a", "dev", 1, True, True, mock=True)]
    for i, r in enumerate(rows):
        p = d / r["model"] / r["spec"] / "t1"
        p.mkdir(parents=True)
        (p / "result.json").write_text(json.dumps(r))
    (d / "meta.json").write_text(json.dumps({"run_id": "r1", "cache": False, "fingerprint": "abc"}))
    res, metas = report.load(["r1"])
    s = report.build(res, metas)
    assert s["tables"]["m|dev"]["pass_at_1_pct"] == 50.0 and s["tables"]["m|dev"]["final_pass_pct"] == 100.0
    assert s["tables"]["m|heldout"]["final_pass_pct"] == 0.0
    assert s["first_failure"]["m"] == {"compile_error": 1, "probe_behavior": 1}
    assert s["per_spec"]["b"]["models"]["m"] == [1, 0, 1]
    assert "mock|dev" in s["mock_tables"] and "mock|dev" not in s["tables"]
    assert report.render(s, res) == report.render(report.build(*report.load(["r1"])), res)


def test_heldout_lock_detects_edits(tmp_path, monkeypatch):
    (tmp_path / "specs").mkdir()
    shutil.copytree(ROOT / "specs/heldout", tmp_path / "specs/heldout")
    monkeypatch.chdir(tmp_path)
    with pytest.raises(SystemExit):
        evalrun.check_lock()  # no lock yet
    evalrun.freeze()
    evalrun.check_lock()
    with pytest.raises(SystemExit):
        evalrun.freeze()  # cannot silently re-freeze
    f = next((tmp_path / "specs/heldout").glob("*.yaml"))
    f.write_text(f.read_text() + "# tweak\n")
    with pytest.raises(SystemExit):
        evalrun.check_lock()
