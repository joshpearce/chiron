package llm

import (
	"context"
	"strings"
	"testing"
	"time"
)

// The case that matters is `--tools ""`. With the built-in tools enabled the
// model can spend its single allowed turn on a tool call; the CLI then exits
// non-zero with subtype error_max_turns and the generated work is discarded.
// That failure is intermittent and surfaces as a bare "exit 1", so it is
// exactly the kind of thing that comes back silently.
func TestCommandDisablesTools(t *testing.T) {
	c := &ClaudeCLI{}
	argv := c.Command("grader", "PROMPT", "SYSTEM")
	flags := map[string]string{}
	for i := 0; i < len(argv)-1; i++ {
		if strings.HasPrefix(argv[i], "--") {
			flags[argv[i]] = argv[i+1]
		}
	}
	if v, ok := flags["--tools"]; !ok || v != "" {
		t.Error("tools are not disabled - a tool turn will exit error_max_turns and discard the grade")
	}
	if flags["--max-turns"] != "1" {
		t.Error("max-turns should stay 1 once tools are off")
	}
	if flags["--output-format"] != "json" {
		t.Error("Structured parses the json envelope")
	}
	if argv[0] != "claude" || argv[1] != "-p" {
		t.Errorf("must invoke the CLI headless, got %v", argv[:2])
	}
	if flags["--append-system-prompt"] != "SYSTEM" {
		t.Error("system prompt must reach the CLI")
	}
	found := false
	for _, a := range argv {
		if a == "PROMPT" {
			found = true
		}
	}
	if !found {
		t.Error("prompt must reach the CLI")
	}
}

func TestPerRoleModelTiers(t *testing.T) {
	c := &ClaudeCLI{}
	for role, want := range map[string]string{
		"grader": "sonnet", "planner": "sonnet", "author": "opus",
	} {
		argv := c.Command(role, "p", "s")
		if got := argv[len(argv)-1]; got != want {
			t.Errorf("%s uses %q, want %q", role, got, want)
		}
	}
	pinned := &ClaudeCLI{Model: "haiku"}
	argv := pinned.Command("author", "p", "s")
	if got := argv[len(argv)-1]; got != "haiku" {
		t.Errorf("an explicit model must override the per-role tiers, got %q", got)
	}
}

func TestUnmarshalLooseHandlesFencedJSON(t *testing.T) {
	var out struct {
		Verdict string `json:"verdict"`
	}
	for _, in := range []string{
		`{"verdict":"pass"}`,
		"```json\n{\"verdict\":\"pass\"}\n```",
		"Here you go:\n{\"verdict\":\"pass\"}\nHope that helps.",
	} {
		out.Verdict = ""
		if err := unmarshalLoose(in, &out); err != nil {
			t.Errorf("%q: %v", in, err)
			continue
		}
		if out.Verdict != "pass" {
			t.Errorf("%q -> %q", in, out.Verdict)
		}
	}
	if err := unmarshalLoose("no json here", &out); err == nil {
		t.Error("prose with no object must error rather than silently succeed")
	}
}

// The planner emits the whole syllabus in one call: every unit with its
// concepts, prereqs, notes and sources, plus the misconception bank. On
// 2026-09-05 three book builds in a row hit the 5-minute deadline at exactly
// that step, and the CLI reports nothing but "timed out (planner)". The
// planner's budget has to match the author's, which also writes a chapter's
// worth of output in one call.
func TestPlannerTimeoutCoversASyllabus(t *testing.T) {
	if got, want := roleTimeouts["planner"], roleTimeouts["author"]; got < want {
		t.Errorf("planner timeout %v is shorter than the author's %v; a syllabus is one call and needs the same room", got, want)
	}
}

// A CLI process killed at the deadline may leave a child holding its
// output pipe; the call must still return promptly rather than wait for
// that child. Four author calls once took 108 minutes to time out at 30.
func TestRunReturnsAtTheDeadlineDespiteAChildOnThePipe(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := run(ctx, []string{"sh", "-c", "sleep 30 & exec sleep 30"})
	if err == nil {
		t.Fatal("a killed call returned no error")
	}
	if took := time.Since(start); took > 20*time.Second {
		t.Fatalf("run returned after %v; the child's pipe held it", took)
	}
}

func TestCLICallsRunWithoutTheUpdater(t *testing.T) {
	env := strings.Join(cliEnv(), "\n")
	for _, want := range []string{"DISABLE_AUTOUPDATER=1", "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1"} {
		if !strings.Contains(env, want) {
			t.Errorf("env lacks %s", want)
		}
	}
}
