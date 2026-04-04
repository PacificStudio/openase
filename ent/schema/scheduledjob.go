package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ScheduledJob defines the ent schema for scheduled workflow jobs.
type ScheduledJob struct {
	ent.Schema
}

// Fields returns the ScheduledJob schema fields.
func (ScheduledJob) Fields() []ent.Field {
	return []ent.Field{
		uuidField(),
		field.UUID("project_id", uuidZero()),
		field.String("name").NotEmpty(),
		field.String("cron_expression").NotEmpty(),
		field.UUID("workflow_id", uuidZero()).
			Optional().
			Nillable(),
		field.JSON("ticket_template", map[string]any{}).Default(emptyMap),
		field.Bool("is_enabled").Default(true),
		field.Time("last_run_at").Optional().Nillable(),
		field.Time("next_run_at").Optional().Nillable(),
	}
}

// Edges returns the ScheduledJob schema edges.
func (ScheduledJob) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("project", Project.Type).
			Ref("scheduled_jobs").
			Field("project_id").
			Unique().
			Required(),
		edge.From("workflow", Workflow.Type).
			Ref("scheduled_jobs").
			Field("workflow_id").
			Unique(),
	}
}

// Indexes returns the ScheduledJob schema indexes.
func (ScheduledJob) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("project_id", "name").Unique(),
		index.Fields("next_run_at").
			Annotations(entsql.IndexWhere("is_enabled = true")),
	}
}
