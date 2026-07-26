"""Pre-render the whole book at default depth into a JSON bundle shipped
inside the iPad app. Worst-case mode (Mac unreachable, no flight wifi): the
app serves chapters from this bundle - full reading experience, JS-graded
mechanical beats, reveal-based self-checks. Only free-text adjudication and
adaptivity are lost.

Run after corpus changes:  .venv/bin/python build_static_bundle.py
"""

import json
from pathlib import Path

from corpus import Corpus
from render import render_chapter
from roles import compose_check

HERE = Path(__file__).parent
OUT = HERE.parent / "ipad-app/DynamicBook/Resources/default-book.json"


class _NullState:
    """Stateless stand-ins so compose_check runs without a learner."""
    def fragile_concepts(self):
        return []

    def active_misconceptions(self):
        return []

    def cleared_units(self):
        return set()


def main():
    corpus = Corpus(HERE.parent / "corpus")
    null = _NullState()
    chapters = []
    for uid in corpus.unit_order():
        unit = corpus.units[uid]
        assembled = [{"heading": s.heading, "markdown": s.markdown(),
                      "beats": [seg["beat"]["id"] for seg in s.segments
                                if seg["type"] == "beat"]}
                     for s in unit.sections]
        directives = {"opening_note_md": "", "next_action":
                      "Static mode - work every beat, self-grade the check honestly.",
                      "summary": ""}
        check = compose_check(unit, null, corpus, 9, 0.0)
        payload = render_chapter(unit, assembled, directives,
                                 list(unit.questions.get("pretest", []) or []), check)
        chapters.append(payload)
    OUT.write_text(json.dumps({"chapters": chapters}))
    print(f"wrote {OUT} ({OUT.stat().st_size // 1024} KB, {len(chapters)} chapters)")


if __name__ == "__main__":
    main()
