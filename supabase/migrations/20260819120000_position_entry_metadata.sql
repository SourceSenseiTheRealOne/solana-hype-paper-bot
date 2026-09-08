-- Preserve the quote-native entry basis required for deterministic paper exits and restart recovery.
ALTER TABLE "paper_positions" ADD COLUMN "quote_mint" text;
ALTER TABLE "paper_positions" ADD COLUMN "mint_address" text;
ALTER TABLE "paper_positions" ADD COLUMN "entry_input_amount" text;
ALTER TABLE "paper_positions" ADD COLUMN "entry_network_fee_micros" bigint;
ALTER TABLE "paper_positions" ADD COLUMN "entry_priority_fee_micros" bigint;
