package sources

import (
	"strings"
	"testing"
)

// The converter turns the HTML a textbook page serves into the Markdown
// the author reads: headings, paragraphs, lists, code, emphasis, links,
// tables, and math left as it came. Navigation, scripts and styles go.
func TestHTMLBecomesReadableMarkdown(t *testing.T) {
	in := `<html><head><title>x</title><style>p{}</style><script>var a;</script></head><body>
<nav><a href="/">home</a></nav>
<h1>Bayes' rule</h1>
<p>Given a <em>prior</em> and a <strong>likelihood</strong>, the posterior is
proportional to their product: $P(H|D) \propto P(D|H)P(H)$.</p>
<h2>An example</h2>
<ul><li>one</li><li>two <a href="https://x.org/y">with a link</a></li></ul>
<ol><li>first</li><li>second</li></ol>
<pre><code class="language-python">x = 1
y = x + 1
</code></pre>
<p>Inline <code>code</code> stays.</p>
<blockquote><p>A quoted line.</p></blockquote>
<table><tr><th>a</th><th>b</th></tr><tr><td>1</td><td>2</td></tr></table>
<img src="fig.png" alt="A figure">
<footer>This page titled ... is shared under</footer>
</body></html>`
	got := HTMLToMarkdown(in)
	for _, want := range []string{
		"# Bayes' rule\n",
		"Given a *prior* and a **likelihood**, the posterior is proportional to their product: $P(H|D) \\propto P(D|H)P(H)$.",
		"## An example\n",
		"- one\n- two [with a link](https://x.org/y)\n",
		"1. first\n2. second\n",
		"```python\nx = 1\ny = x + 1\n```",
		"Inline `code` stays.",
		"> A quoted line.",
		"| a | b |\n| --- | --- |\n| 1 | 2 |",
		"![A figure](fig.png)",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	for _, gone := range []string{"var a;", "p{}", "home", "shared under"} {
		if strings.Contains(got, gone) {
			t.Errorf("should have dropped %q:\n%s", gone, got)
		}
	}
	if strings.Contains(got, "\n\n\n") {
		t.Errorf("runs of blank lines:\n%s", got)
	}
}

func TestMathMLAndKaTeXAnnotationsKeepTheTeX(t *testing.T) {
	// MathJax/KaTeX pages carry the TeX in an annotation or a script tag;
	// that is what should survive, not the rendered spans.
	in := `<p>So <span class="math inline"><span class="katex"><span class="katex-mathml"><math><semantics><mrow><mi>x</mi></mrow><annotation encoding="application/x-tex">x^2</annotation></semantics></math></span><span class="katex-html">x2</span></span></span> and
<script type="math/tex">\int f</script> and <span class="math display">\[ a+b \]</span>.</p>`
	got := HTMLToMarkdown(in)
	for _, want := range []string{"$x^2$", "$\\int f$", "$$ a+b $$"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "x2") {
		t.Errorf("rendered math leaked:\n%s", got)
	}
}
