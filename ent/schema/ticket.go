package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type Ticket struct {
	ent.Schema
}

func (Ticket) Fields() []ent.Field {
	return []ent.Field{
		uuidField(),
		field.UUID("project_id", uuidZero()),
		field.String("identifier").NotEmpty(),
		field.String("title").NotEmpty(),
		field.Text("description").Optional(),
		field.UUID("status_id", uuidZero()),
		field.Bool("archived").Default(false),
		field.Enum("priority").
			Values("urgent", "high", "medium", "low").
			Optional(),
		field.Enum("type").
			Values("feature", "bugfix", "refactor", "chore", "epic").
			Default("feature"),
		field.UUID("workflow_id", uuidZero()).
			Optional().
			Nillable(),
		field.UUID("current_run_id", uuidZero()).
			Optional().
			Nillable(),
		field.UUID("target_machine_id", uuidZero()).
			Optional().
			Nillable(),
		field.String("created_by").NotEmpty(),
		field.UUID("parent_ticket_id", uuidZero()).
			Optional().
			Nillable(),
		field.String("external_ref").Optional(),
		field.Int("attempt_count").Default(0),
		field.Int("consecutive_errors").Default(0),
		field.Time("next_retry_at").Optional().Nillable(),
		field.Bool("retry_paused").Default(false),
		field.String("pause_reason").Optional(),
		field.Int("stall_count").Default(0),
		field.String("retry_token").Optional(),
		field.Int("harness_version").Default(0),
		field.Float("budget_usd").
			SchemaType(currencyColumn()).
			Default(0),
		field.Int64("cost_tokens_input").Default(0),
		field.Int64("cost_tokens_output").Default(0),
		field.Float("cost_amount").
			SchemaType(currencyColumn()).
			Default(0),
		field.JSON("metadata", map[string]any{}).Default(emptyMap),
		field.Time("started_at").Optional().Nillable(),
		field.Time("completed_at").Optional().Nillable(),
		createdAtField(),
	}
}

func (Ticket) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("project", Project.Type).
			Ref("tickets").
			Field("project_id").
			Unique().
			Required(),
		edge.From("status", TicketStatus.Type).
			Ref("tickets").
			Field("status_id").
			Unique().
			Required(),
		edge.From("workflow", Workflow.Type).
			Ref("tickets").
			Field("workflow_id").
			Unique(),
		edge.From("current_run", AgentRun.Type).
			Ref("current_for_ticket").
			Field("current_run_id").
			Unique(),
		edge.From("target_machine", Machine.Type).
			Ref("target_tickets").
			Field("target_machine_id").
			Unique(),
		edge.From("parent", Ticket.Type).
			Ref("children").
			Field("parent_ticket_id").
			Unique(),
		edge.To("children", Ticket.Type),
		edge.To("repo_scopes", TicketRepoScope.Type),
		edge.To("comments", TicketComment.Type),
		edge.To("external_links", TicketExternalLink.Type),
		edge.To("agent_tokens", AgentToken.Type),
		edge.To("agent_trace_events", AgentTraceEvent.Type),
		edge.To("agent_step_events", AgentStepEvent.Type),
		edge.To("activity_events", ActivityEvent.Type),
		edge.To("agent_runs", AgentRun.Type),
		edge.To("repo_workspaces", TicketRepoWorkspace.Type),
		edge.To("outgoing_dependencies", TicketDependency.Type),
		edge.To("incoming_dependencies", TicketDependency.Type),
	}
}

func (Ticket) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("project_id", "identifier").Unique(),
		index.Fields("project_id", "status_id", "current_run_id", "priority", "created_at"),
		index.Fields("project_id", "status_id"),
		index.Fields("project_id", "archived", "created_at"),
		index.Fields("project_id", "external_ref"),
	}
}
