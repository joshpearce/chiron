## Two numbers and a gap

Here is the whole unit in one picture, and it is deliberately domestic.

You want to know how much a particular ingredient contributes to a stew. Not how
much it costs, not how much of it is in there, not whether tasters mention it -
how much it *contributes*. There is exactly one way to find out that settles the
question: make the stew without it, and taste both.

That is the entire idea. Everything else in the attribution literature is a
technique for guessing the answer without cooking the second pot, invented by
people whose pots take six months and forty million dollars. If your pot takes
twenty minutes, you cook the second pot.

Now the part that makes this a unit rather than a sentence. Cook the same stew
twice from the identical recipe, and it does not come out identical. Slightly
different heat, slightly different order of things going in, slightly different
day. Two cooks following the same instructions produce two stews that a taster
can tell apart. So when you compare the with-ingredient pot to the
without-ingredient pot and find a difference, you have a question: is that
difference the ingredient, or is it Tuesday?

The three rows of the opening table are three Tuesdays. The ingredient really is
in there, and the measured size of its effect came back 0.018, then 0.012, then
0.015. Nothing changed between the three except which pot got made first.

This is not a defect in the kitchen and no amount of care removes it. It is what
happens when the process you are measuring has randomness in it, which every
training run does. The rest of the unit is the cook's answer: make more pots,
average, and be honest about the spread.

## Contribution is a counterfactual

Two pictures, for why the size of the thing you remove decides whether you can
measure it.

**The wall.** You have a stone wall and you want to know how much each stone
holds up. For a large block near the base, the experiment is easy: take it out
and watch what happens. The wall sags visibly. For one pebble wedged in a joint
forty courses up, take it out and the wall does exactly what it was already
doing, which includes settling a fraction of a millimetre every day on its own.
The pebble's effect is real and it is smaller than the wall's ordinary daily
motion. No better ruler helps, because the wall is moving.

That ratio - effect size against the ordinary motion of the thing you are
measuring - is the only thing that decides whether a counterfactual is
measurable. Not the quality of the instrument, not the care of the
experimenter, not how long you look.

**The scale.** You want to weigh a letter. You have a bathroom scale. Stand on
it, note the number; hold the letter, stand on it again, subtract. In principle
this works, because weight adds. In practice the scale reads you three pounds
differently depending on where you put your feet, and the letter weighs an ounce.
You could repeat the procedure ten thousand times and average, and eventually the
ounce would emerge from the noise, and that is a true statement about statistics
and a useless statement about weighing letters.

Now weigh a suitcase the same way. One repetition is plenty.

Nothing about the scale changed. What changed is what you put on it. And the
lesson generalizes into the design principle this unit turns on: **if the thing
you want to measure is too small to measure, measure a bigger thing.** Which
sounds like giving up, until you notice that the bigger thing - a whole
publisher's catalogue rather than one paper - is also the thing you were going
to write the cheque to. The measurement got possible because it was aimed at the
right target all along.

## Designing the sweep

**Why differences are noisier than the things they are differences of.** This is
the least intuitive piece of arithmetic in the unit and it deserves a picture.

Two people each estimate the height of a tree, and each is typically off by a
foot. Ask them for the average of their two estimates and you get something
better than either - their errors point in random directions, so they partly
cancel, and the average is off by less than a foot.

Now ask them to estimate two different trees, one each, and you want the
*difference* in height. Person A's error and person B's error do not cancel;
they compound, because an error in A that makes tree one look tall and an error
in B that makes tree two look short both push the difference the same way. The
difference is off by more than a foot, not less.

That is the whole reason a leave-one-out contribution is harder to pin down than
either model's score. Both scores are quite stable. Their difference is where the
instability collects, and their difference is the only thing you care about.

The fix is the obvious one: get more people on each tree. Four estimates per tree
halves the error on each average, and the difference inherits the improvement.
The formula in canon is exactly this and nothing more.

**Why average first, then subtract.** Imagine three pairs of readings, and two
ways to summarize them. You can average the three "with" readings, average the
three "without" readings, and take one difference. Or you can take three
differences and average those. Both give the same answer, and they always will.

Where the two paths diverge is what you then call your uncertainty. Someone who
computed three differences is looking at three numbers that disagree, and it is
overwhelmingly natural to report how much they disagree as the error bar. But
that is the spread of *individual* differences, and what you reported was their
*average*, which is more reliable than any one of them - the whole point of
averaging. Report the spread of the individuals and you have quoted an error bar
for a number you did not report, and it is too wide, and real results disappear
underneath it.

**What a $z$ of 2 feels like.** It means the effect is twice as big as your
uncertainty about it. Not twice as big as the noise on one measurement -
twice as big as the uncertainty on the summary you are actually reporting. Below
1 you have nothing. Around 2 you have something worth saying with a hedge.
Above 6, which is where source 7 lands, you have a fact.

## What the sweep costs

The one thing to feel here is a shape, not a number.

Everybody in this field agrees that retraining is the ground truth and that
retraining is unaffordable. Both halves of that sentence are true at the scale
they are usually said about. The second half stops being true when you shrink the
model, and it stops being true *proportionally* - halve the model and you halve
the whole program, exactly, with no threshold to cross and no cleverness
required.

So the question "can I afford ground truth?" has an answer that is entirely
under your control, and the control is a dial rather than a switch. Turn it far
enough and a research program that reads as a national-lab budget line becomes
two evenings of a laptop being warm.

What you give up by turning the dial is not measurement quality. The subtraction
is just as exact on a small model. What you give up is *reach*: a fact about a
ten-million-parameter model is a fact about a ten-million-parameter model. That
is a real cost and the unit says so plainly. But it is a cost in scope, and the
alternative on offer - a claim about a large model with no counterfactual behind
it at all - is not a broader version of the same thing. It is a different kind of
object, and the honest word for it is a guess.

## The noise floor is a result

The instinct to fight here is the one that says a fuzzy measurement is a failure.

Think about a kitchen scale with a resolution of one gram. You put a grain of
rice on it and it reads zero. The scale did not fail. It told you something true
and useful: *whatever this weighs, it weighs less than a gram*. That is a
measurement, it is correct, and writing it down is not an apology.

The mistake would be to squint at a flickering display, decide it looks more like
0.4 than 0.2, and write down 0.4. Now you have converted a true statement into a
false one, and you have done it in the direction that makes your table look more
complete.

Half of a real contribution table looks like the grain of rice. Sources whose
effect is genuinely below what your experiment could resolve. The correct entry
is not a small number; it is the sentence *below the resolution of this
measurement, which is 0.005*. A reader can do something with that. A reader
cannot do anything with 0.001 except believe it.

And here is the part that turns a caveat into a strategy. Somebody, at some
point, is going to build a payment system on numbers like these. Whether that
system should pay per-contribution or pay everyone a flat fee depends on exactly
one thing: whether the contributions are bigger than the uncertainty about them.
That is the ratio you just measured. If it is large, you have a payment scheme
and a business. If it is small, you have discovered that the correct payment
scheme for this corpus is a flat fee - which is what the industry has largely
converged on, and now you know why, with your own numbers rather than a
suspicion.

Both answers are worth having. Only one of them is worth having *twice*, once as
a hope and once as a surprise in front of an audience.

## Subset regression

Here is the picture that makes datamodels ordinary.

You run a restaurant with twelve dishes and you want to know which ones people
are actually coming for. You cannot ask, because people do not know. So you run
a tasting menu three hundred times, and each night you serve a random half of the
dishes and ask the table for one score at the end. Not a score per dish. One
score for the evening.

At the end of three hundred nights you have three hundred rows: which dishes were
served, and how the evening went. Now sort the nights into the ones that included
the lamb and the ones that did not, average each pile, and look at the gap. If
lamb nights average better, lamb is carrying something.

The thing that makes this work - and it is the only subtle part - is that
everything else washes out. Across a hundred and fifty lamb nights, the soup was
present about half the time; across a hundred and fifty no-lamb nights, the soup
was also present about half the time. The soup contributes equally to both piles
and vanishes from the difference. Same for every other dish. Randomness did the
controlling for you, without anyone designing a control.

That is regression. Not a formula, not a fit, not a solver: sort into two piles
and compare. The formula only shows up when your nights were not balanced and the
piles are contaminated, and then the solver's job is to undo the contamination.
With a balanced design there is nothing to undo.

Two things worth noticing in the picture, because both come back later.

**Every night informs every dish.** A single evening's score contributes to the
lamb comparison, the soup comparison, and ten others simultaneously. This is why
three hundred tasting menus beat thirty-nine careful one-dish-at-a-time trials:
the effort is not partitioned. It is the same reason a well-designed A/B test
matrix beats a sequence of single-variable experiments.

**It answers a slightly different question than you asked.** Lamb's number here
is "how much does lamb add to a typical *half* menu." That is not the same as
"how much does lamb add to the full menu," and the two differ whenever dishes
interact - if the lamb and the beef are both the heavy main and diners only want
one, then lamb looks great on half-menus that lack beef and adds nothing to the
full menu that has both. Keep both numbers. Their disagreement is not an error;
it is the interaction, and it is the thing v5 is built to handle fairly.

## Scoring an attribution method

The question is: how do you grade a cheap method against an expensive truth?

The obvious grading scheme, and the reason it collapses, is worth feeling
concretely. Suppose you want to grade a wine critic. The honest test is to have
them predict something and then check. So you pick a thousand grapes, one at a
time, and ask the critic to rate each grape's contribution to the finished
bottle - and then you measure each grape's true contribution by making the wine
with and without that grape.

Every one of those thousand measurements is indistinguishable from zero, because
a grape is a grape. Your "truth" is a thousand random numbers. Now compare the
critic's list against it. The critic scores zero. So does a coin. So does a
perfect critic, if one existed. The test cannot tell them apart, and it is not
because the critics are bad; it is because you built a ruler out of noise.

The fix is not a better critic and not a better measurement of grapes. It is a
different question. Give the critic five *bottles* - each made from a different
large blend - and ask them to rank the bottles. Then taste the bottles and rank
them yourself. Now the thing being predicted is large enough to actually taste,
and a critic who is genuinely good will rank them mostly right while a coin will
not.

That is the Linear Datamodeling Score in one paragraph. Stop asking a method
about individual grapes. Ask it about whole blends it has never tasted, and check
whether it got the order right.

Two cautions, and both are ordinary once you have the picture. First, ranking
blends correctly does not prove the critic is right about any single grape - two
grapes that always appear together can have their credit traded between them
without changing a single blend prediction, and no amount of blend-tasting
separates them. Second, a critic graded on half-blends has been graded on
half-blends. If the bottle you actually care about contains eleven of the twelve
components, a critic who is excellent at six-component blends has not yet been
tested on your question.

## What ground truth buys

Every serious laboratory owns a set of calibration weights it never uses for
anything. They are not for weighing. They exist so that when the scale says one
kilogram, somebody can find out whether it is telling the truth.

That is what the table from this unit is. Not the product. Not the prototype of
the product. The calibration weight against which the product gets checked.

The reason this matters more here than in most engineering is that attribution
has no other check. An influence-scoring implementation produces a confident,
plausible, well-formatted ranking whether or not it is computing anything at all.
There is no crash, no exception, no obviously wrong output. A subtle sign error
in the middle of it yields a list that looks exactly as reasonable as the correct
list. Without a calibration weight you cannot tell a working implementation from
a broken one, and you will never find out, and neither will anyone reading your
paper.

The field's own measurements say this is not hypothetical. Somebody finally
weighed the standard methods against actual retraining and found that the ones
that scale came out at chance - not slightly disappointing, not
domain-dependent, at chance - while the one that agrees costs several full
training runs for every single question you ask it. And a separate group found
that a mismatch between the optimizer the theory assumed and the optimizer the
models actually used had been moving published numbers by up to a factor of
three. Both of those are exactly what a missing calibration weight produces:
years of confident numbers, none of them checked.

So the sequencing argument writes itself. Most people build the expensive
machinery first because it is the interesting part, and then have no way to know
whether their version works. Building the calibration standard first is boring,
cheap, and means every subsequent number you produce has a stated accuracy.

The one honest limit, in the same picture: a calibration weight certified at one
kilogram tells you nothing about how your scale behaves at ten tonnes. A
contribution measured on a small model is a fact about small models. That is a
real boundary and you say it out loud - and then you notice that it is a
boundary with a known price, since the way to learn whether it transfers is to
repeat the measurement one scale up and compare, which is an experiment rather
than an argument.

## At the bench: the leave-one-out sweep

What you are building here is short, and what makes it valuable is not the
number of rows.

Two evenings of a laptop staying awake produces a table with twelve lines on it.
That is the whole artifact. Anyone can produce twelve lines. What distinguishes
this table from the twelve lines a competitor puts on a slide is the header
above it: the noise floor, the number of seeds, whether the token budget was held
fixed when a source was removed, which papers "worse" was measured against, and
which of two overlapping sources the deduplication stage gave the shared material
to.

None of those five items is interesting. All five are load-bearing, in the sense
that changing any one of them changes the numbers underneath, sometimes by more
than the numbers themselves. A table without them is not a weaker version of a
table with them; it is a different kind of object, one whose rows cannot be
checked, cannot be reproduced, and cannot be defended by the person holding it.

Then there is one thing to do that costs nothing and that almost nobody does:
keep every result. Every run you perform is a pairing of an ingredient list with
an outcome, and later units need those pairings more than they need the table.
v5 wants to know what happens for combinations of sources you did not
specifically plan to test, and every one it can look up is a training run it does
not have to buy. Save the pairs, keyed by exactly which sources went in. It is a
few lines of bookkeeping now and it is the difference between v5 costing an
afternoon and v5 costing a week.

## What you can now do

Strip away the arithmetic and three habits remain, and they are the same three
habits in ascending order of how uncomfortable they are.

**Cook the second pot.** When someone asks what a source contributed, the answer
is not a theory, a score, or a similarity. It is a second training run. If you
can afford it, everything else in this field is optional; if you cannot, the
first thing to try is a smaller pot rather than a cleverer theory.

**Say how sure you are.** Every number in the table travels with the width of its
own uncertainty, and the ones that are narrower than their uncertainty get a
sentence rather than a digit. This is uncomfortable because it makes your table
look less finished than everyone else's, and the reason everyone else's looks
finished is that they left this out.

**Grade the shortcuts.** The moment you have a truth, every cheap method becomes
testable, and testing them is a two-hour job rather than a research project. The
resulting number - does the fast thing agree with the true thing on this corpus -
is the one measurement in the whole project that decides whether any of it
generalizes beyond the scale where you could afford to check.

The through-line is that this unit is the only place in the book where you get to
be certain about anything, and the certainty is purchased by being small. Every
later unit trades some of that certainty for reach: v5 for a defensible split,
v8 for methods that run at scale, v9 for a demo somebody can hold. Trades are
fine. Trades made without knowing what you started with are not, and this table
is what you started with.
