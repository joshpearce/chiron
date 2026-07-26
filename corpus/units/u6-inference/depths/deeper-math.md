---
unit: u6
depth: deeper-math
---

## Where inference sits in the stack

The framing claim of this unit - that inference-time machinery cannot change
what the model knows - has a precise statement worth writing down, because it
tells you exactly which of the four techniques needs a proof and which does not.

Let $f_\theta$ be the network, a deterministic function from a token sequence
$t_{1:n}$ to a logit vector $z \in \mathbb{R}^V$. A *decoding procedure* is a map
$D$ from $z$ to a distribution over the vocabulary, and a token is drawn from
$D(z)$.

- **Sampling** (temperature, top-k, top-p) modifies $D$, never $f_\theta$. It
  cannot be otherwise: $D$ is applied to the return value of $f_\theta$.
- **KV caching** modifies neither. It changes the evaluation strategy of
  $f_\theta$ from full recomputation to memoization. The claim needing proof is
  that the memoized quantities are functions of already-fixed inputs, which
  follows from the causal mask (proved below).
- **Quantization** replaces $\theta$ with $\hat{\theta}$, so it genuinely
  changes $f$. This is the one technique with no exactness guarantee, and the
  entire section on it is about bounding $\|f_{\hat\theta} - f_\theta\|$.
- **Speculative decoding** changes the algorithm that draws from $D(z)$ while
  provably preserving $D(z)$ itself. This one has an exact proof.

So: three of four are exact by construction, one is approximate with a bound.
Keep that ledger in mind; it is the structure of the whole unit.

## Temperature, top-k, top-p on one set of logits

### Temperature is a tempered distribution, not a rescaling of probabilities

The identity that makes temperature intuitive:

$$p_i(T) = \frac{e^{z_i/T}}{\sum_j e^{z_j/T}} = \frac{\left(e^{z_i}\right)^{1/T}}{\sum_j \left(e^{z_j}\right)^{1/T}} = \frac{p_i(1)^{1/T}}{\sum_j p_j(1)^{1/T}}$$

Temperature raises every probability to the power $1/T$ and renormalizes. This
is the statistical-physics tempering operation, and it makes the direction of
the effect obvious: for $T > 1$, exponent $1/T < 1$, which pulls values toward 1
- so small probabilities rise proportionally more than large ones, and the
distribution flattens. For $T < 1$ the exponent exceeds 1 and the ordering is
preserved but the gaps amplify.

Two structural consequences follow immediately, and both are testable:

**Ordering is invariant.** $x \mapsto x^{1/T}$ is strictly increasing on
$(0, \infty)$ for any $T > 0$. So $p_i(T) > p_j(T) \iff p_i(1) > p_j(1)$ for
every $T$. Temperature cannot promote a token past another. Whatever the model
ranked third at $T=1$ is ranked third at $T = 0.1$ and at $T = 50$. This is the
sharpest form of the M6 refutation: temperature cannot make the model "prefer" a
different token, only visit lower-ranked ones more often.

**Support is invariant.** If $p_i(1) > 0$ then $p_i(T) > 0$ for all finite
$T > 0$. Temperature never zeroes a token and never resurrects one. (top-k and
top-p do exactly that, which is the real difference between the knobs: one
reweights, the others truncate.)

### Entropy is monotone in temperature, and the rate is a variance

Write $\beta = 1/T$ (inverse temperature) and let $Z(\beta) = \sum_j e^{\beta z_j}$
be the partition function. Then $p_i = e^{\beta z_i} / Z(\beta)$, and

$$\frac{\partial \ln Z}{\partial \beta} = \frac{\sum_j z_j e^{\beta z_j}}{Z} = \mathbb{E}_p[z], \qquad \frac{\partial^2 \ln Z}{\partial \beta^2} = \mathbb{E}_p[z^2] - \mathbb{E}_p[z]^2 = \mathrm{Var}_p(z)$$

The Shannon entropy $H = -\sum_i p_i \ln p_i$ satisfies
$H = \ln Z - \beta\, \mathbb{E}_p[z]$, and differentiating:

$$\frac{\partial H}{\partial \beta} = \mathbb{E}_p[z] - \mathbb{E}_p[z] - \beta \frac{\partial \mathbb{E}_p[z]}{\partial \beta} = -\beta\, \mathrm{Var}_p(z)$$

Since $\mathrm{Var}_p(z) \ge 0$ and $\beta > 0$, entropy strictly decreases in
$\beta$ and therefore strictly increases in $T$, with equality only when all
logits are equal.

Check it on the canon vector $z = [3, 1, 0, -1, -2]$. At $T = 1$,
$\mathrm{Var}_p(z) = 1.0093$, so $\partial H / \partial \beta = -1.0093$
(numerical differentiation agrees to four decimals). And the entropies:

| $T$ | $H$ (nats) | $e^H$ (effective choices) |
|---|---|---|
| 0.5 | 0.1103 | 1.12 |
| 1.0 | 0.6262 | 1.87 |
| 2.0 | 1.2441 | 3.47 |
| 4.0 | 1.5117 | 4.53 |
| $\to\infty$ | 1.6094 = $\ln 5$ | 5.00 |

The right column is the perplexity of the sampling distribution, the same
quantity u5 defined for the loss. It is the effective number of tokens the
sampler is choosing among, and it interpolates from 1 (greedy) to $V$ (uniform)
as $T$ goes from 0 to $\infty$. This gives you a principled way to think about
what a temperature setting means: $T$ is a dial on effective branching factor,
and the mapping from $T$ to branching factor depends on the logits, so the same
$T$ means different things at different positions.

### top-p, formally

Let $\pi$ be the permutation sorting probabilities descending. Nucleus sampling
with parameter $p$ keeps the set

$$S_p = \{\pi(1), \dots, \pi(m)\}, \qquad m = \min\left\{ m' : \sum_{i=1}^{m'} p_{\pi(i)} \ge p \right\}$$

and samples from $p_i / \sum_{j \in S_p} p_j$ restricted to $S_p$.

The truncation removes mass $1 - \sum_{S_p} p_j \le 1 - p$ and redistributes it
proportionally over the survivors, so each survivor's probability increases by a
factor of at most $1/p$. That bound is what makes top-p safe in a way top-k is
not: no surviving token can be inflated by more than $1/p$, whereas top-k with
small $k$ on a flat distribution can inflate a token arbitrarily.

The relationship to entropy is what makes it adaptive. $|S_p|$ is bounded below
by a function of the entropy: since the maximum probability satisfies
$p_{\pi(1)} \ge e^{-H}$ is false in general but $\sum_{i \le m} p_{\pi(i)} \ge p$
requires $m \ge p / p_{\pi(1)}$, and $p_{\pi(1)}$ falls as $H$ rises, the nucleus
grows with entropy. That is the property the canon section demonstrated
numerically: $|S_{0.9}| = 1, 2, 4$ at $T = 0.5, 1, 2$.

### Why the composition order matters

Temperature-then-truncate and truncate-then-temperature are genuinely different
operators. Let $\mathcal{T}_T$ be tempering and $\mathcal{N}_p$ nucleus
truncation. $\mathcal{T}_T$ preserves ordering, so it commutes with the *sorting*
step - but the nucleus *size* $m$ depends on the cumulative sums, which
tempering changes. Concretely, on the canon vector: $\mathcal{N}_{0.9}$ applied
to $p(1)$ keeps 2 tokens; applying $\mathcal{T}_2$ afterward tempers those 2 to
$[0.731, 0.269]$. Applying $\mathcal{T}_2$ first and then $\mathcal{N}_{0.9}$
keeps 4 tokens. Different support, so not merely different weights - one
composition can emit tokens the other assigns probability zero.

## T=0 is not truth mode

The claim that greedy decoding does not maximize sequence probability deserves
its formal statement, because it is a genuine result about the difference
between a greedy algorithm and an optimization problem.

The decoding objective, if you wanted one, is

$$t^* = \arg\max_{t_{1:m}} \prod_{i=1}^{m} p_\theta(t_i \mid t_{<i})$$

which is a search over $V^m$ sequences. Greedy decoding computes

$$t_i^{g} = \arg\max_{v} p_\theta(v \mid t^{g}_{<i})$$

one position at a time. This is the standard greedy heuristic on a search tree,
and it is optimal only when the problem has matroid structure or an exchange
property - which autoregressive decoding emphatically does not, since the
conditional distribution at step $i$ depends on all earlier choices.

The canon's counterexample in general form: greedy is suboptimal whenever there
exist $u, v$ with $p(u) > p(v)$ at step 1 but
$p(u) \cdot \max_w p(w|u) < p(v) \cdot \max_w p(w|v)$. With
$p = [0.6, 0.4]$ and continuations $\max_w p(w|u) = 0.5$,
$\max_w p(w|v) = 0.99$: greedy gets $0.30$, optimum is $0.396$. The gap is
unbounded in $m$: chaining $m$ such decisions gives greedy $0.6^m \cdot 0.5^m$
against $0.4^m \cdot 0.99^m$, a ratio that grows exponentially.

Beam search with width $b$ narrows this by keeping $b$ partial hypotheses, and
it is still not exact - it is a bounded-width best-first search on an
exponential tree. Note also the length bias: $\prod p_i$ decreases
monotonically in $m$, so unnormalized sequence-probability maximization prefers
short sequences, which is why beam search implementations divide by $m^\alpha$.
There is no principled value of $\alpha$; it is a hyperparameter, which should
tell you how well-posed "the most likely sequence" is as a goal.

## The KV cache is not memory

### Proving the cached quantities are immutable

The claim in canon - appending a token cannot change any earlier position's K or
V - follows from causal masking by induction on depth, and it is worth doing
carefully because the induction is where the argument actually lives.

Let $x^{(\ell)}_i \in \mathbb{R}^{d_{model}}$ be the residual stream at layer
$\ell$, position $i$. Claim: for a sequence $t_{1:n}$, $x^{(\ell)}_i$ depends
only on $t_{1:i}$, for every $\ell$ and $i$.

*Base case*, $\ell = 0$: $x^{(0)}_i = E[t_i] + \text{pos}(i)$, a function of
$t_i$ and $i$ alone.

*Inductive step*: assume $x^{(\ell)}_j$ depends only on $t_{1:j}$ for all $j$.
At layer $\ell+1$, position $i$ computes

$$a_i = \sum_{j \le i} \mathrm{softmax}_j\!\left(\frac{q_i \cdot k_j}{\sqrt{d_{head}}}\right) v_j$$

where $q_i = W_Q x^{(\ell)}_i$, $k_j = W_K x^{(\ell)}_j$,
$v_j = W_V x^{(\ell)}_j$, and the sum runs only to $i$ because the causal mask
sets scores for $j > i$ to $-\infty$, hence softmax weight 0. Every term
involves $x^{(\ell)}_j$ with $j \le i$, each of which depends only on $t_{1:j}
\subseteq t_{1:i}$ by hypothesis. The MLP and normalization are position-wise,
adding no cross-position dependence. So $x^{(\ell+1)}_i$ depends only on
$t_{1:i}$. $\square$

Therefore appending $t_{n+1}$ leaves $x^{(\ell)}_i$ unchanged for all $i \le n$,
hence leaves $k^{(\ell)}_i$ and $v^{(\ell)}_i$ unchanged, hence caching them is
exact. Note where the argument would break: remove the causal mask and the sum
runs over all $j$, position $i$'s stream depends on the whole sequence, and no
KV caching is possible. Bidirectional encoders (BERT and friends) genuinely
cannot cache this way. Causality is not just a training-objective choice; it is
what makes cheap autoregressive inference exist at all.

### The memory formula, and what is not in it

$$M(n) = 2 \cdot L \cdot H_{kv} \cdot d_{head} \cdot b \cdot n \quad \text{bytes}$$

The instructive part is the absence of $d_{model}$. The projection
$W_K \in \mathbb{R}^{d_{model} \times H_{kv} d_{head}}$ takes an input of width
$d_{model}$ and produces an output of width $H_{kv} d_{head}$; only the output is
stored. So a model can be arbitrarily wide in its residual stream and MLPs and
still have a small cache, which is precisely the design freedom that grouped-query
attention exploits.

The GQA arithmetic, stated as a ratio: replacing MHA ($H_{kv} = H_q$) with GQA
of group size $g = H_q / H_{kv}$ divides the cache by exactly $g$. It does not
change the FLOPs of attention at all - all $H_q$ query heads still run, they just
read from shared K/V. So GQA is a pure memory-and-bandwidth optimization with a
small quality cost, and its benefit scales with context length while its cost
does not. That asymmetry is why every long-context model uses it.

Multi-head latent attention (MLA, as in DeepSeek-V2/V3) pushes further: cache a
low-rank latent $c_i \in \mathbb{R}^{d_c}$ with $d_c \ll H_{kv} d_{head}$ and
reconstruct K and V by an up-projection absorbed into $W_Q$ and $W_O$. The cache
becomes $L \cdot d_c \cdot b \cdot n$ - a single factor, no $H_{kv}$, no factor of
2. It is the same trick as GQA (spend compute to shrink the cached
representation) taken to its logical end.

## Prefill and decode are different machines

### The roofline, written out

Both phases run the same weights. The difference is arithmetic intensity
$I = \text{FLOPs} / \text{bytes moved}$, and where each phase lands relative to
the machine's ridge point $I^* = \text{peak FLOPs/s} \,/\, \text{bandwidth}$.

For a weight matrix $W \in \mathbb{R}^{d_{in} \times d_{out}}$ multiplied by a
batch of $B$ activation rows:

$$\text{FLOPs} = 2 B\, d_{in} d_{out}, \qquad \text{bytes} = \underbrace{d_{in} d_{out} \cdot b_w}_{\text{weights}} + \underbrace{B(d_{in} + d_{out}) b_a}_{\text{activations}}$$

For $B$ small the weight term dominates and $I \approx 2B / b_w$: arithmetic
intensity is *linear in batch size* and inversely proportional to weight
precision.

On the M4 Max: 546 GB/s and roughly 34 TFLOP/s fp16 on the GPU, so
$I^* \approx 62$ FLOPs/byte. With $b_w = 0.54$ bytes (4.3125 bits), decode at
$B = 1$ gives $I \approx 2/0.54 \approx 3.7$ FLOPs/byte - a factor of 17 below
the ridge, deep in the bandwidth-bound region. Prefill with $B = 2000$ gives
$I \approx 7400$, far above the ridge, compute-bound.

Two predictions fall straight out of this, and both are things you can observe:

**Quantization speeds up decode roughly linearly and does not speed up prefill.**
Decode time $\propto b_w$; prefill time $\propto$ FLOPs, which quantization does
not reduce (4-bit weights are dequantized to fp16 for the matmul on most
kernels). This is why 4-bit gives you a ~3.7x token-rate improvement and a
negligible prompt-processing improvement.

**Batching is free until the ridge.** Serving $B$ users at once costs almost the
same wall-clock as serving one, up to $B \approx I^* b_w / 2 \approx 17$. This
is the entire economics of hosted inference, and it is why your local
single-user setup gets worse tokens-per-dollar than an API while getting better
tokens-per-second-per-request.

### The quadratic, with the prefill sum done properly

Attention FLOPs for decoding one token at context $n$:

$$C_{\text{dec}}(n) = \underbrace{2 n\, d_{head}}_{QK^\top} + \underbrace{2 n\, d_{head}}_{\text{weighted } V} = 4 n\, d_{head} \quad \text{per head per layer}$$

times $H_q L$. For prefill of an $n$-token prompt you pay this at every position,
and the causal mask means position $i$ attends over $i$ keys:

$$C_{\text{pre}}(n) = 4 d_{head} H_q L \sum_{i=1}^{n} i = 4 d_{head} H_q L \cdot \frac{n(n+1)}{2} \approx 2 d_{head} H_q L\, n^2$$

There is the $O(n^2)$, explicit. For the tray-table model at $n = 2000$:
$786{,}432 \times 2000 \times 2001 / 2 = 1.57$ TFLOPs of attention, against
$2 \times 3.33{\rm e}9 \times 2000 = 13.3$ TFLOPs for the weight matmuls -
attention is 10.6% of prefill. At $n = 32{,}768$ it is
$786{,}432 \times 32768 \times 32769/2 = 422$ TFLOPs against 218 TFLOPs of weight
matmuls, so 66% of prefill and now the majority. The crossover for prefill is at
$n = 4 \times (\text{active params}) / (4 d_{head} H_q L) = 2 n^*_{\text{decode}}
\approx 16{,}940$ tokens, exactly twice the decode crossover, because the
causal mask halves the prefill work.

FlashAttention does not change any of these FLOP counts. It changes the *bytes*:
naive attention materializes the $n \times n$ score matrix in HBM ($O(n^2)$
memory traffic), while FlashAttention tiles the computation and never writes it
out ($O(n^2/\sqrt{M})$ traffic for on-chip memory $M$). The asymptotic *compute*
is untouched. This is the cleanest available illustration of M10's error:
FlashAttention made attention dramatically faster in wall-clock without changing
its complexity class by one iota.

## Quantization: what four bits hold

### Error analysis of the uniform quantizer

For a scalar uniform quantizer with step $\Delta$ and input not clipped, the
error $e = \hat{w} - w$ satisfies $|e| \le \Delta/2$. Under the standard
high-resolution assumption (the input density is roughly flat across one step),
$e \sim \mathrm{Uniform}(-\Delta/2, \Delta/2)$, giving

$$\mathbb{E}[e] = 0, \qquad \mathrm{Var}(e) = \frac{\Delta^2}{12}, \qquad \sigma_e = \frac{\Delta}{\sqrt{12}}$$

For a group of $G$ weights quantized to $2^\beta$ levels over the group's own
min-max range, $\Delta = R_G / (2^\beta - 1)$ where $R_G$ is the group range.
For $G$ i.i.d. $\mathcal{N}(0, \sigma^2)$ weights, $\mathbb{E}[R_G]$ grows like
$\sqrt{\ln G}$ - slowly, which is the useful fact, since it means group size is
a weak lever. (The textbook asymptotic $2\sigma\sqrt{2\ln G}$ converges
notoriously slowly and overshoots badly at practical $G$: it predicts
$5.77\sigma$ at $G=64$ where simulation gives $4.688\sigma$. Use simulated
values.) Measured: $4.14\sigma$ at $G=32$, $4.69\sigma$ at $G=64$,
$5.19\sigma$ at $G=128$, $5.66\sigma$ at $G=256$ - quadrupling the group size
costs about 21% more quantization error. Taking $G = 64$:

$$\frac{\sigma_e}{\sigma} = \frac{4.688}{(2^\beta - 1)\sqrt{12}}$$

Evaluate: $\beta = 4$ gives $4.688/(15 \times 3.464) = 0.0902$, matching the
0.0896 measured by direct simulation. $\beta = 3$: $0.193$. $\beta = 2$: $0.451$.
Each bit removed roughly doubles $\sigma_e$, and the relationship
$\sigma_e \propto 2^{-\beta}$ is exact in the high-resolution limit. In signal
terms, SNR improves 6.02 dB per bit - the standard result.

### The honest version of "high dimensions save you"

There is a popular hand-wave that quantization error averages out in high
dimensions. It does not, and it is worth being precise because the correct
statement is still favorable.

Consider $y = w^\top x$ and $\hat{y} = (w + e)^\top x$ with $e$ zero-mean,
independent of $x$, variance $\sigma_e^2$ per component:

$$\hat y - y = e^\top x, \qquad \mathrm{Var}(\hat y - y) = \sigma_e^2 \|x\|^2$$

Meanwhile, for $w$ with i.i.d. entries of variance $\sigma^2$,
$\mathrm{Var}(y) = \sigma^2 \|x\|^2$. So

$$\frac{\text{std}(\hat y - y)}{\text{std}(y)} = \frac{\sigma_e}{\sigma}$$

**Independent of $d$.** Simulation confirms: at $d = 64, 1024, 4096$ the relative
dot-product error is $0.0901, 0.0895, 0.0900$. A 9% weight perturbation gives a
9% perturbation of every pre-activation, at any width. Dimension does not help.

So where does the robustness actually come from? Three places, none of which is
dimensional averaging:

1. **The perturbation is 9%, not 87.5%.** Bit-counting was simply the wrong
   accounting. This is most of the answer.
2. **Flatness of the loss basin.** SGD with minibatch noise converges to regions
   where $\nabla^2 \mathcal{L}$ has small eigenvalues in most directions. To
   second order, $\mathbb{E}[\mathcal{L}(\theta + e) - \mathcal{L}(\theta)]
   = \tfrac{1}{2}\sigma_e^2\, \mathrm{tr}(\nabla^2 \mathcal{L})$ - the damage is
   the trace of the Hessian times the noise variance, and flat basins have small
   trace. Sharpness-aware training and quantization-robustness are the same
   phenomenon, and this is why quantization-aware training works by explicitly
   flattening.
3. **Scale-invariance downstream.** RMSNorm sets
   $\hat x = x / \sqrt{\tfrac{1}{d}\sum x_i^2} \cdot \gamma$, which is invariant
   to any scalar rescaling of $x$. Perturbations decompose into a radial
   component (killed by the norm) and a tangential one (survives). Only the
   tangential part matters, which is strictly less than the full error.

Point 2 is the deepest one, and it is what makes M16 wrong at the level of the
right abstraction: quantization damage is governed by the curvature of the loss
around the trained weights, not by an information-theoretic bit budget. GPTQ
operationalizes exactly this - it quantizes column by column and updates the
remaining unquantized weights to compensate, using an approximation to the
layerwise Hessian $H = 2 X X^\top$ where $X$ is a batch of calibration
activations. It is second-order optimization in service of quantization, and it
is why GPTQ beats round-to-nearest despite storing the same number of bits.

## Where quantization falls off the cliff

### Outliers, quantitatively

The group range $R_G$ is set by the extremes. If one weight in a group of 64 is
$10\sigma$ while the rest are within $\pm 3\sigma$, then $R_G \approx 12.3\sigma$
instead of $4.7\sigma$, so $\Delta$ grows by 2.6x and $\sigma_e/\sigma$ goes
from 9.0% to 23.7% for the 63 normal weights. One outlier degrades its entire
group to roughly 3-bit quality (19.2%) or slightly worse.

This is why the fix is structural rather than numeric. AWQ observes that the
sensitivity is not symmetric between weights and activations: for
$y = \sum_i w_i x_i$, scaling $w_i \to w_i / s_i$ and $x_i \to s_i x_i$ leaves
$y$ unchanged but changes which quantities need range. Choosing $s_i$ from
activation magnitudes moves the dynamic range out of the weights (where you have
4 bits) into the activations (where you have 16). The number of stored bits is
identical; the allocation of range is not.

SmoothQuant does the same migration with a tunable exponent
$s_i = \max|x_i|^\alpha / \max|w_i|^{1-\alpha}$, and the existence of a tunable
$\alpha$ is the tell that this is a range-allocation problem, not an information
problem.

### Why compounding hits reasoning first

The canon claim - quantization damage appears in long derivations before it
appears in factual recall - has a mechanism. Model a chain of $m$ dependent
generation steps where each step's correctness requires the perturbed logits to
preserve the argmax. If a single step preserves it with probability $1 - \epsilon$
and steps are roughly independent, the chain survives with probability
$(1-\epsilon)^m \approx e^{-\epsilon m}$. Factual recall is $m \approx 1$-$5$
tokens with a large logit margin (high-frequency training targets have big gaps,
so $\epsilon$ is tiny). A 40-step derivation has many low-margin steps, and
$\epsilon m$ is no longer small.

The prediction this makes, which is the falsifiable version: quantization damage
should scale with *output length* and with *logit margin*, not with the rarity of
the fact. Measured degradation curves match this - GSM8K and long-form code drop
before TriviaQA does. The "compressed database" model predicts the reverse
(rare facts, being sparsely represented, should vanish first), and that is not
what happens.

## Speculative decoding: exact, not approximate

### The exactness proof

Let $p$ be the target distribution and $q$ the draft distribution at a given
position. The procedure: draw $x \sim q$, accept with probability
$\min(1, p(x)/q(x))$; on rejection draw from
$p'(\cdot) = \max(0, p - q) / \beta$ where
$\beta = \sum_y \max(0, p(y) - q(y))$.

**Claim:** the emitted token is distributed exactly as $p$.

*Step 1.* The probability of drafting $x$ and accepting it is

$$q(x) \min\left(1, \frac{p(x)}{q(x)}\right) = \min\big(q(x),\, p(x)\big)$$

*Step 2.* The total acceptance probability is $\sum_y \min(p(y), q(y))$. Use the
identity $\min(a,b) = a - \max(0, a - b)$ with $a = p(y)$, $b = q(y)$:

$$\sum_y \min(p(y), q(y)) = \sum_y p(y) - \sum_y \max(0, p(y) - q(y)) = 1 - \beta$$

so the rejection probability is exactly $\beta$. (Note this also shows
$\beta = \tfrac{1}{2}\|p - q\|_1$, the total variation distance - the acceptance
rate $\alpha = 1 - \beta$ is one minus the TV distance between draft and target.
A draft model's usefulness is measured in TV distance, which is a clean thing to
know.)

*Step 3.* Total probability of emitting $x$:

$$P(x) = \underbrace{\min(p(x), q(x))}_{\text{accepted}} + \underbrace{\beta \cdot \frac{\max(0, p(x)-q(x))}{\beta}}_{\text{rejected, then resampled}} = \min(p(x), q(x)) + \max(0, p(x) - q(x))$$

Case $p(x) \le q(x)$: this is $p(x) + 0 = p(x)$.
Case $p(x) > q(x)$: this is $q(x) + p(x) - q(x) = p(x)$.

Either way $P(x) = p(x)$. $\square$

The $\beta$ in the numerator and denominator canceling is the whole trick: the
residual distribution is normalized by precisely the mass that rejection
produces, so the correction is exactly the right size, for any $q$ whatsoever.
Nothing in the proof assumed $q$ was close to $p$, or trained on similar data,
or even sensible. Exactness is unconditional; only the *rate* $\alpha = 1 - \beta$
depends on the draft's quality.

For multiple drafted tokens, the argument applies position by position. Position
$j$ is only reached if positions $1..j-1$ were all accepted, in which case the
prefix is a valid sample from the target, so the target's conditional at
position $j$ is the correct $p$ to verify against - which is exactly what the
single parallel target forward pass computed. Induction on $j$ completes it.

### Expected throughput

With per-position acceptance probability $\alpha$, treated as independent across
positions, let $A$ be the number of accepted drafts before the first rejection,
capped at $k$. Then $P(A \ge j) = \alpha^j$ for $j \le k$, so

$$\mathbb{E}[A] = \sum_{j=1}^{k} \alpha^j = \frac{\alpha(1 - \alpha^k)}{1 - \alpha}$$

Every step emits one additional token beyond the accepted drafts - either the
resampled replacement at the rejection point, or the bonus token from the
target's own distribution at position $k+1$ if all $k$ were accepted. So

$$\mathbb{E}[\text{tokens per step}] = 1 + \frac{\alpha(1-\alpha^k)}{1-\alpha} = \frac{1 - \alpha + \alpha - \alpha^{k+1}}{1-\alpha} = \frac{1 - \alpha^{k+1}}{1 - \alpha}$$

which is the canon formula. Sanity checks: $\alpha \to 0$ gives 1 (no
speculation), $k \to \infty$ gives $1/(1-\alpha)$ (the cap stops binding),
$\alpha \to 1$ gives $k+1$ (everything accepted plus the bonus).

Optimal $k$. Let $c$ be the draft's per-token cost as a fraction of a target
forward pass. Speedup is

$$S(k) = \frac{1 - \alpha^{k+1}}{(1-\alpha)(1 + ck)}$$

Differentiating and setting to zero has no closed form, but the structure is
clear: the numerator saturates at $1/(1-\alpha)$ while the denominator grows
linearly, so $S$ has an interior maximum. At $\alpha = 0.8$, $c = 0.05$:
$S(2) = 2.22$, $S(4) = 2.80$, $S(6) = 3.04$, $S(8) = 3.09$, $S(10) = 3.05$,
$S(12) = 2.95$. The optimum sits at $k = 8$ and the curve is flat within 2%
across $k = 6$ to $10$, which is why implementations pick $k$ in the 4-8 range
and do not tune hard.

The independence assumption in $\alpha^j$ is the weak point of the analysis.
Acceptances are positively correlated in practice - a draft that nails token 1
is on an easy stretch and likely nails token 2 - so measured
$\mathbb{E}[\text{tokens}]$ exceeds the geometric prediction. The formula is a
lower bound in practice, which is the pleasant direction to be wrong in.

## What this unit did and did not change

Collect the four guarantees in one place, since they are of genuinely different
strengths and conflating them is a real error:

| technique | changes $f_\theta$? | guarantee | strength |
|---|---|---|---|
| temperature / top-k / top-p | no | logits bit-identical | definitional |
| KV cache | no | output bit-identical | proved by induction on the causal mask |
| quantization | **yes** | $\sigma_e/\sigma \approx 9\%$ at 4 bits; damage $\approx \tfrac12 \sigma_e^2 \mathrm{tr}(\nabla^2\mathcal{L})$ | a bound, not an identity |
| speculative decoding | no | emitted distribution exactly $p$ | proved, unconditional in $q$ |

Only one row is approximate, and its approximation is quantified rather than
vibes-based. That is the correct shape of the answer to "does this change what
the model knows": three times no by construction, once "yes, by 9% in weight
space, which the loss landscape's flatness converts into a couple of percent in
behavior."
