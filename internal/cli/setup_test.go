package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

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
		"",  // write files confirm
	}, "\n"))

	if err := runSetupFlow(context.Background(), input, &output, service, setupFlowOptions{}); err != nil {
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
	if strings.Contains(output.String(), "browser") {
		t.Fatalf("unexpected browser text in output = %q", output.String())
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
	if err := runSetupFlow(context.Background(), strings.NewReader("n\n"), &output, service, setupFlowOptions{}); err != nil {
		t.Fatalf("runSetupFlow() error = %v", err)
	}

	if service.completeCalls != 0 {
		t.Fatalf("complete calls = %d", service.completeCalls)
	}
	if !strings.Contains(output.String(), "left unchanged") {
		t.Fatalf("output = %q", output.String())
	}
}
