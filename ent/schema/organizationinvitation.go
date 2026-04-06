package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type OrganizationInvitation struct {
	ent.Schema
}

func (OrganizationInvitation) Fields() []ent.Field {
	return []ent.Field{
		uuidField(),
		field.UUID("organization_id", uuidZero()),
		field.UUID("membership_id", uuidZero()),
		field.UUID("accepted_by_user_id", uuidZero()).Optional().Nillable(),
		field.String("email").NotEmpty(),
		field.Enum("role").Values("owner", "admin", "member"),
		field.Enum("status").Values("pending", "accepted", "canceled", "expired").Default("pending"),
		field.String("invited_by").Default(""),
		field.String("invite_token_hash").NotEmpty(),
		field.Time("expires_at"),
		field.Time("sent_at").Default(time.Now),
		field.Time("accepted_at").Optional().Nillable(),
		field.Time("canceled_at").Optional().Nillable(),
		createdAtField(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (OrganizationInvitation) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).
			Ref("invitations").
			Field("organization_id").
			Unique().
			Required(),
		edge.From("membership", OrganizationMembership.Type).
			Ref("invitations").
			Field("membership_id").
			Unique().
			Required(),
		edge.From("accepted_by_user", User.Type).
			Ref("accepted_organization_invitations").
			Field("accepted_by_user_id").
			Unique(),
	}
}

func (OrganizationInvitation) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("invite_token_hash").Unique(),
		index.Fields("organization_id", "status"),
		index.Fields("membership_id", "status"),
	}
}
