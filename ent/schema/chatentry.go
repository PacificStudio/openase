package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ChatEntry stores one append-only transcript item for a project conversation.
type ChatEntry struct {
	ent.Schema
}

// Fields returns the ChatEntry schema fields.
func (ChatEntry) Fields() []ent.Field {
	return []ent.Field{
		uuidField(),
		field.UUID("conversation_id", uuidZero()),
		field.UUID("turn_id", uuidZero()).
			Optional().
			Nillable(),
		field.Int("seq").NonNegative(),
		field.String("kind").NotEmpty(),
		field.JSON("payload_json", map[string]any{}).Default(emptyMap),
		createdAtField(),
	}
}

// Edges returns the ChatEntry schema edges.
func (ChatEntry) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("conversation", ChatConversation.Type).
			Ref("entries").
			Field("conversation_id").
			Unique().
			Required(),
		edge.From("turn", ChatTurn.Type).
			Ref("entries").
			Field("turn_id").
			Unique(),
	}
}

// Indexes returns the ChatEntry schema indexes.
func (ChatEntry) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("conversation_id", "seq").Unique(),
		index.Fields("turn_id", "seq"),
	}
}
