package swishqr

import (
	"bytes"
	"strings"
	"testing"
)

func TestFixAmount(t *testing.T) {
	cases := []struct {
		amount float64
		want   string
	}{
		{0, ""},
		{1, "1,00"},
		{100.99, "100,99"},
		{99.999, "100,00"}, // rounds at 2 decimals
		{0.5, "0,50"},
	}
	for _, c := range cases {
		if got := FixAmount(c.amount); got != c.want {
			t.Errorf("FixAmount(%v) = %q, want %q", c.amount, got, c.want)
		}
	}
}

func TestGenerateSwishCodeValidation(t *testing.T) {
	cases := []struct {
		name    string
		payee   string
		message string
		format  string
	}{
		{"payee too short", "012345678", "", "svg"},
		{"payee too long", "01234567890", "", "svg"},
		{"message too long", "0123456789", strings.Repeat("x", 51), "svg"},
		{"unknown format", "0123456789", "", "jpg"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := GenerateSwishCode(c.payee, 100, c.message, c.format, false, false)
			if err == nil {
				t.Fatalf("expected an error, got nil")
			}
		})
	}
}

func TestGenerateSwishCodeOutputs(t *testing.T) {
	svg, err := GenerateSwishCode("0123456789", 100.99, "Test message!", "svg", false, false)
	if err != nil {
		t.Fatalf("svg: %v", err)
	}
	if !bytes.HasPrefix(svg, []byte("<svg")) {
		t.Errorf("svg output doesn't start with <svg: %.20q", svg)
	}

	png, err := GenerateSwishCode("0123456789", 100.99, "Test message!", "png", false, false)
	if err != nil {
		t.Fatalf("png: %v", err)
	}
	pngMagic := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}
	if !bytes.HasPrefix(png, pngMagic) {
		t.Errorf("png output doesn't start with the PNG signature")
	}

	// Format should be case-insensitive and tolerate surrounding
	// whitespace.
	if _, err := GenerateSwishCode("0123456789", 100.99, "", "  PNG  ", false, false); err != nil {
		t.Errorf("format %q: %v", "  PNG  ", err)
	}
	if _, err := GenerateSwishCode("0123456789", 100.99, "", "SVG", false, false); err != nil {
		t.Errorf("format %q: %v", "SVG", err)
	}

	// amount == 0 should still succeed (payer enters amount themselves).
	if _, err := GenerateSwishCode("0123456789", 0, "", "svg", true, true); err != nil {
		t.Errorf("zero amount with edit flags: %v", err)
	}
}

func TestPyRound(t *testing.T) {
	// Round-half-to-even, matching Python's round().
	cases := []struct {
		in   float64
		want int
	}{
		{0.5, 0},
		{1.5, 2},
		{2.5, 2},
		{-0.5, 0},
		{3.2, 3},
		{3.8, 4},
	}
	for _, c := range cases {
		if got := pyRound(c.in); got != c.want {
			t.Errorf("pyRound(%v) = %d, want %d", c.in, got, c.want)
		}
	}
}
