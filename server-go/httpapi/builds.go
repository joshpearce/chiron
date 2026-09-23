package httpapi

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// App builds the MacBook made (SPRITE-DEV-PLAN.md phases E and F): each
// under builds/<token>/ with build.json, manifest.plist, Chiron.ipa,
// Chiron-mac.zip and build.log; `latest` links to the current one. An
// iPad installs from the manifest through iOS's own downloader, which
// cannot send the bearer key, so the manifest and the two packages are
// open at the token, 32 random bytes nobody lists; what is current, and
// everything else about a build, stays behind the key.

func buildsRoot(cfg *Config, root string) string {
	if cfg.BuildsDir != "" {
		return resolve(root, cfg.BuildsDir)
	}
	if cfg.DataDir != "" {
		return filepath.Join(resolve(root, cfg.DataDir), "builds")
	}
	return filepath.Join(filepath.Dir(root), "builds")
}

var buildToken = regexp.MustCompile(`^[0-9a-f]{64}$`)

// The files an installer fetches, and what they are served as.
var buildFiles = map[string]string{
	"manifest.plist": "text/xml; charset=utf-8",
	"Chiron.ipa":     "application/octet-stream",
	"Chiron-mac.zip": "application/zip",
}

// buildFileOpen says whether a path is one of the installer's files at a
// token: those, and only those, are served without the key.
func buildFileOpen(path string) bool {
	parts := strings.Split(strings.TrimPrefix(path, "/builds/"), "/")
	if !strings.HasPrefix(path, "/builds/") || len(parts) != 2 {
		return false
	}
	_, known := buildFiles[parts[1]]
	return buildToken.MatchString(parts[0]) && known
}

type buildInfo struct {
	ID      string `json:"id"`
	Token   string `json:"token"`
	Commit  string `json:"commit"`
	Version string `json:"version"`
	Build   int    `json:"build"`
	Status  string `json:"status"`
	Reason  string `json:"reason,omitempty"`
	Tests   struct {
		Passed int `json:"passed"`
		Failed int `json:"failed"`
	} `json:"tests"`
	Finished string `json:"finished"`
}

// GET /builds/latest: the current build and where its packages are.
func (s *Server) handleBuildLatest(w http.ResponseWriter, r *http.Request) {
	data, err := os.ReadFile(filepath.Join(s.buildsDir, "latest", "build.json"))
	if err != nil {
		writeError(w, http.StatusNotFound, "no build yet")
		return
	}
	var b buildInfo
	if err := json.Unmarshal(data, &b); err != nil || !buildToken.MatchString(b.Token) {
		writeError(w, http.StatusInternalServerError, "the latest build is not readable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id": b.ID, "commit": b.Commit, "version": b.Version, "build": b.Build,
		"status": b.Status, "reason": b.Reason, "tests": b.Tests, "finished": b.Finished,
		"manifest_path": "/builds/" + b.Token + "/manifest.plist",
		"mac_path":      "/builds/" + b.Token + "/Chiron-mac.zip",
	})
}

// GET /builds/{token}/{file}: one of the installer's files, open.
func (s *Server) handleBuildFile(w http.ResponseWriter, r *http.Request) {
	token, file := r.PathValue("token"), r.PathValue("file")
	ctype, known := buildFiles[file]
	if !buildToken.MatchString(token) || !known {
		writeError(w, http.StatusNotFound, "not a build file")
		return
	}
	f, err := os.Open(filepath.Join(s.buildsDir, token, file))
	if err != nil {
		writeError(w, http.StatusNotFound, "no such build")
		return
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil || st.IsDir() {
		writeError(w, http.StatusNotFound, "no such build")
		return
	}
	w.Header().Set("Content-Type", ctype)
	w.Header().Set("Cache-Control", "private, max-age=0")
	http.ServeContent(w, r, file, st.ModTime(), f)
}
