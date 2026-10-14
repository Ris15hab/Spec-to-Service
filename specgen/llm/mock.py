"""Deterministic fake model so the whole pipeline (incl. repair) runs offline.

Why: tests and the demo fallback must not depend on an API. The mock reads the spec
from the <spec> block in the prompt and writes a generic but valid service for it.
Behaviors (consumed one per call, last repeats): good | compile_error | garbage | weaken_tests.
Deliberately does not: understand the English `behaviors`; it is a plumbing test double,
and its pass rates say nothing about real models. Output is always labeled mock.
"""
import json
import re

import yaml

from specgen.llm.base import Completion, LLMClient
from specgen.spec import parse_spec


def _go_path(p):
    return re.sub(r"\{[^}]*\}", "x", p)


def good_files(spec) -> dict:
    eps = [e for e in spec.endpoints if not e.builtin]
    routes = []
    for e in eps:
        k = json.dumps(e.path)
        if e.method == "POST":
            m = re.search(r"\{([\w, ]+)\}", e.description)
            req = [f.strip() for f in m.group(1).split(",")] if m else []
            missing = "".join(f' || in["{f}"] == nil' for f in req)
            body = (f'\t\tvar in map[string]any\n\t\tif err := json.NewDecoder(r.Body).Decode(&in); err != nil || len(in) == 0{missing} {{\n'
                    f'\t\t\twriteJSON(w, http.StatusBadRequest, map[string]string{{"error": "invalid body"}})\n\t\t\treturn\n\t\t}}\n'
                    f'\t\tmu.Lock()\n\t\tstore[{k}] = append(store[{k}], in)\n\t\tmu.Unlock()\n' +
                    '\t\twriteJSON(w, http.StatusCreated, in)\n')
        elif e.method == "GET":
            body = (f'\t\tmu.Lock()\n\t\titems := append([]map[string]any{{}}, store[{k}]...)\n\t\tmu.Unlock()\n' +
                    '\t\twriteJSON(w, http.StatusOK, items)\n')
        else:
            body = '\t\twriteJSON(w, http.StatusOK, map[string]bool{"ok": true})\n'
        routes.append(f'\tmux.HandleFunc("{e.method} {e.path}", func(w http.ResponseWriter, r *http.Request) {{\n{body}\t}})')
    handlers = f'''package main

import (
	"encoding/json"
	"net/http"
	"sync"
)

var (
	mu    sync.Mutex
	store = map[string][]map[string]any{{}}
)

func writeJSON(w http.ResponseWriter, code int, v any) {{
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}}

func registerRoutes(mux *http.ServeMux) {{
{chr(10).join(routes)}
}}
'''
    all_eps = "\n".join(f'\t\t{{"{e.method}", "{_go_path(e.path)}"}},' for e in spec.endpoints)
    names = "\n".join(f'\t\t"{m.name}",' for m in spec.metrics)
    tests = f'''package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealth(t *testing.T) {{
	srv := httptest.NewServer(newServer())
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/health")
	if err != nil || resp.StatusCode != 200 {{
		t.Fatalf("health failed: %v", err)
	}}
}}

func TestMetricsExposed(t *testing.T) {{
	srv := httptest.NewServer(newServer())
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/metrics")
	if err != nil {{
		t.Fatal(err)
	}}
	b, _ := io.ReadAll(resp.Body)
	for _, n := range []string{{
{names}
	}} {{
		if !strings.Contains(string(b), n) {{
			t.Errorf("metric %s missing", n)
		}}
	}}
}}

func TestEndpointsRespond(t *testing.T) {{
	srv := httptest.NewServer(newServer())
	defer srv.Close()
	for _, e := range [][2]string{{
{all_eps}
	}} {{
		req, _ := http.NewRequest(e[0], srv.URL+e[1], strings.NewReader("{{}}"))
		resp, err := http.DefaultClient.Do(req)
		if err != nil {{
			t.Fatal(err)
		}}
		if resp.StatusCode == 404 || resp.StatusCode >= 500 {{
			t.Errorf("%s %s -> %d", e[0], e[1], resp.StatusCode)
		}}
	}}
}}
'''
    ep_lines = "\n".join(f"- `{e.method} {e.path}`" for e in spec.endpoints)
    m_lines = "\n".join(f"- `{m.name}` ({m.type}): {m.help}" for m in spec.metrics)
    readme = f"# {spec.name}\n\n{spec.description}\n\n## Endpoints\n\n{ep_lines}\n\n## Metrics\n\n{m_lines}\n"
    return {"handlers.go": handlers, "handlers_test.go": tests, "README.md": readme}


def render(files: dict) -> str:
    return "\n".join(f"=== FILE: {n} ===\n{c.rstrip()}\n=== END FILE ===" for n, c in files.items())


class MockClient(LLMClient):
    def __init__(self, name="mock", behavior=("good",)):
        super().__init__(name=name, is_mock=True)
        self.behavior, self.n = list(behavior), 0

    def complete(self, system, user, temperature=0.2):
        m = re.search(r"<spec>\n(.*?)</spec>", user, re.S)
        spec = parse_spec(yaml.safe_load(m.group(1)))
        mode = self.behavior[min(self.n, len(self.behavior) - 1)]
        self.n += 1
        files = good_files(spec)
        if mode == "compile_error":
            files["handlers.go"] = files["handlers.go"].replace("func registerRoutes", "var broken int = \"x\"\n\nfunc registerRoutes")
        elif mode == "weaken_tests":
            files["handlers_test.go"] = "package main\n\nimport \"testing\"\n\nfunc TestNothing(t *testing.T) {}\n"
        text = "I could not do that." if mode == "garbage" else "Here you go:\n" + render(files)
        return Completion(text=text, prompt_tokens=len(user) // 4, completion_tokens=len(text) // 4)
