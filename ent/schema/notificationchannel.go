package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// NotificationChannel defines the ent schema for notification delivery channels.
type NotificationChannel struct {
	ent.Schema
}

// Fields returns the NotificationChannel schema fields.
func (NotificationChannel) Fields() []ent.Field {
	return []ent.Field{
		uuidField(),
		field.UUID("organization_id", uuidZero()),
		field.String("name").NotEmpty(),
		field.String("type").NotEmpty(),
		field.JSON("config", map[string]any{}).Default(emptyMap),
		field.Bool("is_enabled").Default(true),
		createdAtField(),
	}
}

// Edges returns the NotificationChannel schema edges.
func (NotificationChannel) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).
			Ref("notification_channels").
			Field("organization_id").
			Unique().
			Required(),
		edge.To("notification_rules", NotificationRule.Type),
	}
}

// Indexes returns the NotificationChannel schema indexes.
func (NotificationChannel) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organization_id", "name").Unique(),
		index.Fields("organization_id", "type"),
	}
}
