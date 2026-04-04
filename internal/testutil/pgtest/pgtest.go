package pgtest

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"golang.org/x/sys/unix"
)

const defaultDatabase = "postgres"

const sharedServerStartAttempts = 5
const sharedServerAssetsRootEnv = "OPENASE_PGTEST_SHARED_ROOT"
const isolatedDatabaseDDLTimeout = 45 * time.Second

type postgresController interface {
	Start() error
	Stop() error
}

type sharedServerStartResult struct {
	rootDir string
	port    uint32
	pg      postgresController
}

var (
	createSharedServerRootDir = func(prefix string) (string, error) {
		return os.MkdirTemp("", prefix)
	}
	allocateSharedServerPort      = freePort
	removeSharedServerPath        = os.RemoveAll
	resolveSharedServerAssetsRoot = func() (string, error) {
		if override := strings.TrimSpace(os.Getenv(sharedServerAssetsRootEnv)); override != "" {
			return override, nil
		}
		cacheDir, err := os.UserCacheDir()
		if err != nil {
			return "", fmt.Errorf("resolve user cache dir: %w", err)
		}
		return filepath.Join(cacheDir, "openase", "pgtest"), nil
	}
	newPostgresController = func(rootDir string, port uint32) (postgresController, error) {
		paths, err := sharedServerPaths(rootDir)
		if err != nil {
			return nil, err
		}
		if err := ensureSharedServerBinaryLayout(paths); err != nil {
			return nil, err
		}
		return embeddedpostgres.NewDatabase(
			embeddedpostgres.DefaultConfig().
				Version(embeddedpostgres.V16).
				Port(port).
				Username("postgres").
				Password("postgres").
				Database(defaultDatabase).
				CachePath(paths.cachePath).
				RuntimePath(paths.runtimePath).
				BinariesPath(paths.binariesPath).
				DataPath(paths.dataPath),
		), nil
	}
)

type sharedServerPathsResult struct {
	cachePath    string
	runtimePath  string
	binariesPath string
	dataPath     string
}

type Server struct {
	admin            *sql.DB
	baseDSN          string
	port             uint32
	rootDir          string
	pg               postgresController
	prefix           string
	nextID           atomic.Uint64
	templateMu       sync.Mutex
	templateDatabase string
}

type Database struct {
	Name string
	DSN  string
	Port uint32
}

func RunTestMain(m *testing.M, packageName string, assign func(*Server)) int {
	server, err := StartSharedServer(packageName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "start shared embedded postgres for %s: %v\n", packageName, err)
		return 1
	}
	if assign != nil {
		assign(server)
	}

	exitCode := m.Run()
	if err := server.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "stop shared embedded postgres for %s: %v\n", packageName, err)
		if exitCode == 0 {
			exitCode = 1
		}
	}

	return exitCode
}

func StartSharedServer(packageName string) (*Server, error) {
	prefix := sanitizeName(packageName)
	if prefix == "" {
		prefix = "openase"
	}

	started, err := startSharedServerProcess(prefix)
	if err != nil {
		return nil, err
	}

	baseDSN := fmt.Sprintf("postgres://postgres:postgres@127.0.0.1:%d/%s?sslmode=disable", started.port, defaultDatabase)
	admin, err := sql.Open("postgres", baseDSN)
	if err != nil {
		_ = started.pg.Stop()
		_ = os.RemoveAll(started.rootDir)
		return nil, fmt.Errorf("open admin database: %w", err)
	}
	if err := admin.PingContext(context.Background()); err != nil {
		_ = admin.Close()
		_ = started.pg.Stop()
		_ = os.RemoveAll(started.rootDir)
		return nil, fmt.Errorf("ping admin database: %w", err)
	}

	return &Server{
		admin:   admin,
		baseDSN: baseDSN,
		port:    started.port,
		rootDir: started.rootDir,
		pg:      started.pg,
		prefix:  prefix,
	}, nil
}

func startSharedServerProcess(prefix string) (sharedServerStartResult, error) {
	var lastErr error

	for attempt := 1; attempt <= sharedServerStartAttempts; attempt++ {
		rootDir, err := createSharedServerRootDir("openase-pgtest-" + prefix + "-*")
		if err != nil {
			return sharedServerStartResult{}, fmt.Errorf("create temp dir: %w", err)
		}

		port, err := allocateSharedServerPort()
		if err != nil {
			_ = os.RemoveAll(rootDir)
			return sharedServerStartResult{}, fmt.Errorf("allocate free port: %w", err)
		}

		releaseAssetsLock, err := lockSharedServerAssets()
		if err != nil {
			_ = os.RemoveAll(rootDir)
			return sharedServerStartResult{}, fmt.Errorf("lock shared embedded postgres assets: %w", err)
		}

		pg, err := newPostgresController(rootDir, port)
		if err != nil {
			releaseErr := releaseAssetsLock()
			_ = os.RemoveAll(rootDir)
			if releaseErr != nil {
				return sharedServerStartResult{}, fmt.Errorf("create embedded postgres controller: %w; release shared embedded postgres assets lock: %v", err, releaseErr)
			}
			return sharedServerStartResult{}, fmt.Errorf("create embedded postgres controller: %w", err)
		}

		startErr := pg.Start()
		releaseErr := releaseAssetsLock()
		if startErr == nil && releaseErr != nil {
			startErr = fmt.Errorf("release shared embedded postgres assets lock: %w", releaseErr)
		}
		if startErr != nil {
			lastErr = startErr
			_ = pg.Stop()
			_ = os.RemoveAll(rootDir)
			if isRetryablePortStartupError(startErr) && attempt < sharedServerStartAttempts {
				continue
			}
			return sharedServerStartResult{}, fmt.Errorf("start embedded postgres: %w", startErr)
		}

		return sharedServerStartResult{
			rootDir: rootDir,
			port:    port,
			pg:      pg,
		}, nil
	}

	return sharedServerStartResult{}, fmt.Errorf("start embedded postgres: %w", lastErr)
}

func (s *Server) Port() uint32 {
	if s == nil {
		return 0
	}
	return s.port
}

func (s *Server) NewIsolatedDatabase(t *testing.T) Database {
	t.Helper()

	if s == nil {
		t.Fatal("shared postgres server is not initialized")
	}

	dbName := s.nextDatabaseName()
	return s.newIsolatedDatabase(t, dbName, "")
}

func (s *Server) newIsolatedDatabase(t *testing.T, dbName string, templateName string) Database {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), isolatedDatabaseDDLTimeout)
	defer cancel()

	statement := "CREATE DATABASE " + pq.QuoteIdentifier(dbName)
	if templateName != "" {
		statement += " TEMPLATE " + pq.QuoteIdentifier(templateName)
	}
	if _, err := s.admin.ExecContext(ctx, statement); err != nil {
		if templateName != "" {
			t.Fatalf("create isolated database %s from template %s: %v", dbName, templateName, err)
		}
		t.Fatalf("create isolated database %s: %v", dbName, err)
	}

	t.Cleanup(func() {
		// DROP DATABASE ... WITH (FORCE) can block behind checkpoints on slower CI
		// runners, so keep the cleanup budget comfortably above that window.
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), isolatedDatabaseDDLTimeout)
		defer cleanupCancel()
		if _, err := s.admin.ExecContext(cleanupCtx, "DROP DATABASE IF EXISTS "+pq.QuoteIdentifier(dbName)+" WITH (FORCE)"); err != nil {
			t.Errorf("drop isolated database %s: %v", dbName, err)
		}
	})

	return Database{
		Name: dbName,
		DSN:  fmt.Sprintf("postgres://postgres:postgres@127.0.0.1:%d/%s?sslmode=disable", s.port, dbName),
		Port: s.port,
	}
}

func (s *Server) NewIsolatedEntClient(t *testing.T) *ent.Client {
	t.Helper()

	templateName, err := s.ensureEntTemplateDatabase()
	if err != nil {
		t.Fatalf("prepare ent template database: %v", err)
	}

	database := s.newIsolatedDatabase(t, s.nextDatabaseName(), templateName)
	client, err := ent.Open("postgres", database.DSN)
	if err != nil {
		t.Fatalf("open ent client for %s: %v", database.Name, err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close ent client for %s: %v", database.Name, err)
		}
	})

	return client
}

func (s *Server) ensureEntTemplateDatabase() (string, error) {
	s.templateMu.Lock()
	defer s.templateMu.Unlock()

	if s.templateDatabase != "" {
		return s.templateDatabase, nil
	}

	templateName := s.nextDatabaseName() + "_template"
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if _, err := s.admin.ExecContext(ctx, "CREATE DATABASE "+pq.QuoteIdentifier(templateName)); err != nil {
		return "", fmt.Errorf("create template database %s: %w", templateName, err)
	}

	templateDSN := fmt.Sprintf("postgres://postgres:postgres@127.0.0.1:%d/%s?sslmode=disable", s.port, templateName)
	client, err := ent.Open("postgres", templateDSN)
	if err != nil {
		return "", fmt.Errorf("open ent client for template %s: %w", templateName, err)
	}

	closeClient := true
	defer func() {
		if closeClient {
			_ = client.Close()
		}
	}()

	if err := client.Schema.Create(context.Background()); err != nil {
		return "", fmt.Errorf("create schema for template %s: %w", templateName, err)
	}
	if err := client.Close(); err != nil {
		return "", fmt.Errorf("close ent client for template %s: %w", templateName, err)
	}
	closeClient = false

	s.templateDatabase = templateName
	return templateName, nil
}

func (s *Server) Close() error {
	if s == nil {
		return nil
	}

	var firstErr error
	if s.admin != nil {
		if err := s.admin.Close(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("close admin db: %w", err)
		}
	}
	if s.pg != nil {
		if err := s.pg.Stop(); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("stop embedded postgres: %w", err)
		}
	}
	if s.rootDir != "" {
		if err := os.RemoveAll(s.rootDir); err != nil && firstErr == nil {
			firstErr = fmt.Errorf("remove temp dir: %w", err)
		}
	}

	return firstErr
}

func (s *Server) nextDatabaseName() string {
	sequence := s.nextID.Add(1)
	return fmt.Sprintf("openase_%s_%06d_%s", s.prefix, sequence, strings.ReplaceAll(uuid.NewString()[:8], "-", ""))
}

func sanitizeName(raw string) string {
	var builder strings.Builder
	builder.Grow(len(raw))

	for _, r := range strings.ToLower(raw) {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
		default:
			builder.WriteByte('_')
		}
	}

	value := strings.Trim(builder.String(), "_")
	value = strings.ReplaceAll(value, "__", "_")
	if len(value) > 20 {
		value = value[:20]
	}
	return value
}

func freePort() (uint32, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}

	tcpAddr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		_ = listener.Close()
		return 0, fmt.Errorf("expected TCP address, got %T", listener.Addr())
	}
	if err := listener.Close(); err != nil {
		return 0, fmt.Errorf("close listener: %w", err)
	}
	if tcpAddr.Port < 0 || tcpAddr.Port > math.MaxUint16 {
		return 0, fmt.Errorf("expected TCP port in uint16 range, got %d", tcpAddr.Port)
	}

	return uint32(tcpAddr.Port), nil
}

func isRetryablePortStartupError(err error) bool {
	if err == nil {
		return false
	}

	message := strings.ToLower(err.Error())
	switch {
	case strings.Contains(message, "process already listening on port"):
		return true
	case strings.Contains(message, "address already in use"):
		return true
	case strings.Contains(message, "could not bind"):
		return true
	case strings.Contains(message, "failed to create any tcp/ip sockets"):
		return true
	default:
		return false
	}
}

func sharedServerPaths(rootDir string) (sharedServerPathsResult, error) {
	sharedVersionRoot, err := sharedServerVersionRoot()
	if err != nil {
		return sharedServerPathsResult{}, err
	}
	return sharedServerPathsResult{
		cachePath:    filepath.Join(sharedVersionRoot, "cache"),
		runtimePath:  filepath.Join(rootDir, "runtime"),
		binariesPath: filepath.Join(sharedVersionRoot, "binaries"),
		dataPath:     filepath.Join(rootDir, "data"),
	}, nil
}

func ensureSharedServerBinaryLayout(paths sharedServerPathsResult) error {
	binDir := filepath.Join(paths.binariesPath, "bin")
	pgCtlPath := filepath.Join(binDir, "pg_ctl")
	if _, err := os.Stat(pgCtlPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat shared pg_ctl %s: %w", pgCtlPath, err)
	}

	requiredBinaries := []string{"initdb", "postgres"}
	missing := make([]string, 0, len(requiredBinaries))
	for _, name := range requiredBinaries {
		binaryPath := filepath.Join(binDir, name)
		if _, err := os.Stat(binaryPath); err != nil {
			if os.IsNotExist(err) {
				missing = append(missing, name)
				continue
			}
			return fmt.Errorf("stat shared postgres binary %s: %w", binaryPath, err)
		}
	}
	if len(missing) == 0 {
		return nil
	}

	if err := removeSharedServerPath(paths.binariesPath); err != nil {
		return fmt.Errorf("remove incomplete shared postgres binaries %s: %w", paths.binariesPath, err)
	}
	if err := removeSharedServerPath(paths.cachePath); err != nil {
		return fmt.Errorf("remove incomplete shared postgres cache %s: %w", paths.cachePath, err)
	}

	return nil
}

func sharedServerVersionRoot() (string, error) {
	sharedAssetsRoot, err := resolveSharedServerAssetsRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(sharedAssetsRoot, "postgres-"+string(embeddedpostgres.V16)), nil
}

func lockSharedServerAssets() (func() error, error) {
	sharedVersionRoot, err := sharedServerVersionRoot()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(sharedVersionRoot, 0o750); err != nil {
		return nil, fmt.Errorf("create shared version root %s: %w", sharedVersionRoot, err)
	}

	lockFilePath := filepath.Join(sharedVersionRoot, ".lock")
	// #nosec G304 -- lockFilePath is derived from the test-owned shared assets root.
	lockFile, err := os.OpenFile(lockFilePath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open shared lock file %s: %w", lockFilePath, err)
	}
	if err := flockFile(lockFile, unix.LOCK_EX); err != nil {
		_ = lockFile.Close()
		return nil, fmt.Errorf("acquire shared lock %s: %w", lockFilePath, err)
	}

	return func() error {
		unlockErr := flockFile(lockFile, unix.LOCK_UN)
		closeErr := lockFile.Close()
		switch {
		case unlockErr != nil:
			return fmt.Errorf("unlock %s: %w", lockFilePath, unlockErr)
		case closeErr != nil:
			return fmt.Errorf("close %s: %w", lockFilePath, closeErr)
		default:
			return nil
		}
	}, nil
}

func flockFile(lockFile *os.File, operation int) error {
	if lockFile == nil {
		return fmt.Errorf("lock file is required")
	}

	fd := lockFile.Fd()
	maxInt := uintptr(^uint(0) >> 1)
	if fd > maxInt {
		return fmt.Errorf("lock file descriptor %d exceeds int range", fd)
	}
	return unix.Flock(int(fd), operation)
}
