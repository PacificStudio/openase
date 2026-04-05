package userservice

import (
	"context"
	"errors"
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
	//nolint:gosec // test reads a temp file path assembled from the controlled temp home
	unitBytes, err := os.ReadFile(unitPath)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	unit := string(unitBytes)
	for _, expected := range []string{
		"Description=OpenASE -- Auto Software Engineering Platform",
		`ExecStart="/tmp/openase" "all-in-one" "--config" "/tmp/config.yaml"`,
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

func TestSystemdConstructorsPlatformAndLifecycleHelpers(t *testing.T) {
	homeDir := t.TempDir()
	if manager := NewSystemdUserManager(homeDir); manager.homeDir != homeDir || manager.runner == nil {
		t.Fatalf("NewSystemdUserManager() = %+v", manager)
	}

	runner := &recordingRunner{}
	manager := newSystemdUserManagerForTest(homeDir, runner)
	if got := manager.Platform(); got != "systemd --user" {
		t.Fatalf("Platform() = %q", got)
	}
	if err := manager.Down(context.Background(), provider.MustParseServiceName("openase")); err != nil {
		t.Fatalf("Down() error = %v", err)
	}
	if err := manager.Restart(context.Background(), provider.MustParseServiceName("openase")); err != nil {
		t.Fatalf("Restart() error = %v", err)
	}

	expected := []recordedCommand{
		{name: "systemctl", args: []string{"--user", "stop", "openase"}},
		{name: "systemctl", args: []string{"--user", "restart", "openase"}},
	}
	if !reflect.DeepEqual(runner.calls, expected) {
		t.Fatalf("unexpected commands: %+v", runner.calls)
	}

	failing := newSystemdUserManagerForTest(homeDir, &recordingRunner{results: []error{errors.New("boom")}})
	if err := failing.Down(context.Background(), provider.MustParseServiceName("openase")); err == nil || !strings.Contains(err.Error(), "systemctl --user stop openase: boom") {
		t.Fatalf("Down() wrapped error = %v", err)
	}
}

func TestLaunchdApplyWritesPlistAndBootstrapsService(t *testing.T) {
	homeDir := t.TempDir()
	runner := &recordingRunner{
		results: []error{&exec.ExitError{}, &exec.ExitError{}, nil},
	}
	manager := newLaunchdUserManagerForTest(homeDir, runner)
	spec := testInstallSpec(t, homeDir)

	if err := manager.Apply(context.Background(), spec); err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	plistPath := filepath.Join(homeDir, "Library", "LaunchAgents", "com.openase.plist")
	//nolint:gosec // test reads a temp file path assembled from the controlled temp home
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
		{name: "launchctl", args: []string{"print", "user/501/com.openase"}},
		{name: "launchctl", args: []string{"print", "gui/501"}},
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
		results: []error{&exec.ExitError{}, &exec.ExitError{}, nil},
	}
	manager := newLaunchdUserManagerForTest(homeDir, runner)

	if err := manager.Restart(context.Background(), provider.MustParseServiceName("openase")); err != nil {
		t.Fatalf("Restart returned error: %v", err)
	}

	expected := []recordedCommand{
		{name: "launchctl", args: []string{"print", "gui/501/com.openase"}},
		{name: "launchctl", args: []string{"print", "user/501/com.openase"}},
		{name: "launchctl", args: []string{"print", "gui/501"}},
		{name: "launchctl", args: []string{"bootstrap", "gui/501", filepath.Join(homeDir, "Library", "LaunchAgents", "com.openase.plist")}},
		{name: "launchctl", args: []string{"enable", "gui/501/com.openase"}},
		{name: "launchctl", args: []string{"kickstart", "-k", "gui/501/com.openase"}},
	}
	if !reflect.DeepEqual(runner.calls, expected) {
		t.Fatalf("unexpected commands: %+v", runner.calls)
	}
}

func TestLaunchdConstructorsPlatformDownAndRestartLoaded(t *testing.T) {
	homeDir := t.TempDir()
	if manager := NewLaunchdUserManager(homeDir, 501); manager.homeDir != homeDir || manager.uid != 501 || manager.runner == nil {
		t.Fatalf("NewLaunchdUserManager() = %+v", manager)
	}

	runner := &recordingRunner{results: []error{&exec.ExitError{}, &exec.ExitError{}, nil}}
	manager := newLaunchdUserManagerForTest(homeDir, runner)
	if got := manager.Platform(); got != "launchd" {
		t.Fatalf("Platform() = %q", got)
	}
	if err := manager.Down(context.Background(), provider.MustParseServiceName("openase")); err != nil {
		t.Fatalf("Down(unloaded) error = %v", err)
	}
	expectedDownUnloaded := []recordedCommand{
		{name: "launchctl", args: []string{"print", "gui/501/com.openase"}},
		{name: "launchctl", args: []string{"print", "user/501/com.openase"}},
		{name: "launchctl", args: []string{"print", "gui/501"}},
	}
	if !reflect.DeepEqual(runner.calls, expectedDownUnloaded) {
		t.Fatalf("Down(unloaded) calls = %+v", runner.calls)
	}

	runner = &recordingRunner{results: []error{nil}}
	manager = newLaunchdUserManagerForTest(homeDir, runner)
	if err := manager.Down(context.Background(), provider.MustParseServiceName("openase")); err != nil {
		t.Fatalf("Down(loaded) error = %v", err)
	}
	expectedDown := []recordedCommand{
		{name: "launchctl", args: []string{"print", "gui/501/com.openase"}},
		{name: "launchctl", args: []string{"bootout", "gui/501/com.openase"}},
	}
	if !reflect.DeepEqual(runner.calls, expectedDown) {
		t.Fatalf("Down(loaded) calls = %+v", runner.calls)
	}

	runner = &recordingRunner{results: []error{nil}}
	manager = newLaunchdUserManagerForTest(homeDir, runner)
	if err := manager.Restart(context.Background(), provider.MustParseServiceName("openase")); err != nil {
		t.Fatalf("Restart(loaded) error = %v", err)
	}
	expectedRestart := []recordedCommand{
		{name: "launchctl", args: []string{"print", "gui/501/com.openase"}},
		{name: "launchctl", args: []string{"kickstart", "-k", "gui/501/com.openase"}},
	}
	if !reflect.DeepEqual(runner.calls, expectedRestart) {
		t.Fatalf("Restart(loaded) calls = %+v", runner.calls)
	}
}

func TestLaunchdFallsBackToUserDomainWhenGUISessionIsUnavailable(t *testing.T) {
	homeDir := t.TempDir()
	runner := &recordingRunner{
		results: []error{&exec.ExitError{}, &exec.ExitError{}, &exec.ExitError{}, nil},
	}
	manager := newLaunchdUserManagerForTest(homeDir, runner)

	if err := manager.Restart(context.Background(), provider.MustParseServiceName("openase")); err != nil {
		t.Fatalf("Restart returned error: %v", err)
	}

	expected := []recordedCommand{
		{name: "launchctl", args: []string{"print", "gui/501/com.openase"}},
		{name: "launchctl", args: []string{"print", "user/501/com.openase"}},
		{name: "launchctl", args: []string{"print", "gui/501"}},
		{name: "launchctl", args: []string{"print", "user/501"}},
		{name: "launchctl", args: []string{"bootstrap", "user/501", filepath.Join(homeDir, "Library", "LaunchAgents", "com.openase.plist")}},
		{name: "launchctl", args: []string{"enable", "user/501/com.openase"}},
		{name: "launchctl", args: []string{"kickstart", "-k", "user/501/com.openase"}},
	}
	if !reflect.DeepEqual(runner.calls, expected) {
		t.Fatalf("unexpected commands: %+v", runner.calls)
	}
}

func TestLaunchdPrefersLoadedUserDomainOverGUIFallback(t *testing.T) {
	homeDir := t.TempDir()
	runner := &recordingRunner{results: []error{&exec.ExitError{}, nil}}
	manager := newLaunchdUserManagerForTest(homeDir, runner)

	if err := manager.Down(context.Background(), provider.MustParseServiceName("openase")); err != nil {
		t.Fatalf("Down returned error: %v", err)
	}

	expected := []recordedCommand{
		{name: "launchctl", args: []string{"print", "gui/501/com.openase"}},
		{name: "launchctl", args: []string{"print", "user/501/com.openase"}},
		{name: "launchctl", args: []string{"bootout", "user/501/com.openase"}},
	}
	if !reflect.DeepEqual(runner.calls, expected) {
		t.Fatalf("unexpected commands: %+v", runner.calls)
	}
}

func TestLaunchdLogsUsesTail(t *testing.T) {
	homeDir := t.TempDir()
	runner := &recordingRunner{}
	manager := newLaunchdUserManagerForTest(homeDir, runner)
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

func TestLaunchdHelperErrorPathsAndExecRunner(t *testing.T) {
	homeDir := t.TempDir()
	manager := newLaunchdUserManagerForTest(homeDir, &recordingRunner{results: []error{errors.New("boom")}})
	if _, err := manager.isLoaded(context.Background(), provider.MustParseServiceName("openase")); err == nil || !strings.Contains(err.Error(), "probe launchd service gui/501/com.openase: boom") {
		t.Fatalf("isLoaded(unexpected error) = %v", err)
	}

	failing := newLaunchdUserManagerForTest(homeDir, &recordingRunner{results: []error{nil, errors.New("boom")}})
	if err := failing.Restart(context.Background(), provider.MustParseServiceName("openase")); err == nil || !strings.Contains(err.Error(), `restart launchd service "openase" via gui/501`) || !strings.Contains(err.Error(), "launchctl kickstart -k gui/501/com.openase: boom") {
		t.Fatalf("Restart(error) = %v", err)
	}

	command := "sh"
	if _, err := exec.LookPath(command); err != nil {
		t.Skipf("sh not available: %v", err)
	}

	var stdout strings.Builder
	var stderr strings.Builder
	err := (execCommandRunner{}).Run(context.Background(), command, []string{"-c", "printf 'out'; printf 'err' >&2"}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("execCommandRunner.Run() error = %v", err)
	}
	if stdout.String() != "out" || stderr.String() != "err" {
		t.Fatalf("execCommandRunner output = %q / %q", stdout.String(), stderr.String())
	}
}

func TestCheckLaunchdSupportWithRunnerDistinguishesCommonFailureModes(t *testing.T) {
	missingBinary := func(string) (string, error) {
		return "", exec.ErrNotFound
	}
	presentBinary := func(string) (string, error) {
		return "/usr/bin/launchctl", nil
	}
	if _, err := checkLaunchdSupportWithRunner(context.Background(), t.TempDir(), 501, &recordingRunner{}, missingBinary); err == nil || !strings.Contains(err.Error(), "launchctl is not installed") {
		t.Fatalf("missing launchctl error = %v", err)
	}

	missingHome := filepath.Join(t.TempDir(), "missing-home")
	if _, err := checkLaunchdSupportWithRunner(context.Background(), missingHome, 501, &recordingRunner{}, presentBinary); err == nil || !strings.Contains(err.Error(), "launchd LaunchAgent installation requires an accessible home directory") {
		t.Fatalf("missing home error = %v", err)
	}

	runner := &recordingRunner{results: []error{&exec.ExitError{}, &exec.ExitError{}}}
	if _, err := checkLaunchdSupportWithRunner(context.Background(), t.TempDir(), 501, runner, presentBinary); err == nil || !strings.Contains(err.Error(), "launchd is installed, but no usable domain was found for uid 501") || !strings.Contains(err.Error(), "gui/501, user/501") {
		t.Fatalf("session unavailable error = %v", err)
	}
}

func testInstallSpec(t *testing.T, homeDir string) provider.UserServiceInstallSpec {
	t.Helper()

	spec, err := provider.NewUserServiceInstallSpec(
		provider.MustParseServiceName("openase"),
		"OpenASE -- Auto Software Engineering Platform",
		provider.MustParseAbsolutePath("/tmp/openase"),
		[]string{"all-in-one", "--config", "/tmp/config.yaml"},
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
