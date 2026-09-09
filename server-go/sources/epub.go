package sources

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"path"
	"strings"

	"golang.org/x/net/html"
)

// An EPUB the reader owns, read the way its own reader would: the
// container names the package, the package's spine gives the chapters in
// reading order, and the navigation document names them. A chapter is one
// spine document, which is how EPUBs are almost always cut.
type epubBook struct {
	title string
	files map[string]string
	spine []epubItem
}

type epubItem struct {
	path  string // inside the archive, e.g. OEBPS/text/ch01.xhtml
	title string
}

// Only the text is read: an EPUB carries its fonts and pictures too, and
// a book's worth of those has no business in memory.
func epubText(name string) bool {
	switch strings.ToLower(path.Ext(name)) {
	case ".xhtml", ".html", ".htm", ".xml", ".opf", ".ncx", ".txt":
		return true
	}
	return false
}

func openEPUB(file string) (*epubBook, error) {
	z, err := zip.OpenReader(file)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", file, err)
	}
	defer z.Close()
	b := &epubBook{files: map[string]string{}}
	for _, f := range z.File {
		if f.FileInfo().IsDir() || !epubText(f.Name) {
			continue
		}
		r, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("%s: %s: %w", file, f.Name, err)
		}
		data, err := io.ReadAll(r)
		r.Close()
		if err != nil {
			return nil, fmt.Errorf("%s: %s: %w", file, f.Name, err)
		}
		b.files[f.Name] = string(data)
	}

	var container struct {
		Rootfiles []struct {
			Path string `xml:"full-path,attr"`
		} `xml:"rootfiles>rootfile"`
	}
	src, ok := b.files["META-INF/container.xml"]
	if !ok {
		return nil, fmt.Errorf("%s: not an EPUB (no META-INF/container.xml)", file)
	}
	if err := xml.Unmarshal([]byte(src), &container); err != nil || len(container.Rootfiles) == 0 {
		return nil, fmt.Errorf("%s: container names no package", file)
	}
	opfPath := container.Rootfiles[0].Path
	opf, ok := b.files[opfPath]
	if !ok {
		return nil, fmt.Errorf("%s: package %s is missing", file, opfPath)
	}

	var pkg struct {
		Title string `xml:"metadata>title"`
		Items []struct {
			ID         string `xml:"id,attr"`
			Href       string `xml:"href,attr"`
			MediaType  string `xml:"media-type,attr"`
			Properties string `xml:"properties,attr"`
		} `xml:"manifest>item"`
		Spine []struct {
			IDRef string `xml:"idref,attr"`
		} `xml:"spine>itemref"`
	}
	if err := xml.Unmarshal([]byte(opf), &pkg); err != nil {
		return nil, fmt.Errorf("%s: %w", opfPath, err)
	}
	b.title = strings.TrimSpace(pkg.Title)

	base := path.Dir(opfPath)
	href := map[string]string{}
	var navPath, ncxPath string
	for _, it := range pkg.Items {
		full := path.Join(base, it.Href)
		href[it.ID] = full
		if strings.Contains(it.Properties, "nav") {
			navPath = full
		}
		if it.MediaType == "application/x-dtbncx+xml" {
			ncxPath = full
		}
	}
	titles := b.navTitles(navPath, ncxPath)

	titled := false
	for _, ref := range pkg.Spine {
		p, ok := href[ref.IDRef]
		if !ok {
			continue
		}
		title := titles[p]
		if title == "" {
			title = firstHTMLHeading(b.files[p])
		}
		switch {
		case title != "":
			titled = true
		case !titled:
			// What comes before the first named chapter is the cover,
			// the copyright page, the dedication: front matter.
			title = "Front matter"
		default:
			title = strings.TrimSuffix(path.Base(p), path.Ext(p))
		}
		b.spine = append(b.spine, epubItem{path: p, title: title})
	}
	if len(b.spine) == 0 {
		return nil, fmt.Errorf("%s: the package has no spine", file)
	}
	return b, nil
}

// navTitles maps a document's path to the title the book gives it: the
// EPUB 3 navigation document if there is one, else the EPUB 2 NCX.
func (b *epubBook) navTitles(navPath, ncxPath string) map[string]string {
	out := map[string]string{}
	if src, ok := b.files[navPath]; ok {
		doc, err := html.Parse(strings.NewReader(src))
		if err == nil {
			var toc *html.Node
			var find func(*html.Node)
			find = func(n *html.Node) {
				if toc != nil {
					return
				}
				if n.Type == html.ElementNode && n.Data == "nav" {
					// The table of contents, not the landmarks or the page list.
					if t := attr(n, "epub:type"); t == "" || strings.Contains(t, "toc") {
						toc = n
						return
					}
				}
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					find(c)
				}
			}
			find(doc)
			if toc == nil {
				toc = doc
			}
			var walk func(*html.Node)
			walk = func(n *html.Node) {
				if n.Type == html.ElementNode && n.Data == "a" {
					if h := attr(n, "href"); h != "" {
						if title := strings.TrimSpace(text(n)); title != "" {
							out[path.Join(path.Dir(navPath), stripFragment(h))] = title
						}
					}
				}
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					walk(c)
				}
			}
			walk(toc)
		}
	}
	if len(out) > 0 {
		return out
	}
	if src, ok := b.files[ncxPath]; ok {
		var ncx struct {
			Points []struct {
				Label   string `xml:"navLabel>text"`
				Content struct {
					Src string `xml:"src,attr"`
				} `xml:"content"`
			} `xml:"navMap>navPoint"`
		}
		if xml.Unmarshal([]byte(src), &ncx) == nil {
			for _, p := range ncx.Points {
				if title := strings.TrimSpace(p.Label); title != "" && p.Content.Src != "" {
					out[path.Join(path.Dir(ncxPath), stripFragment(p.Content.Src))] = title
				}
			}
		}
	}
	return out
}

func stripFragment(href string) string {
	if i := strings.IndexByte(href, '#'); i >= 0 {
		return href[:i]
	}
	return href
}

// firstHTMLHeading is the text of a document's first heading, which names
// a chapter the navigation document left out.
func firstHTMLHeading(src string) string {
	if src == "" {
		return ""
	}
	doc, err := html.Parse(strings.NewReader(src))
	if err != nil {
		return ""
	}
	var found string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if found != "" {
			return
		}
		if n.Type == html.ElementNode && len(n.Data) == 2 && n.Data[0] == 'h' && n.Data[1] >= '1' && n.Data[1] <= '6' {
			if t := strings.TrimSpace(text(n)); t != "" {
				found = t
				return
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return found
}

func (b *epubBook) sections() []Section {
	out := make([]Section, 0, len(b.spine))
	for _, it := range b.spine {
		out = append(out, Section{Locator: it.path, Title: it.title})
	}
	return out
}

// chapter is one spine document as prose, titled as the book titles it.
func (b *epubBook) chapter(locator string) (title, markdown string, err error) {
	for _, it := range b.spine {
		if it.path != locator {
			continue
		}
		md := HTMLToMarkdown(b.files[it.path])
		if !strings.HasPrefix(strings.TrimSpace(md), "#") {
			md = "# " + it.title + "\n\n" + md
		}
		return it.title, md, nil
	}
	return "", "", fmt.Errorf("no section %q in the book", locator)
}
