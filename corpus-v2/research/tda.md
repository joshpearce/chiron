# Training-data attribution for compensation: survey, August 2026

**The organizing distinction (ATTRIB 2026 CFP): contributive attribution** (causal effect of training data on an output) **vs corroborative attribution** (sources that support an output). A royalty model logically requires the first; every deployed commercial system implements the second, or neither.

## 1. Influence functions

Koh & Liang 2017 (arXiv:1703.04730, ICML best paper): I(z, z_test) = -grad L(z_test)^T H^-1 grad L(z). Derivation requires twice-differentiable, strictly convex loss at an exact optimum - none hold for LLMs.

Grosse et al. 2023 (Anthropic, arXiv:2308.03296): EK-FAC scaled influence to 52B params (still the largest lab run). Findings: influence is SPARSE (few sequences dominate); becomes MORE ABSTRACT with scale (small models match surface tokens, large models match concepts); cross-lingual generalization emerges with scale; influence COLLAPSES to near zero when key-phrase word order is flipped (the estimator picks up token-order-sensitive gradient alignment, not semantics). Published as a generalization study, not an attribution product.

DataInf (arXiv:2310.00902, ICLR 2024): closed-form approximation cheap at LoRA-adapter scale.

The case against, largely from insiders: Basu/Pope/Feizi (ICLR 2021) "often erroneous" for deep nets; Bae et al. (NeurIPS 2022, Grosse's student) - influence functions estimate the proximal Bregman response function, NOT leave-one-out; Li et al. 2024 (arXiv:2409.19998) "consistently poor across most LLM settings"; Rubinstein & Hopkins (arXiv:2512.12572) tight bounds showing Newton-step fundamentally more accurate even in convex logistic regression.

## 2. Gradient/trajectory methods

- **TracIn** (arXiv:2002.08484): sum over checkpoints of grad L(z_test) . grad L(z), LR-scaled. NO Hessian, no convexity/optimality assumption. Easiest method to implement correctly.
- **Datamodels** (arXiv:2202.00622): train thousands of models on random subsets, fit linear surrogate from membership to output. Released 4M trained networks. Origin of the standard metric: **Linear Datamodeling Score (LDS)** - Spearman correlation of predicted vs actual retrained outputs (single-example LOO drowns in seed noise; LDS doesn't).
- **TRAK** (arXiv:2303.14186): JL-projection of gradients + empirical-NTK linearization + small ensembles; "2-3 orders of magnitude cheaper"; BERT-base/QNLI attribution ~2h on 8xA100.
- **LoGra/LogIX** (arXiv:2405.13954): projection fused into backward pass; claims 6,500x throughput, Llama3-8B over 1B tokens.
- Libraries: dattri (NeurIPS 2024 D&B - IF/TracIn/TRAK/KNN-Shapley; largest LM benchmarked = GPT-2 124M), TRAK repo, Kronfluence (EK-FAC to Llama-3-8B), LogIX, pyDVL (Shapley variants incl. GroupedDataset + MSRSampler).

**The measurement that changes priors - MAGIC (arXiv:2504.16430, Ilyas & Engstrom 2025)**: near-optimal method vs ground truth. Gemma-2B LoRA: MAGIC rho=0.97 while baseline methods are "no better than random guessing." Catch: costs 3-5x a full training run PER TEST SAMPLE, doesn't scale to pretraining. **The central dilemma: the only method that demonstrably correlates with ground truth doesn't scale; every method that scales doesn't demonstrably correlate.**

2026: Deng et al. (arXiv:2605.18814) - dominant error in trajectory methods is OPTIMIZER MISMATCH (literature assumes SGD, models use AdamW; correcting yields 10-300% improvement). TrackStar (arXiv:2410.17413, DeepMind; 8B/160B-token full-corpus run): persistent gap between influence and factual attribution - **BM25 beats influence at finding the fact-containing passage**. A royalty system must pick its target (influence vs entailment) and defend it. Also: STRIDE, RISE, interaction-aware group influence (arXiv:2605.15675, first serious relaxation of additivity), SOURCE.

## 3. Game-theoretic valuation

Data Shapley (arXiv:1904.02868): axioms uniquely determine the value; TMC sampling. **Scale check: all per-datum experiments at n~1,000.** The escape hatch is in the original paper: **Group Shapley (section 4.5)** - groups as players, 146 groups demonstrated. Source-level attribution has been in the literature since 2019.

KNN-Shapley (arXiv:1908.08619): exact for all n in O(n log n); demonstrated at 10M points; values a K-NN surrogate, not your model. Variants: Distributional Shapley (value of the distribution - essential for markets), Beta Shapley, **Data Banzhaf (arXiv:2205.15466: constant weights provably maximize robustness to training noise; MSR estimator needs O((1/eps^2) log(n/delta)) total evaluations regardless of n)**, LAVA, Data-OOB.

**In-Run Data Shapley (arXiv:2406.11011, ICLR 2025 Outstanding Paper runner-up)**: ghost dot-products compute all pairwise gradient inner products inside a single backprop; GPT2-small + Pythia-410M on Pile ~10B tokens; 7.5% overhead first-order. The only Shapley-style attribution ever applied to foundation-model pretraining. No public repo found.

**The key precedent: "An Economic Solution to Copyright Challenges of Generative AI" - Wang, Deng, Chiba-Okabe, Barak, Su (arXiv:2404.13964).** Utility = log-density ratio of coalition-fine-tuned model vs public-baseline model on the generated artifact (generative models ARE density estimators - no labeled validation set needed). Shapley Royalty Share = phi_i / sum(phi_j), negatives clipped. Made computable by: players = copyright OWNERS (n=4-5), LoRA fine-tunes off a shared base, Monte Carlo over 20 samples. Experiments: SD v1-4 on 4 artists; **Pythia-410M on 5 Pile sources**. NO cost figures, NO code repo (verified twice). Stated limitations: source merging/splitting manipulation, adversarial padding, developer/owner split undetermined.

Follow-ups constraining design:
- **Faithful Group Shapley (arXiv:2505.19013, NeurIPS 2025)**: formalizes the **shell-company attack** - group Shapley is not split-proof; a rightsholder incorporating as several entities inflates payout. FGSV is provably fragmentation-immune.
- **Cluster Shapley (arXiv:2505.23842)**: cluster-level Shapley with error bounds; finds equal-split and relevance-proportional allocation (the two rules every commercial scheme uses) "highly unfair" vs Shapley.
- **"Computational Copyright" music royalty model (arXiv:2312.06646)**: Content-ID economics with causal TDA; similarity detection under-compensates "hidden influencers" that drive utility without resembling outputs.
- **"What's a Credit Worth?" (arXiv:2607.00641, Jul 2026)**: puts attribution SNR directly into the payment rule; closed-form per-creator payments; validated vs leave-one-catalogue-out on music models. **Headline: when attribution is noisy, the welfare-optimal contract collapses to a flat fee.** A formal threshold for whether a royalty scheme is worth building.
- **"Semivalue-based data valuation is arbitrary and gameable" (arXiv:2506.12619)**: defensible small changes to the utility move valuations substantially.
- **"Rethinking Data Shapley" (arXiv:2405.03875, Jia's group)**: Thm 2 - absent structure, Shapley selection can be useless; Thm 6 - Shapley IS optimal when utility is monotonically-transformed-modular. **Ships the MTM-residual diagnostic - run it before claiming a ranking means anything.** Rebutted in part by NASH (ICML 2026 Spotlight).
- OpenDataVal benchmark: "no single algorithm uniformly best"; several methods at/below random on specific tasks.

Commercial reality: no deployed system documents an attribution algorithm. Production allocation rules: flat pro-rata (Shutterstock: median $0.0069/image), discretionary (Adobe Firefly: one contributor reported $4.88), relevance-proportional patented-undisclosed (Bria), unit-price metering (TollBit/Cloudflare, honestly), undefined (ProRata, Sureel). Jia et al. (arXiv:2601.09966): across 73 public deals, "the majority of value accrues to aggregators, with documented creator royalties rounding to zero." Industry converged on flat fees BECAUSE attribution doesn't clear the SNR bar. No law review engages Shapley royalties on technical merits - mechanism-design and legal literatures aren't talking.

## 4. Memorization/extraction (the degenerate easy case)

Carlini 2021 (arXiv:2012.07805) extraction from GPT-2 -> Carlini 2023 (arXiv:2202.07646): memorization log-linear in capacity, duplication, prompt length; >=1% of training set extractable from GPT-Neo 6B; dedup cuts ~3x. Nasr 2023 (arXiv:2311.17035): divergence attack on ChatGPT emitted training data at 150x normal rate - >10,000 unique memorized examples for ~$200; extractable rates LLaMA-65B 0.789%, Pythia-1.4B 0.453%. **Alignment does not remove memorization; it hides it.**

Cooper et al. 2025 (arXiv:2505.12546, COLM 2026): 200 books x 14 open models - most models memorize most books barely; enormous variance; **Llama 3.1 70B memorizes some books essentially completely** (Harry Potter 1 near-verbatim from a few words). Both litigation positions empirically wrong. Ahmed et al. 2026 (arXiv:2601.02671): jailbroken Claude 3.7 95.8% nv-recall of HP1; Gemini 2.5 Pro 76.8% without jailbreak; GPT-4.1 refuses (4.0%).

Verbatim vs semantic: Ippolito 2022 - perfect verbatim blocking defeated by style transfer; verbatim definitions are a measurement artifact. Counterfactual memorization (arXiv:2112.12938) is the causal definition bridging to attribution. EleutherAI (arXiv:2406.17746): much measured "memorization" is reconstruction of templatic text carrying no attribution signal.

Membership inference: DO NOT BUILD ON IT. Duan et al. (arXiv:2402.07841, MIMIR): barely above random across Pythia/Pile; apparent successes are temporal distribution shift. Das et al. (arXiv:2406.16201): blind classifiers beat published MIAs. Zhang/Das/Kamath/Tramer (arXiv:2409.19798): a training-data PROOF needs a low FPR, which needs sampling the null "model was not trained on my data" - **you cannot sample that null**; sound paths are extraction or PRE-INSERTED canaries. Hayes et al. (arXiv:2505.18773, definitive scaling test): even strong MIAs AUC<0.7 in practical settings, and **per-sample decisions statistically indistinguishable from coin flips under training randomness alone**. Morris et al. 2025 (arXiv:2505.24832): capacity ~3.6 bits/param; their law predicts MIA F1 = chance at >=100 tokens/param (frontier trains at 10^3-10^4).

Positives: LLM Dataset Inference (arXiv:2406.06443) - aggregate collection-level test works (p<0.1 on Pile splits); **the unit of evidence is a corpus, not a sentence**. DE-COP-style designs with matched controls hit AUROC 82% on GPT-4o for paywalled O'Reilly content - power from experimental design, not statistics. CheckMIABench (arXiv:2606.17464) is the correct benchmark now; WikiMIA deprecated.

**Memorization gives ~0.1-1% of tokens under adversarial prompting, adversarially shaped coverage (boilerplate + rare high-entropy strings), missing the ordinary once-seen influential document. A lower bound and existence proof, not a solution.**

## 5. Watermarks/canaries/provenance

Wei, Wang, Jia (arXiv:2402.10892, Findings ACL 2024): **the watermark is sampled from a known space, so unused watermarks ARE the null** - manufactured exactly what post-hoc MIA cannot have. Hypothesis test, black-box loss access. 256 watermarked docs with length-80 random sequences -> strong detection on 70M model/100M tokens; BLOOM-176B retrospective: SHA-512 hashes 10-sigma at 12 occurrences. **Scaling: larger corpora dilute, larger models amplify; roughly cancels at Chinchilla ratios but NOT in the over-training regime - open question.**

Copyright Traps (arXiv:2402.09363, ICML 2024): 1.3B from-scratch; medium traps x100 repeats NOT detectable; long sequences heavily repeated reach only AUC 0.75. Weak result dressed as positive.

Privacy Auditing of LLMs (arXiv:2503.06808, ICLR 2025): **new-token canaries** (tokens added to vocab, used only in canaries) - 49.6% TPR at 1% FPR on Qwen2.5-0.5B vs 4.2% prior SOTA (~12x). Content obviously artificial - fine for self-audit, useless for readable prose.

Hubble (arXiv:2510.19811): 1B/8B models on 100-500B tokens with controlled insertions. **Governing law: memorization risk depends on frequency RELATIVE to corpus size** - a trap that worked at 100B tokens needs ~5x duplication at 500B. Mitigations: dilution, schedule-sensitive-data-early.

Radioactive data (arXiv:2002.00937): image feature-space marking, p<0.0001 with 1% marked - **no clean text analogue exists (structural gap)**. Homoglyphs are dead on arrival vs NFKC normalization. 2026 frontier: KGW-team dataset watermarking via greenlist rewriting (arXiv:2607.00325; 10B-token budget on Qwen3-1.7B; watermarked fraction 0.005-0.077%; works from-scratch, weaker under continued pretraining).

Output watermarking is a DIFFERENT problem: KGW (arXiv:2301.10226), SynthID-Text (Nature 2024, deployed in Gemini). Proves "this text came from my model," carries zero information about training influence. Distillation inherits and can strip watermarks.

Standards: C2PA 2.3 has data-mining assertions and Collection Data Hash assertions - a container awaiting content; no lab publishes one. (NB the verification survey found the do-not-train assertion REMOVED in v2.0 - reconcile before citing.) IETF AIPREF; RSL defines pay-per-inference but NO measurement methodology and NO allocation formula. EU AI Act 53(1)(d): corpus-level summaries, not per-output attribution. Data Provenance Initiative (Longpre, Nature MI 2024): >70% license omission on popular hosts - provenance metadata as practiced is unreliable. All cryptographic attestation attests to what the trainer SAYS it did.

## 6. Retrieval-based attribution (the cheap shortcut that works)

infini-gram (arXiv:2401.17377): suffix array, 5T tokens, ~20ms n-gram counts, no GPU; public API. infini-gram mini (arXiv:2506.12229): FM-index at 44% of corpus size; 83TB indexed in 99 days on one node. SoftMatcha 2 (ICML 2026): FUZZY matching over trillion-scale in <0.3s. OLMoTrace (arXiv:2504.07096, ACL 2025 demo): maximal verbatim spans + rarity ranking + BM25 rerank; 4.46s avg in production over 4.6T tokens; no GPU. Direct quote: "The retrieved documents should not be interpreted as having a causal effect on the LM output." Lexical overlap with an excellent interface; requires publishing the corpus.

RAG citation is a different problem: ALCE, Attributed QA (AIS = corroboration standard), ContextCite (a datamodel over the CONTEXT WINDOW - right math, wrong object). **Compensation built on RAG citations pays for retrieval placement, not training contribution.** Two systems can emit identical citations trained on disjoint corpora.

The one genuine exception: **Source-Aware Training (arXiv:2404.01019, COLM 2024)**: bind document-ID tokens during pretraining, instruction-tune to emit supporting IDs - attribution from the weights. Small scale only, nobody at frontier adopted it. Architecturally the cleanest route to a built-in compensation system.

DATE-LM benchmark: no method dominates; **cheap baselines (BM25, embedding similarity) match or beat expensive gradient attribution**; prior factual-attribution benchmarks were lexically biased.

Commercial: nothing pays for measured training contribution. ProRata publishes zero methodology (whitepaper is a lead-capture form); Perplexity counts citations (equal split regardless of contribution); TollBit/Cloudflare meter access honestly; Microsoft PCM counts grounding tokens (closest to measured, still retrieval accounting). **The industry substituted retrieval accounting for attribution because retrieval accounting is computable.**

## 7. Frontier and meta

ATTRIB workshop returns at NeurIPS 2026 (Sydney, Dec 11-12; submissions opened Aug 1, idea-track 2-4 pages). dattri + DATE-LM are the benchmarks; LDS the standard metric. Anthropic's public TDA work stops at Grosse 2023; RSP "provable inference" commitment is about binding outputs to weights, not data. OpenAI Media Manager never shipped. Attribution is GAMEABLE: adversarial inflation of scores (arXiv:2409.05657); distributed-learning fragility (arXiv:2605.15520 - a participant can inflate its own attribution). **Current TDA is not incentive-compatible; a deployed mechanism needs an identity/registration layer outside the math.** Position papers arguing infeasibility at scale: arXiv:2510.24990 (Economics of AI Training Data), arXiv:2504.12427.

## 8. Feasibility grid (solo engineer, M5 MacBook + modest cloud)

| Family | Verdict |
|---|---|
| Corpus search (suffix arrays) | Trivially feasible; laptop; build FIRST as demo surface; honest about being lexical |
| TracIn | Feasible on laptop at nanoGPT-124M; cheapest causal-ish signal; easiest to get right |
| TRAK/datamodels | Feasible iff you train small - minutes per run means 50-100 models is a weekend; gives LDS ground-truth check |
| Influence (EK-FAC/LoGra) | Runnable to 8B, but 4 independent results say unreliable - demo, don't build payments on |
| MAGIC | Not a system; use as ground-truth oracle for a handful of test points at nanoGPT scale |
| **Shapley over 10-15 SOURCES** | **Feasible and unoccupied. 2^n trainings TOTAL (cached coalitions): n=12 -> 4,096 runs ~68 GPU-hrs, a few hundred dollars. The cliff is n~15, not n~10** |
| Shapley 20-50 sources | Banzhaf+MSR: ~2,000-5,000 runs regardless of n; trade efficiency axiom, renormalize |
| Per-example Shapley | Not feasible, wrong unit anyway |
| Canaries/data watermarks | Feasible second experiment (70M-1.7B from scratch); gives the only PROVABLE claim |

**Error budget: training stochasticity dominates, not combinatorics. Budget >=3 seeds per coalition, averaged before the Shapley sum; if forced to choose, spend on seeds (Banzhaf's safety-margin theorem). Use a smooth value function (held-out log-likelihood) - this single choice matters more than the estimator. LoRA-from-shared-base saves 1-2 orders of magnitude if you state you're valuing fine-tuning contribution.**

## 9. Math prerequisites (from pre-calculus)

- Tier 0 (days): retrieval/corpus search - data structures, logs, ratios.
- Tier 1 (2-4 wks): memorization/MIA/canaries - expectation, variance, binomial z-test, likelihood ratios, ROC and TPR-at-low-FPR, **what it means to sample from a null**.
- Tier 2 (3-6 wks): **Shapley - combinatorics, expectation over permutations, the axioms, Monte Carlo + Hoeffding. NO CALCULUS ANYWHERE IN IT. Lowest barrier of any causal method, and the one whose fairness argument you can explain to a lawyer.** Plus basic mechanism design for the shell-company attack.
- Tier 3 (2-4 months, builds on vol-1): TracIn - partial derivatives, gradients, chain rule, dot products as alignment.
- Tier 4 (4-6 months): TRAK/datamodels - JL projections, least squares, NTK intuition.
- Tier 5 (6-12 months): influence functions - second-order Taylor, Hessians, implicit function theorem, eigendecomposition, Kronecker products.

**The deepest math is required by the family with the weakest empirical support. The family that fits a compensation mechanism sits at Tier 2.**

## 10. The bet: source-level semivalues + retrieval demo surface

Build: nanoGPT-scale model on 10-12 named sources; enumerate all 2^n coalitions, 3 seeds each, v(S) = held-out log-likelihood; exact Shapley + Banzhaf + LOO from one cached utility table (pyDVL GroupedDataset); suffix-array index on top so a user sees lexical matches (fast, honest, wrong question) NEXT TO the Shapley split (slow, principled, right question). **The contrast is the demo.**

Non-negotiable inclusions: (1) the MTM-residual diagnostic before claiming the ranking means anything; (2) LOO ground truth + seed-noise SD alongside every marginal contribution - measuring your own SNR is a headline either way (below threshold, flat fees are optimal and you say so); (3) the shell-company attack demonstrated, then fixed (FGSV/identity constraints) - every deployed scheme is vulnerable and none discuss it.

Why this bet: (1) Shapley's output IS a normalized allocation with a fairness argument (axioms) - no gradient method can answer "why is my cut 3.2%"; (2) at source granularity the combinatorics collapse and cost inverts in your favor; (3) evidence against alternatives is stronger than against this; (4) the direct precedent (2404.13964) has no open code and no cost figures - a faithful reimplementation with the omitted numbers and skipped diagnostics is a real contribution; (5) the math is reachable (Tier 2).

Verification flags preserved from the survey: 2404.13964 never explicitly states all 2^n models trained; no repos for In-Run Shapley/FGSV/Cluster Shapley/Distributional Shapley; ATTRIB-2025 skip inferred; several figures (Nasr 600GB, Carlini manual-inspection counts, SynthID detection power, Meeus trap grid) unverified - do not quote without primary-source check.
