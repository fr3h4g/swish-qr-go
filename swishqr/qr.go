package swishqr

import "fmt"

// This file is a from-scratch, dependency-free QR Code encoder (replacing
// the previous github.com/skip2/go-qrcode dependency), so the whole
// swishqr package only needs the Go standard library.
//
// It only ever encodes in QR "byte mode" at error-correction level H
// ("Highest"), which is all this package needs. Byte mode is less
// bit-efficient than numeric/alphanumeric mode for purely-numeric or
// purely-uppercase input, so QR codes produced here may end up a version
// or two larger than a fully mode-optimizing encoder would produce for
// the same text — a deliberate simplification in exchange for a much
// smaller, easier-to-verify implementation.
//
// It was developed by porting the algorithm (GF(256) Reed-Solomon error
// correction, codeword interleaving, module placement, format/version
// info BCH codes, and mask-pattern penalty scoring) into a Python
// prototype first, and validating that prototype bit-for-bit against the
// well-established `qrcode` PyPI package across hundreds of randomized
// payloads (varying lengths/versions/block splits/UTF-8 content) plus
// dedicated tests of version 39-40 and >1000-byte payloads, before
// transliterating it here. See the package README for more detail.

// alignPos gives the alignment-pattern center coordinates for each QR
// version (index 0 = version 1), per ISO/IEC 18004 table E.1.
var alignPos = [40][]int{
	{},
	{6, 18}, {6, 22}, {6, 26}, {6, 30}, {6, 34},
	{6, 22, 38}, {6, 24, 42}, {6, 26, 46}, {6, 28, 50}, {6, 30, 54},
	{6, 32, 58}, {6, 34, 62}, {6, 26, 46, 66}, {6, 26, 48, 70}, {6, 26, 50, 74},
	{6, 30, 54, 78}, {6, 30, 56, 82}, {6, 30, 58, 86}, {6, 34, 62, 90}, {6, 28, 50, 72, 94},
	{6, 26, 50, 74, 98}, {6, 30, 54, 78, 102}, {6, 28, 54, 80, 106}, {6, 32, 58, 84, 110}, {6, 30, 58, 86, 114},
	{6, 34, 62, 90, 118}, {6, 26, 50, 74, 98, 122}, {6, 30, 54, 78, 102, 126}, {6, 26, 52, 78, 104, 130}, {6, 30, 56, 82, 108, 134},
	{6, 34, 60, 86, 112, 138}, {6, 30, 58, 86, 114, 142}, {6, 34, 62, 90, 118, 146}, {6, 30, 54, 78, 102, 126, 150}, {6, 24, 50, 76, 102, 128, 154},
	{6, 28, 54, 80, 106, 132, 158}, {6, 32, 58, 84, 110, 136, 162}, {6, 26, 54, 82, 110, 138, 166}, {6, 30, 58, 86, 114, 142, 170},
}

// rsBlockSpec describes one group of identically-sized Reed-Solomon
// blocks: count blocks, each `total` codewords long with `data` of those
// being the actual data codewords (the rest is that block's own EC).
type rsBlockSpec struct{ count, total, data int }

// rsBlocksH gives the block layout for each QR version at error
// correction level H ("Highest"), taken verbatim from ISO/IEC 18004
// table 9 (the same table published inside the well-known `qrcode`
// Python package as RS_BLOCK_TABLE, used here purely as a reference to
// transcribe from — there is no runtime dependency on it).
var rsBlocksH = map[int][]rsBlockSpec{
	1: {{1, 26, 9}}, 2: {{1, 44, 16}}, 3: {{2, 35, 13}}, 4: {{4, 25, 9}},
	5: {{2, 33, 11}, {2, 34, 12}}, 6: {{4, 43, 15}}, 7: {{4, 39, 13}, {1, 40, 14}},
	8: {{4, 40, 14}, {2, 41, 15}}, 9: {{4, 36, 12}, {4, 37, 13}}, 10: {{6, 43, 15}, {2, 44, 16}},
	11: {{3, 36, 12}, {8, 37, 13}}, 12: {{7, 42, 14}, {4, 43, 15}}, 13: {{12, 33, 11}, {4, 34, 12}},
	14: {{11, 36, 12}, {5, 37, 13}}, 15: {{11, 36, 12}, {7, 37, 13}}, 16: {{3, 45, 15}, {13, 46, 16}},
	17: {{2, 42, 14}, {17, 43, 15}}, 18: {{2, 42, 14}, {19, 43, 15}}, 19: {{9, 39, 13}, {16, 40, 14}},
	20: {{15, 43, 15}, {10, 44, 16}}, 21: {{19, 46, 16}, {6, 47, 17}}, 22: {{34, 37, 13}},
	23: {{16, 45, 15}, {14, 46, 16}}, 24: {{30, 46, 16}, {2, 47, 17}}, 25: {{22, 45, 15}, {13, 46, 16}},
	26: {{33, 46, 16}, {4, 47, 17}}, 27: {{12, 45, 15}, {28, 46, 16}}, 28: {{11, 45, 15}, {31, 46, 16}},
	29: {{19, 45, 15}, {26, 46, 16}}, 30: {{23, 45, 15}, {25, 46, 16}}, 31: {{23, 45, 15}, {28, 46, 16}},
	32: {{19, 45, 15}, {35, 46, 16}}, 33: {{11, 45, 15}, {46, 46, 16}}, 34: {{59, 46, 16}, {1, 47, 17}},
	35: {{22, 45, 15}, {41, 46, 16}}, 36: {{2, 45, 15}, {64, 46, 16}}, 37: {{24, 45, 15}, {46, 46, 16}},
	38: {{42, 45, 15}, {32, 46, 16}}, 39: {{10, 45, 15}, {67, 46, 16}}, 40: {{20, 45, 15}, {61, 46, 16}},
}

func charCountBits(version int) int {
	if version <= 9 {
		return 8
	}
	return 16 // byte mode, versions 10-40
}

func dataCodewords(version int) int {
	total := 0
	for _, spec := range rsBlocksH[version] {
		total += spec.count * spec.data
	}
	return total
}

func matrixSize(version int) int { return version*4 + 17 }

func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// ---- bit buffer ----

type bitBuf struct {
	bits []bool
}

func (b *bitBuf) append(val uint32, n int) {
	for i := n - 1; i >= 0; i-- {
		b.bits = append(b.bits, (val>>uint(i))&1 == 1)
	}
}

func (b *bitBuf) len() int { return len(b.bits) }

func (b *bitBuf) toBytes() []byte {
	out := make([]byte, (len(b.bits)+7)/8)
	for i, bit := range b.bits {
		if bit {
			out[i/8] |= 0x80 >> uint(i%8)
		}
	}
	return out
}

func encodeSegment(data []byte, version int) *bitBuf {
	bb := &bitBuf{}
	bb.append(0b0100, 4) // byte mode indicator
	bb.append(uint32(len(data)), charCountBits(version))
	for _, by := range data {
		bb.append(uint32(by), 8)
	}
	return bb
}

func chooseVersion(data []byte) (int, error) {
	for v := 1; v <= 40; v++ {
		header := 4 + charCountBits(v)
		neededBits := header + 8*len(data)
		neededBytes := (neededBits + 7) / 8
		if neededBytes <= dataCodewords(v) {
			return v, nil
		}
	}
	return 0, fmt.Errorf("swishqr: data too long for any QR version")
}

func addTerminatorAndPad(bb *bitBuf, version int) {
	capacityBits := dataCodewords(version) * 8

	term := 4
	if rem := capacityBits - bb.len(); rem < term {
		term = rem
	}
	bb.append(0, term)

	for bb.len()%8 != 0 {
		bb.append(0, 1)
	}

	pad := [2]uint32{0xEC, 0x11}
	for i := 0; bb.len() < capacityBits; i++ {
		bb.append(pad[i%2], 8)
	}
}

// ---- Reed-Solomon over GF(256), QR's primitive polynomial 0x11D ----

var gfExp [512]int
var gfLog [256]int

func init() {
	x := 1
	for i := 0; i < 255; i++ {
		gfExp[i] = x
		gfLog[x] = i
		x <<= 1
		if x&0x100 != 0 {
			x ^= 0x11D
		}
	}
	for i := 255; i < 512; i++ {
		gfExp[i] = gfExp[i-255]
	}
}

func gfMul(a, b int) int {
	if a == 0 || b == 0 {
		return 0
	}
	return gfExp[gfLog[a]+gfLog[b]]
}

// rsGeneratorPoly builds the degree-`degree` Reed-Solomon generator
// polynomial, ascending order (index 0 = constant term):
//
//	gen(x) = product_{i=0}^{degree-1} (x + a^i)
func rsGeneratorPoly(degree int) []int {
	poly := []int{1}
	for i := 0; i < degree; i++ {
		next := make([]int, len(poly)+1)
		for j, c := range poly {
			next[j+1] ^= c                    // poly * x: shift up one degree
			next[j] ^= gfMul(c, gfExp[i])     // poly * a^i: scale, same degree
		}
		poly = next
	}
	return poly
}

// rsEncode returns the ecLen error-correction codewords for data, via
// polynomial long division in GF(256).
func rsEncode(data []byte, ecLen int) []byte {
	gen := rsGeneratorPoly(ecLen)
	// Reverse to descending order (gen[0] = leading coefficient = 1).
	for l, r := 0, len(gen)-1; l < r; l, r = l+1, r-1 {
		gen[l], gen[r] = gen[r], gen[l]
	}

	res := make([]byte, len(data)+ecLen)
	copy(res, data)
	for i := 0; i < len(data); i++ {
		factor := int(res[i])
		if factor == 0 {
			continue
		}
		for j := 0; j < len(gen); j++ {
			res[i+j] ^= byte(gfMul(gen[j], factor))
		}
	}
	return res[len(data):]
}

// buildCodewords splits data into this version's RS blocks, computes
// each block's error-correction codewords, and interleaves everything
// (all blocks' data codewords column-by-column, then all blocks' EC
// codewords column-by-column) per the QR spec.
func buildCodewords(data []byte, version int) []byte {
	specs := rsBlocksH[version]

	var dataBlocks, ecBlocks [][]byte
	pos := 0
	for _, spec := range specs {
		for i := 0; i < spec.count; i++ {
			block := data[pos : pos+spec.data]
			pos += spec.data
			dataBlocks = append(dataBlocks, block)
			ecBlocks = append(ecBlocks, rsEncode(block, spec.total-spec.data))
		}
	}

	var out []byte
	maxD := 0
	for _, b := range dataBlocks {
		maxD = max(maxD, len(b))
	}
	for i := 0; i < maxD; i++ {
		for _, b := range dataBlocks {
			if i < len(b) {
				out = append(out, b[i])
			}
		}
	}
	maxE := 0
	for _, b := range ecBlocks {
		maxE = max(maxE, len(b))
	}
	for i := 0; i < maxE; i++ {
		for _, b := range ecBlocks {
			if i < len(b) {
				out = append(out, b[i])
			}
		}
	}
	return out
}

// ---- matrix construction ----

// qrMatrix holds one QR version's module grid while it's being built:
// module[y][x] is the (row, column)-indexed bit value, and isFunction
// marks cells that are part of a fixed pattern (finder, timing,
// alignment, format/version info) rather than actual data, so the data
// placement and masking steps know to skip them.
type qrMatrix struct {
	n          int
	version    int
	module     [][]bool
	isFunction [][]bool
}

func newQRMatrix(version int) *qrMatrix {
	n := matrixSize(version)
	module := make([][]bool, n)
	isFunction := make([][]bool, n)
	for i := range module {
		module[i] = make([]bool, n)
		isFunction[i] = make([]bool, n)
	}
	return &qrMatrix{n: n, version: version, module: module, isFunction: isFunction}
}

func (m *qrMatrix) setFn(x, y int, v bool) {
	m.module[y][x] = v
	m.isFunction[y][x] = true
}

// drawFinder paints one of the three 7x7 finder patterns (dark 3x3 core,
// white ring, dark outer border) plus its 1-module white separator,
// with its top-left corner at (cx, cy).
func (m *qrMatrix) drawFinder(cx, cy int) {
	for dy := 0; dy < 7; dy++ {
		for dx := 0; dx < 7; dx++ {
			ring := max(absInt(dx-3), absInt(dy-3))
			m.setFn(cx+dx, cy+dy, ring != 2)
		}
	}
	for d := -1; d < 8; d++ {
		pts := [4][2]int{{cx + d, cy - 1}, {cx + d, cy + 7}, {cx - 1, cy + d}, {cx + 7, cy + d}}
		for _, p := range pts {
			x, y := p[0], p[1]
			if x >= 0 && x < m.n && y >= 0 && y < m.n {
				m.setFn(x, y, false)
			}
		}
	}
}

func (m *qrMatrix) drawTiming() {
	for i := 8; i < m.n-8; i++ {
		val := i%2 == 0
		if !m.isFunction[6][i] {
			m.setFn(i, 6, val)
		}
		if !m.isFunction[i][6] {
			m.setFn(6, i, val)
		}
	}
}

func (m *qrMatrix) drawAlignment() {
	positions := alignPos[m.version-1]
	for _, cy := range positions {
		for _, cx := range positions {
			// Skip positions that would overlap a finder pattern.
			if (cx < 8 && cy < 8) || (cx < 8 && cy >= m.n-8) || (cx >= m.n-8 && cy < 8) {
				continue
			}
			for dy := -2; dy <= 2; dy++ {
				for dx := -2; dx <= 2; dx++ {
					ring := max(absInt(dx), absInt(dy))
					m.setFn(cx+dx, cy+dy, ring != 1)
				}
			}
		}
	}
}

// reserveFormatAndVersionAreas marks (without yet writing real values
// into) the cells that will later hold format info (both copies) and,
// for version >= 7, version info, so the data-placement zigzag skips
// them.
func (m *qrMatrix) reserveFormatAndVersionAreas() {
	for i := 0; i < 9; i++ {
		if i != 6 {
			m.setFn(i, 8, false)
			m.setFn(8, i, false)
		}
	}
	m.setFn(8, 8, false)
	for i := 0; i < 8; i++ {
		m.setFn(m.n-1-i, 8, false)
	}
	for i := 0; i < 7; i++ {
		m.setFn(8, m.n-1-i, false)
	}

	if m.version >= 7 {
		for y := 0; y < 6; y++ {
			for x := m.n - 11; x < m.n-8; x++ {
				m.setFn(x, y, false)
			}
		}
		for x := 0; x < 6; x++ {
			for y := m.n - 11; y < m.n-8; y++ {
				m.setFn(x, y, false)
			}
		}
	}
}

// formatBits computes the 15-bit format info value (EC level H + the
// given mask pattern, BCH(15,5)-protected and XOR-masked), per
// ISO/IEC 18004 annex C.
func formatBits(maskID int) int {
	const ecLevelH = 0b10
	data5 := (ecLevelH << 3) | maskID
	val := data5 << 10
	const gen = 0b10100110111 // generator polynomial 0x537, degree 10
	for i := 4; i >= 0; i-- {
		if (val>>uint(10+i))&1 != 0 {
			val ^= gen << uint(i)
		}
	}
	combined := (data5 << 10) | val
	return combined ^ 0b101010000010010
}

// drawFormat writes the given 15-bit format info value into its two
// fixed locations (a vertical strip beside the top-left finder pattern,
// continuing down the bottom-left; and a horizontal strip below it,
// continuing right along the top-right), per ISO/IEC 18004 figure 25.
func (m *qrMatrix) drawFormat(target [][]bool, fbits int) {
	n := m.n
	bit := func(i int) bool { return (fbits>>uint(i))&1 != 0 }

	for i := 0; i < 15; i++ {
		switch {
		case i < 6:
			target[i][8] = bit(i)
		case i < 8:
			target[i+1][8] = bit(i)
		default:
			target[n-15+i][8] = bit(i)
		}
	}
	for i := 0; i < 15; i++ {
		switch {
		case i < 8:
			target[8][n-i-1] = bit(i)
		case i < 9:
			target[8][15-i] = bit(i)
		default:
			target[8][14-i] = bit(i)
		}
	}
}

// versionBits computes the 18-bit version info value (6-bit version
// number, BCH(18,6)-protected), used for version >= 7 only.
func versionBits(version int) int {
	val := version << 12
	const gen = 0b1111100100101 // generator polynomial 0x1F25, degree 12
	for i := 5; i >= 0; i-- {
		if (val>>uint(12+i))&1 != 0 {
			val ^= gen << uint(i)
		}
	}
	return (version << 12) | val
}

// drawVersion writes the two 3x6 version info blocks (bottom-left of
// the top-right finder, and top-right of the bottom-left finder), used
// for version >= 7 only.
func (m *qrMatrix) drawVersion() {
	vbits := versionBits(m.version)
	n := m.n
	for i := 0; i < 18; i++ {
		a, b := i/3, i%3
		v := (vbits>>uint(i))&1 != 0
		m.module[n-11+b][a] = v
		m.module[a][n-11+b] = v
	}
}

// placeData walks the module grid in the QR's characteristic zigzag
// (two columns at a time, right to left, snaking up and down and
// skipping the vertical timing-pattern column), writing one data bit
// into every non-function cell it encounters.
func (m *qrMatrix) placeData(codewords []byte) {
	var bits []bool
	for _, by := range codewords {
		for i := 7; i >= 0; i-- {
			bits = append(bits, (by>>uint(i))&1 != 0)
		}
	}

	bitIdx := 0
	upward := true
	for x := m.n - 1; x >= 1; x -= 2 {
		if x == 6 { // skip the timing column
			x--
		}
		for i := 0; i < m.n; i++ {
			y := i
			if upward {
				y = m.n - 1 - i
			}
			for _, xx := range [2]int{x, x - 1} {
				if m.isFunction[y][xx] {
					continue
				}
				val := false
				if bitIdx < len(bits) {
					val = bits[bitIdx]
				}
				m.module[y][xx] = val
				bitIdx++
			}
		}
		upward = !upward
	}
}

// ---- masking ----

var maskFuncs = [8]func(x, y int) bool{
	func(x, y int) bool { return (x+y)%2 == 0 },
	func(x, y int) bool { return y%2 == 0 },
	func(x, y int) bool { return x%3 == 0 },
	func(x, y int) bool { return (x+y)%3 == 0 },
	func(x, y int) bool { return (y/2+x/3)%2 == 0 },
	func(x, y int) bool { return (x*y)%2+(x*y)%3 == 0 },
	func(x, y int) bool { return ((x*y)%2+(x*y)%3)%2 == 0 },
	func(x, y int) bool { return ((x+y)%2+(x*y)%3)%2 == 0 },
}

func (m *qrMatrix) applyMask(maskID int) [][]bool {
	f := maskFuncs[maskID]
	out := make([][]bool, m.n)
	for y := range out {
		out[y] = append([]bool(nil), m.module[y]...)
		for x := 0; x < m.n; x++ {
			if !m.isFunction[y][x] && f(x, y) {
				out[y][x] = !out[y][x]
			}
		}
	}
	return out
}

// penaltyScore computes the QR mask-evaluation penalty (ISO/IEC 18004
// section 8.8.2): same-color runs, 2x2 blocks, finder-like patterns, and
// overall dark/light balance. Lower is better.
func penaltyScore(m [][]bool) int {
	n := len(m)
	score := 0

	for y := 0; y < n; y++ {
		run := 1
		for x := 1; x < n; x++ {
			if m[y][x] == m[y][x-1] {
				run++
			} else {
				if run >= 5 {
					score += 3 + (run - 5)
				}
				run = 1
			}
		}
		if run >= 5 {
			score += 3 + (run - 5)
		}
	}
	for x := 0; x < n; x++ {
		run := 1
		for y := 1; y < n; y++ {
			if m[y][x] == m[y-1][x] {
				run++
			} else {
				if run >= 5 {
					score += 3 + (run - 5)
				}
				run = 1
			}
		}
		if run >= 5 {
			score += 3 + (run - 5)
		}
	}

	for y := 0; y < n-1; y++ {
		for x := 0; x < n-1; x++ {
			v := m[y][x]
			if m[y][x+1] == v && m[y+1][x] == v && m[y+1][x+1] == v {
				score += 3
			}
		}
	}

	patt1 := []bool{true, false, true, true, true, false, true, false, false, false, false}
	patt2 := []bool{false, false, false, false, true, false, true, true, true, false, true}
	match := func(seq []bool, i int, patt []bool) bool {
		if i+len(patt) > len(seq) {
			return false
		}
		for k, p := range patt {
			if seq[i+k] != p {
				return false
			}
		}
		return true
	}
	for y := 0; y < n; y++ {
		row := m[y]
		for x := 0; x <= len(row)-11; x++ {
			if match(row, x, patt1) || match(row, x, patt2) {
				score += 40
			}
		}
	}
	for x := 0; x < n; x++ {
		col := make([]bool, n)
		for y := 0; y < n; y++ {
			col[y] = m[y][x]
		}
		for y := 0; y <= len(col)-11; y++ {
			if match(col, y, patt1) || match(col, y, patt2) {
				score += 40
			}
		}
	}

	dark := 0
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			if m[y][x] {
				dark++
			}
		}
	}
	pct := dark * 100 / (n * n)
	prev := (absInt(pct-50) / 5) * 5
	next := prev + 5
	score += min(absInt(prev-50), absInt(next-50)) / 5 * 10

	return score
}

// buildModules encodes text as a QR code at error correction level
// "Highest" (byte mode only) and returns the raw module matrix, indexed
// as modules[x][y], with no quiet zone. This mirrors the coordinate
// convention used throughout this package (and matches the reference
// Python implementation's qr._modules layout), which is why the
// clearing/drawing helpers index modules[x][y] rather than the more
// common [y][x].
func buildModules(text string) (modules [][]bool, size int, err error) {
	data := []byte(text)

	version, err := chooseVersion(data)
	if err != nil {
		return nil, 0, err
	}

	bb := encodeSegment(data, version)
	addTerminatorAndPad(bb, version)
	codewords := buildCodewords(bb.toBytes(), version)

	mat := newQRMatrix(version)
	mat.drawFinder(0, 0)
	mat.drawFinder(mat.n-7, 0)
	mat.drawFinder(0, mat.n-7)
	mat.drawTiming()
	mat.setFn(8, mat.n-8, true) // dark module
	mat.drawAlignment()
	mat.reserveFormatAndVersionAreas()
	if version >= 7 {
		mat.drawVersion()
	}
	mat.placeData(codewords)

	var best [][]bool
	bestScore := 0
	for maskID := 0; maskID < 8; maskID++ {
		candidate := mat.applyMask(maskID)
		mat.drawFormat(candidate, formatBits(maskID))
		s := penaltyScore(candidate)
		if best == nil || s < bestScore {
			bestScore = s
			best = candidate
		}
	}

	size = mat.n
	modules = make([][]bool, size)
	for x := 0; x < size; x++ {
		modules[x] = make([]bool, size)
		for y := 0; y < size; y++ {
			modules[x][y] = best[y][x]
		}
	}
	return modules, size, nil
}
