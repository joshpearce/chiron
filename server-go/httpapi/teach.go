package httpapi

import (
	"encoding/json"
	"net/http"
	"regexp"
	"sort"
	"strings"

	"github.com/mjbraun/chiron/server/roles"
)

// The learner describes what they want to learn, the tutor asks until it can
// write a brief, and generation runs from that brief. Generation takes minutes
// per unit, so it runs on a worker goroutine and the client polls; the subject
// appears in /subjects once every unit is authored.

var slugOK = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,31}$`)

// Job is the progress of one subject generation.
type Job struct {
	Slug       string `json:"slug"`
	Title      string `json:"title"`
	Stage      string `json:"stage"` // queued | planning | authoring | ready | failed
	UnitsTotal int    `json:"units_total"`
	UnitsDone  int    `json:"units_done"`
	Done       bool   `json:"done"`
	Error      string `json:"error,omitempty"`
}

type teachTurnRequest struct {
	Messages []roles.Message `json:"messages"`
}

// handleTeachTurn runs one elicitation turn. The client keeps the transcript
// and sends it whole, so the server holds no conversation state and a dropped
// connection costs a retry rather than the thread.
func (s *Server) handleTeachTurn(w http.ResponseWriter, r *http.Request) {
	var req teachTurnRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "malformed request: %v", err)
		return
	}
	if len(req.Messages) == 0 {
		writeError(w, http.StatusUnprocessableEntity, "no messages")
		return
	}
	writeJSON(w, http.StatusOK, roles.ElicitTurn(s.chain, req.Messages))
}

type teachCreateRequest struct {
	Slug  string `json:"slug"`
	Title string `json:"title"`
	Brief string `json:"brief"`
	// Sources the book is built from, by index id or title; the first is
	// the spine. Without them the brief's own "starting from X" counts.
	Sources []string `json:"sources,omitempty"`
}

// slugify normalises a model-proposed slug into a directory-safe id.
//
// The slug names a directory, so it has to be constrained - but rejecting the
// model's formatting (underscores, capitals, trailing punctuation) would fail
// the request over something with an obvious right answer. Only what normalises
// to nothing is rejected.
func slugify(raw string) string {
	var b strings.Builder
	lastDash := true
	for _, r := range strings.ToLower(strings.TrimSpace(raw)) {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			lastDash = false
		case !lastDash:
			b.WriteByte('-')
			lastDash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if len(out) > 32 {
		out = strings.Trim(out[:32], "-")
	}
	return out
}

func (s *Server) handleTeachCreate(w http.ResponseWriter, r *http.Request) {
	var req teachCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "malformed request: %v", err)
		return
	}
	slug := slugify(req.Slug)
	if !slugOK.MatchString(slug) {
		writeError(w, http.StatusUnprocessableEntity, "slug must contain letters or digits")
		return
	}
	if _, exists := s.subject(slug); exists {
		writeError(w, http.StatusConflict, "subject '%s' already exists", slug)
		return
	}

	s.jobsMu.Lock()
	if existing, ok := s.jobs[slug]; ok && !existing.Done {
		s.jobsMu.Unlock()
		writeError(w, http.StatusConflict, "'%s' is already being generated", slug)
		return
	}
	job := &Job{Slug: slug, Title: req.Title, Stage: "queued"}
	s.jobs[slug] = job
	s.jobsMu.Unlock()

	s.startGenerate(slug, req.Title, req.Brief, req.Sources)
	writeJSON(w, http.StatusOK, job)
}

func (s *Server) handleTeachJobs(w http.ResponseWriter, r *http.Request) {
	s.jobsMu.Lock()
	defer s.jobsMu.Unlock()
	if slug := r.URL.Query().Get("slug"); slug != "" {
		job, ok := s.jobs[slug]
		if !ok {
			writeError(w, http.StatusNotFound, "no generation job for '%s'", slug)
			return
		}
		writeJSON(w, http.StatusOK, *job)
		return
	}
	out := []Job{}
	for _, j := range s.jobs {
		out = append(out, *j)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Slug < out[j].Slug })
	writeJSON(w, http.StatusOK, map[string]any{"jobs": out})
}

func (s *Server) updateJob(slug string, fn func(*Job)) {
	s.jobsMu.Lock()
	defer s.jobsMu.Unlock()
	if j, ok := s.jobs[slug]; ok {
		fn(j)
	}
}
