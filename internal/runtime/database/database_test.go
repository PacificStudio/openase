package database

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestOpenRejectsEmptyDSN(t *testing.T) {
	t.Helper()

	if _, err := Open(context.Background(), "   "); err == nil || !strings.Contains(err.Error(), "database.dsn is required") {
		t.Fatalf("Open(empty dsn) error = %v", err)
	}
}

func TestOpenFailsForMalformedDSN(t *testing.T) {
	t.Helper()

	if _, err := Open(context.Background(), "postgres://%zz"); err == nil {
		t.Fatal("Open(malformed dsn) expected error")
	}
}

func TestOpenCreatesCurrentSchemaOnFreshDatabase(t *testing.T) {
	t.Helper()

	ctx := context.Background()
	dsn := startEmbeddedPostgres(t)

	client, err := Open(ctx, dsn)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close runtime ent client: %v", err)
		}
	})

	org, err := client.Organization.Create().
		SetName("Acme").
		SetSlug("acme").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}

	if _, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("Payments").
		SetSlug("payments").
		Save(ctx); err != nil {
		t.Fatalf("create project: %v", err)
	}
}

func TestOpenMigratesLegacyAgentProviderCLIRateLimitSchema(t *testing.T) {
	t.Helper()

	ctx := context.Background()
	dsn := startEmbeddedPostgres(t)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open sql db: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close sql db: %v", err)
		}
	})

	orgID := uuid.New()
	machineID := uuid.New()
	providerID := uuid.New()

	statements := []string{
		`CREATE TABLE organizations (
			id uuid NOT NULL PRIMARY KEY,
			name character varying NOT NULL,
			slug character varying NOT NULL,
			status character varying NOT NULL DEFAULT 'active',
			github_outbound_credential jsonb NULL,
			github_token_probe jsonb NULL,
			default_agent_provider_id uuid NULL
		)`,
		`CREATE UNIQUE INDEX organization_slug ON organizations (slug)`,
		`CREATE TABLE machines (
			id uuid NOT NULL PRIMARY KEY,
			name character varying NOT NULL,
			host character varying NOT NULL,
			port bigint NOT NULL DEFAULT 22,
			ssh_user character varying NULL,
			ssh_key_path character varying NULL,
			description text NULL,
			labels text[] NULL,
			status character varying NOT NULL DEFAULT 'maintenance',
			workspace_root character varying NULL,
			agent_cli_path character varying NULL,
			env_vars text[] NULL,
			last_heartbeat_at timestamptz NULL,
			resources jsonb NOT NULL,
			organization_id uuid NOT NULL
		)`,
		`CREATE UNIQUE INDEX machine_organization_id_name ON machines (organization_id, name)`,
		`CREATE TABLE agent_providers (
			id uuid NOT NULL PRIMARY KEY,
			name character varying NOT NULL,
			adapter_type character varying NOT NULL,
			permission_profile character varying NOT NULL DEFAULT 'unrestricted',
			cli_command character varying NOT NULL,
			cli_args text[] NULL,
			auth_config jsonb NOT NULL,
			model_name character varying NOT NULL,
			model_temperature double precision NOT NULL DEFAULT 0,
			model_max_tokens bigint NOT NULL DEFAULT 16384,
			max_parallel_runs bigint NOT NULL DEFAULT 5,
			cost_per_input_token numeric(18,8) NOT NULL DEFAULT 0,
			cost_per_output_token numeric(18,8) NOT NULL DEFAULT 0,
			machine_id uuid NOT NULL,
			organization_id uuid NOT NULL
		)`,
		`CREATE UNIQUE INDEX agentprovider_organization_id_name ON agent_providers (organization_id, name)`,
		`ALTER TABLE machines ADD CONSTRAINT machines_organizations_machines
			FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE NO ACTION`,
		`ALTER TABLE agent_providers
			ADD CONSTRAINT agent_providers_machines_providers
				FOREIGN KEY (machine_id) REFERENCES machines(id) ON DELETE NO ACTION,
			ADD CONSTRAINT agent_providers_organizations_providers
				FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE NO ACTION`,
	}
	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			t.Fatalf("seed legacy schema with %q: %v", statement, err)
		}
	}

	if _, err := db.ExecContext(
		ctx,
		`INSERT INTO organizations (id, name, slug) VALUES ($1, 'Acme', 'acme')`,
		orgID,
	); err != nil {
		t.Fatalf("insert organization: %v", err)
	}
	if _, err := db.ExecContext(
		ctx,
		`INSERT INTO machines (id, name, host, resources, organization_id) VALUES ($1, 'builder-1', '127.0.0.1', '{}'::jsonb, $2)`,
		machineID,
		orgID,
	); err != nil {
		t.Fatalf("insert machine: %v", err)
	}
	if _, err := db.ExecContext(
		ctx,
		`INSERT INTO agent_providers (
			id, name, adapter_type, cli_command, auth_config, model_name, machine_id, organization_id
		) VALUES ($1, 'codex-local', 'codex-app-server', 'codex', '{}'::jsonb, 'gpt-5.4', $2, $3)`,
		providerID,
		machineID,
		orgID,
	); err != nil {
		t.Fatalf("insert legacy provider: %v", err)
	}

	client, err := Open(ctx, dsn)
	if err != nil {
		t.Fatalf("Open() migrating legacy provider schema error = %v", err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close runtime ent client: %v", err)
		}
	})

	var cliRateLimit string
	if err := db.QueryRowContext(
		ctx,
		`SELECT cli_rate_limit::text FROM agent_providers WHERE id = $1`,
		providerID,
	).Scan(&cliRateLimit); err != nil {
		t.Fatalf("query cli_rate_limit: %v", err)
	}
	if cliRateLimit != "{}" {
		t.Fatalf("cli_rate_limit = %q, want {}", cliRateLimit)
	}

	var isNullable string
	if err := db.QueryRowContext(
		ctx,
		`SELECT is_nullable
			FROM information_schema.columns
			WHERE table_schema = 'public' AND table_name = 'agent_providers' AND column_name = 'cli_rate_limit'`,
	).Scan(&isNullable); err != nil {
		t.Fatalf("query cli_rate_limit nullability: %v", err)
	}
	if isNullable != "NO" {
		t.Fatalf("cli_rate_limit is_nullable = %q, want NO", isNullable)
	}
}

func TestWithSchemaBootstrapLockReturnsFunctionError(t *testing.T) {
	t.Helper()

	ctx := context.Background()
	dsn := startEmbeddedPostgres(t)
	wantErr := errors.New("boom")

	err := withSchemaBootstrapLock(ctx, dsn, func() error {
		return wantErr
	})
	if err != wantErr {
		t.Fatalf("withSchemaBootstrapLock() error = %v, want %v", err, wantErr)
	}
}

func TestWithSchemaBootstrapLockExecutesCallbackOnSuccess(t *testing.T) {
	t.Helper()

	ctx := context.Background()
	dsn := startEmbeddedPostgres(t)
	called := false

	if err := withSchemaBootstrapLock(ctx, dsn, func() error {
		called = true
		return nil
	}); err != nil {
		t.Fatalf("withSchemaBootstrapLock() error = %v", err)
	}
	if !called {
		t.Fatal("withSchemaBootstrapLock() expected callback to run")
	}
}

func TestWithSchemaBootstrapLockRejectsMalformedDSN(t *testing.T) {
	t.Helper()

	if err := withSchemaBootstrapLock(context.Background(), "postgres://%zz", func() error {
		return nil
	}); err == nil {
		t.Fatal("withSchemaBootstrapLock(malformed dsn) expected error")
	}
}

func TestWithSchemaBootstrapLockHonorsCanceledContext(t *testing.T) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := withSchemaBootstrapLock(ctx, startEmbeddedPostgres(t), func() error {
		t.Fatal("withSchemaBootstrapLock() should not invoke callback when context is canceled")
		return nil
	})
	if err == nil {
		t.Fatal("withSchemaBootstrapLock(canceled context) expected error")
	}
}

func TestWithSchemaBootstrapLockSerializesConcurrentCallers(t *testing.T) {
	t.Helper()

	ctx := context.Background()
	const waitTimeout = 15 * time.Second

	dsn := startEmbeddedPostgres(t)
	firstEntered := make(chan struct{})
	releaseFirst := make(chan struct{})
	secondEntered := make(chan struct{})
	firstDone := make(chan error, 1)
	secondDone := make(chan error, 1)

	go func() {
		firstDone <- withSchemaBootstrapLock(ctx, dsn, func() error {
			close(firstEntered)
			<-releaseFirst
			return nil
		})
	}()

	select {
	case <-firstEntered:
	case err := <-firstDone:
		t.Fatalf("first schema bootstrap lock caller failed before entering critical section: %v", err)
	case <-time.After(waitTimeout):
		t.Fatal("timed out waiting for first schema bootstrap lock holder")
	}

	go func() {
		secondDone <- withSchemaBootstrapLock(ctx, dsn, func() error {
			close(secondEntered)
			return nil
		})
	}()

	select {
	case <-secondEntered:
		t.Fatal("expected second schema bootstrap caller to wait for lock release")
	case err := <-secondDone:
		t.Fatalf("second schema bootstrap lock caller finished before lock release: %v", err)
	case <-time.After(500 * time.Millisecond):
	}

	close(releaseFirst)

	select {
	case err := <-firstDone:
		if err != nil {
			t.Fatalf("first schema bootstrap lock caller failed: %v", err)
		}
	case <-time.After(waitTimeout):
		t.Fatal("timed out waiting for first schema bootstrap lock caller to finish")
	}

	select {
	case <-secondEntered:
	case err := <-secondDone:
		t.Fatalf("second schema bootstrap lock caller finished before signalling entry: %v", err)
	case <-time.After(waitTimeout):
		t.Fatal("timed out waiting for second schema bootstrap lock caller to enter")
	}

	select {
	case err := <-secondDone:
		if err != nil {
			t.Fatalf("second schema bootstrap lock caller failed: %v", err)
		}
	case <-time.After(waitTimeout):
		t.Fatal("timed out waiting for second schema bootstrap lock caller to finish")
	}
}

func startEmbeddedPostgres(t *testing.T) string {
	t.Helper()

	return testPostgres.NewIsolatedDatabase(t).DSN
}
