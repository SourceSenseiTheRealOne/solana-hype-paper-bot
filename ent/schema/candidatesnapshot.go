package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type CandidateSnapshot struct{ ent.Schema }
func (CandidateSnapshot) Fields() []ent.Field { return []ent.Field{
	field.JSON("market_evidence", map[string]any{}), field.Time("observed_at").Default(nowUTC), field.Time("created_at").Default(nowUTC).Immutable(),
} }
func (CandidateSnapshot) Edges() []ent.Edge { return []ent.Edge{edge.From("candidate", Candidate.Type).Ref("snapshots").Unique().Required()} }
