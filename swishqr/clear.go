package swishqr

// clearSquare blanks out a side x side block of modules whose top-left
// corner is at (x, y). Used to remove the QR finder patterns and the
// center area before overlaying the Swish logo and corner decorations.
func clearSquare(modules [][]bool, x, y, side int) {
	for yOff := 0; yOff < side; yOff++ {
		for xOff := 0; xOff < side; xOff++ {
			modules[x+xOff][y+yOff] = false
		}
	}
}

// clearCorner trims the sharp bottom-right finder-pattern corner so the
// rounded Swish corner accent can be drawn over it cleanly.
func clearCorner(modules [][]bool, size int) {
	for yOff := size - 6; yOff < size; yOff++ {
		for xOff := size - 1; xOff > size-5; xOff-- {
			if xOff > size-6+(size-yOff) {
				modules[xOff][yOff] = false
			}
		}
	}
}
