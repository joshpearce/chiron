## Decode is bandwidth-bound, and that is the whole motivation

Picture the model's weights as a very long conveyor belt loaded with boxes, and
the GPU as a worker standing beside it. To produce one token, every box has to
ride past the worker once. The worker's job at each box is trivial - one
multiply, one add - and they finish it long before the next box arrives. Then
they wait.

The belt runs at 153 GB/s. That is the entire performance story. Making the
worker faster - a bigger GPU, more cores, more FLOPs - changes nothing, because
the worker was already idle 94% of the time. The only two things that matter
are how fast the belt runs and how many boxes are on it.

You cannot speed up the belt; that is soldered to the machine. So the question
becomes: do all the boxes really have to ride past?

Now hold onto the geometry of *when* this is true. During generation, the
worker handles one token at a time, so each box gets one unit of work. But when
the model reads your prompt, all two thousand of its tokens are on the bench at
once - the same box arrives, and the worker does two thousand multiplies before
it moves on. The belt is no longer the constraint; the worker is. Same machine,
same weights, opposite bottleneck. This is why pasting a long document feels
instant and watching the reply appear feels slow.

## The router: replacing one MLP block with N of them

Picture the residual stream as a wide, slow river carrying the token's state
downstream. In a dense model, at every one of the forty-eight stations along
the bank, the same single machine dips into the river, computes something, and
pours its result back in.

In an MoE model, each station has a hundred and twenty-eight machines instead
of one - all the same shape, all built to do the same kind of job, all trained
- plus a tiny gatekeeper. The gatekeeper looks at the water arriving, picks
eight machines, decides how much each of them counts, and pours the blended
result back into the river. The other hundred and twenty sit idle. They are
still there, still occupying floor space, still costing rent. They just do not
run this time.

Two pictures to keep straight.

**The decision is small.** The gatekeeper is not a supervisor with a clipboard.
It is a single measurement of the arriving water against a hundred and
twenty-eight reference patterns, and the eight patterns that match best win.
That measurement is thousands of times cheaper than running even one machine.

**The decision is remade constantly.** Not once per conversation, not once per
sentence. Every token, at every one of the forty-eight stations, from scratch.
The word "the" arriving at station 12 gets a different eight than the same word
"the" arriving at station 13, and a different eight again from the "the" three
words later. Whatever picture you are holding of a request being classified and
routed to a destination, discard it - the routing happens forty-eight times per
token and the token never leaves the river.

And downstream, nothing knows. The water that flows past station 12 carries no
record of which machines touched it. Station 13 sees a river, not a routing
decision.

## Experts are not domain specialists

Here is a way to feel why the specialist picture cannot survive.

Imagine you are told to split all the world's text among a hundred and
twenty-eight workers, and the only hard rule is that every worker must get
almost exactly the same amount of text. Try splitting it by subject. Most text
is ordinary prose, so the "general English" worker drowns while the "Icelandic
legal filings" worker sits idle for weeks. The rule is violated immediately and
badly. Subject matter is a wildly unbalanced way to cut up language, and no
amount of cleverness fixes that, because it is a property of the text, not of
your scheme.

Now imagine you are told to split it so that every worker gets an equal share
*and* each worker's slice is internally predictable. You would end up cutting
along something much finer and much more evenly distributed than topic -
whether the piece ends in punctuation, whether it starts a word or continues
one, whether it is a numeral, how it tends to be followed. Those properties are
spread evenly through all text regardless of subject, which is exactly what the
equal-share rule demands.

That second scheme is roughly what the router discovers, and nobody designed
it. Two pressures produced it: a balance rule that punishes any uneven cut, and
a usefulness rule that rewards cuts which help predict the next token. Neither
pressure contains any preference for the cut being describable in words.

So when you look at a trained MoE and ask "what is expert 47 for", the honest
answer is that the question has no answer of the kind you want. Expert 47
handles roughly one-sixteenth of all tokens, drawn from every subject, sharing
some family resemblance that is real (it demonstrably reduces loss) and
mostly not nameable. It is a partition without a name for its parts.

One more thing that kills the specialist picture on contact: an expert is not a
small model. It is one of 128 parallel fragments in the middle of one of forty-eight layers. There is no
"code expert" you could extract and run, any more than there is a "verbs" chunk
of your brain you could remove and interview. It is a fragment of a computation,
not a participant in a committee.

## Total versus active: the tradeoff table

Hold two separate budgets in mind, because conflating them causes every mistake
in this section.

**The shelf** is how much you must store. Every expert has to be on the shelf,
because you do not know until the moment arrives which eight the next token
will want. If it is not on the shelf, the token that needs it stalls. The shelf
is sized by the *total*.

**The trip** is how much you carry per token. Eight experts out of a hundred
and twenty-eight, plus the parts that always run. The trip is sized by the
*active*.

MoE makes the trip ten times shorter without shrinking the shelf at all. That
is the entire deal, stated in one sentence, and it explains every consequence:
you go fast (short trip), you need a big machine anyway (big shelf), and you
get the knowledge of a large model (the shelf is what was trained).

Why this is a laptop-shaped bargain specifically: a laptop with 64 GB of
unified memory has a generous shelf and a narrow doorway. It can store a lot
and carry very little per trip. MoE is the architecture that spends the
resource you have to buy the one you lack. Put the same model on a datacenter
card with an enormous doorway and the bargain is much less interesting - there,
the constraint was never the doorway.

And here is the part that surprises people. Serve fifty users at once and the
trips merge into one big trip. User A's token wants experts 3 and 47, user B's
wants 12 and 91, and by the time you have fifty users the combined trip is
fetching nearly everything on the shelf. The short-trip advantage was never a
property of the model; it was a property of *one token at a time*. It belongs
to you on your laptop and it evaporates in a serving cluster.

## What MoE costs you

The temptation, once you understand the short trip, is to shrink the shelf -
keep only the popular experts in memory and fetch the rest from disk when
needed. It does not work, and the reason is worth feeling rather than
calculating. The eight experts you need change every single token, so a disk
fetch is not a rare miss you can amortize; it is the steady state. You would be
carrying the same short trip, but over a road thirty times slower than the
one you were on. You would end up below where the dense model started.

The deeper cost is during training, and the shape of it is a familiar one to
anyone who has watched a system with positive feedback. Whichever expert is
slightly better early gets picked slightly more, gets trained slightly more,
becomes better still. Left alone, this runs away completely: a handful of
experts absorb everything and the rest are never selected, never trained, dead
weight on the shelf forever. You would have paid to store a large model and be
running a small one inside it.

The balance rule exists purely to damp that loop. It is a nudge, not a barrier
- a pressure toward evenness applied at every step. Tune it too weakly and the
runaway happens. Tune it too strongly and you have forced the gatekeeper to
pick evenly regardless of what would actually help, which is the same as
picking at random. The good setting is somewhere in the middle and it is not
obvious where.

There is also the thing nobody tells you until you have tried to build one.
A single machine per station is a clean, simple, blindingly fast operation. A
hundred and twenty-eight machines with a gatekeeper is a sorting problem: pull
the tokens apart by destination, run each group, put them back in order. It is
more code, more memory, more ways to be wrong, and across multiple GPUs it adds
a full round of shuffling at every station. The elegance of the idea and the
tidiness of the implementation are inversely related.

Last, the honest trade. Take the thirty-five billion parameters you spent and
arrange them densely instead, and the dense model would be better. Not
dramatically, but reliably. What you bought by scattering them into experts is
not quality per parameter stored - it is quality per parameter *read*, which is
the only kind you can afford when the belt runs at 153 GB/s and you want the
words to appear as fast as you can read them.
