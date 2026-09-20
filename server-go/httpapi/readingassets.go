package httpapi

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/gif"
	"image/jpeg"
	"image/png"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	xdraw "golang.org/x/image/draw"
)

// The pictures a book carries: its charts, its exhibits, its photographs.
// A publisher's EPUB stores them at print resolution and barely
// compressed - this book keeps 749 MB of PNG in a 100 MB archive, one
// exhibit of it 8 MB - which is more than a page needs and more than a
// tablet should hold. They are re-encoded once, on the way in, at a size
// a screen can use, and served from beside the book.

// assetLongEdge is the longest side a picture is kept at: enough for a
// chart to be read on a tablet, and to be pinched into.
const assetLongEdge = 1800

const assetQuality = 82

func readingAssets(root, id string) string { return filepath.Join(root, id, "assets") }

// screenAsset re-encodes a picture for reading. A photograph or a chart
// becomes a JPEG; anything with transparency stays a PNG, since a JPEG
// would fill it with black. What cannot be decoded is kept as it is: it
// is the book's, and something else may read it.
func screenAsset(name string, data []byte) (string, []byte) {
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return name, data
	}
	img = fitted(img)
	if format == "png" && hasAlpha(img) {
		var out bytes.Buffer
		enc := png.Encoder{CompressionLevel: png.BestCompression}
		if enc.Encode(&out, img) != nil || out.Len() >= len(data) {
			return name, data
		}
		return withExt(name, ".png"), out.Bytes()
	}
	var out bytes.Buffer
	if jpeg.Encode(&out, img, &jpeg.Options{Quality: assetQuality}) != nil {
		return name, data
	}
	if out.Len() >= len(data) {
		return name, data
	}
	return withExt(name, ".jpg"), out.Bytes()
}

// fitted scales a picture down to the long edge, and leaves a smaller one
// alone: a chart drawn for print is far larger than any screen shows.
func fitted(img image.Image) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	long := w
	if h > long {
		long = h
	}
	if long <= assetLongEdge {
		return img
	}
	scale := float64(assetLongEdge) / float64(long)
	out := image.NewRGBA(image.Rect(0, 0, int(float64(w)*scale), int(float64(h)*scale)))
	xdraw.CatmullRom.Scale(out, out.Bounds(), img, b, draw.Over, nil)
	return out
}

func hasAlpha(img image.Image) bool {
	switch img.(type) {
	case *image.RGBA, *image.NRGBA, *image.RGBA64, *image.NRGBA64, *image.Alpha, *image.Alpha16:
	default:
		return false
	}
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if _, _, _, a := img.At(x, y).RGBA(); a < 0xffff {
				return true
			}
		}
	}
	return false
}

func withExt(name, ext string) string {
	return strings.TrimSuffix(name, filepath.Ext(name)) + ext
}

// handleReadingAsset serves one of a book's pictures. The name is the
// book's own file name, so it is checked against the directory rather
// than trusted.
func (s *Server) handleReadingAsset(w http.ResponseWriter, r *http.Request) {
	id, name := r.PathValue("id"), r.PathValue("name")
	if name == "" || strings.ContainsAny(name, `/\`) || strings.Contains(name, "..") {
		writeError(w, http.StatusBadRequest, "no such picture")
		return
	}
	file := filepath.Join(readingAssets(s.readingsRoot(), id), name)
	data, err := os.ReadFile(file)
	if err != nil {
		writeError(w, http.StatusNotFound, "no picture %q in this book", name)
		return
	}
	w.Header().Set("Content-Type", contentTypeOf(name))
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Header().Set("Content-Length", fmt.Sprint(len(data)))
	w.Write(data)
}

func contentTypeOf(name string) string {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	case ".webp":
		return "image/webp"
	case ".avif":
		return "image/avif"
	default:
		return "image/jpeg"
	}
}

var _ = gif.Decode
