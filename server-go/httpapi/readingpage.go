package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/mjbraun/chiron/server/sources"
)

// A page the reader is reading somewhere else and wants to read here: a
// blog post, a release note, a piece of documentation. It comes in as a
// reading like an imported book does, so it is on the shelf, it opens in
// the reader, and a highlight in it reaches the tutor with the whole
// piece behind it rather than the sentence alone.
//
// The server fetches it, not the device: the page's pictures are kept
// beside it the way a book's are, so it reads the same on a plane as it
// does at a desk.

// A page is a page: if it has not come back in this long, it is not one.
const pageFetchTimeout = 90 * time.Second

// More pictures than this on one page is a gallery, not a piece of
// writing, and the reader is waiting.
const maxPagePictures = 40

func (s *Server) handleReadingPage(w http.ResponseWriter, r *http.Request) {
	var body struct {
		URL   string `json:"url"`
		Title string `json:"title"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "read the request: %v", err)
		return
	}
	raw := strings.TrimSpace(body.URL)
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		writeError(w, http.StatusBadRequest, "a page is an http or https address; %q is not", raw)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), pageFetchTimeout)
	defer cancel()
	client := sources.NewClient("")
	page, err := client.Page(ctx, raw)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "this page did not come back to be read: %v", err)
		return
	}
	title := strings.TrimSpace(body.Title)
	if title == "" {
		title = page.Title
	}
	if title == "" {
		title = u.Host
	}

	var id [4]byte
	rand.Read(id[:])
	readingID := "read-" + hex.EncodeToString(id[:])
	root := s.readingsRoot()
	dir := readingDir(root, readingID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		writeError(w, http.StatusInternalServerError, "keep the page: %v", err)
		return
	}
	pictures := storePageAssets(ctx, client, root, readingID, page.Images)
	chapters := withAssetPaths([]sources.EPUBChapter{{Title: title, Markdown: page.Markdown}}, pictures)
	if err := writeReadingCorpus(readingCorpus(root, readingID), title, chapters); err != nil {
		os.RemoveAll(dir)
		writeError(w, http.StatusUnprocessableEntity, "lay the page out: %v", err)
		return
	}
	meta := &Reading{ID: readingID, Title: title, Chapters: len(readingUnits(chapters)),
		Size: int64(len(page.Markdown)), ImportedAt: time.Now().UTC().Format(time.RFC3339), URL: raw}
	if err := writeReadingMeta(root, meta); err != nil {
		os.RemoveAll(dir)
		writeError(w, http.StatusInternalServerError, "keep the page: %v", err)
		return
	}
	if err := s.registerReading(meta); err != nil {
		os.RemoveAll(dir)
		writeError(w, http.StatusUnprocessableEntity, "read the page: %v", err)
		return
	}
	writeJSON(w, http.StatusOK, meta)
}

// storePageAssets keeps the page's pictures beside it, each re-encoded
// for a screen, and says what each of the page's own URLs is now called.
// A picture that will not come is left where it is: the words are the
// point, and the page still reads with a gap in it.
func storePageAssets(ctx context.Context, c *sources.Client, root, id string, urls []string) map[string]string {
	if len(urls) == 0 {
		return nil
	}
	if len(urls) > maxPagePictures {
		urls = urls[:maxPagePictures]
	}
	dir := readingAssets(root, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Printf("page %s: pictures: %v", id, err)
		return nil
	}
	out := map[string]string{}
	for i, raw := range urls {
		data, err := c.Get(ctx, raw)
		if err != nil {
			log.Printf("page %s: picture %s: %v", id, raw, err)
			continue
		}
		name, encoded := screenAsset(pictureName(i, raw), data)
		if err := os.WriteFile(filepath.Join(dir, name), encoded, 0o644); err != nil {
			log.Printf("page %s: picture %s: %v", id, raw, err)
			continue
		}
		out[raw] = name
	}
	return out
}

// pictureName is what a picture is called once it is the page's own: its
// place on the page, its name where the URL gives one, and nothing that
// could be read as a path.
func pictureName(i int, raw string) string {
	name, ext := "picture", ""
	if u, err := url.Parse(raw); err == nil {
		base := path.Base(u.Path)
		ext = strings.ToLower(strings.TrimPrefix(path.Ext(base), "."))
		if slug := slugOf(strings.TrimSuffix(base, path.Ext(base)), ""); slug != "" {
			name = slug
		}
	}
	if ext = slugOf(ext, ""); ext == "" || len(ext) > 4 {
		ext = "img"
	}
	return fmt.Sprintf("%d-%s.%s", i+1, name, ext)
}
