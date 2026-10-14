"""Command line: specgen generate | verify | freeze | eval | report | demo.

Why: one entry point so the demo, the eval and ad-hoc use all share the same code paths.
Deliberately does not: hold logic; every command calls into a module.
"""
import argparse
import os
import sys
from pathlib import Path

from rich.console import Console

from specgen.spec import SpecError, load_spec

console = Console()
DELEGATED = {"freeze": "evalrun", "eval": "evalrun", "report": "report", "demo": "demo"}
ICON = {"passed": "[green]PASS[/]", "failed": "[red]FAIL[/]", "skipped": "[yellow]SKIP[/]"}


def load_dotenv(path=".env"):
    """Tiny .env reader (no dependency). Real environment variables win."""
    if Path(path).exists():
        for line in Path(path).read_text().splitlines():
            if "=" in line and not line.lstrip().startswith("#"):
                k, v = line.split("=", 1)
                if v.strip():
                    os.environ.setdefault(k.strip(), v.strip())


def show_stage(r):
    extra = f"  [{r.failure_category}]" if r.failure_category else ""
    tools = r.detail.get("skipped_tools")
    note = f"  (tools skipped: {', '.join(tools)})" if tools else ""
    console.print(f"  {ICON[r.status]} {r.stage:<10} {r.duration:5.1f}s{extra}{note}")
    if r.status == "failed":
        for line in r.output_excerpt.split("\n")[-8:]:
            console.print(f"       [dim]{line}[/]", markup=True, highlight=False)


def cmd_generate(a):
    from specgen.llm import get_client
    from specgen.repair import run_spec
    spec = load_spec(a.spec)
    client = get_client(a.model, cache_dir=".cache/llm" if a.cache else None)
    out = Path(a.out or f"out/{spec.name}")
    label = " [bold yellow](MOCK MODEL: canned output, not a real LLM)[/]" if client.is_mock else ""
    console.print(f"[bold]{spec.name}[/] with [cyan]{a.model}[/]{label}")
    res = run_spec(spec, client, out, max_repairs=0 if a.no_repair else 3,
                   on_stage=show_stage)
    console.print(f"result: pass@1={res['pass_at_1']} final_pass={res['final_pass']} repairs={res['repairs_used']} "
                  f"tokens={res['tokens_in']}+{res['tokens_out']} -> project in {out}/project")
    return 0 if res["final_pass"] else 1


def cmd_verify(a):
    from specgen.verify.pipeline import verify
    spec = load_spec(a.spec)
    vr = verify(a.dir, spec, on_stage=show_stage)
    return 0 if vr.ok else 1


def main(argv=None):
    load_dotenv()
    ap = argparse.ArgumentParser(prog="specgen")
    sub = ap.add_subparsers(dest="cmd", required=True)
    g = sub.add_parser("generate", help="generate, verify and repair a service from a spec")
    g.add_argument("spec")
    g.add_argument("--model", default="mock")
    g.add_argument("--out")
    g.add_argument("--no-repair", action="store_true")
    g.add_argument("--cache", action="store_true", help="dev-only LLM response cache")
    g.set_defaults(fn=cmd_generate)
    v = sub.add_parser("verify", help="verify an existing project directory")
    v.add_argument("dir")
    v.add_argument("--spec", required=True)
    v.set_defaults(fn=cmd_verify)
    for name in DELEGATED:
        sub.add_parser(name, help=f"see `specgen {name} -h`")
    argv = sys.argv[1:] if argv is None else argv
    try:
        if argv and argv[0] in DELEGATED:  # these parse their own flags
            import importlib
            return importlib.import_module(f"specgen.{DELEGATED[argv[0]]}").cli(argv[0], argv[1:], console) or 0
        return (a := ap.parse_args(argv)).fn(a)
    except SpecError as e:
        console.print(f"[red]spec error:[/] {e}")
        return 2


if __name__ == "__main__":
    sys.exit(main())
