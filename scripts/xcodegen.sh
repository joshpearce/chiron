#!/bin/bash
# Generate the ignored Xcode project with the local signing identifiers.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
source "$ROOT/scripts/signing-config.sh"
cd "$ROOT/ipad-app"
exec xcodegen generate "$@"
