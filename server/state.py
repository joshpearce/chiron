"""Learner model: single-writer store with a JSON snapshot and an append-only
JSONL event log. Every mastery change carries provenance (the event that
caused it). Only LearnerState mutates the model; everything else reads.

Mastery levels per concept: unseen -> exposed -> shaky -> mastered.
The known set (for fringe computation) = units whose terminal check passed the
gate OR were explicitly credited by a catch-up check.
"""

from __future__ import annotations

import json
import time
from pathlib import Path

LEVELS = ["unseen", "exposed", "shaky", "mastered"]


def _now() -> float:
    return time.time()


class LearnerState:
    def __init__(self, state_dir: str | Path, corpus):
        self.dir = Path(state_dir)
        self.dir.mkdir(parents=True, exist_ok=True)
        self.snapshot_path = self.dir / "learner.json"
        self.log_path = self.dir / "events.jsonl"
        self.corpus = corpus
        if self.snapshot_path.exists():
            self.data = json.loads(self.snapshot_path.read_text())
        else:
            self.data = self._fresh()
            self._save()

    def _fresh(self) -> dict:
        return {
            "version": 0,
            "concepts": {},          # id -> {level, history: [event refs], confidence: []}
            "units": {},             # id -> {status: locked|available|active|passed|overridden|debt,
                                     #        check_score, attempts}
            "misconceptions": {},    # id -> {evidence: [...], active: bool}
            "debt": [],              # [{unit, concepts, items_missed, reason, ts, retired}]
            "profile": {
                "expert_axes": ["code", "systems", "llm-tooling-behavior"],
                "novice_axes": ["matrix-calculus", "ml-notation"],
                "assumed_known": [],
            },
            "pacing": {"chunks": [], "breaks": [], "fatigue_flag": False,
                       "session_started": None},
            "summary": "New session. No evidence yet; treat per the static profile.",
            "current_unit": None,
            "chapter_cache": {},     # unit -> generated chapter payload (for resume)
        }

    # ---------- event-sourced mutation (the ONLY write path) ----------

    def apply(self, kind: str, payload: dict) -> dict:
        event = {"n": self.data["version"] + 1, "ts": _now(), "kind": kind, **payload}
        # The directory can disappear between boot and write (cleanup script,
        # redeploy, wiped volume). Losing a learner's progress mid-flight to a
        # missing mkdir is not an acceptable failure, so re-create on demand.
        self.dir.mkdir(parents=True, exist_ok=True)
        with self.log_path.open("a") as f:
            f.write(json.dumps(event) + "\n")
        self.data["version"] = event["n"]
        handler = getattr(self, f"_on_{kind}", None)
        if handler:
            handler(event)
        self._save()
        return event

    def _save(self):
        self.dir.mkdir(parents=True, exist_ok=True)
        self.snapshot_path.write_text(json.dumps(self.data, indent=1))

    # ---------- event handlers ----------

    def _concept(self, cid: str) -> dict:
        return self.data["concepts"].setdefault(
            cid, {"level": "unseen", "history": [], "confidence": []})

    def _set_level(self, cid: str, level: str, event_n: int, why: str):
        c = self._concept(cid)
        if c["level"] != level:
            c["history"].append({"n": event_n, "from": c["level"], "to": level, "why": why})
            c["level"] = level

    def _on_exposed(self, ev):
        for cid in ev["concepts"]:
            if self._concept(cid)["level"] == "unseen":
                self._set_level(cid, "exposed", ev["n"], f"read chapter {ev['unit']}")

    def _on_item_graded(self, ev):
        cid, verdict, conf = ev["concept"], ev["verdict"], ev.get("confidence")
        c = self._concept(cid)
        if conf is not None:
            c["confidence"].append({"n": ev["n"], "confidence": conf,
                                    "correct": verdict in ("pass", "valid_alternative_path")})
        ok = verdict in ("pass", "valid_alternative_path")
        cur = c["level"]
        if ok and cur in ("unseen", "exposed", "shaky"):
            # one correct answer never jumps straight to mastered from cold
            nxt = "shaky" if cur in ("unseen", "exposed") else "mastered"
            self._set_level(cid, nxt, ev["n"], f"correct on {ev['item']}")
        elif not ok and cur == "mastered":
            self._set_level(cid, "shaky", ev["n"], f"missed {ev['item']}")
        elif not ok and cur in ("unseen", "exposed"):
            self._set_level(cid, "shaky", ev["n"], f"missed {ev['item']} (needs work)")
        for mid in ev.get("misconceptions", []):
            m = self.data["misconceptions"].setdefault(mid, {"evidence": [], "active": True})
            m["active"] = True
            m["evidence"].append({"n": ev["n"], "item": ev["item"],
                                  "quote": ev.get("evidence", "")})

    def _on_misconception_cleared(self, ev):
        m = self.data["misconceptions"].get(ev["id"])
        if m:
            m["active"] = False
            m["evidence"].append({"n": ev["n"], "cleared_by": ev.get("why", "")})

    def _on_check_result(self, ev):
        u = self.data["units"].setdefault(ev["unit"], {"attempts": 0})
        u["attempts"] = u.get("attempts", 0) + 1
        u["check_score"] = ev["score"]
        u["status"] = "passed" if ev["passed"] else "failed"
        if ev["passed"]:
            for cid in ev.get("mastered_concepts", []):
                self._set_level(cid, "mastered", ev["n"], f"passed {ev['unit']} check @ {ev['score']:.0%}")

    def _on_override(self, ev):
        self.data["units"].setdefault(ev["unit"], {})["status"] = "overridden"
        self.data["debt"].append({
            "unit": ev["unit"], "concepts": ev["concepts"],
            "items_missed": ev.get("items_missed", []),
            "reason": ev["reason"],  # failed_gate | skipped_check
            "ts": _now(), "retired": False,
        })

    def _on_debt_retired(self, ev):
        for d in self.data["debt"]:
            if d["unit"] == ev["unit"] and not d["retired"]:
                d["retired"] = True

    def _on_chunk(self, ev):
        self.data["pacing"]["chunks"].append(
            {"n": ev["n"], "minutes": ev["minutes"], "unit": ev.get("unit")})

    def _on_break_taken(self, ev):
        self.data["pacing"]["breaks"].append({"n": ev["n"], "minutes": ev["minutes"]})
        self.data["pacing"]["fatigue_flag"] = False

    def _on_fatigue(self, ev):
        self.data["pacing"]["fatigue_flag"] = True

    def _on_summary(self, ev):
        self.data["summary"] = ev["text"]

    def _on_unit_started(self, ev):
        self.data["current_unit"] = ev["unit"]
        self.data["units"].setdefault(ev["unit"], {})["status"] = "active"
        if self.data["pacing"]["session_started"] is None:
            self.data["pacing"]["session_started"] = _now()

    def _on_chapter_cached(self, ev):
        self.data["chapter_cache"][ev["unit"]] = ev["ref"]

    def _on_assumed_known(self, ev):
        known = set(self.data["profile"]["assumed_known"])
        known.update(ev["concepts"])
        self.data["profile"]["assumed_known"] = sorted(known)

    # ---------- read-side queries ----------

    def unit_status(self, unit_id: str) -> str:
        return self.data["units"].get(unit_id, {}).get("status", "locked")

    def cleared_units(self) -> set:
        return {u for u, d in self.data["units"].items()
                if d.get("status") in ("passed", "overridden")}

    def fringe(self) -> list:
        """ALEKS-style outer fringe: units whose prereqs are all cleared and
        which aren't cleared themselves. Overridden units count as cleared for
        availability (agency) - the debt ledger carries the difference."""
        graph = self.corpus.prereq_graph()
        cleared = self.cleared_units()
        loaded = set(self.corpus.units)
        out = []
        for uid in self.corpus.unit_order():
            if uid in cleared or uid not in loaded:
                continue  # never offer a unit whose files are missing
            if all(p in cleared for p in graph.get(uid, []) if p in loaded):
                out.append(uid)
        return out

    def active_misconceptions(self) -> list:
        return [mid for mid, m in self.data["misconceptions"].items() if m["active"]]

    def open_debt(self) -> list:
        return [d for d in self.data["debt"] if not d["retired"]]

    def concept_level(self, cid: str) -> str:
        return self.data["concepts"].get(cid, {}).get("level", "unseen")

    def fragile_concepts(self) -> list:
        """Inner fringe: recently mastered or shaky - the review queue for
        cumulative-check callbacks. Low-confidence-correct also lands here."""
        out = []
        for cid, c in self.data["concepts"].items():
            if c["level"] == "shaky":
                out.append(cid)
            elif c["level"] == "mastered" and c["confidence"]:
                last = c["confidence"][-1]
                if last["correct"] and last["confidence"] <= 2:
                    out.append(cid)
        return out

    def session_minutes(self) -> float:
        start = self.data["pacing"]["session_started"]
        return (time.time() - start) / 60 if start else 0.0

    def minutes_since_break(self) -> float:
        breaks = self.data["pacing"]["breaks"]
        chunks = self.data["pacing"]["chunks"]
        if not chunks:
            return 0.0
        return sum(c["minutes"] for c in chunks[-4:]) if not breaks else \
            sum(c["minutes"] for c in chunks if c["n"] > breaks[-1]["n"])
