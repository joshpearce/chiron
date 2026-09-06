package taps

import (
	"strings"
	"testing"
)

const topLevel = `# bank
pretest: []
check:
- id: u0-q1
  kind: constructed
  prompt: one
  check: numeric(0.01)
- id: u0-q2
  kind: constructed
  prompt: |
    two
  check: llm
  rubric: |
    r
- id: u0-q3
  kind: mcq
  prompt: three
  check: choice
screener:
  id: u0-s1
  check: screener
calibration_sets:
  1: [u0-q1]
`

const indented = `pretest:
  - id: u0-p1
    prompt: p
    check: llm
check:
  - id: u0-q1
    kind: constructed
    prompt: one
  - id: u0-q2
    kind: constructed
    prompt: two
    check: exact
`

func TestProseItemsFindsEveryItemAnsweredInProse(t *testing.T) {
	got := ProseItems(topLevel)
	if len(got) != 1 || got[0].ID != "u0-q2" {
		t.Fatalf("top-level: %+v", got)
	}
	if !strings.Contains(got[0].YAML, "rubric: |") || strings.Contains(got[0].YAML, "u0-q3") {
		t.Fatalf("block is not the whole item: %q", got[0].YAML)
	}
	got = ProseItems(indented)
	ids := []string{}
	for _, it := range got {
		ids = append(ids, it.ID)
	}
	// u0-q1 has no check line and is constructed: graded in prose by default.
	if strings.Join(ids, ",") != "u0-p1,u0-q1" {
		t.Fatalf("indented: %v", ids)
	}
}

func TestSpliceReplacesBlocksAtTheirOwnIndent(t *testing.T) {
	out, err := Splice(topLevel, map[string]string{
		"u0-q2": "- id: u0-q2\n  kind: mcq\n  prompt: two?\n  check: choice\n",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Replace(topLevel,
		"- id: u0-q2\n  kind: constructed\n  prompt: |\n    two\n  check: llm\n  rubric: |\n    r\n",
		"- id: u0-q2\n  kind: mcq\n  prompt: two?\n  check: choice\n", 1)
	if out != want {
		t.Fatalf("got:\n%s\nwant:\n%s", out, want)
	}
	out, err = Splice(indented, map[string]string{
		"u0-p1": "- id: u0-p1\n  prompt: p?\n  check: exact\n",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "pretest:\n  - id: u0-p1\n    prompt: p?\n    check: exact\ncheck:\n") {
		t.Fatalf("indent not kept:\n%s", out)
	}
}

func TestSpliceRefusesAReplacementForAnotherItem(t *testing.T) {
	if _, err := Splice(topLevel, map[string]string{"u0-q2": "- id: u0-q9\n  prompt: x\n"}); err == nil {
		t.Fatal("expected an error for a mismatched id")
	}
	if _, err := Splice(topLevel, map[string]string{"u0-q7": "- id: u0-q7\n  prompt: x\n"}); err == nil {
		t.Fatal("expected an error for an unknown id")
	}
}

func TestValidateAcceptsAPretestChoiceWithoutAKind(t *testing.T) {
	block := "- id: u0-p1\n  prompt: which?\n  check: choice\n  options:\n" +
		"    - text: a\n      correct: true\n      explain: yes\n" +
		"    - text: b\n      misconception: M1\n      explain: no\n" +
		"    - text: c\n      misconception: M2\n      explain: no\n"
	if err := validate("u0-p1", block); err != nil {
		t.Fatal(err)
	}
	if err := validate("u0-p1", "- id: u0-p1\n  prompt: p\n  check: llm\n"); err == nil {
		t.Fatal("prose item accepted")
	}
}
