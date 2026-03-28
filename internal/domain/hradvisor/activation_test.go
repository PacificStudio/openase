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
