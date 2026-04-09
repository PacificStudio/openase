package secrets

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	"github.com/BetterAndBetterII/openase/ent/predicate"
	entproject "github.com/BetterAndBetterII/openase/ent/project"
	entsecret "github.com/BetterAndBetterII/openase/ent/secret"
	entsecretbinding "github.com/BetterAndBetterII/openase/ent/secretbinding"
	entticket "github.com/BetterAndBetterII/openase/ent/ticket"
	entworkflow "github.com/BetterAndBetterII/openase/ent/workflow"
	domain "github.com/BetterAndBetterII/openase/internal/domain/secrets"
	"github.com/google/uuid"
)

var (
	ErrSecretNotFound        = errors.New("secret not found")
	ErrSecretNameConflict    = errors.New("secret name conflict")
	ErrBindingNotFound       = errors.New("secret binding not found")
	ErrBindingConflict       = errors.New("secret binding already exists at this scope")
	ErrBindingTargetNotFound = errors.New("secret binding target not found in project")
)

type ProjectContext struct {
	ProjectID      uuid.UUID
	OrganizationID uuid.UUID
}

type Repository interface {
	GetProjectContext(ctx context.Context, projectID uuid.UUID) (ProjectContext, error)
	ListProjectSecretInventory(ctx context.Context, projectID uuid.UUID) ([]domain.InventorySecret, error)
	ListOrganizationSecretInventory(ctx context.Context, organizationID uuid.UUID) ([]domain.InventorySecret, error)
	ListBindings(ctx context.Context, projectID uuid.UUID) ([]domain.BindingRecord, error)
	CreateSecret(ctx context.Context, item domain.Secret) (domain.Secret, error)
	GetSecret(ctx context.Context, projectID uuid.UUID, secretID uuid.UUID) (domain.Secret, error)
	GetOrganizationSecret(ctx context.Context, organizationID uuid.UUID, secretID uuid.UUID) (domain.Secret, error)
	GetBindingTarget(ctx context.Context, projectID uuid.UUID, scope domain.BindingScopeKind, resourceID uuid.UUID) (domain.BindingTarget, error)
	UpdateSecretMetadata(ctx context.Context, projectID uuid.UUID, secretID uuid.UUID, name string, description string) (domain.Secret, error)
	UpdateOrganizationSecretMetadata(ctx context.Context, organizationID uuid.UUID, secretID uuid.UUID, name string, description string) (domain.Secret, error)
	RotateSecret(ctx context.Context, projectID uuid.UUID, secretID uuid.UUID, value domain.StoredValue) (domain.Secret, error)
	RotateOrganizationSecret(ctx context.Context, organizationID uuid.UUID, secretID uuid.UUID, value domain.StoredValue) (domain.Secret, error)
	DisableSecret(ctx context.Context, projectID uuid.UUID, secretID uuid.UUID, disabledAt time.Time) (domain.Secret, error)
	DisableOrganizationSecret(ctx context.Context, organizationID uuid.UUID, secretID uuid.UUID, disabledAt time.Time) (domain.Secret, error)
	DeleteSecret(ctx context.Context, projectID uuid.UUID, secretID uuid.UUID) error
	DeleteOrganizationSecret(ctx context.Context, organizationID uuid.UUID, secretID uuid.UUID) error
	ListResolutionCandidates(ctx context.Context, projectID uuid.UUID, keys []string, ticketID, workflowID, agentID *uuid.UUID) ([]domain.Candidate, error)
	CreateBinding(ctx context.Context, item domain.Binding) (domain.Binding, error)
	DeleteBinding(ctx context.Context, projectID uuid.UUID, bindingID uuid.UUID) error
}

type EntRepository struct {
	client *ent.Client
}

func NewEntRepository(client *ent.Client) *EntRepository {
	return &EntRepository{client: client}
}

func (r *EntRepository) GetProjectContext(ctx context.Context, projectID uuid.UUID) (ProjectContext, error) {
	item, err := r.client.Project.Query().
		Where(entproject.IDEQ(projectID)).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return ProjectContext{}, fmt.Errorf("project not found: %w", err)
		}
		return ProjectContext{}, fmt.Errorf("load project secret context: %w", err)
	}
	return ProjectContext{ProjectID: item.ID, OrganizationID: item.OrganizationID}, nil
}

func (r *EntRepository) ListProjectSecretInventory(ctx context.Context, projectID uuid.UUID) ([]domain.InventorySecret, error) {
	projectContext, err := r.GetProjectContext(ctx, projectID)
	if err != nil {
		return nil, err
	}
	items, err := r.client.Secret.Query().
		Where(
			entsecret.OrganizationIDEQ(projectContext.OrganizationID),
			entsecret.Or(
				entsecret.ProjectIDEQ(uuid.Nil),
				entsecret.ProjectIDEQ(projectID),
			),
		).
		Order(ent.Asc(entsecret.FieldProjectID), ent.Asc(entsecret.FieldName)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list accessible secrets: %w", err)
	}
	return r.mapInventory(ctx, projectContext.OrganizationID, items)
}

func (r *EntRepository) ListOrganizationSecretInventory(ctx context.Context, organizationID uuid.UUID) ([]domain.InventorySecret, error) {
	items, err := r.client.Secret.Query().
		Where(
			entsecret.OrganizationIDEQ(organizationID),
			entsecret.ProjectIDEQ(uuid.Nil),
		).
		Order(ent.Asc(entsecret.FieldName)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list organization secrets: %w", err)
	}
	return r.mapInventory(ctx, organizationID, items)
}

func (r *EntRepository) mapInventory(ctx context.Context, organizationID uuid.UUID, items []*ent.Secret) ([]domain.InventorySecret, error) {
	usageBySecret, err := r.listUsageBySecretID(ctx, organizationID, collectSecretIDs(items))
	if err != nil {
		return nil, err
	}
	result := make([]domain.InventorySecret, 0, len(items))
	for _, item := range items {
		usage := usageBySecret[item.ID]
		result = append(result, domain.InventorySecret{
			Secret:      mapSecret(item),
			UsageCount:  usage.count,
			UsageScopes: usage.scopes,
		})
	}
	return result, nil
}

func (r *EntRepository) ListBindings(ctx context.Context, projectID uuid.UUID) ([]domain.BindingRecord, error) {
	projectContext, err := r.GetProjectContext(ctx, projectID)
	if err != nil {
		return nil, err
	}
	items, err := r.client.SecretBinding.Query().
		Where(
			entsecretbinding.OrganizationIDEQ(projectContext.OrganizationID),
			entsecretbinding.ProjectIDEQ(projectID),
			entsecretbinding.ScopeKindIn(
				entsecretbinding.ScopeKindWorkflow,
				entsecretbinding.ScopeKindTicket,
			),
		).
		WithSecret().
		Order(
			ent.Asc(entsecretbinding.FieldScopeKind),
			ent.Asc(entsecretbinding.FieldScopeResourceID),
			ent.Asc(entsecretbinding.FieldBindingKey),
		).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list secret bindings: %w", err)
	}
	targets, err := r.loadBindingTargets(ctx, projectID, items)
	if err != nil {
		return nil, err
	}
	records := make([]domain.BindingRecord, 0, len(items))
	for _, item := range items {
		if item.Edges.Secret == nil {
			continue
		}
		target, ok := targets[item.ScopeResourceID]
		if !ok {
			return nil, fmt.Errorf("%w: missing target %s", ErrBindingTargetNotFound, item.ScopeResourceID)
		}
		records = append(records, domain.BindingRecord{
			Binding: mapBinding(item),
			Secret:  mapSecret(item.Edges.Secret),
			Target:  target,
		})
	}
	return records, nil
}

func (r *EntRepository) CreateSecret(ctx context.Context, item domain.Secret) (domain.Secret, error) {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return domain.Secret{}, fmt.Errorf("begin create secret transaction: %w", err)
	}
	created, err := tx.Secret.Create().
		SetOrganizationID(item.OrganizationID).
		SetProjectID(item.ProjectID).
		SetScopeKind(entsecret.ScopeKind(item.Scope)).
		SetName(item.Name).
		SetKind(entsecret.Kind(item.Kind)).
		SetDescription(item.Description).
		SetAlgorithm(item.StoredValue.Algorithm).
		SetKeySource(string(item.StoredValue.KeySource)).
		SetKeyID(item.StoredValue.KeyID).
		SetValuePreview(item.StoredValue.Preview).
		SetNonce(item.StoredValue.Nonce).
		SetCiphertext(item.StoredValue.Ciphertext).
		SetRotatedAt(item.StoredValue.RotatedAt.UTC()).
		Save(ctx)
	if err == nil {
		err = ensureDefaultBinding(ctx, tx, mapSecret(created), item.Name)
	}
	if err != nil {
		_ = tx.Rollback()
		if ent.IsConstraintError(err) {
			return domain.Secret{}, ErrSecretNameConflict
		}
		return domain.Secret{}, fmt.Errorf("create secret: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return domain.Secret{}, fmt.Errorf("commit create secret: %w", err)
	}
	return mapSecret(created), nil
}

func (r *EntRepository) GetSecret(ctx context.Context, projectID uuid.UUID, secretID uuid.UUID) (domain.Secret, error) {
	projectContext, err := r.GetProjectContext(ctx, projectID)
	if err != nil {
		return domain.Secret{}, err
	}
	return r.getSecretByOrganizationAndProjects(ctx, projectContext.OrganizationID, []uuid.UUID{uuid.Nil, projectID}, secretID)
}

func (r *EntRepository) GetOrganizationSecret(ctx context.Context, organizationID uuid.UUID, secretID uuid.UUID) (domain.Secret, error) {
	return r.getSecretByOrganizationAndProjects(ctx, organizationID, []uuid.UUID{uuid.Nil}, secretID)
}

func (r *EntRepository) getSecretByOrganizationAndProjects(ctx context.Context, organizationID uuid.UUID, projectIDs []uuid.UUID, secretID uuid.UUID) (domain.Secret, error) {
	item, err := r.client.Secret.Query().
		Where(
			entsecret.IDEQ(secretID),
			entsecret.OrganizationIDEQ(organizationID),
			entsecret.ProjectIDIn(projectIDs...),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return domain.Secret{}, ErrSecretNotFound
		}
		return domain.Secret{}, fmt.Errorf("get secret: %w", err)
	}
	return mapSecret(item), nil
}

func (r *EntRepository) GetBindingTarget(ctx context.Context, projectID uuid.UUID, scope domain.BindingScopeKind, resourceID uuid.UUID) (domain.BindingTarget, error) {
	switch scope {
	case domain.BindingScopeKindWorkflow:
		item, err := r.client.Workflow.Query().
			Where(
				entworkflow.IDEQ(resourceID),
				entworkflow.ProjectIDEQ(projectID),
			).
			Only(ctx)
		if err != nil {
			if ent.IsNotFound(err) {
				return domain.BindingTarget{}, ErrBindingTargetNotFound
			}
			return domain.BindingTarget{}, fmt.Errorf("get workflow binding target: %w", err)
		}
		return domain.BindingTarget{
			ID:    item.ID,
			Scope: domain.BindingScopeKindWorkflow,
			Name:  item.Name,
		}, nil
	case domain.BindingScopeKindTicket:
		item, err := r.client.Ticket.Query().
			Where(
				entticket.IDEQ(resourceID),
				entticket.ProjectIDEQ(projectID),
			).
			Only(ctx)
		if err != nil {
			if ent.IsNotFound(err) {
				return domain.BindingTarget{}, ErrBindingTargetNotFound
			}
			return domain.BindingTarget{}, fmt.Errorf("get ticket binding target: %w", err)
		}
		return domain.BindingTarget{
			ID:         item.ID,
			Scope:      domain.BindingScopeKindTicket,
			Name:       item.Title,
			Identifier: item.Identifier,
		}, nil
	default:
		return domain.BindingTarget{}, ErrBindingTargetNotFound
	}
}

func (r *EntRepository) UpdateSecretMetadata(ctx context.Context, projectID uuid.UUID, secretID uuid.UUID, name string, description string) (domain.Secret, error) {
	item, err := r.GetSecret(ctx, projectID, secretID)
	if err != nil {
		return domain.Secret{}, err
	}
	return r.updateSecretMetadata(ctx, item, name, description)
}

func (r *EntRepository) UpdateOrganizationSecretMetadata(ctx context.Context, organizationID uuid.UUID, secretID uuid.UUID, name string, description string) (domain.Secret, error) {
	item, err := r.GetOrganizationSecret(ctx, organizationID, secretID)
	if err != nil {
		return domain.Secret{}, err
	}
	return r.updateSecretMetadata(ctx, item, name, description)
}

func (r *EntRepository) updateSecretMetadata(ctx context.Context, item domain.Secret, name string, description string) (domain.Secret, error) {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return domain.Secret{}, fmt.Errorf("begin update secret transaction: %w", err)
	}
	updated, err := tx.Secret.UpdateOneID(item.ID).
		SetName(name).
		SetDescription(description).
		Save(ctx)
	if err == nil {
		err = ensureDefaultBinding(ctx, tx, mapSecret(updated), name)
	}
	if err != nil {
		_ = tx.Rollback()
		if ent.IsConstraintError(err) {
			return domain.Secret{}, ErrSecretNameConflict
		}
		return domain.Secret{}, fmt.Errorf("update secret metadata: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return domain.Secret{}, fmt.Errorf("commit update secret metadata: %w", err)
	}
	return mapSecret(updated), nil
}

func (r *EntRepository) RotateSecret(ctx context.Context, projectID uuid.UUID, secretID uuid.UUID, value domain.StoredValue) (domain.Secret, error) {
	item, err := r.GetSecret(ctx, projectID, secretID)
	if err != nil {
		return domain.Secret{}, err
	}
	updated, err := r.client.Secret.UpdateOneID(item.ID).
		SetAlgorithm(value.Algorithm).
		SetKeySource(string(value.KeySource)).
		SetKeyID(value.KeyID).
		SetValuePreview(value.Preview).
		SetNonce(value.Nonce).
		SetCiphertext(value.Ciphertext).
		SetRotatedAt(value.RotatedAt.UTC()).
		Save(ctx)
	if err != nil {
		return domain.Secret{}, fmt.Errorf("rotate secret: %w", err)
	}
	return mapSecret(updated), nil
}

func (r *EntRepository) RotateOrganizationSecret(ctx context.Context, organizationID uuid.UUID, secretID uuid.UUID, value domain.StoredValue) (domain.Secret, error) {
	item, err := r.GetOrganizationSecret(ctx, organizationID, secretID)
	if err != nil {
		return domain.Secret{}, err
	}
	updated, err := r.client.Secret.UpdateOneID(item.ID).
		SetAlgorithm(value.Algorithm).
		SetKeySource(string(value.KeySource)).
		SetKeyID(value.KeyID).
		SetValuePreview(value.Preview).
		SetNonce(value.Nonce).
		SetCiphertext(value.Ciphertext).
		SetRotatedAt(value.RotatedAt.UTC()).
		Save(ctx)
	if err != nil {
		return domain.Secret{}, fmt.Errorf("rotate organization secret: %w", err)
	}
	return mapSecret(updated), nil
}

func (r *EntRepository) DisableSecret(ctx context.Context, projectID uuid.UUID, secretID uuid.UUID, disabledAt time.Time) (domain.Secret, error) {
	item, err := r.GetSecret(ctx, projectID, secretID)
	if err != nil {
		return domain.Secret{}, err
	}
	updated, err := r.client.Secret.UpdateOneID(item.ID).
		SetDisabledAt(disabledAt.UTC()).
		Save(ctx)
	if err != nil {
		return domain.Secret{}, fmt.Errorf("disable secret: %w", err)
	}
	return mapSecret(updated), nil
}

func (r *EntRepository) DisableOrganizationSecret(ctx context.Context, organizationID uuid.UUID, secretID uuid.UUID, disabledAt time.Time) (domain.Secret, error) {
	item, err := r.GetOrganizationSecret(ctx, organizationID, secretID)
	if err != nil {
		return domain.Secret{}, err
	}
	updated, err := r.client.Secret.UpdateOneID(item.ID).
		SetDisabledAt(disabledAt.UTC()).
		Save(ctx)
	if err != nil {
		return domain.Secret{}, fmt.Errorf("disable organization secret: %w", err)
	}
	return mapSecret(updated), nil
}

func (r *EntRepository) DeleteSecret(ctx context.Context, projectID uuid.UUID, secretID uuid.UUID) error {
	item, err := r.GetSecret(ctx, projectID, secretID)
	if err != nil {
		return err
	}
	return r.deleteSecret(ctx, item.ID)
}

func (r *EntRepository) DeleteOrganizationSecret(ctx context.Context, organizationID uuid.UUID, secretID uuid.UUID) error {
	item, err := r.GetOrganizationSecret(ctx, organizationID, secretID)
	if err != nil {
		return err
	}
	return r.deleteSecret(ctx, item.ID)
}

func (r *EntRepository) deleteSecret(ctx context.Context, secretID uuid.UUID) error {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("begin delete secret transaction: %w", err)
	}
	if _, err := tx.SecretBinding.Delete().Where(entsecretbinding.SecretIDEQ(secretID)).Exec(ctx); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("delete secret bindings: %w", err)
	}
	if err := tx.Secret.DeleteOneID(secretID).Exec(ctx); err != nil {
		_ = tx.Rollback()
		if ent.IsNotFound(err) {
			return ErrSecretNotFound
		}
		return fmt.Errorf("delete secret: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit delete secret: %w", err)
	}
	return nil
}

func (r *EntRepository) ListResolutionCandidates(ctx context.Context, projectID uuid.UUID, keys []string, ticketID, workflowID, agentID *uuid.UUID) ([]domain.Candidate, error) {
	projectContext, err := r.GetProjectContext(ctx, projectID)
	if err != nil {
		return nil, err
	}
	predicates := []bindingSelector{
		{scope: entsecretbinding.ScopeKindOrganization, resourceID: projectContext.OrganizationID, projectID: uuid.Nil},
		{scope: entsecretbinding.ScopeKindProject, resourceID: projectID, projectID: projectID},
	}
	if workflowID != nil {
		predicates = append(predicates, bindingSelector{scope: entsecretbinding.ScopeKindWorkflow, resourceID: *workflowID, projectID: projectID})
	}
	if agentID != nil {
		predicates = append(predicates, bindingSelector{scope: entsecretbinding.ScopeKindAgent, resourceID: *agentID, projectID: projectID})
	}
	if ticketID != nil {
		predicates = append(predicates, bindingSelector{scope: entsecretbinding.ScopeKindTicket, resourceID: *ticketID, projectID: projectID})
	}
	bindingPredicates := make([]predicate.SecretBinding, 0, len(predicates))
	for _, item := range predicates {
		bindingPredicates = append(bindingPredicates, entsecretbinding.And(
			entsecretbinding.ScopeKindEQ(item.scope),
			entsecretbinding.ScopeResourceIDEQ(item.resourceID),
			entsecretbinding.ProjectIDEQ(item.projectID),
		))
	}
	queryPredicates := []predicate.SecretBinding{
		entsecretbinding.OrganizationIDEQ(projectContext.OrganizationID),
		entsecretbinding.Or(bindingPredicates...),
		entsecretbinding.HasSecretWith(
			entsecret.OrganizationIDEQ(projectContext.OrganizationID),
			entsecret.Or(
				entsecret.ProjectIDEQ(uuid.Nil),
				entsecret.ProjectIDEQ(projectID),
			),
		),
	}
	if len(keys) > 0 {
		queryPredicates = append(queryPredicates, entsecretbinding.BindingKeyIn(keys...))
	}
	items, err := r.client.SecretBinding.Query().
		Where(queryPredicates...).
		WithSecret().
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list secret resolution candidates: %w", err)
	}
	result := make([]domain.Candidate, 0, len(items))
	for _, item := range items {
		if item.Edges.Secret == nil {
			continue
		}
		result = append(result, domain.Candidate{
			Binding: mapBinding(item),
			Secret:  mapSecret(item.Edges.Secret),
		})
	}
	return result, nil
}

func (r *EntRepository) CreateBinding(ctx context.Context, item domain.Binding) (domain.Binding, error) {
	created, err := r.client.SecretBinding.Create().
		SetOrganizationID(item.OrganizationID).
		SetProjectID(item.ProjectID).
		SetSecretID(item.SecretID).
		SetScopeKind(entsecretbinding.ScopeKind(item.Scope)).
		SetScopeResourceID(item.ScopeResourceID).
		SetBindingKey(item.BindingKey).
		Save(ctx)
	if err != nil {
		if ent.IsConstraintError(err) {
			return domain.Binding{}, ErrBindingConflict
		}
		return domain.Binding{}, fmt.Errorf("create secret binding: %w", err)
	}
	return mapBinding(created), nil
}

func (r *EntRepository) DeleteBinding(ctx context.Context, projectID uuid.UUID, bindingID uuid.UUID) error {
	count, err := r.client.SecretBinding.Delete().
		Where(
			entsecretbinding.IDEQ(bindingID),
			entsecretbinding.ProjectIDEQ(projectID),
			entsecretbinding.ScopeKindIn(
				entsecretbinding.ScopeKindWorkflow,
				entsecretbinding.ScopeKindTicket,
			),
		).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete secret binding: %w", err)
	}
	if count == 0 {
		return ErrBindingNotFound
	}
	return nil
}

type bindingSelector struct {
	scope      entsecretbinding.ScopeKind
	resourceID uuid.UUID
	projectID  uuid.UUID
}

func (r *EntRepository) loadBindingTargets(ctx context.Context, projectID uuid.UUID, items []*ent.SecretBinding) (map[uuid.UUID]domain.BindingTarget, error) {
	workflowIDs := make([]uuid.UUID, 0)
	ticketIDs := make([]uuid.UUID, 0)
	for _, item := range items {
		switch item.ScopeKind {
		case entsecretbinding.ScopeKindWorkflow:
			workflowIDs = append(workflowIDs, item.ScopeResourceID)
		case entsecretbinding.ScopeKindTicket:
			ticketIDs = append(ticketIDs, item.ScopeResourceID)
		}
	}

	targets := make(map[uuid.UUID]domain.BindingTarget, len(items))
	if len(workflowIDs) > 0 {
		workflows, err := r.client.Workflow.Query().
			Where(
				entworkflow.ProjectIDEQ(projectID),
				entworkflow.IDIn(uniqueUUIDs(workflowIDs)...),
			).
			All(ctx)
		if err != nil {
			return nil, fmt.Errorf("list workflow binding targets: %w", err)
		}
		for _, item := range workflows {
			targets[item.ID] = domain.BindingTarget{
				ID:    item.ID,
				Scope: domain.BindingScopeKindWorkflow,
				Name:  item.Name,
			}
		}
	}
	if len(ticketIDs) > 0 {
		tickets, err := r.client.Ticket.Query().
			Where(
				entticket.ProjectIDEQ(projectID),
				entticket.IDIn(uniqueUUIDs(ticketIDs)...),
			).
			All(ctx)
		if err != nil {
			return nil, fmt.Errorf("list ticket binding targets: %w", err)
		}
		for _, item := range tickets {
			targets[item.ID] = domain.BindingTarget{
				ID:         item.ID,
				Scope:      domain.BindingScopeKindTicket,
				Name:       item.Title,
				Identifier: item.Identifier,
			}
		}
	}
	return targets, nil
}

func mapSecret(item *ent.Secret) domain.Secret {
	return domain.Secret{
		ID:             item.ID,
		OrganizationID: item.OrganizationID,
		ProjectID:      item.ProjectID,
		Scope:          domain.ScopeKind(item.ScopeKind),
		Name:           item.Name,
		Kind:           domain.Kind(item.Kind),
		Description:    item.Description,
		DisabledAt:     cloneTime(item.DisabledAt),
		CreatedAt:      item.CreatedAt.UTC(),
		UpdatedAt:      item.UpdatedAt.UTC(),
		StoredValue: domain.StoredValue{
			Algorithm:  item.Algorithm,
			KeySource:  domain.KeySource(item.KeySource),
			KeyID:      item.KeyID,
			Preview:    item.ValuePreview,
			Nonce:      item.Nonce,
			Ciphertext: item.Ciphertext,
			RotatedAt:  item.RotatedAt.UTC(),
		},
	}
}

func mapBinding(item *ent.SecretBinding) domain.Binding {
	return domain.Binding{
		ID:              item.ID,
		OrganizationID:  item.OrganizationID,
		ProjectID:       item.ProjectID,
		SecretID:        item.SecretID,
		Scope:           domain.BindingScopeKind(item.ScopeKind),
		ScopeResourceID: item.ScopeResourceID,
		BindingKey:      item.BindingKey,
		CreatedAt:       item.CreatedAt.UTC(),
		UpdatedAt:       item.UpdatedAt.UTC(),
	}
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copied := value.UTC()
	return &copied
}

func uniqueUUIDs(items []uuid.UUID) []uuid.UUID {
	if len(items) == 0 {
		return nil
	}
	seen := make(map[uuid.UUID]struct{}, len(items))
	result := make([]uuid.UUID, 0, len(items))
	for _, item := range items {
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	return result
}

type secretUsageSummary struct {
	count  int
	scopes []domain.BindingScopeKind
}

func (r *EntRepository) listUsageBySecretID(ctx context.Context, organizationID uuid.UUID, secretIDs []uuid.UUID) (map[uuid.UUID]secretUsageSummary, error) {
	if len(secretIDs) == 0 {
		return map[uuid.UUID]secretUsageSummary{}, nil
	}
	items, err := r.client.SecretBinding.Query().
		Where(
			entsecretbinding.OrganizationIDEQ(organizationID),
			entsecretbinding.SecretIDIn(secretIDs...),
		).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list secret usage: %w", err)
	}
	result := make(map[uuid.UUID]secretUsageSummary, len(secretIDs))
	for _, item := range items {
		summary := result[item.SecretID]
		summary.count++
		scope := domain.BindingScopeKind(item.ScopeKind)
		if !slices.Contains(summary.scopes, scope) {
			summary.scopes = append(summary.scopes, scope)
			slices.SortFunc(summary.scopes, func(a, b domain.BindingScopeKind) int {
				return strings.Compare(string(a), string(b))
			})
		}
		result[item.SecretID] = summary
	}
	return result, nil
}

func collectSecretIDs(items []*ent.Secret) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}

func ensureDefaultBinding(ctx context.Context, tx *ent.Tx, item domain.Secret, bindingKey string) error {
	scopeKind, resourceID, projectID := defaultBindingIdentity(item)
	existing, err := tx.SecretBinding.Query().
		Where(
			entsecretbinding.SecretIDEQ(item.ID),
			entsecretbinding.ScopeKindEQ(scopeKind),
			entsecretbinding.ScopeResourceIDEQ(resourceID),
			entsecretbinding.ProjectIDEQ(projectID),
		).
		Only(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return fmt.Errorf("load default secret binding: %w", err)
	}
	if ent.IsNotFound(err) {
		_, err = tx.SecretBinding.Create().
			SetOrganizationID(item.OrganizationID).
			SetProjectID(projectID).
			SetSecretID(item.ID).
			SetScopeKind(scopeKind).
			SetScopeResourceID(resourceID).
			SetBindingKey(bindingKey).
			Save(ctx)
		if err != nil {
			return fmt.Errorf("create default secret binding: %w", err)
		}
		return nil
	}
	if _, err := tx.SecretBinding.UpdateOneID(existing.ID).SetBindingKey(bindingKey).Save(ctx); err != nil {
		return fmt.Errorf("update default secret binding: %w", err)
	}
	return nil
}

func defaultBindingIdentity(item domain.Secret) (entsecretbinding.ScopeKind, uuid.UUID, uuid.UUID) {
	if item.Scope == domain.ScopeKindOrganization {
		return entsecretbinding.ScopeKindOrganization, item.OrganizationID, uuid.Nil
	}
	return entsecretbinding.ScopeKindProject, item.ProjectID, item.ProjectID
}
