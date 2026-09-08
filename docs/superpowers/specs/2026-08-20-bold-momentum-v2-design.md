# Bold Momentum v2 Paper Strategy Design

**Status:** Proposed for user review
**Date:** 2026-08-20
**Strategy version:** `bold-momentum-v2`

> Historical cohort: new admissions are superseded by [`bold-momentum-v3`](./2026-08-31-fast-transient-candidate-retry-design.md). This document retains the exact v2 policy for comparison.

## 1. Purpose

The existing `production-v1` paper strategy admits no current positions because liquidity is the dominant rejection boundary: nine of the latest ten safely projected reviews failed the `$10,000` liquidity floor and one lacked market evidence. Lowering liquidity alone would increase low-quality candidates, route failures, and manipulation exposure without defining an edge.

`bold-momentum-v2` instead tests a narrower hypothesis:

> Among very fresh Solana pools that pass immutable token and route-safety checks, lower-liquidity pools with sustained five-minute buy pressure, meaningful turnover, controlled positive price momentum, diverse exact-mint social attention, and a low-manipulation Hermes verdict have positive paper expectancy under asymmetric exits.

This is a research hypothesis, not a claim of profitability. The strategy remains structurally paper-only.

## 2. Research basis and limits

DEX Screener pair responses expose transaction counts by direction, volume, price change, liquidity, and creation time.[1] CoinGecko's on-chain pool response exposes five-minute buys, sells, buyers, sellers, price change, total and directional volume, liquidity, and launchpad graduation details.[2][3] These fields permit a source-compatible momentum policy without adding another provider.

Research associates abnormal social attention with contemporaneous and short-horizon cryptocurrency performance, and other work shows that combined market/social signals can also identify pump-and-dump behavior.[6][7] Intraday momentum has empirical support in liquid cryptocurrency markets, especially around elevated volume, but that evidence does not establish profitability for brand-new Solana microcaps.[8] Accordingly, momentum and social features are treated as falsifiable filters, not proven alpha.

Solana rug-pull research identifies freeze-authority abuse, liquidity withdrawal, and pump-and-dump as major patterns and reports extremely short fraudulent lifecycles.[4] Commercial research likewise reports pervasive manipulation on Pump.fun and Raydium, although its exact prevalence figures should be treated as vendor estimates rather than independent ground truth.[5] These findings justify preserving token controls and full-size reciprocal exit-route checks even in a deliberately riskier paper cohort.

## 3. Approaches considered

### 3.1 Threshold-only aggression — rejected

Example: `$3,000` liquidity, five transactions, 15% price impact.

This would increase admissions but add no testable selection edge. It primarily increases slippage, no-route outcomes, provider noise, and manipulation exposure.

### 3.2 Conditional bold momentum — selected

Lower the liquidity floor while raising activity requirements and adding directional market, turnover, controlled momentum, deterministic social-quality, and Hermes manipulation gates.

This deliberately accepts more volatility and thinner markets, but only when independent evidence is directionally aligned.

### 3.3 Extreme lottery cohort — deferred

Example: `$2,500` liquidity, 15% impact, `+75%/-25%` exits.

This may be useful later as an isolated negative-control cohort. It is not suitable as the primary next strategy because markability and route disappearance would likely dominate results.

## 4. Policy

### 4.1 Market admission

| Constraint | `production-v1` | `bold-momentum-v2` |
|---|---:|---:|
| Minimum pool age | none | none |
| Maximum pool age | 90 days | 90 minutes |
| Minimum liquidity | $10,000 | $5,000 |
| Minimum five-minute transactions | 10 | 20 |
| Minimum five-minute buy share | unused | 65% |
| Minimum five-minute volume/liquidity | unused | 15% |
| Five-minute price change | unused | +2% to +60% |
| Maximum entry price impact | 10% | 10% |
| Paper notional | $100 | $100 |

Definitions:

- `buy_share_bps = buys * 10_000 / (buys + sells)`; zero transactions fail validation before division.
- `turnover_bps = five_minute_volume_usd * 10_000 / liquidity_usd`; values use fixed-point money, checked integer arithmetic, and no `float64` persistence.
- Five-minute price change is normalized to signed basis points.
- A field required by the strategy that is absent, malformed, negative where impossible, non-finite, or outside representable bounds makes that provider's market evidence unavailable. It never silently defaults to zero.
- Source routing remains unchanged: a DexScreener-origin pool uses DexScreener market evidence and a GeckoTerminal/CoinGecko-origin pool uses Gecko-compatible market evidence.

At `$5,000` reported liquidity, a `$100` paper quote represents 2% of reported pool liquidity. Jupiter's fresh, full-size quote remains the authoritative route/impact check; reported liquidity never overrides a missing route or impact above 10%.

### 4.2 Deterministic social gate

After bounded social retrieval and persistence, but before Hermes is called, require:

| Metric | Minimum/maximum |
|---|---:|
| Social score | at least 30 |
| Unique authors | at least 3 |
| Original posts | at least 2 |
| Exact mint mentions | at least 2 |
| Warning posts | zero |

Existing repeated-text and simultaneous-post penalties remain part of the social score. Raw posts, text, IDs, authors, and hashes remain absent from the dashboard and public API.

Candidates that fail this gate are recorded under fixed safe reason codes and do not consume a Hermes request. A successful provider request with zero usable posts remains `social_evidence_unavailable`, not a low-score rejection.

### 4.3 Hermes gate

Hermes remains advisory and cannot bypass deterministic rules. Admission requires all of:

| Verdict field | Requirement |
|---|---:|
| Outcome | `BUY` |
| Confidence | at least 70 |
| Hype quality | at least 60 |
| Manipulation probability | at most 35 |

The complete validated verdict may be stored locally under existing protections, but only fixed safe rejection categories may reach dashboard activity. Arbitrary Hermes reasoning or risk text must not be exposed.

### 4.4 Immutable safety controls

The following do not become configurable strategy relaxations:

- mint authority revoked;
- freeze authority revoked;
- legacy SPL Token or explicitly allowlisted Token-2022 extensions;
- full-size executable entry route;
- full-size reciprocal exit route;
- fresh second entry quote within the 10% impact limit;
- quote-only Jupiter GET usage;
- read-only Helius inspection;
- atomic serializable PostgreSQL admission;
- no wallet, key, signature, transaction construction, order, swap, broadcast, or chain write.

## 5. Position and exit policy

`bold-momentum-v2` uses:

| Constraint | Value |
|---|---:|
| Paper notional | $100 |
| Hard stop | -20% net return |
| Take profit | +50% net return |
| Maximum hold | 45 minutes |
| Mark cadence | 30 seconds |
| Maximum open positions | 3 total |
| Maximum admissions | 30 per UTC day |

Before modeled costs, `+50%/-20%` has a 2.5 reward/risk ratio and a 28.57% break-even win rate, compared with 2.0 and 33.33% for `+30%/-15%`. Actual reporting remains based on fresh full-liquidation quotes net of fixed paper fee estimates.

Partial exits and trailing stops are deliberately deferred. They require quantity-splitting, realized/unrealized accounting, additional state transitions, and route modeling. Those changes should follow evidence that the entry hypothesis has value rather than being bundled into the first experiment.

A no-route observation remains a markability failure, never a favorable assumed exit. Existing no-route recording behavior remains fail-closed.

## 6. Architecture changes

### 6.1 Market evidence

Extend normalized market evidence with:

- five-minute buys;
- five-minute sells;
- five-minute volume in fixed-point USD;
- five-minute signed price-change basis points.

Both current market adapters already receive corresponding upstream fields but must decode, validate, and normalize them explicitly. Candidate persistence records the normalized values locally; dashboard DTOs expose only bounded aggregate fields approved by the existing projection policy.

### 6.2 Policy evaluation

Add deterministic rule codes for:

- buy share;
- turnover;
- price-momentum range.

The evaluator remains pure and deterministic once normalized evidence is supplied. Every failed rule maps to a fixed safe reviewed-candidate reason.

### 6.3 Social and verdict admission

Introduce a deterministic social-policy component between `SocialAnalyzer` and Hermes. Extend admission validation to enforce hype quality and manipulation probability in addition to outcome and confidence. The scan job must record a safe non-trade and continue to the next bounded candidate for ordinary candidate-level policy failures.

Systemic storage errors, malformed internal state, context cancellation, and provider-health failures remain scan failures rather than being silently downgraded.

### 6.4 Strategy identity

Persist `bold-momentum-v2` on positions and daily results. Include strategy version in the admission idempotency identity so a previously evaluated `production-v1` candidate cannot incorrectly suppress a legitimate v2 paper experiment. Idempotency remains unique within a strategy.

Only one strategy actively admits positions during this first v2 deployment. Historical v1 results remain immutable; the system does not fabricate a parallel baseline from unmatched market periods.

## 7. Configuration

Add explicit non-secret settings for the new thresholds. Defaults remain fail-closed and must be represented in `.env.example` and product context. The ignored local environment may be updated only by named non-secret replacement without displaying the file.

The following stay fixed:

- scan cadence: exactly five minutes;
- monitor cadence: exactly 30 seconds;
- discovery cap: five candidates;
- serial, non-overlapping scans;
- bounded deadlines and no retry loops;
- maximum three open positions and 30 daily admissions;
- canonical Solana Mainnet USDC paper quote mint.

No Birdeye dependency is required for v2. Birdeye remains deferred until its quota/rate condition recovers.

## 8. Safe observability

Add fixed safe reviewed reasons for market momentum and social-policy rejections. No reason may carry observed raw values unless already approved as a bounded aggregate. Do not expose:

- raw social posts or IDs;
- author identifiers or hashes;
- provider URLs, request data, headers, or responses;
- Hermes reasoning or evidence;
- quote mint, entry amount, route internals, or rule-results JSON;
- credentials or cost ledgers.

The dashboard should make the active strategy version and aggregate rejection categories visible without adding mutation or provider-trigger endpoints.

## 9. Test strategy

Behavioral implementation must use strict RED → GREEN → REFACTOR slices:

1. adapter decoding and malformed-field rejection for both sources;
2. fixed-point buy-share, turnover, and momentum policy boundaries;
3. social-policy boundary and Hermes short-circuit behavior;
4. admission enforcement of confidence, hype quality, and manipulation probability;
5. fixed safe reason mapping and dashboard allowlisting;
6. `+50%/-20%/45m` exit boundaries using net quote-native returns;
7. strategy-scoped idempotency and historical-version separation;
8. paper-only forbidden-dependency and read-only API regressions.

Required closure includes project Go race tests, vet, build, dashboard verification, security/static guards, Compose topology tests, `git diff --check`, CodeGraph affected analysis, and source-index sync. Protected database contract tests remain explicit if their reset test database is unavailable.

## 10. Forward validation and falsification

Do not call v2 profitable or successful based on admissions alone. Run it until at least 50 closed paper positions or 14 consecutive observation days, whichever takes longer.

Report by strategy version:

- net expectancy per closed position;
- win rate and median return;
- maximum drawdown;
- no-route observation rate;
- provider evidence-unavailable rate;
- return by liquidity, age, buy-share, turnover, momentum, and social-score bucket;
- time to close and close-reason distribution.

Tighten, suspend, or reject the hypothesis if any of these hold after the minimum sample:

- net expectancy is non-positive;
- no-route observations exceed 10% of monitoring attempts;
- realized median loss exceeds the intended -20% stop because routes vanish or quotes gap;
- the `$5,000-$7,499` cohort materially underperforms `$7,500-$9,999` without compensating upside;
- deterministic social quality does not improve expectancy over comparable market-qualified candidates;
- provider failures or social/Hermes costs make the bounded five-minute cadence unreliable.

## 11. Runtime acceptance

The prior 24-hour observer process expired and therefore did not prove a complete 24-hour run. It must not be reported as passed.

After v2 implementation and deployment:

1. prove API health and direct/proxied dashboard parity;
2. prove at least one normal scheduled scan lifecycle;
3. start a new durable, read-only fail-fast observation anchored after deployment;
4. require 24 hours without terminal `SCAN / FAILED` for runtime acceptance;
5. report code/test completion separately from the 24-hour runtime outcome.

## Sources

[1] https://docs.dexscreener.com/api/reference — DEX Screener API Reference

[2] https://docs.coingecko.com/reference/pools-addresses — CoinGecko Onchain Multiple Pools Data

[3] https://docs.coingecko.com/reference/latest-pools-list — CoinGecko New Pools List

[4] https://arxiv.org/html/2603.24625v2 — From Hype to Collapse: Investigating Rug Pull Scams on Solana

[5] https://www.soliduslabs.com/reports/solana-rug-pulls-pump-dumps-crypto-compliance — 2025 Rug Pull Report

[6] https://www.isi.edu/results/publications/12609/detecting-cryptocurrency-pump-and-dump-frauds-using-market-and-social-signals — Detecting cryptocurrency pump-and-dump frauds using market and social signals

[7] https://ideas.repec.org/a/eee/jbfina/v178y2025ics0378426625001384.html — Social media-based attention and cryptocurrency returns

[8] https://research.birmingham.ac.uk/en/publications/bitcoin-intraday-time-series-momentum — Bitcoin intraday time-series momentum
