package primer

import (
	"strings"
	"testing"
	"time"

	"github.com/mjbraun/chiron/server/corpus"
)

func TestWriteDocLaysOutALoadableCorpus(t *testing.T) {
	root := t.TempDir()
	m := &Meta{ID: "primer-robots", Title: "Robots: \"the\" file", Status: StatusAuthoring, CapturedAt: time.Now()}
	if err := Save(root, m); err != nil {
		t.Fatal(err)
	}
	doc := "A crawler reads robots.txt first.\n\n## What the file says\n\nThree directives.\n\n## Why it matters\n\nMoney.\n"
	if err := WriteDoc(root, m.ID, m.Title, doc); err != nil {
		t.Fatal(err)
	}
	c, err := corpus.Load(CorpusDir(root, m.ID))
	if err != nil {
		t.Fatal(err)
	}
	u, ok := c.Units[UnitID]
	if !ok || len(u.Sections) != 2 || u.Sections[1].Heading != "Why it matters" || u.Title != m.Title {
		t.Fatalf("unit = %+v", u)
	}
	if c.Syllabus.Title != m.Title {
		t.Fatalf("syllabus title %q", c.Syllabus.Title)
	}

	newDoc, err := AppendSection(root, m.ID, m.Title, "A margin note", "The reader asked; here is more.")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(strings.TrimSpace(newDoc), "here is more.") {
		t.Fatalf("doc = %q", newDoc)
	}
	c, _ = corpus.Load(CorpusDir(root, m.ID))
	if n := len(c.Units[UnitID].Sections); n != 3 {
		t.Fatalf("%d sections after append", n)
	}

	all, err := LoadAll(root)
	if err != nil || len(all) != 1 || all[0].ID != m.ID {
		t.Fatalf("LoadAll = %+v, %v", all, err)
	}
}

func TestADocWithoutHeadingsGetsOne(t *testing.T) {
	root := t.TempDir()
	if err := WriteDoc(root, "primer-x", "Plain", "Just a paragraph."); err != nil {
		t.Fatal(err)
	}
	c, err := corpus.Load(CorpusDir(root, "primer-x"))
	if err != nil {
		t.Fatal(err)
	}
	if s := c.Units[UnitID].Sections; len(s) != 1 || s[0].Heading != "Plain" {
		t.Fatalf("sections = %+v", s)
	}
}
