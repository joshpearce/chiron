// chiron-dev-agent drains the app's change requests on the sprite
// (SPRITE-DEV-PLAN.md phase G): a sprite-env service that takes one
// request at a time and sees it through to a commit, a deploy, a build.
//
//	chiron-dev-agent -repo ~/src/chiron -served ~/chiron [-once]
package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/mjbraun/chiron/server/devagent"
	"github.com/mjbraun/chiron/server/devreq"
)

func main() {
	home, _ := os.UserHomeDir()
	repo := flag.String("repo", filepath.Join(home, "src", "chiron"), "the checkout main lives in")
	work := flag.String("work", filepath.Join(home, "src", "work"), "where request worktrees go")
	served := flag.String("served", filepath.Join(home, "chiron"), "the deployed tree (state/requests, builds)")
	model := flag.String("model", "claude-fable-5-1", "the model Claude Code works with")
	every := flag.Duration("every", 10*time.Second, "how often to look for a request")
	once := flag.Bool("once", false, "take one request, or none, and exit")
	spriteAPI := flag.String("sprite-api", "/.sprite/api.sock", "the sprite runtime's API, which holds the sprite awake")
	flag.Parse()

	a := &devagent.Agent{
		Repo: *repo, Work: *work, Served: *served,
		Store: devreq.Open(filepath.Join(*served, "state", "requests")),
		Model: *model, Exec: devagent.Shell{},
	}
	// On a sprite the runtime's API holds it awake while a request runs.
	if _, err := os.Stat(*spriteAPI); err == nil {
		a.Awake = devagent.SpriteTasks{Socket: *spriteAPI}
	}
	os.MkdirAll(*work, 0o755)
	log.Printf("chiron-dev-agent: %s, requests in %s, %s", *repo, a.Store.Dir(), *model)
	for {
		took, err := a.Once()
		if err != nil {
			log.Printf("requests: %v", err)
		}
		if *once {
			return
		}
		if !took {
			time.Sleep(*every)
		}
	}
}
