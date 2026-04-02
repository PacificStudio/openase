package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ProjectConversationTraceEvent defines fine-grained runtime output for project conversation execution.
type ProjectConversationTraceEvent struct {
	ent.Schema
}

func (ProjectConversationTraceEvent) Fields() []ent.Field {
	return []ent.Field{
		uuidField(),
		field.UUID("principal_id", uuidZero()),
		field.UUID("run_id", uuidZero()),
		field.UUID("conversation_id", uuidZero()),
		field.UUID("project_id", uuidZero()),
		field.Int64("sequence"),
		field.String("provider").NotEmpty(),
		field.String("kind").NotEmpty(),
		field.String("stream").NotEmpty(),
		field.Text("text").Optional().Nillable(),
		field.JSON("payload", map[string]any{}).Default(emptyMap),
		createdAtField(),
	}
}

func (ProjectConversationTraceEvent) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("run_id", "sequence").Unique(),
		index.Fields("conversation_id", "created_at"),
		index.Fields("principal_id", "created_at"),
	}
}
