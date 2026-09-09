package sources

import (
	"strings"
	"testing"
)

// A book's pictures come out of it by the paths its chapters use, which
// are relative to the document that carries them.
func TestEPUBAssetsFollowTheChaptersOwnPaths(t *testing.T) {
	path := writeEPUB(t, map[string]string{
		"mimetype":               "application/epub+zip",
		"META-INF/container.xml": epubContainer,
		"OEBPS/package.opf":      epubPackage,
		"OEBPS/nav.xhtml":        epubNav,
		"OEBPS/cover.xhtml":      `<html><body><p>cover</p></body></html>`,
		"OEBPS/text/ch01.xhtml": `<html><body><h1>What Lending Is</h1>` +
			`<p><img src="../images/fig1.png" alt="A chart"/></p></body></html>`,
		"OEBPS/text/ch02.xhtml": `<html><body><h1>Who Lends, and Why</h1>` +
			`<p><img src="../images/fig2.png" alt="Another"/></p></body></html>`,
		"OEBPS/images/fig1.png": "not really a png, but bytes",
		"OEBPS/images/fig2.png": "more bytes",
	})
	_, chapters, err := EPUBBook(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(chapters[1].Markdown, "![A chart](../images/fig1.png)") {
		t.Fatalf("the chapter should carry its picture:\n%s", chapters[1].Markdown)
	}
	assets, names, err := EPUBAssets(path, chapters)
	if err != nil {
		t.Fatal(err)
	}
	if len(assets) != 2 {
		t.Fatalf("assets: %+v", assets)
	}
	if names["../images/fig1.png"] != "fig1.png" || names["../images/fig2.png"] != "fig2.png" {
		t.Errorf("names: %+v", names)
	}
	if string(assets[0].Data) != "not really a png, but bytes" {
		t.Errorf("asset bytes: %q", assets[0].Data)
	}
}
