// Package gate is the front door on the sprite's one public port. It owns
// :8080, tunnels SSH over a WebSocket at /ssh, and reverse-proxies every
// other request to the book server. It carries no book logic, so a bad book
// deploy never takes the way in down with it, and it changes rarely.
package gate

import (
	"context"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/coder/websocket"

	"github.com/mjbraun/chiron/server/auth"
)

type Config struct {
	// Key is the shared client key, the same one the book server checks.
	// It opens the tunnel; sshd then demands an SSH key of its own.
	Key string
	// Upstream is the book server, on loopback.
	Upstream *url.URL
	// SSHAddr is where sshd listens, on loopback.
	SSHAddr string
}

// The largest frame a client may send. io.Copy on the far side writes in
// 32 KiB pieces, so the default 32 KiB limit sat exactly on the edge.
const readLimit = 1 << 20

func Handler(cfg Config) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/ssh", func(w http.ResponseWriter, r *http.Request) {
		if !auth.Authorized(r, cfg.Key) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		tunnel(w, r, cfg.SSHAddr)
	})
	proxy := httputil.NewSingleHostReverseProxy(cfg.Upstream)
	proxy.ErrorLog = log.Default()
	mux.Handle("/", proxy)
	return mux
}

// tunnel pipes a WebSocket to sshd until either side closes. The target is
// fixed by configuration; nothing in the request chooses where bytes go.
func tunnel(w http.ResponseWriter, r *http.Request, target string) {
	ws, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		// The clients are chiron-dev and the iPad app, not browsers, so
		// there is no origin to check.
		InsecureSkipVerify: true,
		CompressionMode:    websocket.CompressionDisabled,
	})
	if err != nil {
		// Accept has already written the 4xx.
		return
	}
	ws.SetReadLimit(readLimit)

	sshConn, err := net.DialTimeout("tcp", target, 5*time.Second)
	if err != nil {
		log.Printf("gate: sshd at %s: %v", target, err)
		ws.Close(websocket.StatusInternalError, "sshd unreachable")
		return
	}
	defer sshConn.Close()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	wsConn := websocket.NetConn(ctx, ws, websocket.MessageBinary)

	done := make(chan struct{}, 2)
	go func() { io.Copy(sshConn, wsConn); done <- struct{}{} }()
	go func() { io.Copy(wsConn, sshConn); done <- struct{}{} }()
	<-done
	// One direction ended; closing both sides unblocks the other copy.
	sshConn.Close()
	ws.Close(websocket.StatusNormalClosure, "")
}
