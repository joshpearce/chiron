---
unit: u3
depth: deeper-math
---

## The problem: mix information across a sequence with no recurrence

The constraint list in canon has a precise form worth stating. Let $f$ be the
mixing operation, taking $X$ (shape $n \times d_{model}$) to an output of the
same shape. Two properties pin down almost everything.

**Permutation equivariance.** Let $P$ be any $n \times n$ permutation matrix.
Any operation built from per-position projections and pairwise scores
satisfies $f(PX) = Pf(X)$: permute the input rows and the output rows permute
identically. This is forced by the variable-$n$ constraint. A parameter tied
to "position 7" cannot exist in a model that must handle $n = 3$, so the
operation cannot distinguish positions by index.

This is why positional encodings exist. Attention is *inherently* a set
operation. Order is not something the mask restores - the mask only forbids
looking forward. Order enters by making the token representations themselves
carry position, so that $x_i$ for "cat at position 3" differs from $x_i$ for
"cat at position 9" before attention ever sees them.

**Pairwise decomposability.** With no parameter per pair, the influence of
position $j$ on position $i$ must be a function $g(x_i, x_j)$ with parameters
shared across all pairs. Among such functions, bilinear forms
$g(x_i, x_j) = x_i M x_j^T$ are the ones that are differentiable everywhere,
cheap (one matmul), and expressive enough to be interesting. Attention is
the bilinear choice, softmaxed.

## Queries, keys, and values

Substituting the projections into the score reveals what the two matrices
jointly parameterize:

$$q_i \cdot k_j = (x_i W^Q)(x_j W^K)^T = x_i \, W^Q (W^K)^T \, x_j^T
= x_i M x_j^T, \qquad M = W^Q (W^K)^T$$

$M$ has shape $d_{model} \times d_{model}$ and rank at most $d_k$, because it
is a product of a $d_{model} \times d_k$ and a $d_k \times d_{model}$ matrix.
For the canon example, with $d_{model} = 4$ and $d_k = 2$:

$$M = W^Q (W^K)^T = \begin{bmatrix}
0 & 1 & 1 & 0 \\ 1 & 0 & 0 & 1 \\ 0 & 1 & 1 & 0 \\ 1 & 0 & 0 & 1
\end{bmatrix}, \qquad \text{rank}(M) = 2 = d_k$$

and $XMX^T$ reproduces the score matrix $S$ exactly.

Two consequences that matter.

**$Q$ and $K$ are individually meaningless; only $M$ is determined.** For any
invertible $d_k \times d_k$ matrix $R$, the substitution
$W^Q \to W^Q R$, $W^K \to W^K R^{-T}$ leaves $M$ - and therefore every score,
weight, and output - exactly unchanged, while changing $Q$ and $K$ entirely.
In the canon example, $R = \begin{bmatrix} 2 & 1 \\ 1 & 3 \end{bmatrix}$ turns
$q_1 = [2,0]$ into $[4,2]$ and produces bit-identical attention. There is no
canonical query space. Any interpretation attached to a particular coordinate
of $Q$ or $K$ is an artifact of the training run, not a fact about the
computation. This is the sharpest available version of the anti-agentive
argument: the "queries" are not even uniquely defined.

**$d_k$ is a rank budget.** $M$ is constrained to rank $d_k$, so the head can
express a $d_k$-dimensional subspace of matching criteria and no more. Making
$d_k$ small is a deliberate bottleneck: it forces each head to specialize
rather than express arbitrary pairwise relations, and it is the reason many
narrow heads beat one wide head. That argument is the whole of the next unit.

The value path has no such coupling: $W^V$ appears exactly once, unmultiplied
by anything learned, so $V$'s coordinates are as identifiable as any layer's
activations. Key and value are different kinds of object, not just different
matrices.

## Scaled dot-product attention: the equation

**The score matrix is low-rank; the weight matrix is not.** $S = QK^T$ is a
product through a $d_k$-dimensional bottleneck, so $\text{rank}(S) \le d_k$.
Whenever $n > d_k$ - which is always, in practice - $S$ is rank-deficient. In
the canon example $S$ is $3 \times 3$ with $\text{rank}(S) = 2$ and
$\det(S) = 0$.

Applying softmax destroys that structure. $\exp$ is applied elementwise and is
not a linear map, so the resulting $A$ has $\text{rank}(A) = 3$ and
$\det(A) = -0.043$ in the same example. The softmax is not decoration on a
linear operation: it is the step that lets a rank-$d_k$ scoring bottleneck
produce a full-rank mixing matrix. This is also why "linear attention"
variants, which drop the softmax to factor the computation and escape $n^2$,
give up real expressiveness rather than merely re-associating a product.

**Gradients through the aggregation step.** With $O = AV$:

$$\frac{\partial \mathcal{L}}{\partial V} = A^T \frac{\partial \mathcal{L}}{\partial O},
\qquad
\frac{\partial \mathcal{L}}{\partial A} = \frac{\partial \mathcal{L}}{\partial O} V^T$$

The first says something concrete: the gradient reaching value vector $v_j$ is
the sum over all queries of $A_{ij}$ times the gradient arriving at output $i$.
A value vector nobody attends to receives no gradient. In the canon example,
$\partial O_{\cdot 1} / \partial V_{11} = [0.163579, 0.248255, 0.178370]$,
which is exactly column 1 of $A$.

## Why divide by the square root of $d_k$

**The general variance statement.** Canon assumes unit variance. In general,
if the components of $q$ are i.i.d. with mean 0 and variance $\sigma_q^2$, and
those of $k$ likewise with $\sigma_k^2$, and the two are independent, then for
each term $\text{Var}(q_m k_m) = E[q_m^2]E[k_m^2] - (E[q_m]E[k_m])^2
= \sigma_q^2 \sigma_k^2$, and summing $d_k$ independent terms gives

$$\text{Var}(q \cdot k) = d_k \sigma_q^2 \sigma_k^2$$

The architecture controls $d_k$ exactly and controls $\sigma_q, \sigma_k$ only
approximately, through initialization and normalization layers. Dividing by
$\sqrt{d_k}$ removes the part that is known in closed form and leaves the rest
to LayerNorm. That division of labor is why the constant is $\sqrt{d_k}$ and
not something fitted.

**Scaling is exactly a temperature.** Softmax with temperature $T$ is
$\text{softmax}(s/T)$, and $\text{softmax}(S/\sqrt{d_k})$ is precisely
$T = \sqrt{d_k}$. Attention runs its softmax at a temperature that grows with
head dimension. Unlike the sampling temperature of unit u6, this one is fixed
at design time and baked into training.

**The Jacobian, which is where the gradient claim comes from.** For one
softmax row $p = \text{softmax}(s)$, with $\delta_{ij}$ the Kronecker delta
(1 when $i = j$, else 0):

$$\frac{\partial p_i}{\partial s_j} = p_i(\delta_{ij} - p_j),
\qquad J = \text{diag}(p) - pp^T$$

Derivation: write $p_i = e^{s_i}/Z$ with $Z = \sum_m e^{s_m}$. For $i = j$,
the quotient rule gives
$\partial p_i/\partial s_i = (e^{s_i}Z - e^{s_i}e^{s_i})/Z^2 = p_i - p_i^2$.
For $i \neq j$, only $Z$ depends on $s_j$, so
$\partial p_i / \partial s_j = -e^{s_i}e^{s_j}/Z^2 = -p_ip_j$. Both cases are
$p_i(\delta_{ij} - p_j)$.

Three facts fall out. The diagonal is $p_i(1-p_i)$, which is the $p(1-p)$ used
in canon. Every row of $J$ sums to zero, since $\sum_i p_i = 1$ forces the
outputs to trade off rather than move together. And as $p \to$ one-hot, every
entry of $J$ goes to 0 - the saturated softmax is not merely insensitive in
one direction, its entire Jacobian vanishes.

Evaluated on row 2 of the canon example, where
$p = [0.248255, 0.248255, 0.503490]$:

$$J = \begin{bmatrix}
0.186624 & -0.061631 & -0.124994 \\
-0.061631 & 0.186624 & -0.124994 \\
-0.124994 & -0.124994 & 0.249988
\end{bmatrix}$$

Rows sum to zero, and the entries are $O(0.1)$ rather than $O(10^{-5})$. That
is what a healthy, unsaturated attention row looks like, and it is what the
$\sqrt{d_k}$ division buys.

The full gradient through the scores composes this with the aggregation
gradient. Writing $G = \partial\mathcal{L}/\partial A$, row $i$ of the score
gradient is $\frac{1}{\sqrt{d_k}}(J_i G_i^T)$, and the leading factor is the
one place the scaling constant reappears in the backward pass. The complete
derivation, including $\partial\mathcal{L}/\partial Q$ and
$\partial\mathcal{L}/\partial K$, is extension x1.

## The whole computation by hand

**Why the dot product rather than cosine similarity.** Cosine would divide by
$\|q\|\|k\|$, discarding magnitude. The canon numbers show what that would
cost. Against $q_1 = [2,0]$:

| key | dot | $\|k\|$ | cosine |
|---|---|---|---|
| $k_1 = [1,1]$ | $2$ | $1.414214$ | $0.707107$ |
| $k_2 = [2,0]$ | $4$ | $2$ | $1.000000$ |
| $k_3 = [1,2]$ | $2$ | $2.236068$ | $0.447214$ |

$k_1$ and $k_3$ are tied on the raw dot product at 2, but their cosines differ
by a factor of $1.58$. The dot product lets a key be "loud" - a large-norm key
wins attention from many queries regardless of direction, and a near-zero-norm
key is effectively invisible. That norm is a learned, useful degree of freedom:
it is how a position advertises how much it wants to be read at all, separately
from what it wants to be read for. Cosine similarity would throw it away, and
attention would lose its only volume control.

**The softmax row is a Gibbs distribution.** Row $i$ of $A$ is
$e^{-E_j/T}/Z$ with energy $E_j = -q_i \cdot k_j$ and temperature
$T = \sqrt{d_k}$. The output is then the expectation of the value vector under
that distribution, $O_i = E_{j \sim A_i}[v_j]$. Attention is a Boltzmann
average of payloads over an energy landscape defined by query-key alignment,
which is also why it is exactly a Nadaraya-Watson kernel regression estimator
with an exponential kernel: a classical nonparametric smoother whose
bandwidth and feature map happen to be learned.

## Causal masking

**The mask is additive, and that is not an implementation detail.** Write the
mask as a matrix $\text{Mask}$ with $\text{Mask}_{ij} = 0$ for $j \le i$ and
$-\infty$ for $j > i$, and the masked scores as $S' + \text{Mask}$. Additive
composition matters because it keeps masking inside the linear part of the
computation, so the same fused kernel handles masked and unmasked attention
with one extra add, and so multiple masks (causal, padding, document-boundary)
compose by summation.

**Masking before the softmax equals masking after, then renormalizing.** For
row $i$ with visible set $\mathcal{V}_i = \{j : j \le i\}$:

$$\frac{e^{s_j}}{\sum_{m \in \mathcal{V}_i} e^{s_m}}
= \frac{e^{s_j}/\sum_{m=1}^{n} e^{s_m}}{\sum_{m \in \mathcal{V}_i} e^{s_m}/\sum_{m=1}^{n} e^{s_m}}
= \frac{A_{ij}}{\sum_{m \in \mathcal{V}_i} A_{im}}$$

Dividing numerator and denominator by the full normalizer converts one into
the other. So the colleague's post-hoc implementation in question u3-q9 is
recoverable, but only if they also divide each row by its surviving mass -
which nobody remembers to do, and which costs an extra pass over the matrix
that the pre-softmax version gets for free.

**Gradients respect the mask automatically.** With $p_j = 0$ at a masked
position, every entry of the Jacobian $p_i(\delta_{ij} - p_j)$ touching that
position is 0. No gradient flows to a masked score, so the masked entries of
$W^Q$ and $W^K$'s contribution never learn from the future. Causality is
enforced in the backward pass by the same algebra that enforces it forward,
with no extra machinery.

**Why $-10^9$ and not $-\infty$.** A literal $-\infty$ produces
$\infty - \infty = \text{NaN}$ under the standard max-subtraction trick if an
entire row is masked, which happens with padding tokens. A large finite
constant underflows $\exp$ to exactly 0 and stays finite. The result is
bit-identical for any realistic score range, and the failure mode of choosing
the constant too small is an information leak, not a rounding error - see
question u3-q13.

## What attention costs

**Exact FLOP counts** for one head, sequence length $n$, model width
$d_{model}$, head dimension $d = d_k = d_v$, counting a multiply-accumulate as
2 FLOPs:

| Step | FLOPs | In $n$ |
|---|---|---|
| $Q, K, V$ projections | $6 n d_{model} d$ | linear |
| $S = QK^T$ | $2 n^2 d$ | quadratic |
| scale + softmax | $\approx 5 n^2$ | quadratic |
| $O = AV$ | $2 n^2 d$ | quadratic |

The crossover where the quadratic terms overtake the projections is at
$n \approx 1.5 \, d_{model}$, so for a model with $d_{model} = 4096$,
attention is projection-dominated below roughly 6k tokens and
score-dominated above it. Both regimes exist inside the context windows people
actually use, which is why blanket claims about attention's cost share are
usually wrong in one direction or the other.

**Memory is a separate axis, and it is where FlashAttention lives.** Naively
materializing $A$ costs $n^2$ floats per head per layer. At $n = 32768$ that
is over 4 GB in fp32 for a single head, which is why the naive kernel runs out
of memory long before it runs out of time. FlashAttention tiles the
computation so $A$ is never materialized, reducing memory from $O(n^2)$ to
$O(n)$ and cutting wall-clock substantially through better cache behavior. It
performs the same $\Theta(n^2 d)$ arithmetic. It is an exact algorithm with a
better memory schedule, not an approximation and not a complexity improvement -
a distinction that question u3-q11 tests directly.

**Decode is a different cost model.** Generating token $t$ with a KV cache
computes one query against $t$ keys: $\Theta(td)$ work for that step, and
$\Theta(n^2 d)$ summed over a full $n$-token generation - the same total. But
per step it moves the entire $t \times d$ cache from memory to do $\Theta(td)$
arithmetic, an arithmetic intensity of roughly 1 FLOP per byte. That is deep
in the memory-bound regime on any modern accelerator, which is why decoding is
bandwidth-limited while prefill is compute-limited, and why the two have
completely different optimization stories. Unit u6 develops this.

## What you now know

The results above that generalize furthest: scores are a rank-$d_k$ bilinear
form $x_i M x_j^T$ with $M = W^Q(W^K)^T$, identified only up to an invertible
reparameterization of the head; softmax converts that rank-deficient object
into a full-rank mixer; and the softmax Jacobian $\text{diag}(p) - pp^T$
controls both the saturation argument for $\sqrt{d_k}$ and the automatic
gradient masking of future positions.

Two extensions build directly on this section. Extension x1 completes the
backward pass through $Q$ and $K$, where the $1/\sqrt{d_k}$ factor reappears
and the rank structure of $M$ shapes what the gradients can express.
Extension x4 replaces the assumption that position enters additively through
$X$, deriving RoPE as a rotation applied to $q_i$ and $k_j$ that makes
$q_i \cdot k_j$ depend on $i - j$ - which turns the permutation-equivariance
argument of the first section into a design tool rather than a limitation.
