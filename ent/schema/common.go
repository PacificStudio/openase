package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/dialect"
	"entgo.io/ent/schema/field"
	"github.com/BetterAndBetterII/openase/internal/types/pgarray"
	"github.com/google/uuid"
)

func uuidField() ent.Field {
	return field.UUID("id", uuid.UUID{}).
		Default(uuid.New).
		Immutable()
}

func uuidZero() uuid.UUID {
	return uuid.UUID{}
}

func createdAtField() ent.Field {
	return field.Time("created_at").
		Default(time.Now).
		Immutable()
}

func emptyMap() map[string]any {
	return map[string]any{}
}

func emptyUUIDs() []uuid.UUID {
	return []uuid.UUID{}
}

func currencyColumn() map[string]string {
	return map[string]string{
		dialect.Postgres: "numeric(18,6)",
	}
}

func rateColumn() map[string]string {
	return map[string]string{
		dialect.Postgres: "numeric(18,8)",
	}
}

func textArrayColumn() map[string]string {
	return map[string]string{
		dialect.Postgres: "text[]",
	}
}

func textArrayField(name string) ent.Field {
	return field.Other(name, pgarray.StringArray{}).
		SchemaType(textArrayColumn()).
		Optional()
}
