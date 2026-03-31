package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// TicketRepoWorkspace defines the runtime state for one repo inside one ticket execution workspace.
type TicketRepoWorkspace struct {
	ent.Schema
}

func (TicketRepoWorkspace) Fields() []ent.Field {
	return []ent.Field{
		uuidField(),
		field.UUID("ticket_id", uuidZero()),
		field.UUID("agent_run_id", uuidZero()),
		field.UUID("repo_id", uuidZero()),
		field.String("workspace_root").NotEmpty(),
		field.String("repo_path").NotEmpty(),
		field.String("branch_name").NotEmpty(),
		field.Enum("state").
			Values("planned", "materializing", "ready", "dirty", "verifying", "completed", "failed", "cleaning", "cleaned").
			Default("planned"),
		field.String("head_commit").Optional(),
		field.Text("last_error").Optional(),
		field.Time("prepared_at").Optional().Nillable(),
		field.Time("cleaned_at").Optional().Nillable(),
		createdAtField(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (TicketRepoWorkspace) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("ticket", Ticket.Type).
			Ref("repo_workspaces").
			Field("ticket_id").
			Unique().
			Required(),
		edge.From("agent_run", AgentRun.Type).
			Ref("ticket_repo_workspaces").
			Field("agent_run_id").
			Unique().
			Required(),
		edge.From("repo", ProjectRepo.Type).
			Ref("ticket_repo_workspaces").
			Field("repo_id").
			Unique().
			Required(),
	}
}

func (TicketRepoWorkspace) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("agent_run_id", "repo_id").Unique(),
		index.Fields("ticket_id", "state"),
	}
}
