package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type ProjectRepo struct {
	ent.Schema
}

func (ProjectRepo) Fields() []ent.Field {
	return []ent.Field{
		uuidField(),
		field.UUID("project_id", uuidZero()),
		field.String("name").NotEmpty(),
		field.String("repository_url").NotEmpty(),
		field.String("default_branch").Default("main"),
		field.String("workspace_dirname").Default(""),
		textArrayField("labels"),
	}
}

func (ProjectRepo) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("project", Project.Type).
			Ref("repos").
			Field("project_id").
			Unique().
			Required(),
		edge.To("ticket_scopes", TicketRepoScope.Type),
		edge.To("ticket_repo_workspaces", TicketRepoWorkspace.Type),
	}
}

func (ProjectRepo) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("project_id", "name").Unique(),
		index.Fields("labels").
			Annotations(entsql.IndexType("GIN")),
	}
}
