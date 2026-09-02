package httpapi

import (
	"strings"
	"testing"

	"github.com/mjbraun/chiron/server/roles"
)

// The planner used to get "Calibration u0: 40% overall" - which cannot tell a
// novice who cleared the arithmetic floor from one who cleared real dot
// products. The summary has to carry the claim and the per-band record.
func TestCalibrationSummaryReportsClaimAndBands(t *testing.T) {
	s := newServer(t, "")
	sub, _ := s.subject("ai")
	unit := sub.Corpus.Units["u0"]
	results := []Result{
		{ItemID: "u0-n1", Grade: roles.Grade{Verdict: "pass"}},
		{ItemID: "u0-n2", Grade: roles.Grade{Verdict: "pass"}},
		{ItemID: "u0-n5", Grade: roles.Grade{Verdict: "fail"}},
		{ItemID: "u0-q1", Grade: roles.Grade{Verdict: "fail"}},
	}
	got := calibrationSummary(unit, 1, 0.5, results)
	for _, want := range []string{
		"self-rated level 1 of 5 (Absolute novice)",
		"50% overall",
		"band 1: 2/2", "band 2: 0/1", "band 3: 0/1",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("summary lacks %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "band 4") {
		t.Errorf("bands that were never asked must not appear:\n%s", got)
	}
	if got := calibrationSummary(unit, 0, 0.5, nil); strings.Contains(got, "self-rated") {
		t.Errorf("no rating recorded, yet summary claims one:\n%s", got)
	}
}
