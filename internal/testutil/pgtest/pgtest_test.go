package pgtest

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

type fakePostgresController struct {
	startErr error
	stopErr  error
	started  int
	stopped  int
}

func (f *fakePostgresController) Start() error {
	f.started++
	return f.startErr
}

func (f *fakePostgresController) Stop() error {
	f.stopped++
	return f.stopErr
}

func TestStartSharedServerProcessRetriesPortConflicts(t *testing.T) {
	t.Parallel()

	originalCreateRootDir := createSharedServerRootDir
	originalAllocatePort := allocateSharedServerPort
	originalNewPostgres := newPostgresController
	t.Cleanup(func() {
		createSharedServerRootDir = originalCreateRootDir
		allocateSharedServerPort = originalAllocatePort
		newPostgresController = originalNewPostgres
	})

	rootBase := t.TempDir()
	rootDirs := []string{
		filepath.Join(rootBase, "attempt-1"),
		filepath.Join(rootBase, "attempt-2"),
		filepath.Join(rootBase, "attempt-3"),
	}
	for _, rootDir := range rootDirs {
		if err := os.MkdirAll(rootDir, 0o755); err != nil {
			t.Fatalf("MkdirAll(%s) error = %v", rootDir, err)
		}
	}

	createCalls := 0
	createSharedServerRootDir = func(prefix string) (string, error) {
		if createCalls >= len(rootDirs) {
			t.Fatalf("unexpected extra temp dir request %d", createCalls+1)
		}
		rootDir := rootDirs[createCalls]
		createCalls++
		return rootDir, nil
	}

	ports := []uint32{41001, 41002, 41003}
	portCalls := 0
	allocateSharedServerPort = func() (uint32, error) {
		if portCalls >= len(ports) {
			t.Fatalf("unexpected extra port allocation %d", portCalls+1)
		}
		port := ports[portCalls]
		portCalls++
		return port, nil
	}

	controllers := []*fakePostgresController{
		{startErr: errors.New("process already listening on port 41001")},
		{startErr: errors.New("could not start postgres using pg_ctl:\nFATAL: could not bind IPv4 address \"127.0.0.1\": Address already in use")},
		{},
	}
	controllerCalls := 0
	newPostgresController = func(rootDir string, port uint32) postgresController {
		if controllerCalls >= len(controllers) {
			t.Fatalf("unexpected extra postgres controller request %d", controllerCalls+1)
		}
		controller := controllers[controllerCalls]
		controllerCalls++
		return controller
	}

	started, err := startSharedServerProcess("retry")
	if err != nil {
		t.Fatalf("startSharedServerProcess() error = %v", err)
	}

	if started.rootDir != rootDirs[2] {
		t.Fatalf("rootDir = %s, want %s", started.rootDir, rootDirs[2])
	}
	if started.port != ports[2] {
		t.Fatalf("port = %d, want %d", started.port, ports[2])
	}
	if started.pg != controllers[2] {
		t.Fatal("expected final controller to be returned")
	}
	if controllers[0].stopped != 1 || controllers[1].stopped != 1 {
		t.Fatalf("failed attempts should call Stop once, got stopped=%d and %d", controllers[0].stopped, controllers[1].stopped)
	}
	for _, rootDir := range rootDirs[:2] {
		if _, err := os.Stat(rootDir); !os.IsNotExist(err) {
			t.Fatalf("expected failed attempt dir %s to be removed, stat err = %v", rootDir, err)
		}
	}
}

func TestStartSharedServerProcessDoesNotRetryNonPortErrors(t *testing.T) {
	t.Parallel()

	originalCreateRootDir := createSharedServerRootDir
	originalAllocatePort := allocateSharedServerPort
	originalNewPostgres := newPostgresController
	t.Cleanup(func() {
		createSharedServerRootDir = originalCreateRootDir
		allocateSharedServerPort = originalAllocatePort
		newPostgresController = originalNewPostgres
	})

	rootDir := filepath.Join(t.TempDir(), "attempt-1")
	if err := os.MkdirAll(rootDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", rootDir, err)
	}

	createCalls := 0
	createSharedServerRootDir = func(prefix string) (string, error) {
		createCalls++
		return rootDir, nil
	}

	allocateCalls := 0
	allocateSharedServerPort = func() (uint32, error) {
		allocateCalls++
		return 42001, nil
	}

	controller := &fakePostgresController{startErr: errors.New("timed out waiting for database to become available")}
	newPostgresController = func(rootDir string, port uint32) postgresController {
		return controller
	}

	_, err := startSharedServerProcess("nonretry")
	if err == nil {
		t.Fatal("startSharedServerProcess() expected error")
	}
	if createCalls != 1 || allocateCalls != 1 {
		t.Fatalf("expected single attempt, got createCalls=%d allocateCalls=%d", createCalls, allocateCalls)
	}
	if controller.stopped != 1 {
		t.Fatalf("failed startup should stop controller once, got %d", controller.stopped)
	}
	if _, statErr := os.Stat(rootDir); !os.IsNotExist(statErr) {
		t.Fatalf("expected failed attempt dir %s to be removed, stat err = %v", rootDir, statErr)
	}
}

func TestIsRetryablePortStartupError(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "listener", err: errors.New("process already listening on port 41001"), want: true},
		{name: "bind", err: errors.New("FATAL: could not bind IPv4 address \"127.0.0.1\": Address already in use"), want: true},
		{name: "sockets", err: errors.New("failed to create any TCP/IP sockets"), want: true},
		{name: "timeout", err: errors.New("timed out waiting for database to become available"), want: false},
	}

	for _, tt := range cases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := isRetryablePortStartupError(tt.err); got != tt.want {
				t.Fatalf("isRetryablePortStartupError(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}
