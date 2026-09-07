---
unit: v6
title: "What the model remembers, and what counts as evidence"
concepts:
  - d-memorization
  - d-mia-limits
  - d-dataset-inference
  - d-evidence-discipline
assumes:
  - d-data-constrained
  - d-eval-noise
  - d-prob-basics
---

Every unit before this one measured a model. This one asks a different question:
which of those measurements would survive an opposing expert. You will finish
able to say what a model can physically have stored and why, which attacks
recover it and for how much money, why the most popular technique in this
literature cannot produce a proof no matter how good it gets, what does work
once you change the unit of evidence from a sentence to a corpus, and the four
procedural requirements that separate a number from an exhibit.

The split this unit installs runs through the rest of the book. **Retrospective**
evidence goes looking, after the fact, at a model somebody else trained.
**Prospective** evidence is planted before the scrape and collected afterward.
The whole argument here is that only one of those two families can construct the
thing a proof requires. v7 builds the other one. This unit earns the right to
build it.

There is no calculus anywhere in this unit. Counting, averaging, one square
root, and one division.


## Two transcripts, one model

Here are two interactions with the same open-weights model, run five minutes
apart, using the same attack: paste the opening of a document and let the model
continue.

**Transcript one.** The prompt is the first fifty words of a novel that has been
in print for thirty years, quoted in ten thousand reviews, fan-transcribed,
excerpted in study guides, and mirrored on pirate library sites. The model
continues for roughly a page. Not "in the style of." The continuation matches
the published text word for word, through paragraph breaks and dialogue
punctuation, for about 400 tokens before it drifts. Set the two side by side and
you are reading the same page twice.

**Transcript two.** The prompt is the first fifty words of a 2016 workshop
paper on queueing in storage systems. That paper is in the model's training
corpus - you know this because the corpus is public and you searched it. There
are two copies: one on the preprint server, one on the proceedings site. The
model continues for a page. The prose is fluent, plausibly academic, and
completely made up. It invents a related-work section that cites nothing real.
The longest run of words shared with the actual paper is seven, and those seven
are "in this paper we present a system for."

Same model. Same attack. Same fact of membership: both documents were in the
training data.

```beat
id: v6-b1
type: predict
concept: d-memorization
prompt: |
  Before reading on, commit to an answer.

  Both documents were in the training corpus. One comes back near-verbatim and
  one does not come back at all. Which statement names the property of the two
  DOCUMENTS that explains the difference, and draws the right conclusion from
  transcript two about whether the workshop paper was used in training?

  Commit now; being wrong here is the point.
options:
  - text: |
      The difference is how many near-identical copies of the text passed
      through training, relative to the size of the corpus - the novel exists
      in hundreds or thousands of copies, the paper in two - and transcript two
      licenses no conclusion at all about whether the paper was used.
    correct: true
    explain: |
      Right on both halves. Duplication relative to corpus size is the dominant
      dial, which is why reviews, excerpts, study guides and pirated scans put
      the novel in a different regime from a paper with two copies. And the
      second half is the load-bearing one: the paper WAS in the corpus and the
      attack found nothing, because the overwhelming majority of any corpus
      leaves no extractable residue.
  - text: |
      The difference is that the novel is creative prose and the paper is
      technical writing, and transcript two shows the paper either was not used
      or was used without contributing anything.
    misconception: D3
    explain: |
      Genre is not the dial, and the conclusion is the error this whole unit
      exists to correct. You know the paper was in the corpus - the corpus is
      public and you searched it. Extractable content is under about one
      percent of tokens and is skewed toward duplicated and high-entropy
      material, so a null result is exactly what you would predict whether or
      not a document was used. Failure to extract is not evidence of absence.
  - text: |
      The difference is that famous literary works are worth storing, so the
      weights keep a compressed copy of the novel, while the paper survives
      only as statistics - which is why nothing of it comes back.
    misconception: D4
    explain: |
      Both halves of that picture fail. At about 3.6 bits per parameter the
      weights cannot be an archive: the corpus carries far more information
      than the model can hold. And "only statistics" cannot explain transcript
      one, where a page comes back word for word. What separates the two
      documents is duplication, not the value or the genre of the work.
check: choice
```

Two facts have to be held at once, and almost nobody holds both.

**Memorization is real.** Not "in principle" or "in a lab." A study that ran 200
books through 14 open models found that a 70-billion-parameter open model
reproduces certain famous titles essentially completely from a few words of
prompt. That is not a statistical artifact you have to squint at. It is a page
of somebody's book coming out of a weights file.

**Memorization is rare.** The same study found that most of those models
memorize most of those books barely at all, with enormous variance between
models and between books. Broader extraction work puts the extractable fraction
of a training corpus at roughly a tenth of a percent to one percent of tokens,
and that fraction is not a random sample - it is boilerplate and rare
high-entropy strings, the two ends of the distribution, and it systematically
misses the ordinary once-seen document that actually shaped the model.

Both facts are inconvenient for both sides of every argument in this space, and
the rest of the unit is about which claims survive them.


## What memorization is, and what it is not

You need a definition before you can measure anything, and the field has three
of them. They disagree, they are all in use, and the disagreement is where most
bad evidence comes from.

**Definition one: verbatim continuation.** Give the model a prefix from a
document, decode greedily, and check whether the output matches the document's
true continuation for some number of tokens. This is what both transcripts in
the last section did. It is the definition every lawsuit and every headline
uses, because it produces a screenshot.

It is also a measurement artifact, in a way that is easy to demonstrate. A
filter that blocks the model from emitting any 50-token span present in the
training corpus - a perfect verbatim blocker, implemented as a lookup against
the corpus - is defeated by asking for the same content in a different style.
The knowledge is intact; only the surface form moved. So verbatim continuation
measures the surface form of what is retained, not the retention. It is a lower
bound, and a bound that an adversary controls.

**Definition two: probabilistic extraction.** Greedy decoding is one arbitrary
walk through the model's distribution. A document that greedy decoding misses
may still be sitting at high probability. So instead of asking "does the top
choice reproduce it," ask: sampling $n$ continuations at ordinary temperature,
what is the probability that at least one reproduces a span of the true text?
This is the honest measure for a report, for two reasons. It does not depend on
a decoding strategy the reporter chose, and it degrades gracefully - you get a
number between 0 and 1 rather than a yes or a no, and the number is comparable
between documents.

An example of why the distinction matters. Two documents both fail greedy
extraction. Sample 100 continuations from each. One reproduces a 50-token span
of document A in 40 of the 100 samples, and never reproduces document B. Under
the verbatim definition those two documents are identical - both "not
memorized." Under the probabilistic definition A is at 0.40 and B is at 0.00,
which is the difference between a strong exhibit and no exhibit.

**Definition three: counterfactual memorization.** The other two ask what the
model does. This one asks what the model does *because of this document*.

<!-- refutes: D3 -->
**You probably think that if a model cannot reproduce your text, your text was
not used, or at least did not matter.** It is the intuitive test and it is what
both litigation postures reach for: the plaintiff shows the screenshot, the
defendant shows the model refusing.

**Here is the specific prediction that fails.** Under that belief, the set of
documents the model was influenced by would be roughly the set of documents it
can regurgitate. Measure both and they barely overlap. Extraction recovers
boilerplate and rare high-entropy strings - license headers, formatted tables,
hashes, addresses - which is adversarially shaped coverage, not a sample of
influence. Meanwhile at scale, influence becomes *more* abstract, not less: a
model of any real size is measurably shaped by documents that share no
n-grams whatsoever with the outputs they influenced. The two sets are close to
disjoint at the ends that matter.

**Here is why the wrong model is appealing.** Verbatim regurgitation is the only
part of this whole subject that is visible without statistics. You can put it on
a slide. Everything else in this unit requires a null distribution and a
paragraph of explanation, and a screenshot beats a paragraph in every room
either of us will ever be in.

**Here is what is actually true, and it has a definition.** Counterfactual
memorization of a document $x$ is the difference between how well models trained
*with* $x$ recall $x$ and how well models trained *without* $x$ recall $x$ -
averaged over training runs in each condition, because a single run is one draw
from a noisy process. In symbols, with $\mathbb{E}$ meaning the average over
training runs and $r(x)$ meaning some fixed recall score on $x$:

$$\text{cm}(x) = \mathbb{E}_{\text{runs with } x}[\,r(x)\,] - \mathbb{E}_{\text{runs without } x}[\,r(x)\,]$$

That subtraction is the same shape as the leave-one-out counterfactual v4 built
its ground truth from - train with, train without, subtract - pointed at one
document's own recall instead of at held-out loss. It is the causal definition,
it is the only one of the three that means what people think memorization means,
and it costs a retraining sweep per document, which is why nobody uses it above
toy scale and why every number you will ever be handed is definition one or two.

This matters practically because of what counterfactual memorization exposes:
text a model can reproduce that it *would have been able to reproduce anyway*.

**Templatic reconstruction.** A large share of what a naive verbatim test flags
is highly formulaic text - open-source licence headers, standard bibliographic
formatting, HTML skeletons, legal boilerplate, the front matter of academic
papers. A model that never saw your specific copy of the MIT licence emits the
MIT licence perfectly, because it saw a hundred thousand others. The
counterfactual difference is zero. The verbatim match is total. So the match
is not evidence about your document, and an audit that counts it is producing
false positives by construction.

The test that separates them is the one this entire unit turns on, stated here
in its easiest form: **could a model that never saw this document have produced
this output?** If yes, the output is not evidence. Hold that sentence. In
section 5 it comes back as a formal impossibility, and in v7 it comes back as a
design requirement.

```beat
id: v6-b2
type: self-explain
concept: d-memorization
prompt: |
  An audit tool reports that a model "memorized" 4,200 documents from a
  publisher's catalogue, on the basis that for each one, greedy decoding from
  the first 50 tokens reproduced at least a 50-token span of the original.

  You inspect a sample of the 4,200 and find that most of the matched spans
  are the journal's standard copyright footer, its author-affiliation
  formatting block, and its reference-style boilerplate.

  Which explanation of what the tool has and has not shown is correct?
options:
  - text: |
      A model that never saw any of these 4,200 documents would emit that
      footer anyway, having seen it attached to everything else the journal and
      a thousand other journals published, so the counterfactual difference -
      recall with the document minus recall without it - is zero even though
      the verbatim match is total. For a match to be evidence the span has to
      be document-specific, high-entropy text that a model lacking the document
      could not have produced.
    correct: true
    explain: |
      That is the counterfactual test, stated in the form the rest of the unit
      uses: could a model that never saw this document have produced this
      output? For templatic text the answer is yes, so the 4,200 is an upper
      bound made almost entirely of false positives.
  - text: |
      The threshold is too permissive. Raise the required match from 50 tokens
      to 200 and the boilerplate drops out, leaving matches long enough that
      they can only have come from the documents themselves.
    misconception: D3
    explain: |
      Length is orthogonal to the problem. A 500-token licence header is
      reconstructible without your document just as completely as a 50-token
      footer, so a longer boilerplate match is still zero evidence. The filter
      you need is on what kind of content matched, not on how much of it.
  - text: |
      Any verbatim match is a fact about genre statistics rather than about a
      document, since the weights contain nothing of the training data, so
      extraction can never be evidence and the tool's whole category is empty.
    misconception: D4
    explain: |
      That is the correction overshooting into the "just statistics" position,
      which the extraction results refute directly - some heavily duplicated
      books come out of open models close to complete, and extraction is the
      strongest evidence available in this space. What is worthless is
      extraction with no counterfactual filter on what got matched.
  - text: |
      A 50-token exact match is astronomically improbable by chance, so each of
      the 4,200 clears any reasonable significance bar and the tool has found
      4,200 memorized documents.
    misconception: D19
    explain: |
      "Improbable by chance" is a claim about a null you never built. The right
      baseline is not random tokens; it is a model trained without these
      documents, and that model emits the footer with probability near one.
      Against the correct null the match is unremarkable.
check: choice
```


## The arithmetic of what can be stored

Before measuring what a model remembers, settle what it *could* remember. This
is a counting argument, it takes one paragraph, and it rules out one of the two
loud positions in every courtroom in this space.

Start from what a bit is. A bit is one binary distinction - one answer to one
yes-or-no question. Any object that can be in $2^k$ distinguishable states can
carry at most $k$ bits, because to identify which state it is in you must answer
$k$ yes-or-no questions and no fewer. This is not a claim about encodings. It is
counting: $2^k$ messages need $2^k$ distinct states to be told apart, and
anything with fewer states must map two different messages onto the same state,
at which point they are no longer recoverable.

Now apply it to a model. A model is $N$ parameters. Suppose each parameter can
usefully hold $b$ bits about the training data. The finished weights are then
one of at most $2^{Nb}$ distinguishable objects, so the map from *corpora* to
*weights* has at most $2^{Nb}$ possible outputs. If your corpus contains more
than $Nb$ bits of information, two different corpora must produce identical
weights, and the corpus cannot be recovered from the model. The model is not a
copy of the corpus. It cannot be, arithmetically, and no advance in training
technique changes that.

The question is what $b$ is. It is not the storage precision - a parameter
stored in 16-bit floating point does not hold 16 bits *about the training data*,
because most of that precision is spent on where the parameter sits in a
continuous optimization landscape, not on what it distinguishes. The number has
been measured directly, by training models on datasets of pure random strings
(which contain nothing generalizable, so anything learned must be stored) and
finding the corpus size at which the model stops absorbing more:

$$b \approx 3.6 \text{ bits per parameter}$$

Call the product the model's **memorization capacity**:

$$C_{\text{mem}} = N \times 3.6 \text{ bits}$$

where $N$ is the parameter count. This is a ceiling on total stored content, and
it is shared with everything else the model needs to know - every spelling, every
grammatical regularity, every fact, every skill. The corpus-specific residue gets
whatever is left.

<!-- fade: v6-capacity-budget -->

**Worked example.** Take the model from v3: 30 million parameters, trained on a
tagged papers corpus of 400 million tokens.

Capacity:

$$C_{\text{mem}} = 3 \times 10^7 \times 3.6 = 1.08 \times 10^8 \text{ bits}$$

That is 108 megabits, about 13.5 megabytes, for the entire model.

Now the corpus, in the same units. Tokens are not bits, so convert twice. At
roughly 4 bytes per token for English text - the figure the v3 tokenizer
reports, and the figure that makes every bits-per-byte number in this book
comparable - the corpus is

$$4 \times 10^8 \text{ tokens} \times 4 \frac{\text{bytes}}{\text{token}} = 1.6 \times 10^9 \text{ bytes}$$

and its information content, at the compressed rate a decent model achieves on
this kind of text, is about 1 bit per byte:

$$1.6 \times 10^9 \text{ bytes} \times 1 \frac{\text{bit}}{\text{byte}} = 1.6 \times 10^9 \text{ bits}$$

The ratio is the answer:

$$\frac{1.6 \times 10^9}{1.08 \times 10^8} \approx 15$$

The corpus carries about fifteen times more information than the model can hold.
At most one fifteenth of it - under 7% - can be stored in any recoverable sense,
and that budget is competing with everything else the model has to be.

<!-- refutes: D4 -->
**You probably hold one of two clean positions about weights and copies.** Either
the weights are a compressed archive of the training data, which is what
"it memorized the internet" means and what makes the infringement story simple;
or the weights contain nothing of the training data, they are "just statistics"
about it, which is the engineer's instinct and the defence's brief.

**Here are the two predictions, and both fail.** The archive position predicts
that a sufficiently clever query recovers arbitrary training documents. The
arithmetic above forbids it: at frontier scale the ratio is far worse than
fifteen, not better, because corpora grew faster than parameter counts. A
1-trillion-token corpus against a 100-billion-parameter model is roughly
$4 \times 10^{12}$ bits of content against $3.6 \times 10^{11}$ bits of
capacity - the corpus is an order of magnitude too big, and that is before any
of the capacity is spent on being able to speak English. The "just statistics"
position predicts that no document is recoverable. Transcript one refutes it in
one screenshot, and the 200-book study refutes it systematically: some books
come out of some open models essentially complete.

**Here is why both are appealing.** Each is the position its side needs, and
each is defensible from a real observation - one from the capacity arithmetic,
one from the extraction demonstrations. Neither side has an incentive to state
the distribution.

**Here is what is actually true.** Memorization is real, hard-bounded by
capacity, and concentrated almost entirely on content that is either duplicated
many times or is high-entropy and short. It varies enormously by model and by
document, so much so that the study measuring it concluded it helps neither
litigation side cleanly. The honest statement is a distribution over documents,
not a verdict about weights. If you are asked "does the model contain a copy,"
the answer is that a specific list of documents is extractable, here is the
list, here is the probability for each, and here is the corpus-wide rate, which
is under one percent.

```beat
id: v6-b3
type: completion
concept: d-memorization
prompt: |
  A different run: a 100-million-parameter model trained on a 1-billion-token
  corpus. Use 3.6 bits of memorization capacity per parameter, 4 bytes per
  token, and 0.9 bits of information per byte for this corpus.

  Step 1. Model capacity:      $10^8 \times 3.6 =$ ____ bits
  Step 2. Corpus in bytes:     $10^9 \times 4 =$ ____ bytes
  Step 3. Corpus in bits:      (step 2) $\times\ 0.9 =$ ____ bits
  Step 4. Ratio corpus:capacity: (step 3) / (step 1) = ____

  Which filling of the four blanks is right, and what does the arithmetic then
  establish about whether any document in the corpus is recoverable?
options:
  - text: |
      $3.6 \times 10^8$ bits; $4 \times 10^9$ bytes; $3.6 \times 10^9$ bits;
      ratio 10. No - the ratio establishes only that the corpus as a whole
      cannot be stored, and says nothing about how the surviving budget is
      allocated across documents.
    correct: true
    explain: |
      The arithmetic and the conclusion both hold. A bound on the total is an
      aggregate constraint: it is entirely consistent with a handful of heavily
      duplicated documents being stored almost completely while everything else
      leaves no trace. The 200-book extraction results sit comfortably inside a
      ratio of 10.
  - text: |
      $3.6 \times 10^8$ bits; $4 \times 10^9$ bytes; $3.6 \times 10^9$ bits;
      ratio 10. Yes - the corpus is ten times too large to fit, so nothing in
      it is recoverable and the weights hold only statistics about the data.
    misconception: D4
    explain: |
      The four numbers are right and the conclusion is the "just statistics"
      half of the copies misconception. Capacity is not spread evenly over the
      corpus; it concentrates on duplicated and high-entropy content. A ratio
      of 10 forbids storing the corpus, not storing a particular novel.
  - text: |
      $3.6 \times 10^8$ bits; $4 \times 10^9$ bytes; $3.6 \times 10^9$ bits;
      ratio 0.1. Yes, in the other direction - capacity exceeds what the corpus
      needs, so the model holds the corpus outright and any document can be
      recovered with the right prompt.
    misconception: D4
    explain: |
      The ratio asked for is corpus over capacity, $3.6 \times 10^9$ divided by
      $3.6 \times 10^8$, which is 10 - the corpus is the larger object. This is
      the archive position, and the counting argument rules it out: with fewer
      distinguishable weight states than corpora, two different corpora must
      map to the same weights.
check: choice
```

**Three dials set where the surviving budget goes.** All three have been measured
and all three are close to log-linear, meaning each *doubling* of the input buys
a roughly constant increment of memorization rather than a proportional one.

**Duplication.** The number of times near-identical copies of a document pass
through training. This is the dominant term and the one that explains both
transcripts in section 1. Deduplicating a corpus cuts measured memorization by
roughly a factor of three - which is why v2 treated dedup as a decision with
consequences rather than hygiene, and why the duplicate-cluster records built
there are the input to every extraction experiment in this unit.

**Model size.** Bigger models memorize more of the same corpus, which follows
directly from the capacity arithmetic: more parameters, more bits, more room
after the generalizable structure is paid for.

**Prompt length.** Longer prefixes extract more. A 50-token prompt and a
500-token prompt into the same model on the same document give materially
different answers, so any reported extraction rate that does not state its
prompt length is not comparable to any other.

And then the governing law, which is the one that matters for designing anything:

**Memorization risk depends on frequency relative to corpus size, not on
absolute count.** A document repeated 100 times in a 100-billion-token corpus
and the same document repeated 100 times in a 500-billion-token corpus are not
in the same situation. The second is five times more dilute and correspondingly
harder to extract; restoring the first situation takes roughly five times the
duplication. Dilution is a validated mitigation for a lab that wants to reduce
memorization, and it is a moving target for anyone who wants to plant something
detectable. That asymmetry is the whole reason v7's trap design has an
arithmetic section.

```beat
id: v6-b4
type: compute
concept: d-memorization
prompt: |
  You verified in your own from-scratch training run that a marked document
  becomes reliably detectable at 90 occurrences in a 20-billion-token corpus.

  The model you actually want to test was trained on 1.5 trillion tokens.
  Memorization risk depends on frequency relative to corpus size.

  How many occurrences of the same document would you need in that corpus to
  reach the same relative frequency? Give a single number.
answer: 6750
check: numeric(1)
```

Run the second half of that calculation, because it is the one that decides
whether a plan is real. Each occurrence is a document of, say, 150 tokens. So
6,750 occurrences is roughly one million tokens of planted material. If the
catalogue you control is 40 million tokens, that is 2.5% of your own corpus -
large, conspicuous, but achievable. If the catalogue you control is 400,000
tokens, the requirement exceeds your entire holdings and the plan is dead
before it starts. The viability of a trap is a division you can do in advance,
and the numerator moves with somebody else's corpus size.

**One dial, two outcomes.** v3 established that repeating a fixed corpus for up
to about four epochs costs almost nothing against fresh tokens, which is what
makes a small specialist corpus support a real training run. That is the same
dial as duplication. Turning it up buys capability from a corpus you already
have, and simultaneously makes that corpus more extractable from the finished
model. In v3 the epoch count was a budget parameter. Here it is the setting that
determines how much of your own training data is recoverable evidence that you
trained on it. Whether that is a liability or a feature depends on which side of
the table you are sitting on, and it is set deliberately in both cases.


## Extraction as evidence

Extraction is the one method in this unit that does not need a null
distribution, because it is not an inference. It is a demonstration. You put a
prefix in, a page of somebody's book comes out, and the exhibit is the
transcript. Nobody has to accept a statistical model to read it.

Two distinctions before the numbers.

**Discoverable versus extractable.** Discoverable memorization is what you find
when you already hold the document and use its true prefix as the prompt: it
measures what is in there. Extractable memorization is what an adversary with no
copy of the document can pull out: it measures what leaks. Discoverable is
always the larger number, and the two are routinely reported as if they were the
same quantity. A rate quoted without saying which one it is, and with what
prompt length, is not a comparable number.

**Aligned models hide rather than remove.** This one is worth the paragraph
because it is the single most common wrong inference in the field.

<!-- refutes: V6-M2 -->
**You probably think a model that refuses to continue your book has had the
memorized content removed by alignment.** The refusal looks like the fix
working. The lab says the model will not reproduce copyrighted text, you test
it, it declines, and the natural conclusion is that post-training scrubbed the
content out.

**Here is the prediction that fails.** Under that belief, no prompting strategy
recovers the text, because there is nothing left to recover. In fact a
divergence attack - a prompt that pushes the model off its aligned distribution
and back toward raw continuation behavior - raised the rate at which a deployed
chat model emitted training data by two orders of magnitude, and produced over
ten thousand distinct memorized examples for roughly two hundred dollars of API
spend (cost as of the study's 2023 pricing; the order of magnitude, not the
figure, is the durable part). More recent work on a heavily memorized novel
found one frontier model reproducing over 95% of it once jailbroken, another
reproducing over three quarters of it *without* any jailbreak, and a third
refusing almost entirely - three models, three refusal behaviors, and the
refusal behavior told you nothing about what was in the weights.

**Here is why the wrong model is appealing.** Refusal is the only signal a user
can see, and it is genuinely the mechanism the lab shipped for exactly this
problem. It also works, in the sense that ordinary use will not surface the
content.

**Here is what is actually true.** Alignment changes the model's output
distribution over the prompts alignment anticipated. It does not edit the
weights toward forgetting. A refusal is evidence about the safety layer, and it
is evidence about nothing else. Practically: never accept a refusal as a
negative result in an audit, and never report your own model's refusal as
evidence of non-memorization, because the first competent adversary will produce
the transcript that contradicts you.

**The numbers, and their variance.** Extraction rates on open models sit in the
range of half a percent to one and a half percent of training tokens under
adversarial prompting - on the order of one token in a hundred, distributed as
described above. Between models trained on similar data the rates differ by a
factor of three or more. Between documents in the same model, the variance is
far larger than that: the 200-book study found most books barely memorized by
most models, with a handful of book-model pairs at near-complete recall. There
is no such thing as "the memorization rate of a model" as a single number that
predicts what happens to your document.

**Why it is the strongest legal evidence anyway.** As of mid-2026 the pattern
across decided cases is that courts credit documentary proof of acquisition
first - purchase records, download logs, evidence that a pirated library was
obtained - and demonstrated verbatim extraction second. Statistical membership
inference has no track record at all. A forthcoming law-review analysis argues
that courts will likely find a model contains a copy of a work only where it is
straightforward to extract that work from outputs, which makes extraction not
merely persuasive but close to the operative legal test. And it is nearly free:
running an extraction battery against an open model or a public API is a matter
of a few hundred dollars of inference as of mid-2026, against the cost of any
other item in a litigation budget.

**Why it cannot be the audit.** It fires on high-duplication works only. It is
silent on 99.9% of any corpus. It cannot answer "was my catalogue used," only
"here are the three items from my catalogue that come back." Budget it in every
engagement as the exhibit; expect it to fire rarely; never build a product whose
core claim requires it to fire.

```beat
id: v6-b5
type: predict
concept: d-memorization
prompt: |
  A rightsholder hands you 300 documents from their catalogue and asks whether
  a particular open model was trained on them. You run a full extraction
  battery: greedy and sampled continuations, prompt lengths from 20 to 500
  tokens, a divergence attack (the model declined outright on a handful of
  those prompts), and probabilistic extraction over 100 samples per document.
  Nothing comes back on any of the 300.

  Before reading on, commit: which statement is what you now know, and what may
  go in the report?
options:
  - text: |
      You know this model does not extractably memorize these 300 documents
      under these attacks - a fact about extractability, not about membership -
      and because the base rate of extractable content is under about one
      percent of tokens and skewed toward duplicated material, the test has
      essentially no power for ordinary documents. The report says "extraction
      did not fire" as a property of the method, and never says the model does
      not appear to have been trained on this material.
    correct: true
    explain: |
      Exactly the distinction the engagement turns on. The exhibit route is
      closed and that is a genuine finding; the membership question is
      untouched. Next moves: change the unit of evidence to the collection, or
      plant evidence in advance for anything published from here on.
  - text: |
      The battery was thorough and every one of the 300 came back empty, so you
      have good evidence the catalogue was not in the training corpus, and the
      report should say so while noting the method's limits.
    misconception: D3
    explain: |
      Re-read transcript two from the opening: that workshop paper WAS in the
      corpus and produced exactly this result. A test with no power for typical
      documents produces the same null whether or not the documents were used,
      so 300 for 300 is uninformative. That sentence will be quoted back at you
      and it is not supported.
  - text: |
      The refusals on the divergence prompts are the informative part: the
      content has been scrubbed by alignment, so it is no longer in the
      weights, and the report should say the material is not present in the
      model.
    misconception: V6-M2
    explain: |
      A refusal is evidence about the safety layer and nothing else. Alignment
      reshapes the output distribution over anticipated prompts; it does not
      edit weights toward forgetting. Divergence attacks have raised emission
      of training data by two orders of magnitude against models that refused
      politely, and one frontier model reproduced over three quarters of a
      memorized novel with no jailbreak at all.
  - text: |
      Extraction is the weak instrument here, so run a loss-based membership
      attack on the 300 and report its per-document $p$-values as establishing
      membership at a stated confidence.
    misconception: D2
    explain: |
      Aggregating to the collection is the right next move; reporting a
      per-document membership $p$-value as a confidence statement is not. That
      claim needs a false-positive rate, which needs the distribution of the
      statistic over models trained without the document - and you cannot
      sample it without retraining. That argument is the next section.
check: choice
```


## Why membership inference cannot prove training

This is the most important single argument in the book. It is not a hard
argument. It is four steps, each of which you will agree to, and the conclusion
eliminates the entire family of techniques that most people in this space are
currently building on. Read it slowly.

**Step 1: what a proof is, operationally.** Strip away the statistics. You have a
procedure. It takes a document and a model and returns a verdict: trained on it,
or not. The procedure is sometimes wrong, and there are exactly two ways to be
wrong - saying yes when the truth is no, and saying no when the truth is yes.

Name the rates. Of all the cases where the truth is "not trained on," the
fraction where your procedure says yes is the **false positive rate**, written
FPR. Of all the cases where the truth is "trained on," the fraction where your
procedure says yes is the **true positive rate**, TPR. Those two numbers are the
complete description of a detector's quality, and a detector reported without
its FPR has not been reported at all.

<!-- refutes: V0-M1 -->
The word "proof" cashes out entirely as a low FPR, and the most common error in
this whole subject is misreading what that buys. **You probably hear "this model
was trained on my book, $p < 0.001$" as "there is a 99.9% chance the model was
trained on my book."** Here is the prediction that fails: if $p$ were the
probability the claim is true, computing it would require a prior over how
likely training was before you looked - and the test never asked for one. It is
appealing because $p$ is the number attached to the claim, and "$p < 0.001$"
gets spoken aloud as "99.9% sure." What the claim actually says is: *a procedure
applied to a model that had not seen this book would produce a result this
striking less than one time in a thousand.* It is a statement about the
procedure's behavior in a world where you are wrong, not a measurement of how
likely you are to be right. Small $p$ licenses rejecting the null and nothing
more - which is exactly why the null has to be real, and why the next two steps
are the whole argument.

**Step 2: what an FPR requires you to possess.** Read that sentence again and
notice what it presumes. To state the rate at which the procedure fires on
models that were not trained on the book, you have to know how the procedure
behaves on models that were not trained on the book. Not guess. Know - as a
distribution, with a spread, because your test statistic is a number and you
need to know which values of it are ordinary and which are extreme.

That distribution has a name. The **null distribution** is the distribution of
your test statistic in the world where the effect is absent. Here the effect is
"this model trained on this document," so the null is: the distribution of your
statistic across models that did not train on it.

Everything in this unit and the next reduces to one question: *where does your
null distribution come from?*

**Step 3: you cannot sample this null.** To sample it directly you would need
models identical to the target in every respect except that they never saw your
document. That means retraining the target model with the document removed -
and more than once, because the answer moves with the random seed, so a single
run gives you a point and not a distribution.

Count what that requires: the full training corpus, which you do not have; the
training recipe, which is not published; the compute, which is millions of
dollars; and several runs, not one. You have none of the four. Nobody outside
the lab does, and the lab has no reason to run it. There is no clever technique
that gets around this, because the missing ingredient is not a technique. It is
a counterfactual model that does not exist.

**Step 4: the substitute, and why it fails.** Everyone makes the same
substitution, and it is a reasonable thing to try. Instead of retraining, take
documents you believe were *not* in the training set - published after the
model's cutoff, or from a source you think was never crawled - measure your
statistic on those, and call that spread the null.

That substitution carries two assumptions, and both are load-bearing:

1. Those documents really are non-members. You do not have the corpus, so this
   is a belief, not a fact. Anything syndicated, quoted, mirrored, or
   republished may be in there.
2. Those documents are **exchangeable** with yours - identical in every respect
   that moves your statistic, except membership.

Assumption 2 is where the field died, and the way it died is worth walking
through because the same mistake is available in every audit you will ever
design.

The standard benchmarks were built by taking text from before a model's training
cutoff as members and text from after the cutoff as non-members. Sensible on its
face. But now every non-member is also *newer*. A language model assigns lower
loss to text about topics it has seen, entities it knows, events it can place,
and writing conventions current in its training window. Date and membership are
perfectly confounded, and a detector rewarded for separating the two groups will
happily learn the date.

The demonstration is the cleanest experimental result in this literature. Build
a **blind classifier**: a classifier that looks only at the text and never
queries the model at all. It cannot possibly detect membership, because it has
no access to the thing membership is a property of. Run it on the benchmarks.
It beat published state-of-the-art membership attacks across eight datasets.

Sit with that. A detector that cannot see the model outperformed detectors that
could. Whatever those benchmarks were measuring, it was not a property of the
model. It was a property of the text, and the property was the date.

When the confound is removed - matched members and non-members drawn from the
same distribution, evaluated across a family of open models from 160 million to
12 billion parameters on the corpus they were actually trained on - the attacks
sit barely above random. The published method line (a progression of increasingly
refined loss-based statistics) shows real relative gains, and they are relative
gains against an invalid ground truth.

<!-- refutes: V6-M3 -->
**You probably think "held out" means "not trained on," and that any text the
model did not see is a valid control.** It is what the phrase means in every
other part of machine learning, where you make the split yourself.

**Here is the prediction that fails.** Under that belief, replacing your control
set with any other set of confirmed non-members would leave your result
unchanged, because all non-members are equally non-members. They are not. Swap
post-cutoff Wikipedia for same-period Wikipedia from a domain you know was never
crawled, and published attack performance collapses toward chance - the number
moved by more than the effect being measured, which means the control set was
carrying the result.

**Here is why it is appealing.** In supervised learning you construct the split,
so the only difference between train and test really is which side of the split
a sample landed on, by construction. Here you did not construct anything. You
inherited a corpus you cannot see and are trying to reconstruct the split after
the fact.

**Here is what is actually true.** A control set is valid only if it matches the
test set in every respect that moves the statistic, with membership the single
difference. Date is the confound that has broken this field, and it is not the
only one available - topic, register, formatting, source, and length all move
loss. Constructing a matched control is an experimental-design problem, not a
data-collection problem, and it is the thing that separates the methods that
work in the next section from the ones that do not. When you cannot construct a
matched control, you do not have a test.

**Three further results, each independently fatal.**

**Per-document decisions are coin flips under training noise.** Even where a
signal exists, the decision for an individual sample is statistically
indistinguishable from a coin flip under training randomness alone. Change only
the seed - same corpus, same data, same recipe - and a document flips from
flagged to unflagged. You met this instability in v3 as $\sigma$, the seed
spread on held-out bits per byte, and in v4 as the noise floor that a
contribution has to clear. It is the same phenomenon: a quantity measured from
one training run is one draw from a distribution over runs. For a per-document
membership verdict, that distribution is wide enough to swamp the verdict.

**The regime where the signal exists is not the regime you care about.** The
capacity law from section 3 predicts where membership signal survives, and the
prediction is in tokens per parameter: once training exceeds roughly 100 tokens
per parameter, membership inference performance falls to chance. Frontier models
train at a thousand to ten thousand tokens per parameter. The models most people
want to sue sit one to two orders of magnitude past the point where the signal
is gone, and they are moving further past it every year, because that is the
direction the economics point.

**An adversary defeats the audit for pocket change.** Fine-tune the model on
paraphrases of its own training data - semantics preserved, surface form
changed - and state-of-the-art membership attacks collapse. The model keeps
everything it knew. The audit stops working. A 2026 assessment of this concluded
that membership inference is insufficient as a standalone mechanism for
copyright auditing, and the reason is not that the attacks are weak; it is that
they are cheap to defeat without giving anything up.

<!-- refutes: D2 -->
**Now the misconception, in full.** **You probably think a membership-inference
attack can prove a model was trained on your data.** The papers report AUC
numbers that look like detector performance. "The model's loss on my text is
suspiciously low" feels like evidence, and it feels like the kind of evidence
that gets stronger with better methods.

**Here is the prediction that fails.** A proof needs a false-positive rate; an
FPR needs the null "the model was not trained on my data"; and you cannot sample
that null without retraining the model. That is steps 1 through 3, and no
improvement in the attack touches any of them - a better statistic computed
against a null you do not have is a better number with the same status. And
empirically, on properly matched benchmarks, classifiers that never query the
model beat published attacks.

**Here is why it is appealing.** AUC is reported like accuracy, papers show
improvements over prior work, and improvement over prior work reads as progress
toward a goal. The field genuinely is making relative progress. It is relative
progress on a benchmark whose labels are confounded.

**Here is what is actually true, and the founder-facing version.** Post-hoc
membership inference gives suggestive evidence at best, on collections rather
than documents, with no defensible false-positive rate and a per-sample verdict
that flips with the seed. Building a product whose core claim is "our attack
proves your data was used" means standing in front of a technical buyer who has
read the blind-classifier paper. There are two sound routes and this unit and
the next are about them: aggregate to the collection level, where a null can be
borrowed from a matched held-out set; and plant keyed evidence before the
scrape, where the null is constructible by design.

```beat
id: v6-b6
type: self-explain
concept: d-mia-limits
prompt: |
  A well-funded team proposes to solve this. Their plan: collect a much larger
  and cleaner benchmark, train a far better membership classifier on it,
  validate carefully, and publish an attack with an AUC of 0.95.

  Suppose they succeed at everything they set out to do. Which account
  correctly explains what the resulting system can tell a rightsholder about a
  particular model and their book, and names the change that would fix it?
options:
  - text: |
      A confidence statement is a statement about the false-positive rate,
      which is defined against the null - the spread of the statistic across
      models that did not train on the book - and sampling that null means
      retraining the target model without the book several times, which needs
      the corpus, the recipe, the compute and multiple seeds. A better
      classifier scored against an unavailable null has the same status as a
      worse one. The fix is to the plan, not the method: generate content under
      a secret key, publish part of it, keep matched siblings unpublished, and
      commit the key with a timestamp before anyone sees the model.
    correct: true
    explain: |
      That is the four-step argument and its exit. The benchmark's non-members
      are documents; the null they need is over training runs. The unpublished
      siblings are drawn from the same distribution by the same procedure and
      were certainly never trained on, so they are the null rather than a proxy
      for it, and can be sampled without limit.
  - text: |
      The trouble is that existing benchmarks are unrealistic; a large, clean,
      carefully validated one fixes the labels, and an attack that reaches AUC
      0.95 on it does support a stated confidence for a particular model and
      book.
    misconception: D2
    explain: |
      No dataset is a counterfactual model. Even with perfect labels the
      benchmark tells you how well the classifier separates two groups of
      documents; the required object is the behaviour of the procedure on
      models trained without the book. Two further facts survive any benchmark
      improvement: the per-document verdict flips with the training seed alone,
      and paraphrase fine-tuning collapses the attack while the model keeps
      everything it knew.
  - text: |
      An AUC of 0.95 means the detector is right about 95% of the time, so the
      confidence attaches directly and the rightsholder can be told there is
      roughly a 95% chance their book was used; the plan needs more validation,
      not redesign.
    misconception: V6-M1
    explain: |
      AUC is the probability that a randomly chosen member outscores a randomly
      chosen non-member, averaged over every threshold including ones nobody
      would operate at. It is not accuracy and it is not a posterior
      probability that a claim is true. The operational number is TPR at a low
      fixed FPR, and it is routinely far worse than the AUC suggests.
  - text: |
      No statistical procedure can ever establish what a model was trained on,
      so the plan is unfixable in principle; the only real evidence is a
      verbatim extraction transcript.
    misconception: D3
    explain: |
      The obstacle is specific, not universal: this null cannot be sampled
      because the counterfactual model does not exist. Construct the null in
      advance with keyed content and never-published controls and you get a
      defensible false-positive rate - which is exactly what v7 builds. And
      extraction fires on well under one percent of any corpus, so it cannot
      carry the work alone.
check: choice
```

**One reading skill before the next section.** You will be handed AUC numbers,
so you need to know what AUC is and what it is not.

A detector produces a score for each document - a loss, a rank statistic,
anything - and you flag a document when its score crosses a threshold. Sliding
the threshold from permissive to strict traces out pairs of (FPR, TPR), and the
curve those pairs make is the **ROC curve**. The area under it, **AUC**, has one
exact interpretation: it is the probability that a randomly chosen member scores
higher than a randomly chosen non-member. A coin flip gives 0.5.

<!-- refutes: V6-M1 -->
**You probably read AUC 0.7 as "right 70% of the time."** It is a number between
0.5 and 1 attached to a detector, so it reads as a grade.

**Here is the prediction that fails.** Under that reading, a detector at AUC 0.70
would be usable for accusations at roughly a 30% error rate - unpleasant but
workable. Now compute what it does at the operating point an accusation
requires. On a real detector at AUC around 0.7, the true positive rate when the
false positive rate is held to 1% is often in the low single digits. A detector
that flags 1.2% of members while flagging 1% of non-members is doing almost
nothing, and the AUC of 0.7 came entirely from the permissive end of the curve,
where you would be flagging half of everything.

**Here is why it is appealing.** AUC is a single number, it is the number papers
report, and 0.7 sounds like a passing grade in every other context where numbers
between 0 and 1 appear.

**Here is what is actually true.** AUC is an average over all thresholds,
including every threshold nobody would ever operate at. Nothing you do with a
detector is an average over thresholds - you pick one, and for an accusation you
pick a strict one. The honest metric is **TPR at a low fixed FPR**, and section
7 computes one. For calibration: published membership attacks in practical
settings sit below AUC 0.7, while a well-designed planted canary reaches roughly
50% TPR at 1% FPR. Those two things are not the same kind of object, and the AUC
column hides the difference.


## Dataset inference: the collection as the unit of evidence

Everything in the last section was about one document. Change the unit and one
thing becomes possible, for a reason that is pure arithmetic.

Suppose your statistic on a single document carries a real but tiny membership
signal: members average a bit higher than non-members, but the gap is small
relative to how much the statistic varies between documents. Write the gap in
units of that variation - the difference in means divided by the standard
deviation - and call it the effect size $d$. Say $d = 0.05$: members sit one
twentieth of a standard deviation above non-members.

For one document that is hopeless. Your observation is a single draw and the
shift is invisible under the noise.

Now average the statistic over $n$ documents from the same collection. Two things
happen, and they happen at different rates. The mean shift stays at $d$ - every
document carries it. The noise in an average of $n$ independent draws shrinks by
$\sqrt{n}$, because variances add and standard deviations are the square roots
of variances. So the signal-to-noise ratio of the average, which is the $z$ you
can test, is

$$z = d\sqrt{n}$$

where $d$ is the per-document effect size in standard deviations and $n$ is the
number of documents in the collection. That single expression is the entire
reason dataset inference works and per-document membership inference does not.

Put numbers in it. At $d = 0.05$:

- $n = 1$: $z = 0.05$. Nothing.
- $n = 1{,}000$: $z = 0.05 \times 31.6 = 1.58$. Still nothing you would report.
- $n = 10{,}000$: $z = 0.05 \times 100 = 5.0$. A $z$ of 5 is a one-sided
  $p$ of about $3 \times 10^{-7}$.

The same signal that is undetectable in a sentence is overwhelming in a
catalogue. This is not a trick and it is not a stronger attack. It is the square
root, applied to a collection instead of a document.

```beat
id: v6-b7
type: predict
concept: d-dataset-inference
prompt: |
  Per-document effect size is $d = 0.04$ standard deviations, and the
  collection-level statistic is $z = d\sqrt{n}$ for $n$ independent documents.

  (a) How many documents do you need to reach $z = 4$?

  (b) Your 10,000 "documents" turn out to be the individual chapters of 250
  books, and a book's chapters share an author, a topic, a formatting template
  and a publication date - so within a book the statistic barely varies.

  Commit to an answer for both before reading on.
options:
  - text: |
      (a) $\sqrt{n} = 4/0.04 = 100$, so $n = 10{,}000$. (b) The $\sqrt{n}$
      shrinkage assumes independent draws, and chapters of one book are close
      to one draw repeated, so the effective sample size is 250 books:
      $z = 0.04 \times \sqrt{250} = 0.63$. Counting chapters inflated the
      reported $z$ by $\sqrt{10000/250} = \sqrt{40} \approx 6.3$.
    correct: true
    explain: |
      Right, and the factor of 6.3 was manufactured rather than measured - it
      moves $p$ from about 3 in 100,000 to nothing at all. The $n$ in that
      formula is the number of independent units, and deciding what the
      independent unit is is a judgement an opposing expert attacks first,
      because it is the cheapest place to find a factor of six.
  - text: |
      (a) $n = 10{,}000$. (b) You have 10,000 measurements, and every one of
      them carries the same mean shift, so $z = 4$ stands; the chapters being
      similar is a property of the catalogue, not a defect in the statistic.
    misconception: D19
    explain: |
      This is the error the opposing expert opens with. Variances add only for
      independent draws; correlated units give you far less noise reduction
      than $\sqrt{n}$ promises, so the honest effective $n$ here is 250 and the
      honest $z$ is 0.63. More rows is not more evidence when the rows repeat
      each other.
  - text: |
      (a) $n = 10{,}000$. (b) The dependence that matters is training
      randomness, not chapter similarity, so average the statistic over several
      training seeds and $z = 4$ at $n = 10{,}000$ survives.
    misconception: D16
    explain: |
      Seed noise is real and it is a different problem - it is why single-run
      contribution numbers need averaging. It does not repair this $z$, because
      the failure here is in the sample: 10,000 chapters of 250 books are not
      10,000 independent observations no matter how many seeds you average.
      Aggregate within each book first, then test across books.
check: choice
```

**What dataset inference actually does.** The published method that works has
three parts, and each part answers a specific failure from the last section.

First, it does not commit to one statistic. It takes a whole battery of
membership features - the family of loss-based and rank-based statistics that
individual attacks each proposed on their own - and *selects and weights* the
ones that carry signal for this particular data distribution. That matters
because which statistic carries signal is not a universal fact; it depends on
the domain, and a method fixed to one statistic is betting on a domain.

Second, it fits that weighting on one part of the data and tests on another. If
you fit the combination and evaluate it on the same documents, you have chosen
the weights that make your data look extreme and your $p$-value is
meaningless - it no longer reflects the probability of a chance result, because
chance was given a thousand attempts to find the best direction. This is the
same discipline as any held-out evaluation, and skipping it invalidates the
number rather than merely weakening it.

Third, it does a real hypothesis test on the aggregate: the members' aggregate
score against the non-members' aggregate score, as a two-sample test, at the
collection level. Reported on the standard open corpus, it separates
trained-on splits from held-out splits at $p < 0.1$ with no false positives.

The reported document-level ceiling elsewhere in the literature tells the same
story from the other side: the same attacks that fail at sentence level start
working when sentences are aggregated into documents, and document-level AUC
against a 7-billion-parameter open model reached about 0.86 on books and about
0.68 on papers. Books aggregate better than papers because a book is longer and
more distinctive. The direction is monotone: the bigger the unit you aggregate
over, the more the signal survives. **The unit of evidence is a corpus, not a
sentence.**

**The dependency that limits it, and the 2026 fixes.** Look at what step three
needs: a set of non-members from the same distribution, private enough that the
model could not have trained on it. That is the same matched-control problem
from the last section, and it is the method's binding constraint. A publisher
with a back catalogue and nothing unpublished has no held-out set, and
constructing one after the fact reintroduces every confound.

Three developments have attacked the constraint, and as of mid-2026 they are the
only retrospective family with a defensible null:

- **Synthetic held-out sets.** Generate non-member text matched to the members'
  distribution rather than collecting it. The construction is under your
  control, so the matching is a design choice rather than a hope.
- **Natural identifiers.** This is the elegant one. Some text contains
  structured random strings that occur naturally - commit hashes, shortened
  URLs, generated identifiers. They are drawn from a known space, and the space
  is enormous, so you can generate unlimited *additional* strings from the same
  distribution that were certainly never published and therefore certainly never
  trained on. Those are a valid null: same distribution, guaranteed
  non-membership, no retraining, no private data. It is the single most useful
  retrospective development of 2026, and notice what it does - it manufactures
  the exchangeability that section 5 said you could not have, by finding a
  corner of the data where the generating distribution is known.
- **Black-box n-gram coverage.** A membership signal computed from generated
  text alone, with no access to log-probabilities, which makes it the only
  branch of this family that functions against a commercial API.

**What to claim.** Dataset inference is strong statistical evidence at the
collection level when the held-out set is valid, and it costs tens of dollars of
inference as of mid-2026. It is also fragile in exactly the way section 5
described: paraphrase fine-tuning degrades it along with everything else in the
retrospective family. Sell it as screening - it tells you which collections are
worth pursuing, it generates leads, and it is honest about being a lead
generator rather than an exhibit. The moment you describe a collection-level
$p$-value as proof that a specific book was used, you have made a claim your own
method does not support, because the test was never about a specific book.


## The discipline that converts a number into evidence

You now have three retrospective instruments and an honest account of what each
one can say. This section is about the four procedural requirements that decide
whether any of that survives contact with someone paid to break it. None of the
four is mathematically hard. All four are where audits die.

<!-- refutes: D19 -->
**You probably think that a detection at $p < 0.05$ is a result.** It is the
number the field reports, small means good, and the whole apparatus of
statistics appears to exist to produce it.

**Here is the prediction that fails.** Under that belief, an audit that tests a
publisher's whole catalogue and returns a list of documents at $p < 0.05$ has
found those documents. Test 10,000 documents against a model that trained on
none of them and roughly 500 will clear $p < 0.05$ by chance alone - that is
what $p < 0.05$ means. Your list is 500 documents long and every entry on it is
noise. An opposing expert does not dispute your arithmetic; they ask how many
tests you ran.

**Here is why it is appealing.** The threshold is a convention, conventions feel
like standards, and the calculation that produces the $p$-value is genuinely
correct for the one test it describes.

**Here is what is actually true.** Four things, and the fourth is the product.

**One: correct for multiple testing.** The reasoning is a union bound and it
takes one line. If you run $n$ independent tests, each with probability
$\alpha$ of a false positive under the null, then the probability that *at least
one* fires is at most the sum of the individual probabilities, $n\alpha$. To
hold the probability of any false positive across the whole family at $\alpha$,
divide: test each one at $\alpha/n$. That is the **Bonferroni correction**, and
its entire derivation is the previous sentence.

<!-- fade: v6-multiplicity-arithmetic -->

**Worked example.** An audit tests 500 documents from a catalogue against one
model.

Expected false positives at an uncorrected threshold of $\alpha = 0.05$, by
linearity of expectation - each test fires with probability 0.05, so the
expected count is the sum of 500 such probabilities:

$$500 \times 0.05 = 25 \text{ documents}$$

Twenty-five documents on your list, from a model that saw none of them.

The corrected per-document threshold for a family-wise error rate of 0.05:

$$\frac{0.05}{500} = 1 \times 10^{-4}$$

Now convert that to the units your detector reports in. A $z$-score is how many
standard deviations an observation sits from the null's mean, and for a
one-sided test the correspondence between $z$ and $p$ is fixed:

| one-sided $p$ | $z$ |
| --- | --- |
| 0.05 | 1.64 |
| 0.01 | 2.33 |
| 0.001 | 3.09 |
| $10^{-4}$ | 3.72 |
| $10^{-5}$ | 4.27 |
| $10^{-6}$ | 4.75 |
| $10^{-7}$ | 5.20 |

So a document in this audit needs $z \geq 3.72$ to be reportable, not the
$z \geq 1.64$ its own $p < 0.05$ suggested. That is a large gap, and it is the
difference between an audit and a list of coincidences.

```beat
id: v6-b8
type: completion
concept: d-evidence-discipline
prompt: |
  An audit tests 1,000 documents from one catalogue against one model, and you
  want the probability of *any* false positive across the whole report held to
  0.01.

  One-sided z-to-p correspondence:
  p = 0.05 -> z = 1.64;  p = 0.01 -> z = 2.33;  p = 0.001 -> z = 3.09;
  p = 1e-4 -> z = 3.72;  p = 1e-5 -> z = 4.27;  p = 1e-6 -> z = 4.75

  Step 1. Expected false positives at an uncorrected $p < 0.01$:
          $1000 \times 0.01 =$ ____
  Step 2. Bonferroni per-document threshold: $0.01 / 1000 =$ ____
  Step 3. One-sided $z$ required to clear it: ____
  Step 4. Your single best document scores $z = 3.1$. Does it clear the bar,
          and what would you change about the audit to make it reportable?
          ____

  Pick the filling that is correct in all four blanks.
options:
  - text: "10 documents; $1 \\times 10^{-5}$; $z = 4.27$; no - $z = 3.1$ is about $p = 0.001$, a hundred times larger than the corrected threshold, so the fix is to shrink the family: pre-register a small set of specific documents before seeing any model output, or test once at the collection level."
    correct: true
    explain: "Right. $n\\alpha = 1000 \\times 0.01 = 10$ expected false positives; Bonferroni divides, $0.01/1000 = 10^{-5}$, which the table puts at $z = 4.27$; and $z = 3.1$ sits an order of magnitude of $p$ short of it. The bar is set by how many tests the family contains and by nothing about that document, so the only honest lever is running fewer tests - and choosing which ones before you look."
  - text: "10 documents; $1 \\times 10^{-5}$; $z = 4.27$; yes - $z = 3.1$ is significant at $p < 0.01$ uncorrected, so report it with the multiple-testing caveat noted in a footnote."
    misconception: D19
    explain: "The arithmetic is right and the conclusion undoes it. With 1,000 tests behind the number, an uncorrected $p$ is not a caveat, it is the finding being wrong: roughly 10 documents clear $p < 0.01$ against a model that saw none of them. A footnote does not change how many tests you ran."
  - text: "10 documents; $1 \\times 10^{-5}$; $z = 4.27$; yes, once you relax the family-wise rate to 0.05 - that moves the per-document threshold to $5 \\times 10^{-5}$ and lets $z = 3.1$ through."
    misconception: D19
    explain: "Two problems. Arithmetically $0.05/1000 = 5 \\times 10^{-5}$ still needs about $z = 3.9$, so $z = 3.1$ misses anyway. More importantly the threshold was chosen after seeing the score, which is the specific move that destroys the evidence: an opposing expert asks when you set it, and there is no good answer."
  - text: "50 documents; $1 \\times 10^{-5}$; $z = 4.27$; no - the expected false-positive count comes from the conventional $\\alpha = 0.05$ applied to the 1,000 tests."
    misconception: D19
    explain: "The expected count is $n\\alpha$ at the threshold you are actually testing at, and the question fixes that at $p < 0.01$, giving 10. Reaching for 0.05 because it is the customary number is the habit this whole section is about: the convention is not the quantity."
check: choice
```

**Two: dependence between test units.** Bonferroni assumed independence, and
your documents are not independent. Chapters of one book, articles by one
author, papers from one lab, pages from one site - all share topic, register,
formatting, and date, which are exactly the things that move a membership
statistic. You saw the arithmetic in the last section: treating 10,000 dependent
chapters as 10,000 independent documents inflates $z$ by the square root of the
ratio between the nominal and effective sample sizes, and a factor of 6 in $z$
is many orders of magnitude in $p$.

The remedy is to identify the independent unit and aggregate within it before
testing across it. The exposure is that identifying the independent unit is a
judgment, it is arguable, and it is the cheapest place for an opposing expert to
find a factor of six. State your choice explicitly, defend it, and report what
the answer becomes under the most hostile reasonable alternative. An audit
report that names its own effective sample size is doing something no opposing
expert can take away from it.

**Three: report TPR at a low fixed FPR, not AUC.** Section 5 gave the reason;
here is the reading procedure.

<!-- fade: v6-tpr-at-fpr -->

A detector was run on 500 documents known to be members and 500 known to be
non-members. Each row is one threshold setting.

| Threshold | Members flagged (of 500) | Non-members flagged (of 500) |
| --- | --- | --- |
| very loose | 410 | 340 |
| loose | 300 | 200 |
| strict | 25 | 20 |
| very strict | 6 | 5 |

Each row gives one point on the ROC curve. TPR is members flagged divided by
total members; FPR is non-members flagged divided by total non-members.

- very loose: TPR $= 410/500 = 0.82$, FPR $= 340/500 = 0.68$
- loose: TPR $= 300/500 = 0.60$, FPR $= 200/500 = 0.40$
- strict: TPR $= 25/500 = 0.05$, FPR $= 20/500 = 0.04$
- very strict: TPR $= 6/500 = 0.012$, FPR $= 5/500 = 0.010$

The last row is the operating point an accusation would use: hold false
positives to 1%. **TPR at 1% FPR is 1.2%.** The detector catches roughly one
member in eighty while wrongly flagging one non-member in a hundred. It is
barely distinguishable from flagging documents at random, which by definition
gives TPR equal to FPR.

Now compute the AUC-shaped impression this same detector creates. On the loose
row it separates 60% of members from 60% of non-members - it looks like it is
doing something, and a summary number averaged across all four rows will land
comfortably above 0.5. Both descriptions are of the same detector. Only one of
them describes the regime you would use it in.

For contrast, a deliberately planted marker of the kind v7 constructs reaches
roughly 50% TPR at 1% FPR - a detector that catches half of what it looks for
while wrongly flagging one in a hundred. That is a different kind of object,
and putting the two in an AUC column side by side hides it.

```beat
id: v6-b9
type: completion
concept: d-evidence-discipline
prompt: |
  A different detector, run on 800 known members and 800 known non-members.

  | Threshold | Members flagged (of 800) | Non-members flagged (of 800) |
  | --- | --- | --- |
  | loose | 640 | 560 |
  | medium | 240 | 160 |
  | strict | 16 | 8 |

  Step 1. loose:  TPR = 640/800 = ____   FPR = 560/800 = ____
  Step 2. medium: TPR = 240/800 = ____   FPR = 160/800 = ____
  Step 3. strict: TPR = 16/800 = ____    FPR = 8/800 = ____
  Step 4. TPR at 1% FPR = ____

  A colleague wants the medium row, "30% detection at 20% false positives,"
  as the headline of the audit. Pick the filling that is correct in all four
  blanks and judges the headline correctly.
options:
  - text: "0.80 / 0.70; 0.30 / 0.20; 0.02 / 0.01; TPR at 1% FPR = 2%. The medium row cannot be the headline: a 20% FPR means one accused document in five is wrongly accused, so the reportable operating point is the strict row and the honest claim is 2% detection."
    correct: true
    explain: "Right. Each rate is per its own denominator - members flagged over 800 members, non-members flagged over 800 non-members - and the strict row is the only one at an FPR an accusation can stand behind. Quoting the medium row makes the detector look fifteen times better by accepting an error rate that would discredit the report."
  - text: "0.80 / 0.70; 0.30 / 0.20; 0.02 / 0.01; TPR at 1% FPR = 2%. The medium row is the right headline, because 30% detection is the highest rate the detector achieves at a threshold that still separates the two groups."
    misconception: V6-M1
    explain: "That is the AUC error in its operational form: a detection rate quoted without the false-positive rate it was bought at is not a quantity. Every threshold 'separates the groups' to some degree; the loose row separates them too, at an FPR of 0.70. You pick one operating point, and for an accusation you pick a strict one."
  - text: "0.80 / 0.47; 0.30 / 0.40; 0.02 / 0.33; TPR at 1% FPR = 2%. The medium row cannot be the headline because its false-positive rate is too high for an accusation."
    misconception: V6-M1
    explain: "The FPRs here are non-members flagged divided by total flagged (560/1200, 160/400, 8/24). FPR is per non-member: the denominator is the 800 non-members, not the flag count. Dividing by flags makes the rate depend on how many members you caught, which is the one thing it must not depend on."
  - text: "0.80 / 0.70; 0.30 / 0.20; 0.02 / 0.01; TPR at 1% FPR = 2%. Report the average across the three rows, about 37% detection, since a single row is an arbitrary choice of threshold."
    misconception: V6-M1
    explain: "Averaging across thresholds is exactly what AUC does, and it is why AUC hides what a detector does in use. Nothing you do with a detector is an average over thresholds - you deploy one, and the average is dominated by permissive settings where you would be flagging most of everything."
check: choice
```

**Four: pre-registration.** The first three are analysis choices, and every
analysis choice is a degree of freedom. If you pick the test statistic after
seeing which one separates your data, choose the threshold after seeing the
scores, decide the independent unit after seeing which choice gives
significance, and choose which documents to report after seeing all of them,
then you have run an enormous number of implicit tests and your stated
$p$-value describes none of them. Nothing in the arithmetic detects this.
Nothing in the report reveals it. The number looks identical either way.

The only fix is procedural: commit publicly, in advance, to the test statistic,
the threshold, the correction, the unit of aggregation, and - for prospective
methods - the key, all timestamped before you see the model. Then the analysis
has no degrees of freedom left and the $p$-value means what it says.

This is not a mathematical improvement. It is a change in what kind of object
your number is. And it is the part of this whole subject that is a business
rather than a technique: the arithmetic is one intro-statistics chapter that
anyone can reproduce, and the timestamped commitment is the thing a counterparty
cannot manufacture after the fact. That asymmetry is v7's entire product thesis,
and these four requirements are its specification.


## Where the null comes from

One question organizes every method in this unit and the next. Not "how strong
is the attack." **Where does your null distribution come from?**

| Method | Where the null comes from | What it supports |
| --- | --- | --- |
| Extraction | Not needed - it is a demonstration, not an inference | Strongest evidence available; fires on a tiny high-duplication subset |
| Post-hoc membership inference | Must be sampled from "models not trained on this"; unsampleable | Suggestive at best; no defensible FPR |
| Dataset inference | Borrowed from a private, distribution-matched held-out set | Strong at collection level, exactly as good as the held-out set |
| Keyed canaries and watermarks (v7) | Constructed by design: unused keys and never-published controls | The only family that supports the word "proof" |

The first three rows are **retrospective**. You go looking, after the fact, at a
model somebody else built, and the null has to be reconstructed from a world you
do not control. Row one dodges the problem by not making an inference. Row two
cannot solve it. Row three solves it partially, by finding or manufacturing a
matched sample - and every 2026 advance in that row is an advance in *null
construction*, not in attack strength, which is the tell.

The fourth row is **prospective**. You act before the scrape: generate content
under a secret key, publish some of it, keep matched siblings unpublished. The
unpublished siblings were generated from the same distribution by the same
procedure and were certainly never trained on, so they *are* the null - not an
estimate of it, not a proxy for it. Sample them as often as you like. The FPR
that section 6 proved unobtainable becomes a property of your own construction.

That is the destination, and v7 builds it. Two things to carry there.

**The permanent gap.** Prospective evidence protects only data published after
the evidence was embedded. The models people most want to sue trained on data
collected years ago, and no key was committed then. That gap never closes. It is
the honest limitation to state first in any pitch, and it is the entire reason
the retrospective row-three tooling has a business at all: it is the screening
layer for everything published before you arrived.

**The four requirements are the same four.** Nulls, multiple-testing correction,
dependence, and pre-registration are not extra rigor bolted onto a canary
scheme. They are its product specification. A watermark with a beautiful signal
and no pre-committed key is worth exactly as much as a loss-based attack with a
good AUC.


## At the bench: three measurements on your own model

Here is what to build after this unit, and what it feeds.

**Build one: extraction, both outcomes.** Use the duplicate-cluster records from
v2 to find the single most-duplicated document in the tagged corpus, and pick an
ordinary document that appears exactly once. For each, prompt the v3 model with
the first 50 tokens and greedy-decode 200, then record the longest common
substring with the true continuation. Then do it properly: 100 samples at
temperature 1 from each prompt, and report the fraction of samples reproducing a
50-token span. Vary prompt length across 20, 50, 200 and 500 tokens so the
numbers are comparable to anyone else's. Expect the duplicated document to fire
and the ordinary one to produce nothing. Feel both.

**Build two: a membership attack that does not work.** Score the trained-on
papers and the reserved held-out papers with a loss-based statistic. Build the
ROC, report AUC *and* TPR at 1% FPR, and expect the second number to sit near
1%. Then use the second v3 seed: run the identical attack against the
second checkpoint and count how many per-document verdicts flip between the two.
That flip count is the honest measure of what a per-document membership claim is
worth on your own bench, and it is the number to have in your pocket when
somebody proposes selling one.

**Build three: dataset inference that does work.** Take 500 trained-on documents
and 500 held-out documents from the same source. Fit the feature weighting on
half of each, test on the other half, and report the collection-level $p$-value.
Then compute the effective sample size honestly: if your 500 documents come from
40 sources, report the $z$ at $n = 40$ alongside the $z$ at $n = 500$ and say
which one you stand behind.

**Artifact.** One page, three rows, each with its method, its number, its
operating point, and its null. This page is the honest statement of what
retrospective evidence gives you, and every weakness in it is a requirement for
v7: extraction fires too rarely, so you need something that fires by design;
membership inference has no null, so you need one you construct; dataset
inference needs a matched held-out set, so you need controls you generate
yourself under a key you committed before anyone saw the model.

```beat
id: v6-b10
type: self-explain
concept: d-evidence-discipline
prompt: |
  You are writing the one-page artifact described above, before running
  anything: for each of extraction, per-document membership inference, and
  collection-level dataset inference, what a positive result licenses, what a
  negative result licenses, and where the null comes from.

  Four drafts of the page are below. Pick the one you would sign, and be clear
  with yourself about what is wrong with each of the others before you read
  on.
options:
  - text: "Extraction: positive shows this document is reproducible from this model, prompt and decoding stated, and needs no null because it demonstrates rather than infers; negative licenses nothing, since under 1% of tokens are extractable and that fraction is skewed to duplicated and high-entropy text. Membership inference: positive says the statistic is unusual against a comparison set you assembled, which is a claim about that set unless it is exchangeable in every respect but membership; there is no valid null, since sampling it means retraining without the document; negative licenses nothing, and the verdict flips with the seed. Dataset inference: positive says this collection scores differently from a matched held-out collection at a stated $p$ and a stated effective $n$, never about an individual document, with the null borrowed from the held-out set; negative is weak and degrades under paraphrase fine-tuning."
    correct: true
    explain: "Right, and the three nulls are the spine of the page: extraction needs none, membership inference cannot have one, dataset inference borrows one and is exactly as good as the match. Scoping the dataset-inference positive to the collection is the sentence that keeps the page defensible."
  - text: "Same as the signed page for extraction and dataset inference, except: membership inference's null comes from documents you have good reason to believe were never crawled - post-cutoff publications and uncrawled domains - so a positive licenses a stated confidence that this document was used."
    misconception: V6-M3
    explain: "That set is a belief about membership, not a sample of the null, and it is the substitution that broke the field: with post-cutoff text as controls, date and membership are confounded, and a blind classifier that never queries the model beat published attacks across eight datasets. A control set is valid only when it matches on everything that moves the statistic."
  - text: "Same as the signed page for membership inference and dataset inference, except: extraction's negative result licenses the claim that the model does not appear to have been trained on this material, since the battery was exhaustive across prompt lengths and decoding strategies."
    misconception: D3
    explain: "Failure to extract is the expected outcome whether or not the document was used - transcript two in this unit is a paper that was demonstrably in the corpus and came back as invention. An exhaustive battery raises the power of a test that has almost none for ordinary documents; the sentence 'does not appear to have been trained on' is the one that gets quoted back at you."
  - text: "Same as the signed page for extraction and membership inference, except: dataset inference's positive licenses the claim that the collection was used and therefore that each document in it was used, at the collection-level $p$."
    misconception: D19
    explain: "The test was never about a document. It aggregated a per-document effect too small to see into a collection-level statistic via $z = d\\sqrt{n}$; nothing in that construction says which documents carried the shift, and pushing the collection $p$ down onto each member is a claim the method does not support."
check: choice
```


## What you can now do

Four capabilities, and they are the ones that make v7 buildable rather than
aspirational.

**You can bound what a model could have stored, in one division.** Parameters
times 3.6 bits against corpus bytes times bits-per-byte. That single ratio
disposes of "the weights are an archive" without argument, and knowing that it
does not dispose of the opposite claim - that a bound on the total says nothing
about the distribution across documents - is what keeps you honest in both
directions.

**You can read a memorization claim.** Which definition, verbatim or
probabilistic or counterfactual. What prompt length. Discoverable or
extractable. How much of the matched span is templatic and therefore
reconstructible without the document. Whether a refusal was mistaken for an
absence. Every one of those is a place where a reported number means something
other than what it appears to mean.

**You can state why membership inference cannot be a product, in four steps.**
A proof is a low false-positive rate; an FPR is a statement about the null; the
null here is models that did not train on your document; those models do not
exist and cannot be sampled. Then the empirical confirmations: blind classifiers
beating published attacks, per-document verdicts flipping with the seed, and the
capacity law putting frontier training regimes past the point where signal
survives. This is the argument that keeps you out of the most common wrong
business in this space, and it is worth being able to deliver in ninety seconds.

**You can convert a number into evidence, or refuse to.** Correct for the family
of tests. Name the independent unit and report the effective sample size.
Report TPR at a fixed low FPR rather than AUC. Commit the whole procedure
before seeing the model. Those four are not rigor for its own sake; they are the
four places an opposing expert attacks, in the order they attack them, and an
audit that handles all four is doing something no competitor's demo does.

The through-line into v7 is one sentence. Every fix in this unit was a fix to
where the null came from, and the only way to have a null you fully control is
to construct it before the data is scraped. That is what a canary is: not a
cleverer attack, but a null distribution you manufactured in advance and can
sample as often as you like. v7 builds it, tests it, and signs the report.


## Notation in this unit

<!-- canon-only -->

Reference, not reading. Return here when a symbol goes blurry; nothing below is
new.

| Symbol | Means | Typical value here |
| --- | --- | --- |
| $N$ | parameters in the model | $10^7$ to $10^{11}$ |
| $C_{\text{mem}}$ | memorization capacity, $N \times 3.6$ bits | $10^8$ to $4 \times 10^{11}$ bits |
| $n$ | number of independent test units (documents, sources, collections) | 1 to $10^4$ |
| $\alpha$ | per-test false-positive threshold | $10^{-6}$ to 0.05 |
| $d$ | per-document effect size: mean shift in standard deviations | 0.02 to 0.2 |
| $z$ | observed value in standard deviations from the null's mean | 0 to 6 |
| $p$ | probability of a result this extreme if the null is true | $10^{-7}$ to 1 |
| FPR | non-members flagged / total non-members | 0.001 to 0.5 |
| TPR | members flagged / total members | 0.01 to 0.9 |
| AUC | probability a random member outscores a random non-member | 0.5 to 0.9 |
| $\text{cm}(x)$ | counterfactual memorization of document $x$ | 0 to 1 |

The five formulas, each restated with every symbol defined above:

$$C_{\text{mem}} = N \times 3.6 \text{ bits} \qquad
z = d\sqrt{n} \qquad
\alpha_{\text{corrected}} = \frac{\alpha}{n}$$

$$\mathbb{E}[\text{false positives}] = n\alpha \qquad
\text{cm}(x) = \mathbb{E}_{\text{with } x}[r(x)] - \mathbb{E}_{\text{without } x}[r(x)]$$

Here $\mathbb{E}[\cdot]$ is an expectation - a probability-weighted average, not
a typical value - and $r(x)$ is any fixed recall score on document $x$.

One-sided $z$-to-$p$ correspondence, the only table you need:

| $p$ | 0.05 | 0.01 | 0.001 | $10^{-4}$ | $10^{-5}$ | $10^{-6}$ | $10^{-7}$ |
| --- | --- | --- | --- | --- | --- | --- | --- |
| $z$ | 1.64 | 2.33 | 3.09 | 3.72 | 4.27 | 4.75 | 5.20 |

Two deliberate simplifications, flagged so they do not read as contradictions
later. First, $z = d\sqrt{n}$ assumes the $n$ units are independent and that the
statistic is approximately normal; the first assumption is the one that fails in
practice and section 7 is about it, and the second is safe for the aggregate of
a few hundred units by the central limit theorem. Second, Bonferroni is the
crudest multiple-testing correction and it is conservative when tests are
positively correlated - a false-discovery-rate procedure is the usual
alternative and controls a different quantity. Bonferroni is used here because
its derivation is one line and an opposing expert cannot argue with a union
bound, which is worth more in an audit report than statistical power.
