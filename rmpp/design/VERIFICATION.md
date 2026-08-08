# Design verification - SPEC.md + screens.html

Verified 2026-08-08 against DESIGN-brief.md, the existing architecture, and
local Chrome renders of all 15 frames (render-frames.sh -> frames/).

## Verdict: implementable as specified. No constraint violations found.

## Brief compliance

- E-ink rules: no animation anywhere; waits/errors are static dinkus screens;
  "pressed" is a momentary inversion; page changes are full swaps. PASS.
- Monochrome-first, one accent (#8A3D30) reserved for misses/miscalibration/
  errors. PASS.
- Two-renderer split: fully respected, and the spec's one big architectural
  call - results become server-rendered pages - is the CORRECT resolution of
  the math-in-results problem the brief flagged. Native never renders TeX.
- Geometry contract: honored and simplified. 640/970 fixed box heights with
  a published rect+strip per item replaces the 1/6-slot estimator. All
  constants server-enforceable.
- Pedagogy: confidence-before-reveal kept; IDK on every item, styled quieter
  than the attempt path (1px #666 vs 2px #000); ink only for constructed;
  READ AS transcript row mandatory (audit trail); reveals only on results.
- Measurable assertions: present per screen and screenshot-checkable.

## Spec-internal consistency checks (all pass)

- Strip arithmetic: MCQ 26+64+16+64+26 = 196; constructed row centers at
  inner-bottom-60 with strip 120.
- Packing sums: 970+970+20 gap = content height exactly; 3x640+2x20 = 1960.
- Gate tick x = 110 + 1400*0.80 = 1230; fill width formula consistent for
  82%/64% examples.
- Bottom band: controls at y 2078-2142 = 64px centered in the 100px bottom
  margin; folio block 2100-2126 does not collide horizontally.
- HTML matches spec values (strip heights 121/197 include the 1px shelf).

## Notes and deltas for implementation (not blockers)

1. Palette table lists 7 tokens but type specs also use #111/#222/#333/#555/
   #666/#999. Treat as extended grays; they quantize fine on 16-level e-ink.
2. Glyphs: verdict marks and the dinkus render via font fallback in Chrome;
   verify Source Serif 4 coverage at implementation, else pin a fallback
   font for those characters on both sides.
3. "Partial refresh" language in 0.7 is aspirational on the Paper Pro -
   xochitl owns refresh policy; we implement the repaint scoping and accept
   whatever the compositor does.
4. Swipe page turns need pen/touch discrimination (pen = ink, touch =
   swipe); emulator mouse maps to touch outside boxes, ink inside.
5. Results-as-pages: grading wait now includes results typesetting; results
   pages form their own series with their own folio; action label is
   server-driven. Requires a results renderer server-side and a series
   switch in the pages meta.
6. Contents/spine and break screens are new features (phase 2); nothing in
   the core loop depends on them.
7. Fonts: bundle Source Serif 4 + Source Sans 3 TTFs (OFL) for both the
   server wrapper CSS and QML FontLoader; screens.html uses CDN copies for
   preview only.

## Implementation order

1. Server page stylesheet v2 (grid, running head, folio, type scale, box/
   strip/beat anatomy, fonts) + geometry payload rename to rect+strip.
2. Results renderer (headline/gate bar/tally/dek/entries, pagination) and
   client switch to page-based results with native action button.
3. Client controls restyle (state matrix), 5-way IDK/confidence exclusivity,
   completeness gating + caption, ink clipping to rect-minus-strip.
4. Placement + wait/error screens to spec.
5. Phase 2: contents/spine, break screens, print-scaffold divergence.
