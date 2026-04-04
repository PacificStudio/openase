package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type WorkflowSkillBinding struct {
	ent.Schema
}

func (WorkflowSkillBinding) Fields() []ent.Field {
	return []ent.Field{
		uuidField(),
		field.UUID("workflow_id", uuidZero()),
		field.UUID("skill_id", uuidZero()),
		field.UUID("required_version_id", uuidZero()).
			Optional().
			Nillable(),
		createdAtField(),
	}
}

func (WorkflowSkillBinding) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("workflow", Workflow.Type).
			Ref("skill_bindings").
			Field("workflow_id").
			Unique().
			Required(),
		edge.From("skill", Skill.Type).
			Ref("workflow_bindings").
			Field("skill_id").
			Unique().
			Required(),
		edge.From("required_version", SkillVersion.Type).
			Ref("required_by_bindings").
			Field("required_version_id").
			Unique(),
	}
}

func (WorkflowSkillBinding) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("workflow_id", "skill_id").Unique(),
		index.Fields("skill_id", "workflow_id"),
	}
}
