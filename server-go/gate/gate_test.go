package gate

import (
	"bufio"
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

// echoListener stands in for sshd: whatever arrives on a connection comes
// straight back.
func echoListener(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func() { io.Copy(c, c); c.Close() }()
		}
	}()
	return ln.Addr().String()
}

func newGate(t *testing.T, upstream string) *httptest.Server {
	t.Helper()
	up, err := url.Parse(upstream)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(Handler(Config{Key: "k1", Upstream: up, SSHAddr: echoListener(t)}))
	t.Cleanup(srv.Close)
	return srv
}

func wsURL(srv *httptest.Server, path string) string {
	return "ws" + strings.TrimPrefix(srv.URL, "http") + path
}

func TestSSHRefusesWithoutKey(t *testing.T) {
	srv := newGate(t, "http://127.0.0.1:1")
	for _, auth := range []string{"", "Bearer wrong"} {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		opts := &websocket.DialOptions{HTTPHeader: http.Header{}}
		if auth != "" {
			opts.HTTPHeader.Set("Authorization", auth)
		}
		_, resp, err := websocket.Dial(ctx, wsURL(srv, "/ssh"), opts)
		if err == nil {
			t.Fatalf("auth %q: dial succeeded", auth)
		}
		if resp == nil || resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("auth %q: want 401, got %v", auth, resp)
		}
	}
}

func TestSSHPipesBytesToTarget(t *testing.T) {
	srv := newGate(t, "http://127.0.0.1:1")
	for _, way := range []string{"header", "query"} {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		u := wsURL(srv, "/ssh")
		opts := &websocket.DialOptions{HTTPHeader: http.Header{}}
		if way == "header" {
			opts.HTTPHeader.Set("Authorization", "Bearer k1")
		} else {
			u += "?token=k1"
		}
		c, _, err := websocket.Dial(ctx, u, opts)
		if err != nil {
			t.Fatalf("%s: dial: %v", way, err)
		}
		conn := websocket.NetConn(ctx, c, websocket.MessageBinary)
		// Two writes, so framing on the way through is exercised.
		if _, err := conn.Write([]byte("SSH-2.0-test\r\n")); err != nil {
			t.Fatal(err)
		}
		if _, err := conn.Write([]byte("more\n")); err != nil {
			t.Fatal(err)
		}
		r := bufio.NewReader(conn)
		for _, want := range []string{"SSH-2.0-test\r\n", "more\n"} {
			got, err := r.ReadString('\n')
			if err != nil || got != want {
				t.Fatalf("%s: got %q, %v; want %q", way, got, err, want)
			}
		}
		conn.Close()
	}
}

func TestSSHClosesWhenTargetCloses(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		c, err := ln.Accept()
		if err == nil {
			c.Write([]byte("bye\n"))
			c.Close()
		}
	}()
	up, _ := url.Parse("http://127.0.0.1:1")
	srv := httptest.NewServer(Handler(Config{Key: "k1", Upstream: up, SSHAddr: ln.Addr().String()}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c, _, err := websocket.Dial(ctx, wsURL(srv, "/ssh?token=k1"), nil)
	if err != nil {
		t.Fatal(err)
	}
	conn := websocket.NetConn(ctx, c, websocket.MessageBinary)
	all, err := io.ReadAll(conn)
	if string(all) != "bye\n" {
		t.Fatalf("got %q, %v", all, err)
	}
}

func TestOtherPathsProxyToUpstream(t *testing.T) {
	var gotPath, gotAuth string
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.RequestURI()
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTeapot)
		io.WriteString(w, `{"ok":true}`)
	}))
	defer up.Close()
	srv := newGate(t, up.URL)

	req, _ := http.NewRequest("GET", srv.URL+"/subjects?x=1", nil)
	req.Header.Set("Authorization", "Bearer whatever")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusTeapot || string(body) != `{"ok":true}` {
		t.Fatalf("got %d %q", resp.StatusCode, body)
	}
	if gotPath != "/subjects?x=1" || gotAuth != "Bearer whatever" {
		t.Fatalf("upstream saw %q %q", gotPath, gotAuth)
	}
	if resp.Header.Get("Content-Type") != "application/json" {
		t.Fatalf("content type not passed through: %v", resp.Header)
	}
}

func TestUpstreamDownIsBadGateway(t *testing.T) {
	srv := newGate(t, "http://127.0.0.1:1")
	resp, err := http.Get(srv.URL + "/ping")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadGateway {
		t.Fatalf("got %d", resp.StatusCode)
	}
}

func TestSSHWithoutUpgradeIsRejected(t *testing.T) {
	srv := newGate(t, "http://127.0.0.1:1")
	resp, err := http.Get(srv.URL + "/ssh?token=k1")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode < 400 || resp.StatusCode >= 500 {
		t.Fatalf("plain GET on /ssh got %d", resp.StatusCode)
	}
}
