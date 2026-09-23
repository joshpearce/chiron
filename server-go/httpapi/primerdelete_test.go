package httpapi

import (
	"net/http"
	"strings"
	"testing"
)

// A primer that has been written is still the reader's to be rid of: one
// that missed the point, or was captured from the wrong thing. Discard only
// ever took drafts, so a finished primer could not be taken off the shelf
// from any client at all.
func TestAFinishedPrimerCanBeTakenOffTheShelf(t *testing.T) {
	s := newServer(t, "")
	s.chain = &cannedChain{payload: primerPayload()}

	rep := capture(t, s, `{"text":"Each hop appends a caveat.","prompt":"How do macaroons attenuate?"}`)
	s.renders.Wait() // authoring runs in the background
	made := struct{ Subject string }{Subject: rep.Subject}
	if row, ok := shelf(t, s)[made.Subject]; !ok || row.Status != "ready" {
		t.Fatalf("the primer was not written: %+v", row)
	}
	if _, ok := s.subject(made.Subject); !ok {
		t.Fatalf("%s is not on the shelf", made.Subject)
	}

	w := do(t, s, "DELETE", "/primer/"+made.Subject, "", "")
	if w.Code != http.StatusOK {
		t.Fatalf("delete: %d %s", w.Code, w.Body)
	}
	if _, ok := s.subject(made.Subject); ok {
		t.Errorf("%s is still on the shelf", made.Subject)
	}
	w = do(t, s, "GET", "/subjects", "", "")
	if strings.Contains(w.Body.String(), made.Subject) {
		t.Errorf("the shelf still lists it:\n%s", w.Body)
	}

	// Gone for good, including after a restart.
	again := newServerAt(t, s)
	if _, ok := again.subject(made.Subject); ok {
		t.Errorf("it came back after a restart")
	}

	// Deleting what is not there says so rather than pretending.
	if w := do(t, s, "DELETE", "/primer/"+made.Subject, "", ""); w.Code != http.StatusNotFound {
		t.Errorf("deleting it twice -> %d", w.Code)
	}
	if w := do(t, s, "DELETE", "/primer/ai", "", ""); w.Code != http.StatusNotFound {
		t.Errorf("deleting a book through the primer route -> %d", w.Code)
	}
}
