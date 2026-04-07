package humanauth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	domain "github.com/BetterAndBetterII/openase/internal/domain/humanauth"
	repo "github.com/BetterAndBetterII/openase/internal/repo/humanauth"
	"github.com/google/uuid"
)

var (
	ErrOrganizationMembershipNotFound = errors.New("organization membership not found")
	ErrOrganizationInvitationNotFound = errors.New("organization invitation not found")
	ErrOrganizationInvitationExpired  = errors.New("organization invitation expired")
	ErrOrganizationInvitationPending  = errors.New("organization invitation is already pending")
	ErrOrganizationMemberExists       = errors.New("organization member already exists")
	ErrLastOrganizationOwner          = errors.New("cannot change the last active organization owner")
	ErrOrganizationInvitationMismatch = errors.New("organization invitation email does not match the current user")
	ErrOrganizationAcceptanceRequired = errors.New("organization membership cannot become active before invitation acceptance")
)

const organizationInvitationTTL = 7 * 24 * time.Hour

type InviteOrganizationMemberInput struct {
	Email string
	Role  string
}

type InviteOrganizationMemberResult struct {
	Entry       domain.OrganizationMembershipEntry
	Invitation  domain.OrganizationInvitation
	AcceptToken string
}

type UpdateOrganizationMembershipInput struct {
	Role   *string
	Status *string
}

type TransferOrganizationOwnershipInput struct {
	PreviousOwnerRole string
}

func (s *Service) ListOrganizationMembershipEntries(
	ctx context.Context,
	organizationID uuid.UUID,
) ([]domain.OrganizationMembershipEntry, error) {
	entries, err := s.repo.ListOrganizationMembershipEntries(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	return entries, nil
}

func (s *Service) InviteOrganizationMember(
	ctx context.Context,
	organizationID uuid.UUID,
	actor domain.AuthenticatedPrincipal,
	input InviteOrganizationMemberInput,
) (InviteOrganizationMemberResult, error) {
	email := strings.ToLower(strings.TrimSpace(input.Email))
	if email == "" {
		return InviteOrganizationMemberResult{}, fmt.Errorf("email is required")
	}
	role, err := domain.ParseOrganizationMembershipRole(input.Role)
	if err != nil {
		return InviteOrganizationMemberResult{}, err
	}

	membership, err := s.repo.GetOrganizationMembershipByEmail(ctx, organizationID, email)
	switch {
	case err == nil:
		if membership.Status == domain.OrganizationMembershipStatusActive || membership.Status == domain.OrganizationMembershipStatusSuspended {
			return InviteOrganizationMemberResult{}, ErrOrganizationMemberExists
		}
	case errors.Is(err, repo.ErrOrganizationMembershipNotFound):
		membership = domain.OrganizationMembership{}
	case err != nil:
		return InviteOrganizationMemberResult{}, err
	}

	now := time.Now().UTC()
	if membership.ID == uuid.Nil {
		created, createErr := s.repo.CreateOrganizationMembership(ctx, repo.CreateOrganizationMembershipInput{
			OrganizationID: organizationID,
			Email:          email,
			Role:           role,
			Status:         domain.OrganizationMembershipStatusInvited,
			InvitedBy:      actor.ActorID(),
			InvitedAt:      now,
		})
		if createErr != nil {
			return InviteOrganizationMemberResult{}, createErr
		}
		membership = created
	} else {
		entries, listErr := s.repo.ListOrganizationMembershipEntries(ctx, organizationID)
		if listErr != nil {
			return InviteOrganizationMemberResult{}, listErr
		}
		for _, entry := range entries {
			if entry.Membership.ID == membership.ID && entry.ActiveInvitation != nil {
				return InviteOrganizationMemberResult{}, ErrOrganizationInvitationPending
			}
		}
		updated, updateErr := s.repo.UpdateOrganizationMembership(ctx, repo.UpdateOrganizationMembershipInput{
			ID:             membership.ID,
			OrganizationID: membership.OrganizationID,
			Email:          email,
			Role:           role,
			Status:         domain.OrganizationMembershipStatusInvited,
			InvitedBy:      actor.ActorID(),
			InvitedAt:      now,
		})
		if updateErr != nil {
			return InviteOrganizationMemberResult{}, updateErr
		}
		membership = updated
	}

	acceptToken, err := randomToken(24)
	if err != nil {
		return InviteOrganizationMemberResult{}, fmt.Errorf("generate invitation token: %w", err)
	}
	invitation, err := s.repo.CreateOrganizationInvitation(ctx, repo.CreateOrganizationInvitationInput{
		OrganizationID:  organizationID,
		MembershipID:    membership.ID,
		Email:           email,
		Role:            role,
		Status:          domain.OrganizationInvitationStatusPending,
		InvitedBy:       actor.ActorID(),
		InviteTokenHash: hashToken(acceptToken),
		ExpiresAt:       now.Add(organizationInvitationTTL),
		SentAt:          now,
	})
	if err != nil {
		return InviteOrganizationMemberResult{}, err
	}

	entries, err := s.repo.ListOrganizationMembershipEntries(ctx, organizationID)
	if err != nil {
		return InviteOrganizationMemberResult{}, err
	}
	entry, err := findOrganizationMembershipEntry(entries, membership.ID)
	if err != nil {
		return InviteOrganizationMemberResult{}, err
	}
	return InviteOrganizationMemberResult{Entry: entry, Invitation: invitation, AcceptToken: acceptToken}, nil
}

func (s *Service) ResendOrganizationInvitation(
	ctx context.Context,
	organizationID uuid.UUID,
	invitationID uuid.UUID,
	actor domain.AuthenticatedPrincipal,
) (InviteOrganizationMemberResult, error) {
	invitation, membership, now, err := s.loadOrganizationInvitationState(ctx, organizationID, invitationID)
	if err != nil {
		return InviteOrganizationMemberResult{}, err
	}

	if invitation.Status == domain.OrganizationInvitationStatusAccepted || invitation.Status == domain.OrganizationInvitationStatusCanceled {
		return InviteOrganizationMemberResult{}, fmt.Errorf("invitation cannot be resent from status %s", invitation.Status)
	}
	if invitation.Status == domain.OrganizationInvitationStatusPending && now.After(invitation.ExpiresAt) {
		invitation, membership, err = s.expireOrganizationInvitation(ctx, invitation, membership, now)
		if err != nil {
			return InviteOrganizationMemberResult{}, err
		}
	}
	if invitation.Status != domain.OrganizationInvitationStatusPending && invitation.Status != domain.OrganizationInvitationStatusExpired {
		return InviteOrganizationMemberResult{}, fmt.Errorf("invitation cannot be resent from status %s", invitation.Status)
	}

	membership, err = s.repo.UpdateOrganizationMembership(ctx, repo.UpdateOrganizationMembershipInput{
		ID:             membership.ID,
		OrganizationID: membership.OrganizationID,
		Email:          membership.Email,
		Role:           membership.Role,
		Status:         domain.OrganizationMembershipStatusInvited,
		InvitedBy:      actor.ActorID(),
		InvitedAt:      now,
	})
	if err != nil {
		return InviteOrganizationMemberResult{}, err
	}

	acceptToken, err := randomToken(24)
	if err != nil {
		return InviteOrganizationMemberResult{}, fmt.Errorf("generate invitation token: %w", err)
	}
	invitation, err = s.repo.UpdateOrganizationInvitation(ctx, repo.UpdateOrganizationInvitationInput{
		ID:              invitation.ID,
		OrganizationID:  invitation.OrganizationID,
		MembershipID:    invitation.MembershipID,
		Email:           invitation.Email,
		Role:            invitation.Role,
		Status:          domain.OrganizationInvitationStatusPending,
		InvitedBy:       actor.ActorID(),
		InviteTokenHash: hashToken(acceptToken),
		ExpiresAt:       now.Add(organizationInvitationTTL),
		SentAt:          now,
	})
	if err != nil {
		return InviteOrganizationMemberResult{}, err
	}

	entries, err := s.repo.ListOrganizationMembershipEntries(ctx, organizationID)
	if err != nil {
		return InviteOrganizationMemberResult{}, err
	}
	entry, err := findOrganizationMembershipEntry(entries, membership.ID)
	if err != nil {
		return InviteOrganizationMemberResult{}, err
	}
	return InviteOrganizationMemberResult{Entry: entry, Invitation: invitation, AcceptToken: acceptToken}, nil
}

func (s *Service) CancelOrganizationInvitation(
	ctx context.Context,
	organizationID uuid.UUID,
	invitationID uuid.UUID,
) (domain.OrganizationMembershipEntry, error) {
	invitation, membership, now, err := s.loadOrganizationInvitationState(ctx, organizationID, invitationID)
	if err != nil {
		return domain.OrganizationMembershipEntry{}, err
	}
	if invitation.Status == domain.OrganizationInvitationStatusPending && now.After(invitation.ExpiresAt) {
		_, membership, err = s.expireOrganizationInvitation(ctx, invitation, membership, now)
		if err != nil {
			return domain.OrganizationMembershipEntry{}, err
		}
	} else {
		if invitation.Status != domain.OrganizationInvitationStatusPending {
			return domain.OrganizationMembershipEntry{}, fmt.Errorf("invitation cannot be canceled from status %s", invitation.Status)
		}
		membership, err = s.repo.UpdateOrganizationMembership(ctx, repo.UpdateOrganizationMembershipInput{
			ID:             membership.ID,
			OrganizationID: membership.OrganizationID,
			UserID:         membership.UserID,
			Email:          membership.Email,
			Role:           membership.Role,
			Status:         domain.OrganizationMembershipStatusRemoved,
			InvitedBy:      membership.InvitedBy,
			InvitedAt:      membership.InvitedAt,
			AcceptedAt:     membership.AcceptedAt,
			RemovedAt:      &now,
		})
		if err != nil {
			return domain.OrganizationMembershipEntry{}, err
		}
		if _, err := s.repo.UpdateOrganizationInvitation(ctx, repo.UpdateOrganizationInvitationInput{
			ID:              invitation.ID,
			OrganizationID:  invitation.OrganizationID,
			MembershipID:    invitation.MembershipID,
			Email:           invitation.Email,
			Role:            invitation.Role,
			Status:          domain.OrganizationInvitationStatusCanceled,
			InvitedBy:       invitation.InvitedBy,
			InviteTokenHash: invitation.InviteTokenHash,
			ExpiresAt:       invitation.ExpiresAt,
			SentAt:          invitation.SentAt,
			CanceledAt:      &now,
		}); err != nil {
			return domain.OrganizationMembershipEntry{}, err
		}
	}
	entries, err := s.repo.ListOrganizationMembershipEntries(ctx, organizationID)
	if err != nil {
		return domain.OrganizationMembershipEntry{}, err
	}
	return findOrganizationMembershipEntry(entries, membership.ID)
}

func (s *Service) AcceptOrganizationInvitation(
	ctx context.Context,
	principal domain.AuthenticatedPrincipal,
	acceptToken string,
) (domain.OrganizationMembershipEntry, error) {
	invitation, err := s.repo.GetPendingOrganizationInvitationByTokenHash(ctx, hashToken(strings.TrimSpace(acceptToken)))
	if errors.Is(err, repo.ErrOrganizationInvitationNotFound) {
		return domain.OrganizationMembershipEntry{}, ErrOrganizationInvitationNotFound
	}
	if err != nil {
		return domain.OrganizationMembershipEntry{}, err
	}
	membership, err := s.repo.GetOrganizationMembership(ctx, invitation.OrganizationID, invitation.MembershipID)
	if errors.Is(err, repo.ErrOrganizationMembershipNotFound) {
		return domain.OrganizationMembershipEntry{}, ErrOrganizationMembershipNotFound
	}
	if err != nil {
		return domain.OrganizationMembershipEntry{}, err
	}

	now := time.Now().UTC()
	if now.After(invitation.ExpiresAt) {
		if _, _, expireErr := s.expireOrganizationInvitation(ctx, invitation, membership, now); expireErr != nil {
			return domain.OrganizationMembershipEntry{}, expireErr
		}
		return domain.OrganizationMembershipEntry{}, ErrOrganizationInvitationExpired
	}
	if !organizationInvitationMatchesPrincipal(invitation, principal) {
		return domain.OrganizationMembershipEntry{}, ErrOrganizationInvitationMismatch
	}

	membership, err = s.repo.UpdateOrganizationMembership(ctx, repo.UpdateOrganizationMembershipInput{
		ID:             membership.ID,
		OrganizationID: membership.OrganizationID,
		UserID:         &principal.User.ID,
		Email:          strings.ToLower(strings.TrimSpace(principal.User.PrimaryEmail)),
		Role:           membership.Role,
		Status:         domain.OrganizationMembershipStatusActive,
		InvitedBy:      membership.InvitedBy,
		InvitedAt:      membership.InvitedAt,
		AcceptedAt:     &now,
	})
	if err != nil {
		return domain.OrganizationMembershipEntry{}, err
	}
	if _, err := s.repo.UpdateOrganizationInvitation(ctx, repo.UpdateOrganizationInvitationInput{
		ID:               invitation.ID,
		OrganizationID:   invitation.OrganizationID,
		MembershipID:     invitation.MembershipID,
		Email:            strings.ToLower(strings.TrimSpace(principal.User.PrimaryEmail)),
		Role:             invitation.Role,
		Status:           domain.OrganizationInvitationStatusAccepted,
		InvitedBy:        invitation.InvitedBy,
		InviteTokenHash:  invitation.InviteTokenHash,
		ExpiresAt:        invitation.ExpiresAt,
		SentAt:           invitation.SentAt,
		AcceptedByUserID: &principal.User.ID,
		AcceptedAt:       &now,
	}); err != nil {
		return domain.OrganizationMembershipEntry{}, err
	}
	entries, err := s.repo.ListOrganizationMembershipEntries(ctx, invitation.OrganizationID)
	if err != nil {
		return domain.OrganizationMembershipEntry{}, err
	}
	return findOrganizationMembershipEntry(entries, membership.ID)
}

func (s *Service) UpdateOrganizationMembership(
	ctx context.Context,
	organizationID uuid.UUID,
	membershipID uuid.UUID,
	input UpdateOrganizationMembershipInput,
) (domain.OrganizationMembershipEntry, error) {
	membership, err := s.repo.GetOrganizationMembership(ctx, organizationID, membershipID)
	if errors.Is(err, repo.ErrOrganizationMembershipNotFound) {
		return domain.OrganizationMembershipEntry{}, ErrOrganizationMembershipNotFound
	}
	if err != nil {
		return domain.OrganizationMembershipEntry{}, err
	}
	updated, err := s.applyOrganizationMembershipUpdate(ctx, membership, input)
	if err != nil {
		return domain.OrganizationMembershipEntry{}, err
	}
	entries, err := s.repo.ListOrganizationMembershipEntries(ctx, organizationID)
	if err != nil {
		return domain.OrganizationMembershipEntry{}, err
	}
	return findOrganizationMembershipEntry(entries, updated.ID)
}

func (s *Service) TransferOrganizationOwnership(
	ctx context.Context,
	organizationID uuid.UUID,
	targetMembershipID uuid.UUID,
	actor domain.AuthenticatedPrincipal,
	input TransferOrganizationOwnershipInput,
) ([]domain.OrganizationMembershipEntry, error) {
	target, err := s.repo.GetOrganizationMembership(ctx, organizationID, targetMembershipID)
	if errors.Is(err, repo.ErrOrganizationMembershipNotFound) {
		return nil, ErrOrganizationMembershipNotFound
	}
	if err != nil {
		return nil, err
	}
	if target.Status != domain.OrganizationMembershipStatusActive {
		return nil, fmt.Errorf("ownership can only be transferred to an active member")
	}
	if target.UserID == nil {
		return nil, ErrOrganizationAcceptanceRequired
	}

	actorMembership, actorMembershipErr := s.repo.GetOrganizationMembershipByUser(ctx, organizationID, actor.User.ID)
	actorIsOwner := actorMembershipErr == nil && actorMembership.Status == domain.OrganizationMembershipStatusActive && actorMembership.Role == domain.OrganizationMembershipRoleOwner
	if !actorIsOwner && !principalHasRole(actor, domain.RoleInstanceAdmin) {
		return nil, ErrPermissionDenied
	}

	promoted, err := s.applyOrganizationMembershipUpdate(ctx, target, UpdateOrganizationMembershipInput{Role: stringPointer(domain.OrganizationMembershipRoleOwner.String())})
	if err != nil {
		return nil, err
	}
	_ = promoted

	if actorIsOwner && actorMembership.ID != targetMembershipID {
		nextRole := strings.TrimSpace(input.PreviousOwnerRole)
		if nextRole == "" {
			nextRole = domain.OrganizationMembershipRoleAdmin.String()
		}
		if _, err := s.applyOrganizationMembershipUpdate(ctx, actorMembership, UpdateOrganizationMembershipInput{Role: &nextRole}); err != nil {
			return nil, err
		}
	}

	entries, err := s.repo.ListOrganizationMembershipEntries(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	return entries, nil
}

func (s *Service) applyOrganizationMembershipUpdate(
	ctx context.Context,
	membership domain.OrganizationMembership,
	input UpdateOrganizationMembershipInput,
) (domain.OrganizationMembership, error) {
	nextRole := membership.Role
	if input.Role != nil {
		parsedRole, err := domain.ParseOrganizationMembershipRole(*input.Role)
		if err != nil {
			return domain.OrganizationMembership{}, err
		}
		nextRole = parsedRole
	}

	nextStatus := membership.Status
	if input.Status != nil {
		parsedStatus, err := domain.ParseOrganizationMembershipStatus(*input.Status)
		if err != nil {
			return domain.OrganizationMembership{}, err
		}
		nextStatus = parsedStatus
	}

	if input.Status != nil && nextStatus != membership.Status && !membership.Status.CanTransitionTo(nextStatus) {
		return domain.OrganizationMembership{}, fmt.Errorf("membership cannot transition from %s to %s", membership.Status, nextStatus)
	}
	if input.Status != nil &&
		membership.Status != domain.OrganizationMembershipStatusActive &&
		nextStatus == domain.OrganizationMembershipStatusActive &&
		membership.UserID == nil {
		return domain.OrganizationMembership{}, ErrOrganizationAcceptanceRequired
	}

	if membership.Role == domain.OrganizationMembershipRoleOwner && membership.Status == domain.OrganizationMembershipStatusActive {
		if nextRole != domain.OrganizationMembershipRoleOwner || nextStatus != domain.OrganizationMembershipStatusActive {
			count, err := s.repo.CountActiveOrganizationOwners(ctx, membership.OrganizationID)
			if err != nil {
				return domain.OrganizationMembership{}, err
			}
			if count <= 1 {
				return domain.OrganizationMembership{}, ErrLastOrganizationOwner
			}
		}
	}

	now := time.Now().UTC()
	acceptedAt := membership.AcceptedAt
	suspendedAt := membership.SuspendedAt
	removedAt := membership.RemovedAt
	switch nextStatus {
	case domain.OrganizationMembershipStatusActive:
		if acceptedAt == nil {
			acceptedAt = &now
		}
		suspendedAt = nil
		removedAt = nil
	case domain.OrganizationMembershipStatusSuspended:
		suspendedAt = &now
		removedAt = nil
	case domain.OrganizationMembershipStatusRemoved:
		removedAt = &now
		suspendedAt = nil
	}

	updated, err := s.repo.UpdateOrganizationMembership(ctx, repo.UpdateOrganizationMembershipInput{
		ID:             membership.ID,
		OrganizationID: membership.OrganizationID,
		UserID:         membership.UserID,
		Email:          membership.Email,
		Role:           nextRole,
		Status:         nextStatus,
		InvitedBy:      membership.InvitedBy,
		InvitedAt:      membership.InvitedAt,
		AcceptedAt:     acceptedAt,
		SuspendedAt:    suspendedAt,
		RemovedAt:      removedAt,
	})
	if errors.Is(err, repo.ErrOrganizationMembershipNotFound) {
		return domain.OrganizationMembership{}, ErrOrganizationMembershipNotFound
	}
	return updated, err
}

func (s *Service) loadOrganizationInvitationState(
	ctx context.Context,
	organizationID uuid.UUID,
	invitationID uuid.UUID,
) (domain.OrganizationInvitation, domain.OrganizationMembership, time.Time, error) {
	invitation, err := s.repo.GetOrganizationInvitation(ctx, organizationID, invitationID)
	if errors.Is(err, repo.ErrOrganizationInvitationNotFound) {
		return domain.OrganizationInvitation{}, domain.OrganizationMembership{}, time.Time{}, ErrOrganizationInvitationNotFound
	}
	if err != nil {
		return domain.OrganizationInvitation{}, domain.OrganizationMembership{}, time.Time{}, err
	}
	membership, err := s.repo.GetOrganizationMembership(ctx, organizationID, invitation.MembershipID)
	if errors.Is(err, repo.ErrOrganizationMembershipNotFound) {
		return domain.OrganizationInvitation{}, domain.OrganizationMembership{}, time.Time{}, ErrOrganizationMembershipNotFound
	}
	if err != nil {
		return domain.OrganizationInvitation{}, domain.OrganizationMembership{}, time.Time{}, err
	}
	return invitation, membership, time.Now().UTC(), nil
}

func (s *Service) expireOrganizationInvitation(
	ctx context.Context,
	invitation domain.OrganizationInvitation,
	membership domain.OrganizationMembership,
	now time.Time,
) (domain.OrganizationInvitation, domain.OrganizationMembership, error) {
	membership, err := s.repo.UpdateOrganizationMembership(ctx, repo.UpdateOrganizationMembershipInput{
		ID:             membership.ID,
		OrganizationID: membership.OrganizationID,
		UserID:         membership.UserID,
		Email:          membership.Email,
		Role:           membership.Role,
		Status:         domain.OrganizationMembershipStatusRemoved,
		InvitedBy:      membership.InvitedBy,
		InvitedAt:      membership.InvitedAt,
		AcceptedAt:     membership.AcceptedAt,
		RemovedAt:      &now,
	})
	if err != nil {
		return domain.OrganizationInvitation{}, domain.OrganizationMembership{}, err
	}
	invitation, err = s.repo.UpdateOrganizationInvitation(ctx, repo.UpdateOrganizationInvitationInput{
		ID:              invitation.ID,
		OrganizationID:  invitation.OrganizationID,
		MembershipID:    invitation.MembershipID,
		Email:           invitation.Email,
		Role:            invitation.Role,
		Status:          domain.OrganizationInvitationStatusExpired,
		InvitedBy:       invitation.InvitedBy,
		InviteTokenHash: invitation.InviteTokenHash,
		ExpiresAt:       invitation.ExpiresAt,
		SentAt:          invitation.SentAt,
		CanceledAt:      nil,
	})
	if err != nil {
		return domain.OrganizationInvitation{}, domain.OrganizationMembership{}, err
	}
	return invitation, membership, nil
}

func findOrganizationMembershipEntry(
	entries []domain.OrganizationMembershipEntry,
	membershipID uuid.UUID,
) (domain.OrganizationMembershipEntry, error) {
	for _, entry := range entries {
		if entry.Membership.ID == membershipID {
			return entry, nil
		}
	}
	return domain.OrganizationMembershipEntry{}, ErrOrganizationMembershipNotFound
}

func organizationInvitationMatchesPrincipal(
	invitation domain.OrganizationInvitation,
	principal domain.AuthenticatedPrincipal,
) bool {
	email := strings.ToLower(strings.TrimSpace(invitation.Email))
	if email == "" {
		return false
	}
	candidates := []string{
		strings.ToLower(strings.TrimSpace(principal.User.PrimaryEmail)),
		strings.ToLower(strings.TrimSpace(principal.Identity.Email)),
	}
	for _, candidate := range candidates {
		if candidate != "" && candidate == email {
			return true
		}
	}
	return false
}

func principalHasRole(principal domain.AuthenticatedPrincipal, role domain.RoleKey) bool {
	for _, candidate := range principal.EffectiveRoles {
		if candidate == role {
			return true
		}
	}
	return false
}
