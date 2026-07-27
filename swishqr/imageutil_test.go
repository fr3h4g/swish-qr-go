package swishqr

import (
	"image"
	"image/color"
	"testing"
)

func TestResizeRGBAIdentity(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 20, 15))
	for y := 0; y < 15; y++ {
		for x := 0; x < 20; x++ {
			src.SetRGBA(x, y, color.RGBA{
				R: uint8((x * 7) % 256),
				G: uint8((y * 13) % 256),
				B: uint8((x + y) % 256),
				A: 255,
			})
		}
	}

	dst := resizeRGBA(src, 20, 15)
	for y := 0; y < 15; y++ {
		for x := 0; x < 20; x++ {
			want := src.RGBAAt(x, y)
			got := dst.RGBAAt(x, y)
			if got != want {
				t.Fatalf("identity resize changed pixel (%d,%d): got %v, want %v", x, y, got, want)
			}
		}
	}
}

func TestResizeRGBADimensions(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 300, 300))
	cases := []struct{ w, h int }{
		{472, 472}, // typical final downscale
		{40, 40},   // typical logo upscale/downscale
		{1, 1},
	}
	for _, c := range cases {
		dst := resizeRGBA(src, c.w, c.h)
		if dx, dy := dst.Bounds().Dx(), dst.Bounds().Dy(); dx != c.w || dy != c.h {
			t.Errorf("resizeRGBA(src, %d, %d) produced %dx%d image", c.w, c.h, dx, dy)
		}
	}
}

func TestResizeRGBAPreservesConstantColor(t *testing.T) {
	// A weighted average of a spatially-constant field must equal that
	// constant, at any scale factor (upscale, downscale, or identity) -
	// a simple, strong correctness property for a resampler.
	want := color.RGBA{R: 180, G: 47, B: 146, A: 255}
	src := image.NewRGBA(image.Rect(0, 0, 50, 50))
	for y := 0; y < 50; y++ {
		for x := 0; x < 50; x++ {
			src.SetRGBA(x, y, want)
		}
	}

	for _, size := range []int{5, 17, 50, 123} {
		dst := resizeRGBA(src, size, size)
		for _, p := range [][2]int{{0, 0}, {size / 2, size / 2}, {size - 1, size - 1}} {
			got := dst.RGBAAt(p[0], p[1])
			if absDiffUint8(got.R, want.R) > 1 || absDiffUint8(got.G, want.G) > 1 ||
				absDiffUint8(got.B, want.B) > 1 || absDiffUint8(got.A, want.A) > 1 {
				t.Errorf("resize to %dx%d: pixel %v = %v, want ~%v", size, size, p, got, want)
			}
		}
	}
}

func absDiffUint8(a, b uint8) int {
	if a > b {
		return int(a - b)
	}
	return int(b - a)
}

func TestCompositeOver(t *testing.T) {
	dst := image.NewRGBA(image.Rect(0, 0, 10, 10))
	// Fill dst with opaque blue.
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			dst.SetRGBA(x, y, color.RGBA{B: 255, A: 255})
		}
	}

	src := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			src.SetRGBA(x, y, color.RGBA{R: 255, A: 255}) // opaque red
		}
	}

	compositeOver(dst, src, 3, 3)

	// Inside the pasted region: opaque red should have fully replaced blue.
	if got := dst.RGBAAt(4, 4); got != (color.RGBA{R: 255, A: 255}) {
		t.Errorf("inside composited region = %v, want opaque red", got)
	}
	// Outside the pasted region: original blue should be untouched.
	if got := dst.RGBAAt(0, 0); got != (color.RGBA{B: 255, A: 255}) {
		t.Errorf("outside composited region = %v, want untouched opaque blue", got)
	}
}
