package roles

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/mjbraun/chiron/server/llm"
	"github.com/mjbraun/chiron/server/sources"
)

type importChain struct {
	asked string
	reply []map[string]string
}

func (c *importChain) Structured(role, system, user string, schema map[string]any, name string, out any) error {
	c.asked = user
	data, _ := json.Marshal(map[string]any{"items": c.reply})
	return json.Unmarshal(data, out)
}

func (importChain) Status() llm.Status { return llm.Status{Connected: true} }

func TestImportItemsTurnsExercisesIntoBankItems(t *testing.T) {
	chain := &importChain{reply: []map[string]string{
		{"source": "think-bayes chap02.ipynb exercise 1", "yaml": "- id: u1-e1\n  concept: c-bayes\n  kind: constructed\n  prompt: Two coins, one fair. P(heads)?\n  answer: '0.75'\n  check: numeric(0.01)\n  difficulty: core\n"},
		{"source": "think-bayes chap02.ipynb exercise 2", "yaml": "- id: u1-e2\n  prompt: no check here\n"},
	}}
	ex := []Imported{{Source: "think-bayes", Locator: "chap02.ipynb", Title: "Think Bayes 2e", Exercises: []sources.Exercise{
		{Number: 1, Prompt: "Two coins in a box, one fair.", Solution: "0.75"},
		{Number: 2, Prompt: "Needs a data file.", Solution: ""},
	}}}
	got, err := ImportItems(chain, "u1", "id: u1\ntitle: Bayes\n", "contract", "bank", ex)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(chain.asked, "think-bayes chap02.ipynb exercise 1") || !strings.Contains(chain.asked, "Two coins in a box") || !strings.Contains(chain.asked, "SOLUTION:\n0.75") {
		t.Fatalf("the importer did not see the exercises:\n%s", chain.asked)
	}
	if !strings.Contains(got, "- id: u1-e1\n") || !strings.Contains(got, "source: think-bayes chap02.ipynb exercise 1") {
		t.Fatalf("got:\n%s", got)
	}
	if strings.Contains(got, "u1-e2") {
		t.Fatalf("an item without a check was kept:\n%s", got)
	}
}

func TestImportItemsWithNothingToImport(t *testing.T) {
	got, err := ImportItems(&importChain{}, "u1", "", "", "", nil)
	if err != nil || got != "" {
		t.Fatalf("got %q, %v", got, err)
	}
}
