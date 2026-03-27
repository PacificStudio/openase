package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/BetterAndBetterII/openase/internal/cli"
)

var version = "dev"

func main() {
	run(
		func(ctx context.Context) error {
			return cli.NewRootCommand(version).ExecuteContext(ctx)
		},
		os.Stderr,
		func(err error) {
			slog.Error("openase command failed", "error", err)
		},
		os.Exit,
	)
}

func run(execute func(context.Context) error, stderr io.Writer, logError func(error), exit func(int)) {
	if err := execute(context.Background()); err != nil {
		var exitErr interface{ ExitCode() int }
		if errors.As(err, &exitErr) {
			message := strings.TrimSpace(err.Error())
			if message != "" {
				_, _ = fmt.Fprintln(stderr, message)
			}
			exit(exitErr.ExitCode())
			return
		}
		logError(err)
		exit(1)
	}
}
