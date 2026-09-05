package sources

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// gutenbergFetcher reads a Project Gutenberg text: the plain UTF-8 file,
// cut at the START and END markers so the licence boilerplate never
// reaches the author (it stays in the provenance), and split into
// chapters at their headings. A locator is a chapter's number in that
// split, from 1.
type gutenbergFetcher struct{}

var (
	pgStart = regexp.MustCompile(`(?m)^\*\*\* ?START OF (THE|THIS) PROJECT GUTENBERG EBOOK.*$`)
	pgEnd   = regexp.MustCompile(`(?m)^\*\*\* ?END OF (THE|THIS) PROJECT GUTENBERG EBOOK.*$`)
)

func (f gutenbergFetcher) body(ctx context.Context, c *Client, s *Source) (string, string, error) {
	host := base(s.Fetch, "https://www.gutenberg.org")
	url := fmt.Sprintf("%s/ebooks/%d.txt.utf-8", host, s.Fetch.ID)
	data, err := c.get(ctx, url)
	if err != nil {
		return "", "", err
	}
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	if m := pgStart.FindStringIndex(text); m != nil {
		text = text[m[1]:]
	}
	if m := pgEnd.FindStringIndex(text); m != nil {
		text = text[:m[0]]
	}
	return strings.TrimSpace(text), url, nil
}

// A heading in the text: where it starts, where its body starts, its title.
type heading struct {
	title    string
	at, body int
}

var chapterStyle = regexp.MustCompile(`^(?:CHAPTER|Chapter|BOOK|Book|PART|Part|LETTER|Letter|SECTION|Section)\s+[IVXLCDM0-9]+\b`)
var capsLine = regexp.MustCompile(`^[A-Z][A-Z0-9 ,.'\-:;]{3,70}$`)

// headings finds the chapters. Chapter-style headings ("CHAPTER I.")
// win when the text has them; a table of contents lists them one after
// another and is skipped, since a heading in the body is followed by its
// title or its prose, not by another heading. A capitalised line right
// after a chapter heading is its title and joins it. Without chapter
// headings, standalone capitalised lines are the chapters, and the ones
// with almost nothing under them (a title page) fold into their neighbour.
func headings(lines []string) []heading {
	next := func(i int) int {
		for j := i + 1; j < len(lines); j++ {
			if strings.TrimSpace(lines[j]) != "" {
				return j
			}
		}
		return -1
	}
	var out []heading
	for i, line := range lines {
		t := strings.TrimSpace(line)
		if !chapterStyle.MatchString(t) {
			continue
		}
		if i > 0 && strings.TrimSpace(lines[i-1]) != "" {
			continue
		}
		n := next(i)
		if n >= 0 && chapterStyle.MatchString(strings.TrimSpace(lines[n])) {
			continue // a contents entry
		}
		h := heading{title: t, at: i, body: i + 1}
		if n == i+1 && capsLine.MatchString(strings.TrimSpace(lines[n])) && !chapterStyle.MatchString(strings.TrimSpace(lines[n])) {
			h.title = t + " " + strings.TrimSpace(lines[n])
			h.body = n + 1
		}
		out = append(out, h)
	}
	if len(out) >= 2 {
		// A synopsis section repeats every heading before the body does;
		// the last occurrence of a title is the chapter itself.
		last := map[string]int{}
		for i, h := range out {
			last[h.title] = i
		}
		var body []heading
		for i, h := range out {
			if last[h.title] == i {
				body = append(body, h)
			}
		}
		return body
	}
	out = nil
	for i, line := range lines {
		t := strings.TrimSpace(line)
		if t == "" || !capsLine.MatchString(t) {
			continue
		}
		before := i == 0 || strings.TrimSpace(lines[i-1]) == ""
		after := i+1 >= len(lines) || strings.TrimSpace(lines[i+1]) == ""
		if before && after {
			out = append(out, heading{title: t, at: i, body: i + 1})
		}
	}
	// Fold headings with almost nothing under them into the next one.
	var kept []heading
	for i, h := range out {
		end := len(lines)
		if i+1 < len(out) {
			end = out[i+1].at
		}
		if len(strings.TrimSpace(strings.Join(lines[h.body:end], "\n"))) < 400 && i+1 < len(out) {
			continue
		}
		kept = append(kept, h)
	}
	return kept
}

func chapters(text string) []Section {
	var out []Section
	for i, h := range headings(strings.Split(text, "\n")) {
		out = append(out, Section{Locator: strconv.Itoa(i + 1), Title: h.title, Level: 1})
	}
	return out
}

func (f gutenbergFetcher) contents(ctx context.Context, c *Client, s *Source) ([]Section, error) {
	text, _, err := f.body(ctx, c, s)
	if err != nil {
		return nil, err
	}
	out := chapters(text)
	if len(out) == 0 {
		out = []Section{{Locator: "1", Title: s.Title, Level: 1}}
	}
	return out, nil
}

func (f gutenbergFetcher) fetch(ctx context.Context, c *Client, s *Source, locator string) (fetched, error) {
	text, url, err := f.body(ctx, c, s)
	if err != nil {
		return fetched{}, err
	}
	n, err := strconv.Atoi(locator)
	if err != nil || n < 1 {
		return fetched{}, fmt.Errorf("locator %q is not a chapter number", locator)
	}
	lines := strings.Split(text, "\n")
	hs := headings(lines)
	if len(hs) == 0 {
		if n != 1 {
			return fetched{}, fmt.Errorf("the text has no chapter %d", n)
		}
		return fetched{title: s.Title, markdown: text + "\n", url: url}, nil
	}
	if n > len(hs) {
		return fetched{}, fmt.Errorf("the text has %d chapters, not %d", len(hs), n)
	}
	h := hs[n-1]
	end := len(lines)
	if n < len(hs) {
		end = hs[n].at
	}
	body := strings.TrimSpace(strings.Join(lines[h.body:end], "\n"))
	return fetched{title: h.title, markdown: "# " + h.title + "\n\n" + body + "\n", url: url}, nil
}
