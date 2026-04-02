package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ChatConversation stores a persistent direct-chat transcript for project conversations.
type ChatConversation struct {
	ent.Schema
}

// Fields returns the ChatConversation schema fields.
func (ChatConversation) Fields() []ent.Field {
	return []ent.Field{
		uuidField(),
		field.UUID("project_id", uuidZero()),
		field.String("user_id").NotEmpty(),
		field.String("source").NotEmpty(),
		field.UUID("provider_id", uuidZero()),
		field.String("status").Default("active"),
		field.String("provider_thread_id").Optional().Nillable(),
		field.String("last_turn_id").Optional().Nillable(),
		field.String("provider_thread_status").Optional().Nillable(),
		field.JSON("provider_thread_active_flags", []string{}).Optional(),
		field.Text("rolling_summary").Optional(),
		field.Time("last_activity_at").Default(time.Now).UpdateDefault(time.Now),
		createdAtField(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Edges returns the ChatConversation schema edges.
func (ChatConversation) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("project", Project.Type).
			Ref("chat_conversations").
			Field("project_id").
			Unique().
			Required(),
		edge.To("turns", ChatTurn.Type),
		edge.To("entries", ChatEntry.Type),
		edge.To("pending_interrupts", ChatPendingInterrupt.Type),
		edge.To("agent_tokens", AgentToken.Type),
	}
}

// Indexes returns the ChatConversation schema indexes.
func (ChatConversation) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("project_id", "user_id", "source", "provider_id", "last_activity_at"),
	}
}
