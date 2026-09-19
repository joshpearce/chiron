#!/bin/bash
# Put the sprite on the tailnet and give it a way to the MacBook. Run
# from the Mac; everything happens over `ssh chiron2`. Safe to run again.
#
#   scripts/sprite-tailscale.sh <macbook-tailnet-name> <macbook-user>
#
# The auth key comes from 1Password at run time (TS_AUTHKEY_REF, default
# op://<vault>/<item>/credential): a tagged, pre-approved
# key, used once; tailscaled keeps the node identity it earns in
# /var/lib/tailscale from then on. Also made here: the sprite's own key
# to the MacBook (~/.ssh/macbook_ed25519) and a `Host macbook` stanza
# that waits a little for the tunnel after the sprite wakes. The public
# key line printed at the end is what mac-runner-bootstrap.sh takes.
set -euo pipefail
MACBOOK="${1:?the name of the MacBook on the tailnet}"
MACUSER="${2:?the user on the MacBook}"
REF="${TS_AUTHKEY_REF:-op://<vault>/<item>/credential}"
SPRITE="${SPRITE:-chiron2}"

echo "==> tailscale on $SPRITE"
ssh "$SPRITE" 'command -v tailscale >/dev/null || curl -fsSL https://tailscale.com/install.sh | sh >/dev/null'
ssh "$SPRITE" 'sprite-env services list 2>/dev/null | grep -q "\"tailscaled\"" || sprite-env services create tailscaled --cmd sudo --args "tailscaled,--state=/var/lib/tailscale/tailscaled.state,--socket=/var/run/tailscale/tailscaled.sock" >/dev/null'
ssh "$SPRITE" 'sprite-env services start tailscaled >/dev/null 2>&1 || true; sleep 2'

if ssh "$SPRITE" 'sudo tailscale status >/dev/null 2>&1'; then
  echo "==> already on the tailnet"
else
  echo "==> joining (the key is read from 1Password and passed on stdin)"
  op read --account my "$REF" | ssh "$SPRITE" 'sudo tailscale up --auth-key="$(cat)" --hostname chiron-sprite --accept-dns=true --ssh=false'
fi
ssh "$SPRITE" 'sudo tailscale status | head -5'

echo "==> the sprite's key to the MacBook"
ssh "$SPRITE" '[ -f ~/.ssh/macbook_ed25519 ] || ssh-keygen -q -t ed25519 -N "" -C chiron-sprite-runner -f ~/.ssh/macbook_ed25519'
ssh "$SPRITE" "python3 - '$MACBOOK' '$MACUSER'" <<'PY'
import os, re, sys
name, user = sys.argv[1], sys.argv[2]
path = os.path.expanduser("~/.ssh/config")
stanza = f"""Host macbook
  HostName {name}
  User {user}
  IdentityFile ~/.ssh/macbook_ed25519
  IdentitiesOnly yes
  # The sprite wakes with the tunnel down for a moment: keep trying.
  ConnectionAttempts 6
  ConnectTimeout 10
  ServerAliveInterval 15
"""
text = open(path).read() if os.path.exists(path) else ""
text = re.sub(r"Host macbook\n(?:  .*\n)*", "", text)
open(path, "w").write((text.rstrip("\n") + "\n\n" if text.strip() else "") + stanza)
os.chmod(path, 0o600)
PY
ssh "$SPRITE" 'git -C ~/src/chiron remote get-url macbook >/dev/null 2>&1 || git -C ~/src/chiron remote add macbook "macbook:src/chiron"'

echo
echo "The MacBook takes this line (mac-runner-bootstrap.sh <line>):"
ssh "$SPRITE" 'cat ~/.ssh/macbook_ed25519.pub'
echo
echo "Then, from the sprite: ssh macbook true; git push macbook main; make app-build"
