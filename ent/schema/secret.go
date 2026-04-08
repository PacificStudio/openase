package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Secret defines the ent schema for encrypted scoped secrets.
type Secret struct {
	ent.Schema
}

func (Secret) Fields() []ent.Field {
	return []ent.Field{
		uuidField(),
		field.UUID("organization_id", uuidZero()),
		field.UUID("project_id", uuidZero()).Default(uuidZero),
		field.Enum("scope_kind").
			Values("organization", "project"),
		field.String("name").NotEmpty(),
		field.Enum("kind").
			Values("opaque").
			Default("opaque"),
		field.Text("description").Default(""),
		field.String("algorithm").NotEmpty(),
		field.String("key_source").NotEmpty(),
		field.String("key_id").NotEmpty(),
		field.String("value_preview").NotEmpty(),
		field.String("nonce").NotEmpty().Sensitive(),
		field.Text("ciphertext").NotEmpty().Sensitive(),
		field.Time("rotated_at").Default(time.Now).UpdateDefault(time.Now),
		field.Time("disabled_at").Optional().Nillable(),
		createdAtField(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (Secret) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("bindings", SecretBinding.Type),
	}
}

func (Secret) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organization_id", "scope_kind", "project_id", "name").Unique(),
		index.Fields("organization_id", "project_id", "scope_kind", "disabled_at"),
	}
}
