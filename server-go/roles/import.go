package roles

import (
	"fmt"
	"strings"

	"github.com/mjbraun/chiron/server/llm"
	"github.com/mjbraun/chiron/server/sources"
	"gopkg.in/yaml.v3"
)

// Imported is one fetched section's exercises, named so the item that
// comes from each can say where it came from.
type Imported struct {
	Source    string
	Locator   string
	Title     string
	Exercises []sources.Exercise
}

const importSystem = `You turn a textbook's exercises into items for an adaptive book's question bank. You follow the authoring contract's questions.yaml schema exactly and return YAML that parses. A reader answers on a phone: an item is answered by a tap unless nothing shorter than prose checks the skill.`

const importInstructions = `For each exercise below that has a solution, or whose answer follows with certainty from the exercise itself, write one item in the schema:

- ids run <unit>-e1, <unit>-e2, ... in the order given;
- ` + "`check: numeric(tol)`" + ` when the answer is a number (state a tolerance that accepts rounding and rejects the characteristic wrong answer), ` + "`check: exact`" + ` when it is a single term;
- ` + "`kind: mcq`" + ` with ` + "`check: choice`" + ` when the answer is a choice among alternatives or a short claim: one correct option, three or four distractors each keyed to a misconception id from the bank with ` + "`misconception:`" + `, every option with an ` + "`explain`" + `;
- ` + "`check: llm`" + ` with a rubric only when the exercise asks for a derivation or an explanation and nothing shorter checks it, and for at most a fifth of the items;
- ` + "`concept`" + ` from the unit's concepts, ` + "`difficulty`" + ` warmup, core or stretch, ` + "`congruent`" + ` and ` + "`callback_eligible`" + ` set honestly;
- keep the source's wording where it works and its numbers as they are; move notation to the unit's.

Skip an exercise that needs code to be run, a data file, a figure, or an answer the source does not give and you cannot derive. Return every item as one YAML list item starting with "- id: <id>" at zero indent with two-space fields, with the exercise's source label beside it.`

var importSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"items": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"source": map[string]any{"type": "string", "description": "the exercise's source label, as given"},
					"yaml":   map[string]any{"type": "string", "description": "one YAML list item, `- id: ...` at zero indent"},
				},
				"required": []string{"source", "yaml"},
			},
		},
	},
	"required": []string{"items"},
}

// ImportItems writes bank items from the exercises of the sources a unit
// adapts. The result is YAML list items ready to sit in questions.yaml's
// check list, each carrying a source line; an empty string when there was
// nothing to import.
func ImportItems(chain llm.Chain, unitID, unitYAML, contract, bank string, imported []Imported) (string, error) {
	var labelled []string
	for _, im := range imported {
		for _, ex := range im.Exercises {
			label := fmt.Sprintf("%s %s exercise %d", im.Source, im.Locator, ex.Number)
			entry := fmt.Sprintf("=== %s (%s) ===\nEXERCISE:\n%s\n", label, im.Title, ex.Prompt)
			if ex.Solution != "" {
				entry += "SOLUTION:\n" + ex.Solution + "\n"
			}
			labelled = append(labelled, entry)
		}
	}
	if len(labelled) == 0 {
		return "", nil
	}
	context := fmt.Sprintf("AUTHORING CONTRACT (the question bank part):\n%s\n\nMISCONCEPTION BANK (cite these ids):\n%s\n\nUNIT:\n%s\n\n%s",
		contract, bank, unitYAML, strings.ReplaceAll(importInstructions, "<unit>", unitID))
	const batch = 12
	var out []string
	for i := 0; i < len(labelled); i += batch {
		end := min(i+batch, len(labelled))
		var reply struct {
			Items []struct {
				Source string `json:"source"`
				YAML   string `json:"yaml"`
			} `json:"items"`
		}
		user := context + "\n\nEXERCISES:\n\n" + strings.Join(labelled[i:end], "\n")
		if err := chain.Structured("author", importSystem, user, importSchema, "imported_items", &reply); err != nil {
			return "", err
		}
		for _, it := range reply.Items {
			block, ok := importedItem(it.YAML, it.Source)
			if ok {
				out = append(out, block)
			}
		}
	}
	return strings.Join(out, ""), nil
}

// importedItem checks one returned block parses as an item with a prompt
// and a check, and pins its source line.
func importedItem(block, source string) (string, bool) {
	var list []struct {
		ID     string `yaml:"id"`
		Prompt string `yaml:"prompt"`
		Check  string `yaml:"check"`
		Kind   string `yaml:"kind"`
	}
	if err := yaml.Unmarshal([]byte(block), &list); err != nil || len(list) != 1 {
		return "", false
	}
	it := list[0]
	if it.ID == "" || it.Prompt == "" {
		return "", false
	}
	block = strings.TrimRight(block, "\n") + "\n"
	if it.Check == "" {
		if it.Kind != "mcq" {
			return "", false
		}
		block += "  check: choice\n"
	}
	if !strings.Contains(block, "\n  source:") {
		block += "  source: " + source + "\n"
	}
	return block, true
}
