package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ChatPendingInterrupt stores one outstanding provider-native approval or input request.
type ChatPendingInterrupt struct {
	ent.Schema
}

// Fields returns the ChatPendingInterrupt schema fields.
func (ChatPendingInterrupt) Fields() []ent.Field {
	return []ent.Field{
		uuidField(),
		field.UUID("conversation_id", uuidZero()),
		field.UUID("turn_id", uuidZero()),
		field.String("provider_request_id").NotEmpty(),
		field.String("kind").NotEmpty(),
		field.JSON("payload_json", map[string]any{}).Default(emptyMap),
		field.String("status").Default("pending"),
		field.String("decision").Optional().Nillable(),
		field.JSON("response_json", map[string]any{}).Optional(),
		field.Time("resolved_at").Optional().Nillable(),
		createdAtField(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Edges returns the ChatPendingInterrupt schema edges.
func (ChatPendingInterrupt) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("conversation", ChatConversation.Type).
			Ref("pending_interrupts").
			Field("conversation_id").
			Unique().
			Required(),
		edge.From("turn", ChatTurn.Type).
			Ref("pending_interrupts").
			Field("turn_id").
			Unique().
			Required(),
	}
}

// Indexes returns the ChatPendingInterrupt schema indexes.
func (ChatPendingInterrupt) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("conversation_id", "status", "created_at"),
		index.Fields("conversation_id", "provider_request_id").Unique(),
	}
}
