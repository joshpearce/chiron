package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// ocwFetcher reads an MIT OpenCourseWare course: written pages through
// their data.json (the content is an HTML fragment), and PDF resources
// (lecture notes, readings, problem sets) through pdftotext when it is
// installed. Contents are the course's pages, listed from its home page's
// navigation when the index does not list them.
type ocwFetcher struct{}

func (ocwFetcher) contents(ctx context.Context, c *Client, s *Source) ([]Section, error) {
	host := base(s.Fetch, "https://ocw.mit.edu")
	course := host + "/courses/" + strings.Trim(s.Fetch.Course, "/")
	data, err := c.get(ctx, course+"/")
	if err != nil {
		return nil, err
	}
	doc, err := html.Parse(strings.NewReader(string(data)))
	if err != nil {
		return nil, err
	}
	prefix := "/courses/" + strings.Trim(s.Fetch.Course, "/") + "/pages/"
	seen := map[string]bool{}
	var out []Section
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.DataAtom == atom.A {
			href := attr(n, "href")
			if strings.HasPrefix(href, prefix) {
				loc := strings.Trim(strings.TrimPrefix(href, prefix), "/")
				if loc != "" && !strings.Contains(loc, "/") && !seen[loc] {
					seen[loc] = true
					out = append(out, Section{Locator: loc, Title: strings.TrimSpace(text(n))})
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

func (ocwFetcher) fetch(ctx context.Context, c *Client, s *Source, locator string) (fetched, error) {
	host := base(s.Fetch, "https://ocw.mit.edu")
	course := host + "/courses/" + strings.Trim(s.Fetch.Course, "/")
	if strings.HasSuffix(strings.ToLower(locator), ".pdf") {
		url := locator
		if !strings.HasPrefix(url, "http") {
			url = course + "/" + strings.TrimLeft(locator, "/")
		}
		data, err := c.get(ctx, url)
		if err != nil {
			return fetched{}, err
		}
		text, err := pdfText(data)
		if err != nil {
			return fetched{}, err
		}
		title := strings.TrimSuffix(locator[strings.LastIndex(locator, "/")+1:], ".pdf")
		return fetched{title: title, markdown: text, url: url}, nil
	}
	url := course + "/pages/" + strings.Trim(locator, "/") + "/data.json"
	data, err := c.get(ctx, url)
	if err != nil {
		return fetched{}, err
	}
	var page struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(data, &page); err != nil {
		return fetched{}, fmt.Errorf("page data.json: %w", err)
	}
	return fetched{title: page.Title, markdown: HTMLToMarkdown(page.Content), url: course + "/pages/" + locator + "/"}, nil
}

// pdfText runs pdftotext, which is what the sprite installs for this;
// without it the fetch says so rather than guessing.
func pdfText(data []byte) (string, error) {
	if _, err := exec.LookPath("pdftotext"); err != nil {
		return "", fmt.Errorf("pdftotext is not installed; PDF sources need poppler-utils")
	}
	tmp, err := os.CreateTemp("", "chiron-*.pdf")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return "", err
	}
	tmp.Close()
	out, err := exec.Command("pdftotext", "-layout", "-enc", "UTF-8", tmp.Name(), "-").Output()
	if err != nil {
		return "", fmt.Errorf("pdftotext: %w", err)
	}
	return tidy(string(out)), nil
}
