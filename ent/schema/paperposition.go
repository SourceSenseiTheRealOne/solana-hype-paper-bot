package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type PaperPosition struct{ ent.Schema }

func (PaperPosition) Fields() []ent.Field {
	return []ent.Field{
		field.Enum("state").Values("PENDING", "OPEN", "CLOSING", "CLOSED", "UNSELLABLE"), field.Int64("notional_micros").Positive(), field.String("strategy_version").NotEmpty().Default("legacy-unattributed"), field.Int("no_route_count").Default(0).NonNegative(), field.String("quote_mint").Optional().Nillable(), field.String("mint_address").Optional().Nillable(), field.String("entry_price").Optional().Nillable(), field.String("entry_input_amount").Optional().Nillable(), field.Int64("entry_network_fee_micros").Optional().Nillable(), field.Int64("entry_priority_fee_micros").Optional().Nillable(), field.String("token_quantity").Optional().Nillable(), field.Time("opened_at").Optional().Nillable(), field.Time("closed_at").Optional().Nillable(), field.Time("created_at").Default(nowUTC).Immutable(), field.Time("updated_at").Default(nowUTC).UpdateDefault(nowUTC),
	}
}
func (PaperPosition) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("candidate", Candidate.Type).Ref("positions").Unique().Required(), edge.From("decision", TradeDecision.Type).Ref("position").Unique().Required(), edge.To("marks", PositionMark.Type), edge.To("events", PositionEvent.Type),
	}
}
