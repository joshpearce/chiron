package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/mjbraun/chiron/server/sources"
)

// A book the reader owns and imports, read as it is: an EPUB is HTML, so
// it keeps its own words and reflows with the reader's type size, the way
// a chapter of a written book does. Its chapters are the book's own, no
// check stands between them, and every one can be opened from the
// contents. What the reader adds is theirs: highlights, questions, ink
// and passages sent on.
//
// It is a subject like any other, so annotations, positions and the ask
// card all work without knowing anything about EPUBs.

const KindReading = "reading"

// A real book with its figures runs to a hundred megabytes and more; the
// text in it is a fraction of that, but the file is the reader's and is
// kept whole.
const maxReadingBytes = 256 << 20

// Reading is what the shelf and a restart need to know about an imported
// book. The corpus beside it is what the server reads it through.
type Reading struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Chapters   int    `json:"chapters"`
	Size       int64  `json:"size"`
	ImportedAt string `json:"imported_at"`
}

func (s *Server) readingsRoot() string {
	if s.cfg.ReadingsDir != "" {
		return resolve(s.root, s.cfg.ReadingsDir)
	}
	return filepath.Join(filepath.Dir(s.root), "state", "readings")
}

func readingDir(root, id string) string    { return filepath.Join(root, id) }
func readingCorpus(root, id string) string { return filepath.Join(root, id, "corpus") }
func readingState(root, id string) string  { return filepath.Join(root, id, "state") }
func readingMeta(root, id string) string   { return filepath.Join(root, id, "reading.json") }

// handleReadingImport takes the EPUB itself and lays it out as a subject:
// one chapter per document of the book's spine, in reading order.
func (s *Server) handleReadingImport(w http.ResponseWriter, r *http.Request) {
	var raw [4]byte
	rand.Read(raw[:])
	id := "read-" + hex.EncodeToString(raw[:])
	root := s.readingsRoot()
	dir := readingDir(root, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		writeError(w, http.StatusInternalServerError, "keep the book: %v", err)
		return
	}
	// Straight to disk: a book with its figures is a hundred megabytes,
	// and the server is reading one to someone while this arrives.
	file := filepath.Join(dir, "book.epub")
	f, err := os.Create(file)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "keep the book: %v", err)
		return
	}
	size, err := io.Copy(f, io.LimitReader(r.Body, maxReadingBytes+1))
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		os.RemoveAll(dir)
		writeError(w, http.StatusBadRequest, "read body: %v", err)
		return
	}
	if size > maxReadingBytes {
		os.RemoveAll(dir)
		writeError(w, http.StatusRequestEntityTooLarge, "books are at most %d MB", maxReadingBytes>>20)
		return
	}

	title, chapters, err := sources.EPUBBook(file)
	if err != nil {
		os.RemoveAll(dir)
		writeError(w, http.StatusUnprocessableEntity, "this is not a book the reader can open: %v", err)
		return
	}
	if given := strings.TrimSpace(r.URL.Query().Get("title")); given != "" {
		title = given
	}
	if title == "" {
		title = "A book"
	}
	if err := writeReadingCorpus(readingCorpus(root, id), title, chapters); err != nil {
		os.RemoveAll(dir)
		writeError(w, http.StatusInternalServerError, "lay the book out: %v", err)
		return
	}
	meta := &Reading{ID: id, Title: title, Chapters: len(chapters), Size: size,
		ImportedAt: time.Now().UTC().Format(time.RFC3339)}
	if err := writeReadingMeta(root, meta); err != nil {
		os.RemoveAll(dir)
		writeError(w, http.StatusInternalServerError, "keep the book: %v", err)
		return
	}
	if err := s.registerReading(meta); err != nil {
		os.RemoveAll(dir)
		writeError(w, http.StatusUnprocessableEntity, "read the book: %v", err)
		return
	}
	writeJSON(w, http.StatusOK, meta)
}

func (s *Server) handleReadingDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	root := s.readingsRoot()
	if _, err := os.Stat(readingMeta(root, id)); err != nil {
		writeError(w, http.StatusNotFound, "no imported book %q", id)
		return
	}
	s.mu.Lock()
	delete(s.subjects, id)
	s.mu.Unlock()
	if err := os.RemoveAll(readingDir(root, id)); err != nil {
		writeError(w, http.StatusInternalServerError, "forget the book: %v", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": id})
}

// loadReadings puts the imported books back on the shelf at boot.
func (s *Server) loadReadings() {
	root := s.readingsRoot()
	entries, err := os.ReadDir(root)
	if err != nil {
		return
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		data, err := os.ReadFile(readingMeta(root, e.Name()))
		if err != nil {
			continue
		}
		var m Reading
		if json.Unmarshal(data, &m) != nil || m.ID == "" {
			continue
		}
		if err := s.registerReading(&m); err != nil {
			log.Printf("imported book %s: %v", m.ID, err)
		}
	}
}

func (s *Server) registerReading(m *Reading) error {
	root := s.readingsRoot()
	if err := s.register(m.ID, m.Title, readingCorpus(root, m.ID), readingState(root, m.ID)); err != nil {
		return err
	}
	s.mu.Lock()
	if sub := s.subjects[m.ID]; sub != nil {
		sub.Kind = KindReading
	}
	s.mu.Unlock()
	return nil
}

func writeReadingMeta(root string, m *Reading) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(readingMeta(root, m.ID), data, 0o644)
}

// writeReadingCorpus lays the book out as a corpus: a unit per chapter,
// in the book's own order, with no prerequisite between them, so the
// contents opens any of them. There is nothing to answer, so there is no
// bank and no questions file.
func writeReadingCorpus(dir, title string, chapters []sources.EPUBChapter) error {
	if len(chapters) == 0 {
		return fmt.Errorf("the book has no chapters")
	}
	var syllabus strings.Builder
	fmt.Fprintf(&syllabus, "title: %s\nunits:\n", yamlQuote(title))
	for i, ch := range chapters {
		id := fmt.Sprintf("u%d", i+1)
		slug := slugOf(ch.Title, id)
		fmt.Fprintf(&syllabus, "  - id: %s\n    slug: %s\n    title: %s\n    minutes: %d\n",
			id, slug, yamlQuote(ch.Title), readingMinutes(ch.Markdown))
		body := "## " + ch.Title + "\n\n" + strings.TrimSpace(stripLeadingHeading(ch.Markdown)) + "\n"
		unit := filepath.Join(dir, "units", id+"-"+slug)
		if err := os.MkdirAll(unit, 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(unit, "canon.md"), []byte(body), 0o644); err != nil {
			return err
		}
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "syllabus.yaml"), []byte(syllabus.String()), 0o644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "misconception-bank.yaml"), []byte("misconceptions: []\n"), 0o644)
}

// stripLeadingHeading drops a chapter's own title line, which the unit
// carries as its heading.
func stripLeadingHeading(md string) string {
	trimmed := strings.TrimLeft(md, "\n")
	if !strings.HasPrefix(trimmed, "# ") {
		return md
	}
	if i := strings.IndexByte(trimmed, '\n'); i >= 0 {
		return trimmed[i+1:]
	}
	return ""
}

// readingMinutes is the reading time at a middling pace, which is what
// the chapter's header shows.
func readingMinutes(md string) int {
	if n := len(strings.Fields(md)) / 200; n > 0 {
		return n
	}
	return 1
}

func slugOf(title, fallback string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(title) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			dash = false
		case !dash && b.Len() > 0:
			b.WriteByte('-')
			dash = true
		}
	}
	slug := strings.Trim(b.String(), "-")
	if len(slug) > 40 {
		slug = strings.Trim(slug[:40], "-")
	}
	if slug == "" {
		return fallback
	}
	return slug
}

func yamlQuote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
