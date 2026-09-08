package domain

import (
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"
)

type PositionCloseReason string

const (
	PositionCloseNone       PositionCloseReason = ""
	PositionCloseTakeProfit PositionCloseReason = "TAKE_PROFIT"
	PositionCloseStopLoss   PositionCloseReason = "STOP_LOSS"
	PositionCloseTimeout    PositionCloseReason = "TIMEOUT"
)

// ExitMark records a bounded, quote-native liquidation observation. NetOutputAmount
// is quote-native output after the fixed paper network and priority estimates.
type ExitMark struct {
	Quote           QuoteEvidence
	QuoteHash       string
	NetOutputAmount uint64
	FeeEstimate     uint64
	ReturnBPS       int64
}

type ExitPolicy struct {
	TakeProfitBPS   int64
	StopLossBPS     int64
	MaxHoldDuration time.Duration
}

type ExitDecision struct{ Reason PositionCloseReason }

func NewExitMark(position PaperPosition, quote QuoteEvidence, networkFeeMicros, priorityFeeMicros int64) (ExitMark, error) {
	if err := quote.Validate(); err != nil {
		return ExitMark{}, fmt.Errorf("validate exit quote: %w", err)
	}
	if position.State != PositionOpen || strings.TrimSpace(position.QuoteMint) == "" || strings.TrimSpace(position.MintAddress) == "" || position.EntryInputAmount == 0 || position.TokenQuantity == 0 || position.EntryNetworkFeeMicros < 0 || position.EntryPriorityFeeMicros < 0 || networkFeeMicros < 0 || priorityFeeMicros < 0 {
		return ExitMark{}, errors.New("open paper position is incomplete")
	}
	if quote.InputMint != position.MintAddress || quote.OutputMint != position.QuoteMint || quote.InAmount != position.TokenQuantity {
		return ExitMark{}, errors.New("exit quote does not liquidate the full position into its entry quote mint")
	}

	quote = quote.clone()
	hash, err := quote.Hash()
	if err != nil {
		return ExitMark{}, fmt.Errorf("hash exit quote: %w", err)
	}
	exitFees, err := sumNonNegativeFees(networkFeeMicros, priorityFeeMicros)
	if err != nil {
		return ExitMark{}, err
	}
	entryFees, err := sumNonNegativeFees(position.EntryNetworkFeeMicros, position.EntryPriorityFeeMicros)
	if err != nil {
		return ExitMark{}, err
	}
	netOutput := uint64(0)
	if quote.OutAmount > exitFees {
		netOutput = quote.OutAmount - exitFees
	}
	returnBPS, err := returnBPS(netOutput, position.EntryInputAmount, entryFees)
	if err != nil {
		return ExitMark{}, err
	}
	return ExitMark{Quote: quote, QuoteHash: hash, NetOutputAmount: netOutput, FeeEstimate: exitFees, ReturnBPS: returnBPS}, nil
}

func (policy ExitPolicy) Decide(now time.Time, position PaperPosition, mark ExitMark) (ExitDecision, error) {
	if policy.TakeProfitBPS <= 0 || policy.StopLossBPS <= 0 || policy.MaxHoldDuration <= 0 || now.IsZero() || position.OpenedAt.IsZero() || now.Before(position.OpenedAt) {
		return ExitDecision{}, errors.New("exit policy is incomplete")
	}
	if mark.ReturnBPS >= policy.TakeProfitBPS {
		return ExitDecision{Reason: PositionCloseTakeProfit}, nil
	}
	if mark.ReturnBPS <= -policy.StopLossBPS {
		return ExitDecision{Reason: PositionCloseStopLoss}, nil
	}
	if now.Sub(position.OpenedAt) >= policy.MaxHoldDuration {
		return ExitDecision{Reason: PositionCloseTimeout}, nil
	}
	return ExitDecision{Reason: PositionCloseNone}, nil
}

func sumNonNegativeFees(fees ...int64) (uint64, error) {
	total := uint64(0)
	for _, fee := range fees {
		if fee < 0 || uint64(fee) > ^uint64(0)-total {
			return 0, errors.New("paper fee estimates are invalid")
		}
		total += uint64(fee)
	}
	return total, nil
}

func returnBPS(netOutput, entryInput, entryFees uint64) (int64, error) {
	cost := new(big.Int).SetUint64(entryInput)
	cost.Add(cost, new(big.Int).SetUint64(entryFees))
	if cost.Sign() <= 0 {
		return 0, errors.New("entry cost must be positive")
	}
	profit := new(big.Int).SetUint64(netOutput)
	profit.Sub(profit, cost)
	profit.Mul(profit, big.NewInt(10_000))
	profit.Quo(profit, cost)
	if !profit.IsInt64() {
		return 0, errors.New("return basis points are out of range")
	}
	return profit.Int64(), nil
}
