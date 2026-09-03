# Chiron development on the sprite: the plan

Written 2026-09-02. The goal Matt set: the whole Chiron development
environment lives on the sprite that serves the book, and the iPad app can
open a shell on it to drive Claude Code from the iPad (Magic Keyboard in
hand). This document is meant to be executed cold: every fact needed to
start is here or in a file it names.

## 0. Where things stand

- The sprite `chiron` (Fly, org `matthew-braun`, `https://chiron.example`)
  runs the Go book server as a `sprite-env` service from `/home/sprite/chiron`,
  which is a copy of the repo's server, corpora and config, not a git checkout
  in use. Deploys are hand-pushed binaries (see the project memory
  `chiron-deployment`). The sprite has Chromium and poppler installed for page
  renders and an Anthropic OAuth token in the service environment.
- Development happens on the Mac: Xcode for the app, Go on the Mac for the
  server, the iOS Simulator for verification, `scripts/sim-*.sh` and a debug
  harness inside the app. Claude Code runs on the Mac against the repo.
- The iPad talks to the server over HTTPS with a bearer key stored in the
  Keychain per saved server.

## 1. What "development on the sprite" can and cannot mean

The server side moves cleanly: Go builds anywhere, the corpora are text, the
tests run headless, Chromium is already there. Claude Code runs anywhere a
shell does.

The app side does not move: an iPad app needs Xcode and macOS to build and
sign, and the Simulator to verify. Nothing on a Linux sprite can produce a
`.app` or install it on the iPad. Two consequences:

1. The sprite becomes the home of the **server, corpora, tests, and Claude
   Code**. The Mac remains the **build and install** machine for the app,
   driven from the same git repo.
2. To drive the *app* from the iPad without a Mac in the loop, the sprite
   needs a way to reach the app. The app's debug harness (localhost:8087 in
   the Simulator) can be extended so the running app on the iPad connects
   *out* to the sprite and takes commands from it (the reverse of today's
   direction, because the iPad has no inbound path). Claude Code on the
   sprite then has `chiron-app` commands: state, navigate, mark, ask,
   screenshot. That gives an agent on the sprite hands on the app the
   reader is holding, which is the thing Matt is after.

Builds still need the Mac. A GitHub-hosted macOS runner (or the Mac left on
at home with a runner) closes that gap later: the sprite pushes a branch, the
runner builds and signs, and the iPad installs it through TestFlight or an
ad hoc manifest. That is a phase of its own, after the shell works.

## 2. Phases

**Phase A: the repo lives on the sprite.**
- Clone the repo onto the sprite (`/home/sprite/src/chiron`), with the Go
  toolchain, Node (for the `book.js` grading test), and `xcodegen` absent by
  design. The served instance keeps running from `/home/sprite/chiron`;
  deploys become `make deploy` on the sprite: build, swap the binary, restart
  the service, with the rollback naming already in use.
- Git remote: the sprite needs read/write to the GitHub repo. Matt's rule is
  that pushes need his YubiKey; on the sprite that means either a deploy key
  scoped to this repo (his call) or the sprite pushes to a branch the Mac
  pulls. Decision needed before Phase A closes.
- Claude Code on the sprite: install, log in once (browser flow through
  `sprite exec`), and set the project's `CLAUDE.md` so the sprite session
  knows it *is* the server host (never point tests at :8080, use :8084 with
  drive mode, etc.). The memory directory conventions carry over.
- Exit: `go test ./...`, the corpus lint, and `node scripts/test-book-js.mjs`
  pass on the sprite; a deploy from the sprite serves the iPad.

**Phase B: the app reaches the sprite's agent.**
- Server: `POST /agent/connect` (bearer key) registers the app instance and
  returns a session id; a WebSocket at `/agent/app` carries harness commands
  from the sprite to the app and results back. The existing harness routes
  become the command set, plus `screenshot` (the app renders its own window
  to PNG; a UIView snapshot suffices).
- App: an "Agent" toggle in the server settings opens the WebSocket when the
  book is open, with a visible badge while connected, so the reader always
  knows when the app is being driven. Off by default.
- Sprite: a `chiron-app` CLI (Go, in `cmd/chiron-app`) that Claude Code calls:
  `chiron-app state`, `chiron-app open ai`, `chiron-app shot out.png`. Same
  verbs as `scripts/sim-verify.sh` uses today, so the verify script can run on
  the sprite against the real iPad.
- Exit: from a Claude Code session on the sprite, walk the iPad through
  placement, series, results, and a marked passage, with screenshots landing
  on the sprite.

**Phase C: the shell in the app.**
- Transport: SSH to the sprite. Sprites expose SSH through the `sprite`
  CLI's proxy today, not a public port; the plan needs a reachable SSH
  endpoint. Options, in order of preference: (1) `sprite ssh`-style access
  through the Fly proxy if sprites expose it to arbitrary SSH clients (check
  `sprite --help` and the sprites docs); (2) a WebSocket-to-PTY bridge served
  by the book server itself at `/shell` (bearer key, then a PTY running
  `claude` or `bash`), which needs no SSH at all and is the same auth the app
  already has. Option 2 is smaller and keeps one credential; option 1 is
  more standard. Decide after checking (1).
- Key push: on saving a server with a shared key, the app generates an
  Ed25519 keypair in the Keychain and `POST /agent/pubkey` installs it in the
  sprite user's `authorized_keys` (the server does the write, gated by the
  bearer key). The passphrase Matt mentioned is that shared key: possessing
  it is what authorizes enrolling a device key. Keys are listed and revocable
  from the settings sheet.
- Terminal UI: a terminal emulator view (SwiftTerm is the established Swift
  package; iOS 26 has no system terminal view) in a sheet or a split beside
  the reader, with the Magic Keyboard's keys passed through (Ctrl, Esc, Tab,
  arrows). A "Shell" icon in the top chrome opens it. Claude Code runs in a
  `tmux` session on the sprite so a dropped connection resumes where it was.
- Exit: from the iPad, open the shell, `claude` is running, ask it to change
  the app's state through `chiron-app`, and watch the reader view change
  behind the sheet.

**Phase D: builds without the Mac (later).**
- A macOS runner that builds and signs on a push, and an install path to the
  iPad (TestFlight is the clean one; ad hoc over HTTPS is the quick one).
  Until then, app changes made on the sprite are pulled and built on the Mac.

## 3. Security notes

- The bearer key becomes the root of everything: it already gates the book,
  and in this plan it gates enrolling SSH keys and driving the app. Keep it
  long, keep it in the Keychain, and rotate it when the sprite is exposed to
  anything new. The `/agent/*` routes must refuse unauthenticated calls
  exactly as `/exchange` does today.
- The app must show when it is being driven. A remote agent moving the
  reader's screen without a visible indicator is a trust failure.
- The shell is a full shell on the box that holds the Anthropic token. The
  SSH key path is only as safe as the iPad's Keychain; a device-level
  passcode is assumed.

## 4. Open decisions for Matt

1. Git write access from the sprite: deploy key, or branch-and-pull.
2. Shell transport: SSH through Fly's proxy if available, otherwise the
   server's own PTY-over-WebSocket.
3. Whether the served instance should become the git checkout itself
   (simpler, but a bad commit takes the book down) or stay a deploy target.
