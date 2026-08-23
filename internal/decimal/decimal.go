// Package decimal converts decimal strings to exact integer base units.
package decimal

import (
	"fmt"
	"math/big"
	"strings"
)

// ParseUnits converts value into an integer with decimals fractional digits.
// It rejects values that cannot be represented exactly instead of rounding.
func ParseUnits(value string, decimals int) (*big.Int, error) {
	if decimals < 0 {
		return nil, fmt.Errorf("decimal places must not be negative")
	}

	original := value
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, fmt.Errorf("invalid decimal %q", original)
	}

	negative := false
	if value[0] == '+' || value[0] == '-' {
		negative = value[0] == '-'
		value = value[1:]
	}
	if value == "" || strings.Count(value, ".") > 1 {
		return nil, fmt.Errorf("invalid decimal %q", original)
	}

	whole, fraction, found := strings.Cut(value, ".")
	if !found {
		fraction = ""
	}
	if whole == "" {
		whole = "0"
	}
	if fraction == "" && found && value == "." {
		return nil, fmt.Errorf("invalid decimal %q", original)
	}
	if len(fraction) > decimals {
		return nil, fmt.Errorf("decimal %q has more than %d fractional digits", original, decimals)
	}
	if !digitsOnly(whole) || !digitsOnly(fraction) {
		return nil, fmt.Errorf("invalid decimal %q", original)
	}

	digits := strings.TrimLeft(whole+fraction+strings.Repeat("0", decimals-len(fraction)), "0")
	if digits == "" {
		digits = "0"
	}

	units, ok := new(big.Int).SetString(digits, 10)
	if !ok {
		return nil, fmt.Errorf("invalid decimal %q", original)
	}
	if negative && units.Sign() != 0 {
		units.Neg(units)
	}
	return units, nil
}

func digitsOnly(value string) bool {
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
