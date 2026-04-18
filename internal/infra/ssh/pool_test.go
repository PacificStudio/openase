package ssh

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"errors"
	"io"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/google/uuid"
	gossh "golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

func TestPoolReusesLiveConnection(t *testing.T) {
	first := &fakeClient{}
	dialer := &fakeDialer{clients: []Client{first}}
	pool := NewPool("/tmp/openase",
		WithDialer(dialer),
		WithReadFile(func(string) ([]byte, error) { return []byte("key"), nil }),
		WithHostKeyCallback(testHostKeyCallback()),
	)

	machine := testRemoteMachine()
	left, err := pool.Get(context.Background(), machine)
	if err != nil {
		t.Fatalf("first get returned error: %v", err)
	}
	right, err := pool.Get(context.Background(), machine)
	if err != nil {
		t.Fatalf("second get returned error: %v", err)
	}

	if left != right {
		t.Fatalf("expected pooled client to be reused")
	}
	if dialer.calls != 1 {
		t.Fatalf("expected one dial, got %d", dialer.calls)
	}
	if first.keepaliveCalls != 1 {
		t.Fatalf("expected one keepalive, got %d", first.keepaliveCalls)
	}
}

func TestPoolRedialsBrokenConnection(t *testing.T) {
	first := &fakeClient{keepaliveErr: errors.New("broken pipe")}
	second := &fakeClient{}
	dialer := &fakeDialer{clients: []Client{first, second}}
	pool := NewPool("/tmp/openase",
		WithDialer(dialer),
		WithReadFile(func(string) ([]byte, error) { return []byte("key"), nil }),
		WithHostKeyCallback(testHostKeyCallback()),
	)

	machine := testRemoteMachine()
	left, err := pool.Get(context.Background(), machine)
	if err != nil {
		t.Fatalf("first get returned error: %v", err)
	}
	right, err := pool.Get(context.Background(), machine)
	if err != nil {
		t.Fatalf("second get returned error: %v", err)
	}

	if left == right {
		t.Fatalf("expected broken connection to be replaced")
	}
	if dialer.calls != 2 {
		t.Fatalf("expected two dials, got %d", dialer.calls)
	}
	if !first.closed {
		t.Fatalf("expected broken client to be closed")
	}
}

func TestTesterRemoteProbe(t *testing.T) {
	client := &fakeClient{
		session: &fakeSession{output: []byte("openase\ngpu-01\nLinux 6.8 x86_64")},
	}
	dialer := &fakeDialer{clients: []Client{client}}
	pool := NewPool("/tmp/openase",
		WithDialer(dialer),
		WithReadFile(func(string) ([]byte, error) { return []byte("key"), nil }),
		WithHostKeyCallback(testHostKeyCallback()),
	)

	probe, err := NewTester(pool).TestConnection(context.Background(), testRemoteMachine())
	if err != nil {
		t.Fatalf("remote probe returned error: %v", err)
	}
	if probe.Transport != "ssh" {
		t.Fatalf("expected ssh transport, got %+v", probe)
	}
	if probe.Resources["remote_user"] != "openase" || probe.Resources["remote_host"] != "gpu-01" {
		t.Fatalf("unexpected remote probe resources: %+v", probe.Resources)
	}
	if probe.DetectedOS != domain.MachineDetectedOSLinux || probe.DetectedArch != domain.MachineDetectedArchAMD64 {
		t.Fatalf("expected linux/amd64 detection, got %+v", probe)
	}
	if probe.DetectionStatus != domain.MachineDetectionStatusOK {
		t.Fatalf("expected ok detection status, got %+v", probe)
	}
}

func TestTesterLocalProbe(t *testing.T) {
	probe, err := NewTester(nil).TestConnection(context.Background(), domain.Machine{
		ID:   uuid.New(),
		Name: domain.LocalMachineName,
		Host: domain.LocalMachineHost,
	})
	if err != nil {
		t.Fatalf("local probe returned error: %v", err)
	}
	if probe.Transport != "local" {
		t.Fatalf("expected local transport, got %+v", probe)
	}
	if _, ok := probe.Resources["local_host"]; !ok {
		t.Fatalf("expected local host resource, got %+v", probe.Resources)
	}
	if probe.DetectionStatus != domain.MachineDetectionStatusOK {
		t.Fatalf("expected local probe to detect platform, got %+v", probe)
	}
}

func TestDetectRemoteMachinePlatformDegradesOnUnknownArch(t *testing.T) {
	detectedOS, detectedArch, detectionStatus := detectRemoteMachinePlatform("openase\ngpu-01\nLinux 6.8 mystery")
	if detectedOS != domain.MachineDetectedOSLinux {
		t.Fatalf("expected linux detection, got %q", detectedOS)
	}
	if detectedArch != domain.MachineDetectedArchUnknown {
		t.Fatalf("expected unknown arch, got %q", detectedArch)
	}
	if detectionStatus != domain.MachineDetectionStatusDegraded {
		t.Fatalf("expected degraded detection status, got %q", detectionStatus)
	}
}

type fakeDialer struct {
	clients []Client
	calls   int
	onDial  func(DialConfig) error
}

func (d *fakeDialer) DialContext(_ context.Context, cfg DialConfig) (Client, error) {
	if d.onDial != nil {
		if err := d.onDial(cfg); err != nil {
			return nil, err
		}
	}
	if d.calls >= len(d.clients) {
		return nil, errors.New("unexpected dial")
	}
	client := d.clients[d.calls]
	d.calls++
	return client, nil
}

type fakeClient struct {
	keepaliveCalls int
	keepaliveErr   error
	session        Session
	closed         bool
}

func (c *fakeClient) NewSession() (Session, error) {
	if c.session == nil {
		c.session = &fakeSession{}
	}
	return c.session, nil
}

func (c *fakeClient) SendRequest(string, bool, []byte) (bool, []byte, error) {
	c.keepaliveCalls++
	if c.keepaliveErr != nil {
		return false, nil, c.keepaliveErr
	}
	return true, nil, nil
}

func (c *fakeClient) Close() error {
	c.closed = true
	return nil
}

type fakeSession struct {
	output   []byte
	err      error
	closed   bool
	closeErr error
	stdin    *io.PipeWriter
	stdout   *io.PipeReader
	stderr   *io.PipeReader
	waitCh   chan error

	startedCommand string
	startedPTY     [2]int
	resized        [2]int
	signal         string
}

func (s *fakeSession) CombinedOutput(string) ([]byte, error) {
	return s.output, s.err
}

func (s *fakeSession) StdinPipe() (io.WriteCloser, error) {
	if s.stdin == nil {
		_, writer := io.Pipe()
		s.stdin = writer
	}
	return s.stdin, nil
}

func (s *fakeSession) StdoutPipe() (io.Reader, error) {
	if s.stdout == nil {
		reader, _ := io.Pipe()
		s.stdout = reader
	}
	return s.stdout, nil
}

func (s *fakeSession) StderrPipe() (io.Reader, error) {
	if s.stderr == nil {
		reader, _ := io.Pipe()
		s.stderr = reader
	}
	return s.stderr, nil
}

func (s *fakeSession) Start(cmd string) error {
	s.startedCommand = cmd
	return nil
}

func (s *fakeSession) StartPTY(cmd string, cols int, rows int) error {
	s.startedCommand = cmd
	s.startedPTY = [2]int{cols, rows}
	return nil
}

func (s *fakeSession) Resize(cols int, rows int) error {
	s.resized = [2]int{cols, rows}
	return nil
}

func (s *fakeSession) Signal(signal string) error {
	s.signal = signal
	return nil
}

func (s *fakeSession) Wait() error {
	if s.waitCh == nil {
		return nil
	}
	return <-s.waitCh
}

func (s *fakeSession) Close() error {
	s.closed = true
	return s.closeErr
}

func testRemoteMachine() domain.Machine {
	sshUser := "openase"
	keyPath := "keys/gpu-01.pem"
	return domain.Machine{
		ID:         uuid.New(),
		Name:       "gpu-01",
		Host:       "10.0.1.10",
		Port:       22,
		SSHUser:    &sshUser,
		SSHKeyPath: &keyPath,
		Status:     "online",
	}
}

func testHostKeyCallback() gossh.HostKeyCallback {
	return gossh.InsecureIgnoreHostKey() //nolint:gosec
}

type fakeHostKeyScanner struct {
	key   gossh.PublicKey
	err   error
	calls int
}

func (s *fakeHostKeyScanner) ScanContext(context.Context, HostKeyScanConfig) (gossh.PublicKey, error) {
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	return s.key, nil
}

func testHostPublicKey(t *testing.T, fill byte) gossh.PublicKey {
	t.Helper()

	seed := bytes.Repeat([]byte{fill}, ed25519.SeedSize)
	privateKey := ed25519.NewKeyFromSeed(seed)
	publicKey, err := gossh.NewPublicKey(privateKey.Public())
	if err != nil {
		t.Fatalf("new public key: %v", err)
	}
	return publicKey
}

func TestPoolCloseClosesAllClients(t *testing.T) {
	first := &fakeClient{}
	second := &fakeClient{}
	pool := &Pool{
		conns: map[string]Client{
			"a": first,
			"b": second,
		},
	}

	if err := pool.Close(); err != nil {
		t.Fatalf("close returned error: %v", err)
	}
	if !first.closed || !second.closed {
		t.Fatalf("expected all pooled clients to close")
	}
}

func TestTesterFailureIncludesCheckedAt(t *testing.T) {
	client := &fakeClient{
		session: &fakeSession{err: errors.New("permission denied")},
	}
	dialer := &fakeDialer{clients: []Client{client}}
	pool := NewPool("/tmp/openase",
		WithDialer(dialer),
		WithReadFile(func(string) ([]byte, error) { return []byte("key"), nil }),
		WithHostKeyCallback(testHostKeyCallback()),
	)

	probe, err := NewTester(pool).TestConnection(context.Background(), testRemoteMachine())
	if err == nil {
		t.Fatal("expected remote probe error")
	}
	if probe.CheckedAt.Before(time.Now().Add(-time.Minute)) {
		t.Fatalf("expected recent checked_at, got %+v", probe)
	}
}

func TestPoolGetFailsClosedWithoutEnrolledHostKey(t *testing.T) {
	root := t.TempDir()
	hostKey := testHostPublicKey(t, 1)
	dialer := &fakeDialer{
		clients: []Client{&fakeClient{}},
		onDial: func(cfg DialConfig) error {
			return cfg.HostKeyCallback(cfg.Address, &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 22}, hostKey)
		},
	}
	pool := NewPool(root,
		WithDialer(dialer),
		WithReadFile(func(string) ([]byte, error) { return []byte("key"), nil }),
	)

	_, err := pool.Get(context.Background(), testRemoteMachine())
	if err == nil || !strings.Contains(err.Error(), "not enrolled") {
		t.Fatalf("Get() error = %v, want not enrolled guidance", err)
	}
	if dialer.calls != 0 {
		t.Fatalf("expected no dial when trust is missing, got %d", dialer.calls)
	}
}

func TestPoolEnrollHostKeyStoresFirstUseEntry(t *testing.T) {
	root := t.TempDir()
	hostKey := testHostPublicKey(t, 2)
	scanner := &fakeHostKeyScanner{key: hostKey}
	dialer := &fakeDialer{
		clients: []Client{&fakeClient{}},
		onDial: func(cfg DialConfig) error {
			return cfg.HostKeyCallback(cfg.Address, &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 22}, hostKey)
		},
	}
	pool := NewPool(root,
		WithDialer(dialer),
		WithHostKeyScanner(scanner),
		WithReadFile(func(string) ([]byte, error) { return []byte("key"), nil }),
	)

	machine := testRemoteMachine()
	result, err := pool.EnrollHostKey(context.Background(), machine, HostKeyEnrollmentOptions{})
	if err != nil {
		t.Fatalf("EnrollHostKey() error = %v", err)
	}
	if result.AlreadyTrusted || result.Replaced {
		t.Fatalf("EnrollHostKey() unexpected flags = %+v", result)
	}
	if result.FingerprintSHA256 != gossh.FingerprintSHA256(hostKey) {
		t.Fatalf("EnrollHostKey() fingerprint = %q, want %q", result.FingerprintSHA256, gossh.FingerprintSHA256(hostKey))
	}

	line, err := readManagedKnownHostsLine(pool.machineKnownHostsPath(machine))
	if err != nil {
		t.Fatalf("read managed known_hosts: %v", err)
	}
	wantLine := knownhosts.Line([]string{machineConnectionTarget(machine)}, hostKey)
	if line != wantLine {
		t.Fatalf("managed known_hosts line = %q, want %q", line, wantLine)
	}
	info, err := os.Stat(pool.machineKnownHostsPath(machine))
	if err != nil {
		t.Fatalf("stat managed known_hosts: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("managed known_hosts mode = %v, want 0600", info.Mode().Perm())
	}

	if _, err := pool.Get(context.Background(), machine); err != nil {
		t.Fatalf("Get() after enrollment error = %v", err)
	}
	if dialer.calls != 1 {
		t.Fatalf("expected one dial after enrollment, got %d", dialer.calls)
	}
}

func TestPoolEnrollHostKeyRejectsReplacelessMismatch(t *testing.T) {
	root := t.TempDir()
	initialHostKey := testHostPublicKey(t, 3)
	rotatedHostKey := testHostPublicKey(t, 4)
	scanner := &fakeHostKeyScanner{key: initialHostKey}
	pool := NewPool(root,
		WithHostKeyScanner(scanner),
		WithReadFile(func(string) ([]byte, error) { return []byte("key"), nil }),
	)

	machine := testRemoteMachine()
	if _, err := pool.EnrollHostKey(context.Background(), machine, HostKeyEnrollmentOptions{}); err != nil {
		t.Fatalf("initial EnrollHostKey() error = %v", err)
	}

	scanner.key = rotatedHostKey
	_, err := pool.EnrollHostKey(context.Background(), machine, HostKeyEnrollmentOptions{})
	if err == nil || !strings.Contains(err.Error(), "--replace") {
		t.Fatalf("EnrollHostKey() mismatch error = %v, want replace guidance", err)
	}

	line, err := readManagedKnownHostsLine(pool.machineKnownHostsPath(machine))
	if err != nil {
		t.Fatalf("read managed known_hosts: %v", err)
	}
	wantLine := knownhosts.Line([]string{machineConnectionTarget(machine)}, initialHostKey)
	if line != wantLine {
		t.Fatalf("managed known_hosts line after mismatch = %q, want %q", line, wantLine)
	}
}

func TestPoolEnrollHostKeyReplacesStoredKeyWhenRequested(t *testing.T) {
	root := t.TempDir()
	initialHostKey := testHostPublicKey(t, 5)
	rotatedHostKey := testHostPublicKey(t, 6)
	scanner := &fakeHostKeyScanner{key: initialHostKey}
	pool := NewPool(root,
		WithHostKeyScanner(scanner),
		WithReadFile(func(string) ([]byte, error) { return []byte("key"), nil }),
	)

	machine := testRemoteMachine()
	if _, err := pool.EnrollHostKey(context.Background(), machine, HostKeyEnrollmentOptions{}); err != nil {
		t.Fatalf("initial EnrollHostKey() error = %v", err)
	}

	scanner.key = rotatedHostKey
	result, err := pool.EnrollHostKey(context.Background(), machine, HostKeyEnrollmentOptions{Replace: true})
	if err != nil {
		t.Fatalf("EnrollHostKey(replace) error = %v", err)
	}
	if !result.Replaced || result.AlreadyTrusted {
		t.Fatalf("EnrollHostKey(replace) flags = %+v", result)
	}

	line, err := readManagedKnownHostsLine(pool.machineKnownHostsPath(machine))
	if err != nil {
		t.Fatalf("read managed known_hosts: %v", err)
	}
	wantLine := knownhosts.Line([]string{machineConnectionTarget(machine)}, rotatedHostKey)
	if line != wantLine {
		t.Fatalf("managed known_hosts line after replace = %q, want %q", line, wantLine)
	}
}

func TestPoolGetRejectsChangedHostKey(t *testing.T) {
	root := t.TempDir()
	initialHostKey := testHostPublicKey(t, 7)
	rotatedHostKey := testHostPublicKey(t, 8)
	scanner := &fakeHostKeyScanner{key: initialHostKey}
	dialer := &fakeDialer{
		clients: []Client{&fakeClient{}},
		onDial: func(cfg DialConfig) error {
			return cfg.HostKeyCallback(cfg.Address, &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 22}, rotatedHostKey)
		},
	}
	pool := NewPool(root,
		WithDialer(dialer),
		WithHostKeyScanner(scanner),
		WithReadFile(func(string) ([]byte, error) { return []byte("key"), nil }),
	)

	machine := testRemoteMachine()
	if _, err := pool.EnrollHostKey(context.Background(), machine, HostKeyEnrollmentOptions{}); err != nil {
		t.Fatalf("initial EnrollHostKey() error = %v", err)
	}

	_, err := pool.Get(context.Background(), machine)
	if err == nil || !strings.Contains(err.Error(), "mismatch") || !strings.Contains(err.Error(), "--replace") {
		t.Fatalf("Get() changed host key error = %v, want mismatch guidance", err)
	}
}
