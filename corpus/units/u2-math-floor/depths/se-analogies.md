---
unit: u2
depth: se-analogies
---

## Shapes are type signatures

**A matrix is like a function signature, and it is like a schema-mapped
transform.**

`func W(x [n]float32) [m]float32` declares an arity contract that is checked
before any data flows. A schema transform that reads $n$ fields and emits $m$
fields declares the same kind of contract. The shared structure is that in both
cases the shape is the interface, it is verifiable statically, and a mismatch is
a type error caught at the boundary rather than a wrong answer produced quietly
downstream. This is exactly why framework shape errors are the good kind of bug:
they fail loudly at the seam, and once shapes line up all the way through, an
entire class of wiring mistakes is gone.

**Where this breaks:** a Go function can do anything - branch, allocate, call out.
A matrix can only do what linearity permits: no conditionals, no data-dependent
behavior, no bending. Two inputs that are close always produce outputs that are
close, by a fixed factor. The signature analogy tells you about the interface and
tells you nothing about the implementation, and the implementation here is
drastically more constrained than a general function. Reasoning about a matrix as
if it were arbitrary code will make you overestimate what a single layer can do,
which is precisely the mistake that makes "why do we need nonlinearities" feel
like a strange question.

**Rank-deficient projection is like a `SELECT` that drops columns, and it is like
hashing to a smaller keyspace.**

Both take a domain and collapse it many-to-one. In both, the collapse is one-way:
you cannot reconstruct the dropped column from the projected row, and you cannot
recover a key from its hash. Once two distinct inputs share an output, nothing
downstream can distinguish them, no matter how much machinery you add. This is
the right lens for every down-projection in a transformer, and it is why the head
dimension $d_k$ being small is a design decision with real consequences rather
than a memory-saving detail.

**Where this breaks:** a hash is designed to scatter - nearby keys land far
apart, deliberately. A linear projection does the opposite: nearby inputs land
nearby, and the structure that survives is preserved faithfully rather than
destroyed. What is lost is a whole subspace, cleanly and predictably, not
information smeared unrecoverably across the output. And unlike a `SELECT`, which
drops named columns you can enumerate, a projection can drop an arbitrary
direction that corresponds to no coordinate at all - the discarded subspace need
not align with anything you have a name for.

## Composition is multiplication

**Precomputing $AB$ is like fusing a middleware chain into one handler, and it is
like a query planner collapsing nested views into a single scan.**

In all three you have a pipeline of stages, each consuming the previous stage's
output, and you replace the pipeline with a single equivalent stage computed
ahead of time. The shared structure is that composition of pure transformations
is itself a transformation of the same kind, so fusion is always possible in
principle and is purely a performance decision. This is also why the associativity
of matrix multiplication is worth caring about at all: choosing where to put the
parentheses in $ABx$ is exactly the query planner's join-ordering problem, and it
carries the same order-of-magnitude stakes. LoRA is this optimization applied
deliberately, keeping a low-rank update factored so you pay two thin multiplies
instead of one fat one.

**Where this breaks:** middleware can short-circuit, mutate shared state, or fail.
Matrix composition has no side effects and no early exit, so the fusion is
unconditionally sound. More important in the other direction: a query planner may
reorder joins freely because relational operators commute, and matrices do not.
Reparenthesizing $ABx$ is always safe; swapping $A$ and $B$ almost never is. If
you carry the planner analogy too far you will conclude that the order of weight
matrices is an optimization detail, which is U2-M3 and will silently produce a
different model.

**Stacking linear layers with nothing between them is like composing a chain of
pure `map` steps, and it is like chaining a sequence of affine CSS transforms.**

The browser does not apply your six transforms in sequence at render time. It
multiplies them into one matrix and applies it once, because the composite is the
same kind of object as the parts. Six `map` calls over an immutable stream fuse
into one pass for the same reason. The shared structure is closure under
composition: the pipeline collapses because the stage type is closed under
chaining.

**Where this breaks:** the collapse is a straight win for CSS and for `map`
fusion, but it is fatal for a neural network. Twenty stacked linear layers have
exactly the expressive power of one, which means depth bought you nothing but
parameters and latency. Everywhere else in your career, "this pipeline collapses"
was good news. Here it is the failure mode that nonlinearities exist to prevent.

## The dot product is the whole game

**A dot product is like a popcount over ANDed feature bitsets, and it is like a
weighted linear scoring function in a ranking service.**

All three reduce two structured objects to one scalar that says how much they
agree, coordinate by coordinate, with no branching and no lookup. The bitset
version is the cleanest intuition: line up two feature vectors, count where both
fire. The ranking-service version adds the weighting: some coordinates count more
than others. A dot product is both at once, over continuous values. The shared
structure is elementwise agreement summed into a single comparable number, which
is the only comparison primitive this book ever uses.

**Where this breaks:** popcount is unsigned and bounded by the vector width. A
dot product is signed and unbounded, so it distinguishes "disagrees" from
"unrelated" - a distinction bitsets cannot express - and its magnitude conflates
two different things. A long vector that agrees weakly can score identically to a
short vector that agrees perfectly. If you carry the popcount intuition you will
read a high attention score as strong semantic match when it may just be a
high-norm key. That conflation is real and load-bearing: normalizing it away is
what cosine similarity does, and the fact that attention deliberately does not
normalize it away is part of why key and query norms matter.

**A matrix-vector product is like fanning a request out to every backend in a
pool and collecting one number from each, and it is like evaluating a whole
ruleset against one input record.**

Each row of the matrix is one backend, or one rule, holding its own reference
pattern. The input goes to all of them; each returns a score. The shared
structure is a broadcast-and-score fan-out with no coordination between the
scorers, which is exactly why this maps onto GPU hardware without any cleverness.

**Where this breaks:** backends and rules have identity and can be inspected,
named, and reasoned about individually. Rows of a learned weight matrix are
mostly not individually interpretable - the useful directions are frequently
combinations of rows rather than rows themselves, and a single row can participate
in several unrelated computations at once. Expecting to open up a weight matrix
and find named rules is the fully general version of the mistake that
interpretability research spends its time correcting.

## Softmax: why exponentials

**Softmax is like weighted random routing across a backend pool, and it is like
proportional share allocation in a scheduler.**

Both take a set of raw scores of arbitrary scale and convert them into a
normalized allocation that sums to one whole, then use that allocation to pick.
The shared structure is score-to-share normalization followed by sampling, and
the reason both exist is the same: you want the best option favored without the
others starved to exactly zero, because exact zero is brittle.

**Where this breaks:** load balancer weights are configuration - static, set by
an operator, meaningful in absolute terms. Softmax weights are recomputed from
scratch at every token from data, and because of the exponential, a small change
in a logit produces a disproportionate change in the allocation. Worse for the
analogy: LB weights genuinely encode capacity, a real property of the world you
can go measure. Softmax outputs encode nothing but training-corpus frequency.
Treating them as a measured property of anything external is M2, and the routing
analogy is one of the paths engineers take to get there.

**Exponentiating logits is like working in decibels, and it is like exponential
backoff.**

All three share one property: a constant additive change produces a constant
multiplicative effect. Add 3 dB and the power doubles, whether you were at 10 dB
or 100. Add one retry and the wait doubles, whether it is the second attempt or
the ninth. Add 1 to a logit's lead and its share multiplies by $e$, whether the
logits were 1 and 0 or 101 and 100. This is the exact property the derivation
requires - differences must map to ratios - and it is why no polynomial can
substitute.

**Where this breaks:** decibels and backoff have a meaningful absolute reference
point (a reference power level, a base delay). Logits have none. There is no
"neutral" logit, no zero point, and the entire distribution is invariant to
adding a constant. So while the growth behavior transfers exactly, any intuition
about a logit's absolute value meaning something does not. A logit of $-40$ tells
you nothing on its own; only its gap to the others does.

## Temperature: one knob, same evidence

**Temperature is like gamma correction on an already-rendered frame, and it is
like a display-time log-scale toggle on a metrics dashboard.**

Both are applied strictly downstream of the computation that produced the data,
both reshape how the values are distributed for consumption, and neither can
create or destroy the underlying values. Flipping your dashboard from linear to
log makes small spikes visible and compresses the tall ones; it does not change
what the collector recorded. The shared structure is a presentation-layer
transform over a fixed upstream result. That placement is the whole point:
temperature usually lives in the serving layer, not the model, and in many stacks
you can change it without touching the model at all.

**Where this breaks:** gamma correction can genuinely clip - crush blacks to
identical zero and lose distinctions permanently. Temperature does not lose
information in the same way, because the logits still exist upstream and any
temperature can be applied to them. And the dashboard analogy suggests temperature
is only about viewing, which understates it: the transformed distribution is
actually sampled from, so it changes which token comes out, not just how the
numbers look. It is a presentation transform whose output is then acted on.

**"Set temperature to 0 to see what it really thinks" is like setting a log level
to ERROR and concluding the service only ever had errors.**

Both mistake a filter setting for a view of ground truth. The paired case:
querying a database with `LIMIT 1` and concluding there was only one matching row.
The shared structure is a consumer-side narrowing being read as a property of the
producer. Argmax sampling is `LIMIT 1 ORDER BY logit DESC`. It shows you the top
row. It does not tell you the row was the only one, or that it was right.

**Where this breaks:** the log-level and `LIMIT` cases involve information you
could have retrieved by changing the setting and re-querying. Temperature is
better than that - the full distribution is available simultaneously, at every
temperature, from the same single forward pass, because temperature is a pure
function of logits you already have. No re-run is needed. If you want to know what
the model "thinks", read the logits, and temperature never enters the question.

## From derivative to gradient

**Gradient descent is like a PID controller, and it is like an autoscaler
adjusting replica count from an observed error signal.**

All three operate with no global model of the system. Each observes a local
signal, applies a correction proportional to it, and repeats. Nobody solves for
the answer; everybody converges to it by iteration. The shared structure is
local-measurement to proportional-correction with a gain parameter, and the gain
parameter has the same failure modes everywhere: too low and you converge
uselessly slowly, too high and you oscillate or diverge. Every intuition you have
about tuning autoscaler aggressiveness transfers directly to tuning a learning
rate, including that the right value depends on the shape of the response and
cannot be picked from first principles.

**Where this breaks:** a PID controller has a setpoint. You know the target
temperature, the target queue depth, the target replica count, and error is
defined as distance from it. Gradient descent has no setpoint. There is no known
target loss value, no way to measure how far you are from wherever you are going,
and the only thing available is the local slope. This is the difference between
control and search, and it is why "is training done?" is a judgment call rather
than a predicate. It is also why the analogy quietly smuggles in M5: a controller
converges to *the* setpoint, and if you expect that, you will expect training to
converge to *the* minimum.

**A gradient is like a distributed profiler's per-service attribution of end-to-end
latency, and it is like a spreadsheet's sensitivity analysis row.**

Both answer "if I improve this one component by a unit, how much does the
top-line number move", for every component at once, from one measurement pass.
That parallel-attribution-in-one-pass structure is exactly what reverse-mode
autodiff provides, and the cost profile matches: one backward sweep gives you all
the partials, the same way one traced request gives you the whole waterfall.

**Where this breaks:** profiler attributions are additive and roughly stable -
a service's share of latency does not change much because you looked at it.
Gradients are valid only infinitesimally and only at the current point. Take a
step of any real size and every partial changes. Reading a gradient as "this
parameter is responsible for this much of the loss" is a category error that the
profiler analogy invites; it says only "the loss is locally this sensitive to this
parameter, right here, right now".

## The chain rule

**Chaining local derivatives is like unit conversion through a chain of exchange
rates, and it is like composing per-hop multipliers in a latency budget.**

Dollars per euro times euros per yen gives dollars per yen. Loss per activation
times activation per pre-activation times pre-activation per weight gives loss
per weight. The intermediate units cancel in both cases, and the cancellation is
your correctness check: if the units do not cancel cleanly along the path, you
have dropped or double-counted a stage. This is the single most useful habit to
carry into reading a backward pass, and it generalizes into shape-checking when
the quantities become tensors - the inner dimensions cancel the way the units do.

**Where this breaks:** exchange rates are constants for the duration of the
calculation. Local derivatives are functions of the current activations, so every
"rate" in the chain is recomputed on every training step from the values that
just came out of the forward pass. There is no fixed conversion table. That
dependence is the entire reason the forward pass must be retained, and it is why
"just cache the gradients" is not a thing.

**Needing the forward pass before the backward pass is like a two-phase commit
requiring the prepare log, and it is like a build system keeping intermediate
object files for the link step.**

In all three, phase two is not computable without state that only phase one can
produce, so that state must be held live across the boundary. The shared structure
is a mandatory staged dependency with a memory cost proportional to the size of
phase one. This is the honest answer to why training a model needs several times
the memory of running it, and it is the thing gradient checkpointing trades
against: drop some intermediates, recompute them during the backward pass, pay
compute to save memory. That is an ordinary space-time tradeoff and it behaves
like one.

**Where this breaks:** a prepare log is authoritative and must be durable - losing
it loses correctness with no recovery. Activations are pure functions of the
input and the weights, so they are always recomputable from scratch. Nothing is
lost by dropping them except time. Treating activation memory as sacred state
rather than as a cache is what makes checkpointing look like a dangerous hack
instead of the routine optimization it is.

## Loss surfaces and why SGD works anyway

**Many equally good minima is like the many valid outputs of a topological sort,
and it is like two builds of the same source producing different but functionally
equivalent binaries.**

In all three there is an enormous equivalence class of acceptable answers and the
algorithm returns an arbitrary member of it, selected by incidental factors - hash
iteration order, timestamps, the random seed. Asking "why did I get a different
one this time" has the same answer every time: because you never asked for a
specific one, and nothing in the process was ever going to give you a specific
one. The shared structure is an underdetermined problem with a huge solution set
and a nondeterministic selector.

**Where this breaks:** topological orderings are exactly equivalent under a
formal criterion, and you can prove any two are interchangeable. Low-loss weight
regions are only approximately equivalent - they agree closely on the training
distribution and can differ measurably out of distribution, which is why ensembles
of independently seeded models beat any single member. And reproducible builds
exist precisely to eliminate the nondeterminism, which is the opposite of the
situation here: nobody wants seed-identical weights, and if you got them you
should go looking for the bug in your seeding. That inversion is the part of the
analogy worth holding, because every engineering instinct you have says
nondeterminism is a defect to be stamped out.

**High dimensions making local minima rare is like a distributed system with many
independent failure-recovery paths, and it is like a graph with high edge
connectivity.**

To be stuck, every escape route must be blocked simultaneously. Add routes and
the probability of total blockage falls off a cliff. The shared structure is that
"trapped" is a conjunction over all directions, so each added dimension multiplies
the probability of being trapped by roughly one half. Your intuition that
redundancy makes total failure exponentially unlikely is exactly the right
intuition, applied to escape directions instead of replicas.

**Where this breaks:** in a distributed system the failure paths are engineered
to be independent, and correlated failure is the thing that actually takes you
down. Curvature directions on a loss surface are not independent - the Hessian has
strong structure, with a small number of large eigenvalues and a long tail near
zero. The result is that instead of hard traps, you get vast nearly-flat plateaus
where every direction is *almost* blocked. Slow progress from near-zero gradients,
not entrapment, is what actually costs you training time, and it is the problem
momentum and adaptive optimizers were built for.

### The floor is not zero

**Irreducible loss is like a compression ratio bounded by the source's entropy,
and it is like a latency floor set by speed-of-light round trip.**

Both are hard bounds imposed by the problem rather than the implementation. No
compressor beats the entropy of the source; no request beats the propagation
delay. In both cases, effort spent trying to cross the bound is wasted, and the
meaningful engineering question is how close to the bound you are, not how close
to zero. Reading a training curve should feel like reading a compression ratio: a
number that is good relative to a floor, never good relative to zero.

**Where this breaks:** the compression analogy is not an analogy at all - it is
the identical mathematics. Cross-entropy loss in nats, divided by $\ln 2$, is
literally the bits per token a compressor built from the model would achieve, and
the irreducible term is literally the entropy of the source. Treating it as a
loose metaphor undersells it. The latency analogy, by contrast, is loose in an
instructive way: a speed-of-light floor moves if you move the datacenter, so it is
a floor for a fixed deployment rather than for the problem. The entropy of
language does not move for anyone, at any budget.
