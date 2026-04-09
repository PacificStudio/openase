package humanauth

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type OrganizationMembershipRole string

const (
	OrganizationMembershipRoleOwner  OrganizationMembershipRole = "owner"
	OrganizationMembershipRoleAdmin  OrganizationMembershipRole = "admin"
	OrganizationMembershipRoleMember OrganizationMembershipRole = "member"
)

func ParseOrganizationMembershipRole(raw string) (OrganizationMembershipRole, error) {
	switch OrganizationMembershipRole(strings.ToLower(strings.TrimSpace(raw))) {
	case OrganizationMembershipRoleOwner:
		return OrganizationMembershipRoleOwner, nil
	case OrganizationMembershipRoleAdmin:
		return OrganizationMembershipRoleAdmin, nil
	case OrganizationMembershipRoleMember:
		return OrganizationMembershipRoleMember, nil
	default:
		return "", fmt.Errorf("unsupported organization membership role %q", raw)
	}
}

func (r OrganizationMembershipRole) String() string { return string(r) }

func (r OrganizationMembershipRole) RoleKey() RoleKey {
	switch r {
	case OrganizationMembershipRoleOwner:
		return RoleOrgOwner
	case OrganizationMembershipRoleAdmin:
		return RoleOrgAdmin
	default:
		return RoleOrgMember
	}
}

type OrganizationMembershipStatus string

const (
	OrganizationMembershipStatusInvited   OrganizationMembershipStatus = "invited"
	OrganizationMembershipStatusActive    OrganizationMembershipStatus = "active"
	OrganizationMembershipStatusSuspended OrganizationMembershipStatus = "suspended"
	OrganizationMembershipStatusRemoved   OrganizationMembershipStatus = "removed"
)

func ParseOrganizationMembershipStatus(raw string) (OrganizationMembershipStatus, error) {
	switch OrganizationMembershipStatus(strings.ToLower(strings.TrimSpace(raw))) {
	case OrganizationMembershipStatusInvited:
		return OrganizationMembershipStatusInvited, nil
	case OrganizationMembershipStatusActive:
		return OrganizationMembershipStatusActive, nil
	case OrganizationMembershipStatusSuspended:
		return OrganizationMembershipStatusSuspended, nil
	case OrganizationMembershipStatusRemoved:
		return OrganizationMembershipStatusRemoved, nil
	default:
		return "", fmt.Errorf("unsupported organization membership status %q", raw)
	}
}

func (s OrganizationMembershipStatus) String() string { return string(s) }

func (s OrganizationMembershipStatus) CanTransitionTo(next OrganizationMembershipStatus) bool {
	switch s {
	case OrganizationMembershipStatusInvited:
		return next == OrganizationMembershipStatusActive || next == OrganizationMembershipStatusRemoved
	case OrganizationMembershipStatusActive:
		return next == OrganizationMembershipStatusSuspended || next == OrganizationMembershipStatusRemoved
	case OrganizationMembershipStatusSuspended:
		return next == OrganizationMembershipStatusActive || next == OrganizationMembershipStatusRemoved
	case OrganizationMembershipStatusRemoved:
		return false
	default:
		return false
	}
}

type OrganizationInvitationStatus string

const (
	OrganizationInvitationStatusPending  OrganizationInvitationStatus = "pending"
	OrganizationInvitationStatusAccepted OrganizationInvitationStatus = "accepted"
	OrganizationInvitationStatusCanceled OrganizationInvitationStatus = "canceled"
	OrganizationInvitationStatusExpired  OrganizationInvitationStatus = "expired"
)

func ParseOrganizationInvitationStatus(raw string) (OrganizationInvitationStatus, error) {
	switch OrganizationInvitationStatus(strings.ToLower(strings.TrimSpace(raw))) {
	case OrganizationInvitationStatusPending:
		return OrganizationInvitationStatusPending, nil
	case OrganizationInvitationStatusAccepted:
		return OrganizationInvitationStatusAccepted, nil
	case OrganizationInvitationStatusCanceled:
		return OrganizationInvitationStatusCanceled, nil
	case OrganizationInvitationStatusExpired:
		return OrganizationInvitationStatusExpired, nil
	default:
		return "", fmt.Errorf("unsupported organization invitation status %q", raw)
	}
}

func (s OrganizationInvitationStatus) String() string { return string(s) }

type OrganizationMembership struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	UserID         *uuid.UUID
	Email          string
	Role           OrganizationMembershipRole
	Status         OrganizationMembershipStatus
	InvitedBy      string
	InvitedAt      time.Time
	AcceptedAt     *time.Time
	SuspendedAt    *time.Time
	RemovedAt      *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type OrganizationInvitation struct {
	ID               uuid.UUID
	OrganizationID   uuid.UUID
	MembershipID     uuid.UUID
	Email            string
	Role             OrganizationMembershipRole
	Status           OrganizationInvitationStatus
	InvitedBy        string
	InviteTokenHash  string
	ExpiresAt        time.Time
	SentAt           time.Time
	AcceptedByUserID *uuid.UUID
	AcceptedAt       *time.Time
	CanceledAt       *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type OrganizationMembershipUserSummary struct {
	ID           uuid.UUID
	PrimaryEmail string
	DisplayName  string
	AvatarURL    string
}

type OrganizationMembershipEntry struct {
	Membership       OrganizationMembership
	User             *OrganizationMembershipUserSummary
	ActiveInvitation *OrganizationInvitation
}
