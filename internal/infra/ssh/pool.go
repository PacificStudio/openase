package ssh

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	gossh "golang.org/x/crypto/ssh"
)

type Session interface {
	CombinedOutput(cmd string) ([]byte, error)
	Close() error
}

type Client interface {
	NewSession() (Session, error)
	SendRequest(name string, wantReply bool, payload []byte) (bool, []byte, error)
	Close() error
}

type DialConfig struct {
	Address         string
	User            string
	KeyBytes        []byte
	Timeout         time.Duration
	HostKeyCallback gossh.HostKeyCallback
}

type Dialer interface {
	DialContext(ctx context.Context, cfg DialConfig) (Client, error)
}

type PoolOption func(*Pool)

type Pool struct {
	mu              sync.Mutex
	conns           map[string]Client
	openASEHomeDir  string
	dialer          Dialer
	readFile        func(string) ([]byte, error)
	timeout         time.Duration
	hostKeyCallback gossh.HostKeyCallback
}

func NewPool(openASEHomeDir string, opts ...PoolOption) *Pool {
	pool := &Pool{
		conns:           map[string]Client{},
		openASEHomeDir:  filepath.Clean(openASEHomeDir),
		dialer:          realDialer{},
		readFile:        os.ReadFile,
		timeout:         10 * time.Second,
		hostKeyCallback: gossh.InsecureIgnoreHostKey(), //nolint:gosec
	}
	for _, opt := range opts {
		if opt != nil {
			opt(pool)
		}
	}
	return pool
}

func WithDialer(dialer Dialer) PoolOption {
	return func(pool *Pool) {
		if dialer != nil {
			pool.dialer = dialer
		}
	}
}

func WithReadFile(readFile func(string) ([]byte, error)) PoolOption {
	return func(pool *Pool) {
		if readFile != nil {
			pool.readFile = readFile
		}
	}
}

func WithTimeout(timeout time.Duration) PoolOption {
	return func(pool *Pool) {
		if timeout > 0 {
			pool.timeout = timeout
		}
	}
}

func WithHostKeyCallback(callback gossh.HostKeyCallback) PoolOption {
	return func(pool *Pool) {
		if callback != nil {
			pool.hostKeyCallback = callback
		}
	}
}

func (p *Pool) Get(ctx context.Context, machine domain.Machine) (Client, error) {
	if machine.Host == domain.LocalMachineHost {
		return nil, fmt.Errorf("local machine does not use ssh")
	}
	if machine.SSHUser == nil {
		return nil, fmt.Errorf("machine %s is missing ssh_user", machine.Name)
	}
	if machine.SSHKeyPath == nil {
		return nil, fmt.Errorf("machine %s is missing ssh_key_path", machine.Name)
	}

	key := machine.ID.String()

	p.mu.Lock()
	defer p.mu.Unlock()

	if client, ok := p.conns[key]; ok {
		if _, _, err := client.SendRequest("keepalive@openase", true, nil); err == nil {
			return client, nil
		}
		_ = client.Close()
		delete(p.conns, key)
	}

	keyBytes, err := p.readFile(p.resolveKeyPath(*machine.SSHKeyPath))
	if err != nil {
		return nil, fmt.Errorf("read ssh key: %w", err)
	}

	client, err := p.dialer.DialContext(ctx, DialConfig{
		Address:         net.JoinHostPort(machine.Host, fmt.Sprintf("%d", machine.Port)),
		User:            *machine.SSHUser,
		KeyBytes:        keyBytes,
		Timeout:         p.timeout,
		HostKeyCallback: p.hostKeyCallback,
	})
	if err != nil {
		return nil, fmt.Errorf("dial machine %s: %w", machine.Name, err)
	}

	p.conns[key] = client
	return client, nil
}

func (p *Pool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	errs := make([]error, 0, len(p.conns))
	for key, client := range p.conns {
		errs = append(errs, client.Close())
		delete(p.conns, key)
	}

	return joinErrors(errs...)
}

func (p *Pool) resolveKeyPath(raw string) string {
	if filepath.IsAbs(raw) {
		return filepath.Clean(raw)
	}
	return filepath.Join(p.openASEHomeDir, raw)
}

type Tester struct {
	pool *Pool
}

func NewTester(pool *Pool) *Tester {
	return &Tester{pool: pool}
}

func (t *Tester) TestConnection(ctx context.Context, machine domain.Machine) (domain.MachineProbe, error) {
	checkedAt := time.Now().UTC()
	if machine.Host == domain.LocalMachineHost {
		return localProbe(checkedAt)
	}
	if t == nil || t.pool == nil {
		return domain.MachineProbe{CheckedAt: checkedAt, Transport: "ssh"}, fmt.Errorf("ssh pool unavailable")
	}

	client, err := t.pool.Get(ctx, machine)
	if err != nil {
		return domain.MachineProbe{CheckedAt: checkedAt, Transport: "ssh"}, err
	}

	session, err := client.NewSession()
	if err != nil {
		return domain.MachineProbe{CheckedAt: checkedAt, Transport: "ssh"}, fmt.Errorf("open ssh session: %w", err)
	}
	defer func() {
		_ = session.Close()
	}()

	output, err := session.CombinedOutput(`sh -lc 'whoami && hostname && uname -srm'`)
	probe := domain.MachineProbe{
		CheckedAt: checkedAt,
		Transport: "ssh",
		Output:    strings.TrimSpace(string(output)),
		Resources: buildRemoteResources(machine, checkedAt, string(output)),
	}
	if err != nil {
		return probe, fmt.Errorf("run remote probe: %w", err)
	}

	return probe, nil
}

func localProbe(checkedAt time.Time) (domain.MachineProbe, error) {
	currentUser, err := user.Current()
	if err != nil {
		return domain.MachineProbe{CheckedAt: checkedAt, Transport: "local"}, fmt.Errorf("resolve local user: %w", err)
	}
	hostname, err := os.Hostname()
	if err != nil {
		return domain.MachineProbe{CheckedAt: checkedAt, Transport: "local"}, fmt.Errorf("resolve local hostname: %w", err)
	}

	output := strings.Join([]string{currentUser.Username, hostname, runtime.GOOS + " " + runtime.GOARCH}, "\n")
	return domain.MachineProbe{
		CheckedAt: checkedAt,
		Transport: "local",
		Output:    output,
		Resources: map[string]any{
			"transport":    "local",
			"local_user":   currentUser.Username,
			"local_host":   hostname,
			"platform":     runtime.GOOS + "/" + runtime.GOARCH,
			"checked_at":   checkedAt.Format(time.RFC3339),
			"last_success": true,
		},
	}, nil
}

func buildRemoteResources(machine domain.Machine, checkedAt time.Time, output string) map[string]any {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	resources := map[string]any{
		"transport":    "ssh",
		"host":         machine.Host,
		"port":         machine.Port,
		"checked_at":   checkedAt.Format(time.RFC3339),
		"last_success": true,
	}
	if len(lines) > 0 && strings.TrimSpace(lines[0]) != "" {
		resources["remote_user"] = strings.TrimSpace(lines[0])
	}
	if len(lines) > 1 && strings.TrimSpace(lines[1]) != "" {
		resources["remote_host"] = strings.TrimSpace(lines[1])
	}
	if len(lines) > 2 && strings.TrimSpace(lines[2]) != "" {
		resources["kernel"] = strings.TrimSpace(lines[2])
	}
	return resources
}

type realDialer struct{}

func (realDialer) DialContext(ctx context.Context, cfg DialConfig) (Client, error) {
	signer, err := gossh.ParsePrivateKey(cfg.KeyBytes)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}

	netConn, err := (&net.Dialer{Timeout: cfg.Timeout}).DialContext(ctx, "tcp", cfg.Address)
	if err != nil {
		return nil, err
	}

	clientConn, channels, requests, err := gossh.NewClientConn(netConn, cfg.Address, &gossh.ClientConfig{
		User:            cfg.User,
		Auth:            []gossh.AuthMethod{gossh.PublicKeys(signer)},
		HostKeyCallback: cfg.HostKeyCallback,
		Timeout:         cfg.Timeout,
	})
	if err != nil {
		_ = netConn.Close()
		return nil, err
	}

	return &realClient{client: gossh.NewClient(clientConn, channels, requests)}, nil
}

type realClient struct {
	client *gossh.Client
}

func (c *realClient) NewSession() (Session, error) {
	session, err := c.client.NewSession()
	if err != nil {
		return nil, err
	}
	return &realSession{session: session}, nil
}

func (c *realClient) SendRequest(name string, wantReply bool, payload []byte) (bool, []byte, error) {
	return c.client.SendRequest(name, wantReply, payload)
}

func (c *realClient) Close() error {
	return c.client.Close()
}

type realSession struct {
	session *gossh.Session
}

func (s *realSession) CombinedOutput(cmd string) ([]byte, error) {
	return s.session.CombinedOutput(cmd)
}

func (s *realSession) Close() error {
	return s.session.Close()
}

func joinErrors(errs ...error) error {
	var filtered []error
	for _, err := range errs {
		if err != nil {
			filtered = append(filtered, err)
		}
	}
	if len(filtered) == 0 {
		return nil
	}
	if len(filtered) == 1 {
		return filtered[0]
	}
	var parts []string
	for _, err := range filtered {
		parts = append(parts, err.Error())
	}
	return errors.New(strings.Join(parts, "; "))
}
