package issueconnector

import (
	"context"
	"errors"
	"fmt"

	"github.com/BetterAndBetterII/openase/ent"
	entissueconnector "github.com/BetterAndBetterII/openase/ent/issueconnector"
	entproject "github.com/BetterAndBetterII/openase/ent/project"
	domain "github.com/BetterAndBetterII/openase/internal/domain/issueconnector"
	"github.com/google/uuid"
)

var (
	ErrConnectorNotFound = errors.New("issue connector not found")
	ErrConnectorConflict = errors.New("issue connector conflict")
)

type Repository interface {
	ProjectExists(ctx context.Context, projectID uuid.UUID) (bool, error)
	List(ctx context.Context) ([]domain.IssueConnector, error)
	ListByProject(ctx context.Context, projectID uuid.UUID) ([]domain.IssueConnector, error)
	Create(ctx context.Context, input domain.CreateIssueConnector) (domain.IssueConnector, error)
	Get(ctx context.Context, connectorID uuid.UUID) (domain.IssueConnector, error)
	Update(ctx context.Context, input domain.UpdateIssueConnector) (domain.IssueConnector, error)
	Delete(ctx context.Context, connectorID uuid.UUID) error
	Save(ctx context.Context, connector domain.IssueConnector) error
}

type EntRepository struct {
	client *ent.Client
}

func NewEntRepository(client *ent.Client) *EntRepository {
	return &EntRepository{client: client}
}

func (r *EntRepository) ProjectExists(ctx context.Context, projectID uuid.UUID) (bool, error) {
	exists, err := r.client.Project.Query().Where(entproject.IDEQ(projectID)).Exist(ctx)
	if err != nil {
		return false, fmt.Errorf("check project existence: %w", err)
	}

	return exists, nil
}

func (r *EntRepository) List(ctx context.Context) ([]domain.IssueConnector, error) {
	items, err := r.client.IssueConnector.Query().
		Order(ent.Asc(entissueconnector.FieldCreatedAt), ent.Asc(entissueconnector.FieldID)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list issue connectors: %w", err)
	}

	return mapIssueConnectors(items), nil
}

func (r *EntRepository) ListByProject(ctx context.Context, projectID uuid.UUID) ([]domain.IssueConnector, error) {
	items, err := r.client.IssueConnector.Query().
		Where(entissueconnector.ProjectIDEQ(projectID)).
		Order(ent.Asc(entissueconnector.FieldCreatedAt), ent.Asc(entissueconnector.FieldID)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list project issue connectors: %w", err)
	}

	return mapIssueConnectors(items), nil
}

func (r *EntRepository) Create(ctx context.Context, input domain.CreateIssueConnector) (domain.IssueConnector, error) {
	item, err := r.client.IssueConnector.Create().
		SetProjectID(input.ProjectID).
		SetType(string(input.Type)).
		SetName(input.Name).
		SetStatus(string(input.Status)).
		SetConfig(input.Config).
		SetStats(domain.SyncStats{}).
		Save(ctx)
	if err != nil {
		if ent.IsConstraintError(err) {
			return domain.IssueConnector{}, ErrConnectorConflict
		}
		return domain.IssueConnector{}, fmt.Errorf("create issue connector: %w", err)
	}

	return mapIssueConnector(item), nil
}

func (r *EntRepository) Get(ctx context.Context, connectorID uuid.UUID) (domain.IssueConnector, error) {
	item, err := r.client.IssueConnector.Get(ctx, connectorID)
	if err != nil {
		if ent.IsNotFound(err) {
			return domain.IssueConnector{}, ErrConnectorNotFound
		}
		return domain.IssueConnector{}, fmt.Errorf("get issue connector: %w", err)
	}

	return mapIssueConnector(item), nil
}

func (r *EntRepository) Update(ctx context.Context, input domain.UpdateIssueConnector) (domain.IssueConnector, error) {
	item, err := r.client.IssueConnector.UpdateOneID(input.ID).
		SetProjectID(input.ProjectID).
		SetType(string(input.Type)).
		SetName(input.Name).
		SetStatus(string(input.Status)).
		SetConfig(input.Config).
		Save(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return domain.IssueConnector{}, ErrConnectorNotFound
		}
		if ent.IsConstraintError(err) {
			return domain.IssueConnector{}, ErrConnectorConflict
		}
		return domain.IssueConnector{}, fmt.Errorf("update issue connector: %w", err)
	}

	return mapIssueConnector(item), nil
}

func (r *EntRepository) Delete(ctx context.Context, connectorID uuid.UUID) error {
	if err := r.client.IssueConnector.DeleteOneID(connectorID).Exec(ctx); err != nil {
		if ent.IsNotFound(err) {
			return ErrConnectorNotFound
		}
		return fmt.Errorf("delete issue connector: %w", err)
	}

	return nil
}

func (r *EntRepository) Save(ctx context.Context, connector domain.IssueConnector) error {
	builder := r.client.IssueConnector.UpdateOneID(connector.ID).
		SetProjectID(connector.ProjectID).
		SetType(string(connector.Type)).
		SetName(connector.Name).
		SetStatus(string(connector.Status)).
		SetConfig(connector.Config).
		SetLastError(connector.LastError).
		SetStats(connector.Stats)
	if connector.LastSyncAt != nil {
		syncedAt := connector.LastSyncAt.UTC()
		builder = builder.SetLastSyncAt(syncedAt)
	} else {
		builder = builder.ClearLastSyncAt()
	}

	if err := builder.Exec(ctx); err != nil {
		if ent.IsNotFound(err) {
			return ErrConnectorNotFound
		}
		if ent.IsConstraintError(err) {
			return ErrConnectorConflict
		}
		return fmt.Errorf("save issue connector: %w", err)
	}

	return nil
}

func mapIssueConnectors(items []*ent.IssueConnector) []domain.IssueConnector {
	result := make([]domain.IssueConnector, 0, len(items))
	for _, item := range items {
		result = append(result, mapIssueConnector(item))
	}

	return result
}

func mapIssueConnector(item *ent.IssueConnector) domain.IssueConnector {
	if item == nil {
		return domain.IssueConnector{}
	}

	connector := domain.IssueConnector{
		ID:        item.ID,
		ProjectID: item.ProjectID,
		Type:      domain.Type(item.Type),
		Name:      item.Name,
		Config:    item.Config,
		Status:    domain.Status(item.Status),
		LastError: item.LastError,
		Stats:     item.Stats,
	}
	if item.LastSyncAt != nil {
		syncedAt := item.LastSyncAt.UTC()
		connector.LastSyncAt = &syncedAt
	}

	return connector
}
