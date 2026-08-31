package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/mjbraun/chiron/server/pages"
	"github.com/mjbraun/chiron/server/render"
)

// Chapters are delivered inside exchange responses and normally live only on
// the client. E-ink clients cannot render the HTML, so the server keeps the
// most recent chapter per unit on disk and serves it as page images instead.

func chapterPath(sub *Subject, unit string) string {
	return filepath.Join(sub.StateDir, "chapters", unit+".json")
}

// persistChapter records a delivered chapter so /pages can re-render it for
// image clients. Failure is logged into the error, not fatal: the exchange
// response (with the inline chapter) is still the priority.
func persistChapter(sub *Subject, ch *render.Chapter) error {
	if ch == nil {
		return nil
	}
	dir := filepath.Join(sub.StateDir, "chapters")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(ch)
	if err != nil {
		return err
	}
	tmp := chapterPath(sub, ch.Unit) + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, chapterPath(sub, ch.Unit))
}

func loadChapter(sub *Subject, unit string) (*render.Chapter, error) {
	data, err := os.ReadFile(chapterPath(sub, unit))
	if err != nil {
		return nil, err
	}
	var ch render.Chapter
	if err := json.Unmarshal(data, &ch); err != nil {
		return nil, err
	}
	return &ch, nil
}

// currentChapter returns the persisted chapter for the learner's active unit.
func currentChapter(sub *Subject) (*render.Chapter, error) {
	cur := sub.Learner.Data.CurrentUnit
	if cur == nil || *cur == "" {
		return nil, fmt.Errorf("no active unit")
	}
	return loadChapter(sub, *cur)
}

// requestChapter resolves the chapter a pages request is about: the active
// unit, or - with ?unit= - any persisted chapter, read-only, so the contents
// screen can open cleared chapters without touching learner state.
func requestChapter(sub *Subject, r *http.Request) (*render.Chapter, error) {
	if unit := r.URL.Query().Get("unit"); unit != "" {
		return loadChapter(sub, unit)
	}
	return currentChapter(sub)
}

func (s *Server) handlePagesMeta(w http.ResponseWriter, r *http.Request) {
	sub, ok := s.subject(r.PathValue("subject"))
	if !ok {
		http.Error(w, "unknown subject", http.StatusNotFound)
		return
	}
	// Every open of a book performs this fetch, so it is where "the book the
	// reader has open" is observed.
	s.markActive(sub.ID)
	ch, err := requestChapter(sub, r)
	if err != nil {
		if building, buildErr := sub.buildStatus(); building || buildErr != "" {
			writeJSON(w, http.StatusOK, map[string]any{
				"authoring":       building,
				"authoring_error": buildErr,
			})
			return
		}
		http.Error(w, "no chapter available: "+err.Error(), http.StatusNotFound)
		return
	}
	res, err := sub.Pages.Render(ch)
	if err != nil {
		http.Error(w, "render: "+err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"unit":          ch.Unit,
		"title":         ch.Title,
		"subject_title": sub.Title,
		"count":         res.Count,
		"hash":          res.Hash,
		"calibration":   ch.Calibration,
		"items":         pages.ItemPages(ch, res.Count),
		"screener":      pages.Screener(ch),
		"layout": map[string]int{
			"page_w": pages.PageW, "page_h": pages.PageH,
		},
	})
}

// The most recent graded check's typeset results, kept on disk like
// chapters so the pages survive a server restart mid-results.

func resultsPath(sub *Subject) string {
	return filepath.Join(sub.StateDir, "results", "current.json")
}

func persistResults(sub *Subject, doc *pages.ResultsDoc) error {
	if err := os.MkdirAll(filepath.Dir(resultsPath(sub)), 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(doc)
	if err != nil {
		return err
	}
	tmp := resultsPath(sub) + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, resultsPath(sub))
}

func loadResults(sub *Subject) (*pages.ResultsDoc, error) {
	data, err := os.ReadFile(resultsPath(sub))
	if err != nil {
		return nil, err
	}
	var doc pages.ResultsDoc
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	return &doc, nil
}

func (s *Server) handleResultsMeta(w http.ResponseWriter, r *http.Request) {
	sub, ok := s.subject(r.PathValue("subject"))
	if !ok {
		http.Error(w, "unknown subject", http.StatusNotFound)
		return
	}
	doc, err := loadResults(sub)
	if err != nil {
		http.Error(w, "no results available", http.StatusNotFound)
		return
	}
	res, err := sub.Pages.RenderResults(doc)
	if err != nil {
		http.Error(w, "render: "+err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"unit":        doc.Unit,
		"count":       res.Count,
		"hash":        res.Hash,
		"action":      doc.Action,
		"calibration": doc.Calibration,
		"layout": map[string]int{
			"page_w": pages.PageW, "page_h": pages.PageH,
		},
	})
}

func (s *Server) handleResultsPage(w http.ResponseWriter, r *http.Request) {
	sub, ok := s.subject(r.PathValue("subject"))
	if !ok {
		http.Error(w, "unknown subject", http.StatusNotFound)
		return
	}
	n, err := strconv.Atoi(r.PathValue("page"))
	if err != nil || n < 0 {
		http.Error(w, "bad page number", http.StatusBadRequest)
		return
	}
	doc, err := loadResults(sub)
	if err != nil {
		http.Error(w, "no results available", http.StatusNotFound)
		return
	}
	res, err := sub.Pages.RenderResults(doc)
	if err != nil {
		http.Error(w, "render: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if n >= res.Count {
		http.Error(w, "page out of range", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "max-age=86400")
	http.ServeFile(w, r, res.PagePath(n))
}

// buildContentsDoc assembles the spine: every non-calibration unit in
// syllabus order, with cleared/in-progress/unwritten state from the learner
// model and the persisted chapters.
func (s *Server) buildContentsDoc(sub *Subject) *pages.ContentsDoc {
	doc := &pages.ContentsDoc{Subject: sub.Title}
	cur := ""
	if sub.Learner.Data.CurrentUnit != nil {
		cur = *sub.Learner.Data.CurrentUnit
	}
	n := 0
	for _, uid := range sub.Corpus.UnitOrder() {
		unit, ok := sub.Corpus.Units[uid]
		if !ok || unit.IsCalibration() {
			continue
		}
		n++
		row := pages.ContentsRow{Unit: uid, N: n, Title: unit.Title, Current: uid == cur}
		us := sub.Learner.Data.Units[uid]
		_, chErr := loadChapter(sub, uid)
		switch {
		case us != nil && us.Status == "passed":
			row.State = "cleared"
			if us.CheckScore != nil {
				row.Score = int(*us.CheckScore*100 + 0.5)
			}
		case chErr == nil:
			row.State = "in_progress"
		default:
			row.State = "unwritten"
		}
		doc.Rows = append(doc.Rows, row)
	}
	return doc
}

func (s *Server) handleContentsMeta(w http.ResponseWriter, r *http.Request) {
	sub, ok := s.subject(r.PathValue("subject"))
	if !ok {
		http.Error(w, "unknown subject", http.StatusNotFound)
		return
	}
	doc := s.buildContentsDoc(sub)
	res, err := sub.Pages.RenderContents(doc)
	if err != nil {
		http.Error(w, "render: "+err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"count":    res.Count,
		"hash":     res.Hash,
		"rows":     doc.Rows,
		"rows_top": pages.ContentsRowsTop,
		"row_h":    pages.ContentsRowH,
		"layout": map[string]int{
			"page_w": pages.PageW, "page_h": pages.PageH,
		},
	})
}

func (s *Server) handleContentsPage(w http.ResponseWriter, r *http.Request) {
	sub, ok := s.subject(r.PathValue("subject"))
	if !ok {
		http.Error(w, "unknown subject", http.StatusNotFound)
		return
	}
	n, err := strconv.Atoi(r.PathValue("page"))
	if err != nil || n < 0 {
		http.Error(w, "bad page number", http.StatusBadRequest)
		return
	}
	res, err := sub.Pages.RenderContents(s.buildContentsDoc(sub))
	if err != nil {
		http.Error(w, "render: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if n >= res.Count {
		http.Error(w, "page out of range", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	// The contents page changes with learner state; cache briefly.
	w.Header().Set("Cache-Control", "max-age=60")
	http.ServeFile(w, r, res.PagePath(n))
}

func (s *Server) handlePage(w http.ResponseWriter, r *http.Request) {
	sub, ok := s.subject(r.PathValue("subject"))
	if !ok {
		http.Error(w, "unknown subject", http.StatusNotFound)
		return
	}
	n, err := strconv.Atoi(r.PathValue("page"))
	if err != nil || n < 0 {
		http.Error(w, "bad page number", http.StatusBadRequest)
		return
	}
	ch, err := requestChapter(sub, r)
	if err != nil {
		http.Error(w, "no chapter available", http.StatusNotFound)
		return
	}
	res, err := sub.Pages.Render(ch)
	if err != nil {
		http.Error(w, "render: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if n >= res.Count {
		http.Error(w, "page out of range", http.StatusNotFound)
		return
	}
	// Content addressing cuts both ways: a request carrying another
	// chapter's hash must fail, never silently receive the CURRENT
	// chapter's pixels - a client with stale meta would composite its old
	// answer regions over them.
	if want := r.URL.Query().Get("v"); want != "" && want != res.Hash {
		http.Error(w, "stale page hash", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	// Pages are content-addressed by chapter hash, so clients can cache hard.
	w.Header().Set("Cache-Control", "max-age=86400")
	http.ServeFile(w, r, res.PagePath(n))
}
