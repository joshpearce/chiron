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

**Phase E: a Mac the sprite can build on.** Designed 2026-09-19. An
iPad app needs Xcode and macOS to build and sign; nothing on the sprite
can produce a `.app`. The runner is Matt's idle MacBook (macOS 27, Xcode
27), lid closed, reachable from the sprite over Tailscale. The sprite
starts the conversation, so nothing holds the sprite awake between
builds.

- **Network.** Tailscale on both ends, one personal tailnet. The MacBook
  runs the Tailscale app. The sprite runs `tailscaled` as a `sprite-env`
  service (the sprite has `/dev/net/tun`, so kernel mode, no userspace
  proxy), joined once with a tagged, pre-authorised auth key from
  1Password (`op read` at bootstrap, never on disk; the node key it
  leaves in `/var/lib/tailscale` is the sprite's own identity and stays).
  The tailnet ACL allows exactly one flow: `tag:chiron-sprite` to the
  MacBook on port 22. Nothing else on the tailnet is reachable from the
  sprite, and the sprite accepts nothing inbound. The sprite is the only
  node that ever connects; `tailscaled` idles quietly and the sprite's
  hibernation is unaffected by a DERP keepalive (to be measured at
  bootstrap; if it is not, `tailscaled` is started for the build and
  stopped after).
- **The runner.** On the MacBook, Remote Login (sshd) on, sleep off with
  the lid closed (`pmset -c disablesleep 1` on power, plus `caffeinate`
  under launchd), automatic login of Matt's user so the login keychain
  (where the signing certificate lives) is unlocked after a restart;
  FileVault therefore off on this machine, or a reboot waits for a hand.
  Xcode 27 with the Metal toolchain, `xcodegen`, the Simulator runtime,
  and the repo at `~/src/chiron` with the sprite as its only remote.
  Signing without a person: an App Store Connect API key
  (`xcodebuild -authenticationKeyPath ... -allowProvisioningUpdates`)
  rather than an Apple ID session that expires; the key file lives in
  `~/.private_keys` on the MacBook only.
- **The sprite's way in.** A dedicated Ed25519 key on the sprite
  (`~/.ssh/macbook_ed25519`), installed in the MacBook's
  `authorized_keys` with a forced command, `~/bin/chiron-runner`, that
  accepts three verbs and nothing else: `git-receive-pack` into the
  checkout (`receive.denyCurrentBranch=updateInstead`), `build <ref>`,
  and `fetch <build-id>` which streams the artefact back. A shell on the
  sprite therefore gets a build server, not a Mac. `ssh macbook` from the
  sprite is a `~/.ssh/config` stanza over the Tailscale address.
- **A build.** `scripts/mac-build.sh` on the MacBook, run by the
  runner's `build` verb: `xcodegen`, the unit suite in the Simulator
  (`sim-run.sh fast`; the UI walk is opt-in, it is minutes), then
  `xcodebuild archive` and `-exportArchive` with method `development`
  (the team's development profile carries the registered devices) to an
  IPA, plus the Catalyst app as a zip. Output under
  `~/builds/<build-id>/` with `manifest.plist`, `Chiron.ipa`,
  `Chiron-mac.zip`, `build.json` (commit, version, timestamp, test
  counts) and the log. From the sprite, `make app-build` is: push
  `main` to the MacBook, run `build`, fetch the artefacts into
  `/home/sprite/chiron/builds/<build-id>/`. A build is `CFBundleVersion`
  = the commit count on main, `CFBundleShortVersionString` = the date,
  so every build is ordered and named.
- Exit: `make app-build` on a cold sprite produces an IPA on the sprite
  from the current `main`, with the unit suite green in the MacBook's
  Simulator, in under ten minutes.

**Phase F: builds reach the devices.**
- **Serving.** The gate serves `/builds/<token>/manifest.plist` and
  `/builds/<token>/Chiron.ipa` without the bearer header, because iOS
  fetches an OTA install with its own downloader and cannot send one.
  The token is 32 random bytes per build, the directory is unlisted, and
  the IPA holds no secret (the server key lives in the Keychain, never
  in the bundle). `GET /builds/latest` (bearer) returns the build's
  version, commit, summary and the manifest URL.
- **Installing.** The app polls `/builds/latest` when it comes to the
  front and shows "Chiron 2026-09-19 (312) is ready" with **Install**;
  the tap opens
  `itms-services://?action=download-manifest&url=<manifest URL>` and iOS
  installs over the running app, keeping its data. The Mac app offers
  the zip the same way and relaunches from it. The Simulator build and
  the harness keep working exactly as now.
- Exit: a build made on the MacBook, fetched to the sprite, installs on
  the iPad and the iPhone from a tap, with nothing plugged in.

**Phase G: the request button.**
- **In the app.** "Request a change" on the shelf and in the reader: a
  text (dictation works), optionally the current screenshot and the
  harness state, `POST /dev/requests`. A **Requests** list shows each
  request's state: queued, working (with the agent's last line), testing,
  building, ready (with Install), failed (with the reason), and the
  agent's summary of what it changed.
- **On the sprite.** `chiron-dev-agent`, a `sprite-env` service, takes
  requests one at a time. For each: a worktree off `main`, Claude Code
  (`claude -p --model claude-fable-5-1`) with the request, the app's
  state and screenshot, and a brief that names the rules (TDD,
  `LESSONS.md`, Simulator before devices, no push to GitHub); `make
  test`; on green, fast-forward `main`, `make deploy` if the server
  changed, `make app-build`; the request becomes ready. The tap on the
  button is the permission for that deploy: the plan says so, and the
  agent's `CLAUDE.md` on the sprite says so. On red, the request fails
  with the agent's report and the worktree is kept for a person. One
  request at a time, because the box serves the book.
- **Merging back.** `main` on the sprite advances; GitHub does not,
  until Matt pushes with the YubiKey as today. The MacBook's checkout
  and the sprite's `main` are always the same commit after a build.
- Exit: a request typed on the iPad becomes a commit on the sprite, a
  green suite, a build, and an Install button on the same iPad, with no
  Mac opened.

**Security, phases E to G.** The sprite gains two secrets: its Tailscale
node identity and the MacBook key, and the MacBook key opens only the
runner's three verbs. The tailnet ACL is the second wall. A compromise of
the sprite through port 8080 (the bearer key) yields: the book, the
repo, the Anthropic token, the PAT, and now the ability to build the
app and offer it to the devices. The last is the new exposure: an
installed build runs on Matt's devices, so `/builds/latest` is bearer
gated, the app shows the commit it is about to install, and the runner
signs only what it built from the checkout it holds. The request
endpoint is bearer gated like everything else; the agent runs with the
same standing as the Claude Code session Matt already runs on the box.

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

Settled by Matt on 2026-09-19: builds happen on the idle MacBook over
Tailscale, reached only by the sprite (phases E to G).

Still open:
1. Bootstrap timing: `scripts/sprite-bootstrap-ssh.sh` from the Mac with
   the sprite CLI on the personal account. It builds and pushes the gate
   and the current server (which brings `/ask` along), installs sshd and
   the Mac keys, and recreates the three services. Matt runs it; everything
   after goes through `ssh chiron`.

## 6. After this plan

Capture: primers from anything on the iPad (share extension, App Intent,
`kind` on subjects). Design and status in `IPAD-PLAN.md` section 12; built
2026-09-03, device checks pending.
