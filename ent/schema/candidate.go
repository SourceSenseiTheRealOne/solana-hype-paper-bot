package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type Candidate struct{ ent.Schema }

func (Candidate) Fields() []ent.Field {
	return []ent.Field{
		field.String("network").NotEmpty(),
		field.String("mint_address").NotEmpty(),
		field.String("pool_address").NotEmpty(),
		field.Time("discovered_at").Default(nowUTC),
		field.Time("created_at").Default(nowUTC).Immutable(),
		field.Time("updated_at").Default(nowUTC).UpdateDefault(nowUTC),
	}
}
func (Candidate) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("snapshots", CandidateSnapshot.Type), edge.To("social_snapshots", SocialSnapshot.Type),
		edge.To("verdicts", Verdict.Type), edge.To("decisions", TradeDecision.Type), edge.To("positions", PaperPosition.Type),
	}
}
func (Candidate) Indexes() []ent.Index { return []ent.Index{index.Fields("network", "mint_address", "pool_address").Unique()} }
