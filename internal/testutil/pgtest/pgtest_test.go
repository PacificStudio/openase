package pgtest

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
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

var sharedServerTestGlobalsMu sync.Mutex

func lockSharedServerTestGlobals(t *testing.T) {
	t.Helper()
	sharedServerTestGlobalsMu.Lock()
	t.Cleanup(func() {
		sharedServerTestGlobalsMu.Unlock()
	})
}

func TestStartSharedServerProcessRetriesPortConflicts(t *testing.T) {
	lockSharedServerTestGlobals(t)

	originalCreateRootDir := createSharedServerRootDir
	originalAllocatePort := allocateSharedServerPort
	originalResolveAssetsRoot := resolveSharedServerAssetsRoot
	originalNewPostgres := newPostgresController
	t.Cleanup(func() {
		createSharedServerRootDir = originalCreateRootDir
		allocateSharedServerPort = originalAllocatePort
		resolveSharedServerAssetsRoot = originalResolveAssetsRoot
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
	resolveSharedServerAssetsRoot = func() (string, error) {
		return filepath.Join(rootBase, "shared"), nil
	}
	newPostgresController = func(rootDir string, port uint32) (postgresController, error) {
		if controllerCalls >= len(controllers) {
			t.Fatalf("unexpected extra postgres controller request %d", controllerCalls+1)
		}
		controller := controllers[controllerCalls]
		controllerCalls++
		return controller, nil
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
	lockSharedServerTestGlobals(t)

	originalCreateRootDir := createSharedServerRootDir
	originalAllocatePort := allocateSharedServerPort
	originalResolveAssetsRoot := resolveSharedServerAssetsRoot
	originalNewPostgres := newPostgresController
	t.Cleanup(func() {
		createSharedServerRootDir = originalCreateRootDir
		allocateSharedServerPort = originalAllocatePort
		resolveSharedServerAssetsRoot = originalResolveAssetsRoot
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
	resolveSharedServerAssetsRoot = func() (string, error) {
		return filepath.Join(rootDir, "shared"), nil
	}
	newPostgresController = func(rootDir string, port uint32) (postgresController, error) {
		return controller, nil
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

func TestSharedServerPathsUseSharedBinariesPath(t *testing.T) {
	lockSharedServerTestGlobals(t)

	originalResolveAssetsRoot := resolveSharedServerAssetsRoot
	t.Cleanup(func() {
		resolveSharedServerAssetsRoot = originalResolveAssetsRoot
	})

	sharedRoot := filepath.Join(t.TempDir(), "shared")
	resolveSharedServerAssetsRoot = func() (string, error) {
		return sharedRoot, nil
	}

	rootDir := filepath.Join(t.TempDir(), "instance")
	paths, err := sharedServerPaths(rootDir)
	if err != nil {
		t.Fatalf("sharedServerPaths() error = %v", err)
	}

	wantSharedVersionRoot := filepath.Join(sharedRoot, "postgres-"+string(embeddedpostgres.V16))
	if paths.binariesPath != filepath.Join(wantSharedVersionRoot, "binaries") {
		t.Fatalf("binariesPath = %q, want %q", paths.binariesPath, filepath.Join(wantSharedVersionRoot, "binaries"))
	}
	if paths.cachePath != filepath.Join(wantSharedVersionRoot, "cache") {
		t.Fatalf("cachePath = %q, want %q", paths.cachePath, filepath.Join(wantSharedVersionRoot, "cache"))
	}
	if paths.runtimePath != filepath.Join(rootDir, "runtime") {
		t.Fatalf("runtimePath = %q, want %q", paths.runtimePath, filepath.Join(rootDir, "runtime"))
	}
	if paths.dataPath != filepath.Join(rootDir, "data") {
		t.Fatalf("dataPath = %q, want %q", paths.dataPath, filepath.Join(rootDir, "data"))
	}
}

func TestLockSharedServerAssetsSerializesCallers(t *testing.T) {
	lockSharedServerTestGlobals(t)

	originalResolveAssetsRoot := resolveSharedServerAssetsRoot
	t.Cleanup(func() {
		resolveSharedServerAssetsRoot = originalResolveAssetsRoot
	})

	sharedRoot := filepath.Join(t.TempDir(), "shared")
	resolveSharedServerAssetsRoot = func() (string, error) {
		return sharedRoot, nil
	}

	releaseFirst, err := lockSharedServerAssets()
	if err != nil {
		t.Fatalf("lockSharedServerAssets() first lock error = %v", err)
	}
	defer func() {
		if releaseErr := releaseFirst(); releaseErr != nil {
			t.Errorf("release first shared assets lock: %v", releaseErr)
		}
	}()

	acquiredSecond := make(chan func() error, 1)
	secondErr := make(chan error, 1)
	go func() {
		releaseSecond, err := lockSharedServerAssets()
		if err != nil {
			secondErr <- err
			return
		}
		acquiredSecond <- releaseSecond
	}()

	select {
	case err := <-secondErr:
		t.Fatalf("second lock attempt failed unexpectedly: %v", err)
	case <-acquiredSecond:
		t.Fatal("second lock should block while first lock is held")
	case <-time.After(200 * time.Millisecond):
	}

	if err := releaseFirst(); err != nil {
		t.Fatalf("release first shared assets lock: %v", err)
	}
	releaseFirst = func() error { return nil }

	select {
	case err := <-secondErr:
		t.Fatalf("second lock attempt failed unexpectedly: %v", err)
	case releaseSecond := <-acquiredSecond:
		if err := releaseSecond(); err != nil {
			t.Fatalf("release second shared assets lock: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("second lock did not acquire after first lock released")
	}
}

func TestNewIsolatedEntClientReusesTemplateDatabase(t *testing.T) {
	server, err := StartSharedServer("template_reuse")
	if err != nil {
		t.Fatalf("StartSharedServer() error = %v", err)
	}
	t.Cleanup(func() {
		if closeErr := server.Close(); closeErr != nil {
			t.Errorf("Close() error = %v", closeErr)
		}
	})

	first := server.NewIsolatedEntClient(t)
	templateName := server.templateDatabase
	if templateName == "" {
		t.Fatal("expected template database to be recorded after first ent client")
	}

	ctx := context.Background()
	if _, err := first.Organization.Create().
		SetName("Template Reuse").
		SetSlug("template-reuse").
		Save(ctx); err != nil {
		t.Fatalf("seed first database: %v", err)
	}

	second := server.NewIsolatedEntClient(t)
	if server.templateDatabase != templateName {
		t.Fatalf("template database changed from %q to %q", templateName, server.templateDatabase)
	}
	count, err := second.Organization.Query().Count(ctx)
	if err != nil {
		t.Fatalf("count organizations in cloned database: %v", err)
	}
	if count != 0 {
		t.Fatalf("cloned database should start from template snapshot, got %d organizations", count)
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
