package ticketstatus

import (
	"context"
	"errors"
	"testing"

	"github.com/BetterAndBetterII/openase/ent"
	entticketstage "github.com/BetterAndBetterII/openase/ent/ticketstage"
	entticketstatus "github.com/BetterAndBetterII/openase/ent/ticketstatus"
	"github.com/google/uuid"
)

func TestTicketStatusServiceNilClientGuards(t *testing.T) {
	t.Parallel()

	service := NewService(nil)
	ctx := context.Background()
	projectID := uuid.New()
	stageID := uuid.New()
	statusID := uuid.New()

	if _, err := service.List(ctx, projectID); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("List error = %v, want %v", err, ErrUnavailable)
	}
	if _, err := service.ListStages(ctx, projectID); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("ListStages error = %v, want %v", err, ErrUnavailable)
	}
	if _, err := service.CreateStage(ctx, CreateStageInput{ProjectID: projectID, Key: "review", Name: "Review"}); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("CreateStage error = %v, want %v", err, ErrUnavailable)
	}
	if _, err := service.UpdateStage(ctx, UpdateStageInput{StageID: stageID}); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("UpdateStage error = %v, want %v", err, ErrUnavailable)
	}
	if _, err := service.DeleteStage(ctx, stageID); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("DeleteStage error = %v, want %v", err, ErrUnavailable)
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
	if err := service.BackfillDefaultStages(ctx); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("BackfillDefaultStages error = %v, want %v", err, ErrUnavailable)
	}
	if _, err := ListProjectStageRuntimeSnapshots(ctx, nil, projectID); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("ListProjectStageRuntimeSnapshots error = %v, want %v", err, ErrUnavailable)
	}
	if _, err := ListStageRuntimeSnapshots(ctx, nil); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("ListStageRuntimeSnapshots error = %v, want %v", err, ErrUnavailable)
	}
}

func TestTicketStatusHelperSelections(t *testing.T) {
	t.Parallel()

	stageA := uuid.New()
	stageB := uuid.New()
	deletedID := uuid.New()
	defaultSameStageID := uuid.New()
	defaultOtherStageID := uuid.New()
	peerSameStageID := uuid.New()
	fallbackID := uuid.New()

	deleted := &ent.TicketStatus{ID: deletedID, StageID: &stageA}
	defaultSameStage := &ent.TicketStatus{ID: defaultSameStageID, StageID: &stageA, IsDefault: true}
	defaultOtherStage := &ent.TicketStatus{ID: defaultOtherStageID, StageID: &stageB, IsDefault: true}
	peerSameStage := &ent.TicketStatus{ID: peerSameStageID, StageID: &stageA}
	fallback := &ent.TicketStatus{ID: fallbackID}

	tests := []struct {
		name     string
		statuses []*ent.TicketStatus
		wantID   uuid.UUID
		wantErr  error
	}{
		{
			name:     "prefers default in same stage",
			statuses: []*ent.TicketStatus{deleted, defaultOtherStage, defaultSameStage, peerSameStage},
			wantID:   defaultSameStageID,
		},
		{
			name:     "falls back to default in another stage",
			statuses: []*ent.TicketStatus{deleted, defaultOtherStage, peerSameStage},
			wantID:   defaultOtherStageID,
		},
		{
			name:     "falls back to peer in same stage",
			statuses: []*ent.TicketStatus{deleted, fallback, peerSameStage},
			wantID:   peerSameStageID,
		},
		{
			name:     "falls back to first remaining status",
			statuses: []*ent.TicketStatus{deleted, fallback},
			wantID:   fallbackID,
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

	stageOneID := uuid.New()
	stageTwoID := uuid.New()
	projectID := uuid.New()
	maxActiveRuns := 4

	stageOne := &ent.TicketStage{
		ID:            stageOneID,
		ProjectID:     projectID,
		Key:           "backlog",
		Name:          "Backlog",
		Position:      0,
		MaxActiveRuns: &maxActiveRuns,
		Description:   "queued",
	}
	stageTwo := &ent.TicketStage{
		ID:        stageTwoID,
		ProjectID: projectID,
		Key:       "review",
		Name:      "Review",
		Position:  2,
	}

	statusTodoID := uuid.New()
	statusDoneID := uuid.New()
	statusUngroupedID := uuid.New()
	statusTodo := &ent.TicketStatus{
		ID:          statusTodoID,
		ProjectID:   projectID,
		StageID:     &stageOneID,
		Name:        "Todo",
		Color:       "#111111",
		Position:    2,
		IsDefault:   true,
		Description: "todo",
		Edges: ent.TicketStatusEdges{
			Stage: stageOne,
		},
	}
	statusDone := &ent.TicketStatus{
		ID:        statusDoneID,
		ProjectID: projectID,
		StageID:   &stageTwoID,
		Name:      "Done",
		Color:     "#222222",
		Position:  1,
	}
	statusUngrouped := &ent.TicketStatus{
		ID:        statusUngroupedID,
		ProjectID: projectID,
		Name:      "Unsorted",
		Color:     "#333333",
		Position:  1,
	}
	statusTemplate := &ent.TicketStatus{Name: "Backlog"}

	if got := Some("value"); !got.Set || got.Value != "value" {
		t.Fatalf("Some() = %+v, want set value", got)
	}
	if got := nextStagePosition([]*ent.TicketStage{stageOne, stageTwo}); got != 3 {
		t.Fatalf("nextStagePosition = %d, want 3", got)
	}
	if got := nextStatusPosition([]*ent.TicketStatus{statusTodo, statusDone, statusUngrouped}); got != 3 {
		t.Fatalf("nextStatusPosition = %d, want 3", got)
	}
	if !hasDefault([]*ent.TicketStatus{statusUngrouped, statusTodo}) {
		t.Fatal("hasDefault() = false, want true")
	}
	if !hasTemplateStatus([]*ent.TicketStatus{statusUngrouped, statusTemplate}) {
		t.Fatal("hasTemplateStatus() = false, want true")
	}

	replacedIDs, changed := replaceWorkflowStatusBinding(
		[]*ent.TicketStatus{
			{ID: statusTodoID},
			{ID: statusDoneID},
			{ID: statusTodoID},
		},
		statusTodoID,
		statusDoneID,
	)
	if !changed {
		t.Fatal("replaceWorkflowStatusBinding changed = false, want true")
	}
	if len(replacedIDs) != 1 || replacedIDs[0] != statusDoneID {
		t.Fatalf("replaceWorkflowStatusBinding ids = %v, want [%s]", replacedIDs, statusDoneID)
	}

	stageSnapshots := buildStageRuntimeSnapshots([]*ent.TicketStage{stageOne, stageTwo}, map[uuid.UUID]int{
		stageOneID: 2,
		stageTwoID: 1,
	})
	if got := stageActiveRunsByID(stageSnapshots); got[stageOneID] != 2 || got[stageTwoID] != 1 {
		t.Fatalf("stageActiveRunsByID = %v, want backlog=2 review=1", got)
	}
	if got, err := countStageActiveRunsFromStatusCounts(context.Background(), nil, nil); err != nil || len(got) != 0 {
		t.Fatalf("countStageActiveRunsFromStatusCounts(nil) = %v, %v; want empty nil", got, err)
	}

	result := buildListResult(
		[]*ent.TicketStage{stageOne, stageTwo},
		[]*ent.TicketStatus{statusUngrouped, statusTodo, statusDone},
		stageSnapshots,
	)
	if len(result.Stages) != 2 || result.Stages[0].ActiveRuns != 2 || result.Stages[1].ActiveRuns != 1 {
		t.Fatalf("buildListResult stages = %+v", result.Stages)
	}
	if len(result.Statuses) != 3 || result.Statuses[0].Name != "Todo" || result.Statuses[1].Name != "Done" || result.Statuses[2].Name != "Unsorted" {
		t.Fatalf("buildListResult statuses order = %+v", result.Statuses)
	}
	if len(result.StageGroups) != 3 || result.StageGroups[0].Stage == nil || result.StageGroups[0].Stage.ID != stageOneID {
		t.Fatalf("buildListResult groups = %+v", result.StageGroups)
	}
	if result.StageGroups[2].Stage != nil || len(result.StageGroups[2].Statuses) != 1 || result.StageGroups[2].Statuses[0].ID != statusUngroupedID {
		t.Fatalf("unexpected ungrouped status bucket: %+v", result.StageGroups[2])
	}

	statusModel := mapStatus(statusTodo, map[uuid.UUID]int{stageOneID: 7})
	if statusModel.Stage == nil || statusModel.Stage.ActiveRuns != 7 {
		t.Fatalf("mapStatus stage = %+v, want active runs 7", statusModel.Stage)
	}
	if statusModel.StageID == nil || *statusModel.StageID != stageOneID {
		t.Fatalf("mapStatus stage id = %+v, want %s", statusModel.StageID, stageOneID)
	}

	if !sameStage(&stageOneID, &stageOneID) || sameStage(&stageOneID, &stageTwoID) || !sameStage(nil, nil) || sameStage(&stageOneID, nil) {
		t.Fatal("sameStage produced unexpected comparisons")
	}

	clonedStageID := cloneUUIDPointer(&stageOneID)
	if clonedStageID == nil || *clonedStageID != stageOneID || clonedStageID == &stageOneID {
		t.Fatalf("cloneUUIDPointer = %+v, want copied %s", clonedStageID, stageOneID)
	}
	clonedMaxActiveRuns := cloneIntPointer(&maxActiveRuns)
	if clonedMaxActiveRuns == nil || *clonedMaxActiveRuns != maxActiveRuns || clonedMaxActiveRuns == &maxActiveRuns {
		t.Fatalf("cloneIntPointer = %+v, want copied %d", clonedMaxActiveRuns, maxActiveRuns)
	}

	if got := mapStage(stageOne); got.ActiveRuns != 0 || got.MaxActiveRuns == nil || *got.MaxActiveRuns != maxActiveRuns {
		t.Fatalf("mapStage = %+v", got)
	}
	if got := mapStages([]*ent.TicketStage{stageOne, stageTwo}, map[uuid.UUID]int{stageOneID: 9}); got[0].ActiveRuns != 9 || got[1].ActiveRuns != 0 {
		t.Fatalf("mapStages = %+v", got)
	}
	if got := mapStatuses([]*ent.TicketStatus{statusTodo, statusDone}, map[uuid.UUID]int{stageOneID: 5}); len(got) != 2 || got[0].Stage == nil || got[0].Stage.ActiveRuns != 5 {
		t.Fatalf("mapStatuses = %+v", got)
	}

	if got := templateStageKeySet(); !got["backlog"] || !got["done"] {
		t.Fatalf("templateStageKeySet = %v", got)
	}
	if got := templateNameSet(); !got["Backlog"] || !got["Cancelled"] {
		t.Fatalf("templateNameSet = %v", got)
	}
	if got := DefaultTemplateNames(); len(got) != len(defaultStatusTemplate) || got[0] != "Backlog" || got[len(got)-1] != "Cancelled" {
		t.Fatalf("DefaultTemplateNames = %v", got)
	}
}

func TestEnsureStageBelongsToProjectAllowsNilStageID(t *testing.T) {
	t.Parallel()

	if err := ensureStageBelongsToProject(context.Background(), nil, uuid.New(), nil); err != nil {
		t.Fatalf("ensureStageBelongsToProject(nil) error = %v, want nil", err)
	}
}

func TestMapStatusWithoutLoadedStageLeavesStageNil(t *testing.T) {
	t.Parallel()

	stageID := uuid.New()
	projectID := uuid.New()
	status := &ent.TicketStatus{
		ID:        uuid.New(),
		ProjectID: projectID,
		StageID:   &stageID,
		Name:      "Todo",
		Color:     "#fff",
		Icon:      "list",
		Position:  1,
	}

	got := mapStatus(status, map[uuid.UUID]int{stageID: 99})
	if got.Stage != nil {
		t.Fatalf("mapStatus stage = %+v, want nil when edge not loaded", got.Stage)
	}
	if got.StageID == nil || *got.StageID != stageID {
		t.Fatalf("mapStatus stage id = %+v, want %s", got.StageID, stageID)
	}
}

func TestTicketStatusMetadataHelpers(t *testing.T) {
	t.Parallel()

	if got := NewService(&ent.Client{}); got == nil || got.client == nil {
		t.Fatalf("NewService() = %+v, want non-nil service client", got)
	}

	if got := mapStageWithActiveRuns(&ent.TicketStage{
		ID:        uuid.New(),
		ProjectID: uuid.New(),
		Key:       "review",
		Name:      "Review",
		Position:  3,
	}, 6); got.ActiveRuns != 6 {
		t.Fatalf("mapStageWithActiveRuns = %+v, want active runs 6", got)
	}

	unchangedIDs, changed := replaceWorkflowStatusBinding(
		[]*ent.TicketStatus{{ID: uuid.New()}},
		uuid.New(),
		uuid.New(),
	)
	if changed || len(unchangedIDs) != 1 {
		t.Fatalf("replaceWorkflowStatusBinding unchanged = %v, %v", unchangedIDs, changed)
	}

	if got := mapNotFoundError(errors.New("plain"), ErrStageNotFound); got.Error() != "plain" {
		t.Fatalf("mapNotFoundError = %v, want original error", got)
	}
	if got := mapPersistenceError("create", errors.New("plain")); got.Error() != "create: plain" {
		t.Fatalf("mapPersistenceError = %v, want wrapped error", got)
	}
	if got := cloneUUIDPointer(nil); got != nil {
		t.Fatalf("cloneUUIDPointer(nil) = %+v, want nil", got)
	}
	if got := cloneIntPointer(nil); got != nil {
		t.Fatalf("cloneIntPointer(nil) = %+v, want nil", got)
	}

	statuses := []*ent.TicketStatus{
		{
			ID:        uuid.New(),
			ProjectID: uuid.New(),
			Name:      "b",
			Position:  2,
			StageID:   ptrUUID(uuid.New()),
		},
		{
			ID:        uuid.New(),
			ProjectID: uuid.New(),
			Name:      "a",
			Position:  1,
		},
	}
	stageID := *statuses[0].StageID
	stages := []Stage{{ID: stageID}}
	mapped := mapStatuses(statuses, map[uuid.UUID]int{})
	sortStatusesForBoard(stages, mapped)
	if mapped[0].Name != "b" || mapped[1].Name != "a" {
		t.Fatalf("sortStatusesForBoard = %+v", mapped)
	}
}

func ptrUUID(value uuid.UUID) *uuid.UUID {
	return &value
}

var _ = entticketstage.FieldID
var _ = entticketstatus.FieldID
