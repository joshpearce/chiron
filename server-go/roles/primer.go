package roles

import (
	"fmt"
	"strings"

	"github.com/mjbraun/chiron/server/corpus"
	"github.com/mjbraun/chiron/server/llm"
	"github.com/mjbraun/chiron/server/render"
)

// A primer answers one reader's question about something they captured:
// selected text, a page, a transcribed image. Unlike a chapter it has no
// check and no planner; it is written once from the capture and grows from
// the reader's margin notes.

type Capture struct {
	Text   string
	URL    string
	App    string
	Prompt string
	// Brief is what the planning conversation settled on, when there was one.
	Brief string
}

var primerSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"title":    map[string]any{"type": "string", "description": "2-6 words naming the primer"},
		"markdown": map[string]any{"type": "string", "description": "the primer: an opening paragraph, then sections headed with '## '"},
	},
	"required": []string{"title", "markdown"},
}

const primerSystem = `You write a primer: a short, self-contained document that answers one reader's question about material they just captured on their iPad (selected text, a web page, or the transcription of an image).

- Start from what was captured. Quote or paraphrase it where that helps; say plainly when you go beyond it.
- Answer the reader's question first, then give what they need to understand the answer, then what is worth knowing next. Three to six sections headed with "## ". Between 400 and 1200 words.
- Audience: expert software engineer, fast reader, allergic to filler. Direct tone, no exclamation marks. Hyphens only, no em dashes. Real numbers and real names where the captured material has them.
- Prose must chain: each sentence opens on something the previous one put on the table and ends on the new thing. Never refer to a thing before the text has introduced it by that name.
- Markdown only: paragraphs, "## " headings, lists where a list is the right shape, fenced code for code. No front matter, no title line: the title is a separate field.` + acronymRule

// AuthorPrimer writes the document for a capture.
func AuthorPrimer(chain llm.Chain, cap Capture) (title, markdown string, err error) {
	var out struct {
		Title    string `json:"title"`
		Markdown string `json:"markdown"`
	}
	if err := chain.Structured("author", primerSystem, captureContext(cap), primerSchema, "primer", &out); err != nil {
		return "", "", err
	}
	out.Title = strings.TrimSpace(out.Title)
	if out.Title == "" || strings.TrimSpace(out.Markdown) == "" {
		return "", "", fmt.Errorf("the model returned an empty primer")
	}
	return out.Title, out.Markdown, nil
}

var extendSchema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"heading":  map[string]any{"type": "string", "description": "the new section's heading, without '## '"},
		"markdown": map[string]any{"type": "string", "description": "the new section's body"},
	},
	"required": []string{"heading", "markdown"},
}

const extendSystem = `You extend an existing primer with one new section in response to a margin note the reader left on a passage. The note may be a question, an objection, or a request for more.

- The section must stand on its own at the end of the document and answer the note directly, starting from the passage it was left on. Do not restate the rest of the primer.
- 100 to 400 words. Same rules as the primer: expert reader, direct, chained prose, hyphens only, markdown only.
- The heading names what the section adds, not the note.` + acronymRule

// ExtendPrimer writes the section a margin note asks for.
func ExtendPrimer(chain llm.Chain, doc, quote, note string) (heading, markdown string, err error) {
	var b strings.Builder
	fmt.Fprintf(&b, "THE PASSAGE THE NOTE IS ON:\n%s\n\nTHE READER'S NOTE:\n%s\n\nTHE PRIMER SO FAR:\n%s\n",
		strings.TrimSpace(quote), strings.TrimSpace(note), clip(doc, 60000))
	var out struct {
		Heading  string `json:"heading"`
		Markdown string `json:"markdown"`
	}
	if err := chain.Structured("author", extendSystem, b.String(), extendSchema, "primer_section", &out); err != nil {
		return "", "", err
	}
	out.Heading = strings.TrimPrefix(strings.TrimSpace(out.Heading), "## ")
	if out.Heading == "" || strings.TrimSpace(out.Markdown) == "" {
		return "", "", fmt.Errorf("the model returned an empty section")
	}
	return out.Heading, out.Markdown, nil
}

// VerbatimSections is the chapter of a unit with no planner and no rewrite:
// the canon as authored. Primers read this way.
func VerbatimSections(unit *corpus.Unit) []render.AssembledSection {
	out := make([]render.AssembledSection, 0, len(unit.Sections))
	for _, s := range unit.Sections {
		out = append(out, render.AssembledSection{Heading: s.Heading, Markdown: s.Markdown()})
	}
	return out
}
