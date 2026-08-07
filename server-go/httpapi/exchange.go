package httpapi

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/mjbraun/chiron/server/checkers"
	"github.com/mjbraun/chiron/server/corpus"
	"github.com/mjbraun/chiron/server/render"
	"github.com/mjbraun/chiron/server/review"
	"github.com/mjbraun/chiron/server/roles"
	"github.com/mjbraun/chiron/server/state"
	"time"
)

type BeatResponse struct {
	BeatID   string `json:"beat_id"`
	Response string `json:"response"`
	// The learner's own pass/fail after the reveal, on self-explain beats.
	SelfVerdict string `json:"self_verdict,omitempty"`
	// The JS-graded result for compute beats, graded offline while reading.
	MechanicalVerdict string `json:"mechanical_verdict,omitempty"`
}

type ItemResponse struct {
	ItemID        string `json:"item_id"`
	Response      string `json:"response,omitempty"`
	SelectedIndex *int   `json:"selected_index,omitempty"`
	Confidence    int    `json:"confidence"` // 1-4, captured BEFORE the reveal
	// IDK is an explicit "I don't know - move on". It grades as a fail
	// without a model call: pretests expect misses, and typing filler to
	// satisfy a required answer field is worse signal than saying so.
	IDK bool `json:"idk,omitempty"`
}

type Exchange struct {
	Subject          string         `json:"subject"`
	Phase            string         `json:"phase"` // boundary | pretest | start
	Unit             string         `json:"unit,omitempty"`
	BeatResponses    []BeatResponse `json:"beat_responses"`
	PretestResponses []ItemResponse `json:"pretest_responses"`
	CheckResponses   []ItemResponse `json:"check_responses"`
	Override         bool           `json:"override"`
	SkippedCheck     bool           `json:"skipped_check"`
	CatchMeUp        bool           `json:"catch_me_up"`
	Choice           string         `json:"choice,omitempty"`
	ChunkMinutes     float64        `json:"chunk_minutes,omitempty"`
	BreakMinutes     float64        `json:"break_minutes,omitempty"`
}

type Result struct {
	ItemID string `json:"item_id"`
	roles.Grade
	Confidence  *int   `json:"confidence,omitempty"`
	SelfVerdict string `json:"self_verdict,omitempty"`
}

type Gate struct {
	Score             *float64 `json:"score"`
	Passed            bool     `json:"passed"`
	Gate              float64  `json:"gate"`
	ExtensionUnlocked bool     `json:"extension_unlocked"`
	// Calibration marks a measuring unit: the gate always passes and the
	// score is information for the planner, not a verdict on the learner.
	Calibration bool `json:"calibration,omitempty"`
}

type BreakSuggestion struct {
	Minutes int    `json:"minutes"`
	Kind    string `json:"kind"`
	Note    string `json:"note"`
}

func passedVerdict(v string) bool {
	return v == "pass" || v == "valid_alternative_path"
}

// gradeItems grades item responses - mechanically in code, through the model
// only for free text. It returns (passed, total).
func (s *Server) gradeItems(sub *Subject, responses []ItemResponse, results *[]Result) (int, int) {
	type resolved struct {
		r      ItemResponse
		q      *corpus.Question
		unitID string
	}
	// Resolve everything first so free-text items can optionally be graded in a
	// single call. Submission order is preserved for both events and results.
	var items []resolved
	for _, r := range responses {
		q, unitID := sub.Corpus.FindQuestion(r.ItemID)
		if q != nil {
			items = append(items, resolved{r, q, unitID})
		}
	}

	var freeText []roles.BatchItem
	for _, it := range items {
		if it.r.IDK {
			continue // graded without a model call below
		}
		if it.q.Kind != "mcq" && !checkers.IsMechanical(it.q.Check) {
			freeText = append(freeText, roles.BatchItem{
				ItemID: it.r.ItemID, Question: it.q, Answer: it.r.Response,
			})
		}
	}
	batched := map[string]roles.Grade{}
	if s.cfg.BatchGrading && len(freeText) > 1 {
		batched = roles.GradeFreeTextBatch(s.chain, freeText, sub.Corpus.Misconceptions)
	}

	passed := 0
	for _, it := range items {
		var g roles.Grade
		if it.q.Check == "screener" {
			rating := parseRating(it.r)
			sc := float64(rating)
			sub.Learner.Apply(state.Event{Kind: "self_rating", Score: &sc})
			g = roles.Grade{Verdict: "pass",
				FeedbackMD: fmt.Sprintf("Placed at level %d.", rating)}
			*results = append(*results, Result{ItemID: it.r.ItemID, Grade: g})
			passed++
			continue
		}
		switch {
		case it.r.IDK:
			fb := "Marked \"I don't know\"."
			if it.q.Kind != "mcq" && it.q.Answer.String() != "" {
				fb += " Reference: " + it.q.Answer.String()
			}
			g = roles.Grade{Verdict: "fail", Misconceptions: []string{}, FeedbackMD: fb}
		case it.q.Kind == "mcq":
			idx := -1
			if it.r.SelectedIndex != nil {
				idx = *it.r.SelectedIndex
			}
			m := checkers.CheckMCQ(it.q.Options, idx)
			g = roles.Grade{Verdict: m.Verdict, Misconceptions: m.Misconceptions,
				FeedbackMD: m.Explain}
		case checkers.IsMechanical(it.q.Check):
			ok, err := checkers.CheckAnswer(it.q.Check, it.q.Answer.String(), it.r.Response)
			verdict := "fail"
			if err != nil {
				// A check spec the server cannot evaluate is a corpus bug.
				// Marking it wrong would blame the learner for it.
				verdict = "ungraded"
			} else if ok {
				verdict = "pass"
			}
			g = roles.Grade{Verdict: verdict, Misconceptions: []string{},
				FeedbackMD: "Reference: " + it.q.Answer.String()}
		default:
			// Fall back per item for anything the batch did not return, so a
			// partial batch degrades rather than silently dropping grades.
			if b, ok := batched[it.r.ItemID]; ok {
				g = b
			} else {
				g = roles.GradeFreeText(s.chain, it.q, it.r.Response, sub.Corpus.Misconceptions)
			}
		}
		if passedVerdict(g.Verdict) {
			passed++
		}
		conf := it.r.Confidence
		if _, err := sub.Learner.Apply(state.Event{
			Kind: "item_graded", Item: it.r.ItemID, Concept: it.q.Concept,
			Unit: it.unitID, Verdict: g.Verdict, Confidence: &conf,
			Misconceptions: g.Misconceptions, Evidence: g.Evidence,
		}); err != nil {
			return passed, len(responses)
		}
		*results = append(*results, Result{ItemID: it.r.ItemID, Grade: g, Confidence: &conf})
	}
	return passed, len(responses)
}

func (s *Server) gradeBeats(sub *Subject, responses []BeatResponse, results *[]Result) {
	for _, r := range responses {
		beat, unitID := sub.Corpus.FindBeat(r.BeatID)
		if beat == nil {
			continue
		}
		check := beat.Check
		if check == "" {
			check = "llm"
		}
		var g roles.Grade
		if checkers.IsMechanical(check) {
			ok, err := checkers.CheckAnswer(check, beat.Answer.String(), r.Response)
			verdict := "fail"
			if err != nil {
				verdict = "ungraded"
			} else if ok {
				verdict = "pass"
			}
			g = roles.Grade{Verdict: verdict, Misconceptions: []string{}}
		} else {
			g = roles.GradeFreeText(s.chain, &corpus.Question{
				Prompt: beat.Prompt, Answer: beat.Answer, Rubric: beat.Rubric,
			}, r.Response, sub.Corpus.Misconceptions)
		}
		if _, err := sub.Learner.Apply(state.Event{
			Kind: "item_graded", Item: r.BeatID, Concept: beat.Concept, Unit: unitID,
			Verdict: g.Verdict, Misconceptions: g.Misconceptions, Evidence: g.Evidence,
		}); err != nil {
			return
		}
		*results = append(*results, Result{ItemID: r.BeatID, Grade: g, SelfVerdict: r.SelfVerdict})
	}
}

// parseRating reads a 1-5 self-placement from a screener response. Anything
// unreadable (including "I don't know") lands in the middle.
func parseRating(r ItemResponse) int {
	if r.SelectedIndex != nil {
		if n := *r.SelectedIndex + 1; n >= 1 && n <= 5 {
			return n
		}
	}
	for _, c := range r.Response {
		if c >= '1' && c <= '5' {
			return int(c - '0')
		}
	}
	return 3
}

// calibrationItems selects a calibration unit's delivery for a self-rating:
// the screener alone before a rating exists, the level's pre-computed series
// after, the whole bank if no set is defined for the level.
func calibrationItems(unit *corpus.Unit, rating int) []corpus.Question {
	if rating == 0 && unit.Questions.Screener != nil {
		q := *unit.Questions.Screener
		q.Unit = unit.ID
		return []corpus.Question{q}
	}
	var items []corpus.Question
	if ids := unit.Questions.CalibrationSets[rating]; len(ids) > 0 {
		byID := map[string]corpus.Question{}
		for _, q := range unit.Questions.Check {
			byID[q.ID] = q
		}
		for _, id := range ids {
			if q, ok := byID[id]; ok {
				q.Unit = unit.ID
				items = append(items, q)
			}
		}
		return items
	}
	for _, q := range unit.Questions.Check {
		q.Unit = unit.ID
		items = append(items, q)
	}
	return items
}

// prerenderCalibrationSets renders the page stack for every level, middle
// levels first (they are the most likely picks). Chapter composition here
// must stay byte-identical to what buildChapter will produce once the
// rating is recorded, or the cache keys will not line up.
func (s *Server) prerenderCalibrationSets(sub *Subject, unit *corpus.Unit, checkSummary string) {
	d := roles.PlanDirectives(s.chain, sub.Learner, unit, checkSummary)
	sections, _ := roles.AuthorChapter(s.chain, unit, d, sub.Corpus)
	var wg sync.WaitGroup
	for rating := 1; rating <= 5; rating++ {
		if len(unit.Questions.CalibrationSets[rating]) == 0 {
			continue
		}
		cp := *unit
		cp.IntroMD = "" // must match buildChapter's series composition
		ch, err := render.RenderChapter(&cp, sections,
			render.Directives{OpeningNoteMD: d.OpeningNoteMD, NextAction: d.NextAction},
			unit.Questions.Pretest, calibrationItems(unit, rating))
		if err != nil {
			continue
		}
		wg.Add(1)
		go func(rating int, ch *render.Chapter) {
			defer wg.Done()
			if _, err := sub.Pages.Render(ch); err != nil {
				log.Printf("prerender calibration level %d: %v", rating, err)
			}
		}(rating, ch)
	}
	wg.Wait()
}

func (s *Server) nextUnit(sub *Subject, choice string) string {
	fringe := sub.Learner.Fringe()
	if len(fringe) == 0 {
		return ""
	}
	for _, uid := range fringe {
		if uid == choice {
			return choice
		}
	}
	return fringe[0]
}

func (s *Server) buildChapter(sub *Subject, unitID, checkSummary string) (*render.Chapter, error) {
	unit, ok := sub.Corpus.Units[unitID]
	if !ok {
		return nil, fmt.Errorf("unit %s not authored", unitID)
	}
	d := roles.PlanDirectives(s.chain, sub.Learner, unit, checkSummary)
	sections, _ := roles.AuthorChapter(s.chain, unit, d, sub.Corpus)

	var items []corpus.Question
	if unit.IsCalibration() {
		// Self-placement first; then the pre-computed series for that
		// level, complete and in authored order - no shuffle, no cap.
		rating := sub.Learner.Data.Profile.SelfRating
		items = calibrationItems(unit, rating)
		if rating == 0 {
			// Every level's series is already known: render all five page
			// stacks while the learner reads the placement question, so
			// whichever they pick is on disk before they ask for it.
			go s.prerenderCalibrationSets(sub, unit, checkSummary)
		}
	} else {
		s.rngMu.Lock()
		items = roles.ComposeCheck(unit, sub.Learner, sub.Corpus,
			s.cfg.Session.CheckItems, s.cfg.Session.CallbackFraction, s.rng)
		s.rngMu.Unlock()
	}

	if _, err := sub.Learner.Apply(state.Event{Kind: "unit_started", Unit: unitID}); err != nil {
		return nil, err
	}
	if _, err := sub.Learner.Apply(state.Event{Kind: "exposed", Unit: unitID,
		Concepts: unit.ConceptIDs()}); err != nil {
		return nil, err
	}
	if _, err := sub.Learner.Apply(state.Event{Kind: "summary", Text: d.Summary}); err != nil {
		return nil, err
	}
	if assumes, ok := unit.Front["assumes"].([]any); ok && len(assumes) > 0 {
		var concepts []string
		for _, a := range assumes {
			if s, ok := a.(string); ok {
				concepts = append(concepts, s)
			}
		}
		if len(concepts) > 0 {
			if _, err := sub.Learner.Apply(state.Event{Kind: "assumed_known",
				Concepts: concepts}); err != nil {
				return nil, err
			}
		}
	}
	ru := unit
	if unit.IsCalibration() && sub.Learner.Data.Profile.SelfRating > 0 {
		// The intro frames the placement step; once placed, the series
		// opens on its first question, not on stale framing.
		cp := *unit
		cp.IntroMD = ""
		ru = &cp
	}
	return render.RenderChapter(ru, sections,
		render.Directives{OpeningNoteMD: d.OpeningNoteMD, NextAction: d.NextAction},
		unit.Questions.Pretest, items)
}

// buildCatchup is the comprehensive backfill from the whole debt ledger:
// representation-switched sections for every debted concept, then a combined
// check that can retire the debt.
func (s *Server) buildCatchup(sub *Subject) (*render.Chapter, error) {
	debt := sub.Learner.OpenDebt()
	if len(debt) == 0 {
		return nil, nil
	}
	var sections []render.AssembledSection
	var checkPool []corpus.Question

	for _, d := range debt {
		unit, ok := sub.Corpus.Units[d.Unit]
		if !ok {
			continue
		}
		// Remediation must switch representation, never re-present the same
		// text more slowly - that is the documented failure of mastery
		// programs.
		variant := ""
		for _, v := range []string{"more-intuition", "se-analogies", "deeper-math"} {
			if _, ok := unit.Depths[v]; ok {
				variant = v
				break
			}
		}
		debted := map[string]bool{}
		for _, c := range d.Concepts {
			debted[c] = true
		}
		for _, sec := range unit.Sections {
			covers := len(d.Concepts) == 0
			for _, seg := range sec.Segments {
				if seg.Type == "beat" && debted[seg.Beat.Concept] {
					covers = true
				}
			}
			if !covers {
				continue
			}
			md := sec.Markdown()
			if variant != "" {
				if text, ok := unit.Depths[variant][sec.Heading]; ok && len(text) > 400 {
					md = "## " + sec.Heading + "\n\n" + text
				}
			}
			sections = append(sections, render.AssembledSection{
				Heading: fmt.Sprintf("[%s] %s", unit.ID, sec.Heading), Markdown: md,
			})
		}
		missed := map[string]bool{}
		for _, id := range d.ItemsMissed {
			missed[id] = true
		}
		for _, q := range unit.Questions.Check {
			if missed[q.ID] || debted[q.Concept] {
				q.Unit = unit.ID
				checkPool = append(checkPool, q)
			}
		}
	}
	if len(sections) == 0 {
		return nil, nil
	}

	minutes := 8 * len(debt)
	if minutes < 15 {
		minutes = 15
	}
	catchup := &corpus.Unit{
		ID: "catchup", Title: "Catch-up: closing the gaps", Minutes: minutes,
		IntroMD: "This chapter consolidates everything skipped or missed so far, " +
			"explained differently than the first pass. The check at the end " +
			"retires the debt it covers.",
	}
	s.rngMu.Lock()
	s.rng.Shuffle(len(checkPool), func(i, j int) {
		checkPool[i], checkPool[j] = checkPool[j], checkPool[i]
	})
	s.rngMu.Unlock()
	n := len(checkPool) / 2
	if n < 8 {
		n = 8
	}
	if n > len(checkPool) {
		n = len(checkPool)
	}
	return render.RenderChapter(catchup, sections, render.Directives{
		NextAction: "Work every section, then take the combined check.",
	}, nil, checkPool[:n])
}

func (s *Server) handleExchange(w http.ResponseWriter, r *http.Request) {
	var ex Exchange
	if err := json.NewDecoder(r.Body).Decode(&ex); err != nil {
		writeError(w, http.StatusBadRequest, "malformed request: %v", err)
		return
	}
	if ex.Subject == "" {
		ex.Subject = "ai"
	}
	if ex.Phase == "" {
		ex.Phase = "boundary"
	}
	sub, ok := s.subject(ex.Subject)
	if !ok {
		writeError(w, http.StatusNotFound, "unknown subject: %s", ex.Subject)
		return
	}
	writeJSON(w, http.StatusOK, s.processExchange(sub, ex))
}

// processExchange is the whole boundary contract - grade, gate, plan, author,
// persist - shared by the JSON exchange endpoint and the ink check-in.
func (s *Server) processExchange(sub *Subject, ex Exchange) map[string]any {
	results := []Result{}
	var gate *Gate
	var chapter *render.Chapter
	var breakSuggestion *BreakSuggestion
	sess := s.cfg.Session

	// Telemetry first: it feeds the fatigue term.
	if ex.ChunkMinutes > 0 {
		m := ex.ChunkMinutes
		sub.Learner.Apply(state.Event{Kind: "chunk", Minutes: &m, Unit: ex.Unit})
	}
	if ex.BreakMinutes > 0 {
		m := ex.BreakMinutes
		sub.Learner.Apply(state.Event{Kind: "break_taken", Minutes: &m})
	}

	s.gradeBeats(sub, ex.BeatResponses, &results)

	var summary strings.Builder
	if len(ex.PretestResponses) > 0 {
		p, t := s.gradeItems(sub, ex.PretestResponses, &results)
		fmt.Fprintf(&summary, "Pretest: %d/%d. ", p, t)
	}

	screenerOnly := len(ex.CheckResponses) > 0
	for _, r := range ex.CheckResponses {
		if q, _ := sub.Corpus.FindQuestion(r.ItemID); q == nil || q.Check != "screener" {
			screenerOnly = false
			break
		}
	}

	switch {
	case screenerOnly:
		// Self-placement recorded; no gate, no check_result. The chapter
		// build below re-delivers the calibration unit, now carrying the
		// series for the chosen level.
		s.gradeItems(sub, ex.CheckResponses, &results)
	case len(ex.CheckResponses) > 0:
		p, t := s.gradeItems(sub, ex.CheckResponses, &results)
		score := 0.0
		if t > 0 {
			score = float64(p) / float64(t)
		}
		unit, haveUnit := sub.Corpus.Units[ex.Unit]
		calibration := haveUnit && unit.IsCalibration()
		passed := score >= sess.MasteryGate || calibration
		var mastered []string
		if haveUnit && passed && !calibration {
			mastered = unit.ConceptIDs()
		}
		sub.Learner.Apply(state.Event{Kind: "check_result", Unit: ex.Unit,
			Score: &score, Passed: &passed, MasteredConcepts: mastered})

		if passed && ex.Unit == "catchup" {
			for _, d := range sub.Learner.OpenDebt() {
				sub.Learner.Apply(state.Event{Kind: "debt_retired", Unit: d.Unit})
			}
		}
		var missed []string
		for _, res := range results {
			if !passedVerdict(res.Verdict) {
				missed = append(missed, res.ItemID)
			}
		}
		sc := score
		gate = &Gate{Score: &sc, Passed: passed, Gate: sess.MasteryGate,
			ExtensionUnlocked: !calibration && score >= sess.ExtensionTrigger,
			Calibration:       calibration}
		if calibration {
			fmt.Fprintf(&summary, "Calibration %s: %.0f%% overall; per-concept levels are in the state. ", ex.Unit, score*100)
		} else {
			verdict := "below gate"
			if passed {
				verdict = "passed"
			}
			fmt.Fprintf(&summary, "Check %s: %.0f%% (%s). ", ex.Unit, score*100, verdict)
		}

		if !passed && ex.Override && haveUnit {
			var unmastered []string
			for _, c := range unit.ConceptIDs() {
				if sub.Learner.ConceptLevel(c) != "mastered" {
					unmastered = append(unmastered, c)
				}
			}
			sub.Learner.Apply(state.Event{Kind: "override", Unit: ex.Unit,
				Concepts: unmastered, ItemsMissed: missed, Reason: "failed_gate"})
		}

	case ex.SkippedCheck && ex.Unit != "":
		if unit, ok := sub.Corpus.Units[ex.Unit]; ok {
			sub.Learner.Apply(state.Event{Kind: "override", Unit: ex.Unit,
				Concepts: unit.ConceptIDs(), Reason: "skipped_check"})
			gate = &Gate{Passed: false, Gate: sess.MasteryGate}
		}
	}

	// Fatigue heuristic: 90+ minutes without a logged break earns a suggestion
	// and a flag. Late-session errors are fatigue, not knowledge gaps, and must
	// not be read as a mastery downgrade.
	if sub.Learner.MinutesSinceBreak() >= sess.LongBreakEveryChunks*sess.ChunkMinutes {
		sub.Learner.Apply(state.Event{Kind: "fatigue"})
		breakSuggestion = &BreakSuggestion{Minutes: 15, Kind: "long",
			Note: "90+ minutes since a break. Late-session errors will be misread " +
				"as knowledge gaps - rest first."}
	} else if ex.ChunkMinutes >= sess.ChunkMinutes && ex.ChunkMinutes > 0 {
		breakSuggestion = &BreakSuggestion{Minutes: int(sess.BreakMinutes), Kind: "short",
			Note: "Chunk done. Five minutes, eyes off screens."}
	}

	advanceAllowed := gate == nil || gate.Passed || ex.Override || ex.SkippedCheck
	if ex.CatchMeUp {
		if ch, err := s.buildCatchup(sub); err == nil {
			chapter = ch
		}
	}
	if chapter == nil && (ex.Phase == "boundary" || ex.Phase == "start") && advanceAllowed {
		if next := s.nextUnit(sub, ex.Choice); next != "" {
			if ch, err := s.buildChapter(sub, next, summary.String()); err == nil {
				chapter = ch
			}
		}
	} else if chapter == nil && gate != nil && !gate.Passed && !ex.Override && ex.Unit != "" {
		// Remediation loop: rebuild the SAME unit. The planner sees the failed
		// check in the summary and must switch representation.
		if ch, err := s.buildChapter(sub, ex.Unit, summary.String()+
			"REMEDIATE: switch representation, do not re-explain the same way."); err == nil {
			chapter = ch
		}
	}

	if chapter != nil {
		if err := persistChapter(sub, chapter); err != nil {
			log.Printf("persist chapter %s: %v", chapter.Unit, err)
		}
		// Render the page images now, while the learner is still reading
		// their results - by the time they ask for the next chapter it is
		// already on disk. The renderer serializes with the on-demand path.
		go func(ch *render.Chapter) {
			if _, err := sub.Pages.Render(ch); err != nil {
				log.Printf("eager page render %s: %v", ch.Unit, err)
			}
		}(chapter)
	}
	return map[string]any{
		"results":          results,
		"gate":             gate,
		"chapter":          chapter,
		"state":            s.statePayload(sub),
		"break_suggestion": breakSuggestion,
	}
}

func (s *Server) statePayload(sub *Subject) map[string]any {
	type spineRow struct {
		Unit     string   `json:"unit"`
		Title    string   `json:"title"`
		Status   string   `json:"status"`
		Score    *float64 `json:"score"`
		InFringe bool     `json:"in_fringe"`
	}
	fringe := sub.Learner.Fringe()
	inFringe := map[string]bool{}
	for _, uid := range fringe {
		inFringe[uid] = true
	}
	snapshot := sub.Learner.Snapshot()
	spine := []spineRow{}
	for _, uid := range sub.Corpus.UnitOrder() {
		title := uid
		if u, ok := sub.Corpus.Units[uid]; ok {
			title = u.Title
		}
		var score *float64
		if us, ok := snapshot.Units[uid]; ok {
			score = us.CheckScore
		}
		spine = append(spine, spineRow{
			Unit: uid, Title: title, Status: sub.Learner.UnitStatus(uid),
			Score: score, InFringe: inFringe[uid],
		})
	}
	if fringe == nil {
		fringe = []string{}
	}
	debt := sub.Learner.OpenDebt()
	if debt == nil {
		debt = []*state.Debt{}
	}
	mis := sub.Learner.ActiveMisconceptions()
	if mis == nil {
		mis = []string{}
	}
	return map[string]any{
		"spine":                 spine,
		"fringe":                fringe,
		"debt":                  debt,
		"active_misconceptions": mis,
		"summary":               snapshot.Summary,
		"session_minutes":       sub.Learner.SessionMinutes(),
		"llm":                   s.chain.Status(),
	}
}

func (s *Server) handleState(w http.ResponseWriter, r *http.Request) {
	sub, ok := s.subject(s.subjectParam(r))
	if !ok {
		writeError(w, http.StatusNotFound, "unknown subject: %s", s.subjectParam(r))
		return
	}
	writeJSON(w, http.StatusOK, s.statePayload(sub))
}

// handleReviewSchedule returns the day-1/3/10 plan built from what actually
// went wrong. It is self-contained - prompts and answers inline - so it is
// useful with no server, no model and no network.
func (s *Server) handleReviewSchedule(w http.ResponseWriter, r *http.Request) {
	sub, ok := s.subject(s.subjectParam(r))
	if !ok {
		writeError(w, http.StatusNotFound, "unknown subject: %s", s.subjectParam(r))
		return
	}
	// Served as text/plain, not HTML: this is a markdown document meant to be
	// printed or synced to a reader, so there is no markup context for the
	// learner's own answers to escape into.
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, review.Build(sub.Learner, sub.Corpus, sub.Title, time.Now()))
}
