package sources

import (
	"context"
	"strings"
	"testing"
)

// A page the reader sends to Chiron is the article on it: the writer's
// own words, their pictures and their links, without the furniture the
// site puts around them.
func TestAPageComesInAsItsArticle(t *testing.T) {
	const post = `<html><head>
<title>Prompt injection in 2026 | Simon Willison&#39;s Weblog</title>
<meta property="og:title" content="Prompt injection in 2026">
</head><body>
<nav><a href="/">Home</a><a href="/about/">About</a></nav>
<article>
  <h1>Prompt injection in 2026</h1>
  <p>An agent that reads the web reads whatever an attacker wrote on it.</p>
  <p><img src="/static/2026/trifecta.png" alt="Where the attack lands"></p>
  <h2>Why filtering does not save you</h2>
  <p>The <a href="/2025/May/3/lethal-trifecta/">lethal trifecta</a> is the shape to look for.</p>
</article>
<footer>Copyright 2026</footer>
</body></html>`
	host := fakeHost(t, map[string]string{"/2026/Sep/20/injection/": post})
	c := client(t)
	p, err := c.Page(context.Background(), host.URL+"/2026/Sep/20/injection/")
	if err != nil {
		t.Fatal(err)
	}
	if p.Title != "Prompt injection in 2026" {
		t.Errorf("title = %q", p.Title)
	}
	for _, want := range []string{
		"An agent that reads the web",
		"## Why filtering does not save you",
		"[lethal trifecta](" + host.URL + "/2025/May/3/lethal-trifecta/)",
		"![Where the attack lands](" + host.URL + "/static/2026/trifecta.png)",
	} {
		if !strings.Contains(p.Markdown, want) {
			t.Errorf("the article is missing %q:\n%s", want, p.Markdown)
		}
	}
	for _, unwanted := range []string{"About", "Copyright 2026"} {
		if strings.Contains(p.Markdown, unwanted) {
			t.Errorf("the site's furniture came with it (%q):\n%s", unwanted, p.Markdown)
		}
	}
	if len(p.Images) != 1 || p.Images[0] != host.URL+"/static/2026/trifecta.png" {
		t.Errorf("pictures = %v", p.Images)
	}
	if p.URL != host.URL+"/2026/Sep/20/injection/" {
		t.Errorf("url = %q", p.URL)
	}
}

// Not every page marks its article up. The main content is taken where
// it is found, and the title falls back from the page's own heading to
// the browser tab's, without the site's name trailing it.
func TestAPageWithoutAnArticleIsStillRead(t *testing.T) {
	const doc = `<html><head><title>Sizing a Firecracker - The Fly Blog</title></head><body>
<header><h1>The Fly Blog</h1></header>
<main>
  <h1>Sizing a Firecracker</h1>
  <p>A microVM boots in under a second because there is almost nothing in it to boot.</p>
  <p>The kernel is the guest's own, and the device model is four devices wide.</p>
</main>
</body></html>`
	host := fakeHost(t, map[string]string{"/sizing/": doc})
	c := client(t)
	p, err := c.Page(context.Background(), host.URL+"/sizing/")
	if err != nil {
		t.Fatal(err)
	}
	if p.Title != "Sizing a Firecracker" {
		t.Errorf("title = %q", p.Title)
	}
	if !strings.Contains(p.Markdown, "boots in under a second") {
		t.Errorf("the page's prose is missing:\n%s", p.Markdown)
	}
	if strings.Contains(p.Markdown, "The Fly Blog") {
		t.Errorf("the site's masthead came with it:\n%s", p.Markdown)
	}
}

// A page that is not a page is refused, rather than becoming an empty
// chapter on the shelf.
func TestAPageWithNothingOnItIsRefused(t *testing.T) {
	host := fakeHost(t, map[string]string{"/empty/": `<html><body><nav>Home</nav></body></html>`})
	c := client(t)
	if p, err := c.Page(context.Background(), host.URL+"/empty/"); err == nil {
		t.Fatalf("an empty page came back as %+v", p)
	}
}
