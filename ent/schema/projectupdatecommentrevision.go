package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ProjectUpdateCommentRevision stores immutable snapshots for each saved comment version.
type ProjectUpdateCommentRevision struct {
	ent.Schema
}

// Fields returns the ProjectUpdateCommentRevision schema fields.
func (ProjectUpdateCommentRevision) Fields() []ent.Field {
	return []ent.Field{
		uuidField(),
		field.UUID("comment_id", uuidZero()),
		field.Int("revision_number").Positive(),
		field.Text("body_markdown").NotEmpty(),
		field.String("edited_by").NotEmpty(),
		field.Time("edited_at").Default(time.Now),
		field.String("edit_reason").Optional().Nillable(),
	}
}

// Edges returns the ProjectUpdateCommentRevision schema edges.
func (ProjectUpdateCommentRevision) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("comment", ProjectUpdateComment.Type).
			Ref("revisions").
			Field("comment_id").
			Unique().
			Required(),
	}
}

// Indexes returns the ProjectUpdateCommentRevision schema indexes.
func (ProjectUpdateCommentRevision) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("comment_id", "revision_number").Unique(),
		index.Fields("comment_id", "edited_at", "id"),
	}
}
