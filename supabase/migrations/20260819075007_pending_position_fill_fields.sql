-- Allow a newly admitted PENDING paper position to exist before its observed fill.
ALTER TABLE "paper_positions" ALTER COLUMN "entry_price" DROP NOT NULL;
ALTER TABLE "paper_positions" ALTER COLUMN "token_quantity" DROP NOT NULL;
