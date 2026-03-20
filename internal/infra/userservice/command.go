package userservice

import (
	"context"
	"io"
	"os/exec"
)

type commandRunner interface {
	Run(ctx context.Context, name string, args []string, stdout io.Writer, stderr io.Writer) error
}

type execCommandRunner struct{}

func (execCommandRunner) Run(ctx context.Context, name string, args []string, stdout io.Writer, stderr io.Writer) error {
	//nolint:gosec // service manager shells out to fixed platform tooling with controlled arguments
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	return cmd.Run()
}
