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
		field.String("name").NotEmpty(),
		field.Enum("adapter_type").
			Values("claude-code-cli", "codex-app-server", "gemini-cli", "custom"),
		field.String("cli_command").NotEmpty(),
		field.Strings("cli_args").
			SchemaType(textArrayColumn()).
			Optional(),
		field.JSON("auth_config", map[string]any{}).Default(emptyMap),
		field.String("model_name").NotEmpty(),
		field.Float("model_temperature").Default(0),
		field.Int("model_max_tokens").Default(16384),
		field.Float("cost_per_input_token").
			SchemaType(rateColumn()).
			Default(0),
		field.Float("cost_per_output_token").
			SchemaType(rateColumn()).
			Default(0),
	}
}

func (AgentProvider) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).
			Ref("providers").
			Field("organization_id").
			Unique().
			Required(),
		edge.To("agents", Agent.Type),
	}
}

func (AgentProvider) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organization_id", "name").Unique(),
	}
}
