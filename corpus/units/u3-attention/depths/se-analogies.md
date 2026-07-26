---
unit: u3
depth: se-analogies
---

Every analogy below comes in a pair, because a single analogy is
indistinguishable from a definition and gets believed too hard. Two analogies
that agree on some structure and disagree elsewhere let you locate what is
actually being claimed: the shared part is the content, and the disagreement
marks the boundary. Each pair ends with where it breaks. Read that part.

## The problem: mix information across a sequence with no recurrence

**Analogy A: the shuffle stage of a MapReduce job.** Mappers produce records
independently and in parallel; nothing useful can be computed until records
that belong together have found each other, so there is a stage whose entire
job is routing data between workers based on the data's own content.

**Analogy B: an all-reduce in a distributed training job.** Every rank holds a
partial result; every rank needs a function of all the partials; the operation
is collective, has no leader, and completes in a fixed number of steps
regardless of how the data is distributed.

**The shared structure:** an unavoidable all-to-all communication phase
between two embarrassingly parallel phases. In all three cases the per-item
work is trivial and the interesting engineering is entirely in the exchange.
Both analogies also make the right prediction about cost: all-to-all is the
expensive part of MapReduce and the expensive part of distributed training,
and it is the expensive part of a transformer.

**Where this breaks:** the shuffle routes by exact key equality - records with
identical keys land together, records with different keys never meet.
Attention has no keys in that sense: every position exchanges with every other
position, always, with a continuous weight rather than a boolean. There is no
partitioning and nothing is ever excluded, which is precisely why you cannot
build a hash index to make it cheap. All-reduce breaks differently: it
computes the *same* aggregate for every rank, whereas attention computes a
different weighted aggregate per position. If all-reduce were the right
picture, every token would end up with the same output vector.

## Queries, keys, and values

This is the pair that matters most in the unit, and the one whose failure
modes cause the most durable confusion.

**Analogy A: a dictionary lookup with fuzzy matching.** You have a query. The
store holds key-value pairs. You compare the query against the keys, and you
get back the value belonging to the best match. Unlike `dict[k]`, the match is
scored by similarity rather than by equality, so a query that is merely
*close* to a key still retrieves something.

**Analogy B: a weighted aggregate query over a table.** In SQL terms, close
to:

```sql
SELECT SUM(similarity(q.vec, r.key) * r.value) / SUM(similarity(q.vec, r.key))
FROM rows r
```

one such aggregate per query row. You are not selecting a row; you are
computing a normalized weighted average over every row in the table, where the
weight is a function of how well that row matches.

**The shared structure:** content-addressed retrieval. In both, what you get
back is determined by comparing your request against a description that the
data supplies about itself, never by an index, an offset, or a position. Both
also make the key/value split load-bearing in the same way: the thing you
match against and the thing you receive are different fields of the same
record, so data can be findable by one property and useful for another. That
decoupling is the single most important structural fact the pair transmits.

**Where this breaks - analogy A:** three ways, and each one corresponds to a
real misconception.

1. A dictionary returns *one* value. Attention always returns a blend of
   *every* value, weighted. There is no argmax, no winner, and no threshold
   below which a row is excluded. Even the worst-matching position contributes
   a nonzero amount to every output.
2. A dictionary can fail. There is no `KeyError` and no `None` in attention -
   the softmax denominator guarantees the weights sum to 1, so *something* is
   always returned, at full magnitude, no matter how badly everything matched.
   A query with no good match does not get nothing; it gets a diffuse average
   of everything, which is a completely different failure mode and one the
   rest of the network has to detect for itself.
3. The store is not persistent. `dict` outlives the lookup; here $K$ and $V$
   are rebuilt from the current input on every forward pass and thrown away.
   Nothing is stored between calls.

**Where this breaks - analogy B:** the table is not a separate table. In SQL
you have a query relation and a data relation, and they are different objects.
In self-attention the rows being aggregated *are* the query rows: $Q$, $K$,
and $V$ are three projections of one matrix $X$, so every row is
simultaneously the thing issuing a query and a row being aggregated over.
Writing it honestly means self-joining the table against itself, which is also
exactly why the cost is quadratic in the row count and why you cannot make the
"table" bigger without making the query set bigger too. The SQL picture also
suggests an index would help. It would not: `similarity` is a dense dot
product against a learned continuous vector, so there is no equality predicate
to index on, and the approximate-nearest-neighbor structures that would apply
are exactly what sparse-attention research is trying to make work.

The two analogies disagree in a productive place. Analogy A says "you get one
thing back"; analogy B says "you get a blend of everything". Analogy B is
correct, and the gap between them is the size of misconception M1 - the
intuition that attention *selects* comes almost entirely from having only
analogy A.

## Scaled dot-product attention: the equation

**Analogy A: a nested-loop self-join followed by a group-by.** For each row,
scan every row, compute a score, then collapse the scan back down to one
output row. Two loops and a reduction, which is exactly the shape of
$QK^T$ followed by $AV$.

**Analogy B: one round of message passing on a complete graph.** Every node
sends a message to every other node, each edge carries a weight computed from
the two endpoints, and each node sums its incoming messages. This is literally
how graph neural network libraries implement attention.

**The shared structure:** an $n \times n$ intermediate that is materialized,
consumed, and discarded, with the output having the same cardinality as the
input. Both analogies get the crucial shape fact right: you produce $n^2$
things in the middle and $n$ things at the end. Both also make it obvious why
the operation composes - the output is shaped like the input, so you can stack
it, which a join that emitted $n^2$ rows would not let you do.

**Where this breaks:** a join emits its intermediate as the *result*, and a
graph message-pass generally has a fixed, sparse edge set. Attention does
neither. The $n \times n$ matrix is never the answer - treating it as the
output is misconception U3-M4 and is the most common shape error in the topic.
And the graph is complete and re-weighted from scratch on every forward pass
for every input, so there is no fixed topology to exploit; the "edges" are
recomputed values, not structure. The message-passing analogy also invites you
to think the weights are edge attributes learned per edge. They are not -
there are no per-edge parameters at all, which is the entire reason the
mechanism handles any sequence length.

## Why divide by the square root of $d_k$

**Analogy A: the gain stage before a saturating amplifier.** The amplifier
clips: past a certain input level, the output stops responding and further
increases do nothing. So you set the input gain to keep the signal in the
range where the amplifier is still responsive. You do not do this because
something would overflow; you do it because outside that range the stage stops
carrying information.

**Analogy B: the temperature parameter on a sampler.** Same softmax, and
dividing scores by a constant before it is exactly what a temperature does.
Low temperature sharpens toward argmax, high temperature flattens toward
uniform, and the ranking is untouched either way.

**The shared structure:** a scalar that controls how sharply a comparison
resolves, without changing which side wins. In both cases the concern is the
*shape* of the response, not its magnitude, and in both cases the wrong
setting produces a technically-valid output that has stopped conveying
gradations. Analogy B is more than an analogy - it is an identity.
$\text{softmax}(S/\sqrt{d_k})$ is softmax at temperature $T = \sqrt{d_k}$.

**Where this breaks:** temperature in unit u6 is a knob you turn at inference
on a trained model. $\sqrt{d_k}$ is a compile-time constant, fixed by the head
dimension, present during training, and not tunable afterward - the model's
weights adapted to it. Changing it post-hoc breaks the model rather than
adjusting its behavior. The amplifier analogy breaks in a more important
place: it suggests the fix is data-dependent, the way an automatic gain
control watches the signal and adapts. It is not. $\sqrt{d_k}$ never looks at
a single score. It is derived from the architecture alone, which is exactly
what separates it from every normalization step you have written - and
believing otherwise is misconception U3-M2. If you take one thing from this
pair: it is a fixed gain, never an automatic one.

## The whole computation by hand

**Analogy A: a weighted moving average, as in an EWMA over a metrics stream.**
Output is a convex combination of inputs; weights are non-negative and sum to
1; the result is trapped between the minimum and maximum of what it averages.

**Analogy B: an ensemble vote with confidence-weighted members.** Each member
contributes a prediction, weighted by how confident it is, and the ensemble
output is the weighted mean. Nobody is excluded; low-confidence members
contribute little.

**The shared structure:** convexity, and it is worth more as a debugging tool
than as intuition. Any output component must lie within the range of that
component across the inputs, which is a free correctness check on every hand
computation and on every implementation. If your attention output has a
component outside the range of $V$'s corresponding column, you have a bug,
with certainty, before you have looked at anything else.

**Where this breaks:** an EWMA's weights are fixed coefficients chosen by the
engineer, decaying by position. Attention's weights are recomputed from
content for every query on every forward pass, and depend on position only
through whatever positional information was baked into the token vectors. The
ensemble analogy breaks on the source of the weight: ensemble confidence is a
property of the member alone, whereas an attention weight is a property of the
*pair* - the same value vector gets a large weight from one query and a
negligible one from another, in the same forward pass. There is no such thing
as a token's confidence, only a query-key compatibility.

## Causal masking

**Analogy A: a predicate pushed into the join condition.** `... ON k.pos <=
q.pos`, evaluated as part of the join rather than as a filter over its
results. The predicate participates in the aggregation; it does not clean up
afterward.

**Analogy B: snapshot isolation in an MVCC database.** A transaction reads a
consistent view of the world as of its own timestamp. Rows committed later
exist on disk and are perfectly readable by other transactions, but this
reader is constructed so that it cannot observe them.

**The shared structure:** visibility restricted by an ordering, applied at
read time, with the restricted view being internally consistent. Both
analogies transmit the essential point that the future is not *deleted* - the
data is right there, in the same tensor, being used by other rows in the same
matrix multiply. Position 5 sees token 7 in the exact same operation in which
position 3 does not.

**Where this breaks - analogy A:** a `WHERE`-style filter drops rows and the
surviving aggregate is computed over what remains. That is correct only if the
predicate is inside the aggregation. If you filter *after* computing the
softmax, you get misconception U3-M3: rows summing to less than 1, with the
deficit largest at position 1 and vanishing at position $n$, because the
normalization already gave part of each budget away to rows you then deleted.
The SQL analogy is safe only in its pushed-down form, and the difference
between the two forms is the entire content of question u3-q9.

**Where this breaks - analogy B:** MVCC hides rows to give one reader a
coherent view of a world where those rows genuinely exist and could have been
read at a different timestamp. The causal mask is enforcing a counterfactual:
at inference the future tokens do not exist at all, and the mask exists so
that training matches that condition. Snapshot isolation is about consistency
among concurrent readers; causal masking is about not learning a function you
will be unable to evaluate. A leak in MVCC gives you a stale or inconsistent
read. A leak in the causal mask - a mask constant of $-10$ instead of
$-10^9$ - gives you a model that learned to predict a token by looking at it,
which is not an anomaly you notice in the loss curve; it is an anomaly you
notice when generation is worthless.

## What attention costs

**Analogy A: a nested-loop join with no index.** $n$ outer rows times $n$
inner rows, quadratic, and every database engineer's instinct on seeing it is
to reach for a hash join or an index scan.

**Analogy B: a full mesh in a distributed system.** $n$ nodes each talking to
$n$ others is $n^2$ connections. It is why service meshes hit scaling walls,
why $n^2$ gossip gets replaced with trees and epidemics, and why "just have
everyone talk to everyone" stops being an option somewhere in the low
hundreds.

**The shared structure:** quadratic pairwise work that is invisible at small
$n$ and dominant at large $n$, with the crossover arriving suddenly. Both
analogies also transmit the right intuition about mitigation: you do not fix
$n^2$ by getting faster hardware, you fix it by not doing all the pairs -
which is precisely what sliding-window, sparse, and linear attention attempt.

**Where this breaks:** and this is the part that changes what you should
expect from the field. A nested-loop join is quadratic only because it lacks
an index. Give the join an equality predicate and a hash table and it becomes
linear, exactly. Attention has no equality predicate. The match is a dense dot
product in a continuous learned space, so there is no key to hash, no B-tree
to descend, and no exact structure to exploit - which is why fifteen years of
join optimization does not transfer, and why every sub-quadratic attention
variant is an *approximation* that trades quality for cost rather than a free
algorithmic win. The mesh analogy breaks on what is being spent: a full mesh
is limited by connections and bandwidth, whereas attention's $n^2$ is
arithmetic on a device that is very good at arithmetic. That is why attention
survives to far larger $n$ than any real mesh would, and why the thing that
actually breaks first is memory rather than FLOPs - the $n \times n$ matrix
has to live somewhere, which is the problem FlashAttention solves without
touching the complexity at all.

## What you now know

**Analogy A: attention is one instruction, not one algorithm.** Six dense
linear-algebra operations, no branches, no loops over data, no early exit -
closer to a wide SIMD instruction than to any function you would write with
control flow.

**Analogy B: attention is a routing layer, and the MLP block is the compute.**
Attention moves information between positions and can only ever blend what is
already present; the block that follows it transforms information within a
position and is where new content can be produced.

**The shared structure:** attention is narrow and mechanical, and its power
comes from being repeated - many heads wide, many blocks deep - rather than
from any one invocation being clever. Both framings predict correctly that a
single attention layer in isolation is nearly useless.

**Where this breaks:** the instruction framing undersells the learned part -
$W^Q$, $W^K$, and $W^V$ are the entire behavior of the head, and swapping them
changes what it computes completely, which is not true of a SIMD instruction.
The routing framing undersells the value path: attention does not merely move
$x_j$ around, it moves $x_j W^V$, so it applies a learned transformation on
the way. Routing that transforms its payload in transit is doing more than
routing, and unit u4's residual-stream view is the framing that finally
accounts for it.
