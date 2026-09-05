// Package sources is where the book builder gets material to adapt: an
// index of open textbooks, courseware and lecture notes with their
// licences, and fetchers that turn one section of one of them into
// Markdown with a provenance record. OPEN-SOURCES.md is the survey the
// index is drawn from and the rules the fetchers follow.
package sources

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Verdict is what the licence permits: A adapt, Q quote only, R restricted.
type Verdict string

const (
	Adaptable  Verdict = "A"
	QuoteOnly  Verdict = "Q"
	Restricted Verdict = "R"
)

type Licence struct {
	Name    string `yaml:"name" json:"name"`
	URL     string `yaml:"url" json:"url"`
	Checked string `yaml:"checked" json:"checked"`
}

// Recipe says how to reach a source: which fetcher, and what it needs.
type Recipe struct {
	// Kind: github, openstax, libretexts, mediawiki, ocw, gutenberg, pressbooks.
	Kind string `yaml:"kind" json:"kind"`
	// Base overrides the fetcher's default host (tests point it at a fixture).
	Base string `yaml:"base,omitempty" json:"base,omitempty"`
	// github: Repo "owner/name", Ref, Path with {locator}, Dir to list.
	Repo string `yaml:"repo,omitempty" json:"repo,omitempty"`
	Ref  string `yaml:"ref,omitempty" json:"ref,omitempty"`
	Path string `yaml:"path,omitempty" json:"path,omitempty"`
	Dir  string `yaml:"dir,omitempty" json:"dir,omitempty"`
	// openstax: Book uuid, Version when not the default.
	Book    string `yaml:"book,omitempty" json:"book,omitempty"`
	Version string `yaml:"version,omitempty" json:"version,omitempty"`
	// libretexts and pressbooks: URL of the book.
	URL string `yaml:"url,omitempty" json:"url,omitempty"`
	// mediawiki: Site and the book's page Prefix.
	Site   string `yaml:"site,omitempty" json:"site,omitempty"`
	Prefix string `yaml:"prefix,omitempty" json:"prefix,omitempty"`
	// ocw: Course slug.
	Course string `yaml:"course,omitempty" json:"course,omitempty"`
	// gutenberg: ebook ID.
	ID int `yaml:"id,omitempty" json:"id,omitempty"`
}

// Section is one entry of a source's table of contents.
type Section struct {
	Locator string `yaml:"locator" json:"locator"`
	Title   string `yaml:"title" json:"title"`
	Level   int    `yaml:"level,omitempty" json:"level,omitempty"`
}

type Source struct {
	ID        string   `yaml:"id" json:"id"`
	Title     string   `yaml:"title" json:"title"`
	Authors   []string `yaml:"authors" json:"authors"`
	Subjects  []string `yaml:"subjects" json:"subjects"`
	Verdict   Verdict  `yaml:"verdict" json:"verdict"`
	Licence   Licence  `yaml:"licence" json:"licence"`
	Voice     string   `yaml:"voice" json:"voice"`
	Format    string   `yaml:"format" json:"format"`
	Exercises string   `yaml:"exercises,omitempty" json:"exercises,omitempty"`
	Fetch     Recipe   `yaml:"fetch,omitempty" json:"fetch,omitempty"`
	// Contents, when given, is the table of contents; otherwise the fetcher lists it.
	Contents []Section `yaml:"contents,omitempty" json:"contents,omitempty"`
}

type Index struct {
	Sources []Source `yaml:"sources"`
}

func LoadIndex(path string) (*Index, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var idx Index
	if err := yaml.Unmarshal(data, &idx); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if problems := idx.Lint(); len(problems) > 0 {
		return nil, fmt.Errorf("%s:\n  %s", path, strings.Join(problems, "\n  "))
	}
	return &idx, nil
}

var kinds = map[string]bool{
	"github": true, "openstax": true, "libretexts": true, "mediawiki": true,
	"ocw": true, "gutenberg": true, "pressbooks": true,
}

// Lint is what every entry must satisfy before the builder trusts it: an
// id, a title, a licence with the page it was read on, a verdict, and when
// there is a recipe, one a fetcher understands.
func (idx *Index) Lint() []string {
	var out []string
	seen := map[string]bool{}
	for i, s := range idx.Sources {
		where := fmt.Sprintf("sources[%d] %q", i, s.ID)
		if s.ID == "" {
			out = append(out, fmt.Sprintf("sources[%d]: no id", i))
		}
		if seen[s.ID] {
			out = append(out, where+": duplicate id")
		}
		seen[s.ID] = true
		if s.Title == "" {
			out = append(out, where+": no title")
		}
		switch s.Verdict {
		case Adaptable, QuoteOnly, Restricted:
		default:
			out = append(out, where+": verdict must be A, Q or R")
		}
		if s.Licence.Name == "" || s.Licence.URL == "" {
			out = append(out, where+": licence needs a name and the URL it was read on")
		}
		if len(s.Subjects) == 0 {
			out = append(out, where+": no subjects")
		}
		// An adaptable source may have no recipe yet: the planner can still
		// name it, and the unit is then written from scratch with a citation.
		if s.Verdict == Adaptable && s.Fetch.Kind != "" {
			if !kinds[s.Fetch.Kind] {
				out = append(out, where+": unknown fetch kind "+s.Fetch.Kind)
				continue
			}
			switch s.Fetch.Kind {
			case "github":
				if s.Fetch.Repo == "" || s.Fetch.Path == "" {
					out = append(out, where+": github needs repo and path")
				}
				if !strings.Contains(s.Fetch.Path, "{locator}") {
					out = append(out, where+": github path needs {locator}")
				}
				if len(s.Contents) == 0 && s.Fetch.Dir == "" {
					out = append(out, where+": github needs contents or dir")
				}
			case "openstax":
				if s.Fetch.Book == "" {
					out = append(out, where+": openstax needs book")
				}
			case "libretexts", "pressbooks":
				if s.Fetch.URL == "" {
					out = append(out, where+": "+s.Fetch.Kind+" needs url")
				}
			case "mediawiki":
				if s.Fetch.Site == "" || s.Fetch.Prefix == "" {
					out = append(out, where+": mediawiki needs site and prefix")
				}
			case "ocw":
				if s.Fetch.Course == "" {
					out = append(out, where+": ocw needs course")
				}
			case "gutenberg":
				if s.Fetch.ID == 0 {
					out = append(out, where+": gutenberg needs id")
				}
			}
		}
	}
	return out
}

func (idx *Index) Get(id string) *Source {
	for i := range idx.Sources {
		if idx.Sources[i].ID == id {
			return &idx.Sources[i]
		}
	}
	return nil
}

// Matching ranks sources by how many of the tags they carry; ties keep
// index order. Sources with no matching tag are left out.
func (idx *Index) Matching(tags []string) []Source {
	type hit struct {
		s Source
		n int
		i int
	}
	var hits []hit
	for i, s := range idx.Sources {
		n := 0
		for _, t := range tags {
			for _, st := range s.Subjects {
				if strings.EqualFold(t, st) {
					n++
				}
			}
		}
		if n > 0 {
			hits = append(hits, hit{s, n, i})
		}
	}
	sort.SliceStable(hits, func(a, b int) bool {
		if hits[a].n != hits[b].n {
			return hits[a].n > hits[b].n
		}
		return hits[a].i < hits[b].i
	})
	out := make([]Source, len(hits))
	for i, h := range hits {
		out[i] = h.s
	}
	return out
}

// ByTitle finds a source the brief named: an exact id, or a title or
// author match, case-insensitive, whole words.
func (idx *Index) ByTitle(name string) *Source {
	q := strings.ToLower(strings.TrimSpace(name))
	if q == "" {
		return nil
	}
	if s := idx.Get(q); s != nil {
		return s
	}
	for i := range idx.Sources {
		s := &idx.Sources[i]
		if strings.Contains(strings.ToLower(s.Title), q) {
			return s
		}
		for _, a := range s.Authors {
			if strings.Contains(strings.ToLower(a), q) {
				return s
			}
		}
	}
	return nil
}
