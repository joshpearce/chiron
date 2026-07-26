---
unit: u5
depth: more-intuition
---

## Cross-entropy: turning a distribution into one number

Picture the model as a bookmaker. Before each token it publishes odds across
every word in the language. Then reality happens: one word actually occurs. The
loss is the bill the bookmaker pays, and cross-entropy sets the payout schedule.

The schedule has one property that determines everything about the resulting
model: the bill for being confident and wrong is unbounded. Assign 1% to the
word that occurs and you pay a moderate amount. Assign 0.001% and you pay a lot.
Assign literally zero and the bill is infinite. Meanwhile the reward for being
confident and right tops out at zero. There is no prize for certainty, only a
penalty for misplaced certainty.

A bookmaker facing that schedule learns humility as a survival strategy. It
never publishes zero for anything remotely possible, because a single occurrence
would wipe it out. It spreads a little mass over the long tail and concentrates
the rest where it is genuinely sure. That is precisely the behavior you observe:
ask a model for the next word after "the capital of France is" and it does not
put 100% on " Paris", it puts something like 95% there and scatters the rest
over " a", " located", " the", " one". Those leftovers are not a bug in the
model. They are the shape of the payout schedule showing through.

Now the other half. Why is training so much faster than it has any right to be?
Because a single document is not one exam question, it is thousands. Feed in a
2,000-word article and the model is simultaneously graded on its guess at word
2 given word 1, word 3 given words 1-2, and so on to the end. One pass, two
thousand independent verdicts, all in parallel. The bookmaker is not settling
one bet per day. It is settling every bet on the board at once, every time.

And one thing this picture makes obvious: the bookmaker is never allowed to see
its own previous guess. Each bet is settled against what really happened next,
not against what the bookmaker said would happen. So training never rehearses
the situation the model faces in production, where it must build on its own
output. That gap is a real one, and it is where a lot of generation weirdness
comes from.

## Perplexity, and the floor that is not zero

Perplexity turns the loss into a number you can picture: how many equally-likely
options is the model effectively choosing among? Perplexity 1 is a model that
knows exactly what comes next. Perplexity 50,000 on a 50,000-token vocabulary is
a model that knows nothing at all. Real models land in single digits, which is
worth sitting with: at every position, out of fifty thousand possibilities, a
good model has narrowed it to the equivalent of picking among about eight.

Here is the picture for why zero is the wrong target. Imagine grading a
weather forecaster. Some days genuinely are 60/40. A forecaster who says "60%
chance of rain" on those days and is right 60% of the time is doing the best job
physically possible. A forecaster who says "100% rain, guaranteed" every time
scores worse on average even though they are sometimes exactly right, because
the days they miss cost them enormously. Now: is there some amount of extra
meteorological skill that lets a forecaster hit zero error? No. Not more
satellites, not a better model, not more compute. Some of the uncertainty is in
the weather, not in the forecaster.

Language has the same structure and quite a lot of it. "I went to the ___" is
genuinely, irreducibly open. Store, park, doctor, gym, bathroom, movies, wrong
address. A perfect model of English is not one that picks the right one. It is
one that assigns each the frequency it actually has. And that model, the ideal
one, still has substantial loss, because the loss is measuring the openness of
the language, not the ignorance of the model.

So the mental image to carry: the loss curve is not descending toward the floor
of the room. It is descending toward a shelf partway down, and the shelf is
where language itself sits. Every published loss number is a distance above that
shelf. A number below the shelf means something has gone wrong, and the usual
thing that has gone wrong is that the model was tested on text it had already
seen.

## Backprop as blame assignment

A production incident: the checkout request took 400 milliseconds too long. You
want to know how much each of the forty services in the path contributed, so you
know where to spend engineering time.

The bad approach is to speed up one service by a millisecond, rerun the request,
and see how much the total improved. It works, and it tells you the true
sensitivity to that service. But you need to do it forty times, once per service,
and in a network with a hundred billion knobs the equivalent is a hundred billion
reruns. That is forward mode, and it is why nobody differentiates that way.

The good approach is the one every tracing system already implements. Start at
the end with the full 400ms of blame. Hand it backward. The serializer says: "of
the time I took, this fraction came from waiting on the database call, and this
fraction was my own work" and passes the appropriate share of blame to each. The
database says the same to its connection pool. Blame flows backward through the
trace, splitting at every fan-out and summing at every fan-in, and when it
reaches the leaves, every component has its exact contribution. One trace. Not
forty runs.

The rule at each node is local and small: multiply the blame you were handed by
how sensitive your output was to each of your inputs, and pass the products
upstream. A node needs to know nothing about the rest of the system, which is
why this composes across arbitrary architectures. It is also why you can bolt a
new layer type into a framework by writing exactly two functions: what it
computes, and how it splits blame.

Two consequences worth internalizing. First, the backward pass costs about the
same as the forward pass, because blame-splitting at each node is the same size
of arithmetic as computing that node's output was. Training is roughly three
times inference, not a thousand times. Second, blame-splitting needs to know
what each node's inputs *were*, so the forward pass has to keep its intermediate
results around until the backward pass collects them. That is the entire reason
training a model needs so much more memory than running one, and why the standard
trick when you run out is to throw away the saved intermediates and recompute
them on the way back, paying time to buy memory.

There is one more thing this picture gets right that the phrase "the model
learns" gets wrong. Nothing in the backward pass is deciding anything, and no
node knows what the loss means. It is arithmetic flowing backward through a
graph. The intelligence, such as it is, is entirely in the fact that doing this
ten million times moves a hundred billion numbers in a consistent direction.

## Backprop worked: two layers, real numbers

The numbers in canon are the whole story, but here is the shape to hold onto
while you read them.

Think of the two weight matrices as two mixing boards. The input arrives as a
pair of levels. The first board combines them into two new levels. A gate blocks
anything negative. The second board combines the surviving two into three
outputs, which get squashed into a bet across three words. The bet is wrong: the
board is loudest on " dog" when the answer was " mat".

Now the correction travels backward, and it does one thing at each stop: it
converts "your output was off by this much" into "your inputs and your knobs
were off by this much".

At the very top, the correction is startlingly simple. Subtract the truth from
the bet. Whatever probability the model spent on wrong answers becomes the exact
amount of push-down on those answers, and whatever it failed to spend on the
right answer becomes the push-up. If the model had been perfect the subtraction
would give zero and nothing would move. If the model were maximally wrong the
push would be maximal. The error signal is literally "the misallocated
probability", and no other loss function gives you something that clean.

At each mixing board, the correction splits two ways, and this is the part worth
slowing down for. A knob on the board is a connection between one input channel
and one output channel. How much did that knob contribute to the error? Exactly
the product of two things: how loud its input channel was, and how much its
output channel was blamed. A knob wired to a silent input cannot have caused the
problem no matter how blamed its output is. A knob whose output was correct
anyway is not at fault no matter how loud its input. Both must be true for the
knob to move. That is the whole content of the weight gradient.

And then, separately, the board has to tell the *previous* board how much its
incoming channels were at fault. Each incoming channel fed several outgoing
ones, so it collects a share of blame from each and adds them up. This is the
step that makes the network deep rather than just wide: blame from three
different outputs merges into a single number for one hidden unit, and that
merged number is what the layer below sees. Nothing else about the layers above
survives. The entire influence of everything downstream arrives compressed into
one number per unit.

The gate deserves its own note because it behaves differently from everything
else. Every other step scales blame. The gate does not scale, it decides.
Channel was positive on the way forward: blame passes through completely
unaltered. Channel was negative: blame is annihilated, and every knob feeding
that channel receives nothing at all. Not a little, nothing. If a channel is
negative on every example you show it, the knobs feeding it never move again, for
the rest of training. The unit is dead, and no learning rate, no optimizer, no
amount of patience revives it, because the only road back to it is multiplied by
zero.

Finally, notice what the update actually did: the loss went from about 1.4 to
about 0.58 in one step, on one example. That is what makes the whole enterprise
plausible. There is no search, no trial and error, no evaluating alternatives.
One forward pass, one backward pass, everything moves at once, and the answer is
better. Scale that up and you have every model you have ever used.

## Optimizers: four update rules, one loop

You are on a hillside in dense fog and you want to get down. You can feel the
slope directly under your feet, and nothing else. No map, no view.

**SGD** is: take a step downhill proportional to how steep it feels. It has an
obvious pathology. Picture a valley shaped like a long narrow gutter, steep
across, gentle along. The steep direction is where the slope is strongest, so
that is where you take your biggest steps, so you fling yourself across the
gutter and back, ricocheting off the walls while barely progressing along the
floor toward the actual bottom. That is not an exotic case, it is what most
loss surfaces look like: some directions vastly steeper than others.

**Momentum** puts you on a heavy sled. You do not respond only to the slope
underfoot, you carry the accumulated push of everywhere you have been. In the
gutter this is transformative. Sideways, the slope reverses every time you cross,
so the pushes cancel and the sled stops ricocheting. Lengthwise, the gentle slope
always points the same way, so the pushes add and you build up speed until you
are moving ten times faster along the floor than the local slope alone would
carry you. Momentum does not care about the size of any single push. It cares
about whether the pushes agree.

**Adam** does something stranger and more useful. It keeps a running record of
how steep each direction has *typically* been, and divides by it. The effect is
that steepness cancels out entirely: a direction that is always steep and a
direction that is always shallow both get the same size of step. You stop walking
down the hill by feel and start walking a fixed distance per step in whichever
direction is consistently downhill. This is why Adam works on transformers,
where the embedding weights, the attention projections, and the MLP weights see
gradients of wildly different scales and no single step size could serve them all.
It is also why Adam feels reckless: a direction receiving nothing but random
noise still gets a full-size step, because the noise is consistent in magnitude
even though it is meaningless in direction.

There is one repair Adam needs at the very start. Its running records begin
empty, so on the first steps they read far too low, and dividing by a too-low
number would produce an enormous step. The fix simply scales the early readings
up by exactly the amount they are known to be too small, and fades away once the
records have filled in. It is bookkeeping, not an idea, but training diverges in
the first hundred steps without it.

**AdamW** changes one thing and it is not in this picture at all. Both Adam and
AdamW shrink every weight slightly toward zero on every step, to keep the model
from letting any weight grow unboundedly. The difference is bookkeeping order.
Adam pushes that shrink through the same steepness-division as everything else,
so a weight in a steep direction ends up barely shrunk and a weight in a flat
direction gets crushed, which is a regularization scheme nobody chose. AdamW
does the shrink separately, after the step, so it means the same thing for every
weight. That one line is the entire difference, and it is why every large model
uses the W.

Hold onto the contrast that matters: SGD's step size is set by how steep it is.
Adam's step size is set by how *consistent* it is. Everything else follows.

## What gradient descent actually finds

The picture in your head is a 2D landscape with hills and valleys, a ball
rolling, and the danger being that the ball settles in the wrong valley while a
deeper one waits elsewhere. That picture is not a simplification of the real
thing. It is a different thing, and it misleads in a specific way.

To be trapped in a valley, you must be surrounded. Every direction you could
step must go up. In two dimensions that is a modest requirement: two directions,
both up, happens all the time. In three it is harder. In a hundred billion
dimensions, being surrounded means every one of a hundred billion directions
goes up, and the odds of that are not small, they are indistinguishable from
zero. What you find instead, over and over, are mountain passes: up in most
directions, down in a few. And a pass is not a trap. There is always an exit. You
might sit near one for a while if the downhill directions are shallow, which is
what a training plateau usually is, but you are not stuck, you are slow.

The second correction is stranger and more important. There is no "the minimum"
to be found. Take a trained network and swap two of its hidden units, moving all
their connections along with them. The network computes exactly the same
function. Nothing has changed except which slot holds which feature. For one
layer of a few thousand units, the number of such relabelings is a number with
thousands of digits, and every single one is a distinct point in weight space
with identical loss. The bottom of this landscape is not a point. It is a
vast, high-dimensional, wildly connected structure that runs through the whole
space.

Which is why two training runs from different random starts do not agree on a
single weight, and why averaging their weights produces garbage. They are not
two estimates of one answer. They are two different, equally valid,
mutually-unintelligible encodings of similar behavior. Run A's unit 7 and run
B's unit 7 have nothing to do with each other, so averaging them averages two
unrelated things.

What actually differs between good and bad solutions is not depth but width.
Some low regions are narrow ravines where nudging any weight sends the loss
climbing. Others are broad flat plains where you can wander and nothing much
happens. The plains generalize better, and the reason is simple: the test data
draws a slightly different landscape than the training data, shifted a bit. A
narrow ravine in one landscape may be a wall in the shifted one. A broad plain is
still a plain. The jitter that stochastic gradient descent gets from using
random batches is what keeps you out of ravines: you cannot come to rest
somewhere that a small random shove would eject you from.

So when training plateaus, "stuck in a local minimum" is almost certainly not
what happened. Look at the learning rate schedule, the data, or the numerics.

## Scaling laws: what more parameters buy

The finding that changed the field is not that bigger is better. It is that
bigger is better *predictably*, on a curve you can extrapolate. Train a series
of small models, plot loss against scale on log-log axes, get a straight line,
and read off the loss of a model that would cost fifty million dollars before
committing to it. That is why the money got spent. Nobody funds a nine-figure
training run on a hunch, but a straight line through six data points is not a
hunch.

The curve has a floor, and it is the same floor from earlier: the openness of
language. Scale closes the gap to the floor by a constant fraction each time you
multiply everything by ten. Ten times bigger halves the remaining gap. Ten times
again halves it again. You approach and never arrive, and each halving costs ten
times more than the last. This is simultaneously the case for scaling and the
case against it, and both readings are honest.

Now the intuition that needs replacing. The instinct is that parameters are
storage, that a bigger model holds more facts the way a bigger disk holds more
files. Test it: if that were true, then adding parameters while holding the data
fixed would let the model absorb more of the training set verbatim and get worse
at anything it had not seen. That is the classic overfitting curve, and every
engineer has watched it happen to a model that was too big for its dataset. But
at scale the measured curve does the opposite. Bigger models get better on data
they have never seen, smoothly, for five orders of magnitude, with no turn.
Storage cannot do that. Storage has nothing to say about a document it never
saw.

The better image is not a disk but a workshop. Parameters are not shelves for
finished goods, they are tools and fixtures and the space to combine them. More
capacity means more distinct features the model can represent, more relationships
between them, more layers of composition, so that things never seen are built
rather than retrieved. And the identical machinery, asked for something it cannot
build correctly, builds something plausible instead and hands it over with the
same confidence. Generalization and hallucination are one behavior seen from two
sides. A model that could not hallucinate could not generalize.

Then there is where to spend a fixed budget, which for several years the field
got wrong. Given a fixed amount of compute, you can train a big model briefly or
a smaller model for much longer. The industry assumed bigger, and built models
that had far more capacity than they had experience to fill it: enormous
apparatus, barely trained. Chinchilla's finding was that model and data should
grow together, roughly twenty tokens of text per parameter as a planning
heuristic. A three-times-smaller model trained three times longer on the same
budget comes out ahead, and it costs a third as much to run afterward, every day,
forever. The second consideration is why real deployed models are trained far
past the compute-optimal point: training happens once, serving happens
continuously.

## Post-training: SFT, RLHF, RLAIF, DPO

A base model is not a broken assistant. It is a working impressionist. It has
read an enormous amount of text and what it does is continue whatever document
it is handed, in the register that document is written in. Hand it a question and
it might answer, or it might add three more questions, because "a list of
questions" is a real kind of document and it has seen many. Hand it the start of
a manual and it writes manual. It is not confused about your intent. It has no
notion of your intent. There is only a document, and it continues.

Post-training is not education. It is casting. All three approaches choose a
mode from the repertoire that already exists.

**Demonstration** is the simplest: here are ten thousand exchanges where someone
asks and someone helpfully answers. Continue documents that look like this one.
The model was already able to write that kind of continuation; you have made it
the default. Notice what demonstration cannot express. You can show a good
answer. You cannot say "and that other answer would have been worse", because
the only sentence this method knows how to say is "produce this".

**Preference** adds the missing sentence. Show the model two of its own answers
and say which one was better. This is different in kind, because now the training
signal is attached to things the model itself produced rather than things a human
wrote, and because "worse" is now expressible. The mechanism has a well-known
failure worth understanding: the judge of better and worse is itself a learned
model, and a system optimizing hard against a learned judge will find the places
where the judge is wrong and live there. Preference training therefore always
comes with a leash tying the model to where it started, and the leash length is
the most consequential dial in the process. Too loose and the model discovers
that the judge loves long confident answers with bullet points, and you get long
confident answers with bullet points about everything.

**Preference from a written specification** is the same machine with the human
raters replaced by a model following an explicit document of principles. The
motivation is partly throughput and partly that a document can be read,
argued with, and revised, whereas the aggregate taste of a thousand contractors
cannot.

**Direct preference optimization** is the observation that you can skip building
the judge. The relationship between "which answer is preferred" and "how the
model should change" turns out to have a closed form, so you can compute the
update straight from the preference pairs. Simpler, cheaper, one training loop
instead of three. What you give up is that the model never sees its own current
output during training. It learns only from the pairs you collected, so it cannot
discover that its newest behavior has a problem the way the sampling-based
approach can.

Now the thing to actually take away. The gap between a base model and a chat
model is astonishing to anyone who has used both, and the natural conclusion is
that the chat model knows vastly more. It does not. Measure them on factual
knowledge with careful prompting and they are within a couple of points. What
changed is which continuation gets chosen, not what continuations are available.

The scale makes this unavoidable. Pretraining is trillions of tokens. Post-
training is tens of millions, a fraction of a percent of a percent. There is no
route by which that installs a world model. It is enough to change a habit, and
that is exactly what it is: habit formation on top of an education that already
happened.

Which is also the honest answer to why post-training cannot fix hallucination.
If the capability was never built, preference data cannot conjure it, and the
same optimization that rewards helpful answers rewards confident-sounding ones.
You can teach a model to say "I do not know" more often. You cannot teach it,
this way, to know.
