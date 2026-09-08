# Fast transient-candidate retry design

**Date:** 2026-08-31  
**Status:** Approved inline by the operator  
**Scope:** Local paper-only `bold-momentum-v3` automation

## Problem

The PumpSwap pool `2UPzCEtVcbqwSBQiownSeY4iqAFjyGGaUgVK9yzGZw6i` was discovered when it was about two and a half minutes old. The candidate reached the dashboard at `2026-08-31T10:04:24Z` but was classified as `market_evidence_unavailable`, so it was not evaluated again. By the next five-minute discovery cycle, launch-feed churn could remove the pool from the bounded latest-token/latest-pool windows.

Later read-only evidence showed that the pool had sufficient liquidity, transaction count, turnover, controlled positive momentum, and executable $100 entry plus full-size reciprocal exit routes. Its five-minute buy share was around 64%, narrowly below the current 65% threshold.

The Pump.fun curve `G573bFku37k67vzc4cu9J9fTbY56SRfyqQLTtuCPf3H2` exposed a second transient case: the mint was discovered and safely recorded as `market_evidence_unavailable`, but its original zero-liquidity curve address would remain fixed in retry state after graduation. The operator approved migration-aware retry only; pre-graduation reserve-modelled fills remain out of scope.

## Objective

Retain and quickly re-evaluate candidates whose market evidence is temporarily incomplete, while admitting only candidates that subsequently satisfy the complete paper-only pipeline.

Because retry timing and the buy-share boundary both change the admitted cohort, the active strategy identity becomes `bold-momentum-v3`. This prevents new admissions and results from being silently mixed with `bold-momentum-v2` evidence.

## Design

### Durable bounded retry queue

Add a market-evidence retry store separate from future-launch watches. It uses the existing PostgreSQL `bot_state` table and therefore requires no schema migration.

Each entry contains only validated public pool identity and scheduling metadata:

- source, network, mint, pool address, and creation time;
- next-attempt time;
- hard expiry time.

Invariants:

- maximum five queued candidates;
- deterministic oldest-entry eviction at capacity;
- first retry due 30 seconds after a transient market-evidence failure;
- one due candidate consumed per fast tick;
- failed market evidence requeues the candidate for another 30 seconds;
- hard expiry 10 minutes after the first scheduling opportunity, additionally bounded by the normal maximum pool age;
- malformed and expired state is deleted safely;
- no unbounded loops, pagination, retries, or overlapping writers.

### Scheduler integration

Keep full DexScreener/GeckoTerminal discovery at five minutes. Extend the existing serial split cadence with one optional retry job that runs immediately after each 30-second position-monitor job.

Order on a fast tick:

1. monitor all open paper positions;
2. consume at most one due transient candidate;
3. run that candidate through the complete production evaluation/admission path.

The retry remains serial with scans and position monitoring. A retry has its own bounded timeout and cannot overlap another scan, retry, or monitor cycle.

### Candidate processing

On `market_evidence_unavailable`:

1. keep the existing safe dashboard classification;
2. enqueue or refresh the candidate's bounded retry state;
3. continue the current scan without failing the whole cycle.

When market evidence becomes available, the candidate must still pass, in order:

1. deterministic age, liquidity, m5 transaction, buy-share, turnover, and momentum checks;
2. Helius read-only mint/freeze/token-program checks;
3. Jupiter GET-only entry and full-size reciprocal exit-route checks;
4. Twitter aggregate quality policy;
5. restricted local Hermes verdict policy;
6. fresh entry quote, atomic paper admission, and virtual paper open.

A deterministic rejection, social/Hermes rejection, expiry, or successful processing ends retrying for that candidate.

### Migration-aware DexScreener evidence

When a successfully fetched DexScreener pair cannot produce usable market evidence, including a `dexId=pumpfun` pair with zero migrated liquidity, the adapter performs one bounded same-mint resolution through DexScreener's token-pairs endpoint. It follows the result only when the resolved Solana pair is strictly newer than the retained pool and has a different address, then fetches that pair once.

Provider transport/rate-limit failures do not trigger this extra resolution. The retry key and fixed expiry remain unchanged, and the candidate still requires a real $100 Jupiter entry quote plus full-size reciprocal Jupiter exit quote. No Pump.fun curve fill is simulated.

### Bold-v3 threshold adjustment

Change `MIN_FIVE_MINUTE_BUY_SHARE_BPS` default and protected local runtime setting from `6500` to `6000`.

Change `STRATEGY_VERSION` from `bold-momentum-v2` to `bold-momentum-v3` in defaults, examples, protected local runtime configuration, dashboard expectations, and strategy-scoped admission identity.

This preserves buy dominance while admitting the observed 63–65% cohort. All other market, social, Hermes, impact, token-safety, sizing, exit, quota, and paper-only thresholds remain unchanged.

## Safety boundaries

This change does not add or invoke:

- wallets, keypairs, private keys, or seed phrases;
- signing or transaction construction;
- swaps, orders, transaction submission, or blockchain writes;
- browser-triggered mutations;
- a live-trading path.

Jupiter remains `GET /swap/v1/quote` only. Helius remains read-only. The API and dashboard remain loopback-only and read-only.

## Verification

Use strict RED → GREEN tests for:

- queue cap, idempotent refresh, due ordering, one-item consumption, expiry, malformed state, and deterministic eviction;
- initial transient failure scheduling;
- retry success through the complete candidate path;
- repeated transient failure requeue;
- zero-liquidity Pump.fun evidence re-resolving once to a strictly newer same-mint pool;
- deterministic rejection stopping retries;
- position monitor running before retry on each fast tick;
- serial/no-overlap cadence;
- 60% inclusive and 59.99% exclusive buy-share boundaries;
- unchanged social/Hermes, token, quote, admission, and paper-only guards.

Then run focused tests, all Go tests, race detector, vet, dashboard verification, static no-live-trading and topology guards, CodeGraph affected/sync, `git diff --check`, rebuild only the API image, and verify fresh loopback scheduler/dashboard evidence.
