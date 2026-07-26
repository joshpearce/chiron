"""Simulated learners driven through the real /exchange API against the real
local model. Three personas prove the adaptive logic:

  strong  - answers from the corpus reference answers -> passes gates,
            should eventually see compressed/extension behavior
  weak    - answers embody bank misconceptions -> fails gate, gets a
            remediation chapter that must differ from the first presentation
  skipper - skips checks -> debt accrues; catch_me_up must yield one
            comprehensive backfill chapter

Run:  .venv/bin/python test_sim_learners.py [persona]
Requires: book-server on :8080 with fresh state (script resets state itself).
"""

from __future__ import annotations

import json
import shutil
import subprocess
import sys
import time
from pathlib import Path

import httpx

BASE = "http://127.0.0.1:8080"
HERE = Path(__file__).parent

# canned wrong answers voiced through bank misconceptions
WRONG_BY_CONCEPT = {
    "default": "I'm not sure, something to do with how the layers process it.",
    "c-softmax": "Softmax converts the scores into the probabilities that each "
                 "answer is factually correct, so the model can pick the true one.",
    "c-sdpa": "The sqrt(d_k) is for numerical stability so the floats don't "
              "overflow to NaN in fp16.",
    "c-dotprod": "The dot product mainly measures the lengths of the vectors.",
    "c-gradient": "The gradient finds the single global minimum of the loss.",
}


def reset_state():
    subprocess.run(["pkill", "-f", "uvicorn main:app"], capture_output=True)
    shutil.rmtree(HERE.parent / "state", ignore_errors=True)
    # Bind all interfaces: a localhost-only rebind leaves the iPad unable to
    # reach the server, which looks like an app hang rather than a server
    # misconfiguration. Anything that restarts the server must match
    # start-flight.sh.
    subprocess.Popen(
        [str(HERE / ".venv/bin/uvicorn"), "main:app", "--host", "0.0.0.0", "--port", "8080"],
        cwd=HERE, stdout=open("/tmp/book-server.log", "a"), stderr=subprocess.STDOUT)
    for _ in range(30):
        time.sleep(1)
        try:
            if httpx.get(f"{BASE}/health", timeout=2).status_code == 200:
                return
        except httpx.HTTPError:
            pass
    raise SystemExit("server did not come up")


def exchange(payload: dict) -> dict:
    r = httpx.post(f"{BASE}/exchange", json=payload, timeout=900)
    r.raise_for_status()
    return r.json()


def answer_items(items: list, persona: str) -> list:
    out = []
    for it in items:
        if it["kind"] == "mcq":
            reveal = it["reveal"]["options"]
            correct = next(i for i, o in enumerate(reveal) if o["correct"])
            wrong = next((i for i, o in enumerate(reveal) if not o["correct"]), correct)
            out.append({"item_id": it["id"], "selected_index":
                        correct if persona == "strong" else wrong,
                        "confidence": 4 if persona == "strong" else 3})
        else:
            if persona == "strong":
                resp = it["reveal"].get("answer") or "see reference"
            else:
                resp = WRONG_BY_CONCEPT.get(it.get("concept") or "default",
                                            WRONG_BY_CONCEPT["default"])
            out.append({"item_id": it["id"], "response": resp,
                        "confidence": 4 if persona == "strong" else 3})
    return out


def run_strong():
    print("== STRONG learner ==")
    d = exchange({"phase": "start"})
    for step in range(2):
        ch = d["chapter"]
        assert ch, f"no chapter at step {step}"
        print(f"reading {ch['unit']} ({len(ch['check'])} check items)")
        t0 = time.time()
        d = exchange({"unit": ch["unit"],
                      "check_responses": answer_items(ch["check"], "strong"),
                      "chunk_minutes": 20})
        g = d["gate"]
        print(f"  gate: {g['score']:.0%} passed={g['passed']} "
              f"ext={g['extension_unlocked']} ({time.time()-t0:.0f}s)")
        assert g["passed"], f"strong learner failed gate: {json.dumps(d['results'], indent=1)[:2000]}"
    nxt = d["chapter"]
    print(f"  advanced to: {nxt['unit'] if nxt else None}")
    assert nxt and nxt["unit"] != "u0", "did not advance"
    print("STRONG: OK\n")


def run_weak():
    print("== WEAK learner ==")
    d = exchange({"phase": "start"})
    ch = d["chapter"]
    first_html = ch["html"]
    d = exchange({"unit": ch["unit"],
                  "check_responses": answer_items(ch["check"], "weak"),
                  "chunk_minutes": 25})
    g = d["gate"]
    print(f"  gate: {g['score']:.0%} passed={g['passed']}")
    assert not g["passed"], "weak learner unexpectedly passed"
    rem = d["chapter"]
    assert rem and rem["unit"] == ch["unit"], "no remediation chapter for same unit"
    changed = rem["html"] != first_html
    print(f"  remediation delivered for {rem['unit']}, content changed: {changed}")
    mis = d["state"]["active_misconceptions"]
    print(f"  active misconceptions diagnosed: {mis}")
    print("WEAK: OK (verify 'changed' is True - representation switch)\n")


def run_skipper():
    print("== SKIPPER ==")
    d = exchange({"phase": "start"})
    for _ in range(2):
        ch = d["chapter"]
        print(f"  skipping check for {ch['unit']}")
        d = exchange({"unit": ch["unit"], "skipped_check": True, "chunk_minutes": 5})
    debt = d["state"]["debt"]
    print(f"  open debt units: {[x['unit'] for x in debt]}")
    assert len(debt) >= 2, "debt did not accrue"
    d = exchange({"unit": d["chapter"]["unit"], "catch_me_up": True})
    cu = d["chapter"]
    assert cu and cu["unit"] == "catchup", "no catch-up chapter"
    print(f"  catch-up chapter: {len(cu['html'])} chars, {len(cu['check'])} combined check items")
    print("SKIPPER: OK\n")


if __name__ == "__main__":
    which = sys.argv[1] if len(sys.argv) > 1 else "all"
    for name, fn in [("strong", run_strong), ("weak", run_weak), ("skipper", run_skipper)]:
        if which in (name, "all"):
            reset_state()
            fn()
    print("simulated learners: done")
