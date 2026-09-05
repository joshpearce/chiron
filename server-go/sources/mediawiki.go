package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// mediawikiFetcher reads a Wikibooks-style book through the MediaWiki
// API: the book's front page lists its chapters as links under the
// book's prefix; each page comes back as parsed HTML.
type mediawikiFetcher struct{}

func (mediawikiFetcher) api(s *Source, params url.Values) string {
	site := base(Recipe{Base: s.Fetch.Base}, strings.TrimRight(s.Fetch.Site, "/"))
	params.Set("format", "json")
	params.Set("formatversion", "2")
	return site + "/w/api.php?" + params.Encode()
}

func (f mediawikiFetcher) contents(ctx context.Context, c *Client, s *Source) ([]Section, error) {
	data, err := c.get(ctx, f.api(s, url.Values{"action": {"parse"}, "page": {s.Fetch.Prefix}, "prop": {"links"}}))
	if err != nil {
		return nil, err
	}
	var resp struct {
		Parse struct {
			Links []struct {
				Title  string `json:"title"`
				NS     int    `json:"ns"`
				Exists bool   `json:"exists"`
			} `json:"links"`
		} `json:"parse"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("links: %w", err)
	}
	prefix := s.Fetch.Prefix + "/"
	seen := map[string]bool{}
	var out []Section
	for _, l := range resp.Parse.Links {
		if l.NS != 0 || !l.Exists || !strings.HasPrefix(l.Title, prefix) || seen[l.Title] {
			continue
		}
		seen[l.Title] = true
		rest := strings.TrimPrefix(l.Title, prefix)
		out = append(out, Section{Locator: l.Title, Title: strings.ReplaceAll(rest, "_", " "), Level: strings.Count(rest, "/") + 1})
	}
	return out, nil
}

func (f mediawikiFetcher) fetch(ctx context.Context, c *Client, s *Source, locator string) (fetched, error) {
	u := f.api(s, url.Values{"action": {"parse"}, "page": {locator}, "prop": {"text|displaytitle"}, "disableeditsection": {"1"}})
	data, err := c.get(ctx, u)
	if err != nil {
		return fetched{}, err
	}
	var resp struct {
		Parse struct {
			Title        string `json:"title"`
			DisplayTitle string `json:"displaytitle"`
			Text         string `json:"text"`
		} `json:"parse"`
		Error struct {
			Info string `json:"info"`
		} `json:"error"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return fetched{}, fmt.Errorf("parse: %w", err)
	}
	if resp.Error.Info != "" {
		return fetched{}, fmt.Errorf("%s", resp.Error.Info)
	}
	title := plainTitle(resp.Parse.DisplayTitle)
	if title == "" {
		title = resp.Parse.Title
	}
	site := base(Recipe{Base: s.Fetch.Base}, strings.TrimRight(s.Fetch.Site, "/"))
	return fetched{title: title, markdown: HTMLToMarkdown(resp.Parse.Text), url: site + "/wiki/" + url.PathEscape(locator)}, nil
}
