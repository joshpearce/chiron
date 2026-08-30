## A 402, three cheques, and one answer to pay for

The pool split deserves more algebra than canon gives it, because two of its
properties cause every fight that follows.

Write the rule again, with $w_i$ the weight of source $i$, $W = \sum_j w_j$ the
total weight, and $P$ the pool in dollars:

$$\text{payout}_i = \frac{w_i}{W} P$$

**Property 1: your payout is a function of everyone else's weight.** Take the
partial derivative with respect to some other source's weight $w_k$, $k \neq i$.
Since $W$ contains $w_k$:

$$\frac{\partial\, \text{payout}_i}{\partial w_k} = -\frac{w_i}{W^2} P < 0$$

Strictly negative. Every source is a negative externality on every other source,
always, by construction. This is not a flaw in a particular marketplace; it is
what "fixed pot" means. A concrete version: if a new source enters with weight
equal to the existing total, $W$ doubles and every incumbent's payout is halved,
with no incumbent having changed anything. Contrast a metered scheme, where
payout is $\text{price} \times \text{units}_i$ and the derivative with respect to
$w_k$ is exactly zero. Metering is not fairer than pooling, but it is *local*, and
locality is why disputes under metering are between two parties while disputes
under pooling are between everyone.

**Property 2: a pooled split of a power-law weight pays a power law.** Suppose
weights follow Zipf's law over $n$ sources: the source ranked $k$ has weight
proportional to $1/k$. Then

$$s_k = \frac{1/k}{\sum_{j=1}^{n} 1/j} = \frac{1}{k \, H_n}$$

where $H_n = \sum_{j=1}^n 1/j$ is the $n$-th harmonic number, well approximated
by $\ln n + 0.5772$. For $n = 1000$: $H_{1000} \approx \ln 1000 + 0.577 = 6.908 +
0.577 = 7.485$.

- Rank 1 share: $1 / 7.485 = 0.134$, so 13.4% of the entire pool to one source.
- Rank 1000 share: $1 / (1000 \times 7.485) = 1.34 \times 10^{-4}$.
- On the $42.5M pool: $5.68M to the top source, $5,679 to the last.

That is a ratio of exactly $n = 1000$ between first and last, and it came from
nothing but the shape of the weights. No allocation rule that normalizes a
skewed weight can produce an unskewed payout. If you want the tail to receive
more than $5,679, the fix has to be outside the proportionality rule - a floor, a
minimum, a separate tail pool - and it has to be paid for by taking from the head,
which is a political act, not a measurement.

**Property 3: the equal split is the $w_i = $ constant special case.** Flat fee is
not a different family of rule. It is the same rule with every weight set to 1,
giving $s_i = 1/n$. This matters for unit v5, where the question "is a
contribution-proportional split better than a flat fee?" becomes the question "is
the measured $w$ vector distinguishable from the constant vector?" - a
statistical question with a computable answer, not a philosophical one.

## The gatekeeper

Origin blocking has an exposure model, and the model is one line.

Let your content exist on $k$ hosts: your origin plus $k - 1$ syndicated or
mirrored copies. Suppose a corpus builder crawls any given host with probability
$q$, independently. The probability that at least one copy lands in the corpus is

$$P(\text{captured}) = 1 - (1 - q)^{k}$$

Blocking your own origin removes exactly one term:

$$P(\text{captured} \mid \text{origin blocked}) = 1 - (1 - q)^{k - 1}$$

Put numbers on it. With $q = 0.3$ and $k = 10$ (one origin, nine copies - modest
for a wire pickup carried by 300 outlets):

- Unblocked: $1 - 0.7^{10} = 1 - 0.0282 = 0.972$
- Blocked: $1 - 0.7^{9} = 1 - 0.0404 = 0.960$

Blocking moved capture probability from 97.2% to 96.0%. The delta is 1.2
percentage points, and it shrinks geometrically in $k$: the marginal value of
blocking one host is $q(1-q)^{k-1}$, which for $q = 0.3$ and $k = 10$ is
$0.3 \times 0.0404 = 0.012$. Note the shape - the derivative is largest when $k$
is small. Origin blocking is genuinely effective for content that exists in
exactly one place, and asymptotically worthless for content that syndicates. That
single expression is the whole of D11, and it tells you where a leakage product's
value is: not in blocking harder, but in measuring $k$.

Second piece of arithmetic, on the gatekeeper's own stated motivation for moving
from Pay Per Crawl to Pay Per Use. More than half of AI crawler requests re-fetch
unchanged content. If a fraction $r$ of requests are redundant, then a per-fetch
price $p$ collects $p \cdot N$ on $N$ requests covering only $(1 - r)N$ distinct
content units, so the effective price per distinct unit is $p / (1 - r)$. At
$r = 0.5$ the buyer pays twice the headline rate per unit of actual content, and
the seller's revenue is proportional to the buyer's crawl inefficiency rather than
to anything the seller produced. Any per-fetch scheme has this property, and it is
why the buyer side eventually refuses it.

## What the deals actually buy

Three quantities worth computing rather than repeating.

**The growth rate of access deals.** Attribution and live-access deals went
2 (2023) to 18 (2025) to about 34 (projected 2026). Over three years the compound
annual growth rate is

$$\left(\frac{34}{2}\right)^{1/3} - 1 = 17^{1/3} - 1 = 2.571 - 1 = 1.571$$

about 157% per year. Total deals over the same window went from 12 to about 36, a
CAGR of $(36/12)^{1/3} - 1 = 3^{1/3} - 1 = 0.442$, about 44%. The access category
is growing roughly three and a half times faster than the market that contains
it, which is the precise statement of "the mix is shifting" and the number to
quote rather than the vague version.

**What the 40% is a fraction of.** With around 91 public deals since 2023 and an
industry rule of thumb of 50 to 100 private deals per public one, total deals are
somewhere in $[4{,}600,\ 9{,}200]$. Applying the 40% training-rights share gives
roughly 1,800 to 3,700 deals that include training rights - against an estimated
$75-100M per year of total private training-rights spend. Dividing: an average
training-rights deal is on the order of

$$\frac{\$100\text{M/yr} \times 3\ \text{yr}}{3{,}000\ \text{deals}} \approx \$100{,}000$$

per deal, order-of-magnitude. The headline numbers in the canon table are the
extreme upper tail of a distribution whose body is small enough that the
transaction cost of negotiating a deal is a material fraction of the deal.

**The ratio that ends the argument.** Mercor at roughly $2B annualized against
$75-100M per year of training-rights licensing is a ratio of 20 to 27. One
expert-data vendor moves an order of magnitude more money per year than the
entire training-rights market, and the gap is growing on both sides.

## What the courts priced

The lab's build-versus-buy decision after *Bartz* is a two-line calculation, and
it is worth doing because it shows why the settlement is not the beginning of a
royalty regime.

Let $N$ be the number of works, $c$ the one-time clean-copy acquisition cost per
work, and $r$ a hypothetical annual royalty per work. One-time purchase costs
$Nc$ forever. A royalty costs $Nr$ per year, so the purchase pays for itself in

$$T = \frac{c}{r}\ \text{years}$$

At the settlement's implied $c = \$3{,}000$ per work (which is punitive - it
prices a statutory-damages exposure, not a retail book), a royalty would have to
exceed $c / T$ to compete. For the purchase to be the wrong choice over a
five-year horizon, the royalty would have to be below $\$600$ per work per year;
for a ten-year horizon, below $\$300$. Both are far above any per-work royalty
anyone has proposed. And the real acquisition cost of a lawfully purchased book is
not $3,000; it is the cover price plus scanning, on the order of $30. At $c = 30$
the purchase beats any royalty above $\$3$ per work-year over a ten-year horizon.

The conclusion is not "royalties are impossible." It is that under current case
law the *substitute* for a royalty is cheap and legal, which caps what any
royalty can charge at approximately the amortized retail price of a copy. A
royalty market can only exist above that cap if something other than copyright
acquisition - output substitution, market harm, a regulatory obligation - creates
the demand. That is the whole strategic content of *NYT v. OpenAI*.

## Contributive and corroborative

The two quantities are functions of different arguments, and writing their
signatures side by side settles most arguments about them.

Contributive attribution is a function of the training set. Let $S$ be a set of
sources, and let $v(S)$ be the utility of a model trained on exactly $S$ -
concretely, in this book, the negative held-out loss, so that higher is better.
The leave-one-out contribution of source $i$ to the full set $N$ is

$$\text{LOO}_i = v(N) - v(N \setminus \{i\})$$

Its arguments are a corpus and a training procedure. No query, no output, no
retriever appears anywhere in it.

Corroborative attribution is a function of a query, an index, and an output. For
a generated sentence $y$ and a document $d$ retrieved from index $I$ in response
to query $q$, the score is some $\text{sup}(d, y)$ - entailment, embedding
similarity, or the retriever's own ranking score, computed over the retrieved set
$R(q, I)$. Its arguments contain no training corpus at all. Change $I$ and every
number changes; change the training corpus and none of them need to.

Now the failure that motivates unit v5, made precise. Suppose sources 1 and 2 are
exact duplicates: everything source 1 teaches, source 2 also teaches. Take the
smallest possible utility function that expresses this, with $v$ measured in
arbitrary utility units:

$$v(\emptyset) = 0, \quad v(\{1\}) = 1, \quad v(\{2\}) = 1, \quad v(\{1,2\}) = 1$$

Leave-one-out on the full set:

$$\text{LOO}_1 = v(\{1,2\}) - v(\{2\}) = 1 - 1 = 0, \qquad \text{LOO}_2 = 0$$

Both sources measure as worthless, and the sum of the measured contributions is 0
while the total value is 1. Leave-one-out is not *efficient*: the parts do not add
up to the whole whenever sources are redundant, and redundancy is the normal case
on the web.

The repair is to stop measuring each source against the full set and average its
marginal contribution over every coalition it could join. With two sources there
are two orderings. In ordering $(1, 2)$, source 1 arrives first and adds
$v(\{1\}) - v(\emptyset) = 1$, source 2 arrives second and adds
$v(\{1,2\}) - v(\{1\}) = 0$. In ordering $(2, 1)$, the reverse. Averaging over the
two orderings gives each source $(1 + 0)/2 = 0.5$, and now the shares sum to
$v(\{1,2\}) = 1$. That average is the Shapley value, unit v5 derives it from
axioms rather than from an example, and this two-source case is the smallest
demonstration of why the leave-one-out number in this unit's opening example is
not the final word on what S1 deserves.

One caution that unit v4 makes quantitative: everything above assumes $v$ is
measured without error. Each $v(S)$ is one training run, subject to seed noise,
and $\text{LOO}_i$ is a difference of two noisy quantities. If each $v(S)$ has
standard deviation $\sigma$ and the runs are independent, the difference has
standard deviation $\sigma\sqrt{2}$. Differences are noisier than the quantities
they are computed from - always, by a factor of about 1.41 - which is why a
contribution can be smaller than the noise on the loss it was derived from.

## Three ways to pay, and what each one misallocates

Pro-rata and user-centric allocation differ in exactly one place, and writing both
out makes the cross-subsidy visible as arithmetic rather than as a complaint.

Let $c_{u,i}$ be user $u$'s consumption of source $i$ (streams, citations,
grounded answers), and let $p_u$ be what user $u$ paid.

**Pro-rata**: pool everything, split by global share.

$$\text{payout}_i^{\text{pro}} = \left(\sum_u p_u\right) \cdot \frac{\sum_u c_{u,i}}{\sum_u \sum_j c_{u,j}}$$

**User-centric**: split each user's own fee among that user's own sources.

$$\text{payout}_i^{\text{uc}} = \sum_u p_u \cdot \frac{c_{u,i}}{\sum_j c_{u,j}}$$

Worked example, two users and two sources, each user paying $10, total pool $20.
User A consumes source 1 only, 100 units. User B consumes source 2 only, 10 units.

Pro-rata: total consumption is 110, source 1's global share is $100/110 = 0.909$,
so source 1 receives $\$18.18$ and source 2 receives $\$1.82$. User B paid $10 and
$8.18 of it went to a source user B never touched.

User-centric: user A's $10 goes entirely to source 1, user B's $10 goes entirely
to source 2. Payouts are $10 and $10.

The two rules disagree by $8.18 on a $20 pool - 41% of the money - from a two-user
example. The disagreement is driven entirely by the variance in consumption
*volume* across users: pro-rata weights each user's preferences by how much that
user consumed, so heavy users vote with everyone's money. In music this
redistribution empirically moves only 1 to 5 percentage points toward
independents, because real consumption distributions are less extreme than this
example. In an AI marketplace, where one enterprise customer may issue a million
queries and another a hundred, the volume variance is far larger than in music,
and the cross-subsidy is correspondingly worse.

**Farming, as an expected-value calculation.** In a fixed pool $P$ with total
countable weight $C$, an attacker who manufactures $m$ additional units of weight
receives

$$\text{payout}_{\text{attacker}} = P \cdot \frac{m}{C + m}$$

With $P = \$42.5$M and $C = 10$M citations, manufacturing $m = 1$M citations
returns $42.5 \times 1/11 = \$3.86$M. The costs are generation (near zero for
plausible text), hosting across many domains, and passing enrollment. At a budget
of $50,000 the return on a successful attack is roughly 77x. Note also the
denominator effect from the first section: the attacker's million units dilute
every legitimate source by a factor $10/11$, so honest participants fund the
attack whether or not it is detected. Any pooled scheme with a cheaply
manufacturable weight has this structure, and the defense has to be an identity
or admission constraint, because the payout formula itself offers no resistance.

## The strongest case against all of this

Make the two bets explicit and the choice stops being rhetorical.

Let branch A be "a training-royalty market appears" and branch B be "it does not,
and the money stays in access deals, expert data, and compliance." Assign branch A
subjective probability $\pi$; you can pick your own number, the structure does not
depend on it.

A payment-layer product pays off in branch A and is worthless in branch B, so its
expected value is $\pi V_A$. A measurement-layer product - can you say, with a
stated error rate, what a model was trained on - is needed in branch A (a royalty
split nobody can audit is not a royalty split) and in branch B (disclosure
obligations, procurement diligence, leakage claims), so its expected value is
$\pi V_A' + (1 - \pi) V_B'$.

The measurement bet dominates unless $V_A$ exceeds $V_A'$ by enough to overcome
the entire $(1 - \pi) V_B'$ term. For that to hold you have to believe both that
the royalty market appears *and* that owning the payment rail is worth far more
than owning the measurement that the rail depends on. Set $\pi = 0.3$, $V_A = 10$,
$V_A' = 4$, $V_B' = 4$: the payment bet is $3.0$, the measurement bet is
$0.3(4) + 0.7(4) = 4.0$. The numbers are illustrative and yours will differ; the
structure is not illustrative. A product that is needed under both branches beats a
larger product needed under one, unless the ratio is extreme.

This is also why the unit refuses to resolve the skeptical case by predicting the
branch. Predicting it is not required. The dominance argument goes through without
it.

## At the bench: what counts as one source

The combinatorics that set $n$ at 10 to 12 rather than 100 are worth doing
exactly, because they are the reason source-level attribution is affordable at
all.

The number of subsets of $n$ sources is $2^n$. Every subset is a coalition whose
utility $v(S)$ requires one training run:

| $n$ | $2^n$ coalitions | at 6 min/run, 1 seed | at 3 seeds |
| --- | --- | --- | --- |
| 8 | 256 | 26 hours | 3.2 days |
| 10 | 1,024 | 102 hours | 12.8 days |
| 12 | 4,096 | 410 hours | 51 days |
| 15 | 32,768 | 3,277 hours | 1.1 years |

Twelve sources with a single seed is a long weekend on borrowed compute; twelve
with three seeds is not a laptop project, and fifteen is nobody's project. The
practical envelope is $n \approx 10$ with seeds, or $n = 12$ with sampling. Unit
v5 shows how to shrink this with Monte Carlo estimation over permutations and with
caching - every coalition is trained once and reused across the Shapley, Banzhaf
and leave-one-out computations, since all three are different weighted sums over
the *same* utility table.

Two structural facts to carry into that unit. First, the number of coalitions
containing any particular source $i$ is $2^{n-1}$ - exactly half. Every source's
value is computed from half the table, and the two halves pair up into $2^{n-1}$
marginal contributions $v(S \cup \{i\}) - v(S)$. Second, the training cost of a
coalition is not constant: a coalition holding all 12 sources trains on the full
token budget while a singleton trains on roughly a twelfth of it, so if you hold
the number of training *steps* fixed the runs cost the same but see different data
volumes, and if you hold *epochs* fixed they cost wildly different amounts. That
choice is a design decision with consequences for $v$, and unit v4 makes it
explicitly rather than letting the training script make it silently.

Finally, the reason per-paper attribution fails is not only that $2^{100{,}000}$
is absurd. Even with a sampling estimator whose cost is independent of $n$, the
per-item payout is the problem: a $12,000 pool over 100,000 papers averages $0.12
per paper, which is below the cost of the bank transfer. The attribution unit and
the payment unit have to be the same object, and the payment unit has a hard floor
set by transaction cost, not by mathematics.
