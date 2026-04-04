package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Machine defines the ent schema for organization-scoped execution machines.
type Machine struct {
	ent.Schema
}

// Fields returns the Machine schema fields.
func (Machine) Fields() []ent.Field {
	return []ent.Field{
		uuidField(),
		field.UUID("organization_id", uuidZero()),
		field.String("name").NotEmpty(),
		field.String("host").NotEmpty(),
		field.Int("port").Default(22),
		field.Enum("connection_mode").
			Values("local", "ssh", "ws_reverse", "ws_listener").
			Default("ssh"),
		textArrayField("transport_capabilities"),
		field.String("ssh_user").Optional(),
		field.String("ssh_key_path").Optional(),
		field.String("advertised_endpoint").Optional(),
		field.Bool("daemon_registered").Default(false),
		field.Time("daemon_last_registered_at").Optional().Nillable(),
		field.String("daemon_session_id").Optional(),
		field.Enum("daemon_session_state").
			Values("unknown", "connected", "disconnected", "unavailable").
			Default("unknown"),
		field.Enum("detected_os").
			Values("darwin", "linux", "unknown").
			Default("unknown"),
		field.Enum("detected_arch").
			Values("amd64", "arm64", "unknown").
			Default("unknown"),
		field.Enum("detection_status").
			Values("pending", "ok", "degraded", "unknown").
			Default("unknown"),
		field.Enum("channel_credential_kind").
			Values("none", "token", "certificate").
			Default("none"),
		field.String("channel_token_id").Optional(),
		field.String("channel_certificate_id").Optional(),
		field.Text("description").Optional(),
		textArrayField("labels"),
		field.Enum("status").
			Values("online", "offline", "degraded", "maintenance").
			Default("maintenance"),
		field.String("workspace_root").Optional(),
		field.String("agent_cli_path").Optional(),
		textArrayField("env_vars"),
		field.Time("last_heartbeat_at").Optional().Nillable(),
		field.JSON("resources", map[string]any{}).Default(emptyMap),
	}
}

// Edges returns the Machine schema edges.
func (Machine) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).
			Ref("machines").
			Field("organization_id").
			Unique().
			Required(),
		edge.To("channel_tokens", MachineChannelToken.Type),
		edge.To("providers", AgentProvider.Type),
		edge.To("target_tickets", Ticket.Type),
	}
}

// Indexes returns the Machine schema indexes.
func (Machine) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organization_id", "name").Unique(),
		index.Fields("organization_id", "host"),
		index.Fields("labels").
			Annotations(entsql.IndexType("GIN")),
	}
}
