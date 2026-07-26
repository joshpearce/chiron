"""Upstream LLM chain: ordered OpenAI-compatible endpoints, first healthy wins.

Every call is structured output (response_format json_schema), bounded by a
hard max_tokens and client timeout - the MoE model has a documented
infinite-loop pathology and a mid-flight hang is unrecoverable.
"""

from __future__ import annotations

import json
import time

import httpx

HEALTH_TIMEOUT = 3.0


class UpstreamError(RuntimeError):
    pass


class LLMChain:
    def __init__(self, upstreams: list, llm_cfg: dict):
        self.upstreams = [u for u in upstreams if u.get("enabled")]
        self.cfg = llm_cfg
        self._healthy_cache: tuple[float, dict | None] = (0.0, None)

    def healthy_upstream(self) -> dict | None:
        ts, cached = self._healthy_cache
        if time.time() - ts < 30 and cached is not None:
            return cached
        for up in self.upstreams:
            try:
                r = httpx.get(f"{up['base_url']}/models", timeout=HEALTH_TIMEOUT)
                if r.status_code == 200:
                    self._healthy_cache = (time.time(), up)
                    return up
            except httpx.HTTPError:
                continue
        self._healthy_cache = (time.time(), None)
        return None

    def status(self) -> dict:
        up = self.healthy_upstream()
        return {"connected": up is not None, "upstream": up["name"] if up else None}

    def structured(self, role: str, system: str, user: str, schema: dict,
                   schema_name: str = "result") -> dict:
        """Blocking structured-output call. Raises UpstreamError if no upstream
        is reachable or output doesn't parse - callers decide the fallback."""
        up = self.healthy_upstream()
        if up is None:
            raise UpstreamError("no upstream reachable")
        body = {
            "model": up["model"],
            "messages": [{"role": "system", "content": system},
                         {"role": "user", "content": user}],
            "temperature": self.cfg.get("temperature", 0.7),
            "max_tokens": self.cfg["max_tokens"][role],
            "response_format": {
                "type": "json_schema",
                "json_schema": {"name": schema_name, "strict": "true", "schema": schema},
            },
        }
        try:
            r = httpx.post(f"{up['base_url']}/chat/completions", json=body,
                           timeout=self.cfg.get("timeout_s", 240))
            r.raise_for_status()
        except httpx.HTTPError as e:
            self._healthy_cache = (0.0, None)  # force re-probe next call
            raise UpstreamError(f"{up['name']}: {e}") from e
        msg = r.json()["choices"][0]["message"]
        # Thinking models (Qwen3.x) may land the JSON in reasoning_content
        # with content empty; take whichever field yields valid JSON.
        for field in ("content", "reasoning_content"):
            text = (msg.get(field) or "").strip()
            if not text:
                continue
            try:
                return json.loads(text)
            except json.JSONDecodeError:
                start, end = text.find("{"), text.rfind("}")
                if 0 <= start < end:
                    try:
                        return json.loads(text[start:end + 1])
                    except json.JSONDecodeError:
                        continue
        raise UpstreamError(f"{up['name']}: unparseable structured output")
