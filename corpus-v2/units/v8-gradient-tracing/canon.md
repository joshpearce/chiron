---
unit: v8
title: "Tracing by gradient"
concepts:
  - d-gradient-similarity
  - d-tracin
  - d-influence-vs-entailment
  - d-inrun-shapley
assumes:
  - d-counterfactual
  - d-shapley-axioms
  - d-eval-noise
---

The counterfactual answers "what did this source contribute" exactly, and it
charges you a training run per question. This unit is about the family of
methods that promise the same answer out of the training run you already did -
and about the discipline that keeps that promise from becoming a lie. By the end
you can compute a gradient dot product by hand and say what it means, implement
TracIn correctly including the two mistakes that corrupt most implementations,
make it fit on a laptop with a random projection and per-source accumulation,
instrument a Shapley-style royalty ledger into a training loop for about 7.5%
overhead, and state - with a number - exactly how much any of it can be
trusted.

That last one is the unit. Every method here is cheap, plausible, and
unfalsifiable on its own. The only thing that makes one of them evidence rather
than decoration is the rank correlation between its output and a counterfactual
table you measured yourself.

A note on numbers. Gradient vectors, learning rates, and the small worked
examples are **illustrative** - two and three components with clean arithmetic,
so you can check every digit. The hardware costs, the storage arithmetic, and
the published research results are real and sourced.

## The bill for one honest answer

You have a table. Twelve rows, one per source, each carrying a contribution and
an error bar, produced by retraining the model without each source and
subtracting. It cost 39 training runs on a laptop and it answers one question:

> Across everything this model does, which sources mattered?

Here is the question a user actually asks, and it is not that one.

> The model just told me that peer review at Nature averages 3.9 reviewers per
> submission. **Why did it say that?** Which of my twelve sources is
> responsible for that sentence?

Same word "responsible", different object. The table is a property of the
model. This is a property of one output. And there is nothing wrong with the
counterfactual as a definition here - remove source $i$, retrain, and see
whether the model still says it. That is still exactly what "responsible" means.
The only thing that changed is the scale of the ambition, so price it.

Take the realistic artifact rather than the laptop toy: a 561-million-parameter
model trained on 11.2 billion tokens. That run is a measured receipt - 3 hours
51 minutes on a rented 8xH100 node, $92 as of mid-2026. To answer one query by
counterfactual you need the same 13 conditions the leave-one-out table used
(the full corpus, plus one run per source removed), and you need more seeds than
before, not fewer: the table averaged over a whole held-out file, while this
measures the probability of one sentence, and a single sentence's probability is
a far noisier quantity than a corpus-wide average. Call it 3 seeds and know you
are being generous.

| what you are buying | runs | node-hours | dollars |
|---|---|---|---|
| the leave-one-source-out table (once) | 39 | 150 | $3,588 |
| **one query answered by counterfactual** | 39 | 150 | **$3,588** |
| a thousand queries in a day | 39,000 | 150,000 | $3,588,000 |

The middle row is the whole problem. Answering "why did you say that" costs the
same as building the entire ground-truth table, every single time, because the
retraining is what costs money and you have to redo all of it for each new
output you want explained. And the bottom row is not a business: 150,000
node-hours in a 24-hour day is 6,250 rented nodes running continuously, which is
fifty thousand H100s, to serve a thousand questions.

The one line that produces those numbers is worth having in your hands rather
than taking on faith, because you will use it constantly. For each parameter and
each token, the forward pass does one multiply and one add - 2 operations. The
backward pass has to produce two things for that same parameter: the gradient
with respect to the parameter itself, and the gradient with respect to the
activation feeding it, so blame keeps flowing to earlier layers. Two
multiply-accumulates, 4 operations. Six per parameter per token, so the total
floating-point operations for a run are

$$C \approx 6ND$$

where $C$ is operations for the whole run, $N$ is the parameter count, and $D$
is the number of training tokens. Nothing else in this formula, and nothing
about it depends on the architecture beyond every parameter touching every
token.

```beat
id: v8-b1
type: compute
concept: d-tracin
prompt: |
  Price one counterfactual answer at realistic scale.

  The model has $N = 5.61 \times 10^8$ parameters and trains on
  $D = 1.12 \times 10^{10}$ tokens. The rented node sustains
  $3.2 \times 10^{15}$ floating-point operations per second and bills at $24
  per hour. The protocol is 13 conditions (full corpus plus one per source
  removed) at 3 seeds each.

  Step 1. $C = 6ND$ for one run.
  Step 2. Seconds for one run, then hours.
  Step 3. Dollars for one run, then for all 39.

  Report only the final figure: dollars to answer one query, to the nearest
  dollar.
answer: 3063
check: numeric(30)
```

That beat's answer comes out below the $3,588 in the table, and the gap is
worth one sentence: $6ND$ prices pretraining only, while the $92 receipt covers
tokenizer training, midtraining, fine-tuning and evaluation as well. Both
numbers are right about different things, and neither is small.

### What could possibly stand in

So the counterfactual is unaffordable per query and it is also the only thing
that is definitionally correct. That is the shape of every interesting
engineering problem: the right answer exists and costs too much, so you look for
something already lying around that correlates with it.

Look at what a training run actually produces and then discards. It produces
one final set of weights, which you keep. It produces a loss curve, which you
plot. And at every one of hundreds of thousands of steps, it computes - for
every example in the batch - a full-sized vector saying exactly how that example
wanted to change every parameter in the model. Then it averages those vectors
together, takes one step, and throws all of them away.

```beat
id: v8-b2
type: predict
concept: d-tracin
prompt: |
  Before reading on, commit to an answer.

  A training run computes an enormous amount of per-example information and
  then throws nearly all of it away. Something in there could plausibly stand
  in for a counterfactual - but only if it has a specific property, and only
  inside a specific limit.

  Which account names the right artifact, the property the substitution would
  need, and the thing that artifact can never tell you?
options:
  - text: |
      The artifact is the per-example gradient: at each step every example in
      the batch produces a vector the shape of the parameters, and training
      averages them, takes one step, and discards the individuals. The
      property it needs is that the dot product of a training example's
      gradient with a query's gradient predicts what actually happens to that
      query's loss when the model steps on that example - a first-order claim
      that can be checked by taking the step and measuring. What it cannot
      tell you is the counterfactual itself: gradients only ever describe the
      trajectory that happened, and removing a source changes the model every
      later example then sees.
    correct: true
    explain: |
      Right. The per-example gradient is the discarded artifact, its warrant
      is the checkable first-order prediction
      $\Delta\ell(z_q) \approx -\eta\,(g_{z_q} \cdot g_z)$, and its ceiling is
      that no quantity computed along one trajectory can observe a different
      one. That gap is why every method in this unit has to be graded rather
      than trusted.
  - text: |
      The artifact is the per-example gradient, and the property it needs is
      already established by the derivation: since
      $\Delta\ell(z_q) \approx -\eta\,(g_{z_q} \cdot g_z)$ follows from
      calculus, the summed dot products simply are the counterfactual computed
      cheaply, and nothing about the substitution needs measuring. The only
      thing it cannot tell you is anything at frontier scale, where the
      gradients are too large to store.
    misconception: D6
    explain: |
      The artifact is right and the licence is not. The expansion is exact
      only to first order along the trajectory that happened; the
      counterfactual is about a trajectory that did not. Measured against
      actual retraining on a 2B model, the methods that scale come out no
      better than random guessing, and correcting a single assumption in the
      derivation moved published results by 10 to 300%. Whether the
      substitution holds is an empirical question with a number attached, not
      a corollary.
  - text: |
      The artifact is the per-example gradient, and the property it needs is
      that gradients which align belong to examples about the same subject
      matter - so attribution is retrieval carried out in the model's own
      representation instead of in token space. What it cannot tell you is
      anything about a document that never mentions the topic of the output.
    misconception: V8-M1
    explain: |
      The dot product measures whether two examples push the *parameters* the
      same way, which is a statement about the model's current mechanism, not
      about meaning. At 52-billion-parameter scale, flipping the word order of
      a key phrase collapses influence to near zero with the content
      unchanged, and two documents on unrelated subjects can be strong allies
      because both push the same numeral format or citation habit.
  - text: |
      The artifact is the record of which training text the trained model can
      reproduce verbatim. The property it needs is that the model emits the
      passage on prompting, which is directly checkable; what it cannot tell
      you is how much that passage was worth in dollars.
    misconception: D3
    explain: |
      Verbatim emission is not the discarded per-example quantity - it is a
      property of the finished weights, computed after the fact. It is also a
      lower bound: most of what training data contributes is semantic and
      never resurfaces verbatim, and asking for the same content in a
      different style defeats any n-gram filter while the knowledge stays
      intact.
check: choice
```

Those discarded vectors are the ledger this unit is about. They are already
paid for. The question is what they mean.

## A gradient is a direction

Take one training example and freeze everything else. The model has parameters,
and there is a number saying how badly the model does on that one example. The
gradient is the answer to "which way should I move the parameters to make that
number go down fastest, right here."

Small enough to check by hand. Let the model have exactly two parameters,
$w_1$ and $w_2$, and let it predict from two input features:

$$\hat{y} = w_1 x_1 + w_2 x_2$$

where $x_1, x_2$ are the example's two input numbers and $\hat{y}$ is the
prediction. Score the prediction against the true value $y$ with squared error:

$$\ell(w, z) = (\hat{y} - y)^2$$

Here $z = (x_1, x_2, y)$ is one complete example and $\ell$ is its loss - one
number, for one example, at one setting of the parameters. Everything in this
unit is built on per-example loss, never on the batch average, and that is the
distinction that makes attribution possible at all.

Now differentiate. The partial derivative with respect to $w_1$ asks: nudge
$w_1$ alone, holding $w_2$ fixed, how much does $\ell$ move? Since $\hat{y}$
moves by $x_1$ for each unit of $w_1$, and $\ell$ moves by $2(\hat y - y)$ for
each unit of $\hat y$, the two sensitivities multiply:

$$\frac{\partial \ell}{\partial w_1} = 2(\hat{y} - y)\, x_1
\qquad
\frac{\partial \ell}{\partial w_2} = 2(\hat{y} - y)\, x_2$$

Stack them into one vector, which is the gradient, written $\nabla_w \ell$ or
just $g$:

$$g_z = \nabla_w \ell(w, z) = 2(\hat{y} - y)\begin{bmatrix} x_1 \\ x_2\end{bmatrix}$$

Two facts to carry. The gradient has the same shape as the parameters - two
parameters, two components; 561 million parameters, 561 million components. And
$-g_z$ is the direction that most reduces *that example's* loss. Take a small
step along it and that example gets better.

<!-- fade: v8-gradient-dotproduct -->

**Worked, with numbers.** Fix the parameters at $w = (1, 1)$ and take four
examples.

| example | $x$ | $y$ | $\hat{y} = w_1x_1 + w_2x_2$ | $\hat y - y$ | $g = 2(\hat y - y)x$ |
|---|---|---|---|---|---|
| A | $(2, 0)$ | 3 | 2 | $-1$ | $(-4, 0)$ |
| B | $(1, 0)$ | 2 | 1 | $-1$ | $(-2, 0)$ |
| C | $(0, 1)$ | $-1$ | 1 | $2$ | $(0, 4)$ |
| D | $(1, 0)$ | 0 | 1 | $1$ | $(2, 0)$ |

Now compare pairs of gradients with the only tool that compares two vectors: the
dot product, which multiplies matching components and adds the results.

$$g_A \cdot g_B = (-4)(-2) + (0)(0) = 8$$
$$g_A \cdot g_C = (-4)(0) + (0)(4) = 0$$
$$g_A \cdot g_D = (-4)(2) + (0)(0) = -8$$

Three verdicts, and the vocabulary for the rest of the unit. A and B are
**allies**: positive dot product, their gradients point the same way, so
whatever helps one helps the other. A and C are **neutral**: zero, perpendicular
directions, moving on one leaves the other untouched to first order. A and D are
**rivals**: negative, opposed directions, and progress on one costs the other.

### Checking that the verdict is true

That vocabulary is a claim about what happens, so make it happen. Take one
gradient-descent step on example B alone, with step size $\eta = 0.1$:

$$w \leftarrow w - \eta\, g_B = \begin{bmatrix}1\\1\end{bmatrix} - 0.1\begin{bmatrix}-2\\0\end{bmatrix} = \begin{bmatrix}1.2\\1\end{bmatrix}$$

What happened to A, which was never touched? Before the step, $\hat y_A = 2$ and
$\ell_A = (2-3)^2 = 1$. After, $\hat y_A = 1.2 \times 2 = 2.4$ and
$\ell_A = (2.4-3)^2 = 0.36$. A's loss fell from 1 to 0.36 because the model took
a step on a different example. That is training data influencing an output, in
miniature, with every digit visible.

Take the step on D instead: $w \leftarrow (1,1) - 0.1(2, 0) = (0.8, 1)$, so
$\hat y_A = 1.6$ and $\ell_A = (1.6-3)^2 = 1.96$. A got worse. Rivals, as
advertised.

Now the part that matters, because it is where every method in this unit gets
its licence and its limits. The dot product does not merely predict the sign of
the change, it predicts the size. To first order, moving the parameters by
$\Delta w$ changes A's loss by $g_A \cdot \Delta w$, and $\Delta w = -\eta g_B$,
so

$$\Delta \ell_A \approx -\eta\, (g_A \cdot g_B)$$

Check it. At $\eta = 0.1$: predicted $-0.1 \times 8 = -0.8$, actual $-0.64$. At
$\eta = 0.01$: the step gives $w = (1.02, 1)$, so $\ell_A = (2.04-3)^2 = 0.9216$
and the actual change is $-0.0784$ against a predicted $-0.08$. The
approximation got ten times better when the step got ten times smaller, because
the error is second order in $\eta$ - it is the curvature the straight-line
prediction ignores.

That is the entire theoretical content of gradient-based attribution, and it is
honest about itself: **the learning-rate-weighted gradient dot product is the
first-order prediction of how much one example's step moved another example's
loss, and it is accurate to the extent that steps are small.**

```beat
id: v8-b3
type: compute
concept: d-gradient-similarity
prompt: |
  A query point $z_q$ has gradient $g_q = (2, -1, 3)$ at the current
  parameters. Three training examples have gradients

  $$g_1 = (1, 0, 2) \qquad g_2 = (-1, 2, 0) \qquad g_3 = (3, 3, -1)$$

  Compute the three dot products $g_q \cdot g_1$, $g_q \cdot g_2$,
  $g_q \cdot g_3$ by hand. Multiply matching components, add.

  Answer in the form `[d1, d2, d3]`.
answer: "[8, -4, 0]"
check: exact
```

```beat
id: v8-b4
type: completion
concept: d-gradient-similarity
# fade: v8-gradient-dotproduct, stage 2 of 3. The blanked steps are the
# residual, the gradient, and the dot product, which carry the concept; the
# forward value is given because computing it is taught, not tested, here.
# variants: blank P's residual and gradient instead of Q's, which tests the
# positive-residual case rather than the negative one. Or give both gradients
# and blank only the dot product and the verdict.
prompt: |
  Fill the blanks. Same toy model: $\hat y = w_1 x_1 + w_2 x_2$,
  $\ell = (\hat y - y)^2$, gradient $g = 2(\hat y - y)\,(x_1, x_2)$.
  Parameters are $w = (2, 1)$.

      Example P: x = (1, 3), y = 4
        y_hat    = 2(1) + 1(3) = 5
        residual = 5 - 4 = 1
        g_P      = 2(1)(1, 3) = (2, 6)

      Example Q: x = (0, 2), y = 6
        y_hat    = 2(0) + 1(2) = 2
        residual = ____            <- A
        g_Q      = ____            <- B

      g_P . g_Q = ____             <- C

      Verdict (ally / neutral / rival): ____   <- D

  Which filling of A, B, C, D is correct?
options:
  - text: |
      A: -4, B: (0, -16), C: -96, D: rival. The residual is $2 - 6 = -4$, so
      the scalar $2(-4) = -8$ multiplies the input $(0, 2)$ to give
      $(0, -16)$; the dot product is $(2)(0) + (6)(-16) = -96$.
    correct: true
    explain: |
      Right, and the sign is the content of the verdict: a negative dot
      product means a step on P raises Q's loss and a step on Q raises P's.
      Both factors matter - the residual $2(\hat y - y)$ sets the sign and
      scale, the input $(x_1, x_2)$ sets the direction.
  - text: |
      A: -4, B: (0, 2), C: 12, D: ally. Q's gradient is its input vector
      $(0, 2)$, and since P and Q both load on the same second feature they
      push the model the same way, giving $(2)(0) + (6)(2) = 12$.
    misconception: V8-M1
    explain: |
      Dropping the residual factor turns the dot product into a comparison of
      the examples' *inputs* - which is exactly the content-similarity reading
      the chapter refutes. The gradient is $2(\hat y - y)(x_1, x_2)$; with
      $\hat y - y = -4$ the scalar is $-8$, so $g_Q = (0, -16)$ and
      $g_P \cdot g_Q = -96$. P and Q are rivals despite sharing a feature.
  - text: |
      A: -4, B: (0, -16), C: -0.95, D: rival. Divide the dot product by the
      two vectors' lengths so the number sits on a scale comparable with other
      pairs, other queries and other models.
    misconception: V8-M3
    explain: |
      The residual and the gradient are right, but $g_P \cdot g_Q$ is $-96$;
      $-0.95$ is the cosine, a different quantity. Normalizing by the query
      gradient's length is a legitimate move when you need cross-query
      comparability, and it still does not make a score portable: there are no
      units and no shared zero across models or projections. What is
      comparable is the ranking, after it has been graded.
check: choice
```

### What the dot product is not

<!-- refutes: V8-M1 -->

**You probably read a positive gradient dot product as "these two examples are
about the same thing."** Allies share a topic, rivals do not, and gradient
similarity is a fancy semantic similarity. Under that belief, attribution is
retrieval with better math: find the training documents that resemble the
output, and you have found the influential ones.

**Here is the prediction that fails.** If gradient alignment tracked content,
then rewriting a training document as a close paraphrase - same facts, same
entities, same topic, different word order - would leave its influence roughly
unchanged. It does not. Measured at 52-billion-parameter scale by the largest
influence study anyone has run, flipping the word order of a key phrase
**collapses influence to near zero**. The content is identical and the estimator
reports nothing. It was tracking token-order-sensitive gradient alignment the
whole time.

**Here is why the wrong model is appealing.** It is nearly right at small scale.
Small models are shaped heavily by surface tokens, so their influential examples
really do look lexically like the output, and every screenshot in every
attribution demo is drawn from that regime. It also fits the intuition you
already carry from embedding search, where similarity of representation is the
whole game.

**Here is what is actually true.** The dot product measures whether two examples
push the *parameters* the same way. That is a statement about the model's
current mechanism, not about the examples' meaning. Two documents on completely
unrelated subjects can be strong allies because both push the same low-level
machinery - a numeral format, a citation convention, a code-block habit. And two
documents saying the identical thing can be near-orthogonal because one phrases
it in a construction the model has already mastered, so its gradient is small
and points somewhere else entirely. The same study found the effect runs the
other way with scale too: at larger sizes influence becomes *more* abstract,
matching concepts rather than tokens, including across languages. Gradient
alignment is a mechanical quantity that sometimes coincides with semantic
similarity and is not a measurement of it.

## TracIn: influence as a sum over checkpoints

The toy example gave a first-order prediction for one step. A real training run
is hundreds of thousands of steps, and an example only participates in the ones
whose batch contained it. So the idealized story writes itself.

**The idealized story.** Training walks the parameters from their initialization
$\theta_0$ through a sequence of steps to their final value. Watch the loss on
one query point $z_q$ as that walk proceeds. At each step, the loss on $z_q$
moves a little. Attribute each step's movement to the examples in that step's
batch, and add up, per example, every movement it was responsible for. When the
walk ends you have decomposed the total change in the query's loss - from
"untrained model" to "trained model" - into a sum over training examples. That
decomposition is the influence of each example on that query, and it is a
telescoping identity, not an approximation: the pieces add to the whole because
you defined them as the steps of one walk.

Each piece is the thing already derived. If step $t$ used learning rate $\eta_t$
and took its step on training example $z$, the change in the query's loss at
that step is, to first order,

$$\Delta \ell(z_q) \approx -\eta_t\, \nabla_\theta \ell(\theta_t, z_q) \cdot \nabla_\theta \ell(\theta_t, z)$$

where $\theta_t$ is the parameter vector at step $t$, $\nabla_\theta \ell(\theta_t, z_q)$
is the query's gradient there, and $\nabla_\theta \ell(\theta_t, z)$ is the
training example's gradient there. Both are vectors with one component per
parameter. Positive dot product means the query's loss went down, so drop the
minus sign and define influence so that **positive means helped** - the same
sign convention the leave-one-out table uses.

**The problem with the idealized story** is that it needs $\theta_t$ at every
step, which means storing hundreds of thousands of full model checkpoints. Nobody
does that. What people do keep is a handful of checkpoints.

**The checkpoint approximation.** Sum over the checkpoints you saved instead of
the steps you took:

$$\text{TracIn}(z, z_q) = \sum_{c \in \mathcal{C}} \eta_c \; \nabla_\theta \ell(\theta_c, z_q) \cdot \nabla_\theta \ell(\theta_c, z)$$

where $\mathcal{C}$ is the set of saved checkpoints, $\theta_c$ is the parameter
vector at checkpoint $c$, and $\eta_c$ is the learning rate in effect there.
Each checkpoint contributes one learning-rate-weighted dot product, and you add
them. (The script $\mathcal{C}$ is a set of checkpoints; the plain $C$ two
sections ago was floating-point operations. They are different objects and this
is the only place in the unit they appear near each other.)

Read what that formula assumes and, more importantly, what it does not. It does
not assume the loss is convex. It does not assume training converged to an
optimum. It does not require a second-derivative matrix, an inverse of anything,
or a solve. It requires the training trajectory, which you have, and the
first-order step approximation, whose error you can shrink by knowing your
learning rate was small. **That is why TracIn is the honest baseline of the
gradient family: when it is wrong, it is wrong for reasons you can name and
bound.** The alternatives in this space buy a correction term at the cost of
assumptions that are known to be false for language models, and this unit's
reckoning section shows what that trade has actually bought empirically.

<!-- fade: v8-tracin-checkpoints -->

**Worked, over two checkpoints.** Three parameters, so every vector fits on a
line. Two training examples, A and B, and one query $z_q$.

Checkpoint 1, taken early, learning rate $\eta_1 = 0.10$:

| vector | value | dot with $g_q$ | $\times\, \eta_1$ |
|---|---|---|---|
| $g_q^{(1)}$ | $(2, -1, 0)$ | - | - |
| $g_A^{(1)}$ | $(1, 1, 3)$ | $2 - 1 + 0 = 1$ | $0.10$ |
| $g_B^{(1)}$ | $(-2, 0, 1)$ | $-4 + 0 + 0 = -4$ | $-0.40$ |

Checkpoint 2, taken late, learning rate $\eta_2 = 0.02$:

| vector | value | dot with $g_q$ | $\times\, \eta_2$ |
|---|---|---|---|
| $g_q^{(2)}$ | $(1, 2, -1)$ | - | - |
| $g_A^{(2)}$ | $(3, 2, 1)$ | $3 + 4 - 1 = 6$ | $0.12$ |
| $g_B^{(2)}$ | $(0, 1, 1)$ | $0 + 2 - 1 = 1$ | $0.02$ |

Add down the columns:

$$\text{TracIn}(A, z_q) = 0.10 + 0.12 = 0.22
\qquad
\text{TracIn}(B, z_q) = -0.40 + 0.02 = -0.38$$

A helped, B hurt. Three things in those numbers are worth more than the numbers.

First, **the learning rate is not decoration**. A's raw dot product at
checkpoint 2 is 6, six times its value at checkpoint 1, and yet the two
checkpoints contribute almost equally (0.10 against 0.12) because the early
learning rate is five times larger. Early steps move the weights further, so
they matter more per unit of alignment. Drop the $\eta_c$ weights and you get a
different ranking, which is one of the two mistakes that corrupts
implementations.

Second, **signs flip across checkpoints and that is normal**. B was a strong
rival early and a mild ally late. An example's relationship to a query is a
property of where the model currently is, and the model moves.

Third, **B's whole score comes from one checkpoint**. That is the general case,
not an artifact: influence is empirically sparse, concentrated in a few
sequences and a few moments, and any summary that reports a smooth distribution
over your corpus should be checked before it is believed.

```beat
id: v8-b5
type: completion
concept: d-tracin
# fade: v8-tracin-checkpoints, stage 2 of 3. The blanked cells are one dot
# product and one learning-rate weighting, which are the two operations the
# formula consists of.
# variants: blank the checkpoint-1 dot product instead of checkpoint 2, which
# tests the larger learning rate; or give the total and blank eta_2.
prompt: |
  Fill the blanks, then read the sentence that comes with them. One training
  example C, one query $z_q$, two checkpoints.

      Checkpoint 1, eta_1 = 0.05
        g_q = (1, 2, -2)
        g_C = (4, 1, 0)
        dot          = (1)(4) + (2)(1) + (-2)(0) = 6
        contribution = 0.05 x 6 = ____        <- A

      Checkpoint 2, eta_2 = 0.01
        g_q = (0, 3, 1)
        g_C = (2, -1, 5)
        dot          = ____                   <- B
        contribution = ____                   <- C

      TracIn(C, z_q) = ____                   <- D

  Which filling of A, B, C, D is correct, together with the right account of
  which checkpoint dominates the score and why?
options:
  - text: |
      A: 0.30, B: 2, C: 0.02, D: 0.32. Checkpoint 1 dominates, contributing
      0.30 of the 0.32 - not because the alignment is better in any absolute
      sense but because its learning rate is five times larger, so each unit
      of alignment moved the weights five times as far. The raw dot products
      are 6 and 2, a factor of 3; the weighted contributions are 0.30 and
      0.02, a factor of 15.
    correct: true
    explain: |
      Right, including the arithmetic on B: $(0)(2) + (3)(-1) + (1)(5) = 2$.
      The gap between the 3x raw ratio and the 15x weighted ratio is the
      learning rate doing work, which is why $\eta_c$ is not decoration.
  - text: |
      A: 6, B: 2, C: 2, D: 8. Sum the dot products across checkpoints;
      checkpoint 1 dominates because the gradients were better aligned there,
      6 against 2. The learning rate is a property of the training schedule,
      not of the alignment being measured, so it drops out of the score.
    misconception: V8-M3
    explain: |
      Dropping $\eta_c$ is one of the two standard implementation bugs, and it
      changes the ranking: early steps move the weights further, so they
      matter more per unit of alignment. The schedule is inside the number -
      change the schedule and every term rescales - which is precisely why a
      raw score is not a portable quantity. The weighted contributions are
      0.30 and 0.02, totalling 0.32.
  - text: |
      A: 0.30, B: 2, C: 0.02, D: 0.32. Checkpoint 1 dominates only because two
      checkpoints is too coarse a grid: the sum is approximating a sum over
      every step, so adding checkpoints between these two would fill in the
      trajectory and even the contributions out.
    misconception: V8-M2
    explain: |
      The four numbers are right and the account is not. Checkpoint 1's
      dominance comes from $\eta_1$ being five times $\eta_2$, and that gap
      does not close by adding checkpoints. Checkpoints are nowhere near
      independent samples - two a few hundred steps apart in a stable phase
      give nearly identical gradients, so you pay twice to add the same number
      twice. What a checkpoint buys is coverage of a distinct phase; spacing
      matters, count barely does.
  - text: |
      A: 0.30, B: 8, C: 0.08, D: 0.38. Checkpoint 2's dot product is
      $0 + 3 + 5 = 8$: the example and the query are aligned at both
      checkpoints, so both terms add. Checkpoint 1 dominates because it has
      both the larger learning rate and the larger alignment.
    misconception: V8-M1
    explain: |
      The middle term is $(3)(-1) = -3$, not $+3$, so B is
      $0 - 3 + 5 = 2$ and the total is 0.32. Discarding the negative component
      assumes an example related to a query must align with it - but alignment
      is a mechanical fact about the current parameters, and signs flip both
      within a dot product and across checkpoints. In the chapter's worked
      pair, B is a strong rival early and a mild ally late.
check: choice
```

## Making it affordable

The formula is now correct and completely impractical, for a reason you can see
in its shape: $\nabla_\theta \ell(\theta_c, z)$ has one component per parameter.
At 561 million parameters in 16-bit floats, that is

$$5.61 \times 10^8 \times 2 \text{ bytes} = 1.122 \text{ GB}$$

per example, per checkpoint. A corpus of 10 million sequences over 4 checkpoints
would be 45 petabytes of gradients. The method as stated cannot be run by
anyone, at any budget, which is why the interesting engineering is all in the
next three moves.

### Move 1: project, because dot products survive it

You never need the gradients themselves. You need their dot products. That is a
much weaker requirement, and there is a standard fact that exploits it.

Build a random matrix $P$ with $d$ rows and $N$ columns, where $N$ is the
parameter count and $d$ is a target dimension you choose - a few thousand. Fill
it with independent random numbers drawn from a Gaussian with mean 0, then scale
by $1/\sqrt{d}$. For any vector $g$ with $N$ components, $Pg$ has $d$ components.
The fact, called the **Johnson-Lindenstrauss** property, is that for any two
fixed vectors $u$ and $v$,

$$(Pu) \cdot (Pv) \approx u \cdot v$$

and the target dimension $d$ needed to hold that approximation to a given
accuracy depends on *how many* vectors you need to preserve, not on how long
they are. $N$ does not appear. Ten million gradients of 561 million components
each need the same projected dimension as ten million gradients of a billion
components each.

The intuition, which is enough to use it correctly. Each row of $P$ is one
random direction. Dot both vectors against that single direction and multiply
the results: on average, across random directions, that product equals $u \cdot v$
exactly, but any individual direction gives a wildly noisy estimate. The matrix
gives you $d$ independent such estimates and the projected dot product averages
them, so the error falls like $1/\sqrt{d}$. At $d = 4{,}096$ that is
$1/64 \approx 1.6\%$ - relative to $\lVert u\rVert \lVert v\rVert$, the product
of the two vectors' lengths.

Read that last clause carefully, because it is the honest limit of the
technique. The error is 1.6% of the *maximum possible* dot product, not 1.6% of
the actual one. Most pairs of gradients in a real model are close to
perpendicular, so their true dot products are a tiny fraction of
$\lVert u\rVert \lVert v\rVert$, and for those pairs the projection error can
exceed the signal entirely. **Projected TracIn resolves the strongly aligned
examples reliably and the long tail badly.** Since influence is empirically
sparse - a few sequences dominating - that is a survivable trade for finding
top contributors and a fatal one for claiming a full distribution over a corpus.

One implementation consequence follows immediately and it is the thing people
get wrong: the approximation only holds when both vectors go through the *same*
$P$. A different draw of the matrix is a different map into a different space,
and dot products across two different projections mean nothing at all. You do
not store the matrix - at $d \times N$ it is larger than the model. You store
the seed and regenerate.

### Move 2: accumulate per source, because that is the payment unit

A royalty system pays sources, not sequences. The dot product is linear in each
argument, which means the sum of dot products is the dot product of the sums:

$$\sum_{z \in \text{source } i} (Pg_{z_q}) \cdot (Pg_z)
= (Pg_{z_q}) \cdot \Big(\sum_{z \in \text{source } i} Pg_z\Big)$$

So you never store per-sequence vectors at all. Keep one running $d$-component
accumulator per source per checkpoint, add each sequence's projected gradient
into its source's accumulator as you stream through the corpus, and discard the
sequence vector. Storage collapses from one vector per sequence to one vector
per source: from ten million to twelve.

```beat
id: v8-b6
type: compute
concept: d-tracin
prompt: |
  A corpus of $10^7$ training sequences, tagged into $K = 12$ sources.
  Gradients are projected to $d = 4{,}096$ components and stored in 16-bit
  floats (2 bytes each). You save 4 checkpoints.

  Step 1. Bytes for one projected gradient: $4{,}096 \times 2$.
  Step 2. Per-sequence storage at ONE checkpoint: multiply by $10^7$. Convert
          to GB, using 1 GB $= 10^9$ bytes.
  Step 3. Per-source storage across ALL FOUR checkpoints: one vector per
          source per checkpoint. Convert to KB, using 1 KB $= 10^3$ bytes.

  Answer as two numbers, comma-separated: the step-2 figure in GB to one
  decimal place, then the step-3 figure in KB to the nearest whole number.
  Format like `12.3, 456`.
answer: "81.9, 393"
check: exact
```

Eighty-two gigabytes against three hundred and ninety-three kilobytes. The
per-source collapse is a factor of 833,000 and it costs you exactly one thing:
you can no longer ask which *sequence* was influential, only which *source*. For
a royalty ledger that is not a loss, because the source was always the unit that
gets paid. For a debugging tool it is, and you would keep per-sequence vectors
for a sampled subset if you wanted both.

Stack the two moves and the arithmetic inverts. Building the index is one
forward-and-backward pass over the corpus per checkpoint - roughly the cost of a
few extra training epochs, on the order of $50 on top of the $92 run as of
mid-2026. After that, answering a query is one backward pass on one sequence,
one projection, and twelve dot products in 4,096 dimensions. Milliseconds.
Against $3,588 and six days.

### Move 3: choose checkpoints deliberately

<!-- refutes: V8-M2 -->

**You probably think more checkpoints give a better estimate**, since the sum
over checkpoints is approximating a sum over steps and a finer grid is a better
approximation. Under that belief, checkpoint count is a
cost-versus-accuracy dial and the only reason to use few is that you are cheap.

**Here is the prediction that fails.** If checkpoints were independent samples
of the trajectory, doubling them would cut the estimator's error the way
averaging independent measurements does. They are nothing like independent. Two
checkpoints five hundred steps apart in a stable phase have nearly identical
parameters, so they produce nearly identical gradients and nearly identical dot
products. You paid twice the storage and twice the gradient compute to add the
same number to the sum twice. Meanwhile a checkpoint from the first few hundred
steps - where the model is barely better than random - contributes large
gradients whose alignments carry almost no information about the trained model,
adding variance rather than signal.

**Here is why the wrong model is appealing.** It is the correct instinct for
numerical integration, where a finer grid genuinely converges, and the formula
does look like a Riemann sum. The original TracIn work also reports results with
only a handful of checkpoints, which reads like a budget compromise rather than
a design choice.

**Here is what is actually true.** What you are buying with a checkpoint is
coverage of a *distinct phase* of training, because the relationship between an
example and a query changes as the model moves - that is exactly the sign flip
in the worked example above. Spacing matters, count barely does. Choose
checkpoints that span the trajectory after the early chaos, and be aware that
both storage and gradient compute are linear in the count.

There is a scheduling consequence, and it is the reason the training recipe was
chosen the way it was two units ago. Under a cosine decay schedule the learning
rate is different at every checkpoint and every checkpoint sits partway through
a decay, so the $\eta_c$ weights are all different and the checkpoints are not
comparable states. Under a warmup-stable-decay schedule the long middle phase
runs at constant learning rate, so every checkpoint in it carries the *same*
$\eta_c$, is a legitimate branch point, and differs from its neighbours only by
how far training has progressed. Checkpoint spacing becomes a clean design
variable rather than a confound. The recipe that made leave-one-out branching
affordable is the same recipe that makes TracIn's checkpoint sum interpretable.

## The reckoning

<!-- refutes: D6 -->

Everything up to here works. The formula is right, the projection is sound, the
storage fits on a laptop, and the per-query latency is milliseconds. This is the
point where a demo gets built and a claim gets made, so this is the point to
find out what the claim is worth.

**You probably think gradient-based attribution is basically solved and that
scaling it is engineering.** The prior unit stated this and named the central
dilemma; here is the full evidence, because you are now in a position to
implement the thing and the temptation is proportional.

**Here is the prediction that fails.** If cheap gradient methods computed
approximately what retraining computes, then measuring them against retraining
would show approximate agreement. That measurement exists. On a
2-billion-parameter model with actual retraining as ground truth, a near-optimal
method reaches rank correlation $\rho = 0.97$ with the truth - and the methods
that scale come out **no better than random guessing**. Not degraded. Random. And
the method that reaches 0.97 costs three to five full training runs *per test
sample*, which makes it an oracle for a handful of points, not a system.

The second result is worse, because it is about the honest baseline you just
learned. In 2026 a group found that the dominant error in trajectory-based
methods was an assumption sitting in plain sight in every derivation, including
the one above: **the literature assumes SGD and the models were trained with
AdamW.** Correcting that single mismatch moved published results by 10 to 300%.

You can see exactly where it enters. The derivation said: the parameters move by
$\Delta\theta$, so the query's loss changes by
$\nabla_\theta \ell(z_q) \cdot \Delta\theta$. That part is just the first-order
expansion and it is fine. Then it substituted $\Delta\theta = -\eta\, g_z$,
which is the *SGD* update rule - step directly along the negative gradient.
AdamW does not do that. It divides each coordinate of the gradient by a running
estimate of that coordinate's typical magnitude, so its update is

$$\Delta\theta_j = -\eta \, \frac{g_{z,j}}{\sqrt{v_j} + \epsilon}$$

for coordinate $j$, where $v_j$ is the optimizer's running second-moment
estimate for that coordinate and $\epsilon$ is a small constant preventing
division by zero. Put that back into the expansion and the honest summand is not
a dot product at all, it is a **weighted** dot product:

$$\sum_j \frac{g_{z_q,j}\; g_{z,j}}{\sqrt{v_j} + \epsilon}$$

Across a transformer's parameters, $v_j$ spans orders of magnitude. Weighting by
its reciprocal is not a constant rescale that cancels out of a ranking - it
systematically promotes examples whose gradient mass sits in low-variance
coordinates and demotes the others. **The ranking changes.** That is the whole
mechanism behind a headline that otherwise sounds like magic, and it is
actionable: save the optimizer's second-moment state alongside each checkpoint,
and use the weighted form. It is four extra lines and it is the difference
between implementing TracIn and implementing TracIn's derivation for an
optimizer nobody used.

**Here is why the wrong model is appealing.** Every negative result above is
published by the same community that publishes the methods, and none of it makes
headlines. A demonstration that a method *runs* at 52 billion parameters looks
exactly like a demonstration that it *works* there. And the engineer's prior is
reasonable: in most of computing, a technique that is correct at small scale and
runs at large scale is correct at large scale. Attribution is one of the places
that prior fails, because correctness here is defined by an experiment nobody
runs at large scale.

**Here is what is actually true, stated as an instruction.** Run your TracIn
implementation against the leave-one-source-out table you already built. Rank
the twelve sources by TracIn score, rank them by measured contribution, and
compute the rank correlation. Then score it properly the way the field does -
predict the outcomes of held-out subset runs from the per-source scores and
rank-correlate those, which is the Linear Datamodeling Score. **Those numbers are
your trust budget, and until you have measured them you do not have one.**
Everything you say about attribution afterwards is licensed by that figure and
nothing else.

### The Hessian-corrected family, in one paragraph

Influence functions are the other half of this literature and they deserve an
honest sentence rather than a dismissal. The formula is

$$\mathcal{I}(z, z_q) = -\,\nabla_\theta \ell(\theta^*, z_q)^{\top} H^{-1} \nabla_\theta \ell(\theta^*, z)$$

where $\theta^*$ is the trained parameters and $H$ is the Hessian - the matrix of
second derivatives of the training loss, one entry per pair of parameters. The
$H^{-1}$ is a curvature correction with real content: a direction in which the
loss surface is flat lets the optimum slide a long way when you remove an
example, while a steep direction barely moves, and the plain dot product cannot
tell those apart. In a convex problem solved to optimality it is the right
answer to leave-one-out. The derivation requires a twice-differentiable,
strictly convex loss evaluated at an exact optimum, and language model training
satisfies none of the three. The empirical record follows: a student of the
author of the largest scaling demonstration showed the quantity being estimated
is not leave-one-out at all but a different object (the proximal Bregman response
function); three other groups report the method as "often erroneous" for deep
networks and "consistently poor across most LLM settings"; and the 52B
demonstration was published as a study of generalization, not as an attribution
product. So the family with the deepest mathematics has the weakest empirical
support, which is the single most useful thing to know about this field. Unit x1
does the derivation, the approximations that make $H^{-1}$ tractable, and the
critiques properly.

<!-- refutes: V8-M3 -->

One more trap before the beat, because it is the one that shows up in a deck.
**You probably think an influence score is a quantity - that 0.42 means
something you can compare to another 0.42.** It is not. A TracIn score is a sum
of learning-rate-weighted dot products of gradients in one model's parameter
space. Change the model and it is a different space of a different dimension.
Change the learning-rate schedule and every term rescales. Change the projection
seed and the number moves within the same model. There are no units and no
shared zero. What is comparable is the *ranking* within one model for one query -
and even across queries the ranking needs care, because a query whose gradient
happens to be large makes every source's score large. If you want cross-query
comparability, normalize by the query gradient's length, which is what the
production-scale systems in this literature do under various names. Report
rankings and correlations, never raw scores, and never a raw score from one
model beside a raw score from another.

```beat
id: v8-b7
type: self-explain
concept: d-tracin
prompt: |
  Two measurements of the same TracIn implementation, both against
  leave-one-source-out ground truth built the same way.

  **Corpus A.** 12 sources. All 12 contributions cleared the noise floor
  ($|z| > 2$). TracIn's rank correlation against the truth table:
  $\rho = 0.71$.

  **Corpus B.** 12 sources. Nine of the 12 contributions fell below the noise
  floor. TracIn's rank correlation against the truth table: $\rho = 0.11$.

  Which explanation of these two numbers would you put your name to?
options:
  - text: |
      Corpus A's 0.71 is evidence about the method: every entry in its truth
      table is a resolved measurement, so the ordering TracIn is graded
      against is a real ordering. Corpus B's 0.11 is not evidence about the
      method at all - nine of the twelve target entries sit below the noise
      floor, so their relative order was set by the training seed, and a
      perfect attribution method would also score near zero against a
      permutation of noise. To get a usable number on B you fix the
      instrument, not the estimator: more seeds per condition (the standard
      error falls as $1/\sqrt{n}$), a larger model or token budget, or grading
      against held-out subset outcomes, whose differences are large compared
      with seed noise.
    correct: true
    explain: |
      Right, and the last move is the one that costs discipline: the
      correlation can only be as good as the ground truth it is measured
      against, so an unresolved truth table is a finding about the corpus
      rather than a verdict on the method.
  - text: |
      Both numbers are facts about TracIn: it tracks ground truth on corpus A
      and fails on corpus B. Ship it for A-like corpora and use something else
      on B. The nine sources below the floor contributed essentially nothing,
      which is exactly why TracIn cannot rank them - there is nothing there to
      rank.
    misconception: D16
    explain: |
      Below the noise floor means the measurement did not resolve, not that
      the contribution was zero. Retrain with a different seed and a source's
      marginal contribution can swing by more than its own magnitude, so the
      order of those nine entries is seed noise. Correlating against noise
      gives near zero for any method, good or bad, so 0.11 says nothing about
      TracIn - and acting on it would mean discarding a method for no reason.
  - text: |
      0.11 says the implementation is under-resolved on corpus B. Save more
      checkpoints so the trajectory sum is finer, raise the projection
      dimension $d$ so the dot products are less noisy, and the correlation
      will come up toward corpus A's.
    misconception: V8-M2
    explain: |
      Both fixes are aimed at the estimator while the broken thing is the
      instrument: nine of the twelve target ranks are noise, and no estimator
      correlates with a permutation. The checkpoint move would not help even
      on a sound target - checkpoints a few hundred steps apart in a stable
      phase give nearly identical dot products, so spacing matters and count
      barely does.
  - text: |
      The spread between 0.71 and 0.11 is a methods problem. Gradient
      attribution is young; a better estimator - preconditioned, more
      checkpoints, a smarter projection - will lift corpus B toward corpus A,
      and until then report the 0.71 as the method's correlation since it is
      the measurement taken under favourable conditions.
    misconception: D20
    explain: |
      Two errors. The missing signal on corpus B is in the ground truth, not
      in the estimator, so no refinement reaches it. And when an estimator's
      signal-to-noise sits below the proven threshold, the welfare-optimal
      contract collapses to a flat fee - no better estimator changes that,
      only more signal does. Reporting the favourable number alone is
      reporting a scope you did not measure.
check: choice
```

## Influence is not entailment

<!-- refutes: D5 -->

**You probably think there is one question here.** "Which training data is
responsible for this output" sounds like a single question with a single right
answer, and you expect the good methods to converge on it while the bad ones
miss. Under that belief, a lexical search baseline is what you compare against
to show your gradient method is better.

**Here is the prediction that fails.** A large-scale run of a
production-grade gradient influence method - 8 billion parameters, a full 160
billion token corpus, done properly by people with the compute - was compared
against **BM25**, a lexical retrieval scoring function from the 1990s that
counts term overlap with rarity weighting and has no idea a model exists. On the
task of finding the document that contains the fact the model stated, BM25
**wins**. Not narrowly, and not as an artifact of a weak baseline. The same
finding recurs across the benchmark suites: cheap lexical and embedding
baselines match or beat expensive gradient attribution, and the earlier
benchmarks that said otherwise turned out to be lexically biased in the
gradient methods' favour.

**Here is why the wrong model is appealing.** Both methods return a ranked list
of documents in response to an output, so their outputs are type-identical and
invite comparison. The word "responsible" does real damage here too, because it
covers both meanings in English and neither the papers nor the products
distinguish them consistently.

**Here is what is actually true.** There are two different targets and the
methods are each good at one.

**Influence** is the causal question: which training data moved the loss on this
output. That is what a gradient dot product measures, it is what leave-one-out
defines, and it is the quantity a payment for *contribution* would have to be
based on. It is also the quantity that will hand you documents bearing no
lexical resemblance to the output whatsoever - the ones that taught the model
the format, the register, the reasoning step - and will hand you nothing for a
document that states the fact in a phrasing the model had already mastered.

**Entailment** is the evidentiary question: which document contains or supports
this claim. That is what BM25, suffix arrays and embedding search measure. It is
cheap, it is fast, and it produces a screenshot a lawyer can read: here is the
passage, here is the output, compare them.

The two rankings systematically disagree, and neither is wrong. They are
answers to different questions, and this is the same split the book opened with
under different names - influence is contributive, entailment is corroborative.

**A royalty design has to pick one and defend it.** Pay for entailment and you
are paying for lexical resemblance, which means you systematically underpay the
sources that shaped the model without resembling its outputs - the "hidden
influencers" the music-royalty literature identified in exactly this form - and
you have built a corroborative system while claiming a contributive one. Pay for
influence and you are paying a quantity that is measurable, defensible, and
completely unintuitive to the recipient, who will look at the document you paid
them for and see no relationship to the output at all. There is no third option
where one number does both jobs. The honest product shows both, labels which is
which, and lets the contrast be the point.

```beat
id: v8-b8
type: predict
concept: d-influence-vs-entailment
prompt: |
  Before reading on, commit to a prediction.

  Your demo answers a question and displays two ranked lists side by side: the
  top three sources by projected TracIn score, and the top three passages by
  suffix-array verbatim match. A rightsholder from source 7 is watching.

  Which account gets all three right - how the lists disagree, what you can
  honestly tell the rightsholder when source 7 tops the TracIn list and is
  absent lexically, and what you can tell them in the reverse case?
options:
  - text: |
      The lists will frequently be disjoint: the lexical list surfaces sources
      that share wording with the output, the TracIn list surfaces sources
      whose gradients moved the loss on it. Top by TracIn, absent lexically:
      steps taken on source 7's data are estimated to have lowered the loss on
      this output - a claim about what moved the model, worth exactly the rank
      correlation the implementation has measured against retraining ground
      truth. It does not mean source 7 contains the fact or that the text
      derives from it. Top lexically, absent from TracIn: source 7 contains
      text closely matching the output, an exact overlap fact needing no model
      and no estimator. It does not mean source 7 caused the output - the same
      passage may sit in four other sources, and correspondence is not
      causation.
    correct: true
    explain: |
      Right, and the contrast is the product: influence is contributive,
      entailment is corroborative, the rankings systematically disagree, and
      neither is wrong. The honest demo shows both, labels which is which, and
      attaches the measured correlation to the influence column.
  - text: |
      The lists should largely agree, since both answer "which training data
      is responsible for this output". Where they disagree the lexical list is
      the reliable one, and it tells the rightsholder their document is where
      the answer came from. A source at the top of the TracIn list but absent
      lexically is a false positive of the estimator, and should be suppressed
      before anyone outside sees it.
    misconception: D5
    explain: |
      Two different targets, not one question with a reliable and an
      unreliable answer. Influence is what moved the loss; entailment is what
      contains the claim. Run properly at 8 billion parameters over a 160
      billion token corpus, BM25 - a term-overlap function from the 1990s with
      no idea a model exists - beats gradient influence at finding the
      document containing the fact, precisely because they answer different
      questions. Suppressing the disagreement deletes the only informative
      part of the display.
  - text: |
      The TracIn list is the lexical list computed properly, since gradient
      alignment is semantic similarity measured in the model's own
      representation, so expect the two to differ only in ordering. Source 7
      at the top by TracIn and absent lexically means its wording is a
      paraphrase the string matcher missed, and you can tell the rightsholder
      the model was reading their text.
    misconception: V8-M1
    explain: |
      Gradient alignment is a claim about the parameters, not about meaning.
      At 52-billion-parameter scale, flipping the word order of a key phrase
      collapses influence to near zero with the content untouched, and
      documents on unrelated subjects score as strong allies when both push
      the same numeral format or citation convention. A paraphrase is exactly
      the case where the two lists come apart, in either direction.
  - text: |
      Absence from the lexical list settles it. If no passage in source 7
      matches the output, then source 7's data did not shape this output and
      the TracIn ranking is picking up style rather than use; tell the
      rightsholder there is nothing here to claim. When source 7 does top the
      lexical list, that match is the attribution.
    misconception: D3
    explain: |
      Verbatim match is a lower bound and a measurement artifact. Most of what
      training data contributes is semantic and never resurfaces verbatim -
      the documents that taught format, register and reasoning step share no
      n-grams with the outputs they shaped - which is exactly why attribution
      is hard. And a match in the other direction is corroborative: the same
      passage may exist in several sources, so it establishes correspondence,
      not causation.
check: choice
```

## The ledger inside the run

TracIn asks its question after training, about one query at a time. There is a
different question with the same machinery: rather than explaining one output,
compute every source's total contribution *while training happens*, and finish
the run holding a completed royalty ledger.

That is In-Run Data Shapley, and it is the most direct precedent for the thing
this book is building - the only Shapley-style attribution anyone has applied to
foundation-model pretraining. Derive it, because the derivation is short and it
tells you precisely what the method is worth.

**Set up the game at one step.** At training step $t$ the optimizer takes a step
using a batch $B_t$ of examples. There is a validation set, and a validation
loss $L_{\text{val}}$ over it - the same kind of held-out measurement the
counterfactual table used to define "worse". The value produced at this step is
the amount that step reduced the validation loss. The players are the examples
in the batch. The question is how to split that reduction among them.

**Compute the value.** The step moves the parameters by
$\Delta\theta_t = -\eta_t \sum_{z \in B_t} g_z$, where $g_z$ is example $z$'s
gradient at the current parameters and $\eta_t$ is the learning rate. To first
order, the validation loss changes by
$g_{\text{val}} \cdot \Delta\theta_t$, where $g_{\text{val}}$ is the gradient of
the validation loss at the current parameters - one vector, computed once per
step for the whole validation set. Substituting and flipping sign so that
positive means helpful, the reduction in validation loss at this step is

$$\eta_t \sum_{z \in B_t} g_{\text{val}} \cdot g_z$$

**Now split it.** The total is already a sum with exactly one term per player,
and no term mentions any other player. That is an *additive* game, and the
Shapley value of an additive game is trivial: a player's average marginal
contribution over arrival orders is the same in every order, because nothing it
does depends on who arrived first. So each example's Shapley value at this step
is exactly its own term:

$$\phi_z^{(t)} = \eta_t \; g_{\text{val}} \cdot g_z$$

Sum over every step the example participated in, and you have its in-run value
for the whole training run.

Stop and look at what just happened, because it is the honest core of this
section. **That is TracIn's summand.** Identical: a learning-rate-weighted dot
product between a training example's gradient and a target gradient. The only
differences are what the target is and when you compute it. TracIn dots against
one query's gradient at a few saved checkpoints, after training. In-Run dots
against the validation gradient at every step, during training. Same operation,
same first-order approximation, same blindness.

The second-order version is where the Shapley machinery earns its name. Expand
the loss change to second order and cross terms appear - pairs of examples in
the same batch whose joint effect is not the sum of their individual effects.
Those interaction terms genuinely have to be allocated, and Shapley's answer is
to split each pairwise term evenly between the two examples involved. That is
the first place the axioms do any work.

**What it costs.** Computing a separate gradient for every example in a batch
would defeat the point. The paper's contribution is a technique for extracting
all the pairwise gradient inner products from within a single backward pass,
without ever materialising per-example gradients, at about **7.5% overhead** for
the first-order version. On a $92 run that is $6.90. The demonstrated scale is
GPT-2-small and a 410-million-parameter model on roughly 10 billion tokens of
web corpus, which is the same order as the artifact this book is building. As of
the survey behind this unit, no public implementation had been found, so budget
for writing it rather than installing it.

**The royalty ledger, mechanically.** Keep one running scalar per source. At
each step, for each example in the batch, add $\eta_t\, g_{\text{val}} \cdot g_z$
to that example's source's total. Training ends and you have $K$ numbers whose
sum is the total validation-loss reduction the whole run achieved. Divide each
by the sum, clip negatives to zero, and you have a percentage split - computed
as a byproduct of a training run you were doing anyway, for 7.5%.

```beat
id: v8-b9
type: compute
concept: d-inrun-shapley
prompt: |
  One training step. Learning rate $\eta_t = 0.05$. The validation gradient at
  the current parameters is $g_{\text{val}} = (1, -2, 1)$.

  The batch has four examples, tagged by source:

  | example | source | gradient |
  |---|---|---|
  | $z_1$ | A | $(2, 0, 1)$ |
  | $z_2$ | A | $(0, 1, 0)$ |
  | $z_3$ | B | $(1, -1, 3)$ |
  | $z_4$ | B | $(-1, 0, 2)$ |

  Each example's first-order in-run contribution is
  $\eta_t \, (g_{\text{val}} \cdot g_z)$, positive meaning it reduced the
  validation loss.

  Compute what this step adds to source A's running total and to source B's.

  Answer as two numbers, comma-separated, source A first. Format like
  `0.10, 0.20`.
answer: "0.05, 0.35"
check: exact
```

Source A netted 0.05 from two examples pulling in opposite directions. Source B
netted 0.35. Repeat for every step of a hundred-thousand-step run and the ledger
writes itself.

### What the Shapley label does and does not buy

<!-- refutes: V8-M4 -->

**You probably think that because this is called Shapley, it inherits the
fairness argument from the coalition units - that the in-run ledger is the
retraining sweep's answer, computed cheaply.** That belief is worth money, which
is why it needs killing before you put it on a slide.

**Here is the prediction that fails.** If the in-run ledger computed the same
values as the coalition sweep, then the two would agree on the case the
coalition sweep exists to handle: two sources holding near-identical content.
The sweep gets this right by construction - remove either alone and the model
barely notices, remove both and the loss jumps, and the axioms split the shared
value between them. The in-run ledger cannot see it at all. Each duplicate's
gradient is dotted against the validation gradient independently, at steps that
mostly do not even contain the other, and each is credited in full for material
that only one of them needed to supply. The additive game the derivation assumed
is exactly the assumption that redundancy violates, and redundancy is the
central fact the fair-split unit was built around.

**Here is why the wrong model is appealing.** The name is the same, the paper is
a Shapley paper, the axioms really do apply to the game as constructed, and the
derivation above is correct. Nothing was misrepresented. The subtlety is entirely
in *which game* - the players are examples within one batch at one step, and the
value is that step's validation loss reduction along a trajectory that already
happened. The coalition sweep's players are sources, and its value is the quality
of a model that was actually trained without them.

**Here is what is actually true.** These are two different games with two
different value functions, and the axioms hold separately in each. The in-run
value is a Shapley value of a *per-step, first-order, single-trajectory* game.
It cannot answer "what if this source had never been in the corpus," because
that question is about a different trajectory and nothing computed along this one
can observe it. Say precisely that, and the ledger is a strong result: a
per-source allocation, defensible within its own game, computed for 7.5%
overhead, on a run at a scale nobody else has instrumented. Say it is the
counterfactual and the first person across the table who has read the coalition
literature will ask about duplicate sources, and you will have no answer.

What redeems it is the same thing that redeems every method in this unit: the
in-run ledger and the coalition sweep produce two rankings over the same twelve
sources, so you can rank-correlate them. That number - not the axioms - is what
says whether the cheap ledger is standing in for the expensive truth on your
corpus.

## What to trust

Four questions, four instruments, four different warrants. This table is the
unit compressed.

| the question | the instrument | the cost | what licenses the answer |
|---|---|---|---|
| What did source $i$ contribute to this model? | leave-one-source-out plus exact Shapley over cached coalitions | thousands of runs, cached, hundreds of dollars | exact within its stated scope; this is the calibration standard, not a candidate |
| Why did the model say *this*? | projected per-source TracIn over saved checkpoints | tens of dollars once, then milliseconds per query | its measured rank correlation against the table above, and nothing else |
| What does the royalty split say? | In-Run Shapley accumulated during training | +7.5% of one run | the same measured correlation, plus an explicit statement that the game is per-step and additive |
| Where did this exact sentence come from? | suffix-array or BM25 lexical match | milliseconds, no GPU | verbatim overlap only; exact, corroborative, and silent about causation |

Three rules for using it.

**Never report a cheap number without its correlation.** "Source 7 drove this
output" is unfalsifiable. "Source 7 ranks first by projected per-source TracIn,
whose rank correlation against leave-one-source-out ground truth on this corpus
is 0.71 over twelve sources" is a claim with a method, a scope, and a limit, and
a skeptic can attack it productively - which is what makes it worth more than
the confident version.

**Never let the expensive instrument out of the building.** The table is not the
prototype of the product. It is the only object that can tell you the product is
lying, and it stops working the moment you scale past where retraining is
affordable. Which means the correlation you measure at ten million parameters is
the last honest number you will ever have about your method, and you should
treat it as precious rather than as a stepping stone.

**Report which target you chose.** Influence or entailment, contributive or
corroborative. Any system that shows a lexical match and calls it attribution
has answered the easy question and labelled it with the hard question's name.

## At the bench: the gradient ledger

Here is what to build after this unit, and what it feeds.

**Build one: projected per-source TracIn.** Take the checkpoints from the
overnight training run - the warmup-stable-decay recipe means the stable-phase
checkpoints are comparable and carry the same $\eta_c$ - and pick four spanning
the trajectory after the early chaos. Fix a projection seed and write it into
the manifest. For each checkpoint, stream the corpus, compute each sequence's
per-example gradient, project to $d = 4{,}096$, and add it into its source's
accumulator. You finish with $12 \times 4$ vectors, under 400 KB. Use the
optimizer's second-moment state to form the weighted dot product rather than the
plain one, because the model was trained with AdamW and the plain form is
measuring the wrong optimizer.

**Build two: the grading.** For each of the twelve sources, produce a TracIn
score, rank them, and compute the Spearman rank correlation against the
leave-one-source-out table. Then do it properly: predict the held-out subset
outcomes from the per-source scores and rank-correlate those, which is the
Linear Datamodeling Score and is the number the field will ask for. Report both,
with the design stated - how many sources, how many held-out subsets, which
checkpoints, what projection dimension.

**Build three: In-Run Shapley.** Instrument a rerun of the small model. One
validation gradient per step, one dot product per example, one running scalar per
source, and a measured overhead figure of your own to put beside the published
7.5%. The output is a twelve-row ledger produced as a byproduct of a training
run.

**Artifact.** One table, twelve rows, three columns of numbers: the coalition
split from the fair-split unit, the projected TracIn ranking, and the in-run
ledger. Beneath it, three lines: the rank correlation of each cheap column
against the expensive one, the LDS of each with its design, and one sentence
saying which cheap column you would ship and why. Plus the header block that
makes any of it interpretable - projection seed and dimension, checkpoint list
with learning rates, optimizer-preconditioning on or off, and the validation set.

**What feeds forward.** That three-column table is the centre of the synthesis
demo: one question, three ranked answers, the verbatim lexical match beside them
for contrast, and a correlation number underneath saying how much to believe the
cheap ones. The contrast is the demo, and the correlation is the reason a
technical audience stays in the room.

```beat
id: v8-b10
type: self-explain
concept: d-inrun-shapley
prompt: |
  You are writing down, in advance, what the gradient index must record for
  its numbers to mean anything later. A colleague proposes recording the
  projection dimension, the checkpoint step numbers, and the git commit of the
  scoring code.

  That list is incomplete in one way that makes the stored index **silently
  worthless**, and incomplete in further ways that make its numbers
  uninterpretable. Which account of what is missing, and of what goes wrong in
  each case, is right?
options:
  - text: |
      The silent-corruption omission is the **projection seed**. Dot products
      survive projection only when both vectors pass through the same matrix
      $P$; $P$ is larger than the model, so it is regenerated from a seed
      rather than stored. Without the seed, a query gradient projected at
      scoring time goes through a different random map than the source
      accumulators did, and the dot products are meaningless - yet still
      finite, ordered, and productive of a plausible ranking. Also missing:
      the **learning rate at each checkpoint**, since the score is a sum of
      $\eta_c$-weighted dot products and step numbers alone do not recover the
      weights; and **whether the dot product was optimizer-preconditioned**,
      because the plain form implements the SGD update while the model trained
      with AdamW, and the two produce different rankings.
    correct: true
    explain: |
      Right, and the seed is the dangerous one because nothing errors: the
      pipeline returns confident numbers that are noise. The other two are
      what let a score be recomputed and compared at all - correcting the
      SGD-versus-AdamW mismatch moved published results by 10 to 300%. The
      validation or query set the scores were computed against belongs in the
      same manifest, for the same reason.
  - text: |
      The critical omission is the **number of checkpoints and their spacing**:
      the sum over checkpoints approximates a sum over steps, so an index
      built from too few is a coarse approximation and its scores are
      systematically wrong. Record the checkpoint count, then add the
      projection seed and the learning rates as useful extras for anyone
      wanting to reproduce the run exactly.
    misconception: V8-M2
    explain: |
      Checkpoints are not independent samples of the trajectory - two a few
      hundred steps apart in a stable phase give nearly identical gradients,
      so doubling them adds the same number twice. What a checkpoint buys is
      coverage of a distinct phase; spacing matters, count barely does. And
      the seed is not an extra: without it the stored accumulators and the
      query gradient live in different random spaces, and every number the
      index produces is noise wearing a ranking.
  - text: |
      Nothing critical is missing. With the projection dimension, the
      checkpoint steps and the code commit recorded, the scores are quantities
      that can be compared against later runs, other queries and other models;
      the projection seed only affects the exact decimals, and the learning
      rates are already implied by the schedule in the code.
    misconception: V8-M3
    explain: |
      A TracIn score is a sum of learning-rate-weighted dot products in one
      model's parameter space - no units, no shared zero. Change the seed and
      the number moves inside the same model on the same corpus for the same
      query; change the model and it is a different space of a different
      dimension. Worse, a different seed at scoring time does not perturb the
      decimals, it severs the relationship between the two vectors entirely.
  - text: |
      The omissions are the projection seed, the per-checkpoint learning rates
      and the validation set - all three worth recording as reproducibility
      hygiene. The numbers in the index are already computed and are what they
      are; the manifest just makes the run easier to repeat and easier to
      defend if anyone asks later how it was done.
    misconception: D19
    explain: |
      The items are right and their status is wrong, which is the whole point.
      A pipeline that regenerates a different projection matrix on each run is
      reproducibly wrong: the stored numbers are not "already computed and
      correct" without the seed, they are noise. These are preconditions for
      the numbers meaning anything, in the same way a threshold committed
      before seeing the model is what turns a statistic into an exhibit and
      one chosen afterwards proves nothing.
check: choice
```

## What you can now do

Four capabilities, and one sentence that is the point of the unit.

**You can compute influence from one training run.** A per-example gradient is
the direction that most reduces that example's loss. The dot product between two
examples' gradients says whether they push the model the same way, and
multiplied by the learning rate it is the first-order prediction of how much a
step on one moved the other's loss. Sum that over saved checkpoints and you have
TracIn: no Hessian, no convexity, no assumption that training converged, and an
error that shrinks with the step size.

**You can make it fit.** Random projection preserves dot products with an error
that falls like $1/\sqrt{d}$ and does not care how many parameters the model
has - so long as every vector goes through the same seeded matrix. Per-source
accumulation collapses ten million stored vectors to twelve, because linearity
lets you sum before you dot. Together they take a 45-petabyte method to under
400 kilobytes and milliseconds per query, and the two things you must get right
are the projection seed and the optimizer's preconditioning.

**You can build a royalty ledger as a training byproduct.** At first order the
per-step game is additive, so each example's Shapley value is its own
learning-rate-weighted dot product against the validation gradient - the same
summand TracIn uses, accumulated per source, during the run, for about 7.5%
overhead. And you can say exactly what that ledger is a Shapley value *of*: a
per-step, first-order, single-trajectory game, which is not the coalition game
and does not see redundancy between sources.

**And you can say what any of it is worth.** Rank your cheap method's output
against the counterfactual table, report the rank correlation and the LDS with
the design they were measured on, and attach that number to every attribution
claim you make. The published measurements say the methods that scale come out
at chance when someone finally checks, that the one method that correlates costs
three to five training runs per query, and that a lexical scoring function from
the 1990s beats gradient influence at finding the document containing the fact.
None of that means your implementation is bad. It means nobody can tell,
including you, until it is graded.

The sentence: **a gradient method is a hypothesis about the counterfactual, and
its rank correlation against a measured counterfactual is the entire evidence
that the hypothesis is true.** Everything in this unit is cheap. Only that
number is expensive, and it is the only part a skeptic cannot take from you.

The synthesis unit assembles the three-column table into the demo, next to the
verbatim lexical surface, and answers the question the whole book has been
building toward: given the correlation you measured, is contribution-proportional
payment better than a flat fee on this corpus. If you skipped ahead to a pie
chart, this is where it gets audited.

## Notation in this unit

<!-- canon-only -->

Reference, not reading. Return here when a symbol goes blurry; nothing below is
new.

| Symbol | Means | Typical value here |
| --- | --- | --- |
| $\theta$ | every model parameter, stacked into one long vector | $N$ components |
| $N$ | parameter count | $10^7$ to $5.61 \times 10^8$ |
| $D$ | training tokens in a run | $2\times10^8$ to $1.12\times10^{10}$ |
| $C$ | floating-point operations for a whole run, from $C \approx 6ND$ | $10^{16}$ to $4\times10^{19}$ |
| $z$ | one training example - a sequence, with its own loss | - |
| $z_q$ | the query point being explained: one output you want traced | - |
| $\ell(\theta, z)$ | the loss of the model at $\theta$ on the single example $z$ | 1 to 10 |
| $g_z$ | $\nabla_\theta \ell(\theta, z)$, that example's gradient; same shape as $\theta$ | $N$ components |
| $g_{\text{val}}$ | gradient of the held-out validation loss; the in-run target direction | $N$ components |
| $\eta_t$, $\eta_c$ | learning rate at step $t$ or checkpoint $c$ | $10^{-4}$ to $10^{-3}$ |
| $\mathcal{C}$, $c$ | the set of saved checkpoints, and one checkpoint in it | 3 to 10 checkpoints |
| $\theta_c$ | the parameter vector saved at checkpoint $c$ | - |
| $P$ | the random projection matrix, $d$ rows by $N$ columns, regenerated from a seed | never stored |
| $d$ | projected dimension | 2,048 to 8,192 |
| $K$ | number of tagged sources | 12 |
| $v_j$ | AdamW's running second-moment estimate for parameter $j$ | spans orders of magnitude |
| $\phi_z^{(t)}$ | example $z$'s first-order in-run value at step $t$ | - |
| $\rho$ | Spearman rank correlation against ground truth | $-1$ to $1$ |
| $H$ | Hessian of the training loss, $N \times N$; appears only in the influence-function aside | - |

The formulas, with every symbol as defined above:

$$C \approx 6ND
\qquad
g_z = \nabla_\theta \ell(\theta, z)
\qquad
\Delta \ell(z_q) \approx -\eta\,\big(g_{z_q} \cdot g_z\big)$$

$$\text{TracIn}(z, z_q) = \sum_{c \in \mathcal{C}} \eta_c \, \big(g_{z_q}^{(c)} \cdot g_z^{(c)}\big)
\qquad
\text{TracIn}(\text{source } i, z_q) = \sum_{c \in \mathcal{C}} \eta_c \, \Big(Pg_{z_q}^{(c)} \cdot \sum_{z \in i} Pg_z^{(c)}\Big)$$

$$\text{AdamW-corrected summand} = \sum_j \frac{g_{z_q,j}\, g_{z,j}}{\sqrt{v_j} + \epsilon}
\qquad
\phi_z^{(t)} = \eta_t\,\big(g_{\text{val}} \cdot g_z\big)$$

Sign convention throughout, matching the counterfactual table: **positive means
helped.** A positive dot product means a step on the training example lowered
the query's loss.

Two deliberate simplifications, flagged so they do not read as contradictions
later. First, the derivation treats one step as taken on one example, while a
real step is taken on a batch; the first-order expansion is linear, so the batch
version is the sum of the per-example terms and nothing changes, but the
second-order interaction terms this glosses over are precisely what the
second-order in-run method allocates. Second, everything here uses the plain
first-order form for clarity while the reckoning section establishes that the
AdamW-preconditioned form is the correct one to implement. Read the plain form
as the derivation and the preconditioned form as the code.
