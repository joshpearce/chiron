package httpapi

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/mjbraun/chiron/server/generate"
	"github.com/mjbraun/chiron/server/sources"
)

// generate runs plan then authoring on a worker goroutine, reporting progress
// through the job record. It outlives the request that started it.
func (s *Server) generate(slug, title, brief string, named []string) {
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
		Workers:  4,
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
