## Two transcripts, one model

Canon says duplication drives the difference between the two transcripts, and
that the relationship is log-linear. Log-linear is a strong claim and it is
worth writing down, because it is what lets you predict a transcript you have
not run.

Let $k$ be the number of near-duplicate copies of a document that pass through
training, and let $e(k)$ be the fraction of a fixed-length span that comes back
under a fixed attack. Log-linear means

$$e(k) = a\,\log_2 k + b$$

for constants $a$ and $b$ that depend on the model, the corpus size, and the
attack. Read the form before the numbers: each *doubling* of $k$ buys a constant
increment $a$, not a proportional one. Going from 4 copies to 8 buys the same
increment as going from 512 to 1024.

Fit it to two measurements. Suppose a document present in 8 copies returns 5% of
a 200-token span, and one present in 512 copies returns 35%. Then
$\log_2 8 = 3$, $\log_2 512 = 9$, and

$$a = \frac{0.35 - 0.05}{9 - 3} = 0.05, \qquad b = 0.05 - 0.05(3) = -0.10$$

so $e(k) = 0.05 \log_2 k - 0.10$. Three consequences fall straight out.

**The threshold.** $e(k) = 0$ at $\log_2 k = 2$, that is $k = 4$. Below four
copies this model, under this attack, returns nothing at all. Transcript two's
paper had two copies. It was never going to fire, and you could have known that
before running it.

**The ceiling is expensive.** To reach $e = 0.5$ you need
$\log_2 k = (0.5 + 0.10)/0.05 = 12$, so $k = 4096$ copies. Half of a span
requires a thousand times the duplication of the detection threshold. This is
the shape of the whole subject: the first bit of evidence is cheap and every
subsequent bit costs a doubling.

**Interpolation is safe, extrapolation is not.** The fit is over the range you
measured. The functional form has no mechanism behind it - it is a regularity
observed across models and corpora, not a derivation - and it must break at both
ends, since $e$ is bounded in $[0,1]$ and the formula is unbounded. Report the
range you fitted over, always.

The same log-linear form holds, with different constants, in model parameter
count and in prompt length. Three log-linear terms means that in the region
where all three matter you can write

$$e \approx a_1 \log_2 k + a_2 \log_2 N + a_3 \log_2 L + b$$

with $N$ the parameter count and $L$ the prompt length in tokens, and read off
the substitution rate directly: a doubling of duplication is worth
$a_1 / a_3$ doublings of prompt length. That ratio is the only thing you need
when deciding whether to spend an experiment budget on longer prompts or on a
more duplicated target.

## What memorization is, and what it is not

**Probabilistic extraction, stated properly.** Fix a document, a prefix, and a
target span. Let $q$ be the probability that one sample from the model at a
stated temperature reproduces the span. Draw $n$ independent samples. The
probability that at least one reproduces it is

$$P(\text{at least one in } n) = 1 - (1 - q)^n$$

because the samples are independent and $(1-q)^n$ is the probability that all
$n$ miss. This is the whole of $(n,p)$-discoverable extraction: a document is
$(n,p)$-extractable if $n$ samples recover it with probability at least $p$.

Numbers. At $q = 0.01$ and $n = 100$:

$$1 - 0.99^{100} = 1 - 0.366 = 0.634$$

A document that greedy decoding never produces, and that you would have written
down as "not memorized," comes out roughly two times in three when you sample a
hundred times. Invert it to plan an experiment: to reach $p = 0.95$,

$$n = \frac{\ln(1 - p)}{\ln(1 - q)} = \frac{\ln 0.05}{\ln 0.99} = \frac{-3.00}{-0.01005} = 298$$

so 300 samples. That is a cheap experiment, and it converts a binary
non-finding into a measured $q$.

Report $q$ with an interval, since it is a binomial proportion estimated from
$n$ trials. With $\hat q = m/n$ successes, the standard error is

$$\text{SE}(\hat q) = \sqrt{\frac{\hat q (1 - \hat q)}{n}}$$

At $\hat q = 0.01$ from $n = 300$: $\sqrt{0.01 \times 0.99 / 300} = 0.0057$. So
the estimate is $1\% \pm 1.1\%$ at two standard errors, an interval that
includes zero. Three successes out of 300 is not a rate; it is a hint that a
rate exists. The normal approximation is poor this close to zero and a
Wilson interval is the right tool, but the lesson survives either way: a
handful of successes needs more samples before it is a number.

**Counterfactual memorization, and why it is unaffordable.** The definition

$$\text{cm}(x) = \mathbb{E}_{\text{with } x}[r(x)] - \mathbb{E}_{\text{without } x}[r(x)]$$

is a difference of two expectations over training runs, and every expectation
has to be estimated by a sample mean over actual runs. With $k$ seeds in each
condition and per-run standard deviation $\sigma_r$ in the recall score, the
standard error of the difference is

$$\text{SE}(\widehat{\text{cm}}) = \sigma_r \sqrt{\frac{2}{k}}$$

- the same expression v4 used for a leave-one-out contribution, for the same
reason: a difference of two independent means has variance equal to the sum of
their variances, and each of those is $\sigma_r^2 / k$.

Now price it. To resolve a counterfactual memorization of 0.1 at $z = 3$ when
$\sigma_r = 0.15$, you need

$$3 = \frac{0.1}{0.15\sqrt{2/k}} \implies \sqrt{2/k} = 0.222 \implies k = 40$$

Forty training runs, per document. That is the entire reason nobody computes
this quantity above toy scale, and the reason every reported memorization number
is definition one or definition two. It is also the reason the definition is
still worth knowing: it tells you what the cheap definitions are approximating,
and therefore which of their failures are bias and which are noise.

## The arithmetic of what can be stored

**The counting argument, without hand-waving.** Let $\mathcal{C}$ be the set of
possible corpora and $\mathcal{W}$ the set of distinguishable weight
configurations. Training is a map $T : \mathcal{C} \to \mathcal{W}$. If
$|\mathcal{C}| > |\mathcal{W}|$ then $T$ cannot be injective, so there exist
distinct corpora $c_1 \neq c_2$ with $T(c_1) = T(c_2)$, and no procedure
whatsoever can recover which one was used. This is pigeonhole, and it is
independent of architecture, optimizer, and training duration.

With $N$ parameters each carrying $b$ bits about the data,
$|\mathcal{W}| \leq 2^{Nb}$. A corpus with $H$ bits of information gives
$|\mathcal{C}| \approx 2^{H}$. The condition for non-recoverability is
$H > Nb$, which is the ratio canon computes.

Two subtleties the plain statement hides.

**The bound is about the whole corpus, and only about the whole corpus.** It
says the map cannot be injective on the full space. It says nothing about
whether the map is injective when restricted to a small subset - and a corpus
containing one document repeated 4,000 times is, informationally, a small
subset. This is exactly how both facts in section 1 coexist: the aggregate is
unrecoverable and specific high-duplication items are recoverable, with no
contradiction.

**Why $b$ is 3.6 and not 16.** A parameter stored in bfloat16 occupies 16 bits
of memory, but the question is how many bits about the *training data* it
carries, which is a different quantity. Most of a parameter's precision encodes
its position in a continuous optimization landscape where nearby values give
nearly identical functions - variation that is not distinguishing between
corpora. The measurement isolates the data-bearing part by training on datasets
of uniformly random strings, which contain no generalizable structure at all, so
every bit the model reproduces must have been stored rather than inferred.
Increase the random dataset until the model stops absorbing more, divide by
parameter count, and the quotient is $b$. It comes out near 3.6 across model
sizes, which is why it is usable as a constant.

**Where the capacity actually goes.** Split the total into

$$C_{\text{mem}} = C_{\text{general}} + C_{\text{specific}}$$

with $C_{\text{general}}$ the bits spent on structure shared across many
documents - orthography, syntax, common facts, register - and
$C_{\text{specific}}$ the residue tied to individual documents. Nothing measures
the split directly, but its existence is what makes the capacity bound
conservative: the corpus-specific budget is strictly less than $C_{\text{mem}}$,
so a ratio of 1 does not mean "half could be memorized," it means "less than
half, by an unknown margin."

**Relative frequency, derived rather than asserted.** Suppose the training
objective's pressure to memorize a specific string competes against the pressure
to generalize, and that the competition is settled by how much of the gradient
signal that string commands. A document appearing $k$ times in a corpus of $D$
tokens contributes a fraction proportional to $k/D$ of the updates. Hold $k/D$
fixed and you hold the competition fixed; that is the Hubble result restated.
So the invariant is

$$\frac{k_1}{D_1} = \frac{k_2}{D_2} \implies k_2 = k_1 \frac{D_2}{D_1}$$

which is the multiplication the compute beat performs. Two caveats keep this
honest: the argument is a plausibility sketch rather than a derivation, and the
empirical result it matches was measured at 1B and 8B parameters over 100B to
500B tokens, so extrapolating it to a trillion-token corpus is an assumption
being made explicit rather than a fact being reported.

## Extraction as evidence

**Discoverable memorization bounds extractable memorization.** Let $A_d$ be the
event that the true prefix elicits the true continuation, and $A_e$ the event
that some adversary-constructed prompt elicits it. Any adversary who possesses
the prefix can use it, so $A_d \subseteq A_e$ would hold if the adversary had
the document - but the adversary does not, which is the point of the
distinction. In practice, measured over a corpus,

$$\text{rate}_{\text{extractable}} \leq \text{rate}_{\text{discoverable}}$$

because discoverable measurement hands the attack the optimal prompt for free.
The gap between the two is the adversary's search problem, and the divergence
attack's contribution was to shrink that gap by two orders of magnitude without
any per-document knowledge.

**Why prompt length helps, in one line of information theory.** The model's
uncertainty about the next token given a prefix of length $L$ is a conditional
entropy $H(x_{L+1} \mid x_1, \ldots, x_L)$. Conditioning cannot increase
entropy, so this is non-increasing in $L$. A longer prefix therefore puts more
probability mass on the true continuation, mechanically, before any memorization
enters the story. That is also the trap: a long prefix makes extraction easier
for reasons that have nothing to do with whether the document was trained on,
which is why any extraction rate reported without its prompt length is
uninterpretable and why comparisons must hold $L$ fixed.

**Reporting a corpus-level rate.** If you test $n$ documents and $m$ fire, the
point estimate is $m/n$ and the interval is binomial as above. But the documents
are not exchangeable - firing probability varies by orders of magnitude across
documents - so $m/n$ estimates the mean of a wildly skewed distribution, and the
mean of a skewed distribution is a poor summary. Report the rate together with
the count and the identity of the firing documents. A rate of 1% built from 3
documents out of 300 and a rate of 1% built from 300 documents out of 30,000 are
different findings, and only the second is a rate in any useful sense.

## Why membership inference cannot prove training

Canon gives the argument in words. Here it is in the language of hypothesis
testing, because that is the form in which the impossibility is unambiguous.

You have a test statistic $T$ computed from the model and the document - a loss,
a rank statistic, a calibrated ratio, it does not matter which. You choose a
threshold $t$ and declare membership when $T < t$. The false positive rate is

$$\text{FPR}(t) = P\big(T < t \;\big|\; H_0\big)$$

where $H_0$ is the null hypothesis "this model was not trained on this
document." Every word of that expression is a requirement. To evaluate it you
need the distribution of $T$ under $H_0$, and $H_0$ is a statement about the
*model*, not about the document. The random object whose distribution you need
is the training run.

Write that distribution explicitly. Let $\theta \sim \mathcal{T}(\mathcal{D})$
denote parameters produced by the training procedure on corpus $\mathcal{D}$,
with the randomness coming from initialization, data order, and hardware
nondeterminism. Then

$$\text{FPR}(t) = P_{\theta \sim \mathcal{T}(\mathcal{D} \setminus \{x\})}\big(T(\theta, x) < t\big)$$

To sample this you must draw $\theta$ from $\mathcal{T}(\mathcal{D}\setminus\{x\})$,
which means running the training procedure on the corpus with $x$ removed. You
need $\mathcal{D}$, you need $\mathcal{T}$, and you need to run it repeatedly.
There is no estimator of this quantity that avoids the draw, because the
quantity is defined as a probability under that draw.

**Neyman-Pearson makes it worse, not better.** The most powerful test at a given
FPR is the likelihood ratio

$$\Lambda = \frac{P(\text{observation} \mid H_1)}{P(\text{observation} \mid H_0)}$$

so the optimal attack requires the null likelihood in its denominator. Better
attacks are attempts to approximate $\Lambda$; every one of them needs the same
unavailable object. This is why improvements in attack design do not converge on
a proof: they are refining the numerator.

**The substitution, formalized.** The universal workaround replaces the model
distribution with a document distribution: take non-member documents
$x'_1, \ldots, x'_m$, compute $T(\theta, x'_j)$ on the *same* model, and use
their empirical spread as $\hat{F}_0$. This is valid if and only if

$$T(\theta, x'_j) \stackrel{d}{=} T(\theta', x) \quad \text{where } \theta' \sim \mathcal{T}(\mathcal{D}\setminus\{x\})$$

that is, if varying the document across non-members reproduces the distribution
you would get by varying the training run with the document removed. Nothing
guarantees this. It is an exchangeability assumption, and it is the whole
scientific content of every MIA benchmark ever published.

**Why the confound is fatal rather than reducible.** Suppose $T$ depends on both
membership $M \in \{0,1\}$ and a covariate $Z$ (date, topic, register):

$$T = f(M, Z) + \varepsilon$$

In the standard benchmark construction, $Z$ and $M$ are perfectly correlated -
every member is pre-cutoff, every non-member is post-cutoff. When two predictors
are perfectly collinear, no procedure can separate their coefficients; the data
contains no information about which one is doing the work. A better estimator
does not help. A larger sample does not help. Only breaking the collinearity
helps, and breaking it means constructing a control set matched on $Z$, which is
a data-collection problem you may or may not be able to solve.

The blind-classifier result is the empirical proof of collinearity, and it is
elegant: a classifier restricted to $Z$ alone, with no access to $\theta$ at
all, achieved higher accuracy than classifiers with access to both. If
$f(1, Z) - f(0, Z)$ were carrying the discrimination, that could not happen.

**The seed-noise result, in the same notation.** Even granting a valid null,
$T(\theta, x)$ has a distribution over $\theta$ within the member condition too.
If those two distributions - member and non-member - overlap to the point where
a single draw is uninformative, the per-document decision is a coin flip
regardless of how the threshold is set. This is the same $\sigma$ you measured
in v3 and used in v4, appearing here as the width of both conditionals rather
than the width of one.

## Dataset inference: the collection as the unit of evidence

**Deriving $z = d\sqrt{n}$.** Let $T_1, \ldots, T_n$ be the per-document
statistics, independent, each with mean $\mu_1$ under membership, mean $\mu_0$
under non-membership, and common standard deviation $s$. Define the effect size

$$d = \frac{\mu_1 - \mu_0}{s}$$

Take the sample mean $\bar T = \frac{1}{n}\sum_i T_i$. Its expectation is
$\mu_1$ (or $\mu_0$), unchanged by averaging. Its variance is

$$\text{Var}(\bar T) = \frac{1}{n^2}\sum_{i=1}^{n}\text{Var}(T_i) = \frac{n s^2}{n^2} = \frac{s^2}{n}$$

using independence, which is the only place independence enters. So the standard
error is $s/\sqrt{n}$ and the standardized difference of means is

$$z = \frac{\mu_1 - \mu_0}{s/\sqrt{n}} = \frac{\mu_1 - \mu_0}{s}\sqrt{n} = d\sqrt{n}$$

The mean shift is preserved and the noise shrinks. That asymmetry is the entire
mechanism, and it comes from variances adding while means average.

**A two-sample version, since you have both groups.** In practice you compare a
member collection of size $n_1$ against a held-out collection of size $n_0$, and
the standard error of the difference of two independent means is

$$\text{SE} = s\sqrt{\frac{1}{n_1} + \frac{1}{n_0}}$$

which at $n_1 = n_0 = n$ reduces to $s\sqrt{2/n}$ and costs you a factor of
$\sqrt 2$ against the one-sample expression. Budget for it: matching a
one-sample $z$ requires twice the documents.

**Dependence, quantified.** Suppose the $n$ documents fall into $g$ groups
(books, sources, authors) of size $m = n/g$, with correlation $\rho$ between two
statistics in the same group and zero between groups. The variance of the mean
is no longer $s^2/n$. Using
$\text{Var}(\sum_i T_i) = \sum_i \text{Var}(T_i) + \sum_{i \neq j}\text{Cov}(T_i, T_j)$
and counting $g \cdot m(m-1)$ within-group ordered pairs:

$$\text{Var}(\bar T) = \frac{s^2}{n}\big[1 + (m - 1)\rho\big]$$

The bracket is the **design effect**. The effective sample size is

$$n_{\text{eff}} = \frac{n}{1 + (m-1)\rho}$$

Numbers, for the canon example. $n = 10{,}000$ chapters in $g = 250$ books, so
$m = 40$. At $\rho = 0.9$:

$$n_{\text{eff}} = \frac{10{,}000}{1 + 39 \times 0.9} = \frac{10{,}000}{36.1} = 277$$

which is close to the 250 that the crude "count books, not chapters" heuristic
gives, and shows why the heuristic is a good default: at high within-group
correlation, $n_{\text{eff}} \to g$. At $\rho = 0.2$ the same collection has
$n_{\text{eff}} = 10{,}000/8.8 = 1{,}136$, four times larger, so the honest
answer depends on a correlation you have to estimate rather than assume. Estimate
it from your own data: compute the between-group and within-group variance
components and take
$\rho = \sigma_{\text{between}}^2 / (\sigma_{\text{between}}^2 + \sigma_{\text{within}}^2)$.
Reporting that estimate, with the $z$ it produces, is the difference between an
audit that survives cross-examination and one that does not.

**Why fitting and testing on the same data invalidates the $p$-value.** The
method selects a weighting $w$ over a battery of $F$ features to maximize
separation. If $w$ is chosen on the same documents that produce the final
statistic, the maximization has searched a space of directions, and the
resulting statistic is a maximum over that space rather than a single draw. The
distribution of a maximum of $F$ correlated draws is shifted far to the right of
the distribution of one draw, so the nominal $p$-value - computed as if one test
had been run - understates the true tail probability, by roughly the same
multiple-testing factor as running $F$ tests. Splitting the data restores a
single draw, at the cost of half the sample and thus a factor of $\sqrt2$ in
$z$. That trade is always worth taking, because a $p$-value from the unsplit
procedure is not a smaller number - it is not a $p$-value.

## The discipline that converts a number into evidence

**Bonferroni from the union bound.** For events $A_1, \ldots, A_n$,

$$P\left(\bigcup_{i=1}^n A_i\right) \leq \sum_{i=1}^n P(A_i)$$

with no independence assumption anywhere - this is subadditivity of measure and
it is true for arbitrarily dependent events. Let $A_i$ be "test $i$ produces a
false positive," each with probability $\alpha'$ under its null. Then the
family-wise error rate satisfies $\text{FWER} \leq n\alpha'$, and setting
$\alpha' = \alpha/n$ gives $\text{FWER} \leq \alpha$. The proof is two lines and
it holds under any dependence structure, which is why an opposing expert cannot
attack it.

**Šidák, the exact version under independence.** If the tests are independent,
$\text{FWER} = 1 - (1 - \alpha')^n$ exactly, so setting

$$\alpha' = 1 - (1-\alpha)^{1/n}$$

controls FWER at exactly $\alpha$. At $\alpha = 0.05$, $n = 500$: Šidák gives
$1 - 0.95^{1/500} = 1.026 \times 10^{-4}$ against Bonferroni's
$1.000 \times 10^{-4}$. The difference is 2.6%, which is nothing, and it costs
you the independence assumption. Use Bonferroni.

**Expected false positives, and why it is a different quantity.** By linearity of
expectation, which requires no independence either,

$$\mathbb{E}[\text{false positives}] = \sum_{i=1}^{n} P(A_i) = n\alpha'$$

Note that FWER and expected count answer different questions. FWER asks "will
this report contain any error"; the expected count asks "how many." A report
listing 412 hits out of 8,400 tests is judged by the second - 420 expected under
the null - while a report claiming a single document is judged by the first.
Know which one your claim is.

**False discovery rate, since somebody will suggest it.** Benjamini-Hochberg
controls the expected *proportion* of false positives among rejections rather
than the probability of any. Sort the $p$-values $p_{(1)} \leq \cdots \leq
p_{(n)}$ and reject all $i$ up to the largest $i$ with
$p_{(i)} \leq \frac{i}{n}q$. It is far more powerful than Bonferroni and it
controls a weaker guarantee: with 100 rejections at $q = 0.05$ you expect about
5 of them to be wrong, and you do not know which. For a screening report that
generates leads, that is the right trade. For a report that accuses a named
counterparty regarding a named work, it is not, because "5 of these 100
accusations are false and we cannot say which" is not a position you can hold.

**Why a data-dependent threshold has no valid $p$-value.** Let $t^*$ be a
threshold chosen after seeing the data - for example, just below your best
observation. Then

$$P(T < t^* \mid H_0) \neq \alpha$$

because $t^*$ is itself a random variable correlated with $T$. The quantity you
computed conditions on $t^*$ as if it were fixed, and it was not. There is no
correction that repairs this from the numbers alone, because the numbers do not
record how many thresholds were considered. Only a timestamped prior commitment
records it. That is why pre-registration is procedural rather than statistical:
it adds information to the record that the data cannot contain.

**TPR at fixed FPR versus AUC, formally.** AUC is
$P(T_{\text{member}} > T_{\text{non-member}})$ for independent draws, which
equals the integral of TPR over FPR from 0 to 1:

$$\text{AUC} = \int_0^1 \text{TPR}(\text{FPR})\,d(\text{FPR})$$

It weights every FPR equally, including FPR near 1. The operating region for an
accusation is FPR below about 0.01, which is 1% of the integration range, so AUC
assigns it 1% of the weight. Two detectors with identical AUC can differ by an
order of magnitude in TPR at 1% FPR, in either direction. Reporting AUC when the
decision lives in the left tail is not a summary; it is an average dominated by
the region you will never use.

## Where the null comes from

The four rows differ in one formal respect: what randomness the null
distribution is over.

**Extraction.** No null, because no inference. The claim is existential ("this
output exists") rather than probabilistic, and existential claims are verified by
exhibition. There is a hidden statistical step - deciding that the match is not
coincidental - and for a high-entropy span of 50 tokens the probability of
coincidence under any reasonable generative model is astronomically small, which
is why nobody computes it. For a templatic span it is not small at all, which is
why counterfactual filtering matters.

**Post-hoc MIA.** The null is over training runs $\theta \sim
\mathcal{T}(\mathcal{D}\setminus\{x\})$. Inaccessible, as derived above.

**Dataset inference.** The null is over documents $x' \sim \mathcal{P}$ drawn
from the members' distribution but excluded from training. Accessible if and only
if you can realize $\mathcal{P}$ - which is what synthetic held-out generation
does by construction, and what natural identifiers do by finding a corner of the
data whose generating distribution is literally known (a uniform draw from a
known string space).

**Keyed canaries.** The null is over sibling generations from your own keyed
procedure. Fully accessible: you hold the key, so you can generate as many null
samples as you want, and they are exchangeable with the published ones by
construction rather than by assumption. Formally, if $g_K$ is a generator seeded
by secret key $K$ and you publish $S \subset \{g_K(1), \ldots, g_K(m)\}$ while
withholding the complement, then for any statistic $T$ the withheld set gives
i.i.d. draws from exactly the distribution of $T$ under "not trained on," and
the paired comparison between published and withheld siblings removes any
document-level covariate that both share. That is the STAMP construction, and
the reason it detects content present once at under 0.001% of tokens: pairing
eliminates the between-document variance that dominates every unpaired test in
this unit.

## At the bench: three measurements on your own model

Size the three experiments before running them, since two of them are designed
to fail and you want to know in advance that the failure is informative rather
than underpowered.

**Extraction.** With 100 samples per prompt, the smallest per-sample rate you can
distinguish from zero at two standard errors satisfies
$2\sqrt{q/100} < q$, which gives $q > 0.04$. Below a 4% per-sample rate, 100
samples cannot separate the estimate from zero. If you want to characterize
rates down to 1%, budget 400 samples per prompt. This is the difference between
"nothing came back" and "the rate is below 1%," and only the second is a finding.

**Membership inference.** To claim your attack is at chance rather than merely
weak, you need enough documents to bound its AUC. The standard error of an AUC
estimate with $n$ members and $n$ non-members is roughly $1/\sqrt{6n}$ for AUC
near 0.5. With $n = 200$: $\text{SE} \approx 1/\sqrt{1200} = 0.029$. So you can
distinguish AUC 0.5 from AUC 0.56 at two standard errors and no better. Report
the interval, not the point: "AUC 0.52 (95% CI 0.46 to 0.58)" is a result;
"AUC 0.52" invites a reader to believe you measured something.

For the seed-flip count, each document's verdict is a Bernoulli draw and the
flip rate across $n$ documents has the binomial standard error from earlier.
With 200 documents and an observed flip rate of 30%, SE $= \sqrt{0.3 \times 0.7
/ 200} = 0.032$, so you can report 30% plus or minus 6 points. A flip rate
whose interval includes 50% is a coin.

**Dataset inference.** Invert $z = d\sqrt{n}$ to size it. If your bench effect
size is $d = 0.05$ and you want $z = 3$ in the one-sample form,
$n = (3/0.05)^2 = 3{,}600$ documents. In the two-sample form the requirement
doubles to 7,200 total. If your corpus does not hold that many, either the
effect size on your bench is larger than 0.05 - measure it before assuming - or
the honest bench result is an interval that includes no effect, which is still
worth publishing next to the extraction result.

## What you can now do

The five expressions from this unit, with the assumption each one rests on, since
the assumption is what an opposing expert attacks:

$$C_{\text{mem}} = 3.6N \text{ bits}$$

Assumes the measured bits-per-parameter constant transfers to this
architecture and training regime. Conservative in the direction that matters: it
overstates capacity, so a ratio above 1 is a safe conclusion and a ratio below 1
is not.

$$P(\text{extract in } n \text{ samples}) = 1 - (1-q)^n$$

Assumes independent samples, which holds if you resample rather than reusing a
cached generation, and a fixed temperature.

$$z = d\sqrt{n}$$

Assumes $n$ independent units and approximate normality of the aggregate. The
first fails constantly; correct it with the design effect
$1 + (m-1)\rho$ and report $n_{\text{eff}}$.

$$\alpha_{\text{corrected}} = \alpha / n, \qquad \mathbb{E}[\text{FP}] = n\alpha$$

Assume nothing - union bound and linearity of expectation both hold under
arbitrary dependence. These are the only two expressions in the unit that cannot
be attacked on their assumptions, which is exactly why an audit report should be
built on them.

$$\text{FPR}(t) = P_{\theta \sim \mathcal{T}(\mathcal{D}\setminus\{x\})}(T(\theta,x) < t)$$

Assumes you can sample $\mathcal{T}(\mathcal{D}\setminus\{x\})$. You cannot, and
the whole of v7 is the engineering response to that one line.
