package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mjbraun/chiron/server/roles"
	"github.com/mjbraun/chiron/server/state"
)

// The reader highlights a passage and asks about it. The answer comes back
// in the same request (a short model call), and the question is recorded in
// the learner state so the planner sees what confused them.

type askRequest struct {
	Unit     string `json:"unit"`
	Quote    string `json:"quote"`
	Question string `json:"question"`
}

func (s *Server) handleAsk(w http.ResponseWriter, r *http.Request) {
	sub, ok := s.subject(r.PathValue("subject"))
	if !ok {
		http.Error(w, "unknown subject", http.StatusNotFound)
		return
	}
	var req askRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "malformed request: %v", err)
		return
	}
	req.Question = strings.TrimSpace(req.Question)
	if req.Question == "" {
		writeError(w, http.StatusUnprocessableEntity, "question is required")
		return
	}
	if req.Unit == "" {
		if cur := sub.Learner.Data.CurrentUnit; cur != nil {
			req.Unit = *cur
		}
	}
	unit, ok := sub.Corpus.Units[req.Unit]
	if !ok {
		writeError(w, http.StatusNotFound, "unknown unit: %q", req.Unit)
		return
	}
	s.markActive(sub.ID)

	answer, err := roles.AnswerQuestion(s.chain, unit, req.Quote, req.Question, sub.Learner)
	if err != nil && driveEnabled() && !s.chain.Status().Connected {
		// A dev server without a model still lets a client exercise the
		// whole ask flow; the stub names itself.
		answer, err = fmt.Sprintf("[stub answer about %q: %s]", req.Quote, req.Question), nil
	}
	if err != nil {
		writeError(w, http.StatusBadGateway, "no answer: %v", err)
		return
	}
	ev, err := sub.Learner.Apply(state.Event{
		Kind: "asked", Unit: unit.ID, Text: req.Question, Evidence: req.Quote, Why: answer})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "record question: %v", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"unit":      unit.ID,
		"answer_md": answer,
		"n":         ev.N,
	})
}
