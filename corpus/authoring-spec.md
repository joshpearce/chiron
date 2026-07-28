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
                    # (exception: u0's pretest IS the calibration instrument and
                    # spans the whole math floor, so it runs 8 items)
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

## Tone

Direct, technical, zero fluff. Respect the reader's expertise on the systems
axis. Refutation is confrontational by design - "this is wrong, here is the
failing prediction" - never hedged into mush. No exclamation marks, no
"simply", no "just".
