package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entmigrate "github.com/BetterAndBetterII/openase/ent/migrate"
	agentplatformdomain "github.com/BetterAndBetterII/openase/internal/domain/agentplatform"
	ticketrepo "github.com/BetterAndBetterII/openase/internal/repo/ticket"
	// Register ent runtime hooks for generated schema metadata.
	_ "github.com/BetterAndBetterII/openase/ent/runtime"
	// Register the PostgreSQL SQL driver used by database/sql and ent.
	_ "github.com/lib/pq"
)

const schemaBootstrapLockKey int64 = 0x6f70656e617365

// Open connects to PostgreSQL and applies the current schema migration set.
func Open(ctx context.Context, dsn string) (*ent.Client, error) {
	trimmedDSN := strings.TrimSpace(dsn)
	if trimmedDSN == "" {
		return nil, errors.New("database.dsn is required for serve and all-in-one modes")
	}

	client, err := ent.Open("postgres", trimmedDSN)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	ticketrepo.InstallRetryTokenHooks(client)

	if err := withSchemaBootstrapLock(ctx, trimmedDSN, func() error {
		if err := applyLegacySchemaCompat(ctx, trimmedDSN); err != nil {
			return err
		}
		if err := client.Schema.Create(
			ctx,
			entmigrate.WithDropColumn(false),
			entmigrate.WithDropIndex(false),
		); err != nil {
			return fmt.Errorf("migrate database schema: %w", err)
		}
		return nil
	}); err != nil {
		_ = client.Close()
		return nil, err
	}

	return client, nil
}

func applyLegacySchemaCompat(ctx context.Context, dsn string) error {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("open database for legacy schema compat: %w", err)
	}
	defer func() {
		_ = db.Close()
	}()

	var hasAgentProviders bool
	if err := db.QueryRowContext(
		ctx,
		`SELECT EXISTS (
			SELECT 1
			FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = 'agent_providers'
		)`,
	).Scan(&hasAgentProviders); err != nil {
		return fmt.Errorf("detect legacy agent provider schema: %w", err)
	}

	if hasAgentProviders {
		statements := []string{
			`ALTER TABLE agent_providers ADD COLUMN IF NOT EXISTS cli_rate_limit jsonb`,
			`ALTER TABLE agent_providers ADD COLUMN IF NOT EXISTS cli_rate_limit_updated_at timestamptz`,
			`ALTER TABLE agent_providers ADD COLUMN IF NOT EXISTS pricing_config jsonb`,
			`ALTER TABLE agent_providers ADD COLUMN IF NOT EXISTS reasoning_effort character varying`,
			`UPDATE agent_providers SET cli_rate_limit = '{}'::jsonb WHERE cli_rate_limit IS NULL`,
			`UPDATE agent_providers SET pricing_config = '{}'::jsonb WHERE pricing_config IS NULL`,
			`ALTER TABLE agent_providers ALTER COLUMN cli_rate_limit SET DEFAULT '{}'::jsonb`,
			`ALTER TABLE agent_providers ALTER COLUMN cli_rate_limit SET NOT NULL`,
			`ALTER TABLE agent_providers ALTER COLUMN pricing_config SET DEFAULT '{}'::jsonb`,
			`ALTER TABLE agent_providers ALTER COLUMN pricing_config SET NOT NULL`,
			`DROP TABLE IF EXISTS issue_connectors`,
		}
		for _, statement := range statements {
			if _, err := db.ExecContext(ctx, statement); err != nil {
				return fmt.Errorf("apply legacy agent provider schema compat: %w", err)
			}
		}
	}

	if err := applyLegacyWorkflowVersionSchemaCompat(ctx, db); err != nil {
		return err
	}
	if err := applyLegacyProjectSchemaCompat(ctx, db); err != nil {
		return err
	}

	return nil
}

func applyLegacyWorkflowVersionSchemaCompat(ctx context.Context, db *sql.DB) error {
	var hasWorkflowVersions bool
	if err := db.QueryRowContext(
		ctx,
		`SELECT EXISTS (
			SELECT 1
			FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = 'workflow_versions'
		)`,
	).Scan(&hasWorkflowVersions); err != nil {
		return fmt.Errorf("detect legacy workflow version schema: %w", err)
	}
	if !hasWorkflowVersions {
		return nil
	}

	statements := []string{
		`ALTER TABLE workflow_versions ADD COLUMN IF NOT EXISTS name character varying`,
		`ALTER TABLE workflow_versions ADD COLUMN IF NOT EXISTS type character varying`,
		`ALTER TABLE workflow_versions ADD COLUMN IF NOT EXISTS role_slug character varying`,
		`ALTER TABLE workflow_versions ADD COLUMN IF NOT EXISTS role_name character varying`,
		`ALTER TABLE workflow_versions ADD COLUMN IF NOT EXISTS role_description text`,
		`ALTER TABLE workflow_versions ADD COLUMN IF NOT EXISTS pickup_status_ids text[]`,
		`ALTER TABLE workflow_versions ADD COLUMN IF NOT EXISTS finish_status_ids text[]`,
		`ALTER TABLE workflow_versions ADD COLUMN IF NOT EXISTS harness_path character varying`,
		`ALTER TABLE workflow_versions ADD COLUMN IF NOT EXISTS hooks jsonb`,
		`ALTER TABLE workflow_versions ADD COLUMN IF NOT EXISTS platform_access_allowed text[]`,
		`ALTER TABLE workflow_versions ADD COLUMN IF NOT EXISTS max_concurrent bigint`,
		`ALTER TABLE workflow_versions ADD COLUMN IF NOT EXISTS max_retry_attempts bigint`,
		`ALTER TABLE workflow_versions ADD COLUMN IF NOT EXISTS timeout_minutes bigint`,
		`ALTER TABLE workflow_versions ADD COLUMN IF NOT EXISTS stall_timeout_minutes bigint`,
		`ALTER TABLE workflow_versions ADD COLUMN IF NOT EXISTS is_active boolean`,
		`UPDATE workflow_versions AS versions
			SET
				name = COALESCE(NULLIF(BTRIM(versions.name), ''), workflows.name),
				type = COALESCE(NULLIF(BTRIM(versions.type), ''), workflows.type),
				harness_path = COALESCE(NULLIF(BTRIM(versions.harness_path), ''), workflows.harness_path),
				hooks = COALESCE(versions.hooks, workflows.hooks, '{}'::jsonb),
				max_concurrent = COALESCE(versions.max_concurrent, workflows.max_concurrent, 0),
				max_retry_attempts = COALESCE(versions.max_retry_attempts, workflows.max_retry_attempts, 3),
				timeout_minutes = COALESCE(versions.timeout_minutes, workflows.timeout_minutes, 60),
				stall_timeout_minutes = COALESCE(versions.stall_timeout_minutes, workflows.stall_timeout_minutes, 5),
				is_active = COALESCE(versions.is_active, workflows.is_active, true)
			FROM workflows
			WHERE versions.workflow_id = workflows.id`,
		`ALTER TABLE workflow_versions ALTER COLUMN hooks SET DEFAULT '{}'::jsonb`,
		`ALTER TABLE workflow_versions ALTER COLUMN max_concurrent SET DEFAULT 0`,
		`ALTER TABLE workflow_versions ALTER COLUMN max_retry_attempts SET DEFAULT 3`,
		`ALTER TABLE workflow_versions ALTER COLUMN timeout_minutes SET DEFAULT 60`,
		`ALTER TABLE workflow_versions ALTER COLUMN stall_timeout_minutes SET DEFAULT 5`,
		`ALTER TABLE workflow_versions ALTER COLUMN is_active SET DEFAULT true`,
		`ALTER TABLE workflow_versions ALTER COLUMN name SET NOT NULL`,
		`ALTER TABLE workflow_versions ALTER COLUMN type SET NOT NULL`,
		`ALTER TABLE workflow_versions ALTER COLUMN harness_path SET NOT NULL`,
		`ALTER TABLE workflow_versions ALTER COLUMN hooks SET NOT NULL`,
		`ALTER TABLE workflow_versions ALTER COLUMN max_concurrent SET NOT NULL`,
		`ALTER TABLE workflow_versions ALTER COLUMN max_retry_attempts SET NOT NULL`,
		`ALTER TABLE workflow_versions ALTER COLUMN timeout_minutes SET NOT NULL`,
		`ALTER TABLE workflow_versions ALTER COLUMN stall_timeout_minutes SET NOT NULL`,
		`ALTER TABLE workflow_versions ALTER COLUMN is_active SET NOT NULL`,
	}
	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("apply legacy workflow version schema compat: %w", err)
		}
	}

	return nil
}

func applyLegacyProjectSchemaCompat(ctx context.Context, db *sql.DB) error {
	var hasProjects bool
	if err := db.QueryRowContext(
		ctx,
		`SELECT EXISTS (
			SELECT 1
			FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = 'projects'
		)`,
	).Scan(&hasProjects); err != nil {
		return fmt.Errorf("detect legacy project schema: %w", err)
	}
	if !hasProjects {
		return nil
	}

	defaultProjectAIScopesJSON, err := json.Marshal(
		agentplatformdomain.SupportedScopesForPrincipalKind(agentplatformdomain.PrincipalKindProjectConversation),
	)
	if err != nil {
		return fmt.Errorf("marshal project ai platform access defaults: %w", err)
	}
	quotedProjectAIScopesJSON := "'" + strings.ReplaceAll(string(defaultProjectAIScopesJSON), "'", "''") + "'::jsonb"

	statements := []string{
		`ALTER TABLE projects ADD COLUMN IF NOT EXISTS github_outbound_credential jsonb`,
		`ALTER TABLE projects ADD COLUMN IF NOT EXISTS github_token_probe jsonb`,
		`ALTER TABLE projects ADD COLUMN IF NOT EXISTS default_agent_provider_id uuid`,
		`ALTER TABLE projects ADD COLUMN IF NOT EXISTS project_ai_platform_access_allowed jsonb`,
		`ALTER TABLE projects ADD COLUMN IF NOT EXISTS accessible_machine_ids jsonb`,
		`ALTER TABLE projects ADD COLUMN IF NOT EXISTS max_concurrent_agents bigint`,
		`ALTER TABLE projects ADD COLUMN IF NOT EXISTS agent_run_summary_prompt text`,
		fmt.Sprintf(`UPDATE projects
			SET
				project_ai_platform_access_allowed = COALESCE(project_ai_platform_access_allowed, %s),
				accessible_machine_ids = COALESCE(accessible_machine_ids, '[]'::jsonb),
				max_concurrent_agents = COALESCE(max_concurrent_agents, 0)`, quotedProjectAIScopesJSON),
		fmt.Sprintf(`ALTER TABLE projects ALTER COLUMN project_ai_platform_access_allowed SET DEFAULT %s`, quotedProjectAIScopesJSON),
		`ALTER TABLE projects ALTER COLUMN project_ai_platform_access_allowed SET NOT NULL`,
		`ALTER TABLE projects ALTER COLUMN accessible_machine_ids SET DEFAULT '[]'::jsonb`,
		`ALTER TABLE projects ALTER COLUMN accessible_machine_ids SET NOT NULL`,
		`ALTER TABLE projects ALTER COLUMN max_concurrent_agents SET DEFAULT 0`,
		`ALTER TABLE projects ALTER COLUMN max_concurrent_agents SET NOT NULL`,
	}
	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("apply legacy project schema compat: %w", err)
		}
	}

	return nil
}

func withSchemaBootstrapLock(ctx context.Context, dsn string, fn func() error) (err error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("open database for schema bootstrap lock: %w", err)
	}
	defer func() {
		_ = db.Close()
	}()

	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("reserve database connection for schema bootstrap lock: %w", err)
	}
	defer func() {
		_ = conn.Close()
	}()

	if _, err := conn.ExecContext(ctx, `SELECT pg_advisory_lock($1)`, schemaBootstrapLockKey); err != nil {
		return fmt.Errorf("acquire schema bootstrap lock: %w", err)
	}
	defer func() {
		unlockCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if _, unlockErr := conn.ExecContext(unlockCtx, `SELECT pg_advisory_unlock($1)`, schemaBootstrapLockKey); unlockErr != nil {
			releaseErr := fmt.Errorf("release schema bootstrap lock: %w", unlockErr)
			if err == nil {
				err = releaseErr
				return
			}
			err = errors.Join(err, releaseErr)
		}
	}()

	return fn()
}
