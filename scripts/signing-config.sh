#!/bin/bash
# Source the repository-local Apple signing inputs and validate the values used
# by XcodeGen and the build scripts. Machine/account-specific values live in
# signing/local.config, which is ignored; the committed .example documents the
# contract.

CHIRON_REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CHIRON_SIGNING_CONFIG="${CHIRON_SIGNING_CONFIG:-$CHIRON_REPO_ROOT/signing/local.config}"

if [ -f "$CHIRON_SIGNING_CONFIG" ]; then
  CHIRON_TEAM_ID_OVERRIDE="${CHIRON_TEAM_ID:-}"
  CHIRON_BUNDLE_ID_OVERRIDE="${CHIRON_BUNDLE_ID:-}"
  CHIRON_APP_GROUP_OVERRIDE="${CHIRON_APP_GROUP:-}"
  # This is a deliberately local, user-controlled shell fragment.
  # shellcheck source=/dev/null
  source "$CHIRON_SIGNING_CONFIG"
  [ -z "$CHIRON_TEAM_ID_OVERRIDE" ] || CHIRON_TEAM_ID="$CHIRON_TEAM_ID_OVERRIDE"
  [ -z "$CHIRON_BUNDLE_ID_OVERRIDE" ] || CHIRON_BUNDLE_ID="$CHIRON_BUNDLE_ID_OVERRIDE"
  [ -z "$CHIRON_APP_GROUP_OVERRIDE" ] || CHIRON_APP_GROUP="$CHIRON_APP_GROUP_OVERRIDE"
  unset CHIRON_TEAM_ID_OVERRIDE CHIRON_BUNDLE_ID_OVERRIDE CHIRON_APP_GROUP_OVERRIDE
fi

: "${CHIRON_TEAM_ID:?Set CHIRON_TEAM_ID or create signing/local.config from signing/local.config.example}"
: "${CHIRON_BUNDLE_ID:?Set CHIRON_BUNDLE_ID or create signing/local.config from signing/local.config.example}"
: "${CHIRON_APP_GROUP:?Set CHIRON_APP_GROUP or create signing/local.config from signing/local.config.example}"

case "$CHIRON_TEAM_ID" in
  *[!A-Za-z0-9]*) echo "CHIRON_TEAM_ID must contain only letters and digits" >&2; return 1 2>/dev/null || exit 1 ;;
esac
case "$CHIRON_BUNDLE_ID" in
  *.*) ;;
  *) echo "CHIRON_BUNDLE_ID must be a reverse-DNS identifier" >&2; return 1 2>/dev/null || exit 1 ;;
esac
case "$CHIRON_APP_GROUP" in
  group.*.*) ;;
  *) echo "CHIRON_APP_GROUP must start with group. and use reverse-DNS form" >&2; return 1 2>/dev/null || exit 1 ;;
esac

export CHIRON_TEAM_ID CHIRON_BUNDLE_ID CHIRON_APP_GROUP
