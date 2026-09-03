package httpapi

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/mjbraun/chiron/server/corpus"
	"github.com/mjbraun/chiron/server/primer"
	"github.com/mjbraun/chiron/server/render"
	"github.com/mjbraun/chiron/server/roles"
)

// Primers: the reader captures something anywhere on the iPad, asks a
// question, and a document appears on the shelf. Each is a one-unit
// subject with no check; margin notes extend it.

const KindPrimer = "primer"

var primersMu sync.Mutex

func (s *Server) primersRoot() string {
	if s.cfg.PrimersDir != "" {
		return resolve(s.root, s.cfg.PrimersDir)
	}
	return filepath.Join(filepath.Dir(s.root), "state", "primers")
}

// loadPrimers registers the ready ones at boot. One caught mid-authoring
// by a restart is marked failed: the reader sees why the card is grey.
func (s *Server) loadPrimers() {
	metas, err := primer.LoadAll(s.primersRoot())
	if err != nil {
		log.Printf("primers: %v", err)
		return
	}
	for _, m := range metas {
		if m.Status == primer.StatusAuthoring {
			m.Status, m.Error = primer.StatusFailed, "authoring was interrupted by a restart"
			primer.Save(s.primersRoot(), m)
		}
		s.primersMu.Lock()
		s.primers[m.ID] = m
		s.primersMu.Unlock()
		if m.Status == primer.StatusReady {
			if err := s.registerPrimer(m); err != nil {
				log.Printf("primer %s: %v", m.ID, err)
			}
		}
	}
}

func (s *Server) registerPrimer(m *primer.Meta) error {
	root := s.primersRoot()
	if err := s.register(m.ID, m.Title, primer.CorpusDir(root, m.ID), primer.StateDir(root, m.ID)); err != nil {
		return err
	}
	s.mu.Lock()
	if sub := s.subjects[m.ID]; sub != nil {
		sub.Kind = KindPrimer
		sub.Primer = m
	}
	s.mu.Unlock()
	return nil
}

type captureRequest struct {
	Text      string `json:"text"`
	ImagePNG  string `json:"image_png_b64"`
	SourceURL string `json:"source_url"`
	SourceApp string `json:"source_app"`
	Prompt    string `json:"prompt"`
	Title     string `json:"title"`
}

func (s *Server) handlePrimerCapture(w http.ResponseWriter, r *http.Request) {
	var req captureRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "malformed request: %v", err)
		return
	}
	req.Prompt = strings.TrimSpace(req.Prompt)
	req.Text = strings.TrimSpace(req.Text)
	if req.Prompt == "" {
		writeError(w, http.StatusUnprocessableEntity, "prompt is required: what do you want to know about this?")
		return
	}
	var png []byte
	if req.ImagePNG != "" {
		var err error
		if png, err = base64.StdEncoding.DecodeString(req.ImagePNG); err != nil {
			writeError(w, http.StatusBadRequest, "image_png_b64 is not base64: %v", err)
			return
		}
	}
	if req.Text == "" && len(png) == 0 {
		writeError(w, http.StatusUnprocessableEntity, "nothing captured: send text or image_png_b64")
		return
	}

	seed := req.Title
	if seed == "" {
		seed = req.Prompt
	}
	id := s.uniquePrimerID(seed)
	m := &primer.Meta{
		ID: id, Title: strings.TrimSpace(req.Title), Prompt: req.Prompt,
		Source:     primer.Source{Text: req.Text, URL: req.SourceURL, App: req.SourceApp, HasImage: len(png) > 0},
		Status:     primer.StatusAuthoring,
		CapturedAt: time.Now().UTC(),
	}
	if m.Title == "" {
		m.Title = workingTitle(req.Prompt)
	}
	if err := primer.Save(s.primersRoot(), m); err != nil {
		writeError(w, http.StatusInternalServerError, "save primer: %v", err)
		return
	}
	s.primersMu.Lock()
	s.primers[id] = m
	s.primersMu.Unlock()

	s.renders.Add(1)
	go func() {
		defer s.renders.Done()
		s.authorPrimer(m, req.Text, png)
	}()
	writeJSON(w, http.StatusOK, map[string]any{"subject": id, "status": m.Status, "title": m.Title})
}

// uniquePrimerID: "primer-" plus a slug of the seed, with a counter when
// the reader captures the same thing twice.
func (s *Server) uniquePrimerID(seed string) string {
	// Whole words up to the slug's length, so an id reads as a phrase.
	slug := ""
	for _, w := range strings.Fields(seed) {
		word := slugify(w)
		if word == "" {
			continue
		}
		next := strings.TrimPrefix(slug+"-"+word, "-")
		if len(next) > 32 {
			break
		}
		slug = next
	}
	if slug == "" {
		slug = "capture"
	}
	base := "primer-" + slug
	id := base
	for n := 2; ; n++ {
		_, taken := s.subject(id)
		s.primersMu.Lock()
		_, pending := s.primers[id]
		s.primersMu.Unlock()
		if !taken && !pending {
			return id
		}
		id = fmt.Sprintf("%s-%d", base, n)
	}
}

// workingTitle names a primer until the author does: the question, whole
// if it is short, otherwise cut at a word.
func workingTitle(prompt string) string {
	t := strings.Join(strings.Fields(prompt), " ")
	if t == "" {
		return "Capture"
	}
	if len(t) > 72 {
		cut := strings.LastIndex(t[:72], " ")
		if cut < 24 {
			cut = 72
		}
		t = t[:cut]
	}
	return strings.ToUpper(t[:1]) + t[1:]
}

func (s *Server) authorPrimer(m *primer.Meta, text string, png []byte) {
	root := s.primersRoot()
	fail := func(err error) {
		m.Status, m.Error = primer.StatusFailed, err.Error()
		primer.Save(root, m)
		log.Printf("primer %s: %v", m.ID, err)
	}
	if text == "" && len(png) > 0 {
		// The same vision path the ink check-in uses: the image becomes
		// words, and the primer is written from the words.
		tr := s.transcribe
		if tr == nil {
			tr = s.transcriberFor()
		}
		t, err := tr("captured image", png)
		if err != nil {
			fail(fmt.Errorf("transcribe the image: %w", err))
			return
		}
		text = t
		m.Source.Text = t
	}
	cap := roles.Capture{Text: text, URL: m.Source.URL, App: m.Source.App, Prompt: m.Prompt}
	title, doc, err := roles.AuthorPrimer(s.chain, cap)
	if err != nil && driveEnabled() && !s.chain.Status().Connected {
		title, doc, err = stubPrimer(cap)
	}
	if err != nil {
		fail(err)
		return
	}
	m.Title = title
	if err := primer.WriteDoc(root, m.ID, m.Title, doc); err != nil {
		fail(err)
		return
	}
	if err := s.registerPrimer(m); err != nil {
		fail(err)
		return
	}
	m.Status, m.Error = primer.StatusReady, ""
	if err := primer.Save(root, m); err != nil {
		log.Printf("primer %s: save: %v", m.ID, err)
	}
}

// stubPrimer stands in for the model on a dev server: the capture and the
// question come back as a document that names itself a stub.
func stubPrimer(cap roles.Capture) (string, string, error) {
	title := workingTitle(cap.Prompt)
	doc := fmt.Sprintf("[stub primer] You asked: %s\n\n## What you captured\n\n%s\n\n## The answer\n\nA dev server without a model cannot write this; a real one would answer the question from the captured material.\n",
		cap.Prompt, clipText(cap.Text, 2000))
	return title, doc, nil
}

func clipText(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + " ..."
}

type extendRequest struct {
	Quote string `json:"quote"`
	Note  string `json:"note"`
}

func (s *Server) handlePrimerExtend(w http.ResponseWriter, r *http.Request) {
	sub, ok := s.subject(r.PathValue("subject"))
	if !ok || sub.Kind != KindPrimer || sub.Primer == nil {
		writeError(w, http.StatusNotFound, "not a primer: %q", r.PathValue("subject"))
		return
	}
	var req extendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "malformed request: %v", err)
		return
	}
	req.Note = strings.TrimSpace(req.Note)
	if req.Note == "" {
		writeError(w, http.StatusUnprocessableEntity, "note is required")
		return
	}
	s.markActive(sub.ID)
	root := s.primersRoot()
	m := sub.Primer

	primersMu.Lock()
	defer primersMu.Unlock()
	doc, err := primer.ReadDoc(root, m.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "read primer: %v", err)
		return
	}
	heading, body, err := roles.ExtendPrimer(s.chain, doc, req.Quote, req.Note)
	if err != nil && driveEnabled() && !s.chain.Status().Connected {
		heading, body, err = "[stub] "+workingTitle(req.Note), fmt.Sprintf("On \"%s\" you noted: %s\n\nA dev server without a model cannot write this section.", clipText(req.Quote, 200), req.Note), nil
	}
	if err != nil {
		writeError(w, http.StatusBadGateway, "no section: %v", err)
		return
	}
	if _, err := primer.AppendSection(root, m.ID, m.Title, heading, body); err != nil {
		writeError(w, http.StatusInternalServerError, "extend primer: %v", err)
		return
	}
	c, err := corpus.Load(primer.CorpusDir(root, m.ID))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "reload primer: %v", err)
		return
	}
	s.mu.Lock()
	sub.Corpus = c
	s.mu.Unlock()
	os.Remove(chapterPath(sub, primer.UnitID))

	ch, err := s.buildChapter(sub, primer.UnitID, "")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "rebuild primer: %v", err)
		return
	}
	if err := persistChapter(sub, ch); err != nil {
		log.Printf("persist primer %s: %v", m.ID, err)
	}
	m.Entries = append(m.Entries, primer.Entry{Quote: req.Quote, Note: req.Note, Heading: heading, At: time.Now().UTC()})
	if err := primer.Save(root, m); err != nil {
		log.Printf("primer %s: save: %v", m.ID, err)
	}
	writeJSON(w, http.StatusOK, map[string]any{"chapter": ch, "heading": heading, "entries": len(m.Entries)})
}

// buildPrimerChapter is the primer's whole reading contract: the document
// as written, no planner, no pretest, no check.
func (s *Server) buildPrimerChapter(sub *Subject, unitID string) (*render.Chapter, error) {
	unit, ok := sub.Corpus.Units[unitID]
	if !ok {
		return nil, fmt.Errorf("unit %s not authored", unitID)
	}
	ch, err := render.RenderChapter(unit, roles.VerbatimSections(unit), render.Directives{NextAction: "read"}, nil, nil)
	if err != nil {
		return nil, err
	}
	s.markUnitStarted(sub, unit, "")
	return ch, nil
}

// primerRows lists primers still authoring or failed, which are not
// subjects yet but belong on the shelf.
func (s *Server) pendingPrimers() []*primer.Meta {
	s.primersMu.Lock()
	defer s.primersMu.Unlock()
	var out []*primer.Meta
	for _, m := range s.primers {
		if m.Status != primer.StatusReady {
			out = append(out, m)
		}
	}
	return out
}
