"""Claude Code CLI upstream: runs the tutor roles through headless `claude -p`,
authenticated by the user's Claude Code login (subscription billing, no API
key on the box). Same interface as the other chains.

No token-level schema enforcement here - the JSON contract is prompt-enforced,
extracted, checked against the schema's required keys, and retried once.
"""

from __future__ import annotations

import json
import os
import subprocess

from llm import UpstreamError

ROLE_TIMEOUTS = {"grader": 180, "planner": 240, "author": 900}
CRED_PATH = os.path.expanduser("~/.claude/.credentials.json")


class ClaudeCLIChain:
    def __init__(self, model: str = "opus"):
        self.model = model

    def _logged_in(self) -> bool:
        return os.path.exists(CRED_PATH)

    def healthy_upstream(self):
        return {"name": "claude-cli", "model": self.model} if self._logged_in() else None

    def status(self) -> dict:
        ok = self._logged_in()
        return {"connected": ok, "upstream": "claude-cli", "model": self.model,
                "error": None if ok else "claude CLI not logged in"}

    def structured(self, role: str, system: str, user: str, schema: dict,
                   schema_name: str = "result") -> dict:
        if not self._logged_in():
            raise UpstreamError("claude-cli: not logged in")
        prompt = (f"{user}\n\n"
                  f"Respond with ONLY a single JSON object matching this JSON Schema - "
                  f"no prose, no code fences, no tool use:\n"
                  f"{json.dumps(schema)}")
        last_err = "unknown"
        for attempt in range(2):
            try:
                proc = subprocess.run(
                    ["claude", "-p", prompt,
                     "--append-system-prompt", system,
                     "--output-format", "json",
                     "--max-turns", "1",
                     "--model", self.model],
                    capture_output=True, text=True,
                    timeout=ROLE_TIMEOUTS.get(role, 240),
                )
            except subprocess.TimeoutExpired as e:
                raise UpstreamError(f"claude-cli: timed out ({role})") from e
            if proc.returncode != 0:
                last_err = proc.stderr.strip()[-300:] or f"exit {proc.returncode}"
                continue
            try:
                envelope = json.loads(proc.stdout)
                text = envelope.get("result", "")
                start, end = text.find("{"), text.rfind("}")
                out = json.loads(text[start:end + 1])
            except (json.JSONDecodeError, ValueError) as e:
                last_err = f"unparseable output: {e}"
                prompt += "\n\nYour previous reply was not a valid bare JSON object. Return ONLY the JSON object."
                continue
            missing = [k for k in schema.get("required", []) if k not in out]
            if missing:
                last_err = f"missing keys: {missing}"
                prompt += f"\n\nYour previous reply was missing required keys {missing}. Return the complete JSON object."
                continue
            return out
        raise UpstreamError(f"claude-cli: {last_err}")
