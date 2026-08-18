package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type SocialSnapshot struct{ ent.Schema }
func (SocialSnapshot) Fields() []ent.Field { return []ent.Field{
	field.JSON("social_evidence", map[string]any{}), field.Int("score"), field.Time("observed_at").Default(nowUTC), field.Time("created_at").Default(nowUTC).Immutable(),
} }
func (SocialSnapshot) Edges() []ent.Edge { return []ent.Edge{edge.From("candidate", Candidate.Type).Ref("social_snapshots").Unique().Required()} }
