package swishqr

import (
	"math"

	"github.com/fogleman/gg"
)

func deg(d float64) float64 { return d * math.Pi / 180 }

// drawCornerAccents paints the three rounded "swish" corner accents onto
// dc using exactly the same geometry as generateCorners in svg.go (see
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
func drawCornerAccents(dc *gg.Context, modulePx float64, size int) {
	// mtp converts a coordinate given in QR-module units (the same
	// space used by generateCorners: modules offset by the fixed +4
	// used for dot placement) into pixels.
	mtp := func(u float64) float64 { return (u - 4) * modulePx }

	// bracket draws the open rounded-corner band: start -> straight leg
	// -> arc (center cu,cv / radius r / from a1 to a2) -> straight leg
	// -> straight leg, closed back to start.
	bracket := func(startU, startV, midU, midV, cu, cv, r, a1, a2, u3, v3, u4, v4 float64) {
		dc.NewSubPath()
		dc.MoveTo(mtp(startU), mtp(startV))
		dc.LineTo(mtp(midU), mtp(midV))
		dc.DrawArc(mtp(cu), mtp(cv), r*modulePx, deg(a1), deg(a2))
		dc.LineTo(mtp(u3), mtp(v3))
		dc.LineTo(mtp(u4), mtp(v4))
		dc.ClosePath()
		dc.SetLineWidth(modulePx)
		dc.Stroke()
	}

	// pie draws the small filled quarter-disc: start point on the
	// circle -> arc to the other point on the circle -> straight line
	// back to the circle's center -> closed back to start.
	pie := func(startU, startV, cu, cv, r, a1, a2 float64) {
		dc.NewSubPath()
		dc.MoveTo(mtp(startU), mtp(startV))
		dc.DrawArc(mtp(cu), mtp(cv), r*modulePx, deg(a1), deg(a2))
		dc.LineTo(mtp(cu), mtp(cv))
		dc.ClosePath()
		dc.Fill()
	}

	sz := float64(size)

	// Top-left corner.
	bracket(4.5, 10.5, 4.5, 9.625, 9.625, 9.625, 5.125, 180, 270, 10.5, 4.5, 10.5, 10.5)
	pie(6, 9.125, 9, 9.125, 3, 180, 270)

	// Bottom-left corner (mirrored vertically).
	bracket(4.5, sz-2.5, 4.5, sz-1.625, 9.625, sz-1.625, 5.125, 180, 90, 10.5, sz+3.5, 10.5, sz-2.5)
	pie(6, sz-1.125, 9, sz-1.125, 3, 180, 90)

	// Top-right corner (mirrored horizontally).
	bracket(sz+3.5, 10.5, sz+3.5, 9.625, sz-1.625, 9.625, 5.125, 0, -90, sz-2.5, 4.5, sz-2.5, 10.5)
	pie(sz+2, 9.125, sz-1, 9.125, 3, 0, -90)
}
