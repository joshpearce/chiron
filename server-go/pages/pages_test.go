package pages

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/mjbraun/chiron/server/corpus"
	"github.com/mjbraun/chiron/server/render"
)

func testRenderer(t *testing.T) *Renderer {
	t.Helper()
	r := &Renderer{
		KatexDir: "../../ipad-app/Chiron/Resources/katex",
		CacheDir: t.TempDir(),
	}
	if _, err := os.Stat(r.KatexDir); err != nil {
		t.Skip("katex assets not present")
	}
	if _, err := exec.LookPath("pdftoppm"); err != nil {
		t.Skip("pdftoppm not installed")
	}
	if r.chrome() == "" {
		t.Skip("chrome not installed")
	}
	return r
}

func TestRenderPagesProducesDeviceSizedPNGs(t *testing.T) {
	r := testRenderer(t)
	ch := &render.Chapter{
		Unit:  "u-test",
		Title: "Pages under test",
		HTML: `<h1>Attention</h1>
<p>The scaled dot product $\frac{QK^T}{\sqrt{d_k}}$ is the core.</p>
<div class="beat" data-beat-id="b1"></div>
<p>` + loremParagraphs(40) + `</p>
$$H(P^*, P_\theta) = H(P^*) + D_{KL}(P^* \| P_\theta)$$`,
		Beats: []corpus.Beat{{ID: "b1", Prompt: "Compute $[1,2]\\cdot[3,4]$ by hand."}},
	}

	res, err := r.Render(ch)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if res.Count < 2 {
		t.Fatalf("want >=2 pages for long content, got %d", res.Count)
	}
	for i := 0; i < res.Count; i++ {
		p := res.PagePath(i)
		w, h, err := pngSize(p)
		if err != nil {
			t.Fatalf("page %d: %v", i, err)
		}
		if w != PageW || h != PageH {
			t.Fatalf("page %d is %dx%d, want %dx%d", i, w, h, PageW, PageH)
		}
	}

	// Same chapter renders from cache: identical dir, no re-render.
	res2, err := r.Render(ch)
	if err != nil {
		t.Fatalf("cached render: %v", err)
	}
	if res2.Dir != res.Dir || res2.Count != res.Count {
		t.Fatalf("cache miss: %v vs %v", res2, res)
	}
}

func TestWrapOrdersPretestBodyCheck(t *testing.T) {
	r := &Renderer{KatexDir: "k", CacheDir: "c"}
	ch := &render.Chapter{
		Unit:  "u9",
		Title: "Ordering",
		HTML:  `<p>BODY-MARKER</p>`,
		Pretest: []render.ClientItem{{
			ID: "u9-p1", Kind: "constructed",
			Prompt: "Compute $2+2$. <script> must be escaped.",
		}},
		Check: []render.ClientItem{
			{ID: "u9-q1", Kind: "constructed", Prompt: "Free text one."},
			{ID: "u9-q2", Kind: "mcq", Prompt: "Pick one.",
				Options: []render.ClientOption{{Text: "first"}, {Text: "second"}}},
		},
	}
	doc := r.wrap(ch)

	pre := strings.Index(doc, "Before you read")
	body := strings.Index(doc, "BODY-MARKER")
	chk := strings.Index(doc, "Comprehension check")
	if pre < 0 || body < 0 || chk < 0 || !(pre < body && body < chk) {
		t.Fatalf("order wrong: pretest@%d body@%d check@%d", pre, body, chk)
	}
	if !strings.Contains(doc, "&lt;script&gt;") || strings.Contains(doc, "<script> must") {
		t.Fatal("prompt not HTML-escaped")
	}
	for _, want := range []string{`data-item-id="u9-p1"`, `data-item-id="u9-q1"`,
		`data-item-id="u9-q2"`, "first", "second", "mcq-letter"} {
		if !strings.Contains(doc, want) {
			t.Fatalf("missing %q", want)
		}
	}
	// Interactive layout leaves confidence/IDK to the client's controls.
	for _, banned := range []string{"How confident are you?", "I don't know - moving on"} {
		if strings.Contains(doc, banned) {
			t.Fatalf("interactive layout printed %q", banned)
		}
	}
	// The print layout keeps them on the page for the real-paper flow.
	rp := &Renderer{KatexDir: "k", CacheDir: "c", PrintLayout: true}
	pdoc := rp.wrap(ch)
	for _, want := range []string{"How confident are you?", "I don't know - moving on", "mcq-tick"} {
		if !strings.Contains(pdoc, want) {
			t.Fatalf("print layout missing %q", want)
		}
	}
	// Reference answers and rubrics must never reach paper.
	if strings.Contains(doc, "Reference answer") {
		t.Fatal("reveal leaked into pages")
	}
}

func TestBeatInjection(t *testing.T) {
	html := injectBeats(
		`<p>before</p><div class="beat" data-beat-id="x1"></div><p>after</p>`,
		[]corpus.Beat{{ID: "x1", Type: "compute", Prompt: "Work this out."}})
	want := `class="beat-box"`
	if !contains(html, want) || !contains(html, "Work this out.") {
		t.Fatalf("beat not injected: %s", html)
	}
	if contains(html, `data-beat-id="x1"></div><p>after`) {
		t.Fatalf("placeholder left behind: %s", html)
	}
}

func contains(s, sub string) bool { return strings.Contains(s, sub) }

func loremParagraphs(n int) string {
	s := ""
	for i := 0; i < n; i++ {
		s += "Every quantity in this book is a block of numbers, and the only thing you need to track is its shape. Get the shapes right and the equations follow almost mechanically. "
	}
	return s
}

func pngSize(path string) (int, int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()
	var hdr [24]byte
	if _, err := f.Read(hdr[:]); err != nil {
		return 0, 0, err
	}
	w := int(hdr[16])<<24 | int(hdr[17])<<16 | int(hdr[18])<<8 | int(hdr[19])
	h := int(hdr[20])<<24 | int(hdr[21])<<16 | int(hdr[22])<<8 | int(hdr[23])
	return w, h, nil
}

// The placement step is one question: same page as the intro, no series
// header, and the item map still points at it.
func TestScreenerChapterIsOnePage(t *testing.T) {
	r := &Renderer{KatexDir: "k", CacheDir: "c"}
	ch := &render.Chapter{
		Unit: "u0", Title: "Calibration", Calibration: true,
		HTML: `<p>One placement question.</p>`,
		Check: []render.ClientItem{{
			ID: "u0-s1", Kind: "constructed", Check: "screener",
			Prompt: "Rate yourself 1-5."}},
	}
	doc := r.wrap(ch)
	if !strings.Contains(doc, "items-inline") {
		t.Error("screener section not inlined with the intro")
	}
	if strings.Contains(doc, "The series") {
		t.Error("series framing shown over the screener")
	}
	ip := ItemPages(ch, 1)
	if len(ip) != 1 || ip[0].Page != 0 || ip[0].Check != "screener" {
		t.Fatalf("item map = %+v", ip)
	}
}

func TestScreenerLevelsRenderAsVerticalList(t *testing.T) {
	r := &Renderer{KatexDir: "k", CacheDir: "c"}
	ch := &render.Chapter{
		Unit: "u0", Calibration: true,
		HTML: `<p>intro</p>`,
		Check: []render.ClientItem{{
			ID: "u0-s1", Kind: "constructed", Check: "screener",
			Prompt: "Rate yourself.",
			Options: []render.ClientOption{
				{Text: "Absolute novice"}, {Text: "Expert"}},
		}},
	}
	doc := r.wrap(ch)
	if !strings.Contains(doc, `<ul class="screener-list">`) {
		t.Fatal("no screener list")
	}
	if !strings.Contains(doc, `<div class="item-prompt">Rate yourself.`) {
		t.Error("screener box should carry no item number")
	}
	info := Screener(ch)
	if info == nil || info.ItemID != "u0-s1" || len(info.Options) != 2 ||
		info.Intro != "intro" {
		t.Fatalf("Screener() = %+v", info)
	}
	if Screener(&render.Chapter{HTML: "x"}) != nil {
		t.Error("non-screener chapter must yield nil")
	}
	for _, want := range []string{`<span class="item-num">1.</span>Absolute novice`,
		`<span class="item-num">2.</span>Expert`} {
		if !strings.Contains(doc, want) {
			t.Fatalf("missing %q", want)
		}
	}
}
