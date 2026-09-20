package sources

import (
	"context"
	"fmt"
	"math"
	"net/url"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// A page the reader sends to Chiron: a blog post, a documentation page,
// anything with an article on it. What comes back is the writer's own
// words as Markdown, so the reader reads the post and not the site: the
// navigation, the masthead and the footer are left behind, the links and
// the pictures point at where they really are, and the pictures are
// listed so they can be kept beside the page.

// Page is one web page as it will be read.
type Page struct {
	URL      string
	Title    string
	Markdown string
	// Images are the pictures the page shows, absolute, in the order
	// they appear.
	Images []string
}

// An article shorter than this is a teaser, a caption or a sidebar, not
// the page's own body.
const leastArticle = 25

// Page fetches a URL and reads the article on it.
func (c *Client) Page(ctx context.Context, rawURL string) (*Page, error) {
	data, err := c.Get(ctx, rawURL)
	if err != nil {
		return nil, err
	}
	return ParsePage(rawURL, string(data))
}

// Get fetches one URL through the client's cache and politeness, for a
// caller that has its own use for the bytes.
func (c *Client) Get(ctx context.Context, rawURL string) ([]byte, error) {
	return c.get(ctx, rawURL)
}

// ParsePage reads the article out of a page already fetched from rawURL.
func ParsePage(rawURL, src string) (*Page, error) {
	base, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	doc, err := html.Parse(strings.NewReader(src))
	if err != nil {
		return nil, err
	}
	if href := baseHref(doc); href != "" {
		if u, err := base.Parse(href); err == nil {
			base = u
		}
	}
	absolutize(doc, base)
	body := articleOf(doc)
	if body == nil {
		return nil, fmt.Errorf("this page has no article on it")
	}
	md := strings.TrimSpace(HTMLFragmentToMarkdown(body))
	if words(md) < leastArticle {
		return nil, fmt.Errorf("this page has no article on it")
	}
	title := pageTitle(doc, body)
	return &Page{URL: rawURL, Title: title, Markdown: withoutTitle(md, title), Images: imagesIn(body)}, nil
}

func words(s string) int { return len(strings.Fields(s)) }

// withoutTitle drops the heading the article opens with when it is the
// title: the piece is titled where it is read, at whatever level the
// site chose to write it.
func withoutTitle(md, title string) string {
	line, rest, _ := strings.Cut(md, "\n")
	heading := strings.TrimLeft(line, "#")
	if len(heading) == len(line) {
		return md
	}
	if !strings.EqualFold(strings.Join(strings.Fields(heading), " "), strings.Join(strings.Fields(title), " ")) {
		return md
	}
	return strings.TrimLeft(rest, "\n")
}

// articleOf is the element the page's own words are in. Few sites mark
// it up, so it is found the way a person finds it: the container holding
// the most prose, discounted for being mostly links, favoured for saying
// it is the article and against for saying it is the furniture.
func articleOf(doc *html.Node) *html.Node {
	var best *html.Node
	bestScore := 0.0
	for _, c := range candidates(doc) {
		if s := c.score; s > bestScore {
			best, bestScore = c.node, s
		}
	}
	if best == nil {
		best = elementOf(doc, atom.Body)
	}
	if best != nil {
		prune(best)
	}
	return best
}

type candidate struct {
	node  *html.Node
	score float64
}

// candidates scores every container on the page by the prose in it, in
// document order so that a tie goes to the one written first.
func candidates(doc *html.Node) []candidate {
	scores := map[*html.Node]float64{}
	var order []*html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			if container(n) {
				if _, seen := scores[n]; !seen {
					scores[n] = 0
					order = append(order, n)
				}
			}
			if prose(n) {
				t := strings.TrimSpace(text(n))
				// A line of prose, not a caption or a button's label.
				if len(t) >= 25 {
					s := 1 + math.Min(float64(len(t))/100, 3) + float64(strings.Count(t, ","))
					if p := containerOf(n); p != nil {
						add(scores, &order, p, s)
						if g := containerOf(p); g != nil {
							add(scores, &order, g, s/2)
						}
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	out := make([]candidate, 0, len(order))
	for _, n := range order {
		s := scores[n] * (1 - linkDensity(n))
		switch {
		case chromeName(n):
			s *= 0.2
		case n.DataAtom == atom.Article, n.DataAtom == atom.Main, attr(n, "role") == "main", contentName(n):
			s *= 1.5
		}
		out = append(out, candidate{node: n, score: s})
	}
	return out
}

func add(scores map[*html.Node]float64, order *[]*html.Node, n *html.Node, s float64) {
	if _, seen := scores[n]; !seen {
		*order = append(*order, n)
	}
	scores[n] += s
}

func container(n *html.Node) bool {
	switch n.DataAtom {
	case atom.Div, atom.Section, atom.Article, atom.Main, atom.Td, atom.Body:
		return true
	}
	return attr(n, "role") == "main"
}

func prose(n *html.Node) bool {
	switch n.DataAtom {
	case atom.P, atom.Pre, atom.Blockquote:
		return true
	}
	return false
}

func containerOf(n *html.Node) *html.Node {
	for p := n.Parent; p != nil; p = p.Parent {
		if p.Type == html.ElementNode && container(p) {
			return p
		}
	}
	return nil
}

// linkDensity is how much of a container is link text: a list of other
// posts is nearly all of it, a piece of writing barely any.
func linkDensity(n *html.Node) float64 {
	all := len(strings.TrimSpace(text(n)))
	if all == 0 {
		return 1
	}
	links := 0
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.DataAtom == atom.A {
			links += len(strings.TrimSpace(text(n)))
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	if d := float64(links) / float64(all); d < 1 {
		return d
	}
	return 1
}

// What a site calls its parts is the best clue it gives: these are the
// words it uses for the piece, and the words it uses for everything
// around the piece.
var chromeWords = []string{"comment", "sidebar", "footer", "nav", "menu", "related", "recent",
	"share", "social", "promo", "sponsor", "banner", "subscribe", "newsletter", "breadcrumb",
	"pagination", "metabox", "masthead", "widget", "popular", "archive"}

var contentWords = []string{"article", "entry", "post", "content", "story", "blogmark",
	"markdown", "prose", "postbody", "hentry"}

func named(n *html.Node, words []string) bool {
	name := strings.ToLower(classOf(n) + " " + attr(n, "id"))
	for _, w := range words {
		if strings.Contains(name, w) {
			return true
		}
	}
	return false
}

func chromeName(n *html.Node) bool  { return named(n, chromeWords) }
func contentName(n *html.Node) bool { return named(n, contentWords) }

// prune drops what the page put inside the article that is not the
// article: the sponsor's line, the tags, the box of recent posts.
func prune(n *html.Node) {
	var next *html.Node
	for c := n.FirstChild; c != nil; c = next {
		next = c.NextSibling
		if c.Type == html.ElementNode && chromeName(c) && !contentName(c) {
			n.RemoveChild(c)
			continue
		}
		prune(c)
	}
}

func elementOf(n *html.Node, a atom.Atom) *html.Node {
	if n.Type == html.ElementNode && n.DataAtom == a {
		return n
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if found := elementOf(c, a); found != nil {
			return found
		}
	}
	return nil
}

func baseHref(doc *html.Node) string {
	if n := elementOf(doc, atom.Base); n != nil {
		return attr(n, "href")
	}
	return ""
}

// absolutize points every link and picture at where it really is, since
// the page will be read from somewhere else entirely.
func absolutize(n *html.Node, base *url.URL) {
	if n.Type == html.ElementNode {
		key := ""
		switch n.DataAtom {
		case atom.A:
			key = "href"
		case atom.Img:
			key = "src"
		}
		if key != "" {
			for i, a := range n.Attr {
				if a.Key != key {
					continue
				}
				if u, err := base.Parse(strings.TrimSpace(a.Val)); err == nil {
					n.Attr[i].Val = u.String()
				}
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		absolutize(c, base)
	}
}

// pageTitle is what the writer called the piece: the heading over it,
// then what the page tells a reader it is called, then the browser tab's
// title, each with the site's name trimmed off the end.
func pageTitle(doc, body *html.Node) string {
	if h := headingIn(body); h != "" {
		return trimSite(h)
	}
	if t := metaContent(doc, "og:title"); t != "" {
		return trimSite(t)
	}
	if n := elementOf(doc, atom.Title); n != nil {
		return trimSite(strings.TrimSpace(text(n)))
	}
	return ""
}

func metaContent(n *html.Node, property string) string {
	if n.Type == html.ElementNode && n.DataAtom == atom.Meta {
		if attr(n, "property") == property || attr(n, "name") == property {
			return strings.TrimSpace(attr(n, "content"))
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if v := metaContent(c, property); v != "" {
			return v
		}
	}
	return ""
}

func headingIn(n *html.Node) string {
	if n.Type == html.ElementNode && n.DataAtom == atom.H1 {
		return strings.TrimSpace(text(n))
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if h := headingIn(c); h != "" {
			return h
		}
	}
	return ""
}

// trimSite drops the site's name where a page's title carries it after a
// separator, which is how most of them are written. The name comes last
// and is the shorter half; a separator inside the title itself is not
// one, and is left alone.
func trimSite(title string) string {
	cut, sep := -1, ""
	for _, s := range []string{" | ", " - ", " – ", " — ", " :: ", " · "} {
		if i := strings.LastIndex(title, s); i > cut {
			cut, sep = i, s
		}
	}
	if cut <= 0 {
		return title
	}
	head := strings.TrimSpace(title[:cut])
	tail := strings.TrimSpace(title[cut+len(sep):])
	if len(tail) < len(head) {
		return head
	}
	return title
}

func imagesIn(n *html.Node) []string {
	var out []string
	seen := map[string]bool{}
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.DataAtom == atom.Img {
			src := strings.TrimSpace(attr(n, "src"))
			if u, err := url.Parse(src); err == nil && (u.Scheme == "http" || u.Scheme == "https") && !seen[src] {
				seen[src] = true
				out = append(out, src)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return out
}
