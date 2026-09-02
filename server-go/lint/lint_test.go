package lint

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/mjbraun/chiron/server/checkers"
	"github.com/mjbraun/chiron/server/corpus"
)

// The linter has to agree with the grader. A tolerance the linter passes but
// the grader treats as accepting a doubled answer is worse than no linter.
func TestToleranceCheckMirrorsTheGrader(t *testing.T) {
	cases := []struct {
		check   string
		answer  string
		flagged bool
		why     string
	}{
		// This is the item the tolerance bug was found on. Under the old
		// max(tol, |exp|*tol) rule the window was +/-5e7 and it accepted a
		// doubled answer; under the absolute rule the window is +/-1.05 and it
		// is genuinely fine. The linter agreeing with the grader here is the
		// point - both had to change together.
		{"numeric(1)", "50331648", false, "absolute +/-1 is nowhere near 1e8"},
		{"numeric(0.001)", "0.50349", false, "tight enough to exclude every plausible slip"},
		{"numeric(10)", "12", true, "+/-10 on 12 reaches 6 and 24 - the halved and doubled answers"},
		{"numeric(3)", "5", true, "+/-3 reaches 2.5, the halved answer"},
		{"numeric(0.01)", "5.5", false, "well clear of 11, 2.75 and -5.5"},
	}
	for _, c := range cases {
		u := &corpus.Unit{
			ID: "u0",
			Questions: corpus.QuestionFile{Check: []corpus.Question{{
				ID: "q1", Check: c.check, Answer: corpus.Scalar(c.answer),
			}}},
		}
		r := &Report{}
		checkNumericTolerances(u, r)
		got := len(r.Errors) > 0
		if got != c.flagged {
			t.Errorf("%s answer=%s flagged=%v, want %v (%s)\n  %v",
				c.check, c.answer, got, c.flagged, c.why, r.Errors)
			continue
		}
		// Whatever the linter says, the grader must still accept the reference
		// answer as its own correct answer.
		ok, err := checkers.CheckAnswer(c.check, c.answer, c.answer)
		if err != nil || !ok {
			t.Errorf("%s: reference answer does not satisfy its own check", c.check)
		}
	}
}

// 0xB2,0x00,0x2F is exactly what check: exact is for. The machine-answer
// pattern once predated any corpus containing hex and rejected it as prose.
func TestMachineAnswerAcceptsRealAnswers(t *testing.T) {
	for _, ok := range []string{
		"249", "-4", "[7, -4]", "4 x 7", "w|i|d|est_", "0xB2,0x00,0x2F",
		"4,7,6|2,7,10|1|2|4", "0,0|9,1467|10",
		"4, 1; 11, 6", "1.5e-3",
	} {
		if !isMachineAnswer(ok) {
			t.Errorf("rejected a real answer: %q", ok)
		}
	}
	for _, prose := range []string{
		"It is the number of parameters in the model, roughly speaking, once you count embeddings",
		"",
	} {
		if isMachineAnswer(prose) {
			t.Errorf("accepted prose as a machine answer: %q", prose)
		}
	}
}

// An MCQ without check: choice is not graded as an MCQ at all, and a
// distractor with no misconception is a miss rather than a diagnosis.
func TestMCQIntegrity(t *testing.T) {
	c := &corpus.Corpus{Misconceptions: map[string]corpus.Misconception{
		"M1": {ID: "M1"},
	}}
	u := &corpus.Unit{ID: "u0", Questions: corpus.QuestionFile{Check: []corpus.Question{{
		ID: "q1", Kind: "mcq", Check: "", // missing check: choice
		Options: []checkers.Option{
			{Text: "a", Correct: true, Explain: "yes"},
			{Text: "b", Explain: "no"},                       // no misconception
			{Text: "c", Misconception: "M99", Explain: "no"}, // unknown id
		},
	}}}}
	r := &Report{}
	checkMCQs(u, c, r)
	joined := strings.Join(r.Errors, "\n")
	for _, want := range []string{"check: choice", "no misconception id", "unknown misconception"} {
		if !strings.Contains(joined, want) {
			t.Errorf("missing a finding for %q, got:\n%s", want, joined)
		}
	}
}

// The real corpus must stay clean, and this is what proves the port agrees with
// the Python linter it replaces.
func TestRealCorpusLintsClean(t *testing.T) {
	c, err := corpus.Load(filepath.Join("..", "..", "corpus"))
	if err != nil {
		t.Fatal(err)
	}
	r := Run(c)
	for _, e := range r.Errors {
		t.Errorf("corpus error: %s", e)
	}
	if len(r.Warnings) != 2 {
		t.Errorf("%d warnings, expected the 2 known large-integer ones: %v",
			len(r.Warnings), r.Warnings)
	}
}

// A section deliberately written without depth variants - a closing notation
// reference, a correction transplanted from another unit - is marked
// <!-- canon-only --> in canon. The missing-variant warning exists to catch
// ACCIDENTAL gaps, so a marked section must not trip it while an unmarked
// gap still does.
func TestCanonOnlySectionsExemptFromDepthWarning(t *testing.T) {
	u := &corpus.Unit{
		ID: "u1",
		Sections: []corpus.Section{
			{Heading: "Real teaching", Segments: []corpus.Segment{
				{Type: "prose", MD: "body"}}},
			{Heading: "Notation in this unit", Segments: []corpus.Segment{
				{Type: "prose", MD: "<!-- canon-only -->\n\nreference table"}}},
		},
		Depths: map[string]map[string]string{
			"more-intuition": {"Real teaching": strings.Repeat("v", 500)},
		},
	}
	r := &Report{}
	checkDepthHeadings(u, r)
	if len(r.Warnings) != 0 {
		t.Fatalf("marked canon-only section warned anyway: %v", r.Warnings)
	}

	// The same unit without the marker must still warn.
	u.Sections[1].Segments[0].MD = "reference table"
	r = &Report{}
	checkDepthHeadings(u, r)
	if len(r.Warnings) != 1 {
		t.Fatalf("unmarked missing variant: got %d warnings, want 1: %v",
			len(r.Warnings), r.Warnings)
	}
}

// A calibration set must size the learner WITHIN the level they claimed. The
// original u0 sets only ever trimmed from the top of one ladder, so "absolute
// novice" got the same ten dot-product and matrix-shape items as "could work
// through it slowly" - the screener was decorative. Every level's series is
// now a window: a floor from the band below, the bulk from the band itself, a
// ceiling from the band above, and the bank has to actually contain every band.
func calibrationUnit(bands map[string]int, sets map[int][]string) *corpus.Unit {
	u := &corpus.Unit{ID: "u0", Front: map[string]any{"calibration": true}}
	var ids []string
	for id := range bands {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		u.Questions.Check = append(u.Questions.Check, corpus.Question{
			ID: id, Band: bands[id], Kind: "constructed", Check: "exact", Answer: "1",
		})
	}
	u.Questions.CalibrationSets = sets
	return u
}

func TestCalibrationSetsAreWindowsAroundTheirLevel(t *testing.T) {
	// Three items per band, named b<band>-<n>.
	bank := map[string]int{}
	for b := 1; b <= 5; b++ {
		for n := 1; n <= 3; n++ {
			bank[fmt.Sprintf("b%d-%d", b, n)] = b
		}
	}
	band := func(b int) []string {
		return []string{fmt.Sprintf("b%d-1", b), fmt.Sprintf("b%d-2", b), fmt.Sprintf("b%d-3", b)}
	}
	window := func(l int) []string {
		var ids []string
		if l > 1 {
			ids = append(ids, band(l - 1)[0])
		}
		ids = append(ids, band(l)...)
		if l < 5 {
			ids = append(ids, band(l + 1)[0])
		}
		return ids
	}
	good := map[int][]string{}
	for l := 1; l <= 5; l++ {
		good[l] = window(l)
	}
	r := &Report{}
	checkCalibrationSets(calibrationUnit(bank, good), r)
	if len(r.Errors) > 0 {
		t.Fatalf("well-formed windows flagged: %v", r.Errors)
	}

	cases := []struct {
		name   string
		mutate func(bank map[string]int, sets map[int][]string)
		want   string
	}{
		{"level with no items of its own band", func(bank map[string]int, sets map[int][]string) {
			sets[1] = band(2) // a novice handed only band-2 items: the original bug
		}, "band 1"},
		{"ladder that only trims from the top", func(bank map[string]int, sets map[int][]string) {
			sets[1] = append(band(3), band(4)...)
		}, "band 1"},
		{"missing level", func(bank map[string]int, sets map[int][]string) {
			delete(sets, 3)
		}, "level 3"},
		{"unknown item id", func(bank map[string]int, sets map[int][]string) {
			sets[2] = append(sets[2], "b9-9")
		}, "b9-9"},
		{"item without a band", func(bank map[string]int, sets map[int][]string) {
			bank["b2-1"] = 0
		}, "b2-1"},
		{"band absent from the bank", func(bank map[string]int, sets map[int][]string) {
			for n := 1; n <= 3; n++ {
				bank[fmt.Sprintf("b5-%d", n)] = 4
			}
		}, "band 5"},
		{"no ceiling probe", func(bank map[string]int, sets map[int][]string) {
			sets[2] = append([]string{band(1)[0]}, band(2)...)
		}, "band 3"},
		{"no floor", func(bank map[string]int, sets map[int][]string) {
			sets[4] = append(band(4), band(5)[0])
		}, "band 3"},
		{"out of order", func(bank map[string]int, sets map[int][]string) {
			sets[3] = []string{band(4)[0], band(2)[0], band(3)[0], band(3)[1], band(3)[2]}
		}, "easy to hard"},
	}
	for _, c := range cases {
		bank2 := map[string]int{}
		for k, v := range bank {
			bank2[k] = v
		}
		sets2 := map[int][]string{}
		for k, v := range good {
			sets2[k] = append([]string(nil), v...)
		}
		c.mutate(bank2, sets2)
		r := &Report{}
		checkCalibrationSets(calibrationUnit(bank2, sets2), r)
		if len(r.Errors) == 0 {
			t.Errorf("%s: not flagged", c.name)
			continue
		}
		if !strings.Contains(strings.Join(r.Errors, "\n"), c.want) {
			t.Errorf("%s: errors do not mention %q: %v", c.name, c.want, r.Errors)
		}
	}

	// A teaching unit has no sets and no bands; none of this applies.
	plain := calibrationUnit(bank, nil)
	plain.Front = nil
	r = &Report{}
	checkCalibrationSets(plain, r)
	if len(r.Errors) > 0 {
		t.Errorf("non-calibration unit flagged: %v", r.Errors)
	}
}
