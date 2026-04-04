package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type UserGroupMembership struct {
	ent.Schema
}

func (UserGroupMembership) Fields() []ent.Field {
	return []ent.Field{
		uuidField(),
		field.UUID("user_id", uuidZero()),
		field.String("issuer").NotEmpty(),
		field.String("group_key").NotEmpty(),
		field.String("group_name").Default(""),
		field.Time("last_synced_at").Default(time.Now),
		createdAtField(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (UserGroupMembership) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "issuer", "group_key").Unique(),
		index.Fields("group_key"),
	}
}
