#!/bin/bash
# Put the gate and sshd on the sprite, so that from then on the way in is
# `ssh chiron` through chiron-dev, from any Mac account, with no sprite CLI.
#
# Needs the sprite CLI logged into the personal org (the one that can see
# `chiron`). Run it once; it is safe to run again to update the gate, the
# server, or the set of Mac keys.
#
#   scripts/sprite-bootstrap-ssh.sh [pubkey.pub ...]
#
# With no arguments the keys installed are the two YubiKey keys and
# ~/.ssh/chiron_ed25519.pub, generated here if absent (a software key, so a
# Claude Code session on the Mac can ssh without a touch). Keys the iPad
# enrols later are appended by the server, so authorized_keys is merged,
# never overwritten.
#
# SPRITE=<name> picks the sprite (default chiron); the URL comes from
# `sprite info`. On a fresh sprite there is no service to read the key from:
# pass CHIRON_KEY=<shared key> (the same one the iPad and 1Password hold).
# The model credential is read from 1Password at run time.
set -euo pipefail
cd "$(dirname "$0")/.."
. scripts/sprite-service.sh
URL=$(sprite -s "$SPRITE" info 2>/dev/null | awk '/^URL:/{print $2}')
[ -n "$URL" ] || { echo "sprite $SPRITE not visible: is the sprite CLI on the personal org?" >&2; exit 1; }
echo "==> $SPRITE at $URL"

if [ $# -gt 0 ]; then
  PUBKEYS=("$@")
else
  [ -f ~/.ssh/chiron_ed25519 ] || ssh-keygen -q -t ed25519 -N '' -C chiron-dev -f ~/.ssh/chiron_ed25519
  PUBKEYS=(~/.ssh/id_ed25519_sk_yk1.pub ~/.ssh/id_ed25519_sk_yk2.pub ~/.ssh/chiron_ed25519.pub)
fi
for k in "${PUBKEYS[@]}"; do [ -f "$k" ] || { echo "no such key: $k" >&2; exit 1; }; done

echo "==> building for linux/amd64"
STAGE=$(mktemp -d)
trap 'rm -rf "$STAGE"' EXIT
(cd server-go && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o "$STAGE/chiron-gate" ./cmd/chiron-gate)
(cd server-go && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o "$STAGE/chiron-server" ./cmd/chiron-server)
(cd server-go && go test ./... >/dev/null) && echo "    all packages pass"

cat "${PUBKEYS[@]}" > "$STAGE/mac_keys"
cat > "$STAGE/sshd_config" <<'CONF'
# User-mode sshd, reached only through chiron-gate's /ssh tunnel.
Port 2222
ListenAddress 127.0.0.1
HostKey /home/sprite/.ssh/host_ed25519
PidFile none
UsePAM no
PasswordAuthentication no
KbdInteractiveAuthentication no
PubkeyAuthentication yes
AuthorizedKeysFile /home/sprite/.ssh/authorized_keys
PermitRootLogin no
AllowUsers sprite
StrictModes yes
ClientAliveInterval 30
ClientAliveCountMax 4
Subsystem sftp /usr/lib/openssh/sftp-server
CONF

echo "==> the shared key and the model credential"
KEY=$(sprite_current_key)
[ -n "$KEY" ] || { echo "FATAL: no key: no chiron-server service to read it from; pass CHIRON_KEY=..." >&2; exit 1; }
MODEL=$(sprite_anthropic_token)
[ -n "$MODEL" ] || echo "    WARNING: no Anthropic token from 1Password; the server will run without a model"

echo "==> corpus and config (whatever the sprite lacks)"
# COPYFILE_DISABLE stops macOS writing AppleDouble ._* files, which the
# corpus parser chokes on.
COPYFILE_DISABLE=1 tar czf - corpus corpus-v2 server/config.yaml ipad-app/Chiron/Resources/katex assets/fonts \
  | sprite -s "$SPRITE" exec -- bash -c 'mkdir -p /home/sprite/chiron && cd /home/sprite/chiron && tar xzf - && echo "    corpus, config, katex and fonts in place"' \
  2>&1 | grep -v LIBARCHIVE || true

echo "==> copying binaries, sshd config and Mac keys"
tar czf - -C "$STAGE" chiron-gate chiron-server mac_keys sshd_config \
  | sprite -s "$SPRITE" exec -- bash -c '
    set -e
    mkdir -p /home/sprite/chiron/bin /home/sprite/.ssh && chmod 700 /home/sprite/.ssh
    cd /home/sprite/.ssh && tar xzf -
    mv -f chiron-gate /home/sprite/chiron/bin/chiron-gate.new
    mv -f chiron-server /home/sprite/chiron/bin/chiron-server.new
    chmod +x /home/sprite/chiron/bin/*.new
    touch authorized_keys && chmod 600 authorized_keys
    while read -r line; do grep -qxF "$line" authorized_keys || echo "$line" >> authorized_keys; done < mac_keys
    rm mac_keys
    echo "    authorized_keys now holds $(wc -l < authorized_keys) keys"' \
  2>&1 | grep -v LIBARCHIVE || true

echo "==> sshd, tmux and host key"
sprite -s "$SPRITE" exec -- bash -c '
  set -e
  command -v sshd >/dev/null || { echo "installing openssh-server"; sudo DEBIAN_FRONTEND=noninteractive apt-get install -y -q openssh-server >/dev/null; }
  command -v tmux >/dev/null || { echo "installing tmux"; sudo DEBIAN_FRONTEND=noninteractive apt-get install -y -q tmux >/dev/null; }
  # An interactive ssh login lands in the one tmux session, so a dropped
  # connection (the iPad sleeping, the sprite hibernating) resumes where it
  # was. Commands, scp and rsync are non-interactive and unaffected.
  grep -q "tmux new -A -s chiron" /home/sprite/.bashrc || cat >> /home/sprite/.bashrc <<"RC"
if [ -n "$SSH_TTY" ] && [ -z "$TMUX" ] && command -v tmux >/dev/null; then
  exec tmux new -A -s chiron
fi
RC
  [ -f /home/sprite/.ssh/host_ed25519 ] || ssh-keygen -q -t ed25519 -N "" -f /home/sprite/.ssh/host_ed25519
  # The sprite user has no password, which leaves its shadow entry locked
  # ("!"), and sshd refuses a locked account even for key auth. "*" means no
  # password without the lock.
  sudo usermod -p "*" sprite
  echo -n "host key: "; ssh-keygen -lf /home/sprite/.ssh/host_ed25519.pub' 2>&1 | sed 's/^/    /'

echo "==> swapping the services"
sprite -s "$SPRITE" exec -- bash -c "
  set -e
  cd /home/sprite/chiron/bin
  [ -f chiron-server ] && cp chiron-server chiron-server.\$(date +%b%d | tr A-Z a-z) || true
  mv -f chiron-server.new chiron-server
  mv -f chiron-gate.new chiron-gate
  $(sprite_sshd_script)
  $(sprite_service_script "$KEY" "$MODEL")
  $(sprite_gate_script "$KEY")" \
  2>&1 | grep -Ev '"type":"(stdout|stderr|started|stopping|stopped|complete)"' | sed 's/^/    /'

echo "==> the developer's side: PATH, the checkout, Claude Code"
sprite -s "$SPRITE" exec -- bash -c '
  set -e
  # GOBIN: the sprite'"'"'s Go workspace is under /.sprite, off PATH; installs go to ~/go/bin.
  grep -q "/.sprite/bin" /home/sprite/.bashrc || sed -i "1i export GOBIN=\$HOME/go/bin\nexport PATH=/.sprite/bin:\$HOME/go/bin:\$HOME/.local/bin:\$PATH" /home/sprite/.bashrc
  grep -q "/.sprite/bin" /home/sprite/.profile 2>/dev/null || printf "export GOBIN=\$HOME/go/bin\nexport PATH=/.sprite/bin:\$HOME/go/bin:\$HOME/.local/bin:\$PATH\n" >> /home/sprite/.profile
  mkdir -p /home/sprite/src/chiron && cd /home/sprite/src/chiron
  [ -d .git ] || { git init -q -b main && git config receive.denyCurrentBranch updateInstead; }
  command -v claude >/dev/null || [ -x /home/sprite/.local/bin/claude ] || { curl -fsSL https://claude.ai/install.sh | bash >/tmp/claude-install.log 2>&1 && echo "claude installed"; }
  echo "checkout at ~/src/chiron ($(git -C /home/sprite/src/chiron rev-parse --short HEAD 2>/dev/null || echo empty)); push it from the Mac with: git push $SPRITE main"' 2>&1 | sed 's/^/    /'
git remote get-url "$SPRITE" >/dev/null 2>&1 || git remote add "$SPRITE" "sprite@$SPRITE:src/chiron"

echo "==> the public URL"
# The iPad and the tunnel come from the open internet; the shared key is
# what guards the book, the gate refuses /ssh without it, and sshd wants a
# device key on top.
sprite -s "$SPRITE" config update --url-auth public 2>&1 | tail -1 | sed 's/^/    /'

echo "==> verifying from outside"
printf '    /ping            %s\n' "$(curl -s -o /dev/null -w '%{http_code}' -m 20 $URL/ping)"
printf '    /health no token %s\n' "$(curl -s -o /dev/null -w '%{http_code}' -m 20 $URL/health)"
printf '    /health authed   %s\n' "$(curl -s -o /dev/null -w '%{http_code}' -m 20 -H "Authorization: Bearer $KEY" $URL/health)"
printf '    /ssh no token    %s\n' "$(curl -s -o /dev/null -w '%{http_code}' -m 20 $URL/ssh)"
printf '    /agent/keys      %s\n' "$(curl -s -o /dev/null -w '%{http_code}' -m 20 -H "Authorization: Bearer $KEY" $URL/agent/keys)"
echo "Expect 200 / 401 / 200 / 401 / 200 (the last is key enrolment for the iPad)."

echo
echo "Then, once:"
echo "  (cd server-go && go install ./cmd/chiron-dev)      # puts chiron-dev on PATH via GOBIN"
echo "  mkdir -p ~/.config/chiron-dev"
echo "  printf 'url = $URL\nkey = op://<vault>/<item>/password\nop_account = my\n' > ~/.config/chiron-dev/config"
echo "  chiron-dev ssh-config >> ~/.ssh/config"
echo "  ssh chiron"
