package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type OrganizationMembership struct {
	ent.Schema
}

func (OrganizationMembership) Fields() []ent.Field {
	return []ent.Field{
		uuidField(),
		field.UUID("organization_id", uuidZero()),
		field.UUID("user_id", uuidZero()).Optional().Nillable(),
		field.String("email").NotEmpty(),
		field.Enum("role").Values("owner", "admin", "member"),
		field.Enum("status").Values("invited", "active", "suspended", "removed").Default("invited"),
		field.String("invited_by").Default(""),
		field.Time("invited_at").Default(time.Now),
		field.Time("accepted_at").Optional().Nillable(),
		field.Time("suspended_at").Optional().Nillable(),
		field.Time("removed_at").Optional().Nillable(),
		createdAtField(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (OrganizationMembership) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).
			Ref("memberships").
			Field("organization_id").
			Unique().
			Required(),
		edge.From("user", User.Type).
			Ref("organization_memberships").
			Field("user_id").
			Unique(),
		edge.To("invitations", OrganizationInvitation.Type),
	}
}

func (OrganizationMembership) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organization_id", "email").Unique(),
		index.Fields("organization_id", "status"),
		index.Fields("user_id", "status"),
	}
}
