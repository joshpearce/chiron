package sources

import (
	"context"
	"strings"
	"testing"
)

func TestExercisesFromMySTDirectives(t *testing.T) {
	md := "# Lecture\n\nProse.\n\n```{exercise}\n:label: ex1\n\nCompute the mean of a fair die.\n```\n\n" +
		"```{solution-start} ex1\n:class: dropdown\n```\n\nThe mean is 3.5.\n\n```{solution-end}\n```\n\n" +
		"```{exercise-start}\n:label: ex2\n```\n\nWhat is $P(A)$ if\n\n$$P(A)=1/4$$\n\n```{exercise-end}\n```\n"
	got := Exercises(md)
	if len(got) != 2 {
		t.Fatalf("got %d exercises: %+v", len(got), got)
	}
	if got[0].Prompt != "Compute the mean of a fair die." || got[0].Solution != "The mean is 3.5." {
		t.Fatalf("first: %+v", got[0])
	}
	if !strings.Contains(got[1].Prompt, "$$P(A)=1/4$$") || got[1].Solution != "" {
		t.Fatalf("second: %+v", got[1])
	}
}

func TestExercisesFromBoldMarkers(t *testing.T) {
	md := "## Exercises\n\n**Exercise:** Two coins in a box. One is fair.\nWhat is the probability?\n\n```python\n# Solution goes here\n```\n\n" +
		"**Exercise 2:** A second question.\n\n```python\nprior = 0.5\nposterior = prior * 2\n```\n\nSo the answer is 1.\n\n## Summary\n\nDone.\n"
	got := Exercises(md)
	if len(got) != 2 {
		t.Fatalf("got %d: %+v", len(got), got)
	}
	if got[0].Prompt != "Two coins in a box. One is fair.\nWhat is the probability?" || got[0].Solution != "" {
		t.Fatalf("first: %+v", got[0])
	}
	if got[1].Prompt != "A second question." || !strings.Contains(got[1].Solution, "posterior = prior * 2") || !strings.Contains(got[1].Solution, "So the answer is 1.") {
		t.Fatalf("second: %+v", got[1])
	}
}

func TestExercisesFromNumberedHeadings(t *testing.T) {
	md := "# Problems\n\n### Problem 1\n\nFind $x$ if $2x = 6$.\n\n#### Solution\n\n$x = 3$.\n\n### Problem 2\n\nName the distribution.\n\n### Answer\n\nBinomial.\n\n## Notes\n\nnothing\n"
	got := Exercises(md)
	if len(got) != 2 || got[0].Prompt != "Find $x$ if $2x = 6$." || got[0].Solution != "$x = 3$." || got[1].Solution != "Binomial." {
		t.Fatalf("got %+v", got)
	}
	if got := Exercises("# Chapter\n\nNo exercises here.\n"); len(got) != 0 {
		t.Fatalf("found exercises in prose: %+v", got)
	}
}

// Downey keeps the exercises in the chapter notebook and the solutions in
// a companion under soln/; the recipe names the companion and the client
// pairs them.
func TestClientPairsExercisesWithTheSolutionsCompanion(t *testing.T) {
	srv := fakeHost(t, map[string]string{
		"/AllenDowney/ThinkBayes2/master/notebooks/chap02.ipynb": `{"cells":[{"cell_type":"markdown","source":"## Exercises\n\n**Exercise:** First question.\n"},{"cell_type":"code","source":"# Solution goes here\n"},{"cell_type":"markdown","source":"**Exercise:** Second question.\n"},{"cell_type":"code","source":"# Solution goes here\n"}]}`,
		"/AllenDowney/ThinkBayes2/master/soln/chap02.ipynb":      `{"cells":[{"cell_type":"markdown","source":"## Exercises\n\n**Exercise:** First question.\n"},{"cell_type":"code","source":"# Solution\n\nanswer = 0.5\n"},{"cell_type":"markdown","source":"**Exercise:** Second question.\n"},{"cell_type":"code","source":"# Solution\n\nanswer = 0.25\n"}]}`,
	})
	s := &Source{ID: "think-bayes", Verdict: Adaptable,
		Fetch: Recipe{Kind: "github", Base: srv.URL, Repo: "AllenDowney/ThinkBayes2", Ref: "master", Path: "notebooks/{locator}", Solutions: "soln/{locator}"}}
	got, err := client(t).Exercises(context.Background(), s, "chap02.ipynb")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Prompt != "First question." || !strings.Contains(got[0].Solution, "answer = 0.5") || !strings.Contains(got[1].Solution, "answer = 0.25") {
		t.Fatalf("got %+v", got)
	}
}
