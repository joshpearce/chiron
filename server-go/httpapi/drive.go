package httpapi

import (
	"io"
	"net/http"
	"os"
	"sync"
)

// A remote-drive command queue for development: a test harness POSTs
// semantic commands, the client polls them. Enabled only when the server is
// started with CHIRON_DRIVE=1 - dev instances, never a real learner's
// server.

var (
	driveMu      sync.Mutex
	drivePending []byte
	driveAck     []byte
)

func driveEnabled() bool { return os.Getenv("CHIRON_DRIVE") == "1" }

func (s *Server) handleDriveCmd(w http.ResponseWriter, r *http.Request) {
	if !driveEnabled() {
		http.NotFound(w, r)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	driveMu.Lock()
	drivePending = body
	driveMu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleDriveAck(w http.ResponseWriter, r *http.Request) {
	if !driveEnabled() {
		http.NotFound(w, r)
		return
	}
	if r.Method == http.MethodPost {
		body, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		driveMu.Lock()
		driveAck = body
		driveMu.Unlock()
		w.WriteHeader(http.StatusNoContent)
		return
	}
	driveMu.Lock()
	ack := driveAck
	driveMu.Unlock()
	w.Header().Set("Cache-Control", "no-store")
	if ack == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(ack)
}

func (s *Server) handleDriveNext(w http.ResponseWriter, r *http.Request) {
	if !driveEnabled() {
		http.NotFound(w, r)
		return
	}
	driveMu.Lock()
	cmd := drivePending
	drivePending = nil
	driveMu.Unlock()
	w.Header().Set("Cache-Control", "no-store")
	if cmd == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(cmd)
}
