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
		field.String("name").NotEmpty(),
		field.Enum("type").
			Values("coding", "test", "doc", "security", "deploy", "refine-harness", "custom"),
		field.String("harness_path").NotEmpty(),
		field.JSON("hooks", map[string]any{}).
			Default(emptyMap),
		field.Int("max_concurrent").Default(3),
		field.Int("max_retry_attempts").Default(3),
		field.Int("timeout_minutes").Default(60),
		field.Int("stall_timeout_minutes").Default(5),
		field.Int("version").Default(1),
		field.Bool("is_active").Default(true),
		field.UUID("pickup_status_id", uuidZero()),
		field.UUID("finish_status_id", uuidZero()).
			Optional().
			Nillable(),
	}
}

func (Workflow) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("project", Project.Type).
			Ref("workflows").
			Field("project_id").
			Unique().
			Required(),
		edge.From("pickup_status", TicketStatus.Type).
			Ref("pickup_workflows").
			Field("pickup_status_id").
			Unique().
			Required(),
		edge.From("finish_status", TicketStatus.Type).
			Ref("finish_workflows").
			Field("finish_status_id").
			Unique(),
		edge.To("tickets", Ticket.Type),
		edge.To("scheduled_jobs", ScheduledJob.Type),
	}
}

func (Workflow) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("project_id", "name").Unique(),
		index.Fields("project_id", "is_active"),
	}
}
