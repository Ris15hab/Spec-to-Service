"""`specgen demo`: one spec end to end with live stage output, ending in a results table.

Why: the live demo must be short, visible and survive an API outage.
Fallback: if the chosen model cannot be reached, it switches to the mock model and says so
loudly (a banner, and the model name `mock-flaky`), so mock output is never mistaken for real.
Deliberately does not: hide failures; a repair is shown as it happens.
"""
import argparse
import json
from pathlib import Path

from rich.table import Table

from specgen.llm import get_client
from specgen.llm.base import LLMError
from specgen.repair import run_spec
from specgen.spec import load_spec


def cli(cmd, rest, console):
    from specgen.cli import show_stage
    ap = argparse.ArgumentParser(prog="specgen demo")
    ap.add_argument("spec", nargs="?", default="specs/dev/gpu-job-watcher.yaml")
    ap.add_argument("--model", default="mock-flaky")
    ap.add_argument("--out", default="out/demo")
    a = ap.parse_args(rest)
    spec = load_spec(a.spec)
    try:
        client = get_client(a.model)
    except LLMError as e:
        console.print(f"[bold red]model unavailable:[/] {e}")
        client = get_client("mock-flaky")
    if client.is_mock:
        console.print("[bold black on yellow] MOCK MODEL: canned output (first answer has a deliberate bug) so the repair loop can be shown. Not a real LLM. [/]")
    console.print(f"\n[bold]{spec.name}[/] via [cyan]{client.name}[/]\n")
    try:
        res = run_spec(spec, client, a.out + "-run", on_stage=lambda r: show_stage(r))
    except LLMError as e:
        console.print(f"[red]{e}[/]")
        return 1
    t = Table(title=f"attempts ({'MOCK' if client.is_mock else client.name})")
    for c in ("attempt", "result", "stage", "category"):
        t.add_column(c)
    for at in res["attempts"]:
        t.add_row("generate" if at["n"] == 0 else f"repair {at['n']}", "[green]pass[/]" if at["ok"] else "[red]fail[/]",
                  at["failed_stage"] or "-", at["category"] or "-")
    console.print(t)
    console.print(f"pass@1={res['pass_at_1']}  final_pass={res['final_pass']}  tokens={res['tokens_in']}+{res['tokens_out']}  "
                  f"wall={res['wall_time_s']}s  skipped={res['skipped']}")
    summ = Path("results/summary.json")
    if summ.exists():
        s = json.loads(summ.read_text())
        console.print("\n[bold]Latest measured results (results/summary.json, real models only):[/]")
        for k, v in s["tables"].items():
            console.print(f"  {k}: pass@1 {v['pass_at_1_pct']}%, after repair {v['final_pass_pct']}% over {v['n_runs']} runs")
    return 0 if res["final_pass"] else 1
