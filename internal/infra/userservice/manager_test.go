package userservice

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/BetterAndBetterII/openase/internal/provider"
)

func TestSystemdApplyWritesUnitAndRunsLifecycleCommands(t *testing.T) {
	homeDir := t.TempDir()
	runner := &recordingRunner{}
	manager := newSystemdUserManagerForTest(homeDir, runner)
	spec := testInstallSpec(t, homeDir)

	if err := manager.Apply(context.Background(), spec); err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	unitPath := filepath.Join(homeDir, ".config", "systemd", "user", "openase.service")
	unitBytes, err := os.ReadFile(unitPath)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	unit := string(unitBytes)
	for _, expected := range []string{
		"Description=OpenASE -- Auto Software Engineering Platform",
		`ExecStart="/tmp/openase" "all-in-one" "--config" "/tmp/openase.yaml"`,
		"EnvironmentFile=-" + filepath.Join(homeDir, ".openase", ".env"),
		"WorkingDirectory=" + filepath.Join(homeDir, ".openase"),
		"WantedBy=default.target",
	} {
		if !strings.Contains(unit, expected) {
			t.Fatalf("expected unit to contain %q, got:\n%s", expected, unit)
		}
	}

	expectedCalls := []recordedCommand{
		{name: "systemctl", args: []string{"--user", "daemon-reload"}},
		{name: "systemctl", args: []string{"--user", "enable", "openase"}},
		{name: "systemctl", args: []string{"--user", "start", "openase"}},
	}
	if !reflect.DeepEqual(runner.calls, expectedCalls) {
		t.Fatalf("unexpected commands: %+v", runner.calls)
	}
}

func TestSystemdLogsUsesJournalctl(t *testing.T) {
	runner := &recordingRunner{}
	manager := newSystemdUserManagerForTest(t.TempDir(), runner)
	opts, err := provider.NewUserServiceLogsOptions(200, true, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("NewUserServiceLogsOptions returned error: %v", err)
	}

	if err := manager.Logs(context.Background(), provider.MustParseServiceName("openase"), opts); err != nil {
		t.Fatalf("Logs returned error: %v", err)
	}

	expected := recordedCommand{name: "journalctl", args: []string{"--user", "-u", "openase", "-n", "200", "-f"}}
	if len(runner.calls) != 1 || !reflect.DeepEqual(runner.calls[0], expected) {
		t.Fatalf("unexpected commands: %+v", runner.calls)
	}
}

func TestLaunchdApplyWritesPlistAndBootstrapsService(t *testing.T) {
	homeDir := t.TempDir()
	runner := &recordingRunner{
		results: []error{&exec.ExitError{}},
	}
	manager := newLaunchdUserManagerForTest(homeDir, 501, runner)
	spec := testInstallSpec(t, homeDir)

	if err := manager.Apply(context.Background(), spec); err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	plistPath := filepath.Join(homeDir, "Library", "LaunchAgents", "com.openase.plist")
	plistBytes, err := os.ReadFile(plistPath)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	plist := string(plistBytes)
	for _, expected := range []string{
		"<key>Label</key>",
		"<string>com.openase</string>",
		"<string>/bin/sh</string>",
		"<string>-lc</string>",
		"<string>" + xmlEscape(buildLaunchdShellCommand(spec)) + "</string>",
		"<string>" + xmlEscape(filepath.Join(homeDir, ".openase", "logs", "openase.stdout.log")) + "</string>",
	} {
		if !strings.Contains(plist, expected) {
			t.Fatalf("expected plist to contain %q, got:\n%s", expected, plist)
		}
	}

	expectedCalls := []recordedCommand{
		{name: "launchctl", args: []string{"print", "gui/501/com.openase"}},
		{name: "launchctl", args: []string{"bootstrap", "gui/501", plistPath}},
		{name: "launchctl", args: []string{"enable", "gui/501/com.openase"}},
		{name: "launchctl", args: []string{"kickstart", "-k", "gui/501/com.openase"}},
	}
	if !reflect.DeepEqual(runner.calls, expectedCalls) {
		t.Fatalf("unexpected commands: %+v", runner.calls)
	}
}

func TestLaunchdRestartBootstrapsWhenServiceIsNotLoaded(t *testing.T) {
	homeDir := t.TempDir()
	runner := &recordingRunner{
		results: []error{&exec.ExitError{}},
	}
	manager := newLaunchdUserManagerForTest(homeDir, 501, runner)

	if err := manager.Restart(context.Background(), provider.MustParseServiceName("openase")); err != nil {
		t.Fatalf("Restart returned error: %v", err)
	}

	expected := []recordedCommand{
		{name: "launchctl", args: []string{"print", "gui/501/com.openase"}},
		{name: "launchctl", args: []string{"bootstrap", "gui/501", filepath.Join(homeDir, "Library", "LaunchAgents", "com.openase.plist")}},
		{name: "launchctl", args: []string{"enable", "gui/501/com.openase"}},
		{name: "launchctl", args: []string{"kickstart", "-k", "gui/501/com.openase"}},
	}
	if !reflect.DeepEqual(runner.calls, expected) {
		t.Fatalf("unexpected commands: %+v", runner.calls)
	}
}

func TestLaunchdLogsUsesTail(t *testing.T) {
	homeDir := t.TempDir()
	runner := &recordingRunner{}
	manager := newLaunchdUserManagerForTest(homeDir, 501, runner)
	opts, err := provider.NewUserServiceLogsOptions(50, false, io.Discard, io.Discard)
	if err != nil {
		t.Fatalf("NewUserServiceLogsOptions returned error: %v", err)
	}

	if err := manager.Logs(context.Background(), provider.MustParseServiceName("openase"), opts); err != nil {
		t.Fatalf("Logs returned error: %v", err)
	}

	expected := recordedCommand{
		name: "tail",
		args: []string{
			"-n", "50",
			filepath.Join(homeDir, ".openase", "logs", "openase.stdout.log"),
			filepath.Join(homeDir, ".openase", "logs", "openase.stderr.log"),
		},
	}
	if len(runner.calls) != 1 || !reflect.DeepEqual(runner.calls[0], expected) {
		t.Fatalf("unexpected commands: %+v", runner.calls)
	}
}

func testInstallSpec(t *testing.T, homeDir string) provider.UserServiceInstallSpec {
	t.Helper()

	spec, err := provider.NewUserServiceInstallSpec(
		provider.MustParseServiceName("openase"),
		"OpenASE -- Auto Software Engineering Platform",
		provider.MustParseAbsolutePath("/tmp/openase"),
		[]string{"all-in-one", "--config", "/tmp/openase.yaml"},
		provider.MustParseAbsolutePath(filepath.Join(homeDir, ".openase")),
		provider.MustParseAbsolutePath(filepath.Join(homeDir, ".openase", ".env")),
		provider.MustParseAbsolutePath(filepath.Join(homeDir, ".openase", "logs", "openase.stdout.log")),
		provider.MustParseAbsolutePath(filepath.Join(homeDir, ".openase", "logs", "openase.stderr.log")),
	)
	if err != nil {
		t.Fatalf("NewUserServiceInstallSpec returned error: %v", err)
	}

	return spec
}

type recordedCommand struct {
	name string
	args []string
}

type recordingRunner struct {
	calls   []recordedCommand
	results []error
}

func (r *recordingRunner) Run(_ context.Context, name string, args []string, _ io.Writer, _ io.Writer) error {
	r.calls = append(r.calls, recordedCommand{
		name: name,
		args: append([]string(nil), args...),
	})
	if len(r.results) == 0 {
		return nil
	}

	result := r.results[0]
	r.results = r.results[1:]

	return result
}
