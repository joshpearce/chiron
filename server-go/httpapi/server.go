// Package httpapi is the transport. Chiron is a container of subjects; each has
// its own corpus and its own learner state.
//
// One transport-agnostic exchange endpoint per interaction: the app reads fully
// detached, and at a chapter boundary POSTs everything that happened (beat
// responses, check answers, confidence ratings, timings, override and catch-up
// requests) and receives grades, the gate result, the next chapter and updated
// state - over Wi-Fi directly, or pushed through iproxy on the USB path.
package httpapi

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/mjbraun/chiron/server/corpus"
	"github.com/mjbraun/chiron/server/llm"
	"github.com/mjbraun/chiron/server/pages"
	"github.com/mjbraun/chiron/server/state"
)

type SubjectSpec struct {
	ID        string `yaml:"id"`
	Title     string `yaml:"title"`
	CorpusDir string `yaml:"corpus_dir"`
	StateDir  string `yaml:"state_dir"`
}

type SessionConfig struct {
	MasteryGate          float64 `yaml:"mastery_gate"`
	ExtensionTrigger     float64 `yaml:"extension_trigger"`
	CheckItems           int     `yaml:"check_items"`
	CallbackFraction     float64 `yaml:"callback_fraction"`
	ChunkMinutes         float64 `yaml:"chunk_minutes"`
	BreakMinutes         float64 `yaml:"break_minutes"`
	LongBreakEveryChunks float64 `yaml:"long_break_every_chunks"`
}

type Config struct {
	Subjects  []SubjectSpec `yaml:"subjects"`
	StaticDir string        `yaml:"static_dir"`
	// KatexDir points at the KaTeX assets used when rendering chapters to
	// page images for e-ink clients (the same files the iPad bundles).
	KatexDir string `yaml:"katex_dir"`
	// FontsDir holds the bundled page faces (Source Serif 4, Source Sans 3)
	// referenced by the page stylesheet.
	FontsDir string `yaml:"fonts_dir"`
	// VisionModel transcribes handwritten ink submissions (loaded on demand
	// by the same OpenAI-compatible server as the text upstream).
	VisionModel string        `yaml:"vision_model"`
	Session     SessionConfig `yaml:"session"`
	AuthToken   string        `yaml:"auth_token"`
	// Grade all free-text items of a check in one model call. Off by default:
	// the per-item path is the one verified end to end, and a check is the
	// moment a learner is most exposed to a regression.
	BatchGrading bool `yaml:"batch_grading"`

	Provider       string         `yaml:"provider"`
	AnthropicModel string         `yaml:"anthropic_model"`
	ClaudeCLIModel string         `yaml:"claude_cli_model"`
	Upstreams      []llm.Upstream `yaml:"upstreams"`
	LLM            llm.Config     `yaml:"llm"`
}

func LoadConfig(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Config
	if err := yaml.Unmarshal(raw, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

// Subject pairs a corpus with the learner state for it.
type Subject struct {
	ID       string
	Title    string
	Corpus   *corpus.Corpus
	Learner  *state.Learner
	StateDir string
	Pages    *pages.Renderer

	// Background authoring state: one build at a time per subject, with the
	// failure kept for the pages meta to surface.
	buildMu  sync.Mutex
	building bool
	buildErr string
}

func (sub *Subject) beginBuild() bool {
	sub.buildMu.Lock()
	defer sub.buildMu.Unlock()
	if sub.building {
		return false
	}
	sub.building = true
	sub.buildErr = ""
	return true
}

func (sub *Subject) endBuild(err string) {
	sub.buildMu.Lock()
	sub.building = false
	sub.buildErr = err
	sub.buildMu.Unlock()
}

func (sub *Subject) buildStatus() (bool, string) {
	sub.buildMu.Lock()
	defer sub.buildMu.Unlock()
	return sub.building, sub.buildErr
}

type Server struct {
	cfg  *Config
	root string // directory config.yaml lives in
	// transcribe overrides the ink vision transcriber; tests inject one.
	transcribe func(hint string, png []byte) (string, error)
	chain      llm.Chain
	token      string
	rng        *rand.Rand
	rngMu      sync.Mutex

	mu       sync.RWMutex
	subjects map[string]*Subject

	jobsMu sync.Mutex
	jobs   map[string]*Job
}

func New(cfg *Config, root string) (*Server, error) {
	// CHIRON_AUTH_TOKEN is preferred over the config key so the secret lives in
	// the service environment rather than in a file on disk.
	token := strings.TrimSpace(os.Getenv("CHIRON_AUTH_TOKEN"))
	if token == "" {
		token = strings.TrimSpace(cfg.AuthToken)
	}
	s := &Server{
		cfg:      cfg,
		root:     root,
		token:    token,
		rng:      rand.New(rand.NewSource(time.Now().UnixNano())),
		subjects: map[string]*Subject{},
		jobs:     map[string]*Job{},
		chain: llm.New(llm.FactoryConfig{
			Provider: cfg.Provider, AnthropicModel: cfg.AnthropicModel,
			ClaudeCLIModel: cfg.ClaudeCLIModel, Upstreams: cfg.Upstreams, LLM: cfg.LLM,
		}),
	}
	for _, spec := range cfg.Subjects {
		if err := s.register(spec.ID, spec.Title,
			resolve(root, spec.CorpusDir), resolve(root, spec.StateDir)); err != nil {
			return nil, fmt.Errorf("subject %s: %w", spec.ID, err)
		}
	}
	s.Discover()
	return s, nil
}

// resolve interprets a config path relative to the config file, leaving an
// absolute path alone. filepath.Join would otherwise glue the two together,
// which pathlib in the Python implementation does not do.
func resolve(root, p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(root, p)
}

func (s *Server) register(id, title, corpusDir, stateDir string) error {
	c, err := corpus.Load(corpusDir)
	if err != nil {
		return err
	}
	l, err := state.Open(stateDir, c)
	if err != nil {
		return err
	}
	fonts := ""
	if s.cfg.FontsDir != "" {
		fonts = resolve(s.root, s.cfg.FontsDir)
	}
	s.mu.Lock()
	s.subjects[id] = &Subject{ID: id, Title: title, Corpus: c, Learner: l,
		StateDir: stateDir,
		Pages: &pages.Renderer{
			KatexDir: resolve(s.root, s.cfg.KatexDir),
			FontsDir: fonts,
			CacheDir: filepath.Join(stateDir, "pages"),
		}}
	s.mu.Unlock()
	return nil
}

// Discover registers every corpus-<slug>/ sibling directory not already
// configured.
//
// A subject the learner asked for through Teach-me has to survive a restart,
// and the alternative - rewriting config.yaml - would strip the comments that
// explain the flight settings. The corpus directory on disk is the record.
//
// Only fully-authored corpora register. A syllabus exists from the moment
// planning finishes, so a half-generated subject would otherwise appear in the
// library as a book whose chapters are missing.
func (s *Server) Discover() []string {
	parent := filepath.Dir(s.root)
	entries, err := filepath.Glob(filepath.Join(parent, "corpus-*"))
	if err != nil {
		return nil
	}
	sort.Strings(entries)

	s.mu.RLock()
	configured := map[string]bool{}
	for _, sub := range s.subjects {
		if abs, err := filepath.Abs(sub.Corpus.Dir); err == nil {
			configured[abs] = true
		}
	}
	s.mu.RUnlock()

	var found []string
	for _, dir := range entries {
		info, err := os.Stat(dir)
		if err != nil || !info.IsDir() {
			continue
		}
		if _, err := os.Stat(filepath.Join(dir, "syllabus.yaml")); err != nil {
			continue
		}
		abs, err := filepath.Abs(dir)
		if err != nil || configured[abs] {
			continue
		}
		c, err := corpus.Load(dir)
		if err != nil {
			continue // a malformed corpus must not stop boot
		}
		if len(c.Units) < len(c.Syllabus.Units) {
			continue
		}
		id := strings.TrimPrefix(filepath.Base(dir), "corpus-")
		title := c.Syllabus.Title
		if title == "" {
			title = strings.Title(strings.ReplaceAll(id, "-", " ")) //nolint:staticcheck
		}
		if err := s.register(id, title, dir, filepath.Join(parent, "state", id)); err != nil {
			continue
		}
		found = append(found, id)
	}
	return found
}

func (s *Server) subject(id string) (*Subject, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sub, ok := s.subjects[id]
	return sub, ok
}

func (s *Server) allSubjects() []*Subject {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Subject, 0, len(s.subjects))
	for _, sub := range s.subjects {
		out = append(out, sub)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// ---------- transport ----------

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /ping", s.handlePing)
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.HandleFunc("GET /subjects", s.handleSubjects)
	mux.HandleFunc("GET /state", s.handleState)
	mux.HandleFunc("GET /review-schedule", s.handleReviewSchedule)
	mux.HandleFunc("GET /pages/{subject}", s.handlePagesMeta)
	mux.HandleFunc("GET /pages/{subject}/{page}", s.handlePage)
	mux.HandleFunc("GET /pages/{subject}/results", s.handleResultsMeta)
	mux.HandleFunc("GET /pages/{subject}/results/{page}", s.handleResultsPage)
	mux.HandleFunc("POST /ink/{subject}", s.handleInk)
	mux.HandleFunc("POST /drive/cmd", s.handleDriveCmd)
	mux.HandleFunc("GET /drive/next", s.handleDriveNext)
	mux.HandleFunc("POST /drive/ack", s.handleDriveAck)
	mux.HandleFunc("GET /drive/ack", s.handleDriveAck)
	mux.HandleFunc("POST /exchange", s.handleExchange)
	mux.HandleFunc("POST /reset", s.handleReset)
	mux.HandleFunc("POST /teach/turn", s.handleTeachTurn)
	mux.HandleFunc("POST /teach/create", s.handleTeachCreate)
	mux.HandleFunc("GET /teach/jobs", s.handleTeachJobs)
	return s.requireToken(mux)
}

// requireToken guards everything except /ping.
//
// An empty token leaves the server open, which is correct on the flight LAN
// where the only client is the iPad on a Mac-hosted network. It is not optional
// on a public URL: without it, anyone who finds the endpoint spends the tutor's
// model budget.
func (s *Server) requireToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.token != "" && r.URL.Path != "/ping" {
			supplied := r.Header.Get("Authorization")
			expected := "Bearer " + s.token
			// Constant-time compare: a length- or prefix-leaking check on a
			// shared secret is a bad habit even on a small deployment.
			if subtle.ConstantTimeCompare([]byte(supplied), []byte(expected)) != 1 {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"detail": "unauthorized"})
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("write response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, format string, a ...any) {
	writeJSON(w, status, map[string]string{"detail": fmt.Sprintf(format, a...)})
}

// handlePing is an unauthenticated liveness probe that deliberately reveals
// nothing.
func (s *Server) handlePing(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	subjects := map[string][]string{}
	for _, sub := range s.allSubjects() {
		var units []string
		for _, uid := range sub.Corpus.UnitOrder() {
			if _, ok := sub.Corpus.Units[uid]; ok {
				units = append(units, uid)
			}
		}
		subjects[sub.ID] = units
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok": true, "llm": s.chain.Status(), "subjects": subjects,
	})
}

func (s *Server) handleSubjects(w http.ResponseWriter, _ *http.Request) {
	type row struct {
		ID           string  `json:"id"`
		Title        string  `json:"title"`
		UnitsTotal   int     `json:"units_total"`
		UnitsCleared int     `json:"units_cleared"`
		CurrentUnit  *string `json:"current_unit"`
		Debt         int     `json:"debt"`
	}
	out := []row{}
	for _, sub := range s.allSubjects() {
		out = append(out, row{
			ID: sub.ID, Title: sub.Title,
			UnitsTotal:   len(sub.Corpus.UnitOrder()),
			UnitsCleared: len(sub.Learner.ClearedUnits()),
			CurrentUnit:  sub.Learner.Snapshot().CurrentUnit,
			Debt:         len(sub.Learner.OpenDebt()),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"subjects": out})
}

func (s *Server) subjectParam(r *http.Request) string {
	if id := r.URL.Query().Get("subject"); id != "" {
		return id
	}
	return "ai"
}
