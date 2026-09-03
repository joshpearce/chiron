// Package client opens the gate's SSH tunnel from the Mac side. The
// WebSocket connect is an ordinary HTTPS request to the sprite's public URL,
// so it is also what wakes a hibernated sprite; Dial keeps trying while
// that happens.
package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/coder/websocket"
)

// Same figure as the gate: one io.Copy chunk fits with room to spare.
const readLimit = 1 << 20

// Dial connects to <base>/ssh with the shared key and returns the tunnel as
// a net.Conn. It retries for up to patience while the sprite wakes, calling
// notify (if set) once per failed attempt.
func Dial(ctx context.Context, base, key string, patience time.Duration, notify func(string)) (net.Conn, error) {
	u := "wss://" + strings.TrimPrefix(strings.TrimPrefix(strings.TrimRight(base, "/"), "https://"), "wss://") + "/ssh"
	if strings.HasPrefix(base, "http://") || strings.HasPrefix(base, "ws://") {
		// Local testing against a plain gate.
		u = "ws://" + strings.TrimPrefix(strings.TrimPrefix(strings.TrimRight(base, "/"), "http://"), "ws://") + "/ssh"
	}
	opts := &websocket.DialOptions{
		HTTPHeader:      http.Header{"Authorization": {"Bearer " + key}},
		CompressionMode: websocket.CompressionDisabled,
	}
	deadline := time.Now().Add(patience)
	var last error
	for attempt := 1; ; attempt++ {
		actx, cancel := context.WithTimeout(ctx, 15*time.Second)
		ws, resp, err := websocket.Dial(actx, u, opts)
		cancel()
		if err == nil {
			ws.SetReadLimit(readLimit)
			// The tunnel outlives this call; the caller closes the conn.
			return websocket.NetConn(context.Background(), ws, websocket.MessageBinary), nil
		}
		if resp != nil && resp.StatusCode == http.StatusUnauthorized {
			return nil, errors.New("the gate refused the key (401)")
		}
		last = err
		if notify != nil {
			notify(fmt.Sprintf("attempt %d: %v", attempt, err))
		}
		if time.Now().After(deadline) || ctx.Err() != nil {
			return nil, fmt.Errorf("no tunnel after %s: %w", patience, last)
		}
		select {
		case <-time.After(time.Second):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

// Pipe copies in to the tunnel and the tunnel to out until either side
// ends. Under ssh's ProxyCommand, in and out are ssh's own pipes.
func Pipe(conn net.Conn, in io.Reader, out io.Writer) {
	done := make(chan struct{}, 2)
	go func() { io.Copy(conn, in); done <- struct{}{} }()
	go func() { io.Copy(out, conn); done <- struct{}{} }()
	<-done
	conn.Close()
}
