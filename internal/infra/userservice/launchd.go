package userservice

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/BetterAndBetterII/openase/internal/logging"
	"github.com/BetterAndBetterII/openase/internal/provider"
)

var _ = logging.DeclareComponent("userservice-launchd")

type LaunchdUserManager struct {
	homeDir string
	uid     int
	runner  commandRunner
}

func NewLaunchdUserManager(homeDir string, uid int) *LaunchdUserManager {
	return &LaunchdUserManager{
		homeDir: homeDir,
		uid:     uid,
		runner:  execCommandRunner{},
	}
}

func newLaunchdUserManagerForTest(homeDir string, runner commandRunner) *LaunchdUserManager {
	return &LaunchdUserManager{
		homeDir: homeDir,
		uid:     501,
		runner:  runner,
	}
}

func (m *LaunchdUserManager) Platform() string {
	return "launchd"
}

func (m *LaunchdUserManager) Apply(ctx context.Context, spec provider.UserServiceInstallSpec) error {
	if err := ensureServiceRuntimePaths(spec); err != nil {
		return err
	}

	plistPath := m.plistPath(spec.Name)
	if err := writeFile(plistPath, []byte(buildLaunchdPlist(m.label(spec.Name), spec)), 0o644); err != nil {
		return fmt.Errorf("write launchd plist: %w", err)
	}

	loaded, err := m.isLoaded(ctx, spec.Name)
	if err != nil {
		return err
	}
	if loaded {
		if err := m.run(ctx, "bootout", m.target(spec.Name)); err != nil {
			return err
		}
	}
	if err := m.run(ctx, "bootstrap", m.domain(), plistPath); err != nil {
		return err
	}
	if err := m.run(ctx, "enable", m.target(spec.Name)); err != nil {
		return err
	}
	if err := m.run(ctx, "kickstart", "-k", m.target(spec.Name)); err != nil {
		return err
	}

	return nil
}

func (m *LaunchdUserManager) Down(ctx context.Context, name provider.ServiceName) error {
	loaded, err := m.isLoaded(ctx, name)
	if err != nil {
		return err
	}
	if !loaded {
		return nil
	}

	return m.run(ctx, "bootout", m.target(name))
}

func (m *LaunchdUserManager) Restart(ctx context.Context, name provider.ServiceName) error {
	loaded, err := m.isLoaded(ctx, name)
	if err != nil {
		return err
	}
	if !loaded {
		if err := m.run(ctx, "bootstrap", m.domain(), m.plistPath(name)); err != nil {
			return err
		}
		if err := m.run(ctx, "enable", m.target(name)); err != nil {
			return err
		}
	}

	return m.run(ctx, "kickstart", "-k", m.target(name))
}

func (m *LaunchdUserManager) Logs(ctx context.Context, name provider.ServiceName, opts provider.UserServiceLogsOptions) error {
	args := []string{"-n", strconv.Itoa(opts.Lines)}
	if opts.Follow {
		args = append(args, "-f")
	}
	args = append(args, m.stdoutPath(name), m.stderrPath(name))

	return m.runner.Run(ctx, "tail", args, opts.Stdout, opts.Stderr)
}

func (m *LaunchdUserManager) isLoaded(ctx context.Context, name provider.ServiceName) (bool, error) {
	err := m.runner.Run(ctx, "launchctl", []string{"print", m.target(name)}, io.Discard, io.Discard)
	if err == nil {
		return true, nil
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return false, nil
	}

	return false, fmt.Errorf("launchctl print %s: %w", m.target(name), err)
}

func (m *LaunchdUserManager) plistPath(name provider.ServiceName) string {
	return filepath.Join(m.homeDir, "Library", "LaunchAgents", m.label(name)+".plist")
}

func (m *LaunchdUserManager) stdoutPath(name provider.ServiceName) string {
	return filepath.Join(m.homeDir, ".openase", "logs", name.String()+".stdout.log")
}

func (m *LaunchdUserManager) stderrPath(name provider.ServiceName) string {
	return filepath.Join(m.homeDir, ".openase", "logs", name.String()+".stderr.log")
}

func (m *LaunchdUserManager) label(name provider.ServiceName) string {
	return "com." + name.String()
}

func (m *LaunchdUserManager) domain() string {
	return fmt.Sprintf("gui/%d", m.uid)
}

func (m *LaunchdUserManager) target(name provider.ServiceName) string {
	return m.domain() + "/" + m.label(name)
}

func (m *LaunchdUserManager) run(ctx context.Context, args ...string) error {
	if err := m.runner.Run(ctx, "launchctl", args, nil, nil); err != nil {
		return fmt.Errorf("launchctl %s: %w", strings.Join(args, " "), err)
	}

	return nil
}

func buildLaunchdPlist(label string, spec provider.UserServiceInstallSpec) string {
	var builder strings.Builder
	builder.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	builder.WriteString("<!DOCTYPE plist PUBLIC \"-//Apple//DTD PLIST 1.0//EN\" \"http://www.apple.com/DTDs/PropertyList-1.0.dtd\">\n")
	builder.WriteString("<plist version=\"1.0\">\n")
	builder.WriteString("<dict>\n")
	builder.WriteString(plistKeyValue("Label", label))
	builder.WriteString("  <key>ProgramArguments</key>\n")
	builder.WriteString("  <array>\n")
	builder.WriteString(plistString("/bin/sh"))
	builder.WriteString(plistString("-lc"))
	builder.WriteString(plistString(buildLaunchdShellCommand(spec)))
	builder.WriteString("  </array>\n")
	builder.WriteString(plistKeyValue("WorkingDirectory", spec.WorkingDirectory.String()))
	builder.WriteString(plistKeyValue("StandardOutPath", spec.StdoutPath.String()))
	builder.WriteString(plistKeyValue("StandardErrorPath", spec.StderrPath.String()))
	builder.WriteString("  <key>RunAtLoad</key>\n")
	builder.WriteString("  <true/>\n")
	builder.WriteString("  <key>KeepAlive</key>\n")
	builder.WriteString("  <true/>\n")
	builder.WriteString("</dict>\n")
	builder.WriteString("</plist>\n")

	return builder.String()
}

func plistKeyValue(key string, value string) string {
	return "  <key>" + xmlEscape(key) + "</key>\n" + plistString(value)
}

func plistString(value string) string {
	return "    <string>" + xmlEscape(value) + "</string>\n"
}

func buildLaunchdShellCommand(spec provider.UserServiceInstallSpec) string {
	parts := make([]string, 0, 3+len(spec.Arguments))
	parts = append(parts,
		". "+shellQuote(spec.EnvironmentFile.String())+" 2>/dev/null || true;",
		"exec",
		shellQuote(spec.ProgramPath.String()),
	)
	for _, arg := range spec.Arguments {
		parts = append(parts, shellQuote(arg))
	}

	return strings.Join(parts, " ")
}

func xmlEscape(value string) string {
	var buffer bytes.Buffer
	if err := xml.EscapeText(&buffer, []byte(value)); err != nil {
		panic(err)
	}

	return buffer.String()
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
}
