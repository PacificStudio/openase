package humanauth

import "testing"

func TestOrganizationMembershipRoleParsingAndRoleKeys(t *testing.T) {
	t.Parallel()

	cases := []struct {
		raw      string
		wantRole OrganizationMembershipRole
		wantKey  RoleKey
	}{
		{raw: " owner ", wantRole: OrganizationMembershipRoleOwner, wantKey: RoleOrgOwner},
		{raw: "admin", wantRole: OrganizationMembershipRoleAdmin, wantKey: RoleOrgAdmin},
		{raw: "member", wantRole: OrganizationMembershipRoleMember, wantKey: RoleOrgMember},
	}

	for _, tc := range cases {
		parsed, err := ParseOrganizationMembershipRole(tc.raw)
		if err != nil {
			t.Fatalf("ParseOrganizationMembershipRole(%q) error = %v", tc.raw, err)
		}
		if parsed != tc.wantRole {
			t.Fatalf("ParseOrganizationMembershipRole(%q) = %q, want %q", tc.raw, parsed, tc.wantRole)
		}
		if parsed.String() != string(tc.wantRole) {
			t.Fatalf("OrganizationMembershipRole.String() = %q, want %q", parsed.String(), tc.wantRole)
		}
		if parsed.RoleKey() != tc.wantKey {
			t.Fatalf("OrganizationMembershipRole.RoleKey() = %q, want %q", parsed.RoleKey(), tc.wantKey)
		}
	}

	if _, err := ParseOrganizationMembershipRole("viewer"); err == nil {
		t.Fatal("ParseOrganizationMembershipRole() expected unsupported role error")
	}
}

func TestOrganizationMembershipStatusParsingAndTransitions(t *testing.T) {
	t.Parallel()

	cases := []struct {
		raw         string
		wantStatus  OrganizationMembershipStatus
		transitions map[OrganizationMembershipStatus]bool
	}{
		{
			raw:        " invited ",
			wantStatus: OrganizationMembershipStatusInvited,
			transitions: map[OrganizationMembershipStatus]bool{
				OrganizationMembershipStatusActive:    true,
				OrganizationMembershipStatusRemoved:   true,
				OrganizationMembershipStatusSuspended: false,
			},
		},
		{
			raw:        "active",
			wantStatus: OrganizationMembershipStatusActive,
			transitions: map[OrganizationMembershipStatus]bool{
				OrganizationMembershipStatusSuspended: true,
				OrganizationMembershipStatusRemoved:   true,
				OrganizationMembershipStatusInvited:   false,
			},
		},
		{
			raw:        "suspended",
			wantStatus: OrganizationMembershipStatusSuspended,
			transitions: map[OrganizationMembershipStatus]bool{
				OrganizationMembershipStatusActive:  true,
				OrganizationMembershipStatusRemoved: true,
				OrganizationMembershipStatusInvited: false,
			},
		},
		{
			raw:        "removed",
			wantStatus: OrganizationMembershipStatusRemoved,
			transitions: map[OrganizationMembershipStatus]bool{
				OrganizationMembershipStatusActive:  false,
				OrganizationMembershipStatusRemoved: false,
			},
		},
	}

	for _, tc := range cases {
		parsed, err := ParseOrganizationMembershipStatus(tc.raw)
		if err != nil {
			t.Fatalf("ParseOrganizationMembershipStatus(%q) error = %v", tc.raw, err)
		}
		if parsed != tc.wantStatus {
			t.Fatalf("ParseOrganizationMembershipStatus(%q) = %q, want %q", tc.raw, parsed, tc.wantStatus)
		}
		if parsed.String() != string(tc.wantStatus) {
			t.Fatalf("OrganizationMembershipStatus.String() = %q, want %q", parsed.String(), tc.wantStatus)
		}
		for next, want := range tc.transitions {
			if got := parsed.CanTransitionTo(next); got != want {
				t.Fatalf("%q.CanTransitionTo(%q) = %v, want %v", parsed, next, got, want)
			}
		}
	}

	if got := OrganizationMembershipStatus("ghost").CanTransitionTo(OrganizationMembershipStatusActive); got {
		t.Fatal("unknown membership status should not allow transitions")
	}

	if _, err := ParseOrganizationMembershipStatus("disabled"); err == nil {
		t.Fatal("ParseOrganizationMembershipStatus() expected unsupported status error")
	}
}

func TestOrganizationInvitationStatusParsing(t *testing.T) {
	t.Parallel()

	for _, status := range []OrganizationInvitationStatus{
		OrganizationInvitationStatusPending,
		OrganizationInvitationStatusAccepted,
		OrganizationInvitationStatusCanceled,
		OrganizationInvitationStatusExpired,
	} {
		parsed, err := ParseOrganizationInvitationStatus(" " + status.String() + " ")
		if err != nil {
			t.Fatalf("ParseOrganizationInvitationStatus(%q) error = %v", status, err)
		}
		if parsed != status {
			t.Fatalf("ParseOrganizationInvitationStatus(%q) = %q, want %q", status, parsed, status)
		}
		if parsed.String() != string(status) {
			t.Fatalf("OrganizationInvitationStatus.String() = %q, want %q", parsed.String(), status)
		}
	}

	if _, err := ParseOrganizationInvitationStatus("revoked"); err == nil {
		t.Fatal("ParseOrganizationInvitationStatus() expected unsupported status error")
	}
}
