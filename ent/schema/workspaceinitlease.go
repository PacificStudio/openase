package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// WorkspaceInitLease serializes ticket workspace materialization per machine.
type WorkspaceInitLease struct {
	ent.Schema
}

func (WorkspaceInitLease) Fields() []ent.Field {
	return []ent.Field{
		uuidField(),
		field.String("lease_key").NotEmpty(),
		field.UUID("machine_id", uuidZero()),
		field.UUID("owner_run_id", uuidZero()),
		field.Time("lease_expires_at"),
		field.Time("heartbeat_at"),
		createdAtField(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

func (WorkspaceInitLease) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("lease_key").Unique(),
		index.Fields("machine_id"),
		index.Fields("owner_run_id"),
		index.Fields("lease_expires_at"),
	}
}
