package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type LocalBootstrapAuthRequest struct {
	ent.Schema
}

func (LocalBootstrapAuthRequest) Fields() []ent.Field {
	return []ent.Field{
		uuidField(),
		field.String("code_hash").Sensitive().NotEmpty(),
		field.String("nonce_hash").Sensitive().NotEmpty(),
		field.String("purpose").Default("browser_session"),
		field.String("requested_by").Default(""),
		field.Time("expires_at"),
		field.UUID("used_session_id", uuidZero()).
			Optional().
			Nillable(),
		field.Time("used_at").Optional().Nillable(),
		createdAtField(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (LocalBootstrapAuthRequest) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("expires_at"),
		index.Fields("used_at"),
	}
}
