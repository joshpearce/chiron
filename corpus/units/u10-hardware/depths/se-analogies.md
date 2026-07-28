## The machine you are actually running on

**Pair: L1/L2/L3/DRAM on a CPU AND the CDN / origin / cold-storage tiers of a
web system.**

Both are hierarchies where each level outward is roughly an order of magnitude
larger, an order of magnitude slower, and an order of magnitude cheaper per
byte, and in both the entire performance engineering discipline is "keep the
working set at a level where the latency is acceptable." An accelerator's
registers, SRAM, HBM, and cross-node network are the same ladder with the rungs
relabelled.

*The shared structure:* **capacity and speed trade against each other
monotonically across levels**, so the design question is never "how do I make
the slow level fast" but "how do I arrange the access pattern so the slow level
is touched rarely."

*Where this breaks:* CPU caches are **implicit and hardware-managed** - you
influence them by access pattern and hope. An accelerator's SRAM is
**explicit and software-managed**: the shared memory in a CUDA kernel is
allocated, filled, and freed by code you write, with the tile sizes chosen by
you. There is no cache line, no eviction policy, no prefetcher deciding on your
behalf. That is why FlashAttention exists as an *algorithm* rather than as a
compiler optimization - it requires a human to choose the blocking, and the
choice is hardware-specific enough that FlashAttention-2 and -3 are separate
retunings for newer chips.

The second break is temporal: a CPU cache assumes reuse over time (fetch once,
hit repeatedly over the next microseconds). Decode has zero temporal reuse of
weights - every weight is touched once per token and the next token starts over.
There is no working set to keep hot, which is why the entire cache intuition
gives you the wrong prediction here.

---

**Pair: `SELECT` over a covering index AND a vectorized columnar scan.**

Both are cases where the win came from arranging data so that one fetch feeds
many operations, rather than from making the operations faster. A covering index
means the row never gets touched; a columnar layout means one cache line
delivers 64 values that all need the same operation.

*The shared structure:* **the optimization is in the data layout, not the
computation**. A tensor core is the same move made in silicon - one operand
fetch feeding a whole tile of multiply-accumulates instead of one.

*Where this breaks:* a covering index is a *choice among equivalent plans*; the
database could always have done it the other way, more slowly. A tensor core is
not a plan, it is a fixed-shape jig. Work that does not fit the shape does not
get done slowly - it gets done on entirely different, 15x slower hardware paths.
There is no gradual degradation, which is why "express it as a matmul" is a
binary architectural constraint on the whole field rather than a performance
tip.

## Arithmetic intensity and the roofline

**Pair: Amdahl's law AND the USE method (utilization, saturation, errors).**

Amdahl gives you a bound before you optimize, telling you the maximum possible
payoff so you do not spend a week on a 3% path. USE tells you which resource to
look at first by checking each one's utilization and saturation rather than
guessing. The roofline is both at once: it bounds the achievable rate *and* it
names which of two resources is binding.

*The shared structure:* **a cheap upfront calculation that redirects effort
before any code is written**. All three answer "is this worth doing" and "what
should I attack" rather than "how do I make this faster," and all three are
routinely skipped by people who then optimize the wrong thing.

*Where this breaks:* Amdahl and USE are diagnostic; the roofline is
**prescriptive with only two possible prescriptions**. Below the ridge: reduce
bytes. Above it: reduce FLOPs. There is no third recommendation and no
judgement involved once the number is computed, which is unusual and worth
exploiting. Also unlike a profiler, the roofline needs no run - you compute it
from the model config and the spec sheet before the hardware arrives, which is
how capacity planning for a training cluster actually works.

---

**Pair: an I/O-bound service where adding CPU does nothing AND a
`git status` on a cold NFS mount.**

You have profiled a service, seen 4% CPU, and known immediately that a larger
instance type is not the fix. You have watched `git status` take forty seconds
over a network filesystem and known the CPU is not the problem. In both cases
the bound is set by the transfer and the processor is a spectator.

*The shared structure:* **the resource that is 96% idle is not the resource to
buy more of**, and utilization of the wrong resource is a distraction rather
than a clue. Batch-1 decode is the same picture: 5.6% of the GPU's arithmetic
in use, and the answer to "should I buy the faster card" is a question about
which column of the spec sheet, not about how much money.

*Where this breaks:* an I/O-bound service usually has a *fix on the I/O side* -
add caching, batch the requests, denormalize. Decode's I/O has no such fix,
because there is nothing to cache: every weight is genuinely needed exactly
once per token, and it is a different set of bytes each layer. The only
remaining moves are to make the bytes fewer (quantize, MoE) or to make the
fetch serve more work (batch), and the second one is unavailable when you are
one person. That is a narrower option set than any I/O-bound service you have
debugged.

## FlashAttention: changing the bytes, not the FLOPs

**Pair: streaming an aggregation instead of materializing a temp table AND
composing iterators instead of building intermediate lists.**

`SELECT sum(x) FROM (SELECT ... ) t` where the planner materializes `t` to disk,
versus a plan that streams it. Or `[f(x) for x in xs]` then `[g(y) for y in ys]`
versus a generator pipeline that never builds the middle list. Both cases: the
intermediate was never wanted, only the final result was, and materializing it
cost more than computing it.

*The shared structure:* **an intermediate result larger than both its inputs and
its output, created only because the naive decomposition of the computation
names it.** The n x n attention score matrix at n = 8192 is 128 MiB against 2 MiB
inputs and a 2 MiB output. Deleting the materialization is the whole
optimization, in all three cases.

*Where this breaks:* streaming an aggregation is usually **free or nearly so** -
the planner does it, or you change three lines. FlashAttention's streaming is
blocked by a genuine mathematical obstacle: softmax needs a denominator over the
whole row, so the naive stream cannot start emitting until it has seen
everything. Solving that took the online-softmax rescaling, which is real work
and was published as a result. When an intermediate resists streaming, look for
a global operator in the middle - that is almost always what is stopping you,
and it is almost always a normalization.

The second break: iterator composition trades memory for the *same* runtime.
FlashAttention trades a fraction of a percent more arithmetic (the rescaling
multiplies) for an order of magnitude less traffic, which makes it a strictly
better algorithm rather than a memory-versus-speed choice. Those are rare and
worth recognizing.

---

**Pair: `SELECT *` when you needed two columns AND N+1 queries in an ORM.**

Both are "the code is correct and the traffic is absurd." Both are invisible in
a code review and obvious in a trace. Both were introduced because the natural
way to express the computation happens to name data you do not need.

*The shared structure:* **correctness and traffic are independent axes**, and
the language you write the computation in has no opinion about the second one.
`softmax(Q @ K.T) @ V` in PyTorch is a correct, readable, one-line statement of
attention that moves 65x more bytes than necessary. Nothing about the notation
warns you.

*Where this breaks:* the ORM fix is to write different application code, and the
`SELECT *` fix is to name your columns - both are edits at the level you were
already working at. The attention fix required dropping to CUDA and hand-tiling
against a specific chip's SRAM budget, and then required doing it again for the
next chip. Traffic problems in this field are frequently not fixable in the
language the model is written in, which is why "just write a fused kernel" is a
project rather than a code review comment.

## Why training and inference want different iron

**Pair: OLTP versus OLAP AND a latency-SLO service versus a nightly batch job.**

Same data, same schema, same SQL dialect, two systems that share almost no
design decisions. Postgres tuned for point lookups and a columnar warehouse
tuned for full scans are not competitors; they are answers to different
questions. Likewise a service with a p99 budget of 50 ms and a batch job that
processes the day's events overnight - the second one would be insane to build
on the first one's infrastructure and vice versa.

*The shared structure:* **the shape of the request, not the shape of the data,
determines the architecture.** Training is a scan: enormous, throughput-only,
nobody waiting. Single-user decode is a point lookup: tiny, latency-critical,
one at a time. The tensors are identical and the two workloads sit on opposite
sides of the roofline ridge, which is exactly the OLTP/OLAP split expressed in
FLOPs per byte.

*Where this breaks:* OLTP and OLAP differ in their *access patterns over
storage*, and you can bridge them with an HTAP system that accepts a compromise
on both. Training and inference differ in **batch size, which is a property of
the demand and not of anything you control**. There is no compromise setting: if
one user is asking one question, the batch is 1, and no architecture makes it
larger. The bridge that does exist - batching across users - is available only
to whoever has many users, which is a business fact rather than a technical one.

---

**Pair: a build server versus a production replica AND a CI runner versus an
edge node.**

Both pairs run the same code and are provisioned completely differently, and in
both cases the difference is dominated by working-set size rather than by CPU.
A build server needs disk and RAM for intermediate artifacts; the thing that
serves the built artifact needs almost none of that.

*The shared structure:* **the process that produces an artifact has a working
set many times larger than the process that uses it.** Training holds weights,
gradients, an fp32 master copy, two optimizer moments, and saved activations -
16 bytes per parameter against inference's 2, or 0.54 at 4 bits.

*Where this breaks:* a build server's extra footprint is a *convenience* - you
could build with less by spilling to disk and accepting a slower build. The
training footprint is not optional in the same way: gradients and moments are
mathematically required by the algorithm at every step, and they are touched
every step, so spilling them is not slow but pathological. The closest true
analogue is not a build server but a database that must hold its entire index in
RAM to serve at all - and even then, the 1.12 TB for a 70B model exceeds any
single machine, which no build server has ever managed to do.

## Four ways to split a model, and what each one stresses

**Pair: sharding a database by key AND partitioning a Kafka topic.**

Both are the axis-choice problem. You pick what to split on, and that single
choice determines which operations are local and cheap and which become
distributed and expensive. Pick the wrong shard key and every query becomes a
scatter-gather; pick the right one and most queries touch one partition.

*The shared structure:* **the split axis determines the communication pattern,
and the communication pattern determines whether the system works at all.** Data
parallelism, tensor parallelism, pipeline parallelism and expert parallelism are
four shard keys over the same computation, and each produces a different
collective at a different frequency: all-reduce once per step, all-reduce twice
per layer, point-to-point per stage boundary, all-to-all twice per MoE layer.

*Where this breaks:* in a sharded database the axes are **mutually exclusive** -
you shard by customer or by region, not both, and choosing means giving
something up. Model parallelism composes: a real run uses tensor x pipeline x
data x expert simultaneously, nested, with each axis assigned to the
interconnect tier that can carry its collective. There is no equivalent in a
database of "shard by customer inside the rack and by region across racks and by
time across regions, all at once, on the same rows." The composition is what
makes the configuration search hard, and it is why frontier-run configuration is
a specialist skill rather than a decision.

---

**Pair: `Promise.all` fan-out with a slow member AND a distributed transaction
with a straggler.**

A synchronous training step is a barrier: every device must finish before any
device proceeds. You have debugged this exact shape. One slow member sets the
latency of the whole fan-out; one straggler holds the transaction open and
everyone waits.

*The shared structure:* **the collective's cost is the maximum over
participants, not the mean**, which makes tail latency and load imbalance far
more damaging than they are in independent work. This is why expert parallelism
uses fixed per-expert capacity with drop-on-overflow: variable-sized all-to-all
messages mean the slowest pair sets the layer's time, every layer, forever.

*Where this breaks:* your fan-out usually has a **timeout and a degraded
path** - drop the slow member, serve stale data, return partial results. A
training step has none. There is no meaningful "gradient without device 4,183";
the step is all-or-nothing, and the only degraded mode is to stop, evict the bad
node, and restart from a checkpoint. The absence of a partial-success path is
the single biggest difference between operating a training cluster and operating
a distributed service, and it is why the operational answer is checkpoint
frequency rather than circuit breakers.

---

**Pair: a work-stealing thread pool's warm-up AND a CI pipeline's critical
path.**

Pipeline parallelism's bubble is the fill-and-drain of a pipeline, and you have
measured it: a CI pipeline with eight sequential stages has an end-to-end
latency of eight stages regardless of how many builds you push, and the way you
get throughput back is to keep many builds in flight.

*The shared structure:* **the fix for pipeline idleness is more items in flight,
not faster stages.** Bubble fraction (p-1)/(m+p-1) goes from 47% at eight
microbatches to 9.9% at sixty-four, with no hardware change at all.

*Where this breaks:* CI stages are independent, so more builds in flight costs
nothing but scheduling. Microbatches in flight cost **activation memory**, which
is the resource that was already scarce enough to force gradient checkpointing.
The two knobs fight: more microbatches shrink the bubble and grow the memory,
and the run is configured at whatever point the memory runs out. There is no
equivalent tension in CI, which is why "just increase parallelism" is free
advice there and a memory-budget negotiation here.

## Precision: range is the scarce resource, not resolution

**Pair: integer overflow versus float rounding AND timestamp precision versus
timestamp range.**

Every engineer has met both failure modes and knows they are different in kind.
An overflow is a value that becomes wrong catastrophically and discontinuously.
Rounding is a value that stays approximately right. Similarly, storing a
timestamp with millisecond resolution loses sub-millisecond ordering; storing it
in a 32-bit epoch loses the year 2038 entirely. One is imprecision, one is
absence.

*The shared structure:* **range failures are categorical and resolution failures
are proportional**, and systems tolerate the second far better than the first.
fp16's problem with a 1e-8 gradient is not that it stores it coarsely - it
stores it as **exactly zero**, which is a 2038 problem, not a rounding problem.
bf16 spends three bits buying range at the cost of resolution and wins, because
the network absorbs proportional error and cannot absorb missing values.

*Where this breaks:* integer overflow usually **announces itself** - a wrapped
counter produces obviously wrong output, a 2038 bug throws. Gradient underflow
to zero is silent and looks exactly like a converged parameter. The run does not
crash; it trains slightly worse, and you find out from a benchmark weeks later.
This is why fp16 training needed an explicit overflow-detection loop (the
dynamic loss scaler) as a supervisory mechanism rather than relying on the
failure being visible.

---

**Pair: accumulating floats in a hot loop AND summing a large array of doubles
without Kahan compensation.**

You already know that `total += small` loses the increment once `total` is large
enough, and you know the fixes: sum in a wider type, sort ascending, or use
compensated summation. This is exactly, mechanically, why mixed-precision
training keeps fp32 master weights.

*The shared structure:* **accumulation of small increments into a large running
value is where narrow types fail, and the fix is to widen the accumulator rather
than the operands.** A bf16 weight of 1.0 has representable neighbours 0.0039
apart; a weight update of relative size 1e-4 rounds straight back to 1.0 and is
discarded, every step, forever. Tensor cores make the same choice one level
down: bf16 inputs, fp32 accumulator, always.

*Where this breaks:* a numerical-summation bug produces a **wrong number you can
check** against a reference sum. A stalled weight update produces a model that
trains, converges, and is quietly worse - there is no reference to compare
against, because the correct final weights are not known. The failure is
invisible by construction, which is why the practice (fp32 masters, fp32
accumulate) is universal and unconditional rather than something you reach for
after observing a problem.

## What a frontier run physically is

**Pair: running a fleet where instance failure is routine AND chaos engineering
as a design assumption.**

You do not treat an EC2 instance disappearing as an incident; you treat it as
Tuesday, and the architecture assumes it. A frontier training run at 16,384
accelerators sees 419 unexpected interruptions in 54 days - one every three
hours - and the response is identical in spirit: automate detection, keep hot
spares, make replacement a non-event.

*The shared structure:* **at sufficient component count, individual component
failure stops being an exception and becomes a rate**, and the correct response
is to design for the rate rather than to chase reliability. Both disciplines
converge on the same answers: automated detection, no human in the loop, fast
replacement, and state written down often enough that losing recent work is
cheap.

*Where this breaks:* your fleet is **stateless and independent**, so one dead
instance removes 1/N of capacity and the other N-1 keep serving. A training step
is a synchronous barrier across every device: one dead accelerator stops all
16,384, immediately. There is no graceful degradation, no partial service, no
load shedding. The only response is to stop the world, replace, and resume from a
checkpoint - which is why the operational metric is checkpoint interval rather
than error budget, and why the optimum sits near twenty minutes rather than near
zero.

---

**Pair: WAL / periodic snapshots in a database AND a game's autosave interval.**

Both trade write cost against how much work is lost on a crash, and both have an
optimum that people set by feel and that has a closed form. The training-run
version is Young's formula: optimal interval = sqrt(2 x checkpoint cost /
failure rate), which for a one-minute write and a three-hour MTBF gives about
nineteen minutes and about 10% overhead.

*The shared structure:* **the expected-loss-versus-write-cost tradeoff has a
square-root optimum**, so the interval is remarkably insensitive to the inputs -
halve the failure rate and the optimal interval grows only 41%.

*Where this breaks:* a database WAL is **incremental** - you append the delta and
recovery replays it, so the write is small and continuous. A training checkpoint
is a **full dump of a terabyte of optimizer state**, because there is no
meaningful delta: every one of the 70 billion parameters and both of its
optimizer moments changed this step. Nothing about WAL's incrementality
transfers, and that is precisely why the checkpoint interval is minutes rather
than the sub-second durability you are used to.

## The law under all of it

**Pair: Little's law AND the CAP theorem.**

Both are one-line results that constrain an entire design space, and both are
worth memorizing because they let you reject proposals without building
anything. Little's law tells you throughput, latency, and concurrency are not
three independent knobs. CAP tells you which of three properties you are giving
up whether you meant to or not. The roofline tells you that attainable
throughput is min(peak, intensity x bandwidth) and that only one of the two
terms is ever live.

*The shared structure:* **a bound that turns an open-ended optimization
conversation into a bounded one**, and whose main value is in the proposals it
eliminates rather than the ones it produces. "Buy the card with more TFLOPs"
dies at the roofline the same way "we will have consistency and availability
under partition" dies at CAP.

*Where this breaks:* Little's law and CAP are **exact theorems** about their
models. The roofline is a bound with an optimistic assumption baked in - it
assumes compute and transfer overlap perfectly, and real kernels miss the roof
for reasons the model does not represent (occupancy, uncoalesced access,
achievable-versus-peak bandwidth). A kernel at 60% of its roofline bound is
doing well, and treating the roofline as a prediction rather than as a ceiling
will make you chase the last 40% forever. Use it to pick the target, then
profile.
