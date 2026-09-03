# Chiron

An adaptive textbook: a Go server (`server-go/`), two corpora (`corpus/`
is the subject `ai`, `corpus-v2/` is `data`), and an iPad app
(`ipad-app/`, iOS 26, built only on a Mac with Xcode).

## If this session is on the sprite

You are on the machine that serves the book. Facts that matter:

- The live server is `chiron-server` on `127.0.0.1:8081`, behind
  `chiron-gate` on `0.0.0.0:8080` (the public URL). Its state under
  `/home/sprite/chiron/state` is Matt's real learner record. Never point
  a test or a script at 8080 or 8081.
- A throwaway server for experiments: from `server-go`,
  `CHIRON_DRIVE=1 CHIRON_TRANSCRIBE=stub go run ./cmd/chiron-server -addr :8084 -config <a copy of config.yaml with its own state dirs>`.
  Drive mode grades mechanically, serves cached chapters and stubs
  handwriting, so no model budget is spent.
- Deploy is `make deploy` from this checkout; it keeps a dated copy of the
  previous binary in `/home/sprite/chiron/bin` for rollback. `make
  deploy-gate` replaces the way in; do it from tmux.
- The iPad app is driven with `chiron-app` (`chiron-app state`,
  `chiron-app open subject=ai`, `chiron-app shot page.png`); it works only
  while the reader has the agent switched on in the app's server settings.
- Services are `sprite-env services ...`; logs under `/.sprite/logs/services/`.
- The sprite has 8 GB and serves the book while you work. Run one heavy
  thing at a time (`make test` uses `-p 2`; do not run a build beside it).
  On 2026-09-03 a full-parallel test run beside a `go install` took the
  sprite down, and it came back restored to a snapshot from before the
  bootstrap: half an hour of writes gone. Before anything risky:
  `sprite-env checkpoints create`.
- Never push to GitHub unless Matt asks in the conversation, even though
  the credential here allows it.

## Everywhere

- Tests: `make test` (Go and the page script); corpus lint: `make lint`.
- App tests and Simulator walks need the Mac: `scripts/sim-run.sh test`,
  `scripts/sim-verify.sh`.
- Plans and status: `IPAD-PLAN.md`, `SPRITE-DEV-PLAN.md`.
