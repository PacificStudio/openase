package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type BrowserSession struct {
	ent.Schema
}

func (BrowserSession) Fields() []ent.Field {
	return []ent.Field{
		uuidField(),
		field.UUID("user_id", uuidZero()),
		field.String("session_hash").Sensitive().NotEmpty(),
		field.String("device_kind").Default("unknown"),
		field.String("device_os").Default(""),
		field.String("device_browser").Default(""),
		field.String("device_label").Default(""),
		field.Time("expires_at"),
		field.Time("idle_expires_at"),
		field.String("csrf_secret").Sensitive().NotEmpty(),
		field.String("user_agent_hash").Default(""),
		field.String("ip_prefix").Default(""),
		field.Time("revoked_at").Optional().Nillable(),
		createdAtField(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (BrowserSession) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("session_hash").Unique(),
		index.Fields("user_id"),
		index.Fields("expires_at"),
	}
}
