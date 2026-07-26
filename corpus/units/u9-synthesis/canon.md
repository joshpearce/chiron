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

  Do not answer "it followed the more recent instruction." Answer mechanically:
  where does each of those two instructions physically live at the moment of the
  forward pass, what makes one of them win, and what would it take for the model
  to be *unable* to see the first one?
answer: |
  Both live in the same flat token sequence, at different positions - the
  CLAUDE.md text early (low position indices), the user override late (high
  indices). There is no priority field distinguishing them; the only differences
  available to the model are position, surrounding delimiter tokens (system vs
  user framing), and phrasing. "Recency wins" is a learned behavioral regularity
  from post-training over such conflicts, implemented through attention weights
  at the final position favoring nearer, more imperative-looking tokens - not a
  rule the harness enforces.

  The model becomes unable to see the CLAUDE.md instruction only if those tokens
  are not in the stream: the harness dropped them, or compaction summarized them
  away, or they fell outside the context window. Nothing else can hide them.
rubric: |
  Must contain: (1) both instructions are tokens in one sequence, distinguished
  only by position and delimiter tokens; (2) precedence is a learned behavior
  from post-training, not enforced structure; (3) the only way to remove an
  instruction's influence is to remove its tokens (harness truncation,
  compaction, window overflow).
  Pass = (1) plus either (2) or (3).
  Fail patterns: "the system prompt has higher priority in the API" = M13,
  the structure-is-native error. "The model remembers the newer one" = M17-style
  confusion of tokens with memory. Answering only "recency bias" with no
  mechanism = fail, that is a restatement of the observation.
check: llm
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
answer: "524.288"
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
generating a token. Generation on this box runs at about 5.5 ms per token. That
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
across 64 blocks that is $7.04 \times 10^{13}$, or 70 TFLOPs.

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

  Explain the 47x. Not "prefill is parallel" - say what quantity is being
  divided by what, and name the resource each phase is limited by.
answer: |
  Both phases read the same weights from memory; prefill amortizes that read
  across 8,192 tokens while decode amortizes it across one.

  Decode: to produce one token the accelerator must stream the active weights
  (22B params at 4 bits = 11 GB) from memory, and does 2 x 22e9 = 44 GFLOPs of
  arithmetic with them. That is 4 FLOPs per byte read. The hardware can do 225
  FLOPs per byte. So the compute units idle ~98% of the time waiting on memory:
  decode is memory-bandwidth-bound. 11 GB / 2 TB/s = 5.5 ms, which is the entire
  observed per-token latency.

  Prefill: the same weight read services 8,192 tokens at once, so the ratio is
  8,192 x 4 = ~33,000 FLOPs per byte, far above 225. The bottleneck moves to the
  arithmetic units: 430 TFLOPs / 450 TFLOP/s = 0.96 s. Compute-bound.

  The 47x is the ratio of the two limits, which is why buying a faster GPU with
  the same memory bandwidth speeds up prefill and does almost nothing for
  decode.
rubric: |
  Must contain: (1) decode reads the full set of active weights per single
  token, prefill reads them once for the whole batch of positions; (2) the
  explicit naming of decode as memory-bandwidth-bound and prefill as
  compute-bound; (3) some form of the arithmetic-intensity argument (FLOPs per
  byte, or "the matmul becomes a matrix-vector product with no reuse").
  Pass = (1) and (2). Full credit = all three.
  Fail patterns: "prefill is parallel and decode is sequential" with nothing
  further = fail, that is the observation restated, and it is the M9 residue
  (treating parallelism as the cause rather than the enabling condition).
  "Decode is slow because it recomputes the whole prompt each time" = M17,
  the KV cache exists precisely to prevent that.
check: llm
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

  Yet prefill is compute-bound and decode is memory-bound. Explain why the phase
  that reads 10x more weight data is the one that is *not* limited by memory.
  Then state what MoE buys you and what it costs you, in terms of these two
  phases.
answer: |
  Because "bytes read" is the wrong quantity on its own - what matters is bytes
  read per unit of arithmetic. Prefill reads 118 GB once and does 430 TFLOPs
  with it (~3,600 FLOPs/byte). Decode reads 11 GB and does 44 GFLOPs with it
  (4 FLOPs/byte). The hardware's break-even is 225 FLOPs/byte, so prefill sits
  far above it and decode far below.

  What MoE buys: decode reads only the k=8 selected experts per token, so
  per-token decode latency tracks 22B active parameters, not 235B. A dense 235B
  model would read ~118 GB per token and decode ~10x slower. MoE gives you the
  capability of a large parameter count at the memory-bandwidth cost of a small
  one, in the phase that is bandwidth-bound.

  What it costs: all 235B must be resident (or paged) because prefill and
  batched serving touch every expert, so VRAM/HBM footprint is set by total
  params, not active. Plus routing overhead, load-imbalance stalls, and
  all-to-all communication when experts are sharded across devices.
rubric: |
  Must contain: (1) arithmetic intensity, not absolute bytes, decides the
  bound - prefill amortizes its read over thousands of tokens; (2) MoE's win is
  specifically in decode, where per-token bandwidth tracks active params; (3) the
  cost is memory footprint set by total params (and/or routing/comms overhead).
  Pass = (1) and (2).
  Fail patterns: "experts are specialists so only the relevant ones load" = M12.
  "MoE makes the model smaller" = the total-vs-active confusion; the model is
  not smaller, only the per-token read is.
check: llm
```

## Stage 5: the last position becomes a distribution

After block 64, take only the residual vector at position 8,191. The other 8,191
positions' final vectors are discarded for prediction purposes - they were
computed to serve as keys and values for later positions, not because anyone
wanted their outputs. (During training every position's output *is* used; that
is the whole efficiency of the language-modeling objective. At inference, only
the last one matters.)

Final norm, then multiply by the unembedding matrix, shape
$d_{model} \times V = 4096 \times 128000$ (often the transposed embedding
matrix from Stage 2). Out comes a vector of 128,000 real numbers: the **logits**.
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
# variants: blank steps 2 and 4 (exp / normalize) for a learner who missed the
# softmax mechanics; blank steps 1 and 5 for one who missed the
# temperature-is-post-processing point; blank 3 only as a warmup.
prompt: |
  Checkpoint 5. Fill the blanks. Final-position logits over a 3-token
  vocabulary are $z = [3.0,\ 1.0,\ 0.0]$ and the sampler is set to
  $T = 2.0$, top-p $= 0.95$.

  Step 1, scale by temperature: $z/T = $ ____
  Step 2, exponentiate: $[e^{1.5},\ e^{0.5},\ e^{0.0}] = [4.482,\ 1.649,\ 1.000]$
  Step 3, sum: ____
  Step 4, normalize: $p = $ ____
  Step 5, apply top-p $= 0.95$: sorted cumulative mass is
  $0.626 \rightarrow 0.856 \rightarrow 1.000$, so the nucleus is ____ and the
  renormalized distribution is ____
  Step 6, draw one sample.

  Then answer in one sentence: which of these six steps, if any, changed what
  the model knows?
answer: |
  Step 1: $z/T = [1.5,\ 0.5,\ 0.0]$
  Step 3: $4.482 + 1.649 + 1.000 = 7.131$
  Step 4: $p = [4.482/7.131,\ 1.649/7.131,\ 1.000/7.131] = [0.629,\ 0.231,\ 0.140]$
  Step 5: cumulative $0.629 \rightarrow 0.860 \rightarrow 1.000$; 0.95 is not
  reached until the third token, so all three tokens stay in the nucleus and the
  distribution is unchanged: $[0.629,\ 0.231,\ 0.140]$.
  Step 6: sample.

  None of them. The logits were fixed when the forward pass ended; steps 1-6 are
  arithmetic on a frozen vector.
rubric: |
  Step 1 must show division by T (not multiplication) giving [1.5, 0.5, 0.0].
  Step 3 must be 7.131 +/- 0.01. Step 4 must be [0.629, 0.231, 0.140] +/- 0.01.
  Step 5 must conclude that top-p = 0.95 truncates nothing here, because the
  first two tokens hold only 0.860 of the mass - a learner who drops the third
  token has applied the threshold to the wrong quantity.
  The final sentence must say no step changed the model's knowledge. Answering
  "temperature made it more/less creative, so step 1 did" = M6, fail regardless
  of arithmetic.
  Pass = correct steps 1, 3, 4 plus the correct final sentence.
check: llm
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
reads 11 GB of weights *and* 2 GB of KV cache, so 13 GB at 2 TB/s = 6.5 ms, not
5.5. At 100,000 tokens of context the cache alone is 25.6 GB, the read is 36.6
GB, and per-token latency is 18 ms. Same model, same prompt complexity, 3x
slower per token purely because the cache got bigger.

Note what this makes of GQA. Dropping from 32 KV heads to 8 cut the cache by 4x
with a small quality cost. That is not a modeling decision, it is a
serving-economics decision, and it is why every frontier architecture has some
form of KV sharing.

```beat
id: u9-b6
type: completion
concept: c-full-trace
# variants: blank the "x 2 for K and V" and the "x L" steps for a learner who
# under-sizes caches; blank the final multiply for one who has the per-token
# figure but does not connect it to context length.
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
answer: |
  Per token per block: 2 x 8 x 128 x 2 = 4096 bytes = 4 KB.
  Per token, all blocks: 4 KB x 80 = 320 KB.
  Full context: 320 KB x 32,768 = 10,485,760 KB = 10,240 MiB = 10 GiB.

  Decode read: 70e9 params x 0.5 bytes = 35 GB of weights, plus 10 GiB
  (~10.7 GB) of KV cache = ~45.7 GB per token. Weights dominate at roughly
  3.3x the cache, but the cache is no longer a rounding error - it is adding
  ~30% to per-token latency, and it is the term that keeps growing while the
  weights term is fixed.
rubric: |
  Per-token-per-block must be 4096 bytes with the leading factor of 2 explicitly
  attributed to storing both K and V. Per-token must be 320 KB. Full context
  must be 10 GiB (accept 10.7 GB decimal if labeled).
  The final part must (a) compute weights as 35 GB from 4-bit x 70B, and
  (b) identify weights as dominant while noting the cache is the growing term.
  Pass = all three cache figures correct plus (a).
  Fail patterns: omitting the factor of 2 for K and V (gives 5 GiB) = the most
  common sizing error. Multiplying by $h_q$ instead of $h_{kv}$ = has not
  internalized why GQA exists. Claiming the cache must be recomputed each token
  = M17.
check: llm
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
  Checkpoint 7. Predict, before reading on.

  A colleague wants to reduce token costs in a long agentic session. Their plan:
  after every tool result arrives, run a small summarizer over the *entire*
  conversation so far and replace it with a compact 2,000-token summary, keeping
  the context small.

  Assume the summarizer is free and its summaries are perfect. Predict what
  happens to (a) tokens billed per turn, (b) wall-clock latency per turn, and
  (c) the total cost of a 30-turn session versus plain appending. Give the
  mechanism for each, not just the direction.
answer: |
  (a) Tokens billed per turn drop - the context really is smaller, so the
  per-turn input token count is ~2,000 + new material instead of a growing
  history. This is the part the colleague is right about.

  (b) Wall-clock latency per turn gets *worse*, potentially much worse. Rewriting
  the history changes tokens at position ~0, which invalidates the entire KV
  cache. Every turn now pays a full prefill of its whole context from scratch
  instead of prefilling only the newly appended tokens. Plain appending prefills
  ~3,400 new tokens per turn; this plan prefills ~2,000+ every turn with zero
  cache reuse.

  (c) Total cost depends entirely on whether the provider prices cached input
  tokens cheaply. Under cache-aware pricing (cached reads typically ~10% of
  fresh input), plain appending bills nearly all of its large context at the
  discounted rate, while the summarize-every-turn plan bills a smaller context
  at full rate every single turn - and can easily come out more expensive
  despite processing fewer tokens. Under flat per-token pricing the plan wins on
  cost and still loses on latency.

  The correct version of the idea: compact rarely and only at a stable
  boundary, never per-turn, and always append rather than rewrite when possible.
rubric: |
  Must contain: (1) the recognition that rewriting history invalidates the KV
  cache from the first changed position onward, so cache reuse goes to zero;
  (2) the resulting latency regression - full prefill every turn instead of
  incremental prefill; (3) the cost answer conditioned on cache-aware pricing
  rather than asserted unconditionally.
  Pass = (1) and (2). Full credit adds (3).
  Fail patterns: "smaller context is always cheaper and faster" = has not
  connected caching to prefix immutability. "The model will forget things" -
  true but not the asked mechanism, and the prompt stipulated perfect summaries.
  Claiming the cache survives because the summary "means the same thing" = M17,
  treating the cache as semantic rather than positional/token-derived.
check: llm
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
  Final checkpoint. Turn 2 of an agentic session generates a token in 5.6 ms.
  Turn 25, on the same model, same hardware, no other load, generates a token in
  16 ms. Nothing about the request got harder - it is the same kind of small
  edit.

  Give the full causal chain from "turn 25" to "16 ms." Then state the one
  architectural change that would most reduce this specific degradation, and say
  which term in the cost model it touches.
answer: |
  Chain: each turn appends the model's output plus a tool result to the token
  sequence, so context length n grows monotonically - by turn 25 it might be
  ~100k tokens instead of ~10k. Every generated token's forward pass must attend
  to all n prior positions, which means reading the entire KV cache: n x 256 KB.
  At n = 10k that is ~2.6 GB on top of 11 GB of weights (13.6 GB, ~6.8 ms); at
  n = 100k it is ~25.6 GB on top of 11 GB (36.6 GB, ~18 ms). Decode is
  memory-bandwidth-bound, so per-token latency tracks bytes read almost exactly,
  and bytes read now has a term linear in n that has overtaken the fixed weight
  term.

  Most effective architectural change: reduce bytes of KV per token - fewer KV
  heads (GQA/MQA), KV cache quantization to 8-bit or 4-bit, or an architecture
  with sliding-window or latent-compressed attention. That touches b_kv in the
  decode term, the only term that scales with n during decode. Note that
  speculative decoding and weight quantization do NOT help here: they shrink or
  amortize B_w, the fixed term, while the growing term is untouched.
rubric: |
  Must contain: (1) context grows every turn because tool results and model
  output are appended - it is accumulation, not the task getting harder;
  (2) per-token decode reads the whole KV cache, which is linear in n, so decode
  bytes-read = B_w + n x b_kv; (3) decode is memory-bound so latency tracks that
  sum; (4) the fix targets b_kv (KV quantization, fewer KV heads, windowed or
  compressed attention) rather than the weight term.
  Pass = (1), (2), and (3). Full credit adds (4) with the explicit observation
  that weight-side optimizations do not address a growing KV term.
  Fail patterns: "attention is O(n^2) so decode gets quadratically slower" -
  per-token decode attention is linear in n; the quadratic shows up in prefill
  and in the sum across the session. Accepting this without correction rewards
  a real confusion. "The model is thinking about more things" = M17 plus
  anthropomorphism.
check: llm
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
