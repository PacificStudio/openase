package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type TicketStage struct {
	ent.Schema
}

func (TicketStage) Fields() []ent.Field {
	return []ent.Field{
		uuidField(),
		field.UUID("project_id", uuidZero()),
		field.String("key").NotEmpty(),
		field.String("name").NotEmpty(),
		field.Int("position").Default(0),
		field.Int("max_active_runs").Optional().Nillable(),
		field.String("description").Optional(),
	}
}

func (TicketStage) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("project", Project.Type).
			Ref("stages").
			Field("project_id").
			Unique().
			Required(),
		edge.To("statuses", TicketStatus.Type),
	}
}

func (TicketStage) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("project_id", "key").Unique(),
		index.Fields("project_id", "position"),
	}
}
