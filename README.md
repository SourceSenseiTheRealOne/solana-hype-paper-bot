# Solana Hype Paper Bot

Local, public-source research software for discovering and evaluating new Solana token pools with **paper trades only**. It records bounded public market and social evidence, simulates fixed-notional entries/exits, and serves a read-only dashboard on loopback.

## Hard safety boundary

This repository must never contain wallet support, seed phrases, private keys, signing, transaction construction, swap execution, Trigger orders, or any blockchain write. A paper position is always a local database record; it is never an order.

## Policy defaults

- Paper notional: **$10**
- Concurrent virtual positions: **3 maximum**
- New virtual positions: **30 maximum per UTC day**
- Exit policy: **+30% take profit**, **-15% stop loss**, or **60-minute timeout**
- Timestamps and quota buckets: **UTC**

## Local setup

1. Copy `.env.example` to ignored `.env.local` and provide only the server-side credentials you own.
2. Initialize the project-owned Supabase files from the Coding Lab root:

   ```bash
   uv run labctl init-supabase solana-hype-paper-bot --json
   ```

3. Install dashboard dependencies:

   ```bash
   pnpm --dir dashboard install
   ```

   On this Windows Git-Bash host, use `pnpm.cmd` because the Corepack shell shim is broken.

## Verification

```bash
go test ./...
go test -race ./...
go vet ./...
pnpm --dir dashboard run verify
bash scripts/verify-no-live-trading-deps.sh
docker compose config
git diff --check
```

The application, database schema, provider adapters, and container runtime are being implemented in bounded RED → GREEN → REFACTOR slices. See `context/README.md` for the approved architecture and security boundaries.
