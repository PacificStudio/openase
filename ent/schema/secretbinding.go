package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/google/uuid"
)

// SecretBinding defines the ent schema for runtime secret references.
type SecretBinding struct {
	ent.Schema
}

func (SecretBinding) Fields() []ent.Field {
	return []ent.Field{
		uuidField(),
		field.UUID("organization_id", uuidZero()),
		field.UUID("project_id", uuidZero()).Default(uuidZero),
		field.UUID("secret_id", uuid.UUID{}),
		field.Enum("scope_kind").
			Values("organization", "project", "workflow", "agent", "ticket"),
		field.UUID("scope_resource_id", uuid.UUID{}),
		field.String("binding_key").NotEmpty(),
		createdAtField(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (SecretBinding) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("secret", Secret.Type).
			Ref("bindings").
			Field("secret_id").
			Unique().
			Required(),
	}
}

func (SecretBinding) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("scope_kind", "scope_resource_id", "binding_key").Unique(),
		index.Fields("organization_id", "project_id", "binding_key"),
		index.Fields("secret_id"),
	}
}
