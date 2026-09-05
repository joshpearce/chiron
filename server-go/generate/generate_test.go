package generate

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/mjbraun/chiron/server/llm"
	"github.com/mjbraun/chiron/server/sources"
)

// A chain that answers each role with a canned payload and keeps what it
// was asked, so the prompts can be checked.
type stubChain struct {
	payloads map[string]map[string]any
	asked    map[string][]string
	systems  map[string][]string
}

func (c *stubChain) Structured(role, system, user string, schema map[string]any, name string, out any) error {
	if c.asked == nil {
		c.asked = map[string][]string{}
		c.systems = map[string][]string{}
	}
	c.asked[role] = append(c.asked[role], user)
	c.systems[role] = append(c.systems[role], system)
	payload := c.payloads[role]
	if role == "author" {
		// The author is asked for one file at a time; answer with the key it wants.
		for key := range schema["properties"].(map[string]any) {
			payload = map[string]any{key: c.payloads["author"]["body"]}
		}
	}
	data, _ := json.Marshal(payload)
	return json.Unmarshal(data, out)
}

func (stubChain) Status() llm.Status { return llm.Status{Connected: true} }

func TestNamesInABriefAreFound(t *testing.T) {
	got := NamedInBrief(`A short book on Bayesian statistics for an engineer, starting from "Think Bayes" by Allen Downey and interleaving content from MIT 18.05 Introduction to Probability and Statistics, with worked examples.`)
	want := []string{"Think Bayes", "MIT 18.05 Introduction to Probability and Statistics"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("got %v", got)
	}
	if got := NamedInBrief("Teach me hashing from first principles."); len(got) != 0 {
		t.Fatalf("got %v from a brief that names nothing", got)
	}
}

func TestFrontMatterGainsTheSources(t *testing.T) {
	provs := []sources.Provenance{{Source: "think-bayes", Title: "Think Bayes 2e", Authors: "Allen B. Downey", Licence: "CC BY-NC-SA 4.0", LicenceURL: "u", URL: "x", Fetched: "2026-09-05"}}
	canon := "---\nunit: u1\ntitle: Priors\n---\n\nProse.\n"
	got := withSources(canon, provs)
	if !strings.HasPrefix(got, "---\nunit: u1\ntitle: Priors\nsources:\n") || !strings.Contains(got, "  title: Think Bayes 2e\n") || !strings.HasSuffix(got, "---\n\nProse.\n") {
		t.Fatalf("got:\n%s", got)
	}
	// The author's own sources list gives way to the pipeline's.
	doubled := withSources("---\nunit: u1\nsources:\n  - title: The author's guess\n    licence: none\ntitle: Priors\n---\n\nProse.\n", provs)
	if strings.Count(doubled, "\nsources:\n") != 1 || strings.Contains(doubled, "author's guess") || !strings.Contains(doubled, "title: Priors\n") {
		t.Fatalf("doubled:\n%s", doubled)
	}
	bare := withSources("Prose only.\n", provs)
	if !strings.HasPrefix(bare, "---\nsources:\n") || !strings.HasSuffix(bare, "---\nProse only.\n") {
		t.Fatalf("bare:\n%s", bare)
	}
}

// The whole named-sources path: the brief names a book the index has;
// the planner sees its chapters and assigns them; the author gets the
// fetched text as material and the unit records where it came from.
func TestABookIsPlannedAndAuthoredFromANamedSource(t *testing.T) {
	host := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/repos/AllenDowney/ThinkBayes2/contents/notebooks":
			w.Write([]byte(`[{"name":"chap01.ipynb","type":"file"},{"name":"chap02.ipynb","type":"file"}]`))
		case "/AllenDowney/ThinkBayes2/master/notebooks/chap02.ipynb":
			w.Write([]byte(`{"cells":[{"cell_type":"markdown","source":"# Bayes's Theorem\n\nThe cookie problem, in Downey's words.\n"}]}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer host.Close()

	dir := t.TempDir()
	spec := filepath.Join(dir, "authoring-spec.md")
	os.WriteFile(spec, []byte("# Contract\n\nWrite canon.md with front matter.\n"), 0o644)
	idx := &sources.Index{Sources: []sources.Source{{
		ID: "think-bayes", Title: "Think Bayes 2e", Authors: []string{"Allen B. Downey"}, Subjects: []string{"bayesian"},
		Verdict: sources.Adaptable, Licence: sources.Licence{Name: "CC BY-NC-SA 4.0", URL: "lic"}, Voice: "Conversational.",
		Fetch: sources.Recipe{Kind: "github", Base: host.URL, Repo: "AllenDowney/ThinkBayes2", Ref: "master", Path: "notebooks/{locator}", Dir: "notebooks"},
	}}}
	client := sources.NewClient(filepath.Join(dir, "cache"))
	client.Pause = 0
	chain := &stubChain{payloads: map[string]map[string]any{
		"planner": {
			"title": "Bayes for Engineers", "learner_profile": "an engineer",
			"units": []map[string]any{
				{"id": "u0", "slug": "bayes-theorem", "title": "Bayes's Theorem", "minutes": 20, "prereqs": []string{}, "concepts": []map[string]any{{"id": "c-bayes", "name": "Bayes's theorem"}}, "notes": "",
					"sources": []map[string]any{{"source": "think-bayes", "locator": "chap02.ipynb", "role": "spine"}, {"source": "think-bayes", "locator": "nope.ipynb", "role": "spine"}}},
				{"id": "u1", "slug": "from-scratch", "title": "Something Else", "minutes": 20, "prereqs": []string{"u0"}, "concepts": []map[string]any{{"id": "c-x", "name": "x"}}, "notes": "", "sources": []map[string]any{}},
			},
			"misconceptions": []map[string]any{},
		},
		"author": {"body": "---\nunit: u0\ntitle: Bayes's Theorem\n---\n\n## The cookie problem\n\nIn Downey's words, adapted.\n"},
	}}
	g := &Generator{Chain: chain, SpecPath: spec, OutDir: filepath.Join(dir, "corpus-bayes"), Index: idx, Fetch: client, Workers: 1}

	total, err := g.Plan("A short book on Bayesian statistics, starting from Think Bayes.", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 {
		t.Fatalf("total = %d", total)
	}
	planPrompt := chain.asked["planner"][0]
	if !strings.Contains(planPrompt, "[spine] think-bayes") || !strings.Contains(planPrompt, "chap02.ipynb | Bayes's Theorem") {
		t.Fatalf("the planner did not see the spine's sections:\n%s", planPrompt)
	}
	if !strings.Contains(chain.systems["planner"][0], "follow the spine's order") {
		t.Fatal("the planner was not told to follow the spine")
	}
	var syllabus struct {
		Units []unitPlan `yaml:"units"`
	}
	raw, _ := os.ReadFile(filepath.Join(g.OutDir, "syllabus.yaml"))
	if err := yaml.Unmarshal(raw, &syllabus); err != nil {
		t.Fatal(err)
	}
	if len(syllabus.Units[0].Sources) != 1 || syllabus.Units[0].Sources[0].Locator != "chap02.ipynb" {
		t.Fatalf("a made-up locator should have been dropped: %+v", syllabus.Units[0].Sources)
	}
	if _, err := os.Stat(filepath.Join(g.OutDir, "sources.yaml")); err != nil {
		t.Fatal("sources.yaml was not written")
	}

	if failures, err := g.Units([]string{"u0"}, nil); err != nil || failures != 0 {
		t.Fatalf("units: failures=%d err=%v", failures, err)
	}
	authorPrompt := chain.asked["author"][0]
	if !strings.Contains(authorPrompt, "MATERIAL TO ADAPT") || !strings.Contains(authorPrompt, "The cookie problem, in Downey's words.") || !strings.Contains(authorPrompt, "=== spine: Think Bayes 2e") {
		t.Fatalf("the author did not get the material:\n%s", authorPrompt)
	}
	for _, later := range chain.asked["author"][1:] {
		if strings.Contains(later, "MATERIAL TO ADAPT") {
			t.Fatal("only the canon call carries the material")
		}
	}
	unitDir := filepath.Join(g.OutDir, "units", "u0-bayes-theorem")
	canon, _ := os.ReadFile(filepath.Join(unitDir, "canon.md"))
	if !strings.Contains(string(canon), "sources:\n") || !strings.Contains(string(canon), "title: Think Bayes 2e") || !strings.Contains(string(canon), "licence: CC BY-NC-SA 4.0") {
		t.Fatalf("canon front matter lacks the sources:\n%s", canon)
	}
	if _, err := os.Stat(filepath.Join(unitDir, "sources", "01-think-bayes-chap02-ipynb.provenance.yaml")); err != nil {
		t.Fatal("the chunk's provenance was not written beside the unit")
	}

	// A unit with no sources is authored from the brief, with no material block.
	chain.asked["author"] = nil
	if failures, err := g.Units([]string{"u1"}, nil); err != nil || failures != 0 {
		t.Fatalf("units: failures=%d err=%v", failures, err)
	}
	if strings.Contains(chain.asked["author"][0], "MATERIAL TO ADAPT") {
		t.Fatal("a sourceless unit got material")
	}
}

// A rerun keeps the syllabus it has and only writes what is missing.
func TestARerunKeepsTheSyllabus(t *testing.T) {
	dir := t.TempDir()
	spec := filepath.Join(dir, "spec.md")
	os.WriteFile(spec, []byte("contract"), 0o644)
	out := filepath.Join(dir, "out")
	os.MkdirAll(out, 0o755)
	os.WriteFile(filepath.Join(out, "syllabus.yaml"), []byte("units:\n  - id: u0\n    slug: a\n    title: A\n  - id: u1\n    slug: b\n    title: B\n"), 0o644)
	chain := &stubChain{payloads: map[string]map[string]any{}}
	g := &Generator{Chain: chain, SpecPath: spec, OutDir: out}
	total, err := g.Plan("anything", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 || len(chain.asked["planner"]) != 0 {
		t.Fatalf("total=%d planner calls=%d", total, len(chain.asked["planner"]))
	}
}

func TestAnUnknownNameIsRecordedNotFatal(t *testing.T) {
	dir := t.TempDir()
	spec := filepath.Join(dir, "spec.md")
	os.WriteFile(spec, []byte("contract"), 0o644)
	chain := &stubChain{payloads: map[string]map[string]any{"planner": {
		"title": "T", "learner_profile": "p", "units": []map[string]any{{"id": "u0", "slug": "a", "title": "A", "minutes": 10, "prereqs": []string{}, "concepts": []map[string]any{}, "notes": "", "sources": []map[string]any{}}}, "misconceptions": []map[string]any{}}}}
	g := &Generator{Chain: chain, SpecPath: spec, OutDir: filepath.Join(dir, "out"), Index: &sources.Index{}, Fetch: sources.NewClient("")}
	if _, err := g.Plan("Starting from The Book Nobody Indexed.", "", nil); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(g.OutDir, "sources.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "unresolved:") || !strings.Contains(string(raw), "- Book Nobody Indexed") {
		t.Fatalf("sources.yaml:\n%s", raw)
	}
}

func TestBareMCQsGetTheirCheck(t *testing.T) {
	in := "items:\n  - id: q1\n    kind: mcq\n    prompt: a\n    options:\n      - text: x\n        correct: true\n\n  - id: q2\n    kind: mcq\n    check: choice\n    prompt: b\n\n  - id: q3\n    kind: constructed\n    check: llm\n"
	got := withChoiceChecks(in)
	if strings.Count(got, "check: choice") != 2 || !strings.Contains(got, "kind: mcq\n    check: choice\n    prompt: a") {
		t.Fatalf("got:\n%s", got)
	}
	if withChoiceChecks(got) != got {
		t.Fatal("not idempotent")
	}
}
