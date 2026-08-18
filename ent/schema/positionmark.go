package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type PositionMark struct{ ent.Schema }
func (PositionMark) Fields() []ent.Field { return []ent.Field{field.String("price").NotEmpty(), field.Time("observed_at").Default(nowUTC), field.Time("created_at").Default(nowUTC).Immutable()} }
func (PositionMark) Edges() []ent.Edge { return []ent.Edge{edge.From("position", PaperPosition.Type).Ref("marks").Unique().Required()} }
