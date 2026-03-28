package hradvisor

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestParseActivateRecommendation(t *testing.T) {
	projectID := uuid.New()
	createBootstrapTicket := true

	input, err := ParseActivateRecommendation(projectID, ActivateRecommendationRequest{
		RoleSlug:              " qa-engineer ",
		CreateBootstrapTicket: &createBootstrapTicket,
	})
	if err != nil {
		t.Fatalf("ParseActivateRecommendation() error = %v", err)
	}
	if input.ProjectID != projectID || input.RoleSlug != "qa-engineer" || !input.CreateBootstrapTicket {
		t.Fatalf("ParseActivateRecommendation() = %+v", input)
	}
}

func TestParseActivateRecommendationRejectsInvalidRoleSlug(t *testing.T) {
	_, err := ParseActivateRecommendation(uuid.New(), ActivateRecommendationRequest{RoleSlug: "QA Engineer"})
	if err == nil || !strings.Contains(err.Error(), "role_slug must be a lowercase slug") {
		t.Fatalf("ParseActivateRecommendation(invalid) error = %v", err)
	}
}

func TestParseActivateRecommendationRejectsEmptyRoleSlug(t *testing.T) {
	_, err := ParseActivateRecommendation(uuid.New(), ActivateRecommendationRequest{RoleSlug: "   "})
	if err == nil || !strings.Contains(err.Error(), "role_slug must not be empty") {
		t.Fatalf("ParseActivateRecommendation(empty) error = %v", err)
	}
}

func TestParseActivateRecommendationDefaultsBootstrapTicketToFalse(t *testing.T) {
	input, err := ParseActivateRecommendation(uuid.New(), ActivateRecommendationRequest{RoleSlug: "qa-engineer"})
	if err != nil {
		t.Fatalf("ParseActivateRecommendation() error = %v", err)
	}
	if input.CreateBootstrapTicket {
		t.Fatalf("expected bootstrap ticket default false, got %+v", input)
	}
}

func TestParseActivationTemplate(t *testing.T) {
	template, err := ParseActivationTemplate(
		"qa-engineer",
		".openase/harnesses/roles/qa-engineer.md",
		`---
workflow:
  name: "QA Engineer"
  type: "test"
  role: "qa-engineer"
status:
  pickup: "Todo"
  finish: "Done"
---

# QA Engineer
`,
		"Write automated regression coverage.",
	)
	if err != nil {
		t.Fatalf("ParseActivationTemplate() error = %v", err)
	}
	if template.WorkflowName != "QA Engineer" || template.WorkflowType != "test" || template.PickupStatusName != "Todo" || template.FinishStatusName != "Done" {
		t.Fatalf("ParseActivationTemplate() = %+v", template)
	}
}

func TestParseActivationTemplateFallsBackToRequestedRoleSlug(t *testing.T) {
	template, err := ParseActivationTemplate(
		"qa-engineer",
		".openase/harnesses/roles/qa-engineer.md",
		`---
workflow:
  name: "QA Engineer"
  type: "test"
status:
  pickup: "Todo"
  finish: "Done"
---
`,
		"Write automated regression coverage.",
	)
	if err != nil {
		t.Fatalf("ParseActivationTemplate() error = %v", err)
	}
	if template.RoleSlug != "qa-engineer" {
		t.Fatalf("expected requested role slug fallback, got %+v", template)
	}
}

func TestParseActivationTemplateRejectsRoleMismatch(t *testing.T) {
	_, err := ParseActivationTemplate(
		"qa-engineer",
		".openase/harnesses/roles/qa-engineer.md",
		`---
workflow:
  name: "QA Engineer"
  type: "test"
  role: "technical-writer"
status:
  pickup: "Todo"
  finish: "Done"
---
`,
		"Write automated regression coverage.",
	)
	if err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("ParseActivationTemplate(role mismatch) error = %v", err)
	}
}

func TestParseActivationTemplateRejectsInvalidFrontmatter(t *testing.T) {
	_, err := ParseActivationTemplate(
		"qa-engineer",
		".openase/harnesses/roles/qa-engineer.md",
		`---
workflow: [broken
---
`,
		"Write automated regression coverage.",
	)
	if err == nil || !strings.Contains(err.Error(), "parse harness frontmatter") {
		t.Fatalf("ParseActivationTemplate(invalid yaml) error = %v", err)
	}
}

func TestParseActivationTemplateRejectsMissingFields(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    string
	}{
		{
			name: "missing workflow name",
			content: `---
workflow:
  type: "test"
  role: "qa-engineer"
status:
  pickup: "Todo"
  finish: "Done"
---
`,
			want: "workflow.name must not be empty",
		},
		{
			name: "missing workflow type",
			content: `---
workflow:
  name: "QA Engineer"
  role: "qa-engineer"
status:
  pickup: "Todo"
  finish: "Done"
---
`,
			want: "workflow.type must not be empty",
		},
		{
			name: "missing pickup status",
			content: `---
workflow:
  name: "QA Engineer"
  type: "test"
  role: "qa-engineer"
status:
  finish: "Done"
---
`,
			want: "status.pickup must not be empty",
		},
		{
			name: "missing finish status",
			content: `---
workflow:
  name: "QA Engineer"
  type: "test"
  role: "qa-engineer"
status:
  pickup: "Todo"
---
`,
			want: "status.finish must not be empty",
		},
	}

	for _, testCase := range cases {
		_, err := ParseActivationTemplate(
			"qa-engineer",
			".openase/harnesses/roles/qa-engineer.md",
			testCase.content,
			"Write automated regression coverage.",
		)
		if err == nil || !strings.Contains(err.Error(), testCase.want) {
			t.Fatalf("%s error = %v, want substring %q", testCase.name, err, testCase.want)
		}
	}
}

func TestParseActivationTemplateRejectsMissingHarnessPathAndContent(t *testing.T) {
	if _, err := ParseActivationTemplate(
		"qa-engineer",
		" ",
		`---
workflow:
  name: "QA Engineer"
  type: "test"
  role: "qa-engineer"
status:
  pickup: "Todo"
  finish: "Done"
---
`,
		"Write automated regression coverage.",
	); err == nil || !strings.Contains(err.Error(), "harness_path must not be empty") {
		t.Fatalf("ParseActivationTemplate(blank path) error = %v", err)
	}

	if _, err := ParseActivationTemplate("qa-engineer", ".openase/harnesses/roles/qa-engineer.md", "", "Write automated regression coverage."); err == nil || !strings.Contains(err.Error(), "harness frontmatter must start with ---") {
		t.Fatalf("ParseActivationTemplate(blank content) error = %v", err)
	}
}

func TestExtractActivationFrontmatterRejectsInvalidLayouts(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    string
	}{
		{name: "missing opening delimiter", content: "workflow:\n  name: bad\n", want: "must start with ---"},
		{name: "empty frontmatter", content: "---\n---\n", want: "must not be empty"},
		{name: "missing closing delimiter", content: "---\nworkflow:\n  name: bad\n", want: "closing delimiter not found"},
	}

	for _, testCase := range cases {
		_, err := extractActivationFrontmatter(testCase.content)
		if err == nil || !strings.Contains(err.Error(), testCase.want) {
			t.Fatalf("%s error = %v, want substring %q", testCase.name, err, testCase.want)
		}
	}
}
