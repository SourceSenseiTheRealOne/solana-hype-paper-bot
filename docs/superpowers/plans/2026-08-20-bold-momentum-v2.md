# Bold Momentum v2 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the approved `bold-momentum-v2` paper-only strategy with normalized five-minute momentum evidence, deterministic social and Hermes quality gates, version-scoped admission, and visible strategy attribution.

**Architecture:** Extend both bounded market adapters into one fixed-point evidence model, evaluate all market gates before any social/Hermes request, then enforce social and Hermes policies before atomic paper admission. Keep the current provider order, hard token/route checks, quotas, persistence model, and read-only dashboard boundary.

**Tech Stack:** Go, Ent/PostgreSQL, bounded HTTP JSON adapters, Vitest/React/TypeScript, Docker Compose.

## Global Constraints

- Remain paper-only: no wallet, secret key, signing, transaction building, submission, swap, order, or blockchain write path.
- Helius remains read-only; Jupiter remains `GET /swap/v1/quote` only.
- Preserve mint/freeze authority, token-extension, full-size entry-route, reciprocal-exit-route, second-quote, and atomic admission controls.
- Scan every 5 minutes; monitor every 30 seconds; no overlapping scans, retry loops, unbounded pagination, or unbounded result sets.
- Use exact strategy ID `bold-momentum-v2` and $100 virtual notional.
- Use 90m maximum age, $5,000 minimum liquidity, 20 minimum 5m transactions, 6,500 bps minimum buy share, 1,500 bps minimum turnover, and +200 to +6,000 bps 5m price change.
- Require social score 30, 3 unique authors, 2 original posts, 2 exact mint mentions, and no more than 1 warning post.
- Require Hermes `BUY`, confidence 70, hype quality 60, and manipulation probability at most 35.
- Use +5,000 bps take profit, -2,000 bps stop loss, and 45m timeout.
- Never expose raw social/provider/Hermes data or protected configuration.
- Do not read or print `.env.local`; deployment may use only named replacements.
- Do not stage, commit, push, reset, stash, or clean without separate approval.
- Preserve unrelated dirty parent and repository work.

## File Structure

- `internal/domain/percentage.go`: exact signed percentage-to-basis-point parsing.
- `internal/domain/policy.go`: normalized market evidence and deterministic market rules.
- `internal/domain/social_policy.go`: social aggregate and Hermes verdict policies.
- `internal/adapters/httpclient/{dexscreener,geckoterminal}.go`: source normalization.
- `internal/application/{evaluator,admission,production_scan_job}.go`: ordered gates and defense in depth.
- `internal/config/config.go`: bounded settings and defaults.
- `internal/adapters/postgres/candidate_repository.go`: durable normalized evidence.
- `internal/application/dashboard.go`, `internal/transport/httpapi/httpapi.go`, `dashboard/src/{api,App}.tsx`: safe strategy visibility.

---

### Task 1: Normalize bounded five-minute market evidence

**Files:**
- Create: `internal/domain/percentage.go`
- Create: `internal/domain/percentage_test.go`
- Modify: `internal/domain/policy.go:68-95`
- Modify: `internal/adapters/httpclient/dexscreener.go:130-179`
- Modify: `internal/adapters/httpclient/geckoterminal.go:70-95,168-192`
- Test: `internal/adapters/httpclient/dexscreener_test.go`
- Test: `internal/adapters/httpclient/geckoterminal_test.go`

**Interfaces:**
- Produces: `ParsePercentageBPS(string) (int64, error)`.
- Produces: unexported `roundedBasisPoints(*big.Rat) (int64, error)`.
- Produces: `MarketSnapshot{FiveMinuteBuys int, FiveMinuteSells int, FiveMinuteVolumeUSD USD, FiveMinutePriceChangeBPS int64}`.
- Preserves normalized `MarketSnapshot.FiveMinuteTransactions int` for compatibility and requires it to equal buys plus sells.

- [ ] **Step 1: Write fixed-point parser tests**

```go
func TestParsePercentageBPS(t *testing.T) {
	tests := []struct{ raw string; want int64 }{
		{"2", 200}, {"2.345", 235}, {"-18.046", -1805}, {"60", 6000},
	}
	for _, test := range tests {
		got, err := domain.ParsePercentageBPS(test.raw)
		if err != nil || got != test.want { t.Fatalf("ParsePercentageBPS(%q) = %d, %v", test.raw, got, err) }
	}
}
```

Also reject empty, non-decimal, NaN, infinity, and values whose basis-point result exceeds `int64`.

- [ ] **Step 2: Run the parser test and confirm RED**

Run: `go test ./internal/domain -run TestParsePercentageBPS -count=1`

Expected: FAIL because `ParsePercentageBPS` does not exist.

- [ ] **Step 3: Implement exact signed rounding**

Use `math/big.Rat`: parse the trimmed decimal, multiply by 100, divide numerator by denominator, and round half away from zero. Never use `float64`.

```go
func ParsePercentageBPS(value string) (int64, error) {
	value = strings.TrimSpace(value)
	percentage, ok := new(big.Rat).SetString(value)
	if value == "" || !ok { return 0, errors.New("percentage must be a decimal number") }
	percentage.Mul(percentage, big.NewRat(100, 1))
	return roundedBasisPoints(percentage)
}

func roundedBasisPoints(value *big.Rat) (int64, error) {
	quotient, remainder := new(big.Int), new(big.Int)
	quotient.QuoRem(value.Num(), value.Denom(), remainder)
	twiceRemainder := new(big.Int).Lsh(new(big.Int).Abs(remainder), 1)
	if twiceRemainder.Cmp(value.Denom()) >= 0 {
		if value.Sign() < 0 { quotient.Sub(quotient, big.NewInt(1)) } else { quotient.Add(quotient, big.NewInt(1)) }
	}
	if !quotient.IsInt64() { return 0, errors.New("percentage basis points are out of range") }
	return quotient.Int64(), nil
}
```

- [ ] **Step 4: Write source-parity adapter tests**

For each adapter, return one fixture containing liquidity, `m5` buys/sells, `m5` volume, and `m5` price change. Assert the same normalized result:

```go
want := domain.MarketSnapshot{
	LiquidityUSD: domain.USD{Micros: 5_000_000_000},
	FiveMinuteBuys: 13, FiveMinuteSells: 7,
	FiveMinuteVolumeUSD: domain.USD{Micros: 750_000_000},
	FiveMinutePriceChangeBPS: 200,
}
```

Add malformed-fixture cases for missing buys, sells, volume, or price change and for negative counts/volume. Each must return a bounded provider error rather than silently normalizing absence to zero.

- [ ] **Step 5: Run adapter tests and confirm RED**

Run: `go test ./internal/adapters/httpclient -run 'Test(DexScreener|GeckoTerminal).*Market' -count=1`

Expected: FAIL because the DTOs and `MarketSnapshot` lack momentum fields.

- [ ] **Step 6: Implement both adapter mappings**

Use pointer DTO fields for buys/sells so missing JSON differs from zero. Use `json.Number` for DEX Screener volume/price and strings for GeckoTerminal. Parse volume with `ParseUSD`, price with `ParsePercentageBPS`, and set `ObservedAt: time.Now().UTC()`.

- [ ] **Step 7: Verify the normalized evidence slice**

Run: `go test ./internal/domain ./internal/adapters/httpclient -count=1`

Expected: PASS. Capture `git diff --stat`; do not stage or commit.

---

### Task 2: Enforce market momentum in deterministic evaluation

**Files:**
- Modify: `internal/domain/policy.go:9-203`
- Modify: `internal/domain/policy_test.go`
- Modify: `internal/application/evaluator.go:71-77`
- Modify: `internal/application/evaluator_test.go`
- Modify: `internal/adapters/postgres/candidate_repository.go:144-175`
- Test: `internal/adapters/postgres/candidate_repository_test.go`

**Interfaces:**
- Consumes: Task 1 `MarketSnapshot` fields.
- Produces: `CandidatePolicy{MinFiveMinuteBuyShareBPS, MinFiveMinuteTurnoverBPS, MinFiveMinutePriceChangeBPS, MaxFiveMinutePriceChangeBPS int64}`.
- Produces rule codes `five_minute_buy_share`, `five_minute_turnover`, and `five_minute_price_change`.

- [ ] **Step 1: Extend policy boundary tests**

Set valid evidence to 13 buys, 7 sells, $750 volume, $5,000 liquidity, and +200 bps. Add one mutation per new rule:

```go
{"rejects buy share below 65 percent", func(e *domain.CandidateEvidence) { e.FiveMinuteBuys = 12; e.FiveMinuteSells = 8 }, domain.RuleFiveMinuteBuyShare},
{"rejects turnover below 15 percent", func(e *domain.CandidateEvidence) { e.FiveMinuteVolumeUSD = domain.USD{Micros: 749_999_999} }, domain.RuleFiveMinuteTurnover},
{"rejects momentum below two percent", func(e *domain.CandidateEvidence) { e.FiveMinutePriceChangeBPS = 199 }, domain.RuleFiveMinutePriceChange},
{"rejects momentum above sixty percent", func(e *domain.CandidateEvidence) { e.FiveMinutePriceChangeBPS = 6001 }, domain.RuleFiveMinutePriceChange},
```

Also test zero total transactions and zero liquidity so ratio calculation cannot divide by zero or pass accidentally.

- [ ] **Step 2: Run policy tests and confirm RED**

Run: `go test ./internal/domain -run TestCandidatePolicy -count=1`

Expected: FAIL on missing fields/rule constants.

- [ ] **Step 3: Implement overflow-safe ratio rules**

Compute ratios with `math/big.Int`, not integer multiplication that may overflow:

```go
func ratioBPS(numerator, denominator int64) (int64, bool) {
	if numerator < 0 || denominator <= 0 { return 0, false }
	value := new(big.Int).Mul(big.NewInt(numerator), big.NewInt(10_000))
	value.Quo(value, big.NewInt(denominator))
	return value.Int64(), value.IsInt64()
}
```

Use total transactions as `buys+sells`, buy share as `buys/total`, and turnover as `five-minute volume/liquidity`. Require all 12 rule results to be emitted with non-empty observed and limit strings.

- [ ] **Step 4: Propagate market evidence through evaluator and persistence**

Replace the old transaction-only assignment with:

```go
evidence := domain.CandidateEvidence{
	LiquidityUSD: market.LiquidityUSD,
	FiveMinuteBuys: market.FiveMinuteBuys,
	FiveMinuteSells: market.FiveMinuteSells,
	FiveMinuteVolumeUSD: market.FiveMinuteVolumeUSD,
	FiveMinutePriceChangeBPS: market.FiveMinutePriceChangeBPS,
}
```

Persist only normalized aggregates under fixed keys:

```go
"five_minute_buys": evidence.FiveMinuteBuys,
"five_minute_sells": evidence.FiveMinuteSells,
"five_minute_volume_usd_micros": evidence.FiveMinuteVolumeUSD.Micros,
"five_minute_price_change_bps": evidence.FiveMinutePriceChangeBPS,
```

Keep route, authority, and extension evidence unchanged. Do not expose this JSON through the dashboard.

- [ ] **Step 5: Run focused evaluator and repository tests**

Run: `go test ./internal/domain ./internal/application ./internal/adapters/postgres -run 'CandidatePolicy|Evaluator|CandidateRepository' -count=1`

Expected: PASS, or PostgreSQL contract tests explicitly skip only when `TEST_DATABASE_URL` is absent.

- [ ] **Step 6: Verify source impact**

Run `codegraph affected` for the changed Go paths, then `go test ./... -count=1`. Fix every fixture to carry internally consistent buys/sells/volume/price evidence. Do not stage or commit.

---

### Task 3: Add validated bold-v2 configuration and wiring

**Files:**
- Modify: `internal/config/config.go:30-130`
- Modify: `internal/config/config_test.go`
- Modify: `cmd/paper-bot/automation.go:295-384`
- Modify: `.env.example:13-32`
- Modify: `context/product.md`

**Interfaces:**
- Consumes: Task 2 `CandidatePolicy` fields.
- Produces config fields matching the approved thresholds.
- Produces `productionCandidatePolicy(config.Config) domain.CandidatePolicy` with all market gates.

- [ ] **Step 1: Write exact-default and invalid-bound tests**

In `TestLoadDefaultsPaperAutomationDisabled`, clear every strategy environment variable and assert:

```go
if cfg.MaxPoolAge != 90*time.Minute || cfg.MinLiquidityUSD.Micros != 5_000_000_000 ||
	cfg.MinFiveMinuteTransactions != 20 || cfg.MinFiveMinuteBuyShareBPS != 6_500 ||
	cfg.MinFiveMinuteTurnoverBPS != 1_500 || cfg.MinFiveMinutePriceChangeBPS != 200 ||
	cfg.MaxFiveMinutePriceChangeBPS != 6_000 || cfg.TakeProfitBPS != 5_000 ||
	cfg.StopLossBPS != 2_000 || cfg.MaxHoldDuration != 45*time.Minute ||
	cfg.StrategyVersion != "bold-momentum-v2" { t.Fatalf("unexpected bold-v2 defaults: %s", cfg.SafeHealth()) }
```

Add validation mutations for buy share outside `1..10000`, turnover outside `1..10000`, minimum momentum greater than maximum, and momentum outside `-10000..100000`.

- [ ] **Step 2: Run config tests and confirm RED**

Run: `go test ./internal/config -count=1`

Expected: FAIL because the new config fields/defaults do not exist.

- [ ] **Step 3: Implement validated configuration**

Add:

```go
MinFiveMinuteBuyShareBPS     int64 `env:"MIN_FIVE_MINUTE_BUY_SHARE_BPS" envDefault:"6500"`
MinFiveMinuteTurnoverBPS     int64 `env:"MIN_FIVE_MINUTE_TURNOVER_BPS" envDefault:"1500"`
MinFiveMinutePriceChangeBPS  int64 `env:"MIN_FIVE_MINUTE_PRICE_CHANGE_BPS" envDefault:"200"`
MaxFiveMinutePriceChangeBPS  int64 `env:"MAX_FIVE_MINUTE_PRICE_CHANGE_BPS" envDefault:"6000"`
```

Change only the approved defaults: age `90m`, liquidity `5000`, transactions `20`, take profit `5000`, stop loss `2000`, hold `45m`, strategy `bold-momentum-v2`. Keep notional, impact, quotas, cadence, and provider bounds unchanged.

- [ ] **Step 4: Wire policy fields into runtime**

```go
return domain.CandidatePolicy{
	MaxPoolAge: cfg.MaxPoolAge,
	MinLiquidityUSD: cfg.MinLiquidityUSD,
	MinFiveMinuteTransactions: cfg.MinFiveMinuteTransactions,
	MinFiveMinuteBuyShareBPS: cfg.MinFiveMinuteBuyShareBPS,
	MinFiveMinuteTurnoverBPS: cfg.MinFiveMinuteTurnoverBPS,
	MinFiveMinutePriceChangeBPS: cfg.MinFiveMinutePriceChangeBPS,
	MaxFiveMinutePriceChangeBPS: cfg.MaxFiveMinutePriceChangeBPS,
	MaxEntryPriceImpactBPS: cfg.MaxEntryPriceImpactBPS,
}
```

- [ ] **Step 5: Update public configuration and product docs**

Add the four new non-secret keys to `.env.example` and replace old baseline values with approved bold-v2 values. Document that the profile is paper-only and versioned; do not mention or inspect protected values.

- [ ] **Step 6: Verify config and wiring**

Run: `go test ./internal/config ./cmd/paper-bot -count=1`

Expected: PASS. Run `git diff --check -- .env.example context/product.md internal/config cmd/paper-bot`; do not stage or commit.

---

### Task 4: Define deterministic social and Hermes policies

**Files:**
- Create: `internal/domain/social_policy.go`
- Create: `internal/domain/social_policy_test.go`
- Modify: `internal/domain/social.go`
- Modify: `internal/domain/social_test.go`
- Modify: `internal/application/admission.go:34-80`
- Modify: `internal/application/admission_test.go`
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`

**Interfaces:**
- Produces `SocialPolicy.Evaluate(SocialMetrics, int) SocialPolicyEvaluation`.
- Produces `VerdictPolicy{MinimumConfidence, MinimumHypeQuality, MaximumManipulationProbability int}` with `Validate() error` and `Accepts(Verdict) bool`.
- Changes `AdmissionOptions.MinimumConfidence` to `AdmissionOptions.VerdictPolicy domain.VerdictPolicy`.

- [ ] **Step 1: Write social policy table tests**

```go
policy := domain.SocialPolicy{MinScore: 30, MinUniqueAuthors: 3, MinOriginalPosts: 2, MinExactMintMentions: 2, MaxWarningPosts: 1}
valid := domain.SocialMetrics{Posts: 4, UniqueAuthors: 3, OriginalPosts: 2, Reposts: 1, Replies: 1, ExactMintMentions: 2, WarningPosts: 1}
```

Assert the valid boundary passes. Independently fail score `29`, authors `2`, original posts `1`, mint mentions `1`, and warnings `2`. Invalid negative or impossible aggregates must fail closed.

- [ ] **Step 2: Write Hermes policy tests**

Use a valid `BUY/70/60/35` verdict. Independently reject `HOLD`, confidence `69`, hype `59`, and manipulation `36`. Assert configuration rejects values outside `0..100` and zero minimum quality thresholds.

- [ ] **Step 3: Run domain tests and confirm RED**

Run: `go test ./internal/domain -run 'SocialPolicy|VerdictPolicy' -count=1`

Expected: FAIL because the policy types do not exist.

- [ ] **Step 4: Implement policy types without raw evidence**

```go
type SocialPolicy struct {
	MinScore, MinUniqueAuthors, MinOriginalPosts, MinExactMintMentions, MaxWarningPosts int
}

type SocialPolicyEvaluation struct { Eligible bool }

type VerdictPolicy struct {
	MinimumConfidence int
	MinimumHypeQuality int
	MaximumManipulationProbability int
}

func (policy VerdictPolicy) Accepts(verdict Verdict) bool {
	return verdict.Outcome == VerdictBuy && verdict.Confidence >= policy.MinimumConfidence &&
		verdict.HypeQuality >= policy.MinimumHypeQuality &&
		verdict.ManipulationProbability <= policy.MaximumManipulationProbability
}
```

Extend `SocialMetrics` with `OriginalPosts int`. In `AnalyzeSocial`, increment it only when both `IsRepost` and `IsReply` are false; never derive it by subtracting potentially overlapping categories. Do not return arbitrary failure strings from these policies; orchestration maps failures to fixed safe enums.

- [ ] **Step 5: Enforce Hermes policy inside admission**

Replace the confidence-only gate with:

```go
if !options.VerdictPolicy.Accepts(input.Verdict) {
	return errors.New("Hermes verdict does not meet admission policy")
}
```

This is defense in depth even though scan orchestration will reject before calling admission.

- [ ] **Step 6: Add and validate configuration**

Add exact defaults:

```go
MinSocialScore              int `env:"MIN_SOCIAL_SCORE" envDefault:"30"`
MinSocialUniqueAuthors      int `env:"MIN_SOCIAL_UNIQUE_AUTHORS" envDefault:"3"`
MinSocialOriginalPosts      int `env:"MIN_SOCIAL_ORIGINAL_POSTS" envDefault:"2"`
MinSocialExactMintMentions  int `env:"MIN_SOCIAL_EXACT_MINT_MENTIONS" envDefault:"2"`
MaxSocialWarningPosts       int `env:"MAX_SOCIAL_WARNING_POSTS" envDefault:"1"`
MinHermesHypeQuality        int `env:"MIN_HERMES_HYPE_QUALITY" envDefault:"60"`
MaxHermesManipulationRisk   int `env:"MAX_HERMES_MANIPULATION_RISK" envDefault:"35"`
```

Keep `MIN_HERMES_CONFIDENCE=70`. Add the public keys to `.env.example`.

- [ ] **Step 7: Verify policy and admission defense**

Run: `go test ./internal/domain ./internal/application ./internal/config -run 'SocialPolicy|VerdictPolicy|Admission|Config' -count=1`

Expected: PASS with no repository call for any rejected verdict. Do not stage or commit.

---

### Task 5: Order social/Hermes gates and publish safe rejection enums

**Files:**
- Modify: `internal/application/production_scan_job.go:90-239`
- Modify: `internal/application/production_scan_job_test.go`
- Modify: `internal/application/production_scan_job_review_test.go`
- Modify: `internal/domain/dashboard_activity.go:93-141`
- Modify: `internal/domain/dashboard_activity_test.go`
- Modify: `cmd/paper-bot/automation.go:310-343`
- Modify: `dashboard/src/api.ts:43-56,179-182`
- Modify: `dashboard/src/api.test.ts`

**Interfaces:**
- Consumes: Task 4 `SocialPolicy` and `VerdictPolicy`.
- Produces fixed safe reasons: `deterministic_five_minute_buy_share`, `deterministic_five_minute_turnover`, `deterministic_five_minute_price_change`, `deterministic_social_quality`, `hermes_verdict_threshold`.
- Preserves `market_evidence_unavailable` and `social_evidence_unavailable` candidate-level outcomes.

- [ ] **Step 1: Write call-order and no-bypass tests**

Add tests with counting fakes. Assert:

```go
if social.calls != 0 || verdicts.calls != 0 || admissions.calls != 0 || broker.calls != 0 {
	t.Fatal("downstream providers were called after deterministic market rejection")
}
```

For market-eligible but social-policy-ineligible evidence, assert exactly one bounded social search and zero Hermes/admission/broker calls. For Hermes-policy rejection, assert one social search and one Hermes request, verdict audit storage still occurs, and admission/broker remain zero.

- [ ] **Step 2: Run production scan tests and confirm RED**

Run: `go test ./internal/application -run 'ProductionScanJob' -count=1`

Expected: FAIL because options lack policy fields and safe reasons.

- [ ] **Step 3: Add policy options and ordered gates**

Extend `ProductionScanJobOptions` with:

```go
SocialPolicy  domain.SocialPolicy
VerdictPolicy domain.VerdictPolicy
StrategyVersion string
```

After usable social evidence is saved, evaluate `ScoreSocial(social.Metrics)` and stop with `deterministic_social_quality` when ineligible. After a validated Hermes verdict is saved, stop with `hermes_verdict_threshold` when ineligible. Only then obtain the second entry quote and call atomic admission.

- [ ] **Step 4: Map first failed market rule safely**

Extend `reviewedReasonForEvaluation` with fixed mappings:

```go
domain.RuleFiveMinuteBuyShare:    domain.ReviewReasonDeterministicBuyShare,
domain.RuleFiveMinuteTurnover:    domain.ReviewReasonDeterministicTurnover,
domain.RuleFiveMinutePriceChange: domain.ReviewReasonDeterministicPriceChange,
```

Do not include observed values, provider messages, social aggregates, or Hermes reasoning in reviewed-candidate records.

- [ ] **Step 5: Wire exact policies**

Construct both policies from config in `dynamicProductionScanJob.RunOnce` and pass the same `VerdictPolicy` to `AdmissionOptions` and `ProductionScanJobOptions`.

- [ ] **Step 6: Update backend and frontend allowlists**

Add the five exact reason strings to `ReviewedCandidateReason`, `isReviewedCandidateReason`, and TypeScript's union/array. Add one API parser test that accepts each new reason and one test that rejects an arbitrary provider error string.

- [ ] **Step 7: Verify no provider-data exposure**

Run:

```text
go test ./internal/domain ./internal/application ./internal/transport/httpapi -count=1
npm --prefix dashboard test -- --run
npm --prefix dashboard run build
```

Expected: PASS. Search API DTOs and dashboard code for forbidden raw fields (`tweet text`, post IDs, provider request data, Hermes evidence, quote mint/input amount); no new exposure may exist.

---

### Task 6: Scope admission identity and expose active strategy safely

**Files:**
- Modify: `internal/application/production_scan_job.go:185,289-291`
- Modify: `internal/application/production_scan_job_test.go`
- Modify: `internal/application/dashboard.go:24-76`
- Modify: `internal/application/dashboard_test.go`
- Modify: `internal/transport/httpapi/httpapi.go:85-184`
- Modify: `internal/transport/httpapi/dashboard_test.go`
- Modify: `cmd/paper-bot/main.go:66-73`
- Modify: `dashboard/src/api.ts:58-107`
- Modify: `dashboard/src/api.test.ts`
- Modify: `dashboard/src/App.tsx:46-59`
- Modify: `dashboard/src/App.test.tsx`

**Interfaces:**
- Produces `productionAdmissionKey(strategyVersion string, pool domain.DiscoveredPool) string`.
- Adds `StrategyVersion string` to `application.DashboardSnapshot` and `DashboardReadServiceOptions`.
- Adds top-level JSON `strategy_version` and TypeScript `DashboardSnapshot.strategyVersion`.

- [ ] **Step 1: Write strategy-scoped key tests**

```go
keyV1 := productionAdmissionKey("production-v1", pool)
keyV2 := productionAdmissionKey("bold-momentum-v2", pool)
if keyV1 == keyV2 { t.Fatal("strategy versions shared one admission key") }
if productionAdmissionKey("bold-momentum-v2", pool) != keyV2 { t.Fatal("key is not deterministic") }
```

Also assert empty/whitespace strategy versions are rejected by `ProductionScanJob` configuration before provider calls.

- [ ] **Step 2: Run scan tests and confirm RED**

Run: `go test ./internal/application -run 'AdmissionKey|ProductionScanJob' -count=1`

Expected: FAIL because the key currently omits strategy version.

- [ ] **Step 3: Implement version-scoped idempotency**

```go
func productionAdmissionKey(strategyVersion string, pool domain.DiscoveredPool) string {
	return strings.TrimSpace(strategyVersion) + ":" + pool.Identity() + ":" + pool.CreatedAt.UTC().Format(time.RFC3339Nano)
}
```

Pass `cfg.StrategyVersion` into scan options and use it for every candidate. Do not change Ent uniqueness or quotas: the existing globally unique key now contains strategy identity, while daily and position rows already retain strategy version.

- [ ] **Step 4: Write active-version API tests**

Assert the dashboard service rejects an empty strategy version and returns `bold-momentum-v2` even with empty positions/results. Assert the HTTP response contains only:

```json
{"strategy_version":"bold-momentum-v2","utc_date":"2026-08-20","daily_results":[],"open_positions":[],"twitter_analytics":[],"automation_activity":[],"reviewed_candidates":[]}
```

- [ ] **Step 5: Implement safe top-level projection**

Set `DashboardReadServiceOptions.StrategyVersion` from already-validated `cfg.StrategyVersion` in `cmd/paper-bot/main.go`. Copy it through `DashboardSnapshot` and `dashboardResponse`; never derive it from an optional trade row.

- [ ] **Step 6: Parse and render the active version**

Add `strategyVersion: text(record.strategy_version, 'strategy_version')` to `dashboard/src/api.ts`. Change the status panel signature to:

```tsx
function StatusPanel({ openPositionCount, utcDate, strategyVersion }: { openPositionCount: number; utcDate: string; strategyVersion: string })
```

Render `Active strategy: bold-momentum-v2` as plain text while retaining the paper-only label. Add an accessible UI test for that text with zero trades.

- [ ] **Step 7: Verify identity and projection**

Run:

```text
go test ./internal/application ./internal/transport/httpapi ./cmd/paper-bot -count=1
npm --prefix dashboard test -- --run
npm --prefix dashboard run build
```

Expected: PASS. Confirm API output still contains only its allowlisted fields and no protected strategy internals are added.

---

### Task 7: Lock exit geometry and full paper-only regression

**Files:**
- Modify: `internal/domain/exit_test.go`
- Modify: `internal/application/position_manager_test.go`
- Modify: `cmd/paper-bot/automation_test.go`
- Modify: `tests/integration/paper_flow_test.go`
- Modify: `tests/security/paper_only_test.go`

**Interfaces:**
- Consumes config defaults `TakeProfitBPS=5000`, `StopLossBPS=2000`, `MaxHoldDuration=45m`.
- Preserves `ExitPolicy`, `PositionManager`, quote-only marks, and existing no-route/unsellable semantics.

- [ ] **Step 1: Add exact exit boundary tests**

```go
policy := domain.ExitPolicy{TakeProfitBPS: 5_000, StopLossBPS: 2_000, MaxHoldDuration: 45 * time.Minute}
```

Assert +4,999 bps holds and +5,000 closes `TAKE_PROFIT`; -1,999 holds and -2,000 closes `STOP_LOSS`; 44m59.999s holds and 45m closes `TIMEOUT`. Retain fee-aware mark behavior and five-consecutive-no-route unsellable behavior.

- [ ] **Step 2: Run exit tests**

Run: `go test ./internal/domain ./internal/application -run 'Exit|PositionManager' -count=1`

Expected: PASS using explicit v2 policy values. A failure indicates wiring/fixture drift, not permission to change the approved thresholds.

- [ ] **Step 3: Add one end-to-end eligible fixture**

In the integration flow, use one synthetic pool with:

```go
LiquidityUSD: domain.USD{Micros: 5_000_000_000},
FiveMinuteBuys: 13,
FiveMinuteSells: 7,
FiveMinuteVolumeUSD: domain.USD{Micros: 750_000_000},
FiveMinutePriceChangeBPS: 200,
```

Use revoked authorities, legacy SPL, full-size reciprocal routes, social boundary aggregates, and a `BUY/70/60/35` verdict. Assert one atomic `$100` paper position under `bold-momentum-v2` and idempotent rerun behavior.

- [ ] **Step 4: Add immutable-control regressions**

Independently reject active mint authority, active freeze authority, unallowlisted Token-2022 extension, missing entry route, missing reciprocal exit route, second quote over 1,000 bps impact, and any non-quote HTTP method/path. Assert no position opens in every case.

- [ ] **Step 5: Run the repository-required gate matrix**

Run each command bare and preserve real exit codes:

```text
go test ./... -count=1
go test -race ./... -count=1
go vet ./...
go build ./...
npm --prefix dashboard test -- --run
npm --prefix dashboard run lint
npm --prefix dashboard run build
uv run --group dev pytest
uv run --group dev ruff check .
uv run --group dev ruff format --check .
uv run --group dev mypy
uv build
git diff --check
```

If a project-level gate is not applicable to this nested Go project, record the exact command output rather than claiming it passed. PostgreSQL contract skips are acceptable only when their existing explicit prerequisite is absent.

- [ ] **Step 6: Run structural/security checks**

Confirm Compose still defines exactly `api` and `dashboard`, both host bindings remain loopback-only, no database service exists, and no wallet/signing/send/swap dependency or code path was added. Do not run `docker compose config`.

- [ ] **Step 7: Sync CodeGraph and review the bounded diff**

Run `codegraph affected`, inspect every changed source path, then `codegraph sync .`. Verify no `.codegraph/`, `.env.local`, runtime logs, reports, sessions, raw social data, or generated provider bodies appear in Git status. Do not stage or commit.

---

### Task 8: Deploy safely and prove runtime behavior

**Files:**
- Named local configuration replacements only: `.env.local`
- Runtime evidence only: existing ignored `var/log/` and `var/reports/`

**Interfaces:**
- Consumes the verified `bold-momentum-v2` binary and dashboard build.
- Produces real HTTP/browser evidence and a 24-hour paper-only observation result.
- Does not enable Birdeye or any extra provider.

- [ ] **Step 1: Obtain explicit deployment approval**

Before changing protected local configuration or recreating containers, show the final diff/gate results and ask for approval to deploy. Code-plan approval alone does not authorize deployment, Git mutation, or protected-config mutation.

- [ ] **Step 2: Apply named strategy replacements atomically**

Without printing the file or any old values, set only these keys:

```text
MAX_POOL_AGE=90m
MIN_LIQUIDITY_USD=5000
MIN_FIVE_MINUTE_TRANSACTIONS=20
MIN_FIVE_MINUTE_BUY_SHARE_BPS=6500
MIN_FIVE_MINUTE_TURNOVER_BPS=1500
MIN_FIVE_MINUTE_PRICE_CHANGE_BPS=200
MAX_FIVE_MINUTE_PRICE_CHANGE_BPS=6000
MIN_SOCIAL_SCORE=30
MIN_SOCIAL_UNIQUE_AUTHORS=3
MIN_SOCIAL_ORIGINAL_POSTS=2
MIN_SOCIAL_EXACT_MINT_MENTIONS=2
MAX_SOCIAL_WARNING_POSTS=1
MIN_HERMES_CONFIDENCE=70
MIN_HERMES_HYPE_QUALITY=60
MAX_HERMES_MANIPULATION_RISK=35
TAKE_PROFIT_BPS=5000
STOP_LOSS_BPS=2000
MAX_HOLD_DURATION=45m
STRATEGY_VERSION=bold-momentum-v2
```

The updater must reject duplicate target keys, preserve every non-target byte, use a same-directory atomic replacement, remove its temporary file on failure, and print only target key names plus success/failure—not values or file contents.

- [ ] **Step 3: Rebuild exactly two services**

Run: `docker compose up -d --build --force-recreate api dashboard`

Do not run `docker compose config`. Record container IDs and UTC start timestamps without displaying environment variables.

- [ ] **Step 4: Verify real local behavior**

Require HTTP 200 from API health, direct dashboard, and proxied dashboard. Fetch `/api/v1/dashboard?date=YYYY-MM-DD` and assert top-level `strategy_version == "bold-momentum-v2"`, bounded arrays, safe reason enums, and no forbidden keys. Open the dashboard preview and verify the paper-only label and active strategy are visible.

- [ ] **Step 5: Observe at least two scheduled scans**

Wait for two non-overlapping five-minute cycles. Require `SCAN/COMPLETED` records and continued 30-second monitor activity. A candidate-level fixed non-trade reason is acceptable; `SCAN/FAILED`, unclassified internal errors, provider retry loops, or real-transaction behavior fail the gate.

- [ ] **Step 6: Run a bounded 24-hour observer**

Poll only the loopback dashboard endpoint every 30 seconds for 24 hours. The observer must use one request per interval, a 10-second timeout, no retries, stop on the first invalid/unsafe response, verify strategy identity on every sample, and write only timestamps plus safe aggregate/enumerated fields to ignored runtime evidence. At completion, report sample count, first/last UTC timestamps, scan outcome counts, reason counts, daily admission deltas, maximum observed open-position count, and whether any unsafe key appeared. Closed-position counts belong to the later persisted-data analysis because the safe dashboard does not expose them.

If the observer process disappears or its evidence is incomplete, report the gate as incomplete and restart it; never infer success from container readiness.

- [ ] **Step 7: Keep strategy evaluation separate from runtime acceptance**

The 24-hour gate proves operational stability only. Do not call the strategy profitable until at least 50 closed positions or 14 days, whichever takes longer. Then compare expectancy, win rate, median return, drawdown, route/evidence failures, and age/liquidity/social/buy-share buckets against `production-v1` using only persisted normalized aggregates.

- [ ] **Step 8: Defer Birdeye explicitly**

Do not retry Birdeye during implementation or deployment. Revisit it only after quota/rate recovery under its separate bounded-provider task.

---

## Self-Review Results

- **Spec coverage:** All approved market thresholds, social gates, Hermes gates, exit geometry, strategy identity, dashboard visibility, immutable controls, runtime proof, and falsification criteria map to Tasks 1-8.
- **Scope:** This remains one cohesive plan because source normalization, policy, admission, and strategy attribution must deploy atomically; no independent provider or live-trading subsystem is included.
- **Type consistency:** `MarketSnapshot` feeds `CandidateEvidence`; `CandidatePolicy`, `SocialPolicy`, and `VerdictPolicy` are shared by runtime wiring; `StrategyVersion` feeds admission identity and dashboard projection.
- **Data safety:** Only normalized aggregates enter snapshots; only fixed enums and the public strategy ID reach the dashboard.
- **Git safety:** Every task ends without staging or committing. Git delivery requires separate user authorization.
- **Deferred work:** Birdeye validation remains separate. Long-term profitability evaluation begins only after the minimum forward sample.
