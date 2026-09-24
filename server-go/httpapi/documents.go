package httpapi

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// A Document is a PDF the reader put on the shelf. The server keeps the
// file and where the reader is in it, so both devices open it at the
// same page.
type Document struct {
	ID         string  `json:"id"`
	Title      string  `json:"title"`
	Pages      int     `json:"pages"`
	Size       int64   `json:"size"`
	ImportedAt string  `json:"imported_at"`
	Page       int     `json:"page"`
	Position   float64 `json:"position"`
}

const maxDocumentBytes = 200 << 20

func (s *Server) documentsDir() string {
	if s.activePath != "" {
		return filepath.Join(filepath.Dir(s.activePath), "documents")
	}
	return filepath.Join(s.root, "state", "documents")
}

func (s *Server) documentPath(id string) string {
	return filepath.Join(s.documentsDir(), id, "document.json")
}

func (s *Server) documentFile(id string) string {
	return filepath.Join(s.documentsDir(), id, id+".pdf")
}

func (s *Server) readDocument(id string) (*Document, error) {
	if !strings.HasPrefix(id, "doc-") || !slugOK.MatchString(id) {
		return nil, os.ErrNotExist
	}
	data, err := os.ReadFile(s.documentPath(id))
	if err != nil {
		return nil, err
	}
	var d Document
	if err := json.Unmarshal(data, &d); err != nil {
		return nil, err
	}
	return &d, nil
}

func (s *Server) writeDocument(d *Document) error {
	path := s.documentPath(d.ID)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(d)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// documents lists every document on the shelf, oldest first.
func (s *Server) documents() []*Document {
	entries, err := os.ReadDir(s.documentsDir())
	if err != nil {
		return nil
	}
	var out []*Document
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if d, err := s.readDocument(e.Name()); err == nil {
			out = append(out, d)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ImportedAt < out[j].ImportedAt })
	return out
}

func (s *Server) documentExists(id string) bool {
	_, err := s.readDocument(id)
	return err == nil
}

func (s *Server) handleDocumentUpload(w http.ResponseWriter, r *http.Request) {
	title := strings.TrimSpace(r.URL.Query().Get("title"))
	if title == "" {
		writeError(w, http.StatusUnprocessableEntity, "a document needs a title")
		return
	}
	pages, _ := strconv.Atoi(r.URL.Query().Get("pages"))
	data, err := io.ReadAll(io.LimitReader(r.Body, maxDocumentBytes+1))
	if err != nil {
		writeError(w, http.StatusBadRequest, "read body: %v", err)
		return
	}
	if len(data) > maxDocumentBytes {
		writeError(w, http.StatusRequestEntityTooLarge, "documents are at most %d MB", maxDocumentBytes>>20)
		return
	}
	if !bytes.HasPrefix(data, []byte("%PDF")) {
		writeError(w, http.StatusUnprocessableEntity, "not a PDF")
		return
	}
	var raw [4]byte
	rand.Read(raw[:])
	d := &Document{
		ID: "doc-" + hex.EncodeToString(raw[:]), Title: title, Pages: pages, Size: int64(len(data)),
		ImportedAt: time.Now().UTC().Format(time.RFC3339),
	}
	if err := os.MkdirAll(filepath.Dir(s.documentFile(d.ID)), 0o755); err != nil {
		writeError(w, http.StatusInternalServerError, "keep document: %v", err)
		return
	}
	if err := os.WriteFile(s.documentFile(d.ID), data, 0o644); err != nil {
		writeError(w, http.StatusInternalServerError, "keep document: %v", err)
		return
	}
	if err := s.writeDocument(d); err != nil {
		writeError(w, http.StatusInternalServerError, "keep document: %v", err)
		return
	}
	writeJSON(w, http.StatusOK, d)
}

func (s *Server) document(w http.ResponseWriter, r *http.Request) (*Document, bool) {
	d, err := s.readDocument(r.PathValue("doc"))
	if errors.Is(err, os.ErrNotExist) {
		writeError(w, http.StatusNotFound, "no document %q", r.PathValue("doc"))
		return nil, false
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "read document: %v", err)
		return nil, false
	}
	return d, true
}

func (s *Server) handleDocumentGet(w http.ResponseWriter, r *http.Request) {
	if d, ok := s.document(w, r); ok {
		writeJSON(w, http.StatusOK, d)
	}
}

func (s *Server) handleDocumentFile(w http.ResponseWriter, r *http.Request) {
	d, ok := s.document(w, r)
	if !ok {
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	s.markOpened(d.ID)
	http.ServeFile(w, r, s.documentFile(d.ID))
}

func (s *Server) handleDocumentPosition(w http.ResponseWriter, r *http.Request) {
	d, ok := s.document(w, r)
	if !ok {
		return
	}
	var req struct {
		Page     int     `json:"page"`
		Position float64 `json:"position"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "malformed request: %v", err)
		return
	}
	d.Page, d.Position = req.Page, req.Position
	s.markOpened(d.ID)
	if err := s.writeDocument(d); err != nil {
		writeError(w, http.StatusInternalServerError, "save position: %v", err)
		return
	}
	writeJSON(w, http.StatusOK, d)
}

func (s *Server) handleDocumentDelete(w http.ResponseWriter, r *http.Request) {
	d, ok := s.document(w, r)
	if !ok {
		return
	}
	if err := os.RemoveAll(filepath.Dir(s.documentPath(d.ID))); err != nil {
		writeError(w, http.StatusInternalServerError, "forget document: %v", err)
		return
	}
	s.opened.forget(d.ID)
	writeJSON(w, http.StatusOK, map[string]any{"deleted": d.ID})
}

// PageInk is the drawing on one page of a document, versioned so two
// devices that both drew on it while apart are caught.
type PageInk struct {
	Version   int    `json:"version"`
	UpdatedAt string `json:"updated_at"`
	InkB64    string `json:"ink_b64"`
}

func (s *Server) inkPath(id string, page int) string {
	return filepath.Join(s.documentsDir(), id, "ink", strconv.Itoa(page)+".json")
}

func (s *Server) readPageInk(id string, page int) (*PageInk, error) {
	data, err := os.ReadFile(s.inkPath(id, page))
	if err != nil {
		return nil, err
	}
	var ink PageInk
	if err := json.Unmarshal(data, &ink); err != nil {
		return nil, err
	}
	return &ink, nil
}

func (s *Server) handleDocumentInkAll(w http.ResponseWriter, r *http.Request) {
	d, ok := s.document(w, r)
	if !ok {
		return
	}
	pages := map[string]*PageInk{}
	entries, _ := os.ReadDir(filepath.Join(s.documentsDir(), d.ID, "ink"))
	for _, e := range entries {
		name := strings.TrimSuffix(e.Name(), ".json")
		page, err := strconv.Atoi(name)
		if err != nil || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		if ink, err := s.readPageInk(d.ID, page); err == nil {
			pages[name] = ink
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"document": d.ID, "pages": pages})
}

func (s *Server) handleDocumentInkPut(w http.ResponseWriter, r *http.Request) {
	d, ok := s.document(w, r)
	if !ok {
		return
	}
	page, err := strconv.Atoi(r.PathValue("page"))
	if err != nil || page < 0 {
		writeError(w, http.StatusUnprocessableEntity, "page %q", r.PathValue("page"))
		return
	}
	var req struct {
		InkB64      string `json:"ink_b64"`
		BaseVersion int    `json:"base_version"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "malformed request: %v", err)
		return
	}
	annotationsMu.Lock()
	defer annotationsMu.Unlock()
	current, err := s.readPageInk(d.ID, page)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		writeError(w, http.StatusInternalServerError, "read ink: %v", err)
		return
	}
	if current != nil {
		if current.InkB64 == req.InkB64 {
			writeJSON(w, http.StatusOK, current)
			return
		}
		if req.BaseVersion != current.Version {
			writeJSON(w, http.StatusConflict, map[string]any{
				"conflict": true, "server": current,
				"detail": "both copies of this page changed since they last agreed",
			})
			return
		}
	}
	next := &PageInk{Version: 1, UpdatedAt: time.Now().UTC().Format(time.RFC3339), InkB64: req.InkB64}
	if current != nil {
		next.Version = current.Version + 1
	}
	path := s.inkPath(d.ID, page)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		writeError(w, http.StatusInternalServerError, "keep ink: %v", err)
		return
	}
	data, _ := json.Marshal(next)
	werr := os.WriteFile(path+".tmp", data, 0o644)
	if werr == nil {
		werr = os.Rename(path+".tmp", path)
	}
	if werr != nil {
		writeError(w, http.StatusInternalServerError, "keep ink: %v", werr)
		return
	}
	writeJSON(w, http.StatusOK, next)
}
