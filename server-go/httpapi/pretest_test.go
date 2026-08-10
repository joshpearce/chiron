package httpapi

import (
	"net/http/httptest"
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
