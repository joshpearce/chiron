package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// A server that answers the way the book server does, remembering what it
// was asked, so each command is checked for the request it makes and the
// reply it prints.
type fake struct {
	t     *testing.T
	polls int
	last  struct {
		method, path, auth string
		body               map[string]any
	}
}

func (f *fake) serve(w http.ResponseWriter, r *http.Request) {
	f.last.method, f.last.path, f.last.auth = r.Method, r.URL.RequestURI(), r.Header.Get("Authorization")
	f.last.body = nil
	json.NewDecoder(r.Body).Decode(&f.last.body)
	w.Header().Set("Content-Type", "application/json")
	switch {
	case r.URL.Path == "/subjects":
		f.polls++
		status := "authoring"
		if f.polls >= 2 {
			status = "ready"
		}
		json.NewEncoder(w).Encode(map[string]any{
			"active": "ai",
			"subjects": []map[string]any{
				{"id": "ai", "title": "How AI Works", "kind": "book", "units_total": 14, "units_cleared": 3},
				{"id": "primer-why", "title": "Why?", "kind": "primer", "status": status, "scale": "primer", "progress": "writing"},
			},
		})
	case r.Method == "DELETE" && strings.HasPrefix(r.URL.Path, "/primer/"):
		json.NewEncoder(w).Encode(map[string]any{"subject": strings.TrimPrefix(r.URL.Path, "/primer/"), "deleted": true})
	case r.URL.Path == "/readings/page":
		json.NewEncoder(w).Encode(map[string]any{"id": "read-1", "title": "Prompt injection in 2026", "chapters": 1})
	case r.URL.Path == "/feeds" && r.Method == "POST":
		json.NewEncoder(w).Encode(map[string]any{"id": "read-2", "title": "A Weblog", "chapters": 17, "feed": true})
	case r.URL.Path == "/feeds" && r.Method == "GET":
		json.NewEncoder(w).Encode(map[string]any{"feeds": []map[string]any{
			{"id": "read-2", "title": "A Weblog", "url": "https://ablog.example/atom/", "chapters": 17},
		}})
	case r.URL.Path == "/feeds/refresh":
		json.NewEncoder(w).Encode(map[string]any{"checked": 1, "added": 2})
	case r.URL.Path == "/primer/capture":
		json.NewEncoder(w).Encode(map[string]any{"subject": "primer-why", "status": "planning", "reply_md": "Which part?", "done": false})
	case strings.HasSuffix(r.URL.Path, "/build"):
		json.NewEncoder(w).Encode(map[string]any{"subject": "primer-why", "status": "authoring"})
	case strings.HasSuffix(r.URL.Path, "/plan") && r.Method == "GET":
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"detail":"no draft"}`))
	case strings.HasPrefix(r.URL.Path, "/chapter/") && r.URL.Query().Get("unit") == "":
		// Nothing opened yet: the way the app opens it is the exchange.
		json.NewEncoder(w).Encode(map[string]any{"chapter": nil, "authoring": false})
	case strings.HasPrefix(r.URL.Path, "/chapter/") || r.URL.Path == "/exchange":
		json.NewEncoder(w).Encode(map[string]any{"chapter": map[string]any{
			"unit": "u1", "title": "Why?",
			"html": "<h2>Why</h2><p>Because <em>tokens</em> &amp; weights.</p><ul><li>one</li><li>two</li></ul>",
		}})
	case r.URL.Path == "/teach/jobs" && r.URL.Query().Get("slug") == "building":
		f.polls++
		stage := "authoring"
		if f.polls >= 2 {
			stage = "ready"
		}
		json.NewEncoder(w).Encode(map[string]any{"slug": "building", "stage": stage, "units_done": 3, "units_total": 7})
	default:
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"detail":"no such thing"}`))
	}
}

func TestWaitFollowsABookJobThatIsNotOnTheShelfYet(t *testing.T) {
	_, c := newFake(t)
	out, err := run(c, "wait", []string{"building", "-every", "10ms"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `"id": "building"`) {
		t.Fatalf("out:\n%s", out)
	}
}

func newFake(t *testing.T) (*fake, *client) {
	f := &fake{t: t}
	srv := httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(srv.Close)
	return f, &client{url: srv.URL, key: "k1", http: srv.Client(), patience: time.Second}
}

func TestShelfIsATableWithWhereEachRowStands(t *testing.T) {
	f, c := newFake(t)
	out, err := run(c, "shelf", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if f.last.auth != "Bearer k1" {
		t.Fatalf("auth = %q", f.last.auth)
	}
	if !strings.Contains(out, "3/14 units, open now") || !strings.Contains(out, "authoring, writing") {
		t.Fatalf("shelf:\n%s", out)
	}
}

func TestCaptureSendsThePassageFromStdinAtTheScale(t *testing.T) {
	f, c := newFake(t)
	out, err := run(c, "capture", []string{"-scale", "primer", "-prompt", "why?", "-title", "Why"}, strings.NewReader("the words\n"))
	if err != nil {
		t.Fatal(err)
	}
	if f.last.path != "/primer/capture" || f.last.body["text"] != "the words\n" || f.last.body["scale"] != "primer" || f.last.body["prompt"] != "why?" || f.last.body["title"] != "Why" {
		t.Fatalf("request = %s %+v", f.last.path, f.last.body)
	}
	if !strings.Contains(out, `"subject": "primer-why"`) {
		t.Fatalf("reply:\n%s", out)
	}
}

func TestCaptureNeedsAScaleAndAPrompt(t *testing.T) {
	_, c := newFake(t)
	if _, err := run(c, "capture", []string{"-prompt", "why?"}, strings.NewReader("x")); err == nil {
		t.Fatal("no scale accepted")
	}
	if _, err := run(c, "capture", []string{"-scale", "summary", "-prompt", "why?"}, strings.NewReader("  ")); err == nil {
		t.Fatal("an empty passage accepted")
	}
}

func TestBuildCarriesTheBrief(t *testing.T) {
	f, c := newFake(t)
	if _, err := run(c, "build", []string{"primer-why", "-brief", "from first principles"}, nil); err != nil {
		t.Fatal(err)
	}
	if f.last.path != "/primer/primer-why/build" || f.last.body["brief"] != "from first principles" {
		t.Fatalf("request = %s %+v", f.last.path, f.last.body)
	}
	if _, err := run(c, "build", []string{"primer-why"}, nil); err != nil {
		t.Fatal(err)
	}
	if f.last.body != nil {
		t.Fatalf("a build without a brief sent %+v", f.last.body)
	}
}

func TestWaitFollowsTheRowUntilItIsReady(t *testing.T) {
	f, c := newFake(t)
	out, err := run(c, "wait", []string{"primer-why", "-every", "10ms"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if f.polls != 2 || !strings.Contains(out, `"status": "ready"`) {
		t.Fatalf("polls = %d, out:\n%s", f.polls, out)
	}
}

func TestReadPrintsTheChapterAsText(t *testing.T) {
	f, c := newFake(t)
	out, err := run(c, "read", []string{"ai", "-unit", "u1"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if f.last.path != "/chapter/ai?unit=u1" {
		t.Fatalf("path = %s", f.last.path)
	}
	want := "# Why?\n\nWhy\n\nBecause tokens & weights.\n\none\n\ntwo\n"
	if out != want {
		t.Fatalf("read:\n%q\nwant:\n%q", out, want)
	}
}

func TestReadOpensAPrimerNothingHasOpenedYet(t *testing.T) {
	f, c := newFake(t)
	out, err := run(c, "read", []string{"primer-why"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if f.last.path != "/exchange" || f.last.body["subject"] != "primer-why" || f.last.body["phase"] != "start" {
		t.Fatalf("request = %s %+v", f.last.path, f.last.body)
	}
	if !strings.HasPrefix(out, "# Why?\n") {
		t.Fatalf("read:\n%s", out)
	}
}

func TestAServerErrorIsItsDetail(t *testing.T) {
	_, c := newFake(t)
	_, err := run(c, "discard", []string{"nothing"}, nil)
	if err == nil || !strings.Contains(err.Error(), "no such thing") {
		t.Fatalf("err = %v", err)
	}
}

// A page and a blog reach the shelf from a shell, which is how an agent
// hands the reader something to read.
func TestAPageAndABlogGoOnTheShelfFromTheShell(t *testing.T) {
	f, c := newFake(t)

	out, err := run(c, "page", []string{"https://simonwillison.net/2026/Sep/20/injection/"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if f.last.path != "/readings/page" || f.last.body["url"] != "https://simonwillison.net/2026/Sep/20/injection/" {
		t.Errorf("asked %s %s %v", f.last.method, f.last.path, f.last.body)
	}
	if !strings.Contains(out, "read-1") || !strings.Contains(out, "Prompt injection in 2026") {
		t.Errorf("printed %q", out)
	}

	if _, err := run(c, "follow", []string{"https://simonwillison.net/atom/everything/"}, nil); err != nil {
		t.Fatal(err)
	}
	if f.last.path != "/feeds" || f.last.body["url"] != "https://simonwillison.net/atom/everything/" {
		t.Errorf("asked %s %s %v", f.last.method, f.last.path, f.last.body)
	}

	out, err = run(c, "feeds", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if f.last.method != "GET" || f.last.path != "/feeds" || !strings.Contains(out, "A Weblog") {
		t.Errorf("asked %s %s, printed %q", f.last.method, f.last.path, out)
	}

	out, err = run(c, "feeds", []string{"-check"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if f.last.method != "POST" || f.last.path != "/feeds/refresh?force=1" {
		t.Errorf("asked %s %s", f.last.method, f.last.path)
	}
	if !strings.Contains(out, "2") {
		t.Errorf("printed %q", out)
	}
}

// A primer the reader is done with goes from the shell too.
func TestAPrimerCanBeDeletedFromTheShell(t *testing.T) {
	f, c := newFake(t)
	out, err := run(c, "delete", []string{"primer-why"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if f.last.method != "DELETE" || f.last.path != "/primer/primer-why" {
		t.Errorf("asked %s %s", f.last.method, f.last.path)
	}
	if !strings.Contains(out, "deleted") {
		t.Errorf("printed %q", out)
	}
}
