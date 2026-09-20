package devreq

import (
	"encoding/json"
	"testing"
)

// A request typed in the app lands here queued, with the app's state and
// screenshot beside it; the agent takes the oldest queued one, and every
// change it makes to the request is what the app then shows.
func TestARequestIsQueuedListedAndTakenInOrder(t *testing.T) {
	s := Open(t.TempDir())
	first, err := s.Create("make the pen thicker", json.RawMessage(`{"screen":"reading"}`), []byte("PNG"))
	if err != nil {
		t.Fatal(err)
	}
	second, _ := s.Create("and bluer", nil, nil)
	if first.Status != Queued || first.Text != "make the pen thicker" || first.ID == "" || first.ID == second.ID {
		t.Errorf("first = %+v", first)
	}
	if first.Screenshot == "" || second.Screenshot != "" {
		t.Errorf("screenshots: %q %q", first.Screenshot, second.Screenshot)
	}
	if string(first.State) != `{"screen":"reading"}` {
		t.Errorf("state = %s", first.State)
	}

	list, _ := s.List()
	if len(list) != 2 || list[0].ID != second.ID {
		t.Errorf("list newest first: %v", ids(list))
	}
	next, _ := s.NextQueued()
	if next == nil || next.ID != first.ID {
		t.Errorf("next queued should be the oldest: %+v", next)
	}

	next.Status = Working
	next.Say("worktree made")
	if err := s.Save(next); err != nil {
		t.Fatal(err)
	}
	got, _ := s.Get(first.ID)
	if got.Status != Working || len(got.Log) != 1 || got.Last != "worktree made" {
		t.Errorf("saved = %+v", got)
	}
	next2, _ := s.NextQueued()
	if next2 == nil || next2.ID != second.ID {
		t.Errorf("after taking the first, next is the second: %+v", next2)
	}
	if _, err := s.Get("req-nonesuch"); err == nil {
		t.Error("a missing request is an error")
	}
	if _, err := s.Create("   ", nil, nil); err == nil {
		t.Error("an empty request is refused")
	}
}

func ids(rs []*Request) []string {
	out := []string{}
	for _, r := range rs {
		out = append(out, r.ID)
	}
	return out
}
