---
unit: u7
title: "Mixture of Experts"
concepts:
  - c-moe-routing
  - c-moe-tradeoff
assumes:
  - c-matmul
  - c-dotprod
  - c-softmax
  - c-mlp-block
  - c-residual
  - c-mha
  - c-kv-cache
  - c-quant
---

The model reading this book to you is `Qwen3.6-35B-A3B`. Thirty-five billion
parameters total. Roughly three billion of them do any work per token. That
suffix is not marketing; it is the entire architecture, and it is the reason
this thing runs on a laptop at conversational speed instead of at the pace of a
slow typist.

This unit is about why that gap exists and what it costs.

## Decode is bandwidth-bound, and that is the whole motivation

Start with the systems question, not the ML question. You have a Mac with
**153 GB/s** of memory bandwidth and a 4-bit quantization of a 35B model on
disk. How fast can it possibly generate?

```beat
id: u7-b1
type: predict
concept: c-moe-tradeoff
prompt: |
  You have a 35B-parameter model quantized to roughly 4.35 bits per weight,
  so about 19 GB of weights. Your machine sustains 153 GB/s of memory
  bandwidth. Batch size 1, one user, generating one token at a time.

  Before reading on: what is the hard upper bound on tokens per second, and
  what physical quantity sets it? Do not compute precisely - name the
  bottleneck and give an order of magnitude.
answer: |
  Roughly 8 tokens per second. The bottleneck is memory bandwidth, not
  compute: generating one token requires reading every weight from memory
  exactly once, so the ceiling is (bandwidth) / (bytes of weights) =
  153 / 19 = about 8 tok/s. No amount of GPU FLOPs helps.
rubric: |
  Must identify memory bandwidth (not FLOPs, not context length, not the KV
  cache) as the binding constraint, and must express the ceiling as
  bandwidth divided by total weight bytes. Order of magnitude "single-digit
  tok/s" is a pass. Answering "depends on the GPU" or naming compute/FLOPs
  as the limit is the classic systems mis-transfer - fail.
check: llm
```

<!-- fade: active-param-bandwidth -->

Here is the argument in full, because every claim in this unit rests on it.

**Step 1: one token requires one full read of the weights.** At batch size 1,
generating token $t+1$ means one forward pass. Every weight matrix in the
network participates in exactly one matrix-vector product. A weight that sits
in DRAM and is never loaded into the GPU's registers contributes nothing.
So the memory traffic per token is, at minimum, the size of the weights you
actually touch.

**Step 2: that read is not amortized over anything.** Let $W$ be a weight
matrix of shape $m \times n$ ($m$ output features, $n$ input features), and
$x \in \mathbb{R}^{n}$ the activation vector for the single token being
generated. Computing $Wx$ costs:

$$\text{bytes read} = m \cdot n \cdot b, \qquad \text{FLOPs} = 2 \cdot m \cdot n$$

where $b$ is bytes per weight (here $b = 4.35/8 = 0.544$, since 4.35 bits per
weight is what a Q4_K_M quantization actually averages out to) and the factor
2 is one multiply plus one add per weight. The ratio of those two numbers is
the **arithmetic intensity** - useful work per byte moved:

$$\text{intensity} = \frac{2mn}{mn \cdot b} = \frac{2}{b} = \frac{2}{0.544} \approx 3.7 \text{ FLOP/byte}$$

Note what dropped out: $m$ and $n$. The shape of the matrix does not matter.
At batch size 1, a matrix-vector product offers 3.7 FLOPs of work per byte of
traffic, always.

**Step 3: compare against the machine.** A machine's balance point is
$\text{peak FLOP/s} \div \text{peak bytes/s}$. This Mac's GPU does on the order
of 10 TFLOP/s at 153 GB/s:

$$\text{machine balance} = \frac{10 \times 10^{12}}{153 \times 10^{9}} \approx 65 \text{ FLOP/byte}$$

You are offering 3.7 and the machine wants 65. You are short by a factor of
about 18. The GPU spends roughly 94% of decode wall-clock idle, waiting on
DRAM. This is not a tuning problem; it is what a matrix-vector product is.

**Step 4: therefore the ceiling.**

$$\text{tok/s}_{\max} = \frac{\text{bandwidth}}{\text{bytes read per token}} = \frac{153 \text{ GB/s}}{19 \text{ GB}} \approx 8.1 \text{ tok/s}$$

```beat
id: u7-b2
type: compute
concept: c-moe-tradeoff
prompt: |
  Same machine, 153 GB/s. A different dense model quantized to 4 bits comes
  to 25.5 GB of weights. Batch size 1.

  Give the bandwidth-bound ceiling in tokens per second. Number only, one
  decimal place.
answer: 6.0
rubric: |
  153 / 25.5 = 6.0 tok/s. Graded numerically.
check: numeric(0.15)
```

Eight tokens per second is roughly one-third of comfortable reading speed. That
is the problem. And now look at where the leverage is: the numerator is fixed
by the hardware you own. The only term you control is **bytes read per token**.

Everything MoE does follows from taking that observation literally. If most of
the network does not need to run for most tokens, then do not read it.
Parameters you do not read are parameters you do not pay for.

## The router: replacing one MLP block with N of them

Recall the transformer block from u4: attention writes to the residual stream,
then an MLP block reads from it and writes an additive update back. That MLP is
where most of the parameters live - typically two-thirds or more of the
network.

A Mixture-of-Experts block replaces that single MLP with $N$ independent MLPs
of the same shape, called **experts**, plus a small **router** that decides
which ones run.

Concretely, for a token whose residual-stream vector is $x \in \mathbb{R}^{d}$
($d$ = model width, e.g. 2048):

1. **Route.** A single linear layer $W_r \in \mathbb{R}^{N \times d}$ produces
   one score per expert: $\ell = W_r x \in \mathbb{R}^{N}$. That is the entire
   router. It is one matrix, no nonlinearity, roughly $N \cdot d$ parameters -
   for $N = 128$ and $d = 2048$, about 262k parameters, versus hundreds of
   millions in each expert. The router is free.
2. **Select.** Take the indices of the $k$ largest entries of $\ell$
   (top-$k$, typically $k = 8$ of $N = 128$).
3. **Gate.** Softmax over *only those $k$ scores* to get weights
   $g_i$ summing to 1.
4. **Combine.** Run only the selected experts and take the gated sum:

$$y = \sum_{i \in \text{top-}k} g_i \cdot E_i(x)$$

where $E_i$ is the $i$-th expert MLP and $y \in \mathbb{R}^{d}$ is the update
written back to the residual stream. The other $N - k$ experts are never
loaded, never multiplied, never touched.

That is the whole mechanism. There is no dispatcher process, no classifier
trained on topics, no scheduling logic. It is one extra matrix and an argmax.

<!-- fade: router-topk-gate -->

**Worked example.** Tiny shapes: $d = 4$, $N = 4$ experts, $k = 2$.

Token vector and router matrix (each row of $W_r$ is one expert's scoring
vector):

$$x = \begin{bmatrix} 1 \\ 0 \\ 2 \\ -1 \end{bmatrix}, \qquad
W_r = \begin{bmatrix}
1 & 0 & 0 & 0 \\
0 & 0 & 1 & 0 \\
0 & 1 & 0 & 1 \\
0 & 0 & 0 & 0
\end{bmatrix}$$

**Step 1 - route.** Each router logit is a dot product of one row with $x$:

$$\ell_0 = 1(1) + 0(0) + 0(2) + 0(-1) = 1$$
$$\ell_1 = 0(1) + 0(0) + 1(2) + 0(-1) = 2$$
$$\ell_2 = 0(1) + 1(0) + 0(2) + 1(-1) = -1$$
$$\ell_3 = 0$$

So $\ell = [1,\; 2,\; -1,\; 0]$.

**Step 2 - select.** The two largest are $\ell_1 = 2$ and $\ell_0 = 1$.
Experts 1 and 0 run. Experts 2 and 3 do not.

**Step 3 - gate.** Softmax over the selected scores $\{2, 1\}$ only:

$$g_1 = \frac{e^{2}}{e^{2} + e^{1}} = \frac{7.389}{7.389 + 2.718} = \frac{7.389}{10.107} = 0.731$$
$$g_0 = \frac{e^{1}}{e^{2} + e^{1}} = \frac{2.718}{10.107} = 0.269$$

They sum to 1 by construction. This matters: if you softmaxed over all four
logits and kept only the top two weights, they would sum to
$0.237 + 0.644 = 0.881$, and the block's contribution to the residual stream
would be silently scaled down by a token-dependent factor. Renormalizing over
the selected set keeps the block's output magnitude comparable across tokens.

**Step 4 - combine.** Suppose the two selected experts produce

$$E_1(x) = \begin{bmatrix} 0 \\ 1 \\ 0 \\ 2 \end{bmatrix}, \qquad
E_0(x) = \begin{bmatrix} 2 \\ 0 \\ -1 \\ 0 \end{bmatrix}$$

Then

$$y = 0.731 \begin{bmatrix} 0 \\ 1 \\ 0 \\ 2 \end{bmatrix}
    + 0.269 \begin{bmatrix} 2 \\ 0 \\ -1 \\ 0 \end{bmatrix}
    = \begin{bmatrix} 0.538 \\ 0.731 \\ -0.269 \\ 1.462 \end{bmatrix}$$

and $y$ is added to the residual stream exactly as a dense MLP's output would
be. Downstream layers cannot tell which experts ran.

Two consequences worth stating explicitly, because they are where engineers'
intuitions usually break:

- **Routing is per token, per layer.** Not per sequence, not per request. A
  48-layer MoE model makes 48 independent routing decisions for every single
  token, and the token after it may take a completely different path.
- **The router is trained by ordinary backprop, jointly with everything else.**
  There is no separate training stage and no labels. $W_r$ receives gradient
  through $g_i$ in the gated sum: if expert $i$'s output reduced the loss, the
  gradient pushes $\ell_i$ up for tokens like this one. The router learns which
  experts are *useful*, and "useful" is defined only by the next-token loss.

```beat
id: u7-b3
type: completion
concept: c-moe-routing
prompt: |
  Same mechanism, new numbers. $d = 3$, $N = 4$ experts, top-$k = 2$.

      x     = [2, -1, 1]
      W_r   = row 0: [1, 0, 0]
              row 1: [0, 1, 0]
              row 2: [1, 0, 1]
              row 3: [0, 0, 0]

      Step 1 - route:   l_0 = 2,  l_1 = -1,  l_2 = ____,  l_3 = 0
      Step 2 - select:  experts ____ and ____ run
      Step 3 - gate:    softmax over the SELECTED logits only
                        g for the larger  = e^3 / (e^3 + e^2) = ____
                        g for the smaller = ____
      Step 4 - combine: E_2(x) = [1, 0, 0],  E_0(x) = [0, 2, 0]
                        y = ____

  Fill every blank. Show the arithmetic for the gates.
answer: |
  Step 1: l_2 = 1(2) + 0(-1) + 1(1) = 3. So l = [2, -1, 3, 0].
  Step 2: the two largest are l_2 = 3 and l_0 = 2, so experts 2 and 0 run.
  Step 3: e^3 = 20.086, e^2 = 7.389, sum = 27.475.
          g_2 = 20.086 / 27.475 = 0.731
          g_0 =  7.389 / 27.475 = 0.269
  Step 4: y = 0.731*[1,0,0] + 0.269*[0,2,0] = [0.731, 0.538, 0].
rubric: |
  All four blanks required. Pass needs: l_2 = 3; experts 2 and 0 selected;
  gates 0.731 / 0.269 (+/- 0.01) computed over the two selected logits ONLY;
  y = [0.731, 0.538, 0]. Softmaxing over all four logits (giving roughly
  0.696 / 0.256 and a y that sums to less than 1 of the expert mass) is the
  single most common error - mark fail and name the renormalization rule.
  Selecting experts 2 and 3, or averaging the experts unweighted, is a fail.
# variant blanking: blank Step 2 and Step 3 to drill selection+gating;
# blank Step 1 and Step 4 to drill the linear algebra at both ends.
check: llm
```

## Experts are not domain specialists

<!-- refutes: M12 -->

You probably think the experts are specialists: a code expert, a French expert,
a math expert, and the router is a classifier that reads the token, works out
the topic, and dispatches. It is a mixture *of experts*, after all, and you have
built systems exactly like this - a request comes in, a dispatcher inspects it,
it goes to the right handler.

**The specific prediction that fails.** If experts were domain specialists,
expert selection would correlate strongly with domain. Feed the model a Python
file and a French novel, and you would see two largely disjoint sets of experts
light up, consistently, across layers. Researchers have measured exactly this
on trained MoE models. It does not happen. Routing correlates far more with
*token-level and syntactic* properties - punctuation, whitespace, numerals,
common function words, particular subword shapes - than with subject matter,
and the same expert fires enthusiastically across topics that have nothing to
do with each other. In the middle layers, where you would most expect semantic
specialization, routing is closest to uniform. Two prompts about entirely
different domains routinely share most of their experts.

There is a second, sharper failure. If the router were a topic classifier, its
decisions would be stable within a document. They are not: routing changes token
to token *within one sentence*, and it changes again at every layer. A "French
expert" that gets deselected between the article and the noun is not a French
expert.

**Why the specialist story is appealing.** The name. Also, the original 1991
mixture-of-experts paper genuinely was about specialists - small networks
competing to own regions of an input space, with an explicitly interpretable
gate. The name survived; the semantics did not. And the systems analogy is
strong: sharded handlers, feature flags, plugin dispatch. Every one of those is
a design where a human chose the partition.

**What is actually true.** The partition is not designed, and nothing in
training rewards it being human-legible. Two forces shape it:

1. **The loss.** The router is optimized for one thing - pick the experts whose
   outputs lower next-token loss for this token at this layer. Nothing asks for
   the resulting grouping to be nameable.
2. **A load-balancing penalty that actively fights concentration.**

That second force is the mechanism, and it is worth seeing.

**The collapse problem.** Suppose early in training expert 3 is marginally
better than the rest. The router sends it more tokens, so it gets more gradient
updates, so it gets better still, so it gets more tokens. Within a few thousand
steps the router sends nearly everything to a handful of experts and the rest
are dead weight - loaded into memory, never selected, never trained. You have
paid for a 35B model and are running a 4B one. This is a real, reliably
observed failure mode, not a theoretical worry, and it is a positive feedback
loop.

**The fix: an auxiliary loss.** Add a term to the training objective that is
minimized when routing is uniform:

$$L_{\text{aux}} = \alpha \cdot N \sum_{i=1}^{N} f_i \cdot P_i$$

with every symbol defined here:

- $N$ = number of experts.
- $f_i$ = the *fraction of tokens in this batch* that were routed to expert $i$.
  A counting statistic, non-differentiable, $\sum_i f_i = 1$ for top-1
  accounting.
- $P_i$ = the *mean router probability* assigned to expert $i$ across the batch,
  i.e. the average of $\text{softmax}(\ell)_i$ over tokens. Differentiable -
  this is the term the gradient actually flows through. $\sum_i P_i = 1$.
- $\alpha$ = a small coefficient, typically 0.01, so the penalty nudges without
  overriding the language-modeling loss.

The product $f_i \cdot P_i$ is the trick. $f_i$ says "this expert got a lot of
traffic" and $P_i$ says "the router wants to keep sending it traffic". The
gradient can only touch $P_i$, so an expert that is both overloaded and
strongly preferred gets its router logits pushed down hard, while $f_i$ scales
the size of that push by how bad the overload actually is.

**Numeric example.** $N = 4$ experts, one batch.

*Perfectly balanced:* $f = [0.25, 0.25, 0.25, 0.25]$, $P = [0.25, 0.25, 0.25, 0.25]$.

$$L_{\text{aux}} / \alpha = 4 \cdot (4 \times 0.25 \times 0.25) = 4 \times 0.25 = 1.0$$

*Skewed:* $f = [0.5, 0.25, 0.25, 0]$, $P = [0.4, 0.3, 0.2, 0.1]$.

$$L_{\text{aux}} / \alpha = 4 \cdot (0.5 \cdot 0.4 + 0.25 \cdot 0.3 + 0.25 \cdot 0.2 + 0 \cdot 0.1)$$
$$= 4 \cdot (0.2 + 0.075 + 0.05 + 0) = 4 \times 0.325 = 1.3$$

*Collapsed:* $f = [1, 0, 0, 0]$, $P = [0.7, 0.1, 0.1, 0.1]$ gives $4 \times 0.7 = 2.8$.

Two things to notice. The minimum is **1.0, not 0** - the $N$ prefactor is
chosen so a perfectly uniform router sits at exactly 1 regardless of $N$, which
makes the coefficient $\alpha$ mean the same thing across model sizes. And the
penalty grows smoothly with concentration, so it is a pressure, not a hard
constraint.

```beat
id: u7-b4
type: compute
concept: c-moe-routing
prompt: |
  Load-balancing loss, $L_{\text{aux}} = \alpha N \sum_i f_i P_i$, with
  $N = 4$ and $\alpha = 1$ (report the bare sum).

    f = [0.4, 0.4, 0.1, 0.1]
    P = [0.5, 0.3, 0.1, 0.1]

  Give $L_{\text{aux}}$. Number only, two decimal places.
answer: 1.36
rubric: |
  4 * (0.4*0.5 + 0.4*0.3 + 0.1*0.1 + 0.1*0.1) = 4 * (0.20+0.12+0.01+0.01)
  = 4 * 0.34 = 1.36. Graded numerically. A learner who answers 0.34 dropped
  the N prefactor; 1.0 means they answered the floor rather than computing.
check: numeric(0.01)
```

```beat
id: u7-b5
type: self-explain
concept: c-moe-routing
prompt: |
  You now have two facts: (a) the router is trained only by the next-token
  loss, and (b) an auxiliary loss penalizes any routing distribution that
  concentrates on a few experts.

  In your own words, explain why these two forces together make
  *domain-specialist* experts unlikely to emerge - even though "one expert
  per domain" would be a perfectly reasonable way to partition the work.
  Then state what the experts do end up partitioning on.
answer: |
  Domain is a badly balanced variable. Training corpora are wildly
  non-uniform across domains - far more general English than, say, Lisp - so
  a domain partition would put hugely unequal token counts on different
  experts, which is precisely what the load-balancing loss penalizes. The
  aux loss wants every expert to see roughly 1/N of the tokens in every
  batch, and domains cannot supply that. Meanwhile the LM loss is indifferent
  to whether the partition is interpretable; it only rewards picking experts
  that reduce loss on this token. So the router settles on a partition that is
  (i) roughly equal-frequency and (ii) locally predictive - which turns out to
  be token-level and syntactic: subword shape, punctuation, numerals,
  whitespace, position-in-word. Specialization is real but emergent,
  token-granular, and mostly not nameable.
rubric: |
  Pass requires BOTH: (1) the balance argument - domain frequencies in the
  corpus are highly skewed, so a domain-based partition is exactly the kind
  of imbalance the aux loss punishes; (2) the LM-loss argument - nothing in
  the objective rewards interpretability, only per-token loss reduction.
  Must also land that the emergent partition is token-level/syntactic rather
  than semantic. One of the two arguments plus the correct conclusion =
  partial. Answers that still describe the router as classifying content, or
  that say experts specialize by domain "but the router is imperfect", are
  M12 surviving - fail and re-state the failing prediction.
check: llm
```

Say this precisely, because it is the takeaway: experts are parallel MLP blocks
selected per token by a learned router optimizing load-balanced loss reduction.
Specialization is emergent, token-granular, and mostly not human-interpretable.
An individual expert is not a model and cannot be run alone - it is a fraction
of one MLP block in one layer of forty-eight.

## Total versus active: the tradeoff table

Now put the two halves together. Here is the actual decomposition of the
teacher model:

| component | count | params each | total |
|---|---|---|---|
| attention + norms + router + output head | - | - | 1.2B |
| experts | 128 | 0.265B | 33.9B |
| **total** | | | **35.1B** |
| always active | - | - | 1.2B |
| active experts (top-8) | 8 | 0.265B | 2.1B |
| **active per token** | | | **3.3B** |

Active fraction: $3.3 / 35.1 = 9.4\%$. You read roughly one-tenth of the model
per token.

Convert to the only currency decode cares about. At 4.35 bits/weight
($b = 0.544$ bytes):

$$\text{resident} = 35.1 \times 10^{9} \times 0.544 = 19.1 \text{ GB}$$
$$\text{read per token} = 3.3 \times 10^{9} \times 0.544 = 1.8 \text{ GB}$$

$$\text{tok/s}_{\max} = \frac{153}{1.8} \approx 85 \text{ tok/s}$$

against 8 tok/s for a dense 35B on the same machine. A 10.6x ceiling increase
from an architectural change that costs one extra 262k-parameter matrix per
layer.

```beat
id: u7-b6
type: completion
concept: c-moe-tradeoff
prompt: |
  Work the same budget for a hypothetical `48B-A6B` MoE at 4 bits per weight
  (0.5 bytes/param) on a 400 GB/s machine.

      Step 1 - resident bytes:      48e9 * 0.5  = 24 GB
      Step 2 - bytes read/token:    6e9  * 0.5  = ____ GB
      Step 3 - MoE ceiling:         400 / ____  = ____ tok/s
      Step 4 - dense-48B ceiling:   400 / 24    = ____ tok/s
      Step 5 - speedup:             ____ x
      Step 6 - RAM you must own:    ____ GB

  Fill every blank. Step 6 is the one people get wrong.
answer: |
  Step 2: 6e9 * 0.5 = 3 GB read per token.
  Step 3: 400 / 3 = 133.3 tok/s.
  Step 4: 400 / 24 = 16.7 tok/s.
  Step 5: 133.3 / 16.7 = 8x, which is just 48/6 = 8, the total/active ratio.
  Step 6: 24 GB. All 48B of weights must be resident, because any expert may
  be selected by the next token. MoE buys bandwidth, not capacity.
rubric: |
  Pass requires: 3 GB/token; 133 tok/s; 16.7 tok/s; 8x; and critically
  Step 6 = 24 GB (full model resident). Answering 3 GB for Step 6 is the
  "MoE saves memory" error (U7-M1) - automatic fail regardless of the other
  five blanks, and the feedback must state that all experts stay resident.
  Noticing that the speedup equals total/active exactly is a strong signal.
# variant blanking: blank steps 3 and 6 to isolate the bandwidth/capacity
# distinction; blank steps 2 and 5 to drill the ratio itself.
check: llm
```

The full comparison, all at 4-bit on the same 153 GB/s machine:

| model | total params | active/token | RAM required | bytes read/token | ceiling | knowledge scale |
|---|---|---|---|---|---|---|
| dense 3B | 3B | 3B | 1.6 GB | 1.6 GB | ~93 tok/s | a 3B |
| dense 35B | 35B | 35B | 19 GB | 19 GB | ~8 tok/s | a 35B |
| **35B-A3B** | 35B | 3.3B | **19 GB** | **1.8 GB** | **~85 tok/s** | between the two |

Read the columns as three separate resources:

- **Total params buys quality.** Capacity for feature composition and stored
  regularities scales with the parameter count you trained, not the count you
  read. A 35B-A3B genuinely knows more than a dense 3B.
- **Active params buys speed.** Decode latency tracks bytes read per token, and
  nothing else, at batch 1.
- **Total params also buys your RAM bill.** Every expert must be resident,
  because the next token may select any of them. This is the column people
  delete from the table and then get surprised by.

That last row is the sweet spot for a laptop specifically: you have 32 or 64 GB
of unified memory - enough capacity to hold 19 GB - and you have bandwidth in
the low hundreds of GB/s, which is scarce. MoE spends the resource you have
(capacity) to buy the resource you lack (bandwidth). On a datacenter H100 with
3.3 TB/s the calculus is completely different, and so is the answer.

**How much quality does 35B-A3B actually deliver?** Somewhere between a dense
3B and a dense 35B, and the honest answer is that there is no clean law. The
folk heuristic is the geometric mean:

$$\sqrt{\text{total} \times \text{active}} = \sqrt{35 \times 3.3} \approx 10.7\text{B}$$

so "roughly comparable to a dense 10-11B". Treat this as a rule of thumb with
weak empirical backing, not a scaling law - it varies with $k$, with $N$, and
with how much of the model stays dense. What is solid is the ordering: total
params dominate quality, active params dominate speed, and MoE decouples them.

```beat
id: u7-b7
type: predict
concept: c-moe-tradeoff
prompt: |
  You take this same 35B-A3B (128 experts, top-8) and deploy it on a server
  handling 64 concurrent users. You batch their decode steps together: one
  forward pass advances all 64 sequences by one token.

  Before reading on: what fraction of the 128 experts does that batched
  forward pass have to load? Assume routing is well balanced, so each token
  picks 8 experts roughly uniformly at random.
answer: |
  Almost all of them - about 126 of 128, or 98%. The probability a given
  expert is missed by one token is (1 - 8/128) = 0.9375; missed by all 64
  is 0.9375^64 = 0.016. So the expected number of distinct experts touched
  is 128 * (1 - 0.016) = 125.9. The batch must read essentially the entire
  35B model, and the MoE bandwidth advantage collapses to nothing.
rubric: |
  Must reach "nearly all of them" / >90%, and must give the reason: the
  union of 64 independent top-8 selections covers almost the whole expert
  set. Exact arithmetic not required, but the union-of-random-subsets
  argument is. Answering "still 8" or "still 1/16 of them" is the key error -
  it treats routing as a per-request property rather than a per-token,
  per-layer one. Answering "half" without reasoning = partial.
check: llm
```

The general shape: with $N$ experts, top-$k$, and batch size $B$, the expected
number of distinct experts read is

$$N \left(1 - \left(1 - \tfrac{k}{N}\right)^{B}\right)$$

which for $N = 128, k = 8$ gives 8 experts at $B{=}1$, 29 at $B{=}4$, 82 at
$B{=}16$, and 126 at $B{=}64$. MoE's bandwidth win is a *small-batch* win. It is
enormous for one person on a laptop, substantial at low concurrency, and gone
by the time you are saturating a serving node - where the win becomes a FLOPs
win instead, which is a different and smaller benefit. Note the irony: good load
balancing, the thing that makes single-token routing healthy, is exactly what
makes the batched union cover everything.

## What MoE costs you

Nothing here is free. Four real costs, in rough order of how much they will
bite you.

**1. Memory capacity, at full total-parameter price.** Covered above and worth
repeating because it is the most common wrong expectation: 35B-A3B needs the
RAM of a 35B, not of a 3B. Paging experts from SSD as they are selected sounds
clever and is catastrophic - you would be reading 1.8 GB per token over a
~5 GB/s link, which lands you well below the dense ceiling you were trying to
beat.

**2. Training instability.** This is the cost the papers are actually about.
Three interacting problems:

- *Router collapse*, the positive feedback loop described earlier. The aux loss
  suppresses it but does not eliminate it, and the balance between $\alpha$ too
  small (collapse) and $\alpha$ too large (routing becomes noise, quality
  drops) is genuinely delicate.
- *Discontinuous gradients.* Top-$k$ is an argmax, and argmax has zero gradient
  almost everywhere. The router only learns through the gate weights $g_i$ of
  the experts it *already* selected. It gets no gradient signal about the
  experts it did not pick, so it cannot directly learn "expert 5 would have
  been better here". Training relies on noise and the aux loss to keep
  exploring.
- *Loss spikes.* MoE runs diverge more often than dense runs at the same scale,
  and a small routing change can flip a token's entire computational path,
  which makes the loss surface rougher. Practitioners run with more aggressive
  gradient clipping, higher-precision router logits, and checkpoint restarts.

**3. Implementation complexity.** A dense MLP is one batched matmul. An MoE
layer is a gather, a permutation of tokens into per-expert groups, a set of
variable-sized matmuls, and a scatter back - with capacity limits that drop
tokens when an expert overflows its slot budget in training. Distributed
training adds an all-to-all communication step per MoE layer, and expert
parallelism has a completely different sharding story from tensor parallelism.

**4. Quality per total parameter is lower.** A dense 35B beats a 35B-A3B. That
is the trade you are making: you accept somewhat less quality per parameter
stored in exchange for far more quality per parameter *read*. If you are
latency-bound, that is a bargain. If you are memory-capacity-bound, it is a bad
deal, and you want the dense model.

```beat
id: u7-b8
type: self-explain
concept: c-moe-tradeoff
prompt: |
  A colleague has 8 GB of free RAM on a laptop with 100 GB/s of bandwidth.
  They ask whether they should run a 4-bit 35B-A3B (19 GB) or a 4-bit dense
  8B (4.4 GB).

  Give them the answer and the reason, in terms of which resource binds.
  Then state the one machine change that would flip your recommendation.
answer: |
  The dense 8B. The 35B-A3B does not fit: all 128 experts must be resident,
  so it needs 19 GB regardless of the fact that only 1.8 GB is read per
  token. Capacity binds before bandwidth does, and a model that does not fit
  runs at swap speed or not at all. The active-parameter advantage is
  irrelevant when you cannot hold the total parameters.
  The change that flips it: more RAM (32 GB+). Adding bandwidth would not
  help - bandwidth is the resource MoE is good at spending capacity to save,
  and they have no capacity to spend.
rubric: |
  Pass requires: (1) recommend the dense 8B; (2) the reason is capacity -
  total params, not active params, set the RAM requirement; (3) the flip is
  more RAM, not more bandwidth / not a faster GPU. Recommending the A3B
  because "only 3B is active" is U7-M1 and is a fail. Saying "more bandwidth"
  for the flip shows the two resources are still conflated - partial at best.
check: llm
```

**Where this leaves you.** The model narrating this book is 35B of trained
capacity that costs 3.3B of read bandwidth per token. It sits in 19 GB of your
unified memory and streams 1.8 GB of that per token, which is why it feels like
a conversation rather than a progress bar. The router that makes it possible is
one small matrix per layer, trained by the same loss as everything else,
partitioning tokens on a basis nobody chose and nobody can fully name.
