"""Load and validate a service spec (YAML) into a Spec dataclass.

Why: every other module (prompts, templates, probes, docs check) reads the same
validated object, so a typo in a spec fails early with a clear message.
Deliberately does not: infer anything; unknown keys are rejected, not ignored.
"""
import re
from dataclasses import dataclass, field
from pathlib import Path

import yaml

METHODS = {"GET", "POST", "PUT", "PATCH", "DELETE"}
METRIC_TYPES = {"gauge", "counter"}
BUILTIN_PATHS = {"/health", "/metrics"}  # served by the template, not the LLM


class SpecError(ValueError):
    pass


@dataclass
class Endpoint:
    method: str
    path: str
    description: str = ""

    @property
    def builtin(self):
        return self.path in BUILTIN_PATHS

    @property
    def has_param(self):
        return "{" in self.path

    def concrete_path(self):
        return re.sub(r"\{[^}]*\}", "x", self.path)


@dataclass
class Metric:
    name: str
    type: str
    help: str


@dataclass
class Probe:
    """Machine-checkable behavior: send a request, expect one of the statuses."""
    method: str
    path: str
    expect_status: list
    body: str | None = None


@dataclass
class Spec:
    name: str
    description: str
    endpoints: list
    metrics: list
    behaviors: list = field(default_factory=list)
    probes: list = field(default_factory=list)
    min_tests: int = 3
    split: str = "dev"
    path: str = ""

    def to_dict(self):
        d = {
            "name": self.name,
            "description": self.description,
            "endpoints": [{k: v for k, v in e.__dict__.items() if v} for e in self.endpoints],
            "metrics": [m.__dict__ for m in self.metrics],
            "behaviors": self.behaviors,
        }
        if self.probes:
            d["probes"] = [{k: v for k, v in p.__dict__.items() if v is not None} for p in self.probes]
        return d


_ALLOWED = {"name", "description", "endpoints", "metrics", "behaviors", "probes", "min_tests"}
_METRIC_RE = re.compile(r"^[a-zA-Z_:][a-zA-Z0-9_:]*$")


def _check_keys(obj, allowed, where):
    extra = set(obj) - allowed
    if extra:
        raise SpecError(f"{where}: unknown keys {sorted(extra)}")


def parse_spec(raw: dict, path: str = "", split: str = "dev") -> Spec:
    if not isinstance(raw, dict):
        raise SpecError(f"{path}: spec must be a mapping")
    _check_keys(raw, _ALLOWED, path)
    for key in ("name", "description", "endpoints", "metrics"):
        if key not in raw:
            raise SpecError(f"{path}: missing '{key}'")
    if not re.fullmatch(r"[a-z][a-z0-9-]*", str(raw["name"])):
        raise SpecError(f"{path}: name must be lowercase-kebab-case")

    endpoints = []
    for e in raw["endpoints"]:
        _check_keys(e, {"method", "path", "description"}, f"{path} endpoint")
        if e["method"] not in METHODS or not str(e["path"]).startswith("/"):
            raise SpecError(f"{path}: bad endpoint {e}")
        endpoints.append(Endpoint(e["method"], e["path"], e.get("description", "")))
    keys = {(e.method, e.path) for e in endpoints}
    if ("GET", "/health") not in keys or ("GET", "/metrics") not in keys:
        raise SpecError(f"{path}: spec must declare GET /health and GET /metrics")

    metrics = []
    for m in raw["metrics"]:
        _check_keys(m, {"name", "type", "help"}, f"{path} metric")
        if m["type"] not in METRIC_TYPES or not _METRIC_RE.match(m["name"]):
            raise SpecError(f"{path}: bad metric {m}")
        metrics.append(Metric(m["name"], m["type"], m["help"]))

    probes = []
    for p in raw.get("probes", []):
        _check_keys(p, {"method", "path", "body", "expect_status"}, f"{path} probe")
        exp = p["expect_status"]
        exp = exp if isinstance(exp, list) else [exp]
        probes.append(Probe(p["method"], p["path"], exp, p.get("body")))

    return Spec(
        name=raw["name"], description=raw["description"], endpoints=endpoints,
        metrics=metrics, behaviors=list(raw.get("behaviors", [])), probes=probes,
        min_tests=int(raw.get("min_tests", 3)), split=split, path=path,
    )


def load_spec(path) -> Spec:
    p = Path(path)
    with open(p) as f:
        raw = yaml.safe_load(f)
    split = p.parent.name if p.parent.name in ("dev", "heldout") else "dev"
    return parse_spec(raw, str(p), split)


def list_specs(root="specs", split="all"):
    splits = ["dev", "heldout"] if split == "all" else [split]
    out = []
    for s in splits:
        out += [load_spec(f) for f in sorted(Path(root, s).glob("*.yaml"))]
    return out
