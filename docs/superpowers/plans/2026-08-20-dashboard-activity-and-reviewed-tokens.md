# Dashboard Activity and Reviewed Tokens Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add three bounded, safe dashboard cards: recent automation activity, recently reviewed tokens that were not traded, and per-token Twitter search analytics.

**Architecture:** Persist only the last ten sanitized scheduler lifecycle records and the last ten scan-reviewed/non-admitted tokens under separate fixed `bot_states` keys; do not read Docker logs. Read the ten newest persisted social snapshots through an Ent adapter that maps only aggregate Twitter metrics. Extend the existing dashboard snapshot/DTO so the React client keeps one polling endpoint.

**Tech Stack:** Go 1.26, Ent/PostgreSQL, React/Vite, TypeScript, Vitest, Testing Library, Docker Compose.

## Global Constraints

- Preserve the sole public endpoint: `GET /api/v1/dashboard?date=YYYY-MM-DD`.
- The dashboard stays loopback-only and read-only; no raw logs, secrets, URLs/query strings, provider payloads, social/Hermes evidence, or `rule_results` may cross the API boundary.
- `automation_activity` and `reviewed_candidates` each contain at most ten values, sorted newest first.
- Use only fixed activity values: jobs `SCAN|MONITOR`; outcomes `STARTED|COMPLETED|FAILED`; existing safe error categories/stages apply only to failed values.
- Do not change provider limits, five-minute scan cadence, thirty-second monitor cadence, or paper-only safety constraints.
- No migration is needed: the existing `bot_states` table owns the single bounded activity projection.
- Do not commit, push, stage, reset, stash, or clean unless the owner explicitly requests it.

---

### Task 1: Typed bounded dashboard projections

**Files:**
- Create: `internal/domain/dashboard_activity.go`
- Create: `internal/domain/dashboard_activity_test.go`
- Create: `internal/application/dashboard_activity.go`
- Create: `internal/application/dashboard_activity_test.go`

**Interfaces:**
- Produces `domain.AutomationActivity`, `domain.AutomationJob`, and `domain.AutomationOutcome`.
- Produces `domain.ReviewedCandidate`.
- Produces `domain.TwitterAnalytics`.
- Produces `application.AutomationActivityStore` with `Append(context.Context, domain.AutomationActivity) error` and `AutomationActivityReader` with `List(context.Context) ([]domain.AutomationActivity, error)`.
- Produces `application.ReviewedCandidateStore` with `AppendReviewed(context.Context, domain.ReviewedCandidate) error` and `ReviewedCandidatesReader` with `ListReviewed(context.Context) ([]domain.ReviewedCandidate, error)`.
- Produces `application.TwitterAnalyticsReader` with `ListTwitterAnalytics(context.Context) ([]domain.TwitterAnalytics, error)`.

- [ ] **Step 1: Write failing domain tests**

Add tests that accept a UTC scan-completed record and a UTC monitor-failed record with only fixed failure labels. Assert invalid job/outcome, non-UTC/zero time, failed records without category/stage, non-failed records with category/stage, and slices over ten all fail validation.

- [ ] **Step 2: Verify RED**

Run:

```bash
go test ./internal/domain -run 'TestAutomationActivity|TestReviewedCandidate' -count=1
```

Expected: compile failure because the new domain types and validators do not exist.

- [ ] **Step 3: Implement minimum typed domain contract**

Implement immutable-by-convention values with fixed constants, `Validate()` methods, `MaxDashboardItems = 10`, and `ValidateDashboardItems` that rejects more than ten records and non-descending timestamps. `ReviewedCandidate.Validate()` accepts only a non-empty mint, non-zero UTC checked time, and non-empty outcome.

- [ ] **Step 4: Verify GREEN**

Run the focused domain test command and confirm it passes.

- [ ] **Step 5: Write failing application tests**

Test the application projection helper rejects a missing reader/store dependency and that a dashboard reader with a failing projection returns a wrapped error instead of a partial snapshot.

- [ ] **Step 6: Verify RED then implement**

Run the focused application test, implement only the new interfaces and helper validation needed by later adapters, then rerun it to GREEN.

### Task 2: PostgreSQL bounded activity and reviewed-decision adapters

**Files:**
- Create: `internal/adapters/postgres/dashboard_activity_repository.go`
- Create: `internal/adapters/postgres/dashboard_activity_repository_test.go`
- Create: `internal/adapters/postgres/twitter_analytics_repository.go`
- Create: `internal/adapters/postgres/twitter_analytics_repository_test.go`


**Interfaces:**
- Consumes Task 1 `AutomationActivityStore`, `AutomationActivityReader`, `ReviewedCandidateStore`, `ReviewedCandidatesReader`, and `TwitterAnalyticsReader` contracts.
- Produces `postgres.NewDashboardActivityRepository(client, now)` and `postgres.NewTwitterAnalyticsRepository(client)`.

- [ ] **Step 1: Write failing activity-repository tests**

With the existing real Ent test client, append eleven distinct valid activity records and assert `List()` returns exactly ten in descending UTC order. Assert the retained JSON state contains only documented primitive keys and never accepts arbitrary map content.

- [ ] **Step 2: Verify RED**

Run:

```bash
go test ./internal/adapters/postgres -run TestDashboardActivityRepository -count=1
```

Expected: compile failure because the repository constructor does not exist.

- [ ] **Step 3: Implement bounded activity repository**

Use a fixed unexported state key. Under a repository mutex, load/validate the existing typed JSON projection, prepend one validated activity, truncate to ten, and Ent-upsert the one `bot_states` row. List validates the stored projection before returning it. No raw log text parameter exists in its API.

- [ ] **Step 4: Verify GREEN**

Rerun the focused activity-repository test command.

- [ ] **Step 5: Write failing reviewed-candidate projection tests**

Append eleven reviewed/non-admitted public mint summaries with ordered timestamps. Assert `ListReviewed()` returns only the newest ten ordered records and that its persistence API has no rule-results/evidence argument.

- [ ] **Step 6: Verify RED then implement**

Run:

```bash
go test ./internal/adapters/postgres -run 'TestDashboardActivityRepository|TestReviewedCandidateProjection' -count=1
```

Implement the second fixed-state-key bounded projection in the same repository. Map only mint, UTC checked time, and a fixed `NOT_TRADED` outcome; rerun GREEN.

- [ ] **Step 7: Write failing Twitter analytics repository tests**

Seed more than ten social snapshots with candidate edges and controlled aggregate metric maps. Assert `ListTwitterAnalytics()` returns the newest ten with only mint, UTC search time, fixed one-search count, score, posts, unique authors, exact mint mentions, and warning posts. Add a malformed metric map case that fails without returning raw JSON content.

- [ ] **Step 8: Verify RED then implement**

Run:

```bash
go test ./internal/adapters/postgres -run TestTwitterAnalyticsRepository -count=1
```

Implement the Ent query ordered by `observed_at` descending with a hard `Limit(domain.MaxDashboardItems)` and required candidate edge loading. Strictly decode only aggregate numeric metric fields; rerun GREEN.

### Task 3: Dashboard service, HTTP allowlist, and runtime composition

**Files:**
- Modify: `internal/application/dashboard.go`
- Modify: `internal/application/dashboard_test.go`
- Modify: `internal/transport/httpapi/httpapi.go`
- Modify: `internal/transport/httpapi/dashboard_test.go`
- Modify: `cmd/paper-bot/main.go`
- Modify: `cmd/paper-bot/main_test.go`
- Modify: `cmd/paper-bot/automation.go`
- Modify: `cmd/paper-bot/automation_test.go`

**Interfaces:**
- Consumes Task 1 domain/application interfaces and Task 2 adapters.
- Extends `application.DashboardSnapshot` with `AutomationActivity []domain.AutomationActivity` and `ReviewedCandidates []domain.ReviewedCandidate`.
- Extends `application.DashboardSnapshot` with `TwitterAnalytics []domain.TwitterAnalytics`.
- Extends the existing JSON DTO with `automation_activity`, `reviewed_candidates`, and `twitter_analytics` only.

- [ ] **Step 1: Write failing dashboard-service tests**

Extend the dashboard fake readers so a successful `Read()` must include activity/reviewed/Twitter values and each required reader is called. Add a separate test that a failing projection reader prevents a partial snapshot.

- [ ] **Step 2: Verify RED then implement**

Run:

```bash
go test ./internal/application -run TestDashboardReadService -count=1
```

Add the two required reader options, validate returned projections with Task 1 helpers, and return the extended snapshot. Rerun GREEN.

- [ ] **Step 3: Write failing HTTP allowlist test**

Extend the existing handler fixture with one safe scan activity, one reviewed candidate, and one Twitter aggregate. Assert exact serialized fields/timestamps and assert the response contains none of synthetic `rule_results`, provider URL, raw error, tweet text, post ID, or evidence sentinel strings.

- [ ] **Step 4: Verify RED then implement**

Run:

```bash
go test ./internal/transport/httpapi -run TestHandlerServesAllowlistedDashboardSnapshot -count=1
```

Add explicit response structs/mappers for the two new domain values. Keep request parsing, headers, and route set unchanged. Rerun GREEN.

- [ ] **Step 5: Write failing runtime activity tests**

Using a recording activity store and scheduled-job fake, prove an observed scan writes `STARTED` then `COMPLETED`; a failing scan writes `STARTED` then `FAILED` using only `automationErrorCategory` and `automationFailureStage`; and a completed monitor writes `STARTED` then `COMPLETED`. Add a production-scan observer fake proving a reviewed non-admission appends only mint/UTC/`NOT_TRADED`. Assert projection-store errors do not replace the scheduled job’s result or trigger raw error logging.

- [ ] **Step 6: Verify RED then implement**

Run:

```bash
go test ./cmd/paper-bot -run 'TestObserved|TestAutomation' -count=1
```

Implement a small composition-layer `observedScheduledJob` wrapper. It writes the start event, delegates once, writes one terminal safe event, and emits only a fixed warning if persistence itself fails. Add a `ProductionReviewRecorder` port to `ProductionScanJob` so completed non-admissions append only mint/UTC/`NOT_TRADED`; persistence failure is non-fatal to scan/admission behavior. Use the wrapper for both dynamic scan/monitor jobs, construct the bounded projection repository in `run()`, and inject it into the dashboard and scan composition. Rerun GREEN.

### Task 4: Typed React cards and responsive accessible UI

**Files:**
- Modify: `dashboard/src/api.ts`
- Modify: `dashboard/src/api.test.ts`
- Modify: `dashboard/src/App.tsx`
- Modify: `dashboard/src/App.test.tsx`
- Modify: `dashboard/src/App.css`

**Interfaces:**
- Consumes the extended `DashboardSnapshot` DTO.
- Produces `AutomationActivityCard`, `ReviewedCandidatesCard`, and `TwitterAnalyticsCard` UI functions within the existing page module.

- [ ] **Step 1: Write failing API-client tests**

Extend the safe fixture with all three arrays. Assert parsing returns camel-case typed values; reject invalid activity/Twitter timestamps, unknown job/outcome, lists over ten, non-string mints/outcomes, invalid score bounds, and negative aggregate counters.

- [ ] **Step 2: Verify RED then implement**

Run:

```bash
pnpm.cmd --dir dashboard test -- --run src/api.test.ts
```

Add TypeScript types and strict validators for only documented fields. Rerun GREEN.

- [ ] **Step 3: Write failing component tests**

Render the extended fixture and assert the accessible Automation activity region shows `SCAN`, `COMPLETED`, and a fixed failure category/stage without raw response details. Assert the Reviewed, not traded and Twitter search analytics tables have captions, show public mint/aggregate fields, and each card has an explicit empty state.

- [ ] **Step 4: Verify RED then implement**

Run:

```bash
pnpm.cmd --dir dashboard test -- --run src/App.test.tsx
```

Render the two semantic card components below metrics, use `<time>` for UTC values, and add responsive overflow/mint styles without animation. Rerun GREEN.

### Task 5: Full verification and local runtime proof

**Files:**
- Modify: `docs/superpowers/specs/2026-08-20-dashboard-activity-and-reviewed-tokens-design.md` only if implementation exposes a previously missed ambiguity.

- [ ] **Step 1: Run focused regression suites**

```bash
go test ./internal/domain ./internal/application ./internal/adapters/postgres ./internal/transport/httpapi ./cmd/paper-bot -count=1
pnpm.cmd --dir dashboard test -- --run
```

- [ ] **Step 2: Run complete source gates**

```bash
go test ./...
go test -race ./...
go vet ./...
go build ./cmd/paper-bot
pnpm.cmd --dir dashboard run verify
bash scripts/verify-no-live-trading-deps.sh
bash tests/verify-compose-topology.sh
bash tests/verify-dashboard-proxy.sh
bash tests/verify-api-entrypoint.sh
git diff --check
```

- [ ] **Step 3: Rebuild and prove the live read-only contract**

Rebuild/recreate only `api` and `dashboard`. Verify health and direct/proxied dashboard API responses. Assert the response key allowlist is exactly `utc_date`, `daily_results`, `open_positions`, `automation_activity`, `reviewed_candidates`, and `twitter_analytics`; report only aggregate lengths and safe activity labels. Open the dashboard preview and verify all three cards/empty states in the rendered browser.

- [ ] **Step 4: Preserve runtime and report evidence**

Leave the two-service Compose stack running. Report code/test evidence separately from the current paper-position result. Do not read, print, or retain dotenv values, reports, raw logs, raw provider data, or secrets.
