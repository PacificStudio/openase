package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// TicketExternalLink defines the ent schema for external ticket links.
type TicketExternalLink struct {
	ent.Schema
}

// Fields returns the TicketExternalLink schema fields.
func (TicketExternalLink) Fields() []ent.Field {
	return []ent.Field{
		uuidField(),
		field.UUID("ticket_id", uuidZero()),
		field.Enum("link_type").
			Values("github_issue", "gitlab_issue", "jira_ticket", "github_pr", "gitlab_mr", "custom"),
		field.String("url").NotEmpty(),
		field.String("external_id").NotEmpty(),
		field.String("title").Optional(),
		field.String("status").Optional(),
		field.Enum("relation").
			Values("resolves", "related", "caused_by").
			Default("related"),
		createdAtField(),
	}
}

// Edges returns the TicketExternalLink schema edges.
func (TicketExternalLink) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("ticket", Ticket.Type).
			Ref("external_links").
			Field("ticket_id").
			Unique().
			Required(),
	}
}

// Indexes returns the TicketExternalLink schema indexes.
func (TicketExternalLink) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("ticket_id", "external_id").Unique(),
		index.Fields("external_id"),
	}
}
