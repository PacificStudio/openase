package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
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
