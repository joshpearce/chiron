---
unit: u5
title: "Training: loss to aligned model"
concepts:
  - c-xent
  - c-backprop
  - c-optimizers
  - c-scaling
  - c-posttrain
assumes:
  - c-notation
  - c-matmul
  - c-dotprod
  - c-lm-objective
  - c-tokens
  - c-logits
  - c-softmax
  - c-gradient
  - c-loss-surface
  - c-mlp-block
  - c-residual
  - c-depth
---

You now have a forward pass. Tokens in, embeddings, a stack of transformer
blocks writing into a residual stream, an unembedding, logits over the
vocabulary, softmax. Every weight in that stack started as a random number.
This unit is the machinery that turns random numbers into a model that answers
your question, and it is four ideas in a trench coat: a number that says how
wrong you are, a way to blame every weight for its share of that number, a rule
for changing weights given the blame, and a second phase that changes what the
model does with capability it already has.

## Cross-entropy: turning a distribution into one number

The forward pass ends with $p \in \mathbb{R}^{V}$, a probability distribution
over the vocabulary ($V$ = vocabulary size, so $V \approx 50{,}000$ to
$200{,}000$ in practice, and $p_i \ge 0$, $\sum_i p_i = 1$). The training data
tells you which token actually came next. You need to turn "here is a
distribution, here is the truth" into a single scalar, because the entire
optimization machinery downstream consumes exactly one number.

The obvious engineering instinct is accuracy: did the argmax match? That fails
for two reasons, and the second is the interesting one. First, argmax is not
differentiable, so there is nothing to take a gradient of. Second, and worse,
accuracy is blind to confidence. A model that puts 0.34 on the right token and
a model that puts 0.99 on it score identically, so there is no signal pushing
0.34 toward 0.99. Most of what training does is exactly that push.

The loss that works is **cross-entropy**:

$$L = -\sum_{i=1}^{V} y_i \log p_i$$

where $y \in \mathbb{R}^{V}$ is the target distribution and $p \in \mathbb{R}^{V}$
is the model's predicted distribution. For language modeling $y$ is one-hot:
$y_t = 1$ for the token $t$ that actually occurred, $0$ everywhere else. Every
term in that sum where $y_i = 0$ contributes nothing, so the whole sum collapses:

$$L = -\log p_t$$

That is the entire loss function. The negative log probability the model
assigned to the token that actually came next. Note what it does and does not
look at: it reads exactly one entry of $p$. It never inspects the other 50,000.
It does not need to, because $p$ is normalized, so raising $p_t$ necessarily
lowers the rest.

The shape of $-\log$ is the whole design. At $p_t = 1$ the loss is $0$. At
$p_t = 0.5$ it is $0.693$. At $p_t = 0.1$ it is $2.303$. At $p_t \to 0$ it goes
to $+\infty$. Confident and wrong is punished without bound; confident and right
is free. That asymmetry is what makes a model calibrated rather than merely
accurate.

```beat
id: u5-b1
type: predict
concept: c-xent
prompt: |
  Two models are scored on the same token. Model A puts probability $0.5$ on
  the correct token. Model B puts probability $0.25$ on it. Before computing:
  is B's loss twice A's, or something else? Predict the relationship, then
  the two actual values.
answer: |
  Not twice. Loss is $-\log p$, so halving the probability ADDS a constant,
  it does not multiply the loss. $L_A = -\ln 0.5 = 0.693$,
  $L_B = -\ln 0.25 = 1.386$. Here that constant ($\ln 2 = 0.693$) happens to
  equal $L_A$, so B's loss is coincidentally twice A's. Check the general
  claim: going from $0.9$ to $0.45$ also halves the probability, but the loss
  goes $0.105 \to 0.799$, which is 7.6x, not 2x. The invariant is that each
  halving of $p$ adds $\ln 2 = 0.693$ to the loss.
rubric: |
  Must identify that cross-entropy is additive in log-probability, so halving
  $p$ adds $\ln 2 \approx 0.693$ rather than doubling the loss. Getting
  $0.693$ and $1.386$ but concluding "so it is always 2x" = partial credit
  (right arithmetic, wrong invariant) - this is the multiplicative-loss error.
  Answering "twice, because probability halved" with no log reasoning = fail.
check: llm
```

### Worked: three candidate tokens

Take a vocabulary of three tokens after the prefix "The cat sat on the":
$\{\texttt{" mat"}, \texttt{" dog"}, \texttt{" rug"}\}$, indices 0, 1, 2. The
model emits logits $z = [2.0,\ 1.0,\ 0.1]$ ($z \in \mathbb{R}^{3}$ = raw scores,
one per vocabulary entry, unbounded in sign and magnitude).

Softmax first ($\text{softmax}(z)_i = e^{z_i} / \sum_j e^{z_j}$):

$$e^{2.0} = 7.389056,\quad e^{1.0} = 2.718282,\quad e^{0.1} = 1.105171$$
$$\textstyle\sum_j e^{z_j} = 11.212509$$
$$p = [7.389056,\ 2.718282,\ 1.105171] / 11.212509 = [0.659001,\ 0.242433,\ 0.098566]$$

The true next token is `" mat"`, index 0. So:

$$L = -\log p_0 = -\log(0.659001) = 0.417030$$

All logarithms here are natural logs, which is the ML convention; the unit of
loss is then **nats**. If the true token had instead been `" rug"` (index 2,
probability $0.098566$), the loss would have been $-\log(0.098566) = 2.317030$:
5.6x larger for the same forward pass, purely because the model bet on the
wrong horse.

```beat
id: u5-b2
type: compute
concept: c-xent
prompt: |
  Same three-token vocabulary. This time the model emits logits
  $z = [1.0,\ 3.0,\ 0.0]$ and the true next token is index 0.
  Compute the cross-entropy loss in nats. Useful values:
  $e^{1.0} = 2.718282$, $e^{3.0} = 20.085537$, $e^{0.0} = 1.0$.
  Give the number to 3 decimal places.
answer: 2.170
rubric: |
  Sum of exponentials is $23.803819$; $p_0 = 2.718282 / 23.803819 = 0.114195$;
  $L = -\ln(0.114195) = 2.169846$. An answer near $0.170$ means the loss was
  taken on the argmax token (index 1) instead of the target - the "loss scores
  the model's best guess" error. An answer near $0.114$ means the probability
  was reported instead of its negative log.
check: numeric(0.01)
```

### One token, or the whole sequence

A real training step does not score one token. Given a sequence of $n$ tokens,
the forward pass produces $n$ distributions in parallel (one at each position,
each predicting the token at the next position), and the loss is the mean:

$$L = \frac{1}{n} \sum_{k=1}^{n} -\log p^{(k)}_{t_{k+1}}$$

where $p^{(k)}$ is the distribution predicted at position $k$ and $t_{k+1}$ is
the token that actually occurs at position $k+1$. Two things follow that
surprise people. Every position is supervised, so a 2,000-token document yields
2,000 training signals from one forward pass, not one. And every position is
scored against the *real* next token from the corpus, not against whatever the
model itself would have generated. That second property is called **teacher
forcing**: the model is never fed its own outputs during pretraining, so
training does not simulate generation. It grades a parallel exam where every
question comes with the previous answer already filled in correctly.

## Perplexity, and the floor that is not zero

Perplexity is cross-entropy in a friendlier unit:

$$\text{PPL} = e^{L}$$

with $L$ the mean cross-entropy per token in nats. Its interpretation: the
model is as uncertain as if it were choosing uniformly among $\text{PPL}$
equally likely tokens at each step. A perplexity of 1 means total certainty. A
model that has learned nothing and outputs uniform logits over a 50,257-token
vocabulary has $L = \ln(50257) = 10.825$ and $\text{PPL} = 50257$, exactly the
vocabulary size, which is what "no information" should look like. Our worked
example above had $L = 0.417030$, so $\text{PPL} = e^{0.417030} = 1.517$: on
that one token the model is behaving as though it were choosing among 1.5
equally likely options. On a single token perplexity is exactly $1/p_t$, and
$1/0.659001 = 1.517$ confirms it.

Perplexity is a strictly monotone function of loss, so it ranks models
identically. It exists because "the model is effectively choosing among 8 tokens"
lands and "the loss is 2.13" does not.

<!-- refutes: M18 -->

**You probably think training aims for zero loss, and that the loss remaining
at the end is a deficiency you would remove with a better architecture or more
compute.** Every green test suite in your career has taught you that the target
is zero and anything above it is unfinished work.

**Here is the prediction that fails.** If zero loss were the goal, a
loss-0 model would be the ideal language model. Zero loss means $p_t = 1$ at
every position: for the prefix "I went to the", the model assigns probability
1.0 to exactly one continuation and 0 to `store`, `park`, `doctor`, `beach`,
`gym`. Sample from it at any temperature and it emits one deterministic string
per prefix, forever. That is not a better language model, it is a lookup table
with a broken hash function. And empirically the way you *do* get near-zero
loss is to train a large model on a small corpus until it memorizes it, at which
point held-out loss goes up, not down. Zero training loss is a symptom of
failure, not the goal.

**Why it is appealing:** loss is the objective, optimization minimizes the
objective, and the minimum of $-\log p_t$ is zero. The reasoning is locally
valid. It just ignores what $p$ is a distribution *over*.

**What is actually true.** Language is genuinely stochastic. There are many
legitimate continuations of any prefix, and the corpus itself contains
different continuations of identical prefixes. Let $H$ be the entropy of the
true distribution of next tokens given context. The best possible expected
cross-entropy for *any* model, including a perfect one, is exactly $H$. This is
not a limitation of your architecture. It is the information content of the
data. Predicting a fair coin flip has an irreducible loss of $\ln 2 = 0.693$
nats no matter how large your model is, and the model that "achieves" 0 on the
coin flip has memorized the specific sequence and will do worse on the next one.

The gap that matters is not $L - 0$. It is $L - H$: the distance to the floor.
Scaling laws, which we get to shortly, are literally written in this form, with
the irreducible term as a fitted constant. Reported loss numbers hover around
$1.9$ to $2.6$ nats for good models not because those models are 2 nats broken,
but because the floor is somewhere near $1.7$ and they are within half a nat of
it.

```beat
id: u5-b3
type: self-explain
concept: c-xent
prompt: |
  A colleague reports: "I trained our model for another week and the training
  loss dropped from $1.9$ to $0.4$. Big win." In your own words, explain what
  you would check before agreeing, and what the number $0.4$ most likely means.
answer: |
  A training loss of $0.4$ nats means perplexity $e^{0.4} = 1.49$, i.e. the
  model is behaving as though there are about 1.5 plausible next tokens at
  every position. Natural language is nowhere near that predictable; the
  irreducible entropy of text is roughly 1.7 nats on standard corpora. So the
  model is not modeling language, it is reproducing the training set. The check
  is held-out loss: if training loss fell to $0.4$ while validation loss rose
  or flattened well above it, that is memorization. The correct target is
  approaching the entropy floor on unseen data, not approaching zero on seen
  data.
rubric: |
  Must contain (1) the observation that $0.4$ is implausibly far below the
  entropy floor of language, and (2) the proposed check is held-out/validation
  loss, and (3) the diagnosis of memorization/overfitting. Any 2 of 3 = pass.
  An answer that treats the drop as unambiguously good = fail, cites M18.
  An answer saying "loss can never be that low" as a hard mathematical
  impossibility is also wrong - it is achievable on the training set, which is
  precisely the problem; mark partial and correct it.
check: llm
```

## Backprop as blame assignment

You have a scalar $L$. Somewhere upstream sit $10^{11}$ weights. You need, for
each weight $w$, the number $\partial L / \partial w$: how much the loss changes
per unit change in that weight, holding everything else fixed. With that vector
you can move every weight slightly downhill at once.

Forget derivatives for a moment. The problem is blame assignment in a
dependency graph, and you have solved that shape of problem before. A request
took 400ms. That request passed through a load balancer, an auth service, three
database calls, and a serializer. You want a per-component attribution, and you
want it for all components from a single trace rather than by rerunning the
request once per component with that component sped up.

Backprop is the exact analogue and it rests on one local rule. Every node in the
computation graph knows two things: how much the final loss changes per unit
change in *its own output* (call this its blame, handed to it by whatever
consumed its output), and how its own output changes per unit change in each of
its inputs (its local derivative, which it can compute knowing nothing about the
rest of the network). Multiply the two and you get how much the loss changes per
unit change in its inputs, which is exactly the blame to hand to the nodes
upstream. That is the chain rule:

$$\frac{\partial L}{\partial u} = \frac{\partial L}{\partial v} \cdot \frac{\partial v}{\partial u}$$

for $u$ an input to a node and $v$ its output. Blame arriving times local
sensitivity equals blame departing. Start at the loss with blame $\partial L /
\partial L = 1$ and propagate backward. Every weight gets its exact gradient in
one pass.

The word "backprop" names the bookkeeping, not a new mathematical idea. There is
no approximation, no numerical differencing, no finite epsilon. The gradients
are exact to floating-point precision. What backprop contributes is *order*: it
caches each node's blame so that a node feeding into 50 downstream consumers
computes its local derivative once instead of 50 times.

The reason it runs backward rather than forward is a shape argument. You have
$10^{11}$ inputs (weights) and 1 output (the loss). Sweeping forward gives you
the sensitivity of everything to *one input*, so you would need $10^{11}$ passes.
Sweeping backward gives you the sensitivity of *one output* to everything, so
you need one. Backprop costs roughly the same as the forward pass, and that fact
is the only reason training large models is affordable.

There is one bill: to compute local derivatives on the way back, you need the
activations from the way forward. So the forward pass stores its intermediates,
and that stored graph is why training memory dwarfs inference memory for the
same model, and why gradient checkpointing (recompute instead of store) is a
knob worth having.

```beat
id: u5-b4
type: predict
concept: c-backprop
prompt: |
  A network has a ReLU unit whose pre-activation was $a = -0.3$ on this
  example, so its output after $\text{ReLU}(a) = \max(0, a)$ was $0$.
  Some blame $\partial L / \partial h = 4.2$ arrives at that unit's output from
  downstream. Predict the gradient that reaches the weights feeding into this
  unit, and say what that implies if it keeps happening across examples.
answer: |
  Zero. The local derivative of $\text{ReLU}$ at $a = -0.3$ is $0$, so
  $\partial L / \partial a = 4.2 \times 0 = 0$, and every weight feeding this
  unit gets gradient $0 \times x = 0$ regardless of how large the incoming
  blame or the input was. If the unit is negative for every example in the
  batch, those weights receive no gradient at all and never update: the unit is
  dead, and nothing in the update rule can revive it, because the only path
  back to it is multiplied by zero.
rubric: |
  Must state the gradient is exactly zero AND attribute it to the ReLU local
  derivative being zero for negative pre-activation. Must connect to the dead
  unit consequence (no updates, permanently stuck) for full credit. Answering
  "small but nonzero" or "4.2 scaled down" = fail; that is the smooth-gate
  error. Answering zero but attributing it to the activation output being zero
  rather than the derivative being zero = partial (right answer, wrong
  mechanism - the output value is not what gates the gradient).
check: llm
```

## Backprop worked: two layers, real numbers

<!-- fade: backprop -->

Everything above is a claim. Here is the whole procedure on a network small
enough to hold in your head, with every number computed. The network is a
2-layer MLP classifying over 3 tokens, which is the smallest thing that has all
the structural features of a real one: a weight matrix, a nonlinearity, a
second weight matrix, softmax, cross-entropy.

**The network.** Input $x \in \mathbb{R}^{2}$ (the incoming activation vector),
first weight matrix $W_1 \in \mathbb{R}^{2\times 2}$ and bias $b_1 \in
\mathbb{R}^{2}$, ReLU, second weight matrix $W_2 \in \mathbb{R}^{3\times 2}$ and
bias $b_2 \in \mathbb{R}^{3}$, softmax, cross-entropy against target class 0.
Rows of $W_1$ index output units, columns index input components, so
$(W_1)_{ij}$ is the weight from input $j$ to hidden unit $i$. Same convention
for $W_2$.

$$x = \begin{bmatrix} 1.0 \\ 2.0 \end{bmatrix},\quad
W_1 = \begin{bmatrix} 0.5 & -0.1 \\ -0.3 & 0.8 \end{bmatrix},\quad
b_1 = \begin{bmatrix} 0.1 \\ -0.1 \end{bmatrix}$$

$$W_2 = \begin{bmatrix} 1.0 & 0.0 \\ 0.0 & 1.0 \\ -1.0 & 0.5 \end{bmatrix},\quad
b_2 = \begin{bmatrix} 0.0 \\ 0.0 \\ 0.0 \end{bmatrix},\quad \text{target class } t = 0$$

### Forward pass

Pre-activation $a_1 = W_1 x + b_1$, computed one row at a time:

$$a_{1,0} = (0.5)(1.0) + (-0.1)(2.0) + 0.1 = 0.5 - 0.2 + 0.1 = 0.4$$
$$a_{1,1} = (-0.3)(1.0) + (0.8)(2.0) + (-0.1) = -0.3 + 1.6 - 0.1 = 1.2$$

Hidden activation $h = \text{ReLU}(a_1) = \max(0, a_1)$, elementwise. Both
entries are positive, so:

$$h = [0.4,\ 1.2]$$

Logits $z = W_2 h + b_2$:

$$z_0 = (1.0)(0.4) + (0.0)(1.2) = 0.4$$
$$z_1 = (0.0)(0.4) + (1.0)(1.2) = 1.2$$
$$z_2 = (-1.0)(0.4) + (0.5)(1.2) = -0.4 + 0.6 = 0.2$$

Softmax:

$$e^{0.4} = 1.491825,\quad e^{1.2} = 3.320117,\quad e^{0.2} = 1.221403,\quad
\textstyle\sum = 6.033344$$
$$p = [0.247263,\ 0.550295,\ 0.202442]$$

Loss, with target class $t = 0$:

$$L = -\log p_0 = -\log(0.247263) = 1.397301$$

The model currently prefers class 1 at $0.550295$ while the truth is class 0.
Perplexity $e^{1.397301} = 4.044$ on a 3-way choice, which is worse than
guessing uniformly ($\text{PPL} = 3$). This is a bad prediction, so the
gradients should be large and pointed.

### Backward pass

**Step 1: loss into logits.** Softmax followed by cross-entropy has a
famously clean joint derivative. For a one-hot target $y$ (here $y = [1, 0, 0]$):

$$\frac{\partial L}{\partial z} = p - y$$

This is not a coincidence and it is not an approximation; it is what falls out
when you differentiate $-\log \frac{e^{z_t}}{\sum_j e^{z_j}}$ with respect to
each $z_i$ and let the softmax Jacobian cancel against the log (the derivation
is in the deeper-math treatment). Numerically:

$$\frac{\partial L}{\partial z} = [0.247263 - 1,\ 0.550295 - 0,\ 0.202442 - 0]
= [-0.752737,\ 0.550295,\ 0.202442]$$

Read the signs. The gradient on $z_0$ is negative, meaning increasing $z_0$
decreases the loss, and since updates move *against* the gradient, $z_0$ will go
up. The gradients on $z_1$ and $z_2$ are positive, so those logits go down.
Notice the magnitude of the blame on each wrong class equals exactly the
probability the model wasted on it. Cross-entropy's error signal is "how much
probability mass is in the wrong place".

**Step 2: logits into $W_2$.** $z = W_2 h + b_2$, so $z_i = \sum_j (W_2)_{ij} h_j$
and therefore $\partial z_i / \partial (W_2)_{ij} = h_j$. Chain that against the
blame on $z_i$:

$$\frac{\partial L}{\partial (W_2)_{ij}} = \frac{\partial L}{\partial z_i} \cdot h_j$$

which is the outer product $\frac{\partial L}{\partial z} \, h^{\top}$. Row 0
worked out in full, with $h = [0.4, 1.2]$:

$$\frac{\partial L}{\partial (W_2)_{00}} = (-0.752737)(0.4) = -0.301095$$

```beat
id: u5-b5
type: compute
concept: c-backprop
prompt: |
  Continuing the worked network above, with
  $\partial L / \partial z = [-0.752737,\ 0.550295,\ 0.202442]$ and
  $h = [0.4,\ 1.2]$: compute $\partial L / \partial (W_2)_{01}$, the gradient
  on the weight connecting hidden unit 1 to logit 0. Give it to 4 decimal
  places.
answer: -0.9033
rubric: |
  $\partial L / \partial (W_2)_{01} = (\partial L / \partial z_0)(h_1)
  = (-0.752737)(1.2) = -0.903284$. An answer of $-0.301095$ used $h_0$ instead
  of $h_1$ (index transposition). A positive $0.903284$ dropped the sign, which
  inverts the update direction. An answer of $0.660354$ used
  $\partial L / \partial z_1$ instead of $\partial L / \partial z_0$.
check: numeric(0.01)
```

Completing the matrix by the same rule, $\frac{\partial L}{\partial (W_2)_{ij}}
= (\partial L / \partial z_i)(h_j)$:

$$\frac{\partial L}{\partial W_2} = \begin{bmatrix}
-0.301095 & -0.903284 \\
0.220118 & 0.660354 \\
0.080977 & 0.242930
\end{bmatrix}$$

And since $\partial z_i / \partial (b_2)_i = 1$, the bias gradient is the blame
unchanged: $\partial L / \partial b_2 = [-0.752737,\ 0.550295,\ 0.202442]$.

**Step 3: logits into $h$, the step that makes it deep learning.** The same
$z = W_2 h$ also depends on $h$, and $h$ feeds *all three* logits, so blame from
all three converges:

$$\frac{\partial L}{\partial h_j} = \sum_{i=0}^{2} \frac{\partial L}{\partial z_i} \cdot (W_2)_{ij}$$

which is $W_2^{\top} (\partial L / \partial z)$. The transpose is not notational
decoration: forward, $W_2$ maps 2 hidden units to 3 logits; backward, $W_2^{\top}$
maps 3 logit-blames to 2 hidden-blames. Same weights, opposite direction.

$$\frac{\partial L}{\partial h_0} = (-0.752737)(1.0) + (0.550295)(0.0) + (0.202442)(-1.0) = -0.955179$$
$$\frac{\partial L}{\partial h_1} = (-0.752737)(0.0) + (0.550295)(1.0) + (0.202442)(0.5) = 0.651516$$

**Step 4: through the ReLU.** $h = \max(0, a_1)$ elementwise, so
$\partial h_j / \partial a_{1,j}$ is $1$ where $a_{1,j} > 0$ and $0$ where it is
negative. Here $a_1 = [0.4, 1.2]$, both positive, so both gates are open and the
blame passes through untouched:

$$\frac{\partial L}{\partial a_1} = [-0.955179,\ 0.651516]$$

This is the one place where the value of a forward activation changes the
*structure* of the backward pass rather than just scaling it. Had $(W_1)_{01}$
been $-0.35$ instead of $-0.1$, then $a_{1,0} = 0.5 - 0.7 + 0.1 = -0.1$, the
gate would be shut, $\partial L / \partial a_{1,0}$ would be exactly $0$, and
the entire first row of $\partial L / \partial W_1$ below would be zeros no
matter how badly the network got the answer wrong.

**Step 5: into $W_1$.** Identical structure to step 2, one layer down:
$a_1 = W_1 x + b_1$ gives $\partial L / \partial (W_1)_{ij} = (\partial L /
\partial a_{1,i})(x_j)$, with $x = [1.0, 2.0]$:

$$\frac{\partial L}{\partial W_1} = \begin{bmatrix}
(-0.955179)(1.0) & (-0.955179)(2.0) \\
(0.651516)(1.0) & (0.651516)(2.0)
\end{bmatrix} = \begin{bmatrix}
-0.955179 & -1.910358 \\
0.651516 & 1.303031
\end{bmatrix}$$

$$\frac{\partial L}{\partial b_1} = [-0.955179,\ 0.651516]$$

Every parameter now has a gradient, obtained in one backward sweep, using only
each layer's local rule and the blame handed down from above. Column 1 of
$\partial L / \partial W_1$ is exactly twice column 0, because $x_1 = 2 x_0$:
inputs that were larger get proportionally more blame, which is why input
scaling matters and why normalization layers earn their keep.

### The update, and proof it worked

Take one plain gradient-descent step with learning rate $\eta = 0.1$, applying
$w \leftarrow w - \eta \, \partial L / \partial w$ to every parameter. On a
single weight:

$$(W_1)_{01} \leftarrow -0.1 - (0.1)(-1.910358) = -0.1 + 0.191036 = 0.091036$$

Applying it to all four tensors and re-running the forward pass on the same
input gives $L = 0.584927$, down from $1.397301$. The loss fell because every
parameter moved a distance proportional to its own blame, in the direction that
reduces the loss. Scale this to $10^{11}$ parameters and $10^{13}$ tokens and
that is pretraining. There is no other trick.

```beat
id: u5-b6
type: completion
concept: c-backprop
# Variants: blank different steps to shift what is being tested.
# Blanking (1) and (3) tests the chain-rule structure (default, below).
# Blanking (2) and (4) tests the outer-product / transpose mechanics.
# Blanking (5) alone is the warmup variant.
prompt: |
  Here is the same backward pass with three steps removed. Fill in each blank
  with the expression AND the numeric value.

  Given (forward pass, already computed):
  $x = [1.0,\ 2.0]$, $a_1 = [0.4,\ 1.2]$, $h = [0.4,\ 1.2]$,
  $z = [0.4,\ 1.2,\ 0.2]$, $p = [0.247263,\ 0.550295,\ 0.202442]$,
  target class $t = 0$, and
  $W_2 = \begin{bmatrix} 1.0 & 0.0 \\ 0.0 & 1.0 \\ -1.0 & 0.5 \end{bmatrix}$.

  1. $\partial L / \partial z$ = ____
  2. $\partial L / \partial W_2 = (\partial L / \partial z)\, h^{\top}$, whose
     first row is $[-0.301095,\ -0.903284]$
  3. $\partial L / \partial h$ = ____
  4. $\partial L / \partial a_1 = (\partial L / \partial h) \odot
     \mathbb{1}[a_1 > 0] = [-0.955179,\ 0.651516]$
  5. $\partial L / \partial W_1$ = ____
answer: |
  1. $\partial L / \partial z = p - y = [-0.752737,\ 0.550295,\ 0.202442]$,
     where $y = [1, 0, 0]$ is the one-hot target.
  3. $\partial L / \partial h = W_2^{\top} (\partial L / \partial z)
     = [-0.955179,\ 0.651516]$. Each hidden unit collects blame from all three
     logits it feeds.
  5. $\partial L / \partial W_1 = (\partial L / \partial a_1)\, x^{\top}
     = \begin{bmatrix} -0.955179 & -1.910358 \\ 0.651516 & 1.303031
     \end{bmatrix}$.
rubric: |
  Blank 1 must be $p - y$ with the target subtracted from the correct index,
  giving a negative first entry. Blank 3 must use $W_2$ TRANSPOSED - producing
  a 3-vector instead of a 2-vector means the transpose was dropped, which is
  the single most common mechanical error here. Blank 5 must be the outer
  product with $x$, so column 1 is twice column 0. All three expressions
  correct with arithmetic slips = pass. Any blank with the wrong structure
  (missing transpose, wrong operand, gradient of the loss w.r.t. the wrong
  variable) = fail.
check: llm
```

## Optimizers: four update rules, one loop

Backprop hands you $g = \partial L / \partial w$ for every parameter. The
optimizer decides what to do with it. Every optimizer in production use has the
same skeleton,

$$w \leftarrow w - \eta \cdot u(g)$$

with $\eta$ the learning rate (a scalar step size, typically $10^{-4}$ to
$10^{-3}$ for transformers) and $u$ some function of the current and past
gradients. The four you will meet differ only in $u$, and they are easy to
confuse because their pseudocode looks nearly identical. So rather than
describing each in turn, interrogate all four on the same three questions.

**Question 1: what does it remember?**

- **SGD**: nothing. $u(g) = g$. The update depends only on the current batch.
- **Momentum**: one running vector. $v \leftarrow \beta v + g$ with $\beta = 0.9$
  ($v$ = velocity, same shape as $w$; $\beta$ = decay), then $u = v$. It is an
  exponentially weighted sum of past gradients.
- **Adam**: two running vectors. $m \leftarrow \beta_1 m + (1-\beta_1) g$
  (first moment, a mean estimate, $\beta_1 = 0.9$) and
  $v \leftarrow \beta_2 v + (1-\beta_2) g^2$ (second moment, an uncentered
  variance estimate, $\beta_2 = 0.999$, with $g^2$ elementwise). Memory cost:
  2 extra numbers per parameter, which is why optimizer state dominates training
  memory.
- **AdamW**: the same two vectors as Adam. It does not add state.

**Question 2: what is the step size proportional to?**

This is the question that separates them, and it is where the confusion lives.

- **SGD**: proportional to $\|g\|$. Big gradient, big step. If your loss surface
  has a direction 1000x steeper than another, SGD takes steps 1000x larger in
  it, which is exactly wrong: the steep direction is where you want caution.
- **Momentum**: proportional to the *accumulated* gradient. Along a direction
  where the gradient keeps pointing the same way, the steps grow toward
  $g / (1-\beta) = 10g$ at $\beta = 0.9$. Along a direction where the gradient
  oscillates in sign, consecutive terms cancel and the steps shrink. It
  accelerates consistency and damps thrash.
- **Adam**: proportional to $m / \sqrt{v}$, which is roughly the gradient
  divided by its own recent magnitude. That ratio is scale-free: multiply every
  gradient in a direction by 100 and both $m$ and $\sqrt{v}$ scale by 100 and
  the step is unchanged. The step size becomes approximately $\eta$ in *every*
  direction, regardless of curvature. That is what "adaptive" means, and it is
  why Adam trains transformers where SGD stalls: the gradient magnitudes across
  a transformer's embeddings, attention projections, and MLP weights differ by
  orders of magnitude, and Adam normalizes them into a common scale.
- **AdamW**: identical to Adam. The change is elsewhere.

**Question 3: what does it do about weights growing without bound?**

- **SGD** and **momentum**: nothing, unless you add an L2 penalty
  $\frac{\lambda}{2}\|w\|^2$ to the loss, whose gradient is $\lambda w$. Added to
  $g$, that produces $w \leftarrow w - \eta(g + \lambda w)$, which is exactly
  weight decay. For SGD, L2 regularization and weight decay are the same thing.
- **Adam**: adds $\lambda w$ into $g$ before the moment updates, keeping the L2
  framing. This is broken, and the reason is Question 2. The whole point of Adam
  is that it divides the step by $\sqrt{v}$, so the decay term gets normalized
  along with everything else. A parameter with large historical gradients gets
  its decay divided down to nothing; a parameter with tiny gradients gets its
  decay amplified. The regularization strength ends up inversely proportional to
  gradient magnitude, which nobody asked for.
- **AdamW**: **decouples** it. Compute the adaptive step from $g$ alone, then
  subtract the decay separately:
  $w \leftarrow w - \eta \frac{\hat m}{\sqrt{\hat v} + \epsilon} - \eta \lambda w$.
  Now $\lambda$ means one thing across all parameters. That single change is the
  entire difference between Adam and AdamW, and it is why AdamW is the default
  for every large model trained since roughly 2018.

One remaining piece of Adam: at step $t = 1$, $m = (1-\beta_1) g = 0.1g$ and
$v = (1-\beta_2)g^2 = 0.001 g^2$, both badly biased toward zero because they
started at zero. Adam corrects with
$\hat m = m / (1 - \beta_1^t)$ and $\hat v = v / (1 - \beta_2^t)$, which divides
out exactly the initialization bias and decays to a no-op as $t$ grows.

### Worked: one weight, two optimizers

<!-- fade: adamw-step -->

A single weight $w = 0.5$ with gradient $g = 0.2$, learning rate
$\eta = 0.01$, at step $t = 1$ from a fresh optimizer state. Adam
hyperparameters $\beta_1 = 0.9$, $\beta_2 = 0.999$, $\epsilon = 10^{-8}$,
weight decay $\lambda = 0.01$.

**SGD:** $w \leftarrow 0.5 - (0.01)(0.2) = 0.5 - 0.002 = 0.498$.

**AdamW:**

$$m = (1 - 0.9)(0.2) = 0.02, \qquad v = (1 - 0.999)(0.2)^2 = 0.00004$$
$$\hat m = \frac{0.02}{1 - 0.9^1} = \frac{0.02}{0.1} = 0.2, \qquad
\hat v = \frac{0.00004}{1 - 0.999^1} = \frac{0.00004}{0.001} = 0.04$$
$$\text{adaptive step} = \eta \frac{\hat m}{\sqrt{\hat v} + \epsilon}
= 0.01 \cdot \frac{0.2}{0.2 + 10^{-8}} = 0.01$$
$$\text{decay} = \eta \lambda w = (0.01)(0.01)(0.5) = 0.00005$$
$$w \leftarrow 0.5 - 0.01 - 0.00005 = 0.48995$$

SGD moved the weight by $0.002$. AdamW moved it by $0.01005$, five times
further. But the number to stare at is $\hat m / \sqrt{\hat v} = 1$. After bias
correction at $t=1$, $\hat m = g$ and $\sqrt{\hat v} = |g|$, so the ratio is
exactly $\text{sign}(g)$ and the step is exactly $\eta$. Run it again with
$g = 0.002$, one hundred times smaller: SGD would move $0.00002$, while AdamW
computes $\hat m = 0.002$, $\sqrt{\hat v} = 0.002$, and moves $0.0099999$ -
essentially the same $0.01$, a 500x larger step than SGD takes. Adam does not
care how big your gradient is. It cares about its sign and its consistency.

That is the property that makes it work and the property that makes it
dangerous: a parameter receiving pure noise gets moved just as far as one
receiving real signal, which is why Adam needs weight decay and learning-rate
warmup and SGD mostly does not.

```beat
id: u5-b7
type: completion
concept: c-optimizers
prompt: |
  Same procedure, new numbers. Fill the blanks with expression AND value.
  A weight $w = -0.3$ receives gradient $g = -0.05$. Learning rate
  $\eta = 0.001$, weight decay $\lambda = 0.1$, $\beta_1 = 0.9$,
  $\beta_2 = 0.999$, $\epsilon = 10^{-8}$, optimizer state fresh at $t = 1$.

  1. $m = (1 - \beta_1) g = (0.1)(-0.05) = -0.005$
  2. $v = (1 - \beta_2) g^2 = (0.001)(0.0025) = 2.5 \times 10^{-6}$
  3. $\hat m = m / (1 - \beta_1^1) =$ ____ , $\hat v = v / (1 - \beta_2^1) =$ ____
  4. adaptive step $= \eta \, \hat m / (\sqrt{\hat v} + \epsilon) =$ ____
  5. decoupled decay $= \eta \lambda w =$ ____
  6. $w \leftarrow w - (\text{step}) - (\text{decay}) =$ ____
  7. One sentence: the ratio $\hat m / \sqrt{\hat v}$ came out to exactly
     $-1$ here. State why that is guaranteed at $t = 1$ for any $g$, and what
     it implies about the size of Adam's first step. ____
answer: |
  3. $\hat m = -0.005 / 0.1 = -0.05$; $\hat v = 2.5\times10^{-6} / 0.001
     = 0.0025$, so $\sqrt{\hat v} = 0.05$.
  4. $0.001 \times (-0.05 / 0.05) = -0.001$
  5. $(0.001)(0.1)(-0.3) = -0.00003$
  6. $w = -0.3 - (-0.001) - (-0.00003) = -0.29897$
  7. At $t = 1$ bias correction divides $m$ by $(1-\beta_1)$ and $v$ by
     $(1-\beta_2)$, which exactly undoes the factors applied when they were
     formed, so $\hat m = g$ and $\hat v = g^2$. The ratio is therefore
     $g / |g| = \mathrm{sign}(g)$ and the first step is exactly $\eta$ in
     magnitude regardless of how large or small the gradient was.
# Variant blanks: blank 4+6 for a fast arithmetic pass; blank 3+7 to isolate
# bias correction; blank 5+7 for the mastery pass, since the decoupled decay
# and the sign(g) property are the two ideas that separate AdamW from SGD.
rubric: |
  3 must give -0.05 and 0.0025. 4 must be -0.001, NOT -0.00005: an answer that
  scales with the gradient magnitude skipped the normalization and is the SGD
  update in disguise. 5 must be -0.00003 and must be subtracted separately -
  folding $\lambda w$ into $g$ before step 1 is Adam-with-L2, not AdamW, and
  fails the beat. 6 must be -0.29897; -0.29900 means the decay was dropped,
  -0.30103 means the step was added rather than subtracted (a negative gradient
  must move the weight UP).
  7 is mandatory: the answer must say the corrected ratio is sign(g) and the
  first step is exactly $\eta$. An answer that says Adam "picks its own
  learning rate" is U5M3 and fails - $\eta$ still sets the scale, which is
  precisely what step 7 shows.
check: llm
```

## What gradient descent actually finds

<!-- refutes: M5 -->

**You probably think gradient descent is searching for the minimum of the loss,
and that a training run either finds it or gets stuck short of it in a local
minimum.** That is the picture in every optimization tutorial: a bumpy 2D
surface, a ball rolling into whichever valley it happened to start above, the
global minimum sitting elsewhere.

**Here is the prediction that fails.** If training found *the* minimum, two runs
of the same architecture on the same data with different random seeds would
converge to the same weights, or at least to weights related by a known
symmetry. They do not. Take two identically-configured runs of any large model,
compare the weight tensors, and you get essentially unrelated matrices. Average
them naively and you get a model that produces garbage. Yet both runs reach
almost the same loss and both work. A search for a unique target does not
behave like that.

**Why it is appealing:** the loss is a function, functions have minima,
"minimize the loss" is literally what the code says. And in the 2D pictures you
have seen, local minima genuinely are the failure mode.

**What is actually true.** Two things, and both are consequences of dimension.

First, in $10^{11}$ dimensions, local minima are rare. A critical point (zero
gradient) is a local minimum only if the loss curves *upward* in all $10^{11}$
directions simultaneously. If curvature signs were even loosely independent
across directions, that is a coin flip won $10^{11}$ times. The overwhelmingly
more likely critical point is a **saddle**: up in some directions, down in
others. And a saddle is not a trap, because there is always a descent direction
available; stochastic gradient noise from minibatch sampling is enough to find
it. The 2D intuition fails not because it is a simplification but because it is
the wrong regime.

Second, there is no "the" minimum to find. Permute the hidden units of any MLP
layer, permuting the corresponding weight rows and columns to match, and you get
a numerically different network computing the identical function. For a layer of
width $d$ that is $d!$ equivalent weight settings, per layer, multiplied across
layers. The global minimum, if it exists, is an astronomically large set of
points, and that is before counting genuinely different solutions with
comparable loss.

What SGD actually does is find *a* low-loss region. Which one depends on
initialization, data order, and hardware nondeterminism. The empirically useful
distinction is not global versus local, it is **wide versus sharp**: parameter
settings sitting in broad flat basins, where perturbing the weights barely moves
the loss, generalize better than settings in narrow steep ones. Small batch
sizes and higher learning rates inject noise that makes it hard to settle into a
narrow basin, which is one honest mechanism behind "too large a batch hurts
generalization".

So when a training run plateaus, "stuck in a local minimum" is almost certainly
the wrong diagnosis. Look at learning-rate schedule, data quality, gradient
clipping, or numerical issues instead.

```beat
id: u5-b8
type: self-explain
concept: c-optimizers
prompt: |
  Your team trains the same model twice with different seeds. Both reach loss
  $2.14$. A teammate proposes averaging the two weight tensors elementwise to
  get "the best of both", the way you would average two estimates of a
  quantity. Explain in your own words what will actually happen and why.
answer: |
  The averaged model will be far worse than either, likely near-random. The two
  runs did not converge to nearby points in weight space; they landed in
  different low-loss regions, and the loss surface between two such regions is
  not low. Concretely, hidden units are permutation-symmetric: run A's unit 7
  may compute what run B's unit 200 computes, so averaging adds together
  features that have nothing to do with each other. Averaging is valid for
  estimates of one quantity; these are two different parameterizations of
  similar functions, not two noisy measurements of one parameter vector. It
  works only when the checkpoints share a training trajectory (weight
  averaging along one run, or model soups from a common fine-tuning ancestor).
rubric: |
  Must contain (1) the averaged model performs badly, and (2) a reason grounded
  in there being many distinct low-loss regions rather than one minimum -
  permutation symmetry, or "the midpoint of two basins is not in a basin", both
  count. Mentioning that averaging DOES work for checkpoints sharing a
  trajectory = full credit. An answer predicting the average is fine or
  slightly better = fail, cites M5. An answer saying "it fails because of local
  minima" without the multiplicity/symmetry argument = partial.
check: llm
```

## Scaling laws: what more parameters buy

Pretraining consumes three resources: parameters $N$, training tokens $D$, and
compute $C$. They are tied together by a rule of thumb that is accurate enough
to plan with:

$$C \approx 6ND$$

$C$ in FLOPs, $N$ = parameter count, $D$ = tokens processed. The 6 is 2 FLOPs
per parameter per token for the forward pass (a multiply and an add) plus
roughly twice that for the backward pass.

The empirical finding, replicated across many labs, is that held-out loss
follows a smooth power law in $N$ and $D$. The Hoffmann et al. (Chinchilla) fit:

$$L(N, D) = E + \frac{A}{N^{\alpha}} + \frac{B}{D^{\beta}}$$

with fitted constants $E = 1.69$, $A = 406.4$, $B = 410.7$, $\alpha = 0.34$,
$\beta = 0.28$. Read the structure before the numbers. There are three terms: an
irreducible constant $E$, which is the entropy floor from the previous section
made concrete as a fitted parameter; a term that shrinks as you add parameters;
and a term that shrinks as you add data. Neither shrinking term ever reaches
zero. Evaluate it:

| $N$ | $D$ | $A/N^{\alpha}$ | $B/D^{\beta}$ | $L$ | perplexity | excess over floor |
|---|---|---|---|---|---|---|
| $10^{9}$ | $2 \times 10^{10}$ | 0.354 | 0.536 | 2.580 | 13.2 | 0.890 |
| $10^{10}$ | $2 \times 10^{11}$ | 0.162 | 0.281 | 2.133 | 8.44 | 0.443 |
| $10^{11}$ | $2 \times 10^{12}$ | 0.074 | 0.148 | 1.912 | 6.76 | 0.222 |

Each row is 10x the model and 10x the data of the row above. Loss drops by
$0.45$, then $0.22$. The excess over the floor halves each time. This is the
actual shape of "scale works": not a cliff, not a plateau, a predictable
straight line on a log-log plot that lets you forecast the loss of a model you
have not trained.

<!-- refutes: M11 -->

**You probably think more parameters means more memorized training data, that
capacity is storage and a bigger model is a bigger disk holding more facts.**
The instinct is nearly universal among engineers, and it is reinforced by every
"the model has 405 billion parameters" headline, which sounds like a capacity
spec.

**Here is the prediction that fails.** If parameters were storage for training
examples, then adding parameters at fixed data would improve training loss and
*degrade* held-out loss. That is the textbook overfitting curve, and it is what
a memorizer does: more capacity, more of the training set absorbed verbatim,
worse generalization. Measured behavior is the opposite. Held-out loss improves
smoothly and predictably with $N$, over more than five orders of magnitude, with
no U-turn. A pure storage device cannot do that, because the held-out data is by
construction not in storage.

The second failing prediction is sharper. A lookup table can only answer queries
about what it stored. Scaled models answer questions no training example
demonstrates: they translate between language pairs absent from the corpus, they
execute instruction formats they never saw, and they do arithmetic on numbers
that appear nowhere. Meanwhile the *same* mechanism confidently produces
citations, APIs, and dates that were never in any document. Generalization and
hallucination are the same behavior viewed from two sides, and neither is
possible for a store.

**Why it is appealing:** models do reproduce memorized text sometimes, and
larger models memorize more of it, so the correlation is real. The bit-counting
argument feels rigorous.

**What is actually true.** Capacity buys **feature composition**, not rows.
Additional parameters mean more directions in the residual stream, more
attention heads implementing more relational patterns, wider MLPs representing
more features per layer, and more layers composing those features into higher
order ones. What improves with scale is the richness of the function the network
can express, and the loss improvement shows up on data the model has never seen.
Memorization exists, dominates in the small-data regime, and is the exception,
not the mechanism.

Run the arithmetic on the bit-counting claim to see how badly it loses. A 70B
parameter model in bf16 is 140GB. Its training corpus at 15T tokens is on the
order of 50TB of text. Storage would require compressing 50TB into 140GB
losslessly enough to reproduce arbitrary passages, a ratio of roughly 357:1 on
natural language, well past any achievable lossless bound. The model is not storing the
corpus. It is storing a function that predicts it.

### Where the parameters should go

Given a fixed compute budget $C$, and $C \approx 6ND$, you choose: a large model
on few tokens, or a smaller model on many. This is a real decision with real
money attached, and the field got it wrong for several years, training models
far larger than their token budgets justified. Chinchilla's result is that $N$
and $D$ should scale roughly *together*, with a rule of thumb near **20 tokens
per parameter**.

Take $C = 1.2 \times 10^{21}$ FLOPs and evaluate two allocations with the fit
above:

- $N = 10^{10}$, $D = C/(6N) = 2 \times 10^{10}$ tokens (2 tokens per
  parameter): $L = 2.388$.
- $N = 3.16 \times 10^{9}$, $D = 6.32 \times 10^{10}$ tokens (20 tokens per
  parameter): $L = 2.318$.

Same compute, same dollar cost. The three-times-smaller model wins, because the
larger one starved for data. And it wins twice, because it is also three times
cheaper to *serve*, forever, which is why inference-heavy deployments push token
counts well past 20:1 even though that is no longer compute-optimal for
training. Compute-optimal is a statement about the training budget alone.

```beat
id: u5-b9
type: predict
concept: c-scaling
prompt: |
  Using $L(N, D) = 1.69 + 406.4/N^{0.34} + 410.7/D^{0.28}$: you have a model
  with $N = 10^{10}$ and $D = 2 \times 10^{11}$, at $L = 2.133$. Your budget
  allows exactly one of: (a) 10x the parameters at the same data, or (b) 10x
  the data at the same parameters. Predict which gives the larger loss
  reduction and by roughly how much, then say what limits how far this can go.
answer: |
  (a) 10x parameters: the $N$ term goes from $0.162$ to $0.074$, saving
  $0.088$. (b) 10x data: the $D$ term goes from $0.281$ to $0.148$, saving
  $0.133$. Option (b) wins, because at $D/N = 20$ the data term is the larger
  of the two and the one with more room to shrink. What limits it: both terms
  approach zero but the total approaches $E = 1.69$, the irreducible entropy
  floor, and never goes below it. Also note (a) and (b) do not cost the same -
  $C \approx 6ND$ makes both a 10x compute increase here, so the comparison is
  fair on compute but (a) additionally costs 10x more to serve.
rubric: |
  Must (1) identify option (b) as the larger reduction, (2) justify it by
  comparing the two shrinking terms rather than by intuition, and (3) name the
  irreducible floor $E = 1.69$ as the limit. Any 2 of 3 = pass. Predicting that
  loss can be driven arbitrarily low with enough scale = fail, cites M18.
  Predicting (a) because "parameters matter more" = fail, cites M11.
check: llm
```

## Post-training: SFT, RLHF, RLAIF, DPO

Everything so far describes **pretraining**: next-token cross-entropy on a large
corpus, which yields a **base model**. A base model is a very good text
continuation engine and a poor assistant. Ask it a question and it may answer,
or continue with three more questions, or emit a table of contents, because all
of those are plausible continuations of a document containing your question.
Post-training turns that into something that answers.

**Supervised fine-tuning (SFT).** The same cross-entropy loss, the same backprop,
the same AdamW. Only the data changes: curated (prompt, response) pairs written
or vetted by humans, formatted with role delimiters. One mechanical detail
matters: the loss is **masked** over the prompt tokens, so the gradient comes
only from the response tokens. The model is not learning to predict the user's
question, it is learning what follows one. Typical scale is $10^4$ to $10^6$
examples. SFT can only push probability toward responses that were demonstrated,
and it has no way to express "this response is worse than that one", because
cross-entropy only ever says "this is the target".

**RLHF.** Preference data instead of demonstrations. Sample two responses to the
same prompt, ask a human which is better, collect pairs $(y_w, y_l)$ for winner
and loser. Then:

1. Train a **reward model** $r_\phi(x, y)$, a copy of the language model with the
   unembedding replaced by a scalar head, on the objective
   $-\log \sigma(r_\phi(x, y_w) - r_\phi(x, y_l))$ with $\sigma$ the logistic
   function. This says: make the winner score higher than the loser. Note that
   only the *difference* is supervised, so the reward scale is arbitrary.
2. Optimize the language model against that reward with a policy gradient method
   (PPO), maximizing
   $\mathbb{E}_{y \sim \pi_\theta}[r_\phi(x,y)] - \beta \, \mathrm{KL}(\pi_\theta \| \pi_{\text{ref}})$,
   where $\pi_\theta$ is the model being trained, $\pi_{\text{ref}}$ is the frozen
   SFT model, and $\beta$ controls how far the model may drift. The KL term is
   load-bearing: without it the model finds inputs where the reward model is
   wrong and exploits them, producing high-reward gibberish. That failure has a
   name, reward hacking, and it is the standard outcome of setting $\beta$ too low.

The mechanical difference from SFT is that RLHF samples from the *current* model
and scores its own outputs, so it can push probability away from things and it
can reach responses no human wrote.

**RLAIF.** Structurally identical to RLHF, with the preference labels produced
by a model following a written specification of principles rather than by human
raters. It exists because human preference collection is the throughput
bottleneck and because a written spec is auditable and editable in a way that
the aggregate taste of a rater pool is not.

**DPO.** The observation that step 1 and step 2 above can be collapsed. The
optimal policy under the KL-constrained reward objective has a closed form in
terms of the reward, and that relation can be inverted to write the reward in
terms of the policy. Substituting it into the reward model's own loss gives an
objective in the policy directly:

$$L_{\text{DPO}} = -\log \sigma\!\left(\beta \log \frac{\pi_\theta(y_w \mid x)}{\pi_{\text{ref}}(y_w \mid x)} - \beta \log \frac{\pi_\theta(y_l \mid x)}{\pi_{\text{ref}}(y_l \mid x)}\right)$$

No reward model, no sampling during training, no RL loop. It is a supervised
loss over preference pairs that you can train with the same code path as SFT.
The gradient raises $\log \pi_\theta(y_w)$ and lowers $\log \pi_\theta(y_l)$,
weighted by how badly the current model has the pair ordered, with the frozen
reference in the denominator doing the job the KL penalty did. The cost of the
simplification is that it is **offline**: it can only learn from the pairs you
have, whereas PPO keeps generating fresh samples from the improving policy.

Four methods, one axis to keep them straight: what shape of supervision does the
loss consume? SFT consumes a target ("produce this"). RLHF and RLAIF consume a
learned scalar over the model's own samples ("that was worth 0.7"). DPO consumes
a pair ("this one over that one"), directly.

<!-- refutes: M15 -->

**You probably think post-training is where the model learns what it knows,
because the difference between a base model and a chat model is so dramatic that
it feels like the chat model knows vastly more.** Everyone who has prompted a
raw base model has this reaction.

**Here is the prediction that fails.** If post-training added knowledge,
knowledge benchmarks would jump after it. Measure MMLU or any factual QA
benchmark on a base model with careful few-shot prompting, then on the
post-trained version of the same checkpoint: the scores move by a couple of
points, and sometimes down. The dramatic gain is in following the instruction,
formatting the answer, and refusing to continue as a document. Not in knowing.

The compute accounting says the same thing more bluntly. Pretraining processes
on the order of $1.5 \times 10^{13}$ tokens. An SFT set of $10^{5}$ examples at
500 tokens each is $5 \times 10^{7}$ tokens: **0.0003%** of the pretraining
token budget. There is no mechanism by which a 0.0003% perturbation installs a
world model. It is enough to change which of the model's existing behaviors gets
selected, and that is exactly what it does.

**Why it is appealing:** the behavior change is enormous and it is the only
phase most people can observe or run themselves. Post-training is also where the
labs' visible effort goes, so it gets the attention.

**What is actually true.** Capabilities come from pretraining. Post-training
shapes behavior: which distribution of continuations the model draws from, what
format it uses, what it declines. The base model could already write a polite,
correct, well-formatted answer to your question; that continuation had some
probability under the base model, competing with a hundred other plausible ones.
Post-training raises the probability of that mode and lowers the rest. This is
also the honest explanation for why post-training cannot fix hallucination: if
the capability is not in the base model, no amount of preference data conjures
it, and the same preference optimization that rewards helpful answers also
rewards confident-sounding ones.

```beat
id: u5-b10
type: self-explain
concept: c-posttrain
prompt: |
  A product manager asks: "Our model does not know about our internal API. Can
  we fix that with RLHF? We can generate a few thousand preference pairs
  showing good API answers preferred over bad ones." Explain in your own words
  what will happen if you do this, and what the right intervention is.
answer: |
  It will not install the API knowledge. Preference optimization changes which
  of the model's existing continuations get probability mass; it cannot create
  a mapping from endpoint names to behavior that was never in the weights. The
  likely outcome is worse than nothing: the pairs teach the model that
  confident, well-formatted API answers are preferred, so it produces
  confident, well-formatted, invented endpoints. You have optimized the style
  of the hallucination. The right interventions are ones that put the
  information in front of the model as capability or as context: continued
  pretraining or SFT on the actual API documentation (many tokens of real
  content, not preference pairs), or retrieval that places the docs in the
  context window at inference time, which is usually the cheaper and more
  maintainable option since the API will change.
rubric: |
  Must contain (1) preference optimization shapes behavior/selection, not
  knowledge, and (2) the specific failure mode that it will produce
  confident-sounding fabrications, and (3) at least one correct alternative
  (SFT/continued pretraining on the real docs, or retrieval/in-context). Any 2
  of 3 = pass, but (2) alone without (1) = partial. Answering that RLHF will
  work given enough pairs = fail, cites M15. Answering that nothing can teach
  the model the API = fail; pretraining-scale data and retrieval both work.
check: llm
```

## What you now have

A loss that scores a distribution against reality and has a floor above zero. A
backward pass that assigns every one of $10^{11}$ weights its exact share of
blame in one sweep, at the same cost as the forward pass. An update rule whose
step size is set by gradient consistency rather than gradient magnitude, moving
the weights into one of astronomically many good-enough regions. A scaling
relationship that says what more of everything buys and predicts it before you
spend the money. And a post-training phase that selects among behaviors the
pretrained model already had.

That is a trained model. The next unit is what it costs to run one.
