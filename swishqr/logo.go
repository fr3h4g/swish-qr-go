package swishqr

import (
	"bytes"
	_ "embed"
	"image"
	"image/png"
)

//go:embed logo.png
var logoPNGBytes []byte

// swishLogo decodes the embedded Swish logo asset.
func swishLogo() image.Image {
	img, err := png.Decode(bytes.NewReader(logoPNGBytes))
	if err != nil {
		// Embedded asset, should never fail to decode.
		panic("swishqr: failed to decode embedded logo: " + err.Error())
	}
	return img
}
