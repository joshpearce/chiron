## The dialect first

Everything below is the same four tools you have been using informally:
similarity scores, chained transformations, "which way is downhill," and
averages over data you can only sample. The sections ahead give each one a
picture; slow down wherever the calibration series slowed you down.

## The shape contract

Picture a vector as an arrow from the origin. A two-dimensional vector is an
arrow on a sheet of paper; a three-dimensional one is an arrow in the room you
are sitting in. A 4096-dimensional one is the same idea with more directions
available, and the honest thing to say is that nobody visualizes it - you
visualize three dimensions and trust the algebra to behave analogously, which it
mostly does.

A matrix, in this picture, is a machine that takes arrows and hands back arrows,
possibly in a different space. Feed it an arrow in the room, get back an arrow
on the paper. The shape of the matrix is a label on the machine listing what it
accepts and what it emits, exactly like the type signature on a function.

The shape rule then reads as: you can only feed a machine's output into another
machine if the second machine accepts what the first one emits. A machine that
emits arrows-on-paper cannot feed one that expects arrows-in-a-room. That is all
"inner dimensions must match" says. And the composite machine takes what the
first one takes and emits what the last one emits, which is why the outer
dimensions survive.

The row-vs-column convention is a pure question of which side you write the
machine on. Some traditions write the machine on the left of what it eats, some
on the right. The machine is the same machine; write it on the wrong side and
the label reads backwards. It is the difference between `f(x)` and `x |> f` -
same computation, and you have to know which pipeline direction the document
uses before you can read any of its shapes.

The picture to hold for the whole book: a token starts life as an arrow, and
every layer of the model is a machine that nudges that arrow. Generation is
following where the arrow ends up.

## Dot products: alignment, not distance

Sunlight and a shadow. Hold a stick (call it $v$) and shine light straight down
onto a line drawn on the ground pointing along $u$. The stick's shadow on that
line is the projection of $v$ onto $u$'s direction. The dot product is the
length of that shadow, times the length of $u$.

Everything about dot products falls out of the shadow picture:

- **Perpendicular gives zero.** A stick held at right angles to the line casts
  no shadow along it. Zero dot product is not "unrelated" in any deep sense -
  it means the two directions carry no component of each other.
- **Opposite gives negative.** Point the stick backwards along the line and the
  shadow lands on the negative side.
- **Longer stick, longer shadow.** Double the stick's length without rotating
  it and the shadow doubles. The angle did not change. This is the entire
  content of the refutation in canon: shadow length depends on both the angle
  and the size of the object, so a big object at a bad angle can out-shadow a
  small one at a perfect angle. Cosine similarity throws away the size and keeps
  only the angle. The dot product keeps both, deliberately.

The size-matters property is not a defect. Inside a transformer, magnitude is
how a component says "this is important" and direction is how it says "this is
what it is about". The dot product listens to both at once, in one number. That
is what makes it the right primitive for attention, and also what makes
attention distributions sensitive to a single unusually large vector - one very
long stick can dominate the shadows of everything else on the line.

## Matrix multiply: composition, not a loop

The right picture is the darkroom, not the spreadsheet.

A matrix is a transformation of space itself. Not a table of numbers that gets
looped over - a warping. Draw a grid on a rubber sheet, then stretch, rotate,
shear, or flatten the sheet while keeping the origin pinned and all grid lines
straight and evenly spaced. That constraint - straight stays straight, evenly
spaced stays evenly spaced, origin stays put - is exactly what "linear" means,
and every matrix is one such warp.

Multiplying two matrices means doing one warp, then the other, and asking what
single warp does the same job. Rotate 90 degrees, then stretch horizontally.
The combined effect is one warp, and the product matrix is its recipe. Once you
see it that way, three properties stop needing memorization:

- **Order matters.** Rotate-then-stretch is visibly not stretch-then-rotate.
  Rotate a square 45 degrees then stretch horizontally and you get a slanted
  diamond; stretch first then rotate and you get a tilted rectangle. Different
  final shapes, same two operations. $AB \neq BA$.
- **Grouping does not matter.** Doing three warps in a fixed order gives the
  same result no matter which pair you mentally combine first. That is
  associativity, and it is why the same computation can be reassociated for
  speed without changing the answer.
- **Flattening is permanent.** A warp that squashes the sheet onto a line has
  destroyed a dimension, and no later warp recovers it. That is what a
  non-invertible matrix is, and it is the geometric reason a model's projection
  down to a narrower space is a lossy commitment.

The "grid of dot products" formula is the recipe for computing the warp, not the
meaning of it. When you read $XW$ in a later unit, the useful thought is not
"loop over the rows"; it is "every token's arrow gets warped into the space $W$
maps to". A transformer is a long chain of such warps with a nonlinear kink
between each pair, and the kink is there precisely because chained warps with no
kink collapse into a single warp.

## Gradients and expectations

Fog on a hillside.

You are standing on a landscape in thick fog and want to get downhill. You
cannot see the valley - you cannot see ten feet. What you can do is feel the
ground under your feet: which way does it tilt, and how steeply? That local
tilt is the gradient. Step against it, then feel again.

The picture repairs three misreadings at once.

**Why the gradient is not one number.** A hillside does not have "a slope" - it
has a slope in each direction you might face. Facing north it may drop sharply,
facing east it may be flat. The full answer is a reading for every direction you
can face, and with a weight matrix of a million entries there are a million
directions to face. The gradient is that whole set of readings, which is why it
is the same size and arrangement as the thing it describes: one reading per
weight, laid out in the same grid the weights live in.

**Why the shapes must match.** The step you take is "move each weight a little
against its own reading". Weights and readings pair up one-to-one, so they must
be arranged the same way. Trying to subtract a single number from a matrix of
weights would be like responding to a topographic survey by taking one step in
one direction - it throws away every reading but one.

**Why the fog never lifts.** You only ever learn the ground where you stand. You
never learn whether there is a deeper valley two ridges over. Training is
millions of local feel-and-step moves with no global view at any point, which is
why u2's claim that gradient descent does not find "the" minimum should already
feel obvious rather than surprising.

Now the expectation, which is where the fog comes from. The true landscape is
defined over all the text that could ever exist, and you cannot survey it. Each
training batch is a handful of soil samples from where you stand. Average them
and you get an estimate of the local tilt - roughly right, individually noisy.
More samples per step, steadier reading, slower going. Fewer samples, jumpier
path. That trade is the whole of batch-size tuning, and the surprise waiting in
u2 is that the jumpiness turns out to help: a walker who staggers a bit does not
get stuck in every small dip on the way down.

## The objective, stated exactly

Picture a very long strip of text and a window sliding along it, one position at
a time. At each stop, the window covers everything to the left, and the game is:
guess what comes next. Not "guess the topic" or "guess the answer" - guess the
literal next fragment. Then the strip advances by one and you play again.

You play this game once per position in every document you ever see. A single
page of text is a thousand rounds, not one. That is the first thing people get
wrong about how much training signal is in a corpus: every token is a labeled
example, and the label is free, because it is the next token. Nobody had to
annotate anything. That is the entire reason this approach could be scaled to the
whole internet while approaches needing human labels could not.

The score works like a golf handicap on surprise. If you would have guessed the
right fragment with near-certainty, you score near zero. If you thought it was
unlikely, you take a large penalty. If you had ruled it out completely, the
penalty is infinite - so an experienced player never rules anything out entirely,
and keeps a sliver of probability on every possibility. That habit, forced by the
scoring rule, is why a language model always has *some* answer for everything,
including things it has no business having an opinion about.

Notice what the score does not measure. It never asks whether the text was true,
useful, kind, or well-reasoned. It asks only whether you were surprised. A model
trained this way is a model of *what people write*, and everything else it seems
to be is a consequence of the fact that people mostly write about a real world.

## Why prediction forces world modeling

Here is the picture that makes the compression argument click.

Imagine you are trying to send a book to a friend over an expensive channel, and
you both have identical copies of the same very good guesser. You do not send the
book. You send only the *corrections*: at each position, where your guesser's
prediction was wrong, you send just enough information to fix it. Your friend
runs the same guesser, applies the same corrections, and reconstructs the book
exactly.

Now: how much do you send? Exactly as much as your guesser is surprised. A
guesser that always knew what came next would let you send nothing. A guesser
that knew nothing would make you send the whole book. The bill is the surprise,
and the surprise is the loss. Improving the model and shrinking the transmission
are not analogous activities; they are the same activity, measured in different
units.

So ask what it takes to shrink the bill on a passage like this:

> The chef added the salt, tasted it, and grimaced. She reached for the

To send few bits here you have to have understood that salt was added, that
tasting produced a bad reaction, that the likely problem is over-salting, and
that a cook's response to over-salting involves specific ingredients. There is no
way to be cheap on this sentence while being ignorant of kitchens. The bits are
the receipt.

Multiply that across every sentence humans have written about physics, law,
grief, distributed systems, and chess, and the bet becomes visible: the cheapest
possible encoding of everything people have written is a model of everything
people were writing about. Squeezing the file is the same act as building the
model.

The honest caveat, in the same picture: nobody has squeezed the file all the way
down. We are partway, and partway down you get a mix. Some of the compression
comes from real understanding, and some comes from cheap tricks that happen to
work most of the time - the way a good chess player who has never calculated a
line still plays a decent move because it *looks* like the moves that usually
work. When you catch a model confabulating a plausible-shaped answer, you are
watching a cheap trick that has not yet been outcompeted by an expensive
understanding.

## Tokens: BPE from scratch

Think of the tokenizer as a stonemason who, before the building starts, decides
what sizes of block will exist. He walks the quarry, sees which shapes keep
recurring, and cuts a mold for each of the common ones. Very common shapes get
their own mold. Uncommon shapes have to be assembled from smaller molds. Nothing
can be built from anything but molds.

The masonry gets built beautifully, and the joins are invisible in the finished
wall. But the joins are still there, and they determine what the building can and
cannot do. A wall of blocks cannot be sanded down to a smooth curve at a scale
smaller than one block. That is exactly the situation the model is in with
characters.

The clearest way to feel it: imagine reading a book in a language you know
fluently, but where the printer has glued random pairs and triples of letters
into single unbreakable ligatures - and where the gluing depended on how common
the sequence was, so common words are one glyph and rare words are three or four.
You would read it perfectly well. Meaning would flow. But if someone asked you to
count the letter `r` in a word, you would have to reason indirectly - "I know
that glyph is usually spelled that way, so probably..." - rather than looking.
That is precisely the model's relationship to spelling, and precisely why it
fails in the way it fails: not confidently blank, but confidently approximate.

Numbers are worse, because the gluing ignores place value. Picture writing out a
long addition, but the digits arrive pre-bundled in groups of three counted from
the *left*, so the bundles of the two numbers do not line up under each other.
Before you can add anything you have to unbundle and re-align, and the re-
alignment is different for every pair of numbers depending on their lengths. You
could learn to do it. You would make mistakes, and your mistakes would cluster on
unusual lengths. That is exactly the observed error pattern.

One more picture, for why frequent things get single blocks. The tokenizer is
running a compression argument of its own, on a smaller scale: the total size of
the corpus, measured in blocks, shrinks fastest when you make a mold for the
shape you see most. It has no notion of what a word is, what a suffix is, or what
matters. It is counting. Every apparent piece of linguistic insight in a
vocabulary - and there is a lot of it - fell out of counting alone. That should
be your first taste of the pattern that runs through this whole book: structure
that looks designed, produced by an objective that knows nothing about the
structure.

## Embeddings are coordinates, not contents

Here is the picture to replace "the vector contains the meaning."

Think of a city with no street names, where every building is identified only by
its position, and where the positions were chosen by a developer whose only goal
was that people who need to visit the same places should live near each other.
Over years of construction, bakeries drift toward bakeries, hardware stores toward
hardware stores, and a whole quarter emerges where everything is about food -
not because anyone labeled it, but because pushing them together made everyone's
commute shorter.

Now ask: what does the *coordinate* of a bakery tell you? Nothing, on its own.
Latitude 41.2 is not a fact about bread. What tells you something is the
neighborhood - who is near, what direction the market lies in, how far to the
docks. And the moment you carry that coordinate to a different city, it is
meaningless. Not degraded. Meaningless. There is a building at that spot in the
other city and it is a dentist.

That is what an embedding is, and it is why the standard question - "what does
this dimension mean?" - has no answer. It is like asking what the number 41.2
means. It means whatever the rest of the map makes it mean, and the map is
private to this model.

Two more pictures that sharpen it.

**The relabeling picture.** Suppose overnight someone renumbered every axis of
the city grid - what was north is now the third coordinate, and so on - and
simultaneously updated every map, every GPS unit, and every taxi driver's
internal sense of direction, all consistently. Every journey would be identical.
Nothing observable changed. That is the situation with a model's dimensions:
there are astronomically many equally valid renumberings, so no individual
dimension can be carrying a fact. Only the relationships survive relabeling, so
only the relationships are real.

**The single-address picture.** The word `bank` has one address. The riverbank
and the financial institution live at the same coordinates. If the address
contained the meaning, this would be a contradiction. It is not a contradiction,
because the address contains no meaning - it is a starting point, and what
happens next is that the surrounding context *moves* the representation. That
movement is attention, and it is what u3 is about. Embeddings are where the
journey starts, not where the information is.

## Counting the embedding matrix

Two numbers, one ratio, one point.

Take a 7-billion-parameter model. About 130 million of those parameters are the
table that assigns each vocabulary entry a starting position. Call it 2% on the
way in, 4% if you count a separate table on the way out. Everything else - the
other 96% - is machinery that transforms.

Now hold the two facts side by side. The part of the model that *is* a table
contains no facts about the world. It contains starting positions, and a starting
position for the token `Paris` says nothing whatsoever about France. The part of
the model that contains what looks like knowledge is not a table at all, has no
rows, and cannot be indexed.

This matters because "the weights are a compressed database" is the single most
natural mental model for an engineer to reach for, and it produces wrong
predictions everywhere downstream. If you believe in rows, then a wrong answer is
a bad row, and the fix is a better row. But there is nothing to fix. A wrong
answer is the output of a function that produced a wrong output, in the same way
and by the same mechanism that it produces right outputs on questions it was
never asked before. Those are the same capability seen from two sides. u5 makes
this concrete; the picture to carry there is 4% table, 96% machine, and the table
is the part with nothing in it.

## The output end: hidden state to logits

Picture a searchlight and a room full of mirrors.

The stack has finished, and it hands you one arrow - a direction in a very
high-dimensional space. Around you, arranged in that same space, is one fixed
arrow per vocabulary entry: 128,000 of them, pointing in every which way. These
were learned during training and they do not move.

The question "what token comes next?" is answered by shining the stack's arrow
around and asking, for each of the 128,000 fixed arrows, how well it lines up.
Aligned arrows light up brightly. Perpendicular arrows are dark. Opposed arrows
go negative. The brightness of each is that token's logit.

The whole output layer is that one sweep. There is no search, no comparison
between candidates, no tie-breaking logic - just 128,000 independent alignment
measurements, computed simultaneously. Whatever the model has "decided," it
decided upstream, and the decision is encoded entirely in which direction the
final arrow points.

Two consequences fall out of the picture directly.

**The brightness scale has no zero.** Turn up the room lights and everything gets
brighter by the same amount, with no change in which mirrors are brightest or by
how much they lead. Only the *contrast* between mirrors carries information. This
is why a raw logit value tells you nothing and a logit gap tells you everything -
and it is why you should be suspicious of any tooling that reports absolute
logits as if they were scores.

**The same room, a different arrow, a different winner.** Nothing about the
mirrors changed between one token and the next. What changed is where the stack
pointed. All the work is upstream; the output layer is a readout.

And this is the picture that makes the next unit easier: attention does the same
thing. A query arrow, a room of key arrows, alignment measured by how well they
line up. The model has essentially one move - compare a direction to a set of
stored directions - and it uses it at the output and it uses it in every attention
head. Once you see the move here, in its simplest setting, you have already seen
the hard part of u3.

## What the bet actually claims

Here is the whole system in one sentence you can picture: text is chopped into
blocks by a frequency table, each block becomes a point in a private space, a
stack of machinery pushes that point around, the final point is compared against
every block's direction, the comparisons become odds, one block is drawn, and the
loop runs again.

The remarkable thing is what is *absent* from that description. There is no
knowledge base. No inference engine. No goals. No sentences, no grammar, no
concepts as separate objects. Nothing in the design says anything about
understanding. The design says: reduce surprise on text.

The bet was that this is enough - that if you push hard enough on "reduce
surprise," the machinery has no option but to grow the internal structure that
makes text unsurprising, and that structure is a model of the world the text
describes. It was not obvious. Reasonable people expected it to plateau into a
very good autocomplete with no depth behind it.

It did not plateau. But hold the result at the right resolution: what emerged is
neither the shallow autocomplete the skeptics expected nor the reasoning engine
the demos suggest. It is a system with real internal structure, unevenly
distributed, built entirely out of pressure to be less surprised. Every strength
and every failure you have seen in your tooling traces back to that, and the rest
of this book is the trace.
