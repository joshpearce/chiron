package httpapi

import (
	"encoding/json"
	"log"
	"net/http"
)

type resetRequest struct {
	Subject string `json:"subject"`
	// Confirm has to be sent explicitly. This throws away a learner's whole
	// history, and it sits on the same authenticated surface as everything
	// else, so it should not be reachable by a malformed or replayed request.
	Confirm bool `json:"confirm"`
}

// handleReset starts a subject over.
func (s *Server) handleReset(w http.ResponseWriter, r *http.Request) {
	var req resetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "malformed request: %v", err)
		return
	}
	if req.Subject == "" {
		req.Subject = "ai"
	}
	sub, ok := s.subject(req.Subject)
	if !ok {
		writeError(w, http.StatusNotFound, "unknown subject: %s", req.Subject)
		return
	}
	if !req.Confirm {
		writeError(w, http.StatusUnprocessableEntity,
			"reset discards all progress for %q; send confirm: true", req.Subject)
		return
	}
	if err := sub.Learner.Reset(); err != nil {
		log.Printf("reset %s: %v", req.Subject, err)
		writeError(w, http.StatusInternalServerError, "reset failed")
		return
	}
	log.Printf("reset %s", req.Subject)
	writeJSON(w, http.StatusOK, s.statePayload(sub))
}
