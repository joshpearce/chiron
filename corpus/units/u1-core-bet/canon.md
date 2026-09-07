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
next token becomes, at sufficient scale, the systems you use every day. That
idea is the bet, and this unit states it precisely and walks it end to end -
what a token is, how text enters the model, how scores come out, and what the
training objective does and does not promise. By the end you can trace
`The cat sat on` from characters to a scored vocabulary, and say exactly which
claims about "understanding" that loop licenses.


## Why prediction forces world modeling

The bet, in full: *if you make a system good enough at predicting the next
symbol in human-generated text, you get world modeling for free, because there
is no other way to be good at it.*

"Free" is doing a lot of work in that sentence, so the sentence deserves an
argument rather than a nod. The argument starts from behavior, not machinery:
four fragments of ordinary text, each cut off one word early.

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
  Fill each of the four blanks above yourself - all four are easy for you.
  Then, before reading on, commit: what would a system have to be able to DO
  to fill them reliably? Are the four demands the same kind of thing, or
  different kinds?
options:
  - text: "Four different kinds of capability behind one uniform interface: recall a stored fact about the world (Barack Obama), run a sorting algorithm to completion (9), apply a geometric theorem (angle ACB), and track two agents' divergent beliefs about the same object (the drawer, since Alice never saw the move)."
    correct: true
    explain: "Right. Nothing on the surface tells the four apart - each arrives as the same question, what token comes next - yet a fact store, an algorithm, deductive rule application and a theory of other minds are what the blanks demand. That is the whole argument for why prediction pressures toward world modeling."
  - text: "One kind: retrieval. Each of these continuations occurred somewhere in the corpus, so all four are rows fetched from a large store of remembered text; a bigger store covers more blanks."
    misconception: M4
    explain: "A store can only answer for what it holds. Ask for the sorted form of a list nobody ever wrote down, or for what Alice believes in a scene composed on the spot, and both are answered anyway. Parameters define a function over token streams, not rows; generalization and confident hallucination are the same mechanism."
  - text: "Three demands and one impossibility: the first three are fillable, but the keys-in-the-drawer blank needs a belief held across several sentences, and a loss that scores each position only on its own next token cannot reward carrying state that far."
    misconception: U1-M2
    explain: "The loss terms are additive; the computations producing them share parameters. Gradient from a late position flows back into whatever produced the earlier hidden state, so encoding Alice's belief early is directly rewarded by the later term. The objective decomposes per position; credit assignment does not."
check: choice
```

Each blank asks for something different. The first needs a fact about the
world. The second needs a sorting algorithm run to completion. The third needs
a geometric theorem applied. The fourth needs two people's beliefs tracked
separately, because Alice never saw the keys move. Four different capabilities,
and nothing on the surface tells them apart: every one arrives as the same
question, "what token comes next?", and human text is full of all four. So a
system that gets good at next-token prediction across human text must carry
some working version of each capability, because that is what the blanks
demand.

"Must" is the word to press on. The rest of this section makes it precise,
and then marks the exact point where the argument stops.

The precise form goes through compression. Predicting a symbol well and
encoding it cheaply are the same skill: Shannon's source coding theorem says
that a probability model $Q$ over symbols lets you encode a symbol $x$ in
$-\log_2 Q(x)$ bits, and that no code can do better on average than the true
entropy of the source. This is not a metaphor; arithmetic coding achieves that
bound in practice. Now look at the number training pushes down. The loss,
written $\mathcal{L}$, is the average over the corpus of $-\ln$ of the
probability the model gave the symbol that actually came next (the objective
section below states it exactly). That is Shannon's quantity in natural-log
units, so converting to base 2 turns the loss into the compressed size of the
training corpus, in bits per token, under the model's code:

$$\text{bits per token} = \frac{\mathcal{L}}{\ln 2}$$

A language model is therefore a compressor, and lowering its loss is
compressing the corpus harder. That reframing does the work, because
compression has a floor tied to structure: the only way to encode a string in
fewer bits than its length is to exploit regularity in the process that
generated it. Put the four blanks back in that light. A frequency table over
5-token windows spends many bits on every one of them, those bits show up in
the loss, the corpus is full of such continuations, and the gradient pushes on
every one. The cheapest way to pay for a sorting example is to sort. The
cheapest way to pay for Alice is to track what Alice knows.

That is the argument, and it is strong. Here is where it stops, because the
overclaimed version is everywhere. The compression argument is a statement
about the *limit*: a system with loss arbitrarily close to the entropy of
language must model whatever generated that language. It says nothing about a
transformer trained with SGD at achievable loss, which may hold those
mechanisms or may hold a large bag of shallow heuristics that happen to cover
most of the distribution. Those are different claims. The first is a theorem
about codes; the second is an empirical question, and the honest answer is
"considerably more than the skeptics predicted in 2019, considerably less than
the marketing says." The failure modes you have seen with LLM tooling -
confident wrong arithmetic, brittle multi-step reasoning that collapses when
you rename the variables - are what heuristics standing in for mechanisms look
like from the outside.

Hold both halves. The objective *pressures* toward world modeling,
monotonically and without limit. What you get at any particular scale is an
empirical fact about that scale, not a guarantee.

```beat
id: u1-b2
type: self-explain
concept: c-lm-objective
prompt: |
  A colleague asks you to explain, without re-reading, why 'a model that
  predicts text well must model the world that generated the text' is an
  argument about a limit - and to name one concrete observation about current
  LLM behavior showing the limit has not been reached.

  Which explanation gets both halves right?
options:
  - text: "Loss is bits per token, and you can only spend few bits on a continuation by exploiting real structure in the process that generated it. Taken to its endpoint - loss at the entropy floor - the predictor must have captured that structure, including facts, algorithms and agents' mental states. Nothing says a given architecture, optimizer and budget reaches that endpoint: a shallow heuristic covering most of a pattern lowers loss a lot while implementing none of the mechanism. Evidence the limit is unreached: multi-digit arithmetic correct on common operands and wrong on rare ones, and reasoning chains that hold on the textbook phrasing and collapse when the variable names are swapped."
    correct: true
    explain: "Both halves held at once. The compression bound is a theorem about codes in the limit; what a particular model at a particular scale contains is an empirical question, and brittleness under renaming is what heuristics standing in for mechanisms look like from outside."
  - text: "The limit has already been reached for facts, not for mechanisms: a large model has effectively stored the corpus, so the remaining failures are gaps in what got stored, and the fix is more data covering the missing cases."
    misconception: M4
    explain: "This reads the parameters as a store with missing rows. But the arithmetic failures are on operand pairs no corpus could contain, and the model answers them anyway - wrongly and confidently. The same distributed function produces correct generalization and hallucination; there is no row that went missing."
  - text: "It is a limit argument in the sense that the loss has a target of zero: until training drives it there the model is deficient, and the brittle reasoning is simply leftover loss that more training will remove."
    misconception: M18
    explain: "Cross-entropy has an irreducible floor, the entropy of language itself, because the next token is genuinely uncertain. Near-zero loss on training text means memorization, not mastery. The gap that matters is the one to the floor, not the one to zero."
  - text: "It is a limit argument about optimization: gradient descent has not yet reached THE minimum of the loss, and at that minimum the mechanisms are guaranteed to be present. The renaming failures show only that training stopped early."
    misconception: M5
    explain: "There is no single minimum to reach. SGD lands in one of astronomically many good-enough low-loss regions, and two seeds land in different ones while both work. The limit in the compression argument is a loss value approaching the entropy floor, not a distinguished point in weight space."
check: choice
```

## Tokens: BPE from scratch

<!-- fade: bpe-tokenize -->

The argument above kept saying "symbol", and the choice of symbol is the
first design decision in the whole system. The model does not consume
characters and does not consume words. It consumes integers drawn from a fixed
vocabulary of $V$ subword strings, built before training ever starts by an
algorithm called byte-pair encoding and frozen for the model's entire life.

Why not characters? A 4000-character document would be 4000 positions, and u3
will show that attention cost grows as the square of sequence length. Why not
words? The vocabulary would be unbounded, and every typo, identifier, and
non-English string would be an unknown symbol. BPE sits between the two: start
from characters, then repeatedly glue together whichever adjacent pair occurs
most often, until you have $V$ symbols. Frequent words become single symbols,
rare words decompose into pieces, and nothing is ever unrepresentable.

The whole algorithm fits on a corpus small enough to do by hand. Take four
distinct words with these counts:

| word | count |
|---|---|
| `low` | 5 |
| `lower` | 2 |
| `newest` | 6 |
| `widest` | 3 |

Split every word into characters, with `_` marking end-of-word so the
algorithm can tell a word-final `t` from a word-internal one:

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

Three observations follow from those five merges, and each predicts a real
behavior you have seen.

**The tokenizer discovered a morpheme without being told morphemes exist.**
`est_` is the English superlative suffix, and nothing in the algorithm knows
about suffixes; it merged `e`, `s`, `t`, `_` because that byte sequence
recurred. Every piece of linguistic structure a BPE vocabulary appears to know
arrived the same way, as an artifact of byte frequency.

**Frequency, not meaning, decides what gets its own token.** `low` appeared 5
times and became a single symbol. `lower` appeared twice, did not, and now
costs 4 tokens: `low`, `e`, `r`, `_`. Two words of near-identical meaning got
wildly different representations, purely because of how often each appeared in
the tokenizer's training corpus.

**Encoding is deterministic replay.** To tokenize a new string, split it into
characters and apply the merge list *in learned order*. Nothing is searched,
and the order matters.

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

  Which filling of the three blanks, and which token count for `lowest`, is
  right?
options:
  - text: "Blanks `es`, `est_`, `low`; the sequence runs `l o w es t _`, then `l o w est_`, then `low est_`. `lowest` costs 2 tokens: [`low`, `est_`]."
    correct: true
    explain: "Right, and the point of the exercise: a word absent from the tokenizer's corpus still encodes compactly, because the learned merges compose. This is why BPE has no unknown-token problem."
  - text: "Blanks `es`, `est`, `low`; the end-of-word marker is bookkeeping for the training procedure and is dropped when tokenizing real text, so `lowest` costs 2 tokens: [`low`, `est`]."
    misconception: U1-M1
    explain: "The `_` is a byte in the sequence like any other, and merge 3 was learned as est + _ -> est_, one symbol. Dropping it would make word-final `est` indistinguishable from word-internal `est`, which is exactly the distinction the marker exists to keep. Token boundaries are artifacts of byte frequency, not word structure you may tidy up."
  - text: "No blank can be filled: `lowest` was never in the tokenizer's corpus, so no learned merge applies and the word falls back to its base units - 7 tokens, one per character including the marker."
    misconception: U1-M3
    explain: "Encoding is deterministic replay of the merge list over whatever string arrives; the list does not know or care which words it was learned from. Base units are the fallback for byte sequences no merge covers, not the normal case. Nothing is unrepresentable, and rare strings merely cost more tokens."
check: choice
```

The same picture at real scale looks like this. These are actual outputs from
`cl100k_base`, the tokenizer behind GPT-3.5 and GPT-4, with $V = 100{,}277$:

| string | tokens | count |
|---|---|---|
| `The cat sat on the mat.` | `The`, ` cat`, ` sat`, ` on`, ` the`, ` mat`, `.` | 7 |
| ` strawberry` (leading space) | ` strawberry` | 1 |
| `strawberry` (no leading space) | `str`, `aw`, `berry` | 3 |
| `tokenization` | `token`, `ization` | 2 |
| `2847 + 1913` | `284`, `7`, ` +`, ` `, `191`, `3` | 6 |

Two of the rows show that the leading space is part of the token: ` cat` and
`cat` are different integers, and `strawberry` is one symbol with a space
before it and three symbols without. Same word, same meaning, different atoms.

The last row is the one to sit with. `2847` is not four digit symbols; it is
`284` followed by `7`, and `1913` is `191` followed by `3`. Digits get grouped
in threes from the left, so the units column sits at no fixed position within
a token, and the place-value alignment between two operands is scrambled
differently for every pair. Arithmetic in an LLM is not a hard problem that
got solved badly. It is a problem whose inputs arrive pre-shredded.

<!-- refutes: U1-M1 -->
The same shredding explains the character-counting failures. Asked how many
`r`s are in `strawberry`, the model receives three integers, `str`, `aw`,
`berry`, and nowhere in that input is there a character. It has no more direct
access to the letters of `strawberry` than you have to the individual bits of
a `float` you are reading off a screen. Whatever spelling ability the model has
was learned indirectly, from text *about* spelling, and it is exactly as
reliable as that indirect route suggests.

You may find that a current model answers the strawberry question correctly,
and that does not refute the point. Specific famous instances get covered by
training data and by reasoning traces that spell the word out one character at
a time, which is the model routing around its own tokenization. Try an
unfamiliar word.

```beat
id: u1-b4
type: predict
concept: c-tokens
prompt: |
  A colleague proposes fixing character-level failures by adding a
  character-count tool the model can call. Separately, a second colleague
  proposes retraining with a character-level tokenizer ($V$ about 256 bytes,
  no merges).

  Before reading on, commit: what is the main cost of the second proposal,
  in terms of something you already know about sequence length?
options:
  - text: "Sequences get about 4x longer for English (a ~100k BPE vocabulary averages roughly 4 characters per token), and the cost is superlinear: attention compute grows with the square of sequence length, so about 16x for the same document, while the KV cache grows about 4x and the effective context measured in actual text shrinks about 4x."
    correct: true
    explain: "Right, and it is why the tool is the correct engineering answer: the fix for a tokenization limitation is a harness that hands the model a different representation, not a different model. The quadratic term is developed in u3."
  - text: "Sequences get about 4x longer, and the cost is about 4x with it: attention is computed for all positions in parallel, so its cost is linear in sequence length and the extra positions are absorbed by the hardware."
    misconception: M10
    explain: "Parallel hardware hides the cost at small $n$; it does not change the asymptotics. Every query dots every key, so attention is quadratic in sequence length - 4x the tokens is roughly 16x the attention compute, and it is exactly why long context is expensive."
  - text: "The main cost is expressiveness: with $V$ about 256 the model has only a few hundred vocabulary entries, so most English words fall outside it and arrive as unknown tokens."
    misconception: U1-M3
    explain: "256 byte values represent every string that exists, including every word, identifier and non-English fragment - each simply costs more positions. $V$ is a budget trading embedding and output-layer cost against tokens-per-character; it constrains sequence length and parameter count, never what can be expressed."
check: choice
```

## The objective, stated exactly

Every section so far has leaned on "the loss" without writing it down.
Everything in this book is downstream of that one training objective, so here
it is, with every symbol defined.

A document is a sequence of $T$ discrete symbols $x_1, x_2, \ldots, x_T$. Each
$x_t$ is an integer in $\{1, \ldots, V\}$, where $V$ is the vocabulary size,
the number of distinct symbols the model can emit; these are exactly the BPE
tokens of the last section, integers indexing a frozen vocabulary.

The probability of the whole document factors exactly, with no approximation,
by the chain rule of probability:

$$P(x_1, x_2, \ldots, x_T) = \prod_{t=1}^{T} P(x_t \mid x_1, \ldots, x_{t-1})$$

Read the right-hand side as: the probability of symbol 1, times the
probability of symbol 2 given symbol 1, times the probability of symbol 3
given symbols 1 and 2, and so on. The prefix before position $t$, that is
$x_1, \ldots, x_{t-1}$, is abbreviated $x_{<t}$.

That factorization is a tautology, true for any sequence of any kind, and its
use is in what it converts. "Model the distribution over all documents" is a
hopeless object, since there are $V^T$ possible documents. "Model one
conditional distribution over $V$ options, and apply it $T$ times" is a job.

The job goes to a neural network with parameters $\theta$, a big pile of real
numbers, roughly $10^{10}$ of them, which approximates that one conditional:

$$P_\theta(\cdot \mid x_{<t}) \in \mathbb{R}^V, \qquad \sum_{v=1}^{V} P_\theta(v \mid x_{<t}) = 1$$

Training then minimizes the average negative log probability the model
assigned to the symbols that actually occurred:

$$\mathcal{L}(\theta) = -\frac{1}{T}\sum_{t=1}^{T} \log P_\theta(x_t \mid x_{<t})$$

Here $\log$ is the natural log, $x_t$ is the true symbol at position $t$, and
$P_\theta(x_t \mid x_{<t})$ is the single number the model assigned to that
symbol. Assign probability 1 to the correct symbol every time and
$\log 1 = 0$, so the loss is 0. Assign probability 0 to something that
happened and $-\log 0 = \infty$. The loss punishes confident wrongness without
bound.

That expression is cross-entropy loss. u5 derives its gradient and explains
why its floor is not zero; for now, take it as a scalar that goes down when
the model is less surprised by real text.

Two features of that expression are the source of most confusion later, so
notice them now.

First, the sum runs over *every* position $t$. A single 4000-token training
document is therefore 4000 prediction problems, not one: the model predicts
position 2 from position 1, position 3 from positions 1-2, and so on, all in
one forward pass. Training is not "read the document, then guess the end."

Second, nothing in $\mathcal{L}$ mentions truth, helpfulness, reasoning, or
correctness. The only thing being optimized is agreement with the empirical
distribution of the training text, and every capability the finished system
has is a side effect of that agreement.

```beat
id: u1-b1
type: predict
concept: c-lm-objective
prompt: |
  Two prefixes, both from a Python file the model is training on:

  (a) `def add(a, b):\n    return a + `
  (b) `>>> add(2847, 1913)\n`

  A model that has driven $\mathcal{L}$ low must put high probability on `b`
  for (a) and on `4760` for (b). Before reading on, commit: what must the
  model have internalized for (a) versus (b), and what separates the cases?
options:
  - text: "(a) needs only a local surface regularity - inside a function whose parameters are named a and b, the token after `a + ` is overwhelmingly `b`, and a frequency table over short contexts gets it. (b) needs the addition actually performed: no surface statistic of the prefix contains 4760, and the space of operand pairs is far larger than any corpus, so the only way to drive the loss down on the general pattern is to implement an algorithm."
    correct: true
    explain: "Right. (a) is solvable by memorizing co-occurrence; (b) cannot be, which is the compression argument made concrete - paying few bits for case (b) forces the acquisition of a mechanism."
  - text: "Both are the same kind of thing: arithmetic appears constantly in the training data, so the model has the result of 2847 + 1913 stored the same way it has `b` after `a + `, and it looks the pair up."
    misconception: M4
    explain: "Count the pairs: four-digit by four-digit is tens of millions of combinations, and that exact string may appear nowhere near `2847` in the data. What the model has is a function over token streams, which is also why it produces confident wrong sums rather than reporting a miss."
  - text: "(b) is easier than it looks: the digits reach the model one character at a time, so it only needs the column-by-column carrying procedure it has seen written out in text - the same kind of surface pattern that solves (a)."
    misconception: U1-M1
    explain: "The digits do not arrive as characters. `2847 + 1913` tokenizes as `284`, `7`, ` +`, ` `, `191`, `3` - digits grouped in threes from the left, so the units column sits at no fixed position and the place-value alignment between operands is scrambled differently for every pair. The inputs to that procedure arrive pre-shredded."
check: choice
```

## The average is a stand-in

<!-- canon-only -->

The loss above averages over a training corpus. The quantity anyone actually
cares about is defined over text nobody has, and papers write that distinction
in a notation worth owning now.

**Expectation.** $\mathbb{E}_{x \sim \mathcal{D}}[f(x)]$ reads "the expected
value of $f(x)$ when $x$ is drawn from the distribution $\mathcal{D}$": the
average of $f(x)$ over all possible $x$, weighted by how likely each $x$ is
under $\mathcal{D}$. The subscript names the distribution and the brackets
hold what you are averaging. That average cannot be computed, because
$\mathcal{D}$ is "the distribution of all text that could exist" and you have
a hard drive. So every expectation in this book is estimated by a sample mean
over a batch:

$$\mathbb{E}_{x \sim \mathcal{D}}[f(x)] \approx \frac{1}{b} \sum_{i=1}^{b} f(x_i)$$

where $b$ is the batch size and $x_1, \dots, x_b$ are the examples in the batch.

<!-- refutes: U0-M5 -->
You probably read that approximation as an equality with extra ceremony, with
$\mathbb{E}$ meaning "average of the numbers I have". Here is the prediction
that fails: if the objective were defined over your dataset, a model that
memorized the dataset would have optimally solved the stated problem, and
generalization error would not exist as a concept. Yet every number anyone
reports is held-out loss. The belief is appealing because the batch mean is
what the code computes and the dataset is the only concrete object in sight.
What is true: the thing you want is an expectation over a distribution nobody
can enumerate, the thing you compute is an unbiased but noisy estimate of it,
and the gap between them is exactly why batch size affects training stability
(u5). A related trap: $\mathbb{E}$ is a probability-weighted mean, not the
typical value. The expected roll of a fair die is 3.5.

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

The objective takes integers in. Integers are useless as input to a
differentiable function, because there is no sense in which token 5000 is
"between" tokens 4999 and 5001, so the first thing the model does is replace
each ID with a vector.

The vectors live in the embedding matrix $E \in \mathbb{R}^{V \times d_{model}}$,
where $V$ is the vocabulary size and $d_{model}$ is the model's internal width
(4096 for a 7B-class model). Row $i$ of $E$, written
$E[i,:] \in \mathbb{R}^{d_{model}}$, is the embedding of token $i$, so
"embedding a token" is an array index. That is the entire operation.

**You probably think the vector contains the word's meaning** - that $E[i,:]$ is
something like a struct of semantic fields, learned so that dimension 412 holds
formality and dimension 1900 holds animacy, and that this is what "semantic
vector" means.

**Here is the specific prediction that model makes, and it fails.** If meaning
lived in the coordinates, the coordinates would have to be stable. They are
not, and here is the experiment that shows it. Take a trained model, pick any
permutation $\pi$ of the $d_{model}$ coordinate indices, and relabel the
residual stream by it. Relabeling means four consistent edits: permute the
columns of $E$; permute the input side of every weight matrix that *reads*
from the stream; permute the output side of every matrix that *writes* into
it; and permute the gain and bias vectors of every normalization layer. After
those edits every activation in the network is the old activation with its
coordinates shuffled by $\pi$, every read undoes the shuffle before acting on
it, and the logits come out identical on every input. Nothing measurable
changed. There are $4096!$ such relabelings, all equally valid, so "dimension
412 means formality" cannot be a fact about the model; it is a fact about an
arbitrary labeling the training run happened to land on. (Strip the learned
gains and use RMSNorm, which divides by $\lVert x \rVert / \sqrt{d_{model}}$
and is therefore invariant under any rotation, and the argument runs with an
arbitrary orthogonal matrix $Q$ in place of a permutation: not even the axes
are privileged. Real models keep a learned elementwise gain, which breaks the
full rotational symmetry down to the permutations; that is why
interpretability work can find axis-aligned features at all, and why what it
reports are still directions rather than coordinates. u4 covers the
normalization detail.)

A second failing prediction is cruder and easier to check. If the vector held
the meaning, you could take GPT-2's row for `dog` and paste it into another
model's embedding table, since meaning is meaning. In fact you get noise.
Embeddings are not portable across models, across training runs, or even
across two runs of the same code with different seeds; nothing survives the
boundary of the model that learned them.

The third failing prediction is the most decisive for what comes later. The
row for `bank` is *one* row, so in `the river bank` and `the bank approved the
loan` the model's input at that position is the identical vector. If the
vector carried the word's meaning, it would have to carry both meanings at
once, unresolved, forever. Whatever disambiguates them happens downstream, in
attention, from context (u3). The embedding cannot be where meaning lives,
because the embedding does not know which meaning it is.

**Here is why the wrong model is appealing.** The `king - man + woman = queen`
demonstration is genuinely striking, and a decade of "semantic vector"
marketing was built on it. The demo is weaker than folklore holds: the
standard implementation excludes the three input words from the
nearest-neighbor search, and without that exclusion the nearest vector to
`king - man + woman` is usually `king` itself. The analogy result is partly an
artifact of the search procedure.

**Here is what is actually true.** An embedding is a *learned position in a
space that the rest of the model is simultaneously learning to read*. The only
thing in that space with meaning is the relational geometry, which vectors are
close to which and which directions separate which sets, and that geometry
exists solely because it makes downstream prediction easier. Gradient descent
does not push $E[i,:]$ toward "what token $i$ means"; it pushes $E[i,:]$
wherever reduces loss, given what layer 1 currently does with it. The
embedding is an *interface*, co-designed with its consumer, and like any
interface its symbols mean nothing outside the system that agreed on them.

For a systems engineer, the reframe is that embeddings are the model's
internal calling convention. Asking what `dog`'s embedding means in isolation
is asking what the value in register `rsi` means without knowing the ABI.

```beat
id: u1-b5
type: self-explain
concept: c-embeddings
prompt: |
  A teammate says: 'Embeddings are how the model stores what words mean.
  That's why similar words have similar vectors.'

  The second sentence is true and the first is false. Which explanation of how
  both can hold - with an experiment that would distinguish the two claims -
  is right?
options:
  - text: "Proximity is a consequence of shared predictive role, not a storage mechanism: tokens interchangeable in context produce similar downstream predictions, so gradient descent has no reason to separate them. The distinguishing experiment: permute the $d_{model}$ coordinates of $E$ and consistently permute the input side of every matrix that reads the residual stream, the output side of every matrix that writes into it, and the normalization gains and biases - the logits are identical on every input, so nothing measurable is stored in those coordinates."
    correct: true
    explain: "Right, and two cheaper checks agree: an embedding row transplanted into another model is noise, and `bank` gets one identical row in `the river bank` and `the bank approved the loan`, so whatever disambiguates them is downstream. An embedding is a learned position in a space the rest of the model is simultaneously learning to read."
  - text: "Both hold because the meaning is stored in compressed, distributed form rather than one field per dimension: no single coordinate is formality or animacy, but the content is still in the vector, spread across all $d_{model}$ of them."
    misconception: M3
    explain: "Spreading the fields does not survive the permutation argument: there are $d_{model}$ factorial equally valid relabelings and every one produces identical logits, so no arrangement of coordinates is a fact about the model. And a single row serves both senses of `bank`, so it cannot be holding the content of either. What exists is relational geometry, meaningful only inside this model."
  - text: "Both hold because the vectors come from a separate, earlier semantic training stage - word2vec-style - whose own objective put similar words close together. The language model loads that table as fixed input, which is exactly why the table itself does not store anything the model learned."
    misconception: U1-M4
    explain: "There is no separate semantic stage. $E$ is a parameter matrix, randomly initialized and updated by the same backward pass as every other weight, on the language modeling objective alone; freeze it and final loss is measurably worse. A row only gets gradient when its token appears in a batch, which is the mechanism behind glitch tokens."
check: choice
```

## Counting the embedding matrix

<!-- fade: embedding-param-count -->

<!-- refutes: M4 -->

An interface has a size, and $E$'s is easy to count: it is a matrix, so its
parameter count is the product of its dimensions:

$$|E| = V \times d_{model}$$

**Worked example: Llama-2-7B.** Its vocabulary is $V = 32{,}000$ and its width is
$d_{model} = 4096$.

$$|E| = 32{,}000 \times 4096 = 131{,}072{,}000$$

That is 131 million parameters whose entire job is to assign each of 32,000
tokens a starting position. Llama-2 does not tie its input and output matrices
(the output-layer section below explains tying), so a second matrix of the
same size sits at the output end, for 262 million total, against a full model
of about 6.74 billion. The embedding end is roughly 4% of the model, and the
other 96% is transformation.

That ratio sets up a trap, so hold onto it.

**The embedding matrix is a genuine lookup table.** One row per token, indexed
by integer ID, no computation, and it is the only component of the model that
works this way. If your mental model is "the parameters are a compressed
key-value store of facts, and a bigger model has more rows," then notice what
just happened: the one part of the model that literally *is* a table by that
description contains no facts about the world at all. It contains 32,000
positions, and not one of them encodes that Paris is in France.

Everything that could be called knowledge lives in the other 96%, and that 96%
has no rows. It is a stack of matrices that *transform* whatever is handed to
them: no index into it, no key to look up, no row to be missing. u4 locates
where facts appear to concentrate, and u5 explains why the same mechanism
produces both correct generalization and confident hallucination, which are
not two systems, one working and one broken. For now, register the shape of
the thing: 4% table, 96% function.

```beat
id: u1-b6
type: completion
concept: c-embeddings
prompt: |
  GPT-2 small has $V = 50{,}257$, $d_{model} = 768$, and 124 million total
  parameters. Unlike Llama-2, it *ties* its input and output embedding
  matrices - the same $E$ is used at both ends, so it is counted once.

  Step 1. Shape of $E$:            $50{,}257 \times$ ____
  Step 2. Parameters in $E$:       $50{,}257 \times 768 =$ ____
  Step 3. Fraction of the model:   ____ $/\ 124{,}000{,}000 \approx$ ____ %

  Which filling of the three blanks is right, together with the right reason
  that this fraction is so much larger than Llama-2-7B's 2% (input side
  only), given that GPT-2's vocabulary is *larger*?
options:
  - text: |
      Step 1: 768. Step 2: $50{,}257 \times 768 = 38{,}597{,}376$.
      Step 3: $38{,}597{,}376 / 124{,}000{,}000 \approx 31\%$.
      Reason: the table grows linearly in $d_{model}$ while the stack grows
      roughly as $n_{layers} \times d_{model}^2$, and GPT-2 small is narrow
      ($d_{model} = 768$) and shallow (12 layers), so the stack has not yet
      outgrown the table.
    correct: true
    explain: |
      Right. $50{,}257 \times 768 = 38{,}597{,}376$, about 31% of 124 million.
      Width, not vocabulary, is what changed between the two models:
      Llama-2-7B is $d_{model} = 4096$ against GPT-2 small's 768, and the
      transformer stack is quadratic in that number where the embedding
      table is only linear in it.
  - text: |
      Step 1: 768. Step 2: $50{,}257 \times 768 = 38{,}597{,}376$.
      Step 3: $38{,}597{,}376 / 124{,}000{,}000 \approx 31\%$.
      Reason: the fraction is larger because the vocabulary is larger -
      50,257 words is more word knowledge to hold than Llama-2's 32,000, so
      a bigger share of the model goes into holding it.
    misconception: U1-M3
    explain: |
      The three numbers are right and the reason is not. Llama-2 has the
      smaller vocabulary *and* the smaller embedding share, so $V$ cannot be
      what drives the fraction up. $V$ is a budget traded against
      tokens-per-character, not a count of words the model knows; a model
      with $V = 32{,}000$ emits novel identifiers and rare names by
      composing pieces. What changed here is $d_{model}$: 4096 against 768,
      with the stack quadratic in it.
  - text: |
      Step 1: 768. Step 2: $2 \times 38{,}597{,}376 = 77{,}194{,}752$, since
      the model needs one table to look tokens up on the way in and a second
      to look them up on the way out.
      Step 3: $77{,}194{,}752 / 124{,}000{,}000 \approx 62\%$.
    misconception: M4
    explain: |
      Tying means there is exactly one matrix, used at both ends and counted
      once: 38,597,376 parameters, about 31%. The doubling comes from
      picturing $E$ as a key-value store that has to be read in one
      direction and written in the other. It is one table of 50,257
      positions, and it holds no facts in either direction - the facts, such
      as they are, live in the 96% that has no rows at all.
  - text: |
      Step 1: 768. Step 2: $50{,}257 \times 768 = 38{,}597{,}376$, but those
      sit outside the 124 million - the table is pretrained word vectors
      loaded before the run, not parameters the run learns.
      Step 3: $\approx 0\%$ of the trained model.
    misconception: U1-M4
    explain: |
      $E$ is initialized randomly and updated by the same backward pass as
      every other weight, and the 124 million counts it. There is no
      separate semantic training stage to load from: freeze $E$ and final
      loss is measurably worse. Inspect gradients and rows for tokens in the
      batch have nonzero gradient while absent rows have exactly zero -
      which is why very rare tokens end training near their initialization.
check: choice
```

## The output end: hidden state to logits

<!-- fade: hidden-to-logits -->

The 96% is the middle of the model, and u3 and u4 own it; skip over it for
now. Assume it ran, and that at each position it produced a vector
$h \in \mathbb{R}^{d_{model}}$, the hidden state. For generating the next
token, only the hidden state at the *last* position matters.

That vector has to become a distribution over all $V$ tokens, and one matrix
does it. The unembedding matrix is $W_U \in \mathbb{R}^{V \times d_{model}}$,
one row per vocabulary entry, the same layout as $E$, which is what makes
weight tying possible at the end of this section. With $h$ a row vector of
shape $(1, d_{model})$ in this book's convention:

$$z = h\,W_U^T, \qquad z \in \mathbb{R}^{V}$$

The transpose there is load-bearing, not tidying. $h$ carries $d_{model}$ on
its second axis and $W_U$ carries $d_{model}$ on its second axis too, so
nothing contracts until one of them is flipped; $W_U^T$ is $(d_{model}, V)$
and the join works. This is literally what the code does: a PyTorch
`nn.Linear` stores its weight as (out, in) and computes `h @ W.T`.

The result $z$ is the vector of **logits**, one real number per vocabulary
entry, and the componentwise reading is the part that matters:

$$z_j = \langle W_U[j,:],\ h \rangle = \sum_{k=1}^{d_{model}} W_U[j,k] \, h_k$$

$W_U[j,:] \in \mathbb{R}^{d_{model}}$ is row $j$, token $j$'s output direction,
and $\langle \cdot, \cdot \rangle$ is the dot product: multiply the vectors
componentwise, then sum. So the logit for token $j$ is the *similarity between
the hidden state and token $j$'s stored direction*, and the output layer is
$V$ such dot products run in parallel: score the hidden state against every
token's direction, keep all the scores.

That operation is worth pausing on, because you will meet it again in u3 under
a different name. Dot-product-against-a-set-of-stored-directions is the
model's one primitive for "compare this to those." Attention uses it to
compare a query against keys; the output layer uses it to compare a hidden
state against tokens. Same shape, different operands.

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

`" the"` scores highest and `" quantum"` lowest, which is sensible after
`The cat sat on`.

Logits have three properties, and each causes trouble when misremembered.

**They are unnormalized and unbounded.** A logit can be any real number, and
nothing forces them to sum to anything. Turning $z$ into a probability
distribution is softmax's job, which u2 owns; all you need here is that some
function maps $\mathbb{R}^V$ to a distribution.

**Only differences are meaningful.** Add the same constant $c$ to every logit
and the resulting distribution is unchanged, because softmax divides
$e^{z_j + c}$ by $\sum_k e^{z_k + c}$ and the factor $e^c$ cancels top and
bottom. So $[2.0, 1.0, -1.5, -4.0]$ and $[102.0, 101.0, 98.5, 96.0]$ are the
*same* model output. "The logit for token X was 12" is not a statement about
anything; the gap between two logits is.

**The name is about log-odds, and it is nearly literal.** After softmax,
$z_j = \log P_j + \text{const}$: logits are log-probabilities up to that same
additive constant. A gap of $\ln 2 \approx 0.69$ between two logits means one
token is exactly twice as likely as the other, whatever the absolute values
are.

One implementation note closes the loop back to $E$. Many models set
$W_U = E$, so the same matrix that mapped IDs to vectors on the way in maps
hidden states to scores on the way out. This is **weight tying**, and it saves
$V \times d_{model}$ parameters (31% of GPT-2 small, from the last section).
It works because both directions are asking about the same relationship
between tokens and directions in the residual stream: reading in writes token
$j$'s direction into the stream, and scoring out measures how much of token
$j$'s direction is present. Llama-2 and Llama-3 do not tie; GPT-2 does. It is
a tradeoff, not a law.

```beat
id: u1-b7
type: completion
concept: c-logits
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

  Which filling of the four blanks is right, together with the right winning
  token and the right gap between the winner and `" the"`?
options:
  - text: |
      $z_0 = 0$, $z_1 = 3$, $z_2 = 0$, $z_3 = 2$, so $z = [0, 3, 0, 2]$.
      `" a"` wins at 3, and the gap to `" the"` at 0 is 3 logits.
    correct: true
    explain: |
      Right. The same fixed $W_U$ with a new $h$ gives a new ranking: the
      earlier hidden state $h = [2, -1, 0.5]$ put `" the"` first, this one
      puts `" a"` first. The gap of 3 is in logit units, so the probability
      ratio is $e^{3} \approx 20$.
  - text: |
      $z_0 = 0$, $z_1 = 3$, $z_2 = 0$, $z_3 = 2$, so $z = [0, 3, 0, 2]$.
      `" a"` wins, and the gap is about 95%: a score of 3 against 0 means
      `" a"` holds roughly 95% of the probability while `" the"`, at 0, has
      none.
    misconception: M2
    explain: |
      The four fills are right and the gap is not a probability. Logits are
      unnormalized real numbers; only their differences carry information,
      and what a difference gives is a ratio: $e^{3} \approx 20$, not a
      percentage. A logit of 0 is not zero probability either - add 100 to
      every entry of $z$ and the distribution after softmax is unchanged.
      Turning $z$ into probabilities is softmax's job, in u2.
  - text: |
      The scores come off $W_U$ itself: summing each row gives
      $z = [1, 3.5, 0, 1]$, so `" a"` wins by 2.5 over `" the"` - and it
      wins in every context, since $W_U$ is the model's fixed table of token
      scores.
    misconception: M4
    explain: |
      $W_U$ stores one *direction* per token, not one score. The logit is
      $z_j = \langle W_U[j,:],\ h \rangle$, a dot product with the hidden
      state, so the ranking is a function of $h$ and changes from context to
      context - which is precisely what the earlier $h = [2, -1, 0.5]$
      versus this $h = [0, 1, 1]$ demonstrates. Carrying out the dot
      products gives $z = [0, 3, 0, 2]$ and a gap of 3.
check: choice
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

Assemble the pieces in order. Text becomes integers by a frozen,
frequency-derived merge table. Integers become positions in a learned space by
a table lookup. Something in the middle transforms those positions. The result
is dotted against every token's direction to produce $V$ scores, the scores
become a distribution, and one token is drawn from it. Append that token and
run again.

That loop is the whole system, and the bet is that this loop, scaled, produces
something worth calling intelligence: not because anyone designed intelligence
into it, but because prediction has no ceiling and modeling is the only way
up.

Three things to carry forward, each a correction you will need later.

**Nothing here is a database.** 4% of the parameters form a genuine lookup
table containing no facts, and the remaining 96% is a function with no rows to
look up. When u5 explains hallucination, this is why the framing "it retrieved
the wrong row" has no referent.

**Nothing here is portable.** Tokens are integers that mean something only
relative to one merge table, and embeddings are positions that mean something
only relative to one model's downstream layers. The system is internally
coherent and externally meaningless, which is exactly what you would expect
from something whose only constraint was reproducing a corpus.

**The objective is per-token; the computation is not.** Loss is summed over
positions independently, which tempts the conclusion that the model is myopic
and cannot plan. The next unit gives you the machinery to see why that does
not follow: a hidden state at position $t$ is free to encode structure that
only pays off at position $t+40$, and gradient descent rewards it for doing
so, because that is where the loss went down.

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
transpose here or there, the kind of thing you fix when the code throws. Here
is the prediction that fails: under that belief, a paper writing $xW$ and a
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
