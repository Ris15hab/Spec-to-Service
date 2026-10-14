"""LLM client interface shared by every provider.

Why: the rest of the tool only knows `client.complete(system, user)`, so a
hosted API, a mock, or a self-hosted vLLM endpoint are interchangeable.
Deliberately does not: stream, call tools, or hold conversation state. Each
call is a fresh prompt (the repair loop re-sends what it needs).
"""
import hashlib
import json
from dataclasses import dataclass, field
from pathlib import Path


class LLMError(RuntimeError):
    """Provider failed after retries (maps to the `llm_error` failure category)."""


@dataclass
class Completion:
    text: str
    prompt_tokens: int = 0
    completion_tokens: int = 0
    latency_s: float = 0.0
    cached: bool = False


@dataclass
class LLMClient:
    name: str = "base"
    is_mock: bool = False
    calls: list = field(default_factory=list)

    def complete(self, system: str, user: str, temperature: float = 0.2) -> Completion:
        raise NotImplementedError


class CachedClient(LLMClient):
    """Dev-only response cache keyed by hash(model, prompt). OFF for official runs."""

    def __init__(self, inner: LLMClient, cache_dir):
        super().__init__(name=inner.name, is_mock=inner.is_mock)
        self.inner, self.dir = inner, Path(cache_dir)
        self.dir.mkdir(parents=True, exist_ok=True)

    def complete(self, system, user, temperature=0.2):
        key = hashlib.sha256(json.dumps([self.name, system, user, temperature]).encode()).hexdigest()
        f = self.dir / f"{key}.json"
        if f.exists():
            c = Completion(**json.loads(f.read_text()))
            c.cached = True
            return c
        c = self.inner.complete(system, user, temperature)
        f.write_text(json.dumps(c.__dict__))
        return c
