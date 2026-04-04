package githubrepo

import (
	"testing"

	"github.com/google/uuid"
)

func TestParseListRepositories(t *testing.T) {
	projectID := uuid.New()

	parsed, err := ParseListRepositories(projectID, "  backend  ", "2")
	if err != nil {
		t.Fatalf("ParseListRepositories() error = %v", err)
	}
	if parsed.ProjectID != projectID || parsed.Query != "backend" || parsed.Page != 2 {
		t.Fatalf("ParseListRepositories() = %+v", parsed)
	}

	defaulted, err := ParseListRepositories(projectID, "", "")
	if err != nil {
		t.Fatalf("ParseListRepositories(default) error = %v", err)
	}
	if defaulted.Page != 1 {
		t.Fatalf("ParseListRepositories(default) page = %d, want 1", defaulted.Page)
	}

	if _, err := ParseListRepositories(projectID, "", "zero"); err == nil {
		t.Fatal("ParseListRepositories() expected cursor validation error")
	}
}

func TestParseCreateRepository(t *testing.T) {
	projectID := uuid.New()

	parsed, err := ParseCreateRepository(projectID, CreateRepositoryRequest{
		Owner:       "  acme  ",
		Name:        "  backend  ",
		Description: "  control plane  ",
		Visibility:  "private",
	})
	if err != nil {
		t.Fatalf("ParseCreateRepository() error = %v", err)
	}
	if parsed.ProjectID != projectID || parsed.Owner != "acme" || parsed.Name != "backend" {
		t.Fatalf("ParseCreateRepository() = %+v", parsed)
	}
	if parsed.Visibility != VisibilityPrivate || !parsed.AutoInit {
		t.Fatalf("ParseCreateRepository() visibility/auto_init = %+v", parsed)
	}

	autoInit := false
	withOverride, err := ParseCreateRepository(projectID, CreateRepositoryRequest{
		Owner:      "acme",
		Name:       "frontend",
		Visibility: "public",
		AutoInit:   &autoInit,
	})
	if err != nil {
		t.Fatalf("ParseCreateRepository(override) error = %v", err)
	}
	if withOverride.AutoInit {
		t.Fatalf("ParseCreateRepository(override) auto_init = true, want false")
	}

	if _, err := ParseCreateRepository(projectID, CreateRepositoryRequest{Name: "backend", Visibility: "private"}); err == nil {
		t.Fatal("ParseCreateRepository() expected owner validation error")
	}
	if _, err := ParseCreateRepository(projectID, CreateRepositoryRequest{Owner: "acme", Visibility: "private"}); err == nil {
		t.Fatal("ParseCreateRepository() expected name validation error")
	}
	if _, err := ParseCreateRepository(projectID, CreateRepositoryRequest{Owner: "acme", Name: "backend", Visibility: "internal"}); err == nil {
		t.Fatal("ParseCreateRepository() expected visibility validation error")
	}
}
