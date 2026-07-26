---
unit: u9
depth: deeper-math
---

## The trace, and how to read it

The whole unit is an exercise in a single technique: **roofline analysis**. For
any stage, compute two numbers and compare them.

**Arithmetic intensity** $I$ of a stage is the FLOPs it performs divided by the
bytes it must move from main memory:

$$I = \frac{\text{FLOPs}}{\text{bytes moved}} \quad [\text{FLOP} \cdot \text{byte}^{-1}]$$

**Machine balance** $I^*$ of the hardware is its peak arithmetic rate divided by
its memory bandwidth:

$$I^* = \frac{F}{M} = \frac{4.5 \times 10^{14}\ \text{FLOP/s}}{2 \times 10^{12}\ \text{byte/s}} = 225\ \text{FLOP}\cdot\text{byte}^{-1}$$

The achievable rate for a stage is $\min(F,\ I \cdot M)$. If $I > I^*$ the stage
is compute-bound and its time is $\text{FLOPs}/F$. If $I < I^*$ it is
memory-bound and its time is $\text{bytes}/M$ - the arithmetic units are idle
and adding more of them changes nothing.

Every performance claim in this unit is one application of that inequality. It
is worth internalizing because it is not LLM-specific; it is the same analysis
you would run on a sparse solver or a database scan.

## Stage 1: the harness builds one flat token stream

There is no interesting math in serialization, but there is a useful formalism
for the thing the harness is doing.

The model is a function $f_\theta: \mathcal{V}^n \rightarrow \mathbb{R}^{V}$
mapping a sequence of $n$ token IDs from vocabulary $\mathcal{V}$ (with
$|\mathcal{V}| = V$) to a logit vector. Its type signature contains no roles, no
tools, no turn structure. The harness is a separate function
$\sigma: \mathcal{S} \rightarrow \mathcal{V}^n$ mapping structured session state
$\mathcal{S}$ (message list, tool schemas, file contents) into a token sequence.

The composition $f_\theta \circ \sigma$ is what you experience as "the
assistant". Every claim of the form "the model knows it is in a chat" is a claim
that $f_\theta$ has access to $\mathcal{S}$. It does not; it only ever sees
$\sigma(\mathcal{S})$. Prompt injection is precisely the observation that
$\sigma$ is not injective in the way you would want: user-controlled content can
produce token subsequences indistinguishable from ones $\sigma$ would emit for
privileged structure.

## Stage 2: bytes to token IDs to vectors

The gather is a matrix product, and it is worth seeing why, because it explains
why gradients flow to embeddings at all.

Let $e_t \in \mathbb{R}^V$ be the one-hot indicator for token ID $t$: a vector
of 128,000 entries, all zero except a 1 at index $t$. Let
$W_E \in \mathbb{R}^{V \times d_{model}}$ be the embedding matrix. Then

$$x_t = e_t^\top W_E \in \mathbb{R}^{d_{model}}$$

which is exactly row $t$ of $W_E$. Implementations use a gather because
multiplying by a one-hot vector wastes $V \cdot d_{model}$ multiply-adds, but the
mathematical object is a linear map, so backprop is well defined:

$$\frac{\partial \mathcal{L}}{\partial W_E} = e_t \frac{\partial \mathcal{L}}{\partial x_t}^\top$$

which is a rank-1 outer product: an all-zero $V \times d_{model}$ matrix except
for row $t$, which receives $\partial \mathcal{L} / \partial x_t$. Tokens that
never appear in a batch receive exactly zero gradient. This is why rare tokens
have poorly-trained embeddings, and why the "glitch tokens" phenomenon exists -
token IDs that survived into the vocabulary but were filtered out of the
pretraining corpus keep near-random embeddings and produce bizarre behavior.

**Weight tying.** If the unembedding at Stage 5 is $W_E^\top$ (common), then a
single parameter block receives gradient from both ends of the network. The
logit for token $j$ is $z_j = \langle x_{final},\ W_E[j] \rangle$, a dot product
between the final residual vector and the *input* embedding of token $j$. This
is what makes the residual stream and the embedding space literally the same
space, which is the premise of most mechanistic interpretability work.

## Stage 3: prefill - 8,192 tokens, one forward pass

Derive the prefill FLOP count rather than quoting it.

**Parameter-driven work.** A matrix multiply $Y = XW$ with
$X \in \mathbb{R}^{n \times a}$ and $W \in \mathbb{R}^{a \times b}$ costs
$2 n a b$ FLOPs: each of the $n \times b$ output entries is a dot product of
length $a$, which is $a$ multiplies and $a - 1$ adds, counted as $2a$. Since
$ab$ is the parameter count of $W$, the cost is $2 n \times (\text{params})$.
Summing over every weight matrix in the network gives

$$\text{FLOPs}_{\text{params}} = 2 n P_a$$

with $P_a$ the *active* parameter count per token. For $n = 8192$,
$P_a = 22 \times 10^9$: $3.60 \times 10^{14}$.

**Attention work.** Per block, the score matrix computation
$S = QK^\top$ with $Q \in \mathbb{R}^{n \times d_{head}}$ per head costs
$2 n^2 d_{head}$ per head. Note the value-weighting $AV$ has identical shape
cost. Across $h_q$ query heads and $L$ blocks:

$$\text{FLOPs}_{\text{attn}} = 4 n^2 d_{head} h_q L$$

$$= 4 \times 8192^2 \times 128 \times 32 \times 64 = 7.04 \times 10^{13}$$

Note that GQA does not reduce this. Sharing KV heads reduces the *parameters* of
the K and V projections and the *cache*, but every query head still computes a
full score matrix against its assigned KV head. GQA is a memory optimization,
not a FLOP optimization - a distinction people routinely get wrong.

**The crossover.** The ratio of quadratic to linear work is

$$\frac{\text{FLOPs}_{\text{attn}}}{\text{FLOPs}_{\text{params}}} = \frac{4 n^2 d_{head} h_q L}{2 n P_a} = \frac{2 n\, d_{head} h_q L}{P_a}$$

which is linear in $n$. Setting it to 1 and solving gives the context length at
which attention equals parameter work:

$$n^* = \frac{P_a}{2 d_{head} h_q L} = \frac{22 \times 10^9}{2 \times 128 \times 32 \times 64} = \frac{22 \times 10^9}{524288} \approx 41{,}960$$

So for this configuration attention overtakes the MLPs at about 42k tokens. That
single number tells you more about a serving architecture than any benchmark:
below it, you are buying FLOPs for parameters; above it, you are buying FLOPs for
sequence length, and FlashAttention-style kernels (which reduce memory traffic,
not FLOPs) become the dominant engineering concern.

**Prefill intensity.** Bytes moved is dominated by reading all weights once:
$235 \times 10^9 \times 0.5 = 1.18 \times 10^{11}$ bytes at 4 bits. So

$$I_{\text{prefill}} = \frac{4.30 \times 10^{14}}{1.18 \times 10^{11}} \approx 3644 \gg 225$$

Compute-bound by a factor of 16. (This ignores activation traffic, which
FlashAttention exists to keep from spoiling the estimate.)

## Stage 4: inside one block - residual stream, heads, and the router

**The block as an additive update.** Write the residual stream at position $i$
entering block $\ell$ as $x_i^{(\ell)} \in \mathbb{R}^{d_{model}}$. A pre-norm
block computes

$$x_i^{(\ell+1)} = x_i^{(\ell)} + \text{Attn}^{(\ell)}\!\left(\text{RMSNorm}(x_{\leq i}^{(\ell)})\right)_i + \text{MLP}^{(\ell)}\!\left(\text{RMSNorm}(x_i^{(\ell)} + \text{Attn}(\cdot)_i)\right)$$

Unroll the recursion from $\ell = 0$ to $L$:

$$x_i^{(L)} = x_i^{(0)} + \sum_{\ell=0}^{L-1} \Delta_i^{(\ell)}$$

The final residual vector is *literally a sum* of the embedding and $L$ block
contributions. This is the formal content of the "residual stream as workspace"
claim, and it has two immediate consequences. First,
$\partial x^{(L)} / \partial x^{(0)} = I + \sum \partial \Delta / \partial x^{(0)}$,
so the identity term guarantees a gradient path of magnitude 1 regardless of
depth - the ResNet argument. Second, and more interesting, the logit for token
$j$ decomposes linearly:

$$z_j = \langle W_E[j],\ x^{(0)} \rangle + \sum_{\ell} \langle W_E[j],\ \Delta^{(\ell)} \rangle$$

Each block's contribution to each logit is separately measurable. This is the
**logit lens**, and it is only possible because the stream is additive.

**RMSNorm.** For $x \in \mathbb{R}^d$ with learned gain $g \in \mathbb{R}^d$:

$$\text{RMSNorm}(x) = \frac{x}{\sqrt{\frac{1}{d}\sum_{i=1}^{d} x_i^2 + \epsilon}} \odot g$$

No mean subtraction (unlike LayerNorm), no bias. The Jacobian
$\partial \text{RMSNorm}(x)_i / \partial x_j$ contains a term
$-x_i x_j / (d \cdot \text{rms}^3)$, which is what makes the normalization
*couple* all coordinates - and that coupling, not float range, is what
conditions the optimization.

**The router.** With router weights $W_r \in \mathbb{R}^{d_{model} \times E}$:

$$r = x^\top W_r \in \mathbb{R}^{E}, \quad \mathcal{T} = \text{top-}k(r), \quad g_e = \frac{\exp(r_e)}{\sum_{e' \in \mathcal{T}} \exp(r_{e'})} \ \ \text{for } e \in \mathcal{T}$$

$$\text{MoE}(x) = \sum_{e \in \mathcal{T}} g_e \cdot \text{Expert}_e(x)$$

The softmax is over the selected $k = 8$ only, so the gates sum to 1. The top-$k$
operation is not differentiable in its selection, but it is differentiable in
the gates $g_e$, which is what lets gradient descent train the router: an expert
that produced a useful output gets its gate pushed up, which makes it more likely
to be selected next time. That positive feedback is exactly why load balancing is
required. The auxiliary loss, in the Switch Transformer form, is

$$\mathcal{L}_{aux} = \alpha \cdot E \sum_{e=1}^{E} f_e \cdot P_e$$

where $f_e$ is the fraction of tokens in the batch routed to expert $e$ and $P_e$
is the mean router probability assigned to $e$ over the batch. It is minimized
when both are uniform at $1/E$, and it is what prevents collapse to a handful of
experts.

## Stage 5: the last position becomes a distribution

**Temperature is a reparameterization of the softmax, not new information.**
With logits $z \in \mathbb{R}^V$:

$$p_j(T) = \frac{\exp(z_j / T)}{\sum_{k} \exp(z_k / T)}$$

Two limits worth deriving. As $T \rightarrow 0^+$, write
$z_{max} = \max_k z_k$ and factor it out:

$$p_j(T) = \frac{\exp((z_j - z_{max})/T)}{\sum_k \exp((z_k - z_{max})/T)}$$

Every numerator with $z_j < z_{max}$ has a negative exponent divided by
$T \rightarrow 0^+$, so it tends to $e^{-\infty} = 0$. The distribution
converges to a point mass on the argmax. As $T \rightarrow \infty$, every
exponent tends to 0, every term tends to 1, and $p \rightarrow$ uniform over
$V$. So temperature interpolates between argmax and uniform over a *fixed* $z$.

**Why the same-difference invariance matters.** $p$ depends on $z$ only through
differences: adding a constant $c$ to every logit leaves $p$ unchanged, since
$e^{(z_j+c)/T} = e^{c/T} e^{z_j/T}$ and the factor cancels. So absolute logit
values are meaningless, and any interpretation of a single logit as a confidence
score is unfounded.

**Gradient of the log-likelihood at this layer**, since it closes the loop with
u5. For cross-entropy loss $\mathcal{L} = -\log p_y$ on true token $y$:

$$\frac{\partial \mathcal{L}}{\partial z_j} = p_j - \mathbb{1}[j = y]$$

This is the cleanest object in the whole subject: the gradient on the logits is
the predicted distribution minus the one-hot truth. Everything backprop does
from here is the chain rule applied to that residual. Note it is exactly zero
when $p$ is the one-hot truth, which never happens, because the entropy of
language is not zero.

**Top-p, formally.** Sort $p$ descending as $p_{(1)} \geq p_{(2)} \geq \dots$.
Let $m$ be the smallest index with $\sum_{i=1}^{m} p_{(i)} \geq p_{top}$. Keep
$\{(1), \dots, (m)\}$, zero the rest, renormalize by dividing by
$\sum_{i \leq m} p_{(i)}$. The dependence on $m$ being the *smallest* such index
is where learners go wrong: it means the nucleus can be a single token when the
distribution is peaked, and thousands when it is flat, which is exactly the
adaptivity that makes top-p preferable to top-k.

## Stage 6: decode, and what the KV cache actually bought

**Decode intensity.** Bytes moved per decode step:

$$B_{\text{decode}} = \underbrace{P_a \cdot b_w}_{\text{weights}} + \underbrace{n \cdot 2 h_{kv} d_{head} b_{kv}}_{\text{cache}}$$

with $b_w = 0.5$ bytes/param at 4 bits and $b_{kv} = 2$ bytes at fp16. At
$n = 8192$: $11 \times 10^9 + 8192 \times 262144 = 11 \times 10^9 + 2.15 \times 10^9 = 1.32 \times 10^{10}$ bytes.

FLOPs per decode step: $2 P_a + 4 n d_{head} h_q L = 4.4 \times 10^{10} + 4 \times 8192 \times 128 \times 32 \times 64 = 4.4 \times 10^{10} + 8.6 \times 10^9 \approx 5.3 \times 10^{10}$.

$$I_{\text{decode}} = \frac{5.3 \times 10^{10}}{1.32 \times 10^{10}} \approx 4.0 \ll 225$$

Memory-bound by a factor of 56. The accelerator's arithmetic units run at about
1.8% utilization during decode. That single number explains batching, speculative
decoding, and quantization all at once: they are three different ways to raise
$I$ toward $I^*$.

**Batching, derived.** Serve $b$ independent requests simultaneously. The weight
read is shared across all $b$, so

$$I_{\text{decode}}(b) \approx \frac{b \cdot 2 P_a}{P_a b_w + \sum_{j=1}^{b} n_j \cdot 262144}$$

The numerator scales with $b$; the weight term in the denominator does not. So
throughput rises nearly linearly with batch size until either (a) $I$ reaches
$I^*$ and you become compute-bound, or (b) the KV term
$\sum_j n_j \cdot 262144$ overtakes the weight term. Setting the KV term equal to
the weight term: $b \cdot n \cdot 262144 = 11 \times 10^9$, so at $n = 8192$,
$b \approx 5$. Past a handful of concurrent long-context requests, the KV cache
*is* your bandwidth budget. This is why production inference servers do paged
attention and KV quantization rather than simply raising the batch size.

**Speculative decoding, expected speedup.** A draft model proposes $\gamma$
tokens; the target verifies all $\gamma + 1$ positions in one forward pass. If
each drafted token is accepted independently with probability $\alpha$, the
expected number of tokens accepted per target pass is

$$\mathbb{E}[\text{accepted}] = \frac{1 - \alpha^{\gamma+1}}{1 - \alpha}$$

With $\alpha = 0.8$, $\gamma = 4$: $(1 - 0.8^5)/(1 - 0.8) = (1 - 0.328)/0.2 = 3.36$
tokens per target pass. The target pass costs essentially the same as a single
token's pass because it is memory-bound and $\gamma + 1 = 5$ is nowhere near
enough to leave the memory-bound regime ($I$ rises from 4 to 20, still well under
225). Subtract the draft model's cost and you land near a 2.5-3x speedup. The
derivation makes the failure mode obvious: if $\alpha$ is low (draft and target
disagree often) the expected acceptance collapses toward 1 and you have paid for
the draft model for nothing.

## Stage 7: the tool call, the break, and the resume

**Why prefix cache validity is a theorem, not a heuristic.** With causal masking,
the key and value at position $i$ in block $\ell$ are functions of
$x_i^{(\ell)}$, and by induction $x_i^{(\ell)}$ is a function only of tokens
$t_0, \dots, t_i$. Formally, for all $\ell$ and all $i$:

$$k_i^{(\ell)},\ v_i^{(\ell)} = \phi^{(\ell)}(t_0, \dots, t_i)$$

with no dependence on $t_{i+1}, \dots$. Therefore appending tokens at positions
$> i$ cannot change any cached entry at position $\leq i$. Conversely, modifying
$t_m$ invalidates every cached entry at position $\geq m$ in every block, because
the induction chain passes through $m$.

This gives the exact cost of an edit at position $m$ in a sequence of length $n$:

$$\text{cost}(\text{edit at } m) = 2 P_a (n - m) + 4 (n-m) \cdot n \cdot d_{head} h_q L$$

versus $\text{cost}(\text{append of } j) = 2 P_a j + 4 j (n + j) d_{head} h_q L$.
Editing at $m = 40$ in an 11,632-token sequence costs roughly 290x an
appending-only turn of the same size. Order your prompt by mutation frequency,
ascending.

**Session cost accumulation.** Let turn $i$ append $a_i$ tokens (tool result plus
generation) and generate $g_i$ tokens, and let $n_i = \sum_{j<i} a_j$ be the
context at the start of turn $i$. Total session time is

$$T = \sum_{i=1}^{K} \left[ \frac{2 P_a a_i + 4 a_i n_i d_{head} h_q L}{F} + g_i \frac{B_w + n_i b_{kv}}{M} \right]$$

Take $a_i = a$ and $g_i = g$ constant so $n_i = a(i-1)$. Then the terms
containing $n_i$ contribute $\sum_{i} n_i = a \frac{K(K-1)}{2}$, which is
$\Theta(K^2)$, while the terms without contribute $\Theta(K)$. So total session
cost is

$$T = \Theta(K) + \Theta(K^2)$$

with the quadratic term carrying coefficients $4 a^2 d_{head} h_q L / F$ (prefill
attention) and $g a b_{kv} / M$ (KV cache reads during decode). Both are
quadratic in turn count, and they come from *different* mechanisms - one from
attention's $n^2$, one from the cache's linear growth multiplied by a linear
number of decode steps. That is the derivation behind "an agentic session costs
quadratically-ish."

## The cost model that falls out of the trace

Collect everything into one expression and read off the partial derivatives,
because sensitivity is what you actually need when deciding where to spend
engineering effort.

$$t = \frac{2 P_a n_{new} + 4 n_{new} n\, d_{head} h_q L}{F} + g \cdot \frac{P_a b_w + n\, b_{kv}}{M}$$

$$\frac{\partial t}{\partial P_a} = \frac{2 n_{new}}{F} + \frac{g\, b_w}{M}, \qquad \frac{\partial t}{\partial b_w} = \frac{g P_a}{M}, \qquad \frac{\partial t}{\partial b_{kv}} = \frac{g n}{M}$$

$$\frac{\partial t}{\partial n} = \frac{4 n_{new} d_{head} h_q L}{F} + \frac{g\, b_{kv}}{M}, \qquad \frac{\partial t}{\partial n_{new}} = \frac{2 P_a + 4 n\, d_{head} h_q L}{F}$$

Plug in the running numbers with $n = 8192$, $n_{new} = 3400$, $g = 400$:

- $\partial t / \partial b_w$: $400 \times 22\times10^9 / 2\times10^{12} = 4.4$ s
  per byte-per-parameter. Going from fp16 (2 bytes) to 4-bit (0.5 bytes) saves
  $1.5 \times 4.4 = 6.6$ s of a turn. Enormous.
- $\partial t / \partial b_{kv}$: $400 \times 8192 / 2\times10^{12} = 1.6 \times 10^{-6}$ s
  per byte-per-token-of-cache. At $b_{kv} = 262144$ that term is 0.42 s. Halving
  KV precision saves 0.21 s at 8k context - and 2.6 s at 100k, since the
  derivative itself is linear in $n$.
- $\partial t / \partial n_{new}$: $(4.4\times10^{10} + 4 \times 8192 \times 128 \times 32 \times 64)/4.5\times10^{14} = (4.4\times10^{10} + 8.6\times10^9)/4.5\times10^{14} \approx 1.2\times10^{-4}$ s
  per token. So each fresh (uncached) prompt token costs about 0.12 ms of
  prefill - and each *cached* one costs zero. A 90% cache hit rate on an
  11,632-token prompt saves about 1.3 s per turn.

The sensitivities are not close to each other. Weight precision and prefix cache
hit rate dominate at short context; KV cache bytes dominate at long context;
active parameter count matters roughly equally in both. Optimize in that order,
and re-derive the order whenever $n$ moves by an order of magnitude, because
$\partial t / \partial b_{kv}$ is the only one that grows with $n$.
