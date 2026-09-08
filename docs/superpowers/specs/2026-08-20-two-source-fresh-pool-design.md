# Two-Source Fresh Pool Discovery Design

## Goal
Process bounded fresh Solana pools independently from DexScreener and GeckoTerminal/CoinGecko, preserving paper-only behavior and treating a cross-source match as corroboration rather than a prerequisite.

## Decisions
- DexScreener profiles are resolved through a bounded DexScreener token-pairs read to obtain a real Solana pool address and creation time.
- GeckoTerminal new pools are fetched independently every scan cycle.
- Inputs are deduplicated by canonical pool identity; the union is ordered newest-first and capped to five candidates per cycle.
- No minimum pool age applies. Pools older than 90 days are excluded.
- Future-dated pools are stored as at most five durable public-identity watches in the existing `bot_states` projection. A watch is eligible only at/after its advertised creation time and expires 30 minutes afterward.
- Launched watched pools must still obtain current market data and meet the normal $10,000 liquidity requirement before deterministic evaluation.
- Remove market-data age as a deterministic rule. Provider responses remain validated and deadline-bounded.
- Defaults and active non-secret local settings become: $10,000 liquidity, 10 five-minute transactions, 10% maximum entry impact, $100 paper notional.
- Existing caps, five-minute scan cadence, 30-second virtual monitoring, social budget, local Hermes advisory, atomic admission, read-only API, and paper-only restrictions remain unchanged.

## Safety
No wallet, signer, transaction, swap, order, or blockchain-write capability is added. New discovery calls remain public, GET-only, capped, single-attempt, timeout-controlled, and unavailable from browser routes.
