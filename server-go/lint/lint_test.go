package lint

import (
	"path/filepath"
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
