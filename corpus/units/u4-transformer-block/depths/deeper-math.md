---
unit: u4
depth: deeper-math
---

## The block in one equation

The parameter inventory of a block, biases and norm gains included. Let $d$ be
the model width, $d_{ff}$ the MLP hidden width, $h$ heads, $g$ KV heads.

| tensor | shape | count |
|---|---|---|
| $W^Q$ | $d \times d$ | $d^2$ |
| $W^K, W^V$ | $d \times g d_k$ each | $2 d^2 g/h$ |
| $W^O$ | $d \times d$ | $d^2$ |
| $W_1$ | $d \times d_{ff}$ | $d\,d_{ff}$ |
| $W_2$ | $d_{ff} \times d$ | $d\,d_{ff}$ |
| $\gamma_1, \beta_1, \gamma_2, \beta_2$ | $d$ each | $4d$ |
| biases (GPT-style) | $3d + d + d_{ff} + d$ | $5d + d_{ff}$ |

Sequence length $n$ never appears. Parameters are $O(d^2)$; activations are
$O(nd)$; attention's score matrix is $O(n^2 h)$ per layer.

**Forward FLOPs per block.** A matmul with $p$ parameters applied to $n$ tokens
costs $2np$ FLOPs (one multiply, one add per parameter per token). So the
weight-matmul part of a block is

$$ F_{\text{lin}} = 2n(12d^2) = 24 n d^2 $$

using $d_{ff} = 4d$ and $g = h$. The attention score/aggregate part touches no
parameters: $QK^{\top}$ is $n^2 d$ MACs and $AV$ is another $n^2 d$, so

$$ F_{\text{quad}} = 4 n^2 d $$

Set them equal: $4n^2 d = 24 n d^2 \Rightarrow n = 6d$. For $d = 4096$ the
quadratic term overtakes the linear term at **$n = 24{,}576$ tokens**. Below
that, a transformer forward pass is dominated by ordinary dense matmuls; the
$O(n^2)$ term everyone quotes is a minority of the FLOPs at typical context
lengths (at $n = 2048$, $d = 4096$: $8.25 \times 10^{11}$ linear against
$6.87 \times 10^{10}$ quadratic, a 12:1 split). This is the precise version of
M10: attention is genuinely $O(n^2)$, and that term is genuinely not dominant
until long context.

## Multi-head attention: many subspaces, one parameter budget

**The QK and OV circuits.** Drop the softmax and mask for a moment and look at
what the parameters compose into. Head $i$'s pre-softmax scores are

$$ S_i = (xW^Q_i)(xW^K_i)^{\top} = x \left(W^Q_i W^{K\top}_i\right) x^{\top} $$

The parameters only ever appear as the product
$W^{QK}_i := W^Q_i W^{K\top}_i \in \mathbb{R}^{d \times d}$. Since
$W^Q_i, W^K_i \in \mathbb{R}^{d \times d_k}$, we have
$\mathrm{rank}(W^{QK}_i) \le d_k$. The individual $Q$ and $K$ matrices are not
identifiable - $W^Q_i R$ and $W^K_i R^{-\top}$ give the same $W^{QK}_i$ for any
invertible $R \in \mathbb{R}^{d_k \times d_k}$. Only the product is a real
object. Call it the **QK circuit**: it is the bilinear form that decides
*where* each position reads from.

Similarly, from the identity
$\big[\mathrm{head}_1 | \cdots | \mathrm{head}_h\big]W^O = \sum_i \mathrm{head}_i W^O_i$
(row-blocking $W^O$ into $W^O_i \in \mathbb{R}^{d_k \times d}$), head $i$'s
contribution is $A_i \, x \, W^V_i W^O_i$ where $A_i$ is the attention matrix.
Define the **OV circuit** $W^{OV}_i := W^V_i W^O_i \in \mathbb{R}^{d \times d}$,
also rank $\le d_k$: it decides *what* gets written given what was read. So

$$ \mathrm{MHA}(x) = \sum_{i=1}^{h} A_i \big(x\, W^{OV}_i\big), \qquad A_i = \mathrm{softmax}\!\left(\tfrac{x W^{QK}_i x^{\top}}{\sqrt{d_k}} + M\right) $$

Every head is exactly two rank-$d_k$ maps on the $d$-wide stream: one that
picks positions, one that transports content. This is the factorization all of
mechanistic interpretability is written in.

**Proof of the sum identity.** Write $C = [\,H_1 | \cdots | H_h\,] \in
\mathbb{R}^{n \times d}$ with $H_i \in \mathbb{R}^{n \times d_k}$, and block
$W^O$ by rows. Then $(CW^O)_{aj} = \sum_{b=1}^{d} C_{ab} W^O_{bj}$. Splitting
the $b$ sum into $h$ contiguous ranges of length $d_k$, range $i$ contributes
$\sum_{c=1}^{d_k} (H_i)_{ac} (W^O_i)_{cj} = (H_i W^O_i)_{aj}$. Sum over $i$.
Block matrix multiplication, nothing more - but it is what licenses "each head
writes an additive rank-$d_k$ update to the stream."

**Why $\sqrt{d_k}$ and not $\sqrt{d}$.** The scaling exists to keep the
pre-softmax score variance $O(1)$. If $q, k \in \mathbb{R}^{d_k}$ have iid
zero-mean unit-variance entries, $\mathrm{Var}(q \cdot k) = d_k$, so dividing
by $\sqrt{d_k}$ restores unit variance. The dot product is taken in the head's
subspace, so the relevant dimension is $d_k$, not $d$. Splitting into more
heads therefore *lowers* the divisor, which is consistent: shorter vectors,
smaller raw dot products.

**The rank/count trade, quantified.** A single head with $d_k = d$ gives one
attention pattern from a full-rank $W^{QK}$. Splitting into $h$ heads gives $h$
patterns, each from a rank-$d/h$ form. Total "score-forming capacity", counting
free parameters in the $W^{QK}_i$, is $h \cdot (2 d \cdot d/h) = 2d^2$ either
way. The split does not create or destroy capacity; it changes whether the
capacity buys one expressive read or many restricted ones. The empirical
answer - many, until $d_k$ drops below roughly 64 - says that in language, the
useful read patterns are individually low-rank.

## MQA and GQA: the same equation with fewer key-value heads

**GQA as a linear operator.** Let $K_{\text{small}} \in \mathbb{R}^{n \times g
d_k}$ be the compact key tensor. GQA computes the full $n \times h d_k$ key
tensor as $K = K_{\text{small}} R$ where $R \in \mathbb{R}^{g d_k \times h
d_k}$ is a fixed 0/1 repeat matrix (`repeat_interleave` with factor $h/g$).
$R$ has no parameters and $\mathrm{rank}(R) = g d_k$. So GQA is exactly MHA
whose key/value tensors are constrained to a rank-$g d_k$ subspace of the
rank-$h d_k$ space MHA can use. That is the precise sense in which it is a
capacity restriction, and it explains the observed quality ordering
MHA $\ge$ GQA $\gg$ MQA: at $g = 1$ the constraint is rank $d_k$ out of
$h d_k$.

**Roofline for decode, which is the whole argument.** Per new token, per layer,
attention does $2 h n d_k$ MACs = $4 h n d_k$ FLOPs (scores plus aggregate,
both over $h$ query heads) and reads $2 g n d_k b$ bytes of cache. Arithmetic
intensity:

$$ I = \frac{4 h n d_k}{2 g n d_k b} = \frac{2h}{g b} $$

Note that $n$ cancels: the intensity is a constant set by the head ratio and
the dtype. At $h = 32$, fp16 ($b = 2$):

| variant | $g$ | FLOPs/token/layer | bytes read | $I$ (FLOP/byte) |
|---|---|---|---|---|
| MHA | 32 | 134.2 M | 128 MiB | 1.0 |
| GQA | 8 | 134.2 M | 32 MiB | 4.0 |
| MQA | 1 | 134.2 M | 4 MiB | 32.0 |

(at $n = 8192$, $d_k = 128$). An A100's ridge point is
$312\,\mathrm{TFLOP/s} \div 2039\,\mathrm{GB/s} \approx 153$ FLOP/byte; an
H100's is around 295. Every variant is far to the left of the ridge, so decode
attention runs at bandwidth speed and its latency is proportional to bytes
read - which is $\propto g$. Cutting $g$ by 4 cuts attention decode latency by
close to 4, with the FLOP column unchanged. This is the formal statement of
"GQA is a memory optimisation, not a compute optimisation" (U4M3).

The same table read the other way explains why nobody bothers with GQA for
*prefill*: prefill processes $n$ tokens at once, reuses each loaded key across
$n$ queries, and sits near or past the ridge point. There the FLOP column is
what matters, and it is identical.

## The MLP block: where the parameters actually live

**GELU exactly.** $\mathrm{GELU}(z) = z\,\Phi(z) = \tfrac{z}{2}\big(1 +
\mathrm{erf}(z/\sqrt 2)\big)$, where $\Phi$ is the standard normal CDF. The
tanh approximation shipped in most code is
$\tfrac{z}{2}\big(1 + \tanh[\sqrt{2/\pi}(z + 0.044715 z^3)]\big)$, agreeing to
about $10^{-3}$. Its derivative is $\Phi(z) + z\phi(z)$, which is negative for
$z \in (-\infty, \approx -0.75)$ - unlike ReLU, GELU is non-monotone, giving a
small negative dip near $z \approx -0.75$ and a nonzero gradient everywhere.

**The $8/3$ factor for SwiGLU.** Two-matrix MLP: $2 d\,d_{ff}$ parameters.
Gated MLP: three matrices, $3 d\,d_{ff}'$. Match them:

$$ 3 d\, d_{ff}' = 2 d \, d_{ff} = 2d(4d) = 8d^2 \;\Longrightarrow\; d_{ff}' = \tfrac{8}{3} d $$

For $d = 4096$ that is $10{,}922.7$; Llama-2-7B rounds to a
hardware-friendly 11,008 ($= 2.6875 d$, a multiple of 256). Llama-3-8B does not
parameter-match, choosing $d_{ff} = 3.5d = 14{,}336$, so its MLP is 80.8% of the
block rather than 66.7%.

**Why $4d$ at all.** There is no derivation; $4d$ is the 2017 default that
survived ablation. The weak theory: the MLP needs $d_{ff} \gg d$ so that a
piecewise-linear-ish map on a $d$-dimensional input has enough distinct
"regions" to be expressive, and the number of activation patterns grows with
$d_{ff}$. The empirical finding is that the block's quality depends mostly on
total parameters $12d^2$-ish and only weakly on how they are split between
attention and MLP within a broad band, which is why the ratio has drifted from
$4d$ to $3.5d$-with-gating without much consequence.

## Key-value memory, and where that framing breaks

**Derivation of the sum form.** For $u \in \mathbb{R}^{1 \times d}$,
$(uW_1)_i = u \cdot (W_1)_{:,i} = u \cdot k_i$ where $k_i$ is column $i$ of
$W_1$. Let $a_i = \sigma(u \cdot k_i)$, so $a \in \mathbb{R}^{1 \times d_{ff}}$.
Then $(aW_2)_j = \sum_i a_i (W_2)_{ij}$, i.e.
$aW_2 = \sum_i a_i \, (W_2)_{i,:} = \sum_i a_i v_i$. Hence
$\mathrm{MLP}(u) = \sum_{i=1}^{d_{ff}} \sigma(u \cdot k_i)\, v_i$ exactly. The
"keys" are columns of the up-projection, the "values" are rows of the
down-projection, and there is no approximation anywhere.

**Why there are more features than neurons.** The capacity argument for
superposition is Johnson-Lindenstrauss-flavoured. In $\mathbb{R}^m$ one can
place $N$ unit vectors with all pairwise $|\cos| \le \varepsilon$ for

$$ N \sim \exp\!\left(c\, \varepsilon^2 m\right) $$

so the number of *nearly*-orthogonal directions grows exponentially in the
dimension, while the number of exactly-orthogonal ones is only $m$. A layer
that is willing to tolerate small interference between features can therefore
represent vastly more than $m$ features in $m$ dimensions. The cost is
crosstalk, paid as a small dense error term; the nonlinearity $\sigma$ then
suppresses the below-threshold crosstalk. The consequence for the lookup-table
model is decisive: since features occupy non-orthogonal directions shared
across neurons, there is no coordinate whose deletion deletes a feature, and
the observed polysemanticity of single neurons is predicted rather than
anomalous (M4).

**Rank-one model editing, and what it shows.** Methods in the ROME/MEMIT family
change a specific factual association by adding a rank-one update
$\Delta = v_* k_*^{\top}$ to a single MLP down-projection at a mid-stack layer,
solving for $v_*$ so that the layer's output at key $k_*$ becomes the target.
Two readings are tempting and only one is right. It does *not* show a fact
lived in a row; the edit is a rank-one perturbation of a dense matrix, chosen
by least squares, and it degrades neighbouring associations measurably. It does
show that the map is locally linear enough around a key direction that a
low-rank nudge is a well-posed intervention - which is what "smooth learned map
with bumps" predicts and "hash table" does not.

## The residual stream is a workspace, not a shortcut

**Unrolling.** With pre-LN, $x_l = x_{l-1} + F_l(x_{l-1})$ where $F_l$ is a
sublayer composed with its norm. By induction,

$$ x_L = x_0 + \sum_{l=1}^{L} F_l(x_{l-1}), \qquad x_0 = e(t) + p(t) $$

**Jacobian.** $\dfrac{\partial x_L}{\partial x_0} = \prod_{l=L}^{1}\left(I +
J_l\right)$ where $J_l = \partial F_l / \partial x_{l-1}$. Expanding the product
gives $2^L$ terms, one per subset $S \subseteq \{1..L\}$:

$$ \frac{\partial x_L}{\partial x_0} = \sum_{S \subseteq \{1..L\}} \prod_{l \in S} J_l \quad (\text{ordered}) $$

The $S = \emptyset$ term is $I$: the unattenuated gradient path (the ResNet
story). The $|S| = 1$ terms are single-block direct effects. The $|S| = 2$
terms are two-block compositions, which is exactly the induction-head circuit
of the last section. Empirically the mass concentrates on small $|S|$: a deep
transformer behaves like an ensemble of many shallow paths, which is the formal
version of "removing one middle block removes a small fraction of the paths"
(M8). Contrast a plain stack without residuals, where the Jacobian is the
single product $\prod J_l$ and every path must traverse every layer - remove
one and every path is broken.

**The logit lens.** Because $x_L$ is a sum and the unembedding $W_U$ is linear,
$\mathrm{logits} = \mathrm{Norm}(x_L)W_U$ decomposes (up to the norm's
nonlinearity) into per-block logit contributions
$F_l(x_{l-1})W_U$. Applying $W_U$ to intermediate $x_l$ produces readable token
distributions that sharpen with depth. This works *only* because updates are
additive into a stream whose basis is shared with the unembedding, and it is
direct evidence for the read/write-channel view.

**Bandwidth accounting.** An 8B model has $d = 4096$ and $2L = 64$ writers.
If each sublayer needed a private orthogonal subspace of its rank-$d_k$ writes,
attention alone would demand $32 \times 32 \times 128 = 131{,}072$ dimensions
against 4096 available. The factor of 32 shortfall is why the stream is in
superposition too, not only the MLP hidden layer.

## Normalization conditions the optimization; it does not save your floats

**The LayerNorm Jacobian, derived.** Take $\gamma = 1, \beta = 0, \epsilon = 0$.
Let $P = I - \tfrac{1}{d}\mathbf{1}\mathbf{1}^{\top}$ (the centering
projector), $c = Pu$, $\sigma = \sqrt{c^{\top}c/d}$, $y = c/\sigma$.

$\dfrac{\partial c}{\partial u} = P$. From $\sigma^2 = c^{\top}c/d$:
$2\sigma\,\mathrm{d}\sigma = \tfrac{2}{d} c^{\top}\mathrm{d}c$, and since
$c^{\top}P = c^{\top}$ (c is already centered),
$\dfrac{\partial \sigma}{\partial u} = \dfrac{c^{\top}}{d\sigma}$. Then by the
quotient rule

$$ \frac{\partial y}{\partial u} = \frac{P}{\sigma} - \frac{c}{\sigma^2}\cdot\frac{c^{\top}}{d\sigma} = \frac{1}{\sigma}\left(P - \frac{y y^{\top}}{d}\right) = \frac{1}{\sigma}\left(I - \frac{\mathbf{1}\mathbf{1}^{\top}}{d} - \frac{y y^{\top}}{d}\right) $$

with $\|y\|^2 = d$. (Verified against finite differences to $10^{-10}$.)
RMSNorm, with $r = \sqrt{u^{\top}u/d}$ and $y = u/r$, drops the centering term:
$\partial y/\partial u = \tfrac{1}{r}(I - y y^{\top}/d)$.

Read the three factors:

- $\mathbf{1}\mathbf{1}^{\top}/d$ removes the mean direction: the gradient
  carries no signal about a uniform shift of all features. RMSNorm keeps that
  channel, which is one of the two differences between them.
- $y y^{\top}/d$ removes the component *along the current activation*: gradient
  cannot ask for "same direction, bigger". This is the differential form of
  scale invariance, $\mathrm{LN}(au) = \mathrm{LN}(u)$, and it is the
  conditioning claim - the optimizer is forced to spend its updates on
  direction, where the information is.
- $1/\sigma$ auto-scales: an upstream layer whose outputs blow up receives
  proportionally shrunken gradients.

**Implicit learning-rate decay.** Scale invariance has a training-dynamics
consequence. If $W$ feeds a normalization, then for any $a > 0$ the loss
satisfies $\mathcal{L}(aW) = \mathcal{L}(W)$, and differentiating gives
$\nabla \mathcal{L}(aW) = \tfrac{1}{a}\nabla\mathcal{L}(W)$: gradients shrink
as weight norm grows. Since weight norm grows monotonically under SGD for
scale-invariant parameters, the *effective* step size decays automatically over
training, with no schedule. This is a real, load-bearing dynamical effect of
normalization and has nothing whatever to do with float range (M7).

## Pre-LN and post-LN

**Where the identity path goes.** Post-LN:
$x_l = \mathrm{LN}(x_{l-1} + F(x_{l-1}))$, so

$$ \frac{\partial x_l}{\partial x_{l-1}} = J_{\mathrm{LN}}\left(I + J_F\right) $$

The $I$ is still inside, but every layer multiplies by $J_{\mathrm{LN}} =
\tfrac{1}{\sigma}(I - \mathbf{1}\mathbf{1}^{\top}/d - yy^{\top}/d)$, whose
spectral norm is $1/\sigma$ and which is rank-deficient by 2. Over $L$ layers
the backward pass is a product of $L$ such factors and the clean
$S = \emptyset$ path of the previous section no longer exists. Pre-LN keeps
$\partial x_l/\partial x_{l-1} = I + J_F J_{\mathrm{LN}}$: the identity term
survives untouched to the embedding.

Xiong et al. (2020) make this quantitative at initialization: the expected
gradient norm at the output layer is $O\!\left(d\sqrt{\ln d}\right)$ for
post-LN - independent of depth, and large - while for pre-LN it is
$O\!\left(d\sqrt{\ln d / L}\right)$, shrinking with depth. Hence post-LN's
large, badly scaled early updates, and hence warmup: ramping the learning rate
buys time for the layer-wise gradient scales to equilibrate before large steps
are taken. Pre-LN trains without warmup because its gradients are already
depth-normalized.

**Stream growth under pre-LN.** If block $l$'s output has norm $\approx s$ and
successive outputs are roughly orthogonal (true early in training, when they
are near-random directions in $\mathbb{R}^{4096}$), then
$\|x_L\| \approx s\sqrt{2L}$: summing $L$ orthogonal vectors of length $s$
gives length $s\sqrt{L}$, not $sL$. Simulation in $d = 4096$: 8 terms gives
2.79 against $\sqrt 8 = 2.83$; 64 terms gives 7.85 against $\sqrt{64} = 8$. The
practical consequence is that a late block's write, normalized by the stream's
$\sqrt{2L}$-scale norm, is a proportionally smaller rotation of the
representation than an early block's - measured "layer contribution" curves
decline with depth in pre-LN models. Post-LN renormalizes after every add, so
its per-layer contributions stay comparable; that is the mechanism behind
post-LN's slightly better converged loss when it converges.

**DeepNorm** splits the difference: post-LN with the residual branch scaled,
$x_l = \mathrm{LN}(\alpha x_{l-1} + F(x_{l-1}))$ with $\alpha =
(2L)^{1/4}$-ish and matching init downscaling on $F$, which bounds the
per-layer update magnitude and has been used to train 1000-layer post-LN
transformers. Sandwich norm (Gemma-2, Grok-1) instead normalizes both the
sublayer input and its output before the add, keeping pre-LN's identity path
while capping each write's size.

## Depth: composition through the stream

**Three kinds of composition.** Head $B$ in a later layer can depend on head
$A$'s output through any of its three read projections, and the three have
different meanings (Elhage et al., "A Mathematical Framework for Transformer
Circuits"):

- **Q-composition**: $A$'s write lands in the subspace $B$'s $W^Q$ reads, so
  $A$ changes *what $B$ is looking for*.
- **K-composition**: $A$'s write lands in $B$'s $W^K$ subspace, so $A$ changes
  *which positions are findable*. Induction heads are K-composition: the
  previous-token head makes each position advertise its predecessor.
- **V-composition**: $A$'s write lands in $B$'s $W^V$ subspace, so $A$ changes
  *what gets transported* when $B$ fires. Chains of V-composition are what
  build multi-step transported features.

Detecting composition is a low-rank test: compose the circuits and compare
Frobenius norms, e.g. $\|W^{QK\top}_B W^{OV}_A\|_F \big/ (\|W^{QK}_B\|_F
\|W^{OV}_A\|_F)$, which is near zero for unrelated heads and elevated for
composing pairs. Because the write is rank $\le d_k$ and the read is rank
$\le d_k$, the composition either lines up in a shared subspace or it does not,
and this is measurable directly from the weights, with no data.

**Why one attention layer provably cannot do induction.** In a one-layer
attention-only transformer, every head's keys are $e(t_j)W^K$ - a function of
the raw embedding at position $j$ and nothing else. The induction task requires
the head at the final `[A]` to score position $j$ highly *because $t_{j-1} =
A$*. But $j$'s key cannot depend on $t_{j-1}$: there is no earlier layer that
could have written that fact into $j$'s residual stream. Formally, a one-layer
attention-only model implements bigram statistics (from the direct
embedding-to-unembedding path) plus **skip-trigrams** of the form
$[A] \ldots [B] \to [C]$, where the source token $B$ and the destination token
$A$ jointly boost some output $C$. Induction is
$[A][B] \ldots [A] \to [B]$, which requires conditioning on the *pair*
$(t_{j-1}, t_j)$ at the source - a two-stage dependency, and outside the
one-layer function class at any width. Width adds more skip-trigrams; it never
adds a stage.

**Depth versus width, as a scaling question.** At fixed parameter budget
$N \approx 12 L d^2$, choosing $L$ and $d$ trades stages against per-stage
capacity. The empirically observed optimum sits at an aspect ratio
$d/L \approx 100$ over a wide range of scales (GPT-3: $12288/96 = 128$;
Llama-3-8B: $4096/32 = 128$), and loss is flat within roughly a factor of 2
either side and degrades outside it. Too deep-and-thin: each stage is too
narrow to hold enough features, and gradient signal per layer thins. Too
shallow-and-wide: no amount of width supplies a missing composition stage, as
the induction argument shows.

## What to carry forward

The three algebraic facts to keep from this depth pass: MHA is
$\sum_i A_i (x W^{OV}_i)$ with both circuits rank $\le d_k$; the residual
Jacobian expands into $2^L$ paths with the empty path being the identity; and
the normalization Jacobian is $\tfrac{1}{\sigma}(I - \mathbf{1}\mathbf{1}^{\top}/d
- yy^{\top}/d)$, whose second projector is scale invariance in differential
form. Unit 5's backprop derivations use the second; extension x2 is written
entirely in the first and third.
