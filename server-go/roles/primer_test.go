package roles

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/mjbraun/chiron/server/llm"
)

type primerChain struct {
	payload map[string]any
	system  string
	user    string
}

func (c *primerChain) Structured(role, system, user string, schema map[string]any, name string, out any) error {
	c.system, c.user = system, user
	data, _ := json.Marshal(c.payload)
	return json.Unmarshal(data, out)
}
func (primerChain) Status() llm.Status { return llm.Status{Connected: true} }

func TestAuthorPrimerGroundsInTheCapture(t *testing.T) {
	c := &primerChain{payload: map[string]any{"title": "Robots and money", "markdown": "Intro.\n\n## One\n\nBody."}}
	title, md, err := AuthorPrimer(c, Capture{Text: "User-agent: *\nContent-Signal: search=yes", URL: "https://lexweekly.example", App: "Safari", Prompt: "what is Content-Signal?"})
	if err != nil || title != "Robots and money" || !strings.Contains(md, "## One") {
		t.Fatalf("%q %q %v", title, md, err)
	}
	for _, want := range []string{"what is Content-Signal?", "Content-Signal: search=yes", "Safari", "lexweekly.example"} {
		if !strings.Contains(c.user, want) {
			t.Errorf("prompt lacks %q:\n%s", want, c.user)
		}
	}
	if !strings.Contains(c.system, "## ") {
		t.Errorf("system prompt does not ask for sections")
	}
}

func TestExtendPrimerSeesPassageNoteAndDoc(t *testing.T) {
	c := &primerChain{payload: map[string]any{"heading": "## Why 402", "markdown": "Because."}}
	h, md, err := ExtendPrimer(c, "Intro.\n\n## One\n\nBody.", "the 402 status", "why not 403?")
	if err != nil || h != "Why 402" || md != "Because." {
		t.Fatalf("%q %q %v", h, md, err)
	}
	for _, want := range []string{"the 402 status", "why not 403?", "## One"} {
		if !strings.Contains(c.user, want) {
			t.Errorf("prompt lacks %q", want)
		}
	}
}

func TestEmptyPrimerIsAnError(t *testing.T) {
	c := &primerChain{payload: map[string]any{"title": "", "markdown": ""}}
	if _, _, err := AuthorPrimer(c, Capture{Text: "x", Prompt: "y"}); err == nil {
		t.Fatal("accepted an empty primer")
	}
}
