package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type ActivityEvent struct {
	ent.Schema
}

func (ActivityEvent) Fields() []ent.Field {
	return []ent.Field{
		uuidField(),
		field.UUID("project_id", uuidZero()),
		field.UUID("ticket_id", uuidZero()).
			Optional().
			Nillable(),
		field.UUID("agent_id", uuidZero()).
			Optional().
			Nillable(),
		field.String("event_type").NotEmpty(),
		field.Text("message").Optional(),
		field.JSON("metadata", map[string]any{}).Default(emptyMap),
		createdAtField(),
	}
}

func (ActivityEvent) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("project", Project.Type).
			Ref("activity_events").
			Field("project_id").
			Unique().
			Required(),
		edge.From("ticket", Ticket.Type).
			Ref("activity_events").
			Field("ticket_id").
			Unique(),
		edge.From("agent", Agent.Type).
			Ref("activity_events").
			Field("agent_id").
			Unique(),
	}
}

func (ActivityEvent) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("ticket_id", "created_at"),
		index.Fields("project_id", "created_at"),
	}
}
