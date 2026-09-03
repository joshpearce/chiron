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
set -euo pipefail
cd "$(dirname "$0")/.."
. scripts/sprite-service.sh

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

echo "==> reading the current key off the service"
KEY=$(sprite_current_key)
[ -n "$KEY" ] || { echo "FATAL: no CHIRON_AUTH_TOKEN on the existing chiron-server service" >&2; exit 1; }

echo "==> copying binaries, sshd config and Mac keys"
tar czf - -C "$STAGE" chiron-gate chiron-server mac_keys sshd_config \
  | sprite -s chiron exec -- bash -c '
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
sprite -s chiron exec -- bash -c '
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
sprite -s chiron exec -- bash -c "
  set -e
  cd /home/sprite/chiron/bin
  [ -f chiron-server ] && cp chiron-server chiron-server.\$(date +%b%d | tr A-Z a-z) || true
  mv -f chiron-server.new chiron-server
  mv -f chiron-gate.new chiron-gate
  $(sprite_sshd_script)
  $(sprite_service_script "$KEY")
  $(sprite_gate_script "$KEY")" \
  2>&1 | grep -Ev '"type":"(stdout|stderr|started|stopping|stopped|complete)"' | sed 's/^/    /'

echo "==> verifying from outside"
printf '    /ping            %s\n' "$(curl -s -o /dev/null -w '%{http_code}' -m 20 https://chiron.example/ping)"
printf '    /health no token %s\n' "$(curl -s -o /dev/null -w '%{http_code}' -m 20 https://chiron.example/health)"
printf '    /health authed   %s\n' "$(curl -s -o /dev/null -w '%{http_code}' -m 20 -H "Authorization: Bearer $KEY" https://chiron.example/health)"
printf '    /ssh no token    %s\n' "$(curl -s -o /dev/null -w '%{http_code}' -m 20 https://chiron.example/ssh)"
printf '    /agent/keys      %s\n' "$(curl -s -o /dev/null -w '%{http_code}' -m 20 -H "Authorization: Bearer $KEY" https://chiron.example/agent/keys)"
echo "Expect 200 / 401 / 200 / 401 / 200 (the last is key enrolment for the iPad)."

echo
echo "Then, once:"
echo "  (cd server-go && go install ./cmd/chiron-dev)      # puts chiron-dev on PATH via GOBIN"
echo "  mkdir -p ~/.config/chiron-dev"
echo "  printf 'url = https://chiron.example\nkey = op://<vault>/chiron-sprite/password\nop_account = flyio\n' > ~/.config/chiron-dev/config"
echo "  chiron-dev ssh-config >> ~/.ssh/config"
echo "  ssh chiron"
