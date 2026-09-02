package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// A dev server (CHIRON_DRIVE=1) can stand in for the vision model with a
// stub transcriber, so a client's handwriting path can be exercised end to
// end without a model: the audit trail (READ AS) shows the stub's text.
func TestDriveModeStubTranscriber(t *testing.T) {
	t.Setenv("CHIRON_DRIVE", "1")
	t.Setenv("CHIRON_TRANSCRIBE", "stub")
	s := newServer(t, "")
	unit, items := walkToSeries(t, s, "ai")

	var parts []string
	for i, id := range items {
		if i == 0 {
			parts = append(parts, `{"item_id":"`+id+`","strokes":[[{"x":0.1,"y":0.2},{"x":0.5,"y":0.6}],[{"x":0.7,"y":0.2},{"x":0.7,"y":0.8}]],"aspect":3,"confidence":3}`)
		} else {
			parts = append(parts, `{"item_id":"`+id+`","idk":true}`)
		}
	}
	w := do(t, s, "POST", "/ink/ai", `{"unit":"`+unit+`","items":[`+strings.Join(parts, ",")+`]}`, "")
	if w.Code != http.StatusOK {
		t.Fatalf("/ink/ai -> %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Transcripts map[string]string `json:"transcripts"`
		ResultsDoc  struct {
			Entries []struct {
				ReadAs string `json:"read_as"`
			} `json:"entries"`
		} `json:"results_doc"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if got := resp.Transcripts[items[0]]; !strings.Contains(got, "stub") || !strings.Contains(got, items[0]) {
		t.Fatalf("stub transcript = %q, want it to name itself and the item", got)
	}
	if got := resp.ResultsDoc.Entries[0].ReadAs; !strings.Contains(got, "stub") {
		t.Fatalf("READ AS = %q, want the stub transcription", got)
	}
}

// Outside drive mode the stub is never used, whatever the environment says.
func TestStubTranscriberNeedsDriveMode(t *testing.T) {
	t.Setenv("CHIRON_DRIVE", "")
	t.Setenv("CHIRON_TRANSCRIBE", "stub")
	s := newServer(t, "")
	unit, items := walkToSeries(t, s, "ai")
	body := `{"unit":"` + unit + `","items":[{"item_id":"` + items[0] + `","strokes":[[{"x":0.1,"y":0.2},{"x":0.5,"y":0.6}]],"aspect":3}]}`
	if w := do(t, s, "POST", "/ink/ai", body, ""); w.Code == http.StatusOK {
		t.Fatalf("ink transcribed without a model or drive mode: %s", w.Body.String())
	}
}
