package generate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Calibrate gives a book that opens straight into its first chapter the
// placement unit every generated book now opens with: a screener and a
// banded series across what the book leans on, written from the syllabus
// and the misconception bank, put first in the syllabus with the old
// first unit made to follow it. The unit's id is cal, since u0 is taken.
func (g *Generator) Calibrate() (string, error) {
	raw, err := os.ReadFile(filepath.Join(g.OutDir, "syllabus.yaml"))
	if err != nil {
		return "", err
	}
	var syllabus map[string]any
	if err := yaml.Unmarshal(raw, &syllabus); err != nil {
		return "", err
	}
	var typed struct {
		Title   string `yaml:"title"`
		Learner struct {
			Profile string `yaml:"profile"`
		} `yaml:"learner"`
		Units []unitPlan `yaml:"units"`
	}
	if err := yaml.Unmarshal(raw, &typed); err != nil {
		return "", err
	}
	if len(typed.Units) == 0 {
		return "", fmt.Errorf("the syllabus has no units")
	}
	for _, u := range typed.Units {
		if u.Calibration || u.ID == "cal" {
			return "", fmt.Errorf("the book already has a placement unit (%s)", u.ID)
		}
	}
	bank, err := os.ReadFile(filepath.Join(g.OutDir, "misconception-bank.yaml"))
	if err != nil {
		return "", err
	}
	spec, err := os.ReadFile(g.SpecPath)
	if err != nil {
		return "", err
	}

	cal := unitPlan{ID: "cal", Slug: "placement", Title: "Placement: where you are", Minutes: 15, Prereqs: []string{}, Calibration: true}
	var leans []string
	for _, u := range typed.Units {
		for _, c := range u.Concepts {
			cal.Concepts = append(cal.Concepts, c)
			leans = append(leans, c.Name)
		}
	}
	cal.Notes = fmt.Sprintf("The calibration unit of %q (`calibration: true` in canon front matter): "+
		"canon.md is the front matter and one short paragraph framing the placement, with no `## ` sections and no beats. "+
		"questions.yaml has an empty pretest, a check bank of at least fifteen items spanning what the book leans on (%s), "+
		"every item `kind: mcq` with `check: choice` or constructed with `check: numeric(tol)` or `exact`, every item with `band: 1..5` "+
		"and at least three items in every band, then the five-option `screener` and `calibration_sets` for levels 1 to 5, "+
		"each a window: at least three items from the level's own band, one from the band below, one from the band above, ordered easy to hard.",
		typed.Title, strings.Join(leans, "; "))

	if err := g.authorUnit(cal, typed.Learner.Profile, string(bank), string(spec)); err != nil {
		return "", err
	}

	// The placement unit goes first, and the old first unit follows it.
	units, _ := syllabus["units"].([]any)
	if len(units) > 0 {
		if first, ok := units[0].(map[string]any); ok {
			pre, _ := first["prereqs"].([]any)
			if len(pre) == 0 {
				first["prereqs"] = []any{"cal"}
			}
		}
	}
	entry := map[string]any{
		"id": cal.ID, "slug": cal.Slug, "title": cal.Title, "minutes": cal.Minutes, "prereqs": []any{},
		"concepts": []any{}, "notes": "Placement: a screener and a banded series across what the book leans on.", "calibration": true,
	}
	syllabus["units"] = append([]any{entry}, units...)
	return cal.ID, writeYAML(filepath.Join(g.OutDir, "syllabus.yaml"), syllabus)
}
