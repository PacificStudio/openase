package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type Workflow struct {
	ent.Schema
}

func (Workflow) Fields() []ent.Field {
	return []ent.Field{
		uuidField(),
		field.UUID("project_id", uuidZero()),
		field.UUID("agent_id", uuidZero()).
			Optional().
			Nillable(),
		field.UUID("current_version_id", uuidZero()).
			Optional().
			Nillable(),
		field.String("name").NotEmpty(),
		field.Enum("type").
			Values("coding", "test", "doc", "security", "deploy", "refine-harness", "custom"),
		field.String("harness_path").NotEmpty(),
		field.JSON("hooks", map[string]any{}).
			Default(emptyMap),
		field.Int("max_concurrent").Default(0),
		field.Int("max_retry_attempts").Default(3),
		field.Int("timeout_minutes").Default(60),
		field.Int("stall_timeout_minutes").Default(5),
		field.Int("version").Default(1),
		field.Bool("is_active").Default(true),
	}
}

func (Workflow) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("project", Project.Type).
			Ref("workflows").
			Field("project_id").
			Unique().
			Required(),
		edge.From("agent", Agent.Type).
			Ref("workflows").
			Field("agent_id").
			Unique(),
		edge.To("current_version", WorkflowVersion.Type).
			Field("current_version_id").
			Unique(),
		edge.To("versions", WorkflowVersion.Type),
		edge.To("skill_bindings", WorkflowSkillBinding.Type),
		edge.To("pickup_statuses", TicketStatus.Type).
			Required(),
		edge.To("finish_statuses", TicketStatus.Type).
			Required(),
		edge.To("tickets", Ticket.Type),
		edge.To("agent_runs", AgentRun.Type),
		edge.To("scheduled_jobs", ScheduledJob.Type),
	}
}

func (Workflow) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("project_id", "name").Unique(),
		index.Fields("project_id", "is_active"),
	}
}
