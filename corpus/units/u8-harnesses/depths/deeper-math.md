---
unit: u8
depth: deeper-math
---

## It is all one token stream

The claim that "the system prompt is re-read every forward pass" has an exact
form worth stating, because it bounds what any prompt-level defense can do.

Fix a context of $n$ tokens with embeddings $x_0, \ldots, x_{n-1}$, each
$x_i \in \mathbb{R}^{d_{\text{model}}}$. In a causal transformer the hidden state
at position $i$ in layer $\ell$ is a function of exactly the positions $\le i$
in layer $\ell - 1$:

$$h_i^{(\ell)} = F^{(\ell)}\!\left(h_0^{(\ell-1)}, \ldots, h_i^{(\ell-1)}\right)$$

where $F^{(\ell)}$ is the block (attention plus MLP) and the restriction to
indices $\le i$ is the causal mask. Unrolling over $L$ layers, the output at the
last position depends on every earlier position through $L$ rounds of mixing.
There is no persistent state variable carried between forward passes - no $s_t$
of the RNN kind. The function is $\text{logits} = f_\theta(\text{tokens})$, full
stop, and $\theta$ is frozen at inference.

The security consequence is sharp. Any "instruction hierarchy" is a property of
$f_\theta$ - a region of parameter space where sequences containing a system
delimiter followed by imperative text produce compliant continuations. It is
learned, so it is graded and it interpolates. There is no term in the
architecture that could make it a hard constraint, because the architecture has
no place to put one: the only inputs are token IDs, and every token ID is
handled by the same weights.

**Attention dilution, quantified.** Consider one attention head at some query
position, with pre-softmax scores $s_1, \ldots, s_n$ over $n$ keys. Suppose key 1
is the one you care about (a system-prompt instruction) and the other $n-1$
keys are "background" with scores concentrated near $m$. The weight on key 1 is

$$\alpha_1 = \frac{e^{s_1}}{e^{s_1} + \sum_{j \ge 2} e^{s_j}}
\approx \frac{e^{s_1}}{e^{s_1} + (n-1)e^{m}}
= \frac{1}{1 + (n-1)e^{m - s_1}}$$

To hold $\alpha_1$ fixed as $n$ grows, the margin $s_1 - m$ must grow like
$\log n$. Scores are bounded in practice (bounded activations, bounded weight
norms, and the $1/\sqrt{d_k}$ scaling from u3 exists precisely to keep them in
the unsaturated region), so the margin cannot grow without bound. Therefore
attention on any fixed instruction decays as context grows, roughly as $1/n$
once the background dominates. This is the formal content of "your system prompt
gets crowded out". It is not a metaphor and it is not a bug in the model - it is
what a softmax over a growing key set does.

## The tool loop: the agent is the loop

**Constrained decoding, precisely.** Schema conformance can be made hard rather
than statistical, and doing so is a sampler-side operation. Let $V$ be the
vocabulary and let $A_t \subseteq V$ be the set of tokens that keep the emitted
string within the grammar (a JSON schema compiled to a pushdown automaton, with
$A_t$ read off the automaton's current state). Constrained decoding replaces the
logits $z \in \mathbb{R}^{|V|}$ with

$$\tilde{z}_v = \begin{cases} z_v & v \in A_t \\ -\infty & v \notin A_t\end{cases}$$

before the softmax, which zeroes the probability of any non-conforming token.
Note where this happens: after the forward pass, on the logit vector, in the
harness's sampler. The model is unchanged and unaware. This is a clean instance
of the unit's thesis - a capability you might attribute to "the model following
the schema" is a mask applied to a vector outside the model.

It is not free. Renormalizing over $A_t$ redistributes mass the model assigned
to tokens it considered likely, which can force continuations the unconstrained
distribution assigned low probability. Grammar-valid and semantically correct
are different properties.

## What the loop costs

**Forward-pass FLOPs.** For a decoder-only transformer with $N$ active
parameters (u7: active, not total - MoE changes $N$, not the shape of this
formula), $L$ layers, and model width $d_{\text{model}}$, a standard estimate for
the forward cost of one token at context position $n$ is

$$C_{\text{token}}(n) \approx \underbrace{2N}_{\text{matmuls}} +
\underbrace{4\,L\,n\,d_{\text{model}}}_{\text{attention against } n \text{ keys}}$$

The first term is the "2 FLOPs per parameter per token" rule (one multiply, one
add). The second is attention: each of $L$ layers computes $n$ query-key dot
products and an $n$-weighted value sum - two matmuls of $n \cdot d_{\text{model}}$
multiply-adds each, at 2 FLOPs apiece. Constants vary by a factor of two across sources; the scaling
is what matters.

Prefilling a fresh context of $n$ tokens sums this over positions:

$$C_{\text{prefill}}(n) \approx \sum_{i=0}^{n-1}\left(2N + 4 L i\, d_{\text{model}}\right)
= 2Nn + 2 L n^2 d_{\text{model}}$$

which is the $O(n^2)$ of u3 with its constant made explicit. The two terms are
equal when

$$2N = 4 L n\, d_{\text{model}} \quad\Longrightarrow\quad n^{*} = \frac{N}{2\,L\,d_{\text{model}}}$$

For $N = 70 \times 10^9$, $L = 80$, $d_{\text{model}} = 8192$:

$$n^{*} = \frac{70 \times 10^9}{2 \times 80 \times 8192} = \frac{70 \times 10^9}{1{,}310{,}720} \approx 53{,}000$$

Below roughly 50k tokens, you are paying for weights. Above it, you are paying
for attention, and the marginal token gets steadily more expensive. This is the
quantitative version of "long context costs what it costs" (M10).

**Why cached delta-prefill is still not constant-time.** With exact-prefix
reuse, step $i$ prefills only $d$ new tokens - but those tokens sit at offset
$n_i$, and each must attend to all $n_i$ keys behind it:

$$C_{\text{step}} \approx 2Nd + 2L d_{\text{model}} \sum_{j=n_i}^{n_i + d} j
\approx 2Nd + 2L\,d_{\text{model}}\,d\,n_i \quad (d \ll n_i)$$

Linear in $n_i$. So caching turns the $O(k^2)$ token count into $O(k)$, but the
per-step FLOP cost still grows linearly with how much context sits behind the
append point, and the session's total attention FLOPs remain quadratic in $k$.
Caching removes one quadratic. It does not remove the other.

**The naive-versus-cached ratio in closed form.** With $n_0$ initial tokens,
$d$ per step, $k$ steps:

$$\frac{T_{\text{naive}}}{T_{\text{cached}}}
= \frac{(k+1)n_0 + d\,k(k+1)/2}{n_0 + kd}$$

As $k \to \infty$ this behaves like $\frac{d k^2/2}{dk} = k/2$: the waste factor
grows linearly in step count without bound. At $k = 40$ with the unit's
constants ($n_0 = 4000$, $d = 600$) it is $656{,}000 / 28{,}000 = 23.4$, running
ahead of the $k/2 = 20$ asymptote because the $n_0$ term has not yet washed out. There is no session length at which skipping cache reuse
stops mattering; it gets monotonically worse.

## Context is the scarce resource

**KV bytes, and where GQA enters.** Restating the formula with every symbol
defined: the cache holds, for each of $n$ positions, in each of $L$ layers, for
each of $H_{kv}$ key/value heads, one key vector and one value vector of
dimension $d_{\text{head}}$, at $b$ bytes per element:

$$M_{KV} = 2\, n\, L\, H_{kv}\, d_{\text{head}}\, b$$

Multi-head attention (u4) has $H_{kv} = H_q$. Grouped-query attention shares one
K/V head across $g$ query heads, so $H_{kv} = H_q / g$, and $M_{KV}$ drops by
exactly $g$. Multi-query attention is $g = H_q$, so $H_{kv} = 1$. The compute of
the attention itself is essentially unchanged - the queries still all get
computed - which is the point: GQA is a memory-bandwidth optimization aimed at
this formula, not a FLOP optimization.

Concretely, a 32-layer model with $H_q = 32$, $d_{\text{head}} = 128$, fp16:

- MHA ($H_{kv} = 32$): $2 \cdot 32 \cdot 32 \cdot 128 \cdot 2 = 524{,}288$ B/token $= 512$ KiB
- GQA, $g = 4$ ($H_{kv} = 8$): $131{,}072$ B/token $= 128$ KiB
- MQA ($H_{kv} = 1$): $16{,}384$ B/token $= 16$ KiB

At 200k context that is 105 GB, 26 GB, and 3.3 GB respectively. GQA is the
difference between a long-context session fitting on one accelerator and not.

**Memory bandwidth is the real decode bottleneck.** Decoding one token reads the
entire KV cache from HBM. At $n = 200{,}000$ with the GQA numbers above, that is
26 GB read per token generated. On an accelerator with roughly 3 TB/s of
bandwidth, that alone floors the per-token latency near 9 ms before any
arithmetic, and it grows linearly in $n$. Long-context decoding is
bandwidth-bound, not FLOP-bound, which is also why speculative decoding (u6)
pays so well there: it amortizes one cache read across several accepted tokens.

**A budget model for sub-agents.** Suppose a subtask requires $E$ tokens of
exploration and produces $R$ tokens of conclusion, and the parent turn will run
for $m$ more forward passes afterward. Inline, the parent's context carries $E +
R$ extra tokens through all $m$ passes; delegated, it carries $R$. Even with
perfect prefix caching, where each token is prefilled once, the difference
persists in decode-time attention: every subsequent generated token attends over
$E$ extra keys. The delegation wins whenever $E \gg R$, and it wins more the
larger $m$ is - which is why delegation early in a long session is worth far
more than delegation at the end.

## Putting the three claims together

The whole layer is one composition. Let $\sigma$ be the harness's serializer
mapping a structured conversation state $S$ to a token sequence, $f_\theta$ the
network, $\text{samp}$ the sampler (temperature, top-p, plus any logit masks),
$\pi$ the harness's parser mapping emitted text to either a tool call or a
terminal answer, and $\text{exec}$ the tool dispatcher. One loop iteration is

$$S_{t+1} = S_t \,\Vert\, \text{exec}\!\big(\pi(\text{samp}(f_\theta(\sigma(S_t))))\big)$$

where $\Vert$ is append. Read off what is where. The only appearance of $\theta$
is $f_\theta$, a pure function of a token array. Everything else - roles, tool
semantics, permissions, stop policy, retries, compaction (which is the one
operation that violates append-only and instead rewrites $S_t$) - is in
$\sigma$, $\pi$, and $\text{exec}$, all of which are your code. The KV cache is a
memoization of $f_\theta$ over prefixes of $\sigma(S_t)$, valid exactly when
$\sigma(S_{t+1})$ extends $\sigma(S_t)$, which is exactly when the update is
append-only.

