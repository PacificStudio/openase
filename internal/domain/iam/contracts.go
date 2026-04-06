package iam

import (
	"fmt"
	"strings"
)

type AuthMode string

const (
	AuthModeDisabled AuthMode = "disabled"
	AuthModeOIDC     AuthMode = "oidc"
)

func ParseAuthMode(raw string) (AuthMode, error) {
	mode := AuthMode(strings.ToLower(strings.TrimSpace(raw)))
	if mode == "" {
		mode = AuthModeDisabled
	}
	switch mode {
	case AuthModeDisabled, AuthModeOIDC:
		return mode, nil
	default:
		return "", fmt.Errorf("unsupported auth mode %q", raw)
	}
}

func (m AuthMode) String() string { return string(m) }

type ScopeKind string

const (
	ScopeKindInstance     ScopeKind = "instance"
	ScopeKindOrganization ScopeKind = "organization"
	ScopeKindProject      ScopeKind = "project"
)

func ParseScopeKind(raw string) (ScopeKind, error) {
	switch ScopeKind(strings.ToLower(strings.TrimSpace(raw))) {
	case ScopeKindInstance:
		return ScopeKindInstance, nil
	case ScopeKindOrganization:
		return ScopeKindOrganization, nil
	case ScopeKindProject:
		return ScopeKindProject, nil
	default:
		return "", fmt.Errorf("unsupported scope kind %q", raw)
	}
}

type SubjectKind string

const (
	SubjectKindLocalInstanceAdmin  SubjectKind = "local_instance_admin"
	SubjectKindUser                SubjectKind = "user"
	SubjectKindGroup               SubjectKind = "group"
	SubjectKindProjectConversation SubjectKind = "project_conversation"
	SubjectKindTicketAgent         SubjectKind = "ticket_agent"
	SubjectKindMachineAgent        SubjectKind = "machine_agent"
)

func ParseSubjectKind(raw string) (SubjectKind, error) {
	switch SubjectKind(strings.ToLower(strings.TrimSpace(raw))) {
	case SubjectKindLocalInstanceAdmin:
		return SubjectKindLocalInstanceAdmin, nil
	case SubjectKindUser:
		return SubjectKindUser, nil
	case SubjectKindGroup:
		return SubjectKindGroup, nil
	case SubjectKindProjectConversation:
		return SubjectKindProjectConversation, nil
	case SubjectKindTicketAgent:
		return SubjectKindTicketAgent, nil
	case SubjectKindMachineAgent:
		return SubjectKindMachineAgent, nil
	default:
		return "", fmt.Errorf("unsupported subject kind %q", raw)
	}
}

type SubjectRef struct {
	Kind SubjectKind
	Key  string
}

var DefaultLocalInstanceAdminSubject = SubjectRef{
	Kind: SubjectKindLocalInstanceAdmin,
	Key:  "default",
}

func NewSubjectRef(kind SubjectKind, key string) (SubjectRef, error) {
	if _, err := ParseSubjectKind(string(kind)); err != nil {
		return SubjectRef{}, err
	}
	trimmed := strings.TrimSpace(key)
	if trimmed == "" {
		return SubjectRef{}, fmt.Errorf("subject key must not be empty")
	}
	return SubjectRef{Kind: kind, Key: trimmed}, nil
}

func ParseSubjectRef(raw string) (SubjectRef, error) {
	parts := strings.SplitN(strings.TrimSpace(raw), ":", 2)
	if len(parts) != 2 {
		return SubjectRef{}, fmt.Errorf("subject ref must be in kind:key form")
	}
	kind, err := ParseSubjectKind(parts[0])
	if err != nil {
		return SubjectRef{}, err
	}
	return NewSubjectRef(kind, parts[1])
}

func (r SubjectRef) String() string {
	return string(r.Kind) + ":" + r.Key
}

func (r SubjectRef) IsHumanLike() bool {
	switch r.Kind {
	case SubjectKindLocalInstanceAdmin, SubjectKindUser, SubjectKindGroup:
		return true
	default:
		return false
	}
}

type InstanceRole string

const (
	InstanceRoleAdmin InstanceRole = "instance_admin"
)

func ParseInstanceRole(raw string) (InstanceRole, error) {
	switch InstanceRole(strings.ToLower(strings.TrimSpace(raw))) {
	case InstanceRoleAdmin:
		return InstanceRoleAdmin, nil
	default:
		return "", fmt.Errorf("unsupported instance role %q", raw)
	}
}

func (r InstanceRole) String() string { return string(r) }

func (r InstanceRole) ScopeKind() ScopeKind { return ScopeKindInstance }

type OrganizationRole string

const (
	OrganizationRoleOwner  OrganizationRole = "org_owner"
	OrganizationRoleAdmin  OrganizationRole = "org_admin"
	OrganizationRoleMember OrganizationRole = "org_member"
)

func ParseOrganizationRole(raw string) (OrganizationRole, error) {
	switch OrganizationRole(strings.ToLower(strings.TrimSpace(raw))) {
	case OrganizationRoleOwner:
		return OrganizationRoleOwner, nil
	case OrganizationRoleAdmin:
		return OrganizationRoleAdmin, nil
	case OrganizationRoleMember:
		return OrganizationRoleMember, nil
	default:
		return "", fmt.Errorf("unsupported organization role %q", raw)
	}
}

func (r OrganizationRole) String() string { return string(r) }

func (r OrganizationRole) ScopeKind() ScopeKind { return ScopeKindOrganization }

type ProjectRole string

const (
	ProjectRoleAdmin    ProjectRole = "project_admin"
	ProjectRoleOperator ProjectRole = "project_operator"
	ProjectRoleReviewer ProjectRole = "project_reviewer"
	ProjectRoleMember   ProjectRole = "project_member"
	ProjectRoleViewer   ProjectRole = "project_viewer"
)

func ParseProjectRole(raw string) (ProjectRole, error) {
	switch ProjectRole(strings.ToLower(strings.TrimSpace(raw))) {
	case ProjectRoleAdmin:
		return ProjectRoleAdmin, nil
	case ProjectRoleOperator:
		return ProjectRoleOperator, nil
	case ProjectRoleReviewer:
		return ProjectRoleReviewer, nil
	case ProjectRoleMember:
		return ProjectRoleMember, nil
	case ProjectRoleViewer:
		return ProjectRoleViewer, nil
	default:
		return "", fmt.Errorf("unsupported project role %q", raw)
	}
}

func (r ProjectRole) String() string { return string(r) }

func (r ProjectRole) ScopeKind() ScopeKind { return ScopeKindProject }

type MembershipStatus string

const (
	MembershipStatusActive    MembershipStatus = "active"
	MembershipStatusSuspended MembershipStatus = "suspended"
	MembershipStatusRevoked   MembershipStatus = "revoked"
	MembershipStatusLeft      MembershipStatus = "left"
)

func ParseMembershipStatus(raw string) (MembershipStatus, error) {
	switch MembershipStatus(strings.ToLower(strings.TrimSpace(raw))) {
	case MembershipStatusActive:
		return MembershipStatusActive, nil
	case MembershipStatusSuspended:
		return MembershipStatusSuspended, nil
	case MembershipStatusRevoked:
		return MembershipStatusRevoked, nil
	case MembershipStatusLeft:
		return MembershipStatusLeft, nil
	default:
		return "", fmt.Errorf("unsupported membership status %q", raw)
	}
}

func (s MembershipStatus) String() string { return string(s) }

func (s MembershipStatus) CanTransitionTo(next MembershipStatus) bool {
	switch s {
	case MembershipStatusActive:
		return next == MembershipStatusSuspended || next == MembershipStatusRevoked || next == MembershipStatusLeft
	case MembershipStatusSuspended:
		return next == MembershipStatusActive || next == MembershipStatusRevoked
	case MembershipStatusRevoked, MembershipStatusLeft:
		return false
	default:
		return false
	}
}

type InvitationStatus string

const (
	InvitationStatusPending  InvitationStatus = "pending"
	InvitationStatusAccepted InvitationStatus = "accepted"
	InvitationStatusExpired  InvitationStatus = "expired"
	InvitationStatusRevoked  InvitationStatus = "revoked"
)

func ParseInvitationStatus(raw string) (InvitationStatus, error) {
	switch InvitationStatus(strings.ToLower(strings.TrimSpace(raw))) {
	case InvitationStatusPending:
		return InvitationStatusPending, nil
	case InvitationStatusAccepted:
		return InvitationStatusAccepted, nil
	case InvitationStatusExpired:
		return InvitationStatusExpired, nil
	case InvitationStatusRevoked:
		return InvitationStatusRevoked, nil
	default:
		return "", fmt.Errorf("unsupported invitation status %q", raw)
	}
}

func (s InvitationStatus) String() string { return string(s) }

type SessionDevice string

const (
	SessionDeviceBrowser    SessionDevice = "browser"
	SessionDeviceCLI        SessionDevice = "cli"
	SessionDeviceAutomation SessionDevice = "automation"
)

func ParseSessionDevice(raw string) (SessionDevice, error) {
	switch SessionDevice(strings.ToLower(strings.TrimSpace(raw))) {
	case SessionDeviceBrowser:
		return SessionDeviceBrowser, nil
	case SessionDeviceCLI:
		return SessionDeviceCLI, nil
	case SessionDeviceAutomation:
		return SessionDeviceAutomation, nil
	default:
		return "", fmt.Errorf("unsupported session device %q", raw)
	}
}

func (d SessionDevice) String() string { return string(d) }

type AuthModeContract struct {
	Mode                          AuthMode
	RequiresOIDCConfig            bool
	RequiresHumanLogin            bool
	SupportsMultipleHumanUsers    bool
	SupportsUserDirectory         bool
	StableLocalSubject            *SubjectRef
	StableLocalSubjectRole        *InstanceRole
	InteractiveSessionDevice      SessionDevice
	InteractiveSubjectKind        SubjectKind
	AuditSubjectKind              SubjectKind
	ConversationOwnerSubjectKind  SubjectKind
	SupportsBootstrapAdminBinding bool
}

func ContractForMode(mode AuthMode) (AuthModeContract, error) {
	contract, ok := BuiltinAuthModeContracts()[mode]
	if !ok {
		return AuthModeContract{}, fmt.Errorf("unsupported auth mode %q", mode)
	}
	return contract, nil
}

func BuiltinAuthModeContracts() map[AuthMode]AuthModeContract {
	localSubject := DefaultLocalInstanceAdminSubject
	localRole := InstanceRoleAdmin
	return map[AuthMode]AuthModeContract{
		AuthModeDisabled: {
			Mode:                          AuthModeDisabled,
			RequiresOIDCConfig:            false,
			RequiresHumanLogin:            false,
			SupportsMultipleHumanUsers:    false,
			SupportsUserDirectory:         false,
			StableLocalSubject:            &localSubject,
			StableLocalSubjectRole:        &localRole,
			InteractiveSessionDevice:      SessionDeviceBrowser,
			InteractiveSubjectKind:        SubjectKindLocalInstanceAdmin,
			AuditSubjectKind:              SubjectKindLocalInstanceAdmin,
			ConversationOwnerSubjectKind:  SubjectKindLocalInstanceAdmin,
			SupportsBootstrapAdminBinding: false,
		},
		AuthModeOIDC: {
			Mode:                          AuthModeOIDC,
			RequiresOIDCConfig:            true,
			RequiresHumanLogin:            true,
			SupportsMultipleHumanUsers:    true,
			SupportsUserDirectory:         true,
			StableLocalSubject:            nil,
			StableLocalSubjectRole:        nil,
			InteractiveSessionDevice:      SessionDeviceBrowser,
			InteractiveSubjectKind:        SubjectKindUser,
			AuditSubjectKind:              SubjectKindUser,
			ConversationOwnerSubjectKind:  SubjectKindUser,
			SupportsBootstrapAdminBinding: true,
		},
	}
}
