---
unit: v7
title: "The notary: proving it, and selling the proof"
concepts:
  - d-canary-design
  - d-paired-tests
  - d-precommitment
  - d-audit-landscape
assumes:
  - d-mia-limits
  - d-evidence-discipline
---

The previous unit ended in a wall. Every retrospective method - loss thresholds,
membership attacks, anything that looks at a finished model and asks whether
your text is in it - needs to know how the measurement behaves when the answer
is no, and there is no way to find out short of retraining somebody else's
model without your data. The null is not hard to estimate. It is not
samplable.

This unit walks around the wall rather than through it. If you cannot sample
the null after the fact, construct it in advance: generate your evidence from a
key, publish half of what the key produced, keep the other half secret, and let
the secret half be the null. That inversion is the entire idea, and everything
downstream of it - how the published half has to be written to survive a data
pipeline, how many copies it needs, how the test is run, what gets committed
before anyone looks at a model - is a design problem with worked answers.

By the end you can do three things. Design a watermark scheme that yields a
p-value somebody can defend under cross-examination, and compute that p-value
by hand from counts. Wrap it in a pre-commitment protocol a third party can
verify was not retrofitted after the result came in. And place the resulting
product in an audit market whose shape is currently set by a European
regulation that mandates disclosure and, in the same document, disclaims
verification.

That last part is not background colour. The commercially open slot this whole
book has been walking toward is a gap a regulator wrote into its own
enforcement notice, and the first section is that sentence.


## The empty chair

Here is a document that will exist, in some form, for every general-purpose
model on the European market. It is the Article 53(1)(d) public summary of
training content, and the template is mandatory - not a suggested format, a
filled-in form with fixed fields.

```
SECTION 2: DATA SOURCES

  Size of training data              1B - 10T tokens
                                     [bucket; the template has three]

  Large public datasets used         Common Crawl (2019-2024 snapshots)
                                     Wikipedia (all languages)
                                     GitHub public repositories
                                     [any dataset above 3% of that
                                      modality's public data must be named]

  Scraped domains, top 10% by        reddit.com, github.com, medium.com,
  volume of content                  stackexchange.com, ...
                                     [1,247 entries; SMEs may list the
                                      top 5% or 1,000, whichever is smaller]

  Data obtained under licence        agreements with 24 counterparties

  Next refresh due                   2026-12-01
```

You publish a specialist journal. Twelve issues a year, four hundred
subscribers, everything on one domain that has never been in anybody's top ten
per cent of anything. You want to know whether this model was trained on your
archive.

The document above is mandatory. It is refreshed every six months. Failing to
publish it, or publishing a false one, carries a fine of up to three per cent
of global turnover or fifteen million euros, whichever is higher, enforceable
from August 2, 2026. And it cannot answer your question. Not because the
developer concealed anything - assume they filled it in honestly - but because
the form does not have a field for you.

Then the Commission said so out loud. From the explanatory notice accompanying
the template, C(2025) 8311 final, December 5, 2025, paragraph 26: the AI Office
supervises compliance "without performing a work-by-work assessment or checks
whether specific content has been used."

Read that as a product spec rather than as a disappointment. A regulator has
made disclosure mandatory, made it recurring, attached a fifteen-million-euro
fine to it, defined its format down to the field, and then stated in writing
that nobody in the enforcement chain will check whether any particular work is
in there. The chair where a verifier would sit has been drawn on the diagram
and deliberately left empty.

```beat
id: v7-b1
type: predict
concept: d-audit-landscape
prompt: |
  Before reading on, commit to an answer in writing.

  You are the journal publisher. You want a third party - a court, a
  regulator, a counterparty in a negotiation - to be able to answer "was this
  archive used to train that model" with a stated chance of being wrong.

  Name the one piece of information that would make that possible from
  outside the lab, with no access to the training data, the weights, or the
  developer's cooperation. Then state the harder half: what has to be true
  about WHEN that information came into existence.
answer: |
  The information is a matched pair: something you published, and something
  generated the same way that you deliberately did not publish. The
  unpublished twin is what makes the test work, because it tells you how the
  measurement behaves on content the model provably never saw.

  The timing requirement is the half almost nobody states, and it is fatal if
  you get it wrong. The pair has to exist, and the choice of which half to
  publish has to be fixed, BEFORE the model was trained - and, separately, the
  test you are going to run on them has to be fixed before you look at the
  model. Evidence manufactured after seeing the defendant is not evidence, and
  an opposing expert's first question is when each item in your protocol came
  into existence.
rubric: |
  Grade generously on the first half and strictly on the second.
  Pass requires: (1) some notion of a control, comparison set, or secret held
  back - any wording that involves content the model definitely did not see,
  generated the same way as content it might have; AND (2) an explicit timing
  constraint, either that the material must predate the training run or that
  the test must predate seeing the model. Both halves = pass.
  (1) alone = fail, and deliver sections 2 and 7 in full; the learner has the
  statistical idea and not the evidentiary one, which is the half that is the
  product.
  An answer proposing to measure the model's loss on the archive and compare
  it to something = fail, diagnosing D2; that is the retrospective method the
  previous unit ruled out, and the correction must be re-delivered.
  An answer that says it is impossible = fail; this unit is the construction.
check: llm
```

Three ways to try to answer the publisher's question. They are worth laying
side by side, because the difference between them is the whole unit and it is
not a difference in cleverness.

**One: measure the model.** Take a hundred articles from the archive, measure
how surprised the model is by each one, and compare against a hundred similar
articles from a journal you know was not in the training set. The archive
scores lower. Something happened.

Nothing follows. To turn "scores lower" into "was trained on," you need to know
how much lower a hundred articles score when the model has NOT seen them, and
the only honest way to learn that is to have a model that is identical except
for your data - which means retraining a frontier model, which you cannot do.
Worse, the comparison journal differs from yours in period, topic, house style,
and length, and any one of those moves the loss on its own. The published
attacks in this family were largely detecting the publication date of the test
set rather than membership; the giveaway is that classifiers which never query
the model at all beat them on properly matched benchmarks. This is the wall the
previous unit ended on, and it does not have a workaround.

**Two: make the model recite.** Prompt it with the first line of an article and
see whether it continues verbatim. When this fires it is the best evidence in
the entire field - courts have found it persuasive, it needs no statistics, and
an exhibit is a screenshot. It fires on heavily duplicated text. Your journal
has four hundred subscribers and is not quoted anywhere. It will not fire, and
its silence means nothing at all, because it is silent for content that is
definitely in the corpus.

**Three: a matched pair from nine months ago.** Last autumn you published
twenty-five articles, scattered across four issues, each describing a research
instrument that does not exist. Not gibberish - fluent, sober, entirely
plausible prose about the Kesterling aperture analyser: who built it, in what
year, at what institution, and what its characteristic operating wavelength is.
Four specific attributes each, a hundred and fifty words each, indistinguishable
in style from everything else you print.

At the same time, from the same key, you generated two hundred and fifty more.
Same generator, same style, same four-attribute structure, same invented-entity
shape. Those you never published. They sit in a file whose hash you posted
publicly in September.

Now ask the model: "At what wavelength does the Kesterling aperture analyser
operate?" Ask it for all four attributes of all twenty-five published
instruments. Then ask the same questions about the two hundred and fifty
instruments that were never published anywhere.

The second set is the point. It is the answer to "what does this measurement
look like when the model definitely did not see the content," obtained without
retraining anything, because you built the unseen content yourself and kept it
unseen on purpose.

Approach three has a false-positive rate. Approaches one and two do not - one
because its null is unsamplable, two because it produces no rate at all, only a
demonstration. That difference is the whole reason this unit exists, and the
next section is about nothing else.

## Constructing a null

Strip the vocabulary back and say what a statistical claim actually is, because
the rest of the unit is a construction and you need to know what is being
constructed.

You want to say: this model was trained on my content. You cannot verify that
directly, so you do the thing every empirical field does. You define a
measurement, you assume the boring explanation, and you ask how surprising your
measurement would be if the boring explanation were true. The boring
explanation is the **null**: the model was not trained on your content, and
whatever pattern you are looking at came from coincidence.

The number that comes out is the **p-value**: the chance of seeing a
measurement at least as extreme as yours, if the null were true. Small p means
the coincidence story is a poor fit. It does not mean the null is false, it
means the null does a bad job of accounting for what you saw.

Here is the part that matters, and it is where every failure in this literature
lives. To compute that chance, you need the **distribution of your measurement
under the null** - not a point estimate, not an intuition, the whole spread.
What values does the measurement take when the model has not seen your content?
How often does it get as high as what you observed, by luck alone? Without that
spread there is no p-value. There is a number with a decimal point in it, and
no way to convert it into a chance of being wrong.

The retrospective methods die exactly here. To learn what "the model's loss on
these hundred articles" looks like when the model was not trained on them, you
would need models trained without them, and you do not get to run that
experiment on somebody else's frontier system. The null exists; it is simply not
a thing you can draw samples from. That is the previous unit's conclusion in two
sentences, and it is not a limitation of current technique.

Now invert it.

Suppose you did not have to draw the null from the world. Suppose you
manufactured it. Take a generator that produces content of a specific kind, seed
it with a secret key, and run it a few hundred times. Every output is a draw
from the same distribution, by construction - not "similar", not "matched on
observable characteristics", identical in distribution, because one generator
made them all. Publish some of them, into your corpus, where a crawler will find
them. Keep the rest, and never publish them anywhere.

The model was trained on some of the outputs of your generator, and definitely
not on the others. Whatever you measure on the unpublished ones is a sample from
your measurement under the null - a real sample, as many draws as you care to
generate, because unpublished content is free.

That is the whole trick, and it is worth being precise about why it works when
the other approaches do not. The problem with comparing your journal against
somebody else's is that the two differ in a hundred uncontrolled ways, and any
of them could explain a gap. The generator removes every one of those
differences at a stroke. The published and unpublished items differ in exactly
one respect - whether they were published - because a single keyed process
produced both, and the split between them was decided by the key rather than by
anything about the content.

Three consequences follow immediately, and each is a design rule the rest of the
unit spends its time obeying.

**The controls have to be secret, and secret is a claim about the past.** An
item you generate today and label a control tells you nothing about a model
trained last year, because the generator's output distribution today is a thing
you chose today, possibly after seeing how the test was going. The control set
has to have existed, sealed, before the training run.

**The published items have to survive the trip.** A watermark that a quality
filter deletes, a deduplicator collapses, or a normalizer rewrites is a
watermark that never reached the model, and its absence from the model proves
nothing about whether your corpus was scraped. The next section is a catalogue
of designs that die on the way in.

**The test has to be fixed in advance.** With a key, a published set, and a
control set in hand, there are dozens of statistics you could compute and
thresholds you could apply, and if you choose after looking at the model you
will choose the one that fires. That is not a statistical problem, it is an
evidentiary one, and section 7 is about the machinery that closes it.

This is the retrospective-to-prospective pivot the volume has been building
toward, and it is worth naming in one line, because it is the sentence you say
across a table. Retrospective evidence asks a question about a training run that
already happened, and has no access to the counterfactual. Prospective evidence
arranges, before the run, for the counterfactual to exist as a file on your disk.

```beat
id: v7-b2
type: self-explain
concept: d-canary-design
prompt: |
  A colleague proposes this: rather than keeping unpublished twins, generate
  the control items now, at detection time, from the same key. The generator is
  deterministic, so the items are drawn from the same distribution either way,
  and it saves you having to store anything for a year.

  The statistics are fine. Say precisely what breaks, and what the fix costs.
answer: |
  The statistics are indeed fine, in the narrow sense that the items really are
  draws from the same distribution and the null they define is the right one.
  What breaks is that nobody has to believe you.

  Two failures, and the second is worse. First, if the controls are generated
  at detection time you can generate many batches and use whichever produces
  the largest gap; nothing in the artifact distinguishes "the controls" from
  "the best of forty control batches." Second, and fatally, generating at
  detection time means the KEY is in your hands, unconstrained, at a moment
  when you have already seen the model. A key that is chosen after you can test
  keys is a key you can search: try candidate keys until one produces a
  published/control split whose gap is large, then present that key as the one
  you have had all along.

  The fix is not to store the file. It is to commit publicly to the key before
  the training run, by publishing a timestamped hash of it. Then generation at
  detection time is fine, because the key is pinned and the derivation is
  deterministic, so anyone can regenerate both sets and check that they are the
  ones the commitment covers. The cost is that the commitment has to be
  published somewhere append-only and third-party visible, and it has to happen
  before the model exists.
rubric: |
  Must contain: (1) the objection is evidentiary, not statistical - the sampling
  is valid and the problem is that the analyst had freedom after seeing the
  model; (2) a concrete description of the freedom, either selecting among
  control batches or searching over keys; (3) the fix is a public timestamped
  commitment to the key (a hash), not necessarily storage of the items.
  (1) and (2) = pass. All three = full credit.
  An answer that says the statistics break, or that the regenerated controls
  are not really from the same distribution, = fail; that misses the point,
  which is that a valid statistic computed with post-hoc freedom is still not
  evidence. Deliver section 7 before continuing.
  An answer that proposes keeping the key secret forever = fail; the key must
  be revealed at detection time or nobody can reproduce the derivation, and the
  commitment is what makes revealing it safe.
check: llm
```

## The test, end to end

<!-- fade: v7-detection-ztest -->

Now the arithmetic, in full, on real-shaped numbers. There is no calculus in
this section and no distribution more exotic than a coin flip. The strongest
method in this literature has the simplest statistics in it, and that is not a
coincidence: methods are sound when their nulls are constructed, and constructed
nulls are simple by construction.

Before any notation, the shape of what is about to happen, because the
arithmetic is easier to follow if you already know where it is going.

You are going to ask the model a fixed set of questions about the entities you
published, and count how many it answers the way your generator wrote them.
That count on its own means nothing, because some questions get answered
correctly by luck. So you ask the same kind of questions about the entities you
never published, and count those too. That second count is the luck rate,
measured rather than assumed.

Then you need one more thing before the comparison means anything: how much the
luck count itself bounces around. Not every batch of unseen entities produces
the same number of lucky hits. If the typical bounce is small, a modest excess
is remarkable; if the bounce is large, the same excess is nothing. So the final
number is the excess expressed in units of the bounce, and everything below is
that sentence written down carefully.

Three quantities, then: the luck rate, the size of the bounce, and how many
bounces you are away from luck. Everything else in this section is bookkeeping.

**The design.** You planted 25 watermark documents. Each describes one invented
entity with four attributes - inventor, institution, year, operating wavelength.
That gives $4 \times 25 = 100$ detection questions, where each question asks the
model for one attribute of one entity and is scored right if the model returns
the value your generator assigned.

The same key produced 250 more entities that were never published. Those give
$4 \times 250 = 1{,}000$ control questions, asked and scored exactly the same
way.

**The measurement.** Ask all 1,100 questions through the plain API. No logits,
no weights, no gradients - text in, text out. Score each answer against what the
generator wrote. Suppose you get:

- Control questions: 1,000 asked, 100 answered with the generator's value.
- Published questions: 100 asked, 25 answered with the generator's value.

**Step 1: the controls give you the null rate.** Write $p_0$ for the chance that
a single question is answered with the generator's value when the model has
never seen the entity. That is not zero: your generator picks a wavelength from
a plausible range, and a model guessing plausibly sometimes lands on it. Measure
it rather than assuming it:

$$p_0 = \frac{100}{1{,}000} = 0.10$$

Ten per cent of the time, a model that has definitely never seen the entity
produces the generator's answer anyway. That is the coincidence rate, and you
did not have to guess it, argue for it, or borrow it from a paper.

**Step 2: state the null as a distribution over counts.** Write $n$ for the
number of published detection questions, here $n = 100$, and $k$ for how many of
them came back with the generator's value, here $k = 25$.

If the null is true, each of the $n$ questions is an independent trial that
succeeds with probability $p_0$, so $k$ is a sum of $n$ independent coin flips
each landing heads with probability $p_0$. Its mean is what you would expect
from $n$ flips:

$$\mu_0 = n\,p_0 = 100 \times 0.10 = 10$$

and its standard deviation - the typical distance of one draw from that mean -
is

$$\sigma_0 = \sqrt{n\,p_0(1-p_0)} = \sqrt{100 \times 0.10 \times 0.90} = \sqrt{9} = 3$$

That square root is worth thirty seconds. Each individual flip contributes
variance $p_0(1-p_0)$: it is 0 when $p_0$ is 0 or 1, since the outcome is then
certain, and largest at $p_0 = 0.5$, where the flip is least predictable.
Variances of independent things add, so $n$ flips have variance $n\,p_0(1-p_0)$,
and the standard deviation is its square root. The consequence you will use
repeatedly: noise grows like $\sqrt{n}$ while signal grows like $n$, which is
why more detection questions help, and why they help sublinearly.

**Step 3: standardize.** The statistic $z$ is how many null standard deviations
your observation sits above the null mean:

$$z = \frac{k - \mu_0}{\sigma_0} = \frac{25 - 10}{3} = \frac{15}{3} = 5.00$$

**Step 4: convert to a p-value.** For counts this size the sum of flips is close
enough to a normal distribution that the standard tail table applies. You need
one-sided values, because the alternative is directional - training can only
raise the hit rate, never lower it.

| $z$ | one-sided $p$ |
| --- | --- |
| 1.645 | 0.05 |
| 2.00 | 0.023 |
| 2.33 | 0.010 |
| 2.88 | 0.0020 |
| 3.02 | 0.00125 |
| 3.29 | 0.00050 |
| 3.48 | 0.00025 |
| 4.00 | $3.2 \times 10^{-5}$ |
| 4.50 | $3.4 \times 10^{-6}$ |
| 5.00 | $2.9 \times 10^{-7}$ |

So $z = 5.00$ gives $p = 2.9 \times 10^{-7}$: about three in ten million. If this
model had never seen your journal, a hit rate this far above the coincidence
rate would happen roughly once in three and a half million audits.

One caution about that last figure, because it is the number that goes in a
document somebody will attack. The normal curve is an approximation to a sum of
coin flips, it is excellent in the middle and optimistic in the far tail, and out
at five standard deviations it overstates the case. Summing the exact
coin-flip probabilities instead gives $1.3 \times 10^{-5}$, about one in
seventy-six thousand - still overwhelming, and forty-five times less
overwhelming than the approximation claimed. The sum is a one-line computation
with no tradeoff attached, so compute it. Use the $z$ to think with and put the
exact tail in the report.

**Step 5: decide.** Against a pre-registered threshold - and section 7 is about
why the word "pre-registered" is doing more work in that sentence than the rest
of the unit put together - $2.9 \times 10^{-7}$ clears anything reasonable. The
finding is that the model reproduces knowledge that exists nowhere except in
documents you published.

**A sign convention, so published figures do not confuse you.** The statistic
above goes UP when the model has been trained on your content, because it counts
hits. Most of the literature computes the same test on a loss or perplexity
statistic, which goes DOWN when the model has memorized something. A paper
reporting $z = -5.374$ after continued pretraining, or $z = -4.6$ after
instruction tuning, is reporting exactly the strength of evidence that $+5.37$
and $+4.6$ mean here. The sign carries the direction of the statistic, never the
strength of the result.

**And one honesty item, declared now and settled in the deeper-math track.** Step
2 treated $p_0 = 0.10$ as if it were known exactly. It is not; it was estimated
from 1,000 control questions and carries its own uncertainty. The correct
statistic compares two measured proportions and accounts for the error in both,
and on these numbers it gives $z = 4.51$ rather than $5.00$ - still
$p \approx 3 \times 10^{-6}$, still overwhelming, but not the same number.

The design lesson is worth more than the correction. Control items are free: you
generate them and never publish them, so nothing constrains how many you make.
Make the control set large - ten times the published set is a reasonable floor -
and the gap between the two statistics closes to nothing. A protocol that skimps
on controls is paying, in significance, for a file it could have made bigger for
free.

```beat
id: v7-b3
type: completion
concept: d-canary-design
# variants: blank steps 1 and 3 instead of 2, 4 and 5; or supply z and blank
# the observed count k.
prompt: |
  A different engagement. The provider planted 36 watermark entities with four
  attributes each, and generated 360 never-published control entities from the
  same key.

  Step 1. Control questions asked: $4 \times 360 = 1{,}440$; answered with the
          generator's value: 288. So $p_0 = 288 / 1{,}440 =$ ____

  Step 2. Published detection questions: $n = 4 \times 36 =$ ____

  Step 3. Null mean: $\mu_0 = n\,p_0 = 144 \times 0.20 =$ ____

  Step 4. Null standard deviation:
          $\sigma_0 = \sqrt{144 \times 0.20 \times 0.80} = \sqrt{23.04} =$ ____

  Step 5. Observed hits on the published set: $k = 48$.
          $z = (48 - 28.8) / 4.8 =$ ____

  Fill the five blanks. Then read the p-value off the table in this section and
  state, in one sentence, what the result licenses you to say.
answer: |
  Step 1: $p_0 = 0.20$
  Step 2: $n = 144$
  Step 3: $\mu_0 = 28.8$
  Step 4: $\sigma_0 = 4.8$
  Step 5: $z = 19.2 / 4.8 = 4.00$

  From the table, $z = 4.00$ gives a one-sided $p$ of $3.2 \times 10^{-5}$,
  about three in a hundred thousand.

  What it licenses: the model answers questions about the published invented
  entities at a rate that a model which had never seen them would exceed about
  three times in a hundred thousand. It licenses a statement about the
  published documents having reached the training data. It does not by itself
  license anything about which copy of them was scraped, from which
  intermediary, or when.
rubric: |
  Required, exactly: 0.20, 144, 28.8, 4.8, 4.00. All five = pass.
  Diagnose specific errors. Computing $\sigma_0$ as $\sqrt{144 \times 0.20}$
  = 5.37 means the learner dropped the $(1 - p_0)$ factor; re-deliver the
  variance paragraph, because the same error inflates every z they compute.
  Dividing by $n$ instead of $\sigma_0$ means they have standardized by the
  wrong quantity. Using $p_0 = 0.5$ or any assumed rate rather than the
  measured 0.20 means they have not internalized that the controls SUPPLY the
  null; that is the central idea of the unit and the section must be
  re-delivered.
  The one-sentence statement must not claim more than membership of the
  published documents. An answer asserting the result proves a particular
  source, licence breach, or intent = fail.
check: llm
```

## Canary design is the whole game

<!-- refutes: D18 -->

**You probably think any unusual string will do.** The intuition is honeypot
intuition, and it is good intuition in the domain it came from: plant a unique
token, watch for it to surface, and its appearance is unforgeable. Sprinkle a
random 40-character string through your articles, and if the model ever emits
it, you have your proof. The uniqueness of the string is the evidence.

**Here is the specific prediction that fails.** Under that model, the strength
of a canary rises with its improbability, so the weirder and rarer the string,
the better the trap. Run the experiment and the ordering inverts. Four designs,
all built on that intuition, all measured:

**Random high-entropy strings get deleted before training.** A modern pipeline
runs a quality classifier over every document, and a page carrying a 40-character
base64 blob in the middle of a paragraph scores like machine-generated junk,
because it is indistinguishable from machine-generated junk. If the blob does
survive filtering, near-duplicate detection is the next hazard: a canary
sentence repeated across your archive is a duplicated span, and the deduplicator
is specifically built to collapse those to one copy - destroying the repetition
count the detection depends on. The trap's most attractive property, its
conspicuousness, is exactly what the pipeline is tuned to remove.

**In-distribution strings regress to typical loss.** So make the canary ordinary
instead: a normal-looking sentence, nothing that trips a filter. Now it survives
the pipeline and fails at detection, because a sentence that looks like the rest
of the corpus produces a loss like the rest of the corpus. There is nothing to
measure. The two failure modes are opposite ends of one axis, and naive designs
sit at one end or the other.

**Homoglyph and invisible-character tricks die to normalization.** Substituting
Cyrillic characters for Latin lookalikes, or inserting zero-width joiners, is
the cleverest-feeling version and the most reliably dead. Unicode normalization
happens in the first hundred lines of every extraction pipeline. Your signal is
gone before the document is a document.

**And naive traps, done properly, were measured and did not work.** This is the
one that should end the argument, because it was not a thought experiment.
Medium-length trap sequences injected a hundred times each into a training
corpus were flat out undetectable afterwards. Pushing to long sequences with
heavy repetition got detection to an AUC of about 0.75 - which sounds like
something until you notice it means a quarter of your judgements are wrong in a
setting where you control everything.

**Here is why the wrong model is appealing.** In security, unforgeability is
the property that matters, and a high-entropy string is unforgeable. The
reasoning is sound; it is imported into a setting with an extra adversary that
security honeypots do not face. A honeypot only has to survive an attacker who
is not looking for it. A canary has to survive a data pipeline that is
aggressively, automatically, and indiscriminately deleting exactly the kind of
content that looks unusual - and it has to do so while remaining unusual enough
to be measurable afterwards.

**Here is what is actually true.** A working watermark is coherent, plausible,
and false. The design that clears both constraints at once is **fictitious
knowledge**: fluent prose stating facts about an entity that does not exist.

The published shape, with the numbers that were measured rather than argued:

- **Invented entities, four or more attributes each.** One attribute memorizes
  poorly; four or more memorize markedly better, because the attributes reinforce
  each other as a coherent bundle rather than competing as isolated oddities.
- **A hundred to two hundred tokens per document.** Long enough to be a real
  passage, short enough to slot into ordinary content.
- **Twenty-five or more distinct watermarks.** This is a statistical floor, not a
  quality one: the test counts hits across items, and a handful of items cannot
  produce a count with a usable standard deviation.
- **Under 0.1% of the training corpus is enough.** In the reported setting, 256
  injected documents sufficed. The next section is about which denominator that
  is and what you can actually control.
- **Detection by asking the model a question.** No logits, no weights, no
  gradients. A factoid question through the ordinary API, scored against what
  your generator wrote. Roughly three quarters of planted attributes came back
  correctly in the reported runs.

And the properties that make it a product rather than a paper. It survives
quality filtering, because it reads as ordinary domain prose - which it is,
apart from being about nothing. It survives deduplication, because the
twenty-five documents are twenty-five distinct passages, not one passage
repeated. It survives instruction tuning: the reported statistic moves from
about $5.37$ after continued pretraining to about $4.6$ after supervised
fine-tuning, which is a real cost and nowhere near fatal. And it is detectable
through a plain chat endpoint, which means it works against a model you have no
access to beyond a paid API key.

<!-- refutes: V7-M1 -->
That last property deserves a beat of its own, because engineers reliably assume
the opposite. **You probably think the auditor needs the weights, or at least
the logits.** Every membership-inference paper you have skimmed reports
loss-based statistics, and loss needs probabilities, and probabilities need
either open weights or a logprob endpoint that the major APIs have progressively
restricted. The failing prediction is direct: under that belief, no audit of a
closed commercial model is possible at all, and the entire market is
open-weight models. What is actually true is that a fictitious-knowledge
watermark converts the test into a question-answering task, and the statistic is
a count of correct string matches. The measurement instrument is a chat
completion. This is the single most important product property in the unit,
because the models anyone wants audited are exactly the ones that expose
nothing but text.

<!-- refutes: D15 -->
One more confusion to kill before it costs you a meeting, because the words are
the same and the mechanisms are unrelated. **You probably think output
watermarking is the same technology.** SynthID and the greenlist schemes are
called watermarking, involve models, involve hidden signals, and use a z-test on
a count statistic - genuinely the same statistical grammar as section 3.

The failing prediction: if they were the same technology, an output watermark
would carry information about training data. It carries zero bits about it. An
output watermark is inserted by the **model owner**, at inference time, by
biasing token selection, and it proves that a piece of text came out of their
model. The direction is model to output. A data watermark is inserted by the
**data owner**, before scraping, into content, and it proves that content went
into a model. The direction is data to model. They share a hypothesis test and
nothing else.

The one real connection is worth knowing because it is a business, not a
correction: output-watermarked text is radioactive. When someone scrapes
watermarked synthetic text into their training corpus, the signal persists in
the downstream model. That gives a synthetic-data producer accidental
provenance over their own outputs - a different product, with a different buyer,
built on the same test.

```beat
id: v7-b4
type: predict
concept: d-canary-design
prompt: |
  Four proposals for protecting the same specialist journal. Before reading
  the next section, rank them by how much evidence each would produce if the
  journal were scraped, and for each of the three losers name the specific
  stage of a data pipeline or the specific statistical property that kills it.

  (a) Append a unique 48-character random identifier as a footer to every
      article, the same one throughout the archive.
  (b) Replace the Latin letter "a" with the Cyrillic "a" in the journal's
      name everywhere it appears.
  (c) Insert one distinctive but entirely ordinary sentence - "The findings
      were replicated the following spring." - into forty articles.
  (d) Publish twenty-five articles about instruments that do not exist, each
      with four specific attributes, in the journal's normal house style.
answer: |
  (d) is the only one that produces evidence. The ranking of the others is
  close to meaningless because all three produce none, but the failure modes
  differ and the differences are the lesson.

  (a) dies twice. First at quality filtering: a high-entropy blob appended to
  running prose is what a quality classifier is built to score as junk.
  Second, if it survives, at deduplication: the same footer on every article
  is a duplicated span across the archive, and near-duplicate detection
  collapses exactly that, destroying the repetition count the detection needs.
  Its conspicuousness is the reason it is deleted.

  (b) dies at Unicode normalization, which runs in the first stage of
  extraction, before the document is even a document. This is the design that
  feels cleverest and has the shortest life.

  (c) survives the pipeline and fails at detection. A sentence built out of
  ordinary language produces an ordinary loss; there is nothing measurable to
  measure. Measured versions of this - medium-length traps repeated a hundred
  times - were undetectable outright.

  (d) clears both constraints at once. It reads as ordinary domain prose, so
  filters keep it and normalizers leave it alone; the twenty-five documents
  are distinct passages, so the deduplicator has nothing to collapse; and the
  content is false, so a model that answers questions about it correctly can
  only have read it. Detection is a factoid question through the plain API.
rubric: |
  Pass requires (d) ranked first AND at least two of the three losers killed
  by the correct mechanism: (a) quality filter or dedup, (b) Unicode
  normalization, (c) no measurable signal / regression to typical loss.
  Ranking (a) first = fail, diagnosing D18 directly; the failure catalogue
  must be delivered in full rather than summarized.
  An answer that gets the ranking right by reasoning "(d) is the one the unit
  is about" with no mechanism = fail; the mechanisms are the transferable
  content and the ranking is not.
  An answer that kills (d) on the grounds that publishing false content is
  unacceptable = do not fail, and flag: that is a real and separate objection,
  answered in the honest-position section, and a learner raising it early is
  reading correctly.
check: llm
```

## How many watermarks, and how often

<!-- fade: v7-watermark-density -->

Design settled, the next question is quantity, and it has a law behind it that
also tells you the thing you least want to hear about being small.

The law: whether a model memorizes a passage is governed by that passage's
frequency **relative to the size of the corpus**, not by its absolute number of
occurrences. Twenty copies in a hundred-million-token corpus and twenty copies
in a five-hundred-billion-token corpus are not the same intervention, and the
second one is nearly nothing. Stated as an operating rule: when the training
corpus grows fivefold, a trap needs roughly fivefold duplication to retain the
same detectability.

That law hands you two constraints that pull in opposite directions, and the
planning arithmetic is finding a point that satisfies both.

**Constraint A, the detector's floor.** The count statistic needs at least 25
distinct watermarks, and each watermark needs enough effective occurrences in
the training corpus to be learned at all - call it 90 as a working figure, in
line with the point where hash-style canaries became robustly detectable in the
published large-model tests.

**Constraint B, the density you can actually control.** You do not control the
trainer's corpus size, so you cannot directly set frequency-relative-to-corpus.
What you control is density within **your own** corpus, and the operating target
is at least 0.1% of it. The reasoning is that your corpus is ingested as a unit,
so its internal density is what carries through.

Do not conflate the two denominators. The encouraging published figure - 256
documents, under 0.1% - is density relative to the **training** corpus in a
controlled continued-pretraining experiment. The 0.1% you can set is relative to
**your** corpus. They are different fractions with different denominators, and
only the second is a knob you own.

**Worked example.** A publisher with a 400-million-token archive. Watermark
documents run 150 tokens each. Twenty-five distinct watermarks.

Step 1. Watermark tokens needed for the density floor:

$$0.001 \times 400{,}000{,}000 = 400{,}000 \text{ tokens}$$

Step 2. Document instances that buys, at 150 tokens each:

$$\frac{400{,}000}{150} = 2{,}666.7 \rightarrow 2{,}667 \text{ instances}$$

Step 3. Occurrences per distinct watermark, spread over 25 of them:

$$\frac{2{,}667}{25} = 106.7 \rightarrow 107 \text{ occurrences each}$$

Step 4. Check against the detector's floor: 107 exceeds 90. Both constraints are
satisfied, at a cost of one tenth of one per cent of the archive being about
instruments that do not exist.

Now run the same arithmetic for a small publisher, because this is where the
design stops being comfortable. A 40-million-token archive, same 150-token
documents, same 25 watermarks. The density floor gives $0.001 \times
40{,}000{,}000 = 40{,}000$ tokens, which is $40{,}000 / 150 = 267$ instances,
which is $267 / 25 = 10.7$ occurrences per watermark. Against a floor of 90.
Not close.

To reach 90 occurrences each, the small publisher needs $25 \times 90 = 2{,}250$
instances at 150 tokens, which is 337,500 tokens, which is 0.84% of the archive.
Eight times the density, and there is no arithmetic that makes it cheaper. The
options are: accept 0.84% of your archive being fictitious, cut the number of
distinct watermarks below 25 and lose the statistic, or accept detection that
will not fire. That is a real result and it should go in the honest column of any
pitch: **watermark strength scales with the size of the corpus you control**, so
the parties with the least leverage in the licensing market also have the
weakest evidence available to them, and pooling across small publishers under
one key is the only structural answer.

Finally, apply the scaling law forward. Your plan produces 107 occurrences
against the trainer's corpus as it is today. If the next model generation trains
on five times the tokens, the same 107 occurrences deliver roughly a fifth of the
frequency and the design degrades. Holding detectability fixed means multiplying
occurrences by the corpus growth factor - $107 \times 5 = 535$ occurrences,
which at 25 watermarks and 150 tokens is 2,006,250 tokens, or 0.5% of the
archive. Watermarking is not a thing you do once. It is a recurring obligation
whose cost rises with somebody else's compute budget, and that recurrence is
either the ugliest fact about the product or the reason it has a subscription
line, depending on which slide you are on.

```beat
id: v7-b5
type: completion
concept: d-canary-design
# variants: blank the corpus size given a target occurrence count; or supply
# the trainer's corpus growth factor and blank the revised density.
prompt: |
  A trade publisher holds a 120-million-token archive. Watermark documents run
  120 tokens each. The plan calls for 30 distinct watermarks.

  Step 1. Tokens at the 0.1% density floor:
          $0.001 \times 120{,}000{,}000 =$ ____ tokens

  Step 2. Document instances at 120 tokens each:
          $120{,}000 / 120 =$ ____ instances

  Step 3. Occurrences per distinct watermark:
          $1{,}000 / 30 =$ ____ (round down)

  Step 4. Against a detector floor of 90 occurrences, this design ____
          (passes / fails).

  Step 5. Instances needed to reach 90 occurrences on all 30 watermarks:
          $30 \times 90 =$ ____ instances

  Fill the five blanks. Then state the density that step 5 implies, as a
  percentage of the archive, and say in one sentence what the publisher's
  three options are.
answer: |
  Step 1: 120,000 tokens
  Step 2: 1,000 instances
  Step 3: 33 occurrences each
  Step 4: fails
  Step 5: 2,700 instances

  Density implied by step 5: $2{,}700 \times 120 = 324{,}000$ tokens, and
  $324{,}000 / 120{,}000{,}000 = 0.27\%$ of the archive - between two and
  three times the 0.1% floor.

  The three options: accept 0.27% of the archive being documents about
  instruments that do not exist; cut the number of distinct watermarks, which
  buys occurrences at the cost of the count statistic and is capped by the
  floor of 25; or pool with other publishers under one shared key so that the
  effective corpus behind the watermark is larger than any one archive.
rubric: |
  Required, exactly: 120,000; 1,000; 33; fails; 2,700. And the implied density
  must be 0.27% (accept 0.27 to 0.28).
  All five blanks plus the density = pass.
  Answering step 3 as 33.3 and then calling the design a pass = fail; the
  comparison against 90 is the point of the computation.
  Computing step 5 as $30 \times 90 \times 120$ and reporting 324,000 as the
  instance count confuses tokens with instances; partial credit, re-deliver the
  worked example's step 2.
  The three options do not all have to appear, but an answer that offers only
  "publish more watermark documents" without noticing that this IS raising the
  density = fail; the tension between the two constraints is the content.
check: llm
```

## Your own private control

The scheme so far spends its evidence in one currency: a hit rate on questions,
compared against a hit rate measured on unpublished twins. There is a second
design that spends a different currency and buys considerably more detection
power for the same amount of published content, and it is worth working through
because the reason it wins generalizes far past watermarking.

**The design.** Take one document you are going to publish. Under a secret key,
generate several rephrasings of it - same facts, same length, same register,
different sentences. Publish exactly one of them. Keep the others, unpublished,
as controls.

The change from the previous scheme is that the control is now attached to the
individual document rather than to the collection. Every published document
carries its own private siblings, matched to it on topic, length, style, and
subject matter, because they are rewrites of it. Reported systems in this family
detect content appearing **once** in a corpus, at under 0.001% of training
tokens.

**Why it wins, re-derived from scratch.** Start with the measurement. For a
document and a model, compute the model's average surprise per token: at each
position the model assigns some probability to the token that actually occurred,
take the negative natural logarithm of that probability, and average over the
document. Lower means the model finds the document more predictable. Training on
a document lowers it.

Now the two ways to use that measurement, on the same budget of 100 published
documents.

**Unpaired.** Compare the mean surprise over your 100 published documents
against the mean over a reference corpus you believe was not trained on. Say
training lowers surprise by 0.10 nats per token - that is the effect you are
trying to detect. The obstacle is that surprise varies enormously from document
to document for reasons that have nothing to do with training: a technical
methods section and a book review differ by far more than 0.10. Suppose that
document-to-document standard deviation is 0.60.

The precision of a mean over $n$ documents is the standard error, the
document-to-document standard deviation divided by $\sqrt{n}$:

$$\text{SE}_{\text{unpaired}} = \frac{0.60}{\sqrt{100}} = \frac{0.60}{10} = 0.060$$

$$z_{\text{unpaired}} = \frac{0.10}{0.060} = 1.67$$

A $z$ of 1.67 is a one-sided $p$ of about 0.047. Technically significant, and
useless: it is one lucky sample away from nothing, and section 7 will show that
a single multiple-testing correction erases it entirely.

**Paired.** Same 100 documents, same 0.10 effect. But now, for each document,
compute the difference between the published version's surprise and the average
surprise of its own unpublished siblings. Every source of document-to-document
variation - topic, register, length, subject difficulty - is present in both
terms of the difference and cancels. What survives is the variation between
rephrasings of the same content, which is small. Suppose that standard deviation
is 0.12.

$$\text{SE}_{\text{paired}} = \frac{0.12}{\sqrt{100}} = 0.012$$

$$z_{\text{paired}} = \frac{0.10}{0.012} = 8.33$$

Same documents, same effect, $z$ from 1.67 to 8.33. The effect did not grow.
The denominator shrank, by exactly the ratio of the two standard deviations,
$0.60 / 0.12 = 5$, and $z$ rose by exactly that factor.

**Price the alternative, because that is what makes the argument land in a
budget meeting.** Suppose you refuse to pair and try to reach paired precision by
publishing more documents instead. You need $0.60/\sqrt{n} = 0.012$, so
$\sqrt{n} = 50$ and $n = 2{,}500$. Twenty-five times the published content -
$5^2$, since precision improves with $\sqrt{n}$ while pairing improves it
linearly - to reach the same evidence. Pairing is not a refinement. It is a
twenty-five-fold budget difference on these numbers, and the multiplier is the
square of however much nuisance variance the pairing removes.

**What the private siblings are, stated exactly.** They are the null
distribution, per document. Under the null, the model has seen none of the
variants, so which sibling happens to score lowest is a coin flip among them,
and the expected difference between the published one and the average of the
others is zero. Under the alternative, the published one has been trained on and
the others have not. The controls are not a baseline, a sanity check, or a
comparison group in the loose sense. They are literally the distribution the
p-value is computed against, which is why the phrase "the private versions are
the null" is a description and not a slogan.

<!-- refutes: V7-M2 -->
A factor of five in $z$ is enough of a win to invite the wrong generalization,
so kill it here. **You probably think a more powerful detection method makes a
better audit.** The literature is organized around detection performance -
higher AUC, better true-positive rate at low false-positive rate, lower
detection floor - and it is natural to read that leaderboard as a ranking of
audit quality.

The failing prediction: under that belief, the strongest membership-inference
attack in the literature would make a better audit than a weaker one, and both
would beat a test with no attack in them at all. Neither holds. The strongest
retrospective attacks produce no defensible false-positive rate whatsoever,
because their null cannot be sampled, so their power is power to distinguish
nothing from nothing at a stated confidence of unknown. Meanwhile the paired
scheme above wins by removing nuisance variance from a null it already had,
which is a completely different kind of improvement.

What is actually true: an audit has two properties, and they are not on the same
axis. **Validity** is whether the stated false-positive rate is the real one,
and it comes entirely from the null and the pre-commitment. **Power** is whether
a real effect gets found, and it comes from design and sample size. Power
without validity is a number with no interpretation; validity without power is
an honest test that misses things, which is a business problem and not an
evidentiary one. Improve power only along the axis that leaves validity intact -
better pairing, more items, more occurrences, larger control sets - and treat
any technique that buys detection strength by weakening the null as a
downgrade, however good its reported curve.

```beat
id: v7-b6
type: compute
concept: d-paired-tests
prompt: |
  A paired design over 64 published documents. Training lowers per-token
  surprise by 0.08 nats. The document-to-document standard deviation is 0.80;
  the standard deviation of the within-document paired differences is 0.16.

  Compute the paired $z$ - that is, $0.08$ divided by $0.16/\sqrt{64}$.

  Give one number to two decimal places.
answer: 4.00
check: numeric(0.01)
```

## The pre-commitment ledger

<!-- refutes: D19 -->
<!-- fade: v7-bonferroni-threshold -->

Everything so far is a statistic. This section is the product, and the gap
between the two is the whole business.

**You probably think a small p-value is the deliverable.** You ran a valid test,
constructed a genuine null, computed a number, and it came out at $3 \times
10^{-7}$. That is proof, and the report writes itself.

**Here is the specific prediction that fails.** Under that belief, the strength
of the finding is a property of the arithmetic, so an opposing expert would have
to attack the arithmetic. They will not touch it. They will attack the
procedure, and here are the three attacks, with numbers.

**Attack one: you ran this many times.** You audit forty client corpora against
each model release. Set your threshold at the conventional $\alpha = 0.05$ per
test. If not one of the forty clients' data was used, how many of them do you
tell that it was?

$$40 \times 0.05 = 2$$

Two false accusations per release, by construction, from a procedure everyone
would call standard. The correction is Bonferroni: to hold the chance of **any**
false positive across the family at 0.05, divide the threshold by the number of
tests. Write $m$ for the family size:

$$\alpha' = \frac{\alpha}{m} = \frac{0.05}{40} = 0.00125$$

From the table in section 3, $p = 0.00125$ corresponds to $z = 3.02$. So the
threshold that survives cross-examination is $z \geq 3.02$, not $z \geq 1.645$.

Now check your design against it. The section-3 result was $z = 5.00$,
$p = 2.9 \times 10^{-7}$, which is smaller than 0.00125 by more than three
orders of magnitude. The correction costs nothing.

That is the thing to notice, and it is the strongest argument for spending money
on watermark design rather than on statistical sophistication. A well-designed
prospective test has so much headroom that multiple-testing correction is free.
A test that needed $\alpha = 0.05$ to fire - like the unpaired design in the
previous section at $z = 1.67$ - is annihilated by the same correction: $40
\times 0.047 = 1.9$, which is not a p-value at all, it is a statement that you
expect about two such results by chance. A result that survives correction was
evidence; a result that does not was never evidence, and the correction only
made that visible.

**Attack two: your tests are not independent.** Bonferroni assumes forty
separate coin flips. Your forty clients syndicate to each other, quote each
other, and appear in each other's archives; and all forty tests are run against
one model, so they share every source of randomness that model carries - one
training seed, one data order, one checkpoint. Dependence does not have a clean
correction, and pretending it does is worse than naming it. What you can do:
report the family size and the correction you applied, report the known
dependence structure explicitly (which clients share content, measured by the
same near-duplicate clustering the corpus pipeline already runs), and note that
Bonferroni is conservative under positive dependence, so the direction of the
error is known even when its size is not. A report that says "these forty tests
are not independent, here is the overlap matrix, and here is why the correction
still bounds us" is a report that survives the question. A report that is silent
on it does not.

**Attack three: you chose the threshold after seeing the data.** This is the one
that ends engagements, and no amount of statistics answers it, because it is not
a statistical objection. If the analyst had the freedom to choose the statistic,
the threshold, the questions, the family size, or which items counted as
controls after seeing the model, then the p-value describes a procedure nobody
ran. The stated false-positive rate is a property of a protocol executed
blind. Execute it with hindsight and the number is decoration.

**Here is why the wrong model is appealing.** The p-value is the only number in
the report, it is small, and smallness is what everybody was trained to look
for. And in most of empirical practice the procedural discipline really is
handled elsewhere - by a journal, a review board, a registry - so the analyst
genuinely does only have to produce the number.

**Here is what is actually true.** Evidence is a valid null, plus a
multiple-testing correction, plus honesty about dependence, plus a test,
threshold, and key committed before anyone looked at a model. Three of those four
are procedural. The statistical content of this entire product is one z-test that
fits on a napkin, and the reason it is a business is that executing it under
commitment is a thing somebody has to be trusted to have done.

**The ledger, as an ordered protocol.** Six steps. The order is the product, and
each step's position is load-bearing.

1. **Register the corpus.** Record what the collection is, its extent, and a
   content fingerprint for it - the same near-duplicate fingerprint the corpus
   pipeline uses, which is what lets you later answer "was this the copy that was
   scraped, or a syndicated one." This step also binds the corpus to a legal
   entity, which matters when the split you are protecting has to survive
   somebody registering the same catalogue twice.

2. **Generate, from a single key: the watermarks that will be published, the
   controls that will not, and the split between them.** All three derive
   deterministically from the key. Nobody, including you, gets to choose after
   the fact which items were controls.

3. **Publish a timestamped commitment to the key and the protocol.** Not the key
   - a hash of the key together with the protocol document, posted somewhere
   append-only and third-party visible, with a trusted timestamp. The protocol
   document contains the test statistic, the threshold, the family size and
   correction, the exact detection prompts, the decoding settings, and the
   stopping rule. After this moment your degrees of freedom are gone, which is
   the point.

4. **Publish the watermarked content, and wait.** This is where the calendar
   becomes the constraint, and it is the honest limitation the next section
   leads with.

5. **Run the pre-registered test.** Exactly as committed, against a named model
   at a named version on a named date. Do not adjust anything. If the prompts
   were badly chosen you have learned something about your protocol, and the
   answer is a new commitment, not a repaired old one.

6. **Emit a signed report a third party can re-run.** It contains the commitment
   hash and its timestamp, the revealed key, the derivation recipe by which the
   key produces both sets, the pre-registered protocol as committed, the model
   identity and query date, the raw counts, and the computed statistic, threshold
   and decision. A verifier regenerates both sets from the key, checks the hash
   against the timestamped commitment, re-runs the prompts, and recomputes the
   arithmetic. Every step is independently checkable, and none of them requires
   trusting you.

**Why the ledger and not the machine learning is the moat.** The z-test is
public. The watermark design is a published paper. Any competent engineer
reproduces the technical layer in a fortnight, and if you are selling that, you
are selling a commodity with a price that goes to zero.

What does not reproduce in a fortnight is a public, append-only, timestamped
record stretching back years, maintained by a party with no stake in any
particular outcome, containing commitments made before the models in question
existed. That artifact accumulates only by having started early and never having
been caught retrofitting. It is a notary business with a statistics engine
attached, and the statistics engine is the part you would open-source.

```beat
id: v7-b7
type: completion
concept: d-precommitment
# variants: blank the family size given the corrected threshold; or supply a
# z and ask whether it survives.
prompt: |
  An audit service runs one pre-registered detection per client collection
  against each model release. This quarter it covers 25 collections, and holds
  the chance of ANY false positive across the quarter at 0.05.

  Step 1. Family size: $m =$ ____

  Step 2. Corrected per-test threshold: $\alpha' = 0.05 / 25 =$ ____

  Step 3. From the table in section 3, the $z$ corresponding to a one-sided
          $p$ of 0.0020 is approximately ____

  Step 4. One collection returns $z = 2.40$, which is a one-sided $p$ of about
          0.0082. Against the corrected threshold this result ____
          (fires / does not fire).

  Step 5. Its Bonferroni-adjusted p-value is $25 \times 0.0082 =$ ____

  Fill the five blanks. Then state, in one sentence, what you report to that
  client - and what you must not say.
answer: |
  Step 1: $m = 25$
  Step 2: $\alpha' = 0.0020$
  Step 3: $z \approx 2.88$
  Step 4: does not fire (2.40 is below 2.88)
  Step 5: $0.205$

  What you report: the pre-registered test did not reach its committed
  threshold, the observed statistic was $z = 2.40$ against a required 2.88, and
  the family-adjusted p-value is 0.21 - a result you would expect to see about
  once in five quarters by chance alone across a family this size.

  What you must not say: that the result is "suggestive", "trending", or
  "significant before correction". The threshold was committed before the model
  was seen precisely so that this sentence cannot be written, and writing it
  destroys the value of every other report the ledger has ever issued. You also
  must not go back and enlarge the watermark set for this client and re-run
  against the same model; that is a new commitment against a new model release,
  or it is nothing.
rubric: |
  Required, exactly: 25; 0.0020 (accept 0.002); 2.88 (accept 2.85 to 2.90);
  does not fire; 0.205 (accept 0.20 to 0.21).
  All five = pass.
  Answering step 4 as "fires" = fail; the learner is comparing against 0.05
  and has not applied the correction they just computed.
  The second half must contain a refusal to report the uncorrected result as
  meaningful. An answer that offers to report it as "suggestive evidence"
  = fail, diagnosing D19, and section 7 must be re-delivered; that phrase is
  the exact failure the pre-commitment exists to prevent.
  An answer proposing to re-run against the same model with a larger watermark
  set = fail; that is optional stopping and it invalidates the family size.
check: llm
```

```beat
id: v7-b8
type: self-explain
concept: d-precommitment
prompt: |
  List everything that must be fixed and committed BEFORE you look at the
  model, and for each item state the specific manipulation that becomes
  available if it is fixed afterwards instead.

  Then name one thing that may legitimately be decided after the result is in.
answer: |
  Six items, each with the freedom it closes.

  1. The key. Fixed afterwards, you can search key space: try candidate keys
     until one produces a published/control split whose gap is large, then
     present that key as the one you always had.

  2. Which generated items were published and which were controls. Fixed
     afterwards, you choose the split that maximizes the difference - which is
     choosing your own null, the exact failure the construction exists to
     prevent.

  3. The test statistic and its exact computation. Fixed afterwards, you
     compute several and report the one that fires. Hit counts, mean surprise,
     paired differences and attribute recall are all defensible statistics, and
     that is precisely the problem.

  4. The threshold, the family size, and the correction. Fixed afterwards,
     alpha is not a false-positive rate at all, because a threshold chosen to
     sit just below the observed value has a false-positive rate of one.

  5. The detection prompts and decoding settings. Fixed afterwards, you
     prompt-engineer until the model produces the planted attribute, which
     converts a memorization test into a test of your own persistence.

  6. The stopping rule - which models, which versions, how many, over what
     window. Fixed afterwards, you keep testing releases until one fires and
     report that one. This is a multiple-testing violation wearing a different
     coat, and it is invisible in the final report unless the rule was
     committed.

  What may legitimately be decided afterwards: everything that does not touch
  the test. How to present the report, whether to gather corroborating evidence
  such as an extraction attempt, whether to open a negotiation, whether to
  litigate, and whether to commission a fresh protocol against a future
  release. The rule is that the test is frozen and the response to it is not.
rubric: |
  Must contain at least four of the six items WITH the corresponding
  manipulation - the item alone is not credit, since the transferable content
  is the freedom each one closes. Items 1, 2 and 4 are the load-bearing ones;
  missing all three = fail regardless of how many others appear.
  Must also contain a legitimate after-the-fact decision that does not touch
  the test.
  An answer that includes "which model to test" in the after-the-fact list =
  fail; that is the stopping rule, and choosing the model after seeing results
  is the most common real-world version of this failure.
  An answer that says everything must be fixed including the decision to
  litigate = pass but flag: over-restriction here suggests the learner has
  memorized the rule without the reason, which is that freedom matters only
  where it can move the statistic.
check: llm
```

## What a court will actually credit

Lead with the limitation, because it is permanent and because the person across
the table will find it in the first five minutes if you do not.

**A watermark protects only content published after the watermark was embedded.**
Every model trained before your first commitment is outside the scheme,
permanently, and no amount of later cleverness reaches back. The models people
most want to sue were trained on data collected years ago. That gap does not
narrow with better technique; it narrows only with time, one publication cycle
at a time.

That single fact determines the shape of the product line. The prospective
notary is the layer that produces proof, and it can only ever speak about the
future. Retrospective screening - dataset inference over collections, and the
newer trick of using structurally random strings that occur naturally in text,
such as hashes and shortened URLs, as an unlimited supply of same-distribution
null items - exists because of the gap, and it is sold as triage rather than
proof. It generates leads. It tells a rightsholder where to look and whom to
approach. It does not go in an exhibit.

Now the landscape you are entering, as of mid-2026.

**What courts have actually credited, best first.** Documentary evidence of
acquisition, obtained in discovery: records showing a lab downloaded a pirated
library. That is what drove the largest settlement in the field, and it is
evidence about a transaction, not about a model. Second, demonstrated verbatim
extraction: prompt the model, get the work back, put the two side by side. That
has carried a decided case on direct output comparison. Third, everything else.

**Statistical membership inference has no track record whatsoever.** Not a
record of failure - no record. It has not been the basis of a finding in a
decided case, and given that the methods themselves fail on properly matched
benchmarks, this is the correct outcome rather than judicial conservatism.

**A keyed watermark p-value is untested in court and is the only statistical
evidence in this field shaped like something that could survive.** The relevant
standard for admitting expert testimony asks about a known or potential error
rate, testability, and standards controlling the technique's operation. A keyed
watermark has a genuine false-positive rate, derived from a null that a
court-appointed expert can reconstruct from the key and check. The protocol is
executable by an opposing expert who need not trust the party that ran it. That
is unusually good shape for a novel technique, and it is still untested - the
honest sentence is "designed to meet the standard", never "meets the standard."

There is also a doctrinal current worth tracking, because it points at
extraction rather than at statistics: the developing academic argument is that
courts will find a model contains a copy of a work only where it is
straightforward to extract that work from outputs. If that becomes the line, the
extraction exhibit is the legally decisive artifact and the watermark is the
thing that establishes the antecedent question of whether the data was there at
all.

<!-- refutes: D13 -->
One correction to carry into any pitch, because getting it wrong in a room with
a litigator is fatal. **You probably think litigation will force labs into
ongoing royalty payments**, and that a nine- or ten-figure settlement is the dam
breaking. The failing prediction is in the settlement's own structure: it priced
the **acquisition** of pirated copies, roughly a few thousand dollars per work,
once - after the same court held that training on lawfully acquired copies was
fair use. Three jurisdictions have converged on that line. Under it, the rational
lab response is to buy one clean copy, not to enter a royalty relationship. The
theories that could compel ongoing payment - output substitution, market harm -
are unresolved, which is exactly why the space reprices when they resolve, and
exactly why building for verification beats betting on royalties: verification is
needed under every outcome, and royalties under one. This is the single most
common way a founder in this space loses a room, and the person across the table
will know the holding better than the headline.

<!-- refutes: D11 -->
And one that determines what your report can claim. **Blocking crawlers does not
keep content out of training sets**, because content syndicates: wire copies,
aggregators, quote-heavy coverage, scraped mirrors, a hundred copies that never
touch your origin. A watermark rides along with every one of those copies, which
is a genuine advantage of the design - the evidence travels with the content
rather than with the request log. But it also bounds the claim. A fired
watermark establishes that your text reached the training corpus. It does not
establish which copy, from which intermediary, under whose terms. Reports must
say so, and the corpus fingerprint from step 1 of the ledger is what lets you say
anything more specific.

## The audit market, and its graveyard

The regulation from section 1 is a product specification if you read it as an
engineer rather than as a lawyer. Here is the whole mechanism, and then what
falls out of it.

**The template.** Three sections: general information, data sources, data
processing. Training data size reported in three broad buckets - under 1B
tokens, 1B to 10T, over 10T. Any public dataset above 3% of that modality's
public data must be named. Scraped domains disclosed as the top 10% by volume,
with small and medium enterprises permitted the top 5% or 1,000 domains,
whichever is smaller. Retrieval-augmented sources excluded unless the model
learns from them. Open-source models are not exempt from the summary.

**The calendar.** Applies from August 2, 2025; models already on the market have
until August 2, 2027; enforcement begins August 2, 2026. Fines up to 3% of global
turnover or fifteen million euros. Summaries refresh every six months.

**The gap, stated by the regulator.** Paragraph 16 provides a voluntary
mechanism by which a developer may respond to rightsholder queries on request.
Paragraph 26 confirms no work-by-work assessment. So: mandatory, coarse,
self-reported, recurring, unverified, with a query channel nobody is obliged to
use.

Four products fall directly out of that shape, and they are ordered here by how
hard they are to sell.

**Exclusion certification is the friendly one, and it should be first.** A
developer wants to demonstrate they did **not** train on something - a
competitor's catalogue, a plaintiff's works, a restricted dataset they promised
to exclude. The buyer is the developer, they want the certificate, they
cooperate, they hand you access, and they pay again in six months when the
summary refreshes. The technique exists: rank-correlation methods over token
log-probabilities between models can certify non-membership in a grey-box
setting. Compare that sales motion with the adversarial one, where you tell a
lab something they do not want to hear and they have no obligation to answer the
phone.

**Disclosure conformance checking is the second.** Does this summary meet the
template - correct buckets, correct domain coverage, correctly named large
datasets, refreshed on time? It is a compliance product that exists only because
the Commission wrote a form with a fine attached, and its renewal cadence is
printed in the regulation.

**Procurement provenance is the third.** Labs spend heavily on data acquisition
and need provenance-verified sourcing and rights clearance; the empirical
baseline is grim, with the large public dataset audits finding license
information omitted for the large majority of datasets and a majority of
widely-used training data carrying non-commercial restrictions. This is buy-side
work and the money is already flowing there.

**The rightsholder-side notary is the fourth and the hardest, and it is the one
this unit builds.** The buyer wants proof, the counterparty is adverse, and the
regulation gives the counterparty no obligation to cooperate. It is the hardest
sale, and it is also the one with a defensible moat, because the ledger takes
years to accumulate and cannot be bought.

<!-- refutes: D14 -->
Now the graveyard, because these will come up in every technical conversation
you have and one of them will come up from an investor.

**You probably think cryptography can prove what a model was trained on.**
Zero-knowledge proofs verify computation. Training is computation. The
conclusion writes itself, it is elegant, and it has the additional property of
being the answer people want to be true.

**Here is the prediction that fails, with arithmetic.** The best published
throughput for zero-knowledge proof-of-training is about 0.4 GFLOP/s of proved
computation - a 10-million-parameter vision network at roughly fifteen minutes
of prover time per training iteration.

Price a 1-billion-parameter model against that. The floating-point cost of a
training run is well approximated by six times the parameter count times the
token count, because each parameter participates in roughly two operations in
the forward pass and four more in the backward pass. Write $N$ for parameters and
$D$ for training tokens. At a compute-optimal ratio of about 20 tokens per
parameter, $N = 10^9$ gives $D = 2 \times 10^{10}$:

$$C = 6ND = 6 \times 10^{9} \times 2 \times 10^{10} = 1.2 \times 10^{20} \text{ FLOP}$$

At $4 \times 10^{8}$ proved FLOP per second:

$$\frac{1.2 \times 10^{20}}{4 \times 10^{8}} = 3 \times 10^{11} \text{ seconds} \approx 9{,}500 \text{ years}$$

For a 1B model. Frontier models are three orders of magnitude larger. This is not
a gap that engineering closes; it is four-plus orders of magnitude on a curve
that has not been moving four orders per decade.

**Proof-of-Learning, the cheap alternative, is broken.** It asks a trainer to
publish a chain of intermediate checkpoints as proof of having done the work. It
has been spoofed - forged chains produced at a fraction of honest training cost -
in more than one published attack.

<!-- refutes: V7-M3 -->
**Trusted-execution attestation is real, and it proves a narrower thing than
people hear.** Confidential computing on current datacenter GPUs runs at
roughly compute parity, with the bottleneck in encrypted host-to-device
transfer, and federated variants report under 12% overhead. The attestation
proves that a specific binary, whose hash you can check, consumed a specific
dataset, whose hash you can check, and produced specific weights. Read the
sentence again for what is absent: it says nothing about what was **in** the
dataset. The hash commits to the bytes; it does not certify that those bytes are
free of your catalogue, and a trainer who never enrolls is untouched by the
entire mechanism.

So attestation is a licensing-integrity feature for a counterparty who has
already agreed to be checked - genuinely useful, worth building for a
cooperative deal, and structurally incapable of being an enforcement tool.
Notice that this places it beside exclusion certification rather than beside the
notary: both sell to willing developers.

**Here is what is actually true.** No cryptographic mechanism lets a third party
detect unlicensed data in weights. Adversarial verification comes from statistics
on the model's behaviour - keyed watermarks and extraction - and the reason is
structural rather than temporary: cryptography proves things about a computation
you were allowed to observe, and the entire problem is that you were not.

```beat
id: v7-b9
type: predict
concept: d-audit-landscape
prompt: |
  A prospective client - a large developer - tells you they have solved
  provenance: their next training run executes inside attested confidential
  computing, and they will publish the attestation showing which dataset hash
  was consumed by which binary to produce which weights.

  Before reading on: state precisely what a rightsholder can conclude from
  that attestation, and what they cannot. Then say which of the four audit
  products in this section that client is actually a buyer for.
answer: |
  What it establishes: that a binary with a stated hash consumed a dataset with
  a stated hash and produced weights with a stated hash. Given the dataset
  itself, it also establishes that the published summary of it is accurate,
  which is a real compliance benefit.

  What it does not establish: anything about the CONTENTS of that dataset. The
  hash commits to bytes; it does not certify that those bytes exclude any
  particular work, and a rightsholder cannot check without being given the
  dataset, which is the thing that will not happen. It also says nothing about
  any developer who declines to enroll, which is every adversarial case - the
  mechanism only ever describes cooperating parties.

  Which product: this client is a strong buyer for exclusion certification and
  for disclosure conformance checking. They want to demonstrate absence and
  they want their mandatory summary to be correct, and they are cooperative by
  construction, which is exactly the sales motion those two products need. They
  are not a buyer for the rightsholder-side notary; in that product they are
  the counterparty, not the customer.
rubric: |
  Must contain: (1) the attestation binds binary, dataset and weights by hash
  and says nothing about dataset contents; (2) it applies only to a trainer who
  volunteers; (3) the client is a buyer for exclusion certification and/or
  conformance checking, not for the adversarial notary.
  (1) and (3) = pass. All three = full credit.
  An answer that treats the attestation as proof the data was clean = fail,
  diagnosing V7-M3, and the section must be re-delivered; this is the most
  common confusion in the hardware-provenance conversation and it is one an
  informed counterparty will correct in public.
  An answer that dismisses attestation as useless = fail, and flag the
  overcorrection: it is a real licensing-integrity feature with a real buyer,
  and calling it worthless costs you the friendliest sale in the section.
check: llm
```

## The strongest case against the notary

This volume states the strongest argument against each of its own subjects.
Here is this unit's, in four parts, unhedged.

**One: the past is unprotectable and the past is where the money is.** Stated
already, restated here because it is the strongest objection and it does not have
a technical answer. Watermarks protect only what you publish after embedding
them. Every model in production today is outside the scheme. If the litigation
and licensing value in this field is concentrated in what was scraped between
2019 and 2025, then the notary sells a product for a market that begins in
several years and is worth nothing in the current one. The honest answer is that
this is why the retrospective screening layer exists, that screening is triage
rather than proof, and that a business whose defensible product only matures
later needs a revenue line that does not.

**Two: the adversary gets a move, and the move is cheap.** Fictitious-knowledge
watermarks survive today's pipelines because nobody is filtering for them. If the
scheme became standard, a lab could plausibly detect and drop it: watermarks are
coherent statements about entities that appear in one corpus and nowhere else,
which is a signature a cross-corpus entity-linking pass could search for. There
is a live demonstration of the general risk on the retrospective side, where
fine-tuning on semantics-preserving paraphrases collapses state-of-the-art
membership inference outright - the adversary defeats the audit cheaply. Nothing
guarantees the prospective family is immune. What can be said in its favour:
detecting a fictitious-knowledge watermark requires distinguishing invented
entities from real rare ones at web scale, which is a hard and expensive
classification problem rather than a normalization step; and filtering aggressively
enough to catch them costs real corpus. That is a defensible position, not a
settled one.

**Three: nobody has bought this yet.** No buyer has ever paid into a neutral
attribution marketplace. Standards bodies have zero licensees; crawl-payment
products never left beta; the share of content deals including training rights is
around 40% and falling, while inference-time access deals grow. The audit niche is
genuinely open and it is open partly because there is no proven willingness to
pay. The counterweight is specific rather than hopeful: the regulation supplies a
forcing function with a date and a fine attached, and the exclusion-certification
motion sells to a buyer with a compliance budget rather than to one with a
grievance.

**Four: publishing false content is a real cost to a real business.** A journal
that prints twenty-five articles about instruments that do not exist has printed
twenty-five false articles. At the density the small-publisher arithmetic
demanded, nearly one per cent of the archive is fabricated. For a scholarly
publisher this is not a minor operational concern, it is a threat to the thing
being protected, and any deployment needs an answer: watermarks confined to
clearly demarcated sections, or to metadata-only fields, or to content types
where invention is expected. Each of those weakens the design, because the
watermark works by being indistinguishable from the corpus. This tension is real
and this unit does not resolve it.

**What survives all four.** The construction is the only method in this field
that produces a false-positive rate a hostile expert can reconstruct from
published artifacts without trusting anyone. It works against a closed model
through a plain text API, which is the only access anyone will ever have to the
models that matter. Its statistical content is one z-test, which means the
failure modes are procedural and therefore controllable, rather than
methodological and therefore permanent. And the moat is a ledger that accrues
value by existing early - the one asset in this space that a better-funded
competitor cannot buy, because they cannot buy having started in 2026.

## At the bench: the notary run

Here is what to build after this unit, and what it feeds.

**Build.** Take one source from the tagged corpus - the smallest specialist
source, deliberately, because it is the hardest case and its arithmetic is the
one from the small-publisher example.

Generate a 32-byte key. From that key, deterministically derive: 25 published
fictitious-knowledge watermark documents, each an invented entity with four
attributes in 100 to 200 tokens; 250 control documents of identical construction
that will never enter any corpus; and the assignment of which is which. Write the
protocol document alongside them - test statistic, threshold, family size and
correction, the exact detection prompts, the decoding settings, and the stopping
rule.

**Commit before anything else happens.** Publish the hash of the key and protocol
document as a tag in the repository, with a trusted timestamp. This is step 3 of
the ledger and it is the step the whole build exists to rehearse. Everything after
this point is executed, not decided.

Then run the density arithmetic for that source's token count, inject the
published watermarks at the resulting occurrence count into that source only,
before tokenization, and record the injection in the provenance manifest - which
shard, which document IDs, which duplicate cluster. The manifest is what lets a
verifier confirm the watermarks went where the protocol said they would.

**Train two models, not one.** One on the watermarked corpus. One on the
identical corpus without watermarks, same seed, same schedule. The second model
is a model-level negative control, and it is a luxury you have exactly because
you own the training run - no real engagement ever gets one. Its purpose is
brutal: run the identical detection against it. If the pre-registered test fires
on the model that never saw the watermarks, your null is wrong and every other
number in the report is decoration. That check is the actual scientific result of
the build, more so than a successful detection.

**Detect and report.** Ask the 100 published detection questions and the 1,000
control questions through the same interface you would use against a commercial
API. Compute $p_0$ from the controls, then $\mu_0$, $\sigma_0$, $z$, the
uncorrected p-value, and the family-corrected p-value. Compare against the
committed threshold. Emit the report artifact: commitment hash and timestamp,
revealed key, derivation recipe, protocol as committed, model identity and query
date, raw counts, statistic, threshold, decision - signed, and accompanied by a
verifier script that regenerates both item sets from the key and recomputes
everything from the raw answers.

**Artifact.** One signed report and one verifier script, plus a second report
recording the negative-control model's result. The pair is the demonstration: the
test fires on the model that saw the watermarks and does not fire on the one that
did not, and every number in both is reproducible by a stranger holding only the
published commitment and the revealed key.

**What feeds forward.** The report is the lead exhibit in the synthesis unit -
verification, not the royalty split, is what the demo opens with. The
corpus-registration and fingerprinting layer is the same object the split needed
to merge shell registrants, so the two products share a ledger. And the
negative-control discipline is the same discipline the counterfactual sweep used
to separate signal from seed noise: an evidence claim without a control run is
a claim about nothing.

```beat
id: v7-b10
type: self-explain
concept: d-precommitment
prompt: |
  You are writing the protocol document for the build above, before generating
  the key.

  State what the negative-control model establishes and what it does not, and
  explain why a real engagement cannot have one. Then say what the protocol
  must specify in advance so that the negative-control run is meaningful rather
  than decorative.

  Answer from the unit's text; do not reference any run you have or have not
  performed.
answer: |
  What it establishes: that the pre-registered test does not fire on a model
  that provably never saw the watermarks. That is a direct empirical check on
  the false-positive side of the claim, and it validates the null the controls
  are supposed to define - if the test fires here, the null is wrong and the
  stated p-value is meaningless.

  What it does not establish: anything about the false-positive rate at
  frontier scale, on a different architecture, or on a corpus with different
  content. It is one draw from the null, at one scale, not a measured rate. Nor
  does it validate the watermark's survival through a real pipeline, since both
  models were trained on a corpus you built.

  Why a real engagement cannot have one: it requires training a second model
  identical except for your data, which is the unsamplable null from the
  previous unit. The entire prospective construction exists because that
  experiment is unavailable. Owning both training runs is what makes the bench
  version possible and what makes it a rehearsal rather than an audit.

  What the protocol must fix in advance for it to mean anything: that both
  models will be trained and both will be tested, before either is trained -
  otherwise the negative control is a run you can quietly discard if it
  embarrasses you. It must also fix the identical detection prompts, decoding
  settings, statistic and threshold for both runs, and state that both results
  will be published regardless of outcome. A negative control that is only
  reported when it agrees with you is not a control.
rubric: |
  Must contain: (1) it checks the false-positive side by running the committed
  test against a model that never saw the watermarks, and a fire there
  invalidates the null; (2) a real engagement cannot have one because it
  requires a model trained identically minus your data, which is the
  unsamplable null; (3) the protocol must commit in advance to running AND
  publishing both, with identical prompts, statistic and threshold.
  (1) and (2) = pass. All three = full credit.
  Missing (3) = pass but flag: the learner has the science and not the
  evidentiary discipline, which is the half that is the product.
  An answer claiming the negative control measures the false-positive RATE =
  fail; one run is one draw, and treating it as a rate is the same error as
  treating a single p-value as proof.
check: llm
```

## What you can now do

Four capabilities, and the fourth is the one that is a business.

**You can construct a null instead of estimating one.** Seed a generator with a
secret key, publish some of its outputs into your corpus, keep matched siblings
unpublished, and measure both. The unpublished siblings are the null
distribution, sampled as many times as you care to generate, which is what
converts a suggestive number into a p-value with a defensible false-positive
rate. This is the whole difference between retrospective and prospective
evidence, and it costs a file you never publish.

**You can design a watermark that survives the trip and compute what it will
say.** Coherent fictitious knowledge, four or more attributes, 100 to 200
tokens, 25 or more distinct items, detected by asking the model a factoid
question through a plain API - not random strings, not homoglyphs, not ordinary
sentences. Size it with the density arithmetic: 0.1% of your own corpus, checked
against a floor of about 90 occurrences per watermark, re-planned upward as the
trainer's corpus grows. Then run the test: measure $p_0$ on the controls, form
$\mu_0 = np_0$ and $\sigma_0 = \sqrt{np_0(1-p_0)}$, compute $z$, read the
one-sided tail. And when you can rephrase under the key, pair - the same
documents at the same effect gave $z = 1.67$ unpaired and $8.33$ paired, and
buying that back with volume would have cost twenty-five times the published
content.

**You can turn the statistic into an exhibit.** Register, generate under one
key, commit the key and the full protocol with a public timestamp, publish, run
the committed test unchanged, and emit a signed report a stranger can re-run
from the key alone. Correct for the family size and say what the family was.
Name the dependence you cannot correct for. Refuse, in writing, to report an
uncorrected result as suggestive. Three of the four things that make this
evidence are procedural, which is why the ledger and not the model is the
defensible asset.

**And you can place it honestly in the market.** The regulation makes
disclosure mandatory, coarse, self-reported, six-monthly, and explicitly
unverified - the verifier's chair drawn and left empty. Sell exclusion
certification first, because the developer wants the certificate and pays again
at every refresh. Sell conformance checking second. Build the adversarial notary
for the moat, knowing the sale is hardest. Say the limitation before anyone asks
it: this protects what you publish from now on, the already-scraped past is out
of reach, and the screening layer that addresses it is triage rather than proof.
Do not mention zero-knowledge proofs except to bury them, do not let anyone
believe attestation certifies contents, and do not confuse an output watermark
with a data watermark in a room where somebody knows the difference.

The next unit measures contribution by gradient, on the ground truth the earlier
sweep produced. This one produced the thing that goes first in the pitch: not the
split, which is a vision, but the report, which is a document somebody can check.

## Notation in this unit

<!-- canon-only -->

Reference, not reading. Return here when a symbol goes blurry; nothing below is
new.

| Symbol | Means | Typical value here |
| --- | --- | --- |
| $n$ | number of detection items on the published set - questions, or documents in a paired design | 100 to 200 |
| $k$ | how many published items came back as hits | 25 of 100 |
| $p_0$ | the null hit rate, measured on the never-published controls | 0.10 to 0.25 |
| $\mu_0$ | the null mean, $n\,p_0$ | 10 |
| $\sigma_0$ | the null standard deviation, $\sqrt{n\,p_0(1-p_0)}$ | 3 |
| $z$ | how many null standard deviations the observation sits from the null mean | 4 to 5 when the design works |
| $p$ | one-sided tail probability corresponding to $z$ | $10^{-5}$ to $10^{-7}$ |
| $\alpha$ | the committed threshold, family-wide | 0.05 |
| $m$ | family size - how many tests the correction covers | 25 to 200 |
| $\alpha'$ | per-test threshold after Bonferroni, $\alpha/m$ | 0.00125 |
| $\text{SE}$ | standard error of a mean - the spread divided by $\sqrt{n}$ | 0.012 to 0.060 |
| $N$ | model parameters, used only in the training-cost line | $10^9$ |
| $D$ | training tokens, used only in the training-cost line | $2 \times 10^{10}$ |
| $C$ | training cost in FLOP, $6ND$ | $1.2 \times 10^{20}$ |

The four formulas, restated with every symbol defined above:

$$p_0 = \frac{\text{control hits}}{\text{control items}} \qquad \mu_0 = n\,p_0 \qquad \sigma_0 = \sqrt{n\,p_0(1-p_0)} \qquad z = \frac{k - \mu_0}{\sigma_0}$$

$$\alpha' = \frac{\alpha}{m} \qquad \text{SE} = \frac{\text{SD}}{\sqrt{n}} \qquad z = \frac{\text{effect}}{\text{SE}} \qquad C = 6ND$$

The one-sided normal tail, for converting between $z$ and $p$ in either
direction:

| $z$ | one-sided $p$ |
| --- | --- |
| 1.645 | 0.05 |
| 2.00 | 0.023 |
| 2.33 | 0.010 |
| 2.88 | 0.0020 |
| 3.02 | 0.00125 |
| 3.29 | 0.00050 |
| 3.48 | 0.00025 |
| 4.00 | $3.2 \times 10^{-5}$ |
| 4.50 | $3.4 \times 10^{-6}$ |
| 5.00 | $2.9 \times 10^{-7}$ |

The design figures, in one place:

| Quantity | Working value |
| --- | --- |
| attributes per watermark entity | 4 or more |
| tokens per watermark document | 100 to 200 |
| distinct watermarks | 25 or more |
| density in your own corpus | 0.1% or more |
| effective occurrences per watermark | about 90 |
| control set size relative to published set | 10x, since controls are free |
| expected $z$ when it works | 4 to 5 |

Two deliberate simplifications, flagged so they do not read as contradictions
later. First, the test in section 3 treats $p_0$ as known exactly when it was
estimated from a finite control set; the two-proportion form that accounts for
both sources of error gives $z = 4.51$ rather than $5.00$ on the worked numbers,
and the deeper-math track carries the formula. Making the control set large is
what shrinks the difference, and controls cost nothing.

Second, everything here treats detection items as independent. Four attributes
of the same invented entity are not independent - a model that learned the entity
tends to get several of them - which makes $\sigma_0$ an underestimate and $z$
optimistic. The direction is known, the honest fix is to treat the entity rather
than the attribute as the unit of the count, and a report that does not say which
unit it used has left an opposing expert an easy opening.
