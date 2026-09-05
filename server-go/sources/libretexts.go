package sources

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// libretextsFetcher reads a LibreTexts book page by page. Contents come
// from the links on the book's page and its chapter pages; a page's body
// is the first mt-content-container section; the page's own licence tag
// is read and, when it is stricter than the index says, the chunk's
// verdict follows the page (OPEN-SOURCES.md rule 5).
type libretextsFetcher struct{}

var licenceTag = regexp.MustCompile(`license:([a-z]+)`)

// licenceVerdict maps a LibreTexts tag to what it permits.
func licenceVerdict(tag string) (string, Verdict) {
	switch tag {
	case "ccby":
		return "CC BY", Adaptable
	case "ccbysa":
		return "CC BY-SA", Adaptable
	case "ccbync":
		return "CC BY-NC", Adaptable
	case "ccbyncsa":
		return "CC BY-NC-SA", Adaptable
	case "ccbynd", "ccbyncnd", "arr":
		return "CC BY-ND or all rights reserved", QuoteOnly
	case "publicdomain", "cc0":
		return "public domain", Adaptable
	case "gnufdl":
		return "GFDL", Adaptable
	}
	return tag, ""
}

func (libretextsFetcher) contents(ctx context.Context, c *Client, s *Source) ([]Section, error) {
	root := strings.TrimRight(s.Fetch.URL, "/")
	chapters, err := subpages(ctx, c, root, root)
	if err != nil {
		return nil, err
	}
	var out []Section
	for _, ch := range chapters {
		out = append(out, Section{Locator: ch.Locator, Title: ch.Title, Level: 1})
		sections, err := subpages(ctx, c, root, root+"/"+ch.Locator)
		if err != nil {
			continue
		}
		for _, sec := range sections {
			out = append(out, Section{Locator: sec.Locator, Title: sec.Title, Level: 2})
		}
	}
	return out, nil
}

// subpages lists the pages linked from one page that live directly below
// it, as locators relative to the book root.
func subpages(ctx context.Context, c *Client, root, page string) ([]Section, error) {
	data, err := c.get(ctx, page)
	if err != nil {
		return nil, err
	}
	doc, err := html.Parse(strings.NewReader(string(data)))
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var out []Section
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.DataAtom == atom.A {
			href := strings.TrimRight(attr(n, "href"), "/")
			if strings.HasPrefix(href, page+"/") {
				rest := strings.TrimPrefix(href, page+"/")
				if !strings.Contains(rest, "/") && !strings.Contains(rest, "?") && !strings.Contains(rest, "#") {
					loc := strings.TrimPrefix(href, root+"/")
					if !seen[loc] {
						seen[loc] = true
						title := strings.TrimSpace(text(n))
						if title == "" {
							title = rest
						}
						out = append(out, Section{Locator: loc, Title: title})
					}
				}
			}
		}
		for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
			walk(ch)
		}
	}
	walk(doc)
	return out, nil
}

func (libretextsFetcher) fetch(ctx context.Context, c *Client, s *Source, locator string) (fetched, error) {
	url := strings.TrimRight(s.Fetch.URL, "/") + "/" + locator
	data, err := c.get(ctx, url)
	if err != nil {
		return fetched{}, err
	}
	page := string(data)
	doc, err := html.Parse(strings.NewReader(page))
	if err != nil {
		return fetched{}, err
	}
	var body *html.Node
	var title string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			if body == nil && n.DataAtom == atom.Section && hasClass(n, "mt-content-container") {
				body = n
			}
			if title == "" && n.DataAtom == atom.H1 {
				title = strings.TrimSpace(text(n))
			}
		}
		for ch := n.FirstChild; ch != nil && (body == nil || title == ""); ch = ch.NextSibling {
			walk(ch)
		}
	}
	walk(doc)
	if body == nil {
		return fetched{}, fmt.Errorf("no mt-content-container on %s", url)
	}
	out := fetched{title: title, markdown: HTMLFragmentToMarkdown(body), url: url}
	if m := licenceTag.FindStringSubmatch(page); m != nil {
		name, verdict := licenceVerdict(m[1])
		out.note = "page licence tag: " + name
		if verdict == QuoteOnly {
			out.verdict = QuoteOnly
		}
	} else {
		out.note = "no licence tag on the page"
		out.verdict = QuoteOnly
	}
	return out, nil
}
