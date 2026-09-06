package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// Each device keeps a unit's marks, ink and position with the server;
// the version is what tells a clean update from two copies that both
// changed while apart.

func annotationsOf(t *testing.T, s *Server, unit string) (Annotations, int) {
	t.Helper()
	w := do(t, s, "GET", "/annotations/ai/"+unit, "", "")
	var a Annotations
	json.Unmarshal(w.Body.Bytes(), &a)
	return a, w.Code
}

func TestAnnotationsRoundTripWithVersions(t *testing.T) {
	s := newServer(t, "")
	if _, code := annotationsOf(t, s, "u1"); code != http.StatusNotFound {
		t.Fatalf("empty -> %d", code)
	}
	w := do(t, s, "PUT", "/annotations/ai/u1", `{"marks":[{"id":"m1","kind":"highlight","text":"tokens"}],"ink_b64":"AAA=","position":0.4,"device":"iPad","base_version":0}`, "")
	if w.Code != http.StatusOK {
		t.Fatalf("put -> %d: %s", w.Code, w.Body.String())
	}
	a, _ := annotationsOf(t, s, "u1")
	if a.Version != 1 || a.Position != 0.4 || a.InkB64 != "AAA=" || a.Device != "iPad" || !strings.Contains(string(a.Marks), `"id":"m1"`) {
		t.Fatalf("stored = %+v", a)
	}

	// The same content again is not a new version.
	w = do(t, s, "PUT", "/annotations/ai/u1", `{"marks":[{"id":"m1","kind":"highlight","text":"tokens"}],"ink_b64":"AAA=","position":0.4,"base_version":1}`, "")
	var again Annotations
	json.Unmarshal(w.Body.Bytes(), &again)
	if w.Code != http.StatusOK || again.Version != 1 {
		t.Fatalf("unchanged put -> %d version %d", w.Code, again.Version)
	}

	// A change on top of the current version goes through.
	w = do(t, s, "PUT", "/annotations/ai/u1", `{"marks":[{"id":"m1","kind":"highlight","text":"tokens"},{"id":"m2","kind":"question","text":"logits","question":"why?"}],"position":0.6,"base_version":1}`, "")
	if w.Code != http.StatusOK {
		t.Fatalf("second put -> %d: %s", w.Code, w.Body.String())
	}
	if a, _ := annotationsOf(t, s, "u1"); a.Version != 2 || a.Position != 0.6 {
		t.Fatalf("after second put = %+v", a)
	}
	// It counts as the reader being in the book.
	if s.activeSubject() != "ai" {
		t.Fatal("a put should mark the book active")
	}
}

func TestTwoCopiesThatBothChangedAreAConflict(t *testing.T) {
	s := newServer(t, "")
	do(t, s, "PUT", "/annotations/ai/u1", `{"marks":[{"id":"m1"}],"position":0.1,"base_version":0}`, "")
	// The phone writes on top of version 1.
	do(t, s, "PUT", "/annotations/ai/u1", `{"marks":[{"id":"m1"},{"id":"phone"}],"position":0.2,"base_version":1}`, "")
	// The iPad, still at version 1, writes something else.
	w := do(t, s, "PUT", "/annotations/ai/u1", `{"marks":[{"id":"m1"},{"id":"ipad"}],"position":0.3,"base_version":1}`, "")
	if w.Code != http.StatusConflict {
		t.Fatalf("stale put -> %d: %s", w.Code, w.Body.String())
	}
	var rep struct {
		Conflict bool        `json:"conflict"`
		Server   Annotations `json:"server"`
	}
	json.Unmarshal(w.Body.Bytes(), &rep)
	if !rep.Conflict || rep.Server.Version != 2 || !strings.Contains(string(rep.Server.Marks), "phone") {
		t.Fatalf("conflict reply = %s", w.Body.String())
	}
	// Nothing was overwritten.
	if a, _ := annotationsOf(t, s, "u1"); a.Version != 2 || strings.Contains(string(a.Marks), "ipad") {
		t.Fatalf("server copy = %+v", a)
	}
}

func TestReconcileKeepsEverythingFromBothSides(t *testing.T) {
	s := newServer(t, "")
	body := `{"mine":{"version":1,"marks":[{"id":"m1","kind":"highlight","text":"a"},{"id":"ipad","kind":"question","text":"b","question":"why b?","answer":"because"}],"ink_b64":"MINE","position":0.3},
	          "theirs":{"version":2,"marks":[{"id":"m1","kind":"highlight","text":"a"},{"id":"phone","kind":"highlight","text":"c"}],"ink_b64":"THEIRS","position":0.2}}`
	w := do(t, s, "POST", "/annotations/ai/u1/reconcile", body, "")
	if w.Code != http.StatusOK {
		t.Fatalf("reconcile -> %d: %s", w.Code, w.Body.String())
	}
	var rep struct {
		Version  int             `json:"version"`
		Marks    json.RawMessage `json:"marks"`
		Position float64         `json:"position"`
		Ink      string          `json:"ink_b64"`
		Other    string          `json:"ink_other_b64"`
	}
	json.Unmarshal(w.Body.Bytes(), &rep)
	var marks []map[string]any
	json.Unmarshal(rep.Marks, &marks)
	ids := []string{}
	for _, m := range marks {
		ids = append(ids, m["id"].(string))
	}
	if strings.Join(ids, ",") != "m1,ipad,phone" || rep.Position != 0.3 || rep.Version != 2 || rep.Ink != "MINE" || rep.Other != "THEIRS" {
		t.Fatalf("merged = %s", w.Body.String())
	}
}

func TestAnnotationsRefuseBadIds(t *testing.T) {
	s := newServer(t, "")
	if w := do(t, s, "GET", "/annotations/nope/u1", "", ""); w.Code != http.StatusNotFound {
		t.Fatalf("unknown subject -> %d", w.Code)
	}
	if w := do(t, s, "PUT", "/annotations/ai/bad%20id", `{}`, ""); w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("bad unit -> %d", w.Code)
	}
}
