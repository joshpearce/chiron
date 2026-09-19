#!/bin/bash
# Make this Mac the sprite's build machine. Run once on the MacBook, from
# a folder holding this script and chiron-runner side by side (copy both
# over from the main Mac); safe to run again.
#
#   ./mac-runner-bootstrap.sh <sprite-pubkey-line>
#
# The pubkey line is what scripts/sprite-tailscale.sh prints: the key the
# sprite will ssh in with. It lands in ~/.ssh/authorized_keys bound to
# ~/bin/chiron-runner as its forced command, so that key can push, build
# and fetch, and nothing else. Also done here: Remote Login on, no sleep
# with the lid closed on power, an empty checkout at ~/src/chiron that a
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
HERE="$(cd "$(dirname "$0")" && pwd)"
[ -f "$HERE/chiron-runner" ] || { echo "chiron-runner must sit beside this script" >&2; exit 1; }

echo "==> Remote Login and no sleep on power with the lid closed"
sudo systemsetup -setremotelogin on >/dev/null
sudo pmset -c sleep 0 disksleep 0 displaysleep 5
sudo pmset -a disablesleep 1
# Belt and braces: a user agent that holds the machine awake while on power.
mkdir -p ~/Library/LaunchAgents
cat > ~/Library/LaunchAgents/dev.mjbraun.chiron.awake.plist <<'PLIST'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
  <key>Label</key><string>dev.mjbraun.chiron.awake</string>
  <key>ProgramArguments</key><array><string>/usr/bin/caffeinate</string><string>-s</string></array>
  <key>RunAtLoad</key><true/>
  <key>KeepAlive</key><true/>
</dict></plist>
PLIST
launchctl unload ~/Library/LaunchAgents/dev.mjbraun.chiron.awake.plist 2>/dev/null || true
launchctl load ~/Library/LaunchAgents/dev.mjbraun.chiron.awake.plist

echo "==> the runner and the sprite's key"
mkdir -p ~/bin ~/builds ~/src ~/.ssh
install -m 755 "$HERE/chiron-runner" ~/bin/chiron-runner
chmod 700 ~/.ssh
touch ~/.ssh/authorized_keys; chmod 600 ~/.ssh/authorized_keys
KEY=$(awk '{print $1, $2}' <<<"$PUBKEY")
grep -vF "$KEY" ~/.ssh/authorized_keys > ~/.ssh/authorized_keys.new || true
echo "command=\"$HOME/bin/chiron-runner\",no-port-forwarding,no-X11-forwarding,no-agent-forwarding,no-pty $PUBKEY" >> ~/.ssh/authorized_keys.new
mv ~/.ssh/authorized_keys.new ~/.ssh/authorized_keys

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
  cat > ~/.config/chiron-runner/config <<'CONF'
team_id = ${CHIRON_TEAM_ID}
builds_url = https://chiron.example/builds
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
