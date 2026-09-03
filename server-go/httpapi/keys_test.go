package httpapi

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	macKeyLine = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGjKDfDx9bUcUu4Bl4SQhU7zCOZqpUhFWwOg3lpF5uJf matt@mac"
	ipadKey    = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIN3qA5o3Z8lHwlZ0m+XW2bbz4ZFf6wTqjV1p8QqYxuZk"
	rsaKey     = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC7Q6p8gZ6bR9qk7QwT1x4OQ0d7T1mQ0qkq1XqvI8Pz0v4Yw2uGQ5c2VxNTLrPqjZk4V7Pv4tXm8s9eZ8mQxFq3Z5gK1v2Xn4nOa2f0f6bXQ2Ck9oPq0Yq6t8n3L0Yq5uJkQm7l6l2cR2Gx3w7v9hA4Vq2u1y0z8Zt9k1jF7v3cQ0b5nH2rJ8sP6mK1wD4eT3fL9uX7yV2zB0aC8gN5iO1pR4sU6vW9xZ3bD7fH2jK5lM8nP1qS4tV7wY0zA3cE6gI9kL2nQ5rT8uX1yB4dF7hJ0mO3pS6vY9zC2eG5iK8lN1oR4tW7xA0bD3fH6jM9nQ2sU5vX8zB1cE4gJ7kO0pR3tV6wZ9yA2dG5iL8mP1qT4uX7z"
)

func keyServer(t *testing.T) (*Server, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), ".ssh", "authorized_keys")
	t.Setenv("CHIRON_AUTHORIZED_KEYS", path)
	return newTwoSubjectServer(t, t.TempDir()), path
}

func TestEnrolKeyAppendsOnceAndKeepsMacKeys(t *testing.T) {
	s, path := keyServer(t)
	os.MkdirAll(filepath.Dir(path), 0o700)
	os.WriteFile(path, []byte(macKeyLine), 0o600) // no trailing newline, on purpose

	body := `{"pubkey":"` + ipadKey + `","name":"Matt's iPad"}`
	w := do(t, s, "POST", "/agent/pubkey", body, "")
	if w.Code != http.StatusOK {
		t.Fatalf("enrol -> %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Fingerprint string `json:"fingerprint"`
		Installed   bool   `json:"installed"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if !resp.Installed || !strings.HasPrefix(resp.Fingerprint, "SHA256:") {
		t.Fatalf("resp = %+v", resp)
	}

	w = do(t, s, "POST", "/agent/pubkey", body, "")
	json.Unmarshal(w.Body.Bytes(), &resp)
	if w.Code != http.StatusOK || resp.Installed {
		t.Fatalf("second enrol -> %d %s", w.Code, w.Body.String())
	}

	data, _ := os.ReadFile(path)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 || lines[0] != macKeyLine || !strings.HasSuffix(lines[1], " chiron:Matt's iPad") || !strings.HasPrefix(lines[1], ipadKey) {
		t.Fatalf("authorized_keys:\n%s", data)
	}
	if info, _ := os.Stat(path); info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %v", info.Mode())
	}

	w = do(t, s, "GET", "/agent/keys", "", "")
	var list struct {
		Keys []enrolledKey `json:"keys"`
	}
	json.Unmarshal(w.Body.Bytes(), &list)
	if len(list.Keys) != 2 || list.Keys[0].Name != "matt@mac" || list.Keys[1].Name != "Matt's iPad" || list.Keys[1].Fingerprint != resp.Fingerprint {
		t.Fatalf("list = %+v", list.Keys)
	}

	w = do(t, s, "POST", "/agent/keys/revoke", `{"fingerprint":"`+resp.Fingerprint+`"}`, "")
	if w.Code != http.StatusOK {
		t.Fatalf("revoke -> %d: %s", w.Code, w.Body.String())
	}
	data, _ = os.ReadFile(path)
	if strings.TrimSpace(string(data)) != macKeyLine {
		t.Fatalf("after revoke:\n%s", data)
	}
	if w = do(t, s, "POST", "/agent/keys/revoke", `{"fingerprint":"`+resp.Fingerprint+`"}`, ""); w.Code != http.StatusNotFound {
		t.Fatalf("revoking a gone key -> %d", w.Code)
	}
}

func TestEnrolKeyRejectsWhatItShould(t *testing.T) {
	s, path := keyServer(t)
	cases := map[string]string{
		"rsa":          `{"pubkey":"` + rsaKey + `","name":"iPad"}`,
		"garbage":      `{"pubkey":"hello there","name":"iPad"}`,
		"empty name":   `{"pubkey":"` + ipadKey + `","name":""}`,
		"newline name": `{"pubkey":"` + ipadKey + `","name":"iPad\nssh-ed25519 x"}`,
		"options":      `{"pubkey":"command=\"rm -rf /\" ` + ipadKey + `","name":"iPad"}`,
	}
	for name, body := range cases {
		w := do(t, s, "POST", "/agent/pubkey", body, "")
		if w.Code != http.StatusUnprocessableEntity {
			t.Errorf("%s -> %d: %s", name, w.Code, w.Body.String())
		}
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("a rejected key touched the file")
	}
}

func TestKeyRoutesAreOffWithoutAPath(t *testing.T) {
	s := newTwoSubjectServer(t, t.TempDir())
	for _, r := range [][2]string{{"POST", "/agent/pubkey"}, {"GET", "/agent/keys"}, {"POST", "/agent/keys/revoke"}} {
		if w := do(t, s, r[0], r[1], `{"pubkey":"`+ipadKey+`","name":"x","fingerprint":"y"}`, ""); w.Code != http.StatusNotFound {
			t.Errorf("%s %s -> %d", r[0], r[1], w.Code)
		}
	}
}
