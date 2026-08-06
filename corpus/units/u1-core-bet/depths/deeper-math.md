## The dialect first

The four pieces below are stated with full precision in what follows: the dot
product as a bilinear form, matrix multiply as composition of linear maps, the
gradient as the direction of steepest ascent under the Euclidean metric, and
expectation as an integral against a measure you can only sample. If the
calibration series felt easy, skim for the formal statements and move on.

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
For $n = 100{,}000$ and $d = 4096$ that is roughly a $4{,}000$-fold difference
in the dominant term for a bit-identical result. Linear-attention variants are
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

## The objective, stated exactly

The chain rule of probability is not an assumption about language. It falls out
of the definition of conditional probability, $P(A \mid B) = P(A, B)/P(B)$,
applied repeatedly. For three variables:

$$P(x_1, x_2, x_3) = P(x_3 \mid x_1, x_2)\, P(x_1, x_2) = P(x_3 \mid x_1, x_2)\, P(x_2 \mid x_1)\, P(x_1)$$

By induction this extends to any $T$. Nothing about the ordering is forced: you
could factor right-to-left, or in any permutation of positions, and get an
equally exact identity. Left-to-right is chosen because it matches how text is
generated and consumed, and because it makes the causal mask (u3) a triangular
matrix instead of something worse.

**What the loss actually estimates.** Write $P^*$ for the true distribution over
next tokens given a context $c$, and $P_\theta$ for the model's. The training
loss on a corpus is a sample estimate of the expected negative log likelihood,

$$\mathbb{E}_{x \sim P^*}\left[-\log P_\theta(x \mid c)\right] = H(P^*, P_\theta)$$

the cross-entropy between the two distributions. Cross-entropy decomposes
exactly:

$$H(P^*, P_\theta) = H(P^*) + D_{KL}(P^* \,\|\, P_\theta)$$

where $H(P^*) = -\sum_v P^*(v)\log P^*(v)$ is the entropy of the true
distribution and $D_{KL}(P^*\|P_\theta) = \sum_v P^*(v)\log\frac{P^*(v)}{P_\theta(v)} \geq 0$
is the Kullback-Leibler divergence, which is 0 if and only if
$P_\theta = P^*$ everywhere.

Two consequences, both load-bearing later.

First, the *only* term the model can influence is the KL term. $H(P^*)$ is a
property of language, not of the model. This is the formal statement of why
training does not aim at zero loss (M18): the achievable minimum is
$H(P^*) > 0$, and every reported loss number should be read as "floor plus
excess."

Second, minimizing cross-entropy is exactly minimizing $D_{KL}(P^* \| P_\theta)$,
the *forward* KL. Forward KL is mode-covering: it charges $\infty$ for assigning
$P_\theta = 0$ where $P^* > 0$, but charges nothing for assigning mass where
$P^*$ has none. A language model trained this way is structurally biased toward
spreading mass over plausible-looking continuations rather than committing.
That asymmetry is one root of hallucination, and u5 returns to it.

**Per-position independence, not per-position isolation.** The gradient of the
total loss is the sum of per-position gradients,

$$\nabla_\theta \mathcal{L} = -\frac{1}{T}\sum_{t=1}^{T} \nabla_\theta \log P_\theta(x_t \mid x_{<t})$$

but $P_\theta(x_t \mid x_{<t})$ depends on shared parameters that also produce
every other position's prediction. The terms are additive; the computations that
produce them are entangled through $\theta$. This is precisely why an
independently-summed objective does not imply a myopic model.

## Why prediction forces world modeling

The compression claim is a theorem, so state it as one.

**Kraft-McMillan inequality.** For any uniquely decodable code over an alphabet
of size $V$ with codeword lengths $\ell_1, \ldots, \ell_V$ in bits,
$\sum_v 2^{-\ell_v} \leq 1$. Conversely, for any set of lengths satisfying that
inequality, a prefix code with those lengths exists. Setting
$\ell_v = \lceil -\log_2 Q(v)\rceil$ for any distribution $Q$ satisfies it, so
every probability model induces a code, and every code induces a probability
model. The two are the same object.

**Shannon source coding.** The expected code length under $Q$ when symbols are
drawn from $P^*$ is

$$\mathbb{E}_{P^*}[\ell] = -\sum_v P^*(v)\log_2 Q(v) = H_2(P^*) + D_{KL,2}(P^*\|Q)$$

minimized at $Q = P^*$ with value $H_2(P^*)$. Arithmetic coding achieves this
within 2 bits of the entire message, so the bound is not merely asymptotic.

Therefore, with $\mathcal{L}$ in nats:

$$\text{bits per token} = \frac{\mathcal{L}}{\ln 2}$$

A model at $\mathcal{L} = 2.0$ nats encodes its training corpus at 2.89 bits per
token. Against a token that averages ~4 characters of UTF-8 (32 bits raw), that
is a compression ratio above 11:1, and this is a real, checkable number: you can
build an arithmetic coder on top of any LM and observe the file shrink by that
factor. LLMs are, as a measured fact, the strongest general-purpose text
compressors that exist.

**Where the argument's force actually comes from.** Let $\mathcal{H}$ be a
hypothesis class (say, all functions expressible by a transformer of a given
size). Decompose the achieved loss:

$$\mathcal{L}_{\text{achieved}} = \underbrace{H(P^*)}_{\text{irreducible}} + \underbrace{\min_{h \in \mathcal{H}} D_{KL}(P^*\|h)}_{\text{approximation}} + \underbrace{\text{estimation}}_{\text{finite data}} + \underbrace{\text{optimization}}_{\text{SGD did not find the min}}$$

The compression argument constrains only the second term: for the approximation
error to be small, $\mathcal{H}$ must *contain* a function that models the
generating process, and the learned $h$ must be near it. It says nothing about
the third and fourth terms. Scaling laws (u5) are empirical claims that all three
non-irreducible terms shrink as smooth power laws in parameters, data, and
compute. That empirical regularity, not the coding theorem, is why the bet paid.

## Tokens: BPE from scratch

**What BPE optimizes, and what it does not.** The greedy merge rule - take the
most frequent adjacent pair - is a greedy heuristic for minimizing the total
token count of the corpus under a vocabulary budget. It is not optimal. Each
merge reduces corpus length by exactly the merged pair's count, so greedily
taking the max-count pair maximizes immediate length reduction, but merges
interact: taking pair $A$ can destroy the contexts in which pair $B$ would have
been frequent. The optimal vocabulary under a budget is NP-hard to find, and BPE
does not attempt it.

Formally, if $n_0$ is the corpus length in characters and merge $m_i$ has count
$c_i$ at the time it is applied, the corpus length after $k$ merges is

$$n_k = n_0 - \sum_{i=1}^{k} c_i$$

and the counts $c_i$ decay roughly as a power law in $i$, which is why vocabulary
size has sharply diminishing returns: going from 32k to 128k tokens buys
perhaps 10-15% fewer tokens per document while multiplying the embedding and
output matrices by 4.

**The alternative worth knowing.** Unigram LM tokenization (SentencePiece's
default, used by several model families) inverts the procedure: start from a
large candidate vocabulary, define a unigram distribution over pieces, and
iteratively *remove* the pieces whose removal costs the least corpus likelihood
under an EM procedure. It optimizes

$$\max_{\mathcal{V}} \sum_{\text{docs}} \log \sum_{\text{segmentations } s} \prod_{p \in s} P(p)$$

marginalizing over segmentations rather than fixing one greedily. Its practical
consequence is that a Unigram tokenizer can *sample* different segmentations of
the same string (subword regularization), which BPE cannot; the merge list is
deterministic.

**The digit-grouping arithmetic problem, quantified.** With `cl100k_base`,
integers are chunked into runs of up to three digits from the left. For a
$d$-digit number the chunk boundaries are at positions $3, 6, 9, \ldots$ from the
left, so the boundary positions relative to the *units digit* depend on
$d \bmod 3$. Adding a 4-digit and a 4-digit number aligns; adding a 4-digit and
a 5-digit number does not. The model must learn a distinct alignment procedure
for each residue class pair. Models that tokenize digits individually (Llama-2
does, PaLM did) show measurably better arithmetic for exactly this reason, at the
cost of longer sequences.

## Embeddings are coordinates, not contents

**The invariance argument, done properly.** Let $\Pi \in \{0,1\}^{d \times d}$ be
a permutation matrix, so $\Pi^\top \Pi = I$. Consider transforming a trained
model as follows: $E' = E\Pi^\top$ (permute the columns of the embedding matrix),
and for every weight matrix $W \in \mathbb{R}^{m \times d}$ that reads a vector
from the residual stream, set $W' = W\Pi^\top$; for every matrix
$U \in \mathbb{R}^{d \times n}$ that writes into the stream, set $U' = \Pi U$;
for every elementwise normalization gain $g \in \mathbb{R}^d$ and bias
$b \in \mathbb{R}^d$, set $g' = \Pi g$, $b' = \Pi b$.

Then at every point in the network the transformed activation is $\Pi x$ where
the original was $x$, and:

- Linear reads: $W'(\Pi x) = W\Pi^\top \Pi x = Wx$. Unchanged.
- LayerNorm: mean and variance are $\frac{1}{d}\sum_i x_i$ and
  $\frac{1}{d}\sum_i (x_i - \mu)^2$, both symmetric functions of the
  coordinates, hence invariant under permutation. The affine step
  $g \odot \hat{x} + b$ becomes $(\Pi g)\odot(\Pi\hat x) + \Pi b = \Pi(g \odot \hat x + b)$
  because permutation commutes with elementwise product when both operands are
  permuted identically.
- Elementwise nonlinearities: $\sigma(\Pi x) = \Pi\sigma(x)$ for any
  coordinatewise $\sigma$.
- Unembedding: $W_U' = W_U \Pi^\top$, so $W_U'(\Pi h) = W_U h$. Unchanged.

So logits are identical for every input, and $d!$ distinct parameter settings
implement the identical function. Any claim of the form "coordinate $k$ encodes
property $p$" is a claim about the arbitrary representative that SGD landed on.

Widening the group takes more care than it is usually given. Replace $\Pi$ by an
orthogonal $Q$ with $Q^\top Q = I$ and the linear reads and writes still cancel,
but the elementwise steps do not: a coordinatewise $\sigma$ does not commute with
a general rotation, and neither does LayerNorm, whose mean $\frac{1}{d}\sum_i x_i$
is the projection onto the all-ones direction and is preserved only by rotations
that fix that direction. RMSNorm is the exception - it divides by
$\lVert x \rVert / \sqrt{d}$, which is a function of the norm alone and is
therefore invariant under every orthogonal $Q$. So the clean statement is: strip
a network of all coordinatewise operations - gain-free RMSNorm, no pointwise
activation - and its invariance group is the full orthogonal group $O(d)$, in
which not even the coordinate axes are distinguished. Every coordinatewise
operation you add back collapses the group toward the permutations. Real
transformers sit at the collapsed end, since they have both an elementwise
normalization gain and a pointwise activation in every MLP. That collapse is
why interpretability work (x2) can sometimes find axis-aligned features at all,
but the useful features it finds are generally *directions*, not coordinates, and
are typically non-orthogonal and outnumber $d$ (superposition).

**The gradient on an embedding row is sparse.** $E$ enters the computation only
through the rows selected by the batch's token IDs. Writing $e_i = E[i,:]$, and
letting $\mathcal{T}_i$ be the set of positions in the batch where token $i$
appears,

$$\frac{\partial \mathcal{L}}{\partial e_i} = \sum_{t \in \mathcal{T}_i} \frac{\partial \mathcal{L}}{\partial h_t^{(0)}}$$

where $h_t^{(0)}$ is the layer-0 input at position $t$. Rows for tokens absent
from the batch receive exactly zero gradient. Token frequency in the corpus
therefore directly controls how many update steps a row receives: a token
appearing once per billion sees a handful of updates over a whole training run
and its embedding stays near its random initialization. This is the mechanism
behind "glitch tokens" - rare vocabulary entries (typically scraped artifacts
that survived into the vocabulary but were filtered out of the training corpus)
whose embeddings are essentially untrained noise, producing wildly
out-of-distribution behavior when they appear at inference.

Note also what this makes precise about M4: if embeddings were a lookup table of
meanings, a token seen once would still get a correct entry, the way a hash map
does. Instead it gets whatever the initializer wrote, because the only mechanism
available is gradient accumulation over occurrences.

## Counting the embedding matrix

**Where the crossover is.** A standard decoder-only transformer with $L$ layers
of width $d$, MLP expansion factor 4, and no bias terms has approximately

$$N_{\text{stack}} \approx L\left(\underbrace{4d^2}_{\text{attention } W_Q,W_K,W_V,W_O} + \underbrace{8d^2}_{\text{MLP up}+\text{down}}\right) = 12Ld^2$$

parameters, against $N_{\text{embed}} = Vd$ (or $2Vd$ untied). The embedding
fraction is

$$\frac{Vd}{Vd + 12Ld^2} = \frac{1}{1 + 12Ld/V}$$

which depends on $L$ and $d$ only through the product $Ld$, scaled by $V$. Check
it against the two models in canon:

- GPT-2 small: $L = 12$, $d = 768$, $V = 50257$. $12Ld/V = 12(12)(768)/50257 = 2.20$,
  giving $1/3.20 = 31\%$. Matches the measured 31%.
- Llama-2-7B: $L = 32$, $d = 4096$, $V = 32000$. $12Ld/V = 12(32)(4096)/32000 = 49.2$,
  giving $1/50.2 = 2.0\%$. Matches the measured 1.9%.

The formula makes the trend explicit: embedding share falls as $1/(Ld)$. For any
fixed vocabulary, scaling a model makes its embedding table asymptotically
negligible. The much-discussed move to 128k vocabularies costs almost nothing at
70B scale and would have been unaffordable at GPT-2 scale.

**A note on FLOPs versus parameters.** The embedding *lookup* costs no FLOPs - it
is an indexed read. The unembedding costs $2Vd$ FLOPs per token, which at
$V = 128\text{k}$, $d = 4096$ is 1.05 GFLOP per token, comparable to several
transformer layers. Parameter share and compute share are different accountings
and the embedding table sits at opposite ends of them: it is the cheapest
parameters in the model on the input side and among the most expensive
per-parameter on the output side.

## The output end: hidden state to logits

**Shift invariance, proved.** For $z \in \mathbb{R}^V$ and $c \in \mathbb{R}$,

$$\text{softmax}(z + c\mathbf{1})_j = \frac{e^{z_j + c}}{\sum_{k} e^{z_k + c}} = \frac{e^c e^{z_j}}{e^c \sum_k e^{z_k}} = \frac{e^{z_j}}{\sum_k e^{z_k}} = \text{softmax}(z)_j$$

So softmax is invariant to adding any constant to all logits, and the logit
vector is only determined up to $z + c\mathbf{1}$: a $V$-dimensional output with
$V - 1$ degrees of freedom. Implementations exploit this for numerical stability
by computing $\text{softmax}(z - \max_j z_j)$, which bounds every exponent at
most 0 and eliminates overflow at no cost to the result.

Inverting: given $P = \text{softmax}(z)$,

$$\log P_j = z_j - \log\sum_k e^{z_k} \quad\Longrightarrow\quad z_j = \log P_j + \log Z$$

with $Z = \sum_k e^{z_k}$ the partition function. Logits are log-probabilities
offset by the same unknown constant, which is why differences are exactly
log-odds ratios: $z_i - z_j = \log(P_i/P_j)$.

**Weight tying and its gradient.** When $W_U = E$, the shared matrix receives
gradient along two paths. Let $p = \text{softmax}(z)$ and let $y$ be the one-hot
true next token. The standard cross-entropy result (derived in u5) is
$\partial\mathcal{L}/\partial z = p - y$. Then the output-side contribution is

$$\left.\frac{\partial \mathcal{L}}{\partial E}\right|_{\text{out}} = (p - y)\, h^\top \in \mathbb{R}^{V \times d}$$

which is a *dense* rank-1 update touching every row, in contrast to the sparse
input-side gradient derived in the embeddings section. Tying therefore does more
than save parameters: it gives every vocabulary row a gradient signal at every
position, partially repairing the rare-token starvation problem. That is a real
argument for tying that the parameter-count framing misses.

The geometric reading of $(p - y)h^\top$: push the true token's row *toward* $h$
with weight $(1 - p_y)$, and push every other row *away* from $h$ with weight
$p_j$. Output embeddings are trained to be directions that the residual stream
points at when that token should come next. This is the precise sense in which
the "meaning" of a row is defined by the model's own hidden states and nothing
external.

**Why one matrix suffices.** Nothing forces the map from $h$ to logits to be
linear. It is linear because (a) $V$ is large, so any nonlinearity here is
expensive, and (b) the residual stream is already the output of a deep
nonlinear stack, so the last layer's job is only to *read out* a direction, not
to compute. The architectural bet is that the stack has already arranged $h$ so
that a linear probe suffices. Empirically it does, and the fact that a single
linear map recovers the answer is itself evidence about what the stack computes -
this is the premise the entire linear-probing branch of interpretability rests
on.

## What the bet actually claims

State the bet as a falsifiable proposition and check which parts have been
tested.

Let $\mathcal{L}(N, D)$ be the loss achieved by a model with $N$ parameters
trained on $D$ tokens. The empirical scaling law (u5) has the form

$$\mathcal{L}(N, D) = \mathcal{L}_\infty + \frac{A}{N^{\alpha}} + \frac{B}{D^{\beta}}$$

with $\mathcal{L}_\infty$ the irreducible entropy term and $\alpha, \beta$ small
positive exponents (roughly 0.34 and 0.28 in the Chinchilla fit). Three claims
sit on top of this:

1. **The functional form holds over many orders of magnitude.** Tested and true
   across roughly 8 orders of magnitude in compute. This is the strongest-
   supported part of the bet.
2. **Loss is a good proxy for capability.** Partly true and the weakest link.
   Loss is an average over all tokens; capabilities of interest live in thin
   slices of the distribution. Two models at identical loss can differ
   substantially on any given benchmark, and "emergent" jumps in downstream
   metrics at continuous loss improvements are largely an artifact of thresholded
   metrics rather than a discontinuity in the model.
3. **Nothing else is needed.** False as stated, and everyone including its
   proponents now agrees: post-training (u5) is required for usable behavior,
   and inference-time compute (chain of thought, search) buys capability that
   scaling parameters alone did not. The bet was that pretraining on next-token
   prediction produces the *substrate*. That much has held.

The honest summary: next-token prediction turned out to be a sufficient objective
for acquiring an extraordinary amount of structure, and an insufficient objective
for producing a system anyone wants to use directly. Both halves are true and
u5 is where they get reconciled.
