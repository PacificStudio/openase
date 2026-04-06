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

func TestParseRoleKeyForScope(t *testing.T) {
	t.Parallel()

	valid := []struct {
		scope ScopeKind
		raw   string
		want  RoleKey
	}{
		{scope: ScopeKindInstance, raw: " instance_admin ", want: RoleInstanceAdmin},
		{scope: ScopeKindOrganization, raw: "org_admin", want: RoleOrgAdmin},
		{scope: ScopeKindProject, raw: "project_member", want: RoleProjectMember},
	}
	for _, tc := range valid {
		got, err := ParseRoleKeyForScope(tc.scope, tc.raw)
		if err != nil {
			t.Fatalf("ParseRoleKeyForScope(%q, %q) error = %v", tc.scope, tc.raw, err)
		}
		if got != tc.want {
			t.Fatalf("ParseRoleKeyForScope(%q, %q) = %q, want %q", tc.scope, tc.raw, got, tc.want)
		}
	}

	if _, err := ParseRoleKeyForScope(ScopeKindProject, "instance_admin"); err == nil {
		t.Fatal("ParseRoleKeyForScope() expected scope-specific error")
	}
	if _, err := ParseRoleKeyForScope(ScopeKind("workspace"), "project_member"); err == nil {
		t.Fatal("ParseRoleKeyForScope() expected unsupported scope error")
	}
}

func TestRoleKeysForScope(t *testing.T) {
	t.Parallel()

	if got := RoleKeysForScope(ScopeKindInstance); !slices.Equal(got, []RoleKey{RoleInstanceAdmin}) {
		t.Fatalf("RoleKeysForScope(instance) = %#v", got)
	}
	if got := RoleKeysForScope(ScopeKindOrganization); !slices.Equal(got, []RoleKey{RoleOrgOwner, RoleOrgAdmin, RoleOrgMember}) {
		t.Fatalf("RoleKeysForScope(organization) = %#v", got)
	}
	if got := RoleKeysForScope(ScopeKindProject); !slices.Equal(got, []RoleKey{
		RoleProjectAdmin,
		RoleProjectOperator,
		RoleProjectReviewer,
		RoleProjectMember,
		RoleProjectViewer,
	}) {
		t.Fatalf("RoleKeysForScope(project) = %#v", got)
	}
	if got := RoleKeysForScope(ScopeKind("workspace")); got != nil {
		t.Fatalf("RoleKeysForScope(unknown) = %#v, want nil", got)
	}
}

func TestInstanceAndOrganizationRoleHelpers(t *testing.T) {
	t.Parallel()

	if InstanceRoleAdmin.String() != "instance_admin" || InstanceRoleAdmin.ScopeKind() != ScopeKindInstance {
		t.Fatal("instance role helpers should expose role key and scope")
	}
	if role, err := ParseInstanceRole(" instance_admin "); err != nil || role != InstanceRoleAdmin {
		t.Fatalf("ParseInstanceRole() = %q, %v", role, err)
	}
	if _, err := ParseInstanceRole("org_admin"); err == nil {
		t.Fatal("ParseInstanceRole() expected unsupported role error")
	}

	for _, role := range []OrganizationRole{
		OrganizationRoleOwner,
		OrganizationRoleAdmin,
		OrganizationRoleMember,
	} {
		parsed, err := ParseOrganizationRole(role.String())
		if err != nil {
			t.Fatalf("ParseOrganizationRole(%q) error = %v", role, err)
		}
		if parsed != role {
			t.Fatalf("ParseOrganizationRole(%q) = %q, want %q", role, parsed, role)
		}
		if role.ScopeKind() != ScopeKindOrganization {
			t.Fatalf("OrganizationRole.ScopeKind() = %q", role.ScopeKind())
		}
	}
	if _, err := ParseOrganizationRole("project_member"); err == nil {
		t.Fatal("ParseOrganizationRole() expected unsupported role error")
	}
}

func TestSubjectRefsCanonicalizeKeys(t *testing.T) {
	t.Parallel()

	userRef, err := NewSubjectRef(SubjectKindUser, "  USER@Example.com ")
	if err != nil {
		t.Fatalf("NewSubjectRef(user) error = %v", err)
	}
	if userRef.Key != "user@example.com" {
		t.Fatalf("user subject key = %q, want %q", userRef.Key, "user@example.com")
	}

	groupRef, err := ParseGroupSubjectRef(" Platform-Admins ")
	if err != nil {
		t.Fatalf("ParseGroupSubjectRef() error = %v", err)
	}
	if groupRef.Key != "platform-admins" {
		t.Fatalf("group subject key = %q, want %q", groupRef.Key, "platform-admins")
	}
	if groupRef.String() != "group:platform-admins" {
		t.Fatalf("group subject string = %q", groupRef.String())
	}

	if _, err := NewSubjectRef(SubjectKindUser, "   "); err == nil {
		t.Fatal("NewSubjectRef() expected empty subject error")
	}
}

func TestParseScopedRoleBindings(t *testing.T) {
	t.Parallel()

	instanceBinding, err := ParseInstanceRoleBinding(RoleBinding{
		ScopeKind:   ScopeKindInstance,
		SubjectKind: SubjectKindUser,
		SubjectKey:  uuid.Nil.String(),
		RoleKey:     RoleInstanceAdmin,
	})
	if err != nil {
		t.Fatalf("ParseInstanceRoleBinding() error = %v", err)
	}
	if instanceBinding.RoleKey != InstanceRoleAdmin {
		t.Fatalf("instance role = %q", instanceBinding.RoleKey)
	}

	orgBinding, err := ParseOrganizationRoleBinding(RoleBinding{
		ScopeKind:   ScopeKindOrganization,
		ScopeID:     "org-1",
		SubjectKind: SubjectKindGroup,
		SubjectKey:  "platform-admins",
		RoleKey:     RoleOrgAdmin,
	})
	if err != nil {
		t.Fatalf("ParseOrganizationRoleBinding() error = %v", err)
	}
	if orgBinding.OrganizationID != "org-1" || orgBinding.RoleKey != OrganizationRoleAdmin {
		t.Fatalf("organization binding = %#v", orgBinding)
	}

	projectBinding, err := ParseProjectRoleBinding(RoleBinding{
		ScopeKind:   ScopeKindProject,
		ScopeID:     "project-1",
		SubjectKind: SubjectKindUser,
		SubjectKey:  uuid.New().String(),
		RoleKey:     RoleProjectMember,
	})
	if err != nil {
		t.Fatalf("ParseProjectRoleBinding() error = %v", err)
	}
	if projectBinding.ProjectID != "project-1" || projectBinding.RoleKey != ProjectRoleMember {
		t.Fatalf("project binding = %#v", projectBinding)
	}

	if _, err := ParseProjectRoleBinding(RoleBinding{
		ScopeKind:   ScopeKindProject,
		ScopeID:     "project-1",
		SubjectKind: SubjectKindUser,
		SubjectKey:  uuid.New().String(),
		RoleKey:     RoleInstanceAdmin,
	}); err == nil {
		t.Fatal("ParseProjectRoleBinding() expected invalid role error")
	}

	if _, err := ParseInstanceRoleBinding(RoleBinding{
		ScopeKind:   ScopeKindOrganization,
		ScopeID:     "org-1",
		SubjectKind: SubjectKindUser,
		SubjectKey:  uuid.New().String(),
		RoleKey:     RoleInstanceAdmin,
	}); err == nil {
		t.Fatal("ParseInstanceRoleBinding() expected invalid scope error")
	}
	if _, err := ParseInstanceRoleBinding(RoleBinding{
		ScopeKind:   ScopeKindInstance,
		SubjectKind: SubjectKindUser,
		SubjectKey:  uuid.New().String(),
		RoleKey:     RoleProjectMember,
	}); err == nil {
		t.Fatal("ParseInstanceRoleBinding() expected invalid role error")
	}
	if _, err := ParseInstanceRoleBinding(RoleBinding{
		ScopeKind:   ScopeKindInstance,
		SubjectKind: SubjectKind("service-account"),
		SubjectKey:  "svc",
		RoleKey:     RoleInstanceAdmin,
	}); err == nil {
		t.Fatal("ParseInstanceRoleBinding() expected invalid subject kind error")
	}
	if _, err := ParseOrganizationRoleBinding(RoleBinding{
		ScopeKind:   ScopeKindInstance,
		ScopeID:     "org-1",
		SubjectKind: SubjectKindGroup,
		SubjectKey:  "platform-admins",
		RoleKey:     RoleOrgAdmin,
	}); err == nil {
		t.Fatal("ParseOrganizationRoleBinding() expected invalid scope error")
	}
	if _, err := ParseOrganizationRoleBinding(RoleBinding{
		ScopeKind:   ScopeKindOrganization,
		SubjectKind: SubjectKindGroup,
		SubjectKey:  "platform-admins",
		RoleKey:     RoleOrgAdmin,
	}); err == nil {
		t.Fatal("ParseOrganizationRoleBinding() expected missing scope id error")
	}
	if _, err := ParseOrganizationRoleBinding(RoleBinding{
		ScopeKind:   ScopeKindOrganization,
		ScopeID:     "org-1",
		SubjectKind: SubjectKind("service-account"),
		SubjectKey:  "svc",
		RoleKey:     RoleOrgAdmin,
	}); err == nil {
		t.Fatal("ParseOrganizationRoleBinding() expected invalid subject kind error")
	}
	if _, err := ParseOrganizationRoleBinding(RoleBinding{
		ScopeKind:   ScopeKindOrganization,
		ScopeID:     "org-1",
		SubjectKind: SubjectKindGroup,
		SubjectKey:  "platform-admins",
		RoleKey:     RoleProjectMember,
	}); err == nil {
		t.Fatal("ParseOrganizationRoleBinding() expected invalid role error")
	}
	if _, err := ParseProjectRoleBinding(RoleBinding{
		ScopeKind:   ScopeKindProject,
		SubjectKind: SubjectKindUser,
		SubjectKey:  uuid.New().String(),
		RoleKey:     RoleProjectMember,
	}); err == nil {
		t.Fatal("ParseProjectRoleBinding() expected missing scope id error")
	}
	if _, err := ParseProjectRoleBinding(RoleBinding{
		ScopeKind:   ScopeKindOrganization,
		ScopeID:     "project-1",
		SubjectKind: SubjectKindUser,
		SubjectKey:  uuid.New().String(),
		RoleKey:     RoleProjectMember,
	}); err == nil {
		t.Fatal("ParseProjectRoleBinding() expected invalid scope error")
	}
	if _, err := ParseProjectRoleBinding(RoleBinding{
		ScopeKind:   ScopeKindProject,
		ScopeID:     "project-1",
		SubjectKind: SubjectKind("service-account"),
		SubjectKey:  "svc",
		RoleKey:     RoleProjectMember,
	}); err == nil {
		t.Fatal("ParseProjectRoleBinding() expected invalid subject kind error")
	}
}

func TestScopedRoleHelpersAndGenericMappings(t *testing.T) {
	t.Parallel()

	if OrganizationRoleAdmin.String() != "org_admin" || OrganizationRoleAdmin.ScopeKind() != ScopeKindOrganization {
		t.Fatal("organization role helpers should expose role key and scope")
	}
	if ProjectRoleReviewer.String() != "project_reviewer" || ProjectRoleReviewer.ScopeKind() != ScopeKindProject {
		t.Fatal("project role helpers should expose role key and scope")
	}

	subject := NewUserSubjectRef(uuid.New())
	instance := InstanceRoleBinding{
		RoleBindingMetadata: RoleBindingMetadata{Subject: subject, GrantedBy: "user:admin"},
		RoleKey:             InstanceRoleAdmin,
	}
	if generic := instance.Generic(); generic.ScopeKind != ScopeKindInstance || generic.SubjectKey != subject.Key || generic.RoleKey != RoleInstanceAdmin {
		t.Fatalf("instance Generic() = %#v", generic)
	}

	org := OrganizationRoleBinding{
		RoleBindingMetadata: RoleBindingMetadata{Subject: subject, GrantedBy: "user:admin"},
		OrganizationID:      "org-1",
		RoleKey:             OrganizationRoleMember,
	}
	if generic := org.Generic(); generic.ScopeKind != ScopeKindOrganization || generic.ScopeID != "org-1" || generic.RoleKey != RoleOrgMember {
		t.Fatalf("organization Generic() = %#v", generic)
	}

	project := ProjectRoleBinding{
		RoleBindingMetadata: RoleBindingMetadata{Subject: subject, GrantedBy: "user:admin"},
		ProjectID:           "project-1",
		RoleKey:             ProjectRoleViewer,
	}
	if generic := project.Generic(); generic.ScopeKind != ScopeKindProject || generic.ScopeID != "project-1" || generic.RoleKey != RoleProjectViewer {
		t.Fatalf("project Generic() = %#v", generic)
	}
}

func TestParseProjectRoleCoversAllVariants(t *testing.T) {
	t.Parallel()

	for _, role := range []ProjectRole{
		ProjectRoleAdmin,
		ProjectRoleOperator,
		ProjectRoleReviewer,
		ProjectRoleMember,
		ProjectRoleViewer,
	} {
		parsed, err := ParseProjectRole(role.String())
		if err != nil {
			t.Fatalf("ParseProjectRole(%q) error = %v", role, err)
		}
		if parsed != role {
			t.Fatalf("ParseProjectRole(%q) = %q, want %q", role, parsed, role)
		}
	}

	if _, err := ParseProjectRole("org_admin"); err == nil {
		t.Fatal("ParseProjectRole() expected unsupported role error")
	}
}

func TestNormalizeSubjectKeyFallbackPreservesUnknownKinds(t *testing.T) {
	t.Parallel()

	if got := normalizeSubjectKey(SubjectKind("machine"), "  Mixed-Case "); got != "Mixed-Case" {
		t.Fatalf("normalizeSubjectKey(unknown) = %q, want %q", got, "Mixed-Case")
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

	assertHasPermission(t, roles[RoleInstanceAdmin].Permissions, PermissionOrgCreate)
	assertHasPermission(t, roles[RoleInstanceAdmin].Permissions, PermissionRBACManage)
	assertHasPermission(t, roles[RoleOrgOwner].Permissions, PermissionOrgDelete)
	assertHasPermission(t, roles[RoleOrgAdmin].Permissions, PermissionOrgUpdate)
	assertHasPermission(t, roles[RoleOrgMember].Permissions, PermissionTicketUpdate)
	assertMissingPermission(t, roles[RoleOrgMember].Permissions, PermissionRBACManage)
	assertHasPermission(t, roles[RoleProjectAdmin].Permissions, PermissionSecurityUpdate)
	assertHasPermission(t, roles[RoleProjectOperator].Permissions, PermissionWorkflowUpdate)
	assertMissingPermission(t, roles[RoleProjectOperator].Permissions, PermissionProposalApprove)
	assertHasPermission(t, roles[RoleProjectReviewer].Permissions, PermissionProposalApprove)
	assertMissingPermission(t, roles[RoleProjectReviewer].Permissions, PermissionProjectUpdate)
	assertHasPermission(t, roles[RoleProjectMember].Permissions, PermissionTicketCommentUpdate)
	assertMissingPermission(t, roles[RoleProjectMember].Permissions, PermissionSecurityUpdate)
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
		PermissionConversationCreate,
		PermissionConversationRead,
		PermissionConversationUpdate,
		PermissionHarnessRead,
		PermissionNotificationRead,
		PermissionOrgRead,
		PermissionProjectRead,
		PermissionProjectUpdateCreate,
		PermissionProjectUpdateRead,
		PermissionProjectUpdateUpdate,
		PermissionProposalApprove,
		PermissionRepoRead,
		PermissionJobRead,
		PermissionSecurityRead,
		PermissionSkillRead,
		PermissionStatusRead,
		PermissionTicketCreate,
		PermissionTicketRead,
		PermissionTicketUpdate,
		PermissionTicketCommentCreate,
		PermissionTicketCommentRead,
		PermissionTicketCommentUpdate,
		PermissionWorkflowRead,
	}
	if !slices.Equal(permissions, want) {
		t.Fatalf("PermissionsForRoles() = %#v, want %#v", permissions, want)
	}
}

func TestProjectRolePermissionMatrix(t *testing.T) {
	t.Parallel()

	roles := BuiltinRoles()
	cases := []struct {
		name       string
		role       RoleKey
		permission PermissionKey
		allowed    bool
	}{
		{name: "viewer can read repo", role: RoleProjectViewer, permission: PermissionRepoRead, allowed: true},
		{name: "viewer cannot create ticket", role: RoleProjectViewer, permission: PermissionTicketCreate, allowed: false},
		{name: "member can update ticket", role: RoleProjectMember, permission: PermissionTicketUpdate, allowed: true},
		{name: "member cannot delete workflow", role: RoleProjectMember, permission: PermissionWorkflowDelete, allowed: false},
		{name: "admin can delete workflow", role: RoleProjectAdmin, permission: PermissionWorkflowDelete, allowed: true},
		{name: "admin can update security setting", role: RoleProjectAdmin, permission: PermissionSecurityUpdate, allowed: true},
		{name: "owner can delete org", role: RoleOrgOwner, permission: PermissionOrgDelete, allowed: true},
		{name: "viewer cannot delete conversation", role: RoleProjectViewer, permission: PermissionConversationDelete, allowed: false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := slices.Contains(roles[tc.role].Permissions, tc.permission)
			if got != tc.allowed {
				t.Fatalf("%s permission %q = %t, want %t", tc.role, tc.permission, got, tc.allowed)
			}
		})
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
