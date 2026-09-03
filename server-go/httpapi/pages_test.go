package httpapi

import (
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"testing"

	"github.com/mjbraun/chiron/server/render"
)

func TestPagesEndpoints(t *testing.T) {
	if os.Getenv("CHIRON_RENDER") == "0" {
		t.Skip("CHIRON_RENDER=0: no browser renders in this run")
	}
	if _, err := exec.LookPath("pdftoppm"); err != nil {
		t.Skip("pdftoppm not installed")
	}
	s := newServer(t, "tok")
	sub, _ := s.subject("ai")
	if sub.Pages.KatexDir == "" {
		sub.Pages.KatexDir = "../../ipad-app/Chiron/Resources/katex"
	}

	ch := &render.Chapter{
		Unit:  "u1",
		Title: "Pages test",
		HTML:  `<h1>Test</h1><p>Inline math $a^2+b^2=c^2$ should render.</p>`,
	}
	if err := persistChapter(sub, ch); err != nil {
		t.Fatalf("persist: %v", err)
	}
	unit := "u1"
	sub.Learner.Data.CurrentUnit = &unit

	w := do(t, s, "GET", "/pages/ai", "", "tok")
	if w.Code != 200 {
		t.Fatalf("meta: %d %s", w.Code, w.Body.String())
	}
	var meta struct {
		Unit  string `json:"unit"`
		Count int    `json:"count"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &meta); err != nil {
		t.Fatal(err)
	}
	if meta.Unit != "u1" || meta.Count < 1 {
		t.Fatalf("bad meta: %+v", meta)
	}

	w = do(t, s, "GET", "/pages/ai/0", "", "tok")
	if w.Code != 200 || w.Header().Get("Content-Type") != "image/png" {
		t.Fatalf("page 0: %d %s", w.Code, w.Header().Get("Content-Type"))
	}
	if len(w.Body.Bytes()) < 1000 {
		t.Fatalf("suspiciously small png: %d bytes", w.Body.Len())
	}

	w = do(t, s, "GET", "/pages/ai/99", "", "tok")
	if w.Code != 404 {
		t.Fatalf("out of range page: %d", w.Code)
	}
}

// An explicit "I don't know" must grade as a fail without touching a model -
// the test server has no working LLM, so a model call would surface as an
// ungraded or error verdict.
func TestIDKGradesAsFailWithoutModel(t *testing.T) {
	s := newServer(t, "tok")

	do(t, s, "POST", "/exchange", `{"subject":"ai","phase":"start"}`, "tok")
	// A real series item, not the screener - the screener never fails.
	id := "u0-q1"
	body := `{"subject":"ai","phase":"boundary","unit":"u0","check_responses":[` +
		`{"item_id":"` + id + `","idk":true,"confidence":1}]}`
	w := do(t, s, "POST", "/exchange", body, "tok")
	if w.Code != 200 {
		t.Fatalf("exchange: %d %s", w.Code, w.Body.String())
	}
	var resp struct {
		Results []struct {
			ItemID  string `json:"item_id"`
			Verdict string `json:"verdict"`
		} `json:"results"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Results) == 0 {
		t.Fatal("no results")
	}
	if resp.Results[0].Verdict != "fail" {
		t.Fatalf("IDK verdict = %q, want fail", resp.Results[0].Verdict)
	}
}

// With rendering off the book still works end to end; only page images
// are refused, and as unavailable rather than broken.
func TestRenderOffServesTheBookWithoutPages(t *testing.T) {
	t.Setenv("CHIRON_RENDER", "0")
	s := newServer(t, "")
	if w := do(t, s, "POST", "/exchange", `{"subject":"ai","phase":"start"}`, ""); w.Code != http.StatusOK {
		t.Fatalf("start -> %d: %s", w.Code, w.Body.String())
	}
	if w := do(t, s, "GET", "/pages/ai", "", ""); w.Code != http.StatusServiceUnavailable {
		t.Fatalf("/pages/ai -> %d: %s", w.Code, w.Body.String())
	}
	if w := do(t, s, "GET", "/pages/ai/contents", "", ""); w.Code != http.StatusServiceUnavailable {
		t.Fatalf("/pages/ai/contents -> %d: %s", w.Code, w.Body.String())
	}
}
