"""One client for every OpenAI-compatible endpoint (NVIDIA NIM, Gemini, vLLM, Ollama...).

Why: a single code path means a self-hosted endpoint is just another models.yaml entry.
Deliberately does not: use provider-specific SDKs or features.
"""
import os
import time

import openai

from specgen.llm.base import Completion, LLMClient, LLMError
from specgen.llm.ratelimit import RateLimiter, backoff_delay

MAX_TRIES = 6


class OpenAICompatClient(LLMClient):
    def __init__(self, name, base_url, model_id, api_key_env, rpm=30, **_):
        super().__init__(name=name)
        key = os.environ.get(api_key_env)
        if not key:
            raise LLMError(f"{name}: environment variable {api_key_env} is not set")
        self.model_id = model_id
        self.base_url = base_url
        self.limiter = RateLimiter(rpm)
        self.api = openai.OpenAI(base_url=base_url, api_key=key, max_retries=0, timeout=180)

    def verify_model(self):
        """Fail early, with a useful message, if the model ID is not in the catalog."""
        try:
            ids = [m.id for m in self.api.models.list().data]
        except openai.OpenAIError as e:
            raise LLMError(f"{self.name}: could not list models at {self.base_url}: {e}")
        if not {self.model_id, f"models/{self.model_id}"} & set(ids):
            hint = [i for i in ids if self.model_id.split("/")[-1].split("-")[0] in i][:8]
            raise LLMError(f"{self.name}: model '{self.model_id}' not in catalog. Similar IDs: {hint}")
        err = None
        for attempt in range(5):  # listed != usable (some IDs 404 per account): make one tiny real call
            try:
                self.api.chat.completions.create(model=self.model_id, max_tokens=8,
                                                 messages=[{"role": "user", "content": "Say OK"}])
                return
            except (openai.RateLimitError, openai.InternalServerError, openai.APIConnectionError,
                    openai.APITimeoutError) as e:  # temporary (e.g. 503 overloaded): wait and retry
                err = e
                time.sleep(backoff_delay(attempt))
            except openai.OpenAIError as e:
                raise LLMError(f"{self.name}: model '{self.model_id}' is listed but not callable with this key: {str(e)[:150]}")
        raise LLMError(f"{self.name}: model '{self.model_id}' kept failing the startup ping: {str(err)[:150]}")

    def complete(self, system, user, temperature=0.2):
        last = None
        for attempt in range(MAX_TRIES):
            self.limiter.wait()
            t0 = time.monotonic()
            try:
                r = self.api.chat.completions.create(
                    model=self.model_id, temperature=temperature,
                    messages=[{"role": "system", "content": system}, {"role": "user", "content": user}],
                )
                u = r.usage
                return Completion(
                    text=r.choices[0].message.content or "",
                    prompt_tokens=getattr(u, "prompt_tokens", 0) or 0,
                    completion_tokens=getattr(u, "completion_tokens", 0) or 0,
                    latency_s=time.monotonic() - t0,
                )
            except (openai.RateLimitError, openai.InternalServerError,
                    openai.APIConnectionError, openai.APITimeoutError) as e:
                last = e
                ra = getattr(getattr(e, "response", None), "headers", {}).get("retry-after")
                time.sleep(backoff_delay(attempt, float(ra) if ra and ra.isdigit() else None))
            except openai.OpenAIError as e:  # 4xx other than 429: retrying will not help
                raise LLMError(f"{self.name}: {e}")
        raise LLMError(f"{self.name}: gave up after {MAX_TRIES} tries: {last}")
