package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// TicketDependency defines the ent schema for ticket dependency edges.
type TicketDependency struct {
	ent.Schema
}

// Fields returns the TicketDependency schema fields.
func (TicketDependency) Fields() []ent.Field {
	return []ent.Field{
		uuidField(),
		field.UUID("source_ticket_id", uuidZero()),
		field.UUID("target_ticket_id", uuidZero()),
		field.Enum("type").
			Values("blocks", "sub-issue").
			Default("blocks"),
	}
}

// Edges returns the TicketDependency schema edges.
func (TicketDependency) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("source_ticket", Ticket.Type).
			Ref("outgoing_dependencies").
			Field("source_ticket_id").
			Unique().
			Required(),
		edge.From("target_ticket", Ticket.Type).
			Ref("incoming_dependencies").
			Field("target_ticket_id").
			Unique().
			Required(),
	}
}

// Indexes returns the TicketDependency schema indexes.
func (TicketDependency) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("source_ticket_id", "target_ticket_id", "type").Unique(),
		index.Fields("target_ticket_id", "type"),
		index.Fields("source_ticket_id"),
	}
}
