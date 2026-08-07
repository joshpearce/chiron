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
	"strings"
	"sync"

	"github.com/mjbraun/chiron/server/corpus"
	"github.com/mjbraun/chiron/server/render"
)

// reMarkable Paper Pro portrait, native pixels (11.8", 229 ppi).
const (
	PageW = 1620
	PageH = 2160
)

// The interactive item-page geometry contract. Items are packed 1-3 to a
// page into fixed-height slots measured in sixths of the content area; the
// server both enforces the geometry (explicit page containers and box
// heights) and publishes it (per-item regions in the pages meta), which is
// what lets the client place controls INSIDE each box and assign ink to
// items without extracting anything from the render.
const (
	BoxTop     = 100 // content top, page px
	SlotUnit   = 326 // one sixth of the content area, page px
	SlotsTotal = 6
	BoxGap     = 14  // margin between stacked boxes, inside the slot height
	StripH     = 130 // control strip reserved inside each box bottom
)

// Bump when the wrapper HTML/CSS changes so cached renders invalidate.
const styleVersion = "v13"

type Renderer struct {
	// ChromePath overrides Chrome discovery; empty means look in the
	// usual places.
	ChromePath string
	// Renders of DISTINCT chapters run concurrently (the calibration
	// prerender walks five level variants while the learner reads the
	// screener), but identical renders - eager plus on-demand for the same
	// chapter - collapse onto one in-flight job so they cannot trample the
	// same cache directory.
	mu       sync.Mutex
	inflight map[string]*renderJob
	// KatexDir holds katex.min.css/js and contrib/auto-render.min.js
	// (the same assets the iPad bundles).
	KatexDir string
	// CacheDir receives one subdirectory per rendered chapter, keyed by
	// content hash.
	CacheDir string
	// PrintLayout renders answer scaffolding ON the page - confidence
	// pills, IDK tick rows, MCQ tick squares - for clients that are real
	// paper (the rmapi flow). The default interactive layout leaves those
	// to the client's own controls and letters the MCQ options instead.
	PrintLayout bool
}

type Result struct {
	Dir   string `json:"-"`
	Hash  string `json:"hash"`
	Count int    `json:"count"`
}

func (r Result) PagePath(n int) string {
	return filepath.Join(r.Dir, fmt.Sprintf("page-%03d.png", n))
}

// ItemPage says which rendered page an item occupies. Pretest items open the
// stack (the section header rides with the first box), check items close it;
// every box starts its own page, so the mapping is arithmetic. Ink drawn on
// an item's page belongs to that item.
type ItemPage struct {
	ID      string `json:"id"`
	Kind    string `json:"kind"`
	Check   string `json:"check"`
	Options int    `json:"options,omitempty"`
	Page    int    `json:"page"`
	// Region on the page, normalized 0..1 of page height.
	Top float64 `json:"top"`
	H   float64 `json:"h"`
}

// slotSixths estimates how much of a page an item needs: prompt length,
// options for MCQs, writing room for constructed answers (more for prose
// answers than for numbers), plus the control strip.
func slotSixths(it render.ClientItem) int {
	lines := 1 + len(it.Prompt)/75
	h := 170 + lines*44 + StripH
	if it.Kind == "mcq" {
		h += len(it.Options) * 70
	} else if it.Check == "llm" {
		h += 460
	} else {
		h += 250
	}
	s := (h + SlotUnit - 1) / SlotUnit
	if s < 2 {
		s = 2
	}
	if s > SlotsTotal {
		s = SlotsTotal
	}
	return s
}

// packItems fills pages in order: a new page starts when the next item
// does not fit.
func packItems(items []render.ClientItem) [][]render.ClientItem {
	var pages [][]render.ClientItem
	var cur []render.ClientItem
	used := 0
	for _, it := range items {
		s := slotSixths(it)
		if used+s > SlotsTotal && len(cur) > 0 {
			pages = append(pages, cur)
			cur, used = nil, 0
		}
		cur = append(cur, it)
		used += s
	}
	if len(cur) > 0 {
		pages = append(pages, cur)
	}
	return pages
}

func ItemPages(ch *render.Chapter, pageCount int) []ItemPage {
	regions := func(groups [][]render.ClientItem, firstPage int, fromEnd bool) []ItemPage {
		var out []ItemPage
		base := firstPage
		if fromEnd {
			base = pageCount - len(groups)
		}
		for p, group := range groups {
			off := BoxTop
			for _, it := range group {
				hPx := slotSixths(it)*SlotUnit - BoxGap
				out = append(out, ItemPage{
					ID: it.ID, Kind: it.Kind, Check: it.Check,
					Options: len(it.Options), Page: base + p,
					Top: float64(off) / float64(PageH),
					H:   float64(hPx) / float64(PageH),
				})
				off += hPx + BoxGap
			}
		}
		return out
	}
	var out []ItemPage
	out = append(out, regions(packItems(ch.Pretest), 0, false)...)
	out = append(out, regions(packItems(ch.Check), 0, true)...)
	return out
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

type renderJob struct {
	done chan struct{}
	res  Result
	err  error
}

// Render produces the page stack for a chapter, reusing a previous render
// of identical content and joining an in-flight render of the same content.
func (r *Renderer) Render(ch *render.Chapter) (Result, error) {
	doc := r.wrap(ch)
	sum := sha256.Sum256([]byte(doc))
	key := hex.EncodeToString(sum[:8])

	r.mu.Lock()
	if r.inflight == nil {
		r.inflight = map[string]*renderJob{}
	}
	if job, ok := r.inflight[key]; ok {
		r.mu.Unlock()
		<-job.done
		return job.res, job.err
	}
	job := &renderJob{done: make(chan struct{})}
	r.inflight[key] = job
	r.mu.Unlock()

	job.res, job.err = r.renderDoc(ch, doc, key)
	close(job.done)
	r.mu.Lock()
	delete(r.inflight, key)
	r.mu.Unlock()
	return job.res, job.err
}

func (r *Renderer) renderDoc(ch *render.Chapter, doc, hash string) (Result, error) {
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

// ScreenerInfo describes a screener-only calibration chapter so an
// interactive client can render the placement step natively - the question
// box IS the UI there, with the levels as buttons.
type ScreenerInfo struct {
	ItemID  string   `json:"item_id"`
	Intro   string   `json:"intro"`
	Prompt  string   `json:"prompt"`
	Options []string `json:"options"`
}

var tagStrip = regexp.MustCompile(`<[^>]+>`)

func Screener(ch *render.Chapter) *ScreenerInfo {
	if !ch.Calibration || len(ch.Pretest) != 0 || len(ch.Check) != 1 ||
		ch.Check[0].Check != "screener" {
		return nil
	}
	it := ch.Check[0]
	intro := strings.TrimSpace(tagStrip.ReplaceAllString(ch.HTML, ""))
	// The HTML came through markdown escaping; the native client renders
	// plain text, so entities must come back out.
	for ent, lit := range map[string]string{
		"&quot;": "\"", "&#39;": "'", "&lt;": "<", "&gt;": ">", "&amp;": "&",
	} {
		intro = strings.ReplaceAll(intro, ent, lit)
	}
	info := &ScreenerInfo{
		ItemID: it.ID,
		Intro:  intro,
		Prompt: it.Prompt,
	}
	for _, o := range it.Options {
		info.Options = append(info.Options, o.Text)
	}
	return info
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

// itemsSection renders pretest or check items as workable paper pages:
// prompt, ink room, and a confidence scale to circle. Reveals (reference
// answers, rubrics) deliberately never reach the page - the check is
// closed-book, and answers come back with the next exchange.
func itemsSection(title, note string, items []render.ClientItem, printLayout, inline bool) string {
	if len(items) == 0 {
		return ""
	}
	var b []string
	class := "items-section"
	if inline {
		class += " items-inline"
	}
	b = append(b, `<section class="`+class+`">`)
	// Interactive item pages are headerless: the slot geometry is the
	// contract that lets the client place controls INSIDE each box. Print
	// keeps the section framing - paper has no client to explain it.
	if printLayout {
		if title != "" {
			b = append(b, `<h1>`+html.EscapeString(title)+`</h1>`)
		}
		if note != "" {
			b = append(b, `<p class="items-note">`+html.EscapeString(note)+`</p>`)
		}
	}
	num := 0
	renderItem := func(it render.ClientItem, style string) []string {
		num++
		i := num - 1
		var b []string
		b = append(b, fmt.Sprintf(`<div class="item-box" data-item-id=%q%s>`, it.ID, style))
		numSpan := fmt.Sprintf(`<span class="item-num">%d.</span>`, i+1)
		if it.Check == "screener" {
			numSpan = ""
		}
		b = append(b, `<div class="item-prompt">`+numSpan+
			html.EscapeString(it.Prompt)+`</div>`)
		switch {
		case it.Kind == "mcq" && printLayout:
			b = append(b, `<ul class="mcq">`)
			for _, o := range it.Options {
				b = append(b, `<li><span class="mcq-tick"></span>`+html.EscapeString(o.Text)+`</li>`)
			}
			b = append(b, `</ul>`)
		case it.Kind == "mcq":
			// The client renders matching lettered buttons.
			b = append(b, `<ul class="mcq">`)
			for oi, o := range it.Options {
				b = append(b, fmt.Sprintf(`<li><span class="mcq-letter">%c.</span>`, 'A'+oi)+
					html.EscapeString(o.Text)+`</li>`)
			}
			b = append(b, `</ul>`)
		case it.Check == "screener":
			b = append(b, `<ul class="screener-list">`)
			for oi, o := range it.Options {
				tick := ""
				if printLayout {
					tick = `<span class="mcq-tick"></span>`
				}
				b = append(b, fmt.Sprintf(`<li>%s<span class="item-num">%d.</span>`, tick, oi+1)+
					html.EscapeString(o.Text)+`</li>`)
			}
			b = append(b, `</ul>`)
		default:
			b = append(b, `<div class="item-ink"></div>`)
		}
		if printLayout {
			b = append(b, `<div class="idk-row"><span class="mcq-tick"></span>I don't know - moving on</div>`)
			b = append(b, `<div class="confidence">How confident are you?`+
				`<span class="conf-opt">unsure</span><span class="conf-opt">shaky</span>`+
				`<span class="conf-opt">confident</span><span class="conf-opt">sure</span></div>`)
		} else if it.Check != "screener" {
			b = append(b, `<div class="control-strip"></div>`)
		}
		b = append(b, `</div>`)
		return b
	}

	if printLayout || inline {
		// Natural flow: paper pages and the inline screener box.
		for _, it := range items {
			b = append(b, renderItem(it, "")...)
		}
	} else {
		// Slot-packed pages: explicit page containers with fixed-height
		// boxes, exactly mirroring the regions published in the meta.
		for _, group := range packItems(items) {
			b = append(b, `<div class="qpage">`)
			for _, it := range group {
				hPx := slotSixths(it)*SlotUnit - BoxGap
				b = append(b, renderItem(it,
					fmt.Sprintf(` style="height: %dpx"`, hPx))...)
			}
			b = append(b, `</div>`)
		}
	}
	b = append(b, `</section>`)
	return strings.Join(b, "\n")
}

func (r *Renderer) wrap(ch *render.Chapter) string {
	katex := "file://" + mustAbs(r.KatexDir)
	checkTitle, checkNote := "Comprehension check",
		"Closed book. Answer every item before moving on."
	if ch.Calibration {
		checkTitle, checkNote = "The series",
			"Easy to hard. Marking \"I don't know\" freely is part of the design - running out of sure answers is the point."
	}
	// A screener-only chapter is one question: it belongs on the intro
	// page, under the intro, with no series framing above it.
	screenerOnly := ch.Calibration && len(ch.Pretest) == 0 &&
		len(ch.Check) == 1 && ch.Check[0].Check == "screener"
	if screenerOnly {
		checkTitle, checkNote = "", ""
	}
	body := itemsSection("Before you read",
		"You are not supposed to know these yet - answering wrong here is part of how the chapter calibrates.",
		ch.Pretest, r.PrintLayout, false) +
		injectBeats(ch.HTML, ch.Beats) +
		itemsSection(checkTitle, checkNote, ch.Check, r.PrintLayout, screenerOnly)
	return `<!DOCTYPE html><html><head><meta charset="utf-8">
<link rel="stylesheet" href="` + katex + `/katex.min.css">
<script src="` + katex + `/katex.min.js"></script>
<script src="` + katex + `/contrib/auto-render.min.js"></script>
<style>
/* ` + fmt.Sprintf("%s-print=%v", styleVersion, r.PrintLayout) + ` */
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
.items-section { page-break-before: always; }
.items-inline { page-break-before: avoid; }
.items-section:first-child { page-break-before: avoid; }
.items-note { font-style: italic; color: #333; }
.item-box { border: 3px solid #000; margin: 34px 0; padding: 20px 24px; page-break-inside: avoid; page-break-before: always; }
.item-box:first-of-type { page-break-before: avoid; }
` + interactiveItemCSS(r.PrintLayout) + `
.item-num { font-weight: bold; margin-right: 14px; }
.item-ink { height: 420px; border-top: 2px dashed #999; margin-top: 18px; }
.mcq { list-style: none; padding: 0; margin: 16px 0 0 0; }
.mcq li { margin: 14px 0; }
.mcq-tick { display: inline-block; width: 34px; height: 34px; border: 3px solid #000; margin-right: 16px; vertical-align: middle; }
.mcq-letter { font-weight: bold; margin-right: 14px; }
.screener-list { list-style: none; padding: 0; margin: 18px 0 0 0; }
.screener-list li { margin: 14px 0; }
.idk-row { margin-top: 14px; font-size: 26px; color: #333; }
.confidence { margin-top: 16px; font-size: 26px; }
.conf-opt { border: 2px solid #000; border-radius: 24px; padding: 4px 18px; margin-left: 14px; }
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

// interactiveItemCSS pins every interactive answer box to the geometry
// contract (BoxTop..BoxBottom with a reserved control strip). Print keeps
// natural flow - paper needs no reserved zone.
func interactiveItemCSS(printLayout bool) string {
	if printLayout {
		return ""
	}
	return fmt.Sprintf(`.qpage { page-break-before: always; page-break-inside: avoid; height: %dpx; overflow: hidden; }
.qpage .item-box { margin: 0 0 %dpx 0; box-sizing: border-box; position: relative; padding-bottom: %dpx; overflow: hidden; }
.control-strip { position: absolute; left: 24px; right: 24px; bottom: 0; height: %dpx; border-top: 2px dashed #999; }`,
		SlotsTotal*SlotUnit-BoxGap, BoxGap, StripH+16, StripH)
}

func mustAbs(p string) string {
	a, err := filepath.Abs(p)
	if err != nil {
		return p
	}
	return a
}
