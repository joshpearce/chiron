package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// A shelf is a named folder in the library: books and primers the reader
// keeps together. A subject is on at most one shelf; the rest of the
// library is the top level. Shelves live in one file beside the
// active-subject record, so every device sees the same ones.
type Shelf struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Subjects []string `json:"subjects"`
}

type shelves struct {
	mu   sync.Mutex
	path string
	list []*Shelf
}

func loadShelves(path string) *shelves {
	sh := &shelves{path: path}
	if path == "" {
		return sh
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			log.Printf("shelves: %v", err)
		}
		return sh
	}
	if err := json.Unmarshal(data, &sh.list); err != nil {
		log.Printf("shelves: %s: %v", path, err)
	}
	return sh
}

// save writes the list; the caller holds mu.
func (sh *shelves) save() error {
	if sh.path == "" {
		return nil
	}
	data, err := json.MarshalIndent(sh.list, "", "  ")
	if err != nil {
		return err
	}
	tmp := sh.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, sh.path)
}

// view is the list as the library reports it: subjects that no longer
// exist are left out, so a discarded draft does not haunt its shelf.
func (sh *shelves) view(exists func(string) bool) []Shelf {
	sh.mu.Lock()
	defer sh.mu.Unlock()
	out := make([]Shelf, 0, len(sh.list))
	for _, s := range sh.list {
		v := Shelf{ID: s.ID, Name: s.Name, Subjects: []string{}}
		for _, id := range s.Subjects {
			if exists(id) {
				v.Subjects = append(v.Subjects, id)
			}
		}
		out = append(out, v)
	}
	return out
}

// shelfOf is the shelf a subject is on, or "".
func (sh *shelves) shelfOf(subject string) string {
	sh.mu.Lock()
	defer sh.mu.Unlock()
	for _, s := range sh.list {
		for _, id := range s.Subjects {
			if id == subject {
				return s.ID
			}
		}
	}
	return ""
}

func (sh *shelves) find(id string) *Shelf {
	for _, s := range sh.list {
		if s.ID == id {
			return s
		}
	}
	return nil
}

func (sh *shelves) create(name string) (*Shelf, error) {
	sh.mu.Lock()
	defer sh.mu.Unlock()
	var b [6]byte
	rand.Read(b[:])
	s := &Shelf{ID: hex.EncodeToString(b[:]), Name: name, Subjects: []string{}}
	sh.list = append(sh.list, s)
	return s, sh.save()
}

func (sh *shelves) rename(id, name string) (*Shelf, error) {
	sh.mu.Lock()
	defer sh.mu.Unlock()
	s := sh.find(id)
	if s == nil {
		return nil, os.ErrNotExist
	}
	s.Name = name
	return s, sh.save()
}

// remove deletes a shelf; what it held is simply on no shelf, which is
// the top level of the library.
func (sh *shelves) remove(id string) error {
	sh.mu.Lock()
	defer sh.mu.Unlock()
	for i, s := range sh.list {
		if s.ID == id {
			sh.list = append(sh.list[:i], sh.list[i+1:]...)
			return sh.save()
		}
	}
	return os.ErrNotExist
}

// move puts a subject on a shelf ("" for none), taking it off any other.
func (sh *shelves) move(subject, shelf string) error {
	sh.mu.Lock()
	defer sh.mu.Unlock()
	if shelf != "" && sh.find(shelf) == nil {
		return os.ErrNotExist
	}
	for _, s := range sh.list {
		kept := s.Subjects[:0]
		for _, id := range s.Subjects {
			if id != subject {
				kept = append(kept, id)
			}
		}
		s.Subjects = kept
	}
	if shelf != "" {
		s := sh.find(shelf)
		s.Subjects = append(s.Subjects, subject)
	}
	return sh.save()
}

// shelvesPath is beside the active-subject record.
func shelvesPath(activePath string) string {
	if activePath == "" {
		return ""
	}
	return filepath.Join(filepath.Dir(activePath), "shelves.json")
}

// subjectExists: a registered subject, or a primer still on its way.
func (s *Server) subjectExists(id string) bool {
	if _, ok := s.subject(id); ok {
		return true
	}
	s.primersMu.Lock()
	defer s.primersMu.Unlock()
	_, ok := s.primers[id]
	return ok
}

func (s *Server) handleShelves(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"shelves": s.shelves.view(s.subjectExists)})
}

type shelfRequest struct {
	Name string `json:"name"`
}

func (s *Server) handleShelfCreate(w http.ResponseWriter, r *http.Request) {
	var req shelfRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "malformed request: %v", err)
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeError(w, http.StatusUnprocessableEntity, "a shelf needs a name")
		return
	}
	shelf, err := s.shelves.create(name)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "save shelves: %v", err)
		return
	}
	writeJSON(w, http.StatusOK, shelf)
}

func (s *Server) handleShelfRename(w http.ResponseWriter, r *http.Request) {
	var req shelfRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "malformed request: %v", err)
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeError(w, http.StatusUnprocessableEntity, "a shelf needs a name")
		return
	}
	shelf, err := s.shelves.rename(r.PathValue("shelf"), name)
	if errors.Is(err, os.ErrNotExist) {
		writeError(w, http.StatusNotFound, "no shelf %q", r.PathValue("shelf"))
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "save shelves: %v", err)
		return
	}
	writeJSON(w, http.StatusOK, shelf)
}

func (s *Server) handleShelfDelete(w http.ResponseWriter, r *http.Request) {
	err := s.shelves.remove(r.PathValue("shelf"))
	if errors.Is(err, os.ErrNotExist) {
		writeError(w, http.StatusNotFound, "no shelf %q", r.PathValue("shelf"))
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "save shelves: %v", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"shelf": r.PathValue("shelf"), "deleted": true})
}

// Move a subject onto a shelf, or off every shelf with "".
func (s *Server) handleSubjectShelf(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("subject")
	if !s.subjectExists(id) {
		writeError(w, http.StatusNotFound, "no subject %q", id)
		return
	}
	var req struct {
		Shelf string `json:"shelf"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "malformed request: %v", err)
		return
	}
	err := s.shelves.move(id, req.Shelf)
	if errors.Is(err, os.ErrNotExist) {
		writeError(w, http.StatusNotFound, "no shelf %q", req.Shelf)
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "save shelves: %v", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"subject": id, "shelf": req.Shelf})
}
