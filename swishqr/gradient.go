package swishqr

import (
	"image"
	"image/color"
)

// generateSwishGradient renders a w x h diagonal gradient between Swish's
// two brand colors (pink -> orange), top-left to bottom-right.
func generateSwishGradient(w, h int) *image.RGBA {
	colorA := [3]float64{180, 47, 146}
	colorB := [3]float64{239, 64, 35}

	img := image.NewRGBA(image.Rect(0, 0, w, h))
	maxT := float64(w + h - 2)
	if maxT <= 0 {
		maxT = 1
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			t := float64(x+y) / maxT
			c := color.RGBA{
				R: uint8(colorA[0] + t*(colorB[0]-colorA[0])),
				G: uint8(colorA[1] + t*(colorB[1]-colorA[1])),
				B: uint8(colorA[2] + t*(colorB[2]-colorA[2])),
				A: 255,
			}
			img.Set(x, y, c)
		}
	}
	return img
}
