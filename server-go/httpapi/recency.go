package httpapi

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// The shelf can be ordered by recency, so every row carries updated_at:
// the latest of when it arrived (a capture, an import, a build), when
// it last changed (a feed's newest post, a page turned in a document,
// an exchange in a book) and when the reader last opened it. The last
// of those is the one the server has to remember itself, in
// opened.json beside the active subject: a page fetched is a book
// opened, and the stamp is written at most once a minute per subject
// so a page turn is not a disk write.

const openedFile = "opened.json"

// opened is the record of when each subject was last opened.
type opened struct {
	mu   sync.Mutex
	path string
	at   map[string]time.Time
}

func loadOpened(activePath string) *opened {
	o := &opened{at: map[string]time.Time{}}
	if activePath == "" {
		return o
	}
	o.path = filepath.Join(filepath.Dir(activePath), openedFile)
	data, err := os.ReadFile(o.path)
	if err != nil {
		return o
	}
	var raw map[string]string
	if json.Unmarshal(data, &raw) != nil {
		return o
	}
	for id, stamp := range raw {
		if t, err := time.Parse(time.RFC3339, stamp); err == nil {
			o.at[id] = t
		}
	}
	return o
}

// mark stamps id as opened now, and writes the record if the stamp it
// replaces is more than a minute old.
func (o *opened) mark(id string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	now := time.Now().Truncate(time.Second)
	if last, ok := o.at[id]; ok && now.Sub(last) < time.Minute {
		return
	}
	o.at[id] = now
	if o.path == "" {
		return
	}
	raw := make(map[string]string, len(o.at))
	for id, t := range o.at {
		raw[id] = t.Format(time.RFC3339)
	}
	data, _ := json.Marshal(raw)
	if err := os.WriteFile(o.path, data, 0o644); err != nil {
		log.Printf("opened: %v", err)
	}
}

func (o *opened) of(id string) time.Time {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.at[id]
}

// forget drops a subject that is gone from the shelf.
func (o *opened) forget(id string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if _, ok := o.at[id]; !ok {
		return
	}
	delete(o.at, id)
	if o.path == "" {
		return
	}
	raw := make(map[string]string, len(o.at))
	for id, t := range o.at {
		raw[id] = t.Format(time.RFC3339)
	}
	data, _ := json.Marshal(raw)
	os.WriteFile(o.path, data, 0o644)
}

// markOpened records that the reader has id open, without making it the
// active subject: a document is read but never "active" the way a book
// is.
func (s *Server) markOpened(id string) { s.opened.mark(id) }

// updatedAt is the row's recency: the latest of the given moments and
// when it was last opened, as RFC3339, or "" if nothing is known.
func (s *Server) updatedAt(id string, moments ...time.Time) string {
	latest := s.opened.of(id)
	for _, t := range moments {
		if t.After(latest) {
			latest = t
		}
	}
	if latest.IsZero() {
		return ""
	}
	return latest.UTC().Format(time.RFC3339)
}

// stamp parses an RFC3339 stamp kept in a record; a missing or malformed
// one is the zero time.
func stamp(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

// mtime is when a file or directory last changed; zero if it is not there.
func mtime(path string) time.Time {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}

// subjectUpdated is a registered subject's recency: its corpus (a built
// book's build), the learner record (an exchange, a read mark), what
// it came from (a capture, an import) and for a feed its newest post.
func (s *Server) subjectUpdated(sub *Subject) string {
	moments := []time.Time{
		mtime(sub.Corpus.Dir),
		mtime(filepath.Join(sub.StateDir, "learner.json")),
	}
	if sub.Primer != nil {
		moments = append(moments, sub.Primer.CapturedAt)
	}
	if sub.Kind == KindReading || sub.Kind == KindFeed {
		root := s.readingsRoot()
		if data, err := os.ReadFile(readingMeta(root, sub.ID)); err == nil {
			var m Reading
			if json.Unmarshal(data, &m) == nil {
				moments = append(moments, stamp(m.ImportedAt))
			}
		}
		if data, err := os.ReadFile(feedMeta(root, sub.ID)); err == nil {
			var f Subscription
			if json.Unmarshal(data, &f) == nil {
				for _, it := range f.Items {
					moments = append(moments, stamp(it.Updated), stamp(it.Published))
				}
			}
		}
	}
	return s.updatedAt(sub.ID, moments...)
}
