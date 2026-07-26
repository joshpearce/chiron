---
unit: u3
depth: more-intuition
---

## The problem: mix information across a sequence with no recurrence

Picture the sentence as a row of sealed boxes. Each box holds one token's
vector, and at the start no box knows what is in any other. The word "it" sits
in box 7 holding a vector that means roughly "some third-person singular
thing" and nothing more specific, because nothing more specific was available
when the box was packed.

The job is to open channels between boxes. Not fixed channels wired at
manufacture time - the sentence is a different length every time and the
useful connections are different every time. The channel from "it" to
"animal" has to be discovered from what is written in the boxes.

Two obvious designs fail for reasons worth feeling rather than only reading.

A bucket brigade passes each box's contents to its neighbor, accumulating as
it goes. Box 7 receives a summary of boxes 1 through 6. This works, and it is
what a recurrent network does. It fails on throughput: box 7 cannot start
until box 6 finishes, so a thousand boxes take a thousand turns no matter how
many hands you have. You have a warehouse of workers standing idle behind a
single moving parcel.

A switchboard with a dedicated wire for every pair works and is fully
parallel, but the wiring depends on the number of boxes. Build it for a
hundred boxes and it cannot handle a hundred and one.

What survives is a design where every box shouts a short description of what
it is looking for, every box shouts a short description of what it has, and
then every box mixes in a bit of everyone's contents in proportion to how well
the descriptions matched. No wiring, no waiting, no fixed count.

## Queries, keys, and values

The most useful mental image is a room of people, each holding a card with
three things written on it.

On one line, what they are looking for. On another, what they are advertising
about themselves. On the third, what they will actually hand you if you take
them up on it.

Everyone reads their "looking for" line against everyone else's "advertising"
line, scores the match, and then takes a portion of everyone's "what I hand
you" - a big portion from good matches, a sliver from bad ones. Nobody chooses.
Everybody takes a bit of everybody, weighted.

Three things this picture gets right that the spotlight metaphor gets wrong.

The scoring uses the advertisement, never the goods. If you swapped what every
person was carrying while leaving their advertisements untouched, the
proportions everybody took would not change at all. Nobody looked in the bag.

Everyone plays both roles at once. There is no separate group of people being
searched. Each person is simultaneously searching and being searched, and
they are the same people, so the room grows and shrinks as one.

Advertisement and goods are deliberately different lines on the card. A person
can advertise "I am a place name, ask me about geography" and hand over
something about rivers and latitude. Being findable for one reason and useful
for another is the whole point of writing them on separate lines, and it is
the part of the design most explanations quietly collapse into one.

The scoring itself is a projection question: how much does one direction point
along another. Two vectors pointing the same way score high, perpendicular
ones score zero, opposed ones score negative. Length matters too - a person
who writes their advertisement in very large letters gets read by more people,
whatever it says. That is not a bug; it is how a position controls how much it
gets read at all, separately from what it gets read for.

## Scaled dot-product attention: the equation

The whole equation is a grid, a row-by-row election, and an averaging.

The grid is the scoreboard: one cell for every pair of positions. The cell in
row 3, column 5 holds how well position 3's search matched position 5's
advertisement. The grid is square and its side is the sentence length, which
is the single most important structural fact about the whole mechanism.

The grid is not symmetric. Row 3, column 5 and row 5, column 3 ask different
questions - one is "how much does 3 want 5", the other is "how much does 5
want 3" - and there is no reason those agree. A pronoun badly wants its
antecedent; the antecedent is indifferent to the pronoun.

The election happens along each row separately. Row 3 holds all the offers
available to position 3, and softmax turns them into shares of a single unit
of attention that position 3 must spend entirely. Every row spends exactly
one unit. Nothing forces a column to receive any particular amount: a
spectacularly popular token might be read heavily by everyone, and a dull one
by nobody, and both are fine. If you ever find yourself expecting columns to
balance, you have imagined a conservation law that does not exist.

Then the averaging. Position 3 takes its shares and mixes the goods
accordingly. Picture the value vectors as pins stuck in a board. The output is
a point somewhere inside the shape those pins outline - never outside it. With
three values it is a point inside a triangle, and the weights say how far
toward each corner. Weight almost entirely on one pin puts the output almost
on top of that pin; even weights put it at the center.

That containment is a genuine limitation, not a technicality. A single
attention layer can only ever produce blends of what is already present in the
sequence. It cannot invent a direction nobody supplied. Producing genuinely
new content is the job of the other half of the transformer block, and it is
why attention alone is not a network.

And the grid disappears. It is scaffolding: built, used to compute the mix,
discarded. What continues to the next layer is one vector per token, the same
shape that arrived. The pretty heatmap everyone prints is an intermediate
nobody downstream ever sees.

## Why divide by the square root of $d_k$

Roll one die and the outcomes spread over a small range. Roll a hundred dice
and sum them, and the total spreads over a much larger one - not a hundred
times larger, but ten times, because randomness accumulates by square root
rather than proportionally.

A score is a sum of that kind: one term per dimension, each contributing a
small independent nudge up or down. More dimensions means more nudges means a
wider spread of totals. At two dimensions the scores in a row sit close
together. At five hundred they are flung far apart.

Now the part that matters, and it is about softmax's character rather than
about size. Softmax is a contest that gets more brutal as the contestants
separate. When scores are close it is a genuine election with a real
distribution of shares. When one score pulls far ahead the contest becomes a
coronation - the leader takes essentially everything, the rest get essentially
nothing.

A coronation is bad for a reason that is easy to miss. It is not that the
answer is wrong. It is that the answer stops responding. Nudge the leading
score up or down a little and the shares do not move, because the winner was
going to win either way. That deadness is fatal for learning: training works
by nudging things and watching what changes, and a mechanism that does not
respond to nudges cannot be trained. Worse, the wide spread is there from the
first step, before any learning has happened - so the layer is frozen at
birth, not frozen by converging.

Dividing every score by the same number pulls the contestants back together.
It changes nothing about who is ahead - dividing everyone by the same amount
preserves the whole ranking - only how far apart they stand, and therefore how
sharply the contest resolves. The right amount to divide by is exactly how far
apart the extra dimensions pushed them, which is a square root because that is
how randomness accumulates.

The dice picture also tells you why it is not a floating-point concern.
Nothing here is too big to write down. The numbers are perfectly ordinary; it
is the *gaps* between them that are too wide, and gaps do not care what
precision you store them in.

## The whole computation by hand

Two habits make hand computation reliable, and both are about catching errors
where they happen instead of at the end.

Watch the shape at every step. The projections turn a wide token vector into a
short one, three times, so three short vectors per token. The scoreboard is
square with a side equal to the token count - not the vector width, which is a
tempting confusion. The output is back to one short vector per token. If a
shape surprises you, stop; you have transposed something, and every number
downstream is already wrong.

Watch the containment. Every output number has to sit between the smallest and
largest value in the corresponding slot across all the value vectors. It is a
weighted average, and a weighted average is trapped between the extremes of
what it averages. An output larger than everything that went into it is
arithmetically impossible, so if you see one you have a bug and you have it
now, not somewhere upstream.

There is a third check the canon example hands you as a gift. When two scores
in a row come out equal, their shares are equal, exactly - and you can see
that without exponentiating anything, because equal inputs to any function
give equal outputs. Whenever a row has ties, the arithmetic gets easy and the
answer gets exact. Two rows in the worked example have ties, which is why one
output component lands on precisely 2 with no rounding, and why the masked
second row splits precisely half and half. Ties are not luck; they are
handles, and looking for them first will save you most of the arithmetic.

## Causal masking

The picture is a room where everyone sits in a row and can turn their head to
the left but not to the right.

Position 1 turns left and sees nobody, so it has only itself. Its output is
its own goods, unchanged - not because a rule says to pass it through,
but because when there is only one thing to average, the average is that
thing. Position 2 sees itself and position 1, and splits its attention between
them. The last position sees everyone and behaves exactly as it would with no
restriction at all.

The critical detail is *when* the restriction is applied, and there is a
concrete way to feel it. Attention is a budget: every position has exactly one
unit to spend and must spend all of it. If you tell position 2 up front that
only two candidates exist, it divides its unit between those two and spends
the whole thing. If instead you let it divide its unit among all the
candidates including the ones to its right, and only afterward confiscate what
it allocated rightward, then position 2 has spent less than a full unit and
its output is quietly shrunk toward nothing.

That shrinkage is not random noise. It is worst at the beginning of the
sequence, where the most got confiscated, and disappears entirely at the end,
where nothing did. So a model with the bug learns a systematic fade that
depends on where a token sits and how long the sequence is - a bug that
trains, produces a plausible loss curve, and is miserable to find later.
Restrict first, then divide. Never divide, then restrict.

One more thing the head-turning picture gets right by accident and it is worth
making explicit. Turning your head left tells you nothing about *how far* left
someone is. The restriction says who is visible, not who is adjacent. If the
token vectors did not already carry their own position, a model could see its
past perfectly and still be unable to tell the previous word from one four
hundred words back. Order is written into the tokens themselves, before any
of this starts. The mask only enforces the arrow of time.

## What attention costs

Everyone in the room reads everyone's advertisement. That is the cost, and
that is the whole story.

Twenty people, and there are four hundred readings. Forty people, sixteen
hundred - not eight hundred. Doubling the room quadruples the work, because
each new person both reads everyone and is read by everyone.

The trap is that this does not feel expensive, and the reason it does not feel
expensive is worth naming. All four hundred readings happen at once. Nobody
waits for anybody. So the room finishes in about the time one reading takes,
and it feels instantaneous compared to the bucket brigade, which finishes in
the time twenty passes take.

But finishing fast and doing little work are different things, and this is the
distinction that trips up people who are otherwise very good at performance
reasoning. The room did four hundred readings. It did them all at
once, on hardware built to do enormous numbers of small things at
once. Fill that hardware and the illusion holds. Overflow it - which happens
somewhere in the tens of thousands of tokens - and the quadratic term stops
being hidden and starts being the entire bill.

This is the answer to why a long conversation costs disproportionately more
than a short one, why context-window pricing bends upward instead of running
straight, and why so much architectural effort goes into letting people read
only some of the advertisements instead of all of them.

## What you now know

Attention is a room where everyone announces what they want and what they
have, everyone scores everyone, and everyone walks away with a blend of the
room's contents weighted by those scores. Nobody decides anything. Nobody
waits for anybody. The output of each position is a point inside the shape
outlined by the available contents, which is why one attention layer alone
cannot invent anything new. And everybody reading everybody is what makes long
context expensive - not an implementation detail anyone can optimize away,
only what the design costs.

The one head in this unit produces a single blend per position. Real models
run many rooms side by side, each scoring on different criteria, and then
stack the whole arrangement dozens deep so that later rooms are blending
things earlier rooms already blended.
