# From a glob of data to a trained model: the 2026 practitioner's map

The pipeline is acquire -> extract -> filter -> dedup -> decontaminate -> tokenize -> mix -> shard/pack -> train -> evaluate -> post-train. Roughly 80% is systems engineering; the ML concentrates in quality filtering (a classifier), data mixing (small optimization), and training hyperparameters. In 2026 a solo engineer can pretrain a 561M-param chat model end to end for ~$92 in under 4 hours on a rented 8xH100 node, or a GPT-2-quality 124M model for ~$13. Training is NOT the expensive part of an attribution PoC; provenance plumbing and attribution compute are. The key design decision: whether you can afford ground-truth counterfactuals (leave-one-source-out retrains) - and at 10-30M params you can, for tens of dollars.

## 1. Pipeline stages as practiced

All modern pipelines descend from CCNet (Wenzek 2019, arXiv:1911.00359): shard by hash -> fastText langID -> paragraph dedup -> KenLM perplexity filter. Refinements since:

- **Extraction (HTML)**: trafilatura (FineWeb) vs resiliparse (DCLM; higher recall 74.5% vs 69.5%, more boilerplate 22.8% vs 6.6%, faster). 2026 best speed/quality: rs-trafilatura, F1 0.859 at 44ms/page (WCXB, arXiv:2605.21097). Extraction quality alone moved 13-benchmark average by 1.08 points at 62B tokens (AICC, arXiv:2511.16397).
- **Extraction (PDF/papers)**: GROBID (S2ORC/peS2o) -> Nougat -> marker -> olmOCR (arXiv:2502.18443) and olmOCR 2 (arXiv:2510.19817, 7B VLM, RL on unit-test rewards; preferred over marker 61.3%). For arXiv, prefer LaTeX source from the S3 src/ bucket and skip OCR.
- **Quality filtering - the dominant lever**: heuristics (C4, Gopher) as first pass; then model-based classifiers. DCLM: fastText classifier (OpenHermes+ELI5 positives vs random web), keep top 10% -> the single biggest factor in their benchmark. FineWeb-Edu: Llama-3-70B scores 460K samples 0-5 for educational value, regression head on frozen embeddings (82% F1), keep >=3; cost 6,000 H100-hours; moved MMLU 33->37, ARC 46->57. Nemotron-CC (arXiv:2412.02595): classifier ensemble + synthetic rephrasing, +5.6 MMLU over DCLM with 4x more unique tokens.
- **Dedup**: fuzzy MinHash+LSH (FineWeb config: 5-grams, 112 hashes, 14x8 buckets, ~75% Jaccard; in datatrove). Counterintuitive FineWeb finding: per-snapshot dedup beat global dedup (global removes exactly the popular re-crawled material). Exact substring via suffix arrays (Lee 2021, arXiv:2107.06499; ~10x less memorized-text emission; 50-token threshold). text-dedup packages both; Ai2 BFF for Bloom-filter dedup. **For attribution: dedup destroys provenance - dropping a duplicate CHOOSES which source gets credit. Record duplicate clusters instead of silently collapsing. Novel contribution territory.**
- **PII/decontamination**: regex email/IP anonymization; n-gram overlap vs eval sets (GPT-3 convention 8-13 grams); Dolma's Paloma Bloom-filter decontamination.
- **Tokenizer**: byte-level BPE, GPT-4-style regex pretokenizer. nanochat trains 65,536-vocab on 2B chars in ~1 min (~4.8x compression). Vocab scaling law (Tao, NeurIPS 2024): optimal vocab grows with compute, SHRINKS when data-bottlenecked. For small domain models on few B tokens: 16-32K, not 128K.
- **Mixing**: hand proportions (Dolma 3: 28% web, 20% code, 19% math, 14% QA...); DoReMi (arXiv:2305.10429, proxy-model group-DRO); Data Mixing Laws (arXiv:2403.16952), BiMix, RegMix (arXiv:2407.01492: hundreds of tiny models on random mixtures + regression - **directly reusable as the attribution baseline; same runs give mixing weights AND Shapley-style source values**); Aioli unifies.
- **Packing**: fixed-length sequences, EOS separators. **Intra-document causal masking (arXiv:2402.13991) is nearly mandatory for attribution** - without it a training sequence blends sources and per-source attribution is contaminated at the input.

## Stacks

| Stack | What it gives |
|---|---|
| datatrove (HF) | Composable pipeline blocks; FineWeb pipeline ships as ~100-line example. Best solo default |
| Dolma toolkit (Ai2) | Rust tagging framework - per-document attributes; closest existing thing to provenance-preserving processing; BFF dedup |
| DCLM | Fixed pool + training recipe + 53-task eval harness, 412M-7B |
| NeMo Curator | GPU-accelerated curation |
| text-dedup | Standalone MinHash/SimHash/SuffixArray/Bloom |
| Nanotron / TorchTitan | 3D/4D-parallel trainers past single node |
| Levanter (Stanford) | **Bitwise reproducibility regardless of hardware/parallelism - unusually relevant to attribution replay** |

## 2. Datasets

Web: FineWeb 15T (ODC-By), FineWeb-Edu 1.3T (best quality-per-token for small models), DCLM-baseline 3.8T, Dolma 3 ~5.9T mix (incl. olmOCR'd science PDFs), Nemotron-CC 6.3T, RefinedWeb, RedPajama-V2 (raw + 40 quality signals), Common Pile v0.1 (public domain/openly licensed only; Comma models match Llama 1/2 7B), The Pile (compromised - Books3 DMCA; avoid for licensing-sensitive work).

Papers (the PoC domain): S2ORC (81M+ records, per-doc license metadata); **peS2o (~40M open-access papers, cleaned for pretraining, ODC-By, per-document license in metadata; Common Pile mirror restricted to cleanly-licensed subset) - the right PoC corpus**; arXiv bulk S3 (~9.2TB requester-pays; default arXiv license does NOT allow redistribution - filter to CC subset); PubMed Central OA (per-article licenses, clean XML, underrated).

## 3. Small-scale training, 2026

Reading order: nanoGPT / build-nanogpt -> llm.c (GPT-2 124M: 90 min/$20 on 8xA100; GPT-2 1.6B: 24h/$672 on 8xH100) -> **nanochat (Oct 2025, ~8K lines, tokenizer->pretrain->midtrain->SFT->RL->inference->web UI; the best PoC starting point: produces a demo-able artifact)** -> modded-nanogpt speedrun (124M to 3.28 val loss: 45min May 2024 -> 1.23min Jul 2026; Muon optimizer, QK-Norm, ReLU^2, rotary, FP8, FA3, multi-token prediction) -> TinyLlama (1.1B/3T tokens, 90 days, $50-100K - the overtrained reference) -> SmolLM2/3 (HF's fully-open recipes; SmolLM3-3B: 11.2T tokens, staged curriculum, public configs) -> Olmo 3 (most transparent artifact: Dolma 3, checkpoints throughout training, data, code, Dolci post-training sets, **ships OLMoTrace**).

Cost model: FLOPs ~= 6 x params x tokens. 8xH100 node: $12-16/hr spot, $20-32/hr on-demand; ~3.2 PFLOP/s at realistic MFU.

| Target | Wall-clock (8xH100) | Cost |
|---|---|---|
| 10M params, 1B tokens | ~20s | pennies |
| 30M params, 3B tokens | ~3 min | <$1 |
| 124M (GPT-2 small), 10B tokens | ~40 min | ~$9-16 |
| 561M (nanochat d20), 11.2B tokens | ~3.3h (3h51m full pipeline) | **$92 measured** |
| 1B, 20B tokens (Chinchilla) | ~10.5h | ~$150-260 |
| 1.6B (GPT-2 XL) | 24h | $672 measured |

MacBook/MLX: excellent inference, training ~10 TFLOP/s sustained (~1/300 of 8xH100). Laptop ceiling: **10-50M params on 1-3B tokens overnight** - exactly the scale where ground-truth LOO retraining is affordable. Use the Mac for iteration, rented node for headlines.

## 4. Fine-tune vs from-scratch for attribution

- **From-scratch on tagged corpus: the airtight choice.** Every token in the provenance ledger; no "base model already knew it" confound; $1-100 at PoC scale. Attribution research needs a fully-accounted model, not a good one.
- Continued pretraining: 5-50B domain tokens with 5-20% general replay buffer; attributes the delta only.
- LoRA: matches full FT for SFT-scale data with right LR/rank ("LoRA Without Regret", thinkingmachines.ai), falls behind in pretraining-like regimes; weight structure differs ("intruder dimensions", arXiv:2410.21228). **Fine for the SFT/demo layer, wrong for the corpus you attribute.**
- "The Finetuner's Fallacy" (arXiv:2603.16177): include domain data during pretraining rather than reserving for fine-tuning - citable nudge toward from-scratch.

What the TDA literature does: datamodels (thousands of subset-trained models - the ground truth everyone approximates); TRAK (random projections + ensembles; ~600GB GPU memory at LLM scale); EK-FAC influence at 52B (Grosse; sparse influence, more abstract with scale, collapses under word-order flip); TrackStar (arXiv:2410.17413; 8B model, 160B-token corpus, no subsampling; **attribution-vs-influence distinction: BM25 beats gradient methods at finding the fact-containing document**); **In-Run Data Shapley (arXiv:2406.11011: Shapley-style contributions from a SINGLE training run via ghost dot-products; 7.5% overhead first-order; GPT2/Pythia-410M on Pile - very likely the PoC core algorithm**); LoGra/LogIX (Llama3-8B over 1B tokens); LESS; OLMoTrace (arXiv:2504.07096: extended infini-gram verbatim matching, seconds against 4.6T tokens, the only DEPLOYED attribution system - cheap, exact, explainable baseline); DATE-LM (NeurIPS 2025 D&B benchmark - use it, don't invent evaluation); 2026: GRASP, Bayesian influence, DebugLM, and Nishida arXiv:2608.13515 (task-agnostic influence = gradient step's reduction of distance-to-final-params; Pythia 70M-12B from public checkpoints; influence peaks ~40K steps; arguably the right basis for a royalty split since you don't pay per-query).

Royalty-specific: "Most Expensive Part of an LLM should be its Training Data" (arXiv:2504.12427); "Fair Document Valuation in LLM Summaries via Shapley" (arXiv:2505.23842); Sovereign Context Protocol (arXiv:2603.27094). **Nobody has shipped tagged-corpus -> trained-model -> per-source-payout end to end. That is the gap.**

## 5. Evaluation

Primary metric: **bits-per-byte on held-out domain text** (tokenizer-invariant, monotone, low-variance, works at any size). Harnesses: lm-evaluation-harness; OLMES (pinned prompts/normalization, designed for small base models); DCLM CORE (nanochat targets it; GPT-2 CORE = 0.256525, $92 d20 hits 0.2219); Paloma.

**Steal the FineWeb ablation methodology**: benchmarks chosen for minimal seed variance, monotone improvement, above-random scores; TWO models per condition, different seeds, averaged. For attribution most claims are "removing X changed the model" - the noise floor is the enemy; two seeds + pre-registered low-variance metrics = the difference between a result and a coincidence.

Domain eval: QA from HELD-OUT papers (cloze abstracts, definition retrieval), LLM-generate then hand-check, decontaminate ruthlessly. Below ~1B params ignore MMLU-style reasoning benchmarks (chance-level noise).

## 6. Post-training, briefly

nanochat recipe: midtraining (~8 min - chat format adaptation, the underrated stage), SFT (~7 min), optional GRPO. Numbers stay bad at this scale; the point is a typeable artifact for the demo. TRL v1.0 unified SFT/DPO/GRPO. **For the PoC: SFT only** - more stages mean more data needing provenance for zero demo value.

## 7. Scaling-laws lens

Chinchilla (~20 tok/param compute-optimal) - but nobody trains compute-optimal now; inference-aware training goes much longer (Llama-3 8B at ~1,875 tok/param). **Data-constrained scaling (Muennighoff, arXiv:2305.16264): repeating data up to ~4 epochs is nearly free, diminishing to ~16, nothing past ~40 - this is what lets a 500M-token paper corpus honestly support a 2B-token run.** Quality beats quantity decisively (FineWeb-Edu 1.3T beats FineWeb 15T; DCLM +6.6 MMLU from filtering alone).

Caveats the PoC must state: (1) influence patterns change with scale (small = crisp/top-heavy, frontier = long-tail where per-doc payouts round to zero); (2) attribution != entailment - pick one and defend it; (3) dedup is a payout decision; (4) data ORDER introduces the largest variation in influence (Nishida) - if the split changes on reshuffle, measure and report it.

## 8. PoC recipes

Shared substrate: peS2o (or CC-arXiv), K = 10-50 sources (venue/author-group/subfield), manifest (shard,offset,len)->doc->source, document-masked packing, deterministic replayable dataloader (consider Levanter). The provenance plumbing is the novel contribution.

**Recipe A - counterfactual ground truth (first).** 10-30M params, 300M-1B tokens, K=20-50 sources; 1 full model + K leave-one-source-out + 200-500 random-subset models; ~4 min per run on one H100; 550 runs ~= 37 GPU-hours = **$55-150**. Mac variant: 10M x 200M tokens ~20 min/run, ~50 runs overnight, $0. Outputs: exact LOO deltas, subset-regression Shapley - a royalty split defensible by construction. Then run In-Run Shapley, TRAK, EK-FAC, BM25 on the full model and report rank correlation vs ground truth - the whole scientific result. **Math needed: none.**

**Recipe B - realistic-scale artifact.** nanochat d20 561M, ~11B tokens: 70% FineWeb-Edu (one background source) + 30% tagged papers, ~4 epochs of ~3B unique. $92 on-demand / ~$50 spot + 15 min SFT. Attribution stack: (1) In-Run Data Shapley instrumented in the training loop - the royalty ledger computed DURING training; (2) TrackStar-style gradient influence per query (~$50 more; per-source gradient accumulation collapses storage to K x 4096 floats); (3) infini-gram verbatim matching (tens of GB suffix array, ms queries - the screenshot that lands). Demo: question -> answer -> verbatim spans to papers + influence-ranked sources + $1 revenue pie by In-Run Shapley. Total $150-250.

**Recipe C - fine-tune attribution (backup).** Olmo 3 7B or SmolLM3-3B continued pretraining on 500M-2B tagged tokens (+replay), $30-90; LOO sweep via LoRA ~$900-2K. Confound: base already read arXiv - control by evaluating on post-cutoff papers. Use for "does it survive a real model," never the primary claim.

Sequence: A on laptop (build harness, $0) -> A on one H100 ($100) -> B ($150) -> C in reserve. **Under $500 total.**

## 9. Skills map

Systems (Matt's strength, ~70%): acquisition, extraction, MinHash/suffix arrays, Bloom filters, tokenizer config, sharding/manifest/masked-packing/replayable loaders (**the hard part, entirely systems**), training orchestration, gradient stores (FAISS), suffix-array indexes, eval plumbing, demo app.

Light math (weekend each): cross-entropy/BPB conversion, LR schedules (warmup+cosine; WSD gives branchable checkpoints - useful for attribution), batch/LR heuristics, C~=6ND, Muon-vs-AdamW as a fact.

Where math bites: scaling-law fitting (log-space regression - pre-calc OK if optimizer is a black box); influence functions (the real wall: Hessians, EK-FAC, 2-3 months on top of vol-1); Shapley (combinatorics + Monte Carlo - easier than influence).

Strategic: **Recipe A requires no calculus and produces the ground truth every fancy method approximates. Build it first; learn the calculus while it earns its keep. Most people do this in the wrong order and never know whether their EK-FAC implementation is even correct. Matt would have ground truth.**

Reading order: Karpathy Let's-build-GPT + build-nanogpt -> nanochat walkthrough -> FineWeb paper (ablation methodology) -> DCLM -> Chinchilla -> Muennighoff -> Beyond-Chinchilla -> Olmo 3 + Dolma 3 -> Grosse -> TrackStar -> In-Run Shapley -> OLMoTrace -> Nishida 2608.13515.
