package ssh

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/BetterAndBetterII/openase/internal/logging"
	gossh "golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

const machineKnownHostsDir = "ssh/known_hosts.d"

var sshHostKeyComponent = logging.DeclareComponent("ssh-host-keys")

type HostKeyScanConfig struct {
	Address  string
	User     string
	KeyBytes []byte
	Timeout  time.Duration
}

type HostKeyScanner interface {
	ScanContext(context.Context, HostKeyScanConfig) (gossh.PublicKey, error)
}

type HostKeyEnrollmentOptions struct {
	Replace bool
}

type HostKeyEnrollmentResult struct {
	MachineID         string `json:"machine_id"`
	MachineName       string `json:"machine_name"`
	ConnectionTarget  string `json:"connection_target"`
	KnownHostsPath    string `json:"known_hosts_path"`
	Algorithm         string `json:"algorithm"`
	FingerprintSHA256 string `json:"fingerprint_sha256"`
	Replaced          bool   `json:"replaced"`
	AlreadyTrusted    bool   `json:"already_trusted"`
	Summary           string `json:"summary"`
}

type realHostKeyScanner struct{}

func (p *Pool) hostKeyLogger() *slog.Logger {
	return logging.WithComponent(p.logger, sshHostKeyComponent)
}

func (realHostKeyScanner) ScanContext(ctx context.Context, cfg HostKeyScanConfig) (gossh.PublicKey, error) {
	signer, err := gossh.ParsePrivateKey(cfg.KeyBytes)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}

	netConn, err := (&net.Dialer{Timeout: cfg.Timeout}).DialContext(ctx, "tcp", cfg.Address)
	if err != nil {
		return nil, err
	}

	var hostKey gossh.PublicKey
	clientConn, channels, requests, err := gossh.NewClientConn(netConn, cfg.Address, &gossh.ClientConfig{
		User: cfg.User,
		Auth: []gossh.AuthMethod{gossh.PublicKeys(signer)},
		HostKeyCallback: func(_ string, _ net.Addr, key gossh.PublicKey) error {
			hostKey = key
			return nil
		},
		Timeout: cfg.Timeout,
	})
	if err != nil {
		_ = netConn.Close()
		return nil, err
	}

	client := gossh.NewClient(clientConn, channels, requests)
	defer func() {
		_ = client.Close()
	}()

	if hostKey == nil {
		return nil, fmt.Errorf("ssh server did not present a host key")
	}
	return hostKey, nil
}

func WithHostKeyScanner(scanner HostKeyScanner) PoolOption {
	return func(pool *Pool) {
		if scanner != nil {
			pool.hostKeyScanner = scanner
		}
	}
}

func (p *Pool) EnrollHostKey(ctx context.Context, machine domain.Machine, opts HostKeyEnrollmentOptions) (HostKeyEnrollmentResult, error) {
	keyBytes, err := p.readMachinePrivateKey(machine)
	if err != nil {
		return HostKeyEnrollmentResult{}, err
	}

	target := machineConnectionTarget(machine)
	hostKey, err := p.hostKeyScanner.ScanContext(ctx, HostKeyScanConfig{
		Address:  target,
		User:     strings.TrimSpace(*machine.SSHUser),
		KeyBytes: keyBytes,
		Timeout:  p.timeout,
	})
	if err != nil {
		return HostKeyEnrollmentResult{}, fmt.Errorf("scan ssh host key for machine %s (%s): %w", machine.Name, target, err)
	}
	if hostKey == nil {
		return HostKeyEnrollmentResult{}, fmt.Errorf("scan ssh host key for machine %s (%s): ssh server did not present a host key", machine.Name, target)
	}

	path := p.machineKnownHostsPath(machine)
	line := knownhosts.Line([]string{target}, hostKey)
	currentLine, err := readManagedKnownHostsLine(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return HostKeyEnrollmentResult{}, fmt.Errorf("read stored ssh host key for machine %s (%s): %w", machine.Name, target, err)
	}

	result := buildHostKeyEnrollmentResult(machine, path, hostKey)
	if currentLine == line {
		p.hostKeyLogger().Debug("ssh host key already enrolled", "machine_id", machine.ID.String(), "machine_name", machine.Name, "connection_target", target, "known_hosts_path", path, "fingerprint_sha256", result.FingerprintSHA256)
		result.AlreadyTrusted = true
		result.Summary = fmt.Sprintf("SSH host key %s (%s) for machine %s (%s) is already enrolled at %s.", result.Algorithm, result.FingerprintSHA256, machine.Name, target, path)
		return result, nil
	}
	if currentLine != "" && !opts.Replace {
		storedFingerprint := "unknown"
		if storedKey, parseErr := parseManagedKnownHostsPublicKey(currentLine); parseErr == nil {
			storedFingerprint = gossh.FingerprintSHA256(storedKey)
		}
		return HostKeyEnrollmentResult{}, fmt.Errorf("ssh host key for machine %s (%s) is already enrolled as %s; verify the machine and rerun `openase machine ssh-enroll %s --replace` to accept %s", machine.Name, target, storedFingerprint, machine.ID.String(), result.FingerprintSHA256)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return HostKeyEnrollmentResult{}, fmt.Errorf("prepare ssh host key directory for machine %s (%s): %w", machine.Name, target, err)
	}
	if err := os.WriteFile(path, []byte(line+"\n"), 0o600); err != nil {
		return HostKeyEnrollmentResult{}, fmt.Errorf("write ssh host key for machine %s (%s): %w", machine.Name, target, err)
	}

	p.dropCachedMachineConnection(machine.ID.String())
	p.hostKeyLogger().Debug("stored ssh host key", "machine_id", machine.ID.String(), "machine_name", machine.Name, "connection_target", target, "known_hosts_path", path, "fingerprint_sha256", result.FingerprintSHA256, "replaced", currentLine != "")
	if currentLine != "" {
		result.Replaced = true
		result.Summary = fmt.Sprintf("Replaced the stored SSH host key for machine %s (%s) with %s (%s) at %s.", machine.Name, target, result.Algorithm, result.FingerprintSHA256, path)
		return result, nil
	}
	result.Summary = fmt.Sprintf("Enrolled SSH host key %s (%s) for machine %s (%s) at %s.", result.Algorithm, result.FingerprintSHA256, machine.Name, target, path)
	return result, nil
}

func (p *Pool) resolveHostKeyCallback(machine domain.Machine) (gossh.HostKeyCallback, error) {
	if p.hostKeyCallback != nil {
		return p.hostKeyCallback, nil
	}

	path := p.machineKnownHostsPath(machine)
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("ssh host key for machine %s (%s) is not enrolled; run `openase machine ssh-enroll %s` before connecting", machine.Name, machineConnectionTarget(machine), machine.ID.String())
		}
		return nil, fmt.Errorf("inspect stored ssh host key for machine %s (%s): %w", machine.Name, machineConnectionTarget(machine), err)
	}

	callback, err := knownhosts.New(path)
	if err != nil {
		return nil, fmt.Errorf("load stored ssh host key for machine %s (%s): %w", machine.Name, machineConnectionTarget(machine), err)
	}
	return func(hostname string, remote net.Addr, key gossh.PublicKey) error {
		if err := callback(hostname, remote, key); err != nil {
			var keyErr *knownhosts.KeyError
			if errors.As(err, &keyErr) {
				if len(keyErr.Want) == 0 {
					return fmt.Errorf("ssh host key for machine %s (%s) is not enrolled for the current address; run `openase machine ssh-enroll %s` before connecting", machine.Name, machineConnectionTarget(machine), machine.ID.String())
				}
				return fmt.Errorf("ssh host key mismatch for machine %s (%s); verify the machine and rerun `openase machine ssh-enroll %s --replace` if the rotation is expected", machine.Name, machineConnectionTarget(machine), machine.ID.String())
			}
			return err
		}
		return nil
	}, nil
}

func validateRemoteMachineSSH(machine domain.Machine) error {
	if machine.Host == domain.LocalMachineHost {
		return fmt.Errorf("local machine does not use ssh")
	}
	if machine.SSHUser == nil {
		return fmt.Errorf("machine %s is missing ssh_user", machine.Name)
	}
	if machine.SSHKeyPath == nil {
		return fmt.Errorf("machine %s is missing ssh_key_path", machine.Name)
	}
	return nil
}

func (p *Pool) readMachinePrivateKey(machine domain.Machine) ([]byte, error) {
	if err := validateRemoteMachineSSH(machine); err != nil {
		return nil, err
	}

	keyPath := p.resolveKeyPath(*machine.SSHKeyPath)
	keyBytes, err := p.readFile(keyPath)
	if err != nil {
		p.componentLogger().Error("read ssh key failed", "machine_id", machine.ID.String(), "machine_name", machine.Name, "host", machine.Host, "ssh_key_path", keyPath, "error", err)
		return nil, fmt.Errorf("read ssh key: %w", err)
	}
	return keyBytes, nil
}

func (p *Pool) machineKnownHostsPath(machine domain.Machine) string {
	fileName := machine.ID.String()
	if strings.TrimSpace(fileName) == "" {
		fileName = "machine"
	}
	return filepath.Join(p.openASEHomeDir, machineKnownHostsDir, fileName+".known_hosts")
}

func machineConnectionTarget(machine domain.Machine) string {
	return net.JoinHostPort(machine.Host, strconv.Itoa(machine.Port))
}

func buildHostKeyEnrollmentResult(machine domain.Machine, path string, key gossh.PublicKey) HostKeyEnrollmentResult {
	return HostKeyEnrollmentResult{
		MachineID:         machine.ID.String(),
		MachineName:       machine.Name,
		ConnectionTarget:  machineConnectionTarget(machine),
		KnownHostsPath:    path,
		Algorithm:         key.Type(),
		FingerprintSHA256: gossh.FingerprintSHA256(key),
	}
}

func readManagedKnownHostsLine(path string) (string, error) {
	//nolint:gosec // path points to an OpenASE-managed known_hosts file beneath openASEHomeDir
	body, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(string(body), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		return trimmed, nil
	}
	return "", nil
}

func parseManagedKnownHostsPublicKey(line string) (gossh.PublicKey, error) {
	fields := strings.Fields(strings.TrimSpace(line))
	if len(fields) < 3 {
		return nil, fmt.Errorf("invalid known_hosts entry")
	}
	key, _, _, _, err := gossh.ParseAuthorizedKey([]byte(fields[1] + " " + fields[2]))
	if err != nil {
		return nil, err
	}
	return key, nil
}

func (p *Pool) dropCachedMachineConnection(machineKey string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	client, ok := p.conns[machineKey]
	if !ok {
		return
	}
	_ = client.Close()
	delete(p.conns, machineKey)
}
