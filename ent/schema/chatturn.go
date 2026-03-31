package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ChatTurn stores one user-originated turn inside a project conversation.
type ChatTurn struct {
	ent.Schema
}

// Fields returns the ChatTurn schema fields.
func (ChatTurn) Fields() []ent.Field {
	return []ent.Field{
		uuidField(),
		field.UUID("conversation_id", uuidZero()),
		field.Int("turn_index").Positive(),
		field.String("provider_turn_id").Optional().Nillable(),
		field.String("status").Default("pending"),
		field.Time("started_at").Default(time.Now),
		field.Time("completed_at").Optional().Nillable(),
		createdAtField(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Edges returns the ChatTurn schema edges.
func (ChatTurn) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("conversation", ChatConversation.Type).
			Ref("turns").
			Field("conversation_id").
			Unique().
			Required(),
		edge.To("entries", ChatEntry.Type),
		edge.To("pending_interrupts", ChatPendingInterrupt.Type),
	}
}

// Indexes returns the ChatTurn schema indexes.
func (ChatTurn) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("conversation_id", "turn_index").Unique(),
		index.Fields("conversation_id", "started_at"),
	}
}
