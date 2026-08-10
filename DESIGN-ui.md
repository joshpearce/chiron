# UI construction principles

Rules for building Chiron's clients, each learned from a real failure or a
real piece of friction. When adding UI to any client - iPad, reMarkable,
whatever comes next - check the change against these before building.

## Content is an image; interaction is a control

The server renders anything with typography, math, or layout (chapter prose,
question prompts, worked examples) - clients display it, byte-identical
everywhere. Anything the learner *operates* - option selection, ratings,
confidence, "I don't know", page turns - is native client UI producing
structured data.

The dividing line is not "what is hard to render" but "what produces an
answer." An answer channel that runs through rendering (ink, free text in a
picture) needs interpretation, and interpretation fails silently. A control
produces exact data.

- The screener placement question is rendered natively by the client (it has
  no math): the bordered question box is drawn in QML and the level rows are
  buttons inside it. Where a page has nothing to typeset, skip the image.
- MCQ options are lettered in the rendered page and answered by matching
  lettered buttons - the image shows, the control answers.

## Never make a model interpret what a control could have captured

An "I don't know" tick drawn in ink went to the vision transcriber, which had
the question text in its prompt "for context" - and answered the question
itself. Marked-don't-know items came back graded correct.

Two rules fell out:

1. **Interpretation layers get the minimum context to do their one job.**
   The handwriting transcriber never sees the question. It transcribes ink;
   the text grading path judges the transcription. One grading contract,
   shared with every client.
2. **Explicit signals are flags, not marks.** IDK, confidence, and selections
   travel as `idk` / `confidence` / `selected_index` fields. An explicit flag
   also short-circuits cost: an IDK item is graded instantly with no model
   call, and the batch grader skips it.

## Answers must always have an exit

The iPad's Commit button was disabled until text was typed, and paper offered
only blank space - so a learner who didn't know an answer had to fabricate
one to move on. Pretests are *designed* to be failed; forcing filler poisons
the signal. Every answer surface carries an explicit "I don't know" that is
one action, never punished, and recorded as itself.

## Deterministic beats clever: the server enforces the geometry it publishes

Mapping ink to answer regions by extracting element geometry from the
rendered HTML was the obvious design and stayed rejected. The first cut gave
every question its own page (mapping by arithmetic), which burned screen
real estate; the current design packs 2-3 questions per page into
fixed-height slots the SERVER assigns - estimated from prompt length, answer
type, and option count - and publishes each item's page + region in the
pages meta. The renderer emits explicit page containers with those exact
box heights, so the published regions cannot drift from the pixels: the
server enforces the geometry it declares. Clients place per-item controls
into each region and assign strokes to items by containment, renormalized
to the region so transcription sees a tight, undistorted crop.

When a layout invariant carries semantics, it must hold structurally
(explicit containers, fixed heights), not by hoping content fits.

## One corpus, per-medium affordances

The same chapter renders in two layouts keyed by client medium:

- **Interactive** (AppLoad client): no printed pills, tick rows, or answer
  scaffolding - controls own those. MCQ options are lettered to pair with
  buttons.
- **Print** (the rmapi real-paper flow): confidence pills, IDK tick rows, and
  MCQ tick squares are printed, because the page is the only interface.

The layout variant is part of the page-image cache key. Serving one medium's
affordances to the other reads as clutter (printed pills under real buttons)
or as a dead end (a paper page with no way to mark confidence).

## Page images are content-addressed and pre-rendered

- Page URLs carry the chapter content hash, so a regenerated chapter can
  never serve a stale cached page.
- Rendering happens eagerly the moment a chapter is persisted - during the
  seconds the learner reads their results - not lazily on first request. The
  authored chapter arriving fast and the pages arriving slow reads as a hang.
- The renderer serializes: eager and on-demand renders of the same chapter
  must not race into one cache directory.

## E-ink UI language

Plain black on white, serif for content, no animation, big tap targets.
Page turns are explicit buttons, not gestures - the full page surface belongs
to the pen once ink capture exists. Refresh pacing belongs to xochitl on the
Paper Pro (homebrew has no refresh control yet), so nothing in the UI should
depend on fast redraws.

## Small courtesies that turned out to be load-bearing

- Confidence is captured BEFORE any reveal, always (fluency illusion defense
  from the learning-science research; the UI must make it structurally
  impossible to peek first).
- A single-question screen carries no item numbering.
- Controls live inside the question box they answer, not docked at a screen
  edge disconnected from what they refer to.
- A gateless check-in (the screener step) advances straight to the delivered
  content; an empty results screen is a speed bump.
- Buttons that fire an exchange must drop duplicate taps while one is in
  flight - a double-tap once double-graded a whole check, at full model cost.

## QML specifics that bit us

- Assigning a `var` property back to itself after in-place mutation does not
  emit a change signal - bindings silently never re-evaluate ("I selected a
  level but Check in never enabled"). State maps are copy-on-write: build a
  fresh object for every update.
- The AppLoad backend channel is SOCK_SEQPACKET, which macOS lacks - the Go
  backend speaks HTTP on localhost instead, so the emulator and the device
  share one transport.
- On SOCK_SEQPACKET a zero-length packet reads as n=0 with no error -
  byte-identical to EOF - and AppLoad transmits an empty-string message
  payload as exactly such a packet. Treating every zero read as a hangup
  killed the backend (and with it the app) on the first page turn. Never
  send an empty payload, and disambiguate zero reads with a poll for
  POLLHUP/POLLRDHUP before concluding the peer closed.
- MathText-style web views are invisible to the accessibility tree; anything
  that must be automatable or readable belongs in native elements.
- `file://` XMLHttpRequests need `QML_XHR_ALLOW_FILE_READ=1` and get cached
  by the engine after a couple of reads - do not build a polling channel on
  them. Poll HTTP instead.
- The native macOS control style refuses `contentItem` customization and
  renders such buttons blank; the emulator launches with
  `QT_QUICK_CONTROLS_STYLE=Basic`.
- `Text.lineHeight` is a multiplier of the FONT's natural line height, not
  of `pixelSize` - a spec that says "34/53" needs
  `lineHeightMode: Text.FixedHeight; lineHeight: 53` or the leading comes
  out ~30% too airy.
- A standalone italic TTF (SourceSerif4-It) registers under the base family
  name, so `font.family: loader.name` alone silently renders the ROMAN
  face; set `font.italic: true` as well. Non-RIBBI weights (Semibold,
  Medium) register distinct family names and work by name alone.
- Text that reaches native elements from server HTML must have entities
  unescaped; `&quot;` on screen is a class of bug the renderer path never
  shows (KaTeX pages go through a real HTML engine, native Text does not).
- `grabToImage` callbacks never complete while the window is occluded -
  Qt parks the render loop for unexposed windows. It looked like random
  flakiness for a day. Harness runs use `QT_QPA_PLATFORM=offscreen`,
  which renders unconditionally and keeps automated runs off the
  human's screen entirely.

## The design must be verifiable without a human

The emulator is driven end to end by the dev harness, not by asking a person
to click: the app polls `GET /drive/next` on its server for semantic commands
(select a level, ink a stroke, check in, dump state) and acks each with a
`grabToImage` screenshot to `/tmp/chiron-drive/`. The queue only exists on a
server started with `CHIRON_DRIVE=1`, and the client only polls when
`/tmp/chiron-drive/enable` names a drive server - so none of it is reachable
in real use. Commands are semantic rather than raw taps on purpose: the
harness tests the design ("select level 2") rather than pixel coordinates,
and the same channel can drive the physical tablet later.

Verification runs use a disposable server and state dir - never the state a
human is working through. Being able to see the screen ("the pills are at
the bottom, not in the box") is the difference between confirming a design
and hoping about it.

## Every entry field offers pen AND keyboard

Ink is the primary answer medium, but every text-entry field also offers an
on-screen keyboard (the "Type" toggle in the item's control strip). Rules
that came with it:

- The keyboard is our own flat monochrome QML component, not the system
  one: xochitl's input method is not guaranteed to attach inside an AppLoad
  app, and a hand-rolled keyboard keeps the e-ink design language (square
  keys, discrete repaints, no animation) and works identically in the
  emulator.
- The keyboard must never hide the field being edited. It docks at the
  window bottom and the page view shifts up so the active item's box stays
  fully visible above it; closing the keyboard restores the view. All
  overlays track automatically because they position through the page
  image's origin.
- Typed answers go to grading verbatim - no rasterizing, no vision model -
  and still land in the transcript record, so READ AS on the results page
  shows exactly what the system graded, typed or inked. Typing clears IDK
  (it is an attempt); IDK still wins over any content at submission.
- Math-answer symbols (- + = / * ^ ( ) , .) get a dedicated key row;
  answers here are short expressions, not prose.

## E-ink renders grays lighter than you designed them

The panel washes out grays by one to two steps relative to their sRGB
intent: #999 text is near-invisible, #666 reads like #999 - the color
filter array over the Gallery 3 panel eats grays harder than plain
carta would. A screen that is MOSTLY gray text (the contents screen,
with its unwritten chapters) exposes this brutally even when individual
grays looked fine on the emulator. Rule, calibrated twice on device:
nothing meant to be READ sits lighter than #555; #777 is the floor for
purely decorative marks (leaders, rules). Screens dominated by
de-emphasized text get their whole palette shifted rather than
per-element tweaks. Every
screen also needs an exit: a Close affordance must be reachable from
ANY screen, not just via happy-path navigation.

## Pen ink on e-ink: what we measured on the Paper Pro

The QML canvas can never feel like a pen, and the reasons are specific.
Findings from instrumented strokes and the display-pipeline research
(2026-08-10), so nobody re-litigates them:

- Qt's pointer-event delivery stalls ~130ms at every stroke start and
  delivers in bursts; the digitizer hardware (Elan SPI, ~500Hz, ranges
  11180x15340 mapping straight onto the portrait screen) has none of
  that. The ink backend reads /dev/input/event2 directly; xochitl does
  not grab the device, so both readers coexist.
- qtfb's UFAST refresh mode IS xochitl's own Pen waveform - there is no
  faster mode to find. The display floor is ~12ms to first visible ink,
  ~370ms to full completion; the darkening tail is physics, not a bug.
- Refresh-request granularity is a three-way trap, all observed live:
  one request per pen sample (500/s) floods xochitl's event loop, which
  does not coalesce; large batched chunks (15ms of path) render as
  dashes because adjacent pieces sit in different waveform phases; one
  growing re-targeted rect defers everything to pen-up because the
  engine coalesces same-region updates. Small DISJOINT chunks at ~5ms
  cadence read as a continuously growing line.
- Pixels go into the shared framebuffer immediately; only refresh
  REQUESTS are batched. First contact always refreshes instantly.
- The remaining gap to native feel is mostly stroke prediction:
  reMarkable patented ~20ms-horizon pen prediction tuned to minimize
  the visible white tail and to keep small mispredictions rather than
  pay a refresh to erase them. A Kalman predictor over the 500Hz
  timestamped stream is the known next step if ever wanted.
- Per-stroke latency forensics stay in chiron-ink (one journal line per
  stroke: event gaps vs processing time). That split is what located
  every bottleneck above; keep it.

## The audit trail is part of the UI

Every interpreted answer keeps its evidence next to the learner state: the
rasterized ink PNG and the transcription that was graded. A surprising grade
must be checkable against what the system actually saw - "the model said so"
is not an acceptable answer to "why was this marked wrong?"
