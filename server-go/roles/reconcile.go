package roles

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mjbraun/chiron/server/llm"
)

// A mark as the reader's devices keep it: the shape is the app's, read
// here only to merge two copies. Unknown fields ride along untouched.
type mark map[string]any

func (m mark) str(key string) string {
	s, _ := m[key].(string)
	return s
}

func (m mark) thread() []map[string]any {
	raw, _ := m["thread"].([]any)
	out := make([]map[string]any, 0, len(raw))
	for _, t := range raw {
		if tm, ok := t.(map[string]any); ok {
			out = append(out, tm)
		}
	}
	return out
}

// richness orders two copies of one mark: more turns, then an answer,
// then a question, then longer text.
func richness(m mark) int {
	n := len(m.thread()) * 100
	if m.str("answer") != "" {
		n += 10
	}
	if m.str("question") != "" {
		n += 5
	}
	return n + len(m.str("question"))/50
}

const reconcileSystem = `Two devices kept separate copies of one reader's question thread about a passage, and both added to it while apart. Merge the two into one thread that keeps every question the reader asked and every answer the tutor gave, in the order they were most likely asked, with nothing repeated and nothing invented. Where the same question was answered differently on each side, keep the fuller answer.`

// ReconcileMarks merges two lists of marks. Marks only one side has are
// kept. A mark both sides have takes the richer copy; where both sides
// added turns to the same question, the tutor merges the thread so no
// turn is lost. The result is JSON the app decodes as it does its own.
func ReconcileMarks(chain llm.Chain, mine, theirs json.RawMessage) (json.RawMessage, error) {
	var a, b []mark
	if len(mine) > 0 {
		if err := json.Unmarshal(mine, &a); err != nil {
			return nil, fmt.Errorf("my marks: %w", err)
		}
	}
	if len(theirs) > 0 {
		if err := json.Unmarshal(theirs, &b); err != nil {
			return nil, fmt.Errorf("their marks: %w", err)
		}
	}
	byID := map[string]mark{}
	var order []string
	add := func(m mark) {
		id := m.str("id")
		if id == "" {
			return
		}
		if _, seen := byID[id]; !seen {
			order = append(order, id)
		}
		byID[id] = m
	}
	for _, m := range a {
		add(m)
	}
	for _, m := range b {
		id := m.str("id")
		existing, both := byID[id]
		if !both {
			add(m)
			continue
		}
		merged, err := mergeMark(chain, existing, m)
		if err != nil {
			return nil, err
		}
		byID[id] = merged
	}
	out := make([]mark, 0, len(order))
	for _, id := range order {
		out = append(out, byID[id])
	}
	return json.Marshal(out)
}

// mergeMark: identical copies are one; otherwise the richer copy, with
// the thread merged by the tutor when both sides carry turns the other
// lacks.
func mergeMark(chain llm.Chain, x, y mark) (mark, error) {
	xj, _ := json.Marshal(x)
	yj, _ := json.Marshal(y)
	if string(xj) == string(yj) {
		return x, nil
	}
	base, other := x, y
	if richness(y) > richness(x) {
		base, other = y, x
	}
	bt, ot := base.thread(), other.thread()
	if len(ot) == 0 || threadContains(bt, ot) {
		return base, nil
	}
	if chain == nil {
		// No tutor: the base keeps its thread and the other's extra turns
		// follow it, so nothing is lost even if the order is imperfect.
		merged := append([]map[string]any{}, bt...)
		for _, t := range ot {
			if !turnIn(bt, t) {
				merged = append(merged, t)
			}
		}
		base["thread"] = toAny(merged)
		return base, nil
	}
	user := fmt.Sprintf("PASSAGE: %q\nFIRST QUESTION: %q\nFIRST ANSWER: %q\n\nCOPY A THREAD (json): %s\n\nCOPY B THREAD (json): %s",
		base.str("text"), base.str("question"), base.str("answer"), mustJSON(bt), mustJSON(ot))
	var out struct {
		Thread []struct {
			Question string `json:"question"`
			Answer   string `json:"answer"`
		} `json:"thread"`
	}
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"thread": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"question": map[string]any{"type": "string"},
						"answer":   map[string]any{"type": "string"},
					},
					"required": []string{"question", "answer"},
				},
			},
		},
		"required": []string{"thread"},
	}
	if err := chain.Structured("answer", reconcileSystem, user, schema, "merged_thread", &out); err != nil {
		return nil, err
	}
	merged := make([]map[string]any, 0, len(out.Thread))
	for _, t := range out.Thread {
		turn := map[string]any{"question": t.Question}
		if strings.TrimSpace(t.Answer) != "" {
			turn["answer"] = t.Answer
		}
		merged = append(merged, turn)
	}
	base["thread"] = toAny(merged)
	return base, nil
}

func threadContains(outer, inner []map[string]any) bool {
	for _, t := range inner {
		if !turnIn(outer, t) {
			return false
		}
	}
	return true
}

func turnIn(thread []map[string]any, t map[string]any) bool {
	q, _ := t["question"].(string)
	for _, u := range thread {
		if uq, _ := u["question"].(string); uq == q {
			return true
		}
	}
	return false
}

func toAny(ts []map[string]any) []any {
	out := make([]any, len(ts))
	for i, t := range ts {
		out[i] = t
	}
	return out
}

func mustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
