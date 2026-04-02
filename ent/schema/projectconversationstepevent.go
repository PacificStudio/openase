package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ProjectConversationStepEvent defines human-readable runtime step changes for project conversations.
type ProjectConversationStepEvent struct {
	ent.Schema
}

func (ProjectConversationStepEvent) Fields() []ent.Field {
	return []ent.Field{
		uuidField(),
		field.UUID("principal_id", uuidZero()),
		field.UUID("run_id", uuidZero()),
		field.UUID("conversation_id", uuidZero()),
		field.UUID("project_id", uuidZero()),
		field.String("step_status").NotEmpty(),
		field.Text("summary").Optional().Nillable(),
		field.UUID("source_trace_event_id", uuidZero()).Optional().Nillable(),
		createdAtField(),
	}
}

func (ProjectConversationStepEvent) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("run_id", "created_at"),
		index.Fields("conversation_id", "created_at"),
		index.Fields("principal_id", "created_at"),
	}
}
