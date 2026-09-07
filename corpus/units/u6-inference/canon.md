---
unit: u6
title: "Inference: making it run"
concepts:
  - c-sampling
  - c-kv-cache
  - c-quant
  - c-specdec
assumes:
  - c-matmul
  - c-dotprod
  - c-logits
  - c-softmax
  - c-sdpa
  - c-attn-numeric
  - c-causal-mask
  - c-mha
  - c-mlp-block
  - c-residual
---

## Where inference sits in the stack

Training is over. The weights are frozen. Every capability the model has, it
already has, sitting in a file. This unit is about the machinery between that
file and a token appearing on your screen, and the whole unit rests on one
claim: **nothing in it creates or destroys knowledge.**

That claim does real work, because four of the most common wrong beliefs about
LLMs are all beliefs that some inference-time knob reaches back and changes what
the model knows. Temperature does not. The KV cache does not. Quantization
mostly does not. Speculative decoding provably does not. Each of those is a
separate section below, and each one is a separate refutation.

Here is the concrete thing to hold onto. The forward pass you built in u3 and u4
ends at a vector of logits: one real number per vocabulary entry, for the next
position. Call it $z \in \mathbb{R}^{V}$, where $V$ is the vocabulary size and
$z_i$ is the unnormalized score for token $i$. Everything in this unit is either
(a) what you do with $z$ after the forward pass produced it, or (b) how to
produce $z$ using less time and memory. Category (a) is sampling. Category (b)
is the KV cache, quantization, and speculative decoding.

### The machine you are reading this on

You are going to compute real numbers about a real system, so here is the
system. The model generating this book is running on the laptop on your tray
table. Its config:

| field | value |
|---|---|
| total parameters | 35.35B |
| active parameters per token | 3.33B (top-8 of 128 experts) |
| layers $L$ | 48 |
| residual width $d_{model}$ | 2048 |
| query heads $H_q$ | 32 |
| key/value heads $H_{kv}$ | 4 (grouped-query attention) |
| head width $d_{head}$ | 128 |
| vocabulary $V$ | 151,936 |
| weight format | 4-bit, group size 64 |
| file on disk | 19.1 GB |
| host | Apple M5, 32 GB unified memory, 153 GB/s |

The mixture-of-experts part (128 experts, top-8) is u7's job. For this unit the
only thing that matters about it is the number 3.33B: that is how many weights
participate in producing each token, and it is the number that goes into every
cost calculation below.

## Temperature, top-k, top-p on one set of logits

You probably think temperature makes the model more creative, and that a high
temperature lets it reach ideas a low temperature cannot.

Here is the prediction that belief makes: if temperature changed what the model
knows or considers, then two forward passes at $T = 0.5$ and $T = 2.0$ over the
same prompt would produce different logits. They do not. They produce the same
logits, bit for bit, on the same hardware. Temperature is not an argument to the
model. In every serving stack you have used, temperature is applied *after* the
forward pass returns, in a few lines of code that never touch a weight. You
could compute the logits once and sample from them at ten different temperatures
without re-running the model, and people do exactly that.

<!-- refutes: M6 -->

The belief is appealing because the behavior change is real and large. Turn
temperature up and the output genuinely gets stranger. It is easy to read that
as the model loosening up. What is actually happening is that a fixed
probability distribution is being reshaped and then sampled from, and the tail
of that distribution is where the strange tokens live. The tail was always
there. Temperature controls how often you visit it.

<!-- fade: temperature-softmax -->

Let us do it on numbers. Take a five-token vocabulary and one real logit vector
produced by a forward pass:

$$z = [3.0,\ 1.0,\ 0.0,\ -1.0,\ -2.0]$$

where $z_i$ is the score for token $i$ and higher means the model scored that
continuation better. These five numbers are the model's entire output. They do
not change for the rest of this section.

Temperature enters by dividing the logits before the softmax:

$$p_i(T) = \frac{e^{z_i / T}}{\sum_{j=1}^{V} e^{z_j / T}}$$

where $T > 0$ is the temperature and $p_i(T)$ is the probability of sampling
token $i$. Note what $T$ divides: the logits, not the probabilities. Dividing
logits by $T$ and then exponentiating is the same as raising each probability to
the power $1/T$ and renormalizing, which is why the effect is multiplicative in
the tails rather than additive.

**$T = 1.0$** (no rescaling). Exponentiate each logit:

$$e^{3.0} = 20.0855,\quad e^{1.0} = 2.7183,\quad e^{0} = 1.0,\quad e^{-1.0} = 0.3679,\quad e^{-2.0} = 0.1353$$

Sum: $24.3070$. Divide each by the sum:

$$p(1.0) = [0.8263,\ 0.1118,\ 0.0411,\ 0.0151,\ 0.0056]$$

**$T = 2.0$**. First divide the logits by 2: $z/T = [1.5, 0.5, 0, -0.5, -1.0]$.
Exponentiate: $[4.4817,\ 1.6487,\ 1.0,\ 0.6065,\ 0.3679]$, sum $8.1048$.

$$p(2.0) = [0.5530,\ 0.2034,\ 0.1234,\ 0.0748,\ 0.0454]$$

**$T = 0.5$**. Divide the logits by 0.5, which doubles them:
$z/T = [6, 2, 0, -2, -4]$. Exponentiate:
$[403.4288,\ 7.3891,\ 1.0,\ 0.1353,\ 0.0183]$, sum $411.9715$.

$$p(0.5) = [0.9793,\ 0.0179,\ 0.0024,\ 0.00033,\ 0.000044]$$

Look at the last token. Its probability went from 0.0056 at $T=1$ to 0.0454 at
$T=2$ (8x more likely) to 0.000044 at $T=0.5$ (126x less likely). Its *logit*
was $-2.0$ in all three cases. The model's assessment of that token never
moved. What moved was how much of the distribution's mass the sampler is willing
to spend on it.

Two limits worth naming. As $T \to 0$, the ratio between the top probability and
every other probability goes to infinity, so $p \to [1, 0, 0, 0, 0]$: pure
argmax, called greedy decoding. As $T \to \infty$, $z_i/T \to 0$ for every $i$,
every exponential goes to 1, and $p \to$ uniform: the model's scores are erased
entirely. Neither limit adds information. One discards all of the distribution
except its mode; the other discards all of it.

```beat
id: u6-b1
type: predict
concept: c-sampling
prompt: |
  You send the identical prompt to your local model twice. First call sets
  `temperature=0.2`, second sets `temperature=1.4`.

  Commit before reading on. Consider every tensor inside the forward pass -
  embeddings, attention scores, attention outputs, MLP activations, the residual
  stream, the final logits $z$. Which of these accounts is right about what
  differs between the two calls, and about where they first diverge?
options:
  - text: |
      None of them differ. Embeddings, attention scores, attention outputs, MLP
      activations, the residual stream and the final logit vector are
      bit-identical across the two calls. The two calls first diverge strictly
      after the forward pass returns, where the logits are divided by $T$ and
      softmaxed into a sampling distribution.
    correct: true
    explain: |
      Right. Temperature is never an argument to the network - it is applied by a
      few lines of serving code to the logits the pass already returned. That is
      why you can compute $z$ once and sample it at ten temperatures without
      re-running the model, which people do.
  - text: |
      Everything up to the last layer is identical, but the final logits differ:
      temperature is folded into the output projection, so the $T = 1.4$ call
      returns a flatter logit vector.
    misconception: M6
    explain: |
      This is the belief that temperature reaches into the model. The logits are
      produced by frozen weights from the token sequence alone; the division by
      $T$ happens afterwards, outside the pass. $z$ is bit-identical at $T = 0.2$
      and $T = 1.4$.
  - text: |
      From the attention scores onward everything differs. The two calls produce
      visibly different text, so some tensor inside the forward pass must have
      differed to produce it.
    misconception: M6
    explain: |
      The inference runs backwards. The outputs differ because one fixed
      distribution was reshaped and sampled twice at different sharpness. Turning
      the knob up does not loosen the model; it moves mass into a tail that was
      always there.
  - text: |
      Nothing differs, the softmax output included - those probabilities are the
      model's estimate that each token is the right continuation, and only new
      evidence could move them. The two calls diverge only at the random draw.
    misconception: M2
    explain: |
      Softmax output is not a probability of being correct; it is a normalized
      exponential weighting of scores, and $T$ rescales the scores before that
      normalization. On $z = [3.0, 1.0, 0.0, -1.0, -2.0]$ the last token carries
      0.0056 at $T = 1$ and 0.0454 at $T = 2$ - genuinely different
      distributions, not one distribution drawn from differently.
check: choice
```

### top-k and top-p truncate the tail instead of rescaling it

Temperature reshapes the whole distribution. The other two knobs delete part of
it and renormalize what is left.

**top-k** keeps the $k$ highest-probability tokens, zeroes the rest, and
renormalizes so the survivors sum to 1. On $p(1.0)$ with $k=2$: keep 0.8263 and
0.1118, whose sum is 0.9382. Renormalize:

$$0.8263 / 0.9382 = 0.8808, \qquad 0.1118 / 0.9382 = 0.1192$$

The other three tokens now have probability exactly zero. They are unreachable,
not unlikely.

**top-p** (nucleus sampling) keeps the smallest set of top tokens whose
cumulative probability reaches $p$. It adapts: where the distribution is peaked
it keeps few tokens, where the distribution is flat it keeps many. The number of
survivors is data-dependent, which is the whole point.

The interaction with temperature is the part people get wrong, so watch the
cumulative sums for $p = 0.9$ across our three temperatures.

At $T = 0.5$, cumulative: $0.9793$. That first token alone clears 0.9.
**Survivors: 1.** top-p has silently become greedy decoding.

At $T = 1.0$, cumulative: $0.8263,\ 0.9382$. Two tokens to clear 0.9.
**Survivors: 2.**

At $T = 2.0$, cumulative: $0.5530,\ 0.7564,\ 0.8798,\ 0.9546$. Four tokens.
**Survivors: 4.** Renormalizing over those four:
$[0.5793,\ 0.2131,\ 0.1293,\ 0.0784]$.

So `top_p=0.9` is not a fixed setting. Its effect ranges from "no-op that
disables all sampling" to "keep most of the vocabulary" depending on the
temperature it is stacked with, and on how confident the model happens to be at
this particular position. This is why `temperature=0.3, top_p=0.9` and
`temperature=1.2, top_p=0.9` are not two points on one dial - they are different
samplers.

Order of operations matters and is worth checking in whatever stack you use. The
common convention (HuggingFace, vLLM, llama.cpp) is: logits, then temperature
divide, then top-k truncation, then top-p truncation, then softmax and sample.
Applying top-p before temperature gives different survivor sets, and a few older
implementations did exactly that.

```beat
id: u6-b2
type: completion
concept: c-sampling
prompt: |
  Same logit vector, new temperature. One of the fillings below completes all
  four blanks correctly.

  $$z = [3.0,\ 1.0,\ 0.0,\ -1.0,\ -2.0], \qquad T = 4.0, \qquad \text{top-}p = 0.9$$

  Step 1, rescale the logits by dividing by T:
      z/T = [0.75, 0.25, 0.0, -0.25, -0.5]

  Step 2, exponentiate each entry:
      [2.1170, 1.2840, 1.0000, 0.7788, 0.6065]

  Step 3, sum them:
      Z = ____(a)____

  Step 4, divide each exponential by Z to get p(4.0):
      p = [0.3659, ____(b)____, 0.1728, 0.1346, 0.1048]

  Step 5, accumulate p from the largest entry down until the running total
  first reaches 0.9. How many tokens survive top-p = 0.9?
      survivors = ____(c)____

  Step 6, one sentence: state which of the model's five logits changed between
  the T = 0.5 computation earlier in this section and this T = 4.0 computation.
      ____(d)____
options:
  - text: |
      (a) 5.7863   (b) 0.2219   (c) 5
      (d) None of them: the logits are [3.0, 1.0, 0.0, -1.0, -2.0] in both
          computations, and only the divisor applied to them after the forward
          pass changed.
    correct: true
    explain: |
      Right. Cumulative sums are 0.3659, 0.5878, 0.7606, 0.8952, 1.0000 - four
      tokens give 0.8952, which has not reached 0.9, so the fifth is required and
      top-p truncates nothing at all. That is the mirror image of $T = 0.5$,
      where the same setting kept one token and became greedy decoding.
  - text: |
      (a) 5.7863   (b) 0.2219   (c) 4
      (d) None of them: only the divisor applied after the forward pass changed.
    misconception: U6-2
    explain: |
      0.8952 has not reached 0.9. The cut is a mechanical inequality on
      cumulative mass, not an eyeball - treating it as approximate is the same
      instinct that treats top-p as a soft creativity dial. The fifth token is
      required, so survivors = 5.
  - text: |
      (a) 5.7863   (b) 0.2219   (c) 5
      (d) All five changed: at T = 4.0 the pass produced the logit vector
          [0.75, 0.25, 0.0, -0.25, -0.5], which is flatter than at T = 0.5.
    misconception: M6
    explain: |
      $[0.75, 0.25, 0.0, -0.25, -0.5]$ is $z/T$, computed by the sampler after
      the forward pass returned. The model's logits are $[3.0, 1.0, 0.0, -1.0,
      -2.0]$ in both computations; temperature is not an input to the network.
  - text: |
      (a) 5.7863   (b) 0.2219   (c) 2, since top_p = 0.9 kept two tokens at
          T = 1.0 and it is the same setting here.
      (d) None of them changed.
    misconception: U6-2
    explain: |
      top_p = 0.9 is not a fixed number of survivors. On these same logits it
      keeps 1, 2, 4 and 5 tokens at $T = 0.5, 1.0, 2.0, 4.0$: the cut point reads
      the cumulative mass of a distribution temperature has already reshaped.
check: choice
```

Across the four temperatures worked in this section, `top_p=0.9` keeps 1, 2, 4,
and 5 of the five tokens at $T = 0.5, 1.0, 2.0, 4.0$. Same parameter, same
logits, and it ranges from disabling sampling entirely to doing nothing at all.

## T=0 is not truth mode

You probably think that $T = 0$ gives you the model's real answer, and that
higher temperatures add noise on top of it. Set temperature to zero, get the
truth; raise it, get creative deviation from the truth.

Here is the prediction that fails. If $T = 0$ revealed truth, then greedy
decoding would not hallucinate, and hallucination rate would rise monotonically
with temperature from a floor of zero. Measure it: greedy decoding hallucinates
at close to the same rate as $T = 0.7$, and on some tasks greedy is *worse*,
because it locks into a confident wrong opening and then conditions every
subsequent token on that opening. There is no temperature at which the failure
mode disappears, because the failure mode is not in the sampler.

<!-- refutes: M6 -->
<!-- refutes: M2 -->

Why the belief is appealing: $T=0$ is deterministic, and we are trained to
associate determinism with correctness. Reproducible output feels authoritative.
And there is a grain of truth in it - $T = 0$ does show you the argmax, which is
the single token the model scored highest, so it does tell you something the
model "thinks." What it does not tell you is whether that thought is right.

What is actually true: $T = 0$ is argmax over the same distribution every other
temperature samples from. It is one point on the sampling dial, not an escape
from it. And the distribution it takes the argmax of is not a distribution over
truth. Recall from u2 that softmax outputs are a normalized exponential
weighting of scores; in the output layer they parameterize a next-token sampling
distribution calibrated to training-data frequencies. A model that saw a false
claim stated confidently ten thousand times will put high probability on it, and
$T = 0$ will return it every time with perfect reproducibility.

Two practical consequences that follow directly:

**Determinism is not accuracy.** $T=0$ buys you reproducibility, which is
genuinely useful for tests, diffs, and debugging. It buys you nothing on
correctness.

**Greedy is not even the best-scoring sequence.** Argmax at each position does
not maximize the probability of the sequence. Picking the locally best token can
force a low-probability continuation later; a slightly worse token now can open
a much better path. This is exactly why beam search exists, and why it is not
free. If you want to see this clearly: a two-step toy where step 1 offers
$p = [0.6, 0.4]$ and the 0.6 branch leads only to $p = [0.5, 0.5]$ while the 0.4
branch leads to $p = [0.99, 0.01]$ gives greedy a sequence probability of
$0.6 \times 0.5 = 0.30$ against the other branch's $0.4 \times 0.99 = 0.396$.
Greedy is a local heuristic.

```beat
id: u6-b3
type: self-explain
concept: c-sampling
prompt: |
  A colleague reports a bug: "the model hallucinated an API method that does not
  exist. I set temperature to 0 and it still hallucinates the same method, every
  single run. Something is broken in our serving stack - at T=0 it should give
  the real answer."

  Which explanation would you give them? Pick the one that says what T = 0 does
  mechanically, what it guarantees, and where the hallucination comes from.
options:
  - text: |
      Their expectation assumes temperature reaches back into the model and
      selects for correctness. It divides the logits after the forward pass has
      finished, so the logits at T = 0 are the same logits as at T = 1; if the
      top-scoring token starts a hallucinated method name, argmax returns it
      every time. T = 0 guarantees determinism only - the identical wrong answer
      each run is evidence the stack works. The error is upstream, in the logits
      and therefore the weights.
    correct: true
    explain: |
      Right, and the ordering matters: the sampler has no correctness signal to
      consult, so nothing downstream of the logits could have caught this. The
      reproducibility they are reading as a symptom is the one thing T = 0 does
      buy.
  - text: |
      T = 0 is the model's knowledge probe - it shows what the model really
      knows, stripped of sampling noise. Getting the same nonexistent method on
      every run therefore means the weights or the serving stack are corrupted,
      and the next step is to re-verify the checkpoint hash.
    misconception: M6
    explain: |
      T = 0 is argmax over the same distribution every other temperature samples
      from - one point on the sampling dial, not an escape from it. Greedy
      decoding hallucinates at close to the rate of T = 0.7 and sometimes worse,
      because it locks into a confident wrong opening and conditions on it.
  - text: |
      The softmax put the highest probability on that method, which means the
      model assessed it as most likely to be true. The stack is fine and the
      weights are fine; the distribution is miscalibrated, so the fix belongs at
      the sampler - rescale or threshold the probabilities before argmax.
    misconception: M2
    explain: |
      Softmax output in the output layer parameterizes a next-token sampling
      distribution calibrated to training-data frequencies, not a probability of
      truth. A false claim seen ten thousand times gets high probability. No
      rescaling at the sampler can add a correctness signal that the logits never
      carried.
  - text: |
      Greedy decoding returns the single most likely sequence, so the model is
      locked onto it. The correct method name lies on a lower-probability path
      that argmax cannot reach - raise the temperature, or use beam search, to
      let the model find it.
    misconception: U6-5
    explain: |
      Two errors. Greedy does not maximize sequence probability: with step 1 at
      $p = [0.6, 0.4]$ where the 0.6 branch continues at best 0.5 and the 0.4
      branch at 0.99, greedy scores 0.30 against 0.396. And raising $T$ changes
      which wrong token you get, not whether it is wrong, because the logits it
      samples are unchanged.
check: choice
```

## The KV cache is not memory

You probably think of the KV cache as the model's memory of the conversation:
the thing that lets it remember what you said three turns ago, its session
state. Drop the cache and the model forgets.

Here is the prediction that fails, and it is testable in one line of your
serving config. If the KV cache held semantic state accumulated over the
conversation, then discarding it and recomputing it from the same token sequence
would lose that state - the model would behave as though something had been
forgotten. Do it. Evict the cache mid-conversation, force a full recompute over
the identical tokens, and the resulting logits are bit-identical to what you
would have gotten. Not similar. Identical. A cache whose contents can be
regenerated exactly from the inputs, at any time, contains no information that
the inputs did not already contain. It is a memoization table.

<!-- refutes: M17 -->

The belief is appealing because the word "cache" and the word "memory" sit next
to each other in every engineer's head, because the cache genuinely does persist
across tokens, and because chat genuinely does feel stateful. The statefulness
is real - it is located somewhere else. The state is the token sequence.
The harness re-sends the entire conversation as tokens on every request (u8
covers the assembly). The cache is an optimization over reprocessing those
tokens, nothing more.

### What is actually in it

Recall the attention computation from u3. At layer $\ell$, for a sequence of $n$
tokens, you project the residual stream $X \in \mathbb{R}^{n \times d_{model}}$
into queries, keys, and values:

$$Q = X W_Q, \qquad K = X W_K, \qquad V = X W_V$$

where $W_K, W_V \in \mathbb{R}^{d_{model} \times (H_{kv} \cdot d_{head})}$ are
frozen weight matrices, and row $i$ of $K$ is the key vector for token $i$.

Now generate token $n+1$. Its query attends over all $n+1$ keys. But the keys
for tokens $1 \dots n$ are functions of those tokens' residual streams only -
and because of the causal mask, token $i$'s residual stream at layer $\ell$
depends only on tokens $1 \dots i$. Adding token $n+1$ to the end cannot change
any of them. **Every key and value vector for every earlier token is the exact
same vector it was on the previous step.** So you store them and stop
recomputing them.

That is the entire idea. Not memory: the observation that the causal mask makes
earlier positions' K and V immutable, so recomputing them $n$ times is $n-1$
wasted computations.

Note what is *not* cached: queries (each new token needs only its own, and it is
never reused), attention outputs, MLP activations, the residual stream. Those
are all discarded after each step. If the cache were memory, throwing away the
residual stream - the thing u4 called the model's shared workspace - would be an
odd choice.

```beat
id: u6-b4
type: predict
concept: c-kv-cache
prompt: |
  You are 40 turns into a conversation, 18,000 tokens of context. Your serving
  process is killed and restarted. The harness resends the identical 18,000
  tokens and generation continues.

  Commit before reading on. Which account is right about (a) the next-token
  logit vector after the restart, (b) what the restart cost, and (c) the second
  scenario in which the harness instead compacts the history into a 4,000-token
  summary and sends that?
options:
  - text: |
      (a) Identical, exactly - same weights, same tokens, same causal mask,
      therefore the same K and V at every position. (b) Time and arithmetic only:
      one prefill pass over 18,000 tokens instead of reading a table already in
      memory, so latency to the next token rises. (c) The summary does change the
      output, because the token sequence changed, not because the cache did.
    correct: true
    explain: |
      Right, and the contrast is the whole point. A cache whose contents can be
      regenerated exactly from the inputs holds no information the inputs did not
      already hold. The state is the token sequence; compaction is lossy because
      it rewrites that sequence.
  - text: |
      (a) Similar but not identical - 40 turns of accumulated state were in the
      cache and a cold recompute cannot reconstruct all of it, so the model comes
      back slightly flatter. (b) It cost conversational continuity. (c) The
      summary loses more of the same accumulated state.
    misconception: M17
    explain: |
      Evict the cache mid-conversation and force a full recompute over the
      identical tokens: the logits come back bit-identical, not similar. K and V
      for token $i$ are functions of tokens $1 \dots i$ alone under the causal
      mask, so nothing accumulates that the tokens do not already determine.
  - text: |
      (a) Identical. (b) One prefill pass over 18,000 tokens, so latency only.
      (c) The summary is worse specifically because compaction discards the KV
      cache that 40 turns of conversation built up - a fresh 4,000-token cache
      cannot hold what the 18,000-token one did.
    misconception: M17
    explain: |
      Getting (a) right and (c) wrong is the common half-held version. The
      summary degrades the output because different tokens go in, and it would
      degrade it identically on a cold process with no cache at all. The cache is
      a memoization table over the sequence, never a second store of it.
  - text: |
      (a) Identical. (b) Essentially nothing: cached tokens are free, so
      refilling the cache is a memory copy and decode speed does not depend on
      how long the context is. (c) The summary buys back RAM and nothing else.
    misconception: U6-3
    explain: |
      The cache removes re-projection of old tokens, not attention over them. The
      restart pays a real prefill over 18,000 positions, and afterwards every
      decoded token attends across the whole cache: on this model attention
      arithmetic overtakes all the weight matmuls at about 8,500 tokens of
      context, so a shorter sequence is cheaper per token as well as in memory.
check: choice
```

### Deriving what it costs

<!-- fade: kv-cache-sizing -->

Now the part that turns "context is expensive" from a vibe into a number.

Per token, per layer, you store one key vector and one value vector for each
key/value head. Count the scalars:

$$\text{scalars per token} = \underbrace{2}_{K \text{ and } V} \times L \times H_{kv} \times d_{head}$$

where $L$ is the number of layers, $H_{kv}$ is the number of key/value heads,
and $d_{head}$ is the width of each head. Multiply by the bytes per scalar,
$b$ - 2 for fp16, the usual choice, since the cache is bandwidth-bound and
rarely needs more:

$$\boxed{\text{bytes} = 2 \cdot L \cdot H_{kv} \cdot d_{head} \cdot b \cdot n}$$

with $n$ the sequence length in tokens. Note what is absent: $d_{model}$,
$H_q$, the MLP width, the number of experts, the total parameter count. The KV
cache does not care how big your model is. It cares about $L$, $H_{kv}$, and
$d_{head}$.

For the model on your tray table: $L = 48$, $H_{kv} = 4$, $d_{head} = 128$,
$b = 2$.

$$2 \times 48 \times 4 \times 128 \times 2 = 98{,}304 \text{ bytes per token} = 96 \text{ KiB per token}$$

Ninety-six kilobytes of RAM for every single token in your context, and it is
linear in $n$, so:

- 2,048-token context: 192 MiB
- 32,768-token context: exactly 3.0 GiB
- 131,072-token context: exactly 12.0 GiB

At 128k context, the cache is 12 GiB against a 19.1 GB weight file. **The
scratch space is nearly as large as the model.** That is why long context costs
what it costs on the memory axis, and it is why this book's 32 GB host runs the
model at 32k context rather than the 262k the architecture supports.

Now the counterfactual that explains grouped-query attention. Suppose this model
used full multi-head attention, $H_{kv} = H_q = 32$ instead of 4:

$$2 \times 48 \times 32 \times 128 \times 2 = 786{,}432 \text{ bytes per token} = 768 \text{ KiB per token}$$

Eight times larger, exactly the ratio $H_q / H_{kv} = 32/4$. At 128k context
that is 96 GiB of KV cache for a model whose weights also need 19 GB. No
laptop holds that. GQA - 32 query heads sharing 4 key/value heads, as
introduced in u4 - is not a quality optimization. It is the thing that makes
long context fit in memory at all, and it is why every recent model has
$H_{kv} \ll H_q$.

```beat
id: u6-b5
type: completion
concept: c-kv-cache
prompt: |
  Size the KV cache for a different model: L = 64 layers, d_model = 8192,
  H_q = 64 query heads, H_kv = 8 key/value heads, d_head = 128, fp16 cache
  (b = 2 bytes), n = 8192 tokens of context. One filling below is correct
  throughout.

  Step 1, write the formula:
      bytes = 2 * L * H_kv * d_head * b * n

  Step 2, identify which of the given numbers is NOT used by the formula, and
  say in one clause why it is irrelevant:
      unused: ____(a)____  because ____(b)____

  Step 3, bytes per token:
      2 * 64 * 8 * 128 * 2 = ____(c)____ bytes = ____(d)____ KiB

  Step 4, total at n = 8192:
      ____(d)____ KiB * 8192 = ____(e)____ MiB = ____(f)____ GiB

  Step 5, this model has 2x the H_kv of the 35B model in this section but only
  1.33x the layers. Its per-token cache is how many times larger?
      ____(g)____
options:
  - text: |
      (a) d_model = 8192 (and H_q = 64).
      (b) because the cache stores only K and V, which are projected to
          H_kv * d_head dimensions - d_model is the input width to that
          projection and does not appear in the output size.
      (c) 262,144 bytes   (d) 256 KiB   (e) 2048 MiB   (f) 2.0 GiB
      (g) 256 / 96 = 2.667x
    correct: true
    explain: |
      Right, and (a)-(b) are the transferable part: the cache does not care how
      big the model is, only about $L$, $H_{kv}$ and $d_{head}$. $H_q$ sets how
      many queries read the cache, not how large it is.
  - text: |
      (a) Nothing is unused.
      (b) because d_model enters through the residual stream that each layer
          stores per token.
      (c) 2 * 64 * 8192 * 2 = 2,097,152 bytes   (d) 2048 KiB
      (e) 16,384 MiB   (f) 16.0 GiB   (g) 2048 / 96 = 21.3x
    misconception: M17
    explain: |
      That sizes the residual stream at every layer - which is exactly the thing
      that is not cached. Queries, attention outputs, MLP activations and the
      residual stream are all discarded after each step. Only K and V persist,
      and they are $H_{kv} \cdot d_{head} = 1024$ wide, not 8192.
  - text: |
      (a) d_model = 8192.
      (b) because the cache is indexed by heads, and there are 64 query heads per
          layer, so 64 key/value pairs must be stored for each.
      (c) 2 * 64 * 64 * 128 * 2 = 2,097,152 bytes   (d) 2048 KiB
      (e) 16,384 MiB   (f) 16.0 GiB   (g) 2048 / 96 = 21.3x
    misconception: U6-3
    explain: |
      Sizing by $H_q$ is off by exactly $H_q / H_{kv} = 8$, and that factor is
      the entire point of grouped-query attention: 64 query heads share 8
      key/value heads. Many queries read one cached K/V pair; reading it again
      does not store it again.
  - text: |
      (a) d_model = 8192.
      (b) because only the keys need storing - the values can be regenerated from
          the residual stream when they are needed.
      (c) 64 * 8 * 128 * 2 = 131,072 bytes   (d) 128 KiB
      (e) 1024 MiB   (f) 1.0 GiB   (g) 128 / 96 = 1.333x
    misconception: M17
    explain: |
      The residual stream is thrown away after each step, so there is nothing
      left to regenerate V from. The leading 2 in the formula counts K and V, and
      both are immutable for earlier positions under the causal mask - which is
      precisely why both are worth caching.
check: choice
```

## Prefill and decode are different machines

Two phases, two completely different cost profiles, and conflating them is the
source of most wrong intuitions about inference speed.

**Prefill** processes your entire prompt. All $n$ prompt tokens go through the
network in one forward pass, all positions computed simultaneously, K and V
written into the cache for every position at once. This reinforces the point
from u3: within a forward pass there is no left-to-right anything. Every
position's attention is computed in parallel; the causal mask is what makes
position $i$ ignore positions $> i$, and a mask is applied to a matrix that was
already fully computed.

**Decode** generates one token per forward pass. Each pass processes exactly one
position, reads the cache for all previous positions, appends its own K and V,
and produces one logit vector. To generate 500 tokens you run 500 forward
passes, and they are genuinely serial: pass $t+1$ needs the token that pass $t$
sampled. No parallelism available within a single sequence.

So: a 2,000-token prompt is one forward pass; 500 generated tokens are 500
forward passes. Prefill does *more* arithmetic - it processes 4x as many
positions - and finishes in a fraction of the wall-clock time. If you have ever
watched a long prompt get swallowed instantly and then watched tokens dribble
out, you have watched this.

The reason is arithmetic intensity, which is a systems concept you already own.
Prefill multiplies a $2000 \times d$ activation matrix by each weight matrix:
the weights get read once from memory and reused across 2,000 rows. It is
compute-bound, and the GPU or NPU runs near peak FLOPs. Decode multiplies a
$1 \times d$ vector by each weight matrix: every weight is read from memory and
used exactly once. It is memory-bandwidth-bound, and the arithmetic units idle
while waiting on RAM.

Put the numbers on your machine. Decode reads the active weights per token:
3.33B parameters at 4.3125 bits each is 1.80 GB moved per token. At 153 GB/s:

$$153 / 1.80 = 85 \text{ tokens/sec ceiling}$$

That is a hard ceiling from bandwidth alone, before any compute. Had this been a
dense 35B model reading all 19.1 GB per token, the same ceiling would be
$153 / 19.1 = 8$ tokens/sec. That gap - 85 against 8 - is the entire reason
the book is served by an MoE, and u7 will unpack why the router makes it
possible.

### Where the quadratic hides

<!-- refutes: M10 -->

You probably think attention is cheap because it is parallel. That is the trap:
parallelism is a wall-clock property, and asymptotic cost is not. Attention is
$O(n^2)$ in sequence length because every query dots every key. Parallel
hardware hides it at small $n$ and stops hiding it at large $n$.

Here is where it becomes visible on your machine, and the crossover is closer
than most people expect.

Decoding one token at position $n$ requires, per layer per query head, one dot
product of the query against each of $n$ cached keys ($n \cdot d_{head}$
multiply-adds) and one weighted sum over $n$ cached value vectors (another
$n \cdot d_{head}$). Counting a multiply-add as 2 FLOPs:

$$\text{attention FLOPs per decoded token} = 4 \cdot n \cdot d_{head} \cdot H_q \cdot L$$

For this model: $4 \times 128 \times 32 \times 48 = 786{,}432$ FLOPs *per token
of context*. Meanwhile the weight matmuls cost roughly $2 \times$ active
parameters $= 2 \times 3.33 \times 10^9 = 6.66$ GFLOPs per token, and that
number does not depend on $n$ at all.

Set them equal:

$$n^* = \frac{6.66 \times 10^9}{786{,}432} \approx 8{,}469 \text{ tokens}$$

Below ~8.5k of context, generating a token is dominated by pushing activations
through weights. Above it, attention over the cache dominates, and it keeps
growing linearly per token, which means quadratically over a generation. At 32k
context, attention costs 25.8 GFLOPs per decoded token against 6.66 for the
weights - **3.9x more arithmetic spent on attention than on the entire rest of
the network.** At 128k it is 103 GFLOPs, 15x the weights.

That is the honest answer to "why does long context cost so much." Two separate
taxes, and it helps to keep them apart: memory grows *linearly* (96 KiB per
token, the cache), and per-token compute grows *linearly in $n$*, which makes
the cost of generating a whole sequence *quadratic*. Doubling your context
doubles your cache and roughly doubles the attention arithmetic for every single
token you then generate.

```beat
id: u6-b6
type: compute
concept: c-kv-cache
prompt: |
  Same model: L = 48 layers, H_q = 32 query heads, d_head = 128, active
  parameters 3.33e9.

  Attention arithmetic per decoded token is 4 * n * d_head * H_q * L FLOPs,
  where n is the current context length. Weight-matmul arithmetic per decoded
  token is 2 * (active parameters) FLOPs, independent of n.

  At what context length n do the two become equal? Answer in tokens, as a
  bare number rounded to the nearest whole token.
answer: 8469
rubric: |
  Expected work: 4 * 128 * 32 * 48 = 786432 FLOPs per token of context;
  2 * 3.33e9 = 6.66e9; 6.66e9 / 786432 = 8468.6.
  Common wrong answers to watch for in variants: using 35e9 total parameters
  instead of 3.33e9 active gives ~89,000, which is the MoE-cost error; dropping
  the factor of 4 (counting only scores, not the value sum, or forgetting
  multiply-add = 2 FLOPs) gives ~33,900 or ~16,900.
check: numeric(60)
```

## Quantization: what four bits hold

You probably think 4-bit quantization throws away seven eighths of the model's
knowledge - that it is lossy compression applied to a database of facts, and
that the model comes out correspondingly dumber.

Here is the prediction that fails, and it fails badly. If knowledge scaled with
bits, then a 4-bit 35B model would hold about the same knowledge as an 8-bit
17.5B model, or an fp16 8.75B model - same total bits, so same content. Measure
it. The 4-bit 35B beats the fp16 9B on essentially every benchmark, by a wide
margin, and it lands within a couple of points of its own fp16 self. Bit-count
accounting predicts the wrong ordering, not merely the wrong magnitude. The
model on your tray table is the evidence: 19.1 GB, four bits a weight, and it is
writing this.

<!-- refutes: M16 -->

The belief is appealing because bit-counting is usually sound. Engineers have
correct intuitions about lossy compression of images and audio, where throwing
away bits does throw away content, and about databases, where a row is either
stored or not. That intuition imports cleanly only if parameters are a
lookup table of facts - which u1 and u4 already refuted (M4). Parameters define
a *function*. The question is not "how many bits per fact survive," it is "how
much does the function move when you perturb its parameters."

### What is actually stored

<!-- fade: affine-quantization -->

4-bit quantization is affine quantization applied per small group of weights.
Take a group of $G$ consecutive weights (real implementations use $G = 32$,
64, or 128). Store:

- one scale $s$, in fp16
- one zero-point $z$, an integer in $[0, 15]$
- $G$ four-bit codes $q_i \in \{0, 1, \dots, 15\}$

and reconstruct each weight as

$$\hat{w}_i = s \cdot (q_i - z)$$

where $s$ sets the spacing of the 16 representable values and $z$ shifts the
grid so it straddles zero. The encoder picks

$$s = \frac{\max(w) - \min(w)}{15}, \qquad z = \text{round}\!\left(\frac{-\min(w)}{s}\right), \qquad q_i = \text{clamp}\!\left(\text{round}\!\left(\frac{w_i}{s} + z\right),\, 0,\, 15\right)$$

so the 16 levels span exactly the range present in that group.

Work one group of $G = 8$ by hand (real groups are larger; eight fits on a
page). Weights from a real MLP row, rounded to three decimals:

$$w = [0.021,\ -0.043,\ 0.115,\ -0.008,\ 0.067,\ -0.091,\ 0.032,\ 0.004]$$

$\max(w) = 0.115$, $\min(w) = -0.091$, so the range is $0.206$ and

$$s = 0.206 / 15 = 0.013733$$

$$z = \text{round}(0.091 / 0.013733) = \text{round}(6.626) = 7$$

Now $w_i / s$ for each weight:
$[1.529,\ -3.131,\ 8.374,\ -0.583,\ 4.879,\ -6.626,\ 2.330,\ 0.291]$.
Add $z = 7$ and round:

$$q = [9,\ 4,\ 15,\ 6,\ 12,\ 0,\ 9,\ 7]$$

Reconstruct with $\hat{w}_i = 0.013733 (q_i - 7)$:

$$\hat{w} = [0.0275,\ -0.0412,\ 0.1099,\ -0.0137,\ 0.0687,\ -0.0961,\ 0.0275,\ 0.0000]$$

Errors: $[+0.0065,\ +0.0018,\ -0.0051,\ -0.0057,\ +0.0017,\ -0.0051,\ -0.0045,\ -0.0040]$.
Every one is below $s/2 = 0.0069$, as it must be - rounding to a grid of spacing
$s$ cannot miss by more than half a step. The error standard deviation here is
about 7% of the weights' own standard deviation.

Two things to notice, because they are the whole argument.

**The magnitudes are preserved, the fine structure is not.** The largest weight
is still the largest; the sign of every weight survives; the ordering survives
except where two weights fell in the same bucket (0.021 and 0.032 both became
code 9). Direction is kept, precision is spent.

**No fact went anywhere.** There is no row that got dropped. Every weight is
still present, still participating, moved by less than half a quantization step.
The function the network computes is perturbed, not truncated.

```beat
id: u6-b7
type: completion
concept: c-quant
prompt: |
  Quantize this group of 8 weights to 4 bits with the affine scheme
  (16 levels, per-group scale and zero-point). One filling below is correct
  throughout.

      w = [0.10, -0.05, 0.20, 0.00, -0.10, 0.15, 0.05, -0.02]

  Step 1, group statistics:
      max(w) = 0.20,  min(w) = -0.10,  range = 0.30

  Step 2, scale (15 intervals between 16 levels):
      s = 0.30 / 15 = ____(a)____

  Step 3, zero-point:
      z = round( -min(w) / s ) = round( 0.10 / ____(a)____ ) = ____(b)____

  Step 4, codes q_i = clamp(round(w_i / s + z), 0, 15):
      w_i / s   = [5, -2.5, 10, 0, -5, 7.5, 2.5, -1]
      + z       = [10, 2.5, 15, 5, 0, 12.5, 7.5, 4]
      q         = [10, ____(c)____, 15, 5, 0, ____(d)____, ____(e)____, 4]
                  (round half away from zero)

  Step 5, dequantize w_hat_i = s * (q_i - z). Give the reconstruction of the
  FIRST weight, w_1 = 0.10:
      w_hat_1 = ____(f)____

  Step 6, state the bound that every reconstruction error must satisfy, and
  give its numeric value for this group:
      |w_hat_i - w_i| <= ____(g)____
options:
  - text: |
      (a) 0.02   (b) 5   (c) 3   (d) 13   (e) 8
      (f) w_hat_1 = 0.02 * (10 - 5) = 0.10, exact - this weight landed on a grid
          point.
      (g) s/2 = 0.01, because rounding to a grid of spacing s cannot miss by more
          than half a step, and no weight here lies outside [min, max] so nothing
          is clamped.
    correct: true
    explain: |
      Right, and (g) is what makes the argument quantitative: every weight is
      still present and moved by at most half a step. Nothing was deleted; the
      function was perturbed.
  - text: |
      (a) 0.02   (b) 5   (c) 3   (d) 13   (e) 8   (f) 0.10
      (g) s = 0.02, because a weight that falls between two levels can end up a
          full grid step from where it started.
    misconception: M16
    explain: |
      Rounding always goes to the nearer level, so the miss is at most $s/2 =
      0.01$ - check the worked group earlier in this section, where every error
      is under 0.0069. Doubling the assumed error is the bit-counting instinct
      again: 4-bit is roughly a 9% perturbation of each weight, not a whole step
      and not an 87.5% deletion.
  - text: |
      (a) 0.02   (b) 5   (c) 3   (d) 13   (e) 8
      (f) w_hat_1 = 0.00, because 0.10 is not one of the 16 values the group can
          store, so that weight is dropped.
      (g) unbounded - the dropped weights are the ones that lose everything.
    misconception: M4
    explain: |
      Nothing is dropped. $\hat{w}_1 = s(q_1 - z) = 0.02 \times (10 - 5) = 0.10$
      exactly, and every one of the eight weights survives within $s/2$ of its
      original value. "Stored or not stored" is the lookup-table model of
      parameters; affine quantization moves each weight to a nearby grid point
      instead.
  - text: |
      (a) 0.02   (b) 5   (c) 3   (d) 13   (e) 8
      (f) w_hat_1 = s * q_1 = 0.02 * 10 = 0.20
      (g) 0.0896, the 4-bit error figure from the table in this section.
    misconception: M16
    explain: |
      Two slips with one root. The zero-point is what lets the 16 levels straddle
      zero, so the reconstruction is $s(q_i - z) = 0.10$, not $s q_i$. And 8.96%
      is an error standard deviation relative to the weights' own standard
      deviation over a Gaussian group - a statistic about how little the function
      moves, not the per-weight bound, which is $s/2 = 0.01$.
check: choice
```

### Sizing the file on your disk

The overhead is not free, and it is the reason the file is 19.1 GB rather than
17.7 GB. Per group of $G = 64$ weights you store one fp16 scale (16 bits) and
one 4-bit zero-point (4 bits): 20 extra bits amortized over 64 weights.

$$\text{bits per weight} = 4 + \frac{20}{64} = 4 + 0.3125 = 4.3125$$

```beat
id: u6-b8
type: compute
concept: c-quant
prompt: |
  Your local model file: 35.35e9 parameters, 4-bit affine quantization with
  group size 64, one fp16 scale (16 bits) and one 4-bit zero-point per group.

  Compute the file size in gigabytes, where 1 GB = 1e9 bytes. Answer as a bare
  number rounded to two decimal places.
answer: 19.06
rubric: |
  Expected work: overhead = 20 bits / 64 weights = 0.3125 bits per weight;
  total = 4.3125 bits per weight; 35.35e9 * 4.3125 / 8 = 1.9056e10 bytes
  = 19.06 GB.
  Diagnostic wrong answers: 17.68 means the learner ignored the scale/zero-point
  overhead entirely; 17.74 means they computed in GiB rather than GB;
  70.70 means they used fp16 (the un-quantized size) - check whether they
  divided by 8 to convert bits to bytes.
check: numeric(0.05)
```

Against the fp16 original at 70.7 GB, that is a 3.7x reduction. On this book's
32 GB host it is the difference between a model that cannot load at all and one
that runs with room left over for the KV cache. Which is the actual reason 4-bit matters on your tray table -
not that it saves disk, but that it moves 3.7x fewer bytes per decoded token
across a memory bus that is the binding constraint.

## Where quantization falls off the cliff

"It mostly works" is not an explanation, and the failure cases are where the
real understanding is. So: why does it work, and where does it stop?

### Why it works

Quantization error behaves like additive noise on the weights, and the network
is robust to noise on its weights up to a threshold. Three reasons, in
increasing order of importance:

**The error is small in relative terms, not 87.5%.** Bit-counting says four bits
of sixteen keeps 25% of the "information." What actually matters is the relative
error the reconstruction induces. Simulate a group of 64 weights drawn from a
Gaussian - a good model of a trained weight matrix's entries - and measure the
error standard deviation as a fraction of the weights' own standard deviation:

| bits | grid step | error std (as fraction of weight std) |
|---|---|---|
| 8 | $0.018\,\sigma$ | 0.53% |
| 6 | $0.074\,\sigma$ | 2.13% |
| 4 | $0.313\,\sigma$ | 8.96% |
| 3 | $0.670\,\sigma$ | 19.2% |
| 2 | $1.563\,\sigma$ | 45.0% |

Four bits is a 9% perturbation of each weight, not an 87.5% deletion. Halving
the bit width does not halve the surviving content; it doubles the noise floor.

**Trained networks are noise-robust by construction.** Every weight in that file
was found by SGD on stochastic minibatches, with dropout in some layers, in
finite precision. The optimizer converges to wide, flat basins (u2's point about
loss surfaces, and u5's about why SGD works anyway) precisely because narrow
sharp minima do not survive the noise of training. A solution that survived
training noise survives a 9% weight perturbation for the same structural reason.

**Downstream normalization eats scale error.** Each block writes an additive
update into the residual stream and the next block's RMSNorm rescales what it
reads. A perturbation that changes the magnitude of a block's output more than
its direction gets partly normalized away before it can propagate. What
quantization must preserve is direction, and direction is exactly what an
error uniformly distributed over $\pm s/2$ across thousands of dimensions
perturbs least.

Put those together and the correction to M16 is precise: what a quantized model
preserves is the *geometry of the computation*, not the precision of any
individual weight. Redundancy across weights means many different weight vectors
compute nearly the same function, and quantization moves you to a nearby member
of that set.

### Where it stops

The degradation curve is not linear. It is flat, then it is a cliff, and three
things cause the cliff.

**Activation outliers.** From roughly 6B parameters up, trained transformers
develop a small number of feature dimensions carrying activations 10-100x larger
than the rest, concentrated in specific residual-stream channels. Those
dimensions dominate the range of any group they land in, which stretches $s$ and
pushes every other weight in that group into a handful of codes. This is why
naive round-to-nearest 4-bit degrades noticeably while modern quantizers do
not: AWQ rescales the outlier channels' range into the activations before
rounding, and GPTQ compensates each column's rounding error by adjusting the
not-yet-quantized weights. Both respect the outlier structure. The cliff is not really about bit
width; it is about whether the quantizer respects the outlier structure.

**Group size.** A single scale shared across 1,024 weights must span the widest
weight in all 1,024. Shrinking to $G = 64$ costs 0.3125 bits per weight and buys
a much tighter range per group. Group size is the knob people forget: a
"4-bit" model with $G = 256$ and one with $G = 32$ are meaningfully different
models.

**Which tensors you quantize.** Embeddings, the output head, LayerNorm/RMSNorm
gains, and the router weights in an MoE (u7) are usually kept at higher precision
in a good 4-bit build. They are a small fraction of the parameters and a large
fraction of the sensitivity - the router in particular, where a small logit
perturbation flips which experts fire, which is a discrete change, not a smooth
one. Quantizing the router aggressively is one of the few things that produces
the failure people *expect* from quantization.

Below 4 bits the flat region runs out. At 3 bits (19% error) you see measurable
losses on reasoning-heavy and long-context tasks first, because those compound
errors over many steps. At 2 bits (45% error) naive methods collapse entirely,
and the methods that work at all (QuIP#, AQLM) stop being round-to-nearest and
start doing vector quantization with learned codebooks - a different algorithm,
not a smaller $s$.

One prediction worth internalizing, because it tells you what to expect when you
do hit the cliff: quantization damage shows up first in long multi-step outputs
and last in short factual recall. That is the exact opposite of what the
"compressed database" model predicts. If quantization deleted facts, recall
would go first. It does not. Chains of reasoning go first, because that is where
small perturbations compound.

## Speculative decoding: exact, not approximate

The last technique in this unit is the one that sounds like it must be a
tradeoff and is not.

The setup is the bandwidth bound from earlier. Decoding one token requires
reading 1.80 GB of active weights, and produces exactly one token. The
arithmetic units are almost entirely idle. If you fed the same forward pass five
positions instead of one, it would take barely longer - the weights get read
once either way. Decode wastes the machine, and it wastes it in a very specific
way: you have spare compute and no tokens to spend it on, because you do not
know token $t+1$ until you have sampled token $t$.

Speculative decoding buys tokens to spend it on. A small, cheap **draft model** -
typically 10-30x cheaper, sometimes a few layers of the target model itself -
generates $k$ candidate tokens autoregressively. Then the large **target model**
runs *one* forward pass over all $k$ candidates at once, in parallel, exactly as
in prefill, producing its own distribution at each of the $k$ positions. A
verification rule then decides how many of the drafted tokens to keep.

The system-level claim: you traded a bandwidth-bound problem for a
compute-bound one, which is the direction you want to trade on hardware where
bandwidth is the constraint.

### Why it is exact

The obvious verification rule - "keep the drafted token if the target model's
argmax agrees" - is not what is used, and understanding why is the point of this
section. That rule would bias the output toward the draft model's preferences
whenever the target's distribution is close to flat.

The actual rule is modified rejection sampling, and it produces a sample from
the target distribution *exactly*. Let $p$ be the target model's distribution at
some position and $q$ the draft model's distribution at that same position, and
let $x$ be the token the draft sampled. Then:

1. Accept $x$ with probability $\min\left(1, \frac{p(x)}{q(x)}\right)$.
2. If rejected, sample a replacement from the normalized residual
   $\dfrac{\max(0,\ p - q)}{\sum_i \max(0,\ p_i - q_i)}$, and discard all
   remaining drafted tokens.

Verify it on three tokens. Target $p = [0.5,\ 0.3,\ 0.2]$, draft
$q = [0.6,\ 0.2,\ 0.2]$. The draft over-weights token 1 and under-weights
token 2.

Acceptance probabilities: $\min(1, 0.5/0.6) = 0.8333$ for token 1;
$\min(1, 0.3/0.2) = 1$ for token 2; $\min(1, 0.2/0.2) = 1$ for token 3.

Probability each token is drafted *and* accepted is $q(x)$ times its acceptance
probability:

$$\text{token 1}: 0.6 \times 0.8333 = 0.50, \qquad \text{token 2}: 0.2 \times 1 = 0.20, \qquad \text{token 3}: 0.2 \times 1 = 0.20$$

Total accepted mass: $0.90$, so rejection happens with probability $0.10$, and
all of it comes from token 1 being drafted and refused.

The residual: $\max(0, p - q) = [0,\ 0.1,\ 0]$, which normalizes to
$[0,\ 1,\ 0]$. On rejection you emit token 2 with certainty.

Final emission probabilities:

- token 1: $0.50 + 0.10 \times 0 = 0.50$
- token 2: $0.20 + 0.10 \times 1 = 0.30$
- token 3: $0.20 + 0.10 \times 0 = 0.20$

$[0.5,\ 0.3,\ 0.2]$, which is $p$. Exactly. The draft model's bias toward token 1
was corrected precisely by the rejection rule, and the residual distribution
delivered exactly the mass token 2 was owed.

This is a guarantee, not a benchmark result. A speculative decoder emits samples
from the same distribution as the target model alone, for any draft model
whatsoever. A bad draft model makes it *slow* - acceptance rate drops, more
tokens get thrown away - and never makes it wrong. That is the property that
makes speculative decoding an implementation detail rather than a model change,
and it is why a provider can turn it on without telling you.

One consequence people find surprising: the draft model does not need to be
good, or trained on the same data, or even competent. Its quality shows up only
in the acceptance rate. Correctness is unconditional.

### What it buys

Draft $k$ tokens, accept them independently with probability $\alpha$ (the
acceptance rate) until the first rejection. Because a rejection consumes the
rejected slot and produces a corrected token there, the expected number of
tokens emitted per target forward pass is

$$E[\text{tokens}] = \frac{1 - \alpha^{k+1}}{1 - \alpha}$$

The $k+1$ accounts for the bonus token: if all $k$ drafts are accepted, the
target's own forward pass already computed a distribution for position $k+1$,
so you sample one more for free.

```beat
id: u6-b9
type: compute
concept: c-specdec
prompt: |
  Your local setup drafts k = 4 tokens per step with a small draft model, and
  the measured acceptance rate is alpha = 0.8.

  Expected tokens emitted per target-model forward pass is
  ( 1 - alpha^(k+1) ) / ( 1 - alpha ).

  Compute it. Answer as a bare number to two decimal places.
answer: 3.36
rubric: |
  Expected work: 0.8^5 = 0.32768; (1 - 0.32768) / 0.2 = 3.3616.
  Diagnostic wrong answers: 3.36 is correct; 2.95 means the learner used
  alpha^k instead of alpha^(k+1) and dropped the bonus token; 4.0 means they
  answered "k accepted tokens on average" without the geometric series;
  5.0 means they computed 1/(1-alpha) and ignored the k cap.
check: numeric(0.02)
```

3.36 tokens per target pass against 1 without speculation. The cost is not
3.36x speedup, though - each step also runs the draft model 4 times. If the
draft is 20x cheaper, the step costs $1 + 4/20 = 1.2$ target-equivalents, so the
speedup is $3.3616 / 1.2 = 2.8\times$. That is a real 2.8x, on the same weights,
producing the same distribution.

Two failure modes worth knowing. Acceptance rate is heavily content-dependent:
boilerplate, formatting, and code with rigid syntax draft at $\alpha > 0.9$;
dense reasoning drafts at $\alpha \approx 0.5$, where $k=4$ yields only
1.94 tokens per pass and the draft overhead can make it a net loss. And it only
helps when you are bandwidth-bound. Under heavy batched serving the machine is
already compute-saturated with other users' tokens, there is no idle arithmetic
to reclaim, and speculative decoding is turned off. It is a latency optimization
for the single-user case - which is exactly the case on your tray table.

## What this unit did and did not change

Four techniques, one shared property. Run through them against the file on disk:

Sampling turns a fixed logit vector into a token. It changes which token you
get, never what the logits were.

The KV cache stores key and value projections so old tokens are not reprocessed.
Delete it and you get identical output, more slowly.

Quantization stores each weight as one of 16 values per group. It perturbs the
function by about 9% per weight and preserves its geometry, which is why the
model on your tray table performs like a 35B model and not like a 9B one.

Speculative decoding produces samples from the target distribution exactly, with
a mathematical guarantee that holds for any draft model.

Not one of them adds a capability, and not one of them removes a fact. The
knowledge was fixed when training ended (u5). Everything here is about the cost
of running the function you already have - which is precisely why it is worth
knowing exactly what each knob does, because the temptation to reach for
temperature when the real problem is upstream in the weights is the most
expensive mistake in this unit.
