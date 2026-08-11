# Chiron RMPP UI — Implementation Spec

Companion to `Chiron Screens.html` (frames 01–15, all at native 1620×2160).
Page-side values are page-coordinate pixels at 1620×2160. Native-side values
are given in the same pixel space at native resolution; treat them as
device-independent constants (divide by 1620/2160 for normalized form).

## 0. System

### 0.1 Architectural decision — math in results
**The results screen is server-rendered pages**, same pipeline as chapters:
headline, gate bar, and per-item entries are typeset (with KaTeX) into
1620×2160 PNGs and paginated by the server. Native contributes only: page-turn
buttons, folio is page-side, and the action button (bottom-right). No math
ever appears in native text; reference answers render fully, never truncated.
Grading wait covers grade + typeset time. This replaces the current native
results wall entirely.

### 0.2 Layers
- **Page (server stylesheet):** all typography, math, boxes, rules, folio,
  running heads, verdict marks, gate bar.
- **Native (QML):** confidence pills, IDK, MCQ letters, page turns, action
  button + its caption, placement screen (whole), wait/error/break screens
  (whole), invisible TOC row tap targets, ink capture.
- Native chrome may draw ONLY: (a) inside a published item-strip region,
  (b) inside the bottom chrome band y 2060–2160, except on fully-native
  screens (placement, waits, errors, break).

### 0.3 Fonts
- **Source Serif 4** — all content voice, page AND native (bundle TTFs,
  OFL license): 400, 600, 700, italic 400.
- **Source Sans 3** — every native control label (bundle: 400, 500, 600, 700).
- **Menlo** — code, page-side only (server Mac has it).
- **KaTeX** default faces — math, page-side only.
- Replaces Georgia everywhere. Print (rmapi) variant uses the same faces.

### 0.4 Palette (16-level grayscale-safe)
| token | hex | use |
|---|---|---|
| ink | #000000 | body text, box borders, fills of selected controls, gate fill |
| gray-2 | #444444 | running heads, folio, deks, beat-box labels |
| gray-3 | #777777 | micro-labels, confidence tags, whisper, captions, disabled/muted controls (the e-ink floor - see DESIGN-ui.md) |
| gray-4 | #BBBBBB | print variant only - washes out on the panel, never for on-device UI |
| hairline | #CCCCCC | entry separators |
| paper | #FFFFFF | background everywhere |
| accent | #8A3D30 | ONLY: miss ✗ marks, miscalibrated tags, error asterisk. Never decoration. |
Dotted rules #999999. No other colors. No shadows, no radii (all corners square), no gradients.

### 0.5 Type scale
Page side (Source Serif 4 unless noted):
| role | spec |
|---|---|
| Results headline | 72/80, 600 |
| Chapter H1 | 56/64, 600 |
| Section H2 | 44/54, 600 |
| Body | 34/53, 400, justified, hyphens on, max-width 1220 |
| Item prompt | 34/50, 400, ragged (item number 700 inline) |
| MCQ option / beat-box prompt | 32/46–48, 400 |
| Results entry prompt | 30/40, 400, #222, max 2 lines |
| Results value (READ AS / ANSWER) | 30/42, 400, #000 |
| Results WHY note | 28/40, 400, #333 |
| Dek | italic 32/44, #444 |
| Footnote | italic 26/36, #666 |
| Micro-caps | 22/28, 600, +0.10em, UPPERCASE, #777 (beat-box label #444) |
| Running head | 24, 600, +0.08em, UPPERCASE, #444 |
| Folio | 26, 400, #444, centered |
| Tally / gate label | 24 and 20, 600, +0.10em caps, #444 |
| Code | Menlo 0.82em of surrounding |

Native side (Source Sans 3):
| role | spec |
|---|---|
| Control label (pills, IDK) | 26, 600 (IDK 500) |
| MCQ letter | 30, 700 |
| Action button | 28, 600 |
| Caption | 22, 400, #777 |
| Rating row label | 30, 600 (number cell 700) |
| Page-turn glyph | 34, 400 (‹ ›) |
Native prose on placement/waits uses Source Serif 4 at page sizes.

### 0.6 Rules inventory
- Box border: 2px solid #000
- Shelf rule (control strip top): 1px solid #000, full box interior width
- Entry separator: 1px solid #CCC
- Headline divider (results): 1px solid #000
- Ink boundary: 2px dotted #999
- Leader (TOC): 2px dotted #BBB
- Gate bar: 10px tall, 1px #999 outline, #000 fill, tick 2×28px #000

### 0.7 Native control state matrix (all heights 64, corners square, flat)
| control | default | selected | muted (IDK active) | disabled | pressed |
|---|---|---|---|---|---|
| Confidence pill | white, 2px #000 border, label 26/600 #000, pad-x 28 | fill #000, label #FFF | 1px #777 border, label #777 (pad-x 29 to hold width) | — | invert |
| I don't know | white, 1px #666 border, label 26/500 #444, pad-x 28 | fill #000, 2px #000 border, label #FFF (pad-x 27) | — | — | invert |
| MCQ letter | 64×64, 2px #000, letter 30/700 | fill #000, letter #FFF | 1px #777, letter #777 | — | invert |
| Action (Check in / Next chapter / Try again) | 2px #000 border, label 28/600 #000, pad-x 36 | — | — | 1px #777 border, label #777 (pad-x 37) | invert |
| Page turn | 64×64, 1px #666, glyph #333 | — | — | 1px #777, glyph #777 | invert |
| Rating row | 1400×112, 1px #666 border; number cell 96 wide, right divider 1px #666; label pad-x 28 | fill #000, 2px #000 border, all text #FFF, divider #555 | — | — | invert |
"Pressed" = momentary inversion on the next partial refresh; no animation.
Selection changes repaint the control's own rect only (partial refresh);
page changes are full-page swaps.

## 1. Page grid (every page-rendered screen)
- Page 1620×2160. Margins: top/bottom 100, sides 110. Content 1400×1960 (x 110–1510, y 100–2060).
- **Running head:** baseline row at y 40–64; left = chapter ("2 · THE SHAPE OF COMPUTATION"), right = mode ("CHECK" / "RESULTS" / "CONTENTS" / "PLACEMENT") or, on prose pages, the time whisper "≈ 12 MIN LEFT" (24 caps, #777, computed at render time from remaining reading estimate — static per page). Style: 24/600/+0.08em caps #444.
- **Folio:** page-side, centered, 26px #444, baseline 26px above bottom edge (block bottom-aligned at y 2100–2126): "7 / 28". Results/check folios count their own series.
- **Prose measure:** paragraphs justified, hyphenated, max-width 1220 from left content edge. Boxes, tables, display math, rules span the full 1400.
- **Native bottom chrome band** (y 2078–2142, i.e. 64px controls centered in the bottom margin):
  - Page-turn ‹ at x 110, › at x 186 (64×64 each, 12 gap). Shown whenever the current series has >1 page. First page: ‹ disabled; last page: › disabled.
  - Action button right-aligned to x 1510: "Check in" (last check page), "Next chapter" / "Begin chapter 1" / "Back to the chapter" (results — label server-driven).
  - Caption (when present) 22px #777, right-aligned 16px left of the action button, vertically centered.
- Swipe left/right anywhere outside item strips also turns pages.

## 2. Question pages

### 2.1 Packing (replaces the 1/6-slot system)
- Item box heights: **640** (MCQ ≤4 options, short constructed) or **970**
  (constructed with ink work, MCQ ≥5 options or long stems). Inter-item gap **20**.
- Legal pages: 970+970 (=1960, flush), 640+970+gap ends 1630, 640×3+gaps=1960, 640+640.
  Pack top-aligned at y 100; leftover whitespace stays at the bottom.
- Server publishes per item: page index, box rect [x=110, y, w=1400, h], strip height.

### 2.2 Item box anatomy (page-side)
- Border 2px #000. Interior padding: 28 sides, 24 top.
- Number + prompt: "N." 700 inline, prompt 34/50, ragged, full interior width (1340).
- **Constructed:** ink boundary = 2px dotted #999, inset 28 each side, 20px below
  the prompt's last line. Below it: blank ink zone down to the shelf rule.
- **MCQ:** options list instead of ink zone, starting 14px below prompt; each
  option row = "A." (700, 44px column) + option text 32/46, 14px between rows.
  No dotted rule. Options are lettered A, B, C… matching the native letter buttons.
- **Shelf rule:** 1px solid #000 spanning the full interior width, its top edge
  exactly (strip+1)px above the inner bottom edge.
- **Control strip** (reserved for native): constructed **120px**, MCQ **196px**,
  measured from inner bottom edge to the shelf rule.

### 2.3 Native controls in the strip
All controls vertically centered in their row; side insets 30 from box inner edges.
- **Constructed (one row, strip 120):** IDK button left-aligned; confidence
  group [unsure][shaky][confident][sure] right-aligned, 12px gaps.
- **MCQ (two rows, strip 196):** rows of 64px with 16px between, 26px top/bottom pad.
  Row 1: IDK button left-aligned - the SAME position it holds on constructed
  rows, so the exit never moves between questions - then letter buttons
  (64×64, 12 gaps, count = option count) after a 24px gap.
  Row 2: confidence group right-aligned.
- **Selection logic:** IDK and the 4 confidence levels are mutually exclusive
  (5-way). Tapping a confidence level clears IDK and vice versa. For MCQ,
  selecting IDK clears the letter selection and mutes letters + pills;
  muted controls stay tappable (tapping restores attempt mode).
- **Ink:** pen input is captured only inside the item's box, clipped to the
  region between box top and the shelf rule (strokes over the strip are
  rejected). MCQ items capture no ink at all.
- **Answered** means: a confidence level selected (constructed: with or without
  ink; the transcriber handles blanks) or letter+confidence (MCQ) or IDK (any).
- "Check in" appears only on the last page of a check; enabled when every item
  is answered; otherwise disabled with caption "N items still need an answer".

### 2.4 Assertions (verify from captures)
1. Two 970-boxes fill y 100–2060 exactly: first box top edge at y 100, second box bottom edge at y 2060, 20px white between them.
2. Box borders 2px pure black, square corners; shelf rule 1px, spans border-to-border; nothing else horizontal within 24px of it.
3. Dotted ink boundary starts 28px in from each inner edge; 2px dotted, gaps visible at 100%.
4. Selected confidence pill: solid #000 fill, label pure white; its unselected neighbors keep 2px black borders on white.
5. IDK sits left-inset 30 from box inner edge; confidence group's right edge inset 30; both rows of centers at (inner bottom − 60).
6. With IDK selected: IDK solid black/white; all four pills show 1px #BBB borders and #BBB labels; overall pill row width unchanged (±2px).
7. MCQ: selected letter solid black with white letter; exactly one selectable letter per printed option, same letters, same order.
8. Page 1 of 3: ‹ is 1px #CCC border with #BBB glyph (disabled); › 1px #666/#333. Last page: states swap and "Check in" (2px black border) sits right-aligned at x 1510.
9. Ink strokes never appear below a shelf rule.
10. Folio centered, "1 / 3" style, 26px #444; no other chrome in the bottom margin except ‹ ›, caption, action.

## 3. Results pages (page-rendered; see §0.1)

### 3.1 Layout
- Running head: chapter left, "RESULTS" right ("HOW AI WORKS" / "CALIBRATION" for the calibration series).
- Headline 72/80 600 at y ≈ 178 (first baseline ≈ 240): "Gate cleared — 82%." / "Below the gate — 64%." / "Calibration complete — 7%."
- **Gate bar** (checks only), 40px below headline block: 1400×10, 1px #999 outline; #000 fill width = score% of 1400; tick 2px wide × 28px tall (9px above/below bar) at exactly 80% (x = 110+1120 = 1230); label "GATE 80" 20px caps #444, right-aligned to the tick, 20px below the bar.
- **Tally line** (calibration, replaces bar): "1 CORRECT · 0 MISSED · 13 PASSED", 24px caps 600 #444.
- Dek italic 32/44 #444: gate framing / extension note / calibration framing. Server-authored copy.
- Divider 1px #000 full 1400, 44px below dek.
- Entries follow; hairline 1px #CCC separators between entries (34px padding above and below each entry).
- Overflow paginates: repeated running head + folio "2 / 4"; headline block not repeated (entries continue at y 100 + 34).

### 3.2 Entry anatomy
Grid: 72px verdict column + 1328px body.
- **Verdict marks** (40px, 700, aligned to first text baseline): ✓ #000 correct · ~ #000 partial · ✗ #8A3D30 miss · — #999 400-weight for IDK.
- Header row: "N." 700 + prompt, 30/40 #222, clamped to 2 lines; right: confidence tag, 22 caps 600, #777 — **#8A3D30 when confident/sure met ✗** (the miscalibration signal). IDK entries have no tag.
- Meta rows (grid: 170px label + value, 18px between rows). Labels in micro-caps #777:
  - `READ AS` — transcript verbatim in curly quotes (constructed). This row is mandatory on every constructed entry — it is the audit trail.
  - `CHOSE` — "B — 'option text'" (MCQ).
  - `MARKED` — ""I don't know."" (IDK).
  - `ANSWER` — reference, full math allowed. Present on miss/partial/IDK; omitted on correct unless transcript ≠ canonical form.
  - `WHY` — grader/per-option explanation, 28/40 #333; miss and partial only.
- Native: action button bottom-right on every results page (label per §1); ‹ › when >1 page.

### 3.3 Assertions
1. Headline ends with an em-dashed percentage; below-gate uses identical layout with fill ending left of the tick (fill edge at x = 110+1400·score%, tick fixed at x 1230).
2. Gate tick present on check results, absent on calibration; tally line present only on calibration.
3. Every ✗ mark and only ✗ marks (plus miscalibrated tags) use #8A3D30; ✓/~ pure black; — is #999.
4. Every constructed entry shows a READ AS row with the transcript in quotes, even when correct.
5. Every miss/partial/IDK entry shows an ANSWER row; math in it is typeset (no raw $...$ anywhere on the page).
6. A confident/sure tag on a missed item renders in #8A3D30; the same tag on a correct item renders #777.
7. Entry separators are 1px #CCC and never touch the 72px verdict column's marks.
8. Action button reads "Next chapter" (cleared), "Back to the chapter" (below gate), "Begin chapter 1" (calibration); always right-aligned to x 1510, bottom band.
9. No spinner, no progress element anywhere on results.

## 4. Placement (fully native)
- Running head as §1: "HOW AI WORKS" / "PLACEMENT".
- Intro: Source Serif 34/53, max-width 1220, top at y 200 (learner-facing copy as built today).
- Question: 40/56 600, max-width 1300, 72px below intro.
- Five rating rows, 52px below question: 1400×112 each, 20px between; anatomy and states per §0.7. Exactly one selectable at a time; a second tap on the selected row does NOT deselect (there is no empty state after first selection).
- Footnote italic 26/36 #666, 44px below rows: "This sets where the questions begin — nothing more."
- "Check in" bottom-right per §1; disabled (1px #BBB/label #999) until a row is selected, no caption on this screen.

### Assertions
1. Selected row: solid #000 fill, white label and number, 2px black border; unselected rows 1px #666 with white fill.
2. Number cell exactly 96px wide with a full-height 1px divider (#666 unselected, #555 selected).
3. Rows are flush with the 1400 content width; 20px gaps; total block 640px tall.
4. Check in flips from disabled to 2px-black-bordered the moment a row is selected; positions identical in both states (±1px).
5. No math, no page image, no folio on this screen.

## 5. Prose pages
- Grid per §1; body 34/53 justified/hyphenated at 1220; display math centered, full 1400 available; H2 44/54 with 46px above / 24px below.
- Blockquote/planner note: italic 32/48 #333, 2px solid #000 left rule, 28px left padding (interactive and print identical).
- **Beat box:** border 2px #000, padding 28/24; micro-caps label "WORK THIS BEFORE READING ON" 22/600/+0.10em #444, 22px below top padding edge... (label first, 22px gap to prompt); prompt 32/48; dotted ink boundary 20px below prompt; ≥260px ink room below it (padding-bottom 280 total). No control strip, no native controls, no confidence — beats are ungraded ink room.
- Time whisper in running head right slot (prose pages only): "≈ N MIN LEFT", 24 caps #777.

### Assertions
1. Every paragraph's text block ≤1220px wide, justified with hyphenation; display math may exceed it up to 1400.
2. Beat box has NO shelf rule and no native controls; its dotted rule matches question-box dotted style exactly.
3. Running head right slot shows "≈ … MIN LEFT" in #777; left slot shows chapter in #444.
4. Folio "7 / 28" centered; ‹ › both enabled mid-chapter.
5. Code spans render in Menlo at 0.82× surrounding size, no background, no border.

## 6. Waits, errors, break (fully native, static)
Shared skeleton, centered column at page center-x:
- **Dinkus** at y 880 (top of block): "✱ ✱ ✱" 36px, 26px letter-spacing, #000.
- Statement, 56px below: italic Source Serif 40/56 #000.
- Sub-line, 14px below: 28/40 #666, ≤2 lines, centered.
- Optional button row 72px below sub: primary/quiet per §0.7.
- Optional detail line 52px below buttons: 22px Source Sans, +0.06em, #999.

| screen | dinkus | statement | sub | buttons | detail |
|---|---|---|---|---|---|
| Grading | ✱ ✱ ✱ black | "Grading your answers." | "A few seconds." | — | — |
| Authoring | ✱ ✱ ✱ black | "The next chapter is being written." | "Shaped by today's answers. It is usually ready before you finish reading the results." | — | — |
| Authoring, >45s | same | same | + line "Still writing — Chiron checks every few seconds." (26 #999) | — | — |
| Server unreachable | single ✱ #8A3D30 | "The server can't be reached." | "Your work is saved on this tablet. Nothing is lost." | [Try again] | "HOST UNREACHABLE · 09:14" |
| Authoring failed | single ✱ #8A3D30 | "This chapter couldn't be written." | "Nothing is lost — your results are kept. Try again, or come back later." | [Try again] | "PLANNER TIMEOUT · 21:36" |
| Break suggested | ✱ ✱ ✱ black | "A good place to pause." | "24 minutes of reading — and the next section is a climb. Five minutes off pays for itself." | [Take five] [Keep reading] | — |
| Break active | ✱ ✱ ✱ black | "On break." | "Tap anywhere when you're ready." | — (any tap resumes) | — |

State swaps (e.g. grading → results, >45s line appearing) are single discrete
repaints. Nothing on these screens moves, blinks, or counts.

### Assertions
1. No element on any wait/error/break screen changes between two captures 5s apart (except the one designed >45s line).
2. Error screens: the asterisk is single and #8A3D30; wait/break dinkuses are triple and black.
3. [Try again] matches primary anatomy (64px, 2px #000, 28/600); [Keep reading] is the quiet variant (1px #666, 500).
4. Detail line present on errors only, all-caps, #999, includes a timestamp.
5. Statement is italic serif — never sans, never bold.

## 7. Contents / spine (page-rendered + native row taps)
- Running head "HOW AI WORKS" / "CONTENTS". H1 "Contents." 56/64 at y ≈ 204.
- Explainer italic 26/36 #666, 16px below H1: "Chapters are written as you reach them, shaped by your checks."
- Rows 64px below explainer: height 88 each; number column 60 (30/700); title 34 (current chapter 600-weight); dotted leader 2px #BBB filling remaining width (baseline-aligned); right state text.
- States: "CLEARED · 91" (22 caps #444) · "IN PROGRESS · P. 1" (22 caps #000) · unwritten chapters: title #999 + italic 26 "not yet written" #999, right-aligned, NO leader.
- Native: invisible 88px full-width tap targets on cleared/in-progress rows (navigate); unwritten rows inert. No visible buttons.

### Assertions
1. Unwritten rows have no leader dots; written rows do.
2. Exactly one row uses 600-weight title + "IN PROGRESS".
3. Row height 88px constant; tap anywhere in a written row navigates.

## 8. Print (rmapi) divergence
Shared components keep identical page-side chrome (borders, dotted rules,
labels, type). Differences, print only: the control strip zone prints a
scaffold instead of staying blank — confidence boxes ☐ unsure ☐ shaky
☐ confident ☐ sure and ☐ I don't know as printed 26px checkboxes laid out on
the same geometry as the native controls (inset 30, right-aligned group);
MCQ letters print as ☐ A ☐ B ☐ C in row 1's position. Folio and running heads
identical. No native layer exists; the shelf rule stays.

## 9. Geometry payload (server → client, per item)
```json
{ "page": 3, "item": "u2-q4", "kind": "mcq", "options": 3,
  "rect": [110, 760, 1400, 640],   // px; normalize by /1620,/2160
  "strip": 196 }                    // px reserved above inner bottom edge
```
Client derives: shelf rule line, control rows (§2.3), ink clip
(rect minus strip; constructed only). Every value above is a constant the
server enforces at render time.
