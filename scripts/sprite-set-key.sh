#!/bin/bash
# Inject ANTHROPIC_API_KEY into the chiron sprite's service without the key
# touching disk on this Mac. Usage:
#   op read 'op://<vault>/<item>/credential' --account flyio | scripts/sprite-set-key.sh
# Recommendation: mint a DEDICATED key for the sprite in the Anthropic Console
# (workspace-scoped, spend-limited) rather than reusing a personal key.
set -euo pipefail
KEY=$(cat)
[ -n "$KEY" ] || { echo "no key on stdin" >&2; exit 1; }

sprite -s chiron exec -- bash -c "
  sprite-env services stop chiron-server 2>/dev/null || true
  sprite-env services delete chiron-server 2>/dev/null || true
  sprite-env services create chiron-server \
    --cmd /home/sprite/chiron/server/.venv/bin/uvicorn \
    --args 'main:app,--host,0.0.0.0,--port,8080' \
    --env 'ANTHROPIC_API_KEY=$KEY' \
    --dir /home/sprite/chiron/server
  sleep 4
  curl -s http://127.0.0.1:8080/health
"
echo
echo "key installed; verify llm.connected above"
