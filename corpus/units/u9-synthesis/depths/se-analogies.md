---
unit: u9
depth: se-analogies
---

## The trace, and how to read it

**The trace is like reading a flame graph AND like walking a request through a
distributed trace (Jaeger/OpenTelemetry spans).**

The shared structure: in both, you already know what every component does in
isolation, and the artifact's entire value is showing you *where the time
actually went*, which is almost never where you assumed. A flame graph is
useless if you cannot name the functions; it is invaluable once you can, because
it reorders your priors about cost. Same here - you know attention, you know
sampling, and the trace tells you that one of them is 16% of a phase you thought
it dominated.

**Where this breaks:** a distributed trace has real spans with real queueing
between services, and you can optimize a span in isolation. The LLM trace has no
queues and almost no isolable spans - prefill and decode are the *same code*
running at different batch sizes, and the difference in cost comes from hardware
regime, not from the code. If you go looking for a slow function to fix, you will
not find one. The thing to optimize is a ratio, not a span.

## Stage 1: the harness builds one flat token stream

**Prompt assembly is like string-concatenating a SQL query AND like serializing
a struct to a wire format that has no type tags.**

The shared structure: in both, a richly typed, validated, structured object on
the sending side collapses into a flat sequence on the receiving side, and the
receiver reconstructs structure only by convention. Your ORM knows the difference
between a column name and a user-supplied string; the database receives one
string and infers roles from syntax. Your harness knows the difference between a
system message and user content; the model receives one token sequence and infers
roles from delimiter tokens.

That is also why the failure modes rhyme exactly. SQL injection and prompt
injection are the same bug: untrusted data placed into a channel where structure
is signaled in-band. `'; DROP TABLE` and `\n\nAssistant: Certainly` are the same
move.

**Where this breaks - and this is the important part:** SQL injection has a
complete fix. Parameterized queries move the structure out of band, so the
database is told "this is data" by the protocol and no content can change that.
There is no parameterized-query equivalent for LLMs. The delimiters are learned
tokens in the same vocabulary as everything else, and the "parser" is a
statistical model with no hard guarantee. Every defense available today - special
tokens the tokenizer refuses to emit from user text, instruction-hierarchy
training, output filtering - is mitigation, not elimination. Do not carry over
the reflex "we solved this in 2005."

## Stage 2: bytes to token IDs to vectors

**The embedding table is like an interned string pool AND like a perfect hash
into a fixed-size array of feature vectors.**

The shared structure: a variable-length piece of text is replaced by a fixed-size
handle, once, at the boundary, and everything downstream operates on handles.
Interning gives you cheap equality comparison; embedding lookup gives you cheap
arithmetic. Both make the rest of the system's cost independent of the original
string's length.

**Where this breaks:** interned handles are *opaque and arbitrary* - handle 4001
has no relationship to handle 4002, and comparing them is meaningless. Embedding
rows are the opposite: the entire point is that the geometric relationship
between rows carries the signal, and the arithmetic on them (dot products,
differences) is the computation. If you carry the interning intuition, you will
expect the model to treat two token IDs as unrelated atoms and be baffled by
subword generalization. Also, unlike a string pool, the table is *learned*, so
identical text tokenized under a different tokenizer produces a genuinely
different program input - which is why you cannot mix tokenizers between models
and why token counts differ across providers for the same text.

## Stage 3: prefill - 8,192 tokens, one forward pass

**Prefill is like a vectorized SIMD pass over an array AND like a columnar
analytics scan (ClickHouse/DuckDB) over a batch of rows.**

The shared structure: the loop you would naively write per element is replaced by
one pass that processes the entire batch through a fixed sequence of operations,
because the operations are identical for every element and the hardware is built
for throughput on wide data. In both cases the win comes from amortizing
instruction and memory overhead over the batch, not from doing less arithmetic.
And in both cases the naive mental model ("it loops over rows") gives wildly wrong
performance predictions.

**Where this breaks:** a columnar scan is embarrassingly parallel across rows -
row 7,000 genuinely does not care about row 3. Prefill positions *are* coupled:
position 7,000 must consume position 3's keys and values. It is parallel not
because the elements are independent but because the dependency is expressible as
a matrix multiply against a triangular mask, which the hardware executes in one
shot. So the analogy gets the throughput right and the reason wrong, and the
reason is what predicts the $n^2$ term. A columnar scan is linear in rows.
Prefill is not.

## Stage 4: inside one block - residual stream, heads, and the router

**The residual stream is like a shared mutable buffer that each stage
read-modify-writes AND like an event-sourced aggregate where each handler emits a
delta rather than a new snapshot.**

The shared structure: state is never replaced, only accumulated, so any stage's
contribution remains individually identifiable in the final result, and removing
one handler degrades the result rather than corrupting it. This is why you can
delete a transformer block and still get coherent output, and why the "logit
lens" works - you can literally read out what each block contributed, the same
way you can replay an event log and see which event moved a balance.

**Where this breaks:** event sourcing has ordered, semantically typed events, and
replaying a subset in a different order is either well-defined or an error.
Residual contributions are unordered vectors in a shared 4096-dimensional space
with no type tags at all, and different blocks routinely write to overlapping
directions - the space is used in superposition, with more "features" than
dimensions. There is no schema, no reserved region, no ownership. Do not expect
to find "the block that stores dates" writing to "the date field."

---

**MoE routing is like consistent-hash sharding to 128 backends AND like a JIT
choosing among specialized code paths at each call site.**

The shared structure: a cheap per-item decision function picks a small subset of
available machinery, so total capacity greatly exceeds per-request cost. Your
sharded cache has 128 nodes but a `GET` touches one; your MoE has 128 experts but
a token touches eight. Both need the routing to be balanced or the tail latency
of the whole system is set by the hottest shard, and both therefore have explicit
balancing machinery (virtual nodes; the load-balancing auxiliary loss).

**Where this breaks - two ways, both load-bearing:** first, a consistent hash is
*deterministic on the key and semantically meaningless*, while a JIT's dispatch
is *semantically meaningful and interpretable*. MoE routing is the awkward middle
and neither analogy alone gets you there: it is learned (so not arbitrary) but
token-granular and mostly uninterpretable (so not a type dispatch). Anyone
reasoning from the JIT half concludes "experts are specialists"; anyone reasoning
from the hash half concludes "routing is topic-independent noise." Both are
wrong; you need the pair. Second, a sharded backend only needs the shard you hit
to be *running*; an MoE needs all 128 experts *resident in memory* even though it
uses 8, because the next token routes elsewhere. Sharding saves you memory. MoE
does not - it saves you bandwidth.

## Stage 5: the last position becomes a distribution

**Sampling is like a weighted random load balancer AND like `ORDER BY score
LIMIT 1` with a randomized tiebreak.**

The shared structure: a fixed vector of scores, computed by machinery that has
already finished running, is converted into a selection. The selection policy is
a separate, swappable component that cannot change the scores. You can switch
your load balancer from weighted-random to least-connections without redeploying
the backends; you can switch from `T=1.0` to greedy without changing a weight.

**Where this breaks:** a load balancer's weights are static configuration, and
`ORDER BY` operates on rows that exist. The logit vector is recomputed from
scratch at every single token, conditioned on everything generated so far - so
the "configuration" changes 400 times a turn, and each change was caused by the
previous sampling decision. That feedback loop is what the analogies completely
omit, and it is the reason a single unlucky sample at token 12 can send the whole
generation somewhere else. Temperature is not a stable policy applied to a stable
distribution; it is a policy applied to a distribution your earlier draws helped
produce.

## Stage 6: decode, and what the KV cache actually bought

**The KV cache is like a memoization table on a pure function AND like a
materialized view that is refreshed by append only.**

The shared structure: it stores a deterministic function of inputs you already
possess, so it can be dropped and rebuilt at any time with identical results, and
its only purpose is to avoid recomputation. Evicting a memo table costs latency,
never correctness. Same here, exactly.

**Where this breaks:** a memo table is keyed and randomly accessible - you look up
one entry. The KV cache is read in full, every single decode step, for every
layer. It behaves less like a hash map and more like a file you `mmap` and scan
end to end 400 times per turn. That is why its *size* is a latency term and not
just a memory-footprint term, which is the single most common surprise for
engineers who file it mentally under "cache." A materialized view that grew to
25 GB would cost you disk; a KV cache that grows to 25 GB costs you 12 ms on
every token you generate.

---

**Speculative decoding is like branch prediction with speculative execution AND
like optimistic concurrency control with a validate-then-commit step.**

The shared structure: do work before you know it is needed, then check cheaply
and discard on mismatch. The economics only work when the speculative work is
nearly free relative to the checked work, and when the hit rate is high.

**Where this breaks:** a mispredicted branch costs you a pipeline flush - real,
wasted cycles. A rejected speculative token costs you essentially nothing on the
target model, because the verification pass would have happened anyway and
checking five tokens costs the same as checking one in a memory-bound regime. The
only waste is the draft model's time. So the risk profile is inverted from CPU
speculation: bad prediction rates degrade you toward baseline rather than below
it. Also note the guarantee is stronger than OCC's - with the standard rejection
sampling scheme the output distribution is provably identical to non-speculative
sampling, which is not something OCC promises about transaction interleavings.

## Stage 7: the tool call, the break, and the resume

**The tool loop is like a coroutine yielding to its scheduler AND like a CPU
trapping to the kernel on a syscall.**

The shared structure: execution reaches a boundary it cannot cross, packages a
request in a fixed calling convention, transfers control entirely to a different
component with different privileges, and resumes with the result placed where it
can be read. The model cannot open a file for the same categorical reason
userspace cannot touch a disk controller: not lack of skill, lack of mechanism.

**Where this breaks - and this is the difference that costs money:** a syscall
returns into the *same* process with all its state intact - registers, stack,
heap. The model has no state to preserve. It "resumes" only in the sense that a
fresh forward pass runs over a longer token sequence that happens to include the
result. There is no stack, no local variable that survived, nothing carried
across the boundary except tokens. Everything the model "was thinking" before the
tool call exists only insofar as it was written into the visible text. That is
why reasoning traces and scratchpads are load-bearing rather than decorative, and
why a syscall costs microseconds while a tool call costs a partial prefill.

---

**Prefix caching is like Docker layer caching AND like a Merkle tree / content
addressed store.**

The shared structure: validity is determined by an unbroken chain from the
beginning. Change line 3 of your Dockerfile and every layer after it rebuilds,
regardless of whether those later layers "care" about line 3. Change token 40 of
your prompt and every cached KV entry after it is invalid, for exactly the same
structural reason. And the engineering advice is identical in both domains: put
the stable stuff first, the volatile stuff last, and never interleave.

**Where this breaks:** Docker's cache key is a content hash, so reverting a change
restores the old cache entries - the cache is content-addressed and history-free.
The KV cache is positional and typically held in device memory for the life of a
session; reverting an edit does not resurrect the evicted entries, and providers
expire prefixes on a timer measured in minutes. Also, Docker layers are
coarse-grained and you control the boundaries; prefix cache boundaries are set by
the provider's block size (often 128 or 256 tokens), so a one-token change at
position 40 invalidates from the enclosing block, not from token 40.

## The cost model that falls out of the trace

**The growth of an agentic session is like an N+1 query problem AND like an
accidentally quadratic string builder (concatenating in a loop instead of using a
buffer).**

The shared structure: no single operation is wrong, each one is individually
cheap and correct, and the cost comes entirely from the *shape of the loop*. You
cannot find the bug by profiling one call - it looks fine. You find it by noticing
that the number of operations and the cost per operation both grow with the same
variable. And in both cases, the fix is never "make the operation faster," it is
"change the loop."

**Where this breaks:** an N+1 has a clean fix - batch the query, and the
quadratic disappears entirely with no loss of information. String concatenation
has a clean fix - use a buffer. The agentic loop's quadratic term is *intrinsic*:
attention over a growing context is what makes the model able to use the growing
context. You cannot batch it away. Every available mitigation trades away
something real - compaction discards information, sliding-window attention
discards reach, retrieval-instead-of-context discards the model's ability to
notice connections you did not query for. Reach for the N+1 reflex here and you
will waste time looking for the free fix. There isn't one; there is only a choice
about what to give up.
