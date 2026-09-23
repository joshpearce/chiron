# Chiron on a NAS

The image runs one unprivileged `chiron-server` process on port 8080. Run
exactly one replica against a data volume: startup takes a non-blocking lock at
`data/.chiron-writer.lock` and a second writer exits rather than corrupting the
event log or snapshots.

## Storage and secrets

Mount one durable volume at `/var/lib/chiron`. Back up `data/`, `generated/`,
`builds/`, `requests/`, `readings/`, and `primers/`. `/var/cache/chiron` is a
rebuildable page/model cache and `/var/tmp/chiron` is scratch space; either may
be a separate NAS cache dataset or ephemeral storage. The shipped config is
read-only at `/etc/chiron/config.yaml` and the authored corpora are read-only
image content under `/opt/chiron/content`.

Mount newline-terminated secret files, readable by UID 10001, at:

- `/run/secrets/chiron_auth_token`: the bearer token required by every route
  except unauthenticated liveness `GET /ping` and opaque installer artifacts.
- `/run/secrets/deepinfra_api_key`: the DeepInfra API token. It is sent as
  `Authorization: Bearer …` for `/models`, structured text requests, and vision.

Environment overrides `CHIRON_AUTH_TOKEN`, `CHIRON_AUTH_TOKEN_FILE`,
`CHIRON_LLM_API_KEY`, and `CHIRON_LLM_API_KEY_FILE` take precedence. Prefer
files so secrets do not appear in environment inspection.

## Build and run

Build from a clean checkout of the intended commit and record the digest:

```sh
docker buildx build --platform linux/amd64,linux/arm64 \
  --tag REGISTRY/chiron:GIT_COMMIT --push .
docker buildx imagetools inspect REGISTRY/chiron:GIT_COMMIT
```

Pin the homelab deployment to the returned `REGISTRY/chiron@sha256:…`, not a
mutable tag. A representative runtime contract is:

```sh
docker run --name chiron --restart unless-stopped --read-only \
  --user 10001:10001 -p 8080:8080 \
  -v chiron-data:/var/lib/chiron \
  --tmpfs /var/cache/chiron:uid=10001,gid=10001 \
  --tmpfs /var/tmp/chiron:uid=10001,gid=10001 \
  -v /nas/secrets/chiron-auth:/run/secrets/chiron_auth_token:ro \
  -v /nas/secrets/deepinfra:/run/secrets/deepinfra_api_key:ro \
  REGISTRY/chiron@sha256:DIGEST
```

Do not set an ingress response timeout below 15 minutes: text authoring is
configured for ten-minute upstream calls and whole-subject work continues in
background jobs. Graceful shutdown allows 30 seconds for an in-flight exchange.

## Checks and recovery

`GET /ping` is the unauthenticated liveness/readiness probe and returns only
`{"ok":true}`. `GET /health` requires the Chiron bearer token and reports the
DeepInfra connection/model plus corpus inventory:

```sh
curl -fsS http://HOST:8080/ping
curl -fsS -H "Authorization: Bearer $CHIRON_TOKEN" http://HOST:8080/health
```

A 401 from `/health` is expected without the token. A successful `/health`
with `llm.connected:false` means Chiron is serving durable content but its
model dependency is unavailable or unauthorized. Inspect logs before restart.
For recovery, stop Chiron, restore the durable volume as a unit, and start the
same immutable image digest. Never run old and new containers concurrently on
the restored volume.

User-supplied page/feed URLs are resolved and dialed through a public-only
transport. Loopback, private, link-local, CGNAT, documentation, benchmark, and
reserved address ranges are rejected on the original request and redirects.
