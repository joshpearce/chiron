#!/bin/bash
# Render each design frame in screens.html to a 1620x2160 PNG for visual
# verification. Frames are absolutely positioned sections; each render
# isolates one by index and pins it to the origin.
set -euo pipefail
cd "$(dirname "$0")"
CHROME="/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
mkdir -p frames
N=$(grep -c 'class="fr' screens.html)
for i in $(seq 1 "$N"); do
  cat screens.html > /tmp/frame-render.html
  cat >> /tmp/frame-render.html <<EOF
<style>
section.fr { display: none !important; }
section.fr:nth-of-type($i) { display: block !important; left: 0 !important; top: 0 !important; box-shadow: none !important; }
.cap { display: none !important; }
</style>
EOF
  "$CHROME" --headless=new --disable-gpu --no-first-run \
    --virtual-time-budget=20000 --window-size=1620,2160 --hide-scrollbars \
    --screenshot="frames/frame-$(printf '%02d' "$i").png" \
    "file:///tmp/frame-render.html" 2>/dev/null
done
ls frames/
