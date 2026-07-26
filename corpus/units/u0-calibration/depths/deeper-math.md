# Deeper math: u0

## How this unit works

Nothing to deepen. This section is scaffolding, not content.

## The shape contract

The shape rule is a statement about function composition, and it is worth seeing
it stated precisely once, because every later shape argument in this book is a
special case.

A matrix $W$ of shape $(p, q)$ under the row-vector convention defines a linear
map $f_W : \mathbb{R}^p \to \mathbb{R}^q$ by $f_W(x) = xW$, where $x$ is a row
vector of length $p$ and $xW$ is a row vector of length $q$. Linear means
$f_W(\alpha x + \beta y) = \alpha f_W(x) + \beta f_W(y)$ for scalars $\alpha,
\beta$ and row vectors $x, y$ - the map commutes with addition and scaling.

Two facts follow that are easy to state and worth internalizing.

**Every linear map is a matrix.** Let $e_1, \dots, e_p$ be the standard basis
row vectors ($e_i$ has a $1$ in position $i$ and zeros elsewhere). Any $x$
decomposes as $x = \sum_i x_i e_i$, so linearity forces
$f(x) = \sum_i x_i f(e_i)$. The map is therefore fully determined by the $p$
output vectors $f(e_i)$, and stacking those as the rows of a $(p, q)$ matrix
reconstructs $f$ exactly. This is why "matrix" and "linear map" are the same
object seen from two sides, and why counting parameters is the same as counting
degrees of freedom in the map.

**Composition is multiplication.** If $f_A : \mathbb{R}^p \to \mathbb{R}^q$ and
$f_B : \mathbb{R}^q \to \mathbb{R}^r$, then $f_B \circ f_A$ is linear (composite
of linear maps is linear, by direct substitution) and therefore is itself a
matrix, of shape $(p, r)$. Working out its entries from the basis argument above
yields exactly $\sum_t A_{it} B_{tj}$. The inner-dimension rule is the statement
that $f_A$'s codomain and $f_B$'s domain must be the same space.

One consequence that matters in u4: a stack of linear layers with no
nonlinearity between them is a single linear layer.
$X W_1 W_2 W_3 = X (W_1 W_2 W_3) = X W'$ by associativity, and $W'$ has the
shape of a single map from the first input dimension to the last output
dimension. Depth without nonlinearity buys exactly nothing in expressive power -
it only changes the parameterization. That is the formal reason activation
functions are not optional.

## Dot products: alignment, not distance

The geometric identity $u \cdot v = \lVert u \rVert \lVert v \rVert \cos\theta$
is usually asserted. Here is where it comes from, since the argument is three
lines and explains why "angle" is even definable in 512 dimensions.

Start from the law of cosines applied to the triangle with sides $u$, $v$, and
$u - v$:

$$\lVert u - v \rVert^2 = \lVert u \rVert^2 + \lVert v \rVert^2 - 2 \lVert u \rVert \lVert v \rVert \cos\theta$$

Now expand the left side using bilinearity of the dot product (the fact that
$(a + b) \cdot c = a \cdot c + b \cdot c$, which follows immediately from the
componentwise definition) and the identity $\lVert w \rVert^2 = w \cdot w$:

$$\lVert u - v \rVert^2 = (u - v)\cdot(u - v) = u \cdot u - 2 (u \cdot v) + v \cdot v = \lVert u \rVert^2 - 2(u \cdot v) + \lVert v \rVert^2$$

Set the two expressions equal, cancel $\lVert u \rVert^2 + \lVert v \rVert^2$
from both sides, and divide by $-2$:

$$u \cdot v = \lVert u \rVert \lVert v \rVert \cos\theta$$

In two or three dimensions this derives the algebraic formula from a geometric
angle you can see. In $d_{\text{model}} = 4096$ dimensions the logic runs the
other way: there is no visual angle, so the equation is taken as the
**definition** of $\theta$, and the only thing that makes it a legitimate
definition is the Cauchy-Schwarz inequality,
$|u \cdot v| \le \lVert u \rVert \lVert v \rVert$, which guarantees the ratio
$(u \cdot v) / (\lVert u \rVert \lVert v \rVert)$ lands in $[-1, 1]$ and so is
the cosine of something. Every geometric statement about high-dimensional
embeddings in this book rests on that inequality.

Two quantitative facts to carry into u3.

**Random vectors are nearly orthogonal, and increasingly so with dimension.**
If $u$ and $v$ have i.i.d. zero-mean, unit-variance components, then
$\mathbb{E}[u \cdot v] = \sum_i \mathbb{E}[u_i]\mathbb{E}[v_i] = 0$ by
independence, while $\lVert u \rVert \approx \sqrt{d}$. The cosine therefore
concentrates around $0$ with spread on the order of $1/\sqrt{d}$. High
dimensions are mostly empty, which is what lets a model pack many
near-independent directions into one residual stream (u4, and the superposition
extension x2).

**Dot-product variance grows linearly in dimension.** Under the same
assumptions, $\text{Var}(u \cdot v) = \sum_{i=1}^{d} \text{Var}(u_i v_i) = d$,
so the typical magnitude of $u \cdot v$ scales as $\sqrt{d}$. That single line
is the entire justification for the $1/\sqrt{d_k}$ factor in scaled dot-product
attention: dividing by $\sqrt{d_k}$ restores unit-scale scores regardless of
head width, which keeps softmax out of its saturated regime.

## Matrix multiply: composition, not a loop

Two properties, with the reasons.

**Associativity holds: $(AB)C = A(BC)$.** Both sides equal
$\sum_{s}\sum_{t} A_{is} B_{st} C_{tj}$ - the same triple sum, summed in a
different order, and finite sums may be reordered freely. From the function
view it is trivial: composition of functions is associative because both sides
describe "apply $A$, then $B$, then $C$".

This is not a formality. Associativity is a compute-cost lever with no
correctness cost. Take $A$ of shape $(n, d)$, $B$ of shape $(d, d)$, and $C$ of
shape $(d, 1)$. A multiply of shapes $(p,q)$ by $(q,r)$ costs $pqr$
multiply-adds, so $(AB)C$ costs $n d^2 + n d$ while $A(BC)$ costs $d^2 + nd$.
For $n = 100{,}000$ and $d = 4096$ that is roughly a $10^5$-fold difference in
the dominant term for a bit-identical result. Linear-attention variants are
essentially this observation applied to $QK^TV$.

**Commutativity fails: $AB \neq BA$.** Usually the shapes forbid it outright.
When both products exist and are the same shape (square $A$ and $B$), they still
differ in general - a rotation followed by a scaling along one axis is not the
same map as the scaling followed by the rotation. The clean test case:

$$A = \begin{bmatrix} 0 & 1 \\ 0 & 0 \end{bmatrix}, \quad
B = \begin{bmatrix} 0 & 0 \\ 1 & 0 \end{bmatrix}, \quad
AB = \begin{bmatrix} 1 & 0 \\ 0 & 0 \end{bmatrix}, \quad
BA = \begin{bmatrix} 0 & 0 \\ 0 & 1 \end{bmatrix}$$

Different matrices, not even sharing a diagonal. The gap $AB - BA$ has a name,
the commutator, and in the interpretability literature (x2) it is one way to ask
whether two learned transformations interfere.

Finally, transposition reverses order: $(AB)^T = B^T A^T$. Check the shapes -
$A$ is $(p,q)$, $B$ is $(q,r)$, so $(AB)^T$ is $(r,p)$, and $B^T A^T$ is
$(r,q)$ times $(q,p)$, which is $(r,p)$. Entrywise,
$((AB)^T)_{ij} = (AB)_{ji} = \sum_t A_{jt}B_{ti} = \sum_t (B^T)_{it}(A^T)_{tj}$.
This identity is why backpropagation through a linear layer transposes the
weight matrix, and it accounts for most of the transposes you will see in u5.

## Gradients and expectations

**Layout conventions, stated exactly.** For a scalar $L$ and a matrix $W$ of
shape $(m, n)$, denominator layout (also called gradient layout) defines

$$\left(\frac{\partial L}{\partial W}\right)_{ij} = \frac{\partial L}{\partial W_{ij}}$$

giving a result of shape $(m, n)$. Numerator layout (also called Jacobian
layout) defines the transpose, shape $(n, m)$. Neither is more correct; they are
different index-ordering conventions. The reason every framework uses
denominator layout for scalar losses is that the SGD update
$W \leftarrow W - \eta \, \partial L / \partial W$ is then shape-correct without
a transpose.

The general object underneath both is the Jacobian. For a vector function
$f : \mathbb{R}^n \to \mathbb{R}^m$, the Jacobian $J$ has shape $(m, n)$ with
$J_{ij} = \partial f_i / \partial x_j$. A gradient is the special case $m = 1$,
transposed to match the input's shape. The chain rule in full generality is
Jacobian multiplication: for $f = g \circ h$,
$J_f(x) = J_g(h(x)) \, J_h(x)$, a matrix product of shapes
$(m, k)$ times $(k, n)$.

**Why backprop never builds those Jacobians.** For a layer mapping 4096
dimensions to 4096 dimensions the Jacobian holds $1.6 \times 10^7$ entries per
token, and a full forward-mode chain would materialize one per layer. Reverse
mode avoids this by only ever computing vector-Jacobian products. Carrying the
row vector $g = \partial L / \partial y$ backward through $y = xW$ costs one
matrix multiply per side:

$$\frac{\partial L}{\partial x} = g W^T, \qquad \frac{\partial L}{\partial W} = x^T g$$

Check the shapes with $x$ of shape $(1, p)$, $W$ of shape $(p, q)$, $g$ of shape
$(1, q)$: $gW^T$ is $(1,q) \times (q,p) = (1,p)$, matching $x$; and $x^T g$ is
$(p,1) \times (1,q) = (p,q)$, matching $W$. The gradient wearing the shape of
its operand is not a coincidence to memorize - it falls out of the algebra.
Those two lines are the whole of backpropagation through a linear layer, and u5
derives the rest of the network by repeating them.

**Expectations and gradient noise.** The training objective is
$L(\theta) = \mathbb{E}_{x \sim \mathcal{D}}[\ell(x; \theta)]$ where $\theta$
denotes all parameters and $\ell$ is the per-example loss. Gradient and
expectation commute (differentiation under the integral sign, valid under mild
regularity conditions that hold here):

$$\nabla_\theta \, \mathbb{E}_{x \sim \mathcal{D}}[\ell(x;\theta)] = \mathbb{E}_{x \sim \mathcal{D}}[\nabla_\theta \, \ell(x;\theta)]$$

This is the licence for the entire SGD enterprise: the batch-mean gradient is an
**unbiased** estimator of the true gradient. Its variance falls as $1/b$ for
batch size $b$, so the noise scale falls as $1/\sqrt{b}$ - quadrupling the batch
halves the gradient noise. Notice what that implies: the noise never reaches
zero at finite batch size, so SGD is a random walk with drift, not a descent.
u2's treatment of why that noise is useful rather than merely tolerable builds
directly on this.
