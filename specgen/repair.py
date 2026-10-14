"""Generate -> verify -> repair loop for one spec. Plain Python, no agent framework.

Flow: ask the LLM for files, verify; on failure send the spec, current files, failing
stage and a TRIMMED error (last ~60 lines) back, accept only the files it returns, re-verify.
At most `max_repairs` repairs. Every prompt/completion is logged to calls.jsonl.
Anti-cheating: a repair that reduces the number of test functions gets its test files reverted
(other fixes still apply), and the attempt is flagged; the test stage still enforces the spec minimum.
Formatting: the model's .go files are run through gofmt before verification (autoformat), and which
files that changed is logged per attempt. Deliberately does not: accumulate chat history.
"""
import json
import time
from pathlib import Path

from specgen.generate import SYSTEM_PROMPT, build_prompt, autoformat, count_tests, spec_block, write_project
from specgen.llm.base import LLMError
from specgen.parse import ParseError, check_required, parse_files
from specgen.verify.pipeline import VerifyResult, parse_failure, verify

MAX_REPAIRS = 3


def repair_prompt(spec, files, fail, last_raw):
    if files:
        current = "\n".join(f"=== FILE: {n} ===\n{c.rstrip()}\n=== END FILE ===" for n, c in files.items())
    else:
        current = "(your previous answer could not be parsed; it began with:)\n" + (last_raw or "")[:1500]
    return f"""REPAIR REQUEST. The service below failed verification at stage '{fail.stage}' (category: {fail.failure_category}).

{spec_block(spec)}

Current files:
{current}

Error output (trimmed to the last lines):
{fail.output_excerpt}

Fix EVERY problem you can find in these files, not only the first error line (the compiler and vet show just
a few errors at a time, so re-check imports, unused variables and names). Return corrected full contents ONLY for the files that need changes, in the same
=== FILE: name === / === END FILE === format. Do not delete or weaken tests (need at least
{spec.min_tests} test functions). Do not touch main.go or metrics.go. If the answer could not be
parsed, return ALL files (handlers.go, handlers_test.go, README.md).
"""


class CallLog:
    def __init__(self, path):
        self.path = Path(path)
        self.tokens_in = self.tokens_out = 0
        self.llm_time = 0.0

    def call(self, client, purpose, system, user):
        c = client.complete(system, user)
        self.tokens_in += c.prompt_tokens
        self.tokens_out += c.completion_tokens
        self.llm_time += c.latency_s
        with open(self.path, "a") as f:
            f.write(json.dumps({"purpose": purpose, "system": system, "prompt": user, "completion": c.text,
                                "prompt_tokens": c.prompt_tokens, "completion_tokens": c.completion_tokens,
                                "latency_s": round(c.latency_s, 3), "cached": c.cached}) + "\n")
        return c


def _verify_text(text, spec, project, base_files, require_all, timeouts, on_stage):
    """Parse an LLM answer, merge into base files, write project, verify. Returns (files, VerifyResult)."""
    try:
        new = parse_files(text)
        files = {**(base_files or {}), **new}
        if require_all:
            check_required(files)
    except ParseError as e:
        return base_files, parse_failure(str(e)), None, []
    old_tests, note = (count_tests(base_files) if base_files else 0), None
    if base_files and count_tests(files) < old_tests:
        # Never let a repair weaken the tests: keep the original test files, still apply the other fixes.
        files = {**files, **{n: c for n, c in base_files.items() if n.endswith("_test.go")}}
        note = "weakened_tests_reverted"
    files, fmt_changed = autoformat(files)
    write_project(project, spec, files)
    return files, verify(project, spec, timeouts, on_stage), note, fmt_changed


def run_spec(spec, client, run_dir, max_repairs=MAX_REPAIRS, timeouts=None, on_stage=None):
    """Returns a result dict (also written to run_dir/result.json)."""
    run_dir = Path(run_dir)
    run_dir.mkdir(parents=True, exist_ok=True)
    (run_dir / "calls.jsonl").write_text("")
    log, project = CallLog(run_dir / "calls.jsonl"), run_dir / "project"
    t0 = time.monotonic()
    attempts, files, raw, vr = [], None, "", None
    res = {"spec": spec.name, "split": spec.split, "model": client.name, "mock": client.is_mock}
    try:
        raw = log.call(client, "generate", SYSTEM_PROMPT, build_prompt(spec)).text
        files, vr, _, fmt = _verify_text(raw, spec, project, None, True, timeouts, on_stage)
        attempts.append(_attempt(0, vr, "generate", gofmt_fixed=fmt))
        for n in range(1, max_repairs + 1):
            if vr.ok:
                break
            fail = vr.first_failure
            raw = log.call(client, f"repair{n}", SYSTEM_PROMPT, repair_prompt(spec, files, fail, raw)).text
            files, vr, note, fmt = _verify_text(raw, spec, project, files, files is None, timeouts, on_stage)
            attempts.append(_attempt(n, vr, f"repair{n}", note, fail, fmt))
    except LLMError as e:
        vr = parse_failure(str(e))
        vr.stages[0].failure_category = "llm_error"
        vr.stages[0].stage = "llm"
        attempts.append(_attempt(len(attempts), vr, "llm_error"))
    first = attempts[0]
    res.update({
        "pass_at_1": first["ok"], "final_pass": vr.ok, "repairs_used": len(attempts) - 1,
        "first_failure_stage": first["failed_stage"], "first_failure_category": first["category"],
        "final_failure_category": None if vr.ok else vr.first_failure.failure_category,
        "tokens_in": log.tokens_in, "tokens_out": log.tokens_out, "llm_time_s": round(log.llm_time, 2),
        "wall_time_s": round(time.monotonic() - t0, 2), "skipped": vr.skipped, "attempts": attempts,
    })
    (run_dir / "result.json").write_text(json.dumps(res, indent=2))
    return res


def _attempt(n, vr, kind, note=None, prev_fail=None, gofmt_fixed=()):
    f = vr.first_failure
    return {"n": n, "kind": kind, "ok": vr.ok, "failed_stage": f.stage if f else None,
            "category": f.failure_category if f else None, "note": note, "gofmt_fixed": list(gofmt_fixed),
            "fixed_previous": bool(prev_fail) and (vr.ok or (f is not None and f.stage != prev_fail.stage and
                                                              _order(f.stage) > _order(prev_fail.stage))),
            "stages": [s.to_dict() for s in vr.stages]}


def _order(stage):
    from specgen.verify.pipeline import ALL_STAGES
    return ALL_STAGES.index(stage) if stage in ALL_STAGES else -1
