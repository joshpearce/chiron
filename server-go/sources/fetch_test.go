package sources

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Each fetcher against a fake of its host: the table of contents comes
// back as sections, one section comes back as Markdown with provenance,
// and what the page says about its licence is honoured.

func fakeHost(t *testing.T, routes map[string]string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Path
		if r.URL.RawQuery != "" {
			key += "?" + r.URL.RawQuery
		}
		if body, ok := routes[key]; ok {
			w.Write([]byte(body))
			return
		}
		if body, ok := routes[r.URL.Path]; ok {
			w.Write([]byte(body))
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func client(t *testing.T) *Client {
	t.Helper()
	c := NewClient(filepath.Join(t.TempDir(), "cache"))
	c.Pause = 0
	c.Now = func() time.Time { return time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC) }
	return c
}

func TestGitHubReadsMarkdownAndNotebooks(t *testing.T) {
	nb, _ := json.Marshal(map[string]any{
		"metadata": map[string]any{"language_info": map[string]any{"name": "python"}},
		"cells": []map[string]any{
			{"cell_type": "markdown", "source": []string{"# Chapter 2\n", "\n", "Bayes' theorem, once more.\n"}},
			{"cell_type": "code", "source": "p = 0.5\np * 2", "outputs": []map[string]any{{"output_type": "execute_result", "data": map[string]any{"text/plain": "1.0"}}}},
			{"cell_type": "markdown", "source": "**Exercise:** what is the prior?\n"},
		},
	})
	srv := fakeHost(t, map[string]string{
		"/d2l-ai/d2l-en/master/chapter_linear-regression/linear-regression.md": "# Linear Regression\n\nWe begin with the simplest model.\n\n## Exercises\n\n1. Solve it.\n",
		"/AllenDowney/ThinkBayes2/master/notebooks/chap02.ipynb":               string(nb),
		"/api/repos/AllenDowney/ThinkBayes2/contents/notebooks?ref=master":     `[{"name":"chap01.ipynb","type":"file"},{"name":"chap02.ipynb","type":"file"},{"name":"figs","type":"dir"},{"name":"data.csv","type":"file"}]`,
	})
	c := client(t)
	d2l := &Source{ID: "d2l", Title: "Dive into Deep Learning", Authors: []string{"Zhang et al."}, Verdict: Adaptable,
		Licence:  Licence{Name: "CC BY-SA 4.0", URL: "https://github.com/d2l-ai/d2l-en/blob/master/LICENSE"},
		Fetch:    Recipe{Kind: "github", Base: srv.URL, Repo: "d2l-ai/d2l-en", Ref: "master", Path: "chapter_linear-regression/{locator}"},
		Contents: []Section{{Locator: "linear-regression.md", Title: "Linear Regression"}}}
	ch, err := c.Fetch(context.Background(), d2l, "linear-regression.md")
	if err != nil {
		t.Fatal(err)
	}
	if ch.Title != "Linear Regression" || !strings.Contains(ch.Markdown, "simplest model") {
		t.Fatalf("chunk = %+v", ch)
	}
	p := ch.Provenance
	if p.Verdict != Adaptable || p.Licence != "CC BY-SA 4.0" || p.Fetched != "2026-09-05" || !strings.HasSuffix(p.URL, "/linear-regression.md") {
		t.Fatalf("provenance = %+v", p)
	}
	if p.Attribution() != "Adapted from Dive into Deep Learning by Zhang et al. (CC BY-SA 4.0)" {
		t.Fatalf("attribution = %q", p.Attribution())
	}

	think := &Source{ID: "think-bayes", Title: "Think Bayes", Verdict: Adaptable, Licence: Licence{Name: "CC BY-NC-SA 4.0", URL: "x"},
		Fetch: Recipe{Kind: "github", Base: srv.URL, Repo: "AllenDowney/ThinkBayes2", Ref: "master", Path: "notebooks/{locator}", Dir: "notebooks"}}
	secs, err := c.Contents(context.Background(), think)
	if err != nil {
		t.Fatal(err)
	}
	if len(secs) != 2 || secs[0].Locator != "chap01.ipynb" || secs[1].Locator != "chap02.ipynb" {
		t.Fatalf("contents = %+v", secs)
	}
	ch, err = c.Fetch(context.Background(), think, "chap02.ipynb")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"# Chapter 2", "Bayes' theorem, once more.", "```python\np = 0.5\np * 2\n```", "```\n1.0\n```", "**Exercise:**"} {
		if !strings.Contains(ch.Markdown, want) {
			t.Errorf("notebook missing %q:\n%s", want, ch.Markdown)
		}
	}
	if ch.Title != "Chapter 2" {
		t.Errorf("title = %q", ch.Title)
	}
}

func TestFrontMatterIsStripped(t *testing.T) {
	in := "---\njupytext:\n  format_name: myst\n---\n\n# Title\n\nBody.\n"
	if got := stripFrontMatter(in); got != "# Title\n\nBody.\n" {
		t.Fatalf("got %q", got)
	}
	if got := stripFrontMatter("# No front matter\n"); got != "# No front matter\n" {
		t.Fatalf("got %q", got)
	}
}

func TestTheCacheAnswersTheSecondTime(t *testing.T) {
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.Write([]byte("# Once\n"))
	}))
	defer srv.Close()
	c := client(t)
	s := &Source{ID: "s", Title: "S", Verdict: Adaptable, Licence: Licence{Name: "CC BY", URL: "x"},
		Fetch: Recipe{Kind: "github", Base: srv.URL, Repo: "a/b", Path: "{locator}"}}
	for i := 0; i < 2; i++ {
		if _, err := c.Fetch(context.Background(), s, "x.md"); err != nil {
			t.Fatal(err)
		}
	}
	if hits != 1 {
		t.Fatalf("host was asked %d times", hits)
	}
	if entries, _ := os.ReadDir(c.Cache); len(entries) == 0 {
		t.Fatal("nothing cached")
	}
}

func TestARestrictedSourceIsNeverFetched(t *testing.T) {
	c := client(t)
	s := &Source{ID: "feynman", Title: "The Feynman Lectures", Verdict: Restricted, Fetch: Recipe{Kind: "github", Repo: "x/y", Path: "{locator}"}}
	if _, err := c.Fetch(context.Background(), s, "1.md"); err != ErrRestricted {
		t.Fatalf("err = %v", err)
	}
	if _, err := c.Contents(context.Background(), s); err != ErrRestricted {
		t.Fatalf("err = %v", err)
	}
}

func TestOpenStaxWalksTheTreeAndReadsAPage(t *testing.T) {
	book := "8b89d172-2927-466f-8661-01abc7ccdba4"
	srv := fakeHost(t, map[string]string{
		"/rex/release.json": `{"archiveUrl":"/apps/archive/20260101.1","books":{"` + book + `":{"defaultVersion":"8dbc2ce"}}}`,
		"/apps/archive/20260101.1/contents/" + book + "@8dbc2ce.json": `{"title":"Calculus Volume 1","tree":{"id":"` + book + `@8dbc2ce","contents":[
			{"id":"p1@1","title":"<span class=\"os-text\">Preface</span>"},
			{"id":"c1@1","title":"<span class=\"os-number\">1</span> <span class=\"os-text\">Functions</span>","contents":[
				{"id":"p2@1","title":"<span class=\"os-number\">1.1</span> <span class=\"os-text\">Review of Functions</span>"},
				{"id":"p3@1","title":"<span class=\"os-number\">1.2</span> <span class=\"os-text\">Basic Classes</span>"}]}]}}`,
		"/apps/archive/20260101.1/contents/" + book + "@8dbc2ce:p2.json": `{"title":"<span class=\"os-number\">1.1</span> <span class=\"os-text\">Review of Functions</span>","content":"<div data-type=\"page\"><h2>Learning Objectives</h2><p>Use functional notation.</p><div data-type=\"exercise\"><div data-type=\"problem\"><p>1. Find f(2).</p></div><div data-type=\"solution\"><p>4</p></div></div></div>"}`,
	})
	c := client(t)
	s := &Source{ID: "openstax-calc1", Title: "Calculus Volume 1", Verdict: Adaptable, Licence: Licence{Name: "CC BY-NC-SA 4.0", URL: "x"},
		Fetch: Recipe{Kind: "openstax", Base: srv.URL, Book: book}}
	secs, err := c.Contents(context.Background(), s)
	if err != nil {
		t.Fatal(err)
	}
	if len(secs) != 3 || secs[0].Title != "Preface" || secs[1].Locator != "p2" || secs[1].Title != "1.1 Review of Functions" || secs[1].Level != 2 {
		t.Fatalf("contents = %+v", secs)
	}
	ch, err := c.Fetch(context.Background(), s, "p2")
	if err != nil {
		t.Fatal(err)
	}
	if ch.Title != "1.1 Review of Functions" || !strings.Contains(ch.Markdown, "## Learning Objectives") || !strings.Contains(ch.Markdown, "1. Find f(2).") {
		t.Fatalf("chunk = %q", ch.Markdown)
	}
}

func TestLibreTextsHonoursThePagesOwnLicence(t *testing.T) {
	root := "/Bookshelves/Introductory_Statistics/Book"
	srv := fakeHost(t, map[string]string{
		root:                                     `<html><body><a href="HOST` + root + `/01:_Sampling">1: Sampling</a><a href="HOST` + root + `/02:_Descriptive">2: Descriptive</a><a href="HOST/Bookshelves/Other">Other</a></body></html>`,
		root + "/01:_Sampling":                   `<html><body><a href="HOST` + root + `/01:_Sampling/1.01:_Definitions">1.1: Definitions</a><a href="HOST` + root + `/01:_Sampling/1.E:_Exercises">1.E: Exercises</a></body></html>`,
		root + "/02:_Descriptive":                `<html><body></body></html>`,
		root + "/01:_Sampling/1.01:_Definitions": `<html><body><h1>1.1: Definitions of Statistics</h1><a class="mt-tag" href="/Special:Tags?tag=license:ccby">license:ccby</a><section class="mt-content-container"><p>The science of statistics deals with data.</p><p class="box-exercise">Exercise 1. Answer: yes.</p></section><section class="mt-content-container">other</section><footer>This page titled 1.1 is shared under a CC BY license</footer></body></html>`,
		root + "/01:_Sampling/1.E:_Exercises":    `<html><body><h1>1.E: Exercises</h1><span>license:ccbyncnd</span><section class="mt-content-container"><p>1. What is a sample?</p></section></body></html>`,
	})
	c := client(t)
	c.HTTP = srv.Client()
	s := &Source{ID: "lt-stats", Title: "Introductory Statistics", Verdict: Adaptable, Licence: Licence{Name: "CC BY 4.0", URL: "x"},
		Fetch: Recipe{Kind: "libretexts", URL: srv.URL + root}}
	// Rewrite HOST in fixtures now that the server URL is known.
	srv.Config.Handler = rewriting(srv.Config.Handler, "HOST", srv.URL)
	secs, err := c.Contents(context.Background(), s)
	if err != nil {
		t.Fatal(err)
	}
	locs := []string{}
	for _, sec := range secs {
		locs = append(locs, sec.Locator)
	}
	if strings.Join(locs, ",") != "01:_Sampling,01:_Sampling/1.01:_Definitions,01:_Sampling/1.E:_Exercises,02:_Descriptive" {
		t.Fatalf("contents = %v", locs)
	}
	ch, err := c.Fetch(context.Background(), s, "01:_Sampling/1.01:_Definitions")
	if err != nil {
		t.Fatal(err)
	}
	if ch.Title != "1.1: Definitions of Statistics" || !strings.Contains(ch.Markdown, "deals with data") || strings.Contains(ch.Markdown, "shared under") || strings.Contains(ch.Markdown, "other") {
		t.Fatalf("chunk = %+v", ch)
	}
	if ch.Provenance.Verdict != Adaptable || !strings.Contains(ch.Provenance.Note, "CC BY") {
		t.Fatalf("provenance = %+v", ch.Provenance)
	}
	ex, err := c.Fetch(context.Background(), s, "01:_Sampling/1.E:_Exercises")
	if err != nil {
		t.Fatal(err)
	}
	if ex.Provenance.Verdict != QuoteOnly {
		t.Fatalf("an ND page must come back quote-only: %+v", ex.Provenance)
	}
}

// rewriting swaps a placeholder for the server's URL in every response.
func rewriting(h http.Handler, from, to string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, r)
		w.WriteHeader(rec.Code)
		w.Write([]byte(strings.ReplaceAll(rec.Body.String(), from, to)))
	})
}

func TestMediaWikiListsABookAndParsesAPage(t *testing.T) {
	srv := fakeHost(t, map[string]string{
		"/w/api.php?action=parse&disableeditsection=1&format=json&formatversion=2&page=Haskell%2FRecursion&prop=text%7Cdisplaytitle": `{"parse":{"title":"Haskell/Recursion","displaytitle":"Haskell/Recursion","text":"<div class=\"mw-parser-output\"><h2>Numeric recursion</h2><p>The factorial function is <code>fac n = n * fac (n-1)</code>.</p></div>"}}`,
		"/w/api.php?action=parse&format=json&formatversion=2&page=Haskell&prop=links":                                                `{"parse":{"links":[{"ns":0,"exists":true,"title":"Haskell/Getting_set_up"},{"ns":0,"exists":true,"title":"Haskell/Recursion"},{"ns":0,"exists":false,"title":"Haskell/Missing"},{"ns":0,"exists":true,"title":"Python/Intro"},{"ns":4,"exists":true,"title":"Wikibooks:Help"}]}}`,
	})
	c := client(t)
	s := &Source{ID: "wb-haskell", Title: "Haskell", Verdict: Adaptable, Licence: Licence{Name: "CC BY-SA 4.0", URL: "x"},
		Fetch: Recipe{Kind: "mediawiki", Base: srv.URL, Site: "https://en.wikibooks.org", Prefix: "Haskell"}}
	secs, err := c.Contents(context.Background(), s)
	if err != nil {
		t.Fatal(err)
	}
	if len(secs) != 2 || secs[1].Locator != "Haskell/Recursion" || secs[0].Title != "Getting set up" {
		t.Fatalf("contents = %+v", secs)
	}
	ch, err := c.Fetch(context.Background(), s, "Haskell/Recursion")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(ch.Markdown, "## Numeric recursion") || !strings.Contains(ch.Markdown, "`fac n = n * fac (n-1)`") {
		t.Fatalf("chunk = %q", ch.Markdown)
	}
}

func TestOCWListsPagesAndReadsOne(t *testing.T) {
	srv := fakeHost(t, map[string]string{
		"/courses/18-05-introduction-to-probability-and-statistics-spring-2022/":                         `<html><body><nav><a href="/courses/18-05-introduction-to-probability-and-statistics-spring-2022/pages/syllabus/">Syllabus</a><a href="/courses/18-05-introduction-to-probability-and-statistics-spring-2022/pages/readings/">Readings</a><a href="/courses/18-05-introduction-to-probability-and-statistics-spring-2022/pages/readings/">Readings again</a></nav></body></html>`,
		"/courses/18-05-introduction-to-probability-and-statistics-spring-2022/pages/readings/data.json": `{"title":"Readings","content":"<h2>Readings</h2><p>Reading 20: Comparison of frequentist and Bayesian inference.</p>"}`,
	})
	c := client(t)
	s := &Source{ID: "ocw-1805", Title: "18.05", Verdict: Adaptable, Licence: Licence{Name: "CC BY-NC-SA 4.0", URL: "x"},
		Fetch: Recipe{Kind: "ocw", Base: srv.URL, Course: "18-05-introduction-to-probability-and-statistics-spring-2022"}}
	secs, err := c.Contents(context.Background(), s)
	if err != nil {
		t.Fatal(err)
	}
	if len(secs) != 2 || secs[0].Locator != "syllabus" || secs[1].Locator != "readings" {
		t.Fatalf("contents = %+v", secs)
	}
	ch, err := c.Fetch(context.Background(), s, "readings")
	if err != nil {
		t.Fatal(err)
	}
	if ch.Title != "Readings" || !strings.Contains(ch.Markdown, "Reading 20") {
		t.Fatalf("chunk = %+v", ch)
	}
}

func TestGutenbergCutsTheBoilerplateAndSplitsChapters(t *testing.T) {
	// The shape of the real text: a title page, a contents list of
	// consecutive chapter lines, then chapters whose heading and title sit
	// on two lines.
	text := "The Project Gutenberg eBook of On the Origin of Species\r\n\r\n*** START OF THE PROJECT GUTENBERG EBOOK 2009 ***\r\n\r\nON THE ORIGIN OF SPECIES\r\n\r\nBY MEANS OF NATURAL SELECTION,\r\n\r\nBY CHARLES DARWIN\r\n\r\nCONTENTS\r\n\r\nINTRODUCTION.\r\n\r\nCHAPTER I. VARIATION UNDER DOMESTICATION\r\nCHAPTER II. VARIATION UNDER NATURE\r\n\r\nCHAPTER I.\r\nVARIATION UNDER DOMESTICATION.\r\nCauses of Variability (a synopsis).\r\n\r\nCHAPTER II.\r\nVARIATION UNDER NATURE.\r\nVariability (a synopsis).\r\n\r\nINTRODUCTION\r\n\r\nWhen on board H.M.S. Beagle, as naturalist, I was much struck.\r\n\r\nCHAPTER I.\r\nVARIATION UNDER DOMESTICATION.\r\nCauses of Variability.\r\n\r\nWhen we look to the individuals of the same variety.\r\nA second line.\r\n\r\nCHAPTER II.\r\nVARIATION UNDER NATURE.\r\n\r\nBefore applying the principles.\r\n\r\n*** END OF THE PROJECT GUTENBERG EBOOK 2009 ***\r\n\r\nlicence boilerplate\r\n"
	srv := fakeHost(t, map[string]string{"/ebooks/2009.txt.utf-8": text})
	c := client(t)
	s := &Source{ID: "pg-origin", Title: "On the Origin of Species", Verdict: Adaptable, Licence: Licence{Name: "Public domain", URL: "x"},
		Fetch: Recipe{Kind: "gutenberg", Base: srv.URL, ID: 2009}}
	secs, err := c.Contents(context.Background(), s)
	if err != nil {
		t.Fatal(err)
	}
	titles := []string{}
	for _, sec := range secs {
		titles = append(titles, sec.Title)
	}
	want := "CHAPTER I. VARIATION UNDER DOMESTICATION.|CHAPTER II. VARIATION UNDER NATURE."
	if strings.Join(titles, "|") != want {
		t.Fatalf("chapters = %v", titles)
	}
	ch, err := c.Fetch(context.Background(), s, "1")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(ch.Markdown, "# CHAPTER I. VARIATION UNDER DOMESTICATION.\n\nCauses of Variability.\n") || !strings.Contains(ch.Markdown, "A second line.") || strings.Contains(ch.Markdown, "synopsis") || strings.Contains(ch.Markdown, "Before applying") || strings.Contains(ch.Markdown, "boilerplate") {
		t.Fatalf("chunk = %q", ch.Markdown)
	}

	// A text with no chapter headings splits on its capitalised lines,
	// and the title page folds into the first real one.
	plain := "*** START OF THE PROJECT GUTENBERG EBOOK 1 ***\n\nA TITLE\n\nBY SOMEONE\n\nFIRST PART\n\n" + strings.Repeat("Words of the first part. ", 40) + "\n\nSECOND PART\n\n" + strings.Repeat("Words of the second part. ", 40) + "\n\n*** END OF THE PROJECT GUTENBERG EBOOK 1 ***\n"
	srv2 := fakeHost(t, map[string]string{"/ebooks/1.txt.utf-8": plain})
	s2 := &Source{ID: "pg-plain", Title: "Plain", Verdict: Adaptable, Licence: Licence{Name: "Public domain", URL: "x"}, Fetch: Recipe{Kind: "gutenberg", Base: srv2.URL, ID: 1}}
	secs2, err := c.Contents(context.Background(), s2)
	if err != nil {
		t.Fatal(err)
	}
	if len(secs2) != 2 || secs2[0].Title != "FIRST PART" || secs2[1].Title != "SECOND PART" {
		t.Fatalf("plain chapters = %+v", secs2)
	}
}

func TestPressbooksReadsTheTOCAndAChapterWithItsLicence(t *testing.T) {
	srv := fakeHost(t, map[string]string{
		"/intro-philosophy/wp-json/pressbooks/v2/toc":         `{"front-matter":[{"id":1,"title":"Preface","slug":"preface"}],"parts":[{"title":"Part I","chapters":[{"id":12,"title":"What is Philosophy?","slug":"what-is-philosophy"},{"id":13,"title":"Logic","slug":"logic"}]}]}`,
		"/intro-philosophy/wp-json/pressbooks/v2/chapters/12": `{"title":{"rendered":"What is Philosophy?"},"content":{"rendered":"<p>Philosophy begins in wonder.</p>"},"link":"https://press.rebus.community/intro-philosophy/chapter/what-is-philosophy/","metadata":{"license":{"url":"https://creativecommons.org/licenses/by/4.0/","name":"CC BY (Attribution)"}}}`,
		"/intro-philosophy/wp-json/pressbooks/v2/chapters/13": `{"title":{"rendered":"Logic"},"content":{"rendered":"<p>Valid arguments.</p>"},"metadata":{"license":{"url":"https://creativecommons.org/licenses/by-nd/4.0/","name":"CC BY-ND"}}}`,
	})
	c := client(t)
	s := &Source{ID: "rebus-phil", Title: "Introduction to Philosophy", Verdict: Adaptable, Licence: Licence{Name: "CC BY 4.0", URL: "x"},
		Fetch: Recipe{Kind: "pressbooks", URL: srv.URL + "/intro-philosophy"}}
	secs, err := c.Contents(context.Background(), s)
	if err != nil {
		t.Fatal(err)
	}
	if len(secs) != 2 || secs[0].Locator != "12" || secs[0].Title != "Part I: What is Philosophy?" {
		t.Fatalf("contents = %+v", secs)
	}
	ch, err := c.Fetch(context.Background(), s, "12")
	if err != nil {
		t.Fatal(err)
	}
	if ch.Title != "What is Philosophy?" || !strings.Contains(ch.Markdown, "begins in wonder") || ch.Provenance.Verdict != Adaptable || !strings.HasPrefix(ch.Provenance.URL, "https://press.rebus") {
		t.Fatalf("chunk = %+v", ch)
	}
	nd, err := c.Fetch(context.Background(), s, "13")
	if err != nil {
		t.Fatal(err)
	}
	if nd.Provenance.Verdict != QuoteOnly {
		t.Fatalf("an ND chapter must come back quote-only: %+v", nd.Provenance)
	}
}

func TestTheIndexIsLinted(t *testing.T) {
	idx := &Index{Sources: []Source{
		{ID: "ok", Title: "Fine", Subjects: []string{"x"}, Verdict: Adaptable, Licence: Licence{Name: "CC BY", URL: "u"}, Fetch: Recipe{Kind: "github", Repo: "a/b", Path: "{locator}", Dir: "d"}},
		{ID: "ok", Title: "Dup", Subjects: []string{"x"}, Verdict: QuoteOnly, Licence: Licence{Name: "n", URL: "u"}},
		{ID: "nolic", Title: "No licence", Subjects: []string{"x"}, Verdict: Adaptable, Fetch: Recipe{Kind: "openstax", Book: "b"}},
		{ID: "norecipe", Title: "No recipe", Subjects: []string{"x"}, Verdict: Adaptable, Licence: Licence{Name: "n", URL: "u"}},
		{ID: "badgh", Title: "Bad github", Subjects: []string{"x"}, Verdict: Adaptable, Licence: Licence{Name: "n", URL: "u"}, Fetch: Recipe{Kind: "github", Repo: "a/b", Path: "no-placeholder"}},
	}}
	problems := idx.Lint()
	joined := strings.Join(problems, "\n")
	for _, want := range []string{"duplicate id", "licence needs a name", "path needs {locator}", "needs contents or dir"} {
		if !strings.Contains(joined, want) {
			t.Errorf("lint missing %q in:\n%s", want, joined)
		}
	}
	if strings.Contains(joined, "norecipe") {
		t.Errorf("an adaptable source without a recipe is allowed:\n%s", joined)
	}
	if got := idx.Matching([]string{"x"}); len(got) != 5 {
		t.Errorf("matching = %d", len(got))
	}
	if s := idx.ByTitle("bad GitHub"); s == nil || s.ID != "badgh" {
		t.Errorf("by title = %+v", s)
	}
}

// The shipped index loads and lints, and the fetchers can list it.
func TestTheShippedIndexIsClean(t *testing.T) {
	idx, err := LoadIndex(filepath.Join("..", "..", "corpus", "sources", "index.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if len(idx.Sources) < 40 {
		t.Fatalf("only %d sources", len(idx.Sources))
	}
	if s := idx.ByTitle("Think Bayes"); s == nil || s.Fetch.Kind != "github" {
		t.Fatalf("Think Bayes = %+v", s)
	}
	if got := idx.Matching([]string{"bayesian", "statistics"}); len(got) == 0 || got[0].ID != "think-bayes" && got[0].ID != "ocw-18-05" {
		t.Fatalf("bayesian statistics matches = %v", got[0].ID)
	}
}
