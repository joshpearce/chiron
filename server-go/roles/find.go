package roles

import (
	"fmt"
	"strings"

	"github.com/mjbraun/chiron/server/llm"
)

// Terms are what a brief searches for: phrases for the catalogues and
// subject tags for the index.
type Terms struct {
	Terms []string `json:"terms"`
	Tags  []string `json:"tags"`
}

const termsSystem = `You read a brief for a textbook and say what to search open textbook catalogues for. Return three to five search phrases a librarian would type, most specific first, and the subject tags from the given vocabulary that fit the brief.`

var termsSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"terms": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
		"tags":  map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
	},
	"required": []string{"terms", "tags"},
}

// SearchTerms extracts search phrases and subject tags from a brief.
func SearchTerms(chain llm.Chain, brief string, vocabulary []string) (Terms, error) {
	var out Terms
	user := fmt.Sprintf("BRIEF:\n%s\n\nTAG VOCABULARY (use only these):\n%s", brief, strings.Join(vocabulary, ", "))
	if err := chain.Structured("planner", termsSystem, user, termsSchema, "search_terms", &out); err != nil {
		return out, err
	}
	var tags []string
	allowed := map[string]bool{}
	for _, v := range vocabulary {
		allowed[strings.ToLower(v)] = true
	}
	for _, t := range out.Tags {
		if allowed[strings.ToLower(t)] {
			tags = append(tags, strings.ToLower(t))
		}
	}
	out.Tags = tags
	return out, nil
}

// A Pick is one source the planner chose and why.
type Pick struct {
	ID     string `json:"id"`
	Role   string `json:"role"`
	Reason string `json:"reason"`
}

type Picks struct {
	Picks []Pick `json:"picks"`
	// References are titles worth citing but not adapting: quotation-only
	// sources, or books the catalogues lack.
	References []string `json:"references"`
}

const pickSystem = `You choose the open texts an adaptive textbook is built from. From the candidates, pick one spine (the book whose order of ideas and worked examples the syllabus follows) and up to two interleaves (texts whose sections are woven in where they add what the spine lacks: exercises with answers, a contrasting voice, a missing topic). Prefer adaptable verdicts with a fetch recipe and a table of contents, a voice that fits the reader the brief describes, and sources with exercises and answers. A quotation-only or restricted candidate may be listed as a reference, never picked. Give one sentence of reason per pick, in plain prose without exclamation marks.`

var pickSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"picks": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id":     map[string]any{"type": "string"},
					"role":   map[string]any{"type": "string", "enum": []string{"spine", "interleave"}},
					"reason": map[string]any{"type": "string"},
				},
				"required": []string{"id", "role", "reason"},
			},
		},
		"references": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
	},
	"required": []string{"picks", "references"},
}

// PickSources chooses the spine and interleaves from a candidate block.
func PickSources(chain llm.Chain, brief, candidates string) (Picks, error) {
	var out Picks
	user := fmt.Sprintf("BRIEF:\n%s\n\nCANDIDATES:\n%s", brief, candidates)
	err := chain.Structured("planner", pickSystem, user, pickSchema, "source_picks", &out)
	return out, err
}
