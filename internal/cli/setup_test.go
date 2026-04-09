package cli

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BetterAndBetterII/openase/internal/provider"
	"github.com/BetterAndBetterII/openase/internal/setup"
)

type stubSetupFlowService struct {
	bootstrap      setup.Bootstrap
	prepared       setup.PreparedDatabase
	prepareErr     error
	completeResult setup.CompleteResult
	completeErr    error
	prepareInput   setup.RawDatabaseSourceInput
	completeInput  setup.RawCompleteRequest
	completeCalls  int
}

type stubSetupManager struct {
	platform  string
	applySpec provider.UserServiceInstallSpec
}

func (s *stubSetupManager) Platform() string {
	return s.platform
}

func (s *stubSetupManager) Apply(_ context.Context, spec provider.UserServiceInstallSpec) error {
	s.applySpec = spec
	return nil
}

func (s *stubSetupManager) Down(context.Context, provider.ServiceName) error {
	return nil
}

func (s *stubSetupManager) Restart(context.Context, provider.ServiceName) error {
	return nil
}

func (s *stubSetupManager) Logs(context.Context, provider.ServiceName, provider.UserServiceLogsOptions) error {
	return nil
}

func (s *stubSetupFlowService) Bootstrap(context.Context) (setup.Bootstrap, error) {
	return s.bootstrap, nil
}

func (s *stubSetupFlowService) PrepareDatabase(_ context.Context, input setup.RawDatabaseSourceInput) (setup.PreparedDatabase, error) {
	s.prepareInput = input
	return s.prepared, s.prepareErr
}

func (s *stubSetupFlowService) Complete(_ context.Context, input setup.RawCompleteRequest) (setup.CompleteResult, error) {
	s.completeInput = input
	s.completeCalls++
	return s.completeResult, s.completeErr
}

func TestRunSetupFlowManualDatabase(t *testing.T) {
	service := &stubSetupFlowService{
		bootstrap: setup.Bootstrap{
			ConfigPath: "/tmp/.openase/config.yaml",
			Sources: []setup.DatabaseSourceOption{
				{ID: setup.DatabaseSourceDocker, Name: "Docker", Description: "docker"},
				{ID: setup.DatabaseSourceManual, Name: "Manual", Description: "manual"},
			},
			CLI: []setup.CLIDiagnostic{
				{ID: "git", Name: "Git", Command: "git", Status: "ready", Version: "git version 2.48.1", Path: "/usr/bin/git"},
				{ID: "codex", Name: "OpenAI Codex", Command: "codex", Status: "ready", Version: "codex 1.0.0", Path: "/usr/local/bin/codex"},
				{ID: "claude", Name: "Claude Code", Command: "claude", Status: "missing"},
			},
			Defaults: setup.Defaults{
				ManualDatabase: setup.RawDatabaseInput{
					Host:    "127.0.0.1",
					Port:    5432,
					Name:    "openase",
					User:    "openase",
					SSLMode: "disable",
				},
				DockerDatabase: setup.RawDockerDatabaseInput{
					ContainerName: "openase-local-postgres",
					DatabaseName:  "openase",
					User:          "openase",
					Port:          15432,
					VolumeName:    "openase-local-postgres-data",
					Image:         "postgres:16-alpine",
				},
			},
		},
		prepared: setup.PreparedDatabase{
			Source: setup.DatabaseSourceManual,
			Config: setup.DatabaseConfig{
				Host:    "127.0.0.1",
				Port:    5432,
				Name:    "openase",
				User:    "openase",
				SSLMode: "disable",
			},
		},
		completeResult: setup.CompleteResult{
			ConfigPath:       "/tmp/.openase/config.yaml",
			EnvPath:          "/tmp/.openase/.env",
			OrganizationName: setup.DefaultOrganizationName,
			OrganizationSlug: setup.DefaultOrganizationSlug,
			ProjectName:      setup.DefaultProjectName,
			ProjectSlug:      setup.DefaultProjectSlug,
		},
	}

	var output bytes.Buffer
	input := strings.NewReader(strings.Join([]string{
		"2", // manual database
		"",  // host
		"",  // port
		"",  // db
		"",  // user
		"",  // password
		"",  // ssl mode default
		"",  // config-only runtime
		"",  // write files confirm
	}, "\n"))

	if err := runSetupFlowWithDeps(context.Background(), input, &output, service, setupFlowOptions{}, setupFlowDeps{
		buildUserServiceManager: func() (provider.UserServiceManager, error) {
			t.Fatal("buildUserServiceManager should not be called in config-only mode")
			return nil, nil
		},
		buildManagedServiceInstallSpec: func(string) (provider.UserServiceInstallSpec, error) {
			t.Fatal("buildManagedServiceInstallSpec should not be called in config-only mode")
			return provider.UserServiceInstallSpec{}, nil
		},
		checkManagedUserServiceSupport: func(context.Context) error {
			t.Fatal("checkManagedUserServiceSupport should not be called in config-only mode")
			return nil
		},
		verifyManagedUserService: func(context.Context, provider.ServiceName) error {
			t.Fatal("verifyManagedUserService should not be called in config-only mode")
			return nil
		},
	}); err != nil {
		t.Fatalf("runSetupFlow() error = %v", err)
	}

	if service.completeCalls != 1 {
		t.Fatalf("complete calls = %d", service.completeCalls)
	}
	if service.prepareInput.Type != string(setup.DatabaseSourceManual) {
		t.Fatalf("prepare input = %+v", service.prepareInput)
	}
	if !strings.Contains(output.String(), "OpenASE setup completed.") {
		t.Fatalf("output = %q", output.String())
	}
	if !strings.Contains(output.String(), "Open "+defaultSetupURL) {
		t.Fatalf("output = %q", output.String())
	}
	if !strings.Contains(output.String(), "openase auth bootstrap create-link --return-to / --format text") {
		t.Fatalf("output = %q", output.String())
	}
	if strings.Contains(output.String(), "legacy web setup") {
		t.Fatalf("unexpected legacy web setup text in output = %q", output.String())
	}
}

func TestRunSetupFlowOIDCAndManagedService(t *testing.T) {
	manager := &stubSetupManager{platform: "systemd --user"}
	service := &stubSetupFlowService{
		bootstrap: setup.Bootstrap{
			ConfigPath: "/tmp/.openase/config.yaml",
			Sources: []setup.DatabaseSourceOption{
				{ID: setup.DatabaseSourceManual, Name: "Manual", Description: "manual"},
			},
			Defaults: setup.Defaults{
				ManualDatabase: setup.RawDatabaseInput{
					Host:    "127.0.0.1",
					Port:    5432,
					Name:    "openase",
					User:    "openase",
					SSLMode: "disable",
				},
			},
		},
		prepared: setup.PreparedDatabase{
			Source: setup.DatabaseSourceManual,
			Config: setup.DatabaseConfig{
				Host:    "127.0.0.1",
				Port:    5432,
				Name:    "openase",
				User:    "openase",
				SSLMode: "disable",
			},
		},
		completeResult: setup.CompleteResult{
			ConfigPath:       "/tmp/.openase/config.yaml",
			EnvPath:          "/tmp/.openase/.env",
			OrganizationName: setup.DefaultOrganizationName,
			OrganizationSlug: setup.DefaultOrganizationSlug,
			ProjectName:      setup.DefaultProjectName,
			ProjectSlug:      setup.DefaultProjectSlug,
		},
	}

	var output bytes.Buffer
	input := strings.NewReader(strings.Join([]string{
		"",                     // manual database
		"", "", "", "", "", "", // manual db defaults
		"2", // systemd runtime
		"",  // final confirm
	}, "\n"))

	if err := runSetupFlowWithDeps(context.Background(), input, &output, service, setupFlowOptions{}, setupFlowDeps{
		buildUserServiceManager: func() (provider.UserServiceManager, error) {
			return manager, nil
		},
		buildManagedServiceInstallSpec: func(configPath string) (provider.UserServiceInstallSpec, error) {
			return provider.UserServiceInstallSpec{
				Name:       managedServiceName,
				StdoutPath: provider.MustParseAbsolutePath("/tmp/.openase/logs/openase.stdout.log"),
				StderrPath: provider.MustParseAbsolutePath("/tmp/.openase/logs/openase.stderr.log"),
			}, nil
		},
		checkManagedUserServiceSupport: func(context.Context) error {
			return nil
		},
		verifyManagedUserService: func(context.Context, provider.ServiceName) error {
			return nil
		},
	}); err != nil {
		t.Fatalf("runSetupFlowWithDeps() error = %v", err)
	}

	if manager.applySpec.Name != managedServiceName {
		t.Fatalf("service install spec = %+v", manager.applySpec)
	}
	if !strings.Contains(output.String(), "systemctl --user status openase") {
		t.Fatalf("output = %q", output.String())
	}
	if !strings.Contains(output.String(), "openase auth break-glass disable-oidc") {
		t.Fatalf("output = %q", output.String())
	}
}

func TestRunSetupFlowOIDCAndLaunchdManagedService(t *testing.T) {
	manager := &stubSetupManager{platform: "launchd"}
	service := &stubSetupFlowService{
		bootstrap: setup.Bootstrap{
			ConfigPath: "/tmp/.openase/config.yaml",
			Sources: []setup.DatabaseSourceOption{
				{ID: setup.DatabaseSourceManual, Name: "Manual", Description: "manual"},
			},
			Defaults: setup.Defaults{
				ManualDatabase: setup.RawDatabaseInput{
					Host:    "127.0.0.1",
					Port:    5432,
					Name:    "openase",
					User:    "openase",
					SSLMode: "disable",
				},
			},
		},
		prepared: setup.PreparedDatabase{
			Source: setup.DatabaseSourceManual,
			Config: setup.DatabaseConfig{
				Host:    "127.0.0.1",
				Port:    5432,
				Name:    "openase",
				User:    "openase",
				SSLMode: "disable",
			},
		},
		completeResult: setup.CompleteResult{
			ConfigPath:       "/tmp/.openase/config.yaml",
			EnvPath:          "/tmp/.openase/.env",
			OrganizationName: setup.DefaultOrganizationName,
			OrganizationSlug: setup.DefaultOrganizationSlug,
			ProjectName:      setup.DefaultProjectName,
			ProjectSlug:      setup.DefaultProjectSlug,
		},
	}

	var (
		output           bytes.Buffer
		supportChecked   bool
		managerBuilt     bool
		installSpecBuilt bool
		serviceVerified  bool
	)
	input := strings.NewReader(strings.Join([]string{
		"",                     // manual database
		"", "", "", "", "", "", // manual db defaults
		"2", // launchd runtime
		"",  // final confirm
	}, "\n"))

	if err := runSetupFlowWithDeps(context.Background(), input, &output, service, setupFlowOptions{}, setupFlowDeps{
		buildUserServiceManager: func() (provider.UserServiceManager, error) {
			managerBuilt = true
			return manager, nil
		},
		buildManagedServiceInstallSpec: func(configPath string) (provider.UserServiceInstallSpec, error) {
			installSpecBuilt = true
			return provider.UserServiceInstallSpec{
				Name:       managedServiceName,
				StdoutPath: provider.MustParseAbsolutePath("/Users/tester/.openase/logs/openase.stdout.log"),
				StderrPath: provider.MustParseAbsolutePath("/Users/tester/.openase/logs/openase.stderr.log"),
			}, nil
		},
		checkManagedUserServiceSupport: func(context.Context) error {
			supportChecked = true
			return nil
		},
		verifyManagedUserService: func(context.Context, provider.ServiceName) error {
			serviceVerified = true
			return nil
		},
		buildInstalledSetupService: func(context.Context, string, provider.UserServiceInstallSpec) *installedSetupService {
			return &installedSetupService{
				Name:     managedServiceName,
				Platform: "launchd",
				InstallSpec: provider.UserServiceInstallSpec{
					Name:       managedServiceName,
					StdoutPath: provider.MustParseAbsolutePath("/Users/tester/.openase/logs/openase.stdout.log"),
					StderrPath: provider.MustParseAbsolutePath("/Users/tester/.openase/logs/openase.stderr.log"),
				},
				LaunchdTarget: "gui/501/com.openase",
				LaunchdPlist:  "/Users/tester/Library/LaunchAgents/com.openase.plist",
			}
		},
		goos: "darwin",
	}); err != nil {
		t.Fatalf("runSetupFlowWithDeps() error = %v", err)
	}

	if !supportChecked || !managerBuilt || !installSpecBuilt || !serviceVerified {
		t.Fatalf("launchd flow hooks support=%t manager=%t spec=%t verify=%t", supportChecked, managerBuilt, installSpecBuilt, serviceVerified)
	}
	if manager.applySpec.Name != managedServiceName {
		t.Fatalf("service install spec = %+v", manager.applySpec)
	}

	text := output.String()
	for _, want := range []string{
		"Runtime:   managed user service via launchd",
		"Service:  openase via launchd",
		"launchctl print gui/501/com.openase",
		"/Users/tester/Library/LaunchAgents/com.openase.plist",
		"/Users/tester/.openase/logs/openase.stdout.log",
		"/Users/tester/.openase/logs/openase.stderr.log",
		"openase auth break-glass disable-oidc",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("output = %q, want substring %q", text, want)
		}
	}
	if strings.Contains(text, "systemctl --user") {
		t.Fatalf("output should not contain systemd hints: %q", text)
	}
}

func TestRunSetupFlowFallsBackWhenSystemdUnavailable(t *testing.T) {
	service := &stubSetupFlowService{
		bootstrap: setup.Bootstrap{
			ConfigPath: "/tmp/.openase/config.yaml",
			Sources: []setup.DatabaseSourceOption{
				{ID: setup.DatabaseSourceManual, Name: "Manual", Description: "manual"},
			},
			Defaults: setup.Defaults{
				ManualDatabase: setup.RawDatabaseInput{
					Host:    "127.0.0.1",
					Port:    5432,
					Name:    "openase",
					User:    "openase",
					SSLMode: "disable",
				},
			},
		},
		prepared: setup.PreparedDatabase{
			Source: setup.DatabaseSourceManual,
			Config: setup.DatabaseConfig{
				Host:    "127.0.0.1",
				Port:    5432,
				Name:    "openase",
				User:    "openase",
				SSLMode: "disable",
			},
		},
		completeResult: setup.CompleteResult{
			ConfigPath: "/tmp/.openase/config.yaml",
			EnvPath:    "/tmp/.openase/.env",
		},
	}

	var output bytes.Buffer
	input := strings.NewReader(strings.Join([]string{
		"", "", "", "", "", "", "", // manual database defaults
		"2", // choose service mode
		"",  // accept fallback to config-only
		"",  // final confirm
	}, "\n"))

	if err := runSetupFlowWithDeps(context.Background(), input, &output, service, setupFlowOptions{}, setupFlowDeps{
		buildUserServiceManager: func() (provider.UserServiceManager, error) {
			t.Fatal("buildUserServiceManager should not run after fallback")
			return nil, nil
		},
		buildManagedServiceInstallSpec: func(string) (provider.UserServiceInstallSpec, error) {
			t.Fatal("buildManagedServiceInstallSpec should not run after fallback")
			return provider.UserServiceInstallSpec{}, nil
		},
		checkManagedUserServiceSupport: func(context.Context) error {
			return errors.New("systemd --user is unavailable")
		},
		verifyManagedUserService: func(context.Context, provider.ServiceName) error {
			t.Fatal("verifyManagedUserService should not run after fallback")
			return nil
		},
	}); err != nil {
		t.Fatalf("runSetupFlowWithDeps() error = %v", err)
	}

	if service.completeCalls != 1 {
		t.Fatalf("complete calls = %d", service.completeCalls)
	}
	if !strings.Contains(output.String(), "Continue with config-only setup instead?") {
		t.Fatalf("output = %q", output.String())
	}
}

func TestRunSetupFlowFallsBackWhenLaunchdUnavailableOnDarwin(t *testing.T) {
	service := &stubSetupFlowService{
		bootstrap: setup.Bootstrap{
			ConfigPath: "/tmp/.openase/config.yaml",
			Sources: []setup.DatabaseSourceOption{
				{ID: setup.DatabaseSourceManual, Name: "Manual", Description: "manual"},
			},
			Defaults: setup.Defaults{
				ManualDatabase: setup.RawDatabaseInput{
					Host:    "127.0.0.1",
					Port:    5432,
					Name:    "openase",
					User:    "openase",
					SSLMode: "disable",
				},
			},
		},
		prepared: setup.PreparedDatabase{
			Source: setup.DatabaseSourceManual,
			Config: setup.DatabaseConfig{
				Host:    "127.0.0.1",
				Port:    5432,
				Name:    "openase",
				User:    "openase",
				SSLMode: "disable",
			},
		},
		completeResult: setup.CompleteResult{
			ConfigPath: "/tmp/.openase/config.yaml",
			EnvPath:    "/tmp/.openase/.env",
		},
	}

	var (
		output         bytes.Buffer
		supportChecked bool
	)
	input := strings.NewReader(strings.Join([]string{
		"", "", "", "", "", "", "", // manual database defaults
		"2", // choose service mode
		"",  // accept fallback to config-only
		"",  // final confirm
	}, "\n"))

	if err := runSetupFlowWithDeps(context.Background(), input, &output, service, setupFlowOptions{}, setupFlowDeps{
		buildUserServiceManager: func() (provider.UserServiceManager, error) {
			t.Fatal("buildUserServiceManager should not run after fallback")
			return nil, nil
		},
		buildManagedServiceInstallSpec: func(string) (provider.UserServiceInstallSpec, error) {
			t.Fatal("buildManagedServiceInstallSpec should not run after fallback")
			return provider.UserServiceInstallSpec{}, nil
		},
		checkManagedUserServiceSupport: func(context.Context) error {
			supportChecked = true
			return errors.New("launchd is unavailable")
		},
		verifyManagedUserService: func(context.Context, provider.ServiceName) error {
			t.Fatal("verifyManagedUserService should not run after fallback")
			return nil
		},
		goos: "darwin",
	}); err != nil {
		t.Fatalf("runSetupFlowWithDeps() error = %v", err)
	}

	if !supportChecked {
		t.Fatal("expected managed service support check to run")
	}
	if service.completeCalls != 1 {
		t.Fatalf("complete calls = %d", service.completeCalls)
	}

	text := output.String()
	for _, want := range []string{
		"Current machine cannot use the managed OpenASE user service via launchd: launchd is unavailable",
		"Continue with config-only setup instead?",
		"Runtime:   config-only",
		"OpenASE setup completed.",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("output = %q, want substring %q", text, want)
		}
	}
	if strings.Contains(text, "Service:  openase via launchd") {
		t.Fatalf("unexpected launchd success hints in output = %q", text)
	}
}

func TestRunSetupFlowDeclinesOverwrite(t *testing.T) {
	service := &stubSetupFlowService{
		bootstrap: setup.Bootstrap{
			ConfigExists: true,
			ConfigPath:   "/tmp/.openase/config.yaml",
		},
	}

	var output bytes.Buffer
	if err := runSetupFlowWithDeps(context.Background(), strings.NewReader("n\n"), &output, service, setupFlowOptions{}, setupFlowDeps{}); err != nil {
		t.Fatalf("runSetupFlow() error = %v", err)
	}

	if service.completeCalls != 0 {
		t.Fatalf("complete calls = %d", service.completeCalls)
	}
	if !strings.Contains(output.String(), "left unchanged") {
		t.Fatalf("output = %q", output.String())
	}
}

func TestPrintSetupFailureShowsDoctorHint(t *testing.T) {
	var output bytes.Buffer

	printSetupFailure(&output, defaultSetupConfigPath)

	if !strings.Contains(output.String(), "Setup did not complete cleanly.") {
		t.Fatalf("output = %q", output.String())
	}
	if !strings.Contains(output.String(), "openase doctor --config ~/.openase/config.yaml") {
		t.Fatalf("output = %q", output.String())
	}
}

func TestSetupManagedUserServicePrompt(t *testing.T) {
	tests := []struct {
		goos string
		want string
	}{
		{goos: "linux", want: "systemd --user"},
		{goos: "darwin", want: "launchd"},
		{goos: "freebsd", want: "Install Managed User Service:"},
	}

	for _, tt := range tests {
		t.Run(tt.goos, func(t *testing.T) {
			if got := setupManagedUserServicePrompt(tt.goos); !strings.Contains(got, tt.want) {
				t.Fatalf("setupManagedUserServicePrompt(%q) = %q, want substring %q", tt.goos, got, tt.want)
			}
		})
	}
}

func TestPrintSetupSuccessLaunchdHints(t *testing.T) {
	var output bytes.Buffer
	homeDir := "/Users/tester"
	printSetupSuccess(&output, setup.CompleteResult{
		ConfigPath: "/tmp/.openase/config.yaml",
		EnvPath:    "/tmp/.openase/.env",
	}, setup.PreparedDatabase{
		Source: setup.DatabaseSourceManual,
		Config: setup.DatabaseConfig{
			Host: "127.0.0.1",
			Port: 5432,
			Name: "openase",
			User: "openase",
		},
	}, &installedSetupService{
		Name:     managedServiceName,
		Platform: "launchd",
		InstallSpec: provider.UserServiceInstallSpec{
			Name:       managedServiceName,
			StdoutPath: provider.MustParseAbsolutePath(filepath.Join(homeDir, ".openase", "logs", "openase.stdout.log")),
			StderrPath: provider.MustParseAbsolutePath(filepath.Join(homeDir, ".openase", "logs", "openase.stderr.log")),
		},
		LaunchdTarget: "user/501/com.openase",
		LaunchdPlist:  launchdPlistPath(homeDir, managedServiceName),
	})

	text := output.String()
	for _, want := range []string{
		"Service:  openase via launchd",
		"launchctl print user/501/com.openase",
		filepath.Join(homeDir, "Library", "LaunchAgents", "com.openase.plist"),
		filepath.Join(homeDir, ".openase", "logs", "openase.stdout.log"),
		filepath.Join(homeDir, ".openase", "logs", "openase.stderr.log"),
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("printSetupSuccess() output = %q, want substring %q", text, want)
		}
	}
}
