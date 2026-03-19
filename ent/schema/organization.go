package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type Organization struct {
	ent.Schema
}

func (Organization) Fields() []ent.Field {
	return []ent.Field{
		uuidField(),
		field.String("name").NotEmpty(),
		field.String("slug").NotEmpty(),
		field.UUID("default_agent_provider_id", uuidZero()).
			Optional().
			Nillable(),
	}
}

func (Organization) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("projects", Project.Type),
		edge.To("providers", AgentProvider.Type),
		edge.To("default_agent_provider", AgentProvider.Type).
			Field("default_agent_provider_id").
			Unique(),
	}
}

func (Organization) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("slug").Unique(),
	}
}
