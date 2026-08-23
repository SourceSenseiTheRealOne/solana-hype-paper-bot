# Two-Source Fresh Pool Discovery Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers:executing-plans` to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Accept bounded DexScreener and GeckoTerminal fresh-pool candidates independently, watch future pools until launch, and apply the approved fresh-token policy defaults to paper-only evaluation.

**Architecture:** A source-union discovery adapter resolves DexScreener hints to actual Solana pools and merges them with GeckoTerminal new pools. A bounded `bot_states` repository retains future pools until due; production scanning evaluates due watches and new pools through the unchanged deterministic/social/Hermes/admission pipeline.

**Tech Stack:** Go 1.26, Ent/Postgres, bounded HTTP adapters, Docker Compose, Vitest dashboard.

## Global Constraints
- Paper-only: no wallet, signing, transactions, swaps, orders, or blockchain writes.
- Read-only provider calls have one attempt, deadlines, bounded bodies/results, and no browser-triggered route.
- Discovery stays at five-minute cadence and evaluates at most five pools per cycle.
- No dotenv values may be printed; only named non-secret setting lines may be updated.
- Do not commit, push, stage, reset, stash, or clean.

---

### Task 1: Source-union discovery

**Files:**
- Modify: `internal/adapters/httpclient/dexscreener.go`
- Modify: `internal/adapters/httpclient/dexscreener_test.go`
- Replace: `internal/application/hint_matched_pool_discovery.go`
- Modify: `internal/application/hint_matched_pool_discovery_test.go`
- Modify: `cmd/paper-bot/automation.go`

- [ ] Write failing adapter and application tests for a Dex-only pool, Gecko-only pool, cross-source deduplication, provider order, newest-first ordering, and five-pool cap.
- [ ] Run focused tests and confirm the existing intersection-only implementation fails the Dex-only/Gecko-only assertions.
- [ ] Add a capped DexScreener token-pair resolver that returns only validated Solana pool identities with creation time.
- [ ] Replace intersection-only discovery with a union adapter that independently fetches both sources, merges canonical identities, sorts newest-first, and caps to five pools.
- [ ] Wire the union adapter into the production scan factory and run focused tests green.

### Task 2: Future-pool watch

**Files:**
- Create: `internal/application/future_pool_watch.go`
- Create: `internal/application/future_pool_watch_test.go`
- Modify: `internal/adapters/postgres/candidate_repository.go`
- Modify: `internal/adapters/postgres/candidate_repository_test.go`
- Modify: `internal/application/production_scan_job.go`
- Modify: `internal/application/production_scan_job_test.go`
- Modify: `cmd/paper-bot/automation.go`

- [ ] Write failing tests that a future pool is watched rather than evaluated, is evaluated exactly once at/after launch, expires 30 minutes after launch, and watch capacity is five.
- [ ] Run focused tests and confirm future pools are currently skipped instead of watched.
- [ ] Implement validated watch identity/value and a Postgres `bot_states` store with strict five-item bound and delete-on-consume/expiry behavior.
- [ ] Merge due watches with current discovery before the existing five-item production cap; record future discoveries without provider/social/Hermes calls until due.
- [ ] Run focused application/Postgres tests green.

### Task 3: Deterministic policy and approved runtime settings

**Files:**
- Modify: `internal/domain/policy.go`
- Modify: `internal/domain/policy_test.go`
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`
- Modify: `.env.example`
- Modify: `cmd/paper-bot/automation.go`

- [ ] Write failing domain/config tests for: a 90-day-old pool accepted, a pool older than 90 days rejected, stale market observations not rejected solely for age, $10k liquidity, ten five-minute transactions, 1,000 bps impact, and $100 default notional.
- [ ] Run focused tests and confirm they fail under current policy defaults.
- [ ] Remove the market-freshness rule/configuration, set the approved defaults, retain valid timestamps/provider validation, and keep 90-day maximum pool age.
- [ ] Run focused tests green.

### Task 4: Active non-secret settings, verification, and local rebuild

**Files:**
- Modify only named non-secret lines in ignored `.env.local` without displaying its contents: `PAPER_TRADE_USD`, `MAX_POOL_AGE`, `MIN_LIQUIDITY_USD`, `MIN_FIVE_MINUTE_TRANSACTIONS`, `MAX_ENTRY_PRICE_IMPACT_BPS`; remove `MAX_MARKET_EVIDENCE_AGE` if present.

- [ ] Run Go domain/application/config/adapter tests, then full race/vet/build gates.
- [ ] Run dashboard verification, paper-only static guard, Compose/API/proxy/CI guard tests, `git diff --check`, and CodeGraph affected/sync.
- [ ] Rebuild/recreate exactly `api` and `dashboard`.
- [ ] Verify health, direct/proxied dashboard equality, and the next normal scan without forcing provider work.
- [ ] Mark the 24-hour shadow observation restarted; report only fresh runtime evidence.
