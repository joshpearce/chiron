package corpus

import (
	"os"
	"testing"

	"github.com/mjbraun/chiron/server/checkers"
	"gopkg.in/yaml.v3"
)

func TestOptionsAreShuffledByIdAndStayPut(t *testing.T) {
	opts := []checkers.Option{{Text: "a", Correct: true}, {Text: "b"}, {Text: "c"}, {Text: "d"}}
	first := 0
	for _, id := range []string{"u1-q1", "u1-q2", "u1-q3", "u1-q4", "u1-q5", "u1-q6", "u1-q7", "u1-q8"} {
		got := Shuffled(id, opts)
		again := Shuffled(id, opts)
		if len(got) != 4 || !sameSet(got, opts) {
			t.Fatalf("%s: not a permutation: %+v", id, got)
		}
		for i := range got {
			if got[i] != again[i] {
				t.Fatalf("%s: two shuffles differ", id)
			}
		}
		if got[0].Correct {
			first++
		}
	}
	if first == 8 {
		t.Fatal("the correct option stayed first for every id")
	}
	if opts[0].Text != "a" {
		t.Fatal("the caller's slice was reordered")
	}
}

func sameSet(a, b []checkers.Option) bool {
	seen := map[string]int{}
	for _, o := range a {
		seen[o.Text]++
	}
	for _, o := range b {
		seen[o.Text]--
	}
	for _, n := range seen {
		if n != 0 {
			return false
		}
	}
	return true
}

func TestTheScreenerKeepsItsOrderAndBeatsShuffleLikeItems(t *testing.T) {
	c, err := Load("../../corpus")
	if err != nil {
		t.Fatal(err)
	}
	u0 := c.Units["u0"]
	if u0.Questions.Screener == nil || u0.Questions.Screener.Options[0].Text != "Absolute novice" {
		t.Fatalf("screener order changed: %+v", u0.Questions.Screener)
	}
	// The loaded order is the file's order shuffled by id.
	raw, _ := os.ReadFile("../../corpus/units/u0-calibration/questions.yaml")
	var file struct {
		Check []struct {
			ID      string            `yaml:"id"`
			Options []checkers.Option `yaml:"options"`
		} `yaml:"check"`
	}
	if err := yaml.Unmarshal(raw, &file); err != nil {
		t.Fatal(err)
	}
	checked := 0
	for _, fq := range file.Check {
		if len(fq.Options) < 2 {
			continue
		}
		want := Shuffled(fq.ID, fq.Options)
		var got []checkers.Option
		for _, q := range u0.Questions.Check {
			if q.ID == fq.ID {
				got = q.Options
			}
		}
		for i := range want {
			if want[i].Text != got[i].Text {
				t.Fatalf("%s: loaded order is not the file's order shuffled by id", fq.ID)
			}
		}
		checked++
	}
	if checked == 0 {
		t.Fatal("no mcq in u0 to check")
	}
}
