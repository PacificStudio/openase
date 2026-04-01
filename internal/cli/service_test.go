package cli

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BetterAndBetterII/openase/internal/provider"
)

type stubUserServiceManager struct {
	platform  string
	applySpec provider.UserServiceInstallSpec
}

func (s *stubUserServiceManager) Platform() string {
	return s.platform
}

func (s *stubUserServiceManager) Apply(_ context.Context, spec provider.UserServiceInstallSpec) error {
	s.applySpec = spec
	return nil
}

func (s *stubUserServiceManager) Down(context.Context, provider.ServiceName) error {
	return nil
}

func (s *stubUserServiceManager) Restart(context.Context, provider.ServiceName) error {
	return nil
}

func (s *stubUserServiceManager) Logs(context.Context, provider.ServiceName, provider.UserServiceLogsOptions) error {
	return nil
}

func TestUpCommandRunsSetupWizardWhenConfigMissing(t *testing.T) {
	rootOptions := &rootOptions{}
	var setupCalls int

	command := newUpCommandWithDeps(rootOptions, upCommandDeps{
		resolveConfigPath: func(string) (provider.AbsolutePath, error) {
			return "", nil
		},
		runSetupWizard: func(_ context.Context, out io.Writer) error {
			setupCalls++
			_, _ = io.WriteString(out, "wizard started\n")
			return nil
		},
		buildUserServiceManager: func() (provider.UserServiceManager, error) {
			t.Fatal("buildUserServiceManager should not be called when config is missing")
			return nil, nil
		},
		buildManagedServiceInstallSpec: func(string) (provider.UserServiceInstallSpec, error) {
			t.Fatal("buildManagedServiceInstallSpec should not be called when config is missing")
			return provider.UserServiceInstallSpec{}, nil
		},
	})

	var stdout bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&stdout)
	command.SetArgs(nil)

	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}
	if setupCalls != 1 {
		t.Fatalf("expected setup wizard to run once, got %d", setupCalls)
	}
	if !strings.Contains(stdout.String(), "wizard started") {
		t.Fatalf("expected setup output, got %q", stdout.String())
	}
}

func TestUpCommandAppliesManagedServiceWhenConfigExists(t *testing.T) {
	rootOptions := &rootOptions{}
	manager := &stubUserServiceManager{platform: "test-platform"}
	configPath := provider.MustParseAbsolutePath(filepath.Join(t.TempDir(), "config.yaml"))

	command := newUpCommandWithDeps(rootOptions, upCommandDeps{
		resolveConfigPath: func(string) (provider.AbsolutePath, error) {
			return configPath, nil
		},
		runSetupWizard: func(context.Context, io.Writer) error {
			t.Fatal("runSetupWizard should not be called when config exists")
			return nil
		},
		buildUserServiceManager: func() (provider.UserServiceManager, error) {
			return manager, nil
		},
		buildManagedServiceInstallSpec: func(configFile string) (provider.UserServiceInstallSpec, error) {
			if configFile != configPath.String() {
				t.Fatalf("expected config path %q, got %q", configPath.String(), configFile)
			}
			return provider.UserServiceInstallSpec{Name: managedServiceName}, nil
		},
	})

	var stdout bytes.Buffer
	command.SetOut(&stdout)
	command.SetErr(&stdout)
	command.SetArgs(nil)

	if err := command.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("ExecuteContext returned error: %v", err)
	}
	if manager.applySpec.Name != managedServiceName {
		t.Fatalf("expected Apply to receive service name %q, got %q", managedServiceName, manager.applySpec.Name)
	}
	if !strings.Contains(stdout.String(), "openase service applied via test-platform") {
		t.Fatalf("expected service apply message, got %q", stdout.String())
	}
}

func TestUpCommandReturnsConfigResolutionError(t *testing.T) {
	rootOptions := &rootOptions{configFile: "/missing/config.yaml"}

	command := newUpCommandWithDeps(rootOptions, upCommandDeps{
		resolveConfigPath: func(string) (provider.AbsolutePath, error) {
			return "", fmt.Errorf("stat config path %q: no such file or directory", rootOptions.configFile)
		},
		runSetupWizard: func(context.Context, io.Writer) error {
			t.Fatal("runSetupWizard should not be called when config resolution fails")
			return nil
		},
		buildUserServiceManager: func() (provider.UserServiceManager, error) {
			t.Fatal("buildUserServiceManager should not be called when config resolution fails")
			return nil, nil
		},
		buildManagedServiceInstallSpec: func(string) (provider.UserServiceInstallSpec, error) {
			t.Fatal("buildManagedServiceInstallSpec should not be called when config resolution fails")
			return provider.UserServiceInstallSpec{}, nil
		},
	})

	if err := command.ExecuteContext(context.Background()); err == nil {
		t.Fatal("expected ExecuteContext to return an error")
	}
}

func TestManagedServiceConfigCandidatesIncludeSetupConfigPath(t *testing.T) {
	homeDir := t.TempDir()

	candidates := managedServiceConfigCandidates("/workspace", homeDir)

	expected := filepath.Join(homeDir, ".openase", "config.yaml")
	for _, candidate := range candidates {
		if candidate == expected {
			return
		}
	}

	t.Fatalf("expected candidates to include %q, got %v", expected, candidates)
}
