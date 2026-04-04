package localdiag

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
)

type Status string

const (
	StatusReady        Status = "ready"
	StatusMissing      Status = "missing"
	StatusVersionError Status = "version_error"
)

type CommandSpec struct {
	ID          string
	Name        string
	Command     string
	VersionArgs []string
}

type CommandReport struct {
	ID      string
	Name    string
	Command string
	Status  Status
	Path    string
	Version string
	Error   string
}

type Options struct {
	LookPath   func(string) (string, error)
	RunCommand func(context.Context, string, ...string) (string, error)
}

func SetupCommandSpecs() []CommandSpec {
	templates := catalogdomain.BuiltinAgentProviderTemplates()
	specs := make([]CommandSpec, 0, len(templates)+1)
	specs = append(specs, CommandSpec{
		ID:      "git",
		Name:    "Git",
		Command: "git",
	})
	for _, template := range templates {
		specs = append(specs, CommandSpec{
			ID:      template.ID,
			Name:    template.Name,
			Command: template.Command,
		})
	}
	return specs
}

func Inspect(ctx context.Context, specs []CommandSpec, opts Options) []CommandReport {
	lookPath := opts.LookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}

	runCommand := opts.RunCommand
	if runCommand == nil {
		runCommand = runExecCommand
	}

	reports := make([]CommandReport, 0, len(specs))
	for _, spec := range specs {
		report := CommandReport{
			ID:      spec.ID,
			Name:    spec.Name,
			Command: spec.Command,
		}

		path, err := lookPath(spec.Command)
		if err != nil {
			report.Status = StatusMissing
			reports = append(reports, report)
			continue
		}

		versionArgs := spec.VersionArgs
		if len(versionArgs) == 0 {
			versionArgs = []string{"--version"}
		}

		output, err := runCommand(ctx, path, versionArgs...)
		if err != nil {
			report.Status = StatusVersionError
			report.Path = path
			report.Error = err.Error()
			reports = append(reports, report)
			continue
		}

		report.Status = StatusReady
		report.Path = path
		report.Version = firstNonEmptyLine(output)
		if report.Version == "" {
			report.Version = path
		}
		reports = append(reports, report)
	}

	return reports
}

func firstNonEmptyLine(raw string) string {
	for _, line := range strings.Split(raw, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func runExecCommand(ctx context.Context, name string, args ...string) (string, error) {
	//nolint:gosec // local environment diagnostics intentionally execute resolved local commands
	command := exec.CommandContext(ctx, name, args...)
	output, err := command.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}
	return string(output), nil
}
