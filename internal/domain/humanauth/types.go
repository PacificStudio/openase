package humanauth

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusDisabled UserStatus = "disabled"
)

type ScopeKind string

const (
	ScopeKindInstance     ScopeKind = "instance"
	ScopeKindOrganization ScopeKind = "organization"
	ScopeKindProject      ScopeKind = "project"
)

type SubjectKind string

const (
	SubjectKindUser  SubjectKind = "user"
	SubjectKindGroup SubjectKind = "group"
)

type RoleKey string
type PermissionKey string

const (
	PermissionOrgRead         PermissionKey = "org.read"
	PermissionOrgUpdate       PermissionKey = "org.update"
	PermissionProjectRead     PermissionKey = "project.read"
	PermissionProjectUpdate   PermissionKey = "project.update"
	PermissionProjectDelete   PermissionKey = "project.delete"
	PermissionRepoRead        PermissionKey = "repo.read"
	PermissionRepoManage      PermissionKey = "repo.manage"
	PermissionTicketRead      PermissionKey = "ticket.read"
	PermissionTicketCreate    PermissionKey = "ticket.create"
	PermissionTicketUpdate    PermissionKey = "ticket.update"
	PermissionTicketComment   PermissionKey = "ticket.comment"
	PermissionWorkflowRead    PermissionKey = "workflow.read"
	PermissionWorkflowManage  PermissionKey = "workflow.manage"
	PermissionSkillRead       PermissionKey = "skill.read"
	PermissionSkillManage     PermissionKey = "skill.manage"
	PermissionAgentRead       PermissionKey = "agent.read"
	PermissionAgentManage     PermissionKey = "agent.manage"
	PermissionJobRead         PermissionKey = "job.read"
	PermissionJobManage       PermissionKey = "job.manage"
	PermissionSecurityRead    PermissionKey = "security.read"
	PermissionSecurityManage  PermissionKey = "security.manage"
	PermissionProposalApprove PermissionKey = "proposal.approve"
	PermissionRBACManage      PermissionKey = "rbac.manage"
)

const (
	RoleInstanceAdmin   RoleKey = "instance_admin"
	RoleOrgOwner        RoleKey = "org_owner"
	RoleOrgAdmin        RoleKey = "org_admin"
	RoleOrgMember       RoleKey = "org_member"
	RoleProjectAdmin    RoleKey = "project_admin"
	RoleProjectOperator RoleKey = "project_operator"
	RoleProjectReviewer RoleKey = "project_reviewer"
	RoleProjectMember   RoleKey = "project_member"
	RoleProjectViewer   RoleKey = "project_viewer"
)

type User struct {
	ID           uuid.UUID
	Status       UserStatus
	PrimaryEmail string
	DisplayName  string
	AvatarURL    string
	LastLoginAt  *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type UserIdentity struct {
	ID            uuid.UUID
	UserID        uuid.UUID
	Issuer        string
	Subject       string
	Email         string
	EmailVerified bool
	ClaimsVersion int
	RawClaimsJSON string
	LastSyncedAt  time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type UserGroupMembership struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	Issuer       string
	GroupKey     string
	GroupName    string
	LastSyncedAt time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type BrowserSession struct {
	ID            uuid.UUID
	UserID        uuid.UUID
	SessionHash   string
	ExpiresAt     time.Time
	IdleExpiresAt time.Time
	CSRFSecret    string
	UserAgentHash string
	IPPrefix      string
	RevokedAt     *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type RoleBinding struct {
	ID          uuid.UUID
	ScopeKind   ScopeKind
	ScopeID     string
	SubjectKind SubjectKind
	SubjectKey  string
	RoleKey     RoleKey
	GrantedBy   string
	ExpiresAt   *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type ApprovalPolicyRule struct {
	ID                  uuid.UUID
	ScopeKind           ScopeKind
	ScopeID             string
	ActionKey           string
	RequireRoleKey      string
	RequireTicketStatus string
	Enabled             bool
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

type OIDCProfile struct {
	Issuer        string
	Subject       string
	Email         string
	EmailVerified bool
	DisplayName   string
	Username      string
	AvatarURL     string
	Groups        []Group
	RawClaimsJSON string
}

type Group struct {
	Key  string
	Name string
}

type AuthenticatedPrincipal struct {
	User           User
	Identity       UserIdentity
	Groups         []UserGroupMembership
	Session        BrowserSession
	EffectiveRoles []RoleKey
	Permissions    []PermissionKey
}

type ScopeRef struct {
	Kind ScopeKind
	ID   string
}

func (p AuthenticatedPrincipal) ActorID() string {
	return "user:" + p.User.ID.String()
}

func ParseScopeKind(raw string) (ScopeKind, error) {
	switch ScopeKind(strings.TrimSpace(raw)) {
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

func ParseSubjectKind(raw string) (SubjectKind, error) {
	switch SubjectKind(strings.TrimSpace(raw)) {
	case SubjectKindUser:
		return SubjectKindUser, nil
	case SubjectKindGroup:
		return SubjectKindGroup, nil
	default:
		return "", fmt.Errorf("unsupported subject kind %q", raw)
	}
}

func ParseRoleKey(raw string) (RoleKey, error) {
	role := RoleKey(strings.TrimSpace(raw))
	if _, ok := BuiltinRoles()[role]; !ok {
		return "", fmt.Errorf("unsupported role %q", raw)
	}
	return role, nil
}
