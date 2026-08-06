package corpus

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/mjbraun/chiron/server/checkers"
)

// The parser is checked against the real authored corpus rather than a
// fixture: the things that break it (a beat fence with trailing whitespace, a
// heading containing a colon, LaTeX with braces) only occur in real prose.
func load(t *testing.T) *Corpus {
	t.Helper()
	c, err := Load(filepath.Join("..", "..", "corpus"))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	return c
}

func TestLoadsWholeCorpus(t *testing.T) {
	c := load(t)
	if got := len(c.Units); got != 11 {
		t.Errorf("loaded %d units, want 11", got)
	}
	if got := len(c.Misconceptions); got != 77 {
		t.Errorf("%d misconceptions, want 77 (bank + unit-local, bank wins on id clash)", got)
	}
	if len(c.UnitOrder()) != 11 {
		t.Errorf("syllabus order has %d entries", len(c.UnitOrder()))
	}
}

func TestSectionsAndBeatsParse(t *testing.T) {
	c := load(t)
	totalBeats, totalSections := 0, 0
	for _, id := range c.UnitOrder() {
		u := c.Units[id]
		if len(u.Sections) == 0 && !u.IsCalibration() {
			t.Errorf("%s parsed no sections", id)
		}
		totalSections += len(u.Sections)
		for _, b := range u.Beats() {
			totalBeats++
			if b.ID == "" {
				t.Errorf("%s has a beat with no id", id)
			}
			if b.Prompt == "" {
				t.Errorf("%s/%s has no prompt", id, b.ID)
			}
		}
	}
	if totalBeats < 60 {
		t.Errorf("%d beats across the corpus - the spec asks 6-10 per unit", totalBeats)
	}
	t.Logf("%d sections, %d beats", totalSections, totalBeats)
}

// Depth variants are swapped into canon by heading. A heading that does not
// match means the swap silently does nothing, which is invisible at runtime.
func TestDepthHeadingsMatchCanon(t *testing.T) {
	c := load(t)
	for _, id := range c.UnitOrder() {
		u := c.Units[id]
		canon := map[string]bool{}
		for _, s := range u.Sections {
			canon[s.Heading] = true
		}
		for name, secs := range u.Depths {
			for heading := range secs {
				if !canon[heading] {
					t.Errorf("%s/depths/%s: heading %q not in canon", id, name, heading)
				}
			}
		}
	}
}

func TestFindQuestionAndBeat(t *testing.T) {
	c := load(t)
	q, unit := c.FindQuestion("u1-q1")
	if q == nil {
		t.Fatal("u1-q1 not found")
	}
	if unit != "u1" {
		t.Errorf("u1-q1 resolved to unit %q", unit)
	}
	if _, _ = c.FindQuestion("nope-q999"); false {
		t.Fatal("unreachable")
	}
	if q, _ := c.FindQuestion("nope-q999"); q != nil {
		t.Error("a missing question must resolve to nil, not a zero value")
	}
}

// The bug this pins cost the entire offline book: one compute item answered
// with a bare number ("answer: 6"), the client typed the field as a string, and
// the whole 11-chapter bundle failed to decode with the app reporting only
// "No bundled book found".
//
// Authors write the answer in YAML; the app receives it inside a Reveal (for
// check items) or on the beat itself. Both of those are the payload boundary,
// so both are what this checks.
func TestNumericAnswersSerialiseAsStrings(t *testing.T) {
	var b Beat
	if err := yaml.Unmarshal([]byte("id: u2-b1\nprompt: compute it\nanswer: 6\n"), &b); err != nil {
		t.Fatalf("yaml: %v", err)
	}
	if b.Answer != "6" {
		t.Errorf("bare YAML number loaded as %q", b.Answer)
	}
	out, err := json.Marshal(b)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), `"answer":"6"`) {
		t.Errorf("beat encoded as %s - answer must be a JSON string", out)
	}

	var r Reveal
	if err := yaml.Unmarshal([]byte("answer: 6\n"), &r); err != nil {
		t.Fatalf("yaml: %v", err)
	}
	if out, err := json.Marshal(r); err != nil {
		t.Fatal(err)
	} else if !strings.Contains(string(out), `"answer":"6"`) {
		t.Errorf("reveal encoded as %s - answer must be a JSON string", out)
	}
}

func TestScalarNormalisesEveryYAMLScalar(t *testing.T) {
	for in, want := range map[string]string{
		"answer: 6":     "6",
		"answer: 6.5":   "6.5",
		"answer: '6'":   "6",
		"answer: 0x1F":  "31", // YAML parses this as an int; it must not become "0x1F"
		"answer: true":  "true",
		"answer:":       "",
		"answer: 1e3":   "1000",
		"answer: -0.25": "-0.25",
	} {
		var r Reveal
		if err := yaml.Unmarshal([]byte(in), &r); err != nil {
			t.Fatalf("%s: %v", in, err)
		}
		if string(r.Answer) != want {
			t.Errorf("%q -> %q, want %q", in, r.Answer, want)
		}
	}
}

// Every mechanically-checked item in the real corpus must have an answer a
// program can actually compare - the server grades these without an LLM.
func TestRealCorpusMechanicalAnswersAreGradeable(t *testing.T) {
	c := load(t)
	graded := 0
	for _, id := range c.UnitOrder() {
		u := c.Units[id]
		for _, q := range append(append([]Question{}, u.Questions.Pretest...), u.Questions.Check...) {
			if q.Check == "llm" || q.Check == "" || q.Kind == "mcq" {
				continue
			}
			ok, err := checkers.CheckAnswer(q.Check, q.Answer.String(), q.Answer.String())
			if err != nil {
				t.Errorf("%s/%s: %v", id, q.ID, err)
				continue
			}
			if !ok {
				t.Errorf("%s/%s: reference answer %q does not satisfy its own check %q",
					id, q.ID, q.Answer, q.Check)
			}
			graded++
		}
	}
	t.Logf("%d mechanically-checked items self-verify", graded)
}

func TestPartialCorpusIsTolerated(t *testing.T) {
	// corpus-crypto has 8 syllabus units and 1 authored, which is exactly the
	// mid-generation state the server has to survive.
	dir := filepath.Join("..", "..", "corpus-crypto")
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Skip("corpus-crypto not present")
	}
	c, err := Load(dir)
	if err != nil {
		t.Fatalf("a partly-authored corpus must load, got %v", err)
	}
	if len(c.Units) >= len(c.Syllabus.Units) {
		t.Skip("corpus-crypto is now fully authored")
	}
	if len(c.Units) == 0 {
		t.Error("no units loaded from a partly-authored corpus")
	}
}
