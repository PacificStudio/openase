package setup

import (
	"github.com/BetterAndBetterII/openase/ent"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func openEntClient(dsn string) (*ent.Client, error) {
	return ent.Open("pgx", dsn)
}
