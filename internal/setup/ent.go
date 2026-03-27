package setup

import (
	"github.com/BetterAndBetterII/openase/ent"
	// Register the postgres SQL driver for setup-time schema checks.
	_ "github.com/lib/pq"
)

func openEntClient(dsn string) (*ent.Client, error) {
	return ent.Open("postgres", dsn)
}
