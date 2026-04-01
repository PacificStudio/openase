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
