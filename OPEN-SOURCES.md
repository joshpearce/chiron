# Open educational sources for the Smart Book builder

Survey date: 2026-09-05. Every licence claim below was checked on the URL
given next to it on that date unless marked "unverified". Chiron is a
private, single-reader tool whose generated books are never published, so
CC BY, CC BY-SA, CC BY-NC and CC BY-NC-SA all permit what the builder does
(copy, adapt, interleave, rewrite). CC BY-ND and CC BY-NC-ND do not permit
adaptation; "free to read online" with no licence permits quotation only.

This builds on the 2026-08-10 open-courseware survey (commit d112e9a,
"Chapters open with behavior, not notation"). That pass looked at Karpathy,
3Blue1Brown, Alammar, d2l.ai, CS224N/CS324 and fast.ai for chapter shape and
produced the ordering rules in `corpus/authoring-spec.md`. It said nothing
about licences or programmatic access, which is what this document covers.
Note that of the six sources that pass admired, only d2l.ai (CC BY-SA 4.0)
and Alammar (CC BY-NC-SA 4.0) may be adapted; the others are restricted.

## 1. Summary

The open-textbook world is large but concentrated. Four sources cover most of
what Chiron will be asked for. **MIT OpenCourseWare** (CC BY-NC-SA 4.0) has
the widest subject range, a real search API with full-text hits on lecture
notes, per-course JSON, and lecture transcripts, which are the best source of
adaptable spoken human voice anywhere. **LibreTexts** (per-page licence,
mostly CC BY, BY-SA and BY-NC-SA) is the largest catalogue of adaptable
textbook prose, with a public catalogue API and predictable HTML. **OpenStax**
is the most polished, but moved its whole library to CC BY-NC-SA 4.0 on
2026-04-22 (fine for Chiron), and its committee voice is the very thing Matt
wants to escape; use it for scope-and-sequence and worked problems rather than
prose. For Matt's actual interests the standalone books matter more than any
catalogue: d2l.ai (CC BY-SA), SICP (CC BY-SA), Erickson's Algorithms (CC BY),
Downey's Think series (CC BY-NC-SA), Bayes Rules! (CC BY-NC-SA), QuantEcon
(CC BY-SA), the Rust book (MIT/Apache) and Software Foundations (MIT) are
adaptable and have a voice. The famous free ML books (Goodfellow, Sutton and
Barto, Jurafsky and Martin, Boyd, MML, Murphy, OSTEP, Feynman) are free to read
but not adaptable; they are quotation-and-citation sources only. For question banks, MIT OCW problem sets with solutions, OpenStax
CNXML exercises with inline solutions, QuantEcon's exercise-and-solution
directives, Hefferon's full answer book and Downey's solution notebooks are
the sources worth building a parser for (section 2d). The honest
catalogue picture: Open Textbook Library has the only clean, keyless JSON
API; OER Commons needs a token by email; MERLOT needs a licence key; Pressbooks
Directory has no documented API but every Pressbooks network exposes a REST
API per book; DOAB/OAPEN has a keyless REST API and extracted plain text for
open-access monographs.

## 2. Sources by category

Verdict key: **A** = adaptable (licence permits derivative works for
non-commercial use), **Q** = quotation only (free to read, no adaptation
right), **R** = restricted (not even reliably free to read, or terms forbid
reuse). Share-alike (SA) and attribution (BY) obligations are noted; for a
private book they mean "keep the source and licence recorded", nothing more.

The last column, **Question sets**, records whether the source ships
exercises, whether answers or solutions come with them (all, odd-numbered,
instructor-only, none), whether the questions share the prose licence, and
the form they arrive in. Chiron's `questions.yaml` needs constructed items
with answers and rubrics, and MCQs whose distractors map to misconceptions
(`corpus/authoring-spec.md`), so "exercises without solutions" means
prompts the author LLM must still answer and grade itself.

### 2a. Textbook catalogues and publishers

| Source | Subjects | Licence and verdict | Formats and access | Voice and quality | Fetch one chapter | Question sets |
|---|---|---|---|---|---|---|
| OpenStax (openstax.org) | Intro college: calculus 1-3, physics, chemistry, biology, statistics, economics, US and world history, philosophy, psychology, sociology, political science, Python, CS, data science, business | **A.** Library moved to CC BY-NC-SA 4.0 with limited exceptions on 2026-04-22 (https://openstax.org/blog/openstax-licensing, read via the CMS API). CMS API on 2026-09-05: 70 live titles CC BY-NC-SA 4.0, 4 live CC BY 4.0, 33 retired CC BY 4.0 editions. Older CC BY editions stay CC BY where copies exist (LibreTexts mirror of Introductory Statistics 1e is CC BY 4.0: https://stats.libretexts.org/Bookshelves/Introductory_Statistics/Introductory_Statistics_1e_(OpenStax)) | Source is CNXML (XML with MathML) in 57 GitHub repos named `openstax/osbooks-*`, layout `collections/*.collection.xml` (book tree) and `modules/mNNNNN/index.cnxml` (one section each), images in `media/`. Rendered HTML per page from the REX archive JSON. Book list with licence and subject: `https://openstax.org/apps/cms/api/v2/pages/?type=books.Book&fields=title,license_name,license_version,book_subjects,cnx_id&limit=200`. Current archive path and book versions: `https://openstax.org/rex/release.json`. No full-text search. | Committee-written, uniform, bland; strong on learning objectives, worked examples and end-of-chapter problems; weak on motivation. Use for syllabus shape and question banks, not for voice. | `GET https://openstax.org/apps/archive/<archiveUrl>/contents/<book_uuid>@<version>.json` returns the tree; each leaf has a page uuid; `GET .../contents/<book_uuid>@<version>:<page_uuid>.json` returns `{title, content}` with `content` as HTML. Verified with Calculus Volume 1 (8b89d172-2927-466f-8661-01abc7ccdba4@8dbc2ce). Or `raw.githubusercontent.com/openstax/osbooks-<book>/main/modules/<mid>/index.cnxml`. | Rich. Every section's CNXML carries `<exercise><problem>...</problem><solution>...</solution></exercise>` inline (verified: module m54038 of Introductory Business Statistics 2e has 17 exercises with 16 solutions); chapters end with Practice, Homework, Bringing It Together and Review Questions, and an Answer Key back-matter module answers the odd-numbered ones (the even-numbered answers and the test bank are instructor-only behind login, unverified). Same CC BY-NC-SA 4.0 as the prose. Arrives in the module file, so a CNXML parser gets question, answer and section together. |
| LibreTexts (libretexts.org, 13 subject libraries: math, stats, phys, chem, bio, eng, workforce, socialsci, human, geo, med, biz, k12) | Everything undergraduate; the deepest for chemistry, math, statistics, physics; many remixed OpenStax and community texts; humanities and languages thinner | **A (per page).** Every page carries its own licence tag in the HTML (`license:ccby`, `license:ccbysa`, `license:ccbyncsa`, `licenseversion:40`) and a footer sentence "shared under a CC BY 4.0 license" (verified on the page above). Books without a declared licence show "not declared" (e.g. https://chem.libretexts.org/Bookshelves/Analytical_Chemistry). Treat undeclared pages as Q. Each book has a "Detailed Licensing" page. | Catalogue API, no key: `https://commons.libretexts.org/api/v1/commons/catalog` returns all 4054 books as JSON with `bookID`, `title`, `author`, `library`, `license` (`ccby`, `ccbysa`, `ccbyncsa`, empty), `links.online`, `links.pdf`, `links.zip` (all pages as HTML zip via `downloads.libretexts.org/api/v1/download/<bookID>/pages`), `libraryTags`, `summary`. The `search=` parameter I tried was ignored; filter client-side. The MindTouch content API (`/@api/deki/pages/<id>/contents`, `/subpages`) requires a developer token; `/@api/deki/pages/<id>/tags` works anonymously. | Very mixed: OpenStax mirrors, faculty course readers, some excellent originals (Harvey's Analytical Chemistry, Fleming's Physical Chemistry). Reads as a wiki. Judge per book, not per library. | Fetch the page URL with a browser User-Agent and take `<section class="mt-content-container">` (verified: 61 such sections on the stats page above; the first is the body). Subpage list: parse the page's table of contents links, or download the whole book zip from `links.zip`. | Rich but uneven. Most math and science books have a `N.E: ... (Exercises)` page per chapter with numbered exercises and inline `Answer` reveals (verified: Introductory Statistics 1e chapter 1 exercises page has 35 exercises and 17 answer blocks). Same per-page licence as the book. Separate ADAPT homework system (WeBWorK/H5P) needs an account. Arrives as HTML in the same `mt-content-container`. |
| Open Textbook Library (open.umn.edu/opentextbooks) | ~1600 peer-reviewed open textbooks, all subjects; strong in business, CS, math, social sciences, humanities | **A (per book).** Catalogue field `license` holds the CC variant as text (`Attribution`, `Attribution-ShareAlike`, `Attribution-NonCommercial-ShareAlike`, `Attribution-NoDerivs`, `Free Documentation License (GNU)`). Verified by reading https://open.umn.edu/opentextbooks/textbooks.json. Treat `NoDerivs` as Q. | Keyless JSON: `textbooks.json?page=N` (10 per page), `textbooks.json?q=<terms>` (search), `subjects.json` (tree with counts), `subjects/<id>.json` (books in a subject). Each record has `formats[]` with `type` (PDF, Online, eBook, XML, ODF, LaTeX, Hardcopy) and `url` pointing at the publisher, plus reviews with ratings. OTL hosts nothing itself. | A catalogue, not a corpus; the reviews are useful for ranking. | Follow `formats[].url` to the publisher (usually Pressbooks, a PDF, or GitHub) and use that source's fetch method. | None itself; the record's `formats[]` leads to the book. Reviews often say whether exercises and answers exist. |
| BCcampus B.C. Open Collection (collection.bccampus.ca) | Canadian college curriculum; sciences (205 records), business, trades, adult basic education | **A (per book).** Site footer "This site is licensed as CC-BY except where otherwise noted" (https://collection.bccampus.ca/, read with camoufox). Per-record licence shown on each record page. | Cloudflare blocks curl and WebFetch; needs a real browser. No API found (`/wp-json/` returns the HTML site). Most books are on BCcampus's Pressbooks network (opentextbc.ca), which does have the Pressbooks REST API below. | Solid, conventional. Overlaps OTL heavily. | Search OTL instead (it indexes the same books), then hit the Pressbooks API on opentextbc.ca. | None itself; per book, via Pressbooks. |
| Pressbooks networks (opentextbc.ca, press.rebus.community, *.pressbooks.pub, viva.pressbooks.pub, uen.pressbooks.pub, and hundreds more) | Whatever the institution published; Rebus has the Introduction to Philosophy series (CC BY), VIVA has Open Music Theory (CC BY-SA) | **A (per book and per chapter).** Licence is in the book metadata and in each chapter's `metadata.license` (`{url, name}`), verified at https://press.rebus.community/intro-to-phil-logic/wp-json/pressbooks/v2/toc (CC BY 4.0). | Per-book REST API, keyless: `<book>/wp-json/pressbooks/v2/toc` (front matter, parts, chapters with id, slug, word_count, link, licence), `<book>/wp-json/pressbooks/v2/metadata`, `<book>/wp-json/pressbooks/v2/chapters/<id>` (chapter body as rendered HTML; endpoint verified to exist, body field unverified), and `<book>/open/download?type=epub|pdf|xhtml`. Some networks sit behind CloudFront and reject curl's default User-Agent; a browser UA or camoufox works. Pressbooks Directory (pressbooks.directory, ~9000 books) is an Algolia web app; `api.pressbooks.com` is a JavaScript app with no documented endpoints, so directory search is not scriptable without reverse engineering. | Institutional OER; quality varies from excellent (Rebus philosophy) to thin course packs. Word counts in the TOC help filter out the thin ones. | `GET <book>/wp-json/pressbooks/v2/toc`, pick a chapter id, `GET <book>/wp-json/pressbooks/v2/chapters/<id>`, take `content.rendered`. | Per book: exercises live inside chapter HTML when the author wrote them; H5P interactive items (MCQ with feedback) are common on newer books and are counted in the directory metadata. Answers usually inline or absent; no separate keys. Same licence as the chapter (`metadata.license`). |
| OER Commons (oercommons.org) | Aggregator: K-12 through college, all subjects, lesson-level items | **A (per item).** Facets `CC BY`, `CC BY-SA`, `CC BY-NC`, `CC BY-NC-SA`, "Read the Fine Print", "Some rights reserved" on https://oercommons.org/browse?f.search=bayesian. Site-level content is CC BY-NC-SA 4.0. | API exists but every request needs a `token` obtained by emailing info@oercommons.org (http://docs.oercommons.org/api/). Endpoint `https://www.oercommons.org/api/search?f.search=<terms>&batch_start=0&batch_size=20`, filters `f.keyword`, `f.abstract`, `f.material_types`, `f.license`. Returns metadata records only; content lives on the original site. Without a token the browse page HTML is scrapeable. | Aggregator noise; many lesson plans and worksheets. Useful for finding things the other catalogues miss, not as a first stop. | Follow the record's external URL. | Aggregates lesson-level items including assessments (`Material Type: Assessment` facet); per-item licence. |
| MERLOT (merlot.org) | Aggregator, higher education, all subjects | Per item; MERLOT itself asserts nothing. | Web services require a licence key from webmaster@merlot.org (https://info.merlot.org/merlothelp/MERLOT_Technologies.htm, secondary). Not worth the correspondence; OTL and OER Commons cover the same books. | Catalogue only. | Rejected for the pipeline, see section 5. | Catalogue only. |
| Saylor Academy (learn.saylor.org) | Full college courses: CS, math, economics, business, history, philosophy, English | **A.** "Excluding course final exams, content authored by Saylor University is available under a Creative Commons Attribution 3.0 Unported license. Third-party materials ... shared under various licenses" (footer of https://learn.saylor.org/). | Moodle site; course pages are HTML, mostly links to third-party OER with Saylor-written framing. The legacy Flat World Knowledge textbooks live on GitHub: 91 repos `saylordotorg/text_*` (e.g. `text_introductory-statistics`), one HTML file per section (`s05-01-basic-definitions-and-concepts.html`) with a `s00-license.html` per book (those books are CC BY-NC-SA 3.0 per their licence pages; unverified this session). | Saylor framing is thin; the old Flat World books are competent 2010-era mainstream textbooks. | `raw.githubusercontent.com/saylordotorg/text_<book>/master/s<NN>-<MM>-<slug>.html`; read `s00-license.html` first. | Unit review quizzes and practice questions in Moodle (HTML, with answers). **Licence differs**: "Excluding course final exams" from the CC BY 3.0 grant, so finals are not usable; the review quizzes are. The Flat World `text_*` books end each section with Key Takeaways and Exercises, no answer keys. |
| Wikibooks (en.wikibooks.org) | Programming languages (Haskell, C++, Python), math, physics, languages, cookbooks | **A.** "Most of Wikibooks' text is dual licensed under the Creative Commons Attribution-ShareAlike 4.0 International License and the GNU Free Documentation License" (https://en.wikibooks.org/wiki/Wikibooks:Copyrights). | MediaWiki API, keyless: `https://en.wikibooks.org/w/api.php?action=parse&page=Haskell/Getting_set_up&prop=wikitext&format=json` returns wikitext (verified); `prop=text` returns HTML; `action=query&list=search&srsearch=<terms>` searches. | Uneven and often unfinished; the Haskell book is the standout. | As above, one page per section. | Some books have exercises with collapsible solutions (the Haskell book's Recursion chapter has three exercise blocks and one solution block); same CC BY-SA 4.0. Arrives in the wikitext. |
| OpenIntro (openintro.org) | Statistics: OpenIntro Statistics 4e, Introduction to Modern Statistics 2e, Advanced High School Statistics, biostatistics | **A.** "Most of OpenIntro's resources, including the Statistics textbooks, are released under a Creative Commons BY-SA 3.0 license" (https://www.openintro.org/license/); IMS states CC BY-SA 3.0 at https://openintro-ims.netlify.app/. | OpenIntro Statistics: LaTeX source on GitHub (link from https://www.openintro.org/book/os/). IMS: Quarto/R Markdown source on GitHub (link from the IMS site). PDFs free. | Clean, careful, frequentist; the best-edited open stats prose. IMS is the modern one (tidyverse, simulation first). | Clone the repo; one `.qmd` or `.tex` file per chapter. | Rich. End-of-chapter exercises in every chapter; IMS has an Appendix A with solutions for every chapter (verified at https://openintro-ims.netlify.app/exercise-solutions; whether it covers only odd-numbered items is unverified, that is OpenIntro's convention). Same CC BY-SA 3.0. Arrives in the Quarto/LaTeX source and the appendix. Instructor solutions manual separate, login. |
| Open Logic Project (openlogicproject.org) | Mathematical logic: sets, first-order logic, completeness, computability, incompleteness, modal logic, set theory | **A.** CC BY 4.0 (README of https://github.com/OpenLogicProject/OpenLogic; about page at https://openlogicproject.org/about/). Companion forall x: Calgary is CC BY 4.0 (https://forallx.openlogicproject.org/). | LaTeX, modular: "Topics are divided into section-sized chunks" in the repo's `content/` tree, designed for remixing. Built PDFs (Sets, Logic, Computation; Incompleteness and Computability; Boxes and Diamonds) on the site. | Rigorous, dry, textbook-formal; excellent as a correctness backbone, needs a voice layered on. | `content/<part>/<chapter>/<section>.tex` from the repo. | Problems inline in the LaTeX (`\begin{prob}`), no solutions published. forall x: Calgary has "Exercises with solutions" in the PDF only, "solutions are not yet included in the HTML version" (verified on https://forallx.openlogicproject.org/). CC BY 4.0. |
| AIM Open Textbook Initiative (textbooks.aimath.org) | Curated list of approved open math textbooks by course | Per book; the list itself is an index (https://textbooks.aimath.org/textbooks/approved-textbooks/). | Static HTML list. Most entries are PreTeXt or LaTeX on GitHub. | The best quality filter for open math: editorial board reviewed. | Follow the entry to the book's own site. | Index only; its evaluation criteria require exercises, so listed books have them. |
| DOAB and OAPEN (directory.doabooks.org, library.oapen.org) | Open-access scholarly monographs: humanities, social science, some STEM collections | **A or Q per book.** Licence in metadata (`publisher.oalicense`, `dc.rights`); many are CC BY 4.0, many MDPI collections are CC BY-NC-ND (verified on the search response). | Keyless REST: `https://directory.doabooks.org/rest/search?query=bayesian&expand=metadata` returned 100 items with `dc.title`, `dc.description.abstract`, `dc.subject.classification` (Thema codes), `oapen.identifier.doi`, `dc.identifier.uri`. OAPEN mirror: `https://library.oapen.org/rest/search?query=bayesian&expand=bitstreams` gives `bitstreams[]` with the PDF and an extracted `.pdf.txt` plain-text file (`/rest/bitstreams/<uuid>/retrieve`). | Academic monographs; specialist, not introductory. Good for the history, philosophy and economics side once the reader is past a primer. | `GET https://library.oapen.org/rest/bitstreams/<uuid>/retrieve` for the `.txt`; split on chapter headings. | Monographs, rarely any exercises. |
| Open Book Publishers (openbookpublishers.com) | Humanities and social science monographs and a few textbooks (e.g. Learning Statistics with jamovi) | **A (mostly).** Site "Except where otherwise noted, content on this site is licensed under a Creative Commons Attribution 4.0 International license" (https://www.openbookpublishers.com/about/open-access-policy, read with camoufox); each book states its own licence, some are BY-NC-ND. | Site is behind a Vercel bot check; the books are also in DOAB/OAPEN with the same metadata and PDFs. | Scholarly but readable; better prose than most OER. | Go through OAPEN. | Textbook titles (e.g. Learning Statistics with jamovi) carry exercises; monographs do not. |
| Project Gutenberg (gutenberg.org) | 70,000+ public-domain texts: Euclid (Casey's edition), Darwin, Faraday, Newton's Principia (Motte), Galileo, Mill, Hume, Smith, Gibbon, Herodotus, the whole pre-1929 canon | **A.** Texts are public domain in the US; the Project Gutenberg Licence governs only use of the trademark and applies if you keep the PG header, so strip the header and footer and the text is unencumbered (https://www.gutenberg.org/policy/license.html). | Keyless: Gutendex API `https://gutendex.com/books?search=euclid%20elements` (verified, returns id, authors, subjects, bookshelves, formats with URLs); bulk catalogue `https://www.gutenberg.org/cache/epub/feeds/pg_catalog.csv` (verified, columns Text#, Type, Issued, Title, Language, Authors, Subjects, LoCC, Bookshelves). Plain text at `https://www.gutenberg.org/ebooks/<id>.txt.utf-8` (verified for 2009, Origin of Species), HTML at `.../ebooks/<id>.html.images`, EPUB at `.../ebooks/<id>.epub3.images`. Some math titles (21076, Euclid) have no plain text, only HTML and PDF. | Primary sources in the author's own voice. Exactly what the history-of-science side wants. | Fetch the `.txt.utf-8`, cut at the `*** START OF` and `*** END OF` markers, split on chapter headings. | None as such; Euclid's propositions and the old grammars' exercise sets (with keys in some) are public domain. |
| Standard Ebooks (standardebooks.org) | Curated, proofread public-domain literature and some non-fiction | **A.** US public domain; Standard Ebooks dedicates its own work to the public domain (https://standardebooks.org/about). | EPUB and XHTML per book; GitHub org `standardebooks` holds the source XHTML one file per chapter. OPDS feed for the catalogue. | The best-edited public-domain texts; fewer scanning errors than Gutenberg. | Clone the book's repo; `src/epub/text/chapter-N.xhtml`. | None. |
| Wikisource, Perseus Digital Library | Wikisource: transcribed public-domain works. Perseus: Greek and Latin texts with translations | Wikisource: public-domain texts, contributions CC BY-SA (policy page https://en.wikisource.org/wiki/Wikisource:Copyright_policy; version detail unverified). Perseus texts: "This work is licensed under a Creative Commons Attribution-ShareAlike 3.0 United States License. An XML version of this text is available for download" (verified on https://www.perseus.tufts.edu/hopper/text?doc=Perseus:text:1999.01.0133). **A.** | Wikisource: MediaWiki API as for Wikibooks. Perseus: TEI XML download per text; Perseus also publishes source on GitHub (PerseusDL). | Primary sources. | MediaWiki `action=parse`; Perseus XML link on each text page. | None. |
| Internet Archive (archive.org) | Scanned books, OCW mirrors, lecture recordings | Per item; public-domain scans are free, other items carry their own licence. | Keyless search: `https://archive.org/advancedsearch.php?q=<lucene>&fl[]=identifier&fl[]=title&rows=50&output=json` (verified). Item files at `https://archive.org/download/<identifier>/`. OCR text files (`_djvu.txt`) exist for most scans. | Raw OCR; needs cleaning. | `GET https://archive.org/download/<id>/<id>_djvu.txt`. | Scanned problem books and old exam collections exist; OCR quality varies. |
| HathiTrust | Public-domain scans from research libraries | Unverified; the terms page is behind Cloudflare and was not read. Full-text download of Google-digitised public-domain volumes is restricted to page-at-a-time for most users (from memory, unverified). | Data API needs a key. | Not needed given Gutenberg, Standard Ebooks and Internet Archive. | Rejected, section 5. | Not used. |

### 2b. Open courseware and MOOCs

| Source | Subjects | Licence and verdict | Formats and access | Voice and quality | Fetch one chapter | Question sets |
|---|---|---|---|---|---|---|
| MIT OpenCourseWare (ocw.mit.edu) | 2500+ courses: mathematics (18.05 probability and statistics, 18.06 linear algebra, 18.650 statistics, 6.042J math for CS), EECS (6.006, 6.824, 6.828), physics, economics, history, philosophy, linguistics, literature | **A.** CC BY-NC-SA 4.0 for the site as a whole (https://ocw.mit.edu/pages/privacy-and-terms-of-use/). Third-party readings inside a course are excluded. | Per course, keyless: `https://ocw.mit.edu/courses/<slug>/data.json` (verified: course_description, topics, learning_resource_types, instructors, level, year, hide_download) and `https://ocw.mit.edu/courses/<slug>/pages/<page>/data.json` (verified for `syllabus`: `content` is an HTML fragment). Whole-course zip at `https://ocw.mit.edu/courses/<slug>/download/` (verified 200). Lecture notes are usually PDF; video lectures have transcripts (PDF and VTT). Search API (MIT Learn, keyless): `https://api.learn.mit.edu/api/v1/learning_resources_search/?q=<terms>&platform=ocw&limit=10` (course level, with `content_files[]` naming lecture videos and notes) and `https://api.learn.mit.edu/api/v1/content_file_search/?q=<terms>&platform=ocw&limit=10` (file level: pages, PDFs, videos, with `url`, `content_title`, `content_type`, `run_title`, `course_number`). The older Elasticsearch proxy `POST https://open.mit.edu/api/v0/search/` also still answers. GitHub org `mitodl` has the parsers. | The single best source of adaptable prose with a human voice, because the transcripts are people talking. 18.05 (Orloff and Bloom) has short written "class prep" readings that are exactly primer-shaped. Lecture-note PDFs vary from typeset books (6.042J's Mathematics for Computer Science, whose PDF states CC BY-SA 3.0, unverified this session) to scanned handwriting. | For a written page: `GET .../pages/<page>/data.json` and take `content`. For a PDF resource: the `url` from `content_file_search`, then extract text. For a lecture: `.../resources/<lecture-slug>/` page lists the transcript PDF. | The richest source on this list. Course `learning_resource_types` include `Problem Sets`, `Problem Set Solutions`, `Exams`, `Exam Solutions`, `Activity Assignments with Examples` (verified for 18.05 Spring 2022 data.json); 18.05 also has in-class problems with solutions and reading questions. `content_file_search` results carry `content_feature_type` (Lecture Notes, Problem Sets, Readings, Assignments, Problem-solving Notes seen) but filtering on it returned zero, so filter client-side. Same CC BY-NC-SA 4.0 as the notes. Arrives as PDF per set, solutions as a separate PDF; the whole-course zip has both. |
| Stanford Engineering Everywhere (see.stanford.edu) | Ten 2008-era courses: CS229 machine learning (Ng), CS106A/B, CS107, CS143 compilers, EE261 Fourier, EE263 linear systems, EE364A convex optimisation | **A.** "A Creative Commons license allows for free and open use, reuse, adaptation and redistribution" (https://see.stanford.edu/); the specific licence, CC BY-NC-SA 3.0, is stated on the course materials and in press coverage but not on the pages I could fetch (unverified on site). | Static site. Each course page links lecture transcripts (HTML and PDF), handouts and problem sets as PDF (`/materials/aimlcs229/*.pdf`, verified links). No API; the site is small enough to crawl. | Andrew Ng's CS229 transcripts are a well-known teaching voice; Boyd's EE364A transcripts likewise. Dated in ML terms (2008) but the math has not moved. | Fetch the course page, follow the transcript link for lecture N. | Problem sets with full solutions as PDFs (verified links on the CS229 page: `problemset1.pdf` through `problemset4.pdf`, `ps1_solution.pdf`, `ps2_solution.pdf`, plus data zips); midterm and final where the course had them. Same licence as the course. |
| Harvard CS50 (cs50.harvard.edu) | CS50x, CS50 Python, CS50 AI, CS50 Web, CS50 SQL, CS50 Cybersecurity | **A.** CC BY-NC-SA 4.0 (https://cs50.harvard.edu/x/license/). | Lecture notes are HTML at `https://cs50.harvard.edu/x/notes/<week>/` (verified 200); slides and source on GitHub org `cs50`. Videos on YouTube (not needed; the notes are the transcripts, lightly edited). | Malan's lecture voice, transcribed and edited; very intro-level for Matt but a clean example of "contract, concrete artifact, then mechanism". | `GET https://cs50.harvard.edu/x/notes/<week>/` and take the article body. | Problem set specifications per week at `cs50.harvard.edu/x/psets/<n>/` (verified 200), auto-graded by check50; no published solutions (academic honesty policy). Lecture notes have no questions. Specs are usable as project prompts, not as check items. |
| Carnegie Mellon OLI (oli.cmu.edu) | Statistics, logic, computing, biology, chemistry, French, economics | Site footer "Licensed under CC BY-NC-SA 4.0" (https://oli.cmu.edu/courses/). Courses themselves are delivered inside the Torus platform behind an account, some at a price ($0 to $350 shown), and their per-course licence is stated inside. **A in principle, R in practice.** | No content API; no bulk export without an instructor account. | OLI's learning-science design is what Chiron's pretest/remediation loop imitates, but the prose is locked in the platform. | Not fetchable by a pipeline. Rejected, section 5. | The best formative-question design in OER (every page has Learn By Doing and Did I Get This items with per-option feedback) and entirely locked inside Torus. Not liftable. |
| The Open University OpenLearn (open.edu/openlearn) | 1000+ short courses: science, maths, technology, history, languages, philosophy, economics; has "Introduction to Bayesian statistics" | **A.** CC BY-NC-SA (FAQ read through WebFetch at https://www.open.edu/openlearn/about-openlearn/frequently-asked-questions-on-openlearn; version 4.0 unverified). Third-party material inside a course is excluded and cannot be used standalone. | Cloudflare blocks curl, WebFetch and even camoufox on course pages (only the cookie banner rendered). OpenLearn courses are known to offer Word, PDF, ePub, SCORM and OU XML downloads on each course page (unverified this session). | The OU writes for adult distance learners: patient, well-sequenced, British. Voice is institutional but warmer than OpenStax. | Needs a real browser session that accepts cookies; then the "Download this course" links. Low priority until that is scripted. | Self-assessment questions with reveal-answer blocks inside most courses (unverified this session, from memory); same CC BY-NC-SA; arrives in the OU XML download if that can be reached. |
| NPTEL (nptel.ac.in) | Indian IIT/IISc lectures: every engineering discipline, mathematics, CS, some humanities | Footer reads "Distributed under Creative Commons Attribution-ShareAlike CC BY - NC - SA" (https://nptel.ac.in/, internally inconsistent; treat as CC BY-NC-SA, **A**). | Course pages are HTML (`https://nptel.ac.in/courses/<id>`, verified 200); each lecture has a transcript PDF; videos on YouTube. No API found; my Internet Archive search for an `nptel` collection returned nothing. | Spoken lecture transcripts, often lightly edited from speech; heavy on derivation, light on motivation. Useful for depth variants in engineering subjects. | Course page, lecture list, transcript PDF link. | Weekly assignments with answer keys are published for the online-course versions (unverified; the course page did not render for curl). PDF per week. |
| Open Yale Courses (oyc.yale.edu) | ~40 courses: philosophy (Death, PHIL 176), history (Roman, Early Modern England, Civil War), literature, economics (Shiller's financial markets, game theory), physics, astronomy, psychology | **A.** "Most of the lectures and course material within Open Yale Courses are licensed under a Creative Commons Attribution-Noncommercial-Share Alike 3.0 license" (https://oyc.yale.edu/terms). | Static HTML: each lecture page has a full transcript in the page (verified: `<h1 id="transcript-top">` on https://oyc.yale.edu/philosophy/phil-176/lecture-1). Audio and video downloadable. | Excellent spoken voice for the humanities side (Kagan, Freeman, Shiller). This is the humanities counterpart of OCW transcripts. | `GET https://oyc.yale.edu/<dept>/<course>/lecture-<n>` and take the transcript div. | Exams and paper prompts on some course pages, no answers; the transcripts contain many posed questions. Thin for question banks. |
| MIT Open Learning Library (openlearninglibrary.mit.edu) | MITx-derived interactive versions of OCW courses (18.05, 6.0001, 8.01) | Terms of service has a Creative Commons section (https://openlearninglibrary.mit.edu/tos); the specific licence is CC BY-NC-SA per OCW's description, unverified in the ToS text. | Open edX instance; content needs an account; no scriptable export. | Same material as OCW with problem checkers. | Use OCW instead. | Reading questions and problem checkers with answers, but behind an account inside Open edX. Use the OCW copies of the same problem sets. |
| Khan Academy | K-12 through early college math, science, economics, humanities | CC BY-NC-SA for videos and exercises (support article https://support.khanacademy.org/hc/en-us/articles/202262954, read with camoufox; version not stated, 3.0 from memory). Articles' licence is stated only via the ToS "Proprietary Materials; Licenses" section, which I could not extract. **A for video transcripts, uncertain for articles.** | Cloudflare on every page; the public API was retired years ago (unverified). | Below Matt's level for almost everything. | Rejected for the pipeline, section 5. | Enormous exercise bank with hints, but R and no API. |
| Google Machine Learning Crash Course and Google developer docs | ML basics, TensorFlow, Android, Go, web | **A.** "Google Developers documentation is largely licensed under Creative Commons Attribution 4.0" (https://developers.google.com/terms/site-policies); check the page footer for exceptions. | HTML; no bulk source. | Corporate tutorial voice; fine for a glossary, not for a book. | Fetch the page; low priority. | Each module ends with "Test your knowledge" MCQs with explanations and interactive exercises (verified module listing on https://developers.google.com/machine-learning/crash-course/linear-regression). CC BY 4.0. Arrives as HTML with the answers in collapsible blocks. |
| Microsoft "For Beginners" curricula (ML-For-Beginners, Data-Science-For-Beginners, generative-ai-for-beginners, AI-For-Beginners) | Applied ML and data science at intro level | **A.** MIT licence for the whole repo (https://raw.githubusercontent.com/microsoft/ML-For-Beginners/main/LICENSE). | Markdown lessons in numbered folders on GitHub. | Corporate, lesson-plan shaped, lots of quizzes; usable for question banks. | `raw.githubusercontent.com/microsoft/ML-For-Beginners/main/<n>-<topic>/<m>-<lesson>/README.md`. | Pre- and post-lecture MCQ quizzes for all 52 lessons in one JSON file, `quiz-app/src/assets/translations/en.json`, with `questionText`, `answerOptions[]` and `isCorrect` (verified); no explanations per option. MIT licence. The easiest MCQ bank to ingest mechanically, though shallow. |
| Hugging Face course (huggingface.co/learn) | Transformers, NLP, LLM fine-tuning, diffusion, audio, RL | **A.** Apache 2.0 (LICENSE of https://github.com/huggingface/course, verified via the GitHub licence API). | Markdown (MDX) chapters in `chapters/en/chapterN/*.mdx` on GitHub. | Practical library tutorial; explains the library, not the math. | `raw.githubusercontent.com/huggingface/course/main/chapters/en/chapter1/1.mdx`. | End-of-chapter quiz pages as MDX (`<Question>` components with choices and per-choice explanations, from memory; my grep of chapter 1 files did not confirm the syntax, unverified). Apache 2.0. |
| MIT Missing Semester (missing.csail.mit.edu) | Shell, editors, git, debugging, profiling, security | **A.** CC BY-NC-SA 4.0 (https://missing.csail.mit.edu/license/). | Markdown source on GitHub (`missing-semester/missing-semester`), lecture notes are the pages. | Practical, terse, written by students; Matt already knows all of it. | One markdown file per lecture. | Exercises at the end of every lecture page; a solutions page existed for the 2020 run (the URL I tried was 404, unverified). CC BY-NC-SA 4.0. |
| fast.ai Practical Deep Learning and fastbook | Deep learning, top-down | **R.** "The remainder (including all markdown cells in the notebooks and other prose) is not licensed for any redistribution or change of format or medium"; code is GPLv3 (https://raw.githubusercontent.com/fastai/fastbook/master/README.md). | Notebooks on GitHub. | Strong voice, which is why the 08-10 survey liked it, but the prose is closed. | Quotation only. | Questionnaires at chapter ends, no answers in the repo; R anyway. |
| Karpathy Neural Networks: Zero to Hero | Backprop, makemore, GPT from scratch, tokenizer | Code MIT (README of https://github.com/karpathy/nn-zero-to-hero); videos are on YouTube under YouTube's standard licence, so the spoken content is **Q**. | Notebooks and Python on GitHub; no transcripts in the repo. | Reference voice for the "single element before matrix form" rule; cite and imitate, do not copy. | Quotation only. | Exercises posed in videos and notebooks; no answer keys. |
| Stanford CS231n notes (cs231n.github.io) | Convnets, optimisation, backprop | **A.** MIT (GitHub licence API for `cs231n/cs231n.github.io`). CS224N and CS229 course notes carry no licence and are Q. | Markdown on GitHub, one file per note. | Dense, well-motivated, dated (2017) but the fundamentals are fine. | `raw.githubusercontent.com/cs231n/cs231n.github.io/master/<note>.md`. | No exercises in the notes; the course assignments (`cs231n/assignments`, unverified licence) are notebooks with starter code, no solutions. |
| Coursera, edX, Brilliant | Everything | **R.** Coursera grants "a limited, personal, non-exclusive, non-transferable, and revocable license ... only for your personal, non-commercial use" (https://www.coursera.org/about/terms). edX's terms are likewise a platform licence with no open licence (https://www.edx.org/edx-terms-service). Brilliant's terms page could not be found (404 on two URLs); it is proprietary. A handful of edX courses are also published on OCW under CC BY-NC-SA; use the OCW copy. | Behind accounts; no export. | Not sources. | Rejected, section 5. | Locked. |
| 3Blue1Brown | Linear algebra, calculus, neural networks, probability | **R.** "You may not re-upload the content"; clips under 60 seconds with attribution only (https://www.3blue1brown.com/about/). The `3b1b/videos` repo has a custom licence (GitHub reports "Other"). | Animation code on GitHub; article versions on the site under the same terms. | Reference for intuition-first ordering; imitate, do not copy. | Quotation only. | None usable. |

### 2c. Standalone books, notes and reference works

Grouped by subject. "Source format" is what a pipeline actually gets.

**Machine learning, statistics, probability**

| Source | Subjects | Licence and verdict | Formats and access | Voice and quality | Fetch one chapter | Question sets |
|---|---|---|---|---|---|---|
| Dive into Deep Learning, d2l.ai | Deep learning from linear regression to transformers, RL, GPs; code in PyTorch, JAX, TF, MXNet | **A.** CC BY-SA 4.0 for the text (https://raw.githubusercontent.com/d2l-ai/d2l-en/master/LICENSE and README); code under a modified MIT. | Markdown with executable code cells, `chapter_<topic>/<section>.md` on GitHub (verified: `chapter_linear-regression/linear-regression.md` etc.). HTML and PDF built on d2l.ai. | Workmanlike, thorough, code-heavy, weak motivation; the 08-10 survey used it. Good backbone for depth variants. | `raw.githubusercontent.com/d2l-ai/d2l-en/master/chapter_<topic>/<section>.md`. | Exercises at the end of every section (`## Exercises`, verified: 8 items in linear-regression.md), no official solutions; discussion-forum answers only. Same CC BY-SA 4.0. Arrives in the section markdown. |
| Bayes Rules! (Johnson, Ott, Dogucu) | Applied Bayesian modelling in R: priors, MCMC, regression, hierarchical models | **A.** CC BY-NC-SA 4.0 (https://www.bayesrulesbook.com/preface, "License" section). | bookdown site; R Markdown source on GitHub (repo name unverified). | Conversational, quiz-driven, concrete; the closest existing thing to a Chiron chapter on Bayes. R-centric. | Fetch the chapter HTML page or the `.Rmd` from the repo. | Rich. "Quiz yourself" items inside sections with answers at the chapter end, and a numbered exercise set per chapter split into conceptual and applied (verified: 8 exercises found in chapter 1 HTML). No published solutions (instructor manual separate, unverified). Same CC BY-NC-SA 4.0. Arrives in the chapter HTML or Rmd. |
| Think Bayes 2e, Think Stats 3e, Think Python 3e (Downey, Green Tea Press) | Bayesian statistics in Python; exploratory statistics; Python | **A.** Think Bayes 2e: CC BY-NC-SA 4.0 (https://allendowney.github.io/ThinkBayes2/). Think Python 3e: attribution, share-alike, non-commercial, i.e. CC BY-NC-SA (https://greenteapress.com/wp/think-python-3rd-edition/; version not shown in the text I read). Think Stats 3e: unverified, presumably the same. Open Textbook Library lists Think Bayes 1e as "Attribution-NonCommercial". | Jupyter notebooks, one per chapter, on GitHub (`AllenDowney/ThinkBayes2`, `AllenDowney/ThinkPython`, `AllenDowney/ThinkStats`). | Downey's voice: short sentences, computation before formula, one idea per chapter. An excellent primary text for a Bayes book. | `raw.githubusercontent.com/AllenDowney/ThinkBayes2/master/notebooks/chap<NN>.ipynb`. | Rich. Think Bayes 2e notebooks carry exercises with solutions in the same notebook (verified chap02.ipynb: 4 exercises, 5 solution cells). Think Python 3e has a `ThinkPythonSolutions` directory and a solutions notebook zip in the repo (verified listing). Same CC BY-NC-SA. The cleanest constructed-response items with worked answers in the ML/stats group. |
| Probabilistic Programming and Bayesian Methods for Hackers (Davidson-Pilon) | Bayesian inference via PyMC, MCMC intuition | **A.** MIT (LICENSE.txt in the repo, verified). | Jupyter notebooks per chapter on GitHub. | Enthusiastic, hacker-voiced, occasionally sloppy; PyMC2/3 era. | `Chapter<N>_*/Ch<N>_*.ipynb` from the repo. | A few exercises in notebooks; no answer keys. |
| OpenIntro Statistics and IMS | see 2a | **A.** CC BY-SA 3.0. | LaTeX, Quarto. | Clean frequentist grounding. | see 2a. | Rich. End-of-chapter exercises in every chapter; IMS has an Appendix A with solutions for every chapter (verified at https://openintro-ims.netlify.app/exercise-solutions; whether it covers only odd-numbered items is unverified, that is OpenIntro's convention). Same CC BY-SA 3.0. Arrives in the Quarto/LaTeX source and the appendix. Instructor solutions manual separate, login. |
| Learning Statistics with R (Navarro) | Psychology-flavoured intro statistics, R | **A.** Open Textbook Library record says "Attribution-ShareAlike" (found via `textbooks.json?q=bayesian`); the book site states CC BY-SA 4.0 (unverified this session). | bookdown HTML; LaTeX/Rmd source on GitHub. | Funny, personal, honest about statistics' warts; one of the best-voiced open stats books. Has a Bayesian chapter. | Chapter HTML from learningstatisticswithr.com. | No exercises (from memory, unverified). |
| MIT 18.05 readings (Orloff, Bloom) | Probability, Bayesian inference, frequentist inference, regression | **A.** CC BY-NC-SA 4.0 (OCW). | PDF per class, e.g. "Reading 20: Comparison of frequentist and Bayesian inference" found by `content_file_search?q=bayesian inference`. | Short, precise, example-first; primer-shaped. | `url` from the search hit, then PDF text extraction. | Problem sets with solutions, in-class problems with solutions, exams with solutions, all CC BY-NC-SA 4.0, PDFs on OCW; the single best Bayesian question bank available. |
| Stan User's Guide and reference manuals | Bayesian modelling patterns, MCMC diagnostics | **Q.** Text is CC BY-ND 4.0, code BSD 3-clause (https://mc-stan.org/docs/stan-users-guide/index.html). No derivatives. | HTML and PDF. | Authoritative on the practice of Bayesian workflow. | Quotation only. | None. |
| Bayesian Data Analysis 3e (Gelman et al.) | Full Bayesian methodology | **Q.** "the book in pdf form, available for download for non-commercial purposes" (http://www.stat.columbia.edu/~gelman/book/, read with camoufox); no adaptation right. | PDF. | The reference. | Quotation only. | Exercises per chapter; a solutions PDF for some exercises exists on the book page (unverified). Q, so quotation only. |
| Probabilistic Machine Learning books 1 and 2 (Murphy) | All of ML, probabilistically | **Q.** "CC-BY-NC-ND license" (https://probml.github.io/pml-book/book1.html and book2.html). | PDF drafts; figure code on GitHub (MIT). | Encyclopaedic. | Quotation only. | Exercises; solutions instructor-only. Q. |
| Deep Learning (Goodfellow, Bengio, Courville) | Deep learning theory | **Q.** Free HTML online, no licence stated (https://www.deeplearningbook.org/). | HTML chapters. | Formal. | Quotation only. | Almost none; Q. |
| Reinforcement Learning: An Introduction 2e (Sutton, Barto) | RL | **Q.** Free PDF, no licence stated (http://incompleteideas.net/book/the-book-2nd.html). | PDF. | The reference; readable. | Quotation only. | Exercises inline; solutions instructor-only. Q. |
| Speech and Language Processing 3e draft (Jurafsky, Martin) | NLP, LLMs | **Q.** "Feel free to use the draft chapters and slides in your classes, print it out, whatever" (https://web.stanford.edu/~jurafsky/slp3/, August 2026 release); informal permission, no adaptation licence. | PDF per chapter. | Clear and current; the LLM chapters are good. | Quotation only. | Exercises at chapter ends, no solutions. Q. |
| Mathematics for Machine Learning (Deisenroth, Faisal, Ong) | Linear algebra, calculus, probability for ML | **Q.** "We will keep PDFs of this book freely available"; copyright Cambridge University Press (https://mml-book.github.io/). | PDF. | Compact, correct. | Quotation only. | Exercises at chapter ends; no official solutions. Q. |
| Convex Optimization (Boyd, Vandenberghe) | Convex optimisation | **Q.** "Copyright in this book is held by Cambridge University Press, who have kindly agreed to allow us to keep the book available on the web" (https://web.stanford.edu/~boyd/cvxbook/). | PDF. | The reference. | Quotation only. | Exercises; instructor solutions; an "Additional exercises" set with solutions is on the page (unverified). Q. |
| Information Theory, Inference, and Learning Algorithms (MacKay) | Information theory, Bayesian inference, neural nets | **Q.** "The book is copyright (c) Cambridge University Press" with on-screen viewing permitted (https://www.inference.org.uk/mackay/itila/book.html). | PDF. | Idiosyncratic, brilliant, exercises-first; the voice Matt would want, but closed. | Quotation only. | Many exercises with worked solutions printed inline, but Q. |
| An Introduction to Statistical Learning (James et al.) | Statistical learning, R and Python editions | **Q.** "All Rights Reserved" (https://www.statlearning.com/). | PDF. | Standard. | Quotation only. | Conceptual and applied exercises per chapter; no official solutions. Q. |
| Understanding Deep Learning (Prince) | Deep learning | **Q.** CC BY-NC-ND per the book's front matter and MIT Press listing (secondary sources; not verified on the publisher page this session). | PDF; notebooks on GitHub `udlbook/udlbook`. | Modern, figure-driven. | Quotation only. | Problems per chapter; notebooks; solutions instructor-only. Q. |
| The Little Book of Deep Learning (Fleuret) | Deep learning in 160 phone-sized pages | **A, probably.** "distributed under a non-commercial Creative Commons license" (https://fleuret.org/francois/lbdl.html); the PDF states the variant (unverified). If it is BY-NC-ND, Q. | PDF. | Terse, exact. | Check the PDF colophon first. | None. |
| Deep Learning: Foundations and Concepts (Bishop, Bishop) | Deep learning | **Q.** Free online for personal use; no terms stated on https://www.bishopbook.com/. | HTML. | Textbook-formal. | Quotation only. | Exercises; solutions for some chapters posted (page mentions "Solutions to exercises for chapters 2 to 10"). Q. |
| Neural Networks and Deep Learning (Nielsen) | Backprop, MNIST from scratch | **A.** CC BY-NC 3.0 (http://neuralnetworksanddeeplearning.com/about.html). | HTML chapters; source on GitHub (`mnielsen/neural-networks-and-deep-learning` has the code; the prose source repo is unverified). | Warm, first-principles, single-element-before-matrix; the voice the authoring spec asks for. | Chapter HTML pages `chap1.html` to `chap6.html`. | Exercises and problems inline, no solutions. CC BY-NC 3.0, so the prompts are liftable. |
| Interpretable Machine Learning (Molnar) | Interpretability methods | **A.** CC BY-NC-SA 4.0 (https://christophm.github.io/interpretable-ml-book/). | bookdown/Quarto; source on GitHub. | Practical and honest. | Chapter HTML. | None. |
| The Illustrated Transformer and Alammar's other posts | Transformers, attention, GPT-2, word2vec | **A.** CC BY-NC-SA 4.0 per post (https://jalammar.github.io/illustrated-transformer/). | Jekyll markdown on GitHub (`jalammar/jalammar.github.io`). | Visual, patient; the 08-10 survey used it. | `_posts/<date>-illustrated-transformer.md`. | None. |
| Distill.pub | Feature visualisation, momentum, attention, GNNs | Diagrams CC BY and code MIT per the FAQ (https://distill.pub/faq/); article text is stated CC BY 4.0 per article (unverified this session). **A.** | HTML with interactive JS; source on GitHub per article. | Best explanatory writing in ML; archived since 2021. | Article HTML; strip the interactive figures. | None. |
| Probability for Computer Scientists (Piech, Stanford CS109) | Probability with CS examples | **Q.** No licence file in the GitHub repo (licence API 404) and none on the site (https://chrispiech.github.io/probabilityForComputerScientists/en/index.html). | HTML. | Friendly, example-driven. | Quotation only unless a licence appears. | Practice problems with solutions on the site (unverified). Q. |
| Seeing Theory | Visual probability | Repo is Apache 2.0 (GitHub licence API for `seeingtheory/Seeing-Theory`); site is archived. **A.** | HTML and JS. | Visual, short. | Little prose to take. | None. |
| Introduction to Probability (Grinstead, Snell) | Probability | GNU FDL per the book's front matter (unverified; the AMS-era HTML page has moved). **A if confirmed.** | PDF at https://math.dartmouth.edu/~prob/prob/prob.pdf. | Classic, readable. | Confirm the licence from the PDF's first pages. | Exercises per chapter, answers to odd-numbered in the back (unverified). |

**Mathematics**

| Source | Subjects | Licence and verdict | Formats and access | Voice and quality | Fetch one chapter | Question sets |
|---|---|---|---|---|---|---|
| Algorithms (Erickson) | Algorithms, recursion, DP, graphs, flows, NP-hardness | **A.** "The textbook Algorithms ... is licensed under a Creative Commons Attribution 4.0 International license. All other lecture notes are licensed under ... Attribution-NonCommercial-ShareAlike 4.0" (https://jeffe.cs.illinois.edu/teaching/algorithms/). | PDF per chapter (LaTeX source not published). | Opinionated, funny, rigorous; one of the best-voiced open CS texts. | Chapter PDF from the page. | Huge exercise sets per chapter, no solutions: "Please do not ask me for solutions to the exercises" (verified). His CS 374 and CS 473 archive has past homeworks and exams, some with solutions, under CC BY-NC-SA 4.0 (the archive licence is stated; solution coverage unverified). Prompts are CC BY 4.0. |
| Open Data Structures (Morin) | Data structures in Java, C++, Python, pseudocode | **A.** Creative Commons Attribution, "even commercially" (https://opendatastructures.org/; version 2.5 from memory). | LaTeX source and code on GitHub (`patmorin/ods`). | Precise, plain. | `latex/<chapter>.tex` from the repo. | Exercises per chapter, no solutions. CC BY. |
| Linear Algebra (Hefferon) | First linear algebra course | **A.** GFDL or CC BY-SA 3.0 US at the reader's choice (https://hefferon.net/source.html). | LaTeX source on the site; PDF. | Proof-based but gentle; a personal voice. | LaTeX chapter files from the source archive. | Rich. Every exercise has a full worked answer in a separate answers book, `jhanswer.pdf` (verified link on the book page). Same GFDL or CC BY-SA 3.0 US. Arrives as a separate LaTeX-built PDF keyed by section and exercise number. |
| Active Calculus (Boelkins et al.) | Single and multivariable calculus, prelude | **A.** "Each text carries a Creative Commons BY-SA license" (https://activecalculus.org/; version unverified). | PreTeXt XML on GitHub (linked from the site); HTML, PDF. | Inquiry-based; activity-first, which matches the commit-first beat. | PreTeXt `source/<chapter>/<section>.ptx`. | Preview activities and activities inline; WeBWorK exercise sets with answers; activity solutions for instructors only (unverified, the site pages did not render for curl). CC BY-SA. |
| Abstract Algebra: Theory and Applications (Judson) | Groups, rings, fields, Galois | **A.** GFDL (https://judsonbooks.org/). The ODE Project is "a Creative Commons License" (variant unverified). | PreTeXt on GitHub (`twjudson/aata`). | Standard, clear. | PreTeXt section files. | Exercises per chapter; hints and answers to selected exercises in the back; Sage exercises (unverified). GFDL. |
| Basic Analysis I and II (Lebl) | Real analysis | **A.** "dual licensed under a Creative Commons Attribution-Noncommercial-Share Alike 4.0 License and Creative Commons Attribution-Share Alike 4.0 License" (https://www.jirka.org/ra/). Lebl's Notes on Diffy Qs is under the same terms (unverified). | LaTeX source on GitHub (`jirilebl/ra`); PDF, HTML. | Careful, no-nonsense. | `ch-<topic>.tex`. | 815 exercises across both volumes and, deliberately, "There is no solutions manual for the exercises" (verified). Prompts only. |
| Discrete Mathematics: An Open Introduction (Levin) | Discrete math | CC BY-SA 4.0 from memory; the site pages I fetched did not render the licence and the GitHub README does not state it (**unverified**). | PreTeXt on GitHub (`oscarlevin/discrete-book`). | Investigate-first sections; good voice. | PreTeXt files. | Investigate! prompts open every section; exercises with solutions to selected ones in the back (unverified); the repo ships WeBWorK problem sets (`dmoi4-ww.zip`, verified listing). Licence as the text. |
| Book of Proof (Hammack) | Proof writing | **Q.** CC BY-NC-ND 4.0; "does not permit altering of content for anything other than personal use" (https://richardhammack.github.io/BookOfProof/). | PDF. | Beloved intro. | Quotation only (the "personal use" carve-out arguably covers Chiron, but do not rely on it). | Exercises with solutions to odd-numbered in the book. Q. |
| forall x: Calgary (Magnus, Button, Thomas-Bolduc, Zach) | Formal logic | **A.** CC BY 4.0 (https://forallx.openlogicproject.org/). | LaTeX on GitHub; PDF, HTML, SCORM. | Textbook-plain. | LaTeX chapter files. | Exercises with solutions in the PDF (verified statement), not yet in the HTML. CC BY 4.0. |
| Open Logic Project | see 2a | **A.** CC BY 4.0. | LaTeX. | Dry, rigorous. | see 2a. | see 2a. |
| Homotopy Type Theory (Univalent Foundations Program) | HoTT | **A.** CC BY-SA 3.0 (https://raw.githubusercontent.com/HoTT/book/master/README.md). | LaTeX on GitHub. | Research-level. | LaTeX chapter files. | Exercises per chapter, no solutions. |
| Category Theory for Programmers (Milewski, PDF edition by hmemcpy) | Category theory for programmers | **A.** CC BY-SA 4.0 for the PDF, `.tex`, content and figures (https://raw.githubusercontent.com/hmemcpy/milewski-ctfp-pdf/master/LICENSE). | LaTeX on GitHub; Haskell, Scala and OCaml editions. | Blog-born, conversational, programmer-first. | `src/content/<part>/<chapter>.tex`. | Challenges per chapter, no solutions. |
| Mathematics for Computer Science (Lehman, Leighton, Meyer) | Discrete math for CS | OCW page CC BY-NC-SA 4.0; the PDF's own notice states CC BY-SA 3.0 (unverified this session). **A.** | PDF on OCW 6.042J. | Standard MIT lecture-note voice; good problems. | PDF text extraction by chapter. | Problems inline and OCW problem sets with solutions (6.042J Spring 2015 has Problem Sets with Solutions and exams, unverified this session). |
| Encyclopedia of Mathematics, ProofWiki, nLab, PlanetMath | Reference | EoM: original Springer articles stay copyrighted, additions and edits are CC BY-SA (https://encyclopediaofmath.org/wiki/Main_Page), so **Q** for most articles. ProofWiki: CC BY-SA 3.0 and GFDL, **A** (https://proofwiki.org/wiki/ProofWiki:Copyrights). nLab: no formal licence, **Q** (https://ncatlab.org/nlab/show/HomePage). PlanetMath: CC BY-SA 3.0 from memory, unverified. | MediaWiki (EoM, ProofWiki) with the standard API; nLab has Markdown source per page. | Reference tone; ProofWiki proofs are useful as depth-variant raw material. | MediaWiki `action=parse`. | ProofWiki's proofs are worked examples, no exercises. EoM none. |

**Systems, programming languages, software**

| Source | Subjects | Licence and verdict | Formats and access | Voice and quality | Fetch one chapter | Question sets |
|---|---|---|---|---|---|---|
| Structure and Interpretation of Computer Programs 2e | Programs, abstraction, interpreters | **A.** CC BY-SA 4.0 by the MIT Press (https://mitp-content-server.mit.edu/books/content/sectbyfn/books_pres_0/6515/sicp.zip/index.html); the sarabander HTML5/EPUB edition is also CC BY-SA 4.0 (https://raw.githubusercontent.com/sarabander/sicp/master/README.md). | MIT Press HTML; sarabander has Texinfo source and clean HTML on GitHub. SICP JS edition on sourceacademy.org (licence unverified). | A classic with a real voice, and adaptable. | `html/<section>.xhtml` from the sarabander repo. | Famous exercise sets per section, no official solutions; community solutions exist (community.schemewiki.org, licence unverified) but Wikibooks has none. CC BY-SA 4.0 prompts. |
| Structure and Interpretation of Classical Mechanics 2e (Sussman, Wisdom) | Lagrangian and Hamiltonian mechanics via Scheme | Free HTML from the MIT Press content server (https://mitp-content-server.mit.edu/books/content/sectbyfn/books_pres_0/9579/sicm_edition_2.zip/toc.html); the licence page was blocked (403 on mitpress.mit.edu). CC BY-SA 4.0 from memory, **unverified**. | HTML per chapter. | Sussman's voice; computational physics. | Chapter HTML. | Exercises inline, no solutions. |
| The Rust Programming Language (Klabnik, Nichols) | Rust | **A.** Dual MIT and Apache 2.0 (LICENSE-APACHE and LICENSE-MIT in https://github.com/rust-lang/book, verified via the GitHub licence API). Rust by Example is the same (README). | mdBook Markdown, `src/ch<NN>-<MM>-<slug>.md`. | Friendly, exact, an authorial voice; the model for language books. | `raw.githubusercontent.com/rust-lang/book/main/src/ch04-01-what-is-ownership.md`. | No exercises in the official text. The Brown University experimental edition adds inline quizzes with correct answers (verified on https://rust-book.cs.brown.edu/; its licence is unverified). Rustlings is a separate MIT-licensed exercise repo. |
| Pro Git 2e (Chacon, Straub) | Git | **A.** CC BY-NC-SA 3.0 (https://raw.githubusercontent.com/progit/progit2/main/LICENSE.asc). | AsciiDoc on GitHub, `book/<NN>-<topic>/sections/*.asc`. | Practical, plain. | AsciiDoc section files. | None. |
| Software Foundations (Pierce et al.) | Coq, logic, type systems, verified programming | **A.** MIT licence in `lf/LICENSE` inside https://softwarefoundations.cis.upenn.edu/lf-current/lf.tgz (verified). | Literate Coq `.v` files, one per chapter, with the prose in comments; HTML build on the site. | Precise, exercise-driven. | `lf/Basics.v` from the tarball. | Rich prompts: starred exercises inline in every `.v` file, machine-checkable. "Please do not post solutions to the exercises in a public place" (verified); solutions instructor-only. MIT licence covers the prompts. Not usable as answer-keyed check items without writing the answers. |
| Programming Language Foundations in Agda (Wadler, Kokke, Siek) | Type theory in Agda | **A.** CC BY 4.0 (https://plfa.github.io/). | Literate Agda Markdown on GitHub. | Literate, careful. | `src/plfa/part1/<Chapter>.lagda.md`. | Exercises inline (practice, recommended, stretch), no solutions. CC BY 4.0. |
| Theorem Proving in Lean 4, Mathematics in Lean | Lean | **A.** Apache 2.0 (GitHub licence API for `leanprover/theorem_proving_in_lean4`; Mathematics in Lean unverified but the same community). | Markdown/Lean source on GitHub. | Manual voice. | Chapter markdown. | Exercises at chapter ends, no solutions. |
| xv6 book (Cox, Kaashoek, Morris) | Unix-like OS on RISC-V | **A.** MIT-style permission notice (https://raw.githubusercontent.com/mit-pdos/xv6-riscv-book/xv6-riscv/LICENSE). | LaTeX on GitHub, one file per chapter. | Terse, code-anchored; the systems counterpart of SICP's clarity. | `<chapter>.tex` from the repo. | Exercises at chapter ends, no solutions; the 6.1810 labs are on the course site with autograder tests. |
| Is Parallel Programming Hard, And, If So, What Can You Do About It? (McKenney) | Concurrency, memory models, RCU | **A.** CC BY-SA 3.0 US (https://mirrors.edge.kernel.org/pub/linux/kernel/people/paulmck/perfbook/perfbook.html). | LaTeX in a git repo on kernel.org; PDF. | Kernel-developer voice, quizzes throughout. | Chapter `.tex`. | Rich. Quick Quizzes inline throughout with an answers appendix (from memory; the kernel.org tree listing did not render for curl, unverified). CC BY-SA 3.0 US. |
| Computer Networks: A Systems Approach 6e (Peterson, Davie) | Networking | **A.** CC BY 4.0 (https://raw.githubusercontent.com/SystemsApproach/book/master/README.rst); the authors ask to be told about derivatives. Companion books (5G, SDN, TCP congestion control) are the same. | reStructuredText on GitHub, `<chapter>/<section>.rst`. | Textbook-standard, well organised. | `.rst` files. | No end-of-chapter exercises in the open edition (unverified). |
| Computer Networking: Principles, Protocols and Practice (Bonaventure) | Networking | Site states "open-source ebook" (https://www.computer-networking.info/); the licence (CC BY 4.0 from memory) is **unverified**, no LICENSE via the GitHub API. | reStructuredText on GitHub (`obonaventure/cnp3`). | Fine. | `.rst` files. | Exercise chapters with some solutions (unverified). |
| The Architecture of Open Source Applications, 500 Lines or Less | Real system designs | **A.** CC BY 3.0 (https://aosabook.org/en/index.html). | HTML; source on GitHub (`aosabook/500lines`). | Practitioner essays; excellent for case-study beats. | Chapter HTML. | None. |
| Operating Systems: Three Easy Pieces (Arpaci-Dusseau) | OS | **Q.** Free chapter PDFs; no licence stated on https://pages.cs.wisc.edu/~remzi/OSTEP/. | PDF per chapter. | The best-voiced OS book; closed. | Quotation only. | Homework simulators with generated answers, projects; Q. |
| Dive into Systems (Matthews, Newhall, Webb) | Systems from C to architecture | **Q.** CC BY-NC-ND 4.0 (https://diveintosystems.org/book/preface.html). | HTML. | Solid. | Quotation only. | Exercises online; Q. |
| Crafting Interpreters (Nystrom) | Interpreters | **Q.** Prose, illustrations and site CC BY-NC-ND 4.0; code MIT (https://raw.githubusercontent.com/munificent/craftinginterpreters/master/LICENSE). | Markdown on GitHub, but not adaptable. | A great voice; closed by choice ("The words are in my voice"). | Quotation only. | Challenges per chapter, no solutions in the text; Q. |
| Beej's Guides (network programming, C, git) | Sockets, C | **Q.** CC BY-NC-ND 3.0 with a translation exception (https://beej.us/guide/bgnet/html/split/intro.html, section 1.9). | Markdown source; HTML, PDF. | Beloved voice; closed. | Quotation only. | None. |
| Nand2Tetris | Computer from NAND gates up | **A.** CC BY-NC-SA 3.0 for all site materials (https://www.nand2tetris.org/license). The book itself (MIT Press) is not free. | Project PDFs and slides. | Project-driven. | Chapter slide PDFs. | Projects with test scripts, no written questions. |
| Site Reliability Engineering, Software Engineering at Google | SRE, engineering practice | **Q.** CC BY-NC-ND 4.0 (https://sre.google/sre-book/table-of-contents/). | HTML. | Practitioner voice. | Quotation only. | None. |
| Programming Languages: Application and Interpretation (Krishnamurthi) | PL semantics via interpreters | **A.** CC BY-NC-SA (https://www.plai.org/; version unstated). | PDF, HTML, EPUB. | Teaching voice, concrete. | Chapter HTML. | Exercises inline, no solutions. |
| How to Design Programs 2e | Program design | **Q.** CC BY-NC-ND (https://htdp.org/2024-11-6/Book/index.html). | HTML. | Fine. | Quotation only. | Exercises, no solutions. Q. |
| Eloquent JavaScript (Haverbeke) | JavaScript | **A.** CC BY-NC (https://eloquentjavascript.net/; 3.0 from memory); code MIT. | Markdown source on GitHub. | Literate, good. | `<NN>_<slug>.md` from the repo. | Exercises at chapter ends with hints and full solutions (the repo has `code/solutions`, from memory; my listing showed `code/` but not the subdirectory, unverified). CC BY-NC. |
| Learn X in Y Minutes, Go by Example | Language cheat sheets | Learn X in Y: CC BY-SA 3.0 (https://learnxinyminutes.com/), **A**. Go by Example: licence link present, CC BY 3.0 from memory, unverified. | Markdown on GitHub. | Reference cards, not prose. | One file per language. | None. |
| Real World Haskell, Learn You a Haskell | Haskell | Both sites were unreachable on 2026-09-05 (no HTTP response). Real World Haskell is CC BY-NC 3.0 per HaskellWiki and Wikipedia; Learn You a Haskell is CC BY-NC-SA 3.0 from memory. Both **unverified**. | Mirrors exist; the RWH source is on GitHub (`bos/real-world-haskell` from memory). | RWH is dated (2008); LYAH is charming. | Mirror HTML. | Exercises in RWH, no solutions. |

**Physics and natural science**

| Source | Subjects | Licence and verdict | Formats and access | Voice and quality | Fetch one chapter | Question sets |
|---|---|---|---|---|---|---|
| OpenStax University Physics 1-3, College Physics 2e, Chemistry 2e, Biology 2e, Astronomy 2e | Intro science | **A.** CC BY-NC-SA 4.0 (CMS API). | CNXML, REX JSON. | Bland but complete. | see 2a. | As OpenStax above: conceptual questions, problems, additional problems, challenge problems, odd answers in the Answer Key. |
| Light and Matter, Simple Nature (Crowell) | Intro physics, calculus-based physics | **A.** "Web site and books (c) 1998-2019 Benjamin Crowell, CC-BY-SA license" (https://www.lightandmatter.com/; version unstated). | LaTeX source on GitHub (`bcrowell/lm`); PDF, HTML. | Conversational, thoughtful, with history-of-science asides; the best-voiced open physics text. | LaTeX chapter files. | Homework problems per chapter; an online answer checker (verified link) and answers to selected problems in the PDF; solutions manual for instructors (unverified). CC BY-SA. |
| David Tong's lecture notes (Cambridge) | Classical dynamics, QM, QFT, statistical physics, GR, solid state | **Q.** No licence on https://davidtong.org/teaching/. | PDF per course. | Superb; many physicists learned from them. | Quotation only. | Example sheets (problem sets) per course, no solutions. Q. |
| The Feynman Lectures on Physics (Caltech HTML edition) | Physics | **R.** "this edition is only free to read, look at and listen to online, and this posting does not transfer any right to download all or any portion of the book" (https://www.feynmanlectures.caltech.edu/, read with camoufox). | HTML, Cloudflare-protected. | The voice everyone wants; legally the most closed source on this list. | Do not fetch. Cite by chapter number. | None free; R. |
| Motion Mountain (Schiller) | Physics | CC BY-NC-ND 3.0 from memory, **unverified**. | PDF. | Idiosyncratic. | Quotation only if confirmed. | Challenges inline with solutions in the back (unverified). Q if ND. |
| Project Gutenberg science classics: Darwin (2009), Faraday's Chemical History of a Candle (14474), Galileo's Dialogues, Newton's Principia (Motte), Huxley, Maxwell's Matter and Motion, Einstein's Relativity (5001) | History of science, primary sources | **A.** Public domain. | see 2a. | Primary voices. | Gutendex search by author. | None as such; Euclid's propositions and the old grammars' exercise sets (with keys in some) are public domain. |
| Wikipedia | Everything | **A.** CC BY-SA 4.0 and GFDL (https://en.wikipedia.org/wiki/Wikipedia:Copyrights). | MediaWiki API; `action=query&prop=extracts` for plain text; `action=parse&prop=wikitext`. | Encyclopaedic, uneven, no pedagogy; use for fact-checking and glossary, not as a spine. | `action=parse&page=<title>&prop=wikitext`. | None. |

**Economics, history, philosophy, social science, arts**

| Source | Subjects | Licence and verdict | Formats and access | Voice and quality | Fetch one chapter | Question sets |
|---|---|---|---|---|---|---|
| QuantEcon lectures (Sargent, Stachurski) | Quantitative economics, dynamic programming, macro, finance, in Python and Julia | **A.** "This work is licensed under a Creative Commons Attribution-ShareAlike 4.0 International" (https://python.quantecon.org/intro.html). | MyST Markdown (Jupyter Book) on GitHub: `QuantEcon/lecture-python-intro` and `lecture-python` with `lectures/<slug>.md` and `_toc.yml` (verified layout). | Rigorous, computational, written by economists who code; ideal for Matt. | `raw.githubusercontent.com/QuantEcon/lecture-python-intro/main/lectures/<slug>.md`. | Rich. Exercises with full solutions inline in every lecture, marked with MyST `exercise-start` and `solution-start` directives (verified in `lectures/prob_dist.md`). Same CC BY-SA 4.0. Arrives in the lecture markdown; a parser can split prompt and solution mechanically. |
| OpenStax Principles of Economics 3e (micro, macro) | Intro economics | **A.** CC BY-NC-SA 4.0. | CNXML, REX JSON. | Standard. | see 2a. | Self-check questions with answers, review questions, critical thinking questions and problems; odd answers in the Answer Key. |
| CORE Econ, The Economy 2.0 | Intro economics, modern and empirical | **Q.** CC BY-NC-ND 4.0 (https://www.core-econ.org/terms-of-use/). | HTML behind a registration wall for some units. | Better than any other intro econ text; closed to adaptation. | Quotation only. | MCQs with feedback inside units; Q. |
| Saylor economics and history courses | Intro econ, US and world history | **A.** CC BY 3.0 for Saylor-authored framing. | see 2a. | Thin. | see 2a. | see 2a; finals excluded from the licence. |
| The American Yawp | US history survey | **A.** CC BY-SA 4.0, "designed to meet the standards of a Free Cultural Work" (https://www.americanyawp.com/about.html). | HTML chapters; PDF; print via Stanford University Press. | Collaborative but well edited; readable survey voice. | Chapter HTML `https://www.americanyawp.com/text/<NN>-<slug>/`. | None; a companion primary-source reader. |
| OpenStax World History 1 and 2, US History | History surveys | **A.** CC BY-NC-SA 4.0. | CNXML, REX JSON. | Bland. | see 2a. | Review questions and check-your-understanding items, answer key for some. |
| Open Yale Courses history and philosophy transcripts | see 2b | **A.** CC BY-NC-SA 3.0. | HTML transcripts. | The best humanities voice available for adaptation. | see 2b. | Exams and paper prompts on some course pages, no answers; the transcripts contain many posed questions. Thin for question banks. |
| Introduction to Philosophy series (Rebus Community, ed. Hendricks) | Logic, epistemology, ethics, philosophy of mind, of science, of religion, aesthetics | **A.** CC BY 4.0 (Pressbooks metadata at https://press.rebus.community/intro-to-phil-logic/wp-json/pressbooks/v2/toc). | Pressbooks API. | Textbook-plain, competent, short chapters. | see Pressbooks in 2a. | Questions for reflection at chapter ends, no answers (unverified). |
| Early Modern Texts (Bennett) | Descartes, Locke, Hume, Kant, Leibniz, Spinoza and others in modernised English | **Q.** Free PDFs, EPUB and MOBI; no licence statement found on https://www.earlymoderntexts.com/ or its FAQ. | PDF per text. | Bennett's modernisations are the most readable versions of these texts. | Quotation only; use Gutenberg originals for adaptation. | None. |
| Perseus Digital Library | Greek and Latin texts and translations | **A.** CC BY-SA 3.0 US per text page. | TEI XML. | Primary sources and scholarly translations (Jowett's Plato, etc.). | see 2a. | None. |
| Stanford Encyclopedia of Philosophy | Philosophy reference | **R for adaptation.** Entries are author-copyrighted and usable only per the Terms of Use; non-entry pages may not be reproduced beyond fair use (https://plato.stanford.edu/info.html). | HTML. | The best philosophy reference; cite, do not copy. | Quotation only. | None. |
| Internet Encyclopedia of Philosophy | Philosophy reference | **R for adaptation.** "our articles are not open source or in the public domain" (https://iep.utm.edu/home/copyright/). | HTML. | Good. | Quotation only. | None. |
| NOBA Project | Psychology | **A.** CC BY-NC-SA 4.0 granted on sign-up (https://nobaproject.com/license-agreement). | HTML modules; PDF. | Well-edited short modules by named researchers. | Module HTML. | Discussion questions per module in the text; an instructor test bank of MCQs behind login (the textbooks endpoint returned 401 without an account; licence of the test bank unverified). |
| Open Music Theory 2 (VIVA) | Music theory | **A.** CC BY-SA (https://viva.pressbooks.pub/openmusictheory/). | Pressbooks API. | Modern, inclusive of jazz and pop. | Pressbooks API. | A complete workbook of assignments (verified on the book page); answer keys for instructors (unverified). CC BY-SA. |
| Smarthistory | Art history | CC BY-NC-SA 4.0 from memory; site blocked by Cloudflare, **unverified**. | HTML. | Excellent essays. | Needs a browser. | None. |
| OpenStax Introduction to Philosophy, Psychology 2e, Sociology 3e, Political Science, Anthropology | Social science surveys | **A.** CC BY-NC-SA 4.0. | CNXML, REX JSON. | Bland but complete. | see 2a. | Review questions and further reading; some answer keys. |

**Language learning**

| Source | Subjects | Licence and verdict | Formats and access | Voice and quality | Fetch one chapter | Question sets |
|---|---|---|---|---|---|---|
| Foreign Service Institute courses (mirrors: fsi-languages.yojik.eu, Live Lingua) | 40+ languages, full audio courses | **A.** US government works, public domain ("Public Domain Courses" at https://fsi-languages.yojik.eu/; the Live Lingua URL I tried was 404). | PDF course books and MP3 audio; static directory listing on yojik. | 1960s-1980s drill method; dated register, rigorous grammar. Good for structured grammar units, bad for voice. | Directory listing, then the unit PDF. | Drills with answers embedded in the audio and text; public domain. |
| COERLL (University of Texas) | French (Français interactif), German (Deutsch im Blick), Spanish, Arabic, Portuguese, Yoruba, K'iche' and more | **A (per material).** Materials are tagged CC-BY through CC-BY-NC-SA, with some ND (https://coerll.utexas.edu/coerll/materials/). | HTML and PDF per site; Pressbooks for some. | University-course quality with video; the best open language materials in English. | Per material site. | Exercises with keys in several materials; per-material licence. |
| Wiktionary, Tatoeba | Dictionaries; example sentences with translations and audio | Wiktionary CC BY-SA 4.0 (Wikimedia, same as Wikipedia, not separately fetched). Tatoeba sentences CC BY 2.0 FR from memory; the terms page was fetched but the licence line was not extracted (https://tatoeba.org/en/terms_of_use), **unverified**. | MediaWiki API; Tatoeba has bulk TSV exports and an API. | Data, not prose. | Bulk export. | None. |
| Tae Kim's Guide to Japanese Grammar | Japanese | CC BY-NC-SA 3.0 from memory; only the KanjiVG credit (CC BY-SA 3.0) appeared on https://guidetojapanese.org/learn/, so **unverified**. | HTML; PDF. | Personal, logical, the standard self-study grammar. | Page HTML. | None. |
| Dickinson College Commentaries | Latin and Greek readers with vocabulary and notes | CC BY-SA 4.0 from memory; the terms-of-use page exists but was not read (https://dcc.dickinson.edu/), **unverified**. | Drupal HTML. | Scholarly reader format. | Page HTML. | None; vocabulary lists. |
| Wikibooks language books, Gutenberg grammars (D'Ooge's Latin for Beginners, 18251) | Latin, Greek, German, French | Wikibooks CC BY-SA 4.0; Gutenberg public domain. **A.** | MediaWiki API; Gutenberg text. | Wikibooks uneven; the old grammars are thorough and dated. | see 2a. | Wikibooks exercises with answers in some books; the Gutenberg grammars have exercise sets, with keys in some editions. |

**Preprints**

| Source | Subjects | Licence and verdict | Formats and access | Voice and quality | Fetch one chapter | Question sets |
|---|---|---|---|---|---|---|
| arXiv | STEM research papers, many tutorial-style surveys and lecture notes | **Per paper.** Options are CC BY 4.0, CC BY-SA 4.0, CC BY-NC-SA 4.0, CC BY-NC-ND 4.0, CC0, and the default arXiv perpetual non-exclusive licence which grants no adaptation right (https://info.arxiv.org/help/license/index.html). Most papers use the default, so most are **Q**; filter on licence. | arXiv API (`export.arxiv.org/api/query`), OAI-PMH metadata includes the licence URL; LaTeX source via `arxiv.org/e-print/<id>`; HTML via `arxiv.org/html/<id>` for recent papers. | Lecture notes and tutorials on arXiv (e.g. "Lectures on ...") are often the best deep treatment of a topic; the voice is academic. | `arxiv.org/e-print/<id>` tarball, main `.tex`. | Lecture-note style papers sometimes carry exercises, rarely solutions. |

### 2d. The best question-set sources

Ranked by how much of a Chiron check bank can be lifted mechanically:
prompt plus answer in a parseable form, under the same licence as the prose.
Chiron's `questions.yaml` wants constructed items with answers and rubrics
first, and MCQs whose distractors map to misconceptions second.

1. **MIT OCW problem sets, exams and solutions** (CC BY-NC-SA 4.0, same as
   the notes). Every quantitative course has them; 18.05 alone has problem
   sets, in-class problems and exams, all with solutions (verified in its
   `data.json`: `Problem Sets`, `Problem Set Solutions`, `Exams`, `Exam
   Solutions`, `Activity Assignments with Examples`). Form: paired PDFs.
   Fetch: `GET https://ocw.mit.edu/courses/<slug>/data.json`, then the
   course zip at `https://ocw.mit.edu/courses/<slug>/download/`, pair
   `*ps<N>*.pdf` with `*ps<N>*sol*.pdf` by filename, extract text, split
   into items with an LLM pass. Constructed responses with worked solutions.
2. **OpenStax CNXML exercises** (CC BY-NC-SA 4.0, same as the prose).
   `<exercise><problem>...</problem><solution>...</solution></exercise>`
   inside the section module, so prompt, answer and section come from one
   XML parse (verified: module m54038 of Introductory Business Statistics 2e
   has 17 exercises and 16 solutions). The built book prints odd-numbered
   answers only; the CNXML holds every solution OpenStax wrote. Fetch:
   `raw.githubusercontent.com/openstax/osbooks-<book>/main/modules/<mid>/index.cnxml`,
   module ids from `collections/<book>.collection.xml`. Instructor solution
   manuals and test banks are behind login and not under the licence.
3. **QuantEcon** (CC BY-SA 4.0). `exercise-start` and `solution-start`
   MyST directives in the lecture markdown with full solutions and code
   (verified in `lectures/prob_dist.md`). Fetch:
   `raw.githubusercontent.com/QuantEcon/lecture-python-intro/main/lectures/<slug>.md`
   and split on the directives. Economics and computational math only.
4. **Hefferon's Linear Algebra** (GFDL or CC BY-SA 3.0 US). Every exercise
   answered in a separate book, `https://jheffero.w3.uvm.edu/linearalgebra/jhanswer.pdf`
   (verified link), keyed by section and exercise number. Fetch the text
   PDF and the answer PDF, align by number.
5. **Downey's Think Bayes 2e and Think Python 3e** (CC BY-NC-SA). Exercises
   and solutions in the same notebook (verified: `notebooks/chap02.ipynb`
   has 4 exercises and 5 solution cells) or a `ThinkPythonSolutions`
   directory (verified listing). Fetch:
   `raw.githubusercontent.com/AllenDowney/ThinkBayes2/master/notebooks/chap<NN>.ipynb`,
   take cells whose source starts with `**Exercise` and the following
   `# Solution` cells.
6. **OpenIntro IMS and OpenIntro Statistics** (CC BY-SA 3.0). End-of-chapter
   exercises with an Appendix A of solutions (verified at
   https://openintro-ims.netlify.app/exercise-solutions; odd-only coverage
   unverified). Fetch the Quarto source from the GitHub repo linked on the
   IMS site, chapter `.qmd` plus the appendix, align by number.
7. **Stanford SEE problem sets with solutions** (CC BY-NC-SA 3.0). Same shape
   as OCW, ten courses; CS229 and EE364A are the useful ones. Fetch:
   `https://see.stanford.edu/materials/aimlcs229/problemset<N>.pdf` and
   `.../ps<N>_solution.pdf` (verified links).
8. **LibreTexts exercise pages** (per-page licence, mostly CC BY and
   CC BY-NC-SA). Each chapter's `N.E: <title> (Exercises)` page with inline
   `Answer` reveals for about half the items (verified: 35 exercises, 17
   answers on Introductory Statistics 1e chapter 1). Fetch the page HTML with
   a browser User-Agent and parse `section.mt-content-container`.
9. **Bayes Rules!** (CC BY-NC-SA 4.0). Quiz-yourself items with answers at
   chapter end and numbered conceptual and applied exercises without
   solutions (verified: 8 exercises in chapter 1). Fetch:
   `https://www.bayesrulesbook.com/chapter-<N>`.
10. **Google Machine Learning Crash Course** (CC BY 4.0) and **Microsoft
    ML-For-Beginners quizzes** (MIT). Ready-made MCQs: Google's "Test your
    knowledge" items carry explanations (verified module listing), Microsoft's
    52 lessons of pre- and post-quizzes sit in one JSON with `questionText`,
    `answerOptions[]` and `isCorrect` (verified), no explanations. Fetch:
    `raw.githubusercontent.com/microsoft/ML-For-Beginners/main/quiz-app/src/assets/translations/en.json`.
    Shallow, but the only open MCQ banks with machine-readable structure.

Rich prompts with no answers, still worth lifting as `constructed` items the
author LLM must solve and write a rubric for: Erickson's Algorithms
("Please do not ask me for solutions to the exercises", verified), d2l.ai
(`## Exercises` per section, verified), Software Foundations ("Please do
not post solutions", verified), Lebl's Basic Analysis ("There is no
solutions manual", verified), Nielsen, SICP, PLFA, Open Logic. The spec's
`check: llm` with a rubric makes these usable at the cost of model time per
item and a verification pass.

Sources whose exercise licence differs from the text:

- **Saylor** excludes course final exams from its CC BY 3.0 grant (verified
  footer); unit review quizzes are covered, finals are not.
- **Erickson** licenses the book CC BY 4.0 but the homework and exam archive
  CC BY-NC-SA 4.0 (verified statement on the book page).
- **OpenStax** instructor solution manuals and test banks are behind an
  instructor login with no open licence; only the CNXML is licensed.
- **NOBA**'s instructor test bank and **OLI**'s formative items sit behind
  accounts with unstated licences.
- **Rust book (Brown edition) quizzes** and **Missing Semester solutions**
  have unverified licences.
- **forall x: Calgary** solutions exist only in the PDF, not the HTML
  (verified), same CC BY 4.0.

## 3. Discovery: finding the two or three best open texts for a brief

Nothing offers full-text search across OER. The practical design is a
cascade: a curated local index first, then three keyless catalogue APIs,
then a full-text course-material search, with an LLM reading tables of
contents to make the final pick.

### 3.1 The cascade

1. **Local curated index.** Turn section 2 into a YAML file the builder ships
   with: for each source, subject tags, licence verdict, source format, and
   a fetch recipe. Roughly 130 entries. Most briefs Matt will write ("Bayesian
   statistics for an engineer", "Rust ownership", "the history of
   thermodynamics") hit this index directly, and it encodes the voice
   judgements that no API exposes. The LLM planner gets the matching entries
   and their tables of contents, not the whole index.
2. **Open Textbook Library search** for textbooks the index lacks.
3. **LibreTexts catalogue** filtered client-side on title, summary and
   licence.
4. **OpenStax book list** for scope-and-sequence and problems when the
   subject is an intro college course.
5. **MIT Learn content-file search** for lecture notes, readings and
   transcripts on the exact topic; this is the only true full-text search.
6. **DOAB/OAPEN** when the brief is humanities or a research-level
   monograph; **Gutendex** when the brief names a classic or a historical
   figure; **Wikipedia** as a last resort for a glossary.
7. Hand the planner the candidate list with tables of contents and let it
   pick two or three, one as spine and the others to interleave, with the
   authoring spec's ordering rules as constraints.

### 3.2 Endpoints, with the Bayesian brief worked through

Brief: "a short book on Bayesian statistics for an engineer". Search terms
the planner would extract: `bayesian statistics`, `bayesian inference`,
`bayes`.

**Open Textbook Library** (keyless JSON).

```
GET https://open.umn.edu/opentextbooks/textbooks.json?q=bayesian
```

Returned 8 records on 2026-09-05. Relevant ones: Think Bayes (id 288,
licence `Attribution-NonCommercial`, formats Online, PDF, LaTeX on GitHub),
Statistics for Ecologists: A Frequentist and Bayesian Treatment (id 1588,
`Attribution`, PDF), Learning Statistics with jamovi (id 1785,
`Attribution-ShareAlike`, Open Book Publishers PDF), Learning Statistics
with R (`Attribution-ShareAlike`). Each record carries `subjects[]`,
`formats[].url`, `license`, `description`, and `reviews[]` with ratings.
Paging is `?page=N`, 10 per page; the subject tree is at `subjects.json`
and a subject's books at `subjects/<id>.json`.

**LibreTexts catalogue** (keyless JSON, whole catalogue in one call).

```
GET https://commons.libretexts.org/api/v1/commons/catalog
```

Returns `{"numTotal": 4054, "books": [...]}` with `title`, `author`,
`library`, `license`, `summary`, `libraryTags`, `links.online`,
`links.pdf`, `links.zip`. Cache it (it changes weekly) and filter locally:
`stats` library plus `bayes` in title or summary, licence not empty. The
`search` query parameter I tried was ignored.

**OpenStax** (keyless JSON).

```
GET https://openstax.org/apps/cms/api/v2/pages/?type=books.Book&fields=title,license_name,license_version,book_subjects,cnx_id,book_state&limit=200
```

129 records; no Bayesian title. Introductory Statistics 2e (cnx_id
c4b474e4-4e3b-4232-b35c-2c3535bfae73) is the nearest, and only for the
probability chapters. Skip for this brief.

**MIT Learn** (keyless JSON, full text over OCW files).

```
GET https://api.learn.mit.edu/api/v1/learning_resources_search/?q=bayesian%20statistics&platform=ocw&limit=10
GET https://api.learn.mit.edu/api/v1/content_file_search/?q=bayesian%20inference&platform=ocw&limit=20
```

The first returned 13 courses, led by 18.650 Statistics for Applications
(with "Lecture 17: Bayesian Statistics" as a content file) and 18.05. The
second returned 3742 file hits; the top two were an OCW page "Bayesian
Inference in Generative Models" (RES.9-008) and the PDF "Reading 20:
Comparison of frequentist and Bayesian inference" from 18.05 Spring 2022,
each with `url`, `content_type`, `run_title`, `course_number`. Then:

```
GET https://ocw.mit.edu/courses/18-05-introduction-to-probability-and-statistics-spring-2022/data.json
```

gives the course description (which names Bayesian inference), topics
(Mathematics > Probability and Statistics) and resource types (Lecture
Notes, Readings, Problem Sets with solutions). The whole course is one zip
at `.../download/`.

**DOAB** (keyless JSON, monographs).

```
GET https://directory.doabooks.org/rest/search?query=bayesian&expand=metadata
```

100 items, mostly MDPI journal-derived collections under CC BY-NC-ND (Q)
and a few CC BY monographs. For this brief, nothing better than the
textbooks above; DOAB earns its place for briefs in history and
philosophy.

**Gutendex** (keyless JSON, classics).

```
GET https://gutendex.com/books?search=bayes
```

Bayes's 1763 essay is not on Gutenberg; Laplace's Philosophical Essay on
Probabilities (English translation) is. For a history beat, that is the
primary source.

**Planner's pick for this brief**, from the index plus the above: spine
Think Bayes 2e (CC BY-NC-SA 4.0, notebooks, Downey's computation-first
voice fits an engineer), interleave Bayes Rules! (CC BY-NC-SA 4.0,
conversational, adds regression and hierarchical models) and the 18.05
readings 10 through 20 (CC BY-NC-SA 4.0, short, exact, with problem sets
and solutions for the question bank). Quotation-only references for depth
variants: BDA3, Murphy PML1, the Stan User's Guide. OpenIntro IMS
(CC BY-SA 3.0) supplies the frequentist contrast cases the authoring spec
wants above the fold.

### 3.3 Other search surfaces, for completeness

- OER Commons: `https://www.oercommons.org/api/search?f.search=bayesian&batch_size=20&token=<token>`; token by email to info@oercommons.org. Without a token, `https://oercommons.org/browse?f.search=bayesian` is HTML with licence facets.
- MediaWiki (Wikipedia, Wikibooks, Wikiversity, Wikisource, ProofWiki): `api.php?action=query&list=search&srsearch=bayesian%20inference&format=json`, then `action=parse&page=<title>&prop=wikitext`.
- Internet Archive: `https://archive.org/advancedsearch.php?q=title%3A%28bayesian%29+AND+mediatype%3Atexts&fl[]=identifier&fl[]=title&fl[]=licenseurl&rows=50&output=json`.
- GitHub code search for source repos: `https://api.github.com/search/repositories?q=bayesian+textbook+in:readme&sort=stars` finds the notebook-based books; check each repo's LICENSE through `https://api.github.com/repos/<owner>/<repo>/license`, which returns an SPDX id.
- Pressbooks Directory: no API; the per-network `wp-json/pressbooks/v2/books` endpoint lists a network's public books, so a short list of networks (opentextbc.ca, press.rebus.community, pressbooks.pub subdomains from OTL records) can be enumerated instead.

### 3.4 Subject taxonomies worth mapping to

- Open Textbook Library `subjects.json`: two-level tree with Library of Congress call numbers (e.g. Computer Science QA76, Statistics under Mathematics), which makes it a decent pivot to other catalogues.
- OpenStax `book_subjects`: nine coarse names (Math, Science, Social Sciences, Humanities, Business, Computer Science, Nursing, College Success, High School).
- LibreTexts: the library prefix (`math`, `stats`, `phys`, `chem`, `bio`, `eng`, `socialsci`, `human`, `geo`, `med`, `biz`, `workforce`, `k12`) plus the Bookshelves path.
- MIT Learn `topics[]`: a two-level tree ("Science & Math > Mathematics"), and `departments[]` by MIT course number.
- DOAB `dc.subject.classification`: Thema codes ("P Mathematics and Science").
- Gutenberg: Library of Congress classification in `LoCC` plus free-text `Bookshelves`.

## 4. Licence handling rules for the pipeline

1. **Record provenance on every fetched chunk**: source name, URL, licence
   name and version, licence URL, fetch date, and the verdict (A, Q, R).
   Store it next to the chunk, and carry it into the unit's front matter as
   a `sources:` list. This is the whole attribution obligation for a private
   book, and it is also what makes the book auditable.
2. **A (adaptable)** means the author LLM may paraphrase, restructure,
   interleave and rewrite. Permitted licences: CC0, public domain, CC BY,
   CC BY-SA, CC BY-NC, CC BY-NC-SA (any version), GFDL, MIT, Apache 2.0,
   BSD. Share-alike is satisfied by never distributing; if a generated book
   is ever shared, every SA-derived unit inherits the strictest SA licence
   among its sources, so keep the `sources:` list accurate.
3. **Q (quotation only)** means at most a short verbatim quotation with
   citation per unit, and free use of the source's structure and its ideas
   in the author's own words. Triggers: CC BY-ND, CC BY-NC-ND, "free to
   read" with no licence, "for personal use", arXiv default licence, and
   any page whose licence the fetcher cannot find. Do not feed Q sources to
   the author as adaptation material; feed them to the planner as
   references and to the checker as ground truth.
4. **R (restricted)** means do not fetch programmatically at all. Feynman
   Lectures, fast.ai prose, 3Blue1Brown, SEP, IEP, Coursera, edX, Brilliant,
   Khan articles. Cite by title and chapter.
5. **Verify the licence on the chunk itself, not the catalogue.** OpenStax
   editions differ (CC BY 4.0 retired editions versus CC BY-NC-SA 4.0
   current), LibreTexts is per page, Pressbooks is per chapter, OCW excludes
   third-party readings, OpenLearn excludes third-party media, DOAB
   publishers mix CC BY and CC BY-NC-ND. The fetcher reads the licence
   marker on the page (LibreTexts `license:` tag, Pressbooks
   `metadata.license.url`, REX `license` field, repo LICENSE) and refuses to
   return an A chunk without one.
6. **Prefer the most permissive copy of the same text.** The CC BY 4.0
   LibreTexts mirror of an old OpenStax edition is preferable to the
   CC BY-NC-SA 4.0 current edition when the content is equivalent, and the
   Gutenberg original is preferable to a modernised Q-licensed edition.
7. **Strip platform boilerplate** (Project Gutenberg headers and footers,
   LibreTexts "This page titled ... is shared under" footers, MediaWiki
   navigation) before the chunk reaches the author, but keep the licence
   text in the provenance record.
8. **Third-party embeds inside A sources are not A.** Figures credited to
   others, quoted poems, and "used with permission" boxes must be dropped or
   quoted, not adapted. The CNXML and Pressbooks formats mark these; HTML
   scrapes do not, so the author prompt should say "do not reproduce
   figures or credited quotations from the source".
9. **Never fetch behind a login, a paywall or a bot wall with a scripted
   browser** unless Matt says so for that source. camoufox exists on the
   Mac for verification, not for bulk collection; OpenLearn, BCcampus,
   Smarthistory and Khan are behind Cloudflare and stay manual until they
   publish an API.
10. **Question sets follow the same A/Q/R verdict as their book**, with the
    exceptions listed in section 2d. Lifted items keep their source id
    (book, chapter, exercise number) in the `questions.yaml` item so a bad
    answer can be traced. Solutions the author LLM writes for prompt-only
    sources are marked as Chiron-authored, not attributed to the source.
11. **Re-verify annually.** OpenStax changed its licence in April 2026 and
    two Haskell book sites disappeared this year; the survey date at the top
    of this file is the last check.

## 5. Checked and rejected

- **MERLOT**: web services need a licence key by correspondence; its catalogue duplicates OTL and OER Commons. Not worth it.
- **Carnegie Mellon OLI**: content lives inside Torus behind accounts and sometimes a fee; no export. The design ideas are already in Chiron.
- **Khan Academy**: Cloudflare on every page, no maintained API, and the level is below Matt's for nearly everything; article licence could not be confirmed.
- **Brilliant**: proprietary; terms page not even findable (404). No open licence.
- **Coursera and edX**: platform licences for personal use only; the handful of openly licensed courses are mirrored on OCW, which is where to get them.
- **fast.ai fastbook prose**: explicitly "not licensed for any redistribution or change of format or medium". Reference only.
- **Karpathy videos, 3Blue1Brown videos**: YouTube standard licence and an explicit no-reuse policy respectively. Reference only.
- **Feynman Lectures HTML edition**: explicitly free to read only, no download right. Reference only.
- **Stanford Encyclopedia of Philosophy, Internet Encyclopedia of Philosophy**: author-copyrighted; fair-use quotation only.
- **HathiTrust**: behind Cloudflare, data API needs a key; Gutenberg, Standard Ebooks and Internet Archive cover the public domain better for this purpose.
- **Pressbooks Directory as a search API**: `api.pressbooks.com` is a JavaScript app with no documented endpoints; use OTL to find the book and the per-network REST API to fetch it.
- **LibreTexts MindTouch content API**: requires a developer token; the public HTML and the catalogue API suffice.
- **OpenLearn for automated fetching**: Cloudflare defeats curl, WebFetch and camoufox on course pages; keep it on the list for manual use only.
- **Real World Haskell and Learn You a Haskell**: both origin sites were down on 2026-09-05; if Haskell comes up, use mirrors and confirm the licence from the mirror's copy.
- **CORE Econ, Dive into Systems, Crafting Interpreters, Beej's Guides, How to Design Programs, Book of Proof, Google SRE books, Stan docs, Murphy PML, Prince UDL**: all ND licences. Good books, quotation only; listed in section 2 so nobody re-checks them.
- **nLab**: no formal licence at all. Quotation only.
- **Wikipedia as a spine**: licence is fine but there is no pedagogy in it; use for glossary and fact checks only.

## 6. The builder starts from sources (design, 2026-09-05)

What changes in `server-go/generate`: today `Plan` turns a brief into a
syllabus and `Units` writes every unit from nothing. The change is a
source-finding stage before the plan, source material fetched per unit
and handed to the author as the thing to adapt, and question banks lifted
from the sources' exercises. The brief can name the sources itself
("starting from Think Bayes and interleaving 18.05's readings"), and the
finder only runs when it does not.

Decisions (Matt, 2026-09-05): OpenStax prose is allowed as material, and
the planner chooses whichever spine it judges best; the survey's voice
notes inform that choice rather than rule it.

### 6.1 The source index

Section 2 becomes `corpus/sources/index.yaml`, shipped with the server:
one entry per source with `id`, `title`, `authors`, `subjects` (tags from
the taxonomy in 3.4), `verdict` (A, Q, R), `licence` (name, version, URL,
checked date), `voice` (one line), `format`, and a `fetch` recipe naming a
fetcher and its parameters (`github-raw` with repo, ref and path pattern;
`openstax-rex` with book uuid; `libretexts` with the book URL; `mediawiki`
with the site and page prefix; `ocw` with the course id; `gutenberg` with
the ebook id; `pressbooks` with the network and book slug; `file` with
the path of a PDF, EPUB, Markdown or text file on the server's disk, for
a book the reader owns a copy of and keeps beside the corpus: a PDF is
read through pdftotext and split into chapters at the pages that open
with "CHAPTER", a number and a title; an EPUB is read from its own
package and split at its spine, one chapter per document, titled from
its navigation document), plus a `contents` recipe for the
table of contents and an `exercises` note from 2d. Hand-curated, about 130 entries; a lint checks every A entry has a
fetch recipe and a licence URL.

### 6.2 Find

A new `roles.FindSources` runs before `Plan` when the brief names no
sources:

1. The LLM extracts three to five search terms and the subject tags from
   the brief.
2. Candidates: index entries matching the tags, then the keyless
   catalogues in the cascade of 3.1 (Open Textbook Library search,
   LibreTexts catalogue filtered on title and licence, the OpenStax list
   for intro subjects, MIT Learn content-file search for notes and
   transcripts, DOAB and Gutendex when the tags say humanities or a
   classic). Catalogue hits are provisional until the fetcher has read a
   licence marker on the text itself (rule 5).
3. The tables of contents of the best eight are fetched.
4. `roles.PickSources`: the planner chooses one spine and up to two
   interleaves, says why in a sentence each, and lists Q references it
   wants cited. Output is `sources.yaml` in the corpus directory: chosen
   sources with verdicts, licences and the reason, so the book says where
   it came from before a unit exists.

"Starting from X, interleaving Y" in the brief resolves X and Y against
the index by title or author, or against the catalogues, and skips the
rest.

### 6.3 Plan with the spine in hand

`Plan` gets the spine's table of contents and the interleaves' alongside
the brief. The syllabus follows the spine's order of chapters unless the
brief or the ordering rules in the authoring spec say otherwise, and each
unit carries `sources:`, a list of (source id, section locator) pairs for
the spine sections it covers and the interleave sections to weave in.
Units the sources do not cover are marked `from_scratch: true` and
authored as today.

### 6.4 Fetch per unit

Before a unit is authored, its sections are fetched into
`units/<id>/sources/<source>-<locator>.md` with a provenance sidecar
(rule 1: source, URL, licence, verdict, fetch date), converted to Markdown
(HTML through a converter; CNXML through its own; LaTeX and Rmd through
pandoc where installed; PDF through text extraction; MediaWiki through the
parse API), boilerplate stripped (rule 7), credited embeds dropped (rule
8). A cache under `state/sources/` keeps every fetched section so a rerun
or a second book costs nothing. The budget is about twelve thousand tokens
of source per unit; over that, the LLM ranks the sections against the
unit's concepts and the rest is left out with a note in the sidecar. Drive
mode serves fixtures instead of fetching, so tests never touch the web.

### 6.5 Author from the material

`unitSystem` gains a source block for units that have one: adapt the
material, keep its voice and its worked examples, keep its order of ideas
where the ordering rules allow, weave the interleave sections in where the
syllabus says, restate rather than quote the Q references, and never
reproduce a figure or a credited quotation. `canon.md`'s front matter
gains `sources:` with the attribution lines and licences, which is the
share-alike bookkeeping and the reader's answer to "whose voice is
this?". Depth variants get the same material; `deeper-math.md` in
particular should prefer the source's own derivations.

### 6.6 Question banks from the sources' exercises

For every A source in a unit, the exercises and their answers (the 2d
paths) go through `roles.ImportItems`: each exercise becomes an item in
`questions.yaml`'s schema with `check` chosen from the answer's shape
(numeric with a tolerance, exact, choice, or rubric when the answer is a
paragraph), a `source:` field on the item, and a misconception id where
the source's distractors or common errors name one. Exercises without an
answer become rubric items only if the author can state the rubric from
the text; instructor-only solutions are not used. The author then fills
the gaps the spec requires (pretest items, callback items, the constructed
fraction) rather than writing the whole bank.

### 6.7 In the reader and the shell

The contents screen and the end of every chapter show "Adapted from
<title> by <authors> (<licence>)" for its sources; the library card's info
line names the spine. `chiron teach` gains `-source` (repeatable) so an
agent can say "starting from X" without writing it into the brief, and
`chiron sources <query>` runs the finder alone and prints the candidates.

### 6.8 Order of work and cost

Status 2026-09-05: steps 1 and 2 are built (commits f750d68, e15ff07,
0835d43): `server-go/sources` with the index at `corpus/sources/index.yaml`
(66 entries, 45 with recipes) and seven fetchers, all checked live;
`generate` resolves named sources, plans from the spine, authors from the
fetched sections and writes attribution into the unit; the chapter shows
its sources; `chiron teach -source` and the `sources` field on
`/teach/create`. The first sourced book, "Bayes for an Engineer" from
Think Bayes, is on the shelf the same evening: seven units, each adapted
from the chapter the planner assigned. It took four runs, each resuming
from the files on disk: units timed out at fifteen minutes on their depth
variants (the author's limit is now thirty, and only the chapter call
carries the material), the author wrote its own `sources:` key beside the
pipeline's (the pipeline's now replaces it), and eight multiple-choice
items lacked `check: choice` (the pipeline now adds it). Read of unit 0:
Chiron's chapter shape with Downey's cookie problem, though the planner
had assigned chapter 1, whose material (Linda, the survey data) went
unused; the planner is now told to title units after the sections it
assigns.

Status 2026-09-06: step 3 is built. `sources.Exercises` reads three
shapes of exercise from fetched Markdown: MyST exercise and solution
directives (QuantEcon), bold "**Exercise:**" markers with the answer in
the cells that follow (Downey), and "Exercise N" or "Problem N" headings
with a Solution or Answer heading under them. A recipe's `solutions:`
names a companion path for answers kept apart from the questions (Think
Bayes keeps them under `soln/`), and the client pairs the two by order or
number. Checked live: Think Bayes chapters 2 and 4 give four exercises
each with solutions; QuantEcon's `prob_dist.md` and `lln_clt.md` give two
each. `roles.ImportItems` turns them into items in the schema, taps
first, each with a `source:` line; the generator writes them to
`imported.yaml` beside the unit before the bank is written and the author
writes the rest of the bank around them. OCW problem sets flow through
the same path once the planner assigns a PDF locator (the sprite now has
`pdftotext`), but no OCW book has been built yet.

Status 2026-09-06, later: step 4 is built. When a brief names nothing the
index has, `Plan` runs the finder: the planner extracts search phrases
and subject tags from the brief (`roles.SearchTerms`, tags held to the
index's vocabulary); candidates are the index by tag and by title, then
the Open Textbook Library, MIT Learn (courses, with their problem-set
features) and the LibreTexts catalogue (`sources.Client.Search`, each hit
a provisional `Source` with a verdict read from the licence field and a
recipe where a fetcher exists for the host); the tables of contents of
the best eight are fetched; and `roles.PickSources` chooses a spine and
up to two interleaves with a sentence of reason each, plus references.
The picks go to `sources.yaml` with their recipes, so a rerun and the
unit authoring can fetch a found source the index never listed. Live on
"a short book on Bayesian statistics for a software engineer who has
never taken a statistics course": spine Think Bayes 2e, interleaves MIT
18.05 and QuantEcon's intro lectures, references Bayes Rules!, MacKay and
Murphy, in seventy seconds. OpenStax's list, DOAB and Gutendex are not
asked yet; the index covers OpenStax, and the other two wait for a
humanities brief. Step 5 is open.


1. The index and the fetchers for the formats that cover most of section
   2: GitHub raw, OpenStax REX, LibreTexts HTML, MediaWiki, OCW, Gutenberg,
   Pressbooks. About a day and a half, fixtures included.
2. The named-sources path end to end on one book ("starting from Think
   Bayes"): plan with the spine, fetch, author from material, attribution
   in the reader. About a day. This is the first book Matt reads in
   someone else's voice.
3. Exercise import. Half a day.
4. The finder and the catalogue queries. About a day.
5. Primers: the capture's plan stage suggests a source chapter when one
   matches, and the primer adapts it. Later.

Guardrails throughout: rules 1 to 10 of section 4, one fetch at a time
per host, and nothing behind a bot wall.

Status 2026-09-08: the `file` source kind reads a PDF or an EPUB the
reader owns from the sprite's disk (a PDF through pdftotext, chapters
split at the pages that open with "CHAPTER", a number and a title; an
EPUB through its own package, one chapter per spine document, titled
from its navigation document or its own first heading), and
`chiron teach -plan-only` writes
the syllabus and stops so the plan can be cut before a unit is authored.
The first such book, Private Debt for Norm Capital, was planned at 21
units, merged by hand to 17 plus placement, and authored from there.

Next, agreed 2026-09-07 but not started: **a book appears after its
first unit or two, and the rest is authored ahead of the reader.** Today
`generate` registers the subject only once every unit is on disk, so the
whole book's budget is spent before a page is read. The change: register
after u0 and u1, keep authoring in the background in syllabus order, and
have the reader's next-unit request wait on the unit if it is not there
yet (the chapter wait screen already shows a stage). Worth doing when
more books are started than finished.

