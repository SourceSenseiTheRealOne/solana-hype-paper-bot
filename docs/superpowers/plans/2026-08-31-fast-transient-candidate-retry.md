# Fast Transient-Candidate Retry Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Catch very young Solana pools whose m5 market payload is temporarily incomplete by retrying one retained candidate every 30 seconds, and admit the 60–65% buy-dominance cohort under a new auditable `bold-momentum-v3` paper strategy.

**Architecture:** Keep five-minute two-source discovery and 30-second position monitoring. Add a separate five-entry PostgreSQL `bot_state` retry namespace with reservation leases; after every monitor tick, serially reserve at most one due candidate and run it through the unchanged full production evaluation/social/Hermes/quote/admission path. Market-evidence failures remain queued until a fixed 10-minute expiry; any usable market evaluation removes the retry before downstream processing.

**Tech Stack:** Go 1.24, Ent/PostgreSQL, existing `bot_state` JSON storage, Docker Compose, local Supabase, React/Vite dashboard, restricted local Hermes API.

## Global Constraints

- Paper-only: no wallet, keypair, signing, transaction construction, submission, swap execution, order, or blockchain-write path.
- Full discovery remains exactly every five minutes; position monitoring remains exactly every 30 seconds.
- Retry capacity is exactly five; one due retry is reserved per monitor tick; retry delay is 30 seconds; expiry is 10 minutes and never exceeds `MAX_POOL_AGE`.
- Retry, monitor, and scan jobs remain serial and non-overlapping; monitor executes before retry on every fast tick.
- Jupiter remains `GET /swap/v1/quote` only; full-size reciprocal exit quoting remains mandatory.
- A retained DexScreener candidate may resolve its mint once to a strictly newer Solana pair when the original pair payload is unusable; pre-graduation Pump.fun fills are forbidden.
- Helius remains read-only; mint/freeze/token-program restrictions remain unchanged.
- Twitter aggregate policy and restricted Hermes `BUY/70/60/<=35` policy remain mandatory.
- Buy-share boundary becomes inclusive at 6,000 bps and exclusive at 5,999 bps; all other strategy thresholds remain unchanged.
- Active strategy and strategy-scoped admission identity become `bold-momentum-v3`.
- Preserve ignored `.env.local` values and never print, log, stage, or commit them.
- Do not commit, push, merge, stash, reset, or clean unless separately requested by the owner.

---

### Task 1: Application Retry Contract and Candidate Lifecycle

**Files:**
- Create: `internal/application/market_retry.go`
- Create: `internal/application/market_retry_test.go`
- Modify: `internal/application/production_scan_job.go`
- Modify: `internal/application/production_scan_job_test.go`
- Modify: `internal/application/production_scan_job_review_test.go`

**Interfaces:**
- Produces:
  ```go
  const (
      MarketRetryCap      = 5
      MarketRetryInterval = 30 * time.Second
      MarketRetryTTL      = 10 * time.Minute
  )

  type MarketRetryStore interface {
      Schedule(context.Context, domain.DiscoveredPool, time.Time, time.Time) error
      ReserveDue(context.Context, time.Time, time.Time, int) ([]domain.DiscoveredPool, error)
      Complete(context.Context, domain.DiscoveredPool) error
  }

  type MarketRetryDiscoveryOptions struct {
      Store      MarketRetryStore
      Now        time.Time
      LeaseUntil time.Time
      Limit      int
  }

  func NewMarketRetryDiscovery(MarketRetryDiscoveryOptions) ProductionDiscovery
  ```
- Extends `ProductionScanJobOptions` with `MarketRetries MarketRetryStore`.
- Consumes only validated `domain.DiscoveredPool` identities and the existing typed `marketEvidenceUnavailable` marker.

- [ ] **Step 1: Write RED tests for retry discovery validation and one-item reservation**

  In `internal/application/market_retry_test.go`, add fakes and tests that assert:

  ```go
  func TestMarketRetryDiscoveryReservesOneDueCandidateWithThirtySecondLease(t *testing.T)
  func TestMarketRetryDiscoveryRejectsInvalidBoundsBeforeStoreCall(t *testing.T)
  ```

  The first test must assert `Limit == 1`, exact UTC `Now`, exact `LeaseUntil == Now+30s`, and the returned pool identity. The second must cover nil store, zero times, lease not after now, zero limit, and limit above five.

- [ ] **Step 2: Run the new application test and verify RED**

  Run:
  ```bash
  go test ./internal/application -run 'TestMarketRetryDiscovery' -count=1
  ```
  Expected: compile failure because the retry types/constructor do not exist.

- [ ] **Step 3: Add the minimal retry contract and discovery adapter**

  Implement the exact constants, interface, options, and constructor in `market_retry.go`. `Run` must call `ReserveDue` once, reject malformed store results by validating every pool, and return `DiscoveryResult{Pools: pools}` without pagination or retries.

- [ ] **Step 4: Run focused tests and verify GREEN**

  Run:
  ```bash
  go test ./internal/application -run 'TestMarketRetryDiscovery' -count=1
  ```
  Expected: PASS.

- [ ] **Step 5: Write RED scan-lifecycle tests**

  Extend production scan tests with a fake `MarketRetryStore` and these cases:

  ```go
  func TestProductionScanJobSchedulesTransientMarketFailureForThirtySeconds(t *testing.T)
  func TestProductionScanJobCompletesRetryAfterUsableMarketEvaluation(t *testing.T)
  func TestProductionScanJobDoesNotExtendRetryBeyondPoolAge(t *testing.T)
  ```

  Assertions:
  - the current `market_evidence_unavailable` safe review remains recorded;
  - first schedule uses `nextAttemptAt == now+30s` and `expiresAt == min(now+10m, pool.CreatedAt+MaxPoolAge)`;
  - a usable eligible or deterministic-ineligible evaluation calls `Complete` before social/Hermes;
  - schedule/complete errors fail the scan with controlled context and no paper open;
  - all normal downstream social/Hermes/admission/broker tests remain unchanged.

- [ ] **Step 6: Run scan tests and verify RED**

  Run:
  ```bash
  go test ./internal/application -run 'TestProductionScanJob(SchedulesTransient|CompletesRetry|DoesNotExtendRetry)' -count=1
  ```
  Expected: compile/assertion failure because `ProductionScanJob` does not manage retry state.

- [ ] **Step 7: Implement minimal scan integration**

  Require `MarketRetries` during production option validation. In `evaluateAndMaybeOpen`:
  - on typed market-evidence failure, record the safe reason and schedule/refresh the retry;
  - calculate fixed expiry with a small `earlierTime` helper;
  - on any usable evaluation, call `Complete` before deterministic or downstream processing;
  - never retry deterministic, social, Hermes, quote, admission, or broker rejections/errors through this market-only queue.

  Update `validProductionScanOptions` to inject a no-op fake store.

- [ ] **Step 8: Run all application tests and verify GREEN**

  Run:
  ```bash
  go test ./internal/application/... -count=1
  ```
  Expected: PASS.

---

### Task 1A: Migration-Aware DexScreener Retry

**Files:**
- Modify: `internal/adapters/httpclient/dexscreener.go`
- Modify: `internal/adapters/httpclient/dexscreener_test.go`

**Behavior:** After a successful original pair response fails normalization, or reports `dexId=pumpfun` with zero liquidity, resolve the mint once through `/token-pairs/v1/solana/{mint}`. Fetch evidence only when the resolved pair is strictly newer and has a different address. Preserve transport failure behavior, fixed retry identity/expiry, all deterministic gates, and mandatory reciprocal Jupiter routes.

- [ ] **Step 1: Write and observe a RED regression using a zero-liquidity Pump.fun pair followed by a newer migrated pair.**
- [ ] **Step 2: Extract pair-response normalization and implement the single bounded same-mint fallback.**
- [ ] **Step 3: Run focused DexScreener tests and verify GREEN without weakening incomplete-evidence rejection.**

---

### Task 2: Durable PostgreSQL Retry Store

**Files:**
- Create: `internal/adapters/postgres/market_retry_repository.go`
- Create: `internal/adapters/postgres/market_retry_repository_unit_test.go`
- Create: `internal/adapters/postgres/market_retry_repository_test.go`

**Interfaces:**
- Implements `application.MarketRetryStore` on `*CandidateRepository`.
- Uses `bot_state.state_key` prefix `market-retry:` and stores only:
  `source`, `network`, `mint_address`, `pool_address`, `created_at`, `next_attempt_at`, `expires_at`.
- `Schedule` preserves the earliest existing expiry rather than extending a retry forever.
- `ReserveDue` advances `next_attempt_at` to the supplied lease before returning at most `limit` pools.
- `Complete` is idempotent.

- [ ] **Step 1: Write RED pure serialization/validation tests**

  In the in-package unit test, add:

  ```go
  func TestMarketRetryValueRoundTripsValidatedPoolAndUTCMetadata(t *testing.T)
  func TestMarketRetryFromStateRejectsMissingMalformedAndInvertedTimes(t *testing.T)
  func TestOldestMarketRetryUsesPoolCreationThenIdentityTieBreak(t *testing.T)
  ```

  Test malformed values without printing their contents.

- [ ] **Step 2: Run unit tests and verify RED**

  Run:
  ```bash
  go test ./internal/adapters/postgres -run 'Test(MarketRetry|OldestMarketRetry)' -count=1
  ```
  Expected: compile failure because repository helpers do not exist.

- [ ] **Step 3: Implement serialization, keys, and deterministic selection**

  Use SHA-256 of `pool.Identity()` for state keys, RFC3339Nano UTC timestamps, exact pool validation, and controlled errors. Do not reuse the `future-pool-watch:` namespace.

- [ ] **Step 4: Run unit tests and verify GREEN**

  Run the same command. Expected: PASS.

- [ ] **Step 5: Write RED real-store behavior tests**

  Using `openTestEntClient(t)` and unique pool identities, add:

  ```go
  func TestCandidateRepositorySchedulesAndReservesOneDueMarketRetry(t *testing.T)
  func TestCandidateRepositoryPreservesFirstExpiryOnRefresh(t *testing.T)
  func TestCandidateRepositoryCapsRetriesAtFiveAndEvictsOldest(t *testing.T)
  func TestCandidateRepositoryDeletesExpiredRetryAndCompletesIdempotently(t *testing.T)
  ```

  Assert reservations are leased before return, not duplicated before lease expiry, one-item limits are honored, and all test state is explicitly completed/removed.

- [ ] **Step 6: Run database tests against an isolated disposable local database and verify RED**

  Set `TEST_DATABASE_URL` internally from protected configuration without printing it, create a disposable database, apply reviewed migrations/grants, then run:
  ```bash
  go test ./internal/adapters/postgres -run 'TestCandidateRepository.*MarketRetry' -count=1
  ```
  Expected: failure because public methods are missing. Drop the disposable database after the test, even on failure.

- [ ] **Step 7: Implement `Schedule`, `ReserveDue`, and `Complete`**

  Requirements:
  - capacity query is bounded to five;
  - refreshing an existing key keeps the original earlier expiry;
  - at capacity, evict exactly the deterministic oldest state;
  - expired/malformed rows are removed under the bounded query;
  - due rows are leased before return;
  - no retry loop, broad delete, raw state logging, or secret output.

- [ ] **Step 8: Re-run isolated real-store tests and verify GREEN**

  Run the same focused command against a fresh disposable database. Expected: PASS and database dropped afterward.

---

### Task 3: Serial Fast-Tick Scheduler and Production Wiring

**Files:**
- Modify: `internal/application/split_cadence.go`
- Modify: `internal/application/split_cadence_test.go`
- Modify: `cmd/paper-bot/automation.go`
- Modify: `cmd/paper-bot/automation_test.go`

**Interfaces:**
- Extend `SplitCadenceOptions` with:
  ```go
  Retry        ScheduledJob
  RetryTimeout time.Duration
  ```
- Change `productionCadenceOptions` to accept `(scan, monitor, retry application.ScheduledJob)`.
- Add `dynamicMarketRetryJob` that reserves one candidate with `NewMarketRetryDiscovery`, builds the existing complete production scan graph with retry discovery, disables duplicate SCAN lifecycle/report generation for retry ticks, and invokes `RunOnce` once.

- [ ] **Step 1: Write RED cadence order tests**

  Replace the initial-order test with:
  ```go
  func TestSplitCadenceRunsInitialScanMonitorAndRetrySerially(t *testing.T)
  func TestSplitCadenceRunsMonitorBeforeRetryOnFastTickWithoutOverlap(t *testing.T)
  ```

  The fast-tick test uses a long scan interval, short monitor interval, an atomic in-flight counter, and cancellation from the second retry call. It must assert ordered pairs `monitor,retry` and maximum in-flight count exactly one.

- [ ] **Step 2: Run cadence tests and verify RED**

  Run:
  ```bash
  go test ./internal/application -run 'TestSplitCadence' -count=1
  ```
  Expected: compile/assertion failure because retry scheduling is absent.

- [ ] **Step 3: Implement scheduler retry ordering**

  Validate non-nil retry and positive retry timeout. Run initial `scan`, then `monitor`, then `retry`. On each monitor ticker event, always run `monitor` first and `retry` second through the same bounded serial `run` helper. Preserve cancellation and `OnError` behavior.

- [ ] **Step 4: Run cadence tests and verify GREEN**

  Run the same command. Expected: PASS.

- [ ] **Step 5: Write RED production-wiring tests**

  Update `TestProductionCadenceUsesExactBoundedIntervals` to assert:
  - scan interval/timeout `5m/5m`;
  - monitor interval/timeout `30s/30s`;
  - retry timeout `45s`;
  - the exact retry job passed into options.

  Add a focused test for `dynamicMarketRetryJob` with fake dependencies/store proving one reserved candidate reaches the same full production pipeline and no daily report is generated.

- [ ] **Step 6: Run command-package tests and verify RED**

  Run:
  ```bash
  go test ./cmd/paper-bot -run 'Test(ProductionCadence|DynamicMarketRetry)' -count=1
  ```
  Expected: compile/assertion failure because production retry wiring is absent.

- [ ] **Step 7: Implement production wiring**

  Add `productionRetryTimeout = 45*time.Second`, construct `dynamicMarketRetryJob` in `startAutomation`, and factor one private builder that creates a `ProductionScanJob` from an injected `ProductionDiscovery` while preserving every existing policy/provider/store/broker dependency. Normal discovery retains activity/reporting; retry discovery does not generate duplicate scan activity or daily reports.

- [ ] **Step 8: Run command and application tests and verify GREEN**

  Run:
  ```bash
  go test ./cmd/paper-bot ./internal/application/... -count=1
  ```
  Expected: PASS.

---

### Task 4: Versioned 60% Buy-Share Cohort

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`
- Modify: `internal/domain/policy_test.go`
- Modify: `.env.example`
- Modify safely, without output: `.env.local`
- Modify: `context/product.md`
- Modify: `context/architecture.md`
- Modify: `README.md`
- Modify: `docs/superpowers/specs/2026-08-20-bold-momentum-v2-design.md` only to add a supersession note linking the v3 spec; do not rewrite historical v2 thresholds.

**Interfaces:**
- Config default `MIN_FIVE_MINUTE_BUY_SHARE_BPS=6000`.
- Config default and local value `STRATEGY_VERSION=bold-momentum-v3`.
- Existing `CandidatePolicy` arithmetic remains unchanged; only configured boundary changes.

- [ ] **Step 1: Write RED config and policy boundary tests**

  Change default assertions to `6_000` and `bold-momentum-v3`. In `policy_test.go`, set the valid policy to 6,000 bps, make the exact-boundary fixture `12 buys / 8 sells`, and add rejection at `11 buys / 9 sells` (5,500 bps) plus a direct ratio case that proves 5,999 bps fails.

- [ ] **Step 2: Run focused tests and verify RED**

  Run:
  ```bash
  go test ./internal/config ./internal/domain -run 'Test(LoadDefaults|CandidatePolicy)' -count=1
  ```
  Expected: failure because defaults still use v2/6,500 bps.

- [ ] **Step 3: Apply minimal default/runtime changes**

  Change only the two config defaults and their examples/docs. Update `.env.local` with a script that replaces only lines whose keys are exactly `MIN_FIVE_MINUTE_BUY_SHARE_BPS` and `STRATEGY_VERSION`; print only booleans/counts confirming one replacement per key, never values or neighboring lines.

- [ ] **Step 4: Run focused tests and verify GREEN**

  Run the same command. Expected: PASS.

- [ ] **Step 5: Verify no stale active-policy claims remain**

  Search active config/context/README files for `6500`, `65%`, and active `bold-momentum-v2`. Historical v2 specs/plans and generic tests may retain v2 examples; active defaults and operator docs must not.

---

### Task 5: Closure, Runtime Rebuild, and Live Paper Proof

**Files:**
- No new production files beyond Tasks 1–4.
- Runtime state remains ignored under `var/` and local Supabase volumes.

- [ ] **Step 1: Run CodeGraph affected selection and focused tests**

  Run:
  ```bash
  codegraph affected internal/application/market_retry.go internal/application/production_scan_job.go internal/adapters/postgres/market_retry_repository.go internal/application/split_cadence.go cmd/paper-bot/automation.go internal/config/config.go
  go test ./internal/application/... ./internal/adapters/postgres/... ./internal/config ./internal/domain ./cmd/paper-bot -count=1
  ```

- [ ] **Step 2: Run the full project gates**

  Run exactly:
  ```bash
  go test ./... -count=1
  go test -race ./... -count=1
  go vet ./...
  go build ./...
  pnpm.cmd --dir dashboard run verify
  bash scripts/verify-no-live-trading-deps.sh
  bash tests/verify-no-live-trading-deps_test.sh
  bash tests/verify-compose-topology.sh
  bash tests/verify-compose-topology_test.sh
  bash tests/verify-github-actions.sh
  bash tests/verify-github-actions_test.sh
  bash tests/verify-dashboard-proxy.sh
  bash tests/verify-api-entrypoint.sh
  git diff --check
  ```

- [ ] **Step 3: Inspect exact scope and secrets/artifacts**

  Verify `git status --short`, inspect every changed/untracked path, exclude `.env.local`, `.codegraph/`, `var/`, Supabase runtime state, credentials, logs, reports, browser workspaces, and generated binaries. Run a secret-safe candidate scan without printing matched values.

- [ ] **Step 4: Sync CodeGraph after source changes**

  Run:
  ```bash
  codegraph sync
  codegraph status .
  ```
  Expected: index current; `.codegraph/` remains ignored and unstaged.

- [ ] **Step 5: Rebuild/recreate only the API container**

  Preserve dashboard and database state. Run the bounded lifecycle command in tracked background mode:
  ```bash
  docker compose up -d --build --no-deps api
  ```

- [ ] **Step 6: Verify live loopback safety and v3 identity**

  Require HTTP 200 from:
  - `http://127.0.0.1:8642/health`;
  - `http://127.0.0.1:8080/healthz`;
  - `http://127.0.0.1:4173/`;
  - direct and proxied dashboard APIs.

  Assert direct/proxied semantic equality, top-level `strategy_version == "bold-momentum-v3"`, bounded arrays, fixed safe enums, no forbidden keys, loopback-only bindings, non-root API PID 1, and `app:app:750` writable log/report directories.

- [ ] **Step 7: Prove fresh serial runtime activity**

  Observe at least two fresh 30-second monitor completions and one retry tick without failures or overlap. If no transient candidate is naturally queued, use only the tested repository/application path against an isolated disposable database; do not insert product mock data into the live runtime.

- [ ] **Step 8: Report honestly**

  Distinguish:
  - code/test verification;
  - local runtime verification;
  - whether the referenced pool was newly admitted or merely now qualifies for bounded retry;
  - any remaining social/Hermes or current-market rejection.

  Do not claim a paper position unless the dashboard shows an actual atomic admission/open under `bold-momentum-v3`.
