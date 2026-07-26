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
