# The Data Book: concept roadmap and outline

Volume 2 of the Chiron book. Premise under test: LLM output can be correlated
to training input well enough to meter compensation for data providers - a
Spotify model for training data. This document synthesizes a four-agent
literature and ecosystem pass (2026-08-30) into (1) a verdict on the premise,
(2) a proof-of-concept design the evidence supports, (3) the concept roadmap
from the learner's current state to PoC-capable, and (4) the book outline.
Full agent reports live in the session transcript; citations inline here are
the load-bearing ones only.

## 1. The premise, audited

### What the evidence confirms

- **The open web IS closing, faster than the premise assumed.** Cloudflare
  default-blocks AI crawlers for ~20% of websites (Jul 2025), shipped
  Pay Per Crawl (HTTP 402 + signed crawlers) and replaced it with Pay Per
  Use (Jul 2026, pay on value-created); mixed-use crawlers are blocked by
  default from ad-bearing pages starting 2026-09-15. Fifty-plus licensing
  deals were signed in a year with Cloudflare's blocking as leverage.
- **The Cloudflare call was right - and executed.** They own the chokepoint,
  the crawler-identity standard (Web Bot Auth, enforced at Cloudflare, AWS,
  Akamai, Vercel), the preference standard (Content Signals), the payment
  rail (Merchant of Record), and since Jan 2026 the marketplace: they
  acquired Human Native, the training-data licensing startup.
- **The correlation problem is real, named, and open.** The field calls it
  training data attribution (TDA). The key split (ATTRIB 2026 CFP):
  *contributive* attribution (which data caused this output) vs
  *corroborative* (which data supports it). A royalty model needs the
  first; every deployed commercial system does the second, or neither.
  Nobody has shipped an end-to-end tagged-corpus -> trained-model ->
  per-source-payout demonstration. That gap is unoccupied.

### What the evidence challenges

- **"Spotify for training data" has been pitched, funded, and partly
  consolidated.** ProRata ($75M+, per-output attribution + 50% rev share,
  1,000+ publications), TollBit ($31M, literally pitched as "Spotify for AI
  data licensing"), Human Native (bought by Cloudflare), Sureel (bought by
  Warner Music). A VC places this pitch in thirty seconds.
- **No frontier lab pays into any marketplace, by any mechanism.** RSL:
  ~1,500 endorsers, zero AI licensees. IETF AIPREF: no production readers.
  Pay Per Crawl never left beta. Supply-side coordination has no
  counterparty yet.
- **Courts are removing the forcing function.** Bartz v. Anthropic ($1.5B,
  final approval Jul 2026) priced *acquisition*, not use: training on
  lawfully acquired copies is fair use (Alsup, Chhabria, UK High Court all
  converge). The legal pressure produces one-time clean-copy purchases,
  not royalty streams - unless NYT v. OpenAI (trial late 2026) wins on
  output substitution.
- **Deals are migrating away from training rights.** Only ~40% of recent
  deals include training rights (down from near-universal in 2023);
  attribution/live-access (RAG-time) deals grew 2 -> 18 -> ~34/yr. Labs
  spend billions on data - on expert/labeled data (Mercor: ~$2B annualized
  revenue) - while ALL disclosed content licensing totals low single-digit
  billions cumulative.
- **Training-time attribution at frontier scale is not yet credible as a
  payment basis.** MAGIC (2025) measured scalable gradient methods at
  "no better than random guessing" against ground truth on Gemma-2B;
  TrackStar (8B params, 160B tokens - the largest honest run) found BM25
  beats influence methods at finding the document containing a fact; four
  independent results say influence functions are unreliable on LLMs.
  And Spotify's own pro-rata mechanics import known pathologies
  (winner-take-most, cross-subsidy, stream-farming; content farms are
  cheaper than stream farms).

### The recalibrated thesis

The defensible version: **the web closes, the gatekeeper meters access, and
the unsolved technical layer is trustworthy measurement.** Three open
commercial slots the incumbents do not occupy:

1. **Independent verification/audit** - every deployed scheme self-reports
   usage (Microsoft PCM meters itself; Perplexity counts its own citations;
   Cloudflare measures its own traffic). EU AI Act Art. 53(1)(d)
   enforcement began 2026-08-02 (fines to EUR 15M / 3% revenue) - the only
   regulatory forcing function in the space, and it sells to the buy side.
   The regulator itself names the gap: the Commission's explanatory notice
   (C(2025) 8311, para 26) states the AI Office will supervise
   "without performing a work-by-work assessment or checks whether
   specific content has been used" - mandatory disclosure, coarse
   (top-10% of domains, three size buckets), self-reported, six-month
   refresh, explicitly unverified.
2. **Leakage/syndication tracking** - blocking your origin is worthless
   when your content syndicates to a hundred crawlable copies. Tractable
   today (near-dup detection at web scale); nobody sells it.
3. **Honest attribution science** - the SNR question. "What's a Credit
   Worth?" (2026) proved that below an attribution signal-to-noise
   threshold, flat fees are welfare-optimal. Measuring where the royalty
   signal drowns in training noise is a publishable, pitchable result in
   either direction.

The PoC should prove capability across this whole layer, not bet on the
crowded RAG-citation lane.

## 2. The proof of concept the evidence supports

Two agents independently converged on the same design. Total budget
< $500; most of it runs on the MacBook. The provenance plumbing - not the
math - is the novel contribution, and it is pure systems engineering.

**Substrate.** peS2o (40M open-access papers, ODC-By, per-document
licenses) filtered to a corpus tagged into 10-12 named "sources" (venues /
author groups / subfields). Provenance manifest mapping shard+offset ->
doc -> source; document-masked packing so no training sequence blends
sources; deterministic replayable dataloader; dedup that RECORDS duplicate
clusters instead of silently choosing which source gets credit (dedup is a
payout decision - nobody upstream has had to treat it as one).

**Stage 1 - Ground truth (laptop, ~$0; rented H100 rerun ~$100).**
10-30M-param nanoGPT on 300M-1B tokens. Leave-one-source-out retrains for
every source, plus 200-500 random-subset runs, >= 3 seeds per condition
(seed noise, not combinatorics, dominates the error budget). Yields TRUE
per-source contributions by actual counterfactual. Math required: none -
subtraction and linear regression.

**Stage 2 - The fair split (a few hundred dollars).** Exact Shapley over
the sources: at n <= 12-15 all 2^n coalitions are enumerable (~68 GPU-hours
at n=12) - no approximation for a critic to attack. Value function:
held-out log-likelihood (smooth, no labels needed). Compute Banzhaf and
LOO from the same cached utility table. Reimplements Wang-Deng-Chiba-Okabe-
Barak-Su (arXiv:2404.13964, the Shapley-royalty paper) which published no
code and no cost figures - a faithful open reimplementation with the
diagnostics they skipped is a genuine contribution. Include, or it's a toy:
the MTM-residual diagnostic (is the ranking meaningful at all), the
seed-noise SNR report per source, and the shell-company attack demonstrated
then fixed (split one source into three shells, show the payout inflate,
apply Faithful Group Shapley / identity constraints).

**Stage 3 - The demo artifact (~$150).** nanochat-class 561M model
(~$92, 3.3h on 8xH100) on 70% FineWeb-Edu (one "background" source) + 30%
tagged papers, SFT'd so a person can type at it. Attribution stack, all
three layers side by side:
- infini-gram suffix-array index -> verbatim span matches (fast, honest,
  answers the *corroborative* question);
- TracIn/gradient influence vs the Stage-1 ground truth (the
  *contributive* question, with measured trustworthiness);
- In-Run Data Shapley royalty ledger computed during training, rendered
  as the revenue split.
The demo IS the contrast: "what the model copied" vs "what shaped it" vs
"who gets paid," with the correlation numbers from Stage 1 saying how much
to trust each.

**Stage 3b - The provable half (verification layer, same substrate).**
The one family that yields the word "proof" without abuse: prospective
keyed watermarks. Retrospective methods (membership inference) cannot
produce a sound false-positive rate - you cannot sample the null
"the model was NOT trained on my data" (Zhang/Tramer, SaTML 2025);
pre-embedded keyed secrets with unpublished matched controls construct
the null by design. Concretely: inject fictitious-knowledge watermarks
(Cui et al., ACL 2025 - coherent invented-entity statements that survive
dedup, filtering, AND instruction tuning; ~256 docs / <0.1% of corpus;
detectable by asking the model a factoid question, z around -5) into one
source before Stage 3's training run; detect with a pre-committed
one-sided z-test. The commercial insight from the verification survey:
**the moat is the pre-commitment ledger, not the ML** - register corpus,
generate keyed watermarks + never-published controls, publish a
timestamped key commitment, later issue a third-party-verifiable signed
report. A notary business with a statistics engine attached. Adjacent
easier sales sharing the machinery: exclusion certification (PRISM -
developers proving they DIDN'T train on something; friendly compliance
sale) and EU Art. 53 disclosure-conformance checking. Known-dead ends to
avoid: zkML proof-of-training (~10^4 years for a 1B model at published
prover throughput), Proof-of-Learning (spoofable), TEE attestation
(real, <12% overhead, but cooperative-only - a licensing-integrity
feature, not an enforcement tool). Honest limitation to lead with:
watermarks only protect data published AFTER embedding.

**Stage 4 - Optional scale story.** Continued-pretraining attribution on
Olmo 3 7B (~$60-90), evaluating on post-cutoff papers to control for base
knowledge. Answers "does this survive contact with a real model."

## 3. Concept roadmap (learner: Chiron vol. 1 spine, pre-calc + vol-1 math)

Three interleaved tracks. The learning path follows the build path -
every math concept arrives the week the build needs it (the vol-1 ordering
rules applied to a curriculum).

**Track A - Systems (no new math, start immediately)**
corpus acquisition & extraction -> filtering/dedup (MinHash, suffix
arrays - data-structure work) -> provenance manifest & document-masked
packing -> tokenizer training -> nanoGPT training loop mechanics ->
eval harness (BPB, seeds, noise floors) -> suffix-array search index ->
gradient store + demo app.

**Track B - Math (three tiers actually needed, one optional)**
- Tier 1 (2-4 wks): probability & hypothesis testing - expectation,
  variance, binomial z-test, paired tests, likelihood ratios,
  ROC/TPR-at-FPR, and THE concept: what it means to sample from a null
  (why canaries are provable and post-hoc membership inference is not).
  Plus the four things that convert a number into evidence: valid null
  construction, multiple-hypothesis correction (Bonferroni/BH),
  dependence between test units, and pre-registration discipline.
  This tier alone carries the entire verification product.
- Tier 2 (3-6 wks): combinatorics & expectation over permutations -
  the Shapley value END TO END. No calculus anywhere in it. The
  fairness axioms are the VC/lawyer-facing argument.
- Tier 3 (builds on vol-1 u2/u5, 4-8 wks): gradients as vectors, dot
  products as alignment, the chain rule - already 70% covered by
  vol. 1; TracIn is `grad(test) . grad(train)` summed over checkpoints.
- Tier 5 (optional extension, months): second-order calculus, Hessians,
  EK-FAC - required only by the method family with the WEAKEST
  empirical support. Deliberately deferred; the book teaches why.

**Track C - Domain literacy (reading, interleaved)**
FineWeb paper (the ablation methodology - two seeds per condition,
low-variance metrics - is the experimental discipline the PoC needs) ->
Chinchilla + data-constrained scaling (repeating data ~4 epochs is nearly
free; why a 500M-token corpus honestly supports a 2B-token run) -> Grosse
influence functions (findings, not mechanics) -> TrackStar -> In-Run
Shapley -> OLMoTrace -> the economics papers (SNR/flat-fee threshold,
73-deals analysis, semivalue critiques) -> ecosystem primary sources
(Cloudflare announcements, RSL spec, Bartz settlement, EU AI Act Art. 53).

**Milestones** (a sequence, not a schedule - this runs alongside other
work at whatever pace it gets)
1. Tagged mini-corpus + 10M model trains on the Mac overnight.
2. LOO ground-truth table with seed-noise bars (first real result).
3. Watermarks planted with committed key; detection with a sound p-value
   (first verification-product result).
4. Exact Shapley split + diagnostics + shell-company demo.
5. 561M artifact + the full demo, verification leading.
6. Writeup: the notary story, measured SNR, method-vs-ground-truth
   correlations, the pitch narrative. (NeurIPS ATTRIB workshop, Sydney,
   Dec 2026, is the room this audience is in - a target of opportunity,
   not a deadline.)

## 4. Book outline (volume 2, working title: "Where the Words Come From")

Same corpus format, same pedagogy (vol-1 authoring spec + ordering rules).

**First-principles rule (primer style): assume nothing is retained.**
Vol-1 concepts are never referenced as known - every concept a unit
needs is re-derived compactly from first principles at its point of use,
even where vol 1 taught it. `assumes:` links serve sequencing and the
planner's compression decisions only; calibration (v0) dials how BRIEF
the re-derivation gets, never whether it happens. A learner with large
gaps must be able to read any unit cold and lose nothing but time.

Every unit's capstone beats ARE PoC build steps - the book produces the
artifact.

- **v0 - Calibration** (20 min). Measures retention of vol-1 gradients/
  loss/tokens + Tier-1 probability baseline. Placement decides how much
  of v5/v6's math gets the full treatment.
- **v1 - The closing web and the correlation problem** (30 min). Opens
  from observed behavior: a real 402 response from a Cloudflare-gated
  site; the Reddit-Google number; the Bartz price-per-book. States the
  contributive/corroborative distinction as the book's organizing split.
  Contrasting cases: three compensation schemes (flat fee, citation
  counting, causal attribution) applied to one worked example - where
  each one misallocates. Honest limits: the skeptical case, stated
  strongest-form. [ecosystem report]
- **v2 - From glob to corpus** (45 min). The data pipeline as practiced:
  extraction, quality filtering (why a classifier is the biggest lever -
  DCLM/FineWeb-Edu), dedup (MinHash + suffix arrays; dedup as a PAYOUT
  decision), decontamination, tokenizer training, mixing, packing with
  document masks, the provenance manifest. Almost all systems engineering.
  Capstone: build the tagged peS2o corpus.
- **v3 - Training runs you can afford** (40 min). C = 6ND budgeting; what
  $92 buys; LR schedules as recipes (incl. WSD and why branchable
  checkpoints matter for attribution); data-constrained scaling (epochs);
  evaluation that means something at small scale (BPB, the FineWeb
  benchmark-selection discipline, seed noise as the enemy). Capstone:
  the 10-30M model trained overnight on the Mac, with honest error bars.
- **v4 - Ground truth: the counterfactual** (35 min). What "this source
  contributed X" MEANS: leave-one-out, subset regression, datamodels as
  a concept, the Linear Datamodeling Score, why single-example LOO drowns
  in seed noise but source-level does not. No new math. Capstone:
  the LOO table - the first defensible royalty-relevant number.
- **v5 - The fair split: Shapley from scratch** (50 min). The Tier-2 math
  unit. Cooperative games, the four axioms and why they uniquely force
  the formula, permutations and Monte Carlo, Banzhaf and the semivalue
  family, exact enumeration at small n. Then the adversarial reality:
  the shell-company attack, gameability critiques, the MTM diagnostic,
  and the SNR/flat-fee threshold - when the honest answer is "don't build
  the royalty scheme." Capstone: the exact Shapley split + attack demo.
- **v6 - What the model remembers, and what counts as evidence**
  (45 min). The Tier-1 stats unit, organized around the book's second
  great split: retrospective vs prospective evidence. Memorization
  scaling laws (~3.6 bits/param capacity; frequency relative to corpus
  size governs), extraction attacks (the strongest legal evidence,
  available for almost nothing), why membership inference fails (no
  sampleable null; coin-flip decision instability; blind baselines beat
  published attacks), dataset inference at collection level as the
  honest retrospective fallback. THE concept: what it means to sample
  from a null, and the four disciplines that convert a number into
  evidence (valid nulls, multiple-testing correction, dependence,
  pre-registration). Capstone: reproduce a membership-inference failure
  and an extraction success on the v3 model - feel both limits.
- **v7 - The notary: proving it, and selling the proof** (50 min).
  The audit-product unit. The prospective family: canaries,
  fictitious-knowledge watermarks (coherent invented-entity statements
  that survive dedup, filtering, and instruction tuning), paired-
  rephrasing designs (STAMP) - constructed nulls, sound p-values, the
  z-test as the whole detector. Courts credit acquisition records and
  verbatim extraction, never statistical MIA; a keyed watermark is the
  only statistical evidence built to survive expert challenge. The
  product mechanics: the pre-commitment ledger (register corpus, keyed
  watermarks + never-published controls, timestamped key commitment,
  third-party-verifiable signed reports); exclusion certification
  (PRISM) and EU Art. 53 conformance checking as the friendly first
  sales; TEE attestation as the cooperative complement; zkML and
  Proof-of-Learning as the marked graves. Capstone: plant watermarks in
  one source with a committed key, detect via QA with a pre-registered
  z-test, emit the signed report.
- **v8 - Tracing by gradient** (45 min). The Tier-3 unit, direct sequel
  to vol-1 u5: gradients as directions, dot products as "pushed the same
  way," TracIn over checkpoints, projections (JL intuition), why
  influence != entailment (TrackStar/BM25), what MAGIC's verdict means
  for trust, In-Run Shapley as the practical convergence. Capstone:
  TracIn scores rank-correlated against v4 ground truth - the number
  that says whether cheap attribution can be trusted.
- **v9 - The artifact and the argument** (40 min). Retrieval surface
  (suffix arrays, OLMoTrace's honesty about causality), assembling the
  demo - verification leading, royalty split as the vision - then the
  business argument: the ecosystem map, the three open slots, Spotify's
  pathologies transferred, what the VC meeting needs (the notary
  demo, the measured-SNR headline, the attack-and-fix). Synthesis unit
  in the vol-1 u9 style.
- **Extensions** (unlock > 90%): x-v1 influence functions properly
  (Tier-5 math: Hessians, EK-FAC, and the critique literature);
  x-v2 source-aware training (document-ID tokens - the "labs build it
  in" future the original thesis imagined); x-v3 the audit stack in
  depth (EU Art. 53 template mechanics as a product spec; TEE-attested
  training as a cooperative licensing tool; why zkML and
  Proof-of-Learning are dead ends; C2PA's actual scope - its
  do-not-train assertion was REMOVED in v2.0);
  x-v4 fine-tune attribution at 7B (the Stage-4 experiment).

Verification caveats carried from the research pass: two legal details
need primary-source confirmation before appearing in any pitch - the
NYT complaint's verbatim-output exhibit (agent sources conflicted) and
the precise disposition of Getty v. Stability UK (corroborated by a
second agent, but confirm against the judgment itself).

## 5. Decisions (settled 2026-08-30)

1. **PoC framing: both demos on the shared substrate, verification
   leading.** The pitch narrative: the verification notary is the
   product a solo founder can ship into a regulator-named gap; the
   royalty split is the vision slide - where the measurement layer goes
   once it is trusted. The royalty science stays honest-SNR-first.
2. **Sequencing: interleaved.** Unit capstones ARE the PoC build steps.
3. **Corpus: research papers** (peS2o / CC-licensed subset).
4. **Verification grows to two spine units** (v6 evidence, v7 audit
   product) - reflected in the outline above.
5. **No deadline pressure.** This runs alongside other work; milestones
   are a sequence, not a schedule. External events (NYT v. OpenAI
   ruling, EU enforcement actions) get monitored, not raced.
