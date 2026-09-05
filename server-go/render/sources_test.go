package render

import (
	"strings"
	"testing"

	"github.com/mjbraun/chiron/server/corpus"
)

// A unit adapted from open texts says so at the end of its chapter and in
// the chapter's sources list, once per source.
func TestAChapterNamesItsSources(t *testing.T) {
	u := &corpus.Unit{ID: "u0", Title: "Bayes", Front: map[string]any{"sources": []any{
		map[string]any{"title": "Think Bayes 2e", "authors": "Allen B. Downey", "licence": "CC BY-NC-SA 4.0"},
		map[string]any{"title": "Think Bayes 2e", "authors": "Allen B. Downey", "licence": "CC BY-NC-SA 4.0"},
		map[string]any{"title": "18.05 readings", "licence": "CC BY-NC-SA 4.0"},
	}}, IntroMD: "Prose <b>here</b>.\n"}
	ch, err := RenderChapter(u, nil, Directives{NextAction: "read"}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(ch.Sources) != 2 || ch.Sources[0] != "Adapted from Think Bayes 2e by Allen B. Downey (CC BY-NC-SA 4.0)" || ch.Sources[1] != "Adapted from 18.05 readings (CC BY-NC-SA 4.0)" {
		t.Fatalf("sources = %v", ch.Sources)
	}
	if !strings.Contains(ch.HTML, `<section class="sources"><h2>Sources</h2><ul><li>Adapted from Think Bayes 2e`) {
		t.Fatalf("html lacks the sources section:\n%s", ch.HTML)
	}
	plain, _ := RenderChapter(&corpus.Unit{ID: "u1", Title: "x", IntroMD: "p"}, nil, Directives{}, nil, nil)
	if plain.Sources != nil || strings.Contains(plain.HTML, "Sources") {
		t.Fatal("a unit without sources shows none")
	}
}
