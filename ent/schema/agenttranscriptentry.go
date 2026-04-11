package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// AgentTranscriptEntry defines low-noise product-facing transcript entries for a run.
type AgentTranscriptEntry struct {
	ent.Schema
}

func (AgentTranscriptEntry) Fields() []ent.Field {
	return []ent.Field{
		uuidField(),
		field.UUID("project_id", uuidZero()),
		field.UUID("ticket_id", uuidZero()).
			Optional().
			Nillable(),
		field.UUID("agent_id", uuidZero()),
		field.UUID("agent_run_id", uuidZero()),
		field.String("provider").NotEmpty(),
		field.String("entry_key").NotEmpty(),
		field.String("entry_kind").NotEmpty(),
		field.String("activity_kind").Optional().Nillable(),
		field.String("activity_id").Optional().Nillable(),
		field.String("title").Optional().Nillable(),
		field.Text("summary").Optional().Nillable(),
		field.Text("body_text").Optional().Nillable(),
		field.Text("command").Optional().Nillable(),
		field.String("tool_name").Optional().Nillable(),
		field.JSON("metadata", map[string]any{}).Default(emptyMap),
		field.Time("created_at"),
	}
}

func (AgentTranscriptEntry) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("project", Project.Type).
			Ref("agent_transcript_entries").
			Field("project_id").
			Unique().
			Required(),
		edge.From("ticket", Ticket.Type).
			Ref("agent_transcript_entries").
			Field("ticket_id").
			Unique(),
		edge.From("agent", Agent.Type).
			Ref("agent_transcript_entries").
			Field("agent_id").
			Unique().
			Required(),
		edge.From("agent_run", AgentRun.Type).
			Ref("agent_transcript_entries").
			Field("agent_run_id").
			Unique().
			Required(),
	}
}

func (AgentTranscriptEntry) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("agent_run_id", "entry_key").Unique(),
		index.Fields("agent_run_id", "created_at", "id"),
		index.Fields("project_id", "created_at"),
		index.Fields("ticket_id", "created_at"),
	}
}
