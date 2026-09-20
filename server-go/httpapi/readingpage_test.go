package httpapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// pageSite is a blog with one post on it, the post carrying a picture.
func pageSite(t *testing.T) *httptest.Server {
	t.Helper()
	var shot bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 40, 30))
	for x := 0; x < 40; x++ {
		for y := 0; y < 30; y++ {
			img.Set(x, y, color.RGBA{uint8(x * 6), uint8(y * 8), 200, 255})
		}
	}
	if err := png.Encode(&shot, img); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/2026/Sep/20/injection/":
			fmt.Fprint(w, `<html><head><meta property="og:title" content="Prompt injection in 2026"></head><body>
<nav><a href="/about/">About</a></nav>
<article>
  <h1>Prompt injection in 2026</h1>
  <p>An agent that reads the web reads whatever an attacker wrote on it, and
  the words it reads are the words it obeys.</p>
  <p><img src="/static/trifecta.png" alt="Where the attack lands"></p>
  <p><img src="/static/board.webp" alt="The whiteboard afterwards"></p>
  <p>The shape to look for is private data, untrusted content and a way out.</p>
</article>
<footer>Copyright 2026</footer></body></html>`)
		case "/static/trifecta.png":
			w.Header().Set("Content-Type", "image/png")
			w.Write(shot.Bytes())
		case "/static/board.webp":
			// A kind Go cannot decode is kept as it came, and still has
			// to be served as what it is.
			w.Header().Set("Content-Type", "image/webp")
			w.Write([]byte("RIFF\x00\x00\x00\x00WEBPVP8 a whiteboard"))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// A page the reader sends in is a reading like an imported book: one
// chapter, the writer's own words, on the shelf and open to a highlight.
func TestAPageIsReadInChiron(t *testing.T) {
	s := newServer(t, "")
	s.cfg.ReadingsDir = t.TempDir()
	site := pageSite(t)
	post := site.URL + "/2026/Sep/20/injection/"

	w := do(t, s, "POST", "/readings/page", `{"url":"`+post+`"}`, "")
	if w.Code != http.StatusOK {
		t.Fatalf("read the page: %d %s", w.Code, w.Body)
	}
	var made struct {
		ID       string `json:"id"`
		Title    string `json:"title"`
		Chapters int    `json:"chapters"`
		URL      string `json:"url"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &made); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(made.ID, "read-") || made.Title != "Prompt injection in 2026" || made.Chapters != 1 {
		t.Fatalf("read: %+v", made)
	}
	if made.URL != post {
		t.Errorf("the page does not remember where it came from: %q", made.URL)
	}

	// It is on the shelf, as a reading.
	w = do(t, s, "GET", "/subjects", "", "")
	var shelf struct {
		Subjects []struct {
			ID, Kind, Title string
		} `json:"subjects"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &shelf); err != nil {
		t.Fatal(err)
	}
	var found bool
	for _, row := range shelf.Subjects {
		if row.ID == made.ID {
			found = true
			if row.Kind != KindReading || row.Title != "Prompt injection in 2026" {
				t.Errorf("shelf row: %+v", row)
			}
		}
	}
	if !found {
		t.Fatalf("the page is not on the shelf: %+v", shelf.Subjects)
	}

	// It opens as the post, with the site's furniture left behind.
	w = do(t, s, "POST", "/exchange", `{"subject":"`+made.ID+`","phase":"start"}`, "")
	var opened struct {
		Chapter *struct {
			Title string `json:"title"`
			HTML  string `json:"html"`
			Check []any  `json:"check"`
		} `json:"chapter"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &opened); err != nil {
		t.Fatal(err)
	}
	if opened.Chapter == nil {
		t.Fatalf("the page did not open: %s", w.Body)
	}
	if !strings.Contains(opened.Chapter.HTML, "the words it reads are the words it obeys") {
		t.Errorf("the post's prose is missing:\n%s", opened.Chapter.HTML)
	}
	if strings.Contains(opened.Chapter.HTML, "Copyright 2026") {
		t.Errorf("the site's footer came with it:\n%s", opened.Chapter.HTML)
	}
	if len(opened.Chapter.Check) != 0 {
		t.Errorf("a page read as it is has no check: %+v", opened.Chapter.Check)
	}

	// Its picture is kept beside it, so the page reads without the site.
	if strings.Contains(opened.Chapter.HTML, site.URL+"/static/trifecta.png") {
		t.Errorf("the picture is still the site's:\n%s", opened.Chapter.HTML)
	}
	entries, err := os.ReadDir(readingAssets(s.readingsRoot(), made.ID))
	if err != nil || len(entries) != 2 {
		t.Fatalf("pictures kept: %v %v", entries, err)
	}
	for _, e := range entries {
		w = do(t, s, "GET", "/readings/"+made.ID+"/assets/"+e.Name(), "", "")
		if w.Code != http.StatusOK || w.Body.Len() == 0 {
			t.Fatalf("the picture %s does not serve: %d", e.Name(), w.Code)
		}
		if strings.HasSuffix(e.Name(), ".webp") && w.Header().Get("Content-Type") != "image/webp" {
			t.Errorf("%s serves as %q", e.Name(), w.Header().Get("Content-Type"))
		}
		if !strings.Contains(opened.Chapter.HTML, e.Name()) {
			t.Errorf("the chapter does not point at the kept picture %s:\n%s", e.Name(), opened.Chapter.HTML)
		}
	}

	// A restart finds it again.
	again := newServerAt(t, s)
	if _, ok := again.subject(made.ID); !ok {
		t.Fatalf("%s is not on the shelf after a restart", made.ID)
	}
}

// A URL that is not a page to read says so, and leaves nothing behind.
func TestAPageThatIsNotThereIsRefused(t *testing.T) {
	s := newServer(t, "")
	dir := t.TempDir()
	s.cfg.ReadingsDir = dir
	site := pageSite(t)
	w := do(t, s, "POST", "/readings/page", `{"url":"`+site.URL+`/nothing-here/"}`, "")
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("a missing page -> %d %s", w.Code, w.Body)
	}
	if w := do(t, s, "POST", "/readings/page", `{"url":"file:///etc/passwd"}`, ""); w.Code != http.StatusBadRequest {
		t.Fatalf("a URL that is not a page -> %d %s", w.Code, w.Body)
	}
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if _, err := os.Stat(filepath.Join(dir, e.Name(), "reading.json")); err == nil {
			t.Fatalf("a refused page was kept: %s", e.Name())
		}
	}
}
