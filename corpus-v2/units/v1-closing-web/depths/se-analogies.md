## A 402, three cheques, and one answer to pay for

**The pair.** Splitting the pool three ways is like **allocating a shared
Kubernetes cluster's bill across teams**, and it is like **assigning cause in a
postmortem**.

**The shared structure.** In all three you have one fixed total - a pot, a bill,
an outage - and you need a rule that hands out shares. The same three rules
appear every time. Split evenly per team (flat fee). Split by a countable proxy:
pod-hours, log lines, appearances in the trace (citation counting). Or split by
counterfactual: what would the bill have been without this team's workload, which
change if reverted would have prevented the incident (causal attribution). And the
same two pathologies appear every time. The proxy rule measures *visibility* -
the chatty service dominates the trace whether or not it caused anything - and the
counterfactual rule zeroes out *redundant* contributors: when two independent
changes each would have caused the outage alone, reverting either one fixes it, so
both measure as blameless and the shares sum to less than the whole. That
under-summing is exactly what happens to a duplicated source under leave-one-out,
and it is the reason unit v5 exists.

**Where this breaks.** Cluster cost has a conserved ground truth underneath it:
CPU-seconds are additive, they are metered at the kernel, and the allocation
argument is about fairness rather than about measurement. Model capability is not
conserved and has no meter - there is no instrument that reports "this source
supplied 3% of the capability," which is why the counterfactual has to be
*measured by retraining* rather than read off a counter. The postmortem analogy
breaks in the same place from the other side: reverting a commit to test a
counterfactual costs minutes, so you can afford to test many; retraining a model
to test a counterfactual costs a training run, which is why unit v5's
combinatorics matter and unit v8 spends its whole length on cheap approximations.

---

**The pair.** Pay Per Crawl is like **billing an API by request count**, and like
**charging for CDN egress**.

**The shared structure.** The meter sits at the transport layer because that is
where a counter is cheap to install, and the price is therefore attached to
transfer rather than to value. Both produce the familiar distortion: the biggest
payers are the least efficient clients, revenue tracks the buyer's polling
behavior, and a well-engineered client that caches aggressively pays less than a
naive one for identical value received. The gatekeeper's own stated motivation for
moving from Pay Per Crawl to Pay Per Use is precisely this - more than half of AI
crawler requests re-fetch unchanged content - and it is the same reason API
vendors eventually move from per-call pricing to per-seat or per-outcome pricing.

**Where this breaks.** An API request is fungible and the value is roughly
proportional to volume, so per-request pricing is a rough approximation of value.
A page fetch is not: the whole point of this book is that two fetched documents
can differ by orders of magnitude in what they do to a model, and the correlation
between fetch count and contribution is not merely weak, it is nearly absent for
the long tail. Also, an API vendor knows what the caller did with the response
only in aggregate, whereas here the seller has no visibility at all into whether
the bytes went into a training corpus, a retrieval index, or a bucket - which is
what makes the whole verification category a business.

## The gatekeeper

**The pair.** The gatekeeper's position is like **a registrar that also became
the CA, the DNS provider, and the payment processor**, and like **an app store**.

**The shared structure.** In both, an intermediary that started as one useful
layer accumulated the adjacent layers: identity (who are you), policy (what may
you do), distribution (can you be reached), and settlement (who takes the money
and in whose name). Each acquisition is individually defensible and each layer may
even be built on an open specification, but the composition creates a position
where a competitor must either transact on the incumbent's rails or address only
the population outside the walls. Publishers using the word "vendor-locked" are
making the same complaint developers make about app stores, in the same shape, for
the same structural reason.

**Where this breaks.** An app store owns the relationship with the end buyer and
can reject a seller outright; the gatekeeper here owns the chokepoint but the
buyers - the labs - are entirely outside its control and have declined to
participate in its payment schemes. That is a decisive difference: an app store
without buyers is not an app store, and no frontier lab pays into either payment
scheme. Also, moving off a CDN is a DNS change rather than a platform migration,
and roughly 80% of the web is not behind this particular one, so the lock is
softer than the app-store version and the addressable population outside the walls
is much larger.

---

**The pair.** Content leakage past origin blocking is like **a secret committed
to git history**, and like **stale replicas surviving an invalidation**.

**The shared structure.** In each, the thing you are trying to control has been
*replicated*, and the control point you actually hold is the origin. Deleting the
file from HEAD, or purging the cache, or blocking the crawler, addresses one copy
and leaves the fan-out untouched. In all three the mitigation that works is
discovery - scan the history, enumerate the replicas, search the web for
near-duplicates - and the mitigation that feels productive is tightening the
origin. Both engineering cultures learned this the hard way: the useful tool is
`trufflehog` over the whole history, not a stricter `.gitignore`.

**Where this breaks.** A leaked secret can be rotated, which invalidates every
copy at once, no matter where it lives. There is no rotation for an article. This
is the single most important disanalogy in the unit, and it is why leakage
tracking is a measurement business rather than a remediation business: you can
tell a publisher exactly where their content propagated and what that implies for
their negotiating position, and you cannot make the copies stop existing.

## What the deals actually buy

**The pair.** The training-rights versus access-rights split is like **selling a
database dump versus selling an API with an SLA**, and like **a perpetual licence
versus a subscription**.

**The shared structure.** Identical underlying data, two entirely different
revenue shapes and two different relationships. The dump is a one-time transfer:
the buyer takes it home, the seller's leverage ends at the transaction, and the
price has to capture all future value up front. The API is recurring: the seller
stays in the loop, meters usage, retains the ability to reprice, and gets paid for
freshness rather than for content. The industry drift from training deals to
inference-time access deals is the same drift every data business has made, for
the same reasons, and the sellers who moved early captured the recurring side.

**Where this breaks.** A database dump goes stale, and staleness is what pushes
buyers onto the API. A training corpus does not need to stay fresh to have already
done its job: the capability it produced is baked into weights that keep working
forever, and cancelling the relationship does not remove it. So the subscription
analogy misleads about the seller's leverage. In the API case, cutting off a
non-paying customer takes the value away. Here, cutting off a lab that already
trained on your archive takes away only tomorrow's articles. That asymmetry is why
training deals are lump-sum and why the seller's leverage is highest before the
first transfer and roughly zero after.

---

**The pair.** The expert-data market is like **paying for a curated test fixture
suite**, and like **paying a domain consultant rather than buying documentation**.

**The shared structure.** In both, buyers who can obtain the raw material free
still pay large sums for *judgment applied to it*. Anyone can generate inputs; a
good fixture set encodes what a knowledgeable person believes the correct output
is, and that belief is the scarce thing. Labs are in exactly this position with
text: they have more of it than they can use and are short of expert judgment
about it, which is why one labeling vendor books more revenue in a year than the
entire disclosed content-licensing market.

**Where this breaks.** A fixture suite's value is verifiable - run it and see if
it catches regressions - while the value of a labeled dataset to a model is
measurable only by training with and without it, which is the same expensive
counterfactual this whole book is about. So the expert-data market is buying on
reputation and process rather than on demonstrated marginal contribution, which
means it is not evidence that the attribution problem has been solved. It is
evidence that buyers will pay without solving it.

## What the courts priced

**The pair.** The Bartz settlement is like **a licence-compliance settlement after
a software audit**, and like **paying for the font you pirated for the deck**.

**The shared structure.** In each case the penalty prices the *acquisition
defect*, not the ongoing use. The auditor does not charge you per invocation of
the library; it charges for the seats you were running unlicensed, once, and the
cure is to buy the licences and carry on shipping the same product. The rational
response is always the same: acquire cleanly, once, at the lowest legitimate
price, and never enter a metered relationship. That is the response the settlement
makes rational, and it is why "$1.5 billion" is not the beginning of a royalty
regime.

**Where this breaks.** A GPL cure can carry ongoing obligations - source
disclosure, downstream licensing terms - that survive the payment, so not every
compliance settlement ends at "pay once and forget." And the font analogy breaks
where it matters most: nobody argues that using a font substitutes for the market
for that font. The unresolved AI claims are output-side ones, where the argument
is that the model's outputs compete with the source in its own market. That claim
has no analogue in a licence audit, it cannot be cured by buying a clean copy, and
it is the only live path to recurring payment.

## Contributive and corroborative

**The pair.** The two kinds of attribution are like **`git bisect` versus
`grep`**, and like **profiling versus logging**.

**The shared structure.** One member of each pair identifies a cause by
*intervention*: bisect removes and re-tests until the responsible commit is
isolated; a profiler perturbs and samples the running system to find where time
actually goes. The other identifies a *textual or recorded match* by search: grep
finds where the string appears; a log line records what a developer decided to
print. Both are useful, both are used daily, and they systematically disagree.
The function containing the string `timeout` is usually not the function
responsible for the timeout, and the loudest module in the logs is rarely the
bottleneck. That gap between "contains it" and "caused it" is exactly the gap
between corroborative and contributive attribution, and unit v8 shows it
empirically: plain lexical search beats gradient-influence methods at finding the
document that *contains* a fact, precisely because influence is measuring
something else.

**Where this breaks.** Bisect assumes a monotone property and a single culprit -
the bug is present after some commit and absent before it - and it gets a clean
binary answer in $\log_2 n$ steps. Contribution has none of those properties. It is
continuous rather than binary, many sources contribute partially, removing half
the corpus changes the model globally rather than flipping one behavior, and the
measurement is noisy enough that the same experiment run twice can reorder the
result. So bisect is the right analogy for what contributive attribution *asks*
and a badly misleading one for how expensive and how uncertain the answer is.

---

**The pair.** A RAG citation is like **the docs URL in an error message**, and
like **a comment linking the StackOverflow answer someone pasted**.

**The shared structure.** Both point at something that supports or explains the
behavior, both are attached at the surface, and both are chosen by a mechanism
entirely separate from the one that produced the behavior. Someone wrote the URL
into the error string; a retriever ranked the document into the context window.
Neither has any causal relationship with how the system came to work the way it
does, and both can be changed - a different docs site, a different index - without
touching a line of the logic.

**Where this breaks.** The comment analogy is generous to the citation, because a
developer pasting a StackOverflow answer really did get the code from there: that
link is genuinely contributive for that snippet. A retrieval citation is not,
because the model was already capable of the answer before the retriever ran, in
most cases, and the citation was selected for topical similarity to the query
rather than for having taught anything. The exception is the pure-RAG case where
the model contributes nothing but fluency, and that case is worth naming precisely
because it is the one where the corroborative number is nearly causal - for that
answer, and still not for the model.

## Three ways to pay, and what each one misallocates

**The pair.** Pro-rata pooling is like **allocating cloud spend by tag share**,
and like **splitting an on-call bonus by pages received**.

**The shared structure.** A fixed pot, a countable proxy, and shares computed as
proxy-over-total. In every instance the proxy immediately becomes the target.
Teams learn to tag aggressively or to route alerts so that the pages land on them.
Nothing about the underlying good changes; the measurement changes. And in every
instance the participants discover the denominator effect the hard way: your share
falls when someone else's count rises, so a scheme intended to reward contribution
turns into a scheme that rewards relative counting effort, and the arguments are
between participants rather than with the payer.

**Where this breaks.** Internal allocation schemes have a manager who can look at
an obviously absurd result and override it, and the participants are colleagues
with reputations at stake and a shared employer. A public marketplace with
thousands of counterparties has no override, no shared employer, and adversaries
rather than colleagues. That is why internal proxy-gaming stays mild and
marketplace proxy-gaming becomes an industry.

---

**The pair.** Citation farming is like **click fraud in an ad network**, and like
**SEO content farms**.

**The shared structure.** Payout is attached to a countable event; the event is
cheap to manufacture; therefore manufacturing it is a business with a return on
investment that can be computed. In all three, the defense that does not work is
counting more carefully, and the defenses that do work are identity and admission
control - knowing who the participants are and refusing entry - plus raising the
cost of producing the counted event. The AI version is strictly worse than the ad
version on cost, because the "audience" being fooled is a retriever, which is a
deterministic system that can be probed offline until you know what it ranks.

**Where this breaks.** Ad networks have a downstream conversion signal:
eventually somebody either buys something or does not, and that ground truth is
what fraud detection is trained against. A citation pool has no downstream signal.
Nothing observable distinguishes a citation of a farmed page from a citation of a
real one, because the retriever's judgment *is* the outcome being measured. This
is why the honest defense in this space is at the identity layer and at the choice
of what gets counted, and why a scheme that counts genuine contribution is harder
to attack - faking it requires supplying information the model does not already
have, which cannot be synthesized from what it does.

## The strongest case against all of this

**The pair.** Building the measurement layer instead of the payment layer is like
**building observability rather than the workflow product**, and like **selling
picks and shovels rather than mining**.

**The shared structure.** In each, you position on a need that exists under
several futures rather than on one bet about which future arrives. Observability
sells whether the customer's architecture is monolithic, microservice, or
serverless; the requirement to know what your system is doing survives every
architectural fashion. Verification sells whether or not a royalty market appears,
because a disclosure obligation, a procurement diligence process, and a royalty
audit all require the same underlying capability: a defensible statement about
what a model was trained on, with an error rate attached.

**Where this breaks.** Picks and shovels requires a gold rush - the analogy
smuggles in the assumption that a lot of people are already digging. If no one is
mining, the shovel merchant starves alongside the miners. So the analogy is only
load-bearing because the buyers already exist independently of the royalty
question: a regulator with a fining power from August 2026, labs spending billions
on data procurement, and publishers who need evidence for negotiations they are
already having. If you find yourself using this analogy while pointing at a
hypothetical buyer, it has stopped being an argument. The observability analogy
breaks in a different place: observability tools measure a system you control,
with instrumentation you installed, while verification measures somebody else's
model from the outside, adversarially, which is a fundamentally harder statistical
problem and the reason units v6 and v7 are the hardest in this book.

## At the bench: what counts as one source

**The pair.** Choosing the source definition is like **choosing the billing unit
in a metering system**, and like **choosing the aggregate root in a domain model**.

**The shared structure.** In all three the right choice is the entity with an
identity, a lifecycle, and a counterparty - the thing that can be named, that
persists across reruns, and that someone is accountable for. Get it right and
every downstream artifact (invoices, joins, audits, payout tables) has an obvious
shape. Get it wrong and you spend the project's life re-aggregating: measuring at
per-request granularity when customers buy seats means every invoice needs a
rollup rule nobody agreed to, exactly as measuring per paper when payment happens
per publisher means the payout table needs an aggregation rule nobody has
justified. The engineering instinct that the finest available granularity is
safest is wrong here in the same way it is wrong in billing.

**Where this breaks.** An aggregate root can be refactored later with a data
migration, because the underlying facts survive - the events are still in the
store and can be re-projected. Source identity assigned after deduplication cannot
be reconstructed at any price, because the duplicate-cluster membership was
discarded at ingestion and the discarded information is not derivable from what
remains. This is the one place in the pipeline where the usual engineering comfort
- we can always recompute it later - is false, and it is the reason unit v2 treats
dedup as a payout decision rather than as hygiene.

---

**The pair.** The provenance manifest is like **an SBOM**, and like **a lockfile
with hashes**.

**The shared structure.** All three exist so that a later question - what is in
this artifact, where did it come from, which version - is a lookup rather than an
investigation. All three must be produced at build time by the build system,
because reconstructing them afterwards from the artifact is unreliable or
impossible. All three record identity plus version plus a content hash, and all
three are worthless if the identifiers are not stable across builds. The fields
the manifest section demands - source ID, document ID, offsets, token counts,
license, cluster membership, code version, timestamp - are the same fields for the
same reason.

**Where this breaks.** An SBOM's components remain identifiable *in the shipped
artifact*: you can unpack the binary and confirm the library is there, so the SBOM
can be independently audited against the thing it describes. Training data does not
remain identifiable in the weights. Nobody can unpack a model and confirm that
your corpus is inside it. That single asymmetry is why a manifest is necessary and
not sufficient, why "we published our data card" is a claim rather than evidence,
and why units v6 and v7 exist at all: they are the attempt to build the audit step
that the SBOM analogy gets for free.
