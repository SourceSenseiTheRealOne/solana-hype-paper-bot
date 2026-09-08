package domain

import (
	"errors"
	"math/big"
	"strings"
)

// ParsePercentageBPS converts a signed decimal percentage into integer basis
// points, rounding midpoint values away from zero without using floating point.
func ParsePercentageBPS(value string) (int64, error) {
	value = strings.TrimSpace(value)
	if !isSignedDecimal(value) {
		return 0, errors.New("percentage must be a decimal number")
	}
	percentage, ok := new(big.Rat).SetString(value)
	if !ok {
		return 0, errors.New("percentage must be a decimal number")
	}
	percentage.Mul(percentage, big.NewRat(100, 1))
	return roundedBasisPoints(percentage)
}

func roundedBasisPoints(value *big.Rat) (int64, error) {
	quotient, remainder := new(big.Int), new(big.Int)
	quotient.QuoRem(value.Num(), value.Denom(), remainder)
	twiceRemainder := new(big.Int).Lsh(new(big.Int).Abs(remainder), 1)
	if twiceRemainder.Cmp(value.Denom()) >= 0 {
		if value.Sign() < 0 {
			quotient.Sub(quotient, big.NewInt(1))
		} else {
			quotient.Add(quotient, big.NewInt(1))
		}
	}
	if !quotient.IsInt64() {
		return 0, errors.New("percentage basis points are out of range")
	}
	return quotient.Int64(), nil
}

func isSignedDecimal(value string) bool {
	if value == "" {
		return false
	}
	if value[0] == '+' || value[0] == '-' {
		value = value[1:]
	}
	parts := strings.Split(value, ".")
	return len(parts) <= 2 && isDigits(parts[0]) && (len(parts) == 1 || isDigits(parts[1]))
}
