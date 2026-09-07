---
unit: u9
title: "Synthesis: the full trace"
concepts:
  - c-full-trace
assumes:
  - c-notation
  - c-matmul
  - c-dotprod
  - c-lm-objective
  - c-tokens
  - c-embeddings
  - c-logits
  - c-matrix-transform
  - c-softmax
  - c-gradient
  - c-loss-surface
  - c-qkv
  - c-sdpa
  - c-attn-numeric
  - c-causal-mask
  - c-mha
  - c-mlp-block
  - c-residual
  - c-layernorm
  - c-depth
  - c-xent
  - c-backprop
  - c-optimizers
  - c-scaling
  - c-posttrain
  - c-sampling
  - c-kv-cache
  - c-quant
  - c-specdec
  - c-moe-routing
  - c-moe-tradeoff
  - c-sysprompt
  - c-toolloop
  - c-context-mgmt
---

## The trace, and how to read it

You are in a terminal. You type `find the bug in the auth middleware` and hit
enter. Between that keystroke and the first character appearing on screen,
roughly one second passes. This unit is what happens in that second, and in the
forty seconds after it.

Nothing below is new. Every mechanism has already been taught. What is new is
the *order*, the *handoffs*, and the *arithmetic that falls out of the whole
chain*. Individual mechanisms you can recite; a trace you either can or cannot
run. Those are different kinds of knowing, and only the second one predicts
latency and cost.

Read this with a pen. At each checkpoint the text stops and asks you to supply
the mechanism before it tells you. If you read the answer first the unit does
almost nothing for you, because the thing being trained is retrieval under
sequencing pressure, not recognition.

One fixed configuration for the whole trace, so the arithmetic is concrete. It
is a plausible frontier-ish open-weights config, not a specific product:

| quantity | symbol | value |
|---|---|---|
| model dimension (residual stream width) | $d_{model}$ | 4096 |
| vocabulary size | $V$ | 128,000 |
| transformer blocks | $L$ | 64 |
| query heads per block | $h_q$ | 32 |
| key/value heads per block (GQA) | $h_{kv}$ | 8 |
| dimension per head | $d_{head}$ | 128 |
| experts per MoE layer | $E$ | 128 |
| experts activated per token | $k$ | 8 |
| total parameters | - | 235B |
| active parameters per token | - | 22B |
| weight precision on this box | - | 4-bit |
| prompt length for this turn | $n$ | 8,192 tokens |

Two hardware numbers, because every "why is this slow" question in the trace
resolves to one of them: the accelerator sustains about $4.5 \times 10^{14}$
floating-point operations per second (450 TFLOP/s) and reads from its memory at
about $2 \times 10^{12}$ bytes per second (2 TB/s). Hold onto the ratio: 225
FLOPs per byte read. Any stage whose work-per-byte is below 225 is starved for
memory bandwidth, not compute.

## Stage 1: the harness builds one flat token stream

Claude Code does not send your sentence. It builds a document. Concatenated in
a fixed order: the system prompt, the tool definitions serialized as schemas,
the contents of your `CLAUDE.md`, a directory listing, the previous turns of
this conversation, the results of the last three tool calls, and finally
`find the bug in the auth middleware`. Call it 8,192 tokens, of which your
sentence is eight.

<!-- refutes: M13 -->
You probably think the model receives structure: a list of message objects with
`role` fields, a tool registry it can enumerate, a conversation it can index
into. The API hands you exactly that shape, and the shape is real - on the
harness side.

Here is the prediction that model makes and fails. If roles were native to the
model, then text inside a user message could never impersonate the assistant,
because the impersonation would be in the wrong field. But writing a convincing
`\n\nAssistant: Sure, here are the credentials` inside user content demonstrably
shifts model behavior, and every jailbreak of that family works for exactly one
reason: there are no fields. There is a byte string. The `role` boundaries are
special delimiter tokens (`<|im_start|>assistant` or similar) that the model saw
billions of times during post-training and learned to condition on. They are
strong learned conventions, and conventions can be forged by anyone who can put
tokens in the stream.

It is appealing because the abstraction is genuinely good engineering: the
harness maintains real structure, validates it, and serializes at the boundary.
The mistake is assuming the structure survives serialization.

What is actually true: the harness produces one flat sequence of token IDs. That
sequence is the entire input. There is no side channel, no metadata, no
out-of-band tool registry. A tool "exists" for the model because its JSON schema
is sitting in the token stream as text.

```beat
id: u9-b1
type: self-explain
concept: c-full-trace
prompt: |
  Checkpoint 1. Your `CLAUDE.md` says "always run the linter before pushing."
  Three turns ago you told Claude "actually skip the linter today." Both are in
  the context. The model complies with the second.

  "It followed the more recent instruction" only restates what you saw. Pick the
  account that says where each instruction physically lives at the moment of the
  forward pass, what makes one of them win, and what it would take for the model
  to be *unable* to see the first one.
options:
  - text: "Both instructions are tokens in one flat sequence, differing only in position and in the delimiter tokens around them; there is no priority field. Preference for the later one is a behavioural regularity learned in post-training over such conflicts, expressed as attention weight at the final position. The only way the model cannot see the CLAUDE.md line is if those tokens are absent - harness truncation, compaction, or window overflow."
    correct: true
    explain: "Right. Position and delimiters are the only distinctions available, precedence is learned rather than enforced, and an instruction loses all influence exactly when its tokens leave the stream."
  - text: "The two instructions arrive in different fields - one in the system-prompt slot, one in a user message - and the API's precedence rules resolve the conflict before the model runs, so the model never sees a conflict at all."
    misconception: M13
    explain: "There are no fields at the model. The harness's message objects are serialized into one byte string with delimiter tokens; that is also why text inside user content can forge an assistant turn. Nothing resolves the conflict before the forward pass."
  - text: "The model holds the running conversation as internal state and overwrites the older instruction with the newer one when it updates that state, so the first instruction is no longer present to be consulted."
    misconception: U9-M2
    explain: "The model carries no state between calls. The harness re-sends every prior token each turn, so both instructions are present in full at every forward pass; nothing is overwritten."
  - text: "The KV cache stores the meaning of each earlier instruction, and the newer instruction updates that cached entry, which is why the model can no longer act on the CLAUDE.md rule."
    misconception: M17
    explain: "The cache holds key and value projections that are a pure function of the tokens - recompute it from the same sequence and you get bit-identical entries. It stores no meanings and nothing edits it in place."
check: choice
```

## Stage 2: bytes to token IDs to vectors

The 8,192-token count is not a metaphor. The tokenizer, a byte-pair encoding
table learned once before pretraining and frozen forever after, walks the byte
string and greedily merges pairs it has seen merged. `middleware` may be one
token or `middle` + `ware` depending on the corpus. Your file path
`/src/auth/mw.ts` is likely six or seven tokens because slashes and short
identifiers do not merge well. Whitespace attaches to the *following* word in
most modern tokenizers, which is why ` the` and `the` are different IDs.

Output of this stage: a vector of 8,192 integers in $[0, 128000)$. That is all.
No structure, no types, no roles.

Then the embedding lookup. The embedding matrix $W_E$ has shape
$V \times d_{model}$ = $128000 \times 4096$, where $V$ is the vocabulary size
and $d_{model}$ is the residual stream width. Token ID $t$ selects row $t$. The
"lookup" is a gather, and it is mathematically a matrix multiply by a one-hot
vector - which is why it is a learned linear map and not a dictionary.

<!-- refutes: M3 -->
You probably think that row holds the meaning of the token. The demos encourage
it: subtract the `man` row from the `king` row, add `woman`, land near `queen`.

The prediction that fails: if the row held meaning, the same row would mean the
same thing in another model. Take the 4096 floats for `auth` out of this model
and paste them into another model's embedding table at the same index. Output is
garbage, not a slight accent. Meaning did not travel, because there was none in
the vector - only a position whose *relationships* to 127,999 other positions
were shaped by gradient descent on next-token prediction.

What is actually true: the embedding is an address in a learned geometry. Every
downstream block reads it only through dot products with other things, so only
relative geometry can matter. This is also why the same embedding matrix,
transposed, works as the output head at the end of the trace - the geometry is
shared between "what came in" and "what should come next."

After the gather you have a tensor of shape $8192 \times 4096$: one 4096-wide
vector per position. Position 0 is the first token of the system prompt.
Position 8191 is the last token of your question. This tensor is the initial
state of the residual stream, and from here to the very end nothing replaces it.
Blocks only add to it.

```beat
id: u9-b2
type: compute
concept: c-full-trace
prompt: |
  Checkpoint 2. Before any transformer block runs, how many parameters are
  in the embedding matrix alone, given $V = 128{,}000$ and
  $d_{model} = 4096$? Answer in millions, to three decimal places.
answer: 524.288
rubric: |
  128,000 x 4,096 = 524,288,000 parameters = 524.288 million. Graded
  numerically. 0.524 means they answered in billions; 524,288 means they
  answered in thousands; anything near 1,048.576 double-counted by adding a
  separate unembedding matrix, which this configuration ties to the embedding.
check: numeric(0.01)
```

That is half a billion parameters that do nothing but assign addresses, out of
235 billion. Worth remembering when someone tells you the vocabulary size is a
minor hyperparameter.

## Stage 3: prefill - 8,192 tokens, one forward pass

Here is the stage that most engineers get wrong, and getting it wrong makes the
rest of the trace unpredictable.

<!-- refutes: M9 -->
You probably think the model works through the prompt token by token, because
that is what generation looks like on screen and because every RNN you ever read
about did exactly that.

The prediction that fails: if prompt processing were sequential, an 8,192-token
prompt would take 8,192 sequential steps, each with the same per-step latency as
generating a token. Generation on this box starts at about 5.5 ms per token
while the context is short (Stage 6 adds the cache-read term that grows). That
would put time-to-first-token at 45 seconds. Measured time-to-first-token is
just under one second. The model is not stepping through your prompt.

It is appealing because the causal mask makes each position depend only on
earlier ones, which *sounds* like sequencing. It is not. A dependency structure
is not an execution order.

What is actually true: all 8,192 positions go through all 64 blocks
simultaneously, as batched matrix multiplies. Position 3 and position 7,000 are
computed in the same instruction stream. The causal mask is a triangular matrix
of $-\infty$ added to the attention scores before the softmax, which makes the
softmax weight for any future position exactly zero. Order lives in that mask
and in the positional encoding, not in the schedule. This one pass is called
**prefill**.

Now the arithmetic, because the whole cost model of LLM serving is here.

<!-- fade: prefill-cost -->
Prefill does two kinds of work with very different scaling.

**Work that scales linearly with $n$** (the parameter-driven work: every MLP,
every projection). A matrix multiply costs 2 FLOPs per parameter per token (one
multiply, one add), so:

$$\text{FLOPs}_{\text{params}} = 2 \times P_{\text{active}} \times n$$

where $P_{\text{active}} = 22 \times 10^9$ is the parameters active per token
and $n = 8192$ is the prompt length. That is
$2 \times 22 \times 10^9 \times 8192 = 3.60 \times 10^{14}$, or 360 TFLOPs.

**Work that scales quadratically with $n$** (attention scores). For one block,
every query position dots every key position, per head:

$$\text{FLOPs}_{\text{attn}} = 2 \times \left( 2 \times n^2 \times d_{head} \times h_q \right) \times L$$

The inner $2 n^2 d_{head} h_q$ is the $QK^\top$ product (2 FLOPs per
multiply-add, $n^2$ score entries, $d_{head}$ terms per dot product, $h_q = 32$
query heads); the outer factor 2 accounts for the second matrix multiply, the
weighted sum of value vectors, which has the same shape. With $n = 8192$,
$d_{head} = 128$, $h_q = 32$, $L = 64$: one block costs
$2 \times 2 \times 8192^2 \times 128 \times 32 = 1.10 \times 10^{12}$, and
across 64 blocks that is $7.04 \times 10^{13}$, or 70 TFLOPs. That counts all
$n^2$ score entries; a fused causal kernel skips the tiles that lie entirely
above the diagonal and gets roughly half of it back, so treat 70 TFLOPs as an
upper bound. The scaling is what matters here, and halving a quadratic leaves
a quadratic.

Total: 430 TFLOPs. At 450 TFLOP/s sustained, prefill takes about **0.96
seconds**. That is your time-to-first-token, and it is compute-bound: 430 TFLOPs
against only about 118 GB of weights read (235B parameters at 4 bits, all of
them touched because 8,192 different tokens collectively route to every expert).
That is 3,600 FLOPs per byte, against the hardware's 225. Compute-bound by 16x.

Notice the split: 360 TFLOPs linear, 70 TFLOPs quadratic. At 8k context,
attention is only 16% of prefill. Multiply the context by 8 to 65,536 tokens and
the linear part grows 8x to 2,880 TFLOPs while the quadratic part grows 64x to
4,506 TFLOPs. Attention has crossed over and now dominates. This is the entire
content of "long context is expensive," and it is why the 16% number at 8k
misleads people into thinking attention is cheap.

<!-- refutes: M10 -->
If your instinct was "attention is parallelized, so it is fast, so it is linear"
- parallelism changes the wall clock, not the exponent. Doubling context
quadruples the score work regardless of how many cores compute it.

```beat
id: u9-b3
type: self-explain
concept: c-full-trace
prompt: |
  Checkpoint 3, the one that matters most.

  Your prompt is 8,192 tokens. The first token of the reply appears after about
  960 ms. The next 400 tokens arrive at about 5.5 ms each.

  Per token of *work*, prefill handled 8,192 tokens in 960 ms - roughly 0.12 ms
  per token - while decode spends 5.5 ms per token. Prefill is about 47x more
  efficient per token on identical hardware running identical weights.

  Which account explains the 47x? Look for one that says what quantity is
  divided by what, and names the resource limiting each phase.
options:
  - text: "One read of the active weights (22B at 4 bits = 11 GB) is amortized over 8,192 positions in prefill and over a single position in decode. Decode does $2 \\times 22 \\times 10^9 = 44$ GFLOPs against 11 GB, about 4 FLOPs per byte, far under the hardware's 225, so it is memory-bandwidth-bound: 11 GB / 2 TB/s = 5.5 ms. Prefill gets $8192 \\times 4 \\approx 33{,}000$ FLOPs per byte and is compute-bound: 430 TFLOPs / 450 TFLOP/s = 0.96 s."
    correct: true
    explain: "Right. The ratio is arithmetic intensity against the machine's 225 FLOPs-per-byte break-even, and the 47x is the gap between the two limits - which is why a faster GPU with the same bandwidth speeds up prefill and barely touches decode."
  - text: "Prefill runs all 8,192 positions at once on parallel hardware while decode must produce tokens one after another; the 47x is simply the speedup parallel execution gives over sequential execution."
    misconception: M9
    explain: "Parallelism is the enabling condition, not the cause. Both phases run on the same parallel units; what differs is how many tokens share one weight read. Stated this way it also carries the older error of treating decode's sequencing as an execution property rather than a data dependency."
  - text: "Decode is slower because each new token requires re-running the whole 8,192-token prompt through the network again, so it repeats most of prefill's work for every token."
    misconception: M17
    explain: "That is precisely what the KV cache prevents. Prior keys and values are already stored, so a decode step processes one new position; if it re-ran the prompt each token, decode would cost ~960 ms per token, not 5.5 ms."
  - text: "Decode does far more arithmetic per token than prefill does, because generating requires the full depth of the model while prompt processing can shortcut through it, so the 47x is a FLOP-count ratio."
    misconception: U9-M3
    explain: "Both phases run all 64 blocks and decode does *less* arithmetic per step, 44 GFLOPs against prefill's 430 TFLOPs. Latency here is set by bytes crossing the memory bus, not by FLOPs."
check: choice
```

## Stage 4: inside one block - residual stream, heads, and the router

Zoom into block 17 of 64, at position 8,191 (your last token). The block
receives the residual stream vector $x \in \mathbb{R}^{4096}$ for that position,
where 4096 is $d_{model}$.

Normalize, attend, add. RMSNorm rescales $x$ to unit-ish scale; the normalized
copy is what the attention heads read. Thirty-two query heads each project their
own 128-dimensional query from that copy, and the 8 key/value heads (GQA, so
each KV head serves 4 query heads) supply keys and values pulled from *every
position up to 8,191*. Each head produces a similarity-weighted average of value
vectors, the 32 head outputs concatenate back to 4096 dimensions, project once
more, and the result is **added** to $x$. Then normalize again, run the MLP,
add again.

<!-- refutes: M8 -->
You probably think the residual add is there so gradients can flow backward -
the ResNet story, which is what everyone was taught.

The prediction that fails: if residuals existed only for training, deleting a
middle block at inference time would corrupt everything downstream, because the
downstream blocks were trained against that block's output. Delete block 17 at
inference and the model degrades gracefully, often barely measurably. Delete
half the blocks and you get a worse but still coherent model.

What is actually true: the residual stream is a persistent 4096-wide workspace
that runs the entire depth of the network. Each block *reads* from it through
its normalized copy, computes a correction, and *writes* by addition. Blocks
communicate through it the way processes communicate through shared memory, not
the way functions compose. Gradient flow is a real and important consequence, not
the purpose.

Now the MoE part, which happens inside the MLP slot.

<!-- refutes: M12 -->
You probably think the 128 experts are specialists - a code expert, a French
expert - and the router is a classifier that dispatches by topic.

The prediction that fails: if routing were topical, the expert distribution for
your prompt would look nothing like the distribution for a French poem, and the
same expert would fire consistently across a whole code file. Measured routing
is token-granular and largely syntactic: the expert handling the token `auth` in
your prompt also fires on unrelated English tokens, and consecutive tokens in the
same line of code often route to disjoint expert sets. There is no topic
anywhere in the mechanism.

What is actually true: the router is a single linear layer of shape
$4096 \times 128$ producing one score per expert for *this token's current
residual vector*. Top-8 wins, softmax over just those 8 gives mixing weights,
those 8 expert MLPs run, outputs combine, and the combination is added to the
residual stream. During training an auxiliary load-balancing loss punishes the
router for over-using any expert, so the assignment is shaped as much by
utilization pressure as by usefulness. Specialization emerges, is token-level,
and is mostly not human-interpretable.

The consequence for our trace: each token touches 8 of 128 experts, so 22B of
235B parameters are active. But *different* tokens pick *different* experts.
Across 8,192 prompt tokens, essentially all 128 experts in every layer get
selected by somebody, so prefill must have all 235B parameters available and
touches all of them. During decode, one token at a time, only that token's 22B
get read.

```beat
id: u9-b4
type: self-explain
concept: c-full-trace
prompt: |
  Checkpoint 4. The same model, 235B total and 22B active.

  During prefill of your 8,192-token prompt, all 235B parameters get read from
  memory. During decode of each reply token, about 22B get read.

  Yet prefill is compute-bound and decode is memory-bound. Which account
  explains why the phase that reads 10x more weight data is the one that is
  *not* limited by memory - and gets right what MoE buys and costs across the
  two phases?
options:
  - text: "Absolute bytes decide nothing; bytes per unit of arithmetic does. Prefill reads 118 GB once and does 430 TFLOPs with it, ~3,600 FLOPs/byte, above the 225 break-even; decode reads 11 GB and does 44 GFLOPs, 4 FLOPs/byte, far below it. MoE's win lands in decode, where per-token latency tracks the 22B active rather than 235B; its cost is that all 235B must still be resident, since prefill and batched serving touch every expert, plus routing and all-to-all overhead."
    correct: true
    explain: "Right. Arithmetic intensity sets the bound, and MoE is an attack on bytes moved in the bandwidth-bound phase, paid for in footprint set by total parameters."
  - text: "The router sends each token to the experts that match its subject matter, so during decode only the experts relevant to the topic need to be read from memory, while prefill's mixed content forces every specialist to load."
    misconception: M12
    explain: "The router is a $4096 \\times 128$ linear layer scoring this token's residual vector; selection is token-granular and largely syntactic, with load-balancing pressure shaping it. Nothing in the mechanism knows a topic. Decode reads 8 experts because $k = 8$, not because 8 were relevant."
  - text: "MoE makes the model smaller: only 22B parameters exist in any meaningful sense during serving, so both the memory footprint and the read shrink, and decode is memory-bound only because that small model has little arithmetic left to do."
    misconception: M4
    explain: "The model is not smaller - all 235B must be held, and prefill touches all of them. Only the per-token read shrinks. Treating parameters as rows that are present or absent per query is the lookup-table picture."
  - text: "Prefill's 118 GB is read once and then cached in fast on-chip memory, so the bus is idle for the rest of the pass; decode misses that cache every token, which is what makes it bandwidth-bound."
    misconception: U9-M3
    explain: "The weights do not fit on chip in either phase; both stream from HBM. The difference is that prefill's single stream serves 8,192 positions of arithmetic while decode's serves one."
check: choice
```

## Stage 5: the last position becomes a distribution

After block 64, take only the residual vector at position 8,191. The other 8,191
positions' final vectors are discarded for prediction purposes - they were
computed to serve as keys and values for later positions, not because anyone
wanted their outputs. (During training every position's output *is* used; that
is the whole efficiency of the language-modeling objective. At inference, only
the last one matters.)

Final norm, then multiply by the unembedding matrix, shape
$d_{model} \times V = 4096 \times 128000$ (this config reuses the transposed
embedding matrix from Stage 2 - common in smaller models, rarer at frontier
scale). Out comes a vector of 128,000 real numbers: the **logits**.
One per vocabulary entry. They are unbounded, can be negative, and mean nothing
in isolation - only their differences matter.

<!-- refutes: M2 -->
You probably think the softmax of those logits gives the probability that each
token is the *correct* continuation.

The prediction that fails: if those were correctness probabilities, no
temperature knob could change them without new evidence. Temperature changes
them dramatically, and no new information entered the system.

What is actually true: softmax converts scores to a normalized weighting that
parameterizes a sampling distribution. Its calibration target, set by
cross-entropy training, is the *frequency* with which each token followed this
context in the training distribution. Frequency is not truth. That gap is where
hallucination lives.

<!-- fade: logit-to-token -->
Worked, with a 3-token vocabulary so the arithmetic is visible. Suppose the
logits at the final position are

$$z = [2.0,\ 1.0,\ -1.0]$$

for the tokens `The`, `A`, `Zebra`. Sampling at temperature $T = 0.5$:

**Step 1, scale by temperature.** Divide every logit by $T$:
$z/T = [4.0,\ 2.0,\ -2.0]$. Note that $T < 1$ *spreads* the logits apart.

**Step 2, exponentiate.** $e^{4.0} = 54.598$, $e^{2.0} = 7.389$,
$e^{-2.0} = 0.135$.

**Step 3, sum.** $54.598 + 7.389 + 0.135 = 62.122$.

**Step 4, normalize.** $p = [54.598/62.122,\ 7.389/62.122,\ 0.135/62.122]
= [0.879,\ 0.119,\ 0.002]$.

**Step 5, draw** one sample from that distribution.

Run the same logits at $T = 1.0$: $e^{2.0} = 7.389$, $e^{1.0} = 2.718$,
$e^{-1.0} = 0.368$, sum $= 10.475$, so $p = [0.705,\ 0.260,\ 0.035]$. `The` fell
from 0.879 to 0.705; `Zebra` rose from 0.2% to 3.5%.

Then top-p (nucleus) sampling with $p_{top} = 0.9$ applied to the $T = 1.0$
distribution: sort descending $[0.705, 0.260, 0.035]$, accumulate
$0.705 \rightarrow 0.965$. The cumulative mass crosses 0.9 at the second entry,
so the nucleus is $\{$`The`, `A`$\}$ and `Zebra` is deleted outright. Renormalize
over the survivors: $[0.705/0.965,\ 0.260/0.965] = [0.731,\ 0.269]$.

<!-- refutes: M6 -->
The logits $z$ were identical in all three runs. Everything the model knows was
fixed the instant the forward pass ended. Temperature and top-p are
post-processing on a frozen vector. "Raise the temperature to make it more
creative" is a statement about the sampler, not about the model's knowledge, and
$T = 0$ (argmax) does not reveal what the model "really believes" - it reveals
the mode of the same distribution.

```beat
id: u9-b5
type: completion
concept: c-full-trace
prompt: |
  Checkpoint 5. Fill the blanks. Final-position logits over a 3-token
  vocabulary are $z = [3.0,\ 1.0,\ 0.0]$ and the sampler is set to
  $T = 2.0$, top-p $= 0.95$.

  Step 1, scale by temperature: $z/T = $ ____
  Step 2, exponentiate: $[e^{1.5},\ e^{0.5},\ e^{0.0}] = [4.482,\ 1.649,\ 1.000]$
  Step 3, sum: ____
  Step 4, normalize: $p = $ ____
  Step 5, apply top-p $= 0.95$: sorted cumulative mass is
  $0.629 \rightarrow 0.860 \rightarrow 1.000$, so the nucleus is ____ and the
  renormalized distribution is ____
  Step 6, draw one sample.

  Which filling of the blanks is right, and what does it say about the last
  question - did any of these six steps change what the model knows?
options:
  - text: "Step 1: $z/T = [1.5,\\ 0.5,\\ 0.0]$. Step 3: $4.482 + 1.649 + 1.000 = 7.130$. Step 4: $p = [0.629,\\ 0.231,\\ 0.140]$. Step 5: 0.95 is not reached until the third entry, so the nucleus is all three tokens and the distribution is unchanged at $[0.629,\\ 0.231,\\ 0.140]$. No step changed what the model knows - the logits were fixed when the forward pass ended."
    correct: true
    explain: "Right on all four blanks, and right that top-p truncates nothing here: the top two tokens hold only 0.860 of the mass, so the threshold is not met until the third is included."
  - text: "Step 1: $z/T = [1.5,\\ 0.5,\\ 0.0]$. Step 3: $7.130$. Step 4: $p = [0.629,\\ 0.231,\\ 0.140]$. Step 5: the nucleus is the top two tokens, since two entries are enough to pass a 0.95 threshold, so the renormalized distribution is $[0.731,\\ 0.269]$. No step changed what the model knows."
    misconception: M2
    explain: "The arithmetic through step 4 is right, but the threshold was applied to the wrong quantity: cumulative mass over the first two is 0.860, short of 0.95, so the third token survives. Reading the nucleus as 'the entries that look probable enough' rather than a cumulative-mass cut treats the numbers as correctness scores rather than a normalized weighting."
  - text: "Step 1: $z \\times T = [6.0,\\ 2.0,\\ 0.0]$, since $T = 2.0$ raises the scores. Step 3: exponentiating and summing gives $\\approx 411.9$. Step 4: $p \\approx [0.980,\\ 0.018,\\ 0.002]$. Step 5: the nucleus is the first token alone, so the distribution is $[1.000]$. Step 1 changed what the model knows, because raising the temperature made it more creative."
    misconception: M6
    explain: "Temperature divides the logits, so $T = 2.0$ gives $[1.5,\\ 0.5,\\ 0.0]$ and flattens rather than sharpens. And the final claim inverts the point of the checkpoint: $z$ is frozen the instant the forward pass ends; the sampler is post-processing on it."
  - text: "Step 1: $z/T = [1.5,\\ 0.5,\\ 0.0]$. Step 3: $7.130$. Step 4: $p = [0.629,\\ 0.231,\\ 0.140]$. Step 5: the nucleus is all three tokens, distribution unchanged. Step 6 changed what the model knows, because the drawn token is appended and becomes part of what the model has committed to."
    misconception: U9-M2
    explain: "The arithmetic is right but the last sentence is not. Appending the token changes the harness's sequence, not the model - which held no knowledge across the call to begin with. Steps 1-6 are all arithmetic on a frozen vector."
check: choice
```

The sampler picks a token. Say it picks `I`. That token ID is appended to the
sequence, which is now 8,193 long, and it is streamed to your terminal.

## Stage 6: decode, and what the KV cache actually bought

To produce token 8,194 the model needs a forward pass whose final position
attends to all 8,193 prior positions. Naively that means recomputing every
block for every position: another 430 TFLOPs, another second, per token.

It does not, because during prefill every block wrote its key and value
projections for every position into the **KV cache**.

<!-- refutes: M17 -->
You probably think the KV cache is the model's memory of the conversation - the
session state, the thing that makes chat feel stateful.

The prediction that fails: if the cache held information, evicting it and
recomputing from the same tokens would lose something. It does not. Recompute
the cache from the identical token sequence and you get bit-identical keys and
values and bit-identical outputs. A structure whose contents are a pure function
of data you already have is not memory; it is a memo table.

What is actually true: the state of the conversation is the token sequence, held
by the harness. The KV cache is recomputation avoidance, and it is the reason
decode costs 5.5 ms instead of 960 ms.

With the cache, the per-token decode pass does: embed 1 token, and for each
block, project that one token's Q/K/V, append its K and V to the cache, dot its
query against all 8,193 cached keys, weight the cached values, run the MLP for
that one token. Parameter work: $2 \times 22 \times 10^9 \times 1 = 44$ GFLOPs.
Attention work: linear in context length now, not quadratic, because there is
only one query.

<!-- fade: kv-cache-sizing -->
The cache is not free. Size it:

**Per token, per block:** you store one key vector and one value vector for each
of the $h_{kv} = 8$ key/value heads, each $d_{head} = 128$ wide, at 2 bytes
(fp16 - note the cache is usually kept at higher precision than the 4-bit
weights):

$$2 \times h_{kv} \times d_{head} \times \text{bytes} = 2 \times 8 \times 128 \times 2 = 4096 \text{ bytes} = 4\ \text{KB}$$

The leading 2 is "K and V".

**Per token, all blocks:** $4\ \text{KB} \times L = 4 \times 64 = 256\ \text{KB}$.

**For the full 8,192-token context:**
$256\ \text{KB} \times 8192 = 2{,}097{,}152\ \text{KB} = 2\ \text{GiB}$.

Two gigabytes of memory for one conversation, and it grows by 256 KB with every
single token generated. Now put that next to the decode budget: each decode step
reads 11 GB of weights *and* 2 GiB of KV cache, so about 13.1 GB at 2 TB/s =
6.6 ms, not 5.5. At 100,000 tokens of context the cache alone is 26.2 GB, the
read is 37.2 GB, and per-token latency is 18.6 ms. Same model, same prompt
complexity, 2.8x slower per token purely because the cache got bigger.

Note what this makes of GQA. Dropping from 32 KV heads to 8 cut the cache by 4x
with a small quality cost. That is not a modeling decision, it is a
serving-economics decision, and it is why every frontier architecture has some
form of KV sharing.

```beat
id: u9-b6
type: completion
concept: c-full-trace
prompt: |
  Checkpoint 6. Size the KV cache for a different model: $L = 80$ blocks,
  $h_{kv} = 8$ KV heads, $d_{head} = 128$, cache stored in fp16 (2 bytes),
  context of 32,768 tokens.

  Per token per block, bytes = 2 x ____ x ____ x 2 bytes = ____
  Per token, all blocks = ____
  Full context = ____ (give it in GiB, where 1 GiB = 1024^3 bytes)

  Then: this model is dense, not MoE, with 70B parameters at 4 bits. During
  decode at full context, how many bytes does one token's forward pass read, and
  which term dominates?

  Pick the filling that is right at every blank.
options:
  - text: "Per token per block: $2 \\times 8 \\times 128 \\times 2 = 4096$ bytes = 4 KB, the leading 2 being K and V. Per token, all blocks: $4\\ \\text{KB} \\times 80 = 320$ KB. Full context: $320\\ \\text{KB} \\times 32768 = 10$ GiB. Decode reads $70 \\times 10^9 \\times 0.5 = 35$ GB of weights plus ~10.7 GB of cache, ~45.7 GB per token; weights dominate at ~3.3x, but the cache is the term that grows."
    correct: true
    explain: "Right throughout, including the factor of 2 for storing both K and V, and the observation that the weight term is fixed while the cache term keeps climbing with context."
  - text: "Per token per block: $8 \\times 128 \\times 2 = 2048$ bytes = 2 KB. Per token, all blocks: $2\\ \\text{KB} \\times 80 = 160$ KB. Full context: 5 GiB. Decode reads 35 GB of weights plus ~5.4 GB of cache, ~40.4 GB per token, dominated by weights."
    misconception: M17
    explain: "The leading factor of 2 was dropped: every position stores a key vector *and* a value vector, which halves this answer. Losing that factor usually goes with thinking of the cache as one summary per token rather than the two projections a later query needs."
  - text: "Per token per block: $2 \\times 32 \\times 128 \\times 2 = 16384$ bytes = 16 KB, using the 32 query heads. Per token, all blocks: $16\\ \\text{KB} \\times 80 = 1280$ KB. Full context: 40 GiB. Decode reads 35 GB of weights plus ~42.9 GB of cache, ~77.9 GB per token, dominated by the cache."
    misconception: M10
    explain: "Cache size is set by $h_{kv} = 8$, not by the query heads - the whole point of GQA is that several query heads share one stored K/V pair, cutting the cache 4x. Sizing by $h_q$ treats every head's work as needing its own stored state."
  - text: "Per token per block: $2 \\times 8 \\times 128 \\times 2 = 4096$ bytes = 4 KB. Per token, all blocks: 320 KB. Full context: 10 GiB. But the cache is invalidated and rebuilt each step, so decode reads 35 GB of weights plus a full recomputation of the 10 GiB cache every token, and the recompute dominates."
    misconception: M17
    explain: "The sizing is right and the last part is not. Cached entries stay valid because each depends only on an unchanged prefix; decode reads them, appends one new K/V pair, and recomputes nothing."
check: choice
```

Two more decode-stage mechanisms worth placing in the trace. **Quantization**:
the 4-bit weights are why the read is 11 GB and not 44 GB at fp16. Four-bit
costs a few percent on benchmarks, not seven-eighths of the model's knowledge,
because what matters is preserving the direction of the computation, and weights
carry heavy redundancy. **Speculative decoding**: a small draft model proposes 4
tokens, the big model verifies all 4 in a single forward pass. It works
*because* decode is memory-bound - verifying 4 tokens costs almost exactly what
verifying 1 costs, since the weight read is amortized. Speculative decoding
gives you nothing during prefill, which is already compute-bound and has nothing
left to amortize.

## Stage 7: the tool call, the break, and the resume

Forty tokens in, the model emits the token sequence that opens a tool call:
structured text naming the `Read` tool with `file_path` set to
`src/auth/middleware.ts`. It finishes the JSON, and emits the stop token for
tool use.

<!-- refutes: M14 -->
You probably think the model then reads the file.

The prediction that fails: if the model executed tools, the file contents could
arrive *inside* a single generation, mid-stream, with no boundary. Instead there
is always a hard boundary. Generation stops. The GPU goes idle. The harness -
ordinary code, no model involved - parses the emitted JSON, calls `open()`,
reads 3,400 tokens' worth of TypeScript, formats it as a tool-result block, and
appends it to the token sequence. Then, and only then, a *new* forward pass
begins.

What is actually true: tool use is a text protocol. The model's entire
contribution is emitting a string in a format post-training taught it to emit.
The agent is the loop, not the model.

Trace the resume precisely, because the performance consequence is where the
money is. The sequence is now 8,192 (original prompt) + 40 (tokens generated) +
3,400 (file contents, plus the tool-result framing) = 11,632 tokens. The KV
cache already holds entries for the first 8,232 of them, and those entries are
*still valid*, because every one of them was computed from a prefix that has not
changed. Causal masking guarantees this: position 500's key and value depend
only on positions 0 through 500.

So the resume prefills only the 3,400 new tokens. That costs
$2 \times 22 \times 10^9 \times 3400 = 1.5 \times 10^{14}$ FLOPs of parameter
work plus the attention of those 3,400 queries against all 11,632 keys, roughly
0.4 seconds instead of the 1.4 seconds a full re-prefill would cost. This is
exactly what "prompt caching" bills as a cache hit.

And it is why appending is cheap and *editing* is catastrophic. Change one token
at position 40 - swap a line in your `CLAUDE.md`, or let a compaction step
rewrite the middle of the history - and every cached key and value from position
40 onward is invalid, because they were computed from a prefix that no longer
exists. Full re-prefill of 11,592 tokens. This single fact should govern how you
order anything you put in a system prompt: stable content first, volatile
content last.

```beat
id: u9-b7
type: predict
concept: c-full-trace
prompt: |
  Checkpoint 7. Commit to a prediction before reading on.

  A colleague wants to reduce token costs in a long agentic session. Their plan:
  after every tool result arrives, run a small summarizer over the *entire*
  conversation so far and replace it with a compact 2,000-token summary, keeping
  the context small.

  Assume the summarizer is free and its summaries are perfect. What happens to
  (a) tokens billed per turn, (b) wall-clock latency per turn, and (c) the total
  cost of a 30-turn session against plain appending?
options:
  - text: "(a) Tokens per turn drop - the context genuinely is smaller. (b) Latency per turn gets worse: rewriting the history changes tokens near position 0, so every cached key and value from there on is invalid and each turn pays a full prefill of its whole context instead of prefilling only newly appended tokens. (c) Total cost hinges on pricing: under cache-aware pricing, plain appending bills most of its large context at the discounted cached rate while this plan bills a small context at full rate every turn, and can come out more expensive; under flat pricing the plan wins on cost and still loses on latency."
    correct: true
    explain: "Right. The mechanism is prefix immutability: cached entries survive appends and die on edits. Compact rarely, at a stable boundary, and append rather than rewrite."
  - text: "All three improve. A smaller context means fewer tokens billed, fewer tokens to prefill, and a smaller KV cache to read during decode, so the 30-turn session is strictly cheaper and strictly faster than plain appending."
    misconception: M17
    explain: "Smaller is not free when you get it by rewriting. Cached keys and values are a function of the exact preceding tokens; edit position 40 and everything from 40 onward must be recomputed, so each turn pays a full prefill that plain appending avoids."
  - text: "(a) and (c) improve, and (b) is unchanged - the cache survives the rewrite because a perfect summary carries the same information as the text it replaced, so the stored keys and values still represent the same conversation state."
    misconception: M17
    explain: "The cache is indexed by position and derived from token identity, not from meaning. A summary that means the same thing is a different token sequence at those positions, so every entry after the first change is invalid."
  - text: "Latency per turn is roughly unchanged either way, because a turn's time is dominated by the arithmetic of generating the reply, and the reply is the same length under both plans; only the billed token count moves."
    misconception: U9-M3
    explain: "Prefill is a large share of turn latency at these context lengths - about 0.4 s for an incremental 3,400 tokens against ~1.4 s for a full re-prefill - and it is exactly the part this plan destroys. Turn time is not set by generation arithmetic alone."
check: choice
```

The loop continues: more decode, possibly another tool call, another append,
another partial prefill. When the model emits an end-of-turn token instead of a
tool call, the harness stops looping and hands control back to you.

## The cost model that falls out of the trace

Assemble the whole thing into one predictive model. For a session with context
length $n$ tokens and $g$ generated tokens on this box:

$$t_{\text{turn}} \approx \underbrace{\frac{2 P_{a} n_{new} + 4 n_{new} n d_{head} h_q L}{F}}_{\text{prefill, compute-bound}} + \underbrace{g \cdot \frac{B_w + n \cdot b_{kv}}{M}}_{\text{decode, memory-bound}}$$

where $P_a = 22 \times 10^9$ is active params, $n_{new}$ is tokens not already
in the KV cache, $F = 4.5 \times 10^{14}$ FLOP/s, $B_w = 11 \times 10^9$ bytes
of active weights at 4 bits, $b_{kv} = 262144$ bytes of KV per token, and
$M = 2 \times 10^{12}$ bytes/s.

Every knob you have ever touched appears in that expression:

- **Quantization** shrinks $B_w$. Halves decode latency, does nothing for
  prefill.
- **MoE** shrinks $P_a$ and $B_w$ relative to total parameters. Helps both, helps
  decode most.
- **GQA** shrinks $b_{kv}$. Only matters at long context, where it matters
  enormously.
- **Speculative decoding** amortizes $B_w$ across several tokens. Helps decode
  only.
- **Prompt caching** shrinks $n_{new}$ to zero for unchanged prefixes. Helps
  prefill only.
- **Context length** $n$ appears in the prefill term linearly *and* quadratically
  (through the $n_{new} \cdot n$ product) and in the decode term linearly. It is
  the only variable in the expression that hurts you in three places at once.

That last point is the closing insight of the book, so state it plainly: in an
agentic session, every turn appends to $n$, and $n$ appears multiplicatively
against the work of every subsequent turn. A session that does 30 tool calls
does not cost 30 times the first one. It costs something closer to the sum
$\sum_{i} n_i$ where $n_i$ grows monotonically, which is quadratic in the final
context length even when every individual mechanism is well-optimized. Context
is not a container you fill. It is a multiplier you accumulate.

```beat
id: u9-b8
type: self-explain
concept: c-full-trace
prompt: |
  Final checkpoint. Turn 2 of an agentic session generates a token in 6.8 ms.
  Turn 25, on the same model, same hardware, no other load, generates a token in
  18.6 ms. Nothing about the request got harder - it is the same kind of small
  edit.

  Which explanation gives the causal chain from "turn 25" to "18.6 ms", and
  names the change that would most reduce this specific degradation?
options:
  - text: "Every turn appends model output and tool results, so $n$ grows monotonically - roughly 10k tokens by turn 2 and 100k by turn 25. Each decode step reads $B_w + n \\cdot b_{kv}$: 11 GB of weights plus $n \\times 256$ KB of cache, so 13.6 GB at 10k and 37.2 GB at 100k, which at 2 TB/s is 6.8 ms and 18.6 ms. Decode is memory-bandwidth-bound, so latency tracks bytes read. The fix targets $b_{kv}$ - fewer KV heads, KV-cache quantization, windowed or compressed attention - because weight-side tricks only shrink the fixed $B_w$ term."
    correct: true
    explain: "Right. The growing term is $n \\cdot b_{kv}$; it has overtaken the fixed weight term, so only optimizations on bytes of KV per token attack the degradation."
  - text: "Attention is $O(n^2)$ in sequence length, so as the session grows each generated token costs quadratically more arithmetic; at 100k tokens the score matrix is a hundred times larger than at 10k. The fix is a faster accelerator with more FLOP/s, which shortens the dominant $QK^\\top$ computation."
    misconception: M10
    explain: "The quadratic is real in prefill and in the session sum, but per-token decode has exactly one query attending to $n$ keys - linear in $n$. And decode runs its arithmetic units at about 2% utilization, so more FLOP/s buys almost nothing."
  - text: "By turn 25 the KV cache holds far more accumulated conversational state, and the model has to search that richer memory to decide what is relevant, which takes longer than searching the sparse state of turn 2."
    misconception: M17
    explain: "The cache is a memo table of key/value projections, recomputable bit-identically from the tokens; it holds no state to search. Reading it is a fixed bandwidth cost of $n \\times 256$ KB, not a lookup whose difficulty depends on content."
  - text: "Latency tracks the arithmetic the model performs, so the slowdown means turn 25's forward passes are doing more work per token; quantizing the weights further or adding speculative decoding would cut that work and restore 6.8 ms."
    misconception: U9-M3
    explain: "The arithmetic per generated token is identical at both turns - $2 P_a$ FLOPs either way. Quantization and speculative decoding both attack $B_w$, the term that is not growing, so they leave the $n \\cdot b_{kv}$ climb untouched."
check: choice
```

You now have the whole trace: keystroke, harness concatenation, BPE, embedding
gather, one parallel prefill through 64 blocks of normalize-attend-add and
normalize-route-MLP-add, KV cache written along the way, final-position logits,
sampler, one token, repeat with the cache, a tool call, a hard stop, harness
execution, an append, a partial prefill, more tokens.

Every "why is this slow," "why did it hallucinate," "why did it ignore my
instruction," and "why is this bill so large" question you will ever have about
an LLM product resolves to a specific stage in that list. That is what the trace
is for.
