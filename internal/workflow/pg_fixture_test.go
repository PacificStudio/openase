package workflow

import (
	"os"
	"testing"

	"github.com/BetterAndBetterII/openase/internal/testutil/pgtest"
)

var testPostgres *pgtest.Server

func TestMain(m *testing.M) {
	os.Exit(pgtest.RunTestMain(m, "workflow", func(server *pgtest.Server) {
		testPostgres = server
	}))
}
