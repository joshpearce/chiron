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

# Display math first so $$...$$ is not mistaken for two empty $...$ spans.
MATH_SPAN = re.compile(r"\$\$.+?\$\$|(?<!\\)\$(?!\s)(?:\\.|[^$\\])+?(?<!\s)\$", re.S)

md = MarkdownIt("commonmark", {"html": True}).enable("table")


def _protect_math(text: str) -> tuple[str, list[str]]:
    """Pull math spans out before markdown conversion.

    CommonMark treats a backslash before ASCII punctuation as an escape, so
    markdown-it silently rewrites `\\{` -> `{` and `\\}` -> `}` inside what we
    intend as LaTeX. KaTeX then sees an unmatched brace and renders an error
    box. Hiding math behind placeholders keeps the TeX byte-exact.
    """
    spans: list[str] = []

    def stash(m: re.Match) -> str:
        spans.append(m.group(0))
        # Placeholder must survive markdown untouched: letters and digits only.
        return f"MATHPLACEHOLDER{len(spans) - 1}ENDMATH"

    return MATH_SPAN.sub(stash, text), spans


def _restore_math(html: str, spans: list[str]) -> str:
    """Put math back, HTML-escaped.

    LaTeX routinely contains `<`, `>` and `&` (subscripts like x_{<t},
    alignment in matrices). Emitted raw, the browser's HTML parser swallows
    `<t}` as an unknown tag and the equation silently loses characters. Escaped,
    the parser leaves it alone and KaTeX - which reads textContent - still sees
    the original TeX.
    """
    for i, span in enumerate(spans):
        safe = (span.replace("&", "&amp;")
                    .replace("<", "&lt;")
                    .replace(">", "&gt;"))
        html = html.replace(f"MATHPLACEHOLDER{i}ENDMATH", safe)
    return html


def _render_md(text: str) -> str:
    """Markdown -> HTML with LaTeX preserved byte-exact."""
    protected, spans = _protect_math(text)
    return _restore_math(md.render(protected), spans)


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
        html_parts.append(_render_md(_extract_beats(unit.intro_md, beats)))
    for sec in assembled_sections:
        html_parts.append(_render_md(_extract_beats(sec["markdown"], beats)))

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
        # Answers to compute items are numbers in the corpus (`answer: 6`).
        # The client types this field as a string, and a single numeric answer
        # fails the decode of the whole payload - which silently killed the
        # entire offline book. Normalise to text at the boundary.
        out["reveal"] = {"answer": _as_text(q.get("answer", "")),
                         "rubric": _as_text(q.get("rubric", ""))}
    return out


def _as_text(value) -> str:
    if value is None:
        return ""
    if isinstance(value, bool):
        return "true" if value else "false"
    return value if isinstance(value, str) else str(value)
