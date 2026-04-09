package humanauth

import (
	"context"
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
	PermissionOrgRead   PermissionKey = "org.read"
	PermissionOrgCreate PermissionKey = "org.create"
	PermissionOrgUpdate PermissionKey = "org.update"
	PermissionOrgDelete PermissionKey = "org.delete"

	PermissionProjectRead   PermissionKey = "project.read"
	PermissionProjectCreate PermissionKey = "project.create"
	PermissionProjectUpdate PermissionKey = "project.update"
	PermissionProjectDelete PermissionKey = "project.delete"

	PermissionRepoRead   PermissionKey = "repo.read"
	PermissionRepoCreate PermissionKey = "repo.create"
	PermissionRepoUpdate PermissionKey = "repo.update"
	PermissionRepoDelete PermissionKey = "repo.delete"

	PermissionTicketRead   PermissionKey = "ticket.read"
	PermissionTicketCreate PermissionKey = "ticket.create"
	PermissionTicketUpdate PermissionKey = "ticket.update"
	PermissionTicketDelete PermissionKey = "ticket.delete"

	PermissionTicketCommentRead   PermissionKey = "ticket_comment.read"
	PermissionTicketCommentCreate PermissionKey = "ticket_comment.create"
	PermissionTicketCommentUpdate PermissionKey = "ticket_comment.update"
	PermissionTicketCommentDelete PermissionKey = "ticket_comment.delete"

	PermissionProjectUpdateRead   PermissionKey = "project_update.read"
	PermissionProjectUpdateCreate PermissionKey = "project_update.create"
	PermissionProjectUpdateUpdate PermissionKey = "project_update.update"
	PermissionProjectUpdateDelete PermissionKey = "project_update.delete"

	PermissionWorkflowRead   PermissionKey = "workflow.read"
	PermissionWorkflowCreate PermissionKey = "workflow.create"
	PermissionWorkflowUpdate PermissionKey = "workflow.update"
	PermissionWorkflowDelete PermissionKey = "workflow.delete"

	PermissionHarnessRead   PermissionKey = "harness.read"
	PermissionHarnessUpdate PermissionKey = "harness.update"

	PermissionStatusRead   PermissionKey = "status.read"
	PermissionStatusCreate PermissionKey = "status.create"
	PermissionStatusUpdate PermissionKey = "status.update"
	PermissionStatusDelete PermissionKey = "status.delete"

	PermissionSkillRead   PermissionKey = "skill.read"
	PermissionSkillCreate PermissionKey = "skill.create"
	PermissionSkillUpdate PermissionKey = "skill.update"
	PermissionSkillDelete PermissionKey = "skill.delete"

	PermissionAgentRead    PermissionKey = "agent.read"
	PermissionAgentCreate  PermissionKey = "agent.create"
	PermissionAgentUpdate  PermissionKey = "agent.update"
	PermissionAgentDelete  PermissionKey = "agent.delete"
	PermissionAgentControl PermissionKey = "agent.control"

	PermissionJobRead    PermissionKey = "scheduled_job.read"
	PermissionJobCreate  PermissionKey = "scheduled_job.create"
	PermissionJobUpdate  PermissionKey = "scheduled_job.update"
	PermissionJobDelete  PermissionKey = "scheduled_job.delete"
	PermissionJobTrigger PermissionKey = "scheduled_job.trigger"

	PermissionSecurityRead   PermissionKey = "security_setting.read"
	PermissionSecurityUpdate PermissionKey = "security_setting.update"

	PermissionNotificationRead   PermissionKey = "notification.read"
	PermissionNotificationCreate PermissionKey = "notification.create"
	PermissionNotificationUpdate PermissionKey = "notification.update"
	PermissionNotificationDelete PermissionKey = "notification.delete"

	PermissionConversationRead   PermissionKey = "conversation.read"
	PermissionConversationCreate PermissionKey = "conversation.create"
	PermissionConversationUpdate PermissionKey = "conversation.update"
	PermissionConversationDelete PermissionKey = "conversation.delete"

	PermissionMachineRead   PermissionKey = "machine.read"
	PermissionMachineCreate PermissionKey = "machine.create"
	PermissionMachineUpdate PermissionKey = "machine.update"
	PermissionMachineDelete PermissionKey = "machine.delete"

	PermissionProviderRead   PermissionKey = "provider.read"
	PermissionProviderCreate PermissionKey = "provider.create"
	PermissionProviderUpdate PermissionKey = "provider.update"
	PermissionProviderDelete PermissionKey = "provider.delete"

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
	DeviceKind    SessionDeviceKind
	DeviceOS      string
	DeviceBrowser string
	DeviceLabel   string
	ExpiresAt     time.Time
	IdleExpiresAt time.Time
	CSRFSecret    string
	UserAgentHash string
	IPPrefix      string
	RevokedAt     *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type SessionDeviceKind string

const (
	SessionDeviceKindUnknown SessionDeviceKind = "unknown"
	SessionDeviceKindDesktop SessionDeviceKind = "desktop"
	SessionDeviceKindMobile  SessionDeviceKind = "mobile"
	SessionDeviceKindTablet  SessionDeviceKind = "tablet"
	SessionDeviceKindBot     SessionDeviceKind = "bot"
)

type AuthAuditEventType string

const (
	AuthAuditLoginSucceeded         AuthAuditEventType = "login.success"
	AuthAuditLoginFailed            AuthAuditEventType = "login.failed"
	AuthAuditLogout                 AuthAuditEventType = "logout"
	AuthAuditSessionRevoked         AuthAuditEventType = "session.revoked"
	AuthAuditSessionExpired         AuthAuditEventType = "session.expired"
	AuthAuditUserEnabled            AuthAuditEventType = "user.enabled"
	AuthAuditUserDisabled           AuthAuditEventType = "user.disabled"
	AuthAuditUserDisabledAfterLogin AuthAuditEventType = "user.disabled_after_login"
	AuthAuditLocalBootstrapIssued   AuthAuditEventType = "local_bootstrap.issued"
	AuthAuditLocalBootstrapRedeemed AuthAuditEventType = "local_bootstrap.redeemed"
	AuthAuditLocalBootstrapFailed   AuthAuditEventType = "local_bootstrap.failed"
)

type AuthAuditEvent struct {
	ID        uuid.UUID
	UserID    *uuid.UUID
	SessionID *uuid.UUID
	ActorID   string
	EventType AuthAuditEventType
	Message   string
	Metadata  map[string]any
	CreatedAt time.Time
}

type LocalBootstrapAuthRequest struct {
	ID            uuid.UUID
	CodeHash      string
	NonceHash     string
	Purpose       string
	RequestedBy   string
	ExpiresAt     time.Time
	UsedSessionID *uuid.UUID
	UsedAt        *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
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

type principalContextKey struct{}

func WithPrincipal(ctx context.Context, principal AuthenticatedPrincipal) context.Context {
	return context.WithValue(ctx, principalContextKey{}, principal)
}

func PrincipalFromContext(ctx context.Context) (AuthenticatedPrincipal, bool) {
	value := ctx.Value(principalContextKey{})
	principal, ok := value.(AuthenticatedPrincipal)
	return principal, ok
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
