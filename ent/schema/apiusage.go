package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type APIUsage struct{ ent.Schema }
func (APIUsage) Fields() []ent.Field { return []ent.Field{field.String("provider").NotEmpty(), field.Time("utc_date"), field.Int("request_count").NonNegative(), field.Int("failure_count").NonNegative(), field.Time("created_at").Default(nowUTC).Immutable(), field.Time("updated_at").Default(nowUTC).UpdateDefault(nowUTC)} }
func (APIUsage) Indexes() []ent.Index { return []ent.Index{index.Fields("provider", "utc_date").Unique()} }
