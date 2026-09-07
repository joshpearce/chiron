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
  Commit before reading on. Citations that month: S2 6,000, S3 3,600, S1 400,
  out of 10,000. The pool is $12,000.

  Which statement gets all three schemes right - who wins, and what quantity
  each scheme is actually proportional to - and answers the question of whether
  any of them pays S1 for having originated the analysis?
options:
  - text: |
      Flat fee gives each source $4,000 and is proportional to enrollment;
      citation counting pays S2 most and is proportional to retrieval
      placement; causal attribution pays S3 most and is proportional to
      measured marginal effect on held-out loss. None of the three pays S1 for
      originating: causal attribution in fact punishes it, because the copies
      in S2 and S3 make its removal cost the model almost nothing.
    correct: true
    explain: |
      Right, and the last clause is the load-bearing one. Origination is not a
      quantity any of these rules measures. Flat fee pays for existing,
      citations pay for a retriever's ranking, and counterfactual contribution
      is near zero exactly when your content has been duplicated elsewhere.
  - text: |
      Citation counting is the scheme that tracks real use, so it is the fair
      one: S2 and S3 earn their shares because the engine genuinely drew on
      them, and S1's $480 correctly reflects that its analysis reached readers
      only through others.
    misconception: D1
    explain: |
      Citations are corroborative, not contributive. Hold the model fixed and
      swap the retrieval index and every citation changes while nothing the
      model learned moves. That share is a fact about the retriever's index,
      embedding model, recency weighting and your SEO - not about use of what
      you wrote.
  - text: |
      Causal attribution is the one scheme that rewards S1, because
      leave-one-out training would show that removing the origin of the
      analysis degrades the model on this domain - the copies exist only
      because S1 wrote it first.
    misconception: D5
    explain: |
      Counterfactual measurement asks what changes when the source is removed,
      not who wrote it first. With S2's wire copy and S3's encyclopedic entry
      still in the corpus, the model learns the same content without S1, so
      S1's measured drop is near zero. Being copied lowers your measured
      contribution.
  - text: |
      All three schemes are proportional to volume of content supplied, so they
      differ only in the accounting overhead; S1 loses under each simply
      because 400 subscribers is a small business.
    misconception: D10
    explain: |
      They are proportional to three different quantities - enrollment,
      retrieval placement, and marginal effect on the model - and those
      quantities produce three different winners on the same month's data. The
      pool formula $s_i = w_i / \sum_j w_j$ is shared; the choice of $w_i$ is
      the whole design.
check: choice
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

  Commit before reading on: which account best explains how this happened with
  no crawler ever having been unblocked, and names evidence that could
  distinguish the mechanisms?
options:
  - text: |
      The content syndicated: wire pickups and quote-heavy secondary coverage
      carried the substance on domains that were never blocked, and mirrors and
      scrapers serve their own copies from their own origins. To tell those
      apart, run dated near-duplicate search over the open web and public crawl
      archives to see which copies predate the training cutoff, and check
      whether the model reproduces details that existed only in the original
      rendering rather than only what survived into the wire version.
    correct: true
    explain: |
      Yes - the copies never touched the CDN, so the wall could not see them.
      And the discriminating evidence is dating plus original-only detail:
      both are measurements over copies and outputs, not questions put to the
      model.
  - text: |
      The block failed technically: crawlers rotated through residential
      proxies and spoofed user agents, so the dashboard counted the requests it
      recognized and missed the rest. The evidence is server log analysis for
      anomalous fetch patterns from unsignatured clients.
    misconception: D11
    explain: |
      Evasion is real but it is not the main path, and treating it as the main
      path spends the budget on better blocking. Web Bot Auth means the major
      crawlers identify themselves; the leak is that a hundred lawful copies of
      your text live on domains you do not control.
  - text: |
      Nothing is established yet, because a model discussing the reporting is
      not evidence of training on it. Unless the model emits the article
      verbatim, it did not learn from that text, and the only useful evidence
      is prompting for exact reproduction of distinctive sentences.
    misconception: D3
    explain: |
      Verbatim emission is a lower bound on what was learned, not the test.
      Ask for the same content in a different style and the knowledge is
      plainly intact with no n-gram overlap; most of what training data
      contributes is semantic and never resurfaces word for word.
  - text: |
      The publisher's own licensing counterparty passed the archive on, so the
      leak is contractual rather than technical, and the evidence is an audit of
      the deal's downstream-sharing clauses.
    misconception: V1-M2
    explain: |
      A licensed intermediary is one possible path, but it treats a deal
      document as the explanation for a web-scale phenomenon. Syndication and
      mirroring reproduce the substance across hundreds of unblocked domains
      without any contract at all, which is why the fix is near-duplicate
      detection rather than clause review.
check: choice
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

  Which diagnosis of your own pitch is the honest one - naming the most
  dangerous line in the market data, why it is more dangerous than the others,
  and a response that does not depend on wishful thinking?
options:
  - text: |
      The dangerous line is that only about 40% of recent deals include
      training rights, down from near-universal, while inference-time access
      deals double year over year. It is worse than the market's small absolute
      size ($75-100M/yr) because size is a "we are early" objection and a
      falling share is a direction: the buyers who already pay are moving away
      from the thing being measured. The response is not that the trend
      reverses, but that what the trend does not touch is verification - every
      player self-reports what it trained on, and Article 53(1)(d) attaches a
      supervisor and a fine to that self-report from August 2, 2026.
    correct: true
    explain: |
      Right on both halves. A shrinking share is a statement about direction
      that no amount of market growth answers, and the surviving argument is
      the one that holds under every branch: disclosure without verification is
      an unsupported claim, needed even if no royalty is ever paid.
  - text: |
      The dangerous line is the litigation record, and it is also the answer:
      the $1.5B Bartz settlement shows the dam breaking, so ongoing royalties
      are coming and the product is early rather than mispointed.
    misconception: D13
    explain: |
      That settlement priced the acquisition of pirated copies at about $3,000
      per work, once, from a court that had already held training on lawfully
      acquired copies to be fair use. The rational lab response is to buy one
      clean copy, not to enter a royalty relationship - the case law removed the
      forcing function rather than supplying it.
  - text: |
      The dangerous line is the small absolute market size, and the answer is
      that RAG citation products - ProRata, TollBit, Perplexity's pool - are
      already paying out per source, which validates that buyers will pay for
      attribution.
    misconception: D1
    explain: |
      Those products pay for retrieval placement in a context window, which is
      corroborative and a different business from training contribution. Citing
      them as validation invites the reply that your quantity is the one nobody
      is buying. Size is also the weaker objection: it is answered by "we are
      early," whereas a falling share is not.
  - text: |
      The dangerous line is that no lab participates in Pay Per Crawl or RSL,
      and the answer is the posted rate card: roughly 1,500 organizations
      endorse RSL and prices are published, so the market's clearing price is
      already visible and the pool is sizable.
    misconception: V1-M1
    explain: |
      Posted prices are asks with no recorded bid. Pay Per Crawl never left
      private beta and there is no confirmation any AI company has paid under
      RSL terms; quoting those rates as market size is the fastest way to lose
      someone who tracks the demand side.
check: choice
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

  Which explanation correctly states the experiment that settles whether those
  percentages are contributive or corroborative, what each answer predicts, and
  what the 100% sum tells you on its own?
options:
  - text: |
      Hold the weights completely fixed and change the retrieval index - drop a
      cited document, add one saying the same thing, or swap the ranking
      function - then re-ask the identical question. Corroborative percentages
      move, because they were computed over the retrieved set; contributive
      ones cannot move, because a claim about training data can only change if
      you retrain. The clean sum to 100% is itself a tell: it means the
      denominator is a handful of retrieved documents, whereas contribution
      over a full corpus is defined against millions and does not normalize
      tidily over three.
    correct: true
    explain: |
      Exactly. The experiment separates a fact about a retriever's ranking,
      computed seconds before the answer, from a fact about what shaped the
      weights. The cheaper variant is switching retrieval off: the model still
      answers, and the percentages vanish.
  - text: |
      Inspect the attention weights the model places on each retrieved document
      while generating the sentence. High attention mass on a document means
      that document causally drove the output, so if the percentages track
      attention they are contributive.
    misconception: D1
    explain: |
      Attention over the context window measures how the model used text placed
      in front of it, which is still corroborative and says nothing about
      training data. The corpus is gone once training finishes; no reading of
      the forward pass over retrieved documents reaches it.
  - text: |
      Ask the vendor how the numbers are computed. If they are gradient-based
      influence scores rather than retrieval similarity scores, they are
      contributive by construction, and no further experiment is needed.
    misconception: D6
    explain: |
      The method name is not the test - the experiment is. And gradient
      influence at scale is exactly where the credibility problem lives:
      measured against retraining ground truth, the scalable variants have
      performed no better than random guessing, so a label of "gradient-based"
      buys nothing without a reported correlation.
  - text: |
      Check whether the cited documents contain the sentence's claim, phrase by
      phrase. If each percentage is proportional to how much of the claim that
      document entails, the split is contributive; if not, it is arbitrary.
    misconception: D5
    explain: |
      Entailment and influence are different targets: what contains a fact is
      not what moved the loss, and the two rankings systematically disagree.
      Verifying entailment over retrieved documents confirms corroboration, and
      confirms it well - it just never touches training data.
check: choice
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

  The seed-noise standard deviation on each contribution measurement is 0.06
  nats. Which filling of the five blanks, together with its reading of that
  noise figure, is correct?
options:
  - text: |
      0.50; 0.50; $6,000; $2,000; $6,666.67 each. With a seed-noise SD of 0.06
      nats, the S1-S2 gap of 0.10 nats sits inside two standard deviations, so
      the ordering itself is not established - rerun with different seeds and
      S1 could measure above S2. The 3x payout difference between them is not
      supported by the measurement.
    correct: true
    explain: |
      Correct, and the noise reading is the point of the beat. Only S3 is
      clearly separated, and even that is a few sigma at best; unit v5 shows
      that below a signal-to-noise threshold the welfare-optimal contract is
      the flat fee in step 5.
  - text: |
      0.50; 0.50; $6,000; $2,000; $6,666.67 each. The arithmetic is exact and
      the split stands as computed; the 0.06 nats of seed noise is a caveat to
      note in a methods appendix and to shrink later with better estimators or
      more training runs.
    misconception: D20
    explain: |
      The numbers are right and the reading is not. The noise is not a caveat
      on the result - it is the result. There is a proven threshold below which
      no better estimator changes the optimal contract, only more signal does,
      and a 0.10-nat gap against a 0.06-nat SD is on the wrong side of it.
  - text: |
      0.50; 0.50; $6,000; $2,000; $6,666.67 each. Averaging over seeds recovers
      the true contribution for each source, so the 0.06 nats affects only how
      many runs are needed - once averaged, S1's 0.05 and S2's 0.15 are stable
      properties of those sources and can be billed on indefinitely.
    misconception: D16
    explain: |
      Averaging narrows the error on one corpus and model, but contribution is
      not a stable property to measure once and bill forever: it moves with
      seed, with data order, and with the rest of the corpus. A billing-grade
      number needs a reported noise floor attached each time, not a one-off
      calibration.
  - text: |
      0.50; 0.15; $6,000; $2,000; $6,666.67 each. The noise figure of 0.06 nats
      is smaller than every measured contribution, so all three measurements
      clear it and the ranking S3 > S2 > S1 is established.
    misconception: D20
    explain: |
      Step 2's blank is the denominator, the total weight 0.50, not S2's own
      weight. And comparing the SD to each contribution separately is the wrong
      comparison: what must clear the noise is the gap between two sources, and
      S1 and S2 are 0.10 nats apart against a 0.06-nat SD.
check: choice
```

```beat
id: v1-b7
type: predict
concept: d-compensation-schemes
prompt: |
  A marketplace pays a $42.5M annual pool to publishers pro-rata by citations.
  You are an adversary with a budget of $50,000 and no content anyone wants.

  Commit before reading on: which account describes the attack, names the
  scheme hardest to attack this way with the right reason, and says what each
  scheme forces the attacker to actually produce?
options:
  - text: |
      Generate large volumes of plausible, unobjectionable text on
      high-query-volume topics, spread it across many enrolled domains, and
      tune it for the retriever - the embedding model and keyword index are the
      audience, and both can be probed cheaply by issuing queries and watching
      what gets cited. Causal attribution is hardest, because redundant text
      has near-zero marginal contribution by construction: if the model can
      already predict it, removing it changes nothing. Flat fee forces
      enrollment (so the attack becomes incorporating many entities), citation
      counting forces retrieval placement (SEO with a machine reader), and
      causal attribution forces genuinely non-redundant information.
    correct: true
    explain: |
      Right. Text is the cheap part; the spend goes on domains, hosting and
      enough surface legitimacy to pass enrollment. And the reason causal
      attribution resists is that it demands the one thing that cannot be
      synthesized from what the model already knows.
  - text: |
      Flat fee is hardest to attack, because there is no countable weight to
      manufacture: everyone enrolled gets the same cheque, so generating text
      buys the attacker nothing and the pool is safe.
    misconception: D7
    explain: |
      Flat fee does not eliminate the attack, it relocates it onto identity:
      incorporate 200 entities and collect 200 equal shares. Unit v5 shows the
      same source-splitting attack breaks group Shapley, which is why the
      enrollment boundary needs defending however the pool is divided.
  - text: |
      Citation counting is as robust as causal attribution here, because
      manufactured text still has to be retrieved and cited for real user
      queries, which means it genuinely served readers; the marketplace can
      screen the rest with content moderation and fraud detection.
    misconception: D10
    explain: |
      Being retrieved is not serving readers - the listener is a retriever that
      can be reverse-engineered and optimized against, and plausible citable
      text costs essentially nothing to generate, which makes citation farms
      cheaper than stream farms. The design question is what the rule is
      proportional to; any countable proxy invites manufacture, and moderation
      is a patch on a rule that rewards the wrong quantity.
  - text: |
      Causal attribution is the easiest to attack, because the attacker can
      simply flood the corpus with duplicates of high-value content: the more
      copies of a passage the corpus holds, the larger the measured
      leave-one-out contribution of whoever supplied them.
    misconception: D5
    explain: |
      Duplication cuts the other way under a counterfactual measurement.
      Removing one copy of a passage that exists elsewhere changes the model
      almost not at all, so the measured contribution of redundant content is
      near zero - which is precisely why the original source in this unit's
      opening example loses under causal attribution.
check: choice
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

  Commit before reading on: which statement correctly names the fatal defect in
  (a) and the measurement damage that (b) does despite its easy arithmetic?
options:
  - text: |
      (a) fails because a paper is not a payment counterparty - nobody signs a
      contract as "paper #40118", so the attribution unit and the payment unit
      differ and the output has to be re-aggregated by a rule nobody has
      justified. (b) is arithmetically easy but confounds source with topic
      (removing a venue removes a subject area) and its venues are not disjoint
      (preprint and published versions overlap), so the duplicate-cluster policy
      rather than the model decides who gets credit.
    correct: true
    explain: |
      Right on both counts. The unit of attribution must be the unit of payment,
      which is what collapses the combinatorics from $2^{100000}$ to $2^{12}$ and
      makes the answer billable. And a venue-level leave-one-out measures how much
      the model needed that subject area, while overlapping preprints let each
      copy measure near zero because the other still covers it.
  - text: |
      (a) fails only because $2^{100000}$ coalitions can never be enumerated; with
      sampling or an approximation it would be the most precise definition, since
      finer granularity always means a more accurate attribution.
    misconception: D7
    explain: |
      The compute objection is the answerable one - sampling exists. The defect
      that sampling cannot fix is that a paper is not an entity that can be paid.
      Choose the attribution unit to match the payment unit and the combinatorics
      collapse on their own; per-document valuation is simply the wrong problem.
  - text: |
      Both definitions are sound as measurements; (b)'s preprint overlap is
      ordinary pipeline hygiene - deduplicate, keep one copy of each paper, and
      the venues become disjoint with nothing of substance decided.
    misconception: D9
    explain: |
      Keeping one copy is exactly the substantive decision. When the preprint and
      the published version are the same passage and you keep one, you have chosen
      which venue is credited for everything that passage teaches the model. At
      corpus scale that choice is made silently millions of times, and here it
      moves money.
  - text: |
      (b) is the fatal one, because with $n \approx 11$ each source's measured
      contribution is a stable property of the corpus that can be measured once
      and billed forever; (a) at least averages that instability away across
      100,000 papers.
    misconception: D16
    explain: |
      Neither granularity buys stability. Retrain with a different seed and a
      source's marginal contribution to held-out loss can swing by more than its
      own magnitude, and data order introduces the largest variation of all. Small
      $n$ makes the coalitions enumerable; it does not make the numbers
      reproducible.
check: choice
```

```beat
id: v1-b10
type: self-explain
concept: d-contributive-corroborative
prompt: |
  Six months from now, a rightsholder disputes your PoC's payout: "your model
  says my catalogue is worth 3.2% of the pool. Prove it."

  Which explanation of what the manifest must have recorded - and of what it can
  never settle - would you give?
options:
  - text: |
      A stable source_id (are this month's rows comparable to last month's),
      document IDs and token counts (what exactly was included and how much),
      shard and byte offsets (that the tokens the model consumed are the ones
      claimed), duplicate-cluster ID plus full cluster membership (whether these
      passages also sat in other sources, and which policy assigned the credit),
      license provenance, code version, ingestion timestamp. What no manifest can
      settle is whether 3.2% is stable: that is a property of the training run,
      and only repeated seeds with a reported noise floor can say whether 3.2% is
      distinguishable from 2.1%.
    correct: true
    explain: |
      Correct, and the second half is the load-bearing half. The manifest
      establishes what went in; it says nothing about whether the number coming
      out survives a rerun. Cluster membership is what answers the single biggest
      objection - "you paid them for my text" - and token counts are what make any
      per-token normalization possible later.
  - text: |
      The same provenance fields - source_id, token counts, offsets, cluster
      membership, license, code version, timestamp. With that record complete and
      auditable, the 3.2% is fully proven: the measurement was computed from
      exactly these documents, so the number follows from the manifest.
    misconception: D16
    explain: |
      Complete provenance proves what entered training, not that the estimate is
      reproducible. Contribution is a noisy random variable - reseed the run and a
      source's marginal effect can move by more than its own magnitude - so a
      billing-grade number needs seeds averaged and a noise floor reported
      alongside it.
  - text: |
      The records that matter are the answer-time logs: which of the rightsholder's
      documents the retriever placed in the context window, how often the system
      cited them, and the per-sentence citation weights. Those logs show where the
      answers actually came from, so they are what substantiates a 3.2% share.
    misconception: D1
    explain: |
      Those are corroborative records - facts about a retriever's ranking computed
      seconds before each answer. Hold the weights fixed, swap the index, and they
      change completely while nothing the model learned changed. A dispute about
      training contribution cannot be settled with evidence about retrieval
      placement.
  - text: |
      Provenance is beside the point: run the trained model and check whether it
      reproduces passages from the catalogue verbatim. Extractable text proves the
      catalogue was used and how much; no verbatim match means there is nothing to
      pay for, whatever the manifest says.
    misconception: D3
    explain: |
      Verbatim emission is a lower bound and a measurement artifact. Ask for the
      same content in a different style and the knowledge is intact with no shared
      n-grams; most of what training data contributes is semantic and never
      resurfaces verbatim. Extraction is corroborating evidence, not the ledger the
      payout was computed from.
check: choice
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
