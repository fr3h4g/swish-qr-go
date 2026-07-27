// Command swishqr generates a Swish-styled QR code image.
//
// Usage:
//
//	swishqr [--format svg|png] <payee> <amount> <message> <filename>
package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/fredrik/swish-qr-go/swishqr"
)

// version is set at build time via -ldflags "-X main.version=...", e.g.
// from CI on tagged releases (see .goreleaser.yaml). Left as "dev" for
// plain `go build`/`go run`.
var version = "dev"

func main() {
	format := flag.String("format", "png", "image format, svg or png")
	editAmount := flag.Bool("edit-amount", false, "let the payer edit the amount in the Swish app")
	editMessage := flag.Bool("edit-message", false, "let the payer edit the message in the Swish app")
	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags] <payee> <amount> <message> <filename>\n\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		return
	}

	args := flag.Args()
	if len(args) != 4 {
		flag.Usage()
		os.Exit(2)
	}
	payee, amountArg, message, filename := args[0], args[1], args[2], args[3]

	if !isNumeric(payee) {
		fmt.Fprintf(os.Stderr, "Error: invalid value for PAYEE: %q is not a valid number.\n", payee)
		os.Exit(2)
	}
	if len(payee) != 10 {
		fmt.Fprintln(os.Stderr, "Error: wrong length of PAYEE, must be 10 digits.")
		os.Exit(2)
	}

	amount, err := strconv.ParseFloat(amountArg, 64)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: invalid value for AMOUNT: %q is not a valid number.\n", amountArg)
		os.Exit(2)
	}
	if amount < 1 || amount > 150000 {
		fmt.Fprintln(os.Stderr, "Error: wrong value in AMOUNT, allowed between 1 and 150000.")
		os.Exit(2)
	}
	if len(message) > 50 {
		fmt.Fprintln(os.Stderr, "Error: wrong length of MESSAGE, max length 50 characters.")
		os.Exit(2)
	}

	data, err := swishqr.GenerateSwishCode(payee, amount, message, *format, *editAmount, *editMessage)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: can't generate image: %v\n", err)
		os.Exit(2)
	}
	if err := os.WriteFile(filename, data, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "Error: can't write file: %v\n", err)
		os.Exit(2)
	}
}

func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
