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
