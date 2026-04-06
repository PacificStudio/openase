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
}

func TestParseScopeKindAndSubjectKind(t *testing.T) {
	t.Parallel()

	if kind, err := ParseScopeKind(" project "); err != nil || kind != ScopeKindProject {
		t.Fatalf("ParseScopeKind() = %q, %v", kind, err)
	}
	if kind, err := ParseSubjectKind(" ticket_agent "); err != nil || kind != SubjectKindTicketAgent {
		t.Fatalf("ParseSubjectKind() = %q, %v", kind, err)
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

	if _, err := ParseSubjectRef("missing-separator"); err == nil {
		t.Fatal("ParseSubjectRef() expected format error")
	}
}

func TestParseScopedRoles(t *testing.T) {
	t.Parallel()

	if role, err := ParseInstanceRole(" instance_admin "); err != nil || role != InstanceRoleAdmin {
		t.Fatalf("ParseInstanceRole() = %q, %v", role, err)
	}
	if role, err := ParseOrganizationRole(" org_owner "); err != nil || role != OrganizationRoleOwner {
		t.Fatalf("ParseOrganizationRole() = %q, %v", role, err)
	}
	if role, err := ParseProjectRole(" project_reviewer "); err != nil || role != ProjectRoleReviewer {
		t.Fatalf("ParseProjectRole() = %q, %v", role, err)
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
	if !active.CanTransitionTo(MembershipStatusSuspended) {
		t.Fatal("active membership should allow suspension")
	}
	if !active.CanTransitionTo(MembershipStatusLeft) {
		t.Fatal("active membership should allow leaving")
	}
	if MembershipStatusRevoked.CanTransitionTo(MembershipStatusActive) {
		t.Fatal("revoked membership should be terminal")
	}
	if _, err := ParseMembershipStatus("pending"); err == nil {
		t.Fatal("ParseMembershipStatus() expected unsupported status error")
	}
}

func TestParseInvitationStatusAndSessionDevice(t *testing.T) {
	t.Parallel()

	if status, err := ParseInvitationStatus(" pending "); err != nil || status != InvitationStatusPending {
		t.Fatalf("ParseInvitationStatus() = %q, %v", status, err)
	}
	if device, err := ParseSessionDevice("CLI"); err != nil || device != SessionDeviceCLI {
		t.Fatalf("ParseSessionDevice() = %q, %v", device, err)
	}
	if _, err := ParseSessionDevice("tablet"); err == nil {
		t.Fatal("ParseSessionDevice() expected unsupported device error")
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
