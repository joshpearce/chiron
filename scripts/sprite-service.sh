#!/bin/bash
# The sprite's service definitions, in one place.
#
# Recreating is the only way to change a service's environment, and for a
# while each script carried its own copy of the command line. They drifted:
# rotating the key reinstated uvicorn over the Go binary, and every health
# check passed either way, so nothing said so. One definition, sourced by all.
#
# Layout on the sprite (since the gate, 2026-09-02):
#   chiron-gate    0.0.0.0:8080   the one public port: /ssh tunnel, proxy
#   chiron-server  127.0.0.1:8081 the book, behind the gate
#   sshd           127.0.0.1:2222 user-mode, keys only, reached via the gate
#
# Each function prints a script that runs inside the sprite via `sprite exec`;
# $KEY is interpolated on the Mac and never printed.

sprite_service_script() {
  local key="$1"
  cat <<EOF
set -e
sprite-env services stop chiron-server 2>/dev/null || true
sprite-env services delete chiron-server 2>/dev/null || true
sprite-env services create chiron-server \\
  --cmd /home/sprite/chiron/bin/chiron-server \\
  --args '-addr,127.0.0.1:8081,-config,/home/sprite/chiron/server/config.yaml' \\
  --env 'CHIRON_AUTH_TOKEN=${key},CHIRON_AUTHORIZED_KEYS=/home/sprite/.ssh/authorized_keys' \\
  --dir /home/sprite/chiron/server
sleep 4
echo -n 'server unauthenticated /health -> '; curl -s -o /dev/null -w '%{http_code}\\n' http://127.0.0.1:8081/health
echo -n 'server unauthenticated /ping   -> '; curl -s -o /dev/null -w '%{http_code}\\n' http://127.0.0.1:8081/ping
echo -n 'server authenticated  /health  -> '; curl -s -o /dev/null -w '%{http_code}\\n' -H 'Authorization: Bearer ${key}' http://127.0.0.1:8081/health
EOF
}

sprite_gate_script() {
  local key="$1"
  cat <<EOF
set -e
sprite-env services stop chiron-gate 2>/dev/null || true
sprite-env services delete chiron-gate 2>/dev/null || true
sprite-env services create chiron-gate \\
  --cmd /home/sprite/chiron/bin/chiron-gate \\
  --args '-listen,0.0.0.0:8080,-upstream,http://127.0.0.1:8081,-ssh,127.0.0.1:2222' \\
  --env 'CHIRON_AUTH_TOKEN=${key}' \\
  --needs chiron-server,sshd \\
  --dir /home/sprite/chiron
sleep 2
echo -n 'gate /ping via proxy          -> '; curl -s -o /dev/null -w '%{http_code}\\n' http://127.0.0.1:8080/ping
echo -n 'gate /ssh without key         -> '; curl -s -o /dev/null -w '%{http_code}\\n' http://127.0.0.1:8080/ssh
EOF
}

sprite_sshd_script() {
  cat <<'EOF'
set -e
sprite-env services stop sshd 2>/dev/null || true
sprite-env services delete sshd 2>/dev/null || true
sprite-env services create sshd \
  --cmd /usr/sbin/sshd \
  --args '-D,-e,-f,/home/sprite/.ssh/sshd_config' \
  --dir /home/sprite
sleep 2
echo -n 'sshd on 127.0.0.1:2222        -> '; (exec 3<>/dev/tcp/127.0.0.1/2222 && head -c 7 <&3 && echo) || echo 'not listening'
EOF
}

# One --env flag, comma-separated: sprite-env keeps only the last --env
# given, which once silently dropped the key and left the book open.

# Read the key currently installed on a service, without printing it. The
# gate and the server carry the same key; either will do.
sprite_current_key() {
  sprite -s chiron exec -- bash -c "sprite-env services list" 2>/dev/null \
    | python3 -c '
import sys, json
for line in sys.stdin:
    line = line.strip()
    if line.startswith("["):
        by_name = {s.get("name"): s for s in json.loads(line)}
        for name in ("chiron-server", "chiron-gate"):
            key = by_name.get(name, {}).get("env", {}).get("CHIRON_AUTH_TOKEN", "")
            if key:
                print(key, end="")
                sys.exit(0)
sys.exit("no service carries CHIRON_AUTH_TOKEN")
'
}
