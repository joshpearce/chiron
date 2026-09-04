package httpapi

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/mjbraun/chiron/server/primer"
)

// The capture card offers four scales. A summary or a description is
// answered in the card and leaves nothing on the shelf; a primer or a
// smart book is planned first, in a conversation the reader can leave
// and come back to, and built when they say so.

type scaledReply struct {
	Subject  string `json:"subject"`
	Status   string `json:"status"`
	Title    string `json:"title"`
	Scale    string `json:"scale"`
	AnswerMD string `json:"answer_md"`
	ReplyMD  string `json:"reply_md"`
	Done     bool   `json:"done"`
	Book     string `json:"book"`
}

func captureScaled(t *testing.T, s *Server, body string) scaledReply {
	t.Helper()
	w := do(t, s, "POST", "/primer/capture", body, "")
	if w.Code != http.StatusOK {
		t.Fatalf("capture -> %d: %s", w.Code, w.Body.String())
	}
	var rep scaledReply
	json.Unmarshal(w.Body.Bytes(), &rep)
	return rep
}

type planState struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Scale  string `json:"scale"`
	Status string `json:"status"`
	Prompt string `json:"prompt"`
	Brief  string `json:"brief"`
	Done   bool   `json:"done"`
	Plan   []struct {
		Role string `json:"role"`
		Text string `json:"text"`
	} `json:"plan"`
	Source struct {
		Text string `json:"text"`
	} `json:"source"`
}

func planOf(t *testing.T, s *Server, id string) planState {
	t.Helper()
	w := do(t, s, "GET", "/primer/"+id+"/plan", "", "")
	if w.Code != http.StatusOK {
		t.Fatalf("plan -> %d: %s", w.Code, w.Body.String())
	}
	var p planState
	json.Unmarshal(w.Body.Bytes(), &p)
	return p
}

func TestASummaryIsAnsweredInTheCard(t *testing.T) {
	s := newServer(t, "")
	chain := &cannedChain{payload: map[string]any{"markdown": "Content-Signal is a robots.txt line that says what a crawler may do with the page."}}
	s.chain = chain
	before := len(shelf(t, s))
	rep := captureScaled(t, s, `{"text":"Content-Signal: search=yes, ai-input=no","prompt":"what is this?","scale":"summary"}`)
	if rep.AnswerMD == "" || rep.Subject != "" || rep.Scale != "summary" {
		t.Fatalf("reply = %+v", rep)
	}
	if !strings.Contains(chain.user, "what is this?") || !strings.Contains(chain.user, "ai-input=no") {
		t.Errorf("prompt lacks the question or the capture:\n%s", chain.user)
	}
	if got := len(shelf(t, s)); got != before {
		t.Fatalf("a summary landed on the shelf: %d rows, was %d", got, before)
	}
	rep = captureScaled(t, s, `{"text":"Content-Signal: search=yes","prompt":"what is this?","scale":"description"}`)
	if rep.AnswerMD == "" || rep.Scale != "description" {
		t.Fatalf("description reply = %+v", rep)
	}
}

func TestAPrimerCaptureIsPlannedThenBuilt(t *testing.T) {
	s := newServer(t, "")
	chain := &cannedChain{payload: map[string]any{"reply_md": "Which directive matters most to you?", "done": false, "brief": "", "title": "", "slug": ""}}
	s.chain = chain

	rep := captureScaled(t, s, `{"text":"Content-Signal: search=yes, ai-input=no","prompt":"what is Content-Signal?","scale":"primer","source_app":"Safari"}`)
	if rep.Status != "planning" || rep.ReplyMD != "Which directive matters most to you?" || rep.Done || !strings.HasPrefix(rep.Subject, "primer-") {
		t.Fatalf("reply = %+v", rep)
	}
	if !strings.Contains(chain.user, "what is Content-Signal?") || !strings.Contains(chain.user, "ai-input=no") {
		t.Errorf("plan prompt lacks the question or the capture:\n%s", chain.user)
	}
	row := shelf(t, s)[rep.Subject]
	if row.Kind != "primer" || row.Status != "planning" || row.Scale != "primer" {
		t.Fatalf("shelf row = %+v", row)
	}
	p := planOf(t, s, rep.Subject)
	if len(p.Plan) != 1 || p.Plan[0].Role != "tutor" || p.Done || p.Prompt != "what is Content-Signal?" {
		t.Fatalf("plan = %+v", p)
	}

	// The reader answers; the tutor has enough and writes the brief.
	chain.payload = map[string]any{"reply_md": "A primer on the ai-input directive, then.", "done": true,
		"brief": "I want to know what ai-input=no does to crawlers.", "title": "Content-Signal for AI crawlers", "slug": "content-signal"}
	w := do(t, s, "POST", "/primer/"+rep.Subject+"/plan", `{"text":"the ai-input part"}`, "")
	if w.Code != http.StatusOK {
		t.Fatalf("plan turn -> %d: %s", w.Code, w.Body.String())
	}
	var turn scaledReply
	json.Unmarshal(w.Body.Bytes(), &turn)
	if !turn.Done || turn.ReplyMD == "" || turn.Title != "Content-Signal for AI crawlers" {
		t.Fatalf("turn = %+v", turn)
	}
	if !strings.Contains(chain.user, "the ai-input part") || !strings.Contains(chain.user, "Which directive") {
		t.Errorf("second turn lacks the conversation:\n%s", chain.user)
	}

	// A restart keeps the draft and its conversation.
	s2, err := New(s.cfg, s.root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s2.renders.Wait)
	s2.chain = chain
	p = planOf(t, s2, rep.Subject)
	if len(p.Plan) != 3 || !p.Done || p.Brief == "" || p.Title != "Content-Signal for AI crawlers" || p.Status != "planning" {
		t.Fatalf("after restart, plan = %+v", p)
	}
	if row := shelf(t, s2)[rep.Subject]; row.Status != "planning" || row.Title != "Content-Signal for AI crawlers" {
		t.Fatalf("after restart, row = %+v", row)
	}

	// Build: the primer is authored from the capture and the brief.
	chain.payload = primerPayload()
	w = do(t, s2, "POST", "/primer/"+rep.Subject+"/build", "", "")
	if w.Code != http.StatusOK {
		t.Fatalf("build -> %d: %s", w.Code, w.Body.String())
	}
	var built scaledReply
	json.Unmarshal(w.Body.Bytes(), &built)
	if built.Status != "authoring" || built.Subject != rep.Subject {
		t.Fatalf("build = %+v", built)
	}
	s2.renders.Wait()
	if !strings.Contains(chain.user, "ai-input=no does to crawlers") {
		t.Errorf("author prompt lacks the brief:\n%s", chain.user)
	}
	row = shelf(t, s2)[rep.Subject]
	if row.Status != "ready" || row.Title != "Robots and money" || row.UnitsTotal != 1 {
		t.Fatalf("built row = %+v", row)
	}
	// Building again is refused: it is a primer now.
	if w := do(t, s2, "POST", "/primer/"+rep.Subject+"/build", "", ""); w.Code != http.StatusConflict {
		t.Fatalf("second build -> %d", w.Code)
	}
}

func TestABuildBeforeThePlanIsDoneUsesWhatWasSaid(t *testing.T) {
	s := newServer(t, "")
	chain := &cannedChain{payload: map[string]any{"reply_md": "What do you already know?", "done": false, "brief": "", "title": "", "slug": ""}}
	s.chain = chain
	rep := captureScaled(t, s, `{"text":"the words","prompt":"why?","scale":"primer"}`)
	do(t, s, "POST", "/primer/"+rep.Subject+"/plan", `{"text":"nothing at all, start from zero"}`, "")
	chain.payload = primerPayload()
	if w := do(t, s, "POST", "/primer/"+rep.Subject+"/build", "", ""); w.Code != http.StatusOK {
		t.Fatalf("build -> %d: %s", w.Code, w.Body.String())
	}
	s.renders.Wait()
	if !strings.Contains(chain.user, "start from zero") {
		t.Errorf("author prompt lacks what the reader said:\n%s", chain.user)
	}
	if row := shelf(t, s)[rep.Subject]; row.Status != "ready" {
		t.Fatalf("row = %+v", row)
	}
}

func TestABookCaptureBecomesAGenerationJob(t *testing.T) {
	s := newServer(t, "")
	chain := &cannedChain{payload: map[string]any{"reply_md": "That is enough to plan from.", "done": true,
		"brief": "I want a short book on crawler directives.", "title": "Crawler Directives", "slug": "crawler-directives"}}
	s.chain = chain
	var started []string
	s.startGenerate = func(slug, title, brief string) { started = append(started, slug+"|"+title+"|"+brief) }

	rep := captureScaled(t, s, `{"text":"Content-Signal: search=yes","prompt":"teach me crawler directives","scale":"book","source_app":"Safari"}`)
	if rep.Status != "planning" || !rep.Done || rep.Scale != "book" {
		t.Fatalf("reply = %+v", rep)
	}
	w := do(t, s, "POST", "/primer/"+rep.Subject+"/build", "", "")
	if w.Code != http.StatusOK {
		t.Fatalf("build -> %d: %s", w.Code, w.Body.String())
	}
	var built scaledReply
	json.Unmarshal(w.Body.Bytes(), &built)
	if built.Status != "building" || built.Book != "crawler-directives" {
		t.Fatalf("build = %+v", built)
	}
	if len(started) != 1 || !strings.HasPrefix(started[0], "crawler-directives|Crawler Directives|") ||
		!strings.Contains(started[0], "short book on crawler directives") || !strings.Contains(started[0], "Content-Signal: search=yes") {
		t.Fatalf("generation started with %v", started)
	}
	row := shelf(t, s)[rep.Subject]
	if row.Status != "building" || row.Scale != "book" || row.Book != "crawler-directives" {
		t.Fatalf("row = %+v", row)
	}
	if w := do(t, s, "GET", "/teach/jobs?slug=crawler-directives", "", ""); w.Code != http.StatusOK {
		t.Fatalf("no job: %d", w.Code)
	}

	// The job fails: the draft says so, and can be built again.
	s.updateJob("crawler-directives", func(j *Job) { j.Stage, j.Done, j.Error = "failed", true, "the planner gave up" })
	row = shelf(t, s)[rep.Subject]
	if row.Status != "failed" || !strings.Contains(row.Error, "planner gave up") {
		t.Fatalf("after failure, row = %+v", row)
	}
	if w := do(t, s, "POST", "/primer/"+rep.Subject+"/build", "", ""); w.Code != http.StatusOK || len(started) != 2 {
		t.Fatalf("rebuild -> %d, started %v", w.Code, started)
	}

	// The book arrives: the draft has done its work and leaves the shelf.
	s.updateJob("crawler-directives", func(j *Job) { j.Stage, j.Done = "ready", true })
	s.primersMu.Lock()
	s.primers[rep.Subject].Book = "ai" // stands in for the registered book
	s.primersMu.Unlock()
	if _, still := shelf(t, s)[rep.Subject]; still {
		t.Fatalf("the draft stayed on the shelf after its book arrived")
	}
	if _, err := os.Stat(primer.Dir(s.primersRoot(), rep.Subject)); !os.IsNotExist(err) {
		t.Fatalf("draft dir still there: %v", err)
	}
}

func TestADraftCanBeDiscarded(t *testing.T) {
	s := newServer(t, "")
	s.chain = &cannedChain{payload: map[string]any{"reply_md": "Which part?", "done": false, "brief": "", "title": "", "slug": ""}}
	rep := captureScaled(t, s, `{"text":"the words","prompt":"why?","scale":"primer"}`)
	if w := do(t, s, "POST", "/primer/"+rep.Subject+"/discard", "", ""); w.Code != http.StatusOK {
		t.Fatalf("discard -> %d: %s", w.Code, w.Body.String())
	}
	if _, still := shelf(t, s)[rep.Subject]; still {
		t.Fatal("the draft is still on the shelf")
	}
	if w := do(t, s, "GET", "/primer/"+rep.Subject+"/plan", "", ""); w.Code != http.StatusNotFound {
		t.Fatalf("plan of a discarded draft -> %d", w.Code)
	}
	// A primer that was built is not a draft to discard.
	s.chain = &cannedChain{payload: primerPayload()}
	rep = captureScaled(t, s, `{"text":"the words","prompt":"why?"}`)
	if w := do(t, s, "POST", "/primer/"+rep.Subject+"/build", "", ""); w.Code != http.StatusOK {
		t.Fatalf("build -> %d", w.Code)
	}
	s.renders.Wait()
	if w := do(t, s, "POST", "/primer/"+rep.Subject+"/discard", "", ""); w.Code != http.StatusConflict {
		t.Fatalf("discard of a ready primer -> %d", w.Code)
	}
}

func TestADevServerWithoutAModelStubsTheScales(t *testing.T) {
	t.Setenv("CHIRON_DRIVE", "1")
	s := newServer(t, "")
	s.chain = failingChain{}
	rep := captureScaled(t, s, `{"text":"the captured words","prompt":"what is this?","scale":"summary"}`)
	if !strings.Contains(rep.AnswerMD, "stub") || !strings.Contains(rep.AnswerMD, "the captured words") {
		t.Fatalf("stub summary = %+v", rep)
	}
	rep = captureScaled(t, s, `{"text":"the captured words","prompt":"what is this?","scale":"primer"}`)
	if rep.Status != "planning" || !strings.Contains(rep.ReplyMD, "?") {
		t.Fatalf("stub plan = %+v", rep)
	}
	w := do(t, s, "POST", "/primer/"+rep.Subject+"/plan", `{"text":"the basics"}`, "")
	var turn scaledReply
	json.Unmarshal(w.Body.Bytes(), &turn)
	if w.Code != http.StatusOK || !turn.Done {
		t.Fatalf("stub second turn -> %d %+v", w.Code, turn)
	}
	if w := do(t, s, "POST", "/primer/"+rep.Subject+"/build", "", ""); w.Code != http.StatusOK {
		t.Fatalf("stub build -> %d: %s", w.Code, w.Body.String())
	}
	s.renders.Wait()
	if row := shelf(t, s)[rep.Subject]; row.Status != "ready" {
		t.Fatalf("stub built row = %+v", row)
	}
}
