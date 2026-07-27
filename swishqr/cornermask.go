package swishqr

import (
	"image"
	"math"
)

func deg(d float64) float64 { return d * math.Pi / 180 }

// drawCornerAccents paints the three rounded "swish" corner accents onto
// img using exactly the same geometry as generateCorners in svg.go (see
// the coordinates and arc commands there), just converted from
// QR-module units into pixels. Keeping both renderers built from the
// same path data (rather than each having its own hand-drawn shape)
// guarantees the SVG and PNG outputs actually match.
//
// Each corner accent is two shapes:
//   - a "bracket": an open, 1-module-wide stroked band that traces most
//     of the finder pattern's outer edge, rounded off with a single big
//     arc at the true corner.
//   - a "pie": a small solid quarter-disc that fills the inner concave
//     joint of the bracket.
//
// modulePx is the pixel size of one QR module (dotSize+dotSpace); size
// is the QR's module count, needed to place the two corners away from
// the origin.
func drawCornerAccents(img *image.RGBA, modulePx float64, size int) {
	// mtp converts a coordinate given in QR-module units (the same
	// space used by generateCorners: modules offset by the fixed +4
	// used for dot placement) into pixels.
	mtp := func(u float64) float64 { return (u - 4) * modulePx }

	// bracket strokes the open rounded-corner band: start -> straight
	// leg -> arc (center cu,cv / radius r / from a1 to a2) -> straight
	// leg -> straight leg, closed back to start.
	bracket := func(startU, startV, midU, midV, cu, cv, r, a1, a2, u3, v3, u4, v4 float64) {
		start := [2]float64{mtp(startU), mtp(startV)}
		mid := [2]float64{mtp(midU), mtp(midV)}
		p3 := [2]float64{mtp(u3), mtp(v3)}
		p4 := [2]float64{mtp(u4), mtp(v4)}
		arc := arcSeg{mtp(cu), mtp(cv), r * modulePx, deg(a1), deg(a2)}
		arcEnd := [2]float64{arc.cx + arc.r*math.Cos(arc.a2), arc.cy + arc.r*math.Sin(arc.a2)}
		segs := []pathSeg{
			lineSeg{start[0], start[1], mid[0], mid[1]},
			arc,
			lineSeg{arcEnd[0], arcEnd[1], p3[0], p3[1]},
			lineSeg{p3[0], p3[1], p4[0], p4[1]},
			lineSeg{p4[0], p4[1], start[0], start[1]},
		}
		strokePath(img, segs, modulePx)
	}

	// pie fills the small quarter-disc directly (no need to build an
	// explicit path: a circular sector is just "within radius r of the
	// center, and within the given angular range").
	pie := func(cu, cv, r, a1, a2 float64) {
		fillPie(img, mtp(cu), mtp(cv), r*modulePx, deg(a1), deg(a2))
	}

	sz := float64(size)

	// Top-left corner.
	bracket(4.5, 10.5, 4.5, 9.625, 9.625, 9.625, 5.125, 180, 270, 10.5, 4.5, 10.5, 10.5)
	pie(9, 9.125, 3, 180, 270)

	// Bottom-left corner (mirrored vertically).
	bracket(4.5, sz-2.5, 4.5, sz-1.625, 9.625, sz-1.625, 5.125, 180, 90, 10.5, sz+3.5, 10.5, sz-2.5)
	pie(9, sz-1.125, 3, 180, 90)

	// Top-right corner (mirrored horizontally).
	bracket(sz+3.5, 10.5, sz+3.5, 9.625, sz-1.625, 9.625, 5.125, 0, -90, sz-2.5, 4.5, sz-2.5, 10.5)
	pie(sz-1, 9.125, 3, 0, -90)
}
