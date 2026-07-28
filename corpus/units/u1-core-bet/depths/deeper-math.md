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
