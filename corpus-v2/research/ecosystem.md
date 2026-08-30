# The data-licensing ecosystem, August 2026

**Bottom line:** the engineer is directionally right about the closing web and right that Cloudflare wins the gatekeeper role - but he's ~18 months late on both, and the "Spotify royalties for training data" half is aimed at a market that is actively shrinking. Deals are migrating *away* from training rights toward inference-time/RAG access, courts are converging on "training is fair use, piracy is not," and every attribution startup in the space has already been bought or funded. The genuinely unsolved problem is not attribution - it's that **no frontier lab has agreed to pay into any marketplace, by any mechanism, anywhere.**

## 1. Cloudflare: the prediction is correct and already executed

- Jul 1, 2025 - "Content Independence Day": default AI-crawler blocking for new domains; Pay Per Crawl enters private beta.
- Sep 24, 2025 - Content Signals Policy: plain-language robots.txt directives splitting usage into search, ai-input (RAG/answers), ai-train.
- Sep 2025 - AI Index announced: pub/sub model where sites expose structured updates on content change.
- Jan 15, 2026 - Cloudflare acquires Human Native (London, founded 2024, ~$3.6M seed). Terms undisclosed. "Infrastructure to help creators and AI developers discover, price, and license content."
- Jul 1, 2026 - Pay Per Crawl replaced by Pay Per Use, plus an Attribution Business Insights dashboard. Effective Sept 15, 2026, mixed-use crawlers blocked by default from ad-bearing pages for new/free-tier customers.

Pay Per Crawl mechanics: HTTP 402 Payment Required; flat per-request site-wide price; crawler-price / crawler-exact-price / crawler-max-price headers; crawlers sign requests per RFC 9421 with Ed25519 keys; Cloudflare is Merchant of Record. Pay Per Use: compensation on value created, not fetch (motivating stat: >50% of AI crawler requests re-fetch unchanged content). Launch partners: Ceramic.ai, You.com.

Adoption reality: Cloudflare blocks AI crawlers by default for ~20% of websites; claims 50+ major licensing agreements signed in the past year with blocking as leverage (FT, The Atlantic, Ziff Davis, Conde Nast, AP, People Inc). BUT Pay Per Crawl never left private beta, and NO frontier lab (OpenAI, Google, Anthropic, Meta) participates in either payment scheme.

Crawler identity is solved: Web Bot Auth (Cloudflare-led IETF draft, individual not WG-adopted as of Aug 2026), enforced at Cloudflare, AWS WAF, Akamai, Vercel; signers include Anthropic, OpenAI, Perplexity, Common Crawl, Google.

## 2. Licensing standards: broad endorsement, zero demand-side uptake

RSL (Really Simple Licensing): launched Sep 10, 2025 by the RSL Collective (RSS co-creator Eckart Walther, ex-Ask.com CEO Doug Leeds). robots.txt-embedded terms; free/attribution/subscription/pay-per-crawl/pay-per-inference. Spec 1.0 Dec 10, 2025. Adopters: Reddit, Yahoo, Medium, Ziff Davis, later BuzzFeed, Vox, USA Today Co., People Inc., AP. ~1,500 media orgs endorsing, 50+ formal partners. Hired pricing economist Matt Lindsay; model explicitly Spotify/Apple-like. **No confirmation any AI company has agreed to pay under RSL terms.** Supply-side cartel with no counterparty.

IETF AIPREF WG: draft-ietf-aipref-vocab (train-ai / search categories) and draft-ietf-aipref-attach (Content-Usage header). No major AI company reads AIPREF in production.

Pattern: expressing preferences is standardized and free; enforcing them is Cloudflare's; getting paid is unsolved.

## 3. Deal landscape and going rates

~91 publicly announced deals since Jan 2023 (12 in 2023, 28 in 2024, dip in 2025, ~36 projected 2026); industry estimate 50-100 private deals per public one. OpenAI 24 public deals; Anthropic 0.

| Deal | Value |
|---|---|
| News Corp - OpenAI (5yr) | $250M total (largest disclosed) |
| Reddit - Google | ~$60M/yr |
| Reddit total at IPO | $203M aggregate, 2-3yr terms |
| Amazon - NYT | $20-25M/yr |
| Axel Springer - OpenAI (3yr) | ~$13M/yr |
| Wiley AI licensing FY2026 | $49M (+23% YoY), $110M+ lifetime |
| Taylor & Francis - Microsoft | $10M |
| Perplexity Comet Plus pool | $42.5M |

Structure: overwhelmingly lump-sum multi-year bilateral. Private training-rights deals estimated at only $75-100M/yr industry-wide.

**Most important structural finding:** only ~40% of recent deals include training rights, down from near-universal in 2023-24. Attribution/live-access deals grew 2 (2023) -> 18 (2025) -> ~34 projected (2026). The market moved from "buy content to build models" to "license content to deliver answers."

Adjacent and far larger - labeled/expert data: Meta paid $14.3B for 49% of Scale AI (Jun 2025). Mercor: $10B valuation Oct 2025, in talks at $20B, $614M revenue in H1 2026 (~$2B annualized), ~90% from labs. Surge raising up to $1B. A single labeled-data vendor out-earns the entire disclosed content-licensing market.

## 4. Attribution / compensation startups: consolidating, not greenfield

- ProRata.ai (Bill Gross): $40M Series B Sep 2025, $75M+ total. Gist.ai answer engine; RAG-time attribution; 50% ad-revenue split pro-rata; 1,000+ publications. No public payout figures.
- TollBit: $31M total. Two-sided marketplace, per-request bot paywall. ~7,000 publisher sites, ~20% earning revenue (hundreds to tens of thousands $/month). Publicly pitched as "Spotify for AI data licensing."
- Human Native AI: acquired by Cloudflare Jan 15, 2026.
- Music sector (furthest along, already consolidated): Sureel AI (patented "AI DNA") acquired by Warner Music Jun 10, 2026; Vermillio (TraceID) $16M co-led by Sony Music; Musical AI $4.5M Jan 2026, still independent.

Pattern: attribution startups get bought by rightsholders or gatekeepers, not scaled into independent royalty exchanges.

## 5. Litigation: trending against the need for a royalty market

- Bartz v. Anthropic (Alsup, N.D. Cal.): training on lawfully acquired books = "quintessentially transformative" fair use; retaining pirated copies = not. $1.5B settlement, final approval Jul 2026, ~$3,000/work across ~500,000 works. Prices ACQUISITION, once - not a royalty stream. Output-side claims explicitly preserved.
- Kadrey v. Meta (Chhabria): training fair use on the record; interlocutory appeal denied Jul 2026.
- Thomson Reuters v. Ross: summary judgment for TR (non-generative competing product); Third Circuit argued Jun 11, 2026, no ruling yet. Most important pending decision.
- Getty v. Stability (UK), [2025] EWHC 2863 (Ch): Getty abandoned primary copyright claims mid-trial; court endorsed "model weights contain no copies"; near-total rightsholder defeat. [Corroborated by two agents; confirm against judgment before pitch use.]
- NYT v. OpenAI: discovery mid-2026, trial expected late 2026/early 2027. Court rejected "training is inherently transformative," leaving fair use fact-dependent on output substitution and market harm.

Emerging consensus: "acquire your copies legitimately and you may train freely" - forces one clean-copy purchase, not an ongoing royalty relationship.

## 6. Common Crawl and shared corpora

News/Media Alliance removal demands (Apr 2026); Digital Content Next cease-and-desist for AP, NYT, NBCU, Bloomberg, NPR, Fox (Jun 3, 2026). Compliance slow; NYT content still retrievable months after agreed removal. >60% of Common Crawl's 2024 donated funding came from gen-AI-affiliated entities. No credible licensed shared corpus has emerged - a real vacuum, but nonprofit/consortium-shaped.

## 7. Frontier lab posture

- OpenAI: ~24 deals, bilateral lump-sum, no marketplace, no attribution commitment. Media Manager (announced May 2024) never shipped.
- Microsoft: the most serious marketplace competitor. Publisher Content Marketplace (PCM), Feb 2026: pay-per-use when Copilot grounds an answer; launch publishers Business Insider, Vox, USA Today Co., People Inc., AP, Hearst, Conde Nast; "click-to-sign" standardized contracts; pricing admittedly unfinalized.
- Perplexity: Comet Plus, 80% of revenue to publishers, $42.5M pool; pays on visits, citations, agent actions. Simultaneously sued by Reddit (Oct 2025).
- Google: ~20-outlet pilot; Reddit ~$60M/yr.
- Anthropic: zero publisher deals; fair-use posture; $1.5B settlement bought out the acquisition defect.

Nobody has committed to built-in training-level attribution. All "attribution" in market is citation-level in RAG surfaces.

Regulatory: EU AI Act Art. 53(1)(d) - structured training-content summaries by source category, mandatory template; AI Office may verify and fine from Aug 2, 2026 (up to EUR 15M / 3% global revenue). Disclosure-shaped, not payment-shaped. The Commission's explanatory notice C(2025) 8311 para 26: supervision happens "without performing a work-by-work assessment or checks whether specific content has been used."

## 8. The skeptical case (answer each in any pitch)

1. No buyer has ever paid into a neutral marketplace (RSL 0 licensees; AIPREF 0 readers; Pay Per Crawl never left beta).
2. Case law removes the forcing function (acquisition cured by one purchase).
3. Training-data share of deals is shrinking (~40% and falling).
4. Training-time attribution is not technically credible as a payment basis at frontier scale; RAG-time attribution is commoditized (ProRata, TollBit, PCM, Perplexity).
5. Blocking leaks (proxies, impersonation, syndicated copies never touching your CDN).
6. Bilateral lump-sum works for parties with leverage; the long tail sees no revenue either way.
7. The gatekeeper vertically integrated (chokepoint + identity + preference + payment + index + marketplace); publishers already call it vendor-locked.
8. Synthetic data caps willingness-to-pay (with the counterweight of model-collapse risk and labs spending their scarcity budget on expert-labeled data instead).

## 9. Market sizing and the Spotify template

Analyst estimates are vendor noise ($4.8B->$22.6B etc.). Defensible bottom-up: disclosed content licensing is low-single-digit-billions CUMULATIVE; Mercor alone does ~$2B annualized on expert data.

Spotify mechanics transfer badly: pro-rata pooling cross-subsidizes (your subscription pays artists you never played), concentrates winner-take-most (99.8% of artists get effectively nothing), and invites farming (citation farms are cheaper than stream farms since plausible citable text costs ~nothing). User-centric alternative shifts only 1-5% toward independents and majors prefer pro-rata. RSL has already hired an economist to build the Spotify-style model - the analogy is the incumbent standard body's stated design, not a differentiated insight.

## Final assessment: the open niches

Already seen by VCs repeatedly: "marketplace + per-output attribution + rev share." Genuinely open:
(a) **Verification and audit** - every player self-reports usage; EU AI Act gives a regulatory forcing function; sells to the buy side.
(b) **Leakage and syndication tracking** - the loudest unaddressed publisher complaint; tractable with near-dup detection at web scale.
(c) **Buy-side data procurement** - provenance-verified sourcing, rights clearance, indemnification; labs demonstrably spend billions here.

Drop training-time attribution as the core payment mechanism; keep the closing-web and gatekeeper theses (correct, accelerating); position adjacent to Cloudflare - verification, leakage, or procurement - not in its lane.

Key sources: blog.cloudflare.com (pay-per-crawl, content-signals-policy, human-native), techcrunch.com 2026/07/01 Cloudflare policy, pressgazette.co.uk (bot-blocking deals, Common Crawl C&D), digiday.com (PCM, pay-per-value), mediaandthemachine.substack.com (deal tracker), rslstandard.org, datatracker.ietf.org/wg/aipref, jurist.org (Bartz approval), twobirds.com (Getty UK), businesswire.com (ProRata B), musicbusinessresearch.wordpress.com (pro-rata vs user-centric).
