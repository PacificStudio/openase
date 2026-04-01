package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ProjectUpdateThreadRevision stores immutable snapshots for each saved thread version.
type ProjectUpdateThreadRevision struct {
	ent.Schema
}

// Fields returns the ProjectUpdateThreadRevision schema fields.
func (ProjectUpdateThreadRevision) Fields() []ent.Field {
	return []ent.Field{
		uuidField(),
		field.UUID("thread_id", uuidZero()),
		field.Int("revision_number").Positive(),
		field.Enum("status").
			Values("on_track", "at_risk", "off_track"),
		field.String("title").NotEmpty(),
		field.Text("body_markdown").NotEmpty(),
		field.String("edited_by").NotEmpty(),
		field.Time("edited_at").Default(time.Now),
		field.String("edit_reason").Optional().Nillable(),
	}
}

// Edges returns the ProjectUpdateThreadRevision schema edges.
func (ProjectUpdateThreadRevision) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("thread", ProjectUpdateThread.Type).
			Ref("revisions").
			Field("thread_id").
			Unique().
			Required(),
	}
}

// Indexes returns the ProjectUpdateThreadRevision schema indexes.
func (ProjectUpdateThreadRevision) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("thread_id", "revision_number").Unique(),
		index.Fields("thread_id", "edited_at", "id"),
	}
}
