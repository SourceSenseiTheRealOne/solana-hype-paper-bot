package domain

import (
	"fmt"
	"math/big"
	"strings"
	"time"
)

const (
	RulePoolAge               = "pool_age"
	RuleLiquidity             = "liquidity"
	RuleFiveMinuteActivity    = "five_minute_activity"
	RuleFiveMinuteBuyShare    = "five_minute_buy_share"
	RuleFiveMinuteTurnover    = "five_minute_turnover"
	RuleFiveMinutePriceChange = "five_minute_price_change"

	RuleEntryPriceImpact = "entry_price_impact"
	RuleEntryRoute       = "entry_route"
	RuleExitRoute        = "exit_route"
	RuleMintAuthority    = "mint_authority_revoked"
	RuleFreezeAuthority  = "freeze_authority_revoked"
	RuleTokenProgram     = "token_program_extensions"
)

type TokenProgram string

const (
	TokenProgramLegacy TokenProgram = "spl_token"
	TokenProgram2022   TokenProgram = "token_2022"
)

type TokenExtension string

type TokenSafety struct {
	Program                TokenProgram
	MintAuthorityRevoked   bool
	FreezeAuthorityRevoked bool
	Extensions             []TokenExtension
}

type RouteLeg struct {
	AMMKey string
	Label  string
	Fee    QuoteFee
}

// QuoteFee is the native-unit fee returned by the quote route. It is never
// converted to a float or assumed to be denominated in the input asset.
type QuoteFee struct {
	Amount uint64
	Mint   string
}

type QuoteEvidence struct {
	ObservedAt     time.Time
	InputMint      string
	OutputMint     string
	InAmount       uint64
	OutAmount      uint64
	PriceImpactBPS int64
	PlatformFee    QuoteFee
	RoutePlan      []RouteLeg
}

// ExecutableQuote is a normalized route quote that can be evaluated without building a transaction.
type ExecutableQuote = QuoteEvidence

// TokenRiskSnapshot is normalized, read-only mint authority and extension evidence.
type TokenRiskSnapshot = TokenSafety

type MarketSnapshot struct {
	ObservedAt               time.Time
	LiquidityUSD             USD
	FiveMinuteTransactions   int
	FiveMinuteBuys           int
	FiveMinuteSells          int
	FiveMinuteVolumeUSD      USD
	FiveMinutePriceChangeBPS int64
}

type CandidateEvidence struct {
	MintAddress              string
	QuoteMint                string
	PoolCreatedAt            time.Time
	MarketObservedAt         time.Time
	ReceivedAt               time.Time
	LiquidityUSD             USD
	FiveMinuteTransactions   int
	FiveMinuteBuys           int
	FiveMinuteSells          int
	FiveMinuteVolumeUSD      USD
	FiveMinutePriceChangeBPS int64
	EntryInputAmount         uint64
	Token                    TokenSafety
	EntryQuote               QuoteEvidence
	ExitQuote                QuoteEvidence
}

type CandidatePolicy struct {
	MaxPoolAge                  time.Duration
	MinLiquidityUSD             USD
	MinFiveMinuteTransactions   int
	MinFiveMinuteBuyShareBPS    int64
	MinFiveMinuteTurnoverBPS    int64
	MinFiveMinutePriceChangeBPS int64
	MaxFiveMinutePriceChangeBPS int64

	MaxEntryPriceImpactBPS     int64
	AllowedToken2022Extensions []TokenExtension
}

type RuleResult struct {
	Code     string
	Passed   bool
	Observed string
	Limit    string
}

type CandidateEvaluation struct {
	Eligible bool
	Rules    []RuleResult
}

func (policy CandidatePolicy) Evaluate(now time.Time, evidence CandidateEvidence) CandidateEvaluation {
	poolAge := now.Sub(evidence.PoolCreatedAt)
	entryRouteExecutable := executableEntryRoute(evidence)
	exitRouteExecutable := executableExitRoute(evidence)
	momentumEnabled := policy.MinFiveMinuteBuyShareBPS > 0 || policy.MinFiveMinuteTurnoverBPS > 0 || policy.MinFiveMinutePriceChangeBPS != 0 || policy.MaxFiveMinutePriceChangeBPS != 0
	transactionCountsValid := evidence.FiveMinuteBuys >= 0 && evidence.FiveMinuteSells >= 0 && evidence.FiveMinuteBuys <= int(^uint(0)>>1)-evidence.FiveMinuteSells
	computedTransactions := 0
	if transactionCountsValid {
		computedTransactions = evidence.FiveMinuteBuys + evidence.FiveMinuteSells
	}
	activityConsistent := !momentumEnabled || (transactionCountsValid && evidence.FiveMinuteTransactions == computedTransactions)
	buyShareBPS, buyShareValid := ratioBPS(int64(evidence.FiveMinuteBuys), int64(computedTransactions))
	turnoverBPS, turnoverValid := ratioBPS(evidence.FiveMinuteVolumeUSD.Micros, evidence.LiquidityUSD.Micros)
	buySharePassed := policy.MinFiveMinuteBuyShareBPS <= 0 || (buyShareValid && buyShareBPS >= policy.MinFiveMinuteBuyShareBPS)
	turnoverPassed := policy.MinFiveMinuteTurnoverBPS <= 0 || (turnoverValid && turnoverBPS >= policy.MinFiveMinuteTurnoverBPS)
	priceChangeEnabled := policy.MinFiveMinutePriceChangeBPS != 0 || policy.MaxFiveMinutePriceChangeBPS != 0
	priceChangePassed := !priceChangeEnabled || (evidence.FiveMinutePriceChangeBPS >= policy.MinFiveMinutePriceChangeBPS && evidence.FiveMinutePriceChangeBPS <= policy.MaxFiveMinutePriceChangeBPS)
	rules := []RuleResult{
		{Code: RulePoolAge, Passed: !evidence.PoolCreatedAt.IsZero() && poolAge <= policy.MaxPoolAge, Observed: poolAge.String(), Limit: fmt.Sprintf("<=%s", policy.MaxPoolAge)},
		{Code: RuleLiquidity, Passed: evidence.LiquidityUSD.Micros >= policy.MinLiquidityUSD.Micros, Observed: formatUSD(evidence.LiquidityUSD), Limit: fmt.Sprintf(">=%s", formatUSD(policy.MinLiquidityUSD))},
		{Code: RuleFiveMinuteActivity, Passed: activityConsistent && evidence.FiveMinuteTransactions >= policy.MinFiveMinuteTransactions, Observed: fmt.Sprintf("%d", evidence.FiveMinuteTransactions), Limit: fmt.Sprintf(">=%d and equals buys+sells", policy.MinFiveMinuteTransactions)},
		{Code: RuleFiveMinuteBuyShare, Passed: buySharePassed, Observed: ratioObservation(buyShareBPS, buyShareValid), Limit: minimumBPSLimit(policy.MinFiveMinuteBuyShareBPS)},
		{Code: RuleFiveMinuteTurnover, Passed: turnoverPassed, Observed: ratioObservation(turnoverBPS, turnoverValid), Limit: minimumBPSLimit(policy.MinFiveMinuteTurnoverBPS)},
		{Code: RuleFiveMinutePriceChange, Passed: priceChangePassed, Observed: fmt.Sprintf("%dbps", evidence.FiveMinutePriceChangeBPS), Limit: priceChangeLimit(policy.MinFiveMinutePriceChangeBPS, policy.MaxFiveMinutePriceChangeBPS)},

		{Code: RuleEntryPriceImpact, Passed: entryRouteExecutable && evidence.EntryQuote.PriceImpactBPS >= 0 && evidence.EntryQuote.PriceImpactBPS <= policy.MaxEntryPriceImpactBPS, Observed: fmt.Sprintf("%dbps", evidence.EntryQuote.PriceImpactBPS), Limit: fmt.Sprintf("<=%dbps", policy.MaxEntryPriceImpactBPS)},
		{Code: RuleEntryRoute, Passed: entryRouteExecutable, Observed: routeObservation(entryRouteExecutable), Limit: "full-size executable route"},
		{Code: RuleExitRoute, Passed: exitRouteExecutable, Observed: routeObservation(exitRouteExecutable), Limit: "full-size reciprocal executable route"},
		{Code: RuleMintAuthority, Passed: evidence.Token.MintAuthorityRevoked, Observed: authorityObservation(evidence.Token.MintAuthorityRevoked), Limit: "revoked"},
		{Code: RuleFreezeAuthority, Passed: evidence.Token.FreezeAuthorityRevoked, Observed: authorityObservation(evidence.Token.FreezeAuthorityRevoked), Limit: "revoked"},
		{Code: RuleTokenProgram, Passed: policy.tokenProgramAllowed(evidence.Token), Observed: tokenObservation(evidence.Token), Limit: "legacy SPL or allowlisted Token-2022 extensions"},
	}

	eligible := true
	for _, rule := range rules {
		eligible = eligible && rule.Passed
	}
	return CandidateEvaluation{Eligible: eligible, Rules: rules}
}

func ratioBPS(numerator, denominator int64) (int64, bool) {
	if numerator < 0 || denominator <= 0 {
		return 0, false
	}
	value := new(big.Int).Mul(big.NewInt(numerator), big.NewInt(10_000))
	value.Quo(value, big.NewInt(denominator))
	if !value.IsInt64() {
		return 0, false
	}
	return value.Int64(), true
}

func ratioObservation(value int64, valid bool) string {
	if !valid {
		return "unavailable"
	}
	return fmt.Sprintf("%dbps", value)
}

func minimumBPSLimit(value int64) string {
	if value <= 0 {
		return "disabled"
	}
	return fmt.Sprintf(">=%dbps", value)
}

func priceChangeLimit(minimum, maximum int64) string {
	if minimum == 0 && maximum == 0 {
		return "disabled"
	}
	return fmt.Sprintf("%dbps..%dbps", minimum, maximum)
}

func formatUSD(amount USD) string {
	return fmt.Sprintf("$%d.%06d", amount.Micros/1_000_000, amount.Micros%1_000_000)
}

func routeObservation(executable bool) string {
	if executable {
		return "executable"
	}
	return "missing or mismatched"
}

func authorityObservation(revoked bool) string {
	if revoked {
		return "revoked"
	}
	return "active"
}

func tokenObservation(token TokenSafety) string {
	extensions := make([]string, len(token.Extensions))
	for index, extension := range token.Extensions {
		extensions[index] = string(extension)
	}
	if len(extensions) == 0 {
		return string(token.Program)
	}
	return fmt.Sprintf("%s:%s", token.Program, strings.Join(extensions, ","))
}

func executableEntryRoute(evidence CandidateEvidence) bool {
	quote := evidence.EntryQuote
	return evidence.EntryInputAmount > 0 && quote.InputMint == evidence.QuoteMint && quote.OutputMint == evidence.MintAddress && quote.InAmount == evidence.EntryInputAmount && quote.OutAmount > 0 && validRoutePlan(quote.RoutePlan)
}

func executableExitRoute(evidence CandidateEvidence) bool {
	entry := evidence.EntryQuote
	exit := evidence.ExitQuote
	return entry.OutAmount > 0 && exit.InputMint == evidence.MintAddress && exit.OutputMint == evidence.QuoteMint && exit.InAmount == entry.OutAmount && exit.OutAmount > 0 && validRoutePlan(exit.RoutePlan)
}

func validRoutePlan(routePlan []RouteLeg) bool {
	if len(routePlan) == 0 {
		return false
	}
	for _, leg := range routePlan {
		if leg.AMMKey == "" {
			return false
		}
	}
	return true
}

func (policy CandidatePolicy) tokenProgramAllowed(token TokenSafety) bool {
	switch token.Program {
	case TokenProgramLegacy:
		return len(token.Extensions) == 0
	case TokenProgram2022:
		allowed := make(map[TokenExtension]struct{}, len(policy.AllowedToken2022Extensions))
		for _, extension := range policy.AllowedToken2022Extensions {
			allowed[extension] = struct{}{}
		}
		for _, extension := range token.Extensions {
			if _, exists := allowed[extension]; !exists {
				return false
			}
		}
		return true
	default:
		return false
	}
}
