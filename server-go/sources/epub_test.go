package sources

import (
	"archive/zip"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeEPUB makes the smallest thing that is still an EPUB: the mimetype,
// the container pointing at the package, a package with a spine, and a
// navigation document naming the chapters.
func writeEPUB(t *testing.T, entries map[string]string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "book.epub")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	z := zip.NewWriter(f)
	for name, body := range entries {
		w, err := z.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

const epubContainer = `<?xml version="1.0"?>
<container xmlns="urn:oasis:names:tc:opendocument:xmlns:container" version="1.0">
  <rootfiles><rootfile full-path="OEBPS/package.opf" media-type="application/oebps-package+xml"/></rootfiles>
</container>`

const epubPackage = `<?xml version="1.0"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0" unique-identifier="id">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/">
    <dc:title>A Borrowed Book</dc:title>
    <dc:creator>A Writer</dc:creator>
  </metadata>
  <manifest>
    <item id="nav" href="nav.xhtml" media-type="application/xhtml+xml" properties="nav"/>
    <item id="cover" href="cover.xhtml" media-type="application/xhtml+xml"/>
    <item id="c1" href="text/ch01.xhtml" media-type="application/xhtml+xml"/>
    <item id="c2" href="text/ch02.xhtml" media-type="application/xhtml+xml"/>
    <item id="css" href="style.css" media-type="text/css"/>
  </manifest>
  <spine>
    <itemref idref="cover"/>
    <itemref idref="c1"/>
    <itemref idref="c2"/>
  </spine>
</package>`

const epubNav = `<?xml version="1.0" encoding="utf-8"?>
<html xmlns="http://www.w3.org/1999/xhtml" xmlns:epub="http://www.idpf.org/2007/ops">
<body><nav epub:type="toc"><ol>
  <li><a href="text/ch01.xhtml">What Lending Is</a></li>
  <li><a href="text/ch02.xhtml">Who Lends, and Why</a></li>
</ol></nav></body></html>`

func borrowedBook(t *testing.T) string {
	return writeEPUB(t, map[string]string{
		"mimetype":                 "application/epub+zip",
		"META-INF/container.xml":   epubContainer,
		"OEBPS/package.opf":        epubPackage,
		"OEBPS/nav.xhtml":          epubNav,
		"OEBPS/cover.xhtml":        `<html><body><p>A Borrowed Book</p></body></html>`,
		"OEBPS/text/ch01.xhtml": `<html><body><h1>What Lending Is</h1>` +
			`<p>A loan is money now against money later, and the <em>later</em> is the whole risk.</p>` +
			`<p>Two questions follow: who pays, and what happens when they do not.</p></body></html>`,
		"OEBPS/text/ch02.xhtml": `<html><body><h1>Who Lends, and Why</h1>` +
			`<p>Banks lend because deposits cost less than loans earn.</p></body></html>`,
		"OEBPS/style.css": `p { margin: 0 }`,
	})
}

// An EPUB the reader keeps on disk lists its chapters in reading order,
// titled from the navigation document, and a chapter fetched reads as the
// prose it is.
func TestFileSourceSplitsAnEPUBIntoChapters(t *testing.T) {
	src := &Source{ID: "borrowed", Title: "A Borrowed Book", Verdict: Adaptable,
		Fetch: Recipe{Kind: "file", Path: borrowedBook(t)}}
	c := &Client{}
	secs, err := c.Contents(context.Background(), src)
	if err != nil {
		t.Fatal(err)
	}
	want := []Section{
		{Locator: "OEBPS/cover.xhtml", Title: "Front matter"},
		{Locator: "OEBPS/text/ch01.xhtml", Title: "What Lending Is"},
		{Locator: "OEBPS/text/ch02.xhtml", Title: "Who Lends, and Why"},
	}
	if len(secs) != len(want) {
		t.Fatalf("sections: %+v", secs)
	}
	for i, w := range want {
		if secs[i] != w {
			t.Errorf("section %d = %+v, want %+v", i, secs[i], w)
		}
	}

	got, err := c.Fetch(context.Background(), src, "OEBPS/text/ch01.xhtml")
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "What Lending Is" {
		t.Errorf("title = %q", got.Title)
	}
	if !strings.Contains(got.Markdown, "money now against money later") {
		t.Errorf("prose missing:\n%s", got.Markdown)
	}
	if !strings.Contains(got.Markdown, "*later*") {
		t.Errorf("emphasis should survive as markdown:\n%s", got.Markdown)
	}
	if strings.Contains(got.Markdown, "<p>") {
		t.Errorf("markup left in:\n%s", got.Markdown)
	}
	if !strings.Contains(got.Provenance.URL, "ch01.xhtml") {
		t.Errorf("url = %q", got.Provenance.URL)
	}
}

// Without a navigation document the chapter's own heading names it, and a
// document with neither falls back to its file name.
func TestEPUBTitlesFallBackToTheHeading(t *testing.T) {
	path := writeEPUB(t, map[string]string{
		"mimetype":               "application/epub+zip",
		"META-INF/container.xml": epubContainer,
		"OEBPS/package.opf":      epubPackage,
		"OEBPS/cover.xhtml":      `<html><body><p>no heading here</p></body></html>`,
		"OEBPS/text/ch01.xhtml":  `<html><body><h2>A Heading Of Its Own</h2><p>Prose.</p></body></html>`,
		"OEBPS/text/ch02.xhtml":  `<html><body><p>Nothing but prose.</p></body></html>`,
	})
	src := &Source{ID: "plain", Title: "Plain", Verdict: Adaptable,
		Fetch: Recipe{Kind: "file", Path: path}}
	secs, err := (&Client{}).Contents(context.Background(), src)
	if err != nil {
		t.Fatal(err)
	}
	if len(secs) != 3 {
		t.Fatalf("sections: %+v", secs)
	}
	if secs[1].Title != "A Heading Of Its Own" {
		t.Errorf("heading title = %q", secs[1].Title)
	}
	if secs[2].Title != "ch02" {
		t.Errorf("file-name title = %q", secs[2].Title)
	}
}
