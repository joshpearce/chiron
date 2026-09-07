---
unit: u3
title: "Attention from first principles"
concepts:
  - c-qkv
  - c-sdpa
  - c-attn-numeric
  - c-causal-mask
assumes:
  - c-notation
  - c-matmul
  - c-dotprod
  - c-tokens
  - c-embeddings
  - c-matrix-transform
  - c-softmax
---

This is the unit the book exists to reach. By the end you will have computed
attention by hand on real numbers, derived every piece of the equation from a
requirement rather than memorized it, and be able to say exactly what the
causal mask forbids and why.

## The problem: mix information across a sequence with no recurrence

Whether `bank` means a riverbank or a financial institution is decided by
other tokens in the sentence - `river` three words back, or `approved the
loan` just after. But u1 ended with each position carrying a vector built
from that position's token alone: the embedding row for `bank` is the same
vector in both sentences, and nothing computed so far has looked sideways.
Every fact a token needs in order to mean anything sits in *other* positions.
Some operation has to move information between positions.

Give the data a name before designing that operation. You have a sequence of
$n$ token vectors. Call the stack of them $X$, a matrix of shape
$n \times d_{model}$, where $n$ is the number of tokens and $d_{model}$ is the
width of each token's vector (its dimensionality). Row $i$ of $X$, written
$x_i$, is a $d_{model}$-dimensional vector holding everything known about
position $i$ so far - and right now, that means position $i$'s token and
nothing else.

The constraints on that operation are what make the problem interesting:

1. **Variable $n$.** The same weights must handle 3 tokens and 300,000 tokens.
   So the operation cannot have a parameter per position, and cannot have a
   parameter per pair of positions.
2. **Differentiable everywhere.** The whole model is trained by gradient
   descent. Any hard selection ("take token 4, ignore the rest") has zero
   gradient almost everywhere and cannot be learned.
3. **Content-addressed, not position-addressed.** Which token matters depends
   on what the tokens *are*, not on where they sit. "The animal didn't cross
   the street because *it* was too tired" - resolving "it" means finding
   "animal", wherever it happens to be.
4. **No recurrence.** Recurrence (position $i$ reads the output of position
   $i-1$) satisfies 1-3 and is exactly what RNNs do. It is banned here for a
   systems reason you will recognize: it serializes. A dependency chain of
   length $n$ cannot be parallelized across a GPU, so training throughput
   collapses on long sequences.

Those four constraints are most of the answer. Before you read on, build the
mechanism yourself.

```beat
id: u3-b1
type: self-explain
concept: c-qkv
prompt: |
  Design it. You have $n$ token vectors $x_1, \dots, x_n$, each of dimension
  $d_{model}$. You must produce $n$ output vectors where output $i$ depends on
  all $n$ inputs. You may use matrix multiplies, elementwise functions, and
  softmax. You may not use recurrence, and you may not use any parameter whose
  shape depends on $n$.

  Commit before you read on: which of these operations satisfies all four
  constraints, and would you have written it down?
options:
  - text: 'Score every pair of positions with a dot product of linear projections of their vectors, softmax each row of scores into non-negative weights summing to 1, and set output $i$ to the weighted average of the projected input vectors using row $i$ of those weights.'
    correct: true
    explain: 'This is attention. The score is computed from content, so which position matters depends on what the tokens are; the softmax keeps it differentiable everywhere; and every parameter is a fixed $d_{model} \times d_k$ matrix, so nothing in the mechanism grows with $n$.'
  - text: 'Carry a running summary vector along the sequence: output $i$ is a learned function of $x_i$ and the summary produced at position $i-1$, so information reaches position $i$ from everything before it.'
    misconception: M9
    explain: 'This is recurrence, banned by constraint 4. A dependency chain of length $n$ serializes: position $i$ cannot start until position $i-1$ finishes, so the operation cannot be spread across a GPU. Within a transformer forward pass no position ever waits on another.'
  - text: 'For each position $i$, score the other positions on relevance and take the single highest-scoring one: output $i$ is the vector of the position it selects.'
    misconception: M1
    explain: 'A hard selection has zero gradient almost everywhere, so nothing about how to select could ever be learned, which is constraint 2. The mechanism has to be a soft weighted average with no branch and no chosen winner.'
  - text: 'Learn a weight matrix $W$ of shape $n \times n$ whose entry $(i,j)$ records how much position $j$ contributes to output $i$, and compute $O = WX$.'
    misconception: M4
    explain: 'That parameter is a stored table whose shape depends on $n$, so the same weights cannot serve 3 tokens and 300,000, which is constraint 1. The weights have to be computed from the content at run time rather than looked up per position pair.'
check: choice
```

Whatever you wrote, the rest of this unit builds the specific version that
won. Do not skip forward to check whether you were right - the value of having
written something down is already banked, and it is larger if you were wrong.

## Queries, keys, and values

You probably think attention *looks at* the important tokens, or *decides*
what to focus on. The word "attention" invites it, and every popular
explainer leans on the spotlight metaphor.

<!-- refutes: M1 -->

Here is the prediction that model makes, and the experiment that kills it.
If attention decides what is important, then it must be inspecting the
content it selects. So: run a forward pass, record the attention weights.
Now replace every value vector - the actual content being mixed - with
random noise, leave everything else untouched, and run it again. A
mechanism that inspects content and selects the important parts should
produce different weights, because the content it was inspecting is now
garbage.

The weights are bit-identical. Every one of them.

They are identical because the weights are computed from two projections of
the input, and the content being averaged is a third, and the third never
enters the weight computation. Attention is a similarity-weighted average.
The similarity half and the averaged half are separate. Nothing looks at
anything.

The reason the agentive picture is appealing is that it is *predictively
useful at the level of behavior*. Trained attention weights really do
concentrate on the tokens a human would call relevant - that is what the
training objective rewards. Describing the outcome as "the model attended to
the subject of the sentence" is a fine summary of a weight distribution. It
is a catastrophic description of the mechanism, because it hides the fact
that the mechanism is fixed, dumb, and has no branch in it.

What is actually there is three linear projections of the same input.

$$Q = X W^Q \qquad K = X W^K \qquad V = X W^V$$

- $X$ is the input, shape $n \times d_{model}$: one row per token.
- $W^Q$, $W^K$ are learned matrices of shape $d_{model} \times d_k$, where
  $d_k$ is the *head dimension* - the width of the space in which similarity
  gets measured. $d_k$ is a design choice, typically much smaller than
  $d_{model}$ (64 and 128 are common).
- $W^V$ is a learned matrix of shape $d_{model} \times d_v$. $d_v$ is the
  width of the information actually being moved. Usually $d_v = d_k$, and
  this unit assumes that, but nothing requires it.
- $Q$ (shape $n \times d_k$), $K$ (shape $n \times d_k$), and $V$ (shape
  $n \times d_v$) therefore all have exactly $n$ rows: one per token. Every
  token produces a query, a key, and a value. There is no separate store.

The names are borrowed from retrieval, and they carry real meaning:

- **Query** $q_i$ (row $i$ of $Q$, a $d_k$-vector): what position $i$ is
  looking for. A projection of position $i$'s own current content.
- **Key** $k_j$ (row $j$ of $K$, a $d_k$-vector): what position $j$ offers as
  a matching surface - the thing queries get compared against.
- **Value** $v_j$ (row $j$ of $V$, a $d_v$-vector): what position $j$
  contributes to whoever matches it. The payload.

Splitting key from value is the load-bearing design decision, and it is the
one most explanations skip. A token can be *found* by one property and
*contribute* a different one. The token "Paris" can present a key that reads
"I am a capital city, ask me about geography" while its value carries "France,
Seine, 48.85N". Matching and payload are decoupled because they are different
matrices, trained by different gradients.

```beat
id: u3-b2
type: predict
concept: c-qkv
prompt: |
  Two experiments on a trained attention layer. Predict the effect of each on
  (a) the attention weight matrix, and (b) the layer output, and commit to a
  prediction before reading on.

  1. Replace $W^V$ with random values. $W^Q$ and $W^K$ untouched.
  2. Replace $W^K$ with random values. $W^Q$ and $W^V$ untouched.
options:
  - text: 'Experiment 1: the weights are bit-identical and the output is garbage. Experiment 2: the weights are destroyed, and the output is an arbitrary convex combination of the true value vectors.'
    correct: true
    explain: 'Right, and the asymmetry is the whole point. The weights are $\text{softmax}(QK^T/\sqrt{d_k})$, in which $W^V$ never appears, so the content being averaged changed completely and not one weight moved. In experiment 2 the scores are dots against random keys, but the payloads are still the real $v_j$, so the output stays inside their convex hull.'
  - text: 'Both experiments change the weights, because attention inspects the content it selects and in experiment 1 that content is now noise.'
    misconception: M1
    explain: 'Nothing inspects anything. The weights come from $Q$ and $K$ alone; the values enter only at the final multiply $AV$. Run the experiment and the recorded weights are identical to the last decimal.'
  - text: 'Experiment 1 changes the weights, because $W^V$ builds the store the queries are matched against; experiment 2 leaves the weights alone, because keys are only an index into that store.'
    misconception: U3-M1
    explain: 'There is no store. $Q$, $K$ and $V$ are three projections of the same $X$, each with $n$ rows. The key is the matching surface and the value is the payload, so $S_{ij} = q_i \cdot k_j$ is destroyed by randomizing $W^K$ and cannot be touched by randomizing $W^V$.'
  - text: 'Experiment 1 leaves both the weights and the output essentially intact, since the attention map is what the layer produces; experiment 2 destroys both.'
    misconception: U3-M4
    explain: 'The layer produces $AV$, shape $n \times d_v$, not the $n \times n$ map. In experiment 1 the map survives untouched and the output is noise, because the output is an average of the value vectors that were just replaced.'
check: choice
```

## Scaled dot-product attention: the equation

Score every query against every key, normalize each query's scores into
weights, use the weights to average the values. Written out:

$$\text{Attention}(Q, K, V) = \text{softmax}\left(\frac{QK^T}{\sqrt{d_k}}\right) V$$

Every piece, with shapes:

| Expression | Shape | What it is |
|---|---|---|
| $Q$ | $n \times d_k$ | one query vector per token |
| $K$ | $n \times d_k$ | one key vector per token |
| $K^T$ | $d_k \times n$ | keys transposed so the multiply contracts over $d_k$ |
| $QK^T$ | $n \times n$ | entry $(i,j)$ is $q_i \cdot k_j$, the raw compatibility of query $i$ with key $j$ |
| $QK^T / \sqrt{d_k}$ | $n \times n$ | the same scores, divided by a constant |
| $\text{softmax}(\cdot)$ | $n \times n$ | applied **row-wise**: each row becomes non-negative and sums to 1 |
| $V$ | $n \times d_v$ | one value vector per token |
| the product | $n \times d_v$ | one output vector per token |

Three details that are wrong in most people's first reading:

**The softmax is row-wise, not over the whole matrix.** Row $i$ of the
$n \times n$ matrix holds query $i$'s scores against all $n$ keys. Softmaxing
that row gives query $i$ a probability-shaped distribution over positions.
Softmaxing over columns instead would normalize across queries, which means
how much position $i$ reads from position $j$ would depend on what unrelated
queries wanted. Rows sum to 1. Columns do not, and there is no reason they
should.

**The $n \times n$ matrix is not the output.** It is the weight matrix. Call
it $A$, for attention weights. The output is $AV$, of shape $n \times d_v$ -
same number of rows as the input, one vector per token, exactly like every
other layer in the network. The $n \times n$ object is an intermediate that
gets consumed and thrown away. This distinction is the single most common
shape confusion in the topic.

**The output rows are convex combinations of the value rows.** Row $i$ of $A$
is non-negative and sums to 1, so output row $i$ is a weighted average of the
rows of $V$ with weights summing to 1. Geometrically, every output vector
lies inside the convex hull of the value vectors. Attention cannot produce
anything outside that hull - it can only interpolate among the payloads
present in the sequence. That is a strong constraint, and it is a free
correctness check on any hand computation: if your output has a component
larger than the largest value in that column of $V$, you have made an
arithmetic error.

<!-- fade: sdpa -->

The full procedure, start to finish, on an input $X$ of shape
$n \times d_{model}$:

1. **Project.** $Q = X W^Q$ ($n \times d_k$), $K = X W^K$ ($n \times d_k$),
   $V = X W^V$ ($n \times d_v$). Three independent matrix multiplies against
   the same input.
2. **Score.** $S = QK^T$, shape $n \times n$. Entry $S_{ij} = q_i \cdot k_j$,
   a sum of $d_k$ products.
3. **Scale.** $S' = S / \sqrt{d_k}$. Divide every entry by the same scalar
   constant. The next section is entirely about why.
4. **Mask** (causal models only). Set $S'_{ij} = -\infty$ for every $j > i$.
   The causal-masking section below covers it. Skip it for now.
5. **Normalize.** $A = \text{softmax}(S')$, row-wise:
   $A_{ij} = e^{S'_{ij}} / \sum_{m} e^{S'_{im}}$, where the sum runs over all
   $n$ columns of row $i$. Each row is now non-negative and sums to 1.
6. **Aggregate.** $O = AV$, shape $n \times d_v$. Output row $i$ is
   $\sum_j A_{ij} v_j$: the weighted average of the value vectors.

Six steps, four of which are matrix multiplies or elementwise arithmetic.
There is no control flow anywhere in that list.

## Why divide by the square root of $d_k$

Step 3 divides by $\sqrt{d_k}$ and nothing else in the equation is that
arbitrary-looking. It is not arbitrary, and it is not about floating point.

Take the scores as random variables. During early training, and as a decent
model of a healthy network throughout, treat the components of $q_i$ and $k_j$
as independent, mean 0, variance 1. Then

$$S_{ij} = q_i \cdot k_j = \sum_{m=1}^{d_k} q_{im} k_{jm}$$

is a sum of $d_k$ independent terms. Each term $q_{im}k_{jm}$ is a product of
two independent mean-0, variance-1 variables, so it has mean
$E[q]E[k] = 0$ and variance $E[q^2]E[k^2] = 1$. Variance adds over
independent terms, so

$$\text{Var}(S_{ij}) = d_k, \qquad \text{sd}(S_{ij}) = \sqrt{d_k}$$

The typical magnitude of a raw score grows as $\sqrt{d_k}$. At $d_k = 64$ the
scores in a row are spread over a range of roughly $\pm 8$; at $d_k = 512$,
roughly $\pm 22$.

Now feed that into softmax. Softmax exponentiates, so what matters is the
*difference* between scores. Consider only two positions whose raw scores
differ by 10 - unremarkable at $d_k = 64$, where the standard deviation is 8.
Their weights are

$$\frac{e^{10}}{e^{10} + e^{0}} = \frac{22026.5}{22027.5} = 0.9999546,
\qquad \text{the other} = 0.0000454$$

That is a hard argmax wearing a softmax costume. And here is why it is fatal
rather than merely inelegant: the derivative of a softmax output with respect
to its own score is $p(1-p)$, which at $p = 0.9999546$ equals
$4.54 \times 10^{-5}$. The gradient through this layer is effectively zero.
The layer has stopped learning before it learned anything, because the
saturation happened at initialization.

Divide by $\sqrt{d_k} = 8$ first. The gap of 10 becomes 1.25, and the weights
become

$$\frac{e^{1.25}}{e^{1.25} + e^{0}} = 0.7773, \qquad \text{the other} = 0.2227$$

with derivative $p(1-p) = 0.1731$ - about 3800 times larger. The layer can
learn.

Dividing by $\sqrt{d_k}$ restores the score distribution to roughly unit
variance regardless of head dimension, which keeps softmax in the region
where it has usable gradients. That is the entire argument, and it explains
the exact form: $\sqrt{d_k}$, not $d_k$, because standard deviation is the
square root of variance and it is standard deviation that sets the spread.

Two things this is *not*. It is not numerical overflow protection - production
softmax implementations subtract the row max before exponentiating, which
handles overflow completely and independently, and the scaling would still be
needed if floats had infinite range. It is not normalizing the vectors to unit
length either - $\sqrt{d_k}$ is a fixed constant, identical for every entry of
every row, not a per-vector norm. It does not change which score is largest,
only how sharply softmax separates them.

```beat
id: u3-b3
type: predict
concept: c-sdpa
prompt: |
  We have scores $S = QK^T$. Before reading on: what specifically goes wrong
  if we softmax $S$ directly when $d_k = 512$, what is the fix, and does a
  colleague fix it instead by switching the whole layer to float64?
options:
  - text: 'Scores have variance $d_k$, so at $d_k = 512$ they spread over roughly $\pm 22$; the softmax saturates, its local derivative $p(1-p)$ collapses toward zero, and the layer stops learning at initialization. Divide by $\sqrt{d_k}$ before the softmax. float64 does not help.'
    correct: true
    explain: 'Right. The variance of a dot product of $d_k$-dimensional vectors grows with the dimension, pushing softmax into its flat region. And $e^{22}$ is a perfectly representable float32: a saturated softmax has a near-zero derivative in exact arithmetic, so wider floats change nothing.'
  - text: 'The exponentials overflow at that magnitude, so the fix is wider floats or an overflow-safe softmax; float64 does work.'
    misconception: M7
    explain: 'Overflow is real but is handled separately and completely by subtracting the row max before exponentiating, and the scaling would still be needed if floats had infinite range. The scaling exists for the gradients, not the numerics.'
  - text: 'Nothing goes wrong: softmax is scale-invariant, so the magnitude of the scores does not matter, and the precision question is moot.'
    misconception: M2
    explain: 'Softmax is shift-invariant, not scale-invariant. Multiplying every score by ten sharpens the distribution toward a hard argmax; a gap of 10 between two scores already gives weights of $0.9999546$ and $0.0000454$.'
  - text: 'The scores are simply un-normalized, so divide each row by its largest entry, or each $q_i$ by its norm, to bring them into range; float64 would not fix that.'
    misconception: U3-M2
    explain: 'That is a normalization read off the data. The correction here is a fixed variance correction chosen from the architecture alone: raw dot products have variance $d_k$, so the divisor is the standard deviation $\sqrt{d_k}$, the same constant for a row of tiny scores and a row of enormous ones.'
check: choice
```

## The whole computation by hand

Three tokens, $d_{model} = 4$, $d_k = d_v = 2$. Every number below is exact
or given to six decimal places. Work it yourself as you read; the checks
after each stage catch errors before they propagate.

The input, one row per token:

$$X = \begin{bmatrix} 1 & 0 & 1 & 0 \\ 0 & 1 & 1 & 0 \\ 1 & 1 & 0 & 1 \end{bmatrix}
\quad (3 \times 4)$$

The three learned projection matrices, each $4 \times 2$ (that is
$d_{model} \times d_k$):

$$W^Q = \begin{bmatrix} 1 & 0 \\ 0 & 1 \\ 1 & 0 \\ 0 & 1 \end{bmatrix}
\qquad
W^K = \begin{bmatrix} 0 & 1 \\ 1 & 0 \\ 1 & 0 \\ 0 & 1 \end{bmatrix}
\qquad
W^V = \begin{bmatrix} 1 & 0 \\ 0 & 2 \\ 0 & 1 \\ 1 & 0 \end{bmatrix}$$

Real weights are dense floats. These are 0s, 1s, and one 2 so the arithmetic
stays in your head, and the structure stays visible.

### Step 1: project

$Q = XW^Q$. Row $i$ of $Q$ is $x_i$ times $W^Q$, which - because $x_i$ has
0/1 entries - is the sum of the rows of $W^Q$ selected by the 1s in $x_i$.

Token 1, $x_1 = [1, 0, 1, 0]$, selects rows 1 and 3 of $W^Q$:
$[1,0] + [1,0] = [2, 0]$.
Token 2, $x_2 = [0,1,1,0]$, selects rows 2 and 3: $[0,1] + [1,0] = [1,1]$.
Token 3, $x_3 = [1,1,0,1]$, selects rows 1, 2, and 4:
$[1,0] + [0,1] + [0,1] = [1,2]$.

$$Q = \begin{bmatrix} 2 & 0 \\ 1 & 1 \\ 1 & 2 \end{bmatrix} \quad (3 \times 2)$$

The same procedure against $W^K$ and $W^V$:

$$K = \begin{bmatrix} 1 & 1 \\ 2 & 0 \\ 1 & 2 \end{bmatrix}
\qquad
V = \begin{bmatrix} 1 & 1 \\ 0 & 3 \\ 2 & 2 \end{bmatrix}
\qquad (3 \times 2 \text{ each})$$

**Check:** all three came out $3 \times 2$. They must - $(3 \times 4)$ times
$(4 \times 2)$ contracts the 4 and leaves $3 \times 2$. If any of your three
has a different shape, you multiplied in the wrong order.

```beat
id: u3-b4
type: compute
concept: c-attn-numeric
prompt: |
  Verify one entry yourself rather than taking the table on faith. Using
  $x_3 = [1, 1, 0, 1]$ and

  $$W^V = \begin{bmatrix} 1 & 0 \\ 0 & 2 \\ 0 & 1 \\ 1 & 0 \end{bmatrix}$$

  compute $v_3 = x_3 W^V$ and report its **first** component.

  Mechanical check before you answer: the first component of $v_3$ is a sum of
  entries from column 1 of $W^V$, whose entries are 1, 0, 0, 1 - so the answer
  cannot exceed 2.
answer: 2
check: numeric(0.001)
```

### Step 2: score

$S = QK^T$, shape $3 \times 3$. Entry $S_{ij} = q_i \cdot k_j$, a dot product
of two 2-vectors.

Row 1, $q_1 = [2, 0]$:

- $S_{11} = [2,0] \cdot [1,1] = 2\cdot1 + 0\cdot1 = 2$
- $S_{12} = [2,0] \cdot [2,0] = 4 + 0 = 4$
- $S_{13} = [2,0] \cdot [1,2] = 2 + 0 = 2$

Row 2, $q_2 = [1, 1]$:

- $S_{21} = [1,1] \cdot [1,1] = 1 + 1 = 2$
- $S_{22} = [1,1] \cdot [2,0] = 2 + 0 = 2$
- $S_{23} = [1,1] \cdot [1,2] = 1 + 2 = 3$

Row 3, $q_3 = [1, 2]$:

- $S_{31} = [1,2] \cdot [1,1] = 1 + 2 = 3$
- $S_{32} = [1,2] \cdot [2,0] = 2 + 0 = 2$
- $S_{33} = [1,2] \cdot [1,2] = 1 + 4 = 5$

$$S = QK^T = \begin{bmatrix} 2 & 4 & 2 \\ 2 & 2 & 3 \\ 3 & 2 & 5 \end{bmatrix}$$

**Check:** $S$ is $3 \times 3$ - it is $n \times n$, independent of $d_k$ and
of $d_{model}$. Note also that $S$ is *not* symmetric: $S_{12} = 4$ but
$S_{21} = 2$. Symmetry would require $q_i \cdot k_j = q_j \cdot k_i$, which
fails because $W^Q \neq W^K$. Attention is directional. Position 1 finding
position 2 highly compatible says nothing about how position 2 rates
position 1. If your $S$ came out symmetric, you used the same matrix twice.

```beat
id: u3-b5
type: compute
concept: c-attn-numeric
prompt: |
  From $Q = \begin{bmatrix} 2 & 0 \\ 1 & 1 \\ 1 & 2 \end{bmatrix}$ and
  $K = \begin{bmatrix} 1 & 1 \\ 2 & 0 \\ 1 & 2 \end{bmatrix}$, compute the
  raw score $S_{33} = q_3 \cdot k_3$ - the score of token 3's query against
  token 3's own key.

  Report the raw dot product, before any scaling.
answer: 5
check: numeric(0.001)
```

Notice $S_{33} = 5$ is the largest entry in the whole matrix. Self-scores are
often large, because $q_i$ and $k_i$ are two projections of the same $x_i$ and
therefore tend to point in related directions. Nothing enforces this - it is
a tendency, not a rule - and a trained model routinely learns heads whose
diagonal is suppressed.

### Step 3: scale

Divide every entry by $\sqrt{d_k} = \sqrt{2} = 1.414214$. Equivalently,
multiply by $0.707107$.

$$S' = \frac{S}{\sqrt{2}} = \begin{bmatrix}
1.414214 & 2.828427 & 1.414214 \\
1.414214 & 1.414214 & 2.121320 \\
2.121320 & 1.414214 & 3.535534
\end{bmatrix}$$

**Check:** the same constant divides every entry, so the *ordering* within
each row is unchanged and the ratio of any two entries is unchanged. Row 1
was $[2, 4, 2]$ with the middle largest; it still is. Entries that were equal
are still equal.

### Step 4: mask

Skipped. This section computes the unmasked (bidirectional) case. The
causal-masking section below redoes the same numbers with the mask applied.

### Step 5: normalize

Softmax each row. There are only four distinct values in $S'$, so exponentiate
once each:

| $S'$ entry | $e^{S'}$ |
|---|---|
| $1.414214$ | $4.113250$ |
| $2.121320$ | $8.342145$ |
| $2.828427$ | $16.918829$ |
| $3.535534$ | $34.313330$ |

**Row 1**, entries $[1.414214,\ 2.828427,\ 1.414214]$:

$$e^{S'} = [4.113250,\ 16.918829,\ 4.113250], \qquad \text{sum} = 25.145329$$

$$A_1 = \left[\frac{4.113250}{25.145329},\ \frac{16.918829}{25.145329},\ \frac{4.113250}{25.145329}\right]
= [0.163579,\ 0.672842,\ 0.163579]$$

**Row 2**, entries $[1.414214,\ 1.414214,\ 2.121320]$:

$$e^{S'} = [4.113250,\ 4.113250,\ 8.342145], \qquad \text{sum} = 16.568645$$

$$A_2 = [0.248255,\ 0.248255,\ 0.503490]$$

**Row 3**, entries $[2.121320,\ 1.414214,\ 3.535534]$:

$$e^{S'} = [8.342145,\ 4.113250,\ 34.313330], \qquad \text{sum} = 46.768725$$

$$A_3 = [0.178370,\ 0.087949,\ 0.733681]$$

$$A = \begin{bmatrix}
0.163579 & 0.672842 & 0.163579 \\
0.248255 & 0.248255 & 0.503490 \\
0.178370 & 0.087949 & 0.733681
\end{bmatrix}$$

**Checks, all three of which you can do without a calculator:**

1. Every row sums to $1.000000$. Rows only - the columns sum to $0.590204$,
   $1.009046$, and $1.400750$, which are not 1 and carry no meaning.
2. Row 1 has $A_{11} = A_{13}$ exactly, because $S_{11} = S_{13} = 2$. Equal
   scores give equal weights, always.
3. Row 2's first two entries are equal for the same reason, and the largest
   weight in each row sits where that row's largest score was.

```beat
id: u3-b6
type: compute
concept: c-attn-numeric
prompt: |
  Row 2 of the scaled scores is $[1.414214,\ 1.414214,\ 2.121320]$, and

  $$e^{1.414214} = 4.113250, \qquad e^{2.121320} = 8.342145$$

  Compute $A_{23}$ - how much token 2 weights token 3's value. Give it to
  three decimal places.

  Mechanical check before answering: two of the three scores are equal and
  smaller than the third, so $A_{23}$ must be greater than $1/3$ and less
  than $1$. Note that the bound does not hand you the answer - "the two equal
  ones split half and the big one takes half" gives $0.5$, and that is wrong.
  Exponentiate and normalize.
answer: 0.503490
check: numeric(0.002)
```

### Step 6: aggregate

$O = AV$, shape $3 \times 2$. Output row $i$ is
$A_{i1}v_1 + A_{i2}v_2 + A_{i3}v_3$, with

$$v_1 = [1, 1], \qquad v_2 = [0, 3], \qquad v_3 = [2, 2]$$

**Row 1**, weights $[0.163579,\ 0.672842,\ 0.163579]$:

- component 1: $0.163579(1) + 0.672842(0) + 0.163579(2) = 0.163579 + 0 + 0.327158 = 0.490737$
- component 2: $0.163579(1) + 0.672842(3) + 0.163579(2) = 0.163579 + 2.018525 + 0.327158 = 2.509263$

**Row 2**, weights $[0.248255,\ 0.248255,\ 0.503490]$:

- component 1: $0.248255 + 0 + 1.006980 = 1.255235$
- component 2: $0.248255 + 0.744765 + 1.006980 = 2.000000$

**Row 3**, weights $[0.178370,\ 0.087949,\ 0.733681]$:

- component 1: $0.178370 + 0 + 1.467362 = 1.645732$
- component 2: $0.178370 + 0.263846 + 1.467362 = 1.909579$

$$O = \begin{bmatrix}
0.490737 & 2.509263 \\
1.255235 & 2.000000 \\
1.645732 & 1.909579
\end{bmatrix} \quad (3 \times 2)$$

**Checks:**

1. **Shape.** $O$ is $n \times d_v = 3 \times 2$. One output vector per token,
   same count as the input. The $3 \times 3$ matrix $A$ is gone - it was
   scaffolding.
2. **Convex hull.** Column 1 of $V$ holds $\{1, 0, 2\}$, so every entry in
   column 1 of $O$ must lie in $[0, 2]$: $0.490737$, $1.255235$, $1.645732$
   all do. Column 2 of $V$ holds $\{1, 3, 2\}$, so column 2 of $O$ must lie
   in $[1, 3]$: $2.509263$, $2.000000$, $1.909579$ all do. Any output entry
   outside its column's range is an arithmetic error, guaranteed.
3. **The exact 2.** $O_{22} = 2.000000$ is exact, not rounded, and you can
   prove it without the decimals. Row 2 has $A_{21} = A_{22}$; call that
   common weight $w$, so $A_{23} = 1 - 2w$. Column 2 of $V$ is $[1, 3, 2]$,
   so $O_{22} = w(1) + w(3) + (1-2w)(2) = 4w + 2 - 4w = 2$. The weights
   cancel entirely. If your row-2 second component is not exactly 2, the error
   is upstream in the softmax.

```beat
id: u3-b7
type: compute
concept: c-attn-numeric
prompt: |
  Using $A_3 = [0.178370,\ 0.087949,\ 0.733681]$ and

  $$v_1 = [1, 1], \qquad v_2 = [0, 3], \qquad v_3 = [2, 2]$$

  compute the **first** component of output row 3, $O_{31}$.

  Mechanical check before answering: column 1 of $V$ is $[1, 0, 2]$, so your
  answer must lie between 0 and 2, and because most of the weight sits on
  $v_3$ (whose first component is 2), it should land well above 1.
answer: 1.645732
check: numeric(0.005)
```

That is the whole mechanism. Six matrix operations, no branches, no loops
over positions, no state.

```beat
id: u3-b8
type: completion
concept: c-sdpa
prompt: |
  Same procedure, new numbers, three blanks. Two tokens, $d_k = d_v = 2$.
  You are given the projections already computed:

  $$Q = \begin{bmatrix} 1 & 1 \\ 0 & 2 \end{bmatrix}
  \qquad K = \begin{bmatrix} 1 & 0 \\ 1 & 1 \end{bmatrix}
  \qquad V = \begin{bmatrix} 2 & 0 \\ 0 & 4 \end{bmatrix}$$

  Step 2, score. $S = QK^T =$ `____(a)`  (a $2 \times 2$ matrix of integers)

  Step 3, scale. Divide every entry of $S$ by `____(b)`.

  Step 5, normalize. Using $e^{0} = 1$, $e^{0.707107} = 2.028115$,
  $e^{1.414214} = 4.113250$: row 1 of $A$ is $[0.330238,\ 0.669762]$; row 2
  of $A$ is $[0.195570,\ 0.804430]$.

  Step 6, aggregate. Output row 1 of $O$ is `____(c)` - the expression in
  terms of $A$ and the rows of $V$, evaluated to two decimals.

  Which filling of the three blanks is correct?
options:
  - text: '(a) $S = \begin{bmatrix} 1 & 2 \\ 0 & 2 \end{bmatrix}$; (b) $\sqrt{d_k} = \sqrt{2} = 1.414214$; (c) $A_{11}v_1 + A_{12}v_2 = 0.330238[2,0] + 0.669762[0,4] = [0.66,\ 2.68]$.'
    correct: true
    explain: 'Right on all three. $S_{ij} = q_i \cdot k_j$ gives $S_{11}=1$, $S_{12}=2$, $S_{21}=0$, $S_{22}=2$; the divisor is the standard deviation $\sqrt{d_k}$ with $d_k = 2$ here; and output row 1 is a convex combination of the rows of $V$, so it lands inside the hull of $[2,0]$ and $[0,4]$.'
  - text: '(a) $S = \begin{bmatrix} 1 & 0 \\ 2 & 2 \end{bmatrix}$; (b) $\sqrt{2} = 1.414214$; (c) $A_{11}v_1 + A_{21}v_2 = 0.330238[2,0] + 0.195570[0,4] = [0.66,\ 0.78]$.'
    misconception: U3-M5
    explain: 'Both errors are the same axis slip. $S_{12} = q_1 \cdot k_2 = [1,1]\cdot[1,1] = 2$ sits in row 1, not row 2 - this filling computed $KQ^T$. And the softmax and the aggregation both run along rows, so output row 1 must use row 1 of $A$; the column entries do not even sum to 1, which is why the result falls outside the convex hull.'
  - text: '(a) $S = \begin{bmatrix} 1 & 2 \\ 0 & 2 \end{bmatrix}$; (b) $d_k = 2$, the head dimension itself, or equivalently the largest score in each row; (c) $A_{11}v_1 + A_{12}v_2 = [0.66,\ 2.68]$.'
    misconception: U3-M2
    explain: 'The score matrix and the aggregation are right, the divisor is not. The variance of a raw dot product is $d_k$, and it is the standard deviation $\sqrt{d_k} = 1.414214$ that sets the spread the softmax sees. A divisor read off the data - the row max, the norm of $q_i$ - would be a normalization; this is a constant fixed by the architecture.'
  - text: '(a) $S = \begin{bmatrix} 1 & 2 \\ 0 & 2 \end{bmatrix}$; (b) $\sqrt{2} = 1.414214$; (c) the output is $A$ itself, so output row 1 is $[0.33,\ 0.67]$.'
    misconception: U3-M4
    explain: 'The $2 \times 2$ weight matrix is a transient intermediate, consumed by the final multiply and never passed forward. The output is $AV$, of shape $n \times d_v$, so output row 1 is $0.330238[2,0] + 0.669762[0,4] = [0.66,\ 2.68]$.'
check: choice
```

## Causal masking

You probably think a language model reads its prompt left to right, one token
at a time, the way it emits text. Everything you have ever watched a chat
model do supports that: tokens appear one after another, in order.

<!-- refutes: M9 -->

Here is the prediction that model makes. If the model processes tokens
sequentially, then processing a 10,000-token prompt requires 10,000 sequential
steps, so time-to-first-token should scale linearly with prompt length and
should be roughly 10,000 times the cost of one generated token.

Measure it. Prompt processing (prefill) is dramatically faster per token than
generation - often by two orders of magnitude - and it saturates GPU compute
in a way generation never does. That is not what a sequential process looks
like. It is what a single large matrix multiply looks like, because that is
what it is: the entire prompt goes through the network in **one** forward
pass, all positions computed simultaneously.

Look back at the hand computation. At no point did computing row 2 require
row 1's result. $S = QK^T$ produces all nine scores in one multiply. The
softmax rows are independent. $O = AV$ is one more multiply. There is no
ordering in the arithmetic at all.

Which raises the real question: if nothing is sequential, what stops position
2 from reading position 3 - a token that has not been generated yet? Nothing,
in what we computed above. Row 2 of $A$ put weight $0.503490$ on position 3.
For a bidirectional encoder that is correct and desirable. For a model trained
to predict the next token it is fatal: the model would learn to predict token
3 by looking at token 3, achieve near-zero training loss, and generate
nothing useful at inference when the future genuinely does not exist.

So constrain it. Before the softmax, set every score where the key position is
later than the query position to $-\infty$:

$$S'_{ij} \leftarrow -\infty \quad \text{for all } j > i$$

Then $e^{-\infty} = 0$, so those positions get weight exactly 0, and - this is
the part that matters - the softmax denominator sums only the surviving terms,
so the remaining weights **renormalize to sum to 1** over the visible
positions.

The order is load-bearing. Masking before the softmax means each query
redistributes its full attention budget over the positions it can see.
Masking after the softmax - zeroing entries of $A$ - would leave rows summing
to less than 1, so early positions would emit systematically shrunken outputs
purely as a function of their index.

Applied to our scaled scores:

$$S'_{\text{masked}} = \begin{bmatrix}
1.414214 & -\infty & -\infty \\
1.414214 & 1.414214 & -\infty \\
2.121320 & 1.414214 & 3.535534
\end{bmatrix}$$

Row by row:

- **Row 1** sees only position 1. One surviving term, so softmax of a
  single-element row is $[1, 0, 0]$ regardless of the score's value. Output
  row 1 is exactly $v_1 = [1, 1]$. Token 1 has nothing to mix with and
  passes its own value through unchanged.
- **Row 2** sees positions 1 and 2, with equal scaled scores $1.414214$.
  Equal scores give equal weights: $[0.5,\ 0.5,\ 0]$. The exponentials are
  both $4.113250$ and cancel exactly - the score value is irrelevant, only
  the equality matters. Output row 2 is
  $0.5[1,1] + 0.5[0,3] = [0.5,\ 2.0]$, exact.
- **Row 3** sees everything, so it is unchanged from the unmasked
  computation: $[0.178370,\ 0.087949,\ 0.733681]$, output
  $[1.645732,\ 1.909579]$.

$$A_{\text{causal}} = \begin{bmatrix}
1 & 0 & 0 \\
0.5 & 0.5 & 0 \\
0.178370 & 0.087949 & 0.733681
\end{bmatrix}
\qquad
O_{\text{causal}} = \begin{bmatrix}
1 & 1 \\
0.5 & 2 \\
1.645732 & 1.909579
\end{bmatrix}$$

**Check:** $A_{\text{causal}}$ is lower-triangular and every row still sums to
exactly 1. The last row is untouched by masking - it always is, since the
last position can see everything.

```beat
id: u3-b9
type: compute
concept: c-causal-mask
prompt: |
  With causal masking applied to the scaled score matrix

  $$S' = \begin{bmatrix}
  1.414214 & 2.828427 & 1.414214 \\
  1.414214 & 1.414214 & 2.121320 \\
  2.121320 & 1.414214 & 3.535534
  \end{bmatrix}$$

  what is $A_{21}$, the causal attention weight from token 2 onto token 1?

  You should not need to exponentiate anything to answer this, and the answer
  is exact rather than a rounded decimal. If you find yourself reporting a
  number the unmasked computation produced, you have skipped the mask.
answer: 0.5
check: numeric(0.001)
```

Two consequences worth holding onto.

**Order information does not come from the mask.** The mask says which
positions are visible, not where they sit. Strip positional information out
of $X$ and a causal model still cannot see the future, but it also cannot
tell an adjacent token from one 500 positions back. Order lives entirely in
positional encodings added to or applied to the token representations; the
mask only enforces causality.

**Generation is sequential; the forward pass is not.** Emitting 100 tokens
takes 100 forward passes because each new token must be sampled before it can
be embedded and fed back. Within any one of those passes every position is
computed in parallel. The sequential feel of a streaming response is the loop
around the model, not the model.

## What attention costs

You probably think that because attention is parallel, it is cheap in
sequence length - that the whole point of dropping recurrence was to make the
operation linear.

<!-- refutes: M10 -->

Here is the prediction. If attention were $O(n)$ in sequence length, doubling
the context from 4k to 8k tokens would double the attention cost.

It quadruples it. Count the arithmetic in the hand computation and generalize:

- $S = QK^T$: an $(n \times d_k)$ by $(d_k \times n)$ multiply. That is $n^2$
  output entries, each a dot product of length $d_k$: $n^2 d_k$
  multiply-accumulates.
- softmax over $A$: $n^2$ entries, constant work each.
- $O = AV$: an $(n \times n)$ by $(n \times d_v)$ multiply: $n^2 d_v$
  multiply-accumulates.

Total $\Theta(n^2 d)$. The $n^2$ is unavoidable in the mechanism as
specified, because *every query is compared against every key* - that is what
content-addressing without a position-indexed shortcut means. It is not an
implementation artifact and no amount of parallel hardware changes it.

The confusion is a classic systems trap, and worth naming precisely, because
the intuition behind it is not stupid. Parallelism changes **latency**, not
**work**. Attention has $O(n^2)$ work and $O(1)$ sequential depth. An RNN has
$O(n)$ work and $O(n)$ sequential depth. On a GPU with enough cores to hide
the width, attention *feels* faster despite doing more total arithmetic - and
at $n = 512$ it genuinely is faster in wall-clock, which is why the intuition
survives contact with small models. Push $n$ to 128k and the $n^2$ term
dominates everything else in the network, and the same asymptotics that were
invisible become the entire cost model.

Concretely: going from $n = 1024$ to $n = 8192$ is 8 times the tokens and 64
times the attention arithmetic. This is the reason long context is expensive,
the reason context-length pricing is not linear, and the motivation behind
essentially every architectural variant you have heard of - sliding-window
attention, sparse attention, linear attention, and (for the memory side
rather than the compute side) multi-query and grouped-query attention.

```beat
id: u3-b10
type: self-explain
concept: c-sdpa
prompt: |
  A colleague says: "Self-attention is $O(n)$ - it is fully parallel, that is
  the whole reason transformers replaced RNNs."

  Which explanation names exactly what they have confused with what, and
  answers the sharper version of their question: given that attention does
  more total arithmetic than an RNN layer ($n^2 d$ versus $n d^2$), why did
  transformers win?
options:
  - text: 'They have confused parallelism with asymptotic work. Attention does $\Theta(n^2 d)$ arithmetic because every query is dotted against every key; what parallelism buys is sequential depth, $O(1)$ matrix operations against the $O(n)$ dependency chain of an RNN. Transformers won because accelerators are bottlenecked on dependency chains rather than on FLOPs, so trading more total work for a shorter critical path saturates the device.'
    correct: true
    explain: 'Right. At $n = 512$ attention really is faster in wall-clock despite doing more arithmetic, which is why the intuition survives; push $n$ to 128k and the $n^2$ term dominates the whole network, which is the long-context regime everyone is now fighting.'
  - text: 'They are right. Every query-key pair is computed in one parallel step, so the cost in sequence length is linear, and transformers won because that single step replaced a chain of $n$ steps.'
    misconception: M10
    explain: 'Count the arithmetic: $n^2$ scores, each a dot product of length $d_k$, then an $n \times n$ by $n \times d_v$ multiply. Going from $n = 1024$ to $n = 8192$ is 8 times the tokens and 64 times the attention arithmetic. Parallelism changes latency, never work.'
  - text: 'The confusion is about memory rather than compute: attention compute per token really is linear, and what grows quadratically is the KV cache, which is why long context is expensive. Transformers won because the cache made generation cheap.'
    misconception: M17
    explain: 'The KV cache grows linearly in $n$ - one key and one value per token - and it only avoids recomputing projections for tokens already seen. The quadratic term is the query-key product itself: each new token dots its query against all $n$ keys.'
  - text: 'The confusion is that attention is not parallel at all - inside a forward pass each position waits for the one before it, so it is $O(n)$ work with $O(n)$ depth just like an RNN. Transformers won on a better parameterization, not on parallelism.'
    misconception: M9
    explain: 'Nothing in $S = QK^T$, the row-wise softmax, or $O = AV$ needs row 1 before row 2; the whole prompt goes through in one forward pass, which is why prefill saturates a GPU and generation does not. Only generation is sequential, because each new token must be sampled before it can be embedded.'
check: choice
```

## What you now know

Attention is six operations: three projections, a matrix of pairwise dot
products, a division by a constant, a row-wise softmax, and one more matrix
multiply. It moves information between positions by content rather than by
index, which is why it handles any sequence length with fixed-size
parameters. It has no branches, no state, and no agency - "the model attended
to the subject" describes a weight distribution, never a decision. Its causal
variant enforces the arrow of time with an additive $-\infty$ before the
softmax, not with sequential execution. And it costs $n^2$, which is the
single most consequential fact about the economics of long context.

One attention head produces one weighted average per position. Real models run
many in parallel and stack the result dozens of layers deep. That is the next
unit.
