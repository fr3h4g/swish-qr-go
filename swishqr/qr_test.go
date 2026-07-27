package swishqr

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
)

// ---- test-only QR decoder ----
//
// These helpers implement a from-scratch QR decoder used only by tests,
// to check that buildModules' output actually round-trips the way a
// real scanner would decode it: read format info to recover the mask,
// unmask the data area, reverse the zigzag placement, de-interleave the
// Reed-Solomon blocks (checking each block's parity), and parse the
// byte-mode segment back into text. This is a much stronger check than
// just asserting buildModules doesn't error, since it would catch
// placement/masking/interleaving bugs that a shape-only check wouldn't.
//
// The decode logic was first validated in a Python prototype against
// 300+ randomized payloads before being ported here.

func buildSkeleton(version int) *qrMatrix {
	mat := newQRMatrix(version)
	mat.drawFinder(0, 0)
	mat.drawFinder(mat.n-7, 0)
	mat.drawFinder(0, mat.n-7)
	mat.drawTiming()
	mat.setFn(8, mat.n-8, true)
	mat.drawAlignment()
	mat.reserveFormatAndVersionAreas()
	if version >= 7 {
		mat.drawVersion()
	}
	return mat
}

type testBitReader struct {
	data []byte
	pos  int
}

func (r *testBitReader) readBits(n int) int {
	v := 0
	for i := 0; i < n; i++ {
		byteIdx := r.pos / 8
		bitIdx := 7 - (r.pos % 8)
		bit := 0
		if byteIdx < len(r.data) && (r.data[byteIdx]>>uint(bitIdx))&1 == 1 {
			bit = 1
		}
		v = (v << 1) | bit
		r.pos++
	}
	return v
}

func decodeModules(modules [][]bool, n int) (string, error) {
	// modules is [x][y]; convert to row-major m[y][x] to match the
	// convention the encoder's internal qrMatrix uses.
	m := make([][]bool, n)
	for y := 0; y < n; y++ {
		m[y] = make([]bool, n)
		for x := 0; x < n; x++ {
			m[y][x] = modules[x][y]
		}
	}

	raw := 0
	for i := 0; i < 15; i++ {
		var v bool
		switch {
		case i < 6:
			v = m[i][8]
		case i < 8:
			v = m[i+1][8]
		default:
			v = m[n-15+i][8]
		}
		if v {
			raw |= 1 << uint(i)
		}
	}
	raw ^= 0b101010000010010
	data5 := raw >> 10
	maskID := data5 & 0b111

	version := (n - 17) / 4
	skeleton := buildSkeleton(version)
	f := maskFuncs[maskID]

	var bits []bool
	upward := true
	for x := n - 1; x >= 1; x -= 2 {
		if x == 6 {
			x--
		}
		for i := 0; i < n; i++ {
			y := i
			if upward {
				y = n - 1 - i
			}
			for _, xx := range [2]int{x, x - 1} {
				if skeleton.isFunction[y][xx] {
					continue
				}
				v := m[y][xx]
				if f(xx, y) {
					v = !v
				}
				bits = append(bits, v)
			}
		}
		upward = !upward
	}

	codewords := make([]byte, len(bits)/8)
	for i := range codewords {
		var b byte
		for k := 0; k < 8; k++ {
			b <<= 1
			if bits[i*8+k] {
				b |= 1
			}
		}
		codewords[i] = b
	}

	specs := rsBlocksH[version]
	type block struct{ data, ec []byte }
	var blocks []block
	for _, s := range specs {
		for i := 0; i < s.count; i++ {
			blocks = append(blocks, block{data: make([]byte, s.data), ec: make([]byte, s.total-s.data)})
		}
	}

	idx := 0
	maxData := 0
	for _, b := range blocks {
		maxData = max(maxData, len(b.data))
	}
	for i := 0; i < maxData; i++ {
		for bi := range blocks {
			if i < len(blocks[bi].data) {
				blocks[bi].data[i] = codewords[idx]
				idx++
			}
		}
	}
	maxEC := 0
	for _, b := range blocks {
		maxEC = max(maxEC, len(b.ec))
	}
	for i := 0; i < maxEC; i++ {
		for bi := range blocks {
			if i < len(blocks[bi].ec) {
				blocks[bi].ec[i] = codewords[idx]
				idx++
			}
		}
	}

	var data []byte
	for _, b := range blocks {
		gotEC := rsEncode(b.data, len(b.ec))
		for i := range gotEC {
			if gotEC[i] != b.ec[i] {
				return "", fmt.Errorf("swishqr test: reed-solomon parity mismatch in block")
			}
		}
		data = append(data, b.data...)
	}

	br := &testBitReader{data: data}
	mode := br.readBits(4)
	if mode != 0b0100 {
		return "", fmt.Errorf("swishqr test: unexpected mode indicator %04b", mode)
	}
	length := br.readBits(charCountBits(version))
	out := make([]byte, length)
	for i := range out {
		out[i] = byte(br.readBits(8))
	}
	return string(out), nil
}

// ---- actual tests ----

func TestBuildModulesRoundTrip(t *testing.T) {
	cases := []string{
		"",
		"x",
		"Hello, world!",
		"C0123456789;100,00;Test message!;0",
		"C0739316106;110,00;test;0",
		strings.Repeat("A", 49),
		strings.Repeat("1234567890", 5),
		"Ett meddelande med åäö and a euro sign €",
		"abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789",
	}
	for _, text := range cases {
		text := text
		t.Run(fmt.Sprintf("%.30q", text), func(t *testing.T) {
			modules, size, err := buildModules(text)
			if err != nil {
				t.Fatalf("buildModules(%q): %v", text, err)
			}
			if size < 21 || (size-17)%4 != 0 {
				t.Fatalf("size %d isn't a valid QR matrix size (4*version+17)", size)
			}
			got, err := decodeModules(modules, size)
			if err != nil {
				t.Fatalf("decodeModules: %v", err)
			}
			if got != text {
				t.Fatalf("round-trip mismatch: got %q, want %q", got, text)
			}
		})
	}
}

func TestBuildModulesRoundTripFuzz(t *testing.T) {
	rng := rand.New(rand.NewSource(2024))
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 !?;,.-_åäöÅÄÖ€"
	lengths := []int{0, 1, 2, 3, 5, 10, 20, 30, 50, 80, 120, 200}

	for i, n := range lengths {
		b := make([]rune, n)
		for j := range b {
			b[j] = rune(alphabet[rng.Intn(len(alphabet))])
		}
		text := string(b)

		modules, size, err := buildModules(text)
		if err != nil {
			t.Fatalf("case %d (len %d): buildModules: %v", i, n, err)
		}
		got, err := decodeModules(modules, size)
		if err != nil {
			t.Fatalf("case %d (len %d, %q): decodeModules: %v", i, n, text, err)
		}
		if got != text {
			t.Fatalf("case %d (len %d): round-trip mismatch: got %q, want %q", i, n, got, text)
		}
	}
}

func TestBuildModulesDataTooLong(t *testing.T) {
	// Comfortably longer than version 40 can hold in byte mode at EC
	// level H (a little over 1200 data codewords worst case).
	_, _, err := buildModules(strings.Repeat("x", 5000))
	if err == nil {
		t.Fatal("expected an error for data too long for any QR version, got nil")
	}
}

func TestFormatBitsRoundTrip(t *testing.T) {
	for maskID := 0; maskID < 8; maskID++ {
		fbits := formatBits(maskID)
		raw := fbits ^ 0b101010000010010
		data5 := raw >> 10
		gotMask := data5 & 0b111
		gotEC := (data5 >> 3) & 0b11
		if gotMask != maskID {
			t.Errorf("formatBits(%d): decoded mask = %d, want %d", maskID, gotMask, maskID)
		}
		if gotEC != 0b10 {
			t.Errorf("formatBits(%d): decoded EC level = %02b, want 10 (H)", maskID, gotEC)
		}
	}
}

func TestVersionBitsRoundTrip(t *testing.T) {
	for version := 7; version <= 40; version++ {
		vbits := versionBits(version)
		got := vbits >> 12
		if got != version {
			t.Errorf("versionBits(%d): top 6 bits decode to %d", version, got)
		}
	}
}

func TestChooseVersionAndDataCodewords(t *testing.T) {
	// Version 1, EC level H: RS block (1, 26, 9) -> 9 data codewords.
	if got := dataCodewords(1); got != 9 {
		t.Errorf("dataCodewords(1) = %d, want 9", got)
	}

	v, err := chooseVersion(nil)
	if err != nil || v != 1 {
		t.Errorf("chooseVersion(empty) = (%d, %v), want (1, nil)", v, err)
	}

	v, err = chooseVersion(make([]byte, dataCodewords(1)-3)) // leaves room for the 4-bit header
	if err != nil || v != 1 {
		t.Errorf("chooseVersion(small) = (%d, %v), want (1, nil)", v, err)
	}
}

func TestMatrixSize(t *testing.T) {
	for version := 1; version <= 40; version++ {
		want := version*4 + 17
		if got := matrixSize(version); got != want {
			t.Errorf("matrixSize(%d) = %d, want %d", version, got, want)
		}
	}
}
