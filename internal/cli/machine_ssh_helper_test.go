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
			ID:               machineID,
			Name:             "reverse-01",
			Host:             "10.0.1.11",
			ReachabilityMode: catalogdomain.MachineReachabilityModeReverseConnect,
			ExecutionMode:    catalogdomain.MachineExecutionModeWebsocket,
			ConnectionMode:   catalogdomain.MachineConnectionModeWSReverse,
			AgentCLIPath:     &agentCLIPath,
		},
		Topology:          catalogdomain.MachineWebsocketTopologyReverseConnect,
		TokenTTL:          24 * time.Hour,
		ControlPlaneURL:   "https://openase.example.com",
		HeartbeatInterval: 15 * time.Second,
	})
	if err != nil {
		t.Fatalf("runMachineSSHBootstrap() error = %v", err)
	}

	layout := buildMachineSSHLayout("/home/remote", machineAgentServiceName)
	servicePath, _, _ := buildMachineSSHServiceFile(machineSSHPlatformInfo{
		RemoteHome:     "/home/remote",
		ServiceManager: "systemd",
		UID:            "1000",
	}, layout, machineSSHServiceInstallInput{
		ServiceName:       machineAgentServiceName,
		ServiceLabel:      "OpenASE reverse-connect machine-agent",
		BinaryPath:        layout.RemoteBinaryPath,
		EnvironmentFile:   layout.EnvironmentFile,
		WorkingDirectory:  layout.WorkDir,
		StdoutPath:        layout.StdoutPath,
		StderrPath:        layout.StderrPath,
		AgentCLIPath:      agentCLIPath,
		HeartbeatInterval: 15 * time.Second,
		UID:               "1000",
	}, []string{
		"machine-agent",
		"run",
		"--openase-binary-path",
		layout.RemoteBinaryPath,
		"--agent-cli-path",
		agentCLIPath,
		"--heartbeat-interval",
		"15s",
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
	if result.Topology != catalogdomain.MachineWebsocketTopologyReverseConnect.String() {
		t.Fatalf("Topology = %q, want reverse_connect", result.Topology)
	}
	if result.ServiceName != machineAgentServiceName {
		t.Fatalf("ServiceName = %q, want %q", result.ServiceName, machineAgentServiceName)
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

func TestRunMachineSSHBootstrapInstallsRemoteListenerTopology(t *testing.T) {
	ctx := context.Background()
	machineID := uuid.New()
	advertisedEndpoint := "wss://listener.example.com/openase/runtime"
	listenerToken := "listener-secret"

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
			t.Fatal("remote-listener bootstrap should not issue a reverse channel token")
			return machineChannelTokenResponse{}, nil
		},
		readLocalFile: func(string) ([]byte, error) { return []byte("openase-binary"), nil },
		resolveExecutable: func() (string, error) {
			return "/usr/local/bin/openase", nil
		},
	}, machineSSHBootstrapInput{
		Machine: catalogdomain.Machine{
			ID:                 machineID,
			Name:               "listener-01",
			Host:               "10.0.1.21",
			ReachabilityMode:   catalogdomain.MachineReachabilityModeDirectConnect,
			ExecutionMode:      catalogdomain.MachineExecutionModeWebsocket,
			ConnectionMode:     catalogdomain.MachineConnectionModeWSListener,
			AdvertisedEndpoint: &advertisedEndpoint,
			ChannelCredential: catalogdomain.MachineChannelCredential{
				Kind:    catalogdomain.MachineChannelCredentialKindToken,
				TokenID: &listenerToken,
			},
		},
		Topology:        catalogdomain.MachineWebsocketTopologyRemoteListener,
		ListenerAddress: "0.0.0.0:19837",
		ListenerPath:    "/openase/runtime",
	})
	if err != nil {
		t.Fatalf("runMachineSSHBootstrap(remote-listener) error = %v", err)
	}

	layout := buildMachineSSHLayout("/home/remote", machineListenerServiceName)
	if result.ServiceName != machineListenerServiceName {
		t.Fatalf("ServiceName = %q, want %q", result.ServiceName, machineListenerServiceName)
	}
	if result.ConnectionTarget != advertisedEndpoint {
		t.Fatalf("ConnectionTarget = %q, want %q", result.ConnectionTarget, advertisedEndpoint)
	}
	envBody := envUploadSession.stdin.String()
	if !strings.Contains(envBody, `OPENASE_MACHINE_LISTENER_ADDRESS="0.0.0.0:19837"`) {
		t.Fatalf("env upload missing listener address: %q", envBody)
	}
	if !strings.Contains(envBody, `OPENASE_MACHINE_LISTENER_BEARER_TOKEN="listener-secret"`) {
		t.Fatalf("env upload missing listener token: %q", envBody)
	}
	serviceBody := serviceUploadSession.stdin.String()
	if !strings.Contains(serviceBody, `ExecStart="/home/remote/.openase/bin/openase" "machine-agent" "listen"`) {
		t.Fatalf("service upload missing machine-agent listen: %q", serviceBody)
	}
	if !strings.Contains(serviceBody, `"--listen-address" "0.0.0.0:19837" "--path" "/openase/runtime"`) {
		t.Fatalf("service upload missing explicit listener flags: %q", serviceBody)
	}
	if restartCommand := restartSession.combinedCommand; !strings.Contains(restartCommand, "mkdir -p") || !strings.Contains(restartCommand, "/home/remote/.openase/logs") {
		t.Fatalf("restart command should prepare log directories, got %q", restartCommand)
	}
	if result.EnvironmentFile != layout.EnvironmentFile {
		t.Fatalf("EnvironmentFile = %q, want %q", result.EnvironmentFile, layout.EnvironmentFile)
	}
}

func TestRunMachineSSHBootstrapWritesScopedAgentCLIPathsEnv(t *testing.T) {
	ctx := context.Background()
	machineID := uuid.New()

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

	_, err := runMachineSSHBootstrap(ctx, machineSSHBootstrapDeps{
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
			ID:               machineID,
			Name:             "reverse-01",
			Host:             "10.0.1.11",
			ReachabilityMode: catalogdomain.MachineReachabilityModeReverseConnect,
			ExecutionMode:    catalogdomain.MachineExecutionModeWebsocket,
			ConnectionMode:   catalogdomain.MachineConnectionModeWSReverse,
			AgentCLIPaths: catalogdomain.MachineAgentCLIPaths{
				catalogdomain.AgentProviderAdapterTypeCodexAppServer: "/opt/codex/bin/codex",
				catalogdomain.AgentProviderAdapterTypeGeminiCLI:      "/opt/gemini/bin/gemini",
			},
		},
		Topology: catalogdomain.MachineWebsocketTopologyReverseConnect,
		TokenTTL: 24 * time.Hour,
	})
	if err != nil {
		t.Fatalf("runMachineSSHBootstrap() error = %v", err)
	}

	envBody := envUploadSession.stdin.String()
	if !strings.Contains(envBody, `OPENASE_MACHINE_AGENT_CLI_PATHS_JSON="{\"codex-app-server\":\"/opt/codex/bin/codex\",\"gemini-cli\":\"/opt/gemini/bin/gemini\"}"`) {
		t.Fatalf("env upload missing scoped agent cli paths: %q", envBody)
	}
	serviceBody := serviceUploadSession.stdin.String()
	if strings.Contains(serviceBody, "--agent-cli-path") {
		t.Fatalf("service upload should rely on scoped env paths instead of legacy flag: %q", serviceBody)
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
		ID:               machineID,
		Name:             "reverse-01",
		Host:             "10.0.1.11",
		ReachabilityMode: catalogdomain.MachineReachabilityModeReverseConnect,
		ExecutionMode:    catalogdomain.MachineExecutionModeWebsocket,
		ConnectionMode:   catalogdomain.MachineConnectionModeWSReverse,
		SSHUser:          &sshUser,
		WorkspaceRoot:    &workspaceRoot,
		AgentCLIPath:     &agentCLIPath,
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

func TestRunMachineSSHDiagnosticsReportsListenerEndpointIssue(t *testing.T) {
	ctx := context.Background()
	machineID := uuid.New()
	sshUser := "openase"

	client := &machineSSHTestClient{sessions: []sshinfra.Session{
		&machineSSHTestSession{combinedOutput: "Linux\n/home/remote\nsystemd\n1000\n"},
		&machineSSHTestSession{},
		&machineSSHTestSession{},
		&machineSSHTestSession{combinedOutput: "active\n"},
	}}

	result, err := runMachineSSHDiagnostics(ctx, machineSSHDiagnosticDeps{
		getClient: func(context.Context, catalogdomain.Machine) (sshinfra.Client, error) {
			return client, nil
		},
		probeListener: func(context.Context, catalogdomain.Machine) error {
			return errors.New("listener websocket endpoint unreachable for machine listener-01 at wss://listener.example.com/openase/runtime")
		},
	}, catalogdomain.Machine{
		ID:                 machineID,
		Name:               "listener-01",
		Host:               "10.0.1.21",
		ReachabilityMode:   catalogdomain.MachineReachabilityModeDirectConnect,
		ExecutionMode:      catalogdomain.MachineExecutionModeWebsocket,
		ConnectionMode:     catalogdomain.MachineConnectionModeWSListener,
		SSHUser:            &sshUser,
		AdvertisedEndpoint: stringPtr("wss://listener.example.com/openase/runtime"),
	})
	if err != nil {
		t.Fatalf("runMachineSSHDiagnostics(listener) error = %v", err)
	}

	if result.Topology != catalogdomain.MachineWebsocketTopologyRemoteListener.String() {
		t.Fatalf("Topology = %q, want remote_listener", result.Topology)
	}
	found := false
	for _, issue := range result.Issues {
		if issue.Code == "listener_endpoint_unreachable" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected listener_endpoint_unreachable in %+v", result.Issues)
	}
}

func TestRunMachineSSHDiagnosticsChecksScopedAgentCLIPaths(t *testing.T) {
	ctx := context.Background()
	machineID := uuid.New()
	sshUser := "openase"

	client := &machineSSHTestClient{sessions: []sshinfra.Session{
		&machineSSHTestSession{combinedOutput: "Linux\n/home/remote\nsystemd\n1000\n"},
		&machineSSHTestSession{},
		&machineSSHTestSession{},
		&machineSSHTestSession{combinedOutput: "active\n"},
		&machineSSHTestSession{combinedOutput: ""},
		&machineSSHTestSession{combinedOutput: ""},
	}}

	result, err := runMachineSSHDiagnostics(ctx, machineSSHDiagnosticDeps{
		getClient: func(context.Context, catalogdomain.Machine) (sshinfra.Client, error) {
			return client, nil
		},
	}, catalogdomain.Machine{
		ID:               machineID,
		Name:             "reverse-02",
		Host:             "10.0.1.12",
		ReachabilityMode: catalogdomain.MachineReachabilityModeReverseConnect,
		ExecutionMode:    catalogdomain.MachineExecutionModeWebsocket,
		ConnectionMode:   catalogdomain.MachineConnectionModeWSReverse,
		SSHUser:          &sshUser,
		AgentCLIPath:     stringPtr("/usr/local/bin/legacy"),
		AgentCLIPaths: catalogdomain.MachineAgentCLIPaths{
			catalogdomain.AgentProviderAdapterTypeCodexAppServer: "/opt/codex/bin/codex",
			catalogdomain.AgentProviderAdapterTypeGeminiCLI:      "/opt/gemini/bin/gemini",
		},
		DaemonStatus: catalogdomain.MachineDaemonStatus{
			Registered: true,
		},
	})
	if err != nil {
		t.Fatalf("runMachineSSHDiagnostics() error = %v", err)
	}

	checkNames := make([]string, 0, len(result.Checks))
	for _, check := range result.Checks {
		checkNames = append(checkNames, check.Name)
	}
	joined := strings.Join(checkNames, ",")
	if !strings.Contains(joined, "agent_cli_path_codex_app_server") || !strings.Contains(joined, "agent_cli_path_gemini_cli") {
		t.Fatalf("expected scoped agent cli checks, got %+v", result.Checks)
	}
	if strings.Contains(joined, "agent_cli_path,") || joined == "agent_cli_path" {
		t.Fatalf("expected scoped checks to suppress legacy fallback when agent_cli_paths are present, got %+v", result.Checks)
	}
}

func stringPtr(value string) *string { return &value }

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

func (s *machineSSHTestSession) StartPTY(cmd string, _ int, _ int) error {
	s.startCommand = cmd
	return nil
}

func (s *machineSSHTestSession) Resize(int, int) error { return nil }

func (s *machineSSHTestSession) Signal(string) error { return nil }

func (s *machineSSHTestSession) Wait() error { return nil }

func (s *machineSSHTestSession) Close() error { return nil }

type nopWriteCloser struct {
	io.Writer
}

func (nopWriteCloser) Close() error { return nil }
