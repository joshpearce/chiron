package checkers

import "testing"

// The mechanical grading contract. These cases are duplicated in the offline
// JS grader (ipad-app/Chiron/Resources/book.js, gradeMechanical/norm) and in
// the Python implementation's test_checkers.py. All three must agree: a beat is
// graded in JavaScript while reading offline and on the server at the chapter
// boundary, so any divergence means the same answer is right in one place and
// wrong in the other.
var cases = []struct {
	check    string
	expected string
	given    string
	want     bool
	why      string
}{
	{"numeric(1)", "50331648", "50331648", true, "exact hit"},
	{"numeric(1)", "50331648", "100663296", false,
		"double the answer - the old max(tol, |exp|*tol) rule made numeric(1) mean +/-100% and passed this"},
	{"numeric(1)", "50331648", "50331649", true, "within the intended +/-1"},
	{"numeric(0.001)", "0.50349", "0.503", true, "rounded to three places"},
	{"numeric(0.001)", "0.50349", "0.5", false,
		"the naive symmetry guess an attention beat exists to catch"},
	{"numeric(0.01)", "5.5", "-5.5", false, "sign flip"},
	{"exact", "4, 1; 11, 6", "4,1;11,6", true, "separator spacing is not meaning"},
	{"exact", "4, 1; 11, 6", " 4 , 1 ; 11 , 6 ", true, "padded separators"},
	{"exact", "4, 1; 11, 6", "4, 1; 11, 7", false, "one entry wrong"},
	{"exact", "4 x 7", "4 X 7", true, "case-insensitive"},
	{"exact", "4 x 7", "4x7", true, "spacing inside an expression is not meaning"},
	{"exact", "4 x 7", "4 x7", true, "uneven spacing"},
	{"exact", "w|i|d|est_", "w | i | d | est_", true, "spaces around pipes"},
	{"exact", "4 x 7", "47", false, "the operator is meaning"},
	{"exact", "4 x 7", "7 x 4", false, "order is meaning"},
	{"numeric(0.01)", "6", "the answer is 6", true, "prose around a number still parses"},
}

func TestCheckAnswer(t *testing.T) {
	for _, c := range cases {
		got, err := CheckAnswer(c.check, c.expected, c.given)
		if err != nil {
			t.Errorf("%s expected=%q given=%q: unexpected error %v", c.check, c.expected, c.given, err)
			continue
		}
		if got != c.want {
			t.Errorf("%s expected=%q given=%q -> %v, wanted %v (%s)",
				c.check, c.expected, c.given, got, c.want, c.why)
		}
	}
}

func TestUnknownCheckErrorsRatherThanFailingTheLearner(t *testing.T) {
	if _, err := CheckAnswer("wat", "1", "1"); err == nil {
		t.Error("an unimplemented check must error, not silently grade")
	}
}

func TestCheckMCQ(t *testing.T) {
	opts := []Option{
		{Text: "right", Correct: true, Explain: "because"},
		{Text: "wrong", Explain: "diagnosis", Misconception: "M2"},
	}
	if g := CheckMCQ(opts, 0); g.Verdict != "pass" || len(g.Misconceptions) != 0 {
		t.Errorf("correct option graded %+v", g)
	}
	g := CheckMCQ(opts, 1)
	if g.Verdict != "fail" || len(g.Misconceptions) != 1 || g.Misconceptions[0] != "M2" {
		t.Errorf("distractor must carry its misconception, got %+v", g)
	}
	if g.Explain != "diagnosis" {
		t.Errorf("a wrong answer must be a diagnosis, got explain=%q", g.Explain)
	}
	for _, i := range []int{-1, 2} {
		if g := CheckMCQ(opts, i); g.Verdict != "fail" {
			t.Errorf("out-of-range index %d graded %+v", i, g)
		}
	}
}
