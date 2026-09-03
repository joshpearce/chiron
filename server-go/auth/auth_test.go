package auth

import (
	"net/http/httptest"
	"testing"
)

func TestAuthorized(t *testing.T) {
	cases := []struct {
		name   string
		header string
		query  string
		want   bool
	}{
		{"bearer header", "Bearer k1", "", true},
		{"query token", "", "?token=k1", true},
		{"wrong header", "Bearer k2", "", false},
		{"wrong query", "", "?token=k2", false},
		{"nothing", "", "", false},
		{"header wins over query", "Bearer k2", "?token=k1", false},
		{"prefix only", "Bearer k", "", false},
	}
	for _, c := range cases {
		r := httptest.NewRequest("GET", "/x"+c.query, nil)
		if c.header != "" {
			r.Header.Set("Authorization", c.header)
		}
		if got := Authorized(r, "k1"); got != c.want {
			t.Errorf("%s: got %v want %v", c.name, got, c.want)
		}
	}
}
