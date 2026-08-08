package httpapi

import (
	"testing"

	"github.com/mjbraun/chiron/server/corpus"
)

// The chapter directly after a calibration unit skips its pretest: the
// series just measured that ground, and two question blocks back to back
// read as a bug, not pedagogy.
func TestFollowsCalibration(t *testing.T) {
	c := &corpus.Corpus{
		Syllabus: corpus.Syllabus{Units: []corpus.UnitSpec{
			{ID: "u0"}, {ID: "u1"}, {ID: "u2"},
		}},
		Units: map[string]*corpus.Unit{
			"u0": {ID: "u0", Front: map[string]any{"calibration": true}},
			"u1": {ID: "u1"},
			"u2": {ID: "u2"},
		},
	}
	for id, want := range map[string]bool{"u0": false, "u1": true, "u2": false, "zz": false} {
		if got := followsCalibration(c, id); got != want {
			t.Errorf("followsCalibration(%s) = %v, want %v", id, got, want)
		}
	}
}
