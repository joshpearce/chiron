"""Mechanical corpus checks - the automatable half of the verify pass.

Three separate agents audited the AI corpus by hand and found ~50 defects. The
judgement-heavy findings (is this claim about transformers true? does this
rubric actually adjudicate?) still need a reader. Everything else was
mechanical, recurring, and exactly the kind of thing that silently regresses:
answers that are prose where a program must parse them, tolerances loose enough
to accept the very error the item exists to catch, distractors citing
misconception ids that do not exist, depth files whose headings drifted from
canon so the section swap silently does nothing.

Run:  .venv/bin/python corpus_lint.py [corpus_dir]
Exit code is the number of errors (warnings do not fail).
"""

from __future__ import annotations

import math
import re
import sys
from pathlib import Path

import yaml

from corpus import Corpus

NUMERIC = re.compile(r"numeric\(([\d.eE+-]+)\)")
REFUTES = re.compile(r"<!--\s*refutes:\s*([A-Za-z0-9_-]+)\s*-->")


class Report:
    def __init__(self) -> None:
        self.errors: list[str] = []
        self.warnings: list[str] = []

    def error(self, where: str, msg: str) -> None:
        self.errors.append(f"{where}: {msg}")

    def warn(self, where: str, msg: str) -> None:
        self.warnings.append(f"{where}: {msg}")


def _is_machine_answer(value) -> bool:
    """A mechanically-checked item must answer with something a program can
    compare, not a sentence explaining the answer."""
    if isinstance(value, (int, float)):
        return True
    if not isinstance(value, str):
        return False
    text = value.strip()
    if not text or len(text.split()) > 8:
        return False
    # Vectors, shapes, hex bytes, and tokenizations ("[7, -4]", "4 x 7",
    # "0xB2,0x00,0x2F", "w|i|d|est_") all compare fine as exact strings. The
    # ">8 words" guard above is what actually keeps prose out; these classes
    # only have to be wide enough not to reject a real answer.
    return bool(re.fullmatch(r"[-+0-9A-Fa-f.eE/×x,;\[\]() ]+|[A-Za-z0-9 _.\-/|]{1,24}", text))


def _plausible_wrong_answers(expected: float) -> list[float]:
    """Error paths a learner actually takes: a dropped or doubled factor, a
    sign flip, an order-of-magnitude slip. Deliberately excludes off-by-one -
    a tolerance reaching expected+-1 is usually the author's intent, not a
    misconception the item exists to catch."""
    out = [expected * 2, expected / 2, -expected]
    if expected != 0:
        out += [expected * 10, expected / 10]
    return [v for v in out if v != expected and not math.isnan(v)]


def check_numeric_tolerances(unit, report: Report) -> None:
    """The server compares with max(tol, |expected|*tol), so tolerance widens
    relatively on large answers. A tolerance that reaches a plausible wrong
    answer silently marks the target error correct."""
    items = list(unit.beats()) + list(unit.questions.get("check", []) or []) \
        + list(unit.questions.get("pretest", []) or [])
    for item in items:
        check = str(item.get("check", "llm"))
        m = NUMERIC.fullmatch(check.strip())
        if not m:
            continue
        where = f"{unit.id}/{item.get('id', '?')}"
        try:
            expected = float(str(item.get("answer")).strip())
        except (TypeError, ValueError):
            report.error(where, f"check is {check} but answer is not numeric: "
                                f"{item.get('answer')!r}")
            continue
        tol = float(m.group(1))
        window = tol + abs(expected) * 1e-9   # must mirror checkers.check_answer
        for wrong in _plausible_wrong_answers(expected):
            if abs(wrong - expected) <= window:
                report.error(where,
                             f"tolerance {check} accepts {wrong:g} as correct "
                             f"(expected {expected:g}, window +/-{window:g})")
                break
        if abs(expected) >= 1e6 and check != "exact":
            report.warn(where, f"large answer {expected:g} with numeric tolerance - "
                               "prefer check: exact for big integers")


def check_answers_machine_readable(unit, report: Report) -> None:
    items = list(unit.beats()) + list(unit.questions.get("check", []) or []) \
        + list(unit.questions.get("pretest", []) or [])
    for item in items:
        check = str(item.get("check", "llm"))
        if check == "llm" or check == "choice":
            continue
        where = f"{unit.id}/{item.get('id', '?')}"
        if not _is_machine_answer(item.get("answer")):
            report.error(where, f"check is {check} but answer is prose: "
                                f"{str(item.get('answer'))[:60]!r}")


def check_mcqs(unit, corpus, report: Report) -> None:
    for pool in ("pretest", "check"):
        for q in unit.questions.get(pool, []) or []:
            if q.get("kind") != "mcq":
                continue
            where = f"{unit.id}/{q.get('id', '?')}"
            options = q.get("options") or []
            correct = [o for o in options if o.get("correct")]
            if len(correct) != 1:
                report.error(where, f"{len(correct)} options marked correct, expected exactly 1")
            if q.get("check") != "choice":
                report.error(where, f"mcq must set check: choice (found {q.get('check')!r})")
            for o in options:
                if not o.get("explain"):
                    report.error(where, "an option has no explain - feedback must "
                                        "adjudicate every option, not just the right one")
                if not o.get("correct"):
                    mid = o.get("misconception")
                    if not mid:
                        report.error(where, "distractor has no misconception id - "
                                            "a wrong answer must be a diagnosis")
                    elif mid not in corpus.misconceptions:
                        report.error(where, f"distractor cites unknown misconception {mid!r}")


def check_spec_conformance(unit, report: Report) -> None:
    checks = unit.questions.get("check", []) or []
    where = unit.id
    if len(checks) < 8:
        report.error(where, f"{len(checks)} check items, spec requires >= 8 "
                            "(fewer is statistical noise for an 80% gate)")
    constructed = [q for q in checks if q.get("kind") == "constructed"]
    if checks and len(constructed) / len(checks) < 0.6:
        report.error(where, f"{len(constructed)}/{len(checks)} constructed "
                            "(spec requires >= 60% for response congruency)")
    if sum(1 for q in checks if q.get("callback_eligible")) < 2:
        report.warn(where, "fewer than 2 callback_eligible items - later units "
                           "have little to draw on for cumulative checks")
    beats = list(unit.beats())
    if len(beats) < 6:
        report.warn(where, f"{len(beats)} interaction beats, spec asks for 6-10 "
                           "(step-based interaction is where the tutoring effect lives)")
    for b in beats:
        if b.get("type") == "self-explain" and not b.get("rubric"):
            report.error(f"{unit.id}/{b.get('id','?')}",
                         "self-explain beat has no rubric - an unadjudicated "
                         "self-explanation can entrench a wrong model")


def check_depth_headings(unit, report: Report) -> None:
    canon = {s.heading for s in unit.sections}
    for name, sections in unit.depths.items():
        drifted = set(sections) - canon
        if drifted:
            report.error(f"{unit.id}/depths/{name}",
                         f"headings not in canon (swap silently does nothing): "
                         f"{sorted(drifted)[:3]}")
        # The same silent no-op from the other direction: the planner can ask
        # for a section at this depth and get the canon text back, with nothing
        # anywhere saying the variant was never written.
        missing = canon - set(sections)
        if missing:
            report.warn(f"{unit.id}/depths/{name}",
                        f"{len(missing)} canon section(s) have no variant here, so "
                        f"requesting this depth silently returns canon: "
                        f"{sorted(missing)[:3]}")


def check_refutation_ids(unit, corpus, report: Report) -> None:
    text = "\n".join(s.markdown() for s in unit.sections)
    for mid in REFUTES.findall(text):
        if mid not in corpus.misconceptions:
            report.error(unit.id, f"<!-- refutes: {mid} --> does not resolve to a "
                                  "misconception in the bank")


def main() -> int:
    root = Path(sys.argv[1]) if len(sys.argv) > 1 else Path(__file__).parent.parent / "corpus"
    corpus = Corpus(root)
    report = Report()

    for uid in corpus.unit_order():
        unit = corpus.units.get(uid)
        if unit is None:
            report.warn(uid, "declared in syllabus but not authored")
            continue
        check_numeric_tolerances(unit, report)
        check_answers_machine_readable(unit, report)
        check_mcqs(unit, corpus, report)
        check_spec_conformance(unit, report)
        check_depth_headings(unit, report)
        check_refutation_ids(unit, corpus, report)

    print(f"linted {len(corpus.units)} units, {len(corpus.misconceptions)} misconceptions")
    for w in report.warnings:
        print(f"  warn  {w}")
    for e in report.errors:
        print(f"  ERROR {e}")
    print(f"\n{len(report.errors)} errors, {len(report.warnings)} warnings")
    return len(report.errors)


if __name__ == "__main__":
    sys.exit(main())
