## The empty chair

The template's "top 10% of scraped domains by volume of content" sounds like a
substantial disclosure. Price it against how web corpora are actually
distributed, because the shape of the distribution is what decides whether you
appear in the document.

Domain volumes in a scraped corpus follow a heavy-tailed law well approximated
by a power law: the number of domains contributing at least $x$ tokens falls
like $x^{-\gamma}$, with $\gamma$ somewhere near 1 for web text. Under such a
law, rank the domains by volume, and the share of total tokens contributed by
the top $r$ fraction of domains is roughly

$$\text{share}(r) \approx r^{\,1 - 1/\alpha}$$

where $\alpha > 1$ is the tail exponent of the size distribution. At $\alpha = 2$
- a moderate tail - the top 10% of domains carry $0.1^{0.5} \approx 32\%$ of the
tokens. At $\alpha = 1.2$, closer to what web crawls look like, the top 10% carry
$0.1^{1 - 0.833} = 0.1^{0.167} \approx 68\%$.

Now read the number the other way, which is the way that matters to you. A crawl
touching 40 million distinct domains discloses 4 million of them - a genuinely
enormous list, filed as a compliance artifact - while omitting 36 million.
Whether your domain is in the disclosed 10% is decided entirely by your token
volume relative to the largest sites on the internet, and a specialist journal
is at the wrong end of a distribution whose top is Reddit.

The disclosure is therefore honest, large, expensive to produce, and structurally
incapable of answering an individual question. That is not a criticism of the
drafters; a form that named every domain would be a crawl log, and no regulator
mandates that. It is the argument for why the empty chair is empty by design
rather than by oversight.

## Constructing a null

State the test formally once, because every failure in this literature is a
violation of one line of it.

A hypothesis test fixes a null hypothesis $H_0$, a test statistic $T$ computed
from data, and a rejection region. The **p-value** of an observed $t$ is

$$p = \Pr\big(T \geq t \mid H_0\big)$$

for a one-sided upper test. The defining property that makes this useful is:

$$\text{if } H_0 \text{ is true, then } p \sim \text{Uniform}(0,1)$$

A valid p-value is uniform under the null. That is what "$\Pr(p \leq \alpha) =
\alpha$" means, and it is the entire content of the phrase "false-positive rate."
Everything the unit calls validity is this line, and everything it calls a
procedural failure is a way of breaking it.

Two ways to break it, and they are the two the unit spends its time on.

**Break the distribution.** If you do not know $\Pr(T \geq t \mid H_0)$, you
cannot compute $p$ at all, and any number you report in its place has no uniform
property to appeal to. This is the retrospective failure: $H_0$ is "this model
was not trained on my data," and the distribution of loss-based statistics under
that hypothesis is a function of a model that was never trained.

**Break the fixing.** If the statistic $T$, the threshold $\alpha$, or the data
partition is chosen after seeing the data, then the quantity you computed is not
$\Pr(T \geq t \mid H_0)$ for a fixed $T$; it is $\Pr(\max_j T_j \geq t \mid H_0)$
over the implicit family of statistics you would have been willing to report.
That maximum is stochastically larger than any individual $T_j$, so the reported
$p$ understates the true one, often by orders of magnitude. Selecting after the
fact does not weaken the inference slightly; it replaces the statistic with a
different one whose distribution you never characterized.

**What the keyed generator buys, exactly.** Let $G_K$ be a deterministic
generator seeded with key $K$, producing items $X_1, \ldots, X_M$. The key also
determines a partition into a published set $P$ and a control set $C$. Under
$H_0$ - the model was not trained on any of them - the items are exchangeable:
for any permutation $\pi$ of the indices, the joint distribution of
$(T(X_{\pi(1)}), \ldots, T(X_{\pi(M)}))$ equals that of
$(T(X_1), \ldots, T(X_M))$, because a single generator produced them all and
publication is the only thing that distinguishes the two subsets.

Exchangeability is the property. It is what licenses treating the control
statistics as draws from the same distribution as the published statistics under
$H_0$, and it also licenses an exact permutation test as an alternative to the
normal approximation: enumerate (or sample) partitions of the $M$ items into
sets of sizes $|P|$ and $|C|$, compute the statistic on each, and read off the
rank of the observed value. That test needs no normal approximation and no
independence assumption between items, which makes it the right fallback when
attribute-level dependence makes $\sigma_0$ untrustworthy.

## The test, end to end

The canonical computation used the normal approximation and treated $p_0$ as
known. Here are both corrections, with the worked numbers.

**The exact null.** Under $H_0$, the count $k$ of hits among $n$ published items
is exactly binomial:

$$\Pr(K = j) = \binom{n}{j} p_0^{\,j} (1 - p_0)^{\,n - j}, \qquad
p = \Pr(K \geq k) = \sum_{j = k}^{n} \binom{n}{j} p_0^{\,j} (1-p_0)^{\,n-j}$$

At $n = 100$, $p_0 = 0.10$, $k = 25$, the exact upper tail is about
$1.1 \times 10^{-5}$, against the normal approximation's $2.9 \times 10^{-7}$.
The exact value is two orders of magnitude larger, because the binomial's right
tail is heavier than the normal's out at four to five standard deviations, and
the approximation degrades exactly where you most want it. Report the exact tail
in any document that will be read adversarially; use the normal approximation to
think with.

The usual validity condition for the approximation is $n p_0 (1-p_0) \geq 9$,
which the worked example meets exactly ($100 \times 0.1 \times 0.9 = 9$) - that
is, it sits at the boundary. A continuity correction, replacing $k$ with
$k - 0.5$, helps in the body of the distribution and does not rescue the far
tail.

**The estimated null rate.** The canonical computation set $p_0 = 0.10$ from
1,000 control items and then treated it as exact. The correct statistic compares
two measured proportions. Write $\hat p_1 = k/n$ for the published rate,
$\hat p_0 = k_C/m$ for the control rate over $m$ control items, and $\bar p$ for
the pooled rate under $H_0$ that the two are equal:

$$\bar p = \frac{k + k_C}{n + m}, \qquad
z = \frac{\hat p_1 - \hat p_0}{\sqrt{\bar p (1 - \bar p)\left(\frac{1}{n} + \frac{1}{m}\right)}}$$

On the worked numbers, $n = 100$, $k = 25$, $m = 1{,}000$, $k_C = 100$:

$$\bar p = \frac{125}{1{,}100} = 0.11364, \qquad
z = \frac{0.15}{\sqrt{0.11364 \times 0.88636 \times 0.011}} = \frac{0.15}{0.03329} = 4.51$$

The drop from 5.00 decomposes into two separate costs, and it is worth keeping
them apart because only one is fixable.

*Pooling costs 5.00 to 4.73.* Under $H_0$ the two rates are equal, so the best
estimate of the common rate uses both samples: 0.1136, not 0.10. A larger $p_0$
gives a larger $\sqrt{p(1-p)}$ and a smaller gap. This cost is not removable; it
is the correct thing to do.

*The finite control set costs 4.73 to 4.51.* The $1/m$ term inflates the standard
error by the factor

$$\sqrt{1 + \frac{n}{m}} = \sqrt{1 + 0.1} = 1.0488$$

and $4.726 / 1.0488 = 4.507$. This cost IS removable, and cheaply: at
$m = 10n$ it is 4.9%, at $m = 100n$ it is 0.5%, and control items cost nothing
but generation time. The design rule follows directly - make $m$ at least $10n$,
and stop worrying about it.

One further note for confidence intervals rather than tests. The pooled standard
error above is correct for testing $H_0: p_1 = p_0$. For an interval on the
difference $p_1 - p_0$, use the unpooled form,
$\sqrt{\hat p_1(1-\hat p_1)/n + \hat p_0 (1 - \hat p_0)/m}$, which here is
0.0443 and gives $z = 3.38$. The two disagree because they estimate the variance
under different assumptions, and a report that quotes one number for both
purposes is quoting the wrong one somewhere.

## Canary design is the whole game

Two pieces of machinery underneath the design rules.

**Exposure, the metric the canary literature is built on.** Insert a canary
drawn from a randomness space $\mathcal{R}$ of size $|\mathcal{R}|$ - say a
sequence of $d$ digits, so $|\mathcal{R}| = 10^d$. After training, rank all
$|\mathcal{R}|$ candidate canaries by the model's log-likelihood and find the
rank $r$ of the true one. Exposure is

$$\text{exposure} = \log_2 |\mathcal{R}| - \log_2 r$$

Read it as bits of evidence. A canary at rank 1 out of $10^6$ has exposure
$\log_2 10^6 \approx 19.9$ bits; a canary at the median rank $5 \times 10^5$ has
exposure 1 bit, which is what "no evidence" looks like on this scale. The metric
is clean and its weakness is exactly the unit's theme: it requires enumerating or
sampling the randomness space and scoring candidates by likelihood, so it needs
probability access, and it says nothing about a fictitious-knowledge watermark
whose "randomness space" is the space of plausible prose.

**Why four attributes beat one.** Model the model's recall of an entity's
attributes as follows. Let $q$ be the chance the model has acquired the entity at
all - that its representation of the invented name is bound to anything - and,
conditional on acquisition, let $s$ be the chance any given attribute is
recovered. Let $p_0$ be the chance of a correct answer by coincidence when the
entity was not acquired.

With one attribute per entity, the per-question hit rate is
$q s + (1-q) p_0$, and $n$ entities give $n$ questions. With $a$ attributes per
entity, the same $n$ entities give $an$ questions at the same per-question rate,
so the count statistic has $a$ times the sample size for the same amount of
published content. The mean-to-standard-deviation ratio grows like $\sqrt{a}$:

$$z \propto \frac{an\,(qs + (1-q)p_0 - p_0)}{\sqrt{an\,p_0(1-p_0)}} = \sqrt{an}\cdot\frac{q(s - p_0)}{\sqrt{p_0(1-p_0)}}$$

so four attributes buy a factor of $\sqrt{4} = 2$ in $z$ over one, for free.

But the measured effect is larger than $\sqrt{a}$, which means $q$ and $s$ are
not constant in $a$: a bundle of four mutually reinforcing attributes about one
entity is easier to acquire than one isolated fact, so $q$ itself rises with $a$.
That is a claim about how models memorize coherent structures rather than about
sampling, and it is the empirical reason the recommendation is "four or more"
rather than "as many attributes as your sample size needs."

**The dependence this creates, stated precisely.** The $an$ questions are not
independent, because the $a$ questions about one entity share the same $q$.
Conditional on acquisition they are close to independent; unconditionally they
are positively correlated. Write $\rho$ for the within-entity correlation. The
variance of the count is then

$$\text{Var}(K) = an\,p_0(1-p_0)\,\big[1 + (a-1)\rho\big]$$

The bracket is the design effect. At $a = 4$ and $\rho = 0.3$ it is
$1 + 3 \times 0.3 = 1.9$, so the true standard deviation is $\sqrt{1.9} = 1.38$
times the naive one and the naive $z$ is overstated by 38%. The honest fixes, in
order of preference: take the entity as the unit of the count (one hit per
entity, $n$ trials, no within-entity dependence at all), or estimate $\rho$ from
the control set - where you have hundreds of entities and can measure it
directly - and inflate $\sigma_0$ by the design effect.

That second option is the one to notice. The control set is not only the null
rate; it is a large sample from which every nuisance parameter of the null,
including its dependence structure, can be estimated. Controls being free is
worth more than it first appears.

## How many watermarks, and how often

Make the scaling law precise enough to plan against.

The empirical finding underneath the canonical section is that memorization of a
passage is governed by its frequency relative to corpus size, and that
memorization strength grows roughly log-linearly in duplication count. Write $c$
for the number of occurrences of a watermark in the training corpus and $D$ for
the corpus size in tokens. The claim is that detection strength depends on the
pair only through the ratio, and log-linearly in it:

$$\text{signal} \approx \beta \log \frac{c}{D} + \text{const}$$

Two consequences fall straight out.

**The scaling rule.** Holding signal fixed while $D$ grows by a factor $g$
requires $c$ to grow by the same factor $g$, since $\log(gc / gD) = \log(c/D)$.
That is the "corpus grows fivefold, plant fivefold" rule, and it is exact in this
model rather than a rule of thumb.

**The cost of being small is worse than linear in the thing you care about.**
Your budget is a density $\delta$ in your own corpus of size $A$ tokens, so the
tokens you may spend are $\delta A$, and at $L$ tokens per document with $W$
distinct watermarks the occurrences per watermark are

$$c = \frac{\delta A}{L\,W}$$

Signal grows like $\log c$, so signal grows like $\log A$: doubling the size of
your archive buys you a fixed additive increment of evidence, not a doubling.
Conversely, halving your archive costs you the same fixed decrement. A publisher
with a hundredth of another's archive is not a hundred times worse off; they are
$\log 100 / \log 2 \approx 6.6$ doublings worse off, which is a lot, and which is
also why pooling small publishers under a shared key works. Pooling multiplies
$A$, and the shared key keeps the null constructible across the pool, since all
items still come from one generator.

Note where the log helps and where it does not. It softens the disadvantage of
being small, and it also flattens the return on spending more: raising density
from 0.1% to 0.2% buys one additive increment, and it costs twice as much
fabricated content. There is a sensible operating point and it is not far above
the detector's floor.

**Solving for the plan.** Fix a required occurrence count $c^\star$ (about 90 as
a working figure) and a floor $W \geq 25$ on distinct watermarks. Then the
density you must accept is

$$\delta = \frac{c^\star L W}{A}$$

and the worked cases follow immediately. At $A = 4 \times 10^8$, $L = 150$,
$W = 25$, $c^\star = 90$: $\delta = (90 \times 150 \times 25)/(4 \times 10^8) =
337{,}500 / 4\times10^8 = 0.084\%$, comfortably inside a 0.1% budget. At
$A = 4 \times 10^7$ the same numerator over a tenth of the denominator gives
$0.84\%$, which is the small-publisher result, and no rearrangement of $W$
helps, since $W$ is pinned from below by the count statistic.

## Your own private control

The variance-reduction argument, done properly, because the size of the win is
determined by one quantity the canonical section stated and did not derive.

Let $X$ be the per-token surprise of the published version of a document and $Y$
the average per-token surprise of its private siblings. Both are random across
documents. The unpaired test compares the mean of $X$ against a reference mean;
the paired test uses $d = Y - X$, whose expectation is the training effect.

For any two random variables,

$$\text{Var}(Y - X) = \text{Var}(Y) + \text{Var}(X) - 2\,\text{Cov}(X, Y)$$

Write $\sigma^2$ for the common per-document variance and $\rho$ for the
correlation between a document and its own rephrasings. Then

$$\text{Var}(d) = 2\sigma^2(1 - \rho)$$

Pairing helps whenever $\rho > 1/2$, and helps enormously as $\rho \to 1$. The
relative efficiency of paired to unpaired, measured as the ratio of variances, is

$$\frac{\text{Var}(X)}{\text{Var}(d)} = \frac{1}{2(1-\rho)}$$

and since $z$ scales with $1/\sqrt{\text{Var}}$ and required sample size scales
with $\text{Var}$, the sample-size saving is exactly that factor.

Check it against the canonical numbers. There, $\sigma = 0.60$ and the paired
standard deviation was 0.12, so $2\sigma^2 (1-\rho) = 0.12^2 = 0.0144$, giving
$1 - \rho = 0.0144 / (2 \times 0.36) = 0.02$ and $\rho = 0.98$. Relative
efficiency $1/(2 \times 0.02) = 25$. That is precisely the 25x sample-size figure
the canonical section obtained by solving for $n$, and it arrives here as a
one-line consequence of a correlation of 0.98.

A correlation of 0.98 between a document and its own keyed rephrasing is not
optimistic; it is what you should expect, since the rephrasings hold topic,
length, register, vocabulary difficulty and subject matter fixed and vary only
surface form. The design's entire power comes from manufacturing that
correlation deliberately, which is a thing you can only do if you generate the
siblings yourself before publishing.

**The distribution-free alternative.** If you distrust the normal approximation
on likelihood differences - and per-token surprise has heavy tails across
documents - use the sign test. Under $H_0$, with $k$ siblings per document, the
published version is the lowest-surprise sibling with probability exactly $1/k$,
by exchangeability. Count how many of $n$ documents have their published version
ranked first and test against $\text{Binomial}(n, 1/k)$. At $n = 100$ and
$k = 5$: $\mu_0 = 20$, $\sigma_0 = \sqrt{100 \times 0.2 \times 0.8} = 4$, and an
observed 40 gives $z = 5.00$.

The sign test throws away the magnitude of each difference and keeps only its
rank, so it costs power - roughly a third of it in the Gaussian case, by the
usual asymptotic relative efficiency of $2/\pi \approx 0.64$ for the sign test
against the t-test. What it buys is that its null is exact, requires no
distributional assumption whatsoever, and follows from exchangeability alone -
the same property that made the whole construction work. In an adversarial
report, that trade is usually worth making explicitly, and the honest move is to
report both.

## The pre-commitment ledger

**Bonferroni, proved, because the proof is the reason it needs no independence
assumption.** Let $A_j$ be the event that test $j$ falsely rejects, for
$j = 1, \ldots, m$, each at level $\alpha/m$. The family-wise error rate is
$\Pr(\bigcup_j A_j)$, and the union bound gives

$$\Pr\Big(\bigcup_{j=1}^{m} A_j\Big) \leq \sum_{j=1}^{m} \Pr(A_j) = m \cdot \frac{\alpha}{m} = \alpha$$

No independence anywhere. The union bound holds for arbitrarily dependent events,
which is precisely why Bonferroni is the right default in a setting where the
dependence structure is unknown and contested. Its cost is conservatism: under
strong positive dependence the true family-wise rate is well below $\alpha$, so
you are giving away power to buy an assumption-free guarantee.

**Sidak, for comparison.** Under independence the exact per-test level is
$\alpha' = 1 - (1-\alpha)^{1/m}$, which at $\alpha = 0.05$ and $m = 40$ gives
0.001282 against Bonferroni's 0.00125. The gain is under 3%. Bonferroni is
essentially free relative to the exact independent correction, which is the
practical reason nobody bothers with Sidak.

**Benjamini-Hochberg, and when it is the right instrument.** Controlling the
family-wise error rate asks: what is the chance of ANY false rejection?
Controlling the false discovery rate asks: among the tests I rejected, what
fraction do I expect to be false? The BH procedure sorts the $m$ p-values
ascending as $p_{(1)} \leq \cdots \leq p_{(m)}$, finds the largest $j$ with

$$p_{(j)} \leq \frac{j}{m}\alpha$$

and rejects all tests up to that $j$. It controls the false discovery rate at
$\alpha$ under independence and under positive regression dependence, which
covers the shared-model case reasonably well.

Which to use is a business decision, and it should be committed in the protocol
alongside everything else. Family-wise control is the right posture for a report
that will be used as evidence against a named party, where one false accusation
is the whole loss. False-discovery control is the right posture for screening,
where the output is a list of leads and a known contamination fraction is
acceptable. Selling the two under the same name is a mistake the market will
eventually price.

**Optional stopping, quantified, since it is the subtlest of the three
attacks.** Suppose you test each new model release at $\alpha = 0.05$ and stop
when one fires. With $R$ independent releases, the chance of at least one false
positive is $1 - 0.95^R$: 23% at $R = 5$, 40% at $R = 10$, 64% at $R = 20$. A
protocol that does not fix $R$ in advance has an unbounded false-positive rate,
approaching 1 as you keep testing. This is why "which models, which versions,
how many, over what window" belongs in the committed document, and why the
correct family size $m$ counts release-tests rather than clients.

## What a court will actually credit

The evidentiary standard for expert testimony asks, among other factors, about a
known or potential rate of error and about the existence of standards controlling
the technique's operation. Map those onto what the construction actually
produces.

*Known or potential error rate.* The false-positive rate is $\alpha$, and the
uniformity of the p-value under $H_0$ is what makes that a real rate rather than
a label. Crucially, the rate is derived from a null a court-appointed expert can
reconstruct: hand them the key and the derivation recipe and they regenerate the
control set themselves. That is unusual. Most statistical evidence asks a court
to accept a modelling assumption; this asks it to accept a hash.

*Standards controlling the operation.* This is the pre-commitment ledger,
restated in the vocabulary of the standard. The protocol document is the
operating standard, the timestamp is the proof it governed the run, and the
verifier script is the reproduction.

**And the fallacy to keep out of the report, because it is the one an opposing
expert will accuse you of.** The p-value is $\Pr(\text{data} \mid H_0)$. It is
not $\Pr(H_0 \mid \text{data})$. Bayes' rule connects them:

$$\Pr(H_0 \mid \text{data}) = \frac{\Pr(\text{data} \mid H_0)\Pr(H_0)}{\Pr(\text{data} \mid H_0)\Pr(H_0) + \Pr(\text{data} \mid H_1)\Pr(H_1)}$$

Work an example. Suppose $p = 2.9 \times 10^{-7}$, the test has power 0.8 under
$H_1$, and your prior that this particular developer trained on this particular
archive is a pessimistic 1 in 1,000. Then

$$\Pr(H_0 \mid \text{data}) = \frac{2.9\times10^{-7} \times 0.999}{2.9\times10^{-7} \times 0.999 + 0.8 \times 0.001} = \frac{2.90\times10^{-7}}{8.003\times10^{-4}} \approx 3.6 \times 10^{-4}$$

So even at a 1-in-1,000 prior, the posterior chance of the null is about 1 in
2,800. The finding survives a hostile prior, which is worth computing and saying
out loud - but the sentence to write in the report is the p-value with its
definition attached, not a posterior, because the prior is the other side's to
argue and claiming one hands them the argument.

## The audit market, and its graveyard

**Where $C = 6ND$ comes from**, since the zero-knowledge arithmetic rests
entirely on it. Consider one parameter in a matrix multiply. In the forward pass
it participates in one multiply and one add: 2 FLOP. In the backward pass, the
gradient with respect to the activations requires the same multiply-add against
that parameter, and the gradient with respect to the parameter itself requires
another: 4 FLOP. Total 6 FLOP per parameter per token. Multiply by $N$ parameters
and $D$ tokens:

$$C = 6ND$$

Attention's quadratic term and the embedding layers are omitted, which is a few
per cent at the shapes involved and irrelevant against a four-order-of-magnitude
conclusion.

**The proving arithmetic, with its sensitivity.** At $N = 10^9$ and 20 tokens per
parameter, $D = 2\times10^{10}$ and $C = 1.2 \times 10^{20}$ FLOP. At the best
published proved throughput of $4 \times 10^8$ FLOP/s:

$$t = \frac{1.2\times10^{20}}{4\times10^{8}} = 3\times10^{11}\ \text{s} = 9{,}500\ \text{years}$$

Sensitivity, because a claim of this shape should carry one. Suppose proving
throughput improves by $10^3$ - a thousandfold, far beyond any trend in the
field. The 1B model then takes 9.5 years. Suppose parallelism across 1,000
provers on top of that: 3.5 days for a 1B model, which is finally interesting,
and which is still $10^3$ times too slow for a frontier model that is itself
$10^3$ times larger. The conclusion needs six orders of magnitude of improvement
before it changes, and it does not depend on the exact throughput figure being
right to within a factor of ten.

**Why Proof-of-Learning's spoofing is structural rather than a bug.** The scheme
asks a trainer to publish a chain of checkpoints $\theta_0, \theta_1, \ldots,
\theta_T$ such that consecutive pairs are consistent with a small number of
gradient steps on the claimed data. A verifier spot-checks a subset of the
transitions by re-running them. The attack: start from the final weights and
construct a chain backwards, choosing intermediate points that satisfy the local
consistency check without having done the work. The verifier's check is local,
the attacker's freedom is global, and local checks on a path do not constrain how
the path was found. This is not fixable by checking more transitions; it is
fixable only by checking all of them, which is re-running the training.

## The strongest case against the notary

**The adversary's filtering problem, as a hypothesis test in the other
direction.** Suppose a lab wants to strip fictitious-knowledge watermarks from a
crawl. Their detector must decide, for each document, whether the entities in it
are real. Write $\pi$ for the fraction of documents that are watermarks. At a
0.1% watermark density within an affected publisher's corpus, and with such
publishers a small fraction of the crawl, $\pi$ across the whole corpus is on the
order of $10^{-5}$ or smaller.

The lab's detector has some true-positive rate $\text{TPR}$ and false-positive
rate $\text{FPR}$. The fraction of flagged documents that are genuinely
watermarks - their precision - is

$$\text{precision} = \frac{\pi\,\text{TPR}}{\pi\,\text{TPR} + (1-\pi)\text{FPR}}$$

At $\pi = 10^{-5}$, $\text{TPR} = 0.9$, and an already-excellent
$\text{FPR} = 10^{-3}$, precision is

$$\frac{10^{-5} \times 0.9}{10^{-5} \times 0.9 + 10^{-3}} = \frac{9\times10^{-6}}{1.009\times10^{-3}} \approx 0.9\%$$

Ninety-nine per cent of what the lab deletes is real content it wanted. To make
filtering worthwhile they need an FPR around $10^{-7}$ at high TPR, which is
asking a classifier to distinguish invented entities from genuinely rare real
ones at ten million to one. That is the base-rate argument in the design's
favour, and it is the strongest quantitative answer to the "the adversary just
filters" objection.

It is not a proof of safety. The lab does not have to filter the whole crawl;
they can filter only corpora from publishers known to run a watermarking service,
which raises $\pi$ within that subset by orders of magnitude and collapses the
argument. That is the real risk, and the countermeasure is structural rather than
technical: a notary whose clients are not enumerable, or whose watermarked
corpora are indistinguishable from unwatermarked ones at the collection level.
This is unresolved, and the arithmetic above is the reason it is worth resolving
rather than abandoning.

## At the bench: the notary run

**Powering the run before spending compute on it.** You are about to train two
small models. Before you do, ask what $k$ the test would need to reach the
committed threshold, and whether the design can plausibly produce it.

With $n = 100$ published questions, $p_0 = 0.10$, and a Bonferroni-corrected
threshold of $z \geq 3.02$:

$$k^\star = \mu_0 + 3.02\,\sigma_0 = 10 + 3.02 \times 3 = 19.1 \rightarrow 20$$

Twenty hits out of a hundred questions. Now ask whether the design delivers it.
If the attribute recall rate on an acquired entity is around 0.75, as reported,
and the entity acquisition rate is $q$, the expected count is
$100\,[\,0.75q + 0.10(1-q)\,]$, and setting that equal to 20 gives
$q = 10 / 65 \approx 0.15$. The run needs the model to have acquired only about
15% of the planted entities to clear a corrected threshold. That is a comfortable
margin, and computing it before training is what distinguishes an experiment from
a hope.

**The negative control, tested exactly.** On the unwatermarked model the correct
statement is not "the test did not fire" but a specific tail probability. If the
negative-control run yields $k_{\text{neg}}$ hits, its exact binomial upper tail
against $\text{Binomial}(100, p_0)$ is the number to report. A single run is one
draw from the null and cannot estimate a rate; what it can do is falsify the
null's calibration if it lands far out. Note the asymmetry: a fire on the
negative control is decisive evidence that something is wrong, while a
non-fire is weak evidence that everything is right - one draw from a
distribution you claimed puts 99.9% of its mass below the threshold is a very
weak test of that claim.

**What would make it a real calibration check.** Split the 250 control entities
into ten disjoint groups of 25, and run the full detection procedure on each
group against the unwatermarked model as if it were the published set. Ten
independent draws from the null, each giving a p-value; under a correct null
those ten p-values are uniform on $[0,1]$, which you can check by eye or with a
Kolmogorov-Smirnov statistic. That converts a single anecdote into a calibration
plot, and it costs nothing but inference time, because the controls are already
generated and the model is already trained.

## What you can now do

Every formula in the unit, in dependency order, with the conditions under which
each holds.

$$p_0 = \frac{k_C}{m} \qquad \mu_0 = n p_0 \qquad \sigma_0 = \sqrt{n p_0 (1-p_0)} \qquad z = \frac{k - \mu_0}{\sigma_0}$$

valid when $n p_0 (1-p_0) \geq 9$ and the $n$ items are independent under $H_0$;
otherwise use the exact binomial tail and the entity rather than the attribute as
the unit.

$$z_{\text{2-prop}} = \frac{\hat p_1 - \hat p_0}{\sqrt{\bar p(1-\bar p)(1/n + 1/m)}}, \qquad \bar p = \frac{k + k_C}{n + m}$$

the correct form when $p_0$ is estimated; the inflation over the one-sample form
is $\sqrt{1 + n/m}$, which is under 5% at $m \geq 10n$.

$$\text{Var}(K) = n p_0 (1-p_0)\big[1 + (a-1)\rho\big]$$

the design effect from $a$ questions per entity with within-entity correlation
$\rho$; estimate $\rho$ on the control set.

$$\text{Var}(d) = 2\sigma^2(1-\rho), \qquad \text{relative efficiency} = \frac{1}{2(1-\rho)}$$

the paired-design win; $\rho = 0.98$ gives 25x, and pairing helps at all only
when $\rho > 1/2$.

$$c = \frac{\delta A}{LW}, \qquad \text{signal} \approx \beta \log \frac{c}{D}$$

the planning identity and the scaling law; occurrences must scale with the
trainer's corpus size, and evidence grows only with the logarithm of your own
archive.

$$\alpha' = \frac{\alpha}{m}, \qquad \Pr\Big(\bigcup A_j\Big) \leq \sum \Pr(A_j)$$

Bonferroni and the union bound that proves it, valid under arbitrary dependence;
$m$ counts release-tests, not clients, and must include the stopping rule.

$$C = 6ND$$

the training-cost line, used once, to bury a category of product.
