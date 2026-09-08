-- Bind each paper position to the exact admitted idempotency decision.
ALTER TABLE "paper_positions" ADD COLUMN "trade_decision_position" bigint NOT NULL;
ALTER TABLE "paper_positions" ADD CONSTRAINT "paper_positions_trade_decisions_position" FOREIGN KEY ("trade_decision_position") REFERENCES "trade_decisions" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
CREATE UNIQUE INDEX "paperposition_trade_decision_position_key" ON "paper_positions" ("trade_decision_position");
