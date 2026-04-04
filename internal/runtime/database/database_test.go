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

func TestOpenMigratesLegacyWorkflowVersionSchema(t *testing.T) {
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
	projectID := uuid.New()
	workflowID := uuid.New()
	versionID := uuid.New()

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
		`CREATE TABLE projects (
			id uuid NOT NULL PRIMARY KEY,
			name character varying NOT NULL,
			slug character varying NOT NULL,
			description text NOT NULL DEFAULT '',
			status character varying NOT NULL DEFAULT 'Backlog',
			organization_id uuid NOT NULL,
			agent_id uuid NULL,
			github_installation_id bigint NULL,
			created_at timestamptz NOT NULL DEFAULT now(),
			completed_at timestamptz NULL
		)`,
		`CREATE UNIQUE INDEX project_organization_id_slug ON projects (organization_id, slug)`,
		`CREATE TABLE workflows (
			id uuid NOT NULL PRIMARY KEY,
			name character varying NOT NULL,
			type character varying NOT NULL,
			harness_path character varying NOT NULL,
			hooks jsonb NOT NULL,
			max_concurrent bigint NOT NULL DEFAULT 0,
			max_retry_attempts bigint NOT NULL DEFAULT 3,
			timeout_minutes bigint NOT NULL DEFAULT 60,
			stall_timeout_minutes bigint NOT NULL DEFAULT 5,
			version bigint NOT NULL DEFAULT 1,
			is_active boolean NOT NULL DEFAULT true,
			agent_id uuid NULL,
			project_id uuid NOT NULL,
			current_version_id uuid NULL
		)`,
		`CREATE UNIQUE INDEX workflow_project_id_name ON workflows (project_id, name)`,
		`CREATE TABLE workflow_versions (
			id uuid NOT NULL PRIMARY KEY,
			version bigint NOT NULL,
			content_markdown text NOT NULL,
			content_hash character varying NOT NULL,
			created_by character varying NOT NULL DEFAULT 'system:workflow-service',
			created_at timestamptz NOT NULL,
			workflow_id uuid NOT NULL
		)`,
		`CREATE UNIQUE INDEX workflowversion_workflow_id_version ON workflow_versions (workflow_id, version)`,
		`ALTER TABLE projects
			ADD CONSTRAINT projects_organizations_projects
				FOREIGN KEY (organization_id) REFERENCES organizations(id) ON DELETE NO ACTION`,
		`ALTER TABLE workflows
			ADD CONSTRAINT workflows_projects_workflows
				FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE NO ACTION`,
		`ALTER TABLE workflow_versions
			ADD CONSTRAINT workflow_versions_workflows_versions
				FOREIGN KEY (workflow_id) REFERENCES workflows(id) ON DELETE NO ACTION`,
		`ALTER TABLE workflows
			ADD CONSTRAINT workflows_workflow_versions_current_version
				FOREIGN KEY (current_version_id) REFERENCES workflow_versions(id) ON DELETE SET NULL`,
	}
	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			t.Fatalf("seed legacy workflow version schema with %q: %v", statement, err)
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
		`INSERT INTO projects (id, name, slug, organization_id) VALUES ($1, 'Platform', 'platform', $2)`,
		projectID,
		orgID,
	); err != nil {
		t.Fatalf("insert project: %v", err)
	}
	if _, err := db.ExecContext(
		ctx,
		`INSERT INTO workflows (
			id, name, type, harness_path, hooks, max_concurrent, max_retry_attempts, timeout_minutes, stall_timeout_minutes, version, is_active, project_id
		) VALUES ($1, 'Legacy Coding', 'Fullstack Developer', '.openase/harnesses/coding.md', '{"ticket":{"hooks":[]}}'::jsonb, 2, 4, 90, 11, 1, true, $2)`,
		workflowID,
		projectID,
	); err != nil {
		t.Fatalf("insert workflow: %v", err)
	}
	if _, err := db.ExecContext(
		ctx,
		`INSERT INTO workflow_versions (
			id, version, content_markdown, content_hash, created_by, created_at, workflow_id
		) VALUES ($1, 1, '# Legacy workflow', 'hash-1', 'tester', now(), $2)`,
		versionID,
		workflowID,
	); err != nil {
		t.Fatalf("insert workflow version: %v", err)
	}
	if _, err := db.ExecContext(
		ctx,
		`UPDATE workflows SET current_version_id = $1 WHERE id = $2`,
		versionID,
		workflowID,
	); err != nil {
		t.Fatalf("set current_version_id: %v", err)
	}

	client, err := Open(ctx, dsn)
	if err != nil {
		t.Fatalf("Open() migrating legacy workflow version schema error = %v", err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close runtime ent client: %v", err)
		}
	})

	var (
		name                string
		typeLabel           string
		harnessPath         string
		hooks               string
		maxConcurrent       int
		maxRetryAttempts    int
		timeoutMinutes      int
		stallTimeoutMinutes int
		isActive            bool
	)
	if err := db.QueryRowContext(
		ctx,
		`SELECT
			name,
			type,
			harness_path,
			hooks::text,
			max_concurrent,
			max_retry_attempts,
			timeout_minutes,
			stall_timeout_minutes,
			is_active
		FROM workflow_versions
		WHERE id = $1`,
		versionID,
	).Scan(
		&name,
		&typeLabel,
		&harnessPath,
		&hooks,
		&maxConcurrent,
		&maxRetryAttempts,
		&timeoutMinutes,
		&stallTimeoutMinutes,
		&isActive,
	); err != nil {
		t.Fatalf("query migrated workflow version: %v", err)
	}

	if name != "Legacy Coding" {
		t.Fatalf("name = %q, want Legacy Coding", name)
	}
	if typeLabel != "Fullstack Developer" {
		t.Fatalf("type = %q, want Fullstack Developer", typeLabel)
	}
	if harnessPath != ".openase/harnesses/coding.md" {
		t.Fatalf("harness_path = %q, want .openase/harnesses/coding.md", harnessPath)
	}
	if hooks != `{"ticket": {"hooks": []}}` {
		t.Fatalf("hooks = %q, want migrated workflow hooks", hooks)
	}
	if maxConcurrent != 2 || maxRetryAttempts != 4 || timeoutMinutes != 90 || stallTimeoutMinutes != 11 || !isActive {
		t.Fatalf("unexpected migrated execution settings: max=%d retry=%d timeout=%d stall=%d active=%t", maxConcurrent, maxRetryAttempts, timeoutMinutes, stallTimeoutMinutes, isActive)
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
