"""Contract for the headless-Claude upstream's command line.

Offline: asserts on the argv, makes no model calls.

The case that matters is `--tools ""`. With the built-in tools enabled, the
model can spend its single allowed turn on a tool call; the CLI then exits
non-zero with subtype `error_max_turns` and the generated work is discarded.
That failure is intermittent and looks like a bare "exit 1", so it is exactly
the kind of thing that comes back silently.

Run:  .venv/bin/python test_claude_cli.py
"""

from __future__ import annotations

import sys

from llm_claude_cli import ClaudeCLIChain


def pairs(argv: list[str]) -> dict[str, str]:
    return {argv[i]: argv[i + 1] for i in range(len(argv) - 1) if argv[i].startswith("--")}


def main() -> int:
    failures = []

    def check(cond: bool, why: str) -> None:
        if not cond:
            failures.append(why)

    tiers = ClaudeCLIChain()
    argv = tiers.command("grader", "PROMPT", "SYSTEM")
    flags = pairs(argv)

    check("--tools" in flags and flags["--tools"] == "",
          "tools are not disabled - a tool turn will exit error_max_turns and discard the grade")
    check(flags.get("--max-turns") == "1", "max-turns should stay 1 once tools are off")
    check(flags.get("--output-format") == "json", "structured() parses the json envelope")
    check(argv[:2] == ["claude", "-p"], "must invoke the CLI headless")
    check("PROMPT" in argv and flags.get("--append-system-prompt") == "SYSTEM",
          "prompt and system must reach the CLI")

    # Per-role tiers: sonnet grades (fast, 5/5 on the red-team), opus authors.
    check(tiers.command("grader", "p", "s")[-1] == "sonnet", "grader should be sonnet")
    check(tiers.command("planner", "p", "s")[-1] == "sonnet", "planner should be sonnet")
    check(tiers.command("author", "p", "s")[-1] == "opus", "author should be opus")

    pinned = ClaudeCLIChain("haiku")
    check(pinned.command("author", "p", "s")[-1] == "haiku",
          "an explicit model must override the per-role tiers")

    for why in failures:
        print(f"FAIL {why}")
    print(f"{9 - len(failures)}/9 claude-cli command cases pass")
    return 1 if failures else 0


if __name__ == "__main__":
    sys.exit(main())
