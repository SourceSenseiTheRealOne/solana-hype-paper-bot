package application

import "errors"

// marketEvidenceUnavailable marks a candidate-specific market data failure
// that may be recorded safely while the bounded scan continues.
type marketEvidenceUnavailable interface {
	MarketEvidenceUnavailable()
}

type marketEvidenceUnavailableError struct {
	cause error
}

func newMarketEvidenceUnavailableError(cause error) error {
	return marketEvidenceUnavailableError{cause: cause}
}

func (marketEvidenceUnavailableError) Error() string {
	return "market evidence unavailable"
}

func (error marketEvidenceUnavailableError) Unwrap() error {
	return error.cause
}

func (marketEvidenceUnavailableError) MarketEvidenceUnavailable() {}

func isMarketEvidenceUnavailable(err error) bool {
	var unavailable marketEvidenceUnavailable
	return errors.As(err, &unavailable)
}
