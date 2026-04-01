package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type SkillVersion struct {
	ent.Schema
}

func (SkillVersion) Fields() []ent.Field {
	return []ent.Field{
		uuidField(),
		field.UUID("skill_id", uuidZero()),
		field.Int("version"),
		field.Text("content_markdown"),
		field.String("content_hash").NotEmpty(),
		field.String("bundle_hash").Optional(),
		field.JSON("manifest_json", map[string]any{}).Default(emptyMap),
		field.Int64("size_bytes").Default(0),
		field.Int("file_count").Default(0),
		field.String("created_by").Default("system:workflow-service"),
		createdAtField(),
	}
}

func (SkillVersion) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("skill", Skill.Type).
			Ref("versions").
			Field("skill_id").
			Unique().
			Required(),
		edge.To("files", SkillVersionFile.Type),
		edge.To("required_by_bindings", WorkflowSkillBinding.Type),
	}
}

func (SkillVersion) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("skill_id", "version").Unique(),
	}
}
