"""Mechanical grading contract.

These cases are duplicated in the offline JS grader (`book.js`
`gradeMechanical` / `norm`). Both must agree: a beat is graded in JavaScript
while reading offline and in Python at the chapter boundary, so any divergence
means the same answer is right in one place and wrong in the other. If you
change one, change the other and re-run both.

Run:  .venv/bin/python test_checkers.py
"""

from __future__ import annotations

import sys

import checkers

CASES = [
    # (check, expected, given, should_pass, why)
    ("numeric(1)", 50331648, "50331648", True, "exact hit"),
    ("numeric(1)", 50331648, "100663296", False,
     "double the answer - the old max(tol, |exp|*tol) rule made numeric(1) mean +/-100% and passed this"),
    ("numeric(1)", 50331648, "50331649", True, "within the intended +/-1"),
    ("numeric(0.001)", 0.50349, "0.503", True, "rounded to three places"),
    ("numeric(0.001)", 0.50349, "0.5", False,
     "the naive symmetry guess an attention beat exists to catch"),
    ("numeric(0.01)", 5.5, "-5.5", False, "sign flip"),
    ("exact", "4, 1; 11, 6", "4,1;11,6", True, "separator spacing is not meaning"),
    ("exact", "4, 1; 11, 6", " 4 , 1 ; 11 , 6 ", True, "padded separators"),
    ("exact", "4, 1; 11, 6", "4, 1; 11, 7", False, "one entry wrong"),
    ("exact", "4 x 7", "4 X 7", True, "case-insensitive"),
    ("numeric(0.01)", 6, "the answer is 6", True, "prose around a number still parses"),
]


def main() -> int:
    failures = 0
    for check, expected, given, want, why in CASES:
        got = checkers.check_answer(check, expected, given)
        if got != want:
            print(f"FAIL {check} expected={expected!r} given={given!r} "
                  f"-> {got}, wanted {want} ({why})")
            failures += 1
    print(f"{len(CASES) - failures}/{len(CASES)} mechanical grading cases pass")
    if failures == 0:
        print("reminder: book.js gradeMechanical/norm must mirror these")
    return 1 if failures else 0


if __name__ == "__main__":
    sys.exit(main())
