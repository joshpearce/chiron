## The machine you are actually running on

Picture a workshop.

At the bench is a machinist who can perform one operation in a second. Behind
her, an arm's reach away, is a small tray holding a few parts. Across the room
is a shelf holding a few hundred. In a warehouse on the other side of town is
everything else, and a truck runs between the warehouse and the workshop.

The machinist is fast. She is not the problem, and she has never been the
problem. If every part she needs has to come from the warehouse, she spends the
whole day standing at the bench with her hands empty, and hiring a second
machinist changes nothing except that now two people are standing there.

That is the picture. The registers are the tray at her elbow, on-chip SRAM is
the shelf across the room, HBM is the warehouse, and the network to other nodes
is a warehouse in another city. The whole discipline of making accelerators go
fast is the discipline of arranging the work so that when a part arrives from
the warehouse, it gets used for a hundred operations before it is put down.

The energy numbers say the same thing in a different key. Think of it as
distance. Electricity spent on a computation is spent moving charge along a
wire, and the wire's length is what it costs. Inside a chip, wires are
micrometers long. To the memory chip stacked next to it, millimeters. Across a
board, centimeters. The arithmetic happens at the far end of a wire whose
length is the price tag, and the price scales with the distance, not with the
difficulty of the sum.

This is why the phrase "the GPU is idle" is the normal state of affairs rather
than an alarm. When you are generating a token on your laptop, the arithmetic
units are doing nothing about 95% of the time. Not because anything is broken -
because the parts are still on the truck.

### What a tensor core is, without the block diagram

A general processor is a person following instructions one at a time: fetch a
number, fetch another, multiply, store. A vector unit is that person handling
sixteen numbers per instruction instead of one.

A tensor core is a jig. It is a fixture on the bench built to hold a small grid
of parts in a specific arrangement, and it performs the whole grid of
multiply-and-add operations as one motion. You cannot use it for anything
except that shape. But when your work *is* that shape, one motion replaces
hundreds.

That constraint is why everything in this book is written as a matrix multiply.
The hardware has a jig, the jig fits matrices, and any computation you can
express as matrices gets to use it. Anything you cannot gets done by hand.

## Arithmetic intensity and the roofline

Here is the only mental image you need for this entire unit.

Draw two lines on a piece of paper. One rises diagonally from the bottom left.
One runs flat across the top. They meet at a corner. The region under both lines
is everything the machine can do.

The diagonal line is the memory system: the more work you get out of each byte
you fetch, the more work per second you can do, in direct proportion. The flat
line is the arithmetic: no matter how much work you get out of each byte, you
cannot exceed the number of multiplications the silicon can physically perform.

Every workload you will ever run is a single point somewhere along the
horizontal axis. Some of them fall to the left of the corner, under the
diagonal, and for those the answer is "your memory system decides." Some fall
to the right, under the flat part, and for those the answer is "your arithmetic
decides." There is no third case.

The corner is where the two constraints happen to be equally binding. Everything
about a piece of hardware that matters for performance is the location of that
corner, and everything about a workload that matters is which side of it the
workload lands on.

Now the point that makes this useful rather than merely tidy: **the corner
moves right with every hardware generation.** New chips gain arithmetic faster
than they gain bandwidth. So the target keeps getting harder to hit, and a
workload that was compute-bound on last year's chip can be bandwidth-bound on
this year's while doing exactly the same thing. Bigger machines are hungrier
machines. That is why a laptop is a more sensible one-user token generator than
you would guess from the spec sheets, and why an H100 asked to serve one person
looks like a factory floor with one worker on it.

### Two ways to see the same laptop calculation

The bandwidth-bound calculation from u7 - 153 divided by 1.8 gives 85 tokens a
second - has an intuition that is almost physical.

Generating one token means every weight that participates has to make the trip
from memory to the chip. Not most of them. All of them, one round trip each.
Once it arrives it gets multiplied by exactly one number and then it is done and
discarded, because there is only one token in flight and the weight has only
one thing to do.

So generating a token is a pass through the entire active model. The time it
takes is the time to read that much data, full stop, and the number of tokens
per second is how many times per second you can read it. If your model is
1.8 GB of active weights and your memory delivers 153 GB every second, you can
make that pass 85 times a second. There is nothing else in the calculation
because there is nothing else in the physical situation.

Now change one thing: put sixty-four users' tokens in flight at once. The
weights still make one trip. But now each weight, having arrived, gets used
sixty-four times before being discarded. Same trip, sixty-four times the work.
This is why serving providers batch and why you cannot: they have sixty-four
people asking questions and you have one.

That is also the honest answer to "why is the chat interface slower than the
API benchmark." Nothing is throttled. You are one person, and one person cannot
fill a machine designed to be filled.

## FlashAttention: changing the bytes, not the FLOPs

Imagine you have been asked to compute the total of a very large
multiplication table - every row value times every column value - and then
normalize each row.

The obvious approach: get a very large sheet of paper, write out the whole
table, then go back over it row by row. If the table has eight thousand rows and
eight thousand columns, that sheet of paper is enormous. It does not fit on your
desk. So you write it in a notebook in the next room, walk back and forth to
fill it in, then walk back and forth again to read it.

FlashAttention is the observation that you never wanted the sheet of paper. You
wanted the row totals. So: take one strip of rows and one strip of columns, both
small enough to fit on your desk, compute that little rectangle in your head,
add its contribution to the running totals, and throw the rectangle away. Then
the next strip. The full table is produced, every entry of it, and no entry ever
gets written down.

The arithmetic is identical - you multiply exactly the same pairs of numbers.
The walking is what disappears.

The one genuinely subtle part is the normalization, because normalizing a row
requires knowing the row's total, and you only ever hold a piece of a row. The
trick is to keep a running total *and* a running scale factor, and when a new
piece arrives that is larger than anything you have seen, you rescale what you
have accumulated so far to match the new frame of reference before adding.
It is the same manoeuvre as keeping a running average when you cannot store all
the samples, and like a running average it gives the exactly correct answer, not
an approximation. This matters more than it sounds: people assume
FlashAttention must be trading accuracy for speed, because that is what
optimizations usually do. It is not. The outputs match.

### The picture for grouped-query attention

Here is the geometric version of why sharing key/value heads helps twice over.

Think of the cached keys as a set of reference cards laid out on a table.
Multi-head attention gives every reader their own private copy of every card,
so a table with thirty-two readers needs thirty-two sets. Grouped-query
attention has the readers share: thirty-two readers work from four sets, eight
readers per set.

The obvious saving is the table space - four sets instead of thirty-two, which
is the memory argument u6 made. The second saving is less obvious and comes
free: when a card is fetched from the back room, eight people read it before it
goes back. The trip is amortized eight ways.

Same fetch, eight times the reading. That is what "raising arithmetic intensity"
means when you strip the vocabulary off it.

## Why training and inference want different iron

The same set of instruments, two completely different jobs.

**Training is a factory.** Millions of items move through it per batch, the same
operation is applied to every one, throughput is the only metric anyone cares
about, and if any individual item takes an extra second nobody notices. You size
a factory for volume. You buy the biggest, hungriest machines you can feed, and
you organize the whole building around keeping them fed.

**Single-user inference is a kitchen making one dish to order.** The customer is
watching. The metric is how long until the plate arrives. You cannot batch,
because there is one order. All the equipment sized for volume sits idle while
one thing gets made, and the constraint is not how fast the stove is but how
fast you can get ingredients out of the pantry.

Nobody would buy a factory to run a kitchen. The reason people expect one chip
to do both is that the *operations* look identical from a distance: multiply
some matrices, add some numbers. But the shape of the demand is completely
different, and the shape of the demand is what hardware gets designed against.

The memory side has the same character. Inference needs the recipe. Training
needs the recipe, a copy of the recipe being edited, notes on every change made
over the last thousand attempts, and every intermediate result from the current
attempt in case you need to work backwards through it. Eight times as much,
minimum, for the same dish.

There is one number in this section worth carrying around as a standalone fact:
**you cannot train a 70-billion-parameter model on any single chip that
exists.** Not slowly. Not with tricks. The bookkeeping alone is over a terabyte,
and no accelerator has a terabyte. Distributing the work is not an optimization
someone chose; it is the only way the thing can happen at all.

## Four ways to split a model, and what each one stresses

Four ways to divide a large job among many workers, and you have seen all four
outside this field.

**Data parallelism** is a call centre. Every agent has the complete script and
handles their own calls. At the end of the shift everyone gets together and
agrees on updates to the script, so tomorrow they all start from the same one.
The meeting is the expensive part, and it gets no cheaper as you hire more
agents, but it also happens only once a day. Add people freely; just do not hold
the meeting every fifteen minutes.

**Tensor parallelism** is four people carrying one piano. Nobody carries a
piano; everybody carries a corner of the piano, and they have to move in step,
constantly, or the piano falls. This works beautifully when everyone is in the
same room and can see each other. Put them in four different buildings with
walkie-talkies and the piano does not move at all. This is exactly why tensor
parallelism stays inside one node.

**Pipeline parallelism** is an assembly line. Station one does its part and
passes the work along. Almost no coordination is needed - hand off the piece and
forget about it. The cost is that at the start of the day, station eight has
nothing to do until the first piece reaches it, and at the end of the day
station one has nothing to do while the last pieces finish. With eight stations
and eight pieces per day, that dead time is nearly half the day. With eight
stations and sixty-four pieces, it is a tenth. The fix is not faster hand-offs;
it is more pieces in flight.

**Expert parallelism** is a mail room where each clerk handles a different
category, and every incoming letter has to be walked to whichever clerk handles
it. If the letters happen to be evenly distributed, everyone stays busy. If
today's post is 90% one category, one clerk drowns while seven read the paper,
and everyone waits for the drowning one. This is why MoE training has fixed
per-clerk quotas and drops the overflow: an uneven day is not just slow, it
makes the whole floor's schedule unpredictable.

Real training runs use all four at once, arranged in rings: the piano-carriers
are in one room, the assembly line spans rooms, and the call-centre meeting
spans the building. The arrangement is not aesthetic. It is chosen so the
thing that needs to happen constantly happens over the shortest wire.

## Precision: range is the scarce resource, not resolution

Two rulers, both a foot long.

The first is marked in millimetres and measures from zero to thirty
centimetres. Very fine. Cannot measure a room.

The second is marked in centimetres and has a folding mechanism that extends it
to measure anything from the thickness of a coin to the length of a street.
Coarser. Measures nearly everything.

For carpentry you want the first. For surveying a site where some things are
tiny and some are enormous and you do not know in advance which is which, you
want the second, and the fineness of its marks barely matters.

Training gradients are a surveying job. Some are the size of a room and some are
the size of a hair, in the same batch, and they change by orders of magnitude
across a run. What ruins you is not measuring the hair to the nearest
centimetre. What ruins you is a ruler that cannot represent the hair at all and
reports its length as zero - because a gradient reported as zero is not a
slightly wrong update, it is a weight that never moves again.

That is the difference between fp16 and bf16, and it is why the two 16-bit
formats behave nothing alike. fp16 is the millimetre ruler. bf16 is the folding
one.

Loss scaling was the workaround for the millimetre ruler: multiply everything by
a thousand before measuring so the small things land on the scale, then divide
the answers back down. It works. It also requires watching constantly for things
that overflowed off the top of the ruler after being multiplied, backing off,
and re-measuring - an entire supervision loop that exists only because the ruler
was the wrong shape. Once folding rulers were available in hardware, everybody
switched and the supervision loop was deleted.

### Where coarseness does bite

There is one job where coarse marks genuinely fail, and it is worth seeing
because it is the exception that gives the rule its shape.

Suppose you are keeping a running total on a whiteboard, in a notation that
holds three significant figures. The total stands at 1.00 metres. You need to
add one tenth of a millimetre. You write down the result: 1.00 metres. You add
it again: 1.00 metres. You could add it ten thousand times and the whiteboard
will read 1.00 metres forever, because the increment is smaller than the
smallest change your notation can express.

That is exactly what happens to a weight update in 16-bit storage. Each step
wants to nudge the weight by a hundredth of a percent, and the format cannot
represent a change smaller than about half a percent, so the nudge lands on the
same number it started from and is discarded. Not damped - discarded.

The fix is not a finer ruler for measuring; it is a wider notebook for the
running total. Keep the master copy of every weight in 32 bits, add the tiny
updates *there* where they accumulate properly, and hand a 16-bit rounding of it
to the arithmetic each step. Reading tolerates coarseness. Accumulating does
not.

## What a frontier run physically is

Stop thinking of a training run as a program that runs and start thinking of it
as a facility that operates.

Ten thousand machines. Two months. The building draws tens of megawatts and the
cooling is a civil engineering problem. There is a control room with people
watching graphs, on shifts, at three in the morning. Something breaks roughly
every three hours, all day, every day, for the entire two months - and that is
not a sign of a badly built cluster, it is arithmetic. Ten thousand of anything
means the mean time between failures of the *collection* is the mean time
between failures of one thing divided by ten thousand.

The consequence shapes everything. Because the computation is synchronous -
every machine has to finish its step before any machine can start the next - one
dead card stops all ten thousand. So the facility is built the way a chemical
plant is built: assume components fail, write down the state often enough that
losing recent work is tolerable, detect failures automatically, swap in spares
from a hot pool without a human in the loop, and resume.

The two knobs that trade against each other are simple to feel. Write down the
state more often and you spend more time writing. Write it down less often and
you lose more work when something dies. Somewhere between is a minimum, and it
sits around twenty minutes - which is why, if you ever look at a frontier
training log, you see checkpoints roughly three times an hour and restarts
several times a day, forever.

The worst failures are the ones that do not announce themselves. A card that
crashes is easy - the job stops, the automation replaces it. A card that
quietly produces slightly wrong numbers keeps the job running and poisons the
model, and you find out days later when a loss curve looks strange. Runs carry
monitoring specifically for this, watching statistics that should be smooth and
alerting when they are not.

The last thing to picture: the data. Fifteen trillion tokens is tens of
terabytes of text that has to arrive at ten thousand machines continuously, in a
specific order, without ever being the reason anyone waits. That plumbing is a
serious distributed system in its own right, and on a first attempt it is
usually the thing that turns out to be the bottleneck - not the model, not the
GPUs, the file reads.

## The law under all of it

Everything in this unit reduces to a single question you can ask about any
piece of work: **how much arithmetic do you get per byte you had to fetch?**

Compare that number to the machine's appetite. If you are below it, you are
waiting on memory and only two things help - fetch fewer bytes, or fetch them
faster. If you are above it, you are waiting on arithmetic and only two things
help - do less arithmetic, or do it faster.

Look back over the last several units with that question in hand and they
collapse into one shape. The KV cache exists so you do not fetch what you
already fetched. Quantization exists so the fetch is smaller. Mixture of experts
exists so you fetch a tenth of the model. Grouped-query attention exists so one
fetched key serves eight readers. FlashAttention exists so an intermediate never
gets fetched at all. Batching exists so one fetch serves many tokens. Six
techniques, six chapters, one idea approached from six directions: **stop
walking to the warehouse.**

That is not a summary of this unit. It is a summary of the last three, and the
reason this unit comes after them rather than before is that the pattern only
looks obvious once you have watched it happen several times without being told
what it was.
