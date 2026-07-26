"""Corpus loading and parsing.

A unit directory contains canon.md (prose + fenced ```beat blocks), depths/
variants keyed by identical section headings, questions.yaml, and
misconceptions.yaml. Canon is parsed into an ordered list of segments:
{"type": "prose", "md": ...} and {"type": "beat", "beat": {...}}, grouped by
section heading so depth variants can be swapped per section.
"""

from __future__ import annotations

import re
from dataclasses import dataclass, field
from pathlib import Path

import yaml

BEAT_FENCE = re.compile(r"^```beat\s*$(.*?)^```\s*$", re.M | re.S)
FRONT_MATTER = re.compile(r"\A---\s*\n(.*?)\n---\s*\n", re.S)
SECTION = re.compile(r"^## (.+)$", re.M)


@dataclass
class Section:
    heading: str
    segments: list = field(default_factory=list)  # prose/beat dicts in order

    def markdown(self) -> str:
        parts = [f"## {self.heading}"]
        for seg in self.segments:
            if seg["type"] == "prose":
                parts.append(seg["md"])
            else:
                parts.append("```beat\n" + yaml.safe_dump(seg["beat"], sort_keys=False) + "```")
        return "\n\n".join(parts)


@dataclass
class Unit:
    id: str
    slug: str
    title: str
    minutes: int
    prereqs: list
    concepts: list            # [{id, name}]
    notes: str
    front: dict               # canon front matter (assumes, etc.)
    intro_md: str             # prose before the first ## section
    sections: list            # [Section]
    depths: dict              # depth-name -> {heading -> md}
    questions: dict           # {"pretest": [...], "check": [...]}
    misconceptions: list      # unit-local entries

    @property
    def concept_ids(self):
        return [c["id"] for c in self.concepts]

    def beats(self):
        for s in self.sections:
            for seg in s.segments:
                if seg["type"] == "beat":
                    yield seg["beat"]


class Corpus:
    def __init__(self, corpus_dir: str | Path):
        self.dir = Path(corpus_dir)
        self.syllabus = yaml.safe_load((self.dir / "syllabus.yaml").read_text())
        bank = yaml.safe_load((self.dir / "misconception-bank.yaml").read_text())
        self.misconceptions = {m["id"]: m for m in bank["misconceptions"]}
        self.units: dict[str, Unit] = {}
        self.extensions = {x["id"]: x for x in self.syllabus.get("extensions", [])}
        for spec in self.syllabus["units"]:
            unit = self._load_unit(spec)
            if unit:
                self.units[unit.id] = unit
                for m in unit.misconceptions:
                    self.misconceptions.setdefault(m["id"], m)

    # ---------- loading ----------

    def _load_unit(self, spec: dict) -> Unit | None:
        udir = self.dir / "units" / f"{spec['id']}-{spec['slug']}"
        canon_path = udir / "canon.md"
        if not canon_path.exists():
            return None  # unit not authored yet; server tolerates partial corpus
        front, intro_md, sections = self._parse_canon(canon_path.read_text())
        depths = {}
        ddir = udir / "depths"
        if ddir.exists():
            for f in sorted(ddir.glob("*.md")):
                depths[f.stem] = self._sections_by_heading(f.read_text())
        questions = yaml.safe_load((udir / "questions.yaml").read_text()) if (udir / "questions.yaml").exists() else {"pretest": [], "check": []}
        mpath = udir / "misconceptions.yaml"
        local = yaml.safe_load(mpath.read_text())["misconceptions"] if mpath.exists() else []
        return Unit(
            id=spec["id"], slug=spec["slug"], title=spec["title"],
            minutes=spec.get("minutes", 25), prereqs=spec.get("prereqs", []),
            concepts=spec.get("concepts", []), notes=spec.get("notes", ""),
            front=front, intro_md=intro_md, sections=sections, depths=depths,
            questions=questions, misconceptions=local,
        )

    def _parse_canon(self, text: str):
        front = {}
        m = FRONT_MATTER.match(text)
        if m:
            front = yaml.safe_load(m.group(1)) or {}
            text = text[m.end():]

        # Split into sections on ## headings, preserving order.
        indices = [(m.start(), m.group(1).strip()) for m in SECTION.finditer(text)]
        intro = text[: indices[0][0]] if indices else text
        sections = []
        for i, (start, heading) in enumerate(indices):
            end = indices[i + 1][0] if i + 1 < len(indices) else len(text)
            body = text[start:end]
            body = body.split("\n", 1)[1] if "\n" in body else ""
            sections.append(Section(heading=heading, segments=self._parse_segments(body)))
        return front, intro.strip(), sections

    def _parse_segments(self, body: str) -> list:
        segments, cursor = [], 0
        for m in BEAT_FENCE.finditer(body):
            prose = body[cursor:m.start()].strip()
            if prose:
                segments.append({"type": "prose", "md": prose})
            beat = yaml.safe_load(m.group(1))
            segments.append({"type": "beat", "beat": beat})
            cursor = m.end()
        tail = body[cursor:].strip()
        if tail:
            segments.append({"type": "prose", "md": tail})
        return segments

    def _sections_by_heading(self, text: str) -> dict:
        m = FRONT_MATTER.match(text)
        if m:
            text = text[m.end():]
        indices = [(mm.start(), mm.group(1).strip()) for mm in SECTION.finditer(text)]
        out = {}
        for i, (start, heading) in enumerate(indices):
            end = indices[i + 1][0] if i + 1 < len(indices) else len(text)
            body = text[start:end].split("\n", 1)
            out[heading] = body[1].strip() if len(body) > 1 else ""
        return out

    # ---------- queries ----------

    def unit_order(self):
        return [u["id"] for u in self.syllabus["units"]]

    def prereq_graph(self):
        return {u["id"]: u.get("prereqs", []) for u in self.syllabus["units"]}

    def concept_unit(self, concept_id: str) -> str | None:
        for u in self.units.values():
            if concept_id in u.concept_ids:
                return u.id
        return None

    def find_question(self, qid: str):
        for u in self.units.values():
            for pool in ("pretest", "check"):
                for q in u.questions.get(pool, []) or []:
                    if q["id"] == qid:
                        return q, u.id
        return None, None

    def find_beat(self, beat_id: str):
        for u in self.units.values():
            for b in u.beats():
                if b["id"] == beat_id:
                    return b, u.id
        return None, None
