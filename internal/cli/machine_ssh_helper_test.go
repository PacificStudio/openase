package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	sshinfra "github.com/BetterAndBetterII/openase/internal/infra/ssh"
	"github.com/google/uuid"
)

func TestRunMachineSSHBootstrapUploadsBinaryEnvAndService(t *testing.T) {
	ctx := context.Background()
	machineID := uuid.New()
	agentCLIPath := "/usr/local/bin/codex"

	platformSession := &machineSSHTestSession{
		combinedOutput: "Linux\n/home/remote\nsystemd\n1000\n",
	}
	binaryUploadSession := &machineSSHTestSession{}
	envUploadSession := &machineSSHTestSession{}
	serviceUploadSession := &machineSSHTestSession{}
	restartSession := &machineSSHTestSession{combinedOutput: "active\n"}
	client := &machineSSHTestClient{sessions: []sshinfra.Session{
		platformSession,
		binaryUploadSession,
		envUploadSession,
		serviceUploadSession,
		restartSession,
	}}

	result, err := runMachineSSHBootstrap(ctx, machineSSHBootstrapDeps{
		getClient: func(context.Context, catalogdomain.Machine) (sshinfra.Client, error) {
			return client, nil
		},
		issueToken: func(context.Context, uuid.UUID, time.Duration, string, string) (machineChannelTokenResponse, error) {
			return machineChannelTokenResponse{
				Token:           "ase_machine_test",
				TokenID:         "token-1",
				MachineID:       machineID.String(),
				ControlPlaneURL: "https://openase.example.com",
			}, nil
		},
		readLocalFile: func(string) ([]byte, error) { return []byte("openase-binary"), nil },
		resolveExecutable: func() (string, error) {
			return "/usr/local/bin/openase", nil
		},
	}, machineSSHBootstrapInput{
		Machine: catalogdomain.Machine{
			ID:             machineID,
			Name:           "reverse-01",
			Host:           "10.0.1.11",
			ConnectionMode: catalogdomain.MachineConnectionModeWSReverse,
			AgentCLIPath:   &agentCLIPath,
		},
		TokenTTL:          24 * time.Hour,
		ControlPlaneURL:   "https://openase.example.com",
		HeartbeatInterval: 15 * time.Second,
	})
	if err != nil {
		t.Fatalf("runMachineSSHBootstrap() error = %v", err)
	}

	layout := buildMachineSSHLayout("/home/remote")
	servicePath, _, _ := buildMachineSSHServiceFile(machineSSHPlatformInfo{
		RemoteHome:     "/home/remote",
		ServiceManager: "systemd",
		UID:            "1000",
	}, layout, machineSSHServiceInstallInput{
		BinaryPath:        layout.RemoteBinaryPath,
		EnvironmentFile:   layout.EnvironmentFile,
		WorkingDirectory:  layout.WorkDir,
		StdoutPath:        layout.StdoutPath,
		StderrPath:        layout.StderrPath,
		AgentCLIPath:      agentCLIPath,
		HeartbeatInterval: 15 * time.Second,
		UID:               "1000",
	})

	if result.RemoteBinaryPath != layout.RemoteBinaryPath {
		t.Fatalf("RemoteBinaryPath = %q, want %q", result.RemoteBinaryPath, layout.RemoteBinaryPath)
	}
	if result.EnvironmentFile != layout.EnvironmentFile {
		t.Fatalf("EnvironmentFile = %q, want %q", result.EnvironmentFile, layout.EnvironmentFile)
	}
	if result.ServiceFile != servicePath {
		t.Fatalf("ServiceFile = %q, want %q", result.ServiceFile, servicePath)
	}
	if result.TokenID != "token-1" {
		t.Fatalf("TokenID = %q, want token-1", result.TokenID)
	}
	if got := binaryUploadSession.stdin.String(); got != "openase-binary" {
		t.Fatalf("binary upload content = %q", got)
	}
	envBody := envUploadSession.stdin.String()
	if !strings.Contains(envBody, "OPENASE_MACHINE_ID="+`"`+machineID.String()+`"`) {
		t.Fatalf("env upload missing machine id: %q", envBody)
	}
	if !strings.Contains(envBody, `OPENASE_MACHINE_CHANNEL_TOKEN="ase_machine_test"`) {
		t.Fatalf("env upload missing token: %q", envBody)
	}
	if !strings.Contains(envBody, `OPENASE_MACHINE_HEARTBEAT_INTERVAL="15s"`) {
		t.Fatalf("env upload missing heartbeat interval: %q", envBody)
	}
	serviceBody := serviceUploadSession.stdin.String()
	if !strings.Contains(serviceBody, `ExecStart="/home/remote/.openase/bin/openase" "machine-agent" "run"`) {
		t.Fatalf("service upload missing machine-agent exec start: %q", serviceBody)
	}
	if !strings.Contains(serviceBody, `--agent-cli-path" "/usr/local/bin/codex"`) {
		t.Fatalf("service upload missing agent cli path: %q", serviceBody)
	}
	if !strings.Contains(restartSession.combinedCommand, "systemctl --user restart") {
		t.Fatalf("restart command = %q", restartSession.combinedCommand)
	}
}

func TestRunMachineSSHDiagnosticsReportsBootstrapAndRegistrationIssues(t *testing.T) {
	ctx := context.Background()
	machineID := uuid.New()
	sshUser := "openase"
	workspaceRoot := "/srv/openase/workspaces"
	agentCLIPath := "/usr/local/bin/codex"

	client := &machineSSHTestClient{sessions: []sshinfra.Session{
		&machineSSHTestSession{combinedOutput: "Linux\n/home/remote\nsystemd\n1000\n"},
		&machineSSHTestSession{},
		&machineSSHTestSession{combinedErr: errors.New("exit status 1")},
		&machineSSHTestSession{},
		&machineSSHTestSession{combinedOutput: "authentication failed\n"},
	}}

	result, err := runMachineSSHDiagnostics(ctx, machineSSHDiagnosticDeps{
		getClient: func(context.Context, catalogdomain.Machine) (sshinfra.Client, error) {
			return client, nil
		},
	}, catalogdomain.Machine{
		ID:             machineID,
		Name:           "reverse-01",
		Host:           "10.0.1.11",
		ConnectionMode: catalogdomain.MachineConnectionModeWSReverse,
		SSHUser:        &sshUser,
		WorkspaceRoot:  &workspaceRoot,
		AgentCLIPath:   &agentCLIPath,
		DaemonStatus: catalogdomain.MachineDaemonStatus{
			Registered: false,
		},
	})
	if err != nil {
		t.Fatalf("runMachineSSHDiagnostics() error = %v", err)
	}

	if result.ServiceManager != "systemd" {
		t.Fatalf("ServiceManager = %q, want systemd", result.ServiceManager)
	}
	if len(result.Issues) != 3 {
		t.Fatalf("issue count = %d, want 3 (%+v)", len(result.Issues), result.Issues)
	}
	issueCodes := make([]string, 0, len(result.Issues))
	for _, issue := range result.Issues {
		issueCodes = append(issueCodes, issue.Code)
	}
	joined := strings.Join(issueCodes, ",")
	for _, code := range []string{
		"machine_agent_binary_missing",
		"daemon_not_registered",
		"machine_agent_auth_failed",
	} {
		if !strings.Contains(joined, code) {
			t.Fatalf("missing issue code %q in %+v", code, issueCodes)
		}
	}
	if !strings.Contains(result.Summary, "found 3 issue(s)") {
		t.Fatalf("Summary = %q", result.Summary)
	}
}

type machineSSHTestClient struct {
	sessions []sshinfra.Session
	index    int
}

func (c *machineSSHTestClient) NewSession() (sshinfra.Session, error) {
	if c.index >= len(c.sessions) {
		return nil, errors.New("unexpected ssh session request")
	}
	session := c.sessions[c.index]
	c.index++
	return session, nil
}

func (c *machineSSHTestClient) SendRequest(string, bool, []byte) (bool, []byte, error) {
	return true, nil, nil
}

func (c *machineSSHTestClient) Close() error {
	return nil
}

type machineSSHTestSession struct {
	combinedCommand string
	combinedOutput  string
	combinedErr     error
	startCommand    string
	stdin           bytes.Buffer
}

func (s *machineSSHTestSession) CombinedOutput(cmd string) ([]byte, error) {
	s.combinedCommand = cmd
	return []byte(s.combinedOutput), s.combinedErr
}

func (s *machineSSHTestSession) StdinPipe() (io.WriteCloser, error) {
	return nopWriteCloser{Writer: &s.stdin}, nil
}

func (s *machineSSHTestSession) StdoutPipe() (io.Reader, error) { return bytes.NewReader(nil), nil }

func (s *machineSSHTestSession) StderrPipe() (io.Reader, error) { return bytes.NewReader(nil), nil }

func (s *machineSSHTestSession) Start(cmd string) error {
	s.startCommand = cmd
	return nil
}

func (s *machineSSHTestSession) Signal(string) error { return nil }

func (s *machineSSHTestSession) Wait() error { return nil }

func (s *machineSSHTestSession) Close() error { return nil }

type nopWriteCloser struct {
	io.Writer
}

func (nopWriteCloser) Close() error { return nil }
