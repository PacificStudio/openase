package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type UserIdentity struct {
	ent.Schema
}

func (UserIdentity) Fields() []ent.Field {
	return []ent.Field{
		uuidField(),
		field.UUID("user_id", uuidZero()),
		field.String("issuer").NotEmpty(),
		field.String("subject").NotEmpty(),
		field.String("email").Default(""),
		field.Bool("email_verified").Default(false),
		field.Int("claims_version").Default(1),
		field.Text("raw_claims_json").Default("{}"),
		field.Time("last_synced_at").Default(time.Now),
		createdAtField(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (UserIdentity) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("issuer", "subject").Unique(),
		index.Fields("user_id"),
		index.Fields("email"),
	}
}
