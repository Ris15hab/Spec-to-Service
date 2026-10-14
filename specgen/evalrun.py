"""Eval harness: specs x models x trials -> runs/<run_id>/<model>/<spec>/t<k>/result.json.

Also owns the held-out freeze: `specgen freeze` writes specs/HELDOUT.lock (path + SHA-256 of
every held-out spec). `eval --split heldout|all` refuses to run if any held-out file changed.
Why: credible numbers need a test set nobody tuned against.
Resumable: an existing result.json is skipped, so a rate-limited run can be restarted.
Deliberately does not: aggregate (report.py does, from files only) or retry failed runs.
"""
import argparse
import hashlib
import json
import sys
import threading
from concurrent.futures import ThreadPoolExecutor
from datetime import datetime, timezone
from pathlib import Path

from specgen.llm import get_client
from specgen.llm.base import LLMError
from specgen.repair import run_spec
from specgen.spec import list_specs

LOCK = Path("specs/HELDOUT.lock")
RUNS = Path("runs")
_print_lock = threading.Lock()


def sha(path) -> str:
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def freeze(force=False):
    if LOCK.exists() and not force:
        raise SystemExit(f"{LOCK} already exists; held-out set is frozen (use --force only before any eval run)")
    files = sorted(Path("specs/heldout").glob("*.yaml"))
    LOCK.write_text(json.dumps({"files": {str(f): sha(f) for f in files}}, indent=2) + "\n")
    return len(files)


def check_lock():
    if not LOCK.exists():
        raise SystemExit("specs/HELDOUT.lock missing: run `specgen freeze` before any held-out eval")
    locked = json.loads(LOCK.read_text())["files"]
    actual = {str(f): sha(f) for f in Path("specs/heldout").glob("*.yaml")}
    if locked != actual:
        changed = sorted(set(locked) ^ set(actual) | {k for k in locked if k in actual and locked[k] != actual[k]})
        raise SystemExit(f"held-out specs differ from HELDOUT.lock: {changed}")


def fingerprint() -> str:
    """Hash of the code that builds prompts and runs repair; lets the report show 'no prompt changes'."""
    h = hashlib.sha256()
    for f in ("specgen/generate.py", "specgen/repair.py", "specgen/parse.py"):
        h.update(Path(f).read_bytes())
    return h.hexdigest()[:12]


def _log(msg):
    with _print_lock:
        print(msg, flush=True)


def _run_model(model, specs, trials, run_dir, cache, max_repairs):
    try:
        client = get_client(model, cache_dir=".cache/llm" if cache else None)
    except LLMError as e:
        _log(f"[{model}] SKIPPED: {e}")
        return 0
    done = 0
    for spec in specs:
        for t in range(1, trials + 1):
            d = run_dir / model / spec.name / f"t{t}"
            if (d / "result.json").exists():
                continue
            res = run_spec(spec, client, d, max_repairs=max_repairs)
            res["trial"] = t
            (d / "result.json").write_text(json.dumps(res, indent=2))
            done += 1
            flag = "pass@1" if res["pass_at_1"] else ("repaired" if res["final_pass"] else "FAIL")
            _log(f"[{model}] {spec.name} t{t}: {flag} (repairs={res['repairs_used']}, "
                 f"first_fail={res['first_failure_category']}, {res['wall_time_s']}s)")
    return done


def run_eval(models, split, trials, run_id, only=None, cache=False, max_repairs=3):
    if split in ("heldout", "all"):
        check_lock()
    specs = [s for s in list_specs(split=split) if not only or s.name in only]
    run_dir = RUNS / run_id
    run_dir.mkdir(parents=True, exist_ok=True)
    meta = {"run_id": run_id, "started": datetime.now(timezone.utc).isoformat(timespec="seconds"),
            "models": models, "split": split, "trials": trials, "cache": cache, "max_repairs": max_repairs,
            "fingerprint": fingerprint(), "autoformat": True, "specs": [s.name for s in specs],
            "heldout_lock_sha": sha(LOCK) if LOCK.exists() else None}
    (run_dir / "meta.json").write_text(json.dumps(meta, indent=2))
    with ThreadPoolExecutor(max_workers=len(models)) as ex:  # one thread per model: separate rate limits
        list(ex.map(lambda m: _run_model(m, specs, trials, run_dir, cache, max_repairs), models))
    return run_dir


def cli(cmd, rest, console):
    if cmd == "freeze":
        n = freeze(force="--force" in rest)
        console.print(f"froze {n} held-out specs in {LOCK}")
        return 0
    ap = argparse.ArgumentParser(prog="specgen eval")
    ap.add_argument("--models", required=True, help="comma-separated names from models.yaml")
    ap.add_argument("--split", choices=["dev", "heldout", "all"], default="dev")
    ap.add_argument("--trials", type=int, default=1)
    ap.add_argument("--run-id", default=datetime.now().strftime("run-%Y%m%d-%H%M%S"))
    ap.add_argument("--specs", help="comma-separated spec names to include")
    ap.add_argument("--cache", action="store_true", help="dev-only response cache (never for official runs)")
    ap.add_argument("--max-repairs", type=int, default=3)
    a = ap.parse_args(rest)
    run_dir = run_eval(a.models.split(","), a.split, a.trials, a.run_id,
                       a.specs.split(",") if a.specs else None, a.cache, a.max_repairs)
    console.print(f"done: {run_dir}. Now run: python3 -m specgen report --run-id {a.run_id}")
    return 0
