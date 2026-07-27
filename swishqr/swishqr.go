// Package swishqr generates Swish-styled QR codes (SVG or PNG) for
// Swish payment requests, in the style used by the Swish mobile app.
package swishqr

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"
)

// FixAmount formats an amount the way the Swish payload expects: two
// decimals with a comma separator, or an empty string when amount is 0
// (an empty amount lets the payer enter it themselves in the app).
func FixAmount(amount float64) string {
	if amount == 0 {
		return ""
	}
	return strings.Replace(strconv.FormatFloat(amount, 'f', 2, 64), ".", ",", 1)
}

// GenerateSwishCode builds the Swish payload for payee/amount/message and
// renders it as a Swish-styled QR code. format must be "svg" or "png"
// (case-insensitive). editAmount/editMessage control whether the Swish
// app lets the payer change the amount/message after scanning.
func GenerateSwishCode(payee string, amount float64, message, format string, editAmount, editMessage bool) ([]byte, error) {
	format = strings.ToLower(strings.TrimSpace(format))
	if format != "svg" && format != "png" {
		return nil, fmt.Errorf("swishqr: unknown format %q", format)
	}
	if utf8.RuneCountInString(payee) != 10 {
		return nil, fmt.Errorf("swishqr: payee must be exactly 10 characters")
	}
	if utf8.RuneCountInString(message) > 50 {
		return nil, fmt.Errorf("swishqr: message too long, max 50 characters")
	}

	amountStr := FixAmount(amount)

	editMask := 0
	if editAmount {
		editMask += 2
	}
	if editMessage {
		editMask += 4
	}

	text := fmt.Sprintf("C%s;%s;%s;%d", payee, amountStr, message, editMask)

	modules, size, err := buildModules(text)
	if err != nil {
		return nil, fmt.Errorf("swishqr: encoding QR code: %w", err)
	}

	if format == "svg" {
		return makeSwishSVG(modules, size, 0)
	}
	return makeSwishPNG(modules, size, 0)
}

// pyRound mimics Python's round(): round-half-to-even.
func pyRound(x float64) int {
	return int(math.RoundToEven(x))
}
