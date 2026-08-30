## What the extractor actually hands you

Put a number on the interleaving damage, because the number is worse than the
intuition.

A two-column page has $\ell$ text lines per column. A layout-unaware extractor
that walks the page row by row produces $\ell$ splice points, each welding the
end of a left-column line to the start of a right-column line. Every splice
creates a token sequence that appears nowhere in the source document.

The training loss (v1 of this volume's prerequisite material, and vol 1 in
full) is an average over positions of $-\log P_\theta(x_t \mid x_{<t})$: the
model is scored on every single token, not on documents. So the fraction of
the loss corrupted by splicing is not the fraction of *documents* affected, it
is the fraction of *positions* whose preceding context crosses a splice.

With sequence length $L$ and $\ell$ splices in a page of $T$ tokens, the
positions with a splice somewhere in their context window number roughly

$$T - \frac{T}{\ell + 1}$$

once $L$ exceeds the average distance between splices $T/\ell$. For a typical
two-column page, $\ell \approx 50$ and $T \approx 900$, so the mean gap
between splices is about 18 tokens. At $L = 2048$, essentially *every*
position in the page has a splice in its context. There is no "mostly clean"
regime: the damage is not proportional to the splice count, it saturates.

That is why the AICC result (a 1.08 point swing on a 13-benchmark average at
62B tokens from extraction quality alone) is larger than it sounds. Extraction
errors are not additive noise on a per-document basis; below a certain
inter-splice distance they corrupt the conditioning of every prediction.

## Extraction: three source types, three tools

The recall/precision tradeoff between `trafilatura` and `resiliparse` is
usually quoted as two independent numbers. It is one number and a threshold,
and knowing that tells you when the choice is free and when it is not.

Both extractors classify each candidate text block as content or boilerplate.
Write $C$ for the true content blocks and $\hat{C}$ for the retained set.
Recall is $|C \cap \hat{C}| / |C|$ - the fraction of real content kept.
Retained boilerplate is $|\hat{C} \setminus C| / |\hat{C}|$ - the fraction of
the output that should not be there.

For a page whose true content is a fraction $\rho$ of all blocks, choosing a
more permissive threshold moves both numbers together. The quoted figures:

| | recall | boilerplate share |
|---|---|---|
| trafilatura | 0.695 | 0.066 |
| resiliparse | 0.745 | 0.228 |

resiliparse recovers 5 percentage points more content and pays 16.2 points
more boilerplate for it. The exchange rate is roughly 3.2 points of junk per
point of content, which is a bad trade *unless* something downstream removes
the junk more cheaply than the extractor could. That is exactly the DCLM
architecture: extract permissively with resiliparse, then let a classifier
that keeps the top 10% throw away the boilerplate along with everything else
it dislikes. FineWeb, which relies less on a hard top-$k$ classifier cut,
extracts conservatively with trafilatura.

The lesson generalizes: extractor precision and filter aggressiveness are
substitutes, not independent choices, and quoting either in isolation is
meaningless.

**Why F1 is the right summary here and accuracy is not.** F1 is the harmonic
mean of precision $p$ and recall $r$:

$$F_1 = \frac{2pr}{p + r}$$

The harmonic mean is dominated by the smaller argument: if $p = 0.95$ and
$r = 0.30$, then $F_1 = 2(0.285)/1.25 = 0.456$, much closer to the recall than
to the midpoint 0.625. This is the property you want, because an extractor
that is perfectly precise on the 30% of content it recovers has still
destroyed the document. Plain accuracy would be dominated by the vast number
of correctly-rejected boilerplate blocks and would rate that extractor near
1.0.

## Quality filtering: the classifier is the lever

**The keep-rate arithmetic, and why top-10% is not as aggressive as it looks.**

DCLM keeps the top 10% by classifier score. Suppose the underlying corpus is a
fraction $q$ genuinely good documents, and the classifier assigns scores such
that a document's probability of landing in the top decile is $\alpha$ if good
and $\beta$ if bad, with $\alpha \gg \beta$. The retained corpus's good
fraction is, by Bayes:

$$q' = \frac{q\alpha}{q\alpha + (1-q)\beta}$$

Numbers. Take $q = 0.10$ (a tenth of raw web text is worth training on) and a
classifier with $\alpha = 0.60$, $\beta = 0.044$ - which is what a top-decile
cut implies if it is to select 10% overall:
$0.10(0.60) + 0.90(0.044) = 0.060 + 0.040 = 0.100$. Then

$$q' = \frac{0.060}{0.100} = 0.60$$

The good fraction went from 10% to 60%, a factor of 6, while the token count
dropped by a factor of 10. That is the trade the whole filtering literature is
making: buy a 6x purer corpus for a 10x smaller one. Whether that is a good
trade depends entirely on whether you are token-limited, which is why the
answer flipped as corpora grew past what anyone could train on.

**Why the 1.3T subset beats the 15T superset.** State it as an expectation.
Let the value of a training token be $v$ under a distribution over document
quality. Adding tokens with $v < 0$ - text that actively teaches wrong or
useless structure - lowers the corpus mean. But the deeper reason is that
capacity is finite: a model with $N$ parameters can store roughly a fixed
number of bits (v6 makes this exact at ~3.6 bits per parameter). Bits spent
representing the regularities of SEO spam are bits not spent on anything else.
Filtering is not merely removing noise from an average; it is reallocating a
hard capacity budget. That is why the effect is *larger* for small models,
which is exactly the regime you are building in.

**F1 = 0.82 is a low bar and that is the point.** FineWeb-Edu's regression
head reaches 82% F1 against Llama-3-70B's labels. Roughly one document in five
is misclassified relative to the annotator, and the annotator is itself a
noisy proxy for "educational value." A filter this crude moved MMLU by four
points and ARC by eleven. The lever is not precision; it is that the filter
exists at all.

## Deduplication, worked by hand

**The unbiasedness proof, in full.**

Let $A$ and $B$ be shingle sets, $U = A \cup B$, $I = A \cap B$. Let $h$ be a
hash function that is a uniformly random permutation of the universe of
possible shingles - the standard idealization, well approximated in practice
by $h(x) = (ax + b) \bmod p$ for a prime $p$ and random $a, b$.

Consider the element of $U$ with the smallest hash value; call it $x^*$. Since
$h$ is a random permutation restricted to $U$, every element of $U$ is equally
likely to be $x^*$:

$$\Pr[x^* = x] = \frac{1}{|U|} \quad \text{for each } x \in U$$

Now the two minima. $\min_{x \in A} h(x)$ is $h$ of the smallest-hashing
element of $A$; likewise for $B$. These are equal exactly when $x^*$ lies in
both $A$ and $B$ - because if $x^* \in A$ then $x^*$ achieves $A$'s minimum
(it beats every element of $U \supseteq A$), and symmetrically for $B$. If
$x^* \in A \setminus B$, then $A$'s minimum is $h(x^*)$ and $B$'s minimum is
strictly larger. So

$$\Pr\left[\min_{x \in A} h(x) = \min_{x \in B} h(x)\right] = \Pr[x^* \in I] = \frac{|I|}{|U|} = J(A,B)$$

That is the whole thing. Note what it does *not* require: no independence
assumption about the shingles, no distributional assumption about document
length. It is a counting argument about a permutation.

**Variance, and how many hashes you need.** Each of the $k$ signature
positions is an independent Bernoulli trial with success probability $J$. The
estimator $\hat{J}$ is their mean, so

$$\mathbb{E}[\hat{J}] = J, \qquad \operatorname{Var}(\hat{J}) = \frac{J(1-J)}{k}, \qquad \operatorname{SE}(\hat{J}) = \sqrt{\frac{J(1-J)}{k}}$$

Read $\mathbb{E}[\cdot]$ as "average over many repetitions of the experiment"
and $\operatorname{Var}$ as the average squared deviation from that average;
$\operatorname{SE}$ is its square root, on the same scale as $\hat{J}$ itself.

At $J = 0.75$ the numerator $J(1-J) = 0.1875$, so:

| $k$ | standard error |
|---|---|
| 3 | 0.250 |
| 16 | 0.108 |
| 112 | 0.041 |
| 1000 | 0.014 |

At $k = 3$ the standard error is a third of the quantity being measured, which
is why the toy example in canon can produce an estimate of 0.33 for a true
0.667 without anything being wrong. At FineWeb's $k = 112$, the error is 4
points of Jaccard - adequate for a 0.75 threshold, and note that the
$1/\sqrt{k}$ scaling means the last factor of 3 in accuracy costs a factor of
9 in signature storage.

**The LSH S-curve, derived.** A band of $r$ rows matches when all $r$ of its
positions agree. The positions are independent, so

$$\Pr[\text{one band matches}] = J^{r}$$
$$\Pr[\text{one band fails}] = 1 - J^{r}$$
$$\Pr[\text{all } b \text{ bands fail}] = (1 - J^{r})^{b}$$
$$\Pr[\text{candidate}] = 1 - (1 - J^{r})^{b}$$

**Where the knee is.** The curve's steepest point is near the $J$ at which the
expected number of matching bands equals 1, that is $b J^r = 1$, giving

$$J_{\text{knee}} = \left(\frac{1}{b}\right)^{1/r}$$

For FineWeb's $b = 14$, $r = 8$: $\ln(1/14) = -2.639$, divided by 8 is
$-0.3299$, and $e^{-0.3299} = 0.719$. The configuration is tuned for a
threshold near 0.72, consistent with the stated ~0.75 target.

Two tuning facts that follow directly and are worth having as reflexes.
Increasing $r$ (rows per band) at fixed $k$ makes the filter *stricter* - the
knee moves right, since $1/b$ is raised to a smaller power. Increasing $b$
makes it *more permissive*. And the curve gets sharper as $k = br$ grows with
the knee held fixed, which is the only way to buy both fewer false candidates
and fewer misses.

**Suffix arrays: why 50 tokens and why it is affordable.** A suffix array over
a corpus of $n$ tokens is the sorted list of all $n$ suffix start positions,
which is $n$ integers. Sorting it is $O(n \log n)$ comparisons with
$O(\text{match length})$ per comparison, or linear time with the standard
construction algorithms. Once built, all repeated substrings of length at
least $m$ are found by scanning adjacent pairs in the sorted order and
computing their longest common prefix: two suffixes sharing a prefix of length
$\geq m$ must be adjacent or near-adjacent in the sorted array. One linear
pass.

The 50-token threshold is a false-positive argument. With a vocabulary of $V$
tokens and independent sampling, the probability that a specific 50-token
sequence recurs by chance in a corpus of $n$ tokens is about $n / V^{50}$,
which at $V = 32{,}000$ and $n = 10^{12}$ is $10^{12} / 10^{225}$: zero. Any
50-token exact match is a copy, not a coincidence. At 5 tokens it would be
neither.

**Why per-snapshot dedup beats global, stated as a rate.** Let a document
appear in $c$ snapshots. Global dedup keeps 1 copy regardless of $c$;
per-snapshot dedup keeps $c$ copies. Junk pages are ephemeral and have $c$
near 1; durable, linked-to, re-crawled pages have large $c$. So per-snapshot
dedup implicitly weights each document by its persistence, and persistence
correlates with quality. Global dedup deletes that signal by construction. The
finding is not a quirk of FineWeb's configuration; it is what happens whenever
you flatten a multiplicity that was carrying information.

## Dedup is a payout decision

**The credit-assignment problem, stated formally enough to argue about.**

Let $S = \{1, \dots, K\}$ be the sources. A duplicate cluster $c$ has member
set $M_c \subseteq S$ (the distinct sources holding the passage) and a token
count $t_c$. A credit policy is a function

$$w_c : M_c \to [0,1], \qquad \sum_{s \in M_c} w_c(s) = 1$$

assigning each member a share of whatever cluster $c$ earns. The three
policies in canon are:

- earliest publication: $w_c(s) = 1$ for the argmin of publication date, 0 otherwise;
- equal split: $w_c(s) = 1/|M_c|$;
- no credit for shared content: $w_c \equiv 0$, with cluster earnings returned to the pool.

The silent policy imposed by shard order is $w_c(s) = 1$ for whichever member
the iterator saw first, which is a uniformly random draw from $M_c$ if shards
are shuffled. Note that in *expectation over shuffles* this equals the equal
split - and that is precisely the trap. The expected payout is defensible; the
realized payout on the single run you actually did is a winner-take-all draw.
A rightsholder is not paid in expectation over hypothetical shuffles.

**The variance is the argument.** Under the silent policy, source $s$'s
earnings from clusters are $\sum_c t_c \cdot \mathbb{1}[s \text{ won } c]$,
a sum of independent Bernoulli-weighted terms. If $s$ belongs to $m$ clusters
each of size 2 with equal token counts $t$, its expected earnings are $mt/2$
with variance $m t^2 / 4$, so the coefficient of variation is
$1/\sqrt{m}$. With $m$ small - and for a corpus of 10 to 12 sources and a
handful of preprint/camera-ready pairs, $m$ IS small - the realized payout
swings by tens of percent purely on shuffle seed. That is a number you can
report, and reporting it is more persuasive than the principle.

**Why the model is unchanged and the payout is not.** This is the separability
that makes the fix cheap. The trained weights $\theta$ are a function of the
representative tokens alone:

$$\theta = f(\{\text{tokens of } r_c : c\} \cup \{\text{unclustered tokens}\})$$

The payout is a function of the cluster table and the policy:

$$\text{payout}(s) = g(\{(M_c, t_c) : c\},\ w)$$

$f$ and $g$ share no arguments except the cluster identities. So changing $w$
requires no retraining, and recording $M_c$ costs storage proportional to the
number of clusters, not to the corpus. The entire novel contribution is
recognizing that $g$ exists as a separate function, and giving it its inputs.

## Decontamination and PII

**Bloom filter false positives, since Dolma's decontamination is one.**

A Bloom filter with $m$ bits, $n$ inserted items, and $j$ hash functions has
false positive rate

$$\varepsilon \approx \left(1 - e^{-jn/m}\right)^{j}$$

and the optimal number of hashes for given $m, n$ is $j^* = (m/n)\ln 2$,
giving $\varepsilon \approx 0.6185^{m/n}$. For 10 bits per item,
$\varepsilon \approx 0.0082$; for 16 bits per item, $\varepsilon \approx
0.00046$.

Now the asymmetry that makes this the right structure for decontamination. A
false positive means you discard a training document that did not actually
overlap the eval set: cost is one document out of hundreds of millions, which
is nothing. A false negative is impossible - Bloom filters have no false
negatives - so contaminated documents cannot slip through the n-gram test.
The error you can have is the error you do not care about. This is the same
reason BFF works for exact dedup: dropping a few non-duplicates is free.

**How likely is an innocent 13-gram collision?** Suppose eval sets contribute
$n_e$ distinct 13-grams and a training document contains $n_d$ of them. Under
a crude independence model with effective 13-gram entropy $H$ bits, the
chance a specific pair collides is $2^{-H}$ and the expected number of
collisions is $n_e n_d 2^{-H}$. English 13-grams carry well over 100 bits of
entropy, so with $n_e \approx 10^{7}$ and $n_d \approx 10^{3}$ the expectation
is $10^{10} \cdot 2^{-100} \approx 10^{-20}$. A 13-gram match is a copy. This
is the same argument as the 50-token suffix-array threshold, and it is why the
GPT-3-era convention settled where it did: below about 8 tokens, common
phrases collide constantly; above 13, you start missing paraphrased leakage
without gaining anything on false positives.

**Quasi-identifiers, quantified.** The reason regex PII scrubbing has low
recall is that identification is a property of *combinations*. If a population
of size $P$ is described by attributes with $a_1, a_2, \dots, a_j$ levels, the
number of distinct attribute combinations is $\prod_i a_i$, and the expected
number of people sharing any given combination is $P / \prod_i a_i$. Once that
product exceeds $P$, most combinations are unique - and it exceeds $P$ fast,
because it is a product. Institution (say $10^3$ values) times subfield
($10^2$) times year ($10^1$) times career stage ($5$) is $5 \times 10^6$
combinations against a population of maybe $10^6$ researchers. No individual
field is an identifier; the tuple is. Nothing lexical can see this, because
each component is an ordinary word.

## Training the tokenizer

**BPE as greedy compression, and why the greedy step is the right one.**

Each merge replaces the most frequent adjacent pair with one symbol. If that
pair occurs $f$ times in the corpus, the merge reduces the corpus's token
count by exactly $f$ (each occurrence goes from 2 symbols to 1) and increases
the vocabulary by exactly 1. So the merge chosen at each step is the one with
the best immediate ratio of tokens-saved to vocabulary-spent. BPE is greedy
hill-climbing on

$$\text{(corpus token count)} \quad \text{subject to} \quad |V| \leq V_{\max}$$

It is not optimal - a merge that is unpopular now may enable a very popular
merge later, and greedy will not see it - but the objective is explicit and
the failure mode is bounded.

**Why the gains diminish, from the frequency distribution.** Word and n-gram
frequencies in natural text follow an approximately Zipfian law: the $i$-th
most frequent item has frequency proportional to $1/i^{\alpha}$ with $\alpha$
near 1. The cumulative tokens saved after $V$ merges is therefore
proportional to

$$\sum_{i=1}^{V} \frac{1}{i} \approx \ln V + \gamma$$

where $\gamma \approx 0.577$ is a constant. Compression gain grows like
$\ln V$: logarithmic. Meanwhile the embedding table costs $V \times d_{model}$
parameters: linear. Logarithmic benefit against linear cost is the entire
vocabulary-sizing argument in one line, and it explains the shape of the
canon table - going from 16K to 32K doubles the parameter cost of the table to
buy $\ln 2 / \ln(16{,}384) \approx 7\%$ more of whatever compression remained.

**The data-bottleneck direction of the vocabulary law.** Tao et al.'s result
has two halves and the second is the operative one here. Consider the update
count per embedding row. A corpus of $D$ tokens distributed over $V$ vocabulary
entries gives row $i$ an expected $D \cdot p_i$ gradient updates, where $p_i$
is that token's corpus frequency. Under Zipf, $p_i \propto 1/i$, normalized by
$\sum_{j \le V} 1/j \approx \ln V$, so

$$\mathbb{E}[\text{updates to row } i] \approx \frac{D}{i \ln V}$$

For the *last* row, $i = V$: $D / (V \ln V)$. At $D = 5 \times 10^{8}$ and
$V = 128{,}256$: $5\times10^{8} / (128{,}256 \times 11.76) \approx 331$
updates. At $V = 16{,}384$: $5\times10^{8} / (16{,}384 \times 9.70) \approx
3{,}146$ updates - ten times more. A row with a few hundred updates from
random initialization has not been trained; it is a glitch token waiting to
happen. This is what "optimal vocabulary shrinks when data-bottlenecked"
means mechanically.

**The compression arithmetic in its general form.** For corpus size $c$
characters and tokenizer compression $\kappa$ characters per token, the token
count is $D = c/\kappa$. Training cost is proportional to $N D$ (v3 derives
$C \approx 6ND$), so with $N$ fixed the cost ratio between two tokenizers is

$$\frac{D_1}{D_2} = \frac{\kappa_2}{\kappa_1}$$

The saving depends only on the ratio of compression rates, not on corpus size.
Going from 4.0 to 4.8 characters per token is a factor of $4.0/4.8 = 0.833$,
so 16.7% off, at every scale.

## Mixing and packing

**RegMix as a regression, and why it is already an attribution method.**

RegMix trains $M$ small models, model $j$ on mixture proportions
$\mathbf{p}^{(j)} = (p^{(j)}_1, \dots, p^{(j)}_K)$ over $K$ sources, with
$\sum_k p^{(j)}_k = 1$. It records each model's final loss $\ell_j$ and fits

$$\ell_j \approx \beta_0 + \sum_{k=1}^{K} \beta_k\, p^{(j)}_k$$

by least squares - that is, choosing the $\beta$s that minimize
$\sum_j (\ell_j - \hat{\ell}_j)^2$, the summed squared prediction error. The
fitted $\beta_k$ is the estimated change in loss per unit increase in source
$k$'s share, holding the others fixed. Then the mixture is chosen by
minimizing the fitted surface.

Now read the same fit with attribution eyes. $-\beta_k$ is an estimate of
source $k$'s marginal value in loss units, obtained from $M$ small training
runs and one linear regression. That is exactly the datamodel construction v4
builds ground truth from, differing only in the design matrix: RegMix samples
continuous proportions from a Dirichlet distribution, while datamodels sample
binary inclusion vectors $\mathbf{z} \in \{0,1\}^K$. Same estimator, same
runs, two literatures.

Two properties worth carrying to v4:

- The linear form is a first-order approximation. It cannot represent
  interactions - "source A only helps if source B is present" - which a
  Shapley value over a general value function can. Adding interaction terms
  $\beta_{k\ell} p_k p_\ell$ costs $\binom{K}{2}$ more coefficients and
  therefore more runs.
- The estimate's precision is bounded by seed noise, not by $M$. If two
  identical mixtures trained with different seeds differ in loss by $\sigma$,
  no amount of regression recovers effects smaller than $\sigma$. This is the
  noise-floor discipline v3 and v4 are built around.

**Packing without masking, in gradient terms.** Consider a packed sequence
containing document $u$ at positions $1..n_u$ and document $v$ at positions
$n_u+1..L$. The loss contribution at a position $t > n_u$ is
$-\log P_\theta(x_t \mid x_{<t})$, and $x_{<t}$ includes all of $u$. The
gradient of that term with respect to $\theta$ therefore passes through
attention weights placed on $u$'s tokens.

Count the affected pairs. Under a full causal mask, position $t$ attends to
$t$ positions, of which $n_u$ belong to the other document when $t > n_u$. The
number of cross-document attention pairs is

$$n_u \cdot (L - n_u)$$

maximized at $n_u = L/2$, where it is $L^2/4$ - a quarter of all
$\binom{L}{2}$-ish pairs in the sequence. Intra-document masking sets exactly
those $n_u(L - n_u)$ entries to $-\infty$ before the softmax, so their
attention weights are 0 and their gradients are 0. The cost is constructing
the mask, which is a comparison of document-id vectors and is free next to the
attention itself.

For attribution this is not a quality argument. Every per-example influence
method in v8 computes a per-sequence gradient $\nabla_\theta \ell_{\text{seq}}$
and treats it as "what this training example did to the model." If the
sequence spans two sources, that gradient is a sum over both, and no
decomposition after the fact recovers the split, because the two contributions
were added inside a nonlinear function of shared parameters before you saw
them.

## The provenance manifest

**Why floating-point non-associativity breaks replay, precisely.**

IEEE-754 addition is commutative but not associative: $(a + b) + c \neq
a + (b + c)$ in general, because each addition rounds to the nearest
representable value. The classic demonstration in double precision: with
$a = 1$, $b = 10^{-16}$, $c = 10^{-16}$, computing $(a+b)+c$ rounds $b$ away
at each step and yields exactly 1, while $a + (b+c)$ forms $2 \times 10^{-16}$
first, which is large enough to survive the addition, and yields a value
strictly greater than 1.

A distributed gradient reduction sums per-device gradients in an order set by
the collective's tree topology. Change the device count, the accumulation
step count, or the kernel, and the summation order changes, so the summed
gradient differs in the last bits. That difference is amplified by every
subsequent step: training is a chaotic dynamical system in the sense that
nearby trajectories separate. Two runs identical except for reduction order do
not stay close; they end at different points with the same loss.

This is why "the same data and the same seed" is not sufficient for replay,
and it is the whole reason Levanter's bitwise-reproducibility guarantee is
listed in this unit. An attribution claim of the form "removing source $k$
changed held-out loss by $\delta$" is measured as a difference between two
training runs. If the runs differ in reduction order as well as in data, you
have measured $\delta$ plus an unknown perturbation of the same character as
seed noise (v4's central problem). Pinning the arithmetic removes one of the
two nuisance terms.

**Manifest lookup cost.** The queries the later units make are:

1. "Which source produced training token $i$ of shard $s$?" Binary search on
   `offset` within shard $s$: $O(\log n_s)$ where $n_s$ is the document count
   in that shard. The manifest is sorted by `(shard_id, offset)` by
   construction, since documents are appended in order, so no index build is
   needed.
2. "Which token spans belong to source $k$?" One pass over the manifest
   filtering on `source_id`, or a secondary index: $O(n)$ once, cached.
3. "What documents were in batch $n$?" Compute the sequence ids in batch $n$
   from the seed and shuffle algorithm, then map each sequence id to its
   span $[Ls, L(s+1))$ and range-query the manifest. $O(\log n + \text{hits})$
   per sequence.

All three are constant or logarithmic, which is why the layout is
$(\text{shard}, \text{offset}, \text{length})$ and not, say, a per-token
source array. A per-token array would be $n$ entries of $\log_2 K$ bits -
for $10^9$ tokens and 12 sources, about 500 MB - to answer query 1 in $O(1)$
instead of $O(\log n)$, while making queries 2 and 3 worse. The interval
representation dominates it on every axis.
