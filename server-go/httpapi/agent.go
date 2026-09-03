package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/mjbraun/chiron/server/agent"
)

// The sprite's agent drives the app: the app holds /agent/app open, and a
// program on the sprite (chiron-app) posts verbs to /agent/cmd. Both sit
// behind the shared key like everything else.

func (s *Server) handleAgentApp(w http.ResponseWriter, r *http.Request) {
	s.hub.ServeApp(w, r)
}

func (s *Server) handleAgentStatus(w http.ResponseWriter, r *http.Request) {
	ok, name := s.hub.Connected()
	writeJSON(w, http.StatusOK, map[string]any{"connected": ok, "app": name})
}

func (s *Server) handleAgentCmd(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Verb string          `json:"verb"`
		Args json.RawMessage `json:"args"`
		// Seconds to wait; a screenshot or a graded check takes a while.
		Timeout float64 `json:"timeout"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "malformed request: %v", err)
		return
	}
	req.Verb = strings.TrimSpace(req.Verb)
	if req.Verb == "" {
		writeError(w, http.StatusUnprocessableEntity, "verb is required")
		return
	}
	timeout := 60 * time.Second
	if req.Timeout > 0 {
		timeout = time.Duration(req.Timeout * float64(time.Second))
	}
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()
	res, err := s.hub.Command(ctx, req.Verb, req.Args)
	switch {
	case errors.Is(err, agent.ErrNoApp):
		writeError(w, http.StatusServiceUnavailable, "no app is connected; turn on the agent in the app's server settings")
	case errors.Is(err, context.DeadlineExceeded):
		writeError(w, http.StatusGatewayTimeout, "the app did not answer %q within %s", req.Verb, timeout)
	case err != nil:
		writeError(w, http.StatusBadGateway, "%v", err)
	default:
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if len(res) == 0 {
			res = json.RawMessage("null")
		}
		w.Write(res)
	}
}
