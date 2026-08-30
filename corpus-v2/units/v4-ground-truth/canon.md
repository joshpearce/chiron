---
unit: v4
title: "Ground truth: the counterfactual"
concepts:
  - d-counterfactual
  - d-subset-regression
  - d-noise-floor
assumes:
  - d-compute-budget
  - d-eval-noise
---

Every attribution method in the rest of this book is an approximation of one
number, and this unit computes that number exactly. The claim "this source
contributed X" has a definition with no theory in it: remove the source,
retrain, and see how much worse the model gets. That is a counterfactual, and
at the model scale you can afford it is directly measurable. By the end of this
unit you can design a leave-one-source-out sweep, price it, run the arithmetic
that says whether a measured contribution is real or is your own noise, fit a
subset regression that gets per-source values out of a few hundred runs, and
score any attribution method against the truth using the metric the field
actually uses. You will also be able to state precisely what your table does
not mean, which turns out to be the harder half.

A note on numbers before anything else. Every bits-per-byte figure in this unit
is **illustrative**: the magnitudes and the relationships are realistic, the
digits are invented so the arithmetic stays clean enough to do in your head.
The hardware throughputs, the dollar costs, and the published research results
are real and sourced.


## Two numbers and a gap

Here are two models. Same architecture, same 10-million-parameter
configuration, same 200 million training tokens, same learning-rate schedule,
same random seed. They differ in exactly one respect: the corpus.

- Model **F** was trained on the full tagged corpus - all 12 sources from the
  v2 pipeline.
- Model **F\7** was trained on the same corpus with source 7 removed.

Both were then scored on the same held-out set of papers, reserved before
either run and never seen by either model. The score is bits per byte: the
model's total surprise on that held-out text, in bits, divided by the text's
length in UTF-8 bytes. Lower is better. It is used here rather than raw loss
because the denominator - the byte count of a fixed file - was decided before
any tokenizer existed, so the number compares models rather than tokenizers.

| model | corpus | held-out bpb |
|---|---|---|
| F | all 12 sources | **1.043** |
| F\7 | 11 sources, no source 7 | **1.061** |

The gap is 0.018 bits per byte. Removing source 7 made the model measurably
worse at predicting held-out papers.

That gap is source 7's contribution. Not a proxy for it, not a correlate of it,
not an estimate of it. It *is* the thing the phrase "source 7 contributed to
this model" means, computed by doing the only experiment that could settle it.
There is no gradient here, no influence function, no similarity score, no
theory. Two training runs and a subtraction.

```beat
id: v4-b1
type: predict
concept: d-counterfactual
prompt: |
  Before reading on. You are about to send that table to a rightsholder as
  the basis for a payment.

  (1) State, in one sentence, what claim the 0.018 licenses.
  (2) Now attack it. Name at least two distinct things that could be true
      about how those two runs were produced that would make 0.018 the wrong
      number to bill on - not "the model is small" or "bpb is a weird metric",
      but specific properties of the experiment itself.
answer: |
  (1) It licenses: a model of this size and token budget, trained on this
  corpus without source 7, is 0.018 bpb worse on this held-out set than the
  same model trained with it. That is a statement about two runs, and it is
  exact for those two runs.

  Attacks, any two of these count:

  - **One seed.** Both numbers come from a single training run each. Training
    is stochastic - different initialization, different shuffle order - so
    each number is one draw from a distribution. If that distribution is wide
    relative to 0.018, the gap is noise wearing a result's clothes.

  - **The token budget moved.** Removing source 7 removed its tokens too, so
    F\7 trained on less data than F. Some of the 0.018 is "source 7's
    content" and some is "8% fewer tokens", and the experiment as described
    cannot separate them.

  - **Redundancy.** If another source in the corpus carries substantially the
    same material - a mirror, a preprint server, a duplicate cluster the v2
    pipeline recorded - then 0.018 understates source 7 badly, because the
    survivor covered for it. Remove both and the loss might jump by ten times
    this.

  - **One eval set.** The held-out papers define what "worse" means. A source
    that matters enormously for a topic absent from the held-out set scores
    zero here.

  - **The number is not additive.** Nothing yet says the twelve per-source
    gaps add up to anything, so "source 7's share" is undefined even if
    0.018 is exactly right.
rubric: |
  Part (1) must restrict the claim to these two runs, this model
  configuration, and this held-out set - any answer that phrases it as a
  property of source 7 in general has already made the mistake this unit
  spends its length on.
  Part (2) must name at least two distinct mechanisms. Accept any two of:
  single seed / training stochasticity; token budget confounded with content;
  redundancy with another source; eval-set dependence; non-additivity.
  (1) restricted + two valid attacks = pass.
  Attacks that are only "the model is too small to matter" or "bpb is not a
  real benchmark" = fail; both are true and neither is an attack on the
  measurement's validity. The counterfactual is exact at any scale; the
  question is what it is exact ABOUT.
check: llm
```

Now the observation that makes this unit necessary. Run both conditions again
with a different random seed - identical data, identical code, identical
hyperparameters, only the weight initialization and the shuffle order change -
and here is what comes back.

| seed | F (full) | F\7 (no source 7) | gap |
|---|---|---|---|
| 1 | 1.043 | 1.061 | 0.018 |
| 2 | 1.046 | 1.058 | 0.012 |
| 3 | 1.040 | 1.055 | 0.015 |

Source 7's contribution is 0.018. Or 0.012. Or 0.015. The largest is 50%
bigger than the smallest, and nothing about the world changed between the three
rows - only a random number generator's starting value.

So the definition is exact and the measurement is not. The rest of this unit is
about closing that gap: what to average, how many times, and how to say out
loud which entries in your final table are real.

## Contribution is a counterfactual

Hold the two halves apart, because most confusion in this field comes from
sliding between them.

The **definition** of a source's contribution is a difference between two
worlds: the world where you trained on it and the world where you did not. That
definition is complete. It does not need a model of how training works, it does
not need the loss to be differentiable, it does not need any assumption about
what the network learned or where. It needs two runs.

The **measurement** of that difference is an experiment with error bars, like
any other experiment.

Every other method in this book - gradient similarity, influence functions,
TracIn, Shapley, retrieval overlap - is trying to guess the number this
definition gives you, without paying for the second run. That is the whole
field, stated in one sentence. It is worth internalizing early, because it
inverts the usual intuition. The expensive method here is the *simple* one, and
the sophisticated machinery exists entirely to avoid the expense. When you can
afford the expense, the sophistication is optional and becomes something else:
a thing to be checked.

### Why the unit is a source and not a document

The obvious thing to want is per-document contribution. Every paper in the
corpus, its own number. That is the granularity a naive royalty scheme reaches
for, and it is unmeasurable. Here is the arithmetic, which takes about four
lines.

The corpus is roughly 200 million tokens across 12 sources. Take a mid-sized
source at about 8% of the corpus - call it 16 million tokens - and one paper at
about 8,000 tokens, which is 0.004% of the corpus.

Assume, crudely, that the effect on held-out bits per byte of deleting content
is roughly proportional to the fraction of the corpus you deleted. That is not
exact - it ignores quality, redundancy, and diminishing returns - but it is
signed correctly and good enough for an order-of-magnitude check, and it is the
assumption being tested, so it is stated rather than hidden.

Source 7 at 8% of the corpus produced a gap around 0.015 bpb. Scaling down by
the ratio of the shares, $0.004\% / 8\% = 1/2000$, one paper should move
held-out bpb by roughly

$$0.015 / 2000 = 7.5 \times 10^{-6} \text{ bpb}$$

Seven and a half millionths of a bit per byte. Now compare that against the
run-to-run wobble you just watched: the three full-corpus runs came in at 1.043,
1.046, and 1.040, a spread on the order of 0.003 bpb. The signal you want is
**four hundred times smaller than the noise** you have to see it through.

That is not a tooling problem and better code will not fix it. It is a ratio.
Section 3 gives the formula for how many repeat runs it would take; running it
ahead for this case gives about 1.3 million seeds per condition, 2.6 million
training runs, for one paper. At 20 minutes per run on a laptop that is on the
order of a century. The corpus has 25,000 papers in it.

At the source level the same arithmetic gives an answer between one and three
runs per condition. Nothing changed except the size of the thing removed.

This is the single most important design decision in the whole project, and it
is not a mathematical decision. **Choose the attribution unit to match the
payment unit.** A royalty system pays a publisher, a catalogue, a feed, an
archive - an entity that can hold a contract and receive money. It does not pay
a document. Once the unit is the thing you were going to pay anyway, the
measurement becomes affordable, and the affordability is not a coincidence:
payable entities are large, and large things move the loss.

```beat
id: v4-b2
type: self-explain
concept: d-counterfactual
prompt: |
  A colleague accepts that per-document leave-one-out is unmeasurable but
  proposes a fix: "Run the per-document experiment on a 1,000-document
  subcorpus instead of 25,000. Each document is now 0.1% of the corpus
  rather than 0.004%, so the effect is 25 times bigger and the noise is the
  same. Problem solved."

  In your own words: what is right about this, what is wrong about it, and
  what would you actually have measured if you did it?
answer: |
  What is right: the ratio argument is the correct thing to attack, and
  shrinking the corpus really does raise each document's share, which really
  does raise the per-document effect size. The colleague has understood that
  the obstacle is signal-to-noise and not instrumentation.

  What is wrong: the effect grows by 25x, so the required seed count - which
  goes as the inverse SQUARE of the effect size - falls by 625x. From roughly
  1.3 million seeds per condition that lands around 2,000 seeds per condition,
  per document, and there are 1,000 documents. It is 625 times more feasible
  and still not feasible. Shrinking the corpus buys a factor, and the gap is
  orders of magnitude.

  The deeper problem is what the number would mean. In a 1,000-document
  corpus each document carries a tenth of a percent of everything the model
  knows, so its measured contribution is a fact about a model that no longer
  resembles the one being billed for. You would have measured document
  influence in a regime where documents are individually load-bearing, which
  is exactly the regime real pretraining is not in. The measurement got easier
  by changing the thing measured.

  What to do instead: keep the corpus real and raise the unit. A source is
  large by construction, so it moves the loss by construction, and it is also
  the entity that receives the payment.
rubric: |
  Must contain: (1) acknowledgement that the effect-size argument is correct
  as far as it goes, (2) the quantitative point that required seeds scale as
  1/effect^2 so a 25x effect is a 625x reduction and still leaves an
  infeasible number, (3) the validity point - a 1,000-document corpus is a
  different model, so the number does not transfer to the corpus you are
  billing for.
  (2) and (3) = pass. (1) alone or (3) alone = fail.
  An answer that says the plan works = fail, and diagnose the missing square:
  the seed formula has the effect size squared in the denominator, so effect
  size is worth more than any other lever, but "worth more" is not "worth
  enough".
  An answer that rejects the plan only because "small corpora are not
  realistic" without the arithmetic = partial, do not pass; the learner has
  the conclusion without the instrument that produced it.
check: llm
```

### The two symbols this unit needs

Write $v(S)$ for the held-out bits per byte of a model trained on the set of
sources $S$. The letter $v$ is for value, borrowed from the game theory that
v5 will need; $S$ is a subset of the $K$ sources in your corpus. So $v(\text{all
12})$ is the full model's score and $v(\text{all except } 7)$ is the score with
source 7 removed. One number per training run.

Beware the sign, once, and then it will not bother you again. Bits per byte is
a cost: lower is better. So a helpful source makes $v$ go *down*, and removing
it makes $v$ go *up*. Contribution is therefore defined with the removal first:

$$\Delta_i = v(\text{all except } i) - v(\text{all})$$

where $\Delta_i$ is source $i$'s leave-one-out contribution, in bits per byte.
$\Delta_i > 0$ means the source helped. For source 7 on seed 1,
$\Delta_7 = 1.061 - 1.043 = 0.018$.

That is the entire notational apparatus of the first half of this unit.

## Designing the sweep

A design is a list of runs, and there are exactly three decisions: which
conditions, what to hold fixed, and how many seeds.

### Conditions

For $K$ sources you need $K + 1$ conditions: one model trained on everything,
and $K$ models each missing one source. At $K = 12$ that is 13 conditions. Every
$\Delta_i$ is then a subtraction between condition $i$ and the full model, and
the full model's cost is amortized across all twelve differences.

### What to hold fixed: the protocol choice

Remove a source and you have removed its tokens. You now have two coherent
protocols and they measure different things.

**Fixed corpus.** Train the leave-one-out model on whatever is left: 11 sources,
about 8% fewer tokens, correspondingly fewer optimizer steps. The measured
$\Delta_i$ then answers "what happens to my model if this source disappears
from the world," which bundles the source's content together with the loss of
its volume.

**Fixed token budget.** Train the leave-one-out model on the same total number
of tokens as the full model, backfilling the missing 8% by repeating what
remains. The measured $\Delta_i$ then answers "what does this source's content
contribute at equal compute," with volume held constant.

Neither is wrong. They are different questions and the difference is not
small - a source that is 20% of your corpus will look far more valuable under
fixed corpus than under fixed budget, because two thirds of what you measured
was the missing fifth of the training run.

The recommendation for this project is **fixed token budget**, for two reasons.
First, it is the question a payment answers: a rightsholder is being paid for
content, and if they withdrew, a lab would backfill rather than shrink its run.
Second, the backfill is nearly free. Repeating a fixed corpus for up to about
four epochs costs almost nothing against fresh tokens, with returns diminishing
out to roughly sixteen; a corpus small enough to be tagged by hand is already
being repeated, so adding an eighth of an epoch to the survivors changes very
little. The protocol is affordable precisely because the data-constrained
regime is forgiving.

Whichever you pick: **write it down before the sweep, and report it beside the
table.** A contribution number without its protocol is not interpretable, and
the difference between the two protocols is larger than most of the effects
you are trying to detect.

```beat
id: v4-b3
type: predict
concept: d-counterfactual
prompt: |
  Your corpus has 12 sources. Source 3 is a large systems venue at 22% of all
  tokens. Source 11 is a small workshop at 1.5% of all tokens.

  You run the sweep twice, once under each protocol - fixed corpus (train on
  what is left) and fixed token budget (backfill by repetition to the same
  token count).

  Before reading on: predict how $\Delta_3$ and $\Delta_{11}$ each change
  between the two protocols. Which source's number moves more, in absolute
  terms and in relative terms, and why?
answer: |
  Both fixed-corpus numbers are larger, because in the fixed-corpus protocol
  the leave-one-out model is handicapped twice: it lacks the content AND it
  trained on fewer tokens. Backfilling removes the second handicap, so every
  contribution shrinks.

  Source 3 moves more in absolute terms, by a lot. Removing 22% of the tokens
  and not replacing them is a substantial cut to the training run itself, and
  that shortfall shows up in held-out bpb regardless of what the removed text
  said. Under fixed budget that entire component disappears.

  Source 11 moves less in absolute terms - a 1.5% token shortfall is nearly
  nothing - so almost all of its fixed-corpus number was already content.

  In relative terms the same ordering holds: source 3's number can easily be
  cut by more than half, while source 11's barely moves. So the two protocols
  do not merely shift every number by a constant; they reorder the table,
  systematically favoring large sources under fixed corpus.

  That is the reason the protocol has to be pre-registered. A rightsholder who
  learns that their 22% share was scored under fixed corpus has a real
  argument that they were paid for volume rather than content, and the
  argument is correct.
rubric: |
  Must contain: (1) fixed-corpus contributions are systematically larger
  because volume loss is bundled in, (2) the large source moves more in
  absolute terms because its token shortfall is larger, (3) the crucial
  consequence - the two protocols do not just shift the table, they REORDER
  it in favor of large sources.
  (1) and (3) = pass. (1) and (2) without (3) = partial, do not pass; a
  constant offset would be harmless and the whole point is that it is not
  constant.
  An answer predicting the protocol makes no difference = fail; that is the
  belief that the leave-one-out delta measures content by definition, when it
  measures whatever differed between the two runs.
check: llm
```

### How many seeds, and the arithmetic that decides

<!-- fade: v4-loo-contribution -->

Train each condition $n$ times with different seeds. Average within a condition
first, then subtract. Here is why, derived from scratch, because it is four
lines and every error bar in the rest of this book comes out of it.

A single run's held-out bpb is a random quantity. Call the standard deviation of
that quantity across seeds $\sigma$ - the *seed noise*, a property of your model
configuration, measured once by training the same setup several times and
looking at the spread. In the table above, all three full-corpus runs sat within
0.003 of their mean, so $\sigma = 0.003$ bpb for this configuration.

Two facts about averaging, both of which follow from variances adding rather
than standard deviations adding. Independent errors are as likely to cancel as
to reinforce, so the *typical squared* error of a sum is the sum of the typical
squared errors - which is to say variance is what adds, and variance is the
square of the standard deviation.

**Fact one: averaging $n$ runs.** The mean of $n$ independent runs has variance
$\sigma^2 / n$, because you add $n$ variances of $\sigma^2$ and then divide by
$n^2$ from the averaging. Its standard deviation is $\sigma / \sqrt{n}$.

**Fact two: subtracting two independent means.** Subtracting is as bad as
adding, as far as errors go - the two errors are independent, so they do not
cancel. Variances add again:

$$\text{Var}(\Delta) = \frac{\sigma^2}{n} + \frac{\sigma^2}{n} = \frac{2\sigma^2}{n}
\qquad\Longrightarrow\qquad
\text{SE}_{\Delta} = \sigma\sqrt{\frac{2}{n}}$$

where $\text{SE}_{\Delta}$ is the standard error of the measured contribution:
roughly how far your $\Delta_i$ sits from the value infinite seeds would give.

Sanity-check the formula at both ends. At $n = 1$ it gives $\sigma\sqrt{2}$,
*worse* than one run's own noise, which is right: a difference of two noisy
numbers is noisier than either. At $n = 2$ it gives exactly $\sigma$. You need
four runs before the error bar on a difference drops to a single run's spread.

Then the verdict:

$$z = \frac{\Delta_i}{\text{SE}_{\Delta}}$$

A $z$ of 2 means the measured contribution is twice its own error bar. Below
that, you have not measured the source; you have measured your noise.

**Worked, on the source 7 table.** Three seeds per condition, $\sigma = 0.003$.

| seed | $v(\text{all})$ | $v(\text{all} \setminus 7)$ |
|---|---|---|
| 1 | 1.043 | 1.061 |
| 2 | 1.046 | 1.058 |
| 3 | 1.040 | 1.055 |
| **mean** | **1.043** | **1.058** |

Average first:

$$\bar v(\text{all}) = \frac{1.043 + 1.046 + 1.040}{3} = \frac{3.129}{3} = 1.043$$
$$\bar v(\text{all} \setminus 7) = \frac{1.061 + 1.058 + 1.055}{3} = \frac{3.174}{3} = 1.058$$

Then subtract:

$$\Delta_7 = 1.058 - 1.043 = 0.015 \text{ bpb}$$

Then the error bar:

$$\text{SE}_{\Delta} = 0.003 \times \sqrt{\tfrac{2}{3}} = 0.003 \times 0.816 = 0.00245$$

$$z = \frac{0.015}{0.00245} = 6.1$$

Source 7's contribution is $0.015 \pm 0.0025$ bpb, six standard errors from
zero. That is a result. Report it as the pair, never as the bare 0.015.

**The order matters, and here is the trap.** It is tempting to compute the three
per-seed gaps - 0.018, 0.012, 0.015 - average them to 0.015 (the same answer,
correctly), and then take the standard deviation of those three gaps, 0.003, as
the error bar. That number is wrong by a factor of $\sqrt{n}$. The spread of
individual differences is $\sigma\sqrt{2} = 0.0042$; what you want is the
spread of their *mean*, which is smaller by $\sqrt{3}$. Reporting the spread of
the differences overstates your uncertainty and buries real results. The mirror
error - reporting one condition's own standard error, $\sigma/\sqrt{n} =
0.0017$ - understates it by $\sqrt{2}$ and manufactures results that are not
there. Average within condition, subtract, then apply $\sigma\sqrt{2/n}$.

**How many seeds do you need?** Rearrange $z \geq 2$:

$$n \geq \frac{8\sigma^2}{\Delta^2}$$

For $\sigma = 0.003$ and a source with a true $\Delta$ of 0.015, that gives
$n \geq 8(9 \times 10^{-6}) / (2.25 \times 10^{-4}) = 0.32$, so one seed would
nominally do. Run three anyway, for two reasons that have nothing to do with
this source: you cannot estimate $\sigma$ at all without repeats, and half the
sources in a real table sit near the floor where three is not enough either.
Three seeds per condition is the working minimum, and the error budget in the
attribution literature is unambiguous that if you are choosing between more
seeds and more anything else, seeds win.

```beat
id: v4-b4
type: completion
concept: d-noise-floor
# variants: give delta and z, blank sigma; or blank only the seed count in
# step 5; or supply a four-seed table and blank the two means.
prompt: |
  A different source, source 9, from the same sweep. Seed noise for this
  configuration is $\sigma = 0.003$ bpb, measured earlier.

  | seed | $v(\text{all})$ | $v(\text{all} \setminus 9)$ |
  |---|---|---|
  | 1 | 1.043 | 1.040 |
  | 2 | 1.046 | 1.037 |
  | 3 | 1.040 | 1.034 |

  Step 1. $\bar v(\text{all}) = 3.129 / 3 =$ ____
  Step 2. $\bar v(\text{all} \setminus 9) = $ ____ $/\ 3 =$ ____
  Step 3. $\Delta_9 =$ ____
  Step 4. $\text{SE}_{\Delta} = 0.003 \times \sqrt{2/3} =$ ____
  Step 5. $z = \Delta_9 / \text{SE}_{\Delta} =$ ____

  Fill the blanks. Then answer in two sentences: is this a result, and what
  does its sign say about source 9?
answer: |
  Step 1: 1.043
  Step 2: 3.111 / 3 = 1.037
  Step 3: 1.037 - 1.043 = -0.006 bpb
  Step 4: 0.00245
  Step 5: -0.006 / 0.00245 = -2.45

  It is a result. |z| = 2.45 clears the bar, so the negative sign is not
  noise: removing source 9 made the model BETTER by 0.006 bpb, six
  thousandths of a bit per byte, with an error bar of 0.0025.

  What the sign says: source 9's tokens, at this fixed token budget, were
  worth less than the tokens that replaced them. That is a statement about
  this corpus and this held-out set, not a verdict on the source's quality -
  the leading candidates are that source 9 duplicates material another source
  already covers, or that its subject matter is absent from the held-out
  papers. Either way it is displacing tokens that would otherwise have taught
  something the evaluation rewards.
rubric: |
  Required, exactly: step 1 = 1.043; step 2 = 3.111 and 1.037; step 3 =
  -0.006; step 4 = 0.00245 (accept 0.0024-0.0025); step 5 = -2.45 (accept
  -2.4 to -2.5).
  The verdict must be that |z| clears 2, so the negative contribution is
  real rather than noise.
  The interpretation must NOT be "source 9 is bad data". Accept redundancy
  with another source, mismatch with the held-out set, or token displacement
  at fixed budget. Any of the three = pass.
  All five numbers plus a non-"bad data" interpretation = pass.
  Reading the negative delta as an arithmetic mistake and flipping the sign =
  fail; negative contributions are ordinary and the table must be able to
  express them.
  Concluding "source 9 should be dropped from the corpus" = pass on the
  arithmetic, flag V4-M3; at fixed budget a negative delta is a statement
  about the marginal token, and if the duplicate partner were also removed
  the sign could reverse.
check: llm
```

## What the sweep costs

The reason this unit exists at all is that the sweep is cheap, and the reason it
is cheap is one line of arithmetic from v3, restated here so you do not have to
go looking.

A training run performs about six floating-point operations per parameter per
token: two in the forward pass (one multiply, one add) and four in the backward
pass, which has to produce both the gradient for the parameter and the gradient
for the activation feeding it. So

$$C \approx 6ND$$

where $C$ is total floating-point operations for the run, $N$ is the parameter
count, and $D$ is the number of training tokens. Divide $C$ by your hardware's
sustained throughput to get seconds; multiply seconds by the hourly rate to get
dollars. No other formula is involved.

**The sweep, priced.** Three quantities set the bill: $K$ sources, $n$ seeds per
condition, and $M$ random-subset runs (which section 6 explains and which you
should treat as a line item for now).

$$\text{runs} = \underbrace{(K + 1) \times n}_{\text{leave-one-out}} + \underbrace{M}_{\text{subsets}}$$

Take $K = 12$, $n = 3$, $M = 300$. That is $13 \times 3 + 300 = 339$ runs.

Now price one run. At $N = 3 \times 10^7$ parameters and $D = 5 \times 10^8$
tokens:

$$C = 6 \times (3 \times 10^7) \times (5 \times 10^8) = 9 \times 10^{16} \text{ FLOPs}$$

One H100 sustains roughly $4 \times 10^{14}$ FLOP/s on this kind of workload:

$$t = \frac{9 \times 10^{16}}{4 \times 10^{14}} = 225 \text{ s}$$

Under four minutes per run. So the complete sweep is

$$339 \times 225 \text{ s} = 76{,}275 \text{ s} = 21.2 \text{ GPU-hours}$$

At $1.50 to $4.00 per GPU-hour as of mid-2026, that is **$32 to $85**. The
published envelope for this experiment at a larger source count - 20 to 50
sources, 500 subset models, about 550 runs total - is roughly 37 GPU-hours and
**$55 to $150**. Both land in the same place: the complete counterfactual
ground truth for a tagged corpus costs less than a plane ticket.

**And on the laptop, nothing at all.** A MacBook training under MLX sustains
around $10^{13}$ FLOP/s, roughly one three-hundredth of a rented node. Shrink
to $N = 10^7$ and $D = 2 \times 10^8$:

$$C = 6 \times 10^7 \times (2 \times 10^8) = 1.2 \times 10^{16}
\qquad t = \frac{1.2 \times 10^{16}}{10^{13}} = 1{,}200 \text{ s} = 20 \text{ min}$$

Twenty minutes per run, with no invoice. The 39-run leave-one-out portion of the
sweep is 13 hours - two evenings, or one if you trim $D$ and accept a slightly
larger $\sigma$. The 300 subset runs are what you rent a GPU-day for.

Sit with the shape of this, because it is the strategic core of the whole book.
Cost is *linear* in $N$. The attribution literature treats retraining-based
ground truth as the unaffordable thing everybody approximates - and it is
unaffordable at 8 billion parameters. Shrinking the model by a factor of two
hundred shrinks the entire research program by a factor of two hundred, and the
counterfactual becomes an overnight job. Nothing about the *definition* of
contribution got easier. The invoice did.

```beat
id: v4-b5
type: compute
concept: d-counterfactual
prompt: |
  Budget a complete ground-truth sweep on one rented H100 sustaining
  $4 \times 10^{14}$ FLOP/s.

  Corpus: $K = 20$ sources. Design: $n = 3$ seeds on each of the $K + 1$
  leave-one-out conditions, plus $M = 200$ single-seed random-subset runs.
  Each run is $N = 2.5 \times 10^7$ parameters on $D = 3 \times 10^8$ tokens.

  How many GPU-hours is the complete sweep? Give one number to two decimal
  places.
answer: 8.22
check: numeric(0.05)
```

## The noise floor is a result

<!-- refutes: D16 -->

**You probably think a source's measured contribution is a stable property of
that source.** Measure it once, carefully, and you have a number. Better
instrumentation gives a more precise number, the way a better scale gives a more
precise weight. Under this belief a contribution table is a ledger, the
engineering task is to compute it accurately, and the error bars are a
scientific courtesy you could drop from the invoice.

**Here is the prediction that fails, and you have already seen it fail.** If
contribution were a property of the source, the three rows of the opening table
would agree. Source 7's gap came in at 0.018, 0.012, and 0.015 across three
seeds that differed in nothing but a random number generator's starting value.
The largest is 50% larger than the smallest. Push the same experiment down to
the per-document level and the situation degrades from imprecise to
undecidable - per-sample membership decisions in the verification literature
flip like coin tosses under training randomness alone. And of everything that
perturbs an influence measurement, the *order* in which training data arrives
introduces the largest variation: an ordering nobody chose deliberately and
almost nobody records.

**Here is why the wrong model is appealing.** Every instinct from deterministic
systems. Same inputs, same outputs; if two runs disagree, something is broken
and should be fixed. And the apparatus radiates determinism from every surface:
loss curves are smooth, final training losses across seeds agree to three
decimal places, checkpoints are byte-comparable. What is noisy is not the models
but the *differences between* models - and a difference between two nearly equal
noisy numbers is exactly where variance concentrates. The full-corpus runs
agreed to within 0.3%. Their differences from the leave-one-out runs disagreed
by 50%.

**Here is what is actually true.** A contribution is a random variable. You do
not measure it, you estimate it, and the estimate carries an error bar that is
frequently larger than the estimate itself. Three consequences follow, and they
are operational rather than philosophical.

**First: every number in the table ships with $\sigma$.** Not as a footnote. The
deliverable of this unit is not a column of contributions, it is a column of
contributions and a column of standard errors, and a reader who cannot see the
second column cannot use the first.

**Second: a contribution inside the noise is a finding, not a gap.** Here is a
six-row excerpt of the finished table, at $K = 12$, $n = 3$, $\sigma = 0.003$,
so $\text{SE}_{\Delta} = 0.00245$ for every row.

| source | $\Delta_i$ (bpb) | $z$ | reported as |
|---|---|---|---|
| S3 - large systems venue | 0.031 | 12.7 | 0.031 $\pm$ 0.0025 |
| S7 - ML venue | 0.015 | 6.1 | 0.015 $\pm$ 0.0025 |
| S1 - theory venue | 0.007 | 2.9 | 0.007 $\pm$ 0.0025 |
| S11 - small workshop | 0.001 | 0.4 | **below the noise floor** |
| S9 - mirror of S4 | -0.006 | -2.4 | -0.006 $\pm$ 0.0025 |
| S12 - biology preprints | 0.000 | 0.0 | **below the noise floor** |

S11 and S12 do not get an asterisk and a smaller number. They get a sentence:
*at this model scale, this token budget, and three seeds, these sources'
contributions are not distinguishable from zero, and the smallest contribution
this sweep could have resolved is about 0.005 bpb.* That sentence is more
valuable than any number you could have put in its place, because it is the one
thing a person building a payment scheme on top of your table has to know.

**Third: the floor is itself the product.** There is a proven result waiting in
v5 that turns this from methodology into strategy. When an attribution
estimator's signal-to-noise ratio falls below a threshold, the welfare-optimal
payment contract collapses to a flat fee - no better estimator changes that
conclusion, only more signal does. Which side of that threshold your corpus sits
on is an empirical question with two publishable answers, and measuring your own
noise floor is how you answer it.

This one is pitch-fatal if you get it backwards. Walking into a room with a pie
chart and no error bars, against someone who has read the literature, ends the
meeting - the industry's convergence on flat fees and metered access is
consistent with the signal-to-noise bar not being cleared, and a founder who has
not measured their own floor has not established that they are on the other
side of it. Walking in with "here is our measured contribution table, here is
our noise floor, here are the four sources that clear it and the eight that do
not, and here is what that implies about contract design" is a stronger position
than any pie chart, including a true one.

```beat
id: v4-b6
type: self-explain
concept: d-noise-floor
prompt: |
  Your table has eight of twelve sources below the noise floor. A teammate
  proposes three fixes, in order of preference:

  (a) Train a bigger model - 300M parameters instead of 10M - since bigger
      models are more stable.
  (b) Skip the extra seeds and spend the same compute on 600 random-subset
      runs instead of 300, since more runs means more averaging.
  (c) Report the point estimates without error bars and add a methodology
      note, since the ordering of the sources is probably right even if the
      individual numbers are not.

  Take each in turn. State whether it reduces the number of sources below the
  floor, and why or why not.
answer: |
  (a) Bigger model: does not fix it, and misdiagnoses the problem. Seed noise
  is not an artifact of small models that anneals away with scale -
  contribution is a random variable at every scale, and a 300M model has its
  own sigma. What a bigger model does change is the invoice: cost is linear in
  N, so a 30x larger model makes every seed 30x more expensive, which means
  you can afford 30x FEWER seeds. Since seeds are the only thing that shrinks
  SE_delta, scaling up moves you AWAY from resolving the floor. This is the
  most expensive available way to make the problem worse.

  (b) More subset runs instead of more seeds: does not fix the leave-one-out
  table, though it is not a stupid idea. Each Delta_i is a difference between
  exactly two condition means and nothing else enters it; no quantity of
  subset runs appears anywhere in SE_delta = sigma*sqrt(2/n). Only n moves it.
  What the subset runs DO give is a different estimator with its own, tighter
  error bar - the regression coefficient pools across all M runs - but it
  estimates an average marginal effect across many coalitions, not the
  leave-one-out counterfactual, and the two coincide only if contributions are
  additive. You may report both. You may not use one to tighten the other.

  (c) Drop the error bars: does not fix it and is the only one of the three
  that is dishonest. The claim smuggled in is that the ORDERING survives even
  if the magnitudes do not, and for the eight sources below the floor the
  ordering is precisely what does not survive - their relative positions are
  determined by noise, so a reseeded sweep permutes them. Publishing the order
  as though it were information is the failure mode that ends a pitch.

  What actually works: more seeds on the conditions that matter. The
  requirement is n >= 8*sigma^2/Delta^2, so to resolve a 0.002 bpb effect at
  sigma = 0.003 needs n >= 18 seeds per condition. At 225 seconds per run that
  is about 2.3 GPU-hours for one source. Buy seeds.
rubric: |
  Must contain, one judgment per option:
  (a) rejected, with the reason that noise is intrinsic rather than a
  small-model artifact, and ideally the cost inversion - bigger models buy
  fewer seeds, so scaling up makes it worse.
  (b) rejected FOR THE LOO TABLE specifically, with the reason that SE_delta
  depends only on sigma and n. Credit an answer that also notes subset
  regression is a separate estimator with its own tighter error bar; do not
  require it.
  (c) rejected, with the reason that ordering below the floor is exactly what
  is unstable.
  All three correctly rejected with (a)'s and (b)'s reasons = pass.
  Accepting (a) = fail, diagnosing D16 in its most durable form: the belief
  that noise is a defect that better engineering removes.
  Accepting (b) = fail, diagnosing V4-M2.
  Accepting (c) = fail; this is the one that is not a technical error.
  An answer that rejects all three but offers no remedy = partial, do not
  pass; the point of the seed formula is that the remedy is purchasable.
check: llm
```

## Subset regression

The leave-one-out sweep answers one question per source: what happens if this
source alone is missing from a corpus that otherwise has everything. That is a
narrow question. It says nothing about what happens when several sources are
missing at once, which is the situation every payment split actually has to
reason about, and it spends $K + 1$ conditions to produce $K$ numbers, which is
not an efficient use of runs.

Here is a different design that fixes both, and it is ordinary linear
regression - the thing you would already do if nobody had told you this was a
research problem.

### The construction

Train $M$ models, each on a *random subset* of the sources. Record, for each
run, which sources were in and the held-out bpb that came out. You now have a
table with $K$ yes/no columns and one number column, $M$ rows deep. Fit a line.

Concretely: for a run trained on subset $S$, write $x_i = 1$ if source $i$ was
included and $x_i = 0$ if it was not. Model the outcome as

$$v(S) \approx \beta_0 + \beta_1 x_1 + \beta_2 x_2 + \cdots + \beta_K x_K$$

where $v(S)$ is that run's held-out bits per byte, $\beta_0$ is the intercept
(the predicted bpb of a model trained on none of them, which is a mathematical
convenience rather than a run you would do), and $\beta_i$ is the change in bpb
from having source $i$ present. "Fit" means: choose the $\beta$ values that make
the right-hand side reproduce the observed $v(S)$ as closely as possible across
all $M$ runs.

Mind the sign one more time. bpb is a cost, so a helpful source has a
*negative* $\beta_i$. Define the regression contribution to match the
leave-one-out sign convention:

$$c_i = -\beta_i$$

so that $c_i > 0$ means source $i$ helped, exactly as $\Delta_i > 0$ does.

That is the whole construction. It is called a *datamodel* in the literature -
train thousands of models on random subsets, fit a linear surrogate from
membership to outcome - and the original work released four million trained
networks to make the point. Nothing about it requires you to know anything
about the models being fit. It treats each training run as a black box that
turns a set of ingredients into a score, and asks which ingredients predict the
score.

### Worked, small enough to solve by looking at it

<!-- fade: v4-subset-regression -->

Four sources, A B C D. Eight runs. The subsets were not drawn at random for
this example; they were chosen so each source appears in exactly four of the
eight runs, and so that for any two sources, all four in/out combinations appear
twice. That balance is what makes the answer readable off the table.

| run | A | B | C | D | held-out bpb |
|---|---|---|---|---|---|
| 1 | 0 | 0 | 0 | 1 | 1.100 |
| 2 | 1 | 0 | 0 | 0 | 1.070 |
| 3 | 0 | 1 | 0 | 0 | 1.080 |
| 4 | 1 | 1 | 0 | 1 | 1.050 |
| 5 | 0 | 0 | 1 | 0 | 1.090 |
| 6 | 1 | 0 | 1 | 1 | 1.060 |
| 7 | 0 | 1 | 1 | 1 | 1.070 |
| 8 | 1 | 1 | 1 | 0 | 1.040 |

**Read off $\beta_A$.** Average the bpb of the four runs where A was in, average
the four where A was out, subtract.

In (runs 2, 4, 6, 8): $\dfrac{1.070 + 1.050 + 1.060 + 1.040}{4} = \dfrac{4.220}{4} = 1.055$

Out (runs 1, 3, 5, 7): $\dfrac{1.100 + 1.080 + 1.090 + 1.070}{4} = \dfrac{4.340}{4} = 1.085$

$$\beta_A = 1.055 - 1.085 = -0.030 \qquad c_A = 0.030 \text{ bpb}$$

**And the rest, the same way.**

- B in (3, 4, 7, 8): $4.240/4 = 1.060$. B out (1, 2, 5, 6): $4.320/4 = 1.080$.
  $\beta_B = -0.020$, $c_B = 0.020$.
- C in (5, 6, 7, 8): $4.260/4 = 1.065$. C out (1, 2, 3, 4): $4.300/4 = 1.075$.
  $\beta_C = -0.010$, $c_C = 0.010$.
- D in (1, 4, 6, 7): $4.280/4 = 1.070$. D out (2, 3, 5, 8): $4.280/4 = 1.070$.
  $\beta_D = 0.000$, $c_D = 0.000$.

**Why the subtraction is the regression.** Because of the balance, the four runs
with A in and the four with A out are matched on everything else: B appears
twice in each group, C twice in each, D twice in each. So when you subtract the
two averages, every other source's effect cancels exactly, and what remains is
A's. That is what least squares does in general - it is just that in general the
columns are tangled and you need a solver to untangle them. With a balanced
design the untangling is already done and the fit is four subtractions. No
matrix algebra appears anywhere in this section and none is hiding.

The intercept falls out too. Every source appears in half the runs, so the
overall average bpb is $\beta_0$ plus half of each $\beta$:
$1.070 = \beta_0 + \tfrac{1}{2}(-0.030 - 0.020 - 0.010 + 0.000) = \beta_0 -
0.030$, giving $\beta_0 = 1.100$.

```beat
id: v4-b7
type: completion
concept: d-subset-regression
# variants: blank the "out" average instead of the "in" average; or blank two
# different sources' coefficients; or give all four c_i and blank beta_0.
prompt: |
  Same balanced design - each source in four of eight runs - different corpus,
  so different outcomes.

  | run | A | B | C | D | held-out bpb |
  |---|---|---|---|---|---|
  | 1 | 0 | 0 | 0 | 1 | 1.210 |
  | 2 | 1 | 0 | 0 | 0 | 1.160 |
  | 3 | 0 | 1 | 0 | 0 | 1.170 |
  | 4 | 1 | 1 | 0 | 1 | 1.140 |
  | 5 | 0 | 0 | 1 | 0 | 1.200 |
  | 6 | 1 | 0 | 1 | 1 | 1.170 |
  | 7 | 0 | 1 | 1 | 1 | 1.180 |
  | 8 | 1 | 1 | 1 | 0 | 1.130 |

  Step 1. A in (runs 2,4,6,8): $(1.160 + 1.140 + 1.170 + 1.130)/4 =$ ____
  Step 2. A out (runs 1,3,5,7): $(1.210 + 1.170 + 1.200 + 1.180)/4 =$ ____
  Step 3. $\beta_A =$ ____ , so $c_A =$ ____
  Step 4. C in (runs 5,6,7,8) $=$ ____ ; C out (runs 1,2,3,4) $=$ ____ ;
          $c_C =$ ____
  Step 5. D in (runs 1,4,6,7) $=$ ____ ; D out (runs 2,3,5,8) $=$ ____ ;
          $c_D =$ ____

  Fill the blanks. Then state in one sentence what is unusual about $c_D$ and
  give the two explanations that a leave-one-out sweep alone could not
  distinguish between.
answer: |
  Step 1: 4.600 / 4 = 1.150
  Step 2: 4.760 / 4 = 1.190
  Step 3: beta_A = 1.150 - 1.190 = -0.040, so c_A = 0.040
  Step 4: C in = 4.680 / 4 = 1.170; C out = 4.680 / 4 = 1.170; beta_C = 0.000,
          so c_C = 0.000
  Step 5: D in = 4.700 / 4 = 1.175; D out = 4.660 / 4 = 1.165;
          beta_D = +0.010, so c_D = -0.010

  What is unusual: c_D is negative. Averaged across this design, including
  source D made the model WORSE by 0.010 bpb, and unlike C - which is simply
  worth nothing here - D is actively costing something.

  The two explanations a leave-one-out sweep cannot separate: (i) D's content
  is off-distribution for the held-out set, so training on it moves the model
  away from what is being scored; or (ii) at a fixed token budget D's tokens
  displaced tokens from A, B, or C that would have taught something the
  evaluation rewards, which is what redundancy looks like from the outside.
  Distinguishing them needs the interaction terms - does D's coefficient
  change depending on which other sources are present - and that is what v5's
  coalition machinery is for.
rubric: |
  Required, exactly: step 1 = 1.150; step 2 = 1.190; step 3 = -0.040 and
  0.040; step 4 = 1.170, 1.170, 0.000; step 5 = 1.175, 1.165, -0.010.
  Sign discipline is the graded part: c = -beta must be applied, so c_A is
  POSITIVE 0.040 and c_D is NEGATIVE 0.010. Reversing either sign = fail.
  The final sentence must identify c_D as negative and offer at least one of
  {off-distribution content, token displacement / redundancy at fixed budget}.
  All numbers with correct signs plus one valid explanation = pass.
  An answer that reads c_D = -0.010 as "source D contains bad or corrupted
  data" as the only explanation = partial, do not pass, and flag V4-M3.
  An answer that reads c_C = 0 as "source C was not used" = fail; a zero
  coefficient means the model was no better with it than without it on this
  evaluation, which is a measurement about the held-out set, not about
  whether the tokens went through the optimizer.
check: llm
```

### Why this design beats the leave-one-out sweep, and where it does not

Count the error bars. In the leave-one-out sweep, $\Delta_i$ is a difference
between two condition means of $n$ runs each, so
$\text{SE}_{\Delta} = \sigma\sqrt{2/n}$ and nothing else averages it. In the
subset regression, $\beta_i$ is a difference between two averages of about
$M/2$ runs each, so

$$\text{SE}_{\beta} = \sigma\sqrt{\frac{2}{M/2}} = \frac{2\sigma}{\sqrt{M}}$$

At $M = 300$ and $\sigma = 0.003$, that is $2(0.003)/17.3 = 0.00035$ bpb - seven
times tighter than the leave-one-out sweep's 0.00245, from 300 runs against 39.
To match it with leave-one-out you would need $n$ around 150 seeds per
condition, which is 1,950 runs. The subset design is roughly six times more
run-efficient per unit of precision, and it gets there because every run informs
every coefficient, rather than each run informing one.

Two things it does not buy, both important.

**It does not estimate the same quantity.** $\Delta_i$ is the marginal
contribution of source $i$ to the *full* corpus, with all eleven others present.
$c_i$ is an average marginal contribution across the whole range of subsets the
design sampled, most of which contain about half the sources. Those coincide
only if a source's contribution does not depend on what else is in the corpus.
That assumption has a name - additivity - and it is false in exactly the case
you care most about, which section 8 and all of v5 are about. Report both
numbers. When they disagree, the disagreement is information.

**Its error bar assumes the line fits.** The $2\sigma/\sqrt{M}$ formula treats
every deviation from the fitted line as seed noise. If contributions genuinely
interact, some of that deviation is structure the linear model cannot express,
the residuals are larger than $\sigma$, and the real error bar is wider than the
formula says. The check is direct: compare the spread of the residuals against
your independently measured $\sigma$. If the residuals are much larger, the
linear model is missing something, and what it is missing is interaction.

## Scoring an attribution method

<!-- fade: v4-lds-spearman -->

You now have two ways to produce a per-source table. Soon you will have several
cheap ones - gradient dot products, influence approximations, lexical overlap.
The question that decides whether any of them is usable is: how well does a
cheap method agree with the truth? And the obvious way to ask it is wrong,
which is why the field settled on something else.

**The obvious way, and why it fails.** Compute the true leave-one-out value for
every training example, compute your method's score for every training example,
correlate the two vectors. This is the metric everybody reaches for first. It
does not work, and section 2 already said why: per-example leave-one-out values
are individually buried a few hundred-fold under seed noise. The "true" vector
you would correlate against is not a measurement of contribution, it is a
measurement of your random number generator. Correlating a good method against
that vector gives you a number near zero, and correlating a bad method against
it gives you a number near zero, so the metric cannot tell them apart. A metric
whose ground truth is noise is not a strict metric; it is a coin.

**What works instead: predict a held-out subset's score.** The metric the field
uses is the **Linear Datamodeling Score**, and the move it makes is to stop
asking about individual examples and start asking about whole subsets, where the
effects are large.

The procedure has four steps.

1. **Hold out subsets.** From your $M$ subset runs, set aside $J$ of them and
   never let the method see them. Each held-out run has a known ingredient list
   $S_j$ and a measured outcome $v(S_j)$.
2. **Predict.** Ask the method under test what it thinks a model trained on
   $S_j$ would score. For a method that produces per-source numbers, the
   prediction is the sum of the numbers of the sources present:
   $\hat v(S_j) = \beta_0 + \sum_{i \in S_j} \beta_i$.
3. **Compare ranks.** Rank the $J$ predictions from best to worst, rank the $J$
   actual outcomes the same way, and measure how well the two orderings agree.
4. **Report the rank correlation** as the method's score.

The rank correlation used is Spearman's $\rho$, which is one subtraction and one
division once you have the ranks. Give each prediction its rank $1, 2, \ldots,
J$ (1 = lowest bpb, the best-predicted model), do the same for the actuals, and
for each held-out subset let $d_j$ be the difference between its two ranks.
Then

$$\rho = 1 - \frac{6 \sum_{j=1}^{J} d_j^2}{J(J^2 - 1)}$$

$\rho = 1$ is perfect agreement in ordering, $\rho = 0$ is no relationship, and
$\rho = -1$ is exactly reversed. The formula has no content beyond "add up how
badly the ranks disagree, and scale it so perfect is 1" - the $6$ and the
$J(J^2-1)$ exist only to make that scaling come out right.

**Worked.** Five held-out subsets. The method under test is the subset
regression fitted on the other 295 runs.

| held-out subset | predicted bpb | actual bpb | pred rank | actual rank | $d$ | $d^2$ |
|---|---|---|---|---|---|---|
| $S_1$ | 1.050 | 1.048 | 1 | 1 | 0 | 0 |
| $S_2$ | 1.060 | 1.066 | 2 | 3 | -1 | 1 |
| $S_3$ | 1.065 | 1.062 | 3 | 2 | 1 | 1 |
| $S_4$ | 1.072 | 1.071 | 4 | 4 | 0 | 0 |
| $S_5$ | 1.080 | 1.083 | 5 | 5 | 0 | 0 |

$\sum d_j^2 = 2$, and $J = 5$ so $J(J^2 - 1) = 5 \times 24 = 120$.

$$\rho = 1 - \frac{6 \times 2}{120} = 1 - 0.10 = 0.90$$

The method's LDS is 0.90. It got the ordering right except for one adjacent
swap.

**Why this metric can work where the other could not.** Look at what it is
predicting. The held-out subsets differ from each other by whole sources - each
one is missing or containing several of them at once - so the actual bpb values
span 0.035 bpb, more than ten times the seed noise $\sigma$ of 0.003. The
ground truth is a real, well-resolved measurement. Correlating against it
therefore measures the method, not the noise. That is the entire trick: LDS
moved the ground truth from a place where it could not be measured to a place
where it can.

**What an LDS does and does not license.** A high LDS says the method predicts
*aggregate* outcomes of subsets it has not seen. It does not say the method is
right about any individual source, because many different per-source vectors
produce nearly the same subset sums - two sources that always appear together in
your design can trade value between them freely without changing a single
prediction. And an LDS is measured on the design you sampled; a method scored on
half-size subsets tells you nothing certain about a subset with eleven of twelve
sources, which is the regime the leave-one-out table lives in.

```beat
id: v4-b8
type: completion
concept: d-subset-regression
# variants: blank the actual ranks instead of d^2; or give rho and blank the
# sum of d^2; or extend to J = 6 and blank the denominator.
prompt: |
  You are scoring a cheap attribution method - it produces per-source numbers
  in seconds without retraining anything - against your ground truth on six
  held-out subsets.

  | subset | predicted bpb | actual bpb | pred rank | actual rank | $d$ | $d^2$ |
  |---|---|---|---|---|---|---|
  | $S_1$ | 1.040 | 1.043 | 1 | 1 | 0 | 0 |
  | $S_2$ | 1.052 | 1.058 | 2 | ____ | ____ | ____ |
  | $S_3$ | 1.061 | 1.055 | 3 | ____ | ____ | ____ |
  | $S_4$ | 1.068 | 1.071 | 4 | 4 | 0 | 0 |
  | $S_5$ | 1.075 | 1.088 | 5 | ____ | ____ | ____ |
  | $S_6$ | 1.090 | 1.079 | 6 | ____ | ____ | ____ |

  Step 1. Fill the four missing actual ranks (1 = lowest actual bpb).
  Step 2. $\sum d_j^2 =$ ____
  Step 3. $J(J^2 - 1) = 6 \times$ ____ $=$ ____
  Step 4. $\rho = 1 - 6(\text{step 2}) / (\text{step 3}) =$ ____

  Fill the blanks. Then answer in two sentences: the same method, scored
  instead by correlating its per-document scores against per-document
  leave-one-out values, comes out at 0.02. Explain why those two numbers are
  not in conflict, and say which one you would put in a pitch deck.
answer: |
  Step 1. Sorting the actuals: 1.043, 1.055, 1.058, 1.071, 1.079, 1.088.
  So S_2 (1.058) is rank 3, S_3 (1.055) is rank 2, S_5 (1.088) is rank 6,
  S_6 (1.079) is rank 5.
  d values: S_2 = 2-3 = -1 (d^2 = 1); S_3 = 3-2 = +1 (d^2 = 1);
  S_5 = 5-6 = -1 (d^2 = 1); S_6 = 6-5 = +1 (d^2 = 1).

  Step 2. sum d^2 = 0 + 1 + 1 + 0 + 1 + 1 = 4
  Step 3. 6 x 35 = 210
  Step 4. rho = 1 - 24/210 = 1 - 0.1143 = 0.886

  Why 0.886 and 0.02 are not in conflict: they are measured against different
  ground truths, and one of the two ground truths is noise. The per-document
  leave-one-out values are individually a few hundred times smaller than the
  seed noise, so the vector being correlated against is essentially random;
  any method scored that way lands near zero, including a perfect one. The LDS
  is measured against subset outcomes that are ten times LARGER than the seed
  noise, so it is measuring the method.

  Which one goes in the deck: the LDS, with the design stated - how many
  subsets, what size, held out how. The 0.02 is not a fact about the method
  and reporting it as one would be an error in the method's disfavor, but a
  reviewer who has read the literature will ask why you did not report a
  per-document number, and the answer is the noise argument, not silence.
rubric: |
  Required, exactly: actual ranks 3, 2, 6, 5 for S_2, S_3, S_5, S_6; each
  d^2 = 1; step 2 = 4; step 3 = 35 and 210; step 4 = 0.886 (accept
  0.88-0.89).
  The explanation must contain: (1) the per-document ground truth is buried
  under seed noise so the 0.02 measures noise rather than the method, (2) the
  subset-level ground truth is large relative to sigma so LDS measures the
  method.
  All arithmetic plus (1) and (2) = pass.
  An answer that treats 0.02 as evidence the method is bad = fail; that is
  precisely the inference LDS exists to prevent, and it would have you discard
  a working method.
  An answer that reports the LDS without stating the design it was measured on
  = pass, but flag: an LDS on half-size subsets does not transfer to the
  near-full subsets a leave-one-out table lives in.
check: llm
```

## What ground truth buys

<!-- refutes: D6 -->

**You probably think gradient-based attribution is basically solved, and that
running it at frontier scale is an engineering problem.** The papers point that
way: influence functions have been scaled to 52-billion-parameter models by a
frontier lab, libraries ship 8B examples, throughput claims run into the
thousands-fold. Under this belief the small-scale counterfactual work in this
unit is a pedagogical exercise, and the real system is a matter of applying a
known method at size.

**Here is the prediction that fails.** If the scalable methods computed
approximately what retraining computes, then measuring them against retraining
would show approximate agreement. Somebody did that measurement. On a
2-billion-parameter model, evaluated against actual retraining as ground truth,
the near-optimal method reaches a rank correlation of 0.97 - and the methods
that scale perform *no better than random guessing*. Not "worse than hoped."
Random. And the method that hits 0.97 costs three to five full training runs per
test sample, so it is an oracle, not a system. Separately, a 2026 result found
that the dominant error in trajectory-based attribution methods was that the
literature assumed SGD while the models were trained with AdamW; correcting that
single mismatch moved published results by 10 to 300%. Numbers that move by 300%
when you fix one line were not measuring what their authors thought.

**Here is why the wrong model is appealing.** Every result in that paragraph is
published by the same community that publishes the methods, in venues the
methods appear in, and none of it makes headlines. The scaling demonstrations do
make headlines, and a demonstration that a method *runs* at 52B parameters looks
exactly like a demonstration that it *works* at 52B parameters. The papers are
usually careful; the summaries are not. And an engineer's prior is reasonable
here - in most of computing, a technique that works at small scale and runs at
large scale works at large scale.

**Here is what is actually true.** The central dilemma of this field: the
methods that scale do not demonstrably correlate with ground truth, and the
method that correlates does not scale. That is not a temporary state of the art.
It is the shape of the problem, and it has a direct consequence for anything you
build: **an attribution claim at scale is worth exactly the measured
correlation that accompanies it, and that correlation can only be measured
where retraining is affordable.**

Which is here. The table you produce in this unit is not a warm-up for the real
system; it is the only instrument in the building that can tell you whether the
real system is lying. Three concrete things it buys.

**It catches your own bugs.** An attribution implementation produces a
plausible-looking ranking whether or not it is correct. There is no unit test
for "these influence scores are the right influence scores." Against a
counterfactual table there is: run the cheap method on the full model, compute
its LDS, and a badly wrong implementation scores near zero. Most people build
the expensive machinery first and never find out whether their version of it
works. Doing this in the other order is the whole strategic argument for
starting here.

**It converts a hope into a number.** "Our attribution is accurate" is a claim
nobody can evaluate and everybody has heard. "Our TracIn implementation has an
LDS of 0.61 against retraining ground truth on a 12-source corpus at 10M
parameters, measured on 60 held-out subsets" is a claim with a method, a scope,
and a limit. It is also a claim a skeptic can attack productively, which is what
makes it worth more than the vague version.

**It tells you which cheap method to buy.** Nobody can hand you this answer.
Whether gradient similarity, lexical overlap, or a retrieval index best predicts
contribution depends on your corpus, your model scale, and what your held-out
set rewards. The published benchmarks are unanimous that no method dominates,
and that cheap lexical baselines match or beat expensive gradient methods on
some tasks. The only way to know which holds for you is to measure it against
your own truth.

### What the table does not license

One honest limit, stated here rather than discovered by someone across a table
from you. "Ground truth" is a name for a specific computed quantity, not a
claim of universality. Your $\Delta_i$ is exact for: this model architecture,
this parameter count, this token budget, this protocol, this held-out set, and
these twelve sources. Influence patterns are known to change with scale - small
models show crisp, top-heavy attributions while frontier-scale influence is a
long tail where per-document payouts round to zero - so a contribution
established at 10 million parameters is evidence about 10-million-parameter
models and a hypothesis about anything larger.

That is not a fatal limitation, it is a scope statement, and it is a scope
statement every other method in this space also needs and mostly does not make.
State it, and then note what follows: since the counterfactual is exact within
its scope, the only way to learn whether it *transfers* is to repeat it at a
larger scale and compare, which is a well-defined experiment with a known price
rather than a matter of opinion.

### Two things this sets up

**The split is not the table (v5).** Leave-one-out contributions do not add up
to the total. Suppose sources A and B are near-duplicates of one another. Remove
A and the model barely notices, because B still covers the material, so
$\Delta_A \approx 0.002$. Remove B and the same thing happens: $\Delta_B \approx
0.002$. Remove both and the loss jumps by 0.020. The individual contributions
sum to 0.004 and the joint effect is 0.020 - five times larger. Any payout that
distributes money in proportion to $\Delta_i$ has just paid these two sources a
fifth of what their material was worth, and split it between them arbitrarily.
Fixing that is what v5 is for: coalitions, the Shapley axioms, and a split that
is provably fair rather than merely computed.

**The cheap methods get graded (v8).** Every method in v8 produces a per-source
ranking in seconds. Each one gets an LDS against the table you built here. The
resulting number - does cheap attribution correlate with truth on this corpus -
is the single most important measurement in the entire project, because it
determines whether the approach generalizes past the scale where you can afford
to retrain.

```beat
id: v4-b9
type: self-explain
concept: d-counterfactual
prompt: |
  Two facts from your pipeline, both recorded before any model was trained.

  Fact 1. The v2 dedup stage found a cluster of 4,100 near-identical
  passages appearing in both source 4 (a preprint server) and source 9 (a
  journal mirror). Rather than silently collapsing them, the manifest recorded
  the cluster and its source membership, and the pipeline kept source 4's
  copies.

  Fact 2. Your finished table reads $\Delta_4 = 0.028$ and
  $\Delta_9 = -0.006$, both clearing the noise floor.

  Explain, in your own words, what fact 1 predicts about fact 2, what would
  have happened to both numbers if the pipeline had kept source 9's copies
  instead, and what this implies about presenting the table as a basis for
  payment.
answer: |
  What fact 1 predicts: dedup chose which source gets credit for everything
  the shared passages teach. Source 4 kept the copies, so the model learned
  that material from source 4's tokens. Removing source 4 therefore loses it,
  which inflates Delta_4. Removing source 9 loses only its unique material,
  and at a fixed token budget its remaining tokens are partly redundant with
  what source 4 already carries - so Delta_9 is small, and can be negative
  once those tokens are displacing more useful ones.

  What a different dedup decision would have done: keep source 9's copies
  instead and the two numbers largely swap. Source 9 becomes the source that
  carries the shared material and its delta rises; source 4 loses it and its
  delta falls, possibly below zero by the same displacement argument. Nothing
  about the corpus, the sources, the model, or the eval set changed. A choice
  made in a pipeline stage that every engineer regards as hygiene moved a
  large fraction of the measured value from one rightsholder to another.

  What this implies for the table: the contribution numbers are conditional on
  the deduplication policy, and the policy must ship alongside them. The v2
  manifest makes this auditable rather than invisible - the duplicate cluster
  was recorded, so anyone can see that 4,100 passages were assigned to source
  4 by rule rather than by measurement. Without that record the table looks
  like a measurement of two sources and is in part a measurement of a sort
  order. Credit assignment for shared content has to be an explicit, stated
  policy, and the honest framing of Delta_4 is "source 4's contribution UNDER
  this dedup policy," not "source 4's contribution."
rubric: |
  Must contain: (1) the mechanism - whichever source keeps the shared copies
  is the source the model learns the material from, so it absorbs that
  material's contribution, (2) the counterfactual - swapping the dedup
  survivor largely swaps the two deltas, with nothing about the underlying
  data having changed, (3) the consequence - the deduplication policy is part
  of the result and must be reported with the table.
  (1) and (2) = pass. All three = full credit.
  An answer that treats Delta_9's negative sign as evidence that source 9 is
  low quality = fail, diagnosing V4-M3; the sign here is manufactured by a
  pipeline decision.
  An answer that proposes to fix this by deduplicating more aggressively =
  fail; more dedup makes MORE of these decisions, not fewer. The fix is
  recording the clusters and stating the policy, which the pipeline already
  does.
  An answer that says the two sources should split the shared material's
  credit = pass, and note it is the right instinct: that is a coalition
  question and v5 gives it a principled answer.
check: llm
```

## At the bench: the leave-one-out sweep

Here is what to build after this unit, and what it feeds.

**Build.** Take the tagged corpus from v2 and the 10-million-parameter training
configuration from v3 - the one whose seed noise $\sigma$ you already measured.
Run the leave-one-source-out sweep on the Mac. That is $K + 1 = 13$ conditions
at 3 seeds each, 39 runs, roughly 20 minutes apiece, so two evenings of
unattended laptop time. Fix the protocol first and write it in the manifest:
fixed token budget, backfilling each leave-one-out condition by repeating the
surviving sources to the full token count. Fix the held-out paper set first
too - the same one v3 used, decontaminated against every shard.

**Artifact.** One table, twelve rows, four columns: source, $\Delta_i$,
$\text{SE}_{\Delta}$, and $z$. Plus a header block recording $\sigma$, $n$, the
protocol, the held-out set, and the deduplication policy - the five things
without which the table cannot be interpreted. Sources whose $|z| < 2$ are
labeled *below the noise floor*, not rounded to a small number. Add one line
stating the smallest contribution this sweep could have resolved, which is
$2\sigma\sqrt{2/n}$; at $\sigma = 0.003$ and $n = 3$ that is 0.005 bpb.

**Then queue the subset runs for a rented day.** Draw $M = 300$ random subsets
of the 12 sources, one seed each, and train them at the larger configuration
where a run is under four minutes. That is under 19 GPU-hours, call it a
weekday afternoon and under $80. Fit the regression by the same in-minus-out
arithmetic as section 6 if you sample a balanced design, or by any least-squares
solver if you sample uniformly at random. Hold out 50 to 60 of the runs from the
fit and report the LDS of the fitted datamodel against them - that number is
your ceiling, the best any per-source linear summary can do on this corpus, and
every cheap method in v8 gets compared to it as well as to zero.

**What feeds forward.** The contribution table is the ground truth v8 grades
against. The cached subset outcomes - every $(S, v(S))$ pair you trained, kept
in one file - are the utility table v5 computes Shapley values from without
training anything new; a run you did for the regression is a coalition you do
not have to re-train, so cache aggressively and key by the exact source set. The
measured $\sigma$ carries forward unchanged as the significance bar. And the
sentence about which sources sit below the floor is the first real input to the
question v9 has to answer in front of an investor: whether this corpus clears
the signal-to-noise bar that makes contribution-proportional payment better than
a flat fee.

```beat
id: v4-b10
type: self-explain
concept: d-noise-floor
prompt: |
  You are writing, in advance, the header block that ships above your
  contribution table - the metadata without which a reader cannot interpret a
  single row. A colleague proposes recording: the model's parameter count, the
  total training tokens, the date of the run, and the git commit of the
  training code.

  That list is reproducibility metadata and it is missing every item that is
  needed to INTERPRET the numbers. State at least four things that must also
  be recorded, and for each one, name the specific misreading of the table
  that its absence permits. Answer from this unit's text; do not reference any
  run you have or have not performed.
answer: |
  Four required items, each with the misreading it prevents:

  1. **sigma and the seed count n.** Without them there is no SE_delta =
  sigma*sqrt(2/n), so no reader can tell which rows are signal. A source at
  0.001 bpb and a source at 0.031 bpb look like a small contributor and a
  large one, when in fact one of them is indistinguishable from zero. This is
  the misreading that turns a table into a liability, because it is the one
  that assigns money to noise.

  2. **The leave-one-out protocol - fixed corpus or fixed token budget.**
  Without it, a reader cannot tell whether a large source's large number is
  its content or its volume. The two protocols do not offset each other by a
  constant; they systematically favor large sources, so the protocol changes
  the ORDER of the table and not just its scale.

  3. **The held-out evaluation set, chosen and decontaminated before the
  runs.** Contribution is defined relative to a measurement of "worse," and
  that measurement is this file. A source whose subject matter is absent from
  the held-out papers scores zero regardless of its value, and without the
  eval set named, a zero reads as "contributed nothing" rather than
  "contributed nothing measurable by this yardstick."

  4. **The deduplication policy and the duplicate-cluster record.** Where two
  sources shared material, the pipeline chose which one keeps it, and that
  choice moves contribution between them. Without the policy stated, a delta
  inflated by having won a dedup tiebreak reads as a measured property of the
  source.

  Also acceptable in place of any of these: the model scale as a scope
  statement (this is a 10M-parameter result and influence patterns change with
  scale), and the source list itself with each source's token share, since a
  contribution is relative to which other sources were present.
rubric: |
  Must name at least four items with the misreading each prevents. The four
  canonical ones are: sigma and n / the error bar; the fixed-corpus vs
  fixed-budget protocol; the held-out eval set; the dedup policy. Accept
  model-scale scope statement or the source list with token shares as
  substitutes for at most one.
  Naming an item without the misreading it permits earns half credit for that
  item; three fully-explained items = pass.
  Omitting sigma and n = fail regardless of the rest. That is the load-bearing
  item and its absence is the failure this whole unit exists to prevent.
  An answer that lists items but frames them as "good practice" or
  "reproducibility" rather than as interpretation prerequisites = partial, do
  not pass; the distinction matters because reproducibility metadata lets
  someone rerun your experiment while interpretation metadata lets them read
  your result, and only one of those is needed to send an invoice.
check: llm
```

## What you can now do

Three capabilities, and one sentence that is the point of the unit.

**You can compute a contribution rather than estimate one.** Train with, train
without, subtract. The definition needs no theory, works for any model and any
metric, and is exact for the runs you performed. Everything more sophisticated
in this book is an approximation of this subtraction, built for people who
cannot afford the second run - and at 10 to 30 million parameters, you can.

**You can say whether a number is real.** Average within condition, subtract,
divide by $\sigma\sqrt{2/n}$. Report the pair, never the point. When the answer
is "below the noise floor," write that sentence rather than a smaller number,
and state the smallest effect the sweep could have resolved. A contribution
inside the noise is a finding about your instrument, and the instrument's floor
is a headline in either direction.

**You can grade any attribution method against the truth.** Hold out subsets,
predict their outcomes, rank-correlate. That is the LDS, it is the field's
standard metric, and it exists because the obvious metric - correlating against
per-example leave-one-out values - correlates against noise. Any claim about a
cheap method that arrives without an LDS and a stated design has not been
tested; the published measurements say the methods that scale come out at
chance.

The sentence: **the only honest attribution system is one that has been measured
against a counterfactual, and the counterfactual is only affordable small.** So
the small experiment is not the prototype of the real thing. It is the
calibration standard the real thing has to be checked against, and building it
first is the difference between an attribution product and an attribution
demo.

v5 takes the contributions you just measured and turns them into a split -
which requires facing the fact that they do not add up. v8 takes the table and
uses it to grade the cheap methods. Both of those are downstream of one file
you now know how to produce: twelve rows, an error bar on each, and a header
that says what they mean.

## Notation in this unit

<!-- canon-only -->

Reference, not reading. Return here when a symbol goes blurry; nothing below is
new.

| Symbol | Means | Typical value here |
| --- | --- | --- |
| $K$ | number of tagged sources in the corpus | 10 to 50 |
| $S$ | a subset of the sources - the ingredient list for one run | any of $2^K$ |
| $v(S)$ | held-out bits per byte of a model trained on $S$; **lower is better** | 1.0 to 1.2 |
| $\Delta_i$ | leave-one-out contribution of source $i$, positive means helpful | -0.01 to 0.05 |
| $n$ | seeds per condition | 3 to 20 |
| $\sigma$ | seed noise: SD of held-out bpb across seeds at fixed configuration | 0.002 to 0.008 |
| $\text{SE}_\Delta$ | standard error of a measured contribution | 0.0005 to 0.005 |
| $z$ | $\Delta_i / \text{SE}_\Delta$; the bar for "real" is 2 | 0 to 15 |
| $M$ | number of random-subset training runs | 100 to 500 |
| $x_i$ | inclusion indicator: 1 if source $i$ is in this run's subset, else 0 | 0 or 1 |
| $\beta_i$ | regression coefficient on $x_i$: bpb change from including source $i$ | negative when helpful |
| $c_i$ | regression contribution, $= -\beta_i$, positive means helpful | -0.01 to 0.05 |
| $J$ | held-out subsets used to score a method | 30 to 60 |
| $\rho$ | Spearman rank correlation | -1 to 1 |
| LDS | Linear Datamodeling Score: $\rho$ of predicted against actual on held-out subsets | 0 to 1 |
| $N$ | parameters in the model | $10^7$ to $3 \times 10^7$ |
| $D$ | training tokens for one run | $2 \times 10^8$ to $5 \times 10^8$ |
| $C$ | floating-point operations for one run | $10^{16}$ to $10^{17}$ |

The five formulas, with every symbol as defined above:

$$\Delta_i = v(\text{all except } i) - v(\text{all})
\qquad
\text{SE}_\Delta = \sigma\sqrt{\tfrac{2}{n}}
\qquad
z = \frac{\Delta_i}{\text{SE}_\Delta}$$

$$n \geq \frac{8\sigma^2}{\Delta^2}
\qquad
\rho = 1 - \frac{6\sum_j d_j^2}{J(J^2-1)}$$

plus $C \approx 6ND$ from v3 for pricing any of it, and $c_i = -\beta_i$ for
converting a regression coefficient into a contribution with the sign facing
the same way as $\Delta_i$.

Two sign conventions worth writing on the inside of your eyelids, because
almost every error in this material is one of them. **Bits per byte is a cost:
lower is better, so a helpful source makes $v$ go down.** And **contributions
are quoted so that positive means helpful**, which is why $\Delta_i$ puts the
removal first and $c_i$ carries a minus sign.
