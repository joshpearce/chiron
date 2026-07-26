---
unit: u2
depth: deeper-math
---

## Shapes are type signatures

The canon treats a matrix as a function. The precise statement is that for
finite-dimensional spaces with chosen bases, matrices and linear maps are the
same thing, and the matrix is the coordinate representation of the map.

Concretely: a linear map $T : \mathbb{R}^n \rightarrow \mathbb{R}^m$ is fully
determined by what it does to the standard basis vectors
$e_1, \ldots, e_n$ (where $e_j$ has a 1 in position $j$ and 0 elsewhere), because
any $x = \sum_j x_j e_j$ and linearity gives $Tx = \sum_j x_j\, T e_j$. Column $j$
of the matrix is literally the vector $Te_j$. That is a useful debugging tool:
to see what a weight matrix does, look at its columns as images of the basis.

Check it on the canon's rotation
$R = \begin{bmatrix} 0 & -1 \\ 1 & 0\end{bmatrix}$. Column 1 is
$\begin{bmatrix} 0 \\ 1\end{bmatrix}$, which is $e_1$ rotated 90 degrees. Column 2
is $\begin{bmatrix} -1 \\ 0\end{bmatrix}$, which is $e_2$ rotated 90 degrees. The
matrix is nothing but the two images written side by side.

### Rank, determinant, and what "destroys information" means precisely

The **rank** of $W$ is the dimension of its image (its column space). The
**null space** (kernel) is $\{x : Wx = 0\}$, the set of inputs annihilated. The
rank-nullity theorem ties them:

$$\mathrm{rank}(W) + \dim(\ker W) = n$$

for $W \in \mathbb{R}^{m \times n}$. Information is destroyed exactly when
$\dim(\ker W) > 0$, because then a whole subspace of distinct inputs maps to the
same output and no inverse exists.

For the canon's projection $P = \begin{bmatrix} 1 & 0 \\ 0 & 0\end{bmatrix}$:
rank 1, kernel is the $y$-axis (dimension 1), and $1 + 1 = 2 = n$. Every vector
$\begin{bmatrix} 3 \\ t\end{bmatrix}$ for any $t$ maps to
$\begin{bmatrix} 3 \\ 0\end{bmatrix}$.

For square $W$, the **determinant** $\det W$ is the signed factor by which volume
is scaled, and $\det W = 0$ exactly when the kernel is nontrivial. The canon's
three examples: $\det R = 1$ (rotation preserves area and orientation),
$\det S = 2 \cdot 0.5 = 1$ (this particular scaling stretches one axis and
squashes the other by reciprocal amounts, so area is coincidentally preserved),
$\det P = 0$ (flattened to a line, zero area, not invertible).

$P P = P$ is the definition of an **idempotent** operator, and every orthogonal
projection is idempotent. Its eigenvalues are all 0 or 1: eigenvalue 1 on the
subspace it keeps, 0 on the subspace it kills.

### Why this matters for transformer shapes

A transformer's per-head projections $W_Q, W_K \in \mathbb{R}^{d_{\text{model}}
\times d_k}$ with $d_k \ll d_{\text{model}}$ are deliberately rank-limited maps
into a small subspace. The low rank is the point: each head is forced to read a
low-dimensional slice of the residual stream rather than all of it. Unit u4's
"the residual stream is a shared workspace" claim is a statement about many
low-rank reads and writes coexisting in one high-dimensional space, which works
only because random subspaces in high dimension are nearly orthogonal.

## Composition is multiplication

Associativity, $(AB)C = A(BC)$, is the algebraic shadow of the fact that
composing functions is associative: doing $C$ then $B$ then $A$ is one
well-defined thing regardless of how you parenthesize. It is also the single
biggest lever in ML performance engineering, because the two groupings have
wildly different costs.

For $A \in \mathbb{R}^{m \times k}$, $B \in \mathbb{R}^{k \times n}$, computing
$AB$ costs $O(mkn)$ scalar multiply-adds. So with
$A \in \mathbb{R}^{1000 \times 1000}$, $B \in \mathbb{R}^{1000 \times 1000}$, and
$x \in \mathbb{R}^{1000}$:

- $(AB)x$ costs $1000^3 + 1000^2 = 1.001 \times 10^9$ operations.
- $A(Bx)$ costs $1000^2 + 1000^2 = 2 \times 10^6$ operations.

A factor of 500, from parenthesization alone. This is why LoRA works the way it
does: a low-rank update $\Delta W = BA$ with $B \in \mathbb{R}^{d \times r}$,
$A \in \mathbb{R}^{r \times d}$, $r \ll d$, is never materialized as a $d \times d$
matrix. You compute $B(Ax)$ and pay $O(dr)$ twice instead of $O(d^2)$ once.

### Non-commutativity, stated exactly

$AB = BA$ holds if and only if the commutator $[A, B] = AB - BA$ vanishes. For
the canon's example,

$$[R, B] = \begin{bmatrix} 0 & -1 \\ 2 & 0\end{bmatrix} - \begin{bmatrix} 0 & -2 \\ 1 & 0\end{bmatrix} = \begin{bmatrix} 0 & 1 \\ 1 & 0\end{bmatrix} \neq 0$$

The structural condition: two diagonalizable matrices commute if and only if they
are simultaneously diagonalizable, meaning they share a full set of eigenvectors.
Rotation and axis-aligned scaling do not - the scaling's eigenvectors are the
coordinate axes, and a real rotation by 90 degrees has no real eigenvectors at
all (its eigenvalues are $\pm i$). They have no shared frame in which both are
simply "stretch along these directions", so they cannot commute.

### Transposes and the reversal rule

$(AB)^T = B^T A^T$. The order reverses, and forgetting this is the most common
source of shape errors in hand-written backward passes. The reason is dimensional:
if $AB$ is $m \times n$ then $(AB)^T$ is $n \times m$, and the only way to build
that from $A^T$ ($k \times m$) and $B^T$ ($n \times k$) is $B^T A^T$. The types
force the order.

## The dot product is the whole game

The dot product is the standard **inner product** on $\mathbb{R}^n$, and the
geometry the canon uses is derived rather than assumed. Start from
$\lVert a \rVert^2 = a \cdot a$, then expand:

$$\lVert a - b\rVert^2 = (a-b)\cdot(a-b) = a\cdot a - 2\,a \cdot b + b \cdot b$$

Compare that against the law of cosines,
$\lVert a - b\rVert^2 = \lVert a\rVert^2 + \lVert b \rVert^2 - 2\lVert a\rVert
\lVert b\rVert \cos\theta$, and the two expressions agree only if
$a \cdot b = \lVert a\rVert \lVert b\rVert \cos\theta$. The geometric formula is a
theorem, not a definition.

**Cauchy-Schwarz**, $|a \cdot b| \leq \lVert a\rVert \lVert b\rVert$, follows
immediately since $|\cos\theta| \leq 1$, and it is the reason cosine similarity
is bounded in $[-1, 1]$.

### Concentration: why raw dot products blow up with dimension

This is the fact unit u3 needs for the $\sqrt{d_k}$ scaling, and it is a
statement about variance.

Let $q, k \in \mathbb{R}^{d}$ have independent components with mean 0 and
variance 1. Then

$$\mathbb{E}[q \cdot k] = \sum_{i=1}^{d} \mathbb{E}[q_i]\mathbb{E}[k_i] = 0$$

and, because the terms are independent and zero-mean,

$$\mathrm{Var}(q \cdot k) = \sum_{i=1}^{d} \mathrm{Var}(q_i k_i) = \sum_{i=1}^{d} \mathbb{E}[q_i^2]\mathbb{E}[k_i^2] = d$$

So the typical magnitude of a raw score grows like $\sqrt{d}$. At $d_k = 64$ that
is around 8; at $d_k = 512$, around 23. Feed scores of magnitude 23 into a
softmax and the gaps between them are enormous, the distribution saturates onto
one entry, and (per the Jacobian below) the gradient goes to zero. Dividing by
$\sqrt{d_k}$ renormalizes the variance back to 1 and keeps softmax in its
responsive regime. This is a variance argument, not an overflow argument.

## Softmax: why exponentials

### The functional equation, done properly

The canon asserts that requiring $p_i/p_j$ to depend only on $z_i - z_j$ forces
an exponential. The argument: we need a positive continuous $f$ with

$$\frac{f(u)}{f(v)} = g(u - v) \quad \text{for all } u, v$$

Set $v = 0$ and write $c = f(0) > 0$. Then $f(u) = c\, g(u)$. Substituting back,
$g(u)/g(v) = g(u-v)$, so with $s = u - v$ and $t = v$ we get

$$g(s + t) = g(s)\, g(t)$$

This is Cauchy's exponential functional equation. Its only continuous solutions
with $g > 0$ are $g(s) = e^{\beta s}$ for some real $\beta$. Monotonicity forces
$\beta > 0$, and $\beta$ is absorbed into the logit scale (it is precisely the
inverse temperature). Hence $f(z) = c\,e^{\beta z}$, the constant $c$ cancels in
normalization, and softmax is the unique answer.

### Two other derivations that land in the same place

**Maximum entropy.** Among all distributions $p$ over $n$ outcomes satisfying the
expectation constraint $\sum_i p_i z_i = \mu$, the one maximizing entropy
$H(p) = -\sum_i p_i \ln p_i$ is $p_i \propto e^{\beta z_i}$. Derive it with
Lagrange multipliers: maximize $-\sum p_i \ln p_i + \lambda(\sum p_i - 1) +
\beta(\sum p_i z_i - \mu)$; setting $\partial/\partial p_i = 0$ gives
$-\ln p_i - 1 + \lambda + \beta z_i = 0$, hence $p_i = e^{\beta z_i + \lambda - 1}$.
Softmax is the least-committed distribution consistent with the scores. This is
the same computation that produces the Boltzmann distribution in statistical
mechanics, which is where the word "temperature" comes from and why it is not
merely a metaphor.

**Log-sum-exp as a smooth maximum.** Define
$\mathrm{LSE}(z) = \ln \sum_j e^{z_j}$. Then

$$\max_j z_j \leq \mathrm{LSE}(z) \leq \max_j z_j + \ln n$$

so LSE is a maximum that is off by at most $\ln n$, and it is smooth everywhere.
Its gradient is exactly softmax:

$$\frac{\partial}{\partial z_i}\ln \sum_j e^{z_j} = \frac{e^{z_i}}{\sum_j e^{z_j}} = p_i$$

Softmax is the gradient of a smooth max. That is the most compact statement of
what it is for, and it explains "soft-argmax" precisely: the gradient of the max
is the indicator of the argmax, so the gradient of a smoothed max is a smoothed
indicator.

### The softmax Jacobian

Since softmax maps $\mathbb{R}^n \rightarrow \mathbb{R}^n$, its derivative is an
$n \times n$ Jacobian $J_{ij} = \partial p_i / \partial z_j$. Differentiating the
quotient:

$$\frac{\partial p_i}{\partial z_j} = p_i(\delta_{ij} - p_j)
\qquad \text{that is} \qquad J = \mathrm{diag}(p) - p\,p^{T}$$

where $\delta_{ij}$ is 1 when $i = j$ and 0 otherwise. Concretely for the canon's
$p = (0.665, 0.245, 0.090)$:

$$J = \begin{bmatrix}
0.2227 & -0.1628 & -0.0599 \\
-0.1628 & 0.1848 & -0.0220 \\
-0.0599 & -0.0220 & 0.0819
\end{bmatrix}$$

Three properties to read off it. It is symmetric. Every row sums to 0, which is
the derivative form of the constraint $\sum_i p_i = 1$ - mass can move but not
appear. And the off-diagonals are negative: raising one logit necessarily takes
probability from the others. Softmax couples every output to every input; there
is no independent per-token knob.

The vanishing-gradient failure mode is visible here. If $p$ saturates so that
$p_1 \rightarrow 1$ and the rest $\rightarrow 0$, then
$J_{11} = p_1(1 - p_1) \rightarrow 0$ and every other entry $\rightarrow 0$ too.
The Jacobian goes to the zero matrix and no gradient flows back through the
softmax at all. That is the concrete mechanism behind the $\sqrt{d_k}$ scaling in
u3 and behind temperature's effect on training stability.

Unit x1 uses this Jacobian to derive backprop through attention end to end.

## Temperature: one knob, same evidence

Temperature composes with the softmax derivative in the way you would expect:

$$\frac{\partial}{\partial z_i}\,\mathrm{softmax}(z/T)_i = \frac{1}{T}\, p_i(1 - p_i)$$

so low temperature shrinks gradients through two independent routes at once - the
explicit $1/T$ factor, and the saturation of $p_i(1-p_i)$ toward 0 as the
distribution sharpens.

Entropy makes the "flattening" claim quantitative. For the canon's
$z = (2, 1, 0)$, with $H(p) = -\sum_i p_i \ln p_i$ in nats:

| $T$ | $p$ | $H(p)$ |
|-----|-----|--------|
| 0.5 | (0.867, 0.117, 0.016) | 0.441 |
| 1.0 | (0.665, 0.245, 0.090) | 0.832 |
| 2.0 | (0.506, 0.307, 0.186) | 1.020 |

The uniform distribution over 3 outcomes has $H = \ln 3 = 1.099$, which is the
supremum approached as $T \rightarrow \infty$. Entropy is monotone increasing in
$T$ - that is the formal content of "temperature flattens the distribution", and
it is a theorem, not an observation about these particular numbers.

Two places the same knob appears elsewhere in the book, with the same math and
different intent. **Knowledge distillation** trains a student on a teacher's
softmax at $T > 1$, because the flattened distribution exposes the teacher's
relative rankings among non-top tokens (the "dark knowledge") that a sharp
distribution numerically discards. And **MoE routing** in unit u7 applies softmax
over router logits, where a low effective temperature produces confident routing
and a high one produces load-balanced but blurry assignment. Same function, three
jobs.

## From derivative to gradient

### The gradient as the representation of the differential

The clean definition drops the coordinate view. $f : \mathbb{R}^n \rightarrow
\mathbb{R}$ is differentiable at $x$ if there is a linear map $Df(x)$ with

$$f(x + h) = f(x) + Df(x)[h] + o(\lVert h\rVert)$$

$Df(x)$ is the **differential**, a linear functional (it eats a direction vector,
returns a scalar). By the Riesz representation theorem there is a unique vector
$g$ with $Df(x)[h] = g \cdot h$ for all $h$, and that vector is the gradient.
So the gradient is not primitive; it is the vector that represents the
differential under the inner product. Change the inner product and the gradient
changes, which is exactly the content of natural-gradient methods.

Steepest ascent falls out in one line. Among unit directions $h$,

$$Df(x)[h] = g \cdot h = \lVert g \rVert \cos\theta$$

which is maximized at $\theta = 0$, that is $h = g / \lVert g\rVert$. So the
gradient direction maximizes the directional derivative, and the maximum value is
$\lVert g \rVert$. Both canon claims, proved.

### Layout conventions, since they will bite you

For $f : \mathbb{R}^n \rightarrow \mathbb{R}^m$ the Jacobian $J$ is $m \times n$
with $J_{ij} = \partial f_i / \partial x_j$ (numerator layout). For scalar $f$,
$m = 1$, so the derivative is technically a $1 \times n$ row vector, while the
gradient is the $n \times 1$ column $\nabla f = (Df)^T$. Papers switch between
these silently. The reliable rule, and the one worth memorizing over any
convention: **the gradient of a scalar with respect to a tensor has the same
shape as that tensor.** If you have $\partial L / \partial W$ for
$W \in \mathbb{R}^{4096 \times 4096}$, it is $4096 \times 4096$. Any derivation
that produces a different shape has an error in it, and checking shapes catches
transposition mistakes before you run anything.

Three identities that cover most of what appears in this book, for
$W \in \mathbb{R}^{m \times n}$, $x \in \mathbb{R}^n$, symmetric
$A \in \mathbb{R}^{n \times n}$:

$$\frac{\partial (a \cdot x)}{\partial x} = a
\qquad \frac{\partial (x^T A x)}{\partial x} = 2Ax
\qquad \frac{\partial (Wx)}{\partial x} = W$$

and for a scalar loss $L$ with upstream gradient $\delta = \partial L/\partial(Wx)
\in \mathbb{R}^m$,

$$\frac{\partial L}{\partial W} = \delta\, x^{T} \in \mathbb{R}^{m \times n}
\qquad \frac{\partial L}{\partial x} = W^{T} \delta \in \mathbb{R}^{n}$$

That outer product $\delta x^T$ is the single most common expression in a
backward pass. Verify by shapes: $\delta$ is $m \times 1$, $x^T$ is $1 \times n$,
product is $m \times n$, matching $W$.

### Curvature and the zigzag

The canon says gradient descent zigzags in narrow valleys. The quantitative
statement uses the Hessian $H_{ij} = \partial^2 f / \partial x_i \partial x_j$.
For a quadratic $f(x) = \tfrac{1}{2} x^T H x$ with $H$ positive definite,
gradient descent with the optimal fixed step size converges at a rate governed by
the condition number $\kappa = \lambda_{\max}/\lambda_{\min}$:

$$\lVert x_t - x^{*}\rVert \leq \left(\frac{\kappa - 1}{\kappa + 1}\right)^{t} \lVert x_0 - x^{*}\rVert$$

With $H = \mathrm{diag}(2, 20)$, $\kappa = 10$ and the contraction factor is
$9/11 \approx 0.818$ per step. Real loss surfaces have $\kappa$ in the thousands
or worse, pushing the factor arbitrarily close to 1. Momentum improves the
dependence to $\sqrt{\kappa}$, and Adam's per-parameter scaling approximates a
diagonal preconditioner that reduces the effective $\kappa$ directly. That is the
mathematical content of unit u5's optimizer progression.

## The chain rule

### The multivariate statement

For $f : \mathbb{R}^n \rightarrow \mathbb{R}^m$ and $g : \mathbb{R}^m \rightarrow
\mathbb{R}^p$, the Jacobian of the composite is the matrix product of the
Jacobians:

$$J_{g \circ f}(x) = J_g(f(x)) \cdot J_f(x)$$

with shapes $(p \times m)(m \times n) = (p \times n)$. The scalar chain rule from
the canon is this with $m = n = p = 1$, where matrix multiplication degenerates to
scalar multiplication. Nothing new is happening; the types got richer.

When several paths connect a variable to the loss, the contributions **add**.
For $L$ depending on $u$ and $v$, both depending on $w$:

$$\frac{\partial L}{\partial w} = \frac{\partial L}{\partial u}\frac{\partial u}{\partial w} + \frac{\partial L}{\partial v}\frac{\partial v}{\partial w}$$

This is the total derivative, and it is what makes residual connections work: a
value written into the residual stream reaches the loss through the skip path and
through every downstream block, and all those gradients sum at the source. Unit
u4's "gradient highway" is literally the first term of this sum being the
identity.

### Why reverse mode, in one calculation

The composite $L = f_k \circ \cdots \circ f_1$ has Jacobian $J_k J_{k-1} \cdots J_1$.
Matrix product is associative, so you may bracket it either way, and the cost
differs enormously.

Forward mode computes $J_k(\cdots(J_2 J_1))$, propagating an $n \times n$ object
where $n$ is the parameter count. Reverse mode computes $((J_k J_{k-1})\cdots)J_1$,
and since $L$ is scalar, $J_k$ is $1 \times m$ - a row vector. Every intermediate
stays a vector. One backward sweep yields the gradient with respect to all $n$
parameters at a cost of roughly 2 to 3 forward passes, independent of $n$.

That asymmetry is the entire reason deep learning is computationally possible: a
loss is one scalar and parameters number in the billions, so you want the mode
that is cheap when outputs are few and inputs are many. The price is memory - you
must keep every forward intermediate for the backward pass, which is what
activation checkpointing trades back against recompute.

### The canon's example as a graph

$$w \xrightarrow{\ \partial z/\partial w = x\ } z \xrightarrow{\ \partial h/\partial z = 2z\ } h \xrightarrow{\ \partial L/\partial h = 2(h-y)\ } L$$

with $b$ joining at $z$ via $\partial z/\partial b = 1$. Reverse mode walks right
to left, carrying the accumulated scalar $\partial L / \partial(\text{node})$:

$$\bar{h} = -2 \quad\rightarrow\quad \bar{z} = \bar{h}\cdot 2z = -8 \quad\rightarrow\quad
\bar{w} = \bar{z}\cdot x = -16,\ \ \bar{b} = \bar{z}\cdot 1 = -8$$

$\bar{z} = -8$ is computed once and reused for both parameters that feed $z$.
That reuse, applied to a graph with billions of nodes, is backprop, and it is the
only difference between backprop and naive per-parameter differentiation.

Verify with central differences, $\big(L(w+h) - L(w-h)\big)/2h$ at $h = 10^{-6}$:
you get $-15.999999999$ for $w$ and $-8.000000000$ for $b$. This is the gradient
check every framework ships, and it is what you should reach for when a
hand-derived backward pass disagrees with reality.

## Loss surfaces and why SGD works anyway

### The saddle-point argument, made precise

At a critical point $\nabla L = 0$, the second-order behavior is governed by the
Hessian $H$, which is symmetric and therefore has $d$ real eigenvalues. Classify:
all positive means local minimum, all negative means local maximum, mixed signs
means saddle.

Take the crude model that the eigenvalue signs are independent fair coins. Then

$$\Pr[\text{local min}] = 2^{-d}$$

At $d = 10^9$ this is not small, it is zero for any practical purpose. The random
matrix theory result (Bray and Dean 2007, Dauphin et al. 2014) sharpens this:
critical points at high loss are overwhelmingly saddles with many negative
directions, and the fraction of negative eigenvalues decreases monotonically as
loss decreases. Local minima exist essentially only near the bottom of the loss
range, where they are all roughly equally good. Getting "stuck in a bad local
minimum" is a two-dimensional intuition that does not survive the dimension
count.

What actually slows training is different: **plateaus** where $\lVert \nabla L\rVert$
is tiny in every direction, and **ill-conditioning** where $\kappa$ is huge. Both
are curvature problems, not trap problems, and both are what momentum and
adaptive methods address.

The minibatch noise in SGD is not merely tolerated, it is functional. The update
is

$$\theta_{t+1} = \theta_t - \eta\big(\nabla L(\theta_t) + \xi_t\big)$$

where $\xi_t$ is zero-mean noise from sampling a batch instead of the full
dataset. Near a saddle, $\xi_t$ has a component along a negative-curvature
direction with probability essentially 1, so the iterate escapes. Near a sharp
minimum, the same noise ejects the iterate, while a wide flat basin retains it.
SGD is therefore implicitly biased toward flat minima, and flatness correlates
with generalization because a flat basin means the function is insensitive to
weight perturbation, which bounds how much the model can be relying on any
individual parameter setting.

### Permutation symmetry, quantified

Beyond finding different regions, different runs cannot match weights even in
principle, because the loss is invariant under a huge symmetry group. Permute the
$k$ hidden units of a layer and permute the next layer's incoming weights to
match, and the function computed is identical. For a single layer of width $k$
that is $k!$ equivalent settings. For $k = 4096$, $4096!$ exceeds $10^{13000}$.
Multiply across all layers. So the "global minimum", if one exists, is not a
point but an astronomically large orbit, and asking two runs to land on the same
representative of it is meaningless.

Recent work on linear mode connectivity finds that after accounting for
permutation, independently trained networks are often connected by low-loss
paths, which suggests the good regions may be far more connected than the
"isolated basins" picture implies.

### The floor, stated properly

Unit u5 develops this; here is the statement. For a true data distribution $q$
and model distribution $p$, cross-entropy decomposes exactly:

$$H(q, p) = H(q) + D_{\mathrm{KL}}(q \parallel p)$$

where $H(q) = -\sum_x q(x)\ln q(x)$ is the entropy of the data and
$D_{\mathrm{KL}} \geq 0$ is the Kullback-Leibler divergence, zero only when
$p = q$ exactly. Training can drive $D_{\mathrm{KL}}$ toward 0. It cannot touch
$H(q)$, which is a property of language and not of the model.

For a fair binary choice, $H(q) = \ln 2 = 0.693$ nats, or 1 bit. Empirical
estimates put the entropy of English text somewhere near 0.6 to 1.3 bits per
character depending on the estimate and the corpus. Chinchilla-style scaling laws
encode this directly as

$$L(N, D) = E + \frac{A}{N^{\alpha}} + \frac{B}{D^{\beta}}$$

where $N$ is parameters, $D$ is tokens, and $E$ is a fitted irreducible term - the
floor, appearing explicitly in the functional form. A model reporting training
loss materially below $E$ has memorized its training set, and its held-out loss
will show it.
