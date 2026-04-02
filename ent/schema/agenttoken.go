package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type AgentToken struct {
	ent.Schema
}

func (AgentToken) Fields() []ent.Field {
	return []ent.Field{
		uuidField(),
		field.UUID("agent_id", uuidZero()).Optional().Nillable(),
		field.UUID("project_id", uuidZero()),
		field.UUID("ticket_id", uuidZero()).Optional().Nillable(),
		field.UUID("conversation_id", uuidZero()).Optional().Nillable(),
		field.Enum("principal_kind").Values("ticket_agent", "project_conversation"),
		field.UUID("principal_id", uuidZero()),
		field.String("principal_name").NotEmpty(),
		field.String("token_hash").NotEmpty(),
		field.JSON("scopes", []string{}).
			Default([]string{}),
		field.Time("expires_at"),
		createdAtField(),
		field.Time("last_used_at").Optional().Nillable(),
	}
}

func (AgentToken) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("agent", Agent.Type).
			Ref("tokens").
			Field("agent_id").
			Unique(),
		edge.From("project", Project.Type).
			Ref("agent_tokens").
			Field("project_id").
			Unique().
			Required(),
		edge.From("ticket", Ticket.Type).
			Ref("agent_tokens").
			Field("ticket_id").
			Unique(),
		edge.From("conversation", ChatConversation.Type).
			Ref("agent_tokens").
			Field("conversation_id").
			Unique(),
	}
}

func (AgentToken) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("token_hash").Unique(),
		index.Fields("project_id", "ticket_id"),
		index.Fields("project_id", "conversation_id"),
		index.Fields("principal_kind", "principal_id"),
		index.Fields("expires_at"),
	}
}
