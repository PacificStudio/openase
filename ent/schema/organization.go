package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	githubauth "github.com/BetterAndBetterII/openase/internal/domain/githubauth"
)

// Organization defines the ent schema for organizations.
type Organization struct {
	ent.Schema
}

// Fields returns the Organization schema fields.
func (Organization) Fields() []ent.Field {
	return []ent.Field{
		uuidField(),
		field.String("name").NotEmpty(),
		field.String("slug").NotEmpty(),
		field.Enum("status").
			Values("active", "archived").
			Default("active"),
		field.JSON("github_outbound_credential", &githubauth.StoredCredential{}).
			Optional(),
		field.JSON("github_token_probe", &githubauth.TokenProbe{}).
			Optional(),
		field.UUID("default_agent_provider_id", uuidZero()).
			Optional().
			Nillable(),
	}
}

// Edges returns the Organization schema edges.
func (Organization) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("projects", Project.Type),
		edge.To("providers", AgentProvider.Type),
		edge.To("machines", Machine.Type),
		edge.To("notification_channels", NotificationChannel.Type),
		edge.To("daily_token_usage", OrganizationDailyTokenUsage.Type),
		edge.To("default_agent_provider", AgentProvider.Type).
			Field("default_agent_provider_id").
			Unique(),
	}
}

// Indexes returns the Organization schema indexes.
func (Organization) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("slug").Unique(),
	}
}
