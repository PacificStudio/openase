package setup

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/google/uuid"
)

type stubDockerRunner struct {
	commands []string
	results  map[string]stubDockerResult
}

type stubDockerResult struct {
	output string
	err    error
}

func (s *stubDockerRunner) Run(_ context.Context, name string, args ...string) (string, error) {
	command := name + " " + strings.Join(args, " ")
	s.commands = append(s.commands, command)
	if result, ok := s.results[command]; ok {
		return result.output, result.err
	}
	return "", nil
}

func TestServiceHelpersDatabasePreparationAndDockerErrors(t *testing.T) {
	homeDir := t.TempDir()
	connector := &stubConnector{}
	dockerRunner := &stubDockerRunner{
		results: map[string]stubDockerResult{
			"/usr/bin/docker info --format {{.ServerVersion}}":                                                                                                    {output: "27.0.0\n"},
			"/usr/bin/docker ps -a --filter name=^/openase-local-postgres$ --format {{.Names}}":                                                                   {output: ""},
			"/usr/bin/docker volume create openase-local-postgres-data":                                                                                           {output: "openase-local-postgres-data\n"},
			"/usr/bin/docker run -d --name openase-local-postgres --restart unless-stopped -e POSTGRES_DB=openase -e POSTGRES_USER=openase -e POSTGRES_PASSWORD=": {output: ""},
		},
	}

	service, err := NewService(Options{
		HomeDir:      homeDir,
		Resolver:     stubResolver{paths: map[string]string{"docker": "/usr/bin/docker"}},
		Connector:    connector,
		Installer:    &stubInstaller{},
		DockerRunner: dockerRunner,
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	if service.homeDir != homeDir || service.resolver == nil || service.connector == nil || service.installer == nil {
		t.Fatalf("NewService() = %+v", service)
	}

	manual, err := service.PrepareDatabase(context.Background(), RawDatabaseSourceInput{
		Type: string(DatabaseSourceManual),
		Manual: &RawDatabaseInput{
			Host: "127.0.0.1",
			Name: "openase",
			User: "openase",
		},
	})
	if err != nil {
		t.Fatalf("PrepareDatabase(manual) error = %v", err)
	}
	if manual.Source != DatabaseSourceManual || !strings.Contains(connector.pingDSN, "sslmode=disable") {
		t.Fatalf("PrepareDatabase(manual) = %+v, ping=%q", manual, connector.pingDSN)
	}

	if _, err := service.PrepareDatabase(context.Background(), RawDatabaseSourceInput{}); err == nil || !strings.Contains(err.Error(), "database.type must be one of manual, docker") {
		t.Fatalf("PrepareDatabase(invalid) error = %v", err)
	}

	if err := service.ensureHomeLayout(); err != nil {
		t.Fatalf("ensureHomeLayout() error = %v", err)
	}
	if got, exists := service.existingConfigPath(); exists || got != service.configPath() {
		t.Fatalf("existingConfigPath() = (%q, %v)", got, exists)
	}
	if err := os.WriteFile(service.configPath(), []byte("configured"), 0o600); err != nil {
		t.Fatalf("WriteFile(configPath) error = %v", err)
	}
	if got, exists := service.existingConfigPath(); !exists || got != service.configPath() {
		t.Fatalf("existingConfigPath() with config = (%q, %v)", got, exists)
	}

	tokenA, err := generateAuthToken()
	if err != nil {
		t.Fatalf("generateAuthToken() error = %v", err)
	}
	tokenB, err := generateAuthToken()
	if err != nil {
		t.Fatalf("generateAuthToken() second error = %v", err)
	}
	if len(tokenA) != 64 || len(tokenB) != 64 || tokenA == tokenB {
		t.Fatalf("generateAuthToken() tokens = %q %q", tokenA, tokenB)
	}

	password, err := generateDatabasePassword()
	if err != nil {
		t.Fatalf("generateDatabasePassword() error = %v", err)
	}
	if len(password) != 48 {
		t.Fatalf("generateDatabasePassword() len = %d", len(password))
	}
}

func TestServerRouteErrorPathsAndRun(t *testing.T) {
	homeDir := t.TempDir()
	connector := &stubConnector{}
	service, err := NewService(Options{
		HomeDir:    homeDir,
		Resolver:   stubResolver{paths: map[string]string{"git": "/usr/bin/git", "codex": "/usr/local/bin/codex"}},
		RunCommand: stubVersionRunner,
		Connector:  connector,
		Installer:  &stubInstaller{},
	})
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	server := NewServer(ServerOptions{
		Host:    "127.0.0.1",
		Port:    freeSetupPort(t),
		Service: service,
	})

	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusTemporaryRedirect || rec.Header().Get("Location") != "/setup" {
		t.Fatalf("GET / = (%d, %q)", rec.Code, rec.Header().Get("Location"))
	}

	rec = httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "openase-setup") {
		t.Fatalf("GET /healthz = (%d, %q)", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/setup/test-database", bytes.NewBufferString("{")))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("POST test-database invalid JSON = %d", rec.Code)
	}

	rawDatabase, err := json.Marshal(RawDatabaseInput{
		Host: "127.0.0.1",
		Name: "openase",
		User: "openase",
	})
	if err != nil {
		t.Fatalf("Marshal(RawDatabaseInput) error = %v", err)
	}
	rec = httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/setup/test-database", bytes.NewReader(rawDatabase)))
	if rec.Code != http.StatusOK || connector.pingDSN == "" {
		t.Fatalf("POST test-database = (%d, %q)", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/setup/complete", bytes.NewBufferString("{")))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("POST complete invalid JSON = %d", rec.Code)
	}

	if err := service.ensureHomeLayout(); err != nil {
		t.Fatalf("ensureHomeLayout() error = %v", err)
	}
	if err := os.WriteFile(service.configPath(), []byte("configured"), 0o600); err != nil {
		t.Fatalf("WriteFile(configPath) error = %v", err)
	}
	rawComplete, err := json.Marshal(RawCompleteRequest{
		Database: RawDatabaseInput{
			Host: "127.0.0.1",
			Name: "openase",
			User: "openase",
		},
	})
	if err != nil {
		t.Fatalf("Marshal(RawCompleteRequest) error = %v", err)
	}
	rec = httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/setup/complete", bytes.NewReader(rawComplete)))
	if rec.Code != http.StatusConflict {
		t.Fatalf("POST complete existing config = (%d, %q)", rec.Code, rec.Body.String())
	}

	server = NewServer(ServerOptions{
		Host:    "300.300.300.300",
		Port:    freeSetupPort(t),
		Service: service,
	})
	if err := server.Run(context.Background()); err == nil {
		t.Fatal("server.Run() expected listener error for invalid host")
	}
}

func TestSetupSelectionHelpersAndDockerErrorClassification(t *testing.T) {
	if got := safeSlug(" Team Alpha/OpenASE . Pilot "); got != "team-alpha-openase---pilot" {
		t.Fatalf("safeSlug() = %q", got)
	}
	if got := safeSlug("   "); got != "workspace" {
		t.Fatalf("safeSlug(blank) = %q", got)
	}
	if got := intPtr(7); got == nil || *got != 7 {
		t.Fatalf("intPtr() = %v", got)
	}

	selected := []AgentOption{{
		Name:        "OpenAI Codex",
		AdapterType: catalogdomain.AgentProviderAdapterTypeCodexAppServer,
	}}
	providers := []catalogdomain.AgentProvider{
		{
			ID:          uuid.MustParse("11111111-1111-1111-1111-111111111111"),
			Name:        "Claude Code",
			AdapterType: catalogdomain.AgentProviderAdapterTypeClaudeCodeCLI,
			Available:   true,
		},
		{
			ID:          uuid.MustParse("22222222-2222-2222-2222-222222222222"),
			Name:        "OpenAI Codex",
			AdapterType: catalogdomain.AgentProviderAdapterTypeCodexAppServer,
			Available:   true,
		},
	}
	if got := selectSetupDefaultProviderID(selected, providers); got == nil || *got != "22222222-2222-2222-2222-222222222222" {
		t.Fatalf("selectSetupDefaultProviderID(selected) = %v", got)
	}
	if got := selectSetupDefaultProviderID(nil, providers[:1]); got == nil || *got != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("selectSetupDefaultProviderID(fallback) = %v", got)
	}
	if got := selectSetupDefaultProviderID(nil, nil); got != nil {
		t.Fatalf("selectSetupDefaultProviderID(nil) = %v", got)
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}
	occupiedPort := listener.Addr().(*net.TCPAddr).Port
	if err := ensureTCPPortAvailable(occupiedPort); err == nil {
		t.Fatalf("ensureTCPPortAvailable(%d) expected error", occupiedPort)
	}
	if err := listener.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if err := classifyDockerCommandError("docker daemon is unavailable", errors.New("permission denied")); err == nil || !strings.Contains(err.Error(), "permission denied") {
		t.Fatalf("classifyDockerCommandError(permission) = %v", err)
	}
	runErr := classifyDockerRunError(DockerDatabaseConfig{
		ContainerName: "openase-local-postgres",
		Port:          15432,
		Image:         "postgres:16-alpine",
	}, errors.New("port is already allocated"))
	if runErr == nil || !strings.Contains(runErr.Error(), "127.0.0.1:15432") {
		t.Fatalf("classifyDockerRunError(port) = %v", runErr)
	}
}

func freeSetupPort(t *testing.T) int {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}
	tcpAddr := listener.Addr().(*net.TCPAddr)
	if err := listener.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	return tcpAddr.Port
}
