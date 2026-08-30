---
unit: v8
depth: se-analogies
---

## The bill for one honest answer

**The pair.** Answering "why did the model say this" by retraining is like
**running a full `git bisect` from scratch for every incoming bug report**, and
like **serving every lookup with a full table scan because nobody built an
index**.

**The shared structure.** All three have an exact procedure available whose cost
is charged per question rather than amortized across questions. Bisect really
does identify the commit; the table scan really does find the row; the retrain
really does measure the contribution. None of them get cheaper on the second
question, because none of them leave behind an artifact the next question can
reuse. And in all three cases the engineering response is identical: stop
answering the question directly, do one expensive pass that produces a compact
queryable structure, and answer from the structure. The gradient index is that
structure. The one-time sweep is the build; the per-query dot products are the
lookup.

**Where this breaks.** A database index returns *the same rows* the scan would
have returned - the index is a performance optimization with an exactness
guarantee, and a wrong index is a bug. The gradient index has no such guarantee.
It returns a different answer than retraining would, by an unknown amount, and
the size of that gap is the thing the rest of the unit is about. Treating it as
"the same answer, faster" is the error that makes the whole approach dishonest.
Bisect breaks the analogy in the other direction: it has a deterministic answer
that does not depend on a random seed, whereas two retrains with different seeds
give different contributions, so even the expensive procedure here has an error
bar and the index is being compared against a fuzzy target.

## A gradient is a direction

**The pair.** A gradient dot product between two training examples is like
**asking whether two concurrent patches to the same repository will merge
cleanly, conflict, or touch disjoint files**, and like **asking whether two
background jobs sharing a cache warm the same entries, evict each other, or work
on disjoint key ranges**.

**The shared structure.** In every case there is one piece of shared mutable
state (the working tree, the cache, the parameters), each actor has an intent
expressed as a desired change to that state, and the interesting quantity is not
either intent alone but the *sign of their interaction*. Two patches that move
the same lines in the same direction compose; two that move them in opposite
directions conflict; two that touch nothing in common are independent and can be
applied in any order. Same three outcomes as ally, rival, neutral, and the same
practical consequence: you can predict what applying one does to the other
without applying it, by comparing intents rather than by executing.

**Where this breaks.** Both analogies rest on a *fixed, human-legible*
coordinate system - file paths and line numbers, cache keys - so "disjoint"
means the same thing today as tomorrow. The parameter space has no such
structure. Its coordinates are learned, they carry no stable meaning, and an
example's gradient changes at every step because it depends on what the model
currently gets wrong. That is why two documents on entirely unrelated subjects
can be strong gradient allies (both push the same low-level machinery, which has
no counterpart in "different files") and why an ally at one checkpoint is a rival
at the next, which no merge-conflict intuition would predict. The cache analogy
also implies contention is symmetric and roughly static; gradient alignment is
symmetric but not static, and its drift over training is a first-class effect
rather than a nuisance.

## TracIn: influence as a sum over checkpoints

**The pair.** TracIn is like **attributing a file's current contents to the
commits that produced it, weighted by how much each commit changed**, and like
**attributing a latency regression to deploys by differencing the metric across
deploy boundaries and assigning each delta to the deploy that spans it**.

**The shared structure.** All three take a final state, walk backwards through
the history that produced it, and decompose the total into per-event
contributions that sum exactly to the whole. The exactness comes from the same
place in each: consecutive states telescope, so the sum of the differences is
the difference of the ends, with nothing left over. That is what makes this a
decomposition rather than a heuristic score, and it is why an attribution that
does not sum to anything should make you suspicious. The deploy analogy carries
the second structural fact too: a deploy that happened during a period of heavy
traffic moves the metric further than the same change deployed at 3am, which is
the learning-rate weight - the same intent applied at a moment of larger
leverage produces a larger effect.

**Where this breaks.** `git blame` is exact because diffs are exact: the
difference between two commits is a literal record of what changed. TracIn's
per-step difference is a *first-order estimate* of a curved quantity, so the
decomposition sums to the total only up to an error that grows with step size.
Blame also uses a different attribution rule - last writer wins - whereas TracIn
credits everyone who moved the value, which is closer to the deploy analogy and
is the correct rule for payment. The deploy analogy breaks on the checkpoint
approximation: sampling a metric every five minutes and assigning the whole
window's change to the sampled deploy is exactly what the checkpoint sum does,
and it is fine when one deploy dominates a window and quietly wrong when three
deploys land in the same window at different rates. If your corpus repeats data
across epochs, that is precisely the "three deploys in one window" case, and the
frequently-repeated sources are being underweighted.

## Making it affordable

**The pair.** Random projection of gradients is like **MinHash signatures for
near-duplicate detection at web scale**, and like **HyperLogLog registers for
distinct counts**.

**The shared structure.** All three throw away the object and keep a randomized
sketch that is sufficient for exactly one query and useless for anything else.
MinHash discards the document and keeps a fixed-size signature from which
Jaccard similarity is recoverable; HyperLogLog discards the elements and keeps a
register array from which cardinality is recoverable; the projection discards the
gradient and keeps a few thousand numbers from which dot products are
recoverable. The sketch size is fixed and does not grow with the object -
HyperLogLog does not care whether you counted a thousand or a billion items, and
the projection does not care whether the model has ten million or a billion
parameters. The error in all three falls like one over the square root of the
sketch size, which is why quadrupling the sketch only halves the error, and why
everyone lands on a few thousand registers or dimensions rather than a hundred
or a million.

The per-source accumulation has a direct counterpart too: sketches are
**mergeable**. Union two HyperLogLog registers and you get the sketch of the
union, exactly. Add two projected gradients and you get the projection of the
sum, exactly. In both cases the merge is lossless and the approximation lives
entirely in the sketching step, which is why bundling ten million sequences into
twelve source vectors costs no accuracy at all.

**Where this breaks.** Two ways, and the second is operationally dangerous.

MinHash and HyperLogLog degrade *gracefully and symmetrically*: a small sketch
gives a noisy estimate of everything. The projection's error is uniform in
absolute terms and therefore wildly non-uniform in relative terms. Strongly
aligned gradient pairs come through clearly and near-orthogonal pairs - which is
most pairs - are pure noise. The correct mental model is not "everything is 1.6%
off", it is "the top of the ranking is real and the tail is a random
permutation."

And every hash sketch in a real system carries its hash seed in its own
serialization, so mixing sketches from different seeds is a schema error that
tooling catches. The gradient index has no such convention by default. Project a
query gradient under a different seed than the source accumulators used and you
get finite, well-ordered, entirely meaningless numbers - a full ranking with no
error, no warning, and no checksum. This is the one place in the analogy where
the software engineering instinct actively misleads: you are used to sketch
mismatches failing loudly, and this one fails silently.

## The reckoning

**The pair.** Shipping an ungraded attribution method is like **a test suite
with full line coverage and no assertions**, and like **a monitoring dashboard
that is green because no probe is checking the thing that broke**.

**The shared structure.** In all three the machinery runs, completes, and
produces confident output, and the output is byte-for-byte indistinguishable
between the working case and the broken case. Nothing errors. Coverage reports
look excellent; the dashboard is green; the ranked source list is plausible and
well-formed. What is missing in each is an independent statement of what the
right answer was, and the absence of that statement is invisible from inside the
system - which is why these failures persist for years rather than minutes. The
fix is structurally the same in all three: introduce an oracle that was
constructed separately from the thing under test, and compare.

The optimizer mismatch deserves its own pair, because it is a specific and
familiar failure: it is like **a benchmark harness that has been measuring the
wrong operation for two years**, and like **a profiler whose sampling model does
not match how the runtime actually schedules work**. In both, the numbers are
internally consistent, reproducible, comparable across runs, and describe
something other than what everyone believes they describe. Nobody notices,
because there is nothing to notice - a wrong measurement of a real thing looks
exactly like a right one. Correcting the mismatch moves every published number,
and it moves them by different amounts, which is what reordering looks like.

**Where this breaks.** Assertions are cheap. You can add them to a test suite
this afternoon, and once added they cover the code permanently. The oracle here
costs real money, exists only below a scale ceiling, and cannot be extended
upward at any price - so unlike a test suite, you cannot achieve coverage even
in principle. That asymmetry is the reason the discipline has to be different in
kind rather than merely more diligent: you are not being asked to test more, you
are being asked to *state the scope within which you tested*, and to keep saying
so when the product runs outside it. The dashboard analogy also implies that
adding the missing probe makes the problem go away. It does not. Adding the
missing probe here tells you the answer for a 10-million-parameter model and
leaves the 561-million-parameter case a hypothesis forever.

## Influence is not entailment

**The pair.** Influence versus entailment is `git blame` versus `grep`, and it
is **a distributed trace versus a log search**.

**The shared structure.** Each pair contains one *causal-history* question and
one *textual-presence* question, both legitimate, both answerable, both
returning a ranked list of candidates in response to the same trigger. `grep`
finds where a string appears; blame finds which change is responsible for the
line being what it is, and the two routinely name different commits - a
reformatting pass owns the blame for a line whose logic was written years
earlier by someone else. A log search finds which service printed the message
you remember; a trace finds which call actually caused the latency, and it is
frequently a service that logged nothing. In both pairs the textual tool is
faster, cheaper, needs no instrumentation, and is far more convincing to a human
looking at the output, because you can see the match with your own eyes. And in
both pairs, using the textual answer to assign responsibility is a known
mistake that experienced engineers have been trained out of.

**Where this breaks.** `grep` and blame read the same artifact and can be
cross-checked exactly against each other, so the disagreement between them is
resolvable by inspection. Influence and entailment require entirely different
instruments at wildly different costs, and there is no inspection that resolves
their disagreement - the influence answer's ground truth is a training run that
did not happen. The trace analogy breaks harder: tracing has a definite ground
truth, because the request really did traverse those services and the spans are
a record of events. Influence is not a record of events. It is an estimate of a
counterfactual, so even a perfect implementation is answering a question about a
world that does not exist, whereas a perfect trace is a description of one that
does.

The consequence for the product is the thing to carry: you can ship blame next
to grep without controversy, because both are exact. Shipping influence next to
verbatim match requires labelling which one is exact and which one is an
estimate with a stated correlation, or the exact one will lend its credibility
to the estimate.

## The ledger inside the run

**The pair.** In-Run Shapley is like **cost-allocation tags emitted by jobs as
they run**, rather than reconstructing spend from invoices afterwards; and like
**distributed tracing spans emitted inline**, rather than reconstructing a
request path from logs after the fact.

**The shared structure.** All three make the same trade and it is a good one.
Instrument the process while it runs, pay a known and modest percentage
overhead, and attribution falls out as a byproduct with complete coverage.
Reconstruct afterwards and you are doing forensics on whatever evidence happened
to survive: sampled checkpoints instead of every step, one query at a time
instead of the whole run, and a reconstruction whose fidelity depends on how
much the process happened to leave behind. The overhead figure - a few percent -
is exactly the number an engineer expects to see quoted for inline
instrumentation, and it is the number that decides whether it ships. The
per-source running totals are counters, they merge, and they are read at the end
of the run like any other accumulated metric.

**Where this breaks.** A tracing span is a *record of an event that occurred*.
It is ground truth about the request path, and if it disagrees with your mental
model, your mental model is wrong. The in-run ledger is not a record. It is a
first-order estimate of a counterfactual quantity, accumulated inline. Inline
instrumentation does not confer ground truth on an estimate; it only makes the
estimate cheap and complete.

The cost-allocation analogy breaks in exactly the place that matters for money,
and it is the same break the coalition unit spent its length on. Tagging each
job with the resources it requested gives you a correct accounting of what was
requested. It does not tell you what the cluster would have cost without that
team, because the control plane, the base nodes, and the redundancy are there
regardless, and two teams whose workloads are near-duplicates each get billed in
full for capacity only one of them required. That gap between "what this actor
consumed on the run that happened" and "what would have been different without
this actor" is precisely the gap between the in-run ledger and the demolition
table, and anybody who has argued about shared-infrastructure chargeback has
already had this fight in another domain.

## What to trust

**The pair.** The instrument hierarchy in this unit is a **metrology lab's
reference weight and the shop-floor scales calibrated against it**, and it is
**a slow exhaustive integration suite with the fast unit tests that stand in for
it in CI**.

**The shared structure.** In both there is one expensive, slow, authoritative
procedure that is run rarely and never in the hot path, and a set of cheap
proxies that run constantly. The proxies have no authority of their own; their
entire value is the measured agreement with the authority, established once and
restated whenever the proxy's result is quoted. Both systems have the same
characteristic failure: the expensive procedure gets run less and less because
it is inconvenient, the proxies drift, and nobody notices until an external
party disagrees with a number that everybody internally trusted. The discipline
in both cases is procedural rather than technical - keep running the reference,
keep the calibration attached to the number, and treat an uncalibrated proxy
result as unpublishable.

**Where this breaks.** Both analogies assume the authority remains available.
You can always reweigh against the reference; you can always run the full
integration suite if you are willing to wait. Here the authority *stops
existing* above a certain model scale, because retraining is the authority and
retraining is what you cannot afford. So the drift between proxy and truth at
production scale is not merely unmeasured through neglect, it is unmeasurable in
principle. That changes what the calibration means: it is not a periodic check
you keep repeating, it is a single measurement taken at small scale that you
then carry, clearly labelled, into a regime where it can never be repeated. No
engineering discipline you already have quite covers that, which is why the
scope statement has to be written down rather than assumed.

## At the bench: the gradient ledger

**The pair.** Building the gradient index is like **building a search index: one
expensive offline pass, then cheap online queries, with the analyzer
configuration stamped into the index**; and like **recording a build's
dependency manifest so the artifact remains interpretable and reproducible
later**.

**The shared structure.** Both are one-time expensive passes producing a compact
artifact that answers many cheap queries. Both are worthless without metadata
describing how they were produced - a search index built with one tokenizer and
queried with another returns garbage, and a build artifact with no manifest
cannot be audited or rebuilt. In both cases the metadata is small, boring, and
the single most important thing in the directory. The gradient index's manifest
entries map cleanly: the projection seed is the analyzer configuration, the
checkpoint list with learning rates is the source revision, and the
optimizer-preconditioning flag is the compiler flags.

**Where this breaks.** A search index built with a mismatched analyzer usually
fails *visibly*: queries return nothing, or return obviously wrong documents,
and somebody notices within a day. A build with a mismatched manifest fails at
build time. A gradient index built with a lost or mismatched projection seed
returns a complete, well-formed, correctly-typed, entirely plausible ranking
that is pure noise, and it will do so indefinitely. There is no checksum, no
empty result set, no analyzer-mismatch exception. This is the strongest reason
the seed belongs in a manifest and not in a comment or a shell history, and it
is the specific failure the capstone beat is testing: not "you will lose data",
but "you will produce a confident wrong answer and have no way to find out."

## What you can now do

**The pair.** Reporting an attribution score with its measured correlation is
like **shipping a profiler with its documented sampling error** rather than one
that prints numbers, and like **publishing a benchmark with its variance and
methodology** rather than a marketing throughput figure.

**The shared structure.** In all three the credibility of a measurement tool
comes entirely from the stated methodology and the stated error, and not at all
from the magnitude or precision of the number. Everyone in this field has seen a
throughput figure with four significant digits and no configuration, and knows
to discard it - not because the number is wrong but because it is
unfalsifiable, and an unfalsifiable number is not evidence regardless of whether
it happens to be true. The same instinct transfers exactly: "source 7 drove this
output" is the marketing throughput figure. "Source 7 ranks first by projected
per-source TracIn, rank correlation 0.71 against leave-one-source-out ground
truth over twelve sources at ten million parameters, sixty held-out subsets" is
the benchmark with a methods section. The second is a claim someone can attack,
and that is what makes it worth more than the first.

**Where this breaks.** A profiler's sampling error can be characterized once in
the lab and then generalizes: it is a property of the sampling mechanism, and it
holds on workloads nobody has run yet. The correlation here does not generalize.
It is a property of *this corpus at this model scale*, and every reason to
expect it to transfer is a hypothesis rather than a derivation - influence
patterns are documented to change with scale, becoming more abstract and more
long-tailed. So the analogy licenses the habit of attaching the error, and it
does not license treating the attached error as a permanent specification of the
tool. Restate the scale each time, and expect a reviewer who knows the field to
ask what happens above it. The honest answer is that nobody knows, including the
people who published the methods.
