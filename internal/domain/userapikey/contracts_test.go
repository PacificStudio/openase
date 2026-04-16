package userapikey

import (
	"slices"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestParseCreate(t *testing.T) {
	t.Run("normalizes valid input", func(t *testing.T) {
		userID := uuid.New()
		expiresAt := "2026-05-01T10:30:00+02:00"

		got, err := ParseCreate(ParseCreateInput{
			ProjectID: " " + uuid.NewString() + " ",
			UserID:    userID,
			Name:      "  Deploy Key  ",
			Scopes:    []string{" tickets.list ", "project_updates.read", "tickets.list"},
			ExpiresAt: &expiresAt,
		})
		if err != nil {
			t.Fatalf("ParseCreate() error = %v", err)
		}
		if got.UserID != userID {
			t.Fatalf("ParseCreate() user_id = %s, want %s", got.UserID, userID)
		}
		if got.Name != "Deploy Key" {
			t.Fatalf("ParseCreate() name = %q, want %q", got.Name, "Deploy Key")
		}
		wantScopes := []string{"tickets.list", "project_updates.read"}
		if !slices.Equal(got.Scopes, wantScopes) {
			t.Fatalf("ParseCreate() scopes = %v, want %v", got.Scopes, wantScopes)
		}
		if got.ExpiresAt == nil {
			t.Fatal("ParseCreate() expires_at = nil, want parsed time")
		}
		if got.ExpiresAt.Format(time.RFC3339) != "2026-05-01T08:30:00Z" {
			t.Fatalf("ParseCreate() expires_at = %s, want %s", got.ExpiresAt.Format(time.RFC3339), "2026-05-01T08:30:00Z")
		}
	})

	t.Run("accepts omitted expiry", func(t *testing.T) {
		got, err := ParseCreate(ParseCreateInput{
			ProjectID: uuid.NewString(),
			UserID:    uuid.New(),
			Name:      "Key",
			Scopes:    []string{"tickets.list"},
		})
		if err != nil {
			t.Fatalf("ParseCreate() error = %v", err)
		}
		if got.ExpiresAt != nil {
			t.Fatalf("ParseCreate() expires_at = %v, want nil", got.ExpiresAt)
		}
	})

	tests := []struct {
		name string
		raw  ParseCreateInput
		want string
	}{
		{
			name: "invalid project id",
			raw: ParseCreateInput{
				ProjectID: "bad",
				UserID:    uuid.New(),
				Name:      "Key",
				Scopes:    []string{"tickets.list"},
			},
			want: "project_id must be a valid UUID",
		},
		{
			name: "missing user id",
			raw: ParseCreateInput{
				ProjectID: uuid.NewString(),
				Name:      "Key",
				Scopes:    []string{"tickets.list"},
			},
			want: "user_id must be a valid UUID",
		},
		{
			name: "blank name",
			raw: ParseCreateInput{
				ProjectID: uuid.NewString(),
				UserID:    uuid.New(),
				Name:      "   ",
				Scopes:    []string{"tickets.list"},
			},
			want: "name must not be empty",
		},
		{
			name: "blank scope entry",
			raw: ParseCreateInput{
				ProjectID: uuid.NewString(),
				UserID:    uuid.New(),
				Name:      "Key",
				Scopes:    []string{"tickets.list", " "},
			},
			want: "scopes must not contain empty values",
		},
		{
			name: "no scopes",
			raw: ParseCreateInput{
				ProjectID: uuid.NewString(),
				UserID:    uuid.New(),
				Name:      "Key",
			},
			want: "at least one scope must be selected",
		},
		{
			name: "invalid expiry",
			raw: ParseCreateInput{
				ProjectID: uuid.NewString(),
				UserID:    uuid.New(),
				Name:      "Key",
				Scopes:    []string{"tickets.list"},
				ExpiresAt: ptr("tomorrow"),
			},
			want: "expires_at must be RFC3339",
		},
		{
			name: "blank expiry string ignored",
			raw: ParseCreateInput{
				ProjectID: uuid.NewString(),
				UserID:    uuid.New(),
				Name:      "Key",
				Scopes:    []string{"tickets.list"},
				ExpiresAt: ptr(" "),
			},
			want: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseCreate(tc.raw)
			if tc.want == "" {
				if err != nil {
					t.Fatalf("ParseCreate() error = %v", err)
				}
				if got.ExpiresAt != nil {
					t.Fatalf("ParseCreate() expires_at = %v, want nil", got.ExpiresAt)
				}
				return
			}
			if err == nil || err.Error() != tc.want {
				t.Fatalf("ParseCreate() error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestSupportedScopeGroups(t *testing.T) {
	if got := SupportedScopeGroups(nil); got != nil {
		t.Fatalf("SupportedScopeGroups(nil) = %v, want nil", got)
	}

	got := SupportedScopeGroups([]string{
		"project_updates.write",
		"tickets.update",
		"tickets.list",
		"activity",
		"project_updates.read",
	})
	want := []struct {
		category string
		scopes   []string
	}{
		{category: "activity", scopes: []string{"activity"}},
		{category: "project_updates", scopes: []string{"project_updates.read", "project_updates.write"}},
		{category: "tickets", scopes: []string{"tickets.list", "tickets.update"}},
	}
	if len(got) != len(want) {
		t.Fatalf("SupportedScopeGroups() len = %d, want %d", len(got), len(want))
	}
	for i, item := range got {
		if item.Category != want[i].category {
			t.Fatalf("SupportedScopeGroups()[%d].Category = %q, want %q", i, item.Category, want[i].category)
		}
		if !slices.Equal(item.Scopes, want[i].scopes) {
			t.Fatalf("SupportedScopeGroups()[%d].Scopes = %v, want %v", i, item.Scopes, want[i].scopes)
		}
	}
}

func ptr(value string) *string {
	return &value
}
