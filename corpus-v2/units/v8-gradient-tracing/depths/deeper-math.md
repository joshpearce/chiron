---
unit: v8
depth: deeper-math
---

## The bill for one honest answer

The canon prices a per-query counterfactual and calls the result absurd. Two
parts of that pricing deserve to be exact.

**Where $6ND$ is wrong, and by how much.** The count of 6 operations per
parameter per token covers only the work proportional to parameter count.
Attention's score matrix is quadratic in sequence length and does not touch a
weight, so the forward pass performs an extra $2 L d_{\text{model}}$ operations
per token for context length $L$, plus the same again on the value-weighted sum,
and the backward pass roughly doubles it. As a fraction of $6ND$ that is
approximately

$$\frac{\text{attention term}}{6ND} \approx \frac{L}{12\, d_{\text{model}}}$$

At $L = 1{,}024$ and $d_{\text{model}} = 768$ the ratio is 0.11 - the narrow
models in this book undercount by about 11%. The saving grace is that the
throughput figure you divide by is not a hardware peak but an effective rate
fitted so that $6ND$ reproduces observed wall clocks at similar context length,
so the attention term is already absorbed. Keep the pairing intact and refit if
context length moves by an order of magnitude.

**Why a per-output counterfactual needs more seeds than a per-corpus one.** The
contribution table's value function is a mean over the held-out file. If the
per-token log-loss has standard deviation $s$ across tokens and the file has $T$
tokens, the standard error of the mean is $s/\sqrt{T}$, and at $T \sim 10^6$
that is a thousandfold reduction. A single output's log-probability is a sum
over its own handful of tokens, so $T$ is perhaps 20 rather than $10^6$ and the
same reduction is a factor of 4.5. The seed-to-seed standard deviation of the
quantity you are differencing is therefore two orders of magnitude larger,
relative to the effect, than it was for the table. Since the standard error of a
difference of two condition means is

$$\text{SE}_\Delta = \sigma\sqrt{\tfrac{2}{n}}$$

with $n$ seeds per condition, holding the same significance bar at 200 times the
$\sigma$ would need $200^2 = 40{,}000$ times the seeds. The canon's "3 seeds and
know you are being generous" is doing a lot of work; the honest version of the
per-query counterfactual is not merely expensive, it is unaffordable by a
further four orders of magnitude. This is the strongest single reason the
per-query question has to be answered by an estimator rather than by
measurement, and it is worth carrying separately from the dollar figure.

## A gradient is a direction

The canon's first-order claim is stated as an approximation. On the toy model it
can be made an identity, and the remainder term written down exactly, which is
the cleanest way to see what "second order in $\eta$" means.

Write the toy example's loss as $\ell_A(w) = (w \cdot x_A - y_A)^2$, where
$x_A$ is the input vector and $y_A$ the target. Move the parameters by an
arbitrary $\Delta w$ and expand:

$$\ell_A(w + \Delta w) - \ell_A(w) = 2(w\cdot x_A - y_A)\,(x_A \cdot \Delta w) + (x_A \cdot \Delta w)^2$$

Exactly, with no remainder, because the loss is quadratic. The first term is
$g_A \cdot \Delta w$, since $g_A = 2(w\cdot x_A - y_A)\,x_A$. The second term is
the entire curvature correction. Substituting the gradient step
$\Delta w = -\eta\, g_B$:

$$\Delta \ell_A = -\eta\,(g_A \cdot g_B) + \eta^2 (x_A \cdot g_B)^2$$

Check against the canon's numbers. $x_A = (2, 0)$, $g_B = (-2, 0)$, so
$x_A \cdot g_B = -4$. At $\eta = 0.1$: $-0.1(8) + 0.01(16) = -0.8 + 0.16 =
-0.64$. At $\eta = 0.01$: $-0.08 + 0.0001(16) = -0.08 + 0.0016 = -0.0784$. Both
exact to every digit the canon reported.

Two things are now visible that were assertions before. The error term is
$\eta^2$ times something that does not depend on $\eta$, so shrinking the step
by 10 shrinks the relative error by 10. And the error term is **strictly
positive here**, so the first-order prediction systematically overstates how
much the loss falls. That is not a coincidence of these numbers: the correction
is $\tfrac{1}{2}\Delta w^{\top} H_A \Delta w$ where $H_A$ is the Hessian of
$\ell_A$, and for a squared error $H_A = 2\,x_A x_A^{\top}$ is positive
semidefinite. Along a convex-in-this-direction loss, a straight-line prediction
always overshoots the improvement. In a real network the loss is not convex and
$H$ has negative eigenvalues, so the bias goes both ways, but the magnitude
still scales as $\eta^2 \lVert H \rVert$.

**Magnitude versus alignment.** The dot product is
$g_A \cdot g_B = \lVert g_A\rVert \lVert g_B \rVert \cos\theta$, so it conflates
two facts: how well the examples agree, and how loud they are. An example with a
large residual has a large gradient and will outscore a well-fit example
pointing in exactly the same direction. Whether that is a bug depends on the
question. For predicting loss change it is correct - a loud example really does
move the parameters further. For "which source is most aligned with this
query" it is a confound, and the fix is to normalize:
$\cos\theta = (g_A\cdot g_B)/(\lVert g_A\rVert\lVert g_B\rVert)$. Production
systems in this literature use unit-normalized or whitened variants for exactly
this reason, and they are answering a different question than raw TracIn.

## TracIn: influence as a sum over checkpoints

**The telescoping identity, written out.** Let $\theta_0, \theta_1, \ldots,
\theta_T$ be the parameters after each of $T$ training steps. The total change
in the query's loss over the whole run is a sum of per-step changes and nothing
is lost:

$$\ell(\theta_T, z_q) - \ell(\theta_0, z_q) = \sum_{t=0}^{T-1}\big[\ell(\theta_{t+1}, z_q) - \ell(\theta_t, z_q)\big]$$

That is exact - it is the same identity as $\sum (a_{t+1} - a_t)$ collapsing.
The approximation enters only in evaluating each bracket. If step $t$ took an
SGD step on a single example $z_t$, then to first order the bracket is
$-\eta_t\, g_{z_q}^{(t)} \cdot g_{z_t}^{(t)}$, and attributing each bracket to
the example that caused it gives the **idealized** influence

$$\text{TracInIdeal}(z, z_q) = \sum_{t \,:\, z_t = z} \eta_t\; g_{z_q}^{(t)} \cdot g_{z}^{(t)}$$

with the sum running over the steps where $z$ was actually visited. Every
example's influences add up to the total loss change, which is the property that
makes this a decomposition rather than a score.

**What the checkpoint approximation actually assumes**, since it is not a
Riemann sum. Between two saved checkpoints there are thousands of steps, and $z$
was visited on some of them. The checkpoint form replaces every visit in that
interval with one evaluation at the checkpoint, weighted by $\eta_c$. That is
sound when (a) parameters change little across the interval, so the gradient at
the checkpoint stands in for the gradients at the visits, and (b) each example
was visited the **same number of times** in the interval, so the implicit
per-example weighting is uniform. Condition (b) is quietly violated by any
corpus with repeated data: an example appearing 4 times per epoch contributes 4
visits per interval and one checkpoint evaluation, so it is underweighted by a
factor of 4 relative to a once-seen example. If your data-constrained recipe
repeats a small corpus for several epochs, weight each source's checkpoint term
by its visit count in the interval, or you have silently paid the
frequently-repeated sources one quarter of what the identity says they earned.

**Batches.** A real step uses a batch $B_t$, so $\Delta\theta_t = -\eta_t
\sum_{z\in B_t} g_z$ (with a $1/|B_t|$ if the loss is averaged rather than
summed, which folds into $\eta_t$). The first-order expansion is linear in
$\Delta\theta$, so the bracket splits exactly into one term per batch member and
the per-example attribution is unchanged. Nothing about batching costs you
anything at first order; the second-order cross terms it introduces are what the
in-run section allocates.

**Self-influence.** Setting $z_q = z$ gives $\sum_c \eta_c \lVert g_z^{(c)}
\rVert^2 \geq 0$, which is nonnegative by construction and is large exactly for
examples the model keeps being surprised by. It is the standard
mislabelled-data detector in this literature, and it is worth knowing that a
"high influence" example flagged with $z_q = z$ is measuring outlierness, not
importance.

## Making it affordable

**The Johnson-Lindenstrauss statement, with the variance.** Let $P$ have $d$
rows, each row $p_k \in \mathbb{R}^N$ with independent standard Gaussian
entries, and scale the whole matrix by $1/\sqrt{d}$. Then for fixed $u, v \in
\mathbb{R}^N$:

$$(Pu)\cdot(Pv) = \frac{1}{d}\sum_{k=1}^{d} (p_k \cdot u)(p_k \cdot v)$$

Each term is unbiased, since for a standard Gaussian vector $p$,
$\mathbb{E}[(p\cdot u)(p\cdot v)] = \sum_{i,j} u_i v_j \mathbb{E}[p_i p_j] =
\sum_i u_i v_i = u\cdot v$, using $\mathbb{E}[p_ip_j] = \delta_{ij}$. So
$\mathbb{E}[(Pu)\cdot(Pv)] = u\cdot v$ exactly, at any $d$.

The variance of a single term, for jointly Gaussian $(p\cdot u, p\cdot v)$, is

$$\mathrm{Var}\big[(p\cdot u)(p\cdot v)\big] = \lVert u\rVert^2 \lVert v\rVert^2 + (u\cdot v)^2$$

and averaging $d$ independent terms divides it by $d$:

$$\mathrm{SD}\big[(Pu)\cdot(Pv)\big] = \sqrt{\frac{\lVert u\rVert^2\lVert v\rVert^2 + (u\cdot v)^2}{d}} \;\approx\; \frac{\lVert u\rVert \lVert v\rVert}{\sqrt{d}}$$

where the approximation uses $|u\cdot v| \ll \lVert u\rVert\lVert v\rVert$,
which is the typical case in high dimension. That is the canon's $1/\sqrt{d}$,
and at $d = 4{,}096$ it is $1/64 = 1.6\%$ of $\lVert u\rVert\lVert v\rVert$.

**Now the part that decides what projected TracIn can and cannot claim.** Write
the true dot product as $u \cdot v = \rho_{uv}\lVert u\rVert\lVert v\rVert$ with
$\rho_{uv} = \cos\theta$. The signal-to-noise ratio of the projected estimate is

$$\text{SNR} \approx \frac{\rho_{uv}\lVert u\rVert\lVert v\rVert}{\lVert u\rVert\lVert v\rVert/\sqrt{d}} = \rho_{uv}\sqrt{d}$$

The norms cancel completely. At $d = 4{,}096$, $\sqrt d = 64$, so a pair with
$\cos\theta = 0.1$ has SNR 6.4 - comfortably resolved - and a pair with
$\cos\theta = 0.005$ has SNR 0.32, which is noise. **The resolvable alignment
floor is $\cos\theta \approx 1/\sqrt d$.** Push $d$ to 16,384 and the floor
improves only by a factor of 2, because the dependence is a square root. This is
why the technique separates top contributors reliably and cannot produce a
trustworthy full distribution over a corpus, and it is why the sparsity of
influence found in the large-scale studies is what makes the whole approach
viable rather than being an inconvenient detail.

**Per-source accumulation is exact, not an approximation.** $P$ is linear, so

$$\sum_{z \in i} P g_z = P\Big(\sum_{z\in i} g_z\Big)$$

Projecting each gradient then summing, and summing then projecting, give the
identical vector. There is no statistical cost to accumulating - only the loss
of per-sequence resolution. All of the approximation lives in $P$ and none of it
in the accumulation. Note also that the accumulated source vector has a larger
norm than any individual gradient while the noise floor scales with that same
norm, so the alignment floor $1/\sqrt d$ applies unchanged at source level: you
get source rankings at the same fidelity you would have gotten per-sequence,
for a millionth of the storage.

**Sparse and structured alternatives.** Gaussian $P$ is never materialized in
practice. Rademacher entries ($\pm 1/\sqrt d$) have the same first two moments
and are cheaper. Sparse variants keep a fraction of nonzeros. Structured
transforms (a random sign flip followed by a fast Hadamard transform and a
subsample) compute $Pg$ in $O(N \log N)$ instead of $O(Nd)$ with essentially the
same guarantee, which matters because the projection, not the backward pass, is
the bottleneck once $d$ is in the thousands.

## The reckoning

**How large a rank correlation has to be at $n = 12$.** The canon says to
rank-correlate your method against the twelve-source table. That is twelve data
points, and a Spearman correlation on twelve points has a wide null
distribution. Under the null of independent rankings, $\rho$ is approximately
normal with mean 0 and standard deviation $1/\sqrt{n-1}$, which at $n = 12$ is
$1/\sqrt{11} = 0.30$. The usual test statistic is

$$t = \rho\sqrt{\frac{n-2}{1-\rho^2}}$$

compared against a $t$ distribution with $n-2$ degrees of freedom. At $n=12$,
two-sided $\alpha = 0.05$ requires $|\rho| > 0.58$. So a measured $\rho = 0.71$
gives $t = 0.71\sqrt{10/0.496} = 3.19$ on 10 degrees of freedom, $p \approx
0.01$ - real, but not comfortable. A measured $\rho = 0.45$ on twelve sources
is **not distinguishable from chance** and must not be reported as evidence the
method works.

This is the quantitative argument for grading against held-out subsets rather
than against the twelve-row table. With $J = 60$ held-out subset runs the null
SD drops to $1/\sqrt{59} = 0.13$ and the significance threshold falls to about
$\rho > 0.25$. Same effort in method implementation, four times the resolution,
which is why the Linear Datamodeling Score is defined over subsets. Report both,
and note that the twelve-source correlation is the one your audience will find
intuitive and the subset one is the one that carries statistical weight.

**The optimizer correction as a change of metric.** The corrected summand

$$\langle g_{z_q}, g_z\rangle_M = \sum_j \frac{g_{z_q,j}\,g_{z,j}}{\sqrt{v_j}+\epsilon}$$

is an inner product with respect to the diagonal positive-definite matrix
$M = \mathrm{diag}\big(1/(\sqrt{v_j}+\epsilon)\big)$. That is the precise
statement of why it reorders: it is a different geometry on parameter space, and
"which examples point the same way" is a question whose answer depends on the
metric. The same observation underlies natural-gradient methods, where the
metric is the Fisher information rather than AdamW's diagonal second moment, and
it is the reason the influence-function family's $H^{-1}$ can be read as yet
another choice of metric - the one the loss surface's own curvature induces.
Seen this way the three methods are one formula under three metrics: identity
(TracIn as usually written), AdamW's diagonal (the corrected form), and $H^{-1}$
(influence functions).

**The influence-function formula, and what is actually estimated.** Upweight
training example $z$ by $\varepsilon$ in the training objective and let
$\theta^*_\varepsilon$ be the new optimum. Differentiating the stationarity
condition $\sum_{z'} \nabla_\theta \ell(\theta^*_\varepsilon, z') + \varepsilon
\nabla_\theta\ell(\theta^*_\varepsilon, z) = 0$ with respect to $\varepsilon$
and evaluating at 0 gives

$$\frac{d\theta^*_\varepsilon}{d\varepsilon}\bigg|_{0} = -H^{-1}\nabla_\theta \ell(\theta^*, z)$$

and chaining through the query's loss gives the canon's formula. The derivation
uses the implicit function theorem at a stationary point and requires $H$
invertible, hence strictly convex. For a network, $H$ at the end of training has
many near-zero and some negative eigenvalues, so $H^{-1}$ does not exist and
every practical method substitutes a damped surrogate $(H + \lambda I)^{-1}$ or a
Kronecker-factored approximation. The result proved by Bae and colleagues is
that what such an estimator converges to, for a damped objective, is the
derivative of the **proximal Bregman response function** rather than the
leave-one-out difference - a well-defined and different quantity. That is the
technical content of "influence functions do not estimate leave-one-out", and it
is why the negative empirical results are not a matter of insufficient
approximation quality. Unit x1 develops it.

## Influence is not entailment

The two targets are different estimands and the cleanest way to hold them apart
is to write both.

**Influence** is a derivative of the model's behaviour with respect to the data:

$$\mathcal{I}(z, z_q) \;=\; \ell\big(\theta^*(\mathcal{D}\setminus z), z_q\big) - \ell\big(\theta^*(\mathcal{D}), z_q\big)$$

where $\mathcal{D}$ is the corpus and $\theta^*(\cdot)$ is the training
procedure applied to it. Every symbol refers to a training run. The model
appears twice and the text of $z$ appears nowhere.

**Entailment** is a function of two texts and no model at all. BM25 scores a
document $D$ against a query $Q$ as

$$\text{BM25}(D, Q) = \sum_{w \in Q} \text{IDF}(w)\,\frac{f(w, D)\,(k_1+1)}{f(w,D) + k_1\big(1 - b + b\,\frac{|D|}{\bar{L}}\big)}$$

where $f(w,D)$ counts occurrences of term $w$ in $D$, $|D|$ is the document
length, $\bar L$ the mean document length, $\text{IDF}(w)$ is an inverse
document frequency weight rewarding rare terms, and $k_1, b$ are constants
controlling term-frequency saturation and length normalization. There is no
$\theta$ in that expression. It cannot be an estimate of the previous one; it is
a measurement of a different object.

The empirical finding - that the second beats the first at retrieving the
fact-containing document at 8-billion-parameter scale - is therefore not
evidence that the gradient estimator is broken. It is evidence that the
benchmark's target is entailment, and that a direct measurement of entailment
beats an indirect one. The corresponding statement in the other direction has no
benchmark, because measuring influence properly requires retraining, which is
why nobody has built the benchmark on which BM25 would lose.

## The ledger inside the run

**The second-order expansion, and where Shapley starts doing work.** Take one
step on batch $B$ with $\Delta\theta = -\eta \sum_{z\in B} g_z$ and expand the
validation loss to second order:

$$\Delta L_{\text{val}} = g_{\text{val}} \cdot \Delta\theta + \tfrac{1}{2}\,\Delta\theta^{\top} H_{\text{val}}\,\Delta\theta + O(\lVert\Delta\theta\rVert^3)$$

where $H_{\text{val}}$ is the Hessian of the validation loss at the current
parameters. Substituting and flipping sign so positive means helpful, the value
created by this step is

$$v(B) = \eta \sum_{z \in B} g_{\text{val}}\cdot g_z \;-\; \frac{\eta^2}{2}\sum_{z \in B}\sum_{z' \in B} g_z^{\top} H_{\text{val}}\, g_{z'}$$

Restricting to a sub-coalition $S \subseteq B$ - "what would this step have
achieved with only these examples in it" - gives a value function of the form

$$v(S) = \sum_{i \in S} a_i + \sum_{\{i,j\} \subseteq S} b_{ij}$$

with $a_i = \eta\, g_{\text{val}}\cdot g_i - \tfrac{\eta^2}{2} g_i^\top H
g_i$ and $b_{ij} = -\eta^2 g_i^\top H g_j$. That is a game with singleton and
pairwise terms only, and its Shapley value has a closed form:

$$\phi_i = a_i + \tfrac{1}{2}\sum_{j \neq i} b_{ij}$$

The proof is two lines. Shapley is additive over games, so decompose $v$ into
one game per term. A game consisting of the single term $a_i$ pays $a_i$ to $i$
and 0 to everyone else, by the null-player and efficiency requirements. A game
consisting of the single term $b_{ij}$ has only players $i$ and $j$ as
non-null, they are interchangeable in it, so symmetry and efficiency split it
$b_{ij}/2$ each. Sum the pieces.

Drop the second-order terms and $b_{ij} = 0$, so $\phi_i = a_i =
\eta\,g_{\text{val}}\cdot g_i$, which is the canon's first-order result and is
identical to TracIn's summand with $g_{\text{val}}$ in the query slot. **The
axioms contribute exactly one thing to the first-order method: nothing.** They
begin to matter only at second order, where the even split of each pairwise
interaction is a genuine choice that the axioms force.

**The ghost inner product.** Computing $g_{\text{val}}\cdot g_z$ for every $z$ in
a batch looks like it needs per-example gradients, which are the thing you cannot
afford to materialize. For a linear layer they are never needed. Let the layer
map input activation $a$ to output $h = Wa$. Backpropagation already computes,
per example, the input activation $a$ and the gradient of the loss with respect
to the output, $\delta = \partial \ell / \partial h$. The per-example gradient
with respect to $W$ is the outer product $g = \delta\, a^{\top}$, and the
Frobenius inner product of two such gradients factors:

$$\langle \delta_1 a_1^{\top},\; \delta_2 a_2^{\top}\rangle_F = \mathrm{tr}\big(a_1\delta_1^{\top}\delta_2 a_2^{\top}\big) = (\delta_1\cdot\delta_2)\,(a_1\cdot a_2)$$

Two small dot products, in the layer's output width and input width
respectively, and the $d_{\text{in}} \times d_{\text{out}}$ outer product is
never formed. Both $a$ and $\delta$ are already in memory during the backward
pass. That factorization is why the overhead is a modest percentage rather than
a multiple, and it is why the technique is specific to layers whose gradients
have this outer-product structure - which is to say, essentially all of a
transformer's parameters.

**Cost of the second-order version.** The pairwise terms need
$g_i^{\top} H_{\text{val}} g_j$ for every pair in the batch. Hessian-vector
products cost roughly one extra backward pass each, and there are $O(|B|^2)$
pairs, so the second-order ledger is materially more expensive than the
first-order one's 7.5%. That cost, not the mathematics, is why the reported
headline overhead is for the first-order variant, and it is the number to budget
against.

## What to trust

The table is a decision rule, and it is worth seeing what makes it one.

You have a cheap estimator with measured rank correlation $\rho$ against ground
truth, and a choice between paying by estimated contribution and paying a flat
fee. The result the economics literature supplies is a threshold: below some
signal-to-noise ratio in the attribution estimate, the welfare-optimal contract
is the flat fee, and improving the estimator's *bias* does not help - only
improving its *signal* does. The structure of that result is worth holding even
without its derivation. A contract that pays by a noisy signal transfers risk to
the recipient without transferring information, and the recipient prices that
risk. Past some noise level, the risk premium exceeds the allocative gain from
paying differentially, and both parties prefer the flat fee. The threshold is a
property of the noise, not of the estimator's cleverness, which is why "we will
improve the method" is not a response to it.

That converts the canon's rule - never report a cheap number without its
correlation - from good manners into the only way to know which contract you are
entitled to offer. The measurement chain is: seed noise sets the resolution of
the ground-truth table; the table's resolution sets what rank correlations are
statistically distinguishable; the correlation sets the estimator's
signal-to-noise; and the signal-to-noise decides whether contribution-proportional
payment is welfare-improving at all. Every link is arithmetic you can do, and
the chain is the whole argument.

## At the bench: the gradient ledger

**Propagating the two error sources.** Your per-source TracIn score carries
noise from two independent places, and they do not combine the way people
assume.

*Projection noise* is the $1/\sqrt d$ term derived above. It is deterministic
given the seed - rerun the same index and you get the identical number - so it
is a bias for a fixed seed, not a variance you can average away by repeating the
computation. The only way to estimate it is to rebuild the index under several
seeds and look at the spread of the resulting rankings. Budget two extra seeds
for this; it costs gradient compute, not training compute.

*Trajectory noise* is the seed dependence of the training run itself, which is
the same $\sigma$ the counterfactual table measured. A TracIn score computed
from a different training run's checkpoints is a different number for the same
reason a leave-one-out delta is. This one you cannot cheapen: it requires
retraining.

Report the ranking's stability under both. A source whose rank moves by five
places when you change the projection seed has no rank.

**Sizing the grading design.** The correlation you can resolve is set by the
number of things you correlate over. Twelve sources gives a significance
threshold near $\rho = 0.58$; $J$ held-out subsets gives roughly $2/\sqrt{J-1}$.
Choose $J$ so that the threshold sits well below the correlation you expect to
find, and fix $J$ before you look at any results, because a $J$ chosen after
seeing the scatter is not a test. Sixty held-out subsets puts the threshold near
0.26 and is affordable from the runs already cached.

**Two arithmetic checks that catch most implementation bugs.** First,
self-influence must be nonnegative for every example, since it is a sum of
$\eta_c\lVert g\rVert^2$ terms; a negative self-influence means a sign error or
mismatched projections. Second, the first-order in-run ledger's twelve source
totals must sum to the total first-order predicted validation-loss reduction
over the run, which you can compute independently as $\sum_t \eta_t\,
g_{\text{val}}^{(t)} \cdot \bar g^{(t)}$ with $\bar g$ the batch mean gradient.
If they do not agree to within floating-point tolerance, an example is being
counted in the wrong source or a step is being missed.

## What you can now do

Every formula in the unit, with the assumption it rests on, in one place.

$$\Delta\ell(z_q) = -\eta\,(g_{z_q}\cdot g_z) + O(\eta^2\lVert H\rVert)$$
*Assumes:* nothing but differentiability. Exact to the stated order, and on a
quadratic loss the $O(\eta^2)$ term is $(x\cdot\Delta w)^2$ exactly.

$$\text{TracIn}(z,z_q) = \sum_{c\in\mathcal{C}} \eta_c\,\big(g_{z_q}^{(c)}\cdot g_z^{(c)}\big)$$
*Assumes:* the SGD update rule; uniform visit counts per checkpoint interval;
parameters roughly constant within an interval. The first is corrected by the
preconditioned form, the second by visit-count weighting, the third by
checkpoint spacing.

$$\mathrm{SD}\big[(Pu)\cdot(Pv)\big] \approx \lVert u\rVert\lVert v\rVert/\sqrt{d}
\qquad\Rightarrow\qquad \text{SNR} \approx \cos\theta\,\sqrt{d}$$
*Assumes:* $P$ has i.i.d. mean-zero unit-variance entries scaled by
$1/\sqrt d$, and the same $P$ for every vector. The alignment floor
$\cos\theta \approx 1/\sqrt d$ is what bounds every claim about the tail.

$$\phi_i = a_i + \tfrac{1}{2}\sum_{j\neq i} b_{ij}$$
*Assumes:* the per-step game has only singleton and pairwise terms, which holds
exactly for a second-order expansion. At first order $b_{ij}=0$ and the Shapley
machinery is inert.

$$t = \rho\sqrt{\tfrac{n-2}{1-\rho^2}}, \quad \text{threshold } |\rho| > 0.58 \text{ at } n=12$$
*Assumes:* independent rankings under the null. This is the number that decides
whether your headline correlation is a result or a coincidence, and it is the
one most likely to be skipped.
