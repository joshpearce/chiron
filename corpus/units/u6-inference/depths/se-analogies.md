---
unit: u6
depth: se-analogies
---

## Where inference sits in the stack

**The pair.** Inference-time configuration is like runtime flags on an
already-compiled binary, and it is like query planner hints on a fixed schema
with fixed data.

**The shared structure.** In both cases there is an artifact that was produced by
an expensive offline process (compilation, ETL and index building) and is now
immutable, plus a set of knobs that select among execution strategies over that
artifact. `GOGC=200` does not change your program's semantics; a planner hint
does not change what rows exist. The knobs pick a path through fixed content.
Temperature, cache policy, quantization format, and speculative decoding are
that second category, applied to a 19 GB weight file that stopped changing the
day training ended.

**Where this breaks.** A planner hint is guaranteed semantics-preserving: the
result set is identical no matter which plan runs. Three of this unit's four
techniques hold to that standard, but quantization does not - it rewrites the
artifact. It is closer to `-ffast-math` than to a planner hint: a compilation
flag that trades exactness for speed and is usually fine and occasionally is
not. Keep quantization in a different mental bucket from the other three, and
the whole unit stays straight.

## Temperature, top-k, top-p on one set of logits

**The pair.** Sampling from softmaxed logits is like weighted random selection
from a service registry, and top-k/top-p are like `LIMIT k` versus a cursor that
stops once cumulative relevance crosses a threshold.

**The shared structure.** Both start from a ranked list with numeric weights and
end with one selection. Temperature is the exponent you apply to the registry
weights before rolling: weight each backend $w_i^{1/T}$ instead of $w_i$. At
$T \to 0$ you have pinned all traffic to the single highest-weight backend; at
$T \to \infty$ you have round-robin. Anyone who has tuned a weighted balancer
knows the exponent does not change which backends exist or how healthy they are -
it changes traffic distribution over a fixed roster. That is exactly and
completely what temperature does.

The truncation knobs then map onto result-set pagination. `top_k=40` is
`ORDER BY score DESC LIMIT 40`. `top_p=0.9` has no SQL equivalent because it is
data-dependent: keep scanning the sorted result until the running sum of
normalized scores reaches 0.9, then stop. That is a cursor with a mass-based
stop condition, and its page size varies per query.

**Where this breaks.** In a load balancer the weights mean something outside the
balancer - capacity, health, cost - and you can go look at the backend to check.
Logits have no external referent. There is nothing to inspect that would tell you
whether the top-weighted token deserves its weight. Do not import the intuition
that the weights are grounded in an auditable reality; they are grounded in
training-corpus frequency and nothing else.

## T=0 is not truth mode

**The pair.** $T=0$ decoding is like a reproducible build, and it is like a cache
serving a 100% hit rate.

**The shared structure.** Both give you a strong determinism guarantee and zero
correctness guarantee, and both are routinely misread as the second because
people want the second. A reproducible build guarantees byte-identical output
from identical inputs; it says nothing about whether the program is correct. A
100%-hit-rate cache guarantees you get the same response every time; if the
cached value is wrong, it is wrong with perfect consistency and you have removed
the variance that might have tipped you off. `temperature=0` in your API call is
precisely this: a guarantee of same-input-same-output, and a removal of the
variance signal.

There is a second alignment worth naming. Both reproducibility mechanisms are
genuinely valuable for the *engineering* reasons: diffs, regression tests, bug
reproduction, cache-key sanity. Set $T=0$ in your eval harness for exactly the
reasons you pin dependency versions in CI. That is a workflow property, not an
epistemic one.

**Where this breaks.** A reproducible build is checkable against source - you can
read the code and decide whether the binary should be correct, and the build's
determinism lets you bisect. Greedy decoding has no source to check against. And
the analogy inverts on one axis: rebuilding does not make your program correct,
but *resampling* at $T > 0$ genuinely does surface information (disagreement
across samples is a real uncertainty signal, which is why self-consistency
sampling works). Determinism costs you a diagnostic here that it does not cost
you in a build system.

## The KV cache is not memory

**The pair.** The KV cache is memoization of a pure function, and it is an
incremental build cache like ccache or a Bazel action cache.

**The shared structure.** Both store derived values keyed by inputs, both are
exactly reconstructible, and both are correctness-neutral by construction:
clearing them costs time and nothing else. If `ccache` returned a different
object file than a cold build, you would file a bug, not accept it as the cache
"remembering" something. Apply the same standard to the KV cache and M17 dies on
the spot - a cache whose eviction changes output is a broken cache, and the KV
cache is not broken.

The build-cache half of the pair is the sharper one, because it also explains
*why* caching is possible here. `ccache` works because compilation is a pure
function of source plus flags. KV caching works because the causal mask makes
position $i$'s key and value a pure function of tokens $1..i$. Both are the same
purity argument, and both fail the same way when purity fails: `__DATE__` in your
source breaks ccache, and bidirectional attention breaks KV caching.

**The second-order alignment worth having.** Prefix caching in a serving stack is
the same idea as content-addressed build caching one level up: two requests
sharing a system prompt share a cache prefix, so you key the cache on the token
prefix hash and reuse across requests. This is why providers charge less for
cached input tokens - it is literally the same economics as a shared build cache
in CI.

**Where this breaks.** Build caches are keyed on a content hash and are
order-agnostic within a target. The KV cache is strictly sequential and
prefix-structured: you can reuse a prefix but you cannot reuse a suffix, and
inserting a token in the middle invalidates everything after it. It is closer to
an append-only log with a materialized fold than to a hash-keyed store. That
structural constraint is exactly why context compaction is hard, and it is what
u8 has to live with.

## Prefill and decode are different machines

**The pair.** Prefill versus decode is like a batch job versus a REPL, and it is
like vectorized SIMD over a contiguous array versus pointer-chasing a linked
list.

**The shared structure.** Both pairs contrast the same underlying quantity:
useful work per byte moved. A batch job amortizes fixed setup (opening
connections, loading a working set) over millions of records; a REPL pays it per
statement. SIMD over a contiguous array loads a cache line and does sixteen
operations on it; pointer-chasing loads a cache line, uses eight bytes of it, and
stalls on the next dependent load. In both, the slow side is not doing more work,
it is failing to amortize a memory cost.

Decode is the pointer-chasing case, precisely. Each forward pass hauls 1.8 GB of
active weights out of unified memory to perform one token's worth of arithmetic,
and it cannot start the next pass until this one's output is known - a dependent
load, in the literal sense. Every intuition you have about why a linked-list
traversal is slow on modern hardware transfers directly, including the fix:
batching is to decode what array-of-structs-to-struct-of-arrays is to
pointer-chasing. Get more work per loaded byte.

**Where this breaks.** With pointer-chasing you can usually restructure the data
and eliminate the dependency. Autoregressive decode's dependency is semantic, not
representational: token $t+1$ genuinely requires token $t$, and no memory layout
fixes that. It is why the workarounds are exotic - batch across *different users*
(no dependency between them) or speculate and verify (guess the dependency and
check). There is no data-structure change that makes single-stream decode
parallel.

**A second pair, for the quadratic.** Attention over a growing context is like a
self-join with no index, and it is like an $O(n^2)$ all-pairs broadcast in a
cluster. Shared structure: cost grows with pairs, not items, and adding machines
raises throughput without changing the exponent. Where this breaks: you can
usually add an index to kill a self-join, and the attention analogue - sparse,
sliding-window, or linear attention - changes what the model computes, not just
how fast. An index is semantics-preserving; sparse attention is not. That is the
whole reason full attention is still standard.

## Quantization: what four bits hold

**The pair.** 4-bit weight quantization is like fixed-point arithmetic replacing
floats in an embedded DSP, and it is like reducing an image to a 16-entry palette
computed per tile.

**The shared structure.** All three (quantization, fixed-point, palettized
images) replace a value drawn from a continuous range with a small integer index
into a locally-chosen set of representable levels, plus a scale factor that maps
indices back. All three have the same error bound - half a step - and all three
work when the consuming computation is robust to that error and fail when it is
not. The per-tile part of the palette analogy is load-bearing: a global 16-color
palette destroys an image, a per-tile palette often looks fine. That is precisely
group-wise quantization, and it is why group size is a real knob.

**Where this breaks, and it matters.** Both analogies suggest the error is
*visible* - you can look at a palettized image and see the banding, and a DSP
engineer can compute the exact quantization noise floor of their filter. Neither
maps cleanly onto a network, because there is no output space in which you can
inspect the damage directly. The 9% weight perturbation does not produce a 9%
"blurrier" answer; it produces an answer that is usually identical and
occasionally structurally different, and you cannot tell by looking at any
intermediate.

The deeper break: fixed-point DSP and palettization both operate on data. Model
quantization operates on the *program*. The closest honest software analogy is
compiling with reduced-precision intermediates and finding the program still
passes its tests - and the reason it passes is a property of the program's
numerical conditioning, not of the compression ratio. That is exactly M16's
error, and it is why the bit-counting intuition that serves you well on data
misleads you on weights.

## Where quantization falls off the cliff

**The pair.** An activation outlier ruining its quantization group is like one
enormous row forcing a columnar compression block to a wide encoding, and it is
like one slow query pinning a connection pool.

**The shared structure.** A shared resource is sized by its worst member. A
Parquet page's dictionary or bit-width is set by the widest value in it, so one
outlier inflates the encoding for every other value in the page; a connection
pool's effective capacity is set by the slowest query holding a connection. In
all three, the average case is irrelevant and the tail sets the cost. Quantized
weight groups share a scale $s$ chosen from the group's range, so one $10\sigma$
weight in a group of 64 nearly triples the step size for the other 63 - measured,
9.0% error becomes 23.7%.

The fix is the same fix in all three domains, which is the point of the pair:
*separate the outliers out*. Columnar formats keep exceptions in a side list
(PFOR-delta and friends). Connection pools get a separate pool for long queries.
AWQ identifies outlier channels and rescales them into the activations;
LLM.int8() keeps them in a higher-precision side path. In every case you do not widen the common
path to accommodate the tail; you route the tail elsewhere.

**Where this breaks.** Columnar exception lists are exact - the outlier value is
recoverable bit-for-bit. Outlier-aware quantization is only *better*, not exact,
and there is no threshold at which you can declare the model safe. There is also
no closed-form theory for where the cliff sits, unlike queueing theory's
utilization knee, which is derivable. Quantization cliffs are found empirically,
per model, which is why every serious release ships measured benchmark deltas
per quantization format rather than a formula.

**A second pair, for compounding.** Quantization damage accumulating over a long
generation is like floating-point error accumulating in an iterative solver, and
it is like a small clock skew accumulating in a distributed log. Shared
structure: a per-step error far below any single step's tolerance becomes
decisive after enough dependent steps, so the failure appears in long runs and
never in unit tests. Where this breaks: numerical error growth in a solver is
analyzable via condition number, and you can bound it. There is no condition
number for a 40-step chain of thought, and the discreteness of token selection
means the error is not smooth - it is zero until an argmax flips, then it is
total.

## Speculative decoding: exact, not approximate

**The pair.** Speculative decoding is like branch prediction with speculative
execution and rollback, and it is like optimistic concurrency control (a
compare-and-swap loop, or MVCC write validation).

**The shared structure.** All three do work on a guess before the authoritative
answer is available, then validate against the authority and discard on
mismatch. All three are *exact*: a mispredicted branch leaves no architectural
state behind, a failed CAS commits nothing, a rejected draft token is not
emitted. And all three win for the same reason - there is idle capacity while
waiting for the authoritative result, and the guess costs less than the wait.
The CPU's execution units would idle during a branch resolution; your Mac's
arithmetic units idle during a memory-bound decode step. Same shape, same fix.

The economics align too. Branch prediction pays off when prediction accuracy is
high and misprediction penalty is bounded; speculative decoding pays off when
acceptance rate $\alpha$ is high and the draft is cheap. Both degrade gracefully
to "no worse than not doing it, minus overhead" and neither degrades to "wrong."

**Where this breaks, and it is the interesting part.** A mispredicted branch is
pure loss - the pipeline flushes and you redo the work. A rejected draft token is
*not* pure loss: the rejection step emits a corrected token drawn from the
residual distribution, so a rejected step still makes forward progress of exactly
one token. Speculative decoding has no equivalent of a pipeline flush; its worst
case is one token per target pass, which is the non-speculative rate. That is a
strictly better failure mode than branch misprediction, and it is why nobody
worries about pathological acceptance rates.

The OCC half breaks differently. A failed CAS retries the whole operation, and
livelock under contention is a real hazard. Speculative decoding never retries -
the verification step *repairs* rather than *re-runs*, using the residual
distribution to emit precisely the token the target owed. No retry loop, no
livelock, no unbounded worst case. If you want one sentence separating the
analogy from the reality: this is optimistic concurrency where a conflict is
resolved by a closed-form correction instead of an abort.

## What this unit did and did not change

**The pair.** The distinction running through this unit is the distinction
between changing a program and changing its execution plan, and it is the
distinction between a schema migration and index tuning.

**The shared structure.** In both software pairs, one side alters what the system
means and the other alters only how fast it gets there, and the operational
consequences are completely different - migrations need review, backups, and a
rollback plan; index tuning needs a benchmark. Sampling, KV caching, and
speculative decoding are index tuning. Quantization is the one migration in the
set, and it is the only one that belongs in a changelog with a measured quality
delta.

**Where this breaks.** In a database you can always tell which category you are
in by reading the DDL. With a model you cannot, because the serving stack
presents all four as configuration in the same YAML file, side by side, with no
signal that one of them rewrote the weights and three of them did not. That flat
presentation is most of why these misconceptions persist: the interface gives
you no reason to believe `temperature`, `kv_cache_dtype`, `quantization`, and
`num_speculative_tokens` are categorically different things. They are. Three are
free; one has a bill.
