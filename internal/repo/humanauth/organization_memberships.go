package humanauth

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entorganizationinvitation "github.com/BetterAndBetterII/openase/ent/organizationinvitation"
	entorganizationmembership "github.com/BetterAndBetterII/openase/ent/organizationmembership"
	entuser "github.com/BetterAndBetterII/openase/ent/user"
	domain "github.com/BetterAndBetterII/openase/internal/domain/humanauth"
	"github.com/google/uuid"
)

var (
	ErrOrganizationMembershipNotFound = errors.New("organization membership not found")
	ErrOrganizationInvitationNotFound = errors.New("organization invitation not found")
)

type CreateOrganizationMembershipInput struct {
	OrganizationID uuid.UUID
	UserID         *uuid.UUID
	Email          string
	Role           domain.OrganizationMembershipRole
	Status         domain.OrganizationMembershipStatus
	InvitedBy      string
	InvitedAt      time.Time
	AcceptedAt     *time.Time
	SuspendedAt    *time.Time
	RemovedAt      *time.Time
}

type UpdateOrganizationMembershipInput struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	UserID         *uuid.UUID
	Email          string
	Role           domain.OrganizationMembershipRole
	Status         domain.OrganizationMembershipStatus
	InvitedBy      string
	InvitedAt      time.Time
	AcceptedAt     *time.Time
	SuspendedAt    *time.Time
	RemovedAt      *time.Time
}

type CreateOrganizationInvitationInput struct {
	OrganizationID   uuid.UUID
	MembershipID     uuid.UUID
	Email            string
	Role             domain.OrganizationMembershipRole
	Status           domain.OrganizationInvitationStatus
	InvitedBy        string
	InviteTokenHash  string
	ExpiresAt        time.Time
	SentAt           time.Time
	AcceptedByUserID *uuid.UUID
	AcceptedAt       *time.Time
	CanceledAt       *time.Time
}

type UpdateOrganizationInvitationInput struct {
	ID               uuid.UUID
	OrganizationID   uuid.UUID
	MembershipID     uuid.UUID
	Email            string
	Role             domain.OrganizationMembershipRole
	Status           domain.OrganizationInvitationStatus
	InvitedBy        string
	InviteTokenHash  string
	ExpiresAt        time.Time
	SentAt           time.Time
	AcceptedByUserID *uuid.UUID
	AcceptedAt       *time.Time
	CanceledAt       *time.Time
}

func (r *Repository) ListActiveOrganizationMembershipsByUser(
	ctx context.Context,
	userID uuid.UUID,
) ([]domain.OrganizationMembership, error) {
	items, err := r.client.OrganizationMembership.Query().
		Where(
			entorganizationmembership.UserID(userID),
			entorganizationmembership.StatusEQ(entorganizationmembership.StatusActive),
		).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list active organization memberships: %w", err)
	}
	result := make([]domain.OrganizationMembership, 0, len(items))
	for _, item := range items {
		result = append(result, mapOrganizationMembership(item))
	}
	return result, nil
}

func (r *Repository) ListOrganizationMembershipEntries(
	ctx context.Context,
	organizationID uuid.UUID,
) ([]domain.OrganizationMembershipEntry, error) {
	memberships, err := r.client.OrganizationMembership.Query().
		Where(entorganizationmembership.OrganizationIDEQ(organizationID)).
		Order(
			ent.Desc(entorganizationmembership.FieldInvitedAt),
			ent.Desc(entorganizationmembership.FieldCreatedAt),
		).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list organization memberships: %w", err)
	}

	pendingInvitations, err := r.client.OrganizationInvitation.Query().
		Where(
			entorganizationinvitation.OrganizationIDEQ(organizationID),
			entorganizationinvitation.StatusEQ(entorganizationinvitation.StatusPending),
		).
		Order(ent.Desc(entorganizationinvitation.FieldSentAt)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("list organization invitations: %w", err)
	}

	userIDs := make([]uuid.UUID, 0, len(memberships))
	seenUsers := map[uuid.UUID]struct{}{}
	for _, item := range memberships {
		if item.UserID == nil {
			continue
		}
		if _, ok := seenUsers[*item.UserID]; ok {
			continue
		}
		seenUsers[*item.UserID] = struct{}{}
		userIDs = append(userIDs, *item.UserID)
	}

	userSummaries := map[uuid.UUID]*domain.OrganizationMembershipUserSummary{}
	if len(userIDs) > 0 {
		users, err := r.client.User.Query().Where(entuser.IDIn(userIDs...)).All(ctx)
		if err != nil {
			return nil, fmt.Errorf("list organization membership users: %w", err)
		}
		for _, item := range users {
			userSummaries[item.ID] = &domain.OrganizationMembershipUserSummary{
				ID:           item.ID,
				PrimaryEmail: item.PrimaryEmail,
				DisplayName:  item.DisplayName,
				AvatarURL:    item.AvatarURL,
			}
		}
	}

	activeInvitations := map[uuid.UUID]*domain.OrganizationInvitation{}
	for _, item := range pendingInvitations {
		if _, ok := activeInvitations[item.MembershipID]; ok {
			continue
		}
		invitation := mapOrganizationInvitation(item)
		activeInvitations[item.MembershipID] = &invitation
	}

	result := make([]domain.OrganizationMembershipEntry, 0, len(memberships))
	for _, item := range memberships {
		entry := domain.OrganizationMembershipEntry{Membership: mapOrganizationMembership(item)}
		if item.UserID != nil {
			entry.User = userSummaries[*item.UserID]
		}
		entry.ActiveInvitation = activeInvitations[item.ID]
		result = append(result, entry)
	}
	return result, nil
}

func (r *Repository) GetOrganizationMembership(
	ctx context.Context,
	organizationID uuid.UUID,
	membershipID uuid.UUID,
) (domain.OrganizationMembership, error) {
	item, err := r.client.OrganizationMembership.Query().
		Where(
			entorganizationmembership.IDEQ(membershipID),
			entorganizationmembership.OrganizationIDEQ(organizationID),
		).
		Only(ctx)
	switch {
	case ent.IsNotFound(err):
		return domain.OrganizationMembership{}, ErrOrganizationMembershipNotFound
	case err != nil:
		return domain.OrganizationMembership{}, fmt.Errorf("get organization membership: %w", err)
	default:
		return mapOrganizationMembership(item), nil
	}
}

func (r *Repository) GetOrganizationMembershipByEmail(
	ctx context.Context,
	organizationID uuid.UUID,
	email string,
) (domain.OrganizationMembership, error) {
	item, err := r.client.OrganizationMembership.Query().
		Where(
			entorganizationmembership.OrganizationIDEQ(organizationID),
			entorganizationmembership.EmailEQ(strings.ToLower(strings.TrimSpace(email))),
		).
		Only(ctx)
	switch {
	case ent.IsNotFound(err):
		return domain.OrganizationMembership{}, ErrOrganizationMembershipNotFound
	case err != nil:
		return domain.OrganizationMembership{}, fmt.Errorf("get organization membership by email: %w", err)
	default:
		return mapOrganizationMembership(item), nil
	}
}

func (r *Repository) GetOrganizationMembershipByUser(
	ctx context.Context,
	organizationID uuid.UUID,
	userID uuid.UUID,
) (domain.OrganizationMembership, error) {
	item, err := r.client.OrganizationMembership.Query().
		Where(
			entorganizationmembership.OrganizationIDEQ(organizationID),
			entorganizationmembership.UserID(userID),
		).
		Only(ctx)
	switch {
	case ent.IsNotFound(err):
		return domain.OrganizationMembership{}, ErrOrganizationMembershipNotFound
	case err != nil:
		return domain.OrganizationMembership{}, fmt.Errorf("get organization membership by user: %w", err)
	default:
		return mapOrganizationMembership(item), nil
	}
}

func (r *Repository) CreateOrganizationMembership(
	ctx context.Context,
	input CreateOrganizationMembershipInput,
) (domain.OrganizationMembership, error) {
	builder := r.client.OrganizationMembership.Create().
		SetOrganizationID(input.OrganizationID).
		SetEmail(strings.ToLower(strings.TrimSpace(input.Email))).
		SetRole(entorganizationmembership.Role(input.Role)).
		SetStatus(entorganizationmembership.Status(input.Status)).
		SetInvitedBy(strings.TrimSpace(input.InvitedBy)).
		SetInvitedAt(input.InvitedAt.UTC())
	if input.UserID != nil {
		builder.SetUserID(*input.UserID)
	}
	if input.AcceptedAt != nil {
		builder.SetAcceptedAt(input.AcceptedAt.UTC())
	}
	if input.SuspendedAt != nil {
		builder.SetSuspendedAt(input.SuspendedAt.UTC())
	}
	if input.RemovedAt != nil {
		builder.SetRemovedAt(input.RemovedAt.UTC())
	}
	item, err := builder.Save(ctx)
	if err != nil {
		return domain.OrganizationMembership{}, fmt.Errorf("create organization membership: %w", err)
	}
	return mapOrganizationMembership(item), nil
}

func (r *Repository) UpdateOrganizationMembership(
	ctx context.Context,
	input UpdateOrganizationMembershipInput,
) (domain.OrganizationMembership, error) {
	builder := r.client.OrganizationMembership.UpdateOneID(input.ID).
		Where(entorganizationmembership.OrganizationIDEQ(input.OrganizationID)).
		SetEmail(strings.ToLower(strings.TrimSpace(input.Email))).
		SetRole(entorganizationmembership.Role(input.Role)).
		SetStatus(entorganizationmembership.Status(input.Status)).
		SetInvitedBy(strings.TrimSpace(input.InvitedBy)).
		SetInvitedAt(input.InvitedAt.UTC())
	if input.UserID != nil {
		builder.SetUserID(*input.UserID)
	} else {
		builder.ClearUserID()
	}
	if input.AcceptedAt != nil {
		builder.SetAcceptedAt(input.AcceptedAt.UTC())
	} else {
		builder.ClearAcceptedAt()
	}
	if input.SuspendedAt != nil {
		builder.SetSuspendedAt(input.SuspendedAt.UTC())
	} else {
		builder.ClearSuspendedAt()
	}
	if input.RemovedAt != nil {
		builder.SetRemovedAt(input.RemovedAt.UTC())
	} else {
		builder.ClearRemovedAt()
	}
	item, err := builder.Save(ctx)
	switch {
	case ent.IsNotFound(err):
		return domain.OrganizationMembership{}, ErrOrganizationMembershipNotFound
	case err != nil:
		return domain.OrganizationMembership{}, fmt.Errorf("update organization membership: %w", err)
	default:
		return mapOrganizationMembership(item), nil
	}
}

func (r *Repository) CountActiveOrganizationOwners(ctx context.Context, organizationID uuid.UUID) (int, error) {
	count, err := r.client.OrganizationMembership.Query().
		Where(
			entorganizationmembership.OrganizationIDEQ(organizationID),
			entorganizationmembership.StatusEQ(entorganizationmembership.StatusActive),
			entorganizationmembership.RoleEQ(entorganizationmembership.RoleOwner),
		).
		Count(ctx)
	if err != nil {
		return 0, fmt.Errorf("count active organization owners: %w", err)
	}
	return count, nil
}

func (r *Repository) GetOrganizationInvitation(
	ctx context.Context,
	organizationID uuid.UUID,
	invitationID uuid.UUID,
) (domain.OrganizationInvitation, error) {
	item, err := r.client.OrganizationInvitation.Query().
		Where(
			entorganizationinvitation.IDEQ(invitationID),
			entorganizationinvitation.OrganizationIDEQ(organizationID),
		).
		Only(ctx)
	switch {
	case ent.IsNotFound(err):
		return domain.OrganizationInvitation{}, ErrOrganizationInvitationNotFound
	case err != nil:
		return domain.OrganizationInvitation{}, fmt.Errorf("get organization invitation: %w", err)
	default:
		return mapOrganizationInvitation(item), nil
	}
}

func (r *Repository) GetPendingOrganizationInvitationByTokenHash(
	ctx context.Context,
	tokenHash string,
) (domain.OrganizationInvitation, error) {
	item, err := r.client.OrganizationInvitation.Query().
		Where(
			entorganizationinvitation.InviteTokenHashEQ(strings.TrimSpace(tokenHash)),
			entorganizationinvitation.StatusEQ(entorganizationinvitation.StatusPending),
		).
		Order(ent.Desc(entorganizationinvitation.FieldSentAt)).
		First(ctx)
	switch {
	case ent.IsNotFound(err):
		return domain.OrganizationInvitation{}, ErrOrganizationInvitationNotFound
	case err != nil:
		return domain.OrganizationInvitation{}, fmt.Errorf("get organization invitation by token: %w", err)
	default:
		return mapOrganizationInvitation(item), nil
	}
}

func (r *Repository) CreateOrganizationInvitation(
	ctx context.Context,
	input CreateOrganizationInvitationInput,
) (domain.OrganizationInvitation, error) {
	builder := r.client.OrganizationInvitation.Create().
		SetOrganizationID(input.OrganizationID).
		SetMembershipID(input.MembershipID).
		SetEmail(strings.ToLower(strings.TrimSpace(input.Email))).
		SetRole(entorganizationinvitation.Role(input.Role)).
		SetStatus(entorganizationinvitation.Status(input.Status)).
		SetInvitedBy(strings.TrimSpace(input.InvitedBy)).
		SetInviteTokenHash(strings.TrimSpace(input.InviteTokenHash)).
		SetExpiresAt(input.ExpiresAt.UTC()).
		SetSentAt(input.SentAt.UTC())
	if input.AcceptedByUserID != nil {
		builder.SetAcceptedByUserID(*input.AcceptedByUserID)
	}
	if input.AcceptedAt != nil {
		builder.SetAcceptedAt(input.AcceptedAt.UTC())
	}
	if input.CanceledAt != nil {
		builder.SetCanceledAt(input.CanceledAt.UTC())
	}
	item, err := builder.Save(ctx)
	if err != nil {
		return domain.OrganizationInvitation{}, fmt.Errorf("create organization invitation: %w", err)
	}
	return mapOrganizationInvitation(item), nil
}

func (r *Repository) UpdateOrganizationInvitation(
	ctx context.Context,
	input UpdateOrganizationInvitationInput,
) (domain.OrganizationInvitation, error) {
	builder := r.client.OrganizationInvitation.UpdateOneID(input.ID).
		Where(
			entorganizationinvitation.OrganizationIDEQ(input.OrganizationID),
			entorganizationinvitation.MembershipIDEQ(input.MembershipID),
		).
		SetEmail(strings.ToLower(strings.TrimSpace(input.Email))).
		SetRole(entorganizationinvitation.Role(input.Role)).
		SetStatus(entorganizationinvitation.Status(input.Status)).
		SetInvitedBy(strings.TrimSpace(input.InvitedBy)).
		SetInviteTokenHash(strings.TrimSpace(input.InviteTokenHash)).
		SetExpiresAt(input.ExpiresAt.UTC()).
		SetSentAt(input.SentAt.UTC())
	if input.AcceptedByUserID != nil {
		builder.SetAcceptedByUserID(*input.AcceptedByUserID)
	} else {
		builder.ClearAcceptedByUserID()
	}
	if input.AcceptedAt != nil {
		builder.SetAcceptedAt(input.AcceptedAt.UTC())
	} else {
		builder.ClearAcceptedAt()
	}
	if input.CanceledAt != nil {
		builder.SetCanceledAt(input.CanceledAt.UTC())
	} else {
		builder.ClearCanceledAt()
	}
	item, err := builder.Save(ctx)
	switch {
	case ent.IsNotFound(err):
		return domain.OrganizationInvitation{}, ErrOrganizationInvitationNotFound
	case err != nil:
		return domain.OrganizationInvitation{}, fmt.Errorf("update organization invitation: %w", err)
	default:
		return mapOrganizationInvitation(item), nil
	}
}

func mapOrganizationMembership(item *ent.OrganizationMembership) domain.OrganizationMembership {
	return domain.OrganizationMembership{
		ID:             item.ID,
		OrganizationID: item.OrganizationID,
		UserID:         item.UserID,
		Email:          item.Email,
		Role:           domain.OrganizationMembershipRole(item.Role),
		Status:         domain.OrganizationMembershipStatus(item.Status),
		InvitedBy:      item.InvitedBy,
		InvitedAt:      item.InvitedAt,
		AcceptedAt:     item.AcceptedAt,
		SuspendedAt:    item.SuspendedAt,
		RemovedAt:      item.RemovedAt,
		CreatedAt:      item.CreatedAt,
		UpdatedAt:      item.UpdatedAt,
	}
}

func mapOrganizationInvitation(item *ent.OrganizationInvitation) domain.OrganizationInvitation {
	return domain.OrganizationInvitation{
		ID:               item.ID,
		OrganizationID:   item.OrganizationID,
		MembershipID:     item.MembershipID,
		Email:            item.Email,
		Role:             domain.OrganizationMembershipRole(item.Role),
		Status:           domain.OrganizationInvitationStatus(item.Status),
		InvitedBy:        item.InvitedBy,
		InviteTokenHash:  item.InviteTokenHash,
		ExpiresAt:        item.ExpiresAt,
		SentAt:           item.SentAt,
		AcceptedByUserID: item.AcceptedByUserID,
		AcceptedAt:       item.AcceptedAt,
		CanceledAt:       item.CanceledAt,
		CreatedAt:        item.CreatedAt,
		UpdatedAt:        item.UpdatedAt,
	}
}

func sortOrganizationMembershipEntries(entries []domain.OrganizationMembershipEntry) {
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Membership.Email < entries[j].Membership.Email
	})
}
