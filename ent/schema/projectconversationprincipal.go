package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ProjectConversationPrincipal defines the runtime principal for one persistent project conversation.
type ProjectConversationPrincipal struct {
	ent.Schema
}

func (ProjectConversationPrincipal) Fields() []ent.Field {
	return []ent.Field{
		uuidField(),
		field.UUID("conversation_id", uuidZero()).Unique(),
		field.UUID("project_id", uuidZero()),
		field.UUID("provider_id", uuidZero()),
		field.String("name").NotEmpty(),
		field.Enum("status").Values("active", "closed").Default("active"),
		field.Enum("runtime_state").Values("inactive", "ready", "executing", "interrupted").Default("inactive"),
		field.String("current_session_id").Optional().Nillable(),
		field.String("current_workspace_path").Optional().Nillable(),
		field.UUID("current_run_id", uuidZero()).Optional().Nillable(),
		field.Time("last_heartbeat_at").Optional().Nillable(),
		field.String("current_step_status").Optional().Nillable(),
		field.Text("current_step_summary").Optional().Nillable(),
		field.Time("current_step_changed_at").Optional().Nillable(),
		createdAtField(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
		field.Time("closed_at").Optional().Nillable(),
	}
}

func (ProjectConversationPrincipal) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("project_id", "status"),
		index.Fields("project_id", "runtime_state"),
		index.Fields("current_run_id"),
	}
}
