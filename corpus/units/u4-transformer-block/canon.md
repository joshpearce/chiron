---
unit: u4
title: "The transformer block"
concepts: [c-mha, c-mlp-block, c-residual, c-layernorm, c-depth]
assumes: [c-notation, c-matmul, c-dotprod, c-softmax, c-embeddings, c-qkv, c-sdpa, c-attn-numeric, c-causal-mask]
---

You can now compute attention by hand. Attention alone is a single
similarity-weighted average - it moves information between positions and does
nothing else. A working language model is roughly 100 copies of a small
assembly in which attention is one of two components. This unit is that
assembly: what the other component is, what the wiring between them actually
does, and why stacking the assembly buys anything.

## The block in one equation

A transformer block is a function from a sequence of vectors to a sequence of
vectors of the *same shape*. Let $x \in \mathbb{R}^{n \times d}$ be the input,
where $n$ is the number of token positions and $d$ is the model width
(`d_model`: 768 for GPT-2 small, 4096 for an 8B-class model). The modern block
is two lines:

$$ h = x + \mathrm{MHA}\!\left(\mathrm{Norm}_1(x)\right) $$

$$ y = h + \mathrm{MLP}\!\left(\mathrm{Norm}_2(h)\right) $$

with $h, y \in \mathbb{R}^{n \times d}$. $\mathrm{MHA}$ is multi-head attention
(section 2). $\mathrm{MLP}$ is a two-matrix position-wise feed-forward network
(section 4). $\mathrm{Norm}$ is LayerNorm or RMSNorm (section 7). The two `+`
signs are the residual connections (section 6).

Three properties of this equation carry most of the unit:

1. **The shape never changes.** $x$, $h$, $y$ are all $n \times d$. That is why
   you can stack $L$ of these back to back with no glue code. It is also why
   the thing flowing through the stack - the *residual stream* - is a single
   fixed-width channel, $d$ dimensions wide, that every block shares.
2. **Both sublayers write additively.** Neither sublayer produces $y$. Each
   produces a *delta* that is added to what was already there. Nothing is
   overwritten.
3. **$\mathrm{MLP}$ is position-wise.** It maps $\mathbb{R}^d \to \mathbb{R}^d$
   and is applied to each of the $n$ token vectors independently, with the same
   weights. Only $\mathrm{MHA}$ moves information *between* positions. If you
   delete attention, every position becomes an independent MLP stack that never
   sees another token.

That last point is the cleanest division of labour in the architecture:
attention is the only cross-position operation; everything else is per-token.

Note that $n$ appears nowhere in the parameter count. A block's parameters are
functions of $d$ alone. Sequence length changes activation memory and compute,
never weights.

## Multi-head attention: many subspaces, one parameter budget

<!-- refutes: U4M1 -->

You probably think that $h$ heads means $h$ copies of attention, so $h$ times
the attention parameters and $h$ times the attention cost - and that heads exist
because "different heads learn different things, like syntax versus semantics."

Here is the prediction that fails. If heads multiplied cost, moving GPT-2 small
from 12 heads to 1 head would cut its attention parameters by 12x. Count them:
it does not change the parameter count at all. A 12-head block and a 1-head
block of the same $d$ have byte-identical weight shapes.

The model is appealing because "multi-head" sounds like replication, and
because published head-attribution work (this head tracks subject-verb
agreement, this one is a previous-token head) is real. But head specialization
is a *consequence*, not the mechanism, and heads are not copies.

Here is what is actually true. $h$ heads *partition* the same fixed budget.

Take the query projection. In a single-head formulation you would have
$W^Q \in \mathbb{R}^{d \times d}$. In multi-head, you still have
$W^Q \in \mathbb{R}^{d \times d}$; you slice its columns into $h$ contiguous
blocks of width $d_k = d/h$:

$$ W^Q = \big[\, W^Q_1 \;\big|\; W^Q_2 \;\big|\; \cdots \;\big|\; W^Q_h \,\big], \qquad W^Q_i \in \mathbb{R}^{d \times d_k} $$

Same for $W^K$ and $W^V$. Head $i$ runs the attention you already know, at width
$d_k$ instead of $d$:

$$ \mathrm{head}_i = \mathrm{softmax}\!\left(\frac{(xW^Q_i)(xW^K_i)^{\top}}{\sqrt{d_k}} + M\right) x W^V_i \;\in\; \mathbb{R}^{n \times d_k} $$

where $M$ is the causal mask (zeros on and below the diagonal, $-\infty$ above)
and the softmax is row-wise. The $h$ head outputs, each $n \times d_k$, are
concatenated back to width $d$ and passed through one output projection
$W^O \in \mathbb{R}^{d \times d}$:

$$ \mathrm{MHA}(x) = \big[\mathrm{head}_1 \,\big|\, \cdots \,\big|\, \mathrm{head}_h\big] \, W^O \;\in\; \mathbb{R}^{n \times d} $$

So the whole of MHA is four $d \times d$ matrices - $W^Q, W^K, W^V, W^O$ -
regardless of $h$. In an implementation the per-head slicing is a `reshape`,
not extra storage.

```beat
id: u4-b1
type: predict
concept: c-mha
prompt: |
  A block has $d = 512$. Configuration A uses 1 attention head of width
  $d_k = 512$. Configuration B uses 8 heads of width $d_k = 64$. Before
  reading on, commit to an answer: how do B's attention-sublayer parameters
  compare to A's, and what does B buy with whatever resource it does spend?
options:
  - text: "Identical counts - four $512 \\times 512$ matrices ($W^Q, W^K, W^V, W^O$), $4 \\times 262{,}144 = 1{,}048{,}576$ either way. B buys 8 simultaneous attention distributions per query position instead of 1, and pays in per-head rank: each head's score-forming matrix is limited to rank $d_k = 64$ instead of 512."
    correct: true
    explain: "Right. Heads slice the four fixed $d \\times d$ matrices into contiguous column blocks of width $d_k = d/h$; nothing is added. What changes is that A must mix all 512 output dimensions across positions with one softmax row, while B can copy from position 3 into one 64-dim subspace and from position 17 into another in the same layer."
  - text: "B has 8x the parameters ($8{,}388{,}608$ against $1{,}048{,}576$), because each head needs its own $W^Q, W^K, W^V$; that extra capacity is what lets different heads specialise in different linguistic phenomena."
    misconception: U4M1
    explain: "This is the replication model, and it fails on the count. In multi-head attention $W^Q$ is still $d \\times d$; head $i$ uses the column block $W^Q_i \\in \\mathbb{R}^{d \\times d_k}$. Reconfiguring 8 heads to 1 at fixed $d$ produces byte-identical weight shapes. Specialisation is an emergent consequence of having several restricted read/write channels, not the mechanism that pays for them."
  - text: "Identical counts, but B gains nothing an equally wide single head could not do - $\\mathrm{concat}[\\mathrm{head}_1 | \\cdots | \\mathrm{head}_8]W^O$ is just a re-partition of the same linear map, so B is a bookkeeping convention."
    misconception: U4M6
    explain: "The parameter claim is right and the conclusion is wrong. The softmax sits between the slices, so the heads are not one linear map: A computes exactly one probability distribution over source positions per query, B computes eight, each on a different learned subspace. That is a strictly larger set of functions, at the cost of each head's $W^Q_i W^{K\\top}_i$ having rank at most 64."
  - text: "B has 1/8 the parameters, since each head's matrices are only $512 \\times 64$ and there are the same four of them; multi-head attention is a parameter-saving factorization."
    misconception: U4M1
    explain: "There are $8 \\times 4$ such slices, not 4: the per-head $d \\times d_k$ blocks tile the full $d \\times d$ matrices, so they sum back to $4d^2 = 1{,}048{,}576$. Head count is neutral in both directions - it neither multiplies nor divides the budget, it only sets how the budget is sliced."
check: choice
```

The reason to want several patterns at once is mechanical, not linguistic. One
softmax row is one probability distribution over source positions. With one
head, if position 12 needs the subject from position 3 *and* the verb tense
from position 9, it must average them with one set of weights and lose both.
With eight heads it takes both, into different subspaces, in one layer.

The price is rank. Head $i$'s score matrix is
$x W^Q_i (x W^K_i)^{\top} = x (W^Q_i W^{K\top}_i) x^{\top}$, and
$W^Q_i W^{K\top}_i \in \mathbb{R}^{d \times d}$ has rank at most $d_k = 64$.
A single 512-wide head would have a full-rank $512 \times 512$ score-forming
matrix: one very expressive pattern. Multi-head trades pattern expressiveness
for pattern count. Empirically, count wins - up to a point; heads much narrower
than about 64 dimensions start to hurt.

<!-- fade: attention-param-count -->

Work the count once, fully, so the procedure is mechanical:

**Procedure: attention parameter count.** Given $d$ and $h$ (biases ignored):

1. $d_k = d / h$. For $d = 4096, h = 32$: $d_k = 128$.
2. $W^Q$: $d \times d = 4096 \times 4096 = 16{,}777{,}216$.
3. $W^K$: same shape, $16{,}777{,}216$. $W^V$: same, $16{,}777{,}216$.
4. $W^O$: $d \times d = 16{,}777{,}216$.
5. Total $= 4d^2 = 67{,}108{,}864$ parameters, and $h$ appeared only in step 1,
   where it cancelled out of everything downstream.

```beat
id: u4-b2
type: completion
concept: c-mha
prompt: |
  Fill the blanks. Model: $d = 768$, $h = 12$, no biases.

  1. Head width: $d_k = 768 / 12 = $ ____
  2. $W^Q$ has shape $768 \times 768$, so $589{,}824$ parameters.
  3. $W^K$ and $W^V$ have the same shape, adding $2 \times 589{,}824
     = 1{,}179{,}648$.
  4. $W^O$ has shape ____ $\times$ ____, adding $589{,}824$.
  5. Attention sublayer total $=$ ____ parameters, which equals $4d^2$.
  6. Now change $h$ to $4$, holding $d = 768$. The new total is ____ ,
     because ____ .

  Which set of fillings is right?
options:
  - text: "(1) $64$; (4) $768 \\times 768$; (5) $2{,}359{,}296$; (6) $2{,}359{,}296$ - unchanged - because the head count only sets how the four fixed $d \\times d$ matrices are sliced; $h = 4$ changes $d_k$ to $192$ and therefore each head's rank, not the parameter count."
    correct: true
    explain: "Correct throughout. $d_k = d/h$ appears only in the slicing; $W^Q, W^K, W^V, W^O$ are each $768 \\times 768$ regardless, so the total is $4d^2 = 2{,}359{,}296$ at $h = 12$ and at $h = 4$."
  - text: "(1) $64$; (4) $768 \\times 768$; (5) $2{,}359{,}296$; (6) $786{,}432$ - one third of the previous total - because dropping from 12 heads to 4 removes two thirds of the per-head $W^Q, W^K, W^V$ machinery."
    misconception: U4M1
    explain: "Steps 1, 4 and 5 are right, but step 6 treats heads as copies. There is no per-head machinery to remove: the 4 heads at $h = 4$ tile the same $768 \\times 768$ matrices in blocks of width 192 that the 12 heads tiled in blocks of width 64. The total stays $2{,}359{,}296$."
  - text: "(1) $64$; (4) $64 \\times 768$; (5) $2{,}359{,}296$; (6) $2{,}359{,}296$ - unchanged - because $W^O$ absorbs the head count by growing as $h$ shrinks."
    misconception: U4M1
    explain: "$W^O$ is $d \\times d = 768 \\times 768$; it maps the concatenation of all $h$ head outputs (total width $h \\cdot d_k = d$) back to width $d$. The $64 \\times 768$ shape is one row-block $W^O_i$ of the split used in the identity $\\sum_i \\mathrm{head}_i W^O_i$, not the whole matrix, and nothing needs to grow to compensate: the total is $h$-independent because every matrix is $d \\times d$."
  - text: "(1) $192$; (4) $768 \\times 768$; (5) $2{,}359{,}296$; (6) $2{,}359{,}296$ - unchanged - because $d_k$ is fixed by convention at $d/4$ and the head count is only a scheduling choice."
    misconception: U4M2
    explain: "$d_k = d/h = 768/12 = 64$ at step 1; $192$ is the value for the $h = 4$ configuration in step 6. Head width is set by the head count, not fixed by convention - and it is $d_k$, not the parameter total, that moves when $h$ moves."
check: choice
```

One more structural fact, because it connects heads to the residual stream and
is used again in section 6. Split $W^O$ by *rows* into $h$ blocks
$W^O_i \in \mathbb{R}^{d_k \times d}$. Then concatenate-then-project is
identical to project-then-sum:

$$ \big[\mathrm{head}_1 \,\big|\, \cdots \,\big|\, \mathrm{head}_h\big] W^O \;=\; \sum_{i=1}^{h} \mathrm{head}_i \, W^O_i $$

This is an algebraic identity, not an approximation. Each head independently
computes an $n \times d$ update and the sublayer output is their sum. Since
$\mathrm{head}_i W^O_i$ factors through $d_k$ dimensions, each head's write has
rank at most $d_k$: a head writes into a $\le 64$-dimensional subspace of the
$d$-dimensional stream and leaves the rest untouched. Heads are independent
readers and writers on a shared bus, not stages of one computation.

## MQA and GQA: the same equation with fewer key-value heads

<!-- refutes: U4M2 -->
<!-- refutes: U4M3 -->

Multi-query attention (MQA) and grouped-query attention (GQA) are the two
variants you will meet in every current model card, and they are confusable in
a specific way. Get the definitions exactly right first, then the reason.

Let $h$ be the number of **query** heads and $g$ the number of **key/value**
heads, with $g$ dividing $h$.

- **MHA**: $g = h$. Every query head has its own $K$ and its own $V$.
- **MQA**: $g = 1$. All $h$ query heads share a *single* $K$ and a single $V$.
- **GQA**: $1 < g < h$. Query heads are partitioned into $g$ groups of $h/g$;
  each group shares one $K$ and one $V$.

GQA is the family; MHA ($g = h$) and MQA ($g = 1$) are its endpoints. Nothing
else about the equation changes - each query head still forms its own scores,
its own softmax, its own weighted average. It merely dots against a key matrix
that some other query heads are also using. In code the difference is one
`repeat_interleave` on $K$ and $V$ before the scores are formed.

Parameters, for $d = 4096$, $h = 32$, $d_k = 128$, $g = 8$:

| matrix | MHA shape | MHA params | GQA $g{=}8$ shape | GQA params |
|---|---|---|---|---|
| $W^Q$ | $4096 \times 4096$ | 16,777,216 | $4096 \times 4096$ | 16,777,216 |
| $W^K$ | $4096 \times 4096$ | 16,777,216 | $4096 \times 1024$ | 4,194,304 |
| $W^V$ | $4096 \times 4096$ | 16,777,216 | $4096 \times 1024$ | 4,194,304 |
| $W^O$ | $4096 \times 4096$ | 16,777,216 | $4096 \times 4096$ | 16,777,216 |
| total | | 67,108,864 | | 41,943,040 |

(1024 = $g \cdot d_k = 8 \times 128$.) In closed form,
$2d^2 + 2d^2 \cdot g/h$, which is $4d^2$ at $g = h$ and $2d^2$ in the limit.
GQA at $g = 8$ removes 37.5% of the attention sublayer's parameters.

Now the part that is routinely got wrong.

You probably think GQA is a compute optimization - fewer key heads, fewer
FLOPs, faster attention.

Here is the prediction that fails. If GQA cut attention FLOPs by $h/g = 4$, a
prompt-processing benchmark (a big parallel forward pass over 8k tokens, which
is compute-bound) would run ~4x faster on the attention kernels. It does not.
It runs at essentially the same speed. The score matrix is still $h$ query
heads against $n$ positions: $h \cdot n \cdot d_k$ multiply-accumulates per new
token, 33.5M of them at $h=32, n=8192, d_k=128$, *identical* under MHA, GQA and
MQA. The shared keys are broadcast, not skipped.

The model is appealing because in every other systems context, "fewer of X"
means "less work." Here fewer of X means less *state*.

Here is what is actually true. GQA is a memory optimization, and the resource
it saves is KV-cache bytes and the bandwidth to stream them.

<!-- fade: kv-cache-per-token -->

**Procedure: KV cache per token.** During generation, each already-processed
token's $K$ and $V$ vectors are cached so they are not recomputed. Per token,
per layer:

1. Vectors cached per token per layer: one $K$ and one $V$ per **KV head**, so
   $2g$ vectors.
2. Each is $d_k$ wide, at $b$ bytes per element (fp16: $b = 2$).
3. Bytes per token per layer $= 2 \cdot g \cdot d_k \cdot b$.
   MHA, $g = 32$: $2 \times 32 \times 128 \times 2 = 16{,}384$ B = 16 KiB.
4. Multiply by layers $L = 32$: $16{,}384 \times 32 = 524{,}288$ B = 512 KiB
   per token.
5. Multiply by context $n = 8192$: $4{,}294{,}967{,}296$ B = **4.00 GiB** for
   one sequence.

Same procedure at $g = 8$ gives 4 KiB per token per layer, 128 KiB per token,
**1.00 GiB** at 8k context. At $g = 1$ (MQA): 512 B, 16 KiB, **128 MiB**.

Those 4 GiB are not idle storage. Generating one token requires reading the
entire cache from HBM - all 4.00 GiB of it per token at 8k context under MHA
(128 MiB of that per layer), against 1.00 GiB under GQA $g{=}8$. Decode is
memory-bandwidth-bound, so a 4x cut in bytes moved
is close to a 4x cut in the attention part of per-token latency, and it is what
lets you hold 4x the concurrent sequences in the same VRAM. That is the entire
argument for GQA.

Why $g = 8$ rather than $g = 1$: MQA measurably degrades quality, because all
32 query heads are forced to search over one shared key geometry, and the
subspace a head wants to *match on* is not always the subspace another head
wants. GQA with 8 groups recovers essentially all of MHA's quality at a quarter
of the cache. That empirical sweet spot is why $g \in \{4, 8\}$ appears in
nearly every recent open-weights model.

```beat
id: u4-b3
type: completion
concept: c-mha
prompt: |
  Fill the blanks. A model has $d = 8192$, $h = 64$ query heads, $d_k = 128$,
  $L = 80$ layers, fp16 weights and cache, GQA with $g = 8$.

  1. Bytes cached per token per layer $= 2 \times$ ____ $\times\, 128 \times 2
     =$ ____ B
  2. Bytes per token across all 80 layers $=$ ____ B
  3. For a 32,768-token context, one sequence's KV cache $=$ ____ GiB
  4. Under MHA ($g = 64$) the same context would need ____ GiB
  5. The ratio in step 4 over step 3 is 8. The number of multiply-accumulates
     spent forming attention scores for a new token changes by a factor of
     ____ , because ____ .

  Which set of fillings is right?
options:
  - text: "(1) $2 \\times 8 \\times 128 \\times 2 = 4096$ B; (2) $327{,}680$ B; (3) $10$ GiB; (4) $80$ GiB; (5) a factor of $1$ - unchanged - because all 64 query heads still form scores against all $n$ positions at width $d_k$, and shared K/V vectors are broadcast to the query heads sharing them rather than skipped."
    correct: true
    explain: "Correct. One $K$ and one $V$ per KV head gives $2g$ cached vectors per token per layer, so $g = 8$ yields 4 KiB, 320 KiB across 80 layers, 10 GiB at 32k context against MHA's 80 GiB. GQA cuts bytes stored and streamed by $h/g$; the arithmetic of score formation, $h \\cdot n \\cdot d_k$ multiply-accumulates per new token, is identical under every variant."
  - text: "(1) $2 \\times 64 \\times 128 \\times 2 = 32{,}768$ B; (2) $2{,}621{,}440$ B; (3) $80$ GiB; (4) $80$ GiB; (5) a factor of $1$ - unchanged - because GQA reduces query heads to 8 while leaving the 64 K/V heads in the cache."
    misconception: U4M2
    explain: "This puts $h$ where $g$ belongs. GQA leaves the number of query heads at $h = 64$ and reduces the key/value heads to $g = 8$; $W^Q$ and $W^O$ stay $8192 \\times 8192$ while $W^K$ and $W^V$ shrink to $8192 \\times 1024$. Step 1 must use $g = 8$, giving 4096 B and a 10 GiB cache."
  - text: "(1) $2 \\times 8 \\times 128 \\times 2 = 4096$ B; (2) $327{,}680$ B; (3) $10$ GiB; (4) $80$ GiB; (5) a factor of $1/8$ - eight times fewer - because with only 8 key heads there are 8 times fewer query-key dot products to compute."
    misconception: U4M3
    explain: "Steps 1-4 are right and step 5 is the whole point of the item. Every one of the 64 query heads still forms its own scores against every allowed position: $h \\cdot n \\cdot d_k$ multiply-accumulates, unchanged. A shared key is read by the 8 query heads in its group, not read once for all of them. That is why prefill, which is compute-bound, does not speed up under GQA while decode, which is bandwidth-bound, does."
  - text: "(1) $2 \\times 8 \\times 128 \\times 2 = 4096$ B; (2) $327{,}680$ B; (3) $10$ GiB; (4) $80$ GiB; (5) a factor of $8$ - eight times more - because the broadcast of each shared $K$ and $V$ out to its group of query heads is extra work that MHA does not do."
    misconception: U4M3
    explain: "The broadcast is a `repeat_interleave` view, not arithmetic: it re-reads a key that is already in registers or cache rather than recomputing anything. Score formation costs $h \\cdot n \\cdot d_k$ multiply-accumulates per new token under MHA, GQA and MQA alike, so the factor is 1."
check: choice
```

## The MLP block: where the parameters actually live

<!-- refutes: U4M5 -->

You probably think of the feed-forward layer as plumbing - a nonlinearity
stuck between attention layers to keep the stack from collapsing into one
linear map, with attention doing the real work.

Here is the prediction that fails. If the MLP were minor plumbing, it would
hold a minor share of the weights. Count a GPT-2-small block: attention
2,359,296 parameters, MLP 4,718,592 parameters. The plumbing is **two thirds**
of the block, and in a modern SwiGLU model with $d_{ff} = 3.5d$ it is over
**80%**. Roughly two thirds to four fifths of every non-embedding parameter in
every transformer you have used is sitting in MLP blocks.

The framing is appealing because attention is the interesting part and the one
every explainer draws. But by weight, a transformer is mostly MLP with
attention sprinkled between.

Here is what is actually true. The MLP is applied to each token vector
$u \in \mathbb{R}^{d}$ independently:

$$ \mathrm{MLP}(u) = \sigma\!\left(u W_1\right) W_2, \qquad W_1 \in \mathbb{R}^{d \times d_{ff}}, \;\; W_2 \in \mathbb{R}^{d_{ff} \times d} $$

where $d_{ff}$ is the hidden (intermediate) width, conventionally $4d$, and
$\sigma$ is an element-wise nonlinearity - GELU in the GPT family,
$\mathrm{GELU}(z) = z \cdot \Phi(z)$ where $\Phi$ is the standard normal
CDF, so it behaves like ReLU but is smooth near zero. Three steps: project up
to $d_{ff}$, apply $\sigma$ element-wise, project back down to $d$.

<!-- fade: mlp-param-count -->

**Procedure: MLP parameter count.** Given $d$ and $d_{ff}$ (biases ignored):

1. $W_1$: $d \times d_{ff}$. For $d = 768$, $d_{ff} = 3072$:
   $768 \times 3072 = 2{,}359{,}296$.
2. $W_2$: $d_{ff} \times d = 3072 \times 768 = 2{,}359{,}296$.
3. Total $= 2 \, d \, d_{ff} = 4{,}718{,}592$. With the $d_{ff} = 4d$
   convention this is $8d^2$.
4. Block total $= 4d^2$ (attention) $+ 8d^2$ (MLP) $= 12d^2 = 7{,}077{,}888$,
   of which the MLP is $8/12 = 66.7\%$.

Sanity-check against a model you know. GPT-2 small: $d = 768$, $L = 12$,
$d_{ff} = 3072$, vocabulary 50,257, learned positions 1024. Including biases
and the LayerNorm gains, a block is 7,087,872 parameters; twelve of them are
85,054,464; token embeddings are $50{,}257 \times 768 = 38{,}597{,}376$;
position embeddings $1024 \times 768 = 786{,}432$; the final norm 1,536. Total:
**124,439,808**. That is the "124M" on the model card, and 66.6% of the
85,056,000 non-embedding parameters in it are MLP weight.

```beat
id: u4-b4
type: completion
concept: c-mlp-block
prompt: |
  Fill the blanks. A block has $d = 4096$ and $d_{ff} = 16384$, no biases,
  MHA with $h = 32$.

  1. $W_1$ is $4096 \times 16384 =$ ____ parameters
  2. $W_2$ is $16384 \times 4096 =$ same, so MLP total $=$ ____
  3. Attention total (four $d \times d$ matrices) $=$ ____
  4. Block total $=$ ____ , of which the MLP share is ____ %
  5. Doubling $h$ from 32 to 64 changes the MLP share to ____ , because
     ____ .

  Which set of fillings is right?
options:
  - text: "(1) $67{,}108{,}864$; (2) $134{,}217{,}728$, i.e. $8d^2$; (3) $67{,}108{,}864$, i.e. $4d^2$; (4) $201{,}326{,}592 = 12d^2$, MLP share $8/12 = 66.7\\%$; (5) unchanged at $66.7\\%$, because head count does not change the attention parameter count - $W^Q, W^K, W^V, W^O$ stay $d \\times d$ and are merely sliced differently."
    correct: true
    explain: "Correct. $2 d \\, d_{ff} = 8d^2$ against attention's $4d^2$ makes the block two thirds MLP, and doubling $h$ moves only $d_k$ (from 128 to 64), leaving both totals and the share where they were."
  - text: "(1) $67{,}108{,}864$; (2) $134{,}217{,}728$; (3) $67{,}108{,}864$; (4) $201{,}326{,}592$, MLP share $66.7\\%$; (5) $50\\%$, because at $h = 64$ the attention sublayer holds $8d^2$ and matches the MLP."
    misconception: U4M1
    explain: "Steps 1-4 are right; step 5 lets heads multiply the budget. Attention is $4d^2$ at every head count: 64 heads slice the same four $4096 \\times 4096$ matrices into blocks of width 64 instead of 128. The share stays $8/12 = 66.7\\%$."
  - text: "(1) $67{,}108{,}864$; (2) $67{,}108{,}864$, since $W_1$ and $W_2$ are transposes of one shape; (3) $67{,}108{,}864$; (4) $134{,}217{,}728$, MLP share $50\\%$; (5) unchanged at $50\\%$, because head count does not change the attention parameter count."
    misconception: U4M5
    explain: "$W_1$ and $W_2$ have the same shape but are separate matrices, so they add: $2 d \\, d_{ff} = 134{,}217{,}728 = 8d^2$. Counting them once puts the MLP level with attention and hides the fact that two thirds of this block - and four fifths of a SwiGLU block at $d_{ff} = 3.5d$ - is feed-forward weight."
  - text: "(1) $67{,}108{,}864$; (2) $134{,}217{,}728$; (3) $268{,}435{,}456$, since each of the 32 heads contributes its own share of $W^Q, W^K, W^V, W^O$; (4) $402{,}653{,}184$, MLP share $33.3\\%$; (5) $20\\%$ at $h = 64$, because attention doubles again."
    misconception: U4M1
    explain: "Attention is four $d \\times d$ matrices in total, $4d^2 = 67{,}108{,}864$, as the step-3 hint says - the heads partition those matrices rather than each contributing a set. The block is $12d^2$ and two thirds MLP, and nothing in that moves with $h$."
check: choice
```

Two production details worth having.

**SwiGLU.** Llama-family models replace the two-matrix MLP with three matrices
and a gate:

$$ \mathrm{MLP}(u) = \Big(\mathrm{SiLU}(u W_{\text{gate}}) \odot (u W_{\text{up}})\Big) W_{\text{down}} $$

where $\odot$ is element-wise product and $\mathrm{SiLU}(z) = z \cdot
\mathrm{sigmoid}(z)$. One branch computes a value, the other computes a
multiplicative gate on it. Since this is three $d \times d_{ff}$ matrices
rather than two, matching the $8d^2$ budget requires $d_{ff} = \tfrac{8}{3}d$ -
which is where Llama-2-7B's odd-looking 11008 ($= 2.6875 \times 4096$) comes
from. Llama-3-8B spends more, using $d_{ff} = 3.5d = 14336$.

**A whole 8B model, from the two procedures.** $d = 4096$, $h = 32$, $g = 8$,
$d_k = 128$, $d_{ff} = 14336$, $L = 32$, vocabulary 128,256, untied embeddings.
Attention per block $2d^2 + 2d^2(g/h) = 41{,}943{,}040$. MLP per block
$3 \cdot 4096 \cdot 14336 = 176{,}160{,}768$ - 80.8% of the block. Block
218,103,808; times 32 layers, 6,979,321,856; plus two embedding matrices of
$128{,}256 \times 4096 = 525{,}336{,}576$ each. Total **8,029,995,008** = the
8.03B on the card.

## Key-value memory, and where that framing breaks

<!-- refutes: M4 -->

The MLP has a genuinely useful reading as an associative memory, and it is the
reading most likely to harden into a wrong model, so both halves matter.

Write the MLP as a sum instead of a matrix product. Let $k_i \in \mathbb{R}^d$
be the $i$-th **column** of $W_1$ and $v_i \in \mathbb{R}^d$ the $i$-th **row**
of $W_2$, for $i = 1 \dots d_{ff}$. Then, exactly:

$$ \mathrm{MLP}(u) = \sum_{i=1}^{d_{ff}} \sigma\!\left(u \cdot k_i\right)\, v_i $$

Read it as a lookup. $u \cdot k_i$ is a dot product - a match score between the
current token's residual vector and pattern $k_i$. $\sigma$ turns that score
into a nonnegative coefficient, near zero for a bad match. The result is a
coefficient-weighted sum of $d_{ff}$ fixed vectors $v_i$. So: $d_{ff}$ keys,
$d_{ff}$ values, soft matching, and the retrieved values are added to the
residual stream. Probing work does find neurons whose $k_i$ fires on
human-legible patterns (a period followed by a capital letter; "the capital
of ___") with $v_i$ pushing towards a corresponding output.

Now the failure. You probably think this makes the MLP a compressed key-value
store of facts - $d_{ff}$ rows per layer, and a bigger model has more rows.

Here is the prediction that fails, twice.

*First*: a lookup table returns nothing for an absent key. If facts were rows,
asking for an unstored fact would produce a miss - an empty result, a
refusal, a shrug. Instead the model produces a fluent, specific, confident,
wrong answer. That is not a failed lookup. It is the *same* mechanism
succeeding: $u$ dots against every $k_i$, near-misses return nonzero
coefficients, and you get a weighted blend of related values. Interpolation
between stored patterns is exactly what generalization is, and hallucination
is that mechanism running where no pattern was close. You cannot remove the
second without removing the first.

*Second*: rows are separable. If a fact were a row, deleting neuron $i$ would
delete one fact and leave the rest intact. Ablate single neurons and you get
diffuse, small degradation across many behaviours. The reason is superposition:
a layer represents far more features than it has dimensions, storing them as
non-orthogonal directions and tolerating the interference, so individual
neurons are *polysemantic* - one neuron participates in many unrelated
features, and one feature is spread across many neurons. There is no row.

The framing is appealing because engineers reach for the database analogy
immediately, and because the sum-over-$k_i$ algebra above genuinely is a
lookup. It is a lookup over *learned continuous directions in an overloaded
space*, not over rows.

Here is what is actually true, stated so it survives: parameters define a
function. The MLP's $\{k_i, v_i\}$ are a learned basis for a smooth map from
"what is in the stream" to "what to add to the stream". Where the training
distribution put many near-identical examples, that map has a sharp, localised
bump you can call a stored fact. Elsewhere it interpolates. Both are the same
map.

```beat
id: u4-b5
type: self-explain
concept: c-mlp-block
prompt: |
  The equation $\mathrm{MLP}(u) = \sum_i \sigma(u \cdot k_i) v_i$ really does
  look like a key-value lookup. Which explanation correctly says what a strict
  lookup-table reading predicts when the model is asked for a fact it was
  never trained on, what actually happens, and why the difference is a
  property of the mechanism rather than a bug in it?
options:
  - text: "A lookup table predicts a miss - no key matches, nothing is returned. What happens instead is a confident, fluent, specific wrong answer, because $u \\cdot k_i$ is a continuous dot product: every key returns some score, near-misses return substantial ones, and the output is a weighted blend of the values of related patterns. That blending is the same operation that answers questions never seen verbatim, so generalization and hallucination are one mechanism and neither can be removed alone. There is also no row to miss - superposition stores features as overlapping non-orthogonal directions across polysemantic neurons."
    correct: true
    explain: "Right on all three counts. The algebra is an exact lookup, but over learned continuous directions in an overloaded space rather than over rows, so a near-miss returns a blend instead of nothing."
  - text: "A lookup table predicts a miss, and that is roughly what you get: the coefficients $\\sigma(u \\cdot k_i)$ are all near zero for an unstored fact, so the MLP writes almost nothing and the confident wrong answer comes from elsewhere in the stack. The fix is more rows - a larger $d_{ff}$, or an external retrieval store consulted before the MLP - so that the key is present next time."
    misconception: M4
    explain: "This keeps the rows. If the coefficients really collapsed for unstored facts the model would produce a shrug, not a specific wrong name with a plausible date; the observed behaviour is exactly what soft matching over near-miss keys predicts. And enlarging $d_{ff}$ does not eliminate near-misses - it is the same interpolation that produces correct answers to novel questions, so removing it would remove generalization too."
  - text: "A lookup table predicts a miss, and the model does produce a confident wrong answer instead - but the key-value reading is simply mistaken. $\\mathrm{MLP}(u) = \\sigma(u W_1) W_2$ is a matrix product with a nonlinearity, and rewriting it as $\\sum_i \\sigma(u \\cdot k_i) v_i$ is a loose analogy, so no prediction about lookups follows either way."
    misconception: M4
    explain: "The rewriting is an exact identity, not an analogy: $k_i$ is the $i$-th column of $W_1$, $v_i$ the $i$-th row of $W_2$. Rejecting the algebra means missing why the failure is diagnostic. The error in the lookup model is the word 'rows' - discrete separable entries - not the matching-and-retrieval structure, which is real."
  - text: "A lookup table predicts a miss, and what actually happens is a confident wrong answer, because a fact is stored in one neuron $i$ and an unstored fact simply picks up whichever neighbouring neuron's $v_i$ happens to fire. Ablate that neuron and its one fact disappears cleanly, which is how model editing localizes and deletes individual facts."
    misconception: M4
    explain: "The one-neuron-one-fact picture fails its own test: ablating a single neuron produces diffuse, small degradation across many unrelated behaviours, not the clean deletion of one fact. Superposition means a layer represents far more features than it has dimensions, so one neuron participates in many features and one feature is spread across many neurons."
check: choice
```

## The residual stream is a workspace, not a shortcut

<!-- refutes: M8 -->

You probably know residual connections from ResNets, where the motivation is
gradient flow: without skip connections, gradients through a deep stack
vanish, so you add $x +$ around every block and training works. That is
correct, it is the historical motivation, and taken alone it is the single
most limiting mental model in this unit.

Here is the prediction it makes that fails. If residuals were purely a
training-time device, then at inference the stack would be a pipeline - block
$l+1$ consumes block $l$'s output - and deleting a middle block would feed
block $l+2$ an input from the wrong stage and destroy everything downstream.
Delete block 15 of a 32-block model and run it: output quality drops by a few
percent on downstream tasks and the text stays coherent. Delete two or three
non-adjacent middle blocks: it still works, degrading smoothly. Reorder two
adjacent middle blocks: mostly fine. No pipeline behaves like that. This
observation is what layer-pruning and early-exit methods are built on, and it
is why "prune the middle third of the layers" is a viable compression recipe
at all.

Here is what is actually true. Unroll the two-line block equation over $L$
layers. Because both sublayers add, the stream at the output of layer $L$ is
literally:

$$ x_L \;=\; \underbrace{e(t) + p(t)}_{\text{embedding}} \;+\; \sum_{l=1}^{L} \mathrm{MHA}_l\!\left(\mathrm{Norm}(x_{l-1})\right) \;+\; \sum_{l=1}^{L} \mathrm{MLP}_l\!\left(\cdot\right) $$

where $e(t)$ is the token embedding and $p(t)$ the positional contribution. The
final representation is the embedding **plus a sum of $2L$ additive updates**.
Nothing overwrites anything. The residual stream is a shared bus of width $d$
that persists from embedding to unembedding, and each sublayer is a peripheral
with a read port and a write port:

- **Reading** is $W^Q, W^K, W^V$ (attention) and $W_1$ (MLP). Each is a
  projection $\mathbb{R}^d \to$ something narrower, so each sublayer reads a
  *subspace* and is blind to the rest.
- **Writing** is $W^O$ (per head, rank $\le d_k$, from the identity in section
  2) and $W_2$. Each writes into a subspace and adds.

That is why deleting a middle block degrades gracefully: you removed 2 terms
from a sum of $2L$. Everything the earlier blocks wrote is still in the stream,
and later blocks read the stream, not the deleted block. It is also why the
first and last blocks are *not* droppable - early blocks write the features
later reads are conditioned on, and the last blocks rotate the stream into the
directions the unembedding matrix reads.

The gradient story is still true. Differentiating $x_l = x_{l-1} + F(x_{l-1})$
gives $\partial x_l / \partial x_{l-1} = I + \partial F / \partial x_{l-1}$;
that $I$ is an unattenuated path back to every earlier layer. Keep it. Rank it
second. The primary fact is that the transformer's computation is additive
refinement of a persistent shared representation.

One consequence worth carrying into unit 7 and any interpretability reading:
the stream has $d$ dimensions and $2L$ writers competing for them - 64 writers
into 4096 dimensions in an 8B model. They cannot each have an orthogonal
private allocation for every feature, so they share directions with tolerable
interference. Same superposition pressure as the MLP, now at the level of the
whole network's wiring.

```beat
id: u4-b6
type: predict
concept: c-residual
prompt: |
  A 32-layer model. You will run three surgeries at inference time and
  measure downstream task accuracy:
  (A) delete block 16
  (B) delete block 1
  (C) replace the residual additions in block 16 with plain assignment,
      i.e. $y = \mathrm{MLP}(\mathrm{Norm}(\mathrm{MHA}(\mathrm{Norm}(x))))$,
      keeping every weight

  Before reading on, commit to a ranking from least to most damaging, with
  the reason it follows from
  $x_L = e + \sum_{l} (\mathrm{MHA}_l + \mathrm{MLP}_l)$.
options:
  - text: "A, then B, then C. A removes 2 of 64 additive terms and the stream still carries the other 62, so degradation is a few percent; B is worse because block 1 writes the low-level features every later block's read projections expect, corrupting the inputs of all 31 downstream blocks; C is catastrophic because assignment discards the accumulated sum - everything the embedding and blocks 1-15 wrote is thrown away, and blocks 17-32 receive a vector with no history, in the wrong scale and the wrong subspaces."
    correct: true
    explain: "Right, and the A-versus-C gap is the point: what carries the state is the running sum, not the weights. C keeps every parameter and still destroys the model, while A deletes parameters and barely dents it."
  - text: "C, then A, then B. C is the mildest because every weight is still present and the block still computes its usual function - only the bookkeeping changed; A is worse because block 16's output is missing from the pipeline entirely; B is worst because the first block's error propagates through the most subsequent layers."
    misconception: M8
    explain: "This treats the residual addition as bookkeeping around a pipeline. It is the opposite: $x_L$ is the embedding plus a sum of $2L$ updates, so the additions are where the state lives. Assignment at block 16 discards $e$ and every term from blocks 1-15 at once, which is why C is the catastrophic surgery even with all weights intact."
  - text: "C, then B, then A - all three are severe. Deleting block 16 hands block 17 an input from the wrong stage of the pipeline, so everything downstream is misaligned and the output is incoherent; the deeper the deleted block, the more computation is built on the bad input."
    misconception: M8
    explain: "That is the pipeline prediction, and it is what layer-pruning experiments falsify: delete a middle block of a 32-block model and quality drops a few percent with the text still coherent; delete two or three non-adjacent middle blocks and it degrades smoothly. Block 17 reads the stream, not block 16, and the stream still holds the other 62 terms."
  - text: "A, then C, then B. A is mild for the additive-sum reason and C is survivable because a single block's assignment merely renormalizes the stream - later blocks re-derive what they need from the tokens still visible to attention; B is worst because block 1 is the only layer with direct access to the raw embeddings."
    misconception: M8
    explain: "Attention at block 17 reads the residual stream at each position, not the raw tokens, so there is nothing to re-derive from: the token identities were themselves written into the stream by the embedding, and assignment at block 16 erased them. Deleting block 1 is bad because later reads lose the features it writes, but it removes 2 terms from the sum; C removes all of them."
check: choice
```

## Normalization conditions the optimization; it does not save your floats

<!-- refutes: M7 -->

You probably think LayerNorm is there for numerical stability - keeping
activations in range so nothing overflows or NaNs, the way you would clamp or
rescale in any numerical pipeline.

Here is the prediction that fails, and it is cheap to run. If normalization
were about float range, then at inference in fp32 - where range is a
non-problem, $\pm 3.4 \times 10^{38}$ - you could drop the norm layers and get
the same outputs, maybe with a rounding difference in the last decimal. Drop
them: the model emits garbage immediately, in fp32, in fp64, at any precision.
Nothing overflowed. The outputs are wrong because every downstream weight
matrix was trained against normalized inputs and is now receiving vectors
whose scale grows with depth.

The model is appealing because engineers meet "normalize" first in numerics,
where it does mean overflow avoidance, and because transformers do have real
precision problems elsewhere (attention logits in fp16, softmax max-subtraction
- that one *is* numerics).

Here is what is actually true. Normalization is a *training-time conditioning*
device whose effect persists as a *scale contract* at inference.

LayerNorm operates on each token vector independently, across its $d$ features
(never across the batch, never across positions). For $u \in \mathbb{R}^d$:

$$ \mu = \frac{1}{d}\sum_{j=1}^{d} u_j, \qquad \sigma^2 = \frac{1}{d}\sum_{j=1}^{d}(u_j - \mu)^2 $$

$$ \mathrm{LN}(u) = \gamma \odot \frac{u - \mu}{\sqrt{\sigma^2 + \epsilon}} + \beta $$

with learned $\gamma, \beta \in \mathbb{R}^d$ (gain and bias, $2d$ parameters -
negligible) and $\epsilon \approx 10^{-5}$ guarding division by zero. RMSNorm
drops the mean subtraction and the bias:

$$ \mathrm{RMS}(u) = \gamma \odot \frac{u}{\sqrt{\frac{1}{d}\sum_j u_j^2 + \epsilon}} $$

RMSNorm is what Llama, Mistral, Gemma and most current models use: it costs one
pass instead of two and works as well, which is itself evidence that centering
was never the point.

Three things normalization actually does:

1. **Scale invariance.** $\mathrm{LN}(a u) = \mathrm{LN}(u)$ for any $a > 0$
   (exactly - the $a$ cancels between numerator and $\sigma$). So the magnitude
   of whatever the previous layer produced is discarded; only its *direction*
   survives. Multiply a preceding weight matrix by 1000 and the post-norm
   activations are bit-comparable. A device whose defining property is
   "magnitude is irrelevant" is not a magnitude-management device.
2. **A fixed operating point at every depth.** The residual stream grows as
   blocks add to it - roughly like $\sqrt{L}$ if updates were random directions
   of comparable size, faster in practice. Without normalization, block 30
   would see inputs an order of magnitude larger than block 2 saw, and its
   weights would have to be calibrated to that. Normalizing each block's *read*
   means every block trains against unit-scale input regardless of where it
   sits, which is what makes a stack of identical blocks trainable.
3. **Better-conditioned gradients.** The normalization's own Jacobian projects
   out the gradient component along the current activation direction and
   scales by $1/\sigma$, which decouples "change the direction of this
   representation" from "change its length". That is the conditioning claim,
   and it is why removing normalization requires compensating tricks
   (careful residual-branch init, gradient clipping, long warmup) to train at
   all - and why models trained *with* it cannot run *without* it.

```beat
id: u4-b7
type: compute
concept: c-layernorm
prompt: |
  Compute RMSNorm on the vector $u = [3, -4, 0, 0]$ with $\gamma = [1,1,1,1]$
  and $\epsilon = 0$. Recall
  $\mathrm{RMS}(u)_j = u_j \big/ \sqrt{\tfrac{1}{d}\sum_k u_k^2}$ with
  $d = 4$.
  Report the second component of the output.
answer: -1.6
rubric: |
  Mean square = (9 + 16 + 0 + 0)/4 = 6.25; RMS = 2.5; output =
  [1.2, -1.6, 0, 0]. Answer -1.6. Dividing by the L2 norm 5 instead of the
  RMS 2.5 gives -0.8 - a missing 1/d. Subtracting the mean first (-0.25) is
  LayerNorm, giving about -1.51, which indicates the RMSNorm/LayerNorm
  distinction is not held.
check: numeric(0.01)
```

For contrast on the same vector: LayerNorm subtracts $\mu = -0.25$ first,
giving $\sigma \approx 2.487$ and output $\approx [1.307, -1.508, 0.101,
0.101]$. Close to RMSNorm's $[1.2, -1.6, 0, 0]$ but not equal - and the two
zeros becoming nonzero is the whole difference: centering makes every output
component depend on every input component.

## Pre-LN and post-LN

<!-- refutes: U4M4 -->

Two placements, one operation, and the difference decides whether a deep model
trains. This is the second sibling pair in this unit and it is confused in the
same way as MQA/GQA: people remember that there are two, and not which is
which.

**Post-LN** (Vaswani et al. 2017, the original):

$$ x_{l} = \mathrm{Norm}\big(x_{l-1} + \mathrm{Sublayer}(x_{l-1})\big) $$

**Pre-LN** (GPT-2 onward, and every current model):

$$ x_{l} = x_{l-1} + \mathrm{Sublayer}\big(\mathrm{Norm}(x_{l-1})\big) $$

The discriminating question, and the one to answer correctly every time: **is
the residual stream itself normalized?**

- Post-LN: **yes**. The norm sits on the main path, after the add. The stream
  is renormalized $2L$ times on its way through the model.
- Pre-LN: **no**. The norm sits on the *branch*, on the copy handed to the
  sublayer. The main path from embedding to final layer is pure addition, with
  no operation on it at all - an unbroken identity path.

Everything else follows from that one difference.

Post-LN breaks the identity path. $\partial x_l / \partial x_{l-1}$ is no
longer $I + (\cdot)$; it is the LayerNorm Jacobian times that, at every layer,
so the clean gradient highway of section 6 is gone. Consequences: post-LN
models at depth need learning-rate warmup (thousands of steps at a ramped rate)
and are sensitive to initialization; past roughly 12-18 layers they become hard
to train without extra measures such as DeepNorm's residual scaling. When they
do train, post-LN often reaches slightly better final loss, because
renormalizing the stream stops later layers' contributions from being dwarfed.

Pre-LN keeps the identity path and trains stably, without warmup, at 100+
layers. Its cost is the mirror image: the stream is never rescaled, so its norm
grows monotonically with depth, and since each block's *write* is unnormalized
while its *read* is normalized, a late block's fixed-size contribution is
proportionally smaller against a large accumulated stream. Late layers do
relatively less. This is the known trade, and it is why hybrids exist -
sandwich normalization (norm before *and* after each sublayer, as in Gemma-2
and Grok-1) is an attempt to keep pre-LN's trainability while re-controlling
the stream's growth.

One implementation detail follows directly and is worth remembering as a check
on the definitions: a pre-LN model needs a **final normalization** before the
unembedding, because nothing else ever normalizes the stream. That is the
`ln_f` in GPT-2 and the `model.norm` in Llama. A post-LN model does not need
one; its last operation already was a norm. If you can recall which
architecture has a dangling final norm, you can reconstruct which is which.

```beat
id: u4-b8
type: predict
concept: c-layernorm
prompt: |
  You are handed a checkpoint with no architecture documentation, containing
  per-layer weights plus a single extra parameter tensor named `norm.weight`
  of shape $[d]$ applied after the last block and before the unembedding.
  The training log also shows a 2000-step linear learning-rate warmup.
  Before reading on, commit to a reading of all three: which arrangement
  this is, what the warmup tells you, and what happens if you delete that
  final tensor and run inference.
options:
  - text: "Pre-LN, because a trailing norm before the unembedding is only needed when the main path is never normalized; the warmup is weak evidence against but not decisive, since warmup is required for post-LN and merely common for pre-LN; deleting the tensor makes the model emit garbage at any precision, fp32 included."
    correct: true
    explain: "Right on all three. Pre-LN is $x_l = x_{l-1} + \\mathrm{Sublayer}(\\mathrm{Norm}(x_{l-1}))$, so the stream is pure addition from embedding to the end and something must normalize it before the unembedding - that is GPT-2's `ln_f` and Llama's `model.norm`. Warmup helps Adam's second moments settle in any architecture, so it cannot outweigh the trailing norm. And the breakage is a learned scale contract: the unembedding meets a vector whose norm is far outside anything it trained against, which no amount of float range repairs."
  - text: "Post-LN, because the norm sits on the main path after the residual addition, and this tensor is the last such norm in the stack; the 2000-step warmup confirms it, since post-LN at depth cannot be trained without warmup; deleting the tensor mainly costs the last block its normalization."
    misconception: U4M4
    explain: "This has the two placements swapped. In post-LN, $x_l = \\mathrm{Norm}(x_{l-1} + \\mathrm{Sublayer}(x_{l-1}))$ - the norm is already inside every block, so a post-LN checkpoint has no dangling tensor after the last block; its final operation was a norm. A separately named tensor applied after all blocks is precisely the signature of the arrangement that never touches the main path. Warmup is a symptom shared by both and settles nothing."
  - text: "Pre-LN, and deleting the final norm is only a problem in low precision: the accumulated stream has a large magnitude, so in bf16 the logits overflow, but rerunning the same deletion in fp32 restores essentially correct outputs."
    misconception: M7
    explain: "The architecture call is right and the reason for the damage is wrong. Nothing overflows: fp32 reaches $\\pm 3.4 \\times 10^{38}$ and the stream is nowhere near it. The unembedding was trained against normalized inputs and is now receiving vectors whose scale grows with depth, so the logits are badly scaled and the distribution degenerates - in fp32, in fp64, at any precision. Normalization is a conditioning and scale-contract device, not float-range management."
  - text: "Undetermined from the tensor alone, since both arrangements ship a final norm; the 2000-step warmup is the decisive evidence and settles it as post-LN."
    misconception: U4M4
    explain: "The premise is false, and it inverts which signal carries information. Post-LN models do not ship a trailing norm - the last thing the last block did was normalize. Pre-LN models always ship one. The tensor is the strong signal; warmup is the weak one, since it is standard practice for pre-LN training too."
check: choice
```

## Depth: composition through the stream

<!-- refutes: U4M6 -->

You probably picture depth as refinement: each layer takes the previous
layer's representation and cleans it up, so a deeper stack is the same
computation done more carefully.

Here is the prediction that fails. Under refinement, a 1-layer attention model
with enough width should be able to do anything a 2-layer model can, only
worse. It cannot. There is a specific, mechanical capability that appears
exactly at two attention layers and is unreachable at one: in-context pattern
completion. Show the model `... [A][B] ... [A]` and it predicts `[B]`, for
arbitrary token pairs it never saw during training - copying a name's spelling,
continuing a made-up format, following a few-shot pattern. This is what
induction heads do, and a one-layer attention-only transformer does not develop
them, at any width.

The mechanism, and the reason it needs exactly two layers wired through the
residual stream:

1. In layer 1, a **previous-token head** at each position $i$ attends to
   position $i-1$ and writes "the token before me was $x_{i-1}$" into a
   subspace of position $i$'s residual stream. One attention layer suffices;
   this is a fixed positional pattern.
2. In layer 2, an **induction head** at the final position (currently seeing
   `[A]`) forms its query from `[A]`. Its key projection reads the subspace
   layer 1 wrote into. So the position whose key says "the token before me was
   `[A]`" scores highest - that is the position holding `[B]`, immediately
   after the earlier `[A]`. The head attends there and its OV path copies
   `[B]` into the stream, raising `[B]`'s logit.

Layer 2's *keys are a function of what layer 1 wrote*. That is composition -
K-composition specifically - and it is impossible in one layer, because in one
layer every head's keys are functions of the raw embeddings only. Depth is not
a better answer to the same question; depth is the number of stages a feature
can be a function of other features.

The stream is what makes it work. Layer 1 does not hand its output to layer 2;
it writes into a shared channel, and layer 2's read projection was trained to
look in that part of the channel. Any later layer can read it. Any later layer
can also add to it. That is why a 32-block model is not 32 refinement passes
but a shallow-but-wide circuit graph with up to 32 stages of dependency.

The empirical shape of what lives where, consistently across models:

- **Early layers**: token identity, positional patterns, detokenization
  (assembling multi-token words into one feature), local syntax.
- **Middle layers**: the abstract, transferable features - entities, relations,
  task identity - and most factual association, which probing localizes to
  mid-stack MLPs. This is also the region that ablates most gracefully.
- **Late layers**: rotation of features into the directions the unembedding
  matrix reads, i.e. converting "what is true here" into "which token comes
  next". Ablating these is severe.

Depth and width are not interchangeable. Width buys more features *per stage*;
depth buys more *stages*. A computation that needs $k$ sequential dependencies
cannot be done in fewer than $k$ layers at any width, which is the same
argument as circuit depth in hardware. Scaling practice reflects this: at fixed
parameter budget there is an optimal aspect ratio, and both very deep-thin and
very shallow-wide models underperform it.

```beat
id: u4-b9
type: self-explain
concept: c-depth
prompt: |
  A two-layer attention-only transformer completes the pattern
  `[A][B] ... [A] -> [B]` for token pairs it never saw in training. A
  one-layer attention-only transformer of any width cannot. Which
  explanation of that gap is correct? Judge each on what it says the
  residual stream is doing.
options:
  - text: "In layer 1 a previous-token head at each position attends one position back and writes \"my predecessor was $x_{i-1}$\" as an additive update into a subspace of the residual stream. In layer 2 an induction head at the current `[A]` forms its keys by reading that subspace, so the highest-scoring position is the one whose predecessor was `[A]` - the position holding `[B]` - and its OV path copies `[B]` into the stream. Layer 2's keys are a function of what layer 1 wrote, and a one-layer model's keys can only be functions of the raw embeddings, so no width substitutes."
    correct: true
    explain: "That is the mechanism, and it names the right bottleneck. This is K-composition: the second stage's matching criterion depends on the first stage's write. The residual stream is what carries it - layer 1 does not hand its output to layer 2, it adds into a shared $d$-wide bus that layer 2's read projection was trained to look at. Depth is the number of times a feature can be a function of another feature, which is why the boundary here is sharp rather than gradual."
  - text: "The second layer receives a cleaner, better-formed representation than the first layer had, so it can resolve the match that the first layer got approximately right. A one-layer model has only one pass and so only a rough version of the same computation; a sufficiently wide one-layer model closes most of the gap."
    misconception: U4M6
    explain: "This is the refinement picture, and it predicts a gradual gap that width can narrow. The gap is not gradual: a one-layer attention-only model does not develop induction heads at any width. The reason is structural rather than about quality - in one layer every head's keys are functions of the raw embeddings only, so no head can pose the query \"which position follows an earlier `[A]`\" at all. It is a missing composition stage, not a coarse version of one."
  - text: "The two-layer model has seen enough text that pairs like `[A][B]` are stored in its weights and retrieved when `[A]` reappears; the extra layer supplies the capacity for that store, which a one-layer model lacks."
    misconception: M4
    explain: "Storage cannot be the account, because the completion works for token pairs that appear nowhere in training - freshly made-up formats, novel names, arbitrary symbol pairs. Parameters define a function over the token stream rather than holding rows to retrieve. What the second layer supplies is a stage of composition: keys that depend on what an earlier stage wrote into the stream."
  - text: "Two attention layers give twice as many heads, so more simultaneous attention patterns are available per position, and one of them can cover the copy pattern. A one-layer model of double width has the same head budget and can do it too."
    misconception: U4M1
    explain: "Head count is the wrong axis. Heads within a layer all read the same stream state and form their keys from it in parallel, so adding heads buys more patterns at one stage, never a second stage. The induction circuit needs layer 2's keys to be computed from something layer 1 wrote - a sequential dependency that no number of parallel heads at a single stage provides."
check: choice
```

## What to carry forward

The block is: read the stream through projections, compute, add back. Attention
is the only cross-position operation and holds $4d^2$ parameters regardless of
head count; heads partition rather than replicate, and cutting KV heads (GQA)
cuts cache bytes, not FLOPs. The MLP holds two thirds to four fifths of the
parameters and behaves as a soft associative memory over learned directions -
which is why it generalizes and why it hallucinates, with one mechanism.
Everything writes additively to a persistent $d$-wide stream, which is why
middle blocks are droppable and why normalization has to sit on the branch
(pre-LN) rather than the path. Depth buys composition stages, not polish.

Unit 5 trains this object; unit 6 runs it, where the KV-cache arithmetic from
section 3 becomes the dominant cost; unit 7 replaces the MLP of section 4 with
a routed set of them.
