---
unit: u6
depth: more-intuition
---

## Where inference sits in the stack

Picture a mountain range. Training carved it - forty-five days of erosion, and
every valley and ridge is now fixed rock. Inference is water finding its way
down. The water can move fast or slow, take one path or wander among several,
carry more or less silt. It cannot move the mountains.

Every technique in this unit is a property of the water. Temperature is how
adventurous the water is about taking a side channel. The KV cache is a note
someone left saying "you already surveyed this stretch, do not re-survey it."
Quantization is a slightly blurred contour map - the mountain is still there,
you are reading its shape at coarser resolution. Speculative decoding is sending
a scout ahead who guesses the route, with a rule that discards any guess the
real terrain contradicts.

If at any point you find yourself thinking "and this changes the mountain," stop
and look again. Nothing here does.

## Temperature, top-k, top-p on one set of logits

Imagine the model's output as a pile of sand, heaped over a row of buckets - one
bucket per possible next token. After the forward pass, the heap has a shape:
tall over "Paris," a small mound over "France," a few grains over "banana." That
shape is the logits, and it is finished. Nobody touches the sand again.

The sampler is a blindfolded hand reaching into the pile to grab one grain, and
whichever bucket that grain came from is your token. Now:

**Temperature is a shaking of the pile before you reach in.** Low temperature
shakes it so the tall pile gets taller and everything else settles into a thin
film - almost every grain you could grab is from the tall bucket. High
temperature shakes it the other way, spreading sand outward until the heap is
nearly a flat sheet and every bucket is a live possibility. The crucial thing:
shaking never moves sand into a bucket that had none, and never empties a bucket
that had some. It redistributes a fixed amount of sand across a fixed set of
buckets, and it never reorders them - the tallest pile stays the tallest, at
every temperature.

**top-k is a fence.** Put a fence around the two tallest buckets and sweep the
rest of the sand into them. Three buckets that had sand now have none, not
because they were shaken but because you fenced them off. This is a categorical
difference from temperature: fencing removes possibilities, shaking only
reweights them.

**top-p is a fence you place by walking.** Start at the tallest bucket and walk
downhill, counting sand as you go, and plant the fence the moment you have
counted 90% of it. On a peaked heap you plant it after one bucket. On a flat
heap you walk a long way first. The fence's *position* depends on the heap's
shape, which is exactly why shaking the pile first (temperature) changes where
the fence lands. That is the interaction the canon section computed: same
`top_p=0.9`, one survivor at $T=0.5$, four at $T=2.0$.

A useful mental image for the numbers: think of "effective number of choices."
At $T=0.5$ the sampler is choosing among roughly 1.1 options. At $T=1$ about
1.9. At $T=2$ about 3.5. At infinite temperature, all 5. Temperature is a dial
on how many options are genuinely live, and the model's opinion about their
ranking is untouched the whole way.

## T=0 is not truth mode

Here is the picture that makes $T=0$ click.

Imagine a very well-read person who has never once been told whether anything
they read was true. They have read everything, and they have an extraordinarily
good ear for what *sounds like* it belongs next. Ask them a question and they
give you the continuation that fits best.

Now set $T=0$. What you have done is instruct them: "always say the first thing
that comes to mind, never the second." That is it. That is the whole
intervention. If the first thing that comes to mind is wrong, it will be wrong
every single time, with total consistency, and the consistency will feel like
confidence.

This is why $T=0$ hallucinations are the sticky ones. A hallucination at $T=1$
flickers - run it again and you might get a different wrong answer, or a right
one, and the variance itself is a signal that the model is unsure. At $T=0$ the
flicker is gone and you get the same fabricated function name in perfect
deadpan, ten runs out of ten. You have removed the model's only visible tell.

The greedy-suboptimality point has a good picture too. You are driving in fog
with a map that only shows the next intersection. At each one you take the road
that looks widest. Take enough turns this way and you can end up on a wide road
that dead-ends, having passed a narrow turnoff that led to the highway. The
model has no lookahead - it picks the widest road at each intersection, and
"widest at every step" is not "shortest overall." Beam search is driving three
cars down three roads at once and keeping the best; it helps, and it is still
fog.

## The KV cache is not memory

The wrong picture: the cache as a diary the model keeps, accumulating impressions
of your conversation, so that erasing it erases what the model has learned about
you.

The right picture: you are a translator working through a long document,
sentence by sentence, and for each new sentence you need to consider every
sentence that came before. The naive approach is to reread the whole document
from page one for every new sentence. The KV cache is the stack of index cards
you made the first time through - one per sentence, holding what you would have
extracted by rereading it.

Now ask the question that settles it: if someone burns your index cards, what
have you lost? Not information. You still have the document. You lost the *hours*
you spent making the cards, and you will spend them again. That is the entire
content of the KV cache. Information that is fully reconstructible from something
you still have is not memory; it is saved effort.

The reason the cards can be made once and reused forever is the causal mask.
Card 7 was written by reading sentences 1 through 7. Sentence 8 arriving cannot
retroactively change what sentences 1-7 say, so card 7 is still valid. If the
document were a crossword instead - where a later clue genuinely changes how you
read an earlier one - no card would ever stay valid, and you would reread
everything every time. That is the difference between a causal decoder and a
bidirectional encoder, and it is why only one of them gets to be fast.

For the size: 96 kilobytes per token is roughly a small image thumbnail, per
word-and-a-bit of your conversation. Stack up a 130,000-token context and you
have twelve gigabytes of index cards - a stack about as heavy as the model
itself. When people say long context is expensive, this pile is half of what
they mean.

## Prefill and decode are different machines

The image is a printing press versus a typewriter, and the surprise is which one
feels slow.

**Prefill is a printing press.** You have a 2,000-token prompt. The press inks
every character of every page at once and slams down. One motion, whole
document. Enormous total work, and it is over in an instant because the machine
was built to do the whole page in one stroke.

**Decode is a typewriter.** Each key press produces one character, and you
cannot press the next key until you have seen what the last one printed - the
text you are writing depends on the text you have written. Far less total work,
and it takes forever, because the machine's capacity to do a whole page at once
sits idle while it produces one character at a time.

That mismatch is the entire performance story of local inference. Your laptop
can do enormous arithmetic per second, but during decode there is only one
token's worth of arithmetic to do, and meanwhile it must haul all 1.8 gigabytes
of active weights out of memory to produce that one token. It is a freight train
dispatched to deliver one envelope. The train's speed is not the problem - the
loading and unloading is.

This is also why quantization feels like magic on a laptop and barely registers
on a server. Shrinking each weight from two bytes to half a byte means a smaller
train for the same envelope. On a laptop, where the bottleneck is the hauling,
you go nearly four times faster. On a server running a hundred users at once,
each train is already carrying a hundred envelopes and the hauling was never the
constraint.

### Why long context gets expensive twice

Picture a meeting where everyone must acknowledge everyone else before speaking.
Ten people: manageable. A hundred people: each new speaker has a hundred
acknowledgments to make, and if a hundred people speak, that is ten thousand
acknowledgments. Doubling the room does not double the ceremony - it quadruples
it.

That is attention. And notice the shape of it: the meeting is not slow because
people are slow. It is slow because the number of pairwise acknowledgments grows
as the square. Giving everyone a faster mouth (more parallel hardware) postpones
the problem; it does not change its shape. This is the trap in "attention is
parallel so it must be cheap" - parallelism buys you a bigger room, not fewer
handshakes.

The two costs are worth keeping separate in your head, because they behave
differently. The index cards (memory) pile up in a straight line: twice the
context, twice the stack. The handshakes (compute) pile up as a square: twice
the context, four times the ceremony to produce the same amount of new text.

## Quantization: what four bits hold

The picture people have is a JPEG of a photograph: compress it hard and details
vanish, faces smear, text becomes unreadable. Applied to a model, that picture
says 4-bit quantization smears away most of what the model knows.

Try this picture instead. You have a wooden sculpture, and you must describe
every surface point's height using only sixteen height levels instead of a
continuous ruler. Every point still gets a value. Nothing is deleted. The
sculpture comes back slightly faceted - a bit polygonal, like a low-poly
render - but the pose is the pose, the proportions are the proportions, and it
is unmistakably the same sculpture. You did not remove its arm.

Now the important part, which is why a faceted sculpture still works. The model
does not consult individual weights the way a database consults individual rows.
Every computation is a sum over thousands of weights at once. Rounding each of
those thousands by a small amount and then adding them up gives you a sum that
is off by a small amount - the errors do not conspire, they are a bit of
haze on the result. Nothing catastrophic happens because nothing depended on any
one weight being exact.

There is also a reason the sculpture was carved robust in the first place. It
was shaped by a process that was itself shaky - stochastic gradient descent,
noisy batches, dropout. A sculpture carved by a shaky hand is necessarily one
whose shape does not depend on any single millimeter, because if it did, the
shaky hand would never have found it. Quantization noise is one more tremor
applied to something already selected for tremor-tolerance.

The right slogan is: **quantization keeps directions and spends precision.** Which
way each weight points, and roughly how big it is relative to its neighbors, is
what survives. Its fourth decimal place is what you paid.

## Where quantization falls off the cliff

Degradation from quantization is not a slope. It is a plateau with an edge, and
the edge is not where bit-counting predicts.

The reason for the edge: think of each weight's value as being written in a
notebook with a fixed number of ruled lines. With sixteen lines you can place
every weight close enough to its true height that the sculpture still reads.
With eight lines the facets get chunky. With four, distinct features start
merging into the same line - two weights that meant different things now say the
same thing - and once features merge, no downstream computation can pull them
apart. That merging is the cliff. Below it you are not adding noise anymore, you
are erasing distinctions.

Two failure modes are worth picturing because they are the ones you will actually
hit.

**One loud voice ruins the room.** Groups of weights share a single scale - one
ruler for the whole group. If one weight in the group is enormous, the ruler must
stretch to reach it, and now the ruler's smallest division is huge, so the other
sixty-three weights all get rounded coarsely. One shouter forces everyone else to
whisper into a microphone calibrated for shouting. Real transformers grow a few
of these shouting dimensions as they scale, which is why good quantizers hunt
them down and handle them separately, and why naive quantizers hit the cliff much
earlier than they should.

**Damage compounds along chains, not across facts.** Here is the observation that
most cleanly kills the "compressed database" picture. If quantization deleted
knowledge, the obscure facts would go first - the rarely-visited rows. That is
not what happens. Obscure facts survive fine. What degrades first is a long
chain of reasoning: forty steps where each one is slightly nudged, and the nudges
accumulate until step thirty veers off. A photograph compressed too hard loses
detail everywhere at once. A quantized model loses its footing on long walks
while remembering the trivia perfectly. Those are different failure signatures,
and the one you observe tells you which picture is right.

## Speculative decoding: exact, not approximate

The picture is an apprentice and a master carpenter.

The master is slow and expensive, and every cut they make is correct by
definition. The apprentice is fast and cheap and mostly right. So: let the
apprentice make the next four cuts, quickly. Then the master glances at all four
*in a single look* - and this is the crux, because looking at four cuts takes the
master barely longer than looking at one. Cuts that the master agrees with are
kept. At the first cut the master would not have made, the apprentice's work
stops there and the master makes that cut themselves.

Why is the result identical to the master working alone? Because the master
never accepts a cut they would not have made. The apprentice can only *save
time*, never *introduce a decision*. If the apprentice is excellent, the master
rubber-stamps four cuts per look and the job goes four times faster. If the
apprentice is hopeless, every cut gets rejected, the master makes each one
themselves, and you have wasted the apprentice's time but the furniture is
exactly the same furniture.

That last sentence is the whole guarantee, and it is why speculative decoding is
not a quality tradeoff. A bad draft model makes it *slower*, never *wrong*.

The one refinement worth knowing: the master does not accept on a strict "would I
have done exactly this" test, because that would bias the work toward whatever
the apprentice happens to favor. When both master and apprentice are working
probabilistically, the master uses an acceptance rule with a repair step - and
the repair is calibrated so that when you average over many jobs, the
distribution of finished furniture is *exactly* the master's own. Not close.
Exactly. The mathematics of that repair is in the deeper-math treatment; the
intuition is that the master keeps precise account of how much the apprentice
over-favored each option and gives back exactly that much on the rejections.

Where the trick stops helping: the apprentice only saves time because the master
was idle between cuts. In a busy shop where the master already has a queue of
jobs to work through, there is no idle time to reclaim, and the apprentice is
just another person in the way. That is the difference between your laptop
serving one reader and a datacenter serving ten thousand.

## What this unit did and did not change

Return to the mountain range. Four techniques, and the useful question for each
is: did the water do this, or did the mountain move?

- Temperature, top-k, top-p: the water chose its channel differently. Mountain
  untouched.
- KV cache: the water stopped resurveying ground it had already surveyed.
  Mountain untouched, and provably so - burn the survey and redo it, you get the
  same map.
- Quantization: the only one where the mountain actually moved, and it moved by
  about nine percent in a random direction, on terrain that was selected during
  training for being insensitive to exactly that kind of nudge.
- Speculative decoding: a scout ran ahead guessing the route, with a rule that
  guarantees the water ends up in the same place regardless of how bad the
  scout was.

The reflex worth building: when a model does something wrong, the first question
is which of these you are looking at. Almost always the answer is "none of
them" - the mountain has that valley in it, and no amount of adjusting how the
water flows will fill it in.
