package swishqr

import (
	"image"
	"image/color"
	"math"
)

// This file is a small, hand-rolled anti-aliased rasterizer, replacing
// the github.com/fogleman/gg dependency. It does not attempt to be a
// general-purpose vector graphics engine — it only implements the three
// shapes this package ever draws onto its alpha mask: filled circles
// (the QR dots), a filled circular sector ("pie", the small accent
// filling the inner corner of each swish bracket), and a stroked closed
// path made of straight segments and a single arc (the swish bracket
// itself). Antialiasing is done analytically via signed distance to
// each shape's boundary/centerline, smoothed over a ~1px band, which is
// plenty of quality here since this canvas gets downsampled again
// later in the pipeline anyway.
//
// Every shape is painted as opaque white with a coverage-driven alpha,
// composited with the standard Porter-Duff "over" operator so
// overlapping anti-aliased edges blend correctly. Only the alpha
// channel of this canvas is ever read by callers (see buildAlphaMask in
// png.go), so the exact RGB values written here don't matter beyond
// being consistently white.

// blendWhite composites one more sample of opaque white, with the given
// coverage (0..1), over the existing pixel at (x, y) using the "over"
// operator, writing the result back as R=G=B=A (premultiplied white).
func blendWhite(img *image.RGBA, x, y int, coverage float64) {
	if coverage <= 0 {
		return
	}
	if coverage > 1 {
		coverage = 1
	}
	b := img.Bounds()
	if x < b.Min.X || x >= b.Max.X || y < b.Min.Y || y >= b.Max.Y {
		return
	}
	existing := float64(img.RGBAAt(x, y).A) / 255
	combined := coverage + existing*(1-coverage)
	a := uint8(combined*255 + 0.5)
	img.SetRGBA(x, y, color.RGBA{R: a, G: a, B: a, A: a})
}

// aaCoverage turns a signed distance (negative = inside, for fills; or
// "distance to width/2" for strokes) into a 0..1 coverage value, with a
// 1px-wide antialiased transition band centered on the boundary.
func aaCoverage(signedDist float64) float64 {
	// signedDist < -0.5  => fully covered
	// signedDist > 0.5   => fully uncovered
	return math.Max(0, math.Min(1, 0.5-signedDist))
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// fillCircle paints an anti-aliased filled circle centered at (cx, cy)
// with the given radius, restricted to the canvas bounds.
func fillCircle(img *image.RGBA, cx, cy, r float64) {
	b := img.Bounds()
	x0 := clampInt(int(math.Floor(cx-r-1)), b.Min.X, b.Max.X)
	x1 := clampInt(int(math.Ceil(cx+r+1)), b.Min.X, b.Max.X)
	y0 := clampInt(int(math.Floor(cy-r-1)), b.Min.Y, b.Max.Y)
	y1 := clampInt(int(math.Ceil(cy+r+1)), b.Min.Y, b.Max.Y)

	for y := y0; y < y1; y++ {
		py := float64(y) + 0.5
		for x := x0; x < x1; x++ {
			px := float64(x) + 0.5
			dist := math.Hypot(px-cx, py-cy) - r
			cov := aaCoverage(dist)
			if cov > 0 {
				blendWhite(img, x, y, cov)
			}
		}
	}
}

// fillPie paints an anti-aliased filled circular sector ("pie slice")
// centered at (cx, cy), radius r, spanning from angle a1 to a2 (radians,
// screen convention: 0 = +x axis, increasing = toward +y).
func fillPie(img *image.RGBA, cx, cy, r, a1, a2 float64) {
	b := img.Bounds()
	x0 := clampInt(int(math.Floor(cx-r-1)), b.Min.X, b.Max.X)
	x1 := clampInt(int(math.Ceil(cx+r+1)), b.Min.X, b.Max.X)
	y0 := clampInt(int(math.Floor(cy-r-1)), b.Min.Y, b.Max.Y)
	y1 := clampInt(int(math.Ceil(cy+r+1)), b.Min.Y, b.Max.Y)

	e1x, e1y := cx+r*math.Cos(a1), cy+r*math.Sin(a1)
	e2x, e2y := cx+r*math.Cos(a2), cy+r*math.Sin(a2)

	for y := y0; y < y1; y++ {
		py := float64(y) + 0.5
		for x := x0; x < x1; x++ {
			px := float64(x) + 0.5
			dx, dy := px-cx, py-cy
			dist := math.Hypot(dx, dy)

			// Radial edge (the arc itself): inside when dist <= r.
			radialCov := aaCoverage(dist - r)
			if radialCov <= 0 {
				continue
			}

			// Angular edges: inside when the point's angle falls between
			// a1 and a2. Approximate the antialiasing on these two
			// straight edges via distance to the corresponding segment
			// (center -> arc endpoint); anything under that chord on the
			// "outside" of the sector reduces coverage.
			angCov := 1.0
			if !angleWithin(math.Atan2(dy, dx), a1, a2) {
				dSeg := math.Min(
					lineSeg{cx, cy, e1x, e1y}.distance(px, py),
					lineSeg{cx, cy, e2x, e2y}.distance(px, py),
				)
				angCov = aaCoverage(dSeg)
			}

			cov := radialCov
			if angCov < cov {
				cov = angCov
			}
			if cov > 0 {
				blendWhite(img, x, y, cov)
			}
		}
	}
}

// angleWithin reports whether angle a lies on the short arc from a1 to
// a2 (a1/a2 may be given in either order, and may differ by more than a
// full turn in intent, but this package only ever draws <=90 degree
// arcs so a simple normalized-range check is sufficient).
func angleWithin(a, a1, a2 float64) bool {
	lo, hi := a1, a2
	if lo > hi {
		lo, hi = hi, lo
	}
	for a < lo {
		a += 2 * math.Pi
	}
	for a > lo+2*math.Pi {
		a -= 2 * math.Pi
	}
	return a >= lo && a <= hi
}

// pathSeg is one piece of a compound stroked path: either a straight
// line or a circular arc, both able to report the distance from an
// arbitrary point to themselves.
type pathSeg interface {
	distance(px, py float64) float64
	bounds() (minX, minY, maxX, maxY float64)
}

type lineSeg struct{ x0, y0, x1, y1 float64 }

func (s lineSeg) distance(px, py float64) float64 {
	dx, dy := s.x1-s.x0, s.y1-s.y0
	lenSq := dx*dx + dy*dy
	if lenSq == 0 {
		return math.Hypot(px-s.x0, py-s.y0)
	}
	t := ((px-s.x0)*dx + (py-s.y0)*dy) / lenSq
	t = math.Max(0, math.Min(1, t))
	cx, cy := s.x0+t*dx, s.y0+t*dy
	return math.Hypot(px-cx, py-cy)
}

func (s lineSeg) bounds() (minX, minY, maxX, maxY float64) {
	return math.Min(s.x0, s.x1), math.Min(s.y0, s.y1), math.Max(s.x0, s.x1), math.Max(s.y0, s.y1)
}

type arcSeg struct{ cx, cy, r, a1, a2 float64 }

func (s arcSeg) distance(px, py float64) float64 {
	dx, dy := px-s.cx, py-s.cy
	dist := math.Hypot(dx, dy)
	ang := math.Atan2(dy, dx)
	if angleWithin(ang, s.a1, s.a2) {
		return math.Abs(dist - s.r)
	}
	e1x, e1y := s.cx+s.r*math.Cos(s.a1), s.cy+s.r*math.Sin(s.a1)
	e2x, e2y := s.cx+s.r*math.Cos(s.a2), s.cy+s.r*math.Sin(s.a2)
	return math.Min(math.Hypot(px-e1x, py-e1y), math.Hypot(px-e2x, py-e2y))
}

func (s arcSeg) bounds() (minX, minY, maxX, maxY float64) {
	return s.cx - s.r, s.cy - s.r, s.cx + s.r, s.cy + s.r
}

// strokePath paints an anti-aliased stroke of the given width along the
// compound path described by segs (the minimum distance to any segment
// determines each pixel's coverage, so the segments should form one
// connected path for a clean result).
func strokePath(img *image.RGBA, segs []pathSeg, width float64) {
	minX, minY := math.Inf(1), math.Inf(1)
	maxX, maxY := math.Inf(-1), math.Inf(-1)
	for _, s := range segs {
		x0, y0, x1, y1 := s.bounds()
		minX, minY = math.Min(minX, x0), math.Min(minY, y0)
		maxX, maxY = math.Max(maxX, x1), math.Max(maxY, y1)
	}

	half := width / 2
	b := img.Bounds()
	x0 := clampInt(int(math.Floor(minX-half-1)), b.Min.X, b.Max.X)
	x1 := clampInt(int(math.Ceil(maxX+half+1)), b.Min.X, b.Max.X)
	y0 := clampInt(int(math.Floor(minY-half-1)), b.Min.Y, b.Max.Y)
	y1 := clampInt(int(math.Ceil(maxY+half+1)), b.Min.Y, b.Max.Y)

	for y := y0; y < y1; y++ {
		py := float64(y) + 0.5
		for x := x0; x < x1; x++ {
			px := float64(x) + 0.5
			minDist := math.Inf(1)
			for _, s := range segs {
				if d := s.distance(px, py); d < minDist {
					minDist = d
				}
			}
			cov := aaCoverage(minDist - half)
			if cov > 0 {
				blendWhite(img, x, y, cov)
			}
		}
	}
}
