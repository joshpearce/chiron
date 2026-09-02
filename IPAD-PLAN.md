# Chiron for iPad: the plan

Written 2026-09-01 after the Paper Pro turned out to be the wrong device for
this book. This document is meant to be executed cold after a context
compaction: every fact needed to start is here or in a file it names.
Research behind the design choices: `ipad-app/RESEARCH-ipados.md`.
Design lessons that carry over from the e-ink client: `DESIGN-ui.md`.

## 0. Where things stand (measured, not assumed)

**There is already an iPad app.** `ipad-app/` is a SwiftUI app (xcodegen
project, `project.yml`, deployment target iOS 15.0, iPad only, bundle id
`dev.mjbraun.chiron`, signed by the Fly.io team `${CHIRON_TEAM_ID}`). About 2,300
lines in eleven files:

| file | what it owns |
|---|---|
| `App.swift` | `ChironApp`, `ContentView` (screen switch + top-right chrome), `LibraryView`, `StartView`, connection settings/server editor, `ConnectionBadge`, `GeneratingOverlay` |
| `AppModel.swift` | the state machine (`Screen` enum: menu/start/reading/pretest/check/gate/takingBreak/teach), per-subject persistence in Documents, the exchange flow (`run`), start-over, static offline book |
| `Models.swift` | Codable wire models: `ChapterPayload`, `CheckItem` (+`Reveal`), `ItemResponse`, `BeatResponse`, `SubjectInfo`, `ExchangeRequest/Response`, `GradeResult`, `Gate`, `BookState`, `SpineEntry`, `BreakSuggestion`, `JSONValue` |
| `Sync.swift` | HTTP exchange with bearer auth, 600 s timeout, `CHIRON_SERVER` launch-env override, plus a USB-bridge listener on :8081 (flight-era, see roadmap) |
| `ReaderView.swift` | `ReaderContainer` (bottom bar: Catch me up / Take the check / Skip) + `ReaderView` (`WKWebView` loading `Resources/chapter.html`, JS bridge `bridge`, `initChapter(json)`) |
| `MathText.swift` | prompts/options with `$` math render in a per-view `WKWebView` with KaTeX, height measured by JS; plain text otherwise |
| `CheckViews.swift` | `ItemFlowView` (one item at a time, confidence slider BEFORE reveal, commit, "I don't know", per-item reveal), `GateView`, `BreakView`, `SpineView` (sheet, progress, debt, start-over) |
| `Servers.swift`, `Credentials.swift` | saved servers list, Keychain-held shared key |
| `SelfTest.swift` | launch-argument harness: `selftest`, `selftestweak`, `showreader|showcheck|showpretest|showbreak [unit]`, `showteach [demo]`, `server=URL` |
| `TeachView.swift` | "Teach me" conversation (v2 feature; untouched by this plan) |
| `Resources/` | `chapter.html`, `book.css` (paper + dark themes, 19px serif, 44em column), `book.js` (chapter render, beats, mechanical grading mirror of `checkers`), `katex/` (folder reference - must stay a folder or math dies), `default-book.json` (bundled offline book) |

**It builds today** on Xcode 26.6 for the iOS 26.5 simulator (`chiron-ipad`,
device type iPad (A16)). `scripts/sim-run.sh` builds, installs, launches with
`SIMCTL_CHILD_CHIRON_SERVER`, and screenshots.

**Baseline run on 2026-09-01** (fresh two-subject server on :8084, app
launched with `selftest`):

1. `/subjects` decode fails silently and falls back to one built-in subject.
   Cause: the response gained a top-level `active` string (bookshelf work)
   and the app decodes `[String: [SubjectInfo]]`. Client fix, Phase 0.
2. The screener exchange failed to decode: MCQ reveal options omitted
   `"correct": false`. Server bug from the Go port, fixed in `e8b9403`
   (`corpus.RevealOption` no longer `omitempty`), regression test in
   `render/render_test.go`. Real responses saved as fixtures:
   `ipad-app/fixtures/exchange-start.json`, `exchange-screener.json`.
3. After that fix the app reaches the calibration series, but the screener
   is presented as a free-text item (it is `kind: constructed`, `check:
   screener`, with `options`), and `SelfTest` expects a gate after every
   check - the gateless screener step postdates it.
4. Everything the tablet client learned to do since July is absent: native
   placement screen, calibration-series presentation, typeset results with
   READ AS / WHY, contents page, bookshelf, reopen-on-last-book, break
   flows tied to server pacing, error/loading copy, "Start over" from the
   book, authoring wait.

**Server protocol the iPad uses** (unchanged, synchronous, JSON):
- `GET /subjects` -> `{subjects: [{id,title,units_total,units_cleared,current_unit,debt}], active}`
- `POST /exchange {subject, phase: start|boundary|pretest, unit, check_responses[{item_id, response|selected_index, confidence, idk}], pretest_responses, beat_responses, override, skipped_check, catch_me_up, choice, chunk_minutes, break_minutes}`
  -> `{results[{item_id, verdict, feedback_md, misconceptions, confidence}], gate|null, chapter|null, state, break_suggestion|null, results_pages?}`
- `chapter`: `{unit, title, minutes, html, beats[], pretest[], check[], calibration, next_action}`;
  check items: `{id, unit, concept, kind: constructed|mcq, prompt, check: llm|exact|numeric(t)|choice|screener, difficulty, options[{text}], reveal{answer,rubric | options[{explain,correct}]}}`
- `GET /state?subject=` -> `{spine[{unit,title,status,score,in_fringe}], fringe, debt, active_misconceptions, summary, session_minutes, llm}`
- `POST /reset {subject, confirm: true}`; `GET /health`, `GET /ping`
- `GET /pages/{subject}` (meta) marks the subject active server-side; the
  iPad does not use rendered pages, so it needs another way to mark the
  book open (Phase 1, server change A).
- The ink path (`POST /ink/{subject}`) is async-authoring; the JSON path is
  not. A graded exchange can take a couple of minutes while Opus writes the
  next chapter.

**Devices.** Test iPad: iPad mini 4 (iPad5,1), iPadOS 15.8.8, 7.9" at
768x1024 pt (@2x), A8, 2 GB RAM, **no Apple Pencil support** (so no
Scribble, no pencil-only ink). Full screen it is regular x regular; in Split
View it is compact. Signing/device notes are in the project memory
(`dynamic-book-project.md`): profiles from the Fly team last a year; the
device must be registered in the team; `ios-deploy` and `idevicesyslog` are
installed.

**Toolchain.** Xcode 26.6 (17F113), only the iOS 26.5 runtime installed.
Apple's runtime index still offers iOS 15.5 for Xcode <= 26.99; Xcode 27
will probably drop it. `xcodegen`, `idb` + `idb_companion` are installed.
macOS has no `timeout`; use a background process + `kill`.

## 1. Goals and non-goals

Goals, in priority order:
1. The iPad is the primary reading device for both books, with everything
   the Paper Pro client can do, done the iPad way.
2. Runs on the mini 4 (iOS 15.8) and stays correct on current iPadOS.
3. All development and verification in the Simulator, driven by scripts,
   with an iOS 15.5 simulator as the gate and a device pass at milestones.
4. Multiple books in progress is designed for now, built later (section 8).

Non-goals for this plan: the USB bridge and Wi-Fi soft-AP transports
(flight era; delete when convenient), Teach-me changes, the bundled offline
book (keep it loading, do not extend it), App Store anything.

## 2. Design principles for the iPad client

Carried over from `DESIGN-ui.md` (they were learned the hard way):
- Anything with typography or math is rendered content; anything the
  learner operates is a native control producing exact data. Screener and
  confidence and MCQ choice are controls; prompts are typeset.
- Every answer surface has "I don't know", one action, never punished,
  always in the same place (leftmost).
- Confidence is captured before any reveal; the UI makes peeking
  structurally impossible.
- Buttons that fire an exchange drop duplicate taps while one is in flight.
- Controls live inside the question box they answer.
- Interpreted answers (handwriting) keep their evidence beside the grade.

iPad-specific, from the HIG research:
- Readable column: cap prose at about 680 pt and center it; body text
  around 19-20 pt serif with generous leading; support Dynamic Type up to
  200 percent by feeding the web view a CSS variable from
  `UIFontMetrics`, and reflow rather than zoom.
- Two explicit reading themes, Paper (warm, the current `book.css`) and
  Dark, following the system appearance by default; chrome uses semantic
  system colors so it is right in both.
- Regular width: contents (spine) as a sidebar beside the reader; compact
  width (Split View, Slide Over): contents as a pushed list. Design the
  reader for a 320 pt floor.
- Continuous scroll for chapters (Books' Scroll mode); no page-curl. Tap in
  the middle toggles chrome; the reader is the scroller (never a web view
  inside a SwiftUI ScrollView).
- Keyboard first for answers on this device (no Pencil); hardware keyboard
  shortcuts: number keys pick an MCQ letter, Return commits, Cmd-Return
  checks in, `[`/`]` previous/next item. Pointer hover effects on buttons.
- All four orientations; `UIRequiresFullScreen` stays false; save state on
  `.inactive` and `.background`.
- Typography: keep Source Serif 4 / Source Sans 3 (bundled in
  `assets/fonts`, the e-ink book's faces) for content and controls so the
  two clients read as one book; SF for system chrome only.
- Accessibility: every control labeled; MathText web views expose an
  accessibility label with the plain prompt; contrast 4.5:1 minimum; test
  with `simctl ui content_size accessibility-extra-large` and Increase
  Contrast.

## 3. Screens and navigation (what to build)

Mapping the tablet client's modes onto iPad screens:

| tablet mode | iPad screen | notes |
|---|---|---|
| books | **Bookshelf** (`LibraryView` reborn) | one card per subject from `/subjects`: title, progress line, "Open now" marker on the active book; opens straight into the reader when a chapter exists; server-setting sheet stays here |
| contents | **Contents** sidebar/list | from `/state` spine: status glyph, score, tap to browse a cleared chapter read-only, current chapter resumes; "Start over" and "Catch me up" live here |
| reading | **Reader** | `WKWebView`, continuous scroll, chrome on tap; bottom bar: item count, "Take the check" / "Answer the questions"; beats inline as today |
| screener | **Placement** | native: the question, five rows, one tap, advances immediately (gateless) |
| reading (calibration series) | **Calibration series** | the series items one per screen, no per-item reveal (it is measurement); commit or IDK; progress "3 of 11"; ends in results |
| results | **Results** | native: headline (Calibration complete / Gate cleared / Below the gate + score), gate bar, per-item entries with READ AS / ANSWER / WHY typeset via MathText; actions: Continue / Explain it differently / Override |
| screener->series, series->chapter | **Authoring wait** | "The next chapter is being written" static screen; see server change B |
| breakSuggested / breakActive | **Break** | existing `BreakView`, wired to `break_suggestion` and `break_minutes` |
| confirmReset | **Start over** confirmation | existing alert, reachable from Contents |
| loading / submitting / error | **Wait / Error** | same copy as the tablet: "Opening the book.", "Grading your answers.", "The server can't be reached." with Try again |

Navigation on iOS 15: `NavigationView` with `.navigationViewStyle(.stack)`
in compact width; a `UISplitViewController` representable (sidebar =
Contents, secondary = Reader) in regular width. Modal flows (placement,
checks, results, break) present with `fullScreenCover` so the reader cannot
be peeked at mid-check. Deep links `chiron://book/<subject>/<screen>` route
to any screen (used by the harness and by the multi-book future).

## 4. Architecture inside the app

- Keep SwiftUI + `ObservableObject` (`@Observable` is iOS 17). Keep the
  `AppModel` state machine but split it: `BookSession` (per-subject: chapter,
  responses, position, persistence) and `Library` (subjects, active book,
  servers). Multiple `BookSession`s can exist; the UI shows one. That is the
  multi-book foundation and costs nothing now.
- Persistence: per-subject JSON in Application Support (move from
  Documents; mark caches excluded from backup); reading position debounced
  from the web view's scroll offset and top visible section id; save on
  scene phase `.inactive`/`.background`.
- Networking: one `ChironClient` with typed endpoints and `URLSession`
  async; long timeout for exchanges; a `busy` flag that disables all
  exchange buttons.
- Reader: one `WKWebView` per chapter, `loadFileURL` of the bundle template,
  chapter HTML injected via `initChapter`; a `WKUserScript` reports the
  scroll position and the visible section; CSS variables for font scale and
  theme are set from Swift on load and on change.
- MathText: keep, but pool web views per screen (one item on screen at a
  time in the item flow, so the cost is bounded) and give each an
  accessibility label. Consider `SwiftMath` later only if A8 web views prove
  slow (measure first, Phase 5).
- Debug harness (`#if DEBUG`, launch arg `harness`): an `NWListener` HTTP
  server on localhost:8087 with `GET /state` (screen, subject, unit, item
  index, error), `POST /navigate` (deep link), `POST /answer` (fill the
  current item), `GET /shot` (writes a screenshot via the app if useful; the
  shell uses `simctl io` anyway). This replaces the ad-hoc `show*` launch
  args and lets a script drive every screen without taps. Taps that must be
  real taps use idb.

## 5. Server changes (small, all TDD in Go)

- **A. Mark the book open from the JSON path.** Only `GET /pages/{subject}`
  calls `markActive` today (verified: `handleExchange` never does). Add
  `s.markActive(sub.ID)` in `handleExchange` after the subject lookup so
  opening a book on the iPad makes it the active one.
  Test: after an exchange on `data`, `/subjects.active == "data"`.
- **B. Async authoring for the JSON path.** Add `async: true` to
  `Exchange`; when set, `processExchange(..., asyncAuthor=true)` returns the
  grades immediately with `authoring: "<unit>"`. Add `GET
  /chapter/{subject}` returning the persisted current chapter JSON
  (`currentChapter(sub)` already exists for pages) plus `{authoring,
  authoring_error}` from `buildStatus`, so the iPad polls it the way the
  tablet polls pages meta. Tests: exchange with `async` returns no chapter
  and an `authoring` unit; `/chapter` reports authoring then the chapter.
- **C. Results doc in the response.** `buildResultsDoc` already produces
  `{headline pieces, dek, entries[{n, verdict, idk, kind, prompt,
  confidence, read_as, chose, answer, why}]}` for the typeset pages; add
  `out["results_doc"] = doc` so the iPad renders the same content natively.
  Test: a graded exchange carries `results_doc` with one entry per item.
- Nothing else. The screener, calibration sets, bands, reveal, and state
  payload are already what the iPad needs.

## 6. Dev loop (Simulator for everything)

One-time setup, done 2026-09-01. `xcodebuild -downloadPlatform` refuses
iOS 15.5 ("not available for download"): Apple's index lists it only as a
legacy installer package, not a runtime image. What worked:
```
# index: https://devimages-cdn.apple.com/downloads/xcode/simulators/index2.dvtdownloadableindex
curl -L -o ios15.5-sim.dmg https://devimages-cdn.apple.com/downloads/xcode/simulators/com.apple.pkg.iPhoneSimulatorSDK15_5-15.5.1.1653527639.dmg   # 5.4 GB
hdiutil attach -nobrowse -readonly ios15.5-sim.dmg
pkgutil --expand-full /Volumes/Clearwater*/iPhoneSimulatorSDK15_5.pkg sdk15   # 12 GB, minutes
mv sdk15/Payload ~/Library/Developer/CoreSimulator/Profiles/Runtimes/"iOS 15.5.simruntime"   # no sudo
xcrun simctl create chiron-ipad-15 com.apple.CoreSimulator.SimDeviceType.iPad-mini-4 com.apple.CoreSimulator.SimRuntime.iOS-15-5
```
`chiron-ipad-15` (iPad mini 4 device type, 768x1024 pt, iOS 15.5) is the
gate; `chiron-ipad` (iPad A16, iOS 26.5) is the second target. Both must
pass before a phase closes. Quirk: that runtime cannot delta-install over an
existing copy of the app; `sim-run.sh` uninstalls and reinstalls when the
install fails, which wipes the app's Documents on that device.

Scripts (extend, do not fork):
- `scripts/sim-run.sh [device]` - build, install, launch against the
  disposable server; `DEVICE=chiron-ipad-15` selects the runtime; `shot`
  screenshots. Never point it at :8080 (real state) or :8082 (Matt's sim
  server); use `scripts/sim-server.sh` or a throwaway server on :8084 with
  `state_dir`s under the scratchpad, both subjects configured.
- `scripts/sim-verify.sh` (new) - for each device x {light, dark} x {default,
  accessibility-extra-large}: launch with `harness`, walk every screen via
  `/navigate` and the fixtures, screenshot each to `verify/<device>/<theme>/
  <screen>.png`, and print `/state` after each step. Reviewed by eye; the
  harness output is asserted.
- `xcodebuild test` runs the unit tests on both simulators.
- Device pass: `ios-deploy --bundle ... --justlaunch` + `idevicesyslog |
  grep chiron`, at the end of Phases 2, 3 and 5.

Tests to add (a `ChironTests` XCTest target in `project.yml`):
- Decoding: every fixture in `ipad-app/fixtures/` decodes; add a fixture
  for a graded check response, a break suggestion, `/subjects` with
  `active`, `/state`, and the async `authoring` shape as they are captured
  from the real server.
- State machine: `BookSession` transitions with a fake client (start ->
  placement -> series -> results -> reading; gate fail -> remediate /
  override; break; error recovery; reopen restores position).
- Mechanical grading mirror: `book.js` must keep matching `checkers`
  (already covered server-side by `checkers` tests; add a JS test runner
  only if `book.js` changes).

## 7. Phases

Each phase ends with: unit tests green on both simulators, `sim-verify.sh`
screenshots reviewed, a commit. TDD for every behavior change (failing
test first), including the Go changes.

**Phase 0 - Toolchain and baseline (half a day)**
- Install the iOS 15.5 runtime, create `chiron-ipad-15`, parametrize
  `sim-run.sh`, add the `ChironTests` target and the fixture decode tests.
- Fix `/subjects` decoding (struct with `subjects` + `active`).
- Replace `SelfTest.run` with a calibration-aware loop: start -> screener
  (selected_index) -> series (answer from reveal) -> gate present -> PASS.
- Server change A. Exit: self-test passes on both simulators against a
  throwaway server for both subjects.

**Phase 1 - Protocol parity with the tablet**
- Placement screen (native, five rows, IDK-less by design, one tap).
- Calibration series presentation (no reveal, progress, IDK leftmost).
- Results screen from `results_doc` (server change C), replacing the
  feedback wall in `GateView`; actions per gate state.
- Authoring wait with polling (server change B); "Grading your answers"
  wait; error screen with Try again; the exact copy the tablet uses.
- Bookshelf from `/subjects` with the active book marked; reopen on it at
  launch; Start over from Contents; break suggestion and break-taken
  reporting wired to the server.
- Exit: a full pass through v0 -> v1 on `data` and u0 -> u1 on `ai`, driven
  by the harness, screenshots reviewed.

**Phase 2 - iPad shape**
- Split view (sidebar Contents / Reader) in regular width; stack in compact;
  chrome toggle on tap; reader position persistence and restore.
- Typography pass: fonts, readable column, Dynamic Type via CSS variable,
  Paper/Dark themes, dark-mode audit of every screen.
- Keyboard shortcuts and pointer hover; all orientations; Split View by
  hand in the simulator.
- Exit: `sim-verify.sh` at both text sizes, both themes, both devices; a
  device pass on the mini 4.

**Phase 3 - Answers**
- Item flow polish: four confidence pills (replacing the slider), IDK
  leftmost, MCQ letters with number-key shortcuts, keyboard-first text
  entry with a proper `UITextView` (Scribble comes free on Pencil iPads),
  reveal only on teaching-unit checks.
- Handwriting box (optional per item): `PKCanvasView` with `.anyInput` in a
  fixed box; strokes exported from `PKDrawing` as normalized polylines and
  sent through the existing `/ink` contract (`strokes`, `aspect`, `idk`,
  `text`), so the server rasterizes, transcribes, and keeps the audit trail
  exactly as for the tablet. Default is typed; handwriting is a toggle.
- Exit: harness-driven check with typed, MCQ, IDK and one inked answer;
  results show READ AS for the inked one.

**Phase 4 - Pacing, resilience, performance**
- Chunk/break accounting parity with the tablet (chunk minutes from reader
  open, break minutes reported).
- Offline resilience: cached chapter readable when the server is away;
  every server failure is a screen with a way out, never a spinner.
- Performance on A8: one chapter web view at a time; MathText pooling; font
  subsetting for KaTeX; measure chapter load and check-page render on the
  device and record numbers in `DESIGN-ui.md`.

**Phase 5 - Device milestone and cleanup**
- Signing check (profile validity), install on the mini 4, full read of one
  chapter and one check on the device, Wi-Fi to the sprite.
- Remove the USB bridge listener and flight-only code paths; update
  `FLIGHT.md` pointers; update project memory.

## 8. Roadmap: multiple books in progress

Already true on the server: every subject has its own learner state,
chapters, results, and the active-book marker. Already true in the app after
Phase 0: per-subject persistence and a `BookSession` per subject. What
remains is UI and policy, in this order when the time comes:
1. Bookshelf cards showing, per book, the current chapter title, last-read
   time (from the event log's last `ts`; expose `last_active` in
   `/subjects`), and progress; the active book first.
2. Switching books preserves each book's reading position and any
   in-progress (uncommitted) answers; a check in progress in one book is
   never lost by opening another.
3. Pacing across books: session minutes and break suggestions are per human,
   not per book - the server's pacing model should read across subjects
   (server change, small).
4. Extension books (x1-x4) unlock per the syllabus rule and appear on the
   shelf as they are authored.
5. Later still: Pencil iPads get Scribble in every text field automatically
   and pencil-only ink in the handwriting box; a Mac Catalyst build is a
   settings flip if ever wanted.

## 9. Risks and open questions

- The iOS 15.5 runtime download may fail or be withdrawn when Xcode 27
  ships; if it does, the mini 4 itself becomes the iOS 15 gate (slower
  loop, still workable over USB).
- A8 + 2 GB: many `WKWebView`s at once will be evicted. The item flow shows
  one item, so the bound is small, but the reader plus MathText on a results
  screen with eleven entries needs measuring (Phase 4).
- Long synchronous exchanges (Opus authoring) versus iOS background
  suspension: an exchange that outlives the app being backgrounded is lost;
  server change B removes the long request entirely.
- Handwriting with a finger on a 7.9" screen is a poor answer medium; it is
  a toggle, not the default, and the typed path must be excellent.
- `TeachView` and the bundled book are carried, not improved; if they rot,
  they are removed rather than repaired.

## 10. Status (2026-09-01)

Phases 0 through 4 are built and verified in the Simulator on both
`chiron-ipad-15` (iOS 15.5, iPad mini 4 shape) and `chiron-ipad` (iOS 26.5):
unit tests green on both, `scripts/sim-verify.sh` walks every screen on
both in light and dark at the default and accessibility-extra-large text
sizes, and the self-test completes placement, series, results, and the next
chapter on both books. Server changes A, B, C are in (commits 1be81ca,
00332a6) plus the ink check-in marking the book active and a drive-mode stub
transcriber (43aceb8). None of it is on the sprite yet.

Still open:
- Phase 5 device milestone: install on the mini 4, one chapter and one check
  on the device, Wi-Fi to the sprite. Needs the iPad and Matt.
- Deploy the server (commits since e24f73e) to the sprite: the iPad needs
  `async` exchanges, `GET /chapter`, `results_doc`, and the reveal `correct`
  fix. Needs Matt's go-ahead.
- Phase 4 measurements on the A8 (chapter load, results render) and the
  KaTeX font subsetting, if the numbers call for it.
- Split View and rotation checked by hand in the Simulator (no simctl for
  either); the layout is size-class driven and all four orientations are on.
- The pretest still reveals the reference answer per item after commit, as
  the flight-era app did; the plan's "reveal only on teaching-unit checks"
  is read as "never on the calibration series", which holds.
