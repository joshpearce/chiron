package sources

import (
	"context"
	"strings"
	"testing"
)

const atomFeed = `<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>Simon Willison's Weblog</title>
  <link href="https://simonwillison.net/"/>
  <entry>
    <title>Prompt injection in 2026</title>
    <link href="https://simonwillison.net/2026/Sep/20/injection/"/>
    <id>tag:simonwillison.net,2026-09-20:/injection</id>
    <updated>2026-09-20T09:15:00+00:00</updated>
    <summary>A short teaser nobody wants to read.</summary>
    <content type="html">&lt;p&gt;An agent that reads the web reads whatever an attacker wrote on it.&lt;/p&gt;
      &lt;p&gt;&lt;img src="/static/trifecta.png" alt="Where it lands"&gt;&lt;/p&gt;</content>
  </entry>
  <entry>
    <title>Quoting someone</title>
    <link href="https://simonwillison.net/2026/Sep/19/quote/"/>
    <id>tag:simonwillison.net,2026-09-19:/quote</id>
    <updated>2026-09-19T20:00:00+00:00</updated>
    <content type="html">&lt;p&gt;Two sentences, which is all a quotation is.&lt;/p&gt;</content>
  </entry>
</feed>`

const rssFeed = `<?xml version="1.0"?>
<rss version="2.0" xmlns:content="http://purl.org/rss/1.0/modules/content/">
  <channel>
    <title>The Fly Blog</title>
    <link>https://fly.io/blog/</link>
    <item>
      <title>Sizing a Firecracker</title>
      <link>https://fly.io/blog/sizing/</link>
      <guid isPermaLink="false">fly-sizing-1</guid>
      <pubDate>Fri, 19 Sep 2026 12:00:00 GMT</pubDate>
      <description>A microVM boots in under a second.</description>
      <content:encoded>&lt;p&gt;A microVM boots in under a second because there is almost nothing in it to boot.&lt;/p&gt;</content:encoded>
    </item>
  </channel>
</rss>`

// A feed is a list of pieces to read: what each one is called, where it
// is, when it was written, and its own words where it carries them.
func TestAFeedComesInAsItsEntries(t *testing.T) {
	host := fakeHost(t, map[string]string{"/atom/": atomFeed})
	f, err := client(t).Feed(context.Background(), host.URL+"/atom/")
	if err != nil {
		t.Fatal(err)
	}
	if f.Title != "Simon Willison's Weblog" {
		t.Errorf("feed title = %q", f.Title)
	}
	if len(f.Entries) != 2 {
		t.Fatalf("entries = %d", len(f.Entries))
	}
	first := f.Entries[0]
	if first.Title != "Prompt injection in 2026" ||
		first.Link != "https://simonwillison.net/2026/Sep/20/injection/" ||
		first.ID != "tag:simonwillison.net,2026-09-20:/injection" {
		t.Errorf("first entry: %+v", first)
	}
	if first.Published.Format("2006-01-02") != "2026-09-20" {
		t.Errorf("published = %v", first.Published)
	}
	// The content is the piece; the summary is the teaser beside it.
	if !strings.Contains(first.HTML, "reads whatever an attacker wrote") {
		t.Errorf("entry content = %q", first.HTML)
	}
	if strings.Contains(first.HTML, "teaser") {
		t.Errorf("the teaser was taken for the piece: %q", first.HTML)
	}
}

// RSS says the same things by other names.
func TestAnRSSFeedIsReadTheSameWay(t *testing.T) {
	host := fakeHost(t, map[string]string{"/rss/": rssFeed})
	f, err := client(t).Feed(context.Background(), host.URL+"/rss/")
	if err != nil {
		t.Fatal(err)
	}
	if f.Title != "The Fly Blog" || len(f.Entries) != 1 {
		t.Fatalf("feed: %+v", f)
	}
	e := f.Entries[0]
	if e.ID != "fly-sizing-1" || e.Link != "https://fly.io/blog/sizing/" {
		t.Errorf("entry: %+v", e)
	}
	if e.Published.Format("2006-01-02") != "2026-09-19" {
		t.Errorf("published = %v", e.Published)
	}
	if !strings.Contains(e.HTML, "almost nothing in it to boot") {
		t.Errorf("content = %q", e.HTML)
	}
}

// What is not a feed says so, rather than becoming an empty shelf card.
func TestSomethingThatIsNotAFeedIsRefused(t *testing.T) {
	host := fakeHost(t, map[string]string{"/page/": "<html><body><p>not a feed at all</p></body></html>"})
	if f, err := client(t).Feed(context.Background(), host.URL+"/page/"); err == nil {
		t.Fatalf("a page came back as a feed: %+v", f)
	}
}

// An entry that carries no id is still one entry, told apart by where it
// lives.
func TestAnEntryWithoutAnIDIsKnownByItsLink(t *testing.T) {
	const feed = `<feed xmlns="http://www.w3.org/2005/Atom"><title>A blog</title>
  <entry><title>A post</title><link href="https://example.com/a/"/>
  <content type="html">&lt;p&gt;Words enough to be a post.&lt;/p&gt;</content></entry></feed>`
	host := fakeHost(t, map[string]string{"/f/": feed})
	f, err := client(t).Feed(context.Background(), host.URL+"/f/")
	if err != nil {
		t.Fatal(err)
	}
	if f.Entries[0].ID != "https://example.com/a/" {
		t.Errorf("id = %q", f.Entries[0].ID)
	}
}
