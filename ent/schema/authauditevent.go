package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type AuthAuditEvent struct {
	ent.Schema
}

func (AuthAuditEvent) Fields() []ent.Field {
	return []ent.Field{
		uuidField(),
		field.UUID("user_id", uuidZero()).
			Optional().
			Nillable(),
		field.UUID("session_id", uuidZero()).
			Optional().
			Nillable(),
		field.String("actor_id").Default(""),
		field.String("event_type").NotEmpty(),
		field.Text("message").Default(""),
		field.JSON("metadata", map[string]any{}).Default(emptyMap),
		createdAtField(),
	}
}

func (AuthAuditEvent) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id", "created_at"),
		index.Fields("session_id", "created_at"),
		index.Fields("event_type", "created_at"),
	}
}
