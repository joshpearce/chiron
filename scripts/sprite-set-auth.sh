#!/bin/bash
# Install the shared client key into the chiron sprite's service environment.
# The key is read from stdin so it never lands in a file or in shell history:
#
#   op read 'op://<vault>/<item>/credential' --account flyio | scripts/sprite-set-auth.sh
#
# It goes in the service env rather than config.yaml so the secret is not
# sitting in a file inside the sprite either. Recreating the service is the
# only way to change its environment, hence the delete/create - which is also
# why the command below has to stay in step with sprite-deploy.sh. It did not
# once, and rotating the key silently reverted the sprite from the Go binary
# back to uvicorn, with health checks passing either way.
set -euo pipefail
KEY=$(cat)
[ -n "$KEY" ] || { echo "no key on stdin" >&2; exit 1; }

. "$(dirname "$0")/sprite-service.sh"
sprite -s "$SPRITE" exec -- bash -c "$(sprite_service_script "$KEY")" \
  2>&1 | grep -Ev '"type":"(stdout|stderr|started|stopping|stopped|complete)"'
