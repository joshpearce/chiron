package httpapi

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/mjbraun/chiron/server/generate"
	"github.com/mjbraun/chiron/server/sources"
)

// generate runs plan then authoring on a worker goroutine, reporting progress
// through the job record. It outlives the request that started it. With
// planOnly it stops at the syllabus, done, so the plan can be read before
// a second request authors from it.
func (s *Server) generate(slug, title, brief string, named []string, planOnly bool) {
	parent := filepath.Dir(s.root)
	outDir := filepath.Join(parent, "corpus-"+slug)

	fail := func(msg string) {
		s.updateJob(slug, func(j *Job) {
			j.Stage, j.Done, j.Error = "failed", true, msg
		})
		log.Printf("teach %s: %s", slug, msg)
	}

	g := &generate.Generator{
		Chain:    s.chain,
		SpecPath: filepath.Join(parent, "corpus", "authoring-spec.md"),
		OutDir:   outDir,
		Workers:  authorWorkers(),
	}
	// The open sources the book may be built from. A missing or broken
	// index is logged, and the book is written from the brief alone.
	if idx, err := sources.LoadIndex(filepath.Join(parent, "corpus", "sources", "index.yaml")); err != nil {
		log.Printf("teach %s: source index: %v", slug, err)
	} else if !driveEnabled() {
		g.Index = idx
		g.Fetch = sources.NewClient(filepath.Join(filepath.Dir(s.activePath), "sources"))
	}

	s.updateJob(slug, func(j *Job) { j.Stage = "planning" })
	total, err := g.Plan(brief, title, named)
	if err != nil {
		fail("planning failed: " + err.Error())
		return
	}
	if planOnly {
		s.updateJob(slug, func(j *Job) { j.Stage, j.UnitsTotal, j.Done = "planned", total, true })
		log.Printf("teach %s: planned %d units, waiting to be told to author", slug, total)
		return
	}
	s.updateJob(slug, func(j *Job) { j.Stage, j.UnitsTotal = "authoring", total })

	failures, err := g.Units(nil, func(done, of int) {
		s.updateJob(slug, func(j *Job) { j.UnitsDone = done })
	})
	if err != nil {
		fail("authoring failed: " + err.Error())
		return
	}
	if failures > 0 {
		fail(fmt.Sprintf("authoring failed for %d of %d units", failures, total))
		return
	}

	// Registering is what makes the subject readable, so it happens only after
	// every unit is on disk.
	for _, id := range s.Discover() {
		if id == slug {
			s.updateJob(slug, func(j *Job) { j.Stage, j.Done = "ready", true })
			return
		}
	}
	fail("generated corpus did not load")
}

// authorWorkers is how many units are authored at once: four, or
// CHIRON_AUTHOR_WORKERS, so a run can be narrowed to one call at a time
// when the model calls are being watched.
func authorWorkers() int {
	if n, err := strconv.Atoi(os.Getenv("CHIRON_AUTHOR_WORKERS")); err == nil && n > 0 {
		return n
	}
	return 4
}
