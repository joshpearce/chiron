## The screen

The screen's honesty rests on a claim nobody states formally: that the four
panels are *not* four estimators of one quantity. Make that precise, because if
they were, disagreement between them would be evidence that at most one is
right, and the whole layout would be a confession.

Fix a model $M$ trained on a corpus $D$ partitioned into sources
$S_1, \dots, S_K$, an index $I$ over some text collection, and a query $q$
producing output $y$. The four panels compute:

$$
c_i(q, I) = \text{(spans of } y \text{ occurring in } I \cap S_i, \text{ rarity-weighted)}
$$

$$
t_i(q, M) = \sum_{\text{checkpoints}} \eta_t \, \nabla \mathcal{L}(y)^\top \nabla \mathcal{L}(S_i)
\qquad
\phi_i = \sum_{S \subseteq N \setminus \{i\}} \frac{|S|!\,(K-|S|-1)!}{K!}\big[v(S \cup \{i\}) - v(S)\big]
$$

$$
z = \frac{k - n p_0}{\sqrt{n p_0 (1 - p_0)}}
$$

where $\eta_t$ is the learning rate at checkpoint $t$, $\nabla \mathcal{L}$ is a
gradient of the loss with respect to the parameters, $v(S)$ is held-out
log-likelihood of a model trained on coalition $S$, $n$ is the number of
detection items, $k$ the hits on the published set, and $p_0$ the hit rate
measured on the never-published controls.

Read the arguments of each function, which is the entire point. $c_i$ is a
function of $(q, I)$ and not of $M$'s parameters at all except through $y$.
$t_i$ is a function of $(q, M)$ and not of $I$. $\phi_i$ is a function of $v$,
which is a function of the whole training procedure, and not of $q$ at all.
And $z$ is a function of the key and the model, with a null that was fixed
before $M$ existed. Four different domains. Nothing forces them to agree, and
a disagreement between $c$ and $t$ is not a contradiction - it is the
statement that $I \cap S_i$ containing $y$'s substrings is a different event
from $S_i$ having moved the parameters.

Notice too that $\phi_i$ carries no $q$. The in-run ledger accumulates over the
whole training run, so the allocation panel is query-independent while the two
above it are query-conditional. That asymmetry is why animating the pie chart
per query would be a category error rather than a UI flourish: it would be
displaying a function of $q$ that does not exist.

## The retrieval panel, built honestly

**The suffix array, costed properly.** Let the tokenized corpus be
$T = t_1 \dots t_n$. The suffix array $A$ is the permutation of $1..n$ such
that the suffix starting at $A[1]$ is lexicographically smallest, and so on. Two
facts do all the work.

First, *occurrences form a contiguous block*. If pattern $P$ has length $m$, the
set of positions where $P$ occurs is exactly $\{A[l], \dots, A[r]\}$ for some
$l \le r$, because every suffix beginning with $P$ sorts adjacently. So counting
is $r - l + 1$, a subtraction.

Second, *finding $l$ and $r$ is binary search*. Each comparison compares up to
$m$ tokens, so naive binary search is $O(m \log n)$. With the LCP (longest
common prefix) array precomputed this falls to $O(m + \log n)$.

Storage: $n$ integers for $A$, at $\lceil \log_2 n \rceil$ bits each if packed,
plus the corpus. At $n = 10^9$ tokens that is 4 bytes per position, 4 GB, which
is why a twelve-source paper corpus fits in memory and a 5-trillion-token web
corpus needs the compressed variants that store the index at a fraction of the
corpus size. Construction is $O(n)$ with a linear-time algorithm, or
$O(n \log n)$ with a sort you would write yourself in an afternoon.

**Maximality, formally.** A matched span $[a, b]$ of the output $y$ is *maximal*
if $y_{a..b}$ occurs in the corpus and neither $y_{a-1..b}$ nor $y_{a..b+1}$
does. Enumerating maximal spans is one left-to-right pass: extend from each
start position until the count drops to zero, emit, advance. The pass is
$O(|y| \cdot (m + \log n))$ with $m$ the longest match, which for a
few-hundred-token answer is milliseconds.

**Rarity, and why it is the right weight.** Let $df(P)$ be the number of
documents containing $P$ and $N_d$ the corpus document count. Weight a span by
something monotone decreasing in $df$, the obvious choice being

$$w(P) = \log \frac{N_d}{df(P)}$$

which is the inverse-document-frequency term from lexical search, and which has
a direct information-theoretic reading: under a bag-of-documents model where a
span is equally likely to be in any document, observing $P$ in a specific
document carries $\log(N_d / df(P))$ nats of identifying information. At
$df = 1$ and $N_d = 10^7$ that is 16.1 nats; at $df = 3140$ it is 8.1. The
nine-token hapax outranks the four-token common phrase by a factor of two in
information, which is the ordering the panel wants.

Now the adversarial reading of the same formula, which is the payment argument
from the canon made precise. If payout is proportional to $\sum_P w(P)$ over
surfaced spans, then a supplier maximizing revenue maximizes
$\log(N_d / df(P))$ subject to $P$ being emittable. The gradient of that
objective with respect to supplier behavior points at *minimizing $df$*, and
$df = 1$ is achieved by writing anything nobody else wrote. The rule pays a
premium of $\log N_d$ nats for the single cheapest property to manufacture in
this market. Compare the honest alternative: a rule paying proportional to
$\phi_i$ has no such gradient, because $\phi_i$ is defined by a counterfactual
that manufactured text cannot move without actually improving held-out
likelihood.

**Why lexical is a lower bound, quantified.** Let $R$ be the set of documents
that lexically match $y$ and $C$ the set with nonzero contributive influence on
$y$. The panel reports $R$. The measurement to have in mind is
$|R \cap C| / |C|$ - the fraction of genuine contributors the panel can see -
and the structural result is that this falls with scale: influence at larger
model scale is documented to become more abstract, matching concepts rather than
tokens, so $C$ grows to include documents sharing no $n$-grams with $y$ while
$R$ does not. There is no correction factor that repairs this, because the
missing documents are missing by construction: $R$ is defined by string
containment and they contain no shared strings.

## Assembling the pipeline

The chain has an error-propagation structure worth writing down, because
"grading a cheap method against ground truth" quietly involves three noise
sources and only one of them is usually reported.

Each coalition value is measured with seed noise. With $\sigma_{\text{run}}$ the
per-run standard deviation of held-out bits per byte and $k$ seeds averaged,
the table entry has

$$\sigma = \frac{\sigma_{\text{run}}}{\sqrt{k}}$$

A leave-one-out contribution is a difference of two such entries, so

$$\text{SE}_\Delta = \sigma\sqrt{2} = \sigma_{\text{run}}\sqrt{\tfrac{2}{k}}$$

and the significance test is $z = \Delta_i / \text{SE}_\Delta$.

A Shapley share is a weighted sum of $2^{K-1}$ such differences with weights
summing to 1, and because the same coalition values are reused across many
terms the errors are correlated rather than independent. The propagated result
at three sources works out to $\text{SD}(\phi_i) = 0.745\,\sigma$ - notably
*below* $\text{SE}_\Delta = 1.414\,\sigma$, because averaging over arrival
orders averages over noise as well as over orderings.

Now the third source, which is the one the pipeline introduces and nobody
reports: **attenuation of the correlation by noise in its target.** The trust
number $\rho$ is a rank correlation between an estimated ranking and a *measured*
ground-truth ranking, and the measured ranking is itself noisy. For the
Pearson analogue the classical correction is explicit: if $x$ is the estimate and
$y^\star$ the true target, observed through $y = y^\star + \varepsilon$, then

$$\text{corr}(x, y) = \text{corr}(x, y^\star) \cdot \sqrt{\text{rel}(y)}
\qquad \text{where} \qquad
\text{rel}(y) = \frac{\text{Var}(y^\star)}{\text{Var}(y^\star) + \text{Var}(\varepsilon)}$$

The reliability $\text{rel}(y)$ is exactly the quantity the SNR measures. At
$\text{SNR} = 1.37$, treating the spread across sources as $\text{Var}(y^\star)$
and the per-share noise as $\text{Var}(\varepsilon)$ gives
$\text{rel} = 1.37^2/(1.37^2 + 1) = 0.65$, so $\sqrt{\text{rel}} = 0.81$. A
measured 0.61 against that target is consistent with a true correlation near
$0.61/0.81 = 0.75$ against a noiseless target.

Two consequences, and both belong in the memo. First, do not silently apply the
disattenuation: report the measured value and the reliability separately, because
the correction assumes a model of the noise and an opposing reader is entitled to
reject the model. Second, and more useful, the correction tells you where to
spend. Raising $k$ raises $\text{rel}$, which raises the *measured* $\rho$ without
the method improving at all. A correlation quoted without its seed count is
therefore not just imprecise, it is unfalsifiable: any number can be reached by
buying seeds.

The rank version is messier - Spearman's attenuation has no clean closed form -
but the direction and the qualitative conclusion are identical, which is why the
canon insists $\rho$ travels with its seed count and source count.

## What the demo claims and what it does not

**The scale caveat, as arithmetic rather than as a hedge.** Two quantities
change with scale and they change in opposite directions for a payout system.

The first is *concentration of influence*. Let $g_i$ be source $i$'s share of
total influence for a query. At small scale, measured influence is top-heavy on
a handful of sources; at frontier scale it is documented to be sparse per query
but drawn from an enormous tail across queries. Model the per-document share as
following a power law with exponent $\alpha$: the share held by the top $r$
fraction of documents behaves like $r^{1 - 1/\alpha}$. At $K = 12$ sources the
question "what is source 7's share" has an answer of order 1/12. At $K = 10^7$
documents, even a strongly concentrated law puts the median document's share
below any collectible amount, and the fixed per-payment cost - the transaction
itself - exceeds the payment. The mechanism does not break; the economics of
disbursement do.

The second is *estimator quality*, which moves the wrong way over the same
range. Against actual retraining on a 2-billion-parameter model, scalable
gradient estimators measured no better than random guessing, while the estimator
reaching $\rho = 0.97$ costs three to five full training runs per test point. So
as $K$ grows and the per-source signal shrinks, the measurement you would need
becomes both more demanding and less available.

Put the two together and the honest boundary is explicit rather than
apologetic. The claim is valid on the region where the counterfactual is
computable: $K$ small enough that $2^K$ coalitions can be trained, and $N$ small
enough that a run is minutes. Everything the artifact says is exact inside that
region and silent outside it, and *the boundary is stated in terms an auditor can
check* - a source count and a parameter count - rather than as a vague
disclaimer.

**Why the honest boundary is a stronger position.** Formalize the comparison
being made in the room. Your claim is a pair (estimate, measured reliability).
The competing claim is a point estimate with no reliability at all, which is
formally the same as a claim with an unbounded prior on its error. Under any
proper scoring rule, a forecast that reports its own uncertainty dominates a
forecast that omits it whenever the audience is scored on calibration rather
than on confidence - and a technical diligence process is precisely a
calibration test. The only regime where the confident claim wins is one where
the audience never checks, and the assumption that the audience never checks is
the assumption that loses the room in this particular market, where the
threshold theorem and the ground-truth measurements are published.

## The economics of the pitch

**Pro-rata versus Shapley, in general form.** Let $u_i = v(\{i\})$ be source
$i$'s solo value. Pro-rata by measured use assigns

$$p_i = \frac{u_i}{\sum_j u_j}$$

Shapley assigns $\phi_i$ as above. The gap between them has a clean
characterization: the two coincide for every value function if and only if $v$
is *additive*, meaning $v(S) = \sum_{i \in S} u_i$ for all $S$. Any departure
from additivity is a departure between the two rules.

On the running table, measure the departure directly. Additivity would predict
$v(AC) = 50 + 20 = 70$; the observed value is 90, so the pairwise interaction is
$\Delta_{AC} = 20$. Likewise $\Delta_{BC} = 90 - 50 - 20 = 20$ and
$\Delta_{AB} = 70 - 50 - 50 = -30$. The three-way residual is what is left after
the singletons and the pairs are accounted for:

$$\Delta_{ABC} = v(ABC) - \big[v(AB) + v(AC) + v(BC)\big] + \big[u_A + u_B + u_C\big]
= 100 - 250 + 120 = -30$$

Shapley splits each interaction evenly among its members - halves for pairs,
thirds for the triple - which is the direct route to the answer:

$$\phi_i = u_i + \tfrac{1}{2}\sum_{j \neq i} \Delta_{ij} + \tfrac{1}{3}\Delta_{ABC}$$

$$\phi_A = 50 + \tfrac{-30}{2} + \tfrac{20}{2} + \tfrac{-30}{3} = 50 - 15 + 10 - 10 = 35$$

$$\phi_C = 20 + \tfrac{20}{2} + \tfrac{20}{2} + \tfrac{-30}{3} = 20 + 10 + 10 - 10 = 30$$

and $\phi_B = 35$ by symmetry with A, summing to 100 as efficiency requires. No
renormalization anywhere: the decomposition is exact.

The lesson is in the middle terms. C's entire advantage under Shapley is the two
positive pairwise interactions it sits in, each worth 20, each split evenly, for
$+20$ on top of its solo 20. A and B share a *negative* interaction of $-30$,
because they substitute for each other, and each carries half of it. Pro-rata
sees none of this, because $u_i$ is all pro-rata ever looks at.

Quantify the misallocation in one number: the ratio

$$\frac{\phi_C}{p_C} = \frac{30}{16.67} = 1.80$$

Pro-rata pays complements at $1/1.8 = 56\%$ of their fair share here. In
general the ratio is bounded by how large the interaction terms are relative to
the solo values, so the pathology scales with how much of your corpus's value is
synergistic - which for a specialist corpus assembled from complementary
subfields is most of it.

**The SNR threshold, stated as the optimization it comes from.** The result
behind the flat-fee collapse is a contracting problem. A principal pays agents
based on a noisy signal $\hat{\phi}_i = \phi_i + \varepsilon_i$ of their
contribution. A contract $w(\hat{\phi})$ trades off two terms: allocative
efficiency, which improves with sensitivity to $\hat{\phi}$, and risk imposed on
risk-averse agents, which worsens with it. The optimal sensitivity is increasing
in the ratio of signal variance to noise variance. Below a threshold in that
ratio the optimal sensitivity hits zero and the contract is constant - a flat
fee - because the marginal allocative gain from responding to the signal is
dominated by the marginal risk cost of responding to noise.

The two structural facts that matter for the pitch. The threshold is a property
of the *signal*, so a better estimator only helps if it reduces
$\text{Var}(\varepsilon)$; it cannot manufacture $\text{Var}(\phi)$. And the
collapse is a corner solution rather than a gradual flattening, which is why
"our SNR is low but the split is directionally right" is not a coherent
position: below the corner the optimal responsiveness is exactly zero, not
small.

The empirical form used in this book,

$$\text{SNR} = \frac{\text{SD across sources of } \phi}{\text{SD}(\phi_i)}$$

is the sample analogue of $\sqrt{\text{Var}(\phi)/\text{Var}(\varepsilon)}$, and
the customary bar of 2 is the same "two standard errors" convention used for
every other claim in the book, applied to the question of whether sources are
distinguishable at all.

To move it, the arithmetic is direct. Since
$\text{SD}(\phi_i) \propto 1/\sqrt{k}$, reaching a target $\text{SNR}^\star$
from a current $\text{SNR}_0$ at $k_0$ seeds requires

$$k \geq k_0 \left(\frac{\text{SNR}^\star}{\text{SNR}_0}\right)^2$$

At $\text{SNR}_0 = 1.37$, $k_0 = 3$ and $\text{SNR}^\star = 2$, that is
$3 \times (2/1.37)^2 = 6.4$, so 7 seeds - and at 12 sources,
$4{,}096 \times 7 = 28{,}672$ runs. The quadratic is the reason this is a real
budget decision rather than a formality: halving the gap costs four times the
compute.

## The strongest case against the whole venture

Treat the venture as a decision under uncertainty over outcomes rather than as a
belief, because that is the form in which the objection is answerable.

Let the branches be $B_1$ (courts hold the current line: acquisition priced
once, training on lawful copies is fair use), $B_2$ (output-substitution or
market-harm theories succeed, ongoing payment arrives), $B_3$ (status quo
persists: mandatory coarse disclosure, no payment mechanism). Let $V(x, B)$ be
the value of building capability $x$ under branch $B$.

For $x = $ royalty exchange: $V = 0$ under $B_1$ (labs buy one clean copy and
owe nothing recurring), high under $B_2$, and $0$ under $B_3$ (no counterparty).
Expected value is $\Pr(B_2)$ times a large number, and $\Pr(B_2)$ is a
litigation outcome you cannot estimate and cannot influence.

For $x = $ measurement and verification: positive under all three. Under $B_1$,
clean-copy purchasing creates a provenance-establishment need on the buy side.
Under $B_2$, an allocation needs an auditor and the allocation needs to be
defensible. Under $B_3$, disclosure is mandatory, coarse, self-reported and
explicitly unverified by the regulator's own written statement, with enforcement
live and fines up to 3% of global revenue.

The structural point is not that verification has higher expected value - you
cannot compute either expectation honestly. It is that verification's value is
*bounded below by a positive number across the whole branch set*, while the
exchange's is bounded below by zero. That is a dominance argument, not an
optimism argument, and it is the form in which the objection cannot be answered
by disputing anyone's probability estimates.

There is a fourth branch worth naming because it is the one that makes the
objection sharpest: $B_4$, the gatekeeper vertically integrates verification
too. It already holds the chokepoint, the identity standard, the preference
standard, the payment rail and a marketplace. The honest response is that
verification is the one layer where being the measured party's vendor is a
conflict rather than an advantage - an auditor owned by the party whose traffic
is being audited is not an auditor - and that this is the single structural
reason independence has value here. It is also the reason the pre-commitment
ledger rather than the statistics is the asset: an audit's credibility is a
property of who holds the commitment, not of who computes the z-score.

## What to build next

The four extensions are ordered by mathematical prerequisite depth, and the
ordering is inverse to their empirical support - which is the most useful
structural fact in this space and worth stating as such.

Retrieval needs data structures and logarithms. Canaries and the notary need
expectation, variance, and a binomial $z$-test. Shapley needs combinatorics and
expectation over permutations, with no calculus anywhere. TracIn needs partial
derivatives, gradients and dot products. Datamodels and TRAK need
Johnson-Lindenstrauss projections and least squares. Influence functions need
second-order Taylor expansion, Hessians, the implicit function theorem, and
Kronecker-factored eigendecomposition.

Now the empirical ordering. The method with the strongest published evidence
against alternatives is source-level Shapley over an exactly enumerable
coalition set. The method with the strongest legal standing is the keyed
watermark with a constructible null. The method with four independent negative
results against it is influence functions, which is also the one requiring
months of second-order calculus - and the theoretical reason for the negative
results is instructive: the classical derivation requires a twice-differentiable,
strictly convex loss evaluated at an exact optimum, and a language model
satisfies none of the three. A student of Grosse's showed that what the
estimator actually computes is a proximal Bregman response function rather than
leave-one-out, which is not an approximation error to be tightened but a
different quantity.

So the extension you would reach for by mathematical ambition is the one to
reach for last, and the reason is not pedagogical caution. It is that the
mathematics is doing work that the problem's structure does not support.

## At the bench: the demo and the memo

**Sizing the index.** With $n$ corpus tokens the suffix array costs
$4n$ bytes at 32-bit positions ($n < 2^{32}$), plus the token array at 2 or 4
bytes per token depending on vocabulary size, plus the LCP array at another $4n$
if you want $O(m + \log n)$ lookups rather than $O(m \log n)$. For a
500-million-token paper corpus with a 32K vocabulary: 2 GB for the array, 1 GB
for the tokens at 16 bits, 2 GB for LCP. Three to five gigabytes, resident, on a
laptop. Skip the LCP array first if memory is tight; at $m$ in the tens and
$\log_2 n \approx 29$ the difference is a small constant factor on an operation
already measured in microseconds.

**The uncertainty-field rule, made checkable.** The canon's rule that no panel
renders without its uncertainty field is a type constraint. Model each panel's
payload as a pair rather than a value: $(\text{estimate}, \text{uncertainty})$,
with the second component non-nullable. Panel 2's pair is
$(\text{ranking}, (\rho, k, K))$; Panel 3's is
$(\phi, (\text{SD}(\phi_i), \text{SNR}, \text{verdict}))$; the footnote's is
$(\text{decision}, (z, m, p_{\text{corrected}}, \text{commitment hash}))$;
Panel 1's is $(\text{spans}, \text{exact})$, where `exact` is a distinguished
value meaning the panel introduces no noise of its own. Making `exact` an
explicit inhabitant of the uncertainty type rather than a null is the whole
trick: it forces the one panel that genuinely has no error bar to say so, rather
than letting missing data and zero error look identical to the renderer.

**Family size in the memo, correctly.** The footnote's corrected p-value is
$m \times p$ with $m = 25$, the number of distinct watermark entities. Note what
$m$ is *not*: it is not the number of detection questions, because four
attributes of one invented entity are not independent tests - a model that
learned the entity tends to get several. Treating the entity rather than the
attribute as the unit is the conservative choice and it is the one an opposing
expert cannot attack. Nor is $m$ the number of clients audited this quarter,
which would be the correct family for a *service-wide* false-positive claim.
State which family the correction covers, in the report, in one sentence. Two
different corrections for two different claims is honest; one correction
silently doing duty for both is the error.

## What you can now do

One piece of arithmetic worth carrying, because it compresses the whole unit
into a decision procedure.

Given a claim about training data with an attached number, ask what its
derivative is with respect to four interventions: rebuilding the index,
retraining the model, changing the value function, and doing nothing after a
commitment. Exactly one of those derivatives should be nonzero for a
well-formed claim, and which one it is names the claim's type:

$$\frac{\partial}{\partial I} \neq 0 \Rightarrow \text{corroborative}
\qquad
\frac{\partial}{\partial M} \neq 0 \Rightarrow \text{contributive}$$

$$\frac{\partial}{\partial v} \neq 0 \Rightarrow \text{allocation}
\qquad
\text{all four} = 0 \text{ after commitment} \Rightarrow \text{proof}$$

A claim with two nonzero derivatives is a claim that has conflated two panels,
and the conflation is where every misallocation in this market lives. That is
the formal version of the diagnostic question - what experiment would change
this number - and it is the only piece of mathematics in this unit that you will
use in a room rather than at a desk.
