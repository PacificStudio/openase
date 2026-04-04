package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type User struct {
	ent.Schema
}

func (User) Fields() []ent.Field {
	return []ent.Field{
		uuidField(),
		field.Enum("status").Values("active", "disabled").Default("active"),
		field.String("primary_email").Default(""),
		field.String("display_name").Default(""),
		field.String("avatar_url").Default(""),
		field.Time("last_login_at").Optional().Nillable(),
		createdAtField(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (User) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("primary_email"),
	}
}
