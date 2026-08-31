package httpapi

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

// With more than one book on the shelf, the tablet needs to reopen on the one
// the reader last had open. The server is the only durable place to keep that:
// the client has no writable storage, and the sprite suspends between sessions,
// so the marker must survive a restart.
func newTwoSubjectServer(t *testing.T, stateRoot string) *Server {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", "server"))
	if err != nil {
		t.Fatal(err)
	}
	corpusDir := filepath.Join("..", "corpus")
	cfg := &Config{
		Subjects: []SubjectSpec{
			{ID: "ai", Title: "How AI Works",
				CorpusDir: corpusDir, StateDir: filepath.Join(stateRoot, "ai")},
			{ID: "data", Title: "Where the Words Come From",
				CorpusDir: corpusDir, StateDir: filepath.Join(stateRoot, "data")},
		},
		Session: SessionConfig{
			MasteryGate: 0.8, ExtensionTrigger: 0.9, CheckItems: 9,
			CallbackFraction: 0.4, ChunkMinutes: 22, BreakMinutes: 5,
			LongBreakEveryChunks: 4,
		},
	}
	for _, spec := range cfg.Subjects {
		if err := os.MkdirAll(filepath.Join(stateRoot, spec.ID), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	s, err := New(cfg, root)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	t.Cleanup(s.renders.Wait)
	return s
}

func activeSubjectOf(t *testing.T, s *Server) string {
	t.Helper()
	w := do(t, s, "GET", "/subjects", "", "")
	if w.Code != http.StatusOK {
		t.Fatalf("/subjects -> %d", w.Code)
	}
	var body struct {
		Active string `json:"active"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	return body.Active
}

func TestActiveSubjectFollowsTheReaderAndSurvivesRestart(t *testing.T) {
	stateRoot := t.TempDir()
	s := newTwoSubjectServer(t, stateRoot)

	if got := activeSubjectOf(t, s); got != "" {
		t.Fatalf("fresh server active = %q, want empty (client falls back to its default)", got)
	}

	// Opening a book - even before its first chapter exists - makes it the
	// active one. The meta fetch is the one request every open performs.
	do(t, s, "GET", "/pages/data", "", "")
	if got := activeSubjectOf(t, s); got != "data" {
		t.Fatalf("after reading data, active = %q, want data", got)
	}

	do(t, s, "GET", "/pages/ai", "", "")
	if got := activeSubjectOf(t, s); got != "ai" {
		t.Fatalf("after reading ai, active = %q, want ai", got)
	}

	// An unknown subject must not disturb the marker.
	do(t, s, "GET", "/pages/nope", "", "")
	if got := activeSubjectOf(t, s); got != "ai" {
		t.Fatalf("after unknown subject, active = %q, want ai", got)
	}

	s.renders.Wait()
	s2 := newTwoSubjectServer(t, stateRoot)
	if got := activeSubjectOf(t, s2); got != "ai" {
		t.Fatalf("after restart, active = %q, want ai", got)
	}
}
