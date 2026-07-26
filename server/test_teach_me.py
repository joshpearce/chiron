"""Teach-me elicitation: a simulated learner talks until the tutor has a brief.

The failure mode this guards is an interview that never ends. A learner who
opened a book to read it will not answer twelve questions, and an elicitor with
no stopping condition is happy to ask them - so the test plays a deliberately
unhelpful learner (vague, terse, one detail volunteered late) and requires the
tutor to converge anyway, on a brief the planner can actually use.

Run:  CHIRON_PROVIDER=claude-cli .venv/bin/python test_teach_me.py
"""

from __future__ import annotations

import sys
from pathlib import Path

import yaml

import roles
from llm_anthropic import make_chain

HERE = Path(__file__).parent
CFG = yaml.safe_load((HERE / "config.yaml").read_text())

MAX_TURNS = 8            # hard stop; the tutor is supposed to finish well before this
EXPECT_BY = 7            # turns after which we call it a runaway interview

LEARNER_SYSTEM = """You are role-playing a learner being interviewed by a tutor who \
wants to write you a custom book. Stay in character:

- You are a senior backend engineer. You are busy and answer briefly - one or \
two sentences, never a structured list.
- You are initially vague ("I dunno, the usual"), and you only mention the \
concrete thing you actually care about (you keep getting paged for Postgres \
lock contention and cannot reason about it) if asked something that gets at it.
- You have about four hours on a train on Thursday.
- You know SQL well and have read a Postgres internals blog post or two. You do \
not want an intro-to-databases chapter.
- Never write the brief for the tutor, and never summarize your own needs \
neatly. Answer only what was asked."""

LEARNER_SCHEMA = {
    "type": "object",
    "properties": {"text": {"type": "string"}},
    "required": ["text"],
}


def main() -> int:
    chain = make_chain(CFG)
    messages = [{"role": "learner", "text": "I want to learn about databases I guess."}]
    failures: list[str] = []
    result = None

    for turn in range(1, MAX_TURNS + 1):
        result = roles.elicit_turn(chain, messages)
        reply = result.get("reply_md", "")
        print(f"\n--- turn {turn} ---\nTUTOR: {reply.strip()[:600]}")
        messages.append({"role": "tutor", "text": reply})

        if result.get("done"):
            print(f"\n[done after {turn} tutor turns]")
            if turn > EXPECT_BY:
                failures.append(f"took {turn} turns to converge - a learner will not sit through that")
            break

        convo = "\n\n".join(f"{'TUTOR' if m['role'] == 'tutor' else 'YOU'}: {m['text']}"
                            for m in messages)
        answer = chain.structured("grader", LEARNER_SYSTEM,
                                  f"CONVERSATION:\n\n{convo}\n\nReply in character.",
                                  LEARNER_SCHEMA, "learner")
        print(f"LEARNER: {answer['text'].strip()[:400]}")
        messages.append({"role": "learner", "text": answer["text"]})
    else:
        failures.append(f"never finished in {MAX_TURNS} turns")

    brief = (result or {}).get("brief", "")
    slug = (result or {}).get("slug", "")
    title = (result or {}).get("title", "")
    print(f"\nslug={slug!r}  title={title!r}\nbrief: {brief}\n")

    if (result or {}).get("done"):
        if len(brief) < 200:
            failures.append(f"brief is {len(brief)} chars - too thin to plan a syllabus from")
        if not slug or not title:
            failures.append("finished without a slug or title")
        # The brief is the ONLY thing the planner sees; details learned in the
        # interview and left out of it are lost.
        if "lock" not in brief.lower():
            failures.append("brief dropped the concrete problem the learner came with (lock contention)")
        if "hour" not in brief.lower() and "time" not in brief.lower():
            failures.append("brief dropped the time budget, which the spine is sized against")

    for f in failures:
        print(f"FAIL {f}")
    print("teach-me elicitation OK" if not failures else f"{len(failures)} failure(s)")
    return 1 if failures else 0


if __name__ == "__main__":
    sys.exit(main())
