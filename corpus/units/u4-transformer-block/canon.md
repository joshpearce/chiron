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
  reading on, answer both:
  (a) How many parameters does B have in its attention sublayer, relative
  to A?
  (b) If the parameter counts are what you said, what does B actually gain
  over A? Name the resource that heads spend, given that it is not
  parameters.
answer: |
  (a) Identical. Both are four $512 \times 512$ matrices ($W^Q, W^K, W^V,
  W^O$) = $4 \times 262{,}144 = 1{,}048{,}576$ parameters. Heads slice those
  matrices; they do not add matrices.
  (b) B gains 8 independent attention distributions per query position
  instead of 1. Configuration A computes exactly one softmax row per query,
  so every one of the 512 output dimensions is mixed across positions with
  the *same* weights. B can copy from position 3 into one 64-dim subspace
  while copying from position 17 into another, in the same layer. What heads
  spend is per-head rank/width ($d_k$ drops from 512 to 64), not parameters.
rubric: |
  (a) Must say the counts are equal (or "4 d^2 either way"). Saying B has
  8x more is U4M1 - the target error - and fails the item outright.
  (b) Must identify that multiple heads buy multiple simultaneous attention
  patterns / mixing distributions per position. Credit also for "each head
  reads and writes a different subspace". Answers that only say "heads
  specialise in different linguistic features" get partial credit - that is
  an observed consequence, not the mechanism. Mentioning the $d_k$ / rank
  cost is a full-credit bonus, not required.
check: llm
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
answer: |
  1. $d_k = 64$
  4. $768 \times 768$
  5. $2{,}359{,}296$
  6. $2{,}359{,}296$ - unchanged - because the head count only sets how the
     four fixed $d \times d$ matrices are sliced. Changing $h$ changes $d_k$
     (768/4 = 192) and therefore the rank of each head, not the parameter
     count.
# Variant blanks: blank steps 2+3 instead of 1+4 to drill matrix shapes;
# blank only step 6 for a fast recall pass; blank 5+6 for the mastery pass.
rubric: |
  Blank 1 must be 64. Blank 4 must be 768 x 768. Blank 5 must be 2,359,296
  (accept "4 d^2" or "~2.36M"). Blank 6 must state the total is UNCHANGED
  and attribute this to slicing fixed d x d matrices. Any answer that scales
  the total with h is U4M1 and fails regardless of the other blanks.
check: llm
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
answer: |
  1. $2 \times 8 \times 128 \times 2 = 4096$ B
  2. $4096 \times 80 = 327{,}680$ B (320 KiB) per token
  3. $327{,}680 \times 32{,}768 = 10{,}737{,}418{,}240$ B $= 10$ GiB
  4. $80$ GiB
  5. Factor of 1 - unchanged - because all 64 query heads still form scores
     against all $n$ positions at width $d_k$; shared K/V vectors are
     broadcast to the query heads that share them, not skipped. GQA changes
     bytes stored and moved, not arithmetic done.
# Variant blanks: blank steps 2+3 only for a quick recomputation drill;
# blank 1+5 to isolate the g-vs-h distinction; blank 4+5 for mastery.
rubric: |
  Blank 1 must use g = 8 (not h = 64) and yield 4096 B. Blank 2: 327,680 B.
  Blank 3: 10 GiB. Blank 4: 80 GiB. Blank 5 MUST say the factor is 1 /
  unchanged AND give the broadcast reason. Using h in step 1 is U4M2
  (conflating query heads with KV heads) - fail. Saying step 5 is a factor
  of 8 or 1/8 is U4M3 (believing GQA cuts FLOPs) - fail even if 1-4 are
  right, since that is the concept under test.
check: llm
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
answer: |
  1. $67{,}108{,}864$
  2. $134{,}217{,}728$ (this is $8d^2$)
  3. $67{,}108{,}864$ (this is $4d^2$)
  4. $201{,}326{,}592$ ($12d^2$); MLP share $= 8/12 = 66.7\%$
  5. Unchanged at 66.7%, because head count does not change the attention
     parameter count - $W^Q, W^K, W^V, W^O$ stay $d \times d$ and are merely
     sliced differently.
# Variant blanks: blank 1+4 for the arithmetic drill; blank 3+5 to force the
# head-count-independence link back to section 2; blank 2+4+5 for mastery.
rubric: |
  1: 67,108,864. 2: 134,217,728. 3: 67,108,864. 4: 201,326,592 and 66.7%
  (accept 2/3). 5 MUST say unchanged and give the slicing reason. An answer
  that shrinks the MLP share when h doubles is U4M1 carried forward. An
  answer that puts attention above the MLP in step 4 is U4M5.
check: llm
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
  In your own words, in 3-5 sentences: the equation
  $\mathrm{MLP}(u) = \sum_i \sigma(u \cdot k_i) v_i$ really does look like a
  key-value lookup. Explain what a strict lookup-table model predicts about
  a model asked for a fact it was never trained on, what actually happens,
  and why the difference is a property of the mechanism rather than a bug in
  it.
answer: |
  A lookup table predicts a miss: no key matches, so nothing is returned.
  What happens instead is a confident, fluent, specific wrong answer,
  because $u \cdot k_i$ is a continuous dot product - every key returns some
  score, near-misses return substantial ones, and the output is a weighted
  blend of the values of related patterns. That blending is the same
  operation that lets the model answer questions it never saw verbatim, so
  generalization and hallucination are one mechanism, not two. There is also
  no row to miss: superposition means features are stored as overlapping
  non-orthogonal directions across many polysemantic neurons.
rubric: |
  Must contain: (1) the lookup model predicts a miss/empty result;
  (2) what actually happens is a confident wrong answer produced by soft
  matching / weighted blending of near-miss values; (3) an explicit
  statement that this is the same mechanism as generalization, so it cannot
  be removed without removing generalization. All three = pass. Missing (3)
  = partial. An answer that treats hallucination as a lookup failure to be
  fixed with more parameters or better retrieval is M4 uncorrected - fail.
  Credit but do not require polysemanticity/superposition. An answer that
  rejects the key-value framing entirely also fails: the algebra is exact,
  the error is the "rows" reading.
check: llm
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
  measure downstream task accuracy. Rank them from least to most damaging,
  and give the reason your ranking follows from
  $x_L = e + \sum_{l} (\mathrm{MHA}_l + \mathrm{MLP}_l)$:
  (A) delete block 16
  (B) delete block 1
  (C) replace the residual additions in block 16 with plain assignment,
      i.e. $y = \mathrm{MLP}(\mathrm{Norm}(\mathrm{MHA}(\mathrm{Norm}(x))))$,
      keeping every weight
answer: |
  Least to most damaging: A, then B, then C.
  (A) removes 2 of 64 additive terms; the stream still carries the other 62,
  and later blocks read the stream rather than block 16, so degradation is a
  few percent.
  (B) is worse: block 1 writes the low-level features that every later
  block's read projections were trained to expect, so its absence corrupts
  the inputs of all 31 downstream blocks rather than removing one term.
  (C) is catastrophic: assignment discards the accumulated sum. Everything
  written by blocks 1-15 and by the embedding is thrown away at block 16,
  and blocks 17-32 receive a vector with no history, in the wrong scale and
  the wrong subspaces.
rubric: |
  Ordering must be A < B < C in damage. Reason for A must be "removes a
  small number of terms from a sum, stream is intact". Reason for C must be
  "the accumulated stream is discarded / overwritten rather than added to".
  Getting A vs C right is the M8 discrimination and is required to pass;
  swapping A and B is a partial credit miss. Predicting that A is
  catastrophic because downstream blocks get the wrong stage's input is M8
  (pipeline model) - fail. Predicting C is harmless because "the weights are
  all still there" misses that the residual sum, not the weights, carries
  the state - fail.
check: llm
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
  Predict:
  (a) Is this pre-LN or post-LN, and what is the one-sentence reason?
  (b) The training log shows a 2000-step linear learning-rate warmup. Is
  that consistent with your answer, weak evidence against it, or irrelevant?
  (c) You now delete that final norm tensor and run inference. What happens,
  and does your answer depend on running in fp32 rather than bf16?
answer: |
  (a) Pre-LN. A trailing normalization before the unembedding is only needed
  when the residual stream is never normalized on the main path, which is
  exactly the pre-LN arrangement.
  (b) Weak evidence against but not disqualifying. Warmup is *required* for
  post-LN and merely *common* for pre-LN, since it also helps Adam's
  second-moment estimates settle. The trailing norm is the stronger signal.
  (c) The unembedding receives a vector whose norm is far larger than
  anything it was trained against, so the logits are badly scaled and the
  output distribution degenerates - the model emits garbage. Precision is
  irrelevant: this is a learned scale contract, not a float-range problem,
  and fp32 or fp64 changes nothing (M7).
rubric: |
  (a) Must answer pre-LN AND give the reason "the stream is otherwise never
  normalized". Answering post-LN is U4M4 - fail. (b) Must recognize warmup
  is not decisive; either "weak evidence against" or "consistent with both"
  passes; claiming warmup proves post-LN fails. (c) Must say the model
  breaks AND that precision does not matter. Saying fp32 would rescue it is
  M7 - fail.
check: llm
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
  Explain, in 4-6 sentences and without using the word "refine": why can a
  two-layer attention-only transformer complete the pattern
  `[A][B] ... [A] -> [B]` for token pairs it never saw in training, while a
  one-layer attention-only transformer of any width cannot? Your explanation
  must say what the residual stream does in the mechanism.
answer: |
  In layer 1 a previous-token head at each position attends one position
  back and writes "my predecessor was $x_{i-1}$" as an additive update into
  a subspace of that position's residual stream. In layer 2, an induction
  head at the current `[A]` forms its keys by reading that subspace, so the
  position scoring highest is the one whose predecessor was `[A]` - the
  position holding `[B]` - and its value/output path copies `[B]` into the
  stream, raising its logit. The essential step is that layer 2's *keys are
  a function of what layer 1 wrote*, which requires two sequential attention
  stages. A one-layer model's keys can only be functions of the raw
  embeddings, so no head can ever ask "which position follows an earlier
  `[A]`", no matter how wide it is. The residual stream is the channel that
  makes this composition possible: layer 1 does not pass its output to layer
  2, it writes into a shared bus that layer 2's read projection was trained
  to look at.
rubric: |
  Must contain: (1) a two-stage mechanism where the first stage writes
  something about the previous token; (2) the second stage's QUERY/KEY
  matching depends on what the first stage wrote (composition), not merely
  that it runs later; (3) an explicit statement that a one-layer model's
  keys can only depend on raw embeddings, so width cannot substitute;
  (4) the residual stream named as the shared write/read channel.
  3 of 4 = pass, but (2) is mandatory - an answer that says the second layer
  "does more processing" or "has a cleaner input" is U4M6 and fails. An
  answer claiming the pair [A][B] must have been memorized during training
  is M4 and fails.
check: llm
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
