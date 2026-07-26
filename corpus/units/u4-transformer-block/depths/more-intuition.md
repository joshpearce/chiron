---
unit: u4
depth: more-intuition
---

## The block in one equation

Picture a wide flat conveyor belt running the length of the model. The belt is
$d$ lanes wide - 4096 lanes in an 8B model - and there is one belt per token
position, running side by side. A block is a station the belts pass through.
The station does not lift anything off the belt. It looks at what is there,
decides on a small pile of its own, and drops that pile on top. What was
already on the belt stays.

There are two stations per block, and they differ in one way only: the
attention station is allowed to reach sideways to the other belts; the MLP
station can only look at its own. Everything else in the architecture is
plumbing around that difference.

The reason every block has the same input and output shape is the reason the
belt exists at all: a station that changed the belt's width would end the belt.
Constant width is what makes "stack 32 of these" a sentence rather than an
engineering project.

## Multi-head attention: many subspaces, one parameter budget

The wrong picture is eight photocopies of an attention machine. The right
picture is one machine looking through eight narrow slits cut in a screen.

Each slit shows a different 64-lane slice of what is on the belt. Looking
through slit 3, the machine sees only what has been written into those lanes,
computes similarities using only that, and gets a set of weights: how much of
each other position to pull in. Slit 5 sees a different slice, so it gets a
completely different set of weights. Eight slits, eight independent answers to
"where should I look", all computed in one pass and all delivered at once.

Now the point that makes single-head attention obviously insufficient. A
softmax over positions is a single opinion. If you have one slit, then when
position 12 wants both the subject from position 3 and the tense from position
9, it must produce one blend - say 50/50 - and everything it copies is that
blend. Half of a subject and half of a verb tense, superimposed, in every lane.
With eight slits it takes the subject at full strength into lanes 0-63 and the
tense at full strength into lanes 128-191, and neither contaminates the other.

The cost of narrowing the slits is that each one sees less. A slit 64 lanes
wide can only distinguish patterns that are visible in 64 lanes. Widen a slit
and it sees richer structure but you have fewer of them, so fewer simultaneous
opinions. Language apparently wants many crude opinions rather than few refined
ones, down to about 64 lanes per slit, below which the slits get too narrow to
see anything worth seeing.

Nothing about this changes the parameter count. Cutting eight slits in a screen
does not require eight screens.

## MQA and GQA: the same equation with fewer key-value heads

The picture for grouped-query attention is a library reading room.

Under standard multi-head attention, every reader (query head) has a private
card catalogue and a private copy of every book. Thirty-two readers, thirty-two
catalogues, thirty-two full copies. When a new question arrives, each reader
walks their own aisle and pulls their own copies.

Grouped-query attention says: keep thirty-two readers, but let groups of four
share one catalogue and one copy of the books. Eight catalogues instead of
thirty-two. Each reader still walks the aisle, still reads every entry, still
does exactly as much reading as before - so the *work* is unchanged. What
changed is how much shelf space the room needs and how much material has to be
carried in from the warehouse each time.

That distinction is the entire thing. It is easy to assume that fewer
catalogues means less searching. It does not; four readers sharing a catalogue
each search the whole thing. The saving is storage and haulage.

Why haulage dominates: generating one token is a trip to the warehouse to fetch
every cached key and value seen so far, and then a comparatively tiny amount of
arithmetic on them. The truck is the bottleneck, not the reader. Quarter the
volume on the truck and you go roughly four times faster, even though nobody
read any faster.

Why not go all the way to one shared catalogue (multi-query)? Because a
catalogue is organized around some notion of what makes two things similar, and
thirty-two readers looking for thirty-two different kinds of similarity cannot
all be served well by one organization. Eight is enough variety that each
reader finds a catalogue close to what it wanted. One is not.

## The MLP block: where the parameters actually live

Most people picture the transformer as attention with some connective tissue.
By weight it is the opposite: it is a stack of feed-forward layers with
attention slipped between them so that positions can talk. Two thirds of the
weights in a GPT-2 block, four fifths in a Llama block, are in the
feed-forward part.

The geometric picture is a detour through a bigger room. Your token's state
lives in a 4096-dimensional space. Directly transforming it within that space
is limited: without a nonlinearity, any composition of such transforms is one
transform. So you step up into a 14,336-dimensional room, which gives the
state enough elbow room for distinct things that were tangled together in the
smaller space to sit apart. In the big room you apply a bend - the
nonlinearity - which is the only step in the whole model that is not a rotation
and a stretch. Then you come back down.

The bend is the point of the detour. Up-projecting and immediately
down-projecting with no bend would collapse into a single 4096-by-4096 matrix
and buy nothing. The extra room exists so that the bend has something to work
with: in a big enough space, features that overlap can be pulled far enough
apart that a simple threshold separates them.

Gating (the SwiGLU variant) adds one refinement to this picture. Instead of one
detour, you take two: one computes a candidate, the other computes, per
dimension, how much of that candidate to let through. Route and payload,
computed from the same input. It buys a little quality per parameter, which is
why current models pay for the third matrix.

## Key-value memory, and where that framing breaks

Here is the honest version of the lookup picture, and then the exact place it
misleads.

The up-projection's columns are a wall of 14,336 detectors. Each one is a
direction in the token's state space, and the strength with which it fires is
how well the current state points that way. Each detector is wired to its own
little vector, and when it fires it adds its vector to the belt, in proportion.
That is genuinely a memory: patterns in, associated content out. Some detectors
really do fire on things you could name.

Now the misleading part, in two pictures.

**First: there are no rows, only directions.** A database has a row for Paris
and a row for Warsaw and they are separate objects. Here, every fact is a
direction in a continuous space, and the space does not have enough
perpendicular directions to give each fact its own. So directions are packed at
slight angles to one another, thousands more of them than there are dimensions,
accepting that each one bleeds a little into its neighbours. Think of a
crowded noticeboard where every pin overlaps two others slightly. You cannot
remove one pin cleanly, and no single pin is one note.

**Second: every query gets an answer.** A lookup either finds a key or does
not. A wall of detectors always produces something: for a state that matches
nothing well, dozens of detectors fire weakly and their vectors add up into a
plausible-looking blend of the nearest stored patterns. That is not a bug bolted
on to a retrieval system. It is the same operation that lets the model answer a
question phrased in a way it never saw - and if you removed it, you would remove
that too. Fluent confident wrongness and generalization are two views of one
mechanism: the space between the stored patterns is filled in, and the filling
is smooth, and the model has no way to know whether it is standing on a stored
pattern or in the space between two.

## The residual stream is a workspace, not a shortcut

The story you were told about residual connections is that they are a rope
thrown down a deep well so the gradient can climb back out. That is true. It is
also the least interesting thing about them.

Go back to the conveyor belt. Every station drops a pile on it and nothing is
ever removed. So what arrives at the end is: the original token's embedding,
plus sixty-four piles. Not a transformation of a transformation of a
transformation - a sum.

This immediately explains the thing a pipeline picture cannot. If the model
were a pipeline, removing station 16 would be removing the fifth stage of an
assembly line: station 17 receives a half-built object at the wrong stage and
everything after it is nonsense. What actually happens when you delete a middle
block is that the belt arrives at station 17 carrying sixty-two piles instead
of sixty-four, which is very nearly the same belt, so station 17 does very
nearly what it would have done. Output quality dips a few percent. Delete two
or three scattered middle blocks and the same thing happens again.

Two places where this stops being true, both instructive:

The **first** blocks are not droppable, because the later stations were trained
to look for things the early stations put on the belt. Skip station 1 and every
later station is looking at empty lanes. The **last** blocks are not droppable
either, because their job is to turn the accumulated pile into the specific
orientation the output layer reads. It is the middle - the long stretch of
"add another refinement to a belt that already has plenty on it" - that is
redundant.

And the thing that genuinely breaks everything: replace the `+` with `=` at any
single station. Now that station throws the whole belt away and puts down only
its own pile. Every station before it has been erased. This is the difference
between a shortcut and a workspace, and it is the difference the gradient-rope
story cannot express.

One more consequence, worth holding on to for later units. The belt is 4096
lanes wide and sixty-four stations write to it. They cannot each have a private
allocation. They share lanes, deliberately, with small amounts of interference -
the same crowded-noticeboard picture as the MLP's detectors, now at the level of
the whole network's wiring.

## Normalization conditions the optimization; it does not save your floats

You know normalization from numerics, where it means "keep the numbers in
range." Set that aside completely; it is the wrong department.

Here is a better picture. Each station on the belt was trained expecting the
belt to arrive with roughly a certain amount of material on it. But the belt
accumulates: station 30 sees far more material than station 2 did. Without
intervention, each station would need its own calibration for the depth it sits
at, and the whole stack of identical blocks stops being identical.

Normalization is the step where, before a station looks at the belt, it takes a
photograph of the belt's *pattern* with the overall brightness thrown away. Only
the relative arrangement survives. Every station sees a picture at the same
exposure, no matter how much has piled up. That is why you can train a hundred
identical blocks.

The decisive fact about this operation is that scale is discarded exactly.
Multiply everything on the belt by a thousand and the photograph is unchanged.
An operation whose defining property is "magnitude is irrelevant to me" is not
in the business of managing magnitude for safety.

And that is exactly why removing it breaks a trained model at any precision. The
downstream weights were fitted against normalized pictures. Hand them the raw
belt and they are receiving something in units they have never seen - not too
large for the number format, simply not what they were calibrated against. Doing
it in double precision changes nothing at all, because nothing overflowed. The
error is a units error, not a range error.

The second thing normalization does is subtler and lives in training. Because
scale is thrown away, the training process cannot get any credit for making a
representation merely *bigger*. The only way to change the output is to change
the *direction*. This forces every unit of learning into the part of the
representation that carries information, and it is the real meaning of the
phrase "conditions the optimization."

RMSNorm is the same photograph with one adjustment skipped: it rescales for
overall brightness but does not first subtract the average. It works as well,
which is a strong hint that the averaging was never carrying the weight.

## Pre-LN and post-LN

One operation, two places to put it, and remembering which is which is easier
with a picture than with the names.

**Pre-LN**: the station takes a photograph of the belt, works from the
photograph, and drops its pile on the belt. The belt itself is never touched by
the normalizer. From the embedding to the very end, the belt is pure
accumulation - a clean, unbroken line.

**Post-LN**: the station works from the belt directly, drops its pile, and then
*the belt itself is renormalized* before moving on. The belt is scrubbed
sixty-four times on its way through the model.

Once you can see which one touches the belt, everything else follows. Post-LN
scrubs the belt, so the unbroken line from end to beginning is gone, which is
what the training process was using to reach the early stations - hence
post-LN's need to start training with tiny steps and ramp up (warmup), and its
tendency to become untrainable when the stack gets deep. Pre-LN keeps the clean
line and trains without ceremony, at any depth.

Pre-LN pays for it in the mirror-image way. Since nothing ever scrubs the belt,
the pile keeps growing, and a late station's contribution is a smaller fraction
of a taller pile. Late layers matter proportionally less. Post-LN, when it
trains at all, keeps every station's contribution comparable and often ends
slightly better.

The trick for recalling which is which under pressure: a pre-LN model needs one
extra normalizer bolted on at the very end, right before the output layer,
because otherwise nothing ever normalizes the belt and the output layer would
receive the raw accumulated pile. A post-LN model needs no such thing, since its
last action already was a scrub. Dangling final normalizer means pre-LN.

## Depth: composition through the stream

The tempting picture of depth is polishing: each layer takes the previous
layer's answer and improves it. Under that picture, a very wide shallow model
should do anything a deep one does, a bit worse.

Here is the observation that kills it. Show a model a made-up pair - a name it
has never seen followed by an invented suffix - somewhere in the prompt, then
show the name again. It completes the suffix. It has never seen either token
next to the other, so this cannot be memory; it is reading a pattern out of the
prompt. A model with one attention layer cannot do this at any width. A model
with two can. Something appears at exactly two that no amount of width provides.

The mechanism is two stations cooperating through the belt, and it is worth
seeing concretely.

The first station is dull: at every position it reaches back exactly one step
and writes onto the belt a note saying "the token before me was such-and-such."
Every position now carries a note about its own predecessor.

The second station does the interesting part. Sitting at the current token, it
asks: which position on the belt has a note saying its predecessor was *this
token*? Because the first station wrote those notes, that question is now
answerable. The position it finds is precisely the one right after the earlier
occurrence, and the second station copies what is there.

The essential thing is where the second station's *question* comes from. It is
looking for something the first station wrote. In a one-layer model there is no
first station, so every question can only be about the raw tokens themselves -
never about a relationship between them. Width lets you ask more questions
about raw tokens. It never lets you ask a question about an answer.

That is what depth buys, and it is a completely different currency from width.
Width is how many things you can hold at once at a given stage. Depth is how
many times a thing can be a function of another thing. Some computations
require a certain number of stages and there is no way to purchase them
sideways - the same reason a circuit with a fixed number of gate delays cannot
be made shallower by adding more gates in parallel.

Where things sit, roughly, in a stack of 32: the first handful assemble tokens
into words and pick up local grammar. The long middle holds the abstract
material - who is being talked about, what kind of task this is, what facts are
in play - and is the region that survives having pieces removed. The last few
turn all of that into a bet about the next token, which is why removing them is
fatal.

## What to carry forward

One belt, constant width, running the length of the model. Stations read a
photograph of it and drop piles on it; nothing is removed. Attention is the only
station allowed to reach sideways, and it looks through several narrow slits at
once rather than one wide one. The feed-forward station is where most of the
weight is, and it works by a detour through a bigger room where a bend can
separate things. The photograph step exists so every station sees the same
exposure regardless of how tall the pile is. And the reason to have many
stations rather than one enormous one is that a question can only be about an
answer if somebody answered it earlier.
