# Project Decisions

## ADR-001: Project-owned local Supabase/Postgres

- **Status:** accepted
- **Date:** 2026-08-18
- **Context:** The bot needs durable local research evidence and deterministic schema rebuilds.
- **Decision:** Use isolated project-owned local Supabase/Postgres with `project_id = solana-hype-paper-bot`; do not add standalone Postgres to Compose.
- **Alternatives:** shared lab database; standalone Postgres container.
- **Consequences:** One stack runs at a time; migrations are reviewed and resettable.

## ADR-002: Ent typed persistence and Atlas SQL migrations

- **Status:** accepted
- **Date:** 2026-08-18
- **Context:** Go needs typed repository access while database history must remain reviewable and portable.
- **Decision:** Ent owns schemas and queries; Atlas generates reviewed SQL only under `supabase/migrations/`.
- **Alternatives:** runtime auto-migration; handwritten competing migration tree.
- **Consequences:** Runtime auto-migration stays disabled and migration contracts are tested against an empty database.

## ADR-003: Paper-only execution boundary

- **Status:** accepted
- **Date:** 2026-08-18
- **Context:** The product researches and evaluates trades without financial custody or blockchain-write risk.
- **Decision:** Simulate $100 entries and exits from read-only executable quotes; forbid wallets, signing, transaction building, swaps, Trigger orders, and all blockchain writes.
- **Alternatives:** wallet-backed execution; third-party trade automation.
- **Consequences:** A forbidden-dependency script and code review guard the boundary.

## ADR-004: Restricted Hermes evidence adapter

- **Status:** accepted
- **Date:** 2026-08-18
- **Context:** LLM-derived research context is useful but must not become an execution authority or secret sink.
- **Decision:** Use a dedicated restricted local Hermes profile pinned to `gpt-5.6-sol` for sanitized evidence-only verdicts.
- **Alternatives:** unrestricted existing profile; no LLM evidence.
- **Consequences:** No secrets or raw provider payloads may cross the adapter; deterministic policy remains authoritative.

## ADR-005: Loopback-only read-only React/Vite dashboard

- **Status:** accepted
- **Date:** 2026-08-18
- **Context:** One local operator needs visibility, not user accounts or control-plane mutation.
- **Decision:** Serve one React/Vite page on loopback through a bounded Go read API; do not use Clerk, routing, global state, component kits, or direct Supabase browser access.
- **Alternatives:** Supabase browser client; authenticated full-stack app.
- **Consequences:** Fewer credentials and attack surfaces; all state changes remain in the Go process.
