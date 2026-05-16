package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type UserAPIKey struct {
	ent.Schema
}

func (UserAPIKey) Fields() []ent.Field {
	return []ent.Field{
		uuidField(),
		field.UUID("user_id", uuidZero()),
		field.UUID("project_id", uuidZero()),
		field.String("name").NotEmpty(),
		field.String("token_prefix").NotEmpty(),
		field.String("token_hint").NotEmpty(),
		field.String("token_hash").NotEmpty(),
		field.JSON("scopes", []string{}).Default([]string{}),
		field.Enum("status").Values("active", "disabled", "revoked").Default("active"),
		field.Time("expires_at").Optional().Nillable(),
		field.Time("last_used_at").Optional().Nillable(),
		createdAtField(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
		field.Time("disabled_at").Optional().Nillable(),
		field.Time("revoked_at").Optional().Nillable(),
	}
}

func (UserAPIKey) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).
			Ref("api_keys").
			Field("user_id").
			Unique().
			Required(),
		edge.From("project", Project.Type).
			Ref("user_api_keys").
			Field("project_id").
			Unique().
			Required(),
	}
}

func (UserAPIKey) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("token_hash").Unique(),
		index.Fields("project_id", "user_id"),
		index.Fields("project_id", "status"),
		index.Fields("expires_at"),
	}
}
