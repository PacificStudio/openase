package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/BetterAndBetterII/openase/ent"
	entmigrate "github.com/BetterAndBetterII/openase/ent/migrate"
	// Register ent runtime hooks for generated schema metadata.
	_ "github.com/BetterAndBetterII/openase/ent/runtime"
	_ "github.com/lib/pq"
)

// Open connects to PostgreSQL and applies the current schema migration set.
func Open(ctx context.Context, dsn string) (*ent.Client, error) {
	trimmedDSN := strings.TrimSpace(dsn)
	if trimmedDSN == "" {
		return nil, errors.New("database.dsn is required for serve and all-in-one modes")
	}

	client, err := ent.Open("postgres", trimmedDSN)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := client.Schema.Create(
		ctx,
		entmigrate.WithDropColumn(false),
		entmigrate.WithDropIndex(false),
	); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("migrate database schema: %w", err)
	}
	if err := reconcileLegacyTicketIdentifierIndex(ctx, trimmedDSN); err != nil {
		_ = client.Close()
		return nil, err
	}

	return client, nil
}

func reconcileLegacyTicketIdentifierIndex(ctx context.Context, dsn string) error {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("open database for ticket index reconciliation: %w", err)
	}
	defer func() {
		_ = db.Close()
	}()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping database for ticket index reconciliation: %w", err)
	}

	if _, err := db.ExecContext(ctx, `DROP INDEX IF EXISTS "ticket_identifier"`); err != nil {
		return fmt.Errorf("drop legacy ticket identifier index: %w", err)
	}
	if _, err := db.ExecContext(ctx, `CREATE UNIQUE INDEX IF NOT EXISTS "ticket_project_id_identifier" ON "tickets" ("project_id", "identifier")`); err != nil {
		return fmt.Errorf("create project-scoped ticket identifier index: %w", err)
	}

	return nil
}
