---
unit: u4
depth: se-analogies
---

Every analogy here comes as a pair with the shared structure named, and every
pair states where it breaks. An analogy you cannot break is one you will
over-apply.

## The block in one equation

**Pair.** A transformer block is like an HTTP middleware layer, AND like a
stage in a CPU pipeline.

**Shared structure.** Fixed interface in, identical interface out, so instances
compose by stacking with no adapter code. The uniform signature is what makes
"32 of these" a configuration value rather than an architecture.

**Where the middleware half breaks.** Middleware can short-circuit: an auth
layer returns 401 and nothing downstream runs. A block always runs, always
consumes its whole input, always emits the same shape. There is no early exit
and no conditional dispatch anywhere in a dense transformer - the control flow
is a straight line, and every "decision" the model appears to make is a
continuous weighting inside a matmul. (Unit 7's mixture-of-experts is the first
place anything resembling dispatch appears, and even there every token goes
through the same number of matmuls.)

**Where the pipeline half breaks.** CPU pipeline stages pass a value forward;
stage $k+1$ consumes stage $k$'s output and stage $k$'s input is gone. Blocks
add to a shared buffer that carries everything written so far. This is exactly
the difference that section 6 is about, and getting it wrong is the single most
consequential error in this unit - so treat the pipeline analogy as good for
"why the shapes match" and useless for "what flows between blocks."

## Multi-head attention: many subspaces, one parameter budget

**Pair.** Multi-head attention is like running several `GROUP BY` clauses over
one table in a single scan, AND like a struct-of-arrays layout where different
consumers read different field ranges of the same record.

**Shared structure.** One pass over the data yields several independent
aggregations, because the aggregations key on different columns. Nothing is
duplicated; the columns were already there and each consumer reads its own
slice. That is precisely what `reshape(n, h, d_k)` does to a projection: it
reinterprets one tensor's columns as $h$ independent field ranges.

**Where the GROUP BY half breaks.** SQL aggregations partition *rows*. Heads
partition *columns* - the feature space of every token. Every head still sees
every position; what differs is which 64 of the 4096 dimensions it forms its
similarity on. If you think of heads as sharding the sequence, you will predict
that heads see disjoint tokens, and they do not.

**Where the struct-of-arrays half breaks.** In SoA the field ranges are
declared, named, and disjoint by construction. Head subspaces are learned, have
no names, and the *write* subspaces (the row-blocks of $W^O$) routinely overlap
with other heads' and other layers' - the stream is in superposition. So the
layout intuition is right about the mechanics of slicing and wrong about
isolation guarantees.

## MQA and GQA: the same equation with fewer key-value heads

**Pair.** GQA is like a pool of workers sharing one read replica instead of
each having its own, AND like interning duplicate strings so many references
point at one buffer.

**Shared structure.** Both cut *bytes resident and bytes moved* without
reducing the number of operations performed against those bytes. Thirty-two
workers still each run their full query against the shared replica. Every
reference to an interned string still does its own comparison.

**Where the read-replica half breaks.** A read replica returns byte-identical
data to every worker, so sharing is semantically free. Sharing key/value heads
is *not* free: the shared $K$ imposes one similarity geometry on all the query
heads in the group, so GQA computes a different function than MHA and moves
quality. This is the exact reason $g = 8$ beats $g = 1$: the geometry is a
shared resource with contention, not a cache.

**Where the interning half breaks.** Interning is applied post hoc to values
that were already equal. GQA *forces* equality that the model did not choose -
it is a constraint imposed at architecture time and trained around, not a
deduplication of coincidentally identical tensors. Reaching for the interning
picture also invites the assumption that the saving is proportional to how much
duplication happened to exist; it is not, it is exactly $h/g$ by construction.

**The trap both analogies exist to defuse.** In almost every systems context,
"fewer of X" implies "less work." Here it implies less state. The FLOP count is
identical across MHA, GQA and MQA (U4M3), and this is the single most common
error engineers make on this material precisely because their instincts are
otherwise good.

## The MLP block: where the parameters actually live

**Pair.** The up-project/nonlinearity/down-project sandwich is like decompress -
transform - recompress, AND like widening to a sparse feature space before a
linear classifier, the way you would one-hot-encode categoricals before a
regression.

**Shared structure.** Temporary expansion buys separability. In the wide
representation, things that were entangled become individually addressable, you
act on them cheaply there, and then you fold back to the compact form for
transport.

**Where the compression half breaks.** Decompression is lossless and
invertible: the wide form contains exactly the narrow form's information. The
up-projection is $d \to 4d$ but its output lies on a $d$-dimensional manifold
before the nonlinearity - no information is created. The bend is what does the
work, and the analogy has no counterpart for it. If you carry the compression
picture, you will predict that a wider $d_{ff}$ preserves more, when what it
actually does is give the nonlinearity more places to bend.

**Where the one-hot half breaks.** One-hot features are orthogonal and
interpretable by construction - column 7 is `country=FR` and nothing else. MLP
hidden units are neither: they are polysemantic, in superposition, and there are
more features than units. The intuition "wide sparse layer, then linear
readout" is right about the shape of the computation and wrong about the
legibility of the intermediate.

## Key-value memory, and where that framing breaks

**Pair.** The MLP is like a vector-similarity index (embed the query, score
against every stored vector, blend the top matches), AND like a Bloom filter
(probabilistic membership over a space too small to hold the members).

**Shared structure.** Neither has rows. Both answer by similarity or by
overlapping bit patterns rather than by address, both accept collisions as the
price of density, and in both the storage is smaller than the thing being
represented.

**Where the vector-index half breaks.** A vector index has a top-$k$ cutoff and
a distance threshold: you can ask it "is anything close?" and it can answer no.
The MLP has no cutoff and no reject option. Every state produces a weighted sum
over all $d_{ff}$ values, so a query matching nothing well still returns a
confident blend of near misses. That missing reject path is where hallucination
comes from, and it is why "add a similarity threshold" is not an available fix
inside the layer.

**Where the Bloom filter half breaks.** Bloom filters have one-sided error: a
"yes" may be wrong, a "no" never is. The MLP errs in both directions, and its
false positives are *fluent* - not a spurious bit but a well-formed factual
claim. Also, a Bloom filter's parameters are chosen for a target error rate;
the MLP's superposition density is whatever training found worth trading.

**The correction both pairs are pointing at (M4).** Parameters define a
function, not a store. The regions where training saw many near-identical
examples are sharp bumps you may call facts. Everywhere else the function
interpolates. Retrieval and hallucination are the same evaluation of the same
map at different distances from a bump.

## The residual stream is a workspace, not a shortcut

**Pair.** The residual stream is like an append-only event log that every
service reads and appends to, AND like a shared mutable scratch buffer that
each stage `+=` into.

**Shared structure.** State is accumulated, never replaced. Any consumer can
read what any earlier producer wrote. Producers are decoupled: they agree on the
buffer, not on each other.

**Where the event-log half breaks.** Log entries are discrete, addressed, and
typed - you can name the event, filter by producer, and replay a subset. Stream
writes are dense vectors added into shared, unnamed, overlapping subspaces. You
cannot separate "which block wrote this direction" without doing linear algebra
on the weights (and unit 4's extension x2 is exactly that work). The log picture
also implies ordering guarantees; the stream's arithmetic is a sum, which
commutes, even though the *arguments* are computed in order.

**Where the shared-buffer half breaks.** A shared mutable buffer invites the
question "what if two stages write the same address?", and in software that is a
race to be prevented. Here it is the normal case and it is *deliberate* -
sixty-four writers into 4096 dimensions cannot be disjoint, so they share
directions and tolerate interference. Contention is a design feature, not a bug
to lock around.

**The concrete engineering prediction to keep (M8).** Deleting a middle block
removes 2 terms from a sum of 64 - a few percent of quality, which is why
layer-pruning and early-exit are practical techniques. Replacing a `+=` with an
`=` anywhere in the stack destroys the model, because that discards every prior
append. If your mental model does not make those two predictions differently,
it is the pipeline model, and it is wrong.

## Normalization conditions the optimization; it does not save your floats

**Pair.** LayerNorm is like automatic gain control on an audio input stage,
AND like feature standardization before fitting a regression.

**Shared structure.** Both discard absolute magnitude and keep the pattern, so
that the stage downstream always operates at the same point on its curve
regardless of how loud or how large-scaled the input happened to be. Both make
the downstream stage's learned constants portable across inputs.

**Where the AGC half breaks.** AGC is protective: it exists partly so the next
stage does not clip. That protective reading is exactly M7 and must be
discarded. LayerNorm is not preventing anything from saturating; a network run
in fp32 with no normalization does not overflow, it produces garbage because
the downstream weights are being fed something in the wrong units. The
normalizer is a calibration contract, not a limiter.

**Where the feature-standardization half breaks.** Standardization is
preprocessing: done once, outside the model, using statistics from the dataset.
LayerNorm is *inside* the differentiated computation and uses statistics of the
single vector in front of it. That means it also reshapes the gradient - its
Jacobian projects out the component along the current activation, so the
optimizer cannot get credit for making a representation merely larger. A
preprocessing step has no such effect, and this effect is the main reason
LayerNorm is there.

**One line to keep.** $\mathrm{LN}(a \cdot u) = \mathrm{LN}(u)$ exactly, for any
positive $a$. Scale a preceding weight matrix by 1000 and nothing downstream
changes. That is not a property any float-safety device would have as its
defining feature.

## Pre-LN and post-LN

**Pair.** Pre-LN versus post-LN is like validating a request at the handler
boundary versus re-canonicalizing the shared record after every write, AND like
sanitizing each consumer's copy of a message versus rewriting the message on the
bus.

**Shared structure.** Identical operation, two placements relative to shared
state: on the branch that a consumer reads, or on the shared state itself. In
both software cases the question "does the shared thing get modified?" is the
whole distinction, and it is the whole distinction here too.

**Where both halves break, and it is the important part.** In software, either
placement preserves the data - you get the same bytes, and the choice is about
where the cost and the coupling sit. Here, placement decides whether the model
is *trainable*. Post-LN puts the normalizer on the shared path, which severs
the unbroken identity route from the output back to the embedding; that route
is what the training process was using, so post-LN needs learning-rate warmup
and becomes hard to train past a couple of dozen layers. No software analogy
has a counterpart for "modifying the shared state on the way through changes
whether the system can be built at all," so stop the analogy at "which one
touches the shared thing" and take the consequences from the architecture.

**The recall trick, in engineering terms.** A pre-LN model ships one extra
normalizer at the very end, immediately before the unembedding (`ln_f` in
GPT-2, `model.norm` in Llama), because nothing else ever normalizes the shared
state. Post-LN needs none - its last operation already was one. If you can find
a dangling final norm in a checkpoint's key list, it is pre-LN (U4M4).

## Depth: composition through the stream

**Pair.** Depth is like a Unix pipeline where each stage can only work with
what the previous stage emitted, AND like logic depth in a hardware circuit,
where a result needing $k$ dependent gate evaluations takes $k$ gate delays no
matter how many gates you instantiate.

**Shared structure.** Sequential dependency is not purchasable with
parallelism. If step B's *input* is step A's *output*, no amount of widening
produces B without A. That is exactly why one attention layer cannot do
induction: the second head's keys must be a function of what the first head
wrote, and in a one-layer model there is nothing written yet.

**Where the pipeline half breaks.** `a | b | c` passes the whole payload
forward and each stage may replace it entirely. Blocks add into a shared buffer
and cannot delete, so block 20 has access to what block 2 wrote directly, not
only through block 19. Transformer depth is a DAG over a shared bus, not a
chain - which is why the middle is redundant in a way a Unix pipeline never is.

**Where the circuit-depth half breaks.** Logic depth is a hard combinatorial
bound: the function provably cannot be computed in fewer levels. Transformer
depth is soft - a model can often approximate a $k$-stage computation in fewer
layers, badly, by memorizing common cases or by using the attention pattern
itself to carry some of the composition. So "needs $k$ layers" here means
"cannot do it robustly and generally in fewer," not "provably impossible." The
induction case is the cleanest known example where the boundary is sharp.

**The practical consequence.** At a fixed parameter budget you are choosing
stages against per-stage capacity, and the industry's answer has been remarkably
stable: width-to-depth around 128 (GPT-3 at $12288/96$, Llama-3-8B at
$4096/32$). Very deep-and-thin buys stages it cannot fill; very shallow-and-wide
buys capacity it cannot compose.

## What to carry forward

The four analogies worth keeping, each with its leash: heads are a
struct-of-arrays *column* slicing (not sharding of rows); GQA is a shared read
replica that saves haulage (not work); the residual stream is an append-only
log with unnamed overlapping fields (not a pipeline); normalization is a
calibration contract (not a limiter). Each of those four analogies, taken past
its break point, produces one of this unit's target misconceptions - which is
why the break point is the part to memorize.
