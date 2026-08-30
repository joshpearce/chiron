---
unit: v8
depth: more-intuition
---

## The bill for one honest answer

Picture the counterfactual as a physical procedure rather than an equation.
Somebody asks why a bridge stands up, and the only fully honest answer is to
build a second bridge without one of the girders and see whether it falls. That
is exhaustive, it is definitive, and it costs a bridge.

The contribution table was one round of that: twelve bridges, each missing one
girder, one afternoon of building. What changed in this unit is not the method
but the number of questions. Every time a visitor points at a different part of
the bridge and asks "why is that bit like that", the honest answer requires the
whole demolition programme again, because the question is about a different
place and the earlier demolitions were scored somewhere else. The cost does not
amortize. Twelve bridges per visitor, forever.

There is a second thing to feel, beyond the cost, and it is why per-query truth
is worse than the table rather than merely as bad. The table was scored by
weighing the whole finished bridge, which averages over thousands of
measurements and gives a steady number. A single output is one measurement -
does *this* rivet hold - and single measurements of a random process wobble.
Removing a girder changes the whole-bridge weight by an amount you can see over
the wobble; it changes any one rivet's behaviour by an amount usually buried in
it. Not only do you have to rebuild the bridge, you have to rebuild it many more
times before the answer means anything.

So you are looking for something already lying around the construction site
that correlates with what the demolitions would have shown. Not a substitute for
the truth. A gauge that has been calibrated against it.

## A gradient is a direction

Stand on a hillside in fog. You cannot see the valley, but you can feel the
slope under your feet, and that tells you which way is downhill *right here*.
That local downhill direction is a gradient. It is not a map and it does not
know where the bottom is; it is the tilt of the ground where you happen to be
standing.

Now the move that makes attribution possible. Each training example has its own
hillside. The model's parameters are a single position, and every example
evaluates the landscape from that same position but with its own idea of which
way is down - because "down" means "this example gets predicted better", and
different examples want different things. So at any moment you have one position
and thousands of arrows radiating from it, one per example, each pointing where
that example wants the model to go.

Training is the crowd's decision. Average the arrows, take a step in the average
direction, and repeat.

Ask what happens to one example when the crowd steps. If your arrow and mine
point roughly the same way, then a step in my direction carries you partway
toward where you wanted to go, and your loss falls even though nobody asked
about you. We are allies. If our arrows point in opposite directions, a step
toward me drags you backwards. Rivals. If our arrows are at right angles, my
step slides you sideways along your own contour line and your situation is
unchanged. Neutral.

The dot product is just the instrument that reads off which of those three you
are in, and how strongly. It is bigger when the arrows agree and when the arrows
are long, and both of those are the right behaviour: a long arrow is an example
that is badly wrong and pulling hard, and it moves everyone more.

**Why the reading is only approximately right.** The hillside is curved and the
arrow is straight. Step a tiny distance in a direction and the ground behaves
almost exactly as the arrow predicted. Step a long way and you have walked
around the curve of the hill, and the prediction made at the start no longer
describes where you ended up. That is the entire content of "first order, and
accurate for small steps". Every method in this unit is a straight-line
prediction on curved ground, and the size of the training steps is what decides
how good the straight line is.

## TracIn: influence as a sum over checkpoints

Think of training as a long walk and the query's loss as an altimeter reading
you take at every step. Over the whole walk the altimeter falls by some amount.
Each individual step nudged it a little, and the nudges add up to the total -
there is no leftover, because the total drop just *is* the sum of the drops.

Now ask who caused each nudge. Each step was taken because some particular
training examples were in the room, and their arrows determined the direction.
So attribute each step's nudge to the examples that were present for it, and
tally each example's total across the walk. When the walk ends, every example
has a number, and the numbers add up to the altimeter's total fall. That is a
genuine decomposition of the model's improvement on that query, not a score
somebody invented.

The obstacle is bookkeeping. Doing it exactly means remembering where you were
standing at every one of hundreds of thousands of steps, and nobody keeps that
many photographs of the landscape. What you have is a handful of snapshots. So
you use each snapshot to stand in for the stretch of the walk around it, which
is a real approximation and behaves the way approximations do: fine when the
terrain around a snapshot resembles the snapshot, poor when it does not.

**Three things the worked example is showing you, in pictures.**

*The early steps are the long strides.* At the start of training the learning
rate is high and each step covers ground; near the end the steps are shuffles.
An arrow aligned with the walk during a long stride moves things much further
than the same alignment during a shuffle. That is what the learning-rate weight
encodes, and dropping it is like counting a stride and a shuffle as equal
progress.

*Allies become rivals as you move.* An example's relationship to a query is not
a property of the two examples. It is a property of where the crowd is standing.
Early in the walk the ground tilts one way and two examples agree; a thousand
steps later the terrain is different and they disagree. A sign flip across
snapshots is the landscape changing, not an error.

*Almost all of the total comes from a few moments.* In the worked example one
snapshot carries essentially all of one example's score. That is the general
shape of these measurements: influence is spiky. A few examples at a few moments
dominate, and the vast majority contribute a shrug. If your output is a smooth,
plausible-looking spread across all twelve sources, check it, because the
physics of the thing does not usually produce smooth spreads.

**What TracIn refuses to assume, and why that is the point.** The rival family
of methods asks a sharper question - not "what happened along the walk you took"
but "where would the walk have ended up if this example had never existed" - and
to answer it they need a map of the whole valley's curvature at the destination.
Building that map requires the valley to be a bowl with one bottom, and the
valley is not a bowl. TracIn declines the sharper question and answers the one
its data can support: here is the walk that actually happened, and here is who
moved it. Modest, checkable, and honest about the fog.

## Making it affordable

Each arrow has one component per parameter, which for a real model means the
arrow is a list of hundreds of millions of numbers. You need dot products
between billions of pairs of these. Storing the arrows is out of the question,
and this is where two ideas rescue the whole enterprise.

**The shadow trick.** You do not need the arrows. You need the angles between
them. So take the whole forest of arrows and shine a light through it from a
random direction, and record the shadows on a wall. One random shadow tells you
very little: two arrows can have wildly different directions and cast similar
shadows by luck. But shine a few thousand independent random lights and record
all the shadows, and the collection of shadows pins down the angles almost
exactly. Averaging over many random viewpoints reconstructs the geometry that
any single viewpoint destroyed.

The remarkable part is what the required number of lights depends on. Not the
length of the arrows - not the parameter count, at all. It depends on how many
arrows you need to keep straight from one another. Ten million arrows in a
561-million-dimensional space need the same few thousand lights as ten million
arrows in a billion-dimensional space. Dimension is free; population is what
costs.

Two consequences to feel. First, everything has to be lit by the *same* lights.
Shadows cast under one set of random lights and shadows cast under a different
set are describing two unrelated pictures, and comparing them yields confident
nonsense rather than an error. Second, the reconstruction has a resolution
limit. Two arrows pointing in nearly the same direction are easy to confirm as
aligned. Two arrows nearly at right angles - which is what most pairs are, in a
space this big - have a true overlap so small that the shadow noise swamps it.
So the technique reliably tells you who the strong allies are and cannot be
trusted about the crowd of near-strangers. Since the strong allies are the whole
story in these measurements, that is a survivable limitation, and it is fatal
only if you try to present a complete pie chart of the near-strangers.

**The bundling trick.** You are paying sources, not sentences. Adding shadows is
the same as shadowing the sum, because shadows add the way the arrows do. So
rather than keeping ten million shadows, keep twelve running totals - one per
source - and throw each sentence's shadow onto its source's pile as it goes by.
The pile is the same size as one shadow no matter how many sentences went into
it. Ten million becomes twelve, and nothing was approximated in the bundling
itself; you only gave up the ability to ask about an individual sentence, which
was never the question a royalty system asks.

**On snapshot count.** More photographs of the same view is not more
information. Two snapshots taken close together in a steady stretch of the walk
show the same landscape from the same place, so their arrows agree with each
other and add the same number twice. And a snapshot from the first few minutes,
when the model is thrashing, contributes big confident arrows about a landscape
that no longer exists. What you want is coverage of genuinely different stages
of the walk, which is a question about spacing, not about count. Choosing a
training recipe with a long flat middle stretch helps here for a simple reason:
during that stretch the stride length is constant, so every snapshot is
comparable to every other and differs only in how far the walk has progressed.

## The reckoning

Everything so far works. That is precisely when to stop and ask what "works"
means, because the machinery produces a confident ranked list whether or not it
has any relationship to reality, and there is no error message for a wrong
answer.

Here is the shape of the evidence, stripped of the citations.

Somebody finally did the expensive experiment: built the ranked list the cheap
way, built it the expensive way by actual demolition, and compared. The cheap
methods that are fast enough to use came out **indistinguishable from shuffling
the list at random**. A method that did match the demolitions almost perfectly
turned out to cost several complete rebuilds per question, so it is a laboratory
instrument rather than a product. That is the field's central bind, and it is
not a temporary state of the art: the thing that can be checked cannot be
afforded, and the thing that can be afforded has mostly not been checked.

Then a second finding, worse in a quieter way. The derivation everybody had been
using assumed the walk was taken with one particular style of step, and the
walks were actually taken with a different style - one that treats each
direction of the landscape differently, damping down the ones it has learned are
jittery and amplifying the quiet ones. Once you take a step that way, "who was
aligned with the walk" has a different answer, because the walk itself was
reweighted before it was taken. Published numbers moved substantially when
somebody fixed the mismatch. The lesson is not that the field is careless. It is
that an assumption sitting in the third line of a derivation can be wrong for
years without anything visible breaking, because there is nothing to break -
the output is a plausible ranked list either way.

And a third, about the family with the fanciest mathematics. Those methods add a
correction for the curvature of the valley, which is a genuinely better idea and
is the right answer when the valley is a bowl. The valley is not a bowl, the
correction is approximated heavily to make it computable, and four independent
groups have now reported that the result does not do what it claims. The most
useful single fact about this field is the shape of that pattern: **the deeper
the mathematics, the weaker the empirical record.**

So what survives? One instruction, and it is not a hedge. Take your cheap
ranking, take the demolition table you can actually afford to build, and see how
well the two orderings agree. That agreement is a number. It is measurable, it
is small enough to fit in a sentence, and it is the entire warrant for
everything you say afterwards. High agreement and you have a cheap instrument
with a stated calibration. Low agreement and you have learned something real,
early, for a hundred dollars, instead of discovering it in a room full of people
who paid for the pie chart.

**One trap to see clearly before leaving.** These scores look like measurements.
They have decimal places, they sit in tables, and they invite comparison. They
are not quantities. Change the model and the score is computed in a different
space; change the step-size schedule and every term rescales; change the random
lights and the number moves on the same model with the same corpus. There is no
shared zero and no unit. What travels is the ordering, and only after the
ordering has been checked against something real. A score of 0.42 from one model
next to a score of 0.31 from another is two thermometers with unmarked scales,
held up side by side.

## Influence is not entailment

Two people can both answer "where did that come from" and mean entirely
different things, and the confusion is not sloppiness on anyone's part - it is
that English has one phrase for two questions.

**"Which page contains this fact?"** is a question about text. Open the books,
look for the sentence, point at it. No model is involved, no theory is involved,
and the answer is checkable by eye. A lawyer can hold it up.

**"What made the model able to say this?"** is a question about a history. It is
answered by the events of the training walk, not by the contents of any page.
The material that taught the model how to structure a claim, what register to
use, how to move from evidence to conclusion, contributes enormously and shares
not one word with the output. Meanwhile a page that states the fact in a
phrasing the model had already thoroughly mastered may have moved nothing at
all, because there was nothing left to learn from it.

The uncomfortable empirical result is that a term-counting search from the
1990s, which knows nothing about any model, beats a serious modern gradient
method at the first question at large scale. That is not a scandal. It is a
direct measurement finding that a tool aimed at the target hits the target
better than a tool aimed elsewhere.

Now the design consequence, which is where the money is. A payment scheme has to
pick.

Pay for containment and you are paying for resemblance. Everyone can verify it,
nobody argues, and you have systematically stiffed every source that shaped the
model without leaving fingerprints on its outputs. Those sources exist, they are
often the most valuable ones, and the music industry found exactly this failure
mode decades ago under a different name.

Pay for what moved the model and you are paying the right quantity and you will
spend your life explaining it. The recipient looks at the document you paid them
for, looks at the output, sees no connection, and asks what you are talking
about. Your answer has to be a correlation number, and it has to be true.

There is no third choice where one measurement satisfies both. The honest
product puts both on screen, labels which is which, and treats the disagreement
between them as the interesting thing rather than as an embarrassment to be
smoothed over.

## The ledger inside the run

A different use of the same instrument, and the one that pays for itself.

Instead of waiting until the walk is over and then asking about one destination,
carry a notebook during the walk. At each step, the crowd of examples in the
room determines the direction. You also have a compass needle pointing toward
"the model gets better on held-out material" - that is the validation direction,
and it is a single arrow you can compute once per step. Look at each example in
the room, see how much its arrow agrees with the compass, and write that amount
next to its name.

Do this for every step and finish the walk with a completed ledger: how much
each source, across the whole run, contributed to the model getting better.
Divide by the total and you have percentages. A royalty split, produced as a
byproduct of a training run you were doing anyway, for a few percent extra
compute.

**Why it is the same instrument.** Compare what goes in the notebook - how much
this example's arrow agrees with the compass, scaled by the stride length - to
what the previous sections computed - how much this example's arrow agrees with
the query's arrow, scaled by the stride length. It is the same operation. The
only differences are what you point at (a held-out set instead of one output)
and when you look (every step instead of a few snapshots). Same approximation,
same blind spots, same need for grading.

**Why calling it Shapley is true and misleading at once.** At each step, the
total improvement is a plain sum with one term per example in the room, and no
term depends on who else is there. When a total splits into independent
contributions like that, the fair-division question has a trivial answer: each
gets its own term, and any sensible fairness rule agrees. So yes, it is a
Shapley value - of a game so simple that the fairness machinery has nothing to
do.

The fairness machinery earns its keep only when contributions interact. That
happens at the next level of detail, where two examples in the same room have a
joint effect that is not the sum of their separate effects, and the interaction
has to be divided. Splitting it evenly is what the axioms prescribe, and that is
the first place they do any work.

**And here is what the label does not buy.** Picture two sources holding
essentially the same material. The demolition approach handles them correctly
because it actually removes things: pull either one and the model shrugs, since
the other still covers the ground; pull both and it stumbles. The notebook
cannot see this at all. It watched a walk in which both were present, wrote down
each one's agreement with the compass independently, at steps that mostly did
not even contain the other, and credited each in full for material that only one
of them needed to supply. The notebook is not wrong about what it recorded. It
recorded a different thing.

Say that out loud and the ledger is a strong result: a defensible per-source
split, computed for a few percent overhead, at a scale nobody else has
instrumented. Call it the demolition answer and the first person in the room who
knows this literature will ask about duplicate sources, and the honest answer
will be that you did not check.

## What to trust

Four questions, four instruments, and the discipline is refusing to let an
answer from one instrument wear another's authority.

The demolition table is the standard. It is expensive, it is slow, it exists
only at small scale, and it is the only thing in the building that can tell you
whether anything else is lying. Treat it the way a metrology lab treats its
reference weight: it does not go out to customers, it does not do daily work,
and everything else is checked against it.

The gradient ranking is the fast gauge. Cheap after a one-time setup,
milliseconds per question, and worth exactly its calibration against the
standard. A gauge with a calibration certificate is a real instrument. The same
gauge with no certificate is a dial that moves.

The in-run ledger is the same gauge pointed at a different target and read
continuously instead of on demand. Same calibration requirement.

The verbatim search is a different instrument entirely, measuring a different
thing, and it is exact at what it measures. Nothing wrong with it except the
temptation to let it answer a question it was not asked.

Three habits follow, and they are habits rather than theorems.

Never state a cheap number without its calibration. "This source drove that
output" is unfalsifiable and everyone has heard it. The same sentence with a
correlation, a scale, and a design attached is a claim someone can attack, which
is what makes it worth more.

Never let the reference weight leave the lab. The calibration you measure at
small scale is the last honest number you will ever have about your method,
because the moment you scale past where rebuilding is affordable, no new
calibration is possible. Treat that number as precious rather than as a
stepping stone toward something better.

Say which question you answered. A system that shows a verbatim match and calls
it attribution has answered the easy question and put the hard question's label
on it.

## At the bench: the gradient ledger

Three things get built, and they are the same instrument in three
configurations.

First, the fast gauge. Pick a few snapshots from the steady middle stretch of
the training walk, fix your random lights and write the seed down where you
cannot lose it, then sweep the corpus once per snapshot, throwing each
sentence's shadow onto its source's pile. You finish with a dozen piles per
snapshot, small enough to email. Use the step style the model was actually
trained with rather than the one the textbook derivation assumes, because that
mismatch is the field's known dominant error and correcting it is a few lines.

Second, the calibration. Rank the twelve sources by the gauge, rank them by the
demolition table, and see how well the orderings agree. Then do the more
statistically serious version, predicting the outcomes of runs you held back and
correlating those. Report both, with the design attached, because a correlation
without a design is a decoration.

Third, the notebook. Rerun the small model with the ledger instrumented, and
measure your own overhead rather than quoting somebody else's.

The artifact is one table, twelve rows, three columns of numbers - the demolition
split, the gauge's ranking, and the notebook's ledger - with the agreement
between each cheap column and the expensive one written underneath, and one
sentence saying which cheap column you would ship. Plus the details that make
any of it readable later: which snapshots, which random-light seed, which step
style, which held-out set. Leave out the seed and the whole index becomes
confident noise with nothing to indicate it.

That table is the centre of the demo. One question, three answers, the verbatim
match beside them for contrast, and a number underneath saying how much to
believe the fast ones. The contrast is what makes the demo interesting; the
number is what keeps a technical audience in the room.

## What you can now do

Four things, and one sentence.

**You can trace influence from the walk you already took.** Every example casts
an arrow saying where it wants the model to go. Two arrows agreeing means a step
toward one helps the other, and the amount of help is the agreement scaled by
the stride length. Add that up over a few snapshots of the walk and you have a
per-example influence, with no assumption that the valley is a bowl and no need
to map its curvature.

**You can make it fit on a laptop.** Random shadows preserve angles with a
resolution that depends on how many arrows you keep track of and not on how long
they are, provided every arrow is lit by the same lights. Bundling shadows by
source is exact, not approximate, and takes ten million piles down to twelve.
The two things to get right are the seed for the lights and the step style the
model was trained with.

**You can produce a royalty ledger as a byproduct.** Carry the notebook during
the walk, compare each example's arrow against the compass pointing at held-out
quality, and finish holding percentages. And you can say precisely what those
percentages are a fair split *of* - the walk that happened - rather than
pretending they answer the question about the walk that did not.

**And you can say what any of it is worth.** One number: how well your cheap
ordering agrees with the demolitions you could afford to run. Everything else in
this unit is machinery, and machinery is cheap. Only that number is expensive,
and it is the only part nobody can take from you.

The sentence: **a gradient method is a guess about what the demolition would
have shown, and the agreement between the guess and a demolition you actually
ran is the whole of the evidence that the guess is any good.**
