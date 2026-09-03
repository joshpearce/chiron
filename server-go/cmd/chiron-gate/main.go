// Command chiron-gate owns the sprite's public port: SSH over a WebSocket
// at /ssh, everything else proxied to the book server on loopback.
package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/mjbraun/chiron/server/gate"
)

func main() {
	listen := flag.String("listen", "0.0.0.0:8080", "public listen address")
	upstream := flag.String("upstream", "http://127.0.0.1:8081", "the book server")
	sshAddr := flag.String("ssh", "127.0.0.1:2222", "where sshd listens")
	flag.Parse()

	// The same key the book server checks, from the same variable. The gate
	// is only ever on a public URL, so an empty key is a mistake, not a mode.
	key := strings.TrimSpace(os.Getenv("CHIRON_AUTH_TOKEN"))
	if key == "" {
		log.Fatal("CHIRON_AUTH_TOKEN is required")
	}
	up, err := url.Parse(*upstream)
	if err != nil {
		log.Fatalf("upstream: %v", err)
	}

	server := &http.Server{
		Addr:    *listen,
		Handler: gate.Handler(gate.Config{Key: key, Upstream: up, SSHAddr: *sshAddr}),
		// Tunnels and chapter generation both live for a long time; the
		// header timeout is the only one that makes sense here.
		ReadHeaderTimeout: 15 * time.Second,
	}
	go func() {
		log.Printf("chiron-gate on %s: /ssh -> %s, else -> %s", *listen, *sshAddr, *upstream)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	server.Shutdown(ctx)
}
