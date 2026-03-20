package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// NotificationRule defines the ent schema for event-driven notification subscriptions.
type NotificationRule struct {
	ent.Schema
}

// Fields returns the NotificationRule schema fields.
func (NotificationRule) Fields() []ent.Field {
	return []ent.Field{
		uuidField(),
		field.UUID("project_id", uuidZero()),
		field.UUID("channel_id", uuidZero()),
		field.String("name").NotEmpty(),
		field.String("event_type").NotEmpty(),
		field.JSON("filter", map[string]any{}).Default(emptyMap),
		field.Text("template").Optional(),
		field.Bool("is_enabled").Default(true),
		createdAtField(),
	}
}

// Edges returns the NotificationRule schema edges.
func (NotificationRule) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("project", Project.Type).
			Ref("notification_rules").
			Field("project_id").
			Unique().
			Required(),
		edge.From("channel", NotificationChannel.Type).
			Ref("notification_rules").
			Field("channel_id").
			Unique().
			Required(),
	}
}

// Indexes returns the NotificationRule schema indexes.
func (NotificationRule) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("project_id", "name").Unique(),
		index.Fields("project_id", "event_type"),
		index.Fields("channel_id"),
	}
}
