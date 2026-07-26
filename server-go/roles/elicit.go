package roles

import (
	"fmt"
	"strings"

	"github.com/mjbraun/chiron/server/llm"
)

// Message is one turn of the Teach-me conversation.
type Message struct {
	Role string `json:"role"` // learner | tutor
	Text string `json:"text"`
}

// Elicitation is the tutor's reply, plus the brief once it has one.
type Elicitation struct {
	ReplyMD string `json:"reply_md"`
	Done    bool   `json:"done"`
	Brief   string `json:"brief"`
	Slug    string `json:"slug"`
	Title   string `json:"title"`
}

var elicitSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"reply_md": map[string]any{"type": "string",
			"description": "what to say to the learner now: one question, or the summary that accompanies a finished brief"},
		"done": map[string]any{"type": "boolean",
			"description": "true only when the brief below is complete enough to plan a syllabus from"},
		"brief": map[string]any{"type": "string",
			"description": "the full curriculum brief, written in the learner's voice; empty until done"},
		"slug":  map[string]any{"type": "string", "description": "short kebab-case id for the subject; empty until done"},
		"title": map[string]any{"type": "string", "description": "display title for the subject; empty until done"},
	},
	"required": []string{"reply_md", "done", "brief", "slug", "title"},
}

const elicitorSystem = `A learner wants a book written for them. Your job is to find out enough to plan one, then write the brief - not to teach anything yet.

Ask ONE question per turn, and only questions whose answer would change the curriculum. Those are, roughly in order of value:

1. What they want to be able to DO afterwards. "Understand X" is not yet an answer; "debug X in production", "read the papers", "make the build/buy call" each produce different books.
2. What they already know, specifically enough to skip it. An expert bored through three chapters of review stops reading.
3. How much time they have, since the spine is budgeted against it.
4. How much math they want, and whether they want it derived or just used.

Do not ask what you can reasonably infer, and do not ask two things at once. Four questions is usually plenty and seven is too many - a learner who came to read a book is not here to fill in a form. If an answer is vague on something that matters, follow up once; if it is still vague, choose a sensible default and say which default you chose.

Every reply must do one of exactly two things. Either ` + "`done`" + ` is false and reply_md ENDS IN A QUESTION - never a summary of what you are about to build, because the learner has nothing to answer and the conversation stalls - or ` + "`done`" + ` is true and you have written the brief. There is no third kind of turn.

When you have enough, set done and write the brief as a single paragraph in the learner's own voice: what they want, what they already know, how long they have, and how deep the math goes. Everything the planner sees comes from that paragraph, so anything you learned and left out is lost. Your reply_md that turn should state the shape of the book you are about to build and name the defaults you assumed, so a wrong assumption is caught before generation.`

// ElicitTurn runs one turn of the Teach-me conversation. The whole transcript
// is passed in each time, so the server holds no conversation state.
func ElicitTurn(chain llm.Chain, messages []Message) Elicitation {
	var b strings.Builder
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
	user := "CONVERSATION SO FAR:\n\n" + b.String() + "\n\nYour turn."

	var out Elicitation
	if err := chain.Structured("planner", elicitorSystem, user, elicitSchema, "elicit", &out); err != nil {
		return Elicitation{ReplyMD: fmt.Sprintf(
			"No model reachable (%v). Nothing was lost - try again when connected.", err)}
	}
	// A turn that neither asks nor finishes is a dead end: the learner is handed
	// a summary with nothing to reply to. Smaller local models do this, so
	// rather than stall the screen, turn it back into a question.
	if !out.Done && !strings.Contains(out.ReplyMD, "?") {
		out.ReplyMD = strings.TrimRight(out.ReplyMD, " \n") +
			"\n\nHave I got that right, or is there something important I have wrong?"
	}
	return out
}
