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

## After landing

`scripts/stop-flight.sh`, Internet Sharing OFF in System Settings.
Export the spaced-review schedule: `curl http://127.0.0.1:8080/state | ...`
(day 1 / day 3 / day 10 review of missed items - do these or the 8 hours decay).

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
