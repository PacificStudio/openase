package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// AgentRawEvent defines append-only provider event facts for an agent run.
type AgentRawEvent struct {
	ent.Schema
}

func (AgentRawEvent) Fields() []ent.Field {
	return []ent.Field{
		uuidField(),
		field.UUID("project_id", uuidZero()),
		field.UUID("ticket_id", uuidZero()).
			Optional().
			Nillable(),
		field.UUID("agent_id", uuidZero()),
		field.UUID("agent_run_id", uuidZero()),
		field.String("dedup_key").NotEmpty(),
		field.String("provider").NotEmpty(),
		field.String("provider_event_kind").NotEmpty(),
		field.String("provider_event_subtype").Optional(),
		field.String("provider_event_id").Optional().Nillable(),
		field.String("thread_id").Optional().Nillable(),
		field.String("turn_id").Optional().Nillable(),
		field.String("activity_hint_id").Optional().Nillable(),
		field.Time("occurred_at"),
		field.JSON("payload", map[string]any{}).Default(emptyMap),
		field.Text("text_excerpt").Optional(),
		createdAtField(),
	}
}

func (AgentRawEvent) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("project", Project.Type).
			Ref("agent_raw_events").
			Field("project_id").
			Unique().
			Required(),
		edge.From("ticket", Ticket.Type).
			Ref("agent_raw_events").
			Field("ticket_id").
			Unique(),
		edge.From("agent", Agent.Type).
			Ref("agent_raw_events").
			Field("agent_id").
			Unique().
			Required(),
		edge.From("agent_run", AgentRun.Type).
			Ref("agent_raw_events").
			Field("agent_run_id").
			Unique().
			Required(),
	}
}

func (AgentRawEvent) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("agent_run_id", "dedup_key").Unique(),
		index.Fields("agent_run_id", "occurred_at", "id"),
		index.Fields("project_id", "occurred_at"),
		index.Fields("ticket_id", "occurred_at"),
		index.Fields("agent_id", "occurred_at"),
	}
}
