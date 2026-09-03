// Package agent lets a program on the sprite drive the iPad app. The iPad
// has no inbound path, so the app connects out to the book server and holds
// a WebSocket open; commands go down it and results come back. One app at a
// time: a new connection replaces the old one.
package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
)

var ErrNoApp = errors.New("no app is connected")

// Command goes to the app; Reply comes back with the same id.
type Command struct {
	ID   int64           `json:"id"`
	Verb string          `json:"verb"`
	Args json.RawMessage `json:"args,omitempty"`
}

type Reply struct {
	ID     int64           `json:"id"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  string          `json:"error,omitempty"`
}

// A screenshot is a PNG in base64; give it room.
const readLimit = 16 << 20

type Hub struct {
	mu      sync.Mutex
	app     *websocket.Conn
	appName string
	pending map[int64]chan Reply
	nextID  int64
}

func NewHub() *Hub {
	return &Hub{pending: map[int64]chan Reply{}}
}

// Connected reports whether an app is on the line, and what it calls itself.
func (h *Hub) Connected() (bool, string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.app != nil, h.appName
}

// ServeApp is the app's end: accept the WebSocket and read replies until it
// goes away. The caller has already checked the shared key.
func (h *Hub) ServeApp(w http.ResponseWriter, r *http.Request) {
	ws, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
	if err != nil {
		return
	}
	ws.SetReadLimit(readLimit)
	name := r.URL.Query().Get("name")

	h.mu.Lock()
	if h.app != nil {
		h.app.Close(websocket.StatusGoingAway, "another app connected")
	}
	h.app = ws
	h.appName = name
	h.mu.Unlock()

	ctx := r.Context()
	for {
		_, data, err := ws.Read(ctx)
		if err != nil {
			break
		}
		var rep Reply
		if json.Unmarshal(data, &rep) != nil {
			continue
		}
		h.mu.Lock()
		ch := h.pending[rep.ID]
		delete(h.pending, rep.ID)
		h.mu.Unlock()
		if ch != nil {
			ch <- rep
		}
	}
	h.mu.Lock()
	if h.app == ws {
		h.app = nil
		h.appName = ""
	}
	h.mu.Unlock()
	ws.Close(websocket.StatusNormalClosure, "")
}

// Command sends one verb to the app and waits for its reply.
func (h *Hub) Command(ctx context.Context, verb string, args json.RawMessage) (json.RawMessage, error) {
	h.mu.Lock()
	ws := h.app
	if ws == nil {
		h.mu.Unlock()
		return nil, ErrNoApp
	}
	h.nextID++
	id := h.nextID
	ch := make(chan Reply, 1)
	h.pending[id] = ch
	h.mu.Unlock()

	msg, _ := json.Marshal(Command{ID: id, Verb: verb, Args: args})
	wctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	err := ws.Write(wctx, websocket.MessageText, msg)
	cancel()
	if err != nil {
		h.forget(id)
		return nil, fmt.Errorf("send to app: %w", err)
	}
	select {
	case rep := <-ch:
		if rep.Error != "" {
			return nil, fmt.Errorf("app: %s", rep.Error)
		}
		return rep.Result, nil
	case <-ctx.Done():
		h.forget(id)
		return nil, ctx.Err()
	}
}

func (h *Hub) forget(id int64) {
	h.mu.Lock()
	delete(h.pending, id)
	h.mu.Unlock()
}
