"""Claude Code CLI upstream: runs the tutor roles through headless `claude -p`,
authenticated by the user's Claude Code login (subscription billing, no API
key on the box). Same interface as the other chains.

No token-level schema enforcement here - the JSON contract is prompt-enforced,
extracted, checked against the schema's required keys, and retried once.
"""

from __future__ import annotations

import json
import shutil
import subprocess

from llm import UpstreamError

ROLE_TIMEOUTS = {"grader": 300, "planner": 300, "author": 900}

# Per-role model tiers. Grading is a bounded judgement against an explicit
# rubric and sonnet scores 5/5 on the sycophancy red-team at roughly a sixth of
# opus's latency, which is what makes a nine-item check tolerable. Authoring is
# open-ended prose the learner reads for 25 minutes, so it keeps opus.
ROLE_MODELS = {"grader": "sonnet", "planner": "sonnet", "author": "opus"}


class ClaudeCLIChain:
    def __init__(self, model: str | None = None):
        # An explicit model overrides the per-role tiers entirely.
        self.model = model
        self.role_models = ROLE_MODELS

    def _available(self) -> bool:
        # Presence of the binary, not of credentials: the login lives in a
        # file on Linux but in the Keychain on macOS, so probing storage
        # reports "not logged in" on a machine that works fine. An actual
        # auth failure surfaces through the CLI's own stderr, which says more
        # than a guess would.
        return shutil.which("claude") is not None

    def healthy_upstream(self):
        return {"name": "claude-cli", "model": self.model} if self._available() else None

    def model_for(self, role: str) -> str:
        return self.model or self.role_models.get(role, "opus")

    def status(self) -> dict:
        ok = self._available()
        return {"connected": ok, "upstream": "claude-cli",
                "model": self.model or "per-role: " + ", ".join(
                    f"{r}={m}" for r, m in self.role_models.items()),
                "error": None if ok else "`claude` not on PATH"}

    def command(self, role: str, prompt: str, system: str) -> list[str]:
        return ["claude", "-p", prompt,
                "--append-system-prompt", system,
                "--output-format", "json",
                # These roles are pure text generation. Leaving the tools
                # enabled let the model spend its one turn on a tool call,
                # which exits non-zero as error_max_turns and throws the work
                # away; it also prepends ~17k tokens of tool schemas to every
                # call, which is most of what made grading slow.
                "--tools", "",
                "--max-turns", "1",
                "--model", self.model_for(role)]

    def structured(self, role: str, system: str, user: str, schema: dict,
                   schema_name: str = "result") -> dict:
        if not self._available():
            raise UpstreamError("claude-cli: `claude` not on PATH")
        prompt = (f"{user}\n\n"
                  f"Respond with ONLY a single JSON object matching this JSON Schema - "
                  f"no prose, no code fences, no tool use:\n"
                  f"{json.dumps(schema)}")
        last_err = "unknown"
        for attempt in range(2):
            try:
                proc = subprocess.run(
                    self.command(role, prompt, system),
                    capture_output=True, text=True,
                    timeout=ROLE_TIMEOUTS.get(role, 240),
                )
            except subprocess.TimeoutExpired as e:
                raise UpstreamError(f"claude-cli: timed out ({role})") from e
            if proc.returncode != 0:
                # With --output-format json the CLI reports the cause on
                # stdout, so stderr alone leaves a bare "exit 1".
                detail = (proc.stderr.strip() or proc.stdout.strip())[-400:]
                last_err = detail or f"exit {proc.returncode}"
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
