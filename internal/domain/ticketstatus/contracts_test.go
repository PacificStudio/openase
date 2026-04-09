package ticketstatus

import (
	"errors"
	"testing"

	"github.com/BetterAndBetterII/openase/internal/domain/ticketing"
	"github.com/google/uuid"
)

func TestTicketStatusContracts(t *testing.T) {
	projectID := uuid.New()
	statusID := uuid.New()
	maxActiveRuns := 3

	position := Optional[int]{Set: true, Value: 4}
	if !position.Set || position.Value != 4 {
		t.Fatalf("Optional[int] = %#v", position)
	}

	status := Status{
		ID:            statusID,
		ProjectID:     projectID,
		Name:          "In Review",
		Stage:         string(ticketing.StatusStageStarted),
		Color:         "#004488",
		Icon:          "review",
		Position:      2,
		ActiveRuns:    1,
		MaxActiveRuns: &maxActiveRuns,
		IsDefault:     false,
		Description:   "Ready for review",
	}
	if status.ProjectID != projectID {
		t.Fatalf("Status.ProjectID = %s, want %s", status.ProjectID, projectID)
	}
	if status.MaxActiveRuns == nil || *status.MaxActiveRuns != maxActiveRuns {
		t.Fatalf("Status.MaxActiveRuns = %v, want %d", status.MaxActiveRuns, maxActiveRuns)
	}

	snapshot := StatusRuntimeSnapshot{
		StatusID:      statusID,
		ProjectID:     projectID,
		Name:          status.Name,
		Stage:         status.Stage,
		Position:      status.Position,
		MaxActiveRuns: status.MaxActiveRuns,
		ActiveRuns:    status.ActiveRuns,
	}
	if snapshot.StatusID != statusID || snapshot.ProjectID != projectID {
		t.Fatalf("StatusRuntimeSnapshot = %#v", snapshot)
	}

	list := ListResult{Statuses: []Status{status}}
	if len(list.Statuses) != 1 || list.Statuses[0].Name != "In Review" {
		t.Fatalf("ListResult.Statuses = %#v", list.Statuses)
	}

	createInput := CreateInput{
		ProjectID:     projectID,
		Name:          "Todo",
		Stage:         ticketing.StatusStageUnstarted,
		Color:         "#f0c419",
		Icon:          "todo",
		Position:      Optional[int]{Set: true, Value: 1},
		MaxActiveRuns: &maxActiveRuns,
		IsDefault:     true,
		Description:   "Default pickup lane",
	}
	if !createInput.Position.Set || createInput.Position.Value != 1 {
		t.Fatalf("CreateInput.Position = %#v", createInput.Position)
	}

	updateInput := UpdateInput{
		StatusID:      statusID,
		Name:          Optional[string]{Set: true, Value: "Ready"},
		Stage:         Optional[ticketing.StatusStage]{Set: true, Value: ticketing.StatusStageStarted},
		Color:         Optional[string]{Set: true, Value: "#336699"},
		Icon:          Optional[string]{Set: true, Value: "rocket"},
		Position:      Optional[int]{Set: true, Value: 5},
		MaxActiveRuns: Optional[*int]{Set: true, Value: &maxActiveRuns},
		IsDefault:     Optional[bool]{Set: true, Value: false},
		Description:   Optional[string]{Set: true, Value: "Updated"},
	}
	if !updateInput.Name.Set || updateInput.Name.Value != "Ready" {
		t.Fatalf("UpdateInput.Name = %#v", updateInput.Name)
	}
	if !updateInput.Stage.Set || updateInput.Stage.Value != ticketing.StatusStageStarted {
		t.Fatalf("UpdateInput.Stage = %#v", updateInput.Stage)
	}
	if !updateInput.MaxActiveRuns.Set || updateInput.MaxActiveRuns.Value == nil || *updateInput.MaxActiveRuns.Value != maxActiveRuns {
		t.Fatalf("UpdateInput.MaxActiveRuns = %#v", updateInput.MaxActiveRuns)
	}

	deleteResult := DeleteResult{
		DeletedStatusID:     statusID,
		ReplacementStatusID: uuid.New(),
	}
	if deleteResult.DeletedStatusID != statusID {
		t.Fatalf("DeleteResult.DeletedStatusID = %s, want %s", deleteResult.DeletedStatusID, statusID)
	}

	templateStatus := TemplateStatus{
		Name:          "Done",
		Stage:         ticketing.StatusStageCompleted,
		Color:         "#00aa55",
		Icon:          "check",
		Position:      9,
		MaxActiveRuns: nil,
		IsDefault:     false,
		Description:   "Completed work",
	}
	if templateStatus.Stage != ticketing.StatusStageCompleted {
		t.Fatalf("TemplateStatus.Stage = %q", templateStatus.Stage)
	}

	for _, err := range []error{
		ErrProjectNotFound,
		ErrStatusNotFound,
		ErrDuplicateStatusName,
		ErrDefaultStatusRequired,
		ErrDefaultStatusStage,
		ErrCannotDeleteLastStatus,
		ErrReplacementStatusAbsent,
	} {
		if err == nil || err.Error() == "" {
			t.Fatalf("expected non-empty exported error, got %v", err)
		}
		if !errors.Is(err, err) {
			t.Fatalf("expected errors.Is to match %v", err)
		}
	}
}
