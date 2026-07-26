---
unit: u0
title: "Calibration: where you actually are"
concepts:
  - c-notation
  - c-matmul
  - c-dotprod
assumes: []
---

## How this unit works

This unit does two things and then gets out of the way.

First, it measures. The pretest that opens this unit spans the entire math floor
of the book - dot products, matrix shapes, softmax, the chain rule, gradient
notation. You are expected to miss some of it. Missing items here is the point:
a failed attempt before instruction makes the instruction stick harder, and the
pattern of what you miss is what sets the depth of every later unit. Answer
fast, answer wrong, move on.

Second, it front-loads notation. Everything after this unit is written in
symbols, and the symbols are the actual barrier for a working engineer reading
ML papers. Not the concepts - the concepts are ordinary. The barrier is that
$\partial L / \partial W$, $\mathbb{E}_{x \sim \mathcal{D}}[\cdot]$, and
$d_k$ are never defined in the papers that use them.

The four sections below are that notation pre-training. They are self-contained:
nothing later in this unit depends on them, and nothing in them depends on the
rest of the book. If your pretest shows you already read this dialect fluently,
they get skipped, because pre-training notation for someone who already has it
costs time and buys nothing.

<!-- skippable: notation-pretraining begins -->

## The shape contract

Every quantity in this book is a block of numbers, and the only thing you need
to track about it is its shape. Get the shapes right and the equations follow
almost mechanically. Get them wrong and nothing type-checks.

The index letters are near-universal across papers. Memorize these five:

| Symbol | Means | Typical value |
| --- | --- | --- |
| $b$ | batch size - how many independent sequences processed at once | 1 to 1024 |
| $n$ | sequence length - number of tokens in one sequence | 8 to 1,000,000 |
| $d_{\text{model}}$ | width of the residual stream - the vector size carried between layers | 768 to 16384 |
| $d_k$ | width of one attention head's query/key vectors | 64 to 128 |
| $V$ | vocabulary size - how many distinct tokens exist | ~32,000 to ~200,000 |

So a batch of token embeddings has shape $(b, n, d_{\text{model}})$: $b$
sequences, each $n$ tokens long, each token represented by a
$d_{\text{model}}$-dimensional vector. That is the single most common shape in
this book. In most equations the batch dimension is dropped and left implicit,
so you will see $X$ with shape $(n, d_{\text{model}})$ and be expected to
understand that everything applies per-sequence, in parallel, across the batch.

Now the part that trips engineers, because it is a convention choice that nobody
announces.

<!-- refutes: U0-M4 -->
You probably think the row-vs-column orientation of a vector is cosmetic - a
transpose here or there, the kind of thing you fix when the code throws. Here is
the prediction that fails: under that belief, a paper writing $xW$ and a
textbook writing $Wx$ should be describing the same operation with the same
matrix, so you could carry a shape from one to the other. Try it.
In $Wx$, the linear-algebra-textbook form, $x$ is a column vector of shape
$(d_{\text{in}}, 1)$ and $W$ has shape $(d_{\text{out}}, d_{\text{in}})$ -
input dimension **last**. In $xW$, the form nearly all ML papers and every
tensor library use, $x$ is a row vector of shape $(1, d_{\text{in}})$ and $W$
has shape $(d_{\text{in}}, d_{\text{out}})$ - input dimension **first**. The
same weight matrix is stored transposed between the two worlds. Read a paper in
the wrong convention and every shape you derive is backwards.

The belief is appealing because in scalar-land orientation genuinely is
cosmetic, and because Python broadcasting hides orientation errors until they
surface three layers downstream as a wrong-but-plausible shape.

What is actually true: this book, like the papers, uses the **row-vector
convention** throughout. Data comes first, weights come second, dimensions
contract at the join:

$$X W = Y, \quad X: (n, d_{\text{in}}), \quad W: (d_{\text{in}}, d_{\text{out}}), \quad Y: (n, d_{\text{out}})$$

where $X$ holds one token vector per row, $W$ is a learned weight matrix, and
$Y$ holds one output vector per row. The inner dimensions - the $d_{\text{in}}$
appearing on both sides of the join - must match and then vanish. The outer
dimensions survive. That is the whole rule, and it is the only shape rule you
will need for the rest of the book.

```beat
id: u0-b1
type: compute
concept: c-notation
prompt: |
  Row-vector convention. $X$ has shape $(12, 64)$ and $W$ has shape
  $(64, 256)$. Give the shape of $XW$.
  Answer in the exact form `rows x cols`, for example `3 x 8`.
answer: "12 x 256"
check: exact
```

## Dot products: alignment, not distance

The dot product takes two vectors of the same length and returns one number.
That is its whole type signature, and half of what makes attention confusing is
forgetting that the output is a scalar.

<!-- fade: dot-product -->
For $u = [2, -1, 3]$ and $v = [1, 4, 2]$, multiply componentwise and sum:

$$u \cdot v = (2)(1) + (-1)(4) + (3)(2) = 2 - 4 + 6 = 4$$

where $u \cdot v$ (also written $u v^T$ in row-vector convention, or
$\langle u, v \rangle$) denotes the dot product, and the result $4$ is a single
number, not a vector. Three multiplications, two additions, one scalar out.

The geometric identity is where the meaning lives:

$$u \cdot v = \lVert u \rVert \, \lVert v \rVert \cos\theta$$

where $\lVert u \rVert$ is the length (Euclidean norm) of $u$, computed as
$\sqrt{\sum_i u_i^2}$, and $\theta$ is the angle between the two vectors. So the
dot product is **alignment scaled by both magnitudes**. Positive means the
vectors point the same general way, zero means perpendicular, negative means
opposed.

<!-- refutes: U0-M2 -->
You probably think this makes the dot product a similarity score - that a bigger
dot product means "more similar", the way cosine similarity does in every vector
database you have used. Here is the prediction that fails: under that belief,
scaling a vector without rotating it should not change how similar it is to
anything, since its direction is unchanged. Take $q = [3, 0]$ and two keys
pointing in the identical direction, $k_1 = [0.6, 0.8]$ and $k_2 = [6, 8]$.
Cosine similarity is $0.6$ for both - same direction, same angle. But
$q \cdot k_1 = 1.8$ and $q \cdot k_2 = 18$. Ten times the score for zero change
in direction.

The belief is appealing because your tooling normalizes for you: embedding
databases store unit-length vectors, and on unit vectors the dot product and the
cosine are literally the same number. That is a property of the normalization,
not of the dot product.

What is actually true: the dot product is cosine similarity multiplied by both
magnitudes, so it conflates "points the same way" with "is large". Inside a
transformer nothing is normalized before the dot product, and this matters
twice: it is why attention scores need a $1/\sqrt{d_k}$ correction (u3), and it
is why a single high-magnitude key can dominate an attention distribution
regardless of direction.

<!-- refutes: U0-M6 -->
The neighboring instinct - that a bigger dot product means "closer" - fails on
the same example, and harder. If the dot product tracked proximity it would
shrink as vectors move apart. Measure: $\lVert q - k_1 \rVert \approx 2.53$ and
$\lVert q - k_2 \rVert \approx 8.54$, so $k_2$ is more than three times farther
from $q$, and it scores ten times higher. The dot product is not a distance and
is not even monotone in distance. The two are related by
$\lVert u - v \rVert^2 = \lVert u \rVert^2 - 2(u \cdot v) + \lVert v \rVert^2$,
which agrees on ranking only when all the norms are equal - the unit-sphere case
your vector database quietly enforces and a transformer does not.

```beat
id: u0-b2
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

## Matrix multiply: composition, not a loop

<!-- refutes: U0-M1 -->
You probably think matrix multiplication is a triple-nested loop whose
inner-dimensions-must-match rule is bookkeeping - an implementation detail of
how the numbers happen to be stored. Here is the prediction that fails: under
that belief, the rule should be arbitrary enough that some other pairing (say,
multiplying aligned entries, or matching outer dimensions) would be an equally
valid definition, just a different convention. It is not. Matrices are
functions. $A$ of shape $(p, q)$ is a function from $q$-dimensional space to
$p$-dimensional space in column convention, or from $p$ to $q$ in row
convention. $AB$ is the **composition** of those two functions, and the shape
rule is nothing but the requirement that the output type of one function matches
the input type of the next. It is a type check, not a storage detail. That is
also why $AB \neq BA$: composing in the other order is a different function, and
usually not even a well-typed one.

The belief is appealing because the loop is what the hardware does, and as a
systems engineer you correctly reason about the memory-layout level for
performance. That level is real. It is just not where the meaning is.

What is actually true: every entry of the product is a dot product, and the
whole product is a change of representation. Entry $(i, j)$ of $AB$ is row $i$
of $A$ dotted with column $j$ of $B$:

$$(AB)_{ij} = \sum_{t} A_{it} B_{tj}$$

where $t$ runs over the shared inner dimension, $A_{it}$ is the entry of $A$ in
row $i$ and column $t$, and $B_{tj}$ is the entry of $B$ in row $t$ and column
$j$. So a matrix multiply is a grid of dot products - one per (row of $A$,
column of $B$) pair.

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

Note that $BA$ is also defined here - $(3,2)$ times $(2,3)$ gives $(3,3)$ - and
is a completely different object, a $3 \times 3$ matrix. Same two matrices,
different composition, different function.

```beat
id: u0-b3
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
```

## Gradients and expectations

Two more pieces of notation and the pre-training is done. Both are the kind of
symbol that papers use hundreds of times without ever defining.

**Expectation.** $\mathbb{E}_{x \sim \mathcal{D}}[f(x)]$ reads "the expected
value of $f(x)$ when $x$ is drawn from the distribution $\mathcal{D}$" - the
average of $f(x)$ over all possible $x$, weighted by how likely each $x$ is
under $\mathcal{D}$. The subscript names the distribution; the brackets hold
whatever you are averaging.

You cannot compute that average - $\mathcal{D}$ is "the distribution of all
text that could exist", and you have a hard drive, not a distribution. So every
expectation in this book is estimated by a sample mean over a batch:

$$\mathbb{E}_{x \sim \mathcal{D}}[f(x)] \approx \frac{1}{b} \sum_{i=1}^{b} f(x_i)$$

where $b$ is the batch size and $x_1, \dots, x_b$ are the examples in the batch.

<!-- refutes: U0-M5 -->
You probably read that approximation as an equality with extra ceremony - that
$\mathbb{E}$ is just notation for "average of the numbers I have". Here is the
prediction that fails: if the objective were defined over your dataset, then a
model that memorized the dataset would have optimally solved the stated problem,
and generalization error would not exist as a concept. Every number anyone
reports is held-out loss. The belief is appealing because the batch mean is what
the code literally computes, and the dataset is the only concrete object in
sight. What is true is that the thing you want is the expectation over a
distribution nobody can enumerate, the thing you compute is an unbiased but
noisy estimate of it, and the gap between them is exactly why batch size affects
training stability (u5). One related trap: $\mathbb{E}$ is a probability-
weighted mean, not the typical value. The expected roll of a fair die is 3.5.

**Gradients.** For a scalar loss $L$ and a weight matrix $W$, the symbol
$\partial L / \partial W$ - also written $\nabla_W L$ - denotes the collection
of partial derivatives of $L$ with respect to each entry of $W$.

<!-- refutes: U0-M3 -->
You probably think of that symbol as a slope: one number saying which way the
loss is heading. Here is the prediction that fails: under that belief the update
rule $W \leftarrow W - \eta \, \partial L / \partial W$, where $\eta$ is the
learning rate, would be subtracting a scalar from a matrix - it would move every
weight by the identical amount, in the identical direction, forever. Training
would be a single global dial. It obviously is not.

The belief is appealing because that is what a derivative is in one-variable
calculus, which is the last place most engineers used one.

What is actually true, and it is the single most useful fact in this section:
**the gradient has the same shape as the thing you differentiate with respect
to.** If $W$ has shape $(768, 3072)$, then $\partial L / \partial W$ has shape
$(768, 3072)$. Entry $(i,j)$ of the gradient answers one narrow question: if I
nudge $W_{ij}$ up by a hair and change nothing else, how much does $L$ go up?
The update subtracts the whole gradient matrix from the whole weight matrix,
entrywise, so every weight gets its own individually-computed step. That shape
correspondence is called denominator layout, and it is the convention every ML
framework uses because it makes the update rule shape-correct by construction.
(Some math texts use numerator layout, where the gradient comes out transposed.
When a paper's shapes look transposed from what the code does, this is usually
why.)

The chain rule is what makes this computable through a deep stack. In scalar
form: if $L$ depends on $y$ and $y$ depends on $x$, then

$$\frac{\partial L}{\partial x} = \frac{\partial L}{\partial y} \cdot \frac{\partial y}{\partial x}$$

Local derivatives multiply along the path. Backpropagation (u5) is that
identity applied layer by layer, with matrix multiplies in place of the scalar
product - which is why it is bookkeeping rather than new mathematics.

```beat
id: u0-b4
type: self-explain
concept: c-notation
prompt: |
  In your own words, in two or three sentences: a transformer's MLP has a weight
  matrix $W$ of shape $(d_{\text{model}}, d_{\text{ff}})$. What shape is
  $\partial L / \partial W$, and what does one single entry of it tell you?
answer: |
  Same shape as $W$, namely $(d_{\text{model}}, d_{\text{ff}})$. Entry $(i,j)$
  is the partial derivative of the scalar loss with respect to the single weight
  $W_{ij}$: how much $L$ changes per unit increase in that one weight, holding
  all other weights fixed. Matching shapes is what makes the entrywise update
  $W \leftarrow W - \eta \, \partial L / \partial W$ well-defined.
rubric: |
  Must contain both: (1) the gradient has the SAME shape as $W$,
  $(d_{\text{model}}, d_{\text{ff}})$ - stating the shape correctly is required,
  not just "same shape"; (2) one entry is the sensitivity of the scalar loss to
  one individual weight (partial derivative w.r.t. $W_{ij}$).
  Pass = both. Partial = (1) only, or (2) phrased as "how much that weight
  matters" without the derivative/sensitivity idea.
  Fail conditions and what they diagnose: calling the gradient a scalar or a
  single direction = U0-M3. Giving the transposed shape
  $(d_{\text{ff}}, d_{\text{model}})$ = numerator/denominator layout confusion,
  re-teach the layout paragraph. Describing an entry as "the value the weight
  should become" rather than a sensitivity = confusing gradient with update.
check: llm
```

<!-- skippable: notation-pretraining ends -->

That is the whole notation surface. Shapes contract at the join, dot products
return scalars that mix direction with magnitude, matrix multiply composes
functions, gradients wear the shape of their weights, and $\mathbb{E}$ means an
average you can only estimate. Every equation in the next nine units is built
from those five facts.
