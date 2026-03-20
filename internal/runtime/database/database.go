package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entmigrate "github.com/BetterAndBetterII/openase/ent/migrate"
	// Register ent runtime hooks for generated schema metadata.
	_ "github.com/BetterAndBetterII/openase/ent/runtime"
	_ "github.com/lib/pq"
)

const schemaBootstrapLockKey int64 = 0x6f70656e617365

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

	if err := withSchemaBootstrapLock(ctx, trimmedDSN, func() error {
		if err := reconcileLegacyProjectAccessibleMachineIDs(ctx, trimmedDSN); err != nil {
			return err
		}
		if err := client.Schema.Create(
			ctx,
			entmigrate.WithDropColumn(false),
			entmigrate.WithDropIndex(false),
		); err != nil {
			return fmt.Errorf("migrate database schema: %w", err)
		}
		if err := reconcileLegacyTicketIdentifierIndex(ctx, trimmedDSN); err != nil {
			return err
		}

		return nil
	}); err != nil {
		_ = client.Close()
		return nil, err
	}

	return client, nil
}

func withSchemaBootstrapLock(ctx context.Context, dsn string, fn func() error) (err error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("open database for schema bootstrap lock: %w", err)
	}
	defer func() {
		_ = db.Close()
	}()

	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("reserve database connection for schema bootstrap lock: %w", err)
	}
	defer func() {
		_ = conn.Close()
	}()

	if _, err := conn.ExecContext(ctx, `SELECT pg_advisory_lock($1)`, schemaBootstrapLockKey); err != nil {
		return fmt.Errorf("acquire schema bootstrap lock: %w", err)
	}
	defer func() {
		unlockCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, unlockErr := conn.ExecContext(unlockCtx, `SELECT pg_advisory_unlock($1)`, schemaBootstrapLockKey); unlockErr != nil {
			releaseErr := fmt.Errorf("release schema bootstrap lock: %w", unlockErr)
			if err == nil {
				err = releaseErr
				return
			}
			err = errors.Join(err, releaseErr)
		}
	}()

	return fn()
}

func reconcileLegacyProjectAccessibleMachineIDs(ctx context.Context, dsn string) error {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("open database for project accessible machine reconciliation: %w", err)
	}
	defer func() {
		_ = db.Close()
	}()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping database for project accessible machine reconciliation: %w", err)
	}

	var projectTableExists bool
	if err := db.QueryRowContext(
		ctx,
		`SELECT EXISTS (
			SELECT 1
			FROM information_schema.tables
			WHERE table_schema = current_schema()
			  AND table_name = 'projects'
		)`,
	).Scan(&projectTableExists); err != nil {
		return fmt.Errorf("check projects table: %w", err)
	}
	if !projectTableExists {
		return nil
	}

	var columnExists bool
	if err := db.QueryRowContext(
		ctx,
		`SELECT EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_schema = current_schema()
			  AND table_name = 'projects'
			  AND column_name = 'accessible_machine_ids'
		)`,
	).Scan(&columnExists); err != nil {
		return fmt.Errorf("check project accessible machine column: %w", err)
	}
	if !columnExists {
		if _, err := db.ExecContext(
			ctx,
			`ALTER TABLE "projects" ADD COLUMN "accessible_machine_ids" jsonb`,
		); err != nil {
			return fmt.Errorf("add project accessible machine ids column: %w", err)
		}
	}

	if _, err := db.ExecContext(
		ctx,
		`UPDATE "projects" SET "accessible_machine_ids" = '[]'::jsonb WHERE "accessible_machine_ids" IS NULL`,
	); err != nil {
		return fmt.Errorf("backfill project accessible machine ids: %w", err)
	}

	return nil
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
