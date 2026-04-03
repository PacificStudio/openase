package workflow

import "testing"

func TestParsePlatformAccess(t *testing.T) {
	content := " tickets.list \nprojects.update\ntickets.list"

	access, err := ParsePlatformAccess(content)
	if err != nil {
		t.Fatalf("ParsePlatformAccess returned error: %v", err)
	}

	if !access.Configured {
		t.Fatalf("expected platform access to be configured")
	}
	want := []string{"tickets.list", "projects.update"}
	if len(access.Allowed) != len(want) {
		t.Fatalf("Allowed=%v, want %v", access.Allowed, want)
	}
	for index := range want {
		if access.Allowed[index] != want[index] {
			t.Fatalf("Allowed=%v, want %v", access.Allowed, want)
		}
	}
}

func TestParsePlatformAccessAbsent(t *testing.T) {
	content := ""

	access, err := ParsePlatformAccess(content)
	if err != nil {
		t.Fatalf("ParsePlatformAccess returned error: %v", err)
	}
	if access.Configured {
		t.Fatalf("expected platform access to be absent, got %+v", access)
	}
	if len(access.Allowed) != 0 {
		t.Fatalf("expected no allowed scopes, got %v", access.Allowed)
	}
}
