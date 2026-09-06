package sources

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Chunk is one section of one source as the author receives it: Markdown,
// and the record of where it came from and what may be done with it.
type Chunk struct {
	Source     string     `json:"source" yaml:"source"`
	Locator    string     `json:"locator" yaml:"locator"`
	Title      string     `json:"title" yaml:"title"`
	Markdown   string     `json:"-" yaml:"-"`
	Provenance Provenance `json:"provenance" yaml:"provenance"`
}

// Provenance is rule 1 of OPEN-SOURCES.md section 4: enough to attribute,
// to audit, and to know whether the text may be adapted.
type Provenance struct {
	Source     string  `json:"source" yaml:"source"`
	Title      string  `json:"title" yaml:"title"`
	Authors    string  `json:"authors" yaml:"authors"`
	URL        string  `json:"url" yaml:"url"`
	Licence    string  `json:"licence" yaml:"licence"`
	LicenceURL string  `json:"licence_url" yaml:"licence_url"`
	Verdict    Verdict `json:"verdict" yaml:"verdict"`
	Fetched    string  `json:"fetched" yaml:"fetched"`
	// Note records anything the fetcher saw on the page that bears on the
	// verdict, such as a licence tag that differs from the index.
	Note string `json:"note,omitempty" yaml:"note,omitempty"`
}

// Attribution is the line a chapter shows for this chunk.
func (p Provenance) Attribution() string {
	who := p.Title
	if p.Authors != "" {
		who += " by " + p.Authors
	}
	return fmt.Sprintf("Adapted from %s (%s)", who, p.Licence)
}

// A fetcher knows one kind of source.
type fetcher interface {
	contents(ctx context.Context, c *Client, s *Source) ([]Section, error)
	// fetch returns the section's title, Markdown, the URL it came from,
	// and any note about the licence seen on the page.
	fetch(ctx context.Context, c *Client, s *Source, locator string) (fetched, error)
}

type fetched struct {
	title, markdown, url, note string
	// verdict, when set, overrides the index for this chunk (rule 5).
	verdict Verdict
}

var fetchers = map[string]fetcher{
	"github":     githubFetcher{},
	"openstax":   openstaxFetcher{},
	"libretexts": libretextsFetcher{},
	"mediawiki":  mediawikiFetcher{},
	"ocw":        ocwFetcher{},
	"gutenberg":  gutenbergFetcher{},
	"pressbooks": pressbooksFetcher{},
}

// Client fetches, one request at a time per host with a pause between,
// caching every response under Cache so a rerun costs nothing and the
// hosts are not asked twice.
type Client struct {
	HTTP      *http.Client
	Cache     string
	UserAgent string
	// Pause between requests to the same host.
	Pause time.Duration
	// Now is the clock for provenance; tests pin it.
	Now func() time.Time
	// Catalogues are the keyless search endpoints the finder asks.
	Catalogues Catalogues

	mu    sync.Mutex
	hosts map[string]*hostState
}

type hostState struct {
	mu   sync.Mutex
	last time.Time
}

func NewClient(cache string) *Client {
	return &Client{
		HTTP:  &http.Client{Timeout: 60 * time.Second},
		Cache: cache,
		// A plain browser string: some hosts (Pressbooks behind CloudFront)
		// answer 403 to anything that names a tool.
		UserAgent:  "Mozilla/5.0 (Macintosh; Intel Mac OS X 14_0) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Safari/605.1.15",
		Pause:      700 * time.Millisecond,
		Now:        time.Now,
		Catalogues: DefaultCatalogues,
	}
}

// ErrRestricted is returned for a source the rules say not to fetch.
var ErrRestricted = errors.New("source is restricted: cite it, do not fetch it")

// Contents lists a source's sections: the index's list when it has one,
// else what the fetcher finds.
func (c *Client) Contents(ctx context.Context, s *Source) ([]Section, error) {
	if s.Verdict == Restricted {
		return nil, ErrRestricted
	}
	if len(s.Contents) > 0 {
		return s.Contents, nil
	}
	f, ok := fetchers[s.Fetch.Kind]
	if !ok {
		return nil, fmt.Errorf("%s: no fetcher for kind %q", s.ID, s.Fetch.Kind)
	}
	return f.contents(ctx, c, s)
}

// Fetch brings one section of a source in as a chunk.
func (c *Client) Fetch(ctx context.Context, s *Source, locator string) (*Chunk, error) {
	if s.Verdict == Restricted {
		return nil, ErrRestricted
	}
	f, ok := fetchers[s.Fetch.Kind]
	if !ok {
		return nil, fmt.Errorf("%s: no fetcher for kind %q", s.ID, s.Fetch.Kind)
	}
	got, err := f.fetch(ctx, c, s, locator)
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w", s.ID, locator, err)
	}
	verdict := s.Verdict
	if got.verdict != "" {
		verdict = got.verdict
	}
	now := c.Now
	if now == nil {
		now = time.Now
	}
	return &Chunk{
		Source:   s.ID,
		Locator:  locator,
		Title:    got.title,
		Markdown: got.markdown,
		Provenance: Provenance{
			Source: s.ID, Title: s.Title, Authors: strings.Join(s.Authors, ", "),
			URL: got.url, Licence: s.Licence.Name, LicenceURL: s.Licence.URL,
			Verdict: verdict, Fetched: now().UTC().Format("2006-01-02"), Note: got.note,
		},
	}, nil
}

// get fetches a URL through the cache.
func (c *Client) get(ctx context.Context, rawURL string) ([]byte, error) {
	if c.Cache != "" {
		if data, err := os.ReadFile(c.cachePath(rawURL)); err == nil {
			return data, nil
		}
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	c.wait(u.Host)
	var data []byte
	for attempt := 0; attempt < 2; attempt++ {
		req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
		if err != nil {
			return nil, err
		}
		if c.UserAgent != "" {
			req.Header.Set("User-Agent", c.UserAgent)
		}
		req.Header.Set("Accept", "text/html,application/json,text/plain,*/*")
		client := c.HTTP
		if client == nil {
			client = http.DefaultClient
		}
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		data, err = io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}
		if resp.StatusCode >= 500 && attempt == 0 {
			time.Sleep(2 * time.Second)
			continue
		}
		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("GET %s: %d", rawURL, resp.StatusCode)
		}
		break
	}
	if c.Cache != "" {
		p := c.cachePath(rawURL)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err == nil {
			os.WriteFile(p, data, 0o644)
		}
	}
	return data, nil
}

func (c *Client) cachePath(rawURL string) string {
	sum := sha256.Sum256([]byte(rawURL))
	host := "unknown"
	if u, err := url.Parse(rawURL); err == nil && u.Host != "" {
		host = u.Host
	}
	return filepath.Join(c.Cache, host, hex.EncodeToString(sum[:8]))
}

// wait serialises requests to a host and keeps the pause between them.
func (c *Client) wait(host string) {
	c.mu.Lock()
	if c.hosts == nil {
		c.hosts = map[string]*hostState{}
	}
	h := c.hosts[host]
	if h == nil {
		h = &hostState{}
		c.hosts[host] = h
	}
	c.mu.Unlock()
	h.mu.Lock()
	defer h.mu.Unlock()
	if c.Pause > 0 {
		if since := time.Since(h.last); since < c.Pause {
			time.Sleep(c.Pause - since)
		}
	}
	h.last = time.Now()
}

var tagStrip = regexp.MustCompile(`<[^>]*>`)

// plainTitle strips markup from a title a catalogue serves with spans in it.
func plainTitle(s string) string {
	return strings.TrimSpace(tagStrip.ReplaceAllString(s, ""))
}

// firstHeading is the title Markdown gives itself, if any.
func firstHeading(md string) string {
	for _, line := range strings.Split(md, "\n") {
		if strings.HasPrefix(line, "#") {
			return strings.TrimSpace(strings.TrimLeft(line, "# "))
		}
	}
	return ""
}

func base(r Recipe, def string) string {
	if r.Base != "" {
		return strings.TrimRight(r.Base, "/")
	}
	return def
}
