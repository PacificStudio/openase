package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// TicketRepoScope defines the ent schema for per-ticket repository scopes.
type TicketRepoScope struct {
	ent.Schema
}

// Fields returns the TicketRepoScope schema fields.
func (TicketRepoScope) Fields() []ent.Field {
	return []ent.Field{
		uuidField(),
		field.UUID("ticket_id", uuidZero()),
		field.UUID("repo_id", uuidZero()),
		field.String("branch_name"),
		field.String("pull_request_url").Optional(),
		field.Enum("pr_status").
			Values("none", "open", "changes_requested", "approved", "merged", "closed").
			Default("none"),
		field.Enum("ci_status").
			Values("pending", "passing", "failing").
			Default("pending"),
	}
}

// Edges returns the TicketRepoScope schema edges.
func (TicketRepoScope) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("ticket", Ticket.Type).
			Ref("repo_scopes").
			Field("ticket_id").
			Unique().
			Required(),
		edge.From("repo", ProjectRepo.Type).
			Ref("ticket_scopes").
			Field("repo_id").
			Unique().
			Required(),
	}
}

// Indexes returns the TicketRepoScope schema indexes.
func (TicketRepoScope) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("ticket_id", "repo_id").Unique(),
		index.Fields("repo_id", "branch_name"),
		index.Fields("ticket_id"),
	}
}
