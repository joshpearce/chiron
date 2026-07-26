#!/bin/bash
# Undo everything start-flight.sh changed.
# Exports the review schedule FIRST - tearing down the server without it throws
# away the only artifact that makes the session survive the week.
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="$HOME/chiron-review-$(date +%Y-%m-%d).md"
if curl -s --max-time 20 http://127.0.0.1:8080/review-schedule > "$OUT" 2>/dev/null \
   && [ -s "$OUT" ]; then
  echo "review schedule saved: $OUT ($(wc -c < "$OUT" | tr -d ' ') bytes)"
else
  rm -f "$OUT"
  echo "WARNING: could not export the review schedule (server already down?)"
fi

sudo pmset -a disablesleep 0
pkill -f "chiron-server -addr" 2>/dev/null || true
pkill -f "uvicorn main:app" 2>/dev/null || true
pkill -f "usb_bridge.py" 2>/dev/null || true
lms unload --all 2>/dev/null || true
lms server stop 2>/dev/null || true
# Internet Sharing teardown if the GUI toggle is unreachable:
#   sudo defaults write /Library/Preferences/SystemConfiguration/com.apple.nat NAT -dict Enabled -int 0
#   sudo launchctl stop com.apple.InternetSharing
echo "done (GPU memory cap resets on reboot; Internet Sharing off via System Settings)"
