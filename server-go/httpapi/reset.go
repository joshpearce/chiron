package httpapi

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
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
	// A restart never destroys the old book: everything the run produced
	// moves into a timestamped archive beside the fresh state.
	stamp := time.Now().Format("20060102-150405")
	arch := filepath.Join(sub.StateDir, "archive", stamp)
	if err := os.MkdirAll(arch, 0o755); err != nil {
		writeError(w, http.StatusInternalServerError, "archive failed")
		return
	}
	for _, name := range []string{"chapters", "results", "ink", "learner.json", "events.jsonl"} {
		src := filepath.Join(sub.StateDir, name)
		if _, err := os.Stat(src); err == nil {
			if err := os.Rename(src, filepath.Join(arch, name)); err != nil {
				log.Printf("reset archive %s: %v", name, err)
			}
		}
	}
	log.Printf("reset %s: archived to %s", req.Subject, arch)
	if err := sub.Learner.Reset(); err != nil {
		log.Printf("reset %s: %v", req.Subject, err)
		writeError(w, http.StatusInternalServerError, "reset failed")
		return
	}
	log.Printf("reset %s", req.Subject)
	writeJSON(w, http.StatusOK, s.statePayload(sub))
}
