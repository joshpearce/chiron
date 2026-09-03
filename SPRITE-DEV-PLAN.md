# Chiron development on the sprite: the plan

Written 2026-09-02, revised the same day after Matt's constraints on wake
and account. The goal: the whole Chiron development environment lives on
the sprite that serves the book, reachable from the Mac and from the iPad
app with the same mechanism, so Claude Code on the sprite can be driven
from either. This document is meant to be executed cold: every fact needed
to start is here or in a file it names.

## 0. Where things stand

- The sprite `chiron` (Fly, personal org `matthew-braun`,
  `https://chiron.example`) runs the Go book server as a
  `sprite-env` service from `/home/sprite/chiron`, which is a copy of the
  repo's server, corpora and config, not a git checkout. Deploys are
  hand-pushed binaries (see the project memory `chiron-deployment`). The
  sprite has Chromium and poppler for page renders and an Anthropic OAuth
  token in the service environment.
- Development happens on the Mac: Xcode for the app, Go for the server, the
  Simulator for verification, `scripts/sim-*.sh` and a debug harness inside
  the app. Claude Code runs on the Mac against the repo.
- The iPad talks to the server over HTTPS with a bearer key stored in the
  Keychain per saved server.

## 1. The two constraints that shape everything

**The sprite hibernates after about 30 seconds idle** and wakes on the next
inbound request. Measured 2026-09-02: the first request to the public URL
after idle took 0.89 s, the next 0.12 s. Memory and filesystem survive
hibernation, so processes (sshd, tmux, a running `claude`) freeze and
resume rather than restart. Consequences: nothing needs a separate "wake"
call, because the connection that opens the shell is itself the request
that wakes the box; but a long Claude Code task with no connection open
will pause until the next connection, so an open shell (with keepalives)
is what keeps it running.

**The `sprite` CLI is per account.** Logged into the work account, the
personal org is invisible (`sprite list` shows only `fly-util`). So
`sprite console`, `sprite exec` and `sprite proxy` cannot be the day-to-day
path in, and cannot be the iPad's path at all. The only thing reachable
without the CLI is the public HTTPS URL on port 8080, which is the book
server. Therefore: **the shell rides port 8080 as a WebSocket carrying a
real SSH session.** The sprite CLI is kept for bootstrap and for rescue.

## 2. The mechanism: SSH over a WebSocket on the book port

Three parts, all small.

1. **`sshd` on the sprite**, listening on `127.0.0.1:22` only, key auth
   only (`PasswordAuthentication no`), run as a `sprite-env` service so it
   comes back after a restore. `~/.ssh/authorized_keys` holds the Mac's key
   and every enrolled iPad key.
2. **A tunnel route on port 8080.** `GET /ssh` with the bearer key upgrades
   to a WebSocket and pipes bytes to `127.0.0.1:22`; nothing else, no
   command execution, target not configurable. Auth is two layers with no
   new code: the bearer key opens the tunnel, then `sshd` demands a key.
   Library: `github.com/coder/websocket` (zero dependencies, has `NetConn`
   for byte piping). Where the route lives is decision 3 below.
3. **Clients speak SSH into the tunnel.**
   - Mac: `chiron-dev` (Go, `server-go/cmd/chiron-dev`) with one job,
     `chiron-dev proxy`: open the WebSocket, retry the connect for up to
     20 s while the sprite wakes, then pipe stdin/stdout. It is an SSH
     `ProxyCommand`, so `ssh`, `scp`, `rsync`, `git push` over ssh, and
     VS Code Remote all work unchanged:
     ```
     Host chiron
       ProxyCommand chiron-dev proxy
       User sprite
       IdentityFile ~/.ssh/chiron_ed25519
       ServerAliveInterval 15
     ```
     `ServerAliveInterval` keeps frames flowing through the tunnel so an
     open-but-quiet session never counts as idle. `chiron-dev` reads the URL
     and an `op://` reference for the key from `~/.config/chiron-dev/config`
     and resolves the key with `op read` at run time; the secret is never
     written to disk.
   - iPad: the same tunnel, an SSH client library, and a terminal view.
     Details in Phase C. The bearer key the app already holds opens the
     tunnel; the device's Ed25519 key, enrolled once, satisfies `sshd`.

Why real SSH instead of a PTY-over-WebSocket: one protocol on both clients,
key auth done by software that has been audited for it, file transfer and
port forwarding for free, and the Mac side needs no custom client at all.

## 3. Phases

**Phase A: the tunnel and the Mac.** Built 2026-09-02: `server-go/gate`
(the handler), `cmd/chiron-gate`, `gate/client` and `cmd/chiron-dev`, the
shared `auth` package, `scripts/sprite-bootstrap-ssh.sh` and the service
layout in `scripts/sprite-service.sh`. Proven on the Mac end to end with a
user-mode sshd behind the real gate binary: ssh, scp, a refused key, and a
25 s idle session with keepalives. Bootstrapped by Matt on 2026-09-02; the first run
dropped the server's key (two `--env` flags, where `sprite-env` keeps only
the last) and the book ran open for a few minutes until the service was
recreated with one comma-separated `--env`. `/ask` is live. Phase A exit
met the same evening: `ssh chiron` from the Mac through the real gate, and
a 130 s idle session with keepalives beside a one-second clock loop on the
sprite showed 131 ticks with no gap, so an open shell keeps the sprite
awake.
- Bootstrap once from the personal account: `sprite -s chiron console`,
  install `openssh-server`, write `sshd_config` (loopback, keys only),
  register the service, install the Mac's public key. Check `/.sprite/llm.txt`
  for whether any port other than 8080 can be exposed; if a raw TCP port
  can, the tunnel is unnecessary and `sshd` listens there directly.
- Server: the `/ssh` route (decision 3), gated by an env flag so the Mac dev
  servers never carry it. Tests: refuses without the bearer key; pipes
  bytes end to end against a loopback echo listener standing in for `sshd`.
- `chiron-dev proxy` plus the `~/.ssh/config` stanza. Verify: `ssh chiron`
  from the work-account Mac with the sprite hibernated; hold the session
  idle for two minutes and confirm a background `date` loop on the sprite
  shows no gap (the keepalive proves the sprite stays awake).
- Exit: `ssh chiron` works from a cold sprite, from either CLI account.

**Phase B: the repo lives on the sprite.** Done 2026-09-03: checkout at
`/home/sprite/src/chiron` (pushed from the Mac over the tunnel, remote
`sprite`), Go 1.25 and Node 22 from the sprite's own shims, Claude Code
present, `chiron-app` installed and configured there, `make test` green
(16 packages plus the page script), and `make deploy` from the sprite
served the iPad. The evening also had two sprite outages that at first
looked like load from the test suite and were not: the platform logs
(`fly-search logs app sprite66a88ef403`) show a JuiceFS chunk missing from
object storage (404 NoSuchKey), ext4 I/O errors, and the sidecar's
checkpoint-based recovery, which once landed on a snapshot from before the
bootstrap and later held the machine in "replacing" through a sidecar
upgrade to rc48. Fly's status page had a Sprites API incident that
evening. Everything came back intact after the upgrade. The guardrails
stay because they are cheap on an 8 GB box that also serves the book:
`make test` runs `-p 2` with `CHIRON_RENDER=0` (no page images; the iPad
never uses them), one heavy job at a time, and `sprite checkpoint create`
before risky work (checkpoints v1 to v3 exist). Matt then set the
non-expiring repo-scoped PAT (`credential.helper store`) and logged Claude
Code in on the sprite; `origin` fetches, `main` tracks `origin/main`. `sprite-env` keeps only the last `--env`
flag, refuses to restart a service another `--needs`, and installs Go
binaries under `/.sprite` unless `GOBIN` is set. - Clone to `/home/sprite/src/chiron`. Toolchain: Go, Node (for
  `scripts/test-book-js.mjs`), `tmux`, Claude Code (log in once through the
  ssh session). GitHub access is a fine-grained PAT scoped to this one
  repo, Contents read/write, nothing else, one-year expiry, held by the
  sprite user's git credential store (Matt's decision, 2026-09-02). The
  rule for the agent does not change: Claude Code on the sprite never
  pushes unless asked; the sprite's `CLAUDE.md` says so.
- The sprite's `CLAUDE.md` also records that it *is* the server host: the
  live server on 8080 holds Matt's real learner state, test servers use
  drive mode on another port, and deploy means `make deploy` (build, swap
  the binary with the dated rollback name in use today, restart the
  service). The served instance stays a deploy target rather than the
  checkout (decision 1).
- Exit: `go test ./...`, the corpus lint, and the `book.js` test pass on
  the sprite; a deploy from the sprite serves the iPad.

**Phase C: the shell in the app.** Built 2026-09-02, verified in the
Simulator against a user-mode sshd behind the real gate on the Mac: the app
enrols its Ed25519 key through `POST /agent/pubkey`, opens the tunnel,
signs in with that key, and a typed command echoes back in the terminal
(`scripts/sim-verify.sh` step 05f, when `CHIRON_GATE` is set). Files:
`DeviceKey.swift`, `Tunnel.swift`, `ShellSession.swift`, `ShellView.swift`;
server `httpapi/keys.go`. Packages: SwiftTerm 1.20 (needs Xcode's Metal
toolchain and `-skipPackagePluginValidation` on the command line) and
Citadel 0.12. Untested until the sprite is bootstrapped: the real sshd,
tmux attach, and the Magic Keyboard.
- Key enrolment: on saving a server with a shared key, the app generates
  an Ed25519 keypair (CryptoKit `Curve25519.Signing`; the OpenSSH public
  encoding is `ssh-ed25519` plus the 32 raw bytes, base64) in the Keychain
  and `POST /agent/pubkey` appends it to `authorized_keys`. Possessing the
  shared key is what authorizes enrolling a device key. Keys are listed and
  revocable from the settings sheet.
- SSH client: a pure Swift SSH library (Citadel, on SwiftNIO, is the
  current candidate). Libraries connect to host:port, so the app runs a
  loopback `NWListener` and bridges each accepted connection to the
  `/ssh` WebSocket via `URLSessionWebSocketTask`. Terminal: SwiftTerm, with
  the Magic Keyboard's Ctrl, Esc, Tab and arrows passed through. A "Shell"
  icon in the top chrome opens it as a sheet or a split beside the reader.
  `tmux` on the sprite so a dropped connection resumes where it was.
- Fallback if the library fights us: a PTY-over-WebSocket route beside
  `/ssh`, with the enrolled key used for a signed challenge. Same
  enrolment, same terminal view; only the transport differs.
- Exit: from the iPad, open the shell, `claude` is running in tmux, and a
  command there changes the book the reader is holding (Phase D's verbs).

**Phase D: the agent reaches the app.** Built 2026-09-02, verified in the
Simulator: `chiron-app` on the Mac went through the gate to the book server
and the app ran `state`, `shelf`, `open`, and `screenshot` (a PNG of the page
with the purple "driven by the agent" badge). Files: `server-go/agent`
(the hub), `httpapi/agent.go` (`GET /agent/app`, `POST /agent/cmd`,
`GET /agent/status`), `cmd/chiron-app`, `devconfig` (shared with
chiron-dev); app `AppCommands.swift` (the verbs, shared with the debug
harness), `AgentLink.swift`, the Agent section in server settings. The
first screenshot after `open` can be blank while the page loads; take it a
moment later.
- The iPad has no inbound path, so the app connects out: `POST
  /agent/connect` (bearer key) registers the app instance; a WebSocket at
  `/agent/app` carries harness commands from the sprite to the app and
  results back. The existing harness verbs become the command set, plus
  `screenshot`.
- App: an "Agent" toggle in server settings, off by default, with a visible
  badge while connected, so the reader always knows when the app is being
  driven.
- Sprite: `chiron-app` CLI (`cmd/chiron-app`) with the verbs
  `scripts/sim-verify.sh` uses today, so the verify walk can run against
  the real iPad. On the sprite its config points at the book server
  directly: `url = http://127.0.0.1:8081`.
- Exit: from a Claude Code session on the sprite, walk the iPad through
  placement, series, results and a marked passage, screenshots landing on
  the sprite.

**Phase E: builds without the Mac (later).**
- An iPad app needs Xcode and macOS to build and sign; nothing on the
  sprite can produce a `.app`. A macOS runner that builds and signs on a
  push, and TestFlight for install. Until then, app changes made on the
  sprite are pulled and built on the Mac.

## 4. Security notes

- The bearer key gates the book, the tunnel, key enrolment and the agent
  channel. Keep it long, keep it in the Keychain, rotate it when the sprite
  is exposed to anything new. The tunnel alone is not a shell: `sshd` still
  demands a key.
- The sprite holds two secrets after Phase B: the Anthropic OAuth token and
  the repo-scoped PAT. A full shell on the box can read both; the PAT is
  the smaller of the two.
- The app must show when it is being driven.
- The shell path is only as safe as the iPad's Keychain; a device passcode
  is assumed.

## 5. Decisions

Settled by Matt on 2026-09-02:
- Served instance stays a deploy target; `/ssh` lives in `chiron-gate` on
  8080 with the book server behind it on 8081.
- GitHub access from the sprite: a read/write fine-grained PAT scoped to
  this repo.
- Shell transport: real SSH tunnelled over the book port; `chiron-dev` on
  the Mac, an in-app client on the iPad, one mechanism for both.

Still open:
1. Bootstrap timing: `scripts/sprite-bootstrap-ssh.sh` from the Mac with
   the sprite CLI on the personal account. It builds and pushes the gate
   and the current server (which brings `/ask` along), installs sshd and
   the Mac keys, and recreates the three services. Matt runs it; everything
   after goes through `ssh chiron`.

## 6. After this plan

Capture: primers from anything on the iPad (share extension, App Intent,
`kind` on subjects). Design in `IPAD-PLAN.md` section 12.
