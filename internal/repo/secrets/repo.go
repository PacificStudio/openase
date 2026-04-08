package secrets

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	"github.com/BetterAndBetterII/openase/ent/predicate"
	entproject "github.com/BetterAndBetterII/openase/ent/project"
	entsecret "github.com/BetterAndBetterII/openase/ent/secret"
	entsecretbinding "github.com/BetterAndBetterII/openase/ent/secretbinding"
	domain "github.com/BetterAndBetterII/openase/internal/domain/secrets"
	"github.com/google/uuid"
)

var (
	ErrSecretNotFound     = errors.New("secret not found")
	ErrSecretNameConflict = errors.New("secret name conflict")
)

type ProjectContext struct {
	ProjectID      uuid.UUID
	OrganizationID uuid.UUID
}

type Repository interface {
	GetProjectContext(ctx context.Context, projectID uuid.UUID) (ProjectContext, error)
	ListAccessibleSecrets(ctx context.Context, projectID uuid.UUID) ([]domain.Secret, error)
	CreateSecret(ctx context.Context, item domain.Secret) (domain.Secret, error)
	GetSecret(ctx context.Context, projectID uuid.UUID, secretID uuid.UUID) (domain.Secret, error)
	UpdateSecretMetadata(ctx context.Context, projectID uuid.UUID, secretID uuid.UUID, name string, description string) (domain.Secret, error)
	RotateSecret(ctx context.Context, projectID uuid.UUID, secretID uuid.UUID, value domain.StoredValue) (domain.Secret, error)
	DisableSecret(ctx context.Context, projectID uuid.UUID, secretID uuid.UUID, disabledAt time.Time) (domain.Secret, error)
	ListResolutionCandidates(ctx context.Context, projectID uuid.UUID, keys []string, ticketID, workflowID, agentID *uuid.UUID) ([]domain.Candidate, error)
	CreateBinding(ctx context.Context, item domain.Binding) (domain.Binding, error)
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

func (r *EntRepository) ListAccessibleSecrets(ctx context.Context, projectID uuid.UUID) ([]domain.Secret, error) {
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
	result := make([]domain.Secret, 0, len(items))
	for _, item := range items {
		result = append(result, mapSecret(item))
	}
	return result, nil
}

func (r *EntRepository) CreateSecret(ctx context.Context, item domain.Secret) (domain.Secret, error) {
	created, err := r.client.Secret.Create().
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
	if err != nil {
		if ent.IsConstraintError(err) {
			return domain.Secret{}, ErrSecretNameConflict
		}
		return domain.Secret{}, fmt.Errorf("create secret: %w", err)
	}
	return mapSecret(created), nil
}

func (r *EntRepository) GetSecret(ctx context.Context, projectID uuid.UUID, secretID uuid.UUID) (domain.Secret, error) {
	projectContext, err := r.GetProjectContext(ctx, projectID)
	if err != nil {
		return domain.Secret{}, err
	}
	item, err := r.client.Secret.Query().
		Where(
			entsecret.IDEQ(secretID),
			entsecret.OrganizationIDEQ(projectContext.OrganizationID),
			entsecret.Or(
				entsecret.ProjectIDEQ(uuid.Nil),
				entsecret.ProjectIDEQ(projectID),
			),
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

func (r *EntRepository) UpdateSecretMetadata(ctx context.Context, projectID uuid.UUID, secretID uuid.UUID, name string, description string) (domain.Secret, error) {
	item, err := r.GetSecret(ctx, projectID, secretID)
	if err != nil {
		return domain.Secret{}, err
	}
	updated, err := r.client.Secret.UpdateOneID(item.ID).
		SetName(name).
		SetDescription(description).
		Save(ctx)
	if err != nil {
		if ent.IsConstraintError(err) {
			return domain.Secret{}, ErrSecretNameConflict
		}
		return domain.Secret{}, fmt.Errorf("update secret metadata: %w", err)
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
		return domain.Binding{}, fmt.Errorf("create secret binding: %w", err)
	}
	return mapBinding(created), nil
}

type bindingSelector struct {
	scope      entsecretbinding.ScopeKind
	resourceID uuid.UUID
	projectID  uuid.UUID
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
