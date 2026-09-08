package generate

import (
	"os"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

type bank struct {
	Misconceptions []map[string]any `yaml:"misconceptions"`
}

// A misconception file as the author wrote it for the first PDF-built
// book: prose values opening with a quotation mark and holding colons,
// which plain scalars cannot carry. Repaired, it parses, and every
// sentence is still there.
func TestRepairMakesTheAuthorsProseParse(t *testing.T) {
	raw, err := os.ReadFile("testdata/misconceptions-broken.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var probe bank
	if yaml.Unmarshal(raw, &probe) == nil {
		t.Fatal("the fixture parses already; it should not")
	}
	fixed := RepairYAMLProse(string(raw))
	var b bank
	if err := yaml.Unmarshal([]byte(fixed), &b); err != nil {
		t.Fatalf("repaired file does not parse: %v\n%s", err, fixed)
	}
	if len(b.Misconceptions) < 4 {
		t.Fatalf("only %d misconceptions survived", len(b.Misconceptions))
	}
	m4 := b.Misconceptions[3]
	if m4["id"] != "u0-m4" || m4["name"] != "default-rate-is-loss-rate" {
		t.Errorf("m4 = %v", m4)
	}
	why, _ := m4["why_appealing"].(string)
	if !strings.HasPrefix(why, "'Default' is the event everyone tracks") || !strings.Contains(why, "recovery assumption at all.") {
		t.Errorf("why_appealing = %q", why)
	}
	units, _ := m4["units"].([]any)
	if len(units) != 1 || units[0] != "u0" {
		t.Errorf("units = %v", m4["units"])
	}
	if RepairYAMLProse(fixed) != fixed {
		t.Error("a file that parses is not left alone")
	}
}

func TestRepairHandlesColonsAndEscapedQuotes(t *testing.T) {
	raw := "misconceptions:\n- id: x\n  name: a-name\n  correction: The retreat is structural, not a verdict: rules changed,\n    - de-smooth the volatility first,\n    and \\\"Default\\\" is a word.\n  units:\n  - u1\n"
	fixed := RepairYAMLProse(raw)
	var b bank
	if err := yaml.Unmarshal([]byte(fixed), &b); err != nil {
		t.Fatalf("%v\n%s", err, fixed)
	}
	c, _ := b.Misconceptions[0]["correction"].(string)
	if !strings.Contains(c, "not a verdict: rules changed,\n- de-smooth the volatility first,\nand \"Default\" is a word.") {
		t.Errorf("correction = %q", c)
	}
	if b.Misconceptions[0]["name"] != "a-name" {
		t.Errorf("name touched: %v", b.Misconceptions[0]["name"])
	}
}

func TestLocalMisconceptionIDsAreLowercased(t *testing.T) {
	in := "misconceptions:\n- id: U11-M1\n  units:\n  - u11\nchecks:\n- misconception: u11-m1\n- misconception: M15\n- text: see [[U11-M2]] and u11-m3, not the U-turn."
	got := NormaliseLocalIDs(in)
	want := "misconceptions:\n- id: u11-m1\n  units:\n  - u11\nchecks:\n- misconception: u11-m1\n- misconception: M15\n- text: see [[u11-m2]] and u11-m3, not the U-turn."
	if got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
}
