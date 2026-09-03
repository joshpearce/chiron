// Package auth is the shared-key check every Chiron front door uses: the
// book server, and the gate that sits in front of it on the sprite.
package auth

import (
	"crypto/subtle"
	"net/http"
)

// Authorized reports whether r carries the shared key, either as
// "Authorization: Bearer <key>" or as a ?token= query parameter. The query
// form exists for clients that cannot set headers (image elements, some
// WebSocket clients). The compare is constant-time: a length- or
// prefix-leaking check on a shared secret is a bad habit even on a small
// deployment.
func Authorized(r *http.Request, key string) bool {
	supplied := r.Header.Get("Authorization")
	if supplied == "" {
		supplied = "Bearer " + r.URL.Query().Get("token")
	}
	expected := "Bearer " + key
	return subtle.ConstantTimeCompare([]byte(supplied), []byte(expected)) == 1
}
