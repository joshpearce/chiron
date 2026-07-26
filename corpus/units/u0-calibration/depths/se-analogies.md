# Software-engineering analogies: u0

Every analogy here comes as a pair with the shared structure named, and every
pair states where it breaks. An analogy you cannot see the edge of is a
misconception with better marketing.

## How this unit works

**Pair.** Calibration is like a profiler's warmup run, AND it is like the
capability negotiation in a protocol handshake. The shared structure: a cheap
measurement pass whose only product is a configuration for the expensive pass
that follows. Nothing you produce during it is the deliverable.

**Where this breaks:** a handshake fails closed if the peer answers wrong, and a
profiler run that misses hot paths silently degrades the optimization. Here,
wrong answers are the useful signal, not a failure mode - the measurement wants
your errors, and answering conservatively to look good makes the negotiated
configuration worse for you.

## The shape contract

**Pair.** A tensor shape is like a function type signature, AND it is like a
Protocol Buffers schema on a wire format. The shared structure in both: a
declaration of what a value is that can be checked before any value exists, so
that mismatches are caught at the join rather than surfacing as garbage
downstream. `(n, d_model)` is `List[Vector[d_model]]` of length `n` in the same
sense that a `.proto` file tells you a field is a `repeated float` of a certain
length.

**Where this breaks:** a type signature constrains the values a function
accepts, but a tensor shape does not constrain values at all - two tensors with
identical shapes can be an embedding table and a set of attention logits, and
nothing in the type system will notice if you swap them. The bugs that survive
shape checking are exactly the ones where the shapes agree and the semantics do
not. Shapes are structural typing with no nominal component, which is why
transformer code is full of comment-only "documentation" of what a dimension
means. A protobuf schema at least names its fields; a shape does not name its
axes.

**Pair.** The row-vs-column convention is like endianness, AND it is like the
`f(x)` versus `x |> f` split between functional traditions. The shared
structure: a global orientation choice that is invisible while you stay inside
one system and corrupts everything the moment you cross a boundary without
converting.

**Where this breaks:** endianness is checkable at runtime and detectable from a
magic number, and pipeline syntax is enforced by a parser. The vector convention
is announced nowhere, checked by nothing, and frequently switches between a
paper's equations and that same paper's reference implementation. There is no
byte-order mark. You infer the convention from the shapes in the equations, and
if you infer wrong, everything still runs - it just computes a different
function.

## Dot products: alignment, not distance

**Pair.** A dot product is like a hash-based similarity score (MinHash over two
sets), AND it is like a weighted-sum relevance score in a search ranker such as
BM25. The shared structure across both: many-dimensional evidence collapsed into
one comparable scalar, cheaply, so that downstream code can rank rather than
inspect. All three primitives exist because comparing full objects pairwise is
too expensive and a single number is enough to sort by.

**Where this breaks:** MinHash estimates a bounded, normalized quantity - a
Jaccard similarity in $[0,1]$ - and BM25 is deliberately length-normalized so
long documents cannot win by being long. The dot product is neither bounded nor
normalized. Its magnitude scales with the lengths of both inputs, so a vector
can score high purely by being big. Importing the intuition "higher score means
more similar" from ranking systems is precisely the error the canon section
refutes. The nearest honest ranking analogy is BM25 with document-length
normalization deliberately switched off.

**Pair.** Cosine-vs-dot is like comparing normalized rates versus raw counts in
a metrics pipeline, AND like comparing a unit vector's direction versus a
velocity in a physics engine. Shared structure: two quantities that agree
exactly when magnitudes are held at one, and diverge as soon as they are not.

**Where this breaks:** in a metrics pipeline you can always recover the rate
from the count and the denominator, so the two views are interconvertible per
data point. Inside a model, the magnitude carries load-bearing signal that you
cannot simply divide out and still have the same network - some components
encode confidence or salience in norm. Normalizing everything to unit length
would not be "cleaning up the metric", it would be deleting a channel the model
trained itself to use.

## Matrix multiply: composition, not a loop

**Pair.** Matrix multiplication is like function composition, AND it is like a
Unix pipeline. The shared structure that makes both apt: a chain of stages, each
declaring what it consumes and produces, where the chain type-checks stage by
stage, order is semantically load-bearing, and the whole chain can be replaced
by one equivalent stage. `A |> B |> C` is a single transformation with a single
signature, and so is `ABC`.

**Where this breaks:** pipeline stages and general functions can be stateful,
branching, and non-associative in the presence of side effects, and a `grep` in
a pipeline can drop records so the shape of the output depends on the data.
Matrix stages can do none of that. Every stage is total, deterministic,
side-effect free, and shape-determined independent of the values flowing
through, which is exactly why the composite can be precomputed into a single
matrix and why the association can be regrouped for cost. It is also the
limitation: a linear stage cannot branch on its input, and a chain of them
collapses to one stage. The entire reason transformers interleave nonlinearities
is that a pure pipeline of these particular stages is no more expressive than
one of them.

**Pair.** Reassociating $(AB)C$ into $A(BC)$ for cost is like query-plan
reordering in a SQL optimizer, AND like fusing map stages in a stream processor.
Shared structure: an algebraic identity guaranteeing the result is unchanged,
exploited purely to move where the work happens, with cost differences of orders
of magnitude on the same logical operation.

**Where this breaks:** a SQL optimizer reorders based on cardinality statistics
it may estimate wrong, and a bad plan is a correctness-preserving performance
regression that you find in production. Matrix reassociation cost is exactly
computable from shapes alone, with no statistics and no estimation - the optimal
association is decidable up front. The floating-point caveat is the real edge:
reassociation is exactly cost-neutral in real arithmetic but changes rounding,
so `(AB)C` and `A(BC)` can differ in the last bits. Nothing in an optimizer's
guarantees covers that, and it is why bit-exact reproducibility across kernel
implementations is not something you get for free.

## Gradients and expectations

**Pair.** A gradient is like a full profiler report attributing cost to every
line, AND like the per-parameter sensitivity output of a chaos-engineering
sweep. Shared structure across both: rather than one aggregate number, you get a
value per component, telling you how much the global metric would move if that
one component changed while everything else held still. And in all three cases
the response is to act on every component at once, in proportion.

**Where this breaks:** a profiler measures what actually happened over a whole
run, and a chaos sweep measures a large perturbation. A gradient is an
infinitesimal, local, first-order estimate that is only valid in a small
neighborhood of the current weights, and it says nothing about what happens if
you change a weight a lot. Attribution from a profiler stays roughly true as you
edit the code; a gradient goes stale the instant you take the step. That
staleness is the reason learning rates exist and the reason they must be small.

**Pair.** Backpropagation's chain rule is like reverse-mode dependency
resolution in a build system, AND like distributed trace propagation carrying a
context backward through a call graph. Shared structure: a DAG traversed in
reverse topological order, where each node combines what its consumers hand back
with a purely local rule, and no node needs global knowledge.

**Where this breaks:** a build system's reverse pass is about invalidation - a
boolean, is this stale - and can prune subtrees entirely. Backprop's reverse
pass is arithmetic, and it visits every node with a nonzero contribution because
gradients multiply rather than short-circuit. There is no equivalent of "this
target is up to date, skip it". That is the structural reason a backward pass
costs roughly two forward passes and cannot be incrementally cached across
steps.

**Pair.** The expectation-versus-batch-mean distinction is like the difference
between a true population statistic and a sampled metric, AND like the
difference between an SLO defined over all requests and the p99 computed from a
sampled subset of them. Shared structure: the quantity you actually care about
is defined over a population you cannot enumerate, so you estimate it from a
sample, and every property of the estimate - bias, variance, how it improves
with sample size - is a property of the sampling and not of the thing itself.

**Where this breaks:** in observability you can usually turn sampling off and
measure the population exactly, at a cost. Here you cannot, in principle - the
population is "all text that could exist", and no amount of budget makes it
enumerable. The noise floor is permanent. The genuinely alien part, which has no
counterpart in monitoring, is that the sampling noise turns out to be
load-bearing: a training run using the exact population gradient would find
worse solutions than one using noisy batch estimates. There is no observability
system where sampling error improves the outcome.
