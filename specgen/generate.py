"""Turn a Spec into a project: deterministic templates + LLM-written handlers/tests/docs.

Why: scaffolding (server wiring, metrics registry, Dockerfile, k8s) never varies,
so templates make it correct by construction; the LLM only writes what varies
(handlers.go, store.go, tests, README), which keeps the task small and checkable.
Also autoformat(): deterministic gofmt on the model's files before verification (the model cannot align
whitespace reliably; which files it changed is recorded so the report can show it).
Deliberately does not: let the LLM touch template-owned files (parse.py rejects that).
"""
import re
import shutil
import subprocess
from pathlib import Path

import jinja2
import yaml

from specgen.spec import Spec

TEMPLATES = Path(__file__).resolve().parent.parent / "templates"
TEMPLATE_FILES = {
    "go.mod.j2": "go.mod", "main.go.j2": "main.go", "metrics.go.j2": "metrics.go",
    "Dockerfile.j2": "Dockerfile", "Makefile.j2": "Makefile",
    "deployment.yaml.j2": "k8s/deployment.yaml", "service.yaml.j2": "k8s/service.yaml",
}

SYSTEM_PROMPT = """You are an expert Go engineer writing a small, correct HTTP service.
Rules:
- Go 1.22+, package main. Use ONLY the standard library (net/http routing with Go 1.22 patterns
  like mux.HandleFunc("POST /jobs", h) and r.PathValue("id")) plus the Prometheus variables
  already declared in metrics.go. No other third-party imports.
- Output ONLY file blocks in exactly this format, with no other prose:
=== FILE: handlers.go ===
<file content>
=== END FILE ===
- Code must be gofmt-formatted (tabs for indentation, all imports in one block right after `package main`),
  pass `go vet`, and be safe for concurrent requests (use sync.Mutex).
- State isolation: do NOT keep service data in package-level variables. Create the store INSIDE
  registerRoutes(mux), so every call to newServer() starts empty and tests cannot affect each other.
  The Prometheus variables in metrics.go are global and shared by all tests, so tests must not assert
  absolute metric values (compare before/after, or skip).
- Before answering, re-read every file: each import must be used, every identifier (also in tests) must be
  defined or imported, and do not redefine `newServer` or other names that main.go/metrics.go declare.
"""


def go_ident(name: str) -> str:
    parts = name.split("_")
    return parts[0] + "".join(p.capitalize() for p in parts[1:])


def render_templates(spec: Spec) -> dict:
    env = jinja2.Environment(loader=jinja2.FileSystemLoader(TEMPLATES), keep_trailing_newline=True,
                             undefined=jinja2.StrictUndefined)
    ctx = {
        "name": spec.name, "module": spec.name.replace("-", "_"),
        "metrics": [{"name": m.name, "type": m.type, "help": m.help.replace('"', "'"),
                     "ident": go_ident(m.name)} for m in spec.metrics],
    }
    return {out: env.get_template(src).render(**ctx) for src, out in TEMPLATE_FILES.items()}


def metrics_summary(spec: Spec) -> str:
    return "\n".join(f"- {go_ident(m.name)}  ({m.type}) -> exposed as `{m.name}`; "
                     f"use {'Set/Inc/Dec/Add' if m.type == 'gauge' else 'Inc/Add'}" for m in spec.metrics)


def spec_block(spec: Spec) -> str:
    return "<spec>\n" + yaml.safe_dump(spec.to_dict(), sort_keys=False) + "</spec>"


def build_prompt(spec: Spec) -> str:
    endpoints = [e for e in spec.endpoints if not e.builtin]
    return f"""Implement this service.

{spec_block(spec)}

Already provided by templates (do NOT write or register these):
- main.go: `func newServer() http.Handler` serves GET /health and GET /metrics, then calls
  `registerRoutes(mux)`. Do not register /health or /metrics yourself.
- metrics.go: Prometheus variables you must keep up to date from your handlers:
{metrics_summary(spec)}

You must write these files:
1. handlers.go - defines `func registerRoutes(mux *http.ServeMux)` that registers ONLY these routes:
{chr(10).join(f'   - {e.method} {e.path}  {e.description}' for e in endpoints) or '   (none)'}
   Keep state in memory (you may add store.go). Return JSON with proper status codes; reject
   malformed input with 400. The behaviors in the spec are verified at runtime.
2. handlers_test.go - at least {spec.min_tests} `func TestXxx(t *testing.T)` functions using
   net/http/httptest against `newServer()`. Never bind real ports; tests must finish in seconds.
3. README.md - must contain: an "## Endpoints" section listing EVERY spec endpoint (including
   /health and /metrics) as `METHOD /path` in backticks, and a "## Metrics" section listing EVERY
   metric name in backticks. Only document things that actually exist.
"""


def write_project(dest, spec: Spec, llm_files: dict) -> None:
    dest = Path(dest)
    if dest.exists():
        shutil.rmtree(dest)
    for rel, content in {**render_templates(spec), **llm_files}.items():
        p = dest / rel
        p.parent.mkdir(parents=True, exist_ok=True)
        p.write_text(content)


def count_tests(files: dict) -> int:
    """Distinct Test function names (a duplicated name cannot compile, so removing one is not weakening)."""
    names = set()
    for n, c in files.items():
        if n.endswith("_test.go"):
            names |= set(re.findall(r"^func (Test\w*)\(", c, re.M))
    return len(names)


def autoformat(files: dict):
    """Run `gofmt` on the model's .go files (what an editor does on save). Returns (files, names_changed).
    Code with syntax errors is left untouched, so the format stage still reports it."""
    out, changed = {}, []
    for name, content in files.items():
        if name.endswith(".go"):
            r = subprocess.run(["gofmt"], input=content, capture_output=True, text=True, timeout=30)
            if r.returncode == 0 and r.stdout != content:
                content = r.stdout
                changed.append(name)
        out[name] = content
    return out, changed
