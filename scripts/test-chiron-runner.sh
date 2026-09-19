#!/bin/bash
# The runner is the only thing the sprite's key can run on the MacBook.
# It must take its three verbs and refuse everything else, without a
# shell in between. Runs anywhere with git.
set -euo pipefail
RUNNER="$(cd "$(dirname "$0")" && pwd)/chiron-runner"
T=$(mktemp -d); trap 'rm -rf "$T"' EXIT
export HOME="$T"
mkdir -p "$T/src" "$T/builds/b1"
git init -q "$T/src/chiron"
echo '{"id":"b1"}' > "$T/builds/b1/build.json"
pass=0; fail=0
ok()  { pass=$((pass+1)); }
bad() { fail=$((fail+1)); echo "FAIL: $1" >&2; }

run() { SSH_ORIGINAL_COMMAND="$1" "$RUNNER" 2>&1; }

# No command at all is not a shell.
out=$(SSH_ORIGINAL_COMMAND= "$RUNNER" 2>&1) && bad "empty command accepted" || ok
# Arbitrary commands are refused, however they are dressed.
for c in "ls" "bash" "git-receive-pack /etc" "build main; ls" "fetch b1 ../../.ssh/id_ed25519" "fetch b1/../b2 build.json" "git-upload-pack '$T/src/chiron'"; do
  run "$c" >/dev/null && bad "accepted: $c" || ok
done
# A push into the checkout is passed to git-receive-pack, nothing else.
for p in "'$T/src/chiron'" "'src/chiron'" "'~/src/chiron'"; do
  out=$(run "git-receive-pack $p" </dev/null) || true
  echo "$out" | grep -q "report-status" && ok || bad "git-receive-pack $p did not answer: $out"
done
run "git-receive-pack 'src/other'" >/dev/null && bad "push elsewhere accepted" || ok
# fetch streams a file from a build, by build id and plain file name.
out=$(run "fetch b1 build.json") && [ "$out" = '{"id":"b1"}' ] && ok || bad "fetch: $out"
run "fetch nonesuch build.json" >/dev/null && bad "fetch of a missing build accepted" || ok
# build runs the build script with the ref and only the ref.
mkdir -p "$T/src/chiron/scripts"
printf '#!/bin/bash\necho "built $*"\n' > "$T/src/chiron/scripts/mac-build.sh"; chmod +x "$T/src/chiron/scripts/mac-build.sh"
out=$(run "build main") && [ "$out" = "built main" ] && ok || bad "build: $out"
run "build 'main; ls'" >/dev/null && bad "build with a shell in the ref accepted" || ok
run "build" >/dev/null && bad "build without a ref accepted" || ok

echo "runner: $pass passed, $fail failed"
[ "$fail" -eq 0 ]
