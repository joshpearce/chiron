#!/bin/bash
# Build the Chiron AppLoad bundle into output/ and, if the AppLoad PC
# emulator checkout is present, refresh its applications_root copy.
set -euo pipefail
cd "$(dirname "$0")"

RCC=$(command -v rcc || echo /opt/homebrew/opt/qt/share/qt/libexec/rcc)

rm -rf output
mkdir output
cp icon.png manifest.json output
"$RCC" --binary -o output/resources.rcc application.qrc

EMU_ROOT="../vendor/rm-appload/applications_root"
if [ -d "$EMU_ROOT" ]; then
  rm -rf "$EMU_ROOT/chiron"
  cp -r output "$EMU_ROOT/chiron"
  echo "installed into emulator applications_root"
fi
echo "bundle in output/"
