"""chiron-server: the adaptive engine behind the Chiron iPad app.

Chiron is a container of subjects; each subject has its own corpus and its own
learner state. One transport-agnostic exchange endpoint per interaction: the
app reads fully detached; at a chapter boundary it POSTs everything that
happened (beat responses, check answers, confidence ratings, timings,
override/catch-up requests) and receives grades, gate result, the next
chapter, and updated state - over Wi-Fi directly, or pushed through `iproxy`
on the USB path.
"""

from __future__ import annotations

import hmac
import random
from dataclasses import dataclass
from pathlib import Path

import yaml
from fastapi import FastAPI, HTTPException
from fastapi.responses import JSONResponse, PlainTextResponse
from pydantic import BaseModel

import checkers
import review
import roles
from corpus import Corpus
from llm_anthropic import make_chain
from render import render_chapter
from state import LearnerState

HERE = Path(__file__).parent
CFG = yaml.safe_load((HERE / "config.yaml").read_text())

app = FastAPI(title="chiron-server")
chain = make_chain(CFG)

# Optional shared-secret auth. Empty (the default) leaves the server open,
# which is correct on the flight LAN where the only client is the iPad on a
# Mac-hosted network. Set it before exposing the server on a public URL -
# without it, anyone who finds the endpoint can spend the tutor's model budget.
AUTH_TOKEN = (CFG.get("auth_token") or "").strip()

# Grade all free-text items of a check in one model call. Off by default: the
# per-item path is the one verified end to end, and a check is the moment a
# learner is most exposed to a regression. Worth turning on where per-call
# startup dominates (the Claude CLI upstream pays ~55s per item).
BATCH_GRADING = bool(CFG.get("batch_grading", False))


@app.middleware("http")
async def require_token(request, call_next):
    if AUTH_TOKEN and request.url.path != "/ping":
        supplied = request.headers.get("authorization", "")
        expected = f"Bearer {AUTH_TOKEN}"
        # Constant-time compare: a length/prefix-leaking check on a shared
        # secret is a bad habit even on a small deployment.
        if not hmac.compare_digest(supplied, expected):
            return JSONResponse({"detail": "unauthorized"}, status_code=401)
    return await call_next(request)


@app.get("/ping")
def ping():
    """Unauthenticated liveness probe - deliberately reveals nothing."""
    return {"ok": True}
SESSION = CFG["session"]
rng = random.Random()


@dataclass
class Subject:
    id: str
    title: str
    corpus: Corpus
    state: LearnerState


SUBJECTS: dict[str, Subject] = {}
for spec in CFG["subjects"]:
    corpus = Corpus((HERE / spec["corpus_dir"]).resolve())
    SUBJECTS[spec["id"]] = Subject(
        id=spec["id"], title=spec["title"], corpus=corpus,
        state=LearnerState((HERE / spec["state_dir"]).resolve(), corpus))


def _ctx(subject_id: str) -> Subject:
    s = SUBJECTS.get(subject_id)
    if s is None:
        raise HTTPException(404, f"unknown subject: {subject_id}")
    return s


class BeatResponse(BaseModel):
    beat_id: str
    response: str
    self_verdict: str | None = None      # learner's own pass/fail after reveal (self-explain beats)
    mechanical_verdict: str | None = None  # JS-graded result for compute beats


class ItemResponse(BaseModel):
    item_id: str
    response: str | None = None
    selected_index: int | None = None
    confidence: int  # 1-4, captured BEFORE reveal


class Exchange(BaseModel):
    subject: str = "ai"
    phase: str = "boundary"              # boundary | pretest | start
    unit: str | None = None
    beat_responses: list[BeatResponse] = []
    pretest_responses: list[ItemResponse] = []
    check_responses: list[ItemResponse] = []
    override: bool = False               # advance despite gate failure
    skipped_check: bool = False
    catch_me_up: bool = False
    choice: str | None = None            # learner-chosen unit from the fringe
    chunk_minutes: float | None = None
    break_minutes: float | None = None


# ------------------------------------------------------------------ helpers

def _grade_items(ctx: Subject, responses: list[ItemResponse], results: list) -> tuple[int, int]:
    """Grade item responses (mechanical in code, LLM for free text). Appends
    result dicts in submission order; returns (passed, total)."""
    # Resolve every item first so free-text ones can optionally be graded in a
    # single call. Order is preserved for both events and results.
    resolved: list[tuple[ItemResponse, dict, str]] = []
    for r in responses:
        q, unit_id = ctx.corpus.find_question(r.item_id)
        if q is not None:
            resolved.append((r, q, unit_id))

    free_text = [(r.item_id, q, r.response or "") for r, q, _ in resolved
                 if q.get("kind") != "mcq" and not checkers.is_mechanical(q.get("check", "llm"))]
    batched: dict[str, dict] = {}
    if BATCH_GRADING and len(free_text) > 1:
        batched = roles.grade_free_text_batch(chain, free_text, ctx.corpus.misconceptions)

    passed = 0
    for r, q, unit_id in resolved:
        if q.get("kind") == "mcq":
            g = checkers.check_mcq(q.get("options", []), r.selected_index or 0)
            g["feedback_md"] = g.pop("explain", "")
        elif checkers.is_mechanical(q.get("check", "llm")):
            ok = checkers.check_answer(q["check"], q.get("answer"), r.response or "")
            g = {"verdict": "pass" if ok else "fail", "misconceptions": [],
                 "feedback_md": f"Reference: {q.get('answer', '')}"}
        else:
            # Fall back per item for anything the batch did not return, so a
            # partial batch degrades rather than silently dropping grades.
            g = batched.get(r.item_id) or roles.grade_free_text(
                chain, q, r.response or "", ctx.corpus.misconceptions)
        ok = g["verdict"] in ("pass", "valid_alternative_path")
        passed += ok
        ctx.state.apply("item_graded", {
            "item": r.item_id, "concept": q.get("concept"), "unit": unit_id,
            "verdict": g["verdict"], "confidence": r.confidence,
            "misconceptions": g.get("misconceptions", []),
            "evidence": g.get("evidence", ""),
        })
        results.append({"item_id": r.item_id, **g, "confidence": r.confidence})
    return passed, len(responses)


def _grade_beats(ctx: Subject, responses: list[BeatResponse], results: list):
    for r in responses:
        beat, unit_id = ctx.corpus.find_beat(r.beat_id)
        if beat is None:
            continue
        check = beat.get("check", "llm")
        if checkers.is_mechanical(check):
            ok = checkers.check_answer(check, beat.get("answer"), r.response)
            g = {"verdict": "pass" if ok else "fail", "misconceptions": [],
                 "feedback_md": ""}
        else:
            g = roles.grade_free_text(
                chain, {"prompt": beat["prompt"], "answer": beat.get("answer"),
                        "rubric": beat.get("rubric")}, r.response, ctx.corpus.misconceptions)
        ctx.state.apply("item_graded", {
            "item": r.beat_id, "concept": beat.get("concept"), "unit": unit_id,
            "verdict": g["verdict"], "confidence": None,
            "misconceptions": g.get("misconceptions", []),
            "evidence": g.get("evidence", ""),
        })
        results.append({"item_id": r.beat_id, **g,
                        "self_verdict": r.self_verdict})


def _next_unit(ctx: Subject, choice: str | None) -> str | None:
    fringe = ctx.state.fringe()
    if not fringe:
        return None
    if choice and choice in fringe:
        return choice
    return fringe[0]


def _build_chapter(ctx: Subject, unit_id: str, check_summary: str) -> dict:
    unit = ctx.corpus.units[unit_id]
    directives = roles.plan_directives(chain, ctx.state, unit, check_summary)
    assembled, _ = roles.author_chapter(chain, unit, directives, ctx.corpus)
    check_items = roles.compose_check(
        unit, ctx.state, ctx.corpus, SESSION["check_items"],
        SESSION["callback_fraction"], rng)
    pretest = list(unit.questions.get("pretest", []) or [])
    ctx.state.apply("unit_started", {"unit": unit_id})
    ctx.state.apply("exposed", {"unit": unit_id, "concepts": unit.concept_ids})
    ctx.state.apply("summary", {"text": directives["summary"]})
    if unit.front.get("assumes"):
        ctx.state.apply("assumed_known", {"concepts": unit.front["assumes"]})
    return render_chapter(unit, assembled, directives, pretest, check_items)


def _build_catchup(ctx: Subject) -> dict | None:
    """Comprehensive backfill from the whole debt ledger: representation-
    switched sections for every debted concept, then a combined check that can
    retire the debt."""
    debt = ctx.state.open_debt()
    if not debt:
        return None
    sections, check_pool = [], []
    for d in debt:
        unit = ctx.corpus.units.get(d["unit"])
        if not unit:
            continue
        variant = next((v for v in ("more-intuition", "se-analogies", "deeper-math")
                        if v in unit.depths), None)
        for s in unit.sections:
            covers = any(seg["type"] == "beat" and seg["beat"].get("concept")
                         in d["concepts"] for seg in s.segments) or not d["concepts"]
            if not covers:
                continue
            mdtext = (f"## {s.heading}\n\n" + unit.depths[variant][s.heading]) \
                if variant and s.heading in unit.depths[variant] \
                and len(unit.depths[variant][s.heading]) > 400 else s.markdown()
            sections.append({"heading": f"[{unit.id}] {s.heading}",
                             "markdown": mdtext, "beats": []})
        missed = set(d.get("items_missed", []))
        for q in unit.questions.get("check", []) or []:
            if q["id"] in missed or q.get("concept") in d["concepts"]:
                check_pool.append(dict(q, unit=unit.id))
    if not sections:
        return None

    class _CatchupUnit:
        id = "catchup"
        title = "Catch-up: closing the gaps"
        minutes = max(15, 8 * len(debt))
        intro_md = ("This chapter consolidates everything skipped or missed so far, "
                    "explained differently than the first pass. The check at the end "
                    "retires the debt it covers.")
        sections = []

    directives = {"opening_note_md": "", "next_action":
                  "Work every section, then take the combined check.",
                  "summary": ctx.state.data["summary"]}
    rng.shuffle(check_pool)
    return render_chapter(_CatchupUnit(), sections, directives, [],
                          check_pool[: max(8, len(check_pool) // 2)])


def _state_payload(ctx: Subject) -> dict:
    spine = []
    for uid in ctx.corpus.unit_order():
        u = ctx.corpus.units.get(uid)
        spine.append({
            "unit": uid, "title": u.title if u else uid,
            "status": ctx.state.unit_status(uid),
            "score": ctx.state.data["units"].get(uid, {}).get("check_score"),
            "in_fringe": uid in ctx.state.fringe(),
        })
    return {
        "spine": spine,
        "fringe": ctx.state.fringe(),
        "debt": ctx.state.open_debt(),
        "active_misconceptions": ctx.state.active_misconceptions(),
        "summary": ctx.state.data["summary"],
        "session_minutes": ctx.state.session_minutes(),
        "llm": chain.status(),
    }


# ------------------------------------------------------------------ endpoints

@app.get("/health")
def health():
    return {"ok": True, "llm": chain.status(),
            "subjects": {s.id: list(s.corpus.units) for s in SUBJECTS.values()}}


@app.get("/subjects")
def subjects():
    out = []
    for s in SUBJECTS.values():
        total = len(s.corpus.unit_order())
        cleared = len(s.state.cleared_units())
        out.append({"id": s.id, "title": s.title, "units_total": total,
                    "units_cleared": cleared,
                    "current_unit": s.state.data.get("current_unit"),
                    "debt": len(s.state.open_debt())})
    return {"subjects": out}


@app.get("/state")
def get_state(subject: str = "ai"):
    return _state_payload(_ctx(subject))


@app.get("/review-schedule", response_class=PlainTextResponse)
def review_schedule(subject: str = "ai"):
    """Markdown day-1/3/10 review plan built from what actually went wrong.

    Self-contained (prompts + answers inline) so it is useful with no server,
    no model, and no network - print it, or push it to the Remarkable.
    """
    ctx = _ctx(subject)
    return review.build(ctx.state, ctx.corpus, ctx.title)


@app.post("/exchange")
def exchange(ex: Exchange):
    ctx = _ctx(ex.subject)
    state, corpus = ctx.state, ctx.corpus
    results: list = []
    gate = None
    chapter = None
    break_suggestion = None

    # telemetry first - it feeds the fatigue term
    if ex.chunk_minutes:
        state.apply("chunk", {"minutes": ex.chunk_minutes, "unit": ex.unit})
    if ex.break_minutes:
        state.apply("break_taken", {"minutes": ex.break_minutes})

    _grade_beats(ctx, ex.beat_responses, results)

    check_summary = ""
    if ex.pretest_responses:
        p, t = _grade_items(ctx, ex.pretest_responses, results)
        check_summary += f"Pretest: {p}/{t}. "

    if ex.check_responses:
        p, t = _grade_items(ctx, ex.check_responses, results)
        score = p / t if t else 0.0
        passed = score >= SESSION["mastery_gate"]
        unit = corpus.units.get(ex.unit) if ex.unit else None
        mastered = unit.concept_ids if (unit and passed) else []
        state.apply("check_result", {"unit": ex.unit, "score": score,
                                     "passed": passed, "mastered_concepts": mastered})
        if passed and ex.unit == "catchup":
            for d in state.open_debt():
                state.apply("debt_retired", {"unit": d["unit"]})
        missed = [r["item_id"] for r in results if r.get("verdict")
                  not in ("pass", "valid_alternative_path")]
        gate = {"score": score, "passed": passed,
                "gate": SESSION["mastery_gate"],
                "extension_unlocked": score >= SESSION["extension_trigger"]}
        check_summary += f"Check {ex.unit}: {score:.0%} ({'passed' if passed else 'below gate'}). "
        if not passed and ex.override and unit:
            state.apply("override", {"unit": ex.unit, "concepts":
                                     [c for c in unit.concept_ids
                                      if state.concept_level(c) != "mastered"],
                                     "items_missed": missed, "reason": "failed_gate"})
    elif ex.skipped_check and ex.unit and ex.unit in corpus.units:
        unit = corpus.units[ex.unit]
        state.apply("override", {"unit": ex.unit, "concepts": unit.concept_ids,
                                 "items_missed": [], "reason": "skipped_check"})
        gate = {"score": None, "passed": False, "gate": SESSION["mastery_gate"],
                "extension_unlocked": False}

    # fatigue heuristic: 90+ min without a logged break -> suggest one and flag
    if state.minutes_since_break() >= SESSION["long_break_every_chunks"] * SESSION["chunk_minutes"]:
        state.apply("fatigue", {})
        break_suggestion = {"minutes": 15, "kind": "long",
                            "note": "90+ minutes since a break. Late-session errors "
                                    "will be misread as knowledge gaps - rest first."}
    elif ex.chunk_minutes and ex.chunk_minutes >= SESSION["chunk_minutes"]:
        break_suggestion = {"minutes": SESSION["break_minutes"], "kind": "short",
                            "note": "Chunk done. Five minutes, eyes off screens."}

    # what to read next
    advance_allowed = (gate is None) or gate["passed"] or ex.override or ex.skipped_check
    if ex.catch_me_up:
        chapter = _build_catchup(ctx)
    if chapter is None and ex.phase in ("boundary", "start") and advance_allowed:
        nxt = _next_unit(ctx, ex.choice)
        if nxt:
            chapter = _build_chapter(ctx, nxt, check_summary)
    elif chapter is None and gate and not gate["passed"] and not ex.override:
        # remediation loop: rebuild the SAME unit; planner sees the failed
        # check in the summary and must switch representation
        chapter = _build_chapter(ctx, ex.unit, check_summary +
                                 "REMEDIATE: switch representation, do not re-explain the same way.")

    return {
        "results": results,
        "gate": gate,
        "chapter": chapter,
        "state": _state_payload(ctx),
        "break_suggestion": break_suggestion,
    }
