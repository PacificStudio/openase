package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type ApprovalPolicyRule struct {
	ent.Schema
}

func (ApprovalPolicyRule) Fields() []ent.Field {
	return []ent.Field{
		uuidField(),
		field.Enum("scope_kind").Values("instance", "organization", "project"),
		field.String("scope_id").Default(""),
		field.String("action_key").NotEmpty(),
		field.String("require_role_key").Default(""),
		field.String("require_ticket_status").Default(""),
		field.Bool("enabled").Default(true),
		createdAtField(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (ApprovalPolicyRule) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("scope_kind", "scope_id", "action_key").Unique(),
	}
}
