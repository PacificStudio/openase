package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// AgentStepEvent defines the ent schema for human-readable agent step changes.
type AgentStepEvent struct {
	ent.Schema
}

func (AgentStepEvent) Fields() []ent.Field {
	return []ent.Field{
		uuidField(),
		field.UUID("project_id", uuidZero()),
		field.UUID("ticket_id", uuidZero()),
		field.UUID("agent_id", uuidZero()),
		field.UUID("agent_run_id", uuidZero()),
		field.String("step_status").NotEmpty(),
		field.Text("summary").Optional(),
		field.UUID("source_trace_event_id", uuidZero()).
			Optional().
			Nillable(),
		createdAtField(),
	}
}

func (AgentStepEvent) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("project", Project.Type).
			Ref("agent_step_events").
			Field("project_id").
			Unique().
			Required(),
		edge.From("ticket", Ticket.Type).
			Ref("agent_step_events").
			Field("ticket_id").
			Unique().
			Required(),
		edge.From("agent", Agent.Type).
			Ref("agent_step_events").
			Field("agent_id").
			Unique().
			Required(),
		edge.From("agent_run", AgentRun.Type).
			Ref("agent_step_events").
			Field("agent_run_id").
			Unique().
			Required(),
		edge.From("source_trace_event", AgentTraceEvent.Type).
			Ref("step_events").
			Field("source_trace_event_id").
			Unique(),
	}
}

func (AgentStepEvent) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("agent_run_id", "created_at"),
		index.Fields("ticket_id", "created_at"),
		index.Fields("project_id", "created_at"),
	}
}
