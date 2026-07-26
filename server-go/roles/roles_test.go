package roles

import (
	"math/rand"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mjbraun/chiron/server/corpus"
	"github.com/mjbraun/chiron/server/llm"
	"github.com/mjbraun/chiron/server/render"
	"github.com/mjbraun/chiron/server/state"
)

type renderSection = render.AssembledSection

func fixtures(t *testing.T) (*corpus.Corpus, *state.Learner) {
	t.Helper()
	c, err := corpus.Load(filepath.Join("..", "..", "corpus"))
	if err != nil {
		t.Fatalf("corpus: %v", err)
	}
	l, err := state.Open(t.TempDir(), c)
	if err != nil {
		t.Fatalf("state: %v", err)
	}
	return c, l
}

// deadChain stands in for an unreachable model.
type deadChain struct{}

func (deadChain) Structured(role, system, user string, schema map[string]any, name string, out any) error {
	return llm.Errorf("no upstream reachable")
}
func (deadChain) Status() llm.Status { return llm.Status{} }

// A check below eight items is statistical noise for an 80% gate, so the
// composer must always fill the quota even when the callback pool is empty.
func TestComposeCheckAlwaysFillsTheQuota(t *testing.T) {
	c, l := fixtures(t)
	rng := rand.New(rand.NewSource(7))
	items := ComposeCheck(c.Units["u0"], l, c, 9, 0.4, rng)
	if len(items) != 9 {
		t.Errorf("got %d items on a first unit with no callback pool, want 9", len(items))
	}
	seen := map[string]bool{}
	for _, q := range items {
		if seen[q.ID] {
			t.Errorf("item %s appears twice in one check", q.ID)
		}
		seen[q.ID] = true
		if q.Unit == "" {
			t.Errorf("item %s has no unit attribution", q.ID)
		}
	}
}

// Response congruency: the goal is to derive and explain, so most of the check
// must be constructed rather than recognition.
func TestComposeCheckPrefersConstructedItems(t *testing.T) {
	c, l := fixtures(t)
	rng := rand.New(rand.NewSource(3))
	items := ComposeCheck(c.Units["u2"], l, c, 9, 0.4, rng)
	constructed := 0
	for _, q := range items {
		if q.Kind == "constructed" {
			constructed++
		}
	}
	if float64(constructed)/float64(len(items)) < 0.6 {
		t.Errorf("%d/%d constructed, spec wants at least 60%%", constructed, len(items))
	}
}

// Once prior units are cleared, the check must actually reach back into them -
// that is what makes it cumulative rather than a unit quiz.
func TestComposeCheckDrawsCallbacksFromClearedUnits(t *testing.T) {
	c, l := fixtures(t)
	score, passed := 1.0, true
	for _, uid := range []string{"u0", "u1"} {
		if _, err := l.Apply(state.Event{Kind: "check_result", Unit: uid,
			Score: &score, Passed: &passed}); err != nil {
			t.Fatal(err)
		}
	}
	rng := rand.New(rand.NewSource(11))
	items := ComposeCheck(c.Units["u2"], l, c, 9, 0.4, rng)
	fromPrior := 0
	for _, q := range items {
		if q.Unit != "u2" {
			fromPrior++
		}
	}
	if fromPrior == 0 {
		t.Error("no callbacks drawn from cleared units - the check is not cumulative")
	}
}

// With no measured evidence there is nothing to adapt to, so the first chapter
// must not wait on a model call.
func TestPlannerColdStartsWithoutCallingTheModel(t *testing.T) {
	c, l := fixtures(t)
	d := PlanDirectives(deadChain{}, l, c.Units["u0"], "")
	if d.Depth != "default" {
		t.Errorf("cold start depth %q", d.Depth)
	}
	if d.NextAction == "" {
		t.Error("every output must carry a concrete next action")
	}
	if len(d.SectionsToRewrite) != 0 {
		t.Error("nothing to rewrite before any evidence exists")
	}
}

// An unreachable model must degrade to canon, never to a broken chapter.
func TestAuthorFallsBackToCanonWhenTheModelIsDown(t *testing.T) {
	c, _ := fixtures(t)
	unit := c.Units["u1"]
	d := Directives{
		Depth: "default",
		SectionsToRewrite: []SectionRewrite{
			{Heading: unit.Sections[0].Heading, Instruction: "make it simpler"},
		},
	}
	sections, used := AuthorChapter(deadChain{}, unit, d, c)
	if used {
		t.Error("reported an LLM rewrite when the model was unreachable")
	}
	if len(sections) != len(unit.Sections) {
		t.Errorf("got %d sections, want %d", len(sections), len(unit.Sections))
	}
	for i, s := range sections {
		if !strings.HasPrefix(s.Markdown, "## ") {
			t.Errorf("section %d lost its heading", i)
		}
	}
}

// The beats are the teaching. A depth swap replaces prose, so it must re-append
// the canonical beats rather than dropping them.
func TestDepthSwapKeepsBeats(t *testing.T) {
	c, _ := fixtures(t)
	for _, id := range c.UnitOrder() {
		unit := c.Units[id]
		for depth := range unit.Depths {
			sections, _ := AuthorChapter(deadChain{}, unit, Directives{Depth: depth}, c)
			joined := strings.Join(markdowns(sections), "\n")
			for _, b := range unit.Beats() {
				if !strings.Contains(joined, "id: "+b.ID) {
					t.Errorf("%s at depth %q lost beat %s", id, depth, b.ID)
				}
			}
		}
	}
}

func markdowns(ss []renderSection) []string {
	out := make([]string, 0, len(ss))
	for _, s := range ss {
		out = append(out, s.Markdown)
	}
	return out
}

// Grading must never invent a verdict when it could not reach a model - an
// unassessed item goes to the debt ledger instead.
func TestGradeDegradesToUngraded(t *testing.T) {
	c, _ := fixtures(t)
	q, _ := c.FindQuestion("u1-q1")
	if q == nil {
		t.Fatal("fixture question missing")
	}
	g := GradeFreeText(deadChain{}, q, "some answer", c.Misconceptions)
	if g.Verdict != "ungraded" {
		t.Errorf("verdict %q - a guess is worse than admitting the gap", g.Verdict)
	}
	if g.NextAction == "" {
		t.Error("even a degraded grade must tell the learner what to do")
	}
}

// A turn that neither asks a question nor finishes leaves the learner with
// nothing to reply to and the conversation stalled.
type stallingChain struct{}

func (stallingChain) Structured(role, system, user string, schema map[string]any, name string, out any) error {
	e := out.(*Elicitation)
	e.ReplyMD = "I am planning a book on jet engines for you."
	e.Done = false
	return nil
}
func (stallingChain) Status() llm.Status { return llm.Status{} }

func TestElicitTurnNeverEndsWithNothingToAnswer(t *testing.T) {
	got := ElicitTurn(stallingChain{}, []Message{{Role: "learner", Text: "teach me jet engines"}})
	if got.Done {
		t.Fatal("fixture should not be done")
	}
	if !strings.Contains(got.ReplyMD, "?") {
		t.Errorf("an unfinished turn must end in a question, got %q", got.ReplyMD)
	}
}
