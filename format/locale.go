// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package format

import (
	"errors"
	"strconv"
	"strings"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/number"
)

// Formatter displays exact numbers in a caller-selected locale. It is immutable
// and safe for concurrent use. Its zero value uses the root locale.
type Formatter struct {
	tag language.Tag
}

// NewFormatter accepts a canonical ASCII BCP 47 tag of 1–63 bytes recognized by
// the pinned locale parser. Locale data supplies normal language/region fallback.
// The caller chooses a default before construction; process settings are ignored.
func NewFormatter(locale string) (Formatter, error) {
	if len(locale) == 0 || len(locale) > 63 {
		return Formatter{}, errors.New("format: locale must contain 1 to 63 ASCII bytes")
	}

	for i := range len(locale) {
		if locale[i] > 127 {
			return Formatter{}, errors.New("format: locale must be ASCII")
		}
	}

	tag, err := language.Parse(locale)
	if err != nil || tag.String() != locale {
		return Formatter{}, errors.New("format: locale must be a recognized canonical BCP 47 tag")
	}

	return Formatter{tag: tag}, nil
}

// Integer formats a signed 64-bit value with localized grouping and no fraction.
func (f Formatter) Integer(value int64) string {
	digits := strconv.FormatInt(value, 10)
	negative := strings.HasPrefix(digits, "-")

	return f.display(strings.TrimPrefix(digits, "-"), "", negative)
}

// Decimal formats canonical decimal text without rounding or losing fractional
// zeros. It accepts at most 128 ASCII digits, excluding an optional minus sign
// and point. Leading zeros, except zero itself, and empty fractions are invalid.
// Negative zero becomes unsigned zero with the original scale. Errors return
// an empty result and never include the supplied input.
func (f Formatter) Decimal(value string) (string, error) {
	if len(value) == 0 || len(value) > 130 {
		return "", errors.New("format: decimal must contain 1 to 128 ASCII digits")
	}

	negative := strings.HasPrefix(value, "-")

	integer, fraction, point := strings.Cut(strings.TrimPrefix(value, "-"), ".")
	if len(integer) == 0 || len(integer) > 1 && integer[0] == '0' || point && len(fraction) == 0 {
		return "", errors.New("format: decimal requires a canonical integer and a nonempty fraction after a point")
	}

	digits := integer + fraction
	if len(digits) > 128 {
		return "", errors.New("format: decimal exceeds 128 digits")
	}

	zero := true

	for i := range len(digits) {
		if digits[i] < '0' || digits[i] > '9' {
			return "", errors.New("format: decimal must contain only ASCII digits, an optional minus sign and point")
		}

		zero = zero && digits[i] == '0'
	}

	return f.display(integer, fraction, negative && !zero), nil
}

// MinorUnits formats an exact signed amount at a caller-supplied scale from zero
// through nine. It preserves that scale and handles the minimum int64 value.
// Currency identity and arithmetic remain the caller's responsibility.
func (f Formatter) MinorUnits(value int64, scale int) (string, error) {
	if scale < 0 || scale > 9 {
		return "", errors.New("format: minor-unit scale must be between zero and nine")
	}

	negative := value < 0

	digits := strings.TrimPrefix(strconv.FormatInt(value, 10), "-")
	if len(digits) <= scale {
		digits = strings.Repeat("0", scale+1-len(digits)) + digits
	}

	point := len(digits) - scale

	return f.display(digits[:point], digits[point:], negative), nil
}

func (f Formatter) display(integer, fraction string, negative bool) string {
	printer := message.NewPrinter(f.tag)
	digits := []rune(printer.Sprintf("%v", number.Decimal(1234567890, number.NoSeparator())))
	sign := 1

	if negative {
		sign = -1
	}

	// x/text has no public arbitrary-decimal input. A padded integer supplies
	// its grouping, affixes and fraction separator; replace only digit slots.
	pattern := printer.Sprintf("%v", number.Decimal(sign,
		number.MinIntegerDigits(len(integer)), number.Scale(len(fraction))))
	value := integer + fraction
	position := 0

	var result strings.Builder

	for _, r := range pattern {
		if r == digits[9] || r == digits[0] {
			result.WriteRune(digits[(int(value[position]-'0')+9)%10])
			position++
		} else {
			result.WriteRune(r)
		}
	}

	return result.String()
}
