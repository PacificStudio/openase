package catalog

import (
	"context"
	"testing"
	"time"

	chatconversationdomain "github.com/BetterAndBetterII/openase/internal/domain/chatconversation"
	"github.com/BetterAndBetterII/openase/internal/domain/ticketing"
)

func TestGetOrganizationDashboardSummaryIncludesProjectConversationCost(t *testing.T) {
	t.Parallel()

	client := openRepoCatalogTestEntClient(t)
	ctx := context.Background()

	org, err := client.Organization.Create().
		SetName("Better And Better").
		SetSlug("better-and-better").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("OpenASE").
		SetSlug("openase").
		SetDescription("Issue-driven automation").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	now := time.Now().UTC()
	if _, err := client.ActivityEvent.Create().
		SetProjectID(project.ID).
		SetEventType(ticketing.CostRecordedEventType).
		SetMessage("").
		SetMetadata(map[string]any{"cost_usd": 0.21}).
		SetCreatedAt(now).
		Save(ctx); err != nil {
		t.Fatalf("create ticket cost event: %v", err)
	}
	if _, err := client.ActivityEvent.Create().
		SetProjectID(project.ID).
		SetEventType(chatconversationdomain.CostRecordedEventType).
		SetMessage("").
		SetMetadata(map[string]any{"cost_usd": 0.34}).
		SetCreatedAt(now).
		Save(ctx); err != nil {
		t.Fatalf("create project conversation cost event: %v", err)
	}

	repo := NewEntRepository(client)
	summary, err := repo.GetOrganizationDashboardSummary(ctx, org.ID)
	if err != nil {
		t.Fatalf("GetOrganizationDashboardSummary() error = %v", err)
	}
	if summary.TodayCost != 0.55 {
		t.Fatalf("summary.TodayCost = %.2f, want 0.55", summary.TodayCost)
	}
	if len(summary.Projects) != 1 {
		t.Fatalf("project summary count = %d, want 1", len(summary.Projects))
	}
	if summary.Projects[0].TodayCost != 0.55 {
		t.Fatalf("project TodayCost = %.2f, want 0.55", summary.Projects[0].TodayCost)
	}
}
