package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/BetterAndBetterII/openase/internal/cli"
)

var version = "dev"

func main() {
	if err := cli.NewRootCommand(version).ExecuteContext(context.Background()); err != nil {
		var exitErr interface{ ExitCode() int }
		if errors.As(err, &exitErr) {
			message := strings.TrimSpace(err.Error())
			if message != "" {
				fmt.Fprintln(os.Stderr, message)
			}
			os.Exit(exitErr.ExitCode())
		}
		slog.Error("openase command failed", "error", err)
		os.Exit(1)
	}
}
