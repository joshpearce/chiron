## The machine you are actually running on

**Where the 989 TFLOP/s comes from.** The number is not a benchmark; it is a
count of wires, and reconstructing it removes the mystery.

An H100 SXM has 132 streaming multiprocessors, each with 4 tensor cores, at a
boost clock of 1.755 GHz. An H100 bf16 tensor core retires a 16x8x16
matrix-multiply-accumulate per instruction issue, which is
$2 \times 16 \times 8 \times 16 = 4{,}096$ FLOPs, and the pipeline sustains one
such issue every 4 cycles, giving 1,024 FLOPs per tensor core per cycle:

$$132 \times 4 \times 1024 \times 1.755 \times 10^{9} = 9.50 \times 10^{14} = 950 \text{ TFLOP/s}$$

which is within 4% of the published 989 (the difference is the exact clock
NVIDIA quotes). No caching, no cleverness - a fixed number of multipliers
switching at a fixed rate. Peak FLOP/s is a property of the silicon that no
software can exceed, which is what makes it a *roof*.

**Where 3.35 TB/s comes from.** HBM3 on an H100 is 5 stacks of 16 channels,
each channel 64 bits wide, at 5.2 Gbit/s per pin:

$$5 \times 16 \times 64 \times 5.2 \times 10^{9} / 8 = 3.33 \times 10^{12} \text{ bytes/s}$$

Also a count of wires. The ratio of the two counts is the machine balance, and
it has drifted in one direction for twenty years: transistor density improves
faster than off-chip pin bandwidth, so balance rises with every generation. The
V100 sat near 139 FLOP/byte in fp16, the 80 GB A100 near 153, the H100 at 295. Each
generation is harder to feed than the last, which is why algorithmic work on
arithmetic intensity has become more valuable over time rather than less.

**The energy argument, stated properly.** The cost of moving a bit off-chip is
dominated by charging the capacitance of the wire, $E = \tfrac{1}{2} C V^2$ per
transition, with $C$ proportional to wire length. On-die wires are micrometers;
package traces are millimeters; board traces are centimeters. The three regimes
differ by roughly two orders of magnitude in $C$ and therefore in energy, and
$V$ has stopped scaling because threshold voltages have floored out. Arithmetic
energy, by contrast, scales with the number and size of the transistors doing
it and has fallen with every node. Hence the divergence: FLOPs get cheaper,
bytes do not.

## Arithmetic intensity and the roofline

**Where the model comes from and what it hides.** Williams, Waterman and
Patterson's roofline (2009) is a bound argument, not a simulation. Let a
kernel perform $F$ FLOPs and move $Q$ bytes across the boundary of interest,
on a machine with peak rate $\pi$ FLOP/s and peak bandwidth $\beta$ bytes/s.
The kernel cannot finish faster than the arithmetic allows, $t \ge F/\pi$, and
it cannot finish faster than the transfer allows, $t \ge Q/\beta$. Therefore

$$t \ge \max\left(\frac{F}{\pi},\ \frac{Q}{\beta}\right)$$

and the attained rate satisfies

$$\frac{F}{t} \le \min\left(\pi,\ \frac{F}{Q}\beta\right) = \min(\pi,\ I\beta)$$

with $I = F/Q$. The inequality is where all the honesty lives. It assumes
compute and transfer overlap perfectly - that the machine achieves the max of
the two times rather than their sum. Real kernels fall short of the roof for
reasons the model does not represent: cache misses, insufficient occupancy to
hide latency, non-coalesced access patterns, and the fact that $\beta$ itself
is achievable only with near-perfect access streams. The roofline gives an
upper bound and a *diagnosis of which resource to attack*. It does not predict
attained performance, and a kernel at 60% of its roofline bound is doing well.

**Naming the boundary.** $Q$ is defined relative to a chosen memory level, and
changing the level changes every number. Draw the boundary at HBM and a fused
kernel's intensity counts only HBM traffic, so fusion raises $I$. Draw it at
L2 and fusion changes nothing, because the traffic simply moved. Draw it at the
network and you get a *communication* roofline, which is the right tool for the
parallelism section below. A hierarchical roofline plots one line per level and
shows which level is binding. Stating the boundary is not pedantry; it is the
difference between a correct analysis and a number.

**The batched-matmul intensity, done exactly.** In canon the activation traffic
was waved away as negligible. Keep it. For $W \in \mathbb{R}^{m \times n}$ and
$X \in \mathbb{R}^{n \times B}$ at $b$ bytes per element throughout:

$$I = \frac{2mnB}{mn\,b + nB\,b + mB\,b} = \frac{2}{b} \cdot \frac{1}{\frac{1}{B} + \frac{1}{m} + \frac{1}{n}}$$

which is the harmonic-mean form that shows up everywhere in matmul analysis.
Three consequences fall out immediately:

- As $B \to \infty$ with $m, n$ fixed, $I \to \frac{2}{b}\cdot\frac{mn}{m+n}$,
  not infinity. Batching alone cannot make intensity unbounded; the activation
  traffic eventually dominates. For $m = n = 8192$ and $b = 2$ that asymptote is
  4,096 FLOP/byte - comfortably past the ridge, which is why the approximation
  in canon is safe for transformer-shaped matrices.
- Setting $B = 1$ recovers $I = \frac{2}{b} \cdot \frac{1}{1 + 1/m + 1/n}
  \approx 2/b$, the canon result, with the error term explaining why real
  batch-1 decode measures slightly below the ideal.
- The expression is symmetric in $B$, $m$, and $n$. Small matrices hurt exactly
  as much as small batches, which is the formal version of "the matmuls are too
  small."

**Ridge point and the shape of the log-log plot.** On axes $\log I$ versus
$\log(\text{attainable})$, the bandwidth roof $I\beta$ is a straight line of
slope exactly 1 (since $\log(I\beta) = \log I + \log\beta$) and the compute
roof is horizontal. They intersect at $I = \pi/\beta = B$. Note the asymmetry
in how far a workload can be from the roof: below the ridge, the *ratio*
$B/I$ tells you the factor by which you are wasting arithmetic, and above it,
$I/B$ tells you the factor of bandwidth slack you have available to spend. Both
are dimensionless and both are worth computing before touching any code.

## FlashAttention: changing the bytes, not the FLOPs

**The online softmax, derived.** This is the piece canon states and does not
prove, and it is short.

The stable softmax of a vector $x \in \mathbb{R}^{N}$ subtracts the maximum
before exponentiating:

$$\sigma(x)_i = \frac{e^{x_i - m}}{\sum_{j=1}^{N} e^{x_j - m}}, \qquad m = \max_j x_j$$

which is algebraically identical to the naive form (numerator and denominator
are each scaled by $e^{-m}$) and avoids overflow. The problem for tiling is
that $m$ and the denominator are functions of the whole row.

Process the row in blocks. After block $t$, maintain two scalars:

$$m^{(t)} = \max_{j \le \text{end of block } t} x_j, \qquad \ell^{(t)} = \sum_{j \le \text{end of block } t} e^{x_j - m^{(t)}}$$

and one accumulator $o^{(t)} \in \mathbb{R}^{d}$ holding the unnormalized
weighted sum of value vectors seen so far, referenced to the same $m^{(t)}$:

$$o^{(t)} = \sum_{j \le \text{end of block } t} e^{x_j - m^{(t)}} v_j$$

Now block $t+1$ arrives with its own local maximum $\tilde{m}$ and local
quantities $\tilde{\ell} = \sum_{j \in \text{block}} e^{x_j - \tilde{m}}$ and
$\tilde{o} = \sum_{j \in \text{block}} e^{x_j - \tilde{m}} v_j$. Set
$m^{(t+1)} = \max(m^{(t)}, \tilde{m})$. Then

$$\ell^{(t+1)} = e^{m^{(t)} - m^{(t+1)}} \ell^{(t)} + e^{\tilde{m} - m^{(t+1)}} \tilde{\ell}$$
$$o^{(t+1)} = e^{m^{(t)} - m^{(t+1)}} o^{(t)} + e^{\tilde{m} - m^{(t+1)}} \tilde{o}$$

The identity behind both lines is the same one: for any $a$ and $a'$,

$$e^{x_j - a} = e^{a' - a} \cdot e^{x_j - a'}$$

so re-referencing an accumulated sum from maximum $a'$ to maximum $a$ is one
scalar multiplication by $e^{a' - a}$, applied to the whole accumulator. Nothing
is approximated. After the final block, $o^{(T)} / \ell^{(T)}$ is exactly the
attention output for that row.

The cost is one extra multiply per accumulator element per $K$/$V$ block. With
block width $B_c$ there are $n/B_c$ blocks per row and $d$ accumulator elements
per row, so the total is $n^2 d / B_c$ extra FLOPs against $4 n^2 d$ of real
work - a ratio of $1/(4B_c)$, which at a typical $B_c = 64$ is 0.4%. You pay a
fraction of a percent in arithmetic to remove 98% of the memory traffic. That
is the trade the roofline says to make, quantified.

**The backward pass.** FlashAttention's backward is where the memory saving
becomes structural rather than merely large. A conventional backward through
attention needs $P = \text{softmax}(S)$ to compute $\partial L/\partial V =
P^{T} (\partial L/\partial O)$, and $P$ is the $n \times n$ matrix you just
avoided storing. FlashAttention recomputes $S$ and $P$ tile by tile during the
backward pass from the saved $Q$, $K$, $V$ and the saved per-row statistics
$m$ and $\ell$ - which are only $O(n)$ scalars. This is gradient checkpointing
applied at sub-layer granularity, and it is the same recompute-versus-store
trade as the training section, arrived at independently.

**IO complexity, stated as a theorem.** With SRAM of size $M$ and head width
$d$, FlashAttention's HBM traffic is $\Theta(n^2 d^2 / M)$ against standard
attention's $\Theta(nd + n^2)$. Because $M \gg d^2$ on real hardware
($M \approx 10^5$ elements, $d^2 = 16{,}384$ for $d = 128$), the ratio is a
large constant favouring Flash. The dependence on $M$ is the interesting part:
it says the algorithm's advantage is a function of the *hardware's* SRAM
budget, which is why the tile sizes are hardware-specific and why
FlashAttention-2 and -3 exist as retunings for newer chips rather than as new
mathematics.

## Why training and inference want different iron

**A closed form for the ridge in tokens.** Combining the definitions, a
workload with $b$ bytes per weight on a machine of balance $B$ crosses the
ridge at

$$B_{\text{tok}}^{*} = \frac{B \cdot b}{2}$$

positions per forward pass. Read it as a design rule. Lowering $b$ - quantizing
weights - *lowers* the crossover, so quantized models become compute-bound at
smaller batches. At 4 bits ($b = 0.5$) on an H100, $B^{*}_{\text{tok}} = 74$
rather than 295. Quantization is therefore not purely a bandwidth win at scale;
past 74 concurrent positions the machine stops caring about the weight bytes
and the win shrinks. Conversely fp8 tensor cores double $B$ to 591, pushing the
ridge back out - but if you also store the weights in fp8, $b$ halves to 1 and
the two effects cancel *exactly*, leaving the crossover at 295 positions again.
The knobs push in opposite directions and sometimes annihilate, which is why
"is lower precision faster" has no batch-independent answer.

**Training's memory arithmetic in full.** Per parameter, mixed-precision AdamW:

$$\underbrace{2}_{\text{bf16 } w} + \underbrace{2}_{\text{bf16 } g} + \underbrace{4}_{\text{fp32 } w_{\text{master}}} + \underbrace{4}_{\text{fp32 } m} + \underbrace{4}_{\text{fp32 } v} = 16 \text{ bytes}$$

Activations are separate and depend on the parallelism strategy. For one
transformer layer at microbatch $B_{\text{tok}}$ tokens and width $d$, storing
every intermediate needed by the backward pass costs on the order of
$34 \cdot B_{\text{tok}} \cdot d$ bytes per layer in bf16 (the constant counts
the tensors a standard block saves: layernorm inputs, QKV projections, the
attention output, both MLP intermediates, and the dropout masks), plus
$5 \cdot B_{\text{tok}}^2 \cdot H_q$ for the attention probabilities if they are
materialized - which is precisely the term FlashAttention deletes. For $L = 80$,
$d = 8192$, $B_{\text{tok}} = 8192$: the first term is
$34 \times 8192 \times 8192 \times 80 = 1.83 \times 10^{11}$ bytes, 183 GB,
larger than two H100s of HBM for the activations of one microbatch. Activation
checkpointing is what makes this fit, and the quadratic term is what made
long-context training impossible before 2022.

**ZeRO staging, quantified.** For a data-parallel group of size $P$:

| stage | shards | bytes/param/device |
|---|---|---|
| baseline | nothing | 16 |
| ZeRO-1 | optimizer state ($m$, $v$, master) | $4 + 12/P$ |
| ZeRO-2 | + gradients | $2 + 14/P$ |
| ZeRO-3 / FSDP | + weights | $16/P$ |

At $P = 64$: 16 bytes falls to 4.19, 2.22, and 0.25 respectively. The cost is
communication volume: ZeRO-3 must all-gather each layer's weights before use
and free them after, adding roughly $2 \times$ the parameter bytes of traffic
per step on top of the gradient reduce-scatter. Whether that is affordable is
another roofline question, drawn at the network boundary.

## Four ways to split a model, and what each one stresses

**Ring all-reduce, derived.** Split the buffer $S$ into $P$ chunks. The
reduce-scatter phase runs $P-1$ steps, each device sending one chunk
($S/P$ bytes) to its successor and reducing the chunk it receives; after $P-1$
steps each device holds the fully reduced value of one distinct chunk. The
all-gather phase runs another $P-1$ steps circulating those reduced chunks. So
each device sends

$$2 \cdot (P-1) \cdot \frac{S}{P} = 2S\frac{P-1}{P}$$

bytes, and with links of bandwidth $\beta_{\text{link}}$ operating
concurrently, the time is $2S(P-1)/(P\beta_{\text{link}})$. The remarkable
property is that this is *independent of $P$* to first order - doubling the
cluster does not increase per-device all-reduce time. Latency does grow, since
there are $2(P-1)$ sequential steps each carrying a fixed overhead $\alpha$, so
the full alpha-beta cost model is

$$T = 2(P-1)\alpha + \frac{2S(P-1)}{P\beta_{\text{link}}}$$

The first term dominates for small $S$ (which is why gradients are bucketed
into tens of MB rather than reduced per-tensor) and the second for large $S$.
Tree and hierarchical algorithms trade between the two: a tree reduces the step
count to $O(\log P)$ but sends more bytes per device, so it wins on latency for
small buffers and loses on bandwidth for large ones. Production stacks switch
algorithm by message size for exactly this reason.

**Pipeline bubble, derived.** With $p$ stages and $m$ microbatches under the
1F1B (one-forward-one-backward) schedule, each stage does $m$ forward and $m$
backward units of work. The pipeline needs $p-1$ units to fill and $p-1$ to
drain, and each of those is idle time for some stage. Total time in units of
one stage-microbatch is $m + p - 1$ for the forward direction and the same
structure mirrored for backward, so

$$\text{bubble fraction} = \frac{p-1}{m+p-1}$$

Two refinements matter. Interleaved schedules assign $v$ non-contiguous chunks
of layers to each device, dividing the bubble by $v$ at the cost of $v$ times
the point-to-point messages: $(p-1)/(v(m+p-1))$. And the derivation assumes
equal stage times; in practice the first stage carries the embedding and the
last carries the output head and the loss, so stages are unequal and the true
bubble is set by the slowest. Balancing layer counts against those extra
tensors is a real and fiddly part of configuring a run.

**Why tensor parallelism costs activations and data parallelism costs
parameters.** The asymmetry is worth stating as a formula. Tensor-parallel
traffic per step scales as $O(L \cdot B_{\text{tok}} \cdot d)$ - layers times
tokens times width, because it moves activations. Data-parallel traffic scales
as $O(N)$ - the parameter count, because it moves gradients, and it does not
depend on the batch at all. Increase the batch and tensor parallelism gets more
expensive while data parallelism gets *cheaper per token*. That single
observation determines most of the configuration search: large batches favour
data parallelism, and the tensor-parallel degree is pinned near the NVLink
domain size rather than tuned.

## Precision: range is the scarce resource, not resolution

**Representable sets, precisely.** A binary floating-point format with $e$
exponent bits and $s$ mantissa bits represents

$$x = (-1)^{\text{sign}} \cdot 2^{E - \text{bias}} \cdot \left(1 + \frac{M}{2^{s}}\right), \qquad \text{bias} = 2^{e-1} - 1$$

for $1 \le E \le 2^{e} - 2$, with $E = 0$ reserved for zero and subnormals and
$E = 2^{e}-1$ for infinities and NaN. Two derived quantities carry everything:

$$\text{dynamic range} = \frac{x_{\max}}{x_{\min,\text{normal}}} \approx 2^{2^{e} - 2}, \qquad \text{unit roundoff} = 2^{-(s+1)}$$

For fp16 ($e=5$, $s=10$): range $2^{30} \approx 10^{9}$, roundoff
$2^{-11} = 4.9 \times 10^{-4}$. For bf16 ($e=8$, $s=7$): range
$2^{254} \approx 10^{76}$, roundoff $2^{-8} = 3.9 \times 10^{-3}$. bf16 has
$2^{224} \approx 2.7 \times 10^{67}$ times the dynamic range and 8 times the
roundoff error. Since
gradient magnitudes across a deep network routinely span 8 to 10 orders of
magnitude within a single step, and every value is used in a product where a
4x10^-3 relative error is one part of a much larger stochastic-gradient noise
budget, the trade is not close.

**Why relative error is the right metric and absolute error is not.** Floating
point has uniform *relative* spacing: the gap between representable neighbours
scales with the magnitude. So rounding is multiplicative noise,
$\hat{x} = x(1 + \delta)$ with $|\delta| \le u$, the unit roundoff. Propagate
that through a dot product of length $n$ with the standard backward-error
result: the computed value is the exact dot product of slightly perturbed
inputs, with relative perturbation bounded by $nu/(1-nu)$. For bf16 with
$u = 3.9\times10^{-3}$ and $n = 8192$ that bound is useless (it exceeds 1),
which is precisely why **tensor cores accumulate in fp32 even when their inputs
are bf16**. The multiplies are 16-bit; the running sum is 32-bit. The published
"bf16 matmul" is a mixed operation, and the bound that applies is fp32's
$u = 6 \times 10^{-8}$ over the accumulation, with a single bf16 rounding on
each input. Without fp32 accumulation none of this would work, and the
misconception that low precision is fine would be false.

**Loss scaling, and why it is exactly a change of variables.** Backprop is
linear in the incoming gradient: scaling the loss by $S$ scales every gradient
in the graph by $S$ exactly, with no other effect, because
$\partial(SL)/\partial w = S \cdot \partial L / \partial w$ for every $w$. So
loss scaling is a similarity transform on the backward pass that is undone by
dividing by $S$ before the optimizer step. It changes nothing mathematically
and everything numerically. Dynamic loss scaling implements a search: multiply
$S$ by 2 every $k$ successful steps, and on any step where a gradient becomes
inf or NaN, halve $S$ and *discard the step*. The discarded steps are the real
cost, and they are why a badly tuned scaler shows up as a training curve with
periodic flat spots.

**fp8 and per-tensor scaling.** E4M3 has $x_{\max} = 448$ and unit roundoff
$2^{-4} = 0.0625$; E5M2 has $x_{\max} = 57{,}344$ and $2^{-3} = 0.125$. Neither
has the range to hold a raw tensor whose values span several orders of
magnitude, so each tensor carries an fp32 scale $s$ and stores $x/s$, with $s$
chosen from the running maximum absolute value over recent steps (a delayed
scaling scheme, so the scale is available before the tensor is produced). This
is loss scaling generalized from one global constant to one constant per
tensor, updated continuously. The accumulation is still fp32. The pattern is
now three deep: 16-bit storage with 32-bit accumulate, then 8-bit storage with
per-tensor scales and 32-bit accumulate, and the same structure is what 4-bit
inference quantization uses with per-*group* scales (u6). Three techniques, one
idea: keep the exponent in a wider format alongside the data, and spend the
narrow format's bits entirely on mantissa.

## What a frontier run physically is

**Where $6ND$ comes from.** Each parameter participates in one multiply and one
add per token in the forward pass: $2N$ FLOPs per token. The backward pass
computes two gradients per weight-layer - one with respect to the input
activations (needed to continue backward) and one with respect to the weights
(needed for the update) - each costing the same $2N$ per token as the forward
matmul. So backward is $4N$ and the total is $6N$ FLOPs per token, or

$$C \approx 6ND$$

over $D$ tokens. The approximation drops attention's $O(n^2)$ term, which is
exact only in the regime where $n \ll d_{model} \cdot L / (\text{something})$;
a fuller expression adds $12 \cdot L \cdot n \cdot d_{model}$ FLOPs per token
of attention score work. At $L=80$, $d=8192$, $n = 8192$ that term is
$6.4 \times 10^{10}$ against $6N = 4.2 \times 10^{11}$ for a 70B model - about
15%, not negligible at long context, which is why long-context training runs
report lower MFU against the $6ND$ definition. The definition is a convention,
and comparing MFU across runs with different sequence lengths quietly compares
different things.

**The checkpoint-interval optimum.** Let $\tau$ be the checkpoint interval,
$c$ the time to write a checkpoint, and $\lambda$ the failure rate. Expected
overhead fraction per unit of useful work is approximately

$$f(\tau) = \frac{c}{\tau} + \frac{\lambda \tau}{2}$$

- the first term the writing cost amortized over the interval, the second the
expected lost work (half an interval per failure, at $\lambda$ failures per unit
time). Differentiate and set to zero:

$$-\frac{c}{\tau^2} + \frac{\lambda}{2} = 0 \implies \tau^{*} = \sqrt{\frac{2c}{\lambda}}$$

This is Young's formula (1974), rediscovered constantly. With $c = 60$ seconds
to write and $\lambda = 1/(3\ \text{hours}) = 9.26\times10^{-5}$ per second:

$$\tau^{*} = \sqrt{2 \times 60 / 9.26\times10^{-5}} = 1{,}138 \text{ s} \approx 19 \text{ minutes}$$

and the overhead at the optimum is $f(\tau^*) = \sqrt{2c\lambda} = 10.5\%$.
The square root is the useful structure: halving the failure rate lengthens the
optimal interval by only 41% ($\sqrt{2}$) and cuts the overhead by only 29% -
fault tolerance at this scale has sharply diminishing returns from reliability
improvements alone,
which is why the engineering effort goes into making restart *fast* rather than
making failures rare.

**MFU and HFU, related.** With activation checkpointing at recomputation factor
$r$ (1.0 for none, 1.333 for full):

$$\text{HFU} = r \cdot \text{MFU}$$

exactly, since the hardware executes $r \cdot 6ND$ FLOPs in the same wall-clock
in which the model required $6ND$. A run reporting 40% HFU with full
checkpointing is a 30% MFU run. Papers that quote the larger number without
saying which are not lying, but they are not comparable either.

## The law under all of it

**The whole unit as one optimization problem.** Everything in this unit is a
move in the same two-variable space. A workload is a point $(F, Q)$ - FLOPs and
bytes - and running it takes $\max(F/\pi, Q/\beta)$ seconds. Four kinds of move
exist and there are no others:

1. **Lower $Q$ at fixed $F$** - quantization, mixture of experts,
   grouped-query attention, FlashAttention, kernel fusion. This is the dominant
   move in inference and it is what every technique in u6 and u7 turned out to
   be. It has a hard limit: once $Q/\beta < F/\pi$ you are compute-bound and
   further reduction buys exactly nothing.
2. **Lower $F$ at fixed $Q$** - sparsity, early exit, smaller models,
   speculative decoding's amortization. This is the dominant move in training,
   where you are already compute-bound, and it does nothing at all for
   single-user decode.
3. **Raise $\beta$ or $\pi$** - buy different hardware. The only move that
   changes the machine rather than the workload, and the only one that costs
   money rather than cleverness.
4. **Raise both $F$ and $Q$ while raising $F/Q$** - batching. Strictly more
   total work, done faster per unit. The move that feels wrong to an engineer
   trained on latency and is correct almost everywhere in this field.

The reason the unit spends its length on move 1 is that it is the only one
available when the hardware is fixed, the model is fixed, and there is one
user - which is the situation on your tray table, and the situation in which
this book is being generated.
