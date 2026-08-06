// Package pages renders chapter HTML into e-ink page images.
//
// The reMarkable client cannot render HTML or LaTeX; it displays page images
// and captures ink. This package turns a chapter (the same HTML fragment the
// iPad renders in a WKWebView) into a stack of PNGs at the Paper Pro's native
// portrait resolution, via headless Chrome print-to-PDF (which runs KaTeX)
// and pdftoppm.
package pages

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"

	"github.com/mjbraun/chiron/server/corpus"
	"github.com/mjbraun/chiron/server/render"
)

// reMarkable Paper Pro portrait, native pixels (11.8", 229 ppi).
const (
	PageW = 1620
	PageH = 2160
)

// Bump when the wrapper HTML/CSS changes so cached renders invalidate.
const styleVersion = "v1"

type Renderer struct {
	// ChromePath overrides Chrome discovery; empty means look in the
	// usual places.
	ChromePath string
	// KatexDir holds katex.min.css/js and contrib/auto-render.min.js
	// (the same assets the iPad bundles).
	KatexDir string
	// CacheDir receives one subdirectory per rendered chapter, keyed by
	// content hash.
	CacheDir string
}

type Result struct {
	Dir   string `json:"-"`
	Hash  string `json:"hash"`
	Count int    `json:"count"`
}

func (r Result) PagePath(n int) string {
	return filepath.Join(r.Dir, fmt.Sprintf("page-%03d.png", n))
}

func (r *Renderer) chrome() string {
	if r.ChromePath != "" {
		return r.ChromePath
	}
	candidates := []string{
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		"/usr/bin/chromium",
		"/usr/bin/google-chrome",
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	if p, err := exec.LookPath("chromium"); err == nil {
		return p
	}
	return ""
}

// Render produces the page stack for a chapter, reusing a previous render
// of identical content.
func (r *Renderer) Render(ch *render.Chapter) (Result, error) {
	doc := r.wrap(ch)
	sum := sha256.Sum256([]byte(doc))
	hash := hex.EncodeToString(sum[:8])
	dir := filepath.Join(r.CacheDir, ch.Unit+"-"+hash)
	res := Result{Dir: dir, Hash: hash}

	if meta, err := os.ReadFile(filepath.Join(dir, "meta.json")); err == nil {
		if json.Unmarshal(meta, &res) == nil && res.Count > 0 {
			res.Dir = dir
			return res, nil
		}
	}

	chrome := r.chrome()
	if chrome == "" {
		return res, fmt.Errorf("no chrome/chromium found for page rendering")
	}
	if _, err := exec.LookPath("pdftoppm"); err != nil {
		return res, fmt.Errorf("pdftoppm not installed: %w", err)
	}

	work, err := os.MkdirTemp("", "chiron-pages-*")
	if err != nil {
		return res, err
	}
	defer os.RemoveAll(work)

	htmlPath := filepath.Join(work, "chapter.html")
	if err := os.WriteFile(htmlPath, []byte(doc), 0o644); err != nil {
		return res, err
	}
	pdfPath := filepath.Join(work, "chapter.pdf")

	// virtual-time-budget lets KaTeX finish before printing.
	cmd := exec.Command(chrome,
		"--headless=new", "--disable-gpu", "--no-first-run",
		"--virtual-time-budget=15000",
		"--no-pdf-header-footer",
		"--print-to-pdf="+pdfPath,
		"file://"+htmlPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		return res, fmt.Errorf("chrome print: %w: %s", err, out)
	}

	// Scaling to the device dimensions directly avoids the off-by-one that
	// DPI-based sizing hits when the PDF page size rounds to whole points.
	cmd = exec.Command("pdftoppm", "-png",
		"-scale-to-x", fmt.Sprint(PageW), "-scale-to-y", fmt.Sprint(PageH),
		pdfPath, filepath.Join(work, "page"))
	if out, err := cmd.CombinedOutput(); err != nil {
		return res, fmt.Errorf("pdftoppm: %w: %s", err, out)
	}

	rendered, err := filepath.Glob(filepath.Join(work, "page-*.png"))
	if err != nil || len(rendered) == 0 {
		return res, fmt.Errorf("no pages produced (%v)", err)
	}
	sort.Strings(rendered)

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return res, err
	}
	for i, p := range rendered {
		dst := filepath.Join(dir, fmt.Sprintf("page-%03d.png", i))
		if err := os.Rename(p, dst); err != nil {
			// Rename fails across filesystems (tmp vs cache); fall back
			// to a copy.
			data, rerr := os.ReadFile(p)
			if rerr != nil {
				return res, rerr
			}
			if werr := os.WriteFile(dst, data, 0o644); werr != nil {
				return res, werr
			}
		}
		_ = i
	}
	res.Count = len(rendered)
	meta, _ := json.Marshal(res)
	if err := os.WriteFile(filepath.Join(dir, "meta.json"), meta, 0o644); err != nil {
		return res, err
	}
	return res, nil
}

var beatDiv = regexp.MustCompile(`<div class="beat" data-beat-id="([^"]+)"></div>`)

// injectBeats replaces the client-side beat placeholders with printable
// prompt boxes: the beat's prompt plus writing room, so a page is something
// the learner can work directly.
func injectBeats(doc string, beats []corpus.Beat) string {
	byID := map[string]corpus.Beat{}
	for _, b := range beats {
		byID[b.ID] = b
	}
	return beatDiv.ReplaceAllStringFunc(doc, func(m string) string {
		id := beatDiv.FindStringSubmatch(m)[1]
		b, ok := byID[id]
		if !ok {
			return ""
		}
		label := "Work this before reading on"
		if b.Type == "self-explain" {
			label = "Explain in your own words"
		}
		return `<div class="beat-box"><div class="beat-label">` +
			html.EscapeString(label) + `</div><div class="beat-prompt">` +
			b.Prompt + `</div><div class="beat-ink"></div></div>`
	})
}

func (r *Renderer) wrap(ch *render.Chapter) string {
	katex := "file://" + mustAbs(r.KatexDir)
	body := injectBeats(ch.HTML, ch.Beats)
	return `<!DOCTYPE html><html><head><meta charset="utf-8">
<link rel="stylesheet" href="` + katex + `/katex.min.css">
<script src="` + katex + `/katex.min.js"></script>
<script src="` + katex + `/contrib/auto-render.min.js"></script>
<style>
/* ` + styleVersion + ` */
@page { size: ` + fmt.Sprint(PageW) + `px ` + fmt.Sprint(PageH) + `px; margin: 0; }
html, body { margin: 0; padding: 0; background: white; color: black; }
body {
  font-family: Georgia, serif;
  font-size: 34px;
  line-height: 1.55;
  padding: 100px 110px;
}
h1 { font-size: 56px; line-height: 1.2; margin: 0 0 28px 0; }
h2 { font-size: 44px; line-height: 1.25; margin: 44px 0 18px 0; page-break-after: avoid; }
p { margin: 0 0 22px 0; }
table { border-collapse: collapse; margin: 24px 0; page-break-inside: avoid; }
th, td { border: 2px solid #000; padding: 10px 16px; text-align: left; }
code { font-family: Menlo, monospace; font-size: 0.85em; }
pre { border: 2px solid #000; padding: 16px; overflow: hidden; page-break-inside: avoid; }
blockquote, .planner-note { border-left: 6px solid #000; margin: 24px 0; padding: 8px 0 8px 24px; font-style: italic; }
.katex-display { margin: 26px 0; page-break-inside: avoid; }
.beat-box { border: 3px solid #000; margin: 30px 0; padding: 20px 24px; page-break-inside: avoid; }
.beat-label { font-size: 24px; text-transform: uppercase; letter-spacing: 1px; margin-bottom: 10px; }
.beat-ink { height: 340px; }
</style></head><body>` + body + `
<script>
document.addEventListener("DOMContentLoaded", function() {
  renderMathInElement(document.body, {
    delimiters: [{left: "$$", right: "$$", display: true},
                 {left: "$", right: "$", display: false}],
    throwOnError: false
  });
});
</script></body></html>`
}

func mustAbs(p string) string {
	a, err := filepath.Abs(p)
	if err != nil {
		return p
	}
	return a
}
