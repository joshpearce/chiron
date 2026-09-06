package roles

import (
	"encoding/json"
	"strings"
	"testing"
)

// Two copies of one unit's marks, merged: every mark survives, the
// richer copy of a shared mark wins, and threads both sides grew are
// joined with no turn lost.
func TestReconcileMarksKeepsBothSides(t *testing.T) {
	mine := json.RawMessage(`[{"id":"h1","kind":"highlight","text":"a"},{"id":"q1","kind":"question","text":"b","question":"why?","answer":"because","thread":[{"question":"and then?","answer":"then this"}]}]`)
	theirs := json.RawMessage(`[{"id":"h1","kind":"highlight","text":"a"},{"id":"q1","kind":"question","text":"b","question":"why?","answer":"because","thread":[{"question":"but why not?","answer":"since"}]},{"id":"h2","kind":"highlight","text":"c"}]`)
	out, err := ReconcileMarks(nil, mine, theirs)
	if err != nil {
		t.Fatal(err)
	}
	var marks []map[string]any
	json.Unmarshal(out, &marks)
	if len(marks) != 3 || marks[0]["id"] != "h1" || marks[1]["id"] != "q1" || marks[2]["id"] != "h2" {
		t.Fatalf("marks = %s", out)
	}
	thread, _ := json.Marshal(marks[1]["thread"])
	if !strings.Contains(string(thread), "and then?") || !strings.Contains(string(thread), "but why not?") {
		t.Fatalf("thread lost a turn: %s", thread)
	}

	// A copy with an answer beats one without.
	mine = json.RawMessage(`[{"id":"q1","kind":"question","text":"b","question":"why?"}]`)
	theirs = json.RawMessage(`[{"id":"q1","kind":"question","text":"b","question":"why?","answer":"because"}]`)
	out, _ = ReconcileMarks(nil, mine, theirs)
	if !strings.Contains(string(out), `"answer":"because"`) {
		t.Fatalf("the richer copy should win: %s", out)
	}

	// Identical copies are one, and empty sides are fine.
	out, _ = ReconcileMarks(nil, theirs, theirs)
	json.Unmarshal(out, &marks)
	if len(marks) != 1 {
		t.Fatalf("identical copies doubled: %s", out)
	}
	if out, err := ReconcileMarks(nil, nil, theirs); err != nil || !strings.Contains(string(out), "q1") {
		t.Fatalf("empty side: %s %v", out, err)
	}
}
