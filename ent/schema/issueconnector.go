package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	domain "github.com/BetterAndBetterII/openase/internal/domain/issueconnector"
)

// IssueConnector defines the ent schema for project-scoped external issue connectors.
type IssueConnector struct {
	ent.Schema
}

func (IssueConnector) Fields() []ent.Field {
	return []ent.Field{
		uuidField(),
		field.UUID("project_id", uuidZero()),
		field.String("type").NotEmpty(),
		field.String("name").NotEmpty(),
		field.String("status").Default(string(domain.StatusActive)),
		field.JSON("config", domain.Config{}),
		field.Time("last_sync_at").Optional().Nillable(),
		field.Text("last_error").Optional(),
		field.JSON("stats", domain.SyncStats{}).Default(emptyConnectorStats),
		createdAtField(),
	}
}

func (IssueConnector) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("project", Project.Type).
			Ref("issue_connectors").
			Field("project_id").
			Unique().
			Required(),
	}
}

func (IssueConnector) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("project_id", "name").Unique(),
		index.Fields("project_id", "type"),
	}
}

func emptyConnectorStats() domain.SyncStats {
	return domain.SyncStats{}
}
