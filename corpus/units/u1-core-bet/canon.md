---
unit: u1
title: "The core bet: next-token prediction"
concepts:
  - c-lm-objective
  - c-tokens
  - c-embeddings
  - c-logits
assumes: []
---

One idea runs this whole book: a model trained to do nothing but predict the
next token becomes, at sufficient scale, the systems you use every day. This
unit states that bet precisely and walks it end to end - what a token is, how
text enters the model, how scores come out, and what the training objective
does and does not promise. By the end you can trace `The cat sat on` from
characters to a scored vocabulary, and say exactly which claims about
"understanding" that loop licenses.


## Why prediction forces world modeling

The bet is this: *if you make a system good enough at predicting the next symbol
in human-generated text, you get world modeling for free, because there is no
other way to be good at it.*

That claim deserves scrutiny rather than assent. Here is the argument in its
strong form, and then the honest statement of where it stops.

Start from behavior, not machinery. Four fragments of ordinary text, each
cut off one word early:

- `The 44th president of the United States was Barack ____`
- `>>> sorted([5, 2, 9, 1])\n[1, 2, 5, ____`
- `Therefore, since triangle ABC is isosceles with AB = AC, angle ABC = angle ____`
- `Alice put the keys in the drawer and left. Bob moved them to the safe.
  When Alice returned, she looked for the keys in the ____`

```beat
id: u1-b9
type: predict
concept: c-lm-objective
prompt: |
  Fill each blank above yourself - all four are easy for you. Then, before
  reading on: for each one, state in a phrase what a system would have to be
  able to DO to fill it reliably. Are the four demands the same kind of
  thing, or four different kinds?
answer: |
  Barack Obama; 9; angle ACB (base angles of an isosceles triangle); the
  drawer (Alice never saw the move).

  The demands: recall a stored fact about the world; execute a sorting
  algorithm; apply a geometric theorem; track two agents' divergent beliefs
  about the same object. Four different kinds of capability - a fact store,
  an algorithm, deductive rule application, and a theory of other minds -
  hiding behind one uniform interface: predict the next token.
rubric: |
  Pass requires: (1) at least three blanks filled correctly, and (2) an
  explicit recognition that the blanks demand different KINDS of capability
  (fact recall vs computation vs deduction vs belief tracking), in any
  wording. The exact taxonomy does not matter.
  Fail if the answer says all four are "just pattern completion" with no
  differentiation - that is the misconception the section dismantles, so
  deliver the section slowly rather than skipping.
check: llm
```

To fill those blanks a system must, respectively: hold a fact about the
world, execute a sorting algorithm, apply a geometric theorem, and track two
agents' divergent beliefs about where an object is. Nothing about the
interface distinguishes the four cases. The same next-token question is
silently asking for lookup, computation, deduction, and a theory of mind -
and human text is full of all four. Whatever gets good at this game must
carry some working version of each capability. The rest of this section makes
that argument precise, and then states honestly where it stops.

The argument is about compression. Shannon's source coding theorem says that if
you have a probability model $Q$ over symbols, you can encode a symbol $x$ in
$-\log_2 Q(x)$ bits, and you cannot do better on average than the true entropy.
This is not a metaphor. Arithmetic coding achieves it in practice. And the
number training pushes down - the loss, written $\mathcal{L}$: the average
over the corpus of $-\ln$ of the probability the model gave the symbol that
actually came next; the objective section below states it exactly - is this
same quantity in natural-log units. Converted to base 2, the loss *is* the
compressed size of the training corpus in bits per token under the model's
code:

$$\text{bits per token} = \frac{\mathcal{L}}{\ln 2}$$

A language model is therefore a compressor, and lowering its loss is literally
compressing the corpus harder. That reframing does the work, because compression
has a known lower bound tied to structure: the only way to encode a string in
fewer bits than its length is to exploit regularity in the process that
generated it.

A frequency table over 5-token windows spends many bits on every one of the
four blanks above, and those bits show up in the loss. The corpus is full of
such continuations, and the gradient pushes on every one of them.

That is the argument, and it is strong. Here is where it stops, stated plainly
because the overclaimed version of this argument is everywhere.

The compression argument establishes a statement about the *limit*: a system with
loss arbitrarily close to the entropy of language must model whatever generated
that language. It does *not* establish that a transformer trained with SGD at
achievable loss actually acquires those mechanisms rather than a large bag of
shallow heuristics that happen to cover most of the distribution. Those are
different claims. The first is a theorem about codes. The second is an empirical
question, and the honest answer is "considerably more than the skeptics
predicted in 2019, considerably less than the marketing says." The failure modes
you have seen with LLM tooling - confident wrong arithmetic, brittle multi-step
reasoning that collapses when you rename the variables - are exactly the residue
of heuristics standing in for mechanisms.

Hold both halves. The objective *pressures* toward world modeling, monotonically
and without limit. What you get at any particular scale is an empirical fact
about that scale, not a guarantee.

```beat
id: u1-b2
type: self-explain
concept: c-lm-objective
prompt: |
  In your own words, and without re-reading: explain why "a model that predicts
  text well must model the world that generated the text" is an argument about
  a limit, and name one concrete observation about current LLM behavior that
  shows the limit has not been reached.
answer: |
  The compression framing says loss equals bits per token, and you can only
  spend few bits on a continuation by exploiting real structure in the
  generating process. Taken to its endpoint - loss at the entropy floor - the
  system must have captured whatever structure produces the text, including
  facts, algorithms, and agents' mental states. That is a statement about what
  a hypothetically optimal predictor must contain.

  It does not say that a specific architecture trained with a specific
  optimizer for a specific budget reaches that endpoint. A shallow heuristic
  that covers 95% of a pattern lowers loss substantially while implementing
  nothing like the real mechanism. Concrete evidence the limit is unreached:
  multi-digit arithmetic that is correct on common operands and wrong on rare
  ones, or a reasoning chain that works on the textbook phrasing and fails when
  the variable names are swapped. A system with the mechanism would not care
  about the names.
rubric: |
  Must contain: (1) the link from loss to bits/compression to "must exploit
  real structure", (2) an explicit acknowledgement that the conclusion holds
  at or near the entropy floor and not at arbitrary loss, (3) a concrete
  behavioral example of heuristic-not-mechanism (arithmetic on rare operands,
  brittleness under renaming/rephrasing, or equivalent).
  Pass = all three, in any wording.
  Partial (flag for review, do not pass) = (1) and (3) but the answer treats
  the world-modeling claim as established for current models rather than as
  a limit statement.
  Fail = the answer asserts the model "just does statistics" with no engagement
  with why compression pressures toward mechanism. That is the opposite
  overcorrection and is equally wrong.
check: llm
```

## Tokens: BPE from scratch

<!-- fade: bpe-tokenize -->

The model does not consume characters and does not consume words. It consumes
integers drawn from a fixed vocabulary of $V$ subword strings, built before
training ever starts by an algorithm called byte-pair encoding. The vocabulary is
frozen for the model's entire life.

Why not characters? A 4000-character document would be 4000 positions, and u3
will show that attention cost grows as the square of sequence length. Why not
words? The vocabulary would be unbounded, and every typo, identifier, and
non-English string would be an unknown symbol. BPE is the compromise: start from
characters, then repeatedly glue together whichever adjacent pair occurs most
often, until you have $V$ symbols. Frequent words become single symbols; rare
words decompose into pieces; nothing is ever unrepresentable.

Here is the whole algorithm on a corpus small enough to do by hand. The corpus is
four distinct words with these counts:

| word | count |
|---|---|
| `low` | 5 |
| `lower` | 2 |
| `newest` | 6 |
| `widest` | 3 |

Split every word into characters, with `_` marking end-of-word (so the algorithm
can tell a word-final `t` from a word-internal one):

```
l o w _          (5)
l o w e r _      (2)
n e w e s t _    (6)
w i d e s t _    (3)
```

**Merge step.** Count every adjacent pair across the corpus, weighted by word
count. Merge the most frequent pair into a single symbol. Repeat.

*Round 1.* Pair counts, every adjacent pair in the corpus: `(l,o)`=7,
`(o,w)`=7, `(w,_)`=5, `(w,e)`=2+6=8, `(e,r)`=2, `(r,_)`=2, `(n,e)`=6,
`(e,w)`=6, `(e,s)`=6+3=9, `(s,t)`=6+3=9, `(t,_)`=6+3=9, `(w,i)`=3, `(i,d)`=3,
`(d,e)`=3. Three pairs tie at 9; break ties by first occurrence and take
`(e,s)`.

Merge 1: **`es`**

```
l o w _          (5)
l o w e r _      (2)
n e w es t _     (6)
w i d es t _     (3)
```

*Round 2.* `(es,t)`=9 and `(t,_)`=9 tie; take `(es,t)`.

Merge 2: **`est`**

```
n e w est _      (6)
w i d est _      (3)
```

*Round 3.* `(est,_)`=9 is now the unique maximum.

Merge 3: **`est_`**

```
n e w est_       (6)
w i d est_       (3)
```

*Round 4.* Remaining maxima: `(l,o)`=7 and `(o,w)`=7. Take `(l,o)`.

Merge 4: **`lo`**

*Round 5.* `(lo,w)`=7.

Merge 5: **`low`**

```
low _            (5)
low e r _        (2)
```

Stop. The learned merge list, in order, is the tokenizer:

```
1. e + s    -> es
2. es + t   -> est
3. est + _  -> est_
4. l + o    -> lo
5. lo + w   -> low
```

Three observations, because each one predicts a real behavior you have seen.

**The tokenizer discovered a morpheme without being told morphemes exist.**
`est_` is the English superlative suffix. Nothing in the algorithm knows about
suffixes; it merged `e`,`s`,`t`,`_` because that byte sequence recurred. Every
piece of linguistic structure a BPE vocabulary appears to know is an artifact of
byte frequency.

**Frequency, not meaning, decides what gets its own token.** `low` (count 5)
became a single symbol. `lower` (count 2) did not, and now costs 4 tokens:
`low`, `e`, `r`, `_`. Two words of near-identical meaning have wildly different
representations, purely because of how often they appeared in the tokenizer's
training corpus.

**Encoding is deterministic replay.** To tokenize a new string, split it into
characters and apply the merge list *in learned order*. Nothing is searched, and
the merge order matters.

```beat
id: u1-b3
type: completion
concept: c-tokens
# variants: blank steps 2 and 4 instead of 1, 3, 5; or blank the final token count.
prompt: |
  Using exactly the five merges learned above, in order, tokenize the word
  `lowest`. It never appeared in the training corpus.

  Start:      `l o w e s t _`
  Merge 1 (e+s -> es):     `l o w ____ t _`
  Merge 2 (es+t -> est):   `l o w est _`
  Merge 3 (est+_ -> est_): `l o w ____`
  Merge 4 (l+o -> lo):     `lo w est_`
  Merge 5 (lo+w -> low):   `____ est_`

  Fill the three blanks, then state how many tokens `lowest` costs.
answer: |
  Merge 1 blank: `es`   giving `l o w es t _`
  Merge 3 blank: `est_` giving `l o w est_`
  Merge 5 blank: `low`  giving `low est_`

  `lowest` costs 2 tokens: [`low`, `est_`].

  The point: a word absent from the tokenizer's corpus still encodes compactly,
  because the merges compose. This is why BPE has no unknown-token problem.
rubric: |
  Required fills, exactly: blank 1 = `es`, blank 2 = `est_`, blank 3 = `low`.
  Required token count = 2.
  All four correct = pass. Any wrong fill = fail, and diagnose:
  applying merges out of order indicates the learner missed that the merge
  list is ordered; answering 6 tokens indicates they did not apply merges at
  all; answering 3 indicates they dropped the end-of-word marker from `est_`.
check: llm
```

Now the same picture at real scale. These are actual outputs from `cl100k_base`,
the tokenizer behind GPT-3.5 and GPT-4, with $V = 100{,}277$:

| string | tokens | count |
|---|---|---|
| `The cat sat on the mat.` | `The`, ` cat`, ` sat`, ` on`, ` the`, ` mat`, `.` | 7 |
| ` strawberry` (leading space) | ` strawberry` | 1 |
| `strawberry` (no leading space) | `str`, `aw`, `berry` | 3 |
| `tokenization` | `token`, `ization` | 2 |
| `2847 + 1913` | `284`, `7`, ` +`, ` `, `191`, `3` | 6 |

Note that the leading space is part of the token: ` cat` and `cat` are different
integers. And note that `strawberry` is one symbol or three depending on whether
a space precedes it. Same word, same meaning, different atoms.

The last row is the one to sit with. `2847` is not four digit symbols. It is
`284` followed by `7`. The number `1913` is `191` followed by `3`. Digits get
grouped in threes from the left, which means the model's units column is not in a
fixed position within a token, and place-value alignment between two operands is
scrambled differently for every pair. Arithmetic in an LLM is not a hard problem
that got solved badly; it is a problem whose inputs arrive pre-shredded.

<!-- refutes: U1-M1 -->
The same shredding explains the character-counting failures. Asked how many `r`s
are in `strawberry`, the model receives three integers: `str`, `aw`, `berry`.
Nowhere in its input is there a character. It has no more direct access to the
letters of `strawberry` than you have to the individual bits of a `float` you are
reading off a screen. Any spelling ability the model has was learned indirectly,
from text *about* spelling, and it is exactly as reliable as that indirect route
suggests.

You may find that a current model answers the strawberry question correctly. That
does not refute the point. Specific famous instances get covered by training data
and by reasoning traces that spell the word out one character at a time, which is
the model routing around its own tokenization. Try an unfamiliar word.

```beat
id: u1-b4
type: predict
concept: c-tokens
prompt: |
  A colleague proposes fixing character-level failures by adding a
  character-count tool the model can call. Separately, a second colleague
  proposes retraining with a character-level tokenizer ($V \approx 256$
  bytes, no merges).

  Before reading on: state the main cost of the second proposal, in terms of
  something you already know about sequence length. Roughly what factor
  longer do sequences get for English text, and where does that cost land?
answer: |
  English text under a BPE vocabulary of ~100k averages roughly 4 characters
  per token, so a byte-level tokenizer makes sequences about 4x longer.

  Where it lands: attention cost grows with the square of sequence length
  (u3), so 4x tokens is roughly 16x attention compute for the same document.
  The KV cache grows about 4x. Effective context window, measured in actual
  text, shrinks by ~4x for a fixed token budget. And every fact the model
  must learn now has to be composed across 4x more positions, so depth is
  under more pressure too.

  The tool proposal is the correct engineering answer, and it is a preview of
  u8: the fix for a tokenization limitation is a harness that hands the model
  a different representation, not a different model.
rubric: |
  Must identify: (1) sequences get several times longer (accept 3x-5x for
  English), (2) the cost is superlinear because attention is quadratic in
  sequence length, or at minimum that compute/context cost rises sharply.
  Both = pass. Only (1) = partial, do not pass.
  An answer claiming byte-level models cannot represent words, or that
  vocabulary size limits what the model can express, indicates U1-M3
  (vocab-size-is-word-knowledge) - fail and route to that correction.
check: llm
```

## The objective, stated exactly

Everything in this book is downstream of one training objective. Here it is, with
every symbol defined.

A document is a sequence of $T$ discrete symbols $x_1, x_2, \ldots, x_T$. Each
$x_t$ is an integer in $\{1, \ldots, V\}$, where $V$ is the vocabulary size (the
number of distinct symbols the model can emit) - these are exactly the BPE
tokens of the last section, integers indexing a frozen vocabulary.

The probability of the whole document factors exactly, with no approximation, by
the chain rule of probability:

$$P(x_1, x_2, \ldots, x_T) = \prod_{t=1}^{T} P(x_t \mid x_1, \ldots, x_{t-1})$$

Read the right-hand side as: the probability of symbol 1, times the probability
of symbol 2 given symbol 1, times the probability of symbol 3 given symbols 1
and 2, and so on. The notation $x_{<t}$ abbreviates $x_1, \ldots, x_{t-1}$, the
prefix before position $t$.

This factorization is a tautology. It is true for any sequence of any kind. What
makes it useful is that it converts "model the distribution over all documents"
(a hopeless object: there are $V^T$ possible documents) into "model one
conditional distribution over $V$ options, and apply it $T$ times."

A neural network with parameters $\theta$ (a big pile of real numbers, roughly
$10^{10}$ of them) approximates that one conditional:

$$P_\theta(\cdot \mid x_{<t}) \in \mathbb{R}^V, \qquad \sum_{v=1}^{V} P_\theta(v \mid x_{<t}) = 1$$

Training minimizes the average negative log probability that the model assigned
to the symbols that actually occurred:

$$\mathcal{L}(\theta) = -\frac{1}{T}\sum_{t=1}^{T} \log P_\theta(x_t \mid x_{<t})$$

Here $\log$ is natural log, $x_t$ is the true symbol at position $t$, and
$P_\theta(x_t \mid x_{<t})$ is the single number the model assigned to that
symbol. If the model assigned probability 1 to the correct symbol every time,
$\log 1 = 0$ and the loss is 0. If it assigned probability 0 to something that
happened, $-\log 0 = \infty$. The loss punishes confident wrongness without
bound.

That expression is cross-entropy loss; u5 derives its gradient and explains why
its floor is not zero. For now, take it as: a scalar that goes down when the
model is less surprised by real text.

Two things are worth noticing immediately, because they are the source of most
confusion later.

First, the sum runs over *every* position $t$. A single 4000-token training
document produces 4000 prediction problems, not one. The model predicts position
2 from position 1, position 3 from positions 1-2, and so on, all in one forward
pass. Training is not "read the document, then guess the end."

Second, nothing in $\mathcal{L}$ mentions truth, helpfulness, reasoning, or
correctness. The only thing being optimized is agreement with the empirical
distribution of the training text. Every capability the finished system has is a
side effect of that.

```beat
id: u1-b1
type: predict
concept: c-lm-objective
prompt: |
  Two prefixes, both from a Python file the model is training on:

  (a) `def add(a, b):\n    return a + `
  (b) `>>> add(2847, 1913)\n`

  A model that has driven $\mathcal{L}$ low must put high probability on `b`
  for (a) and on `4760` for (b). Before reading on: state, in one sentence
  each, what kind of thing the model must have internalized to succeed at
  (a) versus at (b). What separates the two cases?
answer: |
  (a) requires only a local surface regularity: inside a function whose
  parameters are named a and b, the token after "a + " is overwhelmingly "b".
  A frequency table over short contexts gets this.

  (b) requires the model to actually perform the addition. No surface statistic
  of the prefix contains 4760 - that exact string may never appear near "2847"
  anywhere in the training data. The only way to drive loss down on the general
  case of this pattern is to implement an algorithm that computes sums.

  The separator: (a) can be solved by memorizing co-occurrence, (b) cannot,
  because the space of (operand, operand) pairs is far larger than any corpus.
  Compression of case (b) forces the acquisition of a mechanism.
rubric: |
  Must identify: (1) case (a) is solvable by local co-occurrence statistics
  or pattern matching, (2) case (b) requires computing/implementing addition
  rather than recalling it, (3) some version of "the space of possible operand
  pairs exceeds what could be memorized". Any 2 of 3 = pass.
  Answering that (b) is also just memorization from seeing many arithmetic
  examples, with no acknowledgement that the operand space is too large,
  = M4-style lookup-table thinking, fail.
check: llm
```

## The average is a stand-in

<!-- canon-only -->

The loss above averages over a training corpus. The quantity anyone
actually cares about is defined over text nobody has - and papers write
that distinction in a notation worth owning now.

**Expectation.** $\mathbb{E}_{x \sim \mathcal{D}}[f(x)]$ reads "the expected
value of $f(x)$ when $x$ is drawn from the distribution $\mathcal{D}$" - the
average of $f(x)$ over all possible $x$, weighted by how likely each $x$ is
under $\mathcal{D}$. The subscript names the distribution, the brackets hold
what you are averaging. You cannot compute that average, because $\mathcal{D}$
is "the distribution of all text that could exist" and you have a hard drive.
So every expectation in this book is estimated by a sample mean over a batch:

$$\mathbb{E}_{x \sim \mathcal{D}}[f(x)] \approx \frac{1}{b} \sum_{i=1}^{b} f(x_i)$$

where $b$ is the batch size and $x_1, \dots, x_b$ are the examples in the batch.

<!-- refutes: U0-M5 -->
You probably read that approximation as an equality with extra ceremony, with
$\mathbb{E}$ meaning "average of the numbers I have". Here is the prediction
that fails: if the objective were defined over your dataset, a model that
memorized the dataset would have optimally solved the stated problem, and
generalization error would not exist as a concept. Every number anyone reports
is held-out loss. The belief is appealing because the batch mean is what the
code computes and the dataset is the only concrete object in sight. What is
true: the thing you want is an expectation over a distribution nobody can
enumerate, the thing you compute is an unbiased but noisy estimate of it, and
the gap is exactly why batch size affects training stability (u5). A related
trap - $\mathbb{E}$ is a probability-weighted mean, not the typical value. The
expected roll of a fair die is 3.5.

```beat
id: u0-b5
type: compute
concept: c-notation
prompt: |
  A batch of three examples produces per-example losses $0.1$, $0.2$, and
  $9.0$. Training uses the batch mean as its estimate of
  $\mathbb{E}_{x \sim \mathcal{D}}[\ell(x)]$.

  Compute that estimate, to two decimal places. Then notice that it is larger
  than two of the three numbers it averages: this is what "probability-weighted
  mean, not typical value" looks like on real data, and it is why a handful of
  pathological examples can dominate a training objective.
answer: 3.10
check: numeric(0.01)
```

## Embeddings are coordinates, not contents

<!-- refutes: M3 -->

Token IDs are integers, and integers are useless as input to a differentiable
function - there is no sense in which token 5000 is "between" tokens 4999 and
5001. So the first thing the model does is replace each ID with a vector.

The embedding matrix is $E \in \mathbb{R}^{V \times d_{model}}$, where $V$ is
vocabulary size and $d_{model}$ is the model's internal width (4096 for a 7B-class
model). Row $i$ of $E$, written $E[i,:] \in \mathbb{R}^{d_{model}}$, is the
embedding of token $i$. "Embedding a token" is an array index. That is the entire
operation.

**You probably think the vector contains the word's meaning** - that $E[i,:]$ is
something like a struct of semantic fields, learned so that dimension 412 holds
formality and dimension 1900 holds animacy, and that this is what "semantic
vector" means.

**Here is the specific prediction that model makes, and it fails.** If meaning
lived in the coordinates, the coordinates would have to be stable. Take a trained
model, pick any permutation $\pi$ of the $d_{model}$ coordinate indices, and
relabel the residual stream by it. That means four consistent edits: permute the
columns of $E$; permute the input side of every weight matrix that *reads* from
the stream; permute the output side of every matrix that *writes* into it; and
permute the gain and bias vectors of every normalization layer. Now every
activation in the network is the old activation with its coordinates shuffled by
$\pi$, and every read undoes the shuffle before acting on it, so the logits come
out identical on every input. Nothing measurable changed. There are $4096!$ such
relabelings, all equally valid, so "dimension 412 means formality" cannot be a
fact about the model - it is a fact about an arbitrary labeling that the training
run happened to land on. (Strip the learned gains and use RMSNorm, which divides
by $\lVert x \rVert / \sqrt{d_{model}}$ and is therefore invariant under any
rotation, and the argument runs with an arbitrary orthogonal matrix $Q$ in place
of a permutation - not even the axes are privileged. Real models keep a learned
elementwise gain, which breaks the full rotational symmetry down to the
permutations; that is why interpretability work can find axis-aligned features at
all, and why what it reports are still directions rather than coordinates. u4
covers the normalization detail.)

Second failing prediction, cruder and easier to check: if the vector held the
meaning, you could take GPT-2's row for `dog` and paste it into another model's
embedding table. Meaning is meaning. In fact you get noise. Embeddings are not
portable across models, across training runs, or even across two runs of the same
code with different seeds. Nothing survives the boundary of the model that
learned them.

Third, and most decisive for what comes later: the row for `bank` is *one* row.
In `the river bank` and `the bank approved the loan`, the model's input at that
position is the identical vector. If the vector carried the word's meaning, it
would have to carry both meanings at once, unresolved, forever. Whatever
disambiguates them happens downstream, in attention, from context (u3). The
embedding cannot be where meaning lives, because the embedding does not know
which meaning it is.

**Here is why the wrong model is appealing.** The `king - man + woman = queen`
demonstration, which is genuinely striking, and a decade of "semantic vector"
marketing. Worth knowing: that demo is weaker than folklore holds. The standard
implementation excludes the three input words from the nearest-neighbor search.
Remove that exclusion and the nearest vector to `king - man + woman` is usually
`king` itself. The analogy result is partly an artifact of the search procedure.

**Here is what is actually true.** An embedding is a *learned position in a space
that the rest of the model is simultaneously learning to read*. The only thing
with meaning is the relational geometry - which vectors are close to which, which
directions separate which sets - and that geometry exists solely because it makes
downstream prediction easier. Gradient descent does not push $E[i,:]$ toward
"what token $i$ means." It pushes $E[i,:]$ wherever reduces loss, given what
layer 1 currently does with it. The embedding is an *interface*, co-designed with
its consumer, and like any interface its symbols mean nothing outside the system
that agreed on them.

The useful reframe for a systems engineer: embeddings are the model's internal
calling convention. Asking what `dog`'s embedding means in isolation is like
asking what the value in register `rsi` means without knowing the ABI.

```beat
id: u1-b5
type: self-explain
concept: c-embeddings
prompt: |
  A teammate says: "Embeddings are how the model stores what words mean.
  That's why similar words have similar vectors."

  The second sentence is true and the first is false. Explain, in your own
  words, how both can hold - and state one experiment that would distinguish
  the two claims.
answer: |
  Similar words do land near each other, but that is a consequence, not a
  storage mechanism. Training only ever rewards lower prediction loss. Two
  tokens that are interchangeable in context (`cat`/`dog`) produce similar
  downstream predictions, so gradient descent has no reason to separate them
  and every reason to let the layers above treat them alike. Proximity is the
  residue of shared predictive role.

  "Stores meaning" additionally claims the vector holds the content, portably
  and intrinsically. It does not. Three distinguishing experiments, any one
  is enough:

  1. Permute all d_model coordinates of E consistently, permute the input side
     of every matrix that reads the stream and the output side of every matrix
     that writes into it, and permute the normalization gains and biases.
     Outputs are identical. A store whose fields can be arbitrarily relabeled
     with no effect is not storing anything in those fields.
  2. Transplant one model's embedding row into another model. If meaning were
     in the vector it would transfer. It produces noise.
  3. Feed `bank` in two disambiguating contexts. The input vector is the same
     both times, yet the model's behavior differs, so the disambiguating
     content is not in the embedding.
rubric: |
  Must contain: (1) similarity-as-consequence - near vectors arise because the
  tokens play similar predictive roles, not because meaning was written there,
  (2) at least one concrete distinguishing experiment from {coordinate
  permutation invariance, cross-model transplant fails, identical vector for
  polysemous token in two contexts}, (3) some statement that the geometry is
  relational and model-internal.
  (1) and (2) = pass. Missing (2) = fail; an answer with no falsifiable
  experiment has not actually separated the claims.
  Explicitly diagnose: an answer that says the vector holds meaning "in
  compressed form" or "distributed across dimensions" is M3 surviving in
  disguise - fail, and point at the permutation argument.
check: llm
```

## Counting the embedding matrix

<!-- fade: embedding-param-count -->

<!-- refutes: M4 -->

Time to make this concrete by counting. $E$ is a matrix, so its parameter count
is the product of its dimensions:

$$|E| = V \times d_{model}$$

**Worked example: Llama-2-7B.** Its vocabulary is $V = 32{,}000$ and its width is
$d_{model} = 4096$.

$$|E| = 32{,}000 \times 4096 = 131{,}072{,}000$$

That is 131 million parameters whose entire job is to assign each of 32,000
tokens a starting position. Llama-2 does not tie its input and output matrices
(the output-layer section below explains tying), so there is a second matrix of
the same size at the output end, for 262 million total, against a full model of
about 6.74 billion.
The embedding end is roughly 4% of the model. The other 96% is transformation.

Hold onto that ratio, because it sets up a trap.

**The embedding matrix is a genuine lookup table.** One row per token, indexed by
integer ID, no computation. It is the only component of the model that works this
way. If your mental model is "the parameters are a compressed key-value store of
facts, and a bigger model has more rows," then notice what just happened: the one
part of the model that literally *is* a table by that description contains no
facts about the world at all. It contains 32,000 positions. Not one of them
encodes that Paris is in France.

Everything that could be called knowledge lives in the other 96%, and that 96%
has no rows. It is a stack of matrices that *transform* whatever is handed to
them. There is no index into it, no key to look up, no row to be missing. u4
locates where facts appear to concentrate and u5 explains why the same mechanism
produces both correct generalization and confident hallucination - they are not
two systems, one working and one broken. For now, just register the shape of the
thing: 4% table, 96% function.

```beat
id: u1-b6
type: completion
concept: c-embeddings
# variants: blank the vocab and d_model instead, giving the product and the
# percentage; or supply GPT-2 medium (V=50257, d=1024) and blank the product.
prompt: |
  GPT-2 small has $V = 50{,}257$, $d_{model} = 768$, and 124 million total
  parameters. Unlike Llama-2, it *ties* its input and output embedding
  matrices - the same $E$ is used at both ends, so it is counted once.

  Step 1. Shape of $E$:            $50{,}257 \times$ ____
  Step 2. Parameters in $E$:       $50{,}257 \times 768 =$ ____
  Step 3. Fraction of the model:   ____ $/\ 124{,}000{,}000 \approx$ ____ %

  Fill the blanks, then answer in one sentence: why is this percentage so
  much larger than Llama-2-7B's 2% (input side only), given that GPT-2's
  vocabulary is *larger*?
answer: |
  Step 1: 768
  Step 2: 38,597,376
  Step 3: 38,597,376 / 124,000,000 = 31%

  Why: the embedding matrix scales as V x d_model, which is linear in the
  model's width, while the transformer stack scales roughly as
  n_layers x d_model^2, which is quadratic in width. As models get wider and
  deeper the stack outgrows the embedding table fast. GPT-2 small is narrow
  (d=768) and shallow (12 layers), so its table dominates. Vocabulary size
  barely matters to this comparison - width does.
rubric: |
  Required: blank 1 = 768; blank 2 = 38,597,376 (accept 38.6 million or
  38,597,376 exactly; reject anything off by more than 0.1%); blank 3 result
  = 31% (accept 30-32%).
  The one-sentence answer must contain the scaling contrast: embeddings grow
  linearly in d_model, the transformer stack grows quadratically in d_model
  (times depth). Accept "the rest of the model grows faster with width."
  All three numbers correct AND the scaling contrast = pass.
  Numbers correct but the explanation cites vocabulary size as the driver
  = fail; that misses that width, not V, is what changed.
check: llm
```

## The output end: hidden state to logits

<!-- fade: hidden-to-logits -->

Skip over the middle of the model for now - u3 and u4 own it. Assume it ran, and
that at each position it produced a vector $h \in \mathbb{R}^{d_{model}}$, the
hidden state. For generating the next token, only the hidden state at the *last*
position matters.

That vector has to become a distribution over all $V$ tokens. One matrix does it.
The unembedding matrix is $W_U \in \mathbb{R}^{V \times d_{model}}$ - one row per
vocabulary entry, the same layout as $E$, which is what makes weight tying
possible at the end of this section. With $h$ a row vector of shape
$(1, d_{model})$ in this book's convention:

$$z = h\,W_U^T, \qquad z \in \mathbb{R}^{V}$$

The transpose is load-bearing, not tidying. $h$ carries $d_{model}$ on its second
axis and $W_U$ carries $d_{model}$ on its second axis too, so nothing contracts
until one of them is flipped; $W_U^T$ is $(d_{model}, V)$ and the join works.
This is literally what the code does - a PyTorch `nn.Linear` stores its weight as
(out, in) and computes `h @ W.T`.

Here $z$ is the vector of **logits**, one real number per vocabulary entry. Read
componentwise, this is the part that matters:

$$z_j = \langle W_U[j,:],\ h \rangle = \sum_{k=1}^{d_{model}} W_U[j,k] \, h_k$$

$W_U[j,:] \in \mathbb{R}^{d_{model}}$ is row $j$, token $j$'s output direction,
and $\langle \cdot, \cdot \rangle$ is the dot product: multiply the vectors componentwise, sum. So the logit for
token $j$ is the *similarity between the hidden state and token $j$'s stored
direction*. The output layer is $V$ dot products run in parallel: score the
hidden state against every token's direction, keep all the scores.

This is worth pausing on, because it is the same operation you will meet in u3
under a different name. Dot-product-against-a-set-of-stored-directions is the
model's one primitive for "compare this to those." Attention uses it to compare a
query against keys; the output layer uses it to compare a hidden state against
tokens. Same shape, different operands.

**Worked example.** Take $d_{model} = 3$ and $V = 4$, with vocabulary
$[\ \texttt{" the"},\ \texttt{" a"},\ \texttt{" mat"},\ \texttt{" quantum"}\ ]$
at indices 0-3. Suppose the stack, having processed `The cat sat on`, emitted

$$h = \begin{bmatrix} 2 & -1 & 0.5 \end{bmatrix}$$

and the model's unembedding matrix is

$$W_U = \begin{bmatrix} 1 & 0 & 0 \\ 0.5 & 1 & 2 \\ 0 & 1 & -1 \\ -1 & 2 & 0 \end{bmatrix}$$

with one row per vocabulary entry, in the order listed. Row by row:

- $z_0 = (1)(2) + (0)(-1) + (0)(0.5) = 2 + 0 + 0 = 2.0$ &nbsp; for `" the"`
- $z_1 = (0.5)(2) + (1)(-1) + (2)(0.5) = 1 - 1 + 1 = 1.0$ &nbsp; for `" a"`
- $z_2 = (0)(2) + (1)(-1) + (-1)(0.5) = 0 - 1 - 0.5 = -1.5$ &nbsp; for `" mat"`
- $z_3 = (-1)(2) + (2)(-1) + (0)(0.5) = -2 - 2 + 0 = -4.0$ &nbsp; for `" quantum"`

$$z = [\,2.0,\ 1.0,\ -1.5,\ -4.0\,]$$

`" the"` scores highest, `" quantum"` lowest. Sensible after `The cat sat on`.

Three properties of logits, all of which cause trouble when misremembered.

**They are unnormalized and unbounded.** A logit can be any real number. Nothing
forces them to sum to anything. Turning $z$ into a probability distribution is
softmax's job, and u2 owns it; all you need here is that some function maps
$\mathbb{R}^V$ to a distribution.

**Only differences are meaningful.** Add the same constant $c$ to every logit and
the resulting distribution is unchanged: softmax divides $e^{z_j + c}$ by
$\sum_k e^{z_k + c}$, and the factor $e^c$ cancels top and bottom. So
$[2.0, 1.0, -1.5, -4.0]$ and $[102.0, 101.0, 98.5, 96.0]$ are the *same* model
output. "The logit for token X was 12" is not a statement about anything. The gap
between two logits is.

**The name is about log-odds, and it is nearly literal.** After softmax,
$z_j = \log P_j + \text{const}$: logits are log-probabilities up to that same
additive constant. A gap of $\ln 2 \approx 0.69$ between two logits means one
token is exactly twice as likely as the other, regardless of what the absolute
values are.

One implementation note. Many models set $W_U = E$ - the same matrix that mapped
IDs to vectors on the way in maps hidden states to scores on the way out. This is
**weight tying**, and it saves $V \times d_{model}$ parameters (31% of GPT-2
small, from the last section). It works because both directions are asking about
the same relationship between tokens and directions in the residual stream:
reading in writes token $j$'s direction into the stream, and scoring out measures
how much of token $j$'s direction is present. Llama-2 and Llama-3 do not tie;
GPT-2 does. It is a tradeoff, not a law.

```beat
id: u1-b7
type: completion
concept: c-logits
# variants: blank rows 0 and 3 instead; or give z and blank one entry of h.
prompt: |
  Same $W_U$ as above:

  $$W_U = \begin{bmatrix} 1 & 0 & 0 \\ 0.5 & 1 & 2 \\ 0 & 1 & -1 \\ -1 & 2 & 0 \end{bmatrix}$$

  vocabulary $[\texttt{" the"}, \texttt{" a"}, \texttt{" mat"}, \texttt{" quantum"}]$,
  but a new hidden state from a different context:

  $$h = \begin{bmatrix} 0 & 1 & 1 \end{bmatrix}$$

  $z_0 = (1)(0) + (0)(1) + (0)(1) =$ ____
  $z_1 = (0.5)(0) + (1)(1) + (2)(1) =$ ____
  $z_2 = (0)(0) + (1)(1) + (-1)(1) =$ ____
  $z_3 = (-1)(0) + (2)(1) + (0)(1) =$ ____

  Fill the blanks. Then: which token wins, and what is the logit gap between
  the winner and `" the"`?
answer: |
  z_0 = 0
  z_1 = 3
  z_2 = 0
  z_3 = 2

  z = [0, 3, 0, 2]. The winner is `" a"` at 3. The gap to `" the"` (logit 0)
  is 3.

  Note that the same W_U now ranks `" a"` first where the previous hidden
  state ranked `" the"` first. The unembedding matrix is fixed; the ranking
  is entirely a function of h.
rubric: |
  Required, exactly: z_0 = 0, z_1 = 3, z_2 = 0, z_3 = 2. Winner = `" a"`.
  Gap = 3.
  All six correct = pass. Arithmetic slips = fail (this is mechanical).
  If the learner reports the gap as a probability or a percentage, that is
  M2 leaking early - logits are not probabilities. Fail and flag M2.
check: llm
```

```beat
id: u1-b8
type: compute
concept: c-logits
prompt: |
  A model with $V = 4$ produces logits $z = [1.0,\ 3.0,\ 2.0,\ 0.5]$.

  Using only the fact that a logit gap of $\ln 2$ means one token is twice as
  likely as another - that is, the probability *ratio* of two tokens is
  $e^{z_i - z_j}$ - compute how many times more likely the top-scoring token
  is than the second-highest.

  Give a single number to 2 decimal places.
answer: 2.72
check: numeric(0.01)
```

## What the bet actually claims

Assemble the pieces. Text becomes integers by a frozen frequency-derived merge
table. Integers become positions in a learned space by a table lookup. Something
in the middle transforms those positions. The result is dotted against every
token's direction to produce $V$ scores, which become a distribution, from which
one token is drawn. Append it, run again.

That is the whole system. The bet is that this loop, scaled, produces something
worth calling intelligence - not because anyone designed intelligence into it,
but because prediction has no ceiling and modeling is the only way up.

Three things to carry forward, each of which is a correction you will need:

**Nothing here is a database.** 4% of the parameters form a genuine lookup table
containing no facts. The remaining 96% is a function with no rows to look up. When
u5 explains hallucination, this is why the framing "it retrieved the wrong row"
has no referent.

**Nothing here is portable.** Tokens are integers that mean something only
relative to one merge table. Embeddings are positions that mean something only
relative to one model's downstream layers. The system is internally coherent and
externally meaningless, which is exactly what you would expect from something
whose only constraint was reproducing a corpus.

**The objective is per-token; the computation is not.** Loss is summed over
positions independently, which tempts the conclusion that the model is myopic and
cannot plan. The next unit gives you the machinery to see why that does not
follow: a hidden state at position $t$ is free to encode structure that only pays
off at position $t+40$, and gradient descent rewards it for doing so, because
that is where the loss went down.

u2 builds the math floor - softmax properly, gradients, loss surfaces - and u3
opens the middle of the model.

## Notation in this unit

<!-- canon-only -->

Reference, not reading. Return here when a symbol goes blurry; nothing
below is new.

The index letters are near-universal across papers. These five cover the book:

| Symbol | Means | Typical value |
| --- | --- | --- |
| $b$ | batch size - how many independent sequences processed at once | 1 to 1024 |
| $n$ | sequence length - number of tokens in one sequence | 8 to 1,000,000 |
| $d_{\text{model}}$ | width of the residual stream - the vector size carried between layers | 768 to 16384 |
| $d_k$ | width of one attention head's query/key vectors | 64 to 128 |
| $V$ | vocabulary size - how many distinct tokens exist | ~32,000 to ~200,000 |

A batch of token embeddings therefore has shape $(b, n, d_{\text{model}})$, the
single most common shape in this book. Most equations drop the batch dimension,
so you will see $X$ with shape $(n, d_{\text{model}})$ and are expected to know
that everything applies per-sequence, in parallel, across the batch.

<!-- refutes: U0-M4 -->
You probably think the row-vs-column orientation of a vector is cosmetic - a
transpose here or there, the kind of thing you fix when the code throws. Here is
the prediction that fails: under that belief, a paper writing $xW$ and a
textbook writing $Wx$ describe the same operation with the same matrix, so a
shape derived in one carries to the other. It does not. In $Wx$, the
linear-algebra-textbook form, $x$ is a column vector of shape
$(d_{\text{in}}, 1)$ and $W$ has shape $(d_{\text{out}}, d_{\text{in}})$ - input
dimension **last**. In $xW$, the form nearly all ML papers and every tensor
library use, $x$ is a row vector of shape $(1, d_{\text{in}})$ and $W$ has shape
$(d_{\text{in}}, d_{\text{out}})$ - input dimension **first**. The same weight
matrix is stored transposed between the two worlds.

The belief is appealing because in scalar-land orientation genuinely is
cosmetic, and because broadcasting hides orientation errors until they surface
three layers downstream as a wrong-but-plausible shape.

What is actually true: this book, like the papers, uses the **row-vector
convention** for every equation that carries data through a model. Data on the
left, weights on the right, dimensions contract at the join:

$$X W = Y, \quad X: (n, d_{\text{in}}), \quad W: (d_{\text{in}}, d_{\text{out}}), \quad Y: (n, d_{\text{out}})$$

where $X$ holds one token vector per row, $W$ is a learned weight matrix, and
$Y$ holds one output vector per row. The inner dimensions - the $d_{\text{in}}$
on both sides of the join - must match, and then they vanish. The outer
dimensions survive. That is the only shape rule you need for the rest of the
book.

One deliberate exception, flagged now so it does not read as a contradiction
later. Unit u2 works its two-dimensional geometric examples - rotations,
scalings, projections - in the textbook column form $Wx$, because that is the
form every linear algebra text and every picture of a rotation uses, and it
says so at the point of use. Those are the same maps with the same weights
stored transposed: $(Wx)^T = x^T W^T$. Outside that one section, data is on the
left. When you meet an unfamiliar equation, do not look at the letter order -
look at which index the two factors share, because that is the one that
contracts.

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
