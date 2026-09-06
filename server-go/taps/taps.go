// Package taps rewrites the items of a unit's question bank that a reader
// would have to answer in prose into items answered by a tap: a multiple
// choice question whose distractors carry the misconceptions the rubric
// used to diagnose, or a constructed item whose answer is a number or a
// term. The rest of the file is left byte for byte as it was.
package taps

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/mjbraun/chiron/server/llm"
	"gopkg.in/yaml.v3"
)

// Item is one entry of a pretest or check list as it sits in the file.
type Item struct {
	ID   string
	YAML string // the block at zero indent, ending in a newline
	// indent is the list item's own indent in the file.
	indent string
	start  int // first line
	end    int // one past the last line
}

var (
	itemStart = regexp.MustCompile(`^(\s*)- id:\s*(\S+)\s*$`)
	topKey    = regexp.MustCompile(`^[a-z_]+:`)
	checkLine = regexp.MustCompile(`(?m)^\s*check:\s*(\S+)`)
	kindLine  = regexp.MustCompile(`(?m)^\s*kind:\s*(\S+)`)
)

// items lists every list item under the pretest and check keys.
func items(text string) []Item {
	lines := strings.Split(text, "\n")
	var out []Item
	section := ""
	for i := 0; i < len(lines); i++ {
		if topKey.MatchString(lines[i]) {
			section = strings.TrimSuffix(topKey.FindString(lines[i]), ":")
			continue
		}
		if section != "pretest" && section != "check" {
			continue
		}
		m := itemStart.FindStringSubmatch(lines[i])
		if m == nil {
			continue
		}
		indent := m[1]
		end := i + 1
		for end < len(lines) {
			if topKey.MatchString(lines[end]) {
				break
			}
			if n := itemStart.FindStringSubmatch(lines[end]); n != nil && n[1] == indent {
				break
			}
			end++
		}
		// Trailing blank lines belong to the file, not the item.
		last := end
		for last > i+1 && strings.TrimSpace(lines[last-1]) == "" {
			last--
		}
		var block []string
		for _, l := range lines[i:last] {
			block = append(block, strings.TrimPrefix(l, indent))
		}
		out = append(out, Item{ID: m[2], YAML: strings.Join(block, "\n") + "\n", indent: indent, start: i, end: last})
	}
	return out
}

// ProseItems returns the items a reader answers in prose: check is llm,
// or absent on an item that is not a multiple choice question.
func ProseItems(text string) []Item {
	var out []Item
	for _, it := range items(text) {
		check, kind := "", ""
		if m := checkLine.FindStringSubmatch(it.YAML); m != nil {
			check = m[1]
		}
		if m := kindLine.FindStringSubmatch(it.YAML); m != nil {
			kind = m[1]
		}
		if check == "llm" || (check == "" && kind != "mcq") {
			out = append(out, it)
		}
	}
	return out
}

// Splice replaces items by id with new blocks (given at zero indent) and
// returns the file with everything else untouched.
func Splice(text string, replacements map[string]string) (string, error) {
	byID := map[string]Item{}
	for _, it := range items(text) {
		byID[it.ID] = it
	}
	lines := strings.Split(text, "\n")
	type edit struct {
		item  Item
		block []string
	}
	var edits []edit
	for id, block := range replacements {
		it, ok := byID[id]
		if !ok {
			return "", fmt.Errorf("no item %s in the file", id)
		}
		m := itemStart.FindStringSubmatch(strings.SplitN(block, "\n", 2)[0])
		if m == nil || m[2] != id {
			return "", fmt.Errorf("replacement for %s does not start with its id line: %q", id, strings.SplitN(block, "\n", 2)[0])
		}
		var indented []string
		for _, l := range strings.Split(strings.TrimRight(block, "\n"), "\n") {
			if strings.TrimSpace(l) == "" {
				indented = append(indented, "")
				continue
			}
			indented = append(indented, it.indent+l)
		}
		edits = append(edits, edit{it, indented})
	}
	// Apply from the bottom so earlier line numbers stay valid.
	for i := 0; i < len(edits); i++ {
		for j := i + 1; j < len(edits); j++ {
			if edits[j].item.start > edits[i].item.start {
				edits[i], edits[j] = edits[j], edits[i]
			}
		}
	}
	for _, e := range edits {
		lines = append(lines[:e.item.start], append(e.block, lines[e.item.end:]...)...)
	}
	return strings.Join(lines, "\n"), nil
}

const system = `You are the author of an adaptive textbook, rewriting question bank items so a reader can answer them by tapping instead of typing prose. You follow the authoring contract exactly and return YAML that parses.`

const instructions = `Rewrite each item below as an item answered by a tap, keeping its id, concept, band, difficulty, callback_eligible and congruent fields as they are:

- Prefer ` + "`kind: mcq`" + ` with ` + "`check: choice`" + `: exactly one correct option, three or four distractors, every option with an ` + "`explain`" + `, and every distractor keyed to a misconception id from the banks given (use ` + "`misconception:`" + `). Turn what the old rubric diagnosed ("answering X indicates M") into the distractors, so the item still catches the same errors. A distractor must be a plausible answer a reader holding that misconception would pick, not a joke.
- Where the question has a single short answer, a constructed item is also fine: ` + "`kind: constructed`" + ` with ` + "`check: numeric(tol)`" + ` and a number as the answer, or ` + "`check: exact`" + ` with a single term. Never a sentence as the answer.
- The prompt must ask for a choice, a number or a term - never "explain", "describe" or "derive".
- No ` + "`rubric`" + ` and no ` + "`check: llm`" + ` in the result.
- Keep the mathematics and the notation of the original; keep the item at the same band and difficulty.

Return each rewritten item as one YAML list item starting with "- id: <id>" at zero indent with two-space fields, exactly as the schema in the contract shows, and nothing else in that string.`

// questionsContract is the part of the authoring spec that governs the
// question bank.
func questionsContract(spec string) string {
	i := strings.Index(spec, "## questions.yaml schema")
	if i < 0 {
		return spec
	}
	rest := spec[i:]
	if j := strings.Index(rest[1:], "\n## "); j >= 0 {
		rest = rest[:j+1]
	}
	return rest
}

// Rewrite rewrites the prose items of the unit at dir and returns their ids.
// bankPath and specPath are the corpus's misconception bank and authoring
// spec. Nothing is written when there is nothing to rewrite.
func Rewrite(chain llm.Chain, dir, bankPath, specPath string, batch int) ([]string, error) {
	qpath := filepath.Join(dir, "questions.yaml")
	raw, err := os.ReadFile(qpath)
	if err != nil {
		return nil, err
	}
	text := string(raw)
	prose := ProseItems(text)
	if len(prose) == 0 {
		return nil, nil
	}
	canon, _ := os.ReadFile(filepath.Join(dir, "canon.md"))
	local, _ := os.ReadFile(filepath.Join(dir, "misconceptions.yaml"))
	bank, err := os.ReadFile(bankPath)
	if err != nil {
		return nil, err
	}
	spec, err := os.ReadFile(specPath)
	if err != nil {
		return nil, err
	}
	context := fmt.Sprintf("AUTHORING CONTRACT (the question bank part):\n%s\n\nMISCONCEPTION BANK:\n%s\n\nUNIT-LOCAL MISCONCEPTIONS:\n%s\n\nTHE UNIT'S canon.md:\n%s",
		questionsContract(string(spec)), bank, local, canon)
	if batch <= 0 {
		batch = 6
	}
	replacements := map[string]string{}
	for i := 0; i < len(prose); i += batch {
		end := min(i+batch, len(prose))
		var blocks []string
		for _, it := range prose[i:end] {
			blocks = append(blocks, it.YAML)
		}
		user := context + "\n\n" + instructions + "\n\nITEMS TO REWRITE:\n\n" + strings.Join(blocks, "\n")
		var out struct {
			Items []struct {
				ID   string `json:"id"`
				YAML string `json:"yaml"`
			} `json:"items"`
		}
		if err := chain.Structured("author", system, user, schema(), "rewritten_items", &out); err != nil {
			return nil, err
		}
		for _, r := range out.Items {
			if err := validate(r.ID, r.YAML); err != nil {
				return nil, err
			}
			replacements[r.ID] = strings.TrimRight(r.YAML, "\n") + "\n"
		}
		for _, it := range prose[i:end] {
			if _, ok := replacements[it.ID]; !ok {
				return nil, fmt.Errorf("the model did not return %s", it.ID)
			}
		}
	}
	spliced, err := Splice(text, replacements)
	if err != nil {
		return nil, err
	}
	var whole map[string]any
	if err := yaml.Unmarshal([]byte(spliced), &whole); err != nil {
		return nil, fmt.Errorf("the rewritten file does not parse: %w", err)
	}
	if err := os.WriteFile(qpath, []byte(spliced), 0o644); err != nil {
		return nil, err
	}
	var ids []string
	for _, it := range prose {
		ids = append(ids, it.ID)
	}
	return ids, nil
}

func schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"items": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"id":   map[string]any{"type": "string"},
						"yaml": map[string]any{"type": "string", "description": "one YAML list item, `- id: ...` at zero indent"},
					},
					"required": []string{"id", "yaml"},
				},
			},
		},
		"required": []string{"items"},
	}
}

// validate: the block parses as one item with that id, answered by a tap.
func validate(id, block string) error {
	var list []struct {
		ID      string `yaml:"id"`
		Kind    string `yaml:"kind"`
		Check   string `yaml:"check"`
		Rubric  string `yaml:"rubric"`
		Options []struct {
			Correct       bool   `yaml:"correct"`
			Misconception string `yaml:"misconception"`
			Explain       string `yaml:"explain"`
		} `yaml:"options"`
	}
	if err := yaml.Unmarshal([]byte(block), &list); err != nil {
		return fmt.Errorf("%s: rewritten item does not parse: %w", id, err)
	}
	if len(list) != 1 || list[0].ID != id {
		return fmt.Errorf("%s: rewritten block is not one item with that id", id)
	}
	it := list[0]
	if it.Rubric != "" || it.Check == "llm" || it.Check == "" {
		return fmt.Errorf("%s: still answered in prose (check %q)", id, it.Check)
	}
	if it.Kind == "mcq" && it.Check != "choice" {
		return fmt.Errorf("%s: mcq with check %q", id, it.Check)
	}
	// A pretest item has no kind; its options make it a choice.
	if it.Check == "choice" {
		correct := 0
		for _, o := range it.Options {
			if o.Correct {
				correct++
			} else if o.Misconception == "" {
				return fmt.Errorf("%s: a distractor has no misconception id", id)
			}
			if o.Explain == "" {
				return fmt.Errorf("%s: an option has no explain", id)
			}
		}
		if correct != 1 || len(it.Options) < 3 {
			return fmt.Errorf("%s: %d correct options of %d", id, correct, len(it.Options))
		}
	} else if it.Check != "exact" && !strings.HasPrefix(it.Check, "numeric") {
		return fmt.Errorf("%s: constructed item with check %q", id, it.Check)
	}
	return nil
}
