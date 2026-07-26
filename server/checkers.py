"""Mechanical answer checking. Anything a program can check never goes to the LLM.

check spec grammar (from authoring-spec.md):
    exact           - case/whitespace-insensitive string match
    numeric(0.01)   - parse first number in the response, compare with tolerance
    choice          - MCQ option index/letter match
    llm             - not handled here (grader role)
"""

from __future__ import annotations

import re

NUMERIC = re.compile(r"numeric\(([\d.eE+-]+)\)")
# Accept "3/4"-style fractions as well as plain floats/scientific notation.
NUMBER = re.compile(r"-?\d+(?:\.\d+)?(?:[eE][+-]?\d+)?(?:\s*/\s*-?\d+(?:\.\d+)?)?")


def is_mechanical(check: str) -> bool:
    return check != "llm"


def _parse_number(text: str) -> float | None:
    m = NUMBER.search(str(text).replace(",", ""))
    if not m:
        return None
    token = m.group(0)
    if "/" in token:
        num, den = token.split("/")
        try:
            return float(num) / float(den)
        except (ValueError, ZeroDivisionError):
            return None
    try:
        return float(token)
    except ValueError:
        return None


def check_answer(check: str, expected, given) -> bool:
    """Return pass/fail for a mechanical check."""
    if check == "exact":
        return _norm(given) == _norm(expected)
    m = NUMERIC.fullmatch(check.strip())
    if m:
        tol = float(m.group(1))
        exp, got = _parse_number(expected), _parse_number(given)
        if exp is None or got is None:
            return False
        # Relative tolerance for large magnitudes, absolute for small.
        return abs(got - exp) <= max(tol, abs(exp) * tol)
    if check == "choice":
        return _norm(given) == _norm(expected)
    raise ValueError(f"not a mechanical check: {check}")


def check_mcq(options: list, selected_index: int) -> dict:
    """Grade an MCQ selection; returns verdict plus the misconception diagnosis."""
    if not 0 <= selected_index < len(options):
        return {"verdict": "fail", "misconceptions": [], "explain": "no option selected"}
    opt = options[selected_index]
    if opt.get("correct"):
        return {"verdict": "pass", "misconceptions": [], "explain": opt.get("explain", "")}
    out = {"verdict": "fail", "misconceptions": [], "explain": opt.get("explain", "")}
    if opt.get("misconception"):
        out["misconceptions"] = [opt["misconception"]]
    return out


def _norm(s) -> str:
    return re.sub(r"\s+", " ", str(s)).strip().lower()
