# UI Context

## Confirmed UI system

- **Selection:** React/Vite with strict TypeScript and project-owned CSS.
- **Confirmation date:** 2026-08-18.
- **React Bits:** not used.
- **Accessible primitive layer:** native semantic HTML; no component kit is justified for one read-only page.

## Design language

A compact, dark, data-first operations page with clear status, metric cards, readable tables, and an equity chart. Respect `prefers-reduced-motion`; do not add decorative motion. Use high-contrast text, visible focus rings, responsive table overflow handling, and explicit empty/loading/error states.

## Component architecture

`App` composes `StatusPanel`, `MetricCards`, `OpenPositionsTable`, `ClosedTradesTable`, `CandidatesTable`, and `EquityChart`. Components consume typed API data and polling hooks only. The shared API client owns endpoint, response validation, and polling errors.

## Accessibility and boundaries

Use table captions/headers, button labels, semantic headings, keyboard-reachable controls, and color-independent state labels. Frontend code never accesses secrets, provider APIs, database code, or Supabase directly and never performs mutations.
