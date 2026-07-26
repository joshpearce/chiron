# Flight-day runbook

## At the gate / on the plane

1. Mac: run `scripts/start-flight.sh` (asks for sudo twice: GPU memory cap + disablesleep).
2. Wi-Fi path: System Settings > General > Sharing > Internet Sharing ON
   (share from "AdHoc" to Wi-Fi). iPad: airplane mode ON, then Wi-Fi back ON,
   join the network. App server field: `http://192.168.2.1:8080`.
3. Open DynamicBook on the iPad, hit Begin/Continue. Read. The Mac can sit
   lid-closed in the seat pocket; it only matters at chapter boundaries.

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
