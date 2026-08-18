# Product Context

## Identity

- **Name:** Solana Hype Paper Bot
- **Owner:** SourceSensei
- **Trust boundary:** personal
- **Source/runtime:** public source; local Windows-hosted runtime only.

## Problem and users

A single local operator needs a reproducible way to research new Solana token pools, score bounded public market/social evidence, and evaluate a fixed paper-trading policy without touching a wallet or executing a blockchain transaction.

## In scope

- Discover and paginate public Solana pools; persist candidate evidence and deterministic decisions.
- Open at most three virtual positions concurrently and at most 30 new $10 paper positions per UTC day.
- Apply +30% take-profit, -15% stop-loss, and a 60-minute timeout.
- Persist decisions, marks, closures, daily results, and retained local reports through Ent and project-owned local Supabase/Postgres.
- Serve a small loopback-only, read-only React/Vite dashboard through a bounded Go HTTP API.
- Request a restricted Hermes verdict as evidence only; it must never receive secrets or raw provider payloads.

## Out of scope

- Wallets, seed phrases, private keys, signing, transaction construction, token swaps, Trigger orders, blockchain writes, or live execution.
- Authentication, Clerk, browser-to-Supabase traffic, mutation APIs, Redis, mobile apps, routing, component kits, and social-network posting.

## Non-functional requirements

All timestamps and daily quota buckets are UTC. Money and quantities are never persisted as `float64`. Provider calls are bounded, validated, timeout-controlled, and redact secrets and raw payloads. Logs and reports are local ignored data.

## Success evidence

A fresh local stack rebuilds the empty schema from reviewed migrations, executes a fully simulated discovery-to-close flow, survives restart recovery without duplicate admission, and renders its results in the read-only dashboard. No source dependency may introduce live-trading capability.
