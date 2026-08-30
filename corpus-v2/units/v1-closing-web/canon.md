---
unit: v1
title: "The closing web and the correlation problem"
concepts:
  - d-contributive-corroborative
  - d-data-market
  - d-compensation-schemes
assumes: []
---

The web is closing, the money has started moving, and almost none of it moves in
proportion to anything a model actually learned. This unit walks the machinery
that is really in production as of mid-2026 - the paywall a crawler hits, the
cheques that get signed, the judgments that set the price - and gives you the two
tools you need to read any of it. By the end you can place any product or deal in
the ecosystem map, split any "attribution" claim into the two different things
that word means, and run the three-compensation-schemes comparison on a concrete
example and say who wins, who loses, and what each scheme is secretly
proportional to.

## A 402, three cheques, and one answer to pay for

Here is what a crawler sees today at a gated site. First the site's stated terms,
served from `robots.txt`:

```
User-agent: *
Content-Signal: search=yes, ai-input=no, ai-train=no
Disallow:
```

Three usage categories, in plain language, deliberately separated: index me for
search, do not use me to ground an answer at inference time, do not train on me.
Now the fetch itself, from a bot that identifies itself cryptographically:

```
GET /2026/08/article-53-template-explained HTTP/2
Host: lexweekly.example
User-Agent: ExampleBot/1.0 (+https://example.ai/bot)
Signature-Input: sig1=("@authority" "@method" "@path");created=1756512000;
  keyid="poqkLGiujpBegwhMe0aQd1qHtWmwsJq5cqDA6oO4kA";alg="ed25519";tag="web-bot-auth"
Signature: sig1=:K2qGT5srn2OGbOIDzQ6kYT+ruaycnDAAUpKv+ePFfD0=:
```

```
HTTP/2 402 Payment Required
crawler-price: USD 0.01
```

The bot decides the page is worth it and retries, committing to the price:

```
GET /2026/08/article-53-template-explained HTTP/2
Host: lexweekly.example
Signature-Input: sig1=(...);keyid="poqkLGiu...";alg="ed25519";tag="web-bot-auth"
Signature: sig1=:K2qGT5srn2OGbOIDzQ6kYT+ruaycnDAAUpKv+ePFfD0=:
crawler-exact-price: USD 0.01
```

```
HTTP/2 200 OK
content-type: text/html
```

Four mechanisms, all shipping, none hypothetical. The `Content-Signal` line is
Cloudflare's Content Signals Policy (September 2025): machine-readable usage
categories in `robots.txt`. The `Signature` headers are Web Bot Auth, an
Ed25519-signed request per RFC 9421 - crawler identity is a solved problem, and
the signer list includes Anthropic, OpenAI, Perplexity, Google, and Common Crawl.
The 402 with `crawler-price` is Pay Per Crawl: one flat price per request,
site-wide, with the gatekeeper acting as Merchant of Record. A crawler that wants
a ceiling instead of an exact commitment sends `crawler-max-price` and takes
whatever the site quotes below it.

Note what the price is attached to. It is a price per *fetch*, identical for the
newsletter's best investigative piece and its 2019 holiday-hours notice, and it
is charged whether the fetched bytes end up in a training corpus, in a retrieval
index, or in a bucket nobody ever reads. Hold that thought; it is the whole
problem in miniature.

Now the money. Every figure here is as of mid-2026 and every one of them is
public.

| Payer and payee | Amount | Unit being priced |
| --- | --- | --- |
| Google - Reddit | ~$60M/yr | continuous access to a live feed |
| OpenAI - News Corp (5 years) | $250M total | a bundle: content, product placement, peace |
| Amazon - NYT | $20-25M/yr | access |
| Microsoft - Taylor & Francis | $10M | a corpus, once |
| Wiley AI licensing, FY2026 | $49M (+23% YoY) | corpora, once each |
| Perplexity Comet Plus pool | $42.5M | visits, citations, agent actions |
| Bartz v. Anthropic settlement | $1.5B, ~$3,000 per work, once | *acquisition* of ~500,000 pirated copies |
| Shutterstock contributor pro-rata | median $0.0069 per image | share of a pool |
| Adobe Firefly contributor bonus | one contributor reported $4.88 | discretionary |

Nine rows, at least six different units of account, and a range of eight orders
of magnitude between the top and the bottom of the per-item column. The one thing
none of them is: a payment proportional to what the data did to the model.

Now the case that makes the difference bite. One month, one small AI answer
engine, three sources, and a pot of $12,000 to distribute.

The sources:

- **S1, *Lexweekly***: a 400-subscriber specialist newsletter. It published the
  first careful reading of the EU AI Act's training-data disclosure template,
  months before anyone else. Original, long, tiny audience.
- **S2, a wire agency**: a 600-word filing derived from S1's reading, carried the
  same day by 300 downstream outlets, most of which strip the byline.
- **S3, a large reference site**: continuously updated, encyclopedic, carries the
  dates and definitions, mirrored on a hundred scraper sites.

The engine answers 10,000 questions that month. On the question "when does the
Article 53 disclosure obligation start being enforced, and what must the summary
contain?" it produces a correct answer and cites S2 and S3. It does not cite S1.

Three ways to pay out the $12,000:

- **Flat fee**: the pool is split equally among enrolled sources.
- **Citation counting**: each source's share is its share of citations emitted
  across the month's answers.
- **Causal attribution**: each source's share is its measured effect on what the
  model can do - retrain without it, see how much worse the model gets on
  held-out questions in this domain, and split in proportion to that.

```beat
id: v1-b1
type: predict
concept: d-compensation-schemes
prompt: |
  Commit before reading on. For each of the three schemes, name which source
  does best and which does worst, and - this is the part that matters - state
  in one phrase what quantity each scheme is actually proportional to.

  Then answer the harder question: which scheme, if any, pays S1 for having
  been the origin of the analysis?
answer: |
  Flat fee: everyone gets $4,000. Proportional to nothing - it is proportional
  to being enrolled. Best for the smallest source, worst for the largest.

  Citation counting: S2 and S3 do well, S1 does badly. Proportional to
  retrieval placement - how often a retriever ranked you into the context
  window at answer time. That is a function of the index, the embedding model,
  the recency weighting, and your SEO, and it is only loosely a function of
  whether you wrote anything.

  Causal attribution: S3 does best, S1 does worst. Proportional to measured
  marginal effect on held-out loss - which, for a source whose content has been
  copied into other sources, is close to zero, because removing it changes
  nothing the model could not learn from the copies.

  None of the three pays S1 for being the origin. Flat fee pays it for
  existing, citation counting pays it for placement it does not have, and
  causal attribution actively punishes it for having been copied. Origination
  is not a quantity any of these schemes measures.
rubric: |
  Must contain: (1) an explicit proportionality target for each scheme -
  enrollment, retrieval placement, and marginal effect on the model, in any
  wording; (2) the recognition that citation counting is about the retriever
  and not about writing; (3) the answer "none of them" to the origin question,
  with a reason for at least one scheme.
  All three = pass. (1) and (3) = pass.
  An answer that says causal attribution rewards S1 because S1 wrote it first
  = fail, and deliver the duplication argument below slowly: this is the
  single most common wrong intuition about counterfactual attribution.
  An answer that treats citation counting as "the fair one because it tracks
  actual use" = fail, diagnosing D1.
check: llm
```

Now the numbers. Citations that month: S2 6,000, S3 3,600, S1 400, total 10,000.

<!-- fade: v1-pool-split -->

The procedure is the same one every pooled scheme in this book uses, so it is
worth writing once, explicitly. Give each source a weight $w_i$ - a citation
count, a measured contribution, a subscriber count, anything. The source's share
is its weight over the total weight, and its payout is that share of the pool
$P$:

$$s_i = \frac{w_i}{\sum_j w_j}, \qquad \text{payout}_i = s_i \times P$$

Here $w_i$ is source $i$'s weight, the sum in the denominator runs over every
enrolled source, $s_i$ is a fraction between 0 and 1, and $P$ is the pool in
dollars. The shares sum to 1 by construction, which is the property that makes
this a *pooled* scheme: the pot is fixed in advance and the rule only decides how
to cut it. Nobody's payout depends on their own weight alone. It depends on their
weight relative to everyone else's, which is why adding a large source to the
pool reduces what everyone else receives without anyone's behavior changing.

Run it on the citations, with $P = 12{,}000$:

- $s_{S2} = 6{,}000 / 10{,}000 = 0.60$, payout $= 0.60 \times 12{,}000 = \$7{,}200$
- $s_{S3} = 3{,}600 / 10{,}000 = 0.36$, payout $= 0.36 \times 12{,}000 = \$4{,}320$
- $s_{S1} = 400 / 10{,}000 = 0.04$, payout $= 0.04 \times 12{,}000 = \$480$

Now the causal split, which requires a measurement rather than a log. Train the
model, then train it again with S1 removed, and compare how surprised each
version is by held-out text in this domain. That surprise is the loss, and the
one number you need about it here is that it is measured in nats per token: the
average of $-\ln p$, where $p$ is the probability the model assigned to the token
that actually came next. Lower is better, and a *drop* of 0.10 nats when you add
a source means that source made the model measurably less surprised by real text
it had never seen.

Suppose the measured drops are S1 0.02 nats, S2 0.11, S3 0.37. Total 0.50, so the
same pool-split procedure gives 4%, 22%, and 74%: $480, $2,640, $8,880.

And here is the part that no pie chart shows. Retrain each of those four models
with a different random seed and the numbers move by roughly 0.05 nats in either
direction, for reasons having nothing to do with the data. Two of the three
measurements are smaller than that. The "fair" scheme produced a confident-looking
split in which two of three numbers are indistinguishable from zero, and from each
other. Units v3 and v4 make that noise floor measurable; unit v5 shows there is a
proven threshold below which the welfare-optimal contract is a flat fee, and no
better estimator changes that - only more signal does.

Three schemes, three different winners, and the source that did the original work
loses under all three. That is not a story about a badly designed marketplace. It
is what happens when the thing you can measure and the thing you want to pay for
are different quantities, which is the correlation problem this unit is named
after and the reason the rest of this book exists.

One more arithmetic check before leaving this example, because it connects the
pool back to the wall. S1 earned $480 from the pool. Suppose its editor gives up
on marketplaces and meters at the wall instead: the archive takes 400 paid
crawler fetches a month, so matching $480 requires $480 / 400 = \$1.20$ per
fetch, against a Pay Per Crawl going rate measured in fractions of a cent.
Metering pays the long tail nothing because the long tail's traffic is small. The
pool pays it nothing because its share is small. The difference between the two
mechanisms is not the difference between fair and unfair; both are proportional
to volume, and the long tail has none.

## The gatekeeper

The wall in the first section belongs to one company. Here is how that happened,
because the sequence explains the strategy better than any description of it.

- **July 1, 2025 - "Content Independence Day."** AI-crawler blocking becomes the
  default for new domains. Pay Per Crawl enters private beta.
- **September 24, 2025 - Content Signals Policy.** Plain-language `robots.txt`
  directives splitting usage into `search`, `ai-input` (grounding an answer at
  inference time) and `ai-train`. Free, open, unenforceable on its own.
- **September 2025 - AI Index announced.** A pub/sub model where sites publish
  structured updates when content changes, so crawlers can stop re-fetching.
- **January 15, 2026 - Human Native acquired.** A London content-licensing
  marketplace, founded 2024, roughly $3.6M seed, terms undisclosed. The
  gatekeeper now owns the marketplace layer too.
- **July 1, 2026 - Pay Per Use replaces Pay Per Crawl**, plus an Attribution
  Business Insights dashboard. The pitch: charge for value created rather than
  bytes fetched, motivated by the observation that more than half of AI crawler
  requests re-fetch unchanged content.
- **September 15, 2026 - mixed-use crawlers blocked by default** from ad-bearing
  pages, for new and free-tier customers.

Stack those layers up: the chokepoint (traffic passes through the CDN), identity
(Web Bot Auth, which the gatekeeper leads at the IETF), preference expression
(Content Signals), payment rails (Merchant of Record), an index (AI Index), and
now a marketplace (Human Native). One party owns every layer between a
publisher's bytes and a lab's cheque.

<!-- refutes: V1-M3 -->
You probably read that as neutral infrastructure - plumbing, the way a CDN is
plumbing, and therefore a level surface on which a marketplace can be built. Here
is the prediction that fails: if it were neutral plumbing, a competing
marketplace could reach the same publishers on the same terms. It cannot. Reaching
them requires either riding the incumbent's identity and payment layers, on the
incumbent's terms, or reaching the roughly 80% of the web that is not behind that
particular CDN and therefore has no enforcement at all. Publishers already say
this out loud; "vendor-locked" is their word, not a critic's. The belief is
appealing because each individual layer really is open - the signature spec is an
IETF draft, the content signals are plain text in a file anyone can read - and
open specs feel like open markets. What is true is that the specs are open and the
enforcement point is not, and enforcement is the only part with pricing power.

The adoption picture is genuinely strong on the leverage side and genuinely empty
on the payment side. AI crawlers are blocked by default for around 20% of
websites. The gatekeeper claims 50-plus major licensing agreements signed in the
past year using blocking as leverage, with names like the FT, The Atlantic, Ziff
Davis, Conde Nast, AP and People Inc. And yet: Pay Per Crawl never left private
beta, and no frontier lab - not OpenAI, Google, Anthropic or Meta - participates
in either payment scheme.

<!-- refutes: V1-M1 -->
A posted price is not a market price. `crawler-price: USD 0.01` is an ask with no
recorded bid from any buyer that matters. The same pattern repeats one layer up:
Really Simple Licensing (RSL), launched September 2025 by the RSL Collective, put
licensing terms directly in `robots.txt` - free, attribution, subscription,
pay-per-crawl, pay-per-inference - and picked up Reddit, Yahoo, Medium, Ziff
Davis, BuzzFeed, Vox, USA Today Co., People Inc. and AP, with around 1,500 media
organizations endorsing. There is no confirmation that any AI company has agreed
to pay under RSL terms. The IETF's AIPREF working group standardized a preference
vocabulary and a `Content-Usage` header; no major AI company reads it in
production. Expressing preferences is standardized and free. Enforcing them
belongs to the gatekeeper. Getting paid is unsolved.

<!-- refutes: D11 -->
You probably think that blocking crawlers at your origin keeps your content out
of training sets. That is the entire emotional logic of the 402, and the
dashboards reinforce it: requests blocked, this many, today.

Here is the specific prediction that fails. If origin blocking worked, a
publisher who blocked from day one would find no trace of its content in any
model, and a publisher who blocked late would find only its old content. Neither
holds, because content syndicates. S2 in the opening example was carried by 300
outlets the same day. Wire copies, aggregators, quote-heavy coverage, scraped
mirrors, and the reference site's hundred mirrors are all copies that never touch
your CDN, and a lab that wants the text can take it from any of them. This is the
loudest complaint publishers make in the whole ecosystem, and no product
addresses it.

The belief is appealing because the wall is the one part of this you control and
the one part with a visible metric. Blocking is real leverage - the 50-plus deals
above exist because of it - and leverage feels like protection.

What is actually true: origin blocking is a *negotiating* instrument, not a
protective one. Protection requires knowing where your content's copies live,
which is near-duplicate detection at web scale, plus evidence machinery that
still works when the copy the model saw came from somebody else. Unit v6 builds
the evidence side and unit v7 turns it into something a third party can check. It
is also, per the ecosystem survey, one of the few genuinely open commercial slots
in this space.

```beat
id: v1-b2
type: predict
concept: d-data-market
prompt: |
  A publisher blocked every AI crawler at its CDN on July 1, 2025, the day
  default blocking shipped, and has the dashboards to prove it. Two years
  later a model can discuss its exclusive reporting in detail.

  Before reading on: give two distinct mechanisms by which this happens with no
  crawler ever having been unblocked, and then state what evidence would let
  the publisher tell those mechanisms apart. Be specific about the evidence -
  "check the model" is not an answer.
answer: |
  Mechanisms, any two of: (1) syndication - wire pickups, aggregators, and
  quote-heavy secondary coverage reproduce the substance on domains that were
  never blocked; (2) mirrors and scrapers that copied the content before or
  despite the block and serve it from their own origins; (3) shared corpora
  built before the block, including archived crawls whose removal compliance is
  slow and incomplete; (4) users pasting the article into a chat, which is
  inference-time, not training, but produces the same apparent knowledge;
  (5) a licensed intermediary that itself carries the syndicated copy.

  Telling them apart is a measurement problem, not a modelling one. Useful
  evidence: near-duplicate search across the open web and public crawl archives
  for the distinctive phrasing, dated, to find which copies existed before the
  training cutoff; checking whether the model reproduces details unique to the
  original rendering (a correction only the original carried, a figure caption,
  a paywalled-section-only fact) versus only the details that survived into the
  wire version; and comparing behavior on content published after the model's
  cutoff, which cannot have been trained on at all.
rubric: |
  Must contain: (1) at least two distinct leakage paths, at least one of which
  is syndication or mirroring - the copies-never-touched-your-CDN mechanism;
  (2) at least one concrete piece of evidence tied to dating or to
  original-only detail, not merely "ask the model."
  Both = pass.
  An answer that concludes the block failed technically (bots evaded it,
  residential proxies, user-agent spoofing) and stops there = fail, diagnosing
  D11: evasion is real but it is not the main path, and treating it as the main
  path leads to spending the budget on better blocking.
  An answer that says the model must be reproducing the text verbatim to
  "know" the reporting = flag D3 and do not pass; verbatim is a lower bound on
  what was learned, and unit v6 dismantles it.
check: llm
```

## What the deals actually buy

Roughly 91 content deals have been publicly announced since January 2023: 12 in
2023, 28 in 2024, a dip in 2025, around 36 projected for 2026. The industry rule
of thumb is 50 to 100 private deals for every public one. OpenAI accounts for
about 24 of the public deals. Anthropic accounts for zero.

The structure is almost uniform: lump-sum, multi-year, bilateral. Not per-token,
not per-use, not per-anything. Private training-rights deals across the entire
industry are estimated at $75-100M per year, which is to say the whole
training-data licensing market is smaller than a mid-size Series C.

<!-- refutes: D12 -->
You probably think AI companies buy content mainly to train on it, and therefore
that training rights are where the money is and where a product should point.
This is the received framing, it is the framing every headline uses, and if you
are pitching in this space it is probably the framing in your deck.

Here is the specific prediction it makes, and it fails on the public record.
If training rights were the demand, their share of deals would be stable or
rising as the market matured. Instead: only about 40% of recent deals include
training rights, down from near-universal in 2023-24. Attribution and live-access
deals - the right to fetch your content at inference time and ground an answer in
it - went from 2 in 2023 to 18 in 2025 to roughly 34 projected in 2026, doubling
year over year. The market moved from "buy content to build models" to "license
content to deliver answers."

And the second failing prediction is larger. If content were the scarce input
labs are competing for, content licensing would be where their data budget goes.
It is not close. Meta paid $14.3B for 49% of Scale AI in June 2025. Mercor was
valued at $10B in October 2025, has been in talks at $20B, and booked $614M in
the first half of 2026 - roughly $2B annualized - with about 90% of that revenue
coming from labs. Surge has been raising up to $1B. A single expert-labeled-data
vendor out-earns the entire disclosed content-licensing market. What labs are
short of is not text. It is expert judgment about text.

The belief is appealing because the original scandal was about training, because
"training data licensing" is the phrase everyone uses, and because the earliest
deals really were training deals. It is also appealing because it is the version
of the story in which your attribution technology is the product.

What is actually true, and this is pitch-fatal if you get it wrong in a room with
someone who tracks the deal flow: the paid market is migrating to inference-time
access and to expert data. A training-attribution product must either point at
where money already moves - verification, audit, procurement - or accept that its
first job is creating a market that no buyer has yet joined. Both are defensible
positions. Only one of them is defensible while pretending the other does not
exist.

<!-- refutes: V1-M2 -->
One more reflex to break before you read a deal table again. A headline deal
value is not the price of the content. The $250M News Corp figure buys content,
product placement, an ongoing product relationship, and litigation peace, in
proportions nobody discloses. Analysis across 73 public deals found that the
majority of value accrues to aggregators, with documented creator royalties
rounding to zero. When you see a deal number, ask three questions: what is the
unit (corpus, feed, query, work), who is the counterparty (the creator or an
aggregator holding the creator's rights), and what fraction flows through. The
public answer to the third question is usually "unstated," and the measured
answer is usually "approximately none."

```beat
id: v1-b3
type: self-explain
concept: d-data-market
prompt: |
  You are pitching a training-time attribution product that computes each
  source's share of a royalty pool. Someone across the table who reads the deal
  flow closely asks one question.

  In your own words: which single line of the market data above is the most
  dangerous to your pitch, why it is more dangerous than the others, and what
  the honest response is. Do not defend the product - diagnose it.
answer: |
  The most dangerous line is that only ~40% of recent deals include training
  rights, and that the share is falling while inference-time access deals
  double year over year. It is more dangerous than the small absolute size of
  the market ($75-100M/yr) because size is a "we are early" objection and every
  founder has an answer to it, whereas a falling share is a direction. It says
  the buyers who already pay are moving away from the thing being measured. A
  product priced on training-time contribution is being built for a shrinking
  fraction of a small market.

  The runner-up is the expert-data comparison - one labeling vendor at ~$2B
  annualized against a low-single-digit-billions cumulative content-licensing
  market - because it shows labs will spend enormous sums on data when they
  believe it is scarce, and they do not believe text is scarce.

  The honest response is not to argue the trend will reverse. It is to note
  what the trend does not touch: every player self-reports what it trained on,
  the EU AI Act creates a disclosure obligation with a supervisor and a fine
  attached, and disclosure without verification is an unsupported claim.
  Verification is needed under every outcome, including the outcome where
  nobody ever pays a training royalty. That is the argument that survives the
  question.
rubric: |
  Must contain: (1) identification of the shrinking training-rights share as
  the load-bearing threat, with a reason that distinguishes a trend from a
  size objection; (2) an honest response that does not depend on the trend
  reversing.
  Both = pass. (1) alone = partial, do not pass; the point of the beat is to
  produce a response, not a diagnosis.
  Naming the market's small absolute size as the biggest threat = partial;
  it is a real objection but the weaker one.
  Any answer that responds with "the courts will force royalties" = fail,
  diagnosing D13, which the next section dismantles.
  Any answer that claims RAG citation products validate the market = fail,
  diagnosing D1: those products pay for retrieval placement and are a
  different business.
check: llm
```

## What the courts priced

The single most misread number in this space is $1.5 billion.

In *Bartz v. Anthropic*, Judge Alsup held that training a language model on
lawfully acquired books is "quintessentially transformative" fair use - and that
retaining pirated copies is not. The $1.5B settlement, granted final approval in
July 2026, resolves the second holding, not the first. Spread across roughly
500,000 works, it prices the *acquisition* of an illegitimate copy at about
$3,000 per work, one time. Output-side claims were explicitly preserved and not
resolved.

The rest of the docket points the same direction. In *Kadrey v. Meta*, Judge
Chhabria found training to be fair use on the record before him; interlocutory
appeal was denied in July 2026. *Thomson Reuters v. Ross* went the other way on
summary judgment - a non-generative product competing directly with the source -
and was argued in the Third Circuit in June 2026 with no ruling yet; it is the
most important pending decision. In the UK, *Getty v. Stability* ended in a
near-total defeat for the rightsholder after Getty abandoned its primary
copyright claims mid-trial, with the court endorsing the view that model weights
contain no copies. That last one comes to us second-hand and the details should
be checked against the judgment itself before you repeat them anywhere that
matters.

<!-- refutes: D13 -->
You probably think copyright litigation is going to force AI labs into ongoing
royalty payments, and that the $1.5B settlement is the dam breaking. If you are
building a royalty product, this belief is probably doing structural work in your
business case.

Here is the specific prediction it makes, and it fails. If litigation were
producing a royalty regime, the settlement would have established a rate - per
use, per output, per year, something recurring. It established a one-time price
for an acquisition defect, from a court that had already held that training on
lawfully bought copies is fair use. Now compute the rational response for a lab
that reads that judgment: buy one clean copy of everything, once, at retail. Three
jurisdictions have now converged on some version of the same line.

The belief is appealing because $1.5B is an enormous number and because the
narrative arc of every previous copyright fight - Napster, YouTube, sampling -
ended in licensing regimes. It is also appealing because it converts a market
that does not exist into a market that is merely delayed.

What is actually true: current case law pushes toward one-time clean-copy
purchases and away from ongoing relationships. The theories that could still
force recurring payment are the output-side ones - substitution and market harm -
and they are unresolved. *NYT v. OpenAI* is where they get tested: the court
rejected the claim that training is inherently transformative, leaving fair use
fact-dependent on output substitution and market harm, with discovery through
mid-2026 and trial expected late 2026 or early 2027. That is a real repricing
event, in either direction. Which is exactly why you build for the thing that is
needed under *every* outcome - evidence about what a model was trained on - rather
than the thing that is needed under one.

```beat
id: v1-b4
type: compute
concept: d-data-market
prompt: |
  The Bartz settlement priced acquisition of a work at about $3,000, once.
  Google's Reddit deal runs at about $60M per year.

  At the Bartz price, how many works could a lab acquire outright for the cost
  of one year of the Reddit deal? Give a whole number.
answer: 20000
check: numeric(1)
```

Twenty thousand books a year, bought clean and owned forever, for the price of
one year of access to one forum. That ratio is the argument against the royalty
thesis in one line, and you should be able to produce it from memory.

## Contributive and corroborative

Everything so far has used the word "attribution" for two entirely different
things. Separating them is the single most useful move in this book, and the
field has a name for the split.

**Contributive attribution** identifies the training data that causally shaped a
model's capability or output. **Corroborative attribution** identifies sources
that support a claim the model made. A royalty model logically requires the
first. Every deployed commercial system implements the second, or nothing at all.

Both halves need to be concrete, so here is where each one actually happens in
the machinery, from first principles.

Training is a loop over a fixed corpus. For every position in every document, the
model produces a probability for the token that actually came next, and the
training signal is the average of $-\ln p$ over all those positions, where $p$ is
that probability. A worked number: if the model gives the true next token
probability $0.5$, that position contributes $-\ln 0.5 = 0.69$ nats; if it gives
$0.9$, it contributes $0.105$. Gradient descent nudges the weights to make the
observed tokens more probable. When training finishes, the corpus is *gone*. It
is not stored, referenced, or indexed. What remains is a set of weights that were
pushed around by it, and contributive attribution is the question of how much a
given source did the pushing.

Retrieval happens later and elsewhere. At answer time, a retriever - typically
embedding similarity plus keyword search over an index built independently of the
model - selects a handful of documents, and those documents are pasted into the
prompt as text. The model reads them the way it reads anything else in its
context. The citations you see in the interface are a list of what the retriever
put in the box. Corroborative attribution is the question of which of those
documents supports the sentence.

<!-- refutes: D1 -->
You probably think that when an AI product cites a source, that source is where
the answer came from. Every AI search interface is built to produce exactly this
inference, and the whole "attribution" product category trades on it.

Here is the specific prediction that fails. If citations traced causation, then
two systems trained on disjoint corpora could not emit identical citations for the
same answer - the citations would reflect what each system had learned, and they
learned from different text. But run the experiment the cheap way: hold the model
completely fixed, swap the retrieval index, and re-ask. The citations change
completely. Nothing about the model changed. Nothing it learned changed. Not one
weight moved. The citation was never a fact about the model; it was a fact about a
retriever's ranking, computed seconds before the answer.

Run it in the other direction for the same conclusion: ask a model something it
knows perfectly well with retrieval switched off. It answers, correctly, with no
citations at all. The knowledge and the citation are separable, which they could
not be if the citation were the knowledge's provenance.

The belief is appealing because citations *look* like provenance - they are the
same visual grammar as a footnote, and footnotes in human writing do trace where
the writer got it. It is appealing because in a pure RAG system with an empty
model the citation would be nearly causal for that answer. And it is appealing
because every vendor in the category encourages it.

Here is what is actually true, and it is worth stating as a rule you can apply
mechanically. Citation is corroborative: this source supports this claim.
Attribution is contributive: this data caused this capability. A compensation
scheme built on citations pays for retrieval placement, and retrieval placement is
purchasable, tunable, and gameable in ways that have nothing to do with having
written anything. When ProRata splits ad revenue pro-rata across cited sources,
when Perplexity pays on visits and citations and agent actions, when Microsoft's
Publisher Content Marketplace pays when Copilot grounds an answer in your page -
all of them, honestly and by design, are paying for placement in a context window.
That is a real product and a real business. It is not a training royalty, and
calling it attribution in a pitch will lose you the room if the person across the
table knows the difference.

```beat
id: v1-b5
type: self-explain
concept: d-contributive-corroborative
prompt: |
  A vendor demos a system that shows, next to every generated sentence, the
  documents that "the model used" - complete with percentage weights that sum
  to 100%.

  In your own words: state the one experiment that determines whether those
  percentages are contributive or corroborative, what result each answer
  predicts, and why the vendor's percentages summing to 100% is itself
  informative.
answer: |
  The experiment: hold the model's weights fixed and change the retrieval index
  - remove a cited document, add a new one that says the same thing, or swap
  the retriever's ranking function. Re-ask the identical question.

  If the percentages are corroborative, they change, because they were computed
  over the retrieved set. If they were contributive - a claim about what
  training data shaped the weights - they cannot change, because nothing about
  the weights changed. A contributive number can only move if you retrain.

  A second, cheaper version: turn retrieval off entirely. If the system still
  answers but shows no percentages, the percentages were about retrieval.

  The 100% sum is informative because it says the quantity is a share of a
  fixed set - a normalized allocation over whatever happened to be in the
  context window. Contributive attribution over a full training corpus does not
  naturally normalize over three documents; it is defined against millions, and
  the measured contributions do not conveniently sum to a clean total. A tidy
  three-way split summing to 100% is a strong signal that the denominator is
  the retrieved set.
rubric: |
  Must contain: (1) the swap-the-index-hold-the-weights-fixed experiment, or
  the retrieval-off variant; (2) the correct prediction for each case -
  corroborative numbers move, contributive numbers cannot without retraining;
  (3) some recognition that normalizing over a handful of documents implies the
  denominator is the retrieved set, not the corpus.
  (1) and (2) = pass. (3) upgrades to full credit.
  An answer that proposes inspecting attention weights over the retrieved
  documents to decide = fail: that measures how the model used its context,
  which is still corroborative, and does not touch training data at all.
  An answer that accepts the percentages as contributive if the vendor says
  they are computed with gradients = fail, diagnosing D1; the test is the
  experiment, not the vendor's method name.
check: llm
```

## Three ways to pay, and what each one misallocates

Now assemble the comparison as a tool you can run on anything. Three families,
and what each is proportional to:

| Scheme | Proportional to | Fails whom |
| --- | --- | --- |
| Flat fee / lump sum | leverage, or enrollment | the long tail, always |
| Citation counting | retrieval placement | anyone who wrote but does not rank |
| Causal attribution | measured marginal effect | anyone whose content was copied |

The production reality, as of mid-2026, is that no deployed system documents an
attribution algorithm. The allocation rules actually in use are: flat pro-rata
(Shutterstock, median $0.0069 per image), discretionary bonus (Adobe Firefly, one
contributor reported $4.88), relevance-proportional but patented and undisclosed
(Bria), honest unit-price metering (TollBit, Cloudflare), and undefined (ProRata,
Sureel). The industry converged on flat fees, and unit v5 shows there is a
mathematical reason for that convergence rather than mere laziness.

<!-- refutes: D10 -->
You probably think pro-rata pooling - the Spotify model - pays providers in
proportion to how much they were used. It is the incumbent template, "share of
streams" sounds exactly like "share of use," and it is almost certainly the model
in your deck, because it is the model in everyone's deck. RSL has already hired a
pricing economist to build precisely this.

Here is the specific prediction it makes, and it fails at the first arithmetic
step. Under pro-rata, your subscription does not go to the artists you played. It
goes into a pool, and the pool is divided by *platform-wide* share. If you spend a
month listening to one obscure musician, your money is distributed across the
platform's chart in proportion to everyone else's listening, and your musician
receives a fraction of a cent. Money flows from your pocket to artists you have
never played. That is not a bug in the implementation; it is what the formula
says. Look again at $s_i = w_i / \sum_j w_j$ from the first section: nobody's
payout is a function of their own usage. Every payout is a function of the ratio,
and the denominator belongs to everyone.

Three consequences follow directly, and all three transfer to any AI version:

**Cross-subsidy.** Payment is decoupled from the payer's actual consumption. In
an AI marketplace this means a lab that grounds its answers exclusively in
technical sources still pays a share of its pool to celebrity gossip, if gossip
dominates the platform-wide numerator.

**Winner-take-most concentration.** In music, 99.8% of artists receive
effectively nothing. The distribution of citations, or of measured contribution,
or of anything else you might use as $w_i$ is at least as skewed as the
distribution of streams. A pool divided by a power-law weight pays a power law.

**Farming.** If payout tracks a countable event, manufacturing the event is a
business. Stream farms exist despite needing plausible listening behavior at
scale. The AI analogue - citation farms - is *cheaper*, because plausible citable
text costs essentially nothing to generate and the "listener" is a retriever that
can be reverse-engineered and optimized against. Whatever a scheme counts,
somebody will manufacture.

The proposed fix does not fix it. User-centric allocation, where each user's fee
is divided only among the sources that user actually consumed, is the obvious
repair and it has been modeled: it shifts only 1 to 5 percentage points toward
independents, and the majors prefer pro-rata anyway.

What is actually true: pro-rata pooling imports cross-subsidy, winner-take-most
concentration, and gaming incentives, and any pooled design has to answer all
three by construction rather than by promising to monitor for abuse. Saying
"Spotify for AI training data" in a pitch is not a differentiated insight - it is
the incumbent standards body's stated design, complete with the economist they
already hired, and the pathologies come attached.

```beat
id: v1-b6
type: completion
concept: d-compensation-schemes
# variants: blank the flat-fee row and one causal row instead; or supply the
# payouts and blank the weights.
prompt: |
  Same pool split procedure as before: $s_i = w_i / \sum_j w_j$, payout is
  $s_i \times P$. New month, same three sources, pool $P = \$20{,}000$.

  Measured leave-one-out contributions, in nats of held-out loss:
  S1 = 0.05, S2 = 0.15, S3 = 0.30.

  Step 1. Total weight:        0.05 + 0.15 + 0.30 = ____
  Step 2. S2's share:          0.15 / ____ = 0.30
  Step 3. S2's payout:         0.30 x $20,000 = ____
  Step 4. S1's payout:         (0.05 / 0.50) x $20,000 = ____
  Step 5. Flat-fee payout to each source, same pool: ____

  Fill the five blanks. Then answer in one sentence: the seed-noise standard
  deviation on each contribution measurement is 0.06 nats. What does that do to
  your confidence in the difference between S1's payout and S2's payout?
answer: |
  Step 1: 0.50
  Step 2: 0.50
  Step 3: $6,000
  Step 4: $2,000
  Step 5: $20,000 / 3 = $6,666.67 each

  The noise sentence: with a seed-noise SD of 0.06 nats, S1 (0.05) and S2
  (0.15) are about 0.10 nats apart, well inside two standard deviations of the
  measurement, so the ordering itself is not established - rerun with different
  seeds and S1 could measure above S2. The 3x payout difference between them is
  not supported by the measurement. Only S3 is clearly separated from the
  others, and even that separation is a few sigma at best.
rubric: |
  Required, exactly: 0.50; 0.50; $6,000; $2,000; $6,666.67 (accept $6,667 or
  "one third of the pool").
  The noise sentence must state that the S1-S2 gap is within noise, so the
  ranking and therefore the payout ratio is not established.
  All five numbers plus the noise point = pass.
  Numbers right, noise point missing or hand-waved as "add error bars later"
  = fail, diagnosing D20: the noise is not a caveat on the result, it is the
  result, and unit v5 shows that below a threshold the optimal contract is the
  flat fee in step 5.
check: llm
```

```beat
id: v1-b7
type: predict
concept: d-compensation-schemes
prompt: |
  A marketplace pays a $42.5M annual pool to publishers pro-rata by citations.
  You are an adversary with a budget of $50,000 and no content anyone wants.

  Before reading on: describe the attack, then state which of the three
  schemes - flat fee, citation counting, causal attribution - is hardest to
  attack this way and why. Name what each scheme forces the attacker to
  actually produce.
answer: |
  The attack: generate large volumes of plausible, factually unobjectionable
  text on high-query-volume topics, host it across many domains, and optimize
  it for the retriever rather than for readers - the embedding model and the
  keyword index are the audience, and both can be probed cheaply by issuing
  queries and observing what gets cited. Enroll all the domains. Each is a
  small share; the aggregate is not. Text generation is the cheap part; the
  spend goes on domains, hosting, and enough surface legitimacy to pass
  enrollment.

  Hardest to attack is causal attribution, because the attacker must produce
  content that measurably improves a model that already has the entire web.
  Redundant plausible text has near-zero marginal contribution by construction:
  if the model can already predict it, removing it changes nothing. The
  attacker must produce genuinely new information, which is the thing the
  scheme was trying to buy.

  What each forces the attacker to produce: flat fee forces enrollment - so the
  attack is on identity, incorporating many entities rather than one, which is
  the shell-company attack of unit v5. Citation counting forces retrieval
  placement, which is SEO with a machine reader and costs the price of hosting.
  Causal attribution forces non-redundant information, which cannot be
  synthesized from what the model already knows.
rubric: |
  Must contain: (1) a farming attack aimed at the retriever, with the
  recognition that plausible text is nearly free; (2) causal attribution named
  as hardest, with the redundancy argument - content the model can already
  predict has no marginal contribution; (3) at least one correct statement of
  what a different scheme forces the attacker to produce.
  (1) and (2) = pass.
  An answer that names flat fee as hardest to attack because there is nothing
  to count = partial, do not pass: flat fee moves the attack to identity and
  enrollment rather than eliminating it, and unit v5 shows source-splitting
  breaks group Shapley too.
  An answer claiming detection and moderation solve this = fail; the design
  question is what the rule is proportional to, and any countable proxy invites
  manufacture.
check: llm
```

```beat
id: v1-b8
type: compute
concept: d-compensation-schemes
prompt: |
  Concentration, with numbers. A $42.5M annual pool is split pro-rata among
  1,000 enrolled publications. The citation distribution is power-law shaped:
  the top 1% of publications take 80% of all citations, and assume the
  remaining 990 publications split the rest equally.

  What does one of those 990 publications earn per year, in dollars?

  Give the answer to the nearest dollar.
answer: 8586
check: numeric(1)
```

Eight and a half thousand dollars a year, from the largest publisher pool anyone
has actually funded, under assumptions that are *generous* about the tail - real
power laws do not split the remainder equally, they keep decaying. TollBit
reports roughly 7,000 publisher sites with about 20% earning any revenue at all,
in the range of hundreds to tens of thousands of dollars per month. That is the
shape of every pool: a few real cheques, and a long tail for whom the
administrative cost of participating exceeds the payout.

## The strongest case against all of this

The addendum for this volume requires each of the load-bearing units to state the
strongest argument against its own subject. Here is this unit's, made as strong as
it can be made, without hedging.

1. **No buyer has ever paid into a neutral marketplace.** Not one. RSL has
   roughly 1,500 endorsing organizations and no confirmed licensee. AIPREF has a
   published vocabulary and no production reader at any major lab. Pay Per Crawl
   never left private beta. Every real cheque in the table at the top of this unit
   was a bilateral negotiation between two parties with lawyers.
2. **Case law removed the forcing function.** Acquire your copies legitimately
   and you may train. The one enormous settlement priced a one-time acquisition
   defect.
3. **The training-data share of deals is shrinking**, from near-universal to
   about 40%, while access deals double annually.
4. **Training-time attribution is not technically credible at frontier scale**
   as a basis for payment - unit v8 shows the methods that scale do not
   demonstrably correlate with ground truth - and RAG-time attribution is already
   commoditized by ProRata, TollBit, Microsoft's PCM and Perplexity.
5. **Blocking leaks**, via syndication, mirrors, and copies that never touch your
   origin.
6. **Bilateral lump-sum deals work fine for parties with leverage**, and the long
   tail sees no revenue under any mechanism, marketplace or not.
7. **The gatekeeper is vertically integrated** across chokepoint, identity,
   preference, payment, index and marketplace, and publishers already call it
   vendor-locked.
8. **Synthetic data caps willingness to pay.** Labs that can generate training
   text spend their scarcity budget on expert labeling instead, which is exactly
   what the Scale and Mercor numbers show.

That is a serious case. Take it seriously: an honest reading is that
"marketplace plus per-output attribution plus revenue share" is a pitch VCs have
already heard many times, and that the attribution startups in this space get
acquired by rightsholders or gatekeepers rather than growing into independent
royalty exchanges. Sureel went to Warner Music in June 2026. Vermillio took money
co-led by Sony Music. Human Native went to Cloudflare in January 2026.

Now what survives it.

Every one of the eight points is an argument about *payment*. Not one of them is
an argument about *measurement*. And measurement has a buyer already, for reasons
that hold under every branch of the litigation tree:

- Every player in this ecosystem self-reports what it trained on. Self-reporting
  without verification is an unsupported claim, and the EU AI Act's Article
  53(1)(d) turns it into a mandatory structured summary by source category, with
  a supervisor that may verify and fine from August 2, 2026. Note the shape of
  the obligation precisely, because it constrains what an audit product can be:
  the Commission's own explanatory notice says supervision happens without a
  work-by-work assessment. Disclosure-shaped, not payment-shaped.
- Leakage and syndication tracking is the loudest unaddressed publisher
  complaint, and it is tractable with near-duplicate detection at web scale.
- Buy-side procurement - provenance-verified sourcing, rights clearance,
  indemnification - points at where labs demonstrably spend billions.

All three are verification problems. All three need the same machinery: a way to
say, with a stated error rate, what a model was trained on. That machinery is
units v6 and v7, and it is also, not incidentally, the only thing that makes a
royalty split auditable if a royalty market ever does appear. Build the
measurement; the payment layer is a bet, and the measurement layer is not.

The correlation problem in the unit's title is now statable in one sentence. The
quantity we can measure cheaply (citations, fetches, verbatim overlap) is not the
quantity we want to pay for (contribution), and the quantity we want to pay for is
measurable only at small scale, noisily, and with a seed-dependence that can
exceed the signal. The rest of this book is the attempt to measure it anyway, on a
corpus small enough that the truth is computable, and to report the correlation
honestly whichever way it comes out.

## At the bench: what counts as one source

Every unit from here builds one piece of a proof of concept: a small model
trained on a corpus you control, with per-source contributions measured against
ground truth, and a verification layer that turns a number into evidence. This
unit's build step is the one that determines whether any of the later steps mean
anything, and it involves no code at all.

**What to build.** A one-page source specification for the PoC corpus, drawn from
an open papers corpus (peS2o, the S2ORC-derived open-access paper collection), and
a choice of 10 to 12 sources. The artifact is a written spec plus a
`sources.yaml`. Everything downstream reads it: unit v2's pipeline tags every
document with a source ID, v4's leave-one-source-out table has one row per source,
v5 enumerates coalitions over exactly these sources, and v7's watermarks are
planted in one of them.

**Why 10 to 12 and not more.** Unit v5 enumerates every coalition of sources
exactly, and the number of coalitions is $2^n$ for $n$ sources: 1,024 at $n = 10$,
4,096 at $n = 12$, 32,768 at $n = 15$. Each coalition needs a training run.
Twelve is affordable on a laptop overnight; fifteen is not. The cliff is at about
15, not at a million, which is why source-level attribution is tractable when
per-document attribution is not.

**What a source is.** The definition that makes the whole system coherent: *a
source is a payment counterparty* - an entity that could sign a contract and
receive money. Not a file, not a shard, not a paper. If you cannot name who would
be paid, it is not a source. Two constraints follow immediately. The identity must
be stable across pipeline reruns, or contributions computed in different weeks are
not comparable. And the assignment must happen *before* deduplication, because
when two sources contain the same passage and the pipeline keeps one copy,
whichever copy survives silently receives credit for everything that passage
teaches the model. That is a payout decision made by pipeline ordering; unit v2
makes it an explicit policy.

**What the manifest must record.** For every document that enters training, at
minimum: a stable `source_id`; a document identifier that survives re-extraction;
the shard and byte offset where the tokens landed; the token count; the license
and its provenance; the duplicate-cluster ID and the full set of sources in that
cluster, not just the surviving one; the extraction and filtering code version;
and the timestamp of ingestion. The token count is what makes any per-token
normalization possible later. The duplicate-cluster membership is what makes the
credit policy auditable. The code version is what makes a contribution number from
March comparable to one from June.

```beat
id: v1-b9
type: predict
concept: d-data-market
prompt: |
  Two candidate source definitions for the PoC corpus:

  (a) Per paper: each paper in the corpus is its own source. n is about
      100,000.
  (b) Per venue: each journal or conference is a source. n is about 11.

  Before reading on: name the fatal problem with (a) that is not about compute
  cost, and name the two distinct measurement problems that (b) creates even
  though its arithmetic is easy.
answer: |
  (a)'s fatal problem is not the 2^100000 coalitions - it is that a paper is
  not a payment counterparty. Nobody can sign a contract as "paper #40118."
  Payouts at that granularity are dust, the administrative cost of paying
  exceeds the payment, and the entity that would actually be paid (the
  publisher, the funder, the authors' institution) is a different object that
  the scheme never names. Choosing the attribution unit to be different from
  the payment unit means the output of the whole system has to be re-aggregated
  by a rule nobody has justified.

  (b) creates two distinct problems. First, confounding: removing an entire
  venue removes a topic. A leave-one-source-out measurement then reports how
  much the model needed that subject area, not how much it needed that
  publisher's version of it. The measurement answers a different question than
  the one being billed.

  Second, overlap and containment: preprints and published versions of the same
  paper live in different venues, so venues are not disjoint. The same content
  is in two "sources," each of which measures as low-contribution because the
  other still covers it, and the duplicate-cluster policy - not the model -
  decides who gets paid.
rubric: |
  Must contain: (1) for (a), that a paper is not a payment counterparty / the
  attribution unit must match the payment unit, NOT merely that 2^n is too big;
  (2) for (b), the topic-confounding problem - removing a venue removes a
  subject area, so the measurement is about the domain rather than the source;
  (3) for (b), the overlap problem - preprint/published duplication means
  sources are not disjoint and the dedup policy decides credit.
  (1) plus either (2) or (3) = pass. All three = full credit.
  An answer whose only objection to (a) is computational cost = fail: the
  compute objection is answered by sampling, and it hides the real defect.
  An answer that proposes dropping duplicates and moving on = flag D9 and do
  not pass; which copy survives is a payout decision, not hygiene.
check: llm
```

```beat
id: v1-b10
type: self-explain
concept: d-contributive-corroborative
prompt: |
  Six months from now, a rightsholder disputes your PoC's payout: "your model
  says my catalogue is worth 3.2% of the pool. Prove it."

  In your own words, list the fields the manifest must have recorded at
  ingestion time for you to be able to answer that at all, and for each field
  say what specific question it answers. Then name one thing you cannot answer
  no matter how good the manifest is.
answer: |
  Fields and the question each answers:

  - source_id, stable across reruns: which rows of the payout table are yours,
    and are this month's numbers comparable to last month's.
  - document IDs and token counts: what exactly was included and how much of
    it, so the disputed 3.2% can be tied to a specific set of text rather than
    a name.
  - shard and byte offsets: that the tokens the model actually consumed are the
    ones claimed - the difference between a manifest and an assertion.
  - duplicate-cluster ID plus full cluster membership: whether the passages you
    are being paid for also appeared in other sources, and which policy
    assigned the credit. Without this the biggest single objection - "you paid
    them for my text" - cannot be answered.
  - license and its provenance: whether the content was eligible to be in the
    corpus at all, which is a separate question from what it was worth.
  - extraction and filter code version: whether a changed number reflects a
    changed corpus or a changed pipeline.
  - ingestion timestamp: what the corpus looked like at the time of the run
    being disputed.

  What the manifest cannot answer, no matter how complete: whether the measured
  3.2% is stable. That is a property of the training process, not of the
  corpus. Rerunning with a different seed moves contributions, and only
  repeated runs with a reported noise floor can say whether 3.2% is
  distinguishable from 2.1%. The manifest establishes what went in; it cannot
  establish that the number coming out is reproducible.
rubric: |
  Must contain: (1) at least four fields with the specific question each
  answers, including duplicate-cluster membership and token counts;
  (2) the recognition that the manifest cannot establish stability of the
  measured contribution, which requires seeds and a noise floor.
  Both = pass. (2) missing = fail, diagnosing D16: a manifest is provenance,
  not evidence about the estimate.
  Listing fields with no question attached to each = partial, do not pass; the
  beat tests whether the learner can say what each record is for.
  Naming citation counts or retrieval logs among the required fields = fail,
  diagnosing D1: those are corroborative records and have no bearing on a
  training-contribution dispute.
check: llm
```

The next unit takes this spec and builds the corpus underneath it: extraction,
quality filtering, deduplication with the cluster records this section demands,
and a tokenizer trained on the result. The spec you write here is what makes that
pipeline's output billable rather than merely tidy.

## Vocabulary in this unit

<!-- canon-only -->

Reference, not reading. Return here when a term goes blurry; nothing below is
new.

| Term | Means |
| --- | --- |
| Contributive attribution | Identifying training data by its causal effect on a model's capability or output. What a royalty logically requires. |
| Corroborative attribution | Identifying sources that support a claim the model made. What every deployed system does. |
| Training rights | The right to include content in a training corpus. Present in about 40% of recent deals. |
| Access / inference-time rights | The right to fetch content at answer time and ground an output in it. The growing category. |
| Pro-rata pooling | Fixed pot, split by each participant's share of a countable weight. The Spotify model. |
| Pay Per Crawl / Pay Per Use | The gatekeeper's HTTP 402 payment schemes: flat price per fetch, then per value created. |
| Web Bot Auth | Ed25519-signed crawler requests per RFC 9421. Crawler identity, solved and widely enforced. |
| Content Signals | `robots.txt` directives splitting usage into `search`, `ai-input`, `ai-train`. |
| RSL | Really Simple Licensing: `robots.txt`-embedded license terms. Broad supply-side adoption, no confirmed licensee. |
| AIPREF | The IETF working group standardizing preference vocabulary and the `Content-Usage` header. |
| Article 53(1)(d) | EU AI Act obligation: a structured public summary of training content by source category, supervised from August 2, 2026. |
| Source | In this book: a payment counterparty. The unit at which contribution is measured and money is paid. |
| Manifest | The per-document record tying tokens in a shard back to a source, a license, and a duplicate cluster. |
| Nat | Unit of the training loss: the average of $-\ln p$ over positions, where $p$ is the probability the model gave the token that actually occurred. |
