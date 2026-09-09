package sources

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// fileFetcher reads a document kept on the server's own disk: material
// the reader has a copy of beside the corpus. A PDF goes through
// pdftotext and is split into chapters at the pages that open with a
// chapter heading (a book's "CHAPTER", its number, its title); an EPUB is
// split at its spine, which is where its own reader would turn the page;
// a Markdown or text file is split at its headings. The index entry
// records the licence it was read under, as for any other source.
type fileFetcher struct{}

func (fileFetcher) contents(ctx context.Context, c *Client, s *Source) ([]Section, error) {
	if s.Fetch.Path == "" {
		return nil, fmt.Errorf("%s: file source needs a path", s.ID)
	}
	if strings.EqualFold(filepath.Ext(s.Fetch.Path), ".pdf") {
		pages, err := pdfPages(s.Fetch.Path)
		if err != nil {
			return nil, err
		}
		return pdfChapters(pages), nil
	}
	if strings.EqualFold(filepath.Ext(s.Fetch.Path), ".epub") {
		book, err := openEPUB(s.Fetch.Path)
		if err != nil {
			return nil, err
		}
		return book.sections(), nil
	}
	data, err := os.ReadFile(s.Fetch.Path)
	if err != nil {
		return nil, err
	}
	var out []Section
	for _, h := range mdHeadings(string(data)) {
		out = append(out, Section{Locator: h.slug, Title: h.title})
	}
	if len(out) == 0 {
		out = []Section{{Locator: "all", Title: strings.TrimSuffix(filepath.Base(s.Fetch.Path), filepath.Ext(s.Fetch.Path))}}
	}
	return out, nil
}

func (fileFetcher) fetch(ctx context.Context, c *Client, s *Source, locator string) (fetched, error) {
	url := "file://" + s.Fetch.Path
	if strings.EqualFold(filepath.Ext(s.Fetch.Path), ".epub") {
		book, err := openEPUB(s.Fetch.Path)
		if err != nil {
			return fetched{}, err
		}
		title, md, err := book.chapter(locator)
		if err != nil {
			return fetched{}, fmt.Errorf("%s: %w", s.Fetch.Path, err)
		}
		return fetched{title: title, markdown: md, url: url + "#" + locator}, nil
	}
	if strings.EqualFold(filepath.Ext(s.Fetch.Path), ".pdf") {
		from, to, err := pageRange(locator)
		if err != nil {
			return fetched{}, err
		}
		pages, err := pdfPages(s.Fetch.Path, from, to)
		if err != nil {
			return fetched{}, err
		}
		title, body := chapterOpening(pages)
		if title == "" {
			title = fmt.Sprintf("Pages %d-%d", from, to)
		}
		return fetched{title: title, markdown: "# " + title + "\n\n" + pdfProse(body), url: url + "#" + locator}, nil
	}
	data, err := os.ReadFile(s.Fetch.Path)
	if err != nil {
		return fetched{}, err
	}
	md := string(data)
	if locator == "all" {
		title := firstHeading(md)
		if title == "" {
			title = strings.TrimSuffix(filepath.Base(s.Fetch.Path), filepath.Ext(s.Fetch.Path))
		}
		return fetched{title: title, markdown: md, url: url}, nil
	}
	for _, h := range mdHeadings(md) {
		if h.slug == locator {
			return fetched{title: h.title, markdown: h.text, url: url + "#" + locator}, nil
		}
	}
	return fetched{}, fmt.Errorf("no section %q in %s", locator, s.Fetch.Path)
}

// pdfPages is the text of a PDF's pages, all of them or from..to
// (1-based, inclusive), one string per page.
func pdfPages(path string, span ...int) ([]string, error) {
	if _, err := exec.LookPath("pdftotext"); err != nil {
		return nil, fmt.Errorf("pdftotext is not installed; PDF sources need poppler-utils")
	}
	args := []string{"-enc", "UTF-8"}
	if len(span) == 2 {
		args = append(args, "-f", strconv.Itoa(span[0]), "-l", strconv.Itoa(span[1]))
	}
	args = append(args, path, "-")
	out, err := exec.Command("pdftotext", args...).Output()
	if err != nil {
		return nil, fmt.Errorf("pdftotext %s: %w", path, err)
	}
	pages := strings.Split(string(out), "\f")
	if n := len(pages); n > 0 && strings.TrimSpace(pages[n-1]) == "" {
		pages = pages[:n-1] // the form feed after the last page
	}
	return pages, nil
}

var chapterWord = regexp.MustCompile(`^(?:CHAPTER|Chapter|PART|Part)(?:\s+([0-9]+|[IVXLC]+))?$`)
var chapterNumber = regexp.MustCompile(`^([0-9]+|[IVXLC]+)$`)

// chapterOpening reads a chapter heading off the top of a run of pages:
// "CHAPTER" and its number on their own lines (or together), then the
// title on one or more lines up to the drop capital or the first line of
// prose. It returns the title and the pages with the heading removed;
// no heading, and the pages come back whole.
func chapterOpening(pages []string) (string, []string) {
	if len(pages) == 0 {
		return "", pages
	}
	lines := strings.Split(pages[0], "\n")
	next := func(i int) int {
		for ; i < len(lines); i++ {
			if strings.TrimSpace(lines[i]) != "" {
				return i
			}
		}
		return -1
	}
	i := next(0)
	if i < 0 {
		return "", pages
	}
	m := chapterWord.FindStringSubmatch(strings.TrimSpace(lines[i]))
	if m == nil {
		return "", pages
	}
	i++
	if m[1] == "" {
		// The number on its own line.
		if i = next(i); i < 0 || !chapterNumber.MatchString(strings.TrimSpace(lines[i])) {
			return "", pages
		}
		i++
	}
	// The title: one or more lines, ending at a blank line, the drop
	// capital, or a line that reads as prose.
	var title []string
	if i = next(i); i < 0 {
		return "", pages
	}
	for ; i < len(lines) && len(title) < 4; i++ {
		t := strings.TrimSpace(lines[i])
		if t == "" || isDropCap(t) || looksLikeProse(t, len(title) > 0) {
			break
		}
		title = append(title, t)
	}
	if len(title) == 0 {
		return "", pages
	}
	rest := append([]string{strings.Join(lines[i:], "\n")}, pages[1:]...)
	return strings.Join(title, " "), rest
}

func isDropCap(line string) bool {
	r := []rune(line)
	return len(r) == 1 && unicode.IsUpper(r[0])
}

// looksLikeProse: a title line is short and does not end mid-sentence,
// and the first of them is capitalised (a title's second line may run
// on in lowercase: "Expected Returns and Risks / from Direct Lending").
func looksLikeProse(line string, continuation bool) bool {
	return len(line) > 60 || strings.HasSuffix(line, ",") || strings.HasSuffix(line, ".") ||
		(!continuation && len(line) > 0 && unicode.IsLower([]rune(line)[0]))
}

// pdfChapters lists a book's chapters by the pages that open with a
// heading; the pages before the first are the front matter. A book with
// no such headings is listed in runs of twelve pages.
func pdfChapters(pages []string) []Section {
	var starts []int
	var titles []string
	for i := range pages {
		if title, _ := chapterOpening(pages[i : i+1]); title != "" {
			starts = append(starts, i)
			titles = append(titles, title)
		}
	}
	var out []Section
	if len(starts) == 0 {
		const run = 12
		for from := 0; from < len(pages); from += run {
			to := min(from+run, len(pages))
			out = append(out, Section{Locator: fmt.Sprintf("p%d-%d", from+1, to), Title: fmt.Sprintf("Pages %d-%d", from+1, to)})
		}
		return out
	}
	if starts[0] > 0 {
		out = append(out, Section{Locator: fmt.Sprintf("p1-%d", starts[0]), Title: "Front matter"})
	}
	for k, from := range starts {
		to := len(pages)
		if k+1 < len(starts) {
			to = starts[k+1]
		}
		out = append(out, Section{Locator: fmt.Sprintf("p%d-%d", from+1, to), Title: titles[k]})
	}
	return out
}

var pageSpan = regexp.MustCompile(`^p([0-9]+)-([0-9]+)$`)

func pageRange(locator string) (int, int, error) {
	m := pageSpan.FindStringSubmatch(locator)
	if m == nil {
		return 0, 0, fmt.Errorf("locator %q is not a page range like p12-30", locator)
	}
	from, _ := strconv.Atoi(m[1])
	to, _ := strconv.Atoi(m[2])
	if from < 1 || to < from {
		return 0, 0, fmt.Errorf("locator %q: bad page range", locator)
	}
	return from, to, nil
}

var pageNumber = regexp.MustCompile(`^[0-9]{1,4}$`)

// pdfProse turns pages of extracted text into one run: page numbers at a
// page's edges dropped, a drop capital rejoined to its word, soft
// hyphens removed, pages joined without a break in the paragraph.
func pdfProse(pages []string) string {
	var out []string
	for _, page := range pages {
		lines := strings.Split(strings.ReplaceAll(page, "­", ""), "\n")
		for len(lines) > 0 && (strings.TrimSpace(lines[0]) == "" || pageNumber.MatchString(strings.TrimSpace(lines[0]))) {
			lines = lines[1:]
		}
		for len(lines) > 0 && (strings.TrimSpace(lines[len(lines)-1]) == "" || pageNumber.MatchString(strings.TrimSpace(lines[len(lines)-1]))) {
			lines = lines[:len(lines)-1]
		}
		for i := 0; i < len(lines); i++ {
			t := strings.TrimSpace(lines[i])
			if isDropCap(t) && i+1 < len(lines) {
				lines[i+1] = t + strings.TrimLeft(lines[i+1], " ")
				continue
			}
			out = append(out, lines[i])
		}
	}
	return tidy(strings.Join(out, "\n"))
}

// A heading in a Markdown file, with the text under it to the next
// heading of the same or a higher level.
type mdHeading struct {
	title, slug, text string
}

var mdHeadingLine = regexp.MustCompile(`^(#{1,2})\s+(.+?)\s*$`)

func mdHeadings(md string) []mdHeading {
	lines := strings.Split(md, "\n")
	var out []mdHeading
	var start []int
	for i, l := range lines {
		if m := mdHeadingLine.FindStringSubmatch(l); m != nil {
			out = append(out, mdHeading{title: m[2], slug: slugify(m[2])})
			start = append(start, i)
		}
	}
	for k := range out {
		end := len(lines)
		if k+1 < len(out) {
			end = start[k+1]
		}
		out[k].text = strings.Join(lines[start[k]:end], "\n")
	}
	return out
}

func slugify(s string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(s) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			dash = false
		case !dash && b.Len() > 0:
			b.WriteByte('-')
			dash = true
		}
	}
	return strings.TrimRight(b.String(), "-")
}
