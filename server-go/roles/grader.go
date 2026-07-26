// Package roles holds the three LLM roles: grader, planner, author (plus the
// Teach-me elicitor).
//
// Division of labour: everything deterministic lives in code - mastery updates,
// fringe, gating, debt, callback selection. The model supplies judgement only:
// grading free text against rubrics, choosing depth and representation, and
// rewriting flagged sections.
package roles

import (
	"fmt"
	"sort"
	"strings"

	"github.com/mjbraun/chiron/server/corpus"
	"github.com/mjbraun/chiron/server/llm"
)

// Grade is one adjudicated free-text answer.
type Grade struct {
	Verdict        string   `json:"verdict"` // pass | partial | fail | valid_alternative_path
	Misconceptions []string `json:"misconceptions"`
	Evidence       string   `json:"evidence"`
	FeedbackMD     string   `json:"feedback_md"`
	NextAction     string   `json:"next_action"`
}

var gradeSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"verdict": map[string]any{
			"type": "string",
			"enum": []string{"pass", "partial", "fail", "valid_alternative_path"},
		},
		"misconceptions": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
		"evidence": map[string]any{"type": "string",
			"description": "shortest quote from the learner answer that justifies the verdict"},
		"feedback_md": map[string]any{"type": "string",
			"description": "2-5 sentences: why the correct answer is correct, why THIS answer earned THIS verdict, and what mental model produces the error if any"},
		"next_action": map[string]any{"type": "string",
			"description": "one concrete instruction for the learner right now"},
	},
	"required": []string{"verdict", "misconceptions", "evidence", "feedback_md", "next_action"},
}

const graderSystem = `You grade one answer against one rubric for a technical textbook on how LLMs work.

Rules, in priority order:
1. Grade against the rubric ONLY. The rubric's pass criteria are the contract.
2. Fluent, confident, jargon-rich answers that miss the rubric criteria FAIL. Well-written wrongness is the failure mode you exist to catch. Do not give benefit of the doubt to articulate answers.
3. An answer that reaches a correct result by a legitimate route the rubric didn't anticipate is ` + "`valid_alternative_path`" + `, not ` + "`fail`" + `. Punishing valid unusual reasoning is as wrong as accepting invalid reasoning.
4. ` + "`partial`" + ` means some rubric criteria are genuinely met, not "close in spirit".
5. If the answer exhibits a listed misconception, cite its ID. Only cite IDs from the provided list.
6. The learner may sound tentative or cite authorities ("I read that..."). Neither humility nor authority changes correctness.
Feedback is for an expert engineer: direct, technical, no praise-padding.`

// UngradedFeedback is what a learner sees when no model was reachable. The
// answer is stored but not adjudicated - guessing a verdict would be worse than
// admitting the gap, and the item goes to the debt ledger as unassessed.
func ungraded() Grade {
	return Grade{
		Verdict:        "ungraded",
		Misconceptions: []string{},
		FeedbackMD: "No model reachable - answer stored, not graded. " +
			"Compare against the reference answer shown.",
		NextAction: "Self-check against the reference, then continue.",
	}
}

func misconceptionList(bank map[string]corpus.Misconception) string {
	ids := make([]string, 0, len(bank))
	for id := range bank {
		ids = append(ids, id)
	}
	sort.Strings(ids) // stable prompts: Go map order is deliberately random
	var b strings.Builder
	for _, id := range ids {
		m := bank[id]
		fmt.Fprintf(&b, "- %s: %s: %s\n", m.ID, m.Name, m.WrongModel)
	}
	return b.String()
}

// GradeFreeText adjudicates one constructed answer.
func GradeFreeText(chain llm.Chain, q *corpus.Question, answer string,
	bank map[string]corpus.Misconception) Grade {
	reference := q.Answer.String()
	if reference == "" {
		reference = "(rubric only)"
	}
	rubric := q.Rubric
	if rubric == "" {
		rubric = "(match the reference answer)"
	}
	user := fmt.Sprintf(`QUESTION:
%s

REFERENCE ANSWER:
%s

RUBRIC:
%s

KNOWN MISCONCEPTIONS (cite by ID only if the answer exhibits one):
%s

LEARNER ANSWER:
%s`, q.Prompt, reference, rubric, misconceptionList(bank), answer)

	var g Grade
	if err := chain.Structured("grader", graderSystem, user, gradeSchema, "grade", &g); err != nil {
		return ungraded()
	}
	if g.Misconceptions == nil {
		g.Misconceptions = []string{}
	}
	return g
}

type batchGrade struct {
	ItemID string `json:"item_id"`
	Grade
}

var batchGradeSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"grades": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"item_id": map[string]any{"type": "string"},
					"verdict": map[string]any{
						"type": "string",
						"enum": []string{"pass", "partial", "fail", "valid_alternative_path"},
					},
					"misconceptions": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
					"evidence":       map[string]any{"type": "string"},
					"feedback_md":    map[string]any{"type": "string"},
					"next_action":    map[string]any{"type": "string"},
				},
				"required": []string{"item_id", "verdict", "misconceptions", "evidence",
					"feedback_md", "next_action"},
			},
		},
	},
	"required": []string{"grades"},
}

// BatchItem is one answer awaiting adjudication.
type BatchItem struct {
	ItemID   string
	Question *corpus.Question
	Answer   string
}

// GradeFreeTextBatch grades several answers in one call, keyed by item id.
//
// Grading is independent per item, so batching costs nothing pedagogically -
// each item is still judged against its own rubric and still cites its own
// evidence. It returns whatever it could grade; the caller falls back per item
// for anything missing, so a partial answer degrades instead of failing.
func GradeFreeTextBatch(chain llm.Chain, items []BatchItem,
	bank map[string]corpus.Misconception) map[string]Grade {
	if len(items) == 0 {
		return map[string]Grade{}
	}
	var blocks []string
	var ids []string
	for _, it := range items {
		reference := it.Question.Answer.String()
		if reference == "" {
			reference = "(rubric only)"
		}
		rubric := it.Question.Rubric
		if rubric == "" {
			rubric = "(match the reference answer)"
		}
		blocks = append(blocks, fmt.Sprintf(
			"### ITEM %s\nQUESTION:\n%s\n\nREFERENCE ANSWER:\n%s\n\nRUBRIC:\n%s\n\nLEARNER ANSWER:\n%s",
			it.ItemID, it.Question.Prompt, reference, rubric, it.Answer))
		ids = append(ids, it.ItemID)
	}
	user := fmt.Sprintf(
		"Grade each item below independently against its own rubric.\n\n"+
			"KNOWN MISCONCEPTIONS (cite by ID only if the answer exhibits one):\n%s\n\n%s\n\n"+
			"Return one grade per item, using the exact item ids: %s",
		misconceptionList(bank), strings.Join(blocks, "\n\n"), strings.Join(ids, ", "))

	var out struct {
		Grades []batchGrade `json:"grades"`
	}
	if err := chain.Structured("grader", graderSystem, user, batchGradeSchema, "grades", &out); err != nil {
		return map[string]Grade{}
	}
	result := map[string]Grade{}
	for _, g := range out.Grades {
		if g.ItemID == "" {
			continue
		}
		if g.Misconceptions == nil {
			g.Misconceptions = []string{}
		}
		result[g.ItemID] = g.Grade
	}
	return result
}
