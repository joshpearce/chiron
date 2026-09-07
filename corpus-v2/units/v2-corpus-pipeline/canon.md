---
unit: v2
title: "From glob to corpus"
concepts:
  - d-extraction-filtering
  - d-dedup-provenance
  - d-tokenizer-training
  - d-mixing-packing
assumes:
  - d-contributive-corroborative
---

Everything else in this book runs on one artifact: a corpus where every token
can be traced back to a named owner. This unit builds it. Roughly 80% of what
follows is systems engineering you already do - parsing, hashing, sharding,
byte layout - and two places where a decision that looks like plumbing is
actually the whole product. By the end you can go from a directory of raw
files to a tagged, deduplicated, tokenized, document-masked shard set with a
manifest that answers "which source produced training token number
1,447,092?" in constant time.

One piece of vocabulary, used everywhere below. A **source** is the unit you
would write a cheque to: a publisher, a venue, an author group, a feed. Not a
document, not a token. Ten to fifty of them, named up front. Everything in
this pipeline exists to keep the mapping from tokens back to sources intact
while the pipeline mangles the text.


## What the extractor actually hands you

Take one page of one paper. Two columns, an equation, a footer, a table
reference. Here is what the page looks like to a human, in reading order:

> **Sparse Retrieval for Long-Context Summarization**
> Ana Ruiz, Wei Chen. Institute for Text Systems; Northlake University.
>
> **Abstract.** We present a retrieval scheme that selects blocks by a
> scoring function computed once per document, avoiding the quadratic cost of
> re-ranking at query time. On four summarization benchmarks the method
> matches full attention at 12% of the FLOPs.
>
> **3 Model architecture.** [...] where $q_i$ is the query vector for block
> $i$ and $K \in \mathbb{R}^{n \times d}$ is the key matrix. We set
> $\epsilon = 10^{-6}$ to avoid division by zero. Table 2 reports ablations
> over $\epsilon$ and the block size.
>
> **3.2 Training.** All models were fine-tuned for 3 epochs on 8 A100s. The
> learning rate was 2e-5 with linear decay and a significant warmup.

Now run a layout-unaware text extractor over the PDF - the default one, the
one that ships with everything. It walks the page's text objects in the order
the typesetter emitted them, which for a two-column layout is row by row
across the whole page. You get this, verbatim, and this is what lands in your
training data:

```
Sparse Retrieval for Long-Context     where qi is the query vector for
Summarization                          block i and K ? Rn?d is the key
Ana Ruiz1 Wei Chen2                    matrix. We set  = 10?6 to avoid
1Institute for Text Systems            division by zero. Table 2 reports
2Northlake University                  ablations over  and the block size.
Abstract                               3.2 Training
We present a retrieval scheme that     All models were ne-tuned for 3
selects blocks by a scoring func-      epochs on 8 A100s. The learning
tion computed once per document,       rate was 2e-5 with linear decay
avoiding the quadratic cost of re-     and a signicant warmup.
                    Proceedings of the 41st Conference, page 4
```

Read the first content line as a machine would, left to right:
`We present a retrieval scheme that where qi is the query vector for`. That
is a sentence no human ever wrote, and the model will be trained to predict
every token of it.

Four separate injuries are visible in that block, and they are not the same
kind of injury:

1. **Reading order is destroyed.** Column interleaving splices left-column
   line $n$ to right-column line $n$. Sentences are welded together across a
   topic boundary.
2. **Ligatures are dropped.** `fine-tuned` became `ne-tuned`, `significant`
   became `signicant`. The `fi` and `ffi` glyphs had no ASCII fallback in the
   font's encoding table and were silently discarded.
3. **Math is rubble.** `$\epsilon$` vanished entirely, leaving `We set  =
   10?6`. The matrix dimension `R^{n x d}` became `Rn?d`.
4. **Page furniture is inlined.** The running footer is now a sentence in the
   body.

```beat
id: v2-b1
type: predict
concept: d-extraction-filtering
prompt: |
  Before reading on, commit to one reading. Of the four injuries above -
  interleaving, ligature loss, math rubble, inlined footers - which
  description correctly sorts them into what costs the trained MODEL something
  and what costs the ATTRIBUTION something, and names the downstream step that
  breaks?
options:
  - text: |
      All four hurt model quality, interleaving worst. Two of them also move
      credit: interleaving corrupts the source tag at sub-document
      granularity (a spliced line can carry two owners' text under one
      source label, and nothing later can separate them), and inlined footers
      corrupt deduplication (the identical proceedings footer forms one
      enormous near-duplicate cluster spanning hundreds of documents from many
      author groups, so the cluster credit policy is applied to typesetting
      furniture).
    correct: true
    explain: |
      Right, and the two piles overlap rather than partition. Ligature loss
      and math rubble are pure quality damage; interleaving and footers reach
      into the credit machinery, and they break different steps - the source
      tag and the dedup clusters respectively.
  - text: |
      All four are quality damage and nothing more. Extraction is upstream of
      every tagging step, so whatever text comes out is simply tagged with the
      document's source; the attribution machinery is unaffected by how the
      bytes were recovered.
    misconception: V2-M1
    explain: |
      This is extraction-as-commodity in its most common form. Interleaving
      happens BEFORE anything is tagged, which is exactly why it is fatal:
      a line welding a block quotation to body text gets one source label and
      no later stage can split it. Footers, meanwhile, manufacture
      cross-source duplicate clusters out of nothing.
  - text: |
      Only ligature loss and math rubble matter, and they matter for
      attribution: broken spellings and missing symbols change a document's
      shingles, so it no longer matches its own duplicates. Interleaving and
      footers are cosmetic - quality filters remove both.
    misconception: V2-M1
    explain: |
      Shingle drift is a defensible secondary worry, but the ordering is
      backwards. Interleaving is the worst injury on both axes: it fabricates
      sentences no human wrote at tens per page, and it merges two owners'
      text into one tagged span. And no quality filter reconstructs reading
      order - the extractor reported no error.
  - text: |
      None of the four survives filtering, so the pile assignment is moot: the
      quality classifier scores the mangled text low and drops those documents
      before tokenization.
    misconception: D8
    explain: |
      Interleaved text is fluent line by line and passes heuristics; that is
      the whole danger. A filter selects for a distribution, it does not
      detect that the reading order was destroyed. Extraction damage is
      silent all the way to the trained model.
check: choice
```

Now the same page through a pipeline that knows what a page is. Layout is
detected first, columns are segmented, reading order is recovered from the
geometry, the footer is classified as furniture and dropped, and math is
emitted as markup rather than as glyph soup:

```
TITLE     Sparse Retrieval for Long-Context Summarization
AUTHORS   Ana Ruiz (Institute for Text Systems)
          Wei Chen (Northlake University)

SECTION   Abstract
We present a retrieval scheme that selects blocks by a scoring function
computed once per document, avoiding the quadratic cost of re-ranking at
query time. [...]

SECTION   3 Model architecture
[...] where $q_i$ is the query vector for block $i$ and
$K \in \mathbb{R}^{n \times d}$ is the key matrix. We set
$\epsilon = 10^{-6}$ to avoid division by zero. Table 2 reports ablations
over $\epsilon$ and the block size.

SECTION   3.2 Training
All models were fine-tuned for 3 epochs on 8 A100s. The learning rate was
2e-5 with linear decay and a significant warmup.

DROPPED   page-footer: "Proceedings of the 41st Conference, page 4"
```

Same page. Different training data. The gap between those two blocks is the
subject of the next section, and it is measurable: swapping extraction
quality alone, holding every other pipeline stage and the training recipe
fixed, moved a 13-benchmark average by 1.08 points at 62 billion training
tokens (AICC, arXiv:2511.16397). That is a larger swing than most
architecture changes anyone will propose to you.

## Extraction: three source types, three tools

<!-- refutes: V2-M1 -->

You probably think extraction is a solved commodity - `pip install` a library,
call it, move on to the interesting part. Here is the specific prediction that
belief makes, and it fails: if extraction were commoditized, the choice of
extractor would be a wash, and two teams building the same corpus with
different libraries would land on equivalent data. Instead the two most-used
HTML extractors differ by five points of content recall and a factor of three
in retained boilerplate, and the choice propagates all the way to benchmark
scores.

The belief is appealing because extraction really is a solved problem for the
input you usually meet: one-column, well-formed HTML written this decade. It
stops being solved the moment the input is a scanned two-column PDF with
equations, which is exactly the input a paper corpus is made of.

Here is what is actually true, split by source type.

**HTML.** Two extractors dominate. `trafilatura` (used to build FineWeb)
optimizes precision: 69.5% content recall, 6.6% boilerplate retained.
`resiliparse` (used to build DCLM) optimizes recall and speed: 74.5% recall,
22.8% boilerplate. Neither is correct; they are different points on the same
tradeoff, and the right pick depends on whether a downstream quality
classifier will clean up the boilerplate for you. As of mid-2026 the best
published speed/quality point is `rs-trafilatura` at F1 0.859 and 44
milliseconds per page (WCXB, arXiv:2605.21097), which makes single-machine
extraction of a large web crawl a weekend rather than a cluster job.

**PDF.** This is where the arc is. The lineage runs GROBID (a CRF-based
structure parser, the tool behind S2ORC and peS2o) to Nougat (the first
end-to-end vision transformer for academic PDFs) to `marker` (fast,
pipeline-style) to olmOCR (arXiv:2502.18443) and olmOCR 2 (arXiv:2510.19817).
olmOCR 2 is a 7B vision-language model trained with reinforcement learning
against unit-test rewards - the reward checks properties of the output, such
as whether the reading order is monotone and whether a table's columns parse,
rather than string-matching a reference transcription. Human raters prefer it
to `marker` 61.3% of the time. As of mid-2026 that is the ceiling for OCR on
papers, and it costs a GPU.

**LaTeX source, where you can get it.** This is the move that matters for a
paper corpus, and it is not an ML move at all. arXiv publishes the submitted
source in an S3 bucket (`src/`, roughly 9.2 TB, requester-pays). For any paper
that submitted TeX rather than a camera-ready PDF, you can skip OCR entirely
and read the document's actual structure: section hierarchy, equations as
LaTeX, citation keys as `\cite{}` commands, figure captions attached to
figures. No reading-order inference, no ligature loss, no math rubble. The
extraction problem disappears rather than being solved.

Two caveats that bite immediately, both about provenance rather than quality:

- The **default arXiv license does not permit redistribution.** The bulk
  bucket contains everything; you may only build a redistributable corpus
  from the subset whose per-paper license is Creative Commons or public
  domain. This is a metadata filter you apply before extraction, not after,
  because a document you cannot redistribute is a document you cannot put in
  a manifest anyone can audit.
- LaTeX source is available for a **biased subset**. Older papers, papers
  from communities that submit PDFs, and anything not on arXiv are excluded.
  Mixing LaTeX-derived and OCR-derived text means your corpus has two
  extraction qualities correlated with source identity - and in v4, when you
  measure per-source contribution by retraining, you will not be able to tell
  whether source A beat source B because its content is better or because
  its extraction is cleaner. Record the extraction method per document as an
  attribute. It costs one byte and it is the control variable for every
  measurement later in this book.

## Quality filtering: the classifier is the lever

<!-- refutes: D8 -->

**You probably think model quality is about corpus size.** More tokens, better
model; the headline numbers are all sizes, 15T beats 1.3T, and the whole
scaling-law genre is about how much data you can get.

**Here is the specific prediction that fails.** Under "more is better," a
corpus and any subset of it are ordered by size: the 15T-token corpus must
beat the 1.3T-token corpus carved out of it, since the subset contains
strictly less information. FineWeb-Edu is exactly that carve-out - a 1.3T
filtered subset of FineWeb's 15T - and it beats its own superset on knowledge
benchmarks, for small models decisively. A subset of your data outperforming
all of your data is not a rounding error; it means the discarded 92% was
carrying negative value.

**Here is why the wrong model is appealing.** Scaling laws are real and the
size axis is the one everyone publishes. There is also a floor effect that
makes the intuition locally correct: below some corpus size you genuinely are
data-starved and every token helps.

**Here is what is actually true.** Past that floor, quality dominates, and
quality is operationalized as a classifier. Three generations, in the order
you should build them:

*Heuristics.* The C4 and Gopher rule sets: drop documents without terminal
punctuation, with too few or too many words per line, with a high symbol-to-
word ratio, with excessive repetition of any n-gram. Cheap, and they remove
the obvious garbage. This is also where language identification lives -
fastText langID, inherited from CCNet (arXiv:1911.00359), which is the
ancestor of every pipeline in this section: shard by hash, langID, paragraph
dedup, perplexity filter.

*Perplexity.* Train a small n-gram language model (KenLM) on a reference
corpus you consider good - Wikipedia, traditionally - and score every document
by how surprising it is under that model. Keep the low-perplexity tail. This
works, and it has a known pathology: it selects for text that resembles the
reference corpus in style, which is not the same as text that is useful.

*Classifiers.* The generation that actually moved the numbers, and two designs
worth knowing exactly:

- **DCLM** trains a fastText classifier with positives drawn from
  instruction-style and explanatory text (OpenHermes, ELI5) against random web
  text as negatives, then keeps the top 10% by classifier score. In their own
  benchmark ablations this was the single biggest factor - larger than any
  architecture change they tested - and filtering alone was worth +6.6 MMLU.
- **FineWeb-Edu** takes the LLM-as-annotator route. Llama-3-70B scores 460,000
  sampled documents 0 to 5 for "educational value." A small regression head on
  frozen embeddings is fit to those labels, reaching 82% F1, and then scores
  the full corpus at web scale. Keep everything scoring 3 or above. Total
  annotation cost: about 6,000 H100-hours. Result: MMLU 33 to 37, ARC 46 to
  57.

Sit with the FineWeb-Edu shape for a moment, because it is the argument
against your own instinct. The expensive part is 6,000 GPU-hours of a large
model writing training labels for a tiny model. The tiny model is a
logistic-regression-shaped thing on top of embeddings. There is no clever
math anywhere. The highest-leverage knob in the entire pipeline is a data
labeling job and a classifier you could write in an afternoon.

Nemotron-CC (arXiv:2412.02595) pushes the same lever harder with a classifier
ensemble plus synthetic rephrasing of borderline documents, reporting +5.6
MMLU over DCLM while retaining 4x more unique tokens - which is the tell that
aggressive filtering costs you token count, and that recovering some of that
count is itself worth points.

```beat
id: v2-b2
type: self-explain
concept: d-extraction-filtering
prompt: |
  A colleague building the paper corpus proposes: "peS2o is already
  quality-filtered by Ai2 and every document is a peer-reviewed paper. Quality
  filtering is a web-crawl problem. Skip the classifier, spend the time on
  dedup."

  The premise is largely true. Which explanation best accounts for why the
  conclusion is still wrong for THIS corpus, and names what a classifier would
  be filtering FOR here that it would not be filtering for on a web crawl?
options:
  - text: |
      A filter selects for the distribution you want the model good at; it
      does not merely remove garbage. peS2o has no SEO spam, but it is full of
      reference lists, table rubble, acknowledgements, affiliation blocks and
      half-surviving math - grammatical enough to pass every heuristic, and a
      large share of a paper's tokens. At 300M to 1B tokens there is no
      general-web ballast to dilute that. The thing being selected for is
      connected exposition as against the structured metadata around it.
    correct: true
    explain: |
      Right. The premise (no web garbage) is true and the conclusion does not
      follow, because the classifier's job here is a distribution choice, and
      the artifact class it targets - document furniture inside a legitimate
      paper - is one a web classifier never meets.
  - text: |
      The colleague is right for this corpus. Filtering is a size knob: it
      buys quality by discarding tokens, and peS2o at 300M to 1B tokens is
      already near the data floor, so any classifier can only make a
      quality-filtered corpus smaller and therefore worse.
    misconception: D8
    explain: |
      This still treats filtering as a size dial. FineWeb-Edu is a 1.3T
      carve-out that beats its own 15T superset on knowledge benchmarks, which
      means the discarded material carried negative value. Fewer tokens of the
      right distribution is the whole finding.
  - text: |
      The conclusion is wrong because peS2o is not large enough: the right
      move is to relax filtering and pull in more papers, since past a floor
      corpus size is what sets model quality and a classifier only shrinks the
      pool further.
    misconception: D8
    explain: |
      Backwards. Past the floor, quality dominates size, and quality is
      operationalized as a classifier - the single biggest factor in DCLM's
      own ablations, larger than any architecture change they tested.
  - text: |
      The conclusion is wrong only because peer review is not a reliable
      quality signal - venues differ enormously, so the classifier's job is to
      re-rank documents by venue prestige and drop the weak papers.
    misconception: V2-M1
    explain: |
      Venue quality is a different axis, and sorting by it does not touch the
      concrete miss: extraction residue INSIDE good papers. A strong paper
      whose reference list is 15% of its tokens still spends 15% of the
      model's capacity teaching it to emit citation strings.
check: choice
```

## Deduplication, worked by hand

<!-- fade: v2-minhash-by-hand -->

Two documents are near-duplicates when they share most of their content but
not all of it - a preprint and its camera-ready, a paper and the version
mirrored on an institutional repository with a different header. Comparing
every pair is quadratic and therefore off the table, so the standard machinery
estimates pairwise similarity with a fixed-size sketch and only compares pairs
that a hash-based prefilter puts in the same bucket.

The similarity being estimated is the **Jaccard index**. Represent each
document as a set of shingles - overlapping token n-grams, so that a document
becomes a set of short phrases rather than a bag of words. FineWeb uses
5-grams. For two shingle sets $A$ and $B$:

$$J(A, B) = \frac{|A \cap B|}{|A \cup B|}$$

where $|A \cap B|$ is the number of shingles both documents contain and
$|A \cup B|$ is the number contained by either. $J$ runs from 0 (nothing
shared) to 1 (identical shingle sets).

Work one pair by hand. Hash each document's shingles to small integers - two
short documents, five shingles each:

$$A = \{2,\ 5,\ 9,\ 14,\ 20\}, \qquad B = \{2,\ 5,\ 9,\ 14,\ 31\}$$

The intersection is $\{2, 5, 9, 14\}$, four elements. The union is
$\{2, 5, 9, 14, 20, 31\}$, six elements. So $J(A,B) = 4/6 = 0.667$.

That was cheap because the sets have five elements. A real document has tens
of thousands of shingles and there are billions of documents, so you never
store the sets and never compute that intersection. Instead:

**One hash function, one set.** Pick a hash function $h$ and record only the
minimum value it takes over the set: $\min_{x \in A} h(x)$. The claim, which
is the whole trick, is that

$$\Pr\left[\min_{x \in A} h(x) = \min_{x \in B} h(x)\right] = J(A, B)$$

Why: the minimum over $A \cup B$ is achieved by one particular element, and
$h$ being a random-ish permutation makes each element of $A \cup B$ equally
likely to be that one. The two minima agree exactly when that element happens
to lie in $A \cap B$. The probability of that is $|A \cap B| / |A \cup B|$,
which is $J$. One hash function is therefore an unbiased estimator of the
Jaccard index that returns one bit.

**Many hash functions, one signature.** Use $k$ independent hash functions and
keep all $k$ minima. That vector is the document's **MinHash signature**, of
fixed size $k$ regardless of document length. Estimate $J$ by the fraction of
signature positions where two documents agree.

Do it with $k = 3$ on the sets above. Take $h_i(x) = (a_i x + b_i) \bmod 37$
with $(a, b)$ of $(3,1)$, $(5,2)$, $(7,3)$:

| $x$ | 2 | 5 | 9 | 14 | 20 | 31 |
|---|---|---|---|---|---|---|
| $h_1(x) = (3x+1) \bmod 37$ | 7 | 16 | 28 | 6 | 24 | 20 |
| $h_2(x) = (5x+2) \bmod 37$ | 12 | 27 | 10 | 35 | 28 | 9 |
| $h_3(x) = (7x+3) \bmod 37$ | 17 | 1 | 29 | 27 | 32 | 35 |

$A = \{2,5,9,14,20\}$ uses the first five columns; $B = \{2,5,9,14,31\}$ uses
columns one to four and the last.

- $h_1$: over $A$, $\min(7,16,28,6,24) = 6$. Over $B$, $\min(7,16,28,6,20) = 6$. Agree.
- $h_2$: over $A$, $\min(12,27,10,35,28) = 10$. Over $B$, $\min(12,27,10,35,9) = 9$. Disagree.
- $h_3$: over $A$, $\min(17,1,29,27,32) = 1$. Over $B$, $\min(17,1,29,27,35) = 1$. Agree.

$$\text{sig}(A) = [6,\ 10,\ 1], \qquad \text{sig}(B) = [6,\ 9,\ 1]$$

Two of three positions agree, so the estimate is $2/3 = 0.667$, against a true
$J$ of $0.667$. That agreement is luck at $k=3$; the estimator's standard
error is roughly $1/\sqrt{k}$, which is why production configurations use
$k$ in the hundreds.

```beat
id: v2-b3
type: completion
concept: d-dedup-provenance
prompt: |
  A third document has shingle set $C = \{2,\ 5,\ 9,\ 20,\ 31\}$. Using the
  same three hash functions and the same table of values as above:

    h_1 over C: min(7, 16, 28, ____, 20)   = ____
    h_2 over C: min(12, 27, 10, 28, 9)     = ____
    h_3 over C: min(17, 1, 29, ____, 35)   = ____

  Recall sig(A) = [6, 10, 1]. Which filling of the five blanks, and which
  reading of the result, is correct?
options:
  - text: |
      Blanks 24, 7, 9, 32, 1, so sig(C) = [7, 9, 1]. Only the third position
      agrees with sig(A) = [6, 10, 1], giving an estimate of 1/3 = 0.33, while
      the true index is J(A,C) = |{2,5,9,20}| / |{2,5,9,14,20,31}| = 4/6 =
      0.667. The estimator is unbiased but has standard error about
      $1/\sqrt{k}$, and at k = 3 it cannot resolve this pair from A and B.
    correct: true
    explain: |
      Right on both halves. $h_1(20) = (3 \cdot 20 + 1) \bmod 37 = 24$ and
      $h_3(20) = (7 \cdot 20 + 3) \bmod 37 = 32$, and the true J here equals
      J(A,B) = 0.667 exactly - the signature says 0.33 for one and 0.667 for
      the other. That gap is why production configurations use hundreds of
      hashes.
  - text: |
      Blanks 24, 7, 9, 32, 1, so sig(C) = [7, 9, 1] and the estimate is 1/3 =
      0.33. The true index is also about 0.33, since C shares three of its
      five shingles with A and MinHash is exact for small sets.
    misconception: D3
    explain: |
      The arithmetic on the signature is right, the set arithmetic is not.
      Intersection $\{2,5,9,20\}$ is four elements, union
      $\{2,5,9,14,20,31\}$ is six, so J = 4/6 = 0.667. MinHash is never exact;
      it is an unbiased estimate whose error is what this beat is about.
  - text: |
      Blanks 24, 7, 9, 32, 1, so sig(C) = [7, 9, 1], estimate 0.33 against a
      true J of 0.667. The signature is therefore biased downward on
      small sets, and the fix is to correct the estimate upward before
      thresholding.
    misconception: D19
    explain: |
      The numbers are right and the diagnosis is wrong. Each position agrees
      with probability exactly J, so the estimator is unbiased; what fails at
      k = 3 is variance, not centring. Raising k shrinks the error as
      $1/\sqrt{k}$; no upward correction is warranted.
  - text: |
      Blanks 20, 20, 9, 35, 9, so sig(C) = [20, 9, 9]. No position agrees with
      sig(A), giving an estimate of 0 against a true J of 4/6 = 0.667.
    misconception: D3
    explain: |
      This reads the blanks as h-values of 31 rather than of 20. The set C
      contains 20, so the missing entries are $h_1(20) = 24$ and
      $h_3(20) = 32$; the 31 column supplies the 20 and 35 already printed in
      the prompt. The minima are 7 and 1.
check: choice
```

**Bucketing: LSH.** Signatures still have to be compared pairwise unless you
bucket. Split each $k$-position signature into $b$ bands of $r$ rows, with
$k = b \times r$. Hash each band to a bucket and call two documents candidates
if they collide in at least one band. If a single position agrees with
probability $J$, a whole band of $r$ positions agrees with probability $J^r$,
so

$$\Pr[\text{candidate}] = 1 - (1 - J^{r})^{b}$$

FineWeb's configuration, shipped in `datatrove`: 5-gram shingles, $k = 112$
hashes, $b = 14$ bands of $r = 8$ rows, targeting a similarity threshold near
0.75. Put numbers through the formula: a pair at $J = 0.9$ becomes a candidate
with probability 0.9996, a pair at $J = 0.75$ with probability 0.77, and a
pair at $J = 0.5$ with probability 0.053. That is the S-curve you are buying,
and the knee sits at roughly $(1/b)^{1/r} = (1/14)^{1/8} \approx 0.72$.

```beat
id: v2-b4
type: compute
concept: d-dedup-provenance
prompt: |
  A toy LSH configuration uses $k = 6$ hashes split into $b = 3$ bands of
  $r = 2$ rows. Two documents have true Jaccard index $J = 0.5$.

  Using $\Pr[\text{candidate}] = 1 - (1 - J^{r})^{b}$, compute the probability
  that this pair is emitted as a candidate.

  Give a single number to 2 decimal places.
answer: 0.58
check: numeric(0.01)
```

That 0.58 is the lesson: a pair sharing only half its shingles gets through
this configuration more often than not. Small $k$ means a permissive filter
and a flood of candidates to verify. Widening to 14 bands of 8 drops the same
pair to 0.053.

**Exact substring dedup** is the other half, and it is a different data
structure for a different problem. Build a suffix array over the concatenated
corpus and find every substring of at least 50 tokens that occurs more than
once (Lee et al., arXiv:2107.06499). MinHash finds documents that are mostly
the same; suffix arrays find passages that are exactly the same inside
documents that are otherwise different - the block quotation, the reused
methods paragraph, the standard licence boilerplate. Removing them cut
memorized-text emission by roughly 10x in the original study, which is a fact
you will need again in v6 when memorization becomes evidence rather than a
defect. `text-dedup` packages both; Ai2's BFF does Bloom-filter dedup when you
want exact-duplicate removal at streaming speed and can tolerate a false
positive rate.

<!-- refutes: V2-M5 -->
**You probably think global dedup is strictly better than local dedup** -
same algorithm, larger candidate pool, strictly more duplicates found,
strictly cleaner corpus. Here is the prediction that fails: FineWeb ran both,
deduplicating each Common Crawl snapshot independently versus deduplicating
across all snapshots at once, and the per-snapshot version produced the
*better* model. The global version is appealing because it is obviously more
thorough. What it is more thorough at is deleting the material that got
re-crawled every month, and material that many sites keep linking to and
re-hosting for years is disproportionately the good material. Global dedup
keeps one copy of the popular thing and one copy of every piece of junk, which
raises junk's share of the corpus. Aggressiveness is a tuning parameter with
an interior optimum, not a direction to push.

## Dedup is a payout decision

<!-- refutes: D9 -->

**You probably think deduplication is neutral cleanup.** Find the copies, keep
one, drop the rest. Nothing of substance has been decided; nobody has ever
audited which copy survived a dedup pass, in any pipeline you have built, and
nothing bad happened.

**Here is the specific prediction that fails.** Under "dedup is hygiene," the
choice of survivor is arbitrary and consequence-free, so two runs of the
pipeline that keep different survivors produce equivalent corpora. Now attach
a payment to the corpus. Source A and source B both contain the same 4,000-word
methods section. The pipeline keeps A's copy because A's document was
encountered first in shard order. Whatever the model learns from that passage,
and every attribution measurement made later - the leave-one-source-out delta
in v4, the Shapley value in v5, the influence score in v8 - now credits it
entirely to A, because in the corpus that trained the model, B does not
contain it. Delete A instead and the money moves to B. The two runs are not
equivalent; they differ by a payment.

At web scale this decision is made silently millions of times, and the
deciding factor is shard iteration order.

**Here is why the wrong model is appealing.** In every other pipeline you have
ever written, dedup genuinely is hygiene. Duplicate rows in a warehouse table
have no owner. There is no third party who cares which of two identical log
lines you dropped. The intuition is correct everywhere except here, and what
changed is not the algorithm but the existence of a claimant.

**Here is what is actually true.** Dedup is a credit-assignment step wearing
a maintenance step's clothes, and the fix is a schema change, not an algorithm
change. Do not resolve the cluster. Record it.

Concretely, instead of emitting one surviving document and forgetting, emit:

```
cluster_id      : integer, assigned by union-find over the LSH candidate graph
representative  : the doc_id whose tokens actually enter the corpus
members         : [ {doc_id, source_id, jaccard_to_representative}, ... ]
policy          : the credit rule applied to this cluster
```

The `members` list is the entire novel contribution, and it costs you a table.
The representative still goes into exactly one place in exactly one shard -
you are not training on the duplicates, and the model is unchanged. What
changes is that at payout time the cluster is a known object with a known
source membership, so the credit rule is an explicit, auditable policy rather
than an accident of iteration order.

Three policies are defensible and you should pick one before you look at the
data, because picking after is how you get accused of choosing the one that
paid your favourite source:

- **Winner-take-all by earliest publication date.** The passage originated
  somewhere; credit the origin. Requires trustworthy dates and handles the
  preprint/camera-ready case correctly.
- **Equal split across cluster members.** Every source in the cluster gets
  $1/m$ of whatever the passage earns. Simple, obviously fair-looking, and
  directly gameable: a source that mirrors other people's content joins many
  clusters and collects a share of each. v5 makes this concrete as the
  shell-company attack.
- **No credit for shared content.** Only the passages unique to a source earn.
  Defensible, aggressive, and it makes the split measure distinctiveness
  rather than volume.

There is no correct answer here, which is the point. The failure mode is not
picking the wrong policy. The failure mode is having a policy you did not know
you had, imposed by shard order.

```beat
id: v2-b5
type: self-explain
concept: d-dedup-provenance
prompt: |
  A duplicate cluster contains four documents:

    doc_101, source: Journal of Text Systems, published 2021-03
    doc_417, source: Northlake Preprints,     published 2020-11
    doc_890, source: Northlake Preprints,     published 2020-11
    doc_902, source: OpenMirror Archive,      published 2023-08

  doc_417 and doc_890 are the v1 and v2 preprints of the same work. doc_101 is
  the camera-ready. doc_902 is a scraped mirror of the camera-ready.

  Which account of what the manifest must record - and of why "equal split
  across cluster members" is the wrong default here specifically - is correct?
options:
  - text: |
      Record the cluster id, the full member list with each member's doc_id
      AND source_id, which member is the representative, the representative's
      (shard, offset, length) span, each member's publication date, and each
      member's similarity to the representative - so any of the three policies
      is computable later without retraining. Equal split is wrong as a
      default because it is computed over MEMBERS and Northlake appears twice:
      a source posting n versions collects n/m for a passage it contributed
      once. The split must be over distinct sources.
    correct: true
    explain: |
      Right, and the double-count is the load-bearing part. Dates are needed
      because earliest-publication is a function of them; similarities because
      an exact copy and a 0.76 near-match deserve different arguments. Noticing
      that source identity is the adversary's knob is the v5 shell-company
      attack arriving early.
  - text: |
      Pick the policy first - earliest publication, since doc_417 predates the
      camera-ready by four months - and record only what it needs: the
      representative and the winning member's date. Recording the rest is
      speculative schema for policies you have decided against.
    misconception: D9
    explain: |
      This welds the policy into the corpus. The design requirement is that
      the payout rule be changeable without retraining, which is only true if
      the model is a function of the representative alone and payout a
      function of a cluster table that survives every policy. Discard the
      member list and changing your mind means rerunning dedup.
  - text: |
      Record only the representative and its span. The cluster is resolved by
      construction - one document's tokens enter the corpus - so the members
      are no longer part of the corpus and have nothing to be paid for. Equal
      split is wrong here simply because a scraped mirror should never be paid.
    misconception: D9
    explain: |
      Dedup as hygiene. Whatever the model learns from that passage is
      credited entirely to whoever holds the representative, and every later
      measurement - leave-one-source-out, Shapley, influence - inherits that
      accident of shard order. The member list is the entire novel
      contribution and it costs one table.
  - text: |
      Sidestep the policy question by keeping all four documents in the corpus
      and tagging each with its own source, so credit follows the tokens
      directly and no cluster table is needed.
    misconception: D9
    explain: |
      Training on four copies quadruples the passage's effective weight and
      makes it extractable (v6), so it is a worse corpus and a worse
      attribution substrate at once. You have not avoided the credit decision,
      you have made it by multiplicity.
check: choice
```

Two structural notes for the build, both pure systems work.

Clusters come from **union-find over the LSH candidate graph**: each candidate
pair is a union, and the resulting components are the clusters. Transitivity
is a real consequence here, not a nuisance - a chain of 0.76-similar documents
can produce a component whose extreme members share almost nothing. Cap
component diameter, or record the pairwise similarities and let the payout
policy decide, but do not let a million-document component form silently.

The **cluster table is the join key for everything downstream.** Design it
alongside the manifest in the last section of this unit, not after. Adding
`source_id` to the members list after you have run the pipeline means running
the pipeline again.

## Decontamination and PII

Two short passes, one of which is a trap.

**Decontamination** removes training documents that overlap your evaluation
sets, because a model that has seen the test is not measuring what you think.
The standard technique is n-gram overlap: build the set of n-grams appearing
in every eval set, and drop or flag any training document sharing one. The
GPT-3-era convention is 8 to 13 grams, and Dolma implements the modern version
with a Bloom filter over the Paloma eval suite - a Bloom filter because you
want a membership test over billions of n-grams in bounded memory, and a false
positive here costs you one training document, which is free.

Two disciplines that are not optional in a corpus you will use for
attribution. First, decontaminate **before** dedup, or at least record it as a
document attribute rather than a deletion, because a decontamination drop is
another silent removal of a source's content. Second, the held-out papers you
will use for domain evaluation in v3 must be excluded by **document identity**,
not by n-gram overlap, and excluded before anything else runs. n-gram overlap
is a defence against accidental leakage; whole-document exclusion is a
defence against the leakage you designed.

<!-- refutes: V2-M2 -->
**You probably think PII scrubbing is a regex pass.** Emails, phone numbers,
IP addresses, credit card patterns - a few hundred lines, run it over the
corpus, done. And that is genuinely the standard first pass: Dolma and the
common pipelines do regex email and IP anonymization, replacing matches with
typed placeholders.

**Here is the prediction that fails.** If PII were a lexical pattern, then a
document with every regex-matchable string removed would contain no personal
information. Take a paper's acknowledgements section with the emails stripped:
"We thank the third-year PhD student in the Northlake speech group who ran the
2019 pilot." No pattern matches. The person is identified. The same holds for
the author affiliation blocks, the funding acknowledgements naming grant
numbers tied to individuals, and the clinical case descriptions in medical
papers where the combination of age, condition, institution, and date is
identifying even though no field is.

**Why the wrong model is appealing.** Regexes catch the PII that shows up in
compliance audits, the recall on emails and phone numbers is genuinely high,
and the failure mode is invisible: nothing in your pipeline logs "did not
detect a person."

**What is actually true.** Regex is a high-precision, low-recall first pass
over *formatted identifiers*. Anything quasi-identifying - the combination of
attributes that uniquely picks out a person - is not lexical and is not
solvable by pattern matching. For a paper corpus the practical position is:
run the regex pass, accept that structured identifiers are handled, and then
be explicit that your corpus is public scholarly text whose authors are named
by design. That is a defensible scope statement. "We scrubbed PII" is not,
and it is exactly the claim an opposing expert enjoys taking apart.

## Training the tokenizer

<!-- fade: v2-bpe-compression -->

The model consumes integers, not text. The mapping from text to integers is
fixed before training starts and frozen for the model's life, and you are
going to train your own rather than borrow one. Here is the algorithm from
scratch, on a corpus small enough to run in your head.

**Byte-pair encoding.** Start with every symbol being a single character, then
repeatedly find the most frequent adjacent pair of symbols in the corpus and
replace it everywhere with a new single symbol. Stop when you have as many
symbols as you want. That is the whole algorithm.

Take a four-word corpus with these counts, and write `_` for end-of-word so
the algorithm can distinguish a word-final letter from a word-internal one:

```
t h e _              (6)
t h e o r e m _      (3)
t h e o r y _        (2)
t h e n _            (4)
```

*Round 1.* Count every adjacent pair, weighted by word count:
`(t,h)`=15, `(h,e)`=15, `(e,_)`=6, `(e,o)`=5, `(o,r)`=5, `(r,e)`=3,
`(e,m)`=3, `(m,_)`=3, `(r,y)`=2, `(y,_)`=2, `(e,n)`=4, `(n,_)`=4. Two pairs
tie at 15; break ties by first occurrence and take `(t,h)`.

Merge 1: **`th`**

*Round 2.* `(th,e)`=15 is now the unique maximum.

Merge 2: **`the`**

```
the _                (6)
the o r e m _        (3)
the o r y _          (2)
the n _              (4)
```

*Round 3.* `(the,_)`=6 beats `(the,n)`=4 and `(the,o)`=5.

Merge 3: **`the_`**

*Round 4.* `(the,o)`=5 and `(o,r)`=5 tie; first occurrence takes `(the,o)`.

Merge 4: **`theo`**

*Round 5.* `(theo,r)`=5 is the maximum.

Merge 5: **`theor`**

The learned merge list, in order, *is* the tokenizer:

```
1. t + h     -> th
2. th + e    -> the
3. the + _   -> the_
4. the + o   -> theo
5. theo + r  -> theor
```

Encoding a new string means splitting it into characters and applying that
list in learned order. Nothing is searched; the order is load-bearing. The
word `theorem` becomes `t h e o r e m _` then `th e o r e m _` then
`the o r e m _` then `theo r e m _` then `theor e m _`: four tokens
`[theor, e, m, _]` for the seven letters of `theorem` plus its end-of-word
marker.

Three properties, each of which you will need later:

**Nothing is unrepresentable.** The base symbols survive in the vocabulary, so
any string decomposes at worst into single bytes. Real tokenizers are
byte-level - the base alphabet is the 256 byte values, not characters - which
is what makes them total over arbitrary input including binary junk. There is
no unknown token.

**Frequency decides, not meaning.** `theor` became a symbol because that byte
sequence recurred, not because it is a morpheme. In a corpus of papers the
merges will discover `_tokeniz`, `_arXiv`, `\begin{`, `et al.` - and that is
the entire reason to train your own.

**A pretokenizer runs first.** Production BPE does not merge across arbitrary
boundaries; a regex splits text into chunks (words, numbers, punctuation runs,
whitespace) before merging, so no token ever spans a word boundary or glues
digits to letters. The GPT-4-style pretokenizer regex is the standard choice.
Numbers get special handling because place value is destroyed by frequency
merges - vol 1 covers why in detail; here it is enough to know that digit
grouping is a deliberate configuration and not an accident.

**The compression arithmetic.** A tokenizer's value is one number: characters
per token on *your* text. nanochat trains a 65,536-entry vocabulary on 2
billion characters in about a minute on one machine and reaches roughly 4.8
characters per token. That is the anchor - a domain tokenizer is a one-minute
job, not a project.

Worked, on a corpus the size of the one you will build:

| | chars/token | tokens for 2.4e9 chars |
|---|---|---|
| general-purpose vocabulary | 4.0 | 600,000,000 |
| domain-trained vocabulary | 4.8 | 500,000,000 |

$$2.4 \times 10^{9} / 4.0 = 6.00 \times 10^{8}, \qquad 2.4 \times 10^{9} / 4.8 = 5.00 \times 10^{8}$$

100 million tokens saved, 16.7% of the sequence length for the same text. The
training cost of a run is proportional to the token count (v3 makes this
exact), so that is 16.7% off the bill, or the same bill covering 16.7% more
text, from a one-minute job. It also shortens every document, which means more
documents fit in a fixed-length training sequence - and that interacts with
packing in the next section.

```beat
id: v2-b6
type: completion
concept: d-tokenizer-training
prompt: |
  Your corpus is 3.6e9 characters of paper text. You measure three
  tokenizers on a held-out slice of it.

    A: off-the-shelf 100K web vocabulary   3.6 chars/token -> ____ tokens
    B: domain-trained 16K vocabulary       4.5 chars/token -> ____ tokens
    C: domain-trained 32K vocabulary       4.8 chars/token -> ____ tokens

  Which filling of the three blanks, with the reduction from A to C and the
  reason B to C gains so much less than A to B, is correct?
options:
  - text: |
      1,000,000,000; 800,000,000; 750,000,000. Reduction A to C is
      (1000 - 750)/1000 = 25%. B to C gains little because BPE merges in
      frequency order: the first few thousand merges absorb the
      highest-frequency sequences and capture most of the available
      compression, so each doubling of $V$ buys a thinner slice of the
      frequency tail - roughly logarithmic gain against a table cost linear
      in $V$.
    correct: true
    explain: |
      Right. 3.6e9 divided by chars-per-token gives tokens, and the
      diminishing return is a property of the merge ordering, not of the
      measurement.
  - text: |
      1,000,000,000; 800,000,000; 750,000,000. Reduction A to C is 25%. B to C
      gains little only because 32K is still too small; the trend would
      continue if you kept doubling, so the largest vocabulary you can afford
      is the right one.
    misconception: V2-M3
    explain: |
      Arithmetic right, lesson missed. The gain is logarithmic in $V$ while
      the embedding table $V \times d_{model}$ is linear, and on a fixed small
      corpus the extra rows see too few updates to train. Optimal vocabulary
      shrinks when you are data-bottlenecked.
  - text: |
      12,960,000,000; 16,200,000,000; 17,280,000,000. The token count is
      characters times chars-per-token, so C produces the most tokens and the
      change from A to C is a 33% increase - which is the point, since more
      tokens is more training signal.
    misconception: V2-M3
    explain: |
      The units invert the relationship. Chars per token is a ratio you divide
      by: better compression means FEWER tokens for the same text, which is
      what cuts the bill, since training cost scales with token count.
  - text: |
      1,000,000,000; 800,000,000; 750,000,000. Reduction A to C is 6.25%,
      measured from B to C, and the B-to-C gain is small because held-out
      chars-per-token measurements are noisy at this corpus size.
    misconception: V2-M3
    explain: |
      The reduction asked for is A to C: (1000 - 750)/1000 = 25%. And the
      shrinking gain is structural, not noise - merges are chosen in frequency
      order, so the tail is progressively rarer by construction.
check: choice
```

<!-- refutes: V2-M3 -->
**You probably think a bigger vocabulary is straightforwardly better** - more
merges, better compression, shorter sequences, and the cost is just some
embedding parameters. Frontier models use 100K to 200K entries, so copy them.

**Here is the specific prediction that fails.** Under "bigger is better," a
128K vocabulary should beat a 16K vocabulary at every scale, since it is
strictly a superset of compression opportunities. Now price it. Your model is
30 million parameters with an internal width of $d_{model} = 384$. The input
embedding table has one row per vocabulary entry and $d_{model}$ columns:

$$|E| = V \times d_{model}$$

At $V = 128{,}256$: $128{,}256 \times 384 = 49{,}250{,}304$ parameters. That
is 1.6 times the size of the entire model, for a table of starting positions.
At $V = 16{,}384$: $16{,}384 \times 384 = 6{,}291{,}456$, about 21% of the
model, which is already a lot. The 128K vocabulary is not an improvement with
a cost; it is a different model in which almost all capacity is a lookup
table, and each of those 128,256 rows now gets updated only when its token
appears in a batch - which, on 500 million tokens of a single domain, is
rarely enough that most rows finish training near their random initialization.

```beat
id: v2-b7
type: compute
concept: d-tokenizer-training
prompt: |
  A model has $d_{model} = 384$ and 30,000,000 parameters in total. Its input
  embedding table holds $V \times d_{model}$ parameters, where $V$ is the
  vocabulary size.

  At $V = 128{,}256$, compute the ratio of embedding-table parameters to total
  model parameters.

  Give a single number to 2 decimal places.
answer: 1.64
check: numeric(0.01)
```

**Why the wrong model is appealing.** The frontier-model numbers are public
and large, and vocabulary size looks like a free knob because its cost is
invisible in a 70B model where the table is a couple of percent.

**What is actually true, and it has a law attached.** Optimal vocabulary size
grows with compute budget but *shrinks* when you are data-bottlenecked (Tao et
al., NeurIPS 2024). Both directions matter and the second is the one that
applies to you: when the corpus is fixed and small, a larger vocabulary
spreads a fixed number of token occurrences over more rows, and rows that see
few updates are wasted parameters. For a 10 to 30 million parameter model on
300 million to 1 billion tokens of a single domain, 16K to 32K is the range.
128K is copying a decision made under a completely different constraint.

## Mixing and packing

Two stages, both of which look like data plumbing, and one of which decides
whether attribution is possible at all.

**Mixing** sets how much of each source the model sees. The state of the
practice is still hand-chosen proportions: Dolma 3 uses roughly 28% web, 20%
code, 19% math, 14% question-answering, and so on, arrived at by ablation and
judgement. Three families try to learn the weights instead:

- **DoReMi** (arXiv:2305.10429) trains a small proxy model with group
  distributionally-robust optimization to find weights that minimize
  worst-case group loss, then trains the real model with those weights.
- **Data Mixing Laws** (arXiv:2403.16952) and **BiMix** fit a functional form
  for loss as a function of mixture proportions, then optimize the fitted
  function.
- **RegMix** (arXiv:2407.01492) trains hundreds of tiny models on *random*
  mixtures, regresses final loss on the mixture vector, and picks the argmin.

**Aioli** unifies these under one framework. For your purposes the important
one is RegMix, and not for mixing. Read its methodology again with attribution
eyes: train many small models on random subsets of the sources, fit loss as a
function of which sources were included, and read off each source's
coefficient. That is a per-source value estimate obtained from the same runs
that give you mixing weights. This is exactly the subset-inclusion regression
that v4 builds ground truth from, and v5 turns into a Shapley value. Note it
now: the mixing literature has already built the attribution baseline and
labelled it something else.

**Packing** is the last thing that happens before the GPU sees anything. The
trainer wants fixed-length sequences of $L$ tokens - dense, uniform, no
padding. Documents are variable-length. So the standard move is to concatenate
every document end to end with a separator token between them and slice the
resulting stream every $L$ tokens.

<!-- refutes: V2-M4 -->
**You probably think that concatenation is just batching** - a buffer-filling
detail, invisible to the model, because the separator token tells it where the
boundary is and attention will learn to respect it.

**Here is the specific prediction that fails.** Under "the separator handles
it," a sequence containing the tail of document 1 and the head of document 2
is equivalent to two shorter sequences. It is not, because the causal mask
does not know about the separator. A token in document 2's head attends to
every position before it in the sequence, which includes all of document 1's
tail. Gradients flow from document 2's loss through attention weights placed
on document 1's tokens. The model is being trained to use one source's text as
context for predicting another source's text - text the two documents' authors
never wrote near each other, and in a random pairing determined by shard
order.

For a language model this is a mild quality cost and people have shipped it
for years. For attribution it is fatal at the input, before any method runs.
Every gradient-based attribution technique in v8 asks "how did this training
example move the model," and the training example is a sequence. If the
sequence blends two sources, the attribution unit is a blend, and no amount of
downstream care recovers the split.

**Why the wrong model is appealing.** It genuinely almost does not matter for
loss. The separator token is right there and looks semantically sufficient.
And packing efficiency is a real, measurable win that everyone optimizes.

**What is actually true.** Use **intra-document causal masking**
(arXiv:2402.13991): the mask forbids attention across document boundaries
within a packed sequence, so a packed sequence behaves exactly like the
concatenation of independent shorter sequences. It costs a per-sequence
document-boundary array and a mask construction, both trivial, and it is
independently a small quality win. For this book it is close to mandatory: it
is what makes "this training sequence belongs to source $k$" a true statement
rather than an approximation.

Two consequences worth planning for now:

- With masking on, you can pack multiple documents per sequence freely, and
  the only sequences that mix sources are the boundary ones. Without masking,
  you would need one document per sequence, which wastes an enormous fraction
  of the token budget on padding for a corpus of short papers.
- Shuffling still matters, and it is a provenance decision. A shuffle that
  groups a source's documents together produces sequences that are cleanly
  single-source but training batches that are strongly correlated. Shuffle at
  the document level with a recorded seed, and record the seed, because data
  order is the single largest source of variation in influence measurements
  (v4 and v8 both depend on this being replayable).

```beat
id: v2-b8
type: predict
concept: d-mixing-packing
prompt: |
  You are packing with $L = 2048$ and intra-document causal masking enabled.
  A colleague proposes a packing optimization: sort documents by length
  descending and use a bin-packing heuristic to minimize padding, which raises
  token utilization from 94% to 99.4%.

  Before reading on, commit: which prediction about what this costs, and what
  one-line manifest change makes the cost recoverable rather than fatal, is
  right?
options:
  - text: |
      Which documents share a sequence is now decided by LENGTH, and length
      correlates with source (page limits by venue, a source of short
      abstracts co-packed with other short documents), so any per-sequence
      statistic later carries a source-correlated confound the packer
      introduced. Masking does not help: the problem is which documents are
      co-batched, not what attends to what. Record per training sequence the
      ordered (doc_id, source_id, start, length) spans, and the confound
      becomes conditionable instead of invisible.
    correct: true
    explain: |
      Right, including the part that masking cleans the input and not the
      sampling. And the trade is lopsided: 5.7% utilization against an
      unbounded confound, on a bill this unit already cut 16.7% with a
      one-minute tokenizer job.
  - text: |
      It costs nothing. Intra-document causal masking makes a packed sequence
      behave exactly like the concatenation of independent shorter sequences,
      so which documents land together is invisible to the model and
      therefore to any measurement built on it; take the 99.4%.
    misconception: V2-M4
    explain: |
      Masking fixes the input, not the sampling. No token attends across a
      boundary, true - but a per-sequence gradient still sums whatever
      documents the packer put in that sequence, and now the packer chose them
      by a length that tracks source identity.
  - text: |
      The cost is that gradients will flow from one source's loss through
      attention placed on another source's tokens, blending the attribution
      unit; the fix is to record which sequences are mixed so those can be
      excluded from gradient-based analysis.
    misconception: V2-M4
    explain: |
      That is the unmasked failure, and masking already forbids it here. The
      remaining damage is statistical - sequence composition correlated with
      source - and the answer is to record composition for every sequence, not
      just the mixed ones, so any effect can be tested against it.
  - text: |
      Reject the optimization: any packing order that is not uniformly random
      destroys attribution, so 94% utilization with random packing is the only
      defensible configuration.
    misconception: V2-M4
    explain: |
      Too blunt, and it skips the design skill being tested. A confound you
      have recorded per sequence is one a later analysis can condition on;
      the failure is an unrecorded confound, not a nonrandom packer.
check: choice
```

## The provenance manifest

<!-- fade: v2-shard-offset -->

Everything above produces text. This section produces the artifact the rest of
the book queries: a mapping from any position in the training data back to a
named source, and a dataloader that will produce the same sequence of batches
every time it runs.

**The physical layout.** A shard is a flat file of token IDs, fixed-width
integers, appended in document order. The manifest is one row per document:

```
doc_id | source_id | shard_id | offset | length | cluster_id | attrs
```

where `offset` is the index of the document's first token within its shard and
`length` is its token count. Two documents' spans never overlap and a document
is never split across shards. That last constraint costs padding and buys a
single-row answer to "where is this document," which is the query every later
stage makes.

Work the layout by hand. Shard capacity 100,000 tokens, documents appended in
order:

| doc | tokens | shard | offset | running total |
|---|---|---|---|---|
| doc_0 | 12,400 | 0 | 0 | 12,400 |
| doc_1 | 8,600 | 0 | 12,400 | 21,000 |
| doc_2 | 41,000 | 0 | 21,000 | 62,000 |
| doc_3 | 30,500 | 0 | 62,000 | 92,500 |
| doc_4 | 19,900 | 1 | 0 | - |

doc_4 does not fit: $92{,}500 + 19{,}900 = 112{,}400 > 100{,}000$. Under the
no-split rule it starts shard 1 at offset 0, and shard 0 carries 7,500 tokens
of padding. That is 7.5% waste on this shard, and it is the price of the
constraint.

**The logical layout.** The trainer does not read documents; it reads
fixed-length sequences. With sequence length $L = 2048$, sequence $s$ covers
shard positions $[2048s,\ 2048(s+1))$. So for any document you can compute
which sequences touch it by integer division:

$$s_{\text{first}} = \left\lfloor \frac{\text{offset}}{L} \right\rfloor, \qquad s_{\text{last}} = \left\lfloor \frac{\text{offset} + \text{length} - 1}{L} \right\rfloor$$

Take doc_2: offset 21,000, length 41,000, so its last token is at position
61,999.

- $\lfloor 21{,}000 / 2048 \rfloor = 10$, since $10 \times 2048 = 20{,}480$ and
  $11 \times 2048 = 22{,}528$. Position within sequence 10:
  $21{,}000 - 20{,}480 = 520$.
- $\lfloor 61{,}999 / 2048 \rfloor = 30$, since $30 \times 2048 = 61{,}440$.
  Position within sequence 30: $61{,}999 - 61{,}440 = 559$.

doc_2 spans sequences 10 through 30, which is 21 sequences. Sequences 11
through 29 contain nothing but doc_2 and are cleanly single-source. Sequences
10 and 30 are boundary sequences that also contain a neighbour. Two of
twenty-one, about 10%, and that fraction is the thing document masking makes
harmless.

```beat
id: v2-b9
type: completion
concept: d-mixing-packing
prompt: |
  Same shard, same $L = 2048$. doc_3 sits at offset 62,000 with length 30,500.

    last token position   = 62,000 + 30,500 - 1        = ____
    first sequence        = floor(62,000 / 2048)       = ____
    position within it    = 62,000 - (____ x 2048)     = 560
    last sequence         = floor(____ / 2048)         = 45
    position within it    = ____ - (45 x 2048)         = 339

  Which filling of the five blanks is right, and what does it say about how
  many sequences doc_3 spans and how many of those are shared with another
  document?
options:
  - text: |
      92,499; 30; 30; 92,499; 92,499. doc_3 spans sequences 30 through 45,
      which is 16 sequences, two of which are boundary sequences shared with a
      neighbour (sequence 30 with doc_2, sequence 45 with the shard's trailing
      padding).
    correct: true
    explain: |
      Right. $30 \times 2048 = 61{,}440$ and $31 \times 2048 = 63{,}488$, so
      the first sequence is 30 and the position within it is
      $62{,}000 - 61{,}440 = 560$. The last token is at $92{,}499$, and
      $45 \times 2048 = 92{,}160$, giving position $339$. The range 30 to 45 is
      inclusive: $45 - 30 + 1 = 16$. Sequence 30 begins at 61,440 and holds
      doc_2's last 560 tokens before doc_3 starts.
  - text: |
      92,499; 30; 30; 92,499; 92,499. doc_3 spans $45 - 30 = 15$ sequences,
      two of which are boundary sequences shared with a neighbour.
    misconception: V2-M4
    explain: |
      The blanks are right and the span is not. $45 - 30$ counts the gaps
      between sequence indices, not the sequences themselves; the range is
      inclusive, so it is $45 - 30 + 1 = 16$. This off-by-one is the most
      common error in the whole manifest, and because it drops one sequence
      from every document's record it never shows up as a crash - exactly the
      kind of silent bookkeeping damage that treating packing as plumbing
      invites.
  - text: |
      92,499; 31; 31; 92,499; 92,499. doc_3 begins partway through a sequence,
      so it is counted from sequence 31 and spans 31 through 45, which is 15
      sequences.
    misconception: V2-M4
    explain: |
      This rounds up where the arithmetic floors. $31 \times 2048 = 63{,}488$,
      which is past offset 62,000 entirely - the given anchor
      $62{,}000 - (\_\_ \times 2048) = 560$ can only be satisfied by 30. A
      document's first sequence is the one its first token lands in, partway
      through or not, and that is $\lfloor \text{offset} / L \rfloor$.
  - text: |
      92,499; 30; 30; 92,499; 92,499. doc_3 spans 16 sequences and none of
      them are shared: the separator token written between documents closes
      doc_2 out, so sequence 30 contains only doc_3.
    misconception: V2-M4
    explain: |
      The blanks are right; the boundary claim is not. Sequence 30 covers
      shard positions $[61{,}440,\ 63{,}488)$, and doc_2's last token sits at
      61,999 - so 560 tokens of doc_2 are physically in that sequence, whatever
      separator was written between them. A separator marks a boundary; it does
      not split a sequence. That is why intra-document masking exists and why
      the manifest has to record per-sequence composition.
check: choice
```

**Determinism.** The manifest is only useful if the training run is
replayable. Three things have to be pinned, and all three are ordinary
engineering:

1. **Shuffle seed and shuffle algorithm.** Document order, and the order of
   sequences within an epoch, must be a pure function of a recorded seed.
2. **Batch composition.** Given the seed, batch $n$ contains a specific set of
   sequences, computable without running the training loop. Attribution
   methods in v8 need to ask "what was in batch 4,200" months later.
3. **Bitwise reproducibility of the arithmetic**, which is the hard one.
   Floating-point addition is not associative, so changing the number of GPUs,
   the gradient accumulation steps, or the kernel implementations changes the
   trained weights - not by a rounding error, but into a different basin.
   Levanter (Stanford) is built specifically to give bitwise-identical results
   regardless of hardware and parallelism configuration, which is an unusual
   guarantee and unusually relevant here: an attribution claim that cannot be
   re-derived from the manifest and the seed is an assertion, not a
   measurement.

**Tools.** `datatrove` is the best solo default - composable pipeline blocks,
with the entire FineWeb pipeline shipped as a roughly 100-line example. The
**Dolma toolkit** is the closest existing thing to provenance-preserving
processing: a Rust tagging framework where each stage attaches per-document
*attributes* rather than filtering in place, so a document accumulates a
record of every judgement made about it and nothing is destructively removed
until you choose to materialize a subset. That design is exactly what this
unit has been arguing for, generalized. Use it, or copy its shape.

## At the bench: the tagged corpus

<!-- canon-only -->

The build step for this unit produces the substrate every later unit runs on.
Everything below is systems work; there is no math in it.

**What to build.**

1. **Acquire and filter by licence.** Pull the peS2o subset - roughly 40
   million open-access papers, cleaned for pretraining, distributed under
   ODC-By with a per-document licence in the metadata. Filter to the cleanly
   licensed subset (the Common Pile mirror is already restricted this way). If
   you supplement from arXiv's bulk S3 source bucket, filter to the Creative
   Commons subset first, because the default arXiv licence does not permit
   redistribution. Record the licence string per document as an attribute; it
   is the first column an auditor asks for.
2. **Tag 10 to 12 sources.** Partition by venue, author group, or subfield -
   whatever partition you would defend as a payment unit. Named up front,
   fixed for the rest of the book, one `source_id` per document. Aim for
   sources of roughly comparable token count; wildly unequal sources make the
   v4 and v5 measurements harder to read, though not wrong.
3. **Dedup, recording clusters.** MinHash+LSH over 5-gram shingles, union-find
   over the candidate graph, and emit the cluster table with full member lists
   carrying `source_id`. Choose the credit policy and write it down *before*
   inspecting which sources are in which clusters. Then exact-substring dedup
   with a suffix array at a 50-token threshold, recorded the same way.
4. **Train the domain tokenizer.** Byte-level BPE, 16K to 32K vocabulary, on a
   sample of the filtered corpus. Measure characters per token against an
   off-the-shelf vocabulary on held-out domain text and record both numbers -
   that ratio is a line in the pitch.
5. **Emit shards and the manifest.** Fixed-width token files, no document split
   across a shard, one manifest row per document with `(shard_id, offset,
   length)` plus `source_id`, `cluster_id`, and the attribute set. Emit
   document-boundary arrays per sequence so the trainer can build
   intra-document masks. Pin the shuffle seed.

**What it feeds.** v3 trains on these shards. v4 retrains with one source
removed at a time, which requires the manifest to answer "which spans belong
to source $k$" in one pass. v5 enumerates coalitions of the sources you named
in step 2. v7 injects watermarks into one of them and needs to know exactly
which shards changed. v8 asks which training sequences moved the model, and
gets a real answer only because of step 5.

**The two decisions that are not reversible later.** The source partition
(step 2) and the choice to record duplicate clusters rather than resolve them
(step 3). Everything else in the list can be rerun in an afternoon. Those two
are welded into every measurement in the rest of the book.

```beat
id: v2-b10
type: self-explain
concept: d-dedup-provenance
prompt: |
  Your dedup pass merges doc_417 (source: Northlake Preprints) and doc_101
  (source: Journal of Text Systems) into one cluster, keeping doc_101's tokens
  as the representative. doc_101 lands at shard 3, offset 44,100, length
  9,800.

  In v4 you will retrain with Northlake Preprints removed and measure the
  change in held-out loss. Which account of what the manifest and cluster
  table must hold for that experiment to be well defined is right?
options:
  - text: |
      The manifest row for doc_101 - (shard 3, offset 44,100, length 9,800),
      source_id = Journal of Text Systems, cluster_id = c - plus a cluster row
      for c whose member list carries doc_417 WITH its source_id. Without that
      member list, "remove Northlake" silently means "drop rows tagged
      Northlake", when it could also mean "drop those rows and the
      representatives of clusters Northlake belongs to". The two give different
      numbers and both are legitimate; the delta is only defined once you say
      which you ran.
    correct: true
    explain: |
      Right, and (3) is the load-bearing part: doc_101's tokens ARE the content
      doc_417 contributed, carrying a different owner's label. One reading
      measures Northlake's exclusive content, the other its total content
      including what someone else also holds. With only the representative
      recorded, the second reading is not computable and the first is reported
      as if it were the whole answer - undetectable afterwards, since the corpus
      keeps no trace of the merge.
  - text: |
      Only the manifest span and source_id are needed. Removing a source is a
      filter: drop every row whose source_id is Northlake Preprints and
      retrain. doc_101 belongs to the Journal, so it stays, and the cluster
      table is a maintenance record with no bearing on the experiment.
    misconception: D9
    explain: |
      This is dedup-as-hygiene. The survivor of a duplicate cluster was chosen
      by shard iteration order, and that choice moved a passage's credit from
      Northlake to the Journal. Filtering on source_id therefore measures
      Northlake minus whatever the pipeline happened to reassign - a quantity
      set by iteration order, reported as the source's contribution.
  - text: |
      Sidestep it: keep both doc_417 and doc_101 in the corpus so each source's
      content is present under its own label, and then removing Northlake is
      unambiguous because nothing was ever merged away.
    misconception: D9
    explain: |
      This changes the model rather than defining the measurement. Training on
      both copies doubles that passage's effective weight and makes it far more
      extractable (v6), so the leave-one-out delta you measure belongs to a
      different corpus than the one you ship. Dedup still has to happen; what
      has to change is that the cluster is recorded rather than resolved.
  - text: |
      Record the representative and its cluster's similarity scores. Membership
      detail is not needed, because a merge large enough to matter would show
      up as an anomaly in the leave-one-out delta itself - the delta is a stable
      property of the corpus, so run it and inspect the number.
    misconception: D16
    explain: |
      The delta is not stable enough to serve as its own audit. Seed and data
      order alone can swing a source's marginal contribution by more than its
      own magnitude, so a mis-assigned cluster is indistinguishable from run
      noise in the output. The ambiguity has to be resolved in the schema,
      before the run, not read back out of the result.
check: choice
```

## Notation in this unit

<!-- canon-only -->

Reference, not reading. Nothing below is new.

| Symbol | Means | Typical value |
| --- | --- | --- |
| $J(A,B)$ | Jaccard index: shared shingles over total distinct shingles | 0 to 1; dedup threshold ~0.75 |
| $k$ | number of hash functions in a MinHash signature | 112 (FineWeb) |
| $b$, $r$ | LSH bands and rows per band, with $k = b \times r$ | 14 and 8 (FineWeb) |
| $V$ | vocabulary size: number of distinct tokens | 16K to 32K here; 100K+ frontier |
| $d_{model}$ | model width, the vector size carried between layers | 384 at 30M params |
| $L$ | training sequence length in tokens | 1024 or 2048 |
| $\lvert E \rvert$ | parameters in the embedding table, $V \times d_{model}$ | see the tokenizer section |

Three quantities the manifest is built from, all integers, all per document:
`shard_id` names the file, `offset` is the index of the first token within
that file, `length` is the token count. Sequence index and position within a
sequence are derived, never stored: $\lfloor \text{offset} / L \rfloor$ and
$\text{offset} \bmod L$.

Two abbreviations used above without expansion. **LSH** is locality-sensitive
hashing: any scheme where similar inputs collide in a hash bucket on purpose.
**BPE** is byte-pair encoding, the merge algorithm in the tokenizer section.
