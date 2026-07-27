package swishqr

import (
	"image"
	"math"
	"testing"
)

func TestAACoverage(t *testing.T) {
	cases := []struct {
		dist float64
		want float64
	}{
		{-1, 1},   // well inside
		{0, 0.5},  // exactly on the boundary
		{1, 0},    // well outside
		{-10, 1},
		{10, 0},
	}
	for _, c := range cases {
		if got := aaCoverage(c.dist); got != c.want {
			t.Errorf("aaCoverage(%v) = %v, want %v", c.dist, got, c.want)
		}
	}
}

func TestClampInt(t *testing.T) {
	if got := clampInt(-5, 0, 10); got != 0 {
		t.Errorf("clampInt(-5, 0, 10) = %d, want 0", got)
	}
	if got := clampInt(15, 0, 10); got != 10 {
		t.Errorf("clampInt(15, 0, 10) = %d, want 10", got)
	}
	if got := clampInt(5, 0, 10); got != 5 {
		t.Errorf("clampInt(5, 0, 10) = %d, want 5", got)
	}
}

func TestFillCircle(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	fillCircle(img, 50, 50, 20)

	if a := img.RGBAAt(50, 50).A; a < 250 {
		t.Errorf("circle center alpha = %d, want ~255 (fully covered)", a)
	}
	if a := img.RGBAAt(5, 5).A; a != 0 {
		t.Errorf("far corner alpha = %d, want 0 (fully outside)", a)
	}
	// Somewhere along the boundary ring, at least one pixel should be
	// partially covered (antialiased), not a hard 0/255 step. Scan a
	// small neighborhood rather than one exact pixel, since which pixel
	// centers fall inside the ~1px transition band depends on exact
	// rounding of the chosen center/radius.
	foundPartial := false
	for x := 68; x <= 72; x++ {
		if a := img.RGBAAt(x, 50).A; a != 0 && a != 255 {
			foundPartial = true
			break
		}
	}
	if !foundPartial {
		t.Error("expected at least one partially-covered (antialiased) pixel near the circle's edge")
	}
}

func TestFillCircleOutOfBounds(t *testing.T) {
	// Circles that extend past the canvas edge shouldn't panic.
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	fillCircle(img, 0, 0, 20)
	fillCircle(img, 9, 9, 20)
}

func TestFillPie(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	// Quarter circle from 180deg to 270deg (up-left quadrant), matching
	// the corner accent's own pie usage.
	fillPie(img, 50, 50, 30, math.Pi, 3*math.Pi/2)

	// A point inside the sector (up and to the left of center).
	if a := img.RGBAAt(35, 35).A; a < 200 {
		t.Errorf("point inside pie alpha = %d, want high coverage", a)
	}
	// A point outside the sector's angular range (down-right of center,
	// same radius) should be uncovered even though it's within the
	// circle's radius.
	if a := img.RGBAAt(65, 65).A; a != 0 {
		t.Errorf("point outside pie's angular range alpha = %d, want 0", a)
	}
	// A point outside the circle entirely.
	if a := img.RGBAAt(5, 5).A; a != 0 {
		t.Errorf("point outside pie's radius alpha = %d, want 0", a)
	}
}

func TestAngleWithin(t *testing.T) {
	if !angleWithin(math.Pi, math.Pi, 3*math.Pi/2) {
		t.Error("expected pi to be within [pi, 3pi/2]")
	}
	if angleWithin(0, math.Pi, 3*math.Pi/2) {
		t.Error("expected 0 to be outside [pi, 3pi/2]")
	}
	// Order shouldn't matter.
	if !angleWithin(math.Pi, 3*math.Pi/2, math.Pi) {
		t.Error("angleWithin should tolerate a1 > a2")
	}
}

func TestStrokePath(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))
	segs := []pathSeg{
		lineSeg{10, 50, 90, 50},
	}
	strokePath(img, segs, 10)

	// On the line, centered in the stroke width.
	if a := img.RGBAAt(50, 50).A; a < 250 {
		t.Errorf("on-stroke alpha = %d, want ~255", a)
	}
	// Far from the line.
	if a := img.RGBAAt(50, 5).A; a != 0 {
		t.Errorf("off-stroke alpha = %d, want 0", a)
	}
}

func TestLineSegDistance(t *testing.T) {
	s := lineSeg{0, 0, 10, 0}
	if d := s.distance(5, 0); d != 0 {
		t.Errorf("distance to point on segment = %v, want 0", d)
	}
	if d := s.distance(5, 3); d != 3 {
		t.Errorf("distance to point above midpoint = %v, want 3", d)
	}
	// Beyond the segment's end: distance to the nearest endpoint, not
	// the infinite line.
	if d := s.distance(15, 0); d != 5 {
		t.Errorf("distance beyond segment end = %v, want 5", d)
	}
}
