# Design brief: Chiron on reMarkable Paper Pro

This document is the complete context for designing Chiron's e-ink client
UI. The design that comes back will be implemented exactly, then verified
screen by screen against live captures from an automated harness - so
specify everything in measurable terms (positions, sizes, type scale,
states), not vibes. Where this brief constrains you, the constraint is
load-bearing; everything else is yours.

## What the product is

Chiron is an adaptive textbook for one learner. It teaches a fixed syllabus
("How AI Works": transformers, training, inference, MoE - real math) but
dials depth, representation, and pacing to the learner, measured by a
calibration series and per-chapter comprehension checks. A server on the
learner's Mac does everything adaptive (grading, planning, authoring via
LLMs); the tablet client displays chapters and captures answers. One
learner, one subject at a time, sessions of 20-25 minutes of reading with
checks at chapter boundaries.

The feel to aim for: a beautiful print book that quietly adapts. Not an
app with a book inside it. Chrome earns its place or it goes.

## The device

reMarkable Paper Pro: 11.8" color e-ink, 1620x2160 portrait, 229 ppi.
Pen (with pressure/tilt) plus capacitive touch. Realities that bind:

- E-ink refresh is slow and is controlled by the host UI, not us. NO
  animation, no transitions, no scrolling text, no progress spinners that
  depend on smooth redraws. State changes should be discrete page-like
  swaps.
- Color exists but is muted, low-saturation, and costs refresh quality.
  Design monochrome-first; use color only where it carries meaning (e.g.
  verdicts), never decoration. Assume colors render at roughly newsprint
  saturation.
- The pen is the primary answer instrument; touch is for navigation and
  discrete controls. Palm rests on the screen while writing.
- Big tap targets (finger-sized, ~7mm/64px minimum at native resolution).

## The architecture, which is also the design system's skeleton

Two rendering systems compose every screen. Respect the split - it is the
most important constraint in this document:

1. **Content is a page image.** Anything with typography, math, or layout
   (chapter prose, question prompts, worked examples, tables) is rendered
   server-side (HTML + KaTeX through a real browser engine) into full-page
   PNGs at exactly 1620x2160, displayed 1:1. You design these pages as
   PRINT layout: a CSS-like spec (margins, type scale, rules, boxes) that
   I implement in the server's page stylesheet. Current print conventions:
   Georgia serif, 34px body on 1.55 line height, 100px top/bottom and
   110px side margins, black on white.
2. **Interaction is a native control.** Everything the learner operates -
   option selection, confidence, "I don't know", page turns, check-in,
   the placement rating - is a native widget drawn by the client ON TOP
   of the page image, producing structured data. Controls are Qt Quick
   (Basic style, fully customizable: colors, borders, radii, type). No
   web views. Text in native controls cannot render TeX - any math
   belongs on the page image side.

The bridge between the two: **the server publishes geometry it enforces.**
Answer boxes are packed into fixed-height slots (units of 1/6 of the
1960px content area, 2-3 questions per page), and each item's page +
normalized region rectangle ships to the client, which places that item's
controls inside its box (currently: in a 130px strip reserved at the box
bottom, marked by a dashed rule). You may redesign the strip, the box
chrome, the packing density targets, and the control placement - but
per-item controls must live within that item's published region, and
every geometric decision must be expressible as fixed pixel constants the
server can enforce.

## Non-negotiable pedagogical constraints (they look like UI decisions)

- **Confidence before reveal.** The learner rates confidence (4 levels:
  unsure / shaky / confident / sure) per answer BEFORE any correctness
  feedback exists. Checks are closed-book; reference answers appear only
  in the post-grading results.
- **Every answer has an exit.** "I don't know" is a first-class,
  one-action, never-punished answer on every question. It must be
  visually available but not the path of least resistance compared to
  attempting.
- **Ink is the answer medium for constructed questions.** Free-form
  answers are handwritten in the box; a vision model transcribes them
  server-side. MCQs and ratings are tapped, never inked.
- **The reveal is part of learning.** The results screen carries, per
  item: verdict, what the transcriber read (the learner's answer as the
  system understood it - this is the audit trail and must stay visible),
  the reference answer / per-option explanation for misses, and overall
  score with gate or calibration framing.
- **Waits are honest.** Two designed waits exist: grading (seconds) and
  "the next chapter is being written" (background authoring; usually
  finished before the learner is done reading results). Both need calm,
  static treatments - no spinners.

## Screen inventory (current behavior, all subject to your redesign)

1. **Placement (native, no page image).** One-time self-rating: intro
   paragraph, one question, five levels (1 absolute novice .. 5 expert)
   as full-width rows, Check in enabled once one is selected. Fully
   native because it has no math - you may design this screen freely.
2. **Reading/answering pager.** Page image + ink canvas. Page turns via
   ‹ › buttons bottom-left; "N / M" progress centered bottom; "Check in"
   bottom-right on the last page. Question pages: 2-3 boxed items, each
   with its own control row (IDK toggle + confidence pills, or IDK +
   lettered A/B/C buttons matching lettered options printed in the box).
   Prose pages: pure reading, plus bordered "beat" work boxes with ink
   room (prompts like "work this before reading on").
3. **Submitting.** Static "Grading your answers..." state.
4. **Results.** Headline (calibration: "Calibration complete - N%";
   checks: "Gate cleared" / "Below the gate" with score vs 80% gate,
   extension-unlocked note when earned), per-item verdict list with
   transcripts and reference answers, "Next chapter" action. TODAY THIS
   IS AN UNDESIGNED WALL OF TEXT - it needs real design. Note: native
   text here cannot render TeX; reference answers containing $math$
   currently show raw. Design may route long references back to rendered
   pages, truncate with "see page N", or accept plain-text-only refs -
   your call, but say which.
5. **Waiting for next chapter.** "The next chapter is being written..."
   poll state.
6. **Errors.** Server unreachable, authoring failed. Currently bare text
   + Retry. Needs the same design voice.
7. **Not yet built, design if you have opinions:** break screen (the
   pacing system can suggest 5-minute breaks), spine/progress view
   (units with mastery states), library (multiple subjects), session
   header (chapter title / time-remaining whisper).

## Existing print-page components you may restyle

Item boxes (3px black border, 20/24px padding, bold inline number),
beat boxes (3px border, uppercase micro-label "work this before reading
on", ink area), planner note (italic, left-rule blockquote at chapter
open), tables (2px rules), code (Menlo, bordered), section h1/h2 scale
(56/44px), display math blocks. Print layout for real paper (a future
rmapi flow) also exists and keeps on-page scaffolding (tick rows,
confidence pills printed) - if you restyle shared components, note any
divergence between interactive and print variants.

## Verification: how your design gets checked

I drive the app with a harness (semantic commands: select level, ink a
stroke, set confidence, page, check in) and capture full-screen PNGs at
every state. Your deliverable therefore should include, per screen:

- Layout spec with exact values (page-coordinate pixels for print-side,
  device-independent proportions for native-side), type scale, spacing
  system, control states (default / selected / disabled / toggled-IDK).
- A short checklist of visually verifiable assertions ("confidence row
  right-aligned inside the strip, 24px from box edge; selected pill
  inverts to black fill") that I can confirm from captures.

Current-state captures live in `rmpp/design-captures/` (placement, a
question page with ink and a selected confidence, the results wall, a
prose page with a worked example and a beat box); treat them as the
"before."

## Out of scope

Server behavior, grading, pedagogy structure, corpus content, and the
iPad client. The physical-paper (rmapi) flow only insofar as shared
print components note their divergence.
