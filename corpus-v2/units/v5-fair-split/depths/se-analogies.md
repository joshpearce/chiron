## Eight numbers and a hundred dollars

**The pair.** Splitting a licensing pool across data sources is like
**allocating a shared Kubernetes cluster's bill across the teams running on
it**, and like **attributing an incident to a root cause when three
independent failures had to coincide**.

**The shared structure.** All three have the same defect in the obvious answer.
Bill each team for what it requested and the total exceeds the cluster bill,
because requests overlap with headroom nobody used. Bill each team for what the
cluster would save if that team left and the total falls far short, because
removing any one team leaves the control plane, the base nodes, and the
redundancy intact. In the incident, no single failure was sufficient and no
single failure was necessary, so "what caused it" has no single answer and every
naive rule either over-assigns or under-assigns. In every case the sum of the
parts under one rule is bigger than the whole and under the other rule smaller,
and the truth is a specific point in between that has to be constructed rather
than observed.

**Where this breaks.** The cluster bill has a genuine ground truth available
that a corpus does not: you can meter actual CPU-seconds per pod and get a
defensible usage number without any counterfactual reasoning at all. There is no
metering equivalent for training data - nothing counts how much of a model's
capability came from which document while the run is happening - which is why
the counterfactual is unavoidable here and merely convenient there. The incident
analogy breaks on repeatability: you can re-run a training coalition and get a
number, and you cannot re-run last Tuesday's outage.

---

**The pair.** Leave-one-out as an allocation rule is like **measuring a
service's importance by killing it in production and seeing what breaks**, and
like **judging a library's value by deleting it and counting compile errors**.

**The shared structure.** Each measures replaceability, not value, and each
returns near-zero for exactly the things that are well-engineered. Kill one of
three redundant replicas and nothing breaks, so by this measure the replica is
worthless - and so is the second, and the third. Delete a library that is
vendored in two places and the build survives. In each case the measurement is
correct and the inference from it is wrong, because the quantity is a marginal
effect under one specific configuration, not a share of the whole. The failure
is most severe precisely where redundancy is highest, which is where you were
most hoping for guidance.

**Where this breaks.** Killing a service is a decision-support measurement and
it is the *right* one for the decision it supports: should we keep paying for
this replica. That decision genuinely does turn on the marginal effect, and
leave-one-out genuinely does answer it. So the analogy should not talk anyone
out of computing leave-one-out - the counterfactual unit before this one exists
to compute it, and it remains the ground truth every cheaper method is scored
against. It should only talk them out of dividing money by it. And the compile
error analogy breaks in the other direction: a compiler gives a crisp binary
answer with no noise, whereas every retraining measurement here carries a wobble
larger than many of the effects it is trying to detect.

## What a coalition is worth

**The pair.** A value function over coalitions is like a **memoization table
keyed by a set**, and like a **feature-flag matrix where each combination of
flags is a separately measured build**.

**The shared structure.** In all three, the key is an unordered set and the
value is expensive to produce and cheap to look up, so the whole engineering
discipline is about computing each key once and never again. The set-keyed cache
is why the permutation count never becomes a cost: two orderings that produce
the same predecessor set hit the same cache entry, exactly as two call paths
reaching the same memoized arguments do. And the flag-matrix framing carries the
right warning too - the number of builds is exponential in the number of flags,
which is a fact everyone who has maintained one already knows in their bones.

**Where this breaks.** A memoized function is deterministic and its cache entry
is a fact. A coalition's value is a random variable whose realization depends on
the training seed, so the "cache entry" is an estimate with an error bar, and
writing it once is a bug rather than an optimization. The correct analogue is a
cache of sampled means with recorded variance, and no memoization library you
have used has that shape. The feature-flag analogy breaks on interaction
structure: flag combinations are usually designed to be independent and their
interactions are considered defects, whereas here the interactions between
sources are the entire signal and a corpus with no interactions would not need
this unit at all.

---

**The pair.** Using held-out log-likelihood rather than benchmark accuracy is
like **using p99 latency instead of an SLO breach counter**, and like **using a
continuous error budget instead of a pass/fail health check**.

**The shared structure.** Each pair contrasts a continuous measurement against a
thresholded one, and in each the thresholded version destroys precisely the
information you need for attribution. A change that improves latency from 210ms
to 190ms against a 200ms SLO looks like a total transformation to the breach
counter and a modest improvement to the percentile; a change from 400ms to
300ms looks like nothing to the counter and like real progress to the
percentile. Thresholding turns a smooth quantity into a step function, and a
step function reports zero for every change that does not cross it - which, when
you are measuring hundreds of small marginal effects, is nearly all of them.

**Where this breaks.** The SLO threshold is not an accident of measurement; it
encodes a real business boundary, and there are decisions for which the breach
counter is the correct instrument and the percentile is not. There is no
analogous defence of benchmark accuracy here: nobody's licensing agreement turns
on whether the model crossed 26% on a four-choice quiz, and at these model sizes
that number is sampling noise around a chance floor. The analogy also breaks on
direction - lower latency is better and higher likelihood is better - so any
intuition you carry over about signs needs re-deriving rather than translating.

## Four requirements a contract can cite

**The pair.** The four axioms are like a **set of invariants asserted in
property-based tests**, and like **the guarantees in a service contract that
constrain the implementation without specifying it**.

**The shared structure.** In all three you state properties the answer must
have rather than the procedure that produces it, and the properties are chosen
to be checkable by someone who did not write the implementation. A property test
does not care how you sorted; it cares that the output is a permutation of the
input and is ordered. A rightsholder does not care how you computed the split;
they care that equal contributors got equal shares and that the shares sum to
the pool. And in all three, the properties can be strong enough to pin the
implementation down completely, which is the unusual and valuable case: here,
four properties admit exactly one function.

**Where this breaks.** Property tests are checked against an implementation you
possess, whereas these properties are checked against a table of measurements
whose production the counterparty cannot verify. Symmetry is auditable from the
value table; the value table is not auditable from outside without repeating the
training runs. So the guarantees stop exactly where the measurement begins, and
someone will notice. The service-contract analogy breaks on remedy: a violated
SLA has a defined penalty, and there is no penalty structure whatsoever attached
to a violated fairness axiom, because no jurisdiction has ever adjudicated one.

---

**The pair.** The null player requirement is like **rejecting a benchmark that
rewards padding output length**, and like **refusing to pay a bug bounty for a
report that reproduces a known no-op**.

**The shared structure.** All three close the same class of exploit: a
contributor unilaterally inflates a quantity the payer measures, without
changing the quantity the payer wants. Length-biased judging is defeated by
verbosity that adds no content; a bounty program without a novelty check is
defeated by resubmission; a volume-based data payment is defeated by a text
generator. In each case the defence is not detection after the fact but a
definition that makes the exploit worth zero by construction - score against a
length-controlled baseline, deduplicate against known reports, pay against
measured effect on held-out loss.

**Where this breaks.** Verbosity and duplicate bug reports are cheap to detect
by inspection, and a human reviewer catches them. Padding a training corpus is
not, at scale: half a million words of plausible domain-adjacent filler is
indistinguishable from legitimate content by any cheap inspection, which is
exactly why the defence has to be structural rather than editorial. And the
bounty analogy breaks on the direction of the incentive - a bounty program wants
to encourage submission volume up to a quality bar, while a data payment scheme
has no interest in volume at all.

## Averaging over arrival orders

**The pair.** Averaging marginal contributions over all arrival orders is like
**randomizing the order of a dependency-resolution walk to expose
order-dependent bugs**, and like **shuffling the input to a benchmark so the
result is not an artifact of one traversal**.

**The shared structure.** In each, a procedure that is well-defined for a fixed
order produces order-dependent results, the order is an artifact of the
implementation rather than a fact about the problem, and the fix is to average
over every order rather than to pick a canonical one. Picking a canonical order
would be cheaper and would produce a number; the number would encode a choice
nobody could defend. The averaging is not a smoothing heuristic - it is a
statement that the order carries no information, and the only representation of
"carries no information" is uniform weight.

**Where this breaks.** In a dependency walk or a shuffled benchmark, order
dependence is a *defect* you are trying to detect and eventually remove. Here it
is a permanent feature of the domain: a source genuinely is worth more when it
arrives into an empty corpus than into a full one, that spread is real
information about redundancy, and no amount of engineering removes it. The
spread of a source's marginal contribution across orders is a corpus statistic
worth reporting, not a variance to be engineered down. The analogy also breaks
on the cost model - shuffling a benchmark is free, and averaging over orders
would be prohibitive if the set-keyed cache did not collapse the count.

## Counting the runs: 4,096, not 479 million

**The pair.** The collapse from permutations to coalitions is like
**memoization turning an exponential recursion into a polynomial one**, and like
**a bitmask-indexed dynamic program over subsets**.

**The shared structure.** All three exploit the same observation: the naive
formulation branches over paths, the answer depends only on the state reached,
and the number of states is vastly smaller than the number of paths. Naive
recursive Fibonacci branches over call paths and has a linear number of distinct
states. The permutation formula sums over $n!$ orders and depends only on which
of $2^n$ predecessor sets a source walks into. In each case the implementation
is the same shape - a table indexed by state, filled once, read many times - and
in the subset case the index literally is a bitmask, so `S | (1 << i)` and
`S & ~(1 << i)` are the whole data structure.

**Where this breaks.** Memoization is free because recomputing a cached value
would be wasted work; here each table entry costs a training run, so the table
is not an optimization but the entire budget. That inverts the engineering
instinct: with a memo table you fill it lazily and only where needed, whereas
here you decide up front how many entries you can afford, and that decision -
how many sources to register as distinct players - is the single most
consequential choice in the project. And the bitmask DP analogy breaks on
correctness guarantees: a subset DP over exact values is exact, while these
table entries are noisy estimates, so the "dynamic program" is propagating error
bars and the standard subset-DP toolkit says nothing about that.

---

**The pair.** Choosing Banzhaf with maximum sample reuse over exact Shapley
above fifteen sources is like **switching from exhaustive integration testing to
randomized property testing when the state space explodes**, and like
**replacing exact percentile computation with a t-digest sketch**.

**The shared structure.** Each swaps an exact answer for a bounded-error one at
the point where exactness stops being affordable, and each pays a stated,
quantified price rather than an unknown one. The t-digest does not silently
approximate; it comes with an accuracy guarantee concentrated where you asked
for it. Banzhaf with sample reuse does not silently approximate; it comes with
an evaluation budget that is essentially independent of the number of sources,
and the price is written on the tin - the shares no longer sum to the pool, and
you renormalize and say so.

**Where this breaks.** A sketch's error is purely computational and vanishes if
you spend more; Banzhaf's efficiency violation is not an error at all and does
not vanish with any budget, because it is a different rule rather than an
approximation of the same one. Calling it an approximation in a schedule would
be inaccurate and a competent counterparty would catch it. The property-testing
analogy breaks on the fallback: when randomized testing finds nothing, you have
weak evidence of correctness, whereas when the sampling budget is exhausted here
you have a specific numeric confidence interval, which is a much stronger
epistemic position and should be presented as such.

## The error budget: seeds before coalitions

**The pair.** Averaging seeds before computing shares is like **taking the
median of several load-test runs before comparing two builds**, and like
**requiring a flaky test to pass repeatedly before trusting a green build**.

**The shared structure.** In all three the underlying measurement is
nondeterministic, the quantity of interest is a *difference* between two such
measurements, and differences of nearly-equal noisy numbers have terrible
relative precision. Everyone learns this the first time they benchmark a 3%
optimization against run-to-run variance of 8%. The remedy is identical in all
three: repeat, average, and report the spread alongside the estimate, and refuse
to act on a difference smaller than the spread.

**Where this breaks.** Load-test noise is usually environmental - a noisy
neighbour, a cold cache - and can often be engineered away by isolating the
harness. Training-run noise cannot: it comes from the initialization and the
shuffle, which are intrinsic to the process, and it does not shrink with a
better machine, a longer run, or a bigger model. The only lever is repetition.
The flaky-test analogy breaks harder and in a way worth naming: a flaky test is
a defect, and the correct response is to fix the test. A source's contribution
varying across seeds is not a defect, and treating it as one leads directly to
the belief that a good enough measurement would make the variation go away.

---

**The pair.** The rule "spend the marginal run on a seed, not a coalition" is
like **spending optimization effort on the bottleneck rather than the total**,
and like **fixing the dominant term in an error budget rather than the smallest
one**.

**The shared structure.** Each says the same thing: two independent error
sources add, so effort spent on the one that is already near zero is wasted.
With full enumeration the combinatorial error is exactly zero, so every unit of
remaining uncertainty is seed noise, and every marginal training run should buy
a seed. It is Amdahl's law applied to error rather than to time, and the
arithmetic is the same arithmetic.

**Where this breaks.** Amdahl's law concerns a quantity you can measure directly
while optimizing; here the two error sources are not both observable at once.
The combinatorial error is known to be zero only because you chose to enumerate,
and the seed error is estimated from the seeds you already ran, so the balance
is asserted from the design rather than measured from the result. Above the
cliff, where you are sampling, both terms are nonzero and the analogy becomes
genuinely applicable - and that is exactly the regime where you should stop
reasoning from it and split the budget so the two errors are comparable.

## Gaming the split

**The pair.** The shell company attack is like **Sybil attacks on a
reputation system**, and like **a single owner registering many wallets to farm
a per-address airdrop**.

**The shared structure.** In all three the mechanism is correct with respect to
the identities it can see, one real party controls several of those identities,
and the payoff to controlling several exceeds the payoff to controlling one.
Critically, in all three the mechanism is not being *broken* - every rule is
applied faithfully to every registered identity. The exploit lives entirely in
the gap between "registered identity" and "real party," which is a layer the
mechanism does not model and therefore cannot defend. Every serious system in
these categories eventually grows an identity layer for exactly this reason, and
it always sits outside the elegant part.

**Where this breaks.** A Sybil attack usually costs the attacker something -
stake, proof of work, a verified phone number - so the defence can price it out.
Fragmenting a data catalogue costs an incorporation fee and produces no new
content, so there is nothing to price. The attack is also *cheaper to evaluate*
than an honest change, because duplicate shells create no new coalitions to
train: the attacker can compute their gain from the same cached table the payer
published. And it breaks in one more direction that matters for the defence:
unlike wallets, data has a fingerprint. Two shells offering the same catalogue
are detectable by content overlap, which is a defence Sybil-resistant systems
would love to have and mostly do not.

---

**The pair.** Merging registrants by content fingerprint before computing the
split is like **deduplicating crash reports by stack-trace signature before
counting them**, and like **collapsing duplicate rows on a natural key before
running an aggregate**.

**The shared structure.** Each inserts a normalization step between raw
submissions and an aggregate that would otherwise be dominated by whichever
submitter duplicated the most. In all three the normalization is a *policy*
decision disguised as a data-cleaning step: which stack frames count as the
signature, which columns constitute the natural key, what Jaccard threshold
makes two catalogues one source. Get the threshold wrong in either direction
and you either merge genuine independents or wave through shells. And in all
three the normalization must happen before aggregation, not after, because
afterwards the information needed to do it has been summed away.

**Where this breaks.** Crash deduplication and row collapsing are usually
uncontested - nobody's payout depends on the signature function, so it can live
in code and be tuned by whoever owns the pipeline. Here the threshold moves
money between named parties, so it belongs in the contract schedule with a
stated value and a stated appeal process, not in a config file. The moment a
data-cleaning parameter becomes a payment parameter, it stops being an
engineering choice, and treating it as one is how a well-built system loses an
argument it should have won.

## The honest position

**The pair.** The held-out set's influence on the split is like the way
**benchmark selection determines which database "wins" a comparison**, and like
the way **choosing a baseline period determines whether a metric shows growth**.

**The shared structure.** In each, a methodologically clean procedure sits
downstream of one unconstrained choice, and that choice determines the outcome
more than anything downstream of it does. Everyone in the room can verify the
arithmetic and nobody can verify the choice, because it is not a factual
question - it is a decision about what should count. The defence in all three is
identical and is not technical: name the choice up front, justify it against the
purpose, and commit to it before seeing the results, which converts an
unfalsifiable result into a reproducible one.

**Where this breaks.** A benchmark choice can often be neutralized by running
several benchmarks and reporting all of them, and the additivity property here
makes that route unusually clean - shares over a combined held-out set are
exactly the sum of shares over the parts, so you can publish a per-domain
decomposition rather than one contested aggregate. That is a stronger defence
than the analogy offers, and it is worth using. Where it stops working: someone
still chose the set of domains, and the weights implied by their relative sizes
are still a choice. The regress terminates in disclosure, not in objectivity.

## At the bench: the split

**The pair.** The cached utility table is like a **build cache keyed by content
hash**, and like a **precomputed lookup table replacing a hot analytic
function**.

**The shared structure.** All three trade a large one-time cost for near-free
repeated queries, and in all three the artifact outlives the computation that
produced it and becomes the thing downstream work depends on. The table here is
about thirty kilobytes and answers every question in the unit - three allocation
rules, the modularity diagnostic, the seed bootstrap, the shell attack and its
fix - with no further training. Treat it the way you would treat a build cache:
content-address it by the exact corpus manifest and configuration that produced
it, because a table computed under a different tokenizer or a different held-out
set is a different table and mixing them silently is the failure mode.

**Where this breaks.** A build cache entry is exact and a stale hit is a
correctness bug with a crisp definition. A table entry here is a noisy estimate,
so "stale" is a spectrum: the same configuration re-run with new seeds gives
different numbers, both valid, and the right thing to store is the seed-level
values rather than the mean, so a bootstrap remains possible later. The lookup
table analogy breaks on interpolation - you can interpolate between entries of a
tabulated smooth function, and you cannot interpolate between coalitions,
because the index is a set and there is nothing between two sets.

## What you can now do

**The pair.** Leading a pitch with the measured signal-to-noise ratio is like
**leading a postmortem with the detection gap rather than the fix**, and like
**publishing a benchmark's confidence interval alongside the headline number**.

**The shared structure.** Each volunteers the number that a skeptical expert
would otherwise have to extract, and each converts a potential ambush into a
credential. The postmortem that opens with "we did not notice for four hours"
has already answered the hardest question in the room. The benchmark that
reports its interval has pre-empted the objection that its 3% lead is inside its
own noise. And a data-attribution pitch that opens with a measured
signal-to-noise ratio, on either side of the threshold, has demonstrated the one
capability that distinguishes it from every scheme that ships a pie chart.

**Where this breaks.** A postmortem's audience wants the honest version and is
professionally obligated to reward it. A room full of investors is not, and
leading with a below-threshold measurement will lose some of them regardless of
how correct it is. The analogy holds on the merits and not on the incentives,
and the honest framing of that is: this is a bet that the counterparties worth
having are the ones who check. Given that the threshold result is published, the
economists have already been hired by the incumbent standards body, and the
first technical question anyone informed will ask is about noise, it is not a
reckless bet - but it is a bet, and it should be made deliberately rather than
stumbled into halfway through a slide.
