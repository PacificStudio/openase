package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ProjectUpdateThread stores curated project-level status updates.
type ProjectUpdateThread struct {
	ent.Schema
}

// Fields returns the ProjectUpdateThread schema fields.
func (ProjectUpdateThread) Fields() []ent.Field {
	return []ent.Field{
		uuidField(),
		field.UUID("project_id", uuidZero()),
		field.Enum("status").
			Values("on_track", "at_risk", "off_track"),
		field.String("title").NotEmpty(),
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
		field.Time("last_activity_at").Default(time.Now),
		field.Int("comment_count").Default(0),
	}
}

// Edges returns the ProjectUpdateThread schema edges.
func (ProjectUpdateThread) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("project", Project.Type).
			Ref("update_threads").
			Field("project_id").
			Unique().
			Required(),
		edge.To("revisions", ProjectUpdateThreadRevision.Type),
		edge.To("comments", ProjectUpdateComment.Type),
	}
}

// Indexes returns the ProjectUpdateThread schema indexes.
func (ProjectUpdateThread) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("project_id", "last_activity_at", "id"),
		index.Fields("project_id", "created_at", "id"),
	}
}
