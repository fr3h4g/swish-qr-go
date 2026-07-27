package swishqr

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	stdpng "image/png"
)

// buildAlphaMask paints the three rounded corner accents and every "on"
// QR module as opaque white circles onto an otherwise fully transparent
// canvas. The result is used as the alpha channel for the gradient fill,
// so only the dots and corner accents end up visible.
func buildAlphaMask(modules [][]bool, size, dotSize, dotSpace int) *image.RGBA {
	imageSize := size*dotSize + size*dotSpace
	modulePx := float64(dotSize + dotSpace)

	canvas := image.NewRGBA(image.Rect(0, 0, imageSize, imageSize))

	drawCornerAccents(canvas, modulePx, size)

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			if !modules[x][y] {
				continue
			}
			cx := float64(x*dotSize+x*dotSpace) + float64(dotSize)/2
			cy := float64(y*dotSize+y*dotSpace) + float64(dotSize)/2
			fillCircle(canvas, cx, cy, float64(dotSize)/2)
		}
	}
	return canvas
}

func makeSwishPNG(modules [][]bool, size, border int) ([]byte, error) {
	if border < 0 {
		return nil, fmt.Errorf("swishqr: border must be non-negative")
	}

	const dotSize = 30
	const dotSpace = 3
	imageSize := size*dotSize + size*dotSpace

	centerSide := pyRound(((0.2*float64(size))+3.4)/2)*2 - 1
	center := pyRound((float64(size) - float64(centerSide)) / 2)

	clearSquare(modules, 0, 0, 7)
	clearSquare(modules, size-7, 0, 7)
	clearSquare(modules, 0, size-7, 7)
	clearSquare(modules, center, center, centerSide)
	clearCorner(modules, size)

	alphaMask := buildAlphaMask(modules, size, dotSize, dotSpace)
	gradient := generateSwishGradient(imageSize, imageSize)

	img := image.NewRGBA(image.Rect(0, 0, imageSize, imageSize))
	for y := 0; y < imageSize; y++ {
		for x := 0; x < imageSize; x++ {
			_, _, _, a := alphaMask.At(x, y).RGBA()
			gr, gg2, gb, _ := gradient.At(x, y).RGBA()
			alpha8 := uint8(a >> 8)
			// image.RGBA stores alpha-premultiplied color (R,G,B <= A).
			// The gradient is fully opaque, so its channels must be
			// scaled down by the mask's alpha here, otherwise
			// transparent/partially-transparent pixels end up with
			// "leftover" color baked in, which produces fringing once
			// this buffer is resampled (compositeOver/resizeRGBA).
			img.SetRGBA(x, y, color.RGBA{
				R: uint8(uint32(gr>>8) * uint32(alpha8) / 255),
				G: uint8(uint32(gg2>>8) * uint32(alpha8) / 255),
				B: uint8(uint32(gb>>8) * uint32(alpha8) / 255),
				A: alpha8,
			})
		}
	}

	logoSize := pyRound(6 * float64(size+8))
	logo := resizeRGBA(swishLogo(), logoSize, logoSize)
	logoPos := pyRound(float64(imageSize)/2 - float64(logoSize)/2)
	compositeOver(img, logo, logoPos, logoPos)

	final := resizeRGBA(img, 472, 472)

	var buf bytes.Buffer
	if err := stdpng.Encode(&buf, final); err != nil {
		return nil, fmt.Errorf("swishqr: encoding png: %w", err)
	}
	return buf.Bytes(), nil
}
