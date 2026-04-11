package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	agentplatformdomain "github.com/BetterAndBetterII/openase/internal/domain/agentplatform"
	githubauth "github.com/BetterAndBetterII/openase/internal/domain/githubauth"
	"github.com/google/uuid"
)

// Project defines the ent schema for projects.
type Project struct {
	ent.Schema
}

// Fields returns the Project schema fields.
func (Project) Fields() []ent.Field {
	return []ent.Field{
		uuidField(),
		field.UUID("organization_id", uuidZero()),
		field.String("name").NotEmpty(),
		field.String("slug").NotEmpty(),
		field.Text("description").Optional(),
		field.String("status").Default("Planned"),
		field.JSON("github_outbound_credential", &githubauth.StoredCredential{}).
			Optional(),
		field.JSON("github_token_probe", &githubauth.TokenProbe{}).
			Optional(),
		field.UUID("default_agent_provider_id", uuidZero()).
			Optional().
			Nillable(),
		field.JSON("project_ai_platform_access_allowed", []string{}).
			Default(func() []string {
				return append(
					[]string(nil),
					agentplatformdomain.SupportedScopesForPrincipalKind(
						agentplatformdomain.PrincipalKindProjectConversation,
					)...,
				)
			}),
		field.JSON("accessible_machine_ids", []uuid.UUID{}).
			Default(emptyUUIDs),
		field.Int("max_concurrent_agents").Default(0),
		field.Text("agent_run_summary_prompt").Optional(),
		field.Bool("project_ai_retention_enabled").Default(false),
		field.Int("project_ai_retention_keep_latest_n").Default(0),
		field.Int("project_ai_retention_keep_recent_days").Default(0),
	}
}

// Edges returns the Project schema edges.
func (Project) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).
			Ref("projects").
			Field("organization_id").
			Unique().
			Required(),
		edge.To("repos", ProjectRepo.Type),
		edge.To("skills", Skill.Type),
		edge.To("statuses", TicketStatus.Type),
		edge.To("workflows", Workflow.Type),
		edge.To("tickets", Ticket.Type),
		edge.To("agents", Agent.Type),
		edge.To("agent_tokens", AgentToken.Type),
		edge.To("agent_trace_events", AgentTraceEvent.Type),
		edge.To("agent_step_events", AgentStepEvent.Type),
		edge.To("daily_token_usage", ProjectDailyTokenUsage.Type),
		edge.To("scheduled_jobs", ScheduledJob.Type),
		edge.To("activity_events", ActivityEvent.Type),
		edge.To("update_threads", ProjectUpdateThread.Type),
		edge.To("chat_conversations", ChatConversation.Type),
		edge.To("notification_rules", NotificationRule.Type),
		edge.To("default_agent_provider", AgentProvider.Type).
			Field("default_agent_provider_id").
			Unique(),
	}
}

// Indexes returns the Project schema indexes.
func (Project) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organization_id", "slug").Unique(),
	}
}
