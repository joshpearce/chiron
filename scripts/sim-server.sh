#!/bin/bash
# Ensure an isolated dev server for simulator work. Same corpus and LM
# Studio as the real server, but state lives under /tmp/chiron-sim-state-<port>
# so simulator exchanges never touch the real learner state on :8080.
#
#   ./scripts/sim-server.sh                start if not already running
#   ./scripts/sim-server.sh reset          wipe the throwaway state and restart
#   ./scripts/sim-server.sh stop
#   ./scripts/sim-server.sh start 8085     a second server, its own state:
#                                          what parallel test workers need,
#                                          one server each
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PORT="${2:-8082}"
STATE=/tmp/chiron-sim-state-$PORT
CONF="$STATE/config.yaml"
LOG="$STATE/server.log"
PIDFILE="$STATE/server.pid"
ADDR=":$PORT"

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
  start)
    if running; then echo "already running on $ADDR"; exit 0; fi
    # A server someone else started on this port serves just as well.
    if curl -s -m 1 "http://localhost:$PORT/ping" >/dev/null 2>&1; then
      echo "a server already answers on $ADDR"; exit 0
    fi ;;
  *) echo "usage: $0 [start|reset|stop] [port]" >&2; exit 1 ;;
esac

mkdir -p "$STATE/state/ai" "$STATE/state/data"

# A server with nothing in its state cannot open a book without a model
# behind it: no cached chapter, no learner past the screener. A new one
# starts where an existing sim server is, so a test that expects a book in
# progress works on a worker's own server as it does on the first.
if [ ! -d "$STATE/state/ai/chapters" ]; then
  # No other state at all (a fresh /tmp) is not a failure: the pipeline's
  # empty ls must not take the script down with it.
  donor=$( (ls -td /tmp/chiron-sim-state*/state 2>/dev/null || true) | grep -v "^$STATE/" | while read -r d; do
    [ -d "$d/ai/chapters" ] && echo "$d" && break
  done || true)
  if [ -n "${donor:-}" ]; then
    cp -R "$donor/." "$STATE/state/"
    echo "seeded state from $donor"
  fi
fi

# The real config, with subject state redirected at the throwaway dir.
# Everything else (corpus, static, upstreams, session tuning) should stay
# faithful to server/config.yaml so sim behaviour matches the real thing.
sed -e "s|corpus_dir: ../corpus|corpus_dir: $ROOT/corpus|" \
    -e "s|state_dir: ../state/|state_dir: $STATE/state/|" \
    -e "s|static_dir: static|static_dir: $ROOT/server/static|" \
    -e "s|katex_dir: ../ipad-app|katex_dir: $ROOT/ipad-app|" \
    -e "s|fonts_dir: ../assets|fonts_dir: $ROOT/assets|" \
    "$ROOT/server/config.yaml" > "$CONF"
# Everything the server keeps beside its subjects, under the same
# throwaway dir rather than beside /tmp.
printf 'primers_dir: %s\nreadings_dir: %s\nbuilds_dir: %s\n' "$STATE/state/primers" "$STATE/state/readings" "$STATE/builds" >> "$CONF"

(cd "$ROOT/server-go" && go build -o "$STATE/chiron-server" ./cmd/chiron-server)

# Drive/test mode: enables the dev-drive command queue and the authored-
# chapter cache - this server exists for UI iteration, never real learning.
CHIRON_DRIVE=1 "$STATE/chiron-server" -addr "$ADDR" -config "$CONF" >>"$LOG" 2>&1 &
echo $! > "$PIDFILE"

for _ in $(seq 1 20); do
  if curl -s -m 1 "http://localhost:$PORT/ping" >/dev/null 2>&1; then
    echo "dev server on $ADDR, state in $STATE"
    exit 0
  fi
  sleep 0.5
done
echo "server did not come up - see $LOG" >&2
exit 1
