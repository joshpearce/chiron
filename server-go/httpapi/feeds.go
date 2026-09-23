package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/mjbraun/chiron/server/sources"
)

// A blog the reader follows. It sits on the shelf beside the books, and
// it is a reading like an imported book is: its chapters are its posts,
// newest first, read as the writer wrote them, so a highlight in one
// reaches the tutor with the whole post behind it.
//
// Nothing here runs on a timer. The server sleeps when nobody is reading
// and wakes when a device asks it something, so the check is a thing the
// app asks for when the reader opens it: /feeds/refresh, which asks each
// blog that has not been asked in the last hour.

const KindFeed = "feed"

// A blog is not asked again inside this, however often the app asks.
const feedCheckEvery = time.Hour

// A post shorter than this is a note - a link, a quotation, a sighting -
// and is read with the week's other notes rather than as a chapter of
// its own.
const shortEntryWords = 120

// What a blog just followed brings with it. Enough to read back over,
// not the whole archive.
const feedFirstEntries = 30

// Subscription is a followed blog: where its feed is, when it was last
// asked, and what of it is already here.
type Subscription struct {
	ID          string     `json:"id"`
	URL         string     `json:"url"`
	Title       string     `json:"title"`
	LastChecked string     `json:"last_checked"`
	Next        int        `json:"next"`
	Items       []FeedItem `json:"items"`
}

// FeedItem is one chapter of a followed blog: a post, or the week that
// gathers the notes.
type FeedItem struct {
	Unit      string `json:"unit"`
	Slug      string `json:"slug"`
	Title     string `json:"title"`
	Published string `json:"published"`
	// Updated is when this chapter last gained anything, which is what
	// tells a chapter the reader has seen from one that has changed
	// since: a week of notes gains notes.
	Updated string `json:"updated"`
	Minutes int    `json:"minutes"`
	// Entries are the feed's own ids for what this chapter holds, so a
	// post already here is never taken twice.
	Entries []string `json:"entries"`
	// Week is set on the chapter that gathers a week's notes.
	Week string `json:"week,omitempty"`
}

func feedMeta(root, id string) string { return filepath.Join(root, id, "feed.json") }

// handleFeedFollow takes a feed's address and puts the blog on the shelf.
func (s *Server) handleFeedFollow(w http.ResponseWriter, r *http.Request) {
	var body struct {
		URL   string `json:"url"`
		Title string `json:"title"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "read the request: %v", err)
		return
	}
	raw := strings.TrimSpace(body.URL)
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		writeError(w, http.StatusBadRequest, "a feed is an http or https address; %q is not", raw)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), pageFetchTimeout)
	defer cancel()
	client := s.newPublicClient("")
	feed, err := client.Feed(ctx, raw)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "this is not a feed to follow: %v", err)
		return
	}
	title := strings.TrimSpace(body.Title)
	if title == "" {
		title = feed.Title
	}
	if title == "" {
		title = u.Host
	}

	var id [4]byte
	rand.Read(id[:])
	readingID := "read-" + hex.EncodeToString(id[:])
	root := s.readingsRoot()
	if err := os.MkdirAll(readingDir(root, readingID), 0o755); err != nil {
		writeError(w, http.StatusInternalServerError, "keep the blog: %v", err)
		return
	}
	sub := &Subscription{ID: readingID, URL: raw, Title: title, Next: 1}
	added := s.takeEntries(ctx, client, root, sub, feed.Entries)
	sub.LastChecked = time.Now().UTC().Format(time.RFC3339)
	if added == 0 {
		os.RemoveAll(readingDir(root, readingID))
		writeError(w, http.StatusUnprocessableEntity, "this feed has nothing in it to read")
		return
	}
	meta, err := s.keepFeed(root, sub)
	if err != nil {
		os.RemoveAll(readingDir(root, readingID))
		writeError(w, http.StatusInternalServerError, "keep the blog: %v", err)
		return
	}
	writeJSON(w, http.StatusOK, meta)
}

// handleFeedRefresh asks the blogs what is new. The app asks for this
// when the reader opens it, which is also what woke the server.
func (s *Server) handleFeedRefresh(w http.ResponseWriter, r *http.Request) {
	force := r.URL.Query().Get("force") != ""
	ctx, cancel := context.WithTimeout(r.Context(), pageFetchTimeout)
	defer cancel()
	s.feedMu.Lock()
	defer s.feedMu.Unlock()
	root := s.readingsRoot()
	client := s.newPublicClient("")
	checked, added := 0, 0
	blogs := []map[string]any{}
	for _, sub := range s.subscriptions() {
		if !force && !dueForCheck(sub.LastChecked) {
			continue
		}
		checked++
		feed, err := client.Feed(ctx, sub.URL)
		if err != nil {
			log.Printf("feed %s (%s): %v", sub.Title, sub.URL, err)
			blogs = append(blogs, map[string]any{"id": sub.ID, "title": sub.Title, "error": err.Error()})
			continue
		}
		n := s.takeEntries(ctx, client, root, sub, feed.Entries)
		sub.LastChecked = time.Now().UTC().Format(time.RFC3339)
		if _, err := s.keepFeed(root, sub); err != nil {
			log.Printf("feed %s: %v", sub.Title, err)
			continue
		}
		added += n
		blogs = append(blogs, map[string]any{"id": sub.ID, "title": sub.Title, "added": n})
	}
	writeJSON(w, http.StatusOK, map[string]any{"checked": checked, "added": added, "feeds": blogs})
}

func (s *Server) handleFeedList(w http.ResponseWriter, r *http.Request) {
	feeds := []map[string]any{}
	for _, sub := range s.subscriptions() {
		feeds = append(feeds, map[string]any{
			"id": sub.ID, "title": sub.Title, "url": sub.URL,
			"last_checked": sub.LastChecked, "chapters": len(sub.Items),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"feeds": feeds})
}

func dueForCheck(last string) bool {
	if last == "" {
		return true
	}
	t, err := time.Parse(time.RFC3339, last)
	if err != nil {
		return true
	}
	return time.Since(t) >= feedCheckEvery
}

// subscriptions reads the followed blogs off the shelf.
func (s *Server) subscriptions() []*Subscription {
	root := s.readingsRoot()
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	var out []*Subscription
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		data, err := os.ReadFile(feedMeta(root, e.Name()))
		if err != nil {
			continue
		}
		var sub Subscription
		if json.Unmarshal(data, &sub) != nil || sub.ID == "" {
			continue
		}
		out = append(out, &sub)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Title < out[j].Title })
	return out
}

// takeEntries brings in what the blog has published since the last time,
// oldest first, so a week of notes reads in the order it was written.
// It returns how many posts came in.
func (s *Server) takeEntries(ctx context.Context, c *sources.Client, root string, sub *Subscription, entries []sources.FeedEntry) int {
	fresh := make([]sources.FeedEntry, 0, len(entries))
	for _, e := range entries {
		if !sub.holds(e.ID) {
			fresh = append(fresh, e)
		}
	}
	if len(fresh) > feedFirstEntries {
		fresh = fresh[:feedFirstEntries]
	}
	sort.SliceStable(fresh, func(i, j int) bool { return fresh[i].Published.Before(fresh[j].Published) })
	added := 0
	for _, e := range fresh {
		if err := s.takeEntry(ctx, c, root, sub, e); err != nil {
			log.Printf("feed %s: %s: %v", sub.Title, e.Title, err)
			continue
		}
		added++
	}
	return added
}

func (sub *Subscription) holds(entryID string) bool {
	for _, item := range sub.Items {
		for _, id := range item.Entries {
			if id == entryID {
				return true
			}
		}
	}
	return false
}

func (sub *Subscription) item(unit string) *FeedItem {
	for i := range sub.Items {
		if sub.Items[i].Unit == unit {
			return &sub.Items[i]
		}
	}
	return nil
}

func (sub *Subscription) weekItem(week string) *FeedItem {
	for i := range sub.Items {
		if sub.Items[i].Week == week {
			return &sub.Items[i]
		}
	}
	return nil
}

// takeEntry writes one post into the blog's corpus: its own chapter
// where it is a piece, the week's chapter where it is a note.
func (s *Server) takeEntry(ctx context.Context, c *sources.Client, root string, sub *Subscription, e sources.FeedEntry) error {
	md, pictures := sources.EntryContent(e.Link, e.HTML)
	// A feed that carries nothing, or a sentence and "read the rest on
	// the site": the piece is on the site, so the post is fetched like
	// any other page. A feed whose short entries are genuinely short -
	// a link, a quotation - gets its own words back and keeps them.
	if words(md) < shortEntryWords && e.Link != "" {
		if page, err := c.Page(ctx, e.Link); err == nil && words(page.Markdown) > words(md) {
			md, pictures = page.Markdown, page.Images
		} else if strings.TrimSpace(md) == "" {
			if err != nil {
				return err
			}
			return fmt.Errorf("nothing to read and nothing at %s", e.Link)
		}
	}
	if strings.TrimSpace(md) == "" {
		return fmt.Errorf("nothing to read and nowhere to read it")
	}
	kept := storeWebAssets(ctx, c, readingAssets(root, sub.ID), sub.ID, pictures)
	for ref, name := range kept {
		md = strings.ReplaceAll(md, "]("+ref+")", "](assets/"+name+")")
	}
	when := e.Published
	if when.IsZero() {
		when = time.Now().UTC()
	}
	title := e.Title
	if title == "" {
		title = "A post"
	}
	if words(md) >= shortEntryWords {
		item := FeedItem{
			Unit: fmt.Sprintf("u%d", sub.Next), Slug: slugOf(title, fmt.Sprintf("u%d", sub.Next)),
			Title: title, Published: when.UTC().Format(time.RFC3339),
			Updated: time.Now().UTC().Format(time.RFC3339Nano), Entries: []string{e.ID},
		}
		sub.Next++
		body := md + "\n\n" + entryFooter(e.Link, when)
		if err := writeFeedChapter(root, sub.ID, item, body); err != nil {
			return err
		}
		item.Minutes = readingMinutes(body)
		sub.Items = append(sub.Items, item)
		return nil
	}

	week := weekOf(when)
	item := sub.weekItem(week)
	fresh := item == nil
	if fresh {
		unit := fmt.Sprintf("u%d", sub.Next)
		sub.Next++
		sub.Items = append(sub.Items, FeedItem{
			Unit: unit, Slug: slugOf(weekTitle(when), unit), Title: weekTitle(when),
			Published: when.UTC().Format(time.RFC3339), Week: week,
		})
		item = &sub.Items[len(sub.Items)-1]
	}
	note := "### " + title + "\n\n" + md + "\n\n" + entryFooter(e.Link, when)
	body, err := appendFeedChapter(root, sub.ID, *item, note, fresh)
	if err != nil {
		return err
	}
	item.Entries = append(item.Entries, e.ID)
	item.Minutes = readingMinutes(body)
	item.Updated = time.Now().UTC().Format(time.RFC3339Nano)
	// A week's chapter is as new as the last note in it.
	item.Published = when.UTC().Format(time.RFC3339)
	return nil
}

func entryFooter(link string, when time.Time) string {
	day := when.UTC().Format("2 January 2006")
	if link == "" {
		return "*" + day + "*"
	}
	return "[" + link + "](" + link + ") · *" + day + "*"
}

// weekOf is the week a note belongs to, by the Monday it falls after.
func weekOf(t time.Time) string {
	year, week := t.UTC().ISOWeek()
	return fmt.Sprintf("%d-W%02d", year, week)
}

// weekTitle names the week a note fell in. A week of another year says
// which: a feed brings its archive with it, and a contents with three
// "week of 17 August" in it says nothing.
func weekTitle(t time.Time) string {
	monday := t.UTC()
	for monday.Weekday() != time.Monday {
		monday = monday.AddDate(0, 0, -1)
	}
	if monday.Year() != time.Now().UTC().Year() {
		return "Notes, week of " + monday.Format("2 January 2006")
	}
	return "Notes, week of " + monday.Format("2 January")
}

func feedUnitDir(root, id string, item FeedItem) string {
	return filepath.Join(readingCorpus(root, id), "units", item.Unit+"-"+item.Slug)
}

func writeFeedChapter(root, id string, item FeedItem, body string) error {
	dir := feedUnitDir(root, id, item)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	text := "## " + item.Title + "\n\n" + body + "\n"
	return os.WriteFile(filepath.Join(dir, "canon.md"), []byte(text), 0o644)
}

// appendFeedChapter adds a note to the week that holds it, and answers
// with the week as it now reads.
func appendFeedChapter(root, id string, item FeedItem, note string, fresh bool) (string, error) {
	dir := feedUnitDir(root, id, item)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	file := filepath.Join(dir, "canon.md")
	text := ""
	if !fresh {
		if data, err := os.ReadFile(file); err == nil {
			text = strings.TrimRight(string(data), "\n") + "\n\n"
		}
	}
	if text == "" {
		text = "## " + item.Title + "\n\n"
	}
	text += note + "\n"
	return text, os.WriteFile(file, []byte(text), 0o644)
}

// keepFeed writes the blog's contents, its record and its shelf card,
// and puts it back on the shelf as it now reads.
func (s *Server) keepFeed(root string, sub *Subscription) (*Reading, error) {
	if err := writeFeedSyllabus(readingCorpus(root, sub.ID), sub); err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(sub, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(feedMeta(root, sub.ID), data, 0o644); err != nil {
		return nil, err
	}
	meta := &Reading{ID: sub.ID, Title: sub.Title, Chapters: len(sub.Items),
		ImportedAt: time.Now().UTC().Format(time.RFC3339), URL: sub.URL, Feed: true}
	if err := writeReadingMeta(root, meta); err != nil {
		return nil, err
	}
	if err := s.registerReading(meta); err != nil {
		return nil, err
	}
	return meta, nil
}

// writeFeedSyllabus lays the blog out newest first. Nothing gates a
// chapter: every post can be opened, as the blog's own archive opens.
func writeFeedSyllabus(dir string, sub *Subscription) error {
	items := make([]FeedItem, len(sub.Items))
	copy(items, sub.Items)
	sort.SliceStable(items, func(i, j int) bool { return items[i].Published > items[j].Published })
	var syllabus strings.Builder
	fmt.Fprintf(&syllabus, "title: %s\nunits:\n", yamlQuote(sub.Title))
	for _, item := range items {
		minutes := item.Minutes
		if minutes < 1 {
			minutes = 1
		}
		fmt.Fprintf(&syllabus, "  - id: %s\n    slug: %s\n    title: %s\n    minutes: %d\n",
			item.Unit, item.Slug, yamlQuote(item.Title), minutes)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "syllabus.yaml"), []byte(syllabus.String()), 0o644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, "misconception-bank.yaml"), []byte("misconceptions: []\n"), 0o644)
}

// What the reader has already seen. A chapter is read at a moment; a
// week of notes that gains another note is published later than that
// moment, and so is unread again.

func readMarks(root, id string) map[string]string {
	marks := map[string]string{}
	data, err := os.ReadFile(filepath.Join(readingDir(root, id), "read.json"))
	if err != nil {
		return marks
	}
	json.Unmarshal(data, &marks)
	return marks
}

func writeReadMarks(root, id string, marks map[string]string) error {
	data, err := json.MarshalIndent(marks, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(readingDir(root, id), "read.json"), data, 0o644)
}

// unreadIn counts the chapters the reader has not seen as they now are.
func unreadIn(sub *Subscription, marks map[string]string) int {
	n := 0
	for _, item := range sub.Items {
		if at, ok := marks[item.Unit]; !ok || before(at, item.Updated) {
			n++
		}
	}
	return n
}

// before is whether the first moment is earlier than the second; a
// moment that cannot be read is treated as long ago.
func before(a, b string) bool {
	at, err := time.Parse(time.RFC3339Nano, a)
	if err != nil {
		return true
	}
	bt, err := time.Parse(time.RFC3339Nano, b)
	if err != nil {
		return false
	}
	return at.Before(bt)
}

// handleReadingRead marks a chapter seen, for every device.
func (s *Server) handleReadingRead(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body struct {
		Unit string `json:"unit"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "read the request: %v", err)
		return
	}
	root := s.readingsRoot()
	if _, err := os.Stat(readingMeta(root, id)); err != nil {
		writeError(w, http.StatusNotFound, "nothing read here is called %q", id)
		return
	}
	s.feedMu.Lock()
	defer s.feedMu.Unlock()
	marks := readMarks(root, id)
	marks[body.Unit] = time.Now().UTC().Format(time.RFC3339Nano)
	if err := writeReadMarks(root, id, marks); err != nil {
		writeError(w, http.StatusInternalServerError, "keep what was read: %v", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"unread": s.unreadFor(root, id)})
}

// unreadFor is what a followed blog's card shows; anything else that is
// read has nothing to count.
func (s *Server) unreadFor(root, id string) int {
	data, err := os.ReadFile(feedMeta(root, id))
	if err != nil {
		return 0
	}
	var sub Subscription
	if json.Unmarshal(data, &sub) != nil {
		return 0
	}
	return unreadIn(&sub, readMarks(root, id))
}
