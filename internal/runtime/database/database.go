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
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
	// Register ent runtime hooks for generated schema metadata.
	_ "github.com/BetterAndBetterII/openase/ent/runtime"
	// Register the PostgreSQL SQL driver used by database/sql and ent.
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
	ticketservice.InstallRetryTokenHooks(client)

	if err := withSchemaBootstrapLock(ctx, trimmedDSN, func() error {
		if err := applyLegacySchemaCompat(ctx, trimmedDSN); err != nil {
			return err
		}
		if err := client.Schema.Create(
			ctx,
			entmigrate.WithDropColumn(false),
			entmigrate.WithDropIndex(false),
		); err != nil {
			return fmt.Errorf("migrate database schema: %w", err)
		}
		return nil
	}); err != nil {
		_ = client.Close()
		return nil, err
	}

	return client, nil
}

func applyLegacySchemaCompat(ctx context.Context, dsn string) error {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("open database for legacy schema compat: %w", err)
	}
	defer func() {
		_ = db.Close()
	}()

	var hasAgentProviders bool
	if err := db.QueryRowContext(
		ctx,
		`SELECT EXISTS (
			SELECT 1
			FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = 'agent_providers'
		)`,
	).Scan(&hasAgentProviders); err != nil {
		return fmt.Errorf("detect legacy agent provider schema: %w", err)
	}
	if !hasAgentProviders {
		return nil
	}

	statements := []string{
		`ALTER TABLE agent_providers ADD COLUMN IF NOT EXISTS cli_rate_limit jsonb`,
		`ALTER TABLE agent_providers ADD COLUMN IF NOT EXISTS cli_rate_limit_updated_at timestamptz`,
		`UPDATE agent_providers SET cli_rate_limit = '{}'::jsonb WHERE cli_rate_limit IS NULL`,
		`ALTER TABLE agent_providers ALTER COLUMN cli_rate_limit SET DEFAULT '{}'::jsonb`,
		`ALTER TABLE agent_providers ALTER COLUMN cli_rate_limit SET NOT NULL`,
		`DROP TABLE IF EXISTS issue_connectors`,
	}
	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("apply legacy agent provider schema compat: %w", err)
		}
	}

	return nil
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
