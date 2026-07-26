#!/bin/bash
# Build chiron-server for the sprite and swap the running service onto it.
#
# The sprite is linux/amd64 and the binary is static (CGO off), so nothing but
# the binary itself has to be copied - no venv, no interpreter, no wheels.
#
# The shared client key lives only in the service environment, and recreating a
# service is the only way to change its environment, so the key is read back and
# re-supplied here. It is never printed and never written to disk.
#
#   scripts/sprite-deploy.sh              # build, copy, restart, verify
#   scripts/sprite-deploy.sh --corpus     # also sync corpus/ and config.yaml
set -euo pipefail
cd "$(dirname "$0")/.."

SYNC_CORPUS=false
[ "${1:-}" = "--corpus" ] && SYNC_CORPUS=true

echo "==> building for linux/amd64"
(cd server-go && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" \
  -o /tmp/chiron-server-linux ./cmd/chiron-server)
ls -lh /tmp/chiron-server-linux | awk '{print "    " $5}'

echo "==> running tests before shipping"
(cd server-go && go test ./... >/dev/null) && echo "    all packages pass"

echo "==> copying binary"
# The running binary cannot be overwritten in place, so write beside it and move.
tar czf - -C /tmp chiron-server-linux | sprite -s chiron exec -- bash -c '
  cd /home/sprite/chiron && mkdir -p bin && tar xzf - &&
  mv chiron-server-linux bin/chiron-server.new && chmod +x bin/chiron-server.new' \
  2>&1 | grep -v LIBARCHIVE || true

if $SYNC_CORPUS; then
  echo "==> syncing corpus and config"
  # COPYFILE_DISABLE stops macOS writing AppleDouble ._* files, which the
  # corpus parser chokes on with a UnicodeDecodeError.
  COPYFILE_DISABLE=1 tar czf - corpus server/config.yaml \
    | sprite -s chiron exec -- bash -c 'cd /home/sprite/chiron && tar xzf -' \
    2>&1 | grep -v LIBARCHIVE || true
fi

echo "==> swapping the service"
sprite -s chiron exec -- bash -c '
  set -e
  K=$(sprite-env services list 2>/dev/null | python3 -c "
import sys, json
for line in sys.stdin:
    line = line.strip()
    if line.startswith(\"[\"):
        for s in json.loads(line):
            if s[\"name\"] == \"chiron-server\":
                print(s.get(\"env\", {}).get(\"CHIRON_AUTH_TOKEN\", \"\"), end=\"\")
")
  [ -n "$K" ] || { echo "FATAL: no CHIRON_AUTH_TOKEN on the existing service" >&2; exit 1; }
  sprite-env services stop chiron-server 2>/dev/null || true
  mv /home/sprite/chiron/bin/chiron-server.new /home/sprite/chiron/bin/chiron-server
  sprite-env services delete chiron-server 2>/dev/null || true
  sprite-env services create chiron-server \
    --cmd /home/sprite/chiron/bin/chiron-server \
    --args "-addr,0.0.0.0:8080,-config,/home/sprite/chiron/server/config.yaml" \
    --env "CHIRON_AUTH_TOKEN=$K" \
    --dir /home/sprite/chiron/server
  sleep 4
' 2>&1 | grep -Ev '"type":"(stdout|stderr)"' | tail -2

echo "==> verifying from outside"
K=$(scripts/sprite-get-auth.sh)
printf '    /ping            %s\n' "$(curl -s -o /dev/null -w '%{http_code}' -m 20 https://chiron.example/ping)"
printf '    /health no token %s\n' "$(curl -s -o /dev/null -w '%{http_code}' -m 20 https://chiron.example/health)"
printf '    /health authed   %s\n' "$(curl -s -o /dev/null -w '%{http_code}' -m 20 -H "Authorization: Bearer $K" https://chiron.example/health)"
curl -s -m 20 -H "Authorization: Bearer $K" https://chiron.example/subjects | sed 's/^/    /'
echo
echo "Expect 200 / 401 / 200. Anything else means the swap did not take."
