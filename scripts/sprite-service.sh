#!/bin/bash
# The chiron-server service definition, in one place.
#
# Both sprite-deploy.sh and sprite-set-auth.sh have to recreate the service -
# recreating is the only way to change its environment - and for a while each
# carried its own copy of the command line. They drifted: rotating the key
# reinstated uvicorn over the Go binary, and every health check passed either
# way, so nothing said so. One definition, sourced by both.
#
# Usage:  sprite_recreate_service "$KEY"
# Runs inside the sprite via `sprite exec`; $KEY is interpolated on the Mac.

sprite_service_script() {
  local key="$1"
  cat <<EOF
set -e
sprite-env services stop chiron-server 2>/dev/null || true
sprite-env services delete chiron-server 2>/dev/null || true
sprite-env services create chiron-server \\
  --cmd /home/sprite/chiron/bin/chiron-server \\
  --args '-addr,0.0.0.0:8080,-config,/home/sprite/chiron/server/config.yaml' \\
  --env 'CHIRON_AUTH_TOKEN=${key}' \\
  --dir /home/sprite/chiron/server
sleep 4
echo -n 'unauthenticated /health -> '; curl -s -o /dev/null -w '%{http_code}\\n' http://127.0.0.1:8080/health
echo -n 'unauthenticated /ping   -> '; curl -s -o /dev/null -w '%{http_code}\\n' http://127.0.0.1:8080/ping
echo -n 'authenticated  /health  -> '; curl -s -o /dev/null -w '%{http_code}\\n' -H 'Authorization: Bearer ${key}' http://127.0.0.1:8080/health
EOF
}

# Read the key currently installed on the service, without printing it.
sprite_current_key() {
  sprite -s chiron exec -- bash -c "sprite-env services list" 2>/dev/null \
    | python3 -c '
import sys, json
for line in sys.stdin:
    line = line.strip()
    if line.startswith("["):
        for s in json.loads(line):
            if s.get("name") == "chiron-server":
                print(s.get("env", {}).get("CHIRON_AUTH_TOKEN", ""), end="")
                sys.exit(0)
sys.exit("chiron-server service not found")
'
}
