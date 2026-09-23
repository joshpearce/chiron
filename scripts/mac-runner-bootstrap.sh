#!/bin/bash
# Make this Mac the sprite's build machine. Run once on the MacBook, from
# a folder holding this script and chiron-runner side by side (copy both
# over from the main Mac); safe to run again.
#
#   ./mac-runner-bootstrap.sh <sprite-pubkey-line> [admin-pubkey-file ...]
#
# The pubkey line is what scripts/sprite-tailscale.sh prints: the key the
# sprite will ssh in with. It lands in ~/.ssh/authorized_keys bound to
# ~/bin/chiron-runner as its forced command, so that key can push, build
# and fetch, and nothing else. Any admin pubkey files after it are
# installed plainly, for a person's own shell over the LAN or the
# tailnet: the YubiKey keys, and chiron_ed25519 so a Claude Code session
# on the main Mac gets in without a touch. Also done here: Remote Login on, and a
# daemon that lets the lid be closed without sleeping while on power, an empty checkout at ~/src/chiron that a
# push fills, Xcode's iOS platform and Metal toolchain, xcodegen, the
# Simulator the tests use, and the runner's config with the App Store
# Connect key still to fill in.
#
# Left for a person, once: install Tailscale and sign in to the tailnet;
# put the App Store Connect API key at ~/.private_keys/AuthKey_<id>.p8
# and its id and issuer in ~/.config/chiron-runner/config; open Xcode
# once and accept the licence.
set -euo pipefail
PUBKEY="${1:?the public key line the sprite will use}"
shift
ADMIN_KEYS=("$@")
for k in ${ADMIN_KEYS[@]+"${ADMIN_KEYS[@]}"}; do [ -f "$k" ] || { echo "no such key file: $k" >&2; exit 1; }; done
HERE="$(cd "$(dirname "$0")" && pwd)"
[ -f "$HERE/chiron-runner" ] || { echo "chiron-runner must sit beside this script" >&2; exit 1; }

echo "==> Remote Login, and no sleep with the lid closed while on power"
sudo systemsetup -setremotelogin on >/dev/null
sudo pmset -c sleep 0 disksleep 0 displaysleep 5
# The lid is the awkward one. pmset's disablesleep is what makes a laptop
# ignore its lid, and it is system-wide: set it once and the machine will not
# sleep on battery either, which empties it in a bag. A daemon watches the
# power source instead and sets it only while plugged in.
sudo install -m 755 "$HERE/mac-awake" /usr/local/bin/chiron-awake
sudo tee /Library/LaunchDaemons/dev.mjbraun.chiron.awake.plist >/dev/null <<'PLIST'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
  <key>Label</key><string>dev.mjbraun.chiron.awake</string>
  <key>ProgramArguments</key><array><string>/usr/local/bin/chiron-awake</string></array>
  <key>RunAtLoad</key><true/>
  <key>KeepAlive</key><true/>
  <key>StandardOutPath</key><string>/var/log/chiron-awake.log</string>
  <key>StandardErrorPath</key><string>/var/log/chiron-awake.log</string>
</dict></plist>
PLIST
sudo chown root:wheel /Library/LaunchDaemons/dev.mjbraun.chiron.awake.plist
sudo chmod 644 /Library/LaunchDaemons/dev.mjbraun.chiron.awake.plist
sudo launchctl bootout system/dev.mjbraun.chiron.awake 2>/dev/null || true
sudo launchctl bootstrap system /Library/LaunchDaemons/dev.mjbraun.chiron.awake.plist
# An earlier version of this script held the machine awake with a caffeinate
# agent and a blanket `pmset -a disablesleep 1`; both are gone now.
launchctl unload ~/Library/LaunchAgents/dev.mjbraun.chiron.awake.plist 2>/dev/null || true
rm -f ~/Library/LaunchAgents/dev.mjbraun.chiron.awake.plist

echo "==> the runner and the sprite's key"
mkdir -p ~/bin ~/builds ~/src ~/.ssh
install -m 755 "$HERE/chiron-runner" ~/bin/chiron-runner
chmod 700 ~/.ssh
touch ~/.ssh/authorized_keys; chmod 600 ~/.ssh/authorized_keys
KEY=$(awk '{print $1, $2}' <<<"$PUBKEY")
grep -vF "$KEY" ~/.ssh/authorized_keys > ~/.ssh/authorized_keys.new || true
echo "command=\"$HOME/bin/chiron-runner\",no-port-forwarding,no-X11-forwarding,no-agent-forwarding,no-pty $PUBKEY" >> ~/.ssh/authorized_keys.new
for k in ${ADMIN_KEYS[@]+"${ADMIN_KEYS[@]}"}; do
  line=$(head -1 "$k"); id=$(awk '{print $1, $2}' <<<"$line")
  grep -qF "$id" ~/.ssh/authorized_keys.new || echo "$line" >> ~/.ssh/authorized_keys.new
done
mv ~/.ssh/authorized_keys.new ~/.ssh/authorized_keys
echo "    this Mac answers to $(scutil --get LocalHostName).local on the LAN"

echo "==> the checkout a push fills"
if [ ! -d ~/src/chiron/.git ]; then
  git init -q ~/src/chiron
  git -C ~/src/chiron symbolic-ref HEAD refs/heads/main
fi
git -C ~/src/chiron config receive.denyCurrentBranch updateInstead

echo "==> Xcode: iOS platform, Metal toolchain, xcodegen, the test Simulator"
xcode-select -p >/dev/null || { echo "install Xcode first" >&2; exit 1; }
xcodebuild -downloadPlatform iOS >/dev/null 2>&1 || true
xcodebuild -downloadComponent MetalToolchain >/dev/null 2>&1 || true
command -v xcodegen >/dev/null || brew install xcodegen
if ! xcrun simctl list devices | grep -q "chiron-ipad ("; then
  RUNTIME=$(xcrun simctl list runtimes | grep -E "^iOS" | tail -1 | sed -E 's/.*(com\.apple\.CoreSimulator\.SimRuntime\.[A-Za-z0-9-]+).*/\1/')
  xcrun simctl create chiron-ipad "iPad (A16)" "$RUNTIME" >/dev/null
fi

echo "==> the runner's config"
mkdir -p ~/.config/chiron-runner ~/.private_keys
chmod 700 ~/.private_keys
if [ ! -f ~/.config/chiron-runner/config ]; then
  cat > ~/.config/chiron-runner/config <<CONF
team_id = ${CHIRON_TEAM_ID:-<your Apple team id>}
builds_url = ${CHIRON_BUILDS_URL:-https://<your-sprite>.sprites.app/builds}
# App Store Connect API key (Users and Access > Integrations): signing
# without an Apple ID session. The .p8 goes in ~/.private_keys.
asc_key_id =
asc_issuer_id =
asc_key_path = ~/.private_keys/AuthKey_<id>.p8
ui_tests = 0
CONF
fi

cat <<EOT

Done. Still for a person:
  1. Install Tailscale (App Store) and sign in to the personal tailnet.
  2. Fill asc_key_id, asc_issuer_id and asc_key_path in
     ~/.config/chiron-runner/config, with the .p8 in ~/.private_keys.
  3. Open Xcode once, accept the licence, let it finish installing.
Then from the sprite: git push macbook main; ssh macbook build main.
EOT
