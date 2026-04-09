package humanauth

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestParseUserStatus(t *testing.T) {
	t.Parallel()

	status, err := ParseUserStatus(" ACTIVE ")
	if err != nil {
		t.Fatalf("ParseUserStatus(active) error = %v", err)
	}
	if status != UserStatusActive {
		t.Fatalf("ParseUserStatus(active) = %q, want %q", status, UserStatusActive)
	}

	status, err = ParseUserStatus("disabled")
	if err != nil {
		t.Fatalf("ParseUserStatus(disabled) error = %v", err)
	}
	if status != UserStatusDisabled {
		t.Fatalf("ParseUserStatus(disabled) = %q, want %q", status, UserStatusDisabled)
	}

	if _, err := ParseUserStatus("suspended"); err == nil {
		t.Fatal("ParseUserStatus(invalid) expected error")
	}
}

func TestParseUserDirectoryStatusFilter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		want UserDirectoryStatusFilter
	}{
		{name: "blank defaults to all", raw: "  ", want: UserDirectoryStatusAll},
		{name: "all", raw: "all", want: UserDirectoryStatusAll},
		{name: "active", raw: "active", want: UserDirectoryStatusActive},
		{name: "disabled", raw: "disabled", want: UserDirectoryStatusDisabled},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseUserDirectoryStatusFilter(tc.raw)
			if err != nil {
				t.Fatalf("ParseUserDirectoryStatusFilter(%q) error = %v", tc.raw, err)
			}
			if got != tc.want {
				t.Fatalf("ParseUserDirectoryStatusFilter(%q) = %q, want %q", tc.raw, got, tc.want)
			}
		})
	}

	if _, err := ParseUserDirectoryStatusFilter("unknown"); err == nil {
		t.Fatal("ParseUserDirectoryStatusFilter(invalid) expected error")
	}
}

func TestParseUserStatusTransitionSource(t *testing.T) {
	t.Parallel()

	sources := []UserStatusTransitionSource{
		UserStatusTransitionSourceAdminManual,
		UserStatusTransitionSourceOIDCUpstream,
		UserStatusTransitionSourceWebhook,
		UserStatusTransitionSourceSCIM,
	}
	for _, source := range sources {
		source := source
		t.Run(string(source), func(t *testing.T) {
			t.Parallel()
			got, err := ParseUserStatusTransitionSource(" " + string(source) + " ")
			if err != nil {
				t.Fatalf("ParseUserStatusTransitionSource(%q) error = %v", source, err)
			}
			if got != source {
				t.Fatalf("ParseUserStatusTransitionSource(%q) = %q, want %q", source, got, source)
			}
		})
	}

	if _, err := ParseUserStatusTransitionSource("batch_job"); err == nil {
		t.Fatal("ParseUserStatusTransitionSource(invalid) expected error")
	}
}

func TestNewUserDirectoryFilter(t *testing.T) {
	t.Parallel()

	filter, err := NewUserDirectoryFilter("  alice@example.com  ", "", 0)
	if err != nil {
		t.Fatalf("NewUserDirectoryFilter(defaults) error = %v", err)
	}
	if filter.Query != "alice@example.com" {
		t.Fatalf("filter.Query = %q, want trimmed query", filter.Query)
	}
	if filter.Status != UserDirectoryStatusAll {
		t.Fatalf("filter.Status = %q, want %q", filter.Status, UserDirectoryStatusAll)
	}
	if filter.Limit != UserDirectoryDefaultPageLimit {
		t.Fatalf("filter.Limit = %d, want %d", filter.Limit, UserDirectoryDefaultPageLimit)
	}

	filter, err = NewUserDirectoryFilter("", "active", UserDirectoryMaximumPageLimit+25)
	if err != nil {
		t.Fatalf("NewUserDirectoryFilter(capped) error = %v", err)
	}
	if filter.Limit != UserDirectoryMaximumPageLimit {
		t.Fatalf("filter.Limit = %d, want %d", filter.Limit, UserDirectoryMaximumPageLimit)
	}
	if filter.Status != UserDirectoryStatusActive {
		t.Fatalf("filter.Status = %q, want %q", filter.Status, UserDirectoryStatusActive)
	}

	if _, err := NewUserDirectoryFilter("", "retired", 10); err == nil {
		t.Fatal("NewUserDirectoryFilter(invalid status) expected error")
	}
}

func TestNewUserStatusTransitionInput(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	input, err := NewUserStatusTransitionInput(
		userID,
		UserStatusDisabled,
		"  Left the org  ",
		"  user:admin  ",
		UserStatusTransitionSourceAdminManual,
		true,
	)
	if err != nil {
		t.Fatalf("NewUserStatusTransitionInput(valid) error = %v", err)
	}
	if input.UserID != userID {
		t.Fatalf("input.UserID = %s, want %s", input.UserID, userID)
	}
	if input.Reason != "Left the org" {
		t.Fatalf("input.Reason = %q, want trimmed reason", input.Reason)
	}
	if input.ActorID != "user:admin" {
		t.Fatalf("input.ActorID = %q, want trimmed actor", input.ActorID)
	}
	if !input.RevokeSessions {
		t.Fatal("input.RevokeSessions = false, want true")
	}

	cases := []struct {
		name   string
		userID uuid.UUID
		status UserStatus
		reason string
		actor  string
		source UserStatusTransitionSource
	}{
		{
			name:   "missing user id",
			userID: uuid.Nil,
			status: UserStatusActive,
			reason: "reason",
			actor:  "user:admin",
			source: UserStatusTransitionSourceAdminManual,
		},
		{
			name:   "invalid status",
			userID: userID,
			status: UserStatus("paused"),
			reason: "reason",
			actor:  "user:admin",
			source: UserStatusTransitionSourceAdminManual,
		},
		{
			name:   "blank reason",
			userID: userID,
			status: UserStatusActive,
			reason: " ",
			actor:  "user:admin",
			source: UserStatusTransitionSourceAdminManual,
		},
		{
			name:   "blank actor",
			userID: userID,
			status: UserStatusActive,
			reason: "reason",
			actor:  " ",
			source: UserStatusTransitionSourceAdminManual,
		},
		{
			name:   "invalid source",
			userID: userID,
			status: UserStatusActive,
			reason: "reason",
			actor:  "user:admin",
			source: UserStatusTransitionSource("batch_job"),
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if _, err := NewUserStatusTransitionInput(tc.userID, tc.status, tc.reason, tc.actor, tc.source, false); err == nil {
				t.Fatalf("NewUserStatusTransitionInput(%s) expected error", tc.name)
			}
		})
	}
}

func TestParseUserStatusAuditEvent(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Round(time.Second)
	event := AuthAuditEvent{
		ID:        uuid.New(),
		ActorID:   "user:admin",
		EventType: AuthAuditUserDisabled,
		Message:   "disabled",
		Metadata: map[string]any{
			"reason":                "Left organization",
			"source":                string(UserStatusTransitionSourceWebhook),
			"revoked_session_count": "4",
		},
		CreatedAt: now,
	}
	audit, err := ParseUserStatusAuditEvent(event)
	if err != nil {
		t.Fatalf("ParseUserStatusAuditEvent(disabled) error = %v", err)
	}
	if audit.Status != UserStatusDisabled {
		t.Fatalf("audit.Status = %q, want %q", audit.Status, UserStatusDisabled)
	}
	if audit.Reason != "Left organization" {
		t.Fatalf("audit.Reason = %q, want reason", audit.Reason)
	}
	if audit.Source != UserStatusTransitionSourceWebhook {
		t.Fatalf("audit.Source = %q, want %q", audit.Source, UserStatusTransitionSourceWebhook)
	}
	if audit.RevokedSessionCount != 4 {
		t.Fatalf("audit.RevokedSessionCount = %d, want 4", audit.RevokedSessionCount)
	}
	if !audit.ChangedAt.Equal(now) {
		t.Fatalf("audit.ChangedAt = %s, want %s", audit.ChangedAt, now)
	}

	enabledAudit, err := ParseUserStatusAuditEvent(AuthAuditEvent{
		ID:        uuid.New(),
		ActorID:   "user:admin",
		EventType: AuthAuditUserEnabled,
		Metadata: map[string]any{
			"source":                "not-a-source",
			"revoked_session_count": []string{"bad"},
		},
		CreatedAt: now,
	})
	if err != nil {
		t.Fatalf("ParseUserStatusAuditEvent(enabled) error = %v", err)
	}
	if enabledAudit.Status != UserStatusActive {
		t.Fatalf("enabledAudit.Status = %q, want %q", enabledAudit.Status, UserStatusActive)
	}
	if enabledAudit.Source != UserStatusTransitionSourceAdminManual {
		t.Fatalf("enabledAudit.Source = %q, want default %q", enabledAudit.Source, UserStatusTransitionSourceAdminManual)
	}
	if enabledAudit.RevokedSessionCount != 0 {
		t.Fatalf("enabledAudit.RevokedSessionCount = %d, want 0", enabledAudit.RevokedSessionCount)
	}

	if _, err := ParseUserStatusAuditEvent(AuthAuditEvent{EventType: AuthAuditLoginSucceeded}); err == nil {
		t.Fatal("ParseUserStatusAuditEvent(non-status event) expected error")
	}
}

func TestParseAuditInteger(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		raw     any
		want    int
		wantErr bool
	}{
		{name: "int", raw: 3, want: 3},
		{name: "int64", raw: int64(5), want: 5},
		{name: "float64", raw: float64(7), want: 7},
		{name: "string", raw: "9", want: 9},
		{name: "invalid string", raw: "oops", wantErr: true},
		{name: "unsupported type", raw: true, wantErr: true},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := parseAuditInteger(tc.raw)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("parseAuditInteger(%T) expected error", tc.raw)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseAuditInteger(%T) error = %v", tc.raw, err)
			}
			if got != tc.want {
				t.Fatalf("parseAuditInteger(%T) = %d, want %d", tc.raw, got, tc.want)
			}
		})
	}
}
