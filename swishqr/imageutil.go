package swishqr

import (
	"image"
	"image/color"
	stddraw "image/draw"
	"math"
)

// This file replaces the golang.org/x/image dependency with a small
// hand-rolled separable Catmull-Rom resampler (resizeRGBA) so the
// package only needs the standard library.

// rgbaF64 is one pixel's alpha-premultiplied channels, in the same
// 0..65535 range that color.Color.RGBA() returns.
type rgbaF64 struct{ r, g, b, a float64 }

// catmullRomWeight is the Keys/Catmull-Rom cubic convolution kernel
// (a = -0.5), the same kernel golang.org/x/image/draw's CatmullRom
// scaler uses.
func catmullRomWeight(t float64) float64 {
	const a = -0.5
	t = math.Abs(t)
	switch {
	case t <= 1:
		return (a+2)*t*t*t - (a+3)*t*t + 1
	case t < 2:
		return a*t*t*t - 5*a*t*t + 8*a*t - 4*a
	default:
		return 0
	}
}

// resampleAxis1D computes one output sample (index di, of dstLen total)
// along a single axis, by taking a Catmull-Rom-weighted average of the
// nearby source samples (of srcLen total, fetched via get, with edge
// indices clamped). scale is srcLen/dstLen: when scale > 1 (shrinking),
// the kernel's support is widened proportionally and its input distance
// scaled down accordingly, which is the standard way to keep a cubic
// resampler from aliasing on minification — the same technique
// golang.org/x/image/draw's scalers use.
func resampleAxis1D(get func(si int) rgbaF64, srcLen, di int, scale float64) rgbaF64 {
	center := (float64(di) + 0.5) * scale
	const baseRadius = 2.0

	kernelScale := 1.0
	support := baseRadius
	if scale > 1 {
		kernelScale = scale
		support = baseRadius * scale
	}

	lo := int(math.Floor(center - 0.5 - support))
	hi := int(math.Ceil(center - 0.5 + support))

	var sum rgbaF64
	var sumW float64
	for si := lo; si <= hi; si++ {
		w := catmullRomWeight((float64(si) + 0.5 - center) / kernelScale)
		if w == 0 {
			continue
		}
		clamped := si
		if clamped < 0 {
			clamped = 0
		} else if clamped >= srcLen {
			clamped = srcLen - 1
		}
		p := get(clamped)
		sum.r += p.r * w
		sum.g += p.g * w
		sum.b += p.b * w
		sum.a += p.a * w
		sumW += w
	}
	if sumW == 0 {
		return rgbaF64{}
	}
	return rgbaF64{sum.r / sumW, sum.g / sumW, sum.b / sumW, sum.a / sumW}
}

func rgbaFrom16Bit(p rgbaF64) color.RGBA {
	clamp := func(v float64) uint8 {
		if v < 0 {
			v = 0
		} else if v > 65535 {
			v = 65535
		}
		return uint8(v/257 + 0.5)
	}
	return color.RGBA{R: clamp(p.r), G: clamp(p.g), B: clamp(p.b), A: clamp(p.a)}
}

// resizeRGBA scales src to exactly w x h using a separable Catmull-Rom
// cubic resampler, operating in the alpha-premultiplied space that
// color.Color.RGBA() always returns (matching image.RGBA's own storage
// convention), so it works correctly regardless of src's concrete pixel
// format.
func resizeRGBA(src image.Image, w, h int) *image.RGBA {
	sb := src.Bounds()
	sw, sh := sb.Dx(), sb.Dy()

	scaleX := float64(sw) / float64(w)
	scaleY := float64(sh) / float64(h)

	// Horizontal pass: sw x sh -> w x sh.
	mid := make([][]rgbaF64, sh)
	for y := 0; y < sh; y++ {
		row := make([]rgbaF64, w)
		get := func(si int) rgbaF64 {
			r, g, b, a := src.At(sb.Min.X+si, sb.Min.Y+y).RGBA()
			return rgbaF64{float64(r), float64(g), float64(b), float64(a)}
		}
		for x := 0; x < w; x++ {
			row[x] = resampleAxis1D(get, sw, x, scaleX)
		}
		mid[y] = row
	}

	// Vertical pass: w x sh -> w x h.
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := 0; x < w; x++ {
		get := func(si int) rgbaF64 { return mid[si][x] }
		for y := 0; y < h; y++ {
			dst.SetRGBA(x, y, rgbaFrom16Bit(resampleAxis1D(get, sh, y, scaleY)))
		}
	}
	return dst
}

// compositeOver alpha-composites src onto dst with its top-left corner at (x, y).
func compositeOver(dst *image.RGBA, src image.Image, x, y int) {
	r := image.Rect(x, y, x+src.Bounds().Dx(), y+src.Bounds().Dy())
	stddraw.Draw(dst, r, src, src.Bounds().Min, stddraw.Over)
}
