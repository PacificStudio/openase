package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type TicketStatus struct {
	ent.Schema
}

func (TicketStatus) Fields() []ent.Field {
	return []ent.Field{
		uuidField(),
		field.UUID("project_id", uuidZero()),
		field.String("name").NotEmpty(),
		field.String("color").NotEmpty(),
		field.String("icon").Optional(),
		field.Int("position").Default(0),
		field.Bool("is_default").Default(false),
		field.String("description").Optional(),
	}
}

func (TicketStatus) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("project", Project.Type).
			Ref("statuses").
			Field("project_id").
			Unique().
			Required(),
		edge.To("tickets", Ticket.Type),
		edge.To("pickup_workflows", Workflow.Type),
		edge.To("finish_workflows", Workflow.Type),
	}
}

func (TicketStatus) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("project_id", "name").Unique(),
		index.Fields("project_id", "position"),
	}
}
