package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

type BotState struct{ ent.Schema }
func (BotState) Fields() []ent.Field { return []ent.Field{field.String("state_key").Unique().NotEmpty(), field.JSON("value", map[string]any{}), field.Time("created_at").Default(nowUTC).Immutable(), field.Time("updated_at").Default(nowUTC).UpdateDefault(nowUTC)} }
