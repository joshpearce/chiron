---
unit: u9
depth: more-intuition
---

## The trace, and how to read it

Picture the whole thing as a factory floor rather than a program.

At one end, a loading dock: a truck arrives with a long paper tape, one word
fragment per inch. At the other end, a single machine that prints the receipt
one word fragment at a time. In between, sixty-four identical work
stations arranged in a line.

The counter-intuitive part, and the thing this unit exists to fix, is that the
tape does not travel through the stations one inch at a time. The entire tape
lies flat on a conveyor as wide as it is long, and each station operates on
every inch of it simultaneously. The stations are the pipeline; the tape is not.

The second counter-intuitive part is where the time goes. Getting the whole
8,000-inch tape through all sixty-four stations takes about a second. Printing
each subsequent inch of receipt costs nearly as much time as the whole tape
did, even though it is one inch. If that seems backwards, you have found the
exact place where intuition fails and the rest of the unit is worth reading
slowly.

## Stage 1: the harness builds one flat token stream

The mental image to delete: an envelope with labeled compartments, one marked
SYSTEM, one marked USER, one marked TOOLS, handed to the model which opens the
compartments it cares about.

The mental image to install: a single long strip of paper tape, and the labels
are written *on the tape* in a distinctive handwriting the model has learned to
recognize. Anyone who can write on the tape can write in that handwriting.

This is why "the system prompt has higher priority" is a statement about
training, not architecture. Imagine a clerk who has processed ten million of
these tapes, and in every one of them the instructions written after the
SYSTEM squiggle turned out to be the ones that mattered. The clerk now weights
them heavily. That is a habit, not a rule. Write a convincing SYSTEM squiggle
halfway down the tape and the habit fires on it too.

The everyday version: a conversation is not a filing cabinet the model consults.
It is a single sheet of paper that gets longer, and every turn you hand over the
whole sheet from the top, again.

## Stage 2: bytes to token IDs to vectors

The embedding table is a city, and every token is an address.

The mistake is thinking the address *contains* the building. It does not. "1600
Pennsylvania Avenue" holds no bricks. What it holds is a location, and the
location is useful only because of what is near it, what is far from it, and
which direction you travel to get from one place to another. Move the same
coordinates to a different city and they point at a parking lot.

The famous `king - man + woman = queen` trick is not evidence that meaning is in
the vector. It is evidence that the *streets run in consistent directions* -
that "walk two blocks north" means roughly the same change in role across many
neighborhoods. Direction carries meaning; position alone does not.

This also gives you the right picture for the residual stream, which starts
here. Think of each token's 4096 numbers as a spot on an enormous pinboard, and
the whole prompt as 8,192 pins. Everything the network does from here is nudging
pins. It never picks a pin up and replaces it.

## Stage 3: prefill - 8,192 tokens, one forward pass

Here is the picture that fixes the "transformers read left to right" instinct.

Imagine a hall with 8,192 people standing in a row, each holding a card. A bell
rings. Simultaneously, every person looks at the cards held by everyone to their
left - not to their right, they are wearing blinders in that direction - and
writes a note to themselves based on what they saw. The bell rings again, and
they all do it once more, sixty-four times.

Nobody waits for anybody. Person 7,000 does not need person 3 to "finish" -
person 3's card was already written down before the bell rang. The blinders (the
causal mask) create a *dependency* structure that looks sequential when you draw
it as an arrow diagram, but the actual work happens in sixty-four simultaneous
sweeps, not 8,192 sequential steps.

Now the cost picture. Each bell ring, every person looks at everyone to their
left. Person 1 looks at nobody. Person 8,192 looks at 8,191 people. Total glances
is roughly half of 8,192 squared. Double the number of people in the hall and the
glances quadruple. That is the whole of "long context is expensive," and no
amount of hiring more clerks to count glances changes the number of glances.

The reason it still finishes in a second: the hall is enormous and the glances
happen in parallel. Parallelism buys you wall clock. It never buys you a smaller
number.

## Stage 4: inside one block - residual stream, heads, and the router

**The residual stream as a shared whiteboard.** Sixty-four consultants walk past
a whiteboard in sequence. Each one reads what is on it, thinks, and *adds* a note
in the margin. None of them erases. The final state of the whiteboard is the
original problem statement plus sixty-four annotations.

This is why removing one consultant barely hurts. The whiteboard still carries
the problem and sixty-three annotations; the downstream consultants read a
slightly thinner board and mostly cope. If instead each consultant had wiped the
board and written a fresh version, removing one would leave everyone downstream
reading something that was never written.

The old story - "the skip connection is so feedback can reach the early
consultants during training" - is true and is also the less interesting half.
The whiteboard is a communication channel first and a feedback path second.

**The router as a bank of parallel tellers, not a department directory.** Picture
128 identical-looking tellers. A customer walks up. A greeter glances at the
customer for a fraction of a second and sends them to eight specific tellers.

The instinct is that the greeter is reading the customer's *purpose* - mortgage
here, foreign exchange there. Watch the actual assignments and that story
collapses immediately: two consecutive words of the same sentence go to
completely different sets of tellers, and a teller who handles a word in your
code prompt also handles words in a recipe. Whatever the greeter is reading, it
is not topic.

Two things shaped the greeter's habits. One: over training, tellers who happened
to be useful got sent more customers, and got better at whatever they were
already receiving. Two: management punished the greeter whenever any teller had
a long queue while others sat idle. The second pressure is enormous and is why
the assignment looks arbitrary. Specialization exists, but it grew like a
footpath across a lawn - real, visible, and not designed.

## Stage 5: the last position becomes a distribution

Everything the model will ever know about what comes next is frozen the instant
the sixty-fourth consultant puts down the pen. The final whiteboard state gets
compared against all 128,000 addresses in the city from Stage 2, and each
comparison produces a score.

Now the picture for temperature that keeps people honest.

Imagine the scores as the heights of 128,000 hills on a landscape. Sampling is
dropping a ball somewhere on this landscape, with taller hills more likely to
catch it. Temperature does not move a single hill. What it does is stretch or
squash the *vertical* axis of the whole landscape at once. Cold temperature
stretches it: the tallest hill becomes a spire and everything else flattens into
the plain, so the ball almost always lands on the spire. Hot temperature squashes
it: all the hills approach the same modest height and the ball lands anywhere.

At absolute zero you always land on the tallest hill. That is not "what the model
really believes." It is the tallest hill on a landscape you were already given.
Nothing about the terrain changed; you changed how steeply you are willing to
read it.

Top-p is a different operation on the same landscape: walk downhill from the
peak, keep hills until they account for most of the land area, and bulldoze
everything else to sea level. When the landscape has one spire, you keep one
hill. When it is a gentle plain, you keep thousands. The bulldozing is why a
low top-p makes a model repetitive: you have removed the possibility of surprise
before the ball was ever dropped.

## Stage 6: decode, and what the KV cache actually bought

**The cache is a scratchpad, not a diary.** Return to the hall of 8,192 people.
When the bells finished ringing, each person left their card face-up on their
chair and went home. Now one new person joins the end of the row. They need to
glance at every card to their left - but the cards are already on the chairs. The
new person does not need everyone to come back.

That is the entire KV cache. Notice what it means: if a fire destroyed all the
cards, you could get every one of them back exactly by calling the same people
back in and ringing the bells again. Nothing on those cards is information you
do not already have in the original tape. It is saved labor, not saved knowledge.

**Why generating is slow when reading was fast.** The building has a vault
holding eleven gigabytes of reference manuals, and every single forward pass
requires wheeling the entire vault's contents past the workers. When 8,192 people
were working at once, one pass of the manuals served all of them - superb value.
When one new person joins, the entire vault still has to roll past, for one
person. The manuals are the bottleneck, not the thinking.

This is the intuition behind nearly every inference optimization you have heard
of. Printing the manuals on thinner paper is quantization. Having one clerk who
guesses the next four answers so the vault-roll can check all four at once is
speculative decoding. Serving forty customers off the same vault-roll is
batching. They are all attacking the same thing: the manuals move regardless, so
get more work out of each move.

And the chairs pile up. Every generated token leaves another card on another
chair, and every new person has to scan all of them. At 8,192 chairs the scan is
a minor addition to the vault roll. At 100,000 chairs the scanning takes longer
than the vault. Nothing got harder to think about; there is just more furniture.

## Stage 7: the tool call, the break, and the resume

The model does not read the file. It writes down a *request slip* and stops
talking.

Picture it precisely: the model is speaking, mid-sentence it says "please read
src/auth/middleware.ts", and then goes completely silent. Not thinking-silent -
switched-off silent. A clerk outside the room, who is not the model and has never
been the model, picks up the slip, walks to the filing cabinet, photocopies the
file, staples the copy onto the end of the paper tape, and switches the model
back on. The model, restarted, sees a tape that now happens to contain the file
contents, and continues.

There is no moment at which the model "has" the file and is deciding what to do
with it, distinct from the file simply being more tape. Nothing about the request
is special: the slip is text, the response is text, the model's ability to use
tools is entirely the learned habit of writing well-formed slips.

**Why appending is cheap and editing is ruinous.** Back to the chairs. Stapling
3,400 new pages to the end of the tape means 3,400 new people join the row, glance
leftward at cards that are all still sitting on their chairs, and sit down. Cheap.

But suppose the clerk instead goes back and changes a word on page 2 - which is
what "let me summarize the conversation so far and replace it" does. Everyone
from page 2 onward was looking leftward when they wrote their card, and page 2
was in their field of view. Every card after page 2 is now wrong. Everybody comes
back. Everybody re-glances. You have paid for the entire hall again.

The practical rule falls right out: put the things that never change at the front
of the tape, and the things that change constantly at the back. Every byte you
move earlier in the prompt is a byte that can invalidate everything after it.

## The cost model that falls out of the trace

One image to carry away.

Think of a conversation as a train that only ever adds cars. Each turn couples
another car onto the back. The locomotive has to drag every car, so each turn
costs a little more than the last. That alone would make cost grow like the
number of turns squared, and it does.

But there is a second effect stacked on it. Every new car has to be inspected
against every existing car, so the inspection work grows with the length of the
train too. Two independent quadratic pressures, from two different mechanisms -
attention looking at everything, and the pile of chairs having to be re-scanned
for every single word generated.

This is why an agentic session feels fine for ten minutes and then feels
mysteriously sluggish, and why the bill is never linear in how much you said.
Nothing degraded. Nothing is misconfigured. The train just got long.

Every optimization anyone has invented is one of two moves: make each car
lighter (quantization, grouped-query attention, smaller active parameter counts)
or avoid re-dragging cars you already dragged (prefix caching). Nobody has found
a way to make the train shorter without throwing cars away, which is what
compaction is, and why it always costs you something real.
