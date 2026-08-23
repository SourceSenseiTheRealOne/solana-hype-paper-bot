# Solana Hype Paper Bot

Local, public-source research software for discovering and evaluating new Solana token pools with **paper trades only**. It records bounded public market and social evidence, simulates fixed-notional entries/exits, and serves a read-only dashboard on loopback.

## Hard safety boundary

This repository must never contain wallet support, seed phrases, private keys, signing, transaction construction, swap execution, Trigger orders, or any blockchain write. A paper position is always a local database record; it is never an order.

## Policy defaults

- Paper notional: **$10**
- Concurrent virtual positions: **3 maximum**
- New virtual positions: **30 maximum per UTC day**
- Minimum market activity: **20 combined buys and sells per five minutes**
- Exit policy: **+50% take profit**, **-20% stop loss**, or **45-minute timeout**
- Timestamps and quota buckets: **UTC**

## Local setup

1. Copy `.env.example` to ignored `.env.local` and provide only the server-side credentials you own.

   - `PAPER_AUTOMATION_ENABLED=false` is the safe default. Set it to `true` only for local paper automation.
   - `HELIUS_API_KEY` is required when automation is enabled; the bot derives the Helius Mainnet RPC URL itself and does not fall back to public Solana RPC.
   - `PAPER_QUOTE_MINT` defaults to canonical Solana Mainnet USDC. It is the read-only quote mint used for virtual paper entries and exits.
   - `TWITTERAPIIO_API_KEY` enables bounded, exact-mint social evidence searches.
   - `HERMES_API_KEY` enables the restricted local Hermes verdict adapter with provider `openai-codex` and model `gpt-5.6-sol`.
   - `JUPITER_API_KEY` enables Jupiter read-only quote requests. The bot uses quote responses as evidence only.
2. Initialize the project-owned Supabase files from the Coding Lab root:

   ```bash
   uv run labctl init-supabase solana-hype-paper-bot --json
   ```

3. Install dashboard dependencies:

   ```bash
   pnpm --dir dashboard install
   ```

   On this Windows Git-Bash host, use `pnpm.cmd` because the Corepack shell shim is broken.

## Local containers

Compose runs only the paper-bot API and dashboard. It never creates Postgres: use the project-owned local Supabase stack and place its server-side `DATABASE_URL` in ignored `.env.local` before starting containers.

```bash
docker compose up --build
```

- Dashboard: `http://127.0.0.1:4173/`
- Read-only API health: `http://127.0.0.1:8080/healthz`
- Read-only dashboard API: `http://127.0.0.1:8080/api/v1/dashboard`

The API container receives `APP_ENV=container` and listens on its private container interface; Compose publishes application ports only through `127.0.0.1`. Its ignored logs and retained reports live in the named `runtime-data` volume at `/app/var`. Provider credentials remain server-side in `.env.local`; the browser receives none.

When paper automation is enabled, the Go process scans every 5 minutes and monitors open virtual positions every 30 seconds. Each scan uses DexScreener token hints first, then the GeckoTerminal new-pool page, evaluates at most five matched fresh pools, calls TwitterAPI.io only after deterministic eligibility, stores the Hermes verdict, and can only admit/open local virtual paper positions.

## Verification

```bash
go test ./...
go test -race ./...
go vet ./...
pnpm --dir dashboard run verify
bash scripts/verify-no-live-trading-deps.sh
bash tests/verify-compose-topology.sh
bash tests/verify-github-actions.sh
git diff --check
```

Do not run `docker compose config` in a credential-bearing checkout: Compose may interpolate ignored dotenv values into command output. The static topology verifier above checks the Compose safety boundary without loading dotenv files or requiring Docker.

The application, database schema, provider adapters, and container runtime are being implemented in bounded RED → GREEN → REFACTOR slices. See `context/README.md` for the approved architecture and security boundaries.
