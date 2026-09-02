# Corpus Authoring Spec

Contract for every unit directory under `corpus/units/<id>-<slug>/`. The
book-server parses these files mechanically; the in-flight author LLM assembles
chapters from them. Deviating from the formats below breaks the pipeline.

## Files per unit

```
corpus/units/u3-attention/
  canon.md            # ground-truth exposition with inline beats
  depths/
    deeper-math.md    # per-section deeper treatment (full derivations)
    more-intuition.md # per-section gentler treatment (geometric/visual)
    se-analogies.md   # per-section software-engineering analogy PAIRS
  questions.yaml      # pretest + terminal-check item bank
  misconceptions.yaml # unit-local additions to the global bank (same schema)
```

## Audience (fixed for every file)

Expert software engineer (systems, infra, daily LLM-tooling user). Novice at
matrix calculus and ML notation. Therefore, always:
- **Math fully explained, code explained in one line.** Never both at depth.
- Real equations with dimensions annotated. Every symbol defined inline at
  first use, adjacent to the equation (split-attention rule). Never
  "recall eq. 3.2" - restate it.
- Worked numeric examples use tiny shapes (2- and 3-dim vectors, 2x3 matrices)
  with actual numbers that compute cleanly.
- Hyphens only, no em/en dashes. LaTeX in `$...$` / `$$...$$`.

## Chapter ordering (behavior first, symbols last)

Sequencing rules, derived from the open-courseware survey and the
learning-science evidence pass (2026-08-10; see DESIGN-ui.md commit trail).
Section ORDER in canon.md is what every learner gets - the author LLM can
rewrite prose but never reorder - so ordering mistakes cannot be repaired
downstream.

1. **Open with a contract and a concrete artifact, never a definition.** A
   short preamble (before the first `## `) states what the reader will be
   able to do or explain. The first section starts from observed behavior,
   real input/output, or a specific question - definitions may not appear
   above the fold.
2. **Contrasting cases before mechanism.** The opening section presents
   concrete cases engineered so the mechanism's absence is felt, plus a
   commit-first beat asking the reader to attempt them, then resolves. A
   posed-but-unanswered problem is the load-bearing element; a motivating
   story is not a substitute.
3. **No formal symbol in the first fifth of a chapter**, and never more than
   one new symbol per paragraph. Symbols appear just-in-time, in a sentence,
   at the exact point of first need, with shape stated in place.
4. **No up-front notation tables.** At most 3-5 named components with
   one-line plain-language groundings may open a chapter. The full symbol
   reference lives in a closing `## Notation ...` section (canon-only, no
   depth variants) - a lookup aid, never the reading path.
5. **Single element before matrix form.** Every mechanism is shown once for
   one concrete element (one token, one row, real numbers, or a short loop)
   before its batched/matrix form, and the matrix form arrives with an
   explicit "same computation, all n at once" bridge.
6. **Baseline-then-delta.** No component is introduced before the reader has
   seen the specific failure it fixes. If the failure cannot be named, the
   component is in the wrong chapter.
7. **Declare simplifications and name where they return** ("ignoring
   positions for now; u3 section 4 brings them back") - deferred rigor is
   fine, silent omission is not.
8. Material that EXPLAINS anomalies (tokenization pathologies, numerics,
   interpretability) goes late, where an anomaly exists to be explained -
   not first as a prerequisite.

Robustness note: depth variants swap section prose wholesale, so content
that must survive any depth (opening beats, notation references,
misconception corrections transplanted between units) belongs in beats or
in canon-only sections with no depth-variant heading.

## canon.md structure

1. **Front matter** (YAML): `unit`, `title`, `concepts` (IDs from syllabus.yaml),
   `assumes` (concept IDs treated as known - the author LLM must respect this).
2. **Sections** (`## `), one per major idea, each 3-8 min of reading. Each
   section that teaches a concept with a known misconception MUST use the
   refutation structure:
   > You probably think X. Here is the specific prediction X makes that fails.
   > Here is why X is appealing. Here is what is actually true.
   Reference the misconception ID in an HTML comment: `<!-- refutes: M1 -->`.
3. **Interaction beats**: 6-10 per unit, embedded where they belong in the
   flow (mid-derivation, not appended). Fenced block, language tag `beat`,
   YAML body:

   ````
   ```beat
   id: u3-b2
   type: predict | completion | self-explain | compute
   concept: c-sdpa
   prompt: |
     We have scores $S = QK^T$. Before reading on: what goes wrong if we
     softmax $S$ directly when $d_k = 512$, and what is the fix?
   answer: |
     Dot products grow with d_k (variance ~ d_k), pushing softmax into
     saturated regions with near-zero gradients. Fix: divide by sqrt(d_k).
   rubric: |
     Must identify: (1) score magnitude grows with dimension, (2) softmax
     saturates -> gradients vanish, (3) scale by sqrt(d_k). Any 2 of 3 = pass.
     Mentioning "numerical overflow" alone = M7-style confusion, fail.
   check: llm            # llm | exact | numeric(tolerance)
   ```
   ````

   - `compute` beats use `check: numeric(0.01)` or `exact` - the server grades
     them mechanically; `answer` must then be the bare number/string.
   - `completion` beats: include the full worked sequence in `prompt` with
     2-3 steps replaced by `____`; blank the steps that carry the concept
     being taught, and note in a comment which steps to blank in variants.
   - Every `self-explain` beat MUST have a rubric (unadjudicated
     self-explanation is worse than none).
3. **Fade sequences**: every mathematical procedure appears 3 times: fully
   worked in prose, as a `completion` beat, and as a solo item in
   questions.yaml. Mark the worked version with `<!-- fade: <procedure-name> -->`.

## depths/ files

Same `## ` section headings as canon.md (exact match - the author LLM swaps
sections wholesale). Content rules:
- `deeper-math.md`: derivations, Jacobians, the steps canon elides.
- `more-intuition.md`: geometric pictures, physical analogies, no new symbols.
- `se-analogies.md`: analogies ALWAYS in aligned pairs ("attention is like a
  soft dictionary lookup AND like a weighted DB aggregate; the shared
  structure is X") and ALWAYS with "where this breaks:" stated. A lone
  unbounded analogy is a spec violation.

## questions.yaml schema

```yaml
pretest:            # 2-3 items, EXPECTED to fail, calibration + pretesting effect
                    # (exception: a unit with `calibration: true` in canon front
                    # matter has NO pretest and no body sections - its whole
                    # check bank is one progressive series, delivered complete
                    # and in authored order, and its gate always passes)
  - id: u3-p1
    concept: c-sdpa
    prompt: "..."
    answer: "..."
    check: llm      # llm | exact | numeric(tol) | choice
    rubric: "..."   # required when check: llm

check:              # >=10 items so the server can compose an 8+ item check
  - id: u3-q1
    concept: c-sdpa
    kind: constructed   # constructed (>=60% of items) | mcq
    congruent: true     # true if the response form matches the goal (derive/compute/explain)
    callback_eligible: true   # may appear in later units' cumulative checks
    prompt: "..."
    answer: "..."
    check: llm
    rubric: |
      ...explicit pass/fail criteria; list what a correct answer MUST contain,
      what earns partial credit, and which misconception each characteristic
      error indicates (cite bank IDs)...
    difficulty: core    # warmup | core | stretch
    band: 3             # calibration units only: 1-5 on the screener's own scale
  - id: u3-q7
    concept: c-qkv
    kind: mcq
    congruent: false
    prompt: "..."
    options:
      - text: "..."
        correct: true
        explain: "why right"
      - text: "..."
        misconception: M1      # REQUIRED on every distractor - bank ID or unit-local ID
        explain: "why attractive, why wrong"
    difficulty: warmup
```

Rules:
- Every distractor keyed to a misconception ID. No throwaway options. Every
  option gets an `explain` (feedback adjudicates each option, not just the right one).
- >=60% of check items `kind: constructed` with `congruent: true`.
- Anything a program can check (`numeric`, `exact`, `choice`) must NOT use `check: llm`.
- At least 2 items per unit marked `callback_eligible` and at least 1 `stretch`
  item that the >90% path can use.

### Calibration units: bands and windows

A calibration unit (`calibration: true` in canon front matter) ends its
questions.yaml with a `screener` - one self-placement question with five
options, novice to expert - and `calibration_sets`, the series delivered for
each of those five answers:

```yaml
screener:
  id: u0-s1
  check: screener
  prompt: "Before the questions: how would you rate..."
  options:            # exactly five, in ascending order
    - text: Absolute novice
    - text: Seen the words, could not compute
    - text: Could work through it slowly
    - text: Comfortable
    - text: Expert - I could teach it

calibration_sets:     # one list per level, item ids from check, easy to hard
  1: [u0-n1, u0-n2, ..., u0-q1, u0-p2]
  ...
```

Every item in the bank carries `band: 1..5`, the screener level at which a
learner is expected to get it right without strain. The point of the screener
is to size the learner WITHIN the level they claimed, so each level's series
is a window, not a ladder:

- at least 3 items from the level's own band (the bulk);
- at least 1 from the band below (a floor - catches an overrated learner);
- at least 1 from the band above (a ceiling - catches an underrated one);
- ordered easy to hard by band;
- and the bank holds at least 3 items in every band, so no level is empty.

The linter enforces all of this. The failure it exists to prevent: one ladder
trimmed from the top, where "absolute novice" receives the same first ten
matrix-product items as level 3 and the self-rating changes nothing the
learner can feel. Band 1 is the arithmetic the subject is built from; band 2
is recognising the objects or following a recipe that is spelled out; band 5
is what an expert answers on sight.

## Tone

Direct, technical, zero fluff. Respect the reader's expertise on the systems
axis. Refutation is confrontational by design - "this is wrong, here is the
failing prediction" - never hedged into mush. No exclamation marks, no
"simply", no "just".
