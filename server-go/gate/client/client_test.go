package client

import (
	"bufio"
	"context"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/mjbraun/chiron/server/gate"
)

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

// freePort reserves an address and releases it, so a server can appear
// there later, the way a waking sprite does.
func freePort(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := ln.Addr().String()
	ln.Close()
	return addr
}

func serveGate(t *testing.T, addr string) {
	t.Helper()
	up, _ := url.Parse("http://127.0.0.1:1")
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	srv := &http.Server{Handler: gate.Handler(gate.Config{Key: "k1", Upstream: up, SSHAddr: echoListener(t)})}
	go srv.Serve(ln)
	t.Cleanup(func() { srv.Close() })
}

func TestDialWaitsForTheSpriteToWake(t *testing.T) {
	addr := freePort(t)
	go func() {
		time.Sleep(1500 * time.Millisecond)
		serveGate(t, addr)
	}()
	var attempts []string
	conn, err := Dial(context.Background(), "http://"+addr, "k1", 10*time.Second,
		func(s string) { attempts = append(attempts, s) })
	if err != nil {
		t.Fatalf("dial: %v (attempts %v)", err, attempts)
	}
	defer conn.Close()
	if len(attempts) == 0 {
		t.Fatal("expected at least one failed attempt before the gate came up")
	}
	if _, err := conn.Write([]byte("ping\n")); err != nil {
		t.Fatal(err)
	}
	got, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil || got != "ping\n" {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestDialGivesUp(t *testing.T) {
	addr := freePort(t)
	start := time.Now()
	_, err := Dial(context.Background(), "http://"+addr, "k1", 1500*time.Millisecond, nil)
	if err == nil {
		t.Fatal("dial succeeded with nothing listening")
	}
	if time.Since(start) < 1500*time.Millisecond || time.Since(start) > 10*time.Second {
		t.Fatalf("gave up after %s", time.Since(start))
	}
}

func TestDialReportsWrongKeyWithoutRetrying(t *testing.T) {
	addr := freePort(t)
	serveGate(t, addr)
	start := time.Now()
	_, err := Dial(context.Background(), "http://"+addr, "wrong", 10*time.Second, nil)
	if err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("got %v", err)
	}
	if time.Since(start) > 2*time.Second {
		t.Fatal("a refused key should not be retried")
	}
}

func TestPipeCarriesBothDirections(t *testing.T) {
	addr := freePort(t)
	serveGate(t, addr)
	conn, err := Dial(context.Background(), "http://"+addr, "k1", 5*time.Second, nil)
	if err != nil {
		t.Fatal(err)
	}
	inR, inW := io.Pipe()
	outR, outW := io.Pipe()
	finished := make(chan struct{})
	go func() { Pipe(conn, inR, outW); outW.Close(); close(finished) }()

	inW.Write([]byte("hello\n"))
	got, err := bufio.NewReader(outR).ReadString('\n')
	if err != nil || got != "hello\n" {
		t.Fatalf("got %q, %v", got, err)
	}
	// ssh exiting closes our stdin; Pipe must return rather than hang.
	inW.Close()
	select {
	case <-finished:
	case <-time.After(5 * time.Second):
		t.Fatal("Pipe did not return after stdin closed")
	}
}
