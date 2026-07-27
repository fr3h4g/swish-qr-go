# swish-qr-go

[![CI](https://github.com/fr3h4g/swish-qr-go/actions/workflows/ci.yml/badge.svg)](https://github.com/fr3h4g/swish-qr-go/actions/workflows/ci.yml)
[![Release](https://github.com/fr3h4g/swish-qr-go/actions/workflows/release.yml/badge.svg)](https://github.com/fr3h4g/swish-qr-go/actions/workflows/release.yml)
[![codecov](https://codecov.io/gh/fr3h4g/swish-qr-go/branch/main/graph/badge.svg)](https://codecov.io/gh/fr3h4g/swish-qr-go)

A Go port of [swish-qr-python](https://github.com/fr3h4g/swish-qr-python):
generates Swish-styled payment QR codes (the dotted, gradient, rounded-corner
style used by the Swish app) as SVG or PNG.

![Example](https://raw.githubusercontent.com/fr3h4g/swish-qr-go/main/example.png "Example")

## Install

```sh
go get github.com/fr3h4g/swish-qr-go/swishqr
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

Prebuilt binaries (Linux/macOS/Windows, amd64/arm64) are attached to each
[GitHub release](https://github.com/fr3h4g/swish-qr-go/releases) — see
"Releasing" below. `swishqr --version` prints the build's version.

## CI/CD

- **CI** (`.github/workflows/ci.yml`) runs on every pull request and on
  pushes to `main`: `go build`, `go vet`, `go test` across Go
  1.22–1.26 on Linux/macOS/Windows, plus a dedicated `-race` run and a
  `gofmt -l` check. The `-race` run also generates a coverage profile
  and uploads it to [Codecov](https://codecov.io) — for a public repo
  this works without any secret, but if uploads start failing, add a
  `CODECOV_TOKEN` repo secret from your Codecov project settings (the
  workflow step already reads `secrets.CODECOV_TOKEN` and won't fail
  the build if it's unset, via `fail_ci_if_error: false`).
- **Release** (`.github/workflows/release.yml`) runs when a `vX.Y.Z` tag
  is pushed: it re-runs the test suite, then uses
  [GoReleaser](https://goreleaser.com) (config in `.goreleaser.yaml`) to
  cross-compile `cmd/swishqr` for Linux/macOS/Windows (amd64 + arm64,
  Windows amd64 only) and publish the archives, checksums, and an
  auto-generated changelog to a GitHub release.

### Releasing

```sh
git tag v0.1.0
git push origin v0.1.0
```

That's it — the tag push triggers the release workflow. To verify the
GoReleaser config locally before tagging (no publishing):

```sh
go install github.com/goreleaser/goreleaser/v2@latest
goreleaser check
goreleaser release --snapshot --clean
```

The `release.github.owner`/`name` in `.goreleaser.yaml` and the badge
URLs above assume this repo lives at `github.com/fr3h4g/swish-qr-go` —
update both if you host it elsewhere (see the module path note under
Install).

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
  *_test.go      - tests
cmd/swishqr/         - CLI
.github/workflows/   - CI (ci.yml) + release (release.yml)
.goreleaser.yaml     - cross-platform release build config
```
