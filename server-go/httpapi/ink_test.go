package httpapi

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The paper check-in: strokes go in, transcriptions are graded through the
// normal exchange, and the response carries the gate plus the transcripts.
// The transcriber is injected - no vision model in tests.
func TestInkCheckInGradesTranscriptions(t *testing.T) {
	s := newServer(t, "")
	sub, _ := s.subject("ai")

	// Calibration chapter first, so the unit is active and item ids are known.
	start := do(t, s, "POST", "/exchange", `{"subject":"ai","phase":"start"}`, "")
	var first struct {
		Chapter struct {
			Check []struct {
				ID string `json:"id"`
			} `json:"check"`
		} `json:"chapter"`
	}
	if err := json.Unmarshal(start.Body.Bytes(), &first); err != nil {
		t.Fatal(err)
	}

	// The "handwriting" answers the transcriber will return: right answer
	// for item 1 (dot product of [2,0,-3].[1,4,2] = -4), garbage for item 2,
	// everything else unanswered (no strokes at all).
	s.transcribe = func(hint string, png []byte) (string, error) {
		if strings.Contains(hint, "dot product") {
			return "-4", nil
		}
		return "no idea, sorry", nil
	}

	stroke := `[[{"x":0.3,"y":0.4},{"x":0.5,"y":0.6}]]`
	items := []string{
		`{"item_id":"` + first.Chapter.Check[0].ID + `","strokes":` + stroke + `}`,
		`{"item_id":"` + first.Chapter.Check[1].ID + `","strokes":` + stroke + `}`,
	}
	for _, it := range first.Chapter.Check[2:] {
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
	if got := resp.Transcripts[first.Chapter.Check[0].ID]; got != "-4" {
		t.Errorf("transcript = %q", got)
	}

	// The graded artifacts are kept for audit.
	pngPath := filepath.Join(sub.StateDir, "ink", "u0",
		first.Chapter.Check[0].ID+".png")
	if _, err := os.Stat(pngPath); err != nil {
		t.Errorf("rasterized answer not kept: %v", err)
	}
}

var _ = fmt.Sprintf
