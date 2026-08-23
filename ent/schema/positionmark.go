package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type PositionMark struct{ ent.Schema }
func (PositionMark) Fields() []ent.Field {
	return []ent.Field{
		field.String("price").NotEmpty(), field.String("net_output_amount").Optional().Nillable(), field.Int64("fee_estimate").Optional().Nillable().NonNegative(), field.Int64("return_bps").Optional().Nillable(), field.String("quote_hash").Optional().Nillable(), field.Enum("route_state").Values("EXECUTABLE", "NO_ROUTE").Default("EXECUTABLE"), field.Int64("mfe_bps").Optional().Nillable(), field.Int64("mae_bps").Optional().Nillable(), field.Int("no_route_count").Default(0).NonNegative(), field.Time("observed_at").Default(nowUTC), field.Time("created_at").Default(nowUTC).Immutable(),
	}
}
func (PositionMark) Edges() []ent.Edge { return []ent.Edge{edge.From("position", PaperPosition.Type).Ref("marks").Unique().Required()} }
