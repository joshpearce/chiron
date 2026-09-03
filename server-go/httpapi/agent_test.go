package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"

	"github.com/mjbraun/chiron/server/agent"
)

// A command posted to the server reaches the connected app and its answer
// comes back; without an app the caller is told so; and the app's socket
// is behind the shared key like every other route.
func TestAgentCommandsReachTheApp(t *testing.T) {
	s := newServer(t, "k1")
	srv := httptest.NewServer(s.Handler())
	defer srv.Close()
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/agent/app?name=sim"

	post := func(body string) (int, string) {
		req, _ := http.NewRequest("POST", srv.URL+"/agent/cmd", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer k1")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, string(b)
	}

	if code, body := post(`{"verb":"state"}`); code != http.StatusServiceUnavailable {
		t.Fatalf("without an app: %d %s", code, body)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, resp, err := websocket.Dial(ctx, wsURL, nil); err == nil || resp == nil || resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("app socket without the key: err=%v resp=%v", err, resp)
	}
	ws, _, err := websocket.Dial(ctx, wsURL+"&token=k1", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer ws.Close(websocket.StatusNormalClosure, "")
	go func() {
		for {
			_, data, err := ws.Read(context.Background())
			if err != nil {
				return
			}
			var cmd agent.Command
			json.Unmarshal(data, &cmd)
			rep, _ := json.Marshal(agent.Reply{ID: cmd.ID, Result: json.RawMessage(`{"screen":"reading","verb":"` + cmd.Verb + `"}`)})
			ws.Write(context.Background(), websocket.MessageText, rep)
		}
	}()
	for i := 0; i < 50; i++ {
		if ok, _ := s.hub.Connected(); ok {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	code, body := post(`{"verb":"open","args":{"subject":"ai"}}`)
	if code != http.StatusOK || !strings.Contains(body, `"verb":"open"`) {
		t.Fatalf("with an app: %d %s", code, body)
	}
	req, _ := http.NewRequest("GET", srv.URL+"/agent/status", nil)
	req.Header.Set("Authorization", "Bearer k1")
	resp, _ := http.DefaultClient.Do(req)
	b, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(b), `"connected":true`) || !strings.Contains(string(b), `"app":"sim"`) {
		t.Fatalf("status: %s", b)
	}
}
