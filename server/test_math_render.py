"""Regression tests for math survival through markdown -> HTML.

Two independent hazards, both of which silently corrupted equations in the
shipped book before these guards existed:

  1. CommonMark backslash escapes rewrite `\\{` -> `{` inside what we mean as
     LaTeX, so KaTeX sees an unmatched brace and renders an error box.
  2. LaTeX containing `<` (e.g. the subscript x_{<t}) is swallowed by the
     browser's HTML parser as an unknown tag, truncating the equation.

Run:  .venv/bin/python test_math_render.py
"""

from __future__ import annotations

import sys

import render

CASES = [
    # (name, markdown in, must appear in HTML, must NOT appear)
    ("escaped braces survive",
     r"the nucleus is $\{$`The`, `A`$\}$ done", [r"$\{$", r"$\}$"], []),
    ("less-than in subscript is html-escaped",
     r"$$P_\theta(x_t \mid x_{<t})$$", [r"x_{&lt;t}"], ["x_{<t}"]),
    ("ampersand in alignment is escaped",
     r"$$a &= b$$", ["&amp;= b"], []),
    ("percent and underscore keep backslashes",
     r"$50\%$ and $a\_b$", [r"$50\%$", r"$a\_b$"], []),
    ("display math untouched",
     r"$$\frac{\partial L}{\partial W}$$", [r"\frac{\partial L}{\partial W}"], []),
    ("prose markdown still renders",
     "**bold** and `code`", ["<strong>bold</strong>", "<code>code</code>"], []),
    ("dollar amounts are not math",
     "costs $5 and $10 total", ["costs $5 and $10 total"], []),
    ("greater-than in math is escaped",
     r"$x > 0$", ["&gt;"], []),
]


def main() -> int:
    failures = 0
    for name, src, expect, forbid in CASES:
        out = render._render_md(src)
        for want in expect:
            if want not in out:
                print(f"FAIL {name}: missing {want!r}\n     got {out.strip()[:120]!r}")
                failures += 1
        for bad in forbid:
            if bad in out:
                print(f"FAIL {name}: should not contain {bad!r}\n     got {out.strip()[:120]!r}")
                failures += 1
        if not failures:
            print(f"ok   {name}")
    print(f"\n{len(CASES) - failures}/{len(CASES)} math-rendering cases pass")
    return 1 if failures else 0


if __name__ == "__main__":
    sys.exit(main())
