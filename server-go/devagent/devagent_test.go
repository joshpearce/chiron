package devagent

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/mjbraun/chiron/server/devreq"
)

// A scripted stand-in for git, claude, make: records every call, answers
// from a table keyed by the command's first words.
type fakeExec struct {
	calls  []string
	answer map[string]string // "git log" -> output; "!make test" -> fails with that output
	stream []string          // claude's stream-json lines
}

func (f *fakeExec) key(name string, args []string) string {
	k := name
	if len(args) > 0 {
		k += " " + args[0]
	}
	if name == "git" && len(args) > 2 && args[0] == "-C" {
		k = "git " + args[2]
	}
	return k
}

func (f *fakeExec) Run(dir, name string, args ...string) (string, error) {
	f.calls = append(f.calls, strings.TrimSpace(name+" "+strings.Join(args, " ")))
	k := f.key(name, args)
	if out, ok := f.answer["!"+k]; ok {
		return out, fmt.Errorf("%s failed", k)
	}
	return f.answer[k], nil
}

func (f *fakeExec) Stream(dir, name string, args []string, onLine func(string)) error {
	f.calls = append(f.calls, name+" ...")
	for _, l := range f.stream {
		onLine(l)
	}
	return nil
}

func (f *fakeExec) did(prefix string) bool {
	for _, c := range f.calls {
		if strings.HasPrefix(c, prefix) {
			return true
		}
	}
	return false
}

func agent(t *testing.T, x *fakeExec) (*Agent, *devreq.Store) {
	store := devreq.Open(t.TempDir())
	served := t.TempDir()
	return &Agent{Repo: "/repo", Work: "/work", Served: served, Store: store, Model: "claude-fable-5-1", Exec: x}, store
}

// The happy path: the agent works in a worktree off main, commits, the
// suite passes, main moves, the app changed so a build is made, and the
// request ends ready with the agent's summary and the build.
func TestARequestBecomesACommitAndABuild(t *testing.T) {
	x := &fakeExec{
		answer: map[string]string{
			"git log":        "abc123 A thicker pen\n",
			"git diff":       "ipad-app/Chiron/Palette.swift\nipad-app/ChironTests/PaletteTests.swift\n",
			"git rev-parse":  "abc123\n",
			"git status":     "",
			"make app-build": "build 20260920-1 ready 2026.9.20 (310)\n",
		},
		stream: []string{
			`{"type":"assistant","message":{"content":[{"type":"text","text":"Reading Palette.swift to find the pen width."}]}}`,
			`{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Edit"}]}}`,
			`{"type":"result","subtype":"success","result":"Widened the pen from 2.7 to 3.5 and covered it in PaletteTests."}`,
		},
	}
	a, store := agent(t, x)
	writeBuild(t, a.Served, `{"version":"2026.9.20","build":310,"token":"`+strings.Repeat("ab", 32)+`","status":"ready"}`)
	r, _ := store.Create("the pen is too thin", json.RawMessage(`{"screen":"reading"}`), []byte("PNG"))

	took, err := a.Once()
	if err != nil || !took {
		t.Fatalf("once: %v %v", took, err)
	}
	got, _ := store.Get(r.ID)
	if got.Status != devreq.Ready {
		t.Fatalf("status %s, reason %q, log %v", got.Status, got.Reason, got.Log)
	}
	if got.Summary != "Widened the pen from 2.7 to 3.5 and covered it in PaletteTests." || got.Commit != "abc123" {
		t.Errorf("summary %q commit %q", got.Summary, got.Commit)
	}
	if got.Build == nil || got.Build.Number != 310 || got.Build.Version != "2026.9.20" {
		t.Errorf("build = %+v", got.Build)
	}
	if !x.did("git -C /repo worktree add") || !x.did("claude") || !x.did("make test") || !x.did("git -C /repo merge --ff-only req/"+r.ID) || !x.did("make app-build") {
		t.Errorf("calls: %v", x.calls)
	}
	if x.did("make deploy") {
		t.Error("nothing in the server changed; no deploy")
	}
	if !x.did("git -C /repo worktree remove") {
		t.Error("the worktree is cleared after success")
	}
	joined := strings.Join(got.Log, "\n")
	if !strings.Contains(joined, "Reading Palette.swift") {
		t.Errorf("the agent's own words are in the log: %v", got.Log)
	}
	if took, _ := a.Once(); took {
		t.Error("nothing left to take")
	}
}

// A server change is deployed and needs no app build.
func TestAServerChangeIsDeployed(t *testing.T) {
	x := &fakeExec{answer: map[string]string{
		"git log": "def456 Detail is shorter\n", "git diff": "server-go/roles/capture.go\n", "git rev-parse": "def456\n",
	}, stream: []string{`{"type":"result","result":"done"}`}}
	a, store := agent(t, x)
	r, _ := store.Create("make detail shorter", nil, nil)
	a.Once()
	got, _ := store.Get(r.ID)
	if got.Status != devreq.Ready || !x.did("make deploy") || x.did("make app-build") || got.Build != nil {
		t.Errorf("status %s build %+v calls %v", got.Status, got.Build, x.calls)
	}
}

// When the agent commits nothing, or the suite fails, the request fails
// with the reason and the worktree is kept for a person.
func TestFailuresAreSaidAndTheWorktreeKept(t *testing.T) {
	x := &fakeExec{answer: map[string]string{"git log": ""}, stream: []string{`{"type":"result","result":"I could not find the pen width."}`}}
	a, store := agent(t, x)
	r, _ := store.Create("the pen is too thin", nil, nil)
	a.Once()
	got, _ := store.Get(r.ID)
	if got.Status != devreq.Failed || !strings.Contains(got.Reason, "no change") || got.Summary != "I could not find the pen width." {
		t.Errorf("no-commit: %s %q %q", got.Status, got.Reason, got.Summary)
	}
	if x.did("git -C /repo worktree remove") || x.did("git -C /repo merge") {
		t.Error("kept, not merged")
	}

	x = &fakeExec{answer: map[string]string{"git log": "abc A change\n", "git diff": "server-go/x.go\n", "!make test": "--- FAIL: TestX\nFAIL\n"}, stream: []string{`{"type":"result","result":"done"}`}}
	a, store = agent(t, x)
	r, _ = store.Create("break something", nil, nil)
	a.Once()
	got, _ = store.Get(r.ID)
	if got.Status != devreq.Failed || !strings.Contains(got.Reason, "FAIL: TestX") || x.did("git -C /repo merge") {
		t.Errorf("test failure: %s %q calls %v", got.Status, got.Reason, x.calls)
	}
}

// The brief the agent works from names the request, the app's state,
// the screenshot and the rules.
func TestTheBriefCarriesWhatTheAgentNeeds(t *testing.T) {
	a, store := agent(t, &fakeExec{})
	r, _ := store.Create("the pen is too thin", json.RawMessage(`{"screen":"reading","unit":"u3"}`), []byte("PNG"))
	p := a.prompt(r)
	for _, want := range []string{"the pen is too thin", `"unit":"u3"`, r.Screenshot, "LESSONS.md", "Do not run the whole suite", "git commit", "push"} {
		if !strings.Contains(p, want) {
			t.Errorf("brief lacks %q:\n%s", want, p)
		}
	}
}

func writeBuild(t *testing.T, served, buildJSON string) {
	t.Helper()
	dir := served + "/builds/latest"
	if err := mkdirAll(dir); err != nil {
		t.Fatal(err)
	}
	if err := writeFile(dir+"/build.json", buildJSON); err != nil {
		t.Fatal(err)
	}
}
