## The receipt

Think of a training run as a printing job at a commercial press.

You walk in with two numbers: how many pages, and how many copies. The shop
quotes you from those two numbers and a rate card. Nobody at the counter asks
what the book is about, whether it is any good, or whether anyone will read it.
A brilliant novel and four hundred pages of gibberish cost the same to print,
finish at the same hour, and come off the same machine.

That is not an analogy for how training is *billed*. It is how training *works*.
The machine performs a fixed amount of arithmetic determined entirely by how big
the model is and how much text goes through it, and the arithmetic is identical
whether the text is excellent or worthless.

The consequence is the thing to sit with. If quality is free, then quality is
where all the leverage is - it is the one improvement that does not appear on the
invoice. And in the other direction: if you want an experiment to be cheap, the
only lever you have is to print a smaller book or fewer copies. Everything else
is fixed.

The second thing to sit with is what this does to the scale of what one person
can attempt. Small print jobs are so cheap that you can run five hundred of them
to see which page mattered. That is not a metaphor either; it is section 3, and
it is the reason this book exists.

## One step, from scratch

The training loop is a very patient person tuning a very large mixing desk.

The desk has ten million faders. A track plays - a few thousand words of text -
and at every moment the desk makes a guess about what sound comes next. Someone
records how surprised the desk was by what actually came next. That single
number, the surprise, is the score for the whole pass, and it is the only
feedback that exists. Nobody says which fader was wrong.

Then the clever part. Working backwards through the wiring, the system figures
out, for every single fader, which direction that fader would have to move to
have made the surprise a little smaller. Not by trying each one - that would take
ten million passes - but by tracing the signal path in reverse, once. Every fader
gets a direction and a magnitude.

Then everything moves, all at once, by a tiny amount in its own suggested
direction. Roughly a thousandth of a fader's travel. It makes almost no
difference. That is a step.

Play the next track and do it again. Twenty thousand times.

Two things follow from that picture, both important later. First, no single step
matters, so no single step's data matters much either - which is why attributing
the finished sound to a particular track is genuinely hard, and why the whole
rest of this book exists. Second, the score is *surprise*, not *correctness*.
Nobody ever tells the desk what good music is. It only ever learns not to be
startled.

## The one line: C = 6ND

Here is the picture that makes the formula unforgettable.

Lay the training corpus out as a long ribbon of text. Lay the model out as a wall
of dials. Now: every single dial has to be consulted about every single word on
the ribbon. There is no shortcut, no index, no cache. Word by word, the whole
wall gets touched.

So the total work is the length of the ribbon times the size of the wall. That is
the shape of the formula, and it is the entire content of it. The number 6 is
bookkeeping: about a third of the work is making the guess and about two thirds
is figuring out which dials to blame.

What this picture gets you is a feel for the two levers. Halve the wall, halve
the work. Halve the ribbon, halve the work. They are symmetric and they are
independent, and there is no third lever hiding anywhere.

And it gets you the strategic insight of this entire volume. The experiments this
book needs are not "train one good model." They are "train the same model five
hundred times, each time with a different piece of the ribbon cut out, and see
what changes." Under that plan, the cost of the whole research program is five
hundred times the cost of one run. If one run costs three hundred thousand
dollars you cannot do it. If one run costs fifteen cents, you can do it on a
Tuesday. A small wall is not a compromise. It is what makes the question askable.

## Knobs are recipes, not theory

Take the learning rate as the size of the steps a hiker takes coming down a
mountain in fog.

**Warmup** is the first hundred metres, taken carefully. You have just stepped
out of the hut, you cannot see, and your sense of which way is downhill is built
from nothing. A confident stride here can put you off a ledge you never
recovered from. So you shuffle until you have a feel for the slope, then
lengthen your stride.

**Cosine decay** is a hiker who has been told exactly how long the walk is and
paces accordingly: long strides while there is ground to cover, shorter and
shorter as the destination approaches, so as to arrive settled rather than
overshooting. It works well. It has one cost: the pacing is a function of how far
along you are, so if someone stops you halfway and asks you to walk a different
route from here, you cannot - your stride length was already shortening in
anticipation of a destination that is no longer the one you are heading to.

**Warmup-stable-decay** is a hiker who walks at a steady comfortable pace for
almost the whole route and only slows for the last stretch. Because the pace is
steady in the middle, any point in the middle is a perfectly ordinary place to
stand. You can stop there, hand a photocopy of yourself to twenty other hikers,
and send each of them down a different final stretch. Twenty destinations for the
price of one long walk plus twenty short ones.

The catch, and it is the whole caution: all twenty of those hikers walked the
same first eighty-five percent of the route together. Whatever they saw on that
stretch, they all saw. If you were trying to find out what one particular view
contributes to the experience, sending clones down separate final stretches will
not tell you, because they all already saw it.

**On the loss curve.** The curve on your screen is the hiker's altimeter. It
tells you honestly whether you are going down, and it screams when you fall off
something. It cannot tell you whether you are on the right mountain. A hiker
descending the wrong peak sees a beautifully smooth altimeter trace all the way
to the bottom, and the trace looks *better* than the one from the right mountain,
because the wrong mountain happened to be easier. Every claim in this book is a
comparison between mountains, and altimeters do not do that.

## What repeating data actually costs

Think of reading a difficult textbook.

The first pass gives you the most. The second pass gives you nearly as much -
things you skimmed now land, connections you missed now appear, and honestly the
second read of a hard book is often where the learning happens. The third and
fourth are still worth doing. Somewhere around the eighth you are mostly
recognizing sentences rather than understanding them. By the fortieth you are
reciting.

That curve - lots, nearly as much, still useful, less, nothing - is not a
metaphor for what happens when a model repeats its corpus. It is a fair
description of the measured result. Four passes are worth about as much as four
different books of that size. Sixteen passes are worth about ten books. Forty
passes are worth about fifteen, and a hundred passes are still worth about
sixteen, because there is a ceiling and you have reached it.

The ceiling is the useful part. **A fixed corpus is worth, at most, about sixteen
times its own size in fresh text - ever, no matter how many times you read it.**
That single sentence is what turns a modest tagged collection of papers into a
legitimate training corpus rather than an excuse, and it is also what tells you
when to stop asking more of it.

And the thing to file away for later: rereading is also exactly how a passage
gets stuck in your head word for word. The student who read the chapter forty
times can quote it. That is the same dial, seen from the other side - it sets how
much the model learns *and* how much of the corpus becomes recoverable from the
finished model as evidence that it was there. One knob, two consequences, and
v6 is about the second one.

## The laptop and the node

Picture two workshops.

The rented one has three hundred and twenty craftsmen. The one on your desk has
one. That ratio is roughly right, and it is the only thing you need to hold.

The instinct is that the one-craftsman workshop is for toys. It is not, and the
reason is that the work here is not one large commission - it is five hundred
small identical pieces, each of which must be made from scratch to answer a
question about what one ingredient contributes. Three hundred and twenty
craftsmen can build you one magnificent cabinet overnight. One craftsman can
build you twenty-four small stools. If your question is "what does each leg
contribute to a stool," you want the stools.

**What overnight actually buys you**, on the desk: about thirty million dials
worth of model, on a ribbon of text about a billion and a half words long. That
is not a demo. That model is genuinely finished - it has seen more than fifty
words for every dial it has, which is more than the standard recipe calls for.
Its held-out score is a real measurement.

**On utilization.** The thing that trips people up is thinking of "how well am I
using the machine" as a bragging metric. Turn it around: the machine is rented by
the hour, and utilization is the fraction of each hour that produces model rather
than heat. Halve it and you rent the workshop twice as long for the identical
cabinet. It is not a scoreboard. It is the exchange rate between arithmetic and
money, and it is the only term in your budget that skill can move.

## Evaluation without self-deception

Two things go wrong here, and both are about measuring rulers with rulers.

**The first: measuring reading ability in words per page.** Suppose you want to
know who reads better, and you measure "how often were you surprised by the next
chunk of text." Now give one reader a book printed in tiny chunks and the other a
book printed in large chunks. The one with the tiny chunks is surprised less
often - each guess is easier - and looks like the better reader. They are not.
They had an easier task, chunk by chunk.

The fix is to stop counting chunks and start counting the thing that does not
depend on the printing: the raw characters of the text. Ask "how many bits of
surprise per byte of actual text," and now both readers are measured against the
same fixed object. That is bits per byte, and it is the only number in this
domain that survives comparison between two systems that chop text differently.
Any comparison against someone else's published model crosses that boundary.

**The second: the coin and the thumb on the scale.** Take a coin, flip it a
hundred times, count heads. Now change something about how you flip it and do it
again. You get a different count. Was that the change, or was it the coin?

Training runs are coins. Change nothing except the random seed - the same data,
the same code, the same settings - and the finished model scores differently.
Not wildly, but by an amount that is frequently *larger than the effect you are
trying to measure*. The whole discipline of this section is refusing to report a
difference until you know how big the coin's own wobble is.

Here is the picture that makes it stick. You are trying to weigh a feather on a
kitchen scale in a room with a draught. Weigh it once: you get a number. The
number is real, in the sense that the display showed it. It is also meaningless
until you have put the empty pan on the scale a few times and watched how much
the reading moves on its own. If the empty pan wanders by more than the feather
weighs, you have not weighed the feather. You have measured the draught.

And the good news, which is specific to working small: when the draught is too
big, you can just weigh it twenty more times. Averaging shrinks the wobble. At
these model sizes a repeat weighing costs four minutes and pocket change, so the
honest response to "your effect is under the noise" is to go and do twenty more
runs before dinner - not to argue, and not to publish anyway.

**One last picture, for the benchmarks.** Below about a billion parameters, a
multiple-choice benchmark is a room full of people guessing. Four options means
a quarter of them get each question right by luck. If your model scores
twenty-six percent where the last one scored twenty-five, you have not measured
an improvement; you have watched a coin land slightly differently. Reporting that
gap is not optimism. It is inventing a result, and anyone who has run these tests
will know it at a glance.

## At the bench: the overnight run

The thing you are building tonight is not a model. It is a ruler.

Train the same configuration twice, changing only the random seed, and the gap
between the two scores tells you how much this whole apparatus wobbles when
nothing has changed. Every claim you make for the rest of this book - this source
mattered, that method tracks the truth, this split is fair - is a claim that some
gap is bigger than that wobble. Without the ruler, none of them mean anything;
with it, all of them become checkable.

That is why a run producing one number and no spread has produced nothing usable.
It is a single weighing in a draughty room.

And then, before you go to bed, do the second half: work out on paper what the
same two runs would cost on the rented machine, and what five hundred of them
would cost. That number is a go/no-go. If the full sweep is a few hundred
dollars, everything in the rest of this book is affordable and the only remaining
question is whether you do it. If it is not, shrink the model tonight rather than
discovering the problem three units from now with a half-finished experiment.

## What you can now do

Three habits, and they are habits rather than facts.

**Price it before you run it.** Size times tokens times a rate card. Ten seconds
on the back of an envelope, and it converts "should we try this" from a
discussion into a division.

**Distrust the curve on the screen.** It is an altimeter. It tells you that you
are descending and it screams if you fall. It has nothing to say about whether
you are on the right mountain, and the prettiest trace in your logbook may be
the run where everything went wrong.

**Never report a difference without its wobble.** Weigh the empty pan. Say how
much it moved. Then say whether your feather beat it, and by how much. When it
does not, buy more weighings - which at this scale is an afternoon, not a
grant application.

The last one is the through-line of everything that follows. Contribution,
fairness, influence, evidence: every one of them is a gap between two numbers,
and a gap without a wobble is not a measurement. In a system that is meant to
move money to people, it is worse than that - it is a number someone will rely
on that nobody has checked.
