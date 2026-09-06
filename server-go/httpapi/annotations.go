package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/mjbraun/chiron/server/roles"
)

// Annotations are what a reader adds to one unit on one device: the
// marks (highlights, questions with their answers, margin notes), the
// ink, and the reading position. The server keeps one document per unit
// so every device reads the same, versioned so that two devices that
// both changed it offline are caught rather than silently overwritten.
type Annotations struct {
	Version   int             `json:"version"`
	UpdatedAt string          `json:"updated_at"`
	Device    string          `json:"device,omitempty"`
	Marks     json.RawMessage `json:"marks"`
	InkB64    string          `json:"ink_b64,omitempty"`
	Position  float64         `json:"position"`
}

// annotationsMu serialises writes per unit file.
var annotationsMu sync.Mutex

func annotationsPath(sub *Subject, unit string) string {
	return filepath.Join(sub.StateDir, "annotations", unit+".json")
}

func readAnnotations(sub *Subject, unit string) (*Annotations, error) {
	data, err := os.ReadFile(annotationsPath(sub, unit))
	if err != nil {
		return nil, err
	}
	var a Annotations
	if err := json.Unmarshal(data, &a); err != nil {
		return nil, err
	}
	return &a, nil
}

func writeAnnotations(sub *Subject, unit string, a *Annotations) error {
	path := annotationsPath(sub, unit)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(a)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

var unitOK = slugOK

func (s *Server) annotationsSubject(w http.ResponseWriter, r *http.Request) (*Subject, string, bool) {
	sub, ok := s.subject(r.PathValue("subject"))
	if !ok {
		writeError(w, http.StatusNotFound, "unknown subject %q", r.PathValue("subject"))
		return nil, "", false
	}
	unit := r.PathValue("unit")
	if !unitOK.MatchString(unit) {
		writeError(w, http.StatusUnprocessableEntity, "unit id %q", unit)
		return nil, "", false
	}
	return sub, unit, true
}

func (s *Server) handleAnnotationsGet(w http.ResponseWriter, r *http.Request) {
	sub, unit, ok := s.annotationsSubject(w, r)
	if !ok {
		return
	}
	a, err := readAnnotations(sub, unit)
	if errors.Is(err, os.ErrNotExist) {
		writeError(w, http.StatusNotFound, "no annotations for %s/%s", sub.ID, unit)
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "read annotations: %v", err)
		return
	}
	writeJSON(w, http.StatusOK, a)
}

type annotationsPut struct {
	Annotations
	// BaseVersion is the version the device last saw; a put on top of a
	// newer one is a conflict, answered with the server's copy.
	BaseVersion int `json:"base_version"`
}

// sameContent: nothing to write, and no version to bump.
func sameContent(a, b *Annotations) bool {
	return string(compactJSON(a.Marks)) == string(compactJSON(b.Marks)) && a.InkB64 == b.InkB64 && a.Position == b.Position
}

func compactJSON(raw json.RawMessage) []byte {
	if len(raw) == 0 {
		return []byte("[]")
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return raw
	}
	out, _ := json.Marshal(v)
	return out
}

func (s *Server) handleAnnotationsPut(w http.ResponseWriter, r *http.Request) {
	sub, unit, ok := s.annotationsSubject(w, r)
	if !ok {
		return
	}
	var req annotationsPut
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "malformed request: %v", err)
		return
	}
	if len(req.Marks) == 0 {
		req.Marks = json.RawMessage("[]")
	}
	annotationsMu.Lock()
	defer annotationsMu.Unlock()
	current, err := readAnnotations(sub, unit)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		writeError(w, http.StatusInternalServerError, "read annotations: %v", err)
		return
	}
	if current != nil {
		if sameContent(current, &req.Annotations) {
			writeJSON(w, http.StatusOK, current)
			return
		}
		if req.BaseVersion != current.Version {
			writeJSON(w, http.StatusConflict, map[string]any{
				"conflict": true,
				"server":   current,
				"detail":   "both copies changed since they last agreed",
			})
			return
		}
	}
	next := &Annotations{
		Version:   1,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
		Device:    strings.TrimSpace(req.Device),
		Marks:     compactJSON(req.Marks),
		InkB64:    req.InkB64,
		Position:  req.Position,
	}
	if current != nil {
		next.Version = current.Version + 1
	}
	if err := writeAnnotations(sub, unit, next); err != nil {
		writeError(w, http.StatusInternalServerError, "write annotations: %v", err)
		return
	}
	s.markActive(sub.ID)
	writeJSON(w, http.StatusOK, next)
}

// Reconcile merges two copies that both changed: every mark from either
// side, the richer copy of a mark both hold, a merged thread where both
// added to the same question (the tutor writes that one), the further
// reading position, and both inks so the device can lay one over the
// other. The result is not stored: the device puts it back on top of
// the server's version.
func (s *Server) handleAnnotationsReconcile(w http.ResponseWriter, r *http.Request) {
	sub, unit, ok := s.annotationsSubject(w, r)
	if !ok {
		return
	}
	var req struct {
		Mine   Annotations `json:"mine"`
		Theirs Annotations `json:"theirs"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "malformed request: %v", err)
		return
	}
	merged, err := roles.ReconcileMarks(s.chain, req.Mine.Marks, req.Theirs.Marks)
	if err != nil {
		writeError(w, http.StatusBadGateway, "reconcile: %v", err)
		return
	}
	out := map[string]any{
		"subject":  sub.ID,
		"unit":     unit,
		"version":  max(req.Mine.Version, req.Theirs.Version),
		"marks":    merged,
		"position": max(req.Mine.Position, req.Theirs.Position),
	}
	// Ink is opaque here; the device appends the other drawing to its own.
	switch {
	case req.Mine.InkB64 == "" || req.Mine.InkB64 == req.Theirs.InkB64:
		out["ink_b64"] = req.Theirs.InkB64
	case req.Theirs.InkB64 == "":
		out["ink_b64"] = req.Mine.InkB64
	default:
		out["ink_b64"] = req.Mine.InkB64
		out["ink_other_b64"] = req.Theirs.InkB64
	}
	writeJSON(w, http.StatusOK, out)
}
