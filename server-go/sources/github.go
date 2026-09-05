package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strings"
)

// githubFetcher reads files straight from a repository: Markdown as it
// is, notebooks converted, anything else as a code block. The table of
// contents is the index's list, or a directory listing when the recipe
// names one.
type githubFetcher struct{}

func (githubFetcher) contents(ctx context.Context, c *Client, s *Source) ([]Section, error) {
	if s.Fetch.Dir == "" {
		return nil, fmt.Errorf("%s: no contents and no dir to list", s.ID)
	}
	api := base(Recipe{Base: s.Fetch.Base}, "https://api.github.com")
	if s.Fetch.Base != "" {
		api = s.Fetch.Base + "/api"
	}
	ref := s.Fetch.Ref
	if ref == "" {
		ref = "main"
	}
	data, err := c.get(ctx, fmt.Sprintf("%s/repos/%s/contents/%s?ref=%s", api, s.Fetch.Repo, strings.Trim(s.Fetch.Dir, "/"), ref))
	if err != nil {
		return nil, err
	}
	var entries []struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("list %s: %w", s.Fetch.Dir, err)
	}
	var out []Section
	for _, e := range entries {
		if e.Type != "file" {
			continue
		}
		switch strings.ToLower(path.Ext(e.Name)) {
		case ".md", ".ipynb", ".rst", ".tex", ".txt", ".v", ".markdown":
			out = append(out, Section{Locator: e.Name, Title: strings.TrimSuffix(e.Name, path.Ext(e.Name))})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Locator < out[j].Locator })
	// A filename is a poor title for a planner to choose by; the file's
	// first heading is the real one. The fetches are cached, so the
	// author's later reads cost nothing more.
	for i := range out {
		if i >= 80 {
			break
		}
		got, err := (githubFetcher{}).fetch(ctx, c, s, out[i].Locator)
		if err == nil && got.title != "" && firstHeading(got.markdown) != "" {
			out[i].Title = got.title
		}
	}
	return out, nil
}

func (githubFetcher) fetch(ctx context.Context, c *Client, s *Source, locator string) (fetched, error) {
	raw := base(s.Fetch, "https://raw.githubusercontent.com")
	ref := s.Fetch.Ref
	if ref == "" {
		ref = "main"
	}
	rel := strings.ReplaceAll(s.Fetch.Path, "{locator}", locator)
	url := fmt.Sprintf("%s/%s/%s/%s", raw, s.Fetch.Repo, ref, strings.TrimLeft(rel, "/"))
	data, err := c.get(ctx, url)
	if err != nil {
		return fetched{}, err
	}
	body := string(data)
	var md string
	switch strings.ToLower(path.Ext(rel)) {
	case ".md", ".markdown":
		md = stripFrontMatter(body)
	case ".ipynb":
		md, err = NotebookToMarkdown(data)
		if err != nil {
			return fetched{}, err
		}
	case ".rst", ".txt":
		md = body
	default:
		lang := strings.TrimPrefix(strings.ToLower(path.Ext(rel)), ".")
		md = "```" + lang + "\n" + body + "\n```\n"
	}
	title := firstHeading(md)
	if title == "" {
		title = strings.TrimSuffix(path.Base(rel), path.Ext(rel))
	}
	return fetched{title: title, markdown: md, url: url}, nil
}

// stripFrontMatter drops a leading YAML block (Jupytext, MyST, Jekyll).
func stripFrontMatter(md string) string {
	if !strings.HasPrefix(md, "---\n") {
		return md
	}
	rest := md[4:]
	end := strings.Index(rest, "\n---\n")
	if end < 0 {
		return md
	}
	return strings.TrimLeft(rest[end+5:], "\n")
}
