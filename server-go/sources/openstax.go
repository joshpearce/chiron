package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// openstaxFetcher reads a book through the REX archive: release.json for
// the archive path and the book's current version, the book's tree for
// the table of contents, and one JSON document per page with the HTML.
type openstaxFetcher struct{}

type openstaxRelease struct {
	ArchiveURL string `json:"archiveUrl"`
	Books      map[string]struct {
		DefaultVersion string `json:"defaultVersion"`
	} `json:"books"`
}

type openstaxNode struct {
	ID       string         `json:"id"`
	Title    string         `json:"title"`
	Contents []openstaxNode `json:"contents"`
}

func (openstaxFetcher) archive(ctx context.Context, c *Client, s *Source) (host, archive, version string, err error) {
	host = base(s.Fetch, "https://openstax.org")
	data, err := c.get(ctx, host+"/rex/release.json")
	if err != nil {
		return "", "", "", err
	}
	var rel openstaxRelease
	if err := json.Unmarshal(data, &rel); err != nil {
		return "", "", "", fmt.Errorf("release.json: %w", err)
	}
	version = s.Fetch.Version
	if version == "" {
		version = rel.Books[s.Fetch.Book].DefaultVersion
	}
	if version == "" {
		return "", "", "", fmt.Errorf("book %s is not in release.json and the recipe gives no version", s.Fetch.Book)
	}
	return host, strings.TrimRight(rel.ArchiveURL, "/"), version, nil
}

func (f openstaxFetcher) contents(ctx context.Context, c *Client, s *Source) ([]Section, error) {
	host, archive, version, err := f.archive(ctx, c, s)
	if err != nil {
		return nil, err
	}
	data, err := c.get(ctx, fmt.Sprintf("%s%s/contents/%s@%s.json", host, archive, s.Fetch.Book, version))
	if err != nil {
		return nil, err
	}
	var book struct {
		Tree openstaxNode `json:"tree"`
	}
	if err := json.Unmarshal(data, &book); err != nil {
		return nil, fmt.Errorf("book tree: %w", err)
	}
	var out []Section
	var walk func(n openstaxNode, level int)
	walk = func(n openstaxNode, level int) {
		if len(n.Contents) == 0 {
			id := n.ID
			if i := strings.Index(id, "@"); i > 0 {
				id = id[:i]
			}
			out = append(out, Section{Locator: id, Title: plainTitle(n.Title), Level: level})
			return
		}
		for _, ch := range n.Contents {
			walk(ch, level+1)
		}
	}
	for _, ch := range book.Tree.Contents {
		walk(ch, 1)
	}
	return out, nil
}

func (f openstaxFetcher) fetch(ctx context.Context, c *Client, s *Source, locator string) (fetched, error) {
	host, archive, version, err := f.archive(ctx, c, s)
	if err != nil {
		return fetched{}, err
	}
	url := fmt.Sprintf("%s%s/contents/%s@%s:%s.json", host, archive, s.Fetch.Book, version, locator)
	data, err := c.get(ctx, url)
	if err != nil {
		return fetched{}, err
	}
	var page struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(data, &page); err != nil {
		return fetched{}, fmt.Errorf("page: %w", err)
	}
	md := HTMLToMarkdown(page.Content)
	return fetched{title: plainTitle(page.Title), markdown: md, url: url}, nil
}
