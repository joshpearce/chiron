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
  Before reading on, commit to a reading. You are about to send that table -
  F at 1.043 bpb, F\7 at 1.061 bpb, a gap of 0.018 - to a rightsholder as the
  basis for a payment.

  Which of these correctly states what the 0.018 licenses, together with
  properties of the experiment itself that make it the wrong number to bill
  on?
options:
  - text: |
      It licenses only this: a model of this size and token budget, trained on
      this corpus without source 7, is 0.018 bpb worse on this held-out set
      than the same model trained with it. Two features of the experiment
      already undercut billing on it - each figure is a single draw from a
      stochastic training process, and removing source 7 also removed its
      tokens, so content and volume are confounded in the same 0.018.
    correct: true
    explain: |
      Right. The counterfactual is exact for the two runs that produced it and
      says nothing beyond them. One seed per condition leaves the gap with no
      error bar, and $v(\text{all} \setminus 7)$ was trained on about 8% fewer
      tokens, so part of the gap is a shorter training run. Redundancy with
      another source and the choice of held-out set are two more attacks of
      the same kind.
  - text: |
      It licenses "source 7 contributed 0.018 bpb" as a property of source 7.
      The counterfactual was computed by actually retraining rather than
      estimated, so the only caveat worth stating is that the model is small,
      and that caveat disappears once the same sweep is run at 8B parameters.
    misconception: V4-M4
    explain: |
      Retraining makes the number exact for its configuration - this
      architecture, parameter count, token budget, protocol, held-out set and
      source list - which is a scope statement, not a caveat that scale
      removes. Influence patterns themselves change with scale, and the real
      attacks here (one seed, tokens confounded with content) apply at every
      size.
  - text: |
      It licenses source 7's contribution as measured, because both runs used
      the same random seed. Whatever randomness training has is identical on
      both sides, so it cancels in the subtraction, and the remaining question
      is only how many decimal places of bpb to report.
    misconception: D16
    explain: |
      Sharing a seed does not make the difference deterministic: seeds 2 and 3
      give gaps of 0.012 and 0.015 for the same two conditions. The
      full-corpus runs agree to within 0.3% while their differences from the
      leave-one-out runs disagree by 50% - a difference of two nearly equal
      noisy numbers is exactly where variance concentrates.
  - text: |
      It licenses source 7's share of the model: run the other eleven
      leave-one-out conditions, sum the twelve gaps, and pay source 7
      $0.018$ divided by that sum. The gap is the wrong number to bill on only
      until the other eleven gaps exist to normalise it against.
    misconception: V4-M1
    explain: |
      $\Delta_i$ is a marginal contribution to the full corpus, and marginal
      contributions do not sum to the total whenever sources interact. Two
      near-duplicate sources can each show $\Delta \approx 0.002$ while
      removing both costs 0.020 - the slices miss the pie by a factor of five,
      so the denominator in that division is not a total of anything.
check: choice
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

  Which explanation of what is right and wrong with this would you give?
options:
  - text: |
      The colleague has correctly identified signal-to-noise as the obstacle,
      and shrinking the corpus really does raise the effect. But required
      seeds go as $n \geq 8\sigma^2/\Delta^2$, so a 25x effect cuts the seed
      count by 625x - from roughly 1.3 million seeds per condition to about
      2,000, per document, across 1,000 documents. And in a corpus that small
      each document is individually load-bearing, which is not the regime the
      billed model is in, so the measurement got easier by changing what is
      being measured.
    correct: true
    explain: |
      Both halves matter. The effect size enters squared, so it is the
      strongest available lever and still leaves the sweep orders of magnitude
      out of reach. And the number that came back would describe a model no
      longer resembling the one under discussion. The fix is to raise the
      unit, not shrink the corpus: a source is large by construction, so it
      moves the loss by construction, and it is what receives the payment.
  - text: |
      The plan works. A 25x effect needs 25x fewer seeds, which brings 1.3
      million down to about 52,000 per condition - large but parallelisable
      across a rented cluster. Per-document attribution was never impossible
      in principle; it was waiting on somebody designing the experiment
      properly.
    misconception: D20
    explain: |
      The seed requirement is $8\sigma^2/\Delta^2$, with the effect squared,
      so 25x buys 625x rather than 25x - and 2,000 seeds per condition per
      document over 1,000 documents is still out of reach. Treating the noise
      as a design problem that refinement dissolves is the belief this unit
      exists to price: the floor is a measurable quantity, not a temporary
      state of the art.
  - text: |
      The plan works and the resulting numbers transfer. A counterfactual is a
      counterfactual: retraining on 1,000 documents measures true per-document
      contribution, and corpus size is a detail of how the number was obtained
      rather than part of what it means, so the table can be applied to the
      25,000-document corpus.
    misconception: V4-M4
    explain: |
      A counterfactual is exact for the configuration that produced it, and
      corpus composition is part of that configuration. In a 1,000-document
      corpus each document carries a tenth of a percent of everything the
      model knows; at 25,000 it carries 0.004%, where per-document effects
      round into the noise. The arithmetic also still fails: 2,000 seeds per
      condition per document.
  - text: |
      Reject the subcorpus, and spend the compute on random-subset runs
      instead - a few thousand runs over document subsets averages the seed
      noise down, which shrinks the error bar on each document's leave-one-out
      value without any change to the corpus.
    misconception: V4-M2
    explain: |
      $\text{SE}_{\Delta} = \sigma\sqrt{2/n}$ contains no run count $M$: a
      leave-one-out delta is a difference between two condition means, so only
      seeds on those two conditions tighten it. Subset runs give a different
      estimator of a different quantity - an average marginal effect across
      the subsets sampled - which coincides with $\Delta_i$ only under
      additivity.
check: choice
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

  Before reading on, commit to a prediction: which of these describes how
  $\Delta_3$ and $\Delta_{11}$ change between the two protocols?
options:
  - text: |
      Every fixed-corpus number is larger, because that model is handicapped
      twice - missing content and missing tokens. $\Delta_3$ falls sharply
      when the 22% token shortfall is backfilled, possibly by more than half;
      $\Delta_{11}$ barely moves, since a 1.5% shortfall was nearly nothing
      and its number was already almost all content. So the protocols do not
      offset each other by a constant - they reorder the table in favour of
      large sources.
    correct: true
    explain: |
      Right, and this is why the protocol is pre-registered and reported
      beside the table. A rightsholder scored under fixed corpus on a 22%
      share has a correct argument that they were paid for volume rather than
      content, and the size of that effect exceeds most of the contributions
      you are trying to detect.
  - text: |
      Neither number changes appreciably. $\Delta_i$ is defined as the
      counterfactual effect of removing source $i$, so it measures that
      source's content; how the surviving corpus was assembled afterwards is
      an implementation detail of running the experiment, not part of what the
      number means.
    misconception: V4-M4
    explain: |
      The counterfactual measures whatever differed between the two runs, and
      under fixed corpus what differed was the content and about a fifth of
      the training tokens. Exactness is exactness within a stated scope, and
      the protocol is part of that scope - which is precisely why it goes in
      the header block.
  - text: |
      Fixed corpus gives larger numbers, but by roughly the same offset for
      every source, since each leave-one-out model simply trained a bit short.
      The ranking of sources is therefore stable across protocols, so either
      one can be used as long as the choice is applied consistently.
    misconception: D16
    explain: |
      The shortfall is proportional to each source's token share, so it is
      0.22 of the run for source 3 and 0.015 for source 11 - not a constant.
      A constant offset would indeed be harmless; this one systematically
      inflates large sources, and an inflation that varies by row is a
      reordering.
  - text: |
      Fixed token budget is the flawed protocol, and both numbers come out
      identical under it anyway. Backfilling by repeating surviving sources
      adds no new information - repeated tokens teach the model nothing - so
      the backfilled run is effectively the short run with wasted steps.
    misconception: D17
    explain: |
      Repetition in the data-constrained regime is close to free: up to about
      four epochs costs almost nothing against fresh tokens, with returns
      diminishing out to roughly sixteen. Adding an eighth of an epoch to the
      survivors genuinely restores the compute, which is exactly what makes
      the fixed-budget protocol affordable and what makes the two protocols
      differ.
check: choice
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
  Step 4. $\text{SE}_{\Delta} =$ ____
  Step 5. $z = \Delta_9 / \text{SE}_{\Delta} =$ ____

  Which filling of the blanks, and which reading of the result, is correct?
options:
  - text: |
      1.043; 3.111 / 3 = 1.037; $\Delta_9 = 1.037 - 1.043 = -0.006$;
      $\text{SE}_{\Delta} = 0.003\sqrt{2/3} = 0.00245$; $z = -2.45$. It is a
      result: $|z|$ clears 2, so removing source 9 made the model better by
      0.006 bpb. At this fixed token budget its tokens were worth less than
      the tokens that replaced them - most likely it duplicates material
      another source carries, or its subject matter is absent from the
      held-out papers.
    correct: true
    explain: |
      Right on all five, and right that the sign is a statement about the
      marginal token in this corpus against this held-out set. Average within
      condition, subtract, then apply $\sigma\sqrt{2/n}$ - and report the pair
      $-0.006 \pm 0.0025$ rather than the bare number.
  - text: |
      1.043; 3.111 / 3 = 1.037; $\Delta_9 = -0.006$;
      $\text{SE}_{\Delta} = 0.00245$; $z = -2.45$. It is a result, and what it
      found is bad data: source 9 is noisy or corrupted, which is why training
      on it hurt. Drop it from the corpus and re-run the sweep.
    misconception: V4-M3
    explain: |
      The arithmetic is right and the verdict is not. A negative delta says
      the source's marginal token was worth less than the token that replaced
      it, in this corpus, at this budget, against this held-out set - and a
      clean mirror of material another source already carries produces exactly
      this sign with nothing wrong in it. Remove the duplicate partner too and
      the sign can flip positive, which no property of source 9's own data
      could do.
  - text: |
      1.043; 3.111 / 3 = 1.037; $\Delta_9 = -0.006$;
      $\text{SE}_{\Delta} = \sigma/\sqrt{n} = 0.0017$; $z = -3.5$. A $|z|$ of
      3.5 puts this three and a half standard errors from zero, well clear of
      the bar, so source 9's negative contribution is firmly established.
    misconception: D19
    explain: |
      $\sigma/\sqrt{n}$ is the error on one condition's mean; $\Delta_9$ is a
      difference of two independent means, so the variances add and
      $\text{SE}_{\Delta} = \sigma\sqrt{2/n} = 0.00245$. Understating the
      error bar by $\sqrt{2}$ manufactures confidence, and it is the procedure
      rather than the arithmetic that an opposing reader will attack.
  - text: |
      1.043; 3.111 / 3 = 1.037; $\Delta_9 = 1.043 - 1.037 = +0.006$;
      $\text{SE}_{\Delta} = 0.00245$; $z = +2.45$. Source 9 helped by 0.006
      bpb - the subtraction has to run this way round, since a source's share
      of the model cannot be negative.
    misconception: V4-M1
    explain: |
      $\Delta_i = v(\text{all except } i) - v(\text{all})$ puts the removal
      first precisely so that positive means helpful, and here that gives
      $-0.006$. Negative contributions are ordinary; a table that cannot
      express one cannot express redundancy or displacement at a fixed budget,
      which is most of what a real sweep finds.
check: choice
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

  Which verdict on the three would you give?
options:
  - text: |
      None of them reduces the count. Seed noise is intrinsic at every scale,
      and since cost is linear in $N$ a 30x model buys 30x fewer seeds, so (a)
      moves away from the floor. No run count $M$ appears in
      $\text{SE}_{\Delta} = \sigma\sqrt{2/n}$, so (b) leaves the
      leave-one-out table untouched, however useful the regression is in its
      own right. And below the floor the ordering is exactly what noise
      determines, so (c) publishes a permutation. The remedy is purchasable:
      $n \geq 8\sigma^2/\Delta^2$ means resolving 0.002 bpb at
      $\sigma = 0.003$ takes 18 seeds per condition, about 2.3 GPU-hours.
    correct: true
    explain: |
      Right, and the last clause is the point of the seed formula - the floor
      is a price rather than a verdict. Where the price is not worth paying,
      "these eight sources are not distinguishable from zero, and the smallest
      effect this sweep could resolve is 0.005 bpb" is itself the deliverable.
  - text: |
      (a) is the right call. Contribution is a stable quantity and the wobble
      at 10M parameters is small-model instability - a 300M model has smoother
      training dynamics, so its seed-to-seed spread is far tighter and the
      same sources clear the bar. (b) and (c) are both shortcuts around a
      problem that scale removes.
    misconception: D16
    explain: |
      Contribution is a random variable at every scale and a 300M model has
      its own $\sigma$; what changes with scale is the invoice. Since
      $C \approx 6ND$, each seed costs 30x more, and seeds are the only thing
      in $\sigma\sqrt{2/n}$ that you can move - so this is the most expensive
      available way to make the floor worse.
  - text: |
      (b) is the right call. Doubling to 600 subset runs doubles the amount of
      averaging behind every source, and averaging is the stated remedy for
      seed noise, so the error bars on the leave-one-out contributions tighten
      by $\sqrt{2}$ for the same compute - and the regression comes free with
      it. (a) and (c) are then unnecessary.
    misconception: V4-M2
    explain: |
      Averaging shrinks the error of the estimator doing the averaging.
      $\Delta_i$ reads two condition means and nothing else, so
      $\text{SE}_{\Delta} = \sigma\sqrt{2/n}$ is unchanged by ten thousand
      subset runs. The regression coefficient $c_i$ does tighten as
      $2\sigma/\sqrt{M}$, but it estimates an average marginal effect across
      sampled subsets, and it coincides with $\Delta_i$ only under additivity.
  - text: |
      (c) is the right call as an interim measure. The point estimates are the
      best available estimates, ranking is a weaker claim than magnitude, and
      a methodology note discloses the uncertainty honestly - so the table can
      ship now, with (a) and (b) as later refinements.
    misconception: D19
    explain: |
      For the eight sources under the floor the ordering is the part noise
      determines: reseed the sweep and they permute, so the ranking carries no
      information to disclose a caveat about. A number becomes evidence
      through the procedure reported with it, and a table whose second column
      is missing cannot be read at all - which is why the deliverable is
      contributions and standard errors together.
check: choice
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

  Step 1. A in (runs 2,4,6,8) $=$ ____ ; A out (runs 1,3,5,7) $=$ ____
  Step 2. $\beta_A =$ ____ , so $c_A =$ ____
  Step 3. C in (runs 5,6,7,8) $=$ ____ ; C out (runs 1,2,3,4) $=$ ____ ;
          $c_C =$ ____
  Step 4. D in (runs 1,4,6,7) $=$ ____ ; D out (runs 2,3,5,8) $=$ ____ ;
          $c_D =$ ____

  Which filling of the blanks, and which reading of $c_D$, is correct?
options:
  - text: |
      A in $= 4.600/4 = 1.150$, A out $= 4.760/4 = 1.190$, so
      $\beta_A = -0.040$ and $c_A = 0.040$. C in $= 4.680/4 = 1.170$, C out
      $= 4.680/4 = 1.170$, so $c_C = 0.000$. D in $= 4.700/4 = 1.175$, D out
      $= 4.660/4 = 1.165$, so $\beta_D = +0.010$ and $c_D = -0.010$. What is
      unusual is that $c_D$ is negative: averaged across this design,
      including D made the model worse. Either D's content is off-distribution
      for the held-out set, or at a fixed token budget its tokens displaced
      tokens from A, B or C - and these runs cannot tell the two apart.
    correct: true
    explain: |
      Right, including the sign discipline: $c_i = -\beta_i$, so a helpful
      source has negative $\beta$ and positive $c$. Separating displacement
      from off-distribution content needs the interaction terms - whether D's
      coefficient shifts with which other sources are present - which is what
      v5's coalition machinery supplies.
  - text: |
      A in $= 1.150$, A out $= 1.190$, $\beta_A = -0.040$, $c_A = 0.040$; C in
      $= 1.170$, C out $= 1.170$, $c_C = 0.000$; D in $= 1.175$, D out
      $= 1.165$, $\beta_D = +0.010$, $c_D = -0.010$. What is unusual is that
      $c_D$ is negative, which means source D is noisy or corrupted data - the
      regression has found the junk in the corpus, and D should be dropped.
    misconception: V4-M3
    explain: |
      Every number is right and the diagnosis is not. A negative coefficient
      says D's marginal token was worth less than the token that replaced it
      on this evaluation; a clean mirror of material A already carries scores
      exactly this way, and reading a hundred of its documents finds nothing
      wrong. Check the duplicate-cluster record and whether D's subject matter
      appears in the held-out set before touching the corpus.
  - text: |
      A in $= 1.150$, A out $= 1.190$, $\beta_A = -0.040$, $c_A = 0.040$; C in
      $= 1.170$, C out $= 1.170$, $c_C = 0.000$; D in $= 1.175$, D out
      $= 1.165$, $\beta_D = +0.010$, $c_D = -0.010$. What is unusual is that
      $c_D$ is negative, and since $c_D$ is D's leave-one-out contribution
      computed cheaply, it predicts that removing D from the full four-source
      corpus will improve held-out bpb by exactly 0.010.
    misconception: V4-M2
    explain: |
      $c_i$ and $\Delta_i$ are not estimates of the same quantity. $c_D$ is an
      average marginal effect across the subsets this design sampled, most
      containing about half the sources; $\Delta_D$ is the marginal effect at
      full density with A, B and C all present. They coincide only under
      additivity, and the disagreement between them is the interaction
      diagnostic - report both.
  - text: |
      A in $= 1.150$, A out $= 1.190$, $\beta_A = -0.040$, $c_A = 0.040$; C in
      $= 1.170$, C out $= 1.170$, $c_C = 0.000$; D in $= 1.175$, D out
      $= 1.165$, and since a contribution is a share of the model's value it
      must be non-negative, so $c_D = 1.175 - 1.165 = +0.010$. Nothing is
      unusual: A, C and D take 0.040, 0.000 and 0.010 of the total.
    misconception: V4-M1
    explain: |
      The coefficient is in-minus-out, $\beta_D = 1.175 - 1.165 = +0.010$, and
      $c_D = -\beta_D = -0.010$; reversing the subtraction to avoid a negative
      is the pie-chart instinct, not the arithmetic. Coefficients need not be
      non-negative and need not sum to anything, which is why a payout
      proportional to them is a choice that has to be defended.
check: choice
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

  Step 1. The four missing actual ranks (1 = lowest actual bpb).
  Step 2. $\sum d_j^2 =$ ____
  Step 3. $J(J^2 - 1) = 6 \times$ ____ $=$ ____
  Step 4. $\rho = 1 - 6(\text{step 2}) / (\text{step 3}) =$ ____

  The same method, scored instead by correlating its per-document scores
  against per-document leave-one-out values, comes out at 0.02.

  Which filling of the blanks, and which reading of the two numbers, is right?
options:
  - text: |
      Actual ranks: $S_2 = 3$, $S_3 = 2$, $S_5 = 6$, $S_6 = 5$, so each of the
      four contributes $d^2 = 1$. Step 2: $\sum d_j^2 = 4$. Step 3:
      $6 \times 35 = 210$. Step 4: $\rho = 1 - 24/210 = 0.886$. The 0.886 and
      the 0.02 are measured against different ground truths and one of those
      ground truths is noise: per-document leave-one-out values sit a few
      hundred-fold below $\sigma$, so that correlation is near zero for any
      method, including a correct one. Put the LDS in the deck, with its
      design stated - how many subsets, of what size, held out how.
    correct: true
    explain: |
      Right. Sorting the actual column gives 1.043, 1.055, 1.058, 1.071,
      1.079, 1.088, so two adjacent pairs swap against the predicted order and
      $\sum d_j^2 = 4$; $J^2 - 1 = 35$ at $J = 6$. And the two scores are not
      in conflict because LDS moved the ground truth to subsets, where the
      spread is ten times $\sigma$ instead of a few hundredths of it.
  - text: |
      Actual ranks: $S_2 = 3$, $S_3 = 2$, $S_5 = 6$, $S_6 = 5$; $\sum d_j^2 = 4$;
      $6 \times 35 = 210$; $\rho = 0.886$. But the per-document correlation of
      0.02 is the stricter and more honest measurement, since per-document
      leave-one-out values are the finest-grained ground truth available: the
      method is weak and 0.02 is the number to report.
    misconception: D16
    explain: |
      The arithmetic is right and the conclusion inverts the unit. A single
      paper is about 0.004% of the corpus, so its leave-one-out effect is
      around $7.5 \times 10^{-6}$ bpb against $\sigma = 0.003$ - the vector
      being correlated against is a measurement of the random seed. A metric
      whose ground truth is noise scores a good method and a bad method alike
      near zero, which is exactly why LDS exists.
  - text: |
      The actual column rises in the same order as the predicted column, so
      every actual rank matches its predicted rank, each $d = 0$, and
      $\sum d_j^2 = 0$; $6 \times 35 = 210$; $\rho = 1 - 0 = 1.0$. A perfect
      LDS means the cheap method reproduces the counterfactual, so it can be
      run at 8B parameters and its rankings trusted there.
    misconception: D6
    explain: |
      The actual column is not monotone in the predictions: 1.055 sits below
      1.058 and 1.079 below 1.088, so two pairs swap and $\sum d_j^2 = 4$,
      giving 0.886. And even a perfect LDS would be a statement about this
      scale and this design - measured against retraining on a 2B model, the
      methods that scale come out at chance.
  - text: |
      Actual ranks: $S_2 = 3$, $S_3 = 2$, $S_5 = 6$, $S_6 = 5$; $\sum d_j^2 = 4$;
      $J(J^2 - 1) = 6 \times 36 = 216$; $\rho = 1 - 24/216 = 0.889$. At that
      score the method's per-source numbers are individually validated, so
      they can be summed straight into payout shares.
    misconception: V4-M1
    explain: |
      Two errors. $J^2 - 1 = 35$ at $J = 6$, not 36, so the denominator is 210
      and $\rho = 0.886$. And a high LDS licenses only aggregate predictions
      about unseen subsets: many different per-source vectors produce nearly
      the same subset sums, and two sources that always co-occur in the design
      can trade value freely without changing a single prediction.
check: choice
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

  Which explanation connects fact 1 to fact 2 correctly, and draws the right
  consequence for presenting the table as a basis for payment?
options:
  - text: |
      Dedup chose which source the model learns the shared material from.
      Source 4 kept the copies, so removing it loses that material and
      $\Delta_4$ absorbs it; removing source 9 loses only its unique material,
      and at fixed token budget its surviving tokens partly displace more
      useful ones, which is how $\Delta_9$ goes negative. Had the pipeline
      kept source 9's copies the two numbers would largely swap, with nothing
      about the corpus, the sources, the model or the eval set changed. So the
      deduplication policy is part of the result and must ship with the table:
      the honest framing is source 4's contribution UNDER this policy.
    correct: true
    explain: |
      Right. A choice every engineer treats as hygiene moved a large fraction
      of measured value between two rightsholders, and the recorded cluster is
      what makes that auditable rather than invisible - without it the table
      looks like a measurement of two sources and is in part a measurement of
      a sort order.
  - text: |
      The measurement found the weaker source. A negative $\Delta_9$ clearing
      the noise floor is the sweep telling you source 9 is low-quality or
      corrupted material that actively damages the model, so the right
      response is to inspect a sample of its documents and drop it from the
      corpus, after which $\Delta_4$ stands as source 4's true contribution.
    misconception: V4-M3
    explain: |
      Read a hundred of source 9's documents and you will find nothing wrong -
      it is a clean journal mirror. A property of the data cannot depend on
      which other source is present, and remove source 4 as well and source
      9's sign flips positive. The negative delta says its marginal token was
      worth less than the token that replaced it, in this corpus, at this
      budget.
  - text: |
      Dedup is upstream cleanup, so it cannot affect what the counterfactual
      measures: the model was trained on whatever the pipeline emitted, and
      the two deltas are exact differences between actual runs. The real fix
      is to dedup harder - collapse the cluster more aggressively so no shared
      passage survives twice - and then the table needs no policy footnote.
    misconception: D9
    explain: |
      Deduplication is a payout decision, not hygiene. Keeping one copy of a
      shared passage assigns credit for everything it teaches, and more
      aggressive dedup makes more of those assignments, not fewer. The deltas
      are exact for the runs performed and those runs were built on the
      policy, which is why the policy is part of the result.
  - text: |
      The pair's joint value is fixed and dedup only moved value inside it:
      $\Delta_4 + \Delta_9 = 0.028 - 0.006 = 0.022$ is what sources 4 and 9
      contributed together, so pay out against that total and the dedup
      decision washes out of the settlement.
    misconception: V4-M1
    explain: |
      Leave-one-out deltas do not sum to a joint effect whenever sources
      interact, and near-duplicates are the case where they fail hardest -
      remove both and the loss can jump several times the sum of the two
      individual deltas. The instinct to split the shared material's credit is
      right; the arithmetic that does it fairly is v5's coalition machinery.
check: choice
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

  That list is reproducibility metadata. Which set of additions supplies what
  is needed to INTERPRET the numbers, with the misreading each one prevents?
options:
  - text: |
      Add four things. $\sigma$ and the seed count $n$, without which there is
      no $\text{SE}_\Delta = \sigma\sqrt{2/n}$ and a 0.001 row reads as a small
      contributor rather than as indistinguishable from zero. The protocol -
      fixed corpus or fixed token budget - without which a large source's
      large number cannot be told from its volume, and which reorders the
      table rather than shifting it. The held-out set, chosen and
      decontaminated first, without which a zero reads as contributed nothing
      rather than nothing this yardstick can see. The dedup policy and cluster
      record, without which a delta inflated by winning a tiebreak reads as a
      property of the source.
    correct: true
    explain: |
      Right, and note what each addition is for: reproducibility metadata lets
      someone rerun the experiment, interpretation metadata lets them read the
      result, and only the second is needed to send an invoice.
  - text: |
      Add the per-source point estimates in rank order and a methodology note
      saying training is stochastic. The error bars themselves are a
      scientific courtesy that clutters a payment document - the ordering of
      the sources survives the noise even where the magnitudes do not, so the
      ranking is the interpretable part.
    misconception: D16
    explain: |
      For the rows below the floor the ordering is precisely what does not
      survive: their relative positions are set by noise, so a reseeded sweep
      permutes them. Publishing that order as information is the failure this
      unit exists to prevent, and it is the one that assigns money to noise.
  - text: |
      Add $\sigma$, $n$, and the held-out set. The protocol and the dedup
      policy are implementation detail rather than interpretation: the
      counterfactual was computed by actually retraining, not estimated, so
      $\Delta_i$ is a property of the source that holds however the corpus was
      assembled and at whatever scale it was measured.
    misconception: V4-M4
    explain: |
      The counterfactual is exact for the configuration that produced it -
      this architecture, parameter count, token budget, protocol, held-out set
      and source list - and that scope statement is the strongest thing the
      method has, not a weakness. Fixed corpus versus fixed budget reorders
      the table in favour of large sources, and the dedup survivor decision
      moves value between rightsholders.
  - text: |
      Add the $z$ column and the threshold used. Rows with $|z| > 2$ are
      established at $p < 0.05$ and can be reported as measured contributions;
      rows below the bar failed the test, so drop them from the table
      entirely, which keeps the header block short and the document clean.
    misconception: D19
    explain: |
      Two problems. Clearing a threshold across twelve rows is not proof - the
      test, the threshold and the multiple-comparison correction have to be
      committed before the sweep for the $z$ to mean anything. And the dropped
      rows carry the most valuable sentence in the document: these sources are
      not distinguishable from zero, and the smallest effect this sweep could
      resolve is $2\sigma\sqrt{2/n} = 0.005$ bpb.
check: choice
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
