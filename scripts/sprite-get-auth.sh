#!/bin/bash
# Print the chiron sprite's current shared client key on stdout.
#
# The key exists only in the sprite's service environment - it was generated
# with `tr -dc A-Za-z0-9 </dev/urandom | head -c 48` and piped straight in, so
# it was never written to disk on the Mac. Use this to read it back once, to
# store in 1Password and to type into the app's "shared key" field:
#
#   scripts/sprite-get-auth.sh | op item create --category=password \
#       --title=chiron-sprite --account flyio password=-
#
# To rotate: pipe a fresh key through scripts/sprite-set-auth.sh, then update
# the app. Old keys stop working the moment the service restarts.
set -euo pipefail
sprite -s chiron exec -- bash -c "sprite-env services list" 2>/dev/null \
  | python3 -c '
import sys, json
for line in sys.stdin:
    line = line.strip()
    if not line.startswith("["):
        continue
    for svc in json.loads(line):
        if svc.get("name") == "chiron-server":
            print(svc.get("env", {}).get("CHIRON_AUTH_TOKEN", ""), end="")
            sys.exit(0)
sys.exit("chiron-server service not found")
'
