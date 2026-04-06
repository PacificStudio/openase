package iam

import "testing"

func TestParseAuthMode(t *testing.T) {
	t.Parallel()

	valid := []struct {
		raw  string
		want AuthMode
	}{
		{raw: "", want: AuthModeDisabled},
		{raw: " disabled ", want: AuthModeDisabled},
		{raw: "OIDC", want: AuthModeOIDC},
	}
	for _, tc := range valid {
		got, err := ParseAuthMode(tc.raw)
		if err != nil {
			t.Fatalf("ParseAuthMode(%q) error = %v", tc.raw, err)
		}
		if got != tc.want {
			t.Fatalf("ParseAuthMode(%q) = %q, want %q", tc.raw, got, tc.want)
		}
	}

	if _, err := ParseAuthMode("saml"); err == nil {
		t.Fatal("ParseAuthMode() expected unsupported auth mode error")
	}
	if AuthModeOIDC.String() != "oidc" {
		t.Fatalf("AuthMode.String() = %q", AuthModeOIDC.String())
	}
}

func TestParseScopeKindAndSubjectKind(t *testing.T) {
	t.Parallel()

	for _, kind := range []ScopeKind{ScopeKindInstance, ScopeKindOrganization, ScopeKindProject} {
		parsed, err := ParseScopeKind(string(kind))
		if err != nil || parsed != kind {
			t.Fatalf("ParseScopeKind(%q) = %q, %v", kind, parsed, err)
		}
	}
	for _, kind := range []SubjectKind{
		SubjectKindLocalInstanceAdmin,
		SubjectKindUser,
		SubjectKindGroup,
		SubjectKindProjectConversation,
		SubjectKindTicketAgent,
		SubjectKindMachineAgent,
	} {
		parsed, err := ParseSubjectKind(string(kind))
		if err != nil || parsed != kind {
			t.Fatalf("ParseSubjectKind(%q) = %q, %v", kind, parsed, err)
		}
	}
	if _, err := ParseScopeKind("workspace"); err == nil {
		t.Fatal("ParseScopeKind() expected unsupported scope error")
	}
	if _, err := ParseSubjectKind("service_account"); err == nil {
		t.Fatal("ParseSubjectKind() expected unsupported subject kind error")
	}
}

func TestParseSubjectRef(t *testing.T) {
	t.Parallel()

	got, err := ParseSubjectRef(" local_instance_admin:default ")
	if err != nil {
		t.Fatalf("ParseSubjectRef() error = %v", err)
	}
	if got != DefaultLocalInstanceAdminSubject {
		t.Fatalf("ParseSubjectRef() = %#v, want %#v", got, DefaultLocalInstanceAdminSubject)
	}
	if got.String() != "local_instance_admin:default" {
		t.Fatalf("SubjectRef.String() = %q", got.String())
	}
	if !got.IsHumanLike() {
		t.Fatal("Default local subject should be treated as human-like")
	}
	if other := (SubjectRef{Kind: SubjectKindProjectConversation, Key: "123"}); other.IsHumanLike() {
		t.Fatal("project conversation subject should not be human-like")
	}

	if _, err := ParseSubjectRef("missing-separator"); err == nil {
		t.Fatal("ParseSubjectRef() expected format error")
	}
	if _, err := NewSubjectRef(SubjectKindUser, "   "); err == nil {
		t.Fatal("NewSubjectRef() expected empty key error")
	}
	if _, err := NewSubjectRef(SubjectKind("service_account"), "123"); err == nil {
		t.Fatal("NewSubjectRef() expected unsupported kind error")
	}
	if _, err := ParseSubjectRef("service_account:1"); err == nil {
		t.Fatal("ParseSubjectRef() expected unsupported kind error")
	}
}

func TestParseScopedRoles(t *testing.T) {
	t.Parallel()

	if role, err := ParseInstanceRole(" instance_admin "); err != nil || role != InstanceRoleAdmin {
		t.Fatalf("ParseInstanceRole() = %q, %v", role, err)
	}
	if InstanceRoleAdmin.String() != "instance_admin" || InstanceRoleAdmin.ScopeKind() != ScopeKindInstance {
		t.Fatal("instance role helpers should expose role key and scope")
	}
	if role, err := ParseOrganizationRole(" org_owner "); err != nil || role != OrganizationRoleOwner {
		t.Fatalf("ParseOrganizationRole() = %q, %v", role, err)
	}
	orgRoles := []OrganizationRole{OrganizationRoleOwner, OrganizationRoleAdmin, OrganizationRoleMember}
	for _, role := range orgRoles {
		parsed, err := ParseOrganizationRole(role.String())
		if err != nil || parsed != role {
			t.Fatalf("ParseOrganizationRole(%q) = %q, %v", role, parsed, err)
		}
		if role.ScopeKind() != ScopeKindOrganization {
			t.Fatalf("OrganizationRole.ScopeKind() = %q", role.ScopeKind())
		}
	}
	projectRoles := []ProjectRole{
		ProjectRoleAdmin,
		ProjectRoleOperator,
		ProjectRoleReviewer,
		ProjectRoleMember,
		ProjectRoleViewer,
	}
	for _, role := range projectRoles {
		parsed, err := ParseProjectRole(role.String())
		if err != nil || parsed != role {
			t.Fatalf("ParseProjectRole(%q) = %q, %v", role, parsed, err)
		}
		if role.ScopeKind() != ScopeKindProject {
			t.Fatalf("ProjectRole.ScopeKind() = %q", role.ScopeKind())
		}
	}

	if _, err := ParseInstanceRole("org_owner"); err == nil {
		t.Fatal("ParseInstanceRole() expected scope-specific error")
	}
	if _, err := ParseOrganizationRole("project_admin"); err == nil {
		t.Fatal("ParseOrganizationRole() expected scope-specific error")
	}
	if _, err := ParseProjectRole("instance_admin"); err == nil {
		t.Fatal("ParseProjectRole() expected scope-specific error")
	}
}

func TestMembershipStatusTransitions(t *testing.T) {
	t.Parallel()

	active, err := ParseMembershipStatus("active")
	if err != nil {
		t.Fatalf("ParseMembershipStatus() error = %v", err)
	}
	for _, status := range []MembershipStatus{
		MembershipStatusActive,
		MembershipStatusSuspended,
		MembershipStatusRevoked,
		MembershipStatusLeft,
	} {
		parsed, parseErr := ParseMembershipStatus(status.String())
		if parseErr != nil || parsed != status {
			t.Fatalf("ParseMembershipStatus(%q) = %q, %v", status, parsed, parseErr)
		}
	}
	if !active.CanTransitionTo(MembershipStatusSuspended) {
		t.Fatal("active membership should allow suspension")
	}
	if !active.CanTransitionTo(MembershipStatusLeft) {
		t.Fatal("active membership should allow leaving")
	}
	if active.String() != "active" {
		t.Fatalf("MembershipStatus.String() = %q", active.String())
	}
	if MembershipStatusSuspended.CanTransitionTo(MembershipStatusLeft) {
		t.Fatal("suspended membership should not allow leaving directly")
	}
	if MembershipStatusRevoked.CanTransitionTo(MembershipStatusActive) {
		t.Fatal("revoked membership should be terminal")
	}
	if MembershipStatusLeft.CanTransitionTo(MembershipStatusActive) {
		t.Fatal("left membership should be terminal")
	}
	if MembershipStatus("unknown").CanTransitionTo(MembershipStatusActive) {
		t.Fatal("unknown membership status should not transition")
	}
	if _, err := ParseMembershipStatus("pending"); err == nil {
		t.Fatal("ParseMembershipStatus() expected unsupported status error")
	}
}

func TestParseInvitationStatusAndSessionDevice(t *testing.T) {
	t.Parallel()

	for _, status := range []InvitationStatus{
		InvitationStatusPending,
		InvitationStatusAccepted,
		InvitationStatusExpired,
		InvitationStatusRevoked,
	} {
		parsed, err := ParseInvitationStatus(status.String())
		if err != nil || parsed != status {
			t.Fatalf("ParseInvitationStatus(%q) = %q, %v", status, parsed, err)
		}
	}
	if InvitationStatusAccepted.String() != "accepted" {
		t.Fatalf("InvitationStatus.String() = %q", InvitationStatusAccepted.String())
	}
	for _, device := range []SessionDevice{
		SessionDeviceBrowser,
		SessionDeviceCLI,
		SessionDeviceAutomation,
	} {
		parsed, err := ParseSessionDevice(device.String())
		if err != nil || parsed != device {
			t.Fatalf("ParseSessionDevice(%q) = %q, %v", device, parsed, err)
		}
	}
	if _, err := ParseSessionDevice("tablet"); err == nil {
		t.Fatal("ParseSessionDevice() expected unsupported device error")
	}
	if _, err := ParseInvitationStatus("cancelled"); err == nil {
		t.Fatal("ParseInvitationStatus() expected unsupported status error")
	}
}

func TestBuiltinAuthModeContracts(t *testing.T) {
	t.Parallel()

	disabled, err := ContractForMode(AuthModeDisabled)
	if err != nil {
		t.Fatalf("ContractForMode(disabled) error = %v", err)
	}
	if disabled.RequiresHumanLogin {
		t.Fatal("disabled mode must not require login")
	}
	if disabled.SupportsMultipleHumanUsers {
		t.Fatal("disabled mode must remain single-user")
	}
	if disabled.StableLocalSubject == nil || *disabled.StableLocalSubject != DefaultLocalInstanceAdminSubject {
		t.Fatalf("disabled StableLocalSubject = %#v", disabled.StableLocalSubject)
	}
	if disabled.StableLocalSubjectRole == nil || *disabled.StableLocalSubjectRole != InstanceRoleAdmin {
		t.Fatalf("disabled StableLocalSubjectRole = %#v", disabled.StableLocalSubjectRole)
	}
	if disabled.InteractiveSubjectKind != SubjectKindLocalInstanceAdmin {
		t.Fatalf("disabled InteractiveSubjectKind = %q", disabled.InteractiveSubjectKind)
	}

	oidc, err := ContractForMode(AuthModeOIDC)
	if err != nil {
		t.Fatalf("ContractForMode(oidc) error = %v", err)
	}
	if !oidc.RequiresOIDCConfig || !oidc.RequiresHumanLogin {
		t.Fatal("oidc mode must require OIDC config and login")
	}
	if !oidc.SupportsMultipleHumanUsers || !oidc.SupportsUserDirectory {
		t.Fatal("oidc mode must support multi-user directory semantics")
	}
	if oidc.StableLocalSubject != nil {
		t.Fatalf("oidc StableLocalSubject = %#v, want nil", oidc.StableLocalSubject)
	}
	if !oidc.SupportsBootstrapAdminBinding {
		t.Fatal("oidc mode should allow bootstrap admin binding")
	}

	if _, err := ContractForMode(AuthMode("saml")); err == nil {
		t.Fatal("ContractForMode() expected unsupported mode error")
	}
}
