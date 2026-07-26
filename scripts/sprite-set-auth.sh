#!/bin/bash
# Install the shared client key into the chiron sprite's service environment.
# The key is read from stdin so it never lands in a file or in shell history:
#
#   op read 'op://<vault>/<item>/credential' --account flyio | scripts/sprite-set-auth.sh
#
# It goes in the service env rather than config.yaml so the secret is not
# sitting in a file inside the sprite either. Recreating the service is the
# only way to change its environment, hence the delete/create.
set -euo pipefail
KEY=$(cat)
[ -n "$KEY" ] || { echo "no key on stdin" >&2; exit 1; }

sprite -s chiron exec -- bash -c "
  sprite-env services stop chiron-server 2>/dev/null || true
  sprite-env services delete chiron-server 2>/dev/null || true
  sprite-env services create chiron-server \
    --cmd /home/sprite/chiron/server/.venv/bin/uvicorn \
    --args 'main:app,--host,0.0.0.0,--port,8080' \
    --env 'CHIRON_AUTH_TOKEN=$KEY' \
    --dir /home/sprite/chiron/server
  sleep 5
  echo -n 'unauthenticated /health -> '; curl -s -o /dev/null -w '%{http_code}\n' http://127.0.0.1:8080/health
  echo -n 'unauthenticated /ping   -> '; curl -s -o /dev/null -w '%{http_code}\n' http://127.0.0.1:8080/ping
  echo -n 'authenticated  /health  -> '; curl -s -o /dev/null -w '%{http_code}\n' -H 'Authorization: Bearer $KEY' http://127.0.0.1:8080/health
"
