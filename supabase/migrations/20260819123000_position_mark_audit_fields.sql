-- Preserve bounded executable/no-route observations for deterministic exits and restart recovery.
ALTER TABLE "position_marks" ADD COLUMN "net_output_amount" text;
ALTER TABLE "position_marks" ADD COLUMN "fee_estimate" bigint;
ALTER TABLE "position_marks" ADD COLUMN "return_bps" bigint;
ALTER TABLE "position_marks" ADD COLUMN "quote_hash" text;
ALTER TABLE "position_marks" ADD COLUMN "route_state" text NOT NULL DEFAULT 'EXECUTABLE' CHECK ("route_state" IN ('EXECUTABLE', 'NO_ROUTE'));
ALTER TABLE "position_marks" ADD COLUMN "mfe_bps" bigint;
ALTER TABLE "position_marks" ADD COLUMN "mae_bps" bigint;
ALTER TABLE "position_marks" ADD COLUMN "no_route_count" integer NOT NULL DEFAULT 0 CHECK ("no_route_count" >= 0);
