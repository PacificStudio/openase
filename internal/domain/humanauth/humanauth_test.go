package humanauth

import (
	"slices"
	"testing"

	"github.com/google/uuid"
)

func TestAuthenticatedPrincipalActorID(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	principal := AuthenticatedPrincipal{
		User: User{ID: userID},
	}

	if got := principal.ActorID(); got != "user:"+userID.String() {
		t.Fatalf("ActorID() = %q, want %q", got, "user:"+userID.String())
	}
}

func TestParseScopeKind(t *testing.T) {
	t.Parallel()

	valid := []struct {
		raw  string
		want ScopeKind
	}{
		{raw: " instance ", want: ScopeKindInstance},
		{raw: "organization", want: ScopeKindOrganization},
		{raw: "\tproject\n", want: ScopeKindProject},
	}
	for _, tc := range valid {
		got, err := ParseScopeKind(tc.raw)
		if err != nil {
			t.Fatalf("ParseScopeKind(%q) error = %v", tc.raw, err)
		}
		if got != tc.want {
			t.Fatalf("ParseScopeKind(%q) = %q, want %q", tc.raw, got, tc.want)
		}
	}

	if _, err := ParseScopeKind("team"); err == nil {
		t.Fatal("ParseScopeKind() expected unsupported value error")
	}
}

func TestParseSubjectKind(t *testing.T) {
	t.Parallel()

	valid := []struct {
		raw  string
		want SubjectKind
	}{
		{raw: " user ", want: SubjectKindUser},
		{raw: "group", want: SubjectKindGroup},
	}
	for _, tc := range valid {
		got, err := ParseSubjectKind(tc.raw)
		if err != nil {
			t.Fatalf("ParseSubjectKind(%q) error = %v", tc.raw, err)
		}
		if got != tc.want {
			t.Fatalf("ParseSubjectKind(%q) = %q, want %q", tc.raw, got, tc.want)
		}
	}

	if _, err := ParseSubjectKind("service-account"); err == nil {
		t.Fatal("ParseSubjectKind() expected unsupported value error")
	}
}

func TestParseRoleKey(t *testing.T) {
	t.Parallel()

	valid := []struct {
		raw  string
		want RoleKey
	}{
		{raw: " instance_admin ", want: RoleInstanceAdmin},
		{raw: "org_owner", want: RoleOrgOwner},
		{raw: "org_admin", want: RoleOrgAdmin},
		{raw: "org_member", want: RoleOrgMember},
		{raw: "project_admin", want: RoleProjectAdmin},
		{raw: "project_operator", want: RoleProjectOperator},
		{raw: "project_reviewer", want: RoleProjectReviewer},
		{raw: "project_member", want: RoleProjectMember},
		{raw: "project_viewer", want: RoleProjectViewer},
	}
	for _, tc := range valid {
		got, err := ParseRoleKey(tc.raw)
		if err != nil {
			t.Fatalf("ParseRoleKey(%q) error = %v", tc.raw, err)
		}
		if got != tc.want {
			t.Fatalf("ParseRoleKey(%q) = %q, want %q", tc.raw, got, tc.want)
		}
	}

	if _, err := ParseRoleKey("custom_role"); err == nil {
		t.Fatal("ParseRoleKey() expected unsupported role error")
	}
}

func TestBuiltinRoles(t *testing.T) {
	t.Parallel()

	roles := BuiltinRoles()
	expectedKeys := []RoleKey{
		RoleInstanceAdmin,
		RoleOrgOwner,
		RoleOrgAdmin,
		RoleOrgMember,
		RoleProjectAdmin,
		RoleProjectOperator,
		RoleProjectReviewer,
		RoleProjectMember,
		RoleProjectViewer,
	}
	if len(roles) != len(expectedKeys) {
		t.Fatalf("BuiltinRoles() returned %d roles, want %d", len(roles), len(expectedKeys))
	}
	for _, key := range expectedKeys {
		def, ok := roles[key]
		if !ok {
			t.Fatalf("BuiltinRoles() missing %q", key)
		}
		if def.Key != key {
			t.Fatalf("BuiltinRoles()[%q].Key = %q, want %q", key, def.Key, key)
		}
		if len(def.Permissions) == 0 {
			t.Fatalf("BuiltinRoles()[%q] has no permissions", key)
		}
	}

	assertHasPermission(t, roles[RoleInstanceAdmin].Permissions, PermissionOrgUpdate)
	assertHasPermission(t, roles[RoleInstanceAdmin].Permissions, PermissionRBACManage)
	assertHasPermission(t, roles[RoleOrgOwner].Permissions, PermissionOrgUpdate)
	assertHasPermission(t, roles[RoleOrgAdmin].Permissions, PermissionOrgUpdate)
	assertHasPermission(t, roles[RoleOrgMember].Permissions, PermissionTicketUpdate)
	assertMissingPermission(t, roles[RoleOrgMember].Permissions, PermissionRBACManage)
	assertHasPermission(t, roles[RoleProjectAdmin].Permissions, PermissionSecurityManage)
	assertHasPermission(t, roles[RoleProjectOperator].Permissions, PermissionWorkflowManage)
	assertMissingPermission(t, roles[RoleProjectOperator].Permissions, PermissionProposalApprove)
	assertHasPermission(t, roles[RoleProjectReviewer].Permissions, PermissionProposalApprove)
	assertMissingPermission(t, roles[RoleProjectReviewer].Permissions, PermissionProjectUpdate)
	assertHasPermission(t, roles[RoleProjectMember].Permissions, PermissionProjectUpdate)
	assertMissingPermission(t, roles[RoleProjectMember].Permissions, PermissionSecurityManage)
	assertHasPermission(t, roles[RoleProjectViewer].Permissions, PermissionSecurityRead)
	assertMissingPermission(t, roles[RoleProjectViewer].Permissions, PermissionProjectUpdate)
}

func TestPermissionsForRoles(t *testing.T) {
	t.Parallel()

	permissions := PermissionsForRoles([]RoleKey{
		RoleProjectReviewer,
		RoleProjectMember,
		RoleProjectReviewer,
		RoleKey("custom_role"),
	})

	want := []PermissionKey{
		PermissionAgentRead,
		PermissionJobRead,
		PermissionOrgRead,
		PermissionProjectRead,
		PermissionProjectUpdate,
		PermissionProposalApprove,
		PermissionRepoRead,
		PermissionSkillRead,
		PermissionTicketComment,
		PermissionTicketCreate,
		PermissionTicketRead,
		PermissionTicketUpdate,
		PermissionWorkflowRead,
	}
	if !slices.Equal(permissions, want) {
		t.Fatalf("PermissionsForRoles() = %#v, want %#v", permissions, want)
	}
}

func assertHasPermission(t *testing.T, permissions []PermissionKey, want PermissionKey) {
	t.Helper()

	if !slices.Contains(permissions, want) {
		t.Fatalf("permissions %#v do not include %q", permissions, want)
	}
}

func assertMissingPermission(t *testing.T, permissions []PermissionKey, want PermissionKey) {
	t.Helper()

	if slices.Contains(permissions, want) {
		t.Fatalf("permissions %#v unexpectedly include %q", permissions, want)
	}
}
