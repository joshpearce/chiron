package httpapi

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// The paper check-in: strokes go in, transcriptions are graded through the
// normal exchange, and the response carries the gate plus the transcripts.
// The transcriber is injected - no vision model in tests.
func TestInkCheckInGradesTranscriptions(t *testing.T) {
	s := newServer(t, "")
	sub, _ := s.subject("ai")

	// Screener first (through ink, as the paper client would), then the
	// series arrives and the unit is active with known item ids.
	do(t, s, "POST", "/exchange", `{"subject":"ai","phase":"start"}`, "")
	s.transcribe = func(tag string, png []byte) (string, error) { return "3", nil }
	stroke0 := `[[{"x":0.5,"y":0.5},{"x":0.6,"y":0.6}]]`
	screen := do(t, s, "POST", "/ink/ai",
		`{"unit":"u0","items":[{"item_id":"u0-s1","strokes":`+stroke0+`}]}`, "")
	var first struct {
		Chapter struct {
			Check []struct {
				ID string `json:"id"`
			} `json:"check"`
		} `json:"chapter"`
	}
	if err := json.Unmarshal(screen.Body.Bytes(), &first); err != nil {
		t.Fatal(err)
	}
	if len(first.Chapter.Check) < 3 {
		t.Fatalf("series not delivered after screener: %v", first.Chapter.Check)
	}

	// The "handwriting" answers the transcriber will return, keyed by the
	// item id it is called with (the tag): the reference answer for item 1,
	// garbage for item 2. Everything else sends no strokes.
	firstID := first.Chapter.Check[0].ID
	firstQ, _ := sub.Corpus.FindQuestion(firstID)
	if firstQ == nil || firstQ.Check == "llm" {
		t.Fatalf("first series item %s is not mechanically checkable", firstID)
	}
	firstAnswer := firstQ.Answer.String()
	transcribed := map[string]bool{}
	s.transcribe = func(tag string, png []byte) (string, error) {
		transcribed[tag] = true
		if tag == firstID {
			return firstAnswer, nil
		}
		return "no idea, sorry", nil
	}

	stroke := `[[{"x":0.3,"y":0.4},{"x":0.5,"y":0.6}]]`
	items := []string{
		`{"item_id":"` + first.Chapter.Check[0].ID + `","strokes":` + stroke + `}`,
		`{"item_id":"` + first.Chapter.Check[1].ID + `","strokes":` + stroke + `}`,
		// A ticked "I don't know" WITH stray ink: the toggle must win and
		// the transcriber must never see the page.
		`{"item_id":"` + first.Chapter.Check[2].ID + `","idk":true,"strokes":` + stroke + `}`,
	}
	for _, it := range first.Chapter.Check[3:] {
		items = append(items, `{"item_id":"`+it.ID+`","strokes":[]}`)
	}
	body := `{"unit":"u0","items":[` + strings.Join(items, ",") + `]}`

	w := do(t, s, "POST", "/ink/ai", body, "")
	if w.Code != 200 {
		t.Fatalf("ink -> %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Gate *struct {
			Passed      bool `json:"passed"`
			Calibration bool `json:"calibration"`
		} `json:"gate"`
		Results []struct {
			ItemID  string `json:"item_id"`
			Verdict string `json:"verdict"`
		} `json:"results"`
		Transcripts map[string]string `json:"transcripts"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Gate == nil || !resp.Gate.Passed || !resp.Gate.Calibration {
		t.Fatalf("gate = %+v", resp.Gate)
	}
	verdicts := map[string]string{}
	for _, r := range resp.Results {
		verdicts[r.ItemID] = r.Verdict
	}
	if v := verdicts[first.Chapter.Check[0].ID]; v != "pass" {
		t.Errorf("correct transcription graded %q, want pass", v)
	}
	if v := verdicts[first.Chapter.Check[1].ID]; v == "pass" {
		t.Errorf("wrong transcription graded pass")
	}
	if got := resp.Transcripts[first.Chapter.Check[0].ID]; got != firstAnswer {
		t.Errorf("transcript = %q", got)
	}
	idkID := first.Chapter.Check[2].ID
	if v := verdicts[idkID]; v != "fail" {
		t.Errorf("explicit IDK graded %q, want fail", v)
	}
	if transcribed[idkID] {
		t.Error("IDK item was sent to the transcriber - the toggle must win")
	}

	// The async build must settle before the test's temp dir is cleaned.
	for range 100 {
		if building, _ := sub.buildStatus(); !building {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	// The graded artifacts are kept for audit.
	pngPath := filepath.Join(sub.StateDir, "ink", "u0",
		first.Chapter.Check[0].ID+".png")
	if _, err := os.Stat(pngPath); err != nil {
		t.Errorf("rasterized answer not kept: %v", err)
	}
}

var _ = fmt.Sprintf

// A typed answer must reach grading verbatim - never the transcriber - and
// still land in the transcript record (READ AS shows what the system read,
// typed or inked).
func TestTypedAnswerSkipsTranscriber(t *testing.T) {
	s := newServer(t, "")
	do(t, s, "POST", "/exchange", `{"subject":"ai","phase":"start"}`, "")
	s.transcribe = func(tag string, png []byte) (string, error) { return "3", nil }
	screen := do(t, s, "POST", "/ink/ai",
		`{"unit":"u0","items":[{"item_id":"u0-s1","selected_index":0}]}`, "")
	var first struct {
		Chapter struct {
			Check []struct {
				ID string `json:"id"`
			} `json:"check"`
		} `json:"chapter"`
	}
	if err := json.Unmarshal(screen.Body.Bytes(), &first); err != nil {
		t.Fatal(err)
	}
	if len(first.Chapter.Check) == 0 {
		t.Fatal("no series delivered")
	}
	s.transcribe = func(tag string, png []byte) (string, error) {
		t.Errorf("transcriber called for %s despite typed answer", tag)
		return "", nil
	}
	stray := `[[{"x":0.3,"y":0.4},{"x":0.5,"y":0.6}]]`
	items := []string{`{"item_id":"` + first.Chapter.Check[0].ID +
		`","text":"  -4  ","strokes":` + stray + `,"confidence":3}`}
	for _, it := range first.Chapter.Check[1:] {
		items = append(items, `{"item_id":"`+it.ID+`","idk":true}`)
	}
	w := do(t, s, "POST", "/ink/ai",
		`{"unit":"u0","items":[`+strings.Join(items, ",")+`]}`, "")
	if w.Code != 200 {
		t.Fatalf("ink -> %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Transcripts map[string]string `json:"transcripts"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if got := resp.Transcripts[first.Chapter.Check[0].ID]; got != "-4" {
		t.Fatalf("typed transcript = %q, want -4", got)
	}
}
