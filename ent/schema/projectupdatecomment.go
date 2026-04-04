package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ProjectUpdateComment stores first-class discussion attached to a project update thread.
type ProjectUpdateComment struct {
	ent.Schema
}

// Fields returns the ProjectUpdateComment schema fields.
func (ProjectUpdateComment) Fields() []ent.Field {
	return []ent.Field{
		uuidField(),
		field.UUID("thread_id", uuidZero()),
		field.Text("body_markdown").NotEmpty(),
		field.String("created_by").NotEmpty(),
		createdAtField(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
		field.Time("edited_at").Optional().Nillable(),
		field.Int("edit_count").Default(0),
		field.String("last_edited_by").Optional().Nillable(),
		field.Bool("is_deleted").Default(false),
		field.Time("deleted_at").Optional().Nillable(),
		field.String("deleted_by").Optional().Nillable(),
	}
}

// Edges returns the ProjectUpdateComment schema edges.
func (ProjectUpdateComment) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("thread", ProjectUpdateThread.Type).
			Ref("comments").
			Field("thread_id").
			Unique().
			Required(),
		edge.To("revisions", ProjectUpdateCommentRevision.Type),
	}
}

// Indexes returns the ProjectUpdateComment schema indexes.
func (ProjectUpdateComment) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("thread_id", "created_at", "id"),
	}
}
