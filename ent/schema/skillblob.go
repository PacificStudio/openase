package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type SkillBlob struct {
	ent.Schema
}

func (SkillBlob) Fields() []ent.Field {
	return []ent.Field{
		uuidField(),
		field.String("sha256").NotEmpty(),
		field.Int64("size_bytes").Default(0),
		field.Enum("compression").
			Values("none", "gzip").
			Default("none"),
		field.Bytes("content_bytes"),
		createdAtField(),
	}
}

func (SkillBlob) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("files", SkillVersionFile.Type),
	}
}

func (SkillBlob) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("sha256").Unique(),
	}
}
