package ticket

import "testing"

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
