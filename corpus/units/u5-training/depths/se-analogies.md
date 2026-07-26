---
unit: u5
depth: se-analogies
---

Every analogy below comes in an aligned pair with the shared structure named,
and every one states where it breaks. An analogy you cannot see the edge of is
worse than none.

## Cross-entropy: turning a distribution into one number

**Cross-entropy is like a log-scale SLO error budget, AND it is like the
score function in a well-designed prediction market.** The shared structure:
both convert a probabilistic claim into a scalar cost that is asymmetric in
confidence. In an error budget, the cost of an incident is not linear in the
outage; the further you go past the objective the more disproportionate the
consequence. In a prediction market, a trader who put everything on the
non-occurring outcome is wiped out, while a trader who was merely correct
collects a bounded return. Cross-entropy has exactly this shape: bounded reward
for certainty that pays off, unbounded penalty for certainty that does not. That
asymmetry is what produces calibration rather than mere accuracy, in all three
systems.

**Where this breaks:** an error budget is a *policy* that humans chose and could
change; a prediction market's scoring rule is one of several proper scoring rules
you could pick. Cross-entropy is not a policy choice. It is maximum likelihood
written out, and any other reasonable choice you make either fails to be
differentiable or fails to be a proper scoring rule. Also, budgets and markets
are consumed by an agent that reasons about them. Nothing in the model reasons
about the loss; it is a number the optimizer reads, and the model never sees it.

**Teacher forcing is like replaying a production request log against a new
service version, AND it is like running a test suite where every assertion has
the expected value hardcoded.** The shared structure: each step is evaluated
against ground truth captured beforehand, independently, so the steps do not
compound and can be evaluated in parallel. This is why one document yields
thousands of independent gradients from a single pass, and why pretraining
parallelizes as well as it does.

**Where this breaks:** hard. A replayed request log has the property that the
service under test *also* runs in exactly this mode in production. The language
model does not. In production each token is generated from the model's own
previous tokens, not from ground truth, so training never once rehearses the
regime the model actually operates in. The correct systems analogy for
generation is not log replay, it is a service consuming its own output as input
with no correction, and there is no phase of pretraining that tests that.

## Perplexity, and the floor that is not zero

**The irreducible loss floor is like the speed of light in your latency budget,
AND it is like the theoretical minimum of a lossless compression ratio.** The
shared structure: in all three, a hard bound exists that no amount of
engineering effort crosses, and the useful engineering question is your distance
to the bound rather than your distance to zero. You do not file a bug that a
cross-continental round trip takes 60ms; you compare against the 40ms that
physics permits. You do not expect gzip to compress random bytes; you compare
against the entropy of the source. Loss works identically: 2.1 nats is not "2.1
worse than perfect", it is roughly 0.4 above a floor near 1.7.

**Where this breaks:** the speed of light is known exactly and the compression
bound is computable for a specified source. Language's entropy floor is
*estimated*, from fits to observed scaling curves, and it is only defined
relative to a corpus and a tokenizer. Change the data mixture and the floor
moves. So it is a real bound with a fuzzy value, unlike the two comparisons,
where the bound is crisp and the measurement is the uncertain part.

**A near-zero training loss is like a test suite that passes because the
assertions were generated from the current output, AND it is like a cache with a
100% hit rate that is actually serving one request over and over.** The shared
structure: a metric that looks like total success is in fact evidence that the
system is being measured against itself. In all three cases the diagnostic is the
same, and it is the one from canon: evaluate on inputs the system has not
previously consumed.

**Where this breaks:** self-generated assertions and degenerate cache keys are
outright bugs with a correct version that has neither. Memorization is not a bug
in the loss function and does not have a "fixed" version; it is the correct
optimum of the objective you actually wrote down, on the data you actually
supplied. The objective is doing exactly its job. The mismatch is between that
objective and what you wanted.

## Backprop as blame assignment

**Backprop is like distributed tracing with span-level attribution, AND it is
like reverse-topological-order propagation in a build system's dependency
graph.** The shared structure: a directed acyclic graph, a quantity known at one
end, and a single traversal in the correct order that yields per-node values,
because each node needs only its local rule plus what its consumers already
computed. Attempting it out of order, or without caching, degrades to
exponential recomputation, which is why both tracing and build systems memoize
per node exactly as backprop does.

**Where this breaks:** in tracing, the spans' contributions to total latency
sum, and you can point at one span and say it caused 40ms. Gradients do not sum
to the loss and are not attributions of the loss. A gradient is a *derivative*:
the change in loss per unit change in that weight, valid only in an infinitesimal
neighborhood of the current weights. It says nothing about how much of the
current loss that weight is responsible for, and it changes completely after one
step. "Blame" is a scaffold for the direction of information flow, not a claim
about causal share.

**The choice of reverse mode over forward mode is like choosing to build a
reverse index instead of scanning, AND it is like memoizing on the small side of
a join.** The shared structure: the same computation, two orders, and the right
order is determined entirely by which side has the smaller cardinality. Reverse
mode costs one sweep per *output*, forward mode one per *input*; with $10^{11}$
inputs and 1 output, the choice is not close. Recognizing which side is small is
the identical skill as picking the driving table in a query plan.

**Where this breaks:** a reverse index is built once and amortized over many
queries. The backward pass is rebuilt from scratch every single step, because
the local derivatives depend on the current activations, which depend on the
current batch. There is nothing amortized. This is also why the memory cost is
per-step rather than a fixed structure you allocate once.

## Backprop worked: two layers, real numbers

**The weight gradient being (input activation) x (output blame) is like a
profiler attributing cost to a call site by multiplying call frequency by
per-call cost, AND it is like a feature flag's impact being the product of its
exposure rate and its per-exposure effect.** The shared structure: two
independent factors, both required. A code path called a million times with zero
cost each contributes nothing; a very expensive path never called contributes
nothing. A weight fed by a silent input contributes nothing regardless of how
blamed its output was, and a weight whose output was already correct contributes
nothing regardless of how loud its input was. Only the product moves.

**Where this breaks:** call frequency and per-call cost are independently
measurable and stable across runs. The activation and the blame are both
functions of the *same* forward pass on the *same* example, and they change
completely on the next batch. There is no stable per-weight number to profile.
The product is a momentary reading, and training works only because it is
averaged over batches and repeated millions of times.

**The transpose in $W^\top \delta$ is like the reverse edge index in a graph
database, AND it is like an inverse mapping in a bidirectional data sync.** The
shared structure: forward, you need "given this input, which outputs does it
feed"; backward, you need "given this output, which inputs fed it". Same
relation, opposite direction, and the transpose is the reverse adjacency of the
same weight matrix. No new information is stored, which is the point: one weight
matrix serves both directions.

**Where this breaks:** a reverse edge index is a genuinely separate materialized
structure that can drift from the forward one. $W^\top$ is not materialized and
cannot drift, because it is the same memory read with a different stride, and
BLAS handles it with a flag. If you find yourself thinking about "keeping the
transpose in sync", you have imported the wrong part of the analogy.

**The ReLU gate zeroing gradients is like a circuit breaker in the open state,
AND it is like a short-circuiting `&&` in a hot path.** The shared structure: a
binary condition determined during the forward direction that completely blocks,
rather than attenuates, everything behind it. Nothing downstream of an open
breaker gets traffic; nothing after a false left operand gets evaluated; no
weight behind a closed gate gets gradient.

**Where this breaks:** and this is the important one. A circuit breaker has a
half-open state and probes for recovery; a short-circuit re-evaluates on the next
call with new operands. A dead ReLU unit has neither. If a unit is negative on
every example, its incoming weights receive exactly zero gradient forever, so
they never change, so it stays negative, so they never change. There is no probe,
no retry, no recovery path. It is a permanent, self-sustaining failure with no
analogue in either comparison, and it is why leaky variants and careful
initialization exist.

## Optimizers: four update rules, one loop

**Momentum is like an exponentially weighted moving average in an autoscaler,
AND it is like a PID controller's integral term.** The shared structure: all
three accumulate a signal over time so that persistent trends amplify while
oscillation cancels. An autoscaler on raw instantaneous load thrashes; on an EWMA
it responds to sustained load. The integral term builds up under a consistent
error and pushes harder the longer it persists. Momentum builds up along a
consistently-signed gradient direction, reaching ten times the raw gradient at
$\beta = 0.9$, and cancels along a direction that keeps reversing.

**Where this breaks:** an autoscaler and a PID controller are regulating toward
a *setpoint* that is known. There is no setpoint here. The loss has no target
value the optimizer is aiming at; it only ever moves downhill from where it is.
Any intuition you import about overshoot relative to a target, or steady-state
error, has nothing to attach to. The one intuition that does transfer is that
too much integral action causes overshoot, and momentum does overshoot valleys
for the same reason.

**Adam's normalization is like per-tenant rate limiting expressed in percent of
that tenant's own baseline, AND it is like normalizing heterogeneous metrics to
z-scores before comparing them.** The shared structure: quantities living on
incompatible scales are divided by their own typical magnitude, so a single
threshold or step size becomes meaningful across all of them. One global rate
limit is wrong for both the tenant sending 10 rps and the one sending 100k rps;
one global learning rate is wrong for both the embedding weights and the output
projection. Dividing by each one's own running magnitude makes one setting fit
all of them.

**Where this breaks:** rate limits and z-scores are computed over a
distribution you can inspect and validate. Adam's per-parameter denominator is a
running estimate over the last few thousand steps of a nonstationary process, and
it is never examined by anyone. More importantly, normalizing to relative scale
means a parameter receiving pure noise gets a full-size step, exactly like a
tenant sending nothing but retries getting the same relative allowance as a real
one. Adam has no notion of whether the signal it is normalizing means anything,
which is why it requires weight decay and warmup as guardrails.

**AdamW's decoupled weight decay is like applying a discount to the list price
rather than to the tax-inclusive total, AND it is like putting a middleware
outside versus inside a retry wrapper.** The shared structure: identical
arithmetic pieces, and the composition order changes the semantics. Inside the
adaptive normalization, the decay term gets divided by the gradient magnitude
and therefore means something different for every parameter. Outside it, it is a
flat multiplicative shrink that means the same thing everywhere. Same operands,
different result, purely from placement.

**Where this breaks:** in the middleware case, both orderings are defensible
depending on intent, and you can reasonably want retries logged or not logged.
Here one ordering is simply the one people wrote first and the other is
correct. There is no workload for which coupling weight decay into Adam's
normalization is what you meant; it was an artifact of writing L2 into the
gradient out of habit from SGD, where the two genuinely are identical.

## What gradient descent actually finds

**The multiplicity of good solutions is like the many valid orderings a
topological sort can return, AND it is like two independent teams shipping
services that pass the same contract tests with entirely different internals.**
The shared structure: the specification constrains behavior, not
representation, so the set of conforming implementations is enormous, and two
samples from it will be mutually unintelligible even though both are correct.
Two training runs are two samples from the set of weight configurations with low
loss.

**Where this breaks:** you can diff two topological sorts and match elements, and
two service implementations still expose the same API you can call. Two trained
networks do not have a correspondence you can compute; matching run A's units to
run B's is an open research problem, not a mapping you can look up. Which is why
the operation that seems obvious, averaging their weights, produces a model worse
than either, whereas averaging two teams' load-test results is meaningful.

**Saddle points versus local minima is like the difference between a deadlock
and a slow query.** Paired with: **the difference between an unreachable state
and a hot loop with poor constant factors.** The shared structure: in each pair,
one failure is categorical and requires intervention to escape, the other is
merely slow and resolves with time or a nudge. The 2D optimization picture
predicts deadlock, where the process cannot proceed regardless of patience. What
high-dimensional loss surfaces actually deliver is the slow-query case: a
descent direction always exists, progress is just poor, and gradient noise finds
the exit.

**Where this breaks:** a slow query has a knowable cause you can find in the
plan, and fixing it makes it fast. There is no equivalent inspection for a
training plateau. You cannot look at the Hessian of a hundred-billion-parameter
model and identify the shallow direction. The practical response is empirical
(adjust the schedule, check the data, check for numerical issues) rather than
diagnostic, which is a genuinely worse position than any of these systems
analogies would lead you to expect.

## Scaling laws: what more parameters buy

**A scaling law is like a capacity planning curve fitted from load tests, AND it
is like Amdahl's law with a measured serial fraction.** The shared structure:
each is an empirically parameterized relationship used to extrapolate before
committing resources, and each has an asymptote that no amount of the scaled
resource crosses. You fit throughput against instance count over a range you can
afford to test and read off what 10x would give. You measure the serial fraction
and learn the speedup ceiling regardless of core count. Scaling laws let you fit
loss against parameters and tokens on models you can afford, then price a run you
have not committed to, with the irreducible term playing the role of the serial
fraction.

**Where this breaks:** Amdahl's law is derived from a structural fact about the
computation and the serial fraction is a measurable property of your program.
Scaling law constants are pure curve fits with no derivation behind them, valid
only over the range fitted, and they have been revised as the range extended.
Treat them as calibrated extrapolation, not as law, despite the name. The
capacity-planning comparison also imports a wrong instinct: load-test curves
usually bend *down* as you find a bottleneck, while scaling curves have stayed
straight far past where people expected them to bend.

**Parameters buying feature composition rather than storage is like the
difference between a materialized view and a query planner, AND it is like the
difference between a lookup table of precomputed results and a library of
composable functions.** The shared structure: one artifact answers only
questions anticipated at build time; the other answers questions never
anticipated, by composing pieces. And the composing system has a failure mode the
storage system does not: it will happily compose an answer to a question it
cannot actually answer, and return it with the same interface and no error. A
lookup miss is explicit. A bad composition is silent.

**Where this breaks:** a query planner's composition is exact, auditable, and
you can read the plan. Model composition is learned, distributed across weights,
and largely uninspectable, which is why "just look at what it did" is available
for a query and not for a generation. The analogy gets the *category* right and
badly understates the observability gap.

**Chinchilla's allocation result is like discovering your cluster was
provisioned for peak RAM but starved on IOPS, AND it is like buying more cores
for a workload that was memory-bandwidth-bound all along.** The shared
structure: a fixed budget split across complementary resources, an imbalance that
looks like scale but is actually waste, and a rebalance at constant cost that
improves the outcome. Models were being built with enormous parameter counts and
too few tokens to fill them: capacity provisioned, experience starved.

**Where this breaks:** a bandwidth-bound workload shows an obvious saturation
signal in your metrics. The undertrained model has no such tell. It trains
fine, the loss goes down, and it produces a usable model. You only learn it was
misallocated by training the alternative and comparing, which is why the field
spent years on the wrong side of it despite everyone watching their loss curves.

## Post-training: SFT, RLHF, RLAIF, DPO

**Post-training is like setting defaults and a config profile on a library that
already has all its features, AND it is like a linter and formatter applied to
code that already compiles and passes tests.** The shared structure: the
capability set is fixed beforehand; the later stage selects among behaviors that
already exist, standardizes the presentation, and forbids some options. A
formatter cannot make your code correct and a config profile cannot add a
feature. Both are enormously visible to users and neither changes what the system
can do.

**Where this breaks:** a config profile is explicit, enumerable, and
reversible, and you can read the file to see what changed. Post-training is a
weight update, so there is no config to inspect, no list of what was turned on,
and no clean revert. Furthermore the boundary is not perfectly clean: post-
training on enough tokens does start to move capabilities, so "selects, never
adds" is right at the scales actually used and would stop being right if
post-training were scaled by four orders of magnitude.

**SFT versus preference optimization is like unit tests with expected outputs
versus A/B tests with a preference metric.** Paired with: **specifying a golden
file versus specifying a comparator.** The shared structure: the first form can
only express "produce this exact thing", so it requires someone to author the
right answer and cannot express relative quality at all. The second form only
needs a judgment between two candidates, which is far cheaper to obtain and can
rank outputs no human authored. This is exactly why the field moved from
demonstrations to preferences: writing the ideal response is expensive, choosing
between two is not.

**Where this breaks:** an A/B test measures a real metric on real users, and
Goodhart applies but the metric is at least ground truth. RLHF optimizes against
a *learned* reward model, which is a fitted approximation of preference, so it
degrades under optimization pressure in a way a real conversion metric does not.
The KL leash exists to bound how far you may push into the region where the
learned judge is wrong. There is no equivalent of that leash in an A/B test
because there is no equivalent failure.

**DPO versus PPO is like a closed-form solution versus an iterative solver, AND
it is like a batch job over a fixed dataset versus a control loop reading live
telemetry.** The shared structure: DPO exploits an algebraic identity to compute
the answer directly from data you already have; PPO samples, evaluates, and
corrects in a loop. The closed form is dramatically simpler to operate (one
training loop, no reward model, no sampling infrastructure) and the loop
handles cases the closed form's assumptions do not cover.

**Where this breaks:** a closed-form solution and an iterative solver converge to
the same answer, and preferring one is purely an engineering call. DPO and PPO do
not converge to the same policy in practice, because the derivation assumes the
preference data was generated by the reference policy, and it usually was not.
The characteristic DPO failure has no analogue in the closed-form comparison at
all: because only the *difference* in log-probabilities is constrained, DPO can
drive the loser's probability down much harder than it raises the winner's,
sending mass to sequences in neither set, so both go down together. That is not
a slower or less accurate solution to the same problem. It is a different
answer.

## What you now have

The five analogies worth keeping, each with its leash. Cross-entropy is a
scoring function over a probability distribution, not an assertion count - it
has an irreducible floor the way a compressor has an entropy bound, and a
build that reports zero has cheated. Backprop is distributed tracing over a
computation graph: one trace, per-span attribution, no re-running the request
per component - but unlike a trace it is exact rather than sampled. AdamW is
per-parameter autoscaling on a fleet where each service is normalized to its
own baseline traffic - which is exactly why the decay term had to be moved
outside the autoscaler. SGD's landscape is not a search for a global optimum
but a random walk into any acceptable configuration, the way two independent
deployments of the same service reach different but equally healthy steady
states. And post-training is configuration, not compilation: it selects among
behaviors the binary already contains.

**Where all five break:** every one of them describes an *engineered* system
with named parts and stated contracts. Training has neither. The parts are
learned, the contracts are statistical, and nothing in the system knows what
any of its components are for. Push any of these analogies to the point where
you expect to inspect, name, or individually replace a component, and it fails
- which is the same boundary M4 and M5 are drawn along.
