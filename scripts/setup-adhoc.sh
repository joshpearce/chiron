#!/bin/bash
# Create the dummy upstream that lets the Mac share over Wi-Fi.
#
# macOS will not share a connection *from* Wi-Fi *to* Wi-Fi, so with Wi-Fi as
# the uplink the soft-AP option is simply absent. The way round it is a network
# service on the loopback device: Internet Sharing is happy to treat it as the
# thing being shared, which frees Wi-Fi to be the target. In the air there is no
# uplink to share anyway - the point is only to carry traffic between the Mac
# and the iPad, and 192.168.2.1 is served by the bridge either way.
#
# Run once (needs sudo). It does NOT enable Internet Sharing - that is a GUI
# toggle, and doing it here would drop your Wi-Fi the moment you ran it.
set -euo pipefail

SERVICE="AdHoc"

if networksetup -getinfo "$SERVICE" >/dev/null 2>&1; then
  echo "'$SERVICE' already exists:"
  networksetup -getinfo "$SERVICE" | sed 's/^/  /'
else
  echo "==> creating the '$SERVICE' service on lo0 (sudo)"
  sudo networksetup -createnetworkservice "$SERVICE" lo0
  # Any address off the shared subnet will do; it is never routed anywhere.
  sudo networksetup -setmanual "$SERVICE" 10.55.55.1 255.255.255.0
  echo "  created"
fi

cat <<'EOF'

Next, in the GUI (Internet Sharing cannot be toggled from the command line
without a full-disk-access shell, and flipping it will drop your Wi-Fi):

  System Settings > General > Sharing > Internet Sharing
    Share your connection from : AdHoc
    To devices using           : Wi-Fi
    Wi-Fi Options...           : set a network name and a WPA2 password
  then turn Internet Sharing ON.

On the iPad: airplane mode ON, then Wi-Fi back ON, join that network.
The app's server field stays http://192.168.2.1:8080.

Verify with scripts/preflight-check.sh - it now fails if Wi-Fi sharing is off,
which it did not before, and reports how many devices have joined.

Turning Internet Sharing off again restores normal Wi-Fi. The AdHoc service is
inert when unused; remove it with:
  sudo networksetup -removenetworkservice AdHoc
EOF
