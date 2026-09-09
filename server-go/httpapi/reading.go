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
	// The book's pictures come with it: re-encoded for a screen, kept
	// beside the book, and the chapters point at them by name.
	pictures, err := storeReadingAssets(root, id, file, chapters)
	if err != nil {
		log.Printf("imported book %s: pictures: %v", id, err)
	}
	if err := writeReadingCorpus(readingCorpus(root, id), title, withAssetPaths(chapters, pictures)); err != nil {
		os.RemoveAll(dir)
		writeError(w, http.StatusInternalServerError, "lay the book out: %v", err)
		return
	}
	meta := &Reading{ID: id, Title: title, Chapters: len(readingUnits(chapters)), Size: size,
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

// storeReadingAssets unpacks the book's pictures beside it, each one
// re-encoded for a screen, and says what each of the chapters' own paths
// is now called.
func storeReadingAssets(root, id, file string, chapters []sources.EPUBChapter) (map[string]string, error) {
	assets, names, err := sources.EPUBAssets(file, chapters)
	if err != nil {
		return nil, err
	}
	if len(assets) == 0 {
		return nil, nil
	}
	dir := readingAssets(root, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	renamed := map[string]string{}
	for _, a := range assets {
		name, data := screenAsset(a.Name, a.Data)
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
			return nil, err
		}
		renamed[a.Name] = name
	}
	out := map[string]string{}
	for ref, name := range names {
		if to, ok := renamed[name]; ok {
			out[ref] = to
		}
	}
	return out, nil
}

// withAssetPaths points a chapter's pictures at the book's own asset
// route, so the reader asks the server for them rather than the archive.
func withAssetPaths(chapters []sources.EPUBChapter, pictures map[string]string) []sources.EPUBChapter {
	if len(pictures) == 0 {
		return chapters
	}
	out := make([]sources.EPUBChapter, len(chapters))
	for i, ch := range chapters {
		md := ch.Markdown
		for ref, name := range pictures {
			md = strings.ReplaceAll(md, "]("+ref+")", "](assets/"+name+")")
		}
		out[i] = sources.EPUBChapter{Title: ch.Title, Markdown: md}
	}
	return out
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

// A chapter of a real book runs past twenty thousand words, which is one
// scroll the length of a small book and more ink than a canvas wants to
// hold; over this it is cut at its own headings.
const longChapterWords = 4000

// A page with almost nothing on it is the cover, a part title, a licence:
// part of the book, but not a chapter of it.
const emptyChapterWords = 50

// readingUnit is one unit of an imported book: a chapter, or a part of a
// chapter that was too long to read in one scroll.
type readingUnit struct {
	title    string
	markdown string
}

// readingUnits cuts the book into what the reader turns between: whole
// chapters where they are short enough, and a chapter's own sections
// where it is not.
func readingUnits(chapters []sources.EPUBChapter) []readingUnit {
	var out []readingUnit
	// A book of short pieces is not a book of covers: the rule that drops
	// the empty pages only applies where something is left after it.
	substantial := 0
	for _, ch := range chapters {
		if words(stripLeadingHeading(ch.Markdown)) >= emptyChapterWords {
			substantial++
		}
	}
	for _, ch := range chapters {
		body := strings.TrimSpace(stripLeadingHeading(ch.Markdown))
		if body == "" {
			continue
		}
		if substantial > 0 && words(body) < emptyChapterWords {
			continue
		}
		parts := splitAtHeadings(body)
		if words(body) <= longChapterWords || len(parts) < 2 {
			out = append(out, readingUnit{title: ch.Title, markdown: body})
			continue
		}
		// The contents should read as the book's own does: the chapter
		// named once, then its sections under it.
		for i, p := range parts {
			title := p.heading
			switch {
			case i == 0 && p.heading == "":
				title = ch.Title
			case i == 0:
				title = ch.Title + " · " + p.heading
			case title == "":
				title = fmt.Sprintf("%s · part %d", ch.Title, i+1)
			}
			out = append(out, readingUnit{title: title, markdown: p.body})
		}
	}
	return out
}

type headedPart struct {
	heading string
	body    string
}

// splitAtHeadings cuts markdown at its "## " headings, keeping whatever
// comes before the first with it. A part too short to be worth turning to
// joins the one before it.
func splitAtHeadings(md string) []headedPart {
	var parts []headedPart
	var heading string
	var body []string
	flush := func() {
		text := strings.TrimSpace(strings.Join(body, "\n"))
		if text == "" && heading == "" {
			return
		}
		if n := len(parts); n > 0 && words(text) < emptyChapterWords {
			parts[n-1].body = parts[n-1].body + "\n\n## " + heading + "\n\n" + text
			return
		}
		parts = append(parts, headedPart{heading: heading, body: text})
	}
	for _, line := range strings.Split(md, "\n") {
		if strings.HasPrefix(line, "## ") {
			flush()
			heading = strings.TrimSpace(strings.TrimPrefix(line, "## "))
			body = nil
			continue
		}
		body = append(body, line)
	}
	flush()
	return parts
}

func words(s string) int { return len(strings.Fields(s)) }

// writeReadingCorpus lays the book out as a corpus: a unit per chapter,
// in the book's own order, with no prerequisite between them, so the
// contents opens any of them. There is nothing to answer, so there is no
// bank and no questions file.
func writeReadingCorpus(dir, title string, chapters []sources.EPUBChapter) error {
	units := readingUnits(chapters)
	if len(units) == 0 {
		return fmt.Errorf("the book has no chapters")
	}
	var syllabus strings.Builder
	fmt.Fprintf(&syllabus, "title: %s\nunits:\n", yamlQuote(title))
	for i, u := range units {
		id := fmt.Sprintf("u%d", i+1)
		slug := slugOf(u.title, id)
		fmt.Fprintf(&syllabus, "  - id: %s\n    slug: %s\n    title: %s\n    minutes: %d\n",
			id, slug, yamlQuote(u.title), readingMinutes(u.markdown))
		body := "## " + u.title + "\n\n" + u.markdown + "\n"
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
