package ticketstatus

import (
	"context"
	"errors"
	"testing"

	domain "github.com/BetterAndBetterII/openase/internal/domain/ticketstatus"
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

func TestTicketStatusPureHelpers(t *testing.T) {
	t.Parallel()

	if got := Some("value"); !got.Set || got.Value != "value" {
		t.Fatalf("Some() = %+v, want set value", got)
	}
	if got := DefaultTemplateNames(); len(got) != len(defaultStatusTemplate) || got[0] != "Backlog" || got[len(got)-1] != "Cancelled" {
		t.Fatalf("DefaultTemplateNames() = %v", got)
	}
	if _, err := normalizeStatusStage(""); err != nil {
		t.Fatalf("normalizeStatusStage(empty) error = %v", err)
	}
	if _, err := normalizeStatusStage("nope"); err == nil {
		t.Fatal("normalizeStatusStage(invalid) expected error")
	}
}

type fakeRepository struct {
	listErr    error
	createErr  error
	updateErr  error
	deleteErr  error
	resetErr   error
	projectErr error
	allErr     error
}

func (f fakeRepository) List(ctx context.Context, projectID uuid.UUID) ([]domain.Status, error) {
	return nil, f.listErr
}

func (f fakeRepository) ResolveStatusIDByName(ctx context.Context, projectID uuid.UUID, name string) (uuid.UUID, error) {
	return uuid.Nil, f.listErr
}

func (f fakeRepository) Get(ctx context.Context, statusID uuid.UUID) (domain.Status, error) {
	return domain.Status{}, f.listErr
}

func (f fakeRepository) Create(ctx context.Context, input domain.CreateInput) (domain.Status, error) {
	return domain.Status{}, f.createErr
}

func (f fakeRepository) Update(ctx context.Context, input domain.UpdateInput) (domain.Status, error) {
	return domain.Status{}, f.updateErr
}

func (f fakeRepository) Delete(ctx context.Context, statusID uuid.UUID) (domain.DeleteResult, error) {
	return domain.DeleteResult{}, f.deleteErr
}

func (f fakeRepository) ResetToDefaultTemplate(ctx context.Context, projectID uuid.UUID, template []domain.TemplateStatus) ([]domain.Status, error) {
	return nil, f.resetErr
}

func (f fakeRepository) ListProjectStatusRuntimeSnapshots(ctx context.Context, projectID uuid.UUID) ([]domain.StatusRuntimeSnapshot, error) {
	return nil, f.projectErr
}

func (f fakeRepository) ListStatusRuntimeSnapshots(ctx context.Context) ([]domain.StatusRuntimeSnapshot, error) {
	return nil, f.allErr
}

func TestTicketStatusDelegatesToRepository(t *testing.T) {
	t.Parallel()

	expected := errors.New("boom")
	service := NewService(fakeRepository{
		listErr:    expected,
		createErr:  expected,
		updateErr:  expected,
		deleteErr:  expected,
		resetErr:   expected,
		projectErr: expected,
		allErr:     expected,
	})
	ctx := context.Background()
	projectID := uuid.New()
	statusID := uuid.New()

	if _, err := service.List(ctx, projectID); !errors.Is(err, expected) {
		t.Fatalf("List error = %v, want %v", err, expected)
	}
	if _, err := service.ResolveStatusIDByName(ctx, projectID, "Todo"); !errors.Is(err, expected) {
		t.Fatalf("ResolveStatusIDByName error = %v, want %v", err, expected)
	}
	if _, err := service.Get(ctx, statusID); !errors.Is(err, expected) {
		t.Fatalf("Get error = %v, want %v", err, expected)
	}
	if _, err := service.Delete(ctx, statusID); !errors.Is(err, expected) {
		t.Fatalf("Delete error = %v, want %v", err, expected)
	}
	if _, err := service.ResetToDefaultTemplate(ctx, projectID); !errors.Is(err, expected) {
		t.Fatalf("ResetToDefaultTemplate error = %v, want %v", err, expected)
	}
	if _, err := ListProjectStatusRuntimeSnapshots(ctx, service.repo, projectID); !errors.Is(err, expected) {
		t.Fatalf("ListProjectStatusRuntimeSnapshots error = %v, want %v", err, expected)
	}
	if _, err := ListStatusRuntimeSnapshots(ctx, service.repo); !errors.Is(err, expected) {
		t.Fatalf("ListStatusRuntimeSnapshots error = %v, want %v", err, expected)
	}
}
