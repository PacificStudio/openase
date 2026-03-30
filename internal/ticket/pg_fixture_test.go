package ticket

import (
	"os"
	"testing"

	"github.com/BetterAndBetterII/openase/internal/testutil/pgtest"
)

var testPostgres *pgtest.Server

func TestMain(m *testing.M) {
	os.Exit(pgtest.RunTestMain(m, "ticket", func(server *pgtest.Server) {
		testPostgres = server
	}))
}
