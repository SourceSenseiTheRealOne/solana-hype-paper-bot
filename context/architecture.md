# Architecture Context

## Repository and deployables

One Go process owns discovery, evaluation, atomic admission, paper brokerage, position monitoring, restart recovery, reporting, and a read-only HTTP API. A separate React/Vite dashboard container renders the dashboard. The project-owned Supabase stack owns Postgres; Compose must not add a competing Postgres service.

## Request and data flows

```text
Public providers → bounded Go HTTP adapters → application services → Ent/Postgres
Restricted Hermes profile → evidence-only verdict adapter → application services → Ent/Postgres
Browser on 127.0.0.1 → read-only Go API → Ent/Postgres
```

The browser never contacts Supabase or providers directly. Containers reach Postgres and restricted local Hermes only through explicit `host.docker.internal` boundaries.

TwitterAPI.io receives only bounded exact-mint social-search windows. Birdeye receives only Solana new-listing or single-mint security-report reads; its listing hints are supplemental discovery inputs and its security report availability never replaces the Solana-RPC authority/extension inspection used by deterministic policy.

## Module boundaries

`domain` defines validated value types and policy. `application` orchestrates use cases through `ports`. `adapters` implement provider and Postgres ports. `transport/httpapi` maps only read-only HTTP DTOs. Dependency direction is domain → application → ports → adapters/transport. Domain/application do not import Chi, Ent, database drivers, or provider clients.

## Storage and cache ownership

Ent owns typed reads/writes; Atlas-generated SQL migrations under `supabase/migrations/` own schema history. JSON evidence is bounded. There is no cache, queue, or Redis. Runtime logs and reports are ignored local `var/` data.

## Runtime topology

Only one local Supabase project stack runs at a time. Its standard local ports remain owned by Supabase. The Go API and dashboard publish only `127.0.0.1` application ports. Development, staging, and production branches are separate promotion lanes; implementation is local only.

## Invariants

- Paper trade notional is $100.
- At most 3 open virtual positions and 30 newly admitted positions per UTC day.
- `bold-momentum-v2` exits are +50%, -20%, or 45 minutes.
- UTC controls all timestamps and quota dates.
- Financial persistence excludes `float64`.
- No signing, wallet, transaction-building, swap execution, or blockchain writes exist.

## Vertical tracer

A candidate discovered through a fake bounded provider is evaluated, atomically admitted once, opened as a paper position, marked/closed by policy, persisted, and returned by the Go read API. The dashboard consumes that API through its typed client. Focused Go, Vitest, migration, race, and production-build gates prove the tracer.
