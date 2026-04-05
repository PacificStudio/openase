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
			AuthModes: []setup.AuthModeOption{
				{ID: setup.AuthModeDisabled, Name: "Disabled", Description: "disabled"},
				{ID: setup.AuthModeOIDC, Name: "OIDC", Description: "oidc"},
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
				Auth: setup.RawAuthInput{
					Mode: string(setup.AuthModeDisabled),
					OIDC: &setup.RawOIDCInput{
						ClientID:       "openase",
						RedirectURL:    setup.DefaultOIDCRedirectURL,
						Scopes:         setup.DefaultOIDCScopes,
						SessionTTL:     setup.DefaultOIDCSessionTTL,
						SessionIdleTTL: setup.DefaultOIDCIdleTTL,
					},
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
		"",  // disabled auth
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
	if service.completeInput.Auth.Mode != string(setup.AuthModeDisabled) {
		t.Fatalf("complete auth input = %+v", service.completeInput.Auth)
	}
	if !strings.Contains(output.String(), "OpenASE setup completed.") {
		t.Fatalf("output = %q", output.String())
	}
	if !strings.Contains(output.String(), "Open "+defaultSetupURL) {
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
			AuthModes: []setup.AuthModeOption{
				{ID: setup.AuthModeDisabled, Name: "Disabled", Description: "disabled"},
				{ID: setup.AuthModeOIDC, Name: "OIDC", Description: "oidc"},
			},
			Defaults: setup.Defaults{
				ManualDatabase: setup.RawDatabaseInput{
					Host:    "127.0.0.1",
					Port:    5432,
					Name:    "openase",
					User:    "openase",
					SSLMode: "disable",
				},
				Auth: setup.RawAuthInput{
					Mode: string(setup.AuthModeDisabled),
					OIDC: &setup.RawOIDCInput{
						ClientID:       "openase",
						RedirectURL:    setup.DefaultOIDCRedirectURL,
						Scopes:         setup.DefaultOIDCScopes,
						SessionTTL:     setup.DefaultOIDCSessionTTL,
						SessionIdleTTL: setup.DefaultOIDCIdleTTL,
					},
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
		"2",                          // oidc auth
		"https://example.auth0.com/", // issuer
		"",                           // client id default
		"oidc-secret",                // client secret
		"",                           // redirect default
		"",                           // scopes default
		"admin@example.com",          // bootstrap admins
		"",                           // allowed domains
		"2",                          // systemd runtime
		"",                           // final confirm
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

	if service.completeInput.Auth.Mode != string(setup.AuthModeOIDC) {
		t.Fatalf("complete auth input = %+v", service.completeInput.Auth)
	}
	if service.completeInput.Auth.OIDC == nil || service.completeInput.Auth.OIDC.ClientSecret != "oidc-secret" {
		t.Fatalf("oidc config = %+v", service.completeInput.Auth.OIDC)
	}
	if manager.applySpec.Name != managedServiceName {
		t.Fatalf("service install spec = %+v", manager.applySpec)
	}
	if !strings.Contains(output.String(), "systemctl --user status openase") {
		t.Fatalf("output = %q", output.String())
	}
	if !strings.Contains(output.String(), oidcGuideURL) {
		t.Fatalf("output = %q", output.String())
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
		"",  // disabled auth
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
	}, setup.AuthConfig{Mode: setup.AuthModeDisabled}, &installedSetupService{
		Name:     managedServiceName,
		Platform: "launchd",
		InstallSpec: provider.UserServiceInstallSpec{
			Name:       managedServiceName,
			StdoutPath: provider.MustParseAbsolutePath(filepath.Join(homeDir, ".openase", "logs", "openase.stdout.log")),
			StderrPath: provider.MustParseAbsolutePath(filepath.Join(homeDir, ".openase", "logs", "openase.stderr.log")),
		},
		LaunchdTarget: launchdServiceTarget(501, managedServiceName),
		LaunchdPlist:  launchdPlistPath(homeDir, managedServiceName),
	})

	text := output.String()
	for _, want := range []string{
		"Service:  openase via launchd",
		"launchctl print gui/501/com.openase",
		filepath.Join(homeDir, "Library", "LaunchAgents", "com.openase.plist"),
		filepath.Join(homeDir, ".openase", "logs", "openase.stdout.log"),
		filepath.Join(homeDir, ".openase", "logs", "openase.stderr.log"),
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("printSetupSuccess() output = %q, want substring %q", text, want)
		}
	}
}
