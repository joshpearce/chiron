package llm

import (
	"context"
	"encoding/json"
	"os/exec"
	"strings"
	"time"
)

// ClaudeCLI runs the tutor roles through headless `claude -p`, authenticated by
// the user's Claude Code login - subscription billing, no API key on the box.
//
// There is no token-level schema enforcement here: the JSON contract is
// prompt-enforced, extracted, and retried once.
type ClaudeCLI struct {
	// Model, when set, overrides the per-role tiers entirely.
	Model string
}

// The planner writes a whole syllabus in one call (every unit's concepts,
// prereqs, notes and sources, plus the misconception bank), so it gets a
// long budget: at five minutes a reference-depth book timed out at that
// step three times running. The author's depth variants are longer still:
// a deeper-math file for a 35 KB chapter, written at the CLI's ~33 tokens
// a second with three units in flight, ran past fifteen minutes three
// times on the first sourced book.
var roleTimeouts = map[string]time.Duration{
	"grader":  5 * time.Minute,
	"planner": 30 * time.Minute,
	"author":  30 * time.Minute,
}

// Per-role model tiers. Grading is a bounded judgement against an explicit
// rubric, and sonnet scores 5/5 on the sycophancy red-team at roughly a sixth
// of opus's latency, which is what makes a nine-item check tolerable. Authoring
// is open-ended prose the learner reads for 25 minutes, so it keeps opus.
var roleModels = map[string]string{
	"grader":  "sonnet",
	"planner": "sonnet",
	"author":  "opus",
}

func (c *ClaudeCLI) available() bool {
	// Presence of the binary, not of credentials: the login lives in a file on
	// Linux but in the Keychain on macOS, so probing storage reports "not
	// logged in" on a machine where it works. A real auth failure surfaces
	// through the CLI's own output, which says more than a guess would.
	_, err := exec.LookPath("claude")
	return err == nil
}

func (c *ClaudeCLI) ModelFor(role string) string {
	if c.Model != "" {
		return c.Model
	}
	if m, ok := roleModels[role]; ok {
		return m
	}
	return "opus"
}

func (c *ClaudeCLI) Status() Status {
	if !c.available() {
		return Status{Connected: false, Upstream: "claude-cli", Error: "`claude` not on PATH"}
	}
	model := c.Model
	if model == "" {
		model = "per-role: grader=sonnet, planner=sonnet, author=opus"
	}
	return Status{Connected: true, Upstream: "claude-cli", Model: model}
}

// Command builds the argv. Exported so the contract below can be tested without
// spending a model call.
func (c *ClaudeCLI) Command(role, prompt, system string) []string {
	return []string{
		"claude", "-p", prompt,
		"--append-system-prompt", system,
		"--output-format", "json",
		// These roles are pure text generation. With the built-in tools
		// enabled the model can spend its single allowed turn on a tool call;
		// the CLI then exits non-zero as error_max_turns and the generated work
		// is discarded. It also prepends ~17k tokens of tool schemas to every
		// call, which is most of what made grading slow (~55s/item -> ~7s).
		"--tools", "",
		"--max-turns", "1",
		"--model", c.ModelFor(role),
	}
}

func (c *ClaudeCLI) Structured(role, system, user string, schema map[string]any,
	schemaName string, out any) error {
	if !c.available() {
		return Errorf("claude-cli: `claude` not on PATH")
	}
	schemaJSON, err := json.Marshal(schema)
	if err != nil {
		return err
	}
	prompt := user + "\n\nRespond with ONLY a single JSON object matching this JSON Schema - " +
		"no prose, no code fences, no tool use:\n" + string(schemaJSON)

	timeout, ok := roleTimeouts[role]
	if !ok {
		timeout = 4 * time.Minute
	}
	lastErr := "unknown"
	for attempt := range 2 {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		// The prompt carries learner and corpus text, and on the sprite some of
		// it arrives from a client. It is passed as a single argv element to
		// exec, never through a shell, so there is nothing to inject into: no
		// word splitting, no globbing, no metacharacters. Building a command
		// string and handing it to `sh -c` is what would make this dangerous.
		argv := c.Command(role, prompt, system)
		cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
		stdout, err := cmd.Output()
		cancel()

		if ctx.Err() == context.DeadlineExceeded {
			return Errorf("claude-cli: timed out (%s)", role)
		}
		if err != nil {
			// With --output-format json the CLI reports the cause on stdout,
			// so stderr alone leaves a bare "exit 1".
			detail := strings.TrimSpace(string(stdout))
			if ee, isExit := err.(*exec.ExitError); isExit && len(ee.Stderr) > 0 {
				detail = strings.TrimSpace(string(ee.Stderr))
			}
			lastErr = tail(detail, 400)
			if lastErr == "" {
				lastErr = err.Error()
			}
			continue
		}
		var envelope struct {
			Result string `json:"result"`
		}
		if err := json.Unmarshal(stdout, &envelope); err != nil {
			lastErr = "unreadable envelope: " + err.Error()
			continue
		}
		if err := unmarshalLoose(envelope.Result, out); err != nil {
			lastErr = "unparseable output: " + err.Error()
			if attempt == 0 {
				prompt += "\n\nYour previous reply was not a valid bare JSON object. " +
					"Return ONLY the JSON object."
			}
			continue
		}
		return nil
	}
	return Errorf("claude-cli: %s", lastErr)
}

func tail(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}
