package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type PaperPosition struct{ ent.Schema }
func (PaperPosition) Fields() []ent.Field { return []ent.Field{
	field.Enum("state").Values("PENDING", "OPEN", "CLOSING", "CLOSED", "UNSELLABLE"), field.Int64("notional_micros").Positive(), field.String("entry_price").NotEmpty(), field.String("token_quantity").NotEmpty(), field.Time("opened_at").Optional().Nillable(), field.Time("closed_at").Optional().Nillable(), field.Time("created_at").Default(nowUTC).Immutable(), field.Time("updated_at").Default(nowUTC).UpdateDefault(nowUTC),
} }
func (PaperPosition) Edges() []ent.Edge { return []ent.Edge{
	edge.From("candidate", Candidate.Type).Ref("positions").Unique().Required(), edge.To("marks", PositionMark.Type), edge.To("events", PositionEvent.Type),
} }
