package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/BetterAndBetterII/openase/internal/domain/ticketing"
)

type TicketStatus struct {
	ent.Schema
}

func (TicketStatus) Fields() []ent.Field {
	return []ent.Field{
		uuidField(),
		field.UUID("project_id", uuidZero()),
		field.String("name").NotEmpty(),
		field.Enum("stage").
			Values(
				ticketing.StatusStageBacklog.String(),
				ticketing.StatusStageUnstarted.String(),
				ticketing.StatusStageStarted.String(),
				ticketing.StatusStageCompleted.String(),
				ticketing.StatusStageCanceled.String(),
			).
			Default(ticketing.DefaultStatusStage.String()),
		field.String("color").NotEmpty(),
		field.String("icon").Optional(),
		field.Int("position").Default(0),
		field.Int("max_active_runs").Optional().Nillable(),
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
		edge.From("pickup_workflows", Workflow.Type).
			Ref("pickup_statuses"),
		edge.From("finish_workflows", Workflow.Type).
			Ref("finish_statuses"),
	}
}

func (TicketStatus) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("project_id", "name").Unique(),
		index.Fields("project_id", "position"),
	}
}
