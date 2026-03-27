package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// AgentTraceEvent defines the ent schema for fine-grained agent runtime trace output.
type AgentTraceEvent struct {
	ent.Schema
}

func (AgentTraceEvent) Fields() []ent.Field {
	return []ent.Field{
		uuidField(),
		field.UUID("project_id", uuidZero()),
		field.UUID("ticket_id", uuidZero()),
		field.UUID("agent_id", uuidZero()),
		field.UUID("agent_run_id", uuidZero()),
		field.Int64("sequence"),
		field.String("provider").NotEmpty(),
		field.String("kind").NotEmpty(),
		field.String("stream").NotEmpty(),
		field.Text("text").Optional(),
		field.JSON("payload", map[string]any{}).Default(emptyMap),
		createdAtField(),
	}
}

func (AgentTraceEvent) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("project", Project.Type).
			Ref("agent_trace_events").
			Field("project_id").
			Unique().
			Required(),
		edge.From("ticket", Ticket.Type).
			Ref("agent_trace_events").
			Field("ticket_id").
			Unique().
			Required(),
		edge.From("agent", Agent.Type).
			Ref("agent_trace_events").
			Field("agent_id").
			Unique().
			Required(),
		edge.From("agent_run", AgentRun.Type).
			Ref("agent_trace_events").
			Field("agent_run_id").
			Unique().
			Required(),
		edge.To("step_events", AgentStepEvent.Type),
	}
}

func (AgentTraceEvent) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("agent_run_id", "sequence").Unique(),
		index.Fields("ticket_id", "created_at"),
		index.Fields("agent_id", "created_at"),
		index.Fields("project_id", "created_at"),
	}
}
