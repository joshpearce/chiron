"""Post-session spaced-review schedule.

Multi-day spacing is the one high-utility technique a single long session
cannot use, so the session's payoff is banked by exporting what to review and
when. Cepeda's ridgeline puts the optimal gap at roughly 10-20% of the target
retention interval; for "still know this months from now" that lands on
day 1 / day 3 / day 10 from the session.

What goes in each pass follows the confidence x accuracy routing used during
the flight:
  day 1  - everything missed, and every active misconception (highest decay
           risk, and wrong models harden if left uncorrected)
  day 3  - the day-1 set again, plus fragile items (right but unconfident,
           or shaky concepts) - retrieval while still retrievable
  day 10 - one item per concept covered, error-weighted; a cumulative sweep
           rather than a re-drill

Items are emitted with their prompt and reference answer so the schedule is
self-contained - it has to work with no server and no model.
"""

from __future__ import annotations

import json
from datetime import date, timedelta

OFFSETS = [("Day 1", 1), ("Day 3", 3), ("Day 10", 10)]


def _missed_items(state) -> list[dict]:
    """Items graded wrong, newest verdict wins (a later pass can redeem one)."""
    verdicts: dict[str, dict] = {}
    log = state.log_path
    if not log.exists():
        return []
    for line in log.read_text().splitlines():
        try:
            ev = json.loads(line)
        except json.JSONDecodeError:
            continue
        if ev.get("kind") != "item_graded":
            continue
        verdicts[ev["item"]] = ev
    return [v for v in verdicts.values()
            if v.get("verdict") not in ("pass", "valid_alternative_path")]


def _lookup(corpus, item_id: str) -> dict | None:
    q, unit = corpus.find_question(item_id)
    if q:
        return {"id": item_id, "unit": unit, "prompt": q.get("prompt", ""),
                "answer": q.get("answer", ""), "concept": q.get("concept")}
    beat, unit = corpus.find_beat(item_id)
    if beat:
        return {"id": item_id, "unit": unit, "prompt": beat.get("prompt", ""),
                "answer": beat.get("answer", ""), "concept": beat.get("concept")}
    return None


def build(state, corpus, subject_title: str, today: date | None = None) -> str:
    today = today or date.today()
    missed = [i for i in (_lookup(corpus, m["item"]) for m in _missed_items(state)) if i]
    fragile = state.fragile_concepts()
    misconceptions = state.active_misconceptions()
    mastered = [cid for cid, c in state.data["concepts"].items()
                if c.get("level") == "mastered"]

    # One representative item per concept for the day-10 cumulative sweep,
    # preferring concepts that produced an error at some point.
    by_concept: dict[str, dict] = {}
    for item in missed:
        if item.get("concept"):
            by_concept.setdefault(item["concept"], item)
    for unit in corpus.units.values():
        for q in unit.questions.get("check", []) or []:
            cid = q.get("concept")
            if cid and cid in mastered and cid not in by_concept:
                by_concept[cid] = {"id": q["id"], "unit": unit.id,
                                   "prompt": q.get("prompt", ""),
                                   "answer": q.get("answer", ""), "concept": cid}

    lines = [f"# Review schedule - {subject_title}",
             "",
             f"Session {today.isoformat()}. Retrieval beats rereading: cover the "
             "answer, say it out loud, then check. A pass you skip is the pass "
             "that mattered.",
             ""]

    for label, offset in OFFSETS:
        due = today + timedelta(days=offset)
        lines += [f"## {label} - {due.isoformat()}", ""]

        if label == "Day 1":
            pool, note = missed, "Everything you missed, while the corrections are still fresh."
        elif label == "Day 3":
            pool = missed
            note = "The same misses again, plus anything below."
        else:
            pool = list(by_concept.values())
            note = "One item per concept covered - a sweep, not a re-drill."
        lines += [note, ""]

        if not pool:
            lines += ["Nothing outstanding for this pass.", ""]
        for item in pool:
            lines += [f"**[{item['unit']}]** {item['prompt'].strip()}", "",
                      f"> {item['answer'].strip() or '(see the chapter)'}", ""]

        if label in ("Day 1", "Day 3") and misconceptions:
            lines += ["### Wrong models to actively contradict", ""]
            for mid in misconceptions:
                m = corpus.misconceptions.get(mid)
                if not m:
                    continue
                lines += [f"- **{m['name']}** - you leaned on: *{m['wrong_model']}*",
                          f"  - It fails because: {m['failing_prediction']}",
                          f"  - True: {m['correction']}", ""]

        if label == "Day 3" and fragile:
            names = []
            for cid in fragile:
                unit = corpus.units.get(corpus.concept_unit(cid) or "")
                label_ = next((c["name"] for u in corpus.units.values()
                               for c in u.concepts if c["id"] == cid), cid)
                names.append(f"- {label_} ({unit.id if unit else '?'})")
            lines += ["### Shaky - answered right but without confidence, or missed once", ""]
            lines += names + [""]

    lines += ["---", "",
              f"Concepts mastered this session: {len(mastered)}. "
              f"Still shaky: {len(fragile)}. "
              f"Misconceptions still active: {len(misconceptions)}.", ""]
    return "\n".join(lines)
