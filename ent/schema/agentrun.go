package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// AgentRun defines the ent schema for runtime-managed agent execution records.
type AgentRun struct {
	ent.Schema
}

// Fields returns the AgentRun schema fields.
func (AgentRun) Fields() []ent.Field {
	return []ent.Field{
		uuidField(),
		field.UUID("agent_id", uuidZero()),
		field.UUID("workflow_id", uuidZero()),
		field.UUID("workflow_version_id", uuidZero()).
			Optional().
			Nillable(),
		field.UUID("ticket_id", uuidZero()),
		field.UUID("provider_id", uuidZero()),
		textArrayField("skill_version_ids"),
		field.Enum("status").
			Values("launching", "ready", "executing", "completed", "errored", "interrupted", "terminated"),
		field.String("session_id").Optional(),
		field.Time("runtime_started_at").Optional().Nillable(),
		field.Time("terminal_at").Optional().Nillable(),
		field.Time("snapshot_materialized_at").Optional().Nillable(),
		field.String("last_error").Optional(),
		field.Time("last_heartbeat_at").Optional().Nillable(),
		field.Int64("input_tokens").Default(0),
		field.Int64("output_tokens").Default(0),
		field.Int64("cached_input_tokens").Default(0),
		field.Int64("cache_creation_input_tokens").Default(0),
		field.Int64("reasoning_tokens").Default(0),
		field.Int64("prompt_tokens").Default(0),
		field.Int64("candidate_tokens").Default(0),
		field.Int64("tool_tokens").Default(0),
		field.Int64("total_tokens").Default(0),
		field.String("current_step_status").Optional().Nillable(),
		field.Text("current_step_summary").Optional().Nillable(),
		field.Time("current_step_changed_at").Optional().Nillable(),
		field.Enum("completion_summary_status").
			Values("pending", "completed", "failed").
			Optional().
			Nillable(),
		field.Text("completion_summary_markdown").Optional().Nillable(),
		field.JSON("completion_summary_json", map[string]any{}).Optional(),
		field.JSON("completion_summary_input", map[string]any{}).Optional(),
		field.Time("completion_summary_generated_at").Optional().Nillable(),
		field.Text("completion_summary_error").Optional().Nillable(),
		createdAtField(),
	}
}

// Edges returns the AgentRun schema edges.
func (AgentRun) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("agent", Agent.Type).
			Ref("runs").
			Field("agent_id").
			Unique().
			Required(),
		edge.From("workflow", Workflow.Type).
			Ref("agent_runs").
			Field("workflow_id").
			Unique().
			Required(),
		edge.From("workflow_version", WorkflowVersion.Type).
			Ref("agent_runs").
			Field("workflow_version_id").
			Unique(),
		edge.From("ticket", Ticket.Type).
			Ref("agent_runs").
			Field("ticket_id").
			Unique().
			Required(),
		edge.From("provider", AgentProvider.Type).
			Ref("agent_runs").
			Field("provider_id").
			Unique().
			Required(),
		edge.To("current_for_ticket", Ticket.Type),
		edge.To("ticket_repo_workspaces", TicketRepoWorkspace.Type),
		edge.To("agent_trace_events", AgentTraceEvent.Type),
		edge.To("agent_step_events", AgentStepEvent.Type),
	}
}

// Indexes returns the AgentRun schema indexes.
func (AgentRun) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("agent_id", "status", "last_heartbeat_at"),
		index.Fields("provider_id", "status"),
		index.Fields("ticket_id", "created_at"),
		index.Fields("ticket_id", "terminal_at"),
	}
}
