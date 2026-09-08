# Technology Stack

## Selection status

- **Client-mandated stack:** yes
- **UI selection:** React 19 + Vite 8, strict TypeScript, native semantic components and CSS; React Bits is explicitly not used.
- **Selection date:** 2026-08-18

## Verified host toolchains (2026-08-18 UTC)

| Tool | Selected version | Host evidence | Official source |
| --- | --- | --- | --- |
| Go | 1.26.2 | `go version go1.26.2 windows/amd64` | https://go.dev/dl/ |
| Node.js | 24.13.1 LTS | `node --version` | https://nodejs.org/en/blog/release/v24.18.1 |
| pnpm | 10.30.1 | `pnpm.cmd --version` | https://pnpm.io/blog/releases/10.29 |
| Supabase CLI | 2.78.1 | `supabase --version` | https://supabase.com/docs/guides/local-development/cli/getting-started |
| Docker Engine client | 29.6.1 | `docker version --format '{{.Client.Version}}'` | https://docs.docker.com/engine/release-notes/ |
| Docker Compose | 5.3.0 | `docker compose version` | https://docs.docker.com/compose/releases/ |
| React | 19.2.8 | generated dashboard manifest and lockfile | https://react.dev/blog/2025/10/01/react-19-2 |
| Vite | 8.2.1 | create-vite 9.1.2 generated dashboard manifest and lockfile | https://vite.dev/blog/announcing-vite8 |
| Ent / Atlas | Ent 0.14.6; Atlas 1.3.0 Docker image | Ent-generated schema and reviewed migration evidence | https://entgo.io/docs/versioned-migrations |

The official Supabase registry release observed at scaffold time is 2.115.0; the lab-installed CLI is 2.78.1. The project records and uses the installed CLI until an explicit local-stack upgrade is performed, because such upgrades require stopping affected stacks and may require volume migration.

On this host, the `pnpm` and `corepack` Git-Bash shims incorrectly resolve `C:\\c\\Program Files\\...`; `pnpm.cmd` is the verified native launcher. CI and non-Git-Bash environments use normal `pnpm`. The legacy Go-run Atlas command module is incompatible with Go 1.26, so the project uses the official pinned Atlas 1.3.0 Docker image for migration generation.

## Data and migration ownership

The project owns local Supabase `project_id = solana-hype-paper-bot`. Supabase/Postgres replaces a standalone Postgres container. Ent schemas and typed queries own Go persistence; reviewed Atlas-generated SQL under `supabase/migrations/` is the sole durable migration history. Seed data contains no product mock data. Direct browser Supabase use is disabled.

## Approved deviations

- No Clerk: the dashboard is loopback-only and read-only, with no mutation surface or user accounts.
- No Redis: no measured cache, queue, or coordination requirement exists.
- No React Bits: the approved dashboard is intentionally small, data-first, and uses project-owned semantic React/CSS components.
- No live blockchain dependency: quote and route providers are read-only evidence sources only.
