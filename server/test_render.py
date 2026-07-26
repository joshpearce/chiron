"""Render a real chapter through the app's exact renderer, outside the app.

Builds a standalone page from the same Resources the iPad app bundles
(chapter.html template, book.css, book.js, KaTeX) and injects a live chapter
payload the way the native bridge does. Load the output in a browser to check
what the learner will actually see: math typesetting, beat cards, typography.

Run:  .venv/bin/python test_render.py [unit]
Writes /tmp/chiron-render/index.html and reports anything suspicious.
"""

from __future__ import annotations

import json
import re
import shutil
import sys
from pathlib import Path

import httpx

HERE = Path(__file__).parent
RES = HERE.parent / "ipad-app/Chiron/Resources"
OUT = Path("/tmp/chiron-render")
SERVER = "http://127.0.0.1:8080"


def chapter_payload(unit: str | None) -> dict:
    """Prefer the static bundle (no model wait); fall back to a live exchange."""
    bundle = RES / "default-book.json"
    if bundle.exists():
        chapters = json.loads(bundle.read_text())["chapters"]
        if unit:
            for c in chapters:
                if c["unit"] == unit:
                    return c
        return chapters[0]
    r = httpx.post(f"{SERVER}/exchange", json={"subject": "ai", "phase": "start"}, timeout=900)
    r.raise_for_status()
    return r.json()["chapter"]


def main() -> int:
    unit = sys.argv[1] if len(sys.argv) > 1 else None
    ch = chapter_payload(unit)

    OUT.mkdir(parents=True, exist_ok=True)
    for name in ("book.css", "book.js"):
        shutil.copy(RES / name, OUT / name)
    if (OUT / "katex").exists():
        shutil.rmtree(OUT / "katex")
    shutil.copytree(RES / "katex", OUT / "katex")

    # Same template the app loads, plus the bridge shim the native side provides.
    template = (RES / "chapter.html").read_text()
    shim = (f"<script>window.addEventListener('load', function() {{\n"
            f"  initChapter({json.dumps(ch)});\n"
            f"}});</script>\n")
    page = template.replace("</body>", shim + "</body>")
    (OUT / "index.html").write_text(page)

    # Static sanity checks on the payload the renderer will consume.
    problems = []
    html = ch["html"]
    beat_ids = {b["id"] for b in ch["beats"]}
    placeholders = set(re.findall(r'data-beat-id="([^"]+)"', html))
    if placeholders != beat_ids:
        problems.append(f"beat mismatch: placeholders={placeholders - beat_ids} "
                        f"orphans={beat_ids - placeholders}")
    if "$" not in html:
        problems.append("no math delimiters found - KaTeX would have nothing to render")
    for b in ch["beats"]:
        if not b.get("prompt"):
            problems.append(f"beat {b['id']} has no prompt")
        if b.get("check", "llm") != "llm" and b.get("answer") in (None, ""):
            problems.append(f"beat {b['id']} is machine-checked but has no answer")
    for item in ch["check"]:
        if item["kind"] == "mcq":
            opts = (item.get("reveal") or {}).get("options") or []
            if sum(1 for o in opts if o.get("correct")) != 1:
                problems.append(f"item {item['id']}: not exactly one correct option")
        elif not (item.get("reveal") or {}).get("answer"):
            problems.append(f"item {item['id']}: constructed item with no reference answer")

    print(f"unit={ch['unit']} html={len(html)} beats={len(ch['beats'])} "
          f"pretest={len(ch['pretest'])} check={len(ch['check'])}")
    print(f"wrote {OUT / 'index.html'}")
    if problems:
        print("PROBLEMS:")
        for p in problems:
            print(" -", p)
        return 1
    print("payload checks: OK")
    return 0


if __name__ == "__main__":
    sys.exit(main())
