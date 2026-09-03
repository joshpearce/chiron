package agent

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

// fakeApp connects like the iPad does and answers every command by
// echoing the verb and args back.
func fakeApp(t *testing.T, srv *httptest.Server, name string) *websocket.Conn {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ws, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(srv.URL, "http")+"/agent/app?name="+name, nil)
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		for {
			_, data, err := ws.Read(context.Background())
			if err != nil {
				return
			}
			var cmd Command
			json.Unmarshal(data, &cmd)
			result, _ := json.Marshal(map[string]any{"verb": cmd.Verb, "args": cmd.Args, "app": name})
			rep, _ := json.Marshal(Reply{ID: cmd.ID, Result: result})
			ws.Write(context.Background(), websocket.MessageText, rep)
		}
	}()
	return ws
}

func waitConnected(t *testing.T, h *Hub, name string) {
	t.Helper()
	for i := 0; i < 50; i++ {
		if ok, n := h.Connected(); ok && n == name {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("app %q never connected", name)
}

func TestCommandRoundTrips(t *testing.T) {
	h := NewHub()
	srv := httptest.NewServer(http.HandlerFunc(h.ServeApp))
	defer srv.Close()
	ws := fakeApp(t, srv, "ipad")
	defer ws.Close(websocket.StatusNormalClosure, "")
	waitConnected(t, h, "ipad")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, err := h.Command(ctx, "open", json.RawMessage(`{"subject":"ai"}`))
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Verb string          `json:"verb"`
		Args json.RawMessage `json:"args"`
	}
	json.Unmarshal(res, &got)
	if got.Verb != "open" || string(got.Args) != `{"subject":"ai"}` {
		t.Fatalf("app saw %s", res)
	}
}

func TestNoAppIsAnError(t *testing.T) {
	h := NewHub()
	_, err := h.Command(context.Background(), "state", nil)
	if !errors.Is(err, ErrNoApp) {
		t.Fatalf("got %v", err)
	}
}

func TestCommandTimesOutOnASilentApp(t *testing.T) {
	h := NewHub()
	srv := httptest.NewServer(http.HandlerFunc(h.ServeApp))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ws, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(srv.URL, "http")+"/agent/app?name=mute", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer ws.Close(websocket.StatusNormalClosure, "")
	waitConnected(t, h, "mute")

	short, cancel2 := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel2()
	_, err = h.Command(short, "state", nil)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("got %v", err)
	}
	h.mu.Lock()
	n := len(h.pending)
	h.mu.Unlock()
	if n != 0 {
		t.Fatalf("%d pending commands left behind", n)
	}
}

func TestNewAppReplacesOld(t *testing.T) {
	h := NewHub()
	srv := httptest.NewServer(http.HandlerFunc(h.ServeApp))
	defer srv.Close()
	first := fakeApp(t, srv, "first")
	waitConnected(t, h, "first")
	second := fakeApp(t, srv, "second")
	defer second.Close(websocket.StatusNormalClosure, "")
	waitConnected(t, h, "second")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, err := h.Command(ctx, "state", nil)
	if err != nil || !strings.Contains(string(res), `"app":"second"`) {
		t.Fatalf("res=%s err=%v", res, err)
	}
	// The first app's socket was closed by the hub.
	_, _, err = first.Read(ctx)
	if err == nil {
		t.Fatal("first app still readable")
	}

	second.Close(websocket.StatusNormalClosure, "")
	for i := 0; i < 50; i++ {
		if ok, _ := h.Connected(); !ok {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("hub still thinks an app is connected")
}
