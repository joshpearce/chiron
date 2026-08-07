// Package ink turns captured pen strokes into images a vision model can read.
//
// Clients send strokes as normalized polylines (0..1 in page coordinates);
// the server rasterizes them onto a white canvas for handwriting
// transcription. Rasterizing server-side keeps the client dumb and makes the
// submitted answer inspectable - the PNG that was graded can be kept.
package ink

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
)

type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// Rasterize draws the strokes at the given pixel size. The pen is a fixed
// round nib a few pixels wide - enough for legible transcription input.
func Rasterize(strokes [][]Point, w, h int) ([]byte, error) {
	total := 0
	for _, s := range strokes {
		total += len(s)
	}
	if total == 0 {
		return nil, fmt.Errorf("no ink")
	}

	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := range img.Pix {
		img.Pix[i] = 0xff
	}

	const nib = 2 // radius in pixels
	dot := func(cx, cy int) {
		for dx := -nib; dx <= nib; dx++ {
			for dy := -nib; dy <= nib; dy++ {
				if dx*dx+dy*dy > nib*nib {
					continue
				}
				x, y := cx+dx, cy+dy
				if x >= 0 && x < w && y >= 0 && y < h {
					img.SetRGBA(x, y, color.RGBA{A: 0xff})
				}
			}
		}
	}
	for _, s := range strokes {
		if len(s) == 1 {
			dot(int(s[0].X*float64(w)), int(s[0].Y*float64(h)))
			continue
		}
		for i := 1; i < len(s); i++ {
			x0, y0 := s[i-1].X*float64(w), s[i-1].Y*float64(h)
			x1, y1 := s[i].X*float64(w), s[i].Y*float64(h)
			// Walk the segment densely enough that consecutive nibs overlap.
			dx, dy := x1-x0, y1-y0
			steps := int(max64(abs64(dx), abs64(dy))) + 1
			for t := 0; t <= steps; t++ {
				f := float64(t) / float64(steps)
				dot(int(x0+dx*f), int(y0+dy*f))
			}
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func abs64(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}

func max64(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
