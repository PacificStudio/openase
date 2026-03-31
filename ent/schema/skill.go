package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type Skill struct {
	ent.Schema
}

func (Skill) Fields() []ent.Field {
	return []ent.Field{
		uuidField(),
		field.UUID("project_id", uuidZero()),
		field.UUID("current_version_id", uuidZero()).
			Optional().
			Nillable(),
		field.String("name").NotEmpty(),
		field.String("description").NotEmpty(),
		field.Bool("is_builtin").Default(false),
		field.Bool("is_enabled").Default(true),
		field.String("created_by").Default("user:manual"),
		field.Time("archived_at").
			Optional().
			Nillable(),
		createdAtField(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (Skill) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("project", Project.Type).
			Ref("skills").
			Field("project_id").
			Unique().
			Required(),
		edge.To("current_version", SkillVersion.Type).
			Field("current_version_id").
			Unique(),
		edge.To("versions", SkillVersion.Type),
		edge.To("workflow_bindings", WorkflowSkillBinding.Type),
	}
}

func (Skill) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("project_id", "name").Unique(),
		index.Fields("project_id", "archived_at"),
	}
}
