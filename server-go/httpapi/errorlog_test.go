package httpapi

import (
	"bytes"
	"log"
	"net/http/httptest"
	"strings"
	"testing"
)

// Every error response must also land in the server log. A failed
// check-in whose client has already given up (or whose connection the
// edge proxy cut) is otherwise completely invisible: the only copy of
// the error goes down a dead socket.
func TestWriteErrorLogsServerSide(t *testing.T) {
	var buf bytes.Buffer
	prev := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(prev)

	w := httptest.NewRecorder()
	writeError(w, 502, "transcription failed on %s: %v", "q1", "boom")

	if w.Code != 502 {
		t.Fatalf("status = %d, want 502", w.Code)
	}
	if got := buf.String(); !strings.Contains(got, "transcription failed on q1: boom") ||
		!strings.Contains(got, "502") {
		t.Fatalf("log output %q missing status or detail", got)
	}
}
