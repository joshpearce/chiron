# Flight-day runbook

## MORNING STATUS (overnight of 2026-07-25 -> 26)

**Everything server-side is green. One thing needs your hands: unlock the iPad.**

Do this first:

1. **Unlock the iPad** (it locked overnight, which blocked the last step). Then
   set Settings > Display & Brightness > Auto-Lock > Never for the flight.
2. A self-test is armed and will fire automatically within 60s of unlock. Check
   it with: `grep chiron-selftest /tmp/chiron-selftest.log` - expect a `PASS`
   line. If the watcher already expired, run it manually:
   `ios-deploy --bundle <DerivedData>/Chiron.app --args selftest --justlaunch --noinstall`
3. Then do one human pass in the app (tap Begin, work a beat, take the check)
   to confirm the reading experience, not just the protocol.
4. **Eyeball the check screen specifically.** 36% of check items contain LaTeX,
   which SwiftUI's `Text` cannot render - they would have shown as raw `$q$` to
   you mid-flight. That is now rendered through KaTeX (`MathText.swift`), but it
   is the one change I could not see on the device. Confirm equations look
   typeset and nothing is clipped. If a prompt looks cut off, the height
   measurement failed; it falls back to a deliberate over-estimate, so the
   symptom would be extra blank space rather than lost text.

### Verified overnight (no human needed)

- **Local flight path is up right now**: book-server on `0.0.0.0:8080` reachable
  at `192.168.2.1:8080`, LM Studio loaded, Internet Sharing up.
- **Simulated learners pass on the final 11-unit corpus**: strong (100%, gates
  pass, extensions unlock, advances u0->u1->u2), weak (gate blocks at 0%,
  remediation delivered, misconception diagnosed), skipper (debt accrues on
  u0+u1, catch-up chapter = 21k chars / 12 combined items).
- **Grader red-team 5/5**: fails fluent-wrong, authority-citing, and
  face-saving-wrong; passes tentative-correct and valid-alternative.
- **USB transport proven end-to-end** without the iPad, via a mock device
  listener that mimics the app's embedded listener
  (`server/test_usb_bridge.py`): request out, full chapter back.
- **Sprite path proven end-to-end through Claude**: chapter generation plus a
  real weak-learner check - gate failed correctly, remediation issued,
  misconceptions U0-M1 + M2 diagnosed.

### Every screen has now been seen, and both learner paths run end to end

Driven hands-free in the simulator against the live server (no cable, no
unlock):

- **Strong path**: subjects -> chapter u2 generated over Wi-Fi -> 9-item check
  submitted -> **100%, gate cleared, extensions unlocked, advanced to u3**.
  This is the first time the client side has completed an exchange at all.
- **Weak path**: every item answered with a confident misconception ->
  **0%, gate held**, per-item feedback naming the misconception (M2,
  softmax-truth-probability) with a real explanation, and the
  Remediate / Override choice with the debt-ledger note.
- **Screens confirmed by eye**: library, reader (cream paper, serif, math
  typeset inline), check (math typeset, confidence slider), gate cleared, gate
  failed, break timer.

Your learner state has been **reset**, so you start the flight clean.

**Measured at the real 32K context** the flight uses (not the 16K I developed
against): the model loads at 20.4 GB, chapter delivery is effectively instant,
and a full 9-item check grades in **31 seconds** (~8s per free-text item; MCQ
and numeric items are graded in code and cost nothing). Worst case, an all-
constructed check runs a little over a minute. Against 20-25 minutes of
reading per chunk, the boundary wait is not the bottleneck.

### Two app-breaking bugs the simulator found in ten minutes

Running the app in the iOS Simulator (no cable, no unlock dance) immediately
surfaced two failures that neither the server tests nor the browser proxy could
see, because both live in how Xcode packages the app:

- **KaTeX was flattened into the bundle root.** `chapter.html` and the new
  check-screen renderer both ask for `katex/katex.min.css` and
  `katex/contrib/auto-render.min.js`; those paths did not exist, so **no math
  would have rendered anywhere in the app** - the entire point of the book.
  Fixed by making `Resources/katex` a folder reference in `project.yml`.
  Note the device build cached the old flat layout: it took deleting
  DerivedData, after which output moved to `ipad-app/build/`.
- **One numeric answer killed the whole offline book.** Compute items answer
  with a bare number (`answer: 6`); the client typed that field as a string, so
  a single item made the entire 11-chapter bundle fail to decode and the
  built-in fallback silently reported "No bundled book found". Fixed on both
  sides - the server stringifies at the payload boundary, and the client now
  decodes leniently.

Both are verified fixed on screen in the simulator: the offline book loads and
the check prompt renders as typeset math rather than raw `$[2, 0, -3]$`.

**Use the simulator for UI work from now on** - `xcrun simctl` installs and
launches without touching the iPad, and `simctl io <dev> screenshot` gives you
the screen. Launch arguments drive it hands-free: `selftest` runs the full
exchange loop, `showcheck` jumps straight to the most math-heavy check screen.
Caveat: the simulator runs iOS 26 while the mini 4 runs iOS 15.8, so it is
right for layout and logic but not a substitute for a real-device pass, and it
cannot test the USB transport at all.

### The big one: every equation in the book was at risk

Driving the app's real renderer in a browser (`server/test_render.py` writes a
loadable page) surfaced a **systemic math-corruption bug** that had shipped in
the bundle:

- CommonMark treats `\{` and `\}` as markdown escapes, so markdown-it stripped
  the backslashes before KaTeX saw them - producing an unmatched-brace parse
  error rendered as a red error box in u9.
- LaTeX containing `<` (the subscript `x_{<t}`, all over u1's probability
  notation) was swallowed by the browser's HTML parser as an unknown tag,
  silently truncating equations mid-expression.

Both are fixed in `render.py` (math is pulled out before markdown and
HTML-escaped on the way back), with `server/test_math_render.py` as the guard.
Audited across all 11 units afterwards: **2045 equations render, 0 errors**, no
stray LaTeX. The bundle was regenerated and the corrected build is on the iPad.

### Bugs found and fixed overnight

- **Server bound to localhost** in one start path, so the iPad could not reach
  it - this was the actual cause of last night's "Thinking about what you need
  next" hang. `start-flight.sh` was always correct; the test harness was not,
  and is now fixed to bind `0.0.0.0`.
- **Silent 15-minute spinner**: the app's USB wait had a 900s timeout with no
  error surfaced. Now 150s and it reports failure visibly.
- **State dir was not self-healing**: if `state/` disappeared after boot, every
  write crashed with a 500. Now re-created on demand.
- **USB bridge hid error causes**: it logged bare status codes. Now logs
  response bodies, so a 404/422 explains itself.

### Known, not fixed (deliberate)

- **Sprite grading is slow**: ~55s per item because each grade is a separate
  `claude -p` invocation with full harness startup, so a 9-item check takes
  ~8 min. Fix is to batch all items into one call, but that touches
  flight-path grading code and was not worth the risk the night before. Local
  Qwen grading is ~5s/item and is what tomorrow uses.
- **One unexplained 404** in an early bridge log. The shape is reproducible
  (an unknown `subject` returns 404) but the actual trigger was not confirmed,
  and the stale-app theory was disproven (only `com.mjbraun.chiron` is
  installed). Bridge logging now captures the body if it recurs.
- **iPad UI automation** was attempted via WebDriverAgent and abandoned - it
  needs a host-app scheme. The self-test launch argument replaces it and
  exercises the real protocol.

## At the gate / on the plane

0. **`scripts/preflight-check.sh`** - read-only GO/NO-GO on the whole stack.
   Run it after start-flight.sh and again after switching to airplane mode. It
   specifically catches the localhost-bind failure that looks like an app hang.
1. Mac: run `scripts/start-flight.sh` (asks for sudo twice: GPU memory cap + disablesleep).
2. Wi-Fi path: System Settings > General > Sharing > Internet Sharing ON
   (share from "AdHoc" to Wi-Fi). iPad: airplane mode ON, then Wi-Fi back ON,
   join the network. App server field: `http://192.168.2.1:8080`.
3. Open Chiron on the iPad, pick "How AI Works", hit Begin/Continue. Read. The
   Mac can sit lid-closed in the seat pocket; it only matters at chapter
   boundaries.

## If Wi-Fi sharing dies mid-flight (Tahoe risk)

USB path: plug the USB-C cable Mac <-> iPad, then on the Mac:
```
cd ~/dev/mjbraun/studies/dynamic-book/server && .venv/bin/python usb_bridge.py
```
In the app, just submit the check as normal - the request parks in the outbox
and the bridge delivers it (badge shows "usb"). Cable only needs to be in at
chapter boundaries.

## If the Mac is unusable entirely

App start screen > "No server? Read the built-in book" - the full default-path
book with self-grading checks, no adaptivity.

## If there IS flight Wi-Fi

Buy it, connect the Mac, and hodar becomes the second upstream automatically
(once enabled in `server/config.yaml` - set url/model and `enabled: true`).

## Battery notes

- 60W+ USB-C battery bank in the bag, cable rated for it.
- Mac: Low Power Mode, screen brightness min (or lid closed - start-flight.sh
  sets disablesleep).
- Expect fans during generation bursts; idle between boundaries is cheap.

## After landing - do this before you close the laptop

Export the spaced-review schedule while the server still has your session state:

```
curl -s http://127.0.0.1:8080/review-schedule > ~/chiron-review.md
```

It is self-contained (every prompt with its reference answer, plus the
misconceptions you actually leaned on and why they fail), so it works with no
server and no network. Day 1 / day 3 / day 10 - this is the part that decides
whether the eight hours survive the month. Optionally push it to the Remarkable
with the `remarkable` skill.

Then `scripts/stop-flight.sh` and turn Internet Sharing off in System Settings.

## Emergency debugging

- book-server log: `tail -f /tmp/book-server.log`
- LM Studio alive? `curl http://127.0.0.1:1234/v1/models`
- book-server alive? `curl http://127.0.0.1:8080/health`
- iPad can't see Mac? Check the iPad joined the AdHoc network, then
  `ping 192.168.2.1` is not possible from iPad - just reload the app start
  screen (it re-probes every 15s).
- Model infinite-looping (rare Qwen pathology): `lms unload --all && lms load
  qwen/qwen3.6-35b-a3b --context-length 32768 -y` - the server reconnects
  automatically.
