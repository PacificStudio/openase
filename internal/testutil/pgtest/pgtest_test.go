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
	if err := os.MkdirAll(rootDir, 0o750); err != nil {
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

func TestEnsureSharedServerBinaryLayoutRemovesIncompleteExtraction(t *testing.T) {
	rootDir := t.TempDir()
	paths := sharedServerPathsResult{
		cachePath:    filepath.Join(rootDir, "cache"),
		binariesPath: filepath.Join(rootDir, "binaries"),
	}
	binDir := filepath.Join(paths.binariesPath, "bin")
	if err := os.MkdirAll(binDir, 0o750); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", binDir, err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "pg_ctl"), []byte("pg_ctl"), 0o600); err != nil {
		t.Fatalf("WriteFile(pg_ctl) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(binDir, "initdb"), []byte("initdb"), 0o600); err != nil {
		t.Fatalf("WriteFile(initdb) error = %v", err)
	}
	if err := os.MkdirAll(paths.cachePath, 0o750); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", paths.cachePath, err)
	}
	if err := os.WriteFile(filepath.Join(paths.cachePath, "postgres.txz"), []byte("cached"), 0o600); err != nil {
		t.Fatalf("WriteFile(cache) error = %v", err)
	}

	originalRemove := removeSharedServerPath
	t.Cleanup(func() {
		removeSharedServerPath = originalRemove
	})

	removed := make([]string, 0, 2)
	removeSharedServerPath = func(path string) error {
		removed = append(removed, path)
		return os.RemoveAll(path)
	}

	if err := ensureSharedServerBinaryLayout(paths); err != nil {
		t.Fatalf("ensureSharedServerBinaryLayout() error = %v", err)
	}

	if len(removed) != 2 {
		t.Fatalf("expected two removals, got %v", removed)
	}
	if removed[0] != paths.binariesPath || removed[1] != paths.cachePath {
		t.Fatalf("unexpected removal order: %v", removed)
	}
	if _, err := os.Stat(paths.binariesPath); !os.IsNotExist(err) {
		t.Fatalf("expected binaries path to be removed, stat err = %v", err)
	}
	if _, err := os.Stat(paths.cachePath); !os.IsNotExist(err) {
		t.Fatalf("expected cache path to be removed, stat err = %v", err)
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

func TestStartSharedServerProcessWaitsForAssetsLockBeforeCreatingController(t *testing.T) {
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
	rootDir := filepath.Join(rootBase, "attempt-1")
	if err := os.MkdirAll(rootDir, 0o750); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", rootDir, err)
	}
	createSharedServerRootDir = func(prefix string) (string, error) {
		return rootDir, nil
	}
	allocateSharedServerPort = func() (uint32, error) {
		return 43001, nil
	}

	sharedRoot := filepath.Join(rootBase, "shared")
	resolveSharedServerAssetsRoot = func() (string, error) {
		return sharedRoot, nil
	}

	controllerCreated := make(chan struct{}, 1)
	newPostgresController = func(rootDir string, port uint32) (postgresController, error) {
		controllerCreated <- struct{}{}
		return &fakePostgresController{}, nil
	}

	releaseAssetsLock, err := lockSharedServerAssets()
	if err != nil {
		t.Fatalf("lockSharedServerAssets() error = %v", err)
	}
	defer func() {
		if releaseErr := releaseAssetsLock(); releaseErr != nil {
			t.Errorf("release shared assets lock: %v", releaseErr)
		}
	}()

	startResult := make(chan sharedServerStartResult, 1)
	startErr := make(chan error, 1)
	go func() {
		started, err := startSharedServerProcess("serialized-controller")
		if err != nil {
			startErr <- err
			return
		}
		startResult <- started
	}()

	select {
	case err := <-startErr:
		t.Fatalf("startSharedServerProcess() failed unexpectedly: %v", err)
	case <-controllerCreated:
		t.Fatal("controller creation should wait until shared assets lock is released")
	case <-time.After(200 * time.Millisecond):
	}

	if err := releaseAssetsLock(); err != nil {
		t.Fatalf("release shared assets lock: %v", err)
	}
	releaseAssetsLock = func() error { return nil }

	select {
	case err := <-startErr:
		t.Fatalf("startSharedServerProcess() failed unexpectedly: %v", err)
	case started := <-startResult:
		if started.port != 43001 {
			t.Fatalf("port = %d, want %d", started.port, 43001)
		}
		if err := started.pg.Stop(); err != nil {
			t.Fatalf("Stop() error = %v", err)
		}
		if err := os.RemoveAll(started.rootDir); err != nil {
			t.Fatalf("RemoveAll(%s) error = %v", started.rootDir, err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("startSharedServerProcess() did not proceed after lock release")
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
