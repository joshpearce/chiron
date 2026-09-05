package sources

import (
	"regexp"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// HTMLToMarkdown turns the HTML a textbook page serves into the Markdown
// the author reads. Structure survives (headings, paragraphs, lists, code,
// quotes, tables, images, links, emphasis); chrome goes (scripts, styles,
// navigation, headers, footers, asides); math comes out as the TeX the
// page carried in a KaTeX or MathJax annotation, in $...$ or $$...$$.
func HTMLToMarkdown(src string) string {
	doc, err := html.Parse(strings.NewReader(src))
	if err != nil {
		return src
	}
	w := &mdWriter{}
	w.walk(doc)
	return tidy(w.b.String())
}

// HTMLFragmentToMarkdown converts one element and its subtree.
func HTMLFragmentToMarkdown(n *html.Node) string {
	w := &mdWriter{}
	w.walk(n)
	return tidy(w.b.String())
}

type mdWriter struct {
	b strings.Builder
	// listStack holds the marker for each open list: "-" or "1.".
	listStack []string
	counters  []int
	pre       bool
}

var dropped = map[atom.Atom]bool{
	atom.Script: true, atom.Style: true, atom.Nav: true, atom.Header: true, atom.Footer: true,
	atom.Aside: true, atom.Noscript: true, atom.Head: true, atom.Title: true, atom.Iframe: true,
	atom.Button: true, atom.Form: true, atom.Svg: true,
}

var texAnnotation = regexp.MustCompile(`\\\((.*?)\\\)|\\\[(.*?)\\\]`)

func classOf(n *html.Node) string {
	for _, a := range n.Attr {
		if a.Key == "class" {
			return a.Val
		}
	}
	return ""
}

func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func hasClass(n *html.Node, want string) bool {
	for _, c := range strings.Fields(classOf(n)) {
		if c == want {
			return true
		}
	}
	return false
}

// text is the concatenated text of a subtree, untouched.
func text(n *html.Node) string {
	var b strings.Builder
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)
	return b.String()
}

// tex finds the TeX source a rendered-math element carries, if any.
func tex(n *html.Node) (string, bool) {
	var found string
	var f func(*html.Node) bool
	f = func(n *html.Node) bool {
		if n.Type == html.ElementNode && n.Data == "annotation" && strings.Contains(attr(n, "encoding"), "x-tex") {
			found = text(n)
			return true
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if f(c) {
				return true
			}
		}
		return false
	}
	return found, f(n)
}

func (w *mdWriter) newline() {
	s := w.b.String()
	if len(s) > 0 && !strings.HasSuffix(s, "\n") {
		w.b.WriteString("\n")
	}
}

func (w *mdWriter) blank() {
	w.newline()
	s := w.b.String()
	if len(s) > 0 && !strings.HasSuffix(s, "\n\n") {
		w.b.WriteString("\n")
	}
}

func (w *mdWriter) walk(n *html.Node) {
	switch n.Type {
	case html.TextNode:
		if w.pre {
			w.b.WriteString(n.Data)
			return
		}
		t := strings.ReplaceAll(n.Data, "\n", " ")
		t = regexp.MustCompile(`[ \t]+`).ReplaceAllString(t, " ")
		s := w.b.String()
		if strings.HasSuffix(s, "\n") || s == "" {
			t = strings.TrimLeft(t, " ")
		}
		w.b.WriteString(t)
		return
	case html.DocumentNode:
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			w.walk(c)
		}
		return
	case html.ElementNode:
	default:
		return
	}

	// Rendered math: KaTeX keeps the TeX in an annotation, MathJax in a
	// script, plain pages in \( \) or \[ \] delimiters.
	if n.DataAtom == atom.Script {
		if typ := attr(n, "type"); strings.HasPrefix(typ, "math/tex") {
			w.writeMath(strings.TrimSpace(text(n)), strings.Contains(typ, "display"))
		}
		return
	}
	if dropped[n.DataAtom] {
		return
	}
	if n.DataAtom == atom.Span && (hasClass(n, "math") || hasClass(n, "katex") || hasClass(n, "MathJax")) {
		if hasClass(n, "katex-html") || hasClass(n, "MathJax_Preview") {
			return
		}
		display := hasClass(n, "display")
		if t, ok := tex(n); ok {
			w.writeMath(strings.TrimSpace(t), display)
			return
		}
		raw := strings.TrimSpace(text(n))
		if m := texAnnotation.FindStringSubmatch(raw); m != nil {
			if m[1] != "" {
				w.writeMath(strings.TrimSpace(m[1]), false)
			} else {
				w.writeMath(strings.TrimSpace(m[2]), true)
			}
			return
		}
		if raw != "" {
			w.writeMath(raw, display)
		}
		return
	}
	if hasClass(n, "katex-html") || hasClass(n, "MathJax_Preview") {
		return
	}

	switch n.DataAtom {
	case atom.H1, atom.H2, atom.H3, atom.H4, atom.H5, atom.H6:
		level := int(n.Data[1] - '0')
		w.blank()
		w.b.WriteString(strings.Repeat("#", level) + " ")
		w.children(n)
		w.blank()
	case atom.P, atom.Div, atom.Section, atom.Article, atom.Main, atom.Body, atom.Html, atom.Figure, atom.Figcaption, atom.Details, atom.Summary, atom.Dl, atom.Dd, atom.Dt:
		block := n.DataAtom == atom.P || n.DataAtom == atom.Figcaption || n.DataAtom == atom.Dd || n.DataAtom == atom.Dt
		if block {
			w.blank()
		}
		w.children(n)
		if block {
			w.blank()
		}
	case atom.Br:
		w.newline()
	case atom.Hr:
		w.blank()
		w.b.WriteString("---")
		w.blank()
	case atom.Em, atom.I:
		w.b.WriteString("*")
		w.children(n)
		w.b.WriteString("*")
	case atom.Strong, atom.B:
		w.b.WriteString("**")
		w.children(n)
		w.b.WriteString("**")
	case atom.Code:
		if w.pre {
			w.children(n)
			return
		}
		w.b.WriteString("`")
		w.b.WriteString(strings.TrimSpace(text(n)))
		w.b.WriteString("`")
	case atom.Pre:
		w.blank()
		lang := ""
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.DataAtom == atom.Code {
				for _, cls := range strings.Fields(classOf(c)) {
					if strings.HasPrefix(cls, "language-") {
						lang = strings.TrimPrefix(cls, "language-")
					}
				}
			}
		}
		w.b.WriteString("```" + lang + "\n")
		w.pre = true
		w.children(n)
		w.pre = false
		w.newline()
		w.b.WriteString("```")
		w.blank()
	case atom.Blockquote:
		inner := &mdWriter{}
		inner.children(n)
		w.blank()
		for _, line := range strings.Split(strings.TrimSpace(tidy(inner.b.String())), "\n") {
			w.b.WriteString("> " + line + "\n")
		}
		w.blank()
	case atom.Ul, atom.Ol:
		marker := "-"
		if n.DataAtom == atom.Ol {
			marker = "1."
		}
		if len(w.listStack) == 0 {
			w.blank()
		} else {
			w.newline()
		}
		w.listStack = append(w.listStack, marker)
		w.counters = append(w.counters, 0)
		w.children(n)
		w.listStack = w.listStack[:len(w.listStack)-1]
		w.counters = w.counters[:len(w.counters)-1]
		if len(w.listStack) == 0 {
			w.blank()
		}
	case atom.Li:
		w.newline()
		depth := len(w.listStack) - 1
		if depth < 0 {
			depth = 0
			w.listStack = append(w.listStack, "-")
			w.counters = append(w.counters, 0)
		}
		marker := w.listStack[depth]
		if marker == "1." {
			w.counters[depth]++
			marker = itoa(w.counters[depth]) + "."
		}
		w.b.WriteString(strings.Repeat("  ", depth) + marker + " ")
		w.children(n)
		w.newline()
	case atom.A:
		href := attr(n, "href")
		label := strings.TrimSpace(text(n))
		if label == "" {
			return
		}
		if href == "" || strings.HasPrefix(href, "#") || strings.HasPrefix(href, "javascript:") {
			w.children(n)
			return
		}
		w.b.WriteString("[")
		w.children(n)
		w.b.WriteString("](" + href + ")")
	case atom.Img:
		alt := attr(n, "alt")
		src := attr(n, "src")
		if src == "" {
			return
		}
		w.b.WriteString("![" + alt + "](" + src + ")")
	case atom.Table:
		w.table(n)
	case atom.Sup:
		w.b.WriteString("^")
		w.children(n)
		w.b.WriteString("^")
	case atom.Sub:
		w.b.WriteString("~")
		w.children(n)
		w.b.WriteString("~")
	default:
		w.children(n)
	}
}

func (w *mdWriter) writeMath(t string, display bool) {
	if display {
		w.blank()
		w.b.WriteString("$$ " + t + " $$")
		w.blank()
		return
	}
	w.b.WriteString("$" + t + "$")
}

func (w *mdWriter) children(n *html.Node) {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		w.walk(c)
	}
}

// table writes a pipe table; cells lose their inner block structure.
func (w *mdWriter) table(n *html.Node) {
	var rows [][]string
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.DataAtom == atom.Tr {
			var row []string
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				if c.Type == html.ElementNode && (c.DataAtom == atom.Td || c.DataAtom == atom.Th) {
					cell := &mdWriter{}
					cell.children(c)
					row = append(row, strings.TrimSpace(strings.ReplaceAll(tidy(cell.b.String()), "\n", " ")))
				}
			}
			rows = append(rows, row)
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)
	if len(rows) == 0 {
		return
	}
	w.blank()
	for i, row := range rows {
		w.b.WriteString("| " + strings.Join(row, " | ") + " |\n")
		if i == 0 {
			seps := make([]string, len(row))
			for j := range seps {
				seps[j] = "---"
			}
			w.b.WriteString("| " + strings.Join(seps, " | ") + " |\n")
		}
	}
	w.blank()
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var d []byte
	for n > 0 {
		d = append([]byte{byte('0' + n%10)}, d...)
		n /= 10
	}
	return string(d)
}

var blankRuns = regexp.MustCompile(`\n{3,}`)

// tidy trims trailing spaces and collapses blank-line runs.
func tidy(s string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " \t")
	}
	return strings.TrimSpace(blankRuns.ReplaceAllString(strings.Join(lines, "\n"), "\n\n")) + "\n"
}
