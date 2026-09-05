# Lessons learned

One entry per thing that cost more than ten minutes to learn: the
symptom, the cause, what to do. Add to it in the same commit as the fix.

## iPad app

- **Liquid Glass swallows taps (2026-09-03, again 09-04).** Buttons on a
  view with `.glassEffect()` (plain or `.interactive()`) do not get their
  taps, and the same for a card with buttons on it. Use
  `.buttonStyle(.glass)` on each button inside a `GlassEffectContainer`,
  and material (`.regularMaterial`) for a surface that carries text and
  buttons. The palette UI test is the proof either way.
- **An SF Symbol name that does not exist renders as text (2026-09-04).**
  A `Label` in a toolbar showed its title instead of the symbol:
  `doc.text.badge.plus` is not a symbol. Check a name with
  `swift -e 'import AppKit; print(NSImage(systemSymbolName: "x", accessibilityDescription: nil) != nil)'`.
- **Toolbar items hide a Label's title (2026-09-04).** In iOS 26 bars a
  `Label` shows the symbol only; a titled button needs a `Text` label.
  A `Text` item in a bottom bar gets its own pill and wraps.
- **`hitTest` never sees the touch on the device (2026-09-04).** Deciding
  by `event?.allTouches?.first?.type` worked in the Simulator and let
  nothing through on the iPad. To let a finger scroll under a PencilKit
  canvas, make the canvas the scroll view (`isScrollEnabled`, policy
  `.pencilOnly`) and mirror its offset to the page.
- **Page rectangles are offset by the bar inset (2026-09-04).** With
  `contentInsetAdjustmentBehavior = .always` under a navigation bar, a
  web page's `getBoundingClientRect` is measured from the content's top,
  `adjustedContentInset.top` below the view's top. Convert both ways.
- **A PencilKit canvas inside a WKWebView's scroll view shows strokes late
  (2026-09-03).** WebKit holds touches for the page; strokes appeared on
  lift. Put the canvas over the web view and mirror content size, inset
  and offset.
- **The Simulator draws nothing with the default drawing policy
  (2026-09-03).** No Pencil there; use `.anyInput` under
  `targetEnvironment(simulator)`.
- **XCUI typing into a terminal takes minutes (2026-09-04).** Each
  keystroke waits for the app to idle and the caret never stops blinking.
  Type through the harness; use XCUI for the one key under test.
- **`XCUIApplication.typeText` cannot send control characters
  (2026-09-04).** "\u{03}" arrives as nothing. Send them through the
  harness's `shell/type`.
- **Two sheets in one beat: the second never shows (2026-09-04).** Setting
  the next `sheet(item:)` while the first dismisses loses it. Wait about
  350 ms between them.
- **A ternary between two button styles does not type-check
  (2026-09-04).** `.buttonStyle(cond ? .glass : .glassProminent)` fails;
  branch with `if` around the whole button.
- **`removeLast()` on an empty array crashes the test host silently
  (2026-09-03).** A grep for "passed" hid it. Read the result bundle.
- **New test files need `xcodegen generate` (2026-09-04).** The project
  is generated; a new file is not in it until then, and the count of
  passed tests stays where it was.
- **Date fixtures depend on the timezone (2026-09-03).** Use noon UTC.
- **`NWListener` needs its handler before `start` (2026-09-02).** Otherwise
  EINVAL; and `.any` port, `acceptLocalOnly` and the loopback interface
  all fail; pick a random port and check the peer is loopback.
- **SwiftTerm's Metal shaders need the Metal toolchain (2026-09-02).**
  `xcodebuild -downloadComponent MetalToolchain`, and builds need
  `-skipPackagePluginValidation`.
- **Confirmation dialogs on iPad have no cancel button (2026-09-03).**
  They are popovers; a UI test dismisses with `PopoverDismissRegion`.

## Server and sprite

- **`sprite-env services create` keeps only the last `--env`
  (2026-09-02).** Two flags left the book server without its key and open.
  One flag, comma-separated.
- **No `--needs` between services (2026-09-03).** It blocks restarts.
- **The sprite user's shadow entry is locked (2026-09-02).** sshd refuses
  key auth until `sudo usermod -p '*' sprite`.
- **`claude --update` hangs on the sprite (2026-09-04).** It shells out to
  `npm -g config get prefix` through the nvm shim. Reinstall with
  `curl -fsSL https://claude.ai/install.sh | bash`.
- **A shell with no locale draws ASCII (2026-09-04).** Claude Code showed
  "__" for its glyphs; the client rendered fine. `export LANG=C.UTF-8` in
  the sprite's `.bashrc`, and `tmux set-environment -g LANG C.UTF-8` for
  the running server.
- **Go installs under `/.sprite` without GOBIN (2026-09-03).** Set it.
- **One heavy job at a time on the sprite (2026-09-03).** 8 GB, serving
  the book; `go test -p 2`, `CHIRON_RENDER=0`, no build beside a test run.
  Checkpoint (`sprite-env checkpoints create`) before anything risky: the
  platform restored an older overlay three times in two days.
- **The Anthropic API now requires `additionalProperties: false` on every
  object in a structured-output schema (2026-09-03).** `llm.strictSchema`
  adds it.
- **An OAuth token gets 429 for Opus and Sonnet on the raw API
  (2026-09-04).** Haiku answers. `CHIRON_PROVIDER=claude-cli` runs the
  models through headless Claude Code instead; PATH must be in the service
  env.
- **`op read` times out when non-interactive (2026-09-02).** Automation
  passes the key as `CHIRON_KEY`.
- **`pkill -f` matches your own ssh command line (2026-09-03).** Anchor
  the pattern.
- **A test that gates every model call deadlocks a new call in front of
  it (2026-09-04).** `gatedChain` now gates the author only.

- **The CLI model writes about 33 tokens a second, and a depth variant
  of a long chapter is a 10k-token reply (2026-09-05).** Three units of
  the first sourced book timed out at fifteen minutes on deeper-math.md,
  three runs in a row; the author's limit is now thirty. And only the
  canon call needs the source material: the later five files are written
  from canon.md, and carrying 30k tokens of sources into them made every
  call slower for nothing.
- **A failed `go test` hidden behind a pipe (2026-09-05).** `go test ./x |
  tail -1 && git push ...` pushed a failing test because the pipe's status
  is tail's; the sprite's `make deploy` then refused, and a book resumed
  on the old binary. Gate on `go test` itself, never on a pipe.
- **The served corpus on the sprite is a copy, not the checkout
  (2026-09-05).** `/home/sprite/chiron/corpus` was copied once in August;
  `make deploy` swapped only the binary, so the new source index was not
  there and the first sourced book planned from the brief alone. Deploy
  now copies the files the server reads at build time (the authoring
  contract, `corpus/sources/index.yaml`); unit files stay as they are.
- **A primer has no chapter until something opens it (2026-09-05).**
  `GET /chapter/{id}` is nil for a ready primer nobody has read; the app
  opens it with `POST /exchange {phase: start}`, which makes the first
  unit current. `chiron read` does the same when it finds nothing.
- **A dev server needs the corpus's `authoring-spec.md` to build a book
  (2026-09-05).** A config with made-up corpus paths captures and writes
  primers, and fails a book draft at planning with that file's name.
- **A book draft fails with "claude-cli: timed out (planner)" (2026-09-05).**
  The planner is one `claude -p` call that emits the whole syllabus and
  the misconception bank as JSON; under the claude-cli provider its
  deadline was 5 minutes and a reference-depth brief blew through it
  three builds in a row, after 2-3 minutes of source resolution. Passage
  size is not the lever: a half-size passage failed at the same step. The
  planner now has the author's 15 minutes (`llm/claudecli.go`,
  `roleTimeouts`). The log line is the only symptom; nothing under the
  corpus dir is written before planning succeeds except `sources.yaml`.

- **`simctl openurl` for a custom scheme waits behind an "Open in
  Chiron?" alert (2026-09-05).** The app's `onOpenURL` never fires until
  it is tapped, which looks like a dead handler. `idb ui describe-all`
  shows the alert; `idb ui tap` on its Open button delivers the URL.
  Several opens stack several alerts.
- **A `.confirmationDialog` or `.alert` on a button inside a `Menu` or
  `.contextMenu` never shows (2026-09-05).** The menu's content is torn
  down as it closes, and the state that would present the dialog goes
  with it: "Delete shelf" did nothing on the iPad. Keep the presenting
  state and the modifier on the view that stays on screen; the menu item
  only flips the binding. The harness verb had bypassed the menu, so a UI
  test now goes through it.
- **XCUITest drags a `.draggable` card only if the press is short
  (2026-09-05).** A card with both `.contextMenu` and `.draggable` opens
  its menu at about a second; `press(forDuration: 0.6, thenDragTo:,
  withVelocity: .slow, thenHoldForDuration: 0.8)` lifts the drag first.
  And a failed assertion with `continueAfterFailure = false` skips Swift
  `defer`: clean up in `tearDown`, and at the start of the test.
- **A VStack with `maxHeight: .infinity` centres its overflow
  (2026-09-05).** The shelf lost its header at the top once the cards
  outgrew the screen; a ScrollView is what a growing list needs.

## Mac side

- **xcodebuild will not register a new device (2026-09-05).** With a
  phone plugged in for the first time it fails with "No Accounts: Add a
  new account in Accounts settings" and "provisioning profile doesn't
  include the currently selected device", even with both Apple IDs signed
  into Xcode and outside the sandbox. Open the project in Xcode, pick the
  device as the run destination, and Run once: that registers it and
  refreshes the profiles; `xcodebuild` works from then on.
- **ssh tries IdentityFiles in config order, `Host *` included
  (2026-09-04).** A catch-all with the YubiKey identities above the sprite
  stanza prompted for the YubiKey first. Put specific hosts above `Host *`.
- **The 1Password prompt on `ssh chiron` is the tunnel reading the key
  (2026-09-04).** Expected; the YubiKey prompt was not.
