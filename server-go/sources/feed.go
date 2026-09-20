package sources

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"
)

// A feed a reader follows: the blog's own list of what it has published,
// in Atom or RSS. Both say the same things by different names, so both
// come in as the same handful of facts about each entry. What is not
// said here - the piece's own words, where the feed carries only a
// teaser - is fetched from the entry's page instead.

// Feed is a blog as its feed describes it.
type Feed struct {
	URL     string
	Title   string
	Link    string
	Entries []FeedEntry
}

// FeedEntry is one piece in a feed.
type FeedEntry struct {
	// ID is what the feed calls this entry, and what tells a piece
	// already read from one just published.
	ID        string
	Title     string
	Link      string
	Published time.Time
	// HTML is the entry's own content where the feed carries it.
	HTML string
}

// Feed fetches a feed and reads its entries, newest as the feed orders
// them.
func (c *Client) Feed(ctx context.Context, rawURL string) (*Feed, error) {
	data, err := c.Get(ctx, rawURL)
	if err != nil {
		return nil, err
	}
	return ParseFeed(rawURL, data)
}

// The XML both kinds of feed are written in, read as one shape: the
// fields that differ sit side by side and whichever is there is used.
type xmlFeed struct {
	Title   string      `xml:"title"`
	Links   []xmlLink   `xml:"link"`
	Entries []xmlEntry  `xml:"entry"`
	Channel *xmlChannel `xml:"channel"`
}

type xmlChannel struct {
	Title string     `xml:"title"`
	Link  string     `xml:"link"`
	Items []xmlEntry `xml:"item"`
}

type xmlLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
	Text string `xml:",chardata"`
}

type xmlEntry struct {
	Title   string    `xml:"title"`
	Links   []xmlLink `xml:"link"`
	ID      string    `xml:"id"`
	GUID    string    `xml:"guid"`
	Updated string    `xml:"updated"`
	PubDate string    `xml:"pubDate"`
	Date    string    `xml:"date"`
	Content string    `xml:"content"`
	Encoded string    `xml:"encoded"`
	Summary string    `xml:"summary"`
	Desc    string    `xml:"description"`
}

// ParseFeed reads a feed already fetched from rawURL.
func ParseFeed(rawURL string, data []byte) (*Feed, error) {
	var doc xmlFeed
	dec := xml.NewDecoder(strings.NewReader(string(data)))
	// A feed in the wild is not always the encoding it claims, and a
	// stray entity is not a reason to lose the blog.
	dec.Strict = false
	dec.CharsetReader = func(_ string, r io.Reader) (io.Reader, error) { return r, nil }
	if err := dec.Decode(&doc); err != nil {
		return nil, fmt.Errorf("this is not a feed: %w", err)
	}
	f := &Feed{URL: rawURL, Title: strings.TrimSpace(doc.Title), Link: linkOf(doc.Links)}
	entries := doc.Entries
	if doc.Channel != nil {
		if f.Title == "" {
			f.Title = strings.TrimSpace(doc.Channel.Title)
		}
		if f.Link == "" {
			f.Link = strings.TrimSpace(doc.Channel.Link)
		}
		entries = append(entries, doc.Channel.Items...)
	}
	base, _ := url.Parse(rawURL)
	for _, e := range entries {
		entry := FeedEntry{
			Title:     strings.TrimSpace(e.Title),
			Link:      absolute(base, firstOf(linkOf(e.Links), strings.TrimSpace(e.GUID))),
			Published: feedTime(e.Updated, e.PubDate, e.Date),
			HTML:      firstOf(e.Content, e.Encoded, e.Summary, e.Desc),
		}
		entry.ID = firstOf(strings.TrimSpace(e.ID), strings.TrimSpace(e.GUID), entry.Link, entry.Title)
		if entry.ID == "" {
			continue
		}
		f.Entries = append(f.Entries, entry)
	}
	if len(f.Entries) == 0 {
		return nil, fmt.Errorf("this is not a feed with anything in it")
	}
	return f, nil
}

func linkOf(links []xmlLink) string {
	for _, l := range links {
		if l.Rel != "" && l.Rel != "alternate" {
			continue
		}
		if href := strings.TrimSpace(firstOf(l.Href, l.Text)); href != "" {
			return href
		}
	}
	return ""
}

func absolute(base *url.URL, link string) string {
	if base == nil || link == "" {
		return link
	}
	if u, err := base.Parse(link); err == nil {
		return u.String()
	}
	return link
}

func firstOf(vs ...string) string {
	for _, v := range vs {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}

// The times feeds are written in: Atom's RFC 3339, RSS's RFC 1123, and
// the near misses both of them are published with.
var feedTimes = []string{
	time.RFC3339, "2006-01-02T15:04:05Z0700", time.RFC1123Z, time.RFC1123,
	"Mon, 2 Jan 2006 15:04:05 -0700", "Mon, 2 Jan 2006 15:04:05 MST",
	"2006-01-02 15:04:05", "2006-01-02",
}

func feedTime(candidates ...string) time.Time {
	for _, c := range candidates {
		c = strings.TrimSpace(c)
		if c == "" {
			continue
		}
		for _, layout := range feedTimes {
			if t, err := time.Parse(layout, c); err == nil {
				return t.UTC()
			}
		}
	}
	return time.Time{}
}
