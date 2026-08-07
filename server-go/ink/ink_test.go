package ink

import (
	"bytes"
	"image/png"
	"testing"
)

func TestRasterizeDrawsStrokes(t *testing.T) {
	// A diagonal stroke and a dot, in normalized page coordinates.
	strokes := [][]Point{
		{{X: 0.1, Y: 0.1}, {X: 0.5, Y: 0.5}, {X: 0.9, Y: 0.1}},
		{{X: 0.2, Y: 0.8}},
	}
	data, err := Rasterize(strokes, 810, 1080)
	if err != nil {
		t.Fatal(err)
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("output is not a png: %v", err)
	}
	if img.Bounds().Dx() != 810 || img.Bounds().Dy() != 1080 {
		t.Fatalf("size %v", img.Bounds())
	}
	// The stroke midpoint must be dark, a far corner must be white.
	dark := 0
	for dx := -3; dx <= 3; dx++ {
		for dy := -3; dy <= 3; dy++ {
			r, g, b, _ := img.At(405+dx, 540+dy).RGBA()
			if r < 0x8000 && g < 0x8000 && b < 0x8000 {
				dark++
			}
		}
	}
	if dark == 0 {
		t.Error("stroke midpoint not drawn")
	}
	r, _, _, _ := img.At(790, 1060).RGBA()
	if r != 0xffff {
		t.Error("background is not white")
	}
}

func TestRasterizeEmptyIsError(t *testing.T) {
	if _, err := Rasterize(nil, 100, 100); err == nil {
		t.Error("empty strokes should error, callers treat no-ink as IDK")
	}
}
