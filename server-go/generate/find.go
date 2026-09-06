package generate

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mjbraun/chiron/server/roles"
	"github.com/mjbraun/chiron/server/sources"
	"gopkg.in/yaml.v3"
)

// findSources is the path for a brief that names nothing the index has:
// the planner says what to search for, the index and the catalogues
// offer candidates, the tables of contents of the best are fetched, and
// the planner picks a spine and interleaves. hints are names the brief
// gave that the index lacked; they are searched as they are.
func (g *Generator) findSources(ctx context.Context, brief string, hints []string) ([]chosen, []string, error) {
	if g.Index == nil {
		return nil, nil, nil
	}
	vocabulary := map[string]bool{}
	for _, s := range g.Index.Sources {
		for _, t := range s.Subjects {
			vocabulary[strings.ToLower(t)] = true
		}
	}
	var tags []string
	for t := range vocabulary {
		tags = append(tags, t)
	}
	sort.Strings(tags)
	terms, err := roles.SearchTerms(g.Chain, brief, tags)
	if err != nil {
		return nil, nil, err
	}
	search := append(append([]string{}, hints...), terms.Terms...)
	if len(search) > 5 {
		search = search[:5]
	}

	// Candidates: the index by tag and by title, then the catalogues.
	type candidate struct {
		src         sources.Source
		origin      string
		description string
	}
	var cands []candidate
	seen := map[string]bool{}
	add := func(s sources.Source, origin, description string) {
		if seen[s.ID] || s.Verdict == sources.Restricted {
			return
		}
		seen[s.ID] = true
		cands = append(cands, candidate{s, origin, description})
	}
	for _, s := range g.Index.Matching(terms.Tags) {
		add(s, "index", "")
	}
	for _, term := range search {
		if s := g.Index.ByTitle(term); s != nil {
			add(*s, "index", "")
		}
	}
	if g.Fetch != nil {
		for _, c := range g.Fetch.Search(ctx, search) {
			add(c.Source, c.Origin, c.Description)
		}
	}
	if len(cands) > 12 {
		cands = cands[:12]
	}
	if len(cands) == 0 {
		return nil, nil, nil
	}

	// Tables of contents for the best eight that can be fetched.
	contents := map[string][]sources.Section{}
	listed := 0
	for _, c := range cands {
		if listed >= 8 || g.Fetch == nil || c.src.Verdict != sources.Adaptable || c.src.Fetch.Kind == "" {
			continue
		}
		s := c.src
		secs, err := g.Fetch.Contents(ctx, &s)
		if err != nil {
			continue
		}
		if len(secs) > 240 {
			secs = secs[:240]
		}
		contents[c.src.ID] = secs
		listed++
	}

	var b strings.Builder
	for _, c := range cands {
		s := c.src
		fmt.Fprintf(&b, "\n[%s] %s: %q by %s (%s; verdict %s)", c.origin, s.ID, s.Title, strings.Join(s.Authors, ", "), s.Licence.Name, s.Verdict)
		if s.Voice != "" {
			fmt.Fprintf(&b, "\n  voice: %s", s.Voice)
		}
		if c.description != "" {
			fmt.Fprintf(&b, "\n  about: %.400s", c.description)
		}
		if s.Exercises != "" {
			fmt.Fprintf(&b, "\n  exercises: %s", s.Exercises)
		}
		if s.Fetch.Kind == "" {
			b.WriteString("\n  note: no fetch recipe; can be a reference, not a spine")
		}
		if secs := contents[s.ID]; len(secs) > 0 {
			b.WriteString("\n  sections (locator | title):")
			for i, sec := range secs {
				if i >= 30 {
					fmt.Fprintf(&b, "\n    ... and %d more", len(secs)-30)
					break
				}
				fmt.Fprintf(&b, "\n    %s | %s", sec.Locator, sec.Title)
			}
		}
		b.WriteString("\n")
	}
	picks, err := roles.PickSources(g.Chain, brief, b.String())
	if err != nil {
		return nil, nil, err
	}
	byID := map[string]candidate{}
	for _, c := range cands {
		byID[c.src.ID] = c
	}
	var out []chosen
	for _, p := range picks.Picks {
		c, ok := byID[p.ID]
		if !ok || c.src.Verdict != sources.Adaptable {
			continue
		}
		role := "interleave"
		if len(out) == 0 {
			role = "spine"
		}
		s := c.src
		ch := chosen{ID: s.ID, Title: s.Title, Authors: s.Authors, Licence: s.Licence.Name, LicenceURL: s.Licence.URL,
			Verdict: s.Verdict, Role: role, Voice: s.Voice, Fetch: s.Fetch, Exercises: s.Exercises, Origin: c.origin,
			Sections: contents[s.ID], Note: p.Reason, src: &s}
		if s.Fetch.Kind == "" {
			ch.Note += " (no fetch recipe: the planner may follow its shape, the author writes from scratch)"
		}
		out = append(out, ch)
		if len(out) == 3 {
			break
		}
	}
	return out, picks.References, nil
}

// loadFound reads the sources a build chose from sources.yaml, so a
// rerun and the unit authoring can fetch what the finder found even
// though the index never listed it.
func (g *Generator) loadFound() {
	raw, err := os.ReadFile(filepath.Join(g.OutDir, "sources.yaml"))
	if err != nil {
		return
	}
	var doc struct {
		Sources []chosen `yaml:"sources"`
	}
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return
	}
	g.found = nil
	for _, c := range doc.Sources {
		g.found = append(g.found, sources.Source{ID: c.ID, Title: c.Title, Authors: c.Authors, Verdict: c.Verdict,
			Licence: sources.Licence{Name: c.Licence, URL: c.LicenceURL}, Voice: c.Voice, Exercises: c.Exercises,
			Fetch: c.Fetch, Contents: c.Sections})
	}
}

// source finds a source by id: in the index, or among those a build found.
func (g *Generator) source(id string) *sources.Source {
	if g.Index != nil {
		if s := g.Index.Get(id); s != nil {
			return s
		}
	}
	for i := range g.found {
		if g.found[i].ID == id {
			return &g.found[i]
		}
	}
	return nil
}
