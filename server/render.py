"""Chapter payload rendering: assembled markdown sections -> one self-contained
HTML body plus structured beat/check data.

The iPad app owns the page template and KaTeX assets (bundled); the server
ships only the chapter body HTML. Beats are replaced by <div class="beat">
placeholders; the app's JS instantiates the interactive UI from the `beats`
array (including reference answers and rubrics - reveals are a learning tool
here, not exam security). Mechanical beats grade locally in JS so reading
works fully detached; llm-checked beats capture the response for adjudication
at the next exchange.
"""

from __future__ import annotations

import re

from markdown_it import MarkdownIt

BEAT_FENCE = re.compile(r"```beat\s*\n(.*?)```", re.S)

md = MarkdownIt("commonmark", {"html": True}).enable("table")


def _extract_beats(markdown_text: str, beats_out: list) -> str:
    """Replace beat fences with placeholder divs; collect beat dicts in order."""
    import yaml

    def repl(m):
        beat = yaml.safe_load(m.group(1))
        beats_out.append(beat)
        return f'\n<div class="beat" data-beat-id="{beat["id"]}"></div>\n'

    return BEAT_FENCE.sub(repl, markdown_text)


def render_chapter(unit, assembled_sections: list, directives: dict,
                   pretest: list, check_items: list) -> dict:
    beats: list = []
    html_parts: list[str] = []
    if directives.get("opening_note_md"):
        html_parts.append('<div class="planner-note">'
                          + md.render(directives["opening_note_md"]) + "</div>")
    if unit.intro_md:
        html_parts.append(md.render(_extract_beats(unit.intro_md, beats)))
    for sec in assembled_sections:
        html_parts.append(md.render(_extract_beats(sec["markdown"], beats)))

    return {
        "unit": unit.id,
        "title": unit.title,
        "minutes": unit.minutes,
        "html": "\n".join(html_parts),
        "beats": beats,
        "pretest": [_client_item(q) for q in pretest],
        "check": [_client_item(q) for q in check_items],
        "next_action": directives.get("next_action", ""),
    }


def _client_item(q: dict) -> dict:
    """Question as shipped to the app. Reference answers/rubrics ship too
    (needed for offline self-check display after answering); the app must not
    show them before the learner commits an answer + confidence."""
    out = {
        "id": q["id"], "unit": q.get("unit"), "concept": q.get("concept"),
        "kind": q.get("kind", "constructed"), "prompt": q["prompt"],
        "check": q.get("check", "llm"), "difficulty": q.get("difficulty", "core"),
    }
    if q.get("kind") == "mcq":
        out["options"] = [{"text": o["text"]} for o in q.get("options", [])]
        # per-option explanations revealed only after answer; index of correct
        out["reveal"] = {
            "options": [{"explain": o.get("explain", ""),
                         "correct": bool(o.get("correct"))}
                        for o in q.get("options", [])],
        }
    else:
        out["reveal"] = {"answer": q.get("answer", ""), "rubric": q.get("rubric", "")}
    return out
