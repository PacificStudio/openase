package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type Project struct {
	ent.Schema
}

func (Project) Fields() []ent.Field {
	return []ent.Field{
		uuidField(),
		field.UUID("organization_id", uuidZero()),
		field.String("name").NotEmpty(),
		field.String("slug").NotEmpty(),
		field.Text("description").Optional(),
		field.Enum("status").
			Values("planning", "active", "paused", "archived").
			Default("planning"),
		field.UUID("default_workflow_id", uuidZero()).
			Optional().
			Nillable(),
		field.UUID("default_agent_provider_id", uuidZero()).
			Optional().
			Nillable(),
		field.Int("max_concurrent_agents").Default(5),
	}
}

func (Project) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).
			Ref("projects").
			Field("organization_id").
			Unique().
			Required(),
		edge.To("repos", ProjectRepo.Type),
		edge.To("statuses", TicketStatus.Type),
		edge.To("workflows", Workflow.Type),
		edge.To("tickets", Ticket.Type),
		edge.To("agents", Agent.Type),
		edge.To("agent_tokens", AgentToken.Type),
		edge.To("scheduled_jobs", ScheduledJob.Type),
		edge.To("activity_events", ActivityEvent.Type),
		edge.To("default_workflow", Workflow.Type).
			Field("default_workflow_id").
			Unique(),
		edge.To("default_agent_provider", AgentProvider.Type).
			Field("default_agent_provider_id").
			Unique(),
	}
}

func (Project) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organization_id", "slug").Unique(),
	}
}
