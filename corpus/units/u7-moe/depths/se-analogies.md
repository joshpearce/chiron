## Decode is bandwidth-bound, and that is the whole motivation

**Pair: the cold full-table scan AND `grep` over a huge file.**

A `SELECT * FROM t WHERE x > 5` on a cold 19 GB table costs what it costs
because 19 GB has to come off the device. Tightening the predicate to
`x > 5 AND y < 3` changes the runtime by approximately nothing - the CPU work
per row was never the constraint. Same with `grep` over a 19 GB log: swapping a
literal string for a backtracking regex barely moves the wall clock, because
you are I/O-bound and the regex engine is idle between reads.

*The shared structure:* cost is set by **bytes moved**, not by work done per
byte. Both have an arithmetic intensity far below the machine's balance point,
so the processing unit sits idle and the transfer rate is the only term in the
performance model. Decode at batch 1 is the same shape: 3.7 FLOP/byte offered
against a machine that wants 65.

*Where this breaks:* a table scan amortizes its cost across every row it
returns - scan once, get a million rows. A decode pass reads all 19 GB and
produces **one token**. There is nothing to amortize over. That is why the fix
for a slow scan (add an index, return fewer rows) has no analogue here, and why
the only available fix is the unusual one: read less of the table per row.

The amortization gap also tells you exactly where the analogy comes back to
life - batching. Decoding 32 sequences at once reads the weights once and
produces 32 tokens, which *is* the scan-amortizes-over-rows story, and it is
why serving stacks batch aggressively and single-user laptops cannot.

---

**Pair: page cache warm-up AND JIT warm-up.**

Both are "the first one is slow, later ones are fast because something got
cached." Neither describes decode. Every token re-reads the full working set;
there is no warm state to reach. If you find yourself expecting generation to
speed up after a few tokens, you are running one of these analogies and it does
not apply.

*The shared structure with decode:* none - this pair is listed because it is
the wrong pair, and it is the one engineers reach for first.

*Where this breaks:* the KV cache (u6) is the only thing in decode that
genuinely warms up, and it caches activations, not weights. Weight traffic per
token is constant from token 1 to token 10,000.

## The router: replacing one MLP block with N of them

**Pair: consistent hashing picking $k$ replicas AND a learned L7 load balancer
picking $k$ backends.**

Consistent hashing takes a key, hashes it onto a ring, and walks clockwise to
pick the $k$ nodes that will serve it. A learned load balancer takes a request,
scores the backend pool on features of the request, and dispatches to the best
few. Both are: cheap function, applied per request, selecting a small subset
from a large pool, with the pool fully provisioned regardless.

*The shared structure:* a **$O(\text{pool size})$ scoring step that gates an
$O(\text{expensive work})$ step**, where the selection cost is negligible
against the work cost, and every member of the pool must exist and be reachable
even though almost none of them run for any given request. The router's
262k-parameter matmul against a 265M-parameter expert is exactly this ratio -
one part in a thousand.

*Where this breaks:* two ways, and both matter.

1. **Replicas are interchangeable; experts are not.** Consistent hashing works
   because any replica can serve the request - the choice is about balance, not
   about answer quality. Route a token to the wrong eight experts and you get a
   different, worse output. The selection is semantic, not administrative.
2. **The granularity is wrong by orders of magnitude.** A load balancer routes
   a *request*. The router routes a *token, at a single layer*. One chat message
   of 500 tokens through a 48-layer model makes 24,000 independent routing
   decisions. There is no request-level identity for routing to attach to.

---

**Pair: a feature flag with a percentage rollout AND `if (config.useFastPath)`.**

Both are conditional execution: code exists, gets skipped based on a runtime
check. This is the right *systems* intuition for what MoE buys - skipped code
costs no CPU.

*The shared structure:* **conditional execution of resident code**. The
untaken branch is compiled, loaded, occupying memory, and free at runtime.
That is precisely the MoE bargain: 120 of 128 experts are resident and free
this token.

*Where this breaks:* the condition. A feature flag's predicate is written by a
human and readable in the source. The router's predicate is 262k learned floats
whose behavior nobody specified and nobody can read off. And a flag is
evaluated once per request against durable config; the router's condition is
re-evaluated 24,000 times per message against the content flowing through it.
"Learned, per-token, unreadable feature flag" is roughly right, and notice how
much of the flag's usefulness - auditability, intent, stability - the qualifiers
strip away.

## Experts are not domain specialists

**Pair: an index built on a column you did not choose AND a shard key derived
from observed traffic rather than from the domain model.**

Suppose a system profiles your query load and silently builds whatever index
minimizes total latency. It works. It is measurably better than what you would
have built. And when you ask "what is it indexed on", the answer is a
seventeen-term expression over columns you never considered related, because
that expression happens to partition your actual traffic evenly. Same with an
auto-derived shard key: it balances, it performs, and it does not correspond to
`customer_id` or any other concept in your ERD.

*The shared structure:* an **effective partition chosen by an optimizer against
a measured objective, not by a designer against a conceptual model**. The
partition is real and load-bearing. Its correspondence to human categories is
coincidental, and the optimizer had no term rewarding such correspondence.
Expert routing is this: the partition minimizes next-token loss subject to a
balance constraint, and "domain" appears nowhere in either term.

*Where this breaks:* an auto-generated index or shard key is still
**inspectable**. You can print the expression, and you can point at a row and
say which shard it lands in and why. Router behavior can be measured
statistically (which is exactly how we know it is not domain-based) but the
per-token decision boundary in 2048 dimensions has no compact description.
There is no `EXPLAIN` for it. The analogy gives you the right idea about
*origin* and the wrong idea about *legibility*.

---

**Pair: microservice decomposition by bounded context AND plugin dispatch by
declared capability.**

Both are the specialist picture, and both are worth naming so you can put them
down. In each, a human enumerates the domains, writes a handler per domain, and
writes a dispatcher that classifies incoming work and forwards it.

*The shared structure with MoE:* the shape of the *call graph* only - one
dispatcher, many handlers, one handler set runs per item. That structural
resemblance is why M12 is so sticky for engineers.

*Where this breaks:* everything except the call graph.
- **The partition is designed** in the SE version and **discovered** in MoE.
- **The dispatcher is a classifier** in the SE version (it reads a type field or
  a route) and **a learned similarity score** in MoE.
- **A handler is a deployable unit** in the SE version - you can run the billing
  service alone. An expert is a fragment of one MLP block in one of 48 layers;
  running it alone is not a weaker model, it is not a model.
- **Load imbalance is a bug** in the SE version, tolerated because domains are
  what they are. In MoE, imbalance is **penalized in the objective**, which is
  exactly the force that prevents the domain-shaped partition from forming.

That last bullet is the load-bearing one. If MoE experts were bounded contexts,
the load-balancing loss would be fighting the architecture on every step.

## Total versus active: the tradeoff table

**Pair: resident set size versus per-request working set AND total heap versus
GC live-set touched per collection.**

A process may map 19 GB and touch 1.8 GB serving any given request. A heap may
be 19 GB while a minor collection scans a small nursery. In both, two budgets
move independently: what you must *hold* and what you must *touch per unit of
work*.

*The shared structure:* **capacity and traffic are separate resources with
separate hardware limits** - RAM size versus memory bandwidth - and an
architecture can trade one for the other. MoE deliberately buys traffic
reduction with capacity, which is the right trade when you have spare RAM and
scarce bandwidth. That is a laptop. It is not an H100 with 3.3 TB/s, where
bandwidth is abundant and HBM capacity is the scarce thing - and it is why the
same architecture reads as a brilliant fit locally and a more nuanced call in a
datacenter.

*Where this breaks:* **the OS can page out your untouched pages; you cannot page
out unused experts.** RSS versus VSZ works because untouched pages genuinely
live on disk and stay there. The set of untouched experts is different for
every token at every layer, so the working set churns completely 48 times per
token. Paging turns a 1.8 GB memory read into a 1.8 GB SSD read at ~5 GB/s -
about 2.8 tok/s, worse than the dense model you were trying to beat. The
capacity is not optional.

---

**Pair: read replicas for throughput AND a CDN edge cache.**

Both scale a system by making the *hot path* cheap while the full dataset lives
somewhere complete. Tempting frame for MoE: experts as replicas, active set as
the edge.

*The shared structure:* a cheap fast path over an expensive complete store.

*Where this breaks:* immediately and instructively. A CDN works because
requests have a **skewed, stable** popularity distribution - 5% of objects serve
95% of traffic, and yesterday's hot set predicts today's. Expert selection is
deliberately **flattened by the load-balancing loss** so that every expert takes
about $k/N$ of traffic. There is no hot set to cache; the optimizer spent
training effort specifically to destroy one. This is the cleanest way to see
why "just keep the popular experts in RAM" is not merely hard but structurally
foreclosed.

## What MoE costs you

**Pair: leader-election flapping AND cache-stampede / hot-shard feedback.**

An expert that gets slightly more traffic gets slightly more gradient, becomes
slightly better, and attracts more traffic. This is the same runaway you have
debugged before: a node that responds a little faster wins more elections,
warms its caches, responds faster still, and ends up owning everything until it
falls over.

*The shared structure:* **positive feedback between selection and fitness**,
with no natural damping, converging on degenerate concentration. In both cases
the fix is an explicit anti-concentration term added deliberately - jittered
backoff, randomized election timeouts, power-of-two-choices on one side; the
$\alpha N \sum_i f_i P_i$ penalty on the other.

*Where this breaks:* the fix in distributed systems is **deterministic and
always on** - jitter runs in production, at every election, forever. The MoE
balance term exists **only during training**. At inference there is no balancing
at all: the router does whatever it learned to do, and if it learned a skewed
policy you are stuck with it. You are not damping a live loop, you are shaping
a policy offline and then running it open-loop.

---

**Pair: static thread-pool sizing with a bounded queue AND a fixed-size ring
buffer that drops on overflow.**

The capacity factor is exactly this. Each expert gets a fixed slot budget per
batch, sized at $C \times$ (tokens $\times k / N$). Tokens arriving at a full
expert are dropped - the MoE block contributes nothing for them and only the
residual passes through.

*The shared structure:* **static allocation sized for the balanced case, with
drop-on-overflow as the pressure-relief valve**, chosen because dynamic sizing
would destroy the fixed-shape collective operation (here, the all-to-all;
there, lock-free access).

*Where this breaks:* a dropped message in a ring buffer is a visible, countable
error you can alert on. A dropped token in MoE training produces a
mathematically valid forward pass that is silently a little worse. There is no
exception, no log line by default - it shows up as a training curve that
plateaus for reasons that take a week to find. If you are debugging an MoE run,
instrument the drop rate first; it is the metric most likely to be silently
wrong.
