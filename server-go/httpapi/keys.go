package httpapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"golang.org/x/crypto/ssh"
)

// Device keys for the sprite's sshd. The app generates an Ed25519 key when a
// server with a shared key is saved and enrols it here; possessing the
// shared key is what authorizes that. Enabled only where an authorized_keys
// path is configured (the sprite), never on a Mac dev server.

var keysMu sync.Mutex

// Anything else in a comment is a way to smuggle options or a second key
// onto the line.
var keyNameOK = regexp.MustCompile(`^[A-Za-z0-9 ._'-]{1,64}$`)

type enrolledKey struct {
	Fingerprint string `json:"fingerprint"`
	Type        string `json:"type"`
	Name        string `json:"name"`
}

func (s *Server) keysEnabled(w http.ResponseWriter) bool {
	if s.authorizedKeys == "" {
		writeError(w, http.StatusNotFound, "key enrolment is not enabled on this server")
		return false
	}
	return true
}

func (s *Server) handleEnrolKey(w http.ResponseWriter, r *http.Request) {
	if !s.keysEnabled(w) {
		return
	}
	var req struct {
		Pubkey string `json:"pubkey"`
		Name   string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "malformed request: %v", err)
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if !keyNameOK.MatchString(req.Name) {
		writeError(w, http.StatusUnprocessableEntity, "name must be 1-64 letters, digits, spaces or ._'-")
		return
	}
	pub, _, options, _, err := ssh.ParseAuthorizedKey([]byte(strings.TrimSpace(req.Pubkey)))
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, "not an OpenSSH public key: %v", err)
		return
	}
	if len(options) > 0 {
		writeError(w, http.StatusUnprocessableEntity, "a bare public key, without options")
		return
	}
	if pub.Type() != ssh.KeyAlgoED25519 {
		writeError(w, http.StatusUnprocessableEntity, "only ssh-ed25519 keys are enrolled, not %s", pub.Type())
		return
	}
	line := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(pub))) + " chiron:" + req.Name

	keysMu.Lock()
	defer keysMu.Unlock()
	existing, err := readKeys(s.authorizedKeys)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "read authorized_keys: %v", err)
		return
	}
	fp := ssh.FingerprintSHA256(pub)
	for _, k := range existing {
		if k.Fingerprint == fp {
			writeJSON(w, http.StatusOK, map[string]any{"fingerprint": fp, "name": k.Name, "installed": false})
			return
		}
	}
	if err := appendLine(s.authorizedKeys, line); err != nil {
		writeError(w, http.StatusInternalServerError, "write authorized_keys: %v", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"fingerprint": fp, "name": req.Name, "installed": true})
}

func (s *Server) handleListKeys(w http.ResponseWriter, r *http.Request) {
	if !s.keysEnabled(w) {
		return
	}
	keysMu.Lock()
	defer keysMu.Unlock()
	keys, err := readKeys(s.authorizedKeys)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "read authorized_keys: %v", err)
		return
	}
	if keys == nil {
		keys = []enrolledKey{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"keys": keys})
}

func (s *Server) handleRevokeKey(w http.ResponseWriter, r *http.Request) {
	if !s.keysEnabled(w) {
		return
	}
	var req struct {
		Fingerprint string `json:"fingerprint"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Fingerprint == "" {
		writeError(w, http.StatusBadRequest, "fingerprint is required")
		return
	}
	keysMu.Lock()
	defer keysMu.Unlock()
	data, err := os.ReadFile(s.authorizedKeys)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "read authorized_keys: %v", err)
		return
	}
	var kept []string
	removed := 0
	for _, line := range strings.Split(string(data), "\n") {
		if pub, _, _, _, err := ssh.ParseAuthorizedKey([]byte(line)); err == nil && ssh.FingerprintSHA256(pub) == req.Fingerprint {
			removed++
			continue
		}
		kept = append(kept, line)
	}
	if removed == 0 {
		writeError(w, http.StatusNotFound, "no key with fingerprint %s", req.Fingerprint)
		return
	}
	if err := writeAtomic(s.authorizedKeys, []byte(strings.Join(kept, "\n"))); err != nil {
		writeError(w, http.StatusInternalServerError, "write authorized_keys: %v", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"fingerprint": req.Fingerprint, "removed": removed})
}

// readKeys parses every key line; comments and unparsable lines are skipped
// rather than fatal, since the Mac's keys share the file.
func readKeys(path string) ([]enrolledKey, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []enrolledKey
	for _, line := range strings.Split(string(data), "\n") {
		pub, comment, _, _, err := ssh.ParseAuthorizedKey([]byte(line))
		if err != nil {
			continue
		}
		out = append(out, enrolledKey{
			Fingerprint: ssh.FingerprintSHA256(pub),
			Type:        pub.Type(),
			Name:        strings.TrimPrefix(comment, "chiron:"),
		})
	}
	return out, nil
}

func appendLine(path, line string) error {
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if len(data) > 0 && !bytes.HasSuffix(data, []byte("\n")) {
		data = append(data, '\n')
	}
	data = append(data, []byte(line+"\n")...)
	return writeAtomic(path, data)
}

// sshd reads the file on every login, so it must never see a half-written
// one; and StrictModes refuses anything looser than 0600.
func writeAtomic(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp := fmt.Sprintf("%s.tmp-%d", path, os.Getpid())
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
