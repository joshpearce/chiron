## Two numbers and a gap

**The pair.** The leave-one-source-out experiment is like **`git revert` on a
single commit followed by re-running the benchmark**, and like **removing one
dependency from a build and re-measuring the binary**.

**The shared structure.** All three answer "what did this contribute" the only
way the question can be answered without a model of the system: construct the
world without it and compare. You do not reason about what the commit probably
did, you do not read the dependency's README and estimate. You build both
artifacts and measure both. The contribution is defined as the difference, and
the definition needs no theory of compilers, of caches, or of gradient descent.

**Where this breaks.** Revert a commit, rebuild, and re-run a deterministic
benchmark, and you get the same number every time. Train a model twice from the
same corpus with different seeds and you do not. The revert analogy carries the
epistemics of the counterfactual perfectly and imports exactly the wrong
expectation about reproducibility, which is why the opening table shows three
seeds rather than one. The closer software analogue is reverting a commit and
re-running a benchmark on shared cloud hardware with noisy neighbours: the
comparison is still valid, but a single pair of measurements is not a result.

---

**The pair.** The three-seed table is like **an A/B test with one user in each
arm**, and like **declaring a performance regression from a single profiling
run**.

**The shared structure.** In each case you have made a real comparison of two
real conditions and the comparison is structurally sound. What is missing is any
estimate of how much the measurement moves when nothing changes. Nobody would
ship a pricing change off one user per arm, and the reason is not that the
experiment was designed wrong - it is that a sample of one carries no information
about its own variability.

**Where this breaks.** An A/B test's variance comes from user heterogeneity,
which you can reason about and sometimes reduce with stratification or with
covariate adjustment. Seed variance comes from the training process itself, and
there is no stratification available - the two runs differ in initialization and
shuffle order, both of which are the thing being averaged over rather than
nuisances to be balanced. You cannot get a paired design out of it either, since
changing the corpus changes the shuffle, so the seeds in the two conditions are
not matched in any useful way. Repetition is the entire toolkit.

## Contribution is a counterfactual

**The pair.** Choosing sources rather than documents as the attribution unit is
like **bisecting over release tags instead of individual commits**, and like
**profiling at the function level instead of the instruction level**.

**The shared structure.** Each is a deliberate choice to measure a coarser unit
because the finer one is below the resolution of the instrument. A sampling
profiler cannot attribute time to a single instruction - the sample interval is
longer than the instruction - so it attributes to functions, where the signal
accumulates above the sampling noise. Bisecting over tags rather than commits
works when your test is flaky enough that a single commit's effect cannot be
distinguished from a rerun. In both cases the coarser unit is not a compromise on
rigour; it is the finest unit at which the measurement is valid at all.

**Where this breaks.** With a profiler you can usually go finer if you are
willing to pay - switch to instrumentation, or hardware counters, and the
resolution genuinely improves. Here you cannot. The cost of resolving a unit
grows as the inverse square of its share of the corpus, so halving the
granularity quadruples the bill, and per-document resolution is not expensive but
unreachable: roughly a hundred thousand GPU-hours for a single document. There is
no instrumented mode to switch to. The other break is that a profiler's function
boundaries are arbitrary with respect to what you are optimizing, whereas here
the coarse unit is chosen to match the payment unit, so the coarsening is aligned
with the purpose rather than merely tolerated by it.

---

**The pair.** The signal-to-noise argument is like **trying to measure a
microsecond of latency with a millisecond-resolution timer**, and like
**detecting a 0.01% memory leak against a heap that fluctuates by 3% between
runs**.

**The shared structure.** In all three the obstacle is a ratio, not a technique.
No refactor of the measurement code moves it, no amount of care in running the
experiment moves it, and the correct engineering response is to change what you
are measuring: batch a million iterations and divide, hold the heap under a
harness that removes the fluctuation, aggregate documents into a source. Anyone
who has benchmarked a single fast function call already has the instinct - you do
not time one call, you time ten million and divide.

**Where this breaks.** The batching trick works for the timer because the
quantity you want is per-call and the calls are interchangeable, so the sum
divided by the count is the thing you wanted. Aggregating documents into a source
does *not* give you the per-document number divided by the count - it gives you a
different quantity, the source's contribution, and the per-document numbers are
not recoverable from it. This is a genuinely one-way coarsening. You are not
measuring the small thing more cleverly; you are measuring a different thing and
deciding that the different thing is what you were going to pay for anyway.

## Designing the sweep

**The pair.** The fixed-corpus versus fixed-token-budget protocol choice is like
**deciding whether a benchmark holds wall-clock time or iteration count
constant**, and like **deciding whether removing a cache tier keeps total memory
fixed or gives the memory back**.

**The shared structure.** In each, an obvious-looking ablation has a hidden
second variable that changes along with the thing you meant to change, and the
two reasonable ways to handle it answer different questions. Remove a cache tier
and hand its RAM back to the OS, and you have measured "what if we did not have
this cache." Remove it and reallocate the RAM to the remaining tiers, and you
have measured "is this cache the best use of this memory." Both are legitimate,
neither is the default, and a benchmark that does not say which one it did is not
interpretable.

**Where this breaks.** The memory case is usually reversible and cheap to run
both ways. Here, running both protocols doubles the sweep, and while that is
affordable at this scale it will not be at any larger one, so the choice is
usually made once and lived with. The more important break is who is affected:
in a benchmark, choosing the wrong control makes your conclusion wrong. Here it
systematically reallocates money between large and small rightsholders, because
the fixed-corpus protocol bundles volume into contribution and large sources have
more volume. That makes the protocol a disclosure obligation rather than a
methodological preference.

---

**The pair.** Reporting $\Delta \pm \sigma\sqrt{2/n}$ is like **quoting a
benchmark as a median with a p95 rather than a single number**, and like
**an SLO with an explicit error budget rather than a target**.

**The shared structure.** All three replace a point claim with a claim plus its
own tolerance, and in all three the tolerance is what makes the claim actionable.
A latency number with no distribution cannot be compared to next week's latency
number. An SLO without an error budget cannot tell you whether today's incident
matters. A contribution without a standard error cannot tell you whether the
source below it in the table is actually below it.

**Where this breaks.** A p95 is a fact about the population of requests, which
you observed. $\sigma$ is a fact about a population of training runs you did not
observe and mostly never will - you have three draws and are inferring a spread
from them, which is why the deeper-math treatment spends its length on pooling
$\sigma$ across conditions rather than estimating it per condition. The second
break is directional: a p95 tail is usually caused by something, and finding the
cause is productive work. Seed variance is not caused by anything. Chasing it is
the one debugging instinct that is actively wrong here, and an engineer who
pins the seed to make the numbers stable has not removed the variance, only
hidden it behind one arbitrary draw.

## What the sweep costs

**The pair.** Shrinking the model to afford the sweep is like **running the full
integration suite against a scaled-down fixture rather than a production-sized
one**, and like **fuzzing a parser at reduced input sizes to get more executions
per second**.

**The shared structure.** In each, the interesting property being tested -
correctness of the merge logic, presence of a crash, the sign and rough size of a
contribution - is not a property of scale, while the cost is entirely a property
of scale. So you buy vastly more experiments by shrinking the thing under test,
and the experiments are still valid experiments. Every fuzzing engineer knows
that executions per second is the dominant term and that shrinking inputs is the
cheapest way to buy them.

**Where this breaks.** A fuzzer's crashes usually reproduce at full size, and a
scaled-down fixture usually exercises the same code paths. Neither of those is
guaranteed here, and the literature is explicit that influence patterns change
with scale: small models show crisp top-heavy attributions where frontier-scale
influence is a long tail. So the small sweep is a valid measurement of the small
model and a *hypothesis* about a large one. Unlike the fuzzing case, you cannot
confirm it by rerunning the same finding at full size, because at full size the
counterfactual is unaffordable, which is the whole reason you shrank. The escape
is to repeat one scale up and check the trend, which is a real experiment with a
known price and is what an extension unit is for.

---

**The pair.** Cost being linear in parameter count is like **the observation that
CI time is linear in test count**, and like **storage cost being linear in
retention window**.

**The shared structure.** All three are unglamorous proportionalities that
determine what is possible far more than any clever optimization does. The
practical consequence in each is the same: when the budget does not fit, the
lever with the most travel is the multiplicand, not the constant. Nobody makes CI
affordable by micro-optimizing the runner; they cut the matrix. Nobody makes a
550-run sweep affordable by improving throughput 20%; they shrink the model
twentyfold.

**Where this breaks.** Cutting the CI matrix costs you coverage, and everyone
knows which tests they dropped. Shrinking the model costs you scope in a way that
is much harder to see, because the small sweep produces a table of exactly the
same shape as the large one would have, with the same column headers and
plausible numbers. Nothing about the artifact announces that it is scoped. That
is why the header block matters so much here and not in CI: the missing coverage
is invisible unless it is written down.

## The noise floor is a result

**The pair.** Reporting "below the noise floor" rather than a small number is
like **a profiler refusing to attribute time to a function that received fewer
samples than its confidence threshold**, and like **a monitoring system
suppressing an alert whose signal is inside the metric's own jitter**.

**The shared structure.** In each, the honest output for a below-resolution
observation is a statement about the instrument, not a smaller estimate. A
profiler that reported "0.03% of runtime" for a function it sampled twice would
be lying with more decimal places than a profiler that reported "insufficient
samples." Anyone who has chased a phantom hotspot out of a low-sample profile
already understands the failure mode: precision in the output is being confused
with resolution in the instrument.

**Where this breaks.** A profiler's remedy is to profile longer, and profiling
longer is cheap and reliable. Here the remedy is more seeds, and the required
count scales as the inverse square of the effect you want to resolve, so it goes
from three to eighteen to two thousand quite fast. There is a second, sharper
break: a below-threshold function in a profile is genuinely uninteresting and you
move on. A below-floor source in a contribution table is a rightsholder who is
about to be paid nothing, and "our instrument could not resolve your
contribution" is a materially different statement from "your contribution is
zero." The profiler analogy has no equivalent of somebody reading the suppressed
row and objecting.

---

**The pair.** The signal-to-noise threshold that decides flat fee versus
per-contribution payment is like **the decision to stop autoscaling and buy
reserved capacity**, and like **abandoning fine-grained cache invalidation for a
short TTL**.

**The shared structure.** All three are cases where a measurement-driven policy
is only better than a flat policy if the measurement is good enough, and where
the correct move when it is not is to stop measuring and take the flat rule. A
fine-grained invalidation scheme that is right 60% of the time is worse than a
30-second TTL, and the way you find out is to measure your hit rate rather than
to improve your invalidation logic. The threshold is real, it is crossable in
both directions, and knowing which side you are on is more valuable than any
particular refinement.

**Where this breaks.** Cache and capacity decisions are yours to revisit weekly
with no external audience. This one gets made once, in public, in front of people
whose money is on the other side of it, and a founder who has not measured their
own signal-to-noise cannot answer the single most obvious question about their
product. The other break: with caches you can hedge, running both strategies on
different traffic. A payment scheme is a contract and hedging is not on offer.

## Subset regression

**The pair.** Training many models on random subsets and regressing is like
**multivariate feature-flag experimentation instead of one flag at a time**, and
like **a fractional factorial build matrix instead of a full one**.

**The shared structure.** All three exploit the same fact: if you randomize
several variables simultaneously and independently, every variable's effect can
be read out separately, because the others are balanced across the comparison and
cancel. Sorting nights into "lamb served" and "lamb not served" works because the
soup appeared equally often in both piles. This is why a multivariate experiment
gets you ten flag effects out of one traffic allocation rather than ten
sequential experiments, and it is exactly why 300 subset runs beat 39 careful
one-at-a-time removals.

**Where this breaks.** A feature-flag platform gives you the analysis for free
and its estimand is the one you wanted: the effect of the flag on the population
you are serving. Here the estimand shifts with the design. The regression
coefficient is the average effect of a source across subsets of whatever density
you sampled - typically half-size - while the leave-one-out delta is the effect
at full density. If the sources interact at all, those are different numbers, and
the difference between them is not error but structure. In flag terms it is as if
your measured flag effect depended on what fraction of the other flags were on,
which for most feature flags is an edge case and for data sources is the normal
situation.

---

**The pair.** The balanced design that lets you read coefficients off the table
is like **an orthogonal test matrix where each axis is varied independently**,
and like **an experiment where the treatment assignment is uncorrelated with
every covariate**.

**The shared structure.** The value of orthogonality in all three is the same:
the estimates stop competing. In a tangled design, changing your estimate of one
effect forces a compensating change in another, which is why you need a solver
and why removing one variable can swing the rest. Under orthogonality each
comparison stands alone, and "fit the model" degenerates into a set of independent
subtractions. Anyone who has debugged a regression whose coefficients flip sign
when a correlated column is dropped has felt the alternative.

**Where this breaks.** In a test matrix you control the design completely and
orthogonality is a scheduling decision. Here the design is limited by the
combinatorics - you cannot have a fully orthogonal design over 12 sources in 300
runs the way you can over 4 sources in 8 - so real sweeps sample uniformly at
random and get *approximate* orthogonality, which is good enough in expectation
and leaves residual correlation that a solver has to untangle. The second break:
in a build matrix, two axes that are perfectly correlated is a configuration
mistake you fix. Two sources that genuinely always appear together in your corpus
cannot be separated by any design, and the honest report is that their combined
contribution is measured and their individual split is not identified.

## Scoring an attribution method

**The pair.** The Linear Datamodeling Score is like **validating a cost model
against measured query runtimes on held-out queries**, and like **grading a
cache-hit-rate predictor by its ranking of workloads rather than its per-key
predictions**.

**The shared structure.** In each you have a cheap predictor and an expensive
truth, and you grade the predictor by making it commit to predictions on inputs
it has not seen and then paying for the truth on those. Held-out is doing the
same work it does anywhere in ML: a method that has seen the answer is not being
tested. And in each, the sensible thing to grade is the *ordering*, because a
query planner that ranks plans correctly is useful even if its absolute cost
estimates are off by a constant factor.

**Where this breaks.** A query planner's truth is cheap - run the query and time
it - so you can afford a large held-out set and per-input grading. Here each
held-out point is a training run, so $J$ is small (30 to 60), which means the
score itself has a wide confidence interval; under the null, Spearman's $\rho$ on
50 points has a standard deviation of about 0.14, so an LDS of 0.15 is
indistinguishable from zero. An LDS quoted without its $J$ is not a number. The
second break is subtler: a cost model that ranks plans correctly is doing its
job, whereas an attribution method that ranks *subsets* correctly may still be
wrong about individual sources, because many per-source vectors produce the same
subset sums. Ranking the aggregate does not certify the parts.

---

**The pair.** Rejecting per-example leave-one-out correlation as a metric is like
**rejecting a flaky test as a regression signal**, and like **refusing to tune
against a benchmark whose run-to-run variance exceeds the effect you are
tuning for**.

**The shared structure.** In all three the object being used as a reference is
noisier than the thing being measured against it, so the comparison carries no
information regardless of how carefully it is performed. Anyone who has watched a
team spend a sprint optimizing against a benchmark that turned out to have 8%
run-to-run variance recognizes the shape: the work was real, the measurements
were honest, and the signal was never there.

**Where this breaks.** A flaky test can usually be fixed, and a noisy benchmark
can usually be stabilized with more iterations or a quieter machine. The
per-example ground truth cannot be, at any price you would pay - stabilizing it
means about a million seeds per document. So the response is not "fix the
reference" but "choose a different reference," and that substitution changes what
is being certified. The LDS certifies subset-level prediction, which is a real
and useful property and is not the property the rejected metric was trying to
certify. It is worth being explicit that this is a change of question and not
merely a better measurement of the old one.

## What ground truth buys

**The pair.** The counterfactual table is like **a reference implementation used
as an oracle in differential testing**, and like **a golden file that a fast path
gets diffed against**.

**The shared structure.** In all three, a slow, obviously-correct implementation
exists solely to check a fast, subtle one. Nobody ships the reference; nobody
optimizes it. Its value is entirely that it is right by construction, so any
disagreement is a bug in the fast path rather than an open question. This is
exactly the relationship between the leave-one-out sweep and every gradient
method in v8, and it is why building the slow one first is the correct order
despite being the less interesting half.

**Where this breaks.** A golden file is cheap to produce and, crucially, is a
*complete* oracle - it covers every input you care to test. The counterfactual
table is expensive, covers only source-level questions on one model
configuration, and is itself noisy, so a disagreement between it and a fast
method is not automatically a bug in the fast method. You need the noise floor to
tell the two apart, which is a step that has no analogue in differential testing
where the reference is deterministic. And unlike a golden file, this oracle is
scoped: it certifies the fast path at 10M parameters and says nothing about 8B,
whereas a reference parser is a reference parser at every input size.

---

**The pair.** Building the calibration standard before the attribution machinery
is like **writing the failing test before the fix**, and like **standing up
metrics and dashboards before starting a performance project**.

**The shared structure.** Each inverts the tempting order for the same reason:
without the measurement in place first, you cannot tell whether the work you
subsequently do accomplishes anything, and you will find out much later or never.
A performance project without a baseline produces "it feels faster." An
attribution implementation without a ground truth produces a ranking that looks
exactly as plausible when it is correct as when it has a sign error in the middle
of it. The discipline is identical and so is the temptation to skip it.

**Where this breaks.** A failing test tells you unambiguously when you are done,
and a dashboard tells you unambiguously whether the number moved. An LDS does
neither, because there is no threshold at which an attribution method is
"correct" - the published measurements say the scalable ones land at chance, so
your realistic target is a correlation that is meaningfully above zero rather
than one that is near one. You are calibrating a scale that may turn out to be
unusable, and the outcome "this cheap method does not work on my corpus" is a
successful use of the standard rather than a failure of the project. That is a
different relationship to a test than most engineering has.

## At the bench: the leave-one-out sweep

**The pair.** Caching every $(S, v(S))$ pair keyed by the source set is like
**memoizing an expensive pure function on a content hash**, and like **a build
cache keyed by the hash of inputs plus the toolchain version**.

**The shared structure.** The value function is pure in exactly the sense a build
is: same source set, same configuration, same seed, same output. So it is
cacheable by content hash, and the cache is worth much more than the hit rate
suggests because the misses are expensive and unevenly distributed. The
toolchain-version part of the key is not optional - a build cache keyed only on
sources and not on compiler version silently serves wrong artifacts, and a
coalition cache keyed only on the source set will happily hand v5 a bpb value
that was computed under a different protocol or a different model size.

**Where this breaks.** A build cache's entries are interchangeable with fresh
builds, and a hit is exactly as good as a miss that ran. Here a cached entry
carries the seed it was computed with, so a hit gives you one draw rather than
the mean, and v5 needs seed-averaged values. The correct cache holds all seeds
per key and returns the mean plus the count, and code that treats it as a
one-value-per-key map will quietly reintroduce the single-seed error the whole
unit is about. The other break is retention: a build cache can evict freely
because regeneration is cheap. These entries cost minutes to hours each and are
the input to units that have not been written yet. Do not evict.

## What you can now do

**The pair.** The habit this unit installs is the discipline of a **benchmark
harness you would let someone else's promotion depend on**, and of a **postmortem
that states its confidence rather than its narrative**.

**The shared structure.** Both are about the difference between a number and a
result. A number is what the tool printed. A result is a number plus the
conditions under which it was produced, the variation you would see if you ran it
again, and an explicit statement of what it does not cover. Every engineer has
been on both sides of this: producing a benchmark that was technically accurate
and rhetorically misleading, and receiving one.

**Where this breaks.** In a benchmark or a postmortem the audience is colleagues,
the stakes are a decision, and being wrong is recoverable at the cost of some
credibility. Here the audience includes a rightsholder's lawyer and the stakes
are money moving between parties, so the standard is not "would a reviewer accept
this" but "would this survive an opposing expert who is paid to attack the
procedure rather than the arithmetic." That is a higher bar than engineering
usually operates at, and it is the bar v6 and v7 are built around. The good news
is that the practices are the same ones; only the consequences of skipping them
change.
