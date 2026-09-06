package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// A PDF on the shelf: uploaded once, listed as a row of its own kind,
// read back as the same bytes, with the reading position kept on the
// server so every device opens it where it was left.

func upload(t *testing.T, s *Server, title string, body string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest("POST", "/documents?title="+strings.ReplaceAll(title, " ", "%20")+"&pages=3", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/pdf")
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, r)
	return w
}

func TestAPDFIsUploadedListedReadAndForgotten(t *testing.T) {
	s := newServer(t, "")
	pdf := "%PDF-1.4\n1 0 obj << /Type /Catalog >> endobj\n%%EOF\n"
	w := upload(t, s, "A paper", pdf)
	if w.Code != http.StatusOK {
		t.Fatalf("upload: %d %s", w.Code, w.Body)
	}
	var doc Document
	if err := json.Unmarshal(w.Body.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(doc.ID, "doc-") || doc.Title != "A paper" || doc.Pages != 3 || doc.Size != int64(len(pdf)) || doc.ImportedAt == "" {
		t.Fatalf("document: %+v", doc)
	}

	w = do(t, s, "GET", "/subjects", "", "")
	var shelf struct {
		Subjects []struct {
			ID    string `json:"id"`
			Kind  string `json:"kind"`
			Title string `json:"title"`
			Pages int    `json:"pages"`
			Page  int    `json:"page"`
		} `json:"subjects"`
	}
	json.Unmarshal(w.Body.Bytes(), &shelf)
	found := false
	for _, r := range shelf.Subjects {
		if r.ID == doc.ID {
			found = true
			if r.Kind != "pdf" || r.Title != "A paper" || r.Pages != 3 || r.Page != 0 {
				t.Fatalf("row: %+v", r)
			}
		}
	}
	if !found {
		t.Fatalf("the document is not on the shelf: %s", w.Body)
	}

	w = do(t, s, "GET", "/documents/"+doc.ID+"/file", "", "")
	if w.Code != http.StatusOK || w.Body.String() != pdf || w.Header().Get("Content-Type") != "application/pdf" {
		t.Fatalf("file: %d %q %s", w.Code, w.Header().Get("Content-Type"), w.Body)
	}

	w = do(t, s, "PUT", "/documents/"+doc.ID+"/position", `{"page":2,"position":0.5}`, "")
	if w.Code != http.StatusOK {
		t.Fatalf("position: %d %s", w.Code, w.Body)
	}
	// The position survives a restart, and the shelf shows it.
	s = newServerAt(t, s)
	w = do(t, s, "GET", "/documents/"+doc.ID, "", "")
	json.Unmarshal(w.Body.Bytes(), &doc)
	if doc.Page != 2 || doc.Position != 0.5 {
		t.Fatalf("after restart: %+v", doc)
	}
	w = do(t, s, "GET", "/subjects", "", "")
	json.Unmarshal(w.Body.Bytes(), &shelf)
	for _, r := range shelf.Subjects {
		if r.ID == doc.ID && r.Page != 2 {
			t.Fatalf("row after restart: %+v", r)
		}
	}

	// A document goes on a shelf like anything else.
	w = do(t, s, "POST", "/shelves", `{"name":"Papers"}`, "")
	var made shelfView
	json.Unmarshal(w.Body.Bytes(), &made)
	if w := do(t, s, "PUT", "/subjects/"+doc.ID+"/shelf", `{"shelf":"`+made.ID+`"}`, ""); w.Code != http.StatusOK {
		t.Fatalf("move: %d %s", w.Code, w.Body)
	}
	where, _ := library(t, s)
	if where[doc.ID] != made.ID {
		t.Fatalf("the document is on shelf %q, want %q", where[doc.ID], made.ID)
	}

	if w := do(t, s, "DELETE", "/documents/"+doc.ID, "", ""); w.Code != http.StatusOK {
		t.Fatalf("delete: %d %s", w.Code, w.Body)
	}
	if w := do(t, s, "GET", "/documents/"+doc.ID, "", ""); w.Code != http.StatusNotFound {
		t.Fatalf("after delete: %d", w.Code)
	}
	where, _ = library(t, s)
	if _, still := where[doc.ID]; still {
		t.Fatal("a deleted document is still on the shelf")
	}
}

func TestOnlyAPDFIsAccepted(t *testing.T) {
	s := newServer(t, "")
	if w := upload(t, s, "Not a PDF", "hello"); w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("got %d", w.Code)
	}
	if w := upload(t, s, "", "%PDF-1.4\n"); w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("no title: got %d", w.Code)
	}
}
