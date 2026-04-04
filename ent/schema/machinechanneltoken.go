package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type MachineChannelToken struct {
	ent.Schema
}

func (MachineChannelToken) Fields() []ent.Field {
	return []ent.Field{
		uuidField(),
		field.UUID("machine_id", uuidZero()),
		field.String("token_hash").NotEmpty(),
		field.Enum("status").Values("active", "revoked").Default("active"),
		field.Time("expires_at"),
		createdAtField(),
		field.Time("last_used_at").Optional().Nillable(),
		field.Time("revoked_at").Optional().Nillable(),
	}
}

func (MachineChannelToken) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("machine", Machine.Type).
			Ref("channel_tokens").
			Field("machine_id").
			Unique().
			Required(),
	}
}

func (MachineChannelToken) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("token_hash").Unique(),
		index.Fields("machine_id", "status"),
		index.Fields("expires_at"),
	}
}
