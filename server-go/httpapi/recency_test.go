package httpapi

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

// The shelf can be ordered by recency, so every row says when it last
// changed or was last opened: a book the reader just opened comes first.

func updatedAt(t *testing.T, s *Server, id string) time.Time {
	t.Helper()
	w := do(t, s, "GET", "/subjects", "", "")
	if w.Code != http.StatusOK {
		t.Fatalf("subjects -> %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Subjects []struct {
			ID        string `json:"id"`
			UpdatedAt string `json:"updated_at"`
		} `json:"subjects"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	for _, r := range resp.Subjects {
		if r.ID != id {
			continue
		}
		at, err := time.Parse(time.RFC3339, r.UpdatedAt)
		if err != nil {
			t.Fatalf("%s updated_at %q: %v", id, r.UpdatedAt, err)
		}
		return at
	}
	t.Fatalf("no row for %s", id)
	return time.Time{}
}

func TestEveryRowSaysWhenItLastChangedAndOpeningMovesItForward(t *testing.T) {
	s := newServer(t, "")
	before := updatedAt(t, s, "ai")
	if before.IsZero() || before.After(time.Now()) {
		t.Fatalf("a book never opened still has a stamp, from its corpus: %v", before)
	}

	// Stamps are whole seconds; the learner record was written this one.
	time.Sleep(1100 * time.Millisecond)
	s.markActive("ai")
	after := updatedAt(t, s, "ai")
	if !after.After(before) || time.Since(after) > time.Minute {
		t.Fatalf("opened: before %v after %v", before, after)
	}

	// The stamp is kept on disk: a restart still knows.
	again, err := New(s.cfg, s.root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(again.renders.Wait)
	if got := updatedAt(t, again, "ai"); !got.Equal(after) {
		t.Fatalf("after a restart %v, was %v", got, after)
	}
}

func TestADocumentIsRecentWhenItIsReadOrImported(t *testing.T) {
	s := newServer(t, "")
	w := do(t, s, "POST", "/documents?title=Notes&pages=3", "%PDF-1.4 stub", "")
	if w.Code != http.StatusOK {
		t.Fatalf("upload -> %d: %s", w.Code, w.Body.String())
	}
	var d Document
	json.Unmarshal(w.Body.Bytes(), &d)
	imported := updatedAt(t, s, d.ID)
	if time.Since(imported) > time.Minute {
		t.Fatalf("imported %v", imported)
	}
	time.Sleep(1100 * time.Millisecond)
	if w := do(t, s, "PUT", "/documents/"+d.ID+"/position", `{"page":2,"position":0.5}`, ""); w.Code != http.StatusOK {
		t.Fatalf("position -> %d: %s", w.Code, w.Body.String())
	}
	if read := updatedAt(t, s, d.ID); !read.After(imported) {
		t.Fatalf("reading it did not move it: %v then %v", imported, read)
	}
}
