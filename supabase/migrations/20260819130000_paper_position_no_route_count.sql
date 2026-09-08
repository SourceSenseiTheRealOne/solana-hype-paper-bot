-- Keep recovery-safe no-route streak state on the position row; marks retain the audit history.
ALTER TABLE "paper_positions" ADD COLUMN "no_route_count" integer NOT NULL DEFAULT 0 CHECK ("no_route_count" >= 0);
