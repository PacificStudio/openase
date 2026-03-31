package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entmigrate "github.com/BetterAndBetterII/openase/ent/migrate"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
	// Register ent runtime hooks for generated schema metadata.
	_ "github.com/BetterAndBetterII/openase/ent/runtime"
	"github.com/google/uuid"
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
	ticketservice.InstallRetryTokenHooks(client)

	if err := withSchemaBootstrapLock(ctx, trimmedDSN, func() error {
		if err := reconcileLegacyProjectAccessibleMachineIDs(ctx, trimmedDSN); err != nil {
			return err
		}
		if err := reconcileLegacyAgentProviderMachineIDs(ctx, trimmedDSN); err != nil {
			return err
		}
		if err := client.Schema.Create(
			ctx,
			entmigrate.WithDropColumn(false),
			entmigrate.WithDropIndex(false),
		); err != nil {
			return fmt.Errorf("migrate database schema: %w", err)
		}
		if err := reconcileLegacyTicketStageSchema(ctx, trimmedDSN); err != nil {
			return err
		}
		if err := reconcileLegacyProjectRepoSemantics(ctx, trimmedDSN); err != nil {
			return err
		}
		if err := reconcileLegacyTicketIdentifierIndex(ctx, trimmedDSN); err != nil {
			return err
		}

		return nil
	}); err != nil {
		_ = client.Close()
		return nil, err
	}

	return client, nil
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

func reconcileLegacyProjectAccessibleMachineIDs(ctx context.Context, dsn string) error {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("open database for project accessible machine reconciliation: %w", err)
	}
	defer func() {
		_ = db.Close()
	}()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping database for project accessible machine reconciliation: %w", err)
	}

	var projectTableExists bool
	if err := db.QueryRowContext(
		ctx,
		`SELECT EXISTS (
			SELECT 1
			FROM information_schema.tables
			WHERE table_schema = current_schema()
			  AND table_name = 'projects'
		)`,
	).Scan(&projectTableExists); err != nil {
		return fmt.Errorf("check projects table: %w", err)
	}
	if !projectTableExists {
		return nil
	}

	var columnExists bool
	if err := db.QueryRowContext(
		ctx,
		`SELECT EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_schema = current_schema()
			  AND table_name = 'projects'
			  AND column_name = 'accessible_machine_ids'
		)`,
	).Scan(&columnExists); err != nil {
		return fmt.Errorf("check project accessible machine column: %w", err)
	}
	if !columnExists {
		if _, err := db.ExecContext(
			ctx,
			`ALTER TABLE "projects" ADD COLUMN "accessible_machine_ids" jsonb`,
		); err != nil {
			return fmt.Errorf("add project accessible machine ids column: %w", err)
		}
	}

	if _, err := db.ExecContext(
		ctx,
		`UPDATE "projects" SET "accessible_machine_ids" = '[]'::jsonb WHERE "accessible_machine_ids" IS NULL`,
	); err != nil {
		return fmt.Errorf("backfill project accessible machine ids: %w", err)
	}

	return nil
}

func reconcileLegacyTicketStageSchema(ctx context.Context, dsn string) error {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("open database for ticket stage reconciliation: %w", err)
	}
	defer func() {
		_ = db.Close()
	}()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping database for ticket stage reconciliation: %w", err)
	}

	statusTableExists, err := tableExists(ctx, db, "ticket_status")
	if err != nil {
		return err
	}
	if !statusTableExists {
		return nil
	}

	stageTableExists, err := tableExists(ctx, db, "ticket_stages")
	if err != nil {
		return err
	}
	stageIDExists, err := columnExists(ctx, db, "ticket_status", "stage_id")
	if err != nil {
		return err
	}
	if !stageTableExists && !stageIDExists {
		return nil
	}

	statusCapacityExists, err := columnExists(ctx, db, "ticket_status", "max_active_runs")
	if err != nil {
		return err
	}
	if stageTableExists && stageIDExists && statusCapacityExists {
		if _, err := db.ExecContext(
			ctx,
			`UPDATE "ticket_status" AS status
			SET "max_active_runs" = stage."max_active_runs"
			FROM "ticket_stages" AS stage
			WHERE status."stage_id" = stage."id"
			  AND status."max_active_runs" IS NULL
			  AND stage."max_active_runs" IS NOT NULL`,
		); err != nil {
			return fmt.Errorf("backfill ticket status max_active_runs from stages: %w", err)
		}
	}

	if _, err := db.ExecContext(
		ctx,
		`ALTER TABLE "ticket_status" DROP CONSTRAINT IF EXISTS "ticket_status_ticket_stages_statuses"`,
	); err != nil {
		return fmt.Errorf("drop legacy ticket status stage foreign key: %w", err)
	}
	if _, err := db.ExecContext(
		ctx,
		`DROP INDEX IF EXISTS "ticketstatus_project_id_stage_id_position"`,
	); err != nil {
		return fmt.Errorf("drop legacy ticket status stage index: %w", err)
	}
	if _, err := db.ExecContext(
		ctx,
		`ALTER TABLE "ticket_status" DROP COLUMN IF EXISTS "stage_id"`,
	); err != nil {
		return fmt.Errorf("drop legacy ticket status stage_id column: %w", err)
	}
	if _, err := db.ExecContext(ctx, `DROP TABLE IF EXISTS "ticket_stages"`); err != nil {
		return fmt.Errorf("drop legacy ticket stages table: %w", err)
	}

	return nil
}

func reconcileLegacyTicketIdentifierIndex(ctx context.Context, dsn string) error {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("open database for ticket index reconciliation: %w", err)
	}
	defer func() {
		_ = db.Close()
	}()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping database for ticket index reconciliation: %w", err)
	}

	if _, err := db.ExecContext(ctx, `DROP INDEX IF EXISTS "ticket_identifier"`); err != nil {
		return fmt.Errorf("drop legacy ticket identifier index: %w", err)
	}
	if _, err := db.ExecContext(ctx, `CREATE UNIQUE INDEX IF NOT EXISTS "ticket_project_id_identifier" ON "tickets" ("project_id", "identifier")`); err != nil {
		return fmt.Errorf("create project-scoped ticket identifier index: %w", err)
	}

	return nil
}

func reconcileLegacyProjectRepoSemantics(ctx context.Context, dsn string) error {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("open database for project repo reconciliation: %w", err)
	}
	defer func() {
		_ = db.Close()
	}()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping database for project repo reconciliation: %w", err)
	}

	if _, err := db.ExecContext(
		ctx,
		`UPDATE "project_repos" SET "workspace_dirname" = "name" WHERE COALESCE("workspace_dirname", '') = ''`,
	); err != nil {
		return fmt.Errorf("backfill project repo workspace_dirname defaults: %w", err)
	}

	clonePathExists, err := columnExists(ctx, db, "project_repos", "clone_path")
	if err != nil {
		return err
	}
	if !clonePathExists {
		return nil
	}

	rows, err := db.QueryContext(
		ctx,
		`SELECT pr."id", p."organization_id", pr."name", pr."clone_path", pr."workspace_dirname"
		FROM "project_repos" AS pr
		JOIN "projects" AS p ON p."id" = pr."project_id"
		WHERE pr."clone_path" IS NOT NULL AND BTRIM(pr."clone_path") <> ''`,
	)
	if err != nil {
		return fmt.Errorf("query legacy project repo clone_path rows: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	localMachineByOrg, err := localMachineIDsByOrganization(ctx, db)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	for rows.Next() {
		var (
			projectRepoID    uuid.UUID
			organizationID   uuid.UUID
			repoName         string
			legacyClonePath  string
			workspaceDirname string
		)
		if err := rows.Scan(&projectRepoID, &organizationID, &repoName, &legacyClonePath, &workspaceDirname); err != nil {
			return fmt.Errorf("scan legacy project repo clone_path row: %w", err)
		}

		if legacyWorkspaceDirname, ok := parseLegacyWorkspaceDirname(legacyClonePath); ok {
			if _, err := db.ExecContext(
				ctx,
				`UPDATE "project_repos"
				SET "workspace_dirname" = $1
				WHERE "id" = $2
				  AND (COALESCE("workspace_dirname", '') = '' OR "workspace_dirname" = "name")`,
				legacyWorkspaceDirname,
				projectRepoID,
			); err != nil {
				return fmt.Errorf("backfill project repo workspace_dirname from clone_path: %w", err)
			}
		}

		localPath, ok := parseLegacyMirrorLocalPath(legacyClonePath)
		if !ok {
			continue
		}
		machineID, ok := localMachineByOrg[organizationID]
		if !ok {
			continue
		}

		if _, err := db.ExecContext(
			ctx,
			`INSERT INTO "project_repo_mirrors"
				("id", "project_repo_id", "machine_id", "local_path", "state", "last_verified_at", "created_at", "updated_at")
			VALUES ($1, $2, $3, $4, 'ready', $5, $5, $5)
			ON CONFLICT ("project_repo_id", "machine_id") DO NOTHING`,
			uuid.New(),
			projectRepoID,
			machineID,
			localPath,
			now,
		); err != nil {
			return fmt.Errorf("backfill project repo mirror from clone_path: %w", err)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate legacy project repo clone_path rows: %w", err)
	}

	return nil
}

func tableExists(ctx context.Context, db *sql.DB, tableName string) (bool, error) {
	var exists bool
	if err := db.QueryRowContext(
		ctx,
		`SELECT EXISTS (
			SELECT 1
			FROM information_schema.tables
			WHERE table_schema = current_schema()
			  AND table_name = $1
		)`,
		tableName,
	).Scan(&exists); err != nil {
		return false, fmt.Errorf("check %s table: %w", tableName, err)
	}
	return exists, nil
}

func columnExists(ctx context.Context, db *sql.DB, tableName string, columnName string) (bool, error) {
	var exists bool
	if err := db.QueryRowContext(
		ctx,
		`SELECT EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_schema = current_schema()
			  AND table_name = $1
			  AND column_name = $2
		)`,
		tableName,
		columnName,
	).Scan(&exists); err != nil {
		return false, fmt.Errorf("check %s.%s column: %w", tableName, columnName, err)
	}
	return exists, nil
}

func localMachineIDsByOrganization(ctx context.Context, db *sql.DB) (map[uuid.UUID]uuid.UUID, error) {
	rows, err := db.QueryContext(
		ctx,
		`SELECT "organization_id", "id"
		FROM "machines"
		WHERE "name" = $1`,
		catalogdomain.LocalMachineName,
	)
	if err != nil {
		return nil, fmt.Errorf("query local machines for project repo reconciliation: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	machineIDs := make(map[uuid.UUID]uuid.UUID)
	for rows.Next() {
		var organizationID uuid.UUID
		var machineID uuid.UUID
		if err := rows.Scan(&organizationID, &machineID); err != nil {
			return nil, fmt.Errorf("scan local machine for project repo reconciliation: %w", err)
		}
		machineIDs[organizationID] = machineID
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate local machines for project repo reconciliation: %w", err)
	}

	return machineIDs, nil
}

func parseLegacyWorkspaceDirname(raw string) (string, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || filepath.IsAbs(trimmed) {
		return "", false
	}

	parsed, err := url.Parse(trimmed)
	if err == nil && parsed.Scheme != "" {
		return "", false
	}

	cleaned := strings.TrimPrefix(filepath.ToSlash(filepath.Clean(trimmed)), "./")
	if cleaned == "." || cleaned == "" || strings.HasPrefix(cleaned, "../") || strings.Contains(cleaned, "/../") {
		return "", false
	}

	return cleaned, true
}

func parseLegacyMirrorLocalPath(raw string) (string, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", false
	}
	if filepath.IsAbs(trimmed) {
		return filepath.Clean(trimmed), true
	}

	parsed, err := url.Parse(trimmed)
	if err != nil || parsed.Scheme != "file" {
		return "", false
	}

	repoPath, err := url.PathUnescape(parsed.Path)
	if err != nil || repoPath == "" {
		return "", false
	}
	return filepath.Clean(filepath.FromSlash(repoPath)), true
}

func reconcileLegacyAgentProviderMachineIDs(ctx context.Context, dsn string) error {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("open database for agent provider machine reconciliation: %w", err)
	}
	defer func() {
		_ = db.Close()
	}()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("ping database for agent provider machine reconciliation: %w", err)
	}

	var providerTableExists bool
	if err := db.QueryRowContext(
		ctx,
		`SELECT EXISTS (
			SELECT 1
			FROM information_schema.tables
			WHERE table_schema = current_schema()
			  AND table_name = 'agent_providers'
		)`,
	).Scan(&providerTableExists); err != nil {
		return fmt.Errorf("check agent_providers table: %w", err)
	}
	if !providerTableExists {
		return nil
	}

	var machineTableExists bool
	if err := db.QueryRowContext(
		ctx,
		`SELECT EXISTS (
			SELECT 1
			FROM information_schema.tables
			WHERE table_schema = current_schema()
			  AND table_name = 'machines'
		)`,
	).Scan(&machineTableExists); err != nil {
		return fmt.Errorf("check machines table: %w", err)
	}
	if !machineTableExists {
		return nil
	}

	var columnExists bool
	if err := db.QueryRowContext(
		ctx,
		`SELECT EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_schema = current_schema()
			  AND table_name = 'agent_providers'
			  AND column_name = 'machine_id'
		)`,
	).Scan(&columnExists); err != nil {
		return fmt.Errorf("check agent provider machine_id column: %w", err)
	}
	if !columnExists {
		if _, err := db.ExecContext(
			ctx,
			`ALTER TABLE "agent_providers" ADD COLUMN "machine_id" uuid NULL`,
		); err != nil {
			return fmt.Errorf("add agent provider machine_id column: %w", err)
		}
	}

	if _, err := db.ExecContext(
		ctx,
		`UPDATE "agent_providers" AS ap
		SET "machine_id" = m."id"
		FROM "machines" AS m
		WHERE ap."machine_id" IS NULL
		  AND m."organization_id" = ap."organization_id"
		  AND m."name" = 'local'`,
	); err != nil {
		return fmt.Errorf("backfill agent provider machine ids: %w", err)
	}

	var unresolved int
	if err := db.QueryRowContext(
		ctx,
		`SELECT COUNT(1) FROM "agent_providers" WHERE "machine_id" IS NULL`,
	).Scan(&unresolved); err != nil {
		return fmt.Errorf("count unresolved agent provider machine ids: %w", err)
	}
	if unresolved > 0 {
		return fmt.Errorf("backfill agent provider machine ids: %d providers still missing a local machine binding", unresolved)
	}

	return nil
}
