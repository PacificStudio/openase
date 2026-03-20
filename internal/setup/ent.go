package setup

import (
	"github.com/BetterAndBetterII/openase/ent"
	// Register the pgx SQL driver for setup-time schema checks.
	_ "github.com/jackc/pgx/v5/stdlib"
)

func openEntClient(dsn string) (*ent.Client, error) {
	return ent.Open("pgx", dsn)
}
