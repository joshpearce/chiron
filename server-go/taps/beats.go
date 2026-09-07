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

var beatFence = regexp.MustCompile("(?s)```beat[ \t]*\n(.*?)```")

type beat struct {
	ID      string `yaml:"id"`
	Type    string `yaml:"type"`
	Check   string `yaml:"check"`
	Rubric  string `yaml:"rubric"`
	Options []struct {
		Correct       bool   `yaml:"correct"`
		Misconception string `yaml:"misconception"`
		Explain       string `yaml:"explain"`
	} `yaml:"options"`
}

// proseBeats lists the fence bodies of the beats a reader would answer in
// prose: no options, and no mechanical check.
func proseBeats(canon string) map[string]string {
	out := map[string]string{}
	for _, m := range beatFence.FindAllStringSubmatch(canon, -1) {
		var b beat
		if yaml.Unmarshal([]byte(m[1]), &b) != nil || b.ID == "" {
			continue
		}
		if len(b.Options) == 0 && (b.Check == "" || b.Check == "llm") {
			out[b.ID] = m[1]
		}
	}
	return out
}

// spliceBeats replaces the bodies of the fences whose ids are given.
func spliceBeats(text string, bodies map[string]string) string {
	return beatFence.ReplaceAllStringFunc(text, func(match string) string {
		m := beatFence.FindStringSubmatch(match)
		var b beat
		if yaml.Unmarshal([]byte(m[1]), &b) != nil {
			return match
		}
		body, ok := bodies[b.ID]
		if !ok {
			return match
		}
		return "```beat\n" + strings.TrimRight(body, "\n") + "\n```"
	})
}

const beatSystem = `You are the author of an adaptive textbook, rewriting the interaction beats in a chapter so a reader answers each by tapping instead of typing prose. You follow the authoring contract exactly and return YAML that parses.`

const beatInstructions = `Rewrite each beat below as a choice, keeping its id, type and concept:

- ` + "`options`" + ` with exactly one ` + "`correct: true`" + ` and two or three distractors, every option with an ` + "`explain`" + `, every distractor keyed with ` + "`misconception:`" + ` to an id from the banks given. Turn what the old rubric diagnosed into the distractors: each is what a reader holding that misconception would predict, not a joke.
- The prompt keeps its place in the chapter's argument: a predict beat still asks the reader to commit before reading on, a completion beat's options are candidate fillings for its blanks, a self-explain beat's options are candidate explanations.
- ` + "`check: choice`" + `; no ` + "`answer`" + `, no ` + "`rubric`" + `.
- Keep the mathematics and notation of the original. Never write an option that refers to another option's position.

Return each rewritten beat as the YAML body that goes inside its fence (no fence markers), starting with "id: <id>".`

// RewriteBeats rewrites the prose beats of the unit at dir as choices, in
// canon.md and in every depth variant that carries the same beat, so a
// depth swap re-appends the rewritten beat and not the old one.
func RewriteBeats(chain llm.Chain, dir, bankPath, specPath string, batch int) ([]string, error) {
	canonPath := filepath.Join(dir, "canon.md")
	raw, err := os.ReadFile(canonPath)
	if err != nil {
		return nil, err
	}
	canon := string(raw)
	prose := proseBeats(canon)
	if len(prose) == 0 {
		return nil, nil
	}
	var ids []string
	for _, m := range beatFence.FindAllStringSubmatch(canon, -1) {
		var b beat
		if yaml.Unmarshal([]byte(m[1]), &b) == nil {
			if _, ok := prose[b.ID]; ok {
				ids = append(ids, b.ID)
			}
		}
	}
	local, _ := os.ReadFile(filepath.Join(dir, "misconceptions.yaml"))
	bank, err := os.ReadFile(bankPath)
	if err != nil {
		return nil, err
	}
	spec, err := os.ReadFile(specPath)
	if err != nil {
		return nil, err
	}
	context := fmt.Sprintf("AUTHORING CONTRACT (the beats part):\n%s\n\nMISCONCEPTION BANK:\n%s\n\nUNIT-LOCAL MISCONCEPTIONS:\n%s\n\nTHE CHAPTER (canon.md), for the argument each beat sits in:\n%s",
		beatsContract(string(spec)), bank, local, canon)
	if batch <= 0 {
		batch = 6
	}
	bodies := map[string]string{}
	for i := 0; i < len(ids); i += batch {
		end := min(i+batch, len(ids))
		var blocks []string
		for _, id := range ids[i:end] {
			blocks = append(blocks, prose[id])
		}
		user := context + "\n\n" + beatInstructions + "\n\nBEATS TO REWRITE:\n\n" + strings.Join(blocks, "\n---\n")
		var out struct {
			Beats []struct {
				ID   string `json:"id"`
				YAML string `json:"yaml"`
			} `json:"beats"`
		}
		if err := chain.Structured("author", beatSystem, user, beatSchema(), "rewritten_beats", &out); err != nil {
			return nil, err
		}
		for _, r := range out.Beats {
			if err := validateBeat(r.ID, r.YAML); err != nil {
				return nil, err
			}
			bodies[r.ID] = strings.TrimRight(r.YAML, "\n") + "\n"
		}
		for _, id := range ids[i:end] {
			if _, ok := bodies[id]; !ok {
				return nil, fmt.Errorf("the model did not return %s", id)
			}
		}
	}
	if err := os.WriteFile(canonPath, []byte(spliceBeats(canon, bodies)), 0o644); err != nil {
		return nil, err
	}
	depths, _ := filepath.Glob(filepath.Join(dir, "depths", "*.md"))
	for _, path := range depths {
		text, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		if spliced := spliceBeats(string(text), bodies); spliced != string(text) {
			if err := os.WriteFile(path, []byte(spliced), 0o644); err != nil {
				return nil, err
			}
		}
	}
	return ids, nil
}

// beatsContract is the part of the authoring spec about beats.
func beatsContract(spec string) string {
	i := strings.Index(spec, "**Interaction beats**")
	if i < 0 {
		return spec
	}
	rest := spec[i:]
	if j := strings.Index(rest, "**Fade sequences**"); j >= 0 {
		rest = rest[:j]
	}
	return rest
}

func beatSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"beats": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"id":   map[string]any{"type": "string"},
						"yaml": map[string]any{"type": "string", "description": "the YAML inside the beat fence, starting with id:"},
					},
					"required": []string{"id", "yaml"},
				},
			},
		},
		"required": []string{"beats"},
	}
}

func validateBeat(id, body string) error {
	var b beat
	if err := yaml.Unmarshal([]byte(body), &b); err != nil {
		return fmt.Errorf("%s: rewritten beat does not parse: %w", id, err)
	}
	if b.ID != id {
		return fmt.Errorf("%s: rewritten beat carries id %q", id, b.ID)
	}
	if b.Check != "choice" || b.Rubric != "" || len(b.Options) < 3 {
		return fmt.Errorf("%s: not a choice with three options and no rubric (check %q, %d options)", id, b.Check, len(b.Options))
	}
	correct := 0
	for _, o := range b.Options {
		if o.Correct {
			correct++
		} else if o.Misconception == "" {
			return fmt.Errorf("%s: a distractor has no misconception id", id)
		}
		if o.Explain == "" {
			return fmt.Errorf("%s: an option has no explain", id)
		}
	}
	if correct != 1 {
		return fmt.Errorf("%s: %d correct options", id, correct)
	}
	return nil
}
