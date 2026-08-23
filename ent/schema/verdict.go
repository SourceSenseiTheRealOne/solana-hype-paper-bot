package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type Verdict struct{ ent.Schema }
func (Verdict) Fields() []ent.Field { return []ent.Field{
	field.String("provider").NotEmpty(), field.String("model").NotEmpty(), field.String("outcome").NotEmpty(), field.JSON("evidence", map[string]any{}), field.Time("created_at").Default(nowUTC).Immutable(),
} }
func (Verdict) Edges() []ent.Edge { return []ent.Edge{edge.From("candidate", Candidate.Type).Ref("verdicts").Unique().Required()} }
