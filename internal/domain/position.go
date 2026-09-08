package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrNoExecutableRoute = errors.New("no executable route for the requested quote")

const (
	MaxConsecutiveNoRoutes = 5
	UnsellableReturnBPS    = -10_000
)

type PositionState string

const (
	PositionPending    PositionState = "PENDING"
	PositionOpen       PositionState = "OPEN"
	PositionClosing    PositionState = "CLOSING"
	PositionClosed     PositionState = "CLOSED"
	PositionUnsellable PositionState = "UNSELLABLE"
)

// PaperPosition is the durable, paper-only state returned by position storage.
type PaperPosition struct {
	ID                     int
	CandidateID            int
	State                  PositionState
	NotionalMicros         int64
	NoRouteCount           int
	QuoteMint              string
	MintAddress            string
	EntryPrice             string
	EntryInputAmount       uint64
	EntryNetworkFeeMicros  int64
	EntryPriorityFeeMicros int64
	TokenQuantity          uint64
	OpenedAt               time.Time
}

// EntryFill is an observed virtual fill derived only from a normalized read-only quote.
type EntryFill struct {
	Quote             QuoteEvidence
	QuoteHash         string
	EntryPrice        string
	TokenQuantity     uint64
	NetworkFeeMicros  int64
	PriorityFeeMicros int64
}

func NewEntryFill(quote QuoteEvidence, networkFeeMicros, priorityFeeMicros int64) (EntryFill, error) {
	if err := quote.Validate(); err != nil {
		return EntryFill{}, fmt.Errorf("validate entry quote: %w", err)
	}
	if networkFeeMicros < 0 || priorityFeeMicros < 0 {
		return EntryFill{}, errors.New("paper fee estimates must be non-negative")
	}
	quote = quote.clone()
	hash, err := quote.Hash()
	if err != nil {
		return EntryFill{}, fmt.Errorf("hash entry quote: %w", err)
	}
	return EntryFill{
		Quote:             quote,
		QuoteHash:         hash,
		EntryPrice:        fmt.Sprintf("%d/%d", quote.InAmount, quote.OutAmount),
		TokenQuantity:     quote.OutAmount,
		NetworkFeeMicros:  networkFeeMicros,
		PriorityFeeMicros: priorityFeeMicros,
	}, nil
}

func (quote QuoteEvidence) Validate() error {
	if quote.ObservedAt.IsZero() || strings.TrimSpace(quote.InputMint) == "" || strings.TrimSpace(quote.OutputMint) == "" || quote.InAmount == 0 || quote.OutAmount == 0 || quote.PriceImpactBPS < 0 {
		return errors.New("quote is incomplete")
	}
	if err := validateQuoteFee(quote.PlatformFee); err != nil {
		return fmt.Errorf("platform fee: %w", err)
	}
	if len(quote.RoutePlan) == 0 {
		return errors.New("quote route plan is empty")
	}
	for _, route := range quote.RoutePlan {
		if strings.TrimSpace(route.AMMKey) == "" {
			return errors.New("quote route leg is missing an AMM key")
		}
		if err := validateQuoteFee(route.Fee); err != nil {
			return fmt.Errorf("route fee: %w", err)
		}
	}
	return nil
}

func (quote QuoteEvidence) Hash() (string, error) {
	canonical := struct {
		ObservedAt     string     `json:"observed_at"`
		InputMint      string     `json:"input_mint"`
		OutputMint     string     `json:"output_mint"`
		InAmount       uint64     `json:"in_amount"`
		OutAmount      uint64     `json:"out_amount"`
		PriceImpactBPS int64      `json:"price_impact_bps"`
		PlatformFee    QuoteFee   `json:"platform_fee"`
		RoutePlan      []RouteLeg `json:"route_plan"`
	}{
		ObservedAt: quote.ObservedAt.UTC().Format(time.RFC3339Nano), InputMint: quote.InputMint, OutputMint: quote.OutputMint,
		InAmount: quote.InAmount, OutAmount: quote.OutAmount, PriceImpactBPS: quote.PriceImpactBPS,
		PlatformFee: quote.PlatformFee, RoutePlan: quote.RoutePlan,
	}
	encoded, err := json.Marshal(canonical)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:]), nil
}

func validateQuoteFee(fee QuoteFee) error {
	if fee.Amount > 0 && strings.TrimSpace(fee.Mint) == "" {
		return errors.New("positive fee is missing its mint")
	}
	return nil
}

func (quote QuoteEvidence) clone() QuoteEvidence {
	quote.RoutePlan = append([]RouteLeg(nil), quote.RoutePlan...)
	return quote
}
