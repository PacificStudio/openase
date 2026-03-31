package ticketstatus

import (
	"context"
	"errors"
	"testing"

	"github.com/BetterAndBetterII/openase/ent"
	"github.com/google/uuid"
)

func TestTicketStatusServiceNilClientGuards(t *testing.T) {
	t.Parallel()

	service := NewService(nil)
	ctx := context.Background()
	projectID := uuid.New()
	statusID := uuid.New()

	if _, err := service.List(ctx, projectID); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("List error = %v, want %v", err, ErrUnavailable)
	}
	if _, err := service.Create(ctx, CreateInput{ProjectID: projectID, Name: "Todo", Color: "#fff"}); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Create error = %v, want %v", err, ErrUnavailable)
	}
	if _, err := service.Update(ctx, UpdateInput{StatusID: statusID}); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Update error = %v, want %v", err, ErrUnavailable)
	}
	if _, err := service.Delete(ctx, statusID); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("Delete error = %v, want %v", err, ErrUnavailable)
	}
	if _, err := service.ResetToDefaultTemplate(ctx, projectID); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("ResetToDefaultTemplate error = %v, want %v", err, ErrUnavailable)
	}
	if _, err := ListProjectStatusRuntimeSnapshots(ctx, nil, projectID); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("ListProjectStatusRuntimeSnapshots error = %v, want %v", err, ErrUnavailable)
	}
	if _, err := ListStatusRuntimeSnapshots(ctx, nil); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("ListStatusRuntimeSnapshots error = %v, want %v", err, ErrUnavailable)
	}
}

func TestTicketStatusHelperSelections(t *testing.T) {
	t.Parallel()

	deletedID := uuid.New()
	defaultID := uuid.New()
	peerID := uuid.New()

	deleted := &ent.TicketStatus{ID: deletedID}
	defaultStatus := &ent.TicketStatus{ID: defaultID, IsDefault: true}
	peer := &ent.TicketStatus{ID: peerID}

	tests := []struct {
		name     string
		statuses []*ent.TicketStatus
		wantID   uuid.UUID
		wantErr  error
	}{
		{
			name:     "prefers default replacement",
			statuses: []*ent.TicketStatus{deleted, peer, defaultStatus},
			wantID:   defaultID,
		},
		{
			name:     "falls back to first remaining status",
			statuses: []*ent.TicketStatus{deleted, peer},
			wantID:   peerID,
		},
		{
			name:     "rejects deleting last status",
			statuses: []*ent.TicketStatus{deleted},
			wantErr:  ErrCannotDeleteLastStatus,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := selectReplacementStatus(tt.statuses, deleted)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("selectReplacementStatus error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				return
			}
			if got == nil || got.ID != tt.wantID {
				t.Fatalf("selectReplacementStatus returned %+v, want ID %s", got, tt.wantID)
			}
		})
	}
}

func TestTicketStatusPureHelpers(t *testing.T) {
	t.Parallel()

	projectID := uuid.New()
	statusOneID := uuid.New()
	statusTwoID := uuid.New()
	maxActiveRuns := 4

	statusOne := &ent.TicketStatus{
		ID:            statusOneID,
		ProjectID:     projectID,
		Name:          "Todo",
		Color:         "#111111",
		Position:      2,
		MaxActiveRuns: &maxActiveRuns,
		IsDefault:     true,
		Description:   "todo",
	}
	statusTwo := &ent.TicketStatus{
		ID:        statusTwoID,
		ProjectID: projectID,
		Name:      "Done",
		Color:     "#222222",
		Position:  1,
	}

	if got := Some("value"); !got.Set || got.Value != "value" {
		t.Fatalf("Some() = %+v, want set value", got)
	}
	if got := nextStatusPosition([]*ent.TicketStatus{statusOne, statusTwo}); got != 3 {
		t.Fatalf("nextStatusPosition = %d, want 3", got)
	}
	if !hasDefault([]*ent.TicketStatus{statusTwo, statusOne}) {
		t.Fatal("hasDefault() = false, want true")
	}

	replacedIDs, changed := replaceWorkflowStatusBinding(
		[]*ent.TicketStatus{{ID: statusOneID}, {ID: statusTwoID}, {ID: statusOneID}},
		statusOneID,
		statusTwoID,
	)
	if !changed {
		t.Fatal("replaceWorkflowStatusBinding changed = false, want true")
	}
	if len(replacedIDs) != 1 || replacedIDs[0] != statusTwoID {
		t.Fatalf("replaceWorkflowStatusBinding ids = %v, want [%s]", replacedIDs, statusTwoID)
	}

	snapshots := buildStatusRuntimeSnapshots([]*ent.TicketStatus{statusOne, statusTwo}, map[uuid.UUID]int{
		statusOneID: 2,
		statusTwoID: 1,
	})
	if got := runtimeCountMap(snapshots); got[statusOneID] != 2 || got[statusTwoID] != 1 {
		t.Fatalf("runtimeCountMap = %v, want todo=2 done=1", got)
	}

	result := buildListResult(
		[]*ent.TicketStatus{statusTwo, statusOne},
		map[uuid.UUID]int{statusOneID: 3},
	)
	if len(result.Statuses) != 2 {
		t.Fatalf("buildListResult statuses = %+v", result.Statuses)
	}
	if result.Statuses[0].Name != "Done" || result.Statuses[1].Name != "Todo" {
		t.Fatalf("buildListResult order = %+v", result.Statuses)
	}
	if result.Statuses[1].ActiveRuns != 3 || result.Statuses[1].MaxActiveRuns == nil || *result.Statuses[1].MaxActiveRuns != 4 {
		t.Fatalf("buildListResult todo status = %+v", result.Statuses[1])
	}

	statusModel := mapStatus(statusOne, 7)
	if statusModel.ActiveRuns != 7 {
		t.Fatalf("mapStatus active_runs = %d, want 7", statusModel.ActiveRuns)
	}

	if got := cloneIntPointer(&maxActiveRuns); got == nil || *got != 4 {
		t.Fatalf("cloneIntPointer() = %+v, want 4", got)
	}
	if cloneIntPointer(nil) != nil {
		t.Fatal("cloneIntPointer(nil) should stay nil")
	}

	if got := DefaultTemplateNames(); len(got) != len(defaultStatusTemplate) || got[0] != "Backlog" || got[len(got)-1] != "Cancelled" {
		t.Fatalf("DefaultTemplateNames() = %v", got)
	}
}
