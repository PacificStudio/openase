package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type ApprovalGate struct {
	ent.Schema
}

func (ApprovalGate) Fields() []ent.Field {
	return []ent.Field{
		uuidField(),
		field.UUID("ticket_id", uuidZero()),
		field.String("trigger_status").NotEmpty(),
		field.Enum("status").
			Values("pending", "approved", "rejected").
			Default("pending"),
		field.String("reviewer").Optional(),
		field.Text("comment").Optional(),
		field.JSON("hook_results", map[string]any{}).Default(emptyMap),
		createdAtField(),
		field.Time("resolved_at").Optional().Nillable(),
	}
}

func (ApprovalGate) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("ticket", Ticket.Type).
			Ref("approval_gates").
			Field("ticket_id").
			Unique().
			Required(),
	}
}

func (ApprovalGate) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("status").
			Annotations(entsql.IndexWhere("status = 'pending'")),
		index.Fields("ticket_id", "status"),
	}
}
