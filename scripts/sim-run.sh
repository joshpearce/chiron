#!/bin/bash
# Build the app, install it in the chiron-ipad simulator, and launch it
# pointed at the local server. The CHIRON_SERVER launch-environment override
# means no saved-server state is involved - every run talks to localhost:8080
# (the simulator shares the Mac's loopback).
#
#   ./scripts/sim-run.sh              build + install + launch
#   ./scripts/sim-run.sh shot [name]  screenshot to /tmp/<name>.png (default: sim)
set -euo pipefail

UDID=BFAE8038-7CDE-4B67-90F3-82F15695D845   # chiron-ipad (iPad, iOS 26.5)
BUNDLE=dev.mjbraun.chiron
SCRIPTS="$(cd "$(dirname "$0")" && pwd)"
APPDIR="$SCRIPTS/../ipad-app"
# Default to the isolated dev server so sim runs never touch real learner
# state on :8080. Override with CHIRON_SERVER to point elsewhere.
SERVER="${CHIRON_SERVER:-http://localhost:8082}"

if [ "${1:-}" = "shot" ]; then
  out="/tmp/${2:-sim}.png"
  xcrun simctl io "$UDID" screenshot "$out" >/dev/null 2>&1
  echo "$out"
  exit 0
fi

if [ "$SERVER" = "http://localhost:8082" ]; then
  "$SCRIPTS/sim-server.sh" start
fi

xcrun simctl bootstatus "$UDID" -b >/dev/null
open -a Simulator

cd "$APPDIR"
xcodebuild -project Chiron.xcodeproj -scheme Chiron \
  -destination "platform=iOS Simulator,id=$UDID" \
  -derivedDataPath build-sim build -quiet

xcrun simctl terminate "$UDID" "$BUNDLE" 2>/dev/null || true
xcrun simctl install "$UDID" build-sim/Build/Products/Debug-iphonesimulator/Chiron.app
SIMCTL_CHILD_CHIRON_SERVER="$SERVER" xcrun simctl launch "$UDID" "$BUNDLE"
