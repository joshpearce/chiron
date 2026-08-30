# Verification-side training-data provenance, August 2026

**The headline split, which determines product design:** retrospective methods (membership inference, extraction) cannot produce a sound false-positive rate - to claim "your model trained on my data" at confidence you must characterize the null "the model did NOT train on it," and you cannot sample that null (Zhang, Das, Kamath, Tramer, SaTML 2025, arXiv:2409.19798). **Prospective methods (canaries, data watermarks, copyright traps) can**: keyed secrets embedded before scraping, with matched never-published controls, construct the null by design. Only this family supports the word "proof."

The regulator left the gap open deliberately: EU Commission explanatory notice **C(2025) 8311 final (Dec 5, 2025)**, para 26 - the AI Office supervises Art. 53(1)(d) "without performing a work-by-work assessment or checks whether specific content has been used." Mandatory disclosure, explicitly disclaimed verification. That is the commercial niche, stated by the regulator.

## 1. Membership inference: what works, at what granularity

Negative results (real, they hold): Duan et al. COLM 2024 (MIMIR) - barely above random, Pythia 160M-12B on Pile; apparent successes = temporal shift. Das/Zhang/Tramer - blind baselines (never query the model) beat published SOTA on 8 datasets; benchmarks were learning the date. Method line for completeness: Min-K% (ICLR 2024) -> Min-K%++ (ICLR 2025 Spotlight, Zhang et al./Duke) -> ReCaLL (EMNLP 2024). Real relative gains against invalid ground truth.

Granularity rescues it: Puerto et al. NAACL 2025 - fails at sentence level, succeeds when documents aggregate. Meeus et al. USENIX Sec 2024: document-level AUC 0.856 (books), 0.678 (papers) vs OpenLLaMA-7B. **Dataset inference is the robust form** (Maini et al., NeurIPS 2024, arXiv:2406.06443): select signal-bearing MIA features for THIS distribution, aggregate, hypothesis-test the COLLECTION; p<0.1 on Pile splits, no false positives; needs a private same-distribution held-out set.

2025-26 fixes for the held-out-set dependency (key for retrospective auditing):
- Synthetic held-out DI (ICML 2025, arXiv:2506.15271).
- **Natural Identifiers (ICLR 2026, arXiv:2606.24408)**: structured random strings occurring naturally (hashes, shortened URLs) - unlimited additional strings from the same distribution = valid null with no retraining, no private data. Most important 2026 retrospective development.
- N-gram coverage MIA from text outputs alone, no logits (CoLM 2025, arXiv:2508.09603) - works on GPT-4o-class APIs; the only branch functioning against closed APIs.

2026 verdict (arXiv:2601.12937): fine-tuning on semantics-preserving paraphrases collapses SOTA MIAs - "insufficient as a standalone mechanism for copyright auditing." An adversary can defeat the audit cheaply.

## 2. Memorization and extraction

Carlini ICLR 2023: memorization log-linear in capacity, duplication, context length - all three actionable for trap design. Nasr et al.: discoverable vs extractable memorization; >10,000 verbatim examples from gpt-3.5-turbo for $200 (divergence attack, 150x); open-model rates RedPajama-7B 1.438%, Pythia-6.9B 0.548%; 93% of extracted strings appeared ONCE in training. Probabilistic extraction (arXiv:2410.19482) is the honest metric for reports. Cooper et al. COLM 2026: enormous variance; Llama 3.1 70B fully memorizes certain titles (HP1 near-complete); helps neither litigation side cleanly.

**Operational: extraction is the strongest evidence available - and available for almost nothing.** Fires on high-duplication works only; silent on 99.9% of a corpus; cannot be the basis of a general audit; budget it in every engagement as the exhibit, expect it to fire rarely.

## 3. Watermarks/canaries - the load-bearing family

Lineage: Secret Sharer (USENIX 2019) - canary insertion + exposure metric (rank of true canary among its randomness space).

**Wei, Wang, Jia (Findings ACL 2024, arXiv:2402.10892) - the paper to build on.** Hypothesis test with guaranteed FPR, black-box loss access. BLOOM-176B: robust detection past ~90 occurrences; SHA-512 hashes extreme strength at 12 occurrences. Scaling: larger corpora weaken, larger models strengthen - roughly cancels at frontier.

**Fictitious-knowledge watermarks (Cui et al., Findings ACL 2025, arXiv:2503.04036) - fixes the practical weakness.** Coherent statements about invented entities: survive dedup and quality filters, detectable by ASKING THE MODEL A FACTOID QUESTION (no logits). Numbers: 256 injected docs = <0.1% of training set suffices; 25 distinct watermarks of 100-200 tokens reach significance; z=-5.374 after continued pretraining, -4.6 after instruction tuning; 76.5% attribute recall via QA; 4+ attributes memorize much better than one. Survives continual pretraining AND SFT.

Pessimistic counterweight: Copyright Traps (ICML 2024) - naive medium-length traps x100 undetectable; long+heavy repetition only AUC 0.75. Read Wei+Cui together: design details separate a working watermark from a useless one.

2025-26 detection-floor breakthroughs:
- **STAMP (ICML 2025, arXiv:2504.13416)**: several watermarked rephrasings under a secret key, publish ONE, keep the rest as private controls, PAIRED test on likelihoods. Detects content appearing ONCE at <0.001% of tokens. The private versions ARE the null.
- SPECTRA (AAAI 2026): paraphrase + external scoring model; p-value gap of 9 orders of magnitude at <0.001% presence.
- **PRISM (EACL 2026, arXiv:2603.22707)**: the INVERSE product - certify a dataset was NOT trained on (rank-correlation of token log-probs between models); grey-box. Compliance sale to developers.
- Data Taggants (ICLR 2025): clean-label poisoning with statistical certificates (vision, framing transfers). Kirchenbauer 2026: output-watermarked text is RADIOACTIVE - the signal persists when scraped into someone's training data (accidental provenance for synthetic-data producers).
- Hubble (arXiv:2510.19811): memorization risk governed by frequency RELATIVE to corpus size; dilution and early-scheduling are validated mitigations; a trap needs ~5x duplication when the corpus grows 5x.

**What a data provider embeds TODAY**: per collection - keyed fictitious-knowledge documents (4+ attributes, 100-200 tokens, 25+ distinct, >=0.1% density in their own corpus, ideally 90+ effective occurrences); matched control set generated under the same key, NEVER published; public timestamped key commitment. Expected confidence if trained on: z in the -4 to -5 range (p ~ 1e-5 or better), robust through instruction tuning, detectable via plain API. **Catch: only protects data published after embedding. Permanent gap; name it first in any pitch.**

## 4. Output watermarking (brief)

KGW greenlist z-test (arXiv:2301.10226); SynthID-Text (Nature Oct 2024, deployed in Gemini, open-sourced). Relevance: establishes the statistical grammar data watermarking borrows; and radioactivity gives synthetic-data provenance. Proves model->output, says nothing about data->model.

## 5. Provenance infrastructure

EU Art. 53 template mechanics (from C(2025) 8311): three sections (general info / data sources / processing); sizes in three broad buckets (<1B / 1B-10T / >10T tokens); **top 10% of scraped domains listed (SMEs: top 5% or 1,000)**; "large" public dataset = >3% of that modality's public data, must be named; applies from Aug 2, 2025 (existing models Aug 2, 2027); **enforcement from Aug 2, 2026**, fines 3%/EUR 15M; six-month refresh; para 16 voluntary "upon request" rightsholder queries; para 26 no work-by-work checks; RAG excluded unless the model learns from it; open-source NOT exempt from the summary. **Read as a product spec: mandatory, coarse, self-reported, refreshed, unverified - the audit product lives in the gap between "top 10% of domains" and "was my work used."**

Data Provenance Initiative (MIT): 1,800+ datasets, >70% license omission; ~4,000 datasets audited across modalities, >80% of widely-used training data carries non-commercial restrictions. The empirical baseline and tooling.

C2PA: covers document formats, but **the Training & Data Mining assertion was REMOVED in v2.0**; current 2.2 has only digitalSourceType for AI-generated content. No dedicated do-not-train assertion. A signed carrier for audit reports, not a training-provenance mechanism. [NB: the TDA survey cited 2.3 data-mining assertions - reconcile against the spec before relying.]

## 6. Cryptographic/hardware - honest assessment

- **zkML proof-of-training: not close.** Best (CCS 2024, eprint 2024/162): VGG-11 10M params, 15 min prover per iteration = ~0.4 GFLOP/s proved. A 1B model at Chinchilla = ~10,000 years. Four-plus orders of magnitude out. Do not build on it.
- **Proof-of-Learning: broken** - spoofed at a fraction of honest cost (arXiv:2108.09454, arXiv:2208.03567).
- **TEE-attested training: real, cooperative-only.** H100/H200 confidential computing at compute parity (bottleneck: ~4GB/s CPU-GPU encrypted transfer); federated variants <12% overhead; Laminator (arXiv:2406.17548) produces verifiable property cards; EnclaveX (2026) end-to-end TDX+H200 LLM training. Proves "this attested binary consumed this committed dataset" IF the trainer volunteers. A licensing-integrity feature for cooperating counterparties, not an enforcement tool.

## 7. Legal evidentiary status

Pattern across decided cases: **courts rely on discovery evidence of acquisition, not statistical inference.** Bartz: acquisition records of pirated libraries; $1.5B; final approval Jul 2026. Thomson Reuters v. Ross: direct output comparison. NYT v. OpenAI: MTD largely denied Mar 2025; [the complaint's ~100 side-by-side verbatim exhibits: agent sources conflicted - VERIFY against the complaint before use]. Getty UK: [recollection: primary claims dropped mid-trial, narrow trademark outcome - corroborated by ecosystem agent, verify against judgment]. Kadrey: training-data claim survived early dismissal.

Lemley & Cooper, "Probabilistic Copies in Generative AI Models" (forthcoming Berkeley Tech LJ, arXiv:2607.14532): courts will likely find a model contains a copy "only if it is straightforward to extract that work in outputs."

Evidentiary hierarchy courts accept, best first: (1) documentary acquisition proof, (2) demonstrated verbatim extraction, (3) everything else. Statistical MIA has no track record. **A keyed watermark p-value is untested in court but is the only statistical evidence that could survive a Daubert challenge - it has a genuine, defensible false-positive rate.**

## 8. Feasibility grid (solo engineer, <=1B scale)

| Method | Cost | Claim strength |
|---|---|---|
| Loss/Min-K% MIA | <$10 inference | Suggestive at best; blind baselines beat it |
| Dataset inference | $20-50 | Strong statistical evidence at COLLECTION level if held-out set valid; paraphrase-fragile |
| DI w/ synthetic held-out; Natural Identifiers | $50-150 | Strong; the only retrospective family with a defensible null |
| N-gram coverage MIA (black-box) | $50-200 API | Suggestive-moderate; works where nothing else does |
| Extraction (reproduce Cooper on Llama-70B) | $50-300 | Strongest legally; fires on tiny high-dup subset only |
| Hash data watermark (Wei) | $50-300 (160M-410M from scratch) | PROOF-grade; conspicuous, filterable |
| **Fictitious-knowledge watermark (Cui)** | Pythia $30-150; OLMo-7B CPT on 100M tokens <$50 | **PROOF-grade, survives finetuning, plain-API detection. Best replication target** |
| STAMP/SPECTRA paired rephrasing | $50-200 | Proof-grade, best detection floor (<0.001% presence) |
| PRISM non-membership | ~$50 | Strong exclusion evidence; easier customer |
| Canary exposure (Secret Sharer) | <$50 | Proof-grade for the canary; textbook-simple |
| TEE-attested training | <12% overhead; hardware friction | Proof-grade, cooperative-only |
| zkML | prohibitive by 4+ orders | N/A |
| Proof-of-Learning | cheap | Broken |

## 9. Math prerequisites (pre-calculus audience)

**The strongest methods need the least math - not a coincidence: sound methods are sound because their statistics are simple and their null is constructed.** No calculus on the recommended path.
- Extraction: string matching. No statistics.
- Canary exposure: logs and ranking. One afternoon.
- Data watermarks: a one-sided z-test on a count statistic (mean, SD, normal approx, z, p). z=-1.7 ~ p=0.05; z=-5 ~ p=3e-7. STAMP adds the paired test (easier conceptually). This is the entire mathematical content of the product spine - one intro-stats chapter.
- Dataset inference: two-sample test + fitting a linear feature combination on held-out data (and why fitting and testing on the same data invalidates the p-value).
- MIA evaluation: ROC/AUC and TPR at low FPR.

Where deeper discipline is unavoidable (get outside review): (1) constructing a valid null - every failure in this literature is a null failure; (2) multiple-testing correction (Bonferroni/BH) - non-negotiable for an audit report; (3) dependence between test units - where an opposing expert attacks; (4) pre-registration - test, threshold, key committed publicly BEFORE seeing the model; procedural, and it is what converts a number into evidence.

Skip entirely: zkML math, DP accounting, influence-function calculus.

## 10. Recommendation

**Spine: prospective keyed watermarks with a pre-committed hypothesis test.** Three-layer PoC: (1) product - fictitious-knowledge watermarks + STAMP-style paired nulls + the pre-commitment ledger ("register corpus -> keyed watermarks + never-published controls -> timestamped key commitment -> signed third-party-verifiable report"). **The moat is the pre-commitment ledger, not the ML - a notary business with a statistics engine attached.** (2) Retrospective triage - dataset inference with Natural Identifiers, sold honestly as screening, generating leads for layer 1. (3) The exhibit - extraction; budget it always, expect it rarely.

Adjacent easier first sales: PRISM exclusion certification (friendly recurring compliance sale); EU disclosure-conformance checking (exists only because of the gap the Commission wrote into its own notice).

Do not build: anything resting on MIA alone, zkML proofs, Proof-of-Learning. TEE = licensing-integrity feature only.

Lead with the honest limitation: watermarks protect only data published after embedding; the models people most want to sue trained on data collected years ago; that gap is permanent and layer 2 exists because of it.
