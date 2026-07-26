## Decode is bandwidth-bound, and that is the whole motivation

The canon treats batch size 1 as the whole story. It is not - the general
statement is the roofline model, and it tells you exactly where MoE stops
helping.

Let $B$ be the batch size (number of sequences whose next token is being
generated in this forward pass), $W \in \mathbb{R}^{m \times n}$ a weight
matrix, and $b$ the bytes per stored weight. The forward pass computes
$WX$ where $X \in \mathbb{R}^{n \times B}$ stacks the $B$ activation vectors.

$$\text{bytes read} = mnb + \underbrace{nB \cdot b_a + mB \cdot b_a}_{\text{activations, negligible for } B \ll m,n}$$
$$\text{FLOPs} = 2mnB$$

Dropping activation traffic, the arithmetic intensity is

$$I(B) = \frac{2mnB}{mnb} = \frac{2B}{b}$$

The weight read is paid once per *pass*, not once per sequence, so intensity
scales linearly with batch. This is the single most important fact about LLM
serving economics.

The machine's balance point $\beta = \text{peak FLOP/s} / \text{peak bytes/s}$
partitions the regimes:

$$\text{time per pass} = \max\!\left(\underbrace{\frac{mnb}{\text{bytes/s}}}_{\text{memory-bound}},\ \underbrace{\frac{2mnB}{\text{FLOP/s}}}_{\text{compute-bound}}\right)$$

The crossover is at $I(B) = \beta$:

$$B^{*} = \frac{\beta b}{2} = \frac{65.4 \times 0.544}{2} \approx 17.8$$

Below batch 18 on this machine, adding sequences to the batch is **free** in
wall-clock terms - you are reading the same weights either way. Above it, you
are paying FLOPs. Two corollaries the canon only gestures at:

1. **Prefill is compute-bound.** Processing a 2000-token prompt is
   $B_{\text{eff}} = 2000$ in the formula above (all positions in parallel, one
   weight read). $I = 2 \cdot 2000 / 0.544 = 7353 \gg 65$. So prefill sits far
   on the compute side of the roofline, which is why prompt processing is fast
   per token and generation is slow per token on the same hardware, with the
   same weights.
2. **MoE's benefit is regime-dependent.** In the memory-bound regime it cuts
   the numerator of the binding term by $10.6\times$. In the compute-bound
   regime it cuts FLOPs by the same factor but you were not memory-limited
   anyway, so the win shows up as throughput rather than latency, and it is
   partly eaten by the gather/scatter and all-to-all overheads that a dense
   model does not pay.

## The router: replacing one MLP block with N of them

The canon asserts the router is trained by ordinary backprop. Here is the
actual gradient, because it explains the exploration problem in section 5.

Let $T$ be the selected top-$k$ index set, $\ell \in \mathbb{R}^{N}$ the router
logits, $g = \text{softmax}(\ell_T)$ the gates over the selected set, and

$$y = \sum_{i \in T} g_i E_i(x), \qquad y, E_i(x) \in \mathbb{R}^{d}$$

Let $\delta = \partial L / \partial y \in \mathbb{R}^{d}$ be the gradient
arriving from downstream (loss $L$ is a scalar; $\delta$ has the same shape as
$y$). Define the scalar

$$a_i = \delta^{\top} E_i(x) = \sum_{c=1}^{d} \delta_c \, E_i(x)_c$$

$a_i$ measures how much expert $i$'s output points along the direction that
*increases* loss - so a **more negative** $a_i$ means expert $i$ helped more.

The loss depends on $g_i$ only through $y$, so $\partial L / \partial g_i = a_i$.
The softmax Jacobian (over the selected set) is

$$\frac{\partial g_i}{\partial \ell_j} = g_i(\delta_{ij} - g_j)$$

with $\delta_{ij}$ the Kronecker delta (1 if $i = j$, else 0). Chain them:

$$\frac{\partial L}{\partial \ell_j} = \sum_{i \in T} a_i \, g_i (\delta_{ij} - g_j)
= g_j a_j - g_j \sum_{i \in T} g_i a_i$$

$$\boxed{\ \frac{\partial L}{\partial \ell_j} = g_j\left(a_j - \bar{a}\right), \qquad \bar{a} = \sum_{i \in T} g_i a_i\ }$$

and finally $\partial L / \partial W_r = (\partial L / \partial \ell)\, x^{\top}$,
an outer product of shape $N \times d$.

Read what that says. A gradient step moves $\ell_j$ by $-\eta g_j (a_j - \bar a)$.
If expert $j$ helped more than the gate-weighted average of the selected
experts ($a_j < \bar a$), its logit goes **up** for inputs like $x$. If it
helped less, its logit goes down. The router runs a continuous relative
competition among the experts it already chose.

**Two structural consequences.**

- For $j \notin T$, every term above is zero. $\partial L / \partial \ell_j = 0$
  exactly. The router receives no gradient whatsoever about experts it did not
  select - it cannot learn "expert 5 would have been better here", only
  "expert 5 was worse than expert 2 was, back when I happened to pick both".
  Top-$k$ is an argmax, and argmax is piecewise constant: zero gradient almost
  everywhere, undefined on the ties. Exploration therefore depends entirely on
  (i) initialization noise, (ii) the aux loss dragging under-used logits up,
  and in some architectures (iii) explicit noise added to $\ell$ before the
  top-$k$ (noisy top-$k$ gating).
- The factor $g_j$ out front means a barely-selected expert ($g_j \approx 0$)
  produces a vanishing router gradient even if it was excellent. Selection
  probability gates its own learning signal - a second, subtler feedback loop
  on top of the collapse loop.

**Why the gate is multiplied in at all.** If the block computed the unweighted
sum $\sum_{i \in T} E_i(x)$, then $y$ would not depend on $\ell$, so
$\partial L / \partial W_r = 0$ and the router would never train. The gate
$g_i$ exists to make expert selection differentiable-ish - it is the only path
by which routing quality reaches the optimizer.

## Experts are not domain specialists

**Why $f_i \cdot P_i$ and not something simpler.**

The obvious balance penalty is $\sum_i P_i^2$ (minimized at uniform) or the
negative entropy $\sum_i P_i \log P_i$. Both are differentiable and both are
wrong for this job, because $P_i$ is the router's *average soft probability*
while the thing that actually costs you money is the *hard* top-$k$ assignment
$f_i$. A router can have near-uniform average probabilities while its argmax
still lands on the same three experts every time (imagine probabilities
$[0.26, 0.25, 0.25, 0.24]$ on every token: perfectly uniform $P$, perfectly
collapsed $f$). Penalizing $P$ alone does not see this.

But $f_i$ is a count, and counts have no gradient. The Switch Transformer
formulation

$$L_{\text{aux}} = \alpha N \sum_{i=1}^{N} f_i P_i$$

resolves this by treating $f_i$ as a **constant coefficient** (stop-gradient)
and letting the gradient flow only through $P_i$:

$$\frac{\partial L_{\text{aux}}}{\partial P_i} = \alpha N f_i$$

So the pressure on expert $i$'s average probability is directly proportional to
how overloaded it measurably was. The hard statistic supplies the *magnitude*,
the soft statistic supplies the *derivative*. That is the whole design.

**Where the floor of 1.0 comes from.** Both $f$ and $P$ lie on the simplex
($\sum_i f_i = \sum_i P_i = 1$, all entries $\geq 0$). In the regime that
matters, $f$ is a hardened version of $P$, so they are positively coupled;
take $f = P$ as the idealization. Then

$$N \sum_i P_i^2 = N \|P\|_2^2$$

By Cauchy-Schwarz, $1 = \left(\sum_i P_i\right)^2 \leq N \sum_i P_i^2$, with
equality exactly when all $P_i = 1/N$. So

$$N\sum_i P_i^2 \geq 1$$

with the minimum 1 attained only at uniform routing, for every $N$. The $N$
prefactor is what makes the floor scale-free, so $\alpha = 0.01$ means the same
thing at $N = 8$ and $N = 128$.

Be honest about a gap: the *unconstrained* minimum of $\sum_i f_i P_i$ over
independent $f$ and $P$ is 0 (put all of $f$ on one expert and all of $P$ on
another). The loss is only a sane balance objective because $f$ and $P$ are
two views of the same router output and cannot be adversarially decoupled.
This is why the term is a heuristic that works, not a principled divergence.

**Modern variants, briefly.** Loss-free balancing (used in recent DeepSeek and
Qwen MoE models) drops the aux loss entirely and instead maintains a per-expert
bias $c_i$ added to the logits *for selection only*:

$$\text{select top-}k \text{ of } (\ell + c), \qquad \text{gate on } \ell \text{ alone}$$

with $c_i$ nudged down when expert $i$ is overloaded and up when it is starved,
by a fixed step size, outside the gradient. This removes the interference
between the balance term and the language-modeling gradient - the aux loss
provably degrades LM quality because it pulls the router away from its
loss-minimizing choice - while keeping load flat.

## Total versus active: the tradeoff table

**Deriving the batch-coverage formula.** Assume load balancing has done its job,
so each token's top-$k$ set is a uniformly random $k$-subset of the $N$
experts, independent across tokens. For a fixed expert $i$ and one token,

$$\Pr[\text{token misses } i] = 1 - \frac{k}{N}$$

Across $B$ independent tokens, $\Pr[\text{all miss } i] = (1 - k/N)^{B}$. Let
$Z_i$ indicate that expert $i$ is touched by at least one token in the batch.
By linearity of expectation (which needs no independence between the $Z_i$,
which is fortunate because they are dependent):

$$\mathbb{E}[\#\text{distinct experts}] = \sum_{i=1}^{N} \Pr[Z_i = 1] = N\left(1 - \left(1 - \frac{k}{N}\right)^{B}\right)$$

For $N \gg k$, $(1 - k/N)^B \approx e^{-kB/N}$, giving the clean form

$$\mathbb{E}[\#\text{experts}] \approx N\left(1 - e^{-kB/N}\right)$$

The scale is set by $B \sim N/k$: coverage is essentially complete once the
batch exceeds a few multiples of $N/k = 16$. At $B = 64 = 4 \cdot (N/k)$,
$e^{-4} = 0.018$, so 98% coverage - matching the exact 125.9 of 128.

Define the **effective bandwidth multiplier** at batch $B$:

$$M(B) = \frac{\text{dense bytes/pass}}{\text{MoE bytes/pass}} = \frac{P_{\text{dense}}}{P_{\text{shared}} + P_{\text{exp}}\left(1 - e^{-kB/N}\right)}$$

where $P_{\text{shared}} = 1.2$B is always-resident-and-read and
$P_{\text{exp}} = 33.9$B is the expert pool. At $B = 1$ this is
$35.1 / (1.2 + 33.9 \cdot 0.0625) = 35.1/3.3 = 10.6$. At $B = 64$ it is
$35.1 / (1.2 + 33.9 \cdot 0.984) = 35.1/34.6 = 1.02$. The advantage decays with
the same exponential.

**The knowledge side has no comparable derivation.** There is no accepted
scaling law that predicts MoE quality from $(N, k, P_{\text{total}})$. The
empirical picture from the Switch Transformer and subsequent work is that at
fixed *training FLOPs*, sparse models reach a given loss several times faster
than dense ones, and that quality improves with total parameters at fixed
active parameters but with clearly diminishing returns in $N$. The
$\sqrt{P_{\text{total}} \cdot P_{\text{active}}}$ heuristic is a curve-fit to a
handful of model families, not a law - do not extrapolate it.

## What MoE costs you

**Capacity factor and token dropping.** In distributed training, each expert
gets a fixed-size buffer so the all-to-all has a static shape. The buffer size
is

$$\text{capacity} = \left\lceil C \cdot \frac{B_{\text{tokens}} \cdot k}{N} \right\rceil$$

where $B_{\text{tokens}}$ is tokens per batch and $C$ is the **capacity factor**,
typically 1.0 to 1.25. The bracketed term is the count each expert would get
under perfect balance; $C$ is the slack. Tokens routed to an expert that has
already filled its buffer are **dropped** - their MoE block output is zero, so
only the residual stream passes through. With $C = 1.25$ and good balancing,
drop rates run under 1%; with a collapsing router they can hit double digits,
and the LM loss degrades in a way that looks like a mysterious plateau rather
than an obvious error.

Setting $C$ is a direct memory-versus-quality knob: activation memory for the
MoE layer scales linearly in $C$.

**Why loss spikes are structural, not bad luck.** The block output as a
function of $\ell$ is piecewise smooth with discontinuities on the top-$k$
boundaries. Crossing a boundary swaps an entire 265M-parameter expert into the
computation. The magnitude of the jump is

$$\|\Delta y\| = g_{\text{boundary}} \cdot \|E_{\text{in}}(x) - E_{\text{out}}(x)\|$$

At the boundary the two experts have equal logits, so $g$ is equal for both -
but $g$ at the top-$k$ boundary is the *smallest* of the selected gates, which
bounds the jump. This is the good news: the discontinuity is real but its size
is gated by the smallest selected weight, which is why MoE training is
unstable rather than impossible. The bad news is that this bound is on the
block output; after 48 layers of composition the sensitivity compounds, and
practitioners find they need higher-precision (fp32) router logits, tighter
gradient clipping, and a smaller learning rate on $W_r$ specifically.

**FLOPs accounting for the aux loss.** Free. $f$ is computed from the routing
you already did; $P$ is a mean of softmaxes you already computed. The
load-balancing machinery costs $O(NB)$ per layer against $O(kBd^2)$ for the
experts - a ratio of $N / (k d^2) \approx 128 / (8 \cdot 4 \times 10^6)$, about
four parts per million.
