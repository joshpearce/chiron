---
unit: u2
title: "The math floor"
concepts:
  - c-matrix-transform
  - c-matmul
  - c-dotprod
  - c-softmax
  - c-gradient
  - c-loss-surface
assumes:
  - c-notation
---

Everything in the rest of this book is four operations wearing costumes: multiply
by a matrix, take a dot product, softmax the result, and follow a gradient
downhill. Attention is those four. A transformer block is those four. Training is
those four. If you own this unit, the remaining units are bookkeeping.

You are not being asked to become a mathematician. You are being asked to hold
each piece of machinery precisely enough that you can predict what a change does
before running it. That is the same bar you already hold for a scheduler or an
allocator.

## Shapes are type signatures

A matrix is not a grid of numbers you happen to store in row-major order. A
matrix is a function, and its shape is that function's type signature.

Write $W \in \mathbb{R}^{m \times n}$ for a matrix with $m$ rows and $n$ columns.
Here $\mathbb{R}$ denotes the real numbers, and $m \times n$ means the array is
$m$ tall and $n$ wide. That object is a linear function

$$W : \mathbb{R}^{n} \rightarrow \mathbb{R}^{m}$$

**Orientation note, because it will bite you otherwise.** This unit writes the
map on the left of its argument, $Wx$, with $x$ a column vector. That is the
form every linear algebra text uses and the form every picture of a rotation is
drawn in, and the hand-worked 2D examples below are much easier to read that
way. It is the opposite of the row-vector form $XW$ that u0 established and that
u3 onward uses for data flowing through a model. They are the same maps with the
weights stored transposed: $(Wx)^T = x^T W^T$, so a $W$ of shape $(m, n)$ here
is a $W$ of shape $(n, m)$ there. Read the shape, not the letter order. The rule
that survives both conventions is the only one that matters: the index the two
factors share is the one that contracts, and the outer two survive.

Read the type exactly the way you would read `func W(x [n]float32) [m]float32`. It
consumes a vector with $n$ components and produces a vector with $m$ components.
The columns tell you what it eats; the rows tell you what it emits. When a
framework yells about a shape mismatch, it is a type error, and it is the single
most common bug you will hit when you write model code.

Concretely, with $x \in \mathbb{R}^{n}$, component $i$ of the output is

$$(Wx)_i = \sum_{j=1}^{n} W_{ij}\, x_j$$

where $W_{ij}$ is the entry in row $i$, column $j$, and $x_j$ is component $j$ of
the input. Row $i$ of $W$ is dotted with all of $x$ to make output component $i$.
Nothing more.

"Linear" is a real constraint, not decoration. It means exactly two things hold
for every input and every scalar $c$:

$$W(x + y) = Wx + Wy \qquad W(cx) = c\,(Wx)$$

Consequences you can use: $W\mathbf{0} = \mathbf{0}$ always (the origin cannot
move), and straight lines stay straight, and evenly spaced points stay evenly
spaced. A linear map can rotate, stretch, shear, reflect, and flatten space. It
cannot bend it. That single limitation is why neural networks need
nonlinearities at all, which is a debt this unit incurs and unit u4 pays.

### Three transformations you should be able to recognize by sight

Work in $\mathbb{R}^{2}$ so you can check every number by hand. Take the input
vector $v = \begin{bmatrix} 3 \\ 1 \end{bmatrix}$ throughout.

**Rotation by 90 degrees counterclockwise.**

$$R = \begin{bmatrix} 0 & -1 \\ 1 & 0\end{bmatrix}
\qquad Rv = \begin{bmatrix} 0\cdot 3 + (-1)\cdot 1 \\ 1 \cdot 3 + 0 \cdot 1\end{bmatrix}
= \begin{bmatrix} -1 \\ 3 \end{bmatrix}$$

Lengths are preserved: $\sqrt{3^2 + 1^2} = \sqrt{10}$ before, $\sqrt{(-1)^2 + 3^2}
= \sqrt{10}$ after. Rotations move information around without destroying any.

**Axis-aligned scaling.**

$$S = \begin{bmatrix} 2 & 0 \\ 0 & 0.5\end{bmatrix}
\qquad Sv = \begin{bmatrix} 6 \\ 0.5 \end{bmatrix}$$

The first coordinate got twice as loud, the second half as loud. This is what a
learned weight matrix mostly does: it decides which directions in its input
space matter and by how much.

**Projection onto the first axis.**

$$P = \begin{bmatrix} 1 & 0 \\ 0 & 0\end{bmatrix}
\qquad Pv = \begin{bmatrix} 3 \\ 0 \end{bmatrix}$$

This one is different in kind. It threw the second coordinate away, and no
matrix can bring it back. You can see the damage algebraically: $PP = P$. Applying
it twice does nothing more than applying it once, because there is nothing left
to remove. A matrix that maps a bigger space into a smaller one, or that is
$n \times n$ but has linearly dependent rows, destroys information permanently.
Every "down-projection" in a transformer is doing exactly this on purpose.

```beat
id: u2-b1
type: compute
concept: c-matrix-transform
prompt: |
  Compute by hand, no tool.

  $$A = \begin{bmatrix} 1 & 2 & 0 \\ 0 & 1 & 3 \end{bmatrix}
  \qquad x = \begin{bmatrix} 2 \\ 1 \\ 4 \end{bmatrix}$$

  $A$ is $2 \times 3$ and $x$ is a 3-vector, so $Ax$ is a 2-vector.
  Row $i$ of $A$ dotted with $x$ gives output component $i$.

  Give $Ax$ in the form `[a, b]`.
answer: "[4, 13]"
check: exact
```

## Composition is multiplication

Chain two transformations and you get a third one. That is the entire reason
matrix multiplication is defined the way it is, and it is the reason deep
networks can be written as one long product.

Apply $B$ first, then $A$:

$$A(Bx) = (AB)x$$

The product $AB$ is precomputed composition. If $A$ is $m \times k$ and $B$ is
$k \times n$, then $AB$ is $m \times n$, and the shared $k$ is the interface the
two functions agree on. Read the shapes as types again: $B$ maps
$\mathbb{R}^n \rightarrow \mathbb{R}^k$, $A$ maps
$\mathbb{R}^k \rightarrow \mathbb{R}^m$, so the composite maps
$\mathbb{R}^n \rightarrow \mathbb{R}^m$. The inner dimensions must match because
one function's output type must be the other's input type. That is the whole
rule.

The entry-level formula falls out of that requirement:

$$(AB)_{ij} = \sum_{t=1}^{k} A_{it} B_{tj}$$

where $t$ runs over the shared inner dimension of size $k$. Row $i$ of $A$
dotted with column $j$ of $B$.

### Order matters, and it matters visibly

<!-- refutes: U2-M3 -->

You probably think the order of matrix factors is a convention or an
optimization detail - matrix multiplication is associative, so ordering is
flexible. Here is the prediction that fails: under that belief, swapping two
same-shaped weight matrices in a forward pass would leave the outputs unchanged,
because nothing crashed and the same numbers went in. Watch it not happen.

The belief is appealing because associativity is real, is genuinely used for
large optimizations, and gets glossed as "order is flexible" in casual speech.
Shapes also often still line up after a swap, so nothing raises an error and "it
runs" gets read as "it is equivalent".

What is actually true: associativity lets you regroup parentheses,
$(AB)C = A(BC)$. Commutativity, which would let you swap factors, does not hold.
Function composition does not commute, and neither does matrix multiplication.
Take $R$, the 90-degree rotation, and $B = \begin{bmatrix} 2 & 0 \\ 0 & 1\end{bmatrix}$,
a stretch along the first axis only.

$$RB = \begin{bmatrix} 0 & -1 \\ 1 & 0\end{bmatrix}\begin{bmatrix} 2 & 0 \\ 0 & 1\end{bmatrix}
= \begin{bmatrix} 0 & -1 \\ 2 & 0\end{bmatrix}
\qquad
BR = \begin{bmatrix} 2 & 0 \\ 0 & 1\end{bmatrix}\begin{bmatrix} 0 & -1 \\ 1 & 0\end{bmatrix}
= \begin{bmatrix} 0 & -2 \\ 1 & 0\end{bmatrix}$$

Different matrices, and they act differently. Feed in
$e = \begin{bmatrix} 1 \\ 0\end{bmatrix}$:

$$(RB)e = \begin{bmatrix} 0 \\ 2\end{bmatrix}
\qquad (BR)e = \begin{bmatrix} 0 \\ 1\end{bmatrix}$$

Stretch-then-rotate lands twice as far out as rotate-then-stretch, because the
stretch only ever amplifies the first axis, and rotation moves the vector off
that axis before the stretch can act on it.

```beat
id: u2-b2
type: predict
concept: c-matmul
prompt: |
  You are debugging a model and you swap the order of two weight matrices in a
  forward pass, from $W_2 W_1 x$ to $W_1 W_2 x$. Both are square and the same
  size, so nothing crashes and the shapes still line up.

  Before reading on, answer two things. (1) Is the resulting function the same?
  (2) Is there any case where the swap is genuinely harmless?
answer: |
  (1) No. Matrix multiplication does not commute, so $W_2 W_1 \neq W_1 W_2$ in
  general and the network now computes a different function. Same shapes only
  means no type error, not same semantics.
  (2) Yes, in special cases. If both matrices are diagonal they commute. If one
  is a scalar multiple of the identity ($cI$) it commutes with everything.
  Identical matrices commute with themselves. These are measure-zero cases;
  learned weight matrices essentially never satisfy them.
rubric: |
  Must state that the function changes because matmul is not commutative, and
  must distinguish "shapes still match" from "semantics preserved". Pass without
  part (2). Fail if the learner says order does not matter, or reasons only about
  shape compatibility. Answering "it might be fine, matrices are just numbers"
  indicates U2-M3.
check: llm
```

## The dot product is the whole game

Given two vectors $a, b \in \mathbb{R}^{n}$, the dot product is one number:

$$a \cdot b = \sum_{i=1}^{n} a_i b_i$$

where $a_i$ and $b_i$ are the $i$-th components. It is the only way this book
ever compares two vectors, so it is worth knowing what the number means.

The geometric identity is the payoff:

$$a \cdot b = \lVert a \rVert \, \lVert b \rVert \cos\theta$$

where $\lVert a \rVert = \sqrt{\sum_i a_i^2}$ is the length of $a$ and $\theta$ is
the angle between the two vectors. So the dot product mixes two things: how long
the vectors are, and how aligned their directions are. Large and positive means
"pointing the same way"; zero means perpendicular; negative means opposed.

Numbers. Take $a = \begin{bmatrix} 3 \\ 4\end{bmatrix}$ and
$b = \begin{bmatrix} 4 \\ 3\end{bmatrix}$:

$$a \cdot b = 3\cdot 4 + 4 \cdot 3 = 24
\qquad \lVert a \rVert = \lVert b \rVert = \sqrt{9+16} = 5$$

$$\cos\theta = \frac{24}{5 \cdot 5} = 0.96 \quad\Rightarrow\quad \theta \approx 16.3^\circ$$

Nearly aligned, as you would expect from two vectors that are near-mirror images
across the diagonal. Now take $c = \begin{bmatrix} -4 \\ 3\end{bmatrix}$:

$$a \cdot c = 3\cdot(-4) + 4 \cdot 3 = 0$$

Perpendicular, and the dot product says so with a clean zero, without anyone
computing an angle.

There is a second reading that matters more for attention. The component of $a$
lying along the direction of $b$ has length

$$\frac{a \cdot b}{\lVert b \rVert} = \frac{24}{5} = 4.8$$

That is a projection: "how much of $a$ is $b$-flavored". When unit u3 computes
attention scores as $QK^{T}$, every entry of that matrix is one of these numbers,
asking how much of a given query is flavored like a given key. There is no
lookup, no matching, no search. There is a pile of dot products.

Note the connection back to the last two sections: $(Wx)_i$ is the dot product of
row $i$ of $W$ with $x$. So a matrix-vector product is a batch of similarity
measurements, one per row. A weight matrix is a stack of learned directions, and
applying it asks "how much does this input look like each of my directions".
That framing will carry you through the rest of the book.

```beat
id: u2-b3
type: compute
concept: c-dotprod
prompt: |
  A query vector $q = \begin{bmatrix} 1 \\ 2 \\ 1\end{bmatrix}$ is compared
  against three key vectors:

  $$k_1 = \begin{bmatrix} 2 \\ 0 \\ 3\end{bmatrix} \quad
  k_2 = \begin{bmatrix} 0 \\ 1 \\ 0\end{bmatrix} \quad
  k_3 = \begin{bmatrix} 1 \\ 1 \\ 1\end{bmatrix}$$

  Compute the three scores $q \cdot k_1$, $q \cdot k_2$, $q \cdot k_3$ by hand.

  Answer in the form `[s1, s2, s3]`.
answer: "[5, 2, 4]"
check: exact
```

## Softmax: why exponentials

<!-- refutes: M2 -->

You probably think softmax outputs are probabilities that each option is correct
- that when a model puts 0.87 on a token, it is 87 percent sure that token is the
right answer.

Here is the prediction that model makes and that fails. If those numbers were
correctness probabilities, then changing them would require new evidence. But
divide every logit by 2 before the softmax and the whole distribution reshapes,
with no new input, no new weights, and no new information anywhere in the system.
Nothing about the world got more or less true. A quantity that can be
arbitrarily reshaped by a runtime knob was never measuring correctness.

The belief is appealing because the outputs genuinely are non-negative and
genuinely do sum to 1, which is the definition of a probability distribution, and
every library and paper calls them probabilities. That is not a lie. It is
underspecified: a distribution over what, calibrated against what.

Here is what is actually true. Softmax is a differentiable soft-argmax. It takes
a vector of arbitrary real-valued scores and returns a normalized exponential
weighting of them. In an output layer, that weighting parameterizes a sampling
distribution over the vocabulary, calibrated to how often things followed each
other in training data. That is a frequency claim about text, not a truth claim
about the world.

### Deriving it from the requirements

Suppose you have scores $z = (z_1, \ldots, z_n)$, called logits, which are
the raw outputs of the last linear layer. They can be any real numbers, positive
or negative. You want to turn them into weights $p_i$ that you can sample from.
Write down what you need:

1. **Non-negative.** A negative weight is meaningless for sampling.
2. **Sums to 1.** So it is a distribution.
3. **Monotone.** A bigger score must never produce a smaller weight.
4. **Differentiable everywhere.** You will need to push gradients through it
   during training. Hard $\arg\max$ has zero gradient almost everywhere and is
   therefore useless for learning.
5. **Only differences should matter.** Adding a constant to every score should
   change nothing, because logits have no absolute zero. This one is the
   requirement people skip, and it is the one that forces the answer.

<!-- refutes: U2-M1 -->

Try the obvious cheap options and watch them break.

*Divide by the sum, $p_i = z_i / \sum_j z_j$.* Fails requirement 1 immediately:
logits are routinely negative, and the sum can be zero, which divides by zero.
Dead.

*Square first, $p_i = z_i^2 / \sum_j z_j^2$.* Non-negative, sums to 1. But it
fails monotonicity in the worst possible way. Take $z = (2, 1, 0)$: you get
$(0.8, 0.2, 0)$. Now take $z = (-2, 1, 0)$: you get $(0.8, 0.2, 0)$ again. The
option the network scored *lowest* was handed the *most* weight. Squaring
destroys sign, which is most of the information in a logit.

*Absolute value, $p_i = |z_i| / \sum_j |z_j|$.* Same disease, plus it is not
differentiable at zero.

Now impose requirement 5 seriously. You want a function $f$ with

$$\frac{p_i}{p_j} = \frac{f(z_i)}{f(z_j)} \quad \text{depending only on } z_i - z_j$$

That means $f(z_i)/f(z_j) = g(z_i - z_j)$ for some $g$, which is exactly the
functional equation whose only continuous positive solutions are exponentials,
$f(z) = e^{\beta z}$. Turning differences into ratios is what exponentials do,
and they are the only thing that does it. So:

$$p_i = \mathrm{softmax}(z)_i = \frac{e^{z_i}}{\sum_{j=1}^{n} e^{z_j}}$$

Non-negative because $e^{z} > 0$ for every real $z$, including very negative
ones. Sums to 1 by construction. Monotone because $e^z$ is increasing. Smooth
everywhere. And shift-invariant, which you can check directly.

### Worked example

Take $z = (2, 1, 0)$. Exponentiate each entry:

$$e^{2} = 7.389 \qquad e^{1} = 2.718 \qquad e^{0} = 1.000$$

Sum: $7.389 + 2.718 + 1.000 = 11.107$. Divide:

$$p = \left(\frac{7.389}{11.107},\ \frac{2.718}{11.107},\ \frac{1.000}{11.107}\right)
= (0.665,\ 0.245,\ 0.090)$$

Two checks worth doing by hand. First, the ratio of the top two:
$0.665 / 0.245 = 2.718 = e^{1}$, exactly $e$ raised to the logit gap of $2 - 1 = 1$.
The gap sets the ratio, and nothing else does.

Second, shift every logit up by 5, to $z = (7, 6, 5)$. The exponentials all get
multiplied by $e^5$, numerator and denominator alike, and it cancels: you get
$(0.665, 0.245, 0.090)$ again. This is not a curiosity. It is why every real
implementation subtracts $\max_j z_j$ from the logits before exponentiating - it
is free (the output is provably unchanged) and it prevents $e^{z}$ from
overflowing on large logits.

```beat
id: u2-b4
type: self-explain
concept: c-softmax
prompt: |
  We rejected "square the logits and normalize" using a specific counterexample:
  $z = (2, 1, 0)$ and $z = (-2, 1, 0)$ both produce $(0.8, 0.2, 0)$.

  In your own words, explain what property of the exponential makes it survive
  that test, and say what would go wrong during training if we used squaring
  anyway. Two or three sentences.
answer: |
  $e^{z}$ is strictly increasing over the entire real line and strictly positive,
  so a lower score always yields a strictly lower weight and sign information is
  preserved. Squaring is not monotone on the reals - it folds negatives onto
  positives - so the mapping from score to weight is not order-preserving and is
  many-to-one. During training the gradient would push a logit in a direction
  that reduces its own weight whenever the logit is negative, so the optimizer
  gets a signal pointing the wrong way, and the network cannot express
  "this option is strongly disfavored" at all.
rubric: |
  Must contain: (1) $e^z$ is monotone increasing and positive over all reals,
  therefore order-preserving; (2) squaring folds sign / is not monotone on
  negatives, so the score-to-weight map is not order-preserving. Both required to
  pass. Full credit adds a training consequence (wrong-direction gradient, or
  inability to express strong disfavor).
  Answering only "exp makes things positive" is partial credit - normalization of
  absolute values also makes things positive, so positivity alone is not the
  discriminating property; probe for monotonicity.
  Saying exp is used "to avoid overflow" or "for numerical stability" is
  backwards - exp causes the overflow that the max-subtraction trick fixes - and
  indicates M7-style numerics-first reasoning.
  Saying softmax uses exp "because the outputs are probabilities and probabilities
  are exponential" indicates M2.
check: llm
```

```beat
id: u2-b5
type: compute
concept: c-softmax
prompt: |
  Compute $\mathrm{softmax}(z)$ for $z = (3, 1, 0)$ by hand.

  Useful values: $e^{3} = 20.086$, $e^{1} = 2.718$, $e^{0} = 1.000$.

  Report only the largest of the three probabilities, to three decimal places.
answer: 0.844
check: numeric(0.005)
```

## Temperature: one knob, same evidence

<!-- refutes: M2 -->

Temperature is a single scalar $T > 0$ that divides the logits before the
softmax:

$$p_i(T) = \frac{e^{z_i / T}}{\sum_{j} e^{z_j / T}}$$

Nothing else changes. The logits $z$ are fixed - they are the model's output and
temperature is applied downstream of them, at sampling time, often in the serving
layer rather than the model at all.

Take the same $z = (2, 1, 0)$ from the last section and run three temperatures.

**$T = 0.5$ (sharpen).** Scaled logits $z/T = (4, 2, 0)$:

$$e^{4} = 54.598 \quad e^{2} = 7.389 \quad e^{0} = 1.000
\qquad \text{sum} = 62.987$$

$$p = (0.867,\ 0.117,\ 0.016)$$

**$T = 1$ (identity).** Scaled logits $(2, 1, 0)$, sum of exponentials 11.107:

$$p = (0.665,\ 0.245,\ 0.090)$$

**$T = 2$ (flatten).** Scaled logits $z/T = (1, 0.5, 0)$:

$$e^{1} = 2.718 \quad e^{0.5} = 1.649 \quad e^{0} = 1.000
\qquad \text{sum} = 5.367$$

$$p = (0.506,\ 0.307,\ 0.186)$$

Read the middle option across the three rows: 0.117, 0.245, 0.307. Its
probability more than doubled, a factor of 2.6 from end to end. The gap between
it and the winner went from a factor of 7.4 down to a factor of 1.6. And the
model's logits were byte-identical in all three cases.

Two limits are worth holding. As $T \rightarrow 0$, the scaled gaps blow up and
all the mass collapses onto the top logit: softmax becomes hard $\arg\max$. As
$T \rightarrow \infty$, the scaled logits all approach 0, every exponential
approaches 1, and you get the uniform distribution regardless of what the model
said. Note what is invariant across every $T$: the *ranking* never changes,
because dividing by a positive number preserves order. Temperature moves mass; it
cannot reorder preferences.

This is the concrete kill shot for the "these are correctness probabilities"
model. A correctness probability that a config flag can move from 0.117 to 0.307,
with the model's computation bit-identical throughout, is not measuring
correctness. It is a knob on a sampler. Unit u6 uses this to dismantle the
related and even more common belief that temperature changes what the model
knows.

```beat
id: u2-b6
type: compute
concept: c-softmax
prompt: |
  Logits $z = (2, 0)$, temperature $T = 0.5$.

  Scale first, then exponentiate. Useful values: $e^{4} = 54.598$, $e^{0} = 1.000$.

  What probability does the first option get? Three decimal places.
answer: 0.982
check: numeric(0.005)
```

```beat
id: u2-b7
type: predict
concept: c-softmax
prompt: |
  A colleague reports: "I set temperature to 0 and the model gave a wrong answer
  confidently, so at least now I know what it really believes. At temperature 1 it
  was hedging."

  Before reading on: what is wrong with both halves of that claim? Be specific
  about which quantity changed and which did not.
answer: |
  Neither half holds. The logits are identical at $T = 0$ and $T = 1$ - the model
  ran the same forward pass and produced the same numbers. Temperature is applied
  after the logits, so it changes only how mass is distributed for sampling.
  $T = 0$ is argmax over the same distribution, so it does not reveal a hidden
  belief; it reveals the top-ranked option, which is already visible in the
  $T = 1$ distribution as the largest probability. And "hedging" at $T = 1$ is not
  uncertainty about correctness - a spread-out distribution reflects that many
  continuations were frequent in training data after similar context. Confidence
  displayed at any temperature is a frequency-shaped weighting, not a calibrated
  probability of being right.
rubric: |
  Must identify that logits are unchanged across temperatures and that only the
  sampling distribution shape changes (M6/M2 core). Must also reject the
  "hedging = uncertainty about truth" reading, or at minimum state that softmax
  mass is not a correctness probability.
  Getting only the first point = partial credit.
  Accepting that $T = 0$ reveals true belief indicates M6.
  Accepting that the $T = 1$ spread measures the model's uncertainty about
  correctness indicates M2.
check: llm
```

## From derivative to gradient

Training is a search over a few hundred billion numbers. The only reason that is
tractable is that at any point you can cheaply compute which way is downhill.
Three definitions get you there, and they nest.

**Derivative (one input).** For $f : \mathbb{R} \rightarrow \mathbb{R}$,

$$f'(x) = \lim_{h \to 0} \frac{f(x+h) - f(x)}{h}$$

Operationally: nudge the input by a tiny $h$, see how much the output moved,
divide. It is a sensitivity, in output units per input unit. If $f(x) = x^2$ then
$f'(x) = 2x$, so at $x = 3$ the derivative is 6, meaning a nudge of $+0.01$ in
$x$ buys about $+0.06$ in $f$. Check it: $f(3.01) = 9.0601$ versus $f(3) = 9$, a
change of 0.0601. The derivative is the local linear approximation, and it is
accurate to the extent that you stay local.

**Partial derivative (many inputs, one at a time).** For
$f : \mathbb{R}^{n} \rightarrow \mathbb{R}$, the partial derivative
$\partial f / \partial x_i$ is the ordinary derivative you get by nudging $x_i$
alone and holding every other input frozen. The symbol $\partial$ (curly d) is
purely a reminder that other variables exist and are being pinned.

**Gradient (all of them at once).** Stack the partials into a vector:

$$\nabla f(x) = \left[\frac{\partial f}{\partial x_1},\ \frac{\partial f}{\partial x_2},\ \ldots,\ \frac{\partial f}{\partial x_n}\right]$$

$\nabla$ is pronounced "del" or "nabla". The gradient has exactly the same shape
as the input. That is the single most useful fact about it for reading model
code: if your parameters are a $4096 \times 4096$ matrix, the gradient with
respect to them is a $4096 \times 4096$ matrix, entry for entry. Gradients are
shaped like the thing you are differentiating with respect to, never like the
loss.

### What the gradient means geometrically

Two properties, both load-bearing:

- $\nabla f(x)$ points in the direction of **steepest local increase** of $f$ at
  the point $x$.
- Its length $\lVert \nabla f(x)\rVert$ is the **rate** of increase in that
  direction.

So $-\nabla f(x)$ is the steepest descent direction, which is why the update rule
is

$$x \leftarrow x - \eta \, \nabla f(x)$$

where $\eta$ (eta) is the learning rate, a small positive scalar controlling step
size.

<!-- refutes: U2-M2 -->

Be precise about "steepest local increase", because the natural reading is
wrong. You probably think the gradient points toward the minimum, so the
negative gradient is the direction to travel to get there. Here is the
prediction that fails: if it pointed at the minimum, one appropriately sized
step would arrive, and descent trajectories would be straight lines. They are
not - descent visibly zigzags, and the zigzag gets worse the more elongated the
surface is.

The belief is appealing because it is true on a simple bowl, which is the
picture school calculus leaves you with, and because "steepest descent" sounds
like it means "toward the bottom".

What is actually true: the gradient is computed entirely from the surface at the
current point and carries no information about where any minimum is. It points
the way the surface tilts *right here*, and one step later it will point
somewhere else. On a long narrow valley, the gradient points mostly across the
valley walls rather than along the floor toward the bottom, which is why plain
gradient descent zigzags and why unit u5 needs momentum and Adam.

### Worked example

Let $f(w_1, w_2) = w_1^2 + 3 w_1 w_2 + 2 w_2^2$, a scalar function of two
parameters. Differentiate each variable in turn, holding the other fixed:

$$\frac{\partial f}{\partial w_1} = 2 w_1 + 3 w_2
\qquad \frac{\partial f}{\partial w_2} = 3 w_1 + 4 w_2$$

At the point $(w_1, w_2) = (1, 2)$:

$$f(1,2) = 1 + 6 + 8 = 15
\qquad \nabla f = \begin{bmatrix} 2 + 6 \\ 3 + 8\end{bmatrix} = \begin{bmatrix} 8 \\ 11\end{bmatrix}$$

Take one descent step with $\eta = 0.01$:

$$w \leftarrow \begin{bmatrix} 1 \\ 2\end{bmatrix} - 0.01 \begin{bmatrix} 8 \\ 11\end{bmatrix}
= \begin{bmatrix} 0.92 \\ 1.89 \end{bmatrix}$$

$$f(0.92, 1.89) = 0.8464 + 5.2164 + 7.1442 = 13.207$$

Down from 15. That is the entire algorithm. Everything unit u5 adds - momentum,
per-parameter scaling, weight decay - is refinement on top of this line.

```beat
id: u2-b8
type: compute
concept: c-gradient
prompt: |
  Let $f(a, b) = 2a^2 + ab + 3b^2$.

  Compute $\nabla f$ at the point $(a, b) = (2, -1)$. Differentiate with respect
  to $a$ holding $b$ fixed, then with respect to $b$ holding $a$ fixed.

  Answer in the form `[df/da, df/db]`.
answer: "[7, -4]"
check: exact
```

## The chain rule

<!-- fade: chain-rule -->

Networks are compositions. A loss depends on an output, which depends on a
hidden value, which depends on a weight. To improve the weight you need
$\partial L / \partial w$, but $L$ is not written in terms of $w$ directly. The
chain rule is how you get it: sensitivities multiply along the path.

For $L$ depending on $h$ depending on $z$ depending on $w$:

$$\frac{\partial L}{\partial w} = \frac{\partial L}{\partial h} \cdot \frac{\partial h}{\partial z} \cdot \frac{\partial z}{\partial w}$$

The 1D intuition is exact and worth trusting. If $h$ moves 3 units for every unit
of $z$, and $L$ moves 2 units for every unit of $h$, then $L$ moves 6 units for
every unit of $z$. Sensitivities compose by multiplication, the same way gear
ratios do. Scaling this up to matrices changes only the type of each factor -
each becomes a Jacobian matrix instead of a scalar, and the multiplication
becomes matrix multiplication in the correct order - but the structure is
identical. That upgrade is what backprop is, and unit u5 does it.

### Fully worked

A one-parameter network with a nonlinearity, small enough to check every digit:

$$z = wx + b \qquad h = z^2 \qquad L = (h - y)^2$$

Read it as: a linear layer produces $z$ from input $x$ with weight $w$ and bias
$b$; a nonlinearity squares it to produce activation $h$; a squared-error loss
compares $h$ against target $y$. Squaring stands in for a real activation
function here because its derivative is exact arithmetic.

Fix $x = 2$, $w = 1.5$, $b = -1$, $y = 5$.

**Forward pass.** Compute left to right and keep every intermediate value,
because the backward pass needs them:

$$z = 1.5 \cdot 2 + (-1) = 2 \qquad h = 2^2 = 4 \qquad L = (4 - 5)^2 = 1$$

**Local derivatives.** Differentiate each step with respect to its own input:

$$\frac{\partial L}{\partial h} = 2(h - y) = 2(4 - 5) = -2$$

$$\frac{\partial h}{\partial z} = 2z = 2 \cdot 2 = 4$$

$$\frac{\partial z}{\partial w} = x = 2 \qquad \frac{\partial z}{\partial b} = 1$$

**Multiply along the path.**

$$\frac{\partial L}{\partial w} = (-2)(4)(2) = -16
\qquad \frac{\partial L}{\partial b} = (-2)(4)(1) = -8$$

**Sanity check the sign before trusting the number.** $\partial L/\partial w$ is
negative, so increasing $w$ decreases the loss. Does that make sense? $h = 4$ and
the target is $y = 5$, so we want $h$ bigger, so we want $z$ bigger, so we want
$w$ bigger. It agrees. Sign-checking gradients against a physical story catches
most transcription errors, and it costs ten seconds.

**Take the step**, with $\eta = 0.005$:

$$w \leftarrow 1.5 - 0.005(-16) = 1.58 \qquad b \leftarrow -1 - 0.005(-8) = -0.96$$

$$z = 1.58 \cdot 2 - 0.96 = 2.2 \qquad h = 4.84 \qquad L = (4.84 - 5)^2 = 0.0256$$

Loss fell from 1 to 0.0256. Two things to notice. The bias got a smaller-magnitude
gradient than the weight ($-8$ versus $-16$) purely because $\partial z/\partial b
= 1$ while $\partial z/\partial w = x = 2$: an input that is large in magnitude
produces a large gradient on the weight it multiplies, which is most of why input
normalization matters. And $\partial L/\partial h = -2$ was computed once and
reused for both parameters. That reuse, done systematically over a whole graph,
is backprop.

```beat
id: u2-b9
type: completion
concept: c-gradient
prompt: |
  Same network, new numbers. Fill in the four blanks.

  $$z = wx + b \qquad h = z^2 \qquad L = (h - y)^2$$

  with $x = 3$, $w = 1$, $b = -1$, $y = 5$.

  Forward pass:
  $z = 1 \cdot 3 + (-1) = 2$
  $h = 2^2 = 4$
  $L = (4 - 5)^2 = 1$

  Local derivatives:
  $\dfrac{\partial L}{\partial h} = 2(h - y) = $ ____
  $\dfrac{\partial h}{\partial z} = 2z = $ ____
  $\dfrac{\partial z}{\partial w} = x = 3$

  Chain them:
  $\dfrac{\partial L}{\partial w} = $ ____

  And state in one sentence why $\dfrac{\partial L}{\partial w}$ here has a larger
  magnitude than the $-16$ we got in the worked example, even though the loss is
  the same value of 1. ____
answer: |
  $\partial L / \partial h = 2(4 - 5) = -2$
  $\partial h / \partial z = 2 \cdot 2 = 4$
  $\partial L / \partial w = (-2)(4)(3) = -24$
  Larger magnitude because $\partial z / \partial w = x$, and the input is $x = 3$
  here versus $x = 2$ in the worked example; the two upstream factors are
  identical, so the gradient scales directly with the input magnitude.
rubric: |
  All three numeric blanks must be exactly $-2$, $4$, and $-24$. A sign error on
  $\partial L/\partial h$ (giving $+2$ and $+24$) is a fail - the sign is the part
  that determines the update direction.
  The explanation must attribute the difference to $\partial z/\partial w = x$
  being larger, not to the loss, the weight, or the bias. Attributing it to the
  loss value indicates the learner is treating the gradient as a function of the
  loss magnitude alone rather than of the path.
  Getting all three numbers but flubbing the explanation = partial credit.
check: llm
# fade: chain-rule, stage 2 of 3. Blanked steps are the two local derivatives and
# the product, which carry the concept. Variant A: blank the forward pass instead
# and give the derivatives (tests forward/backward separation). Variant B: blank
# only the product and ask for dL/db as well (tests path reuse of dL/dh).
```

## Loss surfaces and why SGD works anyway

<!-- refutes: M5 -->

You probably think gradient descent finds the minimum of the loss - that
training is an optimization problem with a right answer, and better training gets
you closer to it.

Here is the prediction that fails. If training converged on the global minimum,
two runs of the same architecture on the same data with different random seeds
would end up at the same weights, or at least at weights you could match up. They
never do. The weights are wildly different, they are not even close under any
permutation you care to apply, and both models work about equally well. That is
not a near miss. It is evidence that "the minimum" is not what is being found.

The belief is appealing because your school math was about convex functions -
parabolas, single bottoms - where gradient descent provably converges to the one
minimum. It is also appealing because the language of "optimization" implies an
optimum. Both are honest sources of the error.

Here is what is actually true. The loss surface is a function
$L : \mathbb{R}^{d} \rightarrow \mathbb{R}$ where $d$ is the parameter count,
routinely $10^{9}$ to $10^{12}$. SGD finds *one of* an astronomically large
number of low-loss regions, and which one it finds depends on the seed, the data
order, and the hardware nondeterminism. Nearly all of them are fine.

### Why high dimensions are kind rather than cruel

<!-- refutes: U2-M4 -->

The intuition that more dimensions means more local minima to get trapped in is
exactly backwards, and the reason is worth carrying.

At a critical point (where $\nabla L = 0$), whether you are at a minimum depends
on the curvature in every direction. Curvature in direction $v$ is positive if
the surface bends upward along $v$, negative if it bends downward. For the point
to be a local minimum, the surface must curve upward in **all $d$ directions at
once**. If even one direction curves down, that direction is an escape route, and
the point is a saddle rather than a trap.

The smallest saddle: $f(x,y) = x^2 - y^2$ at the origin. The gradient is
$(2x, -2y) = (0,0)$ there, so descent stalls if you land exactly on it. But it is
a minimum along $x$ and a maximum along $y$. Step anywhere off the $x$-axis and
you fall away. In two dimensions this is a coin flip. In $d$ dimensions you need
$d$ coin flips to all come up "curves upward", and $d$ is a billion. Local minima
are not the enemy; they are vanishingly rare above the very bottom of the loss
range. Saddles and flat regions dominate, and gradient noise from minibatching
walks you off them.

The second fact is that the good regions are wide, flat basins rather than sharp
pits, and wide basins are exactly the ones where small weight perturbations do
not change the function much, which is a large part of why the result
generalizes. So "found a different low-loss region than last time" is not a
defect to be engineered away. It is the normal case.

```beat
id: u2-b10
type: self-explain
concept: c-loss-surface
prompt: |
  Two engineers train the same architecture on the same dataset, changing only the
  random seed. The final weight tensors are nothing alike, but both models score
  within noise of each other on every benchmark.

  Explain in three or four sentences why this is expected rather than a bug, using
  the dimensionality argument. Then say what you *would* conclude if the two runs
  produced identical weights.
answer: |
  The loss surface lives in a space with billions of dimensions and has an
  enormous number of distinct low-loss regions, so the seed - which sets weight
  initialization and data order - selects which one the trajectory lands in.
  There is no reason for two trajectories starting from different points to arrive
  at the same place, and no need for them to. Matching benchmark scores show the
  regions are of comparable quality, which is the property that matters; the
  identity of the weights is not. Additionally, permutation symmetry means many
  weight configurations compute the identical function, so even functionally
  identical models need not have matching tensors.
  Identical weights from different seeds would mean the run is not actually
  seeded - some source of randomness is pinned or the initialization is
  deterministic - which is a bug in the experiment, not a triumph of optimization.
rubric: |
  Must contain: (1) many distinct low-loss regions exist, and the seed selects
  among them; (2) equivalent benchmark performance is the criterion, not weight
  identity. Both required to pass. Credit for mentioning permutation symmetry or
  wide-basin generalization as a bonus, not a requirement.
  Answering that one run found the global minimum and the other found a local
  minimum indicates M5 persists - the learner has kept the "one true minimum"
  frame and merely relabeled the second run as a failure. This is the most common
  failure mode; fail it explicitly.
  Answering that high dimension means more local minima to get trapped in
  indicates U2-M4.
  Failing to flag identical weights as a seeding bug = partial credit.
check: llm
```

### The floor is not zero

<!-- refutes: M18 -->

One more thing before you leave this surface, because it changes what "training
went well" means. You probably think the target is zero loss and that whatever
loss remains is a deficiency to be engineered away.

The prediction that fails: if a language model reached zero loss, it would be
assigning probability 1.0 to exactly one continuation of every context. Pick any
real context - "the meeting is scheduled for" - and there are dozens of correct
continuations. A model that puts all its mass on one of them is wrong about
language, not right about it.

The correct frame: the loss has an irreducible floor set by the actual entropy of
the data. If the next symbol is a genuinely fair coin flip, the best achievable
cross-entropy is $\ln 2 \approx 0.693$ nats, and no amount of scale, data, or
compute goes below it. The gap that tells you how a model is doing is the gap to
that floor, not the gap to zero. A training loss approaching zero is a symptom,
and the diagnosis is memorization. Unit u5 makes this quantitative when it
defines cross-entropy properly and connects it to scaling laws; for now, carry
the reframe: low loss is good, zero loss is broken.

---

You now hold the whole toolkit. A matrix is a typed function that reshapes a
space. Composing them is multiplying them, and order matters. Every comparison
this book makes between two vectors is a dot product. Softmax turns scores into a
sampling distribution by exponentiating, which is forced by requiring that only
score differences matter, and temperature rescales that distribution without
touching the scores. The gradient is the vector of partials, shaped like the
parameters, pointing locally uphill. The chain rule multiplies sensitivities
along a path. And descent finds a good region, not the minimum, on a surface
whose floor is above zero.

Unit u3 spends all of it at once.
