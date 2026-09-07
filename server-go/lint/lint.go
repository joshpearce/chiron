// Package lint holds the mechanical corpus checks - the automatable half of
// the verify pass.
//
// Three agents audited the AI corpus by hand and found ~50 defects. The
// judgement-heavy findings (is this claim about transformers true? does this
// rubric actually adjudicate?) still need a reader. Everything else was
// mechanical, recurring, and exactly the kind of thing that silently regresses:
// answers that are prose where a program must parse them, tolerances loose
// enough to accept the very error the item exists to catch, distractors citing
// misconception ids that do not exist, depth files whose headings drifted from
// canon so the section swap silently does nothing.
package lint

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/mjbraun/chiron/server/corpus"
)

var (
	numericSpec   = regexp.MustCompile(`^numeric\(([\d.eE+-]+)\)$`)
	refutesMarker = regexp.MustCompile(`<!--\s*refutes:\s*([A-Za-z0-9_-]+)\s*-->`)
	machineAnswer = regexp.MustCompile(`^(?:[-+0-9A-Fa-f.eE/×x,;|\[\]() ]+|[A-Za-z0-9 _.\-/|]{1,24})$`)
)

type Report struct {
	Errors   []string
	Warnings []string
}

func (r *Report) errorf(where, format string, a ...any) {
	r.Errors = append(r.Errors, where+": "+fmt.Sprintf(format, a...))
}

func (r *Report) warnf(where, format string, a ...any) {
	r.Warnings = append(r.Warnings, where+": "+fmt.Sprintf(format, a...))
}

// isMachineAnswer reports whether a mechanically-checked item answers with
// something a program can compare, rather than a sentence explaining it.
func isMachineAnswer(v string) bool {
	text := strings.TrimSpace(v)
	if text == "" || len(strings.Fields(text)) > 8 {
		return false
	}
	// Vectors, shapes, hex bytes and tokenizations ("[7, -4]", "4 x 7",
	// "0xB2,0x00,0x2F", "w|i|d|est_") all compare fine as exact strings. The
	// word-count guard above is what actually keeps prose out; these classes
	// only have to be wide enough not to reject a real answer.
	return machineAnswer.MatchString(text)
}

// plausibleWrongAnswers are the error paths a learner actually takes: a dropped
// or doubled factor, a sign flip, an order-of-magnitude slip. Off-by-one is
// deliberately excluded - a tolerance reaching expected±1 is usually the
// author's intent, not a misconception the item exists to catch.
func plausibleWrongAnswers(expected float64) []float64 {
	out := []float64{expected * 2, expected / 2, -expected}
	if expected != 0 {
		out = append(out, expected*10, expected/10)
	}
	var keep []float64
	for _, v := range out {
		if v != expected && !math.IsNaN(v) {
			keep = append(keep, v)
		}
	}
	return keep
}

type checkable struct {
	id     string
	check  string
	answer string
}

func checkables(u *corpus.Unit) []checkable {
	var out []checkable
	for _, b := range u.Beats() {
		out = append(out, checkable{b.ID, b.Check, b.Answer.String()})
	}
	for _, pool := range [][]corpus.Question{u.Questions.Check, u.Questions.Pretest} {
		for _, q := range pool {
			out = append(out, checkable{q.ID, q.Check, q.Answer.String()})
		}
	}
	return out
}

// checkNumericTolerances catches a tolerance loose enough to accept the very
// error the item exists to catch.
func checkNumericTolerances(u *corpus.Unit, r *Report) {
	for _, it := range checkables(u) {
		m := numericSpec.FindStringSubmatch(strings.TrimSpace(it.check))
		if m == nil {
			continue
		}
		where := u.ID + "/" + it.id
		expected, err := strconv.ParseFloat(strings.TrimSpace(it.answer), 64)
		if err != nil {
			r.errorf(where, "check is %s but answer is not numeric: %q", it.check, it.answer)
			continue
		}
		tol, err := strconv.ParseFloat(m[1], 64)
		if err != nil {
			r.errorf(where, "unparseable tolerance in %s", it.check)
			continue
		}
		// Must mirror checkers.CheckAnswer exactly.
		window := tol + math.Abs(expected)*1e-9
		for _, wrong := range plausibleWrongAnswers(expected) {
			if math.Abs(wrong-expected) <= window {
				r.errorf(where, "tolerance %s accepts %g as correct (expected %g, window +/-%g)",
					it.check, wrong, expected, window)
				break
			}
		}
		if math.Abs(expected) >= 1e6 && it.check != "exact" {
			r.warnf(where, "large answer %g with numeric tolerance - prefer check: exact for big integers", expected)
		}
	}
}

func checkAnswersMachineReadable(u *corpus.Unit, r *Report) {
	for _, it := range checkables(u) {
		check := it.check
		if check == "" || check == "llm" || check == "choice" {
			continue
		}
		if !isMachineAnswer(it.answer) {
			answer := it.answer
			if len(answer) > 60 {
				answer = answer[:60]
			}
			r.errorf(u.ID+"/"+it.id, "check is %s but answer is prose: %q", check, answer)
		}
	}
}

func checkMCQs(u *corpus.Unit, c *corpus.Corpus, r *Report) {
	for _, pool := range [][]corpus.Question{u.Questions.Pretest, u.Questions.Check} {
		for _, q := range pool {
			if q.Kind != "mcq" {
				continue
			}
			where := u.ID + "/" + q.ID
			correct := 0
			for _, o := range q.Options {
				if o.Correct {
					correct++
				}
			}
			if correct != 1 {
				r.errorf(where, "%d options marked correct, expected exactly 1", correct)
			}
			if q.Check != "choice" {
				r.errorf(where, "mcq must set check: choice (found %q)", q.Check)
			}
			for _, o := range q.Options {
				if o.Explain == "" {
					r.errorf(where, "an option has no explain - feedback must adjudicate "+
						"every option, not just the right one")
				}
				if o.Correct {
					continue
				}
				if o.Misconception == "" {
					r.errorf(where, "distractor has no misconception id - a wrong answer "+
						"must be a diagnosis")
				} else if _, ok := c.Misconceptions[o.Misconception]; !ok {
					r.errorf(where, "distractor cites unknown misconception %q", o.Misconception)
				}
			}
		}
	}
}

func checkSpecConformance(u *corpus.Unit, r *Report) {
	checks := u.Questions.Check
	where := u.ID
	if len(checks) < 8 {
		r.errorf(where, "%d check items, spec requires >= 8 (fewer is statistical "+
			"noise for an 80%% gate)", len(checks))
	}
	// The mix: about half MCQ, most constructed items answered with a
	// number or a term, prose at most a fifth. A reader on a phone taps.
	constructed, prose := 0, 0
	for _, q := range checks {
		if q.Kind == "constructed" {
			constructed++
			if q.Check == "" || q.Check == "llm" {
				prose++
			}
		}
	}
	if len(checks) > 0 && float64(constructed)/float64(len(checks)) < 0.2 {
		r.warnf(where, "%d/%d constructed; a bank that is all choices never asks for a number "+
			"or a term the reader has to produce", constructed, len(checks))
	}
	if prose > 0 {
		r.errorf(where, "%d/%d check items are answered in prose (check: llm); every item is mcq, "+
			"numeric or exact, so the reader taps and nothing waits on a grader", prose, len(checks))
	}
	checkTapOnly(u, r)
	eligible := 0
	for _, q := range checks {
		if q.CallbackEligible {
			eligible++
		}
	}
	if eligible < 2 {
		r.warnf(where, "fewer than 2 callback_eligible items - later units have little "+
			"to draw on for cumulative checks")
	}
	beats := u.Beats()
	if len(beats) < 6 && !u.IsCalibration() {
		r.warnf(where, "%d interaction beats, spec asks for 6-10 (step-based interaction "+
			"is where the tutoring effect lives)", len(beats))
	}
	proseBeats := 0
	for _, b := range beats {
		if b.Type == "self-explain" && b.Rubric == "" && len(b.Options) == 0 {
			r.errorf(u.ID+"/"+b.ID, "self-explain beat has no rubric - an unadjudicated "+
				"self-explanation can entrench a wrong model")
		}
		if len(b.Options) == 0 && (b.Check == "" || b.Check == "llm") {
			proseBeats++
		}
	}
	if proseBeats > 0 {
		r.errorf(where, "%d beats answered in prose; a beat is a choice (options) or a computed "+
			"number or term, so the reader taps and nothing waits on a grader", proseBeats)
	}
}

func checkDepthHeadings(u *corpus.Unit, r *Report) {
	canon := map[string]bool{}
	canonOnly := map[string]bool{}
	var canonOrder []string
	for _, s := range u.Sections {
		canon[s.Heading] = true
		canonOrder = append(canonOrder, s.Heading)
		// Sections marked canon-only (notation references, corrections
		// transplanted between units) are variantless by design.
		if strings.Contains(s.Markdown(), "<!-- canon-only -->") {
			canonOnly[s.Heading] = true
		}
	}
	var names []string
	for name := range u.Depths {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		sections := u.Depths[name]
		where := fmt.Sprintf("%s/depths/%s", u.ID, name)
		var drifted []string
		for heading := range sections {
			if !canon[heading] {
				drifted = append(drifted, heading)
			}
		}
		if len(drifted) > 0 {
			sort.Strings(drifted)
			r.errorf(where, "headings not in canon (swap silently does nothing): %v",
				first(drifted, 3))
		}
		// The same silent no-op from the other direction: the planner can ask
		// for a section at this depth and get the canon text back, with nothing
		// anywhere saying the variant was never written.
		var missing []string
		for _, heading := range canonOrder {
			if _, ok := sections[heading]; !ok && !canonOnly[heading] {
				missing = append(missing, heading)
			}
		}
		if len(missing) > 0 {
			r.warnf(where, "%d canon section(s) have no variant here, so requesting this "+
				"depth silently returns canon: %v", len(missing), first(missing, 3))
		}
	}
}

func checkRefutationIDs(u *corpus.Unit, c *corpus.Corpus, r *Report) {
	var text strings.Builder
	for _, s := range u.Sections {
		text.WriteString(s.Markdown())
		text.WriteString("\n")
	}
	for _, m := range refutesMarker.FindAllStringSubmatch(text.String(), -1) {
		if _, ok := c.Misconceptions[m[1]]; !ok {
			r.errorf(u.ID, "<!-- refutes: %s --> does not resolve to a misconception in the bank", m[1])
		}
	}
}

// checkCalibrationSets holds a calibration unit to the one thing its screener
// exists for: sizing the learner within the level they claimed. The original
// u0 sets only trimmed from the top of a single ladder, so "absolute novice"
// received the same ten dot-product and matrix-shape items as level 3 - the
// self-rating changed nothing a novice could feel. A level's series must be a
// window: a floor from the band below, the bulk from its own band, a ceiling
// from the band above; and the bank must actually contain every band.
func checkCalibrationSets(u *corpus.Unit, r *Report) {
	if !u.IsCalibration() {
		return
	}
	byID := map[string]corpus.Question{}
	perBand := map[int]int{}
	for _, q := range u.Questions.Check {
		byID[q.ID] = q
		if q.Band < 1 || q.Band > 5 {
			r.errorf(u.ID+"/"+q.ID, "calibration item has no band (1-5) - without one the "+
				"level series cannot be a window around the level")
			continue
		}
		perBand[q.Band]++
	}
	for b := 1; b <= 5; b++ {
		if perBand[b] < 3 {
			r.errorf(u.ID, "bank has %d item(s) in band %d, need >= 3 - a level with nothing "+
				"at its own band cannot size a learner within it", perBand[b], b)
		}
	}
	for l := 1; l <= 5; l++ {
		ids := u.Questions.CalibrationSets[l]
		where := fmt.Sprintf("%s/level %d", u.ID, l)
		if len(ids) == 0 {
			r.errorf(u.ID, "no calibration set for level %d - that screener answer falls "+
				"through to the whole bank", l)
			continue
		}
		counts := map[int]int{}
		prev, ordered := 0, true
		for _, id := range ids {
			q, ok := byID[id]
			if !ok {
				r.errorf(where, "set names unknown item %q (the server silently drops it)", id)
				continue
			}
			if q.Band == 0 {
				continue // reported above
			}
			counts[q.Band]++
			if q.Band < prev {
				ordered = false
			}
			prev = q.Band
		}
		if counts[l] < 3 {
			r.errorf(where, "%d item(s) of band %d in its own series, need >= 3 - a series "+
				"that only trims a ladder cannot size the learner within the level", counts[l], l)
		}
		if l > 1 && counts[l-1] < 1 {
			r.errorf(where, "no floor: nothing from band %d, so an overrated learner is never caught", l-1)
		}
		if l < 5 && counts[l+1] < 1 {
			r.errorf(where, "no ceiling: nothing from band %d, so an underrated learner is never caught", l+1)
		}
		if !ordered {
			r.errorf(where, "series is not ordered easy to hard by band")
		}
	}
}

func first(xs []string, n int) []string {
	if len(xs) > n {
		return xs[:n]
	}
	return xs
}

// Run lints a corpus and returns the report.
// checkTapOnly: placement and pretests are answered by tapping. A prose
// item there is what makes the intake feel like an exam.
func checkTapOnly(u *corpus.Unit, r *Report) {
	prose := func(q corpus.Question) bool {
		return q.Kind != "mcq" && (q.Check == "" || q.Check == "llm")
	}
	for _, q := range u.Questions.Pretest {
		if prose(q) {
			r.errorf(u.ID+"/"+q.ID, "pretest item answered in prose; pretests are taps (choice, numeric or exact)")
		}
	}
	if u.IsCalibration() {
		n := 0
		for _, q := range u.Questions.Check {
			if prose(q) {
				n++
			}
		}
		if n > 0 {
			r.errorf(u.ID, "%d prose items in a calibration unit; placement is answered by taps, "+
				"so every item should be mcq, numeric or exact", n)
		}
	}
}

func Run(c *corpus.Corpus) *Report {
	r := &Report{}
	for _, uid := range c.UnitOrder() {
		u, ok := c.Units[uid]
		if !ok {
			r.warnf(uid, "declared in syllabus but not authored")
			continue
		}
		checkNumericTolerances(u, r)
		checkAnswersMachineReadable(u, r)
		checkMCQs(u, c, r)
		checkSpecConformance(u, r)
		checkDepthHeadings(u, r)
		checkRefutationIDs(u, c, r)
		checkCalibrationSets(u, r)
	}
	return r
}
