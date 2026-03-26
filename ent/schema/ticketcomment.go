package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// TicketComment stores first-class user discussion attached to a ticket.
type TicketComment struct {
	ent.Schema
}

// Fields returns the TicketComment schema fields.
func (TicketComment) Fields() []ent.Field {
	return []ent.Field{
		uuidField(),
		field.UUID("ticket_id", uuidZero()),
		field.Text("body").NotEmpty(),
		field.String("created_by").NotEmpty(),
		createdAtField(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Edges returns the TicketComment schema edges.
func (TicketComment) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("ticket", Ticket.Type).
			Ref("comments").
			Field("ticket_id").
			Unique().
			Required(),
	}
}

// Indexes returns the TicketComment schema indexes.
func (TicketComment) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("ticket_id", "created_at", "id"),
	}
}
