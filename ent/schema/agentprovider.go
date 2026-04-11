package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type AgentProvider struct {
	ent.Schema
}

func (AgentProvider) Fields() []ent.Field {
	return []ent.Field{
		uuidField(),
		field.UUID("organization_id", uuidZero()),
		field.UUID("machine_id", uuidZero()),
		field.String("name").NotEmpty(),
		field.Enum("adapter_type").
			Values("claude-code-cli", "codex-app-server", "gemini-cli", "custom"),
		field.Enum("permission_profile").
			Values("standard", "unrestricted").
			Default("unrestricted"),
		field.String("cli_command").NotEmpty(),
		textArrayField("cli_args"),
		field.JSON("auth_config", map[string]any{}).Default(emptyMap),
		field.JSON("cli_rate_limit", map[string]any{}).Default(emptyMap),
		field.Time("cli_rate_limit_updated_at").Optional().Nillable(),
		field.String("model_name").NotEmpty(),
		field.String("reasoning_effort").Optional().Nillable(),
		field.Float("model_temperature").Default(0),
		field.Int("model_max_tokens").Default(16384),
		field.Int("max_parallel_runs").Default(0),
		field.Float("cost_per_input_token").
			SchemaType(rateColumn()).
			Default(0),
		field.Float("cost_per_output_token").
			SchemaType(rateColumn()).
			Default(0),
		field.JSON("pricing_config", map[string]any{}).Default(emptyMap),
	}
}

func (AgentProvider) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).
			Ref("providers").
			Field("organization_id").
			Unique().
			Required(),
		edge.From("machine", Machine.Type).
			Ref("providers").
			Field("machine_id").
			Unique().
			Required(),
		edge.To("agents", Agent.Type),
		edge.To("agent_runs", AgentRun.Type),
	}
}

func (AgentProvider) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organization_id", "name").Unique(),
	}
}
