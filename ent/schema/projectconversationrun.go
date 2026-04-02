package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ProjectConversationRun defines one observable runtime turn/run for a project conversation principal.
type ProjectConversationRun struct {
	ent.Schema
}

func (ProjectConversationRun) Fields() []ent.Field {
	return []ent.Field{
		uuidField(),
		field.UUID("principal_id", uuidZero()),
		field.UUID("conversation_id", uuidZero()),
		field.UUID("project_id", uuidZero()),
		field.UUID("provider_id", uuidZero()),
		field.UUID("turn_id", uuidZero()).Optional().Nillable(),
		field.Enum("status").Values("launching", "executing", "interrupted", "completed", "failed", "terminated"),
		field.String("session_id").Optional().Nillable(),
		field.String("workspace_path").Optional().Nillable(),
		field.String("provider_thread_id").Optional().Nillable(),
		field.String("provider_turn_id").Optional().Nillable(),
		field.Time("runtime_started_at").Optional().Nillable(),
		field.Time("terminal_at").Optional().Nillable(),
		field.String("last_error").Optional().Nillable(),
		field.Time("last_heartbeat_at").Optional().Nillable(),
		field.Float("cost_amount").Default(0),
		field.Int64("input_tokens").Default(0),
		field.Int64("output_tokens").Default(0),
		field.Int64("cached_input_tokens").Default(0),
		field.Int64("cache_creation_tokens").Default(0),
		field.Int64("reasoning_tokens").Default(0),
		field.Int64("prompt_tokens").Default(0),
		field.Int64("candidate_tokens").Default(0),
		field.Int64("tool_tokens").Default(0),
		field.Int64("total_tokens").Default(0),
		field.String("current_step_status").Optional().Nillable(),
		field.Text("current_step_summary").Optional().Nillable(),
		field.Time("current_step_changed_at").Optional().Nillable(),
		createdAtField(),
	}
}

func (ProjectConversationRun) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("principal_id", "created_at"),
		index.Fields("conversation_id", "created_at"),
		index.Fields("status", "last_heartbeat_at"),
		index.Fields("turn_id").Unique(),
	}
}
