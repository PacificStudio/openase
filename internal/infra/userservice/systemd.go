package userservice

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/BetterAndBetterII/openase/internal/logging"
	"github.com/BetterAndBetterII/openase/internal/provider"
)

var _ = logging.DeclareComponent("userservice-systemd")

type SystemdUserManager struct {
	homeDir string
	runner  commandRunner
}

func NewSystemdUserManager(homeDir string) *SystemdUserManager {
	return &SystemdUserManager{
		homeDir: homeDir,
		runner:  execCommandRunner{},
	}
}

func newSystemdUserManagerForTest(homeDir string, runner commandRunner) *SystemdUserManager {
	return &SystemdUserManager{
		homeDir: homeDir,
		runner:  runner,
	}
}

func (m *SystemdUserManager) Platform() string {
	return "systemd --user"
}

func (m *SystemdUserManager) Apply(ctx context.Context, spec provider.UserServiceInstallSpec) error {
	if err := ensureServiceRuntimePaths(spec); err != nil {
		return err
	}

	unitPath := m.unitPath(spec.Name)
	if err := writeFile(unitPath, []byte(buildSystemdUnit(spec)), 0o644); err != nil {
		return fmt.Errorf("write systemd unit: %w", err)
	}
	if err := m.run(ctx, "--user", "daemon-reload"); err != nil {
		return err
	}
	if err := m.run(ctx, "--user", "enable", spec.Name.String()); err != nil {
		return err
	}
	if err := m.run(ctx, "--user", "start", spec.Name.String()); err != nil {
		return err
	}

	return nil
}

func (m *SystemdUserManager) Down(ctx context.Context, name provider.ServiceName) error {
	return m.run(ctx, "--user", "stop", name.String())
}

func (m *SystemdUserManager) Restart(ctx context.Context, name provider.ServiceName) error {
	return m.run(ctx, "--user", "restart", name.String())
}

func (m *SystemdUserManager) Logs(ctx context.Context, name provider.ServiceName, opts provider.UserServiceLogsOptions) error {
	args := []string{"--user", "-u", name.String(), "-n", strconv.Itoa(opts.Lines)}
	if opts.Follow {
		args = append(args, "-f")
	}

	return m.runner.Run(ctx, "journalctl", args, opts.Stdout, opts.Stderr)
}

func (m *SystemdUserManager) unitPath(name provider.ServiceName) string {
	return filepath.Join(m.homeDir, ".config", "systemd", "user", name.String()+".service")
}

func (m *SystemdUserManager) run(ctx context.Context, args ...string) error {
	if err := m.runner.Run(ctx, "systemctl", args, nil, nil); err != nil {
		return fmt.Errorf("systemctl %s: %w", strings.Join(args, " "), err)
	}

	return nil
}

func buildSystemdUnit(spec provider.UserServiceInstallSpec) string {
	var builder strings.Builder
	builder.WriteString("[Unit]\n")
	builder.WriteString("Description=" + spec.Description + "\n")
	builder.WriteString("After=network.target\n\n")
	builder.WriteString("[Service]\n")
	builder.WriteString("Type=simple\n")
	builder.WriteString("ExecStart=" + buildSystemdExecStart(spec.ProgramPath.String(), spec.Arguments) + "\n")
	builder.WriteString("EnvironmentFile=-" + spec.EnvironmentFile.String() + "\n")
	builder.WriteString("WorkingDirectory=" + spec.WorkingDirectory.String() + "\n")
	builder.WriteString("Restart=on-failure\n")
	builder.WriteString("RestartSec=3\n")
	builder.WriteString("StandardOutput=journal\n")
	builder.WriteString("StandardError=journal\n\n")
	builder.WriteString("[Install]\n")
	builder.WriteString("WantedBy=default.target\n")

	return builder.String()
}

func buildSystemdExecStart(program string, args []string) string {
	parts := make([]string, 0, 1+len(args))
	parts = append(parts, strconv.Quote(program))
	for _, arg := range args {
		parts = append(parts, strconv.Quote(arg))
	}

	return strings.Join(parts, " ")
}

func ensureServiceRuntimePaths(spec provider.UserServiceInstallSpec) error {
	directories := []string{
		spec.WorkingDirectory.String(),
		filepath.Dir(spec.StdoutPath.String()),
		filepath.Dir(spec.StderrPath.String()),
	}
	for _, directory := range directories {
		if err := os.MkdirAll(directory, 0o750); err != nil {
			return fmt.Errorf("create directory %q: %w", directory, err)
		}
	}
	for _, path := range []string{spec.StdoutPath.String(), spec.StderrPath.String()} {
		if err := ensureFile(path, 0o644); err != nil {
			return err
		}
	}

	return nil
}

func ensureFile(path string, mode os.FileMode) error {
	//nolint:gosec // service paths are derived from validated install specs
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, mode)
	if err != nil {
		return fmt.Errorf("ensure file %q: %w", path, err)
	}

	return file.Close()
}

func writeFile(path string, content []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("create parent directory for %q: %w", path, err)
	}

	//nolint:gosec // service unit paths are derived from validated install specs
	existing, err := os.ReadFile(path)
	if err == nil && bytes.Equal(existing, content) {
		return nil
	}
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read existing file %q: %w", path, err)
	}
	if err := os.WriteFile(path, content, mode); err != nil {
		return fmt.Errorf("write file %q: %w", path, err)
	}

	return nil
}
