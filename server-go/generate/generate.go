// Package generate builds a whole subject corpus from a one-paragraph brief.
//
// This automates what was done by hand for "How AI Works": a syllabus with a
// prerequisite graph, a misconception bank, then one authored unit per syllabus
// entry. The authoring contract is corpus/authoring-spec.md, which is
// subject-agnostic on purpose.
//
// Units are authored one file per call and skip files already on disk, so an
// interrupted run resumes rather than re-paying. Asking for all six files in a
// single response produced a ~30k-token object that ran past ten minutes and
// failed as a unit - one malformed character anywhere lost the whole chapter.
package generate

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"

	"github.com/mjbraun/chiron/server/llm"
	"github.com/mjbraun/chiron/server/sources"
)

// Progress is called after each unit finishes so a caller can report it.
type Progress func(done, total int)

type unitPlan struct {
	ID       string   `json:"id" yaml:"id"`
	Slug     string   `json:"slug" yaml:"slug"`
	Title    string   `json:"title" yaml:"title"`
	Minutes  int      `json:"minutes" yaml:"minutes"`
	Prereqs  []string `json:"prereqs" yaml:"prereqs"`
	Concepts []struct {
		ID   string `json:"id" yaml:"id"`
		Name string `json:"name" yaml:"name"`
	} `json:"concepts" yaml:"concepts"`
	Notes string `json:"notes" yaml:"notes"`
	// Sources are the sections of the chosen sources this unit adapts.
	Sources []UnitSource `json:"sources,omitempty" yaml:"sources,omitempty"`
}

type misconceptionPlan struct {
	ID                string   `json:"id" yaml:"id"`
	Name              string   `json:"name" yaml:"name"`
	WrongModel        string   `json:"wrong_model" yaml:"wrong_model"`
	WhyAppealing      string   `json:"why_appealing" yaml:"why_appealing"`
	FailingPrediction string   `json:"failing_prediction" yaml:"failing_prediction"`
	Correction        string   `json:"correction" yaml:"correction"`
	Units             []string `json:"units" yaml:"units"`
}

type plan struct {
	Title          string              `json:"title"`
	LearnerProfile string              `json:"learner_profile"`
	Units          []unitPlan          `json:"units"`
	Misconceptions []misconceptionPlan `json:"misconceptions"`
}

const planSystem = `You design curricula that will be taught by an adaptive textbook.

Two hard constraints shape every choice:

1. **The spine must be completable.** Progress is gated on ~80% mastery per unit, so an overstuffed syllabus does not run long - it stalls. Budget the whole spine at roughly 60% of the learner's stated time, leaving room for remediation and breaks. Order units so every prerequisite precedes its dependents; the graph is what decides what the learner may read next.

2. **Misconceptions are the teaching material, not a footnote.** For this domain, list the wrong models a capable learner actually arrives with - especially ones that are confidently held and partially correct. Each needs a *specific prediction it makes that observably fails*; that failing prediction is what the chapter will use to break it. Vague "some people think X is hard" entries are useless. Aim for at least one per unit, more where the domain is counterintuitive.

Concept ids are kebab-case and prefixed ` + "`c-`" + `. Unit ids are u0, u1, ... in teaching order. Misconception ids are M1, M2, ... Write for the specific learner described in the brief: name what they already know so the chapters do not re-explain it.`

const unitSystem = `You author one unit of an adaptive textbook corpus, following the authoring contract exactly. The contract is not advice - the server parses these files mechanically, and a deviation breaks the unit.

The two things that most often go wrong, and that you must get right:

- **Mechanically-checked items need machine-comparable answers.** If ` + "`check`" + ` is ` + "`numeric(tol)`" + ` or ` + "`exact`" + `, ` + "`answer`" + ` is a bare value, never a sentence. Choose ` + "`tol`" + ` tighter than the distance to every plausible wrong answer you name in the rubric - a tolerance that accepts the error the item exists to catch makes the item worthless.
- **Every MCQ carries ` + "`check: choice`" + `**, exactly one option marked ` + "`correct`" + `, an ` + "`explain`" + ` on every option, and a misconception id from the bank on every distractor. An MCQ without ` + "`check: choice`" + ` is not graded as an MCQ at all, and a wrong answer must be a diagnosis rather than a miss.`

func planSchema() map[string]any {
	conceptItem := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"id":   map[string]any{"type": "string"},
			"name": map[string]any{"type": "string"},
		},
		"required": []string{"id", "name"},
	}
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"title": map[string]any{"type": "string"},
			"learner_profile": map[string]any{"type": "string",
				"description": "who this is for, including what they already know"},
			"units": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"id":       map[string]any{"type": "string", "description": "u0, u1, ..."},
						"slug":     map[string]any{"type": "string"},
						"title":    map[string]any{"type": "string"},
						"minutes":  map[string]any{"type": "integer"},
						"prereqs":  map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
						"concepts": map[string]any{"type": "array", "items": conceptItem},
						"notes":    map[string]any{"type": "string"},
						"sources": map[string]any{
							"type":        "array",
							"description": "sections of the given sources this unit adapts; empty when none apply",
							"items": map[string]any{
								"type": "object",
								"properties": map[string]any{
									"source":  map[string]any{"type": "string", "description": "the source id in brackets"},
									"locator": map[string]any{"type": "string", "description": "the section locator exactly as listed"},
									"role":    map[string]any{"type": "string", "description": "spine or interleave"},
								},
								"required": []string{"source", "locator", "role"},
							},
						},
					},
					"required": []string{"id", "slug", "title", "minutes", "prereqs", "concepts", "notes", "sources"},
				},
			},
			"misconceptions": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"id":            map[string]any{"type": "string", "description": "M1, M2, ..."},
						"name":          map[string]any{"type": "string", "description": "short kebab-case handle"},
						"wrong_model":   map[string]any{"type": "string"},
						"why_appealing": map[string]any{"type": "string"},
						"failing_prediction": map[string]any{"type": "string",
							"description": "a specific prediction the wrong model makes that observably fails"},
						"correction": map[string]any{"type": "string"},
						"units":      map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
					},
					"required": []string{"id", "name", "wrong_model", "why_appealing",
						"failing_prediction", "correction", "units"},
				},
			},
		},
		"required": []string{"title", "learner_profile", "units", "misconceptions"},
	}
}

func fileSchema(key, description string) map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			key: map[string]any{"type": "string", "description": description},
		},
		"required": []string{key},
	}
}

// Generator writes a corpus into outDir using chain, against the authoring
// contract at specPath.
type Generator struct {
	Chain    llm.Chain
	SpecPath string
	OutDir   string
	// Index and Fetch are the open sources the book may be built from; with
	// either nil the book is written from the brief alone.
	Index *sources.Index
	Fetch *sources.Client
	// Workers bounds concurrent unit authoring. Units are independent and the
	// wall clock is dominated by generation, but each unit is six sequential
	// calls, so this is what decides whether a subject takes 30 minutes or two
	// hours.
	Workers int
}

// Plan writes syllabus.yaml and misconception-bank.yaml, returning the
// unit count. Named sources (or the ones the brief states) are resolved
// first; the planner then follows the spine's chapters and says which
// sections each unit adapts.
func (g *Generator) Plan(brief, title string, named []string) (int, error) {
	spec, err := os.ReadFile(g.SpecPath)
	if err != nil {
		return 0, fmt.Errorf("authoring spec: %w", err)
	}
	contract := string(spec)
	if len(contract) > 6000 {
		contract = contract[:6000]
	}
	if err := os.MkdirAll(g.OutDir, 0o755); err != nil {
		return 0, err
	}
	// A syllabus already on disk is kept: a rerun after a failed unit
	// finishes the book rather than planning a different one.
	if raw, err := os.ReadFile(filepath.Join(g.OutDir, "syllabus.yaml")); err == nil {
		var existing struct {
			Units []unitPlan `yaml:"units"`
		}
		if yaml.Unmarshal(raw, &existing) == nil && len(existing.Units) > 0 {
			return len(existing.Units), nil
		}
	}
	chosen, unknown := g.resolveSources(context.Background(), named, brief)
	if err := g.writeSources(chosen, unknown); err != nil {
		return 0, err
	}
	system := planSystem
	if len(chosen) > 0 {
		system += sourcesPlanRule
	}
	user := fmt.Sprintf("BRIEF:\n%s\n\n%sDesign the syllabus and misconception bank. "+
		"The authoring contract the units will follow is below, for context on what "+
		"each unit must eventually contain.\n\n%s", brief, sourcesBlock(chosen), contract)

	var p plan
	if err := g.Chain.Structured("planner", system, user, planSchema(), "plan", &p); err != nil {
		return 0, err
	}
	if len(p.Units) == 0 {
		return 0, fmt.Errorf("planner returned no units")
	}
	if title == "" {
		title = p.Title
	}
	// Keep only sections the sources really have, with the role they were given.
	known := map[string]map[string]bool{}
	for _, c := range chosen {
		m := map[string]bool{}
		for _, sec := range c.Sections {
			m[sec.Locator] = true
		}
		known[c.ID] = m
	}
	for i := range p.Units {
		var kept []UnitSource
		for _, us := range p.Units[i].Sources {
			if known[us.Source][us.Locator] {
				if us.Role != "interleave" {
					us.Role = "spine"
				}
				kept = append(kept, us)
			}
		}
		p.Units[i].Sources = kept
	}

	syllabus := map[string]any{
		"title":   title,
		"learner": map[string]any{"profile": p.LearnerProfile},
		"defaults": map[string]any{
			"mastery_gate": 0.80, "extension_trigger": 0.90,
			"check_min_items": 8, "check_constructed_fraction": 0.6,
			"check_callback_fraction": 0.4, "chunk_minutes": []int{20, 25},
		},
		"units": p.Units,
	}
	if err := writeYAML(filepath.Join(g.OutDir, "syllabus.yaml"), syllabus); err != nil {
		return 0, err
	}
	if err := writeYAML(filepath.Join(g.OutDir, "misconception-bank.yaml"),
		map[string]any{"misconceptions": p.Misconceptions}); err != nil {
		return 0, err
	}
	return len(p.Units), nil
}

func writeYAML(path string, v any) error {
	out, err := yaml.Marshal(v)
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0o644)
}

type depthFile struct{ name, what string }

var depths = []depthFile{
	{"deeper-math.md", "the same sections, with the derivations worked in full"},
	{"more-intuition.md", "the same sections, leading with physical/geometric intuition before any symbol"},
	{"se-analogies.md", "the same sections, explained through analogies to software systems, each with its breakdown point stated"},
}

// Units authors every planned unit. It returns the number that failed.
func (g *Generator) Units(only []string, progress Progress) (int, error) {
	raw, err := os.ReadFile(filepath.Join(g.OutDir, "syllabus.yaml"))
	if err != nil {
		return 0, err
	}
	var syllabus struct {
		Learner struct {
			Profile string `yaml:"profile"`
		} `yaml:"learner"`
		Units []unitPlan `yaml:"units"`
	}
	if err := yaml.Unmarshal(raw, &syllabus); err != nil {
		return 0, err
	}
	bank, err := os.ReadFile(filepath.Join(g.OutDir, "misconception-bank.yaml"))
	if err != nil {
		return 0, err
	}
	spec, err := os.ReadFile(g.SpecPath)
	if err != nil {
		return 0, err
	}

	wanted := map[string]bool{}
	for _, id := range only {
		wanted[id] = true
	}
	var todo []unitPlan
	for _, u := range syllabus.Units {
		if len(wanted) == 0 || wanted[u.ID] {
			todo = append(todo, u)
		}
	}

	workers := g.Workers
	if workers < 1 {
		workers = 4
	}
	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup
	var mu sync.Mutex
	failures, done := 0, 0

	for _, u := range todo {
		wg.Add(1)
		go func(u unitPlan) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			err := g.authorUnit(u, syllabus.Learner.Profile, string(bank), string(spec))
			mu.Lock()
			done++
			if err != nil {
				failures++
				log.Printf("generate %s: unit %s: %v", filepath.Base(g.OutDir), u.ID, err)
			}
			d := done
			mu.Unlock()
			if progress != nil {
				progress(d, len(todo))
			}
		}(u)
	}
	wg.Wait()
	return failures, nil
}

func (g *Generator) authorUnit(u unitPlan, learner, bank, spec string) error {
	dir := filepath.Join(g.OutDir, "units", u.ID+"-"+u.Slug)
	if err := os.MkdirAll(filepath.Join(dir, "depths"), 0o755); err != nil {
		return err
	}
	unitYAML, err := yaml.Marshal(u)
	if err != nil {
		return err
	}
	material, provs, err := g.material(context.Background(), dir, u)
	if err != nil {
		return err
	}
	context := fmt.Sprintf("AUTHORING CONTRACT:\n%s\n\nLEARNER:\n%s\n\n"+
		"MISCONCEPTION BANK (cite these ids):\n%s\n\nUNIT TO AUTHOR:\n%s",
		spec, learner, bank, unitYAML)
	// The material is for writing canon.md; every later file is written
	// from canon.md itself, and carrying the sources again only makes those
	// calls slower (three units timed out on it).
	withMaterial := context
	if material != "" {
		withMaterial += "\n\n" + material
	}

	// Files already on disk are kept, so re-running after a failure resumes
	// instead of re-paying for what worked.
	write := func(path, key, description, extra string) (string, error) {
		if info, err := os.Stat(path); err == nil && info.Size() > 0 {
			existing, err := os.ReadFile(path)
			return string(existing), err
		}
		prompt := context
		if key == "canon_md" {
			prompt = withMaterial
		}
		var out map[string]string
		if err := g.Chain.Structured("author", unitSystem, prompt+"\n\n"+extra,
			fileSchema(key, description), key, &out); err != nil {
			return "", err
		}
		body := out[key]
		if strings.TrimSpace(body) == "" {
			return "", fmt.Errorf("%s came back empty", filepath.Base(path))
		}
		return body, os.WriteFile(path, []byte(body), 0o644)
	}

	canonPath := filepath.Join(dir, "canon.md")
	fresh := true
	if info, err := os.Stat(canonPath); err == nil && info.Size() > 0 {
		fresh = false
	}
	canon, err := write(canonPath, "canon_md",
		"full canon.md: front matter, prose, and ```beat blocks",
		"Write canon.md for this unit, and nothing else.")
	if err != nil {
		return err
	}
	// Attribution is the pipeline's job, not the model's: the chunks that
	// were adapted go into the front matter as they were fetched.
	if fresh && len(provs) > 0 {
		canon = withSources(canon, provs)
		if err := os.WriteFile(canonPath, []byte(canon), 0o644); err != nil {
			return err
		}
	}
	var headings []string
	for _, line := range strings.Split(canon, "\n") {
		if strings.HasPrefix(line, "## ") {
			headings = append(headings, line)
		}
	}
	headingList := strings.Join(headings, "\n")

	questionsPath := filepath.Join(dir, "questions.yaml")
	questions, err := write(questionsPath, "questions_yaml",
		"full questions.yaml content",
		"Here is the canon.md you just wrote:\n\n"+canon+
			"\n\nNow write questions.yaml for it, and nothing else.")
	if err != nil {
		return err
	}
	// An MCQ without `check: choice` is not graded as one; the model
	// forgets it often enough that the pipeline adds it.
	if fixed := withChoiceChecks(questions); fixed != questions {
		if err := os.WriteFile(questionsPath, []byte(fixed), 0o644); err != nil {
			return err
		}
	}
	if _, err := write(filepath.Join(dir, "misconceptions.yaml"), "misconceptions_yaml",
		"unit-local misconceptions.yaml content",
		"Here are this unit's headings:\n"+headingList+
			"\n\nWrite the unit-local misconceptions.yaml, and nothing else."); err != nil {
		return err
	}
	for _, d := range depths {
		// The server swaps sections by heading, so a drifted heading silently
		// does nothing. Hand the model the exact list rather than hoping.
		extra := fmt.Sprintf("Here is canon.md:\n\n%s\n\nWrite the %s depth variant: %s. "+
			"It must use these exact `## ` headings, in this order - the server swaps "+
			"sections by heading, so a drifted heading silently does nothing:\n%s",
			canon, d.name, d.what, headingList)
		if _, err := write(filepath.Join(dir, "depths", d.name), "content", d.what, extra); err != nil {
			return err
		}
	}
	return nil
}

// withChoiceChecks adds `check: choice` to every `kind: mcq` item that has
// no check line of its own.
func withChoiceChecks(questions string) string {
	lines := strings.Split(questions, "\n")
	var out []string
	for i, l := range lines {
		out = append(out, l)
		trimmed := strings.TrimSpace(l)
		if trimmed != "kind: mcq" {
			continue
		}
		indent := l[:len(l)-len(strings.TrimLeft(l, " "))]
		has := false
		for j := i + 1; j < len(lines); j++ {
			next := lines[j]
			if strings.TrimSpace(next) == "" || strings.HasPrefix(strings.TrimSpace(next), "- id:") {
				break
			}
			if strings.HasPrefix(next, indent+"check:") {
				has = true
				break
			}
		}
		if !has {
			out = append(out, indent+"check: choice")
		}
	}
	return strings.Join(out, "\n")
}
