# swish-qr-go

A Go port of [swish-qr-python](https://github.com/fr3h4g/swish-qr-python):
generates Swish-styled payment QR codes (the dotted, gradient, rounded-corner
style used by the Swish app) as SVG or PNG.

![Example](https://raw.githubusercontent.com/fr3h4g/swish-qr-go/main/example.png "Example")

## Install

```sh
go get github.com/fredrik/swish-qr-go/swishqr
```

(Rename the module path in `go.mod` to wherever you actually host this.)

## Usage

```go
package main

import (
 "os"

 "github.com/fr3h4g/swish-qr-go/swishqr"
)

func main() {
 svg, err := swishqr.GenerateSwishCode("0123456789", 100.99, "Test message!", "svg", false, false)
 if err != nil {
  panic(err)
 }
 os.WriteFile("example.svg", svg, 0o644)

 png, err := swishqr.GenerateSwishCode("0123456789", 100.99, "Test message!", "png", false, false)
 if err != nil {
  panic(err)
 }
 os.WriteFile("example.png", png, 0o644)
}
```

`GenerateSwishCode(payee, amount, message, format, editAmount, editMessage)`:

- `payee`: exactly 10 characters (a Swish number).
- `amount`: pass `0` to leave the amount blank/editable in the app.
- `message`: max 50 characters.
- `format`: `"svg"` or `"png"`.
- `editAmount` / `editMessage`: whether the payer can edit these fields
  after scanning.

## CLI

```sh
go run ./cmd/swishqr --format png 0123456789 100.99 "Test message!" out.png
```

## How it differs from the Python original

- QR encoding is a from-scratch, dependency-free encoder (byte mode
  only, error correction level "Highest"), vendored directly into
  `qr.go` rather than pulled in as a library — mirroring how the Python
  original vendors `qrcodegen.py` instead of depending on a QR package.
  Because it only ever uses byte mode, it's a bit less bit-efficient
  than a fully mode-optimizing encoder for purely-numeric or
  purely-uppercase text, so it may land on a version or two larger for
  the same content — a deliberate simplification.
- PNG drawing uses a small hand-rolled anti-aliased rasterizer
  (`raster.go`, signed-distance based) instead of Pillow, and image
  resizing uses a hand-rolled Catmull-Rom resampler (`imageutil.go`)
  instead of a resize library — so, like the Python original, this
  package has **zero third-party dependencies**, only the standard
  library.
- The diagonal brand gradient is computed directly (`t = (x+y)/(w+h-2)`)
  rather than by rotating and cropping a horizontal gradient — same
  visual result, simpler code.
- The three rounded "swish" corner accents are built from the exact
  same path geometry (coordinates and arc parameters) used by the SVG
  renderer, just converted from QR-module units into pixels, so the SVG
  and PNG outputs are guaranteed to match rather than being two
  independently hand-drawn shapes.

Functionally, payload format, validation rules, dot/finder-pattern
layout, and overall styling match the original.

## Package layout

```
swishqr/
  qr.go          - dependency-free QR encoder -> raw module matrix
  clear.go       - finder-pattern / corner punch-outs
  swishqr.go     - FixAmount + GenerateSwishCode (public API)
  svg.go         - SVG renderer
  png.go         - PNG renderer
  cornermask.go  - rounded corner accent geometry (shared with svg.go)
  raster.go      - hand-rolled anti-aliased rasterizer (circles/pie/stroke)
  gradient.go    - brand gradient
  imageutil.go   - Catmull-Rom resize + alpha compositing
  logo.go/.png   - embedded Swish logo
cmd/swishqr/     - CLI
```
