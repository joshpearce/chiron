package httpapi

import (
	"net/http"
	"strings"
	"testing"
)

// Reset throws away a learner's whole history, so it must not be reachable by a
// malformed or replayed request that happens to carry the right token.
func TestResetRequiresExplicitConfirmation(t *testing.T) {
	s := newServer(t, "")
	sub, _ := s.subject("ai")

	if w := do(t, s, "POST", "/exchange", `{"subject":"ai","phase":"start"}`, ""); w.Code != http.StatusOK {
		t.Fatalf("setup exchange -> %d", w.Code)
	}
	if sub.Learner.Snapshot().Version == 0 {
		t.Fatal("setup did not record any events")
	}

	if w := do(t, s, "POST", "/reset", `{"subject":"ai"}`, ""); w.Code != http.StatusUnprocessableEntity {
		t.Errorf("reset without confirm -> %d, want 422", w.Code)
	}
	if sub.Learner.Snapshot().Version == 0 {
		t.Error("an unconfirmed reset wiped the state anyway")
	}
	if w := do(t, s, "POST", "/reset", `{"subject":"nope","confirm":true}`, ""); w.Code != http.StatusNotFound {
		t.Errorf("reset of an unknown subject -> %d, want 404", w.Code)
	}

	w := do(t, s, "POST", "/reset", `{"subject":"ai","confirm":true}`, "")
	if w.Code != http.StatusOK {
		t.Fatalf("confirmed reset -> %d: %s", w.Code, w.Body.String())
	}
	snap := sub.Learner.Snapshot()
	if snap.Version != 0 || len(snap.Units) != 0 || len(snap.Concepts) != 0 {
		t.Errorf("state survived the reset: version=%d units=%d concepts=%d",
			snap.Version, len(snap.Units), len(snap.Concepts))
	}
	if fringe := sub.Learner.Fringe(); len(fringe) != 1 || fringe[0] != "u0" {
		t.Errorf("fringe after reset is %v, want [u0]", fringe)
	}
	// The response is the fresh state, so the client can adopt it directly.
	if !strings.Contains(w.Body.String(), `"spine"`) {
		t.Error("reset should return the new state payload")
	}
}
