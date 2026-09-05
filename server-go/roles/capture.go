package roles

import (
	"fmt"
	"strings"

	"github.com/mjbraun/chiron/server/llm"
)

// The capture card's scales. A summary or a description is answered on
// the spot; a primer or a book is planned first, in a short conversation,
// and written from the brief that comes out of it.
const (
	ScaleSummary     = "summary"
	ScaleDescription = "description"
	ScalePrimer      = "primer"
	ScaleBook        = "book"
)

var captureAnswerSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"markdown": map[string]any{"type": "string", "description": "the answer, in markdown"},
	},
	"required": []string{"markdown"},
}

const captureAnswerSystem = `You answer one reader's question about material they just captured on their iPad (selected text, a web page, or the transcription of an image).

- Start from what was captured; say plainly when you go beyond it.
- Audience: expert software engineer, fast reader, allergic to filler. Direct tone, no exclamation marks. Hyphens only, no em dashes. Real numbers and real names where the captured material has them.
- Markdown only, no headings, no title line.
` + acronymRule

const summaryRule = "- Length: one paragraph, at most 120 words. The answer and nothing else.\n"
const descriptionRule = "- Length: two or three paragraphs, 200 to 350 words: the answer, then what the reader needs to make sense of it.\n"

// AnswerCapture answers the question at the summary or description scale.
func AnswerCapture(chain llm.Chain, cap Capture, scale string) (string, error) {
	system := captureAnswerSystem + summaryRule
	if scale == ScaleDescription {
		system = captureAnswerSystem + descriptionRule
	}
	var out struct {
		Markdown string `json:"markdown"`
	}
	if err := chain.Structured("author", system, captureContext(cap), captureAnswerSchema, "capture-"+scale, &out); err != nil {
		return "", err
	}
	if strings.TrimSpace(out.Markdown) == "" {
		return "", fmt.Errorf("the model returned an empty answer")
	}
	return out.Markdown, nil
}

const planPrimerSystem = `A reader captured something on their iPad and asked a question about it. They want a primer: a short document answering the question. Your job is to learn what would make it the right primer for them, then write the brief - not to answer anything yet.

Ask ONE question per turn, and at most two questions in all. Worth asking: what they want to be able to do with the answer, or which part of the captured material matters to them; and what they already know, so the primer can skip it. Do not ask what the capture and the question already answer.

Every reply does one of exactly two things. Either ` + "`done`" + ` is false and reply_md ENDS IN A QUESTION, or ` + "`done`" + ` is true and you have written the brief: one paragraph in the reader's own voice saying what the primer should cover, what it can skip, and how deep to go. That turn's reply_md says in a sentence what you are about to write. The title is 2-6 words naming the primer; leave slug empty.`

const planBookPreface = `The learner captured material on their iPad (below) and wants a whole book grown from it. The captured material is the seed: the brief must say what the book grows from it, and everything the learner tells you about what they want comes on top.

`

// PlanCapture runs one turn of the conversation that precedes a primer or
// a book. The whole transcript is passed in; the caller holds it.
func PlanCapture(chain llm.Chain, cap Capture, scale string, messages []Message) (Elicitation, error) {
	system := planPrimerSystem
	if scale == ScaleBook {
		system = planBookPreface + elicitorSystem
	}
	var b strings.Builder
	b.WriteString(captureContext(cap))
	b.WriteString("\nCONVERSATION SO FAR:\n\n")
	if len(messages) == 0 {
		b.WriteString("(nothing yet: open with your first question)")
	}
	for i, m := range messages {
		who := "YOU"
		if m.Role == "learner" {
			who = "LEARNER"
		}
		if i > 0 {
			b.WriteString("\n\n")
		}
		fmt.Fprintf(&b, "%s: %s", who, m.Text)
	}
	b.WriteString("\n\nYour turn.")
	var out Elicitation
	if err := chain.Structured("planner", system, b.String(), elicitSchema, "plan-"+scale, &out); err != nil {
		return Elicitation{}, err
	}
	if !out.Done && !strings.Contains(out.ReplyMD, "?") {
		out.ReplyMD = strings.TrimRight(out.ReplyMD, " \n") +
			"\n\nHave I got that right, or is there something important I have wrong?"
	}
	return out, nil
}

func captureContext(cap Capture) string {
	var b strings.Builder
	fmt.Fprintf(&b, "THE READER'S QUESTION:\n%s\n\n", strings.TrimSpace(cap.Prompt))
	if cap.Brief != "" {
		fmt.Fprintf(&b, "WHAT THE READER SAID THEY WANT:\n%s\n\n", strings.TrimSpace(cap.Brief))
	}
	if cap.URL != "" || cap.App != "" {
		fmt.Fprintf(&b, "CAPTURED FROM: %s %s\n\n", cap.App, cap.URL)
	}
	fmt.Fprintf(&b, "CAPTURED MATERIAL:\n%s\n", clip(strings.TrimSpace(cap.Text), 60000))
	return b.String()
}
