---
unit: u5
depth: deeper-math
---

## Cross-entropy: turning a distribution into one number

Canon asserts that cross-entropy is the right loss. Here is why it is not a
choice at all: it is maximum likelihood, written down.

Let the model define $p_\theta(t \mid c)$, the probability of token $t$ given
context $c$, parameterized by weights $\theta$. Given a corpus of $n$ tokens
$t_1, \dots, t_n$, the probability the model assigns to the whole corpus
factorizes exactly by the chain rule of probability:

$$p_\theta(t_1, \dots, t_n) = \prod_{k=1}^{n} p_\theta(t_k \mid t_{<k})$$

Maximum likelihood says: pick $\theta$ maximizing that product. Products of
$10^{13}$ numbers below 1 underflow instantly, so take logs, which is monotone
and therefore preserves the argmax, and negate to get a minimization:

$$-\log p_\theta(t_1,\dots,t_n) = \sum_{k=1}^{n} -\log p_\theta(t_k \mid t_{<k})$$

That sum is the cross-entropy loss, unnormalized. Dividing by $n$ gives the
per-token mean reported in practice. So "sum the per-token cross-entropies" and
"maximize the joint probability of the corpus" are the same statement.

**The information-theoretic reading.** For a general target distribution $q$
and model distribution $p$, cross-entropy is

$$H(q, p) = -\sum_i q_i \log p_i = H(q) + D_{\mathrm{KL}}(q \,\|\, p)$$

where $H(q) = -\sum_i q_i \log q_i$ is the entropy of the target and
$D_{\mathrm{KL}}(q\|p) = \sum_i q_i \log(q_i/p_i)$ is the Kullback-Leibler
divergence. Verify the identity by expanding the KL term:

$$H(q) + D_{\mathrm{KL}}(q\|p) = -\sum_i q_i \log q_i + \sum_i q_i \log q_i - \sum_i q_i \log p_i = -\sum_i q_i \log p_i$$

Since $H(q)$ does not depend on $\theta$, minimizing cross-entropy over $\theta$
is *exactly* minimizing $D_{\mathrm{KL}}(q \| p_\theta)$. The loss is a distance
between distributions with an additive constant bolted on, and that constant is
the entropy floor of the next section.

Note the KL is asymmetric and we minimize $D_{\mathrm{KL}}(q\|p)$, the
"forward" direction, which is **mode-covering**: it is infinitely penalized for
assigning $p_i = 0$ where $q_i > 0$, so the model must spread mass over
everything the data ever does. The reverse direction $D_{\mathrm{KL}}(p\|q)$ is
mode-seeking and would let the model ignore rare continuations. Maximum
likelihood hands you the mode-covering one, which is a large part of why base
models are so willing to produce unlikely continuations. It is also why the KL
term in RLHF, which is the reverse direction under sampling from $\pi_\theta$,
behaves differently from the pretraining objective.

## Perplexity, and the floor that is not zero

The floor claim deserves a proof, not an assertion. Fix a true conditional
distribution $q$ over next tokens. For any model distribution $p$:

$$H(q, p) - H(q) = D_{\mathrm{KL}}(q\|p) \ge 0$$

with equality if and only if $p = q$. The inequality is Gibbs', and the proof
is one application of Jensen. Since $\log$ is concave:

$$-D_{\mathrm{KL}}(q\|p) = \sum_i q_i \log \frac{p_i}{q_i} \le \log \sum_i q_i \frac{p_i}{q_i} = \log \sum_i p_i = \log 1 = 0$$

so $D_{\mathrm{KL}} \ge 0$. Therefore $H(q,p) \ge H(q)$ for every $p$: no model,
of any size, trained on any amount of data, can achieve expected cross-entropy
below $H(q)$. The floor is a property of the data-generating process. This is
the formal content of M18.

**Perplexity's units problem.** $\text{PPL} = e^{L}$ with $L$ in nats per
*token*, and tokens are a tokenizer artifact. A tokenizer with a larger
vocabulary packs more characters into each token, so it has fewer, harder
predictions and a higher per-token loss for the same underlying model quality.
Perplexity is therefore only comparable across models sharing a tokenizer. The
tokenizer-invariant quantity is bits per character:

$$\text{BPC} = \frac{L \cdot n_{\text{tokens}}}{n_{\text{chars}} \cdot \ln 2}$$

which converts nats to bits and re-denominates per character. Published
comparisons across model families that quote raw perplexity without a shared
tokenizer are not measuring what they claim.

## Backprop as blame assignment

Formally, backprop is reverse-mode automatic differentiation on the computation
graph, and the cost argument in canon can be made precise.

Let $f = f_L \circ \cdots \circ f_1 : \mathbb{R}^{n} \to \mathbb{R}$ be the
network as a composition of layers. The chain rule for the full Jacobian is

$$\frac{\partial f}{\partial x} = J_L J_{L-1} \cdots J_1$$

where $J_k$ is the Jacobian of layer $k$ evaluated at its input, with shape
(outputs of $k$) x (inputs of $k$). Matrix multiplication is associative, so you
may bracket this product either way, and the two bracketings are the two modes
of automatic differentiation:

- **Forward mode** computes $J_L(J_{L-1}(\cdots(J_1 v)))$, a sequence of
  **Jacobian-vector products**. One sweep gives you the derivative of every
  output with respect to *one* input direction $v$. To get all $n$ inputs, run
  $n$ sweeps.
- **Reverse mode** computes $(((u^\top J_L) J_{L-1}) \cdots) J_1$, a sequence of
  **vector-Jacobian products**. One sweep gives you the derivative of one output
  direction $u$ with respect to *all* inputs. For a scalar loss, $u = 1$ and one
  sweep suffices.

Neither mode ever materializes $J_k$. A dense layer with $d_{\text{in}} = d_{\text{out}} = 4096$
has a Jacobian with $1.7 \times 10^{7}$ entries, but the VJP against it is just
$W^\top \delta$, an $O(d^2)$ matrix-vector product using the weights you already
have. This is what "the backward pass costs about 2x the forward pass" means:
each layer does one VJP for its input blame and one outer product for its weight
gradient, each comparable in cost to the forward matmul.

The general result is the Baur-Strassen theorem: for any function computable by
an arithmetic circuit of size $s$, the gradient of that function with respect to
all its inputs is computable by a circuit of size $O(s)$. The gradient of a
scalar is never more than a constant factor harder than the scalar. That theorem
is the reason deep learning is economically possible.

**The memory bill, quantified.** Reverse mode needs each layer's input at the
time its VJP is evaluated, so activations live from the forward pass until the
backward pass reaches them. For a transformer with $L$ layers, batch $B$,
sequence $S$, and width $d$, stored activations are $O(L \cdot B \cdot S \cdot d)$
plus attention intermediates. Gradient checkpointing stores only every
$\sqrt{L}$-th layer's activations and recomputes the rest during the backward
pass, trading $O(L)$ memory for $O(\sqrt{L})$ memory at the cost of roughly one
extra forward pass, which is a 33% compute increase for an asymptotic memory win.

## Backprop worked: two layers, real numbers

Canon uses $\partial L / \partial z = p - y$ without derivation. Here it is.

Let $z \in \mathbb{R}^{V}$ be the logits, $p_i = e^{z_i} / \sum_j e^{z_j}$, and
$L = -\log p_t$ for the true class $t$. First the softmax Jacobian. For $i = j$:

$$\frac{\partial p_i}{\partial z_i} = \frac{e^{z_i}\sum_j e^{z_j} - e^{z_i}e^{z_i}}{(\sum_j e^{z_j})^2} = p_i - p_i^2 = p_i(1 - p_i)$$

and for $i \ne j$, the numerator's first term vanishes:

$$\frac{\partial p_i}{\partial z_j} = \frac{0 - e^{z_i}e^{z_j}}{(\sum_k e^{z_k})^2} = -p_i p_j$$

Both cases combine into $\partial p_i / \partial z_j = p_i(\delta_{ij} - p_j)$
with $\delta_{ij}$ the Kronecker delta (1 if $i=j$, else 0). This is a dense
$V \times V$ Jacobian, and materializing it for $V = 50{,}000$ would be 2.5
billion entries. It never gets materialized, because of what happens next.

Now chain it against the loss. Since $L = -\log p_t$ depends on $p$ only through
entry $t$:

$$\frac{\partial L}{\partial p_t} = -\frac{1}{p_t}, \qquad \frac{\partial L}{\partial p_i} = 0 \ \text{ for } i \ne t$$

$$\frac{\partial L}{\partial z_j} = \sum_i \frac{\partial L}{\partial p_i}\frac{\partial p_i}{\partial z_j} = -\frac{1}{p_t} \cdot p_t(\delta_{tj} - p_j) = p_j - \delta_{tj}$$

The $p_t$ cancels exactly, the $V \times V$ Jacobian collapses to a $V$-vector,
and $\delta_{tj}$ is precisely the one-hot target $y_j$. Hence
$\partial L / \partial z = p - y$. This cancellation is also why the two
operations are fused in every framework: computed separately, softmax saturates
and $1/p_t$ overflows when $p_t$ is tiny, which is exactly the case where the
gradient is most important. Fused, the result is a subtraction that is stable
for any $p_t$.

**The transpose rule, derived rather than asserted.** For $z = Wh$ with
$W \in \mathbb{R}^{m \times n}$, write $z_i = \sum_j W_{ij} h_j$. Let
$\delta = \partial L / \partial z \in \mathbb{R}^{m}$. Then

$$\frac{\partial L}{\partial h_j} = \sum_{i=1}^{m} \frac{\partial L}{\partial z_i} \frac{\partial z_i}{\partial h_j} = \sum_{i=1}^{m} \delta_i W_{ij} = (W^\top \delta)_j$$

$$\frac{\partial L}{\partial W_{ij}} = \frac{\partial L}{\partial z_i}\frac{\partial z_i}{\partial W_{ij}} = \delta_i h_j = (\delta h^\top)_{ij}$$

Two different contractions of the same three objects. The shapes force the
answer: $\partial L/\partial h$ must be $n$-dimensional and $\delta$ is
$m$-dimensional, so $W$ can only enter transposed. $\partial L / \partial W$
must be $m \times n$, and the only product of an $m$-vector and an $n$-vector
with that shape is the outer product. When you forget the rule, recover it from
dimensions.

**Batched.** With a batch of $B$ examples stacked as rows, $Z = HW^\top$,
$\Delta = \partial L / \partial Z \in \mathbb{R}^{B \times m}$, the weight
gradient sums over the batch: $\partial L / \partial W = \Delta^\top H$. The sum
over $B$ is why gradient noise falls as $1/\sqrt{B}$ and why the loss is defined
as a mean rather than a sum (so that the learning rate does not have to be
retuned when the batch size changes).

**The ReLU gate as a diagonal Jacobian.** $h = \max(0, a)$ has Jacobian
$\mathrm{diag}(\mathbb{1}[a > 0])$, a diagonal matrix of zeros and ones. The VJP
against a diagonal matrix is elementwise multiplication, which is why canon can
write $\partial L/\partial a = (\partial L/\partial h) \odot \mathbb{1}[a>0]$.
The derivative at exactly $a = 0$ is undefined; frameworks pick 0 by convention,
and it has never mattered because exact zeros have measure zero in floating
point.

## Optimizers: four update rules, one loop

**Where bias correction comes from.** Adam's $m_t = \beta_1 m_{t-1} + (1-\beta_1)g_t$
with $m_0 = 0$ unrolls to a geometric sum:

$$m_t = (1-\beta_1)\sum_{k=1}^{t} \beta_1^{t-k} g_k$$

Take expectations assuming the gradient distribution is roughly stationary with
$\mathbb{E}[g_k] = \mathbb{E}[g]$:

$$\mathbb{E}[m_t] = \mathbb{E}[g](1-\beta_1)\sum_{k=1}^{t}\beta_1^{t-k} = \mathbb{E}[g](1-\beta_1)\frac{1-\beta_1^{t}}{1-\beta_1} = \mathbb{E}[g](1-\beta_1^{t})$$

So $m_t$ underestimates the mean by the factor $(1-\beta_1^t)$, and dividing it
out gives an unbiased estimate. At $t=1$ that factor is $0.1$, a 10x correction;
at $t=100$ it is $1 - 0.9^{100} = 0.99997$, effectively 1. The same argument
applies to $v_t$ with $\beta_2$, where the correction matters far longer:
$1 - 0.999^{t}$ is still only $0.63$ at $t = 1000$. Without correction, $\hat v$
would be far too small early in training, the step would be far too large, and
training would diverge in the first few hundred steps. This is also why warmup
and bias correction are partially redundant.

**Adam as a diagonal preconditioner.** Newton's method uses
$w \leftarrow w - \eta H^{-1} g$ with $H$ the Hessian, which is $O(n^2)$ to store
and $O(n^3)$ to invert: impossible at $n = 10^{11}$. Adam can be read as
approximating $H^{-1}$ by $\mathrm{diag}(\sqrt{v})^{-1}$: use the running
second moment of the gradient as a stand-in for curvature, and keep only the
diagonal. The approximation is crude (the second moment of the gradient is not
the curvature; they coincide only under specific assumptions, as in the Fisher
information / Gauss-Newton correspondence) but it is $O(n)$ and it captures the
one thing that matters, which is that different parameter groups in a
transformer live on wildly different scales.

**The sign-descent limit.** As $\epsilon \to 0$ and with $\beta_1 = \beta_2 = 0$
(no memory), the Adam update becomes

$$\frac{m}{\sqrt{v}} = \frac{g}{\sqrt{g^2}} = \mathrm{sign}(g)$$

so Adam degenerates to signed gradient descent, taking a step of exactly $\eta$
in every coordinate. With nonzero $\beta$s it interpolates between sign descent
and normalized momentum. This is the precise sense in which "Adam ignores
gradient magnitude" is true, and it explains both its robustness to bad scaling
and its willingness to move parameters that are receiving nothing but noise.

**Why decoupling matters, algebraically.** Adam with L2 regularization forms
$g' = g + \lambda w$ and then computes the step $\eta \hat m' / (\sqrt{\hat v'} + \epsilon)$.
The decay contribution is therefore divided by $\sqrt{\hat v'}$, which is
dominated by the gradient magnitude. A parameter whose gradients are large gets
its effective decay rate divided by a large number; a parameter whose gradients
are near zero gets it divided by a near-zero number and is decayed hard. The
effective per-parameter decay is $\lambda / \sqrt{\hat v}$, which is not a
regularization scheme anyone would design on purpose. AdamW's

$$w \leftarrow w - \eta\frac{\hat m}{\sqrt{\hat v}+\epsilon} - \eta\lambda w$$

makes the decay a fixed multiplicative shrink of $(1 - \eta\lambda)$ per step,
independent of gradients, which is what "weight decay" meant before Adam
existed.

## What gradient descent actually finds

The saddle-point argument in canon can be made quantitative. At a critical point
of a loss in $n$ dimensions, classification depends on the eigenvalue signs of
the Hessian $H \in \mathbb{R}^{n \times n}$: all positive is a minimum, all
negative a maximum, mixed a saddle. Model the eigenvalue signs as independent
fair coin flips, which is the crude version of the random-matrix argument from
statistical physics of spin glasses (Bray and Dean; Dauphin et al. applied it to
neural nets). Then

$$P(\text{local minimum}) = 2^{-n}$$

At $n = 10^{11}$ this is not small, it is zero for any practical purpose. The
real theory is sharper and more interesting: for random Gaussian fields the
*fraction of negative eigenvalues* at a critical point is tightly coupled to
that point's loss value. Critical points near the global minimum are nearly pure
minima; critical points at high loss are saddles with many descent directions.
The consequence is that high-loss regions are easy to escape and the only places
you can actually stall are already good. Gradient descent on a high-dimensional
loss is not a search that can get trapped; it is a descent that slows as it runs
out of downhill directions.

**Counting the symmetries.** For an MLP layer of width $d$ with an elementwise
activation, let $P$ be any $d \times d$ permutation matrix. Replacing
$W_1 \to PW_1$, $b_1 \to Pb_1$, $W_2 \to W_2P^\top$ leaves the function
unchanged, since elementwise nonlinearities commute with permutation. That is
$d!$ weight settings per layer computing the identical function. For $d = 4096$,
$\log_{10}(4096!) \approx 13{,}000$, and this multiplies across every layer.
ReLU adds a continuous symmetry on top: scaling $W_1 \to cW_1$, $W_2 \to W_2/c$
for $c > 0$ is also function-preserving, because $\text{ReLU}$ is positively
homogeneous. So the set of weights achieving any given loss is not a point, it
is a high-dimensional manifold of astronomically large volume. "The minimum" is
not a well-posed object.

**Sharp versus flat, in Hessian terms.** Take a second-order expansion around a
minimum $w^*$: $L(w^* + \Delta) \approx L(w^*) + \frac{1}{2}\Delta^\top H \Delta$.
A basin is flat when $H$'s eigenvalues are small, meaning the loss is insensitive
to perturbation. The generalization argument is that the train and test loss
surfaces are shifted copies of each other; under a shift of size $\epsilon$, the
test loss at $w^*$ changes by roughly $\frac{1}{2}\epsilon^\top H \epsilon$, so
small $\|H\|$ means the shift costs little. SGD noise has covariance
proportional to $\eta / B$ (learning rate over batch size), and a basin can only
retain the iterate if its curvature is small enough relative to that noise, which
gives a mechanism for why large $\eta/B$ selects flat basins. The caveat is that
sharpness is not reparameterization-invariant, so this whole picture is a useful
heuristic rather than a theorem.

## Scaling laws: what more parameters buy

Canon quotes "20 tokens per parameter" as a rule of thumb. Here is the
optimization it comes from, and the honest reason the number is fuzzy.

Minimize $L(N, D) = E + AN^{-\alpha} + BD^{-\beta}$ subject to $6ND = C$.
Substitute $D = C/(6N)$ to make it one-dimensional:

$$L(N) = E + AN^{-\alpha} + B\left(\frac{6N}{C}\right)^{\beta}$$

Differentiate and set to zero:

$$\frac{dL}{dN} = -\alpha AN^{-\alpha-1} + \beta B \, 6^{\beta} C^{-\beta} N^{\beta-1} = 0$$

$$\alpha A N^{-\alpha-1} = \beta B\, 6^{\beta}C^{-\beta}N^{\beta-1}
\quad\Longrightarrow\quad
N^{\alpha+\beta} = \frac{\alpha A}{\beta B\, 6^{\beta}} C^{\beta}$$

$$N^{*} = \left(\frac{\alpha A}{\beta B\, 6^{\beta}}\right)^{\frac{1}{\alpha+\beta}} C^{\frac{\beta}{\alpha+\beta}}, \qquad D^{*} = \frac{C}{6N^{*}} \propto C^{\frac{\alpha}{\alpha+\beta}}$$

Both are power laws in the compute budget. With $\alpha = 0.34$, $\beta = 0.28$:
$N^* \propto C^{0.452}$ and $D^* \propto C^{0.548}$, so both grow, and roughly
in step, which is Chinchilla's headline claim against the earlier practice of
scaling $N$ far faster than $D$.

Now the wrinkle worth knowing. The optimal ratio is

$$\frac{D^{*}}{N^{*}} \propto C^{\frac{\alpha - \beta}{\alpha+\beta}} = C^{0.097}$$

It is **not a constant**. It drifts upward with compute, slowly. Plugging the
fitted constants in: at $C = 10^{21}$ FLOPs the ratio is about 50; at
$C = 10^{23}$ about 78; at $C = 10^{25}$ about 122. Chinchilla's own paper gives
three estimation approaches (fixed-model training curves, isoFLOP profiles, and
this parametric fit) which agree on the exponents to within error bars but
produce visibly different coefficients, and the famous "20 tokens per parameter"
comes from the first two evaluated at the scale of their experiments. Treat 20:1
as an order-of-magnitude planning heuristic with a genuine spread, not a law. In
practice production models are trained far past it anyway, because that analysis
optimizes training cost alone and ignores that a smaller model is cheaper to
serve for its entire deployed life.

**Why the fit has this functional form.** The $E$ term is the entropy floor from
Gibbs. The $D^{-\beta}$ term has a clean statistical reading: it is the excess
risk of estimating a model from finite samples, which for many estimators falls
as a power of sample count. The $N^{-\alpha}$ term is approximation error, the
gap between the best function in your $N$-parameter hypothesis class and the
true conditional distribution. Approximation error plus estimation error plus
irreducible noise is the standard decomposition of risk in statistical learning.
Scaling laws are that decomposition, with exponents measured rather than derived.

## Post-training: SFT, RLHF, RLAIF, DPO

Canon says DPO comes from inverting the KL-constrained RL optimum. Here is the
derivation, which is short and worth doing because it makes clear exactly what
DPO assumes.

**Step 1: solve the KL-constrained objective in closed form.** RLHF maximizes

$$\max_{\pi} \ \mathbb{E}_{y\sim\pi(\cdot\mid x)}[r(x,y)] - \beta\, D_{\mathrm{KL}}(\pi \,\|\, \pi_{\text{ref}})$$

over distributions $\pi$, with $r$ the reward, $\pi_{\text{ref}}$ the frozen
reference policy, $\beta > 0$ the KL weight. Expand the KL and divide by $\beta$:

$$\max_{\pi}\ \mathbb{E}_{y\sim\pi}\left[\frac{r(x,y)}{\beta} - \log\frac{\pi(y\mid x)}{\pi_{\text{ref}}(y\mid x)}\right]$$

Define $Z(x) = \sum_y \pi_{\text{ref}}(y\mid x)e^{r(x,y)/\beta}$ and rewrite the
bracket as a single KL against a normalized distribution:

$$= \min_{\pi}\ D_{\mathrm{KL}}\!\left(\pi \ \Big\|\ \frac{1}{Z(x)}\pi_{\text{ref}}(y\mid x)e^{r(x,y)/\beta}\right) - \log Z(x)$$

A KL is minimized at exactly zero when its arguments are equal, and $\log Z(x)$
does not depend on $\pi$. So the optimum is

$$\pi^{*}(y\mid x) = \frac{1}{Z(x)}\pi_{\text{ref}}(y\mid x)\exp\!\left(\frac{r(x,y)}{\beta}\right)$$

The optimal policy is the reference reweighted exponentially by reward. This is
a Boltzmann distribution with $\beta$ as temperature, and it is why $\beta$
controls drift: large $\beta$ flattens the reweighting toward $\pi_{\text{ref}}$.

**Step 2: invert it.** Take logs and solve for the reward:

$$r(x,y) = \beta\log\frac{\pi^{*}(y\mid x)}{\pi_{\text{ref}}(y\mid x)} + \beta\log Z(x)$$

Every reward function is expressible in terms of its own optimal policy. The
$Z(x)$ term is intractable (a sum over all sequences), which is what makes this
look useless.

**Step 3: substitute into Bradley-Terry, and watch $Z$ cancel.** The preference
model used to train reward models is

$$P(y_w \succ y_l \mid x) = \sigma\big(r(x,y_w) - r(x,y_l)\big)$$

with $\sigma(u) = 1/(1+e^{-u})$. It depends on the reward only through a
*difference at the same $x$*, and $\beta\log Z(x)$ is identical in both terms,
so it cancels exactly. Substituting and taking the negative log-likelihood over
a preference dataset gives the DPO loss with $\pi_\theta$ in place of $\pi^*$:

$$L_{\mathrm{DPO}} = -\mathbb{E}_{(x,y_w,y_l)}\log\sigma\!\left(\beta\log\frac{\pi_\theta(y_w\mid x)}{\pi_{\text{ref}}(y_w\mid x)} - \beta\log\frac{\pi_\theta(y_l\mid x)}{\pi_{\text{ref}}(y_l\mid x)}\right)$$

**The gradient, which shows what it does per example.** Write the implicit
reward $\hat r_\theta(x,y) = \beta\log\frac{\pi_\theta(y|x)}{\pi_{\text{ref}}(y|x)}$.
Then

$$\nabla_\theta L_{\mathrm{DPO}} = -\beta\,\sigma\big(\hat r_\theta(x,y_l) - \hat r_\theta(x,y_w)\big)\Big[\nabla_\theta\log\pi_\theta(y_w\mid x) - \nabla_\theta\log\pi_\theta(y_l\mid x)\Big]$$

Read it in two parts. The bracket raises the log-probability of the winner and
lowers the loser's, which is the behavior you wanted. The scalar prefactor is
$\sigma$ of *how wrong the current model has the pair ordered*: near 1 when the
model prefers the loser (large gradient), near 0 when it already prefers the
winner by a margin (no gradient). It is a built-in hard-example weighting, and
it is the term that would be missing if you naively did "SFT on winners, negative
SFT on losers".

**What the derivation assumes, and where it bites.** It assumes preferences
follow Bradley-Terry, that $\pi_{\text{ref}}$ is the policy the preferences were
collected under, and that the optimum is attainable within your parameterization.
When the preference data comes from a different model than $\pi_{\text{ref}}$,
the derivation's premises are violated and the practical failure is characteristic:
DPO can reduce $\pi_\theta(y_l)$ far more than it raises $\pi_\theta(y_w)$,
since only the difference is constrained. Probability mass flows to sequences in
neither the winner nor the loser set, and both go down together. PPO does not
have this failure because it samples from the current policy and scores what it
actually produces.

**Post-training against the compute ledger, precisely.** Let $C \approx 6ND$
hold for both phases. Pretraining at $N = 7\times10^{10}$, $D = 1.5\times10^{13}$
is $C \approx 6.3\times10^{24}$ FLOPs. SFT at the same $N$ over
$D = 5\times10^{7}$ tokens is $C \approx 2.1\times10^{19}$ FLOPs, a factor of
$3.3\times10^{-6}$. Even generous accounting for RLHF (multiple sampled rollouts
per prompt, a reward model forward pass, several epochs) moves this to perhaps
$10^{-4}$. Whatever post-training does, it does it by moving the weights a very
short distance. That is a mechanism-level argument for M15 that does not depend
on any benchmark.
