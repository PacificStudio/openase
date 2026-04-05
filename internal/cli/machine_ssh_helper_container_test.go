package cli

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	sshinfra "github.com/BetterAndBetterII/openase/internal/infra/ssh"
	"github.com/BetterAndBetterII/openase/internal/testutil/containerharness"
	"github.com/google/uuid"
	gossh "golang.org/x/crypto/ssh"
)

func TestMachineSSHHelperContainerE2E(t *testing.T) {
	containerharness.RequireContainerSuite(t)

	openaseBinary := containerharness.BuiltOpenASEBinary(t)
	privateKeyPath, authorizedKeysPath := writeSSHKeyPair(t)
	hostPort := containerharness.FreeTCPPort(t)
	project := containerharness.NewProject(t, containerharness.Options{
		ProjectName: "ase41-ssh-" + strings.ToLower(strings.ReplaceAll(uuid.NewString(), "-", "")),
		Env: map[string]string{
			"OPENASE_TEST_SSH_AUTHORIZED_KEYS":  authorizedKeysPath,
			"OPENASE_TEST_SSH_HELPER_HOST_PORT": fmt.Sprintf("%d", hostPort),
		},
	})
	project.Up(t, nil, "ssh-helper")
	project.WriteLogs(t, "ssh-helper-compose.log", nil, "ssh-helper")
	containerharness.WaitForTCPPort(t, fmt.Sprintf("127.0.0.1:%d", hostPort), 20*time.Second)

	ctx := context.Background()
	machineID := uuid.New()
	sshUser := "openase"
	workspaceRoot := "/home/openase/workspaces"
	agentCLIPath := "/bin/sh"
	machine := catalogdomain.Machine{
		ID:             machineID,
		Name:           "reverse-ssh-helper",
		Host:           "127.0.0.1",
		Port:           hostPort,
		ConnectionMode: catalogdomain.MachineConnectionModeWSReverse,
		SSHUser:        &sshUser,
		SSHKeyPath:     &privateKeyPath,
		WorkspaceRoot:  &workspaceRoot,
		AgentCLIPath:   &agentCLIPath,
		DaemonStatus: catalogdomain.MachineDaemonStatus{
			Registered: true,
		},
	}

	pool := sshinfra.NewPool(t.TempDir(), sshinfra.WithTimeout(5*time.Second))
	defer func() {
		_ = pool.Close()
	}()

	bootstrapResult, err := runMachineSSHBootstrap(ctx, machineSSHBootstrapDeps{
		getClient: func(ctx context.Context, item catalogdomain.Machine) (sshinfra.Client, error) {
			return pool.Get(ctx, item)
		},
		issueToken: func(context.Context, uuid.UUID, time.Duration, string, string) (machineChannelTokenResponse, error) {
			return machineChannelTokenResponse{
				Token:           "ase_machine_test_container",
				TokenID:         "token-container",
				MachineID:       machineID.String(),
				ControlPlaneURL: "http://127.0.0.1:19836",
			}, nil
		},
		readLocalFile:     os.ReadFile,
		resolveExecutable: func() (string, error) { return openaseBinary, nil },
	}, machineSSHBootstrapInput{
		Machine:           machine,
		TokenTTL:          time.Hour,
		ControlPlaneURL:   "http://127.0.0.1:19836",
		OpenASEBinaryPath: openaseBinary,
		HeartbeatInterval: 15 * time.Second,
	})
	if err != nil {
		t.Fatalf("runMachineSSHBootstrap() error = %v", err)
	}

	layout := buildMachineSSHLayout("/home/openase")
	if bootstrapResult.RemoteBinaryPath != layout.RemoteBinaryPath {
		t.Fatalf("RemoteBinaryPath = %q, want %q", bootstrapResult.RemoteBinaryPath, layout.RemoteBinaryPath)
	}
	if bootstrapResult.EnvironmentFile != layout.EnvironmentFile {
		t.Fatalf("EnvironmentFile = %q, want %q", bootstrapResult.EnvironmentFile, layout.EnvironmentFile)
	}
	if bootstrapResult.TokenID != "token-container" {
		t.Fatalf("TokenID = %q, want token-container", bootstrapResult.TokenID)
	}

	diagnosticsResult, err := runMachineSSHDiagnostics(ctx, machineSSHDiagnosticDeps{
		getClient: func(ctx context.Context, item catalogdomain.Machine) (sshinfra.Client, error) {
			return pool.Get(ctx, item)
		},
	}, machine)
	if err != nil {
		t.Fatalf("runMachineSSHDiagnostics() error = %v", err)
	}
	if len(diagnosticsResult.Issues) != 0 {
		t.Fatalf("Diagnostics issues = %+v, want none", diagnosticsResult.Issues)
	}

	client, err := pool.Get(ctx, machine)
	if err != nil {
		t.Fatalf("pool.Get() error = %v", err)
	}
	remoteState, err := runRemoteSSHCommand(ctx, client, "sh -lc "+sshinfra.ShellQuote(strings.Join([]string{
		"set -eu",
		"test -x " + sshinfra.ShellQuote(layout.RemoteBinaryPath),
		"test -w " + sshinfra.ShellQuote(workspaceRoot),
		"test -f " + sshinfra.ShellQuote(layout.StdoutPath),
		"cat " + sshinfra.ShellQuote(layout.EnvironmentFile),
		"printf '\\n--SERVICE--\\n'",
		"cat " + sshinfra.ShellQuote(layout.ServiceFile),
	}, "\n")))
	if err != nil {
		t.Fatalf("read remote ssh helper state: %v (%s)", err, strings.TrimSpace(remoteState))
	}
	if !strings.Contains(remoteState, `OPENASE_MACHINE_CHANNEL_TOKEN="ase_machine_test_container"`) {
		t.Fatalf("remote env file missing channel token: %s", remoteState)
	}
	if !strings.Contains(remoteState, `"machine-agent" "run"`) {
		t.Fatalf("remote service file missing machine-agent run command: %s", remoteState)
	}
}

func writeSSHKeyPair(t *testing.T) (string, string) {
	t.Helper()

	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate ed25519 key pair: %v", err)
	}

	privatePKCS8, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatalf("marshal private key: %v", err)
	}
	privatePEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privatePKCS8})

	sshPublicKey, err := gossh.NewPublicKey(publicKey)
	if err != nil {
		t.Fatalf("marshal public ssh key: %v", err)
	}

	dir := t.TempDir()
	privateKeyPath := filepath.Join(dir, "id_ed25519")
	authorizedKeysPath := filepath.Join(dir, "authorized_keys")

	if err := os.WriteFile(privateKeyPath, privatePEM, 0o600); err != nil {
		t.Fatalf("write private key: %v", err)
	}
	if err := os.WriteFile(authorizedKeysPath, gossh.MarshalAuthorizedKey(sshPublicKey), 0o600); err != nil {
		t.Fatalf("write authorized_keys: %v", err)
	}

	return privateKeyPath, authorizedKeysPath
}
