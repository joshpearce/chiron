#!/bin/bash
# Walk every screen of the iPad app in the Simulator and screenshot each,
# driven through the app's debug harness (launch argument `harness`, HTTP on
# localhost:8087). The harness output is asserted; the screenshots are for
# review by eye.
#
#   ./scripts/sim-verify.sh                 both devices, light and dark, both text sizes
#   DEVICES="chiron-ipad-15" THEMES=light SIZES=default ./scripts/sim-verify.sh
#   SUBJECT=data ./scripts/sim-verify.sh    walk the other book
#
# Output: $OUT/<device>/<theme>-<size>/<nn>-<screen>.png (default
# ipad-app/verify/, gitignored). Needs a throwaway server with both subjects
# on CHIRON_SERVER (default :8084); never point it at :8080.
set -euo pipefail

SCRIPTS="$(cd "$(dirname "$0")" && pwd)"
ROOT="$SCRIPTS/.."
BUNDLE=dev.mjbraun.chiron
SERVER="${CHIRON_SERVER:-http://localhost:8084}"
SUBJECT="${SUBJECT:-ai}"
LEVEL="${LEVEL:-2}"
OUT="${OUT:-$ROOT/ipad-app/verify}"
DEVICES="${DEVICES:-chiron-ipad chiron-ipad-15}"
THEMES="${THEMES:-light dark}"
SIZES="${SIZES:-default accessibility-extra-large}"
PORT=8087
H="http://localhost:$PORT"

# The server must be fresh for the subject so the walk starts at placement.
reset_subject() {
  curl -sf -o /dev/null -X POST -H 'Content-Type: application/json' \
    -d "{\"subject\":\"$SUBJECT\",\"confirm\":true}" "$SERVER/reset"
}

udid_of() {
  xcrun simctl list devices | grep -E "^\s+$1 \(" | head -1 | sed -E 's/.*\(([0-9A-F-]{36})\).*/\1/'
}

# state: print the harness state, one line.
state() { curl -sf "$H/state"; echo; }

# expect <screen>: assert the harness reports that screen.
expect() {
  local got
  got=$(curl -sf "$H/state" | python3 -c 'import json,sys; print(json.load(sys.stdin).get("screen",""))')
  if [ "$got" != "$1" ]; then
    echo "expected screen $1, got $got" >&2
    curl -sf "$H/state" >&2; echo >&2
    exit 1
  fi
}

# cmd <path> [json]: a harness POST.
cmd() {
  curl -sf -X POST -H 'Content-Type: application/json' -d "${2:-{\}}" "$H$1" >/dev/null
}

wait_for_harness() {
  for _ in $(seq 1 60); do
    curl -sf "$H/state" >/dev/null 2>&1 && return 0
    sleep 0.5
  done
  echo "harness never came up on $PORT" >&2
  exit 1
}

# settle: SwiftUI and the web views need a beat after a state change before
# a screenshot shows the new screen.
settle() { sleep "${1:-1.5}"; }

shot() {
  settle "${3:-1.5}"
  xcrun simctl io "$UDID" screenshot "$DIR/$1-$2.png" >/dev/null 2>&1
  echo "  $1-$2.png  $(state)"
}

for DEVICE in $DEVICES; do
  UDID=$(udid_of "$DEVICE")
  [ -n "$UDID" ] || { echo "no simulator named $DEVICE" >&2; exit 1; }
  echo "== $DEVICE ($UDID)"
  xcrun simctl bootstatus "$UDID" -b >/dev/null
  (cd "$ROOT/ipad-app" && xcodebuild -project Chiron.xcodeproj -scheme Chiron \
    -destination "platform=iOS Simulator,id=$UDID" -derivedDataPath build-sim build -quiet 2>&1 \
    | grep -E "error:" || true)
  APP="$ROOT/ipad-app/build-sim/Build/Products/Debug-iphonesimulator/Chiron.app"
  xcrun simctl terminate "$UDID" "$BUNDLE" 2>/dev/null || true
  if ! xcrun simctl install "$UDID" "$APP" 2>/dev/null; then
    xcrun simctl uninstall "$UDID" "$BUNDLE" 2>/dev/null || true
    xcrun simctl install "$UDID" "$APP"
  fi

  for THEME in $THEMES; do
    for SIZE in $SIZES; do
      DIR="$OUT/$DEVICE/$THEME-$SIZE"
      mkdir -p "$DIR"
      rm -f "$DIR"/*.png
      echo "-- $THEME $SIZE -> $DIR"
      xcrun simctl ui "$UDID" appearance "$THEME" >/dev/null 2>&1 || true
      xcrun simctl ui "$UDID" content_size "$SIZE" >/dev/null 2>&1 || true
      reset_subject
      xcrun simctl terminate "$UDID" "$BUNDLE" 2>/dev/null || true
      SIMCTL_CHILD_CHIRON_SERVER="$SERVER" xcrun simctl launch "$UDID" "$BUNDLE" harness "harness_port=$PORT" >/dev/null
      wait_for_harness
      # The app reopens on the active book; the shelf is the screen behind it.
      cmd /shelf
      expect bookshelf
      shot 01 bookshelf
      cmd /open "{\"subject\":\"$SUBJECT\"}"
      expect placement
      shot 02 placement
      cmd /place "{\"level\":$LEVEL}"
      expect series
      shot 03 series
      cmd /answer '{"mode":"correct"}'
      expect results
      shot 04 results-calibration 2.5
      cmd /proceed
      expect reading
      shot 05 reading 3
      cmd /check
      expect check
      shot 06 check
      cmd /answer '{"mode":"wrong"}'
      expect results
      shot 07 results-below-gate 2.5
      cmd /proceed
      # A long chunk earns a break suggestion; a short one goes straight on.
      screen=$(curl -sf "$H/state" | python3 -c 'import json,sys; print(json.load(sys.stdin)["screen"])')
      if [ "$screen" = "break" ]; then
        shot 08 break
        cmd /proceed
      fi
      expect reading
      shot 09 remediation 3
      xcrun simctl terminate "$UDID" "$BUNDLE" 2>/dev/null || true
    done
  done
  xcrun simctl ui "$UDID" appearance light >/dev/null 2>&1 || true
  xcrun simctl ui "$UDID" content_size default >/dev/null 2>&1 || true
done
echo "screenshots in $OUT"
