## What the extractor actually hands you

**The pair.** Running a layout-unaware extractor on a two-column PDF is like
running **`strings` on a binary** and calling it decompilation, and like
**parsing HTML with a regex**.

**The shared structure.** In all three, a tool that operates one level below
the structure you care about produces output that is locally plausible and
globally meaningless. `strings` gives you real, correctly-decoded text - every
byte of it is genuinely in the binary - arranged in an order that has nothing
to do with program structure. The regex extracts real tag contents, correctly,
from a document whose nesting it cannot see. The naive extractor emits every
glyph that is actually on the page, in the order the renderer emitted them.
None of them is buggy. Each is answering a question one level too low, and the
tell is the same in every case: the output passes eyeball inspection in small
samples and falls apart on anything with real structure.

**Where this breaks.** Both software analogies have an unambiguous ground
truth one level up - the binary has a real control-flow graph, the HTML has a
real DOM, and a correct tool recovers it exactly. A PDF has no stored reading
order at all. There is no structure to parse, only geometry to *infer*, which
is why the state of the art here is a 7B vision-language model rather than a
grammar. The analogy also understates the damage: a wrong `strings` output is
obviously garbage and gets discarded, while wrong extraction is fluent enough
to survive every quality heuristic and land in training data.

## Extraction: three source types, three tools

**The pair.** Choosing LaTeX source over OCR is like **reading the git history
instead of diffing two release tarballs**, and like **parsing a build's
structured log output instead of scraping its console text**.

**The shared structure.** In each, the information you are trying to
reconstruct was explicitly recorded upstream by a producer who had to record
it, and the hard reconstruction problem downstream exists only because someone
threw the record away. The author typed `\section{Model architecture}`; the
committer wrote a commit message and a parent pointer; the build tool emitted
a structured event. Reconstructing any of those from the rendered artifact is
possible, actively researched, and strictly worse than reading the original.
The engineering instinct is identical in all three: before you build the
inference layer, check whether the producer still has the source.

**Where this breaks.** Git history and structured logs are complete - every
commit has a message, every event is emitted. LaTeX source is available for a
*biased subset*: newer papers, TeX-submitting communities, arXiv only. So the
analogy suggests "always use the source," while the correct move is "use the
source and record which documents got it," because coverage correlates with
document properties that will later be confounded with source identity. There
is no equivalent in the git case of "history exists for the commits from one
team and not another."

---

**The pair.** The trafilatura/resiliparse choice is like tuning a **spam
filter's threshold ahead of a manual review queue**, and like choosing
**compiler warning levels ahead of a linter**.

**The shared structure.** In both, the correct aggressiveness for stage one
depends entirely on what stage two does. A high-recall spam filter is right if
humans triage the quarantine and wrong if the quarantine is deleted. `-Wall`
plus a strict linter beats `-Werror` alone. DCLM picks the permissive
extractor because a top-10% classifier follows; FineWeb picks the conservative
one because it does not lean on a hard classifier cut. Quoting either stage's
metrics in isolation is a category error in all three settings.

**Where this breaks.** Warnings and spam quarantines are reversible - the
message sits there, the warning is in the log, and you can re-triage. Content
dropped by an extractor is gone before anything is recorded, because the
extractor runs before the document exists as an object in your pipeline. That
is the specific reason the Dolma toolkit's design (tag, do not delete) is
worth copying: it converts an irreversible stage into a reversible one, which
is what the analogy assumes for free.

## Quality filtering: the classifier is the lever

**The pair.** The filtering classifier is like a **query planner's cost
model**, and like **branch-prediction hints in a hot loop**.

**The shared structure.** In each, a small, unglamorous, statistically-fitted
component sits in front of an enormously expensive process and decides what
the expensive process spends itself on. The cost model is a few hundred lines
of estimation and it is worth more than any single operator implementation,
because it decides which operators run at all. The classifier is a regression
head on frozen embeddings and it is worth more than the architecture changes,
because it decides which tokens the training run spends its capacity on. In
both cases the leverage comes from position in the pipeline, not from
sophistication, and in both cases teams systematically under-invest because
the component looks trivial next to what it steers.

**Where this breaks.** A cost model has a checkable ground truth: run the plan,
measure the time, and you know whether the estimate was right. The quality
classifier's target ("educational value") is a judgement with no ground truth
at all - FineWeb-Edu's labels are one large model's opinion, and its 82% F1 is
agreement with that opinion, not with correctness. You cannot validate the
filter directly; you can only train two models and compare downstream
benchmarks, which costs thousands of GPU-hours per evaluation of the filter.
The feedback loop is four orders of magnitude slower than a query planner's,
which is why this field moves by large public ablations rather than by local
iteration.

---

**The pair.** The 1.3T-beats-15T result is like **a smaller working set
beating a larger one because it fits in cache**, and like **a curated
allowlist outperforming a comprehensive blocklist**.

**The shared structure.** Both are cases where the "more coverage is strictly
better" intuition fails because a fixed budget is being allocated, not
extended. Cache lines spent on cold data are cache lines not spent on hot
data; the larger working set is not neutral, it is actively evicting. Model
capacity is the same kind of budget (v6 puts it at roughly 3.6 bits per
parameter), so tokens spent representing spam regularities are capacity not
spent elsewhere. In both cases the effect is strongest when the budget is
tightest - small caches, small models - which is exactly your regime.

**Where this breaks.** Cache contents are transient and the working set can be
re-shaped at runtime; a model's allocation of capacity is decided once, during
a training run you cannot rewind. There is also no analogue of a cache miss:
when a model lacks capacity for something it does not stall and fetch, it
simply produces a worse answer with no signal that anything was missing. That
silence is why filtering decisions have to be made and measured up front
rather than diagnosed from production behaviour.

## Deduplication, worked by hand

**The pair.** MinHash signatures are like **rsync's rolling-checksum block
matching**, and like a **Bloom filter as a fixed-size stand-in for a set**.

**The shared structure.** All three replace an object with a small, fixed-size
sketch whose comparison approximates a property of the original, and all three
are chosen because the exact computation is impossible at the target scale.
rsync will not ship the file to compare it; MinHash will not materialize the
shingle set. In each, the sketch size is a tunable that trades accuracy for
memory with a known error curve, and in each the correct engineering posture
is to treat the sketch as a *filter* whose output gets verified, not as an
answer. rsync verifies with a strong hash after the rolling match; LSH
verifies candidate pairs after bucketing.

**Where this breaks.** A Bloom filter's error is one-sided and adversarially
bounded: no false negatives ever, by construction. MinHash error is two-sided
and probabilistic - genuine duplicates get missed, and the miss rate is a
function of the band configuration you chose. So "we deduplicated" is a
statement with an error rate attached, and the error rate is a design
parameter you must report. rsync's analogy breaks on the other side: rsync is
matching *positions in a known pair of files*, whereas LSH is doing a
similarity join across a billion documents, and the entire difficulty is
avoiding the quadratic comparison that rsync gets to assume away.

---

**The pair.** LSH banding is like a **multi-level page table walk that
short-circuits on the first miss**, and like **early-exit predicate evaluation
in a query filter**.

**The shared structure.** In all three you arrange a sequence of cheap tests so
that the common case - "these are not related" - is rejected as early and as
cheaply as possible, and the expensive check runs only on survivors. Requiring
all $r$ rows of a band to match is a conjunctive test that fails fast on the
first mismatched row; trying $b$ bands is a disjunction that succeeds fast on
the first hit. The whole structure is selectivity engineering, and the tuning
question - how many rows, how many bands - is the same shape as ordering
predicates by selectivity.

**Where this breaks.** Predicate ordering does not change *which* rows come
back; it changes only how fast you get them. Band configuration changes the
answer. Raise $r$ and real duplicates start slipping through undetected; raise
$b$ and you drown in candidates. There is no configuration that is merely
faster, so this is not an optimization at all - it is choosing the similarity
threshold, wearing an optimization's clothing. Treating it as a performance
knob and tuning it for throughput is a real and common way to silently change
what your corpus contains.

## Dedup is a payout decision

**The pair.** Silently choosing a dedup survivor is like **content-addressed
storage collapsing two users' identical blocks**, and like **`git gc` keeping
one blob for a file that two branches both introduced**.

**The shared structure.** In all three, an identical byte sequence arriving
from two independent origins gets stored once, and the storage layer has no
concept of origin because origin was never part of its job. Deduplicating
block storage is unambiguously correct engineering. Git's object store is
unambiguously correct - the blob is the content, and which branch introduced
it is recorded elsewhere, in the commit graph. The shared structure is the
separation itself: content storage collapses duplicates, and a *separate*
structure retains provenance.

**Where this breaks.** And here the analogy stops being a comparison and
becomes the fix. Git gets this right: the blob is deduplicated and the commit
graph still knows both branches touched it, so `git log --follow` answers the
provenance question against a deduplicated store. A dedup pass that keeps one
document and forgets the cluster is not the git model - it is the git model
with the commit graph deleted. Content-addressed filesystems get away with
having no commit graph because no user is owed money for a block. The moment
there is a claimant, the storage layer alone is insufficient, and the answer is
not to stop deduplicating. It is to add the missing side table, exactly as git
already has.

---

**The pair.** A credit policy imposed by shard iteration order is like
**last-write-wins conflict resolution in a distributed store**, and like
depending on **unspecified iteration order of a hash map**.

**The shared structure.** In each, a real decision with real consequences is
made by an implementation detail that nobody chose, that is not documented,
and that changes when unrelated things change - the shuffle seed, the shard
count, the hash function's seed, the clock skew between replicas. Every one of
these is a well-known category of bug, and the standard fix is always the
same: make the resolution rule explicit and deterministic rather than
inherited from the substrate. You have written that fix many times.

**Where this breaks.** Last-write-wins is at least *auditable after the fact* -
the surviving record carries a timestamp, and you can reconstruct what
happened. A dedup survivor carries nothing. The corpus contains one copy of
the passage tagged with one source, and there is no artifact anywhere,
including the trained model, from which the merge can be detected. Hash
iteration order at least reproduces if you pin the seed; here even pinning
gives you a stable *arbitrary* answer rather than a defensible one. The
undetectability is what makes this worse than the familiar bugs it resembles,
and it is why the fix has to be preventive rather than diagnostic.

## Decontamination and PII

**The pair.** n-gram decontamination is like **checking a benchmark suite for
inputs that appear in the code under test**, and like **cache-key collision
auditing before trusting a hit rate**.

**The shared structure.** All three are guards against measuring the wrong
thing because two supposedly-separate populations overlap. The mechanism is
identical: enumerate one set, test membership for the other, and accept a
generous false-positive rate because the cost of a false positive (drop one
item) is trivially small next to the cost of a false negative (your headline
number is meaningless). Bloom filters show up in all three for exactly that
asymmetry.

**Where this breaks.** A test input either appears in the code or it does not,
and the check is exact. Decontamination is a lexical proxy for a semantic
property: a training document that paraphrases an eval question shares no
13-gram and leaks anyway. So the n-gram pass bounds *accidental verbatim*
leakage and nothing else, and reporting "we decontaminated" implies a
guarantee the technique cannot give. The whole-document exclusion of your
held-out eval papers is the part that is actually exact, and it is a different
mechanism doing a different job.

---

**The pair.** Regex PII scrubbing is like **secret-scanning a repository with
entropy heuristics**, and like **field-level masking in a database export**.

**The shared structure.** Each catches *formatted* sensitive values with high
precision - an AWS key has a shape, an email has a shape, a `ssn` column has a
name - and each gives an operator a satisfying, auditable report of what it
removed. All three are correct and worth running, and all three have a recall
ceiling set by the fact that they can only see what is syntactically marked.

**Where this breaks.** Secret scanning has a hard target: a credential either
authenticates or it does not, so a missed secret is at least discoverable by
someone trying it, and rotation is a remedy. Personal identifiability has no
such boundary and no remedy. It emerges from *combinations* of individually
innocuous fields - institution, subfield, year, career stage - and combinations
multiply out of the population size quickly. Field-level masking assumes the
identifying information lives in identifiable fields, which is precisely the
assumption that fails. The correct engineering output here is not a better
matcher but a scope statement: "formatted identifiers removed; this is public
scholarly text with named authors." That is a claim that survives an adversary
reading it closely, which "PII scrubbed" is not.

## Training the tokenizer

**The pair.** BPE is like **building a compression dictionary from the corpus
you are about to compress**, and like **interning strings into an enum before
the hot path runs**.

**The shared structure.** All three replace variable-length, frequently-recurring
byte sequences with short fixed-width identifiers, chosen by frequency, so that
the expensive downstream stage handles fewer, denser symbols. The dictionary is
built once from a sample and then applied deterministically; the mapping is
frozen; downstream code never sees the original bytes. The engineering payoff
is the same in each case, and so is the tuning question: a bigger dictionary
compresses better and costs more to carry.

**Where this breaks.** Two ways, and both matter here. Against compression
dictionaries: a compressor's dictionary is chosen to minimize output size and
nothing else, and it is exactly invertible - decompression reproduces the
input. The tokenizer's vocabulary is chosen the same way but the downstream
consumer is a learner, not a decoder, so vocabulary size trades against
*parameter budget and update frequency per row*, considerations a compressor
does not have. A vocabulary entry that is never exercised is a compression win
and a training loss, simultaneously.

Against interning: an enum's members are chosen by a person who knows the
domain, and each one means something. BPE's symbols are frequency artifacts
with no semantic warrant whatsoever. `theor` is not a morpheme; it is a byte
sequence that recurred. Reading meaning into token boundaries is the single
most common mistake people make with tokenizers, and the interning analogy is
what encourages it.

---

**The pair.** Copying a frontier model's 128K vocabulary into a 30M-parameter
model is like **importing a service's production connection-pool settings into
a sidecar**, and like **sizing a hash table for the peak load of a different
system**.

**The shared structure.** In all three, a number that was correct under one
resource envelope is copied into a very different one, and the failure is
quiet: nothing errors, the system runs, and the waste is a fixed cost that
never shows up as a symptom. A 500-connection pool in a process handling three
requests per second does not crash; it holds memory and file descriptors that
nothing else can use. A 128K vocabulary in a 30M model does not crash; the
embedding table becomes 1.6 times the size of the entire model and each row
finishes training near its initialization.

**Where this breaks.** An oversized pool or hash table is pure waste that you
can shrink at any time by editing a config. The oversized vocabulary is not
recoverable: it is frozen before training and every parameter in the model is
keyed to it, so changing it means retraining from scratch. It is also not pure
waste - it genuinely does compress better - which makes it the more dangerous
kind of misconfiguration, the kind with a real benefit attached and an invisible
cost.

## Mixing and packing

**The pair.** Packing documents without intra-document masking is like
**coalescing unrelated writes into one block and losing which caller wrote
what**, and like **batching independent gRPC requests into one call whose
handler shares mutable state across them**.

**The shared structure.** Each is a legitimate throughput optimization that
merges independent units of work into one physical unit, and in each the
merge is safe only if the merged items cannot influence one another. Coalescing
is fine until you need per-caller accounting. Request batching is fine until
one request's state leaks into another's handling. Packing is fine until you
need per-source accounting or until attention crosses the boundary - and
without masking, both problems occur at once.

**Where this breaks.** In the write-coalescing case you can usually reconstruct
attribution afterwards, because the block is a concatenation and the byte
ranges are recoverable. Here you cannot. The contamination is not in the stored
bytes; it happens inside a nonlinear function of shared parameters during the
forward and backward pass, and the two sources' contributions are summed before
any quantity you could log exists. There is no post-hoc decomposition. The
request-batching analogy is the closer one for exactly this reason: state
leakage inside a handler is also not recoverable from the response, and the fix
is also structural isolation rather than better logging.

---

**The pair.** RegMix's methodology is like **A/B testing feature flags in
random combinations and regressing the outcome on the flag vector**, and like
**a sensitivity sweep over config parameters to find which ones actually move
a metric**.

**The shared structure.** In all three you cannot afford to isolate one factor
at a time, so you randomize the whole configuration many times, measure the
outcome, and let a regression tell you each factor's marginal effect. The
per-factor coefficient is the deliverable, and it is obtained without ever
running a clean one-factor-at-a-time experiment. This is why the technique
scales to a dozen sources when pairwise ablation would not.

**Where this breaks.** Feature-flag experiments have cheap, fast, low-variance
outcomes; you can run thousands of trials and detect small effects. Here each
trial is a training run, and two identical configurations with different random
seeds produce different losses. That seed noise is a hard floor: a coefficient
smaller than it is unmeasurable no matter how many runs you do, which is the
opposite of the A/B-testing situation where more traffic always buys more
resolution. The regression also assumes effects add up linearly, so "source A
only helps when source B is present" is invisible to it - and interaction
effects are precisely what the Shapley machinery in v5 exists to capture.

## The provenance manifest

**The pair.** The manifest is like an **inode's extent map**, and like a **git
packfile index**.

**The shared structure.** All three are the same object: a compact side table
mapping a logical entity to a `(container, offset, length)` triple in a large
opaque blob, sorted so lookups are logarithmic or better, built once at write
time because building it later would require rescanning everything. The
performance argument is identical in each case, and so is the design
constraint: the blob format is chosen to make the index simple, not the other
way around. The no-split-across-shards rule is exactly the reasoning behind
preferring contiguous extents - you accept fragmentation waste to keep the
map one entry per object.

**Where this breaks.** An extent map and a packfile index are pure performance
artifacts. Delete them and you can rebuild them by scanning, losing time and
nothing else. The manifest is not reconstructible: `source_id` and `cluster_id`
are facts about where the tokens came from, and once the shard is written they
exist nowhere in the data. Scanning a shard tells you the tokens and nothing
about their owner. That is the inversion worth holding on to - this looks
exactly like an index, and it is actually the primary record, with the shard as
the derived artifact.

---

**The pair.** Requiring bitwise-reproducible training is like insisting on
**hermetic, reproducible builds**, and like **pinning a lockfile before you
bisect**.

**The shared structure.** In all three, the goal is not the artifact itself but
the ability to attribute a *difference* between two artifacts to a specific
input change. Nobody wants reproducible builds for their own sake; they want to
be able to say "this binary differs because of that commit" rather than
"because of the build host." A bisect over an unpinned dependency graph
produces confident, wrong answers. The discipline is identical: eliminate every
source of variation except the one under study, before running the comparison,
because afterwards you cannot separate them.

**Where this breaks.** A non-hermetic build usually produces an artifact that
is functionally identical - different timestamps, same behaviour - so the
irreproducibility is cosmetic and you tolerate it until you need it. Training
is chaotic: a last-bit difference in one gradient reduction sends the run to a
genuinely different model with the same loss. So the failure is not cosmetic
and not detectable by comparing outputs, since both models perform equally
well. The analogy also understates the stakes here in a specific way: with
builds, non-determinism costs you debugging time; with attribution, it silently
adds a term to a number you are going to put in front of a rightsholder.
