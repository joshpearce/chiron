package devagent

import (
	"context"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mjbraun/chiron/server/devreq"
)

// The sprite's API as the runtime serves it: HTTP on a unix socket. This
// one records what it was asked.
func spriteAPI(t *testing.T) (socket string, asked func() []string) {
	dir, err := filepathShort(t)
	if err != nil {
		t.Fatal(err)
	}
	socket = filepath.Join(dir, "api.sock")
	ln, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	var mu sync.Mutex
	var got []string
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		got = append(got, strings.TrimSpace(r.Method+" "+r.URL.Path+" "+string(body)))
		mu.Unlock()
		w.Write([]byte(`{}`))
	})}
	go srv.Serve(ln)
	t.Cleanup(func() { srv.Shutdown(context.Background()) })
	return socket, func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), got...)
	}
}

// A unix socket path must be short (about 100 bytes); t.TempDir() on a
// Mac is not.
func filepathShort(t *testing.T) (string, error) {
	dir, err := os.MkdirTemp("/tmp", "sock")
	if err != nil {
		return "", err
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir, nil
}

// A task is made, refreshed and deleted through the sprite's API, with
// its expiry in seconds.
func TestSpriteTasksHoldRenewAndRelease(t *testing.T) {
	socket, asked := spriteAPI(t)
	k := SpriteTasks{Socket: socket}
	if err := k.Hold("request-x", 15*time.Minute); err != nil {
		t.Fatal(err)
	}
	if err := k.Renew("request-x", 15*time.Minute); err != nil {
		t.Fatal(err)
	}
	if err := k.Release("request-x"); err != nil {
		t.Fatal(err)
	}
	want := []string{
		`POST /v1/tasks {"name":"request-x","expire":900}`,
		`PUT /v1/tasks/request-x {"expire":900}`,
		`DELETE /v1/tasks/request-x`,
	}
	got := asked()
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("asked\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestSpriteTasksSayWhenTheAPIRefuses(t *testing.T) {
	dir, _ := filepathShort(t)
	socket := filepath.Join(dir, "api.sock")
	ln, _ := net.Listen("unix", socket)
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "expire too long", http.StatusBadRequest)
	})}
	go srv.Serve(ln)
	defer srv.Shutdown(context.Background())
	err := SpriteTasks{Socket: socket}.Hold("request-x", 2*time.Hour)
	if err == nil || !strings.Contains(err.Error(), "expire too long") {
		t.Fatalf("err = %v, want the API's reason", err)
	}
}

// Records holds, renewals and releases.
type fakeKeeper struct {
	mu     sync.Mutex
	events []string
}

func (k *fakeKeeper) note(s string) error {
	k.mu.Lock()
	defer k.mu.Unlock()
	k.events = append(k.events, s)
	return nil
}
func (k *fakeKeeper) Hold(name string, d time.Duration) error  { return k.note("hold " + name) }
func (k *fakeKeeper) Renew(name string, d time.Duration) error { return k.note("renew " + name) }
func (k *fakeKeeper) Release(name string) error                { return k.note("release " + name) }
func (k *fakeKeeper) seen() []string {
	k.mu.Lock()
	defer k.mu.Unlock()
	return append([]string(nil), k.events...)
}

// The sprite sleeps when no one is connected and the request's work
// freezes with it, so the agent holds it awake from taking a request to
// finishing it, renewing the hold while the work runs.
func TestARequestHoldsTheSpriteAwakeUntilItIsDone(t *testing.T) {
	x := &fakeExec{
		answer: map[string]string{"git log": "abc123 A change\n", "git diff": "README.md\n", "git rev-parse": "abc123\n"},
		pause:  60 * time.Millisecond,
	}
	a, store := agent(t, x)
	k := &fakeKeeper{}
	a.Awake, a.RenewEvery = k, 10*time.Millisecond
	r, _ := store.Create("A change", nil, nil)

	if took, err := a.Once(); !took || err != nil {
		t.Fatalf("Once = %v, %v", took, err)
	}
	ev := k.seen()
	name := "request-" + r.ID
	if len(ev) < 3 || ev[0] != "hold "+name || ev[len(ev)-1] != "release "+name {
		t.Fatalf("events %v: want a hold first and a release last", ev)
	}
	renewals := 0
	for _, e := range ev[1 : len(ev)-1] {
		if e != "renew "+name {
			t.Fatalf("events %v: only renewals between the hold and the release", ev)
		}
		renewals++
	}
	if renewals == 0 {
		t.Fatalf("events %v: the hold was never renewed while the work ran", ev)
	}
	if got, _ := store.Get(r.ID); got.Status != devreq.Ready {
		t.Fatalf("status %s", got.Status)
	}
}

// A request that fails lets the sprite sleep again too.
func TestAFailedRequestReleasesTheSprite(t *testing.T) {
	x := &fakeExec{answer: map[string]string{"git log": ""}}
	a, store := agent(t, x)
	k := &fakeKeeper{}
	a.Awake = k
	r, _ := store.Create("A change", nil, nil)
	a.Once()
	ev := k.seen()
	if len(ev) < 2 || ev[len(ev)-1] != "release request-"+r.ID {
		t.Fatalf("events %v: want the hold released", ev)
	}
	if got, _ := store.Get(r.ID); got.Status != devreq.Failed {
		t.Fatalf("status %s", got.Status)
	}
}
