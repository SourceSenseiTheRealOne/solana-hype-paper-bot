package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type TradeDecision struct{ ent.Schema }
func (TradeDecision) Fields() []ent.Field { return []ent.Field{
	field.String("idempotency_key").NotEmpty(), field.String("outcome").NotEmpty(), field.JSON("rule_results", map[string]any{}), field.Time("created_at").Default(nowUTC).Immutable(),
} }
func (TradeDecision) Edges() []ent.Edge { return []ent.Edge{edge.From("candidate", Candidate.Type).Ref("decisions").Unique().Required()} }
func (TradeDecision) Indexes() []ent.Index { return []ent.Index{index.Fields("idempotency_key").Unique()} }
