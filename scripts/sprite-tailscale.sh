#!/bin/bash
# Put the sprite on the tailnet and give it a way to the MacBook. Run
# from the Mac; everything happens over `ssh chiron2`. Safe to run again.
#
#   scripts/sprite-tailscale.sh <macbook-tailnet-name> <macbook-user>
#
# The MacBook must already be on the tailnet: its address is looked up
# there and written into the sprite's ssh stanza, because the sprite's
# resolv.conf is read-only and MagicDNS cannot take hold.
#
# The auth key comes from 1Password at run time (TS_AUTHKEY_REF, default
# op://<vault>/<tailscale item>/credential): a tagged, pre-approved
# key, used once; tailscaled keeps the node identity it earns in
# /var/lib/tailscale from then on. Also made here: the sprite's own key
# to the MacBook (~/.ssh/macbook_ed25519) and a `Host macbook` stanza
# that waits a little for the tunnel after the sprite wakes. The public
# key line printed at the end is what mac-runner-bootstrap.sh takes.
set -euo pipefail
MACBOOK="${1:?the name of the MacBook on the tailnet}"
MACUSER="${2:?the user on the MacBook}"
REF="${TS_AUTHKEY_REF:?the 1Password reference for a tagged Tailscale auth key, e.g. op://<vault>/<item>/credential}"
SPRITE="${SPRITE:-chiron2}"

echo "==> tailscale on $SPRITE"
ssh "$SPRITE" 'command -v tailscale >/dev/null || curl -fsSL https://tailscale.com/install.sh | sh >/dev/null'
ssh "$SPRITE" 'sprite-env services list 2>/dev/null | grep -q "\"tailscaled\"" || sprite-env services create tailscaled --cmd sudo --args "tailscaled,--state=/var/lib/tailscale/tailscaled.state,--socket=/var/run/tailscale/tailscaled.sock" >/dev/null'
ssh "$SPRITE" 'sprite-env services start tailscaled >/dev/null 2>&1 || true; sleep 2'

if ssh "$SPRITE" 'sudo tailscale status >/dev/null 2>&1'; then
  echo "==> already on the tailnet"
else
  echo "==> joining (the key is read from 1Password and passed on stdin)"
  # No MagicDNS: the sprite's /etc/resolv.conf is read-only, so names
  # are looked up below and the address goes into the ssh stanza.
  op read --account my "$REF" | ssh "$SPRITE" 'sudo tailscale up --auth-key="$(cat)" --hostname chiron-sprite --accept-dns=false --ssh=false'
fi
ssh "$SPRITE" 'sudo tailscale status | head -5'
ADDR=$(ssh "$SPRITE" "sudo tailscale status --json" | python3 -c '
import json, sys
name = sys.argv[1].rstrip(".") + "."
for p in json.load(sys.stdin)["Peer"].values():
    if p["DNSName"] == name:
        print(p["TailscaleIPs"][0]); break
' "$MACBOOK")
[ -n "$ADDR" ] || { echo "$MACBOOK is not on the tailnet yet (tailscale status on the sprite does not list it)" >&2; exit 1; }
echo "==> $MACBOOK is $ADDR"

echo "==> the sprite's key to the MacBook"
ssh "$SPRITE" '[ -f ~/.ssh/macbook_ed25519 ] || ssh-keygen -q -t ed25519 -N "" -C chiron-sprite-runner -f ~/.ssh/macbook_ed25519'
ssh "$SPRITE" "python3 - '$ADDR' '$MACUSER' '$MACBOOK'" <<'PY'
import os, re, sys
addr, user, name = sys.argv[1], sys.argv[2], sys.argv[3]
path = os.path.expanduser("~/.ssh/config")
stanza = f"""Host macbook
  # {name}, by address: the sprite has no MagicDNS.
  HostName {addr}
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
