package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
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

func (User) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("organization_memberships", OrganizationMembership.Type),
		edge.To("accepted_organization_invitations", OrganizationInvitation.Type),
		edge.To("api_keys", UserAPIKey.Type),
	}
}

func (User) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("primary_email"),
	}
}
