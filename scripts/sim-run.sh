#!/bin/bash
# Build the app, install it in a simulator, and launch it pointed at a dev
# server. The CHIRON_SERVER launch-environment override means no saved-server
# state is involved (the simulator shares the Mac's loopback).
#
#   ./scripts/sim-run.sh [app args...]      build + install + launch
#   ./scripts/sim-run.sh selftest [args]    launch the self-test, wait for its
#                                           verdict, exit 0 on PASS
#                                           (args: level=1..5 subject=<id> selftestweak)
#   ./scripts/sim-run.sh test               run the ChironTests unit tests
#   ./scripts/sim-run.sh shot [name]        screenshot to /tmp/<name>.png
#
# DEVICE picks the simulator by name: chiron-ipad (iPad A16, current iPadOS)
# or chiron-ipad-15 (iPad mini 4 shape, iOS 15.5 - the gate for the real
# device). CHIRON_SERVER picks the server; never point it at :8080 (real
# learner state). The default :8082 is the isolated sim server.
set -euo pipefail

DEVICE="${DEVICE:-chiron-ipad}"
BUNDLE=dev.mjbraun.chiron
SCRIPTS="$(cd "$(dirname "$0")" && pwd)"
APPDIR="$SCRIPTS/../ipad-app"
SERVER="${CHIRON_SERVER:-http://localhost:8082}"

UDID=$(xcrun simctl list devices | grep -E "^\s+$DEVICE \(" | head -1 | sed -E 's/.*\(([0-9A-F-]{36})\).*/\1/')
if [ -z "$UDID" ]; then
  echo "no simulator named $DEVICE (xcrun simctl list devices)" >&2
  exit 1
fi

cmd="${1:-}"
case "$cmd" in
  shot)
    out="/tmp/${2:-sim}.png"
    xcrun simctl io "$UDID" screenshot "$out" >/dev/null 2>&1
    echo "$out"
    exit 0 ;;
  test)
    xcrun simctl bootstatus "$UDID" -b >/dev/null
    cd "$APPDIR"
    xcodebuild -project Chiron.xcodeproj -scheme Chiron \
      -destination "platform=iOS Simulator,id=$UDID" \
      -derivedDataPath "build-sim" test -quiet 2>&1 | grep -vE "^$|objc\[[0-9]+\]: Class" || true
    # xcodebuild's own exit status is lost in the pipe; the result bundle says.
    log=$(ls -td build-sim/Logs/Test/*.xcresult 2>/dev/null | head -1)
    [ -n "$log" ] || { echo "no test result bundle" >&2; exit 1; }
    xcrun xcresulttool get test-results summary --path "$log" 2>/dev/null | grep -E '"(result|totalTestCount|passedTests|failedTests)"' || true
    xcrun xcresulttool get test-results summary --path "$log" 2>/dev/null | grep -q '"result" : "Passed"'
    exit $? ;;
esac

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
APP=build-sim/Build/Products/Debug-iphonesimulator/Chiron.app
if ! xcrun simctl install "$UDID" "$APP" 2>/dev/null; then
  # The iOS 15.5 runtime cannot delta-install over a copy that is already
  # there ("Could not hardlink copy"); a clean install works. The app's
  # Documents go with it, so on that device every run starts from nothing.
  xcrun simctl uninstall "$UDID" "$BUNDLE" 2>/dev/null || true
  xcrun simctl install "$UDID" "$APP"
fi

if [ "$cmd" = "selftest" ]; then
  log="/tmp/chiron-selftest-$DEVICE.log"
  : > "$log"
  SIMCTL_CHILD_CHIRON_SERVER="$SERVER" xcrun simctl launch --console-pty "$UDID" "$BUNDLE" "$@" >"$log" 2>&1 &
  pid=$!
  for _ in $(seq 1 "${SELFTEST_TIMEOUT:-600}"); do
    grep -qE "chiron-selftest (PASS|FAIL)" "$log" && break
    sleep 1
  done
  kill "$pid" 2>/dev/null || true
  xcrun simctl terminate "$UDID" "$BUNDLE" 2>/dev/null || true
  grep "chiron-selftest" "$log" || { echo "no self-test output in $log" >&2; exit 1; }
  grep -q "chiron-selftest PASS" "$log"
  exit $?
fi

SIMCTL_CHILD_CHIRON_SERVER="$SERVER" xcrun simctl launch "$UDID" "$BUNDLE" "$@"
