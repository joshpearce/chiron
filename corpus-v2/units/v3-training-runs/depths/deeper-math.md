## The receipt

State the cost-quality independence formally, because it is the load-bearing
claim of the section and it is a statement about function arguments.

Let $\mathcal{D}$ be a dataset, $N$ a parameter count, $D$ a token count. Two
functions:

$$C = C(N, D) = 6ND \qquad\qquad \mathcal{L}_{\text{final}} = \mathcal{L}(N, D, \mathcal{D})$$

The cost function does not take $\mathcal{D}$ as an argument. That is the entire
content of "quality is free at fixed compute": along any level set
$\{(N,D) : 6ND = C_0\}$, the dataset can be varied without bound at zero cost.
Every dollar of improvement available from data work is therefore a strict
Pareto gain, which is why the largest measured effects in the corpus literature
come from filtering classifiers rather than architecture.

**Chinchilla's allocation, derived.** Given a fixed compute budget, how should
it be split between parameters and tokens? Take the empirical scaling form

$$\mathcal{L}(N, D) = \mathcal{L}_\infty + \frac{A}{N^{\alpha}} + \frac{B}{D^{\beta}}$$

where $\mathcal{L}_\infty$ is the irreducible entropy of the text, and $A, B,
\alpha, \beta$ are fitted positive constants. Substitute the constraint
$D = C/(6N)$ to make it a one-variable problem:

$$\mathcal{L}(N) = \mathcal{L}_\infty + A N^{-\alpha} + B\left(\frac{6N}{C}\right)^{\beta}$$

Differentiate and set to zero:

$$-\alpha A N^{-\alpha - 1} + \beta B \left(\frac{6}{C}\right)^{\beta} N^{\beta - 1} = 0
\quad\Longrightarrow\quad
N^{\alpha + \beta} = \frac{\alpha A}{\beta B}\left(\frac{C}{6}\right)^{\beta}$$

$$N_{\text{opt}} \propto C^{\frac{\beta}{\alpha + \beta}},
\qquad D_{\text{opt}} \propto C^{\frac{\alpha}{\alpha + \beta}}$$

With the Chinchilla fits $\alpha \approx 0.34$ and $\beta \approx 0.28$, the
exponents are $0.28/0.62 = 0.45$ and $0.34/0.62 = 0.55$ - close enough to
one half each that the headline simplification is "scale both together," and
the ratio $D/N$ is roughly constant at about 20 tokens per parameter.

Two things this makes precise. First, the famous "20 tokens per parameter" is
not a law of nature; it is the ratio of two fitted exponents that happen to be
nearly equal, on one corpus, at one point in history. Second, and more useful:
nobody trains compute-optimal any more, because the objective is wrong. Chinchilla
minimizes loss at fixed *training* compute. If the model will be served, total
cost includes inference, and the correct objective pushes toward smaller models
trained far longer - Llama-3-8B sits near 1,875 tokens per parameter, roughly
90 times past Chinchilla. For this project neither objective applies: the
constraint is *number of runs*, and minimizing wall clock per run at adequate
quality is a third optimization with a third answer.

## One step, from scratch

**The gradient of cross-entropy, at the logits.** This is the one derivation
worth having by hand, because it is why the backward pass is cheap.

Let $z \in \mathbb{R}^V$ be the logits at one position, $p = \text{softmax}(z)$,
and $y$ the one-hot indicator of the token that actually occurred. The
per-position loss is $\ell = -\log p_c$ where $c$ is the true token index.

$$p_j = \frac{e^{z_j}}{\sum_k e^{z_k}}, \qquad
\ell = -z_c + \log\sum_k e^{z_k}$$

Differentiate with respect to an arbitrary logit $z_j$:

$$\frac{\partial \ell}{\partial z_j} = -[j = c] + \frac{e^{z_j}}{\sum_k e^{z_k}} = p_j - y_j$$

$$\boxed{\nabla_z \ell = p - y}$$

No exponentials survive, no division survives. The gradient at the output is
the model's predicted distribution minus the truth - a subtraction. Everything
upstream is the chain rule applied to matrix multiplies, which is the next
paragraph.

**Why backward costs exactly twice forward.** Consider one linear layer, row-vector
convention: $Y = XW$ with $X \in \mathbb{R}^{n \times d_{\text{in}}}$,
$W \in \mathbb{R}^{d_{\text{in}} \times d_{\text{out}}}$. Forward is one matrix
multiply, costing $2 n\, d_{\text{in}} d_{\text{out}}$ FLOPs (one multiply and
one add per inner-product term).

Backward needs two things, and each is one matrix multiply of the same size:

$$\frac{\partial \mathcal{L}}{\partial X} = \frac{\partial \mathcal{L}}{\partial Y} W^\top
\qquad\qquad
\frac{\partial \mathcal{L}}{\partial W} = X^\top \frac{\partial \mathcal{L}}{\partial Y}$$

The first propagates blame to the previous layer; the second is the parameter
gradient. Both contract over one axis of the same three tensors, so both cost
$2 n\, d_{\text{in}} d_{\text{out}}$. Backward is $4 n\, d_{\text{in}}
d_{\text{out}}$, forward is $2$, and the ratio is exactly 2 for every
parameter-bearing operation in the network. That is the origin of the 6, and it
is not an empirical constant - it is the count of matrix multiplies.

**AdamW, and why the step size is approximately $\eta$.** With gradient $g_t$
for one parameter:

$$m_t = \beta_1 m_{t-1} + (1-\beta_1) g_t, \qquad
v_t = \beta_2 v_{t-1} + (1-\beta_2) g_t^2$$

$$\hat m_t = \frac{m_t}{1 - \beta_1^t}, \qquad
\hat v_t = \frac{v_t}{1 - \beta_2^t}, \qquad
\theta_t = \theta_{t-1} - \eta\left(\frac{\hat m_t}{\sqrt{\hat v_t} + \epsilon} + \lambda \theta_{t-1}\right)$$

The bias corrections divide by $1 - \beta^t$ because $m_t$ starts at zero and
takes $O(1/(1-\beta))$ steps to reach the running average's true scale.

Now the claim from canon. Suppose the gradient for this parameter is stationary
with mean $\mu$ and standard deviation $s$ across batches. Then
$\hat m_t \to \mu$ and $\hat v_t \to \mu^2 + s^2$, so

$$\frac{\hat m_t}{\sqrt{\hat v_t}} \to \frac{\mu}{\sqrt{\mu^2 + s^2}} \in [-1, 1]$$

The bracket is bounded by 1 in magnitude regardless of how large or small $g$
is. The per-step distance a parameter travels is therefore at most $\eta$ and
typically a decent fraction of it, which is why a learning rate is a meaningful
number to quote in isolation and to transfer between projects. It also explains
the failure mode: a parameter whose gradient is pure noise ($\mu \approx 0$,
$s > 0$) has $\hat m / \sqrt{\hat v} \approx 0$ and correctly stops moving,
while under plain SGD it would random-walk with step size $\eta s$.

The $\lambda \theta$ term is decoupled weight decay - the W in AdamW. It is
outside the normalization deliberately, so decay strength does not depend on
gradient magnitude.

## The one line: C = 6ND

**Full parameter accounting for a decoder-only transformer.** One layer, width
$d$, MLP expansion factor 4, no biases:

$$\underbrace{4d^2}_{W_Q, W_K, W_V, W_O} + \underbrace{4d^2 + 4d^2}_{\text{MLP up, down}} = 12d^2$$

so a stack of $L$ layers holds $N_{\text{stack}} \approx 12 L d^2$ parameters,
plus $Vd$ for the embedding table.

**The attention term, counted properly.** Attention performs two matrix
multiplies that involve no parameters: scores $QK^\top$ and the value
aggregation $AV$. For one token attending over a context of $n$ positions, each
of these contracts a $d$-dimensional vector against $n$ others, costing $2nd$
FLOPs, summed across all heads. Causal masking means the average token attends
over $n/2$ positions, so per layer per token the forward cost is about

$$2 \times 2 \times \tfrac{n}{2} \times d = 2nd$$

Forward per token, whole model: $2 N_{\text{stack}} + 2Lnd = 24Ld^2 + 2Lnd$.
Multiply by 3 for forward plus backward:

$$C_{\text{per token}} = 72Ld^2 + 6Lnd$$

The fraction $6ND$ misses is the ratio of the second term to the first:

$$\frac{6Lnd}{72Ld^2} = \frac{n}{12d}$$

Check it at both ends. A narrow model in this volume, $d = 512$, $n = 1024$:
$1024/6144 = 16.7\%$. A frontier-width model, $d = 12{,}288$, $n = 2048$:
$2048/147{,}456 = 1.4\%$. The approximation is excellent where it was
popularized and mediocre where you will actually use it, which is why the canon
section pairs it with an effective throughput rather than a hardware number.

**Why the pairing is self-consistent.** Model FLOPs Utilization is defined with
$6ND$ in the numerator by convention:

$$\text{MFU} = \frac{6ND / t}{P_{\text{peak}}}$$

where $t$ is elapsed seconds and $P_{\text{peak}}$ is the hardware's dense peak.
So MFU deliberately does *not* count the attention term, activation
recomputation, or anything else. Rearranged, it is a definition of effective
throughput:

$$t = \frac{6ND}{\text{MFU} \times P_{\text{peak}}}$$

Every uncounted operation is absorbed into a lower measured MFU. A budget built
from $6ND$ and an MFU measured on a similar configuration is therefore exact,
not approximate - the two errors are the same error, entered once with a minus
sign and once with a plus. (The distinct quantity Hardware FLOPs Utilization,
HFU, counts everything the hardware actually did, including the recomputation
that activation checkpointing forces. HFU exceeds MFU on any checkpointed run
and is the right number for judging kernels, not for budgeting.)

**Sensitivity.** Since $t \propto ND / \text{MFU}$, the elasticity of cost with
respect to each factor is exactly 1. Halving $N$, halving $D$, and doubling MFU
are worth precisely the same amount. The asymmetry is in what they cost you:
halving $N$ or $D$ changes the experiment, doubling MFU does not.

## Knobs are recipes, not theory

**The schedules, written out.** With $t$ the step index and $T$ the total:

$$\eta^{\text{cos}}_t = \eta_{\min} + \tfrac{1}{2}(\eta_{\max} - \eta_{\min})\left(1 + \cos\frac{\pi t}{T}\right)$$

$$\eta^{\text{WSD}}_t = \begin{cases}
\eta_{\max}\, t / T_w & t < T_w \quad\text{(warmup)}\\
\eta_{\max} & T_w \le t < T_s \quad\text{(stable)}\\
\eta_{\max}\, f\!\left(\frac{t - T_s}{T - T_s}\right) & t \ge T_s \quad\text{(decay)}
\end{cases}$$

with $f$ decreasing from 1 to ~0; linear and $1/\sqrt{\cdot}$ shapes both work.

Note that $T$ appears inside $\eta^{\text{cos}}_t$ and does not appear in the
stable phase of WSD. That is the whole formal content of "cosine must know its
length in advance" and of the branchability argument: on the stable segment the
optimizer's update rule is time-invariant, so the joint state
$(\theta_t, m_t, v_t)$ is a sample from a stationary distribution and any two
checkpoints in that segment are exchangeable as starting points. Under cosine
the update rule is a function of $t$, and a checkpoint carries an implicit
timestamp that its continuation must honor.

**Batch and learning rate, from gradient noise.** Let $g$ be the true full-batch
gradient and $\hat g_B$ the estimate from a batch of $B$ examples, so
$\mathbb{E}[\hat g_B] = g$ and $\text{Cov}(\hat g_B) = \Sigma / B$ for a
per-example covariance $\Sigma$. The SGD update $\theta \leftarrow \theta -
\eta \hat g_B$ has a deterministic part $-\eta g$ and a noise part with
covariance $\eta^2 \Sigma / B$. The ratio of noise energy to signal energy per
unit of progress is governed by $\eta / B$; holding that constant under
$B \to kB$ requires $\eta \to k\eta$. That is linear scaling, and it is a
derivation rather than a heuristic.

For Adam-family optimizers the update is divided by $\sqrt{\hat v} \approx
\sqrt{\mu^2 + s^2/B}$, which itself shrinks as $B$ grows, so part of the
correction happens automatically and the residual empirical rule is
$\eta \to \sqrt{k}\,\eta$. The two rules coincide only in the noise-dominated
limit and diverge as batches grow.

**Critical batch size.** Both rules assume the noise term matters. Define the
gradient noise scale

$$B_{\text{crit}} \approx \frac{\text{tr}(\Sigma)}{g^\top H g}$$

with $H$ the local Hessian. For $B \ll B_{\text{crit}}$, per-step progress is
noise-limited and grows nearly linearly in $B$; for $B \gg B_{\text{crit}}$ the
gradient is already accurate and doubling $B$ doubles cost for negligible gain.
Every scaling rule above is valid only below $B_{\text{crit}}$, which is why
they are quoted with a caveat rather than as laws.

**Muon, as actual math rather than a name.** AdamW normalizes each parameter
independently, which is a per-coordinate operation and ignores that a weight
matrix has structure. Muon asks a different question: given the momentum matrix
$M$, what is the update of bounded spectral norm that maximizes progress along
$M$?

$$\arg\max_{\|X\|_2 \le 1} \langle M, X\rangle = UV^\top, \qquad \text{where } M = U\Sigma V^\top$$

That is, the answer is the orthogonal factor of $M$ - the SVD with every
singular value replaced by 1. Computing an SVD every step is unaffordable, so
Muon approximates $UV^\top$ with a few iterations of a Newton-Schulz
polynomial, which needs only matrix multiplies.

What this does, in words: momentum matrices in practice are dominated by a few
large singular directions, so an unnormalized step moves almost entirely along
them. Orthogonalizing equalizes the singular values, so the update pushes in all
directions of the matrix at comparable magnitude. Empirically it converges in
fewer steps. It is applied only to 2D parameters; embeddings, biases, and
normalization gains stay on AdamW, because the argument above is about matrices.

## What repeating data actually costs

The three thresholds in canon - free to about 4 epochs, diminishing to about 16,
dead past about 40 - are not three independent measurements. They fall out of a
single fitted formula, and deriving them from it is worth ten minutes.

Let $U_D$ be the number of *unique* tokens and $R_D$ the number of *repetitions*
beyond the first pass, so epochs $E = R_D + 1$. The data-constrained scaling
work fits an effective data count:

$$D' = U_D + U_D R_D^{*}\left(1 - e^{-R_D / R_D^{*}}\right)$$

with a fitted decay constant $R_D^{*} \approx 15$. Read the pieces: the first
term is the unique tokens, seen once. The second is the value of every
subsequent pass, decaying exponentially with a scale of about 15 repetitions.

**The marginal value of one more pass** is the derivative:

$$\frac{dD'}{dR_D} = U_D\, e^{-R_D/R_D^{*}}$$

At $R_D = 0$ a repeated token is worth exactly $U_D \cdot 1$ - the same as a
fresh token. That is the formula's most important consequence and it is exact,
not approximate: *the second epoch's tokens are worth what fresh tokens are
worth.* The decay is what makes later epochs cheaper in value.

Now generate the three thresholds. Compare $D'$ against $E \cdot U_D$, which is
what the same token count would be worth if every pass were fresh:

| Epochs $E$ | $R_D$ | $D' / U_D$ | Fresh equivalent $E$ | Efficiency |
|---|---|---|---|---|
| 4 | 3 | $1 + 15(1 - e^{-0.2}) = 3.72$ | 4 | **93%** |
| 8 | 7 | $1 + 15(1 - e^{-0.467}) = 6.59$ | 8 | 82% |
| 16 | 15 | $1 + 15(1 - e^{-1}) = 10.48$ | 16 | 65% |
| 40 | 39 | $1 + 15(1 - e^{-2.6}) = 14.89$ | 40 | 37% |
| $\infty$ | $\infty$ | $1 + 15 = 16$ | - | 0% |

Three facts, all now derived rather than asserted. Four epochs retains 93% of
the value of fresh tokens - "nearly free" is a measurement, and you can see how
nearly. Sixteen epochs is where you are paying about a third more compute than
the information warrants. And the asymptote is $D'/U_D = 1 + R_D^{*} = 16$:
**a fixed corpus is worth at most 16 times its unique token count, ever.** By 40
epochs you have collected 14.89 of that 16, which is 93% of everything the
corpus will ever give you. That is the precise content of "nothing past 40" - not
that the returns are small, but that the ceiling has been reached.

An analogous relation holds for parameters in the data-constrained regime, with
a much shorter decay constant $R_N^{*} \approx 5$: excess parameters past the
data-supported size stop helping several times faster than excess epochs do.
Practical consequence for this project: when the corpus binds, spend the surplus
compute on epochs before spending it on width.

## The laptop and the node

**Why the laptop's ratio is worse than its FLOPs suggest.** The $1/320$ figure is
sustained training throughput, and small models on any accelerator are often
memory-bound rather than compute-bound. The roofline argument makes this precise.

A weight matrix $W \in \mathbb{R}^{d \times d}$ applied to a batch of $B$ tokens
performs $2Bd^2$ FLOPs while reading $d^2$ parameters. In BF16 that is $2d^2$
bytes, so the arithmetic intensity is

$$I = \frac{2Bd^2}{2d^2} = B \quad \text{FLOPs per byte}$$

The intensity of a weight-stationary matmul is the batch size, full stop. Machine
balance - peak FLOP/s divided by peak memory bandwidth - is roughly
$990 \times 10^{12} / 3.35 \times 10^{12} \approx 295$ FLOPs per byte for an
H100. So you need on the order of 300 tokens per weight read to be compute-bound
at all. Below that the GPU is idle waiting on HBM and MFU collapses regardless of
how good the kernels are.

Two consequences you will hit directly. First, this is why gradient accumulation
exists: it raises effective batch without raising memory, moving you right along
the roofline. Second, it is why the 10-30M-parameter runs in this book should use
the largest batch that fits rather than a "reasonable" one - at these widths,
under-batching costs more than any hyperparameter mistake available to you.

**The overnight envelope as a level set.** The constraint

$$ND = \frac{\text{MFU} \times P_{\text{peak}} \times t_{\text{budget}}}{6}$$

is a hyperbola in the $(N, D)$ plane. Every configuration on it costs one night.
Moving along it trades model size against tokens at constant cost, and the
tokens-per-parameter ratio $D/N$ varies as $1/N^2$ along the curve, which is why
small changes in chosen $N$ swing the ratio hard: going from 30M to 50M
parameters on the same envelope drops $D/N$ from 53 to 19, from comfortably
overtrained to below Chinchilla-optimal.

## Evaluation without self-deception

**Why bits per byte is genuinely tokenizer-invariant, and the fine print.**

A model with a deterministic tokenizer induces a code for text. For an eval
document $x$ with token sequence $(t_1, \dots, t_m)$ under this tokenizer, the
model's total surprise is

$$S(x) = -\sum_{i=1}^{m} \ln P_\theta(t_i \mid t_{<i}) \ \text{nats}$$

By the source coding theorem this is exactly the code length, in nats, of an
arithmetic coder driven by this model. Convert to bits and divide by the
document's length in bytes:

$$\text{bpb}(x) = \frac{S(x)}{\ln 2 \cdot |x|_{\text{bytes}}}$$

The denominator is a property of the text alone. The numerator is a real code
length for that exact byte string, achievable by a program you could write. So
bpb is a comparison of two compressors on one file, and comparing compressors on
one file is well-posed no matter how differently they work internally. That is
the whole argument, and it is why bpb crosses tokenizer boundaries when
perplexity cannot.

The fine print, stated because someone will raise it: $S(x)$ is an upper bound
on $-\ln P_\theta(x)$ rather than equal to it, because the model places some
mass on non-canonical token sequences that decode to the same string, and this
marginalization is not performed. The bound is tight in practice and identically
signed for every model, so orderings are safe. Do not, however, report bpb as
"the entropy of the text under the model" - it is a code length under one
tokenization.

**The variance algebra, done properly.** Let $X_{i}$ be condition A's held-out
bpb on seed $i$ and $Y_{j}$ condition B's, each with variance $\sigma^2$ and
independent across seeds. For $n$ seeds per condition:

$$\text{Var}(\bar X - \bar Y) = \frac{\sigma^2}{n} + \frac{\sigma^2}{n} = \frac{2\sigma^2}{n},
\qquad \text{SE}_{\text{diff}} = \sigma\sqrt{\frac{2}{n}}$$

At $n = 2$ this is $\sigma$ exactly. Setting $\Delta / \text{SE}_{\text{diff}}
\ge z^*$ and solving:

$$n \ge \frac{2 z^{*2} \sigma^2}{\Delta^2}$$

which at $z^* = 2$ gives the canon formula $n \ge 8\sigma^2/\Delta^2$.

**One correction canon does not make, and you should.** That formula gives the
$n$ at which the *expected* $z$ equals 2 - which means that if the true effect is
exactly $\Delta$, you clear the bar about half the time. Detecting it reliably
requires power, and for 80% power at a two-sided 5% level the multiplier is
$(z_{\alpha/2} + z_{\beta})^2 = (1.96 + 0.84)^2 = 7.84$ rather than
$z^{*2} = 4$:

$$n \ge \frac{2 (1.96 + 0.84)^2 \sigma^2}{\Delta^2} = \frac{15.7\,\sigma^2}{\Delta^2}$$

Roughly double canon's number. Run the canon formula to decide whether an
experiment is plausible and this one to decide how many seeds to actually
launch.

**And one more that v6 and v7 will enforce.** Testing $K$ sources is $K$
hypothesis tests. At $K = 20$ and an uncorrected 5% threshold, the probability
of at least one false positive under a global null is $1 - 0.95^{20} = 64\%$.
Bonferroni sets the per-test level to $\alpha/K$, which for $K = 20$ moves the
threshold from $z = 1.96$ to $z = 3.02$ and raises the required seed count by
$(3.02/1.96)^2 = 2.4\times$. Budget for it now: a 20-source sweep needs roughly
twice the canon seed count for power and twice again for multiplicity, so the
honest planning figure is about $4\times n_{\text{canon}}$.

## At the bench: the overnight run

**Why two seeds per condition is enough, and what makes it enough.** With $n=2$
the within-condition variance estimate is

$$s_i^2 = \frac{(x_{i1} - x_{i2})^2}{2}$$

which has exactly 1 degree of freedom. A $\chi^2_1$-based estimate of $\sigma$
has a relative standard deviation around 76% - it is nearly worthless on its
own, and this is the correct objection to two-seed protocols taken one condition
at a time.

What rescues it is pooling. Across $K$ conditions run at two seeds each, assuming
a common $\sigma$:

$$s^2_{\text{pooled}} = \frac{1}{K}\sum_{i=1}^{K} \frac{(x_{i1} - x_{i2})^2}{2}$$

with $K$ degrees of freedom. At $K = 20$ that is a solid estimate (relative
standard deviation about 16%), obtained for free from runs you were doing anyway.
This is the actual statistical justification for the two-models-per-condition
ablation methodology: the seeds are not there to make each condition precise,
they are there to build one good shared variance estimate that every comparison
then borrows. Report $s_{\text{pooled}}$ once, use it everywhere, and state $K$.

The common-$\sigma$ assumption is worth a sentence of scepticism. If one
condition is far from the others - a source whose removal changes the corpus
composition substantially - its variance may genuinely differ, and pooling
understates uncertainty for that condition. Diagnostic: plot $|x_{i1} - x_{i2}|$
against condition and look for outliers before pooling.

**Propagation into what comes next.** Every downstream quantity is a linear
combination of noisy $v(S)$ values. For v5's Shapley values, each is a weighted
average over coalitions, so if $M$ independent subset evaluations each carry
variance $\sigma^2$ and enter with weights $w_k$ summing appropriately, the
resulting variance is $\sigma^2 \sum_k w_k^2$. Averaging over many coalitions
therefore *reduces* variance relative to a single leave-one-out difference - a
genuine and underappreciated argument for the Shapley construction over plain
LOO, quite separate from the fairness axioms. v5 makes it explicit.

## What you can now do

The whole unit compresses to three formulas with stated domains:

$$C \approx 6ND \quad (\text{valid when } n \ll 12d \text{, or paired with an MFU fitted at the same } n)$$

$$\text{bpb} = \frac{\mathcal{L}}{\ln 2 \cdot \text{bytes/token}} \quad (\text{valid for any deterministic tokenizer; an upper bound on the true code length})$$

$$\text{SE}_{\text{diff}} = \sigma\sqrt{2/n} \quad (\text{valid for independent seeds with common } \sigma)$$

And they compose into one budget for an entire research program. For $R$ total
runs at $n$ seeds each:

$$\text{cost} = R \cdot n \cdot \frac{6ND}{\text{MFU} \times P_{\text{peak}}} \times \text{price per second}$$

Every factor enters linearly, so the elasticity of the program's cost with
respect to each is exactly 1, and the ones you control differ only in what they
cost you scientifically: $R$ and $n$ are the experiment's statistical power, $N$
and $D$ are the model's adequacy, MFU is free money.

**Where the math gets harder, so you can see the road.** Fitting a scaling law of
the form $\mathcal{L} = \mathcal{L}_\infty + AN^{-\alpha} + BD^{-\beta}$ is a
nonlinear regression, usually done by taking logs of the reducible terms and
fitting in log space; pre-calculus is sufficient provided you treat the optimizer
as a black box. Nothing in v4 or v5 needs more than what is here - v4 is
subtraction and linear regression, v5 is combinatorics and expectation over
permutations. The genuine wall is v8's influence functions, where second
derivatives arrive. That is the correct order: build the ground truth with
arithmetic first, then learn the calculus while it has something to check itself
against.
