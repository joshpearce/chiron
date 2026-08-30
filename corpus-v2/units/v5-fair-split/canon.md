---
unit: v5
title: "The fair split: Shapley from scratch"
concepts:
  - d-coop-game
  - d-shapley-axioms
  - d-semivalue-family
  - d-split-gaming
assumes:
  - d-counterfactual
  - d-noise-floor
---

Every other unit in this book measures something. This one divides something.
You will finish holding a rule that takes a table of measurements and returns a
percentage per data source, together with the four requirements that force that
rule and no other, the arithmetic that shows it is affordable at the only
granularity that matters, the error bar it carries, and the attack that breaks
it in the field.

The reason to spend fifty minutes here rather than on a better estimator is
that this is the only piece of machinery in the volume whose output a lawyer
can audit. A gradient method can tell you that source 7 has an influence score
of 0.0412. It cannot tell a rightsholder why that means 3.2% of the pool. The
argument in this unit can, and the argument is four sentences long. There is no
calculus anywhere in it - counting, averaging, and one square root.


## Eight numbers and a hundred dollars

Three data sources went into a training run.

- **A** is a mid-size trade publisher's back catalogue: 40,000 books.
- **B** is a news wire archive. It covers much of the same ground as A, often
  quoting and summarizing the same material.
- **C** is one small specialist journal, in a technical niche neither of the
  others touches.

Somebody trained a model on every possible combination of those three and
measured how much better each resulting model got at predicting held-out text
it never saw. The unit of measurement is thousandths of a bit per byte of that
held-out text, relative to a model trained on none of the three. Bigger is
better. Section 2 takes that measurement apart; for now it is just a number
that came off a bench.

| Sources in the run | Value |
| --- | --- |
| none | 0 |
| A alone | 50 |
| B alone | 50 |
| C alone | 20 |
| A and B | 70 |
| A and C | 90 |
| B and C | 90 |
| A, B and C | 100 |

Eight rows, and all the structure of the problem is already in them. A and B are
each worth 50 alone but only 70 together, so 30 units of what they carry is
shared - the wire archive is largely reporting on the same books. C is worth
almost nothing alone, 20 units, because it is tiny; but adding C to A takes the
model from 50 to 90, so in company C is worth 40. The niche journal is dense
technical text that only pays off once the model has enough general prose to
read it with.

A licensee has agreed to pay $100 for the whole corpus.

```beat
id: v5-b1
type: predict
concept: d-coop-game
prompt: |
  Before reading on, commit to an answer. Using only the eight numbers in the
  table, split the $100 among A, B and C. Write down three dollar figures that
  sum to 100.

  Then write down the rule you used, in one sentence, as if you had to put it
  in a contract that A, B and C all sign.
answer: |
  The dollar figures are not graded and there is no single right answer at this
  point in the unit - that is the point of asking now. What matters is that you
  wrote down a rule, because the rest of this section is going to break the
  three rules people usually reach for.

  The rules people reach for, in order of frequency:

  1. Split it equally: $33.33 each. Ignores the table entirely.
  2. Split it in proportion to what each source is worth alone (50, 50, 20):
     $41.67, $41.67, $16.66.
  3. Split it in proportion to what is lost when each source is removed
     (100 - 90 = 10 for A, 10 for B, 100 - 70 = 30 for C), renormalized:
     $20, $20, $60.

  The answer this unit arrives at is $35, $35, $30, and it is the only split
  that satisfies four requirements you will almost certainly agree to before
  you see the formula.
rubric: |
  Grade only the second half - whether a stated rule exists and is a function
  of the table rather than of the story. Any of the three rules above, or a
  hybrid, passes.
  An answer with dollar figures and no stated rule = fail, and re-deliver the
  prompt: the whole unit is about the rule, and a split you cannot write down
  as a rule cannot go in a contract.
  An answer that refuses to commit ("not enough information") = fail. There is
  enough information; there are too MANY defensible answers, which is a
  different problem and the one being set up.
check: llm
```

Here is what those three rules pay, side by side.

| Rule | A | B | C |
| --- | --- | --- | --- |
| Equal split | $33.33 | $33.33 | $33.34 |
| Proportional to solo value | $41.67 | $41.67 | $16.66 |
| Proportional to leave-one-out | $20.00 | $20.00 | $60.00 |

Look at C's row. The same eight measurements, three rules that all sound
reasonable when read aloud, and the specialist journal is paid anywhere from
$16.66 to $60.00 - a factor of 3.6. Nobody cheated. Nobody fabricated a number.
The measurement is not in dispute. The allocation rule is doing all the work,
and it is doing it invisibly.

**Equal split fails first and hardest**, because it is not a function of the
data at all. Add a fourth source consisting of 500,000 words of press-release
boilerplate that changes no model's behavior in any combination, and equal
split hands it $25. Every deployed scheme surveyed in this space uses either
equal split or relevance-proportional allocation, and the one paper that
measured both against a principled allocation found them "highly unfair." Set
it aside.

**Proportional to solo value double-pays for the overlap.** A and B share 30
units of content. Under this rule, A is billed 50 - including all 30 shared
units - and B is billed 50, including the same 30 units again. The licensee
pays twice for one thing. You can see the double-count directly: the solo
values sum to 120 while the corpus is worth 100, and the rule handles the
excess 20 by silently scaling everything down by a factor of 100/120, which
spreads the correction over the innocent party too. C, who shares nothing with
anyone and unlocks 40 units of value in company, is cut to $16.66 - less than
the equal split, for being small.

**And now the rule that looks most rigorous.**

<!-- refutes: V5-M4 -->
**You probably think leave-one-out is the contribution.** It is the
counterfactual, it is what v4 spent an entire unit computing, and it has the
cleanest possible definition: retrain without source $i$, see how much worse
the model gets, and that difference is what source $i$ contributed. If that is
what a source contributed, it should be what a source is paid for.

**Here is the prediction that fails.** If leave-one-out were an allocation
rule, the amounts it computes would add up to the thing being allocated. They
do not. On this table the three leave-one-out values are 10, 10 and 30, which
sum to 50 while the corpus is worth 100. Half the pool is unassigned and the
rule has nothing to say about who gets it. The standard patch is to renormalize
- multiply everything by 100/50 - and that patch is where the money moves: it
takes C from a measured 30 to a paid 60, doubling a number that was measured
and not doubling anything else, for reasons that appear nowhere in the data.

Now make the failure total. Two sources, P and Q, holding byte-identical
corpora - the same catalogue licensed through two channels. Train on either one
and the model reaches the same place; train on both and nothing improves.

| Sources in the run | Value |
| --- | --- |
| none | 0 |
| P alone | 100 |
| Q alone | 100 |
| P and Q | 100 |

Leave-one-out for P is $100 - 100 = 0$. For Q it is 0. Both sources are
individually worthless by the counterfactual test, both are individually
sufficient by inspection, and the renormalization step now divides by zero.
There is no split. Two sources that jointly carry the entire corpus are each
assigned nothing, and the rule does not degrade gracefully; it stops existing.

**Here is why the wrong model is appealing.** Leave-one-out is the correct
answer to a different question, asked well. "What happens to this model if this
source disappears" is a real, measurable, decision-relevant quantity, and it is
exactly the number a lab needs when deciding whether to renew a licence. It is
also the ground truth every attribution method in this book is scored against.
None of that makes it an allocation.

**Here is what is actually true.** Leave-one-out measures replaceability, and
replaceability is not worth. In a corpus with any redundancy at all, everyone is
replaceable and nobody is, and the sum of "what breaks without you" is
systematically less than "what we have" - because the parts that two sources
both supply are counted for neither. Allocation needs a rule whose outputs sum
to the pool by construction. That requirement has a name, it is the first of
four, and together the four leave exactly one rule standing.

## What a coalition is worth

Before splitting anything, be precise about the number being split. Two
definitions carry the rest of the unit, and both are things you have already
computed rather than new machinery.

**A coalition is a set of sources.** Write $N$ for the full set of sources -
here $N = \{A, B, C\}$ - and $S$ for any subset of it, including the empty set
$\varnothing$ and $N$ itself. With three sources there are eight subsets, which
is why the table has eight rows.

**A value function is the score of a model trained on exactly that set.** Write
$v(S)$ for it: a single number attached to each subset, produced by actually
training a model on the sources in $S$ and evaluating it. In the table,
$v(\{A,C\}) = 90$. The pair $(N, v)$ - a set of players and a number for every
subset of them - is all a cooperative game is. There is no strategy in it, no
turns, no opponent. It is a table with $2^n$ rows.

**What $v$ is made of, from scratch.** Fix a held-out set of text before any
training - real documents, decontaminated against every shard, chosen once and
never touched again. Run a trained model over it. At each token position the
model assigns some probability to the token that actually occurred; take the
negative natural logarithm of that probability, which is the model's surprise
at that token in nats, and add those up over the whole held-out set. Divide the
total by the held-out set's size in bytes, and divide by $\ln 2 = 0.6931$ to
convert nats into bits. The result is **bits per byte**: how many bits of
information the model still needs, on average, to be told each byte of text it
had not seen.

Worked, with a 200,000-byte held-out set. A model trained on nothing relevant
accumulates 174,000 nats of surprise over it:

$$\text{bpb}_{\varnothing} = \frac{174{,}000}{0.6931 \times 200{,}000} = \frac{174{,}000}{138{,}620} = 1.255$$

A model trained on all three sources accumulates 160,000 nats:

$$\text{bpb}_{N} = \frac{160{,}000}{138{,}620} = 1.154$$

The difference is $1.255 - 1.154 = 0.101$ bits per byte, and every value in the
table is that difference in thousandths - 101, which is the 100 in the table to
the only precision the seed noise in section 6 supports. Two
properties come free from defining it this way. $v(\varnothing) = 0$ by
construction, because the empty coalition is the baseline you subtracted. And
the number is comparable across coalitions, because the same held-out bytes and
the same $\ln 2$ sit in every denominator.

This is not a convenience choice. It is the value function the one direct
precedent in the literature uses: the log-density ratio between a
coalition-trained model and a public baseline model, evaluated on held-out
artifacts. Generative models are density estimators, so they hand you a smooth,
signed, unbounded quality number with no labels and no benchmark required.
Section 6 shows what happens to everything downstream if you use accuracy on a
multiple-choice benchmark instead, and the answer is that the whole unit stops
working.

**Two properties deliberately not assumed.** First, $v$ need not increase when
you add a source. A source of machine-translated sludge can make a model worse,
giving $v(S \cup \{i\}) < v(S)$, and the arithmetic in this unit handles that
without modification - it produces a negative share, which is a real finding
about that source and not an error. The precedent clips negatives to zero and
renormalizes before paying; note that clipping is a fifth rule bolted on after
the fact, and it breaks one of the four requirements in the next section. If
you clip, say so out loud.

Second, sources need not help each other. $v(\{A,B\}) = 70$ is less than
$v(\{A\}) + v(\{B\}) = 100$, and that is the normal case for overlapping text
corpora. Nothing below requires the parts to add up; the entire point is to
handle the case where they do not.

```beat
id: v5-b2
type: compute
concept: d-coop-game
prompt: |
  Using the three-source table (none 0; A 50; B 50; C 20; A+B 70; A+C 90;
  B+C 90; A+B+C 100), compute the marginal contribution of C to the coalition
  $\{A, B\}$ - that is, $v(\{A,B,C\}) - v(\{A,B\})$.

  Give one number.
answer: 30
check: exact
```

## Four requirements a contract can cite

Stop thinking about formulas. You are drafting a schedule to a licensing
agreement, and you have to write down the properties the split must have, in
language a court could apply, before anyone computes anything. Here are four.
Each one is a sentence a rightsholder would sign, and each one is followed by
the specific absurdity that occurs if you drop it, worked on the three-source
table.

Notation, introduced one symbol at a time and needed for the rest of the unit:
$\phi_i$ (phi, sub $i$) is the number of units allocated to source $i$. Three
sources means three of them, $\phi_A$, $\phi_B$, $\phi_C$, and the licensee's
$100 is divided in proportion to them.

**Requirement 1: efficiency. The shares sum to exactly the value of the whole
corpus.**

$$\sum_{i \in N} \phi_i = v(N)$$

where the sum runs over every source and $v(N)$ is the value of the coalition
containing all of them - 100 here.

*Dropped:* leave-one-out, from section 1. The shares sum to 50 against a pool of
100, and the missing half is distributed by whatever renormalization somebody
picks afterwards. A rule that needs a second, unstated rule to finish is not a
rule.

<!-- refutes: V5-M2 -->
**You probably think efficiency means the split is fair.** The word is doing
that. In ordinary use, an efficient allocation is one that does not waste
anything, and "efficient" sitting at the top of a list of fairness axioms reads
like the axiom that delivers fairness.

**Here is the prediction that fails.** If efficiency implied fairness, no
efficient split could be obviously unfair. Take $\phi_A = 100$, $\phi_B = 0$,
$\phi_C = 0$. The shares sum to 100. Efficiency is satisfied exactly. B and C
get nothing, and there is nothing in this requirement that objects.

**Here is why the wrong model is appealing.** Efficiency is the requirement that
makes the output a *split* rather than a set of scores, which is a genuine and
large step - every gradient method in this book produces scores that sum to
nothing in particular, and no amount of post-hoc normalization makes them an
allocation. Efficiency does real work. It is one-quarter of the work.

**Here is what is actually true.** Efficiency is an accounting constraint and
carries no fairness content whatsoever. It says the pool is fully distributed
and not overdrawn. Every fairness claim in this unit comes from the other three
requirements, and when you drop efficiency deliberately - which section 5 does,
for a good reason - you lose the accounting, not the fairness.

**Requirement 2: symmetry. Two sources that do the identical thing get the
identical share.** Formally, if for every coalition $S$ that contains neither
of them,

$$v(S \cup \{i\}) = v(S \cup \{j\})$$

then $\phi_i = \phi_j$.

Check the condition on A and B. Coalitions containing neither: $\varnothing$,
where $v(\{A\}) = v(\{B\}) = 50$; and $\{C\}$, where $v(\{A,C\}) =
v(\{B,C\}) = 90$. A and B are interchangeable in every context. So any rule
obeying symmetry must pay them equally.

*Dropped:* a rule that pays A more than B. It is paying for something that is
not in the measurements - the size of the catalogue, the name on the letterhead,
who signed first, who has the better lawyer. This is the requirement that makes
the split auditable, because it is falsifiable from the value table alone. A
rightsholder who suspects they are being underpaid relative to an equivalent
peer can check symmetry without access to the model, the weights, or the
algorithm. They need eight numbers.

**Requirement 3: the null player. A source that adds nothing to any coalition
gets nothing.** If for every coalition $S$ not containing $i$,

$$v(S \cup \{i\}) = v(S)$$

then $\phi_i = 0$.

*Dropped:* padding pays. Suppose a fourth source D arrives: 500,000 words of
press releases and templated legal boilerplate. Extend the table - $v(S \cup
\{D\}) = v(S)$ for all eight subsets $S$, because the model learns nothing from
it. Under equal split, D collects $25 of a $100 pool. Under any
volume-proportional rule, D collects in proportion to its token count, which its
owner controls directly and can raise to any level for the price of a text
generator. The null player requirement is the one that makes the split
adversarially robust to the cheapest possible attack, and it is exactly the
attack the precedent paper lists as an open limitation under the name
"adversarial padding."

```beat
id: v5-b3
type: self-explain
concept: d-shapley-axioms
prompt: |
  A prospective data partner proposes billing by token count: each source is
  paid its share of the corpus's total tokens. They point out, correctly, that
  this is cheap to compute, fully transparent, and impossible to dispute.

  Using the null player requirement, explain in two or three sentences what
  goes wrong, and name the specific thing the partner can do that a
  measurement-based split would catch and this one will not.
answer: |
  Token count is a property the source controls and the model does not respond
  to. A source can generate half a million words of boilerplate, press
  releases, or templated filler, add it to its submission, and raise its payout
  in exact proportion, while every coalition's measured value is unchanged -
  formally, it is a null player, since v(S union {D}) = v(S) for every S, and
  the null player requirement says its share must be zero. A measurement-based
  split assigns it zero automatically because the padding moves no held-out
  loss; a volume-based split assigns it whatever fraction of the corpus the
  padding represents. Transparency is not the problem with the proposal - it is
  perfectly transparent, and perfectly gameable, which are unrelated
  properties.
rubric: |
  Must contain: (1) the padded content is a null player - adding it leaves
  every coalition's value unchanged, (2) token count is under the source's
  unilateral control, so payout can be inflated at will, (3) the null player
  requirement forces a zero share for such content.
  (1) and (2) = pass. All three = full credit.
  An answer objecting only that token count "does not measure quality", with no
  reference to the manipulation or to the null player condition, = partial; the
  learner has the sentiment and not the argument.
  An answer defending token count as fair because it is transparent and
  auditable = fail. Auditability and incentive-compatibility are different
  properties, and this is the confusion the section exists to break.
check: llm
```

**Requirement 4: additivity. Splitting two payments separately gives the same
result as splitting them together.** If the licensee runs two evaluations - say
the model is scored on held-out legal text and, separately, on held-out
clinical text - producing two value functions $v$ and $w$ over the same
sources, then for every source

$$\phi_i(v + w) = \phi_i(v) + \phi_i(w)$$

where $v + w$ is the value function that assigns each coalition the sum of its
two scores.

*Dropped:* the accounting period becomes a negotiating lever. If the shares
from a combined evaluation differ from the sum of the shares from two separate
evaluations, then whether you settle quarterly or annually, and whether the two
held-out domains are scored in one run or two, changes what everyone is paid.
Every party then has an interest in the calendar rather than in the data, and
the schedule to the agreement has to specify the aggregation order, which is a
term nobody can justify on the merits.

**Worked, because this is the requirement that is hardest to feel.** Add a
second held-out set, clinical notes. Source A happens to be the only one of the
three carrying any medical prose, so on that held-out set:

| Coalition | $w$ |
| --- | --- |
| none | 0 |
| A | 20 |
| B | 0 |
| C | 0 |
| A, B | 20 |
| A, C | 20 |
| B, C | 0 |
| A, B, C | 20 |

A contributes 20 to every coalition it joins; B and C contribute nothing to any
coalition, so both are null players in $w$. The requirements above already force
$\phi_A(w) = 20$, $\phi_B(w) = 0$, $\phi_C(w) = 0$.

Now the combined table, adding the two row by row: none 0; A 70; B 50; C 20;
A+B 90; A+C 110; B+C 90; A+B+C 120. Section 4 computes shares for it and gets
$\phi_A = 55$, $\phi_B = 35$, $\phi_C = 30$. Check the requirement:
$35 + 20 = 55$ for A, $35 + 0 = 35$ for B, $30 + 0 = 30$ for C. It holds
exactly, and it is not an accident of these numbers.

<!-- refutes: V5-M3 -->
**You probably think these four are natural law.** They are called axioms, they
are stated in a theorem, and every treatment of the subject presents them as
the properties fairness has. Under that belief the rule that follows from them
is *the* fair split, and a scheme that does something else is wrong.

**Here is the prediction that fails.** If the four were forced, no serious
alternative would drop one. Section 5 introduces a rule used across the
valuation literature specifically because it is more robust to training noise,
and the thing it gives up is efficiency - its shares do not sum to the pool,
and practitioners renormalize and say so. Meanwhile the fix for the attack in
section 7 works by *adding* a fifth requirement, one about behavior across
different player sets, which none of the four constrains. The set is neither
minimal nor closed.

**Here is why the wrong model is appealing.** The uniqueness theorem is real and
it is strong: these four requirements admit exactly one rule. When four
statements you would sign force one answer, it feels like discovery. And the
requirements really are unusually easy to defend - that is why they are the
right ones to lead a negotiation with.

**Here is what is actually true, and it matters more in a pitch than in a
paper.** The axioms are a choice a contract adopts, not a fact about the world.
Their power is that they are legible: a counterparty can read four sentences,
agree to them, and be bound by the number that follows. That is a
mechanism-design property, not a mathematical one. The correct way to present
this is "here are the four properties your split will have, and here is what
you give up if you want a different set" - never "this is the fair number." The
second version loses the room to the first person who asks why efficiency and
not, say, guaranteed minimums.

**The uniqueness claim, stated.** There is exactly one rule assigning a share to
every source in every value function over a fixed set of sources that satisfies
efficiency, symmetry, the null player property, and additivity. Not one family.
One. It was proved in 1953 and the proof is short: the four requirements pin
the answer down on a set of very simple games, and additivity then forces it
everywhere else, because every value function can be written as a combination
of those simple games. The deeper-math depth for this section carries the
construction. You do not need the proof to use the result, and you do need the
result to defend the rule.

## Averaging over arrival orders

Here is the rule the four requirements force, and it is worth meeting as a
procedure before meeting it as a formula.

**Imagine the sources arriving one at a time, in some order.** The first one to
arrive is measured against nothing. The second is measured against whatever the
first one already provided. The third against the first two, and so on. Each
source's **marginal contribution** in that order is the amount the value goes up
when it walks in.

One order, walked all the way through. Order C, A, B on the three-source table.

- C arrives first, into an empty room: $v(\{C\}) - v(\varnothing) = 20 - 0 = 20$.
- A arrives second, joining C: $v(\{A,C\}) - v(\{C\}) = 90 - 20 = 70$.
- B arrives third, joining A and C: $v(\{A,B,C\}) - v(\{A,C\}) = 100 - 90 = 10$.

The three marginal contributions are 20, 70 and 10, and they sum to 100, which
is $v(N)$. That is not a coincidence and it is the whole trick: in any order,
the marginal contributions are a telescoping sum, each term cancelling the
previous coalition's value, so they always add up to exactly the value of the
full corpus. Efficiency comes free from measuring this way.

But the order chose the winner. A walked into a room containing C and collected
70. Run the order A, B, C instead and A arrives first, collecting only 50. Every
source's number depends on when it showed up, and no arrival order is the real
one - the sources did not arrive, they were licensed.

**So use all of them.** With three sources there are $3! = 6$ orders, where
$3! = 3 \times 2 \times 1$ is "three factorial", the number of distinct
sequences you can put three things in. Compute every source's marginal
contribution in every order, and average.

<!-- fade: v5-shapley-by-hand -->
All six orders on the three-source table, computed with nothing but subtraction:

| Order | A gets | B gets | C gets |
| --- | --- | --- | --- |
| A, B, C | $50 - 0 = 50$ | $70 - 50 = 20$ | $100 - 70 = 30$ |
| A, C, B | $50 - 0 = 50$ | $100 - 90 = 10$ | $90 - 50 = 40$ |
| B, A, C | $70 - 50 = 20$ | $50 - 0 = 50$ | $100 - 70 = 30$ |
| B, C, A | $100 - 90 = 10$ | $50 - 0 = 50$ | $90 - 50 = 40$ |
| C, A, B | $90 - 20 = 70$ | $100 - 90 = 10$ | $20 - 0 = 20$ |
| C, B, A | $100 - 90 = 10$ | $90 - 20 = 70$ | $20 - 0 = 20$ |
| **Sum** | **210** | **210** | **180** |
| **Average** | **35** | **35** | **30** |

Read the columns. A's marginal contribution ranges from 10 to 70 depending on
when it arrives - a factor of seven, from the same eight measurements. The
average is 35. C's ranges from 20 to 40, averaging 30. And the three averages
sum to 100, because every individual row sums to 100 and the average of rows
each summing to 100 sums to 100.

**That is the rule.** Written out, with the sum over orders:

$$\phi_i = \frac{1}{n!}\sum_{\pi} \Big[\, v\big(P_i^{\pi} \cup \{i\}\big) - v\big(P_i^{\pi}\big) \Big]$$

where $n$ is the number of sources, $n!$ is the number of distinct orders of
them, $\pi$ (pi) ranges over those orders, and $P_i^{\pi}$ is the set of sources
that arrive before $i$ in order $\pi$. The bracketed quantity is source $i$'s
marginal contribution in that order. The whole formula is: average the marginal
contribution over every arrival order, treating all orders as equally likely.

**Check it against all four requirements**, on the numbers above.

*Efficiency:* $35 + 35 + 30 = 100 = v(N)$. Holds, by telescoping, in every game.

*Symmetry:* A and B are interchangeable, and both get 35. Holds, because
swapping two interchangeable sources maps the set of orders onto itself.

*Null player:* the padding source D from section 3 contributes 0 to every
coalition, so its marginal contribution is 0 in every one of the $n!$ orders,
and the average of $n!$ zeros is 0. Holds.

*Additivity:* each marginal contribution under $v + w$ is the marginal
contribution under $v$ plus the one under $w$, term by term, so the averages
add. Holds. Run the combined table from section 3 - none 0; A 70; B 50; C 20;
A+B 90; A+C 110; B+C 90; A+B+C 120 - through the same six rows and you get 55,
35, 30, matching $35 + 20$, $35 + 0$, $30 + 0$.

Four for four, and by the uniqueness theorem nothing else manages it.

**The same rule counted by coalitions instead of orders.** Averaging over $n!$
orders is how to understand it; enumerating $n!$ orders is not how to compute
it, and section 5 is about why. The reason the count collapses is visible in
the table above: A's marginal contribution is the same 10 in row "B, C, A" and
row "C, B, A", because in both cases A walks into a room containing exactly
$\{B, C\}$. The marginal contribution depends only on the *set* that arrived
first, never on the order they arrived in. So group the orders by that set:

$$\phi_i = \sum_{S \subseteq N \setminus \{i\}} \frac{|S|!\,\big(n - |S| - 1\big)!}{n!}\Big[\, v\big(S \cup \{i\}\big) - v(S) \Big]$$

where $S$ ranges over every subset of the sources other than $i$, $|S|$ is how
many sources are in $S$, and the fraction is just the count of orders that put
exactly $S$ before $i$, divided by $n!$. That count is $|S|!$ ways to arrange
the ones that came first times $(n - |S| - 1)!$ ways to arrange the ones that
came after.

Verify it on A, with $n = 3$ so $n! = 6$:

- $S = \varnothing$: weight $\frac{0! \times 2!}{6} = \frac{2}{6} = \frac{1}{3}$,
  marginal $50 - 0 = 50$.
- $S = \{B\}$: weight $\frac{1! \times 1!}{6} = \frac{1}{6}$,
  marginal $70 - 50 = 20$.
- $S = \{C\}$: weight $\frac{1}{6}$, marginal $90 - 20 = 70$.
- $S = \{B, C\}$: weight $\frac{2! \times 0!}{6} = \frac{1}{3}$,
  marginal $100 - 90 = 10$.

$$\phi_A = \tfrac{1}{3}(50) + \tfrac{1}{6}(20) + \tfrac{1}{6}(70) + \tfrac{1}{3}(10) = 16.667 + 3.333 + 11.667 + 3.333 = 35$$

Same answer, four terms instead of six, and the four terms reference exactly
four of the eight rows in the value table. That observation is the whole
feasibility argument in section 5.

Note the shape of those weights, because section 6 comes back to it and it is
the most surprising thing on this page. The weights are not uniform. The
extreme coalitions - the empty one and the all-but-$i$ one - carry $1/3$ each,
while the two singletons carry $1/6$. Shapley leans hard on the smallest and
largest coalitions.

```beat
id: v5-b4
type: completion
concept: d-shapley-axioms
# variants: blank the B column instead of the C column, which tests the
# middle-arrival case rather than the first-and-last cases. Or blank the
# "Sum" row and give the averages, which tests the division rather than the
# subtraction.
prompt: |
  A different three-source corpus. The value table is: none 0; A 40; B 30;
  C 30; A+B 60; A+C 80; B+C 50; A+B+C 90.

  Fill the four blanks in the arrival-order table. Each entry is the value of
  everything up to and including that source, minus the value of everything
  before it.

      Order        A gets        B gets        C gets
      A, B, C      40 - 0 = 40   60 - 40 = 20  90 - 60 = 30
      A, C, B      40 - 0 = 40   90 - 80 = 10  80 - 40 = 40
      B, A, C      60 - 30 = 30  30 - 0 = 30   ____            <- A
      B, C, A      90 - 50 = 40  30 - 0 = 30   50 - 30 = 20
      C, A, B      80 - 30 = 50  ____          30 - 0 = 30     <- B
      C, B, A      90 - 50 = 40  50 - 30 = 20  30 - 0 = 30
      Sum          240           ____          ____            <- C, D

  Then give the three averages and confirm they sum to 90.

  Answer with the four blanks in order (A, B, C, D), comma-separated, then the
  three averages.
answer: |
  A = 30 (that is 90 - 60, C arriving last into {A,B})
  B = 10 (that is 90 - 80, B arriving last into {A,C})
  C = 120 (the B column: 20 + 10 + 30 + 30 + 10 + 20)
  D = 180 (the C column: 30 + 40 + 30 + 20 + 30 + 30)

  Averages: A = 240/6 = 40, B = 120/6 = 20, C = 180/6 = 30.
  They sum to 90, which equals v(A,B,C). Efficiency holds.
rubric: |
  All four blanks must be exactly 30, 10, 120, 180, and the three averages
  exactly 40, 20, 30.
  The sum check to 90 must be stated. An answer with correct blanks whose
  averages do not sum to v(N) has an arithmetic error somewhere and should be
  failed rather than partially credited - the sum check is the built-in
  verification and the reason to run it.
  A learner who computes blank A as 90 - 50 = 40 has taken the marginal against
  the wrong predecessor set (they used {B,C} rather than {A,B}); re-deliver the
  paragraph explaining that the predecessor set is everything EARLIER in that
  row's order.
check: llm
# fade: v5-shapley-by-hand, stage 2 of 3. The blanked cells are the two
# last-arrival marginals, which are the ones that require reading the
# predecessor set off the order rather than off the table.
```

## Counting the runs: 4,096, not 479 million

The formula says to average over every arrival order. With 12 sources there are
$12! = 479{,}001{,}600$ orders. If each order needed its own trained model this
would be over before it started, and that is exactly the objection the method
is usually dismissed with.

<!-- refutes: V5-M1 -->
**You probably think the cost is the number of orders.** The rule is written as
an average over $n!$ permutations, the permutation count explodes, and
therefore the method is infeasible past about eight sources.

**Here is the prediction that fails.** If the cost were $n!$, the two orders
"B, C, A" and "C, B, A" would require different training runs. Look at what A's
marginal contribution is in each: $v(\{A,B,C\}) - v(\{B,C\}) = 10$ in both,
identically, because A walked into a room containing $\{B, C\}$ both times.
Every quantity in the formula is a difference between two entries of the value
table, and the value table has $2^n$ rows, not $n!$. Train each coalition once,
cache the number, and every one of the 479 million orders is answered by lookup.

**Here is why the wrong model is appealing.** The permutation form is the one
that makes the rule comprehensible, so it is the one everybody carries, and it
has $n!$ written directly in it. The critique is also correct in the setting
where it is usually made - see below.

**Here is what is actually true.** The number of models you train is $2^n$, once
each, and everything else is arithmetic over the cache. For 12 sources:

$$2^{12} = 4{,}096 \text{ trained models}, \qquad 12! = 479{,}001{,}600 \text{ orders answered by them}$$

Put a price on the 4,096. Take a 20-million-parameter model trained on 200
million tokens, the low end of the overnight envelope. Total floating-point
operations for one run are about six times parameters times tokens:

$$C = 6 \times (2 \times 10^7) \times (2 \times 10^8) = 2.4 \times 10^{16} \text{ FLOPs}$$

One H100 sustains roughly $4 \times 10^{14}$ FLOP/s on real training work, so
one run takes $2.4 \times 10^{16} / 4 \times 10^{14} = 60$ seconds. The full
enumeration:

$$4{,}096 \times 60 \text{ s} = 245{,}760 \text{ s} = 68.3 \text{ GPU-hours}$$

At $1.50 to $4.00 per GPU-hour as of mid-2026, that is **$102 to $273** for the
complete, exact, unapproximated value table over 12 sources. Section 6 triples
it for seeds and the number stays in the hundreds.

<!-- refutes: D7 -->
**Now the version of the objection that is actually right, and why it does not
apply.** The famous infeasibility critique of this method is not about
permutations at all - it is about granularity. Valuing every *document* in a
corpus means $n$ in the millions, $2^n$ is beyond astronomical, and no
enumeration, sampling scheme, or hardware budget rescues it. That critique is
correct and it has never been answered.

**Here is the prediction it makes that fails.** If the granularity were forced,
then a royalty system would have to value documents, and any source-level claim
would be an approximation to a per-document computation. It is the reverse. A
royalty system pays a *publisher*, a *catalogue*, a *feed* - a legal entity with
a bank account. There is no cheque made out to a document. Choose the unit of
attribution to match the unit of payment and $n$ is 12, not $10^6$. Group-level
valuation with groups as players has been in the literature since the original
2019 paper, demonstrated there on 146 groups.

**Here is why the wrong model is appealing.** Almost every paper in data
valuation works at per-example granularity, because the field's motivating
problems are data cleaning and example selection, where the example genuinely is
the unit. The infeasibility results are stated in that setting, correctly, and
then quoted out of it.

**Here is what is actually true, with the cliff located.** Cost is $2^n$, so it
doubles per source added. The 60-second run above gives:

| Sources | Coalitions | GPU-hours | Cost, mid-2026 |
| --- | --- | --- | --- |
| 10 | 1,024 | 17 | $26 - $68 |
| 12 | 4,096 | 68 | $102 - $273 |
| 15 | 32,768 | 546 | $819 - $2,184 |
| 18 | 262,144 | 4,369 | $6,554 - $17,476 |
| 20 | 1,048,576 | 17,476 | $26,214 - $69,905 |

The cliff is at about 15 sources, not at 10 and not at $10^6$. Below it,
exactness is affordable and there is no approximation for a critic to attack.
Above it, you approximate - and the next two subsections are the two ways.

```beat
id: v5-b5
type: compute
concept: d-semivalue-family
prompt: |
  You have 14 tagged sources. Each coalition costs one training run of 60
  seconds on one H100, and you plan to enumerate all of them exactly, once
  each, with no seed averaging.

  How many GPU-hours is the sweep? Give one number to one decimal place.
answer: 273.1
check: numeric(0.5)
```

**Approximation one: sample the orders.** Draw $m$ arrival orders uniformly at
random, compute every source's marginal contribution in each, and average over
the $m$ you drew instead of over all $n!$. The estimate is unbiased - the
average over a random sample of orders is an unbiased estimate of the average
over all of them, which is what an expectation over a uniform distribution
means. One sampled order gives you a marginal contribution for every source at
once, so you get $n$ estimates from $n+1$ cached value lookups.

How many orders do you need? Hoeffding's inequality answers this for any average
of independent bounded quantities. If every marginal contribution lies in a
range of width $R$, then after $m$ samples,

$$\Pr\big[\,|\hat{\phi}_i - \phi_i| \geq \epsilon\,\big] \leq 2\exp\!\left(\frac{-2m\epsilon^2}{R^2}\right)$$

where $\hat{\phi}_i$ (phi-hat) is your estimate, $\phi_i$ is the true value,
$\epsilon$ (epsilon) is the accuracy you want, and the left side is the chance
you miss by more than that. Set the right side to a failure probability
$\delta$ (delta) and solve for $m$:

$$m \geq \frac{R^2}{2\epsilon^2}\ln\!\frac{2}{\delta}$$

Worked. Marginal contributions in a corpus whose full value is 100 units lie
somewhere in a range of width at most 100, so take $R = 100$. Want to be within
$\epsilon = 5$ units with 95% confidence, so $\delta = 0.05$ and
$\ln(2/0.05) = \ln 40 = 3.689$:

$$m \geq \frac{100^2}{2 \times 5^2} \times 3.689 = 200 \times 3.689 = 737.8 \rightarrow 738 \text{ orders}$$

At $n = 30$ sources, 738 orders need at most $738 \times 31 = 22{,}878$
coalition evaluations, against $2^{30} = 1{,}073{,}741{,}824$ for the exact
version. A factor of 47,000, for an error bar of 5 units on a 100-unit pool.
Note what the bound depends on and what it does not: $R$, $\epsilon$, $\delta$.
The number of sources appears nowhere in it.

**Approximation two: change the weights, and lose an axiom.** Go back to the
coalition form of the rule and look at what those weights do at scale. For 12
sources, the weight on the empty coalition is $\frac{0!\,11!}{12!} = \frac{1}{12}$,
and the weight on each of the 462 coalitions of size 5 is
$\frac{5!\,6!}{12!} = 1.8 \times 10^{-4}$. One single value measurement -
$v(\varnothing)$ - carries 462 times the leverage on $\phi_i$ that any one
mid-size coalition's measurement does. Every value in the table is a noisy
measurement from a stochastic training run. Shapley's weights concentrate that
noise onto a handful of terms.

The **Banzhaf value** uses constant weights instead: average the marginal
contribution over all $2^{n-1}$ coalitions not containing $i$, giving every one
of them equal say.

$$\beta_i = \frac{1}{2^{n-1}}\sum_{S \subseteq N \setminus \{i\}} \Big[\, v\big(S \cup \{i\}\big) - v(S) \Big]$$

Worked on the three-source table. For A there are $2^{2} = 4$ coalitions not
containing A, with marginal contributions $50$, $20$, $70$, $10$ - the same four
numbers from section 4, now weighted equally instead of $\frac13, \frac16,
\frac16, \frac13$:

$$\beta_A = \frac{50 + 20 + 70 + 10}{4} = \frac{150}{4} = 37.5$$

$$\beta_B = \frac{50 + 20 + 70 + 10}{4} = 37.5 \qquad
\beta_C = \frac{20 + 40 + 40 + 30}{4} = \frac{130}{4} = 32.5$$

Now the price. $37.5 + 37.5 + 32.5 = 107.5$, and the pool is 100. **Banzhaf
violates efficiency**, here by 7.5%, and it is not a rounding artifact - the
constant weights are simply not the ones that make marginal contributions
telescope. To pay with it you renormalize: $37.5/107.5 = 34.88\%$ and
$32.5/107.5 = 30.23\%$, so $34.88, $34.88, $30.24 against Shapley's $35, $35,
$30. The two rules agree on the ranking and disagree in the third digit.

```beat
id: v5-b6
type: compute
concept: d-semivalue-family
prompt: |
  A different three-source corpus: none 0; A 30; B 60; C 40; A+B 70; A+C 60;
  B+C 80; A+B+C 100. Its exact Shapley shares are 21.67, 46.67, 31.67, which
  sum to 100.

  Compute the three Banzhaf values. For each source, average its marginal
  contribution over all four coalitions not containing it, weighting every
  coalition equally:

  $\beta_i = \frac{1}{4}\sum_{S} \big[ v(S \cup \{i\}) - v(S) \big]$

  Then add the three together to see the efficiency gap.

  Answer with four numbers, comma-separated: the three Banzhaf values in the
  order A, B, C, then their sum. Example format: `1, 2, 3, 6`
answer: "20, 45, 30, 95"
check: exact
```

**What you bought.** Equal weights provably maximize robustness to training
noise: with every measurement weighted the same, no single unlucky run can move
a source's number the way an unlucky $v(\varnothing)$ moves a Shapley value.
And Banzhaf's structure admits an estimator - maximum sample reuse - with a
property Shapley's does not have. Sample coalitions uniformly at random; each
sampled coalition $S$ feeds *every* source's estimate at once, contributing to
source $i$'s "with" average if $i \in S$ and its "without" average otherwise.
One training run informs all $n$ estimates rather than one difference. The total
number of evaluations needed is $O\!\left(\frac{1}{\epsilon^2}\log\frac{n}{\delta}\right)$
- the source count enters only inside a logarithm, so the budget is essentially
independent of $n$. In practice, 2,000 to 5,000 runs whether you have 20 sources
or 50.

**Which to use, as a decision rule.**

- **Up to about 15 sources: exact Shapley.** $2^n$ runs, cached, all four
  requirements satisfied, no approximation for a counterparty to attack. Under
  $273 at 12 sources before seeds. This is the PoC.
- **15 to about 50 sources: Banzhaf with maximum sample reuse.** Roughly 2,000
  to 5,000 runs regardless of $n$. Renormalize, and state in the schedule that
  you did, because renormalizing is how you restore efficiency after choosing
  a rule that does not have it.
- **Order sampling sits between and usually loses.** Its Hoeffding budget is
  independent of $n$ too, but at 30 sources the 738 orders above cost 22,878
  evaluations against MSR's few thousand, so above the cliff Banzhaf is
  cheaper for comparable accuracy and noisier data hurts it less.
- **Per-document, at any $n$: do not.** Wrong unit, and the infeasibility
  critique is correct there.

## The error budget: seeds before coalitions

Every entry in the value table came out of a training run, and training runs
are stochastic. This section puts a number on what that does to a split, and
the number decides whether the split is worth computing at all.

<!-- refutes: D16 -->
**You probably think a source's share is a property of the source.** Measure it
carefully, and you have a number you can put in a schedule and bill against
until the contract renews. Better tooling would give a more precise number, the
way a better scale gives a more precise weight.

**Here is the prediction that fails.** If a share were a property, retraining
every coalition with different random seeds - same data, same code, same
hyperparameters, only the initialization and the shuffle differ - would
reproduce the table and therefore the split. It does not. A single source's
measured marginal contribution to held-out loss can swing by more than its own
magnitude across seeds; per-document membership decisions in the verification
literature flip like coins under training randomness alone; and of everything
that perturbs an influence measurement, the *order* the data arrives in
introduces the largest variation - an ordering nobody chose deliberately and
nobody recorded.

**Here is why the wrong model is appealing.** Every instinct from deterministic
systems. The apparatus radiates determinism: loss curves are smooth, final
losses across seeds agree to three decimal places. It is only the *differences
between models* that are noisy, and this entire unit is built out of
differences between models.

**Here is what is actually true, with the arithmetic.** Each source's share is a
random variable, and you can compute its standard deviation from the value
table's noise directly, because the share is a weighted sum of table entries
with known weights.

Start from the per-run noise. Train one configuration several times with
different seeds and take the standard deviation of held-out bits per byte across
those runs; call it $\sigma_{\text{run}}$, and in these units - thousandths of a
bit per byte - a realistic figure for a 20 to 30 million parameter model is
$\sigma_{\text{run}} = 4$. If you average $k$ seeds per coalition before doing
anything else, each table entry has standard deviation

$$\sigma = \frac{\sigma_{\text{run}}}{\sqrt{k}}$$

because averaging $k$ independent measurements divides their standard deviation
by $\sqrt{k}$. With $k = 3$ seeds, $\sigma = 4/\sqrt{3} = 2.31$.

Now propagate. Rewrite $\phi_A$ from section 4 as a weighted sum of table
entries rather than of differences, by expanding each difference:

$$\phi_A = \tfrac{1}{3}\big[v(A) - v(\varnothing)\big] + \tfrac{1}{6}\big[v(AB) - v(B)\big] + \tfrac{1}{6}\big[v(AC) - v(C)\big] + \tfrac{1}{3}\big[v(ABC) - v(BC)\big]$$

Every one of the eight table entries appears exactly once, with coefficient
$\pm\frac13$ or $\pm\frac16$. For independent measurements, the variance of a
weighted sum is the sum of the squared weights times each variance - the squares
appear because variances add and a coefficient scales a standard deviation
linearly, hence a variance quadratically. So

$$\text{SD}(\phi_A) = \sigma \sqrt{\textstyle\sum_S c_S^2}
= \sigma\sqrt{4 \times \left(\tfrac13\right)^2 + 4 \times \left(\tfrac16\right)^2}
= \sigma\sqrt{\tfrac{4}{9} + \tfrac{1}{9}} = \sigma\sqrt{\tfrac59} = 0.745\,\sigma$$

With $\sigma = 2.31$: $\text{SD}(\phi_A) = 1.72$ units. So the split from
section 4 is properly written $35 \pm 1.7$, $35 \pm 1.7$, $30 \pm 1.7$.

**And now the result that is worth the algebra.** Compare against leave-one-out
on the same table. $\text{LOO}_A = v(ABC) - v(BC)$ is a difference of two noisy
entries, so its standard deviation is $\sigma\sqrt{1^2 + 1^2} = 1.414\,\sigma$.
The Shapley value's is $0.745\,\sigma$ - **roughly half the noise of the
leave-one-out number it is built from.** Averaging over orders is averaging, and
averaging suppresses noise. The combinatorics that look like the expensive part
of this method are also the part that buys precision back.

**The rule that follows: spend on seeds before coalitions.** Suppose a fixed
budget of training runs. You can spend it on more seeds per coalition, which
shrinks $\sigma$ as $1/\sqrt{k}$ in every term at once, or on more sampled
coalitions, which shrinks only the approximation error. Look at where the two
errors sit. The approximation error is already zero if you enumerated exactly -
which, below 15 sources, you can afford - and even when you sample, the
Hoeffding budget above buys $\epsilon = 5$ for a few hundred orders. Seed noise
appears in every single term of the sum and never goes away by itself. The
budget line for a 12-source exact split is therefore 4,096 coalitions times
$k$ seeds:

$$4{,}096 \times 3 \times 60 \text{ s} = 205 \text{ GPU-hours} \approx \$307 - \$819 \text{ as of mid-2026}$$

Three seeds is the floor, not the target. Section 7 computes how many you
actually need, and the answer comes from the split rather than from taste.

**And the choice that matters more than the estimator: use a smooth value
function.** Held-out bits per byte moves continuously - add a source and the
number shifts by however much the model improved, however slightly. Accuracy on
a multiple-choice benchmark does not: it moves only when a source flips an
argmax. Below a billion parameters, a four-choice benchmark sits at its 25%
chance floor, where on 14,000 items the sampling standard deviation alone is
$\sqrt{0.25 \times 0.75 / 14{,}000} = 0.4$ percentage points and seed variation
is larger still. Build a value table out of that and most coalitions return
identical values, every marginal contribution is zero or noise, and every source
looks like a null player. The table would be arithmetically valid and
scientifically empty. Choosing bits per byte over accuracy changes the answer
more than choosing Shapley over Banzhaf does.

```beat
id: v5-b7
type: self-explain
concept: d-semivalue-family
prompt: |
  A colleague proposes economizing: run each of the 4,096 coalitions exactly
  once with a single seed, arguing that with 4,096 measurements averaging into
  each source's share, the individual runs' noise will wash out.

  Explain in three or four sentences whether the argument holds, using the
  weight structure from section 5 and the propagation formula from this
  section. Then say which of the two errors - seed noise or combinatorial
  approximation - is already zero in this plan, and what that implies about
  where the budget should go.
answer: |
  The argument is partly right and wrong where it counts. Averaging does
  suppress noise - a Shapley value carries about 0.745 times the per-table-entry
  standard deviation, less than a single leave-one-out difference does - but
  the suppression is bounded by the weight structure and it does not fall with
  the number of coalitions the way the colleague expects. The weights are
  extremely uneven: at 12 sources the empty coalition carries weight 1/12 while
  each of the 462 size-5 coalitions carries 1.8e-4, so one unlucky run on one
  extreme coalition moves a source's share by 462 times what an unlucky
  mid-size run does. There is no law of large numbers protecting the terms that
  carry the most weight, because there are only a handful of them.

  The combinatorial approximation error in this plan is already exactly zero -
  all 4,096 coalitions are enumerated, nothing is sampled, nothing is estimated.
  So the entire remaining error is seed noise, and the only lever that reduces
  it is more seeds per coalition, which shrinks every term at once as
  1/sqrt(k). The budget goes to seeds.
rubric: |
  Must contain: (1) the Shapley weights are extremely uneven, so a few
  coalitions dominate and noise on them is not averaged away, (2) with full
  enumeration the approximation error is already zero, so all remaining error
  is seed noise, (3) therefore additional budget buys seeds, not coalitions.
  (2) and (3) = pass. All three = full credit.
  Crediting the averaging with reducing noise, without the weight-structure
  caveat, = partial; the learner has half the mechanism and will under-budget
  seeds.
  An answer asserting that seed noise is a defect to be removed by a longer
  run, a bigger model, or a better estimator = fail, diagnosing D16. Contribution
  is a random variable at every scale; the remedy is seeds and a reported error
  bar, never a cleaner run.
check: llm
```

## Gaming the split

Everything so far is a demo. This section is the difference between a demo and
a design, and it is where the four requirements stop being enough.

**The shell company attack, worked.** Source A - the trade publisher, currently
allocated $35 - incorporates two subsidiaries, A1 and A2, and licenses the
identical catalogue through both. Nothing about the content changes. The
licensee now sees four sources instead of three, and every coalition containing
either shell behaves exactly as it did with A in it, because the shells are the
same books.

That last sentence has a consequence worth pausing on: **the attacker needs no
new training runs to evaluate this.** Any coalition of the four new sources maps
onto a coalition of the old three by deleting duplicate shells, so the new value
table is a relabelling of the cached one. Sixteen rows, all read from eight.

| Coalition | Value | Read from |
| --- | --- | --- |
| none | 0 | $v(\varnothing)$ |
| A1 / A2 / A1,A2 | 50 | $v(A)$ |
| B | 50 | $v(B)$ |
| C | 20 | $v(C)$ |
| A1,B / A2,B / A1,A2,B | 70 | $v(AB)$ |
| A1,C / A2,C / A1,A2,C | 90 | $v(AC)$ |
| B,C | 90 | $v(BC)$ |
| A1,B,C / A2,B,C / A1,A2,B,C | 100 | $v(ABC)$ |

Now compute A1's share in the four-source game. With $n = 4$, $n! = 24$, and the
coalition weights $\frac{|S|!\,(4 - |S| - 1)!}{24}$ come out to $\frac14$ for
$|S| = 0$, $\frac{1}{12}$ for $|S| = 1$, $\frac{1}{12}$ for $|S| = 2$, and
$\frac14$ for $|S| = 3$. A1's marginal contribution is zero in any coalition
already containing A2 - the second copy adds nothing - so only four of the eight
terms survive:

<!-- fade: v5-shell-recompute -->

| $S$ | $|S|$ | weight | marginal contribution of A1 |
| --- | --- | --- | --- |
| $\varnothing$ | 0 | $\tfrac14$ | $50 - 0 = 50$ |
| $\{A2\}$ | 1 | $\tfrac1{12}$ | $50 - 50 = 0$ |
| $\{B\}$ | 1 | $\tfrac1{12}$ | $70 - 50 = 20$ |
| $\{C\}$ | 1 | $\tfrac1{12}$ | $90 - 20 = 70$ |
| $\{A2,B\}$ | 2 | $\tfrac1{12}$ | $70 - 70 = 0$ |
| $\{A2,C\}$ | 2 | $\tfrac1{12}$ | $90 - 90 = 0$ |
| $\{B,C\}$ | 2 | $\tfrac1{12}$ | $100 - 90 = 10$ |
| $\{A2,B,C\}$ | 3 | $\tfrac14$ | $100 - 100 = 0$ |

$$\phi_{A1} = \tfrac14(50) + \tfrac1{12}(0 + 20 + 70 + 0 + 0 + 10) + \tfrac14(0) = 12.5 + \tfrac{100}{12} = 20.833$$

By symmetry $\phi_{A2} = 20.833$, so the publisher collects
$20.833 + 20.833 = 41.67$ where it previously collected 35. **A 19% raise for
filing incorporation paperwork.**

Run the other two through the same procedure and the full picture appears:

| | Before | After | Change |
| --- | --- | --- | --- |
| A (as A1 + A2) | 35.00 | 41.67 | **+6.67** |
| B | 35.00 | 27.50 | **-7.50** |
| C | 30.00 | 30.83 | +0.83 |
| Total | 100.00 | 100.00 | 0 |

Efficiency held perfectly throughout - the shares still sum to 100 - and that is
the point. The pool is fixed, so A's raise came out of somebody's pocket, and it
came almost entirely out of B's. That is not arbitrary: B is A's substitute, the
source that shared 30 units of content with A. Adding a second copy of A makes B
even more replaceable, and B is charged for it. **The shell company attack is a
transfer from your competitors, and the parties it hurts are precisely the ones
whose content most overlaps yours.** In a real corpus that means the attack
concentrates its damage on the small rightsholders in a crowded niche.

**It gets worse with more shells, and the ceiling is computable.** Three shells,
$n = 5$: A1's marginal contribution is again zero in any coalition containing
another shell, so only the four coalitions drawn from $\{B, C\}$ contribute,
with weights $\frac{|S|!\,(5-|S|-1)!}{120}$ giving $\frac15$, $\frac1{20}$,
$\frac1{20}$, $\frac1{30}$:

$$\phi_{A1} = \tfrac15(50) + \tfrac1{20}(20) + \tfrac1{20}(70) + \tfrac1{30}(10) = 10 + 1 + 3.5 + 0.333 = 14.833$$

Three of those is 44.50. The sequence runs 35.00, 41.67, 44.50 for one, two and
three shells, and with $k$ shells it converges upward to $v(\{A\}) = 50$ - the
publisher's standalone value, which is what it would be paid if the redundancy
with B were ignored entirely. **Fragmentation is how a source unwinds the
redundancy discount, and 43% is the most it can steal here.**

**Why the four requirements do not stop this.** Look at what each one says.
Efficiency, symmetry, the null player, and additivity are all statements about
one game with one fixed set of players. Symmetry compares two sources *within* a
game. The null player condition tests a source *within* a game. Splitting A into
A1 and A2 does not violate any of them; it replaces the game with a different
game, and the four requirements say nothing whatsoever about how values in one
game relate to values in another. **Group-level valuation is not split-proof,
and the axioms are silent by construction, not by oversight.** No amount of
care in applying the rule fixes this, because the rule is being applied
correctly.

**The fix, in two layers.**

*Layer one, in the math.* A fifth requirement can be added: the value assigned
to a group of sources must not depend on how that group is partitioned into
registered entities. A recent construction - Faithful Group Shapley - builds a
value with provable immunity to fragmentation, at the price of being a different
value than the one four requirements forced. Adopt it and you are no longer
computing the Shapley value; you are computing something that agrees with it on
honestly registered corpora and refuses to reward splitting. That is a good
trade and it is one more piece of evidence that the axioms are a menu.

*Layer two, and the one to actually ship.* Put an identity constraint outside
the math. Before the game is played, every registered source is fingerprinted
and checked for near-duplicate overlap against every other registered source,
and any two registrants whose content overlaps above a threshold are merged into
a single player. This is not new machinery - it is the duplicate-cluster
detection the corpus pipeline already ran, pointed at registrants instead of at
documents, and it is why recording the duplicate clusters rather than silently
dropping copies was a payout decision from the beginning. Under a merge rule, A1
and A2 fingerprint identically, merge back into A, and the split returns to
$35, $35, $30 with no change to the value table at all.

The general statement, and it is the honest one to put in a pitch: current
attribution methods are not incentive-compatible on their own, and a deployed
mechanism needs a registration and identity layer that the mathematics does not
provide. Every commercial scheme in this space is vulnerable to this attack and
none of them discuss it.

```beat
id: v5-b8
type: completion
concept: d-split-gaming
# variants: blank the {B} and {B,C} marginals instead, which tests reading the
# predecessor coalition rather than recognizing the duplicate-shell zeros. Or
# give the four surviving marginals and blank the weights, testing the
# |S|!(n-|S|-1)!/n! count at n=4.
prompt: |
  Source C - allocated $30 in the honest split - tries the same attack,
  registering as two shells C1 and C2 that both offer the identical journal
  archive. The value table is read off the cached three-source one exactly as
  before: none 0; A 50; B 50; C 20; A+B 70; A+C 90; B+C 90; A+B+C 100.

  Fill the blanks in C1's marginal contributions. C1 adds nothing to any
  coalition already containing C2.

      S            |S|   weight   C1's marginal contribution
      {}            0     1/4     20 - 0 = 20
      {C2}          1     1/12    20 - 20 = 0
      {A}           1     1/12    90 - 50 = ____        <- A
      {B}           1     1/12    90 - 50 = 40
      {A,C2}        2     1/12    90 - 90 = 0
      {B,C2}        2     1/12    90 - 90 = 0
      {A,B}         2     1/12    100 - 70 = ____       <- B
      {A,B,C2}      3     1/4     100 - 100 = 0

      phi_C1 = (1/4)(20) + (1/12)(0 + ____ + 40 + 0 + 0 + ____) + (1/4)(0)
             = ____                                     <- C, D, E

  Give the five blanks in order, then state whether C's total across the two
  shells is higher or lower than the $30 it was allocated honestly, and by how
  much.
answer: |
  A = 40
  B = 30
  C = 40 (the same value as blank A, re-entered in the sum)
  D = 30 (the same value as blank B)
  E = phi_C1 = 5 + (1/12)(110) = 5 + 9.167 = 14.167

  Two shells collect 2 x 14.167 = 28.33, which is LOWER than the honest $30 by
  1.67. Fragmenting C loses money.
rubric: |
  Blanks must be exactly 40, 30, 40, 30, and 14.167 (accept 14.17 or 85/6).
  The verdict must be that C's total FALLS, to 28.33, a loss of about 1.67.
  All five blanks plus the correct direction = pass.
  Getting the arithmetic right but predicting a gain because "the attack
  inflates payouts" = fail. The attack does not always pay: it unwinds a
  redundancy discount, and C has no redundancy with anyone to unwind - it is
  the complementary source. Splitting a complement gives away the very
  scarcity that made it valuable. A learner who cannot say WHICH sources
  profit from fragmenting has not understood why the attack works and will not
  be able to design the defence.
check: llm
# fade: v5-shell-recompute, stage 2 of 3. The blanked cells are the two
# nonzero marginals and the final weighted sum; the zeros are given because
# recognizing them is taught, not tested, at this stage.
```

**Before you rank anything, check that ranking means something.** There is a
diagnostic that belongs in front of every split and appears in almost no
deployed system. The theoretical result behind it: this rule is provably the
right selection criterion when the value function is a monotone transform of a
*modular* function - one where each source contributes a fixed amount
independent of who else is present - and absent that structure there are games
where the ranking it produces is no better than arbitrary.

The operational version is a linear fit. Find the best additive approximation to
your value table: numbers $a_i$, one per source, such that
$\hat{v}(S) = \sum_{i \in S} a_i$ is as close as possible to $v(S)$ across all
$2^n$ coalitions, by least squares. Report how much of the table's variance that
approximation explains.

Worked on the three-source table. The best additive fit is $a_A = a_B = 40$,
$a_C = 35$, giving predictions and residuals:

| Coalition | actual | additive fit | residual |
| --- | --- | --- | --- |
| none | 0 | 0 | 0 |
| A | 50 | 40 | +10 |
| B | 50 | 40 | +10 |
| C | 20 | 35 | -15 |
| A, B | 70 | 80 | -10 |
| A, C | 90 | 75 | +15 |
| B, C | 90 | 75 | +15 |
| A, B, C | 100 | 115 | -15 |

The residual sum of squares is 1,200 against a total sum of squares about the
mean of 8,887, so the additive model explains 86.5% of the variance and 13.5%
is genuine interaction - the A-B overlap and the C-with-anything synergy that
the table was built to have.

How to read that number. High explained variance means the corpus is close to
modular, the ranking is reporting real structure, and one number per source is
an honest summary. Low explained variance means the interactions dominate,
which means a single scalar per source is compressing something that is not a
scalar, and the ranking can reorder under small, defensible changes to the
value function. At 86.5% the ranking here is trustworthy and the third digit is
not - which agrees with the error bars from section 6, arrived at completely
independently. Run this diagnostic before showing anyone a pie chart.

**And now the question that decides whether any of this ships.**

<!-- refutes: D20 -->
**You probably think attribution noise is a methods problem.** The estimates are
noisy today, the field is young, better estimators are coming, and
pay-per-contribution is what the technology is heading toward. Under that belief
the flat fees the industry has settled on are a symptom of immaturity, and the
job is to build the better estimator.

**Here is the prediction that fails.** If noise were only a methods problem,
then for any noise level there would exist a payment contract that beats a flat
fee, given a good enough estimator. There does not. There is a proven threshold:
when the attribution estimator's signal-to-noise ratio falls below it, the
welfare-optimal contract *is* a flat fee. Not "is approximately"; the optimum
collapses to it. No estimator improvement crosses that threshold, because the
threshold is about how much signal exists in the measurement, not about how
cleverly you process it. Only more signal - more seeds, bigger effects, a
different corpus - moves you across.

**Here is why the wrong model is appealing.** Builder's optimism, and the pie
chart. The output of this unit is three percentages that look completely
definite on a slide, and the error bars are not on the slide. It is also true
that the field is young and estimators are improving, which makes the wrong
inference locally plausible.

**Here is what is actually true, and this is the pitch-fatal one.** Whether a
contribution-proportional split beats a flat fee is an empirical question about
*your* corpus, *your* model scale and *your* noise floor, and you can answer it
with the numbers already in this unit. Compute the spread of shares across
sources and compare it to the noise on a single share, because a
contribution-proportional contract only differs from a flat fee to the extent
that sources differ:

$$\text{SNR} = \frac{\text{standard deviation of } \phi \text{ across sources}}{\text{SD}(\phi_i)}$$

Worked, on the running example. The shares are 35, 35, 30, with mean 33.33 and
deviations $+1.67$, $+1.67$, $-3.33$, giving a spread of

$$\sqrt{\frac{1.67^2 + 1.67^2 + 3.33^2}{3}} = \sqrt{\frac{16.67}{3}} = 2.36$$

Section 6 gave $\text{SD}(\phi_i) = 1.72$ at three seeds. So
$\text{SNR} = 2.36 / 1.72 = 1.37$.

**Read the verdict honestly: at three seeds, this corpus does not support a
contribution-proportional split.** The differences between sources are 1.37
noise units apart. Against the customary bar of 2 they are not resolvable, and
a schedule that pays A and C differently on this evidence is paying out a
difference it cannot demonstrate. The correct output is a flat fee - or more
seeds.

Compute how many more. To reach $\text{SNR} = 2$ you need
$\text{SD}(\phi_i) \leq 2.36/2 = 1.18$, and since
$\text{SD}(\phi_i) = 0.745\,\sigma_{\text{run}}/\sqrt{k}$:

$$\sqrt{k} \geq \frac{0.745 \times 4}{1.18} = 2.53 \quad\Rightarrow\quad k \geq 6.4 \rightarrow 7 \text{ seeds}$$

At 12 sources that is $4{,}096 \times 7 = 28{,}672$ runs, 478 GPU-hours, roughly
$717 to $1,912 as of mid-2026. Still an affordable number, and now you know what
it buys and why you are spending it.

Either answer is a real result. "We measured our own signal-to-noise and it
clears the bar, here is the split" is a product. "We measured our own
signal-to-noise, it does not clear the bar, and the welfare-optimal contract for
this corpus at this scale is a flat fee" is *also* a product, and it is a more
credible one than a pie chart with no error bars, because the person across the
table may already know the threshold result exists. Lead with the measured SNR
either way.

```beat
id: v5-b9
type: self-explain
concept: d-split-gaming
prompt: |
  You have run the full sweep. The shares across sources have a spread
  (standard deviation across sources) of 3.10 units, and each individual
  share has a standard deviation of 2.05 units from seed noise.

  Compute the signal-to-noise ratio, state which side of a bar of 2 it falls
  on, and say what you would put in front of a prospective data partner as the
  headline. Then say what changes if the spread had been 3.10 and the
  per-share standard deviation 1.20.

  Answer from the unit's text. Do not reference any sweep you have or have not
  run.
answer: |
  SNR = 3.10 / 2.05 = 1.51, which is below 2. At this seed count the corpus
  does not support a contribution-proportional split: the differences between
  sources are not resolvable against seed noise, and paying sources different
  amounts on this evidence pays out a difference that cannot be demonstrated.

  The headline is the measurement itself: "we measured our attribution
  signal-to-noise at 1.5, below the threshold at which contribution-proportional
  payment beats a flat fee, so the correct contract for this corpus at this
  scale is a flat fee." That is a result, not a failure - the threshold theorem
  says no better estimator changes it, only more signal does, and the honest
  options are more seeds or a corpus with more differentiated sources. Quoting
  the number required to reach SNR 2 - a seed count and its GPU-hour cost -
  turns the negative result into a priced decision.

  With per-share SD of 1.20 the SNR is 3.10/1.20 = 2.58, above the bar. Now the
  split itself is the headline, reported with its error bars, and the SNR is
  the credential that says the error bars were checked.
rubric: |
  Required: SNR = 1.51 (accept 1.5), identified as below the bar; and
  SNR = 2.58 (accept 2.6), identified as above it.
  Must also contain: the below-bar verdict is a reportable result and points at
  a flat fee, not a hidden failure.
  Both numbers plus the flat-fee verdict = pass. Naming the remedy as more
  seeds or more signal (never a better estimator) upgrades to full credit.
  An answer proposing to fix a below-threshold SNR with a better attribution
  method = fail, diagnosing D20. The threshold is a property of the signal
  available in the measurement; processing it more cleverly does not create
  signal.
  An answer that reports the split anyway with a caveat = fail. A caveat under
  a pie chart is not the same claim as a flat-fee recommendation, and this is
  the specific error the section exists to prevent.
check: llm
```

## The honest position

Volume 2 states the strongest case against its own subjects. Here is this
unit's, in three parts, none of them hedged.

**One: whoever picks the held-out set picks the split.** Section 3 proved
additivity by showing that adding a second held-out domain moved A's share from
35 to 55. That is the axiom working correctly, and it is also the attack.
Defensible, small changes to the value function move valuations substantially -
this has been demonstrated across the semivalue family, and the arithmetic in
this unit shows exactly why. There is no axiom about choosing $v$, and there
cannot be one, because the choice is a business decision: which held-out
documents represent what the licensee is buying. The rule is auditable *given* a
value function, and the value function is not.

The answer is procedural, not mathematical: publish the held-out set's
construction, its decontamination procedure, and its size, and commit to them
before running anything, exactly as a pre-registration commits a test before
seeing the data. That converts an unbounded degree of freedom into a disclosed
one. It does not eliminate it.

**Two: none of this has been shown to work at frontier scale, and the reason is
structural.** Every number in this unit came from actually retraining models.
That is what makes it exact, and it is also what caps it: the methods that scale
to frontier models have not been shown to correlate with counterfactual truth,
and the method that does correlate costs multiple training runs per query. The
honest claim is bounded to the scale where the counterfactual is computable, and
at that scale it is a real, exact, cheap measurement rather than an estimate.
Anyone who extends the claim past that boundary is selling something, and the
person across the table can check.

**Three: the market has not asked for this.** Across dozens of public content
deals, most of the value accrues to aggregators and documented creator royalties
round to zero. No deployed system documents an attribution algorithm at all -
production allocation is flat pro-rata, discretionary, or undisclosed. The
industry converged on flat fees, and section 7 supplies the uncomfortable
explanation: that convergence is consistent with nobody's signal-to-noise
clearing the bar, which would make it correct rather than lazy.

**What survives all three.** The output of this rule is a normalized allocation
carrying a fairness argument that a non-technical counterparty can read and
agree to in four sentences. No gradient method can answer "why is my cut 3.2%".
At source granularity the combinatorics collapse and the cost inverts in your
favour - the exact computation is cheaper at 12 sources than approximating it at
$10^6$. The evidence against the alternatives is stronger than the evidence
against this. And the direct precedent - the one paper that built exactly this,
with owners as players and log-density ratios as the value function - published
no cost figures and no code, and left source merging and splitting listed as an
open limitation.

So the scientific contribution of the build in the next section is specific and
defensible: a faithful reimplementation of that precedent, carrying the numbers
it omitted (what the sweep cost, in hours and dollars), running the diagnostics
it skipped (the modularity residual, the seed-noise bars, the measured SNR), and
demonstrating then fixing the attack it listed and did not address. That is a
smaller claim than "we solved attribution" and it is one you can defend line by
line.

## At the bench: the split

Here is what to build after this unit, and what it feeds.

**Build.** Take the 12 tagged sources from the corpus pipeline and the utility
table from the counterfactual unit. Enumerate all $2^{12} = 4{,}096$ coalitions,
training each at the configuration whose seed noise you measured - 3 seeds per
coalition to start, averaged into a single $v(S)$ before any Shapley arithmetic
touches it. Cache the table. Every number below is then arithmetic over that
cache and costs no further GPU time.

From the one cached table compute three allocations: exact Shapley, Banzhaf
(renormalized, with the efficiency gap reported rather than hidden), and
leave-one-out. Report them side by side, because their disagreement is the
argument for having done the work.

**Then attack your own split.** Take the source with the largest share and
recompute the allocation with that source registered as two shells, then three,
holding the value table fixed. Because duplicate shells introduce no new
content, this costs zero additional training runs - the expanded table is a
relabelling of the cached one, exactly as in section 7. Record the inflation
curve. Then apply the fix: run the corpus pipeline's near-duplicate clustering
across registrants, merge any pair above threshold into one player, recompute,
and show the split return to baseline.

**And run the diagnostics before believing any of it.** The additive-fit
residual, to say whether a single number per source is an honest summary. The
per-share standard deviation from the propagation formula. And the
signal-to-noise ratio, with the seed count required to reach 2 if you are under
it.

**Artifact.** One table with 12 rows - source, Shapley share, its standard
deviation, Banzhaf share, leave-one-out - plus four lines beneath it: the
additive-fit explained variance, the measured SNR, the verdict (split or flat
fee), and the total cost of the sweep in GPU-hours and dollars. That last line
is the one the precedent paper omitted, and publishing it is a contribution on
its own.

**What feeds forward.** The Shapley ranking becomes the target that the cheap
gradient methods in v8 are rank-correlated against - the number that says
whether attribution can be done without retraining. The identity and merge layer
becomes part of the audit story in v7, because a registration ledger that binds
an entity to a content fingerprint is the same object a pre-commitment ledger
needs. And the SNR verdict, whichever way it lands, is the headline of the
synthesis in v9.

```beat
id: v5-b10
type: self-explain
concept: d-split-gaming
prompt: |
  You are writing down, in advance, what the coalition sweep must record for
  the shell company attack to be demonstrable afterwards without buying any
  additional GPU time.

  State what the cached artifact has to contain, explain why the
  duplicate-shell version of the attack needs no new training runs while a
  version where the attacker genuinely partitions its catalogue into two
  disjoint halves does, and say how many coalitions that second version would
  require at 12 original sources.

  Answer from the unit's text; do not reference any sweep you have or have not
  performed.
answer: |
  The cached artifact is the full value table: one seed-averaged value per
  coalition for all 4,096 subsets of the 12 sources, keyed by the exact source
  set, together with the per-run seed noise used to average them.

  Duplicate shells need no new runs because the shells carry identical content.
  Any coalition of the expanded player set has the same content as the
  coalition you get by collapsing duplicate shells to one, so its value is
  already in the cache - the expanded table is a relabelling of the cached one,
  not a new measurement. That is what makes the attack cheap for the attacker
  as well as cheap to demonstrate.

  A genuine partition is different: half a catalogue is content the model has
  never been trained on in isolation, so v(half) and every coalition containing
  exactly one half is a value that was never measured. Splitting one source
  into two disjoint halves gives 13 players, so the exact table needs
  2^13 = 8,192 coalitions - double the original sweep, and only 4,096 of them
  are already cached.
rubric: |
  Must contain: (1) the cached artifact is the seed-averaged value for every
  coalition, keyed by source set; (2) duplicate shells map onto existing
  coalitions because the content is identical, so the expanded table is a
  relabelling and needs no training; (3) a genuine partition creates content
  combinations never trained, and at 13 players requires 2^13 = 8,192
  coalitions.
  (2) and (3) = pass, with 8,192 stated exactly. All three = full credit.
  Answering 2^12 = 4,096 for the partition case = fail; the partition adds a
  player and the count doubles, and getting this wrong is getting the
  economics of the defence wrong.
  An answer that cannot say why the duplicate case is free = fail; that
  property is the reason the attack is cheap in the field and the reason it
  must be defended against procedurally rather than by making it expensive.
check: llm
```

## What you can now do

Four capabilities, and the fourth is the one that travels.

**You can turn a table of measurements into a defensible allocation.** Average
each source's marginal contribution over every arrival order. Six orders by hand
at three sources; $2^n$ cached coalition values and a weighted sum at any
$n$. The shares sum to the pool by construction, interchangeable sources are
paid equally, sources that change nothing are paid nothing, and separate
evaluations add.

**You can state why that rule and not another, in four sentences a
non-technical counterparty will agree to.** Those four requirements admit
exactly one rule. That is the entire fairness argument, it is auditable from the
value table alone without access to the model, and it is the thing no gradient
method can supply.

**You can price it and bound its error.** $2^n$ runs, cached, doubling per
source, with the cliff at about 15 - $102 to $273 for 12 sources before seeds,
triple that with three. Above the cliff, Banzhaf with maximum sample reuse at a
few thousand runs regardless of $n$, renormalized, with efficiency declared as
the thing you traded. And the error bar comes from the same table:
$\text{SD}(\phi_i) = 0.745\,\sigma$ at three sources, roughly half the noise of
the leave-one-out number underneath it.

**And you can say when not to use it at all.** Run the additive-fit residual
before ranking anything. Run the shell company attack on your own split before
someone else does, and merge registrants by content fingerprint. Then compute
the signal-to-noise ratio and report which side of the threshold you are on,
because below it the welfare-optimal contract is a flat fee no matter how good
the estimator gets, and saying so is a stronger position than a pie chart
without error bars.

The three numbers in this unit that carry forward are the split, its standard
deviation, and the SNR. The first is the vision slide. The second is what makes
it survive a technical audience. The third is what makes it survive an
economist.

## Notation in this unit

<!-- canon-only -->

Reference, not reading. Return here when a symbol goes blurry; nothing below is
new.

| Symbol | Means | Typical value here |
| --- | --- | --- |
| $N$ | the full set of sources | $\{A, B, C\}$; 12 in the build |
| $n$ | how many sources there are | 3 in the worked example, 12 in the build |
| $S$ | a coalition - any subset of the sources | one of $2^n$ |
| $v(S)$ | value of a model trained on exactly $S$, in thousandths of a bit per byte of held-out text, relative to a model trained on none of them | 0 to 100 |
| $\varnothing$ | the empty coalition; $v(\varnothing) = 0$ by construction | - |
| $\phi_i$ | source $i$'s allocated share | 30 to 35 |
| $\beta_i$ | source $i$'s Banzhaf value - the same average with constant weights | 32.5 to 37.5 |
| $n!$ | $n$ factorial, the number of distinct arrival orders | $3! = 6$, $12! = 4.79 \times 10^8$ |
| $2^n$ | the number of coalitions, and the number of models trained | $2^{12} = 4{,}096$ |
| $\pi$ | one arrival order | one of $n!$ |
| $P_i^{\pi}$ | the sources arriving before $i$ in order $\pi$ | a subset of $N$ |
| $\sigma_{\text{run}}$ | standard deviation of held-out bits per byte across seeds, per run | 2 to 8 thousandths |
| $k$ | seeds averaged per coalition | 3 to 7 |
| $\sigma$ | standard deviation of one table entry after seed averaging, $\sigma_{\text{run}}/\sqrt{k}$ | 1 to 5 |
| $\epsilon$ | accuracy target for a sampled estimate | 4 to 5 |
| $\delta$ | failure probability for that accuracy target | 0.05 |
| $m$ | arrival orders sampled | several hundred to a few thousand |

The four formulas, restated with every symbol defined above:

$$\phi_i = \frac{1}{n!}\sum_{\pi} \Big[ v\big(P_i^{\pi} \cup \{i\}\big) - v\big(P_i^{\pi}\big) \Big]
\qquad
\phi_i = \sum_{S \subseteq N \setminus \{i\}} \frac{|S|!\,(n - |S| - 1)!}{n!}\Big[ v\big(S \cup \{i\}\big) - v(S) \Big]$$

$$\beta_i = \frac{1}{2^{n-1}}\sum_{S \subseteq N \setminus \{i\}} \Big[ v\big(S \cup \{i\}\big) - v(S) \Big]
\qquad
m \geq \frac{R^2}{2\epsilon^2}\ln\frac{2}{\delta}$$

The running value table, for checking any of the above:

| $S$ | $\varnothing$ | A | B | C | AB | AC | BC | ABC |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| $v(S)$ | 0 | 50 | 50 | 20 | 70 | 90 | 90 | 100 |

Shapley: 35, 35, 30. Banzhaf: 37.5, 37.5, 32.5 (sums to 107.5). Leave-one-out:
10, 10, 30 (sums to 50). Proportional to solo value: 41.67, 41.67, 16.66.

One deliberate simplification, flagged so it does not read as a contradiction
later. Everything here treats each coalition's value as one number with one
error bar, which assumes the runs behind different coalitions are independent.
They are not quite: a shared trunk checkpoint, a shared tokenizer, or a shared
held-out set correlates them, and correlated measurements make the propagated
standard deviation in section 6 an underestimate. At the scale of this build the
effect is small and the direction is known - your true error bars are a little
wider than the formula says. If you branch coalitions from a shared checkpoint
rather than training each from scratch, the effect stops being small, and the
counterfactual you are measuring changes as well.
