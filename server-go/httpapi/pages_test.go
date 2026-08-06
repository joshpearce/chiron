package httpapi

import (
	"encoding/json"
	"os/exec"
	"testing"

	"github.com/mjbraun/chiron/server/render"
)

func TestPagesEndpoints(t *testing.T) {
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
