---
unit: u2
depth: more-intuition
---

## Shapes are type signatures

Forget the grid of numbers for a moment. Picture a sheet of graph paper with a
few dots drawn on it. A matrix is a machine that grabs the whole sheet at once
and deforms it - stretching, spinning, skewing, or flattening - while three rules
hold: the origin stays pinned where it is, every straight line stays straight,
and every set of evenly spaced points stays evenly spaced. That is the complete
list of what "linear" means, in pictures.

The dots go along for the ride. You never move a dot individually. You move the
paper, and wherever a dot's patch of paper ends up, that is where the dot is now.

Because the grid lines stay straight and evenly spaced, you only need to know
where two arrows go. Watch where the arrow pointing one unit east lands, and
where the arrow pointing one unit north lands. Everything else follows, because
every other point on the sheet is built from those two by walking east some
amount and north some amount, and the deformation preserves that arithmetic.
Those two landing spots are exactly what the columns of the matrix record. When
you read a matrix, you are reading a list of "here is where each basic direction
went".

The three examples in the main text are three ways to deform the sheet.

The rotation spins the whole sheet a quarter turn about the pin at the origin.
Nothing is squashed, nothing is stretched. Distances between dots are unchanged.
You could spin it back and recover exactly what you had.

The scaling stretches the sheet horizontally to twice its width while squeezing
it vertically to half its height. A drawn circle becomes an ellipse. Nothing is
lost - you could apply the reverse stretch and get the circle back - but the
sheet now emphasizes east-west differences and mutes north-south ones. This is
the useful picture for a learned weight matrix. Training is the process of
deciding which directions get amplified and which get muted.

The projection is different in kind, and it is worth sitting with. It takes the
whole two-dimensional sheet and presses it flat onto a single horizontal line, as
if closing a book. Every dot slides straight down onto the line. Two dots that
were far apart vertically now sit on top of each other, indistinguishable. You
cannot open the book back up, because the paper no longer records how high each
dot used to be. That is what "destroys information" means: not that a number got
smaller, but that two different inputs became the same output and nothing
downstream can tell them apart.

Pressing the book flat twice does no more damage than pressing it once. It is
already flat. That is the entire content of the observation that applying the
projection twice equals applying it once.

## Composition is multiplication

Two machines bolted together in a line. The sheet goes through the first, comes
out deformed, goes through the second, comes out deformed again. The combined
effect is itself a deformation of the same kind - the origin is still pinned,
lines are still straight, spacing is still even - so there is some single machine
that does the whole job in one pass. Multiplying the two matrices is the act of
building that single machine, in advance, so you never have to run the sheet
through twice.

That is worth stating plainly because it is why deep networks are not
automatically deep. Stack twenty of these machines and, absent any nonlinearity
between them, the whole stack collapses into one machine. Twenty layers of pure
matrix multiplication have exactly the expressive power of one. The bending that
linear maps cannot do is the reason nonlinearities exist, and it is why removing
them from a network does not make it a weaker network but a fundamentally
different and far more limited kind of object.

Order matters, and the physical picture makes it obvious rather than surprising.
Take a photograph. Rotate it a quarter turn, then stretch it horizontally: the
thing that gets stretched is what used to be the vertical direction of the
original photo. Now do it the other way. Stretch it horizontally first, then
rotate: the stretched direction gets carried around to vertical. Two different
final pictures, from the same two operations. You already know this from image
editing, from applying CSS transforms, from any pipeline where a filter's input
is the previous filter's output. Function composition has never commuted, in any
domain, and matrices are just function composition with the bookkeeping done
ahead of time.

The special cases where order does not matter have an equally physical
description. Two horizontal-and-vertical stretches commute because they act on
separate axes and neither one moves anything off the axis the other cares about.
Uniform scaling - blowing the whole photo up by a factor - commutes with
everything, because it does not favor any direction, so there is no direction for
a later operation to have moved into or out of. The pattern: operations commute
when they act on the same set of preferred directions. Rotation and axis-aligned
stretch do not, because rotation's whole job is to move things between the
directions the stretch cares about.

## The dot product is the whole game

The dot product answers one question: how much do these two arrows agree?

Picture two arrows from the same origin. Shine a light straight down onto the
second arrow's line, perpendicular to it, and look at the shadow the first arrow
casts on that line. If the two arrows point roughly the same way, the shadow is
long and lies in the same direction as the second arrow. If they are
perpendicular, the shadow collapses to a point at the origin. If they oppose, the
shadow falls on the far side, pointing backward.

The dot product is that shadow's length, scaled by the length of the arrow you
cast it onto. So it blends two separate facts: how aligned the arrows are, and
how big they are. Two long arrows that agree only somewhat can produce the same
number as two short arrows that agree completely. That conflation matters. When
you want alignment alone, you divide out the lengths, and that is what cosine
similarity is - the shadow measured as a fraction of the arrow's own length,
always between negative one and one.

The three cases are worth having as reflexes: strongly positive means pointing
the same way, zero means at right angles with nothing in common, negative means
pointing against each other.

Now the payoff for the rest of the book. Every time a model compares two things,
it is casting one shadow onto another. Not searching, not matching, not looking
up. Casting a shadow and reading its length. When the attention chapter builds a
big table of scores, every cell in that table is one shadow: how much of this
query lies along this key. Nothing more mysterious is happening anywhere in the
architecture, which is exactly why the architecture runs fast on hardware built
for multiplying and adding.

And a matrix-vector product is a batch of shadow measurements. The matrix holds a
collection of reference directions, one per row, and running a vector through it
reports how much of that vector lies along each reference direction in turn. A
trained weight matrix is a collection of directions the network learned to care
about, and applying it is asking the input "how much do you look like each of
these?".

## Softmax: why exponentials

You have a handful of scores and you need to turn them into a share of the pot -
a way of dividing one unit of confidence among the options, with the better-scored
options getting more.

The naive move is to just divide each score by the total. That breaks the moment
a score is negative, and scores are routinely negative. A negative share of the
pot is not a thing.

The next move is to square everything first, so it is all positive, then divide.
This is worse than it looks, and the reason is the whole point of this section.
Squaring throws away the sign, so an option the network scored as strongly
disfavored gets treated identically to one it scored as strongly favored. The
option the network liked least walks away with the largest share. The scoring is
not just distorted; it is inverted for half the range.

What you actually need is a way of turning scores into shares that never
reverses the ordering, never produces a negative, and never breaks - and one more
requirement that is less obvious and turns out to decide everything. Scores have
no absolute meaning. There is no zero point, no "neutral" score. Only the gaps
between them mean anything. So adding the same amount to every score has to leave
the shares completely unchanged.

That last requirement is the one that forces the answer, and there is a
physical way to feel it. You need a rule that turns differences into ratios: a
score that is one unit better should get some fixed multiple more of the pot,
regardless of whether the scores were 1 and 0 or 101 and 100. Compound growth is
the only thing that behaves this way. Money in an account at a fixed rate doubles
over a fixed interval, whether you started with ten dollars or ten thousand. A
constant time gap always means the same growth factor, never the same growth
amount. Exponentiating the scores gives exactly that: a fixed score gap always
buys a fixed ratio of shares.

Once you accept that framing, the shift-invariance stops being a curiosity. Add
the same amount to every score, and every share grows by the same factor, so
after dividing by the new total, nothing moved. This is the same reason inflating
everyone's salary by ten percent leaves the relative distribution untouched.

One last picture worth carrying. Suppose you skipped all this and just took the
highest-scoring option, winner take all. That is a cliff: the shares are one and
zero, and nudging a score changes nothing at all until it crosses the leader, at
which point everything flips at once. There is no local signal telling you which
way to nudge. Exponentiating and dividing replaces the cliff with a slope. Nudge
any score and the shares respond a little, immediately, in a direction you can
measure. Training needs the slope. That is the entire reason this function exists
rather than a comparison.

## Temperature: one knob, same evidence

Temperature is a contrast dial on a photograph.

The image is fixed. The pixels came out of the camera and nothing you do at the
dial changes what the camera saw. Turn contrast up and the bright regions blow
out to white while the dim ones crush to black. Turn it down and everything drifts
toward a flat grey. At maximum contrast you get pure black and white with nothing
in between; at minimum, a uniform grey rectangle carrying no information at all.

Cranking the contrast does not give you a better look at what was really there.
It gives you a more emphatic rendering of the same data. That is the whole
relationship between temperature and a model's output.

The thermodynamic name is not decorative, and the physical picture is exact. Put
a gas in a container. At high temperature the molecules have plenty of energy to
spare and spread themselves across every available state, including the
high-energy ones - the distribution is broad. Cool it down and they crowd into
the lowest-energy states available, because the energy to reach anything else is
no longer there. At absolute zero everything piles into the single lowest state.
Read "energy state" as "option" and "lowest energy" as "highest score" and you
have the model's sampling distribution, exactly. This is the same equation
physicists have used since the 1870s, which is why the parameter kept its name.

Two things are worth noticing about what the dial cannot do.

It cannot change the ranking. Turning contrast up and down never makes a dark
pixel brighter than a light one. Whatever the model scored highest stays highest
at every setting. The dial redistributes emphasis among a fixed ordering; it does
not reorder.

And it cannot add information. Turning the contrast to zero and looking at a grey
rectangle does not mean the camera saw nothing. Turning it to maximum and getting
stark black and white does not mean the camera was certain. The claim "I set it
to zero temperature so now I can see what the model really believes" is the claim
"I turned the contrast to maximum so now I can see what the camera really saw",
and it is wrong for the same reason. The full picture was already visible at the
neutral setting. Every setting is a rendering of one fixed thing.

## From derivative to gradient

You are standing somewhere on a hillside in dense fog. You cannot see the valley,
you cannot see the peak, you cannot see ten feet ahead. But you can feel the
ground under your boots, and you can tell which way is uphill and how steeply.

That local feel is the gradient. It is a direction and a steepness, and it is
purely local - it tells you about the patch of ground you are standing on and
absolutely nothing about where the valley bottom is. Walk a few steps and the
ground tilts differently, and you have to feel again.

Coming down the hill is a matter of repeatedly feeling for the steepest downhill
direction and taking a step that way. Feel, step, feel, step. That is gradient
descent, complete, and everything else in optimization is refinement on the size
and direction of the step.

Two failure modes fall straight out of the picture and are worth having as
physical intuition.

Steps that are too small: you descend, but you take a thousand of them to cover
ground you could have crossed in ten, and if the fog lifts before you get
anywhere you will conclude the hill is flat.

Steps that are too large: you stride so far that you overshoot the bottom of the
gully and end up partway up the far slope, higher than where you started. Feel
again, stride back, overshoot again. You can bounce between the walls
indefinitely, or bounce your way clean out of the gully and up the mountain. This
is what a divergent training run looks like from the inside, and the fix is a
smaller stride.

The most important thing the fog picture teaches is why the gradient does not
point at the destination. Imagine a long narrow gully - a drainage channel running
gently downhill, with steep sides. Standing partway up the side wall, the steepest
downhill direction is almost directly across the gully toward the opposite wall,
because the side walls are steep and the channel's own slope along its length is
gentle. So you stride across, overshoot, land partway up the other wall, and the
steepest direction is now back the way you came. You ricochet from wall to wall
while creeping only slowly along the channel toward where you actually want to
be.

Nothing is broken. The gradient told you the truth about the ground beneath your
feet, and the truth was locally unhelpful. The whole family of tricks in the
training chapter - momentum, adaptive step sizes - exists to notice that you have
been crossing back and forth without making progress and to average that
thrashing out into steady motion down the channel. Momentum is exactly what the
name says: a ball rolling down the gully accumulates speed along the channel while
its side-to-side oscillations cancel themselves out.

## The chain rule

A run of gears, or a chain of levers, or a stack of currency conversions - pick
whichever you find most vivid, they are the same picture.

Three gears in a line. Turn the first gear one notch and the second turns three
notches. Turn the second one notch and the third turns two. So turning the first
gear one notch turns the third by six. You multiply the ratios along the chain.
You do not add them, and you do not take the biggest one. Each stage passes along
whatever it received, scaled by its own ratio.

That is the chain rule in its entirety. Every layer of a network is a gear with
its own ratio - a local sensitivity saying how much its output moves per unit of
input movement. To find out how much the final loss moves when you nudge some
weight buried deep in the middle, you walk the chain from that weight to the loss
and multiply the ratios.

The currency version makes the units vivid. Dollars per euro, times euros per
yen, gives dollars per yen. Intermediate units cancel; the endpoints survive. A
gradient is a rate with units of "loss per unit of weight", assembled by
cancelling every intermediate quantity in the network. And that cancellation is
your error check: if the units do not cancel cleanly along the path, you have
skipped a stage or double-counted one.

Two consequences of gear ratios, both of which you will meet again by their
formal names.

If several gears in a row each have a ratio below one, the product shrinks fast.
Twenty gears at a ratio of one half apiece, and the last gear barely quivers when
you turn the first. Nudge a weight in an early layer and the loss simply does not
notice. That layer receives no usable signal and effectively stops learning. This
is the vanishing gradient problem, and it is the reason deep networks were hard to
train before residual connections gave the signal a direct route back that skips
past all those gears.

If the ratios exceed one, the product explodes instead. A tiny nudge at the front
produces an enormous swing at the back, and the training step overshoots into
garbage. That is the exploding gradient, and the crude but effective fix is to
clip the step's size.

One practical note that the picture explains. To compute the ratios you need to
know how hard each gear is being turned, because most real gears are not
constant-ratio - their sensitivity depends on where they currently sit. So you
must run the whole chain forward first, recording each stage's position, before
you can walk backward computing ratios. That is why training a network takes so
much memory: every intermediate value from the forward pass has to be kept alive
until the backward pass consumes it. The memory is not for the weights. It is for
the positions of the gears.

## Loss surfaces and why SGD works anyway

The fog-and-hillside picture is where most people's intuition goes wrong, and it
goes wrong specifically because the picture has two dimensions and the real thing
has a billion.

In two dimensions, a bowl-shaped dip is genuinely a trap. Walk downhill into a
small crater on the mountainside and you are stuck: every direction from the
bottom of the crater leads up, so the local rule "step downhill" has nowhere to
send you. Everyone extrapolates from this that a more complicated landscape means
more craters and more traps, and that training is a matter of luck about which
crater you fall into.

Turn it around. To be trapped, you need every single direction to lead uphill. In
two dimensions that is two conditions, and craters are common. In three, you need
all three - and notice that a mountain pass, the low point of a saddle between two
peaks, is not a trap at all: it is downhill along the road even though it is
uphill toward both summits. Stand at a pass and the local rule still knows where
to send you.

Now ask for a billion directions to all lead uphill simultaneously. Every extra
dimension is another chance for an escape route, and there are a billion chances.
Traps do not become more common as dimensions increase. They become
unimaginably rarer. Almost every place where the ground is momentarily flat is a
pass, not a crater, and passes leak.

This is the single most counterintuitive thing about high-dimensional spaces, and
it is worth holding onto: they are roomy. There is always somewhere else to go.

Two more pieces complete the picture.

The ground you are walking on is not measured perfectly. Each step, you feel the
slope using a different random handful of training examples, so the reading is
noisy. That noise is not a defect. Standing on a pass where the true slope reads
as zero, a noisy reading has some downhill component in it, and you stumble off
in that direction and keep going. The noise is what keeps you from stalling on
flat ground. It also shakes you out of narrow sharp gullies while leaving you
comfortably in broad shallow ones - and broad shallow basins turn out to be
precisely the ones that produce models that work on data they have not seen,
because sitting in a wide flat region means nothing terrible happens if the
weights are a little different from what they are.

And the good regions are everywhere. Not one valley to be found but an
uncountable number of comparably good ones, scattered across a space so large
that two runs starting from different random points have no reason to ever meet.
When two engineers train the same model with different seeds and get completely
different weights that perform identically, they did not fail to find the same
valley. They each found a perfectly good one, out of an unfathomable supply.

### The floor is not zero

The last piece of the picture is that this landscape has a floor, and the floor
is not at sea level.

Predicting the next word is not like solving an equation with one right answer.
Ask what follows "the meeting is scheduled for" and there are dozens of genuinely
correct continuations - a day, a time, a month, a room. A model that put all its
confidence on one of them would not be a better model; it would be a model that
was wrong about how language works. The spread is the correct answer.

So the landscape bottoms out somewhere above zero, at a level set by how much
genuine unpredictability is in the text itself. Descending to that level is the
whole job. Getting below it is only possible by memorizing the specific text you
trained on, which shows up immediately as a model that performs beautifully on
what it has seen and poorly on everything else.

The habit to build: when you see a training loss, do not ask how far it is from
zero. Ask how far it is from the floor.
