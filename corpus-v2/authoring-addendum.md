# Volume 2 authoring addendum

The vol-1 contract applies in full: read `corpus/authoring-spec.md` first -
file formats, beat schema, refutation template, fade sequences, audience
rules, tone, and the "Chapter ordering (behavior first, symbols last)"
section all bind here. This file is only the delta.

## First-principles rule (overrides any contrary instinct)

Never treat a vol-1 concept as retained. Every concept a unit needs -
gradients, loss, tokens, softmax, anything - gets a compact
first-principles re-derivation at its exact point of use, even though
vol 1 taught it. The re-derivation is BRIEF (a paragraph or two, one
worked number), not a re-teaching; `prereqs` and calibration dial its
compression, never its existence. Test: a learner who read nothing else
can follow this unit cold, losing only time.

## Source material and figures

- Authoring sources live in `corpus-v2/research/` (ecosystem.md,
  pipeline.md, tda.md, verification.md) plus `DATABOOK-ROADMAP.md`.
  Ground every concrete number in them.
- The research files carry inline verification flags. A flagged figure
  may appear in prose only with its uncertainty stated, and MUST NOT
  appear in a beat answer, question answer, or rubric.
- This learner is also a founder in this space: where a misconception is
  pitch-fatal (D10, D12, D13, D20), say so explicitly in the correction.

## Namespace

- Units `v0`-`v9`, extensions `x1`-`x4`; concept IDs `d-*`; global bank
  IDs `D1`-`D20` (in `corpus-v2/misconception-bank.yaml`); unit-local
  misconception IDs `V<unit>-M<n>` (e.g. `V2-M1`).
- Beat IDs `v<unit>-b<n>`. Fade names must be unique across BOTH volumes
  (prefix with the unit, e.g. `v5-shapley-by-hand`).
- Vol-1 misconception IDs (M*, U*-M*) must not be referenced here.

## Capstone beats (the interleaved-build contract)

Each unit's closing section frames the PoC build step ("at the bench")
in prose - what to build, what artifact results, what it feeds later.
But every GRADED beat must be answerable from the unit's text alone:
test the understanding the build step requires (predict what the
manifest must record, compute the coalition count, state what the
z-threshold means), never the learner's local machine state. A beat
whose answer depends on having run the build is a defect.

## Voice notes for this volume

- The book's two organizing splits recur everywhere; keep the vocabulary
  exact: contributive vs corroborative (v1 onward), retrospective vs
  prospective evidence (v6 onward).
- Money numbers age fast. State the as-of date ("as of mid-2026") on
  market figures, never on math.
- The skeptical case is load-bearing, not hedging: units v1, v5, v7 and
  v9 each state the strongest argument AGAINST their own subject and
  answer it honestly or concede it.
