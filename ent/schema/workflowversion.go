package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	workflowdomain "github.com/BetterAndBetterII/openase/internal/domain/workflow"
)

type WorkflowVersion struct {
	ent.Schema
}

func (WorkflowVersion) Fields() []ent.Field {
	return []ent.Field{
		uuidField(),
		field.UUID("workflow_id", uuidZero()),
		field.Int("version"),
		field.Text("content_markdown"),
		field.String("name").NotEmpty(),
		field.String("type").
			Validate(func(value string) error {
				_, err := workflowdomain.ParseTypeLabel(value)
				return err
			}),
		field.String("role_slug").Optional(),
		field.String("role_name").Optional(),
		field.Text("role_description").Optional(),
		textArrayField("pickup_status_ids"),
		textArrayField("finish_status_ids"),
		field.String("harness_path").NotEmpty(),
		field.JSON("hooks", map[string]any{}).Default(emptyMap),
		textArrayField("platform_access_allowed"),
		field.Int("max_concurrent").Default(0),
		field.Int("max_retry_attempts").Default(3),
		field.Int("timeout_minutes").Default(60),
		field.Int("stall_timeout_minutes").Default(5),
		field.Bool("is_active").Default(true),
		field.String("content_hash").NotEmpty(),
		field.String("created_by").Default("system:workflow-service"),
		createdAtField(),
	}
}

func (WorkflowVersion) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("workflow", Workflow.Type).
			Ref("versions").
			Field("workflow_id").
			Unique().
			Required(),
		edge.To("agent_runs", AgentRun.Type),
	}
}

func (WorkflowVersion) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("workflow_id", "version").Unique(),
	}
}
