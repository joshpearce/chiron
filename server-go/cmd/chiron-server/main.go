// Command chiron-server is the adaptive engine behind the Chiron iPad app.
package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/mjbraun/chiron/server/httpapi"
)

func main() {
	// Binding to all interfaces is the default on purpose. Binding to loopback
	// makes the iPad's request time out, which presents as the app hanging on
	// "Thinking about what you need next" - a failure that cost real debugging
	// time once and looked like an app bug.
	addr := flag.String("addr", ":8080", "listen address")
	configPath := flag.String("config", "config.yaml", "path to config.yaml")
	flag.Parse()

	abs, err := filepath.Abs(*configPath)
	if err != nil {
		log.Fatalf("config path: %v", err)
	}
	cfg, err := httpapi.LoadConfig(abs)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	srv, err := httpapi.New(cfg, filepath.Dir(abs))
	if err != nil {
		log.Fatalf("start: %v", err)
	}

	server := &http.Server{
		Addr:    *addr,
		Handler: srv.Handler(),
		// Chapter generation on a local model legitimately takes minutes, and
		// authoring a whole subject takes far longer, so there is no write
		// timeout; the model backends carry their own.
		ReadHeaderTimeout: 15 * time.Second,
	}

	go func() {
		log.Printf("chiron-server listening on %s", *addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	// Let an in-flight exchange finish: the learner is waiting on a chapter,
	// and their graded answers are already written to the event log.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}
