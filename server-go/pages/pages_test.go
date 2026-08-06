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
