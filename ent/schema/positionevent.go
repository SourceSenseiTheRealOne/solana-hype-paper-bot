package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type PositionEvent struct{ ent.Schema }
func (PositionEvent) Fields() []ent.Field { return []ent.Field{field.String("event_type").NotEmpty(), field.JSON("details", map[string]any{}), field.Time("created_at").Default(nowUTC).Immutable()} }
func (PositionEvent) Edges() []ent.Edge { return []ent.Edge{edge.From("position", PaperPosition.Type).Ref("events").Unique().Required()} }
