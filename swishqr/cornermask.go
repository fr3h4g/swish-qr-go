package swishqr

import (
	"image"
	"math"

	"github.com/fogleman/gg"
)

// cornerMaskSize is the working resolution of the corner accent artwork
// before it gets scaled down to the QR code's actual corner size.
const cornerMaskSize = 228

func deg(d float64) float64 { return d * math.Pi / 180 }

// generateCornerMask draws the rounded "swish" accent shape used to mask
// the QR code's finder patterns, oriented for the top-left corner. The
// other two corners are produced by rotating this image 90 degrees.
//
// The shape is built from a handful of circular arcs and straight
// connectors; it's a faithful geometric re-creation of the accent rather
// than a pixel-for-pixel port, since Go's rasterizer (anti-aliased) and
// Pillow's (aliased) draw arcs differently.
func generateCornerMask() *image.RGBA {
	dc := gg.NewContext(cornerMaskSize, cornerMaskSize)
	dc.SetRGBA(1, 1, 1, 1)
	dc.SetLineCap(gg.LineCapButt)

	// Large diagonal sweep connecting the two straight edges.
	dc.SetLineWidth(100)
	dc.DrawArc(162.5, 166.5, 98, deg(180), deg(270))
	dc.Stroke()

	// Outer arc giving the sweep its rounded silhouette.
	dc.SetLineWidth(32)
	dc.DrawArc(183, 183, 183, deg(180), deg(270))
	dc.Stroke()

	// Small rounded fillets at the three non-right-angle joins.
	dc.SetLineWidth(16)
	dc.DrawArc(15, 212, 15, deg(90), deg(180))
	dc.Stroke()
	dc.DrawArc(212, 212, 15, deg(0), deg(90))
	dc.Stroke()
	dc.DrawArc(212, 15, 15, deg(270), deg(360))
	dc.Stroke()

	// Straight connectors along the edges.
	dc.SetLineWidth(32)
	dc.DrawLine(16, 211, 211, 211)
	dc.Stroke()
	dc.DrawLine(211, 16, 211, 211)
	dc.Stroke()
	dc.DrawLine(15, 183, 15, 211)
	dc.Stroke()
	dc.DrawLine(183, 15, 211, 15)
	dc.Stroke()

	return dc.Image().(*image.RGBA)
}
