package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ProjectRepoMirror defines the project-level local mirror for a repository on a machine.
type ProjectRepoMirror struct {
	ent.Schema
}

func (ProjectRepoMirror) Fields() []ent.Field {
	return []ent.Field{
		uuidField(),
		field.UUID("project_repo_id", uuidZero()),
		field.UUID("machine_id", uuidZero()),
		field.String("local_path").NotEmpty(),
		field.Enum("state").
			Values("missing", "provisioning", "ready", "stale", "syncing", "error", "deleting").
			Default("missing"),
		field.String("head_commit").Optional(),
		field.Time("last_synced_at").Optional().Nillable(),
		field.Time("last_verified_at").Optional().Nillable(),
		field.Text("last_error").Optional(),
		createdAtField(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (ProjectRepoMirror) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("project_repo", ProjectRepo.Type).
			Ref("mirrors").
			Field("project_repo_id").
			Unique().
			Required(),
		edge.From("machine", Machine.Type).
			Ref("project_repo_mirrors").
			Field("machine_id").
			Unique().
			Required(),
	}
}

func (ProjectRepoMirror) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("project_repo_id", "machine_id").Unique(),
		index.Fields("machine_id", "state"),
	}
}
