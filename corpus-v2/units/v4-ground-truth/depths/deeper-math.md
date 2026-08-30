## Two numbers and a gap

**What "exact" means, stated carefully.** The claim in canon is that
$\Delta_7 = v(\text{all} \setminus 7) - v(\text{all})$ is exact rather than
estimated. That is true and it is narrower than it sounds, so here is the
precise version.

Fix everything about a training run except the random seed: architecture,
parameter count, token budget, schedule, corpus, held-out set. What remains is a
map from a seed to a trained model, and composing it with the evaluation gives a
random variable. Write $V(S)$ for that random variable - the held-out bpb of a
model trained on source set $S$ with a seed drawn at random - and $v(S)$ for one
realized draw of it.

There are then two different objects, and the whole unit is about not confusing
them:

$$\delta_i = \mathbb{E}[V(\text{all} \setminus i)] - \mathbb{E}[V(\text{all})]
\qquad\text{versus}\qquad
\hat\Delta_i = \bar v(\text{all} \setminus i) - \bar v(\text{all})$$

where $\mathbb{E}[\cdot]$ is the expectation over seeds - the average you would
get from infinitely many runs - and the bars are averages over the $n$ seeds you
actually ran.

$\delta_i$ is the estimand: the thing that deserves the name "source $i$'s
contribution." It is a fixed number, not a random one, and it is defined without
any theory of learning. $\hat\Delta_i$ is the estimator: unbiased for $\delta_i$,
because the expectation of a difference of sample means is the difference of the
population means, and noisy in a way the rest of this unit quantifies.

So the honest sentence is: *the definition is exact, the estimator is unbiased,
and the realization has an error bar.* What is not true, and what the opening
table punishes, is that a single realized $v$ is a measurement of $\delta$.

**One more subtlety that matters later.** $\delta_i$ is defined relative to a
distribution over seeds, and that distribution includes the data shuffle. If
your dataloader is deterministic given the seed - which it should be, and which
Levanter-style bitwise reproducibility gives you for free - then $V(S)$ is a
genuine function of the seed and the whole apparatus is well-posed. If your
pipeline has nondeterminism outside the seed (nondeterministic reductions,
varying hardware, preemption-driven restart points), then $V(S)$ depends on
things you are not sampling deliberately, $\sigma$ silently absorbs them, and
your error bars are still valid but your reproducibility claims are not.

## Contribution is a counterfactual

**The proportionality assumption, examined.** Canon estimates a per-document
effect by scaling the per-source effect by the ratio of token shares, and flags
the assumption. Here is what is actually known about that scaling and why the
conclusion survives anyway.

Held-out loss as a function of training tokens $D$ follows, empirically, a power
law of the form

$$\mathcal{L}(D) \approx \mathcal{L}_\infty + \frac{A}{D^{\alpha}}$$

where $\mathcal{L}_\infty$ is the irreducible term, $A$ is a fitted constant, and
$\alpha$ is a fitted exponent typically near 0.3 to 0.5 for language models.
Removing a fraction $f$ of the corpus takes $D$ to $(1-f)D$, so

$$\Delta(f) = \mathcal{L}\big((1-f)D\big) - \mathcal{L}(D)
= \frac{A}{D^\alpha}\left[(1-f)^{-\alpha} - 1\right]$$

For small $f$, $(1-f)^{-\alpha} - 1 \approx \alpha f$, so $\Delta(f)$ is
approximately linear in $f$ with slope $\alpha A / D^\alpha$. That is the
proportionality canon assumed, and the expansion says it is a good approximation
exactly in the small-$f$ regime - which is where both the source case
($f = 0.08$) and the document case ($f = 4 \times 10^{-5}$) live.

Note what this does *not* justify: extrapolating to $f$ near 1. At $f = 0.5$,
$(1-f)^{-\alpha} - 1 = 2^{\alpha} - 1 = 0.23$ at $\alpha = 0.3$, against a linear
prediction of $0.15$ - a 50% underestimate. The half-size subsets of section 6
are outside the linear regime, which is one more reason the regression's
coefficients and the leave-one-out deltas are not the same number.

Note also what it ignores: this treats all tokens as interchangeable. They are
not, and the difference between "removing 8% of tokens" and "removing *these* 8%
of tokens" is precisely the signal being hunted. The power law gives the volume
term; the residual after subtracting it is the content term, and under the
fixed-token-budget protocol the volume term is held constant by construction so
the residual is what you measure. That is the mathematical statement of why the
protocol choice matters.

**The infeasibility arithmetic, done exactly.** Required seeds for $z \ge 2$:

$$n \geq \frac{8\sigma^2}{\Delta^2}$$

With $\sigma = 3 \times 10^{-3}$ and a per-document $\Delta = 7.5 \times 10^{-6}$:

$$n \geq \frac{8 \times 9 \times 10^{-6}}{5.625 \times 10^{-11}}
= \frac{7.2 \times 10^{-5}}{5.625 \times 10^{-11}} = 1.28 \times 10^{6}$$

Two conditions, so $2.56 \times 10^6$ runs per document. At 225 s per run on one
H100 that is $5.76 \times 10^8$ seconds, or $1.6 \times 10^5$ GPU-hours - about
$240{,}000$ to $640{,}000$ at mid-2026 rates, for one document out of 25,000.

The scaling worth carrying: because $n$ goes as $1/\Delta^2$ and $\Delta$ goes
as the token share $f$, **the cost of resolving a unit goes as $1/f^2$.** Halve
the granularity and quadruple the bill. This is why there is no gradual path
from source-level to document-level attribution: the cost curve is a wall, not a
slope.

## Designing the sweep

**The variance algebra, in full.** Let $X_1, \dots, X_n$ be condition A's
held-out bpb across seeds and $Y_1, \dots, Y_n$ condition B's, all independent,
each with variance $\sigma^2$. Then

$$\text{Var}(\bar X) = \text{Var}\left(\frac{1}{n}\sum_i X_i\right)
= \frac{1}{n^2}\sum_i \text{Var}(X_i) = \frac{n\sigma^2}{n^2} = \frac{\sigma^2}{n}$$

and since $\text{Var}(-Y) = \text{Var}(Y)$,

$$\text{Var}(\bar X - \bar Y) = \frac{\sigma^2}{n} + \frac{\sigma^2}{n}
= \frac{2\sigma^2}{n}, \qquad
\text{SE}_\Delta = \sigma\sqrt{\frac{2}{n}}$$

Setting $\Delta/\text{SE}_\Delta \ge z^*$ and solving for $n$:

$$n \geq \frac{2 z^{*2}\sigma^2}{\Delta^2}$$

which at $z^* = 2$ is canon's $8\sigma^2/\Delta^2$.

**Why you must pool $\sigma$ across conditions.** Canon says to measure $\sigma$
once. Here is the reason, and it is quantitative. If the per-run bpb is
approximately normal, the sample variance from $n$ runs satisfies

$$\frac{(n-1)s^2}{\sigma^2} \sim \chi^2_{n-1}$$

and the relative standard deviation of $s$ itself is approximately
$1/\sqrt{2(n-1)}$. At $n = 3$ that is $1/\sqrt{4} = 50\%$. An $s$ estimated from
three seeds of one condition is routinely off by half, and since $s$ is the
denominator of every $z$, a low draw manufactures significance across the whole
row.

The fix costs nothing, because you already have the runs. With $K+1 = 13$
conditions at $n = 3$ each, pool:

$$s^2_{\text{pooled}} = \frac{1}{(K+1)(n-1)}\sum_{c=1}^{K+1}\sum_{i=1}^{n}
\left(v_{c,i} - \bar v_c\right)^2$$

on $(K+1)(n-1) = 26$ degrees of freedom. The relative error on $s$ drops to
$1/\sqrt{52} = 13.9\%$. This assumes $\sigma$ is common across conditions, which
is testable - compare the per-condition spreads - and is a good assumption when
the conditions differ only by which 8% of the corpus is absent.

**$z$ or $t$.** With $\sigma$ estimated rather than known, the test statistic is
Student's $t$ on the pooled degrees of freedom, not a normal $z$. It matters
enormously if you did not pool and not at all if you did:

| $\sigma$ estimated from | df | two-sided 5% critical value |
|---|---|---|
| one condition, $n = 3$ | 2 | 4.30 |
| one condition, $n = 6$ | 5 | 2.57 |
| 13 conditions pooled, $n = 3$ | 26 | 2.06 |
| known exactly | $\infty$ | 1.96 |

Pooling moves the bar from 4.30 to 2.06, which is a factor of $(4.30/2.06)^2 =
4.4$ in required seeds. Pooling $\sigma$ is the cheapest statistical decision
available in this unit.

**Power, not just significance.** $n \geq 8\sigma^2/\Delta^2$ gives the $n$ at
which the *expected* $z$ equals 2. If the true effect is exactly $\Delta$, you
clear the bar roughly half the time. To detect it reliably the multiplier is
$(z_{\alpha/2} + z_\beta)^2$ rather than $z^{*2}$; for 80% power at a two-sided
5% level that is $(1.96 + 0.84)^2 = 7.84$ against $4$:

$$n_{\text{power}} \geq \frac{2(1.96+0.84)^2\sigma^2}{\Delta^2}
= \frac{15.7\,\sigma^2}{\Delta^2}$$

Roughly double canon's figure. Use canon's formula to decide whether an
experiment is plausible and this one to decide how many seeds to launch.

**Multiplicity, which v6 and v7 will enforce.** A $K$-source table is $K$
hypothesis tests. At $K = 12$ and an uncorrected 5% threshold, the probability of
at least one false positive under a global null is $1 - 0.95^{12} = 46\%$. On a
table you intend to attach money to, that is not acceptable. Bonferroni sets the
per-test level to $\alpha/K = 0.00417$, moving the critical value from 1.96 to
2.86 and raising required seeds by $(2.86/1.96)^2 = 2.13$.

Stacking the three corrections - pooling, power, multiplicity - the honest
planning figure is roughly $4\times$ canon's $n$. At $\sigma = 0.003$ and a
target resolvable effect of 0.005 bpb, canon says $n \ge 2.9$ and the planning
figure says 12 seeds per condition, which is 156 runs and still under a
GPU-hour of the sweep's budget.

## What the sweep costs

**Allocating a fixed run budget.** Suppose you can afford $R$ total runs and
must split them between the leave-one-out sweep and the subset design. The two
estimators have different variances per run:

$$\text{SE}_\Delta = \sigma\sqrt{\frac{2}{n}} \quad\text{with}\quad
R_{\text{LOO}} = (K+1)n
\qquad\qquad
\text{SE}_\beta = \frac{2\sigma}{\sqrt{M}} \quad\text{with}\quad
R_{\text{sub}} = M$$

Rewrite the first in terms of its own run count:
$n = R_{\text{LOO}}/(K+1)$, so

$$\text{SE}_\Delta = \sigma\sqrt{\frac{2(K+1)}{R_{\text{LOO}}}}
= \frac{\sigma\sqrt{2(K+1)}}{\sqrt{R_{\text{LOO}}}}$$

Both fall as $1/\sqrt{\text{runs}}$; only the constant differs. The leave-one-out
constant is $\sigma\sqrt{2(K+1)} = \sigma\sqrt{26} = 5.1\sigma$ at $K = 12$; the
subset constant is $2\sigma$. So per run the subset design is
$(5.1/2)^2 = 6.5$ times more efficient, exactly the factor canon quotes - and
the reason is visible in the algebra: the leave-one-out design pays a factor of
$K+1$ because each run informs one condition, while every subset run informs
every coefficient.

Note what happens as $K$ grows. The subset constant does not depend on $K$ at
all; the leave-one-out constant grows as $\sqrt{K}$. At $K = 50$ the efficiency
ratio is $2(51)/4 = 25.5$. This is the mathematical content of the advice to use
regression rather than exhaustive removal once the source count is large - and
it is also, in a different notation, the argument that will produce
Monte-Carlo semivalue estimation in v5.

**The marginal run.** Since both standard errors fall as $R^{-1/2}$, the
derivative of precision with respect to spend falls as $R^{-3/2}$. Doubling the
budget buys a $\sqrt{2} = 1.41$ times tighter error bar; a tenfold budget buys
$3.16$. Any plan that needs an order of magnitude more resolution needs two
orders of magnitude more money, and at that point the right move is to raise the
effect size - coarser sources - rather than to buy runs. Effect size enters as
$\Delta^2$; budget enters as $R$. They are not comparable levers.

## The noise floor is a result

**What $\sigma$ actually contains.** Decompose the variance of a single run's
held-out bpb by what is being resampled:

$$\sigma^2 = \underbrace{\sigma^2_{\text{init}}}_{\text{weight initialization}}
+ \underbrace{\sigma^2_{\text{order}}}_{\text{shuffle and batch composition}}
+ \underbrace{\sigma^2_{\text{eval}}}_{\text{finite held-out set}}
+ \underbrace{\sigma^2_{\text{sys}}}_{\text{nondeterministic reductions, hardware}}$$

These are separately measurable and it is worth doing once, because the remedies
differ. Fix the seed and vary only the data order: what remains is
$\sigma^2_{\text{order}}$, which the literature reports as the largest term, and
which no amount of careful initialization touches. Hold the model fixed and
resample the held-out set: $\sigma^2_{\text{eval}}$, which you shrink by
reserving more held-out text - the only term you can reduce for free, since eval
is not a training run. If $\sigma^2_{\text{sys}}$ is nonzero you have a
reproducibility bug rather than a statistical fact, and it is worth finding
because it makes replay-based auditing impossible in v7.

**The minimum resolvable effect.** Canon reports it as $2\sigma\sqrt{2/n}$,
which is just the $z = 2$ bar rearranged. It is worth naming as a design
quantity because it is what you promise a reader:

$$\Delta_{\min} = z^*\sigma\sqrt{\frac{2}{n}}$$

At $\sigma = 0.003$, $n = 3$, $z^* = 2$: $\Delta_{\min} = 0.0049$ bpb. Any source
whose true contribution is below that will land below the floor no matter how
carefully you run the experiment, and saying so in advance is what separates a
pre-registered result from a fishing expedition.

**Signal-to-noise as the quantity that matters downstream.** Define, for the
table as a whole,

$$\text{SNR} = \frac{\text{SD}_i(\delta_i)}{\text{SE}_\Delta}$$

the spread of the true contributions across sources divided by the error bar on
each. This is the number the payment-contract literature threshold is stated in.
Note it has a numerator you must also estimate, and that the naive estimate is
biased upward: the observed spread of your $\hat\Delta_i$ includes measurement
noise, so

$$\text{Var}_i(\hat\Delta_i) = \text{Var}_i(\delta_i) + \text{SE}_\Delta^2$$

Subtract before quoting. For the six-row excerpt in canon, the observed spread of
the deltas is about 0.013 and $\text{SE}_\Delta = 0.00245$, so the corrected
signal spread is $\sqrt{0.013^2 - 0.00245^2} = 0.0128$ and the SNR is about 5.2.
The correction is small here because the signal dominates; on a table where most
sources are near the floor it is the difference between an SNR of 1.5 and an SNR
of 0.

## Subset regression

**Least squares, and why the balanced design decouples it.** With $M$ runs, let
$y_m$ be run $m$'s held-out bpb and $x_{m,i} \in \{0,1\}$ its inclusion
indicators. Ordinary least squares chooses the coefficients minimizing

$$\text{RSS}(\beta) = \sum_{m=1}^{M}\left(y_m - \beta_0 - \sum_{i=1}^{K}\beta_i x_{m,i}\right)^2$$

Setting the derivative with respect to $\beta_i$ to zero gives the normal
equation for source $i$:

$$\sum_m x_{m,i}\left(y_m - \beta_0 - \sum_j \beta_j x_{m,j}\right) = 0$$

In general these $K+1$ equations are coupled: $\beta_i$ appears in every other
source's equation through the cross terms $\sum_m x_{m,i}x_{m,j}$. The design is
*orthogonal* when, after centring the indicators at their means, those cross
terms vanish for every pair $i \neq j$ - which is exactly the balance property
canon describes, that every in/out combination of any two sources appears equally
often. Under orthogonality each equation involves only its own coefficient, and
the solution collapses to

$$\hat\beta_i = \bar y_{\{x_i = 1\}} - \bar y_{\{x_i = 0\}}$$

the difference of group means. So "solve by inspection" is not a shortcut for a
special case; it is what least squares *is*, once the columns stop competing to
explain the same variation.

**Variance of a coefficient under random sampling.** If each source is included
independently with probability $p$, then about $Mp$ runs have it and $M(1-p)$ do
not, so

$$\text{Var}(\hat\beta_i) \approx \sigma^2\left(\frac{1}{Mp} + \frac{1}{M(1-p)}\right)
= \frac{\sigma^2}{M}\cdot\frac{1}{p(1-p)}$$

$$\text{SE}_\beta = \frac{\sigma}{\sqrt{M}}\cdot\frac{1}{\sqrt{p(1-p)}}$$

Minimized at $p = 1/2$, where $1/\sqrt{p(1-p)} = 2$ and the formula reduces to
canon's $2\sigma/\sqrt{M}$. The penalty for sampling elsewhere is real but mild
until the extremes: at $p = 0.75$ the factor is $2.31$ (15% worse), at $p = 0.9$
it is $3.33$ (67% worse), at $p = 0.95$ it is $4.59$.

That is a genuine tension and worth naming. The most statistically efficient
design samples half-size coalitions. The regime you actually want to describe -
"what does this source add to a corpus that has everything else" - is
$p \to 1$. Efficiency and relevance point in opposite directions, and the next
result says exactly how much that costs you.

**Interactions, and what the regression coefficient really estimates.** Suppose
the true value function has pairwise structure:

$$v(S) = \beta_0 + \sum_i \beta_i x_i + \sum_{i<j}\gamma_{ij}x_i x_j$$

where $\gamma_{ij}$ is the interaction between sources $i$ and $j$ - negative
when they are complements (together they help more than separately), positive
when they are substitutes or duplicates (together they help less).

Compute the leave-one-out delta from the full corpus. Setting all $x = 1$ except
$x_i$:

$$\Delta_i = v(\text{all} \setminus i) - v(\text{all})
= -\left(\beta_i + \sum_{j \neq i}\gamma_{ij}\right)$$

Now compute what the main-effects regression recovers under independent
$\text{Bernoulli}(p)$ sampling. The conditional means differ by

$$\mathbb{E}[v \mid x_i = 1] - \mathbb{E}[v \mid x_i = 0]
= \beta_i + \sum_{j\neq i}\gamma_{ij}\,\mathbb{E}[x_j]
= \beta_i + p\sum_{j\neq i}\gamma_{ij}$$

so $c_i = -\big(\beta_i + p\sum_{j}\gamma_{ij}\big)$.

Put the two side by side:

$$\Delta_i = -\Big(\beta_i + 1 \cdot \textstyle\sum_j \gamma_{ij}\Big)
\qquad\qquad
c_i = -\Big(\beta_i + p \cdot \textstyle\sum_j \gamma_{ij}\Big)$$

**They are the same functional evaluated at different subset densities.** The
leave-one-out delta is the density-1 case; the regression coefficient is the
density-$p$ case. They agree if and only if either the interactions cancel,
$\sum_j \gamma_{ij} = 0$, or $p = 1$. This is the exact statement of what canon
calls "additivity," and it says that the disagreement between the two tables is
not an error in either - it is a direct readout of $\sum_j\gamma_{ij}$, scaled
by $1 - p$:

$$\Delta_i - c_i = -(1-p)\sum_{j\neq i}\gamma_{ij}$$

At $p = 1/2$ the gap is exactly half the total interaction of source $i$ with
everything else. Compute it for every source. A table of $\Delta_i - c_i$ is a
free interaction diagnostic that costs no additional runs, and a source with a
large value is a source whose payout depends on which other sources are in the
deal.

**And the preview to v5.** The Shapley value of source $i$ is a weighted average
of its marginal contribution over coalitions of every size, with the weights
chosen so each *size* is equally likely. In the language above, that is
averaging the density-$p$ answer uniformly over $p$ from 0 to 1, rather than
picking $p = 1$ (leave-one-out) or a single sampled $p$ (regression). Which is to
say: the Shapley value is what you get when you refuse to choose a subset
density, and the axioms in v5 are the argument that refusing is the only fair
choice.

## Scoring an attribution method

**Why the per-example metric fails, quantified.** The relevant result is the
attenuation of correlation by noise in the ground truth. Let $t$ be the true
per-unit contribution, and let the measured ground truth be $m = t + e$ where $e$
is measurement noise with variance $\tau^2$, independent of $t$. For any method
producing scores $g$,

$$\text{corr}(g, m) = \text{corr}(g, t)\sqrt{\frac{\text{Var}(t)}{\text{Var}(t) + \tau^2}}$$

The square root is the *reliability* of the ground truth, and it is a hard
ceiling: no method, however perfect, can correlate with $m$ better than that.

Per-document leave-one-out. The spread of true per-document contributions is on
the order of $\text{SD}(t) \approx 7.5\times 10^{-6}$ bpb, and a single measured
per-document delta has $\tau = \sigma\sqrt 2 = 4.2 \times 10^{-3}$. Then

$$\sqrt{\frac{(7.5\times10^{-6})^2}{(7.5\times10^{-6})^2 + (4.2\times10^{-3})^2}}
= \sqrt{\frac{5.6\times10^{-11}}{1.76\times10^{-5}}} = \sqrt{3.2\times10^{-6}}
= 0.0018$$

**A perfect attribution method scores 0.0018 on this metric.** That is the
precise sense in which per-example leave-one-out correlation is a coin: the
ceiling is two thousandths, so every method scores zero and the metric has no
resolving power at all.

Now the same calculation for LDS. The held-out subsets differ by whole sources,
so $\text{SD}(t)$ across them is on the order of $0.012$ bpb, and a single-seed
measured subset outcome has $\tau = \sigma = 0.003$:

$$\sqrt{\frac{0.012^2}{0.012^2 + 0.003^2}} = \sqrt{\frac{1.44\times10^{-4}}{1.53\times10^{-4}}}
= \sqrt{0.941} = 0.97$$

**The ceiling on LDS is 0.97.** So an observed LDS of 0.90 is 93% of the way to
the maximum the measurement permits, and an observed LDS of 0.05 is genuinely
0.05 rather than an artifact. Same methods, same models, same seed noise; the
entire difference is which quantity was chosen as the ground truth. That is the
whole design insight behind the metric, and it is a variance calculation rather
than a statistical convention.

Two operational consequences. First, quote your ceiling alongside your LDS -
it is computable from $\sigma$ and the spread of your held-out actuals, and it
tells a reader how much headroom the number had. Second, if you can afford
multiple seeds on the held-out subsets, $\tau$ drops to $\sigma/\sqrt{n}$ and the
ceiling rises; at $n = 3$ the ceiling above moves from 0.97 to 0.990.

**Spearman rather than Pearson.** The formula
$\rho = 1 - 6\sum d_j^2 / (J(J^2-1))$ is Pearson correlation computed on ranks,
which is why it can be written without any means or variances - for a
permutation of $1..J$ the mean and variance of the ranks are known constants and
the algebra collapses. Ranks are the right choice here for a specific reason
rather than out of habit: a method can be systematically miscalibrated in scale
(predicting bpb changes twice as large as they are) while ordering subsets
perfectly, and for choosing which method to trust the ordering is what matters.
The cost is that Spearman is blind to exactly that miscalibration, so if you
intend to use predicted magnitudes for money rather than for ranking, report a
calibration slope alongside it.

## What ground truth buys

**What "no better than random guessing" means as a number.** Under the null that
a method's scores are independent of the truth, Spearman's $\rho$ on $J$
held-out subsets has mean 0 and standard deviation approximately
$1/\sqrt{J-1}$. At $J = 50$ that is $0.143$, so the 95% range under the null is
roughly $\pm 0.28$. A reported LDS of 0.15 on 50 subsets is *not* evidence of
anything, and a reported LDS with no $J$ attached is uninterpretable. When the
literature says a scalable method performs no better than random against ground
truth, that is the claim that its measured correlation sits inside this band.

The complementary calculation is what you need to detect a real effect: to
distinguish an LDS of $\rho$ from zero at $z = 2$ you need
$J \gtrsim 1 + 4/\rho^2$. For $\rho = 0.3$ that is 45 held-out subsets; for
$\rho = 0.15$, 179. Budget the held-out fraction of your subset runs against the
smallest correlation you would still act on.

**The cost accounting behind the dilemma.** The method that reaches $\rho = 0.97$
against retraining ground truth costs 3 to 5 full training runs per test sample.
Write $R_{\text{train}}$ for one training run's cost. For $Q$ test queries the
oracle costs $\approx 4QR_{\text{train}}$, while the counterfactual sweep in this
unit costs $((K+1)n + M)R_{\text{train}}$ *once*, independent of $Q$. At
$K = 12$, $n = 3$, $M = 300$ the sweep is 339 runs, so it breaks even against the
oracle at $Q \approx 85$ queries and is cheaper thereafter forever. The
per-source table is an amortizing asset and the per-query oracle is not - which
is the structural reason a compensation system wants source-level counterfactuals
and not query-level influence, quite apart from any argument about fairness.

## At the bench: the leave-one-out sweep

**The cache, and how much it is worth.** Every run you perform produces a pair
$(S, v(S))$, and $S$ is a set of at most $K$ source identifiers. Key the cache by
a canonical hash of the sorted source set together with the seed, and store the
held-out bpb, the protocol, and the configuration hash. Two things then follow.

First, v5's exact Shapley computation over $K$ sources needs $v(S)$ for all
$2^K$ subsets. At $K = 12$ that is 4,096 coalitions. Your leave-one-out sweep
supplies 13 of them and your $M = 300$ subset runs supply up to 300 more - about
7.6% of the enumeration, free. Small, but the overlap is not the point: the point
is that the cache makes the marginal cost of a coalition equal to one training
run rather than one training run plus rediscovering that you already had it.

Second, the number of *distinct* subsets you draw is not $M$ if you sample
carelessly. Drawing $M$ subsets uniformly from $2^K$ with replacement, the
expected number of distinct values is

$$2^K\left(1 - \left(1 - 2^{-K}\right)^{M}\right) \approx M - \frac{M^2}{2^{K+1}}$$

At $K = 12$ and $M = 300$ the expected number of collisions is
$300^2/8192 = 11$. Eleven wasted runs out of 300 is 3.7% of your budget, and it
is avoidable for free by sampling without replacement. At $K = 8$ the same
formula gives 176 collisions out of 300, which is a disaster; at small $K$,
enumerate rather than sample.

**Estimating $\sigma$ from the sweep you already ran.** Do not spend runs on a
dedicated noise measurement. The $(K+1)n$ leave-one-out runs already give you 26
degrees of freedom of pooled within-condition variance, as derived in section 3.
The single-seed subset runs give you nothing directly - but they give you the
regression residuals, and comparing the residual standard deviation against the
pooled $\sigma$ is the additivity test from section 6: residuals materially
larger than $\sigma$ mean the linear model is missing interaction structure,
which is a finding rather than a defect.

## What you can now do

**The three estimators, side by side.** Everything in this unit reduces to three
formulas with the same shape - a difference of averages, divided by its own
standard error.

| Estimator | Point estimate | Standard error | Runs required | Estimand |
|---|---|---|---|---|
| Leave-one-out | $\bar v(\text{all}\setminus i) - \bar v(\text{all})$ | $\sigma\sqrt{2/n}$ | $(K+1)n$ | $-(\beta_i + \sum_j\gamma_{ij})$ |
| Subset regression | $\bar y_{\{x_i=1\}} - \bar y_{\{x_i=0\}}$, negated | $\dfrac{\sigma}{\sqrt{M}}\dfrac{1}{\sqrt{p(1-p)}}$ | $M$ | $-(\beta_i + p\sum_j\gamma_{ij})$ |
| Interaction diagnostic | $\Delta_i - c_i$ | $\sqrt{\text{SE}_\Delta^2 + \text{SE}_\beta^2}$ | none extra | $-(1-p)\sum_j \gamma_{ij}$ |

The third row is free and almost nobody computes it. It is the cheapest evidence
you will ever get about whether the additivity assumption underneath every
proportional payout scheme actually holds on your corpus, and it is the direct
bridge into v5: a corpus whose interaction diagnostic is uniformly near zero can
be paid out proportionally and Shapley will agree; a corpus with large entries
cannot, and the size of those entries is how much money the choice of rule
moves.
