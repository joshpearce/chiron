package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

// The API contract the iPad app depends on. Everything here runs against the
// real corpus with an isolated state directory, and never reaches a model - the
// planner cold-starts without a call, which is what makes the first chapter
// instant.
func newServer(t *testing.T, token string) *Server {
	t.Helper()
	root := filepath.Join("..", "..", "server")
	abs, err := filepath.Abs(root)
	if err != nil {
		t.Fatal(err)
	}
	cfg := &Config{
		Subjects: []SubjectSpec{{
			ID: "ai", Title: "How AI Works",
			CorpusDir: filepath.Join("..", "corpus"),
			StateDir:  t.TempDir(),
		}},
		AuthToken: token,
		Session: SessionConfig{
			MasteryGate: 0.8, ExtensionTrigger: 0.9, CheckItems: 9,
			CallbackFraction: 0.4, ChunkMinutes: 22, BreakMinutes: 5,
			LongBreakEveryChunks: 4,
		},
	}
	s, err := New(cfg, abs)
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	return s
}

func do(t *testing.T, s *Server, method, path, body, token string) *httptest.ResponseRecorder {
	t.Helper()
	var r *http.Request
	if body == "" {
		r = httptest.NewRequest(method, path, nil)
	} else {
		r = httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, r)
	return w
}

// On a public URL the token is the only thing between the endpoint and someone
// else's model budget. /ping stays open as a liveness probe that reveals
// nothing.
func TestAuthGuardsEverythingButPing(t *testing.T) {
	s := newServer(t, "sekrit")
	for _, tc := range []struct {
		path, token string
		want        int
	}{
		{"/ping", "", http.StatusOK},
		{"/health", "", http.StatusUnauthorized},
		{"/health", "wrong", http.StatusUnauthorized},
		{"/health", "sekrit", http.StatusOK},
		{"/subjects", "", http.StatusUnauthorized},
		{"/state", "", http.StatusUnauthorized},
		{"/review-schedule", "", http.StatusUnauthorized},
	} {
		if got := do(t, s, "GET", tc.path, "", tc.token).Code; got != tc.want {
			t.Errorf("GET %s with token %q -> %d, want %d", tc.path, tc.token, got, tc.want)
		}
	}
	if got := do(t, s, "POST", "/exchange", `{"subject":"ai"}`, "").Code; got != http.StatusUnauthorized {
		t.Errorf("unauthenticated /exchange -> %d - this is the expensive one", got)
	}
}

// An empty token is the flight LAN case: the only client is the iPad on a
// Mac-hosted network and there is nobody else to keep out.
func TestNoTokenLeavesTheServerOpen(t *testing.T) {
	s := newServer(t, "")
	if got := do(t, s, "GET", "/health", "", "").Code; got != http.StatusOK {
		t.Errorf("open server rejected an unauthenticated request: %d", got)
	}
}

func TestUnknownSubjectIs404(t *testing.T) {
	s := newServer(t, "")
	if got := do(t, s, "GET", "/state?subject=nope", "", "").Code; got != http.StatusNotFound {
		t.Errorf("unknown subject -> %d, want 404", got)
	}
	w := do(t, s, "POST", "/exchange", `{"subject":"nope","phase":"start"}`, "")
	if w.Code != http.StatusNotFound {
		t.Errorf("exchange with unknown subject -> %d, want 404", w.Code)
	}
}

// The first exchange must deliver a readable chapter without any model call.
func TestStartExchangeDeliversAChapter(t *testing.T) {
	s := newServer(t, "")
	w := do(t, s, "POST", "/exchange", `{"subject":"ai","phase":"start"}`, "")
	if w.Code != http.StatusOK {
		t.Fatalf("exchange -> %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Chapter *struct {
			Unit    string           `json:"unit"`
			HTML    string           `json:"html"`
			Beats   []map[string]any `json:"beats"`
			Pretest []map[string]any `json:"pretest"`
			Check   []map[string]any `json:"check"`
		} `json:"chapter"`
		State map[string]any `json:"state"`
		Gate  *struct{}      `json:"gate"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Chapter == nil {
		t.Fatal("no chapter delivered")
	}
	if resp.Chapter.Unit != "u0" {
		t.Errorf("first chapter is %s, want u0", resp.Chapter.Unit)
	}
	// u0 is the calibration unit: a bare progressive question series - no
	// beats, no pretest, the whole bank as the check.
	if len(resp.Chapter.Beats) != 0 {
		t.Errorf("calibration chapter has %d beats, want none", len(resp.Chapter.Beats))
	}
	if len(resp.Chapter.Pretest) != 0 {
		t.Errorf("calibration chapter has %d pretest items, want none", len(resp.Chapter.Pretest))
	}
	if len(resp.Chapter.Check) != 15 {
		t.Errorf("%d check items, want the full 15-item series", len(resp.Chapter.Check))
	}
	if strings.Contains(resp.Chapter.HTML, "MATHPLACEHOLDER") {
		t.Error("math placeholder leaked into the delivered html")
	}
	if resp.Gate != nil {
		t.Error("a start exchange has nothing to gate")
	}
	// Every answer the client sees must be a string, whatever the YAML said.
	for _, item := range resp.Chapter.Check {
		rev, ok := item["reveal"].(map[string]any)
		if !ok {
			continue
		}
		if a, present := rev["answer"]; present {
			if _, isString := a.(string); !isString {
				t.Errorf("item %v answer is %T, must be a string or the whole payload fails to decode",
					item["id"], a)
			}
		}
	}
}

// Failing the gate must hold the learner, and overriding must let them past
// while recording the debt. u0 is a calibration unit and never gates, so
// the gate mechanics are exercised on u1's bank directly - grading needs
// only the corpus, not a delivered chapter.
func TestGateHoldsAndOverrideAccruesDebt(t *testing.T) {
	s := newServer(t, "")
	sub, _ := s.subject("ai")

	// Answer mechanically checkable u1 items wrong, so no model is involved.
	var answers []string
	for _, q := range sub.Corpus.Units["u1"].Questions.Check {
		answers = append(answers, `{"item_id":"`+q.ID+`","response":"definitely wrong","selected_index":99,"confidence":4}`)
	}
	if len(answers) == 0 {
		t.Fatal("u1 has no check items")
	}
	body := `{"subject":"ai","phase":"boundary","unit":"u1","check_responses":[` +
		strings.Join(answers, ",") + `]}`
	w := do(t, s, "POST", "/exchange", body, "")
	if w.Code != http.StatusOK {
		t.Fatalf("check exchange -> %d: %s", w.Code, w.Body.String())
	}
	var graded struct {
		Gate *Gate `json:"gate"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &graded); err != nil {
		t.Fatal(err)
	}
	if graded.Gate == nil {
		t.Fatal("no gate result for a submitted check")
	}
	if graded.Gate.Passed {
		t.Error("all-wrong check passed the gate")
	}
	if len(sub.Learner.OpenDebt()) != 0 {
		t.Error("failing without overriding should not accrue debt on its own")
	}

	override := `{"subject":"ai","phase":"boundary","unit":"u1","override":true,"check_responses":[` +
		strings.Join(answers, ",") + `]}`
	if w := do(t, s, "POST", "/exchange", override, ""); w.Code != http.StatusOK {
		t.Fatalf("override exchange -> %d", w.Code)
	}
	if len(sub.Learner.OpenDebt()) == 0 {
		t.Error("override must record the debt - that is the bargain")
	}
	if sub.Learner.UnitStatus("u1") != "overridden" {
		t.Errorf("unit status after override: %s", sub.Learner.UnitStatus("u1"))
	}
}

// The calibration unit is the opposite contract: the full bank arrives in
// authored order, and even an all-"I don't know" run clears the gate - the
// score is measurement, not a verdict.
func TestCalibrationDeliversFullBankAndNeverGates(t *testing.T) {
	s := newServer(t, "")
	sub, _ := s.subject("ai")

	start := do(t, s, "POST", "/exchange", `{"subject":"ai","phase":"start"}`, "")
	var first struct {
		Chapter struct {
			Unit  string `json:"unit"`
			Check []struct {
				ID string `json:"id"`
			} `json:"check"`
		} `json:"chapter"`
	}
	if err := json.Unmarshal(start.Body.Bytes(), &first); err != nil {
		t.Fatal(err)
	}
	if first.Chapter.Unit != "u0" {
		t.Fatalf("first unit = %s, want u0", first.Chapter.Unit)
	}
	bank := sub.Corpus.Units["u0"].Questions.Check
	if len(first.Chapter.Check) != len(bank) {
		t.Fatalf("delivered %d items, want the whole bank (%d)",
			len(first.Chapter.Check), len(bank))
	}
	for i, item := range first.Chapter.Check {
		if item.ID != bank[i].ID {
			t.Fatalf("item %d is %s, want authored order (%s)", i, item.ID, bank[i].ID)
		}
	}

	var answers []string
	for _, item := range first.Chapter.Check {
		answers = append(answers, `{"item_id":"`+item.ID+`","idk":true,"confidence":1}`)
	}
	body := `{"subject":"ai","phase":"boundary","unit":"u0","check_responses":[` +
		strings.Join(answers, ",") + `]}`
	w := do(t, s, "POST", "/exchange", body, "")
	if w.Code != http.StatusOK {
		t.Fatalf("calibration exchange -> %d: %s", w.Code, w.Body.String())
	}
	var graded struct {
		Gate *Gate `json:"gate"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &graded); err != nil {
		t.Fatal(err)
	}
	if graded.Gate == nil || !graded.Gate.Passed || !graded.Gate.Calibration {
		t.Fatalf("calibration gate = %+v, want passed calibration", graded.Gate)
	}
	if lvl := sub.Learner.ConceptLevel("c-dotprod"); lvl == "mastered" {
		t.Error("all-IDK calibration must not master concepts")
	}
}

func TestReviewScheduleIsSelfContainedMarkdown(t *testing.T) {
	s := newServer(t, "")
	w := do(t, s, "GET", "/review-schedule", "", "")
	if w.Code != http.StatusOK {
		t.Fatalf("-> %d", w.Code)
	}
	body := w.Body.String()
	for _, want := range []string{"# Review schedule", "## Day 1", "## Day 3", "## Day 10"} {
		if !strings.Contains(body, want) {
			t.Errorf("schedule is missing %q", want)
		}
	}
	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/plain") {
		t.Errorf("content type %q - this is meant to be printed or synced", ct)
	}
}

func TestTeachEndpointGuards(t *testing.T) {
	s := newServer(t, "")
	for _, tc := range []struct {
		name, path, body string
		want             int
	}{
		{"empty conversation", "/teach/turn", `{"messages":[]}`, http.StatusUnprocessableEntity},
		{"slug with no letters", "/teach/create", `{"slug":"---","title":"x","brief":"y"}`, http.StatusUnprocessableEntity},
		{"existing subject", "/teach/create", `{"slug":"ai","title":"x","brief":"y"}`, http.StatusConflict},
	} {
		if got := do(t, s, "POST", tc.path, tc.body, "").Code; got != tc.want {
			t.Errorf("%s -> %d, want %d", tc.name, got, tc.want)
		}
	}
	if got := do(t, s, "GET", "/teach/jobs?slug=nope", "", "").Code; got != http.StatusNotFound {
		t.Errorf("unknown job -> %d, want 404", got)
	}
}

// A model-proposed slug names a directory, so it must be constrained - but
// failing the request over underscores or capitals refuses something with an
// obvious right answer.
func TestSlugify(t *testing.T) {
	for in, want := range map[string]string{
		"Jet_Engine_Book_Plan!":       "jet-engine-book-plan",
		"  postgres locks  ":          "postgres-locks",
		"already-fine":                "already-fine",
		"---":                         "",
		"Über Crypto":                 "ber-crypto",
		strings.Repeat("verylong", 8): strings.Repeat("verylong", 4),
	} {
		if got := slugify(in); got != want {
			t.Errorf("slugify(%q) = %q, want %q", in, got, want)
		}
	}
}
