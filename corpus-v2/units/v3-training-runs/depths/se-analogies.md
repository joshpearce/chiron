## The receipt

**The pair.** A training run's cost model is like **CI minutes on a build farm**,
and like **S3 storage billing**.

**The shared structure.** All three price a resource by size times time against a
rate card, and all three are completely blind to the value of what they produce.
A build that fails costs the same minutes as one that passes. A gigabyte of
irreplaceable production data costs the same per month as a gigabyte of logs
nobody will read. A training run on a beautifully filtered corpus costs the same
as one on scrape. In each case the biller has no access to the property you care
about, so the property is free to improve - and that is exactly why the highest-
leverage work in each domain is quality work that does not appear on the invoice.

**Where this breaks.** CI minutes are consumed by retries, flakes, and queue
time, so a badly-run pipeline genuinely does cost more; training compute is
determined before the job starts and cannot be inflated by the job going badly.
And the S3 analogy breaks on the direction of the lever: storage cost is reduced
by deleting things, which loses you the data, whereas training cost is reduced by
shrinking the model, which loses you nothing about the corpus. That asymmetry is
the reason a 550-run research program at 30M parameters is a real strategy rather
than a compromise.

---

**The pair.** "Quality is free at fixed compute" is like **choosing a better
algorithm without changing the instance type**, and like **fixing a schema before
the first migration**.

**The shared structure.** In each, the improvement is available at exactly zero
marginal infrastructure cost, and is therefore a strict Pareto gain. Nobody
weighs a better index against a bigger box, because the better index is free. The
same reasoning says: never trade compute for data quality, because the trade does
not exist. Spend the effort on the classifier.

**Where this breaks.** Better algorithms take engineering time, and so does data
filtering - the largest published filtering effort cost thousands of GPU-hours to
build the classifier alone, which is compute, just not training compute. "Free"
means free in the training budget, not free in your calendar. The schema analogy
breaks harder: a schema can be migrated later at a cost, while a training run's
corpus is baked in at the moment the run starts, and the only migration is
another run.

## One step, from scratch

**The pair.** Backpropagation is like **latency attribution in a distributed
trace**, and like **an exception unwinding a call stack with an error value
attached**.

**The shared structure.** In each, a single outcome measured at one point is
distributed back over every component that contributed to it, by walking the call
graph in reverse exactly once. A trace does not re-run each span to find out
which one was slow; it records the structure on the way in and attributes on the
way out. Backprop does the same thing with the same asymptotics: forward pass
records what it needs, backward pass assigns blame to ten million parameters in a
single reverse traversal rather than ten million forward evaluations. That single
reverse pass is why training is affordable at all, and it is why the backward
pass costs about twice the forward rather than a million times it.

**Where this breaks.** A distributed trace attributes latency *exactly and
additively* - the spans sum to the total, and a span's contribution is the same
regardless of how large it is. A gradient is a local linearization: it says which
way to move each parameter for an infinitesimal improvement, and it is only valid
in a small neighbourhood. Move a parameter by a large amount in its gradient
direction and the prediction fails entirely. This distinction matters directly in
v8, where gradient-based influence methods inherit the same limitation:
"this document pushed the loss this way" is a statement about a tiny step, not
about what would happen if the document were removed. That gap between the local
derivative and the actual counterfactual is the central problem of the whole
attribution literature.

---

**The pair.** The loss is like a **checksum over an entire response**, and like
a **single scalar fitness in a genetic algorithm**.

**The shared structure.** In both, an enormous structured object is collapsed to
one number, and that one number is the only channel through which information
flows back into the system. Nothing else about the output is observable to the
optimizer. Everything the model becomes has to be squeezed through that scalar.

**Where this breaks.** A checksum is designed to be maximally uninformative about
its input - that is its purpose. The loss is the opposite: it is an average of
per-position terms, and the backward pass has access to all of them individually,
not just the average. The scalar framing is a convenience for talking about the
objective, not a description of the signal's bandwidth. That distinction is
exactly what makes per-example influence a coherent idea later.

## The one line: C = 6ND

**The pair.** $C \approx 6ND$ is like **capacity planning from QPS times cost per
request**, and like a **query planner's cost model**.

**The shared structure.** All three are closed-form estimates you compute before
writing any code, from two or three quantities you already know, and all three
are accurate enough to make go/no-go decisions with. None of them models the
system faithfully; all of them predict the number you actually need. The
discipline is the same too: you compute it first, and if the answer is absurd you
change the design rather than starting the work.

**Where this breaks.** Capacity planning has a queueing nonlinearity - past
roughly 70% utilization, latency goes superlinear and the linear model stops
predicting anything. $6ND$ has no such cliff; it is exactly linear in both
arguments over every scale in this book, which makes it a *better* tool than the
capacity formula, not a worse one. The Big-O instinct also misleads here: Big-O
deliberately drops constants, whereas $6ND$ is entirely constant - there is no
asymptotic content in it at all, only accounting.

---

**The pair.** Pairing $6ND$ with a fitted effective throughput is like a **query
planner's calibrated cost constants**, and like **story points**.

**The shared structure.** In each, a deliberately simplified model is paired with
a fudge factor calibrated against observed outcomes, and the pair predicts well
even though neither half is individually true. A planner's `random_page_cost` is
not a physical quantity; it is whatever number makes the cost model match
measured runtimes on this hardware. The 3.2 PFLOP/s figure is the same kind of
object: it is not what the GPUs do, it is what makes $6ND$ reproduce wall clocks.
The two errors cancel, and they cancel because one was fitted to cancel the
other.

**Where this breaks.** Calibrated constants are only valid in the regime they
were fitted in, and both analogies fail the same way when you leave it. Move a
Postgres cost constant from spinning disks to NVMe and the plans go wrong; move
an effective throughput from 2,048-token context to 100,000 and the budget goes
wrong, because the attention term it was silently absorbing has grown by a factor
of fifty. Story points break worse and are worth naming: they are calibrated
against a team, not a machine, and there is no measurement that validates them.
The training version is checkable against a stopwatch, which makes it a real
engineering practice rather than a ritual.

## Knobs are recipes, not theory

**The pair.** A WSD stable-phase checkpoint is like a **container base image
layer**, and like a **database snapshot you restore before each test case**.

**The shared structure.** In all three, an expensive shared prefix is computed
once and many cheap divergent suffixes are built on top of it. You do not rebuild
the base image per service; you do not re-seed the database per test; you do not
retrain the trunk per ablation. The economics are identical - $n$ variants cost
one expensive thing plus $n$ cheap things instead of $n$ expensive things - and
so is the precondition: the shared prefix has to be in a state that is legitimate
to start from. A half-applied migration is not a snapshot, and a checkpoint from
the middle of a cosine decay is not a starting state.

**Where this breaks.** The base-image analogy is the one that gets the caveat
right, so use that one. You cannot remove a package from a lower layer by editing
an upper layer - you can only shadow it, and the bytes are still in the image. In
exactly the same way, you cannot remove a source from a branched variant, because
the trunk trained on it and whatever it taught is baked into the branch point.
Every "leave-one-source-out" model built by branching still has the source in its
lower layers. The database-snapshot analogy is actively misleading here, because a
restored snapshot really is clean, and it will make you overconfident in numbers
that are biased toward zero by an unknown amount.

---

**The pair.** Reading the training loss curve is like **judging correctness from
CPU utilization graphs**, and like **treating a green build as proof the feature
works**.

**The shared structure.** Each is a real-time signal that is a genuine health
check and is routinely mistaken for a correctness check. Utilization tells you the
process is alive and doing work; it says nothing about whether the work is right.
A green build tells you nothing broke catastrophically. The loss curve tells you
the optimizer has not diverged. All three are worth watching, all three scream
usefully when something breaks, and none of them can answer the question people
ask of them.

**Where this breaks.** The green-build analogy flatters the loss curve, and the
gap is worth being precise about. A green build actually does verify the
assertions someone wrote - it is weak evidence of correctness, but it is
evidence, and you can strengthen it by writing more assertions. The training loss
curve verifies nothing about held-out behavior at any strength, because it is
computed on data the model trained on, under a schedule whose shape produces a
late drop regardless. It is not a weak test. It is a different measurement
entirely, and no amount of staring at it converts it into the one you want.

## What repeating data actually costs

**The pair.** Epoch returns are like **coverage returns from re-running a fuzzer
on the same seed corpus**, and like **the saturation curve of a compression
dictionary**.

**The shared structure.** All three describe the same shape: the second pass over
fixed input yields nearly as much as the first, the eighth yields noticeably
less, and there is a hard ceiling that no amount of additional passes exceeds
because the information in the input is finite. In each case the practical
question is identical - where on that curve am I, and is the next pass worth its
cost - and in each case the answer has been measured rather than reasoned about.
The measured answer for training data is that a corpus is worth at most about 16
times its unique token count, and that four epochs already collect 93% of what
four times the fresh tokens would have given.

**Where this breaks.** A fuzzer's plateau is directly observable while it happens
- coverage counters stop moving and you stop the campaign. The epoch plateau is
invisible during the run: the training loss keeps descending happily deep into
the dead zone, because the model keeps getting better at the text it has
memorized. You find out from held-out evaluation, after you have paid. The
compression analogy breaks in the other direction: a dictionary's saturation is a
deterministic property of the input, whereas the 16x ceiling is a fitted
empirical constant on one set of experiments, and you should hold it as a strong
prior rather than a law.

---

**The pair.** Turning up the epoch count is like **increasing cache TTL**, and
like **raising a log retention window**.

**The shared structure.** One knob, two consequences that point in opposite
directions, and the second consequence is usually discovered by someone else.
Longer TTL means better hit rates and staler data. Longer retention means better
debugging and a bigger compliance surface. More epochs means better use of a
small corpus and more of that corpus becoming extractable from the finished
model. In every case the correct move is to set the knob deliberately, write down
why, and know who else the setting affects.

**Where this breaks.** TTL and retention are reversible - flush the cache, purge
the logs. Epochs are not. Once a model has been trained with high repetition, the
memorization is in the weights, and there is no operation that removes it short of
retraining. This is why v6 treats the epoch count as an evidentiary decision made
at training time rather than a tunable, and why it belongs in the provenance
manifest alongside the corpus composition.

## The laptop and the node

**The pair.** The laptop-versus-node choice is like **unit tests versus
integration tests**, and like **local development versus a staging cluster**.

**The shared structure.** In each, the small environment is not a degraded copy
of the big one - it is the correct tool for a different question, and the
difference is iteration count. You do not run the integration suite in a tight
loop; you run the unit suite, thousands of times, because the question there is
"which change broke it" and answering it requires many cheap trials rather than
one expensive one. The attribution question has exactly that shape: "which source
mattered" is answered by 21 to 550 trials, so the machine that gives you 24 runs
a night beats the machine that gives you one, by a wide margin.

**Where this breaks.** Local development is *supposed* to be behaviorally
identical to production, and when it is not, that is a bug someone should fix. A
30M-parameter model is genuinely not a small version of a 7B model, and the
difference is not a bug. Influence patterns are known to change with scale -
crisp and top-heavy at small scale, long-tailed at frontier scale where per-
document payouts round to zero. So the small-scale ground truth you build is
authoritative about small-scale models and is a hypothesis about large ones. State
it that way. It is the honest version of the claim and it is still a strong claim,
because nobody else has the ground truth at any scale.

---

**The pair.** MFU is like **cache hit rate**, and like **effective throughput
versus link speed**.

**The shared structure.** All three are ratios of achieved to theoretical, all
three are determined by engineering rather than by the workload's purpose, and all
three multiply straight into cost. A 50% hit rate doubles your backend load; 40%
MFU means renting the machine 2.5 times as long as the spec sheet suggests. In
each case the number is quoted alongside vanity metrics and is routinely
dismissed as one, and in each case dismissing it is a direct financial error.

**Where this breaks.** Cache hit rates have a meaningful ceiling of 1 that good
systems approach, so "we're at 40%" reads as failure. MFU does not work that way:
50% is near the practical ceiling for transformer training on current hardware,
40% is a good run, and the missing 60% is structural - memory movement, inter-GPU
communication, and the attention work the metric's own numerator declines to
count. Comparing MFU to 100% is a category error. Compare it to other people's
runs on the same hardware.

## Evaluation without self-deception

**The pair.** Bits per byte is like **quoting compression ratio as output bytes
per input byte**, and like **normalizing latency per request rather than per
batch**.

**The shared structure.** Each fixes the same class of bug: a metric whose
denominator is chosen by the system under test. If you measure a compressor in
"bits per block" and let it pick the block size, you have measured its block size.
If you report latency per batch and let the service pick the batch, you have
measured batching. If you report loss per token and let the model pick the
tokenizer, you have measured the tokenizer. The fix in all three is to normalize
by something fixed before the system existed - input bytes, individual requests,
UTF-8 bytes of the held-out file.

**Where this breaks.** A compression ratio is exact and reproducible; bits per
byte is an upper bound on the true code length, because the model spreads a
little mass over alternative tokenizations of the same string that the measurement
does not marginalize over. The bound is tight and identically signed for every
model, so orderings are safe, but do not describe bpb as "the entropy of the text
under the model" in anything a reviewer will read. The latency analogy breaks in
that per-request latency is directly meaningful to a user, whereas bits per byte
is meaningful only comparatively - 0.69 bpb is not good or bad, it is better than
0.72.

---

**The pair.** Seed noise is like **flaky tests**, and like **benchmarking on a
box with noisy neighbours**.

**The shared structure.** In all three, a real signal is buried in run-to-run
variance, and the only valid response is repetition plus a stated variance rather
than a better single measurement. Anyone who has chased a performance regression
on shared infrastructure already has the right instincts: run it ten times, report
the median and the spread, and be deeply suspicious of a single number that
happens to support the conclusion you wanted.

**Where this breaks.** And this is the most important break in the file. A flaky
test has a root cause, the root cause is a defect, and the correct engineering
response is to find and eliminate it. Seed noise has no defect. It is a genuine
property of the training process - different initialization, different shuffle,
different final model, all of them correct - and it does not go away with better
code, a longer run, or a bigger model. Every engineering instinct you have says
"this variance is a bug, fix it." Here the variance is the phenomenon. The
response is not to eliminate it but to measure it, report it, and average over it.
A learner who tries to make training deterministic to escape this has
misunderstood what is being measured: contribution is a random variable, and
pinning the seed does not make it a constant - it just hides the distribution
behind one arbitrary draw from it.

## At the bench: the overnight run

**The pair.** Measuring seed noise before any attribution work is like
**recording a performance baseline before optimizing**, and like **running a load
test against an unchanged system to characterize the harness**.

**The shared structure.** In each, you spend real time producing a number that
answers no question anyone asked, because every subsequent claim is a comparison
against it. Optimizing without a baseline produces "it feels faster." Load testing
without a control run produces numbers that might be your harness. Attribution
without a measured noise floor produces a table of source values with no way to
tell which entries are signal. In all three the baseline is the least interesting
artifact and the most load-bearing one.

**Where this breaks.** A performance baseline is a single number you can beat. A
noise baseline is a *variance*, and it is used differently - it does not go in
the numerator of anything, it goes in the denominator of every comparison. The
load-test analogy is closer but breaks on reusability: you re-run a control load
test whenever the environment changes, whereas seed noise is a property of the
model configuration and can be pooled across many conditions and reused all the
way through v8, provided the configuration does not change. Change the model size
or the token count and you have a new $\sigma$, and every threshold downstream
moves with it.

## What you can now do

**The pair.** The three habits of this unit are the discipline of a **benchmark
harness you would actually trust**, and of a **postmortem with a stated
confidence level**.

**The shared structure.** Pre-register the metric before you look at the data.
Repeat every condition. Report the spread alongside the point estimate. State
what would have changed your conclusion. Anyone who has watched a team ship a
regression because someone benchmarked once on a laptop already believes all four
of these; the only new thing in this unit is that the same discipline applies to
training runs, where the temptation to skip it is stronger because each trial
costs money rather than milliseconds.

**Where this breaks.** A benchmark harness can usually be made deterministic
enough that a single run means something - pin the CPU, disable turbo, isolate
the core. There is no equivalent move here, and pinning the seed is the trap that
looks like one: it makes the number reproducible without making it true, because
what you want to know is how the result varies across the distribution of training
runs, not what one draw from that distribution happened to be. The postmortem
analogy also flatters the situation: a postmortem's confidence is about an
explanation of something that already happened, whereas these error bars are
predictions about what a rerun would show, and they are checkable. That is
better, not worse - it means every claim you make in the rest of this book can be
falsified by someone with $50 and a night.
