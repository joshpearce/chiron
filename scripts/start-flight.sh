#!/bin/bash
# One command at the gate: memory cap, model, server, bridge.
# Run from anywhere:  ~/dev/mjbraun/studies/dynamic-book/scripts/start-flight.sh
set -e
ROOT="$(cd "$(dirname "$0")/.." && pwd)"

echo "== 1/5 GPU memory cap (needs sudo, resets every reboot) =="
sudo sysctl iogpu.wired_limit_mb=25600

echo "== 2/5 keep Mac awake lid-closed (undo: sudo pmset -a disablesleep 0) =="
sudo pmset -a disablesleep 1

echo "== 3/5 load model (32K ctx) =="
lms load qwen/qwen3.6-35b-a3b --context-length 32768 -y || echo "already loaded?"

echo "== 4/5 LM Studio API server =="
lms server start || true

echo "== 5/5 book-server on :8080 =="
cd "$ROOT/server"
pkill -f "uvicorn main:app" 2>/dev/null || true
nohup .venv/bin/uvicorn main:app --host 0.0.0.0 --port 8080 > /tmp/book-server.log 2>&1 &
sleep 3
curl -s http://127.0.0.1:8080/health && echo

cat <<'EOF'

Ready. Now:
  Wi-Fi path : System Settings > Sharing > Internet Sharing ON (AdHoc -> Wi-Fi).
               iPad joins the network, app server = http://192.168.2.1:8080
  USB path   : plug the cable and run:  server/.venv/bin/python server/usb_bridge.py
  Teardown   : scripts/stop-flight.sh
EOF
