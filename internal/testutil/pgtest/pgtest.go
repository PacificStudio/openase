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
)

const defaultDatabase = "postgres"

type Server struct {
	admin    *sql.DB
	baseDSN  string
	port     uint32
	rootDir  string
	pg       *embeddedpostgres.EmbeddedPostgres
	prefix   string
	nextID   atomic.Uint64
	schemaMu sync.Mutex
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

	rootDir, err := os.MkdirTemp("", "openase-pgtest-"+prefix+"-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}

	port, err := freePort()
	if err != nil {
		_ = os.RemoveAll(rootDir)
		return nil, fmt.Errorf("allocate free port: %w", err)
	}

	pg := embeddedpostgres.NewDatabase(
		embeddedpostgres.DefaultConfig().
			Version(embeddedpostgres.V16).
			Port(port).
			Username("postgres").
			Password("postgres").
			Database(defaultDatabase).
			RuntimePath(filepath.Join(rootDir, "runtime")).
			BinariesPath(filepath.Join(rootDir, "binaries")).
			DataPath(filepath.Join(rootDir, "data")),
	)
	if err := pg.Start(); err != nil {
		_ = os.RemoveAll(rootDir)
		return nil, fmt.Errorf("start embedded postgres: %w", err)
	}

	baseDSN := fmt.Sprintf("postgres://postgres:postgres@127.0.0.1:%d/%s?sslmode=disable", port, defaultDatabase)
	admin, err := sql.Open("postgres", baseDSN)
	if err != nil {
		_ = pg.Stop()
		_ = os.RemoveAll(rootDir)
		return nil, fmt.Errorf("open admin database: %w", err)
	}
	if err := admin.PingContext(context.Background()); err != nil {
		_ = admin.Close()
		_ = pg.Stop()
		_ = os.RemoveAll(rootDir)
		return nil, fmt.Errorf("ping admin database: %w", err)
	}

	return &Server{
		admin:   admin,
		baseDSN: baseDSN,
		port:    port,
		rootDir: rootDir,
		pg:      pg,
		prefix:  prefix,
	}, nil
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
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if _, err := s.admin.ExecContext(ctx, "CREATE DATABASE "+pq.QuoteIdentifier(dbName)); err != nil {
		t.Fatalf("create isolated database %s: %v", dbName, err)
	}

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 15*time.Second)
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

	database := s.NewIsolatedDatabase(t)
	client, err := ent.Open("postgres", database.DSN)
	if err != nil {
		t.Fatalf("open ent client for %s: %v", database.Name, err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close ent client for %s: %v", database.Name, err)
		}
	})

	s.schemaMu.Lock()
	defer s.schemaMu.Unlock()
	if err := client.Schema.Create(context.Background()); err != nil {
		t.Fatalf("create schema for %s: %v", database.Name, err)
	}

	return client
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
