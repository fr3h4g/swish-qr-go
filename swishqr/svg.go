package swishqr

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
)

// generateCorners builds the three gradient-stroked rounded corner
// accents (the "swish" look) that sit over the QR finder patterns.
func generateCorners(size int) string {
	var b strings.Builder
	fmt.Fprint(&b, `<path fill="none" stroke="url(#gradient)" stroke-linejoin="round" stroke-width="1.0" `+
		`d="M 4.5 10.5 v -0.875 a 5.125 5.125 0 0 1 5.125 -5.125 h 0.875 v 6 Z"/>`)
	fmt.Fprint(&b, `<path fill="url(#gradient)" d="M 6 9.125 a 3 3 0 0 1 3 -3 v 3 Z"/>`)
	fmt.Fprintf(&b, `<path fill="none" stroke="url(#gradient)" stroke-linejoin="round" stroke-width="1.0" `+
		`d="M 4.5 %d.5 v 0.875 a 5.125 5.125 0 0 0 5.125 5.125 h 0.875 v -6 Z"/>`, size-3)
	fmt.Fprintf(&b, `<path fill="url(#gradient)" d="M 6 %d.875 a 3 3 0 0 0 3 3 v -3 Z"/>`, size-2)
	fmt.Fprintf(&b, `<path fill="none" stroke="url(#gradient)" stroke-linejoin="round" stroke-width="1.0" `+
		`d="M %d.5 10.5 v -0.875 a 5.125 5.125 0 0 0 -5.125 -5.125 h -0.875 v 6 Z"/>`, size+3)
	fmt.Fprintf(&b, `<path fill="url(#gradient)" d="M %d 9.125 a 3 3 0 0 0 -3 -3 v 3 Z"/>`, size+2)
	return b.String()
}

// generateLogoUse places the embedded Swish logo, scaled to the QR size.
func generateLogoUse(size int) string {
	scale := 0.00083643122676579925 * float64(size+8)
	return fmt.Sprintf(`<use x="490" y="490" transform="scale(%s)" xlink:href="#w"/>`, formatFloat(scale))
}

// logoImageDef embeds the Swish logo as a base64 PNG <image> definition.
func logoImageDef() string {
	b64 := base64.StdEncoding.EncodeToString(logoPNGBytes)
	return fmt.Sprintf(`<image id="w" width="210" height="210" preserveAspectRatio="none" `+
		`xlink:href="data:image/png;base64,%s"/>`, b64)
}

func formatFloat(f float64) string {
	s := strconv.FormatFloat(f, 'f', -1, 64)
	return s
}

func makeSwishSVG(modules [][]bool, size, border int) ([]byte, error) {
	if border < 0 {
		return nil, fmt.Errorf("swishqr: border must be non-negative")
	}

	const dotSize = 30
	const dotSpace = 3
	imageSize := size*dotSize + size*dotSpace

	margin := border
	dimensions := fmt.Sprintf(` width="%d" height="%d"`, imageSize, imageSize)
	viewbox := fmt.Sprintf("%d %d %d %d", 4-margin, 4-margin, size+margin*2, size+margin*2)
	corners := generateCorners(size)
	logoUse := generateLogoUse(size)

	centerSide := pyRound(((0.2*float64(size))+3.4)/2)*2 - 1
	center := pyRound((float64(size) - float64(centerSide)) / 2)

	clearSquare(modules, 0, 0, 7)
	clearSquare(modules, size-7, 0, 7)
	clearSquare(modules, 0, size-7, 7)
	clearSquare(modules, center, center, centerSide)
	clearCorner(modules, size)

	var circles strings.Builder
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			if modules[x][y] {
				fmt.Fprintf(&circles, `<circle cx="%d.5" cy="%d.5" r="0.46"/>`, x+margin+4, y+margin+4)
			}
		}
	}

	var svg strings.Builder
	fmt.Fprintf(&svg, `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" viewBox="%s"%s>`, viewbox, dimensions)
	svg.WriteString("<defs>")
	svg.WriteString(logoImageDef())
	svg.WriteString(`<linearGradient id="gradient" x1="0%" x2="100%" y1="100%" y2="0%" gradientUnits="userSpaceOnUse">` +
		`<stop offset="0%" stop-color="#B43092"/><stop offset="100%" stop-color="#EF4123"/></linearGradient>`)
	svg.WriteString("</defs>")
	svg.WriteString(`<g fill="url(#gradient)">`)
	svg.WriteString(circles.String())
	svg.WriteString("</g>")
	svg.WriteString(corners)
	svg.WriteString(logoUse)
	svg.WriteString("</svg>")

	return []byte(svg.String()), nil
}
