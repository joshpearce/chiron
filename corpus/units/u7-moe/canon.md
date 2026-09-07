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

  Before reading on, commit to an answer: what is the hard upper bound on
  tokens per second, and what physical quantity sets it?
options:
  - text: "About 8 tok/s, set by memory bandwidth: one token needs one full read of the weights, so the ceiling is $153 / 19 \\approx 8$ tok/s."
    correct: true
    explain: "Right. At batch 1 every weight matrix takes part in exactly one matrix-vector product, so the traffic per token is the whole 19 GB, and bandwidth divided by that is the ceiling."
  - text: "There is no fixed ceiling from the weights - the limit is the GPU's FLOP/s, so a faster GPU raises it roughly in proportion."
    misconception: M10
    explain: "A matrix-vector product offers only $2/b \\approx 3.7$ FLOP per byte moved while the machine wants about 65. The GPU is idle ~94% of decode; adding FLOPs buys nothing."
  - text: "The ceiling is set by the KV cache: the growing cache is what must be re-read each step, so tokens per second falls as the context lengthens."
    misconception: M17
    explain: "The KV cache exists precisely so old tokens are not re-processed, and at short context it is small beside 19 GB of weights. The weights are re-read every single token; the cache is the optimization over the sequence, not the binding cost here."
  - text: "About 90 tok/s: only the roughly 2 GB of weights that matter for this token have to be read, since the rest of the network contributes nothing."
    misconception: U7-M1
    explain: "That is the MoE answer applied to a dense model. In a dense 35B there is no selection - every weight participates in every token, so all 19 GB moves."
check: choice
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
                        g for the larger  = ____
                        g for the smaller = ____
      Step 4 - combine: E_2(x) = [1, 0, 0],  E_0(x) = [0, 2, 0]
                        y = ____

  Which filling of the blanks is correct?
options:
  - text: "$l_2 = 3$; experts 2 and 0 run; $g_2 = e^3/(e^3+e^2) = 20.086/27.475 = 0.731$ and $g_0 = 7.389/27.475 = 0.269$; $y = 0.731[1,0,0] + 0.269[0,2,0] = [0.731,\\; 0.538,\\; 0]$."
    correct: true
    explain: "Right. $l_2 = 1(2) + 0(-1) + 1(1) = 3$, the two largest of $[2,-1,3,0]$ are $l_2$ and $l_0$, and the softmax is taken over those two logits alone so the gates sum to 1."
  - text: "$l_2 = 3$; experts 2 and 0 run; softmax over all four logits gives $g_2 = e^3/(e^2+e^{-1}+e^3+e^0) = 0.696$ and $g_0 = 0.256$; $y = [0.696,\\; 0.512,\\; 0]$."
    misconception: M2
    explain: "The gates must be renormalized over the selected set only. Softmaxing over all $N$ and keeping two weights leaves them summing to 0.952, so the block's write to the residual stream is silently scaled down by a token-dependent factor."
  - text: "$l_2 = 3$; experts 2 and 3 run (the top score and the router's default row); $g = 0.953 / 0.047$; $y = 0.953[1,0,0] = [0.953,\\; 0,\\; 0]$."
    misconception: M12
    explain: "Top-$k$ is just the $k$ largest entries of $\\ell = [2,-1,3,0]$: that is $l_2 = 3$ and $l_0 = 2$. Row 3 is not a default or fallback handler, it is an expert that scored 0 and lost."
  - text: "$l_2 = 3$; experts 2 and 0 run; the selected experts are averaged unweighted, $g_2 = g_0 = 0.5$; $y = [0.5,\\; 1,\\; 0]$."
    misconception: M12
    explain: "The gate weights are the point of the softmax: they carry how strongly the router preferred each selected expert, and they are the path by which $W_r$ receives gradient. An unweighted average discards both."
check: choice
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
  loss, and (b) an auxiliary loss $L_{\text{aux}} = \alpha N \sum_i f_i P_i$
  penalizes any routing distribution that concentrates on a few experts.

  Which explanation correctly accounts for why *domain-specialist* experts do
  not emerge - even though "one expert per domain" would be a perfectly
  reasonable way to partition the work - and for what the experts do end up
  partitioning on?
options:
  - text: "Domain is a badly balanced variable: corpora hold far more general English than, say, Lisp, so a domain partition gives wildly unequal $f_i$ - exactly what the aux loss punishes, since it wants each expert near $1/N$ of every batch. Meanwhile the LM loss rewards only per-token loss reduction and is indifferent to whether the partition is nameable. So the router settles on something roughly equal-frequency and locally predictive: subword shape, punctuation, numerals, whitespace."
    correct: true
    explain: "Both forces, correctly. The aux loss rules out the skew that any domain partition would produce, and nothing in the objective ever pays for interpretability - so the emergent partition is token-level and syntactic."
  - text: "Specialists do form - the router really is classifying content - but the aux loss forces it to spread overflow onto neighbouring experts, so the domain structure is there and merely blurred by imperfect routing."
    misconception: M12
    explain: "This is the specialist story surviving as 'specialists plus noise'. The failing prediction still holds: routing changes token to token inside one sentence and again at every one of 48 layers. A French expert deselected between the article and the noun was never a French expert."
  - text: "The router has its own training objective - the balance loss - which is a different goal from the LM loss, and optimizing that separate objective is what scrambles the domain partition the language model would otherwise have learned."
    misconception: U7-M2
    explain: "There is no separate router objective or training stage. $W_r$ is one linear layer inside the block receiving gradient through the gates $g_i$ from the same next-token loss; the aux term is added to that one objective, not optimized apart from it."
  - text: "Each expert is effectively a small complete model, and complete models trained on the same data all learn much the same thing, so no expert can end up specialized in any particular domain."
    misconception: U7-M3
    explain: "An expert is one MLP sublayer in one layer of forty-eight - no attention, no embeddings, no output head. It maps a residual-stream vector to a residual-stream update and cannot be run alone."
check: choice
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

  Which filling of the blanks is correct?
options:
  - text: "Step 2: 3 GB. Step 3: $400/3 = 133$ tok/s. Step 4: $400/24 = 16.7$ tok/s. Step 5: 8x, which is exactly the total/active ratio $48/6$. Step 6: 24 GB."
    correct: true
    explain: "Right, and Step 6 is the one that matters: all 48B stay resident because the next token may select any expert. The speedup equalling total/active is not a coincidence - both ceilings are $400$ divided by bytes read."
  - text: "Step 2: 3 GB. Step 3: $400/3 = 133$ tok/s. Step 4: $400/24 = 16.7$ tok/s. Step 5: 8x. Step 6: 3 GB - only the active slice has to be held in RAM."
    misconception: U7-M1
    explain: "Total parameters set the capacity requirement; active parameters set the per-token bandwidth cost. Any expert may be selected by the next token, so all 24 GB must be resident - streaming 3 GB per token off a ~5 GB/s SSD would land you below the dense ceiling you were beating."
  - text: "Step 2: 3 GB. Step 3: $400/3 = 133$ tok/s. Step 4: $400/24 = 16.7$ tok/s. Step 5: 8x. Step 6: 6 GB - enough to hold the 6B of active parameters plus room for the router and attention."
    misconception: U7-M1
    explain: "Same error in a more careful disguise. The active set is not a fixed 6B of weights you could keep resident - it is re-chosen per token and per layer, so the residency requirement is the full 24 GB."
  - text: "Step 2: 24 GB, since a forward pass still touches every weight. Step 3: $400/24 = 16.7$ tok/s. Step 4: 16.7 tok/s. Step 5: 1x - MoE saves FLOPs, not memory traffic. Step 6: 24 GB."
    misconception: M10
    explain: "At batch 1 the unselected experts are never loaded, so only 3 GB moves; the FLOP saving and the bandwidth saving are the same ratio here. The union over experts only approaches the full model at large batch, which is a separate regime."
check: choice
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

  Commit before reading on: what fraction of the 128 experts does that
  batched forward pass have to load? Assume routing is well balanced, so
  each token picks 8 experts roughly uniformly at random.
options:
  - text: "Nearly all of them - about 126 of 128. One token misses a given expert with probability $1 - 8/128 = 0.9375$, all 64 miss it with probability $0.9375^{64} = 0.016$, so $128(1 - 0.016) \\approx 125.9$ distinct experts are read and the bandwidth advantage collapses."
    correct: true
    explain: "Right. The union of 64 independent top-8 draws covers almost the whole pool, so the batched pass reads essentially all 35B. Note the irony: good load balancing is exactly what makes the union cover everything."
  - text: "Still 8 of 128. The batch shares a routing decision per forward pass, so batching 64 users multiplies the tokens produced without changing which experts are loaded."
    misconception: U7-M4
    explain: "Routing is per token and per layer, never per pass or per request. Sixty-four tokens make sixty-four independent top-8 selections at every layer, and their union is nearly the whole set."
  - text: "About 64 of 128 - roughly half. The experts each token selects overlap heavily with the others', so the batch converges on the commonly-used half of the pool."
    misconception: M12
    explain: "Heavy overlap on a favoured subset is what the aux loss is built to prevent; routing is close to uniform. Compute the union rather than guessing at overlap: $128(1 - (1 - 8/128)^{64}) \\approx 126$."
  - text: "A number that grows with users but stays small: about 1/16 of the experts per user, so the serving node keeps the roughly 10x bandwidth win at any batch size."
    misconception: U7-M4
    explain: "The distinct-expert count grows as $N(1 - (1-k/N)^B)$ - 8 at $B{=}1$, 29 at $B{=}4$, 82 at $B{=}16$, 126 at $B{=}64$. The win is 10.6x at batch 1 and about 1.02x by batch 64: a small-batch, memory-bound win, not an unconditional one."
check: choice
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

  Which answer would you give them, and for which reason?
options:
  - text: "The dense 8B. The A3B needs all 128 experts resident, so it wants 19 GB whatever the 1.8 GB read per token says - capacity binds before bandwidth does, and a model that does not fit runs at swap speed or not at all. What would flip the recommendation is more RAM (32 GB+), not more bandwidth."
    correct: true
    explain: "Right on both counts. MoE spends the resource you have (capacity) to buy the one you lack (bandwidth); with no spare capacity there is nothing to spend, and adding bandwidth improves the resource MoE was already going to save you."
  - text: "The 35B-A3B. Only 3.3B parameters are active per token, about 1.8 GB, which fits inside 8 GB comfortably - and you get the knowledge of a 35B for the footprint of a small model."
    misconception: U7-M1
    explain: "Active parameters are a bandwidth figure, not a residency figure. The selected set changes every token at every layer, so all 19 GB must be held; below about 19 GB it does not run."
  - text: "The dense 8B, because 100 GB/s is too little bandwidth to feed an MoE's gather and scatter. What would flip it is a faster memory bus - at 400 GB/s the 35B-A3B becomes the better choice on this machine."
    misconception: U7-M1
    explain: "The recommendation is right but the reason is not, so the fix is wrong: bandwidth is the resource MoE saves. Tripling it still leaves 19 GB of weights with nowhere to sit; only more RAM changes the answer."
  - text: "The 35B-A3B, keeping only the hot experts in the 8 GB and paging the rest from SSD as the router selects them - the working set is small enough that most selections hit RAM."
    misconception: U7-M1
    explain: "There is no stable working set: routing is re-decided per token per layer, so you would stream about 1.8 GB per token over a ~5 GB/s link - roughly 2.8 tok/s, worse than the dense ceiling you were trying to beat."
check: choice
```

**Where this leaves you.** The model narrating this book is 35B of trained
capacity that costs 3.3B of read bandwidth per token. It sits in 19 GB of your
unified memory and streams 1.8 GB of that per token, which is why it feels like
a conversation rather than a progress bar. The router that makes it possible is
one small matrix per layer, trained by the same loss as everything else,
partitioning tokens on a basis nobody chose and nobody can fully name.
