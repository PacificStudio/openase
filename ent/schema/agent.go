package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Agent defines the ent schema for project agents.
type Agent struct {
	ent.Schema
}

// Fields returns the Agent schema fields.
func (Agent) Fields() []ent.Field {
	return []ent.Field{
		uuidField(),
		field.UUID("provider_id", uuidZero()),
		field.UUID("project_id", uuidZero()),
		field.String("name").NotEmpty(),
		field.Enum("status").
			Values("idle", "claimed", "running", "failed", "terminated").
			Default("idle"),
		field.UUID("current_ticket_id", uuidZero()).
			Optional().
			Nillable(),
		field.String("session_id").Optional(),
		field.String("workspace_path").Optional(),
		field.Strings("capabilities").
			SchemaType(textArrayColumn()).
			Optional(),
		field.Int64("total_tokens_used").Default(0),
		field.Int("total_tickets_completed").Default(0),
		field.Time("last_heartbeat_at").Optional().Nillable(),
	}
}

// Edges returns the Agent schema edges.
func (Agent) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("provider", AgentProvider.Type).
			Ref("agents").
			Field("provider_id").
			Unique().
			Required(),
		edge.From("project", Project.Type).
			Ref("agents").
			Field("project_id").
			Unique().
			Required(),
		edge.To("current_ticket", Ticket.Type).
			Field("current_ticket_id").
			Unique(),
		edge.To("assigned_tickets", Ticket.Type),
		edge.To("tokens", AgentToken.Type),
		edge.To("activity_events", ActivityEvent.Type),
	}
}

// Indexes returns the Agent schema indexes.
func (Agent) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("project_id", "name").Unique(),
		index.Fields("project_id", "status", "last_heartbeat_at"),
		index.Fields("capabilities").
			Annotations(entsql.IndexType("GIN")),
	}
}
