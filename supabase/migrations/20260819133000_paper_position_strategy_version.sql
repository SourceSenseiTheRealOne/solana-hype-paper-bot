-- Preserve historical position attribution without assigning it to a future configured strategy.
ALTER TABLE "paper_positions" ADD COLUMN "strategy_version" character varying NOT NULL DEFAULT 'legacy-unattributed';
