# Chiron

Chiron is an adaptive textbook that teaches from whatever you want to learn
about. It writes books and primers to order, reads the things you already
wanted to read - imported EPUBs, PDFs, web pages, blogs you follow - and keeps
all of it on one shelf. The point of putting them together is the highlight: a
passage you mark in anything on that shelf reaches the tutor with the whole
chapter behind it, so the answer is about the argument you are reading rather
than a sentence torn out of it. What it writes for you is shaped by what you
have read and answered, which it remembers.

A server holds the material and the learner record; apps for iPad, iPhone and
the Mac read from it and sync, so a chapter you mark up on one device is marked
up on the others. Reading works with the server out of reach, and on recent
Apple hardware a question asked offline is answered by the on-device model from
the chapter alone rather than not at all.

<p align="center">
  <img src="ipad-app/Chiron/Assets.xcassets/Logo.imageset/logo.png" width="180" alt="">
</p>

## What it looks like

| | |
|---|---|
| <img src="docs/screenshots/library.png" alt="The library: two authored books, a followed blog with unread posts, and a web page saved to read"> | <img src="docs/screenshots/reading-a-page.png" alt="A web page read as a chapter, with the pen and highlight palette"> |
| **The library.** Books it wrote, blogs followed by feed, pages and documents saved to read - all as cards on one shelf. | **A page read in Chiron.** Any URL comes in as its own article, with its code and pictures kept, ready to highlight. |
| <img src="docs/screenshots/capture.png" alt="The capture card: captured text, a question field, and how much to ask for"> | <img src="docs/screenshots/placement.png" alt="The placement screener that runs before a book begins"> |
| **Capture.** Highlight anything, ask a question, and say how much you want back: a summary, more detail, a primer, or a whole book. A shared link can instead be read here or followed as a blog. | **Before a book begins.** A short screener places you, so the first chapter is written for the reader you actually are. |

## How it works

You feed it something - a passage, a file of notes, a link, a question - and
say how much you want back. A **summary** or a bit more **detail** comes back
in the card. A **primer** or a **book** goes through a short planning
conversation first, then a brief, and then a model writes it; it lands on the
shelf as real chapters with exercises and checks. Books are taught rather than
just displayed: you read a chapter, work the beats inside it, take a
cumulative check, and what comes next depends on how that went.

Everything else on the shelf is read as it is, with nothing invented on top. An
EPUB keeps its own chapters, a PDF its pages, a web page its article, a
followed blog its posts - and all of them accept highlights, questions, and
Pencil annotations that sync between devices.

The same server can be driven from a shell (`chiron capture`, `chiron page`,
`chiron follow`), which is how an agent does a piece of research and hands the
result to the shelf.

## Why it is built this way

The design follows the evidence on what actually moves learning, which is not
what an "AI textbook" usually does.

**Interaction lives inside the chapter, not after it.** A chapter-terminal quiz
is answer-based tutoring, the weakest form. The gains are in step-level work, so
every chapter carries 6-10 *beats* - predict the next step, fill in a blanked
derivation, explain why a term is there - that you commit to before the reveal.

**Wrong models are the teaching material.** Every unit ships a misconception
bank: the wrong model, why it is appealing, and *the specific prediction it
makes that observably fails*. Chapters are written in refutation structure and
every multiple-choice distractor cites a bank entry, so a wrong answer is a
diagnosis rather than a miss.

**Remediation switches representation.** Failing a gate does not re-present the
same text more slowly - that is the documented failure of mastery programs. It
swaps symbolic for numeric, or geometric, or code.

**The gate is a default, not a wall.** Below 80% you are held, but you can
always override. Overriding accrues *knowledge debt*, and "catch me up" later
generates one consolidated chapter covering everything skipped.

**Confidence is recorded before every reveal.** High-confidence-wrong is a
misconception and gets refuted; low-confidence-right is fragile and comes back
as a callback. Checks are cumulative - roughly 60% current unit, 40% earlier
material weighted toward what you got wrong.

**The model supplies judgement, code supplies everything else.** Mastery
updates, the prerequisite graph, gating, debt and check composition are
deterministic. What gets asked decides what you retain; it should not vary with
a model's mood.

UI construction principles - each learned from a real failure - live in
[DESIGN-ui.md](DESIGN-ui.md).

## Shape

```mermaid
flowchart LR
    subgraph Clients["iPad · iPhone · Mac"]
        R[Reader<br/>WKWebView + KaTeX]
        C[Capture and checks<br/>confidence before reveal]
        K[(kept on device:<br/>chapters, pictures, marks)]
        D[on-device model<br/>when the server is away]
    end
    subgraph Server["chiron-server (Go), on a Fly sprite"]
        E[/exchange · capture · readings · feeds/]
        RO[grader / planner / author]
        ST[(event-sourced<br/>learner state)]
    end
    subgraph Engines
        L[LM Studio<br/>local, offline]
        CC[claude -p<br/>subscription]
        A[Anthropic API]
    end
    R --> C --> E
    E --> RO --> ST
    RO --> L & CC & A
    K -.-> R
    D -.-> R
```

Reading is detached by design. Chapters, their pictures and the reader's own
marks are kept on the device, so a book opens and turns with no server at all;
what happened goes up when there is one again. When a question is asked with
the server out of reach, Apple's on-device model answers it from the chapter
alone and says that is what it did.

## Layout

| Path | What it is |
|---|---|
| `corpus/`, `corpus-v2/` | Authored subjects: units with canon, three depth variants, a question bank and unit-local misconceptions |
| `server-go/` | The service, and the `chiron` command that drives it from a shell. One static binary |
| `server/` | The original Python implementation, kept as the reference the Go port was proven against |
| `ipad-app/` | SwiftUI client for iPad, iPhone and the Mac (Catalyst), built with xcodegen |
| `skills/chiron/` | The agent skill: how a coding agent sends work to the shelf and reads it back |
| `scripts/` | Simulator and Mac test runners, dev servers, sprite deployment |
| `IPAD-PLAN.md` | What the app does and what is planned, section by section |
| `DESIGN-v2-sprite.md`, `SPRITE-DEV-PLAN.md` | The hosted-tutor design, and moving development onto the sprite |
| `LESSONS.md` | Everything that cost more than ten minutes to learn, with the fix |

## Getting set up

### What you need

| | For what | Notes |
|---|---|---|
| macOS with Xcode 26+ | The iPad, iPhone and Mac apps | The server alone needs no Mac |
| Go 1.26+ | The server and the `chiron` command | |
| [XcodeGen](https://github.com/yonaskolb/XcodeGen) | Generating the Xcode project | `brew install xcodegen`; the project is not tracked |
| A model to write with | Everything the tutor authors | One of: [LM Studio](https://lmstudio.ai) running locally (free, offline), an Anthropic API key, or Claude Code on a subscription |
| A [Fly.io](https://fly.io) account with sprites | Hosting the server so devices reach it anywhere | Optional - a laptop on the same Wi-Fi works |
| An Apple Developer account | Installing on your own iPhone or iPad | Free accounts sign for 7 days; a paid one is needed for over-the-air installs |
| [Tailscale](https://tailscale.com) and a spare Mac | The self-building loop, where the app builds itself | Optional, and the last thing to set up |
| The [1Password CLI](https://developer.1password.com/docs/cli/) | The sprite scripts, which read tokens by reference | Optional - they take any `op://` reference you give them |

### Run it on your own machine

```bash
git clone <this repo> && cd chiron
cd server-go && go build -o ../bin/chiron-server ./cmd/chiron-server
cd ../server && ../../bin/chiron-server -addr 0.0.0.0:8080 -config config.yaml
```

Point `server/config.yaml` at a model before the tutor can write anything:
leave `provider:` unset for any OpenAI-compatible endpoint (LM Studio's default
is already in the file), set `claude-cli` to use Claude Code on a subscription,
or `anthropic` with `ANTHROPIC_API_KEY` in the environment. `CHIRON_PROVIDER`
overrides the file without editing it.

Bind to `0.0.0.0`, not loopback: a tablet cannot reach a loopback bind, and it
presents as the app hanging rather than as a server problem.

### Build the apps

```bash
cp signing/local.config.example signing/local.config
# Fill in your team, unique bundle ID, and matching App Group.
bash scripts/xcodegen.sh              # generates ipad-app/Chiron.xcodeproj
./scripts/sim-run.sh                  # iPad Simulator
./scripts/mac-run.sh                  # the same app on the Mac
```

`signing/local.config` is ignored. XcodeGen, the simulator scripts, the
Catalyst build, the share extension, and the unattended build all read the
same values through `scripts/signing-config.sh`. Signing remains automatic:
do not put certificate names or provisioning profiles in the project. Xcode
selects platform-appropriate assets for the configured team.

After changing the team or bundle ID, regenerate the project before building.
The app's Keychain access group is derived from that signing identity; an old
installed build cannot retain a newly entered server key if its provisioning
profile does not include Keychain Sharing.

On first launch, open **Server** (the gear) and add the address the server is
listening on. A second device takes the same settings from a QR code, or from a
`chiron://server?...` link pasted into **Paste a setup link**.

### Host it on a sprite

A [sprite](https://fly.io/docs/sprites/) is a persistent Fly microVM, which is
what makes the server reachable from a phone on cellular data without running a
laptop at home.

```bash
scripts/sprite-bootstrap-ssh.sh       # ssh in, a checkout, the landing shell
scripts/sprite-set-key.sh             # the shared client key, from stdin
scripts/sprite-set-auth.sh            # the model token, from stdin
make deploy                           # build, copy, restart, verify
```

Both key scripts read from standard input, so pipe them from wherever you keep
secrets (`op read 'op://<vault>/<item>/credential' | scripts/sprite-set-key.sh`)
rather than typing them where a shell history can catch them. The server
listens on loopback behind `chiron-gate`, which is what the public URL reaches.

### The self-building loop

The last piece, and entirely optional: a change-request button in the app that
ends with a new build on your devices. It needs a Mac with Xcode that stays
awake, reachable from the sprite over Tailscale, and an App Store Connect API
key so it can sign without a signed-in Xcode:

```bash
scripts/sprite-tailscale.sh <mac-tailnet-name> <user>   # the sprite joins your tailnet
scripts/mac-runner-bootstrap.sh <sprite-pubkey>         # on the build Mac, once
make deploy-agent                                       # the agent that takes requests
```

`SPRITE-DEV-PLAN.md` has the whole design, including what each half is trusted
to do. Nothing else in this repo depends on it.

## Checking a corpus

```bash
cd server-go && go build -o ../bin/corpus-lint ./cmd/corpus-lint
../bin/corpus-lint ../corpus
```

## Tests

```bash
cd server-go && go test ./...                       # unit and API tests
cd server && .venv/bin/python test_sim_learners.py  # simulated learners
CHIRON_TEST_BASE=http://host:8080 .venv/bin/python test_sim_learners.py
```

For the persistent Incus system container on `orca`, including the exact
filesystem, service, Claude subscription login, health-check, and backup
contract, see [docs/ORCA-OPERATIONS.md](docs/ORCA-OPERATIONS.md).

The simulated learners are the interesting ones: a strong learner who should
clear gates and unlock extensions, a weak learner who should be held with the
representation actually switched, and a skipper who should accrue debt and get a
coherent catch-up chapter. `CHIRON_TEST_BASE` points them at any running server,
which is how the Go port was checked against the Python one.

The mechanical grading rules are duplicated in three places - Go, Python, and
the offline JavaScript grader - because a beat is graded in the browser while
reading and on the server at the boundary. `checkers_test.go` and
`test_checkers.py` pin that contract. If you change one, change all three.

## Generating a new subject

In the app: **Library -> "Teach me something else"**. It asks until it can
write a brief, then plans a syllabus and authors every unit against
`corpus/authoring-spec.md`, which is subject-agnostic. `/teach/jobs` reports
progress; the subject appears in the library once every unit exists.

From a shell, the same pipeline in stages:

```bash
cd server
CHIRON_PROVIDER=claude-cli .venv/bin/python generate_subject.py plan  <slug> "<brief>"
CHIRON_PROVIDER=claude-cli .venv/bin/python generate_subject.py units <slug>
../bin/corpus-lint ../corpus-<slug>
```

Budget around 30 minutes per unit. Units are written one file per call and skip
files already on disk, so an interrupted run resumes rather than re-paying -
asking for a whole unit in one response produced a 30k-token reply that ran past
ten minutes and failed as a unit.

Output lands in `corpus-<slug>/` and the server discovers it at boot; there is
no config to edit, and a half-generated corpus stays invisible rather than
appearing as a book with missing chapters.

**A generated subject is unverified prose until someone reads it.** `corpus-lint`
covers the mechanical half - tolerances that would accept the error an item
exists to catch, answers a program cannot compare, distractors citing
misconceptions that do not exist, depth headings drifted from canon. The
judgement-heavy half (is this claim true? does this rubric adjudicate?) has no
substitute yet.

## Status

Working and used. The full loop has run end to end on real hardware against both
a local model and a hosted one. See `FLIGHT.md` for what is verified, what is
not, and the failures that cost real time - the tolerance rule that accepted
answers double the correct one, the markdown escaping that corrupted every
equation, the flag that made a model discard its own work.

## Licence and credits

The code and the artwork are under the MIT licence; see [LICENSE](LICENSE).
The bundled third-party material keeps its own terms:

- KaTeX, under the MIT licence, in `ipad-app/Chiron/Resources/katex/`.
- Source Sans and Source Serif, under the SIL Open Font Licence, with its text
  beside them in `assets/fonts/`.
- The centaur logo and app icon were generated for this project and are
  covered by the licence above.

Nothing committed in the repo carries an account of its own: Apple team and
identifier values come from ignored `signing/local.config` (or equivalent
environment variables) at generation time, your server's address from the app's
settings screen and `~/.config/chiron-runner/config`, and every 1Password
reference from an environment variable.
