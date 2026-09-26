# Chiron on orca

This is the Chiron-owned contract for the persistent Incus **system
container** on the homelab host `orca`. Creating the guest, networking it,
installing host packages, copying releases, and starting or changing the live
service belong to `homelab_agent`.

Chiron runs directly under systemd as an unprivileged `chiron` user. There is
no nested Podman, Quadlet, image registry, or mutable container image. Run one
service instance only: startup takes a non-blocking lock at
`/var/lib/chiron/state/.chiron-writer.lock`; a second writer exits.

## Install and update contract

The guest needs CA certificates, Chromium, Poppler utilities, the supported
Claude Code CLI, and any runtime libraries those packages require. Go is only
needed in a build workspace. Build the Linux binary from the intended commit:

```sh
cd /path/to/chiron/server-go
CGO_ENABLED=0 go build -trimpath -buildvcs=false -o ../build/chiron-server ./cmd/chiron-server
```

Stage a release under `/opt/chiron/releases/<git-commit>/` containing the
binary plus `assets/`, `corpus/`, and `corpus-v2/`, owned by root and not
writable by `chiron`. Atomically point `/opt/chiron/current` at that directory,
and install these repository files:

```text
deploy/orca/config.yaml          -> /etc/chiron/config.yaml
deploy/orca/chiron.service       -> /etc/systemd/system/chiron.service
deploy/orca/chiron-tmpfiles.conf -> /etc/tmpfiles.d/chiron.conf
build/chiron-server              -> /usr/local/libexec/chiron-server
```

Create `/etc/chiron/auth-token` as a newline-terminated random bearer token,
owned `root:chiron`, mode `0640`. Then run `systemd-tmpfiles --create`, reload
systemd, and enable/start `chiron.service`. Updates replace the immutable
release and binary while the service is stopped, then restart it. Never start
old and new releases concurrently against the same state tree.

The service listens on container port 8080. Homelab ingress must allow requests
to remain open for at least 45 minutes: Claude planner/author calls have
40-minute application deadlines. Shutdown gets 45 seconds at systemd and the
HTTP server gives an in-flight exchange 30 seconds.

## Claude subscription authentication

The configured provider is the personal branch's supported `claude-cli`
backend: headless `claude -p`, billed through Josh's Claude subscription. It
does not use an Anthropic API key. Authenticate interactively as the service
user, with exactly the same home and config directory the unit uses:

```sh
sudo -u chiron env HOME=/var/lib/chiron \
  CLAUDE_CONFIG_DIR=/var/lib/chiron/claude \
  /usr/local/bin/claude
```

Choose the Claude App subscription login and finish its browser flow, then exit
the CLI. Verify non-interactively before starting Chiron:

```sh
sudo -u chiron env HOME=/var/lib/chiron \
  CLAUDE_CONFIG_DIR=/var/lib/chiron/claude \
  /usr/local/bin/claude -p 'Reply with OK' --tools '' --max-turns 1
```

The CLI credential and settings live in `/var/lib/chiron/claude`; protect them
like a password. Re-run the login as `chiron` if the subscription session is
revoked or expires. Chiron disables Claude auto-update/nonessential traffic at
runtime; `homelab_agent` owns deliberate CLI upgrades and should run the smoke
test after each one.

## Filesystem ownership and backup

The boundaries are intentional:

| Path | Class | Backup? |
|---|---|---|
| `/var/lib/chiron/state` | durable learner state, imports, primers, requests | yes |
| `/var/lib/chiron/generated` | generated corpora | yes |
| `/var/lib/chiron/deployable/apple` | deployable Apple artifacts produced by the Mac | yes, if they must remain installable |
| `/var/lib/chiron/claude` | Claude subscription login/settings | yes, encrypted and access-restricted |
| `/var/cache/chiron` | reproducible page/fetch cache | no |
| `/opt/chiron/releases/<commit>` | root-owned release/build output | no; reproduce from Git commit |
| `/var/tmp/chiron` and systemd private `/tmp` | render/temp scratch | no |
| `/etc/chiron/auth-token` | client bearer secret | yes, in the secret store rather than a general filesystem snapshot |

Stop `chiron.service` (or otherwise guarantee it is quiescent) before taking a
consistent backup of the four `/var/lib/chiron` trees. Restore them as a unit,
restore the matching auth secrets, select a known release, fix ownership to
`chiron:chiron`, and start exactly one service instance. Cache and temp content
may be discarded at any time.

Apple source builds, Xcode project generation, signing identities, profiles,
and App Store credentials remain on the macOS/Xcode runner. The Linux guest
only stores and serves completed deployable artifacts; it never signs them.

## Checks and security expectations

`GET /ping` is unauthenticated and returns only liveness. Every normal API and
`GET /health` requires the bearer token:

```sh
curl -fsS http://CONTAINER:8080/ping
curl -fsS -H "Authorization: Bearer $CHIRON_TOKEN" \
  http://CONTAINER:8080/health
```

`/health` reports the configured Claude CLI as available when its executable
is on `PATH`; the explicit noninteractive smoke test above proves credentials.
User-supplied page/feed/image URLs use a public-only dialer. Loopback, private,
link-local, CGNAT, documentation, benchmark, and reserved address ranges are
rejected on the initial request and redirects, protecting the homelab from
outbound-fetch SSRF.

Inspect logs with `journalctl -u chiron.service`. A writer-lock failure means
another Chiron process already owns the state tree; find it rather than deleting
the lock file (the kernel lock, not the file's presence, is authoritative).

## Personal-branch migration assessment

Keep the runtime behavior from `4287d87`: the single-writer lease, explicit
storage roots, secret-file support, long HTTP request/shutdown behavior, and
public-only fetching. Keep Apple signing local to the Mac; it is independent of
Linux deployment. Drop its Dockerfile, `.dockerignore`, NAS configuration,
NAS operations text, and DeepInfra API-key/vision changes; the target uses the
already-supported Claude subscription backend. Drop `3d4a2c3` and `017b735`
entirely: they only publish NAS OCI images. `3dc750a` is an Apple UI/Xcode
compatibility change, not an Incus prerequisite, and should be evaluated or
cherry-picked separately against the current Apple tree rather than coupled to
this server migration. Likewise, `883de80` is a useful Mac ownership cleanup
but not a server-deployment prerequisite; do not mix it into the homelab
rollout.
