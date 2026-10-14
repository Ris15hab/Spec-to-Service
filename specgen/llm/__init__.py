"""Build an LLM client from a models.yaml entry."""
from pathlib import Path

import yaml

from specgen.llm.base import CachedClient, LLMError


def load_models(path="models.yaml"):
    return yaml.safe_load(Path(path).read_text())


def get_client(name, models_path="models.yaml", cache_dir=None, verify=True):
    cfg = load_models(models_path).get(name)
    if cfg is None:
        raise LLMError(f"unknown model '{name}' (see models.yaml)")
    if cfg["provider"] == "mock":
        from specgen.llm.mock import MockClient
        client = MockClient(name, cfg.get("behavior", ["good"]))
    else:
        from specgen.llm.openai_compat import OpenAICompatClient
        client = OpenAICompatClient(name, **cfg)
        if verify:
            client.verify_model()
    return CachedClient(client, cache_dir) if cache_dir else client
