package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/mjbraun/chiron/server/llm"
)

// The canned primer: one document, and one section for any extend.
func primerPayload() map[string]any {
	return map[string]any{
		"title":    "Robots and money",
		"markdown": "The web is closing.\n\n## Signals\n\nThree directives, plainly separated.\n\n## Money\n\nCheques get signed.",
		"heading":  "Why 402 and not 403",
	}
}

type captureReply struct {
	Subject string `json:"subject"`
	Status  string `json:"status"`
	Title   string `json:"title"`
}

type shelfRow struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Kind       string `json:"kind"`
	Status     string `json:"status"`
	Error      string `json:"error"`
	UnitsTotal int    `json:"units_total"`
	CapturedAt string `json:"captured_at"`
	Source     *struct {
		Text string `json:"text"`
		URL  string `json:"url"`
		App  string `json:"app"`
	} `json:"source"`
}

func shelf(t *testing.T, s *Server) map[string]shelfRow {
	t.Helper()
	w := do(t, s, "GET", "/subjects", "", "")
	var resp struct {
		Subjects []shelfRow `json:"subjects"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	out := map[string]shelfRow{}
	for _, r := range resp.Subjects {
		out[r.ID] = r
	}
	return out
}

func capture(t *testing.T, s *Server, body string) captureReply {
	t.Helper()
	w := do(t, s, "POST", "/primer/capture", body, "")
	if w.Code != http.StatusOK {
		t.Fatalf("capture -> %d: %s", w.Code, w.Body.String())
	}
	var rep captureReply
	json.Unmarshal(w.Body.Bytes(), &rep)
	return rep
}

func TestACaptureBecomesAPrimerOnTheShelf(t *testing.T) {
	s := newServer(t, "")
	chain := &cannedChain{payload: primerPayload()}
	s.chain = chain

	rep := capture(t, s, `{"text":"User-agent: *\nContent-Signal: search=yes, ai-input=no","source_url":"https://lexweekly.example/robots.txt","source_app":"Safari","prompt":"what is Content-Signal?"}`)
	if !strings.HasPrefix(rep.Subject, "primer-") || rep.Status != "authoring" {
		t.Fatalf("reply = %+v", rep)
	}
	s.renders.Wait() // authoring runs in the background

	row, ok := shelf(t, s)[rep.Subject]
	if !ok || row.Kind != "primer" || row.Status != "ready" || row.Title != "Robots and money" || row.UnitsTotal != 1 {
		t.Fatalf("shelf row = %+v", row)
	}
	if row.Source == nil || row.Source.App != "Safari" || !strings.Contains(row.Source.Text, "Content-Signal") || row.CapturedAt == "" {
		t.Fatalf("source = %+v", row.Source)
	}
	if !strings.Contains(chain.user, "what is Content-Signal?") || !strings.Contains(chain.user, "ai-input=no") {
		t.Errorf("author prompt lacks the question or the capture:\n%s", chain.user)
	}

	// Reading it: the document as written, no pretest, no check.
	w := do(t, s, "POST", "/exchange", `{"subject":"`+rep.Subject+`","phase":"start"}`, "")
	if w.Code != http.StatusOK {
		t.Fatalf("start -> %d: %s", w.Code, w.Body.String())
	}
	var ex struct {
		Chapter *struct {
			Unit    string            `json:"unit"`
			HTML    string            `json:"html"`
			Check   []json.RawMessage `json:"check"`
			Pretest []json.RawMessage `json:"pretest"`
		} `json:"chapter"`
	}
	json.Unmarshal(w.Body.Bytes(), &ex)
	if ex.Chapter == nil || ex.Chapter.Unit != "p1" || len(ex.Chapter.Check) != 0 || len(ex.Chapter.Pretest) != 0 {
		t.Fatalf("chapter = %+v", ex.Chapter)
	}
	if !strings.Contains(ex.Chapter.HTML, "Three directives") || !strings.Contains(ex.Chapter.HTML, "Cheques get signed") {
		t.Fatalf("html lacks the document: %s", ex.Chapter.HTML)
	}

	// A margin note extends it, and the chapter grows in place.
	w = do(t, s, "POST", "/primer/"+rep.Subject+"/extend", `{"quote":"Cheques get signed.","note":"why 402 and not 403?"}`, "")
	if w.Code != http.StatusOK {
		t.Fatalf("extend -> %d: %s", w.Code, w.Body.String())
	}
	var ext struct {
		Chapter struct {
			HTML string `json:"html"`
		} `json:"chapter"`
		Heading string `json:"heading"`
		Entries int    `json:"entries"`
	}
	json.Unmarshal(w.Body.Bytes(), &ext)
	if ext.Heading != "Why 402 and not 403" || ext.Entries != 1 || !strings.Contains(ext.Chapter.HTML, "Why 402 and not 403") || !strings.Contains(ext.Chapter.HTML, "Three directives") {
		t.Fatalf("extend = %+v", ext)
	}
	for _, want := range []string{"why 402 and not 403?", "Cheques get signed.", "## Signals"} {
		if !strings.Contains(chain.user, want) {
			t.Errorf("extend prompt lacks %q", want)
		}
	}
	w = do(t, s, "GET", "/chapter/"+rep.Subject, "", "")
	if !strings.Contains(w.Body.String(), "Why 402 and not 403") {
		t.Fatalf("GET /chapter after extend: %s", w.Body.String())
	}

	// A restart finds it on disk, ready, with the extension.
	s2, err := New(s.cfg, s.root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s2.renders.Wait)
	row2, ok := shelf(t, s2)[rep.Subject]
	if !ok || row2.Status != "ready" || row2.Kind != "primer" {
		t.Fatalf("after restart: %+v", row2)
	}
	w = do(t, s2, "GET", "/chapter/"+rep.Subject, "", "")
	if !strings.Contains(w.Body.String(), "Why 402 and not 403") {
		t.Fatalf("after restart, chapter: %s", w.Body.String())
	}
}

func TestCaptureRefusesWhatItShould(t *testing.T) {
	s := newServer(t, "")
	cases := map[string]int{
		`{"text":"something","prompt":""}`:             http.StatusUnprocessableEntity,
		`{"text":"","prompt":"why?"}`:                  http.StatusUnprocessableEntity,
		`{"image_png_b64":"not base64!","prompt":"?"}`: http.StatusBadRequest,
	}
	for body, want := range cases {
		if w := do(t, s, "POST", "/primer/capture", body, ""); w.Code != want {
			t.Errorf("%s -> %d, want %d", body, w.Code, want)
		}
	}
	if w := do(t, s, "POST", "/primer/ai/extend", `{"quote":"x","note":"y"}`, ""); w.Code != http.StatusNotFound {
		t.Errorf("extending a book -> %d", w.Code)
	}
}

func TestAnImageCaptureIsTranscribedFirst(t *testing.T) {
	s := newServer(t, "")
	chain := &cannedChain{payload: primerPayload()}
	s.chain = chain
	s.transcribe = func(hint string, png []byte) (string, error) {
		return "TRANSCRIBED: Content-Signal: search=yes", nil
	}
	rep := capture(t, s, `{"image_png_b64":"iVBORw0KGgo=","prompt":"what does this header mean?"}`)
	s.renders.Wait()
	if !strings.Contains(chain.user, "TRANSCRIBED: Content-Signal") {
		t.Fatalf("author prompt lacks the transcription:\n%s", chain.user)
	}
	row := shelf(t, s)[rep.Subject]
	if row.Status != "ready" || row.Source == nil || !strings.Contains(row.Source.Text, "TRANSCRIBED") {
		t.Fatalf("row = %+v", row)
	}
}

// A chain that answers when told to, so the shelf can be read mid-authoring.
type gatedChain struct {
	release chan struct{}
	payload map[string]any
}

func (c *gatedChain) Structured(role, system, user string, schema map[string]any, name string, out any) error {
	<-c.release
	data, _ := json.Marshal(c.payload)
	return json.Unmarshal(data, out)
}
func (gatedChain) Status() llm.Status { return llm.Status{Connected: true} }

func TestAPrimerStillAuthoringIsOnTheShelfAsSuch(t *testing.T) {
	s := newServer(t, "")
	chain := &gatedChain{release: make(chan struct{}), payload: primerPayload()}
	s.chain = chain
	rep := capture(t, s, `{"text":"captured","prompt":"tell me"}`)
	row, ok := shelf(t, s)[rep.Subject]
	if !ok || row.Status != "authoring" || row.Kind != "primer" || row.UnitsTotal != 0 {
		t.Fatalf("mid-authoring row = %+v", row)
	}
	if w := do(t, s, "POST", "/exchange", `{"subject":"`+rep.Subject+`","phase":"start"}`, ""); w.Code != http.StatusNotFound {
		t.Fatalf("opening an unfinished primer -> %d", w.Code)
	}
	close(chain.release)
	s.renders.Wait()
	if row := shelf(t, s)[rep.Subject]; row.Status != "ready" {
		t.Fatalf("after release: %+v", row)
	}
}

type failingChain struct{}

func (failingChain) Structured(role, system, user string, schema map[string]any, name string, out any) error {
	return errors.New("no model")
}
func (failingChain) Status() llm.Status { return llm.Status{} }

func TestADevServerWithoutAModelWritesAStubPrimer(t *testing.T) {
	t.Setenv("CHIRON_DRIVE", "1")
	s := newServer(t, "")
	s.chain = failingChain{}
	rep := capture(t, s, `{"text":"the captured words","prompt":"what is this?"}`)
	s.renders.Wait()
	row := shelf(t, s)[rep.Subject]
	if row.Status != "ready" {
		t.Fatalf("row = %+v", row)
	}
	w := do(t, s, "POST", "/exchange", `{"subject":"`+rep.Subject+`","phase":"start"}`, "")
	if !strings.Contains(w.Body.String(), "stub primer") || !strings.Contains(w.Body.String(), "the captured words") {
		t.Fatalf("stub chapter: %s", w.Body.String()[:300])
	}
	w = do(t, s, "POST", "/primer/"+rep.Subject+"/extend", `{"quote":"the captured words","note":"more please"}`, "")
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "[stub]") {
		t.Fatalf("stub extend -> %d: %s", w.Code, w.Body.String()[:200])
	}
}

func TestAFailedPrimerSaysWhy(t *testing.T) {
	s := newServer(t, "")
	s.chain = failingChain{}
	rep := capture(t, s, `{"text":"words","prompt":"why?"}`)
	s.renders.Wait()
	row := shelf(t, s)[rep.Subject]
	if row.Status != "failed" || !strings.Contains(row.Error, "no model") {
		t.Fatalf("row = %+v", row)
	}
}

func TestPrimerNamesReadAsPhrases(t *testing.T) {
	s := newServer(t, "")
	if got := s.uniquePrimerID("What does Content-Signal mean for a crawler?"); got != "primer-what-does-content-signal-mean" {
		t.Errorf("id = %q", got)
	}
	if got := workingTitle("what does Content-Signal mean for a crawler?"); got != "What does Content-Signal mean for a crawler?" {
		t.Errorf("title = %q", got)
	}
	long := strings.Repeat("word ", 30)
	if got := workingTitle(long); len(got) > 72 || strings.HasSuffix(got, " ") {
		t.Errorf("long title = %q", got)
	}
}
