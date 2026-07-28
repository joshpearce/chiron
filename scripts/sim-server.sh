#!/bin/bash
# Ensure an isolated dev server for simulator work on :8082. Same corpus and
# LM Studio as the real server, but state lives in /tmp/chiron-sim-state so
# simulator exchanges never touch the real learner state on :8080.
#
#   ./scripts/sim-server.sh          start if not already running
#   ./scripts/sim-server.sh reset    wipe the throwaway state and restart
#   ./scripts/sim-server.sh stop
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
STATE=/tmp/chiron-sim-state
CONF="$STATE/config.yaml"
LOG="$STATE/server.log"
PIDFILE="$STATE/server.pid"
ADDR=:8082

running() {
  [ -f "$PIDFILE" ] && kill -0 "$(cat "$PIDFILE")" 2>/dev/null
}

stop() {
  if running; then kill "$(cat "$PIDFILE")"; fi
  rm -f "$PIDFILE"
}

case "${1:-start}" in
  stop)  stop; echo "stopped"; exit 0 ;;
  reset) stop; rm -rf "$STATE" ;;
  start) if running; then echo "already running on $ADDR"; exit 0; fi ;;
  *) echo "usage: $0 [start|reset|stop]" >&2; exit 1 ;;
esac

mkdir -p "$STATE/state/ai"

# The real config, with subject state redirected at the throwaway dir.
# Everything else (corpus, static, upstreams, session tuning) should stay
# faithful to server/config.yaml so sim behaviour matches the real thing.
sed -e "s|corpus_dir: ../corpus|corpus_dir: $ROOT/corpus|" \
    -e "s|state_dir: ../state/ai|state_dir: $STATE/state/ai|" \
    -e "s|static_dir: static|static_dir: $ROOT/server/static|" \
    "$ROOT/server/config.yaml" > "$CONF"

(cd "$ROOT/server-go" && go build -o "$STATE/chiron-server" ./cmd/chiron-server)

"$STATE/chiron-server" -addr "$ADDR" -config "$CONF" >>"$LOG" 2>&1 &
echo $! > "$PIDFILE"

for _ in $(seq 1 20); do
  if curl -s -m 1 "http://localhost:8082/ping" >/dev/null 2>&1; then
    echo "dev server on $ADDR, state in $STATE"
    exit 0
  fi
  sleep 0.5
done
echo "server did not come up - see $LOG" >&2
exit 1
