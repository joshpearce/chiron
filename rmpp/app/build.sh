#!/bin/bash
# Build the Chiron AppLoad bundle into output/ and, if the AppLoad PC
# emulator checkout is present, refresh its applications_root copy.
#
# CHIRON_SERVER bakes a server address into the bundle for the tablet:
#   CHIRON_SERVER=http://192.168.1.20:8080 ./build.sh
# CHIRON_TOKEN bakes the server auth token alongside it.
# Unset, the bundle keeps the dev default (localhost:8082, the sim server).
set -euo pipefail
cd "$(dirname "$0")"

RCC=$(command -v rcc || echo /opt/homebrew/opt/qt/share/qt/libexec/rcc)

rm -rf output
mkdir output
cp icon.png manifest.json output

SRC=.
if [ -n "${CHIRON_SERVER:-}" ]; then
  SRC=$(mktemp -d)
  trap 'rm -rf "$SRC"' EXIT
  cp application.qrc "$SRC/"
  cp -r ui "$SRC/"
  sed -i '' "s|property string serverBase: \"[^\"]*\"|property string serverBase: \"$CHIRON_SERVER\"|" \
    "$SRC/ui/main.qml"
  grep -q "$CHIRON_SERVER" "$SRC/ui/main.qml" || { echo "server bake failed" >&2; exit 1; }
  echo "baked server: $CHIRON_SERVER"
  if [ -n "${CHIRON_TOKEN:-}" ]; then
    sed -i '' "s|property string authToken: \"\"|property string authToken: \"$CHIRON_TOKEN\"|" \
      "$SRC/ui/main.qml"
    grep -q "authToken: \"$CHIRON_TOKEN\"" "$SRC/ui/main.qml" || { echo "token bake failed" >&2; exit 1; }
    echo "baked auth token"
  fi
fi
"$RCC" --binary -o output/resources.rcc "$SRC/application.qrc"

EMU_ROOT="../vendor/rm-appload/applications_root"
if [ -d "$EMU_ROOT" ]; then
  rm -rf "$EMU_ROOT/chiron"
  cp -r output "$EMU_ROOT/chiron"
  echo "installed into emulator applications_root"
fi
echo "bundle in output/"
