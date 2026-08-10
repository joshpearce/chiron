package pages

import (
	"fmt"
	"strings"
)

// The contents / spine screen (rmpp/design/SPEC.md §7): page-rendered rows
// with dotted leaders and mastery states; the native layer adds invisible
// full-width tap targets on written rows. Row geometry is fixed and
// published so the client can map a tap to a chapter without reading the
// image: heading block heights are pinned, every row is exactly RowH tall.
const (
	ContentsRowsTop = 384 // content-area y of the first row (page y, from top)
	ContentsRowH    = 88
)

// ContentsRow is one chapter line. State is cleared | in_progress |
// unwritten; Score is the cleared check score in percent.
type ContentsRow struct {
	Unit    string `json:"unit"`
	N       int    `json:"n"`
	Title   string `json:"title"`
	State   string `json:"state"`
	Score   int    `json:"score,omitempty"`
	Current bool   `json:"current,omitempty"` // the chapter being read now
}

type ContentsDoc struct {
	Subject string        `json:"subject"`
	Rows    []ContentsRow `json:"rows"`
}

func contentsRowHTML(r ContentsRow) string {
	var b strings.Builder
	cls := "crow"
	if r.State == "unwritten" {
		cls += " crow-unwritten"
	}
	if r.Current {
		cls += " crow-current"
	}
	b.WriteString(`<div class="` + cls + `">`)
	b.WriteString(fmt.Sprintf(`<div class="cnum">%d</div>`, r.N))
	b.WriteString(`<div class="ctitle">` + escapeText(r.Title) + `</div>`)
	switch r.State {
	case "unwritten":
		b.WriteString(`<div class="cstate-unwritten">not yet written</div>`)
	case "cleared":
		b.WriteString(`<div class="cleader"></div>`)
		b.WriteString(fmt.Sprintf(`<div class="cstate">CLEARED · %d</div>`, r.Score))
	default:
		b.WriteString(`<div class="cleader"></div>`)
		b.WriteString(`<div class="cstate cstate-now">IN PROGRESS</div>`)
	}
	b.WriteString(`</div>`)
	return b.String()
}

func (r *Renderer) wrapContents(d *ContentsDoc) string {
	var b strings.Builder
	b.WriteString(`<h1 class="contents-h1">Contents.</h1>`)
	b.WriteString(`<div class="contents-why">Chapters are written as you reach them, shaped by your checks.</div>`)
	b.WriteString(`<div class="contents-gap"></div>`)
	for _, row := range d.Rows {
		b.WriteString(contentsRowHTML(row))
	}
	return r.shell(b.String(), contentsCSS(), strings.ToUpper(d.Subject), "CONTENTS")
}

// RenderContents produces the contents page, cached like every render.
func (r *Renderer) RenderContents(d *ContentsDoc) (Result, error) {
	return r.renderShared("contents", r.wrapContents(d))
}

func contentsCSS() string {
	// The heading block is height-pinned so rows start exactly at
	// ContentsRowsTop: 104 (offset) + 64 (H1) + 16 + 36 (explainer) + 64.
	// padding-top, not margin: the page container zeroes its first child's
	// top margin.
	return fmt.Sprintf(`.contents-h1 { font-size: 56px; line-height: 64px; height: 64px; overflow: hidden; font-weight: 600; padding-top: 104px; margin: 0 0 16px 0; }
.contents-why { font-style: italic; font-size: 26px; line-height: 36px; height: 36px; overflow: hidden; color: #555; }
.contents-gap { height: 64px; }
.crow { height: %dpx; display: flex; align-items: center; }
.cnum { flex: 0 0 60px; font-size: 30px; font-weight: 700; }
.ctitle { flex: 0 1 auto; font-size: 34px; padding-right: 24px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.crow-current .ctitle { font-weight: 600; }
.crow-unwritten .ctitle, .crow-unwritten .cnum { color: #777; }
.cleader { flex: 1 1 auto; border-bottom: 2px dotted #999; margin: 0 24px 10px 0; align-self: flex-end; height: 54%%; }
.cstate { flex: 0 0 auto; font-size: 22px; font-weight: 600; letter-spacing: 0.10em; color: #333; }
.cstate-now { color: #000; }
.cstate-unwritten { flex: 0 0 auto; margin-left: auto; font-style: italic; font-size: 26px; color: #777; }`,
		ContentsRowH)
}
