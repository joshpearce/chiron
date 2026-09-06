// Command corpus-taps rewrites the prose-answered items of a unit's
// question bank as items answered by a tap, then reports what changed.
//
//	corpus-taps [-model M] [-batch N] <unit dir> ...
//
// The corpus's misconception bank and authoring spec are read from the
// unit's corpus root (two directories up) unless -bank and -spec say
// otherwise. The model runs through the claude CLI.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mjbraun/chiron/server/llm"
	"github.com/mjbraun/chiron/server/taps"
)

func main() {
	model := flag.String("model", "", "model for every call (default: the author tier)")
	batch := flag.Int("batch", 6, "items per model call")
	bank := flag.String("bank", "", "misconception bank (default: <corpus>/misconception-bank.yaml)")
	spec := flag.String("spec", "", "authoring spec (default: <corpus>/authoring-spec.md)")
	flag.Parse()
	if flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "usage: corpus-taps [-model M] [-batch N] <unit dir> ...")
		os.Exit(2)
	}
	chain := llm.New(llm.FactoryConfig{Provider: "claude-cli", ClaudeCLIModel: *model})
	failed := 0
	for _, dir := range flag.Args() {
		root := filepath.Dir(filepath.Dir(dir))
		b, s := *bank, *spec
		if b == "" {
			b = filepath.Join(root, "misconception-bank.yaml")
		}
		if s == "" {
			s = filepath.Join(root, "authoring-spec.md")
		}
		ids, err := taps.Rewrite(chain, dir, b, s, *batch)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", dir, err)
			failed++
			continue
		}
		if len(ids) == 0 {
			fmt.Printf("%s: nothing answered in prose\n", dir)
			continue
		}
		fmt.Printf("%s: rewrote %d items: %s\n", dir, len(ids), strings.Join(ids, " "))
	}
	if failed > 0 {
		os.Exit(1)
	}
}
