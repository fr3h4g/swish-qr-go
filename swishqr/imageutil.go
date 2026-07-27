package swishqr

import (
	"image"
	stddraw "image/draw"

	xdraw "golang.org/x/image/draw"
)

// resizeRGBA scales src to exactly w x h using a high-quality resampler.
func resizeRGBA(src image.Image, w, h int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), xdraw.Src, nil)
	return dst
}

// compositeOver alpha-composites src onto dst with its top-left corner at (x, y).
func compositeOver(dst *image.RGBA, src image.Image, x, y int) {
	r := image.Rect(x, y, x+src.Bounds().Dx(), y+src.Bounds().Dy())
	stddraw.Draw(dst, r, src, src.Bounds().Min, stddraw.Over)
}

// rotate90CW rotates src 90 degrees clockwise into a new image.
func rotate90CW(src *image.RGBA) *image.RGBA {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	dst := image.NewRGBA(image.Rect(0, 0, h, w))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dst.Set(h-1-y, x, src.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return dst
}
