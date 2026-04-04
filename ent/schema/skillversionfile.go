package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type SkillVersionFile struct {
	ent.Schema
}

func (SkillVersionFile) Fields() []ent.Field {
	return []ent.Field{
		uuidField(),
		field.UUID("skill_version_id", uuidZero()),
		field.UUID("content_blob_id", uuidZero()),
		field.String("path").NotEmpty(),
		field.Enum("file_kind").
			Values("entrypoint", "metadata", "script", "reference", "asset").
			Default("asset"),
		field.String("media_type").Default("application/octet-stream"),
		field.Enum("encoding").
			Values("utf8", "base64", "binary").
			Default("binary"),
		field.Bool("is_executable").Default(false),
		field.Int64("size_bytes").Default(0),
		field.String("sha256").NotEmpty(),
		createdAtField(),
	}
}

func (SkillVersionFile) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("skill_version", SkillVersion.Type).
			Ref("files").
			Field("skill_version_id").
			Unique().
			Required(),
		edge.From("content_blob", SkillBlob.Type).
			Ref("files").
			Field("content_blob_id").
			Unique().
			Required(),
	}
}

func (SkillVersionFile) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("skill_version_id", "path").Unique(),
		index.Fields("content_blob_id"),
	}
}
