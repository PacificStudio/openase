package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type RoleBinding struct {
	ent.Schema
}

func (RoleBinding) Fields() []ent.Field {
	return []ent.Field{
		uuidField(),
		field.Enum("scope_kind").Values("instance", "organization", "project"),
		field.String("scope_id").Default(""),
		field.Enum("subject_kind").Values("user", "group"),
		field.String("subject_key").NotEmpty(),
		field.String("role_key").NotEmpty(),
		field.String("granted_by").Default(""),
		field.Time("expires_at").Optional().Nillable(),
		createdAtField(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (RoleBinding) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("scope_kind", "scope_id", "subject_kind", "subject_key", "role_key").Unique(),
		index.Fields("subject_kind", "subject_key"),
		index.Fields("scope_kind", "scope_id"),
	}
}
