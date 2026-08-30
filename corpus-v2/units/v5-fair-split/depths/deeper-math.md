## Eight numbers and a hundred dollars

Canon shows leave-one-out under-allocating on one table. The under-allocation is
a theorem, not an accident of those numbers.

A value function is **submodular** if adding a source to a bigger coalition
helps no more than adding it to a smaller one: for $S \subseteq T$ and
$i \notin T$,

$$v(S \cup \{i\}) - v(S) \;\geq\; v(T \cup \{i\}) - v(T)$$

Overlapping text corpora are the canonical submodular case, because the more
material a model already has, the less any additional source can add that is
genuinely new. The running table satisfies it: A's marginal contribution is 50
into the empty set, 20 into $\{B\}$, 70 into $\{C\}$, 10 into $\{B,C\}$ - and
$\{C\} \subseteq \{B,C\}$ gives $70 \geq 10$, $\varnothing \subseteq \{B\}$
gives $50 \geq 20$, both ways round.

Leave-one-out takes each source's marginal contribution into the *largest*
possible coalition, $N \setminus \{i\}$, which submodularity makes the smallest
of that source's marginal contributions. So

$$\sum_{i \in N} \text{LOO}_i \;=\; \sum_i \big[v(N) - v(N \setminus \{i\})\big] \;\leq\; v(N)$$

with equality only when the game is modular - no interaction anywhere. The
deficit $v(N) - \sum_i \text{LOO}_i$ measures total redundancy in the corpus,
and it is 50 out of 100 on the running table. That single number is a useful
corpus statistic on its own: it says half the corpus's value is carried by more
than one source.

The mirror-image failure belongs to solo values, which take each source's
marginal contribution into the *smallest* coalition. Submodularity makes those
the largest, so $\sum_i v(\{i\}) \geq v(N)$, and the excess is the same
redundancy counted the other way: 120 against 100, an excess of 20. The two
naive rules sit at opposite ends of the same range, and the rule canon derives
sits at a specific weighted average of every point in between.

## What a coalition is worth

The value function canon uses is a log-density ratio, and it is worth writing
that out because it is what makes the whole method label-free.

A language model is a probability distribution over token sequences. For a
held-out document $x = (x_1, \ldots, x_T)$, a model with parameters
$\theta_S$ trained on coalition $S$ assigns

$$\log p_{\theta_S}(x) = \sum_{t=1}^{T} \log p_{\theta_S}(x_t \mid x_{<t})$$

Define the value as the log-density ratio against a fixed baseline model
$\theta_{\varnothing}$ trained without any of the sources:

$$v(S) = \log p_{\theta_S}(x) - \log p_{\theta_{\varnothing}}(x)$$

Two facts follow immediately. $v(\varnothing) = 0$ by construction, which the
game requires. And $v$ is a difference of log-likelihoods on a *fixed* held-out
set, so no labels, no benchmark, and no task definition enter anywhere - the
generative model is its own evaluator. That is the precedent's central trick.

Canon's bits-per-byte version is the same quantity in different units. Total
surprise in nats is $-\log p_{\theta_S}(x)$, so

$$\text{bpb}_{\varnothing} - \text{bpb}_S = \frac{\log p_{\theta_S}(x) - \log p_{\theta_\varnothing}(x)}{\ln 2 \times \text{bytes}} = \frac{v(S)}{\ln 2 \times \text{bytes}}$$

a positive constant times the log-density ratio. Since the Shapley value is
linear in $v$, multiplying $v$ by a positive constant multiplies every share by
the same constant, and the *percentages* - the thing you actually pay against -
are identical. Units are free. This is worth knowing because it means you can
report in whichever unit your audience reads and nobody can accuse you of
having chosen it to move money.

**What is not free is a nonlinear transform.** Replace $v$ by $\log v$, or by
$v^2$, or clip negatives to zero, and the shares genuinely change, because
linearity is exactly what breaks. Clipping is the common one: the precedent
computes shares, clips negatives to zero, and renormalizes. That composite rule
is not the Shapley value of anything. It satisfies symmetry and the null player
condition; it does not satisfy additivity, and its efficiency is restored by
hand. Disclose it as a separate step in the schedule rather than folding it into
"we used Shapley values."

**Negative shares are meaningful.** A source with $\phi_i < 0$ made the model
worse on the held-out set across the weighted average of contexts - typically
machine-translated text, aggressive boilerplate, or a domain mismatch. The
arithmetic handles it without modification. What a contract does with it is a
policy question with three defensible answers: pay zero and renormalize
(the precedent's choice), pay zero and leave the pool short (efficiency
violated, disclosed), or decline the source. Only the first two are splits.

## Four requirements a contract can cite

Here is the uniqueness proof canon states and does not prove. It is three
paragraphs and it is worth owning, because "these four force one answer" is the
sentence the whole product rests on.

**Step one: a basis of very simple games.** For each nonempty coalition
$T \subseteq N$, define the **unanimity game** $u_T$:

$$u_T(S) = \begin{cases} 1 & \text{if } T \subseteq S \\ 0 & \text{otherwise}\end{cases}$$

In words: this game is worth 1 exactly when everyone in $T$ has shown up, and
nothing otherwise. It is the game of a project that needs a specific committee
present and does not care who else is in the room.

**Step two: three of the four axioms pin down the answer on those games.** Take
any $i \notin T$. Adding $i$ to any coalition never changes whether $T$ is
contained in it, so $u_T(S \cup \{i\}) = u_T(S)$ for all $S$ - source $i$ is a
null player, and the null player axiom forces $\phi_i(u_T) = 0$. Now take any
two members $i, j \in T$. For any $S$ containing neither, $T \not\subseteq S
\cup \{i\}$ and $T \not\subseteq S \cup \{j\}$ alike (both are missing the
other), so their values agree and symmetry forces $\phi_i(u_T) =
\phi_j(u_T)$. Efficiency then forces those equal shares to sum to
$u_T(N) = 1$. So

$$\phi_i(u_T) = \begin{cases} 1/|T| & i \in T \\ 0 & i \notin T \end{cases}$$

No formula was assumed. Three axioms and a coalition of one.

**Step three: every game is a combination of those, and additivity finishes
it.** The games with $v(\varnothing) = 0$ on $n$ players form a vector space of
dimension $2^n - 1$, one coordinate per nonempty coalition, and the $2^n - 1$
unanimity games are a basis of it. So every $v$ has a unique expansion

$$v = \sum_{T \neq \varnothing} c_T \, u_T
\qquad\text{with}\qquad
c_T = \sum_{R \subseteq T} (-1)^{|T| - |R|}\, v(R)$$

The coefficients $c_T$ are called **Harsanyi dividends**. Additivity (extended
to scalar multiples, which is how the axiom is normally stated - linearity)
gives $\phi_i(v) = \sum_T c_T \,\phi_i(u_T)$, and step two evaluates every term:

$$\phi_i(v) = \sum_{T \ni i} \frac{c_T}{|T|}$$

That is the Shapley value, arrived at without ever writing down a permutation,
and it is unique because the expansion is unique.

**Worked on the running table**, because the dividends are a diagnostic in their
own right:

$$c_A = 50 \qquad c_B = 50 \qquad c_C = 20$$
$$c_{AB} = 70 - 50 - 50 = -30 \qquad c_{AC} = 90 - 50 - 20 = 20 \qquad c_{BC} = 20$$
$$c_{ABC} = 100 - 70 - 90 - 90 + 50 + 50 + 20 = -30$$

Read them. The singletons are the standalone values. $c_{AB} = -30$ is the
publisher-wire overlap, stated as a number, with the right sign. $c_{AC} =
c_{BC} = +20$ is the specialist journal's synergy with general prose. And
$c_{ABC} = -30$ is the three-way correction that stops the pairwise synergies
being counted twice.

Now split each dividend equally among its members and add up what each source
receives:

$$\phi_A = \frac{50}{1} + \frac{-30}{2} + \frac{20}{2} + \frac{-30}{3} = 50 - 15 + 10 - 10 = 35$$
$$\phi_C = \frac{20}{1} + \frac{20}{2} + \frac{20}{2} + \frac{-30}{3} = 20 + 10 + 10 - 10 = 30$$

Matching canon's six-order table exactly. This is the most useful mental model
of the rule for anyone building it: **every interaction in the corpus generates
a dividend, positive or negative, and each dividend is split equally among the
sources that jointly produce it.** Overlap generates a debt shared by the
overlappers; synergy generates a bonus shared by the synergists.

It also gives the modularity diagnostic from canon's section 7 a precise form.
A game is modular exactly when every dividend on a coalition of size 2 or more
is zero. The additive-fit residual canon computes is a least-squares proxy for
"how much mass sits in the higher-order dividends," and on the running table
that mass is $|{-30}| + 20 + 20 + |{-30}| = 100$ against singleton mass 120 - a
game with substantial but not dominant interaction, which is what the 86.5%
explained variance said.

## Averaging over arrival orders

Two things canon asserts and does not derive: where the coalition weights come
from, and why the shares sum to the whole in every game.

**The weights, by counting.** Fix a source $i$ and a coalition
$S \subseteq N \setminus \{i\}$. How many of the $n!$ arrival orders put exactly
the members of $S$ before $i$, and everyone else after? Arrange $S$ freely among
the first $|S|$ positions: $|S|!$ ways. Put $i$ next. Arrange the remaining
$n - |S| - 1$ sources freely after: $(n - |S| - 1)!$ ways. So the count is
$|S|!\,(n - |S| - 1)!$, and its share of all orders is

$$w_{|S|} = \frac{|S|!\,(n - |S| - 1)!}{n!}$$

which is the coefficient in canon's second formula. There is a tidier way to
write it that makes the structure obvious:

$$w_s = \frac{s!\,(n - s - 1)!}{n!} = \frac{1}{n \binom{n-1}{s}}$$

where $\binom{n-1}{s} = \frac{(n-1)!}{s!\,(n-s-1)!}$ counts the coalitions of
size $s$ drawn from the other $n-1$ sources. Read it as a two-stage uniform
draw: pick a coalition *size* uniformly from the $n$ possible sizes
$0, 1, \ldots, n-1$, then pick a coalition of that size uniformly. Every
cardinality class gets total weight exactly $1/n$, regardless of how many
coalitions are in it. That is the source of the extreme unevenness canon flags:
at $n = 12$, the single empty coalition holds the same total weight as all 462
coalitions of size 5 put together.

**Efficiency, in one line.** Fix any single order $\pi$ and write the sources in
that order as $\pi_1, \ldots, \pi_n$. The marginal contributions along that
order are

$$\big[v(\{\pi_1\}) - v(\varnothing)\big] + \big[v(\{\pi_1,\pi_2\}) - v(\{\pi_1\})\big] + \cdots + \big[v(N) - v(N \setminus \{\pi_n\})\big]$$

Every interior term appears once positive and once negative and cancels. What
survives is $v(N) - v(\varnothing) = v(N)$. So the marginal contributions in
*every* order sum to $v(N)$, and an average of quantities each summing to $v(N)$
sums to $v(N)$. Efficiency is a telescoping sum. It is the cheapest of the four
axioms to satisfy and the reason the permutation construction was the right
place to start looking.

**Symmetry, in one more line.** If $i$ and $j$ are interchangeable, the map that
swaps them in every order is a bijection from the set of orders to itself, and
it carries $i$'s marginal contribution in one order to $j$'s in the image order.
The two multisets of $n!$ marginal contributions are therefore identical, so
their averages are.

## Counting the runs: 4,096, not 479 million

**The family canon draws from, named properly.** A **semivalue** is any rule of
the form

$$\phi_i = \sum_{S \subseteq N \setminus \{i\}} w_{|S|}\big[v(S \cup \{i\}) - v(S)\big]
\qquad\text{with}\qquad \sum_{s=0}^{n-1} \binom{n-1}{s} w_s = 1$$

The constraint normalizes the weights so that a source contributing a constant
$c$ to every coalition receives exactly $c$. Every rule in this unit is a
semivalue; they differ only in the weight profile $w_s$.

| Rule | $w_s$ | Weight given to cardinality class $s$ |
| --- | --- | --- |
| Shapley | $\dfrac{1}{n\binom{n-1}{s}}$ | $1/n$, uniform across sizes |
| Banzhaf | $\dfrac{1}{2^{n-1}}$ | $\binom{n-1}{s}/2^{n-1}$, binomial, peaked at $s \approx n/2$ |
| Leave-one-out | $1$ if $s = n-1$, else $0$ | all on the largest class |
| Solo value | $1$ if $s = 0$, else $0$ | all on the empty class |

All four naive rules canon rejects are members of the same family with degenerate
weights. And within this family, **Shapley is the unique efficient member** -
that is the uniqueness theorem restated in the language of weights, and it is
why Banzhaf's 107.5 against a pool of 100 is not a bug to be patched but the
defining trade.

**Hoeffding, stated precisely.** Let $X_1, \ldots, X_m$ be independent random
variables with $a \leq X_t \leq b$, write $R = b - a$, and let $\bar{X}$ be
their mean. Then for any $\epsilon > 0$,

$$\Pr\big[|\bar{X} - \mathbb{E}\bar{X}| \geq \epsilon\big] \leq 2\exp\!\left(\frac{-2m\epsilon^2}{R^2}\right)$$

In the permutation sampler, $X_t$ is source $i$'s marginal contribution in the
$t$-th sampled order, the orders are drawn independently, and
$\mathbb{E}\bar{X} = \phi_i$ because a uniform draw over orders has the Shapley
value as its mean by definition. Setting the right side to $\delta$ and solving
gives canon's $m \geq \frac{R^2}{2\epsilon^2}\ln\frac{2}{\delta}$.

Two refinements worth knowing. The bound uses only the *range* $R$, so it is
loose whenever marginal contributions are concentrated - Bernstein's inequality
replaces $R^2$ with a variance term plus a smaller range term and typically
cuts the sample count by a large factor once you have measured the variance.
And the bound is per source; covering all $n$ sources simultaneously costs a
union bound, replacing $\delta$ by $\delta/n$, which adds $\ln n$ inside the
logarithm and essentially nothing to the budget.

**Why maximum sample reuse has the property it has, in one calculation.** Draw a
coalition $S$ uniformly at random from all $2^n$ subsets - equivalently, include
each source independently with probability $1/2$. Condition on $i \in S$: then
$S \setminus \{i\}$ is uniform over the $2^{n-1}$ subsets of $N \setminus \{i\}$.
Condition on $i \notin S$: then $S$ is uniform over the same $2^{n-1}$ subsets.
So

$$\mathbb{E}\big[v(S) \,\big|\, i \in S\big] - \mathbb{E}\big[v(S) \,\big|\, i \notin S\big]
= \frac{1}{2^{n-1}}\sum_{S \subseteq N \setminus \{i\}}\big[v(S \cup \{i\}) - v(S)\big] = \beta_i$$

exactly the Banzhaf value. Every sampled coalition therefore lands in one of the
two buckets for *every* source at once, and one training run advances all $n$
estimates. The evaluation budget is
$O\!\left(\frac{1}{\epsilon^2}\log\frac{n}{\delta}\right)$, with $n$ appearing
only through the union bound. Nothing analogous exists for the Shapley weights,
because they depend on $|S|$ and a uniform coalition draw does not deliver the
cardinality-uniform weighting without reweighting - and reweighting reintroduces
variance.

## The error budget: seeds before coalitions

Canon computes the noise coefficient for $n = 3$ and reports $\sqrt{5/9}
= 0.745$. Here is the general formula and what it says about the build.

Expand $\phi_i$ into coefficients on individual table entries. Each $v(T)$ with
$i \in T$ appears once with coefficient $+w_{|T|-1}$; each $v(T)$ with
$i \notin T$ appears once with coefficient $-w_{|T|}$. Treating the entries as
independent measurements each with standard deviation $\sigma$,

$$\text{Var}(\phi_i) = \sigma^2 \sum_T c_T^2 = \sigma^2 \cdot 2\sum_{s=0}^{n-1}\binom{n-1}{s} w_s^2$$

Substitute $w_s = 1/\big(n\binom{n-1}{s}\big)$ and the binomial coefficients
partly cancel:

$$\sum_T c_T^2 = \frac{2}{n^2}\sum_{s=0}^{n-1}\frac{1}{\binom{n-1}{s}}$$

At $n = 3$ the inner sum is $1 + \tfrac12 + 1 = \tfrac52$, giving
$\frac{2}{9}\cdot\frac52 = \frac59$ and canon's $0.745\,\sigma$. For larger $n$
the inner sum converges rapidly to 2, because only the two extreme terms are
appreciable - $\binom{11}{5} = 462$ makes its reciprocal negligible. So

$$\text{SD}(\phi_i) \approx \frac{2\sigma}{n}$$

At $n = 12$ the exact inner sum is 2.241, giving $\text{SD}(\phi_i) =
0.176\,\sigma$ against leave-one-out's $1.414\,\sigma$ - a factor of eight less
noise, from averaging alone.

**Do not believe that number without checking it.** The derivation assumes the
$2^n$ coalition measurements are independent, and in a real sweep they are not.
They share one held-out set, so any peculiarity of that text moves every entry
the same way. They share a tokenizer. If you branch coalitions from a common
trunk checkpoint rather than training each from scratch, they share most of
their optimization trajectory and the correlation is severe. Positive
correlation between the entries inflates the variance of a weighted sum relative
to the independent case, pushing the true standard deviation back up toward
$\sigma$. The $2\sigma/n$ result is a floor, not an estimate.

The remedy is empirical and costs nothing extra: you already trained $k$ seeds
per coalition, so **bootstrap over seeds**. Draw one seed's value per coalition
at random with replacement, compute the full split from that draw, repeat a few
hundred times, and take the standard deviation of each source's share across the
draws. That resamples the actual correlation structure instead of assuming it
away, and it costs a few seconds of arithmetic over a 32-kilobyte table. Report
the bootstrap standard deviation, not the analytic one, and use the analytic one
only as a sanity check that you have not made an arithmetic error.

**What this does to the seeds-versus-coalitions decision.** With exact
enumeration the combinatorial error is exactly zero and every remaining unit of
error is seed noise, which falls as $1/\sqrt{k}$. So the marginal run is always
better spent on a seed. The trade only becomes real above the cliff, where you
are sampling: there, doubling the sample count $m$ cuts the sampling error by
$\sqrt{2}$ and doubling $k$ cuts the seed error by $\sqrt{2}$, and you should
split the budget so the two errors are comparable rather than driving one to
zero while the other dominates.

## Gaming the split

Canon computes the shell attack for two and three shells. Here is the closed
form and the ceiling.

Let source A have marginal contributions $m(S) = v(S \cup \{A\}) - v(S)$ over
the coalitions $S$ of the $p$ other sources, and let A register as $k$
identical shells, so the game has $n = k + p$ players and any shell's marginal
contribution is zero in any coalition containing another shell. Only the $2^p$
coalitions drawn from the other sources survive, so

$$\phi_{A_1} = \sum_{S \subseteq \text{others}} \frac{|S|!\,(n - |S| - 1)!}{n!}\, m(S)
\qquad\text{and the total take is}\qquad \Phi_A(k) = k\,\phi_{A_1}$$

On canon's numbers, with $p = 2$, $m(\varnothing) = 50$, $m(B) = 20$,
$m(C) = 70$, $m(BC) = 10$, and $n = k + 2$:

$$\Phi_A(k) = k\left[\frac{50}{n} + \frac{90}{n(n-1)} + \frac{20}{n(n-1)(n-2)}\right],
\qquad n = k+2$$

| $k$ | $n$ | $\Phi_A(k)$ |
| --- | --- | --- |
| 1 | 3 | 35.00 |
| 2 | 4 | 41.67 |
| 3 | 5 | 44.50 |
| 5 | 7 | 46.90 |
| 10 | 12 | 48.64 |
| $\to \infty$ | | $\to 50.00$ |

As $k$ grows, $k/n \to 1$ while every term beyond the first carries an extra
factor of $1/(n-1)$ and vanishes. The limit is $m(\varnothing) = v(\{A\})$.

**The ceiling has a clean interpretation, and it is the sentence to remember.**
Fragmenting converts a source's Shapley value into its standalone value. It
does not manufacture value out of nothing; it unwinds, in the limit exactly, the
discount that redundancy with other sources imposed. A source with no overlap
gains nothing from fragmenting - and can lose, as canon's beat shows for the
complementary source C, because splitting a complement destroys the scarcity
that made it complementary. **The attack is profitable in proportion to how
much your content duplicates other registrants' content, which is precisely the
population most motivated to try it.**

**Why the axioms cannot see this.** All four are statements about a fixed player
set $N$. Splitting replaces the game $(N, v)$ with a different game $(N', v')$
on a different player set, and nothing in the axiom system relates
$\phi(N, v)$ to $\phi(N', v')$. The fix is to add an axiom that does. Write
$M: N \to N^{\ast}$ for a merge map that groups registered entities into real
rightsholders. A **fragmentation-invariance** requirement says

$$\sum_{i \,:\, M(i) = g} \phi_i(N, v) \;=\; \phi_g\big(N^{\ast}, v \circ M^{-1}\big)$$

for every merge map - a group's total does not depend on how it is partitioned.
Faithful Group Shapley constructs a value satisfying this. Note what it must
give up: the running example's split under fragmentation invariance cannot be
both 35 for A-as-one-player and 41.67 for A-as-two-players, so the fixed-player
Shapley value and the fragmentation-invariant value disagree on at least one of
those games. Adding a fifth axiom to four that already forced a unique answer
means abandoning one of the four - in this construction, additivity is the one
that weakens.

**The practical layer, stated as a computation.** Fingerprint each registrant's
corpus with a MinHash sketch, estimate pairwise Jaccard overlap, and merge any
pair above threshold into one player before the game is played. Two identical
shells have Jaccard 1 and merge unconditionally. The interesting cases are
partial: a registrant offering 60% of another's catalogue is neither a shell
nor an independent source, and where you put the threshold is a policy decision
that moves money and must therefore be in the schedule rather than in the code.
This is the same clustering the corpus pipeline already runs for deduplication,
pointed at registrants rather than at documents, and it is the concrete reason
duplicate clusters had to be recorded rather than silently resolved.

## The honest position

Canon says defensible small changes to the value function move valuations
substantially. Here is the precise sense in which that is and is not true,
because the distinction is the difference between a fixable problem and a
fundamental one.

The Shapley value is a linear operator on value functions, so a perturbation
$\Delta v$ produces $\Delta\phi_i = \sum_T c_T \Delta v(T)$ with the same
coefficients as before. Bound it crudely:

$$|\Delta\phi_i| \;\leq\; \Big(\sum_T |c_T|\Big) \max_T |\Delta v(T)|
\;=\; 2\max_T |\Delta v(T)|$$

using $\sum_T |c_T| = 2\sum_s \binom{n-1}{s} w_s = 2$, which holds for every
$n$. **The operator norm is exactly 2, independent of the number of sources.**
So a uniform perturbation of every coalition's measured value by at most
$\epsilon$ moves any source's share by at most $2\epsilon$. The rule is
perfectly stable against uniform measurement error, at every scale.

That is why the "arbitrary and gameable" critique is not about noise. It is
about *structured* perturbations. Changing the held-out set does not perturb
every coalition by a similar amount; it perturbs the coalitions containing the
newly-favoured source by a lot and the others by nothing, which is exactly the
$w$ example from canon's section 3 that moved A from 35 to 55. The bound above
is satisfied - the perturbation was large - and it is satisfied uselessly. The
degree of freedom is not in the arithmetic, and no property of the arithmetic
can constrain it.

Which locates the defence exactly. It is procedural and it is the same
discipline the verification units use: fix the held-out set, its construction,
its decontamination and its size in writing, and commit to them before any run.
A committed value function makes the split reproducible by a third party from
the published table. An uncommitted one makes it an opinion with a formula
attached.

Two further limits worth stating in the same breath, since a technical
counterparty will raise them. The rule assumes the value function is
well-defined, meaning $v(S)$ is a property of $S$ - but $v(S)$ is really the
expectation over training seeds, and canon's section 6 is entirely about the
gap between the two. And it assumes the player set is the right one: if the
model's capability is shaped substantially by sources that are not registrants
- public web text, synthetic data, the base checkpoint - then the pool being
split is the *marginal* value of the registered corpus over that background,
not the value of the model. That is the correct thing to be splitting, and it
is a smaller number than a rightsholder expects.

## At the bench: the split

**Complexity, so you budget the right thing.** From a cached table of $2^n$
values, computing all $n$ shares takes $n \cdot 2^{n-1}$ multiply-adds -
24,576 operations at $n = 12$, microseconds. The table itself is 4,096
double-precision floats: 32 kilobytes. Every diagnostic in the unit runs over
that same 32 kilobytes: Banzhaf, leave-one-out, the additive least-squares fit
(a $12 \times 12$ normal-equations solve), the Harsanyi dividends (a Möbius
transform over the subset lattice, $O(n 2^n)$ by the standard zeta-transform
recursion), the shell recomputation, and the seed bootstrap.

So the entire computational content of this unit is a small file and a few
loops. The cost is 100% in producing the table, and the table is 4,096
coalitions times $k$ seeds of training runs. Budget accordingly: every
engineering hour spent optimizing the Shapley arithmetic is an hour wasted, and
every hour spent raising training throughput divides the whole project's bill.

**Index the table by bitmask.** A coalition of 12 sources is a 12-bit integer,
the value table is an array of length 4,096 indexed by it, and the marginal
contribution of source $i$ to coalition $S$ is
`v[S | (1 << i)] - v[S & ~(1 << i)]`. Enumerating the coalitions not containing
$i$ is enumeration of the integers with bit $i$ clear. This is the whole
implementation, it removes an entire category of set-handling bug, and it makes
the shell recomputation a relabelling of indices rather than new code.

**One correctness check that catches almost everything.** After computing the
shares, verify $\sum_i \phi_i = v(N)$ to floating-point tolerance. Efficiency
holds identically for the true Shapley value, so any deviation beyond rounding
is an indexing error, a missing coalition, or a weight computed with the wrong
$n$. Run it on every recomputation, including each shell variant, where it
also confirms you built the expanded table correctly.

## What you can now do

The four formulas of the unit, in their general form, with the semivalue
family that contains them:

$$\phi_i = \sum_{S \subseteq N \setminus \{i\}} w_{|S|}\big[v(S \cup \{i\}) - v(S)\big],
\qquad \sum_{s=0}^{n-1}\binom{n-1}{s}w_s = 1$$

$$\text{Shapley: } w_s = \frac{1}{n\binom{n-1}{s}}
\qquad \text{Banzhaf: } w_s = \frac{1}{2^{n-1}}$$

$$\phi_i(v) = \sum_{T \ni i} \frac{c_T}{|T|},
\qquad c_T = \sum_{R \subseteq T}(-1)^{|T|-|R|} v(R)$$

$$\text{SD}(\phi_i) = \sigma\sqrt{\frac{2}{n^2}\sum_{s=0}^{n-1}\binom{n-1}{s}^{-1}} \;\approx\; \frac{2\sigma}{n}
\qquad
m \geq \frac{R^2}{2\epsilon^2}\ln\frac{2}{\delta}$$

Three results to carry, none of which appear in the canon path:

The **Harsanyi dividend** view - each interaction in the corpus generates a
signed dividend, split equally among the sources that jointly produce it - is
the fastest way to explain a surprising split to someone who does not accept the
number. It turns "the formula says 35" into "your overlap with the wire archive
generates a debt of 30 that the two of you share."

The **operator norm of 2** says the rule is unconditionally stable against
uniform measurement error at every $n$, which quarantines the instability
critique to the choice of held-out set, where it belongs and where the defence
is pre-commitment rather than mathematics.

And the **$2\sigma/n$ noise floor**, treated as a floor rather than an estimate,
says that more sources make each share *less* noisy under independence - and
that the way to find out whether your sweep behaves that way is to bootstrap
over the seeds you already paid for, not to trust the algebra.
