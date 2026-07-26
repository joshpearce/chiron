"""Anthropic upstream: Claude via the official SDK with native structured
outputs. Used when the server runs in a sprite (Chiron v2) or on the Mac with
internet. Implements the same interface as llm.LLMChain so roles.py doesn't
care which engine is behind it.

Per the Claude API guidance: official SDK (no OpenAI-compat shim),
model claude-opus-5, structured output via output_config.format json_schema
(requires additionalProperties: false on every object).
"""

from __future__ import annotations

import json

from llm import UpstreamError

MODEL = "claude-opus-5"

# max_tokens caps thinking + response text together on claude-opus-5, so these
# are far above the JSON payload sizes themselves.
ROLE_LIMITS = {
    "grader": {"max_tokens": 4000, "effort": "medium"},
    "planner": {"max_tokens": 6000, "effort": "medium"},
    "author": {"max_tokens": 16000, "effort": "high"},
}


def _strictify(schema: dict) -> dict:
    """Anthropic json_schema format requires additionalProperties: false on
    every object; walk the schema and add it."""
    if isinstance(schema, dict):
        out = {k: _strictify(v) for k, v in schema.items()}
        if out.get("type") == "object":
            out.setdefault("additionalProperties", False)
            out.setdefault("required", list(out.get("properties", {}).keys()))
        return out
    if isinstance(schema, list):
        return [_strictify(x) for x in schema]
    return schema


class AnthropicChain:
    def __init__(self, model: str = MODEL):
        import glob
        import os

        self.model = model
        self.client = None
        self._client_error = None
        has_env = bool(os.environ.get("ANTHROPIC_API_KEY") or os.environ.get("ANTHROPIC_AUTH_TOKEN"))
        # `ant auth login` OAuth profiles are a first-class SDK credential
        # source; detect them so health reporting is honest.
        cfg_dir = os.environ.get("ANTHROPIC_CONFIG_DIR", os.path.expanduser("~/.config/anthropic"))
        has_profile = bool(glob.glob(os.path.join(cfg_dir, "credentials", "*.json")))
        if not (has_env or has_profile):
            self._client_error = ("no credentials: set ANTHROPIC_API_KEY, or run "
                                  "`ant auth login --no-browser` for an OAuth profile")
            return
        try:
            import anthropic  # deferred so the Mac offline path never needs it

            self.client = anthropic.Anthropic()
        except Exception as e:  # must not crash server boot
            self._client_error = str(e)

    def healthy_upstream(self):
        return {"name": "anthropic", "model": self.model} if self.client else None

    def status(self) -> dict:
        return {"connected": self.client is not None, "upstream": "anthropic",
                "model": self.model, "error": self._client_error}

    def structured(self, role: str, system: str, user: str, schema: dict,
                   schema_name: str = "result") -> dict:
        import anthropic

        if self.client is None:
            raise UpstreamError(f"anthropic: no credentials ({self._client_error})")
        limits = ROLE_LIMITS.get(role, ROLE_LIMITS["planner"])
        try:
            response = self.client.messages.create(
                model=self.model,
                max_tokens=limits["max_tokens"],
                system=system,
                output_config={
                    "effort": limits["effort"],
                    "format": {"type": "json_schema", "schema": _strictify(schema)},
                },
                messages=[{"role": "user", "content": user}],
            )
        except anthropic.APIStatusError as e:
            raise UpstreamError(f"anthropic: {e.status_code} {e.message}") from e
        except anthropic.APIConnectionError as e:
            raise UpstreamError(f"anthropic: connection failed") from e

        if response.stop_reason == "refusal":
            raise UpstreamError("anthropic: request refused by safety classifiers")
        text = next((b.text for b in response.content if b.type == "text"), "")
        try:
            return json.loads(text)
        except json.JSONDecodeError as e:
            raise UpstreamError("anthropic: unparseable structured output") from e


def make_chain(cfg: dict):
    """Chain factory: 'anthropic' -> Claude SDK; 'claude-cli' -> headless
    Claude Code (subscription auth); anything else -> LM Studio."""
    from llm import LLMChain

    provider = cfg.get("provider")
    if provider == "anthropic":
        return AnthropicChain(cfg.get("anthropic_model", MODEL))
    if provider == "claude-cli":
        from llm_claude_cli import ClaudeCLIChain

        # No model configured means per-role tiers (see llm_claude_cli).
        return ClaudeCLIChain(cfg.get("claude_cli_model"))
    return LLMChain(cfg["upstreams"], cfg["llm"])
