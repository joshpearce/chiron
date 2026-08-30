---
unit: v3
title: "Training runs you can afford"
concepts:
  - d-compute-budget
  - d-training-recipes
  - d-data-constrained
  - d-eval-noise
assumes:
  - d-dedup-provenance
  - d-mixing-packing
---

Every attribution claim in this book is a claim about a difference between two
trained models. That makes training runs the unit of currency, and it makes
three questions load-bearing: what does a run cost, what do the knobs actually
do, and when is a difference between two runs real rather than luck. By the end
of this unit you can budget any run in one line of arithmetic, read a loss curve
without superstition, and compute whether a measured gap between two conditions
survives seed noise. Those three skills are what make every experiment in v4
through v8 plannable instead of aspirational.


## The receipt

Three real line items from 2026. Each one is a complete pretraining run,
executed by an individual on rented hardware, with a published invoice.

- **(a)** A 10-million-parameter model trained on 1 billion tokens.
- **(b)** A 561-million-parameter chat model, tokenizer through pretraining
  through supervised fine-tuning through a working web UI - the whole pipeline,
  end to end, one command.
- **(c)** A replication of GPT-2 XL, 1.6 billion parameters.

```beat
id: v3-b1
type: predict
concept: d-compute-budget
prompt: |
  Before reading on, commit to three numbers. Put a dollar cost on (a), (b),
  and (c) above - to within an order of magnitude is fine. Then answer the
  question that matters more: what would you need to know to *compute* those
  numbers rather than guess them? List the inputs.
answer: |
  As of mid-2026, on a rented 8xH100 node: (a) pennies, about 20 seconds of
  wall clock. (b) $92, measured, 3 hours 51 minutes end to end. (c) $672,
  measured, 24 hours.

  The inputs you need are exactly four, and only four:

  1. The number of parameters in the model.
  2. The number of tokens it will be trained on.
  3. How many floating-point operations per second the hardware actually
     sustains (not its spec sheet peak).
  4. The hourly rental price of that hardware.

  Notably absent from that list: the architecture, the optimizer, the dataset,
  the quality of the result, and how good you are at this. None of them enter
  the cost. That is the whole point of the next two sections.
rubric: |
  The dollar guesses are not graded - they are a commitment device, and being
  wrong is the expected outcome. Grade only the second half.
  Pass requires the input list to contain, in any wording: (1) model size /
  parameter count, (2) amount of training data / token count, (3) some notion
  of hardware throughput or speed, (4) a price per unit time.
  Three of four = pass. Listing only "GPU hours" or "how long it takes"
  without decomposing into size and data = fail; that is the guess restated,
  not a budget.
  An answer that includes dataset quality, architecture choice, or model
  capability as a cost input = fail, and diagnose V3-M1. That is the belief
  this section exists to break.
```

Here are the invoices. As of mid-2026, an 8xH100 node rents for $12-16/hour on
spot and $20-32/hour on demand, and sustains about 3.2 PFLOP/s on real training
work.

| What | Wall clock | Cost |
|---|---|---|
| 10M params, 1B tokens | ~20 seconds | pennies |
| 30M params, 3B tokens | ~3 minutes | under $1 |
| 124M params (GPT-2 small), 10B tokens | ~40 minutes | $9-16 |
| 561M params, 11.2B tokens, full pipeline | 3h51m | **$92 measured** |
| 1B params, 20B tokens | ~10.5 hours | $150-260 |
| 1.6B params (GPT-2 XL) | 24 hours | **$672 measured** |

<!-- refutes: V3-M1 -->
**You probably think cost tracks quality.** Not crudely - you know a bigger
model costs more. But the working belief is that the price of a training run is
somehow a price for the *result*: that a good model is expensive because it is
good, that the $92 in row four is buying a coherent chat model, and that if your
data were worse or your recipe sloppier the same run would somehow be cheaper or
the good outcome would cost more.

**Here is the prediction that fails.** Take the 561M row and train it twice:
once on a carefully filtered corpus and once on unfiltered scrape of identical
token count. One produces a model that answers questions; the other produces
gibberish with a plausible-looking loss curve. Both cost exactly $92, to the
cent, and both finish at the same minute. Now take the 124M row and run it with
a broken data loader that feeds the same shard 400 times. Same $9-16. The
invoice is blind to every property of the artifact.

**Here is why the wrong model is appealing.** Frontier training runs cost
hundreds of millions of dollars and produce the best models, so cost and quality
are correlated across the only examples that make the news. And in almost every
other engineering domain you have worked in, the expensive version is the good
version, because the money buys better components.

**Here is what is actually true.** A training run's cost is fully determined by
two integers you choose before it starts - how many parameters, how many tokens
- multiplied by a hardware constant and a market price. Nothing about the data,
the architecture, or the outcome enters. This has a consequence that reorganizes
the whole enterprise: **quality is free at fixed compute.** Two runs of identical
size and token count cost the same whether the corpus is excellent or worthless,
which is exactly why the filtering classifier is the highest-leverage component
in the pipeline and why the leading data benchmarks are won by data work rather
than by architecture work.

The second consequence is the one this book runs on. If cost depends only on
size and tokens, and you choose both, then you can choose them small enough that
an experiment costing 550 training runs is affordable. That is the entire reason
ground-truth attribution is reachable by one person with a credit card, and it is
the subject of section 3.

## One step, from scratch

Before budgeting a run, be precise about what a run is made of. A run is a loop
over *steps*, and a step is four things in sequence. This is a compact
re-derivation - if you already have it, you lose thirty seconds.

**Forward.** Take a batch of token sequences. Push them through the network,
producing at every position a score for every vocabulary entry, then a
probability distribution over the vocabulary.

**Loss.** At every position you know which token actually came next. Take the
probability the model assigned to that specific token, take its negative natural
logarithm, and average over all positions in the batch. That average is the loss
$\mathcal{L}$, in units of nats per token. It is the model's average surprise:
zero if it was certain and right every time, larger the more probability it put
somewhere else.

**Worked, on a batch of three positions.** A model with a four-entry vocabulary
processes three positions. At each position the true next token gets some
probability from the model: $0.5$, then $0.2$, then $0.1$.

$$\mathcal{L} = -\frac{1}{3}\left(\ln 0.5 + \ln 0.2 + \ln 0.1\right)
= -\frac{1}{3}\left(-0.693 - 1.609 - 2.303\right) = \frac{4.605}{3} = 1.535$$

1.535 nats per token. For scale: a uniform guess over a four-entry vocabulary
would score $\ln 4 = 1.386$, so this model is currently *worse than random* on
this batch, which is what an untrained network looks like from the inside.

**Backward.** For every parameter in the network, compute how much the loss
would change if that parameter changed slightly. That number is the parameter's
gradient. Backpropagation computes all of them in one sweep from the loss back
to the inputs, at a cost of roughly twice the forward pass.

**Update.** Move every parameter a small distance opposite its gradient. In the
plainest form, with learning rate $\eta$ (a small positive number, the step
size) and gradient $g$ for one parameter $\theta$:

$$\theta \leftarrow \theta - \eta\, g$$

With $\eta = 3 \times 10^{-4}$ and $g = 0.8$, that parameter moves by
$2.4 \times 10^{-4}$. Against a typical initialized weight of magnitude $0.02$,
that is about a 1% move, in one step, from one batch.

One practical note that explains why learning rates are quoted as bare numbers
and compared across models. Real runs do not use the plain rule above; they use
AdamW, which divides the gradient by a running estimate of its own magnitude
before stepping. The ratio it steps by is order 1 regardless of how large the
raw gradient was, so the actual distance each parameter moves per step is
approximately $\eta$ itself. That is why $3 \times 10^{-4}$ is a meaningful
thing to say out loud, and why gradient magnitude barely appears in a recipe.

That is the entire loop. Repeat it. A run is a step count, and the step count is
tokens divided by batch size. The 561M run above used a batch of $2^{19}
= 524{,}288$ tokens:

$$\frac{11.2 \times 10^9 \text{ tokens}}{524{,}288 \text{ tokens/step}} = 21{,}362 \text{ steps}$$

Twenty-one thousand repetitions of forward, loss, backward, update. Hold that
number; section 4 spends it.

```beat
id: v3-b2
type: compute
concept: d-compute-budget
prompt: |
  A model processes a batch of four positions. At those positions it assigns
  the token that actually occurred the probabilities $0.4$, $0.4$, $0.25$, and
  $0.1$.

  Compute the loss in nats per token. Give one number to two decimal places.

  ($\ln 0.4 = -0.916$, $\ln 0.25 = -1.386$, $\ln 0.1 = -2.303$)
answer: 1.38
check: numeric(0.01)
```

## The one line: C = 6ND

<!-- fade: v3-compute-budget -->

Now count the arithmetic in that loop, and the whole cost table falls out.

Take one parameter and one token. In the forward pass, that parameter is used
once: it multiplies an activation and the product is added into a running sum.
One multiply plus one add is 2 floating-point operations.

The backward pass has to produce two different things for that same parameter.
It needs the gradient with respect to the parameter itself (so it can be
updated), and it needs the gradient with respect to the activation that fed into
it (so blame can keep flowing backward to earlier layers). Each of those is
another multiply-accumulate, so each costs 2 operations, and the backward pass
costs 4.

Total per parameter per token: $2 + 4 = 6$.

Multiply by the number of parameters and the number of tokens:

$$C \approx 6ND$$

where $C$ is the total floating-point operations for the run, $N$ is the number
of parameters in the model, and $D$ is the number of tokens it is trained on.
That is the whole formula. Three symbols, one constant, no calculus.

Two honest caveats, declared here rather than discovered later. First, attention
does work that is not proportional to parameter count - comparing every query
against every key is quadratic in sequence length - so $6ND$ genuinely
undercounts the operations a run performs. The fraction it misses is roughly
sequence length divided by twelve times model width, which is under 2% for a
frontier-width model and closer to 15% for the narrow models in this volume. It
does not corrupt any budget in this unit, and the reason is worth understanding
rather than hand-waving: the throughput figure you divide by is not a hardware
measurement. It is the *effective* rate that makes $6ND$ reproduce observed wall
clocks, so the attention term is already inside it. Keep the pairing intact -
$6ND$ with an effective throughput fitted at similar context length - and the
arithmetic is honest. Change context length by an order of magnitude and refit.
Second, the 6 assumes every parameter participates in every token. Mixture-of-
experts models route each token to a fraction of the network, and for those you
substitute *active* parameters for $N$.

**Now use it.** The point of this formula is that it is one line, so use it
constantly.

**Budget 1: the $92 run.** $N = 5.61 \times 10^8$ parameters,
$D = 1.12 \times 10^{10}$ tokens.

$$C = 6 \times (5.61 \times 10^8) \times (1.12 \times 10^{10}) = 3.77 \times 10^{19} \text{ FLOPs}$$

The node sustains $3.2 \times 10^{15}$ FLOP/s:

$$t = \frac{3.77 \times 10^{19}}{3.2 \times 10^{15}} = 11{,}780 \text{ s} = 3.27 \text{ hours}$$

The published wall clock for the full pipeline is 3 hours 51 minutes. Our line
of arithmetic accounts for 3.27 of those 3.85 hours - 85% of the run - and the
missing 35 minutes is tokenizer training, chat-format midtraining, supervised
fine-tuning, and evaluation, none of which is pretraining. At $24/hour on
demand, 3.85 hours is $92. The receipt reconstructs.

**Budget 2: the laptop, overnight.** A MacBook training under MLX sustains
roughly $10^{13}$ FLOP/s - ten teraflops, about one three-hundredth of the
rented node. Take $N = 10^7$ parameters and $D = 2 \times 10^8$ tokens:

$$C = 6 \times 10^7 \times (2 \times 10^8) = 1.2 \times 10^{16} \text{ FLOPs}
\qquad t = \frac{1.2 \times 10^{16}}{10^{13}} = 1{,}200 \text{ s} = 20 \text{ minutes}$$

Twenty minutes per run, on hardware you already own, with no invoice at all.

**Budget 3: the leave-one-source-out sweep.** This is the one the rest of the
book depends on. You have a corpus tagged into $K = 20$ sources. Ground truth for
"what did source $i$ contribute" is: train one model on everything, then train
20 more, each with one source removed, and compare. That is 21 runs.

On the laptop, at 20 minutes each: $21 \times 20 = 420$ minutes = 7 hours. One
night.

On a rented node the same sweep goes bigger. Take $N = 3 \times 10^7$ and
$D = 5 \times 10^8$ per run:

$$C = 6 \times (3 \times 10^7) \times (5 \times 10^8) = 9 \times 10^{16} \text{ FLOPs}$$

One H100 sustains about $4 \times 10^{14}$ FLOP/s (the node's 3.2 PFLOP/s split
across 8 cards), so one run is $9 \times 10^{16} / 4 \times 10^{14} = 225$
seconds, under 4 minutes. Now you can afford the *serious* version: 1 full model,
20 leave-one-out models, and 500 models trained on random subsets of the sources
- 521 runs, call it 550 with reruns.

$$550 \times 225 \text{ s} = 123{,}750 \text{ s} = 34 \text{ GPU-hours}$$

At $1.50-4.00 per GPU-hour as of mid-2026, that is **$51-136**. The complete
counterfactual ground truth for a 20-source corpus, for the price of a dinner.

Sit with that number, because it is the strategic core of this entire book. The
attribution literature treats retraining-based ground truth as the thing everyone
approximates and nobody can afford. At 30M parameters it costs less than a
hundred dollars, and the reason is one line of arithmetic: cost is linear in
$N$, so shrinking the model by a factor of 20 shrinks the whole research program
by a factor of 20.

```beat
id: v3-b3
type: completion
concept: d-compute-budget
# variants: blank the FLOP/s and the cost instead of the FLOPs and the hours;
# or give the wall clock and blank D.
prompt: |
  Budget a 124M-parameter model trained on 10 billion tokens, on the 8xH100
  node ($3.2 \times 10^{15}$ FLOP/s sustained, $24/hour as of mid-2026).

  Step 1. $C = 6 \times (1.24 \times 10^8) \times (10^{10}) =$ ____ FLOPs
  Step 2. $t = C \,/\, (3.2 \times 10^{15}) =$ ____ seconds
  Step 3. In minutes: ____
  Step 4. Cost at $24/hour: ____

  Fill the four blanks. Then state, in one sentence, what would change about
  every one of these four numbers if the corpus were replaced with one of
  identical token count but far worse quality.
answer: |
  Step 1: $7.44 \times 10^{18}$ FLOPs
  Step 2: 2,325 seconds
  Step 3: 38.75 minutes, call it 40
  Step 4: 0.646 hours x $24 = $15.50

  Nothing would change. Not one of the four numbers. $C = 6ND$ contains no
  term for data quality, and neither does the throughput or the price. The
  worse corpus produces a worse model on exactly the same schedule for exactly
  the same money.
rubric: |
  Required, to within 2%: blank 1 = 7.44e18; blank 2 = 2,325 s (accept
  2,300-2,350); blank 3 = ~39 minutes (accept 38-40); blank 4 = ~$15.50
  (accept $15-16).
  The one-sentence answer must say that none of the numbers change, and cite
  that C = 6ND has no data-quality term.
  All four numbers plus the correct final sentence = pass.
  An answer that says the worse corpus would need more tokens or more steps to
  reach the same loss = fail, and diagnose V3-M1. That may be true of the
  QUALITY you get, but the question fixed the token count; the budget is
  determined by the integers you chose, not by the outcome you wanted.
```

```beat
id: v3-b4
type: compute
concept: d-compute-budget
prompt: |
  You have a corpus tagged into $K = 40$ sources and you want counterfactual
  ground truth: one model trained on the full corpus, plus one model per
  source with that source removed.

  Each run is $N = 2 \times 10^7$ parameters on $D = 3 \times 10^8$ tokens,
  executed on one H100 sustaining $4 \times 10^{14}$ FLOP/s.

  How many GPU-hours does the complete sweep take? Give one number to two
  decimal places.
answer: 1.03
check: numeric(0.02)
```

## Knobs are recipes, not theory

The training loop has a handful of knobs. They are widely mystified, and there
is nothing to mystify: each one is a recipe with a measured effect and a stated
reason. Here they are, in the order you will set them.

**Learning rate, and why it warms up.** $\eta$ is the step size from section 2.
At step 0 the network is random, gradients are large and mutually inconsistent,
and AdamW's running estimates of gradient magnitude - the thing it divides by -
are built from almost no data and are therefore garbage. A full-size step under
those conditions can put the model somewhere it never recovers from. So runs
*warm up*: $\eta$ rises linearly from 0 to its peak over the first 1-2% of
steps. For the 21,362-step run from section 2, a 2% warmup is 427 steps. That is
the entire justification. It is a recipe.

**Cosine decay, and the failure it causes.** After warmup, the standard schedule
decays $\eta$ smoothly to near zero following a cosine curve over the remaining
steps. Large steps early to travel, small steps late to settle. It works, it is
the default in every reference implementation, and it has one property that
matters enormously for the experiments in this book:

*The cosine schedule needs to know the total step count in advance, and every
checkpoint it produces is mid-decay.* You cannot extend a cosine run - the
learning rate is already at the bottom. You cannot branch from the middle of one
and get a finished model, because a model pulled from step 12,000 of a
21,362-step cosine run has a learning rate that was on its way somewhere. Under
cosine, every experimental condition costs a full run from scratch.

**Warmup-stable-decay, and what it buys.** WSD replaces the cosine with three
phases: warm up, then hold $\eta$ *constant* for the bulk of the run, then decay
it over a short tail, typically the last 10-20% of steps. Final quality is
comparable to cosine. What changes is that the stable phase runs at a constant
learning rate, so every checkpoint in it is a legitimate branch point: copy it,
run only the decay tail on whatever variant you want, and you have a finished,
comparable model for 10-20% of a run.

Count what that does to the sweep in section 3. Twenty leave-one-source-out
conditions:

- Under cosine: $1 + 20 = 21$ full runs.
- Under WSD with a 15% decay tail: one shared trunk, plus $20 \times 0.15 = 3$
  run-equivalents, for 4 total. **Five times cheaper.**

And now the caveat, because this is the kind of shortcut that quietly invalidates
a result. A WSD-branched leave-one-out model *saw the removed source during the
trunk*. It answers "what does this source contribute over the last 15% of
training," not "what would this model be if this source had never existed." Those
are different counterfactuals and they can disagree. Use branching for cheap
screening, and pay for from-scratch runs for any number you intend to defend;
v4 makes the distinction precise and shows what it does to the measured
contributions.

**Batch size and learning rate move together.** A larger batch averages more
examples, so its gradient estimate is less noisy, so you can afford a bigger
step. The heuristics: double the batch and double $\eta$ (linear scaling) for
plain SGD; double the batch and multiply $\eta$ by $\sqrt{2}$ for Adam-family
optimizers. Both stop working past a critical batch size where the gradient is
already about as accurate as it is going to get and larger batches buy nothing
but memory pressure. The practical PoC rule: pick the largest batch that fits in
memory, then take $\eta$ from a published recipe at a similar batch size, and do
not tune it. Learning-rate tuning is the least profitable hyperparameter search
available to you.

**Muon, as a named fact.** AdamW normalizes each parameter's step by that
parameter's own gradient history. Muon does something structurally different for
the two-dimensional weight matrices: it takes the momentum matrix and
orthogonalizes it before stepping, so the update pushes in many directions at
comparable magnitude rather than being dominated by a few. It is a drop-in
replacement for AdamW on matrix-shaped parameters, it is measurably faster per
unit of progress, and that is all you need to know to use it.

The measured effect is worth citing precisely because it demystifies. The
`modded-nanogpt` speedrun trains a 124M-parameter model to a fixed validation
loss of 3.28 on FineWeb. In May 2024 that took 45 minutes. As of July 2026 it
takes 1.23 minutes. That is a factor of roughly 35 in 26 months, and it came
from a stack of ordinary changes - Muon, query-key normalization, a squared-ReLU
activation, rotary position encoding, FP8 arithmetic, FlashAttention-3,
multi-token prediction - no single one of which is magic. Recipes compound.

<!-- refutes: V3-M2 -->
**You probably think a descending loss curve means the run is working.** You
watch the curve, it goes down smoothly, you relax. If it were flat or spiky you
would intervene; it is neither, so the run is healthy and the data is fine.

**Here is the prediction that fails, twice.** First: under a cosine schedule,
loss drops noticeably in the last 10% of *every* run, including runs on
worthless data, because the learning rate is going to zero and the model is
settling into whatever basin it is in. If the late-run drop were evidence of
learning, it would not appear identically in a run you deliberately sabotaged.
It does. Second, and worse: take a corpus in which a data-loader bug repeats the
same shard four hundred times. The loss curve is smooth, monotone, and lower
than the correct run's, because the model is faithfully learning the boilerplate
and is being tested on more of the same. The prettiest curve in your logbook can
belong to the most broken run.

**Here is why the wrong model is appealing.** The loss curve is the only thing
that updates in real time. It is on the screen for three hours, it looks like a
progress bar, and progress bars are trustworthy. Also, when a run genuinely
breaks - divergence, a NaN, an optimizer blowup - the curve does show it
immediately, which builds a real and justified habit of watching it.

**Here is what is actually true.** The training loss curve is a diagnostic for
*optimizer health* and nothing else. Spikes, plateaus at initialization value,
divergence, sudden NaNs: all real signals, all worth acting on. Data quality,
model quality, and whether one condition beats another: none of these are
readable from it. Those come from held-out bits-per-byte on text you selected
before the run started, compared across multiple seeds, which is section 7. A
training curve compares a model to its own past. Every claim in this book
compares a model to a different model.

```beat
id: v3-b5
type: self-explain
concept: d-training-recipes
prompt: |
  You are planning the leave-one-source-out sweep and a colleague proposes:
  "Train one model with WSD, keep a checkpoint from the middle of the stable
  phase, then branch 20 times - once per source removed - and run only the
  decay tail on each. Twenty conditions for the price of four runs."

  In your own words: explain what makes this technically possible, then state
  precisely what question the resulting numbers answer and what question they
  do NOT answer. Say when you would use it anyway.
answer: |
  Why it works: during the stable phase the learning rate is constant, so no
  checkpoint in that phase is partway through a schedule. It is a legitimate
  starting state. A cosine run has no such checkpoint - every one of them is
  mid-decay, so a branch inherits a learning rate that was on its way to zero
  and the resulting model is not comparable to a full run.

  The economics are right: with a 15% decay tail, 20 branches cost 20 x 0.15 =
  3 run-equivalents plus the one shared trunk, about 4 total instead of 21.

  What the numbers answer: how much source i contributes to the model's final
  loss GIVEN that the model already trained on source i for 85% of the run.
  That is a real quantity and it is measurable.

  What they do not answer: what the model would be if source i had never been
  in the corpus at all. The trunk saw source i. Whatever it learned from it -
  vocabulary, domain structure, facts - is baked into the shared branch point
  and is present in all 20 variants including the one that supposedly excludes
  it. The branched estimate is systematically biased toward zero contribution,
  and by an unknown amount.

  When to use it anyway: screening. If you have 200 candidate sources and want
  to find the 20 worth measuring properly, branch-and-decay is the right tool -
  the bias is roughly shared across conditions so the ranking is informative
  even when the magnitudes are not. Then pay for from-scratch runs on the
  survivors, for any number you intend to put in front of someone.
rubric: |
  Must contain: (1) the stable phase has constant learning rate, which is what
  makes a mid-run checkpoint a valid starting state, (2) the branched variants
  all inherit the trunk's exposure to the removed source, so the counterfactual
  is "contribution over the tail" not "contribution overall", (3) some
  acknowledgement that this biases the measured contribution downward or makes
  it not comparable to a from-scratch leave-one-out.
  (1) and (2) = pass. (3) upgrades to full credit.
  An answer that endorses the shortcut with no caveat = fail. The whole point
  of v4's ground truth is that it is a from-scratch counterfactual; a learner
  who will substitute a branched estimate for it without flagging the
  substitution will produce a number that cannot survive review.
  An answer that rejects branching entirely as invalid = partial. It is a
  legitimate screening tool and refusing it costs real money.
```

## What repeating data actually costs

<!-- fade: v3-epoch-arithmetic -->

Your tagged corpus is finite. The papers corpus for this project is a few
hundred million tokens, and section 3's budgets casually specified runs of 500
million or 1.6 billion tokens. Where do the extra tokens come from? They come
from reading the same corpus more than once. An *epoch* is one full pass through
the corpus, and the question is what the second, fourth, and fortieth passes are
worth.

<!-- refutes: D17 -->
**You probably think repeating data is cheating, or at best worthless.** Fresh
tokens carry new information; a token you have already trained on carries none,
so a second epoch is at best a no-op and at worst teaches the model to memorize
its own training set. Under this belief a 500-million-token corpus supports a
500-million-token run and nothing more, and any experiment needing more data
needs more data.

**Here is the prediction that fails.** If repeated tokens were worthless, a model
trained for 4 epochs on 500M unique tokens would land at the loss of a model
trained on 500M tokens, not near the loss of a model trained on 2B fresh tokens.
It has been measured directly and carefully in the data-constrained scaling
work, and the result is the opposite: **up to about 4 epochs, repeated tokens
are worth very nearly as much as fresh ones.** Returns diminish steadily out to
roughly 16 epochs, where a repeated token is worth meaningfully less than a
fresh one but still positive. Past about 40 epochs, additional passes contribute
essentially nothing and the run stops improving.

**Here is why the wrong model is appealing.** Overfitting folklore from
supervised learning, where you really do watch validation loss turn upward after
a few epochs on a small labeled dataset, and where "more epochs" is a classic
beginner error. That regime is real. It is not this regime: language modeling on
hundreds of millions of tokens with a model of tens of millions of parameters is
nowhere near the capacity ratio where classical overfitting bites.

**Here is what is actually true, with the arithmetic.** Three thresholds, and
the whole calculation is a division.

Take a corpus of 400 million unique tagged tokens.

$$\text{tokens available at } E \text{ epochs} = E \times 400\text{M}$$

- 4 epochs, the nearly-free zone: **1.6B tokens.**
- 16 epochs, the diminishing zone: 6.4B tokens.
- 40 epochs, the dead zone: 16B tokens, and the last several billion bought you
  nothing.

Now check that against a run you actually want. A 30M-parameter model trained on
1.6B tokens is at $1.6 \times 10^9 / 3 \times 10^7 = 53$ tokens per parameter.
Chinchilla's compute-optimal ratio is about 20 tokens per parameter, so the free
zone alone lets you train a 30M model to 2.7 times past compute-optimal. At PoC
scale **you are not data-constrained at all** - the corpus is ample and the
binding constraint is your patience.

Flip the question to find where the constraint does bite. What is the largest
model 400M unique tokens supports at compute-optimal ratio without leaving the
free zone?

$$N_{\max} = \frac{4 \times 400\text{M}}{20} = 80\text{M parameters}$$

Past 80M parameters, a 400M-token corpus forces you to choose: repeat harder and
accept diminishing returns, or mix in background data. The realistic-scale
recipe in this project's plan does the latter - roughly 70% general web text and
30% tagged papers, 11.2B tokens over about 3B unique, which is 3.7 epochs, right
at the edge of the free zone by construction.

One more thing about this knob, and it is the reason it appears again later.
Repetition is also the primary driver of *memorization*. The more times a
document passes through training, the more extractable it becomes from the
finished model. So the epoch count is a single dial that simultaneously sets how
much capability you wring out of a small corpus and how much of that corpus
becomes recoverable evidence that it was trained on. In v3 that dial is a budget
parameter. In v6, where the question becomes what a model provably remembers and
what counts as evidence of training, it is the same dial pointed at a different
outcome. Set it deliberately.

```beat
id: v3-b6
type: completion
concept: d-data-constrained
# variants: blank the epoch count and give the token target; or blank N_max
# and give the ratio.
prompt: |
  You have 250 million unique tagged tokens and want to train a model on a
  budget of 1 billion tokens.

  Step 1. Epochs required: $10^9 / (2.5 \times 10^8) =$ ____
  Step 2. Which zone is that in - nearly free (up to ~4), diminishing (to
          ~16), or dead (past ~40)? ____
  Step 3. At 20 tokens per parameter, the largest model this run is
          compute-optimal for: $10^9 / 20 =$ ____ parameters
  Step 4. If you instead insisted on one epoch only, your token budget would
          be ____ and the compute-optimal model size would be ____ parameters.

  Fill the blanks. Then state in one sentence what the epoch limit costs you
  in model size, comparing step 3 to step 4.
answer: |
  Step 1: 4 epochs
  Step 2: Nearly free - right at the edge of the zone where a repeated token
          is worth about as much as a fresh one.
  Step 3: 50,000,000 parameters (50M)
  Step 4: 250M tokens, supporting a 12,500,000-parameter (12.5M) model.

  The one sentence: staying inside the free repetition zone lets the same
  fixed corpus support a model four times larger, because the free zone
  multiplies the effective token budget by 4 and compute-optimal model size is
  linear in tokens.
rubric: |
  Required, exactly: blank 1 = 4; blank 2 = nearly free / free zone; blank 3 =
  50M (50,000,000); blank 4 = 250M tokens and 12.5M parameters.
  The final sentence must state the 4x factor in supportable model size and
  tie it to the free repetition zone.
  All blanks plus the factor = pass.
  An answer that marks step 2 as "diminishing" or "overfitting territory" =
  fail, diagnosing D17; 4 epochs is the measured boundary of the nearly-free
  zone, not the start of trouble.
  An answer that computes the blanks correctly but claims one epoch is
  methodologically safer or more honest = fail; it is neither, it is just a
  four-times-smaller experiment.
```

## The laptop and the node

You have two machines and they differ by a factor of about 320. Knowing exactly
what each one can do turns "can I run this experiment" into a division.

**The node.** Eight H100s. Their combined dense BF16 peak is about 7.9 PFLOP/s
on the spec sheet. A well-tuned training run sustains about 3.2 PFLOP/s. The
ratio of those two numbers is the run's **Model FLOPs Utilization**:

$$\text{MFU} = \frac{\text{FLOP/s you actually get}}{\text{FLOP/s the hardware peaks at}} = \frac{3.2}{7.9} \approx 40\%$$

Forty percent is a good number for a real training run. Note carefully what the
numerator is: MFU counts *model* FLOPs, meaning $6ND$ divided by elapsed time,
not every operation the hardware performed. So the missing 60% is where all the
work that $6ND$ does not count ends up - memory bandwidth stalls, communication
between cards, kernel launch overhead, and the attention term from section 3.
This is why the pairing works. A budget built from $6ND$ and a throughput fitted
the same way is self-consistent even though neither number is the whole truth.

**The laptop.** A MacBook training under MLX sustains roughly 10 TFLOP/s. Its
inference performance is excellent and not the point here; sustained *training*
throughput is about $1/320$ of the node's.

Put MFU into the budget line and it becomes a complete answer:

$$t_{\text{hours}} = \frac{6ND}{\text{MFU} \times \text{peak FLOP/s} \times 3600}$$

<!-- refutes: V3-M3 -->
**You probably think MFU is a vanity metric.** Something infrastructure teams
put on slides to demonstrate that their kernels are good - a leaderboard number,
adjacent to the actual work, the sort of thing that gets optimized because it is
measurable rather than because it matters.

**Here is the prediction that fails.** If MFU were decorative, changing it would
not change anything you care about. Run the $92 job at 20% MFU instead of 40%:
the model is bit-identical, the loss curve is identical, the evaluation is
identical, and the invoice reads $184. Nothing about the artifact changed and
the price doubled. A metric that halves your budget when it halves is not
decorative.

**Here is why the wrong model is appealing.** MFU is reported alongside genuinely
vain numbers - peak TFLOPs, parameter counts, benchmark leaderboards - and it is
usually reported by the people whose work it evaluates. And in most software you
have written, utilization percentages really were diagnostic curiosities rather
than line items.

**Here is what is actually true.** $C = 6ND$ gives you floating-point
operations. Operations are not hours and hours are not dollars. MFU is the
exchange rate between them, and it is the only term in the budget that
engineering effort can move. You cannot make the model smaller without changing
the experiment or make the tokens fewer without changing the result, but you can
double your throughput and halve your bill. For this project specifically: at
550 runs, a 40% MFU sweep costs $51-136 and a 20% MFU sweep costs $102-272. Same
science, different budget.

**The overnight ceiling, computed.** Eight hours on the laptop:

$$C = 8 \times 3600 \times 10^{13} = 2.88 \times 10^{17} \text{ FLOPs}
\qquad ND = \frac{C}{6} = 4.8 \times 10^{16}$$

That single product is the entire overnight envelope, and you spend it however
you like:

| Model | Tokens | Tokens/param |
|---|---|---|
| 10M params | 4.8B | 480 |
| 30M params | 1.6B | 53 |
| 50M params | 0.96B | 19 |

So the honest statement of what a laptop is for: **10 to 50 million parameters
on 1 to 3 billion tokens, overnight.** Not a toy budget. The 30M row sits at 53
tokens per parameter, well past Chinchilla-optimal, which means these models are
genuinely trained rather than undertrained stubs - a 30M model at 1.6B tokens
has converged in the sense that matters, and its held-out loss is a real
measurement of the corpus it saw.

That scale is not a compromise. It is precisely the scale at which ground-truth
attribution is affordable, because everything in v4 through v8 costs $K+1$ runs
or 550 runs rather than one, and only linear-in-$N$ costs make that arithmetic
survivable. A researcher who insists on working at 7B parameters cannot retrain
even once for an ablation, has no ground truth, and therefore cannot know
whether any attribution method they implement is correct. You can. That is the
trade, and it is a good one.

What you give up is stated plainly in the next section: at this scale, most
benchmarks are noise.

```beat
id: v3-b7
type: predict
concept: d-compute-budget
prompt: |
  Your overnight laptop envelope is $ND = 4.8 \times 10^{16}$. You are choosing
  between two configurations for the counterfactual sweep:

  (i) 50M parameters on 960M tokens, one run per night.
  (ii) 10M parameters on 200M tokens, 24 runs per night.

  Before reading on: which one do you pick for the leave-one-source-out ground
  truth over 20 tagged sources, and why? Then state the one thing configuration
  (ii) makes worse.
answer: |
  Pick (ii), and it is not close. Ground truth requires 21 runs minimum
  (full plus 20 leave-one-out), and the whole enterprise wants 500-plus
  random-subset runs on top of that for the regression in v4. Configuration
  (i) delivers 21 runs in three weeks; configuration (ii) delivers them in one
  night and the full 550-run program in about three weeks. The experiment is
  measured in runs, not in model quality, so throughput in runs is the only
  axis that matters.

  What (ii) makes worse: the model. A 10M-parameter model at 20 tokens per
  parameter is undertrained AND small, so its held-out loss is high, its
  behavior is poor, and the differences you are trying to measure between
  conditions are smaller in absolute terms and sit on top of proportionally
  larger seed noise. There is a real floor below which the contributions you
  are measuring drop under the noise, and the next section is about finding it.

  The practical resolution: something in between. 20-30M parameters at
  400-800M tokens keeps runs at 30-60 minutes, stays past Chinchilla ratio, and
  still lands the sweep in a few nights.
rubric: |
  Pass requires: (1) choosing (ii) or an intermediate configuration, (2) the
  reason being that the experiment's cost is measured in NUMBER OF RUNS
  because ground truth is a counterfactual sweep, (3) naming the cost as
  smaller effect sizes / worse signal relative to seed noise, or the model
  being undertrained.
  (1) and (2) = pass. Missing (3) = pass but flag: the learner has not yet
  connected model scale to the noise floor and section 7 should be delivered
  in full.
  An answer choosing (i) on the grounds that a better model gives more
  trustworthy attribution = fail. One model of any quality gives zero
  counterfactual information; the ground truth is a difference across runs and
  does not exist below 21 of them.
```

## Evaluation without self-deception

<!-- fade: v3-bpb-conversion -->

You have two models and you want to know which is better, or whether removing a
source hurt. This is where careful people produce confident nonsense, so the
discipline here is more important than any single technique in this unit.

**Loss is not comparable across tokenizers, and perplexity is worse.** The
training loss $\mathcal{L}$ is nats *per token*, and a token is whatever your
tokenizer says it is. Change the tokenizer and you change the denominator, which
changes the number without changing the model's actual command of the text.
Perplexity, $e^{\mathcal{L}}$, inherits the problem and amplifies it by
exponentiating.

**Bits per byte fixes it by changing the denominator to something objective.**
The held-out text has a length in UTF-8 bytes that was fixed before any
tokenizer existed. Divide total surprise by that, and convert nats to bits:

$$\text{bpb} = \frac{\mathcal{L}}{\ln 2 \times \text{bytes per token}}$$

where $\mathcal{L}$ is the model's mean loss in nats per token on the held-out
text and "bytes per token" is that text's byte count divided by its token count
under this model's tokenizer. Equivalently: total nats over the eval set divided
by $\ln 2$ times total bytes. Same number.

**Worked.** A model scores $\mathcal{L} = 2.30$ nats per token on held-out
papers, with a 65,536-entry tokenizer that averages 4.8 bytes per token on that
text.

$$\text{bpb} = \frac{2.30}{0.6931 \times 4.8} = \frac{2.30}{3.327} = 0.691$$

The model spends 0.691 bits per byte of held-out paper text.

**Now the contrast that shows why this matters.** Two models, evaluated on the
identical held-out file.

| | Loss (nats/token) | Bytes/token | Perplexity | bpb |
|---|---|---|---|---|
| Model A | 2.30 | 4.8 | 9.97 | **0.691** |
| Model B | 1.60 | 3.2 | 4.95 | **0.721** |

By perplexity, model B is twice as good as model A. By bits per byte - the only
one of these numbers that refers to the text rather than to a tokenizer - model
A is better. B's advantage was entirely an artifact of a smaller vocabulary
chopping the text into more, easier pieces. Predicting a shorter piece is an
easier problem, and perplexity rewards you for making the problem easier.

This is not a hypothetical trap. Any comparison between your domain model and a
public baseline crosses a tokenizer boundary, and every such comparison reported
in perplexity is meaningless. Report bits per byte. It is tokenizer-invariant,
it is monotone in model quality, it has low seed variance, and it works at every
model size including the 10M-parameter ones you can actually afford.

```beat
id: v3-b8
type: completion
concept: d-eval-noise
# variants: give bpb and blank the loss; or blank bytes-per-token given both
# ends.
prompt: |
  A model scores $\mathcal{L} = 2.60$ nats per token on a held-out set of
  papers. Its tokenizer averages 3.5 bytes per token on that text.

  Step 1. Bits per token: $2.60 / \ln 2 = 2.60 / 0.6931 =$ ____
  Step 2. Bits per byte: ____ $/\ 3.5 =$ ____
  Step 3. A second model, different tokenizer, scores 2.20 nats per token at
          4.1 bytes per token. Its bpb: ____

  Fill the blanks, then state which model is better on this text and what the
  raw loss numbers alone would have told you.
answer: |
  Step 1: 3.752 bits per token
  Step 2: 3.752 / 3.5 = 1.072 bits per byte
  Step 3: 2.20 / (0.6931 x 4.1) = 2.20 / 2.842 = 0.774 bits per byte

  The second model is better, at 0.774 bpb against 1.072.

  The raw losses would have said the same thing here (2.20 < 2.60), but only by
  luck - the second model has BOTH the lower loss and the longer tokens, so the
  two effects point the same way. Had the second model scored 2.90 nats at 4.1
  bytes per token, its bpb would be 1.020, still better than the first model's
  1.072, while its loss looked 12% worse. Loss comparisons across tokenizers
  are unsafe in both directions.
rubric: |
  Required, to within 1%: blank 1 = 3.75; blank 2 = 1.07; blank 3 = 0.77.
  The verdict must be that the second model is better.
  The final sentence must recognize that comparing raw losses across different
  tokenizers is invalid, whether or not it happens to agree here.
  All three numbers plus the verdict = pass; the "agreed by luck" observation
  upgrades to full credit.
  An answer that divides by ln 2 in the wrong direction (multiplying instead)
  = fail; check the direction by sanity: bits are SMALLER units than nats, so
  a value in bits is LARGER than the same value in nats.
```

**Now the harder half: is the difference real?**

<!-- refutes: D16 -->
**You probably think a source's measured contribution is a stable property of
that source.** Measure once, carefully, and you have a number. Better tooling
gives a more precise number, the way a better scale gives a more precise weight.
Under this belief the output of an attribution pipeline is a table of source
values, and the engineering task is to compute it accurately.

**Here is the prediction that fails.** If contribution were a property of the
source, retraining with a different random seed - same data, same code, same
hyperparameters, only the initialization and shuffle order differ - would
reproduce it. It does not. A single source's measured marginal contribution to
held-out loss can swing by more than its own magnitude across seeds. Per-document
membership decisions in the verification literature flip like coins under
training randomness alone. And of all the things that perturb influence
measurements, the *order* in which data arrives introduces the largest variation
- an ordering you did not choose deliberately and did not record.

**Here is why the wrong model is appealing.** Every instinct from deterministic
systems. Same inputs, same outputs; if the numbers differ, something is wrong
and should be fixed. Loss curves are smooth, final losses across seeds agree to
three decimal places, and the whole apparatus radiates determinism. It is only
the *differences between models* - the quantity you actually care about - that
are noisy, and differences of nearly-equal noisy numbers are where variance
lives.

**Here is what is actually true, with the arithmetic.** A source's contribution
is a random variable. You do not measure it; you estimate it, with an error bar,
and the error bar is frequently larger than the estimate. Here is the whole
procedure.

Train each condition with $n$ different random seeds and average. Let
$\sigma$ be the standard deviation of a single run's held-out bpb across seeds -
the *seed noise*, measured once for your setup by training the same
configuration several times. Then the standard error of one condition's
$n$-seed mean is $\sigma/\sqrt{n}$, and the standard error of the *difference*
between two conditions, each with $n$ seeds, is

$$\text{SE}_{\text{diff}} = \sigma\sqrt{\frac{1}{n} + \frac{1}{n}} = \sigma\sqrt{\frac{2}{n}}$$

Note what that gives at $n = 2$, the ablation-methodology standard: $\sigma
\sqrt{1} = \sigma$. **With two seeds per condition, the uncertainty on a
difference is exactly one single-run standard deviation.** Two seeds does not
buy you precision; it buys you the ability to *estimate* $\sigma$ and to know
that you have not bought precision.

**Worked.** You measure seed noise for your 30M-parameter configuration and find
$\sigma = 0.0040$ bpb. You then run two conditions, two seeds each: the full
corpus averages 0.7420 bpb, and the corpus with source S removed averages
0.7455 bpb. Removing S made the model worse by 0.0035 bpb, which looks like a
real contribution.

$$z = \frac{|{\Delta}|}{\text{SE}_{\text{diff}}} = \frac{0.0035}{0.0040 \times \sqrt{2/2}} = \frac{0.0035}{0.0040} = 0.88$$

Less than one standard error. **That difference is not distinguishable from
nothing.** You have not measured source S's contribution; you have measured your
own noise floor and found the contribution underneath it.

**And here is what saves the project.** Ask how many seeds it would take. To get
$z \geq 2$ you need $\Delta \geq 2\sigma\sqrt{2/n}$, which rearranges to

$$n \geq \frac{8\sigma^2}{\Delta^2} = \frac{8 \times (0.0040)^2}{(0.0035)^2} = \frac{1.28 \times 10^{-4}}{1.225 \times 10^{-5}} = 10.4 \rightarrow 11 \text{ seeds per condition}$$

Twenty-two runs to resolve one source. At 4 minutes per run on one H100, that is
88 minutes and roughly $5. The reason this book works at 30 million parameters
rather than 7 billion is contained in that sentence: at this scale the honest
answer to "your effect is under the noise floor" is *run it twenty more times*,
not *publish anyway*.

```beat
id: v3-b9
type: completion
concept: d-eval-noise
# variants: give n and delta, blank sigma; or blank the required n only.
prompt: |
  Your setup has measured seed noise $\sigma = 0.0060$ bpb on held-out papers.
  You run two conditions with $n = 2$ seeds each. The full corpus averages
  0.8100 bpb; with source T removed it averages 0.8244 bpb.

  Step 1. $\Delta = 0.8244 - 0.8100 =$ ____
  Step 2. $\text{SE}_{\text{diff}} = \sigma\sqrt{2/n} = 0.0060 \times \sqrt{2/2} =$ ____
  Step 3. $z = \Delta / \text{SE}_{\text{diff}} =$ ____
  Step 4. Real or not, at a $z \geq 2$ bar? ____
  Step 5. Now a different source U shows $\Delta = 0.0050$. Seeds needed to
          reach $z \geq 2$: $n \geq 8\sigma^2/\Delta^2 =$ ____

  Fill the blanks. Then state in one sentence why step 5's answer is good news
  rather than bad, given a run cost of 4 minutes.
answer: |
  Step 1: 0.0144 bpb
  Step 2: 0.0060
  Step 3: 2.4
  Step 4: Real - it clears the z >= 2 bar.
  Step 5: n >= 8 x (0.0060)^2 / (0.0050)^2 = 8 x 3.6e-5 / 2.5e-5 = 11.52,
          so 12 seeds per condition.

  Why that is good news: 12 seeds per condition is 24 runs, which at 4 minutes
  each is 96 minutes of one GPU. At this model scale the remedy for an
  underpowered result is to buy more seeds, and it costs about $6. The seed
  count is a budget line, not a limitation.
rubric: |
  Required, exactly: step 1 = 0.0144; step 2 = 0.0060; step 3 = 2.4; step 4 =
  real / significant; step 5 = 11.52 rounded up to 12.
  The final sentence must connect the required seed count back to run cost and
  observe that more seeds are affordable at this scale.
  All five plus the final observation = pass.
  Computing step 2 as sigma/sqrt(2) = 0.0042 = fail; that is the standard
  error of ONE condition's mean, not of the difference between two, and the
  error makes every result look more significant than it is. This is the single
  most common way an attribution claim gets overstated.
  An answer treating step 4's z = 2.4 as "proof" rather than as clearing a
  pre-set bar = pass on the arithmetic, but flag D19 for v6 and v7; a z-score
  is evidence against a null, not a proof.
```

**The ablation discipline, in four rules.** These are lifted directly from the
methodology behind the leading open web corpus, where the same problem - does
this data change help - had to be answered hundreds of times without fooling
anyone.

1. **Two models per condition minimum, different seeds, averaged.** Not for
   precision, as the arithmetic above shows, but because a single run per
   condition makes the seed noise structurally invisible and every reported
   difference unfalsifiable.
2. **Choose metrics for low seed variance and monotone improvement, before the
   run.** A metric that jumps around across seeds cannot detect anything; a
   metric that is not monotone in training progress cannot be interpreted when
   it moves. Held-out bpb has both properties. Most benchmarks have neither.
3. **Require scores above random.** A benchmark on which your model scores at
   chance is measuring nothing. Which brings us to:
4. **Below about 1 billion parameters, ignore reasoning benchmarks entirely.**
   A 4-choice multiple-choice benchmark has a chance floor of 25%. On roughly
   14,000 items, sampling alone gives a standard deviation of
   $\sqrt{0.25 \times 0.75 / 14{,}000} = 0.4$ percentage points, and seed
   variation is larger still. A 30M-parameter model that "improves" from 25.1%
   to 26.3% has done nothing. Reporting that number is not optimism, it is a
   fabricated result, and anyone who has run these evaluations will recognize it
   instantly.

For calibration on what these small models actually are: the $92 561M-parameter
model from section 1 scores 0.2219 on the DCLM CORE aggregate, where GPT-2
scores 0.2565. It is, on that measure, worse than a 2019 model. It is
nevertheless a completely adequate subject for attribution research, because
attribution asks which data moved held-out loss, and held-out loss is measurable
with precision at any scale. Do not confuse "the model is weak" with "the
measurement is weak." They are unrelated.

The domain evaluation for this project, concretely: hold out a set of papers
before training - chosen before, decontaminated ruthlessly against the training
shards - and report bits per byte on them, with two or more seeds and an error
bar, for every condition. That is the whole eval harness. It fits in an
afternoon and it is the number every claim in v4 through v8 rests on.

## At the bench: the overnight run

Here is what to build after this unit, and what it feeds.

**Build.** Take the tagged corpus from v2 - shards with a manifest mapping
(shard, offset, length) to document to source, document-masked packing so no
training sequence blends two sources. Train a 10 to 30 million parameter model on
it overnight on the Mac, using MLX or a nanoGPT-style trainer. Use WSD rather
than cosine, so the stable-phase checkpoints are branchable later. Train it
**twice, with different seeds**, changing nothing else.

**Artifact.** Two checkpoints, and a single table: held-out bits per byte on the
reserved papers, reported as a mean over the two seeds with the spread stated.
The spread is $\sigma$, your setup's seed noise, and it is the most valuable
number in this unit - every contribution estimate in v4, every Shapley value in
v5, and every influence score in v8 gets compared against it. A run that
produces one number and no $\sigma$ has produced nothing usable.

**Then budget the same thing for a node.** Same $N$, same $D$, the arithmetic
from section 3 against $3.2 \times 10^{15}$ FLOP/s and mid-2026 rental prices.
Write down the wall clock, the dollar cost, and the cost of the full 550-run
sweep at that configuration. That budget is the go/no-go for v4: if the
counterfactual ground truth costs under a few hundred dollars, the whole
attribution program is affordable and everything downstream is a matter of doing
it. If your chosen $N$ and $D$ push it past that, shrink them now, before v4
rather than during it.

**What feeds forward.** $\sigma$ becomes the significance bar in v4. The
branchable checkpoints become the cheap screening tool. The epoch count you
chose becomes, in v6, the memorization dial. And the tokenizer's bytes-per-token
figure is what makes every bpb number in this book comparable to every other.

```beat
id: v3-b10
type: self-explain
concept: d-eval-noise
prompt: |
  You are writing down, in advance, what your overnight run must record for
  the v4 counterfactual work to be possible at all. A colleague suggests
  recording the final training loss, the total wall clock, and the checkpoint.

  That list is missing the load-bearing item. State what it is, why the other
  three cannot substitute for it, and what specifically becomes impossible in
  v4 without it. Answer from the unit's text; do not reference any run you
  have or have not performed.
answer: |
  The missing item is the seed noise: held-out bits-per-byte from two or more
  runs of the identical configuration with different seeds, reported as a mean
  and a spread. The spread, sigma, is the load-bearing number.

  Why the other three cannot substitute:

  - Final training loss compares the model to its own past, on data it trained
    on, under a schedule whose shape produces a late drop regardless of data
    quality. It says nothing about how this model compares to a different
    model, which is the only comparison v4 makes.
  - Wall clock is a budget fact. It has no bearing on whether a measured
    difference is real.
  - The checkpoint is one sample from a distribution over training runs. One
    sample carries no information about the width of that distribution.

  What becomes impossible in v4: every ground-truth number in v4 is a
  DIFFERENCE - full-corpus loss minus leave-one-source-out loss. The standard
  error of a difference between two conditions with n seeds each is
  sigma x sqrt(2/n), so without sigma there is no denominator, no z, and no way
  to say whether a source's measured contribution is distinguishable from
  zero. You would produce a table of numbers with no way to tell which entries
  are signal. Since a source's contribution can swing by more than its own
  magnitude across seeds, the table would be actively misleading rather than
  merely incomplete.
rubric: |
  Must contain: (1) the missing item is seed noise / a multi-seed spread on a
  held-out metric, ideally naming bpb, (2) a reason training loss is not a
  substitute - it is same-data, self-comparison, and schedule-shaped, (3) the
  specific v4 consequence: contributions are differences, and a difference
  needs a standard error, which needs sigma.
  (1) and (3) = pass. All three = full credit.
  An answer naming "more seeds" without connecting sigma to the standard error
  of a DIFFERENCE = partial, do not pass; the learner has the ritual and not
  the reason.
  An answer proposing to substitute a larger model or longer training to
  reduce noise = fail, diagnosing D16. Noise is not a defect that goes away
  with a better run; contribution is a random variable at every scale, and the
  remedy is seeds, not size.
```

## What you can now do

Three capabilities, each of which is a precondition for the rest of this book.

**You can price any experiment before running it.** $C \approx 6ND$, divided by
sustained throughput, times the hourly rate. That line turns "should we try
this" from a judgment call into a division, and it is what makes a 550-run
counterfactual sweep an ordinary Tuesday rather than a research proposal.

**You know what the knobs do and what they do not.** Warmup exists because early
gradients are unreliable. Cosine decay costs you the ability to branch; WSD buys
it back, at the price of a different counterfactual. Muon is faster and is not
magic. And the loss curve on your screen is an optimizer health monitor, not a
quality metric - the prettiest curve in the logbook can belong to the most
broken run.

**You know when a difference is real.** Report bits per byte, never perplexity
across tokenizers. Run every condition at least twice. Measure $\sigma$ once and
carry it everywhere. Compute $z = \Delta / (\sigma\sqrt{2/n})$ before believing
anything, and when $z$ comes up short, buy more seeds rather than more
confidence.

That last one is the through-line. v4 defines contribution as a counterfactual
difference and needs $\sigma$ to say whether any given difference exists. v5
builds a fair split out of those differences and inherits their noise. v8
compares cheap attribution methods against the expensive ground truth, and the
comparison is only meaningful relative to the floor you measured here. A number
without an error bar is not a result, and in a system that moves money it is a
liability.

## Notation in this unit

<!-- canon-only -->

Reference, not reading. Return here when a symbol goes blurry; nothing below is
new.

| Symbol | Means | Typical value here |
| --- | --- | --- |
| $N$ | parameters in the model | $10^7$ to $6 \times 10^8$ |
| $D$ | training tokens (with repeats counted) | $2 \times 10^8$ to $10^{10}$ |
| $C$ | total floating-point operations for the run | $10^{16}$ to $4 \times 10^{19}$ |
| $\mathcal{L}$ | mean loss, nats per token | 1.5 to 3.5 |
| bpb | bits per byte on held-out text | 0.6 to 1.2 |
| $\eta$ | learning rate, the per-step distance a parameter moves | $10^{-4}$ to $10^{-3}$ |
| MFU | achieved FLOP/s divided by hardware peak FLOP/s | 20% to 50% |
| $K$ | number of tagged sources in the corpus | 10 to 50 |
| $n$ | seeds per experimental condition | 2 to 12 |
| $\sigma$ | standard deviation of held-out bpb across seeds | 0.002 to 0.008 |
| $\Delta$ | difference in bpb between two conditions | 0.001 to 0.05 |

The three formulas, restated with every symbol defined above:

$$C \approx 6ND \qquad
\text{bpb} = \frac{\mathcal{L}}{\ln 2 \times \text{bytes/token}} \qquad
\text{SE}_{\text{diff}} = \sigma\sqrt{\tfrac{2}{n}}$$

A conversion worth memorizing, because it appears in every budget: one H100
sustains roughly $4 \times 10^{14}$ FLOP/s, an 8xH100 node roughly
$3.2 \times 10^{15}$, and a MacBook under MLX roughly $10^{13}$. Those three
numbers plus $6ND$ price every run in this book.

One deliberate simplification, flagged so it does not read as a contradiction
later: $C \approx 6ND$ counts only parameter-proportional work, ignoring
attention's sequence-length term, and it assumes every parameter is active for
every token. The first is safe here only because the throughput figures above
are effective rates fitted against $6ND$ at comparable context lengths - the two
approximations are a matched pair and must travel together. The second fails
outright for mixture-of-experts models, where you substitute active parameters
for $N$. Take a configuration from here to a 100,000-token context or to an MoE
architecture and revisit the formula before trusting the invoice.
