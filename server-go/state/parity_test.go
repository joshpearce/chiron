package state

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mjbraun/chiron/server/corpus"
)

// The Go implementation must read a snapshot the Python one wrote, in place and
// without a migration - that is the whole point of keeping the format. The file
// is copied to a temp dir first so running the tests can never touch a real
// learner's progress.
func TestReadsPythonWrittenSnapshot(t *testing.T) {
	src := filepath.Join("..", "..", "state", "ai", "learner.json")
	raw, err := os.ReadFile(src)
	if os.IsNotExist(err) {
		t.Skip("no Python-written state on this machine")
	}
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "learner.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}

	c, err := corpus.Load(filepath.Join("..", "..", "corpus"))
	if err != nil {
		t.Fatal(err)
	}
	l, err := Open(dir, c)
	if err != nil {
		t.Fatalf("could not load the Python-written snapshot: %v", err)
	}
	if l.Data.Concepts == nil || l.Data.Units == nil || l.Data.Misconceptions == nil {
		t.Error("maps came back nil - a write would panic")
	}
	if l.Data.Summary == "" {
		t.Error("summary did not survive the round trip")
	}
	// Opening an existing snapshot must not rewrite it.
	after, err := os.ReadFile(filepath.Join(dir, "learner.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(raw) {
		t.Error("opening an existing snapshot rewrote it")
	}
	t.Logf("version=%d concepts=%d units=%d debt=%d fringe=%v",
		l.Data.Version, len(l.Data.Concepts), len(l.Data.Units), len(l.Data.Debt), l.Fringe())
}
