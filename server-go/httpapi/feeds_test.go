package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// blog is a feed that gains entries between one check and the next.
type blog struct {
	mu      sync.Mutex
	entries []string
}

func (b *blog) add(entry string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.entries = append([]string{entry}, b.entries...)
}

func (b *blog) atom() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return `<?xml version="1.0" encoding="utf-8"?>
<feed xmlns="http://www.w3.org/2005/Atom"><title>A Weblog</title>
<link href="https://ablog.example/"/>` + strings.Join(b.entries, "\n") + `</feed>`
}

func longEntry(id, title, date string) string {
	return fmt.Sprintf(`<entry><title>%s</title><link href="https://ablog.example/%s/"/>
<id>%s</id><updated>%sT10:00:00Z</updated>
<content type="html">&lt;p&gt;%s&lt;/p&gt;</content></entry>`,
		title, id, id, date, strings.Repeat("This is a paragraph of a piece worth reading on its own. ", 12))
}

func shortEntry(id, title, date string) string {
	return fmt.Sprintf(`<entry><title>%s</title><link href="https://ablog.example/%s/"/>
<id>%s</id><updated>%sT09:00:00Z</updated>
<content type="html">&lt;p&gt;A line about %s, which is all this one is.&lt;/p&gt;</content></entry>`,
		title, id, id, date, title)
}

func feedSite(t *testing.T, b *blog) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/atom/" {
			w.Header().Set("Content-Type", "application/atom+xml")
			fmt.Fprint(w, b.atom())
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func subjectRow(t *testing.T, s *Server, id string) (kind, title string, ok bool) {
	t.Helper()
	w := do(t, s, "GET", "/subjects", "", "")
	var shelf struct {
		Subjects []struct{ ID, Kind, Title string } `json:"subjects"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &shelf); err != nil {
		t.Fatal(err)
	}
	for _, row := range shelf.Subjects {
		if row.ID == id {
			return row.Kind, row.Title, true
		}
	}
	return "", "", false
}

func spineOf(t *testing.T, s *Server, id string) []struct {
	Unit  string `json:"unit"`
	Title string `json:"title"`
} {
	t.Helper()
	w := do(t, s, "POST", "/exchange", `{"subject":"`+id+`","phase":"start"}`, "")
	var opened struct {
		State struct {
			Spine []struct {
				Unit  string `json:"unit"`
				Title string `json:"title"`
			} `json:"spine"`
		} `json:"state"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &opened); err != nil {
		t.Fatalf("open %s: %v %s", id, err, w.Body)
	}
	return opened.State.Spine
}

// A blog the reader follows is a card on the shelf whose chapters are its
// posts, newest first. The pieces worth reading stand on their own; the
// one-line notes are gathered into the week they were written in, so the
// contents reads as a blog and not as a stream.
func TestABlogIsFollowedAsAShelfCard(t *testing.T) {
	s := newServer(t, "")
	s.cfg.ReadingsDir = t.TempDir()
	b := &blog{}
	b.add(longEntry("injection", "Prompt injection in 2026", "2026-09-14"))
	b.add(shortEntry("quote-1", "Quoting someone", "2026-09-15"))
	b.add(shortEntry("quote-2", "Quoting someone else", "2026-09-16"))
	b.add(longEntry("routes", "Generating running routes", "2026-09-17"))
	site := feedSite(t, b)

	w := do(t, s, "POST", "/feeds", `{"url":"`+site.URL+`/atom/"}`, "")
	if w.Code != http.StatusOK {
		t.Fatalf("follow: %d %s", w.Code, w.Body)
	}
	var made struct {
		ID       string `json:"id"`
		Title    string `json:"title"`
		Chapters int    `json:"chapters"`
		URL      string `json:"url"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &made); err != nil {
		t.Fatal(err)
	}
	if made.Title != "A Weblog" || made.URL != site.URL+"/atom/" {
		t.Fatalf("followed: %+v", made)
	}
	if made.Chapters != 3 {
		t.Fatalf("two pieces and a week of notes make three chapters, not %d", made.Chapters)
	}
	if kind, title, ok := subjectRow(t, s, made.ID); !ok || kind != KindFeed || title != "A Weblog" {
		t.Fatalf("shelf row: %q %q %v", kind, title, ok)
	}

	spine := spineOf(t, s, made.ID)
	if len(spine) != 3 {
		t.Fatalf("spine: %+v", spine)
	}
	// Newest first, and a week of notes is as new as the last note in it.
	if spine[0].Title != "Generating running routes" || spine[2].Title != "Prompt injection in 2026" {
		t.Errorf("the pieces are not newest first: %+v", spine)
	}
	if !strings.HasPrefix(spine[1].Title, "Notes, week of") {
		t.Errorf("the notes are not gathered: %+v", spine)
	}

	// The week of notes holds both of them, each with where it came from.
	w = do(t, s, "POST", "/exchange", `{"subject":"`+made.ID+`","phase":"start","choice":"`+spine[1].Unit+`"}`, "")
	var notes struct {
		Chapter *struct {
			HTML string `json:"html"`
		} `json:"chapter"`
	}
	json.Unmarshal(w.Body.Bytes(), &notes)
	if notes.Chapter == nil {
		t.Fatalf("the notes did not open: %s", w.Body)
	}
	for _, want := range []string{"Quoting someone", "Quoting someone else", "https://ablog.example/quote-1/"} {
		if !strings.Contains(notes.Chapter.HTML, want) {
			t.Errorf("the week of notes is missing %q:\n%s", want, notes.Chapter.HTML)
		}
	}
}

// A check asks the blog what is new. What was already read keeps the
// chapter it had; what is new goes on top; and a check with nothing new
// changes nothing.
func TestAFeedCheckBringsInWhatIsNew(t *testing.T) {
	s := newServer(t, "")
	s.cfg.ReadingsDir = t.TempDir()
	b := &blog{}
	b.add(longEntry("injection", "Prompt injection in 2026", "2026-09-14"))
	site := feedSite(t, b)
	w := do(t, s, "POST", "/feeds", `{"url":"`+site.URL+`/atom/"}`, "")
	var made struct {
		ID string `json:"id"`
	}
	json.Unmarshal(w.Body.Bytes(), &made)
	first := spineOf(t, s, made.ID)
	if len(first) != 1 {
		t.Fatalf("spine: %+v", first)
	}

	b.add(longEntry("agents", "Agents that read the web", "2026-09-20"))
	w = do(t, s, "POST", "/feeds/refresh?force=1", "", "")
	if w.Code != http.StatusOK {
		t.Fatalf("check: %d %s", w.Code, w.Body)
	}
	var checked struct {
		Checked int `json:"checked"`
		Added   int `json:"added"`
	}
	json.Unmarshal(w.Body.Bytes(), &checked)
	if checked.Checked != 1 || checked.Added != 1 {
		t.Fatalf("checked: %+v", checked)
	}
	after := spineOf(t, s, made.ID)
	if len(after) != 2 || after[0].Title != "Agents that read the web" {
		t.Fatalf("the new piece is not on top: %+v", after)
	}
	if after[1].Unit != first[0].Unit {
		t.Errorf("the piece already there changed chapter: %q became %q", first[0].Unit, after[1].Unit)
	}

	// Nothing new, nothing done.
	w = do(t, s, "POST", "/feeds/refresh?force=1", "", "")
	json.Unmarshal(w.Body.Bytes(), &checked)
	if checked.Added != 0 {
		t.Errorf("a second check added %d", checked.Added)
	}
	if again := spineOf(t, s, made.ID); len(again) != 2 {
		t.Errorf("the spine changed on a check with nothing new: %+v", again)
	}
}

// A check left to itself does not ask a blog it asked a moment ago.
func TestAFeedIsNotAskedTwiceInAnHour(t *testing.T) {
	s := newServer(t, "")
	s.cfg.ReadingsDir = t.TempDir()
	b := &blog{}
	b.add(longEntry("injection", "Prompt injection in 2026", "2026-09-14"))
	site := feedSite(t, b)
	do(t, s, "POST", "/feeds", `{"url":"`+site.URL+`/atom/"}`, "")

	w := do(t, s, "POST", "/feeds/refresh", "", "")
	var checked struct {
		Checked int `json:"checked"`
	}
	json.Unmarshal(w.Body.Bytes(), &checked)
	if checked.Checked != 0 {
		t.Errorf("a blog just followed was asked again: %+v", checked)
	}
}

// Following something that is not a feed says so and leaves no card.
func TestFollowingSomethingThatIsNotAFeedIsRefused(t *testing.T) {
	s := newServer(t, "")
	s.cfg.ReadingsDir = t.TempDir()
	site := feedSite(t, &blog{})
	if w := do(t, s, "POST", "/feeds", `{"url":"`+site.URL+`/atom/"}`, ""); w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("an empty feed -> %d %s", w.Code, w.Body)
	}
	w := do(t, s, "GET", "/feeds", "", "")
	var list struct {
		Feeds []any `json:"feeds"`
	}
	json.Unmarshal(w.Body.Bytes(), &list)
	if len(list.Feeds) != 0 {
		t.Errorf("a card was left behind: %+v", list.Feeds)
	}
}

// The shelf card counts what has not been read, and a chapter opened
// stops counting. A week of notes that gains another note is unread
// again: there is something in it the reader has not seen.
func TestAFollowedBlogCountsWhatIsUnread(t *testing.T) {
	s := newServer(t, "")
	s.cfg.ReadingsDir = t.TempDir()
	b := &blog{}
	b.add(longEntry("injection", "Prompt injection in 2026", "2026-09-14"))
	b.add(shortEntry("quote-1", "Quoting someone", "2026-09-15"))
	site := feedSite(t, b)
	w := do(t, s, "POST", "/feeds", `{"url":"`+site.URL+`/atom/"}`, "")
	var made struct {
		ID string `json:"id"`
	}
	json.Unmarshal(w.Body.Bytes(), &made)

	if n := unreadOf(t, s, made.ID); n != 2 {
		t.Fatalf("a blog just followed has %d unread, not 2", n)
	}
	spine := spineOf(t, s, made.ID)
	w = do(t, s, "POST", "/readings/"+made.ID+"/read", `{"unit":"`+spine[0].Unit+`"}`, "")
	if w.Code != http.StatusOK {
		t.Fatalf("mark read: %d %s", w.Code, w.Body)
	}
	if n := unreadOf(t, s, made.ID); n != 1 {
		t.Errorf("after reading one: %d unread", n)
	}
	// Reading the same chapter twice is still one read chapter.
	do(t, s, "POST", "/readings/"+made.ID+"/read", `{"unit":"`+spine[0].Unit+`"}`, "")
	if n := unreadOf(t, s, made.ID); n != 1 {
		t.Errorf("after reading it again: %d unread", n)
	}

	// The week that was read gains a note, so it is unread again.
	for _, unit := range spine {
		do(t, s, "POST", "/readings/"+made.ID+"/read", `{"unit":"`+unit.Unit+`"}`, "")
	}
	if n := unreadOf(t, s, made.ID); n != 0 {
		t.Fatalf("after reading everything: %d unread", n)
	}
	b.add(shortEntry("quote-2", "Quoting someone else", "2026-09-16"))
	do(t, s, "POST", "/feeds/refresh?force=1", "", "")
	if n := unreadOf(t, s, made.ID); n != 1 {
		t.Errorf("a week with a new note in it: %d unread", n)
	}
}

func unreadOf(t *testing.T, s *Server, id string) int {
	t.Helper()
	w := do(t, s, "GET", "/subjects", "", "")
	var shelf struct {
		Subjects []struct {
			ID     string `json:"id"`
			Unread int    `json:"unread"`
		} `json:"subjects"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &shelf); err != nil {
		t.Fatal(err)
	}
	for _, row := range shelf.Subjects {
		if row.ID == id {
			return row.Unread
		}
	}
	t.Fatalf("%s is not on the shelf", id)
	return 0
}
