package ticket

import (
	"testing"

	"github.com/google/uuid"
)

func TestParsePriority(t *testing.T) {
	got, err := ParsePriority(" High ")
	if err != nil {
		t.Fatalf("ParsePriority() error = %v", err)
	}
	if got != PriorityHigh {
		t.Fatalf("ParsePriority() = %q", got)
	}
	if got.String() != "high" {
		t.Fatalf("Priority.String() = %q", got.String())
	}

	if _, err := ParsePriority("critical"); err == nil {
		t.Fatal("expected invalid priority to fail")
	}
}

func TestParseType(t *testing.T) {
	got, err := ParseType(" Refactor ")
	if err != nil {
		t.Fatalf("ParseType() error = %v", err)
	}
	if got != TypeRefactor {
		t.Fatalf("ParseType() = %q", got)
	}
	if got.String() != "refactor" {
		t.Fatalf("Type.String() = %q", got.String())
	}

	if _, err := ParseType("incident"); err == nil {
		t.Fatal("expected invalid type to fail")
	}
}

func TestParseDependencyType(t *testing.T) {
	got, err := ParseDependencyType(" sub-issue ")
	if err != nil {
		t.Fatalf("ParseDependencyType() error = %v", err)
	}
	if got != DependencyTypeSubIssue {
		t.Fatalf("ParseDependencyType() = %q", got)
	}
	if got.String() != "sub-issue" {
		t.Fatalf("DependencyType.String() = %q", got.String())
	}

	if _, err := ParseDependencyType("blocked_by"); err == nil {
		t.Fatal("expected invalid dependency type to fail")
	}
}

func TestParseExternalLinkType(t *testing.T) {
	got, err := ParseExternalLinkType(" GitHub_PR ")
	if err != nil {
		t.Fatalf("ParseExternalLinkType() error = %v", err)
	}
	if got != ExternalLinkTypeGithubPR {
		t.Fatalf("ParseExternalLinkType() = %q", got)
	}
	if got.String() != "github_pr" {
		t.Fatalf("ExternalLinkType.String() = %q", got.String())
	}

	if _, err := ParseExternalLinkType("trello"); err == nil {
		t.Fatal("expected invalid link type to fail")
	}
}

func TestParseExternalLinkRelation(t *testing.T) {
	got, err := ParseExternalLinkRelation(" caused_by ")
	if err != nil {
		t.Fatalf("ParseExternalLinkRelation() error = %v", err)
	}
	if got != ExternalLinkRelationCausedBy {
		t.Fatalf("ParseExternalLinkRelation() = %q", got)
	}
	if got.String() != "caused_by" {
		t.Fatalf("ExternalLinkRelation.String() = %q", got.String())
	}

	if _, err := ParseExternalLinkRelation("duplicates"); err == nil {
		t.Fatal("expected invalid relation to fail")
	}
}

func TestParseArchivedListInput(t *testing.T) {
	projectID := uuid.New()

	got, err := ParseArchivedListInput(projectID, ArchivedListRawInput{})
	if err != nil {
		t.Fatalf("ParseArchivedListInput() defaults error = %v", err)
	}
	if got.ProjectID != projectID {
		t.Fatalf("project id = %s", got.ProjectID)
	}
	if got.Page != DefaultArchivedTicketPage {
		t.Fatalf("default page = %d", got.Page)
	}
	if got.PerPage != DefaultArchivedTicketPerPage {
		t.Fatalf("default per_page = %d", got.PerPage)
	}

	got, err = ParseArchivedListInput(projectID, ArchivedListRawInput{
		Page:    " 3 ",
		PerPage: " 40 ",
	})
	if err != nil {
		t.Fatalf("ParseArchivedListInput() parsed error = %v", err)
	}
	if got.Page != 3 || got.PerPage != 40 {
		t.Fatalf("parsed input = %+v", got)
	}
}

func TestParseArchivedListInputRejectsInvalidValues(t *testing.T) {
	projectID := uuid.New()

	tests := []struct {
		name string
		raw  ArchivedListRawInput
		want string
	}{
		{
			name: "page not integer",
			raw: ArchivedListRawInput{
				Page: "abc",
			},
			want: "page must be a valid integer",
		},
		{
			name: "page non positive",
			raw: ArchivedListRawInput{
				Page: "0",
			},
			want: "page must be greater than zero",
		},
		{
			name: "per page not integer",
			raw: ArchivedListRawInput{
				PerPage: "abc",
			},
			want: "per_page must be a valid integer",
		},
		{
			name: "per page non positive",
			raw: ArchivedListRawInput{
				PerPage: "-1",
			},
			want: "per_page must be greater than zero",
		},
		{
			name: "per page above max",
			raw: ArchivedListRawInput{
				PerPage: "101",
			},
			want: "per_page must be less than or equal to 100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseArchivedListInput(projectID, tt.raw)
			if err == nil {
				t.Fatal("expected ParseArchivedListInput() to fail")
			}
			if err.Error() != tt.want {
				t.Fatalf("error = %q, want %q", err.Error(), tt.want)
			}
		})
	}
}
