# Dashboard Activity and Reviewed Tokens Design

## Goal

Extend the existing loopback-only, read-only dashboard snapshot with (1) a bounded preview of sanitized automation lifecycle activity and (2) the most recent publicly identifiable candidates that received a persisted non-trade decision.

## Constraints

- Keep the single `GET /api/v1/dashboard?date=YYYY-MM-DD` endpoint; do not add mutation or provider-trigger routes.
- Keep the existing five-minute scan and thirty-second monitor cadence unchanged.
- Do not expose credentials, URLs/query strings, request headers/bodies, raw provider payloads, raw container logs, social evidence, Hermes evidence, or rule-by-rule/internal decision basis.
- Use only typed application/domain contracts in the application layer; database/Ent access remains in adapters.
- All timestamps are RFC3339 UTC. Each section returns at most ten records.
- No wallet, signing, transaction construction, swap execution, or blockchain write capability is introduced.

## Architecture

### Automation activity projection

The worker will persist bounded, browser-safe activity and reviewed-token projections in the existing `bot_states` table under separate fixed state keys. Each JSON value contains at most ten records, newest first. A dedicated adapter owns state-key validation, JSON shape, bounded append, and serialization so callers cannot persist arbitrary log text.

Each record is an `AutomationActivity` domain value:

- `occurred_at`: RFC3339 UTC timestamp.
- `job`: fixed enum `SCAN` or `MONITOR`.
- `outcome`: fixed enum `STARTED`, `COMPLETED`, or `FAILED`.
- `category`: optional fixed safe error category for a failed job.
- `stage`: optional fixed safe pipeline stage for a failed job.

The automation composition writes exactly one `STARTED` event at a job boundary and exactly one terminal `COMPLETED` or `FAILED` event. It reuses the existing safe `automationErrorCategory` and `automationFailureStage` functions; it never stores or returns wrapped error text. A process-local synchronization boundary serializes scan and monitor writes to the single bounded state record. Restarting preserves the last retained events; no historical event table or unbounded log is created.

### Reviewed, not traded candidates

`trade_decisions` are deliberately created only for persisted `BUY` admissions, so they cannot truthfully represent non-traded candidates. The scan pipeline instead appends an explicit safe review projection whenever it completes a review without admitting/opening a paper position. The second bounded state value returns typed `ReviewedCandidate` records:

- `mint_address`: the public Solana mint address.
- `checked_at`: the decision creation timestamp in RFC3339 UTC.
- `outcome`: the stored top-level decision outcome.

The review recorder never accepts or maps `rule_results`, candidate snapshots, social snapshots, Hermes verdict evidence, or any provider payload. The ordering is descending checked time with a hard ten-record limit.

### Dashboard snapshot and HTTP DTO

`DashboardSnapshot` gains:

- `automation_activity`: `[]AutomationActivity`
- `reviewed_candidates`: `[]ReviewedCandidate`
- `twitter_analytics`: `[]TwitterAnalytics`

`DashboardReadService` receives required reader interfaces for all three projections and fails closed when any cannot be loaded. The existing handler continues to allow only GET, reject bodies/unknown query keys, emit `no-store`, and return a single explicit allowlist. The response adds only:

```json
{
  "automation_activity": [
    {
      "occurred_at": "2026-08-20T12:00:00Z",
      "job": "SCAN",
      "outcome": "COMPLETED"
    }
  ],
  "reviewed_candidates": [
    {
      "mint_address": "public-token-mint",
      "checked_at": "2026-08-20T12:00:00Z",
      "outcome": "REJECT"
    }
  ]
}
```

Safe failure fields are omitted when they do not apply; browser parsing rejects unknown types and invalid timestamp/enum values.

### Twitter search analytics

Every successful bounded TwitterAPI.io search already persists one `social_snapshots` row linked to its candidate. A dedicated read adapter loads only the ten newest snapshots and their candidate mint addresses. It maps the stored aggregate metrics to `TwitterAnalytics`:

- `mint_address`: public Solana mint address.
- `searched_at`: the snapshot observation timestamp in RFC3339 UTC.
- `search_count`: fixed `1` because each persisted snapshot represents one completed bounded search.
- `score`: persisted bounded social score, from `0` through `100`.
- `posts`, `unique_authors`, `exact_mint_mentions`, and `warning_posts`: non-negative aggregate values from the persisted metrics object.

The adapter reads the metrics object server-side with strict numeric type/range checks. It never maps or serializes raw posts, text excerpts, post/author IDs, hashes, provider URLs, budget details, or the `social_evidence` object itself.

### Dashboard UI

The API client validates the two new arrays and their bounded field shapes. The page adds two semantic cards below the daily metrics:

1. **Automation activity**: compact, newest-first list with UTC time, scan/monitor label, outcome, and applicable fixed failure category/stage. It has an explicit empty state.
2. **Reviewed, not traded**: compact, newest-first table with mint, checked UTC, and outcome. It has an explicit empty state.
3. **Twitter search analytics**: compact, newest-first table with mint, searched UTC, one completed search, social score, post count, unique authors, exact mint mentions, and warning-post count. It has an explicit empty state.

Both remain polling-only through the existing React Query dashboard snapshot. They use native HTML, readable text labels rather than color alone, accessible headings, and responsive overflow for mint addresses.

## Error handling

- A projection-reader failure makes the existing dashboard request fail generically with `service unavailable`; raw database details are never returned.
- Automation activity storage failures are handled through the existing safe scheduler warning path without changing trading/admission behavior.
- Browser parsing rejects malformed, oversized, or non-allowlisted records and renders the existing generic dashboard error panel.

## Tests and verification

### Backend RED/GREEN coverage

- Domain validation accepts only fixed activity enums, UTC timestamps, safe failure labels, and a maximum of ten records.
- Activity persistence retains exactly the newest ten safe records and rejects arbitrary text/unknown keys.
- Reviewed-token persistence retains only scan-reviewed but non-admitted candidates, orders newest first, caps at ten, and cannot accept rule/evidence payloads.
- Twitter analytics reads only the ten newest social snapshots, joins each candidate, strictly maps only aggregate numeric metrics, and proves raw social evidence cannot leak.
- Dashboard service composes both readers into its snapshot and fails if either reader fails.
- HTTP handler serializes the two added allowlisted fields, preserves existing safe headers/date semantics, and proves excluded sensitive fields cannot leak.
- Automation runner emits start plus exactly one terminal sanitized activity record for successful scan, failed scan, and successful monitor paths.

### Frontend RED/GREEN coverage

- API parser accepts only the documented activity/reviewed-candidate DTO fields and validates timestamps/enums.
- The dashboard renders all three cards, their accessible labels, safe fixed failure values, aggregate Twitter analytics, and each empty state.
- Existing API error handling continues to suppress response text.

### Final gates

Run the project’s focused Go and dashboard tests, then:

```bash
go test ./...
go test -race ./...
go vet ./...
pnpm.cmd --dir dashboard run verify
bash scripts/verify-no-live-trading-deps.sh
bash tests/verify-compose-topology.sh
bash tests/verify-dashboard-proxy.sh
git diff --check
```

Rebuild the two-service Compose stack and verify direct/proxied dashboard API parity plus browser rendering. Do not run `docker compose config` because local dotenv interpolation is sensitive.
