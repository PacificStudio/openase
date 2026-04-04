package catalog

import (
	"os"
	"testing"

	"github.com/BetterAndBetterII/openase/internal/testutil/pgtest"
)

var testPostgres *pgtest.Server

func TestMain(m *testing.M) {
	os.Exit(pgtest.RunTestMain(m, "repo_catalog", func(server *pgtest.Server) {
		testPostgres = server
	}))
}
