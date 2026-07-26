"""Generate a new subject corpus from a one-paragraph brief.

This automates what was done by hand for "How AI Works": a syllabus with a
prerequisite graph, a misconception bank, then one authored unit per syllabus
entry, then a lint pass. The authoring contract is `corpus/authoring-spec.md`,
which is subject-agnostic on purpose.

Stages are separate commands so a failure costs one stage, not the whole run,
and so the expensive per-unit stage can be resumed:

    python generate_subject.py plan   <slug> "<brief>"   # syllabus + bank
    python generate_subject.py units  <slug> [u1 u2 ...] # author units
    python generate_subject.py verify <slug>             # lint

Output lands in corpus-<slug>/ and is registered by adding it to config.yaml.
"""

from __future__ import annotations

import concurrent.futures
import sys
from pathlib import Path

import yaml

from llm_anthropic import make_chain

HERE = Path(__file__).parent
# Authoring wants the strongest engine available, which is rarely the one the
# flight config points at; CHIRON_PROVIDER overrides it (see make_chain).
CFG = yaml.safe_load((HERE / "config.yaml").read_text())
SPEC = (HERE.parent / "corpus/authoring-spec.md")

PLAN_SCHEMA = {
    "type": "object",
    "properties": {
        "title": {"type": "string"},
        "learner_profile": {"type": "string",
                            "description": "who this is for, including what they already know"},
        "units": {
            "type": "array",
            "items": {
                "type": "object",
                "properties": {
                    "id": {"type": "string", "description": "u0, u1, ..."},
                    "slug": {"type": "string"},
                    "title": {"type": "string"},
                    "minutes": {"type": "integer"},
                    "prereqs": {"type": "array", "items": {"type": "string"}},
                    "concepts": {
                        "type": "array",
                        "items": {
                            "type": "object",
                            "properties": {"id": {"type": "string"}, "name": {"type": "string"}},
                            "required": ["id", "name"],
                        },
                    },
                    "notes": {"type": "string"},
                },
                "required": ["id", "slug", "title", "minutes", "prereqs", "concepts", "notes"],
            },
        },
        "misconceptions": {
            "type": "array",
            "items": {
                "type": "object",
                "properties": {
                    "id": {"type": "string", "description": "M1, M2, ..."},
                    "name": {"type": "string", "description": "short kebab-case handle"},
                    "wrong_model": {"type": "string"},
                    "why_appealing": {"type": "string"},
                    "failing_prediction": {
                        "type": "string",
                        "description": "a specific prediction the wrong model makes that observably fails"},
                    "correction": {"type": "string"},
                    "units": {"type": "array", "items": {"type": "string"}},
                },
                "required": ["id", "name", "wrong_model", "why_appealing",
                             "failing_prediction", "correction", "units"],
            },
        },
    },
    "required": ["title", "learner_profile", "units", "misconceptions"],
}

PLAN_SYSTEM = """You design curricula that will be taught by an adaptive textbook.

Two hard constraints shape every choice:

1. **The spine must be completable.** Progress is gated on ~80% mastery per
unit, so an overstuffed syllabus does not run long - it stalls. Budget the
whole spine at roughly 60% of the learner's stated time, leaving room for
remediation and breaks. Order units so every prerequisite precedes its
dependents; the graph is what decides what the learner may read next.

2. **Misconceptions are the teaching material, not a footnote.** For this
domain, list the wrong models a capable learner actually arrives with -
especially ones that are confidently held and partially correct. Each needs a
*specific prediction it makes that observably fails*; that failing prediction
is what the chapter will use to break it. Vague "some people think X is hard"
entries are useless. Aim for at least one per unit, more where the domain is
counterintuitive.

Concept ids are kebab-case and prefixed `c-`. Unit ids are u0, u1, ... in
teaching order. Misconception ids are M1, M2, ... Write for the specific
learner described in the brief: name what they already know so the chapters do
not re-explain it."""


def _plan_path(slug: str) -> Path:
    return HERE.parent / f"corpus-{slug}"


def plan(slug: str, brief: str) -> int:
    chain = make_chain(CFG)
    out_dir = _plan_path(slug)
    user = (f"BRIEF:\n{brief}\n\n"
            f"Design the syllabus and misconception bank. The authoring contract "
            f"the units will follow is below, for context on what each unit must "
            f"eventually contain.\n\n{SPEC.read_text()[:6000]}")
    print(f"planning '{slug}' ...")
    result = chain.structured("planner", PLAN_SYSTEM, user, PLAN_SCHEMA, "plan")

    out_dir.mkdir(parents=True, exist_ok=True)
    syllabus = {
        "title": result["title"],
        "learner": {"profile": result["learner_profile"]},
        "defaults": {"mastery_gate": 0.80, "extension_trigger": 0.90,
                     "check_min_items": 8, "check_constructed_fraction": 0.6,
                     "check_callback_fraction": 0.4, "chunk_minutes": [20, 25]},
        "units": result["units"],
    }
    (out_dir / "syllabus.yaml").write_text(yaml.safe_dump(syllabus, sort_keys=False))
    (out_dir / "misconception-bank.yaml").write_text(
        yaml.safe_dump({"misconceptions": result["misconceptions"]}, sort_keys=False))

    total = sum(u["minutes"] for u in result["units"])
    print(f"  {len(result['units'])} units, {total} min spine, "
          f"{len(result['misconceptions'])} misconceptions")
    print(f"  wrote {out_dir}/syllabus.yaml and misconception-bank.yaml")
    print(f"  next: python generate_subject.py units {slug}")
    return 0


UNIT_SYSTEM = """You author one unit of an adaptive textbook corpus, following the
authoring contract exactly. The contract is not advice - the server parses these
files mechanically, and a deviation breaks the unit.

The two things that most often go wrong, and that you must get right:

- **Mechanically-checked items need machine-comparable answers.** If `check` is
`numeric(tol)` or `exact`, `answer` is a bare value, never a sentence. Choose
`tol` tighter than the distance to every plausible wrong answer you name in the
rubric - a tolerance that accepts the error the item exists to catch makes the
item worthless.
- **Every MCQ distractor cites a misconception id from the bank**, and every
option carries an `explain`. A wrong answer must be a diagnosis, not a miss."""


def author_unit(slug: str, unit: dict, syllabus: dict, bank: str) -> tuple[str, bool, str]:
    """Author one unit. Returns (unit_id, ok, message)."""
    chain = make_chain(CFG)
    out_dir = _plan_path(slug) / "units" / f"{unit['id']}-{unit['slug']}"
    schema = {
        "type": "object",
        "properties": {
            "canon_md": {"type": "string", "description": "full canon.md including front matter and ```beat blocks"},
            "questions_yaml": {"type": "string", "description": "full questions.yaml content"},
            "misconceptions_yaml": {"type": "string", "description": "unit-local misconceptions.yaml content"},
            "depth_deeper_math": {"type": "string"},
            "depth_more_intuition": {"type": "string"},
            "depth_se_analogies": {"type": "string"},
        },
        "required": ["canon_md", "questions_yaml", "misconceptions_yaml",
                     "depth_deeper_math", "depth_more_intuition", "depth_se_analogies"],
    }
    user = (f"AUTHORING CONTRACT:\n{SPEC.read_text()}\n\n"
            f"LEARNER:\n{syllabus['learner']['profile']}\n\n"
            f"MISCONCEPTION BANK (cite these ids):\n{bank}\n\n"
            f"UNIT TO AUTHOR:\n{yaml.safe_dump(unit, sort_keys=False)}\n\n"
            f"Author every file for this unit. Depth files must use the exact same "
            f"`## ` headings as canon.md.")
    try:
        result = chain.structured("author", UNIT_SYSTEM, user, schema, "unit")
    except Exception as e:                      # noqa: BLE001 - report, don't abort the batch
        return unit["id"], False, str(e)[:200]

    out_dir.mkdir(parents=True, exist_ok=True)
    (out_dir / "canon.md").write_text(result["canon_md"])
    (out_dir / "questions.yaml").write_text(result["questions_yaml"])
    (out_dir / "misconceptions.yaml").write_text(result["misconceptions_yaml"])
    depths = out_dir / "depths"
    depths.mkdir(exist_ok=True)
    (depths / "deeper-math.md").write_text(result["depth_deeper_math"])
    (depths / "more-intuition.md").write_text(result["depth_more_intuition"])
    (depths / "se-analogies.md").write_text(result["depth_se_analogies"])
    return unit["id"], True, f"{len(result['canon_md'])} chars"


def units(slug: str, only: list[str]) -> int:
    root = _plan_path(slug)
    syllabus = yaml.safe_load((root / "syllabus.yaml").read_text())
    bank = (root / "misconception-bank.yaml").read_text()
    todo = [u for u in syllabus["units"] if not only or u["id"] in only]
    print(f"authoring {len(todo)} unit(s) of '{slug}' ...")
    failures = 0
    # Units are independent; the wall clock here is dominated by generation.
    with concurrent.futures.ThreadPoolExecutor(max_workers=4) as pool:
        futures = [pool.submit(author_unit, slug, u, syllabus, bank) for u in todo]
        for f in concurrent.futures.as_completed(futures):
            uid, ok, msg = f.result()
            print(f"  {'ok  ' if ok else 'FAIL'} {uid}: {msg}")
            failures += not ok
    print(f"\n{len(todo) - failures}/{len(todo)} units authored")
    print(f"  next: python generate_subject.py verify {slug}")
    return failures


def verify(slug: str) -> int:
    import corpus_lint
    sys.argv = ["corpus_lint", str(_plan_path(slug))]
    return corpus_lint.main()


def main() -> int:
    if len(sys.argv) < 3:
        print(__doc__)
        return 2
    cmd, slug = sys.argv[1], sys.argv[2]
    if cmd == "plan":
        if len(sys.argv) < 4:
            print("need a brief")
            return 2
        return plan(slug, sys.argv[3])
    if cmd == "units":
        return units(slug, sys.argv[3:])
    if cmd == "verify":
        return verify(slug)
    print(f"unknown command {cmd!r}")
    return 2


if __name__ == "__main__":
    sys.exit(main())
