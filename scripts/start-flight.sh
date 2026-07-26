#!/bin/bash
# One command at the gate: memory cap, model, server. Ends with a GO/NO-GO.
# Run from anywhere:  ~/dev/mjbraun/studies/chiron/scripts/start-flight.sh
set -e
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
MODEL="qwen/qwen3.6-35b-a3b"
CTX=32768

echo "== 1/5 GPU memory cap (needs sudo, resets every reboot) =="
sudo sysctl iogpu.wired_limit_mb=25600

echo "== 2/5 keep Mac awake lid-closed (undo: sudo pmset -a disablesleep 0) =="
sudo pmset -a disablesleep 1

echo "== 3/5 load model at ${CTX} context =="
# A model already loaded at a different context length is the failure that
# looks like success: `lms load` errors, the guard swallows it, and the flight
# runs at whatever was loaded before. Check, and only reload if needed.
# Column positions in `lms ps` are not stable (SIZE is "20.43 GB", two fields),
# so pick the first bare integer >= 1024 on the model's row - that is CONTEXT.
CURRENT=$(lms ps 2>/dev/null | awk -v m="$MODEL" '$1 == m {
    for (i = 1; i <= NF; i++) if ($i ~ /^[0-9]+$/ && $i + 0 >= 1024) { print $i; exit }
  }')
if [ -n "$CURRENT" ] && [ "$CURRENT" -ge "$CTX" ] 2>/dev/null; then
  echo "   already loaded with ${CURRENT} context - keeping it"
else
  [ -n "$CURRENT" ] && { echo "   loaded at ${CURRENT}, reloading"; lms unload --all >/dev/null 2>&1 || true; }
  lms load "$MODEL" --context-length "$CTX" -y
fi

echo "== 4/5 LM Studio API server =="
lms server start || true

echo "== 5/5 book-server on :8080 (all interfaces, so the iPad can reach it) =="
# Binding all interfaces is not optional: a localhost bind leaves the iPad
# unable to reach the Mac, which presents as the app hanging on "Thinking about
# what you need next" rather than as a server misconfiguration.
cd "$ROOT/server"
pkill -f "chiron-server -addr" 2>/dev/null || true
pkill -f "uvicorn main:app" 2>/dev/null || true
sleep 1
if [ ! -x "$ROOT/bin/chiron-server" ]; then
  echo "  building chiron-server"
  (cd "$ROOT/server-go" && go build -o "$ROOT/bin/chiron-server" ./cmd/chiron-server)
fi
nohup "$ROOT/bin/chiron-server" -addr 0.0.0.0:8080 -config "$ROOT/server/config.yaml" \
  >> /tmp/book-server.log 2>&1 &
sleep 3

echo
"$ROOT/scripts/preflight-check.sh" || true

cat <<EOF

Next:
  Wi-Fi path : System Settings > General > Sharing > Internet Sharing ON
               (share from "AdHoc" to Wi-Fi). iPad: airplane mode, then Wi-Fi
               back on, join the network. App server field: http://192.168.2.1:8080
  USB path   : plug the cable, then run
               $ROOT/server/.venv/bin/python $ROOT/server/usb_bridge.py
  No Mac     : app start screen > "No server? Read the built-in book"
  Landing    : curl -s http://127.0.0.1:8080/review-schedule > ~/chiron-review.md
  Teardown   : $ROOT/scripts/stop-flight.sh
EOF
