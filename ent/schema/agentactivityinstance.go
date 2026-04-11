package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// AgentActivityInstance defines the current state for a provider/runtime activity.
type AgentActivityInstance struct {
	ent.Schema
}

func (AgentActivityInstance) Fields() []ent.Field {
	return []ent.Field{
		uuidField(),
		field.UUID("project_id", uuidZero()),
		field.UUID("ticket_id", uuidZero()).
			Optional().
			Nillable(),
		field.UUID("agent_id", uuidZero()),
		field.UUID("agent_run_id", uuidZero()),
		field.String("provider").NotEmpty(),
		field.String("activity_kind").NotEmpty(),
		field.String("activity_id").NotEmpty(),
		field.String("id_source").NotEmpty(),
		field.String("identity_confidence").NotEmpty(),
		field.String("parent_activity_id").Optional().Nillable(),
		field.String("thread_id").Optional().Nillable(),
		field.String("turn_id").Optional().Nillable(),
		field.Text("command").Optional().Nillable(),
		field.String("tool_name").Optional().Nillable(),
		field.String("title").Optional().Nillable(),
		field.String("status").NotEmpty(),
		field.Text("live_text").Optional().Nillable(),
		field.Text("final_text").Optional().Nillable(),
		field.Int("live_text_bytes").Default(0),
		field.Int("final_text_bytes").Default(0),
		field.JSON("metadata", map[string]any{}).Default(emptyMap),
		field.Time("started_at").Optional().Nillable(),
		field.Time("updated_at"),
		field.Time("completed_at").Optional().Nillable(),
		createdAtField(),
	}
}

func (AgentActivityInstance) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("project", Project.Type).
			Ref("agent_activity_instances").
			Field("project_id").
			Unique().
			Required(),
		edge.From("ticket", Ticket.Type).
			Ref("agent_activity_instances").
			Field("ticket_id").
			Unique(),
		edge.From("agent", Agent.Type).
			Ref("agent_activity_instances").
			Field("agent_id").
			Unique().
			Required(),
		edge.From("agent_run", AgentRun.Type).
			Ref("agent_activity_instances").
			Field("agent_run_id").
			Unique().
			Required(),
	}
}

func (AgentActivityInstance) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("agent_run_id", "activity_kind", "activity_id").Unique(),
		index.Fields("agent_run_id", "updated_at"),
		index.Fields("project_id", "updated_at"),
		index.Fields("ticket_id", "updated_at"),
	}
}
