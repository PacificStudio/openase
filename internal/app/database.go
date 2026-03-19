package app

import (
	"github.com/BetterAndBetterII/openase/ent"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func openEntClient(dsn string) (*ent.Client, error) {
	if dsn == "" {
		return nil, nil
	}
	return ent.Open("pgx", dsn)
}
