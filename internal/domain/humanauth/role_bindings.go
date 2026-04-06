package humanauth

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type SubjectRef struct {
	Kind SubjectKind
	Key  string
}

func NewSubjectRef(kind SubjectKind, key string) (SubjectRef, error) {
	if _, err := ParseSubjectKind(string(kind)); err != nil {
		return SubjectRef{}, err
	}
	normalized := normalizeSubjectKey(kind, key)
	if normalized == "" {
		return SubjectRef{}, fmt.Errorf("subject key must not be empty")
	}
	return SubjectRef{Kind: kind, Key: normalized}, nil
}

func NewUserSubjectRef(userID uuid.UUID) SubjectRef {
	return SubjectRef{Kind: SubjectKindUser, Key: userID.String()}
}

func ParseGroupSubjectRef(raw string) (SubjectRef, error) {
	return NewSubjectRef(SubjectKindGroup, raw)
}

func (r SubjectRef) String() string {
	return string(r.Kind) + ":" + r.Key
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

func (r InstanceRole) String() string       { return string(r) }
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

func (r OrganizationRole) String() string       { return string(r) }
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

func (r ProjectRole) String() string       { return string(r) }
func (r ProjectRole) ScopeKind() ScopeKind { return ScopeKindProject }

func ParseRoleKeyForScope(scope ScopeKind, raw string) (RoleKey, error) {
	switch scope {
	case ScopeKindInstance:
		role, err := ParseInstanceRole(raw)
		return RoleKey(role), err
	case ScopeKindOrganization:
		role, err := ParseOrganizationRole(raw)
		return RoleKey(role), err
	case ScopeKindProject:
		role, err := ParseProjectRole(raw)
		return RoleKey(role), err
	default:
		return "", fmt.Errorf("unsupported scope kind %q", scope)
	}
}

func RoleKeysForScope(scope ScopeKind) []RoleKey {
	switch scope {
	case ScopeKindInstance:
		return []RoleKey{RoleInstanceAdmin}
	case ScopeKindOrganization:
		return []RoleKey{RoleOrgOwner, RoleOrgAdmin, RoleOrgMember}
	case ScopeKindProject:
		return []RoleKey{
			RoleProjectAdmin,
			RoleProjectOperator,
			RoleProjectReviewer,
			RoleProjectMember,
			RoleProjectViewer,
		}
	default:
		return nil
	}
}

type RoleBindingMetadata struct {
	ID        uuid.UUID
	Subject   SubjectRef
	GrantedBy string
	ExpiresAt *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

type InstanceRoleBinding struct {
	RoleBindingMetadata
	RoleKey InstanceRole
}

type OrganizationRoleBinding struct {
	RoleBindingMetadata
	OrganizationID string
	RoleKey        OrganizationRole
}

type ProjectRoleBinding struct {
	RoleBindingMetadata
	ProjectID string
	RoleKey   ProjectRole
}

type UpdateRoleBindingMetadata struct {
	Subject   SubjectRef
	GrantedBy string
	ExpiresAt *time.Time
}

type UpdateInstanceRoleBinding struct {
	UpdateRoleBindingMetadata
	RoleKey InstanceRole
}

type UpdateOrganizationRoleBinding struct {
	UpdateRoleBindingMetadata
	RoleKey OrganizationRole
}

type UpdateProjectRoleBinding struct {
	UpdateRoleBindingMetadata
	RoleKey ProjectRole
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

func (b RoleBinding) SubjectRef() (SubjectRef, error) {
	return NewSubjectRef(b.SubjectKind, b.SubjectKey)
}

func (b InstanceRoleBinding) Generic() RoleBinding {
	return RoleBinding{
		ID:          b.ID,
		ScopeKind:   ScopeKindInstance,
		ScopeID:     "",
		SubjectKind: b.Subject.Kind,
		SubjectKey:  b.Subject.Key,
		RoleKey:     RoleKey(b.RoleKey),
		GrantedBy:   b.GrantedBy,
		ExpiresAt:   b.ExpiresAt,
		CreatedAt:   b.CreatedAt,
		UpdatedAt:   b.UpdatedAt,
	}
}

func (b OrganizationRoleBinding) Generic() RoleBinding {
	return RoleBinding{
		ID:          b.ID,
		ScopeKind:   ScopeKindOrganization,
		ScopeID:     b.OrganizationID,
		SubjectKind: b.Subject.Kind,
		SubjectKey:  b.Subject.Key,
		RoleKey:     RoleKey(b.RoleKey),
		GrantedBy:   b.GrantedBy,
		ExpiresAt:   b.ExpiresAt,
		CreatedAt:   b.CreatedAt,
		UpdatedAt:   b.UpdatedAt,
	}
}

func (b ProjectRoleBinding) Generic() RoleBinding {
	return RoleBinding{
		ID:          b.ID,
		ScopeKind:   ScopeKindProject,
		ScopeID:     b.ProjectID,
		SubjectKind: b.Subject.Kind,
		SubjectKey:  b.Subject.Key,
		RoleKey:     RoleKey(b.RoleKey),
		GrantedBy:   b.GrantedBy,
		ExpiresAt:   b.ExpiresAt,
		CreatedAt:   b.CreatedAt,
		UpdatedAt:   b.UpdatedAt,
	}
}

func ParseInstanceRoleBinding(input RoleBinding) (InstanceRoleBinding, error) {
	if input.ScopeKind != ScopeKindInstance || strings.TrimSpace(input.ScopeID) != "" {
		return InstanceRoleBinding{}, fmt.Errorf("instance role binding must use instance scope")
	}
	subject, err := input.SubjectRef()
	if err != nil {
		return InstanceRoleBinding{}, err
	}
	role, err := ParseInstanceRole(string(input.RoleKey))
	if err != nil {
		return InstanceRoleBinding{}, err
	}
	return InstanceRoleBinding{
		RoleBindingMetadata: RoleBindingMetadata{
			ID:        input.ID,
			Subject:   subject,
			GrantedBy: input.GrantedBy,
			ExpiresAt: input.ExpiresAt,
			CreatedAt: input.CreatedAt,
			UpdatedAt: input.UpdatedAt,
		},
		RoleKey: role,
	}, nil
}

func ParseOrganizationRoleBinding(input RoleBinding) (OrganizationRoleBinding, error) {
	if input.ScopeKind != ScopeKindOrganization || strings.TrimSpace(input.ScopeID) == "" {
		return OrganizationRoleBinding{}, fmt.Errorf("organization role binding must use organization scope")
	}
	subject, err := input.SubjectRef()
	if err != nil {
		return OrganizationRoleBinding{}, err
	}
	role, err := ParseOrganizationRole(string(input.RoleKey))
	if err != nil {
		return OrganizationRoleBinding{}, err
	}
	return OrganizationRoleBinding{
		RoleBindingMetadata: RoleBindingMetadata{
			ID:        input.ID,
			Subject:   subject,
			GrantedBy: input.GrantedBy,
			ExpiresAt: input.ExpiresAt,
			CreatedAt: input.CreatedAt,
			UpdatedAt: input.UpdatedAt,
		},
		OrganizationID: strings.TrimSpace(input.ScopeID),
		RoleKey:        role,
	}, nil
}

func ParseProjectRoleBinding(input RoleBinding) (ProjectRoleBinding, error) {
	if input.ScopeKind != ScopeKindProject || strings.TrimSpace(input.ScopeID) == "" {
		return ProjectRoleBinding{}, fmt.Errorf("project role binding must use project scope")
	}
	subject, err := input.SubjectRef()
	if err != nil {
		return ProjectRoleBinding{}, err
	}
	role, err := ParseProjectRole(string(input.RoleKey))
	if err != nil {
		return ProjectRoleBinding{}, err
	}
	return ProjectRoleBinding{
		RoleBindingMetadata: RoleBindingMetadata{
			ID:        input.ID,
			Subject:   subject,
			GrantedBy: input.GrantedBy,
			ExpiresAt: input.ExpiresAt,
			CreatedAt: input.CreatedAt,
			UpdatedAt: input.UpdatedAt,
		},
		ProjectID: strings.TrimSpace(input.ScopeID),
		RoleKey:   role,
	}, nil
}

func normalizeSubjectKey(kind SubjectKind, raw string) string {
	trimmed := strings.TrimSpace(raw)
	switch kind {
	case SubjectKindUser:
		return strings.ToLower(trimmed)
	case SubjectKindGroup:
		return strings.ToLower(trimmed)
	default:
		return trimmed
	}
}
