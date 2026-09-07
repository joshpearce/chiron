// Command corpus-calibrate gives a generated book the placement unit it
// lacks: a screener and a banded series, authored from the book's own
// syllabus and misconception bank, put first in the syllabus.
//
//	corpus-calibrate [-model M] [-spec F] <book dir>
//
// The model runs through the claude CLI. Lint the book afterwards: the
// linter holds the bands and the calibration sets to the contract.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mjbraun/chiron/server/generate"
	"github.com/mjbraun/chiron/server/llm"
)

func main() {
	model := flag.String("model", "", "model for every call (default: the author tier)")
	spec := flag.String("spec", "", "authoring spec (default: <book>/../corpus/authoring-spec.md)")
	flag.Parse()
	if flag.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: corpus-calibrate [-model M] [-spec F] <book dir>")
		os.Exit(2)
	}
	dir := filepath.Clean(flag.Arg(0))
	s := *spec
	if s == "" {
		s = filepath.Join(filepath.Dir(dir), "corpus", "authoring-spec.md")
	}
	g := &generate.Generator{Chain: llm.New(llm.FactoryConfig{Provider: "claude-cli", ClaudeCLIModel: *model}), SpecPath: s, OutDir: dir, Workers: 1}
	id, err := g.Calibrate()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", dir, err)
		os.Exit(1)
	}
	fmt.Printf("%s: placement unit %s written and put first\n", dir, id)
}
