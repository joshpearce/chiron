package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var noDerivatives = regexp.MustCompile(`by-nc-nd|by-nd|noderiv|no derivatives`)

// pressbooksFetcher reads a book on any Pressbooks network through its
// REST API: the table of contents, then one chapter at a time, with the
// chapter's own licence read from its metadata (rule 5).
type pressbooksFetcher struct{}

type pressbooksChapter struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	Slug  string `json:"slug"`
}

func (pressbooksFetcher) contents(ctx context.Context, c *Client, s *Source) ([]Section, error) {
	book := strings.TrimRight(s.Fetch.URL, "/")
	data, err := c.get(ctx, book+"/wp-json/pressbooks/v2/toc")
	if err != nil {
		return nil, err
	}
	var toc struct {
		FrontMatter []pressbooksChapter `json:"front-matter"`
		Parts       []struct {
			Title    string              `json:"title"`
			Chapters []pressbooksChapter `json:"chapters"`
		} `json:"parts"`
	}
	if err := json.Unmarshal(data, &toc); err != nil {
		return nil, fmt.Errorf("toc: %w", err)
	}
	var out []Section
	for _, p := range toc.Parts {
		for _, ch := range p.Chapters {
			title := plainTitle(ch.Title)
			if p.Title != "" {
				title = plainTitle(p.Title) + ": " + title
			}
			out = append(out, Section{Locator: strconv.Itoa(ch.ID), Title: title, Level: 2})
		}
	}
	return out, nil
}

func (pressbooksFetcher) fetch(ctx context.Context, c *Client, s *Source, locator string) (fetched, error) {
	book := strings.TrimRight(s.Fetch.URL, "/")
	url := book + "/wp-json/pressbooks/v2/chapters/" + locator
	data, err := c.get(ctx, url)
	if err != nil {
		return fetched{}, err
	}
	var ch struct {
		Title struct {
			Rendered string `json:"rendered"`
		} `json:"title"`
		Content struct {
			Rendered string `json:"rendered"`
		} `json:"content"`
		Link     string `json:"link"`
		Metadata struct {
			License struct {
				URL  string `json:"url"`
				Name string `json:"name"`
			} `json:"license"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(data, &ch); err != nil {
		return fetched{}, fmt.Errorf("chapter: %w", err)
	}
	out := fetched{title: plainTitle(ch.Title.Rendered), markdown: HTMLToMarkdown(ch.Content.Rendered), url: ch.Link}
	if out.url == "" {
		out.url = url
	}
	if lic := ch.Metadata.License; lic.Name != "" || lic.URL != "" {
		out.note = "chapter licence: " + lic.Name + " " + lic.URL
		if noDerivatives.MatchString(strings.ToLower(lic.URL + " " + lic.Name)) {
			out.verdict = QuoteOnly
		}
	}
	return out, nil
}
