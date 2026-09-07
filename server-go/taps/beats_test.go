package taps

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mjbraun/chiron/server/llm"
)

type beatChain struct {
	asked string
	reply []map[string]string
}

func (c *beatChain) Structured(role, system, user string, schema map[string]any, name string, out any) error {
	c.asked = user
	data, _ := json.Marshal(map[string]any{"beats": c.reply})
	return json.Unmarshal(data, out)
}

func (beatChain) Status() llm.Status { return llm.Status{Connected: true} }

const canonWithBeats = "---\nunit: u1\ntitle: T\n---\n\n## One\n\nProse.\n\n```beat\nid: u1-b1\ntype: predict\nconcept: c-a\nprompt: |\n  What happens?\nanswer: |\n  It saturates.\nrubric: |\n  Must say saturate.\ncheck: llm\n```\n\nMore prose.\n\n```beat\nid: u1-b2\ntype: compute\nconcept: c-a\nprompt: 2+2?\nanswer: '4'\ncheck: numeric(0.01)\n```\n\n## Two\n\nEnd.\n"

func TestProseBeatsBecomeChoicesInCanonAndEveryDepth(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "depths"), 0o755)
	os.WriteFile(filepath.Join(dir, "canon.md"), []byte(canonWithBeats), 0o644)
	depth := "## One\n\nShorter prose.\n\n```beat\nid: u1-b1\ntype: predict\nconcept: c-a\nprompt: |\n  What happens?\nanswer: |\n  It saturates.\nrubric: |\n  Must say saturate.\ncheck: llm\n```\n\n## Two\n\nEnd.\n"
	os.WriteFile(filepath.Join(dir, "depths", "less.md"), []byte(depth), 0o644)
	bank := filepath.Join(dir, "bank.yaml")
	spec := filepath.Join(dir, "spec.md")
	os.WriteFile(bank, []byte("misconceptions: []\n"), 0o644)
	os.WriteFile(spec, []byte("# Contract\n\n3. **Interaction beats**: rules\n\n3. **Fade sequences**: x\n"), 0o644)
	chain := &beatChain{reply: []map[string]string{{"id": "u1-b1",
		"yaml": "id: u1-b1\ntype: predict\nconcept: c-a\nprompt: |\n  What happens?\noptions:\n  - text: It saturates.\n    correct: true\n    explain: Yes.\n  - text: It overflows.\n    misconception: M7\n    explain: No.\n  - text: Nothing.\n    misconception: M2\n    explain: No.\ncheck: choice\n"}}}
	ids, err := RewriteBeats(chain, dir, bank, spec, 6)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(ids, ",") != "u1-b1" {
		t.Fatalf("rewrote %v", ids)
	}
	if !strings.Contains(chain.asked, "id: u1-b1") || strings.Contains(chain.asked, "id: u1-b2\ntype: compute") && !strings.Contains(chain.asked, "BEATS TO REWRITE") {
		t.Fatalf("prompt:\n%s", chain.asked)
	}
	canon, _ := os.ReadFile(filepath.Join(dir, "canon.md"))
	got := string(canon)
	if !strings.Contains(got, "```beat\nid: u1-b1\ntype: predict\nconcept: c-a\nprompt: |\n  What happens?\noptions:\n") || strings.Contains(got, "rubric:") {
		t.Fatalf("canon:\n%s", got)
	}
	if !strings.Contains(got, "```beat\nid: u1-b2\ntype: compute\nconcept: c-a\nprompt: 2+2?\nanswer: '4'\ncheck: numeric(0.01)\n```") || !strings.HasSuffix(got, "## Two\n\nEnd.\n") {
		t.Fatalf("the rest of canon changed:\n%s", got)
	}
	less, _ := os.ReadFile(filepath.Join(dir, "depths", "less.md"))
	if !strings.Contains(string(less), "options:\n  - text: It saturates.") || !strings.Contains(string(less), "Shorter prose.") {
		t.Fatalf("the depth's copy of the beat was not rewritten to match:\n%s", less)
	}
	// A second run finds nothing left to do.
	ids, err = RewriteBeats(chain, dir, bank, spec, 6)
	if err != nil || len(ids) != 0 {
		t.Fatalf("second run: %v %v", ids, err)
	}
}
