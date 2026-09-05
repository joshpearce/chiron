package sources

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A smoke test against the real hosts, run by hand:
//
//	CHIRON_LIVE=1 go test ./sources -run Live -v
//
// It fetches one section from each kind in the shipped index and prints
// the first lines, so a changed site shows up here and not in a build.
func TestLiveFetchOneOfEachKind(t *testing.T) {
	if os.Getenv("CHIRON_LIVE") == "" {
		t.Skip("set CHIRON_LIVE=1 to fetch from the real hosts")
	}
	idx, err := LoadIndex(filepath.Join("..", "..", "corpus", "sources", "index.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	c := NewClient(filepath.Join(t.TempDir(), "cache"))
	want := map[string]string{"d2l": "", "think-bayes": "", "openstax-calculus-1": "", "lt-intro-stats-1e": "", "wikibooks-haskell": "", "ocw-18-05": "", "pg-origin": "", "rebus-phil-logic": "", "quantecon-intro": "", "rust-book": ""}
	for id := range want {
		s := idx.Get(id)
		if s == nil {
			t.Errorf("%s: not in the index", id)
			continue
		}
		secs, err := c.Contents(context.Background(), s)
		if err != nil {
			t.Errorf("%s contents: %v", id, err)
			continue
		}
		if len(secs) == 0 {
			t.Errorf("%s: no sections", id)
			continue
		}
		// A section with prose in it: not a title page, not the front matter.
		pick := secs[0]
		for _, sec := range secs {
			front := strings.Contains(sec.Locator, "Front_Matter") || strings.Contains(sec.Locator, "TitlePage")
			if (sec.Level >= 2 || strings.Contains(strings.ToLower(sec.Title), "regression")) && !front {
				pick = sec
				break
			}
		}
		ch, err := c.Fetch(context.Background(), s, pick.Locator)
		if err != nil {
			t.Errorf("%s fetch %s: %v", id, pick.Locator, err)
			continue
		}
		lines := strings.Split(ch.Markdown, "\n")
		if len(lines) > 6 {
			lines = lines[:6]
		}
		t.Logf("%s: %d sections; %q (%d bytes, verdict %s, note %q)\n  %s", id, len(secs), ch.Title, len(ch.Markdown), ch.Provenance.Verdict, ch.Provenance.Note, strings.Join(lines, "\n  "))
		if len(ch.Markdown) < 200 {
			t.Errorf("%s: suspiciously short: %q", id, ch.Markdown)
		}
	}
}
