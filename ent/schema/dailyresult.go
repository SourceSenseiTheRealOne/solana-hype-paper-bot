package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type DailyResult struct{ ent.Schema }
func (DailyResult) Fields() []ent.Field { return []ent.Field{
	field.Time("utc_date"), field.String("strategy_version").NotEmpty(), field.Int("daily_admitted_count").NonNegative().Max(30), field.Int64("realized_pnl_micros").Default(0), field.Time("created_at").Default(nowUTC).Immutable(), field.Time("updated_at").Default(nowUTC).UpdateDefault(nowUTC),
} }
func (DailyResult) Indexes() []ent.Index { return []ent.Index{index.Fields("utc_date", "strategy_version").Unique()} }
