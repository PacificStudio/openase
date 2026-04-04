package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// OrganizationDailyTokenUsage defines persisted UTC-day token aggregates per organization.
type OrganizationDailyTokenUsage struct {
	ent.Schema
}

func (OrganizationDailyTokenUsage) Fields() []ent.Field {
	return []ent.Field{
		uuidField(),
		field.UUID("organization_id", uuidZero()),
		field.Time("usage_date").
			SchemaType(map[string]string{
				dialect.Postgres: "date",
			}),
		field.Int64("input_tokens").Default(0),
		field.Int64("output_tokens").Default(0),
		field.Int64("cached_input_tokens").Default(0),
		field.Int64("reasoning_tokens").Default(0),
		field.Int64("total_tokens").Default(0),
		field.Int("finalized_run_count").Default(0),
		field.Time("recomputed_at"),
		field.Enum("source_mode").
			Values("materialized", "lazy_backfill").
			Default("materialized"),
	}
}

func (OrganizationDailyTokenUsage) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("organization", Organization.Type).
			Ref("daily_token_usage").
			Field("organization_id").
			Unique().
			Required(),
	}
}

func (OrganizationDailyTokenUsage) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("organization_id", "usage_date").Unique(),
		index.Fields("usage_date"),
	}
}
