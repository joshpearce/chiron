package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mjbraun/chiron/server/llm"
	"github.com/mjbraun/chiron/server/sources"
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
		DataDir: t.TempDir(),
		Subjects: []SubjectSpec{{
			ID: "ai", Title: "How AI Works",
			CorpusDir: filepath.Join("..", "corpus"),
			StateDir:  t.TempDir(),
		}},
		AuthToken:  token,
		PrimersDir: t.TempDir(),
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
	// HTTP fixtures listen on loopback. Production page/feed imports use the
	// public-only client and reject these addresses to prevent SSRF.
	s.newPublicClient = sources.NewClient
	// Eager page renders run in the background; the state dir is a TempDir,
	// so cleanup must not race them.
	t.Cleanup(func() { s.renders.Wait(); _ = s.Close() })
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

func TestFileSecretsAndExplicitPersistentRoots(t *testing.T) {
	t.Setenv("CHIRON_PROVIDER", "")
	root := t.TempDir()
	data := filepath.Join(root, "persistent")
	claude := filepath.Join(root, "claude-state")
	auth := filepath.Join(root, "auth")
	if err := os.WriteFile(auth, []byte(" bearer-key\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := &Config{
		DataDir: data, GeneratedCorporaDir: filepath.Join(data, "corpora"),
		Provider: "claude-cli", ClaudeConfigDir: claude,
		AuthTokenFile: auth, PrimersDir: "", ReadingsDir: "", RequestsDir: "", BuildsDir: "",
	}
	s, err := New(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	if s.token != "bearer-key" {
		t.Fatalf("file auth token was not loaded")
	}
	if got := s.chain.(*llm.ClaudeCLI).ConfigDir; got != claude {
		t.Fatalf("Claude config dir = %q, want %q", got, claude)
	}
	for got, want := range map[string]string{
		s.readingsRoot():         filepath.Join(data, "readings"),
		s.primersRoot():          filepath.Join(data, "primers"),
		s.requests.Dir():         filepath.Join(data, "requests"),
		s.buildsDir:              filepath.Join(data, "builds"),
		s.documentsDir():         filepath.Join(data, "documents"),
		s.generatedCorporaRoot(): filepath.Join(data, "corpora"),
		s.cacheRoot():            filepath.Join(data, "cache"),
		s.tempRoot():             filepath.Join(data, "tmp"),
	} {
		if got != want {
			t.Errorf("path %q, want %q", got, want)
		}
	}
}

func TestDataRootAllowsOnlyOneServerWriter(t *testing.T) {
	root := t.TempDir()
	cfg := &Config{DataDir: filepath.Join(root, "data")}
	first, err := New(cfg, root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := New(&Config{DataDir: cfg.DataDir}, root); err == nil ||
		!strings.Contains(err.Error(), "already has a Chiron writer") {
		t.Fatalf("second server error = %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	second, err := New(&Config{DataDir: cfg.DataDir}, root)
	if err != nil {
		t.Fatalf("lock was not released: %v", err)
	}
	t.Cleanup(func() { _ = second.Close() })
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
	// u0 is the calibration unit: it opens with the self-placement screener
	// alone - no beats, no pretest, the series follows the answer.
	if len(resp.Chapter.Beats) != 0 {
		t.Errorf("calibration chapter has %d beats, want none", len(resp.Chapter.Beats))
	}
	if len(resp.Chapter.Pretest) != 0 {
		t.Errorf("calibration chapter has %d pretest items, want none", len(resp.Chapter.Pretest))
	}
	if len(resp.Chapter.Check) != 1 {
		t.Errorf("%d check items, want the screener alone", len(resp.Chapter.Check))
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
	if len(first.Chapter.Check) != 1 || first.Chapter.Check[0].ID != "u0-s1" {
		t.Fatalf("first calibration chapter = %v, want the screener alone", first.Chapter.Check)
	}

	// Rating 4 selects the level-4 pre-computed set - no gate on the way.
	screen := do(t, s, "POST", "/exchange",
		`{"subject":"ai","phase":"boundary","unit":"u0","check_responses":[{"item_id":"u0-s1","response":"4","confidence":3}]}`, "")
	var second struct {
		Gate    *Gate `json:"gate"`
		Chapter struct {
			Unit  string `json:"unit"`
			Check []struct {
				ID string `json:"id"`
			} `json:"check"`
		} `json:"chapter"`
	}
	if err := json.Unmarshal(screen.Body.Bytes(), &second); err != nil {
		t.Fatal(err)
	}
	if second.Gate != nil {
		t.Fatalf("screener answer produced a gate: %+v", second.Gate)
	}
	if second.Chapter.Unit != "u0" {
		t.Fatalf("after screener got unit %s, want u0 again", second.Chapter.Unit)
	}
	want := sub.Corpus.Units["u0"].Questions.CalibrationSets[4]
	if len(second.Chapter.Check) != len(want) {
		t.Fatalf("level-4 series has %d items, want %d", len(second.Chapter.Check), len(want))
	}
	for i, item := range second.Chapter.Check {
		if item.ID != want[i] {
			t.Fatalf("item %d is %s, want %s (pre-computed order)", i, item.ID, want[i])
		}
	}
	first.Chapter.Check = second.Chapter.Check

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

// A book made from named sources: the names reach the generator with the
// brief, first the spine, then what interleaves.
func TestTeachCreatePassesTheSourcesOn(t *testing.T) {
	s := newServer(t, "")
	var got []string
	var planOnly bool
	s.startGenerate = func(slug, title, brief string, named []string, plan bool) {
		got = append([]string{slug, title, brief}, named...)
		planOnly = plan
	}
	w := do(t, s, "POST", "/teach/create", `{"slug":"bayes","title":"Bayes for Engineers","brief":"short","sources":["Think Bayes","MIT 18.05"]}`, "")
	if w.Code != http.StatusOK {
		t.Fatalf("create -> %d: %s", w.Code, w.Body.String())
	}
	if strings.Join(got, "|") != "bayes|Bayes for Engineers|short|Think Bayes|MIT 18.05" || planOnly {
		t.Fatalf("generator got %v, plan only %v", got, planOnly)
	}
	// Plan first, author later: the request says so, and the generator
	// stops once the syllabus is on disk.
	s.jobs["bayes"].Done = true
	w = do(t, s, "POST", "/teach/create", `{"slug":"bayes","title":"Bayes for Engineers","brief":"short","plan_only":true}`, "")
	if w.Code != http.StatusOK || !planOnly {
		t.Fatalf("plan-only create -> %d: %s, plan only %v", w.Code, w.Body.String(), planOnly)
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
