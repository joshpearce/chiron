package sources

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// A PDF the reader keeps on disk lists its chapters from the pages that
// open with "CHAPTER", a number and a title, with what comes before the
// first as front matter; a chapter fetched reads as prose, with the drop
// capital rejoined and the page number gone.
func TestFileSourceSplitsAPDFIntoChapters(t *testing.T) {
	if _, err := exec.LookPath("pdftotext"); err != nil {
		t.Skip("pdftotext not installed")
	}
	src := &Source{ID: "little", Title: "A Little Book", Verdict: Adaptable,
		Fetch: Recipe{Kind: "file", Path: filepath.Join("testdata", "chapters.pdf")}}
	c := &Client{}
	secs, err := c.Contents(context.Background(), src)
	if err != nil {
		t.Fatal(err)
	}
	want := []Section{
		{Locator: "p1-1", Title: "Front matter"},
		{Locator: "p2-3", Title: "First Things"},
		{Locator: "p4-4", Title: "Second Things and Their Names"},
	}
	if len(secs) != len(want) {
		t.Fatalf("contents = %+v, want %+v", secs, want)
	}
	for i := range want {
		if secs[i] != want[i] {
			t.Errorf("contents[%d] = %+v, want %+v", i, secs[i], want[i])
		}
	}
	ch, err := c.Fetch(context.Background(), src, "p2-3")
	if err != nil {
		t.Fatal(err)
	}
	if ch.Title != "First Things" || !strings.HasPrefix(ch.Markdown, "# First Things\n\nThe first chapter begins here") {
		t.Errorf("chapter = %q: %q", ch.Title, ch.Markdown)
	}
	if strings.Contains(ch.Markdown, "\n7\n") || !strings.Contains(ch.Markdown, "runs for a while.\nMore of the first chapter") {
		t.Errorf("page number kept or pages not joined: %q", ch.Markdown)
	}
	if ch.Provenance.URL != "file://testdata/chapters.pdf#p2-3" {
		t.Errorf("url = %q", ch.Provenance.URL)
	}
	if _, err := c.Fetch(context.Background(), src, "chapter-1"); err == nil {
		t.Error("a locator that is not a page range fetched")
	}
}

// A Markdown file is split at its headings; a file with none is one
// section named for the file.
func TestFileSourceSplitsMarkdownAtHeadings(t *testing.T) {
	dir := t.TempDir()
	md := filepath.Join(dir, "notes.md")
	os.WriteFile(md, []byte("# Notes\n\nintro\n\n## Yield and IRR\n\nyield text\n\n## Fees\n\nfee text\n"), 0o644)
	src := &Source{ID: "notes", Verdict: Adaptable, Fetch: Recipe{Kind: "file", Path: md}}
	c := &Client{}
	secs, err := c.Contents(context.Background(), src)
	if err != nil {
		t.Fatal(err)
	}
	if len(secs) != 3 || secs[1].Locator != "yield-and-irr" || secs[1].Title != "Yield and IRR" {
		t.Fatalf("contents = %+v", secs)
	}
	ch, err := c.Fetch(context.Background(), src, "fees")
	if err != nil {
		t.Fatal(err)
	}
	if ch.Markdown != "## Fees\n\nfee text\n" || ch.Title != "Fees" {
		t.Errorf("section = %q %q", ch.Title, ch.Markdown)
	}
	plain := filepath.Join(dir, "memo.txt")
	os.WriteFile(plain, []byte("just a memo\n"), 0o644)
	src.Fetch.Path = plain
	secs, _ = c.Contents(context.Background(), src)
	if len(secs) != 1 || secs[0].Locator != "all" || secs[0].Title != "memo" {
		t.Errorf("plain file contents = %+v", secs)
	}
}

func TestIndexValidatesFileSources(t *testing.T) {
	idx := &Index{Sources: []Source{{ID: "own", Title: "Own", Authors: []string{"A"}, Subjects: []string{"x"}, Verdict: Adaptable,
		Licence: Licence{Name: "n", URL: "u"}, Fetch: Recipe{Kind: "file"}}}}
	problems := idx.Lint()
	if len(problems) != 1 || !strings.Contains(problems[0], "file needs path") {
		t.Errorf("problems = %v", problems)
	}
}

// A title runs on to a second line that may start in lowercase; prose
// starts at the drop capital, a blank line, or a sentence's end.
func TestChapterOpeningReadsMultiLineTitles(t *testing.T) {
	cases := map[string]string{
		"CHAPTER\n\n21\n\nExpected Returns and Risks\nfrom Direct Lending\n\nD\nirect lending is.": "Expected Returns and Risks from Direct Lending",
		"CHAPTER 3\nPerformance Comparisons\nThe chapter compares things.":                         "Performance Comparisons",
		"Chapter IV\n\nOn Yield\n":               "On Yield",
		"CHAPTER\nnot a number\nTitle":           "",
		"Some running header\nCHAPTER\n1\nTitle": "",
	}
	for page, want := range cases {
		if got, _ := chapterOpening([]string{page}); got != want {
			t.Errorf("%q: title = %q, want %q", page, got, want)
		}
	}
}
