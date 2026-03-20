package database

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"net"
	"path/filepath"
	"testing"

	"github.com/BetterAndBetterII/openase/ent"
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
			SetPickupStatusID(todoID).
			SetFinishStatusID(doneID).
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
