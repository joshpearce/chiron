#!/bin/bash
# Print the chiron sprite's current shared client key on stdout.
#
# The key exists only in the sprite's service environment - it was generated
# with `tr -dc A-Za-z0-9 </dev/urandom | head -c 48` and piped straight in, so
# it was never written to disk on the Mac. Use this to read it back once, to
# store in 1Password and to type into the app's "shared key" field:
#
#   scripts/sprite-get-auth.sh | python3 -c 'import json,sys; print(json.dumps({
#       "title":"chiron-sprite","category":"PASSWORD","fields":[{"id":"password",
#       "type":"CONCEALED","purpose":"PASSWORD","label":"password",
#       "value":sys.stdin.read().strip()}]}))' \
#     | op item create --account my --vault Private
#
# (op item create reads stdin as a JSON item template; "password=-" does not
# read stdin. The item lives in the personal account, vault Private, and
# ~/.config/chiron-dev/config refers to it as
# op://<vault>/<item>/password.)
#
# To rotate: pipe a fresh key through scripts/sprite-set-auth.sh, then update
# the app. Old keys stop working the moment the service restarts.
set -euo pipefail
. "$(dirname "$0")/sprite-service.sh"
sprite_current_key
