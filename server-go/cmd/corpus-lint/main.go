// Command corpus-lint runs the mechanical half of the corpus verify pass.
//
//	corpus-lint [-repair] [corpus_dir]
//
// -repair first rewrites any unit YAML that does not parse because prose
// was written as plain scalars (see generate.RepairYAMLProse).
//
// Exit code is the number of errors; warnings do not fail.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mjbraun/chiron/server/corpus"
	"github.com/mjbraun/chiron/server/generate"
	"github.com/mjbraun/chiron/server/lint"
)

func main() {
	repair := flag.Bool("repair", false, "rewrite unit YAML that prose broke, then lint")
	flag.Parse()
	dir := "corpus"
	if flag.NArg() > 0 {
		dir = flag.Arg(0)
	}
	if *repair {
		files, _ := filepath.Glob(filepath.Join(dir, "units", "*", "*.yaml"))
		for _, f := range files {
			raw, err := os.ReadFile(f)
			if err != nil {
				continue
			}
			if fixed := generate.RepairYAMLProse(string(raw)); fixed != string(raw) {
				if err := os.WriteFile(f, []byte(fixed), 0o644); err != nil {
					fmt.Fprintf(os.Stderr, "repair %s: %v\n", f, err)
					os.Exit(1)
				}
				fmt.Printf("repaired %s\n", f)
			}
		}
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
