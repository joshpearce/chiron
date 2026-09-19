package httpapi

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A build made on the MacBook lands under builds/<token>/ with a
// manifest an iPad installs from. iOS fetches the manifest and the IPA
// with its own downloader, which cannot send the bearer key, so those
// two files are open at an unguessable path; everything about which
// build is current stays behind the key.
func TestABuildIsOfferedAtItsTokenAndDescribedBehindTheKey(t *testing.T) {
	builds := t.TempDir()
	token := strings.Repeat("ab", 32)
	dir := filepath.Join(builds, token)
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, "build.json"), []byte(`{"id":"20260919-1","token":"`+token+`","commit":"c97416a","version":"2026.9.19","build":295,"status":"ready","tests":{"passed":100,"failed":0}}`), 0o644)
	os.WriteFile(filepath.Join(dir, "manifest.plist"), []byte("<plist/>"), 0o644)
	os.WriteFile(filepath.Join(dir, "Chiron.ipa"), []byte("PK ipa"), 0o644)
	os.WriteFile(filepath.Join(dir, "Chiron-mac.zip"), []byte("PK mac"), 0o644)
	os.WriteFile(filepath.Join(dir, "build.log"), []byte("secret-ish"), 0o644)
	os.Symlink(token, filepath.Join(builds, "latest"))

	s := newServer(t, "sekrit")
	s.buildsDir = builds

	w := do(t, s, "GET", "/builds/latest", "", "sekrit")
	if w.Code != 200 {
		t.Fatalf("latest: %d %s", w.Code, w.Body)
	}
	var latest struct {
		Version  string `json:"version"`
		Build    int    `json:"build"`
		Commit   string `json:"commit"`
		Manifest string `json:"manifest_path"`
		Mac      string `json:"mac_path"`
		Tests    struct{ Passed int }
	}
	json.Unmarshal(w.Body.Bytes(), &latest)
	if latest.Version != "2026.9.19" || latest.Build != 295 || latest.Commit != "c97416a" || latest.Tests.Passed != 100 {
		t.Errorf("latest = %s", w.Body)
	}
	if latest.Manifest != "/builds/"+token+"/manifest.plist" || latest.Mac != "/builds/"+token+"/Chiron-mac.zip" {
		t.Errorf("paths = %q %q", latest.Manifest, latest.Mac)
	}
	if w := do(t, s, "GET", "/builds/latest", "", ""); w.Code != 401 {
		t.Errorf("latest without the key: %d", w.Code)
	}

	// The installer's two files, open, typed for what they are.
	for _, tc := range []struct{ file, body, ctype string }{
		{"manifest.plist", "<plist/>", "text/xml"},
		{"Chiron.ipa", "PK ipa", "application/octet-stream"},
		{"Chiron-mac.zip", "PK mac", "application/zip"},
	} {
		w := do(t, s, "GET", "/builds/"+token+"/"+tc.file, "", "")
		if w.Code != 200 || w.Body.String() != tc.body || !strings.HasPrefix(w.Header().Get("Content-Type"), tc.ctype) {
			t.Errorf("%s: %d %q %s", tc.file, w.Code, w.Body, w.Header().Get("Content-Type"))
		}
	}
	// Nothing else at the token, and nothing at any other path.
	for _, path := range []string{
		"/builds/" + token + "/build.log",
		"/builds/" + token + "/build.json",
		"/builds/" + token + "/../latest/Chiron.ipa",
		"/builds/latest/Chiron.ipa",
		"/builds/" + strings.Repeat("cd", 32) + "/Chiron.ipa",
		"/builds/" + token[:20] + "/Chiron.ipa",
		"/builds/",
	} {
		if w := do(t, s, "GET", path, "", ""); w.Code == 200 {
			t.Errorf("%s served: %s", path, w.Body)
		}
	}
}

func TestNoBuildYetIsSaidPlainly(t *testing.T) {
	s := newServer(t, "sekrit")
	s.buildsDir = t.TempDir()
	if w := do(t, s, "GET", "/builds/latest", "", "sekrit"); w.Code != 404 {
		t.Errorf("no build: %d %s", w.Code, w.Body)
	}
}
