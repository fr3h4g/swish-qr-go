package swishqr

import (
	qrcode "github.com/skip2/go-qrcode"
)

// buildModules encodes text as a QR code at the "Highest" error-correction
// level and returns the raw module matrix, indexed as modules[x][y], with
// no quiet zone. This mirrors the coordinate convention used throughout
// this package (and matches the reference Python implementation's
// qr._modules layout), which is why the clearing/drawing helpers index
// modules[x][y] rather than the more common [y][x].
func buildModules(text string) (modules [][]bool, size int, err error) {
	qr, err := qrcode.New(text, qrcode.Highest)
	if err != nil {
		return nil, 0, err
	}

	// go-qrcode's Bitmap() includes a fixed 4-module quiet zone border on
	// every side; strip it to get the pure symbol.
	const quietZone = 4
	bitmap := qr.Bitmap()
	size = len(bitmap) - 2*quietZone

	modules = make([][]bool, size)
	for x := 0; x < size; x++ {
		modules[x] = make([]bool, size)
		for y := 0; y < size; y++ {
			// bitmap[row][col] == bitmap[y][x]
			modules[x][y] = bitmap[y+quietZone][x+quietZone]
		}
	}
	return modules, size, nil
}
