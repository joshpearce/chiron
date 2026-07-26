"""The three LLM roles: grader, planner, author.

Division of labor: everything deterministic lives in code (mastery updates,
fringe, gating, debt, callback selection). The LLM supplies judgment only:
grading free text against rubrics, choosing depth/representation directives,
rewriting flagged sections, and the prose learner summary.
"""

from __future__ import annotations

import random

from llm import LLMChain, UpstreamError

# ---------------------------------------------------------------- grader

GRADE_SCHEMA = {
    "type": "object",
    "properties": {
        "verdict": {"type": "string",
                    "enum": ["pass", "partial", "fail", "valid_alternative_path"]},
        "misconceptions": {"type": "array", "items": {"type": "string"}},
        "evidence": {"type": "string",
                     "description": "shortest quote from the learner answer that justifies the verdict"},
        "feedback_md": {"type": "string",
                        "description": "2-5 sentences: why the correct answer is correct, why THIS answer earned THIS verdict, and what mental model produces the error if any"},
        "next_action": {"type": "string",
                        "description": "one concrete instruction for the learner right now"},
    },
    "required": ["verdict", "misconceptions", "evidence", "feedback_md", "next_action"],
}

GRADER_SYSTEM = """You grade one answer against one rubric for a technical textbook on how LLMs work.

Rules, in priority order:
1. Grade against the rubric ONLY. The rubric's pass criteria are the contract.
2. Fluent, confident, jargon-rich answers that miss the rubric criteria FAIL. \
Well-written wrongness is the failure mode you exist to catch. Do not give \
benefit of the doubt to articulate answers.
3. An answer that reaches a correct result by a legitimate route the rubric \
didn't anticipate is `valid_alternative_path`, not `fail`. Punishing valid \
unusual reasoning is as wrong as accepting invalid reasoning.
4. `partial` means some rubric criteria are genuinely met, not "close in spirit".
5. If the answer exhibits a listed misconception, cite its ID. Only cite IDs \
from the provided list.
6. The learner may sound tentative or cite authorities ("I read that..."). \
Neither humility nor authority changes correctness.
Feedback is for an expert engineer: direct, technical, no praise-padding."""


def grade_free_text(chain: LLMChain, question: dict, learner_answer: str,
                    misconception_bank: dict) -> dict:
    ids = "\n".join(f"- {mid}: {m['name']}: {m['wrong_model']}"
                    for mid, m in misconception_bank.items())
    user = f"""QUESTION:
{question['prompt']}

REFERENCE ANSWER:
{question.get('answer', '(rubric only)')}

RUBRIC:
{question.get('rubric', '(match the reference answer)')}

KNOWN MISCONCEPTIONS (cite by ID only if the answer exhibits one):
{ids}

LEARNER ANSWER:
{learner_answer}"""
    try:
        return chain.structured("grader", GRADER_SYSTEM, user, GRADE_SCHEMA, "grade")
    except UpstreamError:
        # Offline-degraded: cannot adjudicate free text. Mark ungraded rather
        # than guessing; the item goes to the debt ledger as unassessed.
        return {"verdict": "ungraded", "misconceptions": [], "evidence": "",
                "feedback_md": "No model reachable - answer stored, not graded. "
                               "Compare against the reference answer shown.",
                "next_action": "Self-check against the reference, then continue."}


BATCH_GRADE_SCHEMA = {
    "type": "object",
    "properties": {
        "grades": {
            "type": "array",
            "items": {
                "type": "object",
                "properties": {
                    "item_id": {"type": "string"},
                    "verdict": {"type": "string",
                                "enum": ["pass", "partial", "fail", "valid_alternative_path"]},
                    "misconceptions": {"type": "array", "items": {"type": "string"}},
                    "evidence": {"type": "string"},
                    "feedback_md": {"type": "string"},
                    "next_action": {"type": "string"},
                },
                "required": ["item_id", "verdict", "misconceptions", "evidence",
                             "feedback_md", "next_action"],
            },
        },
    },
    "required": ["grades"],
}


def grade_free_text_batch(chain: LLMChain, items: list[tuple[str, dict, str]],
                          misconception_bank: dict) -> dict[str, dict]:
    """Grade several free-text answers in one call, keyed by item id.

    Per-item calls dominate check latency when each one pays model or harness
    startup (~55s per item through the Claude CLI). Grading is independent per
    item, so batching costs nothing pedagogically - the rubric for each item is
    still the only contract, and each grade still cites its own evidence.

    Returns whatever it could grade; the caller falls back per item for any id
    missing from the response, so a partial answer degrades instead of failing.
    """
    if not items:
        return {}
    ids = "\n".join(f"- {mid}: {m['name']}: {m['wrong_model']}"
                    for mid, m in misconception_bank.items())
    blocks = []
    for item_id, q, answer in items:
        blocks.append(
            f"### ITEM {item_id}\n"
            f"QUESTION:\n{q['prompt']}\n\n"
            f"REFERENCE ANSWER:\n{q.get('answer', '(rubric only)')}\n\n"
            f"RUBRIC:\n{q.get('rubric', '(match the reference answer)')}\n\n"
            f"LEARNER ANSWER:\n{answer}")
    user = (f"Grade each item below independently against its own rubric.\n\n"
            f"KNOWN MISCONCEPTIONS (cite by ID only if the answer exhibits one):\n{ids}\n\n"
            + "\n\n".join(blocks)
            + f"\n\nReturn one grade per item, using the exact item ids: "
              f"{', '.join(i[0] for i in items)}")
    try:
        out = chain.structured("grader", GRADER_SYSTEM, user, BATCH_GRADE_SCHEMA, "grades")
    except UpstreamError:
        return {}
    return {g["item_id"]: g for g in out.get("grades", []) if "item_id" in g}


# ---------------------------------------------------------------- planner

DIRECTIVE_SCHEMA = {
    "type": "object",
    "properties": {
        "depth": {"type": "string", "enum": ["compressed", "default", "deeper-math", "more-intuition"]},
        "sections_to_rewrite": {
            "type": "array",
            "items": {
                "type": "object",
                "properties": {
                    "heading": {"type": "string"},
                    "instruction": {"type": "string",
                                    "description": "one sentence: what to change and why, tied to learner evidence"},
                },
                "required": ["heading", "instruction"],
            },
        },
        "refutation_targets": {"type": "array", "items": {"type": "string"},
                               "description": "misconception IDs to attack explicitly in this chapter"},
        "opening_note_md": {"type": "string",
                            "description": "2-4 sentences addressed to the learner: what the last check showed and what this chapter will do about it. Specific, not cheerleading."},
        "summary": {"type": "string",
                    "description": "3-6 sentence prose learner-state summary, sufficient for a fresh model to predict how this learner does on the next check"},
        "next_action": {"type": "string"},
    },
    "required": ["depth", "sections_to_rewrite", "refutation_targets",
                 "opening_note_md", "summary", "next_action"],
}

PLANNER_SYSTEM = """You are the curriculum planner for an adaptive textbook teaching an expert \
software engineer how LLMs work, down to the math. You receive the learner's \
measured state and the next unit's outline; you emit directives for the chapter \
author.

Principles:
- Depth and directness are parameters, not personality. High mastery -> compress \
and raise difficulty. Struggle -> switch representation (symbolic <-> numeric <-> \
code <-> geometric), never re-explain the same way slower.
- The learner is expert in code/systems, novice in matrix calculus. Math gets \
full treatment; code gets one line. Expertise reversal is real: redundant \
explanation of known material actively hurts.
- Active misconceptions get named and refuted in the chapter body, not politely \
re-taught around.
- Fatigue evidence means lighter representation and a break suggestion, NOT a \
mastery downgrade.
- sections_to_rewrite: flag ONLY sections needing change from the default text \
(target 0-3). An empty list is a good answer when the default fits."""


def plan_directives(chain: LLMChain, state, unit, check_summary: str) -> dict:
    fallback = {
        "depth": "default", "sections_to_rewrite": [], "refutation_targets":
            [m for m in state.active_misconceptions()][:3],
        "opening_note_md": "", "summary": state.data["summary"],
        "next_action": "Read the chapter and work every beat before its reveal.",
    }
    # Cold start: no measured evidence yet -> nothing for a planner to adapt.
    # Default directives, no LLM call, instant first chapter.
    if not check_summary and not state.active_misconceptions() and not state.open_debt():
        return fallback
    sections = "\n".join(f"- {s.heading}" for s in unit.sections)
    depths = ", ".join(unit.depths.keys()) or "(none)"
    user = f"""LEARNER STATE SUMMARY:
{state.data['summary']}

ACTIVE MISCONCEPTIONS: {', '.join(state.active_misconceptions()) or 'none'}
FRAGILE CONCEPTS (recently shaky/low-confidence): {', '.join(state.fragile_concepts()) or 'none'}
OPEN DEBT: {[d['unit'] for d in state.open_debt()] or 'none'}
SESSION: {state.session_minutes():.0f} min elapsed, fatigue_flag={state.data['pacing']['fatigue_flag']}

LAST CHECK:
{check_summary or '(first unit - pretest evidence only)'}

NEXT UNIT: {unit.id} "{unit.title}" ({unit.minutes} min budget)
Unit notes: {unit.notes or '(none)'}
Sections:
{sections}
Available depth variants: {depths}
Concepts: {', '.join(f"{c['id']} ({state.concept_level(c['id'])})" for c in unit.concepts)}"""
    try:
        out = chain.structured("planner", PLANNER_SYSTEM, user, DIRECTIVE_SCHEMA, "directives")
        # sanitize: only real headings and known misconception ids survive,
        # and degenerate text fields fall back rather than propagating junk
        headings = {s.heading for s in unit.sections}
        out["sections_to_rewrite"] = [s for s in out["sections_to_rewrite"]
                                      if s["heading"] in headings][:3]
        if len(out.get("summary", "").strip()) < 20:
            out["summary"] = state.data["summary"]
        if len(out.get("next_action", "").strip()) < 10:
            out["next_action"] = fallback["next_action"]
        return out
    except UpstreamError:
        return fallback


# ---------------------------------------------------------------- author

REWRITE_SCHEMA = {
    "type": "object",
    "properties": {
        "sections": {
            "type": "array",
            "items": {
                "type": "object",
                "properties": {
                    "heading": {"type": "string"},
                    "markdown": {"type": "string"},
                },
                "required": ["heading", "markdown"],
            },
        },
    },
    "required": ["sections"],
}

AUTHOR_SYSTEM = """You rewrite specific sections of a textbook chapter for one specific learner. \
You receive the canonical section text, optional depth-variant text for the \
same section, and a one-sentence instruction from the planner.

Hard constraints:
- Preserve full technical coverage of the canonical section. You may change \
representation, depth, examples, and framing - never drop a concept.
- Keep every ```beat fenced block byte-for-byte intact and in a sensible position.
- Refutation style when attacking a misconception: state the wrong model, give \
the specific prediction it makes that fails, say why it is appealing, then the \
correct model.
- Real equations with symbols defined inline at first use. Restate rather than \
cross-reference. Worked numbers use tiny shapes that compute cleanly.
- Audience: expert software engineer, novice at ML math. Math explained fully, \
code in one line. Direct tone, no filler, no exclamation marks. Hyphens only, \
no em dashes.
Return ONLY the rewritten sections you were asked to rewrite."""


def author_chapter(chain: LLMChain, unit, directives: dict, corpus) -> tuple[list, bool]:
    """Assemble chapter sections. Returns (sections_markdown, llm_used).

    Code does the deterministic assembly (depth-variant swap); the LLM rewrites
    only flagged sections. If the rewrite mangles beats/headings, that section
    falls back to the code-assembled version - the reader always gets a chapter.
    """
    depth = directives.get("depth", "default")
    assembled = []  # [{heading, markdown, beats:[ids]}]
    for s in unit.sections:
        md = s.markdown()
        # swap only when the variant is substantial - depth files legitimately
        # contain "nothing to add here" stubs for some sections
        if (depth in unit.depths and s.heading in unit.depths[depth]
                and len(unit.depths[depth][s.heading]) > 400):
            # swap prose, keep the canonical beats appended after variant prose
            beats = [seg for seg in s.segments if seg["type"] == "beat"]
            md = f"## {s.heading}\n\n" + unit.depths[depth][s.heading]
            for seg in beats:
                import yaml as _yaml
                md += "\n\n```beat\n" + _yaml.safe_dump(seg["beat"], sort_keys=False) + "```"
        assembled.append({"heading": s.heading, "markdown": md,
                          "beats": [seg["beat"]["id"] for seg in s.segments
                                    if seg["type"] == "beat"]})

    rewrites = directives.get("sections_to_rewrite", [])
    if not rewrites:
        return assembled, False

    bank = corpus.misconceptions
    targets = "\n".join(
        f"- {mid}: {bank[mid]['wrong_model']} -> {bank[mid]['correction']}"
        for mid in directives.get("refutation_targets", []) if mid in bank)
    blocks = []
    for r in rewrites:
        sec = next(a for a in assembled if a["heading"] == r["heading"])
        variants = "\n\n".join(
            f"[variant: {name}]\n{sections[sec['heading']]}"
            for name, sections in unit.depths.items() if sec["heading"] in sections)
        blocks.append(f"""SECTION: {sec['heading']}
INSTRUCTION: {r['instruction']}
CANONICAL TEXT:
{sec['markdown']}
{variants and 'DEPTH VARIANTS AVAILABLE:' + chr(10) + variants or ''}""")
    user = f"""LEARNER NOTE FROM PLANNER: {directives.get('opening_note_md', '')}
MISCONCEPTIONS TO REFUTE WHERE RELEVANT:
{targets or '(none)'}

{chr(10).join(blocks)}"""
    try:
        out = chain.structured("author", AUTHOR_SYSTEM, user, REWRITE_SCHEMA, "rewrite")
    except UpstreamError:
        return assembled, False

    ok = False
    for rewritten in out.get("sections", []):
        target = next((a for a in assembled if a["heading"] == rewritten["heading"]), None)
        if target is None:
            continue
        md = rewritten["markdown"]
        if not md.lstrip().startswith("## "):
            md = f"## {target['heading']}\n\n{md}"
        # validation: every beat id from the original section must survive intact
        if all(f"id: {bid}" in md for bid in target["beats"]):
            target["markdown"] = md
            ok = True
    return assembled, ok


# ---------------------------------------------------------------- check composition (deterministic)

def compose_check(unit, state, corpus, n_items: int, callback_fraction: float,
                  rng: random.Random | None = None) -> list:
    """Deterministic cumulative check: ~60% current unit (constructed-first),
    ~40% callbacks from prior units weighted toward fragile concepts and
    active misconceptions."""
    rng = rng or random.Random()
    current_pool = list(unit.questions.get("check", []) or [])
    rng.shuffle(current_pool)
    current_pool.sort(key=lambda q: 0 if q.get("kind") == "constructed" else 1)
    n_current = max(1, round(n_items * (1 - callback_fraction)))
    items = [dict(q, unit=unit.id) for q in current_pool[:n_current]]

    fragile = set(state.fragile_concepts())
    active_mis = set(state.active_misconceptions())
    callbacks = []
    for uid in state.cleared_units():
        prior = corpus.units.get(uid)
        if not prior or uid == unit.id:
            continue
        for q in prior.questions.get("check", []) or []:
            if not q.get("callback_eligible"):
                continue
            weight = 1
            if q.get("concept") in fragile:
                weight += 2
            qmis = {o.get("misconception") for o in q.get("options", []) if isinstance(o, dict)}
            if qmis & active_mis:
                weight += 2
            callbacks.append((weight, rng.random(), dict(q, unit=uid)))
    callbacks.sort(key=lambda t: (-t[0], t[1]))
    items += [q for _, _, q in callbacks[: n_items - len(items)]]
    if len(items) < n_items:  # early units have no callback pool - top up from current
        seen = {q["id"] for q in items}
        items += [dict(q, unit=unit.id) for q in current_pool
                  if q["id"] not in seen][: n_items - len(items)]
    rng.shuffle(items)
    return items
