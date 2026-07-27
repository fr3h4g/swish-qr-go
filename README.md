# swish-qr-go

A Go port of [swish-qr-python](https://github.com/fr3h4g/swish-qr-python):
generates Swish-styled payment QR codes (the dotted, gradient, rounded-corner
style used by the Swish app) as SVG or PNG.

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

	"github.com/fredrik/swish-qr-go/swishqr"
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

- QR encoding uses [`skip2/go-qrcode`](https://github.com/skip2/go-qrcode)
  instead of a ported copy of Nayuki's `qrcodegen`.
- PNG drawing uses [`fogleman/gg`](https://github.com/fogleman/gg)
  (anti-aliased) instead of Pillow (aliased), so the rounded corner
  accents look slightly smoother here.
- The diagonal brand gradient is computed directly (`t = (x+y)/(w+h-2)`)
  rather than by rotating and cropping a horizontal gradient — same
  visual result, simpler code.
- The three rounded "swish" corner accents are a geometric
  re-implementation of the original's arc/line drawing, rotated 90°
  for the other two corners, rather than a literal instruction-by-
  instruction port.

Functionally, payload format, validation rules, dot/finder-pattern
layout, and overall styling match the original.

## Package layout

```
swishqr/
  qr.go          - QR encoding -> raw module matrix
  clear.go       - finder-pattern / corner punch-outs
  swishqr.go     - FixAmount + GenerateSwishCode (public API)
  svg.go         - SVG renderer
  png.go         - PNG renderer
  cornermask.go  - rounded corner accent artwork
  gradient.go    - brand gradient
  imageutil.go   - resize/rotate/composite helpers
  logo.go/.png   - embedded Swish logo
cmd/swishqr/     - CLI
```
