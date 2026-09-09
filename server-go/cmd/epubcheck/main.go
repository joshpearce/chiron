// epubcheck says what an EPUB looks like to Chiron before it is imported:
// the title the book gives itself, and the chapters of its spine with the
// length of each. What the reader will see in the contents is these, cut
// at their own headings where they are too long to read in one scroll.
//
//	go run ./cmd/epubcheck <book.epub>
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/mjbraun/chiron/server/sources"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: epubcheck <book.epub>")
		os.Exit(2)
	}
	title, chapters, err := sources.EPUBBook(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	total := 0
	fmt.Printf("%s\n%d chapters\n\n", title, len(chapters))
	for i, c := range chapters {
		n := len(strings.Fields(c.Markdown))
		total += n
		fmt.Printf("  %3d %8d words  %s\n", i+1, n, c.Title)
	}
	fmt.Printf("\n%d words in all\n", total)
}
