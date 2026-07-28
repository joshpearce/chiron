#!/bin/bash
# GO / NO-GO for the flight stack. Read-only - starts nothing, changes nothing.
# Run at the gate after start-flight.sh, and again after switching to airplane mode.
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PASS=0; FAIL=0
ok()   { printf "  \033[32mOK\033[0m   %s\n" "$1"; PASS=$((PASS+1)); }
bad()  { printf "  \033[31mNO\033[0m   %s\n" "$1"; FAIL=$((FAIL+1)); }
warn() { printf "  --   %s\n" "$1"; }

echo "== Chiron preflight =="

# 1. Inference engine
if curl -s --max-time 3 http://127.0.0.1:1234/v1/models >/dev/null 2>&1; then
  MODEL=$(curl -s --max-time 3 http://127.0.0.1:1234/v1/models \
    | python3 -c "import json,sys; print(json.load(sys.stdin)['data'][0]['id'])" 2>/dev/null)
  ok "LM Studio up (first model: ${MODEL:-unknown})"
else
  bad "LM Studio not answering on :1234 - run 'lms server start' and load the model"
fi

# 2. Book server, and crucially that it is reachable off-box
if curl -s --max-time 5 http://127.0.0.1:8080/health >/dev/null 2>&1; then
  ok "book-server up on localhost"
else
  bad "book-server not answering on :8080 - run scripts/start-flight.sh"
fi

if lsof -nP -i :8080 2>/dev/null | grep -q "TCP \*:8080"; then
  ok "book-server bound to all interfaces (iPad can reach it)"
else
  bad "book-server bound to localhost ONLY - the iPad will hang. Restart with -addr 0.0.0.0:8080"
fi

# 3. The address the iPad is configured to use.
#
# The bridge IP existing is NOT sufficient: Internet Sharing can be on and
# sharing to a wired interface with Wi-Fi sharing off, which leaves 192.168.2.1
# answering perfectly from the Mac while the iPad has no way to reach it. That
# is the same shape as the localhost-bind failure - green here, dead in the
# seat - so the AirPort share flag is checked explicitly.
if ifconfig bridge100 2>/dev/null | grep -q "inet 192.168.2.1"; then
  ok "Internet Sharing up - Mac is 192.168.2.1"
  if curl -s --max-time 5 http://192.168.2.1:8080/health >/dev/null 2>&1; then
    ok "server reachable at http://192.168.2.1:8080 (the app's server field)"
  else
    bad "192.168.2.1:8080 not answering - check the bind above"
  fi

  # Read the flag out of the plist rather than scraping `defaults` output.
  AIRPORT=$(python3 - <<'PYEOF' 2>/dev/null
import plistlib
try:
    with open("/Library/Preferences/SystemConfiguration/com.apple.nat.plist", "rb") as f:
        nat = plistlib.load(f)["NAT"]
    ap = nat.get("AirPort", {})
    sd = nat.get("SharingDevices", [])
    ap_on = 1 if 'en0' in sd else 0
    print(f'{ap_on}|{ap.get("NetworkName", "")}')

except Exception:
    print("?|")
PYEOF
)
  AIRPORT_ON=${AIRPORT%%|*}
  SSID=${AIRPORT#*|}
  if [ "$AIRPORT_ON" = "1" ]; then
    ok "Wi-Fi sharing ON - iPad joins \"${SSID:-the shared network}\""
  elif [ "$AIRPORT_ON" = "?" ]; then
    warn "could not read the Internet Sharing config - verify Wi-Fi sharing by hand"
  else
    bad "Wi-Fi sharing OFF - the iPad cannot reach the Mac even though 192.168.2.1 answers here.
       System Settings > General > Sharing > Internet Sharing, tick Wi-Fi
       under the share-to list, then re-run this check."
  fi

  LEASES=$(grep -c ip_address /var/db/dhcpd_leases 2>/dev/null || echo 0)
  if [ "$LEASES" -gt 0 ]; then
    ok "$LEASES device(s) have taken a lease on the shared network"
  else
    warn "no device has joined the shared network yet - the iPad still needs to connect"
  fi
else
  warn "Internet Sharing off - Wi-Fi path unavailable (USB bridge or built-in book only)"
fi

# 4. Model actually generates (the slow, real check)
HEALTH=$(curl -s --max-time 5 http://127.0.0.1:8080/health 2>/dev/null)
if echo "$HEALTH" | grep -q '"connected": *true'; then
  ok "server reports its LLM upstream connected"
else
  bad "server cannot reach its LLM upstream"
fi

# 5. Offline fallback present
BUNDLE="$ROOT/ipad-app/Chiron/Resources/default-book.json"
if [ -f "$BUNDLE" ]; then
  CH=$(python3 -c "import json;print(len(json.load(open('$BUNDLE'))['chapters']))" 2>/dev/null)
  ok "built-in book present (${CH:-?} chapters) - last-resort path if all else fails"
else
  bad "built-in book missing - no fallback if the Mac is unreachable"
fi

# 6. Power
if pmset -g batt 2>/dev/null | grep -q "AC Power"; then
  ok "Mac on AC power"
else
  PCT=$(pmset -g batt 2>/dev/null | grep -o "[0-9]*%" | head -1)
  warn "Mac on battery (${PCT:-?}) - inference is the heaviest thing it will do"
fi

echo
if [ $FAIL -eq 0 ]; then
  echo "GO - $PASS checks passed."
else
  echo "NO-GO - $FAIL problem(s), $PASS ok. Fix the NO lines above."
fi
exit $FAIL
