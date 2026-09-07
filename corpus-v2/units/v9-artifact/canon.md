---
unit: v9
title: "The artifact and the argument"
concepts:
  - d-retrieval-attribution
  - d-demo-assembly
  - d-pitch-economics
assumes:
  - d-semivalue-family
  - d-precommitment
  - d-tracin
---

Nothing in this unit is new. Every mechanism has been built: a tagged corpus, a
trained model, a counterfactual table, a fair split, a watermark, a signed
report, a gradient trace. What is new is the *order*, the *handoffs*, and the
*argument the assembled thing supports*. Individual mechanisms you can recite;
a trace you can either run or not, and only the second predicts whether you can
answer the question after the demo.

There are two destinations. One is a screen with three panels on it. The other
is a one-page memo that says what the screen proves and, more importantly, what
it does not. By the end of this unit you can build both, and you can defend
every sentence in the second one line by line.

Read this with a pen. The unit stops repeatedly and asks you to supply a claim
classification, a handoff, or a number before it gives you one. Reading the
answer first costs you the only thing being trained here, which is retrieval
under sequencing pressure rather than recognition.

## The screen

Someone types a question into the model you built. Not a frontier model - the
561-million-parameter one from the realistic-scale recipe, trained on 70%
general background text and 30% papers tagged into twelve named sources, SFT'd
just enough that a person can type at it.

```
> What does document-masked packing do to attribution?

Packing concatenates documents into fixed-length training sequences.
Without intra-document causal masking a single sequence blends text
from several documents, so a per-source attribution measured on that
sequence is contaminated at the input. Document-masked packing
prevents a training sequence from blending two sources, which is
what makes per-source attribution well defined.
```

Under the answer, three panels and a footnote.

**Panel 1 - Corroborative: text this corpus contains.**

| span (verbatim, from the answer) | tokens | source | occurrences in corpus |
| --- | --- | --- | --- |
| "prevents a training sequence from blending two sources" | 9 | source 3 | 1 |
| "a per-source attribution measured on that sequence is contaminated at the input" | 12 | source 3 | 1 |
| "intra-document causal masking" | 4 | 9 sources | 812 |
| "fixed-length training sequences" | 4 | 11 sources | 3,140 |

Ranked by rarity, not by length. The two rare spans sit at the top and both
resolve to one document in source 3. The panel's header, in the interface, is
one sentence long: *these documents contain this text; they are not claimed to
have caused it.*

**Panel 2 - Contributive: sources that moved the model, with a trust number.**

| rank | source | influence score | ground-truth rank |
| --- | --- | --- | --- |
| 1 | source 3 | 0.42 | 3 |
| 2 | source 7 | 0.31 | 1 |
| 3 | source 11 | 0.09 | 2 |

Header: *Spearman rank correlation against the counterfactual table: $\rho =
0.61$, measured over 12 sources at 3 seeds.* Beneath it, in the same type size
as the ranking: *the counterfactual table ranks source 7 first and source 3
third. This panel disagrees with the truth it was graded against, by the amount
the header states.*

(Throughout this unit $\rho = 0.61$ is the figure carried so the arithmetic is
concrete. Your own build produces its own number, and the number, not the
ranking, is the deliverable.)

**Panel 3 - Allocation: a $1 pool split by the in-run ledger.**

| source | share of $1 |
| --- | --- |
| source 7 | $0.19 |
| source 3 | $0.14 |
| source 11 | $0.11 |
| nine others | $0.56 |

Beneath the pie: *signal-to-noise ratio of this allocation: 1.37, against a bar
of 2. Verdict at this seed count: flat fee.* The pie is rendered grey, and the
interface says why: the split is displayed as an illustration of a rule, not as
a schedule of payments, because the differences between these sources are not
resolvable against the noise in the measurement that produced them.

**Footnote.** *Source 7 membership: proven. $z = 4.5$, one-sided
$p = 3.4 \times 10^{-6}$; family-corrected $p = 8.5 \times 10^{-5}$ at family
size $m = 25$. Key commitment `sha256:9c41...e07a`, timestamped 2026-03-14,
before this model was trained. Verifier script attached.*

That screen is the whole book. Four claims, four different epistemic statuses,
four different vocabularies, and they are printed next to each other precisely
so that nobody can slide between them.

Read the four in the book's words, because the words are load-bearing:

- Panel 1 makes a **corroborative** claim. These documents support this text.
  It is a statement about string containment in a corpus. It is exact, it is
  cheap, and it says nothing about causation.
- Panel 2 makes a **contributive** claim with **measured trust**. These sources
  moved the model in the direction of this answer, and here is how well that
  measurement tracked an actual counterfactual when we checked.
- Panel 3 makes an **allocation** claim. Given a value function, given a fixed
  pool, and given four fairness requirements, this is the unique split - and
  here is whether the split is resolvable against noise.
- The footnote makes a **proof** claim. A test fixed in advance, with a null
  constructed rather than estimated, fired at a stated false-positive rate.

Every product in this market picks one of those four and presents it as if it
were another. That is the entire commercial and intellectual content of the
screen: not that the panels exist, but that they are labeled.

```beat
id: v9-b1
type: predict
concept: d-demo-assembly
prompt: |
  Before reading on, commit to an answer. Three interventions, one at a time, on
  the system that produced the screen above.

  (a) You leave the model's weights untouched and rebuild the suffix-array
      index over a different corpus.
  (b) You retrain the model from scratch with source 7 removed, same seed,
      same everything else.
  (c) You retrain the model from scratch on the identical corpus with a
      different random seed.

  Which account of what changes, what cannot change, and which intervention the
  skeptic in the room actually proposes is right?
options:
  - text: |
      (a) changes Panel 1 completely and cannot touch Panels 2, 3 or the
      footnote: the influence scores are gradients of a model whose weights did
      not move, the allocation comes from a ledger accumulated inside a run that
      did not happen again, and the watermark test queries the same model.
      (b) changes all four, including the output string itself, and the
      footnote's test should now fail to fire, which is the negative control.
      (c) moves Panels 2 and 3 by seed noise - not by a fact about the world -
      which is what $\sigma$ and the SNR are printed to quantify; the footnote's
      $z$ moves but its validity does not. The skeptic proposes (a).
    correct: true
    explain: |
      Right, and (a) is the whole diagnostic: a quantity that moves when only the
      index moves is a property of the index. It is free, it takes minutes, and
      it decides whether the screen is a search engine with a language model
      attached or an attribution system.
  - text: |
      (a) changes Panel 1 and Panel 2 together, because the influence ranking is
      computed over the documents the index surfaces, so a new corpus gives new
      contributors. (b) changes everything. (c) leaves Panels 2 and 3 essentially
      fixed, since the corpus is identical and only the seed differs. The skeptic
      proposes (b), the expensive retrain.
    misconception: D1
    explain: |
      This is the citation-is-attribution error in its most durable form. No
      weight moved under (a), so nothing gradient-based and nothing accumulated
      during training can respond to it; Panel 2 is a fact about the model, Panel
      1 a fact about whatever text you pointed the index at. And (c) is exactly
      where Panels 2 and 3 do move, by seed noise.
  - text: |
      (a) changes Panel 1 only. (b) leaves Panel 1's spans as they stand - the
      text the answer matches is still sitting in the corpus - while Panels 2 and
      3 change; and if the verbatim evidence is unchanged after removing source
      7, that is itself the sign that source 7 contributed little. (c) moves
      Panels 2 and 3 by seed noise. The skeptic proposes (a).
    misconception: D3
    explain: |
      Removing a source changes the model, and the model produces the string the
      panel matches against, so Panel 1's spans change downstream of that before
      you even reach source 7's documents leaving the index. Verbatim match is a
      lower bound anyway: knowledge survives intact while the surface form
      changes, so unchanged spans license no conclusion about contribution.
  - text: |
      (a) changes Panel 1 only. (b) changes all four. (c) invalidates the
      footnote as well as moving Panels 2 and 3: $z$ shifts with the training
      seed, so the stated $p = 3.4 \times 10^{-6}$ is not stable and membership
      has to be re-established for every run. The skeptic proposes (c), because
      seed sensitivity is the cheapest attack on a proof claim.
    misconception: D19
    explain: |
      The measured statistic moves with the seed; the null does not. The
      footnote's false-positive rate comes from 250 never-published controls
      generated under a key committed before the model existed, and those
      controls can be resampled at will regardless of seed. A constructed null is
      exactly what makes this the only line on the screen using the word
      "proven".
check: choice
```

## The retrieval panel, built honestly

Panel 1 is the cheapest thing on the screen and it will be the most convincing
thing on the screen. Both halves of that sentence are the reason to build it
carefully.

The systems part, compactly, because this is your axis. Concatenate the whole
tokenized corpus into one array of $n$ tokens. Build a suffix array over it: the
list of all $n$ starting positions, sorted by the sequence that begins at each
position. Construction is a sort with a linear-time algorithm available, storage
is one integer per token plus the corpus, and the payoff is that any contiguous
token sequence's occurrences form a single contiguous block in the sorted list,
findable by binary search in time logarithmic in $n$. Counting occurrences of an
n-gram is then subtracting two array indices. This is what lets a public service
answer n-gram count queries over five trillion tokens in tens of milliseconds
with no GPU in the loop, and what lets the deployed version of this panel return
maximal verbatim spans against a 4.6-trillion-token corpus in a few seconds. The
compressed variant stores the index at under half the corpus size, and a
trillion-scale fuzzy matcher now answers approximate queries in well under a
second. None of that is research. It is a data structure you could implement
this weekend, and at twelve sources of papers your index fits in memory.

The matching part is a policy, not an algorithm. Take the model's answer, find
every span of it that occurs verbatim somewhere in the corpus, keep the
*maximal* ones (a span that is not contained in a longer matching span), and
discard spans that are so common they carry no information. Then rank what
survives. Ranking by length is the obvious choice and the wrong one: a
twenty-token span of boilerplate methods language is longer and less
interesting than a nine-token span that occurs exactly once. Rank by rarity -
some monotone function of how few documents contain the span - and the panel
surfaces the spans that actually distinguish one document from the corpus. In
the screen above, the nine-token span with one occurrence outranks the
four-token span with 3,140.

That is the entire panel. Suffix array, maximal spans, rarity ranking, a link
to the document, a link to the source. It runs on a laptop. It is exact. And it
answers the wrong question, which is why the header sentence matters more than
the ranking.

<!-- refutes: D1 -->
You probably think - or, more dangerously, your audience probably thinks - that
a panel showing which documents contain the model's words is showing where the
answer came from. Every AI search interface on the market is built to produce
that inference, the visual grammar is the grammar of a footnote, and footnotes
in human writing do trace where the writer got it. This is the third time this
book has attacked the belief, and the reason for the third time is that it is
the belief that will be in the room, held by the person you most need to
convince.

Here is the prediction it makes, and it fails in the cheapest possible
experiment, which is the one your skeptic already proposed. If a containment
panel traced causation, then holding the model completely fixed and rebuilding
the index over a different corpus could not change the panel. It changes
completely. Not one weight moved. Nothing the model learned changed. The panel
was never a fact about the model; it is a fact about what strings live in
whatever text you pointed the index at.

The team that shipped the deployed version of this panel over a
4.6-trillion-token corpus wrote the disclaimer into their own paper rather than
leaving it to readers: the retrieved documents should not be interpreted as
having a causal effect on the model's output. They built the best instance of
this panel that exists and they attached that sentence to it. Copy the sentence,
literally, into your header. It costs you nothing, because the panel is
genuinely useful for what it does claim, and it buys you the only thing that
matters in a room with a technical skeptic in it, which is that you said the
limitation before they did.

<!-- refutes: D3 -->
The second failure of this panel is subtler and it is the one that will bite in
the pitch. You probably think that verbatim matching, whatever its philosophical
status, at least catches the cases that matter - that if the model cannot
reproduce a source's text, that source did not contribute much.

Here is the prediction that fails. A filter that blocks all verbatim n-gram
emission perfectly is defeated by asking for the same content in a different
style: the knowledge is intact, only the surface form changed. And the direction
of the failure gets worse, not better, with scale: influence measured on large
models becomes more abstract, so a document that shares no n-grams at all with
an output can be among the strongest contributors to it, while small models
match surface tokens. Your panel is blind to exactly the contribution that a
frontier-scale royalty system would most need to see.

Both failures point the same way. Panel 1 is a lower bound and a measurement
artifact - useful, exact, cheap, legible, and structurally incapable of
answering the question a compensation scheme asks. Which brings the trap into
focus.

<!-- refutes: V9-M1 -->
You probably think the panel's legibility is a feature. It is a feature and it
is also the demo's largest risk, because a viewer's belief is allocated by
legibility rather than by the label above the panel. Panel 1 highlights actual
words in the actual answer and links to an actual paper. Panel 2 is a table of
scores from a method the viewer cannot check. Panel 3 is a grey pie chart with a
disclaimer under it. Watch where the eyes go.

The failing prediction of "legibility is just presentation" is easy to test and
worth testing on a colleague: show all three panels, then ask them, an hour
later, what the system demonstrated. They will describe Panel 1, and they will
describe it as evidence of what the model used. The label did not survive the
hour. Presentation dominance is not a soft problem you fix with wording; it is
the mechanism by which a demo that is honest in its markup becomes dishonest in
its effect.

The fix is structural rather than typographic. Put the ground-truth
disagreement inside Panel 2, at full size, as the screen above does, so the two
panels visibly disagree and the disagreement is the exhibit. Rank the panels in
the layout by claim strength rather than by conversion rate. And rehearse the
sentence you say aloud while the panel is on screen: *this is what the corpus
contains, and it is on the screen so you can see how different it is from what
moved the model.* A demo where the strongest visual carries the weakest claim
needs the contrast to be the point, or it is a search engine wearing an
attribution costume.

```beat
id: v9-b2
type: self-explain
concept: d-retrieval-attribution
prompt: |
  A competitor ships a product with exactly Panel 1 in it - suffix array,
  maximal verbatim spans, rarity ranking - and calls it "training data
  attribution." They pay publishers pro-rata by number of surfaced spans.

  Which explanation correctly states what their product measures, the one
  experiment that establishes it, and the specific way their payment rule is
  gameable?
options:
  - text: |
      It measures string containment: for a given output string and a given
      indexed corpus, which documents contain which spans, ranked by how few
      documents contain each. The establishing experiment is to hold the weights
      fixed and rebuild or extend the index - the rankings change, the model did
      not. The gaming follows from the ranking: payment rises with rarity, so the
      optimal supplier publishes text that is plausible enough to be emitted and
      rare in the corpus, both of which are cheap. Rarity ranking is the right
      choice for making the panel informative and the wrong one for making it a
      payment basis, because "nobody else wrote this" is the cheapest property to
      manufacture. Metering surfaced spans is a real product; the failure is the
      word "attribution" on the label.
    correct: true
    explain: |
      Exactly the shape of the diagnosis: a corroborative measurement, a
      swap-the-index discriminator, and a payment rule whose incentive runs
      backwards. Unlike a stream farm, manufactured prose needs nobody to consume
      it - it needs only to sit in an index and occasionally match.
  - text: |
      It measures where the answer came from: maximal spans resolving to single
      documents are the causal trace, and what establishes it is that the rare
      spans resolve to one document in one source rather than to the corpus at
      large. The payment rule is sound for the same reason - rarity ranking pays
      the documents that genuinely distinguish themselves from everything else in
      the index.
    misconception: D1
    explain: |
      Resolving to one document shows only that one document contains the string.
      Hold the weights fixed, rebuild the index over other text, and the panel
      changes completely while nothing the model learned changed; the quantity is
      a property of the index. And the property being paid for - rarity - is the
      one an adversary can manufacture at essentially zero cost.
  - text: |
      It measures contribution well enough in practice, because content a source
      really taught the model can be surfaced verbatim. The establishing
      experiment is to remove a source and watch its spans disappear from
      answers. Farming is a limited worry, since manufactured text still has to
      be emitted by the model before anyone is paid for it.
    misconception: D3
    explain: |
      A filter that blocks verbatim emission perfectly is defeated by asking for
      the same content in a different style, and influence at scale becomes more
      abstract - documents sharing no n-grams with an output can be among its
      strongest contributors. The panel is a lower bound. Emission is also a low
      bar: plausible distinctive prose costs nothing to generate in volume.
  - text: |
      It measures containment in an index, and swapping the index is what shows
      that. But paying pro-rata by surfaced spans is a defensible proxy for use -
      share of spans is share of use - and rarity ranking makes farming harder
      rather than easier, because rare, distinctive text is expensive to produce
      at scale.
    misconception: D10
    explain: |
      Under pro-rata nobody's payout is a function of their own usage; it is a
      share of a platform-wide total whose denominator belongs to everyone, which
      is how the reference market leaves 99.8% of suppliers with effectively
      nothing. And rarity is cheap, not expensive: generating large volumes of
      plausible, distinctive prose costs almost nothing and needs no reader.
check: choice
```

## Assembling the pipeline

Now the trace. Eight steps, each one paragraph, each naming the unit that
built it and the artifact it hands forward. Read it as a chain of file
handoffs, because that is literally what it is, and because a chain is
something you can either walk end to end or cannot.

**Step 1, the corpus.** From the pipeline unit: peS2o filtered to the cleanly
licensed subset, partitioned into 12 named sources by venue or subfield, deduped
with the duplicate clusters *recorded* rather than silently collapsed, tokenized
with a domain BPE vocabulary, sharded, and packed with intra-document causal
masking so no training sequence blends two sources. The artifact is the shards
plus the provenance manifest: one row per document carrying `doc_id`,
`source_id`, `shard_id`, offset, length, `cluster_id`, and the attribute set.
Every later step is a query against that manifest. Without `source_id` in the
duplicate cluster members, the payout decision that dedup silently makes becomes
unrecoverable, and there is no later step that can repair it.

**Step 2, the model and its noise floor.** From the training unit: a 10 to 30
million parameter model trained overnight on the Mac, or the 561-million
parameter realistic-scale model on a rented node for about $92 as of mid-2026,
with a WSD schedule so the stable-phase checkpoints branch. Trained at least
twice with different seeds, changing nothing else. The artifact is the
checkpoints and one number: $\sigma$, the standard deviation of held-out bits
per byte across seeds, around 0.003 at this scale. That single number is the
denominator of nearly every claim downstream. A run that produced a model and no
$\sigma$ produced half an artifact.

**Step 3, the counterfactual table.** From the ground-truth unit: train with
each source removed in turn, three seeds per condition, subtract. The artifact
is twelve rows - source, $\Delta_i$, $\text{SE}_\Delta$, $z$ - with a header
block recording $\sigma$, the seed count, the protocol, the held-out set, and
the dedup policy, and with sources at $|z| < 2$ labeled *below the noise floor*
rather than rounded to a small number. Plus, cached in one file, every
$(S, v(S))$ pair any run produced. This table is the only thing in the book that
is true by construction rather than by estimate, and everything cheap gets
graded against it.

**Step 4, the split.** From the fair-split unit: enumerate all
$2^{12} = 4{,}096$ coalitions over the cached utility table, three seeds each
averaged before any Shapley arithmetic touches them, and compute exact Shapley,
Banzhaf, and leave-one-out from the same cache. Then attack it: re-register the
largest source as two shells and then three, watch the share inflate, and fix it
by merging registrants on content fingerprint. Then diagnose it: the additive
fit residual, the per-share standard deviation, and the signal-to-noise ratio.
The artifact is a twelve-row allocation table and four lines beneath it -
explained variance, measured SNR, verdict, and the sweep's cost in GPU-hours and
dollars. The last line is the one the precedent paper omitted.

**Step 5, the evidence wall.** From the memorization unit: the capacity
arithmetic that disposes of "the weights are an archive", an extraction success
on a heavily duplicated document, and a membership-inference failure on an
ordinary one. The artifact is a negative result, and it is load-bearing rather
than embarrassing: it is the argument that keeps you out of the most common
wrong business in this space, and it is what forces the next step to be
prospective rather than retrospective.

**Step 6, the proof.** From the notary unit: generate 25 fictitious-knowledge
watermark documents and 250 never-published controls from one 32-byte key,
commit the hash of the key and the full protocol with a public timestamp
*before* anything else happens, inject the published set into one source at the
density the arithmetic requires, train, then run the committed test unchanged
and correct for the family. Train the negative-control model too - identical
corpus, no watermarks - and run the identical test against it. The artifact is a
signed report, a verifier script a stranger can run from the revealed key, and a
second report recording that the test did not fire on the control model.

**Step 7, the cheap tracer and its trust number.** From the gradient-tracing
unit: TracIn scores per source for a given query, accumulated as gradient dot
products across checkpoints, plus the In-Run ledger accumulated inside the
training run itself. The artifact is a ranking and, inseparable from it, the
Spearman rank correlation of that ranking against the counterfactual table from
step 3. The ranking without the correlation is a demo; the correlation is the
result. This is the step that converts "we have an attribution method" into "we
have an attribution method and here is how much it can be trusted on this
corpus at this scale."

**Step 8, the screen and the memo.** This unit. Panel 1 reads the suffix array
built over step 1's corpus. Panel 2 reads step 7's scores and prints step 7's
correlation next to them. Panel 3 reads the in-run ledger and prints step 4's
SNR verdict beneath it. The footnote reads step 6's report. The memo says what
the four of them jointly establish.

Notice what the chain is made of. Not ideas: files. A manifest, a sigma, a
table of deltas, a cached utility table, a signed report, a score vector, a
correlation. Each one is small, each one is checkable by someone who does not
trust you, and each one is the input to exactly the next thing. That property is
what a pitch means by "we built it," and it is also why the order cannot be
rearranged. You cannot grade a cheap method before you have a truth to grade it
against. You cannot construct a null after the model exists. You cannot merge
shell registrants without a fingerprinting layer that the corpus pipeline
already needed for dedup. The dependencies are real, and each one of them is the
reason a unit sits where it sits.

```beat
id: v9-b3
type: completion
concept: d-demo-assembly
prompt: |
  Fill the blanks. Each row is one step of the chain: the artifact it emits, and
  the single thing that becomes impossible if that artifact is missing.

  | step | artifact emitted | impossible without it |
  | --- | --- | --- |
  | corpus pipeline | shards + ____ | answering "which spans belong to source k" in one pass |
  | training run | checkpoints + ____ | ____ |
  | counterfactual sweep | 12 rows of (Delta, SE, z) + the cached ____ | grading any cheap method against truth |
  | fair split | allocation table + SNR verdict | ____ |
  | notary run | signed report + verifier script | the footnote's stated false-positive rate |
  | gradient tracing | per-source scores + ____ | the header of panel 2 |

  Then: which two artifacts, if produced out of order, cannot be repaired by
  re-running anything? Choose the filling that gets both the blanks and that
  question right.
options:
  - text: |
      Corpus pipeline: the provenance manifest (`doc_id`, `source_id`,
      `shard_id`, offset, length, `cluster_id`). Training run: $\sigma$, the
      seed-noise SD of held-out bits per byte - without it you cannot say whether
      any measured difference downstream is real, since every $z$ in the book
      divides by a standard error built from $\sigma$. Counterfactual sweep: the
      cached $(S, v(S))$ utility table. Fair split: without it you cannot state
      what a source's share is, or whether the split is resolvable at all rather
      than a flat fee. Gradient tracing: the Spearman rank correlation against
      the counterfactual table. Unrepairable out of order: the pre-commitment
      (key and protocol hash timestamped before the model is trained) and the
      source partition with its recorded duplicate clusters.
    correct: true
    explain: |
      Right on both counts. Committing after the model exists is not a weaker
      commitment, it is no commitment; and once duplicates are collapsed without
      recording which sources were in the cluster, the credit decision has been
      made and the information to revisit it is gone.
  - text: |
      Corpus pipeline: the provenance manifest. Training run: $\sigma$ - without
      it you cannot say whether the run trained well, since $\sigma$ is how you
      check that held-out loss landed where it should. Counterfactual sweep: the
      cached $(S, v(S))$ utility table. Fair split: without it you cannot state
      each source's share. Gradient tracing: the Spearman rank correlation.
      Unrepairable out of order: the trained model itself, which is the most
      expensive thing in the chain, and the pre-commitment.
    misconception: D16
    explain: |
      $\sigma$ is not a quality check, it is the denominator of significance:
      contribution is a noisy random variable, and $\text{SE}_\Delta =
      \sigma\sqrt{2/n}$ is what decides whether a source sits above the noise
      floor. And retraining is exactly what this book budgets for - the question
      is which decisions are not re-runnable, and the model is not one of them.
  - text: |
      Corpus pipeline: the provenance manifest. Training run: $\sigma$, the
      seed-noise SD of held-out bits per byte - without it no downstream
      difference can be called real. Counterfactual sweep: the cached
      $(S, v(S))$ utility table. Fair split: without it you cannot state each
      source's share. Gradient tracing: the per-checkpoint gradient dot products
      the scores were accumulated from. Unrepairable out of order: the
      pre-commitment and the corpus-side manifest.
    misconception: D6
    explain: |
      Panel 2's header is the measured correlation against the counterfactual
      table, not the internals of the estimator. A ranking without its
      correlation is a demo; measured against actual retraining, scalable
      gradient methods on a 2B model came out no better than random guessing, so
      the trust number is the deliverable. The fair-split row also needs the
      flat-fee verdict, not just the shares.
  - text: |
      Corpus pipeline: per-shard token counts and the tokenizer vocabulary -
      duplicate-cluster records are pipeline hygiene rather than an artifact.
      Training run: $\sigma$, the seed-noise SD, without which no downstream
      difference can be called real. Counterfactual sweep: the cached
      $(S, v(S))$ utility table. Fair split: without it you cannot state a
      source's share or whether the split beats a flat fee. Gradient tracing: the
      Spearman rank correlation. Unrepairable out of order: the trained model and
      the counterfactual sweep, because they cost the most.
    misconception: D9
    explain: |
      Token counts cannot answer "which spans belong to source k"; `source_id`
      and `cluster_id` can. Dedup is a payout decision, not hygiene - keeping one
      copy chooses which source gets credit for everything that passage teaches -
      and doing it silently is precisely the unrepairable step. Cost is not the
      criterion; re-runnability is.
check: choice
```

## What the demo claims and what it does not

Here is the table that goes in the appendix of the memo, and that you should be
able to reproduce from memory in front of a whiteboard.

| | Panel 1 | Panel 2 | Panel 3 | Footnote |
| --- | --- | --- | --- | --- |
| Question it answers | which documents contain this text | which sources moved the model toward this answer | what each source's share of a fixed pool is | was this source in the training set |
| Vocabulary | corroborative | contributive, with measured trust | allocation | proof |
| May claim | this span occurs in this document, this many times in the corpus | these scores rank sources by influence, and the ranking correlates with counterfactual truth at $\rho = 0.61$ on this corpus at this scale | given this value function and these four requirements, this is the unique split | the committed test fired at $z = 4.5$, family-corrected $p = 8.5 \times 10^{-5}$ |
| May not claim | this document caused this output | this ranking is the counterfactual truth, or holds at other scales | these are payments owed | this generalizes to a false-positive rate at frontier scale |
| Where noise enters | nowhere of its own; deterministic given corpus and output string | twice: the model's training seed, and the seed noise in the table $\rho$ was measured against | propagated through the Shapley sum: $\text{SD}(\phi_i)$, and summarized by the SNR | by design, quantified: the never-published controls define the null |

Three of those rows are the honest part. The fourth row is the one that gets
demos killed, so take each panel's noise seriously and separately.

**Panel 1 has no noise of its own.** Given the corpus and the output string,
the spans are a deterministic function. Change the seed and the output string
changes, so the panel's contents change - but the panel introduced nothing. This
is genuinely a strength and is worth saying out loud, because it is the only
panel about which you can make that statement.

**Panel 2 carries noise twice over.** The scores come from a model that is one
draw from a distribution over training seeds. And the trust number itself,
$\rho = 0.61$, was measured against a counterfactual table whose entries each
carry $\text{SE}_\Delta = \sigma\sqrt{2/n}$. A correlation measured against a
noisy target is attenuated: some of the gap between 0.61 and 1.0 is the method
being wrong, and some is the target being fuzzy, and the honest report says
which sources in the target were below the noise floor and therefore contribute
rank information that is itself noise. Quote $\rho$ with the seed count and the
source count attached, always, or it is a decoration.

**Panel 3's noise is the reason the pie is grey.** The Shapley shares inherit
seed noise through the sum, at roughly $0.745\,\sigma$ per share on a
three-source example - notably *less* noise than the leave-one-out numbers
underneath, because averaging over arrival orders averages over noise too. But
less noise is not no noise, and what decides whether the split is a schedule or
an illustration is the ratio of the spread of shares across sources to the noise
on one share. At 1.37 against a bar of 2, this corpus at this seed count does
not support paying sources differently.

**The footnote's noise is the only kind that was constructed rather than
suffered.** Its false-positive rate comes from a control set you generated under
a key and never published, so it can be sampled as often as you like. That is
the whole difference between prospective and retrospective evidence, and it is
why the footnote is the only line on the screen that uses the word "proven".

<!-- refutes: D6 -->
Now the caveat that a founder is most tempted to bury, stated plainly and then
turned around. You probably think the scale gap is a weakness to be managed:
this is a 561-million-parameter model on a twelve-source corpus, frontier models
are three orders of magnitude larger on corpora four orders larger, and the
demo's numbers therefore do not transfer.

The prediction that a "just scale it up" story makes, and that fails: if
attribution quality were a smooth function of engineering effort, the methods
that scale would show degraded but positive correlation with ground truth at
larger scale. They do not. Measured against actual retraining on a
2-billion-parameter model, the scalable gradient methods came out no better than
random guessing, while the method that does correlate at $\rho = 0.97$ costs
three to five full training runs *per query* and cannot be scaled. Influence
also becomes structurally different with scale, not merely noisier: it
concentrates on a few sequences, becomes more abstract, and moves from surface
token matching toward conceptual matching. The distribution you would be
allocating over changes shape, which is why per-document payouts at frontier
scale round to zero even when the measurement is working.

What is actually true, and this is the version that is *stronger* in a pitch
rather than weaker: the artifact is an existence proof plus a measurement
methodology, and it is honest about being exactly that. The existence proof is
that a tagged corpus, a trained model, a counterfactual ground truth, a
principled split, a keyed proof of membership, and a graded cheap tracer can be
assembled end to end by one person for under $500 - which nobody has published.
The methodology is the part that transfers: measure your own signal-to-noise
against your own counterfactual and report which side of the threshold you are
on. That instruction is scale-free even though its answer is not.

The reason this framing is stronger is a fact about the room. Everyone in the
market claims their method works. Nobody publishes a correlation against
counterfactual truth, because at frontier scale nobody can compute one. You are
the only party in the conversation who can say the sentence "we measured it, and
here is the number," and the sentence is worth more with a modest number in it
than a large claim with no number at all. A demo that says "$\rho = 0.61$ at
this scale, and here is what we did to find out" survives a technical audience.
A demo that says "our attribution is accurate" does not survive one question.

```beat
id: v9-b4
type: compute
concept: d-demo-assembly
prompt: |
  Classify each claim below with exactly one letter:

    C = corroborative      I = contributive      A = allocation      P = proof
    X = none of these; the claim is not licensed by anything in the demo

  1. "This nine-token span occurs once in the corpus, in a source 3 document."
  2. "Source 7 moved this model more than source 11 did, and our ranking tracks
     the retrained truth at rho = 0.61 on this corpus."
  3. "Under held-out log-likelihood as the value function, source 7's share of
     the pool is 19%."
  4. "The committed test fired at z = 4.5 with a family-corrected p of 8.5e-5,
     so this source was in the training set."
  5. "Because source 3's text appears verbatim in the answer, source 3 is owed
     the largest share of this query's revenue."

  Answer as a five-letter string with no spaces, in order.
answer: CIAPX
rubric: |
  Graded exactly. The string is CIAPX.
  Item 5 is the whole point of the drill: it takes a corroborative observation
  and attaches an allocation conclusion to it. Answering A for item 5 = D1, the
  citation-is-attribution error, and it is the single most pitch-fatal answer in
  this unit. Answering C for item 5 = the learner classified the evidence rather
  than the claim; the span observation is corroborative, the sentence built on
  it is not licensed at all.
  Answering I for item 1 = D1 in the other direction.
  Answering P for item 2 = has not registered that a measured correlation is a
  trust number attached to an estimate, not a proof.
check: exact
```

## The economics of the pitch

Everything above is the artifact. This section is the argument, and it starts
with an uncomfortable map: who pays for data today, and for what, as of
mid-2026.

**Labs pay for content, bilaterally, in lump sums.** Roughly 91 publicly
announced deals since January 2023, with industry estimates of 50 to 100 private
deals per public one. The largest disclosed is News Corp with OpenAI at $250M
over five years. Reddit takes about $60M a year from Google. Amazon pays the New
York Times $20 to 25M a year. The structure is overwhelmingly multi-year
bilateral lump sum, and the total private training-rights market is estimated at
only $75 to 100M a year industry-wide.

**Labs pay far more for expert-labeled data.** Meta paid $14.3B for 49% of a
single data-labeling company in June 2025. Another does roughly $2B annualized,
about 90% of it from labs. One labeled-data vendor out-earns the entire
disclosed content-licensing market. When people say "labs spend billions on
data," this is the spending they are describing.

**Gatekeepers meter access.** Cloudflare default-blocks AI crawlers for about
20% of websites, ships payment rails as Merchant of Record, owns the crawler
identity standard and the preference standard, and since January 2026 owns a
training-data licensing marketplace it acquired. That is a chokepoint, an
identity layer, a preference layer, a payment layer, and a marketplace in one
company.

**Answer engines pay for placement.** A $42.5M publisher pool at one answer
engine paying on visits, citations and agent actions; a publisher marketplace at
a major platform paying per-use when its assistant grounds an answer in your
page; an attribution startup splitting ad revenue pro-rata across cited sources.
All of these are real money and all of them are Panel 1's question with a
payment attached.

**Nobody pays for measured training contribution.** Not one deployed system
documents an attribution algorithm. Production allocation rules are flat
pro-rata, discretionary, relevance-proportional but undisclosed, or honest unit
metering. The most-cited supply-side licensing standard has around 1,500
endorsing organizations and zero confirmed AI licensees. The preference standard
at the IETF has no production readers. A supply-side cartel with no counterparty.

<!-- refutes: D12 -->
You probably think training rights are where the money is, because the original
scandal was about training and "training data licensing" is the headline phrase.
The prediction that fails is checkable in the deal record: only about 40% of
recent deals include training rights, down from near-universal in 2023 to 2024,
while attribution and live-access deals - the ones that pay for delivering
answers, not for building models - grew from 2 in 2023 to 18 in 2025 to roughly
34 projected in 2026. The market moved from "buy content to build models" to
"license content to deliver answers," and it moved *away* from the thing a
training-attribution product prices. What is actually true: a
training-attribution product must either target where money already flows -
verification, audit, procurement - or carry the burden of creating a market no
buyer has joined.

<!-- refutes: D13 -->
You probably also think litigation will force the issue, because a $1.5B
settlement sounds like a dam breaking. The prediction that fails is what that
settlement actually priced: acquisition of pirated copies, roughly $3,000 per
work, once, across about 500,000 works, *after* the same court held that
training on lawfully acquired copies is fair use. Three jurisdictions have
converged on that line. Read as an incentive, it tells a rational lab to buy one
clean copy, not to enter a royalty relationship. What is actually true: current
case law pushes toward one-time clean-copy purchases; the theories that could
force ongoing payment, output substitution and market harm, are unresolved and
being litigated now - which is exactly why the space reprices when they resolve,
and exactly why a measurement layer that is needed under every outcome beats a
royalty layer that is needed under one.

### The three open slots

Set against that map, three things the incumbents do not occupy.

**Verification and audit.** Every player in the market measures itself. The
publisher marketplace meters its own grounding events. The answer engine counts
its own citations. The gatekeeper measures its own traffic. There is no
independent verifier anywhere in the chain, and there is now a regulatory
forcing function: the EU AI Act's training-content disclosure obligation carries
enforcement from August 2, 2026, with fines up to EUR 15M or 3% of global
revenue. Better still, the regulator wrote the gap into its own explanatory
notice: supervision proceeds "without performing a work-by-work assessment or
checks whether specific content has been used." Mandatory disclosure, coarse,
self-reported, six-monthly, and explicitly unverified. The verifier's chair has
been drawn on the floor plan and left empty. And it sells to the buy side, which
is the side with money.

**Leakage and syndication tracking.** Blocking your origin is worthless when
your content syndicates to a hundred crawlable copies that never touch your CDN.
This is the loudest unaddressed publisher complaint in the ecosystem, it is
tractable today with near-duplicate detection at web scale, and nobody sells it.

**Buy-side data procurement.** Provenance-verified sourcing, rights clearance,
indemnification. Labs demonstrably spend billions on data acquisition and have
no systematic way to establish that what they bought is what they think they
bought.

Which does the artifact demonstrate? Be precise, because overclaiming here is
how a good demo becomes a bad pitch. The artifact is a working instance of
**verification**: the notary report is exactly the deliverable that slot needs,
and the pre-commitment ledger behind it is the defensible asset. It demonstrates
the **measurement science** any future compensation layer would require, with
its own trust number attached. It does **not** demonstrate leakage tracking,
which needs web-scale near-duplicate infrastructure the PoC never built, and it
does **not** demonstrate procurement, which is a sales and indemnification
business more than a measurement one. The corpus pipeline's fingerprinting layer
is common ground with both, and saying "adjacent, not built" is a stronger
sentence than implying coverage.

<!-- refutes: D10 -->
Now the design everyone reaches for, and the last time this book will take it
apart. You probably think pro-rata pooling - the Spotify model - pays providers
in proportion to how much they were used. It is the incumbent template, "share
of streams" sounds exactly like "share of use," it is in almost every deck in
this category, and the leading supply-side standards body has hired a pricing
economist to build precisely this.

Here is the prediction, and it fails at the first arithmetic step. Under
pro-rata, each provider's payout is their share of a platform-wide total, so
nobody's payout is a function of their own usage - it is a function of a ratio
whose denominator belongs to everyone. Your subscription is distributed by
platform-wide share, which means money flows from your pocket to artists you
never played. In the reference market, 99.8% of artists receive effectively
nothing. And the design invites farming: the AI analogue is cheaper than a
stream farm, because plausible citable text costs nothing to generate and does
not need anyone to consume it.

Watch the misallocation on the book's own numbers rather than on an anecdote.
Three sources with the running value table, in thousandths of a bit per byte of
held-out improvement:

| $S$ | $\varnothing$ | A | B | C | AB | AC | BC | ABC |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| $v(S)$ | 0 | 50 | 50 | 20 | 70 | 90 | 90 | 100 |

Two ways to split a $1 pool. Pro-rata by each source's own measured value, which
is what "share of use" means when you have measurements: the solo values are 50,
50, 20, summing to 120, so A and B each get $50/120 = 41.67$ cents and C gets
$20/120 = 16.67$ cents. Shapley, averaging each source's marginal contribution
over all six arrival orders: 35, 35, 30 cents.

C is the source that pro-rata underpays by nearly half, and the reason is
visible in the table. Alone, C adds 20. Added to A, it takes 50 to 90, a
marginal contribution of 40. Added to B, the same. C is a *complement*: its
value lives in what it unlocks in combination, and a rule that reads only solo
usage cannot see it. This is not an artifact of a toy table. It is the general
shape of the pathology: pro-rata pays for volume of surfaced use, so it
systematically underpays the sources whose contribution is interaction rather
than volume, and systematically overpays whatever is most-surfaced. The music
version of this argument has a name - hidden influencers, the material that
drives utility without resembling outputs - and the same 40-versus-20 gap is why
it exists.

```beat
id: v9-b5
type: compute
concept: d-pitch-economics
prompt: |
  Use the running three-source value table:

  | S    | none | A  | B  | C  | AB | AC | BC | ABC |
  | v(S) | 0    | 50 | 50 | 20 | 70 | 90 | 90 | 100 |

  A pool of exactly $1.00 is split pro-rata by each source's solo value v({i}).

  How many cents does source C receive? Give the answer in cents to two
  decimal places.
answer: 16.67
rubric: |
  Graded numerically. 20 / (50 + 50 + 20) = 20/120 = 0.1667, so 16.67 cents.
  30.00 means they computed the Shapley share instead of the pro-rata share -
  correct arithmetic, wrong rule, and the gap between the two answers (13.33
  cents, a 44% underpayment of C relative to Shapley) is the point of the item.
  20.00 means they reported the raw value rather than the normalized share.
  33.33 means they split evenly, which is the equal-split rule, a third
  allocation and not the one asked for.
check: numeric(0.01)
```

### The scientific spine of the pitch

<!-- refutes: D20 -->
This is the last refutation in the book and it is the one that decides whether
the pitch is credible. You probably think attribution noise is a temporary
methods problem - the estimates are noisy today, the field is young, better
estimators are coming, and the flat fees the industry settled on are a symptom
of immaturity.

Here is the prediction that fails. If noise were only a methods problem, then
for any noise level there would exist a payment contract that beats a flat fee,
given a good enough estimator. There does not. There is a proven threshold:
below a certain signal-to-noise ratio in the attribution estimate, the
welfare-optimal contract *is* a flat fee. Not approximately - the optimum
collapses to it. No estimator improvement crosses that threshold, because the
threshold is about how much signal exists in the measurement, not about how
cleverly the measurement is processed. Only more signal moves you across: more
seeds, larger effects, or a corpus whose sources actually differ.

It is appealing because of builder's optimism and because of the pie chart,
which looks completely definite on a slide while the error bars are not on the
slide. And it is appealing because the field really is young, which makes the
wrong inference locally plausible.

What is actually true, and this is the sentence that goes at the top of the
memo: whether contribution-proportional payment beats a flat fee is an empirical
question about *your* corpus, *your* model scale, and *your* noise floor, and
you measured it.

$$\text{SNR} = \frac{\text{standard deviation of } \phi \text{ across sources}}{\text{SD}(\phi_i)}$$

where $\phi_i$ is source $i$'s allocated share, the numerator is how much the
shares differ from each other, and the denominator is how much one share moves
under seed noise alone. On the running example the shares are 35, 35, 30, with
mean 33.33, so the spread is
$\sqrt{(1.67^2 + 1.67^2 + 3.33^2)/3} = \sqrt{16.67/3} = 2.36$; at three seeds
$\text{SD}(\phi_i) = 1.72$; so $\text{SNR} = 2.36/1.72 = 1.37$, against a
customary bar of 2.

<!-- refutes: V9-M2 -->
You probably think that number kills the pitch. It is a below-threshold result
on the central mechanism, it says the pie chart is not a payment schedule, and
every instinct says to bury it or to keep collecting seeds until it moves.

Here is the prediction that belief makes, and it fails in the room. If a
below-threshold SNR were fatal, then the audience's alternative would be
somebody with an above-threshold measurement. There is no such person. Nobody in
this market has published a correlation against counterfactual truth or a
measured SNR at all, because at frontier scale neither is computable and at
small scale nobody bothered. The comparison is not between your 1.37 and
somebody's 3.0; it is between a measured 1.37 and an unmeasured assertion.

What is actually true: the measurement is the asset, and it is an asset in
either direction. "We measured our attribution signal-to-noise at 1.37, below
the threshold at which contribution-proportional payment beats a flat fee, so
the correct contract for this corpus at this scale is a flat fee - and reaching
an SNR of 2 requires 7 seeds per coalition, which is 28,672 runs and roughly
$717 to $1,912 as of mid-2026" is a complete, priced, defensible result. It
converts a negative into a decision with a number attached. The person across
the table may already know the threshold theorem exists, in which case a pie
chart with no error bars marks you as someone who does not, and the measured
number marks you as someone who does.

Lead with the measured SNR either way. It is the one sentence in the pitch that
nobody else in the market can say.

### The order of the story

So the pitch has an order, and the order is the argument.

**Verification leads.** It is the slot with a regulatory forcing function whose
enforcement is live as of August 2026. It sells to the buy side. Its deliverable
is a document a stranger can check rather than a number they must trust. It
needs no lab to agree to anything and no marketplace to exist. And you have one:
a signed report, a verifier script, and a negative-control report showing the
test did not fire on a model that never saw the watermarks. Sell exclusion
certification first, because the developer wants the certificate and pays again
at every refresh; sell disclosure-conformance checking second; build the
adversarial notary for the moat, knowing that sale is hardest.

**The royalty split is the vision slide.** It is where the market goes if the
unresolved output-substitution theories land, or if a lab decides that
demonstrable provenance is worth paying for. It is the part with the fairness
argument a non-technical counterparty can read and agree to in four sentences,
which no gradient method can supply. And it is the part you have measured
honestly enough to say when it does not yet work. A vision slide with a measured
SNR under it is a roadmap. The same slide without one is the pitch a VC places
in thirty seconds, because they have seen "marketplace plus per-output
attribution plus rev share" many times.

Say the limitation before anyone asks it, in both halves. On the verification
side: watermarks protect only data published after embedding, the models people
most want to sue trained on data collected years ago, and that gap is permanent
- the screening layer that addresses it is triage, not proof. On the royalty
side: this is a small-model existence proof plus a measurement methodology,
influence concentration changes shape with scale, and no lab has agreed to pay
into any marketplace by any mechanism anywhere.

```beat
id: v9-b6
type: compute
concept: d-pitch-economics
prompt: |
  Four described companies. Label each with exactly one letter:

    V = independent verification and audit
    L = leakage and syndication tracking
    P = buy-side data procurement
    X = an already-occupied lane, not one of the three open slots

  1. Sells publishers a service that finds near-duplicate copies of their
     articles across wire services, aggregators and scraped mirrors, and reports
     which of those copies are crawlable.
  2. Sells labs a rights-cleared, provenance-verified corpus with
     indemnification attached.
  3. Runs an answer engine that cites publisher pages and splits ad revenue
     pro-rata across the cited sources.
  4. Sells developers a certificate, backed by a pre-committed statistical test,
     that a named dataset was NOT in their training corpus.

  Answer as a four-letter string with no spaces, in order.
answer: LPXV
rubric: |
  Graded exactly. The string is LPXV.
  Item 3 is the discriminator: it is RAG-time citation revenue sharing, which is
  crowded and funded, and it is corroborative attribution with a payment
  attached. Answering V for item 3 = has confused "attributes something" with
  "verifies someone else's claim", which is the distinction the whole slot
  analysis rests on.
  Item 4 is exclusion certification, which is verification sold to the friendly
  side of the market; answering X for it = has assumed any developer-facing
  compliance product is occupied.
  Answering V for item 1 = plausible-sounding and wrong; leakage tracking is
  about where copies live, not about checking anyone's disclosure.
check: exact
```

## The strongest case against the whole venture

This volume states the strongest argument against each of its own subjects.
Here is the one against all of it, in three parts, none of them hedged, and then
the honest response.

**One: there is no buyer.** No frontier lab has agreed to pay into any
marketplace, by any mechanism, anywhere. The leading supply-side licensing
standard has roughly 1,500 endorsing organizations and zero confirmed AI
licensees. The IETF preference vocabulary has no production readers. The
gatekeeper's per-crawl payment scheme never left private beta before being
replaced. This is not a market with slow adoption; it is a supply-side cartel
that has not yet found a counterparty, and every attribution startup in the
space has been acquired by a rightsholder or a gatekeeper rather than scaled
into an independent exchange.

**Two: the law is removing the forcing function.** The largest settlement in the
space priced acquisition once, at roughly $3,000 per work, after the court held
that training on lawfully acquired copies is transformative fair use. Three
jurisdictions have converged. The rational response for a lab is to buy a clean
copy and train freely. Meanwhile the paid market has been migrating away from
training rights for two years, and the spending that is genuinely enormous -
expert-labeled data - has nothing to do with published content at all.

**Three: the technical claim does not reach frontier scale, and the incumbent
owns the chokepoint.** Scalable attribution methods measured against ground
truth on a 2-billion-parameter model performed no better than random guessing.
The only method that correlates costs multiple training runs per query. And the
gatekeeper already holds the chokepoint, the identity standard, the preference
standard, the payment rail, and a marketplace - publishers already describe
themselves as vendor-locked to it.

Take those three seriously. They are correct. Now the response, which is not a
rebuttal of any of them.

**The measurement layer is needed under every outcome.** Run the branches. If
the courts hold the current line, labs buy clean copies and need to prove
provenance of what they bought - procurement and verification. If output
substitution wins in the pending litigation, ongoing payment arrives and needs a
defensible allocation and an auditor - all three slots at once. If the market
stays where it is, disclosure remains mandatory and unverified, and somebody has
to check the disclosures. There is no branch in which "an independent party can
establish what a model was trained on" is worthless. There are several branches
in which "a royalty exchange" is worthless, which is exactly why the royalty
exchange is the vision slide and not the lead.

**The notary has a forcing function that does not depend on anyone's goodwill.**
Disclosure is mandatory with enforcement live from August 2, 2026 and fines up
to EUR 15M or 3% of global revenue, and the regulator has stated in writing that
it will not check whether specific content was used. That is a mandatory
document, a real penalty, a recurring six-month deadline, and a declared refusal
to answer the question rightsholders care about. Every other slot in this space
waits for a voluntary buyer. This one has a deadline attached to it.

**And the artifact exists either way.** This is the part that is easy to
undersell. Nobody has published an end-to-end tagged-corpus to trained-model to
per-source-payout demonstration with a counterfactual ground truth attached. The
direct precedent for the Shapley-royalty construction published no code and no
cost figures, listed source merging and splitting as an unaddressed limitation,
and skipped the diagnostics. A faithful reimplementation carrying the omitted
numbers, running the skipped diagnostics, and demonstrating then fixing the
listed attack is a contribution that is true regardless of what any lab decides.
It is a paper, a workshop submission, a credential, and the substrate of every
one of the three slots. The failure mode this book is designed to prevent is not
"the venture does not work." It is "the venture does not work and there is
nothing to show for it."

```beat
id: v9-b7
type: self-explain
concept: d-pitch-economics
prompt: |
  You are twenty minutes into a meeting. The person across the table says:

  "I like the engineering. But no lab pays into any marketplace, the courts just
  told them they don't have to, and Cloudflare owns the pipe. What exactly am I
  funding?"

  Which answer states their case back in its strongest form, concedes what is
  true, and then names what survives all three objections - without contradicting
  anything they said, and without appealing to a market development that has not
  happened?
options:
  - text: |
      All three are correct: the leading licensing standard has ~1,500 endorsers
      and zero confirmed AI licensees, the largest settlement priced one-time
      acquisition of pirated copies after the court held training on lawful
      copies is fair use, and the gatekeeper holds the chokepoint, the identity
      and preference standards, the payment rail and a marketplace. So I concede
      the royalty exchange has no counterparty today; it is the vision slide, and
      my own SNR measurement currently says flat fee. What survives is
      measurement, specifically independent verification. Run the branches:
      courts hold the line and labs must establish provenance of the clean copies
      they bought; output substitution wins and ongoing payment needs an
      auditable allocation and an auditor; nothing changes and disclosure stays
      mandatory and explicitly unverified. There is no branch where an
      independent party establishing what a model trained on is worthless - and
      unlike every other slot, this one has a deadline rather than a hope:
      enforcement live from 2 August 2026, fines to 3% of global revenue,
      six-monthly refresh, and a regulator that has said in writing it will not
      check whether specific content was used. The downside is bounded too: the
      artifact is a publishable contribution whatever any lab decides.
    correct: true
    explain: |
      Right. The objections are conceded intact and the answer is the branch
      argument plus a forcing function that depends on nobody's goodwill. That
      combination is what makes verification a considered position rather than a
      retreat.
  - text: |
      Two of the three stand, but the legal one is overstated: a $1.5B settlement
      is the dam breaking, the pending output-substitution and market-harm cases
      are the ones that matter, and when they land labs will be in ongoing
      royalty relationships. What you are funding is the exchange that will
      already exist when that happens.
    misconception: D13
    explain: |
      That settlement priced acquisition of pirated copies at roughly $3,000 per
      work, once, across ~500,000 works, after the same court held that training
      on lawfully acquired copies is fair use, and three jurisdictions have
      converged on that line. Read as an incentive it tells a lab to buy one
      clean copy. Disputing a fact the objector already knows is the answer that
      loses the room; the substitution theories are unresolved, which is why they
      are a reprice, not a plan.
  - text: |
      Concede the marketplace point and answer that the money is in training
      rights: that is what the original scandal was about, content deals are the
      visible spend, and a product that prices measured training contribution
      plugs directly into the licensing that labs are already doing.
    misconception: D12
    explain: |
      Only about 40% of recent deals include training rights, down from
      near-universal, while attribution and live-access deals went 2 in 2023 to
      18 in 2025 to roughly 34 projected in 2026, and expert-labeled data - $14.3B
      for 49% of one labeller - dwarfs all disclosed content licensing. The market
      moved away from the thing this prices, which is why the answer has to point
      at verification, audit and procurement.
  - text: |
      Concede all three without argument, then say what is being funded is the
      royalty engine itself: the demo runs tagged corpus to trained model to
      per-source Shapley payout end to end, so the mechanism demonstrably works,
      and the residual noise in the shares is a methods problem that the next
      generation of estimators closes.
    misconception: D20
    explain: |
      Below a threshold signal-to-noise ratio the welfare-optimal contract *is* a
      flat fee - no estimator improvement crosses it, only more signal does - and
      this corpus measures 1.37 against a bar of 2. Claiming the demo proves the
      royalty mechanism works fails the unit's central discipline, and it also
      skips the branch argument, which is the part that makes the answer
      survivable.
check: choice
```

## What to build next

Four extensions, one paragraph each. Nothing here is taught; this is a map of
where each road goes and what it costs, so that "what's next" in the meeting has
an answer with a shape.

**Influence functions properly.** The second-order family - the formula with the
inverse Hessian in it, the EK-FAC approximation that scaled it to 52 billion
parameters, and the four independent results saying it is unreliable on
language models. It is the deepest mathematics in this space and it is required
by the family with the weakest empirical support, which is the single most
useful fact about it. Study it to read the literature and to answer the question
when it is asked, not to build payments on it. Budget months, and only after the
counterfactual ground truth exists to check it against.

**Source-aware training.** Bind document-identifier tokens into the corpus
during pretraining and instruction-tune the model to emit the supporting
identifiers with its answer. Attribution from the weights rather than from a
post-hoc estimate - architecturally the cleanest route to a built-in
compensation system that anyone has published. It exists at small scale only and
nobody at the frontier has adopted it. It is also the one direction that would
make Panel 2 cheap and Panel 1 unnecessary, which is a strategic reason to
understand it whether or not you build it.

**The audit stack in depth.** Trusted-execution attestation and what it actually
proves - that a cooperating party ran a committed binary on committed bytes,
which is a genuine licensing-integrity feature with a genuine buyer and is
structurally incapable of being an enforcement tool. Alongside it, the
provenance-container standards and their real scope, and the graveyard:
zero-knowledge proof-of-training, four-plus orders of magnitude away at published
prover throughput, and proof-of-learning, spoofed at a fraction of honest
training cost. This is the extension closest to the product that leads the pitch.

**Attribution at real scale.** Continued pretraining on an open 7-billion
parameter model over 500 million to 2 billion tagged tokens, for tens of dollars,
evaluated on papers published after the base model's cutoff so that "the base
model already knew it" is controlled rather than assumed. It answers the one
question the existence proof cannot: does any of this survive contact with a
model somebody would actually deploy. It is the backup claim, never the primary
one, and its result is interesting in either direction.

## At the bench: the demo and the memo

Here is what to build after this unit. It is the last build in the book, and
unlike every other one it produces nothing new - it wires together what already
exists.

**Build the screen.** A single page over the artifacts from the preceding units.
Panel 1 queries the suffix array built over the tagged corpus and renders
maximal verbatim spans ranked by rarity, with the corroborative header sentence
in place and non-optional. Panel 2 renders the per-source influence scores and,
in the same type size, the rank correlation against the counterfactual table
together with the seed count and source count it was measured at, plus an
explicit line naming the places where the ranking and the ground truth disagree.
Panel 3 renders the in-run ledger as a share of a fixed pool, with the measured
SNR and the verdict underneath, and greys the chart when the SNR is under the
bar. The footnote renders the notary report's committed hash, timestamp,
statistic, family size, corrected p-value, and a link to the verifier script.

Four implementation rules, each of which is a claim-discipline decision rather
than a UI decision. The panels are labeled with the vocabulary, not with product
names. No panel may be rendered without its uncertainty field populated; a
missing $\rho$ or a missing SNR blanks the panel rather than defaulting it. The
disagreement between Panel 2 and the ground truth is displayed rather than
logged. And the pool in Panel 3 is notional and says so on the screen.

**Write the memo.** One page, four blocks.

*The headline* is the measured signal-to-noise ratio and its verdict, stated in
one sentence in whichever direction it landed, with the seed count and the cost
to move it. This goes first because it is the sentence nobody else in the market
can say.

*The attack and the fix* is the shell-company demonstration: the largest source
re-registered as two shells and then three, the share inflating, the
fingerprint-merge fix applied, the share returning to baseline. Include the
inflation curve. Every deployed scheme is vulnerable to this and none of them
discuss it.

*The proof* is the notary report in miniature: what was committed, when, what
fired, at what corrected p-value, and the negative-control result showing the
test did not fire on the model that never saw the watermarks. The pair is the
exhibit, not the single detection.

*The map* is the ecosystem slide: who pays today and for what as of mid-2026,
the three open slots, which one the artifact is a working instance of, which two
it is adjacent to and has not built, and the sentence naming the permanent
limitation - watermarks protect only what is published after embedding.

**Artifact.** One page that runs, one page that reads, and a verifier script.
Between them they contain no claim that is not licensed by a file produced in an
earlier unit, and that property - not the visual design and not the model - is
what the build is for.

**What feeds forward.** Nothing, in this book. The extensions in the previous
section each start from one of these files.

```beat
id: v9-b8
type: self-explain
concept: d-demo-assembly
prompt: |
  You are writing the memo's headline block. Your sweep produced a measured SNR
  of 1.37 against a bar of 2, and your gradient tracer correlated with the
  counterfactual table at $\rho = 0.61$.

  Which version states the claim you are entitled to make, together with the
  three inferences your screen invites that you must explicitly disclaim, each
  attached to the panel that invites it?
options:
  - text: |
      Entitled: on this corpus, at this model scale, with this value function and
      this seed count, the attribution SNR is 1.37, below the threshold at which
      contribution-proportional payment beats a flat fee, so the correct contract
      here is a flat fee, and reaching SNR 2 costs a stated number of additional
      seeds at a stated price; separately, the cheap tracer ranks sources at
      $\rho = 0.61$ against the counterfactual table, over 12 sources at 3 seeds.
      Disclaimers - Panel 1: the spans show containment, not causation; the panel
      is a function of the indexed corpus, and swapping the index changes it
      while no weight moves. Panel 2: the ranking is an estimate that agrees with
      retrained truth at 0.61 here, it disagrees with the ground truth at the
      top, and nothing licenses the figure at another scale, where scalable
      methods measured against retraining came out no better than chance. Panel
      3: the shares are the unique allocation implied by four fairness
      requirements given this value function, illustrated not owed, and at SNR
      1.37 the differences between sources are not resolvable against seed noise.
    correct: true
    explain: |
      Right: scoped claim, flat-fee verdict stated rather than hedged, and each
      disclaimer attached to the panel that manufactures the wrong inference.
  - text: |
      Entitled: the per-source split is the headline - source 7 at 19%, source 3
      at 14%, source 11 at 11% - presented as the schedule the method produces,
      with the measured SNR of 1.37 noted underneath as a caveat and more seeds
      already in progress to move it. Disclaimers as usual: Panel 1 shows
      containment, not causation; Panel 2 is an estimate at $\rho = 0.61$ that
      does not transfer to frontier scale; Panel 3's shares are provisional.
    misconception: V9-M2
    explain: |
      A caveat under a pie chart is not the same claim as a flat-fee
      recommendation, and the difference is exactly what an opposing expert or a
      diligence process finds. The comparison in the room is not 1.37 against
      somebody's better number - nobody publishes one - it is a measured 1.37
      against an unmeasured assertion. The measurement is the asset in either
      direction; lead with it.
  - text: |
      Entitled: on this corpus, at this scale, with this value function and seed
      count, the SNR is 1.37, below the bar, so the correct contract here is a
      flat fee. Disclaimers - Panel 2: the ranking is an estimate at $\rho =
      0.61$ and does not transfer to other scales. Panel 3: the shares are an
      illustration of a rule, not payments owed, which is why the chart is grey.
      Panel 1 needs no disclaimer and is the verifiable part of the exhibit: the
      spans are exact, they link to real documents, and they show which sources
      the answer drew on.
    misconception: D1
    explain: |
      That is the inference the panel's legibility manufactures, and it is the
      one the whole unit exists to close. Hold the weights fixed, rebuild the
      index over different text, and the panel changes completely - it was never
      a fact about the model. The team that shipped the deployed version wrote
      the disclaimer into their own paper; the header sentence is non-optional.
  - text: |
      Entitled: on this corpus, at this scale, with this value function and seed
      count, the SNR is 1.37, below the bar, so the correct contract here is a
      flat fee. Disclaimers - Panel 1: containment, not causation. Panel 3: an
      illustration of a rule, not payments owed, unresolvable against seed noise
      at 1.37. Panel 2: $\rho = 0.61$ is a conservative floor obtained on a tiny
      model, and the correlation improves as the model and corpus grow, so the
      figure understates what the method delivers in production.
    misconception: D6
    explain: |
      Measured against actual retraining on a 2B-parameter model, scalable
      gradient methods performed no better than random guessing, and influence
      becomes structurally different with scale - concentrated, more abstract -
      rather than merely noisier. The number is scoped to this corpus at this
      scale; what transfers is the instruction to measure your own correlation,
      not the value.
check: choice
```

## What you can now do

Four things, and the fourth is the one the book was for.

**You can read any attribution claim and say which of four things it is.**
Corroborative: this source supports this text. Contributive: this data moved
this model. Allocation: this is the share a rule assigns. Proof: this test, fixed
in advance against a constructed null, fired at this rate. Almost every product
in this market presents one of the four as another, and the diagnosis takes one
question: what experiment would change this number. If the answer is "rebuild the
index," it is corroborative. If it is "retrain the model," it is contributive. If
it is "change the value function," it is an allocation. If it is "nothing, the
null was built before the model existed," it is a proof.

**You can assemble the chain and name what breaks at each missing link.**
Manifest, sigma, counterfactual table, cached utility table, allocation with its
SNR, signed report with its negative control, influence scores with their
correlation. Each is a file; each is checkable by someone who does not trust you;
each is the input to exactly one next thing. Two decisions in that chain cannot
be repaired by re-running anything - the pre-commitment must predate the model,
and the source partition with its duplicate clusters must predate everything -
and knowing which two is the difference between a build that can be audited and
one that has to be redone.

**You can present a demo whose strongest visual carries its weakest claim
without the demo becoming dishonest.** That takes three structural moves rather
than careful wording: put the ground-truth disagreement inside the panel that
disagrees, at full size; grey the allocation when its SNR is under the bar; and
refuse to render any panel whose uncertainty field is empty. A demo that cannot
display a number without displaying its error bar has made honesty a property of
the code rather than of your discipline in the room.

**And you can make the argument.** The map: labs buy content bilaterally in lump
sums and buy expert data for vastly more, gatekeepers meter access, answer
engines pay for placement, and nobody anywhere pays for measured training
contribution. The slots: verification with a live regulatory deadline, leakage
tracking, buy-side procurement. The lead: verification, because it needs no
buyer's permission and produces a document a stranger can check. The vision:
the split, with a fairness argument a non-technical counterparty can agree to in
four sentences and a measured SNR under it saying whether it works yet. The
concession: no lab pays into any marketplace, the courts are pricing acquisition
rather than use, and the measurement does not reach frontier scale. And the
response: the measurement layer is needed under every branch, the notary has a
deadline rather than a hope, and the artifact is a contribution whatever the
market does.

The book began with a 402 and a closing web. It ends with a screen showing four
different kinds of claim side by side, correctly labeled, with their error bars
visible. Nobody else in this market has shipped that screen. The reason is not
that it is hard to render. It is that each panel required a file that somebody
had to be disciplined enough to produce first, and the discipline is the product.

## Notation in this unit

<!-- canon-only -->

Reference, not reading. Nothing below is new; every symbol arrived in an earlier
unit and is restated here so the screen can be read without leaving the page.

| Symbol | Means | Typical value here |
| --- | --- | --- |
| $\rho$ | Spearman rank correlation of an estimated ranking against the counterfactual table | 0 to 1; 0.61 in this unit |
| $\sigma$ | seed noise: SD of held-out bits per byte across seeds at fixed configuration | 0.002 to 0.008 |
| $\Delta_i$ | leave-one-out contribution of source $i$; positive means helpful | -0.01 to 0.05 |
| $\phi_i$ | source $i$'s allocated share under the Shapley rule | 30 to 35 of 100 |
| $\text{SD}(\phi_i)$ | seed noise on one share | 1.2 to 2.1 |
| SNR | spread of $\phi$ across sources divided by $\text{SD}(\phi_i)$ | 1.37 here; bar is 2 |
| $z$ | observation in null standard deviations from the null mean | 4 to 5 when a watermark design works |
| $p$ | one-sided tail probability corresponding to $z$ | $10^{-5}$ to $10^{-7}$ |
| $m$ | family size for the multiple-testing correction | 25 |
| $\alpha'$ | per-test threshold after correction, $\alpha/m$ | 0.002 at $\alpha = 0.05$, $m = 25$ |
| $n$ | tokens in the indexed corpus, for the suffix array | $10^8$ to $5 \times 10^{12}$ |

The three formulas the screen depends on, restated with every symbol defined
above:

$$\text{SNR} = \frac{\text{SD across sources of } \phi}{\text{SD}(\phi_i)}
\qquad
\alpha' = \frac{\alpha}{m}
\qquad
\text{corrected } p = m \times p$$

The four claim types, which are the actual vocabulary of this unit:

| Type | Statement form | Changed by |
| --- | --- | --- |
| corroborative | this source contains this text | rebuilding the index |
| contributive | this source moved this model | retraining the model |
| allocation | this is source $i$'s share | changing the value function |
| proof | this committed test fired at this rate | nothing after the commitment |

And the two organizing splits of the volume, in one line each. *Contributive
versus corroborative*: caused the capability, versus supports the claim.
*Retrospective versus prospective*: looked at a finished model and tried to infer
the past, versus planted a null before the data was scraped. The first split
decides which panel you are looking at. The second decides whether the word
"proof" is available.

One deliberate simplification, flagged so it does not read as a contradiction.
The screen shows one query's panels, and Panels 1 and 2 are query-conditional
while Panel 3 is not: the in-run ledger is accumulated over the whole training
run and does not depend on what was typed. Rendering all four together is
honest only because the panels are labeled with what they answer. A product that
animated the pie chart in response to each query would be making a per-query
allocation claim that nothing in this book supports, and it is a tempting
interaction to build precisely because it looks like the others.
