package database

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entticketstage "github.com/BetterAndBetterII/openase/ent/ticketstage"
	entticketstatus "github.com/BetterAndBetterII/openase/ent/ticketstatus"
	entworkflow "github.com/BetterAndBetterII/openase/ent/workflow"
	ticketservice "github.com/BetterAndBetterII/openase/internal/ticket"
	"github.com/BetterAndBetterII/openase/internal/ticketstatus"
	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/google/uuid"
)

func TestOpenReconcilesLegacyGlobalTicketIdentifierIndex(t *testing.T) {
	t.Helper()

	ctx := context.Background()
	dsn := startEmbeddedPostgres(t)

	bootstrapClient, err := ent.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open bootstrap ent client: %v", err)
	}
	if err := bootstrapClient.Schema.Create(ctx); err != nil {
		t.Fatalf("create bootstrap schema: %v", err)
	}
	if err := bootstrapClient.Close(); err != nil {
		t.Fatalf("close bootstrap ent client: %v", err)
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open sql db: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close sql db: %v", err)
		}
	})

	if _, err := db.ExecContext(ctx, `CREATE UNIQUE INDEX "ticket_identifier" ON "tickets" ("identifier")`); err != nil {
		t.Fatalf("create legacy ticket identifier index: %v", err)
	}

	client, err := Open(ctx, dsn)
	if err != nil {
		t.Fatalf("open runtime database: %v", err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close runtime ent client: %v", err)
		}
	})

	if legacyIndexExists(ctx, t, db, "ticket_identifier") {
		t.Fatal("expected runtime database open to remove legacy global ticket identifier index")
	}
	if !legacyIndexExists(ctx, t, db, "ticket_project_id_identifier") {
		t.Fatal("expected runtime database open to create project-scoped ticket identifier index")
	}

	org, err := client.Organization.Create().
		SetName("Better And Better").
		SetSlug("better-and-better-runtime").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}

	ticketSvc := ticketservice.NewService(client)
	statusSvc := ticketstatus.NewService(client)

	for index := range 2 {
		project, err := client.Project.Create().
			SetOrganizationID(org.ID).
			SetName(fmt.Sprintf("Project %d", index+1)).
			SetSlug(fmt.Sprintf("project-%d", index+1)).
			Save(ctx)
		if err != nil {
			t.Fatalf("create project %d: %v", index+1, err)
		}

		statuses, err := statusSvc.ResetToDefaultTemplate(ctx, project.ID)
		if err != nil {
			t.Fatalf("reset statuses for project %d: %v", index+1, err)
		}
		todoID := findStatusID(t, statuses, "Todo")
		doneID := findStatusID(t, statuses, "Done")
		workflowItem, err := client.Workflow.Create().
			SetProjectID(project.ID).
			SetName(fmt.Sprintf("workflow-%d", index+1)).
			SetType(entworkflow.TypeCoding).
			SetHarnessPath(fmt.Sprintf(".openase/harnesses/%d.md", index+1)).
			AddPickupStatusIDs(todoID).
			AddFinishStatusIDs(doneID).
			Save(ctx)
		if err != nil {
			t.Fatalf("create workflow for project %d: %v", index+1, err)
		}

		ticketItem, err := ticketSvc.Create(ctx, ticketservice.CreateInput{
			ProjectID:  project.ID,
			Title:      fmt.Sprintf("Ticket %d", index+1),
			Priority:   "high",
			Type:       "feature",
			WorkflowID: &workflowItem.ID,
			CreatedBy:  "user:blackbox",
		})
		if err != nil {
			t.Fatalf("create first ticket in project %d: %v", index+1, err)
		}
		if ticketItem.Identifier != "ASE-1" {
			t.Fatalf("expected first ticket in project %d to use ASE-1, got %+v", index+1, ticketItem)
		}
	}
}

func TestOpenBackfillsDefaultTicketStagesForLegacyStatuses(t *testing.T) {
	t.Helper()

	ctx := context.Background()
	dsn := startEmbeddedPostgres(t)

	bootstrapClient, err := ent.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open bootstrap ent client: %v", err)
	}
	if err := bootstrapClient.Schema.Create(ctx); err != nil {
		t.Fatalf("create bootstrap schema: %v", err)
	}

	org, err := bootstrapClient.Organization.Create().
		SetName("Better And Better").
		SetSlug("better-and-better-stage-backfill").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	projectItem, err := bootstrapClient.Project.Create().
		SetOrganizationID(org.ID).
		SetName("OpenASE").
		SetSlug("openase-stage-backfill").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	legacyStatuses := []struct {
		name      string
		position  int
		isDefault bool
	}{
		{name: "Backlog", position: 0, isDefault: true},
		{name: "Todo", position: 1},
		{name: "Done", position: 2},
		{name: "Research", position: 3},
	}
	for _, item := range legacyStatuses {
		if _, err := bootstrapClient.TicketStatus.Create().
			SetProjectID(projectItem.ID).
			SetName(item.name).
			SetColor("#111111").
			SetPosition(item.position).
			SetIsDefault(item.isDefault).
			Save(ctx); err != nil {
			t.Fatalf("create legacy status %s: %v", item.name, err)
		}
	}
	if err := bootstrapClient.Close(); err != nil {
		t.Fatalf("close bootstrap ent client: %v", err)
	}

	client, err := Open(ctx, dsn)
	if err != nil {
		t.Fatalf("open runtime database: %v", err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close runtime ent client: %v", err)
		}
	})

	stages, err := client.TicketStage.Query().
		Where(entticketstage.ProjectIDEQ(projectItem.ID)).
		Order(ent.Asc(entticketstage.FieldPosition)).
		All(ctx)
	if err != nil {
		t.Fatalf("query backfilled stages: %v", err)
	}
	if len(stages) != 4 {
		t.Fatalf("expected 4 backfilled stages, got %+v", stages)
	}

	statuses, err := client.TicketStatus.Query().
		Where(entticketstatus.ProjectIDEQ(projectItem.ID)).
		Order(ent.Asc(entticketstatus.FieldPosition)).
		All(ctx)
	if err != nil {
		t.Fatalf("query statuses after backfill: %v", err)
	}
	if status := findStatusByName(statuses, "Backlog"); status == nil || status.StageID == nil || *status.StageID != findStageIDByKey(stages, "backlog") {
		t.Fatalf("expected Backlog to backfill into backlog stage, got %+v", status)
	}
	if status := findStatusByName(statuses, "Todo"); status == nil || status.StageID == nil || *status.StageID != findStageIDByKey(stages, "backlog") {
		t.Fatalf("expected Todo to backfill into backlog stage, got %+v", status)
	}
	if status := findStatusByName(statuses, "Done"); status == nil || status.StageID == nil || *status.StageID != findStageIDByKey(stages, "done") {
		t.Fatalf("expected Done to backfill into done stage, got %+v", status)
	}
	if status := findStatusByName(statuses, "Research"); status == nil || status.StageID != nil {
		t.Fatalf("expected custom status to remain ungrouped after backfill, got %+v", status)
	}
}

func TestOpenBackfillsNullProjectAccessibleMachineIDsBeforeMigration(t *testing.T) {
	t.Helper()

	ctx := context.Background()
	dsn := startEmbeddedPostgres(t)

	bootstrapClient, err := ent.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open bootstrap ent client: %v", err)
	}
	if err := bootstrapClient.Schema.Create(ctx); err != nil {
		t.Fatalf("create bootstrap schema: %v", err)
	}

	org, err := bootstrapClient.Organization.Create().
		SetName("Better And Better").
		SetSlug("better-and-better-null-access").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	projectItem, err := bootstrapClient.Project.Create().
		SetOrganizationID(org.ID).
		SetName("OpenASE").
		SetSlug("openase-null-access").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	if err := bootstrapClient.Close(); err != nil {
		t.Fatalf("close bootstrap ent client: %v", err)
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open sql db: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close sql db: %v", err)
		}
	})

	if _, err := db.ExecContext(ctx, `ALTER TABLE "projects" ALTER COLUMN "accessible_machine_ids" DROP NOT NULL`); err != nil {
		t.Fatalf("drop project accessible machine ids not null: %v", err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE "projects" SET "accessible_machine_ids" = NULL WHERE "id" = $1`, projectItem.ID); err != nil {
		t.Fatalf("set project accessible machine ids null: %v", err)
	}

	client, err := Open(ctx, dsn)
	if err != nil {
		t.Fatalf("open runtime database: %v", err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close runtime ent client: %v", err)
		}
	})

	projectAfter, err := client.Project.Get(ctx, projectItem.ID)
	if err != nil {
		t.Fatalf("reload project: %v", err)
	}
	if len(projectAfter.AccessibleMachineIds) != 0 {
		t.Fatalf("expected empty accessible machine ids after backfill, got %+v", projectAfter.AccessibleMachineIds)
	}
}

func TestOpenAddsMissingProjectAccessibleMachineIDsBeforeMigration(t *testing.T) {
	t.Helper()

	ctx := context.Background()
	dsn := startEmbeddedPostgres(t)

	bootstrapClient, err := ent.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open bootstrap ent client: %v", err)
	}
	if err := bootstrapClient.Schema.Create(ctx); err != nil {
		t.Fatalf("create bootstrap schema: %v", err)
	}

	org, err := bootstrapClient.Organization.Create().
		SetName("Better And Better").
		SetSlug("better-and-better-missing-access").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	projectItem, err := bootstrapClient.Project.Create().
		SetOrganizationID(org.ID).
		SetName("OpenASE").
		SetSlug("openase-missing-access").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	if err := bootstrapClient.Close(); err != nil {
		t.Fatalf("close bootstrap ent client: %v", err)
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open sql db: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close sql db: %v", err)
		}
	})

	if _, err := db.ExecContext(ctx, `ALTER TABLE "projects" DROP COLUMN "accessible_machine_ids"`); err != nil {
		t.Fatalf("drop project accessible machine ids column: %v", err)
	}

	client, err := Open(ctx, dsn)
	if err != nil {
		t.Fatalf("open runtime database: %v", err)
	}
	t.Cleanup(func() {
		if err := client.Close(); err != nil {
			t.Errorf("close runtime ent client: %v", err)
		}
	})

	projectAfter, err := client.Project.Get(ctx, projectItem.ID)
	if err != nil {
		t.Fatalf("reload project: %v", err)
	}
	if len(projectAfter.AccessibleMachineIds) != 0 {
		t.Fatalf("expected empty accessible machine ids after column add, got %+v", projectAfter.AccessibleMachineIds)
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
	firstErr := make(chan error, 1)
	secondErr := make(chan error, 1)

	go func() {
		firstErr <- withSchemaBootstrapLock(ctx, dsn, func() error {
			close(firstEntered)
			<-releaseFirst
			return nil
		})
	}()

	select {
	case <-firstEntered:
	case err := <-firstErr:
		t.Fatalf("first schema bootstrap lock caller failed before entering critical section: %v", err)
	case <-time.After(waitTimeout):
		t.Fatal("timed out waiting for first schema bootstrap lock holder")
	}

	go func() {
		secondErr <- withSchemaBootstrapLock(ctx, dsn, func() error {
			close(secondEntered)
			return nil
		})
	}()

	select {
	case <-secondEntered:
		t.Fatal("expected second schema bootstrap caller to wait for lock release")
	case err := <-secondErr:
		t.Fatalf("second schema bootstrap lock caller failed before entering critical section: %v", err)
	case <-time.After(500 * time.Millisecond):
	}

	close(releaseFirst)

	select {
	case err := <-firstErr:
		if err != nil {
			t.Fatalf("first schema bootstrap lock caller failed: %v", err)
		}
	case <-time.After(waitTimeout):
		t.Fatal("timed out waiting for first schema bootstrap lock caller to finish")
	}

	select {
	case <-secondEntered:
	case err := <-secondErr:
		t.Fatalf("second schema bootstrap lock caller failed before entering critical section: %v", err)
	case <-time.After(waitTimeout):
		t.Fatal("timed out waiting for second schema bootstrap lock caller to enter")
	}

	select {
	case err := <-secondErr:
		if err != nil {
			t.Fatalf("second schema bootstrap lock caller failed: %v", err)
		}
	case <-time.After(waitTimeout):
		t.Fatal("timed out waiting for second schema bootstrap lock caller to finish")
	}
}

func startEmbeddedPostgres(t *testing.T) string {
	t.Helper()

	port := freePort(t)
	dataDir := t.TempDir()
	pg := embeddedpostgres.NewDatabase(
		embeddedpostgres.DefaultConfig().
			Version(embeddedpostgres.V16).
			Port(port).
			Username("postgres").
			Password("postgres").
			Database("openase").
			RuntimePath(filepath.Join(dataDir, "runtime")).
			BinariesPath(filepath.Join(dataDir, "binaries")).
			DataPath(filepath.Join(dataDir, "data")),
	)
	if err := pg.Start(); err != nil {
		t.Fatalf("start embedded postgres: %v", err)
	}
	t.Cleanup(func() {
		if err := pg.Stop(); err != nil {
			t.Errorf("stop embedded postgres: %v", err)
		}
	})

	return fmt.Sprintf("postgres://postgres:postgres@127.0.0.1:%d/openase?sslmode=disable", port)
}

func legacyIndexExists(ctx context.Context, t *testing.T, db *sql.DB, name string) bool {
	t.Helper()

	var count int
	if err := db.QueryRowContext(
		ctx,
		`SELECT count(*) FROM pg_indexes WHERE schemaname = current_schema() AND indexname = $1`,
		name,
	).Scan(&count); err != nil {
		t.Fatalf("query index %s: %v", name, err)
	}

	return count > 0
}

func findStatusID(t *testing.T, statuses []ticketstatus.Status, name string) uuid.UUID {
	t.Helper()

	for _, status := range statuses {
		if status.Name == name {
			return status.ID
		}
	}
	t.Fatalf("status %q not found in %+v", name, statuses)
	return uuid.UUID{}
}

func findStageIDByKey(stages []*ent.TicketStage, key string) uuid.UUID {
	for _, stage := range stages {
		if stage.Key == key {
			return stage.ID
		}
	}
	return uuid.UUID{}
}

func findStatusByName(statuses []*ent.TicketStatus, name string) *ent.TicketStatus {
	for _, status := range statuses {
		if status.Name == name {
			return status
		}
	}
	return nil
}

func freePort(t *testing.T) uint32 {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("allocate free port: %v", err)
	}

	tcpAddr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		t.Fatalf("expected TCP address, got %T", listener.Addr())
	}
	if err := listener.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}
	if tcpAddr.Port < 0 || tcpAddr.Port > math.MaxUint16 {
		t.Fatalf("expected TCP port in uint16 range, got %d", tcpAddr.Port)
	}

	return uint32(tcpAddr.Port) //nolint:gosec // validated above to fit the TCP port range
}
