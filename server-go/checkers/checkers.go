// Package checkers does mechanical answer checking. Anything a program can
// check never goes to the LLM.
//
// check spec grammar (from authoring-spec.md):
//
//	exact           - case/whitespace-insensitive string match
//	numeric(0.01)   - parse first number in the response, compare with tolerance
//	choice          - MCQ option index/letter match
//	llm             - not handled here (grader role)
package checkers

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

var (
	numericSpec = regexp.MustCompile(`^numeric\(([\d.eE+-]+)\)$`)
	// Accept "3/4"-style fractions as well as plain floats/scientific notation.
	number    = regexp.MustCompile(`-?\d+(?:\.\d+)?(?:[eE][+-]?\d+)?(?:\s*/\s*-?\d+(?:\.\d+)?)?`)
	spaces    = regexp.MustCompile(`\s+`)
	separator = regexp.MustCompile(`\s*([,;:])\s*`)
)

func IsMechanical(check string) bool { return check != "llm" }

func parseNumber(text string) (float64, bool) {
	token := number.FindString(strings.ReplaceAll(text, ",", ""))
	if token == "" {
		return 0, false
	}
	if strings.Contains(token, "/") {
		parts := strings.SplitN(token, "/", 2)
		num, err1 := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		den, err2 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if err1 != nil || err2 != nil || den == 0 {
			return 0, false
		}
		return num / den, true
	}
	v, err := strconv.ParseFloat(token, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

// CheckAnswer returns pass/fail for a mechanical check. It errors only when
// handed a check it does not implement, which is a corpus bug, not a wrong
// answer - grading such an item as "fail" would blame the learner for it.
func CheckAnswer(check string, expected, given string) (bool, error) {
	switch {
	case check == "exact", check == "choice":
		// Whitespace inside an exact answer is never the meaning: "4x7" and
		// "4 x 7" are the same shape, "w|i|d|est_" and "w | i | d | est_"
		// the same tokens. Compare with it collapsed, then with it gone.
		return Norm(given) == Norm(expected) || Compact(given) == Compact(expected), nil
	}
	if m := numericSpec.FindStringSubmatch(strings.TrimSpace(check)); m != nil {
		tol, err := strconv.ParseFloat(m[1], 64)
		if err != nil {
			return false, fmt.Errorf("unparseable tolerance in %q", check)
		}
		exp, okE := parseNumber(expected)
		got, okG := parseNumber(given)
		if !okE || !okG {
			return false, nil
		}
		// Absolute tolerance, which is what `numeric(0.01)` reads as and what
		// the authoring spec documents. An earlier max(tol, |exp|*tol) rule
		// made the tolerance *relative* at the same magnitude, so numeric(1)
		// silently meant +/-100% and marked an answer twice the correct one as
		// right. The tiny relative term only absorbs float representation
		// error on large values.
		return math.Abs(got-exp) <= tol+math.Abs(exp)*1e-9, nil
	}
	return false, fmt.Errorf("not a mechanical check: %s", check)
}

// Option is one MCQ choice as the corpus stores it.
type Option struct {
	Text          string `yaml:"text" json:"text"`
	Correct       bool   `yaml:"correct" json:"correct,omitempty"`
	Explain       string `yaml:"explain" json:"explain,omitempty"`
	Misconception string `yaml:"misconception" json:"misconception,omitempty"`
}

// MCQGrade is the verdict plus the misconception diagnosis.
type MCQGrade struct {
	Verdict        string   `json:"verdict"`
	Misconceptions []string `json:"misconceptions"`
	Explain        string   `json:"explain"`
}

func CheckMCQ(options []Option, selectedIndex int) MCQGrade {
	if selectedIndex < 0 || selectedIndex >= len(options) {
		return MCQGrade{Verdict: "fail", Misconceptions: []string{}, Explain: "no option selected"}
	}
	opt := options[selectedIndex]
	if opt.Correct {
		return MCQGrade{Verdict: "pass", Misconceptions: []string{}, Explain: opt.Explain}
	}
	out := MCQGrade{Verdict: "fail", Misconceptions: []string{}, Explain: opt.Explain}
	if opt.Misconception != "" {
		out.Misconceptions = []string{opt.Misconception}
	}
	return out
}

// Norm normalises for exact comparison.
//
// Multi-value answers ("4, 1; 11, 6") are the common exact-match case, and a
// learner who types "4,1;11,6" has the right answer. Spacing around separators
// carries no meaning here, so collapse it rather than failing someone on
// punctuation style.
func Norm(s string) string {
	text := strings.ToLower(strings.TrimSpace(spaces.ReplaceAllString(s, " ")))
	return separator.ReplaceAllString(text, "$1")
}

// Compact is the exact-match fallback: lower case with every space gone.
func Compact(s string) string {
	return strings.ToLower(spaces.ReplaceAllString(s, ""))
}
