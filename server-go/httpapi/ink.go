package httpapi

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/mjbraun/chiron/server/ink"
	"github.com/mjbraun/chiron/server/llm"
	"github.com/mjbraun/chiron/server/pages"
)

// The paper check-in. The e-ink client sends the strokes it captured per
// item; the server rasterizes each item's ink, has a vision model transcribe
// the handwriting, and feeds the transcriptions through the same boundary
// exchange the iPad uses. An item with no ink is an explicit "I don't know".

type InkItem struct {
	ItemID  string        `json:"item_id"`
	Strokes [][]ink.Point `json:"strokes"`
	// SelectedIndex answers structured items (MCQ options, the screener
	// rating) from the client's own controls - no ink, no transcription.
	SelectedIndex *int `json:"selected_index,omitempty"`
	// IDK is the client-side "I don't know" toggle. It wins over any ink on
	// the page - a tick mark is not an answer.
	IDK bool `json:"idk,omitempty"`
	// Confidence is optional; paper has no slider, so absent means "shaky".
	Confidence int `json:"confidence,omitempty"`
	// Aspect is the width/height ratio of the region the strokes were
	// captured in; the raster canvas matches it so handwriting is not
	// distorted for the transcriber.
	Aspect float64 `json:"aspect,omitempty"`
}

type InkSubmission struct {
	Unit         string    `json:"unit"`
	Items        []InkItem `json:"items"`
	ChunkMinutes float64   `json:"chunk_minutes,omitempty"`
	// BreakMinutes reports a break actually taken since the last exchange,
	// so the fatigue model sees rest, not absence.
	BreakMinutes float64 `json:"break_minutes,omitempty"`
}

// transcriber is swappable for tests.
type transcriber func(hint string, png []byte) (string, error)

func (s *Server) transcriberFor() transcriber {
	base := ""
	for _, up := range s.cfg.Upstreams {
		if up.Enabled {
			base = up.BaseURL
			break
		}
	}
	model := s.cfg.VisionModel
	if model == "" {
		model = "google/gemma-3-4b"
	}
	return func(hint string, png []byte) (string, error) {
		return llm.Transcribe(base, model, hint, png)
	}
}

func (s *Server) handleInk(w http.ResponseWriter, r *http.Request) {
	sub, ok := s.subject(r.PathValue("subject"))
	if !ok {
		http.Error(w, "unknown subject", http.StatusNotFound)
		return
	}
	var in InkSubmission
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "malformed request: %v", err)
		return
	}
	if in.Unit == "" {
		writeError(w, http.StatusBadRequest, "unit is required")
		return
	}

	transcribe := s.transcribe
	if transcribe == nil {
		transcribe = s.transcriberFor()
	}

	// Keep the graded artifacts: the rasterized answer and its transcription
	// land next to the learner state, so a surprising grade can be audited.
	inkDir := filepath.Join(sub.StateDir, "ink", in.Unit)
	_ = os.MkdirAll(inkDir, 0o755)

	responses := make([]ItemResponse, 0, len(in.Items))
	transcripts := map[string]string{}
	for _, item := range in.Items {
		if item.IDK {
			responses = append(responses, ItemResponse{
				ItemID: item.ItemID, IDK: true, Confidence: 1})
			transcripts[item.ItemID] = "[I don't know]"
			continue
		}
		conf := item.Confidence
		if conf == 0 {
			conf = 2
		}
		if item.SelectedIndex != nil {
			responses = append(responses, ItemResponse{
				ItemID: item.ItemID, SelectedIndex: item.SelectedIndex,
				Confidence: conf})
			transcripts[item.ItemID] = fmt.Sprintf("[option %d]", *item.SelectedIndex+1)
			continue
		}
		rw, rh := pages.PageW/2, pages.PageH/2
		if item.Aspect > 0.1 && item.Aspect < 20 {
			rh = int(float64(rw) / item.Aspect)
		}
		png, err := ink.Rasterize(item.Strokes, rw, rh)
		if err != nil {
			// No ink on the page: an explicit pass on the question.
			responses = append(responses, ItemResponse{
				ItemID: item.ItemID, IDK: true, Confidence: 1})
			transcripts[item.ItemID] = "[no answer]"
			continue
		}
		_ = os.WriteFile(filepath.Join(inkDir, item.ItemID+".png"), png, 0o644)

		text, err := transcribe(item.ItemID, png)
		if err != nil {
			writeError(w, http.StatusBadGateway, "transcription failed on %s: %v",
				item.ItemID, err)
			return
		}
		text = strings.TrimSpace(text)
		transcripts[item.ItemID] = text
		if text == "" || strings.EqualFold(text, "[no answer]") {
			responses = append(responses, ItemResponse{
				ItemID: item.ItemID, IDK: true, Confidence: 1})
			continue
		}
		responses = append(responses, ItemResponse{
			ItemID: item.ItemID, Response: text, Confidence: conf})
	}
	if data, err := json.MarshalIndent(transcripts, "", " "); err == nil {
		_ = os.WriteFile(filepath.Join(inkDir, "transcripts.json"), data, 0o644)
	}
	log.Printf("ink check-in %s/%s: %d items", sub.ID, in.Unit, len(responses))

	out := s.processExchange(sub, Exchange{
		Subject:        sub.ID,
		Phase:          "boundary",
		Unit:           in.Unit,
		CheckResponses: responses,
		ChunkMinutes:   in.ChunkMinutes,
		BreakMinutes:   in.BreakMinutes,
	}, true)
	out["transcripts"] = transcripts
	writeJSON(w, http.StatusOK, out)
}

var _ = fmt.Sprintf
