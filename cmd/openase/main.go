package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/BetterAndBetterII/openase/internal/cli"
)

var version = "dev"

func main() {
	if err := cli.NewRootCommand(version).ExecuteContext(context.Background()); err != nil {
		slog.Error("openase command failed", "error", err)
		os.Exit(1)
	}
}
