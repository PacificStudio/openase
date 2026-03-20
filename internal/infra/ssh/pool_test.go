package ssh

import (
	"context"
	"errors"
	"testing"
	"time"

	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/google/uuid"
)

func TestPoolReusesLiveConnection(t *testing.T) {
	first := &fakeClient{}
	dialer := &fakeDialer{clients: []Client{first}}
	pool := NewPool("/tmp/openase", WithDialer(dialer), WithReadFile(func(string) ([]byte, error) {
		return []byte("key"), nil
	}))

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
	pool := NewPool("/tmp/openase", WithDialer(dialer), WithReadFile(func(string) ([]byte, error) {
		return []byte("key"), nil
	}))

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
	pool := NewPool("/tmp/openase", WithDialer(dialer), WithReadFile(func(string) ([]byte, error) {
		return []byte("key"), nil
	}))

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
}

type fakeDialer struct {
	clients []Client
	calls   int
}

func (d *fakeDialer) DialContext(context.Context, DialConfig) (Client, error) {
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
	output []byte
	err    error
	closed bool
}

func (s *fakeSession) CombinedOutput(string) ([]byte, error) {
	return s.output, s.err
}

func (s *fakeSession) Close() error {
	s.closed = true
	return nil
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
	pool := NewPool("/tmp/openase", WithDialer(dialer), WithReadFile(func(string) ([]byte, error) {
		return []byte("key"), nil
	}))

	probe, err := NewTester(pool).TestConnection(context.Background(), testRemoteMachine())
	if err == nil {
		t.Fatal("expected remote probe error")
	}
	if probe.CheckedAt.Before(time.Now().Add(-time.Minute)) {
		t.Fatalf("expected recent checked_at, got %+v", probe)
	}
}
