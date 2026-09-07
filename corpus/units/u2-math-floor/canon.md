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
way. It is the opposite of the row-vector form $XW$ that u1 established and that
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

  Before reading on, commit: is the resulting function the same, and is there any
  case where the swap is genuinely harmless?
options:
  - text: |
      No - the network now computes a different function. Matrix multiplication
      does not commute, so $W_2 W_1 \neq W_1 W_2$ in general, and matching shapes
      only mean there is no type error. The swap is harmless in special cases -
      both matrices diagonal, one a scalar multiple of the identity $cI$, or the
      two matrices equal - which learned weights essentially never satisfy.
    correct: true
    explain: |
      Right. Composition order is semantics, not convention. With the rotation $R$
      and the stretch $B = \begin{bmatrix} 2 & 0 \\ 0 & 1\end{bmatrix}$,
      $(RB)e = \begin{bmatrix} 0 \\ 2\end{bmatrix}$ while
      $(BR)e = \begin{bmatrix} 0 \\ 1\end{bmatrix}$ for $e = \begin{bmatrix} 1 \\ 0\end{bmatrix}$.
      The commuting cases are measure-zero.
  - text: |
      Yes, the function is unchanged. Matrix multiplication is associative, so the
      ordering of the factors is a convention; only how you group the parentheses
      is a real choice, and that is an optimization question.
    misconception: U2-M3
    explain: |
      Associativity lets you regroup - $(AB)C = A(BC)$, which is why LoRA computes
      $B(Ax)$ rather than $(BA)x$. It does not let you swap factors. Commutativity
      would, and $AB = BA$ only when the matrices share a full set of eigenvectors:
      $RB = \begin{bmatrix} 0 & -1 \\ 2 & 0\end{bmatrix}$ against
      $BR = \begin{bmatrix} 0 & -2 \\ 1 & 0\end{bmatrix}$.
  - text: |
      The computation is equivalent because nothing errored. Both matrices are
      square and the same size, so every shape still lines up end to end; if the
      swap changed the semantics the framework would have raised.
    misconception: U2-M6
    explain: |
      Shape is the interface, not the implementation. Swapping two same-shaped
      operands runs cleanly and silently computes a different function - the model
      simply trains to a worse loss with no error anywhere. Shape checking rules
      out one specific class of wiring error and is the first check, never the last.
check: choice
```

## A matrix multiply by hand

<!-- canon-only -->

<!-- refutes: U0-M1 -->
You probably think matrix multiplication is a triple-nested loop whose
inner-dimensions-must-match rule is bookkeeping - an implementation detail of
how the numbers happen to be stored. Here is the prediction that fails: under
that belief the rule is arbitrary enough that another pairing (multiplying
aligned entries, say, or matching outer dimensions) would be an equally valid
definition. It would not. Matrices are functions. $A$ of shape $(p, q)$ is a
function from $p$-dimensional space to $q$-dimensional space in row convention,
and $AB$ is the **composition** of two such functions. The shape rule is nothing
but the requirement that one function's output type matches the next one's input
type. It is a type check, not a storage detail - which is also why
$AB \neq BA$: composing in the other order is a different function, and often
not a well-typed one.

The belief is appealing because the loop is what the hardware runs, and
reasoning at the memory-layout level is usually the productive instinct. That
level is real. It is not where the meaning is.

What is actually true: every entry of the product is a dot product. Entry
$(i, j)$ of $AB$ is row $i$ of $A$ dotted with column $j$ of $B$:

$$(AB)_{ij} = \sum_{t} A_{it} B_{tj}$$

where $t$ runs over the shared inner dimension, $A_{it}$ is the entry of $A$ in
row $i$ and column $t$, and $B_{tj}$ is the entry of $B$ in row $t$ and column
$j$. A matrix multiply is a grid of dot products, one per (row of $A$, column of
$B$) pair.

<!-- fade: matrix-multiply -->
Worked, with $A$ of shape $(2, 3)$ and $B$ of shape $(3, 2)$:

$$A = \begin{bmatrix} 1 & 0 & 2 \\ 3 & 1 & -1 \end{bmatrix}, \quad
B = \begin{bmatrix} 4 & 1 \\ 0 & 2 \\ -1 & 5 \end{bmatrix}$$

The inner dimensions are both $3$, so the product is defined and has shape
$(2, 2)$ - the surviving outer dimensions. Four entries, four dot products:

$$(AB)_{11} = (1)(4) + (0)(0) + (2)(-1) = 4 + 0 - 2 = 2$$
$$(AB)_{12} = (1)(1) + (0)(2) + (2)(5) = 1 + 0 + 10 = 11$$
$$(AB)_{21} = (3)(4) + (1)(0) + (-1)(-1) = 12 + 0 + 1 = 13$$
$$(AB)_{22} = (3)(1) + (1)(2) + (-1)(5) = 3 + 2 - 5 = 0$$

$$AB = \begin{bmatrix} 2 & 11 \\ 13 & 0 \end{bmatrix}$$

$BA$ is also defined here - $(3,2)$ times $(2,3)$ gives $(3,3)$ - and is a
different object of a different size. Same two matrices, different composition,
different function.

```beat
id: u0-b4
type: completion
concept: c-matmul
# variants: blank (AB)_11 and (AB)_21 instead, which tests column-of-B
#           selection rather than row-of-A selection.
prompt: |
  Fill the blanks. $A = \begin{bmatrix} 2 & 1 & 0 \\ 1 & 0 & 3 \end{bmatrix}$
  with shape $(2,3)$, $B = \begin{bmatrix} 1 & 2 \\ 4 & 0 \\ 1 & 1 \end{bmatrix}$
  with shape $(3,2)$.

      (AB)_11 = (2)(1) + (1)(4) + (0)(1) = 6
      (AB)_12 = (2)(2) + (1)(0) + (0)(1) = ____      <- A
      (AB)_21 = (1)(1) + (0)(4) + (3)(1) = 4
      (AB)_22 = (1)(2) + (0)(0) + (3)(1) = ____      <- B

  Answer with the two values in order, comma-separated, like `7, 9`.
answer: "4, 5"
check: exact

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

<!-- fade: dot-product -->
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

## What the dot product is not

<!-- canon-only -->

Two readings of that one number come pre-installed from tooling you use
every day, and both fail on the same three vectors.

<!-- refutes: U0-M2 -->
You probably read that as a similarity score - a bigger dot product means "more
similar", the way cosine similarity does in every vector database you have used.
Here is the prediction that fails: under that belief, scaling a vector without
rotating it cannot change how similar it is to anything, since its direction is
unchanged. Take $q = [3, 0]$ and two keys pointing in identical directions,
$k_1 = [0.6, 0.8]$ and $k_2 = [6, 8]$. Cosine similarity is $0.6$ for both. But
$q \cdot k_1 = 1.8$ and $q \cdot k_2 = 18$ - ten times the score for zero change
in direction.

The belief is appealing because your tooling normalizes for you: embedding
databases store unit-length vectors, and on unit vectors the dot product and the
cosine are literally the same number. That is a property of the normalization,
not of the dot product.

What is actually true: the dot product is cosine similarity multiplied by both
magnitudes, so it conflates "points the same way" with "is large". Nothing
inside a transformer normalizes before the dot product, which matters twice - it
is why attention scores need a $1/\sqrt{d_k}$ correction (u3), and why a single
high-magnitude key can dominate an attention distribution regardless of
direction.

```beat
id: u0-b2
type: predict
concept: c-dotprod
prompt: |
  Same three vectors: $q = [3, 0]$, $k_1 = [0.6, 0.8]$, $k_2 = [6, 8]$, with
  $k_1$ and $k_2$ pointing in identical directions.

  Before reading on, predict: which of $k_1$, $k_2$ is physically *closer* to
  $q$ in the plane, and does the ranking by distance agree with the ranking by
  dot product? Commit before you compute anything.
options:
  - text: |
      $k_1$ is much closer - $\lVert q - k_1 \rVert \approx 2.53$ against
      $\lVert q - k_2 \rVert \approx 8.54$ - and the two rankings disagree
      completely: the nearer vector scores $1.8$ while the one over three times
      farther away scores $18$.
    correct: true
    explain: |
      Right. The dot product is not a distance and is not even a decreasing
      function of distance. The two quantities are related by
      $\lVert u - v \rVert^2 = \lVert u \rVert^2 - 2(u \cdot v) + \lVert v \rVert^2$,
      which ranks the same way only when all the norms are equal.
  - text: |
      $k_2$ is closer. It scores $18$ against $1.8$, and a larger dot product means
      the vectors sit nearer one another, so the two rankings have to agree.
    misconception: U0-M6
    explain: |
      Measure it instead: $\lVert q - k_1 \rVert \approx 2.53$ and
      $\lVert q - k_2 \rVert \approx 8.54$. The higher-scoring key is over three
      times farther away, so the score is rising as the distance grows - the exact
      opposite of a proximity measure.
  - text: |
      $k_1$ is closer, but the two keys score identically: they point in exactly
      the same direction, and the dot product measures how aligned two vectors are,
      so scaling one of them cannot change its score.
    misconception: U0-M2
    explain: |
      That describes cosine similarity, which is $0.6$ for both. The dot product is
      the cosine multiplied by both magnitudes, so $q \cdot k_1 = 1.8$ and
      $q \cdot k_2 = 18$ - ten times the score for zero change in direction. Your
      vector database normalizes first; a transformer does not.
check: choice
```

<!-- refutes: U0-M6 -->
The neighboring instinct, that a bigger dot product means "closer", fails on the
same example and harder. Proximity would mean the score shrinks as vectors move
apart. Measure it: $\lVert q - k_1 \rVert \approx 2.53$ while
$\lVert q - k_2 \rVert \approx 8.54$, so $k_2$ is over three times farther away
and scores ten times higher. The two quantities are related by
$\lVert u - v \rVert^2 = \lVert u \rVert^2 - 2(u \cdot v) + \lVert v \rVert^2$,
which agrees on ranking only when all norms are equal - the unit-sphere case
your vector database quietly enforces and a transformer does not.

```beat
id: u0-b3
type: completion
concept: c-dotprod
# variants: blank lines 1 and 3 instead of 2 and 3; or blank all three products
#           and keep the sum, which tests componentwise pairing rather than
#           sign handling and summation.
prompt: |
  Fill the blanks. $q = [4, -2, 1]$, $k = [3, 5, 2]$.

      (4)(3)   = 12
      (-2)(5)  = ____      <- A
      (1)(2)   = 2
      q . k    = ____      <- B

  Answer with the two values in order, comma-separated, like `7, 9`.
answer: "-10, 4"
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

  Which explanation correctly says what property of the exponential survives that
  test, and what would go wrong during training if we used squaring anyway?
options:
  - text: |
      $e^{z}$ is strictly increasing and strictly positive over the entire real
      line, so a lower score always yields a strictly lower weight and the sign of
      a logit is preserved. Squaring folds negatives onto positives, so its
      score-to-weight map is not order-preserving and is many-to-one; inside a
      softmax it would hand the optimizer a gradient that raises an option's weight
      as its negative logit falls, and the network could never express "strongly
      disfavored" at all.
    correct: true
    explain: |
      Right. Monotonicity over all of $\mathbb{R}$, not positivity, is what kills
      squaring - and requirement 5, that only differences matter, is what then
      forces the exponential specifically.
  - text: |
      The exponential's job is to make the numbers positive, since logits are
      routinely negative and you cannot sample from negative weights. Squaring is
      positive too but throws the sign away, so exp is simply the cleaner way of
      handling negatives before normalizing.
    misconception: U2-M1
    explain: |
      Positivity does not discriminate: $|z_i| / \sum_j |z_j|$ is also positive and
      also broken. What exp alone gives you is
      $e^{z_i}/e^{z_j} = e^{z_i - z_j}$ - differences turned into ratios - which is
      the property requirement 5 forces, and normalization is the visible step that
      hides it.
  - text: |
      The exponential keeps the computation numerically safe: it maps any real
      logit into a well-behaved positive range so nothing overflows, whereas
      squaring large logits would blow up in floating point and destabilize
      training.
    misconception: M7
    explain: |
      Backwards. $e^{z}$ is what threatens to overflow, which is why every
      implementation subtracts $\max_j z_j$ first - free, because softmax is
      shift-invariant. The exponential is there for order preservation and for
      turning differences into ratios, not for float safety.
  - text: |
      Softmax outputs are probabilities that each option is correct, and
      probabilities are exponential in the score - that is why exp is the right
      function, and squaring would not produce valid probabilities.
    misconception: M2
    explain: |
      Squaring and normalizing also produces non-negative numbers summing to 1,
      which is all "valid distribution" means - it is rejected for breaking
      monotonicity, not for failing to be a distribution. Softmax is a
      differentiable soft-argmax parameterizing a sampling distribution; nothing
      about exp makes those numbers correctness probabilities.
check: choice
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

  Before reading on, commit: which reading is right about what changed and what
  did not?
options:
  - text: |
      Both halves fail. The logits are identical at $T = 0$ and $T = 1$ - the same
      forward pass produced them - and temperature only rescales them downstream at
      sampling time, so $T = 0$ is argmax over the same distribution and reveals
      only the option already ranked top at $T = 1$. The spread at $T = 1$ is not
      uncertainty about correctness either: it is a frequency-shaped weighting over
      continuations, not a calibrated probability of being right.
    correct: true
    explain: |
      Right on both counts. $p_i(T) = e^{z_i/T} / \sum_j e^{z_j/T}$ moves mass over
      fixed logits and cannot reorder them, so nothing hidden is exposed at $T = 0$
      and nothing about truth is being reported at $T = 1$.
  - text: |
      The first half is sound: $T = 0$ strips out the sampler's randomness, so what
      is left is the model's actual belief. Only the second half is sloppy - the
      "hedging" at $T = 1$ is the noise that temperature adds back in.
    misconception: M6
    explain: |
      Nothing is added or removed. $T = 0$ is argmax over the very same numbers
      that produced the $T = 1$ spread, and the top-ranked option is already visible
      there as the largest probability. Ranking is invariant across every $T$,
      because dividing by a positive number preserves order.
  - text: |
      The first half is wrong - the logits do not change - but the second half
      stands: a spread-out distribution at $T = 1$ is the model reporting genuine
      uncertainty about which answer is correct, which is what "hedging" describes.
    misconception: M2
    explain: |
      Softmax mass is a normalized exponential weighting calibrated to how often
      continuations followed similar context in training, not to truth. A quantity a
      config flag can move from $0.117$ to $0.307$ with the model's computation
      bit-identical throughout was never measuring correctness.
  - text: |
      Temperature is part of the forward pass, so at $T = 0$ the model computes
      different, sharper logits than at $T = 1$; the colleague is reading a real
      change in the model's own scores, which is why the answer came out confident.
    misconception: M6
    explain: |
      Temperature is applied after the logits, often in the serving layer rather
      than the model at all. With $z = (2, 1, 0)$ the logits are byte-identical at
      every $T$; only $z/T$ entering the softmax differs, giving
      $(0.867, 0.117, 0.016)$ at $T = 0.5$ and $(0.665, 0.245, 0.090)$ at $T = 1$.
check: choice
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

## Gradients of matrices

<!-- canon-only -->

For a scalar loss $L$ and a weight matrix $W$, the symbol
$\partial L / \partial W$ - also written $\nabla_W L$ - denotes the collection
of partial derivatives of $L$ with respect to each entry of $W$.

<!-- refutes: U0-M3 -->
You probably think of that symbol as a slope: one number saying which way the
loss is heading. Here is the prediction that fails: the update rule
$W \leftarrow W - \eta \, \partial L / \partial W$, where $\eta$ is the learning
rate, would then be subtracting a scalar from a matrix, moving every weight by
the identical amount in the identical direction forever. Training would be a
single global dial.

The belief is appealing because that is what a derivative is in one-variable
calculus, which is the last place most engineers used one.

What is actually true, and it is the most useful fact in this section: **the
gradient has the same shape as the thing you differentiate with respect to.** If
$W$ has shape $(768, 3072)$, so does $\partial L / \partial W$. Entry $(i,j)$
answers one narrow question - nudge $W_{ij}$ up by a hair, change nothing else,
how much does $L$ go up? The update subtracts the whole gradient matrix from the
whole weight matrix entrywise, so every weight gets its own step. That shape
correspondence is called denominator layout, and every ML framework uses it
because it makes the update rule shape-correct by construction. (Some math texts
use numerator layout, where the gradient comes out transposed. When a paper's
shapes look transposed from what the code does, this is usually why.)


```beat
id: u0-b6
type: self-explain
concept: c-notation
prompt: |
  A transformer's MLP has a weight matrix $W$ of shape
  $(d_{\text{model}}, d_{\text{ff}})$. Which account correctly states the shape of
  $\partial L / \partial W$ and what one single entry of it tells you?
options:
  - text: |
      Same shape as $W$, namely $(d_{\text{model}}, d_{\text{ff}})$. Entry $(i,j)$
      is $\partial L / \partial W_{ij}$: how much the scalar loss $L$ moves per unit
      increase in that one weight, holding every other weight fixed. The matching
      shapes are what make the entrywise update
      $W \leftarrow W - \eta \, \partial L / \partial W$ well-defined.
    correct: true
    explain: |
      Right. The gradient is shaped like the thing you differentiate with respect
      to, never like the loss - that is denominator layout, and it is why the update
      rule is shape-correct by construction.
  - text: |
      It is a single number - the slope of the loss - which is what a derivative is.
      The update $W \leftarrow W - \eta \, \partial L / \partial W$ subtracts that
      one number from the weight matrix.
    misconception: U0-M3
    explain: |
      That would move every weight by the identical amount in the identical
      direction forever, making training a single global dial. There are
      $d_{\text{model}} \times d_{\text{ff}}$ partial derivatives, one per weight,
      and each weight gets its own step.
  - text: |
      Shape $(d_{\text{model}}, d_{\text{ff}})$, matching $W$ entry for entry. Entry
      $(i,j)$ measures how important that weight is to the network - the large
      entries are the load-bearing weights and the small ones are the ones you can
      safely prune.
    misconception: U2-M5
    explain: |
      Shape right, reading wrong. A partial derivative is a local sensitivity at
      this exact point, not an attribution of the current loss. A weight sitting at
      its optimum has gradient zero and can still be load-bearing, and the ranking
      by gradient magnitude changes completely from step to step.
  - text: |
      Shape $(d_{\text{ff}}, d_{\text{model}})$, the transpose of $W$, and entry
      $(i,j)$ is the value that weight should be moved to, since the gradient tells
      you where the loss is minimized.
    misconception: U2-M2
    explain: |
      Two errors. Frameworks use denominator layout, so the gradient is
      $(d_{\text{model}}, d_{\text{ff}})$, entry for entry with $W$. And an entry is
      computed entirely from the surface right here and carries no information about
      where any minimum lies - which is why you take a small step $\eta$ and
      recompute rather than jumping to a target value.
check: choice
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
  Same network, new numbers.

  $$z = wx + b \qquad h = z^2 \qquad L = (h - y)^2$$

  with $x = 3$, $w = 1$, $b = -1$, $y = 5$.

  Forward pass:
  $z = 1 \cdot 3 + (-1) = 2$
  $h = 2^2 = 4$
  $L = (4 - 5)^2 = 1$

  Local derivatives:
  $\dfrac{\partial L}{\partial h} = 2(h - y) = $ ____   <- A
  $\dfrac{\partial h}{\partial z} = 2z = $ ____   <- B
  $\dfrac{\partial z}{\partial w} = x = 3$

  Chain them:
  $\dfrac{\partial L}{\partial w} = $ ____   <- C

  And why does $\dfrac{\partial L}{\partial w}$ here have a larger magnitude than
  the $-16$ of the worked example, when the loss is the same value of 1? ____ <- D

  Choose the filling that completes all four blanks.
options:
  - text: |
      A $= -2$, B $= 4$, C $= (-2)(4)(3) = -24$. D: the magnitude is larger because
      $\partial z / \partial w = x$, and the input is $x = 3$ here against $x = 2$
      in the worked example; the two upstream factors are identical, so the gradient
      scales directly with the input magnitude.
    correct: true
    explain: |
      Right: $2(4 - 5) = -2$, $2 \cdot 2 = 4$, and $(-2)(4)(3) = -24$. The path sets
      the magnitude and the input sits on that path, which is most of why input
      normalization matters.
  - text: |
      A $= 2$, B $= 4$, C $= (2)(4)(3) = 24$. D: larger because the input $x = 3$ is
      bigger than the $x = 2$ of the worked example, and the gradient scales with
      the input.
    misconception: U2-M2
    explain: |
      The magnitudes are right and the sign is not: $\partial L / \partial h = 2(h - y) = 2(4 - 5) = -2$,
      so $\partial L / \partial w = -24$. The sign is the part that sets the update
      direction. Sanity-check it: $h = 4$ sits below the target $y = 5$, so we want
      $z$ bigger, so we want $w$ bigger, and $w \leftarrow w - \eta(-24)$ raises $w$.
      A positive gradient would walk uphill.
  - text: |
      A $= -2$, B $= 4$, C $= (-2)(4)(3) = -24$. D: larger because $w$ carries more
      of the blame for the loss in this network - a bigger gradient means more of
      the current loss is attributable to that weight.
    misconception: U2-M5
    explain: |
      Numbers right, story wrong. The loss is 1 in both networks; the only thing
      that differs is one factor on the path, $\partial z / \partial w = x$. A
      partial derivative is a local sensitivity, not a share of the loss - a
      parameter sitting at its optimum has zero gradient and can still be
      load-bearing.
check: choice
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

  Which explanation of that result is right - and which reading of "the two runs
  came out with identical weights" goes with it?
options:
  - text: "A surface in $10^9$-plus dimensions has an astronomical number of distinct low-loss regions, and the seed - which fixes initialization and data order - selects which one the trajectory lands in. Comparable benchmark scores say the two regions are of comparable quality, which is the property that matters; weight identity is not, especially since permutation symmetry means many different tensors compute the same function. Identical weights across different seeds would mean some source of randomness is pinned - a bug in the experiment."
    correct: true
    explain: "Right. There is no reason two trajectories from different starting points should meet, and no need for them to. The seed picks a region; the benchmark checks the region is good. And because different seeds genuinely should diverge, matching tensors is evidence the seeding is broken, not evidence of convergence on a true answer."
  - text: "One run reached the global minimum and the other settled in a nearby local minimum that happens to be almost as deep, which is why the scores are within noise. Identical weights from two seeds would be the ideal outcome - it would mean both runs found the one true optimum."
    misconception: M5
    explain: "This keeps the 'one true minimum' frame and just relabels the second run a near-miss. SGD is not hunting a unique optimum: it finds one of astronomically many good-enough low-loss regions, and which one depends on seed, data order and hardware nondeterminism. Nearly all of them are fine, so neither run is the failure."
  - text: "With billions of parameters the surface has vastly more local minima to fall into, so each run gets trapped in a different one. The matching scores are luck; identical weights would show the optimizer had escaped the traps."
    misconception: U2-M4
    explain: "Backwards. Being trapped requires the surface to curve upward along all $d$ directions at once, so each added dimension is another chance for an escape route - entrapment gets rarer with scale, not commoner. Almost every critical point is a saddle, and minibatch gradient noise walks off saddles."
  - text: "The weight tensors only look different because the loss landed at the same depth by chance; benchmark scores are too coarse to see the difference. A finer evaluation would separate the runs, and identical weights would simply mean the evaluation was finally sensitive enough."
    misconception: M5
    explain: "This still assumes there is a single correct set of weights that a sharp enough test would reveal. The regions really are different and really are of comparable quality - wide low-loss basins are the normal case, and 'found a different one this time' is not a measurement artifact to be resolved away."
check: choice
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
