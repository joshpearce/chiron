// Command corpus-lint runs the mechanical half of the corpus verify pass.
//
//	corpus-lint [corpus_dir]
//
// Exit code is the number of errors; warnings do not fail.
package main

import (
	"fmt"
	"os"

	"github.com/mjbraun/chiron/server/corpus"
	"github.com/mjbraun/chiron/server/lint"
)

func main() {
	dir := "corpus"
	if len(os.Args) > 1 {
		dir = os.Args[1]
	}
	c, err := corpus.Load(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load %s: %v\n", dir, err)
		os.Exit(1)
	}
	r := lint.Run(c)

	fmt.Printf("linted %d units, %d misconceptions\n", len(c.Units), len(c.Misconceptions))
	for _, w := range r.Warnings {
		fmt.Printf("  warn  %s\n", w)
	}
	for _, e := range r.Errors {
		fmt.Printf("  ERROR %s\n", e)
	}
	fmt.Printf("\n%d errors, %d warnings\n", len(r.Errors), len(r.Warnings))

	// Cap the exit code: shells truncate above 255, and "0" would read as clean.
	code := len(r.Errors)
	if code > 100 {
		code = 100
	}
	os.Exit(code)
}
