package httpapi

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// epubBytes is the smallest thing that is still an EPUB: a container
// naming the package, a spine, and a navigation document titling it.
func epubBytes(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	z := zip.NewWriter(&buf)
	entries := map[string]string{
		"mimetype": "application/epub+zip",
		"META-INF/container.xml": `<?xml version="1.0"?>
<container xmlns="urn:oasis:names:tc:opendocument:xmlns:container" version="1.0">
  <rootfiles><rootfile full-path="OEBPS/package.opf" media-type="application/oebps-package+xml"/></rootfiles>
</container>`,
		"OEBPS/package.opf": `<?xml version="1.0"?>
<package xmlns="http://www.idpf.org/2007/opf" version="3.0">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/"><dc:title>A Borrowed Book</dc:title></metadata>
  <manifest>
    <item id="nav" href="nav.xhtml" media-type="application/xhtml+xml" properties="nav"/>
    <item id="c1" href="ch01.xhtml" media-type="application/xhtml+xml"/>
    <item id="c2" href="ch02.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine><itemref idref="c1"/><itemref idref="c2"/></spine>
</package>`,
		"OEBPS/nav.xhtml": `<html xmlns:epub="http://www.idpf.org/2007/ops"><body><nav epub:type="toc"><ol>
  <li><a href="ch01.xhtml">What Lending Is</a></li>
  <li><a href="ch02.xhtml">Who Lends, and Why</a></li>
</ol></nav></body></html>`,
		"OEBPS/ch01.xhtml": `<html><body><h1>What Lending Is</h1>` +
			`<p>A loan is money now against money later, and the later is the whole risk.</p></body></html>`,
		"OEBPS/ch02.xhtml": `<html><body><h1>Who Lends, and Why</h1>` +
			`<p>Banks lend because deposits cost less than loans earn.</p></body></html>`,
	}
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
	return buf.Bytes()
}

// An EPUB the reader imports is read as it is: a subject whose chapters
// are the book's own, kept whole, with no check to pass between them. It
// is on the shelf, its chapters are in the contents in reading order, and
// any of them can be opened, since nothing gates them.
func TestAnEPUBIsImportedAndReadAsItIs(t *testing.T) {
	s := newServer(t, "")
	r := httptest.NewRequest("POST", "/readings", bytes.NewReader(epubBytes(t)))
	r.Header.Set("Content-Type", "application/epub+zip")
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("import: %d %s", w.Code, w.Body)
	}
	var made struct {
		ID       string `json:"id"`
		Title    string `json:"title"`
		Chapters int    `json:"chapters"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &made); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(made.ID, "read-") || made.Title != "A Borrowed Book" || made.Chapters != 2 {
		t.Fatalf("imported: %+v", made)
	}

	w = do(t, s, "GET", "/subjects", "", "")
	var shelf struct {
		Subjects []struct {
			ID    string `json:"id"`
			Kind  string `json:"kind"`
			Title string `json:"title"`
		} `json:"subjects"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &shelf); err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, row := range shelf.Subjects {
		if row.ID == made.ID {
			found = true
			if row.Kind != KindReading || row.Title != "A Borrowed Book" {
				t.Errorf("shelf row: %+v", row)
			}
		}
	}
	if !found {
		t.Fatalf("not on the shelf: %+v", shelf.Subjects)
	}

	// The book opens at its first chapter, whole, with nothing to answer.
	w = do(t, s, "POST", "/exchange", `{"subject":"`+made.ID+`","phase":"start"}`, "")
	if w.Code != http.StatusOK {
		t.Fatalf("open: %d %s", w.Code, w.Body)
	}
	var opened struct {
		Chapter *struct {
			Unit  string `json:"unit"`
			Title string `json:"title"`
			HTML  string `json:"html"`
			Check []any  `json:"check"`
		} `json:"chapter"`
		State struct {
			Spine []struct {
				Unit     string `json:"unit"`
				Title    string `json:"title"`
				InFringe bool   `json:"in_fringe"`
			} `json:"spine"`
		} `json:"state"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &opened); err != nil {
		t.Fatal(err)
	}
	if opened.Chapter == nil {
		t.Fatalf("no chapter: %s", w.Body)
	}
	if opened.Chapter.Title != "What Lending Is" {
		t.Errorf("chapter title = %q", opened.Chapter.Title)
	}
	if !strings.Contains(opened.Chapter.HTML, "money now against money later") {
		t.Errorf("the chapter is not the book's prose:\n%s", opened.Chapter.HTML)
	}
	if len(opened.Chapter.Check) != 0 {
		t.Errorf("a book read as it is has no check: %+v", opened.Chapter.Check)
	}
	if len(opened.State.Spine) != 2 {
		t.Fatalf("spine: %+v", opened.State.Spine)
	}
	if opened.State.Spine[1].Title != "Who Lends, and Why" {
		t.Errorf("second chapter = %q", opened.State.Spine[1].Title)
	}
	for _, e := range opened.State.Spine {
		if !e.InFringe {
			t.Errorf("every chapter can be opened; %s cannot", e.Unit)
		}
	}

	// Any chapter opens on request: the reader picks from the contents.
	w = do(t, s, "POST", "/exchange", `{"subject":"`+made.ID+`","phase":"start","choice":"`+opened.State.Spine[1].Unit+`"}`, "")
	var second struct {
		Chapter *struct {
			Title string `json:"title"`
			HTML  string `json:"html"`
		} `json:"chapter"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &second); err != nil {
		t.Fatal(err)
	}
	if second.Chapter == nil || second.Chapter.Title != "Who Lends, and Why" {
		t.Fatalf("second chapter: %s", w.Body)
	}
	if !strings.Contains(second.Chapter.HTML, "deposits cost less") {
		t.Errorf("second chapter's prose:\n%s", second.Chapter.HTML)
	}
}

// A restart finds the imported books again.
func TestImportedBooksComeBackAfterARestart(t *testing.T) {
	s := newServer(t, "")
	r := httptest.NewRequest("POST", "/readings", bytes.NewReader(epubBytes(t)))
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("import: %d %s", w.Code, w.Body)
	}
	var made struct {
		ID string `json:"id"`
	}
	json.Unmarshal(w.Body.Bytes(), &made)

	again := newServerAt(t, s)
	if _, ok := again.subject(made.ID); !ok {
		t.Fatalf("%s is not on the shelf after a restart", made.ID)
	}
}
