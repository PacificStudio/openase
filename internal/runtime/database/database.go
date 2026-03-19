package database

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/BetterAndBetterII/openase/ent"
	entmigrate "github.com/BetterAndBetterII/openase/ent/migrate"
	_ "github.com/BetterAndBetterII/openase/ent/runtime"
	_ "github.com/lib/pq"
)

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

	return client, nil
}
