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

## Deterministic beats clever: one answer, one page

Mapping ink to answer regions by extracting element geometry from the
rendered HTML was the obvious design. Instead, every question box gets its
own page, so the mapping is arithmetic: pretest items open the page stack,
check items close it, strokes on a page belong to its item. No geometry
extraction, no coordinate contract to drift.

When a layout invariant carries semantics (like "one item per page"), it
must hold structurally (page-break CSS), not by hoping content fits.

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
- MathText-style web views are invisible to the accessibility tree; anything
  that must be automatable or readable belongs in native elements.
- `file://` XMLHttpRequests need `QML_XHR_ALLOW_FILE_READ=1` and get cached
  by the engine after a couple of reads - do not build a polling channel on
  them. Poll HTTP instead.
- The native macOS control style refuses `contentItem` customization and
  renders such buttons blank; the emulator launches with
  `QT_QUICK_CONTROLS_STYLE=Basic`.
- Text that reaches native elements from server HTML must have entities
  unescaped; `&quot;` on screen is a class of bug the renderer path never
  shows (KaTeX pages go through a real HTML engine, native Text does not).

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

## The audit trail is part of the UI

Every interpreted answer keeps its evidence next to the learner state: the
rasterized ink PNG and the transcription that was graded. A surprising grade
must be checkable against what the system actually saw - "the model said so"
is not an acceptable answer to "why was this marked wrong?"
