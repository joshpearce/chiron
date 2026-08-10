package httpapi

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/mjbraun/chiron/server/corpus"
)

// The chapter directly after a calibration unit skips its pretest: the
// series just measured that ground, and two question blocks back to back
// read as a bug, not pedagogy.
func TestFollowsCalibration(t *testing.T) {
	c := &corpus.Corpus{
		Syllabus: corpus.Syllabus{Units: []corpus.UnitSpec{
			{ID: "u0"}, {ID: "u1"}, {ID: "u2"},
		}},
		Units: map[string]*corpus.Unit{
			"u0": {ID: "u0", Front: map[string]any{"calibration": true}},
			"u1": {ID: "u1"},
			"u2": {ID: "u2"},
		},
	}
	for id, want := range map[string]bool{"u0": false, "u1": true, "u2": false, "zz": false} {
		if got := followsCalibration(c, id); got != want {
			t.Errorf("followsCalibration(%s) = %v, want %v", id, got, want)
		}
	}
}

// The e-ink client's Image elements cannot send headers; the token rides as
// a query parameter there. Both forms must authenticate, and a wrong token
// in either form must not.
func TestQueryParamToken(t *testing.T) {
	s := newServer(t, "sekrit")
	r := httptest.NewRequest("GET", "/state?token=sekrit", nil)
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("query token rejected: %d", w.Code)
	}
	r = httptest.NewRequest("GET", "/state?token=wrong", nil)
	w = httptest.NewRecorder()
	s.Handler().ServeHTTP(w, r)
	if w.Code != 401 {
		t.Fatalf("wrong query token accepted: %d", w.Code)
	}
	r = httptest.NewRequest("GET", "/state", nil)
	w = httptest.NewRecorder()
	s.Handler().ServeHTTP(w, r)
	if w.Code != 401 {
		t.Fatalf("missing token accepted: %d", w.Code)
	}
}

// Starting over archives the old book instead of destroying it: chapters,
// results, ink, and the learner log all move under archive/<stamp>, and the
// learner comes back fresh.
func TestResetArchivesInsteadOfDeleting(t *testing.T) {
	s := newServer(t, "")
	do(t, s, "POST", "/exchange", `{"subject":"ai","phase":"start"}`, "")
	sub, _ := s.subject("ai")
	if _, err := os.Stat(filepath.Join(sub.StateDir, "chapters", "u0.json")); err != nil {
		t.Fatalf("no chapter persisted before reset: %v", err)
	}
	w := do(t, s, "POST", "/reset", `{"subject":"ai","confirm":true}`, "")
	if w.Code != 200 {
		t.Fatalf("reset: %d %s", w.Code, w.Body.String())
	}
	entries, err := os.ReadDir(filepath.Join(sub.StateDir, "archive"))
	if err != nil || len(entries) != 1 {
		t.Fatalf("archive dir: %v entries=%d", err, len(entries))
	}
	arch := filepath.Join(sub.StateDir, "archive", entries[0].Name())
	if _, err := os.Stat(filepath.Join(arch, "chapters", "u0.json")); err != nil {
		t.Fatalf("chapter not archived: %v", err)
	}
	if _, err := os.Stat(filepath.Join(sub.StateDir, "chapters", "u0.json")); err == nil {
		t.Fatal("live chapter survived reset")
	}
	if cur := sub.Learner.Data.CurrentUnit; cur != nil && *cur != "" {
		t.Fatalf("learner not fresh after reset: current=%v", *cur)
	}
	// Unconfirmed resets must refuse.
	w = do(t, s, "POST", "/reset", `{"subject":"ai"}`, "")
	if w.Code != 422 {
		t.Fatalf("unconfirmed reset: %d", w.Code)
	}
}
