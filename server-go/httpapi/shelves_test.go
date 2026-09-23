package httpapi

import (
	"encoding/json"
	"net/http"
	"testing"
)

// The library can be organised into shelves: named folders that hold
// books and primers. A shelf is renamed in place; deleting one returns
// its contents to the library. Both devices read the same shelves.

type shelfView struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Subjects []string `json:"subjects"`
}

func library(t *testing.T, s *Server) (map[string]string, []shelfView) {
	t.Helper()
	w := do(t, s, "GET", "/subjects", "", "")
	var resp struct {
		Subjects []struct {
			ID    string `json:"id"`
			Shelf string `json:"shelf"`
		} `json:"subjects"`
		Shelves []shelfView `json:"shelves"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	where := map[string]string{}
	for _, r := range resp.Subjects {
		where[r.ID] = r.Shelf
	}
	return where, resp.Shelves
}

func TestAShelfHoldsSubjectsUntilItIsDeleted(t *testing.T) {
	s := newServer(t, "")
	w := do(t, s, "POST", "/shelves", `{"name":"Systems"}`, "")
	if w.Code != http.StatusOK {
		t.Fatalf("create -> %d: %s", w.Code, w.Body.String())
	}
	var made shelfView
	json.Unmarshal(w.Body.Bytes(), &made)
	if made.ID == "" || made.Name != "Systems" {
		t.Fatalf("made = %+v", made)
	}

	if w := do(t, s, "PUT", "/subjects/ai/shelf", `{"shelf":"`+made.ID+`"}`, ""); w.Code != http.StatusOK {
		t.Fatalf("move in -> %d: %s", w.Code, w.Body.String())
	}
	where, shelves := library(t, s)
	if where["ai"] != made.ID || len(shelves) != 1 || shelves[0].Name != "Systems" || len(shelves[0].Subjects) != 1 {
		t.Fatalf("where = %v shelves = %+v", where, shelves)
	}

	if w := do(t, s, "PUT", "/shelves/"+made.ID, `{"name":"Systems and models"}`, ""); w.Code != http.StatusOK {
		t.Fatalf("rename -> %d: %s", w.Code, w.Body.String())
	}
	if _, shelves := library(t, s); shelves[0].Name != "Systems and models" {
		t.Fatalf("shelves = %+v", shelves)
	}

	// A restart reads the shelves back.
	s2 := newServerAt(t, s)
	if where, shelves := library(t, s2); where["ai"] != made.ID || len(shelves) != 1 {
		t.Fatalf("after restart: where = %v shelves = %+v", where, shelves)
	}

	if w := do(t, s2, "DELETE", "/shelves/"+made.ID, "", ""); w.Code != http.StatusOK {
		t.Fatalf("delete -> %d: %s", w.Code, w.Body.String())
	}
	if where, shelves := library(t, s2); where["ai"] != "" || len(shelves) != 0 {
		t.Fatalf("after delete: where = %v shelves = %+v", where, shelves)
	}
}

func TestASubjectIsOnOneShelfAndCanLeaveIt(t *testing.T) {
	s := newServer(t, "")
	var a, b shelfView
	json.Unmarshal(do(t, s, "POST", "/shelves", `{"name":"A"}`, "").Body.Bytes(), &a)
	json.Unmarshal(do(t, s, "POST", "/shelves", `{"name":"B"}`, "").Body.Bytes(), &b)
	do(t, s, "PUT", "/subjects/ai/shelf", `{"shelf":"`+a.ID+`"}`, "")
	do(t, s, "PUT", "/subjects/ai/shelf", `{"shelf":"`+b.ID+`"}`, "")
	where, shelves := library(t, s)
	if where["ai"] != b.ID || len(shelves[0].Subjects) != 0 || len(shelves[1].Subjects) != 1 {
		t.Fatalf("where = %v shelves = %+v", where, shelves)
	}
	if w := do(t, s, "PUT", "/subjects/ai/shelf", `{"shelf":""}`, ""); w.Code != http.StatusOK {
		t.Fatalf("move out -> %d: %s", w.Code, w.Body.String())
	}
	if where, _ := library(t, s); where["ai"] != "" {
		t.Fatalf("where = %v", where)
	}
}

func TestShelvesRefuseWhatMakesNoSense(t *testing.T) {
	s := newServer(t, "")
	if w := do(t, s, "POST", "/shelves", `{"name":"  "}`, ""); w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("blank name -> %d", w.Code)
	}
	if w := do(t, s, "PUT", "/shelves/nope", `{"name":"x"}`, ""); w.Code != http.StatusNotFound {
		t.Fatalf("rename unknown -> %d", w.Code)
	}
	if w := do(t, s, "DELETE", "/shelves/nope", "", ""); w.Code != http.StatusNotFound {
		t.Fatalf("delete unknown -> %d", w.Code)
	}
	if w := do(t, s, "PUT", "/subjects/ai/shelf", `{"shelf":"nope"}`, ""); w.Code != http.StatusNotFound {
		t.Fatalf("move to unknown -> %d", w.Code)
	}
	if w := do(t, s, "PUT", "/subjects/nope/shelf", `{"shelf":""}`, ""); w.Code != http.StatusNotFound {
		t.Fatalf("move unknown subject -> %d", w.Code)
	}
}

// newServerAt is the same server after a restart: the same config and
// state directories, read back from disk.
func newServerAt(t *testing.T, prev *Server) *Server {
	t.Helper()
	prev.renders.Wait()
	if err := prev.Close(); err != nil {
		t.Fatal(err)
	}
	s, err := New(prev.cfg, prev.root)
	if err != nil {
		t.Fatalf("restart: %v", err)
	}
	t.Cleanup(func() { s.renders.Wait(); _ = s.Close() })
	return s
}
