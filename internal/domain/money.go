package domain

import (
	"errors"
	"math/big"
	"strconv"
	"strings"
)

const microsPerUSD int64 = 1_000_000

// USD stores a USD amount in integer microdollars.
type USD struct {
	Micros int64
}

func ParseUSD(value string) (USD, error) {
	value = strings.TrimSpace(value)
	if value == "" || strings.HasPrefix(value, "-") {
		return USD{}, errors.New("USD amount must be non-negative and non-empty")
	}

	parts := strings.Split(value, ".")
	if len(parts) > 2 || !isDigits(parts[0]) || (len(parts) == 2 && !isDigits(parts[1])) {
		return USD{}, errors.New("USD amount must be a decimal number")
	}

	whole, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || whole > (int64(^uint64(0)>>1)/microsPerUSD) {
		return USD{}, errors.New("USD amount is out of range")
	}

	fraction := ""
	if len(parts) == 2 {
		fraction = parts[1]
	}
	fraction = fraction + "000000"
	micros, _ := strconv.ParseInt(fraction[:6], 10, 64)
	if len(parts) == 2 && len(parts[1]) > 6 && parts[1][6] >= '5' {
		micros++
	}
	if micros == microsPerUSD {
		whole++
		micros = 0
	}
	if whole > (int64(^uint64(0)>>1)-micros)/microsPerUSD {
		return USD{}, errors.New("USD amount is out of range")
	}
	return USD{Micros: whole*microsPerUSD + micros}, nil
}

func (u *USD) UnmarshalText(value []byte) error {
	parsed, err := ParseUSD(string(value))
	if err != nil {
		return err
	}
	*u = parsed
	return nil
}

func (u USD) Add(v USD) USD { return USD{Micros: u.Micros + v.Micros} }
func (u USD) Sub(v USD) USD { return USD{Micros: u.Micros - v.Micros} }

func (u USD) ReturnBPS(cost USD) (int64, error) {
	if cost.Micros <= 0 {
		return 0, errors.New("cost must be positive")
	}
	profit := big.NewInt(u.Micros - cost.Micros)
	profit.Mul(profit, big.NewInt(10_000))
	profit.Quo(profit, big.NewInt(cost.Micros))
	if !profit.IsInt64() {
		return 0, errors.New("return basis points are out of range")
	}
	return profit.Int64(), nil
}

// RealizedPNLMicros converts a terminal fixed-point return into paper P&L.
func RealizedPNLMicros(notionalMicros, returnBPS int64) (int64, error) {
	if notionalMicros <= 0 {
		return 0, errors.New("paper notional must be positive")
	}
	pnl := big.NewInt(notionalMicros)
	pnl.Mul(pnl, big.NewInt(returnBPS))
	pnl.Quo(pnl, big.NewInt(10_000))
	if !pnl.IsInt64() {
		return 0, errors.New("realized paper P&L is out of range")
	}
	return pnl.Int64(), nil
}

func isDigits(value string) bool {
	if value == "" {
		return false
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}
