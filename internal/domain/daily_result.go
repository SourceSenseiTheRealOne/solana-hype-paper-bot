package domain

import "time"

// DailyResult is a persisted UTC-day, strategy-scoped paper-trading summary.
type DailyResult struct {
	UTCDate            time.Time
	StrategyVersion    string
	DailyAdmittedCount int
	RealizedPNLMicros  int64
}
