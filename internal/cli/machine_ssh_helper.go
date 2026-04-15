package cli

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/BetterAndBetterII/openase/internal/config"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	machinechanneldomain "github.com/BetterAndBetterII/openase/internal/domain/machinechannel"
	sshinfra "github.com/BetterAndBetterII/openase/internal/infra/ssh"
	machinechannelservice "github.com/BetterAndBetterII/openase/internal/machinechannel"
	machinechannelrepo "github.com/BetterAndBetterII/openase/internal/repo/machinechannel"
	"github.com/BetterAndBetterII/openase/internal/runtime/database"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/spf13/cobra"
)

const (
	machineAgentServiceName    = "openase-machine-agent"
	machineListenerServiceName = "openase-machine-listener"
	defaultRemoteOpenASEBin    = ".openase/bin/openase"
)

type cliMachineEnvelope struct {
	Machine cliMachineRecord `json:"machine"`
}

type cliMachineRecord struct {
	ID                 string                      `json:"id"`
	Name               string                      `json:"name"`
	Host               string                      `json:"host"`
	Port               int                         `json:"port"`
	ReachabilityMode   string                      `json:"reachability_mode"`
	ExecutionMode      string                      `json:"execution_mode"`
	SSHUser            *string                     `json:"ssh_user"`
	SSHKeyPath         *string                     `json:"ssh_key_path"`
	AdvertisedEndpoint *string                     `json:"advertised_endpoint"`
	WorkspaceRoot      *string                     `json:"workspace_root"`
	AgentCLIPath       *string                     `json:"agent_cli_path"`
	DaemonStatus       cliMachineDaemonInfo        `json:"daemon_status"`
	ChannelCredential  cliMachineChannelCredential `json:"channel_credential"`
}

type cliMachineDaemonInfo struct {
	Registered bool `json:"registered"`
}

type cliMachineChannelCredential struct {
	Kind    string  `json:"kind"`
	TokenID *string `json:"token_id"`
}

type machineSSHBootstrapInput struct {
	Machine             catalogdomain.Machine
	Topology            catalogdomain.MachineWebsocketTopology
	Token               string
	TokenTTL            time.Duration
	ControlPlaneURL     string
	OpenASEBinaryPath   string
	HeartbeatInterval   time.Duration
	ListenerAddress     string
	ListenerPath        string
	ListenerBearerToken string
}

type machineSSHBootstrapResult struct {
	MachineID        string   `json:"machine_id"`
	MachineName      string   `json:"machine_name"`
	Topology         string   `json:"topology"`
	ServiceManager   string   `json:"service_manager"`
	ServiceName      string   `json:"service_name"`
	ServiceStatus    string   `json:"service_status"`
	ConnectionTarget string   `json:"connection_target"`
	RemoteHome       string   `json:"remote_home"`
	RemoteBinaryPath string   `json:"remote_binary_path"`
	EnvironmentFile  string   `json:"environment_file"`
	ServiceFile      string   `json:"service_file"`
	TokenID          string   `json:"token_id,omitempty"`
	Commands         []string `json:"commands"`
	RetryAdvice      []string `json:"retry_advice,omitempty"`
	RollbackAdvice   []string `json:"rollback_advice,omitempty"`
	Summary          string   `json:"summary"`
}

type machineSSHDiagnosticResult struct {
	MachineID        string                      `json:"machine_id"`
	MachineName      string                      `json:"machine_name"`
	Topology         string                      `json:"topology"`
	ConnectionTarget string                      `json:"connection_target"`
	ServiceManager   string                      `json:"service_manager"`
	RemoteHome       string                      `json:"remote_home"`
	Checks           []machineSSHDiagnosticCheck `json:"checks"`
	Issues           []machineSSHDiagnosticIssue `json:"issues"`
	Summary          string                      `json:"summary"`
}

type machineSSHDiagnosticCheck struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}

type machineSSHDiagnosticIssue struct {
	Code        string `json:"code"`
	Title       string `json:"title"`
	Detail      string `json:"detail"`
	Remediation string `json:"remediation"`
}

type machineSSHPlatformInfo struct {
	OS             string
	RemoteHome     string
	ServiceManager string
	UID            string
}

type machineSSHBootstrapDeps struct {
	getClient         func(context.Context, catalogdomain.Machine) (sshinfra.Client, error)
	issueToken        func(context.Context, uuid.UUID, time.Duration, string, string) (machineChannelTokenResponse, error)
	readLocalFile     func(string) ([]byte, error)
	resolveExecutable func() (string, error)
}

type machineSSHDiagnosticDeps struct {
	getClient     func(context.Context, catalogdomain.Machine) (sshinfra.Client, error)
	probeListener func(context.Context, catalogdomain.Machine) error
}

type machineSSHBootstrapPlan struct {
	Topology         catalogdomain.MachineWebsocketTopology
	ServiceName      string
	ServiceLabel     string
	EnvironmentBody  string
	CommandArgs      []string
	ConnectionTarget string
	TokenID          string
	RetryAdvice      []string
	RollbackAdvice   []string
	Summary          string
}

func newMachineSSHBootstrapCommand(options *rootOptions) *cobra.Command {
	var apiOptions apiCommandOptions
	var topology string
	var channelToken string
	var ttl time.Duration
	var controlPlaneURL string
	var openaseBinaryPath string
	var heartbeatInterval time.Duration
	var listenerAddress string
	var listenerPath string
	var listenerBearerToken string

	command := &cobra.Command{
		Use:   "ssh-bootstrap [machineId]",
		Short: "Use SSH helper access to install or refresh a websocket topology service on the remote machine.",
		Long: strings.TrimSpace(`
Use SSH helper access to install or refresh a websocket topology service on the remote machine.

This command is the supported SSH bootstrap helper for websocket runtime rollout.
The [machineId] argument must be a machine UUID.
It fetches the machine record from the OpenASE API, resolves the target topology
from the stored reachability + execution semantics, uploads the current OpenASE
binary to the remote host, writes the topology environment file, installs a user
service, and starts it.

Normal ticket execution no longer falls back to SSH. Use this helper to repair
reverse-connect daemon registration or to bootstrap a remote websocket listener
when you can still reach the machine over SSH.
`),
		Example: strings.TrimSpace(`
  openase machine ssh-bootstrap $OPENASE_MACHINE_ID
  openase machine ssh-bootstrap 550e8400-e29b-41d4-a716-446655440000 --control-plane-url https://openase.example.com
  openase machine ssh-bootstrap $OPENASE_MACHINE_ID --channel-token ase_machine_xxxxx --control-plane-url https://openase.example.com
`),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			apiContext, err := apiOptionsFromFlags(cmd.Flags()).resolveResource()
			if err != nil {
				return err
			}
			machine, err := fetchCLIMachine(cmd.Context(), apiContext, strings.TrimSpace(args[0]))
			if err != nil {
				return err
			}
			resolvedTopology, err := resolveMachineSSHBootstrapTopology(machine, topology)
			if err != nil {
				return err
			}

			homeDir, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("resolve user home directory: %w", err)
			}
			pool := sshinfra.NewPool(filepath.Join(homeDir, ".openase"))
			defer func() {
				_ = pool.Close()
			}()

			result, err := runMachineSSHBootstrap(cmd.Context(), machineSSHBootstrapDeps{
				getClient: func(ctx context.Context, item catalogdomain.Machine) (sshinfra.Client, error) {
					return pool.Get(ctx, item)
				},
				issueToken: func(ctx context.Context, machineID uuid.UUID, tokenTTL time.Duration, explicitControlPlaneURL string, rawToken string) (machineChannelTokenResponse, error) {
					return issueMachineBootstrapToken(ctx, options, machineID, tokenTTL, explicitControlPlaneURL, rawToken)
				},
				readLocalFile:     os.ReadFile,
				resolveExecutable: os.Executable,
			}, machineSSHBootstrapInput{
				Machine:             machine,
				Topology:            resolvedTopology,
				Token:               strings.TrimSpace(channelToken),
				TokenTTL:            ttl,
				ControlPlaneURL:     strings.TrimSpace(controlPlaneURL),
				OpenASEBinaryPath:   strings.TrimSpace(openaseBinaryPath),
				HeartbeatInterval:   heartbeatInterval,
				ListenerAddress:     strings.TrimSpace(listenerAddress),
				ListenerPath:        strings.TrimSpace(listenerPath),
				ListenerBearerToken: strings.TrimSpace(listenerBearerToken),
			})
			if err != nil {
				return err
			}

			body, err := json.Marshal(result)
			if err != nil {
				return fmt.Errorf("marshal ssh bootstrap result: %w", err)
			}
			return writePrettyJSON(cmd.OutOrStdout(), body)
		},
	}

	command.SetFlagErrorFunc(flagErrorWithNormalize)
	applyCLICommandFlagNormalization(command)
	bindAPICommandFlags(command.Flags(), &apiOptions)
	command.Flags().StringVar(&topology, "topology", "", "Optional topology override: reverse-connect or remote-listener. Defaults to the machine's stored reachability + execution topology.")
	command.Flags().StringVar(&channelToken, "channel-token", "", "Existing machine channel token. When omitted, the helper issues a fresh token from the local control plane config.")
	command.Flags().DurationVar(&ttl, "ttl", 24*time.Hour, "Fresh token lifetime when the helper issues a new token.")
	command.Flags().StringVar(&controlPlaneURL, "control-plane-url", "", "Control-plane base URL override. Required when reusing --channel-token and defaults to the local server URL when issuing a fresh token.")
	command.Flags().StringVar(&openaseBinaryPath, "openase-binary-path", "", "Local OpenASE binary to upload. Defaults to the current executable.")
	command.Flags().DurationVar(&heartbeatInterval, "heartbeat-interval", machinechannelservice.DefaultHeartbeatInterval, "Daemon heartbeat interval written into the remote environment file.")
	command.Flags().StringVar(&listenerAddress, "listener-address", defaultMachineListenerAddress, "Remote websocket listener bind address when installing the remote-listener topology.")
	command.Flags().StringVar(&listenerPath, "listener-path", defaultMachineListenerPath, "Remote websocket listener HTTP path when installing the remote-listener topology.")
	command.Flags().StringVar(&listenerBearerToken, "listener-bearer-token", "", "Optional bearer token override for the remote-listener topology. Defaults to the machine channel credential token when present.")
	return command
}

func newMachineSSHDiagnosticsCommand() *cobra.Command {
	var apiOptions apiCommandOptions

	command := &cobra.Command{
		Use:   "ssh-diagnostics [machineId]",
		Short: "Run SSH helper diagnostics for machine bootstrap and daemon health.",
		Long: strings.TrimSpace(`
Run SSH helper diagnostics for machine bootstrap and daemon health.

This command keeps SSH in the helper lane: it validates SSH access, workspace
permissions, machine-agent binary presence, service health, daemon registration,
and recent logs, then returns actionable issues for common misconfiguration
classes. The [machineId] argument must be a machine UUID.
`),
		Example: strings.TrimSpace(`
  openase machine ssh-diagnostics $OPENASE_MACHINE_ID
  openase machine ssh-diagnostics 550e8400-e29b-41d4-a716-446655440000
`),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			apiContext, err := apiOptionsFromFlags(cmd.Flags()).resolveResource()
			if err != nil {
				return err
			}
			machine, err := fetchCLIMachine(cmd.Context(), apiContext, strings.TrimSpace(args[0]))
			if err != nil {
				return err
			}

			homeDir, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("resolve user home directory: %w", err)
			}
			pool := sshinfra.NewPool(filepath.Join(homeDir, ".openase"))
			defer func() {
				_ = pool.Close()
			}()

			result, err := runMachineSSHDiagnostics(cmd.Context(), machineSSHDiagnosticDeps{
				getClient: func(ctx context.Context, item catalogdomain.Machine) (sshinfra.Client, error) {
					return pool.Get(ctx, item)
				},
				probeListener: probeMachineSSHListenerEndpoint,
			}, machine)
			if err != nil {
				return err
			}

			body, err := json.Marshal(result)
			if err != nil {
				return fmt.Errorf("marshal ssh diagnostics result: %w", err)
			}
			return writePrettyJSON(cmd.OutOrStdout(), body)
		},
	}

	command.SetFlagErrorFunc(flagErrorWithNormalize)
	applyCLICommandFlagNormalization(command)
	bindAPICommandFlags(command.Flags(), &apiOptions)
	return command
}

func resolveMachineSSHBootstrapTopology(
	machine catalogdomain.Machine,
	raw string,
) (catalogdomain.MachineWebsocketTopology, error) {
	storedTopology, err := machine.WebsocketTopology()
	if err != nil {
		return "", err
	}
	if storedTopology == catalogdomain.MachineWebsocketTopologyLocalProcess {
		return "", fmt.Errorf("local machines do not use ssh bootstrap helper")
	}

	explicit := strings.ToLower(strings.TrimSpace(raw))
	if explicit == "" {
		return storedTopology, nil
	}

	var requested catalogdomain.MachineWebsocketTopology
	switch explicit {
	case "reverse-connect", "reverse_connect":
		requested = catalogdomain.MachineWebsocketTopologyReverseConnect
	case "remote-listener", "remote_listener":
		requested = catalogdomain.MachineWebsocketTopologyRemoteListener
	default:
		return "", fmt.Errorf("topology must be one of reverse-connect or remote-listener")
	}
	if requested != storedTopology {
		return "", fmt.Errorf(
			"topology %q does not match machine %s reachability_mode %q and execution_mode %q",
			requested,
			machine.Name,
			machine.ReachabilityMode,
			machine.ExecutionMode,
		)
	}
	return requested, nil
}

func buildMachineSSHBootstrapPlan(
	ctx context.Context,
	deps machineSSHBootstrapDeps,
	input machineSSHBootstrapInput,
) (machineSSHBootstrapPlan, error) {
	switch input.Topology {
	case catalogdomain.MachineWebsocketTopologyReverseConnect:
		tokenResp, err := deps.issueToken(ctx, input.Machine.ID, input.TokenTTL, input.ControlPlaneURL, input.Token)
		if err != nil {
			return machineSSHBootstrapPlan{}, err
		}
		return machineSSHBootstrapPlan{
			Topology:         input.Topology,
			ServiceName:      machineAgentServiceName,
			ServiceLabel:     "OpenASE reverse-connect machine-agent",
			EnvironmentBody:  buildMachineSSHEnvironmentFile(tokenResp, input.HeartbeatInterval),
			CommandArgs:      []string{"machine-agent", "run"},
			ConnectionTarget: tokenResp.ControlPlaneURL,
			TokenID:          tokenResp.TokenID,
			RetryAdvice: []string{
				"Run `openase machine ssh-diagnostics <machine-id>` to inspect service output, daemon registration, and authentication failures.",
				"Confirm the control-plane URL and machine channel token are still valid before rerunning bootstrap.",
			},
			RollbackAdvice: []string{
				"Disable the remote user service if you need to stop reverse-connect retries immediately.",
				"Remove the remote environment and service files only after capturing diagnostics output for auditability.",
			},
			Summary: fmt.Sprintf(
				"SSH bootstrap refreshed the reverse-connect service for machine %s.",
				input.Machine.Name,
			),
		}, nil
	case catalogdomain.MachineWebsocketTopologyRemoteListener:
		listenerAddress := firstNonEmpty(input.ListenerAddress, defaultMachineListenerAddress)
		listenerPath := normalizeMachineListenerPath(input.ListenerPath)
		listenerToken := firstNonEmpty(input.ListenerBearerToken, strings.TrimSpace(pointerString(input.Machine.ChannelCredential.TokenID)))
		values := map[string]string{
			envMachineListenerAddress: listenerAddress,
			envMachineListenerPath:    listenerPath,
		}
		if listenerToken != "" {
			values[envMachineListenerBearerToken] = listenerToken
		}
		return machineSSHBootstrapPlan{
			Topology:     input.Topology,
			ServiceName:  machineListenerServiceName,
			ServiceLabel: "OpenASE remote websocket listener",
			EnvironmentBody: buildMachineSSHKeyValueEnvironmentFile(values, []string{
				envMachineListenerAddress,
				envMachineListenerPath,
				envMachineListenerBearerToken,
			}),
			CommandArgs:      []string{"machine-agent", "listen"},
			ConnectionTarget: firstNonEmpty(strings.TrimSpace(pointerString(input.Machine.AdvertisedEndpoint)), listenerAddress+listenerPath),
			RetryAdvice: []string{
				"Confirm the advertised websocket endpoint and any firewall or proxy rules before rerunning bootstrap.",
				"Use `openase machine ssh-diagnostics <machine-id>` to distinguish service-start issues from control-plane reachability failures.",
			},
			RollbackAdvice: []string{
				"Disable the remote websocket listener service if you need to stop inbound websocket traffic immediately.",
				"Keep the uploaded binary and logs in place until the listener reachability issue is fully diagnosed.",
			},
			Summary: fmt.Sprintf(
				"SSH bootstrap refreshed the remote-listener service for machine %s.",
				input.Machine.Name,
			),
		}, nil
	default:
		return machineSSHBootstrapPlan{}, fmt.Errorf("unsupported ssh bootstrap topology %q", input.Topology)
	}
}

func normalizeMachineListenerPath(raw string) string {
	path := strings.TrimSpace(raw)
	if path == "" {
		path = defaultMachineListenerPath
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + strings.TrimPrefix(path, "/")
	}
	return path
}

func machineSSHServiceNameForTopology(topology catalogdomain.MachineWebsocketTopology) string {
	switch topology {
	case catalogdomain.MachineWebsocketTopologyRemoteListener:
		return machineListenerServiceName
	default:
		return machineAgentServiceName
	}
}

func machineSSHConnectionTarget(machine catalogdomain.Machine, topology catalogdomain.MachineWebsocketTopology) string {
	switch topology {
	case catalogdomain.MachineWebsocketTopologyReverseConnect:
		return "reverse websocket control-plane session"
	case catalogdomain.MachineWebsocketTopologyRemoteListener:
		return strings.TrimSpace(pointerString(machine.AdvertisedEndpoint))
	default:
		return ""
	}
}

func runMachineSSHBootstrap(
	ctx context.Context,
	deps machineSSHBootstrapDeps,
	input machineSSHBootstrapInput,
) (machineSSHBootstrapResult, error) {
	if deps.getClient == nil {
		return machineSSHBootstrapResult{}, fmt.Errorf("ssh bootstrap client is unavailable")
	}
	if deps.issueToken == nil {
		return machineSSHBootstrapResult{}, fmt.Errorf("ssh bootstrap token issuer is unavailable")
	}
	if deps.readLocalFile == nil {
		return machineSSHBootstrapResult{}, fmt.Errorf("ssh bootstrap file reader is unavailable")
	}
	if deps.resolveExecutable == nil {
		return machineSSHBootstrapResult{}, fmt.Errorf("ssh bootstrap executable resolver is unavailable")
	}

	if input.Machine.ID == uuid.Nil {
		return machineSSHBootstrapResult{}, fmt.Errorf("machine id must not be empty")
	}
	if input.Machine.Host == catalogdomain.LocalMachineHost {
		return machineSSHBootstrapResult{}, fmt.Errorf("local machines do not use ssh bootstrap helper")
	}

	openaseBinaryPath := strings.TrimSpace(input.OpenASEBinaryPath)
	if openaseBinaryPath == "" {
		resolvedPath, resolveErr := deps.resolveExecutable()
		if resolveErr != nil {
			return machineSSHBootstrapResult{}, fmt.Errorf("resolve openase binary: %w", resolveErr)
		}
		openaseBinaryPath = resolvedPath
	}
	openaseBinaryPath = filepath.Clean(openaseBinaryPath)
	binaryBytes, err := deps.readLocalFile(openaseBinaryPath)
	if err != nil {
		return machineSSHBootstrapResult{}, fmt.Errorf("read openase binary %s: %w", openaseBinaryPath, err)
	}

	client, err := deps.getClient(ctx, input.Machine)
	if err != nil {
		return machineSSHBootstrapResult{}, err
	}
	defer func() {
		_ = client.Close()
	}()

	platform, err := detectMachineSSHPlatform(ctx, client)
	if err != nil {
		return machineSSHBootstrapResult{}, err
	}
	if platform.ServiceManager == "unknown" {
		return machineSSHBootstrapResult{}, fmt.Errorf("remote host %s has no supported user service manager; expected systemd or launchd", input.Machine.Name)
	}

	plan, err := buildMachineSSHBootstrapPlan(ctx, deps, input)
	if err != nil {
		return machineSSHBootstrapResult{}, err
	}

	layout := buildMachineSSHLayout(platform.RemoteHome, plan.ServiceName)
	commandArgs := append([]string(nil), plan.CommandArgs...)
	if plan.Topology == catalogdomain.MachineWebsocketTopologyReverseConnect {
		commandArgs = append(commandArgs, "--openase-binary-path", layout.RemoteBinaryPath)
		if agentCLIPath := strings.TrimSpace(pointerString(input.Machine.AgentCLIPath)); agentCLIPath != "" {
			commandArgs = append(commandArgs, "--agent-cli-path", agentCLIPath)
		}
		if input.HeartbeatInterval > 0 {
			commandArgs = append(commandArgs, "--heartbeat-interval", input.HeartbeatInterval.String())
		}
	}
	serviceFilePath, serviceFile, restartCommand := buildMachineSSHServiceFile(platform, layout, machineSSHServiceInstallInput{
		ServiceName:       plan.ServiceName,
		ServiceLabel:      plan.ServiceLabel,
		BinaryPath:        layout.RemoteBinaryPath,
		EnvironmentFile:   layout.EnvironmentFile,
		WorkingDirectory:  layout.WorkDir,
		StdoutPath:        layout.StdoutPath,
		StderrPath:        layout.StderrPath,
		AgentCLIPath:      strings.TrimSpace(pointerString(input.Machine.AgentCLIPath)),
		HeartbeatInterval: input.HeartbeatInterval,
		UID:               platform.UID,
	}, commandArgs)

	if err := uploadRemoteFile(client, layout.RemoteBinaryPath, binaryBytes, 0o755); err != nil {
		return machineSSHBootstrapResult{}, err
	}
	if err := uploadRemoteFile(client, layout.EnvironmentFile, []byte(plan.EnvironmentBody), 0o600); err != nil {
		return machineSSHBootstrapResult{}, err
	}
	if err := uploadRemoteFile(client, serviceFilePath, []byte(serviceFile), 0o644); err != nil {
		return machineSSHBootstrapResult{}, err
	}

	commandOutput, err := runRemoteSSHCommand(ctx, client, restartCommand)
	if err != nil {
		return machineSSHBootstrapResult{}, fmt.Errorf("activate remote machine-agent service: %w: %s", err, strings.TrimSpace(commandOutput))
	}

	return machineSSHBootstrapResult{
		MachineID:        input.Machine.ID.String(),
		MachineName:      input.Machine.Name,
		Topology:         plan.Topology.String(),
		ServiceManager:   platform.ServiceManager,
		ServiceName:      plan.ServiceName,
		ServiceStatus:    firstNonEmpty(strings.TrimSpace(commandOutput), "active"),
		ConnectionTarget: plan.ConnectionTarget,
		RemoteHome:       platform.RemoteHome,
		RemoteBinaryPath: layout.RemoteBinaryPath,
		EnvironmentFile:  layout.EnvironmentFile,
		ServiceFile:      serviceFilePath,
		TokenID:          plan.TokenID,
		Commands:         append([]string{"upload openase binary", "write topology environment", "install user service"}, strings.TrimSpace(restartCommand)),
		RetryAdvice:      append([]string(nil), plan.RetryAdvice...),
		RollbackAdvice:   append([]string(nil), plan.RollbackAdvice...),
		Summary:          plan.Summary,
	}, nil
}

func runMachineSSHDiagnostics(
	ctx context.Context,
	deps machineSSHDiagnosticDeps,
	machine catalogdomain.Machine,
) (machineSSHDiagnosticResult, error) {
	if deps.getClient == nil {
		return machineSSHDiagnosticResult{}, fmt.Errorf("ssh diagnostics client is unavailable")
	}
	if machine.ID == uuid.Nil {
		return machineSSHDiagnosticResult{}, fmt.Errorf("machine id must not be empty")
	}
	if machine.Host == catalogdomain.LocalMachineHost {
		return machineSSHDiagnosticResult{}, fmt.Errorf("local machines do not use ssh diagnostics")
	}
	topology, err := machine.WebsocketTopology()
	if err != nil {
		return machineSSHDiagnosticResult{}, err
	}

	client, err := deps.getClient(ctx, machine)
	if err != nil {
		return machineSSHDiagnosticResult{}, err
	}
	defer func() {
		_ = client.Close()
	}()

	platform, err := detectMachineSSHPlatform(ctx, client)
	if err != nil {
		return machineSSHDiagnosticResult{}, err
	}
	layout := buildMachineSSHLayout(platform.RemoteHome, machineSSHServiceNameForTopology(topology))

	checks := make([]machineSSHDiagnosticCheck, 0, 6)
	issues := make([]machineSSHDiagnosticIssue, 0, 4)
	checks = append(checks, machineSSHDiagnosticCheck{
		Name:   "ssh_connectivity",
		Status: "pass",
		Detail: fmt.Sprintf("SSH helper connected as %s on %s.", strings.TrimSpace(pointerString(machine.SSHUser)), machine.Host),
	})

	if workspaceRoot := strings.TrimSpace(pointerString(machine.WorkspaceRoot)); workspaceRoot != "" {
		workspaceCmd := "sh -lc " + sshinfra.ShellQuote("mkdir -p "+sshinfra.ShellQuote(workspaceRoot)+" && test -w "+sshinfra.ShellQuote(workspaceRoot))
		if _, checkErr := runRemoteSSHCommand(ctx, client, workspaceCmd); checkErr != nil {
			checks = append(checks, machineSSHDiagnosticCheck{Name: "workspace_root", Status: "fail", Detail: checkErr.Error()})
			issues = append(issues, machineSSHDiagnosticIssue{
				Code:        "workspace_not_writable",
				Title:       "Workspace root is not writable",
				Detail:      fmt.Sprintf("SSH diagnostics could not create or write %s.", workspaceRoot),
				Remediation: "Fix remote filesystem ownership or update the machine workspace root to a writable directory.",
			})
		} else {
			checks = append(checks, machineSSHDiagnosticCheck{Name: "workspace_root", Status: "pass", Detail: workspaceRoot + " is writable."})
		}
	}

	binaryCommand := "sh -lc " + sshinfra.ShellQuote("test -x "+sshinfra.ShellQuote(layout.RemoteBinaryPath))
	if _, checkErr := runRemoteSSHCommand(ctx, client, binaryCommand); checkErr != nil {
		checks = append(checks, machineSSHDiagnosticCheck{Name: "machine_agent_binary", Status: "fail", Detail: checkErr.Error()})
		issues = append(issues, machineSSHDiagnosticIssue{
			Code:        "machine_agent_binary_missing",
			Title:       "Bootstrap binary is missing",
			Detail:      fmt.Sprintf("Expected an executable OpenASE binary at %s.", layout.RemoteBinaryPath),
			Remediation: "Run `openase machine ssh-bootstrap <machine-id>` to upload the current OpenASE binary and reinstall the machine-agent service.",
		})
	} else {
		checks = append(checks, machineSSHDiagnosticCheck{Name: "machine_agent_binary", Status: "pass", Detail: layout.RemoteBinaryPath + " is executable."})
	}

	agentCLIPath := strings.TrimSpace(pointerString(machine.AgentCLIPath))
	if agentCLIPath != "" {
		agentCLICmd := "sh -lc " + sshinfra.ShellQuote("test -x "+sshinfra.ShellQuote(agentCLIPath))
		if _, checkErr := runRemoteSSHCommand(ctx, client, agentCLICmd); checkErr != nil {
			checks = append(checks, machineSSHDiagnosticCheck{Name: "agent_cli_path", Status: "fail", Detail: checkErr.Error()})
			issues = append(issues, machineSSHDiagnosticIssue{
				Code:        "agent_cli_missing",
				Title:       "Configured agent CLI is missing",
				Detail:      fmt.Sprintf("The machine record points to %s, but SSH diagnostics could not execute it.", agentCLIPath),
				Remediation: "Install the preferred agent CLI on the remote host or update the machine's agent CLI path.",
			})
		} else {
			checks = append(checks, machineSSHDiagnosticCheck{Name: "agent_cli_path", Status: "pass", Detail: agentCLIPath + " is executable."})
		}
	}

	serviceOutput, serviceErr := runRemoteSSHCommand(ctx, client, buildMachineSSHDiagnosticCommand(platform, layout))
	serviceDetail := strings.TrimSpace(serviceOutput)
	if serviceErr != nil {
		serviceDetail = strings.TrimSpace(serviceErr.Error() + ": " + serviceDetail)
		checks = append(checks, machineSSHDiagnosticCheck{Name: "topology_service", Status: "fail", Detail: serviceDetail})
		issues = append(issues, machineSSHDiagnosticIssue{
			Code:        "topology_service_unhealthy",
			Title:       "Installed topology service is not healthy",
			Detail:      serviceDetail,
			Remediation: "Restart the installed topology service after fixing the underlying configuration or connectivity problem.",
		})
	} else {
		checks = append(checks, machineSSHDiagnosticCheck{Name: "topology_service", Status: "pass", Detail: firstNonEmpty(serviceDetail, "service check succeeded")})
	}

	lowerServiceOutput := strings.ToLower(serviceOutput)
	switch topology {
	case catalogdomain.MachineWebsocketTopologyReverseConnect:
		if !machine.DaemonStatus.Registered {
			checks = append(checks, machineSSHDiagnosticCheck{Name: "daemon_registration", Status: "fail", Detail: "Control plane does not currently see an active machine-agent registration."})
			issues = append(issues, machineSSHDiagnosticIssue{
				Code:        "daemon_not_registered",
				Title:       "Machine-agent is not registered",
				Detail:      "The machine record still shows daemon_status.registered=false.",
				Remediation: "Confirm the service started successfully, then inspect recent logs for authentication, URL, or network errors.",
			})
		} else {
			checks = append(checks, machineSSHDiagnosticCheck{Name: "daemon_registration", Status: "pass", Detail: "Control plane reports an active machine-agent registration."})
		}
		if strings.Contains(lowerServiceOutput, "auth_failed") || strings.Contains(lowerServiceOutput, "authentication failed") {
			issues = append(issues, machineSSHDiagnosticIssue{
				Code:        "machine_agent_auth_failed",
				Title:       "Machine-agent authentication failed",
				Detail:      "Recent service output includes machine channel authentication failures.",
				Remediation: "Issue a fresh channel token, verify the machine is stored as reverse_connect + websocket, and rerun ssh bootstrap.",
			})
		}
	case catalogdomain.MachineWebsocketTopologyRemoteListener:
		if endpoint := strings.TrimSpace(pointerString(machine.AdvertisedEndpoint)); endpoint == "" {
			checks = append(checks, machineSSHDiagnosticCheck{Name: "listener_endpoint", Status: "fail", Detail: "Machine record does not expose an advertised websocket listener endpoint."})
			issues = append(issues, machineSSHDiagnosticIssue{
				Code:        "listener_endpoint_missing",
				Title:       "Listener endpoint is missing",
				Detail:      "The machine record does not currently expose an advertised websocket endpoint for the remote listener topology.",
				Remediation: "Save a valid advertised_endpoint and confirm the control plane can reach it before rerunning diagnostics.",
			})
		} else {
			if deps.probeListener != nil {
				if probeErr := deps.probeListener(ctx, machine); probeErr != nil {
					checks = append(checks, machineSSHDiagnosticCheck{Name: "listener_endpoint", Status: "fail", Detail: probeErr.Error()})
					issues = append(issues, machineSSHDiagnosticIssue{
						Code:        "listener_endpoint_unreachable",
						Title:       "Listener endpoint is not reachable",
						Detail:      probeErr.Error(),
						Remediation: "Fix advertised endpoint DNS/TLS/networking or the listener bearer token before retrying control-plane checks.",
					})
				} else {
					checks = append(checks, machineSSHDiagnosticCheck{Name: "listener_endpoint", Status: "pass", Detail: endpoint})
				}
			} else {
				checks = append(checks, machineSSHDiagnosticCheck{Name: "listener_endpoint", Status: "pass", Detail: endpoint})
			}
		}
	}

	summary := fmt.Sprintf("SSH diagnostics passed for machine %s.", machine.Name)
	if len(issues) > 0 {
		summary = fmt.Sprintf("SSH diagnostics found %d issue(s) for machine %s.", len(issues), machine.Name)
	}
	return machineSSHDiagnosticResult{
		MachineID:        machine.ID.String(),
		MachineName:      machine.Name,
		Topology:         topology.String(),
		ConnectionTarget: machineSSHConnectionTarget(machine, topology),
		ServiceManager:   platform.ServiceManager,
		RemoteHome:       platform.RemoteHome,
		Checks:           checks,
		Issues:           issues,
		Summary:          summary,
	}, nil
}

func fetchCLIMachine(ctx context.Context, apiContext apiCommandContext, machineID string) (catalogdomain.Machine, error) {
	response, err := apiContext.do(ctx, apiCommandDeps{httpClient: http.DefaultClient}, apiRequest{
		Method: http.MethodGet,
		Path:   "machines/" + urlPathEscape(strings.TrimSpace(machineID)),
	})
	if err != nil {
		return catalogdomain.Machine{}, err
	}

	var payload cliMachineEnvelope
	if err := json.Unmarshal(response.Body, &payload); err != nil {
		return catalogdomain.Machine{}, fmt.Errorf("decode machine response: %w", err)
	}
	parsedID, err := uuid.Parse(strings.TrimSpace(payload.Machine.ID))
	if err != nil {
		return catalogdomain.Machine{}, fmt.Errorf("decode machine response: invalid machine id %q", payload.Machine.ID)
	}
	reachabilityMode, err := catalogdomain.ParseStoredMachineReachabilityMode(payload.Machine.ReachabilityMode, payload.Machine.Host)
	if err != nil {
		return catalogdomain.Machine{}, err
	}
	executionMode, err := catalogdomain.ParseStoredMachineExecutionMode(payload.Machine.ExecutionMode, payload.Machine.Host)
	if err != nil {
		return catalogdomain.Machine{}, err
	}
	connectionMode, _, _, err := catalogdomain.ResolveMachineConnectionMode(
		"",
		payload.Machine.ReachabilityMode,
		payload.Machine.ExecutionMode,
		payload.Machine.Host,
	)
	if err != nil {
		return catalogdomain.Machine{}, err
	}
	channelCredentialKind, err := catalogdomain.ParseStoredMachineChannelCredentialKind(payload.Machine.ChannelCredential.Kind)
	if err != nil {
		return catalogdomain.Machine{}, err
	}

	return catalogdomain.Machine{
		ID:                 parsedID,
		Name:               strings.TrimSpace(payload.Machine.Name),
		Host:               strings.TrimSpace(payload.Machine.Host),
		Port:               payload.Machine.Port,
		ReachabilityMode:   reachabilityMode,
		ExecutionMode:      executionMode,
		ConnectionMode:     connectionMode,
		SSHUser:            optionalTrimmedString(payload.Machine.SSHUser),
		SSHKeyPath:         optionalTrimmedString(payload.Machine.SSHKeyPath),
		AdvertisedEndpoint: optionalTrimmedString(payload.Machine.AdvertisedEndpoint),
		WorkspaceRoot:      optionalTrimmedString(payload.Machine.WorkspaceRoot),
		AgentCLIPath:       optionalTrimmedString(payload.Machine.AgentCLIPath),
		DaemonStatus: catalogdomain.MachineDaemonStatus{
			Registered: payload.Machine.DaemonStatus.Registered,
		},
		ChannelCredential: catalogdomain.MachineChannelCredential{
			Kind:    channelCredentialKind,
			TokenID: optionalTrimmedString(payload.Machine.ChannelCredential.TokenID),
		},
	}, nil
}

func issueMachineBootstrapToken(
	ctx context.Context,
	options *rootOptions,
	machineID uuid.UUID,
	tokenTTL time.Duration,
	explicitControlPlaneURL string,
	rawToken string,
) (machineChannelTokenResponse, error) {
	trimmedToken := strings.TrimSpace(rawToken)
	if trimmedToken != "" {
		controlPlaneURL := strings.TrimSpace(explicitControlPlaneURL)
		if controlPlaneURL == "" {
			return machineChannelTokenResponse{}, fmt.Errorf("control-plane-url is required when reusing an existing --token")
		}
		return machineChannelTokenResponse{
			Token:           trimmedToken,
			MachineID:       machineID.String(),
			ControlPlaneURL: controlPlaneURL,
			Environment: map[string]string{
				machinechanneldomain.EnvMachineID:              machineID.String(),
				machinechanneldomain.EnvMachineChannelToken:    trimmedToken,
				machinechanneldomain.EnvMachineControlPlaneURL: controlPlaneURL,
			},
		}, nil
	}

	cfg, err := config.Load(config.LoadOptions{ConfigFile: options.configFile})
	if err != nil {
		logConfigLoadFailure(options.configFile, nil, err)
		return machineChannelTokenResponse{}, err
	}
	client, err := database.Open(ctx, cfg.Database.DSN)
	if err != nil {
		return machineChannelTokenResponse{}, err
	}
	defer func() {
		_ = client.Close()
	}()

	issued, err := machinechannelservice.NewService(machinechannelrepo.NewEntRepository(client)).IssueToken(ctx, machinechanneldomain.IssueInput{
		MachineID: machineID,
		TTL:       tokenTTL,
	})
	if err != nil {
		return machineChannelTokenResponse{}, err
	}
	resolvedControlPlaneURL, err := resolveControlPlaneURL(cfg, explicitControlPlaneURL)
	if err != nil {
		return machineChannelTokenResponse{}, err
	}
	return machineChannelTokenResponse{
		Token:           issued.Token,
		TokenID:         issued.TokenID.String(),
		MachineID:       issued.MachineID.String(),
		ExpiresAt:       issued.ExpiresAt.UTC().Format(time.RFC3339),
		ControlPlaneURL: resolvedControlPlaneURL,
		Environment: map[string]string{
			machinechanneldomain.EnvMachineID:              issued.MachineID.String(),
			machinechanneldomain.EnvMachineChannelToken:    issued.Token,
			machinechanneldomain.EnvMachineControlPlaneURL: resolvedControlPlaneURL,
		},
	}, nil
}

func detectMachineSSHPlatform(ctx context.Context, client sshinfra.Client) (machineSSHPlatformInfo, error) {
	output, err := runRemoteSSHCommand(ctx, client, "sh -lc "+sshinfra.ShellQuote("uname -s\nprintf '%s\n' \"$HOME\"\nif command -v systemctl >/dev/null 2>&1; then printf 'systemd\n'; elif command -v launchctl >/dev/null 2>&1; then printf 'launchd\n'; else printf 'unknown\n'; fi\nid -u"))
	if err != nil {
		return machineSSHPlatformInfo{}, fmt.Errorf("detect remote ssh platform: %w", err)
	}
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 4 {
		return machineSSHPlatformInfo{}, fmt.Errorf("detect remote ssh platform: unexpected output %q", strings.TrimSpace(output))
	}
	return machineSSHPlatformInfo{
		OS:             strings.TrimSpace(lines[0]),
		RemoteHome:     strings.TrimSpace(lines[1]),
		ServiceManager: strings.TrimSpace(lines[2]),
		UID:            strings.TrimSpace(lines[3]),
	}, nil
}

type machineSSHLayout struct {
	BaseDir          string
	ServiceName      string
	RemoteBinaryPath string
	EnvironmentFile  string
	ServiceFile      string
	WorkDir          string
	StdoutPath       string
	StderrPath       string
}

func buildMachineSSHLayout(remoteHome string, serviceName string) machineSSHLayout {
	baseDir := filepath.Join(remoteHome, ".openase")
	return machineSSHLayout{
		BaseDir:          baseDir,
		ServiceName:      serviceName,
		RemoteBinaryPath: filepath.Join(remoteHome, defaultRemoteOpenASEBin),
		EnvironmentFile:  filepath.Join(baseDir, serviceName, serviceName+".env"),
		ServiceFile:      filepath.Join(remoteHome, ".config", "systemd", "user", serviceName+".service"),
		WorkDir:          filepath.Join(baseDir, serviceName),
		StdoutPath:       filepath.Join(baseDir, "logs", serviceName+".stdout.log"),
		StderrPath:       filepath.Join(baseDir, "logs", serviceName+".stderr.log"),
	}
}

type machineSSHServiceInstallInput struct {
	ServiceName       string
	ServiceLabel      string
	BinaryPath        string
	EnvironmentFile   string
	WorkingDirectory  string
	StdoutPath        string
	StderrPath        string
	AgentCLIPath      string
	HeartbeatInterval time.Duration
	UID               string
}

func buildMachineSSHEnvironmentFile(response machineChannelTokenResponse, heartbeatInterval time.Duration) string {
	values := map[string]string{
		machinechanneldomain.EnvMachineID:              response.MachineID,
		machinechanneldomain.EnvMachineChannelToken:    response.Token,
		machinechanneldomain.EnvMachineControlPlaneURL: response.ControlPlaneURL,
	}
	if heartbeatInterval > 0 {
		values[machinechanneldomain.EnvMachineHeartbeatInterval] = heartbeatInterval.String()
	}

	order := []string{
		machinechanneldomain.EnvMachineID,
		machinechanneldomain.EnvMachineChannelToken,
		machinechanneldomain.EnvMachineControlPlaneURL,
		machinechanneldomain.EnvMachineHeartbeatInterval,
	}
	return buildMachineSSHKeyValueEnvironmentFile(values, order)
}

func buildMachineSSHKeyValueEnvironmentFile(values map[string]string, order []string) string {
	var builder strings.Builder
	for _, key := range order {
		value, ok := values[key]
		if !ok || strings.TrimSpace(value) == "" {
			continue
		}
		builder.WriteString(key)
		builder.WriteString("=")
		builder.WriteString(strconv.Quote(value))
		builder.WriteString("\n")
	}
	return builder.String()
}

func buildMachineSSHServiceFile(
	platform machineSSHPlatformInfo,
	layout machineSSHLayout,
	input machineSSHServiceInstallInput,
	args []string,
) (string, string, string) {
	switch platform.ServiceManager {
	case "launchd":
		servicePath := filepath.Join(platform.RemoteHome, "Library", "LaunchAgents", "com."+input.ServiceName+".plist")
		target := buildLaunchdBootstrapTarget(platform.UID)
		return servicePath, buildMachineAgentLaunchdPlist("com."+input.ServiceName, input, args), "sh -lc " + sshinfra.ShellQuote("set -eu\nlaunchctl bootout "+sshinfra.ShellQuote(target)+" >/dev/null 2>&1 || true\nlaunchctl bootstrap "+sshinfra.ShellQuote(target)+" "+sshinfra.ShellQuote(servicePath)+"\nlaunchctl enable "+sshinfra.ShellQuote(target+"/com."+input.ServiceName)+" >/dev/null 2>&1 || true\nlaunchctl kickstart -k "+sshinfra.ShellQuote(target+"/com."+input.ServiceName)+"\nlaunchctl print "+sshinfra.ShellQuote(target+"/com."+input.ServiceName))
	default:
		return layout.ServiceFile, buildMachineAgentSystemdUnit(input, args), "sh -lc " + sshinfra.ShellQuote("set -eu\nsystemctl --user daemon-reload\nsystemctl --user enable "+sshinfra.ShellQuote(input.ServiceName+".service")+" >/dev/null\nsystemctl --user restart "+sshinfra.ShellQuote(input.ServiceName+".service")+"\nsystemctl --user is-active "+sshinfra.ShellQuote(input.ServiceName+".service"))
	}
}

func buildMachineSSHDiagnosticCommand(platform machineSSHPlatformInfo, layout machineSSHLayout) string {
	switch platform.ServiceManager {
	case "launchd":
		target := buildLaunchdBootstrapTarget(platform.UID)
		return "sh -lc " + sshinfra.ShellQuote("launchctl print "+sshinfra.ShellQuote(target+"/com."+layout.ServiceName)+" 2>&1 || true\ntail -n 20 "+sshinfra.ShellQuote(layout.StdoutPath)+" "+sshinfra.ShellQuote(layout.StderrPath)+" 2>&1 || true")
	default:
		return "sh -lc " + sshinfra.ShellQuote("systemctl --user show -p ActiveState -p SubState "+sshinfra.ShellQuote(layout.ServiceName+".service")+" 2>&1 || true\njournalctl --user -u "+sshinfra.ShellQuote(layout.ServiceName+".service")+" -n 20 --no-pager 2>&1 || true")
	}
}

func buildMachineAgentSystemdUnit(input machineSSHServiceInstallInput, args []string) string {
	var builder strings.Builder
	builder.WriteString("[Unit]\n")
	builder.WriteString("Description=" + firstNonEmpty(input.ServiceLabel, "OpenASE machine service") + "\n")
	builder.WriteString("After=network.target\n\n")
	builder.WriteString("[Service]\n")
	builder.WriteString("Type=simple\n")
	builder.WriteString("ExecStart=")
	builder.WriteString(strconv.Quote(input.BinaryPath))
	for _, arg := range args {
		builder.WriteString(" ")
		builder.WriteString(strconv.Quote(arg))
	}
	builder.WriteString("\n")
	builder.WriteString("EnvironmentFile=-" + input.EnvironmentFile + "\n")
	builder.WriteString("WorkingDirectory=" + input.WorkingDirectory + "\n")
	builder.WriteString("Restart=on-failure\n")
	builder.WriteString("RestartSec=3\n")
	builder.WriteString("StandardOutput=append:" + input.StdoutPath + "\n")
	builder.WriteString("StandardError=append:" + input.StderrPath + "\n\n")
	builder.WriteString("[Install]\n")
	builder.WriteString("WantedBy=default.target\n")
	return builder.String()
}

func buildMachineAgentLaunchdPlist(label string, input machineSSHServiceInstallInput, args []string) string {
	commandParts := make([]string, 0, 3+len(args))
	commandParts = append(commandParts, ". "+sshinfra.ShellQuote(input.EnvironmentFile)+" 2>/dev/null || true;", "exec", sshinfra.ShellQuote(input.BinaryPath))
	for _, arg := range args {
		commandParts = append(commandParts, sshinfra.ShellQuote(arg))
	}

	var builder strings.Builder
	builder.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	builder.WriteString("<!DOCTYPE plist PUBLIC \"-//Apple//DTD PLIST 1.0//EN\" \"http://www.apple.com/DTDs/PropertyList-1.0.dtd\">\n")
	builder.WriteString("<plist version=\"1.0\">\n<dict>\n")
	builder.WriteString(plistKeyValue("Label", label))
	builder.WriteString("  <key>ProgramArguments</key>\n  <array>\n")
	builder.WriteString(plistString("/bin/sh"))
	builder.WriteString(plistString("-lc"))
	builder.WriteString(plistString(strings.Join(commandParts, " ")))
	builder.WriteString("  </array>\n")
	builder.WriteString(plistKeyValue("WorkingDirectory", input.WorkingDirectory))
	builder.WriteString(plistKeyValue("StandardOutPath", input.StdoutPath))
	builder.WriteString(plistKeyValue("StandardErrorPath", input.StderrPath))
	builder.WriteString("  <key>RunAtLoad</key>\n  <true/>\n")
	builder.WriteString("  <key>KeepAlive</key>\n  <true/>\n")
	builder.WriteString("</dict>\n</plist>\n")
	return builder.String()
}

func plistKeyValue(key string, value string) string {
	return "  <key>" + plistEscape(key) + "</key>\n" + plistString(value)
}

func plistString(value string) string {
	return "    <string>" + plistEscape(value) + "</string>\n"
}

func plistEscape(value string) string {
	var buffer bytes.Buffer
	if err := xml.EscapeText(&buffer, []byte(value)); err != nil {
		panic(err)
	}
	return buffer.String()
}

func buildLaunchdBootstrapTarget(uid string) string {
	trimmedUID := strings.TrimSpace(uid)
	if trimmedUID == "" {
		return "user/501"
	}
	return "gui/" + trimmedUID
}

func uploadRemoteFile(client sshinfra.Client, remotePath string, content []byte, mode os.FileMode) error {
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("open ssh session for upload %s: %w", remotePath, err)
	}
	defer func() {
		_ = session.Close()
	}()

	stdin, err := session.StdinPipe()
	if err != nil {
		return fmt.Errorf("open ssh stdin for upload %s: %w", remotePath, err)
	}

	tmpPath := remotePath + ".tmp"
	command := "sh -lc " + sshinfra.ShellQuote(
		"set -eu\nmkdir -p "+sshinfra.ShellQuote(filepath.Dir(remotePath))+"\ncat > "+sshinfra.ShellQuote(tmpPath)+"\nchmod "+fmt.Sprintf("%#o", mode)+" "+sshinfra.ShellQuote(tmpPath)+"\nmv "+sshinfra.ShellQuote(tmpPath)+" "+sshinfra.ShellQuote(remotePath),
	)
	if err := session.Start(command); err != nil {
		_ = stdin.Close()
		return fmt.Errorf("start ssh upload %s: %w", remotePath, err)
	}
	if _, err := stdin.Write(content); err != nil {
		_ = stdin.Close()
		return fmt.Errorf("write ssh upload %s: %w", remotePath, err)
	}
	if err := stdin.Close(); err != nil {
		return fmt.Errorf("close ssh upload %s stdin: %w", remotePath, err)
	}
	if err := session.Wait(); err != nil {
		return fmt.Errorf("finish ssh upload %s: %w", remotePath, err)
	}
	return nil
}

func runRemoteSSHCommand(ctx context.Context, client sshinfra.Client, command string) (string, error) {
	_ = ctx
	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("open ssh session: %w", err)
	}
	defer func() {
		_ = session.Close()
	}()

	output, err := session.CombinedOutput(command)
	return string(output), err
}

func probeMachineSSHListenerEndpoint(ctx context.Context, machine catalogdomain.Machine) error {
	endpoint := strings.TrimSpace(pointerString(machine.AdvertisedEndpoint))
	if endpoint == "" {
		return fmt.Errorf("listener websocket endpoint is not configured for machine %s", machine.Name)
	}

	header := http.Header{}
	switch machine.ChannelCredential.Kind {
	case catalogdomain.MachineChannelCredentialKindNone, "":
	case catalogdomain.MachineChannelCredentialKindToken:
		token := strings.TrimSpace(pointerString(machine.ChannelCredential.TokenID))
		if token == "" {
			return fmt.Errorf("listener websocket token is not configured for machine %s", machine.Name)
		}
		header.Set("Authorization", "Bearer "+token)
	default:
		return fmt.Errorf("listener websocket credential kind %q is not supported", machine.ChannelCredential.Kind)
	}

	conn, response, err := (&websocket.Dialer{
		HandshakeTimeout: 5 * time.Second,
	}).DialContext(ctx, endpoint, header)
	if response != nil && response.Body != nil {
		_ = response.Body.Close()
	}
	if err != nil {
		return classifyMachineSSHListenerProbeError(machine, endpoint, response, err)
	}
	_ = conn.Close()
	return nil
}

func classifyMachineSSHListenerProbeError(
	machine catalogdomain.Machine,
	endpoint string,
	response *http.Response,
	err error,
) error {
	if response != nil {
		switch response.StatusCode {
		case http.StatusUnauthorized, http.StatusForbidden:
			return fmt.Errorf("listener websocket authentication failed for machine %s at %s", machine.Name, endpoint)
		default:
			return fmt.Errorf("listener websocket handshake failed for machine %s at %s: %s", machine.Name, endpoint, response.Status)
		}
	}

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return fmt.Errorf("listener websocket DNS resolution failed for machine %s at %s: %w", machine.Name, endpoint, err)
	}

	var hostnameErr x509.HostnameError
	if errors.As(err, &hostnameErr) {
		return fmt.Errorf("listener websocket host verification failed for machine %s at %s: %w", machine.Name, endpoint, err)
	}

	var authorityErr x509.UnknownAuthorityError
	if errors.As(err, &authorityErr) {
		return fmt.Errorf("listener websocket TLS verification failed for machine %s at %s: %w", machine.Name, endpoint, err)
	}

	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return fmt.Errorf("listener websocket endpoint unreachable for machine %s at %s: %w", machine.Name, endpoint, err)
	}

	return fmt.Errorf("listener websocket dial failed for machine %s at %s: %w", machine.Name, endpoint, err)
}

func optionalTrimmedString(raw *string) *string {
	if raw == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*raw)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func pointerString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
